// Package ui — shared font face and text-drawing helpers.
// All text rendering in pkg/ui goes through these helpers so that:
//   - The font face is allocated exactly once at program start (no per-frame allocs).
//   - Every panel uses a consistent typeface and colour-scale API.
//
// Font choice: basicfont.Face7x13 is a 7×13 px monospace bitmap font shipped
// with golang.org/x/image.  It is already an indirect dependency via
// Ebitengine's text/v2 package, so no new dependency is introduced.
// When a higher-quality font is embedded (e.g. via go:embed + text/v2
// GoTextFaceSource), replace the single `defaultFont` assignment here and
// every call site automatically benefits.

//go:build !test
// +build !test

package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// defaultFont is the shared font face for all UI text rendering in pkg/ui.
// Initialised from basicfont.Face7x13 so it is never nil, preventing the
// nil-pointer panic that occurred when text.Draw was called before the font
// was set up.
var defaultFont text.Face = text.NewGoXFace(basicfont.Face7x13)

// drawUIText draws str at (x, y) using defaultFont and the supplied color.
// x, y are the top-left origin of the text baseline.
func drawUIText(dst *ebiten.Image, str string, x, y float64, clr color.Color) {
	if str == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, str, defaultFont, op)
}

// drawUICenteredText draws str centered horizontally around cx, at baseline y.
func drawUICenteredText(dst *ebiten.Image, str string, cx, y float64, clr color.Color) {
	if str == "" {
		return
	}
	w, _ := text.Measure(str, defaultFont, 0)
	drawUIText(dst, str, cx-w/2, y, clr)
}

// DrawCenteredText is the exported equivalent of drawUICenteredText.
// It is intended for use by packages outside pkg/ui (e.g. pkg/pulsemap) that
// need to render text consistently with the shared font face.
func DrawCenteredText(dst *ebiten.Image, str string, cx, y float64, clr color.Color) {
	drawUICenteredText(dst, str, cx, y, clr)
}

// measureUIText returns the rendered width and height of str with defaultFont.
func measureUIText(str string) (float64, float64) {
	return text.Measure(str, defaultFont, 0)
}

// measureRuneAdvance returns the pixel advance width of the first n runes of s
// using the shared defaultFont. This is used for cursor positioning in text
// areas where fixed-width assumptions break for multi-byte characters.
// Per AUDIT.md MEDIUM fix: replaced fixed 7px/char with actual advance.
func measureRuneAdvance(s string, n int) int {
	runes := []rune(s)
	if n > len(runes) {
		n = len(runes)
	}
	w, _ := text.Measure(string(runes[:n]), defaultFont, 0)
	return int(w)
}

// truncateRunes truncates s to at most max runes, appending "…" if truncated.
// Unlike a byte-slice truncation (s[:n]), this never splits a multi-byte UTF-8
// rune and is safe for any Unicode input.
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
