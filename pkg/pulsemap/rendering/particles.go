// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file implements ambient particle effects for atmospheric depth.
//
//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AmbientParticle represents a single drifting particle.
type AmbientParticle struct {
	X, Y     float64 // World position
	VX, VY   float64 // Velocity
	Size     float32 // Particle radius
	Alpha    uint8   // Transparency (0-255)
	Depth    float64 // Parallax depth factor (0.5-1.0, smaller = further away)
	LifeTime float64 // Time since creation (for fade-in)
	MaxAlpha uint8   // Target alpha after fade-in
}

// AmbientParticleField manages sparse drifting particles for atmospheric effect.
// Per ROADMAP.md line 687, this creates a subtle particle field for visual depth.
type AmbientParticleField struct {
	particles []AmbientParticle

	// Particle spawn parameters.
	maxParticles int     // Maximum number of particles on screen
	spawnRate    float64 // Particles to spawn per second
	spawnTimer   float64 // Accumulator for spawn timing

	// Visual parameters.
	baseColor color.RGBA // Base particle color (light blue-gray)
	minSize   float32    // Minimum particle size
	maxSize   float32    // Maximum particle size
	minSpeed  float64    // Minimum drift speed
	maxSpeed  float64    // Maximum drift speed

	// Screen bounds for culling.
	screenWidth  int
	screenHeight int

	// Camera offset for parallax effect.
	cameraX float64
	cameraY float64

	// Random source for particle generation.
	rng *rand.Rand
}

// NewAmbientParticleField creates a new particle field with default settings.
func NewAmbientParticleField() *AmbientParticleField {
	return &AmbientParticleField{
		particles:    make([]AmbientParticle, 0, 100),
		maxParticles: 80,                            // Sparse field per spec
		spawnRate:    2.0,                           // 2 particles per second
		baseColor:    color.RGBA{120, 140, 160, 30}, // Very subtle light blue-gray
		minSize:      1.0,
		maxSize:      2.5,
		minSpeed:     5.0, // Slow drift
		maxSpeed:     15.0,
		rng:          rand.New(rand.NewSource(42)), // Deterministic for consistency
	}
}

// Update advances particle simulation by one tick (1/60 second).
func (f *AmbientParticleField) Update(dt, cameraX, cameraY float64, screenW, screenH int) {
	f.screenWidth = screenW
	f.screenHeight = screenH
	f.cameraX = cameraX
	f.cameraY = cameraY

	// Update existing particles.
	active := f.particles[:0]
	for i := range f.particles {
		p := &f.particles[i]

		// Update position based on velocity and parallax depth.
		p.X += p.VX * dt * p.Depth
		p.Y += p.VY * dt * p.Depth

		// Fade in over first 0.5 seconds.
		p.LifeTime += dt
		if p.LifeTime < 0.5 {
			fadeProgress := p.LifeTime / 0.5
			p.Alpha = uint8(float64(p.MaxAlpha) * fadeProgress)
		} else {
			p.Alpha = p.MaxAlpha
		}

		// Transform to screen space with parallax.
		screenX := (p.X - cameraX*p.Depth) * p.Depth
		screenY := (p.Y - cameraY*p.Depth) * p.Depth

		// Cull particles outside extended screen bounds (margin for wrapping).
		margin := 100.0
		if screenX < -margin || screenX > float64(screenW)+margin ||
			screenY < -margin || screenY > float64(screenH)+margin {
			continue // Discard this particle
		}

		active = append(active, *p)
	}
	f.particles = active

	// Spawn new particles.
	f.spawnTimer += dt
	spawnInterval := 1.0 / f.spawnRate
	for f.spawnTimer >= spawnInterval && len(f.particles) < f.maxParticles {
		f.spawnTimer -= spawnInterval
		f.spawnParticle()
	}
}

// spawnParticle creates a new particle at a random position.
func (f *AmbientParticleField) spawnParticle() {
	// Spawn at screen edges with camera-relative world position.
	edge := f.rng.Intn(4) // 0=top, 1=right, 2=bottom, 3=left
	var x, y float64

	margin := 50.0
	switch edge {
	case 0: // Top
		x = f.rng.Float64() * float64(f.screenWidth)
		y = -margin
	case 1: // Right
		x = float64(f.screenWidth) + margin
		y = f.rng.Float64() * float64(f.screenHeight)
	case 2: // Bottom
		x = f.rng.Float64() * float64(f.screenWidth)
		y = float64(f.screenHeight) + margin
	case 3: // Left
		x = -margin
		y = f.rng.Float64() * float64(f.screenHeight)
	}

	// Convert screen position to world position with parallax.
	depth := 0.5 + f.rng.Float64()*0.5 // Depth range [0.5, 1.0]
	worldX := x/depth + f.cameraX*depth
	worldY := y/depth + f.cameraY*depth

	// Random velocity (slow drift).
	speed := f.minSpeed + f.rng.Float64()*(f.maxSpeed-f.minSpeed)
	angle := f.rng.Float64() * 2 * math.Pi
	vx := math.Cos(angle) * speed
	vy := math.Sin(angle) * speed

	// Random size and alpha.
	size := f.minSize + f.rng.Float32()*(f.maxSize-f.minSize)
	maxAlpha := uint8(15 + f.rng.Intn(20)) // Very subtle: 15-35 alpha

	f.particles = append(f.particles, AmbientParticle{
		X:        worldX,
		Y:        worldY,
		VX:       vx,
		VY:       vy,
		Size:     size,
		Alpha:    0, // Start transparent for fade-in
		Depth:    depth,
		LifeTime: 0,
		MaxAlpha: maxAlpha,
	})
}

// Draw renders all particles to the screen.
func (f *AmbientParticleField) Draw(screen *ebiten.Image, cameraX, cameraY float64) {
	for i := range f.particles {
		p := &f.particles[i]

		// Transform to screen space with parallax.
		screenX := float32((p.X-cameraX*p.Depth)*p.Depth + float64(f.screenWidth)/2)
		screenY := float32((p.Y-cameraY*p.Depth)*p.Depth + float64(f.screenHeight)/2)

		// Skip if off-screen.
		if screenX < 0 || screenX > float32(f.screenWidth) ||
			screenY < 0 || screenY > float32(f.screenHeight) {
			continue
		}

		// Draw particle as filled circle with alpha.
		particleColor := color.RGBA{
			R: f.baseColor.R,
			G: f.baseColor.G,
			B: f.baseColor.B,
			A: p.Alpha,
		}

		vector.DrawFilledCircle(screen, screenX, screenY, p.Size, particleColor, true)
	}
}

// SetMaxParticles adjusts the maximum number of particles on screen.
func (f *AmbientParticleField) SetMaxParticles(max int) {
	f.maxParticles = max
}

// SetSpawnRate adjusts how many particles spawn per second.
func (f *AmbientParticleField) SetSpawnRate(rate float64) {
	f.spawnRate = rate
}

// ParticleCount returns the current number of active particles.
func (f *AmbientParticleField) ParticleCount() int {
	return len(f.particles)
}
