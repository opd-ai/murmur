package ui

import (
	"errors"
	"testing"
)

func TestPassphrasePromptPanel_Creation(t *testing.T) {
	theme := DefaultTheme()

	onSubmit := func(passphrase string) error {
		if passphrase == "test123" {
			return nil
		}
		return errors.New("invalid passphrase")
	}

	onCancel := func() {
	}

	panel := NewPassphrasePromptPanel(theme, onSubmit, onCancel)
	if panel == nil {
		t.Fatal("Expected panel to be created")
	}

	if panel.Visible() {
		t.Error("Panel should not be visible by default")
	}
}

func TestPassphrasePromptPanel_ShowHide(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPassphrasePromptPanel(theme, nil, nil)

	// Show panel
	panel.Show("Test Title", "Test message")

	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	// Hide panel
	panel.Hide()

	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestPassphrasePromptPanel_ErrorHandling(t *testing.T) {
	theme := DefaultTheme()

	onSubmit := func(passphrase string) error {
		return errors.New("test error")
	}

	panel := NewPassphrasePromptPanel(theme, onSubmit, nil)

	// Set error
	panel.SetError("Test error message")

	// Verify panel still functional
	panel.Show("", "")
	if !panel.Visible() {
		t.Error("Panel should be visible after error")
	}
}

func TestPassphrasePromptPanel_CustomMessages(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPassphrasePromptPanel(theme, nil, nil)

	// Test with custom title and message
	panel.Show("Custom Title", "Custom message text")

	if !panel.Visible() {
		t.Error("Panel should be visible with custom messages")
	}
}
