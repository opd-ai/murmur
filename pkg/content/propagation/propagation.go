// Package propagation provides gossip relay logic, hop counting, and deduplication.
// Per WAVE_PROPAGATION.md, Waves propagate through the mesh with hop tracking.
package propagation

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"

	"github.com/opd-ai/murmur/pkg/content/waves"
	pb "github.com/opd-ai/murmur/proto"
)

// MaxHops is the maximum number of hops a Wave can traverse.
const MaxHops = 10

// DefaultCacheDuration is how long to retain wave IDs for deduplication.
const DefaultCacheDuration = 24 * time.Hour

// DefaultDifficulty is the PoW difficulty for Wave validation.
const DefaultDifficulty = 8 // Lower for testing; production uses 20

// Errors for propagation operations.
var (
	ErrMaxHopsExceeded = errors.New("wave exceeded maximum hop count")
	ErrDuplicateWave   = errors.New("duplicate wave detected")
	ErrExpiredWave     = errors.New("wave has expired")
	ErrInvalidWave     = errors.New("wave validation failed")
)

// Relay handles Wave propagation through the gossip network.
type Relay struct {
	mu       sync.RWMutex
	seen     map[string]time.Time // wave ID -> first seen time
	maxHops  uint32
	cacheTTL time.Duration

	// Handler is called for each valid Wave received.
	Handler func(wave *pb.Wave)
}

// NewRelay creates a new propagation relay.
func NewRelay() *Relay {
	return &Relay{
		seen:     make(map[string]time.Time),
		maxHops:  MaxHops,
		cacheTTL: DefaultCacheDuration,
	}
}

// RelayConfig configures the relay behavior.
type RelayConfig struct {
	MaxHops  uint32
	CacheTTL time.Duration
	Handler  func(wave *pb.Wave)
}

// NewRelayWithConfig creates a relay with custom configuration.
func NewRelayWithConfig(cfg RelayConfig) *Relay {
	maxHops := cfg.MaxHops
	if maxHops == 0 {
		maxHops = MaxHops
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = DefaultCacheDuration
	}

	return &Relay{
		seen:     make(map[string]time.Time),
		maxHops:  maxHops,
		cacheTTL: cacheTTL,
		Handler:  cfg.Handler,
	}
}

// Receive processes an incoming Wave from the gossip network.
// Returns the Wave with incremented hop count if it should be relayed,
// or an error if the Wave should be dropped.
func (r *Relay) Receive(wave *pb.Wave) (*pb.Wave, error) {
	if err := r.validateIncomingWave(wave); err != nil {
		return nil, err
	}

	waveID := string(wave.WaveId)
	r.markSeen(waveID)
	r.notifyHandler(wave)

	return waves.IncrementHop(wave), nil
}

// validateIncomingWave checks all constraints for an incoming wave.
func (r *Relay) validateIncomingWave(wave *pb.Wave) error {
	if wave == nil {
		return ErrInvalidWave
	}

	if r.hasSeen(string(wave.WaveId)) {
		return ErrDuplicateWave
	}

	if wave.HopCount >= r.maxHops {
		return ErrMaxHopsExceeded
	}

	if waves.IsExpired(wave) {
		return ErrExpiredWave
	}

	if err := waves.Validate(wave, DefaultDifficulty); err != nil {
		return ErrInvalidWave
	}

	return nil
}

// notifyHandler calls the handler callback if set.
func (r *Relay) notifyHandler(wave *pb.Wave) {
	if r.Handler != nil {
		r.Handler(wave)
	}
}

// hasSeen checks if a Wave ID has been seen recently.
func (r *Relay) hasSeen(waveID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.seen[waveID]
	return exists
}

// markSeen records a Wave ID as seen.
func (r *Relay) markSeen(waveID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seen[waveID] = time.Now()
}

// CleanExpired removes expired entries from the seen cache.
func (r *Relay) CleanExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.cacheTTL)
	count := 0

	for id, seenAt := range r.seen {
		if seenAt.Before(cutoff) {
			delete(r.seen, id)
			count++
		}
	}

	return count
}

// CacheSize returns the current number of cached Wave IDs.
func (r *Relay) CacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.seen)
}

// StartCleanup runs periodic cache cleanup.
// Returns a cancel function to stop the cleanup goroutine.
func (r *Relay) StartCleanup(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.CleanExpired()
			}
		}
	}()

	return cancel
}

// ComputeWaveID generates a BLAKE3 hash for deduplication.
func ComputeWaveID(data []byte) []byte {
	h := blake3.New()
	h.Write(data)
	return h.Sum(nil)
}
