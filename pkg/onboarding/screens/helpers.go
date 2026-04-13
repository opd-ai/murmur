// Package screens provides shared rendering helpers for onboarding screens.
//
//go:build !noebiten
// +build !noebiten

package screens

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawCenteredText draws text centered at the given position.
// Note: This is a simplified placeholder. Production code should use text/v2 with a proper font.
func DrawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	// Approximate text width for centering (rough estimate)
	charWidth := float32(size) * 0.5
	textWidth := float32(len(str)) * charWidth
	textX := x - textWidth/2

	// Draw each character as a small rectangle (placeholder for actual text rendering)
	charColor := clr.(color.RGBA)
	charH := float32(size) * 0.8
	charW := charWidth * 0.8
	charY := y - charH/2

	for i, char := range str {
		if char == ' ' {
			continue
		}
		charX := textX + float32(i)*charWidth
		vector.DrawFilledRect(screen, charX, charY, charW, charH, charColor, true)
	}
}

// IdentityCardStyle defines style parameters for identity cards.
type IdentityCardStyle struct {
	CardWidth    float32
	CardHeight   float32
	LabelSize    float64
	NameSize     float64
	NameYOffset  float32 // Offset from card bottom
	GlowEnabled  bool
	GlowPhase    float64 // Animation phase for glow effect
	EmptyNameStr string  // Display string for empty name
}

// DefaultIdentityCardStyle returns the default card style (ModeScreen style).
func DefaultIdentityCardStyle() IdentityCardStyle {
	return IdentityCardStyle{
		CardWidth:    180,
		CardHeight:   160,
		LabelSize:    12,
		NameSize:     14,
		NameYOffset:  30,
		GlowEnabled:  false,
		EmptyNameStr: "(No name)",
	}
}

// CompletionIdentityCardStyle returns the style used by CompletionScreen.
func CompletionIdentityCardStyle(animPhase float64) IdentityCardStyle {
	return IdentityCardStyle{
		CardWidth:    200,
		CardHeight:   180,
		LabelSize:    11,
		NameSize:     14,
		NameYOffset:  35,
		GlowEnabled:  true,
		GlowPhase:    animPhase,
		EmptyNameStr: "(Anonymous)",
	}
}

// DrawIdentityCard renders an identity card with optional animation.
// This consolidates duplicate card drawing logic from CompletionScreen and ModeScreen.
func DrawIdentityCard(screen *ebiten.Image, x, y float32, label, name string, sigil *ebiten.Image, isSpecter bool, style IdentityCardStyle) {
	cardX := x - style.CardWidth/2
	cardY := y - style.CardHeight/2

	// Optional glow effect
	if style.GlowEnabled {
		glowRadius := float32(3 + 2*math.Sin(style.GlowPhase*2*math.Pi))
		glowColor := color.RGBA{80, 100, 200, 30}
		if !isSpecter {
			glowColor = color.RGBA{200, 160, 80, 30}
		}
		vector.DrawFilledRect(screen, cardX-glowRadius, cardY-glowRadius, style.CardWidth+2*glowRadius, style.CardHeight+2*glowRadius, glowColor, true)
	}

	// Card background
	bgColor := color.RGBA{30, 35, 50, 255}
	if isSpecter {
		bgColor = color.RGBA{25, 30, 55, 255}
	}
	vector.DrawFilledRect(screen, cardX, cardY, style.CardWidth, style.CardHeight, bgColor, true)

	// Border
	borderColor := color.RGBA{70, 80, 120, 255}
	if isSpecter {
		borderColor = color.RGBA{80, 100, 180, 255}
	}
	vector.StrokeRect(screen, cardX, cardY, style.CardWidth, style.CardHeight, 1.5, borderColor, true)

	// Label
	labelY := cardY + 20
	labelColor := color.RGBA{150, 150, 160, 255}
	if isSpecter {
		labelColor = color.RGBA{120, 150, 200, 255}
	}
	DrawCenteredText(screen, label, x, labelY, style.LabelSize, labelColor)

	// Sigil
	if sigil != nil {
		sigilOpts := &ebiten.DrawImageOptions{}
		sigilX := x - float32(sigil.Bounds().Dx())/2
		sigilY := cardY + 40
		sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
		screen.DrawImage(sigil, sigilOpts)
	}

	// Name
	nameY := cardY + style.CardHeight - style.NameYOffset
	nameColor := color.RGBA{200, 200, 210, 255}
	if isSpecter {
		nameColor = color.RGBA{140, 170, 230, 255}
	}
	displayName := name
	if displayName == "" {
		displayName = style.EmptyNameStr
	}
	DrawCenteredText(screen, displayName, x, nameY, style.NameSize, nameColor)
}

// ButtonStyle defines visual parameters for buttons.
type ButtonStyle struct {
	Width     float32
	Height    float32
	TextSize  float64
	TextColor color.RGBA
}

// DefaultButtonStyle returns a standard button style.
func DefaultButtonStyle() ButtonStyle {
	return ButtonStyle{
		Width:     160,
		Height:    40,
		TextSize:  14,
		TextColor: color.RGBA{220, 220, 230, 255},
	}
}

// DrawButton renders a button at the given position.
func DrawButton(screen *ebiten.Image, label string, x, y float32, style ButtonStyle) {
	bx := x - style.Width/2
	by := y - style.Height/2

	bgColor := color.RGBA{60, 70, 100, 255}
	vector.DrawFilledRect(screen, bx, by, style.Width, style.Height, bgColor, true)

	borderColor := color.RGBA{100, 120, 180, 255}
	vector.StrokeRect(screen, bx, by, style.Width, style.Height, 1.5, borderColor, true)

	DrawCenteredText(screen, label, x, y+6, style.TextSize, style.TextColor)
}
