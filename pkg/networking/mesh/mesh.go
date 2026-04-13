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
)

// Mesh configuration constants per DESIGN_DOCUMENT.md.
const (
	// MinPeers is the minimum number of peer connections to maintain.
	MinPeers = 6

	// MaxPeers is the maximum number of peer connections to maintain.
	MaxPeers = 12

	// HeartbeatInterval is the interval between heartbeat pings.
	HeartbeatInterval = 30 * time.Second

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
	h          host.Host
	peers      map[peer.ID]*PeerState
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	heartbeatC chan peer.ID
}

// NewManager creates a new connection manager.
func NewManager(h host.Host) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		h:          h,
		peers:      make(map[peer.ID]*PeerState),
		ctx:        ctx,
		cancel:     cancel,
		heartbeatC: make(chan peer.ID, 100),
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
}

// Stop stops the connection manager.
func (m *Manager) Stop() {
	m.cancel()
}

func (m *Manager) onConnect(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[p] = &PeerState{
		ID:       p,
		Priority: PriorityRandom, // Default priority
		LastSeen: time.Now(),
	}
}

func (m *Manager) onDisconnect(p peer.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.peers, p)
}

func (m *Manager) heartbeatLoop() {
	ticker := time.NewTicker(HeartbeatInterval)
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
		if now.Sub(state.LastSeen) > HeartbeatInterval*2 {
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

	m.mu.Lock()

	// Find lowest priority peer
	var lowestID peer.ID
	lowestPriority := PeerPriority(-1)

	for id, state := range m.peers {
		if lowestPriority < 0 || state.Priority > lowestPriority {
			lowestPriority = state.Priority
			lowestID = id
		}
	}

	if lowestID != "" {
		delete(m.peers, lowestID)
	}
	m.mu.Unlock()

	if lowestID != "" {
		_ = m.h.Network().ClosePeer(lowestID)
	}

	return lowestID
}
