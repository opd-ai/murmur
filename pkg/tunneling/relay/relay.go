// Package relay implements the tunnel exit relay (client → operator forwarding).
// This is a Phase 6.3 minimal prototype: single-hop, HTTP-only forwarding.
package relay

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
	"github.com/opd-ai/murmur/pkg/tunneling/protocol"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

const unauthorizedResponse = "HTTP/1.1 401 Unauthorized\r\nContent-Type: text/plain\r\n\r\nUnauthorized\n"

type operatorSession struct {
	conn net.Conn
	mu   sync.Mutex
	seq  uint32
}

// Relay manages tunnel connections from operators and forwards client traffic.
type Relay struct {
	listenAddr string
	tunnels    map[tunneling.TunnelID]*operatorSession
	mu         sync.RWMutex
	listener   net.Listener
	stopCh     chan struct{}
}

// NewRelay creates a new exit relay.
func NewRelay(listenAddr string) *Relay {
	return &Relay{
		listenAddr: listenAddr,
		tunnels:    make(map[tunneling.TunnelID]*operatorSession),
		stopCh:     make(chan struct{}),
	}
}

// Start begins listening for tunnel registrations and client connections.
func (r *Relay) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", r.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", r.listenAddr, err)
	}
	r.listener = listener

	go r.acceptLoop(ctx)
	return nil
}

// acceptLoop accepts incoming connections (operators and clients).
func (r *Relay) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		default:
		}

		// F-RES-2 fix: Set accept deadline to prevent blocking indefinitely after context cancellation.
		// Type-assert to *net.TCPListener to access SetDeadline.
		if tcpListener, ok := r.listener.(*net.TCPListener); ok {
			_ = tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}

		conn, err := r.listener.Accept()
		if err != nil {
			// Check if this is a timeout error (expected during shutdown)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout is expected, continue to check ctx.Done()
				continue
			}
			// Other errors (e.g., listener closed) should also loop back to check ctx.Done()
			continue
		}

		go r.handleConnection(ctx, conn)
	}
}

// handleConnection reads the first byte to determine if this is framed operator
// traffic or an HTTP client request.
func (r *Relay) handleConnection(ctx context.Context, conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	firstByte := make([]byte, 1)
	if _, err := io.ReadFull(conn, firstByte); err != nil {
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	if firstByte[0] == protocol.FrameMagic {
		r.handleFramedOperator(ctx, conn)
		return
	}

	defer conn.Close()
	reader := bufio.NewReader(io.MultiReader(strings.NewReader(string(firstByte[0])), conn))
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	firstLine = strings.TrimSpace(firstLine)

	if strings.HasPrefix(firstLine, "UNREGISTER ") {
		_, _ = conn.Write([]byte(unauthorizedResponse))
		return
	}

	r.handleClientRequest(ctx, conn, firstLine, reader)
}

// handleFramedOperator registers and maintains an operator tunnel session.
func (r *Relay) handleFramedOperator(ctx context.Context, conn net.Conn) {
	frameType, payload, err := protocol.ReadFrameWithFirstByte(conn, protocol.FrameMagic)
	if err != nil {
		conn.Close()
		return
	}
	if frameType != protocol.FrameTypeRegister {
		conn.Close()
		return
	}

	cell, err := protocol.DecodeAndVerifyRegisterCell(payload, time.Now())
	if err != nil {
		conn.Close()
		return
	}

	tunnelID := tunneling.TunnelID(cell.TunnelId)
	if err := tunnelID.Validate(); err != nil {
		conn.Close()
		return
	}

	session := &operatorSession{conn: conn}
	r.mu.Lock()
	r.tunnels[tunnelID] = session
	r.mu.Unlock()

	_, _ = conn.Write([]byte("OK\n"))

	defer func() {
		r.mu.Lock()
		if existing, ok := r.tunnels[tunnelID]; ok && existing == session {
			delete(r.tunnels, tunnelID)
		}
		r.mu.Unlock()
		conn.Close()
	}()

	select {
	case <-ctx.Done():
	case <-r.stopCh:
	}
}

func (r *Relay) handleTeardownCell(payload []byte, expectedTunnelID tunneling.TunnelID) bool {
	cell := &pb.TunnelTeardownCell{}
	if err := proto.Unmarshal(payload, cell); err != nil {
		return false
	}
	// Teardown is accepted only on the already authenticated operator TCP
	// session that originally registered this tunnel.
	if tunneling.TunnelID(cell.TunnelId) != expectedTunnelID {
		return false
	}
	return true
}

// handleClientRequest forwards a client's HTTP request to the tunnel operator.
func (r *Relay) handleClientRequest(ctx context.Context, clientConn net.Conn, firstLine string, reader *bufio.Reader) {
	tunnelID, err := r.parseTunnelID(firstLine, clientConn)
	if err != nil {
		return
	}

	session, ok := r.lookupTunnel(tunnelID)
	if !ok {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\n\r\nTunnel not found\n"))
		return
	}

	fullRequest := r.reconstructHTTPRequest(firstLine, reader)
	session.mu.Lock()
	defer session.mu.Unlock()

	if !r.forwardRequestToOperator(session, tunnelID, fullRequest, clientConn) {
		return
	}

	r.forwardResponseToClient(session, tunnelID, clientConn)
}

// parseTunnelID extracts and validates the tunnel ID from the HTTP request path.
func (r *Relay) parseTunnelID(firstLine string, clientConn net.Conn) (tunneling.TunnelID, error) {
	parts := strings.Fields(firstLine)
	if len(parts) < 2 {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return "", fmt.Errorf("malformed request")
	}

	path := parts[1]
	const prefix = "/tunnel/"
	if !strings.HasPrefix(path, prefix) {
		clientConn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return "", fmt.Errorf("invalid path")
	}

	pathAfterPrefix := strings.TrimPrefix(path, prefix)
	tunnelIDStr := pathAfterPrefix
	if idx := strings.Index(pathAfterPrefix, "/"); idx != -1 {
		tunnelIDStr = pathAfterPrefix[:idx]
	}

	tunnelID := tunneling.TunnelID(tunnelIDStr)
	if err := tunnelID.Validate(); err != nil {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return "", err
	}

	return tunnelID, nil
}

// lookupTunnel retrieves the operator connection for a tunnel ID.
func (r *Relay) lookupTunnel(tunnelID tunneling.TunnelID) (*operatorSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.tunnels[tunnelID]
	return session, ok
}

// reconstructHTTPRequest rebuilds the full HTTP request from first line and headers.
func (r *Relay) reconstructHTTPRequest(firstLine string, reader *bufio.Reader) string {
	fullRequest := firstLine + "\r\n"
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			fullRequest += "\r\n"
			break
		}
		fullRequest += line
	}
	return fullRequest
}

// forwardRequestToOperator sends the HTTP request to the operator connection.
func (r *Relay) forwardRequestToOperator(session *operatorSession, tunnelID tunneling.TunnelID, fullRequest string, clientConn net.Conn) bool {
	cell := &pb.TunnelDataCell{
		TunnelId: []byte(tunnelID),
		Payload:  []byte(fullRequest),
		Sequence: session.seq,
	}
	payload, err := proto.Marshal(cell)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return false
	}

	if err := protocol.WriteFrame(session.conn, protocol.FrameTypeData, payload); err != nil {
		r.removeTunnel(tunnelID, session)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return false
	}
	session.seq++
	return true
}

// forwardResponseToClient reads the operator's response and forwards it to the client.
func (r *Relay) forwardResponseToClient(session *operatorSession, tunnelID tunneling.TunnelID, clientConn net.Conn) {
	payload, ok := r.readOperatorDataFrame(session, tunnelID)
	if !ok {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	responsePayload, ok := r.extractResponsePayload(payload, tunnelID)
	if !ok {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	_, _ = clientConn.Write(responsePayload)
}

func (r *Relay) readOperatorDataFrame(session *operatorSession, tunnelID tunneling.TunnelID) ([]byte, bool) {
	session.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	frameType, payload, err := protocol.ReadFrame(session.conn)
	if err != nil {
		r.removeTunnel(tunnelID, session)
		return nil, false
	}
	if frameType == protocol.FrameTypeTeardown {
		if !r.handleTeardownCell(payload, tunnelID) {
			return nil, false
		}
		r.removeTunnel(tunnelID, session)
		return nil, false
	}
	if frameType != protocol.FrameTypeData {
		return nil, false
	}
	return payload, true
}

func (r *Relay) extractResponsePayload(payload []byte, tunnelID tunneling.TunnelID) ([]byte, bool) {
	respCell := &pb.TunnelDataCell{}
	if err := proto.Unmarshal(payload, respCell); err != nil {
		return nil, false
	}
	if string(respCell.TunnelId) != string(tunnelID) {
		return nil, false
	}
	return respCell.Payload, true
}

func (r *Relay) removeTunnel(tunnelID tunneling.TunnelID, expectedSession *operatorSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.tunnels[tunnelID]
	// Session pointer match ensures only the currently registered operator
	// session for this tunnel ID can remove the mapping.
	if !ok || session != expectedSession {
		return
	}
	_ = session.conn.Close()
	delete(r.tunnels, tunnelID)
}

// Stop gracefully shuts down the relay.
func (r *Relay) Stop(ctx context.Context) error {
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}

	if r.listener != nil {
		r.listener.Close()
	}

	r.mu.Lock()
	for _, session := range r.tunnels {
		session.conn.Close()
	}
	r.tunnels = make(map[tunneling.TunnelID]*operatorSession)
	r.mu.Unlock()

	return nil
}

// ListenAddr returns the relay's listening address.
func (r *Relay) ListenAddr() string {
	if r.listener != nil {
		return r.listener.Addr().String()
	}
	return r.listenAddr
}
