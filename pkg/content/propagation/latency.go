// Package propagation provides gossip relay logic, hop counting, and deduplication.
// This file implements propagation latency tracking per TECHNICAL_IMPLEMENTATION.md §7.2.
//
// Per TECHNICAL_IMPLEMENTATION.md: "Wave propagation latency (time from publication
// on one node to receipt on a peer 3 hops away in the gossip mesh) must be below
// 500 milliseconds under normal network conditions."
package propagation

import (
	"sync"
	"time"
)

// LatencyTarget is the target propagation latency for 3-hop delivery.
// Per TECHNICAL_IMPLEMENTATION.md, this must be below 500ms.
const LatencyTarget = 500 * time.Millisecond

// PropagationMetrics tracks latency statistics for Wave propagation.
// Thread-safe for concurrent access from multiple goroutines.
type PropagationMetrics struct {
	mu sync.RWMutex

	// Per-hop latency tracking.
	hopLatencies []time.Duration

	// Per-wave end-to-end latency (publication to receipt at each hop).
	waveLatencies map[string][]time.Duration

	// Statistics.
	totalWaves int64
	totalHops  int64
	sumLatency time.Duration
	maxLatency time.Duration
	minLatency time.Duration
	violations int64 // Count of waves exceeding LatencyTarget at 3+ hops
}

// NewPropagationMetrics creates a new metrics tracker.
func NewPropagationMetrics() *PropagationMetrics {
	return &PropagationMetrics{
		hopLatencies:  make([]time.Duration, 0, 1000),
		waveLatencies: make(map[string][]time.Duration),
		minLatency:    time.Duration(1<<63 - 1), // Max int64 duration
	}
}

// RecordHopLatency records the latency for a single hop.
func (m *PropagationMetrics) RecordHopLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hopLatencies = append(m.hopLatencies, latency)
	m.totalHops++
	m.sumLatency += latency

	if latency > m.maxLatency {
		m.maxLatency = latency
	}
	if latency < m.minLatency {
		m.minLatency = latency
	}
}

// RecordWaveHop records a hop latency for a specific wave by ID.
// Returns the cumulative latency for all hops of this wave.
func (m *PropagationMetrics) RecordWaveHop(waveID string, hopLatency time.Duration) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	latencies, exists := m.waveLatencies[waveID]
	if !exists {
		latencies = make([]time.Duration, 0, 20) // Pre-allocate for max hops
		m.totalWaves++
	}

	latencies = append(latencies, hopLatency)
	m.waveLatencies[waveID] = latencies

	// Calculate cumulative latency.
	var cumulative time.Duration
	for _, l := range latencies {
		cumulative += l
	}

	// Check if this is hop 3+ and exceeds target.
	if len(latencies) >= 3 && cumulative > LatencyTarget {
		m.violations++
	}

	return cumulative
}

// GetWaveLatency returns the cumulative latency for a wave at a given hop count.
// Returns 0 if the wave hasn't been recorded or hasn't reached that hop.
func (m *PropagationMetrics) GetWaveLatency(waveID string, hopCount int) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	latencies, exists := m.waveLatencies[waveID]
	if !exists || len(latencies) < hopCount {
		return 0
	}

	var cumulative time.Duration
	for i := 0; i < hopCount && i < len(latencies); i++ {
		cumulative += latencies[i]
	}
	return cumulative
}

// Stats returns current propagation statistics.
type Stats struct {
	TotalWaves     int64
	TotalHops      int64
	AverageLatency time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	Violations     int64
	ViolationRate  float64
}

// Stats returns current propagation statistics.
func (m *PropagationMetrics) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		TotalWaves: m.totalWaves,
		TotalHops:  m.totalHops,
		MaxLatency: m.maxLatency,
		MinLatency: m.minLatency,
		Violations: m.violations,
	}

	if m.totalHops > 0 {
		stats.AverageLatency = m.sumLatency / time.Duration(m.totalHops)
	}

	if m.totalWaves > 0 {
		stats.ViolationRate = float64(m.violations) / float64(m.totalWaves)
	}

	// Handle case where no data recorded.
	if stats.MinLatency == time.Duration(1<<63-1) {
		stats.MinLatency = 0
	}

	return stats
}

// MeetsTarget returns true if the average 3-hop latency meets the target.
func (m *PropagationMetrics) MeetsTarget() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if we have any 3-hop violations.
	return m.violations == 0
}

// Reset clears all recorded metrics.
func (m *PropagationMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hopLatencies = make([]time.Duration, 0, 1000)
	m.waveLatencies = make(map[string][]time.Duration)
	m.totalWaves = 0
	m.totalHops = 0
	m.sumLatency = 0
	m.maxLatency = 0
	m.minLatency = time.Duration(1<<63 - 1)
	m.violations = 0
}

// ThreeHopLatencies returns a slice of 3-hop cumulative latencies for all waves
// that have reached at least 3 hops.
func (m *PropagationMetrics) ThreeHopLatencies() []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]time.Duration, 0)
	for _, latencies := range m.waveLatencies {
		if len(latencies) >= 3 {
			var cumulative time.Duration
			for i := 0; i < 3; i++ {
				cumulative += latencies[i]
			}
			result = append(result, cumulative)
		}
	}
	return result
}
