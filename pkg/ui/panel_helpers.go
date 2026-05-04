// Package ui provides shared helpers for panel rendering.
//
//go:build !test
// +build !test

package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PanelDrawContext holds common panel drawing state.
type PanelDrawContext struct {
	ScreenWidth  int
	ScreenHeight int
	PanelX       int
	PanelY       int
}

// InitPanelDraw performs common Draw() initialization and returns nil if the panel should not render.
// This consolidates the duplicate pattern:
//   - Lock mutex (caller must defer unlock)
//   - Check visibility
//   - Get screen dimensions
//   - Calculate panel position
//
// Returns PanelDrawContext with computed values, or nil if panel is not visible.
func InitPanelDraw(screen *ebiten.Image, visible bool, calculatePos func(w, h int) (int, int)) *PanelDrawContext {
	if !visible {
		return nil
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	px, py := calculatePos(w, h)

	return &PanelDrawContext{
		ScreenWidth:  w,
		ScreenHeight: h,
		PanelX:       px,
		PanelY:       py,
	}
}

// InitPanelDrawWithScreen is a convenience wrapper that also updates screen dimensions.
// Consolidates the common pattern:
//
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//	ctx := InitPanelDraw(screen, p.visible, p.calculatePosition)
//	if ctx == nil { return }
//	p.screenWidth = ctx.ScreenWidth
//	p.screenHeight = ctx.ScreenHeight
//
// Returns PanelDrawContext or nil if not visible.
func InitPanelDrawWithScreen(screen *ebiten.Image, visible bool, calculatePos func(w, h int) (int, int), screenWidth, screenHeight *int) *PanelDrawContext {
	ctx := InitPanelDraw(screen, visible, calculatePos)
	if ctx == nil {
		return nil
	}
	*screenWidth = ctx.ScreenWidth
	*screenHeight = ctx.ScreenHeight
	return ctx
}

// DrawButton draws a filled button with optional border at the specified position.
func DrawButton(screen *ebiten.Image, x, y, width, height int, bgColor, borderColor color.Color, withBorder bool) {
	vector.DrawFilledRect(screen, float32(x), float32(y),
		float32(width), float32(height), bgColor, true)
	if withBorder {
		vector.StrokeRect(screen, float32(x), float32(y),
			float32(width), float32(height), 1.0, borderColor, true)
	}
}

// DrawCancelSubmitButtons draws standard cancel (left) and submit (right) buttons.
// The submit button background changes based on the enabled parameter.
func DrawCancelSubmitButtons(screen *ebiten.Image, px, py, panelWidth, panelHeight int, theme Theme, submitWidth int, submitLabel string, enabled bool) (cancelX, submitX, buttonY int) {
	buttonY = py + panelHeight - theme.Padding - theme.ButtonHeight

	// Cancel button (80px wide, left side).
	cancelX = px + theme.Padding
	cancelW := 80
	DrawButton(screen, cancelX, buttonY, cancelW, theme.ButtonHeight, theme.ButtonBackground, theme.PanelBorder, true)

	// Submit button (right side).
	submitX = px + panelWidth - theme.Padding - submitWidth
	submitBg := theme.AccentPrimary
	if !enabled {
		submitBg = theme.ButtonBackground
	}
	DrawButton(screen, submitX, buttonY, submitWidth, theme.ButtonHeight, submitBg, theme.PanelBorder, false)

	return cancelX, submitX, buttonY
}
