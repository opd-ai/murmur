// Package transport provides libp2p host construction and peer connection management.
// This file implements the four-tier connection priority system per NETWORK_ARCHITECTURE.md §7.

package transport

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ConnectionTier represents the priority tier for a peer connection.
// Per NETWORK_ARCHITECTURE.md §7: Topology Management, connections are classified into four tiers.
type ConnectionTier int

const (
	// TierSocial is the highest priority tier for declared social connections.
	// Per NETWORK_ARCHITECTURE.md §7: "Peers who are the user's declared social connections...
	// These are the highest priority and are never dropped unless the social connection itself is revoked."
	TierSocial ConnectionTier = iota + 1

	// TierMesh is for GossipSub mesh partners on subscribed topics.
	// Per NETWORK_ARCHITECTURE.md §7: "Peers who are mesh partners in the user's subscribed GossipSub topics.
	// These connections are essential for gossip propagation."
	TierMesh

	// TierDHT is for Kademlia DHT routing table neighbors.
	// Per NETWORK_ARCHITECTURE.md §7: "Peers who are in the user's Kademlia routing table,
	// particularly those in sparse buckets."
	TierDHT

	// TierOpportunistic is for peers discovered but not serving any specific role.
	// Per NETWORK_ARCHITECTURE.md §7: "Peers discovered through peer exchange or DHT queries
	// that are not currently serving any specific role."
	TierOpportunistic
)

// Connection priority tag values for libp2p connection manager.
// Higher values mean higher priority (less likely to be pruned).
// Per NETWORK_ARCHITECTURE.md §7: "When the connection count approaches the maximum,
// the connection manager drops Tier 4 connections first, then Tier 3 if necessary."
const (
	// TagValueSocial is the tag value for Tier 1 (Social) connections.
	// Very high value ensures these connections are never pruned by the connection manager.
	TagValueSocial = 1000

	// TagValueMesh is the tag value for Tier 2 (Mesh) connections.
	// High value protects mesh partners from pruning.
	TagValueMesh = 500

	// TagValueDHT is the tag value for Tier 3 (DHT) connections.
	// Moderate value, pruned after Tier 4 if necessary.
	TagValueDHT = 100

	// TagValueOpportunistic is the tag value for Tier 4 (Opportunistic) connections.
	// Low value, these connections are pruned first.
	TagValueOpportunistic = 10
)

// Tag keys for identifying connection roles.
const (
	// TagKeySocial identifies a social connection.
	TagKeySocial = "murmur-social"

	// TagKeyMesh identifies a GossipSub mesh partner.
	TagKeyMesh = "murmur-mesh"

	// TagKeyDHT identifies a DHT routing table neighbor.
	TagKeyDHT = "murmur-dht"

	// TagKeyOpportunistic identifies an opportunistic connection.
	TagKeyOpportunistic = "murmur-opportunistic"
)

// PeerPriority manages the priority tier classification for peer connections.
// It tracks which tier each peer belongs to and tags them appropriately
// in the libp2p connection manager.
type PeerPriority struct {
	mu      sync.RWMutex
	tiers   map[peer.ID]ConnectionTier
	connMgr connmgr.ConnManager
}

// NewPeerPriority creates a new PeerPriority manager.
// If connMgr is nil, priority tracking is local-only (no tagging).
func NewPeerPriority(connMgr connmgr.ConnManager) *PeerPriority {
	return &PeerPriority{
		tiers:   make(map[peer.ID]ConnectionTier),
		connMgr: connMgr,
	}
}

// SetTier assigns a priority tier to a peer and updates connection manager tags.
// If the peer already has a tier, this replaces it.
func (pp *PeerPriority) SetTier(peerID peer.ID, tier ConnectionTier) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	oldTier, hadOldTier := pp.tiers[peerID]
	pp.tiers[peerID] = tier

	if pp.connMgr == nil {
		return
	}

	// Remove old tier tag if different.
	if hadOldTier && oldTier != tier {
		pp.untagPeer(peerID, oldTier)
	}

	// Apply new tier tag.
	pp.tagPeer(peerID, tier)
}

// GetTier returns the priority tier for a peer.
// Returns TierOpportunistic if the peer has no assigned tier.
func (pp *PeerPriority) GetTier(peerID peer.ID) ConnectionTier {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	tier, ok := pp.tiers[peerID]
	if !ok {
		return TierOpportunistic
	}
	return tier
}

// RemovePeer removes a peer from priority tracking and clears its tags.
func (pp *PeerPriority) RemovePeer(peerID peer.ID) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	tier, ok := pp.tiers[peerID]
	if !ok {
		return
	}

	delete(pp.tiers, peerID)

	if pp.connMgr != nil {
		pp.untagPeer(peerID, tier)
	}
}

// PromoteToSocial elevates a peer to TierSocial.
// Per NETWORK_ARCHITECTURE.md §7: Social connections are never dropped.
func (pp *PeerPriority) PromoteToSocial(peerID peer.ID) {
	pp.SetTier(peerID, TierSocial)
}

// PromoteToMesh elevates a peer to TierMesh.
// Per NETWORK_ARCHITECTURE.md §7: Mesh partners are essential for gossip propagation.
func (pp *PeerPriority) PromoteToMesh(peerID peer.ID) {
	currentTier := pp.GetTier(peerID)
	// Don't demote from Social to Mesh.
	if currentTier == TierSocial {
		return
	}
	pp.SetTier(peerID, TierMesh)
}

// PromoteToDHT elevates a peer to TierDHT.
// Per NETWORK_ARCHITECTURE.md §7: DHT neighbors maintain routing structure.
func (pp *PeerPriority) PromoteToDHT(peerID peer.ID) {
	currentTier := pp.GetTier(peerID)
	// Don't demote from Social or Mesh to DHT.
	if currentTier == TierSocial || currentTier == TierMesh {
		return
	}
	pp.SetTier(peerID, TierDHT)
}

// DemoteToOpportunistic demotes a peer to TierOpportunistic.
// This should only be called when the peer loses its role (e.g., mesh partner removed).
// Social connections are never demoted by this method.
func (pp *PeerPriority) DemoteToOpportunistic(peerID peer.ID) {
	currentTier := pp.GetTier(peerID)
	// Never demote Social connections via this method.
	if currentTier == TierSocial {
		return
	}
	pp.SetTier(peerID, TierOpportunistic)
}

// PeersInTier returns all peers assigned to the given tier.
func (pp *PeerPriority) PeersInTier(tier ConnectionTier) []peer.ID {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	var peers []peer.ID
	for peerID, t := range pp.tiers {
		if t == tier {
			peers = append(peers, peerID)
		}
	}
	return peers
}

// TierCounts returns the count of peers in each tier.
func (pp *PeerPriority) TierCounts() map[ConnectionTier]int {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	counts := make(map[ConnectionTier]int)
	for _, tier := range pp.tiers {
		counts[tier]++
	}
	return counts
}

// tagPeer applies the appropriate tag for a tier.
func (pp *PeerPriority) tagPeer(peerID peer.ID, tier ConnectionTier) {
	key, value := tierToTag(tier)
	pp.connMgr.TagPeer(peerID, key, value)
}

// untagPeer removes the tag for a tier.
func (pp *PeerPriority) untagPeer(peerID peer.ID, tier ConnectionTier) {
	key, _ := tierToTag(tier)
	pp.connMgr.UntagPeer(peerID, key)
}

// tierToTag converts a tier to its tag key and value.
func tierToTag(tier ConnectionTier) (string, int) {
	switch tier {
	case TierSocial:
		return TagKeySocial, TagValueSocial
	case TierMesh:
		return TagKeyMesh, TagValueMesh
	case TierDHT:
		return TagKeyDHT, TagValueDHT
	default:
		return TagKeyOpportunistic, TagValueOpportunistic
	}
}

// TierString returns a human-readable name for a tier.
func (t ConnectionTier) String() string {
	switch t {
	case TierSocial:
		return "Social"
	case TierMesh:
		return "Mesh"
	case TierDHT:
		return "DHT"
	case TierOpportunistic:
		return "Opportunistic"
	default:
		return "Unknown"
	}
}

// IsProtected returns true if the tier is protected from automatic pruning.
// Per NETWORK_ARCHITECTURE.md §7: Tier 1 and Tier 2 connections are never dropped
// by the connection manager (only by explicit user action or protocol-level events).
func (t ConnectionTier) IsProtected() bool {
	return t == TierSocial || t == TierMesh
}
