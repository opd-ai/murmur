// Package overlays — Oracle Pool visualization tests.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"
	"time"
)

func TestNewOraclePoolOverlay(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	if overlay == nil {
		t.Fatal("NewOraclePoolOverlay returned nil")
	}
	if !overlay.Visible {
		t.Error("Overlay should be visible by default")
	}
	if overlay.Opacity != 0.8 {
		t.Errorf("Expected default opacity 0.8, got %f", overlay.Opacity)
	}
	if overlay.Count() != 0 {
		t.Errorf("Expected 0 pools, got %d", overlay.Count())
	}
}

func TestOraclePoolOverlaySetOpacity(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	tests := []struct {
		input    float32
		expected float32
	}{
		{0.5, 0.5},
		{0.0, 0.0},
		{1.0, 1.0},
		{-0.5, 0.0},
		{1.5, 1.0},
	}

	for _, tc := range tests {
		overlay.SetOpacity(tc.input)
		if overlay.Opacity != tc.expected {
			t.Errorf("SetOpacity(%f): expected %f, got %f", tc.input, tc.expected, overlay.Opacity)
		}
	}
}

func TestOraclePoolOverlayAddRemove(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	id1 := [32]byte{1, 2, 3}
	id2 := [32]byte{4, 5, 6}
	id3 := [32]byte{7, 8, 9}

	p1 := NewOraclePoolVisual(id1, 100, 100)
	p2 := NewOraclePoolVisual(id2, 200, 200)
	p3 := NewOraclePoolVisual(id3, 300, 300)

	overlay.AddPool(p1)
	if overlay.Count() != 1 {
		t.Errorf("Expected 1 pool, got %d", overlay.Count())
	}

	overlay.AddPool(p2)
	overlay.AddPool(p3)
	if overlay.Count() != 3 {
		t.Errorf("Expected 3 pools, got %d", overlay.Count())
	}

	// Remove middle pool.
	overlay.RemovePool(id2)
	if overlay.Count() != 2 {
		t.Errorf("Expected 2 pools after removal, got %d", overlay.Count())
	}

	// Verify p2 is gone.
	if overlay.GetPool(id2) != nil {
		t.Error("Pool id2 should be removed")
	}

	// Verify p1 and p3 remain.
	if overlay.GetPool(id1) == nil {
		t.Error("Pool id1 should remain")
	}
	if overlay.GetPool(id3) == nil {
		t.Error("Pool id3 should remain")
	}
}

func TestOraclePoolOverlayClear(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	overlay.AddPool(NewOraclePoolVisual([32]byte{1}, 0, 0))
	overlay.AddPool(NewOraclePoolVisual([32]byte{2}, 0, 0))

	overlay.ClearPools()
	if overlay.Count() != 0 {
		t.Errorf("Expected 0 pools after clear, got %d", overlay.Count())
	}
}

func TestOraclePoolOverlayUpdate(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	pool := NewOraclePoolVisual([32]byte{1}, 0, 0)
	overlay.AddPool(pool)

	initialPhase := pool.AnimationPhase
	overlay.Update(0.016)

	if pool.AnimationPhase <= initialPhase {
		t.Error("Animation phase should advance after Update")
	}
}

func TestNewOraclePoolVisual(t *testing.T) {
	id := [32]byte{10, 20, 30}
	pool := NewOraclePoolVisual(id, 150.5, 250.5)

	if pool == nil {
		t.Fatal("NewOraclePoolVisual returned nil")
	}
	if pool.PoolID != id {
		t.Error("Pool ID mismatch")
	}
	if pool.X != 150.5 {
		t.Errorf("Expected X 150.5, got %f", pool.X)
	}
	if pool.Y != 250.5 {
		t.Errorf("Expected Y 250.5, got %f", pool.Y)
	}
	if pool.State != OraclePoolPending {
		t.Errorf("Expected initial state OraclePoolPending, got %d", pool.State)
	}
}

func TestOraclePoolVisualSetters(t *testing.T) {
	pool := NewOraclePoolVisual([32]byte{}, 0, 0)

	// Test SetState.
	pool.SetState(OraclePoolRevealing)
	if pool.State != OraclePoolRevealing {
		t.Errorf("Expected OraclePoolRevealing, got %d", pool.State)
	}

	pool.SetState(OraclePoolResolved)
	if pool.State != OraclePoolResolved {
		t.Errorf("Expected OraclePoolResolved, got %d", pool.State)
	}

	pool.SetState(OraclePoolExpired)
	if pool.State != OraclePoolExpired {
		t.Errorf("Expected OraclePoolExpired, got %d", pool.State)
	}

	// Test SetPosition.
	pool.SetPosition(500, 600)
	if pool.X != 500 || pool.Y != 600 {
		t.Errorf("Expected position (500, 600), got (%f, %f)", pool.X, pool.Y)
	}

	// Test SetQuestion.
	pool.SetQuestion("Will message count exceed 1000?")
	if pool.Question != "Will message count exceed 1000?" {
		t.Errorf("Question mismatch")
	}

	// Test SetDeadline.
	deadline := time.Now().Add(24 * time.Hour)
	pool.SetDeadline(deadline)
	if !pool.Deadline.Equal(deadline) {
		t.Error("Deadline mismatch")
	}

	// Test SetResolutionTime.
	resolution := time.Now().Add(48 * time.Hour)
	pool.SetResolutionTime(resolution)
	if !pool.ResolutionTime.Equal(resolution) {
		t.Error("Resolution time mismatch")
	}

	// Test SetPredictionCount.
	pool.SetPredictionCount(15)
	if pool.PredictionCount != 15 {
		t.Errorf("Expected prediction count 15, got %d", pool.PredictionCount)
	}
}

func TestOraclePoolStates(t *testing.T) {
	// Verify state constants.
	if OraclePoolPending != 0 {
		t.Errorf("OraclePoolPending should be 0, got %d", OraclePoolPending)
	}
	if OraclePoolRevealing != 1 {
		t.Errorf("OraclePoolRevealing should be 1, got %d", OraclePoolRevealing)
	}
	if OraclePoolResolved != 2 {
		t.Errorf("OraclePoolResolved should be 2, got %d", OraclePoolResolved)
	}
	if OraclePoolExpired != 3 {
		t.Errorf("OraclePoolExpired should be 3, got %d", OraclePoolExpired)
	}
}

func TestOraclePoolGetNonexistent(t *testing.T) {
	overlay := NewOraclePoolOverlay()

	if overlay.GetPool([32]byte{99, 99, 99}) != nil {
		t.Error("GetPool should return nil for nonexistent ID")
	}
}

func TestOraclePoolRemoveNonexistent(t *testing.T) {
	overlay := NewOraclePoolOverlay()
	overlay.AddPool(NewOraclePoolVisual([32]byte{1}, 0, 0))

	// Removing nonexistent should not panic or affect existing.
	overlay.RemovePool([32]byte{99, 99, 99})
	if overlay.Count() != 1 {
		t.Errorf("Expected 1 pool after removing nonexistent, got %d", overlay.Count())
	}
}
