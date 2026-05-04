// Package ui - Radial Menu test stubs.
// This file provides no-op stubs for tests that import pkg/ui but don't need
// Ebitengine dependencies.

//go:build test
// +build test

package ui

// RadialMenuAction represents an action available in the radial menu.
type RadialMenuAction uint8

const (
	ActionComposeWave RadialMenuAction = iota
	ActionSendGift
	ActionPlaceMark
	ActionSendWhisper
	ActionJoinGame
	ActionViewDetail
)

// RadialMenuCallbacks provides callbacks for radial menu actions.
type RadialMenuCallbacks struct {
	OnAction          func(action RadialMenuAction, nodeID string)
	IsActionAvailable func(action RadialMenuAction, nodeID string) bool
}

// RadialMenuItem represents a single menu item.
type RadialMenuItem struct {
	Action   RadialMenuAction
	Label    string
	IconCode rune
}

// RadialMenu provides a circular context menu for node actions.
type RadialMenu struct {
	visible bool
}

// NewRadialMenu creates a new radial menu.
func NewRadialMenu(theme Theme, callbacks RadialMenuCallbacks) *RadialMenu {
	return &RadialMenu{}
}

// Show displays the radial menu.
func (r *RadialMenu) Show(screenX, screenY float64, nodeID string) {
	r.visible = true
}

// Hide hides the radial menu.
func (r *RadialMenu) Hide() {
	r.visible = false
}

// Visible returns true if the menu is shown.
func (r *RadialMenu) Visible() bool { return r.visible }

// Toggle toggles menu visibility.
func (r *RadialMenu) Toggle() {
	r.visible = !r.visible
}

// Update handles input (stub).
func (r *RadialMenu) Update() bool { return false }

// Draw renders the menu (stub).
func (r *RadialMenu) Draw(screen Screen) {}
