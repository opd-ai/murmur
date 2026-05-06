// Package client implements the tunnel client (external → tunnel → localhost).
// This is a Phase 6.3 minimal prototype: single-hop, HTTP-only requests.
package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
)

// Client sends HTTP requests through a tunnel.
type Client struct {
	tunnelID  tunneling.TunnelID
	relayAddr string
}

// NewClient creates a new tunnel client for the given tunnel address.
func NewClient(tunnelAddr, relayAddr string) (*Client, error) {
	id, err := tunneling.ParseTunnelAddress(tunnelAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid tunnel address: %w", err)
	}

	return &Client{
		tunnelID:  id,
		relayAddr: relayAddr,
	}, nil
}

// Get sends a GET request through the tunnel.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, "GET", path, nil)
}

// Post sends a POST request through the tunnel.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, "POST", path, body)
}

// do sends an HTTP request through the tunnel.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Connect to exit relay
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", c.relayAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to relay: %w", err)
	}
	defer conn.Close()

	// Build HTTP request with tunnel path
	// Format: GET /tunnel/<tunnel-id><path> HTTP/1.1
	// e.g., "/tunnel/test-abc123def/" for root path
	tunnelPath := fmt.Sprintf("/tunnel/%s%s", string(c.tunnelID), path)
	req := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: tunnel\r\nConnection: close\r\n\r\n", method, tunnelPath)

	// Send request
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}

// TunnelID returns the client's tunnel ID.
func (c *Client) TunnelID() tunneling.TunnelID {
	return c.tunnelID
}
