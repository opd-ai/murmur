// Package ui provides the Compose panel for creating Waves.
//
//go:build !test
// +build !test

package ui

import (
	"image/color"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MaxWaveLength is the maximum content length for a Wave (2048 bytes per WAVES.md).
const MaxWaveLength = 2048

// ComposePanel provides a UI for composing and submitting Waves.
// Per ROADMAP.md, this is the primary content creation interface.
type ComposePanel struct {
	mu sync.RWMutex

	// Visibility and position
	visible  bool
	x, y     int
	width    int
	height   int
	position PanelPosition

	// Input state
	content      string
	cursorPos    int
	targetNodeID string
	waveType     uint8

	// Callbacks
	onSubmit WaveSubmitCallback

	// Styling
	theme Theme

	// Animation
	anim PanelAnimation

	// Screen dimensions (updated each frame)
	screenWidth, screenHeight int
}

// NewComposePanel creates a new Wave composition panel.
func NewComposePanel(theme Theme, onSubmit WaveSubmitCallback) *ComposePanel {
	return &ComposePanel{
		theme:    theme,
		onSubmit: onSubmit,
		width:    400,
		height:   280,
		position: PositionBottomRight,
		waveType: 0x01, // Surface Wave default
	}
}

// Visible returns true if the panel is shown.
func (p *ComposePanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with animation.
func (p *ComposePanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.anim.ResetAnimation()
}

// Hide hides the panel.
func (p *ComposePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.content = ""
	p.cursorPos = 0
	p.anim.SetError("")
}

// Toggle toggles panel visibility.
func (p *ComposePanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetTargetNode sets the node to send the Wave to (for direct messages).
func (p *ComposePanel) SetTargetNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targetNodeID = nodeID
}

// SetWaveType sets the Wave type to create.
func (p *ComposePanel) SetWaveType(waveType uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waveType = waveType
}

// Update handles input and updates panel state.
// Returns true if the panel consumed the input.
func (p *ComposePanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Update common animation/error handling.
	p.anim.UpdateAnimation()
	p.refreshPositionForInput()

	// Handle mouse button clicks on Submit/Cancel before text input
	// so that clicking the button does not also insert a character.
	if p.handleMouseClick() {
		return true
	}

	// Handle text input.
	p.handleTextInput()

	// Handle special keys.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		// Shift+Enter for newline, Enter alone to submit.
		if !ebiten.IsKeyPressed(ebiten.KeyShift) {
			p.submit()
			return true
		}
		p.insertChar('\n')
	}

	return true // Panel consumes all input when visible.
}

// refreshPositionForInput updates cached panel origin used by click hit-testing.
// This keeps Update() independent from Draw() timing and window resize order.
func (p *ComposePanel) refreshPositionForInput() {
	w, h := ebiten.WindowSize()
	if w <= 0 || h <= 0 {
		w = p.screenWidth
		h = p.screenHeight
	}
	if w <= 0 || h <= 0 {
		w, h = 800, 600
	}
	p.x, p.y = p.calculatePosition(w, h)
}

// handleMouseClick detects left-clicks on the Submit and Cancel buttons.
// Must be called under p.mu write lock.
// Returns true if a button was clicked (input consumed).
func (p *ComposePanel) handleMouseClick() bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	cx, cy := ebiten.CursorPosition()
	return p.handleClickAt(cx, cy)
}

// handleClickAt checks whether (cx, cy) hits a button and acts on it.
// Extracted so the logic can be reused in tests via the stub.
// Must be called under p.mu write lock.
func (p *ComposePanel) handleClickAt(cx, cy int) bool {
	const submitWidth = 100
	const cancelWidth = 80
	buttonY := p.y + p.height - p.theme.Padding - p.theme.ButtonHeight
	submitX := p.x + p.width - p.theme.Padding - submitWidth
	cancelX := p.x + p.theme.Padding
	if cy >= buttonY && cy < buttonY+p.theme.ButtonHeight {
		if cx >= submitX && cx < submitX+submitWidth {
			p.submit()
			return true
		}
		if cx >= cancelX && cx < cancelX+cancelWidth {
			p.visible = false
			return true
		}
	}
	return false
}

// handleTextInput processes keyboard input for the text field.
func (p *ComposePanel) handleTextInput() {
	p.processCharacterInput()
	p.processBackspace()
	p.processDelete()
	p.processCursorMovement()
}

// processCharacterInput handles typed characters.
func (p *ComposePanel) processCharacterInput() {
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		p.insertChar(ch)
	}
}

// processBackspace handles backspace key.
func (p *ComposePanel) processBackspace() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) < 20 {
		return
	}
	if p.cursorPos > 0 && len(p.content) > 0 {
		runes := []rune(p.content)
		if p.cursorPos <= len(runes) {
			p.content = string(runes[:p.cursorPos-1]) + string(runes[p.cursorPos:])
			p.cursorPos--
		}
	}
}

// processDelete handles delete key.
func (p *ComposePanel) processDelete() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		return
	}
	runes := []rune(p.content)
	if p.cursorPos < len(runes) {
		p.content = string(runes[:p.cursorPos]) + string(runes[p.cursorPos+1:])
	}
}

// processCursorMovement handles arrow keys and home/end.
func (p *ComposePanel) processCursorMovement() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && p.cursorPos > 0 {
		p.cursorPos--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && p.cursorPos < utf8.RuneCountInString(p.content) {
		p.cursorPos++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		p.cursorPos = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		p.cursorPos = utf8.RuneCountInString(p.content)
	}
}

// insertChar inserts a character at the cursor position.
func (p *ComposePanel) insertChar(ch rune) {
	if len(p.content) >= MaxWaveLength {
		p.anim.SetError("Maximum length reached")
		return
	}

	p.content, p.cursorPos = InsertRuneAtCursor(p.content, p.cursorPos, ch)
}

// submit validates and submits the Wave.
func (p *ComposePanel) submit() {
	if len(p.content) == 0 {
		p.anim.SetError("Cannot send empty Wave")
		return
	}

	if p.onSubmit != nil {
		p.onSubmit(p.content, p.waveType, p.targetNodeID)
	}

	// Clear content after successful submit.
	p.content = ""
	p.cursorPos = 0
	p.visible = false
}

// Draw renders the panel to the screen.
func (p *ComposePanel) Draw(screen *ebiten.Image) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx := InitPanelDrawWithScreen(screen, p.visible, p.calculatePosition, &p.screenWidth, &p.screenHeight)
	if ctx == nil {
		return
	}

	// Store base panel position so Update() can compute button hit-rects
	// without recalculating from screen dimensions independently.
	p.x = ctx.PanelX
	p.y = ctx.PanelY

	px := ctx.PanelX
	py := ctx.PanelY + int(p.anim.SlideOffset()) // Apply slide animation.

	// Draw panel background with border.
	p.drawBackground(screen, px, py)

	// Draw title.
	p.drawTitle(screen, px, py)

	// Draw text area.
	p.drawTextArea(screen, px, py)

	// Draw character count.
	p.drawCharCount(screen, px, py)

	// Draw buttons.
	p.drawButtons(screen, px, py)

	// Draw error message if present.
	if p.anim.ErrorMessage() != "" {
		p.drawError(screen, px, py)
	}
}

// calculatePosition returns the panel's top-left corner based on position setting.
func (p *ComposePanel) calculatePosition(screenW, screenH int) (int, int) {
	p.applyResponsiveLayout(screenW, screenH)
	margin := 20

	switch p.position {
	case PositionCenter:
		return (screenW - p.width) / 2, (screenH - p.height) / 2
	case PositionTopLeft:
		return margin, margin
	case PositionTopRight:
		return screenW - p.width - margin, margin
	case PositionBottomLeft:
		return margin, screenH - p.height - margin
	case PositionBottomRight:
		return screenW - p.width - margin, screenH - p.height - margin
	case PositionLeft:
		return margin, (screenH - p.height) / 2
	case PositionRight:
		return screenW - p.width - margin, (screenH - p.height) / 2
	default:
		return (screenW - p.width) / 2, (screenH - p.height) / 2
	}
}

func (p *ComposePanel) applyResponsiveLayout(screenW, screenH int) {
	if screenW <= 768 {
		p.width = screenW - 24
		if p.width < 200 {
			p.width = screenW - 8
		}
		if p.width < 120 {
			p.width = 120
		}
		if p.width > screenW {
			p.width = screenW
		}
		p.height = screenH - 24
		if p.height > 360 {
			p.height = 360
		}
		if p.height < 220 {
			p.height = 220
		}
		p.position = PositionCenter
		return
	}

	if screenW <= 1024 {
		p.width = 420
		if p.width > screenW-32 {
			p.width = screenW - 32
		}
		p.height = 300
		if p.height > screenH-32 {
			p.height = screenH - 32
		}
		p.position = PositionBottomRight
		return
	}

	p.width = 400
	p.height = 280
	p.position = PositionBottomRight
}

// drawBackground draws the panel background and border.
func (p *ComposePanel) drawBackground(screen *ebiten.Image, px, py int) {
	// Draw shadow.
	shadowColor := color.RGBA{0, 0, 0, 60}
	vector.DrawFilledRect(screen, float32(px+4), float32(py+4),
		float32(p.width), float32(p.height), shadowColor, true)

	// Draw background.
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)

	// Draw border.
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 1.5, p.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (p *ComposePanel) drawTitle(screen *ebiten.Image, px, py int) {
	// Title bar background.
	titleHeight := 40
	titleBg := color.RGBA{
		R: p.theme.PanelBackground.R + 10,
		G: p.theme.PanelBackground.G + 10,
		B: p.theme.PanelBackground.B + 15,
		A: p.theme.PanelBackground.A,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(titleHeight), titleBg, true)

	// Title text.
	title := "Compose Wave"
	if p.targetNodeID != "" {
		title = "Reply to " + p.targetNodeID[:8] + "..."
	}

	drawUICenteredText(screen, title, float64(px)+float64(p.width)/2, float64(py)+float64(titleHeight)/2-4, p.theme.TextPrimary)
}

// drawTextArea draws the text input area.
func (p *ComposePanel) drawTextArea(screen *ebiten.Image, px, py int) {
	// Text area position and size.
	textX := px + p.theme.Padding
	textY := py + 50
	textW := p.width - p.theme.Padding*2
	textH := 140

	// Background.
	vector.DrawFilledRect(screen, float32(textX), float32(textY),
		float32(textW), float32(textH), p.theme.InputBackground, true)

	// Border.
	borderColor := p.theme.PanelBorder
	vector.StrokeRect(screen, float32(textX), float32(textY),
		float32(textW), float32(textH), 1.0, borderColor, true)

	// Render wave content (placeholder text shown when empty).
	if p.content == "" {
		drawUIText(screen, "Type your Wave here...", float64(textX)+8, float64(textY)+6, p.theme.TextPlaceholder)
	} else {
		lineHeight := 16
		maxVisible := (textH - 12) / lineHeight
		if maxVisible < 1 {
			maxVisible = 1
		}

		lines := composeWrapLines(p.content, float64(textW-16))
		prefix := ""
		runes := []rune(p.content)
		if p.cursorPos > len(runes) {
			p.cursorPos = len(runes)
		}
		if p.cursorPos > 0 {
			prefix = string(runes[:p.cursorPos])
		}
		cursorLines := composeWrapLines(prefix, float64(textW-16))
		cursorLine := len(cursorLines) - 1
		if cursorLine < 0 {
			cursorLine = 0
		}

		startLine := 0
		if len(lines) > maxVisible {
			startLine = len(lines) - maxVisible
		}
		if cursorLine >= startLine+maxVisible {
			startLine = cursorLine - maxVisible + 1
		}
		if cursorLine < startLine {
			startLine = cursorLine
		}

		endLine := startLine + maxVisible
		if endLine > len(lines) {
			endLine = len(lines)
		}

		drawY := textY + 6
		for i := startLine; i < endLine; i++ {
			drawUIText(screen, lines[i], float64(textX)+8, float64(drawY), p.theme.TextPrimary)
			drawY += lineHeight
		}

		// Draw cursor at wrapped position.
		if int(p.anim.AnimTime()*2)%2 == 0 {
			cursorDrawLine := cursorLine - startLine
			if cursorDrawLine >= 0 && cursorDrawLine < maxVisible {
				cursorPrefix := ""
				if len(cursorLines) > 0 {
					cursorPrefix = cursorLines[len(cursorLines)-1]
				}
				prefixW, _ := measureUIText(cursorPrefix)
				cursorX := textX + 8 + int(prefixW)
				cursorY := textY + 4 + cursorDrawLine*lineHeight
				vector.DrawFilledRect(screen, float32(cursorX), float32(cursorY),
					2, 14, p.theme.TextPrimary, true)
			}
		}
	}
}

func composeWrapLines(text string, maxWidth float64) []string {
	if text == "" {
		return []string{""}
	}

	lines := make([]string, 0, 8)
	current := ""
	for _, r := range []rune(text) {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
			continue
		}

		next := current + string(r)
		nextW, _ := measureUIText(next)
		if nextW > maxWidth && current != "" {
			lines = append(lines, current)
			current = string(r)
		} else {
			current = next
		}
	}
	lines = append(lines, current)
	return lines
}

// drawCharCount draws the character count indicator.
func (p *ComposePanel) drawCharCount(screen *ebiten.Image, px, py int) {
	countX := px + p.width - p.theme.Padding - 80
	countY := py + 195

	charCount := len(p.content)
	var countColor color.RGBA
	if charCount > MaxWaveLength-100 {
		countColor = p.theme.TextError
	} else if charCount > MaxWaveLength-500 {
		countColor = color.RGBA{255, 200, 100, 255}
	} else {
		countColor = p.theme.TextSecondary
	}
	countText := strconv.Itoa(charCount) + "/" + strconv.Itoa(MaxWaveLength)
	drawUIText(screen, countText, float64(countX), float64(countY), countColor)
}

// drawButtons draws the submit and cancel buttons.
func (p *ComposePanel) drawButtons(screen *ebiten.Image, px, py int) {
	enabled := len(p.content) > 0
	DrawCancelSubmitButtons(screen, px, py, p.width, p.height, p.theme, 100, "Submit", enabled)
}

// drawError draws the error message.
func (p *ComposePanel) drawError(screen *ebiten.Image, px, py int) {
	errorY := py + p.height - p.theme.Padding - p.theme.ButtonHeight - 25

	// Error background.
	errorBg := color.RGBA{80, 30, 30, 200}
	vector.DrawFilledRect(screen, float32(px+p.theme.Padding), float32(errorY),
		float32(p.width-p.theme.Padding*2), 20, errorBg, true)

	// Error message text.
	if msg := p.anim.ErrorMessage(); msg != "" {
		drawUIText(screen, msg, float64(px+p.theme.Padding)+6, float64(errorY)+4, p.theme.TextError)
	}
}

// Content returns the current input content (for testing).
func (p *ComposePanel) Content() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.content
}

// SetContent sets the input content (for testing).
func (p *ComposePanel) SetContent(content string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(content) > MaxWaveLength {
		content = content[:MaxWaveLength]
	}
	p.content = content
	p.cursorPos = utf8.RuneCountInString(content)
}
