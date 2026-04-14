package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestNewHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate a test keypair
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false // Disable DHT for faster test

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	// Verify host is created
	if h.Host == nil {
		t.Error("Host.Host is nil")
	}

	// Verify PeerID is derived from key
	if h.PeerID() == "" {
		t.Error("PeerID is empty")
	}

	// Verify host has listen addresses
	addrs := h.Addrs()
	if len(addrs) == 0 {
		t.Error("Host has no listen addresses")
	}

	t.Logf("Host created with PeerID: %s", h.PeerID())
	t.Logf("Listen addresses: %v", addrs)
}

func TestNewHostRequiresPrivateKey(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	// Don't set PrivateKey

	_, err := NewHost(ctx, cfg)
	if err == nil {
		t.Error("NewHost should fail without private key")
	}
}

func TestHostAddrInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	info := h.AddrInfo()
	if info.ID != h.PeerID() {
		t.Errorf("AddrInfo.ID = %s, want %s", info.ID, h.PeerID())
	}
	if len(info.Addrs) == 0 {
		t.Error("AddrInfo.Addrs is empty")
	}
}

func TestTwoHostsConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two hosts
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, err := NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Connect h2 to h1
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("h2.Connect to h1 failed: %v", err)
	}

	// Verify connection
	conns := h2.Network().ConnsToPeer(h1.PeerID())
	if len(conns) == 0 {
		t.Error("h2 has no connections to h1")
	}

	t.Logf("h2 (%s) connected to h1 (%s)", h2.PeerID(), h1.PeerID())
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.ListenAddrs) == 0 {
		t.Error("DefaultConfig has no listen addresses")
	}

	if !cfg.EnableDHT {
		t.Error("DefaultConfig should enable DHT")
	}
}

func TestNewHostWithDHT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true
	cfg.DHTServerMode = true

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost with DHT failed: %v", err)
	}
	defer h.Close()

	// Verify DHT is created
	if h.DHT() == nil {
		t.Error("DHT() returned nil when DHT is enabled")
	}
}

func TestNewHostWithDHTClientMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true
	cfg.DHTServerMode = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost with DHT client mode failed: %v", err)
	}
	defer h.Close()

	// Verify DHT is created
	if h.DHT() == nil {
		t.Error("DHT() returned nil when DHT is enabled in client mode")
	}
}

func TestNewHostInvalidListenAddr(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.ListenAddrs = []string{"invalid-multiaddr"}

	_, err := NewHost(ctx, cfg)
	if err == nil {
		t.Error("NewHost should fail with invalid listen address")
	}
}

func TestHostDHTNilWhenDisabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	// Verify DHT is nil when disabled
	if h.DHT() != nil {
		t.Error("DHT() should return nil when DHT is disabled")
	}
}

func TestConfigTimeouts(t *testing.T) {
	// Verify timeout constants are set
	if DefaultConnectionTimeout == 0 {
		t.Error("DefaultConnectionTimeout should not be zero")
	}
	if DefaultStreamTimeout == 0 {
		t.Error("DefaultStreamTimeout should not be zero")
	}
	if DefaultIdleTimeout == 0 {
		t.Error("DefaultIdleTimeout should not be zero")
	}

	// Verify they are reasonable values
	if DefaultConnectionTimeout != 30*time.Second {
		t.Errorf("DefaultConnectionTimeout = %v, want 30s", DefaultConnectionTimeout)
	}
	if DefaultStreamTimeout != 60*time.Second {
		t.Errorf("DefaultStreamTimeout = %v, want 60s", DefaultStreamTimeout)
	}
	if DefaultIdleTimeout != 30*time.Second {
		t.Errorf("DefaultIdleTimeout = %v, want 30s", DefaultIdleTimeout)
	}
}

func TestHostCloseWithDHT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}

	// Close should succeed and close DHT
	if err := h.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestHostCloseWithoutDHT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}

	// Close should succeed even without DHT
	if err := h.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestHostAddrsNotEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	addrs := h.Addrs()
	if len(addrs) == 0 {
		t.Error("Host should have at least one listen address")
	}

	// Verify addresses are valid multiaddrs
	for i, addr := range addrs {
		if addr == nil {
			t.Errorf("Address %d is nil", i)
		}
		if addr.String() == "" {
			t.Errorf("Address %d has empty string representation", i)
		}
	}
}

func TestConfigWithBootstrapPeers(t *testing.T) {
	cfg := DefaultConfig()

	// Bootstrap peers should be empty by default
	if len(cfg.BootstrapPeers) != 0 {
		t.Errorf("Default BootstrapPeers should be empty, got %d", len(cfg.BootstrapPeers))
	}
}

func TestDefaultListenAddrs(t *testing.T) {
	cfg := DefaultConfig()

	// Should have both TCP and QUIC
	hasTCP := false
	hasQUIC := false
	for _, addr := range cfg.ListenAddrs {
		if addr == "/ip4/0.0.0.0/tcp/0" {
			hasTCP = true
		}
		if addr == "/ip4/0.0.0.0/udp/0/quic-v1" {
			hasQUIC = true
		}
	}

	if !hasTCP {
		t.Error("Default config should include TCP listen address")
	}
	if !hasQUIC {
		t.Error("Default config should include QUIC listen address")
	}
}

func TestDefaultConfigWithWebSocket(t *testing.T) {
	cfg := DefaultConfigWithWebSocket()

	if !cfg.EnableWebSocket {
		t.Error("DefaultConfigWithWebSocket should have WebSocket enabled")
	}

	// Should have TCP, QUIC, and WebSocket
	hasTCP := false
	hasQUIC := false
	hasWS := false
	for _, addr := range cfg.ListenAddrs {
		if addr == "/ip4/0.0.0.0/tcp/0" {
			hasTCP = true
		}
		if addr == "/ip4/0.0.0.0/udp/0/quic-v1" {
			hasQUIC = true
		}
		if addr == "/ip4/0.0.0.0/tcp/0/ws" {
			hasWS = true
		}
	}

	if !hasTCP {
		t.Error("Config with WebSocket should include TCP listen address")
	}
	if !hasQUIC {
		t.Error("Config with WebSocket should include QUIC listen address")
	}
	if !hasWS {
		t.Error("Config with WebSocket should include WebSocket listen address")
	}
}

func TestNewHostWithWebSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := DefaultConfigWithWebSocket()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost with WebSocket failed: %v", err)
	}
	defer h.Close()

	// Verify host has listen addresses including WebSocket
	addrs := h.Addrs()
	if len(addrs) == 0 {
		t.Error("Host should have listen addresses")
	}

	t.Logf("Host with WebSocket created, addresses: %v", addrs)
}
