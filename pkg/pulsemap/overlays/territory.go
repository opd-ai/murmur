// Package overlays — Territory Drift Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md and ROADMAP.md line 446:
// "Pulse Map visualization — translucent watermarks with territory boundaries".
// Territories show controller sigils as watermarks and boundary states
// (neutral, controlled, contested).
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TerritoryState mirrors the state from mechanics/territory.go.
type TerritoryState uint8

const (
	TerritoryNeutral    TerritoryState = iota // No dominant controller.
	TerritoryControlled                       // Single controller.
	TerritoryContested                        // Multiple contenders.
)

// TerritoryVisual contains rendering data for a territory overlay.
type TerritoryVisual struct {
	ID              string
	CentroidX       float64 // World-space center X.
	CentroidY       float64 // World-space center Y.
	Boundary        []Point // Convex hull vertices in world-space.
	State           TerritoryState
	ControllerSigil *ebiten.Image // Controller's sigil (64x64).
	ContenderSigils []*ebiten.Image
	Influence       float64 // Controller's influence for opacity scaling.
	AnimationPhase  float64 // Animation time accumulator.
}

// Point is a 2D coordinate.
type Point struct {
	X, Y float64
}

// TerritoryOverlay renders territory visuals on the Pulse Map.
type TerritoryOverlay struct {
	Territories []*TerritoryVisual
	Visible     bool
	Opacity     float32 // Global opacity multiplier (0-1).
}

// NewTerritoryOverlay creates a new territory overlay.
func NewTerritoryOverlay() *TerritoryOverlay {
	return &TerritoryOverlay{
		Territories: make([]*TerritoryVisual, 0),
		Visible:     true,
		Opacity:     0.6, // Default 60% opacity per spec.
	}
}

// SetOpacity sets the global opacity for all territory visuals.
func (o *TerritoryOverlay) SetOpacity(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	o.Opacity = opacity
}

// AddTerritory adds a territory visual to the overlay.
func (o *TerritoryOverlay) AddTerritory(t *TerritoryVisual) {
	o.Territories = append(o.Territories, t)
}

// RemoveTerritory removes a territory by ID.
func (o *TerritoryOverlay) RemoveTerritory(id string) {
	for i, t := range o.Territories {
		if t.ID == id {
			o.Territories = append(o.Territories[:i], o.Territories[i+1:]...)
			return
		}
	}
}

// ClearTerritories removes all territories.
func (o *TerritoryOverlay) ClearTerritories() {
	o.Territories = o.Territories[:0]
}

// GetTerritory retrieves a territory by ID.
func (o *TerritoryOverlay) GetTerritory(id string) *TerritoryVisual {
	for _, t := range o.Territories {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// Update advances animation phases for all territories.
func (o *TerritoryOverlay) Update(dt float64) {
	for _, t := range o.Territories {
		t.AnimationPhase += dt
	}
}

// Render draws all territory overlays.
// cameraX, cameraY: current camera position in world-space.
// scale: zoom scale factor.
func (o *TerritoryOverlay) Render(dst *ebiten.Image, cameraX, cameraY, scale float64) {
	if !o.Visible {
		return
	}

	screenW := float64(dst.Bounds().Dx())
	screenH := float64(dst.Bounds().Dy())

	for _, t := range o.Territories {
		renderTerritory(dst, t, cameraX, cameraY, scale, screenW, screenH, o.Opacity)
	}
}

// renderTerritory draws a single territory.
func renderTerritory(dst *ebiten.Image, t *TerritoryVisual, cameraX, cameraY, scale, screenW, screenH float64, baseOpacity float32) {
	// Transform centroid to screen-space.
	cx := (t.CentroidX-cameraX)*scale + screenW/2
	cy := (t.CentroidY-cameraY)*scale + screenH/2

	// Draw boundary.
	renderTerritoryBoundary(dst, t, cameraX, cameraY, scale, screenW, screenH, baseOpacity)

	// Draw watermark sigil at centroid.
	renderTerritoryWatermark(dst, t, cx, cy, scale, baseOpacity)
}

// renderTerritoryBoundary draws the territory boundary line.
func renderTerritoryBoundary(dst *ebiten.Image, t *TerritoryVisual, cameraX, cameraY, scale, screenW, screenH float64, baseOpacity float32) {
	if len(t.Boundary) < 3 {
		return
	}

	// Choose boundary color and style based on state.
	var boundaryColor color.RGBA
	var lineWidth float32
	switch t.State {
	case TerritoryNeutral:
		// Neutral: faint gray dashed (rendered as dotted).
		boundaryColor = color.RGBA{128, 128, 128, uint8(80 * baseOpacity)}
		lineWidth = 1.0
	case TerritoryControlled:
		// Controlled: solid purple glow.
		boundaryColor = color.RGBA{150, 100, 200, uint8(120 * baseOpacity)}
		lineWidth = 2.0
	case TerritoryContested:
		// Contested: shimmering boundary (animated).
		shimmer := float32(0.7 + 0.3*math.Sin(t.AnimationPhase*3.0))
		boundaryColor = color.RGBA{200, 150, 100, uint8(150 * baseOpacity * shimmer)}
		lineWidth = 2.5
	}

	// Draw boundary polygon.
	for i := 0; i < len(t.Boundary); i++ {
		p1 := t.Boundary[i]
		p2 := t.Boundary[(i+1)%len(t.Boundary)]

		// Transform to screen-space.
		x1 := float32((p1.X-cameraX)*scale + screenW/2)
		y1 := float32((p1.Y-cameraY)*scale + screenH/2)
		x2 := float32((p2.X-cameraX)*scale + screenW/2)
		y2 := float32((p2.Y-cameraY)*scale + screenH/2)

		vector.StrokeLine(dst, x1, y1, x2, y2, lineWidth, boundaryColor, true)
	}
}

// renderTerritoryWatermark draws the controller sigil as a translucent watermark.
func renderTerritoryWatermark(dst *ebiten.Image, t *TerritoryVisual, cx, cy, scale float64, baseOpacity float32) {
	// Compute watermark opacity based on influence.
	// Higher influence = more visible watermark.
	influenceOpacity := float32(math.Min(1.0, t.Influence/50.0))
	watermarkAlpha := baseOpacity * influenceOpacity * 0.5 // Max 50% opacity.

	if watermarkAlpha < 0.1 {
		watermarkAlpha = 0.1 // Minimum visibility.
	}

	switch t.State {
	case TerritoryNeutral:
		// No watermark for neutral territories.
		return
	case TerritoryControlled:
		if t.ControllerSigil != nil {
			drawWatermarkSigil(dst, t.ControllerSigil, cx, cy, scale, watermarkAlpha)
		}
	case TerritoryContested:
		// Contested: alternate between contender sigils.
		if len(t.ContenderSigils) == 0 {
			return
		}
		// Cycle through contenders based on animation phase.
		idx := int(t.AnimationPhase) % len(t.ContenderSigils)
		sigil := t.ContenderSigils[idx]
		if sigil != nil {
			// Pulsing effect for contested.
			pulse := float32(0.7 + 0.3*math.Sin(t.AnimationPhase*2.0))
			drawWatermarkSigil(dst, sigil, cx, cy, scale, watermarkAlpha*pulse)
		}
	}
}

// drawWatermarkSigil draws a sigil image as a translucent watermark.
func drawWatermarkSigil(dst, sigil *ebiten.Image, cx, cy, scale float64, alpha float32) {
	if sigil == nil {
		return
	}

	// Sigil size scales with zoom, minimum 32px, maximum 128px.
	sigilSize := math.Max(32, math.Min(128, 64*scale))
	sw := float64(sigil.Bounds().Dx())

	// Center the sigil.
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-sw/2, -sw/2)
	scaleRatio := sigilSize / sw
	opts.GeoM.Scale(scaleRatio, scaleRatio)
	opts.GeoM.Translate(cx, cy)

	// Apply translucency.
	opts.ColorScale.Scale(1, 1, 1, float32(alpha))

	dst.DrawImage(sigil, opts)
}

// TerritoryBoundaryPoint returns points for a circular territory boundary.
// Useful when exact convex hull is not available.
func TerritoryBoundaryCircle(cx, cy, radius float64, segments int) []Point {
	if segments < 3 {
		segments = 12
	}
	points := make([]Point, segments)
	for i := 0; i < segments; i++ {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		points[i] = Point{
			X: cx + radius*math.Cos(angle),
			Y: cy + radius*math.Sin(angle),
		}
	}
	return points
}

// NewTerritoryVisual creates a new territory visual.
func NewTerritoryVisual(id string, centroidX, centroidY float64) *TerritoryVisual {
	return &TerritoryVisual{
		ID:        id,
		CentroidX: centroidX,
		CentroidY: centroidY,
		State:     TerritoryNeutral,
	}
}

// SetBoundary sets the territory boundary polygon.
func (t *TerritoryVisual) SetBoundary(boundary []Point) {
	t.Boundary = boundary
}

// SetState sets the territory's visual state.
func (t *TerritoryVisual) SetState(state TerritoryState) {
	t.State = state
}

// SetControllerSigil sets the controller's sigil for watermarking.
func (t *TerritoryVisual) SetControllerSigil(sigil *ebiten.Image) {
	t.ControllerSigil = sigil
}

// SetContenderSigils sets the contender sigils for contested display.
func (t *TerritoryVisual) SetContenderSigils(sigils []*ebiten.Image) {
	t.ContenderSigils = sigils
}

// SetInfluence sets the influence level for opacity scaling.
func (t *TerritoryVisual) SetInfluence(influence float64) {
	t.Influence = influence
}

// Count returns the number of territories in the overlay.
func (o *TerritoryOverlay) Count() int {
	return len(o.Territories)
}
