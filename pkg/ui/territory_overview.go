// Package ui - Territory overview panel implementation.
// Per ROADMAP.md line 447: "UI: Territory overview panel — controlled regions,
// influence scores, weekly cycle status".
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
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// TerritoryOverviewPanel displays territory control status.
type TerritoryOverviewPanel struct {
	mu sync.RWMutex

	visible      bool
	territories  []TerritoryInfo
	selectedIdx  int
	scrollOffset int
	cycleEndTime time.Time // When the weekly cycle ends.
	myInfluence  float64   // User's total influence across all territories.
	theme        Theme

	onTerritorySelect func(territoryID string)
	onNavigate        func(centroidX, centroidY float64)
}

// NewTerritoryOverviewPanel creates a new territory overview panel.
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

// Update handles input and updates panel state.
func (p *TerritoryOverviewPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Navigation keys.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if p.selectedIdx > 0 {
			p.selectedIdx--
			p.ensureSelectedVisible()
		} else if p.selectedIdx == -1 && len(p.territories) > 0 {
			p.selectedIdx = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if p.selectedIdx < len(p.territories)-1 {
			p.selectedIdx++
			p.ensureSelectedVisible()
		} else if p.selectedIdx == -1 && len(p.territories) > 0 {
			p.selectedIdx = 0
		}
	}

	// Selection/navigation.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
			t := p.territories[p.selectedIdx]
			if p.onTerritorySelect != nil {
				p.onTerritorySelect(t.ID)
			}
		}
	}

	// Navigate to territory location.
	if inpututil.IsKeyJustPressed(ebiten.KeyG) { // Go to.
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
			t := p.territories[p.selectedIdx]
			if p.onNavigate != nil {
				p.onNavigate(t.CentroidX, t.CentroidY)
			}
		}
	}

	// Close panel.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
	}

	return true
}

// ensureSelectedVisible adjusts scroll to keep selection visible.
func (p *TerritoryOverviewPanel) ensureSelectedVisible() {
	const visibleRows = 8
	if p.selectedIdx < p.scrollOffset {
		p.scrollOffset = p.selectedIdx
	}
	if p.selectedIdx >= p.scrollOffset+visibleRows {
		p.scrollOffset = p.selectedIdx - visibleRows + 1
	}
}

// Draw renders the panel.
func (p *TerritoryOverviewPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	// Panel dimensions.
	const (
		panelWidth  = 400
		panelHeight = 400
		padding     = 16
		rowHeight   = 32
	)

	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	panelX := float32((screenW - panelWidth) / 2)
	panelY := float32((screenH - panelHeight) / 2)

	// Background.
	vector.DrawFilledRect(screen, panelX, panelY, panelWidth, panelHeight, p.theme.PanelBackground, true)
	vector.StrokeRect(screen, panelX, panelY, panelWidth, panelHeight, 2, p.theme.PanelBorder, true)

	// Title.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, "Territory Overview", defaultFont, op)
	}

	// Cycle status.
	cycleText := p.formatCycleStatus()
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+20))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, cycleText, defaultFont, op)
	}

	// My influence.
	influenceText := fmt.Sprintf("Your influence: %.1f", p.myInfluence)
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+40))
		op.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
		text.Draw(screen, influenceText, defaultFont, op)
	}

	// Territory list.
	listY := panelY + padding + 70
	const visibleRows = 8

	for i := p.scrollOffset; i < len(p.territories) && i < p.scrollOffset+visibleRows; i++ {
		t := p.territories[i]
		rowY := listY + float32(i-p.scrollOffset)*rowHeight

		// Selection highlight.
		if i == p.selectedIdx {
			vector.DrawFilledRect(screen, panelX+padding-4, rowY-2, panelWidth-padding*2+8, rowHeight-4, p.theme.Selection, true)
		}

		// Territory info.
		statusIcon := "○" // Neutral.
		statusColor := p.theme.TextSecondary
		if t.IsControlled {
			statusIcon = "◆" // Controlled.
			statusColor = p.theme.Success
		} else if t.IsContested {
			statusIcon = "◇" // Contested.
			statusColor = p.theme.Warning
		}

		if defaultFont != nil {
			// Status icon.
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(panelX+padding), float64(rowY))
			op.ColorScale.ScaleWithColor(statusColor)
			text.Draw(screen, statusIcon, defaultFont, op)

			// Territory ID.
			op2 := &text.DrawOptions{}
			op2.GeoM.Translate(float64(panelX+padding+20), float64(rowY))
			op2.ColorScale.ScaleWithColor(p.theme.TextPrimary)
			idText := t.ID
			if len(idText) > 12 {
				idText = idText[:12] + "…"
			}
			text.Draw(screen, idText, defaultFont, op2)

			// Influence.
			op3 := &text.DrawOptions{}
			op3.GeoM.Translate(float64(panelX+padding+140), float64(rowY))
			op3.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
			text.Draw(screen, fmt.Sprintf("%.1f", t.Influence), defaultFont, op3)

			// Member count.
			op4 := &text.DrawOptions{}
			op4.GeoM.Translate(float64(panelX+padding+200), float64(rowY))
			op4.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, fmt.Sprintf("%d nodes", t.MemberCount), defaultFont, op4)
		}
	}

	// Instructions.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+panelHeight-30))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "↑↓:Select  Enter:Details  G:Go to  Esc:Close", defaultFont, op)
	}
}

// formatCycleStatus formats the weekly cycle countdown.
func (p *TerritoryOverviewPanel) formatCycleStatus() string {
	remaining := time.Until(p.cycleEndTime)
	if remaining <= 0 {
		return "Cycle: Resetting..."
	}

	days := int(remaining.Hours()) / 24
	hours := int(remaining.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("Cycle ends in %dd %dh", days, hours)
	}
	return fmt.Sprintf("Cycle ends in %dh", hours)
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
