// Package overlays - Pulse Beat overlay stub for non-Ebiten builds.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"image/color"
	"time"
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
	ID          [32]byte
	Type        BeatType
	Priority    BeatPriority
	Title       string
	TargetX     float64
	TargetY     float64
	CreatedAt   time.Time
	DisplayedAt time.Time
	Color       color.RGBA
	Read        bool
}

// PulseBeatOverlay is a stub for non-Ebiten builds.
type PulseBeatOverlay struct {
	visible    bool
	beats      []*DisplayBeat
	maxVisible int
}

// NewPulseBeatOverlay creates a new stub overlay.
func NewPulseBeatOverlay() *PulseBeatOverlay {
	return &PulseBeatOverlay{
		visible:    true,
		beats:      make([]*DisplayBeat, 0),
		maxVisible: 3,
	}
}

// SetVisible controls visibility.
func (o *PulseBeatOverlay) SetVisible(visible bool) { o.visible = visible }

// IsVisible returns visibility status.
func (o *PulseBeatOverlay) IsVisible() bool { return o.visible }

// AddBeat adds a beat to the display queue.
func (o *PulseBeatOverlay) AddBeat(beat *DisplayBeat) {
	if beat == nil {
		return
	}
	if beat.DisplayedAt.IsZero() {
		beat.DisplayedAt = time.Now()
	}
	// Check if beat already exists.
	for i, b := range o.beats {
		if b.ID == beat.ID {
			o.beats[i] = beat
			return
		}
	}
	o.beats = append(o.beats, beat)
	if len(o.beats) > o.maxVisible {
		o.beats = o.beats[:o.maxVisible]
	}
}

// RemoveBeat removes a beat by ID.
func (o *PulseBeatOverlay) RemoveBeat(id [32]byte) {
	for i, b := range o.beats {
		if b.ID == id {
			o.beats = append(o.beats[:i], o.beats[i+1:]...)
			return
		}
	}
}

// GetBeat returns a beat by ID.
func (o *PulseBeatOverlay) GetBeat(id [32]byte) *DisplayBeat {
	for _, b := range o.beats {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// SetOnBeatTapped is a no-op stub.
func (o *PulseBeatOverlay) SetOnBeatTapped(cb func(beatID [32]byte)) {}

// Update is a no-op stub.
func (o *PulseBeatOverlay) Update(dt float64) {}

// BeatCount returns the number of active beats.
func (o *PulseBeatOverlay) BeatCount() int { return len(o.beats) }

// ClearBeats removes all beats.
func (o *PulseBeatOverlay) ClearBeats() { o.beats = make([]*DisplayBeat, 0) }

// HandleClick is a no-op stub.
func (o *PulseBeatOverlay) HandleClick(clickX, clickY, cameraX, cameraY, zoom, screenW, screenH float64) bool {
	return false
}

// SetMaxVisible sets the maximum number of visible beats.
func (o *PulseBeatOverlay) SetMaxVisible(max int) { o.maxVisible = max }

// SetEdgeMargin is a no-op stub.
func (o *PulseBeatOverlay) SetEdgeMargin(margin float32) {}

// MarkBeatRead marks a beat as read.
func (o *PulseBeatOverlay) MarkBeatRead(id [32]byte) {
	for _, b := range o.beats {
		if b.ID == id {
			b.Read = true
			return
		}
	}
}

// BeatTypeString returns a human-readable name.
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

// BeatPriorityString returns a human-readable name.
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
