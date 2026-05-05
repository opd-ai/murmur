// Package rendering tests verify background generation.
package rendering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBackgroundRenderer(t *testing.T) {
	bg := NewBackgroundRenderer()
	assert.NotNil(t, bg)
	assert.Equal(t, 0.015, bg.noiseScale)
	assert.Equal(t, 0.12, bg.noiseAmplitude)
}

func TestBackgroundRendererSetNoiseScale(t *testing.T) {
	bg := NewBackgroundRenderer()

	bg.SetNoiseScale(0.03)
	assert.Equal(t, 0.03, bg.noiseScale)
}

func TestBackgroundRendererSetNoiseAmplitude(t *testing.T) {
	bg := NewBackgroundRenderer()

	bg.SetNoiseAmplitude(0.5)
	assert.Equal(t, 0.5, bg.noiseAmplitude)
}

func TestBackgroundRendererGetBaseColor(t *testing.T) {
	bg := NewBackgroundRenderer()

	color := bg.GetBaseColor()
	assert.Equal(t, uint8(11), color.R)
	assert.Equal(t, uint8(14), color.G)
	assert.Equal(t, uint8(21), color.B)
	assert.Equal(t, uint8(255), color.A)
}
