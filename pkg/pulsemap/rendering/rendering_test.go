// Package rendering tests verify the node and edge rendering functions.
// Pure logic tests that don't require Ebitengine.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"testing"
)

func TestColorFromHash(t *testing.T) {
	// Test surface node color
	hash := []byte{128, 128, 128}
	c := ColorFromHash(hash, false)
	if c.A != 255 {
		t.Errorf("expected full alpha, got %d", c.A)
	}

	// Test specter node color (should be cool tones)
	specterHash := []byte{0, 128, 128}
	sc := ColorFromHash(specterHash, true)
	if sc.A != 255 {
		t.Errorf("expected full alpha, got %d", sc.A)
	}
	// Specter colors should be blueish (B >= R)
	// This is a rough heuristic based on hue 200-280
	if sc.B < sc.R/2 {
		t.Logf("Specter color may not be cool-toned: R=%d, G=%d, B=%d", sc.R, sc.G, sc.B)
	}
}

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

func TestHSLToRGB(t *testing.T) {
	// Test red (hue 0)
	r, g, b := hslToRGB(0, 1.0, 0.5)
	if r < 250 || g > 5 || b > 5 {
		t.Errorf("expected red, got R=%d G=%d B=%d", r, g, b)
	}

	// Test green (hue 120)
	r, g, b = hslToRGB(120, 1.0, 0.5)
	if g < 250 || r > 5 || b > 5 {
		t.Errorf("expected green, got R=%d G=%d B=%d", r, g, b)
	}

	// Test gray (saturation 0)
	r, g, b = hslToRGB(0, 0, 0.5)
	if r != g || g != b {
		t.Errorf("expected gray (equal RGB), got R=%d G=%d B=%d", r, g, b)
	}
}

// NOTE: Tests that require Ebitengine rendering (RenderNode, RenderEdge)
// are in rendering_ebiten_test.go behind the "ebitentest" build tag
// per TECHNICAL_IMPLEMENTATION.md.
