// Package effects provides milestone visual effects stubs for non-Ebitengine builds.
//
//go:build noebiten
// +build noebiten

package effects

import "image/color"

// SurfaceMilestone represents a Surface Resonance milestone.
type SurfaceMilestone int

const (
	MilestoneNone    SurfaceMilestone = 0
	MilestoneEmber   SurfaceMilestone = 10
	MilestoneSpark   SurfaceMilestone = 25
	MilestoneFlame   SurfaceMilestone = 50
	MilestoneBlaze   SurfaceMilestone = 100
	MilestoneInferno SurfaceMilestone = 200
	MilestoneCorona  SurfaceMilestone = 500
)

// SpecterMilestone represents a Specter Resonance milestone.
type SpecterMilestone int

const (
	SpecterMilestoneNone        SpecterMilestone = 0
	SpecterMilestoneWhisper     SpecterMilestone = 10
	SpecterMilestoneShade       SpecterMilestone = 25
	SpecterMilestoneWraith      SpecterMilestone = 50
	SpecterMilestoneShadeWraith SpecterMilestone = 75
	SpecterMilestonePhantom     SpecterMilestone = 100
	SpecterMilestoneRevenant    SpecterMilestone = 200
	SpecterMilestoneAbyss       SpecterMilestone = 500
)

// SurfaceMilestoneFromScore returns the milestone for a given resonance score.
func SurfaceMilestoneFromScore(score int) SurfaceMilestone {
	switch {
	case score >= 500:
		return MilestoneCorona
	case score >= 200:
		return MilestoneInferno
	case score >= 100:
		return MilestoneBlaze
	case score >= 50:
		return MilestoneFlame
	case score >= 25:
		return MilestoneSpark
	case score >= 10:
		return MilestoneEmber
	default:
		return MilestoneNone
	}
}

// SpecterMilestoneFromScore returns the milestone for a given Specter resonance score.
func SpecterMilestoneFromScore(score int) SpecterMilestone {
	switch {
	case score >= 500:
		return SpecterMilestoneAbyss
	case score >= 200:
		return SpecterMilestoneRevenant
	case score >= 100:
		return SpecterMilestonePhantom
	case score >= 75:
		return SpecterMilestoneShadeWraith
	case score >= 50:
		return SpecterMilestoneWraith
	case score >= 25:
		return SpecterMilestoneShade
	case score >= 10:
		return SpecterMilestoneWhisper
	default:
		return SpecterMilestoneNone
	}
}

// MilestoneEffects renders visual effects based on resonance milestones.
type MilestoneEffects struct {
	flameParticles   []milestoneParticle
	coronaParticles  []milestoneParticle
	specterParticles []milestoneParticle
	time             float32
}

// milestoneParticle represents a single particle in a milestone effect.
type milestoneParticle struct {
	X, Y     float32
	VX, VY   float32
	Life     float32
	MaxLife  float32
	Size     float32
	Color    color.RGBA
	Angle    float32
	AngleVel float32
}

// NewMilestoneEffects creates a new milestone effects renderer.
func NewMilestoneEffects() *MilestoneEffects {
	return &MilestoneEffects{
		flameParticles:   make([]milestoneParticle, 0, 32),
		coronaParticles:  make([]milestoneParticle, 0, 64),
		specterParticles: make([]milestoneParticle, 0, 32),
	}
}

// Update advances all milestone effect animations.
func (m *MilestoneEffects) Update(dt float32) {
	m.time += dt

	m.updateParticles(&m.flameParticles, dt)
	m.updateParticles(&m.coronaParticles, dt)
	m.updateParticles(&m.specterParticles, dt)
}

// updateParticles advances particle physics.
func (m *MilestoneEffects) updateParticles(particles *[]milestoneParticle, dt float32) {
	alive := (*particles)[:0]
	for _, p := range *particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Angle += p.AngleVel * dt
		alive = append(alive, p)
	}
	*particles = alive
}

// GetTime returns the current animation time (for testing).
func (m *MilestoneEffects) GetTime() float32 {
	return m.time
}

// FlameParticleCount returns the number of flame particles (for testing).
func (m *MilestoneEffects) FlameParticleCount() int {
	return len(m.flameParticles)
}

// CoronaParticleCount returns the number of corona particles (for testing).
func (m *MilestoneEffects) CoronaParticleCount() int {
	return len(m.coronaParticles)
}

// SpecterParticleCount returns the number of specter particles (for testing).
func (m *MilestoneEffects) SpecterParticleCount() int {
	return len(m.specterParticles)
}
