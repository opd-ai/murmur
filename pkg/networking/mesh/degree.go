// Package mesh provides peer scoring, mesh health monitoring, and connection management.
// This file implements mesh degree control with target 6, bounds 4-12.
// Per DESIGN_DOCUMENT.md Part II §6.
package mesh

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Degree control constants per DESIGN_DOCUMENT.md.
const (
	// TargetDegree is the ideal number of peer connections.
	TargetDegree = 6

	// LowDegreeThreshold triggers connection acquisition.
	LowDegreeThreshold = 4

	// HighDegreeThreshold triggers connection pruning.
	HighDegreeThreshold = 12

	// DegreeCheckInterval is how often to check mesh degree.
	DegreeCheckInterval = 30 * time.Second

	// ConnectionAcquisitionTimeout is how long to try connecting to a peer.
	ConnectionAcquisitionTimeout = 10 * time.Second
)

// DegreeController manages mesh degree within specified bounds.
type DegreeController struct {
	h               host.Host
	manager         *Manager
	peerSource      PeerSource
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	pruneCallback   PruneCallback
	acquireCallback AcquireCallback
}

// PeerSource provides candidate peers for connection acquisition.
type PeerSource interface {
	// GetCandidatePeers returns peers we could connect to.
	GetCandidatePeers(ctx context.Context, count int) []peer.AddrInfo
}

// PruneCallback is called when a peer is pruned.
type PruneCallback func(p peer.ID, reason string)

// AcquireCallback is called when attempting to acquire a new peer.
type AcquireCallback func(p peer.AddrInfo, success bool)

// DegreeStatus represents the current mesh degree state.
type DegreeStatus struct {
	Current     int
	Target      int
	LowBound    int
	HighBound   int
	NeedsMore   bool
	NeedsPrune  bool
	IsHealthy   bool
}

// NewDegreeController creates a new mesh degree controller.
func NewDegreeController(h host.Host, manager *Manager) *DegreeController {
	ctx, cancel := context.WithCancel(context.Background())
	return &DegreeController{
		h:       h,
		manager: manager,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetPeerSource sets the source for candidate peers.
func (dc *DegreeController) SetPeerSource(source PeerSource) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.peerSource = source
}

// SetPruneCallback sets the callback for prune events.
func (dc *DegreeController) SetPruneCallback(cb PruneCallback) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.pruneCallback = cb
}

// SetAcquireCallback sets the callback for acquire events.
func (dc *DegreeController) SetAcquireCallback(cb AcquireCallback) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.acquireCallback = cb
}

// Start begins the degree control loop.
func (dc *DegreeController) Start() {
	go dc.controlLoop()
}

// Stop stops the degree controller.
func (dc *DegreeController) Stop() {
	dc.cancel()
}

// Status returns the current mesh degree status.
func (dc *DegreeController) Status() DegreeStatus {
	current := dc.manager.PeerCount()
	return DegreeStatus{
		Current:    current,
		Target:     TargetDegree,
		LowBound:   LowDegreeThreshold,
		HighBound:  HighDegreeThreshold,
		NeedsMore:  current < LowDegreeThreshold,
		NeedsPrune: current > HighDegreeThreshold,
		IsHealthy:  current >= LowDegreeThreshold && current <= HighDegreeThreshold,
	}
}

// controlLoop periodically checks and adjusts mesh degree.
func (dc *DegreeController) controlLoop() {
	ticker := time.NewTicker(DegreeCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.adjustDegree()
		}
	}
}

// adjustDegree adjusts mesh degree towards target.
func (dc *DegreeController) adjustDegree() {
	status := dc.Status()

	if status.NeedsMore {
		dc.acquirePeers(TargetDegree - status.Current)
	} else if status.NeedsPrune {
		dc.prunePeers(status.Current - HighDegreeThreshold)
	}
}

// acquirePeers attempts to connect to additional peers.
func (dc *DegreeController) acquirePeers(count int) {
	dc.mu.RLock()
	source := dc.peerSource
	callback := dc.acquireCallback
	dc.mu.RUnlock()

	if source == nil {
		return
	}

	// Get candidate peers
	ctx, cancel := context.WithTimeout(dc.ctx, ConnectionAcquisitionTimeout)
	defer cancel()

	candidates := source.GetCandidatePeers(ctx, count*2) // Request extra candidates

	connected := 0
	for _, addrInfo := range candidates {
		if connected >= count {
			break
		}

		// Skip if already connected
		if dc.h.Network().Connectedness(addrInfo.ID) == 1 { // Connected
			continue
		}

		// Attempt connection
		err := dc.h.Connect(ctx, addrInfo)
		success := err == nil

		if callback != nil {
			callback(addrInfo, success)
		}

		if success {
			connected++
		}
	}
}

// prunePeers disconnects lowest priority peers.
func (dc *DegreeController) prunePeers(count int) {
	dc.mu.RLock()
	callback := dc.pruneCallback
	dc.mu.RUnlock()

	peers := dc.manager.Peers()
	if len(peers) == 0 {
		return
	}

	// Sort by priority (highest priority value = lowest importance)
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Priority > peers[j].Priority
	})

	// Prune lowest priority peers
	pruned := 0
	for _, p := range peers {
		if pruned >= count {
			break
		}

		// Don't prune identity connections
		if p.Priority == PriorityIdentity {
			continue
		}

		_ = dc.h.Network().ClosePeer(p.ID)

		if callback != nil {
			callback(p.ID, "degree_pruning")
		}

		pruned++
	}
}

// ForceAdjust forces an immediate degree adjustment.
func (dc *DegreeController) ForceAdjust() {
	dc.adjustDegree()
}

// ScoreBasedPruning prunes peers with scores below threshold.
type ScoreBasedPruning struct {
	dc         *DegreeController
	scoreFunc  PeerScoreFunc
	threshold  float64
}

// PeerScoreFunc returns the score for a peer.
type PeerScoreFunc func(p peer.ID) float64

// NewScoreBasedPruning creates a score-based pruning controller.
func NewScoreBasedPruning(dc *DegreeController, scoreFunc PeerScoreFunc, threshold float64) *ScoreBasedPruning {
	return &ScoreBasedPruning{
		dc:        dc,
		scoreFunc: scoreFunc,
		threshold: threshold,
	}
}

// PruneLowScorePeers removes peers with scores below threshold.
func (sbp *ScoreBasedPruning) PruneLowScorePeers() int {
	dc := sbp.dc
	dc.mu.RLock()
	callback := dc.pruneCallback
	dc.mu.RUnlock()

	peers := dc.manager.Peers()
	pruned := 0

	for _, p := range peers {
		score := sbp.scoreFunc(p.ID)
		if score < sbp.threshold {
			// Don't prune if we'd go below minimum
			if dc.manager.PeerCount() <= LowDegreeThreshold {
				break
			}

			_ = dc.h.Network().ClosePeer(p.ID)

			if callback != nil {
				callback(p.ID, "low_score")
			}

			pruned++
		}
	}

	return pruned
}

// DegreeStats holds statistics about degree control operations.
type DegreeStats struct {
	TotalAcquireAttempts int64
	SuccessfulAcquires   int64
	TotalPrunes          int64
	ScoreBasedPrunes     int64
}

// StatTracker tracks degree control statistics.
type StatTracker struct {
	stats DegreeStats
	mu    sync.RWMutex
}

// NewStatTracker creates a new statistics tracker.
func NewStatTracker() *StatTracker {
	return &StatTracker{}
}

// RecordAcquireAttempt records an acquisition attempt.
func (st *StatTracker) RecordAcquireAttempt(success bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.stats.TotalAcquireAttempts++
	if success {
		st.stats.SuccessfulAcquires++
	}
}

// RecordPrune records a pruning event.
func (st *StatTracker) RecordPrune(reason string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.stats.TotalPrunes++
	if reason == "low_score" {
		st.stats.ScoreBasedPrunes++
	}
}

// GetStats returns current statistics.
func (st *StatTracker) GetStats() DegreeStats {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.stats
}
