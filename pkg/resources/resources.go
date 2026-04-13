// Package resources provides resource management for MURMUR.
// Per ROADMAP.md Priority 12, this package implements memory monitoring,
// connection limit enforcement, and bandwidth throttling.
package resources

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultMaxMemoryMB is the default memory limit (256 MiB per spec).
const DefaultMaxMemoryMB = 256

// DefaultMaxConnections is the default maximum connection count.
const DefaultMaxConnections = 128

// DefaultBandwidthBytesPerSec is the default sustained bandwidth (~50 KB/s).
const DefaultBandwidthBytesPerSec = 50 * 1024

// MemoryMonitor tracks memory usage and triggers warnings.
type MemoryMonitor struct {
	maxMemoryMB   uint64
	warningThresh float64
	lastStats     runtime.MemStats
	mu            sync.RWMutex
}

// NewMemoryMonitor creates a memory monitor with the given limit.
func NewMemoryMonitor(maxMemoryMB uint64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemoryMB:   maxMemoryMB,
		warningThresh: 0.8, // Warn at 80%.
	}
}

// Check reads current memory stats and returns usage info.
func (m *MemoryMonitor) Check() MemoryStatus {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	m.mu.Lock()
	m.lastStats = stats
	m.mu.Unlock()

	usedMB := stats.Alloc / (1024 * 1024)
	limitMB := m.maxMemoryMB
	usageRatio := float64(usedMB) / float64(limitMB)

	return MemoryStatus{
		UsedMB:     usedMB,
		LimitMB:    limitMB,
		UsageRatio: usageRatio,
		Warning:    usageRatio >= m.warningThresh,
		Critical:   usageRatio >= 1.0,
	}
}

// MemoryStatus describes current memory usage.
type MemoryStatus struct {
	UsedMB     uint64
	LimitMB    uint64
	UsageRatio float64
	Warning    bool
	Critical   bool
}

// ConnectionLimiter enforces connection limits.
type ConnectionLimiter struct {
	maxConnections int32
	current        atomic.Int32
}

// NewConnectionLimiter creates a connection limiter.
func NewConnectionLimiter(max int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections: int32(max),
	}
}

// TryAcquire attempts to acquire a connection slot.
// Returns true if successful, false if limit reached.
func (l *ConnectionLimiter) TryAcquire() bool {
	for {
		current := l.current.Load()
		if current >= l.maxConnections {
			return false
		}
		if l.current.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// Release releases a connection slot.
func (l *ConnectionLimiter) Release() {
	l.current.Add(-1)
}

// Current returns the current connection count.
func (l *ConnectionLimiter) Current() int {
	return int(l.current.Load())
}

// Available returns the number of available slots.
func (l *ConnectionLimiter) Available() int {
	return int(l.maxConnections) - int(l.current.Load())
}

// Max returns the maximum connection count.
func (l *ConnectionLimiter) Max() int {
	return int(l.maxConnections)
}

// BandwidthThrottler enforces bandwidth limits using token bucket.
type BandwidthThrottler struct {
	mu          sync.Mutex
	bytesPerSec int64
	tokens      int64
	maxTokens   int64
	lastRefill  time.Time
	enabled     atomic.Bool
}

// NewBandwidthThrottler creates a bandwidth throttler.
func NewBandwidthThrottler(bytesPerSec int64) *BandwidthThrottler {
	t := &BandwidthThrottler{
		bytesPerSec: bytesPerSec,
		maxTokens:   bytesPerSec, // 1 second burst capacity.
		tokens:      bytesPerSec,
		lastRefill:  time.Now(),
	}
	t.enabled.Store(true)
	return t
}

// Enable enables bandwidth throttling.
func (t *BandwidthThrottler) Enable() {
	t.enabled.Store(true)
}

// Disable disables bandwidth throttling.
func (t *BandwidthThrottler) Disable() {
	t.enabled.Store(false)
}

// IsEnabled returns whether throttling is enabled.
func (t *BandwidthThrottler) IsEnabled() bool {
	return t.enabled.Load()
}

// Consume attempts to consume bytes from the bucket.
// Returns the number of bytes that can be consumed immediately
// and the wait duration if throttled.
func (t *BandwidthThrottler) Consume(bytes int64) (allowed int64, wait time.Duration) {
	if !t.enabled.Load() {
		return bytes, 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.refill()

	if bytes <= t.tokens {
		t.tokens -= bytes
		return bytes, 0
	}

	// Partial consumption.
	allowed = t.tokens
	needed := bytes - t.tokens
	t.tokens = 0

	// Calculate wait time for remaining bytes.
	wait = time.Duration(needed * int64(time.Second) / t.bytesPerSec)
	return allowed, wait
}

// refill adds tokens based on elapsed time (must be called with lock held).
func (t *BandwidthThrottler) refill() {
	now := time.Now()
	elapsed := now.Sub(t.lastRefill)
	t.lastRefill = now

	// Add tokens for elapsed time.
	newTokens := int64(elapsed.Seconds() * float64(t.bytesPerSec))
	t.tokens += newTokens
	if t.tokens > t.maxTokens {
		t.tokens = t.maxTokens
	}
}

// Wait waits until enough bandwidth is available.
func (t *BandwidthThrottler) Wait(bytes int64) {
	for {
		_, wait := t.Consume(bytes)
		if wait == 0 {
			return
		}
		time.Sleep(wait)
	}
}

// CurrentRate returns an estimate of current usage rate.
func (t *BandwidthThrottler) CurrentRate() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.refill()
	return t.bytesPerSec - t.tokens
}

// Manager coordinates all resource limits.
type Manager struct {
	Memory      *MemoryMonitor
	Connections *ConnectionLimiter
	Bandwidth   *BandwidthThrottler
}

// NewManager creates a resource manager with default limits.
func NewManager() *Manager {
	return &Manager{
		Memory:      NewMemoryMonitor(DefaultMaxMemoryMB),
		Connections: NewConnectionLimiter(DefaultMaxConnections),
		Bandwidth:   NewBandwidthThrottler(DefaultBandwidthBytesPerSec),
	}
}

// NewManagerWithConfig creates a resource manager with custom limits.
func NewManagerWithConfig(memoryMB uint64, maxConns int, bandwidthBps int64) *Manager {
	return &Manager{
		Memory:      NewMemoryMonitor(memoryMB),
		Connections: NewConnectionLimiter(maxConns),
		Bandwidth:   NewBandwidthThrottler(bandwidthBps),
	}
}

// Status returns the current resource status.
func (m *Manager) Status() ResourceStatus {
	return ResourceStatus{
		Memory:            m.Memory.Check(),
		ActiveConnections: m.Connections.Current(),
		MaxConnections:    m.Connections.Max(),
		BandwidthEnabled:  m.Bandwidth.IsEnabled(),
	}
}

// ResourceStatus describes overall resource usage.
type ResourceStatus struct {
	Memory            MemoryStatus
	ActiveConnections int
	MaxConnections    int
	BandwidthEnabled  bool
}
