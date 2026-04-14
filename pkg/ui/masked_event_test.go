// Package ui - Masked Event panel tests.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"testing"
	"time"
)

func TestNewMaskedEventPanel(t *testing.T) {
	theme := Theme{}
	panel := NewMaskedEventPanel(theme)
	if panel == nil {
		t.Fatal("NewMaskedEventPanel returned nil")
	}
	if panel.IsVisible() {
		t.Error("Panel should not be visible initially")
	}
	if panel.Mode() != MaskedEventModeList {
		t.Errorf("Initial mode = %d, want %d", panel.Mode(), MaskedEventModeList)
	}
}

func TestMaskedEventPanel_ShowHide(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	panel.Show()
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestMaskedEventPanel_ShowLobby(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	event := &MaskedEventInfo{
		EventID: [32]byte{1, 2, 3},
		Topic:   "Test Event",
	}

	panel.ShowLobby(event)
	if !panel.IsVisible() {
		t.Error("Panel should be visible after ShowLobby()")
	}
	if panel.Mode() != MaskedEventModeLobby {
		t.Errorf("Mode = %d, want %d", panel.Mode(), MaskedEventModeLobby)
	}
	if panel.ActiveEvent() == nil {
		t.Error("ActiveEvent should be set")
	}
}

func TestMaskedEventPanel_SetEvents(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	events := []*MaskedEventInfo{
		{EventID: [32]byte{1}, Topic: "Event 1"},
		{EventID: [32]byte{2}, Topic: "Event 2"},
	}

	panel.SetEvents(events)
	if panel.EventCount() != 2 {
		t.Errorf("EventCount = %d, want 2", panel.EventCount())
	}
}

func TestMaskedEventPanel_SetWaves(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	waves := []*MaskedWaveInfo{
		{WaveID: [32]byte{1}, Pseudonym: "Anonymous 1"},
		{WaveID: [32]byte{2}, Pseudonym: "Anonymous 2"},
		{WaveID: [32]byte{3}, Pseudonym: "Anonymous 3"},
	}

	panel.SetWaves(waves)
	if panel.WaveCount() != 3 {
		t.Errorf("WaveCount = %d, want 3", panel.WaveCount())
	}
}

func TestMaskedEventPanel_SetMode(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	modes := []MaskedEventPanelMode{
		MaskedEventModeList,
		MaskedEventModeCreate,
		MaskedEventModeJoin,
		MaskedEventModeLobby,
		MaskedEventModeCompose,
	}

	for _, mode := range modes {
		panel.SetMode(mode)
		if panel.Mode() != mode {
			t.Errorf("SetMode(%d): Mode = %d, want %d", mode, panel.Mode(), mode)
		}
	}
}

func TestMaskedEventPanel_SetActiveEvent(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	if panel.ActiveEvent() != nil {
		t.Error("ActiveEvent should be nil initially")
	}

	event := &MaskedEventInfo{
		EventID: [32]byte{1},
		Topic:   "Test",
	}
	panel.SetActiveEvent(event)

	if panel.ActiveEvent() == nil {
		t.Error("ActiveEvent should not be nil")
	}
	if panel.ActiveEvent().Topic != "Test" {
		t.Errorf("Topic = %q, want %q", panel.ActiveEvent().Topic, "Test")
	}
}

func TestMaskedEventPanel_Callbacks(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	createCalled := false
	joinCalled := false
	leaveCalled := false
	postCalled := false
	amplifyCalled := false

	panel.SetOnCreate(func(topic string, duration time.Duration, maxParticipants int) {
		createCalled = true
	})
	panel.SetOnJoin(func(eventID [32]byte) {
		joinCalled = true
	})
	panel.SetOnLeave(func(eventID [32]byte) {
		leaveCalled = true
	})
	panel.SetOnPost(func(eventID [32]byte, content string) {
		postCalled = true
	})
	panel.SetOnAmplify(func(eventID, waveID [32]byte) {
		amplifyCalled = true
	})

	// Callbacks should be set but not called yet.
	if createCalled || joinCalled || leaveCalled || postCalled || amplifyCalled {
		t.Error("Callbacks should not be called just from setting them")
	}
}

func TestMaskedEventPanel_Update(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	// Update should not panic when not visible.
	err := panel.Update()
	if err != nil {
		t.Errorf("Update returned error: %v", err)
	}

	// Update should not panic when visible.
	panel.Show()
	err = panel.Update()
	if err != nil {
		t.Errorf("Update returned error: %v", err)
	}
}

func TestMaskedEventPanel_SetTheme(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})

	newTheme := Theme{} // In real test, would have different values.
	panel.SetTheme(newTheme)
	// Just verify no panic.
}

func TestMaskedEventPanel_SetError(t *testing.T) {
	panel := NewMaskedEventPanel(Theme{})
	panel.SetError("Test error")
	// Just verify no panic.
}

func TestMaskedEventStateString(t *testing.T) {
	tests := []struct {
		state MaskedEventState
		want  string
	}{
		{MaskedEventStatePending, "Pending"},
		{MaskedEventStateActive, "Active"},
		{MaskedEventStateEnded, "Ended"},
		{MaskedEventState(99), "Unknown"},
	}

	for _, tc := range tests {
		got := MaskedEventStateString(tc.state)
		if got != tc.want {
			t.Errorf("MaskedEventStateString(%d) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestMaskedEventInfo(t *testing.T) {
	now := time.Now()
	info := MaskedEventInfo{
		EventID:          [32]byte{1},
		Topic:            "Test Event",
		State:            MaskedEventStateActive,
		StartTime:        now,
		EndTime:          now.Add(1 * time.Hour),
		Duration:         1 * time.Hour,
		ParticipantCount: 5,
		MaxParticipants:  10,
		IsJoined:         true,
		MyPseudonym:      "Flickering Mask",
	}

	if info.Topic != "Test Event" {
		t.Errorf("Topic = %q, want %q", info.Topic, "Test Event")
	}
	if !info.IsJoined {
		t.Error("IsJoined should be true")
	}
}

func TestMaskedWaveInfo(t *testing.T) {
	wave := MaskedWaveInfo{
		WaveID:    [32]byte{1},
		Pseudonym: "Silent Echo",
		Content:   "Hello, anonymous world!",
		Timestamp: time.Now(),
		Amplified: 3,
		IsOwnWave: false,
	}

	if wave.Pseudonym != "Silent Echo" {
		t.Errorf("Pseudonym = %q, want %q", wave.Pseudonym, "Silent Echo")
	}
	if wave.Amplified != 3 {
		t.Errorf("Amplified = %d, want 3", wave.Amplified)
	}
}

func TestMaskedEventPanelModes(t *testing.T) {
	// Verify mode constants.
	if MaskedEventModeList != 0 {
		t.Errorf("MaskedEventModeList = %d, want 0", MaskedEventModeList)
	}
	if MaskedEventModeCreate != 1 {
		t.Errorf("MaskedEventModeCreate = %d, want 1", MaskedEventModeCreate)
	}
	if MaskedEventModeJoin != 2 {
		t.Errorf("MaskedEventModeJoin = %d, want 2", MaskedEventModeJoin)
	}
	if MaskedEventModeLobby != 3 {
		t.Errorf("MaskedEventModeLobby = %d, want 3", MaskedEventModeLobby)
	}
	if MaskedEventModeCompose != 4 {
		t.Errorf("MaskedEventModeCompose = %d, want 4", MaskedEventModeCompose)
	}
}

func TestMaskedEventStates(t *testing.T) {
	// Verify state constants.
	if MaskedEventStatePending != 0 {
		t.Errorf("MaskedEventStatePending = %d, want 0", MaskedEventStatePending)
	}
	if MaskedEventStateActive != 1 {
		t.Errorf("MaskedEventStateActive = %d, want 1", MaskedEventStateActive)
	}
	if MaskedEventStateEnded != 2 {
		t.Errorf("MaskedEventStateEnded = %d, want 2", MaskedEventStateEnded)
	}
}
