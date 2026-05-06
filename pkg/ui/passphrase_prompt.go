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

	// Handle Escape to cancel
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		if p.onCancel != nil {
			p.onCancel()
		}
		return true
	}

	// Handle Enter to submit
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		if p.passphrase != "" && p.onSubmit != nil {
			if err := p.onSubmit(p.passphrase); err != nil {
				p.errorMsg = err.Error()
			} else {
				p.visible = false
			}
		}
		return true
	}

	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(p.passphrase) > 0 {
			p.passphrase = p.passphrase[:len(p.passphrase)-1]
		}
		return true
	}

	// Handle text input
	p.passphrase += string(ebiten.AppendInputChars(nil))

	return true
}

// Draw renders the panel.
func (p *PassphrasePromptPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	px := (w - p.width) / 2
	py := (h - p.height) / 2

	// Draw overlay
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), p.theme.PanelBackground, true)

	// Draw panel background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 2.0, p.theme.PanelBorder, true)

	// Draw title
	titleY := py + 30
	drawUICenteredText(screen, p.title, float64(px+p.width/2), float64(titleY), p.theme.TextPrimary)

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

	// Draw passphrase (masked)
	passText := p.passphrase
	if !p.showPass && len(passText) > 0 {
		passText = string(make([]byte, len(passText)))
		for i := range passText {
			passText = passText[:i] + "•"
		}
	}
	if passText == "" {
		passText = "(passphrase)"
	}

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

	// Submit button
	submitX := x + (p.width/2 + btnSpacing/2)
	vector.DrawFilledRect(screen, float32(submitX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.AccentPrimary, true)
	drawUICenteredText(screen, "Submit", float64(submitX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
}

// SetError sets an error message to display.
func (p *PassphrasePromptPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMsg = msg
}
