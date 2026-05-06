// Package ui provides UI components for MURMUR.
// Test stub for viewport controls.
//
//go:build test
// +build test

package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ViewportControls test stub.
type ViewportControls struct {
	theme        Theme
	onMacro      func()
	onMeso       func()
	onMicro      func()
	buttonWidth  int
	buttonHeight int
	buttonGap    int
}

// ViewportCallbacks test stub.
type ViewportCallbacks struct {
	OnMacro func()
	OnMeso  func()
	OnMicro func()
}

// NewViewportControls test stub.
func NewViewportControls(theme Theme, callbacks ViewportCallbacks) *ViewportControls {
	return &ViewportControls{
		theme:        theme,
		onMacro:      callbacks.OnMacro,
		onMeso:       callbacks.OnMeso,
		onMicro:      callbacks.OnMicro,
		buttonWidth:  70,
		buttonHeight: 44,
		buttonGap:    5,
	}
}

// Update test stub.
func (v *ViewportControls) Update() bool {
	return false
}

// Draw test stub.
func (v *ViewportControls) Draw(screen *ebiten.Image) {
	// No-op for tests.
}
