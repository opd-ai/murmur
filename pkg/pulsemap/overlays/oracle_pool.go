// Package overlays — Oracle Pool Pulse Map visualization.
// Per ROADMAP.md line 461: "Pulse Map visualization — swirling vortex icon at pool location".
// Oracle Pools display as animated vortex icons with prediction state indicators.
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// OraclePoolState represents the current state of an Oracle Pool.
type OraclePoolState uint8

const (
	OraclePoolPending   OraclePoolState = iota // Accepting predictions.
	OraclePoolRevealing                        // Reveal phase (after deadline).
	OraclePoolResolved                         // Outcome determined.
	OraclePoolExpired                          // Pool expired without resolution.
)

// OraclePoolVisual contains rendering data for an Oracle Pool overlay.
type OraclePoolVisual struct {
	PoolID          [32]byte
	X, Y            float64 // World-space position.
	State           OraclePoolState
	Question        string // Pool question (truncated for display).
	Deadline        time.Time
	ResolutionTime  time.Time
	PredictionCount int     // Number of predictions submitted.
	AnimationPhase  float64 // Animation time accumulator.
}

// OraclePoolOverlay manages Oracle Pool visualizations.
type OraclePoolOverlay struct {
	Pools   []*OraclePoolVisual
	Visible bool
	Opacity float32
}

// NewOraclePoolOverlay creates a new Oracle Pool overlay.
func NewOraclePoolOverlay() *OraclePoolOverlay {
	return &OraclePoolOverlay{
		Pools:   make([]*OraclePoolVisual, 0),
		Visible: true,
		Opacity: 0.8,
	}
}

// AddPool adds an Oracle Pool to the overlay.
func (o *OraclePoolOverlay) AddPool(pool *OraclePoolVisual) {
	o.Pools = append(o.Pools, pool)
}

// RemovePool removes an Oracle Pool by ID.
func (o *OraclePoolOverlay) RemovePool(poolID [32]byte) {
	for i, p := range o.Pools {
		if p.PoolID == poolID {
			o.Pools = append(o.Pools[:i], o.Pools[i+1:]...)
			return
		}
	}
}

// ClearPools removes all pools.
func (o *OraclePoolOverlay) ClearPools() {
	o.Pools = o.Pools[:0]
}

// GetPool retrieves a pool by ID.
func (o *OraclePoolOverlay) GetPool(poolID [32]byte) *OraclePoolVisual {
	for _, p := range o.Pools {
		if p.PoolID == poolID {
			return p
		}
	}
	return nil
}

// Count returns the number of pools.
func (o *OraclePoolOverlay) Count() int {
	return len(o.Pools)
}

// Update advances animation phases.
func (o *OraclePoolOverlay) Update(dt float64) {
	for _, p := range o.Pools {
		p.AnimationPhase += dt
	}
}

// Render draws all Oracle Pool overlays.
func (o *OraclePoolOverlay) Render(dst *ebiten.Image, cameraX, cameraY, scale float64) {
	if !o.Visible {
		return
	}

	screenW := float64(dst.Bounds().Dx())
	screenH := float64(dst.Bounds().Dy())

	for _, p := range o.Pools {
		renderOraclePool(dst, p, cameraX, cameraY, scale, screenW, screenH, o.Opacity)
	}
}

// renderOraclePool draws a single Oracle Pool as a swirling vortex.
func renderOraclePool(dst *ebiten.Image, p *OraclePoolVisual, cameraX, cameraY, scale, screenW, screenH float64, baseOpacity float32) {
	// Transform to screen-space.
	x := float32((p.X-cameraX)*scale + screenW/2)
	y := float32((p.Y-cameraY)*scale + screenH/2)

	// Base radius scales with zoom.
	baseRadius := float32(math.Max(20, math.Min(50, 30*scale)))

	// Choose colors based on state.
	var coreColor, spiralColor color.RGBA
	switch p.State {
	case OraclePoolPending:
		// Blue-purple for pending (accepting predictions).
		coreColor = color.RGBA{80, 80, 200, uint8(200 * baseOpacity)}
		spiralColor = color.RGBA{150, 100, 220, uint8(180 * baseOpacity)}
	case OraclePoolRevealing:
		// Yellow-orange for reveal phase.
		coreColor = color.RGBA{220, 180, 50, uint8(200 * baseOpacity)}
		spiralColor = color.RGBA{255, 200, 100, uint8(180 * baseOpacity)}
	case OraclePoolResolved:
		// Green for resolved.
		coreColor = color.RGBA{80, 200, 100, uint8(180 * baseOpacity)}
		spiralColor = color.RGBA{120, 220, 140, uint8(150 * baseOpacity)}
	case OraclePoolExpired:
		// Gray for expired.
		coreColor = color.RGBA{100, 100, 100, uint8(150 * baseOpacity)}
		spiralColor = color.RGBA{80, 80, 80, uint8(100 * baseOpacity)}
	}

	// Draw outer glow.
	glowRadius := baseRadius * 2
	glowColor := color.RGBA{coreColor.R, coreColor.G, coreColor.B, uint8(40 * baseOpacity)}
	vector.DrawFilledCircle(dst, x, y, glowRadius, glowColor, true)

	// Draw swirling vortex arms.
	numArms := 3
	armLength := baseRadius * 1.5
	rotationSpeed := 1.5 // Radians per second.

	for arm := 0; arm < numArms; arm++ {
		startAngle := p.AnimationPhase*rotationSpeed + float64(arm)*2*math.Pi/float64(numArms)
		drawVortexArm(dst, x, y, float32(startAngle), armLength, spiralColor, baseOpacity)
	}

	// Draw core.
	vector.DrawFilledCircle(dst, x, y, baseRadius*0.5, coreColor, true)

	// Draw center eye (darker).
	eyeColor := color.RGBA{20, 20, 40, uint8(220 * baseOpacity)}
	vector.DrawFilledCircle(dst, x, y, baseRadius*0.2, eyeColor, true)

	// Draw prediction count indicator.
	if p.PredictionCount > 0 && p.State == OraclePoolPending {
		drawPredictionIndicator(dst, x, y, baseRadius, p.PredictionCount, baseOpacity)
	}
}

// drawVortexArm draws a single spiral arm of the vortex.
func drawVortexArm(dst *ebiten.Image, cx, cy, startAngle, length float32, c color.RGBA, baseOpacity float32) {
	segments := 12
	prevX, prevY := cx, cy

	for i := 1; i <= segments; i++ {
		t := float32(i) / float32(segments)
		radius := length * t
		angle := startAngle + t*2*3.14159 // One full rotation along arm.

		armX := cx + radius*float32(math.Cos(float64(angle)))
		armY := cy + radius*float32(math.Sin(float64(angle)))

		// Fade alpha along arm.
		alpha := uint8(float32(c.A) * (1 - t*0.5))
		lineColor := color.RGBA{c.R, c.G, c.B, alpha}

		width := 3 - 2*t // Taper from 3 to 1.
		if width < 1 {
			width = 1
		}
		vector.StrokeLine(dst, prevX, prevY, armX, armY, width, lineColor, true)

		prevX, prevY = armX, armY
	}
}

// drawPredictionIndicator shows the number of predictions.
func drawPredictionIndicator(dst *ebiten.Image, cx, cy, radius float32, count int, baseOpacity float32) {
	// Draw small dots around the vortex for predictions (max 10 visible).
	visibleCount := count
	if visibleCount > 10 {
		visibleCount = 10
	}

	indicatorRadius := radius * 1.8
	dotColor := color.RGBA{255, 255, 255, uint8(180 * baseOpacity)}

	for i := 0; i < visibleCount; i++ {
		angle := 2 * math.Pi * float64(i) / float64(visibleCount)
		dotX := cx + indicatorRadius*float32(math.Cos(angle))
		dotY := cy + indicatorRadius*float32(math.Sin(angle))
		vector.DrawFilledCircle(dst, dotX, dotY, 2, dotColor, true)
	}
}

// NewOraclePoolVisual creates a new Oracle Pool visual.
func NewOraclePoolVisual(poolID [32]byte, x, y float64) *OraclePoolVisual {
	return &OraclePoolVisual{
		PoolID: poolID,
		X:      x,
		Y:      y,
		State:  OraclePoolPending,
	}
}

// SetState sets the pool's visual state.
func (p *OraclePoolVisual) SetState(state OraclePoolState) {
	p.State = state
}

// SetPosition sets the pool's position.
func (p *OraclePoolVisual) SetPosition(x, y float64) {
	p.X = x
	p.Y = y
}

// SetQuestion sets the pool question.
func (p *OraclePoolVisual) SetQuestion(question string) {
	p.Question = question
}

// SetDeadline sets the prediction deadline.
func (p *OraclePoolVisual) SetDeadline(deadline time.Time) {
	p.Deadline = deadline
}

// SetResolutionTime sets when the pool resolves.
func (p *OraclePoolVisual) SetResolutionTime(t time.Time) {
	p.ResolutionTime = t
}

// SetPredictionCount sets the number of predictions.
func (p *OraclePoolVisual) SetPredictionCount(count int) {
	p.PredictionCount = count
}

// SetOpacity sets the overlay opacity.
func (o *OraclePoolOverlay) SetOpacity(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	o.Opacity = opacity
}
