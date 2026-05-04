// Package ui - Sigil Forge submission panel stub for noebiten builds.
// Per ROADMAP.md line 477: "UI: Forge submission panel — create/submit entries,
// view competitors".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
)

// ForgeType represents the type of Sigil Forge event.
type ForgeType uint8

const (
	ForgeTypeSigilArt   ForgeType = iota // Sigil art creation.
	ForgeTypeMicroFic                    // Micro-fiction writing.
	ForgeTypeRemixChain                  // Collaborative remix.
)

// ForgeEntryInfo contains information about a forge entry.
type ForgeEntryInfo struct {
	EntryID        [32]byte
	SpecterKey     [32]byte
	SpecterName    string
	Preview        string
	Amplifications int
	SubmittedAt    time.Time
	IsOwn          bool
	IsWinner       bool
}

// ForgeInfo contains information about a forge event.
type ForgeInfo struct {
	ForgeID   [32]byte
	Type      ForgeType
	Prompt    string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	IsActive  bool
	IsCreator bool
	Entries   []ForgeEntryInfo
}

// ForgePanelMode represents the current panel mode.
type ForgePanelMode uint8

const (
	ForgeModeView ForgePanelMode = iota
	ForgeModeCreate
	ForgeModeSubmit
	ForgeModeEntries
)

// ForgePanel provides UI for Sigil Forge interaction.
// Stub implementation for noebiten builds.
type ForgePanel struct {
	mu sync.RWMutex

	visible        bool
	forge          *ForgeInfo
	mode           ForgePanelMode
	selectedEntry  int
	scrollOffset   int
	entryText      string
	promptText     string
	selectedType   ForgeType
	durationChoice int
	errorMessage   string
	theme          Theme

	onCreate  func(forgeType ForgeType, prompt string, duration time.Duration)
	onSubmit  func(forgeID [32]byte, content string)
	onAmplify func(forgeID, entryID [32]byte)
}

// NewForgePanel creates a new Forge panel.
func NewForgePanel(theme Theme) *ForgePanel {
	return &ForgePanel{
		theme:          theme,
		mode:           ForgeModeView,
		selectedType:   ForgeTypeSigilArt,
		durationChoice: 0,
	}
}

// Visible returns true if the panel is shown.
func (p *ForgePanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show makes the panel visible.
func (p *ForgePanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *ForgePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.errorMessage = ""
}

// Toggle toggles visibility.
func (p *ForgePanel) Toggle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = !p.visible
	if !p.visible {
		p.errorMessage = ""
	}
}

// SetForge sets the forge to display.
func (p *ForgePanel) SetForge(forge *ForgeInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.forge = forge
	p.selectedEntry = 0
	p.scrollOffset = 0
	p.mode = ForgeModeView
}

// SetMode sets the panel mode.
func (p *ForgePanel) SetMode(mode ForgePanelMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
	p.errorMessage = ""
}

// SetOnCreate sets the callback for forge creation.
func (p *ForgePanel) SetOnCreate(fn func(ForgeType, string, time.Duration)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onCreate = fn
}

// SetOnSubmit sets the callback for entry submission.
func (p *ForgePanel) SetOnSubmit(fn func(forgeID [32]byte, content string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSubmit = fn
}

// SetOnAmplify sets the callback for entry amplification.
func (p *ForgePanel) SetOnAmplify(fn func(forgeID, entryID [32]byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onAmplify = fn
}

// SetError displays an error message.
func (p *ForgePanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMessage = msg
}

// Update handles input (stub - no-op).
func (p *ForgePanel) Update() {}

// GetMode returns the current mode.
func (p *ForgePanel) GetMode() ForgePanelMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// GetForge returns the current forge.
func (p *ForgePanel) GetForge() *ForgeInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.forge
}

// GetSelectedEntry returns the selected entry index.
func (p *ForgePanel) GetSelectedEntry() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedEntry
}

// SetSelectedEntry sets the selected entry index.
func (p *ForgePanel) SetSelectedEntry(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.selectedEntry = idx
}

// GetEntryText returns the current entry text.
func (p *ForgePanel) GetEntryText() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.entryText
}

// SetEntryText sets the entry text.
func (p *ForgePanel) SetEntryText(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entryText = text
}

// GetPromptText returns the current prompt text.
func (p *ForgePanel) GetPromptText() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.promptText
}

// SetPromptText sets the prompt text.
func (p *ForgePanel) SetPromptText(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.promptText = text
}

// GetSelectedType returns the selected forge type.
func (p *ForgePanel) GetSelectedType() ForgeType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedType
}

// SetSelectedType sets the selected forge type.
func (p *ForgePanel) SetSelectedType(t ForgeType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.selectedType = t
}

// GetDurationChoice returns the duration choice.
func (p *ForgePanel) GetDurationChoice() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.durationChoice
}

// SetDurationChoice sets the duration choice.
func (p *ForgePanel) SetDurationChoice(choice int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.durationChoice = choice
}

// GetErrorMessage returns the current error message.
func (p *ForgePanel) GetErrorMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorMessage
}

// TriggerCreate invokes the onCreate callback.
func (p *ForgePanel) TriggerCreate() {
	p.mu.RLock()
	callback := p.onCreate
	forgeType := p.selectedType
	prompt := p.promptText
	choice := p.durationChoice
	p.mu.RUnlock()

	if callback != nil {
		duration := 30 * time.Minute
		if choice == 1 {
			duration = 60 * time.Minute
		}
		callback(forgeType, prompt, duration)
	}
}

// TriggerSubmit invokes the onSubmit callback.
func (p *ForgePanel) TriggerSubmit() {
	p.mu.RLock()
	callback := p.onSubmit
	forge := p.forge
	entry := p.entryText
	p.mu.RUnlock()

	if callback != nil && forge != nil {
		callback(forge.ForgeID, entry)
	}
}

// TriggerAmplify invokes the onAmplify callback.
func (p *ForgePanel) TriggerAmplify() {
	p.mu.RLock()
	callback := p.onAmplify
	forge := p.forge
	idx := p.selectedEntry
	p.mu.RUnlock()

	if callback != nil && forge != nil && idx < len(forge.Entries) {
		callback(forge.ForgeID, forge.Entries[idx].EntryID)
	}
}
