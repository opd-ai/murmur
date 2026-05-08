// Package ui - Territory overview panel implementation.
// Per ROADMAP.md line 447: "UI: Territory overview panel — controlled regions,
// influence scores, weekly cycle status".
//

//go:build !test
// +build !test

package ui

import (
	"image/color"
	"strconv"
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

	p.handleNavigationKeys()
	p.handleActionKeys()
	p.handleCloseKey()

	return true
}

// handleNavigationKeys processes up/down arrow keys for list navigation.
func (p *TerritoryOverviewPanel) handleNavigationKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		p.handleUpKey()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		p.handleDownKey()
	}
}

// handleUpKey moves selection up in the territory list.
func (p *TerritoryOverviewPanel) handleUpKey() {
	if p.selectedIdx > 0 {
		p.selectedIdx--
		p.ensureSelectedVisible()
	} else if p.selectedIdx == -1 && len(p.territories) > 0 {
		p.selectedIdx = 0
	}
}

// handleDownKey moves selection down in the territory list.
func (p *TerritoryOverviewPanel) handleDownKey() {
	if p.selectedIdx < len(p.territories)-1 {
		p.selectedIdx++
		p.ensureSelectedVisible()
	} else if p.selectedIdx == -1 && len(p.territories) > 0 {
		p.selectedIdx = 0
	}
}

// handleActionKeys processes Enter and G keys for territory actions.
func (p *TerritoryOverviewPanel) handleActionKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.handleSelectTerritory()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		p.handleNavigateToTerritory()
	}
}

// handleSelectTerritory triggers territory selection callback.
func (p *TerritoryOverviewPanel) handleSelectTerritory() {
	if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
		t := p.territories[p.selectedIdx]
		if p.onTerritorySelect != nil {
			p.onTerritorySelect(t.ID)
		}
	}
}

// handleNavigateToTerritory triggers navigation to territory centroid.
func (p *TerritoryOverviewPanel) handleNavigateToTerritory() {
	if p.selectedIdx >= 0 && p.selectedIdx < len(p.territories) {
		t := p.territories[p.selectedIdx]
		if p.onNavigate != nil {
			p.onNavigate(t.CentroidX, t.CentroidY)
		}
	}
}

// handleCloseKey closes the panel on Escape.
func (p *TerritoryOverviewPanel) handleCloseKey() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
	}
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

	const (
		panelWidth  = 400
		panelHeight = 400
		padding     = 16
		rowHeight   = 32
	)

	panelX, panelY := p.calculatePanelPosition(screen, panelWidth, panelHeight)

	p.drawPanelBackground(screen, panelX, panelY, panelWidth, panelHeight)
	p.drawHeader(screen, panelX, panelY, padding)
	p.drawTerritoryList(screen, panelX, panelY, panelWidth, padding, rowHeight)
	p.drawInstructions(screen, panelX, panelY, panelWidth, panelHeight, padding)
}

func (p *TerritoryOverviewPanel) calculatePanelPosition(screen *ebiten.Image, width, height int) (float32, float32) {
	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	return float32((screenW - width) / 2), float32((screenH - height) / 2)
}

func (p *TerritoryOverviewPanel) drawPanelBackground(screen *ebiten.Image, x, y float32, width, height int) {
	vector.DrawFilledRect(screen, x, y, float32(width), float32(height), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, x, y, float32(width), float32(height), 2, p.theme.PanelBorder, true)
}

func (p *TerritoryOverviewPanel) drawHeader(screen *ebiten.Image, x, y float32, padding int) {
	if defaultFont == nil {
		return
	}

	p.drawTextAt(screen, "Territory Overview", x+float32(padding), y+float32(padding), p.theme.TextPrimary)
	p.drawTextAt(screen, p.formatCycleStatus(), x+float32(padding), y+float32(padding)+20, p.theme.TextSecondary)
	p.drawTextAt(screen, "Your influence: "+formatOneDecimal(p.myInfluence), x+float32(padding), y+float32(padding)+40, p.theme.AccentPrimary)
}

func (p *TerritoryOverviewPanel) drawTerritoryList(screen *ebiten.Image, panelX, panelY float32, panelWidth, padding, rowHeight int) {
	listY := panelY + float32(padding) + 70
	const visibleRows = 8

	for i := p.scrollOffset; i < len(p.territories) && i < p.scrollOffset+visibleRows; i++ {
		t := p.territories[i]
		rowY := listY + float32(i-p.scrollOffset)*float32(rowHeight)

		p.drawTerritoryRow(screen, t, i, panelX, rowY, panelWidth, padding, rowHeight)
	}
}

func (p *TerritoryOverviewPanel) drawTerritoryRow(screen *ebiten.Image, t TerritoryInfo, idx int, panelX, rowY float32, panelWidth, padding, rowHeight int) {
	if idx == p.selectedIdx {
		vector.DrawFilledRect(screen, panelX+float32(padding)-4, rowY-2, float32(panelWidth-padding*2+8), float32(rowHeight)-4, p.theme.Selection, true)
	}

	statusIcon, statusColor := p.getTerritoryStatus(t)

	if defaultFont != nil {
		p.drawTextAt(screen, statusIcon, panelX+float32(padding), rowY, statusColor)

		idText := t.ID
		if len(idText) > 12 {
			idText = idText[:12] + "…"
		}
		p.drawTextAt(screen, idText, panelX+float32(padding)+20, rowY, p.theme.TextPrimary)
		p.drawTextAt(screen, formatOneDecimal(t.Influence), panelX+float32(padding)+140, rowY, p.theme.AccentPrimary)
		p.drawTextAt(screen, strconv.Itoa(t.MemberCount)+" nodes", panelX+float32(padding)+200, rowY, p.theme.TextSecondary)
	}
}

func (p *TerritoryOverviewPanel) getTerritoryStatus(t TerritoryInfo) (string, color.RGBA) {
	if t.IsControlled {
		return "◆", p.theme.Success
	}
	if t.IsContested {
		return "◇", p.theme.Warning
	}
	return "○", p.theme.TextSecondary
}

func (p *TerritoryOverviewPanel) drawInstructions(screen *ebiten.Image, panelX, panelY float32, panelWidth, panelHeight, padding int) {
	if defaultFont != nil {
		p.drawTextAt(screen, "↑↓:Select  Enter:Details  G:Go to  Esc:Close",
			panelX+float32(padding), panelY+float32(panelHeight)-30, p.theme.TextSecondary)
	}
}

func (p *TerritoryOverviewPanel) drawTextAt(screen *ebiten.Image, content string, x, y float32, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, content, defaultFont, op)
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
		return "Cycle ends in " + strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h"
	}
	return "Cycle ends in " + strconv.Itoa(hours) + "h"
}

func formatOneDecimal(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
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
