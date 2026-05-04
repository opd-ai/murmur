// Package effects provides milestone visual effects for the Pulse Map.
// Per RESONANCE_SYSTEM.md, Surface milestones unlock visual effects:
// - Ember (10) — warm glow
// - Spark (25) — pulsing ring
// - Flame (50) — particle trail
// - Blaze (100) — custom color palette
// - Inferno (200) — animated aura
// - Corona (500) — multi-layered corona
//
// Specter milestones have distinct effects:
// - Whisper (10) — ghostly trail
// - Shade (25) — shadow effect
// - Wraith (50) — wispy tendrils
// - Shade-Wraith (75) — spectral glow
// - Phantom (100) — mask overlay
// - Revenant (200) — ethereal aura
// - Abyss (500) — void shader (Fortress only)
//

package effects

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	// Particle systems for milestone effects.
	flameParticles   []milestoneParticle
	coronaParticles  []milestoneParticle
	specterParticles []milestoneParticle

	// Animation state.
	time float32
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

// DrawSurfaceMilestone renders the visual effect for a Surface milestone.
func (m *MilestoneEffects) DrawSurfaceMilestone(dst *ebiten.Image, x, y, radius float32, milestone SurfaceMilestone) {
	switch milestone {
	case MilestoneEmber:
		m.drawEmberGlow(dst, x, y, radius)
	case MilestoneSpark:
		m.drawSparkRing(dst, x, y, radius)
	case MilestoneFlame:
		m.drawFlameTrail(dst, x, y, radius)
	case MilestoneBlaze:
		m.drawBlazeAura(dst, x, y, radius)
	case MilestoneInferno:
		m.drawInfernoAura(dst, x, y, radius)
	case MilestoneCorona:
		m.drawCorona(dst, x, y, radius)
	}
}

// DrawSpecterMilestone renders the visual effect for a Specter milestone.
func (m *MilestoneEffects) DrawSpecterMilestone(dst *ebiten.Image, x, y, radius float32, milestone SpecterMilestone, isFortress bool) {
	switch milestone {
	case SpecterMilestoneWhisper:
		m.drawWhisperTrail(dst, x, y, radius)
	case SpecterMilestoneShade:
		m.drawShadeEffect(dst, x, y, radius)
	case SpecterMilestoneWraith:
		m.drawWraithTendrils(dst, x, y, radius)
	case SpecterMilestoneShadeWraith:
		m.drawShadeWraithGlow(dst, x, y, radius)
	case SpecterMilestonePhantom:
		m.drawPhantomMask(dst, x, y, radius)
	case SpecterMilestoneRevenant:
		m.drawRevenantAura(dst, x, y, radius)
	case SpecterMilestoneAbyss:
		if isFortress {
			m.drawAbyssVoid(dst, x, y, radius)
		}
	}
}

// --- Surface Milestone Effects ---

// drawEmberGlow renders the Ember (10) warm glow effect.
func (m *MilestoneEffects) drawEmberGlow(dst *ebiten.Image, x, y, radius float32) {
	// Warm orange glow
	pulse := 1.0 + 0.2*float32(math.Sin(float64(m.time*2)))
	glowRadius := radius * 1.5 * pulse
	glowColor := color.RGBA{255, 150, 50, 60}

	// Draw multiple layers for soft glow
	for i := 0; i < 3; i++ {
		layer := float32(i + 1)
		r := glowRadius * (1.0 - layer*0.2)
		alpha := uint8(60 / layer)
		c := color.RGBA{glowColor.R, glowColor.G, glowColor.B, alpha}
		vector.DrawFilledCircle(dst, x, y, r, c, true)
	}
}

// drawSparkRing renders the Spark (25) pulsing ring animation.
func (m *MilestoneEffects) drawSparkRing(dst *ebiten.Image, x, y, radius float32) {
	// Pulsing ring effect
	ringPhase := m.time * 3
	ringRadius := radius * (1.2 + 0.3*float32(math.Sin(float64(ringPhase))))
	ringColor := color.RGBA{255, 220, 100, 150}

	strokeWidth := float32(2.0 + math.Sin(float64(ringPhase))*0.5)
	vector.StrokeCircle(dst, x, y, ringRadius, strokeWidth, ringColor, true)

	// Secondary inner ring
	innerRadius := ringRadius * 0.7
	innerColor := color.RGBA{255, 200, 50, 100}
	vector.StrokeCircle(dst, x, y, innerRadius, 1.5, innerColor, true)
}

// drawFlameTrail renders the Flame (50) particle trail effect.
func (m *MilestoneEffects) drawFlameTrail(dst *ebiten.Image, x, y, radius float32) {
	// Emit flame particles
	if len(m.flameParticles) < 32 {
		angle := m.time * 2
		for i := 0; i < 2; i++ {
			p := milestoneParticle{
				X:       x + float32(math.Cos(float64(angle+float32(i))))*radius*0.5,
				Y:       y + float32(math.Sin(float64(angle+float32(i))))*radius*0.5,
				VX:      float32(math.Cos(float64(angle))) * 20,
				VY:      -30 - float32(i)*10, // Flames rise
				Life:    0.8,
				MaxLife: 0.8,
				Size:    5 + float32(i)*2,
				Color:   color.RGBA{255, 150, 50, 200},
			}
			m.flameParticles = append(m.flameParticles, p)
		}
	}

	// Draw flame particles
	for _, p := range m.flameParticles {
		lifeRatio := p.Life / p.MaxLife
		alpha := uint8(float32(p.Color.A) * lifeRatio)
		// Color shifts from orange to yellow as it rises
		r := uint8(255)
		g := uint8(150 + 100*(1-lifeRatio))
		b := uint8(50 * (1 - lifeRatio))
		c := color.RGBA{r, g, b, alpha}
		size := p.Size * lifeRatio
		vector.DrawFilledCircle(dst, p.X, p.Y, size, c, true)
	}
}

// drawBlazeAura renders the Blaze (100) custom color palette effect.
func (m *MilestoneEffects) drawBlazeAura(dst *ebiten.Image, x, y, radius float32) {
	// Multi-colored flame aura
	colors := []color.RGBA{
		{255, 100, 0, 80}, // Orange
		{255, 200, 0, 60}, // Yellow
		{255, 50, 50, 70}, // Red
	}

	for i, c := range colors {
		phase := m.time*2 + float32(i)*1.5
		r := radius * (1.4 + 0.2*float32(math.Sin(float64(phase))))
		vector.DrawFilledCircle(dst, x, y, r*(1.0-float32(i)*0.15), c, true)
	}
}

// drawInfernoAura renders the Inferno (200) animated aura effect.
func (m *MilestoneEffects) drawInfernoAura(dst *ebiten.Image, x, y, radius float32) {
	// Intense animated fire aura
	numRays := 12
	for i := 0; i < numRays; i++ {
		angle := float32(i)*float32(math.Pi)*2/float32(numRays) + m.time
		rayLength := radius * (1.0 + 0.5*float32(math.Sin(float64(angle*3+m.time*5))))

		x2 := x + float32(math.Cos(float64(angle)))*rayLength
		y2 := y + float32(math.Sin(float64(angle)))*rayLength

		intensity := 0.5 + 0.5*float32(math.Sin(float64(angle+m.time*4)))
		c := color.RGBA{255, uint8(150 * intensity), 0, uint8(150 * intensity)}
		vector.StrokeLine(dst, x, y, x2, y2, 2, c, true)
	}

	// Core glow
	coreColor := color.RGBA{255, 220, 100, 100}
	vector.DrawFilledCircle(dst, x, y, radius*1.2, coreColor, true)
}

// drawCorona renders the Corona (500) multi-layered corona effect.
func (m *MilestoneEffects) drawCorona(dst *ebiten.Image, x, y, radius float32) {
	// Multiple expanding rings
	numRings := 4
	for i := 0; i < numRings; i++ {
		phase := m.time*2 - float32(i)*0.5
		ringRadius := radius * (1.5 + 0.5*float32(math.Sin(float64(phase))))
		alpha := uint8(80 - i*15)
		c := color.RGBA{255, 200, 100, alpha}
		vector.StrokeCircle(dst, x, y, ringRadius*(1.0+float32(i)*0.2), 2, c, true)
	}

	// Corona rays
	numRays := 16
	for i := 0; i < numRays; i++ {
		angle := float32(i)*float32(math.Pi)*2/float32(numRays) + m.time*0.5
		rayLength := radius * (2.0 + 0.5*float32(math.Sin(float64(angle*2+m.time*3))))

		x2 := x + float32(math.Cos(float64(angle)))*rayLength
		y2 := y + float32(math.Sin(float64(angle)))*rayLength

		c := color.RGBA{255, 220, 150, 100}
		vector.StrokeLine(dst, x, y, x2, y2, 1.5, c, true)
	}

	// Emit corona particles
	if len(m.coronaParticles) < 64 {
		angle := m.time * 5
		p := milestoneParticle{
			X:       x + float32(math.Cos(float64(angle)))*radius,
			Y:       y + float32(math.Sin(float64(angle)))*radius,
			VX:      float32(math.Cos(float64(angle))) * 40,
			VY:      float32(math.Sin(float64(angle))) * 40,
			Life:    1.5,
			MaxLife: 1.5,
			Size:    3,
			Color:   color.RGBA{255, 220, 150, 180},
		}
		m.coronaParticles = append(m.coronaParticles, p)
	}

	for _, p := range m.coronaParticles {
		lifeRatio := p.Life / p.MaxLife
		alpha := uint8(float32(p.Color.A) * lifeRatio)
		c := color.RGBA{p.Color.R, p.Color.G, p.Color.B, alpha}
		vector.DrawFilledCircle(dst, p.X, p.Y, p.Size*lifeRatio, c, true)
	}
}

// --- Specter Milestone Effects ---

// drawWhisperTrail renders the Whisper (10) ghostly trail effect.
func (m *MilestoneEffects) drawWhisperTrail(dst *ebiten.Image, x, y, radius float32) {
	// Faint trailing wisps
	numWisps := 6
	for i := 0; i < numWisps; i++ {
		angle := m.time*0.5 + float32(i)*float32(math.Pi)/3
		dist := radius * (0.5 + 0.3*float32(math.Sin(float64(angle*2))))
		wx := x + float32(math.Cos(float64(angle)))*dist
		wy := y + float32(math.Sin(float64(angle)))*dist

		alpha := uint8(30 + 20*float32(math.Sin(float64(angle+m.time*2))))
		c := color.RGBA{150, 200, 255, alpha}
		vector.DrawFilledCircle(dst, wx, wy, 4, c, true)
	}
}

// drawShadeEffect renders the Shade (25) shadow effect.
func (m *MilestoneEffects) drawShadeEffect(dst *ebiten.Image, x, y, radius float32) {
	// Dark shadow aura
	pulse := 1.0 + 0.1*float32(math.Sin(float64(m.time*2)))
	shadowRadius := radius * 1.4 * pulse
	shadowColor := color.RGBA{50, 50, 80, 50}

	vector.DrawFilledCircle(dst, x+3, y+3, shadowRadius, shadowColor, true)
}

// drawWraithTendrils renders the Wraith (50) wispy tendril effect.
func (m *MilestoneEffects) drawWraithTendrils(dst *ebiten.Image, x, y, radius float32) {
	// Wispy ethereal tendrils
	numTendrils := 8
	for i := 0; i < numTendrils; i++ {
		baseAngle := float32(i) * float32(math.Pi) * 2 / float32(numTendrils)
		angle := baseAngle + m.time*0.3

		// Wavy tendril
		segments := 6
		prevX, prevY := x, y
		for j := 1; j <= segments; j++ {
			t := float32(j) / float32(segments)
			dist := radius * t * 1.5
			wave := float32(math.Sin(float64(angle+t*5+m.time*3))) * 10 * t
			tx := x + float32(math.Cos(float64(angle)))*dist + wave
			ty := y + float32(math.Sin(float64(angle)))*dist + wave*0.5

			alpha := uint8(80 * (1 - t))
			c := color.RGBA{100, 150, 200, alpha}
			vector.StrokeLine(dst, prevX, prevY, tx, ty, 2*(1-t)+0.5, c, true)
			prevX, prevY = tx, ty
		}
	}
}

// drawShadeWraithGlow renders the Shade-Wraith (75) spectral glow effect.
func (m *MilestoneEffects) drawShadeWraithGlow(dst *ebiten.Image, x, y, radius float32) {
	// Combined shadow and spectral glow
	m.drawShadeEffect(dst, x, y, radius)

	// Spectral inner glow
	pulse := 1.0 + 0.15*float32(math.Sin(float64(m.time*3)))
	glowRadius := radius * 1.3 * pulse
	glowColor := color.RGBA{150, 180, 255, 70}

	for i := 0; i < 2; i++ {
		r := glowRadius * (1.0 - float32(i)*0.15)
		alpha := uint8(70 - i*20)
		c := color.RGBA{glowColor.R, glowColor.G, glowColor.B, alpha}
		vector.DrawFilledCircle(dst, x, y, r, c, true)
	}
}

// drawPhantomMask renders the Phantom (100) mask overlay effect.
func (m *MilestoneEffects) drawPhantomMask(dst *ebiten.Image, x, y, radius float32) {
	// Phantom mask shape above node
	maskY := y - radius*1.5
	maskColor := color.RGBA{200, 200, 255, 100}

	// Simple mask shape (oval with eye holes)
	vector.DrawFilledCircle(dst, x, maskY, radius*0.8, maskColor, true)

	// Eye holes
	eyeOffset := radius * 0.25
	eyeColor := color.RGBA{50, 50, 80, 200}
	vector.DrawFilledCircle(dst, x-eyeOffset, maskY-2, 3, eyeColor, true)
	vector.DrawFilledCircle(dst, x+eyeOffset, maskY-2, 3, eyeColor, true)
}

// drawRevenantAura renders the Revenant (200) ethereal aura effect.
func (m *MilestoneEffects) drawRevenantAura(dst *ebiten.Image, x, y, radius float32) {
	// Ethereal swirling aura
	numSpirals := 3
	for i := 0; i < numSpirals; i++ {
		baseAngle := float32(i) * float32(math.Pi) * 2 / float32(numSpirals)

		// Spiral effect
		segments := 20
		prevX, prevY := x, y
		for j := 1; j <= segments; j++ {
			t := float32(j) / float32(segments)
			angle := baseAngle + m.time + t*float32(math.Pi)*2
			dist := radius * t * 2

			sx := x + float32(math.Cos(float64(angle)))*dist
			sy := y + float32(math.Sin(float64(angle)))*dist

			alpha := uint8(100 * (1 - t*0.7))
			c := color.RGBA{180, 200, 255, alpha}
			if j > 1 {
				vector.StrokeLine(dst, prevX, prevY, sx, sy, 2*(1-t)+0.5, c, true)
			}
			prevX, prevY = sx, sy
		}
	}
}

// drawAbyssVoid renders the Abyss (500) void shader effect.
// Only rendered in Fortress mode per RESONANCE_SYSTEM.md.
func (m *MilestoneEffects) drawAbyssVoid(dst *ebiten.Image, x, y, radius float32) {
	// Deep void effect with swirling darkness
	voidRadius := radius * 2

	// Dark center
	voidColor := color.RGBA{20, 20, 40, 200}
	vector.DrawFilledCircle(dst, x, y, voidRadius*0.5, voidColor, true)

	// Swirling void particles
	numParticles := 16
	for i := 0; i < numParticles; i++ {
		angle := float32(i)*float32(math.Pi)*2/float32(numParticles) + m.time*0.5
		dist := voidRadius * (0.5 + 0.3*float32(math.Sin(float64(angle*3+m.time*2))))

		px := x + float32(math.Cos(float64(angle)))*dist
		py := y + float32(math.Sin(float64(angle)))*dist

		alpha := uint8(100 + 50*float32(math.Sin(float64(angle+m.time*3))))
		c := color.RGBA{80, 80, 120, alpha}
		vector.DrawFilledCircle(dst, px, py, 4, c, true)
	}

	// Void ring
	ringColor := color.RGBA{100, 80, 150, 80}
	vector.StrokeCircle(dst, x, y, voidRadius, 3, ringColor, true)
}
