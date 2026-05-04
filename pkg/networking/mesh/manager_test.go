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

	m := NewManager(h.Host, 0)
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

	m1 := NewManager(h1.Host, 0)
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

	m1 := NewManager(h1.Host, 0)
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

	m1 := NewManager(h1.Host, 0)
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

	m := NewManager(h.Host, 0)

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
	// HeartbeatInterval is now configurable via NewManager parameter, no longer a constant
	if MissedHeartbeatsThreshold != 3 {
		t.Errorf("MissedHeartbeatsThreshold = %d, want 3", MissedHeartbeatsThreshold)
	}
}

func TestHasTooManyPeers(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)

	// With no peers, should not have too many
	if m.HasTooManyPeers() {
		t.Error("HasTooManyPeers should be false with 0 peers")
	}
}

func TestPruneLowestPriorityNoPeers(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)

	// Pruning with no peers should return empty
	pruned := m.PruneLowestPriority()
	if pruned != "" {
		t.Errorf("PruneLowestPriority with no peers returned %s, want empty", pruned)
	}
}

func TestPeersSnapshot(t *testing.T) {
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

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Connect
	h2.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	// Get peers snapshot
	peers := m1.Peers()
	if len(peers) != 1 {
		t.Fatalf("Peers count = %d, want 1", len(peers))
	}

	// Verify snapshot is a copy
	if peers[0].ID != h2.PeerID() {
		t.Errorf("Peer ID = %s, want %s", peers[0].ID, h2.PeerID())
	}
}

func TestPriorityConstants(t *testing.T) {
	// Verify priority ordering (lower value = higher priority)
	if PriorityIdentity >= PriorityGossip {
		t.Error("PriorityIdentity should be lower value than PriorityGossip")
	}
	if PriorityGossip >= PriorityRandom {
		t.Error("PriorityGossip should be lower value than PriorityRandom")
	}
}

func TestSetPriorityNonExistentPeer(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)

	// Setting priority on non-existent peer should not panic
	m.SetPriority("12D3KooWNonExistent", PriorityIdentity)

	// Verify no crash and empty peers
	if m.PeerCount() != 0 {
		t.Errorf("PeerCount = %d after setting priority on non-existent peer, want 0", m.PeerCount())
	}
}

func TestRecordHeartbeatNonExistentPeer(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)
	m.Start()
	defer m.Stop()

	// Recording heartbeat for non-existent peer should not panic
	m.RecordHeartbeat("12D3KooWNonExistent")
	time.Sleep(50 * time.Millisecond)

	// Verify no crash
	if m.PeerCount() != 0 {
		t.Errorf("PeerCount = %d, want 0", m.PeerCount())
	}
}

func TestManagerStopIdempotent(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)
	m.Start()

	// Stop multiple times should not panic
	m.Stop()
	m.Stop()
}

func TestPeerStateFields(t *testing.T) {
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

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Connect
	h2.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	peers := m1.Peers()
	if len(peers) != 1 {
		t.Fatalf("Peers count = %d, want 1", len(peers))
	}

	peer := peers[0]
	if peer.ID != h2.PeerID() {
		t.Errorf("Peer ID mismatch")
	}
	if peer.Priority != PriorityRandom {
		t.Errorf("Default priority = %d, want PriorityRandom", peer.Priority)
	}
	if peer.MissedHeartbeat != 0 {
		t.Errorf("Initial MissedHeartbeat = %d, want 0", peer.MissedHeartbeat)
	}
	if peer.LastSeen.IsZero() {
		t.Error("LastSeen should not be zero")
	}
}

func TestCheckHeartbeatsNoPeers(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, _ := transport.NewHost(ctx, cfg)
	defer h.Close()

	m := NewManager(h.Host, 0)

	// Directly call checkHeartbeats with no peers - should not panic
	m.checkHeartbeats()

	if m.PeerCount() != 0 {
		t.Errorf("PeerCount = %d after checkHeartbeats, want 0", m.PeerCount())
	}
}

func TestFindLowestPriorityPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create 4 hosts: 1 manager + 3 peers with different priorities
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	_, priv3, _ := ed25519.GenerateKey(rand.Reader)
	_, priv4, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	cfg3 := transport.DefaultConfig()
	cfg3.PrivateKey = priv3
	cfg3.EnableDHT = false

	cfg4 := transport.DefaultConfig()
	cfg4.PrivateKey = priv4
	cfg4.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	h2, _ := transport.NewHost(ctx, cfg2)
	defer h2.Close()

	h3, _ := transport.NewHost(ctx, cfg3)
	defer h3.Close()

	h4, _ := transport.NewHost(ctx, cfg4)
	defer h4.Close()

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Connect all peers to h1
	h2.Connect(ctx, h1.AddrInfo())
	h3.Connect(ctx, h1.AddrInfo())
	h4.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	if m1.PeerCount() != 3 {
		t.Fatalf("Expected 3 peers, got %d", m1.PeerCount())
	}

	// Set different priorities
	m1.SetPriority(h2.PeerID(), PriorityIdentity) // Highest priority (lowest value)
	m1.SetPriority(h3.PeerID(), PriorityGossip)   // Medium priority
	m1.SetPriority(h4.PeerID(), PriorityRandom)   // Lowest priority (highest value)

	// Find lowest priority peer
	m1.mu.RLock()
	lowestID := m1.findLowestPriorityPeerLocked()
	m1.mu.RUnlock()

	if lowestID != h4.PeerID() {
		t.Errorf("Lowest priority peer should be h4 (%s), got %s", h4.PeerID(), lowestID)
	}
}

func TestRemoveLowestPriorityPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	_, priv3, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	cfg3 := transport.DefaultConfig()
	cfg3.PrivateKey = priv3
	cfg3.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	h2, _ := transport.NewHost(ctx, cfg2)
	defer h2.Close()

	h3, _ := transport.NewHost(ctx, cfg3)
	defer h3.Close()

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Connect peers
	h2.Connect(ctx, h1.AddrInfo())
	h3.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	if m1.PeerCount() != 2 {
		t.Fatalf("Expected 2 peers, got %d", m1.PeerCount())
	}

	// Set different priorities
	m1.SetPriority(h2.PeerID(), PriorityIdentity) // Higher priority
	m1.SetPriority(h3.PeerID(), PriorityRandom)   // Lower priority

	// Remove lowest priority peer
	removed := m1.removeLowestPriorityPeer()

	if removed != h3.PeerID() {
		t.Errorf("Removed peer should be h3 (%s), got %s", h3.PeerID(), removed)
	}

	// Verify peer count reduced
	if m1.PeerCount() != 1 {
		t.Errorf("Expected 1 peer after removal, got %d", m1.PeerCount())
	}
}

func TestPruneLowestPriorityWithPeers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Create and connect more than MaxPeers hosts
	hosts := make([]*transport.Host, MaxPeers+2)
	for i := 0; i < MaxPeers+2; i++ {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		cfg := transport.DefaultConfig()
		cfg.PrivateKey = priv
		cfg.EnableDHT = false
		h, _ := transport.NewHost(ctx, cfg)
		hosts[i] = h
		defer h.Close()
	}

	// Connect all hosts
	for _, h := range hosts {
		h.Connect(ctx, h1.AddrInfo())
	}
	time.Sleep(200 * time.Millisecond)

	// We should have more than MaxPeers
	if m1.PeerCount() <= MaxPeers {
		t.Skipf("Not enough peers connected (%d), skipping prune test", m1.PeerCount())
	}

	// Now prune should work
	if !m1.HasTooManyPeers() {
		t.Error("HasTooManyPeers should be true")
	}

	pruned := m1.PruneLowestPriority()
	if pruned == "" {
		t.Error("PruneLowestPriority should return non-empty peer ID")
	}
}

func TestCheckHeartbeatsMissedThreshold(t *testing.T) {
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

	m1 := NewManager(h1.Host, 0)
	// Don't call Start() so we control heartbeat checking manually

	// Connect
	h2.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	if m1.PeerCount() != 1 {
		t.Fatalf("Expected 1 peer, got %d", m1.PeerCount())
	}

	// Manually set LastSeen to far in the past to simulate missed heartbeats
	m1.mu.Lock()
	for _, state := range m1.peers {
		state.LastSeen = time.Now().Add(-m1.heartbeatInterval * 3)
		state.MissedHeartbeat = MissedHeartbeatsThreshold // Trigger disconnect threshold
	}
	m1.mu.Unlock()

	// Call checkHeartbeats which should disconnect the stale peer
	m1.checkHeartbeats()

	// Wait a bit for the async close
	time.Sleep(100 * time.Millisecond)

	// Peer should be removed from the map (even if connection isn't closed yet)
	if m1.PeerCount() != 0 {
		t.Errorf("Expected 0 peers after checkHeartbeats with missed threshold, got %d", m1.PeerCount())
	}
}
