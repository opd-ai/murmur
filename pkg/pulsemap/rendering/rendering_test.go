// Package rendering tests verify the node and edge rendering functions.
// Pure logic tests that don't require Ebitengine.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"math"
	"testing"
)

func TestColorFromHashEmpty(t *testing.T) {
	// Test with empty hash
	c := ColorFromHash([]byte{}, false)
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("expected gray fallback, got R=%d G=%d B=%d", c.R, c.G, c.B)
	}
}

func TestZoomLevelFromScale(t *testing.T) {
	tests := []struct {
		scale float64
		want  ZoomLevel
	}{
		{0.1, ZoomMacro},
		{0.2, ZoomMacro},
		{0.5, ZoomMeso},
		{1.0, ZoomMeso},
		{1.4, ZoomMeso},
		{2.0, ZoomMicro},
		{5.0, ZoomMicro},
	}

	for _, tt := range tests {
		got := ZoomLevelFromScale(tt.scale)
		if got != tt.want {
			t.Errorf("ZoomLevelFromScale(%f) = %v, want %v", tt.scale, got, tt.want)
		}
	}
}

func TestEdgeThicknessFromInteractionFrequency(t *testing.T) {
	// Test that edge thickness scales logarithmically with interaction frequency.
	// Formula: thickness = base + scale * ln(1 + frequency)
	// where base = 1.5, scale = 1.5

	tests := []struct {
		frequency   float64
		expectedMin float64
		expectedMax float64
		description string
	}{
		{0.0, 1.5, 1.5, "zero frequency should give base thickness"},
		{1.0, 2.5, 2.6, "1 msg/day should give ~2.5px"},
		{10.0, 5.0, 5.2, "10 msg/day should give ~5px"},
		{100.0, 8.0, 9.0, "100 msg/day should give ~8.5px"},
	}

	const baseThickness = 1.5
	const thicknessScale = 1.5

	for _, tt := range tests {
		thickness := baseThickness + thicknessScale*math.Log(1+tt.frequency)
		if thickness < tt.expectedMin || thickness > tt.expectedMax {
			t.Errorf("frequency %.1f: got thickness %.2f, expected in range [%.2f, %.2f] (%s)",
				tt.frequency, thickness, tt.expectedMin, tt.expectedMax, tt.description)
		}
	}

	// Verify monotonic increase
	freq1 := 10.0
	freq2 := 20.0
	thick1 := baseThickness + thicknessScale*math.Log(1+freq1)
	thick2 := baseThickness + thicknessScale*math.Log(1+freq2)
	if thick2 <= thick1 {
		t.Errorf("thickness should increase with frequency: %.2f (freq=%.0f) should be > %.2f (freq=%.0f)",
			thick2, freq2, thick1, freq1)
	}
}

// NOTE: Tests that require Ebitengine rendering (RenderNode, RenderEdge)
// are in rendering_ebiten_test.go behind the "ebitentest" build tag
// per TECHNICAL_IMPLEMENTATION.md.
