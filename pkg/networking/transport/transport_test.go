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
