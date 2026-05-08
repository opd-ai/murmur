// Package gossip – breaking change consensus tracking.
// Per PLAN.md: "Breaking change consensus mechanism".
//
// During a rolling protocol upgrade, nodes need a reliable signal for when
// it is safe to remove support for the old protocol version ("break" v1).
// The VersionConsensusTracker observes which topic generations peers are
// actively publishing on and maintains a sliding-window tally.
//
// When GetReadiness() reports that ≥ ConsensusReadinessThreshold of observed
// peers have been seen on v2 topics, the operator can safely move
// MinSupportedVersion to 2 and stop handling v1.
//
// This is intentionally a passive, data-collection mechanism.  The actual
// decision to increment MinSupportedVersion is a human/operator action
// documented in a new ADR.
package gossip

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ConsensusReadinessThreshold is the fraction of active peers that must be
// observed on v2 topics before the network is considered ready to drop v1.
// A value of 0.90 means 90% of peers must have published on a v2 topic.
const ConsensusReadinessThreshold = 0.90

// consensusPeerWindow is the sliding window for peer version observations.
// Peers not observed within this window are considered stale and excluded.
const consensusPeerWindow = 10 * time.Minute

// peerVersionRecord records which protocol generation a peer was last seen on.
type peerVersionRecord struct {
	highestVersion TopicVersion
	lastSeen       time.Time
}

// VersionConsensusTracker counts peer version observations to determine when
// it is safe to drop support for the previous protocol version.
//
// Usage:
//
//	tracker := NewVersionConsensusTracker()
//	// call tracker.Observe(peerID, TopicVersionV2) for each v2 message received
//	ready, ratio := tracker.GetReadiness()
type VersionConsensusTracker struct {
	mu      sync.RWMutex
	peers   map[peer.ID]*peerVersionRecord
	window  time.Duration
}

// NewVersionConsensusTracker creates a tracker with the default observation window.
func NewVersionConsensusTracker() *VersionConsensusTracker {
	return &VersionConsensusTracker{
		peers:  make(map[peer.ID]*peerVersionRecord),
		window: consensusPeerWindow,
	}
}

// Observe records that p was seen publishing on version v.
// Only upgrades are recorded: if p was previously seen on v2, a subsequent
// v1 observation does not downgrade the record (peers may dual-publish).
func (t *VersionConsensusTracker) Observe(p peer.ID, v TopicVersion) {
	t.mu.Lock()
	defer t.mu.Unlock()

	rec, ok := t.peers[p]
	if !ok {
		t.peers[p] = &peerVersionRecord{highestVersion: v, lastSeen: time.Now()}
		return
	}
	if v > rec.highestVersion {
		rec.highestVersion = v
	}
	rec.lastSeen = time.Now()
}

// GetReadiness returns whether ConsensusReadinessThreshold of active peers
// have been observed on TopicVersionV2, plus the current ratio.
//
// Only peers seen within the consensus window contribute to the tally.
func (t *VersionConsensusTracker) GetReadiness() (ready bool, v2Ratio float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-t.window)
	var total, v2Count int

	for id, rec := range t.peers {
		if rec.lastSeen.Before(cutoff) {
			// Evict stale entry.
			delete(t.peers, id)
			continue
		}
		total++
		if rec.highestVersion >= TopicVersionV2 {
			v2Count++
		}
	}

	if total == 0 {
		return false, 0
	}

	v2Ratio = float64(v2Count) / float64(total)
	return v2Ratio >= ConsensusReadinessThreshold, v2Ratio
}

// ActivePeerCount returns the number of peers seen within the consensus window.
func (t *VersionConsensusTracker) ActivePeerCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-t.window)
	count := 0
	for _, rec := range t.peers {
		if !rec.lastSeen.Before(cutoff) {
			count++
		}
	}
	return count
}
