package tunneling_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
	"github.com/opd-ai/murmur/pkg/tunneling/client"
	"github.com/opd-ai/murmur/pkg/tunneling/initiator"
	"github.com/opd-ai/murmur/pkg/tunneling/relay"
)

// TestEndToEndTunnel validates the complete tunneling flow:
// Client → Relay → Initiator → Localhost → back
func TestEndToEndTunnel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Start a simple localhost HTTP server
	testPort := 18080
	server := initiator.SimpleHTTPServer(testPort)
	go server.ListenAndServe()
	defer server.Shutdown(ctx)
	time.Sleep(100 * time.Millisecond) // let server start

	// Step 2: Start exit relay on random port
	relayListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create relay listener: %v", err)
	}
	relayAddr := relayListener.Addr().String()
	relayListener.Close() // close to let relay bind it

	r := relay.NewRelay(relayAddr)
	if err := r.Start(ctx); err != nil {
		t.Fatalf("failed to start relay: %v", err)
	}
	defer r.Stop(ctx)
	time.Sleep(100 * time.Millisecond) // let relay start

	// Step 3: Generate operator keypair and start tunnel initiator
	pubkey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	cfg := tunneling.Config{
		LocalPort:     testPort,
		TunnelName:    "test-tunnel",
		ExitRelayAddr: relayAddr,
		Ephemeral:     false,
	}
	init := initiator.NewInitiator(cfg, pubkey)
	tunnelID, err := init.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start initiator: %v", err)
	}
	defer init.Stop(ctx)

	t.Logf("Tunnel started: %s", tunnelID.String())
	time.Sleep(200 * time.Millisecond) // let tunnel establish

	// Step 4: Create client and send request through tunnel
	c, err := client.NewClient(tunnelID.String(), relayAddr)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := c.Get(ctx, "/")
	if err != nil {
		t.Fatalf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	// Step 5: Validate response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := fmt.Sprintf("Hello from localhost:%d\n", testPort)
	if string(body) != expected {
		t.Errorf("unexpected response body:\nwant: %q\ngot:  %q", expected, string(body))
	}

	t.Logf("✅ End-to-end tunnel test passed")
}

// TestTunnelIDDeterminism validates that tunnel IDs are deterministic.
func TestTunnelIDDeterminism(t *testing.T) {
	pubkey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	id1 := tunneling.GenerateTunnelID(pubkey, "test")
	id2 := tunneling.GenerateTunnelID(pubkey, "test")

	if id1 != id2 {
		t.Errorf("tunnel IDs not deterministic: %q != %q", id1, id2)
	}
}

// TestTunnelNotFound validates error handling when tunnel doesn't exist.
func TestTunnelNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start relay
	relayListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create relay listener: %v", err)
	}
	relayAddr := relayListener.Addr().String()
	relayListener.Close()

	r := relay.NewRelay(relayAddr)
	if err := r.Start(ctx); err != nil {
		t.Fatalf("failed to start relay: %v", err)
	}
	defer r.Stop(ctx)
	time.Sleep(100 * time.Millisecond)

	// Try to connect to non-existent tunnel
	c, err := client.NewClient("murmur://tunnel/nonexist-abcdefghijklm", relayAddr)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := c.Get(ctx, "/")
	if err != nil {
		t.Fatalf("expected 502 response, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502 Bad Gateway, got %d", resp.StatusCode)
	}
}
