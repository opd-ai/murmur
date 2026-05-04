// Package transport provides libp2p host construction and peer connection management.
// This file provides backward-compatible type aliases for the priority system.
// The priority implementation has been moved to pkg/networking/priority/ to reduce coupling.

package transport

import (
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/opd-ai/murmur/pkg/networking/priority"
)

// Type aliases for backward compatibility with existing code.
type (
	ConnectionTier = priority.ConnectionTier
	PeerPriority   = priority.Manager
)

// Constant aliases for backward compatibility.
const (
	TierSocial        = priority.TierSocial
	TierMesh          = priority.TierMesh
	TierDHT           = priority.TierDHT
	TierOpportunistic = priority.TierOpportunistic

	TagValueSocial        = priority.TagValueSocial
	TagValueMesh          = priority.TagValueMesh
	TagValueDHT           = priority.TagValueDHT
	TagValueOpportunistic = priority.TagValueOpportunistic

	TagKeySocial        = priority.TagKeySocial
	TagKeyMesh          = priority.TagKeyMesh
	TagKeyDHT           = priority.TagKeyDHT
	TagKeyOpportunistic = priority.TagKeyOpportunistic
)

// NewPeerPriority creates a new PeerPriority manager.
// If connMgr is nil, priority tracking is local-only (no tagging).
// Deprecated: Use priority.NewManager directly.
func NewPeerPriority(connMgr connmgr.ConnManager) *PeerPriority {
	return priority.NewManager(connMgr)
}

// PriorityManager returns a new priority manager for the given connection manager.
// This is a convenience wrapper for priority.NewManager.
func PriorityManager(connMgr connmgr.ConnManager) *priority.Manager {
	return priority.NewManager(connMgr)
}

// TierFromString parses a tier name string.
// Returns TierOpportunistic if the string doesn't match a known tier.
func TierFromString(s string) ConnectionTier {
	switch s {
	case "Social":
		return TierSocial
	case "Mesh":
		return TierMesh
	case "DHT":
		return TierDHT
	case "Opportunistic":
		return TierOpportunistic
	default:
		return TierOpportunistic
	}
}

// SetPeerTier is a convenience function to set a peer's priority tier.
func SetPeerTier(mgr *priority.Manager, peerID peer.ID, tier ConnectionTier) {
	mgr.SetTier(peerID, tier)
}
