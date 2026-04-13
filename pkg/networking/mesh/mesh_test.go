package mesh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/networking/transport"
)

func TestNewManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	m := NewManager(h.Host)
	if m == nil {
		t.Error("NewManager returned nil")
	}
}

func TestPeerConnectDisconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two hosts
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	m1 := NewManager(h1.Host)
	m1.Start()
	defer m1.Stop()

	// Initially no peers
	if n := m1.PeerCount(); n != 0 {
		t.Errorf("Initial PeerCount = %d, want 0", n)
	}

	// Connect h2 to h1
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Wait for connection notification
	time.Sleep(100 * time.Millisecond)

	// Verify peer is tracked
	if n := m1.PeerCount(); n != 1 {
		t.Errorf("PeerCount after connect = %d, want 1", n)
	}

	// Disconnect
	if err := h2.Close(); err != nil {
		t.Fatalf("Close h2 failed: %v", err)
	}

	// Wait for disconnect notification
	time.Sleep(100 * time.Millisecond)

	// Verify peer is removed
	if n := m1.PeerCount(); n != 0 {
		t.Errorf("PeerCount after disconnect = %d, want 0", n)
	}
}

func TestRecordHeartbeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	m1 := NewManager(h1.Host)
	m1.Start()
	defer m1.Stop()

	// Connect
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Record heartbeat
	m1.RecordHeartbeat(h2.PeerID())
	time.Sleep(50 * time.Millisecond)

	// Verify peer state updated
	peers := m1.Peers()
	if len(peers) != 1 {
		t.Fatalf("Peers count = %d, want 1", len(peers))
	}
}

func TestSetPriority(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	h2, _ := transport.NewHost(ctx, cfg2)
	defer h2.Close()

	m1 := NewManager(h1.Host)
	m1.Start()
	defer m1.Stop()

	// Connect
	h2.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	// Set priority
	m1.SetPriority(h2.PeerID(), PriorityIdentity)

	peers := m1.Peers()
	if len(peers) != 1 {
		t.Fatalf("Peers count = %d, want 1", len(peers))
	}
	if peers[0].Priority != PriorityIdentity {
		t.Errorf("Priority = %v, want PriorityIdentity", peers[0].Priority)
	}
}

func TestNeedsMorePeers(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host)

	// With no peers, should need more
	if !m.NeedsMorePeers() {
		t.Error("NeedsMorePeers should be true with 0 peers")
	}
}

func TestConstants(t *testing.T) {
	if MinPeers != 6 {
		t.Errorf("MinPeers = %d, want 6", MinPeers)
	}
	if MaxPeers != 12 {
		t.Errorf("MaxPeers = %d, want 12", MaxPeers)
	}
	if HeartbeatInterval != 30*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 30s", HeartbeatInterval)
	}
	if MissedHeartbeatsThreshold != 3 {
		t.Errorf("MissedHeartbeatsThreshold = %d, want 3", MissedHeartbeatsThreshold)
	}
}
