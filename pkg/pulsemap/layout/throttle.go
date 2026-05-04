// Package layout - Data update throttling for performance optimization.
// Per ROADMAP.md line 595: "Data update throttling — 30Hz nodes, 10Hz state,
// 5Hz connections, 2Hz content".
// Per PULSE_MAP.md: Different data types update at different frequencies
// to balance responsiveness with computational cost.
package layout

import (
	"sync"
	"time"
)

// UpdateCategory represents different types of data that can be updated.
type UpdateCategory int

const (
	// UpdateNodes is position/force updates (30Hz, ~33ms).
	UpdateNodes UpdateCategory = iota
	// UpdateState is node state changes like activity, mode (10Hz, 100ms).
	UpdateState
	// UpdateConnections is edge additions/removals (5Hz, 200ms).
	UpdateConnections
	// UpdateContent is content data like wave counts (2Hz, 500ms).
	UpdateContent
)

// Default update intervals per ROADMAP.md specification.
const (
	DefaultNodeInterval       = time.Second / 30 // 30Hz = ~33ms
	DefaultStateInterval      = time.Second / 10 // 10Hz = 100ms
	DefaultConnectionInterval = time.Second / 5  // 5Hz = 200ms
	DefaultContentInterval    = time.Second / 2  // 2Hz = 500ms
)

// ThrottleConfig holds interval settings for each update category.
type ThrottleConfig struct {
	NodeInterval       time.Duration
	StateInterval      time.Duration
	ConnectionInterval time.Duration
	ContentInterval    time.Duration
}

// DefaultThrottleConfig returns the default throttle configuration.
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		NodeInterval:       DefaultNodeInterval,
		StateInterval:      DefaultStateInterval,
		ConnectionInterval: DefaultConnectionInterval,
		ContentInterval:    DefaultContentInterval,
	}
}

// UpdateThrottler manages rate-limited updates for different data categories.
type UpdateThrottler struct {
	mu sync.RWMutex

	config ThrottleConfig

	// Last update times per category.
	lastUpdate map[UpdateCategory]time.Time

	// Pending update flags.
	pending map[UpdateCategory]bool

	// Update counters for statistics.
	updateCounts map[UpdateCategory]int64
	dropCounts   map[UpdateCategory]int64

	// Optional callbacks when updates are allowed.
	callbacks map[UpdateCategory]func()
}

// NewUpdateThrottler creates a new update throttler with default config.
func NewUpdateThrottler() *UpdateThrottler {
	return &UpdateThrottler{
		config:       DefaultThrottleConfig(),
		lastUpdate:   make(map[UpdateCategory]time.Time),
		pending:      make(map[UpdateCategory]bool),
		updateCounts: make(map[UpdateCategory]int64),
		dropCounts:   make(map[UpdateCategory]int64),
		callbacks:    make(map[UpdateCategory]func()),
	}
}

// NewUpdateThrottlerWithConfig creates a throttler with custom config.
func NewUpdateThrottlerWithConfig(config ThrottleConfig) *UpdateThrottler {
	t := NewUpdateThrottler()
	t.config = config
	return t
}

// SetConfig updates the throttle configuration.
func (t *UpdateThrottler) SetConfig(config ThrottleConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.config = config
}

// GetConfig returns the current configuration.
func (t *UpdateThrottler) GetConfig() ThrottleConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config
}

// SetInterval sets the interval for a specific category.
func (t *UpdateThrottler) SetInterval(category UpdateCategory, interval time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch category {
	case UpdateNodes:
		t.config.NodeInterval = interval
	case UpdateState:
		t.config.StateInterval = interval
	case UpdateConnections:
		t.config.ConnectionInterval = interval
	case UpdateContent:
		t.config.ContentInterval = interval
	}
}

// GetInterval returns the interval for a category.
func (t *UpdateThrottler) GetInterval(category UpdateCategory) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.getIntervalLocked(category)
}

func (t *UpdateThrottler) getIntervalLocked(category UpdateCategory) time.Duration {
	switch category {
	case UpdateNodes:
		return t.config.NodeInterval
	case UpdateState:
		return t.config.StateInterval
	case UpdateConnections:
		return t.config.ConnectionInterval
	case UpdateContent:
		return t.config.ContentInterval
	default:
		return DefaultNodeInterval
	}
}

// ShouldUpdate checks if an update is allowed for the given category.
// If allowed, records the update time. If not allowed, increments drop count.
func (t *UpdateThrottler) ShouldUpdate(category UpdateCategory) bool {
	return t.ShouldUpdateNow(category, time.Now())
}

// ShouldUpdateNow is like ShouldUpdate but takes the current time as parameter.
// Useful for batch checking multiple categories at the same instant.
func (t *UpdateThrottler) ShouldUpdateNow(category UpdateCategory, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	interval := t.getIntervalLocked(category)
	last := t.lastUpdate[category]

	if now.Sub(last) >= interval {
		t.lastUpdate[category] = now
		t.updateCounts[category]++
		t.pending[category] = false
		return true
	}

	t.dropCounts[category]++
	t.pending[category] = true
	return false
}

// ForceUpdate forces an update for a category regardless of timing.
func (t *UpdateThrottler) ForceUpdate(category UpdateCategory) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastUpdate[category] = time.Now()
	t.updateCounts[category]++
	t.pending[category] = false
}

// TimeUntilUpdate returns how long until the next update is allowed.
// Returns 0 if update is allowed now.
func (t *UpdateThrottler) TimeUntilUpdate(category UpdateCategory) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	interval := t.getIntervalLocked(category)
	last := t.lastUpdate[category]
	elapsed := now.Sub(last)

	if elapsed >= interval {
		return 0
	}
	return interval - elapsed
}

// IsPending returns whether an update is pending (was throttled).
func (t *UpdateThrottler) IsPending(category UpdateCategory) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pending[category]
}

// SetCallback sets a callback to be invoked when updates are triggered.
func (t *UpdateThrottler) SetCallback(category UpdateCategory, callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callbacks[category] = callback
}

// TriggerIfReady checks if update is allowed and invokes callback if set.
// Returns true if update was triggered.
func (t *UpdateThrottler) TriggerIfReady(category UpdateCategory) bool {
	if !t.ShouldUpdate(category) {
		return false
	}

	t.mu.RLock()
	callback := t.callbacks[category]
	t.mu.RUnlock()

	if callback != nil {
		callback()
	}
	return true
}

// ThrottleStats contains statistics about throttling behavior.
type ThrottleStats struct {
	Category    UpdateCategory
	UpdateCount int64
	DropCount   int64
	TotalCount  int64
	DropRatio   float64
	Interval    time.Duration
	IsPending   bool
	TimeToNext  time.Duration
}

// GetStats returns statistics for a category.
func (t *UpdateThrottler) GetStats(category UpdateCategory) ThrottleStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	updates := t.updateCounts[category]
	drops := t.dropCounts[category]
	total := updates + drops

	ratio := 0.0
	if total > 0 {
		ratio = float64(drops) / float64(total)
	}

	now := time.Now()
	interval := t.getIntervalLocked(category)
	last := t.lastUpdate[category]
	timeToNext := interval - now.Sub(last)
	if timeToNext < 0 {
		timeToNext = 0
	}

	return ThrottleStats{
		Category:    category,
		UpdateCount: updates,
		DropCount:   drops,
		TotalCount:  total,
		DropRatio:   ratio,
		Interval:    interval,
		IsPending:   t.pending[category],
		TimeToNext:  timeToNext,
	}
}

// GetAllStats returns statistics for all categories.
func (t *UpdateThrottler) GetAllStats() []ThrottleStats {
	categories := []UpdateCategory{
		UpdateNodes,
		UpdateState,
		UpdateConnections,
		UpdateContent,
	}

	stats := make([]ThrottleStats, len(categories))
	for i, cat := range categories {
		stats[i] = t.GetStats(cat)
	}
	return stats
}

// Reset clears all timing and counter state.
func (t *UpdateThrottler) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastUpdate = make(map[UpdateCategory]time.Time)
	t.pending = make(map[UpdateCategory]bool)
	t.updateCounts = make(map[UpdateCategory]int64)
	t.dropCounts = make(map[UpdateCategory]int64)
}

// ResetCounters clears only the counters, preserving timing state.
func (t *UpdateThrottler) ResetCounters() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.updateCounts = make(map[UpdateCategory]int64)
	t.dropCounts = make(map[UpdateCategory]int64)
}

// CategoryName returns the string name of an update category.
func CategoryName(category UpdateCategory) string {
	switch category {
	case UpdateNodes:
		return "Nodes"
	case UpdateState:
		return "State"
	case UpdateConnections:
		return "Connections"
	case UpdateContent:
		return "Content"
	default:
		return "Unknown"
	}
}

// BatchThrottler provides batch update checking for multiple categories.
type BatchThrottler struct {
	throttler *UpdateThrottler
}

// NewBatchThrottler creates a batch throttler wrapping an UpdateThrottler.
func NewBatchThrottler(throttler *UpdateThrottler) *BatchThrottler {
	return &BatchThrottler{throttler: throttler}
}

// CheckAll checks all categories at once and returns which are ready.
type BatchResult struct {
	NodesReady       bool
	StateReady       bool
	ConnectionsReady bool
	ContentReady     bool
	Timestamp        time.Time
}

// CheckAll checks all categories at the same instant.
func (bt *BatchThrottler) CheckAll() BatchResult {
	now := time.Now()
	return BatchResult{
		NodesReady:       bt.throttler.ShouldUpdateNow(UpdateNodes, now),
		StateReady:       bt.throttler.ShouldUpdateNow(UpdateState, now),
		ConnectionsReady: bt.throttler.ShouldUpdateNow(UpdateConnections, now),
		ContentReady:     bt.throttler.ShouldUpdateNow(UpdateContent, now),
		Timestamp:        now,
	}
}

// ReadyCount returns how many categories are ready in the result.
func (br BatchResult) ReadyCount() int {
	count := 0
	if br.NodesReady {
		count++
	}
	if br.StateReady {
		count++
	}
	if br.ConnectionsReady {
		count++
	}
	if br.ContentReady {
		count++
	}
	return count
}

// AnyReady returns true if any category is ready.
func (br BatchResult) AnyReady() bool {
	return br.NodesReady || br.StateReady || br.ConnectionsReady || br.ContentReady
}

// ThrottledUpdater wraps an UpdateThrottler with convenient update methods.
type ThrottledUpdater struct {
	throttler *UpdateThrottler
	mu        sync.RWMutex

	// Pending data to be applied on next allowed update.
	pendingNodes       func()
	pendingState       func()
	pendingConnections func()
	pendingContent     func()
}

// NewThrottledUpdater creates a new throttled updater.
func NewThrottledUpdater() *ThrottledUpdater {
	return &ThrottledUpdater{
		throttler: NewUpdateThrottler(),
	}
}

// Throttler returns the underlying UpdateThrottler.
func (tu *ThrottledUpdater) Throttler() *UpdateThrottler {
	return tu.throttler
}

// QueueNodeUpdate queues a node update to run when throttle allows.
func (tu *ThrottledUpdater) QueueNodeUpdate(update func()) {
	tu.mu.Lock()
	tu.pendingNodes = update
	tu.mu.Unlock()
}

// QueueStateUpdate queues a state update.
func (tu *ThrottledUpdater) QueueStateUpdate(update func()) {
	tu.mu.Lock()
	tu.pendingState = update
	tu.mu.Unlock()
}

// QueueConnectionUpdate queues a connection update.
func (tu *ThrottledUpdater) QueueConnectionUpdate(update func()) {
	tu.mu.Lock()
	tu.pendingConnections = update
	tu.mu.Unlock()
}

// QueueContentUpdate queues a content update.
func (tu *ThrottledUpdater) QueueContentUpdate(update func()) {
	tu.mu.Lock()
	tu.pendingContent = update
	tu.mu.Unlock()
}

// ProcessUpdates checks all categories and runs pending updates if allowed.
// Returns which categories were updated.
func (tu *ThrottledUpdater) ProcessUpdates() BatchResult {
	now := time.Now()
	result := BatchResult{Timestamp: now}

	tu.mu.Lock()
	defer tu.mu.Unlock()

	if tu.pendingNodes != nil && tu.throttler.ShouldUpdateNow(UpdateNodes, now) {
		tu.pendingNodes()
		tu.pendingNodes = nil
		result.NodesReady = true
	}

	if tu.pendingState != nil && tu.throttler.ShouldUpdateNow(UpdateState, now) {
		tu.pendingState()
		tu.pendingState = nil
		result.StateReady = true
	}

	if tu.pendingConnections != nil && tu.throttler.ShouldUpdateNow(UpdateConnections, now) {
		tu.pendingConnections()
		tu.pendingConnections = nil
		result.ConnectionsReady = true
	}

	if tu.pendingContent != nil && tu.throttler.ShouldUpdateNow(UpdateContent, now) {
		tu.pendingContent()
		tu.pendingContent = nil
		result.ContentReady = true
	}

	return result
}

// HasPending returns whether any updates are pending.
func (tu *ThrottledUpdater) HasPending() bool {
	tu.mu.RLock()
	defer tu.mu.RUnlock()
	return tu.pendingNodes != nil || tu.pendingState != nil ||
		tu.pendingConnections != nil || tu.pendingContent != nil
}
