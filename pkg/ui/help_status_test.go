// Package ui – tests for help button and status bar (test build).
//
//go:build test
// +build test

package ui

import "testing"

// TestHelpButtonTrigger verifies the onClick callback fires via Trigger().
func TestHelpButtonTrigger(t *testing.T) {
	t.Parallel()

	called := false
	btn := NewHelpButton(10, 10, Theme{}, func() { called = true })
	btn.Trigger()
	if !called {
		t.Error("HelpButton.Trigger() should call onClick")
	}
}

// TestHelpButtonNilCallback verifies no panic when onClick is nil.
func TestHelpButtonNilCallback(t *testing.T) {
	t.Parallel()

	btn := NewHelpButton(0, 0, Theme{}, nil)
	btn.Trigger() // should not panic
}

// TestHelpButtonSetVisible verifies visibility can be toggled.
func TestHelpButtonSetVisible(t *testing.T) {
	t.Parallel()

	btn := NewHelpButton(0, 0, Theme{}, nil)
	btn.SetVisible(false)
	if btn.visible {
		t.Error("SetVisible(false) should set visible to false")
	}
	btn.SetVisible(true)
	if !btn.visible {
		t.Error("SetVisible(true) should set visible to true")
	}
}

// TestHelpButtonSetPosition verifies position update.
func TestHelpButtonSetPosition(t *testing.T) {
	t.Parallel()

	btn := NewHelpButton(0, 0, Theme{}, nil)
	btn.SetPosition(42, 99)
	if btn.x != 42 || btn.y != 99 {
		t.Errorf("SetPosition(42,99): got (%d,%d)", btn.x, btn.y)
	}
}

// TestStatusBarDefaults verifies StatusBar zero-value fields.
func TestStatusBarDefaults(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(0, 0, 800, 20, Theme{})
	if sb.PeerCount != 0 {
		t.Errorf("initial PeerCount = %d, want 0", sb.PeerCount)
	}
	if sb.ShroudActive {
		t.Error("initial ShroudActive should be false")
	}
	if sb.IdentityPublished {
		t.Error("initial IdentityPublished should be false")
	}
	if sb.PowBusy {
		t.Error("initial PowBusy should be false")
	}
}

// TestStatusBarFieldMutation verifies that public fields can be updated.
func TestStatusBarFieldMutation(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(0, 0, 800, 20, Theme{})
	sb.PeerCount = 12
	sb.ShroudActive = true
	sb.IdentityPublished = true
	sb.PowBusy = true
	sb.PowProgress = 0.5

	if sb.PeerCount != 12 {
		t.Errorf("PeerCount = %d, want 12", sb.PeerCount)
	}
	if !sb.ShroudActive {
		t.Error("ShroudActive should be true")
	}
	if !sb.IdentityPublished {
		t.Error("IdentityPublished should be true")
	}
	if sb.PowProgress != 0.5 {
		t.Errorf("PowProgress = %f, want 0.5", sb.PowProgress)
	}
}
