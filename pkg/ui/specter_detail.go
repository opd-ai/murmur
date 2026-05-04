// Package ui - Specter node detail panel with trophy display.
// Per ROADMAP.md line 577: "Trophy display on Specter node detail panel".
// Per ANONYMOUS_GAME_MECHANICS.md: Specters earn trophies through milestones,
// activity, and rare achievements.
//

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// All types moved to specter_detail_types.go to eliminate duplication with specter_detail_stub.go.

// NewSpecterDetailPanel creates a new Specter detail panel.
func NewSpecterDetailPanel(theme Theme, callbacks SpecterDetailCallbacks) *SpecterDetailPanel {
	return &SpecterDetailPanel{
		theme:          theme,
		callbacks:      callbacks,
		mode:           SpecterModeOverview,
		hoverButton:    -1,
		selectedTrophy: -1,
		trophyHover:    -1,
	}
}

// Show makes the panel visible and sets the Specter to display.
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
	p.trophyScroll = 0

	// Load trophies if callback available.
	if p.callbacks.GetTrophies != nil && info != nil {
		p.trophies = p.callbacks.GetTrophies(info.ID)
	} else {
		p.trophies = nil
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
	} else {
		p.trophies = nil
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

// Update handles input and updates panel state.
// Returns true if the panel consumed the input.
func (p *SpecterDetailPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible || p.specter == nil {
		return false
	}

	p.animTime += 1.0 / 60.0

	mx, my := ebiten.CursorPosition()

	if !p.isMouseInPanel(mx, my) {
		return p.handleOutsideClick()
	}

	if p.handleTabSelection(mx, my) {
		return true
	}

	if p.handleCloseButton(mx, my) {
		return true
	}

	return p.handleModeInput(mx, my)
}

// isMouseInPanel checks if mouse is inside panel bounds.
func (p *SpecterDetailPanel) isMouseInPanel(mx, my int) bool {
	return mx >= p.panelX && mx <= p.panelX+p.panelW &&
		my >= p.panelY && my <= p.panelY+p.panelH
}

// handleOutsideClick closes panel if user clicks outside.
func (p *SpecterDetailPanel) handleOutsideClick() bool {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p.visible = false
		if p.callbacks.OnClose != nil {
			p.callbacks.OnClose()
		}
		return true
	}
	return false
}

// handleTabSelection processes tab clicks.
func (p *SpecterDetailPanel) handleTabSelection(mx, my int) bool {
	tabY := p.panelY + 60
	tabHeight := 30
	if my >= tabY && my < tabY+tabHeight {
		tabWidth := p.panelW / 4
		tabIndex := (mx - p.panelX) / tabWidth
		if tabIndex >= 0 && tabIndex < 4 {
			p.hoverButton = tabIndex
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				p.mode = SpecterDetailMode(tabIndex)
				p.selectedTrophy = -1
				return true
			}
		}
	} else {
		p.hoverButton = -1
	}
	return false
}

// handleCloseButton processes close button click.
func (p *SpecterDetailPanel) handleCloseButton(mx, my int) bool {
	closeX := p.panelX + p.panelW - 30
	closeY := p.panelY + 10
	if mx >= closeX && mx < closeX+20 && my >= closeY && my < closeY+20 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			p.visible = false
			if p.callbacks.OnClose != nil {
				p.callbacks.OnClose()
			}
			return true
		}
	}
	return false
}

// handleModeInput dispatches to mode-specific handlers.
func (p *SpecterDetailPanel) handleModeInput(mx, my int) bool {
	switch p.mode {
	case SpecterModeTrophies:
		return p.updateTrophies(mx, my)
	case SpecterModeInteract:
		return p.updateInteract(mx, my)
	}
	return true
}

// updateTrophies handles input in trophy mode.
func (p *SpecterDetailPanel) updateTrophies(mx, my int) bool {
	// Trophy grid area.
	gridX := p.panelX + 20
	gridY := p.panelY + 110
	gridW := p.panelW - 40
	trophySize := 48
	cols := gridW / (trophySize + 10)
	if cols < 1 {
		cols = 1
	}

	// Calculate which trophy is hovered.
	relX := mx - gridX
	relY := my - gridY + p.trophyScroll

	if relX >= 0 && relX < gridW && my >= gridY {
		col := relX / (trophySize + 10)
		row := relY / (trophySize + 10)
		idx := row*cols + col

		if col < cols && idx >= 0 && idx < len(p.trophies) {
			p.trophyHover = idx

			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				p.selectedTrophy = idx
				return true
			}
		} else {
			p.trophyHover = -1
		}
	} else {
		p.trophyHover = -1
	}

	// Handle scroll.
	_, wy := ebiten.Wheel()
	if wy != 0 {
		p.trophyScroll -= int(wy * 20)
		if p.trophyScroll < 0 {
			p.trophyScroll = 0
		}
		maxScroll := (len(p.trophies)/cols+1)*(trophySize+10) - (p.panelH - 150)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.trophyScroll > maxScroll {
			p.trophyScroll = maxScroll
		}
		return true
	}

	return true
}

// updateInteract handles input in interact mode.
func (p *SpecterDetailPanel) updateInteract(mx, my int) bool {
	buttonY := p.panelY + 120
	buttonH := 40
	buttonSpacing := 50

	// Gift button.
	if my >= buttonY && my < buttonY+buttonH {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if p.callbacks.OnSendGift != nil && p.specter != nil {
				p.callbacks.OnSendGift(p.specter.ID)
			}
			return true
		}
	}

	// Waves button.
	buttonY += buttonSpacing
	if my >= buttonY && my < buttonY+buttonH {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if p.callbacks.OnViewWaves != nil && p.specter != nil {
				p.callbacks.OnViewWaves(p.specter.ID)
			}
			return true
		}
	}

	// Mark button.
	buttonY += buttonSpacing
	if my >= buttonY && my < buttonY+buttonH {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if p.callbacks.OnAddMark != nil && p.specter != nil {
				p.callbacks.OnAddMark(p.specter.ID)
			}
			return true
		}
	}

	return true
}

// Draw renders the panel.
func (p *SpecterDetailPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible || p.specter == nil {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Panel dimensions.
	p.panelW = 400
	p.panelH = 450
	p.panelX = (sw - p.panelW) / 2
	p.panelY = (sh - p.panelH) / 2

	// Draw panel background.
	p.drawPanelBackground(screen)

	// Draw header with Specter name.
	p.drawHeader(screen)

	// Draw tabs.
	p.drawTabs(screen)

	// Draw content based on mode.
	switch p.mode {
	case SpecterModeOverview:
		p.drawOverview(screen)
	case SpecterModeTrophies:
		p.drawTrophies(screen)
	case SpecterModeActivity:
		p.drawActivity(screen)
	case SpecterModeInteract:
		p.drawInteract(screen)
	}
}

// drawPanelBackground draws the panel background with border.
func (p *SpecterDetailPanel) drawPanelBackground(screen *ebiten.Image) {
	// Outer border.
	vector.DrawFilledRect(
		screen,
		float32(p.panelX-2), float32(p.panelY-2),
		float32(p.panelW+4), float32(p.panelH+4),
		p.theme.PanelBorder, true,
	)

	// Background.
	vector.DrawFilledRect(
		screen,
		float32(p.panelX), float32(p.panelY),
		float32(p.panelW), float32(p.panelH),
		p.theme.PanelBackground, true,
	)
}

// drawHeader draws the panel header with Specter name and close button.
func (p *SpecterDetailPanel) drawHeader(screen *ebiten.Image) {
	// Header background.
	headerBg := color.RGBA{
		R: p.theme.PanelBackground.R - 10,
		G: p.theme.PanelBackground.G - 10,
		B: p.theme.PanelBackground.B - 10,
		A: 255,
	}
	vector.DrawFilledRect(
		screen,
		float32(p.panelX), float32(p.panelY),
		float32(p.panelW), 55,
		headerBg, true,
	)

	// Specter pseudonym - draw as simple text placeholder.
	// Draw a colored rectangle as name placeholder.
	nameX := float32(p.panelX + 20)
	nameY := float32(p.panelY + 15)
	vector.DrawFilledRect(screen, nameX, nameY, 200, 20, p.theme.TextPrimary, true)

	// Resonance and rank indicator.
	rankColor := p.getResonanceColor(p.specter.Resonance)
	vector.DrawFilledCircle(screen, nameX+220, nameY+10, 8, rankColor, true)

	// Close button (X).
	closeX := float32(p.panelX + p.panelW - 30)
	closeY := float32(p.panelY + 15)
	vector.StrokeLine(screen, closeX, closeY, closeX+15, closeY+15, 2, p.theme.TextSecondary, true)
	vector.StrokeLine(screen, closeX+15, closeY, closeX, closeY+15, 2, p.theme.TextSecondary, true)
}

// drawTabs draws the mode selection tabs.
func (p *SpecterDetailPanel) drawTabs(screen *ebiten.Image) {
	tabY := float32(p.panelY + 60)
	tabWidth := float32(p.panelW / 4)
	tabHeight := float32(30)

	tabs := []string{"Overview", "Trophies", "Activity", "Interact"}

	for i := range tabs {
		tabX := float32(p.panelX) + float32(i)*tabWidth

		// Tab background.
		var tabBg color.RGBA
		if SpecterDetailMode(i) == p.mode {
			tabBg = p.theme.AccentPrimary
		} else if p.hoverButton == i {
			tabBg = p.theme.ButtonHover
		} else {
			tabBg = p.theme.ButtonBackground
		}

		vector.DrawFilledRect(screen, tabX+1, tabY, tabWidth-2, tabHeight, tabBg, true)

		// Tab text placeholder.
		textX := tabX + tabWidth/2 - 25
		textY := tabY + 8
		textColor := p.theme.TextPrimary
		if SpecterDetailMode(i) == p.mode {
			textColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		vector.DrawFilledRect(screen, textX, textY, 50, 12, textColor, true)
	}
}

// drawOverview draws the overview content.
func (p *SpecterDetailPanel) drawOverview(screen *ebiten.Image) {
	y := float32(p.panelY + 110)
	x := float32(p.panelX + 20)
	lineHeight := float32(28)

	// Resonance score.
	p.drawStatLine(screen, x, y, "Resonance:", fmt.Sprintf("%.1f", p.specter.Resonance))
	y += lineHeight

	// Rank.
	p.drawStatLine(screen, x, y, "Rank:", p.specter.Rank)
	y += lineHeight

	// Waves published.
	p.drawStatLine(screen, x, y, "Waves:", fmt.Sprintf("%d", p.specter.WaveCount))
	y += lineHeight

	// Gifts sent/received.
	p.drawStatLine(screen, x, y, "Gifts Sent:", fmt.Sprintf("%d", p.specter.GiftsSent))
	y += lineHeight
	p.drawStatLine(screen, x, y, "Gifts Received:", fmt.Sprintf("%d", p.specter.GiftsReceived))
	y += lineHeight

	// Game stats.
	p.drawStatLine(screen, x, y, "Puzzles Solved:", fmt.Sprintf("%d", p.specter.PuzzlesSolved))
	y += lineHeight
	p.drawStatLine(screen, x, y, "Hunts Completed:", fmt.Sprintf("%d", p.specter.HuntsCompleted))
	y += lineHeight

	// Created/last seen.
	p.drawStatLine(screen, x, y, "Created:", p.formatTime(p.specter.CreatedAt))
	y += lineHeight
	p.drawStatLine(screen, x, y, "Last Seen:", p.formatTime(p.specter.LastSeenAt))
}

// drawStatLine draws a label: value pair.
func (p *SpecterDetailPanel) drawStatLine(screen *ebiten.Image, x, y float32, label, value string) {
	// Label placeholder.
	vector.DrawFilledRect(screen, x, y, float32(len(label)*7), 14, p.theme.TextSecondary, true)
	// Value placeholder.
	vector.DrawFilledRect(screen, x+130, y, float32(len(value)*7), 14, p.theme.TextPrimary, true)
}

// drawTrophies draws the trophy grid.
func (p *SpecterDetailPanel) drawTrophies(screen *ebiten.Image) {
	gridX := float32(p.panelX + 20)
	gridY := float32(p.panelY + 110)
	gridW := p.panelW - 40
	trophySize := float32(48)
	spacing := float32(10)
	cols := int(float32(gridW) / (trophySize + spacing))
	if cols < 1 {
		cols = 1
	}

	// Trophy count header.
	unlockedCount := len(p.trophies)
	totalCount := 0
	if p.callbacks.GetTotalTrophyCount != nil {
		totalCount = p.callbacks.GetTotalTrophyCount()
	}

	// Trophy count indicator.
	countText := fmt.Sprintf("%d/%d", unlockedCount, totalCount)
	vector.DrawFilledRect(screen, gridX, gridY-25, float32(len(countText)*8), 16, p.theme.TextSecondary, true)

	// Draw trophy grid.
	for i, trophy := range p.trophies {
		col := i % cols
		row := i / cols

		tx := gridX + float32(col)*(trophySize+spacing)
		ty := gridY + float32(row)*(trophySize+spacing) - float32(p.trophyScroll)

		// Skip if off-screen.
		if ty < gridY-trophySize || ty > float32(p.panelY+p.panelH) {
			continue
		}

		p.drawTrophyCell(screen, tx, ty, trophySize, i, &trophy)
	}

	// Draw selected trophy details if any.
	if p.selectedTrophy >= 0 && p.selectedTrophy < len(p.trophies) {
		p.drawTrophyDetails(screen)
	}
}

// drawTrophyCell draws a single trophy in the grid.
func (p *SpecterDetailPanel) drawTrophyCell(screen *ebiten.Image, x, y, size float32, index int, trophy *TrophyDisplayInfo) {
	// Cell background.
	var cellBg color.RGBA
	if index == p.selectedTrophy {
		cellBg = p.theme.Selection
	} else if index == p.trophyHover {
		cellBg = p.theme.ButtonHover
	} else {
		cellBg = p.theme.InputBackground
	}

	vector.DrawFilledRect(screen, x, y, size, size, cellBg, true)

	// Trophy category color.
	var categoryColor color.RGBA
	if trophy.Def != nil {
		switch trophy.Def.Category {
		case mechanics.TrophyCategoryMilestone:
			categoryColor = color.RGBA{R: 180, G: 140, B: 220, A: 255} // Purple.
		case mechanics.TrophyCategoryActivity:
			categoryColor = color.RGBA{R: 80, G: 180, B: 160, A: 255} // Cyan.
		case mechanics.TrophyCategoryRare:
			categoryColor = color.RGBA{R: 220, G: 180, B: 80, A: 255} // Gold.
		default:
			categoryColor = p.theme.TextSecondary
		}
	} else {
		categoryColor = p.theme.TextSecondary
	}

	// Draw glyph placeholder with category color.
	glyphSize := size * 0.7
	glyphOffset := (size - glyphSize) / 2
	vector.DrawFilledRect(screen, x+glyphOffset, y+glyphOffset, glyphSize, glyphSize, categoryColor, true)

	// Animated shimmer for rare trophies.
	if trophy.Def != nil && trophy.Def.Animated {
		shimmerAlpha := uint8(80 + 40*float64(1+sin(p.animTime*3)))
		shimmerColor := color.RGBA{R: 255, G: 255, B: 255, A: shimmerAlpha}
		vector.StrokeRect(screen, x+2, y+2, size-4, size-4, 2, shimmerColor, true)
	}

	// Border.
	vector.StrokeRect(screen, x, y, size, size, 1, p.theme.PanelBorder, true)
}

// drawTrophyDetails draws the selected trophy's details.
func (p *SpecterDetailPanel) drawTrophyDetails(screen *ebiten.Image) {
	// Detail area at bottom of panel.
	detailY := float32(p.panelY + p.panelH - 100)
	detailX := float32(p.panelX + 20)

	// Background.
	vector.DrawFilledRect(
		screen,
		float32(p.panelX+10), detailY-10,
		float32(p.panelW-20), 90,
		p.theme.InputBackground, true,
	)

	// Get trophy info.
	trophy := &p.trophies[p.selectedTrophy]
	if trophy.Def == nil {
		return
	}

	// Trophy name placeholder.
	vector.DrawFilledRect(screen, detailX, detailY, float32(len(trophy.Def.Name)*8), 16, p.theme.TextPrimary, true)

	// Trophy description placeholder.
	vector.DrawFilledRect(screen, detailX, detailY+25, float32(len(trophy.Def.Description)*6), 12, p.theme.TextSecondary, true)

	// Unlock date.
	unlockText := "Unlocked: " + p.formatTime(trophy.Trophy.UnlockedAt)
	vector.DrawFilledRect(screen, detailX, detailY+50, float32(len(unlockText)*6), 12, p.theme.TextSecondary, true)

	// Resonance bonus if any.
	if trophy.Def.Bonus > 0 {
		bonusText := fmt.Sprintf("+%d Resonance", trophy.Def.Bonus)
		vector.DrawFilledRect(screen, detailX+250, detailY, float32(len(bonusText)*7), 14, p.theme.Success, true)
	}
}

// drawActivity draws the activity content.
func (p *SpecterDetailPanel) drawActivity(screen *ebiten.Image) {
	y := float32(p.panelY + 110)
	x := float32(p.panelX + 20)

	// Activity placeholder text.
	vector.DrawFilledRect(screen, x, y, 200, 14, p.theme.TextSecondary, true)
	y += 30

	// Recent activity entries would be listed here.
	// For now, show placeholder lines.
	for i := 0; i < 5; i++ {
		vector.DrawFilledRect(screen, x, y, 300, 12, p.theme.TextSecondary, true)
		y += 25
	}
}

// drawInteract draws the interaction options.
func (p *SpecterDetailPanel) drawInteract(screen *ebiten.Image) {
	y := float32(p.panelY + 120)
	x := float32(p.panelX + 20)
	buttonW := float32(p.panelW - 40)
	buttonH := float32(40)
	spacing := float32(50)

	// Don't show interaction buttons for own Specter.
	if p.specter.IsOwnSpecter {
		vector.DrawFilledRect(screen, x, y, 200, 14, p.theme.TextSecondary, true)
		return
	}

	// Send Gift button.
	p.drawButton(screen, x, y, buttonW, buttonH, "Send Phantom Gift", p.theme.AccentPrimary)
	y += spacing

	// View Waves button.
	p.drawButton(screen, x, y, buttonW, buttonH, "View Waves", p.theme.ButtonBackground)
	y += spacing

	// Add Mark button.
	p.drawButton(screen, x, y, buttonW, buttonH, "Add Specter Mark", p.theme.ButtonBackground)
}

// drawButton draws a button.
func (p *SpecterDetailPanel) drawButton(screen *ebiten.Image, x, y, w, h float32, label string, bg color.RGBA) {
	vector.DrawFilledRect(screen, x, y, w, h, bg, true)
	vector.StrokeRect(screen, x, y, w, h, 1, p.theme.PanelBorder, true)

	// Label placeholder centered.
	labelW := float32(len(label) * 7)
	labelX := x + (w-labelW)/2
	labelY := y + (h-14)/2
	vector.DrawFilledRect(screen, labelX, labelY, labelW, 14, p.theme.TextPrimary, true)
}

// getResonanceColor returns a color based on Resonance level.
func (p *SpecterDetailPanel) getResonanceColor(resonance float64) color.RGBA {
	switch {
	case resonance >= 500:
		return color.RGBA{R: 20, G: 20, B: 40, A: 255} // Abyss - deep purple-black.
	case resonance >= 200:
		return color.RGBA{R: 180, G: 50, B: 50, A: 255} // Council-Eligible - crimson.
	case resonance >= 100:
		return color.RGBA{R: 140, G: 80, B: 180, A: 255} // Phantom - purple.
	case resonance >= 75:
		return color.RGBA{R: 100, G: 100, B: 160, A: 255} // Shade-Wraith - blue-purple.
	case resonance >= 50:
		return color.RGBA{R: 80, G: 120, B: 160, A: 255} // Wraith - blue.
	case resonance >= 25:
		return color.RGBA{R: 100, G: 100, B: 120, A: 255} // Shade - gray.
	default:
		return color.RGBA{R: 60, G: 60, B: 80, A: 255} // Unranked - dark gray.
	}
}

// formatTime formats a time for display.
func (p *SpecterDetailPanel) formatTime(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.Format("2006-01-02")
}

// sin helper for animation.
func sin(x float64) float64 {
	// Simple sine approximation for shimmer effect.
	// Using Taylor series approximation.
	x = x - float64(int(x/(2*3.14159)))*(2*3.14159)
	if x > 3.14159 {
		x -= 2 * 3.14159
	}
	return x - (x*x*x)/6 + (x*x*x*x*x)/120
}

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
