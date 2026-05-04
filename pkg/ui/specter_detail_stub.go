// Package ui - Specter node detail panel stub for builds without Ebitengine.
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// SpecterDetailMode represents the panel tab/mode.
type SpecterDetailMode uint8

const (
	SpecterModeOverview SpecterDetailMode = iota // Overview with basic info.
	SpecterModeTrophies                          // Trophy display.
	SpecterModeActivity                          // Recent activity.
	SpecterModeInteract                          // Interaction options.
)

// SpecterInfo contains information about a Specter for display.
type SpecterInfo struct {
	ID             [32]byte
	Pseudonym      string
	Resonance      float64
	Rank           string
	CreatedAt      time.Time
	LastSeenAt     time.Time
	WaveCount      int
	GiftsSent      int
	GiftsReceived  int
	PuzzlesSolved  int
	HuntsCompleted int
	IsOwnSpecter   bool
}

// TrophyDisplayInfo contains trophy info for UI display.
type TrophyDisplayInfo struct {
	Trophy   mechanics.TrophyUnlock
	Def      *mechanics.TrophyDefinition
	Glyph    *mechanics.TrophyGlyph
	Selected bool
}

// SpecterDetailCallbacks provides callbacks for panel actions.
type SpecterDetailCallbacks struct {
	OnClose             func()
	OnSendGift          func(specterID [32]byte)
	OnViewWaves         func(specterID [32]byte)
	OnAddMark           func(specterID [32]byte)
	GetTrophies         func(specterID [32]byte) []TrophyDisplayInfo
	GetTotalTrophyCount func() int
}

// SpecterDetailPanel stub for builds without Ebitengine.
type SpecterDetailPanel struct {
	mu             sync.RWMutex
	visible        bool
	mode           SpecterDetailMode
	theme          Theme
	callbacks      SpecterDetailCallbacks
	specter        *SpecterInfo
	trophies       []TrophyDisplayInfo
	selectedTrophy int
}

// NewSpecterDetailPanel creates a new stub panel.
func NewSpecterDetailPanel(theme Theme, callbacks SpecterDetailCallbacks) *SpecterDetailPanel {
	return &SpecterDetailPanel{
		theme:          theme,
		callbacks:      callbacks,
		mode:           SpecterModeOverview,
		selectedTrophy: -1,
	}
}

// Show makes the panel visible.
func (p *SpecterDetailPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// ShowForSpecter shows the panel with a specific Specter's info.
func (p *SpecterDetailPanel) ShowForSpecter(info *SpecterInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.specter = info
	p.mode = SpecterModeOverview
	p.selectedTrophy = -1
	if p.callbacks.GetTrophies != nil && info != nil {
		p.trophies = p.callbacks.GetTrophies(info.ID)
	}
}

// Hide hides the panel.
func (p *SpecterDetailPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *SpecterDetailPanel) Toggle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = !p.visible
}

// Visible returns whether the panel is visible.
func (p *SpecterDetailPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// SetSpecter updates the displayed Specter.
func (p *SpecterDetailPanel) SetSpecter(info *SpecterInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.specter = info
	if p.callbacks.GetTrophies != nil && info != nil {
		p.trophies = p.callbacks.GetTrophies(info.ID)
	}
}

// GetSpecter returns the currently displayed Specter.
func (p *SpecterDetailPanel) GetSpecter() *SpecterInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.specter
}

// SetMode sets the panel mode/tab.
func (p *SpecterDetailPanel) SetMode(mode SpecterDetailMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
}

// GetMode returns the current panel mode.
func (p *SpecterDetailPanel) GetMode() SpecterDetailMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// Update stub - returns false.
func (p *SpecterDetailPanel) Update() bool {
	return false
}

// Draw stub - no-op.
func (p *SpecterDetailPanel) Draw(screen Screen) {}

// TrophyCount returns the number of trophies displayed.
func (p *SpecterDetailPanel) TrophyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.trophies)
}

// GetSelectedTrophy returns the currently selected trophy index.
func (p *SpecterDetailPanel) GetSelectedTrophy() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedTrophy
}

// SetSelectedTrophy sets the selected trophy index.
func (p *SpecterDetailPanel) SetSelectedTrophy(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx >= -1 && idx < len(p.trophies) {
		p.selectedTrophy = idx
	}
}

// RefreshTrophies reloads trophies from callback.
func (p *SpecterDetailPanel) RefreshTrophies() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.callbacks.GetTrophies != nil && p.specter != nil {
		p.trophies = p.callbacks.GetTrophies(p.specter.ID)
	}
}
