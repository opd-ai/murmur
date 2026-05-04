// Package overlays provides stub types for testing without Ebitengine.
// This file is used when building with the test tag.
//
//go:build test
// +build test

package overlays

import (
	"image/color"
)

// LayerBlend controls the visibility blending between Surface and Anonymous layers.
type LayerBlend struct {
	SurfaceOpacity   float32
	AnonymousOpacity float32
	IsFortressMode   bool
}

// NewDefaultBlend creates a default layer blend.
func NewDefaultBlend() *LayerBlend {
	return &LayerBlend{
		SurfaceOpacity:   1.0,
		AnonymousOpacity: 0.5,
		IsFortressMode:   false,
	}
}

// NewFortressBlend creates a Fortress mode blend.
func NewFortressBlend() *LayerBlend {
	return &LayerBlend{
		SurfaceOpacity:   0.0,
		AnonymousOpacity: 1.0,
		IsFortressMode:   true,
	}
}

// SetBlendRatio adjusts the blend between layers.
func (b *LayerBlend) SetBlendRatio(ratio float32) {
	if b.IsFortressMode {
		return
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	b.SurfaceOpacity = 1 - ratio
	b.AnonymousOpacity = ratio
}

// SpecterParticle represents an animated particle.
type SpecterParticle struct {
	X, Y    float32
	VX, VY  float32
	Life    float32
	MaxLife float32
	Size    float32
	Color   color.RGBA
}

// ParticleEmitter manages particle emission.
type ParticleEmitter struct {
	Particles    []SpecterParticle
	MaxParticles int
	EmitRate     float32
	accumulator  float32
}

// NewParticleEmitter creates a new particle emitter.
func NewParticleEmitter(maxParticles int, emitRate float32) *ParticleEmitter {
	return &ParticleEmitter{
		Particles:    make([]SpecterParticle, 0, maxParticles),
		MaxParticles: maxParticles,
		EmitRate:     emitRate,
	}
}

// Update advances particle physics.
func (e *ParticleEmitter) Update(dt, nodeX, nodeY, nodeRadius, resonance float32) {
	updateParticles(e, dt, nodeX, nodeY, nodeRadius, resonance)
}

func cos(angle float32) float32 {
	return particleCos(angle)
}

func sin(angle float32) float32 {
	return particleSin(angle)
}

// MiniGameVisualization represents a mini-game event.
type MiniGameVisualization struct {
	Player1X, Player1Y float32
	Player2X, Player2Y float32
	Color1, Color2     color.RGBA
	Intensity          float32
	Phase              float32
}
