// Package overlays - Whisper Chain test stub.

//go:build test
// +build test

package overlays

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// WhisperChainIndicator represents a single whisper delivery indicator.
type WhisperChainIndicator struct {
	NodeID     [32]byte
	X, Y       float64
	ReceivedAt time.Time
	Duration   float64
	Color      color.RGBA
}

// WhisperChainOverlay test stub.
type WhisperChainOverlay struct {
	visible       bool
	indicators    map[[32]byte]*WhisperChainIndicator
	recipientNode [32]byte
	pulseColor    color.RGBA
	minZoomLevel  float64
	pulseDuration float64
}

// NewWhisperChainOverlay creates a stub overlay.
func NewWhisperChainOverlay(recipientNode [32]byte) *WhisperChainOverlay {
	return &WhisperChainOverlay{
		visible:       true,
		indicators:    make(map[[32]byte]*WhisperChainIndicator),
		recipientNode: recipientNode,
		pulseColor:    color.RGBA{150, 170, 200, 180},
		minZoomLevel:  3.0,
		pulseDuration: 1.5,
	}
}

// SetVisible controls visibility.
func (o *WhisperChainOverlay) SetVisible(visible bool) {
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *WhisperChainOverlay) IsVisible() bool {
	return o.visible
}

// SetRecipientNode sets the local user's node ID.
func (o *WhisperChainOverlay) SetRecipientNode(nodeID [32]byte) {
	o.recipientNode = nodeID
}

// AddWhisperDelivery adds a whisper delivery indicator.
func (o *WhisperChainOverlay) AddWhisperDelivery(messageID, nodeID [32]byte, x, y float64) {
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
	for _, indicator := range o.indicators {
		if indicator.NodeID == nodeID {
			indicator.X = x
			indicator.Y = y
		}
	}
}

// Update advances animation state and removes expired indicators.
func (o *WhisperChainOverlay) Update(dt float64) {
	now := time.Now()
	for id, indicator := range o.indicators {
		elapsed := now.Sub(indicator.ReceivedAt).Seconds()
		if elapsed >= indicator.Duration {
			delete(o.indicators, id)
		}
	}
}

// Draw renders the whisper chain indicators (stub: no-op).
func (o *WhisperChainOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	// Stub: no rendering.
}

// IndicatorCount returns the total number of active indicators.
func (o *WhisperChainOverlay) IndicatorCount() int {
	return len(o.indicators)
}

// ClearExpired removes expired indicators.
func (o *WhisperChainOverlay) ClearExpired() int {
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
	o.indicators = make(map[[32]byte]*WhisperChainIndicator)
}

// GetActiveIndicators returns all active indicators.
func (o *WhisperChainOverlay) GetActiveIndicators() []*WhisperChainIndicator {
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
	o.pulseDuration = duration
}

// SetMinZoomLevel sets the minimum zoom level to show indicators.
func (o *WhisperChainOverlay) SetMinZoomLevel(zoom float64) {
	o.minZoomLevel = zoom
}

// GetMinZoomLevel returns the minimum zoom level.
func (o *WhisperChainOverlay) GetMinZoomLevel() float64 {
	return o.minZoomLevel
}
