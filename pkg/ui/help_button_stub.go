// Package ui provides stub types for the Help button (test build).
//
//go:build test
// +build test

package ui

// HelpButton is a stub for the help/onboarding tutorial button.
// Per PLAN.md: "Help button — reopen onboarding tutorials at any time".
type HelpButton struct {
	x, y    int
	visible bool
	onClick func()
	theme   Theme
}

// NewHelpButton creates a help button stub.
func NewHelpButton(x, y int, theme Theme, onClick func()) *HelpButton {
	return &HelpButton{x: x, y: y, visible: true, theme: theme, onClick: onClick}
}

// SetPosition updates the button's screen position.
func (h *HelpButton) SetPosition(x, y int) { h.x = x; h.y = y }

// SetVisible controls whether the button is visible.
func (h *HelpButton) SetVisible(v bool) { h.visible = v }

// Update processes input (no-op in test mode).
func (h *HelpButton) Update() {}

// Draw renders the button (no-op in test mode).
func (h *HelpButton) Draw(_ interface{}) {}

// Trigger calls the onClick callback directly (test helper).
func (h *HelpButton) Trigger() {
	if h.onClick != nil {
		h.onClick()
	}
}
