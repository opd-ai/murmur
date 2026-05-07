// Package ui - ComposePanel unit tests.
//
//go:build test
// +build test

package ui

import (
	"testing"
)

// buttonCoords returns the expected pixel coordinates of the Submit and Cancel
// buttons for a ComposePanel at position (panelX, panelY) with the given theme.
// Mirrors the layout in DrawCancelSubmitButtons and handleClickAt.
func buttonCoords(panelX, panelY, panelW, panelH int, t Theme) (submitX, cancelX, buttonY int) {
	buttonY = panelY + panelH - t.Padding - t.ButtonHeight
	submitX = panelX + panelW - t.Padding - 100
	cancelX = panelX + t.Padding
	return submitX, cancelX, buttonY
}

func TestComposePanel_SimulateClick_Submit(t *testing.T) {
	submitted := false
	var submittedContent string

	theme := DefaultTheme()
	panel := NewComposePanel(theme, func(content string, waveType uint8, targetNodeID string) {
		submitted = true
		submittedContent = content
	})

	panel.Show()
	panel.SetContent("hello wave")

	// Panel defaults: x=0, y=0, width=400, height=280.
	submitX, _, buttonY := buttonCoords(0, 0, 400, 280, theme)

	// Click inside the submit button.
	handled := panel.SimulateClick(submitX+50, buttonY+theme.ButtonHeight/2)
	if !handled {
		t.Error("SimulateClick on submit button should return true")
	}
	if !submitted {
		t.Error("onSubmit callback should have been called after Submit button click")
	}
	if submittedContent != "hello wave" {
		t.Errorf("submitted content = %q, want %q", submittedContent, "hello wave")
	}
	if panel.Visible() {
		t.Error("Panel should be hidden after submit")
	}
}

func TestComposePanel_SimulateClick_Cancel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)

	panel.Show()
	panel.SetContent("some draft text")

	// Panel defaults: x=0, y=0, width=400, height=280.
	_, cancelX, buttonY := buttonCoords(0, 0, 400, 280, theme)

	// Click inside the cancel button.
	handled := panel.SimulateClick(cancelX+40, buttonY+theme.ButtonHeight/2)
	if !handled {
		t.Error("SimulateClick on cancel button should return true")
	}
	if panel.Visible() {
		t.Error("Panel should be hidden after Cancel click")
	}
}

func TestComposePanel_SimulateClick_EmptyContent_DoesNotSubmit(t *testing.T) {
	submitted := false
	theme := DefaultTheme()
	panel := NewComposePanel(theme, func(content string, waveType uint8, targetNodeID string) {
		submitted = true
	})

	panel.Show()
	// Leave content empty.

	submitX, _, buttonY := buttonCoords(0, 0, 400, 280, theme)
	panel.SimulateClick(submitX+50, buttonY+theme.ButtonHeight/2)

	if submitted {
		t.Error("onSubmit should not be called when content is empty")
	}
	// Panel remains visible (or stays up with error) — the key invariant is no callback.
}

func TestComposePanel_SimulateClick_OutsideButtons(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)
	panel.Show()
	panel.SetContent("text")

	// Click far from any button.
	handled := panel.SimulateClick(200, 100)
	if handled {
		t.Error("SimulateClick outside buttons should return false")
	}
	if !panel.Visible() {
		t.Error("Panel should still be visible after a miss-click")
	}
}

func TestComposePanel_SimulateClick_WhenHidden(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)
	// Panel is hidden by default.

	submitX, _, buttonY := buttonCoords(0, 0, 400, 280, theme)
	handled := panel.SimulateClick(submitX+50, buttonY+theme.ButtonHeight/2)
	if handled {
		t.Error("SimulateClick on hidden panel should return false")
	}
}

// TestComposePanel_SimulateBackspace_FirstPress verifies that a single backspace
// deletes the character before the cursor (first-press behavior).
// Per AUDIT.md MEDIUM finding: processBackspace uses < 20 (exclusive) so the first
// key-press event is not blocked by the hold-duration guard.
func TestComposePanel_SimulateBackspace_FirstPress(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)
	panel.Show()
	panel.SetContent("abc")

	// Cursor is at end (position 3). One backspace should remove 'c'.
	panel.SimulateBackspace()

	if panel.Content() != "ab" {
		t.Errorf("after 1 backspace got %q, want %q", panel.Content(), "ab")
	}
}

func TestComposePanel_SimulateBackspace_MultiByte(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)
	panel.Show()
	panel.SetContent("日本語")

	// Cursor at end (position 3). One backspace removes '語'.
	panel.SimulateBackspace()

	if panel.Content() != "日本" {
		t.Errorf("after backspace on CJK content got %q, want %q", panel.Content(), "日本")
	}
}

func TestComposePanel_SimulateBackspace_AtStart(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)
	panel.Show()
	panel.SetContent("x")

	// Move cursor to start: content has 1 rune, cursor at 1. After one backspace → empty.
	panel.SimulateBackspace()
	if panel.Content() != "" {
		t.Errorf("expected empty content, got %q", panel.Content())
	}

	// Backspace at start should be a no-op.
	panel.SimulateBackspace()
	if panel.Content() != "" {
		t.Errorf("backspace at start of empty content modified content: got %q", panel.Content())
	}
}
