// Package mesh provides peer scoring, mesh health monitoring, and connection management.
// This file implements churn handling: mesh repair and DHT refresh on disconnect.
// Per DESIGN_DOCUMENT.md Part II §6, the system must handle peer churn gracefully.
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

// Churn handling constants.
const (
	// ChurnDetectionWindow is the time window to measure churn rate.
	ChurnDetectionWindow = 5 * time.Minute

	// HighChurnThreshold is disconnects per minute triggering repair mode.
	HighChurnThreshold = 5

	// RepairCooldown prevents repair spam.
	RepairCooldown = 30 * time.Second

	// DHTRefreshInterval is how often to refresh DHT routing table.
	DHTRefreshInterval = 10 * time.Minute

	// QuickRepairDelay is the delay before starting mesh repair.
	QuickRepairDelay = 5 * time.Second

	// ReconnectAttemptLimit is max reconnect attempts for important peers.
	ReconnectAttemptLimit = 3
)

// ChurnHandler manages mesh repair and DHT refresh on peer disconnection.
type ChurnHandler struct {
	h               host.Host
	dht             *dht.IpfsDHT
	degreeCtrl      *DegreeController
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	disconnectTimes []time.Time
	lastRepair      time.Time
	importantPeers  map[peer.ID]bool
	callbacks       ChurnCallbacks
}

// ChurnCallbacks allows external handlers to be notified of churn events.
type ChurnCallbacks struct {
	OnHighChurn     func()
	OnMeshRepair    func(added int)
	OnDHTRefresh    func()
	OnReconnect     func(p peer.ID, success bool)
}

// ChurnStats provides statistics about peer churn.
type ChurnStats struct {
	DisconnectsLastWindow int
	ChurnRatePerMinute    float64
	IsHighChurn           bool
	LastRepairTime        time.Time
	TotalImportantPeers   int
}

// NewChurnHandler creates a new churn handler.
func NewChurnHandler(h host.Host, dht *dht.IpfsDHT, degreeCtrl *DegreeController) *ChurnHandler {
	ctx, cancel := context.WithCancel(context.Background())
	ch := &ChurnHandler{
		h:              h,
		dht:            dht,
		degreeCtrl:     degreeCtrl,
		ctx:            ctx,
		cancel:         cancel,
		importantPeers: make(map[peer.ID]bool),
	}

	// Register disconnect notifee
	h.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(n network.Network, c network.Conn) {
			ch.onDisconnect(c.RemotePeer())
		},
	})

	return ch
}

// SetCallbacks sets the churn event callbacks.
func (ch *ChurnHandler) SetCallbacks(callbacks ChurnCallbacks) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks = callbacks
}

// MarkImportant marks a peer as important for reconnection.
func (ch *ChurnHandler) MarkImportant(p peer.ID) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.importantPeers[p] = true
}

// UnmarkImportant removes a peer from the important list.
func (ch *ChurnHandler) UnmarkImportant(p peer.ID) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	delete(ch.importantPeers, p)
}

// Start begins the churn handling background tasks.
func (ch *ChurnHandler) Start() {
	go ch.dhtRefreshLoop()
}

// Stop stops the churn handler.
func (ch *ChurnHandler) Stop() {
	ch.cancel()
}

// onDisconnect handles peer disconnection.
func (ch *ChurnHandler) onDisconnect(p peer.ID) {
	ch.mu.Lock()
	// Record disconnect time
	ch.disconnectTimes = append(ch.disconnectTimes, time.Now())
	isImportant := ch.importantPeers[p]
	ch.mu.Unlock()

	// Prune old disconnect times
	ch.pruneDisconnectTimes()

	// Check for high churn
	if ch.isHighChurn() {
		ch.triggerHighChurnResponse()
	}

	// Attempt reconnection for important peers
	if isImportant {
		go ch.attemptReconnection(p)
	}

	// Schedule mesh repair if needed
	if ch.needsMeshRepair() {
		go ch.scheduleMeshRepair()
	}
}

// pruneDisconnectTimes removes disconnect times outside the detection window.
func (ch *ChurnHandler) pruneDisconnectTimes() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	cutoff := time.Now().Add(-ChurnDetectionWindow)
	pruned := make([]time.Time, 0)

	for _, t := range ch.disconnectTimes {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}

	ch.disconnectTimes = pruned
}

// isHighChurn returns true if churn rate exceeds threshold.
func (ch *ChurnHandler) isHighChurn() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.disconnectTimes) == 0 {
		return false
	}

	// Calculate disconnects per minute
	windowMinutes := ChurnDetectionWindow.Minutes()
	rate := float64(len(ch.disconnectTimes)) / windowMinutes

	return rate >= float64(HighChurnThreshold)
}

// triggerHighChurnResponse handles high churn detection.
func (ch *ChurnHandler) triggerHighChurnResponse() {
	ch.mu.RLock()
	callback := ch.callbacks.OnHighChurn
	ch.mu.RUnlock()

	if callback != nil {
		callback()
	}

	// Trigger immediate DHT refresh
	go ch.refreshDHT()
}

// needsMeshRepair returns true if mesh needs repair.
func (ch *ChurnHandler) needsMeshRepair() bool {
	if ch.degreeCtrl == nil {
		return false
	}

	status := ch.degreeCtrl.Status()
	return status.NeedsMore
}

// scheduleMeshRepair schedules a mesh repair after a short delay.
func (ch *ChurnHandler) scheduleMeshRepair() {
	ch.mu.Lock()
	// Check cooldown
	if time.Since(ch.lastRepair) < RepairCooldown {
		ch.mu.Unlock()
		return
	}
	ch.lastRepair = time.Now()
	ch.mu.Unlock()

	select {
	case <-ch.ctx.Done():
		return
	case <-time.After(QuickRepairDelay):
		ch.repairMesh()
	}
}

// repairMesh attempts to acquire new peers to repair the mesh.
func (ch *ChurnHandler) repairMesh() {
	if ch.degreeCtrl == nil {
		return
	}

	// Force an immediate degree adjustment
	ch.degreeCtrl.ForceAdjust()

	ch.mu.RLock()
	callback := ch.callbacks.OnMeshRepair
	ch.mu.RUnlock()

	if callback != nil {
		status := ch.degreeCtrl.Status()
		callback(status.Target - status.Current)
	}
}

// attemptReconnection attempts to reconnect to an important peer.
func (ch *ChurnHandler) attemptReconnection(p peer.ID) {
	ch.mu.RLock()
	callback := ch.callbacks.OnReconnect
	ch.mu.RUnlock()

	// Get peer's addresses from peerstore
	addrs := ch.h.Peerstore().Addrs(p)
	if len(addrs) == 0 {
		if callback != nil {
			callback(p, false)
		}
		return
	}

	addrInfo := peer.AddrInfo{
		ID:    p,
		Addrs: addrs,
	}

	var success bool
	for attempt := 0; attempt < ReconnectAttemptLimit; attempt++ {
		select {
		case <-ch.ctx.Done():
			return
		default:
		}

		// Exponential backoff
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(delay)
		}

		ctx, cancel := context.WithTimeout(ch.ctx, 10*time.Second)
		err := ch.h.Connect(ctx, addrInfo)
		cancel()

		if err == nil {
			success = true
			break
		}
	}

	if callback != nil {
		callback(p, success)
	}
}

// dhtRefreshLoop periodically refreshes the DHT routing table.
func (ch *ChurnHandler) dhtRefreshLoop() {
	ticker := time.NewTicker(DHTRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ch.ctx.Done():
			return
		case <-ticker.C:
			ch.refreshDHT()
		}
	}
}

// refreshDHT refreshes the DHT routing table.
func (ch *ChurnHandler) refreshDHT() {
	if ch.dht == nil {
		return
	}

	// DHT Bootstrap handles routing table refresh
	ctx, cancel := context.WithTimeout(ch.ctx, 30*time.Second)
	defer cancel()

	_ = ch.dht.Bootstrap(ctx)

	ch.mu.RLock()
	callback := ch.callbacks.OnDHTRefresh
	ch.mu.RUnlock()

	if callback != nil {
		callback()
	}
}

// Stats returns current churn statistics.
func (ch *ChurnHandler) Stats() ChurnStats {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	windowMinutes := ChurnDetectionWindow.Minutes()
	rate := 0.0
	if len(ch.disconnectTimes) > 0 {
		rate = float64(len(ch.disconnectTimes)) / windowMinutes
	}

	return ChurnStats{
		DisconnectsLastWindow: len(ch.disconnectTimes),
		ChurnRatePerMinute:    rate,
		IsHighChurn:           rate >= float64(HighChurnThreshold),
		LastRepairTime:        ch.lastRepair,
		TotalImportantPeers:   len(ch.importantPeers),
	}
}

// PartitionDetector detects network partitions.
type PartitionDetector struct {
	h                host.Host
	minConnected     int
	lastConnectedAt  time.Time
	partitionStart   time.Time
	isPartitioned    bool
	mu               sync.RWMutex
	onPartition      func(bool)
}

// NewPartitionDetector creates a new partition detector.
func NewPartitionDetector(h host.Host, minConnected int) *PartitionDetector {
	pd := &PartitionDetector{
		h:               h,
		minConnected:    minConnected,
		lastConnectedAt: time.Now(),
	}

	// Monitor connection changes
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			pd.onConnectionChange()
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			pd.onConnectionChange()
		},
	})

	return pd
}

// SetPartitionCallback sets the callback for partition events.
func (pd *PartitionDetector) SetPartitionCallback(cb func(partitioned bool)) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.onPartition = cb
}

func (pd *PartitionDetector) onConnectionChange() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	connected := len(pd.h.Network().Peers())
	wasPartitioned := pd.isPartitioned

	if connected < pd.minConnected {
		if !pd.isPartitioned {
			pd.isPartitioned = true
			pd.partitionStart = time.Now()
		}
	} else {
		pd.isPartitioned = false
		pd.lastConnectedAt = time.Now()
	}

	// Notify on state change
	if wasPartitioned != pd.isPartitioned && pd.onPartition != nil {
		pd.onPartition(pd.isPartitioned)
	}
}

// IsPartitioned returns true if currently partitioned.
func (pd *PartitionDetector) IsPartitioned() bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	return pd.isPartitioned
}

// PartitionDuration returns how long we've been partitioned.
func (pd *PartitionDetector) PartitionDuration() time.Duration {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	if !pd.isPartitioned {
		return 0
	}
	return time.Since(pd.partitionStart)
}
