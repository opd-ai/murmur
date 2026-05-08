// Package ui provides the Help button for reopening onboarding tutorials.
//
//go:build !test
// +build !test

package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// HelpButtonSize is the width and height of the help button in pixels.
const HelpButtonSize = 32

// HelpButton is a small "?" icon button that calls a callback when clicked.
// Clicking it reopens the onboarding tutorial hints so users can revisit
// guidance at any time.  Per PLAN.md: "Help button — reopen onboarding
// tutorials at any time".
type HelpButton struct {
	x, y    int
	visible bool
	hovered bool
	onClick func()
	theme   Theme
}

// NewHelpButton creates a help button at the given position.
// onClick is called when the button is activated (click or '?' key).
func NewHelpButton(x, y int, theme Theme, onClick func()) *HelpButton {
	return &HelpButton{
		x:       x,
		y:       y,
		visible: true,
		theme:   theme,
		onClick: onClick,
	}
}

// SetPosition updates the button's screen position.
func (h *HelpButton) SetPosition(x, y int) {
	h.x = x
	h.y = y
}

// SetVisible controls whether the button is drawn and interactive.
func (h *HelpButton) SetVisible(v bool) { h.visible = v }

// Update processes mouse and keyboard input.
func (h *HelpButton) Update() {
	if !h.visible {
		return
	}

	cx, cy := ebiten.CursorPosition()
	h.hovered = cx >= h.x && cx < h.x+HelpButtonSize && cy >= h.y && cy < h.y+HelpButtonSize

	// Click or '?' key activates the button.
	if (h.hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)) ||
		inpututil.IsKeyJustPressed(ebiten.KeySlash) {
		if h.onClick != nil {
			h.onClick()
		}
	}
}

// Draw renders the help button to screen.
func (h *HelpButton) Draw(screen *ebiten.Image) {
	if !h.visible {
		return
	}

	bg := h.theme.ButtonBackground
	if h.hovered {
		bg = h.theme.AccentPrimary
	}

	x, y := float32(h.x), float32(h.y)
	sz := float32(HelpButtonSize)

	// Background circle.
	vector.DrawFilledCircle(screen, x+sz/2, y+sz/2, sz/2, bg, true)

	// Border.
	vector.StrokeCircle(screen, x+sz/2, y+sz/2, sz/2, 1.5, h.theme.PanelBorder, true)

	// Draw "?" glyph using a simple pixel representation.
	drawQuestionMark(screen, h.x+HelpButtonSize/2, h.y+HelpButtonSize/2, h.theme.TextPrimary)
}

// drawQuestionMark draws a minimal "?" using small rectangles.
func drawQuestionMark(screen *ebiten.Image, cx, cy int, c color.Color) {
	// Stem (top arc approximated as short horizontal bar + vertical drop)
	// Top bar of question-mark arc.
	vector.DrawFilledRect(screen, float32(cx-4), float32(cy-10), 8, 2, c, false)
	// Right side of arc.
	vector.DrawFilledRect(screen, float32(cx+2), float32(cy-10), 2, 5, c, false)
	// Bottom-left curve connector.
	vector.DrawFilledRect(screen, float32(cx-2), float32(cy-5), 4, 2, c, false)
	// Stem drop.
	vector.DrawFilledRect(screen, float32(cx-1), float32(cy-4), 2, 4, c, false)
	// Dot.
	vector.DrawFilledRect(screen, float32(cx-1), float32(cy+2), 2, 2, c, false)
}
