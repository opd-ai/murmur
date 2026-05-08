// Package gossip – tests for the version consensus tracker.
//
//go:build test
// +build test

package gossip

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestVersionConsensusTracker_NoObservations(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	ready, ratio := tr.GetReadiness()
	if ready {
		t.Error("empty tracker should not be ready")
	}
	if ratio != 0 {
		t.Errorf("empty tracker ratio = %f, want 0", ratio)
	}
}

func TestVersionConsensusTracker_AllV1(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	for i := 0; i < 10; i++ {
		tr.Observe(peer.ID(string(rune('A'+i))), TopicVersionV1)
	}
	ready, ratio := tr.GetReadiness()
	if ready {
		t.Errorf("all-v1 tracker should not be ready, ratio=%f", ratio)
	}
	if ratio != 0 {
		t.Errorf("all-v1 ratio = %f, want 0", ratio)
	}
}

func TestVersionConsensusTracker_AllV2(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	for i := 0; i < 10; i++ {
		tr.Observe(peer.ID(string(rune('A'+i))), TopicVersionV2)
	}
	ready, ratio := tr.GetReadiness()
	if !ready {
		t.Errorf("all-v2 tracker should be ready, ratio=%f", ratio)
	}
	if ratio != 1.0 {
		t.Errorf("all-v2 ratio = %f, want 1.0", ratio)
	}
}

func TestVersionConsensusTracker_MixedBelowThreshold(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	// 80% on v2 (8 out of 10) — below ConsensusReadinessThreshold (90%)
	for i := 0; i < 8; i++ {
		tr.Observe(peer.ID(string(rune('A'+i))), TopicVersionV2)
	}
	for i := 8; i < 10; i++ {
		tr.Observe(peer.ID(string(rune('A'+i))), TopicVersionV1)
	}
	ready, ratio := tr.GetReadiness()
	if ready {
		t.Errorf("80%% v2 should not be ready (threshold 90%%), ratio=%f", ratio)
	}
	if ratio < 0.79 || ratio > 0.81 {
		t.Errorf("ratio = %f, want ~0.80", ratio)
	}
}

func TestVersionConsensusTracker_ObserveUpgradeOnly(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	p := peer.ID("test-peer")

	// First see on v2.
	tr.Observe(p, TopicVersionV2)

	// Now a v1 observation should NOT downgrade the record.
	tr.Observe(p, TopicVersionV1)

	_, ratio := tr.GetReadiness()
	if ratio != 1.0 {
		t.Errorf("downgrade should be ignored: ratio = %f, want 1.0", ratio)
	}
}

func TestVersionConsensusTracker_StaleEviction(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	tr.window = 10 * time.Millisecond // very short window for test

	p := peer.ID("stale-peer")
	tr.Observe(p, TopicVersionV2)

	// Wait for the observation to become stale.
	time.Sleep(20 * time.Millisecond)

	ready, ratio := tr.GetReadiness()
	if ready {
		t.Errorf("stale peer should be evicted; ready=%v ratio=%f", ready, ratio)
	}
	if ratio != 0 {
		t.Errorf("all peers stale; ratio = %f, want 0", ratio)
	}
}

func TestVersionConsensusTracker_ActivePeerCount(t *testing.T) {
	t.Parallel()

	tr := NewVersionConsensusTracker()
	for i := 0; i < 5; i++ {
		tr.Observe(peer.ID(string(rune('A'+i))), TopicVersionV1)
	}
	if got := tr.ActivePeerCount(); got != 5 {
		t.Errorf("ActivePeerCount() = %d, want 5", got)
	}
}
