// Package ui - Specter Mark placement interface panel.
// Per ROADMAP.md line 534: "UI: Mark placement panel — choose mark type,
// select target node".
// Per ANONYMOUS_GAME_MECHANICS.md: Specter Marks are visible annotations
// placed by Specters (Resonance ≥100) on any node in the network.
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// MarkPanelMode represents the panel display mode.
type MarkPanelMode uint8

const (
	MarkModeCategorySelect MarkPanelMode = iota // Select mark category.
	MarkModeTarget                              // Choose target node.
	MarkModeConfirm                             // Confirm placement.
	MarkModePlacing                             // Placing in progress.
	MarkModeSuccess                             // Mark placed successfully.
	MarkModeError                               // Error occurred.
)

// TargetInfo contains info about a potential mark target.
type TargetInfo struct {
	NodeID      string // Hex-encoded public key.
	DisplayName string // Sigil name or pseudonym.
	IsSurface   bool   // True if Surface identity.
	IsSelf      bool   // True if own identity (prevent self-marking).
	HasMark     bool   // True if already has mark from this Specter.
}

// MarkPanelCallbacks provides callbacks for mark panel actions.
type MarkPanelCallbacks struct {
	// OnPlaceMark is called when user confirms placing a mark.
	// Returns error if placement fails, nil on success.
	OnPlaceMark func(category mechanics.MarkCategory, targetID, note string) error

	// OnClose is called when user closes the panel.
	OnClose func()

	// GetMyResonance returns the user's current Specter Resonance level.
	GetMyResonance func() int

	// GetTargets returns available targets for marking.
	GetTargets func() []TargetInfo

	// GetActiveMarkCount returns how many active marks the user has.
	GetActiveMarkCount func() int
}

// MarkPanel provides the UI for placing Specter Marks.
// Per ANONYMOUS_GAME_MECHANICS.md, marks require Resonance ≥100.
type MarkPanel struct {
	mu sync.RWMutex

	// State.
	visible    bool
	mode       MarkPanelMode
	theme      Theme
	callbacks  MarkPanelCallbacks
	errorMsg   string
	successMsg string

	// Category selection.
	selectedCategory int // 0=Watcher, 1=Ally, 2=Rival

	// Target selection.
	targets        []TargetInfo
	selectedTarget int // Index into targets.
	targetScroll   int // Scroll offset for target list.

	// Note input.
	note        string
	noteMaxLen  int
	noteFocused bool
	cursorBlink bool
	lastBlinkAt time.Time

	// Dimensions (set in Draw).
	panelX, panelY int
	panelW, panelH int
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

	// Check resonance requirement.
	resonance := 0
	if p.callbacks.GetMyResonance != nil {
		resonance = p.callbacks.GetMyResonance()
	}

	if resonance < mechanics.MarkMinResonance {
		p.visible = true
		p.mode = MarkModeError
		p.errorMsg = fmt.Sprintf("Resonance %d required (you have %d)", mechanics.MarkMinResonance, resonance)
		return
	}

	// Check active mark limit.
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

	// Initialize panel state.
	p.visible = true
	p.mode = MarkModeCategorySelect
	p.selectedCategory = 0
	p.selectedTarget = 0
	p.targetScroll = 0
	p.note = ""
	p.noteFocused = false
	p.errorMsg = ""
	p.successMsg = ""

	// Load targets.
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

// Update processes input and updates panel state.
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

	// Handle input based on mode.
	switch p.mode {
	case MarkModeCategorySelect:
		p.updateCategorySelect()
	case MarkModeTarget:
		p.updateTargetSelect()
	case MarkModeConfirm:
		p.updateConfirm()
	case MarkModeSuccess, MarkModeError:
		p.updateResult()
	}

	return nil
}

// updateCategorySelect handles input in category selection mode.
func (p *MarkPanel) updateCategorySelect() {
	// Escape to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		if p.callbacks.OnClose != nil {
			go p.callbacks.OnClose()
		}
		return
	}

	// Up/Down to select category.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		if p.selectedCategory > 0 {
			p.selectedCategory--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		if p.selectedCategory < 2 {
			p.selectedCategory++
		}
	}

	// Enter to proceed.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(p.targets) == 0 {
			p.mode = MarkModeError
			p.errorMsg = "No targets available"
		} else {
			p.mode = MarkModeTarget
		}
	}
}

// updateTargetSelect handles input in target selection mode.
func (p *MarkPanel) updateTargetSelect() {
	// Escape to go back.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.mode = MarkModeCategorySelect
		return
	}

	// Up/Down to select target.
	maxVisible := 6
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		if p.selectedTarget > 0 {
			p.selectedTarget--
			if p.selectedTarget < p.targetScroll {
				p.targetScroll = p.selectedTarget
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		if p.selectedTarget < len(p.targets)-1 {
			p.selectedTarget++
			if p.selectedTarget >= p.targetScroll+maxVisible {
				p.targetScroll = p.selectedTarget - maxVisible + 1
			}
		}
	}

	// Enter to proceed.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(p.targets) > 0 {
			target := p.targets[p.selectedTarget]
			if target.IsSelf {
				p.mode = MarkModeError
				p.errorMsg = "Cannot mark yourself"
			} else if target.HasMark {
				p.mode = MarkModeError
				p.errorMsg = "Already marked this target"
			} else {
				p.mode = MarkModeConfirm
			}
		}
	}
}

// updateConfirm handles input in confirmation mode.
func (p *MarkPanel) updateConfirm() {
	// Escape to go back.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.mode = MarkModeTarget
		return
	}

	// Tab to toggle note focus.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		p.noteFocused = !p.noteFocused
	}

	// Handle note input when focused.
	if p.noteFocused {
		p.handleNoteInput()
	}

	// Enter to confirm (when not editing note).
	if !p.noteFocused && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.confirmPlacement()
	}

	// Y to confirm placement.
	if inpututil.IsKeyJustPressed(ebiten.KeyY) {
		p.confirmPlacement()
	}

	// N to cancel.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		p.mode = MarkModeTarget
	}
}

// handleNoteInput processes text input for the note field.
func (p *MarkPanel) handleNoteInput() {
	// Backspace to delete.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(p.note) > 0 {
		p.note = p.note[:len(p.note)-1]
	}

	// Get typed runes.
	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		if len(p.note) < p.noteMaxLen && r >= 32 && r < 127 {
			p.note += string(r)
		}
	}
}

// confirmPlacement initiates the mark placement.
func (p *MarkPanel) confirmPlacement() {
	if p.callbacks.OnPlaceMark == nil {
		p.mode = MarkModeError
		p.errorMsg = "Mark placement not available"
		return
	}

	if len(p.targets) == 0 || p.selectedTarget >= len(p.targets) {
		p.mode = MarkModeError
		p.errorMsg = "Invalid target"
		return
	}

	target := p.targets[p.selectedTarget]
	category := getCategoryFromIndex(p.selectedCategory)

	p.mode = MarkModePlacing

	// Place mark (async).
	go func() {
		err := p.callbacks.OnPlaceMark(category, target.NodeID, p.note)
		p.mu.Lock()
		defer p.mu.Unlock()

		if err != nil {
			p.mode = MarkModeError
			p.errorMsg = err.Error()
		} else {
			p.mode = MarkModeSuccess
			p.successMsg = fmt.Sprintf("%s mark placed on %s",
				mechanics.CategoryString(category), target.DisplayName)
		}
	}()
}

// updateResult handles input in success/error modes.
func (p *MarkPanel) updateResult() {
	// Any key to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		p.visible = false
		if p.callbacks.OnClose != nil {
			go p.callbacks.OnClose()
		}
	}
}

// Draw renders the panel.
func (p *MarkPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	// Calculate panel dimensions.
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	p.panelW = 380
	p.panelH = 340
	p.panelX = (sw - p.panelW) / 2
	p.panelY = (sh - p.panelH) / 2

	// Draw background.
	p.drawBackground(screen)

	// Draw content based on mode.
	switch p.mode {
	case MarkModeCategorySelect:
		p.drawCategorySelect(screen)
	case MarkModeTarget:
		p.drawTargetSelect(screen)
	case MarkModeConfirm:
		p.drawConfirm(screen)
	case MarkModePlacing:
		p.drawPlacing(screen)
	case MarkModeSuccess:
		p.drawSuccess(screen)
	case MarkModeError:
		p.drawError(screen)
	}
}

// drawBackground draws the panel background.
func (p *MarkPanel) drawBackground(screen *ebiten.Image) {
	// Semi-transparent overlay.
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	overlay := color.RGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(sw), float32(sh), overlay, false)

	// Panel background.
	vector.DrawFilledRect(screen,
		float32(p.panelX), float32(p.panelY),
		float32(p.panelW), float32(p.panelH),
		p.theme.PanelBackground, false)

	// Panel border.
	vector.StrokeRect(screen,
		float32(p.panelX), float32(p.panelY),
		float32(p.panelW), float32(p.panelH),
		2, p.theme.PanelBorder, false)
}

// drawCategorySelect draws the category selection UI.
func (p *MarkPanel) drawCategorySelect(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	// Title.
	p.drawText(screen, "Select Mark Category", x, y, p.theme.TextPrimary)
	y += 30

	// Instructions.
	p.drawText(screen, "↑/↓ to select, Enter to confirm", x, y, p.theme.TextSecondary)
	y += 35

	// Category options.
	categories := []struct {
		cat  mechanics.MarkCategory
		name string
		desc string
	}{
		{mechanics.MarkWatcher, "👁 Watcher", "Neutral observation"},
		{mechanics.MarkAlly, "🛡 Ally", "Positive association"},
		{mechanics.MarkRival, "⚔ Rival", "Competitive/adversarial"},
	}

	for i, cat := range categories {
		bgColor := p.theme.PanelBackground
		textColor := p.theme.TextPrimary
		if i == p.selectedCategory {
			bgColor = p.theme.Selection
			textColor = p.theme.TextPrimary
		}

		// Draw selection background.
		vector.DrawFilledRect(screen, x-5, y-2, float32(p.panelW-30), 50, bgColor, false)

		// Draw category info.
		p.drawText(screen, cat.name, x, y, textColor)
		p.drawText(screen, cat.desc, x+20, y+20, p.theme.TextSecondary)
		y += 55
	}

	// Hint at bottom.
	p.drawText(screen, "Esc to cancel", x, float32(p.panelY+p.panelH-35), p.theme.TextPlaceholder)
}

// drawTargetSelect draws the target selection UI.
func (p *MarkPanel) drawTargetSelect(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	category := getCategoryFromIndex(p.selectedCategory)

	// Title.
	title := fmt.Sprintf("Select Target (%s)", mechanics.CategoryString(category))
	p.drawText(screen, title, x, y, p.theme.TextPrimary)
	y += 30

	// Instructions.
	p.drawText(screen, "↑/↓ to select, Enter to confirm", x, y, p.theme.TextSecondary)
	y += 35

	// Target list.
	maxVisible := 5
	if len(p.targets) == 0 {
		p.drawText(screen, "No targets available", x, y, p.theme.TextPlaceholder)
	} else {
		end := p.targetScroll + maxVisible
		if end > len(p.targets) {
			end = len(p.targets)
		}

		for i := p.targetScroll; i < end; i++ {
			target := p.targets[i]
			bgColor := p.theme.PanelBackground
			textColor := p.theme.TextPrimary
			if i == p.selectedTarget {
				bgColor = p.theme.Selection
			}
			if target.IsSelf || target.HasMark {
				textColor = p.theme.TextPlaceholder
			}

			// Draw selection background.
			vector.DrawFilledRect(screen, x-5, y-2, float32(p.panelW-30), 38, bgColor, false)

			// Draw target info.
			name := target.DisplayName
			if target.IsSelf {
				name += " (self)"
			} else if target.HasMark {
				name += " (marked)"
			}
			p.drawText(screen, name, x, y, textColor)

			// Layer indicator.
			layerText := "Surface"
			if !target.IsSurface {
				layerText = "Specter"
			}
			p.drawText(screen, layerText, x+20, y+18, p.theme.TextSecondary)
			y += 42
		}

		// Scroll indicators.
		if p.targetScroll > 0 {
			p.drawText(screen, "▲ more above", x, float32(p.panelY+p.panelH-60), p.theme.TextPlaceholder)
		}
		if end < len(p.targets) {
			p.drawText(screen, "▼ more below", x+200, float32(p.panelY+p.panelH-60), p.theme.TextPlaceholder)
		}
	}

	// Hint at bottom.
	p.drawText(screen, "Esc to go back", x, float32(p.panelY+p.panelH-35), p.theme.TextPlaceholder)
}

// drawConfirm draws the confirmation UI.
func (p *MarkPanel) drawConfirm(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	// Title.
	p.drawText(screen, "Confirm Mark Placement", x, y, p.theme.TextPrimary)
	y += 35

	// Mark details.
	category := getCategoryFromIndex(p.selectedCategory)
	var target TargetInfo
	if p.selectedTarget < len(p.targets) {
		target = p.targets[p.selectedTarget]
	}

	p.drawText(screen, fmt.Sprintf("Category: %s", mechanics.CategoryString(category)), x, y, p.theme.TextSecondary)
	y += 25
	p.drawText(screen, fmt.Sprintf("Target: %s", target.DisplayName), x, y, p.theme.TextSecondary)
	y += 35

	// Note input.
	p.drawText(screen, "Note (optional):", x, y, p.theme.TextSecondary)
	y += 20

	// Note input field.
	noteBg := p.theme.PanelBackground
	if p.noteFocused {
		noteBg = p.theme.ButtonActive
	}
	vector.DrawFilledRect(screen, x, y, float32(p.panelW-40), 50, noteBg, false)
	vector.StrokeRect(screen, x, y, float32(p.panelW-40), 50, 1, p.theme.PanelBorder, false)

	noteText := p.note
	if p.noteFocused && p.cursorBlink {
		noteText += "_"
	}
	if noteText == "" && !p.noteFocused {
		p.drawText(screen, "Tab to add a note...", x+5, y+15, p.theme.TextPlaceholder)
	} else {
		p.drawText(screen, noteText, x+5, y+15, p.theme.TextPrimary)
	}
	y += 60

	// Character count.
	countText := fmt.Sprintf("%d/%d", len(p.note), p.noteMaxLen)
	p.drawText(screen, countText, float32(p.panelX+p.panelW-80), y-45, p.theme.TextPlaceholder)

	// Confirmation prompt.
	p.drawText(screen, "Place this mark?", x, y, p.theme.TextPrimary)
	y += 25
	p.drawText(screen, "Y = Yes  N = No  Tab = Edit note", x, y, p.theme.TextSecondary)

	// Hint at bottom.
	p.drawText(screen, "Esc to go back", x, float32(p.panelY+p.panelH-35), p.theme.TextPlaceholder)
}

// drawPlacing draws the "placing" progress indicator.
func (p *MarkPanel) drawPlacing(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	p.drawText(screen, "Placing Mark", x, y, p.theme.TextPrimary)
	y += 50

	p.drawText(screen, "Broadcasting to network...", x, y, p.theme.TextSecondary)
	y += 30

	// Animated dots.
	dots := "."
	ticks := int(time.Now().UnixMilli()/500) % 4
	for i := 0; i < ticks; i++ {
		dots += "."
	}
	p.drawText(screen, dots, x, y, p.theme.TextPlaceholder)
}

// drawSuccess draws the success message.
func (p *MarkPanel) drawSuccess(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	p.drawText(screen, "✓ Mark Placed", x, y, p.theme.TextPrimary)
	y += 40

	p.drawText(screen, p.successMsg, x, y, p.theme.TextSecondary)
	y += 50

	p.drawText(screen, "Press Enter to close", x, y, p.theme.TextPlaceholder)
}

// drawError draws the error message.
func (p *MarkPanel) drawError(screen *ebiten.Image) {
	x := float32(p.panelX + 20)
	y := float32(p.panelY + 20)

	p.drawText(screen, "✗ Cannot Place Mark", x, y, p.theme.TextError)
	y += 40

	p.drawText(screen, p.errorMsg, x, y, p.theme.TextSecondary)
	y += 50

	p.drawText(screen, "Press Enter to close", x, y, p.theme.TextPlaceholder)
}

// getCategoryFromIndex converts selection index to MarkCategory.
func getCategoryFromIndex(index int) mechanics.MarkCategory {
	switch index {
	case 0:
		return mechanics.MarkWatcher
	case 1:
		return mechanics.MarkAlly
	case 2:
		return mechanics.MarkRival
	default:
		return mechanics.MarkWatcher
	}
}

// drawText draws text at the specified position.
func (p *MarkPanel) drawText(dst *ebiten.Image, text string, x, y float32, col color.RGBA) {
	// Placeholder: In production, use ebiten/v2/text/v2.
	// For now, draw small rectangles to indicate text position.
	textWidth := len(text) * 7
	vector.DrawFilledRect(dst, x, y, float32(textWidth), 14, col, true)
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
func (p *MarkPanel) GetSelectedCategory() mechanics.MarkCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return getCategoryFromIndex(p.selectedCategory)
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
