package mesh

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestPartitionStateString(t *testing.T) {
	tests := []struct {
		state PartitionState
		want  string
	}{
		{StateNormal, "normal"},
		{StateDegraded, "degraded"},
		{StatePartitioned, "partitioned"},
		{PartitionState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("PartitionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewPartitionManager(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	if pm.h != h {
		t.Error("Host not set correctly")
	}

	if pm.State() != StateNormal {
		t.Errorf("Initial state = %v, want StateNormal", pm.State())
	}
}

func TestPartitionManagerStatus(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	status := pm.Status()

	if status.State != StateNormal {
		t.Errorf("Status.State = %v, want StateNormal", status.State)
	}

	if status.PeerCount != 0 {
		t.Errorf("Status.PeerCount = %d, want 0", status.PeerCount)
	}

	if status.HealingActive {
		t.Error("HealingActive should be false initially")
	}

	if status.StateDuration < 0 {
		t.Error("StateDuration should be non-negative")
	}
}

func TestPartitionManagerCalculateState(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	tests := []struct {
		peerCount int
		want      PartitionState
	}{
		{0, StatePartitioned},
		{1, StatePartitioned},
		{2, StateDegraded},
		{3, StateDegraded},
		{4, StateNormal},
		{10, StateNormal},
	}

	for _, tt := range tests {
		got := pm.calculateState(tt.peerCount)
		if got != tt.want {
			t.Errorf("calculateState(%d) = %v, want %v", tt.peerCount, got, tt.want)
		}
	}
}

func TestPartitionManagerIsPartitioned(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	// Initially not partitioned (state is Normal)
	if pm.IsPartitioned() {
		t.Error("Should not be partitioned initially")
	}

	// Check IsDegraded (also false when Normal)
	if pm.IsDegraded() {
		t.Error("Should not be degraded initially")
	}
}

func TestPartitionManagerCallbacks(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	var stateChangeCalled int32
	var healingStartCalled int32

	pm.SetCallbacks(PartitionCallbacks{
		OnStateChange: func(old, new PartitionState) {
			atomic.AddInt32(&stateChangeCalled, 1)
		},
		OnHealingStart: func() {
			atomic.AddInt32(&healingStartCalled, 1)
		},
	})

	// Verify callbacks are set
	pm.mu.RLock()
	hasStateChange := pm.callbacks.OnStateChange != nil
	hasHealingStart := pm.callbacks.OnHealingStart != nil
	pm.mu.RUnlock()

	if !hasStateChange {
		t.Error("OnStateChange callback not set")
	}
	if !hasHealingStart {
		t.Error("OnHealingStart callback not set")
	}
}

func TestPartitionManagerRememberPeer(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	// Create a mock peer ID
	h2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create second host: %v", err)
	}
	defer h2.Close()

	// Add peer addresses to peerstore
	h.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), time.Hour)

	// Remember the peer
	pm.rememberPeer(h2.ID())

	pm.mu.RLock()
	peerCount := len(pm.historicalPeers)
	pm.mu.RUnlock()

	if peerCount != 1 {
		t.Errorf("Historical peers = %d, want 1", peerCount)
	}
}

func TestPartitionManagerHistoricalPeerDedup(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	h2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create second host: %v", err)
	}
	defer h2.Close()

	h.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), time.Hour)

	// Remember same peer multiple times
	pm.rememberPeer(h2.ID())
	pm.rememberPeer(h2.ID())
	pm.rememberPeer(h2.ID())

	pm.mu.RLock()
	peerCount := len(pm.historicalPeers)
	pm.mu.RUnlock()

	if peerCount != 1 {
		t.Errorf("Historical peers = %d, want 1 (should deduplicate)", peerCount)
	}
}

func TestGracefulDegradation(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	gd := NewGracefulDegradation(pm)

	// In normal state, all operations should be allowed
	ops := gd.AllowedOperations()

	if !ops.WavePublish {
		t.Error("WavePublish should be allowed in normal state")
	}
	if !ops.ShroudCircuits {
		t.Error("ShroudCircuits should be allowed in normal state")
	}
	if !ops.FullGossip {
		t.Error("FullGossip should be allowed in normal state")
	}
}

func TestGracefulDegradationMethods(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	gd := NewGracefulDegradation(pm)

	// In normal state
	if gd.ShouldQueueWaves() {
		t.Error("ShouldQueueWaves should be false in normal state")
	}
	if gd.ShouldReduceGossip() {
		t.Error("ShouldReduceGossip should be false in normal state")
	}
	if gd.ShouldDeferShroud() {
		t.Error("ShouldDeferShroud should be false in normal state")
	}
}

func TestHealingProtocol(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	hp := NewHealingProtocol(pm, nil)

	if hp.pm != pm {
		t.Error("PartitionManager not set correctly")
	}
}

func TestHealingProtocolAddHistoricalPeer(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	hp := NewHealingProtocol(pm, nil)

	h2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create second host: %v", err)
	}
	defer h2.Close()

	// Add addresses to peerstore first
	h.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), time.Hour)

	addrInfo := peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}
	hp.AddHistoricalPeer(addrInfo)

	pm.mu.RLock()
	peerCount := len(pm.historicalPeers)
	pm.mu.RUnlock()

	if peerCount != 1 {
		t.Errorf("Historical peers = %d, want 1", peerCount)
	}
}

func TestDegradedOperationsAllStates(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	gd := NewGracefulDegradation(pm)

	// Test each state manually by simulating state transitions
	tests := []struct {
		name            string
		state           PartitionState
		wantWavePublish bool
		wantShroud      bool
	}{
		{"normal", StateNormal, true, true},
		{"degraded", StateDegraded, true, false},
		{"partitioned", StatePartitioned, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily set state for testing
			pm.mu.Lock()
			oldState := pm.state
			pm.state = tt.state
			pm.mu.Unlock()

			ops := gd.AllowedOperations()

			if ops.WavePublish != tt.wantWavePublish {
				t.Errorf("WavePublish = %v, want %v", ops.WavePublish, tt.wantWavePublish)
			}
			if ops.ShroudCircuits != tt.wantShroud {
				t.Errorf("ShroudCircuits = %v, want %v", ops.ShroudCircuits, tt.wantShroud)
			}

			// Restore state
			pm.mu.Lock()
			pm.state = oldState
			pm.mu.Unlock()
		})
	}
}

func TestPartitionManagerConcurrency(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	pm := NewPartitionManager(h, nil, nil)
	defer pm.Stop()

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pm.State()
			_ = pm.Status()
			_ = pm.IsPartitioned()
			_ = pm.IsDegraded()
		}()
	}

	wg.Wait()
}
