// Package initiator implements the tunnel operator side (localhost → exit relay).
// This is a Phase 6.3 minimal prototype: single-hop, HTTP-only forwarding.
package initiator

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/shroud"
	"github.com/opd-ai/murmur/pkg/tunneling"
	"github.com/opd-ai/murmur/pkg/tunneling/accounting"
	"github.com/opd-ai/murmur/pkg/tunneling/protocol"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// Initiator manages a localhost tunnel to an exit relay.
type Initiator struct {
	config   tunneling.Config
	pubkey   ed25519.PublicKey
	privkey  ed25519.PrivateKey
	tunnelID tunneling.TunnelID
	exitConn net.Conn
	recorder *accounting.Recorder
	beacon   *shroud.Beacon
	mode     string
	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
}

// NewInitiator creates a new tunnel initiator.
func NewInitiator(cfg tunneling.Config, pubkey ed25519.PublicKey, privkey ed25519.PrivateKey) *Initiator {
	return &Initiator{
		config:   cfg,
		pubkey:   pubkey,
		privkey:  privkey,
		tunnelID: tunneling.GenerateTunnelID(pubkey, cfg.TunnelName),
		recorder: accounting.NewRecorder(),
		mode:     "single-hop",
		stopCh:   make(chan struct{}),
	}
}

// SetBeacon configures Shroud relay discovery for multi-hop availability checks.
func (i *Initiator) SetBeacon(beacon *shroud.Beacon) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.beacon = beacon
}

// Start begins forwarding localhost traffic to the exit relay.
// Returns the tunnel ID for clients to connect to.
func (i *Initiator) Start(ctx context.Context) (tunneling.TunnelID, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.running {
		return "", fmt.Errorf("tunnel already running")
	}

	mode, err := i.selectTransportMode()
	if err != nil {
		return "", err
	}
	i.mode = mode

	// Step 1: Connect to exit relay transport endpoint.
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", i.config.ExitRelayAddr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to exit relay %s: %w", i.config.ExitRelayAddr, err)
	}
	i.exitConn = conn

	// Step 2: Send signed tunnel registration cell.
	regPayload, err := protocol.EncodeRegisterCell(i.tunnelID, i.pubkey, i.privkey, i.config.BandwidthLimit)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to encode register cell: %w", err)
	}
	if err := protocol.WriteFrame(conn, protocol.FrameTypeRegister, regPayload); err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to register tunnel: %w", err)
	}

	// Step 3: Read acknowledgment
	ackBuf := make([]byte, 8)
	n, err := conn.Read(ackBuf)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to read ack: %w", err)
	}
	ack := string(ackBuf[:n])
	if ack != "OK\n" {
		conn.Close()
		return "", fmt.Errorf("exit relay rejected registration: %s", ack)
	}

	i.recorder.Register(i.tunnelID)
	i.running = true

	// Step 4: Start forwarding goroutine
	go i.forwardLoop(ctx)

	return i.tunnelID, nil
}

// forwardLoop reads requests from exit relay and forwards to localhost.
func (i *Initiator) forwardLoop(ctx context.Context) {
	defer i.markNotRunning()
	defer i.recorder.Unregister(i.tunnelID)

	for {
		if i.shouldStopLoop(ctx) {
			return
		}

		dataCell, shouldContinue := i.readDataCell()
		if !shouldContinue {
			return
		}

		if dataCell != nil {
			i.forwardCellToLocalhost(ctx, dataCell)
		}
	}
}

// markNotRunning sets the running flag to false on loop exit.
func (i *Initiator) markNotRunning() {
	i.mu.Lock()
	i.running = false
	if i.exitConn != nil {
		i.exitConn.Close()
		i.exitConn = nil
	}
	i.mu.Unlock()
}

// shouldStopLoop checks if the loop should terminate.
func (i *Initiator) shouldStopLoop(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	case <-i.stopCh:
		return true
	default:
		return false
	}
}

// readFromExitRelay reads data from the exit connection with timeout.
func (i *Initiator) readDataCell() (*pb.TunnelDataCell, bool) {
	i.exitConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	frameType, payload, err := protocol.ReadFrame(i.exitConn)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, true
		}
		return nil, false
	}

	if frameType == protocol.FrameTypeTeardown {
		return nil, false
	}
	if frameType != protocol.FrameTypeData {
		return nil, true
	}

	cell := &pb.TunnelDataCell{}
	if err := proto.Unmarshal(payload, cell); err != nil {
		i.recorder.RecordError(i.tunnelID)
		return nil, true
	}
	if string(cell.TunnelId) != string(i.tunnelID) {
		i.recorder.RecordError(i.tunnelID)
		return nil, true
	}

	i.recorder.RecordRequest(i.tunnelID)
	i.recorder.RecordBytesReceived(i.tunnelID, uint64(len(cell.Payload)))
	return cell, true
}

// forwardCellToLocalhost sends request data to localhost:port and replies via data cell.
func (i *Initiator) forwardCellToLocalhost(ctx context.Context, cell *pb.TunnelDataCell) error {
	rewrittenData := i.rewriteHTTPRequest(cell.Payload)

	conn, err := i.connectToLocalhost(ctx)
	if err != nil {
		i.recorder.RecordError(i.tunnelID)
		return err
	}
	defer conn.Close()

	if err := i.sendRequestToLocalhost(conn, rewrittenData); err != nil {
		i.recorder.RecordError(i.tunnelID)
		return err
	}

	return i.relayResponseToExit(conn, cell.Sequence)
}

// rewriteHTTPRequest removes the /tunnel/<id> prefix from the request path.
func (i *Initiator) rewriteHTTPRequest(data []byte) []byte {
	lines := strings.Split(string(data), "\r\n")
	if len(lines) == 0 {
		return data
	}

	firstLine := lines[0]
	parts := strings.Fields(firstLine)
	if len(parts) >= 2 {
		path := parts[1]
		const prefix = "/tunnel/"
		if strings.HasPrefix(path, prefix) {
			pathAfterPrefix := strings.TrimPrefix(path, prefix)
			if idx := strings.Index(pathAfterPrefix, "/"); idx != -1 {
				parts[1] = pathAfterPrefix[idx:]
			} else {
				parts[1] = "/"
			}
			lines[0] = strings.Join(parts, " ")
		}
	}

	return []byte(strings.Join(lines, "\r\n"))
}

// connectToLocalhost establishes a connection to the local service.
func (i *Initiator) connectToLocalhost(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	localhostAddr := fmt.Sprintf("127.0.0.1:%d", i.config.LocalPort)
	conn, err := dialer.DialContext(ctx, "tcp", localhostAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to localhost: %w", err)
	}
	return conn, nil
}

// sendRequestToLocalhost writes the rewritten request to the localhost connection.
func (i *Initiator) sendRequestToLocalhost(conn net.Conn, data []byte) error {
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write to localhost: %w", err)
	}
	return nil
}

// relayResponseToExit reads the localhost response and forwards it to the exit relay.
func (i *Initiator) relayResponseToExit(conn net.Conn, sequence uint32) error {
	resp := make([]byte, 65536)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(resp)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read from localhost: %w", err)
	}

	i.recorder.RecordBytesSent(i.tunnelID, uint64(n))

	respCell := &pb.TunnelDataCell{
		TunnelId: []byte(i.tunnelID),
		Payload:  resp[:n],
		Sequence: sequence,
		IsFinal:  true,
	}
	payload, err := proto.Marshal(respCell)
	if err != nil {
		return fmt.Errorf("marshal response cell: %w", err)
	}

	if err := protocol.WriteFrame(i.exitConn, protocol.FrameTypeData, payload); err != nil {
		return fmt.Errorf("failed to write response to exit: %w", err)
	}

	if i.recorder.QuotaExceeded(i.tunnelID, i.config.BandwidthLimit) {
		_ = i.sendTeardown(pb.TeardownReason_QUOTA_EXCEEDED, "tunnel bandwidth quota exceeded")
		return fmt.Errorf("tunnel bandwidth quota exceeded")
	}

	return nil
}

func (i *Initiator) sendTeardown(reason pb.TeardownReason, message string) error {
	cell := &pb.TunnelTeardownCell{
		TunnelId: []byte(i.tunnelID),
		Reason:   reason,
		Message:  message,
	}
	payload, err := proto.Marshal(cell)
	if err != nil {
		return err
	}
	return protocol.WriteFrame(i.exitConn, protocol.FrameTypeTeardown, payload)
}

// Stop gracefully shuts down the tunnel.
func (i *Initiator) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return nil
	}

	// Signal stop
	select {
	case <-i.stopCh:
	default:
		close(i.stopCh)
	}

	_ = i.sendTeardown(pb.TeardownReason_OPERATOR_REQUEST, "operator requested shutdown")

	if i.exitConn != nil {
		i.exitConn.Close()
	}

	i.running = false
	return nil
}

// Mode returns the current tunnel transport mode.
func (i *Initiator) Mode() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.mode
}

func (i *Initiator) selectTransportMode() (string, error) {
	if i.beacon == nil {
		if i.config.RequireShroud {
			return "", fmt.Errorf("shroud required but no beacon configured")
		}
		return "single-hop", nil
	}

	if i.beacon.RelayCount() < shroud.CircuitLength {
		if i.config.RequireShroud {
			return "", fmt.Errorf("shroud required but only %d relays available", i.beacon.RelayCount())
		}
		_, _ = fmt.Fprintln(os.Stderr, "warning: insufficient Shroud relays, falling back to single-hop mode")
		return "single-hop", nil
	}

	// Circuit readiness check; full circuit forwarding is handled by relay path.
	manager := shroud.NewCircuitManager(i.beacon, nil)
	if _, err := manager.GetCircuit(); err != nil {
		if i.config.RequireShroud {
			return "", fmt.Errorf("shroud required but circuit build failed: %w", err)
		}
		_, _ = fmt.Fprintln(os.Stderr, "warning: Shroud circuit build failed, falling back to single-hop mode")
		return "single-hop", nil
	}

	return "shroud-3hop", nil
}

// TunnelID returns the tunnel's address.
func (i *Initiator) TunnelID() tunneling.TunnelID {
	return i.tunnelID
}

// IsRunning returns true if the tunnel is active.
func (i *Initiator) IsRunning() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.running
}

// SimpleHTTPServer creates a test HTTP server on localhost for testing.
func SimpleHTTPServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from localhost:%d\n", port)
	})
	return &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}
}
