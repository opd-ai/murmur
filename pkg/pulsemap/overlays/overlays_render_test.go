// Package overlays provides Ebitengine-dependent tests for overlay rendering.
// These tests require a display and are skipped in headless CI environments.
//
//go:build ebitentest
// +build ebitentest

package overlays

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestParticleEmitterRender(t *testing.T) {
	e := NewParticleEmitter(10, 10.0)
	e.Update(1.0, 50, 50, 10, 50)

	img := ebiten.NewImage(100, 100)
	// Should not panic
	e.Render(img, 0, 0, 1.0)
}

func TestShroudIndicator(t *testing.T) {
	img := ebiten.NewImage(100, 100)

	// Should not panic with active indicator
	ShroudIndicator(img, 50, 50, true, 0.5)

	// Should not panic with inactive indicator
	ShroudIndicator(img, 50, 50, false, 0.5)
}

func TestMiniGameVisualizationRender(t *testing.T) {
	d := &MiniGameVisualization{
		Player1X:  20,
		Player1Y:  20,
		Player2X:  80,
		Player2Y:  80,
		Color1:    color.RGBA{255, 0, 0, 255},
		Color2:    color.RGBA{0, 0, 255, 255},
		Intensity: 0.8,
		Phase:     0.5,
	}

	img := ebiten.NewImage(100, 100)
	// Should not panic
	d.Render(img, 0, 0, 1.0)
}
