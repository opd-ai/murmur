// Package rendering tests verify the node and edge rendering functions.
// Pure logic tests that don't require Ebitengine.
//
//go:build noebiten
// +build noebiten

package rendering

import (
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

// NOTE: Tests that require Ebitengine rendering (RenderNode, RenderEdge)
// are in rendering_ebiten_test.go behind the "ebitentest" build tag
// per TECHNICAL_IMPLEMENTATION.md.
