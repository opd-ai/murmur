// Package mesh provides peer scoring, mesh health monitoring, and connection management.
// Per DESIGN_DOCUMENT.md Part II §6, nodes maintain 6-12 peer connections
// with priority tiers and heartbeat monitoring.
package mesh

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Mesh configuration constants per DESIGN_DOCUMENT.md.
const (
	// MinPeers is the minimum number of peer connections to maintain.
	MinPeers = 6

	// MaxPeers is the maximum number of peer connections to maintain.
	MaxPeers = 12

	// MissedHeartbeatsThreshold is the number of missed heartbeats before disconnect.
	MissedHeartbeatsThreshold = 3

	// ReconnectBackoff is the initial backoff duration for reconnection attempts.
	ReconnectBackoff = 5 * time.Second

	// MaxReconnectBackoff is the maximum backoff duration.
	MaxReconnectBackoff = 5 * time.Minute
)

// PeerPriority defines priority tiers for connection management.
// Per DESIGN_DOCUMENT.md Part II §6.
type PeerPriority int

const (
	// PriorityIdentity is the highest priority for identity connections.
	PriorityIdentity PeerPriority = iota
	// PriorityGossip is for useful gossip peers.
	PriorityGossip
	// PriorityRandom is for random discovery peers.
	PriorityRandom
)

// PeerState tracks the health state of a connected peer.
type PeerState struct {
	ID              peer.ID
	Priority        PeerPriority
	LastSeen        time.Time
	MissedHeartbeat int
	Latency         time.Duration
}

// Manager manages peer connections and mesh health.
type Manager struct {
	h                 host.Host
	peers             map[peer.ID]*PeerState
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	heartbeatC        chan peer.ID
	heartbeatInterval time.Duration
	scoreFunc         PeerScoreFunc           // Optional: for score-based pruning
	diversityMgr      *RegionDiversityManager // Optional: for eclipse attack resistance
}

// NewManager creates a new connection manager with the given heartbeat interval.
// If interval is 0, defaults to 30 seconds.
func NewManager(h host.Host, heartbeatInterval time.Duration) *Manager {
	if heartbeatInterval == 0 {
		heartbeatInterval = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		h:                 h,
		peers:             make(map[peer.ID]*PeerState),
		ctx:               ctx,
		cancel:            cancel,
		heartbeatC:        make(chan peer.ID, 100),
		heartbeatInterval: heartbeatInterval,
		diversityMgr:      NewRegionDiversityManager(),
	}

	// Register connection notifee
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			m.onConnect(c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			m.onDisconnect(c.RemotePeer())
		},
	})

	return m
}

// Start begins the connection management background tasks.
func (m *Manager) Start() {
	go m.heartbeatLoop()
	go m.processHeartbeats()
	if m.scoreFunc != nil {
		go m.scorePruneLoop()
	}
}

// Stop stops the connection manager.
func (m *Manager) Stop() {
	m.cancel()
}

// SetScoreFunc configures the peer scoring function for score-based pruning.
// Per AUDIT.md remediation, scores < -50 trigger peer disconnection.
func (m *Manager) SetScoreFunc(f PeerScoreFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scoreFunc = f
}

// scorePruneLoop periodically prunes peers with low GossipSub scores.
// Runs every 5 minutes per AUDIT.md specification.
func (m *Manager) scorePruneLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.pruneByScore()
		}
	}
}

// pruneByScore disconnects peers with scores below -50.
// Respects priority tiers: never prunes Identity-priority peers.
func (m *Manager) pruneByScore() {
	m.mu.RLock()
	scoreFunc := m.scoreFunc
	m.mu.RUnlock()

	if scoreFunc == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for peerID, state := range m.peers {
		// Never prune Identity-priority peers (direct connections)
		if state.Priority == PriorityIdentity {
			continue
		}

		// Check score threshold
		score := scoreFunc(peerID)
		if score < -50.0 {
			// Don't prune if we'd go below minimum
			if len(m.peers) <= MinPeers {
				break
			}

			// Close connection
			_ = m.h.Network().ClosePeer(peerID)
			delete(m.peers, peerID)
		}
	}
}

func (m *Manager) onConnect(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[p] = &PeerState{
		ID:       p,
		Priority: PriorityRandom, // Default priority
		LastSeen: time.Now(),
	}

	// Track region diversity for eclipse resistance
	if m.diversityMgr != nil {
		addrs := m.h.Peerstore().Addrs(p)
		m.diversityMgr.AddPeer(p, addrs)
	}
}

func (m *Manager) onDisconnect(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.peers, p)

	// Remove from region diversity tracking
	if m.diversityMgr != nil {
		m.diversityMgr.RemovePeer(p)
	}
}

func (m *Manager) heartbeatLoop() {
	ticker := time.NewTicker(m.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkHeartbeats()
		}
	}
}

func (m *Manager) checkHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toDisconnect []peer.ID

	for id, state := range m.peers {
		// Check if heartbeat is overdue
		if now.Sub(state.LastSeen) > m.heartbeatInterval*2 {
			state.MissedHeartbeat++
			if state.MissedHeartbeat >= MissedHeartbeatsThreshold {
				toDisconnect = append(toDisconnect, id)
			}
		}
	}

	// Disconnect unresponsive peers
	for _, id := range toDisconnect {
		delete(m.peers, id)
		go func(p peer.ID) {
			_ = m.h.Network().ClosePeer(p)
		}(id)
	}
}

func (m *Manager) processHeartbeats() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case p := <-m.heartbeatC:
			m.recordHeartbeat(p)
		}
	}
}

// RecordHeartbeat records a heartbeat from a peer.
func (m *Manager) RecordHeartbeat(p peer.ID) {
	select {
	case m.heartbeatC <- p:
	default:
		// Channel full, drop heartbeat
	}
}

func (m *Manager) recordHeartbeat(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, ok := m.peers[p]; ok {
		state.LastSeen = time.Now()
		state.MissedHeartbeat = 0
	}
}

// SetPriority sets the priority of a peer.
func (m *Manager) SetPriority(p peer.ID, priority PeerPriority) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, ok := m.peers[p]; ok {
		state.Priority = priority
	}
}

// PeerCount returns the current number of connected peers.
func (m *Manager) PeerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}

// Peers returns a snapshot of all connected peers.
func (m *Manager) Peers() []PeerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PeerState, 0, len(m.peers))
	for _, state := range m.peers {
		result = append(result, *state)
	}
	return result
}

// NeedsMorePeers returns true if we are below the minimum peer count.
func (m *Manager) NeedsMorePeers() bool {
	return m.PeerCount() < MinPeers
}

// HasTooManyPeers returns true if we are above the maximum peer count.
func (m *Manager) HasTooManyPeers() bool {
	return m.PeerCount() > MaxPeers
}

// PruneLowestPriority disconnects the lowest priority peer if we have too many.
// Returns the pruned peer ID, or empty if no pruning was needed.
func (m *Manager) PruneLowestPriority() peer.ID {
	if !m.HasTooManyPeers() {
		return ""
	}

	// First check if we need to prune for diversity (eclipse resistance)
	if m.diversityMgr != nil {
		toDrop := m.diversityMgr.GetPeersToDropForDiversity()
		if len(toDrop) > 0 {
			// Prune the first overloaded region peer
			lowestID := toDrop[0]
			m.mu.Lock()
			delete(m.peers, lowestID)
			m.mu.Unlock()
			_ = m.h.Network().ClosePeer(lowestID)
			return lowestID
		}
	}

	lowestID := m.removeLowestPriorityPeer()
	if lowestID != "" {
		_ = m.h.Network().ClosePeer(lowestID)
	}

	return lowestID
}

// removeLowestPriorityPeer finds and removes the lowest priority peer from the map.
// Returns the removed peer ID, or empty if none found.
func (m *Manager) removeLowestPriorityPeer() peer.ID {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowestID := m.findLowestPriorityPeerLocked()
	if lowestID != "" {
		delete(m.peers, lowestID)
	}
	return lowestID
}

// findLowestPriorityPeerLocked finds the peer with the lowest priority.
// Must be called with m.mu held.
func (m *Manager) findLowestPriorityPeerLocked() peer.ID {
	var lowestID peer.ID
	lowestPriority := PeerPriority(-1)

	for id, state := range m.peers {
		if lowestPriority < 0 || state.Priority > lowestPriority {
			lowestPriority = state.Priority
			lowestID = id
		}
	}
	return lowestID
}

// DiversityStatus returns the current region diversity status.
// Returns nil if diversity manager is not enabled.
func (m *Manager) DiversityStatus() *DiversityStatus {
	if m.diversityMgr == nil {
		return nil
	}
	status := m.diversityMgr.Status()
	return &status
}

// ShouldAcceptPeerFromAddrs returns whether we should accept a peer
// based on region diversity constraints. Returns true if diversity
// manager is disabled or if the peer should be accepted.
func (m *Manager) ShouldAcceptPeerFromAddrs(addrs []ma.Multiaddr) bool {
	if m.diversityMgr == nil {
		return true
	}
	return m.diversityMgr.ShouldAcceptPeer(addrs)
}
