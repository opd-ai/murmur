// Package ui provides UI components for MURMUR.
// This file implements viewport control buttons for zoom presets.
//
//go:build !test
// +build !test

package ui

import (
	"image/color"

	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ViewportControls provides buttons for Macro/Meso/Micro preset zoom levels.
// Per ROADMAP.md line 682, this enables quick navigation between zoom presets.
type ViewportControls struct {
	theme Theme

	// Callbacks for zoom preset activation.
	onMacro func()
	onMeso  func()
	onMicro func()

	// Button dimensions and spacing.
	buttonWidth  int
	buttonHeight int
	buttonGap    int

	// Button positions (computed in Update based on screen size).
	macroX, macroY int
	mesoX, mesoY   int
	microX, microY int

	// Screen dimensions.
	screenWidth  int
	screenHeight int

	// Hover state for visual feedback.
	hoverMacro bool
	hoverMeso  bool
	hoverMicro bool
}

// ViewportCallbacks contains callback functions for viewport control actions.
type ViewportCallbacks struct {
	OnMacro func() // Called when Macro button is clicked
	OnMeso  func() // Called when Meso button is clicked
	OnMicro func() // Called when Micro button is clicked
}

// NewViewportControls creates a new viewport control widget.
func NewViewportControls(theme Theme, callbacks ViewportCallbacks) *ViewportControls {
	return &ViewportControls{
		theme:        theme,
		onMacro:      callbacks.OnMacro,
		onMeso:       callbacks.OnMeso,
		onMicro:      callbacks.OnMicro,
		buttonWidth:  70,
		buttonHeight: 30,
		buttonGap:    5,
	}
}

// Update handles input for viewport controls.
// Returns true if input was consumed.
func (v *ViewportControls) Update() bool {
	w, h := ebiten.WindowSize()
	v.screenWidth = w
	v.screenHeight = h

	// Position buttons in top-right corner with padding from edge.
	padding := v.theme.Padding
	startY := padding

	// Stack buttons vertically in top-right.
	v.macroX = w - padding - v.buttonWidth
	v.macroY = startY

	v.mesoX = w - padding - v.buttonWidth
	v.mesoY = startY + v.buttonHeight + v.buttonGap

	v.microX = w - padding - v.buttonWidth
	v.microY = startY + 2*(v.buttonHeight+v.buttonGap)

	// Check mouse position for hover state.
	mx, my := ebiten.CursorPosition()
	v.hoverMacro = v.isInButton(mx, my, v.macroX, v.macroY)
	v.hoverMeso = v.isInButton(mx, my, v.mesoX, v.mesoY)
	v.hoverMicro = v.isInButton(mx, my, v.microX, v.microY)

	// Handle clicks.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if v.hoverMacro && v.onMacro != nil {
			v.onMacro()
			return true
		}
		if v.hoverMeso && v.onMeso != nil {
			v.onMeso()
			return true
		}
		if v.hoverMicro && v.onMicro != nil {
			v.onMicro()
			return true
		}
	}

	return false
}

// Draw renders the viewport controls.
func (v *ViewportControls) Draw(screen *ebiten.Image) {
	v.drawButton(screen, v.macroX, v.macroY, "Macro", v.hoverMacro)
	v.drawButton(screen, v.mesoX, v.mesoY, "Meso", v.hoverMeso)
	v.drawButton(screen, v.microX, v.microY, "Micro", v.hoverMicro)
}

// drawButton draws a single viewport control button.
func (v *ViewportControls) drawButton(screen *ebiten.Image, x, y int, label string, hover bool) {
	// Choose background color based on hover state.
	bg := v.theme.ButtonBackground
	if hover {
		bg = v.theme.AccentPrimary
	}

	DrawButton(screen, x, y, v.buttonWidth, v.buttonHeight, bg, v.theme.PanelBorder, true)

	// Draw label centered in button using the shared defaultFont helper.
	textColor := v.theme.TextPrimary
	if hover {
		textColor = color.RGBA{255, 255, 255, 255} // White text on hover
	}

	cx := float64(x) + float64(v.buttonWidth)/2
	ty := float64(y) + float64(v.buttonHeight)/2 - 4
	drawUICenteredText(screen, label, cx, ty, textColor)
}

// isInButton checks if the given point is inside a button.
func (v *ViewportControls) isInButton(mx, my, bx, by int) bool {
	return mx >= bx && mx <= bx+v.buttonWidth &&
		my >= by && my <= by+v.buttonHeight
}
