// Package rendering provides Ebitengine-dependent tests for node/edge rendering.
// These tests require a display and are skipped in headless CI environments.
//
//go:build ebitentest
// +build ebitentest

package rendering

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRenderNode(t *testing.T) {
	// Create a small test image
	img := ebiten.NewImage(100, 100)

	style := NodeStyle{
		CoreColor:   color.RGBA{255, 0, 0, 255},
		RingColor:   color.RGBA{0, 0, 255, 255},
		HasRing:     true,
		HasHalo:     true,
		HaloAlpha:   0.5,
		Connections: 5,
		Activity:    10.0,
	}

	// Should not panic
	RenderNode(img, 50, 50, style, ZoomMeso)
}

func TestRenderEdge(t *testing.T) {
	img := ebiten.NewImage(100, 100)

	style := EdgeStyle{
		Color:  color.RGBA{255, 200, 0, 255},
		Age:    30,
		Active: true,
	}

	// Should not panic
	RenderEdge(img, 10, 10, 90, 90, style, ZoomMeso)
}
