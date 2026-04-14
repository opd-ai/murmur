// Package layout tests verify the force-directed graph layout engine.
package layout

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", e.NodeCount())
	}
}

func TestAddRemoveNode(t *testing.T) {
	e := NewEngine()
	node := &Node{ID: "test-node", Connections: 5, Activity: 10.0}

	e.AddNode(node)
	if e.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", e.NodeCount())
	}

	e.RemoveNode("test-node")
	if e.NodeCount() != 0 {
		t.Errorf("expected 0 nodes after removal, got %d", e.NodeCount())
	}
}

func TestPositionBuffer(t *testing.T) {
	pb := NewPositionBuffer()

	// Initial buffer should be empty
	positions := pb.Get()
	if len(positions) != 0 {
		t.Errorf("expected empty positions, got %d", len(positions))
	}

	// Swap in new positions
	newPos := map[string]Position{
		"a": {X: 100, Y: 200, VX: 1, VY: 2},
		"b": {X: 300, Y: 400, VX: 3, VY: 4},
	}
	pb.Swap(newPos)

	positions = pb.Get()
	if len(positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(positions))
	}
	if positions["a"].X != 100 || positions["a"].Y != 200 {
		t.Errorf("position a mismatch: %+v", positions["a"])
	}
}

func TestTickSimulation(t *testing.T) {
	e := NewEngine()

	// Add two connected nodes
	e.AddNode(&Node{ID: "node1", Connections: 1})
	e.AddNode(&Node{ID: "node2", Connections: 1})
	e.AddEdge(Edge{SourceID: "node1", TargetID: "node2", Age: 0})

	// Get initial positions
	initialPos := e.Positions().Get()
	pos1Init := initialPos["node1"]
	pos2Init := initialPos["node2"]

	// Run simulation ticks
	for i := 0; i < 10; i++ {
		e.Tick()
	}

	// Positions should have changed
	finalPos := e.Positions().Get()
	pos1Final := finalPos["node1"]
	pos2Final := finalPos["node2"]

	// At least one position should have moved
	if pos1Init.X == pos1Final.X && pos1Init.Y == pos1Final.Y &&
		pos2Init.X == pos2Final.X && pos2Init.Y == pos2Final.Y {
		t.Error("expected positions to change after simulation")
	}
}

func TestSpringAttraction(t *testing.T) {
	e := NewEngine()
	params := DefaultParams()
	params.GravityConstant = 0 // Disable gravity for this test
	e.SetParams(params)

	// Add two distant connected nodes
	e.SetCenter(0, 0)
	e.AddNode(&Node{ID: "a"})
	e.AddNode(&Node{ID: "b"})

	// Manually set positions far apart
	e.mu.Lock()
	e.positions["a"] = Position{X: -500, Y: 0}
	e.positions["b"] = Position{X: 500, Y: 0}
	e.mu.Unlock()

	e.AddEdge(Edge{SourceID: "a", TargetID: "b", Age: 0})

	// Run many ticks
	for i := 0; i < 100; i++ {
		e.Tick()
	}

	// Nodes should be closer together due to spring attraction
	finalPos := e.Positions().Get()
	initialDist := 1000.0
	finalDist := finalPos["b"].X - finalPos["a"].X
	if finalDist < 0 {
		finalDist = -finalDist
	}

	if finalDist >= initialDist {
		t.Errorf("expected nodes to move closer, initial dist: %f, final dist: %f",
			initialDist, finalDist)
	}
}

func TestBarnesHutThreshold(t *testing.T) {
	e := NewEngine()

	// Add nodes below threshold
	for i := 0; i < BarnesHutThreshold-10; i++ {
		e.AddNode(&Node{ID: string(rune(i))})
	}

	// Should use naive algorithm
	e.Tick() // Should not panic

	// Add more nodes to exceed threshold
	for i := 0; i < 20; i++ {
		e.AddNode(&Node{ID: "extra" + string(rune(i))})
	}

	if e.NodeCount() <= BarnesHutThreshold {
		t.Fatalf("expected more than %d nodes, got %d", BarnesHutThreshold, e.NodeCount())
	}

	// Should use Barnes-Hut algorithm
	e.Tick() // Should not panic
}

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	if params.RepulsionConstant <= 0 {
		t.Error("RepulsionConstant should be positive")
	}
	if params.SpringConstant <= 0 {
		t.Error("SpringConstant should be positive")
	}
	if params.SpringRestLength <= 0 {
		t.Error("SpringRestLength should be positive")
	}
	if params.DampingCoefficient <= 0 || params.DampingCoefficient >= 1 {
		t.Error("DampingCoefficient should be between 0 and 1")
	}
	if params.TicksPerSecond != DefaultTicksPerSecond {
		t.Errorf("expected %d ticks/sec, got %d", DefaultTicksPerSecond, params.TicksPerSecond)
	}
}

func TestCenterGravity(t *testing.T) {
	e := NewEngine()
	e.SetCenter(0, 0)

	// Add a single node far from center
	e.AddNode(&Node{ID: "far"})
	e.mu.Lock()
	e.positions["far"] = Position{X: 1000, Y: 1000}
	e.mu.Unlock()

	// Run simulation
	for i := 0; i < 200; i++ {
		e.Tick()
	}

	// Node should have moved toward center
	finalPos := e.Positions().Get()
	pos := finalPos["far"]

	// Should be closer to (0,0) than (1000,1000)
	distFromCenter := pos.X*pos.X + pos.Y*pos.Y
	initialDistFromCenter := 1000.0*1000.0 + 1000.0*1000.0

	if distFromCenter >= initialDistFromCenter {
		t.Errorf("expected node to move toward center, pos: (%f, %f)", pos.X, pos.Y)
	}
}

func BenchmarkTickNaive(b *testing.B) {
	e := NewEngine()
	for i := 0; i < 100; i++ {
		e.AddNode(&Node{ID: string(rune(i))})
	}
	// Add some edges
	for i := 0; i < 90; i++ {
		e.AddEdge(Edge{SourceID: string(rune(i)), TargetID: string(rune(i + 1))})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Tick()
	}
}

func BenchmarkTickBarnesHut(b *testing.B) {
	e := NewEngine()
	for i := 0; i < 600; i++ {
		e.AddNode(&Node{ID: string(rune(i))})
	}
	// Add some edges
	for i := 0; i < 590; i++ {
		e.AddEdge(Edge{SourceID: string(rune(i)), TargetID: string(rune(i + 1))})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Tick()
	}
}
