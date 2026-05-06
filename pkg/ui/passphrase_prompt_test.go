package ui

import (
	"errors"
	"testing"
)

// TestMaskPassphrase verifies the masking helper produces one bullet per rune.
// Per AUDIT.md HIGH finding: the original inline loop was broken and produced
// at most one bullet; this test would have caught the regression.
func TestMaskPassphrase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"abc", "•••"},
		{"", ""},
		{"a", "•"},
		{"héllo", "•••••"}, // multi-byte runes — count by rune not byte
	}
	for _, c := range cases {
		got := maskPassphrase(c.input)
		if got != c.want {
			t.Errorf("maskPassphrase(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

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
