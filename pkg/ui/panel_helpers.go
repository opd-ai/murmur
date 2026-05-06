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

// CheckPanelVisibilityAndCenter checks if a panel is visible and calculates centered position.
// Returns (px, py, w, h, shouldRender) where shouldRender=false means caller should return early.
// Consolidates the common pattern:
//
//	if !p.visible {
//	    return
//	}
//	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
//	px := (w - p.width) / 2
//	py := (h - p.height) / 2
func CheckPanelVisibilityAndCenter(screen *ebiten.Image, visible bool, panelWidth, panelHeight int) (px, py, w, h int, shouldRender bool) {
	if !visible {
		return 0, 0, 0, 0, false
	}
	w, h = screen.Bounds().Dx(), screen.Bounds().Dy()
	px = (w - panelWidth) / 2
	py = (h - panelHeight) / 2
	return px, py, w, h, true
}

// DrawModalOverlayAndPanel draws a semi-transparent overlay and centered panel.
// Consolidates the pattern:
//
//	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), theme.PanelBackground, true)
//	vector.DrawFilledRect(screen, float32(px), float32(py), float32(width), float32(height), theme.PanelBackground, true)
//	vector.StrokeRect(screen, float32(px), float32(py), float32(width), float32(height), 2.0, theme.PanelBorder, true)
func DrawModalOverlayAndPanel(screen *ebiten.Image, px, py, w, h, panelWidth, panelHeight int, theme Theme) {
	// Draw full-screen overlay
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), theme.PanelBackground, true)

	// Draw panel background and border
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(panelWidth), float32(panelHeight), theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(panelWidth), float32(panelHeight), 2.0, theme.PanelBorder, true)
}
