// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file contains platform-independent particle physics shared by both
// ebiten and noebiten builds.

package overlays

import (
	"image/color"
)

// particleCos returns cosine approximation using Taylor series.
// Uses first three terms: 1 - x²/2 + x⁴/24
func particleCos(angle float32) float32 {
	return float32(1.0) - angle*angle/2.0 + angle*angle*angle*angle/24.0
}

// particleSin returns sine approximation using Taylor series.
// Uses first three terms: x - x³/6 + x⁵/120
func particleSin(angle float32) float32 {
	return angle - angle*angle*angle/6.0 + angle*angle*angle*angle*angle/120.0
}

// updateParticles advances particle physics and emits new particles.
// This is the shared implementation called by ParticleEmitter.Update() in
// both ebiten and noebiten builds.
func updateParticles(e *ParticleEmitter, dt, nodeX, nodeY, nodeRadius, resonance float32) {
	// Update existing particles
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

	// Emit new particles based on resonance
	// Higher resonance = more particles per PULSE_MAP.md
	adjustedRate := e.EmitRate * (1.0 + resonance/100.0)
	e.accumulator += dt * adjustedRate

	for e.accumulator >= 1.0 && len(e.Particles) < e.MaxParticles {
		e.accumulator -= 1.0
		// Emit particle at node edge, drifting outward and upward
		angle := float32(len(e.Particles)%360) * 0.0175 // Radians
		e.Particles = append(e.Particles, SpecterParticle{
			X:       nodeX + nodeRadius*particleCos(angle),
			Y:       nodeY + nodeRadius*particleSin(angle),
			VX:      particleCos(angle) * 10,
			VY:      particleSin(angle)*5 - 15, // Drift upward
			Life:    1.0,
			MaxLife: 2.0 + resonance/200.0, // Longer life with higher resonance
			Size:    2.0 + resonance/50.0,
			Color:   color.RGBA{200, 220, 255, 200}, // Luminous blue-white
		})
	}
}
