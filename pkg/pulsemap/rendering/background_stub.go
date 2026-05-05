// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// Test stub for background rendering.
//
//go:build test
// +build test

package rendering

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// BackgroundRenderer test stub.
type BackgroundRenderer struct {
	noiseScale     float64
	noiseAmplitude float64
}

// NewBackgroundRenderer test stub.
func NewBackgroundRenderer() *BackgroundRenderer {
	return &BackgroundRenderer{
		noiseScale:     0.015,
		noiseAmplitude: 0.12,
	}
}

// Draw test stub.
func (b *BackgroundRenderer) Draw(screen *ebiten.Image) {
	// No-op for tests.
}

// SetNoiseScale test stub.
func (b *BackgroundRenderer) SetNoiseScale(scale float64) {
	b.noiseScale = scale
}

// SetNoiseAmplitude test stub.
func (b *BackgroundRenderer) SetNoiseAmplitude(amplitude float64) {
	b.noiseAmplitude = amplitude
}

// GetBaseColor test stub.
func (b *BackgroundRenderer) GetBaseColor() color.RGBA {
	return color.RGBA{11, 14, 21, 255}
}
