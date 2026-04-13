// Package overlays provides stub types for testing without Ebitengine.
// This file is used when building with the noebiten tag.
//
//go:build noebiten
// +build noebiten

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
	alive := e.Particles[:0]
	for i := range e.Particles {
		p := &e.Particles[i]
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Life -= dt / p.MaxLife
		if p.Life > 0 {
			alive = append(alive, *p)
		}
	}
	e.Particles = alive

	adjustedRate := e.EmitRate * (1.0 + resonance/100.0)
	e.accumulator += dt * adjustedRate

	for e.accumulator >= 1.0 && len(e.Particles) < e.MaxParticles {
		e.accumulator -= 1.0
		angle := float32(len(e.Particles)%360) * 0.0175
		e.Particles = append(e.Particles, SpecterParticle{
			X:       nodeX + nodeRadius*cos(angle),
			Y:       nodeY + nodeRadius*sin(angle),
			VX:      cos(angle) * 10,
			VY:      sin(angle)*5 - 15,
			Life:    1.0,
			MaxLife: 2.0 + resonance/200.0,
			Size:    2.0 + resonance/50.0,
			Color:   color.RGBA{200, 220, 255, 200},
		})
	}
}

func cos(angle float32) float32 {
	return float32(1.0) - angle*angle/2.0 + angle*angle*angle*angle/24.0
}

func sin(angle float32) float32 {
	return angle - angle*angle*angle/6.0 + angle*angle*angle*angle*angle/120.0
}

// DuelVisualization represents a Specter Duel.
type DuelVisualization struct {
	Duelist1X, Duelist1Y float32
	Duelist2X, Duelist2Y float32
	Color1, Color2       color.RGBA
	Intensity            float32
	Phase                float32
}
