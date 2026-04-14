// Package mesh provides peer scoring, mesh health monitoring, and connection management.
// This file implements network partition detection and healing protocol.
// Per DESIGN_DOCUMENT.md Part II §6, the system must detect partitions and heal gracefully.
package mesh

import (
	"context"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Partition detection and healing constants.
const (
	// PartitionThreshold is the minimum peer count before partition is declared.
	PartitionThreshold = 2

	// GracefulDegradationThreshold triggers reduced functionality mode.
	GracefulDegradationThreshold = 4

	// HealingInterval is time between healing attempts during partition.
	HealingInterval = 30 * time.Second

	// BootstrapRefreshTimeout is timeout for bootstrap peer connections.
	BootstrapRefreshTimeout = 15 * time.Second

	// HistoricalPeerCacheSize is max historical peers to remember.
	HistoricalPeerCacheSize = 100

	// ReconnectBatchSize is peers to try reconnecting per healing cycle.
	ReconnectBatchSize = 10

	// PartitionConfirmationDelay avoids false positives from brief dips.
	PartitionConfirmationDelay = 10 * time.Second
)

// PartitionState represents the current network partition state.
type PartitionState int

const (
	// StateNormal indicates healthy connectivity.
	StateNormal PartitionState = iota
	// StateDegraded indicates reduced but functional connectivity.
	StateDegraded
	// StatePartitioned indicates severe connectivity loss.
	StatePartitioned
)

// String returns string representation of partition state.
func (s PartitionState) String() string {
	switch s {
	case StateNormal:
		return "normal"
	case StateDegraded:
		return "degraded"
	case StatePartitioned:
		return "partitioned"
	default:
		return "unknown"
	}
}

// PartitionManager handles partition detection and healing.
type PartitionManager struct {
	h                  host.Host
	dht                *dht.IpfsDHT
	bootstrapPeers     []peer.AddrInfo
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	state              PartitionState
	stateChangeTime    time.Time
	pendingPartition   bool
	pendingPartitionAt time.Time
	historicalPeers    []peer.AddrInfo
	healingActive      bool
	callbacks          PartitionCallbacks
}

// PartitionCallbacks are callbacks for partition events.
type PartitionCallbacks struct {
	OnStateChange   func(old, new PartitionState)
	OnHealingStart  func()
	OnHealingEnd    func(success bool)
	OnPeerRecovered func(p peer.ID)
}

// PartitionStatus contains current partition status information.
type PartitionStatus struct {
	State            PartitionState
	PeerCount        int
	StateDuration    time.Duration
	HealingActive    bool
	HistoricalPeers  int
	LastStateChange  time.Time
	PendingPartition bool
}

// NewPartitionManager creates a new partition manager.
func NewPartitionManager(
	h host.Host,
	dht *dht.IpfsDHT,
	bootstrapPeers []peer.AddrInfo,
) *PartitionManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PartitionManager{
		h:               h,
		dht:             dht,
		bootstrapPeers:  bootstrapPeers,
		ctx:             ctx,
		cancel:          cancel,
		state:           StateNormal,
		stateChangeTime: time.Now(),
		historicalPeers: make([]peer.AddrInfo, 0, HistoricalPeerCacheSize),
	}

	// Register connection notifications
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			pm.onConnect(c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			pm.onDisconnect(c.RemotePeer())
		},
	})

	// Start background monitoring
	go pm.monitorLoop()

	return pm
}

// SetCallbacks sets the partition event callbacks.
func (pm *PartitionManager) SetCallbacks(cb PartitionCallbacks) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.callbacks = cb
}

// Stop stops the partition manager.
func (pm *PartitionManager) Stop() {
	pm.cancel()
}

// Status returns current partition status.
func (pm *PartitionManager) Status() PartitionStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return PartitionStatus{
		State:            pm.state,
		PeerCount:        len(pm.h.Network().Peers()),
		StateDuration:    time.Since(pm.stateChangeTime),
		HealingActive:    pm.healingActive,
		HistoricalPeers:  len(pm.historicalPeers),
		LastStateChange:  pm.stateChangeTime,
		PendingPartition: pm.pendingPartition,
	}
}

// State returns the current partition state.
func (pm *PartitionManager) State() PartitionState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.state
}

// IsPartitioned returns true if in partitioned state.
func (pm *PartitionManager) IsPartitioned() bool {
	return pm.State() == StatePartitioned
}

// IsDegraded returns true if in degraded or partitioned state.
func (pm *PartitionManager) IsDegraded() bool {
	state := pm.State()
	return state == StateDegraded || state == StatePartitioned
}

// onConnect handles new peer connections.
func (pm *PartitionManager) onConnect(p peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clear pending partition on connection
	pm.pendingPartition = false

	// Check if this was a recovered peer
	if pm.healingActive && pm.callbacks.OnPeerRecovered != nil {
		pm.callbacks.OnPeerRecovered(p)
	}

	pm.updateStateLocked()
}

// onDisconnect handles peer disconnection.
func (pm *PartitionManager) onDisconnect(p peer.ID) {
	// Remember peer for future reconnection
	pm.rememberPeer(p)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.updateStateLocked()
}

// rememberPeer adds a peer to historical cache for healing.
func (pm *PartitionManager) rememberPeer(p peer.ID) {
	addrs := pm.h.Peerstore().Addrs(p)
	if len(addrs) == 0 {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	addrInfo := peer.AddrInfo{ID: p, Addrs: addrs}

	// Check if already in cache
	for i, existing := range pm.historicalPeers {
		if existing.ID == p {
			// Update addresses and move to front
			pm.historicalPeers = append(
				[]peer.AddrInfo{addrInfo},
				append(pm.historicalPeers[:i], pm.historicalPeers[i+1:]...)...,
			)
			return
		}
	}

	// Add to front of cache
	pm.historicalPeers = append([]peer.AddrInfo{addrInfo}, pm.historicalPeers...)

	// Trim if over capacity
	if len(pm.historicalPeers) > HistoricalPeerCacheSize {
		pm.historicalPeers = pm.historicalPeers[:HistoricalPeerCacheSize]
	}
}

// updateStateLocked updates partition state (caller must hold lock).
func (pm *PartitionManager) updateStateLocked() {
	peerCount := len(pm.h.Network().Peers())
	newState := pm.calculateState(peerCount)

	// Handle transition to partitioned with confirmation delay
	if pm.state != StatePartitioned && newState == StatePartitioned {
		if !pm.pendingPartition {
			pm.pendingPartition = true
			pm.pendingPartitionAt = time.Now()
			return // Don't change state yet
		}
		// Check if confirmation delay has passed
		if time.Since(pm.pendingPartitionAt) < PartitionConfirmationDelay {
			return // Still waiting for confirmation
		}
	}

	if newState != pm.state {
		oldState := pm.state
		pm.state = newState
		pm.stateChangeTime = time.Now()
		pm.pendingPartition = false

		// Trigger healing on partition
		if newState == StatePartitioned && !pm.healingActive {
			go pm.startHealing()
		}

		// Notify callback
		if pm.callbacks.OnStateChange != nil {
			go pm.callbacks.OnStateChange(oldState, newState)
		}
	}
}

// calculateState determines state based on peer count.
func (pm *PartitionManager) calculateState(peerCount int) PartitionState {
	if peerCount < PartitionThreshold {
		return StatePartitioned
	}
	if peerCount < GracefulDegradationThreshold {
		return StateDegraded
	}
	return StateNormal
}

// monitorLoop periodically checks partition state.
func (pm *PartitionManager) monitorLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.mu.Lock()
			pm.updateStateLocked()
			pm.mu.Unlock()
		}
	}
}

// startHealing initiates the healing protocol.
func (pm *PartitionManager) startHealing() {
	pm.mu.Lock()
	if pm.healingActive {
		pm.mu.Unlock()
		return
	}
	pm.healingActive = true
	callback := pm.callbacks.OnHealingStart
	pm.mu.Unlock()

	if callback != nil {
		callback()
	}

	// Run healing loop
	pm.healingLoop()

	pm.mu.Lock()
	pm.healingActive = false
	success := pm.state == StateNormal
	endCallback := pm.callbacks.OnHealingEnd
	pm.mu.Unlock()

	if endCallback != nil {
		endCallback(success)
	}
}

// healingLoop runs the healing protocol.
func (pm *PartitionManager) healingLoop() {
	ticker := time.NewTicker(HealingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			// Check if healing is still needed
			if pm.State() == StateNormal {
				return
			}

			// Try to heal
			pm.attemptHeal()
		}
	}
}

// attemptHeal tries various strategies to reconnect.
func (pm *PartitionManager) attemptHeal() {
	// Strategy 1: Bootstrap peers
	pm.tryBootstrapPeers()

	// Strategy 2: DHT refresh
	pm.tryDHTRefresh()

	// Strategy 3: Historical peers
	pm.tryHistoricalPeers()
}

// tryBootstrapPeers attempts to connect to bootstrap peers.
func (pm *PartitionManager) tryBootstrapPeers() {
	pm.mu.RLock()
	bootstraps := pm.bootstrapPeers
	pm.mu.RUnlock()

	for _, addrInfo := range bootstraps {
		if pm.h.Network().Connectedness(addrInfo.ID) == network.Connected {
			continue
		}

		ctx, cancel := context.WithTimeout(pm.ctx, BootstrapRefreshTimeout)
		_ = pm.h.Connect(ctx, addrInfo)
		cancel()
	}
}

// tryDHTRefresh refreshes the DHT routing table.
func (pm *PartitionManager) tryDHTRefresh() {
	if pm.dht == nil {
		return
	}

	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	_ = pm.dht.Bootstrap(ctx)
}

// tryHistoricalPeers attempts to reconnect to historical peers.
func (pm *PartitionManager) tryHistoricalPeers() {
	pm.mu.RLock()
	peers := make([]peer.AddrInfo, 0, ReconnectBatchSize)
	for i, p := range pm.historicalPeers {
		if i >= ReconnectBatchSize {
			break
		}
		if pm.h.Network().Connectedness(p.ID) != network.Connected {
			peers = append(peers, p)
		}
	}
	pm.mu.RUnlock()

	for _, addrInfo := range peers {
		ctx, cancel := context.WithTimeout(pm.ctx, 10*time.Second)
		_ = pm.h.Connect(ctx, addrInfo)
		cancel()
	}
}

// GracefulDegradation provides methods for operating in degraded mode.
type GracefulDegradation struct {
	pm *PartitionManager
}

// NewGracefulDegradation creates graceful degradation helper.
func NewGracefulDegradation(pm *PartitionManager) *GracefulDegradation {
	return &GracefulDegradation{pm: pm}
}

// ShouldQueueWaves returns true if Waves should be queued for later.
func (gd *GracefulDegradation) ShouldQueueWaves() bool {
	return gd.pm.IsPartitioned()
}

// ShouldReduceGossip returns true if gossip should be reduced.
func (gd *GracefulDegradation) ShouldReduceGossip() bool {
	return gd.pm.IsDegraded()
}

// ShouldDeferShroud returns true if Shroud circuits should be deferred.
func (gd *GracefulDegradation) ShouldDeferShroud() bool {
	return gd.pm.IsPartitioned()
}

// AllowedOperations returns allowed operations in current state.
func (gd *GracefulDegradation) AllowedOperations() DegradedOperations {
	state := gd.pm.State()

	switch state {
	case StateNormal:
		return DegradedOperations{
			WavePublish:      true,
			WaveRelay:        true,
			ShroudCircuits:   true,
			ResonanceVoting:  true,
			FullGossip:       true,
			DirectMessages:   true,
			ContentRetrieval: true,
		}
	case StateDegraded:
		return DegradedOperations{
			WavePublish:      true,
			WaveRelay:        true,
			ShroudCircuits:   false, // Defer complex operations
			ResonanceVoting:  true,
			FullGossip:       false, // Reduce gossip rate
			DirectMessages:   true,
			ContentRetrieval: true,
		}
	case StatePartitioned:
		return DegradedOperations{
			WavePublish:      false, // Queue for later
			WaveRelay:        false,
			ShroudCircuits:   false,
			ResonanceVoting:  false,
			FullGossip:       false,
			DirectMessages:   true, // Direct connections still work
			ContentRetrieval: true, // Can read local cache
		}
	default:
		return DegradedOperations{}
	}
}

// DegradedOperations defines what operations are allowed.
type DegradedOperations struct {
	WavePublish      bool
	WaveRelay        bool
	ShroudCircuits   bool
	ResonanceVoting  bool
	FullGossip       bool
	DirectMessages   bool
	ContentRetrieval bool
}

// HealingProtocol provides advanced healing strategies.
type HealingProtocol struct {
	pm           *PartitionManager
	peerExchange PeerExchanger
}

// PeerExchanger is an interface for peer exchange protocol.
type PeerExchanger interface {
	RequestPeers(ctx context.Context, from peer.ID) ([]peer.AddrInfo, error)
}

// NewHealingProtocol creates a new healing protocol.
func NewHealingProtocol(pm *PartitionManager, pe PeerExchanger) *HealingProtocol {
	return &HealingProtocol{
		pm:           pm,
		peerExchange: pe,
	}
}

// TriggerManualHeal triggers manual healing attempt.
func (hp *HealingProtocol) TriggerManualHeal() {
	go hp.pm.attemptHeal()
}

// ExchangeWithPeer requests peers from a connected peer.
func (hp *HealingProtocol) ExchangeWithPeer(ctx context.Context, p peer.ID) error {
	if hp.peerExchange == nil {
		return nil
	}

	newPeers, err := hp.peerExchange.RequestPeers(ctx, p)
	if err != nil {
		return err
	}

	// Try connecting to new peers
	for _, addrInfo := range newPeers {
		if hp.pm.h.Network().Connectedness(addrInfo.ID) == network.Connected {
			continue
		}

		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_ = hp.pm.h.Connect(connectCtx, addrInfo)
		cancel()
	}

	return nil
}

// AddHistoricalPeer manually adds a peer to historical cache.
func (hp *HealingProtocol) AddHistoricalPeer(addrInfo peer.AddrInfo) {
	hp.pm.rememberPeer(addrInfo.ID)

	hp.pm.mu.Lock()
	// Update with full address info
	for i, p := range hp.pm.historicalPeers {
		if p.ID == addrInfo.ID {
			hp.pm.historicalPeers[i] = addrInfo
			hp.pm.mu.Unlock()
			return
		}
	}
	hp.pm.mu.Unlock()
}
