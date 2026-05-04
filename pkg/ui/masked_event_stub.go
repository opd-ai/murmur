// Package ui - Masked Event lobby interface panel (stub for noebiten builds).
// Per ROADMAP.md line 507: "UI: Event lobby — create event, join event,
// compose Masked Waves".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
)

// MaskedEventState represents the event lifecycle state for UI display.
type MaskedEventState uint8

const (
	MaskedEventStatePending MaskedEventState = iota // Waiting for start time.
	MaskedEventStateActive                          // Event in progress.
	MaskedEventStateEnded                           // Event has concluded.
)

// MaskedEventStateString returns a human-readable string.
func MaskedEventStateString(s MaskedEventState) string {
	switch s {
	case MaskedEventStatePending:
		return "Pending"
	case MaskedEventStateActive:
		return "Active"
	case MaskedEventStateEnded:
		return "Ended"
	default:
		return "Unknown"
	}
}

// MaskedEventInfo contains event information for UI display.
type MaskedEventInfo struct {
	EventID          [32]byte         // Unique event identifier.
	Topic            string           // Event topic.
	State            MaskedEventState // Current state.
	StartTime        time.Time        // Event start time.
	EndTime          time.Time        // Event end time.
	Duration         time.Duration    // Event duration.
	ParticipantCount int              // Current participant count.
	MaxParticipants  int              // Max allowed (0 = unlimited).
	IsJoined         bool             // True if user has joined.
	MyPseudonym      string           // User's masked pseudonym if joined.
	HostResonance    int              // Host's Resonance level.
	WaveCount        int              // Waves posted in event.
}

// MaskedWaveInfo contains a Wave posted within a Masked Event.
type MaskedWaveInfo struct {
	WaveID    [32]byte  // Wave identifier.
	Pseudonym string    // Sender's masked pseudonym.
	Content   string    // Wave content.
	Timestamp time.Time // When posted.
	Amplified int       // Amplification count.
	IsOwnWave bool      // True if posted by current user.
}

// MaskedEventPanelMode represents the panel display mode.
type MaskedEventPanelMode uint8

const (
	MaskedEventModeList    MaskedEventPanelMode = iota // List available events.
	MaskedEventModeCreate                              // Create new event form.
	MaskedEventModeJoin                                // Join confirmation.
	MaskedEventModeLobby                               // Event lobby with Waves.
	MaskedEventModeCompose                             // Compose Masked Wave.
)

// MaskedEventPanel provides UI for Masked Event interaction (stub).
type MaskedEventPanel struct {
	mu sync.RWMutex

	visible     bool
	mode        MaskedEventPanelMode
	events      []*MaskedEventInfo
	waves       []*MaskedWaveInfo
	activeEvent *MaskedEventInfo
	selectedIdx int
	theme       Theme

	// Callbacks.
	onCreate  func(topic string, duration time.Duration, maxParticipants int)
	onJoin    func(eventID [32]byte)
	onLeave   func(eventID [32]byte)
	onPost    func(eventID [32]byte, content string)
	onAmplify func(eventID, waveID [32]byte)
}

// NewMaskedEventPanel creates a new Masked Event panel (stub).
func NewMaskedEventPanel(theme Theme) *MaskedEventPanel {
	return &MaskedEventPanel{
		theme:  theme,
		mode:   MaskedEventModeList,
		events: make([]*MaskedEventInfo, 0),
		waves:  make([]*MaskedWaveInfo, 0),
	}
}

// SetTheme updates the panel theme.
func (mp *MaskedEventPanel) SetTheme(theme Theme) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.theme = theme
}

// Show displays the panel.
func (mp *MaskedEventPanel) Show() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.visible = true
	mp.mode = MaskedEventModeList
}

// ShowLobby displays the event lobby for a specific event.
func (mp *MaskedEventPanel) ShowLobby(event *MaskedEventInfo) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.visible = true
	mp.mode = MaskedEventModeLobby
	mp.activeEvent = event
}

// Hide hides the panel.
func (mp *MaskedEventPanel) Hide() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.visible = false
}

// IsVisible returns true if panel is shown.
func (mp *MaskedEventPanel) IsVisible() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.visible
}

// SetEvents updates the list of available events.
func (mp *MaskedEventPanel) SetEvents(events []*MaskedEventInfo) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.events = events
}

// SetWaves updates the Waves in the active event.
func (mp *MaskedEventPanel) SetWaves(waves []*MaskedWaveInfo) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.waves = waves
}

// SetActiveEvent updates the current event info.
func (mp *MaskedEventPanel) SetActiveEvent(event *MaskedEventInfo) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.activeEvent = event
}

// SetMode sets the panel mode.
func (mp *MaskedEventPanel) SetMode(mode MaskedEventPanelMode) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.mode = mode
}

// SetError displays an error message (stub - no-op).
func (mp *MaskedEventPanel) SetError(msg string) {
	// No-op in stub.
}

// SetOnCreate sets the create event callback.
func (mp *MaskedEventPanel) SetOnCreate(cb func(topic string, duration time.Duration, maxParticipants int)) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.onCreate = cb
}

// SetOnJoin sets the join event callback.
func (mp *MaskedEventPanel) SetOnJoin(cb func(eventID [32]byte)) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.onJoin = cb
}

// SetOnLeave sets the leave event callback.
func (mp *MaskedEventPanel) SetOnLeave(cb func(eventID [32]byte)) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.onLeave = cb
}

// SetOnPost sets the post Wave callback.
func (mp *MaskedEventPanel) SetOnPost(cb func(eventID [32]byte, content string)) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.onPost = cb
}

// SetOnAmplify sets the amplify Wave callback.
func (mp *MaskedEventPanel) SetOnAmplify(cb func(eventID, waveID [32]byte)) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.onAmplify = cb
}

// Update handles input and state updates (stub - no-op).
func (mp *MaskedEventPanel) Update() error {
	return nil
}

// Mode returns the current panel mode.
func (mp *MaskedEventPanel) Mode() MaskedEventPanelMode {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.mode
}

// ActiveEvent returns the currently active event.
func (mp *MaskedEventPanel) ActiveEvent() *MaskedEventInfo {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.activeEvent
}

// EventCount returns the number of events.
func (mp *MaskedEventPanel) EventCount() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.events)
}

// WaveCount returns the number of waves.
func (mp *MaskedEventPanel) WaveCount() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.waves)
}
