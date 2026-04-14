package mesh

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestChurnHandlingConstants(t *testing.T) {
	// Verify constants are reasonable
	if ChurnDetectionWindow < time.Minute {
		t.Error("ChurnDetectionWindow should be at least 1 minute")
	}
	if HighChurnThreshold < 1 {
		t.Error("HighChurnThreshold should be at least 1")
	}
	if RepairCooldown < time.Second {
		t.Error("RepairCooldown should be at least 1 second")
	}
	if DHTRefreshInterval < time.Minute {
		t.Error("DHTRefreshInterval should be at least 1 minute")
	}
	if ReconnectAttemptLimit < 1 {
		t.Error("ReconnectAttemptLimit should be at least 1")
	}
}

func TestNewChurnHandler(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	// Create without DHT (nil is allowed)
	ch := NewChurnHandler(h, nil, nil)
	if ch == nil {
		t.Fatal("expected non-nil churn handler")
	}
	if ch.h != h {
		t.Error("host not set correctly")
	}

	ch.Stop()
}

func TestChurnHandler_MarkImportant(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	testPeer := peer.ID("test-peer-1")

	ch.MarkImportant(testPeer)

	stats := ch.Stats()
	if stats.TotalImportantPeers != 1 {
		t.Errorf("expected 1 important peer, got %d", stats.TotalImportantPeers)
	}

	ch.UnmarkImportant(testPeer)

	stats = ch.Stats()
	if stats.TotalImportantPeers != 0 {
		t.Errorf("expected 0 important peers, got %d", stats.TotalImportantPeers)
	}
}

func TestChurnHandler_SetCallbacks(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	highChurnCalled := false
	ch.SetCallbacks(ChurnCallbacks{
		OnHighChurn: func() { highChurnCalled = true },
	})

	// Verify callback is set (can't easily trigger without actual disconnects)
	_ = highChurnCalled
}

func TestChurnHandler_Stats(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	stats := ch.Stats()

	if stats.DisconnectsLastWindow != 0 {
		t.Errorf("expected 0 disconnects, got %d", stats.DisconnectsLastWindow)
	}
	if stats.ChurnRatePerMinute != 0 {
		t.Errorf("expected 0 churn rate, got %f", stats.ChurnRatePerMinute)
	}
	if stats.IsHighChurn {
		t.Error("should not be high churn initially")
	}
	if stats.TotalImportantPeers != 0 {
		t.Errorf("expected 0 important peers, got %d", stats.TotalImportantPeers)
	}
}

func TestChurnHandler_StartStop(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)

	ch.Start()

	// Give background tasks time to start
	time.Sleep(50 * time.Millisecond)

	ch.Stop()

	// Should stop cleanly without panic
}

func TestChurnHandler_isHighChurn(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	// Initially not high churn
	if ch.isHighChurn() {
		t.Error("should not be high churn initially")
	}

	// Simulate many disconnects
	ch.mu.Lock()
	for i := 0; i < 100; i++ {
		ch.disconnectTimes = append(ch.disconnectTimes, time.Now())
	}
	ch.mu.Unlock()

	// Should now be high churn
	if !ch.isHighChurn() {
		t.Error("should be high churn after many disconnects")
	}
}

func TestChurnHandler_pruneDisconnectTimes(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	// Add old disconnect times
	ch.mu.Lock()
	oldTime := time.Now().Add(-ChurnDetectionWindow * 2)
	ch.disconnectTimes = append(ch.disconnectTimes, oldTime)
	ch.disconnectTimes = append(ch.disconnectTimes, time.Now())
	ch.mu.Unlock()

	// Prune
	ch.pruneDisconnectTimes()

	ch.mu.RLock()
	count := len(ch.disconnectTimes)
	ch.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 disconnect time after prune, got %d", count)
	}
}

func TestNewPartitionDetector(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	pd := NewPartitionDetector(h, 2)

	if pd == nil {
		t.Fatal("expected non-nil partition detector")
	}
	if pd.h != h {
		t.Error("host not set correctly")
	}
	if pd.minConnected != 2 {
		t.Errorf("minConnected = %d, expected 2", pd.minConnected)
	}
}

func TestPartitionDetector_IsPartitioned(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	pd := NewPartitionDetector(h, 2)

	// With no peers and minConnected=2, should be partitioned
	// But the initial state isn't partitioned until a connection change occurs
	// Let's manually trigger the check
	pd.onConnectionChange()

	if !pd.IsPartitioned() {
		t.Error("should be partitioned with 0 peers and minConnected=2")
	}
}

func TestPartitionDetector_PartitionDuration(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	pd := NewPartitionDetector(h, 2)

	// Not partitioned initially
	if pd.PartitionDuration() != 0 {
		t.Error("should have 0 duration when not partitioned")
	}

	// Trigger partition
	pd.onConnectionChange()

	time.Sleep(50 * time.Millisecond)

	duration := pd.PartitionDuration()
	if duration < 50*time.Millisecond {
		t.Errorf("partition duration should be >= 50ms, got %v", duration)
	}
}

func TestPartitionDetector_SetCallback(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	pd := NewPartitionDetector(h, 0) // minConnected=0 so won't be partitioned

	callbackCalled := false
	var partitioned bool

	pd.SetPartitionCallback(func(p bool) {
		callbackCalled = true
		partitioned = p
	})

	// Trigger partition by setting minConnected higher
	pd.mu.Lock()
	pd.minConnected = 10
	pd.mu.Unlock()

	pd.onConnectionChange()

	if !callbackCalled {
		t.Error("callback should have been called")
	}
	if !partitioned {
		t.Error("should be partitioned")
	}
}

func TestChurnStats_Fields(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	// Add some state
	ch.MarkImportant(peer.ID("peer-1"))
	ch.MarkImportant(peer.ID("peer-2"))

	ch.mu.Lock()
	ch.disconnectTimes = append(ch.disconnectTimes, time.Now())
	ch.lastRepair = time.Now()
	ch.mu.Unlock()

	stats := ch.Stats()

	if stats.DisconnectsLastWindow != 1 {
		t.Errorf("expected 1 disconnect, got %d", stats.DisconnectsLastWindow)
	}
	if stats.TotalImportantPeers != 2 {
		t.Errorf("expected 2 important peers, got %d", stats.TotalImportantPeers)
	}
	if stats.LastRepairTime.IsZero() {
		t.Error("LastRepairTime should not be zero")
	}
}

func TestChurnHandler_needsMeshRepair(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	// Without degree controller
	ch := NewChurnHandler(h, nil, nil)
	defer ch.Stop()

	if ch.needsMeshRepair() {
		t.Error("should not need repair without degree controller")
	}

	// With degree controller
	manager := NewManager(h)
	degreeCtrl := NewDegreeController(h, manager)

	ch2 := NewChurnHandler(h, nil, degreeCtrl)
	defer ch2.Stop()

	// With no peers, should need repair
	if !ch2.needsMeshRepair() {
		t.Error("should need repair with 0 peers")
	}
}

func TestChurnCallbacks_AllFields(t *testing.T) {
	// Test that callbacks struct has all expected fields
	callbacks := ChurnCallbacks{
		OnHighChurn:  func() {},
		OnMeshRepair: func(added int) {},
		OnDHTRefresh: func() {},
		OnReconnect:  func(p peer.ID, success bool) {},
	}

	if callbacks.OnHighChurn == nil {
		t.Error("OnHighChurn should be set")
	}
	if callbacks.OnMeshRepair == nil {
		t.Error("OnMeshRepair should be set")
	}
	if callbacks.OnDHTRefresh == nil {
		t.Error("OnDHTRefresh should be set")
	}
	if callbacks.OnReconnect == nil {
		t.Error("OnReconnect should be set")
	}
}

func TestChurnHandler_repairMesh(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	degreeCtrl := NewDegreeController(h, manager)

	ch := NewChurnHandler(h, nil, degreeCtrl)
	defer ch.Stop()

	repairCalled := false
	ch.SetCallbacks(ChurnCallbacks{
		OnMeshRepair: func(added int) { repairCalled = true },
	})

	// Trigger repair
	ch.repairMesh()

	if !repairCalled {
		t.Error("OnMeshRepair callback should have been called")
	}
}

func TestChurnHandler_ContextCancellation(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ch := NewChurnHandler(h, nil, nil)
	ch.Start()

	// Cancel immediately
	ch.Stop()

	// Verify context is cancelled
	select {
	case <-ch.ctx.Done():
		// Good
	default:
		t.Error("context should be cancelled after Stop()")
	}
}
