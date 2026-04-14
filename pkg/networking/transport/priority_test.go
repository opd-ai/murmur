package transport

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnManager implements connmgr.ConnManager for testing.
type mockConnManager struct {
	tags map[peer.ID]map[string]int
}

func newMockConnManager() *mockConnManager {
	return &mockConnManager{
		tags: make(map[peer.ID]map[string]int),
	}
}

func (m *mockConnManager) TagPeer(p peer.ID, tag string, val int) {
	if m.tags[p] == nil {
		m.tags[p] = make(map[string]int)
	}
	m.tags[p][tag] = val
}

func (m *mockConnManager) UntagPeer(p peer.ID, tag string) {
	if m.tags[p] != nil {
		delete(m.tags[p], tag)
	}
}

func (m *mockConnManager) UpsertTag(p peer.ID, tag string, upsert func(int) int) {
	if m.tags[p] == nil {
		m.tags[p] = make(map[string]int)
	}
	m.tags[p][tag] = upsert(m.tags[p][tag])
}

func (m *mockConnManager) GetTagInfo(p peer.ID) *connmgr.TagInfo {
	return nil
}

func (m *mockConnManager) TrimOpenConns(ctx context.Context)     {}
func (m *mockConnManager) Notifee() network.Notifiee             { return nil }
func (m *mockConnManager) Protect(id peer.ID, tag string)        {}
func (m *mockConnManager) Unprotect(id peer.ID, tag string) bool { return false }
func (m *mockConnManager) IsProtected(id peer.ID, tag string) bool {
	return false
}
func (m *mockConnManager) Close() error { return nil }
func (m *mockConnManager) CheckLimit(l connmgr.GetConnLimiter) error {
	return nil
}

func TestPeerPriority_SetAndGetTier(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// Default tier should be Opportunistic.
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))

	// Set to Social.
	pp.SetTier(peerID, TierSocial)
	assert.Equal(t, TierSocial, pp.GetTier(peerID))

	// Set to Mesh.
	pp.SetTier(peerID, TierMesh)
	assert.Equal(t, TierMesh, pp.GetTier(peerID))

	// Set to DHT.
	pp.SetTier(peerID, TierDHT)
	assert.Equal(t, TierDHT, pp.GetTier(peerID))

	// Set to Opportunistic.
	pp.SetTier(peerID, TierOpportunistic)
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))
}

func TestPeerPriority_RemovePeer(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	pp.SetTier(peerID, TierSocial)
	assert.Equal(t, TierSocial, pp.GetTier(peerID))

	pp.RemovePeer(peerID)
	// After removal, should return default Opportunistic.
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))
}

func TestPeerPriority_PromoteToSocial(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// Start at Opportunistic.
	pp.SetTier(peerID, TierOpportunistic)
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))

	// Promote to Social.
	pp.PromoteToSocial(peerID)
	assert.Equal(t, TierSocial, pp.GetTier(peerID))
}

func TestPeerPriority_PromoteToMesh_NoSocialDemotion(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// Start at Social.
	pp.SetTier(peerID, TierSocial)

	// Try to promote to Mesh (should not demote from Social).
	pp.PromoteToMesh(peerID)
	assert.Equal(t, TierSocial, pp.GetTier(peerID), "Social should not be demoted to Mesh")
}

func TestPeerPriority_PromoteToMesh_FromDHT(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// Start at DHT.
	pp.SetTier(peerID, TierDHT)

	// Promote to Mesh.
	pp.PromoteToMesh(peerID)
	assert.Equal(t, TierMesh, pp.GetTier(peerID))
}

func TestPeerPriority_PromoteToDHT_NoHigherTierDemotion(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// Start at Social.
	pp.SetTier(peerID, TierSocial)
	pp.PromoteToDHT(peerID)
	assert.Equal(t, TierSocial, pp.GetTier(peerID), "Social should not be demoted to DHT")

	// Start at Mesh.
	pp.SetTier(peerID, TierMesh)
	pp.PromoteToDHT(peerID)
	assert.Equal(t, TierMesh, pp.GetTier(peerID), "Mesh should not be demoted to DHT")
}

func TestPeerPriority_DemoteToOpportunistic(t *testing.T) {
	pp := NewPeerPriority(nil)

	peerID := peer.ID("test-peer-1")

	// DHT can be demoted.
	pp.SetTier(peerID, TierDHT)
	pp.DemoteToOpportunistic(peerID)
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))

	// Mesh can be demoted.
	pp.SetTier(peerID, TierMesh)
	pp.DemoteToOpportunistic(peerID)
	assert.Equal(t, TierOpportunistic, pp.GetTier(peerID))

	// Social cannot be demoted via DemoteToOpportunistic.
	pp.SetTier(peerID, TierSocial)
	pp.DemoteToOpportunistic(peerID)
	assert.Equal(t, TierSocial, pp.GetTier(peerID), "Social should not be demoted via DemoteToOpportunistic")
}

func TestPeerPriority_PeersInTier(t *testing.T) {
	pp := NewPeerPriority(nil)

	peer1 := peer.ID("peer-1")
	peer2 := peer.ID("peer-2")
	peer3 := peer.ID("peer-3")
	peer4 := peer.ID("peer-4")

	pp.SetTier(peer1, TierSocial)
	pp.SetTier(peer2, TierMesh)
	pp.SetTier(peer3, TierMesh)
	pp.SetTier(peer4, TierDHT)

	socialPeers := pp.PeersInTier(TierSocial)
	assert.Len(t, socialPeers, 1)
	assert.Contains(t, socialPeers, peer1)

	meshPeers := pp.PeersInTier(TierMesh)
	assert.Len(t, meshPeers, 2)
	assert.Contains(t, meshPeers, peer2)
	assert.Contains(t, meshPeers, peer3)

	dhtPeers := pp.PeersInTier(TierDHT)
	assert.Len(t, dhtPeers, 1)
	assert.Contains(t, dhtPeers, peer4)

	opportunisticPeers := pp.PeersInTier(TierOpportunistic)
	assert.Len(t, opportunisticPeers, 0)
}

func TestPeerPriority_TierCounts(t *testing.T) {
	pp := NewPeerPriority(nil)

	pp.SetTier(peer.ID("peer-1"), TierSocial)
	pp.SetTier(peer.ID("peer-2"), TierSocial)
	pp.SetTier(peer.ID("peer-3"), TierMesh)
	pp.SetTier(peer.ID("peer-4"), TierMesh)
	pp.SetTier(peer.ID("peer-5"), TierMesh)
	pp.SetTier(peer.ID("peer-6"), TierDHT)
	pp.SetTier(peer.ID("peer-7"), TierOpportunistic)

	counts := pp.TierCounts()
	assert.Equal(t, 2, counts[TierSocial])
	assert.Equal(t, 3, counts[TierMesh])
	assert.Equal(t, 1, counts[TierDHT])
	assert.Equal(t, 1, counts[TierOpportunistic])
}

func TestPeerPriority_WithConnManager(t *testing.T) {
	cm := newMockConnManager()
	pp := NewPeerPriority(cm)

	peerID := peer.ID("test-peer-1")

	// Set to Social - should tag.
	pp.SetTier(peerID, TierSocial)
	require.NotNil(t, cm.tags[peerID])
	assert.Equal(t, TagValueSocial, cm.tags[peerID][TagKeySocial])

	// Change to Mesh - should remove Social tag and add Mesh tag.
	pp.SetTier(peerID, TierMesh)
	assert.Equal(t, 0, cm.tags[peerID][TagKeySocial], "Social tag should be removed")
	assert.Equal(t, TagValueMesh, cm.tags[peerID][TagKeyMesh])

	// Change to DHT.
	pp.SetTier(peerID, TierDHT)
	assert.Equal(t, 0, cm.tags[peerID][TagKeyMesh], "Mesh tag should be removed")
	assert.Equal(t, TagValueDHT, cm.tags[peerID][TagKeyDHT])

	// Change to Opportunistic.
	pp.SetTier(peerID, TierOpportunistic)
	assert.Equal(t, 0, cm.tags[peerID][TagKeyDHT], "DHT tag should be removed")
	assert.Equal(t, TagValueOpportunistic, cm.tags[peerID][TagKeyOpportunistic])
}

func TestConnectionTier_String(t *testing.T) {
	assert.Equal(t, "Social", TierSocial.String())
	assert.Equal(t, "Mesh", TierMesh.String())
	assert.Equal(t, "DHT", TierDHT.String())
	assert.Equal(t, "Opportunistic", TierOpportunistic.String())
	assert.Equal(t, "Unknown", ConnectionTier(99).String())
}

func TestConnectionTier_IsProtected(t *testing.T) {
	// Social and Mesh are protected.
	assert.True(t, TierSocial.IsProtected())
	assert.True(t, TierMesh.IsProtected())

	// DHT and Opportunistic are not protected.
	assert.False(t, TierDHT.IsProtected())
	assert.False(t, TierOpportunistic.IsProtected())
}

func TestTagValues_Priority(t *testing.T) {
	// Ensure tag values are in the correct priority order.
	assert.Greater(t, TagValueSocial, TagValueMesh, "Social should have higher priority than Mesh")
	assert.Greater(t, TagValueMesh, TagValueDHT, "Mesh should have higher priority than DHT")
	assert.Greater(t, TagValueDHT, TagValueOpportunistic, "DHT should have higher priority than Opportunistic")
}
