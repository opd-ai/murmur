// Package propagation provides gossip relay logic, hop counting, and deduplication.
// This file implements cross-layer bridge injection for Veiled Waves.
// Per WAVE_PROPAGATION.md, Hybrid+ nodes relay Veiled Waves between layers.
package propagation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// Bridge topic names per WAVE_PROPAGATION.md.
const (
	// TopicSurfaceWaves is the Surface Layer gossip topic.
	TopicSurfaceWaves = "/murmur/surface/waves/1.0"

	// TopicAnonymousWaves is the Anonymous Layer gossip topic.
	TopicAnonymousWaves = "/murmur/anonymous/waves/1.0"
)

// Wave type constants per WAVES.md.
const (
	// WaveTypeVeiledNum is the numeric type byte for Veiled Waves (0x03).
	WaveTypeVeiledNum = 3
)

// Errors for bridge operations.
var (
	ErrNotVeiledWave     = errors.New("wave is not a Veiled Wave")
	ErrBridgeDisabled    = errors.New("bridge injection is disabled")
	ErrNoSurfacePublisher = errors.New("no surface layer publisher configured")
	ErrNoAnonymousPublisher = errors.New("no anonymous layer publisher configured")
)

// BridgeConfig configures the cross-layer bridge.
type BridgeConfig struct {
	// Enabled controls whether bridge injection is active.
	Enabled bool

	// SurfacePublisher publishes to the Surface Layer topic.
	SurfacePublisher Publisher

	// AnonymousPublisher publishes to the Anonymous Layer topic.
	AnonymousPublisher Publisher

	// MaxBridgeRate limits injections per second (0 = unlimited).
	MaxBridgeRate float64

	// DeduplicationTTL is how long to track injected wave IDs.
	DeduplicationTTL time.Duration
}

// Bridge handles cross-layer Veiled Wave injection.
// Per WAVE_PROPAGATION.md, bridge nodes (Hybrid+) inject Veiled Waves
// from the Anonymous Layer into the Surface Layer gossip topic.
type Bridge struct {
	mu                sync.RWMutex
	enabled           atomic.Bool
	surfacePublisher  Publisher
	anonymousPublisher Publisher
	injected          map[string]time.Time // Wave ID -> injection time
	deduplicationTTL  time.Duration
	rateLimiter       *bridgeRateLimiter
	stats             BridgeStats
}

// BridgeStats tracks bridge injection statistics.
type BridgeStats struct {
	mu                sync.RWMutex
	InjectedToSurface uint64
	InjectedToAnonymous uint64
	DuplicatesSkipped uint64
	RateLimited       uint64
	InvalidWaves      uint64
}

// bridgeRateLimiter implements token bucket rate limiting.
type bridgeRateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxRate  float64
	lastTime time.Time
	capacity float64
}

// NewBridge creates a new cross-layer bridge.
func NewBridge(cfg BridgeConfig) *Bridge {
	b := &Bridge{
		injected:         make(map[string]time.Time),
		deduplicationTTL: cfg.DeduplicationTTL,
		surfacePublisher: cfg.SurfacePublisher,
		anonymousPublisher: cfg.AnonymousPublisher,
	}

	if cfg.DeduplicationTTL == 0 {
		b.deduplicationTTL = 24 * time.Hour
	}

	b.enabled.Store(cfg.Enabled)

	if cfg.MaxBridgeRate > 0 {
		b.rateLimiter = &bridgeRateLimiter{
			tokens:   cfg.MaxBridgeRate,
			maxRate:  cfg.MaxBridgeRate,
			lastTime: time.Now(),
			capacity: cfg.MaxBridgeRate * 2, // 2-second burst
		}
	}

	return b
}

// NewBridgeWithPublishers creates a bridge with both publishers.
func NewBridgeWithPublishers(surface, anonymous Publisher) *Bridge {
	return NewBridge(BridgeConfig{
		Enabled:            true,
		SurfacePublisher:   surface,
		AnonymousPublisher: anonymous,
		DeduplicationTTL:   24 * time.Hour,
	})
}

// SetEnabled enables or disables bridge injection.
func (b *Bridge) SetEnabled(enabled bool) {
	b.enabled.Store(enabled)
}

// IsEnabled returns whether bridge injection is enabled.
func (b *Bridge) IsEnabled() bool {
	return b.enabled.Load()
}

// SetSurfacePublisher sets the Surface Layer publisher.
func (b *Bridge) SetSurfacePublisher(p Publisher) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.surfacePublisher = p
}

// SetAnonymousPublisher sets the Anonymous Layer publisher.
func (b *Bridge) SetAnonymousPublisher(p Publisher) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.anonymousPublisher = p
}

// bridgeDirection indicates the direction of cross-layer bridge injection.
type bridgeDirection int

const (
	directionToSurface bridgeDirection = iota
	directionToAnonymous
)

// injectTo performs cross-layer Veiled Wave injection with validation,
// deduplication, and rate limiting. This is the shared implementation
// for both injection directions.
func (b *Bridge) injectTo(ctx context.Context, wave *pb.Wave, dir bridgeDirection) error {
	if !b.enabled.Load() {
		return ErrBridgeDisabled
	}

	// Validate Wave type.
	if !isVeiledWave(wave) {
		b.recordInvalid()
		return ErrNotVeiledWave
	}

	// Check for duplicate injection.
	waveID := string(wave.WaveId)
	if b.hasInjected(waveID) {
		b.recordDuplicate()
		return ErrDuplicateWave
	}

	// Check rate limit.
	if b.rateLimiter != nil && !b.rateLimiter.allow() {
		b.recordRateLimited()
		return ErrRateLimited
	}

	// Get publisher and nil-publisher error based on direction.
	b.mu.RLock()
	var publisher Publisher
	var nilErr error
	if dir == directionToSurface {
		publisher = b.surfacePublisher
		nilErr = ErrNoSurfacePublisher
	} else {
		publisher = b.anonymousPublisher
		nilErr = ErrNoAnonymousPublisher
	}
	b.mu.RUnlock()

	if publisher == nil {
		return nilErr
	}

	// Wrap and publish.
	envelope, err := wrapWaveInEnvelope(wave)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	if err := publisher.Publish(ctx, data); err != nil {
		return err
	}

	// Mark as injected and record stats.
	b.markInjected(waveID)
	if dir == directionToSurface {
		b.recordSurfaceInjection()
	} else {
		b.recordAnonymousInjection()
	}

	return nil
}

// InjectToSurface injects a Veiled Wave from Anonymous Layer to Surface Layer.
// Per WAVE_PROPAGATION.md, this is the primary bridge injection direction.
// The Wave is forwarded without modification - the bridge does not sign it.
func (b *Bridge) InjectToSurface(ctx context.Context, wave *pb.Wave) error {
	return b.injectTo(ctx, wave, directionToSurface)
}

// InjectToAnonymous injects a Veiled Wave from Surface Layer to Anonymous Layer.
// This is the reverse direction, used when Surface peers discover new Veiled Waves.
func (b *Bridge) InjectToAnonymous(ctx context.Context, wave *pb.Wave) error {
	return b.injectTo(ctx, wave, directionToAnonymous)
}

// ProcessAnonymousWave handles a Wave received from the Anonymous Layer.
// If it's a Veiled Wave, it injects to Surface Layer.
func (b *Bridge) ProcessAnonymousWave(ctx context.Context, wave *pb.Wave) error {
	if !isVeiledWave(wave) {
		return nil // Not a Veiled Wave, nothing to bridge
	}
	return b.InjectToSurface(ctx, wave)
}

// ProcessSurfaceWave handles a Wave received from the Surface Layer.
// If it's a Veiled Wave not yet seen, it injects to Anonymous Layer.
func (b *Bridge) ProcessSurfaceWave(ctx context.Context, wave *pb.Wave) error {
	if !isVeiledWave(wave) {
		return nil // Not a Veiled Wave, nothing to bridge
	}
	return b.InjectToAnonymous(ctx, wave)
}

// isVeiledWave checks if a Wave is a Veiled Wave (type 0x03).
func isVeiledWave(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	return wave.WaveType == pb.WaveType_WAVE_TYPE_VEILED
}

// hasInjected checks if a Wave ID has already been injected.
func (b *Bridge) hasInjected(waveID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.injected[waveID]
	return exists
}

// markInjected records a Wave ID as injected.
func (b *Bridge) markInjected(waveID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.injected[waveID] = time.Now()
}

// CleanExpiredInjections removes old entries from the injection cache.
func (b *Bridge) CleanExpiredInjections() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-b.deduplicationTTL)
	count := 0

	for id, injectedAt := range b.injected {
		if injectedAt.Before(cutoff) {
			delete(b.injected, id)
			count++
		}
	}

	return count
}

// InjectionCacheSize returns the number of tracked injections.
func (b *Bridge) InjectionCacheSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.injected)
}

// Stats returns a copy of the bridge statistics.
func (b *Bridge) Stats() BridgeStats {
	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()
	return BridgeStats{
		InjectedToSurface:   b.stats.InjectedToSurface,
		InjectedToAnonymous: b.stats.InjectedToAnonymous,
		DuplicatesSkipped:   b.stats.DuplicatesSkipped,
		RateLimited:         b.stats.RateLimited,
		InvalidWaves:        b.stats.InvalidWaves,
	}
}

func (b *Bridge) recordSurfaceInjection() {
	b.stats.mu.Lock()
	b.stats.InjectedToSurface++
	b.stats.mu.Unlock()
}

func (b *Bridge) recordAnonymousInjection() {
	b.stats.mu.Lock()
	b.stats.InjectedToAnonymous++
	b.stats.mu.Unlock()
}

func (b *Bridge) recordDuplicate() {
	b.stats.mu.Lock()
	b.stats.DuplicatesSkipped++
	b.stats.mu.Unlock()
}

func (b *Bridge) recordRateLimited() {
	b.stats.mu.Lock()
	b.stats.RateLimited++
	b.stats.mu.Unlock()
}

func (b *Bridge) recordInvalid() {
	b.stats.mu.Lock()
	b.stats.InvalidWaves++
	b.stats.mu.Unlock()
}

// ErrRateLimited is returned when the bridge rate limit is exceeded.
var ErrRateLimited = errors.New("bridge rate limit exceeded")

// allow checks if a request is allowed under the rate limit.
func (rl *bridgeRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.lastTime = now

	// Add tokens based on elapsed time.
	rl.tokens += elapsed * rl.maxRate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}

	// Check if we have a token.
	if rl.tokens < 1 {
		return false
	}

	rl.tokens--
	return true
}

// StartCleanup runs periodic cleanup of the injection cache.
func (b *Bridge) StartCleanup(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.CleanExpiredInjections()
			}
		}
	}()

	return cancel
}

// BridgeRelay combines a standard Relay with Bridge capabilities.
// This is the complete bridge node implementation for Hybrid+ users.
type BridgeRelay struct {
	*Relay
	*Bridge
	surfacePublisher  Publisher
	anonymousPublisher Publisher
}

// NewBridgeRelay creates a relay with cross-layer bridge functionality.
func NewBridgeRelay(surfacePub, anonymousPub Publisher) *BridgeRelay {
	return &BridgeRelay{
		Relay:             NewRelay(),
		Bridge:            NewBridgeWithPublishers(surfacePub, anonymousPub),
		surfacePublisher:  surfacePub,
		anonymousPublisher: anonymousPub,
	}
}

// ReceiveFromAnonymous processes a Wave from the Anonymous Layer.
// If it's a Veiled Wave, it's bridged to the Surface Layer.
func (br *BridgeRelay) ReceiveFromAnonymous(ctx context.Context, wave *pb.Wave) error {
	// Standard relay processing.
	_, err := br.Relay.Receive(wave)
	if err != nil {
		return err
	}

	// Bridge Veiled Waves to Surface.
	if isVeiledWave(wave) {
		return br.Bridge.InjectToSurface(ctx, wave)
	}

	return nil
}

// ReceiveFromSurface processes a Wave from the Surface Layer.
// If it's a Veiled Wave not yet bridged, it's bridged to Anonymous.
func (br *BridgeRelay) ReceiveFromSurface(ctx context.Context, wave *pb.Wave) error {
	// Standard relay processing.
	_, err := br.Relay.Receive(wave)
	if err != nil {
		return err
	}

	// Bridge Veiled Waves to Anonymous.
	if isVeiledWave(wave) {
		return br.Bridge.InjectToAnonymous(ctx, wave)
	}

	return nil
}
