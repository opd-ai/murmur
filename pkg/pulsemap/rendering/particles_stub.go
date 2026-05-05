// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// Test stub for ambient particles.
//
//go:build test
// +build test

package rendering

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// AmbientParticle test stub.
type AmbientParticle struct {
	X, Y float64
}

// AmbientParticleField test stub.
type AmbientParticleField struct {
	particles    []AmbientParticle
	maxParticles int
	spawnRate    float64
}

// NewAmbientParticleField test stub.
func NewAmbientParticleField() *AmbientParticleField {
	return &AmbientParticleField{
		particles:    make([]AmbientParticle, 0),
		maxParticles: 80,
		spawnRate:    2.0,
	}
}

// Update test stub.
func (f *AmbientParticleField) Update(dt, cameraX, cameraY float64, screenW, screenH int) {
	// No-op for tests.
}

// Draw test stub.
func (f *AmbientParticleField) Draw(screen *ebiten.Image, cameraX, cameraY float64) {
	// No-op for tests.
}

// SetMaxParticles test stub.
func (f *AmbientParticleField) SetMaxParticles(max int) {
	f.maxParticles = max
}

// SetSpawnRate test stub.
func (f *AmbientParticleField) SetSpawnRate(rate float64) {
	f.spawnRate = rate
}

// ParticleCount test stub.
func (f *AmbientParticleField) ParticleCount() int {
	return len(f.particles)
}
