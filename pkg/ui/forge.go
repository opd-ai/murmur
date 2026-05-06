// Package ui - Sigil Forge submission panel implementation.
// Per ROADMAP.md line 477: "UI: Forge submission panel — create/submit entries,
// view competitors".
// Per ANONYMOUS_GAME_MECHANICS.md: "Sigil Forge events are timed creative
// challenges where Specters compete to produce the most compelling content."
//

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// All types moved to forge_types.go to eliminate duplication with forge_stub.go.

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

// Update handles input for the panel.
func (p *ForgePanel) Update() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return
	}

	// Close on Escape.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if p.mode != ForgeModeView {
			p.mode = ForgeModeView
		} else {
			p.visible = false
		}
		return
	}

	switch p.mode {
	case ForgeModeView:
		p.updateViewMode()
	case ForgeModeCreate:
		p.updateCreateMode()
	case ForgeModeSubmit:
		p.updateSubmitMode()
	case ForgeModeEntries:
		p.updateEntriesMode()
	}
}

func (p *ForgePanel) updateViewMode() {
	// N for new forge.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		p.mode = ForgeModeCreate
		p.promptText = ""
		p.selectedType = ForgeTypeSigilArt
		p.durationChoice = 0
		return
	}

	// S for submit entry (if forge is active).
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && p.forge != nil && p.forge.IsActive {
		p.mode = ForgeModeSubmit
		p.entryText = ""
		return
	}

	// E for view entries.
	if inpututil.IsKeyJustPressed(ebiten.KeyE) && p.forge != nil {
		p.mode = ForgeModeEntries
		p.selectedEntry = 0
		p.scrollOffset = 0
		return
	}
}

func (p *ForgePanel) updateCreateMode() {
	// Tab to switch forge type.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		p.selectedType = (p.selectedType + 1) % 3
	}

	// D to toggle duration.
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		p.durationChoice = (p.durationChoice + 1) % 2
	}

	// Text input for prompt.
	p.handleTextInput(&p.promptText, 256)

	// Enter to create.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if len(p.promptText) < 10 {
			p.errorMessage = "Prompt must be at least 10 characters"
			return
		}

		duration := 30 * time.Minute
		if p.durationChoice == 1 {
			duration = 60 * time.Minute
		}

		if p.onCreate != nil {
			p.onCreate(p.selectedType, p.promptText, duration)
		}
		p.mode = ForgeModeView
	}
}

func (p *ForgePanel) updateSubmitMode() {
	// Text input for entry.
	p.handleTextInput(&p.entryText, 2048)

	// Enter to submit (if content is non-empty).
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(p.entryText) > 0 {
		if p.forge != nil && p.onSubmit != nil {
			p.onSubmit(p.forge.ForgeID, p.entryText)
		}
		p.mode = ForgeModeView
	}
}

func (p *ForgePanel) updateEntriesMode() {
	if p.forge == nil || len(p.forge.Entries) == 0 {
		return
	}

	p.handleEntryNavigation()
	p.handleEntryAmplify()
}

// handleEntryNavigation processes up/down arrow keys for entry navigation.
func (p *ForgePanel) handleEntryNavigation() {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && p.selectedEntry > 0 {
		p.selectedEntry--
		if p.selectedEntry < p.scrollOffset {
			p.scrollOffset = p.selectedEntry
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && p.selectedEntry < len(p.forge.Entries)-1 {
		p.selectedEntry++
		if p.selectedEntry >= p.scrollOffset+5 {
			p.scrollOffset = p.selectedEntry - 4
		}
	}
}

// handleEntryAmplify processes the 'A' key to amplify selected entry.
func (p *ForgePanel) handleEntryAmplify() {
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		entry := &p.forge.Entries[p.selectedEntry]
		if !entry.IsOwn && p.onAmplify != nil {
			p.onAmplify(p.forge.ForgeID, entry.EntryID)
		}
	}
}

func (p *ForgePanel) handleTextInput(target *string, maxLen int) {
	// Handle backspace — delete the last Unicode rune, not the last byte,
	// so that multibyte characters (e.g. CJK, emoji) are removed correctly.
	// Per audit MEDIUM finding: (*target)[:len-1] truncates the last byte,
	// which corrupts multibyte UTF-8 characters.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && *target != "" {
		runes := []rune(*target)
		*target = string(runes[:len(runes)-1])
	}

	// Handle regular text input.
	chars := ebiten.AppendInputChars(nil)
	for _, c := range chars {
		if len(*target) < maxLen {
			*target += string(c)
		}
	}
}

// Draw renders the panel to the screen.
func (p *ForgePanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	screenW, screenH := screen.Bounds().Dx(), screen.Bounds().Dy()
	panelW, panelH := 450, 400
	panelX := (screenW - panelW) / 2
	panelY := (screenH - panelH) / 2
	padding := p.theme.Padding

	// Background.
	vector.DrawFilledRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(panelX), float32(panelY),
		float32(panelW), float32(panelH), 1.5, p.theme.PanelBorder, true)

	// Title.
	title := p.getTitle()
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, title, defaultFont, op)
	}

	// Mode-specific content.
	contentY := panelY + 40
	switch p.mode {
	case ForgeModeView:
		p.drawViewMode(screen, panelX, contentY, panelW, panelH-50, padding)
	case ForgeModeCreate:
		p.drawCreateMode(screen, panelX, contentY, panelW, panelH-50, padding)
	case ForgeModeSubmit:
		p.drawSubmitMode(screen, panelX, contentY, panelW, panelH-50, padding)
	case ForgeModeEntries:
		p.drawEntriesMode(screen, panelX, contentY, panelW, panelH-50, padding)
	}

	// Error message.
	if p.errorMessage != "" && defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+panelH-30))
		op.ColorScale.ScaleWithColor(p.theme.TextError)
		text.Draw(screen, p.errorMessage, defaultFont, op)
	}
}

func (p *ForgePanel) getTitle() string {
	switch p.mode {
	case ForgeModeCreate:
		return "Create New Forge"
	case ForgeModeSubmit:
		return "Submit Entry"
	case ForgeModeEntries:
		return "Forge Entries"
	default:
		return "Sigil Forge"
	}
}

func (p *ForgePanel) drawViewMode(screen *ebiten.Image, x, y, w, h, padding int) {
	if p.forge == nil {
		p.drawNoForgeMessage(screen, x, y, padding)
		p.drawViewHints(screen, x, y+h-60, padding)
		return
	}

	lineY := y + padding
	lineY = p.drawForgeTypeAndPrompt(screen, x, lineY, padding)
	lineY = p.drawTimeStatus(screen, x, lineY, padding)
	lineY = p.drawEntryCount(screen, x, lineY, padding)
	p.drawWinnerIfEnded(screen, x, lineY, padding)
	p.drawViewHints(screen, x, y+h-60, padding)
}

// drawNoForgeMessage renders the "No forge selected" message.
func (p *ForgePanel) drawNoForgeMessage(screen *ebiten.Image, x, y, padding int) {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+padding))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "No forge selected", defaultFont, op)
	}
}

// drawForgeTypeAndPrompt renders forge type and prompt text.
func (p *ForgePanel) drawForgeTypeAndPrompt(screen *ebiten.Image, x, lineY, padding int) int {
	if defaultFont != nil {
		typeText := fmt.Sprintf("Type: %s", forgeTypeString(p.forge.Type))
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, typeText, defaultFont, op)
	}
	lineY += 25

	if defaultFont != nil {
		promptText := "Prompt: " + truncateString(p.forge.Prompt, 50)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, promptText, defaultFont, op)
	}
	return lineY + 25
}

// drawTimeStatus renders remaining time or "Forge has ended" message.
func (p *ForgePanel) drawTimeStatus(screen *ebiten.Image, x, lineY, padding int) int {
	if p.forge.IsActive {
		return p.drawRemainingTime(screen, x, lineY, padding)
	}
	return p.drawEndedMessage(screen, x, lineY, padding)
}

// drawRemainingTime renders the countdown timer for active forges.
func (p *ForgePanel) drawRemainingTime(screen *ebiten.Image, x, lineY, padding int) int {
	remaining := time.Until(p.forge.EndTime)
	if remaining < 0 {
		remaining = 0
	}
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60

	if defaultFont != nil {
		timeText := fmt.Sprintf("Time remaining: %02d:%02d", mins, secs)
		timeColor := p.theme.TextPrimary
		if remaining < 5*time.Minute {
			timeColor = p.theme.TextError
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(timeColor)
		text.Draw(screen, timeText, defaultFont, op)
	}
	return lineY + 25
}

// drawEndedMessage renders "Forge has ended" for inactive forges.
func (p *ForgePanel) drawEndedMessage(screen *ebiten.Image, x, lineY, padding int) int {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "Forge has ended", defaultFont, op)
	}
	return lineY + 25
}

// drawEntryCount renders the number of entries.
func (p *ForgePanel) drawEntryCount(screen *ebiten.Image, x, lineY, padding int) int {
	if defaultFont != nil {
		entryText := fmt.Sprintf("Entries: %d", len(p.forge.Entries))
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, entryText, defaultFont, op)
	}
	return lineY + 35
}

// drawWinnerIfEnded renders the winner information if forge has ended.
func (p *ForgePanel) drawWinnerIfEnded(screen *ebiten.Image, x, lineY, padding int) {
	if p.forge.IsActive {
		return
	}
	for _, e := range p.forge.Entries {
		if e.IsWinner {
			if defaultFont != nil {
				winText := fmt.Sprintf("Winner: %s (%d amps)", e.SpecterName, e.Amplifications)
				op := &text.DrawOptions{}
				op.GeoM.Translate(float64(x+padding), float64(lineY))
				op.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
				text.Draw(screen, winText, defaultFont, op)
			}
			break
		}
	}
}

func (p *ForgePanel) drawViewHints(screen *ebiten.Image, x, y, padding int) {
	if defaultFont == nil {
		return
	}

	hints := "N:New Forge  E:View Entries"
	if p.forge != nil && p.forge.IsActive {
		hints = "N:New Forge  S:Submit Entry  E:View Entries"
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+padding), float64(y))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, hints, defaultFont, op)

	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(float64(x+padding), float64(y+20))
	op2.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, "Esc:Close", defaultFont, op2)
}

func (p *ForgePanel) drawCreateMode(screen *ebiten.Image, x, y, w, h, padding int) {
	lineY := y + padding
	lineY = p.drawTypeSelection(screen, x, lineY, w, padding)
	lineY = p.drawDurationSelection(screen, x, lineY, padding)
	lineY = p.drawPromptInput(screen, x, lineY, w, padding)
	p.drawCreateInstructions(screen, x, y, h, padding)
}

// drawTypeSelection renders forge type options.
func (p *ForgePanel) drawTypeSelection(screen *ebiten.Image, x, lineY, w, padding int) int {
	types := []string{"Sigil Art", "Micro-Fiction", "Remix Chain"}
	for i, typeName := range types {
		isSelected := ForgeType(i) == p.selectedType
		typeColor := p.theme.TextSecondary
		if isSelected {
			typeColor = p.theme.AccentPrimary
			vector.DrawFilledRect(screen, float32(x+padding-5), float32(lineY-2),
				float32(w-padding*2+10), 22, p.theme.Selection, true)
		}

		if defaultFont != nil {
			prefix := "  "
			if isSelected {
				prefix = "> "
			}
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(lineY))
			op.ColorScale.ScaleWithColor(typeColor)
			text.Draw(screen, prefix+typeName, defaultFont, op)
		}
		lineY += 25
	}
	return lineY + 10
}

// drawDurationSelection renders duration options.
func (p *ForgePanel) drawDurationSelection(screen *ebiten.Image, x, lineY, padding int) int {
	durations := []string{"30 minutes", "60 minutes"}
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		durText := fmt.Sprintf("Duration: %s (D to change)", durations[p.durationChoice])
		text.Draw(screen, durText, defaultFont, op)
	}
	return lineY + 30
}

// drawPromptInput renders the prompt input box.
func (p *ForgePanel) drawPromptInput(screen *ebiten.Image, x, lineY, w, padding int) int {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, "Prompt:", defaultFont, op)
	}
	lineY += 20

	vector.DrawFilledRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), 60, p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), 60, 1, p.theme.PanelBorder, true)

	p.drawPromptText(screen, x, lineY, padding)
	return lineY
}

// drawPromptText renders the prompt text inside the input box.
func (p *ForgePanel) drawPromptText(screen *ebiten.Image, x, lineY, padding int) {
	if defaultFont != nil {
		displayText := p.promptText
		if displayText == "" {
			displayText = "Enter creative prompt..."
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding+5), float64(lineY+5))
		textColor := p.theme.TextPrimary
		if p.promptText == "" {
			textColor = p.theme.TextPlaceholder
		}
		op.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, truncateString(displayText, 60), defaultFont, op)
	}
}

// drawCreateInstructions renders keyboard shortcuts at the bottom.
func (p *ForgePanel) drawCreateInstructions(screen *ebiten.Image, x, y, h, padding int) {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+h-40))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "Tab:Switch Type  D:Duration  Enter:Create  Esc:Cancel", defaultFont, op)
	}
}

func (p *ForgePanel) drawSubmitMode(screen *ebiten.Image, x, y, w, h, padding int) {
	lineY := y + padding
	lineY = p.drawForgePromptLabel(screen, x, lineY, padding)
	lineY = p.drawEntryInputArea(screen, x, lineY, w, h, padding)
	p.drawSubmitFooter(screen, x, y, w, h, padding)
}

// drawForgePromptLabel renders the forge prompt at the top.
func (p *ForgePanel) drawForgePromptLabel(screen *ebiten.Image, x, lineY, padding int) int {
	if p.forge != nil && defaultFont != nil {
		promptText := "Prompt: " + truncateString(p.forge.Prompt, 45)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, promptText, defaultFont, op)
	}
	return lineY + 30
}

// drawEntryInputArea renders the large text input box for entry submission.
func (p *ForgePanel) drawEntryInputArea(screen *ebiten.Image, x, lineY, w, h, padding int) int {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, "Your Entry:", defaultFont, op)
	}
	lineY += 20

	inputH := h - 120
	p.drawInputBox(screen, x, lineY, w, inputH, padding)
	p.drawEntryText(screen, x, lineY, padding)
	return lineY
}

// drawInputBox renders the input box background and border.
func (p *ForgePanel) drawInputBox(screen *ebiten.Image, x, lineY, w, inputH, padding int) {
	vector.DrawFilledRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), float32(inputH), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), float32(inputH), 1, p.theme.PanelBorder, true)
}

// drawEntryText renders the entry text or placeholder inside the input box.
func (p *ForgePanel) drawEntryText(screen *ebiten.Image, x, lineY, padding int) {
	if defaultFont == nil {
		return
	}

	displayText := p.entryText
	if displayText == "" {
		displayText = "Enter your creative submission..."
	}

	textColor := p.theme.TextPrimary
	if p.entryText == "" {
		textColor = p.theme.TextPlaceholder
	}

	lines := wrapText(displayText, 55)
	for i, line := range lines {
		if i >= 8 {
			break
		}
		lineOp := &text.DrawOptions{}
		lineOp.GeoM.Translate(float64(x+padding+5), float64(lineY+5+i*18))
		lineOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, line, defaultFont, lineOp)
	}
}

// drawSubmitFooter renders character count and instructions.
func (p *ForgePanel) drawSubmitFooter(screen *ebiten.Image, x, y, w, h, padding int) {
	if defaultFont == nil {
		return
	}

	countText := fmt.Sprintf("%d/2048", len(p.entryText))
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+w-padding-80), float64(y+h-40))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, countText, defaultFont, op)

	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(float64(x+padding), float64(y+h-40))
	op2.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, "Enter:Submit  Esc:Cancel", defaultFont, op2)
}

func (p *ForgePanel) drawEntriesMode(screen *ebiten.Image, x, y, w, h, padding int) {
	if p.forge == nil || len(p.forge.Entries) == 0 {
		p.drawNoEntriesMessage(screen, x, y, padding)
		return
	}

	lineY := y + padding
	rowHeight := 50
	visibleCount := (h - 80) / rowHeight

	for i := p.scrollOffset; i < len(p.forge.Entries) && i < p.scrollOffset+visibleCount; i++ {
		entry := &p.forge.Entries[i]
		isSelected := i == p.selectedEntry

		if isSelected {
			p.drawEntrySelectionHighlight(screen, x, lineY, w, rowHeight, padding)
		}

		lineY = p.drawEntryRow(screen, entry, x, lineY, w, padding, isSelected)
	}

	p.drawEntriesInstructions(screen, x, y, h, padding)
}

// drawNoEntriesMessage shows a message when there are no entries.
func (p *ForgePanel) drawNoEntriesMessage(screen *ebiten.Image, x, y, padding int) {
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+padding))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "No entries yet", defaultFont, op)
	}
}

// drawEntrySelectionHighlight renders the selection rectangle.
func (p *ForgePanel) drawEntrySelectionHighlight(screen *ebiten.Image, x, lineY, w, rowHeight, padding int) {
	vector.DrawFilledRect(screen, float32(x+padding-5), float32(lineY-2),
		float32(w-padding*2+10), float32(rowHeight-4), p.theme.Selection, true)
}

// drawEntryRow renders a single entry row with name, amplifications, and preview.
func (p *ForgePanel) drawEntryRow(screen *ebiten.Image, entry *ForgeEntryInfo, x, lineY, w, padding int, isSelected bool) int {
	if defaultFont == nil {
		return lineY + 50
	}

	nameText := buildEntryNameText(entry)
	textColor := p.theme.TextPrimary
	if isSelected {
		textColor = p.theme.AccentPrimary
	}

	p.drawEntryName(screen, x, lineY, padding, nameText, textColor)
	p.drawEntryAmplifications(screen, x, lineY, w, padding, entry.Amplifications)
	p.drawEntryPreview(screen, x, lineY, padding, entry.Preview)

	return lineY + 50
}

// buildEntryNameText constructs the entry name with status indicators.
func buildEntryNameText(entry *ForgeEntryInfo) string {
	nameText := entry.SpecterName
	if entry.IsOwn {
		nameText += " (you)"
	}
	if entry.IsWinner {
		nameText += " ★"
	}
	return nameText
}

// drawEntryName renders the entry author name.
func (p *ForgePanel) drawEntryName(screen *ebiten.Image, x, lineY, padding int, nameText string, textColor color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+padding), float64(lineY))
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, nameText, defaultFont, op)
}

// drawEntryAmplifications renders the amplification count.
func (p *ForgePanel) drawEntryAmplifications(screen *ebiten.Image, x, lineY, w, padding, amplifications int) {
	ampText := fmt.Sprintf("%d amps", amplifications)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+w-padding-80), float64(lineY))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, ampText, defaultFont, op)
}

// drawEntryPreview renders the entry content preview.
func (p *ForgePanel) drawEntryPreview(screen *ebiten.Image, x, lineY, padding int, preview string) {
	previewText := truncateString(preview, 50)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+padding+10), float64(lineY+20))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, previewText, defaultFont, op)
}

// drawEntriesInstructions shows navigation hints at the bottom.
func (p *ForgePanel) drawEntriesInstructions(screen *ebiten.Image, x, y, h, padding int) {
	if defaultFont != nil {
		hints := "↑↓:Navigate  A:Amplify  Esc:Back"
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+h-40))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, hints, defaultFont, op)
	}
}

// Helper functions.

func forgeTypeString(ft ForgeType) string {
	switch ft {
	case ForgeTypeSigilArt:
		return "Sigil Art"
	case ForgeTypeMicroFic:
		return "Micro-Fiction"
	case ForgeTypeRemixChain:
		return "Remix Chain"
	default:
		return "Unknown"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(s string, lineLen int) []string {
	if len(s) == 0 {
		return []string{s}
	}

	var lines []string
	for len(s) > lineLen {
		// Find a good break point.
		breakPoint := lineLen
		for i := lineLen; i > 0; i-- {
			if s[i] == ' ' {
				breakPoint = i
				break
			}
		}
		lines = append(lines, s[:breakPoint])
		s = s[breakPoint:]
		if len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return lines
}
