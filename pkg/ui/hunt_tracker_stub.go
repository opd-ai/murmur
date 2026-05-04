// Package ui provides stub types for the Hunt Tracker panel.
// Per ROADMAP.md line 434: "UI: Hunt tracker overlay — fragment locations, clue display, leaderboard".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
)

// HuntInfo contains information about a hunt for display.
type HuntInfo struct {
	ID               [32]byte
	Theme            string
	ExpiresAt        time.Time
	FragmentCount    int
	ClaimedCount     int
	Fragments        []FragmentInfo
	Leaderboard      []LeaderboardEntry
	SelectedFragment int
	UserClaims       int
}

// FragmentInfo contains information about a hunt fragment.
type FragmentInfo struct {
	Index        int
	Claimed      bool
	ClaimedByMe  bool
	Clues        []string
	LocationHint string
}

// LeaderboardEntry represents a participant in the leaderboard.
type LeaderboardEntry struct {
	Pseudonym string
	Claims    int
	IsMe      bool
}

// HuntTrackerPanel provides a UI overlay for tracking Specter Hunts (stub).
type HuntTrackerPanel struct {
	mu sync.RWMutex

	visible          bool
	hunt             *HuntInfo
	selectedTab      int
	scrollOffset     int
	errorMessage     string
	onFragmentSelect func(huntID [32]byte, fragmentIndex int)
	onClaimAttempt   func(huntID [32]byte, fragmentIndex int)
	theme            Theme
}

// NewHuntTrackerPanel creates a new hunt tracker panel (stub).
func NewHuntTrackerPanel(theme Theme) *HuntTrackerPanel {
	return &HuntTrackerPanel{
		theme:       theme,
		selectedTab: 0,
	}
}

// Visible returns true if the panel is shown.
func (p *HuntTrackerPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *HuntTrackerPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *HuntTrackerPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *HuntTrackerPanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetHunt sets the current hunt to track.
func (p *HuntTrackerPanel) SetHunt(hunt *HuntInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hunt = hunt
	p.scrollOffset = 0
}

// SetOnFragmentSelect sets the callback for fragment selection.
func (p *HuntTrackerPanel) SetOnFragmentSelect(callback func(huntID [32]byte, fragmentIndex int)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onFragmentSelect = callback
}

// SetOnClaimAttempt sets the callback for claim attempts.
func (p *HuntTrackerPanel) SetOnClaimAttempt(callback func(huntID [32]byte, fragmentIndex int)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onClaimAttempt = callback
}

// Update handles input and updates panel state (stub).
func (p *HuntTrackerPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
}

// GetHunt returns the current hunt info.
func (p *HuntTrackerPanel) GetHunt() *HuntInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hunt
}

// GetSelectedTab returns the selected tab index.
func (p *HuntTrackerPanel) GetSelectedTab() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedTab
}

// SetSelectedTab sets the selected tab index.
func (p *HuntTrackerPanel) SetSelectedTab(tab int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if tab >= 0 && tab <= 2 {
		p.selectedTab = tab
		p.scrollOffset = 0
	}
}

// GetSelectedFragment returns the selected fragment index from the current hunt.
func (p *HuntTrackerPanel) GetSelectedFragment() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.hunt == nil {
		return -1
	}
	return p.hunt.SelectedFragment
}

// SetSelectedFragment sets the selected fragment index.
func (p *HuntTrackerPanel) SetSelectedFragment(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hunt == nil {
		return
	}
	if idx >= -1 && idx < len(p.hunt.Fragments) {
		p.hunt.SelectedFragment = idx
	}
}

// SetError sets the error message.
func (p *HuntTrackerPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMessage = msg
}

// SelectFragment selects a fragment by index (for testing).
func (p *HuntTrackerPanel) SelectFragment(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hunt == nil {
		return
	}
	if idx >= 0 && idx < len(p.hunt.Fragments) {
		p.hunt.SelectedFragment = idx
		if p.onFragmentSelect != nil {
			p.onFragmentSelect(p.hunt.ID, idx)
		}
	}
}

// AttemptClaim attempts to claim the selected fragment (for testing).
func (p *HuntTrackerPanel) AttemptClaim() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hunt == nil || p.hunt.SelectedFragment < 0 {
		return
	}
	if p.onClaimAttempt != nil {
		p.onClaimAttempt(p.hunt.ID, p.hunt.SelectedFragment)
	}
}
