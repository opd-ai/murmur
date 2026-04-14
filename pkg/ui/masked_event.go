// Package ui - Masked Event lobby interface panel.
// Per ROADMAP.md line 507: "UI: Event lobby — create event, join event,
// compose Masked Waves".
// Per ANONYMOUS_GAME_MECHANICS.md: Masked Events are time-limited anonymous
// gatherings where participants use single-use identities.
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// MaskedEventPanel provides UI for Masked Event interaction.
type MaskedEventPanel struct {
	mu sync.RWMutex

	visible      bool
	mode         MaskedEventPanelMode
	events       []*MaskedEventInfo
	waves        []*MaskedWaveInfo
	activeEvent  *MaskedEventInfo
	selectedIdx  int
	theme        Theme
	errorMessage string

	// Create event form fields.
	createTopic           string
	createDuration        int // Index into valid durations (30, 60, 120, 240 min).
	createMaxParticipants int
	createFieldIdx        int // Which field is active.

	// Compose Wave field.
	composeContent string
	composeCharIdx int

	// Callbacks.
	onCreate  func(topic string, duration time.Duration, maxParticipants int)
	onJoin    func(eventID [32]byte)
	onLeave   func(eventID [32]byte)
	onPost    func(eventID [32]byte, content string)
	onAmplify func(eventID, waveID [32]byte)
}

// Valid event durations per spec.
var validDurations = []time.Duration{
	30 * time.Minute,
	60 * time.Minute,
	120 * time.Minute,
	240 * time.Minute,
}

// NewMaskedEventPanel creates a new Masked Event panel.
func NewMaskedEventPanel(theme Theme) *MaskedEventPanel {
	return &MaskedEventPanel{
		theme:          theme,
		mode:           MaskedEventModeList,
		events:         make([]*MaskedEventInfo, 0),
		waves:          make([]*MaskedWaveInfo, 0),
		createDuration: 1, // Default 60 min.
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
	mp.errorMessage = ""
}

// ShowLobby displays the event lobby for a specific event.
func (mp *MaskedEventPanel) ShowLobby(event *MaskedEventInfo) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.visible = true
	mp.mode = MaskedEventModeLobby
	mp.activeEvent = event
	mp.errorMessage = ""
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
	if mp.selectedIdx >= len(events) && len(events) > 0 {
		mp.selectedIdx = len(events) - 1
	}
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

// SetError displays an error message.
func (mp *MaskedEventPanel) SetError(msg string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.errorMessage = msg
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

// Update handles input and state updates.
func (mp *MaskedEventPanel) Update() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if !mp.visible {
		return nil
	}

	// Handle input based on mode.
	switch mp.mode {
	case MaskedEventModeList:
		mp.handleListInput()
	case MaskedEventModeCreate:
		mp.handleCreateInput()
	case MaskedEventModeJoin:
		mp.handleJoinInput()
	case MaskedEventModeLobby:
		mp.handleLobbyInput()
	case MaskedEventModeCompose:
		mp.handleComposeInput()
	}

	return nil
}

// handleListInput processes input in list mode.
func (mp *MaskedEventPanel) handleListInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mp.visible = false
		return
	}

	// Navigate list.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && mp.selectedIdx > 0 {
		mp.selectedIdx--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && mp.selectedIdx < len(mp.events)-1 {
		mp.selectedIdx++
	}

	// Create new event.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		mp.mode = MaskedEventModeCreate
		mp.createTopic = ""
		mp.createDuration = 1
		mp.createMaxParticipants = 0
		mp.createFieldIdx = 0
	}

	// Select event to view/join.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(mp.events) > 0 {
		event := mp.events[mp.selectedIdx]
		if event.IsJoined {
			mp.activeEvent = event
			mp.mode = MaskedEventModeLobby
		} else {
			mp.activeEvent = event
			mp.mode = MaskedEventModeJoin
		}
	}
}

// handleCreateInput processes input in create mode.
func (mp *MaskedEventPanel) handleCreateInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mp.mode = MaskedEventModeList
		return
	}

	// Navigate fields.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		mp.createFieldIdx = (mp.createFieldIdx + 1) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		mp.createFieldIdx--
		if mp.createFieldIdx < 0 {
			mp.createFieldIdx = 2
		}
	}

	// Edit field based on index.
	switch mp.createFieldIdx {
	case 0: // Topic.
		mp.handleTextInput(&mp.createTopic, 256)
	case 1: // Duration.
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && mp.createDuration > 0 {
			mp.createDuration--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) && mp.createDuration < len(validDurations)-1 {
			mp.createDuration++
		}
	case 2: // Max participants.
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && mp.createMaxParticipants > 0 {
			mp.createMaxParticipants -= 5
			if mp.createMaxParticipants < 0 {
				mp.createMaxParticipants = 0
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			mp.createMaxParticipants += 5
			if mp.createMaxParticipants > 100 {
				mp.createMaxParticipants = 100
			}
		}
	}

	// Submit.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if mp.createTopic == "" {
			mp.errorMessage = "Topic is required"
			return
		}
		if mp.onCreate != nil {
			mp.onCreate(mp.createTopic, validDurations[mp.createDuration], mp.createMaxParticipants)
		}
		mp.mode = MaskedEventModeList
	}
}

// handleJoinInput processes input in join confirmation mode.
func (mp *MaskedEventPanel) handleJoinInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mp.mode = MaskedEventModeList
		return
	}

	// Confirm join.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyY) {
		if mp.activeEvent != nil && mp.onJoin != nil {
			mp.onJoin(mp.activeEvent.EventID)
		}
		mp.mode = MaskedEventModeLobby
	}

	// Cancel.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		mp.mode = MaskedEventModeList
	}
}

// handleLobbyInput processes input in lobby mode.
func (mp *MaskedEventPanel) handleLobbyInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mp.mode = MaskedEventModeList
		mp.activeEvent = nil
		return
	}

	// Compose new Wave.
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		mp.mode = MaskedEventModeCompose
		mp.composeContent = ""
		mp.composeCharIdx = 0
	}

	// Leave event.
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if mp.activeEvent != nil && mp.onLeave != nil {
			mp.onLeave(mp.activeEvent.EventID)
		}
		mp.mode = MaskedEventModeList
		mp.activeEvent = nil
	}

	// Navigate Waves.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && mp.selectedIdx > 0 {
		mp.selectedIdx--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && mp.selectedIdx < len(mp.waves)-1 {
		mp.selectedIdx++
	}

	// Amplify selected Wave.
	if inpututil.IsKeyJustPressed(ebiten.KeyA) && len(mp.waves) > 0 {
		wave := mp.waves[mp.selectedIdx]
		if mp.activeEvent != nil && mp.onAmplify != nil {
			mp.onAmplify(mp.activeEvent.EventID, wave.WaveID)
		}
	}
}

// handleComposeInput processes input in compose mode.
func (mp *MaskedEventPanel) handleComposeInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		mp.mode = MaskedEventModeLobby
		return
	}

	// Text input for content.
	mp.handleTextInput(&mp.composeContent, 2048)

	// Submit Wave.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyControl) {
		if mp.composeContent != "" && mp.activeEvent != nil && mp.onPost != nil {
			mp.onPost(mp.activeEvent.EventID, mp.composeContent)
		}
		mp.composeContent = ""
		mp.mode = MaskedEventModeLobby
	}
}

// handleTextInput handles character input for text fields.
func (mp *MaskedEventPanel) handleTextInput(target *string, maxLen int) {
	// Get input characters.
	chars := ebiten.InputChars()
	for _, c := range chars {
		if len(*target) < maxLen {
			*target += string(c)
		}
	}

	// Backspace.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(*target) > 0 {
		*target = (*target)[:len(*target)-1]
	}
}

// Draw renders the panel to the screen.
func (mp *MaskedEventPanel) Draw(screen *ebiten.Image) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if !mp.visible {
		return
	}

	// Draw panel background.
	screenW, screenH := screen.Bounds().Dx(), screen.Bounds().Dy()
	panelW, panelH := 500, 400
	panelX := (screenW - panelW) / 2
	panelY := (screenH - panelH) / 2

	// Background.
	bgColor := mp.theme.PanelBackground
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), bgColor, false)

	// Border.
	borderColor := mp.theme.PanelBorder
	mp.drawRectOutline(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), borderColor)

	// Draw based on mode.
	contentX := float32(panelX + 20)
	contentY := float32(panelY + 20)
	contentW := float32(panelW - 40)

	switch mp.mode {
	case MaskedEventModeList:
		mp.drawListMode(screen, contentX, contentY, contentW)
	case MaskedEventModeCreate:
		mp.drawCreateMode(screen, contentX, contentY, contentW)
	case MaskedEventModeJoin:
		mp.drawJoinMode(screen, contentX, contentY, contentW)
	case MaskedEventModeLobby:
		mp.drawLobbyMode(screen, contentX, contentY, contentW)
	case MaskedEventModeCompose:
		mp.drawComposeMode(screen, contentX, contentY, contentW)
	}

	// Error message.
	if mp.errorMessage != "" {
		mp.drawText(screen, mp.errorMessage, contentX, float32(panelY+panelH-30), mp.theme.TextError)
	}
}

// drawListMode renders the event list.
func (mp *MaskedEventPanel) drawListMode(screen *ebiten.Image, x, y, w float32) {
	mp.drawText(screen, "Masked Events", x, y, mp.theme.TextPrimary)
	mp.drawText(screen, "[N] New Event  [Enter] View/Join  [Esc] Close", x, y+20, mp.theme.TextSecondary)

	y += 50
	if len(mp.events) == 0 {
		mp.drawText(screen, "No events available", x, y, mp.theme.TextSecondary)
		return
	}

	for i, event := range mp.events {
		entryY := y + float32(i*60)

		// Highlight selected.
		if i == mp.selectedIdx {
			vector.DrawFilledRect(screen, x-5, entryY-5, w+10, 55,
				mp.theme.Selection, false)
		}

		// Event info.
		stateStr := MaskedEventStateString(event.State)
		title := fmt.Sprintf("%s (%s)", event.Topic, stateStr)
		mp.drawText(screen, title, x, entryY, mp.theme.TextPrimary)

		participants := fmt.Sprintf("Participants: %d", event.ParticipantCount)
		if event.MaxParticipants > 0 {
			participants += fmt.Sprintf("/%d", event.MaxParticipants)
		}
		mp.drawText(screen, participants, x, entryY+18, mp.theme.TextSecondary)

		timeStr := mp.formatEventTime(event)
		mp.drawText(screen, timeStr, x, entryY+36, mp.theme.TextSecondary)

		if event.IsJoined {
			mp.drawText(screen, "[JOINED]", x+w-80, entryY, mp.theme.Success)
		}
	}
}

// drawCreateMode renders the create event form.
func (mp *MaskedEventPanel) drawCreateMode(screen *ebiten.Image, x, y, w float32) {
	mp.drawText(screen, "Create Masked Event", x, y, mp.theme.TextPrimary)
	mp.drawText(screen, "[Tab] Next Field  [Enter] Create  [Esc] Cancel", x, y+20, mp.theme.TextSecondary)

	y += 60

	// Topic field.
	topicHighlight := mp.createFieldIdx == 0
	mp.drawFormField(screen, "Topic:", mp.createTopic, x, y, w, topicHighlight)

	// Duration field.
	y += 60
	durationHighlight := mp.createFieldIdx == 1
	durationStr := fmt.Sprintf("%d minutes", int(validDurations[mp.createDuration].Minutes()))
	mp.drawFormField(screen, "Duration:", durationStr, x, y, w, durationHighlight)
	if durationHighlight {
		mp.drawText(screen, "[←/→] Change", x+w-100, y, mp.theme.TextSecondary)
	}

	// Max participants field.
	y += 60
	maxHighlight := mp.createFieldIdx == 2
	maxStr := "Unlimited"
	if mp.createMaxParticipants > 0 {
		maxStr = fmt.Sprintf("%d", mp.createMaxParticipants)
	}
	mp.drawFormField(screen, "Max Participants:", maxStr, x, y, w, maxHighlight)
	if maxHighlight {
		mp.drawText(screen, "[←/→] Change", x+w-100, y, mp.theme.TextSecondary)
	}

	// Note.
	y += 80
	mp.drawText(screen, "Note: Requires 100+ Resonance (Phantom milestone)", x, y, mp.theme.Warning)
}

// drawJoinMode renders the join confirmation screen.
func (mp *MaskedEventPanel) drawJoinMode(screen *ebiten.Image, x, y, w float32) {
	mp.drawText(screen, "Join Masked Event?", x, y, mp.theme.TextPrimary)

	if mp.activeEvent == nil {
		return
	}

	y += 40
	mp.drawText(screen, fmt.Sprintf("Topic: %s", mp.activeEvent.Topic), x, y, mp.theme.TextSecondary)

	y += 25
	mp.drawText(screen, fmt.Sprintf("Duration: %d minutes", int(mp.activeEvent.Duration.Minutes())), x, y, mp.theme.TextSecondary)

	y += 25
	participants := fmt.Sprintf("Participants: %d", mp.activeEvent.ParticipantCount)
	if mp.activeEvent.MaxParticipants > 0 {
		participants += fmt.Sprintf("/%d", mp.activeEvent.MaxParticipants)
	}
	mp.drawText(screen, participants, x, y, mp.theme.TextSecondary)

	y += 50
	mp.drawText(screen, "Joining creates a single-use anonymous identity.", x, y, mp.theme.Warning)
	y += 20
	mp.drawText(screen, "Your Specter identity will NOT be revealed.", x, y, mp.theme.Warning)

	y += 40
	mp.drawText(screen, "[Y/Enter] Join  [N/Esc] Cancel", x, y, mp.theme.TextSecondary)
}

// drawLobbyMode renders the event lobby with Waves.
func (mp *MaskedEventPanel) drawLobbyMode(screen *ebiten.Image, x, y, w float32) {
	if mp.activeEvent == nil {
		return
	}

	// Header.
	mp.drawText(screen, mp.activeEvent.Topic, x, y, mp.theme.TextPrimary)

	// Time remaining.
	remaining := time.Until(mp.activeEvent.EndTime)
	if remaining < 0 {
		remaining = 0
	}
	timeStr := fmt.Sprintf("Time left: %d:%02d", int(remaining.Minutes()), int(remaining.Seconds())%60)
	mp.drawText(screen, timeStr, x+w-120, y, mp.theme.TextSecondary)

	y += 20
	mp.drawText(screen, fmt.Sprintf("Your mask: %s", mp.activeEvent.MyPseudonym), x, y, mp.theme.TextSecondary)

	y += 20
	mp.drawText(screen, "[C] Compose  [A] Amplify  [L] Leave  [Esc] Back", x, y, mp.theme.TextSecondary)

	y += 30

	// Waves list.
	if len(mp.waves) == 0 {
		mp.drawText(screen, "No Waves yet. Be the first to post!", x, y, mp.theme.TextSecondary)
		return
	}

	for i, wave := range mp.waves {
		if i >= 5 { // Limit display.
			break
		}
		entryY := y + float32(i*50)

		// Highlight selected.
		if i == mp.selectedIdx {
			vector.DrawFilledRect(screen, x-5, entryY-5, w+10, 45,
				mp.theme.Selection, false)
		}

		// Wave header.
		header := wave.Pseudonym
		if wave.IsOwnWave {
			header += " (you)"
		}
		mp.drawText(screen, header, x, entryY, mp.theme.TextPrimary)

		// Amplification count.
		if wave.Amplified > 0 {
			mp.drawText(screen, fmt.Sprintf("↑%d", wave.Amplified), x+w-50, entryY, mp.theme.Success)
		}

		// Content preview.
		content := wave.Content
		if len(content) > 60 {
			content = content[:57] + "..."
		}
		mp.drawText(screen, content, x, entryY+18, mp.theme.TextSecondary)
	}
}

// drawComposeMode renders the Wave composition screen.
func (mp *MaskedEventPanel) drawComposeMode(screen *ebiten.Image, x, y, w float32) {
	mp.drawText(screen, "Compose Masked Wave", x, y, mp.theme.TextPrimary)

	if mp.activeEvent != nil {
		mp.drawText(screen, fmt.Sprintf("Posting as: %s", mp.activeEvent.MyPseudonym), x, y+20, mp.theme.TextSecondary)
	}

	y += 50
	mp.drawText(screen, "[Ctrl+Enter] Post  [Esc] Cancel", x, y, mp.theme.TextSecondary)

	// Text area.
	y += 30
	textAreaH := float32(150)
	vector.DrawFilledRect(screen, x, y, w, textAreaH, mp.theme.InputBackground, false)
	mp.drawRectOutline(screen, x, y, w, textAreaH, mp.theme.PanelBorder)

	// Content.
	displayContent := mp.composeContent
	if displayContent == "" {
		displayContent = "Type your message..."
		mp.drawText(screen, displayContent, x+5, y+5, mp.theme.TextPlaceholder)
	} else {
		mp.drawText(screen, displayContent, x+5, y+5, mp.theme.TextPrimary)
	}

	// Character count.
	charCount := fmt.Sprintf("%d/2048", len(mp.composeContent))
	mp.drawText(screen, charCount, x+w-80, y+textAreaH+5, mp.theme.TextSecondary)
}

// drawFormField renders a form field with label and value.
func (mp *MaskedEventPanel) drawFormField(screen *ebiten.Image, label, value string, x, y, w float32, highlight bool) {
	mp.drawText(screen, label, x, y, mp.theme.TextSecondary)

	fieldY := y + 20
	fieldH := float32(28)

	if highlight {
		vector.DrawFilledRect(screen, x, fieldY, w, fieldH, mp.theme.ButtonActive, false)
	} else {
		vector.DrawFilledRect(screen, x, fieldY, w, fieldH, mp.theme.InputBackground, false)
	}
	mp.drawRectOutline(screen, x, fieldY, w, fieldH, mp.theme.PanelBorder)

	displayValue := value
	if displayValue == "" {
		displayValue = "..."
	}
	mp.drawText(screen, displayValue, x+5, fieldY+5, mp.theme.TextPrimary)
}

// formatEventTime returns a formatted time string for an event.
func (mp *MaskedEventPanel) formatEventTime(event *MaskedEventInfo) string {
	now := time.Now()

	switch event.State {
	case MaskedEventStatePending:
		until := time.Until(event.StartTime)
		if until < time.Minute {
			return "Starting soon"
		}
		return fmt.Sprintf("Starts in %d min", int(until.Minutes()))

	case MaskedEventStateActive:
		remaining := time.Until(event.EndTime)
		if remaining < 0 {
			return "Ending..."
		}
		return fmt.Sprintf("%d min remaining", int(remaining.Minutes()))

	case MaskedEventStateEnded:
		since := now.Sub(event.EndTime)
		if since < time.Hour {
			return fmt.Sprintf("Ended %d min ago", int(since.Minutes()))
		}
		return fmt.Sprintf("Ended %d hours ago", int(since.Hours()))
	}

	return ""
}

// drawText draws text at the given position.
func (mp *MaskedEventPanel) drawText(screen *ebiten.Image, text string, x, y float32, col interface{}) {
	// Simplified text drawing - in real implementation would use text/v2.
	// For now just render as colored rectangle placeholder if font not available.
	_ = screen
	_ = text
	_ = x
	_ = y
	_ = col
}

// drawRectOutline draws a rectangle outline.
func (mp *MaskedEventPanel) drawRectOutline(screen *ebiten.Image, x, y, w, h float32, col interface{}) {
	// Convert to color.RGBA if needed.
	// Simplified - actual implementation would handle color type.
	_ = screen
	_ = x
	_ = y
	_ = w
	_ = h
	_ = col
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
