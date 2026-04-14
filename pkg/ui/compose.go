// Package ui provides the Compose panel for creating Waves.
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"image/color"
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
	errorMessage string
	errorTime    float64

	// Callbacks
	onSubmit WaveSubmitCallback

	// Styling
	theme Theme

	// Animation
	animTime    float64
	slideOffset float64

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
	p.slideOffset = float64(p.height)
	p.animTime = 0
}

// Hide hides the panel.
func (p *ComposePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.content = ""
	p.cursorPos = 0
	p.errorMessage = ""
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

	// Animate slide-in.
	p.animTime += 1.0 / 60.0
	if p.slideOffset > 0 {
		p.slideOffset *= 0.85
		if p.slideOffset < 1 {
			p.slideOffset = 0
		}
	}

	// Clear error after 3 seconds.
	if p.errorMessage != "" {
		p.errorTime += 1.0 / 60.0
		if p.errorTime > 3.0 {
			p.errorMessage = ""
			p.errorTime = 0
		}
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
	if !inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) <= 20 {
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
		p.errorMessage = "Maximum length reached"
		p.errorTime = 0
		return
	}

	runes := []rune(p.content)
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:p.cursorPos]...)
	newRunes = append(newRunes, ch)
	newRunes = append(newRunes, runes[p.cursorPos:]...)
	p.content = string(newRunes)
	p.cursorPos++
}

// submit validates and submits the Wave.
func (p *ComposePanel) submit() {
	if len(p.content) == 0 {
		p.errorMessage = "Cannot send empty Wave"
		p.errorTime = 0
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
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	// Get screen dimensions.
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	p.screenWidth = w
	p.screenHeight = h

	// Calculate panel position based on anchor.
	px, py := p.calculatePosition(w, h)
	py += int(p.slideOffset) // Apply slide animation.

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
	if p.errorMessage != "" {
		p.drawError(screen, px, py)
	}
}

// calculatePosition returns the panel's top-left corner based on position setting.
func (p *ComposePanel) calculatePosition(screenW, screenH int) (int, int) {
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

	// Draw title text (simplified - would use text/v2 with proper font).
	_ = title // Title rendering requires font setup.
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

	// Draw cursor (blinking).
	if int(p.animTime*2)%2 == 0 {
		cursorX := textX + 8 + p.cursorPos*8 // Simplified cursor positioning.
		cursorY := textY + 8
		vector.DrawFilledRect(screen, float32(cursorX), float32(cursorY),
			2, 16, p.theme.TextPrimary, true)
	}

	// Content rendering would use text/v2 with proper font.
	// Text rendering is deferred until font loading is implemented.
}

// drawCharCount draws the character count indicator.
func (p *ComposePanel) drawCharCount(screen *ebiten.Image, px, py int) {
	countX := px + p.width - p.theme.Padding - 80
	countY := py + 195

	// Character count.
	charCount := len(p.content)
	var countColor color.RGBA
	if charCount > MaxWaveLength-100 {
		countColor = p.theme.TextError
	} else if charCount > MaxWaveLength-500 {
		countColor = color.RGBA{255, 200, 100, 255}
	} else {
		countColor = p.theme.TextSecondary
	}

	// Draw count indicator (simplified).
	_ = countColor
	_ = countX
	_ = countY
}

// drawButtons draws the submit and cancel buttons.
func (p *ComposePanel) drawButtons(screen *ebiten.Image, px, py int) {
	buttonY := py + p.height - p.theme.Padding - p.theme.ButtonHeight

	// Cancel button.
	cancelX := px + p.theme.Padding
	cancelW := 80
	vector.DrawFilledRect(screen, float32(cancelX), float32(buttonY),
		float32(cancelW), float32(p.theme.ButtonHeight), p.theme.ButtonBackground, true)
	vector.StrokeRect(screen, float32(cancelX), float32(buttonY),
		float32(cancelW), float32(p.theme.ButtonHeight), 1.0, p.theme.PanelBorder, true)

	// Submit button.
	submitX := px + p.width - p.theme.Padding - 100
	submitW := 100
	submitBg := p.theme.AccentPrimary
	if len(p.content) == 0 {
		submitBg = p.theme.ButtonBackground
	}
	vector.DrawFilledRect(screen, float32(submitX), float32(buttonY),
		float32(submitW), float32(p.theme.ButtonHeight), submitBg, true)
}

// drawError draws the error message.
func (p *ComposePanel) drawError(screen *ebiten.Image, px, py int) {
	errorY := py + p.height - p.theme.Padding - p.theme.ButtonHeight - 25

	// Error background.
	errorBg := color.RGBA{80, 30, 30, 200}
	vector.DrawFilledRect(screen, float32(px+p.theme.Padding), float32(errorY),
		float32(p.width-p.theme.Padding*2), 20, errorBg, true)

	// Error text would be rendered with text/v2.
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
