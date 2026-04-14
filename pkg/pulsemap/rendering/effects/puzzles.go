// Package effects provides puzzle visual effects for the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, puzzles are visualized as rotating cryptographic
// symbols at the puzzle location on the Pulse Map.
//
// Visual styles by puzzle type:
// - Fragment: rotating hexagon with golden glow
// - Mosaic: interlocking pieces animation
// - Cascade: flowing waterfall effect
//
// State indicators:
// - Active: bright pulsing animation
// - Solved: green checkmark overlay
// - Expired: faded gray with crack effect
//
//go:build !noebiten
// +build !noebiten

package effects

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PuzzleType identifies the visual style for a puzzle.
type PuzzleType int

const (
	PuzzleTypeFragment PuzzleType = iota + 1 // Rotating hexagon.
	PuzzleTypeMosaic                         // Interlocking pieces.
	PuzzleTypeCascade                        // Flowing waterfall.
)

// PuzzleState determines the visual indicator overlay.
type PuzzleState int

const (
	PuzzleStateActive  PuzzleState = iota // Bright pulsing.
	PuzzleStateSolved                     // Green checkmark.
	PuzzleStateExpired                    // Faded gray.
)

// PuzzleVisual represents a puzzle to be rendered on the Pulse Map.
type PuzzleVisual struct {
	ID       [32]byte
	X, Y     float32     // Position in screen coordinates.
	Type     PuzzleType  // Visual style.
	State    PuzzleState // Current state.
	Progress float32     // 0.0-1.0 for Mosaic contributions.
}

// PuzzleEffects renders puzzle visualizations on the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, puzzles appear as rotating cryptographic symbols.
type PuzzleEffects struct {
	mu      sync.RWMutex
	time    float32
	puzzles map[[32]byte]*PuzzleVisual

	// Fragment type particles.
	hexagonParticles []puzzleParticle

	// Mosaic piece states.
	mosaicPieces [5]float32 // Animation offset per piece.

	// Cascade flow particles.
	cascadeParticles []puzzleParticle
}

// puzzleParticle represents a visual particle for effects.
type puzzleParticle struct {
	X, Y    float32
	VX, VY  float32
	Life    float32
	MaxLife float32
	Size    float32
	Color   color.RGBA
	Angle   float32
}

// NewPuzzleEffects creates a new puzzle effects renderer.
func NewPuzzleEffects() *PuzzleEffects {
	return &PuzzleEffects{
		puzzles: make(map[[32]byte]*PuzzleVisual),
	}
}

// AddPuzzle adds a puzzle to be rendered.
func (p *PuzzleEffects) AddPuzzle(pv *PuzzleVisual) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.puzzles[pv.ID] = pv
}

// RemovePuzzle removes a puzzle from rendering.
func (p *PuzzleEffects) RemovePuzzle(id [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.puzzles, id)
}

// UpdatePuzzle updates a puzzle's state.
func (p *PuzzleEffects) UpdatePuzzle(id [32]byte, state PuzzleState, progress float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pv, ok := p.puzzles[id]; ok {
		pv.State = state
		pv.Progress = progress
	}
}

// Update advances animation state.
func (p *PuzzleEffects) Update(deltaTime float32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.time += deltaTime

	// Update hexagon particles.
	p.updateParticles(&p.hexagonParticles, deltaTime)

	// Update cascade particles.
	p.updateParticles(&p.cascadeParticles, deltaTime)

	// Update mosaic piece animations.
	for i := range p.mosaicPieces {
		p.mosaicPieces[i] += deltaTime * float32(i+1) * 0.5
		if p.mosaicPieces[i] > 2*math.Pi {
			p.mosaicPieces[i] -= 2 * math.Pi
		}
	}
}

// updateParticles updates particle physics.
func (p *PuzzleEffects) updateParticles(particles *[]puzzleParticle, dt float32) {
	alive := make([]puzzleParticle, 0, len(*particles))
	for i := range *particles {
		part := &(*particles)[i]
		part.X += part.VX * dt
		part.Y += part.VY * dt
		part.Life -= dt
		part.Angle += dt * 2
		if part.Life > 0 {
			alive = append(alive, *part)
		}
	}
	*particles = alive
}

// Draw renders all puzzle effects to the destination image.
func (p *PuzzleEffects) Draw(dst *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, pv := range p.puzzles {
		p.drawPuzzle(dst, pv)
	}
}

// drawPuzzle renders a single puzzle visualization.
func (p *PuzzleEffects) drawPuzzle(dst *ebiten.Image, pv *PuzzleVisual) {
	// Base size varies by state.
	baseSize := float32(20.0)
	if pv.State == PuzzleStateExpired {
		baseSize = 15.0
	}

	// Draw type-specific effect.
	switch pv.Type {
	case PuzzleTypeFragment:
		p.drawFragmentPuzzle(dst, pv.X, pv.Y, baseSize, pv.State)
	case PuzzleTypeMosaic:
		p.drawMosaicPuzzle(dst, pv.X, pv.Y, baseSize, pv.State, pv.Progress)
	case PuzzleTypeCascade:
		p.drawCascadePuzzle(dst, pv.X, pv.Y, baseSize, pv.State)
	}

	// Draw state overlay.
	p.drawStateOverlay(dst, pv.X, pv.Y, baseSize, pv.State)
}

// drawFragmentPuzzle renders a rotating hexagon for Fragment puzzles.
func (p *PuzzleEffects) drawFragmentPuzzle(dst *ebiten.Image, x, y, size float32, state PuzzleState) {
	// Golden color for active, gray for expired.
	baseColor := color.RGBA{255, 215, 0, 200} // Gold.
	if state == PuzzleStateExpired {
		baseColor = color.RGBA{128, 128, 128, 100}
	} else if state == PuzzleStateSolved {
		baseColor = color.RGBA{100, 255, 100, 200}
	}

	// Rotation animation.
	rotation := p.time * 1.5
	pulse := 1.0 + 0.15*float32(math.Sin(float64(p.time*3)))

	// Draw hexagon outline.
	sides := 6
	for i := 0; i < sides; i++ {
		angle1 := rotation + float32(i)*2*math.Pi/float32(sides)
		angle2 := rotation + float32(i+1)*2*math.Pi/float32(sides)

		x1 := x + size*pulse*float32(math.Cos(float64(angle1)))
		y1 := y + size*pulse*float32(math.Sin(float64(angle1)))
		x2 := x + size*pulse*float32(math.Cos(float64(angle2)))
		y2 := y + size*pulse*float32(math.Sin(float64(angle2)))

		vector.StrokeLine(dst, x1, y1, x2, y2, 2.5, baseColor, true)
	}

	// Inner rotating triangle.
	innerRotation := -p.time * 2
	innerSize := size * 0.5 * pulse
	innerColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, 150}

	for i := 0; i < 3; i++ {
		angle1 := innerRotation + float32(i)*2*math.Pi/3
		angle2 := innerRotation + float32(i+1)*2*math.Pi/3

		x1 := x + innerSize*float32(math.Cos(float64(angle1)))
		y1 := y + innerSize*float32(math.Sin(float64(angle1)))
		x2 := x + innerSize*float32(math.Cos(float64(angle2)))
		y2 := y + innerSize*float32(math.Sin(float64(angle2)))

		vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, innerColor, true)
	}

	// Glow effect for active puzzles.
	if state == PuzzleStateActive {
		glowColor := color.RGBA{255, 215, 0, 40}
		glowSize := size * 1.8 * pulse
		vector.DrawFilledCircle(dst, x, y, glowSize, glowColor, true)
	}
}

// drawMosaicPuzzle renders interlocking pieces for Mosaic puzzles.
func (p *PuzzleEffects) drawMosaicPuzzle(dst *ebiten.Image, x, y, size float32, state PuzzleState, progress float32) {
	// Blue/purple color scheme for Mosaic.
	baseColor := color.RGBA{100, 150, 255, 200}
	if state == PuzzleStateExpired {
		baseColor = color.RGBA{128, 128, 128, 100}
	} else if state == PuzzleStateSolved {
		baseColor = color.RGBA{100, 255, 100, 200}
	}

	// 5 interlocking pieces arranged in a cross.
	piecePositions := [][2]float32{
		{0, 0},           // Center.
		{-size * 0.8, 0}, // Left.
		{size * 0.8, 0},  // Right.
		{0, -size * 0.8}, // Top.
		{0, size * 0.8},  // Bottom.
	}

	completedPieces := int(progress * 5)

	for i, pos := range piecePositions {
		offsetX := pos[0]
		offsetY := pos[1]

		// Animation: pieces pulse individually.
		piecePulse := 1.0 + 0.1*float32(math.Sin(float64(p.mosaicPieces[i])))
		pieceSize := size * 0.35 * piecePulse

		// Color based on completion.
		pieceColor := baseColor
		if i < completedPieces {
			pieceColor = color.RGBA{100, 255, 150, 200} // Green for completed.
		}

		// Draw square piece with rounded corners effect.
		px := x + offsetX
		py := y + offsetY
		vector.DrawFilledRect(dst, px-pieceSize, py-pieceSize, pieceSize*2, pieceSize*2, pieceColor, true)

		// Connection lines between pieces.
		if i > 0 {
			lineColor := color.RGBA{pieceColor.R, pieceColor.G, pieceColor.B, 100}
			vector.StrokeLine(dst, x, y, px, py, 1.0, lineColor, true)
		}
	}

	// Glow for active.
	if state == PuzzleStateActive {
		glowColor := color.RGBA{100, 150, 255, 30}
		glowSize := size * 1.6
		vector.DrawFilledCircle(dst, x, y, glowSize, glowColor, true)
	}
}

// drawCascadePuzzle renders flowing waterfall effect for Cascade puzzles.
func (p *PuzzleEffects) drawCascadePuzzle(dst *ebiten.Image, x, y, size float32, state PuzzleState) {
	// Cyan/teal for Cascade.
	baseColor := color.RGBA{0, 200, 200, 200}
	if state == PuzzleStateExpired {
		baseColor = color.RGBA{128, 128, 128, 100}
	} else if state == PuzzleStateSolved {
		baseColor = color.RGBA{100, 255, 100, 200}
	}

	// Draw cascading layers.
	layers := 3
	for i := 0; i < layers; i++ {
		layerOffset := float32(i) * size * 0.5
		layerY := y + layerOffset
		layerWidth := size * (1.0 - float32(i)*0.2)

		// Wave animation.
		wave := float32(math.Sin(float64(p.time*3 + float32(i)*1.5)))
		layerX := x + wave*5

		// Draw layer as horizontal bar.
		alpha := uint8(200 - i*50)
		layerColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, alpha}
		vector.DrawFilledRect(dst, layerX-layerWidth, layerY-3, layerWidth*2, 6, layerColor, true)

		// Connecting flow lines.
		if i < layers-1 {
			nextWave := float32(math.Sin(float64(p.time*3 + float32(i+1)*1.5)))
			nextX := x + nextWave*5
			nextY := y + float32(i+1)*size*0.5
			flowColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, 80}
			vector.StrokeLine(dst, layerX, layerY+3, nextX, nextY-3, 1.5, flowColor, true)
		}
	}

	// Stage indicator arrows.
	arrowColor := color.RGBA{baseColor.R, baseColor.G, baseColor.B, 150}
	arrowY := y - size*0.4
	arrowSize := float32(5.0)
	animOffset := float32(math.Sin(float64(p.time*4))) * 3

	// Down arrow.
	vector.StrokeLine(dst, x, arrowY+animOffset, x-arrowSize, arrowY-arrowSize+animOffset, 2.0, arrowColor, true)
	vector.StrokeLine(dst, x, arrowY+animOffset, x+arrowSize, arrowY-arrowSize+animOffset, 2.0, arrowColor, true)

	// Glow for active.
	if state == PuzzleStateActive {
		glowColor := color.RGBA{0, 200, 200, 30}
		glowSize := size * 1.6
		vector.DrawFilledCircle(dst, x, y, glowSize, glowColor, true)
	}
}

// drawStateOverlay draws state indicators (checkmark, crack).
func (p *PuzzleEffects) drawStateOverlay(dst *ebiten.Image, x, y, size float32, state PuzzleState) {
	switch state {
	case PuzzleStateSolved:
		// Green checkmark.
		checkColor := color.RGBA{50, 255, 50, 255}
		checkSize := size * 0.6

		// Checkmark path: start lower-left, go down-center, then up-right.
		x1 := x - checkSize*0.5
		y1 := y
		x2 := x - checkSize*0.1
		y2 := y + checkSize*0.4
		x3 := x + checkSize*0.5
		y3 := y - checkSize*0.4

		vector.StrokeLine(dst, x1, y1, x2, y2, 3.0, checkColor, true)
		vector.StrokeLine(dst, x2, y2, x3, y3, 3.0, checkColor, true)

	case PuzzleStateExpired:
		// Crack/X overlay.
		crackColor := color.RGBA{180, 80, 80, 150}
		crackSize := size * 0.4

		// X pattern.
		vector.StrokeLine(dst, x-crackSize, y-crackSize, x+crackSize, y+crackSize, 2.0, crackColor, true)
		vector.StrokeLine(dst, x+crackSize, y-crackSize, x-crackSize, y+crackSize, 2.0, crackColor, true)
	}
}

// Clear removes all puzzles.
func (p *PuzzleEffects) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.puzzles = make(map[[32]byte]*PuzzleVisual)
	p.hexagonParticles = nil
	p.cascadeParticles = nil
}

// Count returns the number of active puzzles being rendered.
func (p *PuzzleEffects) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.puzzles)
}
