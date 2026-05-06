package propagation

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkWavePropagationLatency measures latency tracking for 3-hop propagation.
// Per TECHNICAL_IMPLEMENTATION.md, target is <500ms for 3-hop delivery.
func BenchmarkWavePropagationLatency(b *testing.B) {
	metrics := NewPropagationMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Track 3-hop propagation.
		waveID := fmt.Sprintf("wave-%d", i)
		for hop := 0; hop < 3; hop++ {
			hopLatency := time.Since(start)
			metrics.RecordWaveHop(waveID, hopLatency)
		}
	}
}

// BenchmarkLatencyTrackerRecordHop measures latency tracking overhead.
func BenchmarkLatencyTrackerRecordHop(b *testing.B) {
	metrics := NewPropagationMetrics()
	latency := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordHopLatency(latency)
	}
}

// BenchmarkLatencyTrackerRecordWaveHop measures per-wave tracking overhead.
func BenchmarkLatencyTrackerRecordWaveHop(b *testing.B) {
	metrics := NewPropagationMetrics()
	latency := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		waveID := fmt.Sprintf("wave-%c", rune('a'+(i%26)))
		metrics.RecordWaveHop(waveID, latency)
	}
}

// BenchmarkLatencyStats measures statistics computation overhead.
func BenchmarkLatencyStats(b *testing.B) {
	metrics := NewPropagationMetrics()

	// Populate with sample data.
	for i := 0; i < 1000; i++ {
		metrics.RecordHopLatency(10 * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.Stats()
	}
}

// BenchmarkThreeHopLatencies measures latency extraction performance.
func BenchmarkThreeHopLatencies(b *testing.B) {
	metrics := NewPropagationMetrics()

	// Populate with sample data for 100 waves.
	for i := 0; i < 100; i++ {
		waveID := fmt.Sprintf("wave-%d", i)
		for hop := 0; hop < 5; hop++ {
			metrics.RecordWaveHop(waveID, 10*time.Millisecond)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.ThreeHopLatencies()
	}
}

// BenchmarkThreeHopPropagationTracking measures realistic 3-hop latency tracking.
func BenchmarkThreeHopPropagationTracking(b *testing.B) {
	metrics := NewPropagationMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		waveID := fmt.Sprintf("wave-%d", i)

		// Simulate propagation through 3 hops with realistic inter-hop delays.
		for hopIdx := 0; hopIdx < 3; hopIdx++ {
			// Simulate processing delay for each hop (50-150µs typical)
			time.Sleep(100 * time.Microsecond)

			hopLatency := time.Since(start)
			metrics.RecordWaveHop(waveID, hopLatency)
		}

		// Verify 3-hop latency is tracked correctly.
		threeHopLatency := metrics.GetWaveLatency(waveID, 3)
		if threeHopLatency == 0 {
			b.Fatal("3-hop latency not recorded")
		}
	}
}
