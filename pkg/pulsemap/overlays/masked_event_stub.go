// Package overlays - Masked Event Pulse Map visualization (stub for noebiten builds).
// Per ROADMAP.md line 506: "Pulse Map visualization — translucent dome with
// identical featureless dots inside".
//
//go:build test
// +build test

package overlays

import (
	"math"
	"sync"
	"time"
)

// MaskedEventState represents the lifecycle state of a Masked Event.
type MaskedEventState uint8

const (
	MaskedEventPending MaskedEventState = iota // Waiting for start time.
	MaskedEventActive                          // Event in progress.
	MaskedEventEnded                           // Event has concluded.
)

// MaskedParticipant represents a participant inside a Masked Event.
type MaskedParticipant struct {
	X, Y float64 // Position inside dome (relative to center).
}

// MaskedEventInfo contains information about a Masked Event for visualization.
type MaskedEventInfo struct {
	EventID          [32]byte            // Unique event identifier.
	Topic            string              // Event topic (shown on hover only).
	CenterX, CenterY float64             // Center position on Pulse Map.
	StartTime        time.Time           // Event start time.
	EndTime          time.Time           // Event end time.
	State            MaskedEventState    // Current event state.
	Participants     []MaskedParticipant // Anonymous participants.
}

// MaskedEventOverlay renders Masked Events on the Pulse Map (stub).
type MaskedEventOverlay struct {
	mu        sync.RWMutex
	events    map[string]*MaskedEventInfo
	domePhase map[string]float64
	dotPhase  map[string]float64
}

// NewMaskedEventOverlay creates a new Masked Event overlay renderer (stub).
func NewMaskedEventOverlay() *MaskedEventOverlay {
	return &MaskedEventOverlay{
		events:    make(map[string]*MaskedEventInfo),
		domePhase: make(map[string]float64),
		dotPhase:  make(map[string]float64),
	}
}

// AddEvent adds a Masked Event to the overlay.
func (mo *MaskedEventOverlay) AddEvent(info *MaskedEventInfo) {
	if info == nil {
		return
	}
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(info.EventID[:])
	mo.events[key] = info
	mo.domePhase[key] = 0
	mo.dotPhase[key] = 0
}

// RemoveEvent removes a Masked Event from the overlay.
func (mo *MaskedEventOverlay) RemoveEvent(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(eventID[:])
	delete(mo.events, key)
	delete(mo.domePhase, key)
	delete(mo.dotPhase, key)
}

// UpdateEvent updates event state.
func (mo *MaskedEventOverlay) UpdateEvent(info *MaskedEventInfo) {
	if info == nil {
		return
	}
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(info.EventID[:])
	mo.events[key] = info
}

// Update advances animations (stub - no-op).
func (mo *MaskedEventOverlay) Update() {
	// No-op in stub version.
}

// computeDomeRadius calculates dome radius based on participant count.
func (mo *MaskedEventOverlay) computeDomeRadius(info *MaskedEventInfo) float64 {
	base := 60.0
	perParticipant := 5.0
	return base + float64(len(info.Participants))*perParticipant
}

// GetEvent retrieves event info by ID.
func (mo *MaskedEventOverlay) GetEvent(eventID [32]byte) *MaskedEventInfo {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	return mo.events[string(eventID[:])]
}

// EventCount returns the number of events being rendered.
func (mo *MaskedEventOverlay) EventCount() int {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	return len(mo.events)
}

// Clear removes all events from the overlay.
func (mo *MaskedEventOverlay) Clear() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	mo.events = make(map[string]*MaskedEventInfo)
	mo.domePhase = make(map[string]float64)
	mo.dotPhase = make(map[string]float64)
}

// SetParticipantPositions distributes participants evenly in a circular pattern.
func (mo *MaskedEventOverlay) SetParticipantPositions(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(eventID[:])
	info, ok := mo.events[key]
	if !ok || len(info.Participants) == 0 {
		return
	}

	radius := mo.computeDomeRadius(info) * 0.5
	count := len(info.Participants)

	for i := range info.Participants {
		angle := 2 * math.Pi * float64(i) / float64(count)
		radiusJitter := radius * (0.3 + 0.7*float64((i*7+3)%10)/10)
		info.Participants[i].X = radiusJitter * math.Cos(angle)
		info.Participants[i].Y = radiusJitter * math.Sin(angle)
	}
}

// AddParticipant adds a participant to an event with auto-positioning.
func (mo *MaskedEventOverlay) AddParticipant(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(eventID[:])
	info, ok := mo.events[key]
	if !ok {
		return
	}

	radius := mo.computeDomeRadius(info) * 0.5
	angle := float64(len(info.Participants)) * 2.39996 // Golden angle.

	info.Participants = append(info.Participants, MaskedParticipant{
		X: radius * 0.6 * math.Cos(angle),
		Y: radius * 0.6 * math.Sin(angle),
	})
}

// RemoveParticipant removes a participant.
func (mo *MaskedEventOverlay) RemoveParticipant(eventID [32]byte) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(eventID[:])
	info, ok := mo.events[key]
	if !ok || len(info.Participants) == 0 {
		return
	}

	info.Participants = info.Participants[:len(info.Participants)-1]
}

// SetState updates the event state.
func (mo *MaskedEventOverlay) SetState(eventID [32]byte, state MaskedEventState) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	key := string(eventID[:])
	if info, ok := mo.events[key]; ok {
		info.State = state
	}
}
