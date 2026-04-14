// Package overlays provides Anonymous Layer overlay and activity heatmap.
// Per DESIGN_DOCUMENT.md, the Pulse Map shows anonymous activity as overlays.
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// LayerBlend controls the visibility blending between Surface and Anonymous layers.
// Per PULSE_MAP.md, users can adjust layer blend with a slider.
type LayerBlend struct {
	SurfaceOpacity   float32 // 0-1, opacity of Surface Layer
	AnonymousOpacity float32 // 0-1, opacity of Anonymous Layer
	IsFortressMode   bool    // If true, only Anonymous Layer visible
}

// NewDefaultBlend creates a default layer blend (both layers visible).
func NewDefaultBlend() *LayerBlend {
	return &LayerBlend{
		SurfaceOpacity:   1.0,
		AnonymousOpacity: 0.5,
		IsFortressMode:   false,
	}
}

// NewFortressBlend creates a Fortress mode blend (only Anonymous Layer).
func NewFortressBlend() *LayerBlend {
	return &LayerBlend{
		SurfaceOpacity:   0.0,
		AnonymousOpacity: 1.0,
		IsFortressMode:   true,
	}
}

// SetBlendRatio adjusts the blend between layers (0 = Surface only, 1 = Anonymous only).
func (b *LayerBlend) SetBlendRatio(ratio float32) {
	if b.IsFortressMode {
		return // Fortress mode locks to Anonymous only
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

// SpecterParticle represents an animated particle for Specter node effects.
// Per PULSE_MAP.md, Specter nodes emit luminous particles.
type SpecterParticle struct {
	X, Y    float32 // Current position
	VX, VY  float32 // Velocity
	Life    float32 // Remaining life (0-1)
	MaxLife float32 // Starting life
	Size    float32 // Particle size
	Color   color.RGBA
}

// ParticleEmitter manages particle emission for Specter nodes.
type ParticleEmitter struct {
	Particles    []SpecterParticle
	MaxParticles int
	EmitRate     float32 // Particles per second
	accumulator  float32 // Time accumulator for emission
}

// NewParticleEmitter creates a new particle emitter.
func NewParticleEmitter(maxParticles int, emitRate float32) *ParticleEmitter {
	return &ParticleEmitter{
		Particles:    make([]SpecterParticle, 0, maxParticles),
		MaxParticles: maxParticles,
		EmitRate:     emitRate,
	}
}

// Update advances particle physics and emits new particles.
func (e *ParticleEmitter) Update(dt, nodeX, nodeY, nodeRadius, resonance float32) {
	updateParticles(e, dt, nodeX, nodeY, nodeRadius, resonance)
}

// Render draws all particles to the screen.
func (e *ParticleEmitter) Render(dst *ebiten.Image, cameraX, cameraY, scale float32) {
	for _, p := range e.Particles {
		alpha := uint8(float32(p.Color.A) * p.Life)
		c := color.RGBA{p.Color.R, p.Color.G, p.Color.B, alpha}
		screenX := (p.X-cameraX)*scale + float32(dst.Bounds().Dx())/2
		screenY := (p.Y-cameraY)*scale + float32(dst.Bounds().Dy())/2
		size := p.Size * p.Life * scale
		vector.DrawFilledCircle(dst, screenX, screenY, size, c, true)
	}
}

// cos returns cosine of angle in radians.
func cos(angle float32) float32 {
	return particleCos(angle)
}

// sin returns sine of angle in radians.
func sin(angle float32) float32 {
	return particleSin(angle)
}

// ShroudIndicator renders the Shroud routing indicator.
// Per PULSE_MAP.md, shows a faint animated path through relay shields.
func ShroudIndicator(dst *ebiten.Image, nodeX, nodeY float32, active bool, phase float32) {
	if !active {
		return
	}

	// Draw three shield glyphs representing relay hops
	shieldColor := color.RGBA{100, 150, 200, uint8(100 + 50*sin(phase*3))}

	offsets := []float32{-40, 0, 40}
	for i, offset := range offsets {
		x := nodeX + offset
		y := nodeY - 60 - float32(i)*20

		// Draw simple shield shape (triangle top)
		vector.DrawFilledCircle(dst, x, y, 8, shieldColor, true)

		// Connecting line
		if i < len(offsets)-1 {
			nextX := nodeX + offsets[i+1]
			nextY := nodeY - 60 - float32(i+1)*20
			vector.StrokeLine(dst, x, y, nextX, nextY, 1, shieldColor, true)
		}
	}
}

// MiniGameVisualization renders an active mini-game event.
// Per PULSE_MAP.md, mini-games show type-specific animated icons.
type MiniGameVisualization struct {
	Player1X, Player1Y float32
	Player2X, Player2Y float32
	Color1, Color2     color.RGBA
	Intensity          float32 // 0-1, increases with game activity
	Phase              float32 // Animation phase
}

// Render draws the mini-game visualization.
func (d *MiniGameVisualization) Render(dst *ebiten.Image, cameraX, cameraY, scale float32) {
	x1 := (d.Player1X-cameraX)*scale + float32(dst.Bounds().Dx())/2
	y1 := (d.Player1Y-cameraY)*scale + float32(dst.Bounds().Dy())/2
	x2 := (d.Player2X-cameraX)*scale + float32(dst.Bounds().Dx())/2
	y2 := (d.Player2Y-cameraY)*scale + float32(dst.Bounds().Dy())/2

	// Draw jagged electric-arc line
	segments := 8
	prevX, prevY := x1, y1
	for i := 1; i <= segments; i++ {
		t := float32(i) / float32(segments)

		// Linear interpolation with jitter
		jitter := sin(d.Phase*10+t*20) * 10 * d.Intensity
		x := x1 + (x2-x1)*t + jitter
		y := y1 + (y2-y1)*t + jitter*0.5

		// Alternate colors
		var c color.RGBA
		if i%2 == 0 {
			c = d.Color1
		} else {
			c = d.Color2
		}
		c.A = uint8(180 * d.Intensity)

		vector.StrokeLine(dst, prevX, prevY, x, y, 2+d.Intensity*2, c, true)
		prevX, prevY = x, y
	}
}
