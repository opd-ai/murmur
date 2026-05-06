// Package relay implements the tunnel exit relay (client → operator forwarding).
// This is a Phase 6.3 minimal prototype: single-hop, HTTP-only forwarding.
package relay

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
)

// Relay manages tunnel connections from operators and forwards client traffic.
type Relay struct {
	listenAddr string
	tunnels    map[tunneling.TunnelID]net.Conn
	mu         sync.RWMutex
	listener   net.Listener
	stopCh     chan struct{}
}

// NewRelay creates a new exit relay.
func NewRelay(listenAddr string) *Relay {
	return &Relay{
		listenAddr: listenAddr,
		tunnels:    make(map[tunneling.TunnelID]net.Conn),
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

		conn, err := r.listener.Accept()
		if err != nil {
			continue
		}

		go r.handleConnection(ctx, conn)
	}
}

// handleConnection reads the first line to determine if this is a
// tunnel registration (REGISTER/UNREGISTER) or a client request.
func (r *Relay) handleConnection(ctx context.Context, conn net.Conn) {
	// Set read deadline for first message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}

	firstLine = strings.TrimSpace(firstLine)

	if strings.HasPrefix(firstLine, "REGISTER ") {
		// Registration connections stay open for forwarding
		r.handleRegister(ctx, conn, firstLine)
	} else if strings.HasPrefix(firstLine, "UNREGISTER ") {
		conn.Close()
		r.handleUnregister(firstLine)
	} else {
		// Client connections close after handling the request
		defer conn.Close()
		// Assume this is a client HTTP request
		// First line should be "GET /tunnel/<id> HTTP/1.1" or similar
		r.handleClientRequest(ctx, conn, firstLine, reader)
	}
}

// handleRegister registers a tunnel from an operator.
func (r *Relay) handleRegister(ctx context.Context, conn net.Conn, msg string) {
	parts := strings.Fields(msg)
	if len(parts) != 2 {
		conn.Write([]byte("ERROR: invalid REGISTER format\n"))
		return
	}

	tunnelID := tunneling.TunnelID(parts[1])
	if err := tunnelID.Validate(); err != nil {
		conn.Write([]byte(fmt.Sprintf("ERROR: invalid tunnel ID: %v\n", err)))
		return
	}

	r.mu.Lock()
	r.tunnels[tunnelID] = conn
	r.mu.Unlock()

	conn.Write([]byte("OK\n"))

	// Keep connection open for forwarding
	// Connection will be used bidirectionally:
	// - Read from exit relay (client requests)
	// - Write to exit relay (responses from operator)
}

// handleUnregister removes a tunnel registration.
func (r *Relay) handleUnregister(msg string) {
	parts := strings.Fields(msg)
	if len(parts) != 2 {
		return
	}

	tunnelID := tunneling.TunnelID(parts[1])
	r.mu.Lock()
	if conn, ok := r.tunnels[tunnelID]; ok {
		conn.Close()
		delete(r.tunnels, tunnelID)
	}
	r.mu.Unlock()
}

// handleClientRequest forwards a client's HTTP request to the tunnel operator.
func (r *Relay) handleClientRequest(ctx context.Context, clientConn net.Conn, firstLine string, reader *bufio.Reader) {
	// Parse tunnel ID from HTTP request path
	// Expected format: "GET /tunnel/<tunnel-id> HTTP/1.1" or "GET /tunnel/<tunnel-id>/<path> HTTP/1.1"
	parts := strings.Fields(firstLine)
	if len(parts) < 2 {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	path := parts[1]
	const prefix = "/tunnel/"
	if !strings.HasPrefix(path, prefix) {
		clientConn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	// Extract tunnel ID from path (format: /tunnel/<id> or /tunnel/<id>/...)
	pathAfterPrefix := strings.TrimPrefix(path, prefix)
	tunnelIDStr := pathAfterPrefix
	if idx := strings.Index(pathAfterPrefix, "/"); idx != -1 {
		tunnelIDStr = pathAfterPrefix[:idx]
	}

	tunnelID := tunneling.TunnelID(tunnelIDStr)
	if err := tunnelID.Validate(); err != nil {
		// DEBUG: log validation failure
		fmt.Printf("DEBUG: Tunnel ID validation failed for %q: %v\n", tunnelIDStr, err)
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Lookup tunnel connection
	r.mu.RLock()
	operatorConn, ok := r.tunnels[tunnelID]
	r.mu.RUnlock()

	if !ok {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\n\r\nTunnel not found\n"))
		return
	}

	// Reconstruct full HTTP request
	fullRequest := firstLine + "\r\n"

	// Read remaining headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			fullRequest += "\r\n"
			break
		}
		fullRequest += line
	}

	// Forward to operator
	if _, err := operatorConn.Write([]byte(fullRequest)); err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	// Read response from operator and forward to client
	respBuf := make([]byte, 65536)
	operatorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := operatorConn.Read(respBuf)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	clientConn.Write(respBuf[:n])
}

// Stop gracefully shuts down the relay.
func (r *Relay) Stop(ctx context.Context) error {
	close(r.stopCh)

	if r.listener != nil {
		r.listener.Close()
	}

	r.mu.Lock()
	for _, conn := range r.tunnels {
		conn.Close()
	}
	r.tunnels = make(map[tunneling.TunnelID]net.Conn)
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
