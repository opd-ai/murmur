// Package interaction tests verify camera and input handling.
package interaction

import (
	"math"
	"testing"
)

func TestNewCamera(t *testing.T) {
	c := NewCamera()
	if c.X != 0 || c.Y != 0 {
		t.Errorf("expected origin, got (%f, %f)", c.X, c.Y)
	}
	if c.Scale != 1.0 {
		t.Errorf("expected scale 1.0, got %f", c.Scale)
	}
}

func TestCameraPan(t *testing.T) {
	c := NewCamera()
	c.Pan(100, 50)

	// Pan should move camera in opposite direction
	if c.X != -100 || c.Y != -50 {
		t.Errorf("expected (-100, -50), got (%f, %f)", c.X, c.Y)
	}
}

func TestCameraZoom(t *testing.T) {
	c := NewCamera()

	// Zoom in (factor > 1)
	c.Zoom(2.0, 400, 300, 800, 600)
	if c.Scale != 2.0 {
		t.Errorf("expected scale 2.0, got %f", c.Scale)
	}

	// Zoom out
	c.Zoom(0.5, 400, 300, 800, 600)
	if c.Scale != 1.0 {
		t.Errorf("expected scale 1.0, got %f", c.Scale)
	}
}

func TestCameraZoomLimits(t *testing.T) {
	c := NewCamera()

	// Try to zoom beyond max
	for i := 0; i < 10; i++ {
		c.Zoom(2.0, 400, 300, 800, 600)
	}
	if c.Scale > MaxScale {
		t.Errorf("scale %f exceeds max %f", c.Scale, MaxScale)
	}

	// Try to zoom beyond min
	for i := 0; i < 10; i++ {
		c.Zoom(0.1, 400, 300, 800, 600)
	}
	if c.Scale < MinScale {
		t.Errorf("scale %f below min %f", c.Scale, MinScale)
	}
}

func TestCameraAnimation(t *testing.T) {
	c := NewCamera()
	c.AnimateTo(100, 200)

	if !c.Animating {
		t.Error("expected animating to be true")
	}
	if c.TargetX != 100 || c.TargetY != 200 {
		t.Errorf("expected target (100, 200), got (%f, %f)", c.TargetX, c.TargetY)
	}

	// Run animation until complete
	for c.Animating {
		c.Update()
	}

	// Should be at target
	if math.Abs(c.X-100) > 1 || math.Abs(c.Y-200) > 1 {
		t.Errorf("expected position near (100, 200), got (%f, %f)", c.X, c.Y)
	}
}

func TestScreenToWorld(t *testing.T) {
	c := NewCamera()
	c.X = 100
	c.Y = 100
	c.Scale = 2.0

	// Screen center should map to camera position
	wx, wy := c.ScreenToWorld(400, 300, 800, 600)
	if math.Abs(wx-100) > 0.001 || math.Abs(wy-100) > 0.001 {
		t.Errorf("screen center should map to camera pos, got (%f, %f)", wx, wy)
	}
}

func TestWorldToScreen(t *testing.T) {
	c := NewCamera()
	c.X = 100
	c.Y = 100
	c.Scale = 2.0

	// Camera position should map to screen center
	sx, sy := c.WorldToScreen(100, 100, 800, 600)
	if math.Abs(sx-400) > 0.001 || math.Abs(sy-300) > 0.001 {
		t.Errorf("camera pos should map to screen center, got (%f, %f)", sx, sy)
	}
}

func TestViewBounds(t *testing.T) {
	c := NewCamera()
	c.Scale = 1.0

	minX, minY, maxX, maxY := c.ViewBounds(800, 600)

	// At scale 1, view should span [-400, 400] x [-300, 300] centered at origin
	if math.Abs(minX+400) > 0.001 || math.Abs(maxX-400) > 0.001 {
		t.Errorf("unexpected X bounds: [%f, %f]", minX, maxX)
	}
	if math.Abs(minY+300) > 0.001 || math.Abs(maxY-300) > 0.001 {
		t.Errorf("unexpected Y bounds: [%f, %f]", minY, maxY)
	}
}

func TestInputStateDrag(t *testing.T) {
	s := NewInputState()

	s.StartDrag(100, 100)
	if !s.Dragging {
		t.Error("expected dragging to be true")
	}

	dx, dy := s.UpdateDrag(150, 120)
	if dx != 50 || dy != 20 {
		t.Errorf("expected delta (50, 20), got (%f, %f)", dx, dy)
	}

	s.EndDrag()
	if s.Dragging {
		t.Error("expected dragging to be false")
	}
}

func TestHitTest(t *testing.T) {
	// Point inside circle
	if !HitTest(50, 50, 52, 53, 10) {
		t.Error("expected hit inside circle")
	}

	// Point outside circle
	if HitTest(50, 50, 100, 100, 10) {
		t.Error("expected miss outside circle")
	}

	// Point on edge
	if !HitTest(0, 0, 10, 0, 10) {
		t.Error("expected hit on edge")
	}
}

func TestNodeSelection(t *testing.T) {
	s := NewInputState()

	s.SelectNode("node-123")
	if s.SelectedNodeID != "node-123" {
		t.Errorf("expected node-123, got %s", s.SelectedNodeID)
	}

	s.ClearSelection()
	if s.SelectedNodeID != "" {
		t.Errorf("expected empty selection, got %s", s.SelectedNodeID)
	}
}
