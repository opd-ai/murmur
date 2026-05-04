// Package layout provides the force-directed graph engine for the Pulse Map.
// This file implements performance benchmarks per TECHNICAL_IMPLEMENTATION.md.
package layout

import (
	"fmt"
	"testing"
)

// BenchmarkStep measures a single layout step with 100 nodes.
// Per TECHNICAL_IMPLEMENTATION.md, target is <16ms (60fps).
func BenchmarkStep(b *testing.B) {
	engine := NewEngine()

	// Add 100 nodes
	for i := 0; i < 100; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 5,
			Activity:    0.5,
		})
	}

	// Add some edges
	for i := 0; i < 100; i++ {
		for j := 0; j < 3; j++ {
			target := (i + j + 1) % 100
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkStep500Nodes measures layout step with 500 nodes (Barnes-Hut threshold).
// Per PULSE_MAP.md, Barnes-Hut optimization kicks in at 500+ nodes.
func BenchmarkStep500Nodes(b *testing.B) {
	engine := NewEngine()

	// Add 500 nodes
	for i := 0; i < 500; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 6,
			Activity:    0.5,
		})
	}

	// Add mesh-like edges (each node connects to ~6 neighbors)
	for i := 0; i < 500; i++ {
		for j := 0; j < 6; j++ {
			target := (i + j + 1) % 500
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkStep1000Nodes measures layout step with 1000 nodes (large graph).
func BenchmarkStep1000Nodes(b *testing.B) {
	engine := NewEngine()

	// Add 1000 nodes
	for i := 0; i < 1000; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 8,
			Activity:    0.5,
		})
	}

	// Add edges
	for i := 0; i < 1000; i++ {
		for j := 0; j < 8; j++ {
			target := (i + j + 1) % 1000
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkAddNode measures node addition performance.
func BenchmarkAddNode(b *testing.B) {
	engine := NewEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 5,
			Activity:    0.5,
		})
	}
}

// BenchmarkAddEdge measures edge addition performance.
func BenchmarkAddEdge(b *testing.B) {
	engine := NewEngine()

	// Pre-populate nodes
	for i := 0; i < 100; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 5,
			Activity:    0.5,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source := i % 100
		target := (i + 1) % 100
		engine.AddEdge(Edge{
			SourceID: fmt.Sprintf("node-%d", source),
			TargetID: fmt.Sprintf("node-%d", target),
			Age:      1.0,
		})
	}
}

// BenchmarkPositionBufferSwap measures atomic position buffer swap performance.
func BenchmarkPositionBufferSwap(b *testing.B) {
	pb := NewPositionBuffer()
	newPositions := map[string]Position{
		"node-1": {X: 100, Y: 100, VX: 1, VY: 1},
		"node-2": {X: 200, Y: 200, VX: 2, VY: 2},
		"node-3": {X: 300, Y: 300, VX: 3, VY: 3},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.Swap(newPositions)
	}
}

// BenchmarkPositionBufferGet measures lock-free read performance.
func BenchmarkPositionBufferGet(b *testing.B) {
	pb := NewPositionBuffer()
	positions := map[string]Position{
		"node-1": {X: 100, Y: 100, VX: 1, VY: 1},
		"node-2": {X: 200, Y: 200, VX: 2, VY: 2},
		"node-3": {X: 300, Y: 300, VX: 3, VY: 3},
	}
	pb.Swap(positions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.Get()
	}
}

// BenchmarkStepSparseGraph measures layout with sparse connections (2 edges/node).
func BenchmarkStepSparseGraph(b *testing.B) {
	engine := NewEngine()

	// Add 500 nodes with sparse connections
	for i := 0; i < 500; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 2,
			Activity:    0.5,
		})
	}

	// Add only 2 edges per node
	for i := 0; i < 500; i++ {
		for j := 0; j < 2; j++ {
			target := (i + j*10 + 1) % 500
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkStepDenseGraph measures layout with dense connections (20 edges/node).
func BenchmarkStepDenseGraph(b *testing.B) {
	engine := NewEngine()

	// Add 100 nodes with dense connections
	for i := 0; i < 100; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 20,
			Activity:    0.5,
		})
	}

	// Add 20 edges per node
	for i := 0; i < 100; i++ {
		for j := 0; j < 20; j++ {
			target := (i + j + 1) % 100
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}

// BenchmarkStep500Nodes2000Edges validates ROADMAP.md line 593 requirement:
// 60fps (16.67ms/frame) with 500 nodes and 2000 edges.
func BenchmarkStep500Nodes2000Edges(b *testing.B) {
	engine := NewEngine()

	// Add 500 nodes
	for i := 0; i < 500; i++ {
		engine.AddNode(&Node{
			ID:          fmt.Sprintf("node-%d", i),
			Connections: 4,
			Activity:    0.5,
		})
	}

	// Add 2000 edges (4 edges per node)
	for i := 0; i < 500; i++ {
		for j := 0; j < 4; j++ {
			target := (i + j + 1) % 500
			engine.AddEdge(Edge{
				SourceID: fmt.Sprintf("node-%d", i),
				TargetID: fmt.Sprintf("node-%d", target),
				Age:      1.0,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Tick()
	}
}
