// Package effects provides visual effects for the Pulse Map.
// This file provides stub implementations for GPU particle system in test builds.

//go:build test
// +build test

package effects

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// GPUParticle represents a single particle in the GPU particle system.
type GPUParticle struct {
	X, Y    float32
	VX, VY  float32
	Life    float32
	MaxLife float32
	Size    float32
	Color   color.RGBA
}

// GPUParticleSystem manages GPU-accelerated particle rendering.
type GPUParticleSystem struct {
	Particles    []GPUParticle
	MaxParticles int
	EmitRate     float32
	accumulator  float32
}

// NewGPUParticleSystem creates a GPU-accelerated particle system (stub).
func NewGPUParticleSystem(maxParticles int, emitRate float32) (*GPUParticleSystem, error) {
	return &GPUParticleSystem{
		Particles:    make([]GPUParticle, 0, maxParticles),
		MaxParticles: maxParticles,
		EmitRate:     emitRate,
	}, nil
}

// Update advances particle physics (CPU-based in test build).
func (s *GPUParticleSystem) Update(dt, emitX, emitY, emitRadius, resonance float32) {
	s.updateImpl(dt, emitX, emitY, emitRadius, resonance)
}

// updateImpl implements the particle update logic.
// Shared between gpu_particles.go and gpu_particles_stub.go.
func (s *GPUParticleSystem) updateImpl(dt, emitX, emitY, emitRadius, resonance float32) {
	// Update existing particles
	alive := s.Particles[:0]
	for i := range s.Particles {
		p := &s.Particles[i]
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Life -= dt / p.MaxLife
		if p.Life > 0 {
			alive = append(alive, *p)
		}
	}
	s.Particles = alive

	// Emit new particles based on resonance
	adjustedRate := s.EmitRate * (1.0 + resonance/100.0)
	s.accumulator += dt * adjustedRate

	for s.accumulator >= 1.0 && len(s.Particles) < s.MaxParticles {
		s.accumulator -= 1.0
		s.emitParticle(emitX, emitY, emitRadius, resonance)
	}
}

// emitParticle creates a single new particle at the emission point.
func (s *GPUParticleSystem) emitParticle(x, y, radius, resonance float32) {
	angle := float32(len(s.Particles)%360) * (math.Pi / 180.0)
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))

	s.Particles = append(s.Particles, GPUParticle{
		X:       x + radius*cos,
		Y:       y + radius*sin,
		VX:      cos * 10,
		VY:      sin*5 - 15,
		Life:    1.0,
		MaxLife: 2.0 + resonance/200.0,
		Size:    2.0 + resonance/50.0,
		Color:   color.RGBA{200, 220, 255, 200},
	})
}

// Render draws all particles (no-op in test build - no GPU rendering).
func (s *GPUParticleSystem) Render(dst *ebiten.Image, cameraX, cameraY, scale float32) {
	// No-op in test build - GPU rendering not available
}

// Clear removes all particles from the system.
func (s *GPUParticleSystem) Clear() {
	s.Particles = s.Particles[:0]
	s.accumulator = 0
}

// ParticleCount returns the current number of active particles.
func (s *GPUParticleSystem) ParticleCount() int {
	return len(s.Particles)
}
