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

func TestMomentumScrolling(t *testing.T) {
	c := NewCamera()

	// Apply momentum from a fast pan (screen-space delta)
	c.ApplyMomentum(50.0, 30.0)

	// Velocity should be set (inverted and scaled)
	if c.velocityX == 0 && c.velocityY == 0 {
		t.Error("expected non-zero velocity after ApplyMomentum")
	}

	initialX := c.X
	initialY := c.Y

	// Update multiple times - camera should continue moving
	for i := 0; i < 10; i++ {
		c.Update()
	}

	// Camera should have moved due to momentum
	if c.X == initialX && c.Y == initialY {
		t.Error("expected camera to move with momentum")
	}

	// Velocity should be decaying
	if math.Abs(c.velocityX) >= 50.0 || math.Abs(c.velocityY) >= 30.0 {
		t.Error("expected velocity to decay over time")
	}
}

func TestMomentumDeceleration(t *testing.T) {
	c := NewCamera()

	// Start with some momentum
	c.ApplyMomentum(20.0, 20.0)

	// Run update many times until momentum stops
	for i := 0; i < 200; i++ {
		c.Update()
	}

	// Momentum should have fully decayed
	if c.velocityX != 0 || c.velocityY != 0 {
		t.Errorf("expected zero velocity after deceleration, got (%f, %f)", c.velocityX, c.velocityY)
	}
}

func TestPanResetsMomentum(t *testing.T) {
	c := NewCamera()

	// Start with momentum
	c.ApplyMomentum(50.0, 30.0)

	// User pans - should reset momentum
	c.Pan(10, 10)

	if c.velocityX != 0 || c.velocityY != 0 {
		t.Errorf("expected momentum to be reset by Pan, got (%f, %f)", c.velocityX, c.velocityY)
	}
}

func TestAnimationClearsMomentum(t *testing.T) {
	c := NewCamera()

	// Start with momentum
	c.ApplyMomentum(50.0, 30.0)

	// Start animation - should clear momentum
	c.AnimateTo(100, 100)
	c.Update()

	if c.velocityX != 0 || c.velocityY != 0 {
		t.Errorf("expected momentum to be cleared by animation, got (%f, %f)", c.velocityX, c.velocityY)
	}
}

func TestMinimumMomentumThreshold(t *testing.T) {
	c := NewCamera()

	// Apply very small momentum (below threshold)
	c.ApplyMomentum(0.1, 0.1)

	// Should not start momentum
	if c.velocityX != 0 || c.velocityY != 0 {
		t.Error("expected no momentum for negligible velocity")
	}
}

func TestMaximumMomentumCap(t *testing.T) {
	c := NewCamera()

	// Apply extremely large momentum
	c.ApplyMomentum(10000.0, 10000.0)

	// Velocity should be capped
	maxVel := 50.0 // maxMomentumVelocity constant
	if math.Abs(c.velocityX) > maxVel || math.Abs(c.velocityY) > maxVel {
		t.Errorf("expected velocity capped at %f, got (%f, %f)", maxVel, c.velocityX, c.velocityY)
	}
}

func TestInputStateEndDragReturnsLastDelta(t *testing.T) {
	s := NewInputState()

	s.StartDrag(100, 100)
	s.UpdateDrag(150, 120) // dx=50, dy=20

	lastDx, lastDy := s.EndDrag()

	if lastDx != 50 || lastDy != 20 {
		t.Errorf("expected last delta (50, 20), got (%f, %f)", lastDx, lastDy)
	}

	if s.Dragging {
		t.Error("expected dragging to be false after EndDrag")
	}
}

func TestInputStateDeltaResets(t *testing.T) {
	s := NewInputState()

	s.StartDrag(100, 100)
	if s.LastDx != 0 || s.LastDy != 0 {
		t.Error("expected zero delta after StartDrag")
	}

	s.UpdateDrag(150, 120)
	if s.LastDx != 50 || s.LastDy != 20 {
		t.Errorf("expected delta (50, 20), got (%f, %f)", s.LastDx, s.LastDy)
	}

	s.EndDrag()
	if s.LastDx != 0 || s.LastDy != 0 {
		t.Error("expected zero delta after EndDrag")
	}
}
