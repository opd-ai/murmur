// Package resonance provides local reputation computation and rank thresholds.
// This file implements the decay manager for periodic background computation.
package resonance

import (
	"context"
	"sync"
	"time"
)

// DecayManager handles periodic decay computation for all Resonance scores.
// Per RESONANCE_SYSTEM.md, decay should be applied every 60 seconds.
type DecayManager struct {
	mu sync.RWMutex

	// Registered scorers.
	surfaceScorer *SurfaceScorer
	specterScorer *SpecterScorer
	legacyScorer  *Scorer

	// Configuration.
	interval time.Duration
	running  bool
	cancel   context.CancelFunc
	done     chan struct{}

	// Stats.
	lastRun     time.Time
	runCount    int64
	surfaceHits int64
	specterHits int64
	legacyHits  int64
}

// DecayManagerConfig configures the decay manager.
type DecayManagerConfig struct {
	Interval time.Duration // Time between decay runs (default: 60s)
}

// DefaultDecayManagerConfig returns the standard configuration.
func DefaultDecayManagerConfig() DecayManagerConfig {
	return DecayManagerConfig{
		Interval: 60 * time.Second,
	}
}

// NewDecayManager creates a new decay manager.
func NewDecayManager() *DecayManager {
	return &DecayManager{
		interval: 60 * time.Second,
		done:     make(chan struct{}),
	}
}

// NewDecayManagerWithConfig creates a decay manager with custom configuration.
func NewDecayManagerWithConfig(cfg DecayManagerConfig) *DecayManager {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &DecayManager{
		interval: interval,
		done:     make(chan struct{}),
	}
}

// RegisterSurfaceScorer registers a SurfaceScorer for periodic decay.
func (dm *DecayManager) RegisterSurfaceScorer(scorer *SurfaceScorer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.surfaceScorer = scorer
}

// RegisterSpecterScorer registers a SpecterScorer for periodic decay.
func (dm *DecayManager) RegisterSpecterScorer(scorer *SpecterScorer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.specterScorer = scorer
}

// RegisterLegacyScorer registers a legacy Scorer for periodic decay.
func (dm *DecayManager) RegisterLegacyScorer(scorer *Scorer) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.legacyScorer = scorer
}

// Start begins periodic decay computation in a background goroutine.
func (dm *DecayManager) Start(ctx context.Context) {
	dm.mu.Lock()
	if dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = true

	// Create cancellable context.
	ctx, cancel := context.WithCancel(ctx)
	dm.cancel = cancel
	dm.done = make(chan struct{})
	dm.mu.Unlock()

	go dm.runLoop(ctx)
}

// Stop halts the decay manager.
func (dm *DecayManager) Stop() {
	dm.mu.Lock()
	if !dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = false
	if dm.cancel != nil {
		dm.cancel()
	}
	done := dm.done
	dm.mu.Unlock()

	// Wait for loop to exit.
	<-done
}

// IsRunning returns whether the manager is running.
func (dm *DecayManager) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.running
}

// RunOnce performs a single decay computation pass.
// This can be called manually or is called automatically by the background loop.
func (dm *DecayManager) RunOnce() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.runCount++
	dm.lastRun = time.Now()

	// Apply decay to Surface Resonance scores.
	if dm.surfaceScorer != nil {
		dm.surfaceScorer.DecayAll()
		dm.surfaceHits += int64(dm.surfaceScorer.Count())
	}

	// Apply decay to Specter Resonance scores.
	// Note: SpecterScore doesn't have explicit DecayAll yet, but invalidating
	// cache forces recomputation which incorporates time-based decay.
	if dm.specterScorer != nil {
		dm.invalidateSpecterCaches()
		dm.specterHits += int64(dm.specterScorer.Count())
	}

	// Apply decay to legacy scores.
	if dm.legacyScorer != nil {
		dm.invalidateLegacyCaches()
		dm.legacyHits += int64(dm.legacyScorer.Count())
	}
}

// invalidateSpecterCaches invalidates all SpecterScore caches.
func (dm *DecayManager) invalidateSpecterCaches() {
	if dm.specterScorer == nil {
		return
	}

	dm.specterScorer.mu.RLock()
	defer dm.specterScorer.mu.RUnlock()

	for _, score := range dm.specterScorer.scores {
		score.invalidateCache()
	}
}

// invalidateLegacyCaches invalidates all legacy Score caches.
func (dm *DecayManager) invalidateLegacyCaches() {
	if dm.legacyScorer == nil {
		return
	}

	dm.legacyScorer.mu.RLock()
	defer dm.legacyScorer.mu.RUnlock()

	for _, score := range dm.legacyScorer.scores {
		score.invalidateCache()
	}
}

// runLoop is the background goroutine that periodically applies decay.
func (dm *DecayManager) runLoop(ctx context.Context) {
	ticker := time.NewTicker(dm.interval)
	defer ticker.Stop()

	defer func() {
		close(dm.done)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.RunOnce()
		}
	}
}

// Stats returns statistics about decay manager operation.
func (dm *DecayManager) Stats() DecayStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return DecayStats{
		Running:     dm.running,
		Interval:    dm.interval,
		LastRun:     dm.lastRun,
		RunCount:    dm.runCount,
		SurfaceHits: dm.surfaceHits,
		SpecterHits: dm.specterHits,
		LegacyHits:  dm.legacyHits,
	}
}

// DecayStats contains statistics about decay manager operation.
type DecayStats struct {
	Running     bool
	Interval    time.Duration
	LastRun     time.Time
	RunCount    int64
	SurfaceHits int64
	SpecterHits int64
	LegacyHits  int64
}

// SetInterval updates the decay interval (only takes effect on next Start).
func (dm *DecayManager) SetInterval(d time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if d > 0 {
		dm.interval = d
	}
}
