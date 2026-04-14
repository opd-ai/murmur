// Package ui - Sigil Forge submission panel implementation.
// Per ROADMAP.md line 477: "UI: Forge submission panel — create/submit entries,
// view competitors".
// Per ANONYMOUS_GAME_MECHANICS.md: "Sigil Forge events are timed creative
// challenges where Specters compete to produce the most compelling content."
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
	Preview        string // Short preview of entry content.
	Amplifications int
	SubmittedAt    time.Time
	IsOwn          bool // True if this is the current user's entry.
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
	IsCreator bool // True if current user created this forge.
	Entries   []ForgeEntryInfo
}

// ForgePanelMode represents the current panel mode.
type ForgePanelMode uint8

const (
	ForgeModeView    ForgePanelMode = iota // Viewing forge details.
	ForgeModeCreate                        // Creating a new forge.
	ForgeModeSubmit                        // Submitting an entry.
	ForgeModeEntries                       // Browsing entries.
)

// ForgePanel provides UI for Sigil Forge interaction.
type ForgePanel struct {
	mu sync.RWMutex

	visible        bool
	forge          *ForgeInfo
	mode           ForgePanelMode
	selectedEntry  int
	scrollOffset   int
	entryText      string // For submission.
	promptText     string // For creation.
	selectedType   ForgeType
	durationChoice int // 0=30min, 1=60min.
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

	// Navigation.
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

	// A to amplify selected entry.
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		entry := &p.forge.Entries[p.selectedEntry]
		if !entry.IsOwn && p.onAmplify != nil {
			p.onAmplify(p.forge.ForgeID, entry.EntryID)
		}
	}
}

func (p *ForgePanel) handleTextInput(target *string, maxLen int) {
	// Handle backspace.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(*target) > 0 {
		*target = (*target)[:len(*target)-1]
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
		if defaultFont != nil {
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(y+padding))
			op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, "No forge selected", defaultFont, op)
		}
		p.drawViewHints(screen, x, y+h-60, padding)
		return
	}

	lineY := y + padding

	// Forge type and status.
	if defaultFont != nil {
		typeText := fmt.Sprintf("Type: %s", forgeTypeString(p.forge.Type))
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, typeText, defaultFont, op)
	}
	lineY += 25

	// Prompt.
	if defaultFont != nil {
		promptText := "Prompt: " + truncateString(p.forge.Prompt, 50)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, promptText, defaultFont, op)
	}
	lineY += 25

	// Time remaining.
	if p.forge.IsActive {
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
				timeColor = p.theme.TextError // Urgent.
			}
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(lineY))
			op.ColorScale.ScaleWithColor(timeColor)
			text.Draw(screen, timeText, defaultFont, op)
		}
	} else {
		if defaultFont != nil {
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(lineY))
			op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, "Forge has ended", defaultFont, op)
		}
	}
	lineY += 25

	// Entry count.
	if defaultFont != nil {
		entryText := fmt.Sprintf("Entries: %d", len(p.forge.Entries))
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, entryText, defaultFont, op)
	}
	lineY += 35

	// Winner (if forge ended).
	if !p.forge.IsActive {
		for _, e := range p.forge.Entries {
			if e.IsWinner {
				if defaultFont != nil {
					winText := fmt.Sprintf("Winner: %s (%d amps)",
						e.SpecterName, e.Amplifications)
					op := &text.DrawOptions{}
					op.GeoM.Translate(float64(x+padding), float64(lineY))
					op.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
					text.Draw(screen, winText, defaultFont, op)
				}
				break
			}
		}
	}

	p.drawViewHints(screen, x, y+h-60, padding)
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

	// Forge type selection.
	types := []string{"Sigil Art", "Micro-Fiction", "Remix Chain"}
	for i, typeName := range types {
		isSelected := ForgeType(i) == p.selectedType
		typeColor := p.theme.TextSecondary
		if isSelected {
			typeColor = p.theme.AccentPrimary
			// Draw selection indicator.
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

	lineY += 10

	// Duration selection.
	durations := []string{"30 minutes", "60 minutes"}
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		durText := fmt.Sprintf("Duration: %s (D to change)", durations[p.durationChoice])
		text.Draw(screen, durText, defaultFont, op)
	}
	lineY += 30

	// Prompt input.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, "Prompt:", defaultFont, op)
	}
	lineY += 20

	// Input box.
	vector.DrawFilledRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), 60, p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), 60, 1, p.theme.PanelBorder, true)

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

	// Instructions.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+h-40))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "Tab:Switch Type  D:Duration  Enter:Create  Esc:Cancel", defaultFont, op)
	}
}

func (p *ForgePanel) drawSubmitMode(screen *ebiten.Image, x, y, w, h, padding int) {
	lineY := y + padding

	if p.forge != nil && defaultFont != nil {
		promptText := "Prompt: " + truncateString(p.forge.Prompt, 45)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, promptText, defaultFont, op)
	}
	lineY += 30

	// Entry input area.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(lineY))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, "Your Entry:", defaultFont, op)
	}
	lineY += 20

	// Large input box.
	inputH := h - 120
	vector.DrawFilledRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), float32(inputH), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(x+padding), float32(lineY),
		float32(w-padding*2), float32(inputH), 1, p.theme.PanelBorder, true)

	if defaultFont != nil {
		displayText := p.entryText
		if displayText == "" {
			displayText = "Enter your creative submission..."
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding+5), float64(lineY+5))
		textColor := p.theme.TextPrimary
		if p.entryText == "" {
			textColor = p.theme.TextPlaceholder
		}
		op.ColorScale.ScaleWithColor(textColor)
		// Show first few lines.
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

	// Character count.
	if defaultFont != nil {
		countText := fmt.Sprintf("%d/2048", len(p.entryText))
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+w-padding-80), float64(y+h-40))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, countText, defaultFont, op)
	}

	// Instructions.
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+padding), float64(y+h-40))
		op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
		text.Draw(screen, "Enter:Submit  Esc:Cancel", defaultFont, op)
	}
}

func (p *ForgePanel) drawEntriesMode(screen *ebiten.Image, x, y, w, h, padding int) {
	if p.forge == nil || len(p.forge.Entries) == 0 {
		if defaultFont != nil {
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(y+padding))
			op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, "No entries yet", defaultFont, op)
		}
		return
	}

	lineY := y + padding
	rowHeight := 50
	visibleCount := (h - 80) / rowHeight

	for i := p.scrollOffset; i < len(p.forge.Entries) && i < p.scrollOffset+visibleCount; i++ {
		entry := &p.forge.Entries[i]
		isSelected := i == p.selectedEntry

		// Selection highlight.
		if isSelected {
			vector.DrawFilledRect(screen, float32(x+padding-5), float32(lineY-2),
				float32(w-padding*2+10), float32(rowHeight-4), p.theme.Selection, true)
		}

		// Entry info.
		if defaultFont != nil {
			// Name and amplifications.
			nameText := entry.SpecterName
			if entry.IsOwn {
				nameText += " (you)"
			}
			if entry.IsWinner {
				nameText += " ★"
			}

			textColor := p.theme.TextPrimary
			if isSelected {
				textColor = p.theme.AccentPrimary
			}

			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(x+padding), float64(lineY))
			op.ColorScale.ScaleWithColor(textColor)
			text.Draw(screen, nameText, defaultFont, op)

			// Amplifications.
			ampText := fmt.Sprintf("%d amps", entry.Amplifications)
			op2 := &text.DrawOptions{}
			op2.GeoM.Translate(float64(x+w-padding-80), float64(lineY))
			op2.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, ampText, defaultFont, op2)

			// Preview.
			preview := truncateString(entry.Preview, 50)
			op3 := &text.DrawOptions{}
			op3.GeoM.Translate(float64(x+padding+10), float64(lineY+20))
			op3.ColorScale.ScaleWithColor(p.theme.TextSecondary)
			text.Draw(screen, preview, defaultFont, op3)
		}

		lineY += rowHeight
	}

	// Instructions.
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
