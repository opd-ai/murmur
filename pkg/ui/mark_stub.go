// Package ui - Specter Mark placement interface panel (noebiten stub).
// Per ROADMAP.md line 534: "UI: Mark placement panel — choose mark type,
// select target node".
//
//go:build test
// +build test

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

// MarkPanelMode represents the panel display mode.
type MarkPanelMode uint8

const (
	MarkModeCategorySelect MarkPanelMode = iota
	MarkModeTarget
	MarkModeConfirm
	MarkModePlacing
	MarkModeSuccess
	MarkModeError
)

// TargetInfo contains info about a potential mark target.
type TargetInfo struct {
	NodeID      string
	DisplayName string
	IsSurface   bool
	IsSelf      bool
	HasMark     bool
}

// MarkPanelCallbacks provides callbacks for mark panel actions.
type MarkPanelCallbacks struct {
	OnPlaceMark        func(category marks.MarkCategory, targetID, note string) error
	OnClose            func()
	GetMyResonance     func() int
	GetTargets         func() []TargetInfo
	GetActiveMarkCount func() int
}

// MarkPanel provides the UI for placing Specter Marks.
type MarkPanel struct {
	mu sync.RWMutex

	visible          bool
	mode             MarkPanelMode
	theme            Theme
	callbacks        MarkPanelCallbacks
	errorMsg         string
	successMsg       string
	selectedCategory int
	targets          []TargetInfo
	selectedTarget   int
	targetScroll     int
	note             string
	noteMaxLen       int
	noteFocused      bool
	cursorBlink      bool
	lastBlinkAt      time.Time
	panelX, panelY   int
	panelW, panelH   int
}

// NewMarkPanel creates a new mark placement panel.
func NewMarkPanel(theme Theme, callbacks MarkPanelCallbacks) *MarkPanel {
	return &MarkPanel{
		theme:       theme,
		callbacks:   callbacks,
		mode:        MarkModeCategorySelect,
		noteMaxLen:  140,
		lastBlinkAt: time.Now(),
	}
}

// Show makes the panel visible and initializes state.
func (p *MarkPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()

	resonance := 0
	if p.callbacks.GetMyResonance != nil {
		resonance = p.callbacks.GetMyResonance()
	}

	if resonance < marks.MarkMinResonance {
		p.visible = true
		p.mode = MarkModeError
		p.errorMsg = fmt.Sprintf("Resonance %d required (you have %d)", marks.MarkMinResonance, resonance)
		return
	}

	activeMarks := 0
	if p.callbacks.GetActiveMarkCount != nil {
		activeMarks = p.callbacks.GetActiveMarkCount()
	}

	if activeMarks >= 5 {
		p.visible = true
		p.mode = MarkModeError
		p.errorMsg = "Maximum 5 active marks reached"
		return
	}

	p.visible = true
	p.mode = MarkModeCategorySelect
	p.selectedCategory = 0
	p.selectedTarget = 0
	p.targetScroll = 0
	p.note = ""
	p.noteFocused = false
	p.errorMsg = ""
	p.successMsg = ""

	if p.callbacks.GetTargets != nil {
		p.targets = p.callbacks.GetTargets()
	}
}

// Hide closes the panel.
func (p *MarkPanel) Hide() {
	p.mu.Lock()
	p.visible = false
	p.mu.Unlock()

	if p.callbacks.OnClose != nil {
		p.callbacks.OnClose()
	}
}

// IsVisible returns whether the panel is showing.
func (p *MarkPanel) IsVisible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// GetMode returns the current panel mode.
func (p *MarkPanel) GetMode() MarkPanelMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// Update is a no-op in noebiten mode.
func (p *MarkPanel) Update() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return nil
	}

	// Update cursor blink.
	if time.Since(p.lastBlinkAt) > 500*time.Millisecond {
		p.cursorBlink = !p.cursorBlink
		p.lastBlinkAt = time.Now()
	}

	return nil
}

// Draw is a no-op in noebiten mode.
func (p *MarkPanel) Draw(screen interface{}) {
	// No-op in noebiten build.
}

// SetError sets an error message and switches to error mode.
func (p *MarkPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMsg = msg
	p.mode = MarkModeError
}

// SetSuccess sets a success message and switches to success mode.
func (p *MarkPanel) SetSuccess(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.successMsg = msg
	p.mode = MarkModeSuccess
}

// GetSelectedCategory returns the currently selected mark category.
func (p *MarkPanel) GetSelectedCategory() marks.MarkCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return getCategoryFromIndexStub(p.selectedCategory)
}

// GetSelectedTarget returns the currently selected target, or nil if none.
func (p *MarkPanel) GetSelectedTarget() *TargetInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.selectedTarget >= 0 && p.selectedTarget < len(p.targets) {
		target := p.targets[p.selectedTarget]
		return &target
	}
	return nil
}

// GetNote returns the current note text.
func (p *MarkPanel) GetNote() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.note
}

// getCategoryFromIndexStub converts selection index to MarkCategory.
func getCategoryFromIndexStub(index int) marks.MarkCategory {
	switch index {
	case 0:
		return marks.MarkWatcher
	case 1:
		return marks.MarkAlly
	case 2:
		return marks.MarkRival
	default:
		return marks.MarkWatcher
	}
}

// Test helpers for noebiten builds.

// SimulateSelectCategory selects a category by index (for testing).
func (p *MarkPanel) SimulateSelectCategory(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index >= 0 && index <= 2 {
		p.selectedCategory = index
	}
}

// SimulateSelectTarget selects a target by index (for testing).
func (p *MarkPanel) SimulateSelectTarget(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index >= 0 && index < len(p.targets) {
		p.selectedTarget = index
	}
}

// SimulateSetNote sets the note text (for testing).
func (p *MarkPanel) SimulateSetNote(note string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(note) <= p.noteMaxLen {
		p.note = note
	}
}

// SimulateAdvanceMode advances to the next mode (for testing).
func (p *MarkPanel) SimulateAdvanceMode() {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.mode {
	case MarkModeCategorySelect:
		if len(p.targets) > 0 {
			p.mode = MarkModeTarget
		}
	case MarkModeTarget:
		if p.selectedTarget < len(p.targets) {
			target := p.targets[p.selectedTarget]
			if !target.IsSelf && !target.HasMark {
				p.mode = MarkModeConfirm
			}
		}
	case MarkModeConfirm:
		p.mode = MarkModePlacing
	}
}

// SimulateConfirmPlacement simulates confirming the mark placement (for testing).
func (p *MarkPanel) SimulateConfirmPlacement() error {
	p.mu.Lock()

	if p.callbacks.OnPlaceMark == nil {
		p.mode = MarkModeError
		p.errorMsg = "Mark placement not available"
		p.mu.Unlock()
		return fmt.Errorf("mark placement not available")
	}

	if len(p.targets) == 0 || p.selectedTarget >= len(p.targets) {
		p.mode = MarkModeError
		p.errorMsg = "Invalid target"
		p.mu.Unlock()
		return fmt.Errorf("invalid target")
	}

	target := p.targets[p.selectedTarget]
	category := getCategoryFromIndexStub(p.selectedCategory)
	note := p.note
	p.mode = MarkModePlacing
	p.mu.Unlock()

	err := p.callbacks.OnPlaceMark(category, target.NodeID, note)

	p.mu.Lock()
	defer p.mu.Unlock()

	if err != nil {
		p.mode = MarkModeError
		p.errorMsg = err.Error()
		return err
	}

	p.mode = MarkModeSuccess
	p.successMsg = fmt.Sprintf("%s mark placed on %s",
		marks.CategoryString(category), target.DisplayName)
	return nil
}

// SimulateClose closes the panel (for testing).
func (p *MarkPanel) SimulateClose() {
	p.Hide()
}

// GetErrorMessage returns the current error message (for testing).
func (p *MarkPanel) GetErrorMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorMsg
}

// GetSuccessMessage returns the current success message (for testing).
func (p *MarkPanel) GetSuccessMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.successMsg
}
