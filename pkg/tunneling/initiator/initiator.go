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
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
)

// Initiator manages a localhost tunnel to an exit relay.
type Initiator struct {
	config   tunneling.Config
	pubkey   ed25519.PublicKey
	tunnelID tunneling.TunnelID
	exitConn net.Conn
	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
}

// NewInitiator creates a new tunnel initiator.
func NewInitiator(cfg tunneling.Config, pubkey ed25519.PublicKey) *Initiator {
	return &Initiator{
		config:   cfg,
		pubkey:   pubkey,
		tunnelID: tunneling.GenerateTunnelID(pubkey, cfg.TunnelName),
		stopCh:   make(chan struct{}),
	}
}

// Start begins forwarding localhost traffic to the exit relay.
// Returns the tunnel ID for clients to connect to.
func (i *Initiator) Start(ctx context.Context) (tunneling.TunnelID, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.running {
		return "", fmt.Errorf("tunnel already running")
	}

	// Step 1: Connect to exit relay (single-hop for prototype)
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", i.config.ExitRelayAddr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to exit relay %s: %w", i.config.ExitRelayAddr, err)
	}
	i.exitConn = conn

	// Step 2: Send tunnel registration message
	regMsg := fmt.Sprintf("REGISTER %s\n", i.tunnelID)
	if _, err := conn.Write([]byte(regMsg)); err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to register tunnel: %w", err)
	}

	// Step 3: Read acknowledgment
	ackBuf := make([]byte, 32)
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

	i.running = true

	// Step 4: Start forwarding goroutine
	go i.forwardLoop(ctx)

	return i.tunnelID, nil
}

// forwardLoop reads requests from exit relay and forwards to localhost.
func (i *Initiator) forwardLoop(ctx context.Context) {
	defer func() {
		i.mu.Lock()
		i.running = false
		i.mu.Unlock()
	}()

	buf := make([]byte, 65536)
	for {
		select {
		case <-ctx.Done():
			return
		case <-i.stopCh:
			return
		default:
		}

		// Set read deadline to allow periodic context checks
		i.exitConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := i.exitConn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // timeout, check context and retry
			}
			return // connection closed or error
		}

		// Simple protocol: REQUEST_ID<space>HTTP_REQUEST_BYTES
		// For prototype, just forward entire buffer to localhost
		if err := i.forwardToLocalhost(ctx, buf[:n]); err != nil {
			// Log error but continue forwarding
			continue
		}
	}
}

// forwardToLocalhost sends request data to localhost:port.
func (i *Initiator) forwardToLocalhost(ctx context.Context, data []byte) error {
	// Parse HTTP request and rewrite path
	// Input: "GET /tunnel/<id>/path HTTP/1.1\r\n..."
	// Output: "GET /path HTTP/1.1\r\n..."

	lines := strings.Split(string(data), "\r\n")
	if len(lines) == 0 {
		return fmt.Errorf("empty request")
	}

	// Rewrite first line to remove /tunnel/<id> prefix
	firstLine := lines[0]
	parts := strings.Fields(firstLine)
	if len(parts) >= 2 {
		// Extract actual path from /tunnel/<id>/path format
		path := parts[1]
		const prefix = "/tunnel/"
		if strings.HasPrefix(path, prefix) {
			// Find the second slash after /tunnel/
			pathAfterPrefix := strings.TrimPrefix(path, prefix)
			if idx := strings.Index(pathAfterPrefix, "/"); idx != -1 {
				// Path is /tunnel/<id>/path -> extract /path
				actualPath := pathAfterPrefix[idx:]
				parts[1] = actualPath
			} else {
				// Path is /tunnel/<id> -> use /
				parts[1] = "/"
			}
			lines[0] = strings.Join(parts, " ")
		}
	}

	rewrittenData := []byte(strings.Join(lines, "\r\n"))

	// Connect to localhost
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	localhostAddr := fmt.Sprintf("127.0.0.1:%d", i.config.LocalPort)
	conn, err := dialer.DialContext(ctx, "tcp", localhostAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to localhost: %w", err)
	}
	defer conn.Close()

	// Forward request
	if _, err := conn.Write(rewrittenData); err != nil {
		return fmt.Errorf("failed to write to localhost: %w", err)
	}

	// Read response and forward back to exit relay
	resp := make([]byte, 65536)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(resp)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read from localhost: %w", err)
	}

	if _, err := i.exitConn.Write(resp[:n]); err != nil {
		return fmt.Errorf("failed to write response to exit: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the tunnel.
func (i *Initiator) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return nil
	}

	// Signal stop
	close(i.stopCh)

	// Send unregister message
	if i.exitConn != nil {
		unregMsg := fmt.Sprintf("UNREGISTER %s\n", i.tunnelID)
		i.exitConn.Write([]byte(unregMsg))
		i.exitConn.Close()
	}

	i.running = false
	return nil
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
