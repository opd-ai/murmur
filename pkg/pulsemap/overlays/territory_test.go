// Package overlays — Territory overlay tests.
//
//go:build noebiten
// +build noebiten

package overlays

import "testing"

func TestNewTerritoryOverlay(t *testing.T) {
	overlay := NewTerritoryOverlay()

	if overlay == nil {
		t.Fatal("NewTerritoryOverlay returned nil")
	}
	if !overlay.Visible {
		t.Error("Overlay should be visible by default")
	}
	if overlay.Opacity != 0.6 {
		t.Errorf("Expected default opacity 0.6, got %f", overlay.Opacity)
	}
	if overlay.Count() != 0 {
		t.Errorf("Expected 0 territories, got %d", overlay.Count())
	}
}

func TestTerritoryOverlaySetOpacity(t *testing.T) {
	overlay := NewTerritoryOverlay()

	tests := []struct {
		input    float32
		expected float32
	}{
		{0.5, 0.5},
		{0.0, 0.0},
		{1.0, 1.0},
		{-0.5, 0.0}, // Clamped to 0.
		{1.5, 1.0},  // Clamped to 1.
	}

	for _, tc := range tests {
		overlay.SetOpacity(tc.input)
		if overlay.Opacity != tc.expected {
			t.Errorf("SetOpacity(%f): expected %f, got %f", tc.input, tc.expected, overlay.Opacity)
		}
	}
}

func TestTerritoryOverlayAddRemove(t *testing.T) {
	overlay := NewTerritoryOverlay()

	t1 := NewTerritoryVisual("territory-1", 100, 100)
	t2 := NewTerritoryVisual("territory-2", 200, 200)
	t3 := NewTerritoryVisual("territory-3", 300, 300)

	overlay.AddTerritory(t1)
	if overlay.Count() != 1 {
		t.Errorf("Expected 1 territory, got %d", overlay.Count())
	}

	overlay.AddTerritory(t2)
	overlay.AddTerritory(t3)
	if overlay.Count() != 3 {
		t.Errorf("Expected 3 territories, got %d", overlay.Count())
	}

	// Remove middle territory.
	overlay.RemoveTerritory("territory-2")
	if overlay.Count() != 2 {
		t.Errorf("Expected 2 territories after removal, got %d", overlay.Count())
	}

	// Verify t2 is gone.
	if overlay.GetTerritory("territory-2") != nil {
		t.Error("Territory-2 should be removed")
	}

	// Verify t1 and t3 remain.
	if overlay.GetTerritory("territory-1") == nil {
		t.Error("Territory-1 should remain")
	}
	if overlay.GetTerritory("territory-3") == nil {
		t.Error("Territory-3 should remain")
	}
}

func TestTerritoryOverlayClear(t *testing.T) {
	overlay := NewTerritoryOverlay()

	overlay.AddTerritory(NewTerritoryVisual("t1", 0, 0))
	overlay.AddTerritory(NewTerritoryVisual("t2", 0, 0))

	overlay.ClearTerritories()
	if overlay.Count() != 0 {
		t.Errorf("Expected 0 territories after clear, got %d", overlay.Count())
	}
}

func TestTerritoryOverlayUpdate(t *testing.T) {
	overlay := NewTerritoryOverlay()

	tv := NewTerritoryVisual("test", 0, 0)
	overlay.AddTerritory(tv)

	initialPhase := tv.AnimationPhase
	overlay.Update(0.016) // ~60fps delta.

	if tv.AnimationPhase <= initialPhase {
		t.Error("Animation phase should advance after Update")
	}
}

func TestNewTerritoryVisual(t *testing.T) {
	tv := NewTerritoryVisual("my-territory", 123.5, 456.7)

	if tv == nil {
		t.Fatal("NewTerritoryVisual returned nil")
	}
	if tv.ID != "my-territory" {
		t.Errorf("Expected ID 'my-territory', got '%s'", tv.ID)
	}
	if tv.CentroidX != 123.5 {
		t.Errorf("Expected CentroidX 123.5, got %f", tv.CentroidX)
	}
	if tv.CentroidY != 456.7 {
		t.Errorf("Expected CentroidY 456.7, got %f", tv.CentroidY)
	}
	if tv.State != TerritoryNeutral {
		t.Errorf("Expected initial state TerritoryNeutral, got %d", tv.State)
	}
}

func TestTerritoryVisualSetters(t *testing.T) {
	tv := NewTerritoryVisual("test", 0, 0)

	// Test SetState.
	tv.SetState(TerritoryControlled)
	if tv.State != TerritoryControlled {
		t.Errorf("Expected TerritoryControlled, got %d", tv.State)
	}

	tv.SetState(TerritoryContested)
	if tv.State != TerritoryContested {
		t.Errorf("Expected TerritoryContested, got %d", tv.State)
	}

	// Test SetInfluence.
	tv.SetInfluence(75.5)
	if tv.Influence != 75.5 {
		t.Errorf("Expected influence 75.5, got %f", tv.Influence)
	}

	// Test SetBoundary.
	boundary := []Point{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 100, Y: 100},
		{X: 0, Y: 100},
	}
	tv.SetBoundary(boundary)
	if len(tv.Boundary) != 4 {
		t.Errorf("Expected 4 boundary points, got %d", len(tv.Boundary))
	}
}

func TestTerritoryBoundaryCircle(t *testing.T) {
	points := TerritoryBoundaryCircle(100, 100, 50, 8)

	if len(points) != 8 {
		t.Errorf("Expected 8 points, got %d", len(points))
	}

	// First point should be at (150, 100) - right side of circle.
	first := points[0]
	if first.X < 140 || first.X > 160 {
		t.Errorf("First point X should be ~150, got %f", first.X)
	}
	if first.Y < 90 || first.Y > 110 {
		t.Errorf("First point Y should be ~100, got %f", first.Y)
	}
}

func TestTerritoryBoundaryCircleMinSegments(t *testing.T) {
	// When segments < 3, should default to 12.
	points := TerritoryBoundaryCircle(0, 0, 100, 1)
	if len(points) != 12 {
		t.Errorf("Expected 12 points for segments < 3, got %d", len(points))
	}
}

func TestTerritoryStates(t *testing.T) {
	// Verify state constants match expected values.
	if TerritoryNeutral != 0 {
		t.Errorf("TerritoryNeutral should be 0, got %d", TerritoryNeutral)
	}
	if TerritoryControlled != 1 {
		t.Errorf("TerritoryControlled should be 1, got %d", TerritoryControlled)
	}
	if TerritoryContested != 2 {
		t.Errorf("TerritoryContested should be 2, got %d", TerritoryContested)
	}
}

func TestTerritoryGetNonexistent(t *testing.T) {
	overlay := NewTerritoryOverlay()

	if overlay.GetTerritory("nonexistent") != nil {
		t.Error("GetTerritory should return nil for nonexistent ID")
	}
}

func TestTerritoryRemoveNonexistent(t *testing.T) {
	overlay := NewTerritoryOverlay()
	overlay.AddTerritory(NewTerritoryVisual("t1", 0, 0))

	// Removing nonexistent should not panic or affect existing.
	overlay.RemoveTerritory("nonexistent")
	if overlay.Count() != 1 {
		t.Errorf("Expected 1 territory after removing nonexistent, got %d", overlay.Count())
	}
}
