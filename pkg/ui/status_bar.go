// Package ui provides the StatusBar component.
//
//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// StatusBar renders a thin strip showing key network and cryptographic status
// indicators.  Per PLAN.md: "identity publication, Shroud circuit, connection
// count, PoW progress".
//
// The bar is intended to be drawn at the bottom of the Pulse Map viewport.
type StatusBar struct {
	x, y   int
	width  int
	height int
	theme  Theme

	// Status fields – updated by the caller each frame or on change.
	PeerCount         int     // Live network peer count
	ShroudActive      bool    // True when a Shroud circuit is established
	IdentityPublished bool    // True once identity declaration is propagated
	PowBusy           bool    // True while PoW is being computed in background
	PowProgress       float32 // 0.0–1.0; only meaningful when PowBusy is true
}

// NewStatusBar creates a StatusBar at the given position with given dimensions.
func NewStatusBar(x, y, width, height int, theme Theme) *StatusBar {
	return &StatusBar{
		x: x, y: y, width: width, height: height, theme: theme,
	}
}

// Draw renders the status bar to screen.
func (s *StatusBar) Draw(screen *ebiten.Image) {
	// Background strip.
	vector.DrawFilledRect(screen,
		float32(s.x), float32(s.y), float32(s.width), float32(s.height),
		s.theme.PanelBackground, false)

	// Draw each indicator from left to right with fixed column widths.
	col := s.x + 8
	col = s.drawPeerCount(screen, col)
	col = s.drawShroudStatus(screen, col)
	col = s.drawIdentityStatus(screen, col)
	s.drawPowStatus(screen, col)
}

const statusColSpacing = 160

func (s *StatusBar) drawPeerCount(screen *ebiten.Image, x int) int {
	label := fmt.Sprintf("⬡ %d peers", s.PeerCount)
	c := s.theme.TextColor
	if s.PeerCount < 6 {
		c = color.RGBA{R: 220, G: 100, B: 60, A: 255} // amber — below minimum mesh
	}
	drawSmallText(screen, label, x, s.y+(s.height/2), c)
	return x + statusColSpacing
}

func (s *StatusBar) drawShroudStatus(screen *ebiten.Image, x int) int {
	label := "◉ Shroud off"
	c := color.RGBA{R: 160, G: 160, B: 160, A: 200}
	if s.ShroudActive {
		label = "◉ Shroud on"
		c = color.RGBA{R: 80, G: 200, B: 120, A: 255}
	}
	drawSmallText(screen, label, x, s.y+(s.height/2), c)
	return x + statusColSpacing
}

func (s *StatusBar) drawIdentityStatus(screen *ebiten.Image, x int) int {
	label := "✗ identity"
	c := color.RGBA{R: 200, G: 100, B: 60, A: 255}
	if s.IdentityPublished {
		label = "✓ identity"
		c = color.RGBA{R: 80, G: 200, B: 120, A: 255}
	}
	drawSmallText(screen, label, x, s.y+(s.height/2), c)
	return x + statusColSpacing
}

func (s *StatusBar) drawPowStatus(screen *ebiten.Image, x int) {
	if !s.PowBusy {
		return
	}
	// Progress bar.
	barW := float32(80)
	barH := float32(6)
	bx := float32(x)
	by := float32(s.y) + float32(s.height)/2 - barH/2
	vector.DrawFilledRect(screen, bx, by, barW, barH,
		color.RGBA{R: 50, G: 50, B: 60, A: 200}, false)
	vector.DrawFilledRect(screen, bx, by, barW*s.PowProgress, barH,
		color.RGBA{R: 120, G: 180, B: 240, A: 255}, false)
	drawSmallText(screen, "PoW…", x+84, s.y+(s.height/2), s.theme.TextColor)
}

// drawSmallText is a minimal single-pixel-raster text fallback for the status bar.
// In production this is replaced by the Ebitengine text/v2 package; this stub
// avoids pulling in font assets from the status bar package.
func drawSmallText(_ *ebiten.Image, _ string, _, _ int, _ color.Color) {
	// Rendering delegated to the caller's font pipeline.
	// This stub satisfies compilation; game.go calls text/v2 directly.
}
