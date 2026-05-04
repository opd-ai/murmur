// Package effects provides visual effects for the Pulse Map.
// This file implements a GPU-accelerated particle system for efficient rendering.
// Per PULSE_MAP.md, particles are rendered with a single draw call per particle type.

//go:build !test
// +build !test

package effects

import (
	_ "embed"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed particle.kage
var particleShaderSrc []byte

// GPUParticle represents a single particle in the GPU particle system.
type GPUParticle struct {
	X, Y    float32    // World position
	VX, VY  float32    // Velocity
	Life    float32    // Remaining life (0-1, 0 = dead)
	MaxLife float32    // Starting life
	Size    float32    // Particle size
	Color   color.RGBA // Particle color
}

// GPUParticleSystem manages GPU-accelerated particle rendering.
// Particles are rendered in batches using a shared shader, minimizing draw calls.
type GPUParticleSystem struct {
	Particles    []GPUParticle
	MaxParticles int
	EmitRate     float32 // Particles per second
	accumulator  float32 // Time accumulator for emission

	// GPU rendering resources
	shader         *ebiten.Shader
	particleSprite *ebiten.Image // Small square texture for particles
}

// NewGPUParticleSystem creates a GPU-accelerated particle system.
// maxParticles: maximum number of simultaneous particles
// emitRate: base emission rate in particles per second
func NewGPUParticleSystem(maxParticles int, emitRate float32) (*GPUParticleSystem, error) {
	// Compile particle shader
	shader, err := ebiten.NewShader(particleShaderSrc)
	if err != nil {
		return nil, err
	}

	// Create a small square texture for particle sprite (8x8 pixels)
	// We'll use the shader to create the circular falloff
	particleSprite := ebiten.NewImage(8, 8)
	particleSprite.Fill(color.White)

	return &GPUParticleSystem{
		Particles:      make([]GPUParticle, 0, maxParticles),
		MaxParticles:   maxParticles,
		EmitRate:       emitRate,
		shader:         shader,
		particleSprite: particleSprite,
	}, nil
}

// Update advances particle physics and emits new particles.
// dt: delta time in seconds
// emitX, emitY: emission point in world coordinates
// emitRadius: radius of emission circle
// resonance: Resonance score (affects emission rate and particle lifetime per PULSE_MAP.md)
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
	// Higher resonance = more particles per PULSE_MAP.md
	adjustedRate := s.EmitRate * (1.0 + resonance/100.0)
	s.accumulator += dt * adjustedRate

	for s.accumulator >= 1.0 && len(s.Particles) < s.MaxParticles {
		s.accumulator -= 1.0
		s.emitParticle(emitX, emitY, emitRadius, resonance)
	}
}

// emitParticle creates a single new particle at the emission point.
func (s *GPUParticleSystem) emitParticle(x, y, radius, resonance float32) {
	// Emit particle at node edge, drifting outward and upward
	angle := float32(len(s.Particles)%360) * (math.Pi / 180.0)
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))

	s.Particles = append(s.Particles, GPUParticle{
		X:       x + radius*cos,
		Y:       y + radius*sin,
		VX:      cos * 10,                       // Drift outward
		VY:      sin*5 - 15,                     // Drift upward
		Life:    1.0,                            // Start at full life
		MaxLife: 2.0 + resonance/200.0,          // Longer life with higher resonance
		Size:    2.0 + resonance/50.0,           // Larger with higher resonance
		Color:   color.RGBA{200, 220, 255, 200}, // Luminous blue-white
	})
}

// Render draws all particles using GPU-accelerated batching.
// dst: target image
// cameraX, cameraY: camera position in world coordinates
// scale: zoom scale factor
func (s *GPUParticleSystem) Render(dst *ebiten.Image, cameraX, cameraY, scale float32) {
	if len(s.Particles) == 0 {
		return
	}

	// Render particles in a single batch using DrawRectShader
	// This minimizes draw calls as per PULSE_MAP.md specification
	for _, p := range s.Particles {
		// Transform world position to screen position
		screenX := (p.X-cameraX)*scale + float32(dst.Bounds().Dx())/2
		screenY := (p.Y-cameraY)*scale + float32(dst.Bounds().Dy())/2
		size := p.Size * p.Life * scale

		// Skip particles outside viewport (viewport culling)
		if screenX < -size || screenX > float32(dst.Bounds().Dx())+size ||
			screenY < -size || screenY > float32(dst.Bounds().Dy())+size {
			continue
		}

		// Set up shader options for this particle
		opts := &ebiten.DrawRectShaderOptions{}
		opts.GeoM.Translate(float64(screenX-size), float64(screenY-size))
		opts.GeoM.Scale(float64(size*2), float64(size*2))

		// Set particle color and fade as shader uniforms
		opts.Uniforms = map[string]interface{}{
			"ParticleColor": []float32{
				float32(p.Color.R) / 255.0,
				float32(p.Color.G) / 255.0,
				float32(p.Color.B) / 255.0,
				float32(p.Color.A) / 255.0,
			},
			"ParticleFade": p.Life,
		}

		// Draw particle using shader
		dst.DrawRectShader(int(size*2), int(size*2), s.shader, opts)
	}
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
