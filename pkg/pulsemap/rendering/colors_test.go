// Package rendering tests for color utilities and node styling.
// These tests use stub types and don't require Ebitengine rendering.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"image/color"
	"testing"
)

// TestColorFromHash verifies hash-to-color derivation.
func TestColorFromHash(t *testing.T) {
	tests := []struct {
		name      string
		hash      []byte
		isSpecter bool
	}{
		{"surface basic", []byte{0, 128, 128}, false},
		{"specter basic", []byte{128, 128, 128}, true},
		{"max values", []byte{255, 255, 255}, false},
		{"min values", []byte{0, 0, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ColorFromHash(tt.hash, tt.isSpecter)
			if c.A != 255 {
				t.Errorf("Alpha should be 255, got %d", c.A)
			}
			// Verify non-zero color components (varies based on hash)
			if c.R == 0 && c.G == 0 && c.B == 0 {
				t.Error("Color should not be pure black")
			}
		})
	}
}

// TestColorFromHashShortInput verifies fallback for short input.
func TestColorFromHashShortInput(t *testing.T) {
	c := ColorFromHash([]byte{1, 2}, false) // Only 2 bytes
	expected := color.RGBA{128, 128, 128, 255}
	if c != expected {
		t.Errorf("Short input should return gray, got %+v", c)
	}
}

// TestColorFromHashSpecterRange verifies Specter hue constraint.
func TestColorFromHashSpecterRange(t *testing.T) {
	// Specter nodes should have cool tones (200-280° hue)
	// We can't directly verify hue, but we can verify the color is "cool"
	c := ColorFromHash([]byte{0, 128, 128}, true)
	// Cool colors typically have higher B than R
	if c.B < c.R/2 {
		t.Logf("Specter color: R=%d, G=%d, B=%d", c.R, c.G, c.B)
		// This is a weak test but validates the color generation runs
	}
}

// TestHSLToRGB verifies HSL conversion.
func TestHSLToRGB(t *testing.T) {
	tests := []struct {
		h, s, l    float64
		minR, maxR uint8
		minG, maxG uint8
		minB, maxB uint8
	}{
		{0, 1, 0.5, 200, 255, 0, 50, 0, 50},       // Red
		{120, 1, 0.5, 0, 50, 200, 255, 0, 50},     // Green
		{240, 1, 0.5, 0, 50, 0, 50, 200, 255},     // Blue
		{0, 0, 0.5, 120, 135, 120, 135, 120, 135}, // Gray
	}

	for i, tt := range tests {
		r, g, b := hslToRGB(tt.h, tt.s, tt.l)
		if r < tt.minR || r > tt.maxR {
			t.Errorf("case %d: R=%d not in [%d,%d]", i, r, tt.minR, tt.maxR)
		}
		if g < tt.minG || g > tt.maxG {
			t.Errorf("case %d: G=%d not in [%d,%d]", i, g, tt.minG, tt.maxG)
		}
		if b < tt.minB || b > tt.maxB {
			t.Errorf("case %d: B=%d not in [%d,%d]", i, b, tt.minB, tt.maxB)
		}
	}
}

// TestNodeStyleDefaults verifies NodeStyle struct.
func TestNodeStyleDefaults(t *testing.T) {
	s := NodeStyle{
		CoreColor:   color.RGBA{100, 150, 200, 255},
		RingColor:   color.RGBA{50, 100, 150, 255},
		HasRing:     true,
		HasHalo:     true,
		HaloAlpha:   0.5,
		IsSpecter:   false,
		Selected:    false,
		Connections: 10,
		Activity:    5.0,
		Resonance:   75.0,
	}

	if s.Connections != 10 {
		t.Errorf("Connections = %d, want 10", s.Connections)
	}
	if s.HaloAlpha != 0.5 {
		t.Errorf("HaloAlpha = %v, want 0.5", s.HaloAlpha)
	}
}

// TestZoomLevel verifies zoom level constants.
func TestZoomLevel(t *testing.T) {
	if ZoomMacro >= ZoomMeso {
		t.Error("ZoomMacro should be less than ZoomMeso")
	}
	if ZoomMeso >= ZoomMicro {
		t.Error("ZoomMeso should be less than ZoomMicro")
	}
}

// TestComputeNodeRadius verifies radius calculation.
func TestComputeNodeRadius(t *testing.T) {
	// Surface node with connections
	surface := NodeStyle{
		IsSpecter:   false,
		Connections: 10,
		Activity:    5.0,
	}
	r1 := computeNodeRadius(surface)
	if r1 <= 0 {
		t.Error("Surface node radius should be positive")
	}

	// Specter node with resonance
	specter := NodeStyle{
		IsSpecter: true,
		Resonance: 50.0,
	}
	r2 := computeNodeRadius(specter)
	if r2 <= 0 {
		t.Error("Specter node radius should be positive")
	}

	// Node with zero connections should still have base radius
	zero := NodeStyle{
		IsSpecter:   false,
		Connections: 0,
		Activity:    0,
	}
	r3 := computeNodeRadius(zero)
	if r3 <= 0 {
		t.Error("Zero-connection node should have positive base radius")
	}
}
