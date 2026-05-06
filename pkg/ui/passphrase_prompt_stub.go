// Package ui provides master key passphrase prompt stubs for testing.

//go:build test
// +build test

package ui

import "image"

// PassphrasePromptPanel displays a passphrase input prompt (stub).
type PassphrasePromptPanel struct {
	visible bool
}

// NewPassphrasePromptPanel creates a new passphrase prompt panel (stub).
func NewPassphrasePromptPanel(theme Theme, onSubmit func(string) error, onCancel func()) *PassphrasePromptPanel {
	return &PassphrasePromptPanel{}
}

// Visible returns true if the panel is shown (stub).
func (p *PassphrasePromptPanel) Visible() bool {
	return p.visible
}

// Show displays the panel (stub).
func (p *PassphrasePromptPanel) Show(title, message string) {
	p.visible = true
}

// Hide hides the panel (stub).
func (p *PassphrasePromptPanel) Hide() {
	p.visible = false
}

// Update handles input (stub).
func (p *PassphrasePromptPanel) Update() bool {
	return p.visible
}

// Draw renders the panel (stub).
func (p *PassphrasePromptPanel) Draw(screen image.Image) {
	// Stub: no rendering
}

// SetError sets an error message (stub).
func (p *PassphrasePromptPanel) SetError(msg string) {
	// Stub: no error display
}
