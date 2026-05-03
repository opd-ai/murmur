// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file implements Specter Mark visualization on the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, marks are anonymous annotations that appear
// as orbiting sigil icons on marked Surface nodes.
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

// MarkDisplay represents a single mark being displayed on a node.
type MarkDisplay struct {
	Mark       *marks.Mark // The underlying mark
	OrbitAngle float32     // Current orbital position (radians)
	OrbitSpeed float32     // Radians per second
	PulsePhase float32     // Pulse animation phase
}

// MarkOverlay manages Specter Mark visualization on the Pulse Map.
// Per ROADMAP.md line 533, shows orbiting sigil icons on marked Surface nodes.
type MarkOverlay struct {
	mu         sync.RWMutex
	marks      map[string][]*MarkDisplay // By target node ID (hex pubkey)
	orbitBase  float32                   // Base orbit radius
	pulseSpeed float32                   // Pulse animation speed
}

// NewMarkOverlay creates a new mark overlay manager.
func NewMarkOverlay() *MarkOverlay {
	return &MarkOverlay{
		marks:      make(map[string][]*MarkDisplay),
		orbitBase:  24.0, // Base orbit radius in pixels
		pulseSpeed: 1.5,  // Pulse cycles per second
	}
}

// AddMark registers a mark for display on a target node.
func (o *MarkOverlay) AddMark(targetID string, mark *marks.Mark) {
	if mark == nil || mark.IsExpired() {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Check for duplicates.
	for _, d := range o.marks[targetID] {
		if d.Mark != nil && d.Mark.ID == mark.ID {
			return // Already added
		}
	}

	// Calculate unique orbit speed based on mark ID for variety.
	orbitSpeed := 0.5 + float32(mark.ID[0]%64)/128.0 // 0.5 to 1.0 rad/sec

	o.marks[targetID] = append(o.marks[targetID], &MarkDisplay{
		Mark:       mark,
		OrbitAngle: float32(mark.ID[1]) / 40.0, // Start at different angles
		OrbitSpeed: orbitSpeed,
		PulsePhase: 0,
	})
}

// RemoveMark removes a specific mark from display.
func (o *MarkOverlay) RemoveMark(markID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for targetID, displays := range o.marks {
		for i, d := range displays {
			if d.Mark != nil && d.Mark.ID == markID {
				o.marks[targetID] = append(displays[:i], displays[i+1:]...)
				if len(o.marks[targetID]) == 0 {
					delete(o.marks, targetID)
				}
				return
			}
		}
	}
}

// RemoveAllMarksForTarget removes all marks from a target node.
func (o *MarkOverlay) RemoveAllMarksForTarget(targetID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.marks, targetID)
}

// ClearExpiredMarks removes all expired marks.
func (o *MarkOverlay) ClearExpiredMarks() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for targetID, displays := range o.marks {
		active := displays[:0]
		for _, d := range displays {
			if d.Mark != nil && !d.Mark.IsExpired() {
				active = append(active, d)
			}
		}
		if len(active) > 0 {
			o.marks[targetID] = active
		} else {
			delete(o.marks, targetID)
		}
	}
}

// Update advances animation state for all marks.
func (o *MarkOverlay) Update(dt float32) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, displays := range o.marks {
		for _, d := range displays {
			// Advance orbit.
			d.OrbitAngle += d.OrbitSpeed * dt
			if d.OrbitAngle > 2*math.Pi {
				d.OrbitAngle -= 2 * math.Pi
			}

			// Advance pulse animation.
			d.PulsePhase += o.pulseSpeed * dt
			if d.PulsePhase > 2*math.Pi {
				d.PulsePhase -= 2 * math.Pi
			}
		}
	}
}

// Render draws mark overlays for a specific node.
func (o *MarkOverlay) Render(screen *ebiten.Image, targetID string, nodeX, nodeY float32) {
	o.mu.RLock()
	displays := o.marks[targetID]
	o.mu.RUnlock()

	if len(displays) == 0 {
		return
	}

	// Calculate orbit radii to distribute marks.
	count := len(displays)
	for i, d := range displays {
		if d.Mark == nil || d.Mark.IsExpired() {
			continue
		}

		// Stack orbits slightly for multiple marks.
		orbitRadius := o.orbitBase + float32(i)*6.0

		// Calculate orbit position.
		x := nodeX + float32(math.Cos(float64(d.OrbitAngle)))*orbitRadius
		y := nodeY + float32(math.Sin(float64(d.OrbitAngle)))*orbitRadius

		// Get visibility (fades over 30 days).
		visibility := float32(d.Mark.CurrentVisibility())

		// Render based on category.
		o.renderMark(screen, d, x, y, visibility, count)
	}
}

// renderMark draws a single mark icon.
func (o *MarkOverlay) renderMark(screen *ebiten.Image, d *MarkDisplay, x, y, visibility float32, totalMarks int) {
	// Pulse effect modulates size and alpha.
	pulseFactor := 1.0 + 0.15*float32(math.Sin(float64(d.PulsePhase)))

	// Base size for mark icon.
	baseSize := float32(8.0)
	if totalMarks > 3 {
		baseSize = 6.0 // Smaller when crowded
	}
	size := baseSize * pulseFactor

	// Get category color.
	clr := o.getCategoryColor(d.Mark.Category)
	clr.A = uint8(float32(clr.A) * visibility)

	// Draw mark icon based on category.
	switch d.Mark.Category {
	case marks.MarkWatcher:
		o.drawWatcherIcon(screen, x, y, size, clr)
	case marks.MarkAlly:
		o.drawAllyIcon(screen, x, y, size, clr)
	case marks.MarkRival:
		o.drawRivalIcon(screen, x, y, size, clr)
	default:
		// Fallback: simple circle.
		vector.DrawFilledCircle(screen, x, y, size/2, clr, true)
	}

	// Draw glow effect.
	o.drawGlow(screen, x, y, size*1.5, clr, visibility*0.3)
}

// getCategoryColor returns the display color for a mark category.
func (o *MarkOverlay) getCategoryColor(cat marks.MarkCategory) color.RGBA {
	switch cat {
	case marks.MarkWatcher:
		// Neutral blue-gray for observation.
		return color.RGBA{R: 130, G: 150, B: 180, A: 200}
	case marks.MarkAlly:
		// Warm green for positive association.
		return color.RGBA{R: 100, G: 200, B: 130, A: 200}
	case marks.MarkRival:
		// Deep red for adversarial.
		return color.RGBA{R: 200, G: 80, B: 80, A: 200}
	default:
		return color.RGBA{R: 150, G: 150, B: 150, A: 200}
	}
}

// drawWatcherIcon draws an eye-like icon for Watcher marks.
func (o *MarkOverlay) drawWatcherIcon(screen *ebiten.Image, x, y, size float32, clr color.RGBA) {
	// Outer eye shape (horizontal ellipse).
	halfW := size * 0.8
	halfH := size * 0.4

	// Draw eye outline.
	for angle := float32(0); angle < 2*math.Pi; angle += 0.1 {
		px := x + halfW*float32(math.Cos(float64(angle)))
		py := y + halfH*float32(math.Sin(float64(angle)))
		vector.DrawFilledCircle(screen, px, py, 1.0, clr, true)
	}

	// Inner pupil.
	vector.DrawFilledCircle(screen, x, y, size*0.25, clr, true)
}

// drawAllyIcon draws a shield-like icon for Ally marks.
func (o *MarkOverlay) drawAllyIcon(screen *ebiten.Image, x, y, size float32, clr color.RGBA) {
	// Simple shield shape using triangular approximation.
	topY := y - size*0.4
	bottomY := y + size*0.5
	leftX := x - size*0.4
	rightX := x + size*0.4

	// Draw shield outline.
	vector.StrokeLine(screen, leftX, topY, x, bottomY, 1.5, clr, true)
	vector.StrokeLine(screen, rightX, topY, x, bottomY, 1.5, clr, true)
	vector.StrokeLine(screen, leftX, topY, rightX, topY, 1.5, clr, true)

	// Inner checkmark.
	vector.StrokeLine(screen, x-size*0.15, y, x, y+size*0.15, 1.0, clr, true)
	vector.StrokeLine(screen, x, y+size*0.15, x+size*0.2, y-size*0.15, 1.0, clr, true)
}

// drawRivalIcon draws a crossed-swords icon for Rival marks.
func (o *MarkOverlay) drawRivalIcon(screen *ebiten.Image, x, y, size float32, clr color.RGBA) {
	halfDiag := size * 0.5

	// First sword (top-left to bottom-right).
	vector.StrokeLine(screen, x-halfDiag, y-halfDiag, x+halfDiag, y+halfDiag, 1.5, clr, true)

	// Second sword (top-right to bottom-left).
	vector.StrokeLine(screen, x+halfDiag, y-halfDiag, x-halfDiag, y+halfDiag, 1.5, clr, true)

	// Center circle.
	vector.DrawFilledCircle(screen, x, y, size*0.15, clr, true)
}

// drawGlow draws a soft glow effect around the mark.
func (o *MarkOverlay) drawGlow(screen *ebiten.Image, x, y, radius float32, clr color.RGBA, alpha float32) {
	// Simple radial glow using concentric circles.
	steps := 4
	for i := 0; i < steps; i++ {
		r := radius * (1.0 - float32(i)/float32(steps))
		a := alpha * float32(i) / float32(steps)
		glowClr := color.RGBA{
			R: clr.R,
			G: clr.G,
			B: clr.B,
			A: uint8(float32(clr.A) * a),
		}
		vector.DrawFilledCircle(screen, x, y, r, glowClr, true)
	}
}

// GetMarkCount returns the number of marks on a target.
func (o *MarkOverlay) GetMarkCount(targetID string) int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.marks[targetID])
}

// GetTotalMarkCount returns total marks being displayed.
func (o *MarkOverlay) GetTotalMarkCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	total := 0
	for _, displays := range o.marks {
		total += len(displays)
	}
	return total
}

// HasMarks returns true if the target has any visible marks.
func (o *MarkOverlay) HasMarks(targetID string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.marks[targetID]) > 0
}

// GetDominantCategory returns the most common mark category for a target.
func (o *MarkOverlay) GetDominantCategory(targetID string) marks.MarkCategory {
	o.mu.RLock()
	displays := o.marks[targetID]
	o.mu.RUnlock()

	if len(displays) == 0 {
		return 0
	}

	counts := make(map[marks.MarkCategory]int)
	for _, d := range displays {
		if d.Mark != nil {
			counts[d.Mark.Category]++
		}
	}

	var dominant marks.MarkCategory
	maxCount := 0
	for cat, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = cat
		}
	}
	return dominant
}

// SyncFromStore updates the overlay from a MarkStore.
func (o *MarkOverlay) SyncFromStore(store *marks.MarkStore) {
	if store == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Clear expired marks first.
	for targetID, displays := range o.marks {
		active := displays[:0]
		for _, d := range displays {
			if d.Mark != nil && !d.Mark.IsExpired() {
				active = append(active, d)
			}
		}
		if len(active) > 0 {
			o.marks[targetID] = active
		} else {
			delete(o.marks, targetID)
		}
	}

	// Add any new marks from store.
	allMarks := store.GetAllActiveMarks()
	for _, mark := range allMarks {
		if mark == nil || mark.IsExpired() {
			continue
		}

		targetID := keyToHex(mark.TargetKey[:])

		// Check if already tracked.
		found := false
		for _, d := range o.marks[targetID] {
			if d.Mark != nil && d.Mark.ID == mark.ID {
				found = true
				break
			}
		}

		if !found {
			orbitSpeed := 0.5 + float32(mark.ID[0]%64)/128.0
			o.marks[targetID] = append(o.marks[targetID], &MarkDisplay{
				Mark:       mark,
				OrbitAngle: float32(mark.ID[1]) / 40.0,
				OrbitSpeed: orbitSpeed,
				PulsePhase: 0,
			})
		}
	}
}

// keyToHex converts a byte slice to hex string.
func keyToHex(key []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(key)*2)
	for i, b := range key {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}
