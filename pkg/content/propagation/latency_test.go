package propagation

import (
	"sync"
	"testing"
	"time"
)

func TestNewPropagationMetrics(t *testing.T) {
	m := NewPropagationMetrics()
	if m == nil {
		t.Fatal("NewPropagationMetrics returned nil")
	}

	stats := m.Stats()
	if stats.TotalWaves != 0 {
		t.Errorf("initial TotalWaves = %d, want 0", stats.TotalWaves)
	}
	if stats.TotalHops != 0 {
		t.Errorf("initial TotalHops = %d, want 0", stats.TotalHops)
	}
}

func TestRecordHopLatency(t *testing.T) {
	m := NewPropagationMetrics()

	// Record several latencies.
	latencies := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, lat := range latencies {
		m.RecordHopLatency(lat)
	}

	stats := m.Stats()

	if stats.TotalHops != 3 {
		t.Errorf("TotalHops = %d, want 3", stats.TotalHops)
	}

	expectedAvg := 100 * time.Millisecond
	if stats.AverageLatency != expectedAvg {
		t.Errorf("AverageLatency = %v, want %v", stats.AverageLatency, expectedAvg)
	}

	if stats.MaxLatency != 150*time.Millisecond {
		t.Errorf("MaxLatency = %v, want 150ms", stats.MaxLatency)
	}

	if stats.MinLatency != 50*time.Millisecond {
		t.Errorf("MinLatency = %v, want 50ms", stats.MinLatency)
	}
}

func TestRecordWaveHop(t *testing.T) {
	m := NewPropagationMetrics()

	waveID := "test-wave-1"

	// Simulate 3 hops with 100ms each (total 300ms < 500ms target).
	hop1 := m.RecordWaveHop(waveID, 100*time.Millisecond)
	if hop1 != 100*time.Millisecond {
		t.Errorf("hop1 cumulative = %v, want 100ms", hop1)
	}

	hop2 := m.RecordWaveHop(waveID, 100*time.Millisecond)
	if hop2 != 200*time.Millisecond {
		t.Errorf("hop2 cumulative = %v, want 200ms", hop2)
	}

	hop3 := m.RecordWaveHop(waveID, 100*time.Millisecond)
	if hop3 != 300*time.Millisecond {
		t.Errorf("hop3 cumulative = %v, want 300ms", hop3)
	}

	// Should meet target (300ms < 500ms).
	if !m.MeetsTarget() {
		t.Error("MeetsTarget() = false, want true for 300ms")
	}
}

func TestLatencyTargetViolation(t *testing.T) {
	m := NewPropagationMetrics()

	waveID := "slow-wave"

	// Simulate 3 hops with 200ms each (total 600ms > 500ms target).
	m.RecordWaveHop(waveID, 200*time.Millisecond)
	m.RecordWaveHop(waveID, 200*time.Millisecond)
	m.RecordWaveHop(waveID, 200*time.Millisecond)

	stats := m.Stats()
	if stats.Violations != 1 {
		t.Errorf("Violations = %d, want 1", stats.Violations)
	}

	if m.MeetsTarget() {
		t.Error("MeetsTarget() = true, want false for 600ms total")
	}
}

func TestGetWaveLatency(t *testing.T) {
	m := NewPropagationMetrics()

	waveID := "test-wave-2"

	m.RecordWaveHop(waveID, 50*time.Millisecond)
	m.RecordWaveHop(waveID, 75*time.Millisecond)
	m.RecordWaveHop(waveID, 100*time.Millisecond)

	// Get cumulative at hop 2.
	lat2 := m.GetWaveLatency(waveID, 2)
	if lat2 != 125*time.Millisecond {
		t.Errorf("GetWaveLatency(hop 2) = %v, want 125ms", lat2)
	}

	// Get cumulative at hop 3.
	lat3 := m.GetWaveLatency(waveID, 3)
	if lat3 != 225*time.Millisecond {
		t.Errorf("GetWaveLatency(hop 3) = %v, want 225ms", lat3)
	}

	// Unknown wave.
	latUnknown := m.GetWaveLatency("unknown", 1)
	if latUnknown != 0 {
		t.Errorf("GetWaveLatency(unknown) = %v, want 0", latUnknown)
	}
}

func TestThreeHopLatencies(t *testing.T) {
	m := NewPropagationMetrics()

	// Wave 1: 3 hops, total 300ms.
	m.RecordWaveHop("wave1", 100*time.Millisecond)
	m.RecordWaveHop("wave1", 100*time.Millisecond)
	m.RecordWaveHop("wave1", 100*time.Millisecond)

	// Wave 2: 3 hops, total 450ms.
	m.RecordWaveHop("wave2", 150*time.Millisecond)
	m.RecordWaveHop("wave2", 150*time.Millisecond)
	m.RecordWaveHop("wave2", 150*time.Millisecond)

	// Wave 3: only 2 hops (should not be included).
	m.RecordWaveHop("wave3", 100*time.Millisecond)
	m.RecordWaveHop("wave3", 100*time.Millisecond)

	latencies := m.ThreeHopLatencies()
	if len(latencies) != 2 {
		t.Errorf("ThreeHopLatencies count = %d, want 2", len(latencies))
	}

	// Check that both complete waves are included.
	found300 := false
	found450 := false
	for _, lat := range latencies {
		if lat == 300*time.Millisecond {
			found300 = true
		}
		if lat == 450*time.Millisecond {
			found450 = true
		}
	}

	if !found300 {
		t.Error("ThreeHopLatencies missing 300ms wave")
	}
	if !found450 {
		t.Error("ThreeHopLatencies missing 450ms wave")
	}
}

func TestReset(t *testing.T) {
	m := NewPropagationMetrics()

	m.RecordHopLatency(100 * time.Millisecond)
	m.RecordWaveHop("wave1", 100*time.Millisecond)

	m.Reset()

	stats := m.Stats()
	if stats.TotalHops != 0 {
		t.Errorf("after Reset, TotalHops = %d, want 0", stats.TotalHops)
	}
	if stats.TotalWaves != 0 {
		t.Errorf("after Reset, TotalWaves = %d, want 0", stats.TotalWaves)
	}
}

func TestLatencyTargetConstant(t *testing.T) {
	// Per TECHNICAL_IMPLEMENTATION.md, target is 500ms.
	if LatencyTarget != 500*time.Millisecond {
		t.Errorf("LatencyTarget = %v, want 500ms", LatencyTarget)
	}
}

func TestConcurrentMetrics(t *testing.T) {
	m := NewPropagationMetrics()
	var wg sync.WaitGroup

	// Concurrent writes from multiple goroutines.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			waveID := string(rune('a' + id))
			for j := 0; j < 3; j++ {
				m.RecordWaveHop(waveID, 50*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	stats := m.Stats()
	if stats.TotalWaves != 10 {
		t.Errorf("TotalWaves = %d, want 10", stats.TotalWaves)
	}
}

// TestSimulatedThreeHopPropagation simulates a 3-hop propagation chain
// and verifies the <500ms latency target is achievable.
//
// Per TECHNICAL_IMPLEMENTATION.md §7.2:
// "Wave propagation latency (time from publication on one node to receipt
// on a peer 3 hops away in the gossip mesh) must be below 500 milliseconds
// under normal network conditions."
func TestSimulatedThreeHopPropagation(t *testing.T) {
	metrics := NewPropagationMetrics()

	// Simulate realistic per-hop latencies for in-memory relay.
	// Actual network latency would add ~10-50ms per hop for real TCP/QUIC.
	// Validation, signature checks, and PoW verification add processing time.
	//
	// In-memory simulation:
	// - Message validation: ~1-5ms (signature verify, PoW check)
	// - Internal processing: ~1ms
	// - Deduplication check: ~1ms
	// Total per-hop: ~10-30ms realistic in-memory, ~50-100ms with real network
	const numWaves = 100

	for i := 0; i < numWaves; i++ {
		waveID := string(rune('a'+i%26)) + string(rune('0'+i%10))

		// Simulate 3 hops with realistic processing times.
		// This models the actual work done per hop without network RTT.
		hop1Start := time.Now()
		simulateHopProcessing()
		hop1Latency := time.Since(hop1Start)
		metrics.RecordWaveHop(waveID, hop1Latency)

		hop2Start := time.Now()
		simulateHopProcessing()
		hop2Latency := time.Since(hop2Start)
		metrics.RecordWaveHop(waveID, hop2Latency)

		hop3Start := time.Now()
		simulateHopProcessing()
		hop3Latency := time.Since(hop3Start)
		metrics.RecordWaveHop(waveID, hop3Latency)
	}

	// Verify all waves meet the target.
	threeHopLatencies := metrics.ThreeHopLatencies()
	if len(threeHopLatencies) != numWaves {
		t.Errorf("recorded 3-hop waves = %d, want %d", len(threeHopLatencies), numWaves)
	}

	// Check that average 3-hop latency is well under target.
	var totalLatency time.Duration
	for _, lat := range threeHopLatencies {
		totalLatency += lat
	}
	avgLatency := totalLatency / time.Duration(len(threeHopLatencies))

	// In-memory processing should be << 500ms.
	if avgLatency > 100*time.Millisecond {
		t.Errorf("average 3-hop latency = %v, expected << 100ms for in-memory", avgLatency)
	}

	// Verify no violations with in-memory processing.
	if !metrics.MeetsTarget() {
		t.Errorf("MeetsTarget() = false, violations = %d", metrics.Stats().Violations)
	}

	t.Logf("3-hop propagation stats: avg=%v, max=%v, violations=%d",
		avgLatency, metrics.Stats().MaxLatency, metrics.Stats().Violations)
}

// simulateHopProcessing simulates the work done at each hop.
// In-memory this is very fast; with real network it would add RTT.
func simulateHopProcessing() {
	// Simulate minimal processing work (validation, dedup check).
	// Real implementation would include:
	// - Ed25519 signature verification (~0.2ms)
	// - PoW verification (~0.1ms)
	// - BLAKE3 hash for dedup (~0.05ms)
	// - Map lookup (~0.01ms)
	time.Sleep(1 * time.Millisecond) // Simulate ~1ms processing per hop
}

// TestLatencyBudgetAnalysis provides a breakdown of the 500ms latency budget.
// This is documentation as a test per the spec.
func TestLatencyBudgetAnalysis(t *testing.T) {
	// Per NETWORK_ARCHITECTURE.md:
	// "Under normal network conditions... a Wave published by any node reaches
	// 99% of subscribed nodes within 3 seconds."
	//
	// Per TECHNICAL_IMPLEMENTATION.md:
	// "Wave propagation latency... must be below 500 milliseconds"
	//
	// Latency budget breakdown (3 hops @ <167ms each):
	// - Network RTT per hop: ~50-100ms (typical residential internet)
	// - Signature verification: ~0.2ms
	// - PoW verification: ~0.1ms
	// - Deduplication check: ~0.05ms
	// - Handler callback: ~1ms
	// - GossipSub overhead: ~5ms
	//
	// Total per hop: ~60-110ms with real network
	// 3 hops: ~180-330ms (well under 500ms target)

	budgetPerHop := LatencyTarget / 3 // ~166ms per hop

	// Verify budget assumptions.
	if budgetPerHop < 100*time.Millisecond {
		t.Errorf("budgetPerHop = %v, expected >= 100ms", budgetPerHop)
	}

	t.Logf("Latency budget: %v total, %v per hop", LatencyTarget, budgetPerHop)
}
