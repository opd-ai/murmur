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

func TestPassphrasePromptPanel_HandleButtonsClick_Submit(t *testing.T) {
	submitCalls := 0
	panel := NewPassphrasePromptPanel(DefaultTheme(), func(passphrase string) error {
		submitCalls++
		if passphrase != "secret" {
			t.Fatalf("unexpected passphrase: %q", passphrase)
		}
		return nil
	}, nil)

	panel.Show("", "")
	panel.passphrase = "secret"
	panel.submitBtnX, panel.submitBtnY = 100, 100
	panel.submitBtnW, panel.submitBtnH = 100, 40

	hit := panel.handleButtonsClick(120, 120, true)
	if !hit {
		t.Fatal("expected submit button click to be handled")
	}
	if submitCalls != 1 {
		t.Fatalf("expected submit callback count 1, got %d", submitCalls)
	}
	if panel.Visible() {
		t.Fatal("panel should hide after successful submit click")
	}
}

func TestPassphrasePromptPanel_HandleButtonsClick_Cancel(t *testing.T) {
	cancelCalls := 0
	panel := NewPassphrasePromptPanel(DefaultTheme(), nil, func() {
		cancelCalls++
	})

	panel.Show("", "")
	panel.cancelBtnX, panel.cancelBtnY = 200, 150
	panel.cancelBtnW, panel.cancelBtnH = 100, 40

	hit := panel.handleButtonsClick(220, 170, true)
	if !hit {
		t.Fatal("expected cancel button click to be handled")
	}
	if cancelCalls != 1 {
		t.Fatalf("expected cancel callback count 1, got %d", cancelCalls)
	}
	if panel.Visible() {
		t.Fatal("panel should hide after cancel click")
	}
}
