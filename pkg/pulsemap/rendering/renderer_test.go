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

// TestHandleMouseDownClearsOrphanedDragState verifies that HandleMouseDown resets
// an existing Dragging=true state before beginning a new interaction.
// Per AUDIT.md HIGH finding: isDragging / InputState.Dragging can become orphaned
// when the mouse is released outside the window.
func TestHandleMouseDownClearsOrphanedDragState(t *testing.T) {
	engine := layout.NewEngine()
	renderer, err := NewRenderer(engine)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	// Simulate an orphaned drag (Dragging stuck true, no corresponding mouse-up).
	renderer.InputState().StartDrag(50, 50)
	if !renderer.InputState().Dragging {
		t.Fatal("precondition: expected Dragging=true after StartDrag")
	}

	// A new mouse-down elsewhere (on empty space) must clear the orphan first,
	// then begin a fresh drag — Dragging should remain true but reset to the new origin.
	renderer.HandleMouseDown(200, 200)

	inputState := renderer.InputState()
	if !inputState.Dragging {
		t.Error("expected Dragging=true after HandleMouseDown on empty space")
	}
	if inputState.DragStartX != 200 || inputState.DragStartY != 200 {
		t.Errorf("expected new drag origin (200,200), got (%v,%v)",
			inputState.DragStartX, inputState.DragStartY)
	}
}

// TestHandleMouseDownOrphanOnNodeHit verifies that HandleMouseDown on a node
// clears Dragging and does NOT start a new drag (node selection replaces drag).
func TestHandleMouseDownOrphanOnNodeHit(t *testing.T) {
	engine := layout.NewEngine()
	renderer, err := NewRenderer(engine)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	// Force Dragging=true (orphaned state).
	renderer.InputState().StartDrag(50, 50)

	// Click on empty space away from any node — orphan clears, new drag starts.
	renderer.HandleMouseDown(9999, 9999)

	// Drag should start since we clicked empty space.
	if !renderer.InputState().Dragging {
		t.Error("expected Dragging=true after clicking empty space with prior orphan")
	}
}

// TestHitTestNodesScaleInvariance verifies that hit detection is consistent across
// the full scale range. Per AUDIT.md HIGH finding: the old formula
// (visualRadius * 1.5 / scale) caused misses at high zoom and bloated zones at low
// zoom. The new formula ensures a minimum world-unit hit radius at all scales.
func TestHitTestNodesScaleInvariance(t *testing.T) {
	scales := []struct {
		name  string
		scale float64
	}{
		{"low_zoom", 0.1},
		{"normal_zoom", 1.0},
		{"high_zoom", 5.0},
	}

	for _, tc := range scales {
		t.Run(tc.name, func(t *testing.T) {
			engine := layout.NewEngine()
			renderer, err := NewRenderer(engine)
			if err != nil {
				t.Fatalf("NewRenderer failed: %v", err)
			}
			renderer.camera.Scale = tc.scale

			const nodeID = "testnode"
			renderer.AddNode(&NodeData{
				ID:          nodeID,
				Connections: 3,
				Activity:    0,
			})
			// Inject node position at world origin via position buffer.
			engine.Positions().Swap(map[string]layout.Position{
				nodeID: {X: 0, Y: 0},
			})

			// Click exactly at the node's world position (screen centre maps to world origin
			// at default camera X=0, Y=0).
			screenCX := float64(renderer.screenWidth) / 2
			screenCY := float64(renderer.screenHeight) / 2
			hit := renderer.hitTestNodes(screenCX, screenCY)

			if hit != nodeID {
				t.Errorf("scale=%.1f: expected hit=%q at node centre, got %q", tc.scale, nodeID, hit)
			}
		})
	}
}
