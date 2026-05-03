// Package overlays - Sigil Forge Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Active Forge events appear on the Pulse Map
// as a glowing anvil glyph. During a Sigil Art Forge, submitted Sigil Waves
// orbit the glyph as miniature animated thumbnails".
// Per ROADMAP.md line 476: "Pulse Map visualization — anvil-and-flame icon
// with orbiting entries".
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ForgeType represents the type of Sigil Forge event.
type ForgeType uint8

const (
	ForgeSigilArt   ForgeType = iota // Sigil art creation challenge.
	ForgeMicroFic                    // Micro-fiction writing challenge.
	ForgeRemixChain                  // Collaborative remix chain.
)

// ForgeState represents the current state of a forge event.
type ForgeState uint8

const (
	ForgeActive    ForgeState = iota // Accepting submissions.
	ForgeEvaluate                    // Evaluation period.
	ForgeCompleted                   // Event finished, winner determined.
)

// ForgeEntry represents a submission to a Sigil Forge.
type ForgeEntry struct {
	EntryID    [32]byte   // Unique entry identifier.
	SpecterKey [32]byte   // Submitter's Specter key.
	SigilColor color.RGBA // Color derived from entry sigil.
	Score      float64    // Amplification score.
	IsWinner   bool       // True if this entry won.
}

// ForgeEventInfo contains information about a Sigil Forge event.
type ForgeEventInfo struct {
	ForgeID   [32]byte   // Unique forge identifier.
	Type      ForgeType  // Type of forge event.
	State     ForgeState // Current state.
	X, Y      float64    // Position on the Pulse Map.
	StartTime time.Time  // When the forge started.
	EndTime   time.Time  // When submissions close.
	Entries   []ForgeEntry
}

// ForgeOverlay renders Sigil Forge events on the Pulse Map.
type ForgeOverlay struct {
	mu sync.RWMutex

	visible bool
	forges  map[[32]byte]*ForgeEventInfo
	time    float64 // Animation time.

	// Visual settings.
	anvilColor      color.RGBA
	flameColor      color.RGBA
	entryOrbitColor color.RGBA
	glowColor       color.RGBA
	winnerColor     color.RGBA
}

// NewForgeOverlay creates a new Sigil Forge overlay.
func NewForgeOverlay() *ForgeOverlay {
	return &ForgeOverlay{
		visible: true,
		forges:  make(map[[32]byte]*ForgeEventInfo),
		anvilColor: color.RGBA{
			R: 100,
			G: 110,
			B: 120,
			A: 255,
		},
		flameColor: color.RGBA{
			R: 255,
			G: 140,
			B: 40,
			A: 255,
		},
		entryOrbitColor: color.RGBA{
			R: 80,
			G: 200,
			B: 220,
			A: 200,
		},
		glowColor: color.RGBA{
			R: 255,
			G: 180,
			B: 80,
			A: 100,
		},
		winnerColor: color.RGBA{
			R: 255,
			G: 215,
			B: 0,
			A: 255,
		},
	}
}

// SetVisible controls visibility.
func (o *ForgeOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *ForgeOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetForge adds or updates a forge event.
func (o *ForgeOverlay) SetForge(forge *ForgeEventInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.forges[forge.ForgeID] = forge
}

// RemoveForge removes a forge event.
func (o *ForgeOverlay) RemoveForge(forgeID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.forges, forgeID)
}

// GetForge returns a forge event by ID.
func (o *ForgeOverlay) GetForge(forgeID [32]byte) *ForgeEventInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.forges[forgeID]
}

// GetAllForges returns all active forge events.
func (o *ForgeOverlay) GetAllForges() []*ForgeEventInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	forges := make([]*ForgeEventInfo, 0, len(o.forges))
	for _, f := range o.forges {
		forges = append(forges, f)
	}
	return forges
}

// Update advances animation state.
func (o *ForgeOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.time += dt
}

// Draw renders the forge events.
func (o *ForgeOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	screenW, screenH, centerX, centerY := getCameraSetup(screen)

	for _, forge := range o.forges {
		sx, sy := worldToScreen(forge.X, forge.Y, cameraX, cameraY, centerX, centerY, zoom)

		// Skip if off-screen.
		if isOffScreen(sx, sy, screenW, screenH, 100) {
			continue
		}

		o.drawForgeIcon(screen, float32(sx), float32(sy), float32(zoom), forge)
	}
}

// drawForgeIcon draws the anvil-and-flame icon with orbiting entries.
func (o *ForgeOverlay) drawForgeIcon(screen *ebiten.Image, x, y, zoom float32, forge *ForgeEventInfo) {
	scale := zoom * 0.5
	if scale < 0.3 {
		scale = 0.3
	}
	if scale > 1.5 {
		scale = 1.5
	}

	// Draw glow behind icon.
	o.drawGlow(screen, x, y, scale, forge.State)

	// Draw anvil shape.
	o.drawAnvil(screen, x, y, scale)

	// Draw flames above anvil.
	o.drawFlames(screen, x, y-20*scale, scale)

	// Draw orbiting entries.
	o.drawOrbitingEntries(screen, x, y, scale, forge)

	// Draw winner highlight if completed.
	if forge.State == ForgeCompleted {
		o.drawWinnerHighlight(screen, x, y, scale, forge)
	}
}

// drawGlow draws the ambient glow around the forge icon.
func (o *ForgeOverlay) drawGlow(screen *ebiten.Image, x, y, scale float32, state ForgeState) {
	// Pulse the glow for active forges.
	pulsePhase := float32(math.Sin(o.time * 2))
	baseRadius := 40 * scale
	pulseRadius := baseRadius + 5*scale*pulsePhase

	// More intense glow for active forges.
	glowAlpha := uint8(60)
	if state == ForgeActive {
		glowAlpha = uint8(80 + 20*pulsePhase)
	}

	glowColor := color.RGBA{
		R: o.glowColor.R,
		G: o.glowColor.G,
		B: o.glowColor.B,
		A: glowAlpha,
	}

	// Draw multiple glow rings.
	for i := 3; i >= 0; i-- {
		radius := pulseRadius * float32(1+i) / 2
		alpha := uint8(float32(glowAlpha) / float32(i+1))
		ringColor := color.RGBA{glowColor.R, glowColor.G, glowColor.B, alpha}
		vector.DrawFilledCircle(screen, x, y, radius, ringColor, true)
	}
}

// drawAnvil draws the anvil shape.
func (o *ForgeOverlay) drawAnvil(screen *ebiten.Image, x, y, scale float32) {
	// Anvil body (trapezoid shape approximated with rectangles).
	// Top surface.
	topW := 30 * scale
	topH := 8 * scale
	vector.DrawFilledRect(screen, x-topW/2, y-topH/2, topW, topH, o.anvilColor, true)

	// Body (slightly narrower).
	bodyW := 24 * scale
	bodyH := 15 * scale
	vector.DrawFilledRect(screen, x-bodyW/2, y+topH/2, bodyW, bodyH, o.anvilColor, true)

	// Base (wider).
	baseW := 35 * scale
	baseH := 6 * scale
	vector.DrawFilledRect(screen, x-baseW/2, y+topH/2+bodyH, baseW, baseH, o.anvilColor, true)

	// Horn (triangular protrusion on one side).
	hornX := x + topW/2
	hornY := y
	hornLen := 10 * scale
	hornH := 5 * scale
	vector.DrawFilledRect(screen, hornX, hornY-hornH/2, hornLen, hornH, o.anvilColor, true)

	// Add highlight on top surface.
	highlightColor := color.RGBA{
		R: o.anvilColor.R + 30,
		G: o.anvilColor.G + 30,
		B: o.anvilColor.B + 30,
		A: 255,
	}
	vector.DrawFilledRect(screen, x-topW/2+2, y-topH/2+1, topW-4, topH/3, highlightColor, true)
}

// drawFlames draws animated flames above the anvil.
func (o *ForgeOverlay) drawFlames(screen *ebiten.Image, x, y, scale float32) {
	// Multiple flame particles.
	numFlames := 5
	for i := 0; i < numFlames; i++ {
		// Offset each flame horizontally.
		offset := float32(i-numFlames/2) * 4 * scale

		// Animate flame height and flicker.
		phase := o.time*3 + float64(i)*0.5
		flicker := float32(math.Sin(phase)*0.3 + 0.7)
		height := 15 * scale * flicker

		// Draw flame (teardrop shape approximated with circles).
		flameX := x + offset
		flameY := y - height/2

		// Gradient from yellow to red.
		t := float32(i) / float32(numFlames)
		flameColor := color.RGBA{
			R: 255,
			G: uint8(180 - 80*t),
			B: uint8(40 * (1 - t)),
			A: uint8(200 * flicker),
		}

		vector.DrawFilledCircle(screen, flameX, flameY, 4*scale*flicker, flameColor, true)

		// Smaller tip.
		tipY := flameY - 6*scale*flicker
		tipColor := color.RGBA{255, 255, 200, uint8(150 * flicker)}
		vector.DrawFilledCircle(screen, flameX, tipY, 2*scale*flicker, tipColor, true)
	}
}

// drawOrbitingEntries draws entry submissions orbiting the forge.
func (o *ForgeOverlay) drawOrbitingEntries(screen *ebiten.Image, x, y, scale float32, forge *ForgeEventInfo) {
	if len(forge.Entries) == 0 {
		return
	}

	// Orbit radius.
	orbitRadius := 55 * scale

	// Limit displayed entries to prevent clutter.
	maxDisplayed := 8
	displayCount := len(forge.Entries)
	if displayCount > maxDisplayed {
		displayCount = maxDisplayed
	}

	// Orbit speed varies by forge state.
	orbitSpeed := 0.3
	if forge.State == ForgeActive {
		orbitSpeed = 0.5
	}

	for i := 0; i < displayCount; i++ {
		entry := &forge.Entries[i]

		// Calculate orbit position.
		angle := o.time*orbitSpeed + float64(i)*2*math.Pi/float64(displayCount)
		entryX := x + orbitRadius*float32(math.Cos(angle))
		entryY := y + orbitRadius*float32(math.Sin(angle))

		// Entry size based on score (larger = higher score).
		baseSize := 6 * scale
		scoreBonus := float32(math.Min(entry.Score/100, 1)) * 4 * scale
		entrySize := baseSize + scoreBonus

		// Use entry's sigil color or default.
		entryColor := entry.SigilColor
		if entryColor.A == 0 {
			entryColor = o.entryOrbitColor
		}

		// Draw orbit trail.
		trailAlpha := uint8(30)
		trailColor := color.RGBA{entryColor.R, entryColor.G, entryColor.B, trailAlpha}
		for j := 1; j <= 3; j++ {
			trailAngle := angle - float64(j)*0.1
			trailX := x + orbitRadius*float32(math.Cos(trailAngle))
			trailY := y + orbitRadius*float32(math.Sin(trailAngle))
			trailSize := entrySize * float32(1-float64(j)*0.2)
			vector.DrawFilledCircle(screen, trailX, trailY, trailSize, trailColor, true)
		}

		// Draw entry marker.
		vector.DrawFilledCircle(screen, entryX, entryY, entrySize, entryColor, true)

		// Winner gets a special ring.
		if entry.IsWinner {
			vector.StrokeCircle(screen, entryX, entryY, entrySize+3*scale, 2, o.winnerColor, true)
		}
	}

	// Show count if there are more entries.
	if len(forge.Entries) > maxDisplayed {
		// Draw indicator at bottom to show there are more entries.
		indicatorY := y + orbitRadius + 15*scale
		indicatorColor := color.RGBA{200, 200, 200, 180}
		vector.DrawFilledCircle(screen, x, indicatorY, 8*scale, indicatorColor, true)
		// Additional entries beyond maxDisplayed are indicated by this marker.
	}
}

// drawWinnerHighlight draws special effect for completed forge with winner.
func (o *ForgeOverlay) drawWinnerHighlight(screen *ebiten.Image, x, y, scale float32, forge *ForgeEventInfo) {
	// Find winner entry.
	var winner *ForgeEntry
	for i := range forge.Entries {
		if forge.Entries[i].IsWinner {
			winner = &forge.Entries[i]
			break
		}
	}

	if winner == nil {
		return
	}

	// Draw winner's sigil at the center (larger than orbiting entries).
	winnerSize := 12 * scale
	winnerColor := winner.SigilColor
	if winnerColor.A == 0 {
		winnerColor = o.winnerColor
	}

	// Pulsing effect.
	pulse := float32(math.Sin(o.time*2)*0.2 + 1)
	pulsedSize := winnerSize * pulse

	// Draw golden ring.
	vector.StrokeCircle(screen, x, y-30*scale, pulsedSize+5*scale, 3, o.winnerColor, true)

	// Draw winner sigil.
	vector.DrawFilledCircle(screen, x, y-30*scale, pulsedSize, winnerColor, true)

	// Sparkle effects.
	numSparkles := 6
	sparkleRadius := 25 * scale
	for i := 0; i < numSparkles; i++ {
		angle := o.time*1.5 + float64(i)*math.Pi/3
		sparkleX := x + sparkleRadius*float32(math.Cos(angle))
		sparkleY := y - 30*scale + sparkleRadius*float32(math.Sin(angle))
		sparkleSize := 2 * scale * float32(math.Sin(o.time*3+float64(i))*0.5+0.5)
		sparkleColor := color.RGBA{255, 255, 220, 200}
		vector.DrawFilledCircle(screen, sparkleX, sparkleY, sparkleSize, sparkleColor, true)
	}
}

// ForgeCount returns the number of active forge events.
func (o *ForgeOverlay) ForgeCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.forges)
}

// ClearCompleted removes all completed forge events.
func (o *ForgeOverlay) ClearCompleted() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	var removed int
	for id, forge := range o.forges {
		if forge.State == ForgeCompleted && time.Since(forge.EndTime) > 24*time.Hour {
			delete(o.forges, id)
			removed++
		}
	}
	return removed
}

// ForgeTypeString returns human-readable name for a forge type.
func ForgeTypeString(ft ForgeType) string {
	switch ft {
	case ForgeSigilArt:
		return "Sigil Art"
	case ForgeMicroFic:
		return "Micro-Fiction"
	case ForgeRemixChain:
		return "Remix Chain"
	default:
		return "Unknown"
	}
}

// ForgeStateString returns human-readable name for a forge state.
func ForgeStateString(fs ForgeState) string {
	switch fs {
	case ForgeActive:
		return "Active"
	case ForgeEvaluate:
		return "Evaluating"
	case ForgeCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}
