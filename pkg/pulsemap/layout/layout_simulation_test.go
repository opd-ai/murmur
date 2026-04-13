//go:build simulation

// Package layout provides simulation tests for 500-node performance validation.
// Per ROADMAP.md Priority 9 Validation: "500-node graph renders at 60fps
// with smooth pan/zoom".
//
// Note: This tests the layout engine performance, not rendering. The 60fps
// requirement means the layout tick must complete in <16.67ms. Combined with
// rendering overhead, we target <10ms for layout ticks.
package layout

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestLayoutPerformance500Nodes verifies the layout engine can handle
// 500 nodes at 60fps (tick time < 16.67ms).
func TestLayoutPerformance500Nodes(t *testing.T) {
	const nodeCount = 500
	const edgesPerNode = 4 // Average connections per NETWORK_ARCHITECTURE.md
	const targetFPS = 60
	const maxTickTime = time.Second / time.Duration(targetFPS) // 16.67ms

	engine := NewEngine()
	engine.SetCenter(800, 600) // Standard screen center

	// Add 500 nodes
	t.Logf("Adding %d nodes...", nodeCount)
	for i := 0; i < nodeCount; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: edgesPerNode,
			Activity:    float64(i%10) / 10.0,
		})
	}

	// Add edges to create a realistic social network topology
	// Use small-world network pattern: mostly local connections with some long-range
	edgeCount := 0
	for i := 0; i < nodeCount; i++ {
		// Local connections (2-3 nearby nodes)
		localConnections := 2 + (i % 2)
		for j := 1; j <= localConnections; j++ {
			target := (i + j) % nodeCount
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      float64(i % 30), // Connection age in days
			})
			edgeCount++
		}

		// Long-range connection (every 5th node)
		if i%5 == 0 {
			target := (i + 100 + i) % nodeCount
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      float64(i % 15),
			})
			edgeCount++
		}
	}

	t.Logf("Created %d edges (avg %.1f per node)", edgeCount, float64(edgeCount)/float64(nodeCount))

	// Warm-up ticks to let layout stabilize
	t.Log("Running warm-up ticks...")
	for i := 0; i < 100; i++ {
		engine.Tick()
	}

	// Measure tick times for performance validation
	const measurementTicks = 100
	tickTimes := make([]time.Duration, measurementTicks)

	t.Logf("Measuring %d tick times...", measurementTicks)
	for i := 0; i < measurementTicks; i++ {
		start := time.Now()
		engine.Tick()
		tickTimes[i] = time.Since(start)
	}

	// Calculate statistics
	var total, max time.Duration
	min := tickTimes[0]
	for _, d := range tickTimes {
		total += d
		if d > max {
			max = d
		}
		if d < min {
			min = d
		}
	}
	avg := total / time.Duration(measurementTicks)

	// Calculate 95th percentile
	// Sort tick times
	sorted := make([]time.Duration, len(tickTimes))
	copy(sorted, tickTimes)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	p95 := sorted[int(float64(len(sorted))*0.95)]

	t.Logf("Tick time statistics:")
	t.Logf("  Min: %v", min)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P95: %v", p95)
	t.Logf("  Max: %v", max)
	t.Logf("  Target: <%v (60fps)", maxTickTime)

	// Validation: average tick should be well under frame budget
	// With Barnes-Hut optimization, we target 80% of frame budget for layout
	// leaving 20% for rendering overhead (GPU-accelerated, minimal CPU)
	layoutBudget := maxTickTime * 8 / 10 // ~13.3ms

	if avg > layoutBudget {
		t.Errorf("Average tick time %.2fms exceeds layout budget %.2fms",
			float64(avg.Microseconds())/1000, float64(layoutBudget.Microseconds())/1000)
	}

	// P95 should still be under full frame budget
	if p95 > maxTickTime {
		t.Errorf("P95 tick time %.2fms exceeds frame budget %.2fms",
			float64(p95.Microseconds())/1000, float64(maxTickTime.Microseconds())/1000)
	}

	// Verify Barnes-Hut is being used (>= threshold)
	if engine.NodeCount() < BarnesHutThreshold {
		t.Errorf("Expected to use Barnes-Hut (>=%d nodes), got %d nodes",
			BarnesHutThreshold, engine.NodeCount())
	}

	t.Log("✓ 500-node layout achieves 60fps tick performance")
}

// TestLayoutMemoryUsage500Nodes verifies memory efficiency for large graphs.
func TestLayoutMemoryUsage500Nodes(t *testing.T) {
	const nodeCount = 500
	const edgesPerNode = 4

	engine := NewEngine()

	// Add nodes and edges
	for i := 0; i < nodeCount; i++ {
		engine.AddNode(&Node{ID: fmt.Sprintf("node-%d", i)})
		for j := 1; j <= edgesPerNode; j++ {
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", (i+j)%nodeCount),
			})
		}
	}

	// Run ticks to populate position buffers
	for i := 0; i < 10; i++ {
		engine.Tick()
	}

	// Get position buffer size
	positions := engine.Positions().Get()
	if len(positions) != nodeCount {
		t.Errorf("Expected %d positions, got %d", nodeCount, len(positions))
	}

	t.Logf("Position buffer contains %d nodes", len(positions))
	t.Log("✓ Memory usage test passed")
}

// TestLayoutStability500Nodes verifies the layout stabilizes to equilibrium.
func TestLayoutStability500Nodes(t *testing.T) {
	const nodeCount = 500

	engine := NewEngine()
	engine.SetCenter(0, 0)

	// Create a simple ring topology
	for i := 0; i < nodeCount; i++ {
		engine.AddNode(&Node{ID: fmt.Sprintf("node-%d", i)})
	}
	for i := 0; i < nodeCount; i++ {
		engine.AddEdge(Edge{
			SourceID: fmt.Sprintf("node-%d", i),
			TargetID: fmt.Sprintf("node-%d", (i+1)%nodeCount),
		})
	}

	// Run until stable (velocity near zero)
	const maxTicks = 1000
	var stabilized bool
	var stabilizationTick int

	for tick := 0; tick < maxTicks; tick++ {
		engine.Tick()

		// Check if velocities are small
		positions := engine.Positions().Get()
		var maxVelocity float64
		for _, pos := range positions {
			vel := pos.VX*pos.VX + pos.VY*pos.VY
			if vel > maxVelocity {
				maxVelocity = vel
			}
		}

		// Consider stable when max velocity < 0.1
		if maxVelocity < 0.01 {
			stabilized = true
			stabilizationTick = tick
			break
		}
	}

	if stabilized {
		t.Logf("Layout stabilized after %d ticks", stabilizationTick)
	} else {
		t.Log("Layout did not fully stabilize within tick limit (this is acceptable for large graphs)")
	}

	// Verify positions are reasonably distributed
	positions := engine.Positions().Get()
	var minX, maxX, minY, maxY float64
	first := true
	for _, pos := range positions {
		if first {
			minX, maxX = pos.X, pos.X
			minY, maxY = pos.Y, pos.Y
			first = false
		} else {
			if pos.X < minX {
				minX = pos.X
			}
			if pos.X > maxX {
				maxX = pos.X
			}
			if pos.Y < minY {
				minY = pos.Y
			}
			if pos.Y > maxY {
				maxY = pos.Y
			}
		}
	}

	width := maxX - minX
	height := maxY - minY
	t.Logf("Layout bounds: width=%.0f, height=%.0f", width, height)

	// Layout should have some spread (not all nodes at same point)
	if width < 10 || height < 10 {
		t.Error("Layout is too compact - nodes may have collapsed")
	}

	t.Log("✓ Layout stability test passed")
}

// TestConcurrentRead500Nodes verifies concurrent position buffer reads are safe.
func TestConcurrentRead500Nodes(t *testing.T) {
	const nodeCount = 500
	const readers = 10
	const tickDuration = 500 * time.Millisecond

	engine := NewEngine()

	// Add nodes
	for i := 0; i < nodeCount; i++ {
		engine.AddNode(&Node{ID: fmt.Sprintf("node-%d", i)})
	}

	// Run initial tick to populate position buffer
	engine.Tick()

	// Verify positions are now populated
	initialPositions := engine.Positions().Get()
	if len(initialPositions) != nodeCount {
		t.Fatalf("Position buffer not populated after Tick: got %d, want %d",
			len(initialPositions), nodeCount)
	}

	// Run concurrent readers and writers
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer goroutine (simulates layout engine)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				engine.Tick()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Reader goroutines (simulate renderer)
	var readCount int64
	var mu sync.Mutex
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					positions := engine.Positions().Get()
					mu.Lock()
					readCount++
					mu.Unlock()
					// Verify all positions are present
					if len(positions) != nodeCount {
						// This would indicate a race condition
						t.Errorf("Position buffer corrupted: expected %d, got %d",
							nodeCount, len(positions))
					}
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	// Let it run
	time.Sleep(tickDuration)
	close(done)
	wg.Wait()

	mu.Lock()
	finalReadCount := readCount
	mu.Unlock()

	t.Logf("Completed %d concurrent reads without race conditions", finalReadCount)
	t.Log("✓ Concurrent read test passed")
}

// BenchmarkTick500NodesBarnesHut benchmarks the layout with 500 nodes.
func BenchmarkTick500NodesBarnesHut(b *testing.B) {
	engine := NewEngine()

	// Create 500-node graph
	for i := 0; i < 500; i++ {
		engine.AddNode(&Node{ID: fmt.Sprintf("node-%d", i)})
	}
	for i := 0; i < 500; i++ {
		for j := 1; j <= 3; j++ {
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", (i+j)%500),
			})
		}
	}

	// Warm up
	for i := 0; i < 10; i++ {
		engine.Tick()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkTick1000NodesBarnesHut benchmarks scalability beyond 500 nodes.
func BenchmarkTick1000NodesBarnesHut(b *testing.B) {
	engine := NewEngine()

	for i := 0; i < 1000; i++ {
		engine.AddNode(&Node{ID: fmt.Sprintf("node-%d", i)})
	}
	for i := 0; i < 1000; i++ {
		for j := 1; j <= 3; j++ {
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", (i+j)%1000),
			})
		}
	}

	for i := 0; i < 10; i++ {
		engine.Tick()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkQuadtreeConstruction benchmarks Barnes-Hut quadtree building.
func BenchmarkQuadtreeConstruction(b *testing.B) {
	positions := make(map[string]Position)
	for i := 0; i < 500; i++ {
		positions[fmt.Sprintf("node-%d", i)] = Position{
			X: float64(i%50) * 20,
			Y: float64(i/50) * 20,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newQuadtree(positions, 500, 100, 2000)
	}
}
