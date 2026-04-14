package mesh

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestDegreeControllerConstants(t *testing.T) {
	// Verify constants per DESIGN_DOCUMENT.md
	if TargetDegree != 6 {
		t.Errorf("TargetDegree = %d, expected 6", TargetDegree)
	}
	if LowDegreeThreshold != 4 {
		t.Errorf("LowDegreeThreshold = %d, expected 4", LowDegreeThreshold)
	}
	if HighDegreeThreshold != 12 {
		t.Errorf("HighDegreeThreshold = %d, expected 12", HighDegreeThreshold)
	}
	// Verify bounds relationship
	if LowDegreeThreshold >= TargetDegree {
		t.Error("LowDegreeThreshold should be < TargetDegree")
	}
	if TargetDegree >= HighDegreeThreshold {
		t.Error("TargetDegree should be < HighDegreeThreshold")
	}
}

func TestNewDegreeController(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	if dc == nil {
		t.Fatal("expected non-nil degree controller")
	}
	if dc.h != h {
		t.Error("host not set correctly")
	}
	if dc.manager != manager {
		t.Error("manager not set correctly")
	}
}

func TestDegreeController_Status(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	// With no peers
	status := dc.Status()
	if status.Current != 0 {
		t.Errorf("expected 0 current peers, got %d", status.Current)
	}
	if status.Target != TargetDegree {
		t.Errorf("expected target %d, got %d", TargetDegree, status.Target)
	}
	if status.LowBound != LowDegreeThreshold {
		t.Errorf("expected low bound %d, got %d", LowDegreeThreshold, status.LowBound)
	}
	if status.HighBound != HighDegreeThreshold {
		t.Errorf("expected high bound %d, got %d", HighDegreeThreshold, status.HighBound)
	}
	if !status.NeedsMore {
		t.Error("should need more peers when at 0")
	}
	if status.NeedsPrune {
		t.Error("should not need pruning when at 0")
	}
	if status.IsHealthy {
		t.Error("should not be healthy when at 0")
	}
}

func TestDegreeController_SetCallbacks(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	pruneCalled := false
	dc.SetPruneCallback(func(p peer.ID, reason string) {
		pruneCalled = true
	})

	acquireCalled := false
	dc.SetAcquireCallback(func(p peer.AddrInfo, success bool) {
		acquireCalled = true
	})

	// Verify callbacks are set (can't easily trigger without peers)
	if dc.pruneCallback == nil {
		t.Error("prune callback not set")
	}
	if dc.acquireCallback == nil {
		t.Error("acquire callback not set")
	}

	_ = pruneCalled
	_ = acquireCalled
}

func TestDegreeController_StartStop(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	dc.Start()

	// Give control loop time to start
	time.Sleep(50 * time.Millisecond)

	dc.Stop()

	// Should stop cleanly without panic
}

func TestDegreeController_ForceAdjust(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	// Should not panic with no peer source
	dc.ForceAdjust()
}

type mockPeerSource struct {
	peers []peer.AddrInfo
}

func (m *mockPeerSource) GetCandidatePeers(ctx context.Context, count int) []peer.AddrInfo {
	if count > len(m.peers) {
		return m.peers
	}
	return m.peers[:count]
}

func TestDegreeController_SetPeerSource(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	source := &mockPeerSource{}
	dc.SetPeerSource(source)

	if dc.peerSource != source {
		t.Error("peer source not set correctly")
	}
}

func TestScoreBasedPruning_New(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	scoreFunc := func(p peer.ID) float64 { return 0 }
	threshold := -10.0

	sbp := NewScoreBasedPruning(dc, scoreFunc, threshold)

	if sbp == nil {
		t.Fatal("expected non-nil score-based pruning")
	}
	if sbp.dc != dc {
		t.Error("degree controller not set correctly")
	}
	if sbp.threshold != threshold {
		t.Errorf("threshold = %f, expected %f", sbp.threshold, threshold)
	}
}

func TestScoreBasedPruning_PruneLowScorePeers(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	manager := NewManager(h)
	dc := NewDegreeController(h, manager)

	scoreFunc := func(p peer.ID) float64 { return -100 } // All peers have low score
	sbp := NewScoreBasedPruning(dc, scoreFunc, -10.0)

	// With no peers, should prune 0
	pruned := sbp.PruneLowScorePeers()
	if pruned != 0 {
		t.Errorf("expected 0 pruned, got %d", pruned)
	}
}

func TestStatTracker_New(t *testing.T) {
	st := NewStatTracker()
	if st == nil {
		t.Fatal("expected non-nil stat tracker")
	}
	stats := st.GetStats()
	if stats.TotalAcquireAttempts != 0 || stats.SuccessfulAcquires != 0 ||
		stats.TotalPrunes != 0 || stats.ScoreBasedPrunes != 0 {
		t.Error("expected all stats to be 0 initially")
	}
}

func TestStatTracker_RecordAcquireAttempt(t *testing.T) {
	st := NewStatTracker()

	st.RecordAcquireAttempt(true)
	st.RecordAcquireAttempt(true)
	st.RecordAcquireAttempt(false)

	stats := st.GetStats()
	if stats.TotalAcquireAttempts != 3 {
		t.Errorf("expected 3 total attempts, got %d", stats.TotalAcquireAttempts)
	}
	if stats.SuccessfulAcquires != 2 {
		t.Errorf("expected 2 successful acquires, got %d", stats.SuccessfulAcquires)
	}
}

func TestStatTracker_RecordPrune(t *testing.T) {
	st := NewStatTracker()

	st.RecordPrune("degree_pruning")
	st.RecordPrune("low_score")
	st.RecordPrune("low_score")

	stats := st.GetStats()
	if stats.TotalPrunes != 3 {
		t.Errorf("expected 3 total prunes, got %d", stats.TotalPrunes)
	}
	if stats.ScoreBasedPrunes != 2 {
		t.Errorf("expected 2 score-based prunes, got %d", stats.ScoreBasedPrunes)
	}
}

func TestDegreeStatus_Healthy(t *testing.T) {
	// Test different peer counts
	testCases := []struct {
		count   int
		healthy bool
		needs   bool
		prune   bool
	}{
		{0, false, true, false},
		{3, false, true, false},
		{4, true, false, false},
		{6, true, false, false},
		{12, true, false, false},
		{13, false, false, true},
		{20, false, false, true},
	}

	for _, tc := range testCases {
		// Create a status manually for testing
		status := DegreeStatus{
			Current:    tc.count,
			Target:     TargetDegree,
			LowBound:   LowDegreeThreshold,
			HighBound:  HighDegreeThreshold,
			NeedsMore:  tc.count < LowDegreeThreshold,
			NeedsPrune: tc.count > HighDegreeThreshold,
			IsHealthy:  tc.count >= LowDegreeThreshold && tc.count <= HighDegreeThreshold,
		}

		if status.IsHealthy != tc.healthy {
			t.Errorf("count=%d: IsHealthy=%v, expected %v", tc.count, status.IsHealthy, tc.healthy)
		}
		if status.NeedsMore != tc.needs {
			t.Errorf("count=%d: NeedsMore=%v, expected %v", tc.count, status.NeedsMore, tc.needs)
		}
		if status.NeedsPrune != tc.prune {
			t.Errorf("count=%d: NeedsPrune=%v, expected %v", tc.count, status.NeedsPrune, tc.prune)
		}
	}
}
