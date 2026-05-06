// Package ui provides UI components for MURMUR.
// Tests for viewport controls.
package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewViewportControls(t *testing.T) {
	theme := DefaultTheme()

	macroCalled := false
	mesoCalled := false
	microCalled := false

	callbacks := ViewportCallbacks{
		OnMacro: func() { macroCalled = true },
		OnMeso:  func() { mesoCalled = true },
		OnMicro: func() { microCalled = true },
	}

	controls := NewViewportControls(theme, callbacks)
	assert.NotNil(t, controls)
	assert.NotNil(t, controls.onMacro)
	assert.NotNil(t, controls.onMeso)
	assert.NotNil(t, controls.onMicro)

	// Test callbacks are wired correctly.
	controls.onMacro()
	assert.True(t, macroCalled)

	controls.onMeso()
	assert.True(t, mesoCalled)

	controls.onMicro()
	assert.True(t, microCalled)
}

func TestViewportControlsUpdate(t *testing.T) {
	theme := DefaultTheme()
	callbacks := ViewportCallbacks{
		OnMacro: func() {},
		OnMeso:  func() {},
		OnMicro: func() {},
	}

	controls := NewViewportControls(theme, callbacks)

	// Update should not panic and should return false (no input consumed in test stub).
	consumed := controls.Update()
	assert.False(t, consumed)
}

func TestViewportControlsButtonLayout(t *testing.T) {
	theme := DefaultTheme()
	callbacks := ViewportCallbacks{}

	controls := NewViewportControls(theme, callbacks)

	// Test button dimensions are set.
	// Per AUDIT.md LOW finding: buttonHeight must be ≥44px for WCAG 2.5.5 touch targets.
	assert.Equal(t, 70, controls.buttonWidth)
	assert.Equal(t, 44, controls.buttonHeight)
	assert.Equal(t, 5, controls.buttonGap)
}
