// Package overlays - Pulse Beat edge-of-viewport notification rendering.
// Per ANONYMOUS_GAME_MECHANICS.md §10: "Pulse Beats appear as pulsing glyphs at the
// edge of the viewport, pointing toward the event source. Tapping a beat pans the
// camera to the relevant location."
// Per ROADMAP.md line 570: "Edge-of-viewport notification rendering"
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BeatType identifies the type of Pulse Beat notification.
type BeatType uint8

const (
	BeatGift BeatType = iota + 1
	BeatHunt
	BeatForge
	BeatChain
	BeatTerritory
	BeatSpark
	BeatPuzzle
	BeatCouncil
	BeatMark
	BeatWave
)

// BeatPriority indicates the urgency of a Pulse Beat.
type BeatPriority uint8

const (
	BeatPriorityLow BeatPriority = iota + 1
	BeatPriorityNormal
	BeatPriorityHigh
	BeatPriorityUrgent
)

// DisplayBeat contains beat information for visualization.
type DisplayBeat struct {
	ID          [32]byte     // Unique beat ID.
	Type        BeatType     // Beat type.
	Priority    BeatPriority // Display priority.
	Title       string       // Brief title.
	TargetX     float64      // World X coordinate of event source.
	TargetY     float64      // World Y coordinate of event source.
	CreatedAt   time.Time    // When the beat was created.
	DisplayedAt time.Time    // When the beat started displaying.
	Color       color.RGBA   // Beat glyph color.
	Read        bool         // Whether the beat has been acknowledged.
}

// PulseBeatOverlay renders Pulse Beats at the edge of the viewport.
type PulseBeatOverlay struct {
	mu sync.RWMutex

	visible     bool
	beats       []*DisplayBeat
	time        float64 // Animation time.
	edgeMargin  float32 // Distance from screen edge.
	maxVisible  int     // Maximum beats to display.
	displayTime time.Duration
	fadeTime    time.Duration

	// Callbacks.
	onBeatTapped func(beatID [32]byte)

	// Visual settings per beat type.
	typeColors map[BeatType]color.RGBA
	typeGlyphs map[BeatType]rune

	// Default colors.
	backgroundColor color.RGBA
	pointerColor    color.RGBA
}

// NewPulseBeatOverlay creates a new Pulse Beat overlay.
func NewPulseBeatOverlay() *PulseBeatOverlay {
	return &PulseBeatOverlay{
		visible:     true,
		beats:       make([]*DisplayBeat, 0),
		edgeMargin:  20,
		maxVisible:  3,
		displayTime: 5 * time.Second,
		fadeTime:    500 * time.Millisecond,
		typeColors: map[BeatType]color.RGBA{
			BeatGift:      {R: 220, G: 100, B: 220, A: 255}, // Purple for gifts.
			BeatHunt:      {R: 255, G: 60, B: 60, A: 255},   // Red for hunts.
			BeatForge:     {R: 255, G: 140, B: 40, A: 255},  // Orange for forges.
			BeatChain:     {R: 255, G: 200, B: 80, A: 255},  // Gold for chains.
			BeatTerritory: {R: 80, G: 180, B: 80, A: 255},   // Green for territory.
			BeatSpark:     {R: 100, G: 200, B: 255, A: 255}, // Cyan for sparks.
			BeatPuzzle:    {R: 180, G: 100, B: 220, A: 255}, // Purple for puzzles.
			BeatCouncil:   {R: 100, G: 120, B: 200, A: 255}, // Blue for councils.
			BeatMark:      {R: 200, G: 80, B: 120, A: 255},  // Pink for marks.
			BeatWave:      {R: 120, G: 180, B: 220, A: 255}, // Light blue for waves.
		},
		typeGlyphs: map[BeatType]rune{
			BeatGift:      '♦',
			BeatHunt:      '◆',
			BeatForge:     '▲',
			BeatChain:     '◯',
			BeatTerritory: '■',
			BeatSpark:     '★',
			BeatPuzzle:    '?',
			BeatCouncil:   '●',
			BeatMark:      '✕',
			BeatWave:      '~',
		},
		backgroundColor: color.RGBA{R: 30, G: 30, B: 40, A: 200},
		pointerColor:    color.RGBA{R: 255, G: 255, B: 255, A: 180},
	}
}

// SetVisible controls visibility.
func (o *PulseBeatOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *PulseBeatOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// AddBeat adds a beat to the display queue.
func (o *PulseBeatOverlay) AddBeat(beat *DisplayBeat) {
	if beat == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.initializeBeat(beat)

	if o.updateExistingBeat(beat) {
		return
	}

	o.insertBeatByPriority(beat)
	o.enforceMaxVisible()
}

// initializeBeat sets default values for beat if not already set.
func (o *PulseBeatOverlay) initializeBeat(beat *DisplayBeat) {
	if beat.DisplayedAt.IsZero() {
		beat.DisplayedAt = time.Now()
	}

	if beat.Color.A == 0 {
		if c, ok := o.typeColors[beat.Type]; ok {
			beat.Color = c
		} else {
			beat.Color = color.RGBA{R: 200, G: 200, B: 200, A: 255}
		}
	}
}

// updateExistingBeat updates a beat if it already exists.
// Returns true if the beat was found and updated.
func (o *PulseBeatOverlay) updateExistingBeat(beat *DisplayBeat) bool {
	for i, b := range o.beats {
		if b.ID == beat.ID {
			o.beats[i] = beat
			return true
		}
	}
	return false
}

// insertBeatByPriority inserts beat in priority order.
func (o *PulseBeatOverlay) insertBeatByPriority(beat *DisplayBeat) {
	for i, b := range o.beats {
		if beat.Priority > b.Priority {
			newBeats := make([]*DisplayBeat, 0, len(o.beats)+1)
			newBeats = append(newBeats, o.beats[:i]...)
			newBeats = append(newBeats, beat)
			newBeats = append(newBeats, o.beats[i:]...)
			o.beats = newBeats
			return
		}
	}
	o.beats = append(o.beats, beat)
}

// enforceMaxVisible trims beat list to maximum visible count.
func (o *PulseBeatOverlay) enforceMaxVisible() {
	if len(o.beats) > o.maxVisible {
		o.beats = o.beats[:o.maxVisible]
	}
}

// RemoveBeat removes a beat by ID.
func (o *PulseBeatOverlay) RemoveBeat(id [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for i, b := range o.beats {
		if b.ID == id {
			o.beats = append(o.beats[:i], o.beats[i+1:]...)
			return
		}
	}
}

// GetBeat returns a beat by ID.
func (o *PulseBeatOverlay) GetBeat(id [32]byte) *DisplayBeat {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, b := range o.beats {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// SetOnBeatTapped sets the callback for when a beat is tapped.
func (o *PulseBeatOverlay) SetOnBeatTapped(cb func(beatID [32]byte)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.onBeatTapped = cb
}

// Update advances animation state and removes expired beats.
func (o *PulseBeatOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.time += dt

	// Remove expired beats.
	now := time.Now()
	var active []*DisplayBeat
	for _, b := range o.beats {
		displayEnd := b.DisplayedAt.Add(o.displayTime)
		if now.Before(displayEnd) {
			active = append(active, b)
		}
	}
	o.beats = active
}

// Draw renders the pulse beats at viewport edges.
func (o *PulseBeatOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible || len(o.beats) == 0 {
		return
	}

	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())
	centerX := screenW / 2
	centerY := screenH / 2

	now := time.Now()

	for i, beat := range o.beats {
		// Calculate position of target on screen.
		targetSX := centerX + (beat.TargetX-cameraX)*zoom
		targetSY := centerY + (beat.TargetY-cameraY)*zoom

		// Check if target is on screen.
		if targetSX >= 0 && targetSX <= screenW && targetSY >= 0 && targetSY <= screenH {
			// Target is visible - draw indicator at target position.
			o.drawOnScreenIndicator(screen, float32(targetSX), float32(targetSY), beat, i, now)
		} else {
			// Target is off-screen - draw at edge pointing to target.
			o.drawEdgeIndicator(screen, float32(centerX), float32(centerY),
				float32(targetSX), float32(targetSY),
				float32(screenW), float32(screenH), beat, i, now)
		}
	}
}

// drawOnScreenIndicator draws a subtle indicator at the target location.
func (o *PulseBeatOverlay) drawOnScreenIndicator(screen *ebiten.Image, x, y float32, beat *DisplayBeat, index int, now time.Time) {
	// Calculate fade based on display time.
	elapsed := now.Sub(beat.DisplayedAt).Seconds()
	displaySecs := o.displayTime.Seconds()
	fadeSecs := o.fadeTime.Seconds()

	alpha := float32(1.0)
	if elapsed > displaySecs-fadeSecs {
		alpha = float32((displaySecs - elapsed) / fadeSecs)
		if alpha < 0 {
			alpha = 0
		}
	}

	// Pulsing effect.
	pulse := float32(math.Sin(o.time*3+float64(index))*0.2 + 0.8)

	// Draw pulsing ring at target location.
	ringRadius := 15 * pulse
	ringAlpha := uint8(float32(beat.Color.A) * alpha * 0.6)
	ringColor := color.RGBA{beat.Color.R, beat.Color.G, beat.Color.B, ringAlpha}

	vector.StrokeCircle(screen, x, y, ringRadius, 2, ringColor, true)

	// Inner dot.
	dotRadius := 5 * pulse
	dotAlpha := uint8(float32(beat.Color.A) * alpha)
	dotColor := color.RGBA{beat.Color.R, beat.Color.G, beat.Color.B, dotAlpha}
	vector.DrawFilledCircle(screen, x, y, dotRadius, dotColor, true)
}

// drawEdgeIndicator draws the beat indicator at the screen edge pointing to target.
func (o *PulseBeatOverlay) drawEdgeIndicator(screen *ebiten.Image, centerX, centerY, targetX, targetY, screenW, screenH float32, beat *DisplayBeat, index int, now time.Time) {
	dirX, dirY := o.calculateDirection(centerX, centerY, targetX, targetY)
	if dirX == 0 && dirY == 0 {
		return
	}

	margin := o.edgeMargin
	edgeX, edgeY := o.findEdgeIntersection(centerX, centerY, dirX, dirY, screenW, screenH, margin)
	edgeX, edgeY = o.applyStackOffset(edgeX, edgeY, screenW, screenH, margin, index)

	alpha := o.calculateFadeAlpha(beat, now)
	o.drawBeatGlyph(screen, edgeX, edgeY, dirX, dirY, beat, alpha, index)
}

// calculateDirection computes and normalizes the direction vector to target.
func (o *PulseBeatOverlay) calculateDirection(centerX, centerY, targetX, targetY float32) (float32, float32) {
	dx := targetX - centerX
	dy := targetY - centerY
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 1 {
		return 0, 0
	}
	return dx / dist, dy / dist
}

// applyStackOffset adjusts edge position to stack multiple beats vertically.
func (o *PulseBeatOverlay) applyStackOffset(edgeX, edgeY, screenW, screenH, margin float32, index int) (float32, float32) {
	stackOffset := float32(index) * 50

	if edgeX <= margin {
		edgeY = o.clampVertical(edgeY+stackOffset, screenH, margin)
	} else if edgeX >= screenW-margin {
		edgeY = o.clampVertical(edgeY+stackOffset, screenH, margin)
	} else if edgeY <= margin {
		edgeX = o.clampHorizontal(edgeX+stackOffset, screenW, margin)
	} else {
		edgeX = o.clampHorizontal(edgeX+stackOffset, screenW, margin)
	}
	return edgeX, edgeY
}

// clampVertical clamps Y coordinate to screen bounds.
func (o *PulseBeatOverlay) clampVertical(y, screenH, margin float32) float32 {
	if y > screenH-margin-40 {
		return screenH - margin - 40
	}
	return y
}

// clampHorizontal clamps X coordinate to screen bounds.
func (o *PulseBeatOverlay) clampHorizontal(x, screenW, margin float32) float32 {
	if x > screenW-margin-40 {
		return screenW - margin - 40
	}
	return x
}

// calculateFadeAlpha computes alpha based on display time.
func (o *PulseBeatOverlay) calculateFadeAlpha(beat *DisplayBeat, now time.Time) float32 {
	elapsed := now.Sub(beat.DisplayedAt).Seconds()
	displaySecs := o.displayTime.Seconds()
	fadeSecs := o.fadeTime.Seconds()

	alpha := float32(1.0)
	if elapsed > displaySecs-fadeSecs {
		alpha = float32((displaySecs - elapsed) / fadeSecs)
		if alpha < 0 {
			alpha = 0
		}
	}
	return alpha
}

// findEdgeIntersection finds where a ray from center intersects the screen edge.
func (o *PulseBeatOverlay) findEdgeIntersection(centerX, centerY, dirX, dirY, screenW, screenH, margin float32) (float32, float32) {
	tMin := o.findMinEdgeIntersectionParam(centerX, centerY, dirX, dirY, screenW, screenH, margin)

	if tMin == 1e9 {
		return o.clampToMarginBounds(centerX+dirX*100, centerY+dirY*100, screenW, screenH, margin)
	}

	return centerX + tMin*dirX, centerY + tMin*dirY
}

// findMinEdgeIntersectionParam computes the minimum ray parameter for edge intersection.
func (o *PulseBeatOverlay) findMinEdgeIntersectionParam(centerX, centerY, dirX, dirY, screenW, screenH, margin float32) float32 {
	tMin := float32(1e9)

	tMin = o.tryIntersectLeftEdge(centerX, centerY, dirX, dirY, screenH, margin, tMin)
	tMin = o.tryIntersectRightEdge(centerX, centerY, dirX, dirY, screenW, screenH, margin, tMin)
	tMin = o.tryIntersectTopEdge(centerX, centerY, dirX, dirY, screenW, margin, tMin)
	tMin = o.tryIntersectBottomEdge(centerX, centerY, dirX, dirY, screenW, screenH, margin, tMin)

	return tMin
}

// tryIntersectLeftEdge checks intersection with left screen edge.
func (o *PulseBeatOverlay) tryIntersectLeftEdge(centerX, centerY, dirX, dirY, screenH, margin, tMin float32) float32 {
	if dirX >= 0 {
		return tMin
	}
	t := (margin - centerX) / dirX
	if t > 0 && t < tMin {
		y := centerY + t*dirY
		if y >= margin && y <= screenH-margin {
			return t
		}
	}
	return tMin
}

// tryIntersectRightEdge checks intersection with right screen edge.
func (o *PulseBeatOverlay) tryIntersectRightEdge(centerX, centerY, dirX, dirY, screenW, screenH, margin, tMin float32) float32 {
	if dirX <= 0 {
		return tMin
	}
	t := (screenW - margin - centerX) / dirX
	if t > 0 && t < tMin {
		y := centerY + t*dirY
		if y >= margin && y <= screenH-margin {
			return t
		}
	}
	return tMin
}

// tryIntersectTopEdge checks intersection with top screen edge.
func (o *PulseBeatOverlay) tryIntersectTopEdge(centerX, centerY, dirX, dirY, screenW, margin, tMin float32) float32 {
	if dirY >= 0 {
		return tMin
	}
	t := (margin - centerY) / dirY
	if t > 0 && t < tMin {
		x := centerX + t*dirX
		if x >= margin && x <= screenW-margin {
			return t
		}
	}
	return tMin
}

// tryIntersectBottomEdge checks intersection with bottom screen edge.
func (o *PulseBeatOverlay) tryIntersectBottomEdge(centerX, centerY, dirX, dirY, screenW, screenH, margin, tMin float32) float32 {
	if dirY <= 0 {
		return tMin
	}
	t := (screenH - margin - centerY) / dirY
	if t > 0 && t < tMin {
		x := centerX + t*dirX
		if x >= margin && x <= screenW-margin {
			return t
		}
	}
	return tMin
}

// clampToMarginBounds clamps a point to stay within margin bounds.
func (o *PulseBeatOverlay) clampToMarginBounds(x, y, screenW, screenH, margin float32) (float32, float32) {
	if x < margin {
		x = margin
	}
	if x > screenW-margin {
		x = screenW - margin
	}
	if y < margin {
		y = margin
	}
	if y > screenH-margin {
		y = screenH - margin
	}
	return x, y
}

// drawBeatGlyph draws the beat indicator with glyph, background, and pointer.
func (o *PulseBeatOverlay) drawBeatGlyph(screen *ebiten.Image, x, y, dirX, dirY float32, beat *DisplayBeat, alpha float32, index int) {
	// Pulsing effect.
	pulse := float32(math.Sin(o.time*3+float64(index))*0.15 + 0.85)

	// Background pill shape.
	bgWidth := 40 * pulse
	bgHeight := 40 * pulse
	bgAlpha := uint8(float32(o.backgroundColor.A) * alpha)
	bgColor := color.RGBA{o.backgroundColor.R, o.backgroundColor.G, o.backgroundColor.B, bgAlpha}

	// Draw rounded rectangle (approximated with circle + rect).
	vector.DrawFilledCircle(screen, x, y, bgWidth/2, bgColor, true)

	// Draw priority indicator ring for high/urgent.
	if beat.Priority >= BeatPriorityHigh {
		ringPulse := float32(math.Sin(o.time*5)*0.3 + 0.7)
		ringAlpha := uint8(float32(beat.Color.A) * alpha * ringPulse)
		ringColor := color.RGBA{beat.Color.R, beat.Color.G, beat.Color.B, ringAlpha}

		if beat.Priority == BeatPriorityUrgent {
			// Double ring for urgent.
			vector.StrokeCircle(screen, x, y, bgWidth/2+3, 2, ringColor, true)
			vector.StrokeCircle(screen, x, y, bgWidth/2+7, 2, ringColor, true)
		} else {
			// Single ring for high.
			vector.StrokeCircle(screen, x, y, bgWidth/2+3, 2, ringColor, true)
		}
	}

	// Draw glyph shape based on type.
	glyphSize := bgHeight * 0.5
	glyphAlpha := uint8(float32(beat.Color.A) * alpha)
	glyphColor := color.RGBA{beat.Color.R, beat.Color.G, beat.Color.B, glyphAlpha}

	o.drawTypeGlyph(screen, x, y, glyphSize, beat.Type, glyphColor)

	// Draw direction pointer.
	pointerLen := 12 * pulse
	pointerX := x + dirX*bgWidth/2
	pointerY := y + dirY*bgHeight/2

	tipX := pointerX + dirX*pointerLen
	tipY := pointerY + dirY*pointerLen

	pointerAlpha := uint8(float32(o.pointerColor.A) * alpha)
	ptrColor := color.RGBA{o.pointerColor.R, o.pointerColor.G, o.pointerColor.B, pointerAlpha}

	// Draw arrow pointer.
	o.drawArrow(screen, pointerX, pointerY, tipX, tipY, 3, ptrColor)
}

// drawTypeGlyph draws a simplified shape for each beat type.
func (o *PulseBeatOverlay) drawTypeGlyph(screen *ebiten.Image, x, y, size float32, beatType BeatType, c color.RGBA) {
	switch beatType {
	case BeatGift:
		// Diamond shape.
		o.drawDiamond(screen, x, y, size, c)
	case BeatHunt:
		// Filled diamond.
		o.drawDiamond(screen, x, y, size, c)
		vector.DrawFilledCircle(screen, x, y, size*0.3, c, true)
	case BeatForge:
		// Triangle (anvil symbol).
		o.drawTriangle(screen, x, y-size*0.2, size, c)
	case BeatChain:
		// Circle (chain link).
		vector.StrokeCircle(screen, x, y, size*0.4, 3, c, true)
	case BeatTerritory:
		// Square.
		hs := size * 0.4
		vector.DrawFilledRect(screen, x-hs, y-hs, hs*2, hs*2, c, true)
	case BeatSpark:
		// Star shape.
		o.drawStar(screen, x, y, size*0.5, c)
	case BeatPuzzle:
		// Question mark approximation.
		vector.StrokeCircle(screen, x, y-size*0.2, size*0.25, 2, c, true)
		vector.DrawFilledCircle(screen, x, y+size*0.3, size*0.1, c, true)
	case BeatCouncil:
		// Filled circle.
		vector.DrawFilledCircle(screen, x, y, size*0.35, c, true)
	case BeatMark:
		// X mark.
		hs := size * 0.35
		vector.StrokeLine(screen, x-hs, y-hs, x+hs, y+hs, 3, c, true)
		vector.StrokeLine(screen, x+hs, y-hs, x-hs, y+hs, 3, c, true)
	case BeatWave:
		// Wave line.
		for i := 0; i < 3; i++ {
			wx := x - size*0.4 + float32(i)*size*0.4
			wy := y + float32(math.Sin(float64(i)*1.5))*size*0.2
			vector.DrawFilledCircle(screen, wx, wy, size*0.12, c, true)
		}
	default:
		// Default: circle.
		vector.DrawFilledCircle(screen, x, y, size*0.3, c, true)
	}
}

// drawDiamond draws a diamond shape.
func (o *PulseBeatOverlay) drawDiamond(screen *ebiten.Image, x, y, size float32, c color.RGBA) {
	hs := size * 0.4
	// Draw four lines forming diamond.
	vector.StrokeLine(screen, x, y-hs, x+hs, y, 2, c, true)
	vector.StrokeLine(screen, x+hs, y, x, y+hs, 2, c, true)
	vector.StrokeLine(screen, x, y+hs, x-hs, y, 2, c, true)
	vector.StrokeLine(screen, x-hs, y, x, y-hs, 2, c, true)
}

// drawTriangle draws an upward-pointing triangle.
func (o *PulseBeatOverlay) drawTriangle(screen *ebiten.Image, x, y, size float32, c color.RGBA) {
	hs := size * 0.4
	// Top point.
	topX, topY := x, y-hs
	// Bottom left.
	blX, blY := x-hs, y+hs*0.6
	// Bottom right.
	brX, brY := x+hs, y+hs*0.6

	vector.StrokeLine(screen, topX, topY, blX, blY, 2, c, true)
	vector.StrokeLine(screen, blX, blY, brX, brY, 2, c, true)
	vector.StrokeLine(screen, brX, brY, topX, topY, 2, c, true)
}

// drawStar draws a simple star shape.
func (o *PulseBeatOverlay) drawStar(screen *ebiten.Image, x, y, size float32, c color.RGBA) {
	// 5-pointed star using lines from center.
	for i := 0; i < 5; i++ {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		px := x + size*float32(math.Cos(angle))
		py := y + size*float32(math.Sin(angle))
		vector.StrokeLine(screen, x, y, px, py, 2, c, true)
	}
}

// drawArrow draws an arrow from start to end point.
func (o *PulseBeatOverlay) drawArrow(screen *ebiten.Image, x1, y1, x2, y2, width float32, c color.RGBA) {
	// Main line.
	vector.StrokeLine(screen, x1, y1, x2, y2, width, c, true)

	// Arrowhead.
	dx := x2 - x1
	dy := y2 - y1
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 1 {
		return
	}

	// Normalize.
	dx /= length
	dy /= length

	// Arrowhead size.
	headLen := length * 0.4
	headWidth := headLen * 0.6

	// Perpendicular.
	perpX := -dy
	perpY := dx

	// Arrowhead points.
	baseX := x2 - dx*headLen
	baseY := y2 - dy*headLen

	left := struct{ x, y float32 }{baseX + perpX*headWidth/2, baseY + perpY*headWidth/2}
	right := struct{ x, y float32 }{baseX - perpX*headWidth/2, baseY - perpY*headWidth/2}

	// Draw arrowhead lines.
	vector.StrokeLine(screen, x2, y2, left.x, left.y, width, c, true)
	vector.StrokeLine(screen, x2, y2, right.x, right.y, width, c, true)
}

// BeatCount returns the number of active beats.
func (o *PulseBeatOverlay) BeatCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.beats)
}

// ClearBeats removes all beats.
func (o *PulseBeatOverlay) ClearBeats() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.beats = make([]*DisplayBeat, 0)
}

// HandleClick checks if a click is on a beat and triggers callback.
func (o *PulseBeatOverlay) HandleClick(clickX, clickY, cameraX, cameraY, zoom, screenW, screenH float64) bool {
	beats, callback := o.getBeatsAndCallback()
	if callback == nil || len(beats) == 0 {
		return false
	}

	centerX, centerY := screenW/2, screenH/2

	for i, beat := range beats {
		indicatorX, indicatorY := o.calculateIndicatorPosition(beat, i, centerX, centerY, cameraX, cameraY, zoom, screenW, screenH)
		if indicatorX == 0 && indicatorY == 0 {
			continue // Skip invalid indicators
		}

		if o.isClickWithinIndicator(clickX, clickY, indicatorX, indicatorY) {
			callback(beat.ID)
			return true
		}
	}

	return false
}

// getBeatsAndCallback safely retrieves beats and callback under lock.
func (o *PulseBeatOverlay) getBeatsAndCallback() ([]*DisplayBeat, func([32]byte)) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	beats := make([]*DisplayBeat, len(o.beats))
	copy(beats, o.beats)
	return beats, o.onBeatTapped
}

// calculateIndicatorPosition computes the screen position for a beat indicator.
func (o *PulseBeatOverlay) calculateIndicatorPosition(beat *DisplayBeat, index int, centerX, centerY, cameraX, cameraY, zoom, screenW, screenH float64) (float64, float64) {
	targetSX := centerX + (beat.TargetX-cameraX)*zoom
	targetSY := centerY + (beat.TargetY-cameraY)*zoom

	if o.isOnScreen(targetSX, targetSY, screenW, screenH) {
		return targetSX, targetSY
	}
	return o.calculateEdgeIndicatorPosition(targetSX, targetSY, index, centerX, centerY, screenW, screenH)
}

// isOnScreen checks if coordinates are within screen bounds.
func (o *PulseBeatOverlay) isOnScreen(x, y, screenW, screenH float64) bool {
	return x >= 0 && x <= screenW && y >= 0 && y <= screenH
}

// calculateEdgeIndicatorPosition computes edge indicator position with stacking.
func (o *PulseBeatOverlay) calculateEdgeIndicatorPosition(targetSX, targetSY float64, index int, centerX, centerY, screenW, screenH float64) (float64, float64) {
	dx := targetSX - centerX
	dy := targetSY - centerY
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		return 0, 0 // Invalid, signal to skip
	}
	dirX := dx / dist
	dirY := dy / dist

	edgeX, edgeY := o.findEdgeIntersection(
		float32(centerX), float32(centerY),
		float32(dirX), float32(dirY),
		float32(screenW), float32(screenH),
		o.edgeMargin,
	)

	// Apply stack offset for multiple beats at same edge
	stackOffset := float32(index) * 50
	if edgeX <= o.edgeMargin || edgeX >= float32(screenW)-o.edgeMargin {
		edgeY += stackOffset
	} else {
		edgeX += stackOffset
	}

	return float64(edgeX), float64(edgeY)
}

// isClickWithinIndicator checks if click is within 25px of indicator.
func (o *PulseBeatOverlay) isClickWithinIndicator(clickX, clickY, indicatorX, indicatorY float64) bool {
	dx := clickX - indicatorX
	dy := clickY - indicatorY
	clickDist := math.Sqrt(dx*dx + dy*dy)
	return clickDist <= 25
}

// SetMaxVisible sets the maximum number of visible beats.
func (o *PulseBeatOverlay) SetMaxVisible(max int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.maxVisible = max
}

// SetEdgeMargin sets the margin from screen edge.
func (o *PulseBeatOverlay) SetEdgeMargin(margin float32) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.edgeMargin = margin
}

// MarkBeatRead marks a beat as read.
func (o *PulseBeatOverlay) MarkBeatRead(id [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, b := range o.beats {
		if b.ID == id {
			b.Read = true
			return
		}
	}
}

// BeatTypeString returns a human-readable name for a beat type.
func BeatTypeString(t BeatType) string {
	switch t {
	case BeatGift:
		return "Gift"
	case BeatHunt:
		return "Hunt"
	case BeatForge:
		return "Forge"
	case BeatChain:
		return "Chain"
	case BeatTerritory:
		return "Territory"
	case BeatSpark:
		return "Spark"
	case BeatPuzzle:
		return "Puzzle"
	case BeatCouncil:
		return "Council"
	case BeatMark:
		return "Mark"
	case BeatWave:
		return "Wave"
	default:
		return "Unknown"
	}
}

// BeatPriorityString returns a human-readable name for a priority.
func BeatPriorityString(p BeatPriority) string {
	switch p {
	case BeatPriorityLow:
		return "Low"
	case BeatPriorityNormal:
		return "Normal"
	case BeatPriorityHigh:
		return "High"
	case BeatPriorityUrgent:
		return "Urgent"
	default:
		return "Unknown"
	}
}
