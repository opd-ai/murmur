// Package ui - Territory overview panel stub implementation.
// Per ROADMAP.md line 447: "UI: Territory overview panel — controlled regions,
// influence scores, weekly cycle status".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
)

// TerritoryInfo contains information about a territory for display.
type TerritoryInfo struct {
	ID           string
	IsControlled bool
	IsContested  bool
	Influence    float64
	TopInfluence float64 // Highest influence in territory.
	MemberCount  int
	CentroidX    float64
	CentroidY    float64
}

// TerritoryOverviewPanel displays territory control status (stub).
type TerritoryOverviewPanel struct {
	mu sync.RWMutex

	visible      bool
	territories  []TerritoryInfo
	selectedIdx  int
	scrollOffset int
	cycleEndTime time.Time
	myInfluence  float64
	theme        Theme

	onTerritorySelect func(territoryID string)
	onNavigate        func(centroidX, centroidY float64)
}

// NewTerritoryOverviewPanel creates a new territory overview panel (stub).
func NewTerritoryOverviewPanel(theme Theme) *TerritoryOverviewPanel {
	return &TerritoryOverviewPanel{
		theme:       theme,
		selectedIdx: -1,
	}
}

// Visible returns true if the panel is shown.
func (p *TerritoryOverviewPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *TerritoryOverviewPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *TerritoryOverviewPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *TerritoryOverviewPanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetTerritories sets the list of territories to display.
func (p *TerritoryOverviewPanel) SetTerritories(territories []TerritoryInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.territories = territories
	p.scrollOffset = 0
	if p.selectedIdx >= len(territories) {
		p.selectedIdx = -1
	}
}

// SetCycleEndTime sets when the weekly cycle ends.
func (p *TerritoryOverviewPanel) SetCycleEndTime(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cycleEndTime = t
}

// SetMyInfluence sets the user's total influence.
func (p *TerritoryOverviewPanel) SetMyInfluence(influence float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.myInfluence = influence
}

// SetOnTerritorySelect sets the callback for territory selection.
func (p *TerritoryOverviewPanel) SetOnTerritorySelect(callback func(territoryID string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onTerritorySelect = callback
}

// SetOnNavigate sets the callback for navigating to a territory.
func (p *TerritoryOverviewPanel) SetOnNavigate(callback func(centroidX, centroidY float64)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onNavigate = callback
}

// Update handles input and updates panel state (stub).
func (p *TerritoryOverviewPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
}

// GetSelectedTerritory returns the currently selected territory ID.
func (p *TerritoryOverviewPanel) GetSelectedTerritory() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
		return p.territories[p.selectedIdx].ID
	}
	return ""
}

// GetTerritoryCount returns the number of territories.
func (p *TerritoryOverviewPanel) GetTerritoryCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.territories)
}

// GetSelectedIndex returns the selected index.
func (p *TerritoryOverviewPanel) GetSelectedIndex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedIdx
}

// SetSelectedIndex sets the selected index (for testing).
func (p *TerritoryOverviewPanel) SetSelectedIndex(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx >= -1 && idx < len(p.territories) {
		p.selectedIdx = idx
	}
}

// SelectTerritory selects a territory by ID and triggers the callback.
func (p *TerritoryOverviewPanel) SelectTerritory(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, t := range p.territories {
		if t.ID == id {
			p.selectedIdx = i
			if p.onTerritorySelect != nil {
				p.onTerritorySelect(id)
			}
			return true
		}
	}
	return false
}

// NavigateToSelected navigates to the selected territory.
func (p *TerritoryOverviewPanel) NavigateToSelected() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
		t := p.territories[p.selectedIdx]
		if p.onNavigate != nil {
			p.onNavigate(t.CentroidX, t.CentroidY)
		}
	}
}

// GetCycleEndTime returns the cycle end time.
func (p *TerritoryOverviewPanel) GetCycleEndTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cycleEndTime
}

// GetMyInfluence returns the user's total influence.
func (p *TerritoryOverviewPanel) GetMyInfluence() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.myInfluence
}
