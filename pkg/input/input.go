// Package input normalizes desktop and mobile/browser events into shared gameplay actions.
package input

import "time"

// SourceType indicates original device source.
type SourceType string

const (
	SourceMouse    SourceType = "mouse"
	SourceKeyboard SourceType = "keyboard"
	SourceTouch    SourceType = "touch"
)

// EventType represents low-level input transitions.
type EventType string

const (
	EventDown EventType = "down"
	EventUp   EventType = "up"
	EventMove EventType = "move"
	EventKey  EventType = "key"
)

// Action represents normalized gameplay intents.
type Action string

const (
	ActionPrimaryPress   Action = "primary_press"
	ActionPrimaryRelease Action = "primary_release"
	ActionPan            Action = "pan"
	ActionZoomIn         Action = "zoom_in"
	ActionZoomOut        Action = "zoom_out"
	ActionOpenMenu       Action = "open_menu"
)

// RawEvent is a platform event before normalization.
type RawEvent struct {
	Source    SourceType
	Type      EventType
	X         float64
	Y         float64
	Delta     float64
	Key       string
	PointerID int
	Timestamp time.Time
}

// NormalizedAction is emitted by Mapper and consumed by gameplay/UI code.
type NormalizedAction struct {
	Action    Action
	X         float64
	Y         float64
	Source    SourceType
	Timestamp time.Time
}

// Mapper translates raw platform input into normalized actions.
type Mapper struct{}

// NewMapper constructs an input mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// Normalize maps one raw input event to a normalized action set.
func (m *Mapper) Normalize(evt RawEvent) []NormalizedAction {
	switch evt.Type {
	case EventDown:
		return []NormalizedAction{{
			Action:    ActionPrimaryPress,
			X:         evt.X,
			Y:         evt.Y,
			Source:    evt.Source,
			Timestamp: evt.Timestamp,
		}}
	case EventUp:
		return []NormalizedAction{{
			Action:    ActionPrimaryRelease,
			X:         evt.X,
			Y:         evt.Y,
			Source:    evt.Source,
			Timestamp: evt.Timestamp,
		}}
	case EventMove:
		return []NormalizedAction{{
			Action:    ActionPan,
			X:         evt.X,
			Y:         evt.Y,
			Source:    evt.Source,
			Timestamp: evt.Timestamp,
		}}
	case EventKey:
		if evt.Key == "Escape" {
			return []NormalizedAction{{
				Action:    ActionOpenMenu,
				Source:    evt.Source,
				Timestamp: evt.Timestamp,
			}}
		}
	}

	return nil
}
