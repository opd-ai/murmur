// Package rendering provides tests for the Renderer type.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"image/color"
	"testing"

	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
)

func TestNewRenderer(t *testing.T) {
	engine := layout.NewEngine()
	renderer, err := NewRenderer(engine)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}
	if renderer == nil {
		t.Fatal("NewRenderer returned nil")
	}
	if renderer.Camera() == nil {
		t.Error("Expected camera to be initialized")
	}
	if renderer.InputState() == nil {
		t.Error("Expected input state to be initialized")
	}
}

func TestRendererAddRemoveNode(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	data := &NodeData{
		ID:          "node1",
		Connections: 5,
		Activity:    10.0,
	}
	renderer.AddNode(data)

	// Verify node was added.
	renderer.mu.RLock()
	_, ok := renderer.nodeData["node1"]
	renderer.mu.RUnlock()
	if !ok {
		t.Error("Node was not added")
	}

	renderer.RemoveNode("node1")

	renderer.mu.RLock()
	_, ok = renderer.nodeData["node1"]
	renderer.mu.RUnlock()
	if ok {
		t.Error("Node was not removed")
	}
}

func TestRendererAddEdge(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	edge := EdgeData{
		SourceID: "node1",
		TargetID: "node2",
		Age:      30,
		Active:   true,
	}
	renderer.AddEdge(edge)

	renderer.mu.RLock()
	edgeCount := len(renderer.edges)
	renderer.mu.RUnlock()

	if edgeCount != 1 {
		t.Errorf("Expected 1 edge, got %d", edgeCount)
	}
}

func TestRendererClearEdges(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	renderer.AddEdge(EdgeData{SourceID: "a", TargetID: "b"})
	renderer.AddEdge(EdgeData{SourceID: "b", TargetID: "c"})
	renderer.ClearEdges()

	renderer.mu.RLock()
	edgeCount := len(renderer.edges)
	renderer.mu.RUnlock()

	if edgeCount != 0 {
		t.Errorf("Expected 0 edges after clear, got %d", edgeCount)
	}
}

func TestRendererSetEdges(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	edges := []EdgeData{
		{SourceID: "a", TargetID: "b"},
		{SourceID: "b", TargetID: "c"},
		{SourceID: "c", TargetID: "d"},
	}
	renderer.SetEdges(edges)

	renderer.mu.RLock()
	edgeCount := len(renderer.edges)
	renderer.mu.RUnlock()

	if edgeCount != 3 {
		t.Errorf("Expected 3 edges, got %d", edgeCount)
	}
}

func TestRendererUpdate(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Call Update multiple times and verify no errors.
	for i := 0; i < 10; i++ {
		if err := renderer.Update(); err != nil {
			t.Errorf("Update() returned error: %v", err)
		}
	}
}

func TestRendererLayout(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	w, h := renderer.Layout(1024, 768)
	if w != 1024 || h != 768 {
		t.Errorf("Layout returned %dx%d, expected 1024x768", w, h)
	}
}

func TestRendererMouseInteraction(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Test drag start/update/end.
	renderer.HandleMouseDown(100, 100)
	renderer.HandleMouseMove(150, 150)
	renderer.HandleMouseUp()

	// Verify camera panning occurred.
	cam := renderer.Camera()
	if cam.X == 0 && cam.Y == 0 {
		t.Error("Expected camera to have moved after drag")
	}
}

func TestRendererMouseWheel(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	initialScale := renderer.Camera().Scale

	// Zoom in.
	renderer.HandleMouseWheel(400, 300, 1.0)
	if renderer.Camera().Scale <= initialScale {
		t.Error("Expected scale to increase after zoom in")
	}

	// Zoom out.
	renderer.HandleMouseWheel(400, 300, -1.0)
	renderer.HandleMouseWheel(400, 300, -1.0)
	// Should have decreased from zoomed-in state.
}

func TestRendererNodeSelection(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Add a node to the engine and renderer.
	node := &layout.Node{ID: "testnode", Connections: 3}
	engine.AddNode(node)
	engine.Tick() // Populate position buffer.

	renderer.AddNode(&NodeData{
		ID:          "testnode",
		Connections: 3,
	})

	// Initially no selection.
	if renderer.SelectedNode() != "" {
		t.Error("Expected no selection initially")
	}

	// Select via InputState directly for testing.
	renderer.InputState().SelectNode("testnode")
	if renderer.SelectedNode() != "testnode" {
		t.Error("Expected testnode to be selected")
	}
}

func TestRendererFocusNode(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Add a node at a known position.
	node := &layout.Node{ID: "focus-target", Connections: 1}
	engine.AddNode(node)
	engine.Tick()

	renderer.AddNode(&NodeData{ID: "focus-target", Connections: 1})

	// Focus on the node.
	renderer.FocusNode("focus-target")

	// Camera should be animating.
	cam := renderer.Camera()
	if !cam.Animating {
		t.Error("Expected camera to be animating after FocusNode")
	}
}

func TestRendererSetBackgroundColor(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	newColor := color.RGBA{50, 60, 70, 255}
	renderer.SetBackgroundColor(newColor)

	renderer.mu.RLock()
	bg := renderer.backgroundColor
	renderer.mu.RUnlock()

	if bg != newColor {
		t.Errorf("Background color not set: got %v, want %v", bg, newColor)
	}
}

func TestRendererSetCamera(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	newCamera := interaction.NewCamera()
	newCamera.X = 100
	newCamera.Y = 200

	renderer.SetCamera(newCamera)

	cam := renderer.Camera()
	if cam.X != 100 || cam.Y != 200 {
		t.Errorf("Camera not set correctly: got (%f, %f), want (100, 200)", cam.X, cam.Y)
	}
}

func TestRendererHitTestNoEngine(t *testing.T) {
	renderer := &Renderer{
		engine: nil,
		camera: interaction.NewCamera(),
		input:  interaction.NewInputState(),
	}

	nodeID := renderer.hitTestNodes(100, 100)
	if nodeID != "" {
		t.Error("Expected empty string for nil engine")
	}
}

func TestRendererHitTestNoCamera(t *testing.T) {
	engine := layout.NewEngine()
	renderer := &Renderer{
		engine:   engine,
		camera:   nil,
		input:    interaction.NewInputState(),
		nodeData: make(map[string]*NodeData),
	}

	nodeID := renderer.hitTestNodes(100, 100)
	if nodeID != "" {
		t.Error("Expected empty string for nil camera")
	}
}

func TestRendererFocusNodeNotFound(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Focus on non-existent node - should not panic.
	renderer.FocusNode("nonexistent")

	cam := renderer.Camera()
	if cam.Animating {
		t.Error("Camera should not animate for non-existent node")
	}
}

func TestRendererConcurrentAccess(t *testing.T) {
	engine := layout.NewEngine()
	renderer, _ := NewRenderer(engine)

	// Add some nodes.
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		renderer.AddNode(&NodeData{ID: id, Connections: i})
	}

	// Concurrent operations.
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			renderer.Update()
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			renderer.AddEdge(EdgeData{SourceID: "a", TargetID: "b"})
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 100; i++ {
			_ = renderer.SelectedNode()
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}
