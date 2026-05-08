package mesh

import (
	"strconv"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestManager_ScorePruning(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	defer h.Close()

	m := NewManager(h, 0)

	// Add enough peers to exceed MinPeers (6) so pruning can occur
	// Add 4 neutral-score peers to fill up to 6 peers minimum
	for i := 0; i < 4; i++ {
		neutralPeer := peer.ID("neutral_peer_" + strconv.Itoa(i))
		m.mu.Lock()
		m.peers[neutralPeer] = &PeerState{
			ID:       neutralPeer,
			Priority: PriorityRandom,
			LastSeen: time.Now(),
		}
		m.mu.Unlock()
	}

	// Add low-score peer
	lowScorePeer := peer.ID("low_score_peer")
	m.mu.Lock()
	m.peers[lowScorePeer] = &PeerState{
		ID:       lowScorePeer,
		Priority: PriorityRandom,
		LastSeen: time.Now(),
	}
	m.mu.Unlock()

	// Add high-score peer
	highScorePeer := peer.ID("high_score_peer")
	m.mu.Lock()
	m.peers[highScorePeer] = &PeerState{
		ID:       highScorePeer,
		Priority: PriorityGossip,
		LastSeen: time.Now(),
	}
	m.mu.Unlock()

	// Add Identity-priority peer with low score (should not be pruned)
	identityPeer := peer.ID("identity_peer")
	m.mu.Lock()
	m.peers[identityPeer] = &PeerState{
		ID:       identityPeer,
		Priority: PriorityIdentity,
		LastSeen: time.Now(),
	}
	m.mu.Unlock()

	// Verify we have 7 peers total
	require.Equal(t, 7, m.PeerCount())

	// Configure score function
	scoreFunc := func(p peer.ID) float64 {
		switch p {
		case lowScorePeer:
			return -100.0 // Below -50 threshold
		case highScorePeer:
			return 10.0 // Above threshold
		case identityPeer:
			return -100.0 // Below threshold but identity priority
		default:
			return 0.0 // Neutral for filler peers
		}
	}
	m.SetScoreFunc(scoreFunc)

	// Trigger pruning LowScoreStrikeLimit times to hit the consistently-low threshold.
	for i := 0; i < LowScoreStrikeLimit; i++ {
		m.pruneByScore()
	}

	// Verify results
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Low-score peer should be pruned
	_, exists := m.peers[lowScorePeer]
	require.False(t, exists, "low-score peer should be pruned")

	// High-score peer should remain
	_, exists = m.peers[highScorePeer]
	require.True(t, exists, "high-score peer should remain")

	// Identity peer should remain despite low score
	_, exists = m.peers[identityPeer]
	require.True(t, exists, "Identity-priority peer should not be pruned")
}

func TestManager_SetScoreFunc(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	defer h.Close()

	m := NewManager(h, 0)

	// Initially no score function
	require.Nil(t, m.scoreFunc)

	// Set score function
	scoreFunc := func(p peer.ID) float64 { return 0.0 }
	m.SetScoreFunc(scoreFunc)

	// Verify it's set
	m.mu.RLock()
	defer m.mu.RUnlock()
	require.NotNil(t, m.scoreFunc)
}
