// Package ui provides master key passphrase prompt for device operations.

//go:build !test
// +build !test

package ui

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PassphrasePromptPanel displays a passphrase input prompt.
type PassphrasePromptPanel struct {
	mu sync.RWMutex

	visible bool
	x, y    int
	width   int
	height  int
	theme   Theme

	title      string
	message    string
	passphrase string
	showPass   bool
	errorMsg   string

	// toggleBtnX/Y/W/H cache the Show/Hide button rect set during Draw() so
	// Update() can perform a hit-test without recomputing modal geometry.
	toggleBtnX, toggleBtnY int
	toggleBtnW, toggleBtnH int
	submitBtnX, submitBtnY int
	submitBtnW, submitBtnH int
	cancelBtnX, cancelBtnY int
	cancelBtnW, cancelBtnH int

	onSubmit func(passphrase string) error
	onCancel func()
}

// NewPassphrasePromptPanel creates a new passphrase prompt panel.
func NewPassphrasePromptPanel(theme Theme, onSubmit func(string) error, onCancel func()) *PassphrasePromptPanel {
	return &PassphrasePromptPanel{
		theme:    theme,
		width:    500,
		height:   300,
		title:    "Master Key Required",
		message:  "Enter your master key passphrase to continue:",
		onSubmit: onSubmit,
		onCancel: onCancel,
	}
}

// Visible returns true if the panel is shown.
func (p *PassphrasePromptPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with optional custom title and message.
func (p *PassphrasePromptPanel) Show(title, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.passphrase = ""
	p.errorMsg = ""
	if title != "" {
		p.title = title
	}
	if message != "" {
		p.message = message
	}
}

// Hide hides the panel.
func (p *PassphrasePromptPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.passphrase = ""
	p.errorMsg = ""
}

// Update handles input and updates panel state.
func (p *PassphrasePromptPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	if p.handleEscapeKey() {
		return true
	}
	leftJustPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	mx, my := ebiten.CursorPosition()
	if p.handleEnterKey() {
		return true
	}
	// Check the Show/Hide toggle before backspace so a click on the button
	// doesn't also trigger a character deletion.
	if p.handleShowPassToggle(mx, my, leftJustPressed) {
		return true
	}
	if p.handleButtonsClick(mx, my, leftJustPressed) {
		return true
	}
	if p.handleBackspaceKey() {
		return true
	}
	p.handleTextInput()
	return true
}

// handleEscapeKey processes Escape key for cancel action.
func (p *PassphrasePromptPanel) handleEscapeKey() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		if p.onCancel != nil {
			p.onCancel()
		}
		return true
	}
	return false
}

// handleEnterKey processes Enter key for submit action.
func (p *PassphrasePromptPanel) handleEnterKey() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		p.submit()
		return true
	}
	return false
}

func (p *PassphrasePromptPanel) submit() {
	if p.passphrase == "" || p.onSubmit == nil {
		return
	}
	if err := p.onSubmit(p.passphrase); err != nil {
		p.errorMsg = err.Error()
		return
	}
	p.visible = false
}

// handleShowPassToggle detects a left-click on the Show/Hide button and toggles
// the showPass state so the user can verify what they typed.
// Per AUDIT.md HIGH finding: showPass was declared but never set.
func (p *PassphrasePromptPanel) handleShowPassToggle(mx, my int, leftJustPressed bool) bool {
	if p.toggleBtnW == 0 || p.toggleBtnH == 0 {
		return false // Button rect not yet set (first frame before Draw).
	}
	if !leftJustPressed {
		return false
	}
	if mx >= p.toggleBtnX && mx < p.toggleBtnX+p.toggleBtnW &&
		my >= p.toggleBtnY && my < p.toggleBtnY+p.toggleBtnH {
		p.showPass = !p.showPass
		return true
	}
	return false
}

// handleButtonsClick processes mouse clicks on the Cancel and Submit buttons.
func (p *PassphrasePromptPanel) handleButtonsClick(mx, my int, leftJustPressed bool) bool {
	if !leftJustPressed {
		return false
	}
	if p.cancelBtnW > 0 && p.cancelBtnH > 0 &&
		mx >= p.cancelBtnX && mx < p.cancelBtnX+p.cancelBtnW &&
		my >= p.cancelBtnY && my < p.cancelBtnY+p.cancelBtnH {
		p.visible = false
		if p.onCancel != nil {
			p.onCancel()
		}
		return true
	}
	if p.submitBtnW > 0 && p.submitBtnH > 0 &&
		mx >= p.submitBtnX && mx < p.submitBtnX+p.submitBtnW &&
		my >= p.submitBtnY && my < p.submitBtnY+p.submitBtnH {
		p.submit()
		return true
	}
	return false
}

// handleBackspaceKey processes Backspace key for character deletion.
func (p *PassphrasePromptPanel) handleBackspaceKey() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(p.passphrase) > 0 {
			p.passphrase = p.passphrase[:len(p.passphrase)-1]
		}
		return true
	}
	return false
}

// handleTextInput appends typed characters to passphrase.
func (p *PassphrasePromptPanel) handleTextInput() {
	p.passphrase += string(ebiten.AppendInputChars(nil))
}

// Draw renders the panel.
func (p *PassphrasePromptPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	px, py, _ := DrawModalWithTitle(screen, p.visible, p.width, p.height, p.theme, p.title)
	if px == 0 {
		return
	}

	// Draw message
	msgY := py + 80
	drawUICenteredText(screen, p.message, float64(px+p.width/2), float64(msgY), p.theme.TextSecondary)

	// Draw input box
	inputY := py + 130
	inputWidth := p.width - 80
	inputHeight := 40
	inputX := px + 40

	vector.DrawFilledRect(screen, float32(inputX), float32(inputY),
		float32(inputWidth), float32(inputHeight), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(inputX), float32(inputY),
		float32(inputWidth), float32(inputHeight), 1.0, p.theme.PanelBorder, true)

	// Draw Show/Hide toggle button at the right edge of the input box.
	// Per AUDIT.md HIGH finding: showPass was declared but the toggle was never drawn.
	const toggleW = 40
	toggleH := inputHeight
	toggleX := inputX + inputWidth - toggleW
	toggleY := inputY
	toggleLabel := "Show"
	if p.showPass {
		toggleLabel = "Hide"
	}
	vector.DrawFilledRect(screen, float32(toggleX), float32(toggleY),
		float32(toggleW), float32(toggleH), p.theme.ButtonBackground, true)
	vector.StrokeRect(screen, float32(toggleX), float32(toggleY),
		float32(toggleW), float32(toggleH), 1.0, p.theme.PanelBorder, true)
	drawUICenteredText(screen, toggleLabel,
		float64(toggleX+toggleW/2), float64(toggleY+toggleH/2), p.theme.TextPrimary)

	// Cache toggle button geometry so Update() can hit-test it next frame.
	p.toggleBtnX = toggleX
	p.toggleBtnY = toggleY
	p.toggleBtnW = toggleW
	p.toggleBtnH = toggleH

	// Draw passphrase (masked or visible depending on showPass).
	// Per AUDIT.md HIGH finding: the original masking loop was broken —
	// maskPassphrase() in passphrase_mask.go produces the correct bullet count.
	passText := p.passphrase
	if !p.showPass {
		passText = maskPassphrase(p.passphrase)
	}
	if passText == "" {
		passText = "(passphrase)"
	}
	drawUICenteredText(screen, passText,
		float64(inputX+inputWidth/2), float64(inputY+inputHeight/2), p.theme.TextPrimary)

	// Draw error message if present
	if p.errorMsg != "" {
		errY := inputY + inputHeight + 20
		drawUICenteredText(screen, p.errorMsg, float64(px+p.width/2), float64(errY), p.theme.TextError)
	}

	// Draw buttons
	p.drawButtons(screen, px, py+p.height-60)
}

// drawButtons draws submit and cancel buttons.
func (p *PassphrasePromptPanel) drawButtons(screen *ebiten.Image, x, y int) {
	btnWidth := 100
	btnHeight := 40
	btnSpacing := 20

	// Cancel button
	cancelX := x + (p.width/2 - btnWidth - btnSpacing/2)
	vector.DrawFilledRect(screen, float32(cancelX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.ButtonBackground, true)
	vector.StrokeRect(screen, float32(cancelX), float32(y),
		float32(btnWidth), float32(btnHeight), 1.0, p.theme.PanelBorder, true)
	drawUICenteredText(screen, "Cancel", float64(cancelX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
	p.cancelBtnX = cancelX
	p.cancelBtnY = y
	p.cancelBtnW = btnWidth
	p.cancelBtnH = btnHeight

	// Submit button
	submitX := x + (p.width/2 + btnSpacing/2)
	vector.DrawFilledRect(screen, float32(submitX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.AccentPrimary, true)
	drawUICenteredText(screen, "Submit", float64(submitX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
	p.submitBtnX = submitX
	p.submitBtnY = y
	p.submitBtnW = btnWidth
	p.submitBtnH = btnHeight
}

// SetError sets an error message to display.
func (p *PassphrasePromptPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMsg = msg
}
