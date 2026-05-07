// Package overlays - Whisper Chain Pulse Map visualization.
// Per PULSE_MAP.md: "Whisper Chain activity is deliberately not visualized on the Pulse Map.
// The only visual indication of Whisper Chain activity is a subtle incoming-message indicator
// on the recipient's node: a brief, very small pulse effect (distinct from the Wave publication
// shockwave) that is visible only to the recipient at micro zoom. Other observers cannot
// distinguish a Whisper Chain delivery from normal node activity."
// Per ROADMAP.md line 646: "Whisper Chain indicator — subtle pulse (recipient only, per privacy spec)"

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// WhisperChainIndicator represents a single whisper delivery indicator.
type WhisperChainIndicator struct {
	NodeID     [32]byte  // Recipient node ID.
	X, Y       float64   // Position on Pulse Map.
	ReceivedAt time.Time // When the whisper was received.
	Duration   float64   // Total animation duration in seconds.
	Color      color.RGBA
}

// WhisperChainOverlay renders subtle whisper chain delivery indicators.
type WhisperChainOverlay struct {
	mu sync.RWMutex

	visible       bool
	indicators    map[[32]byte]*WhisperChainIndicator // Key: message ID.
	recipientNode [32]byte                            // The local user's node ID (only this node sees indicators).

	// Visual settings.
	pulseColor    color.RGBA
	minZoomLevel  float64 // Minimum zoom level to show indicators (micro zoom only).
	pulseDuration float64 // Total pulse animation duration in seconds.
}

// NewWhisperChainOverlay creates a new Whisper Chain indicator overlay.
func NewWhisperChainOverlay(recipientNode [32]byte) *WhisperChainOverlay {
	return &WhisperChainOverlay{
		visible:       true,
		indicators:    make(map[[32]byte]*WhisperChainIndicator),
		recipientNode: recipientNode,
		pulseColor: color.RGBA{
			R: 150,
			G: 170,
			B: 200,
			A: 180,
		},
		minZoomLevel:  3.0, // Only visible at micro zoom (>=3x).
		pulseDuration: 1.5, // 1.5 second pulse.
	}
}

// SetVisible controls visibility.
func (o *WhisperChainOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *WhisperChainOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetRecipientNode sets the local user's node ID.
func (o *WhisperChainOverlay) SetRecipientNode(nodeID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.recipientNode = nodeID
}

// AddWhisperDelivery adds a whisper delivery indicator.
// Only the recipient node should call this method.
func (o *WhisperChainOverlay) AddWhisperDelivery(messageID, nodeID [32]byte, x, y float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Only add indicator if this is the recipient node.
	if nodeID != o.recipientNode {
		return
	}

	o.indicators[messageID] = &WhisperChainIndicator{
		NodeID:     nodeID,
		X:          x,
		Y:          y,
		ReceivedAt: time.Now(),
		Duration:   o.pulseDuration,
		Color:      o.pulseColor,
	}
}

// UpdateNodePosition updates the position of a node if it has an active indicator.
func (o *WhisperChainOverlay) UpdateNodePosition(nodeID [32]byte, x, y float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, indicator := range o.indicators {
		if indicator.NodeID == nodeID {
			indicator.X = x
			indicator.Y = y
		}
	}
}

// Update advances animation state and removes expired indicators.
func (o *WhisperChainOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	for id, indicator := range o.indicators {
		elapsed := now.Sub(indicator.ReceivedAt).Seconds()
		if elapsed >= indicator.Duration {
			delete(o.indicators, id)
		}
	}
}

// Draw renders the whisper chain indicators.
func (o *WhisperChainOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	// Only render at micro zoom level.
	if zoom < o.minZoomLevel {
		return
	}

	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())
	centerX := screenW / 2
	centerY := screenH / 2

	now := time.Now()

	for _, indicator := range o.indicators {
		// Calculate animation progress.
		elapsed := now.Sub(indicator.ReceivedAt).Seconds()
		if elapsed >= indicator.Duration {
			continue
		}

		progress := elapsed / indicator.Duration

		// Transform to screen coordinates.
		sx, sy := worldToScreen(indicator.X, indicator.Y, cameraX, cameraY, centerX, centerY, zoom)

		// Skip if off-screen.
		margin := 50.0
		if sx < -margin || sx > screenW+margin || sy < -margin || sy > screenH+margin {
			continue
		}

		o.drawPulse(screen, float32(sx), float32(sy), float32(zoom), float32(progress))
	}
}

// drawPulse draws a subtle pulse effect at the indicator position.
func (o *WhisperChainOverlay) drawPulse(screen *ebiten.Image, x, y, zoom, progress float32) {
	// Pulse grows from 0 to peak at 0.3 progress, then fades.
	peakTime := float32(0.3)
	var intensity float32
	if progress < peakTime {
		intensity = progress / peakTime
	} else {
		intensity = 1.0 - (progress-peakTime)/(1.0-peakTime)
	}

	// Subtle size — much smaller than Wave propagation pulse.
	baseSize := 4.0 * zoom
	if baseSize < 2 {
		baseSize = 2
	}
	if baseSize > 6 {
		baseSize = 6
	}

	// Expand during pulse.
	expansionFactor := float32(1.0 + progress*0.5)
	pulseSize := baseSize * expansionFactor

	// Alpha fades with intensity.
	pulseAlpha := uint8(float32(o.pulseColor.A) * intensity)
	pulseColor := color.RGBA{
		R: o.pulseColor.R,
		G: o.pulseColor.G,
		B: o.pulseColor.B,
		A: pulseAlpha,
	}

	// Draw outer glow (very subtle).
	glowSize := pulseSize * 1.5
	glowAlpha := uint8(float32(pulseAlpha) / 3)
	glowColor := color.RGBA{
		R: pulseColor.R,
		G: pulseColor.G,
		B: pulseColor.B,
		A: glowAlpha,
	}
	vector.DrawFilledCircle(screen, x, y, glowSize, glowColor, true)

	// Draw core pulse.
	vector.DrawFilledCircle(screen, x, y, pulseSize, pulseColor, true)

	// Draw inner highlight (peak brightness).
	if progress < peakTime {
		highlightSize := pulseSize * 0.4
		highlightAlpha := uint8(float32(200) * intensity)
		highlightColor := color.RGBA{200, 220, 240, highlightAlpha}
		vector.DrawFilledCircle(screen, x, y, highlightSize, highlightColor, true)
	}
}

// IndicatorCount returns the total number of active indicators.
func (o *WhisperChainOverlay) IndicatorCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.indicators)
}

// ClearExpired removes expired indicators.
func (o *WhisperChainOverlay) ClearExpired() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, indicator := range o.indicators {
		elapsed := now.Sub(indicator.ReceivedAt).Seconds()
		if elapsed >= indicator.Duration {
			delete(o.indicators, id)
			removed++
		}
	}

	return removed
}

// ClearAll removes all indicators.
func (o *WhisperChainOverlay) ClearAll() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.indicators = make(map[[32]byte]*WhisperChainIndicator)
}

// GetActiveIndicators returns all active indicators.
func (o *WhisperChainOverlay) GetActiveIndicators() []*WhisperChainIndicator {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	var active []*WhisperChainIndicator

	for _, indicator := range o.indicators {
		elapsed := now.Sub(indicator.ReceivedAt).Seconds()
		if elapsed < indicator.Duration {
			active = append(active, indicator)
		}
	}

	return active
}

// SetPulseDuration sets the pulse animation duration.
func (o *WhisperChainOverlay) SetPulseDuration(duration float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pulseDuration = duration
}

// SetMinZoomLevel sets the minimum zoom level to show indicators.
func (o *WhisperChainOverlay) SetMinZoomLevel(zoom float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.minZoomLevel = zoom
}

// GetMinZoomLevel returns the minimum zoom level.
func (o *WhisperChainOverlay) GetMinZoomLevel() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.minZoomLevel
}
