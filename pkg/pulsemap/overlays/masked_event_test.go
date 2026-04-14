// Package overlays — Masked Event overlay tests.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"
	"time"
)

func TestNewMaskedEventOverlay(t *testing.T) {
	overlay := NewMaskedEventOverlay()
	if overlay == nil {
		t.Fatal("NewMaskedEventOverlay returned nil")
	}
	if overlay.EventCount() != 0 {
		t.Errorf("New overlay should have 0 events, got %d", overlay.EventCount())
	}
}

func TestMaskedEventOverlay_AddRemoveEvent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID:   [32]byte{1, 2, 3},
		Topic:     "Test Event",
		CenterX:   100.0,
		CenterY:   200.0,
		State:     MaskedEventActive,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	// Add event.
	overlay.AddEvent(info)
	if overlay.EventCount() != 1 {
		t.Errorf("Expected 1 event, got %d", overlay.EventCount())
	}

	// Get event.
	retrieved := overlay.GetEvent(info.EventID)
	if retrieved == nil {
		t.Fatal("GetEvent returned nil")
	}
	if retrieved.Topic != "Test Event" {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, "Test Event")
	}

	// Remove event.
	overlay.RemoveEvent(info.EventID)
	if overlay.EventCount() != 0 {
		t.Errorf("Expected 0 events after removal, got %d", overlay.EventCount())
	}
}

func TestMaskedEventOverlay_UpdateEvent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID: [32]byte{1},
		Topic:   "Original",
		State:   MaskedEventPending,
	}
	overlay.AddEvent(info)

	// Update.
	updated := &MaskedEventInfo{
		EventID: info.EventID,
		Topic:   "Updated",
		State:   MaskedEventActive,
	}
	overlay.UpdateEvent(updated)

	retrieved := overlay.GetEvent(info.EventID)
	if retrieved.Topic != "Updated" {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, "Updated")
	}
	if retrieved.State != MaskedEventActive {
		t.Errorf("State = %d, want %d", retrieved.State, MaskedEventActive)
	}
}

func TestMaskedEventOverlay_AddNil(t *testing.T) {
	overlay := NewMaskedEventOverlay()
	overlay.AddEvent(nil)
	if overlay.EventCount() != 0 {
		t.Error("Adding nil should not increase event count")
	}
}

func TestMaskedEventOverlay_Update(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID: [32]byte{1},
		State:   MaskedEventActive,
	}
	overlay.AddEvent(info)

	// Update should not panic.
	overlay.Update()
	overlay.Update()
	overlay.Update()
}

func TestMaskedEventOverlay_Clear(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	// Add multiple events.
	for i := 0; i < 5; i++ {
		overlay.AddEvent(&MaskedEventInfo{
			EventID: [32]byte{byte(i)},
		})
	}

	if overlay.EventCount() != 5 {
		t.Fatalf("Expected 5 events, got %d", overlay.EventCount())
	}

	overlay.Clear()

	if overlay.EventCount() != 0 {
		t.Errorf("Expected 0 events after clear, got %d", overlay.EventCount())
	}
}

func TestMaskedEventOverlay_AddParticipant(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID:      [32]byte{1},
		Participants: make([]MaskedParticipant, 0),
	}
	overlay.AddEvent(info)

	// Add participants.
	overlay.AddParticipant(info.EventID)
	overlay.AddParticipant(info.EventID)
	overlay.AddParticipant(info.EventID)

	retrieved := overlay.GetEvent(info.EventID)
	if len(retrieved.Participants) != 3 {
		t.Errorf("Expected 3 participants, got %d", len(retrieved.Participants))
	}
}

func TestMaskedEventOverlay_RemoveParticipant(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID: [32]byte{1},
		Participants: []MaskedParticipant{
			{X: 10, Y: 20},
			{X: 30, Y: 40},
		},
	}
	overlay.AddEvent(info)

	overlay.RemoveParticipant(info.EventID)

	retrieved := overlay.GetEvent(info.EventID)
	if len(retrieved.Participants) != 1 {
		t.Errorf("Expected 1 participant, got %d", len(retrieved.Participants))
	}
}

func TestMaskedEventOverlay_RemoveParticipant_Empty(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID:      [32]byte{1},
		Participants: []MaskedParticipant{},
	}
	overlay.AddEvent(info)

	// Should not panic on empty.
	overlay.RemoveParticipant(info.EventID)
}

func TestMaskedEventOverlay_SetParticipantPositions(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID: [32]byte{1},
		Participants: []MaskedParticipant{
			{X: 0, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: 0},
		},
	}
	overlay.AddEvent(info)

	overlay.SetParticipantPositions(info.EventID)

	retrieved := overlay.GetEvent(info.EventID)
	// Positions should now be distributed.
	allZero := true
	for _, p := range retrieved.Participants {
		if p.X != 0 || p.Y != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("SetParticipantPositions should distribute participants")
	}
}

func TestMaskedEventOverlay_SetState(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	info := &MaskedEventInfo{
		EventID: [32]byte{1},
		State:   MaskedEventPending,
	}
	overlay.AddEvent(info)

	overlay.SetState(info.EventID, MaskedEventActive)

	retrieved := overlay.GetEvent(info.EventID)
	if retrieved.State != MaskedEventActive {
		t.Errorf("State = %d, want %d", retrieved.State, MaskedEventActive)
	}

	overlay.SetState(info.EventID, MaskedEventEnded)
	retrieved = overlay.GetEvent(info.EventID)
	if retrieved.State != MaskedEventEnded {
		t.Errorf("State = %d, want %d", retrieved.State, MaskedEventEnded)
	}
}

func TestMaskedEventOverlay_SetState_NonExistent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	// Should not panic.
	overlay.SetState([32]byte{99}, MaskedEventActive)
}

func TestMaskedEventOverlay_GetEvent_NonExistent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	event := overlay.GetEvent([32]byte{99})
	if event != nil {
		t.Error("GetEvent should return nil for non-existent event")
	}
}

func TestMaskedEventOverlay_AddParticipant_NonExistent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	// Should not panic.
	overlay.AddParticipant([32]byte{99})
}

func TestMaskedEventOverlay_RemoveParticipant_NonExistent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	// Should not panic.
	overlay.RemoveParticipant([32]byte{99})
}

func TestMaskedEventOverlay_SetParticipantPositions_NonExistent(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	// Should not panic.
	overlay.SetParticipantPositions([32]byte{99})
}

func TestMaskedEventOverlay_computeDomeRadius(t *testing.T) {
	overlay := NewMaskedEventOverlay()

	tests := []struct {
		participants int
		minRadius    float64
		maxRadius    float64
	}{
		{0, 55, 65},    // Base radius.
		{5, 80, 95},    // With some participants.
		{20, 150, 170}, // Many participants.
	}

	for _, tc := range tests {
		info := &MaskedEventInfo{
			Participants: make([]MaskedParticipant, tc.participants),
		}
		radius := overlay.computeDomeRadius(info)
		if radius < tc.minRadius || radius > tc.maxRadius {
			t.Errorf("computeDomeRadius(%d) = %f, want between %f and %f",
				tc.participants, radius, tc.minRadius, tc.maxRadius)
		}
	}
}

func TestMaskedEventStates(t *testing.T) {
	// Verify state constants.
	if MaskedEventPending != 0 {
		t.Errorf("MaskedEventPending = %d, want 0", MaskedEventPending)
	}
	if MaskedEventActive != 1 {
		t.Errorf("MaskedEventActive = %d, want 1", MaskedEventActive)
	}
	if MaskedEventEnded != 2 {
		t.Errorf("MaskedEventEnded = %d, want 2", MaskedEventEnded)
	}
}

func TestMaskedParticipant(t *testing.T) {
	p := MaskedParticipant{X: 10.5, Y: -20.3}
	if p.X != 10.5 {
		t.Errorf("X = %f, want 10.5", p.X)
	}
	if p.Y != -20.3 {
		t.Errorf("Y = %f, want -20.3", p.Y)
	}
}
