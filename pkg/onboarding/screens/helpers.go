// Package screens provides shared rendering helpers for onboarding screens.

package screens

import (
	"bytes"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

// helperFaceSource is the shared scalable font source for onboarding helpers.
var helperFaceSource = mustNewHelperFaceSource()

func mustNewHelperFaceSource() *text.GoTextFaceSource {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	return src
}

func helperFaceForSize(size float64) text.Face {
	if size <= 0 {
		size = 14
	}
	return &text.GoTextFace{Source: helperFaceSource, Size: size}
}

// DrawCenteredText draws str centered horizontally around (x, y) using the
// shared scalable font source.
func DrawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	if str == "" {
		return
	}
	face := helperFaceForSize(size)
	w, _ := text.Measure(str, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)-w/2, float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op)
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

	drawCardGlow(screen, cardX, cardY, style, isSpecter)
	drawCardBackground(screen, cardX, cardY, style, isSpecter)
	drawCardLabel(screen, x, cardY, label, style, isSpecter)
	drawCardSigil(screen, x, cardY, sigil)
	drawCardName(screen, x, cardY, name, style, isSpecter)
}

// drawCardGlow renders the optional glow effect around the card.
func drawCardGlow(screen *ebiten.Image, cardX, cardY float32, style IdentityCardStyle, isSpecter bool) {
	if !style.GlowEnabled {
		return
	}
	glowRadius := float32(3 + 2*math.Sin(style.GlowPhase*2*math.Pi))
	glowColor := selectGlowColor(isSpecter)
	vector.DrawFilledRect(screen, cardX-glowRadius, cardY-glowRadius, style.CardWidth+2*glowRadius, style.CardHeight+2*glowRadius, glowColor, true)
}

// selectGlowColor returns the glow color based on identity type.
func selectGlowColor(isSpecter bool) color.RGBA {
	if isSpecter {
		return color.RGBA{80, 100, 200, 30}
	}
	return color.RGBA{200, 160, 80, 30}
}

// drawCardBackground renders the card background and border.
func drawCardBackground(screen *ebiten.Image, cardX, cardY float32, style IdentityCardStyle, isSpecter bool) {
	bgColor := selectBackgroundColor(isSpecter)
	vector.DrawFilledRect(screen, cardX, cardY, style.CardWidth, style.CardHeight, bgColor, true)

	borderColor := selectBorderColor(isSpecter)
	vector.StrokeRect(screen, cardX, cardY, style.CardWidth, style.CardHeight, 1.5, borderColor, true)
}

// selectBackgroundColor returns the card background color based on identity type.
func selectBackgroundColor(isSpecter bool) color.RGBA {
	if isSpecter {
		return color.RGBA{25, 30, 55, 255}
	}
	return color.RGBA{30, 35, 50, 255}
}

// selectBorderColor returns the card border color based on identity type.
func selectBorderColor(isSpecter bool) color.RGBA {
	if isSpecter {
		return color.RGBA{80, 100, 180, 255}
	}
	return color.RGBA{70, 80, 120, 255}
}

// drawCardLabel renders the card label text.
func drawCardLabel(screen *ebiten.Image, x, cardY float32, label string, style IdentityCardStyle, isSpecter bool) {
	labelY := cardY + 20
	labelColor := selectLabelColor(isSpecter)
	DrawCenteredText(screen, label, x, labelY, style.LabelSize, labelColor)
}

// selectLabelColor returns the label color based on identity type.
func selectLabelColor(isSpecter bool) color.RGBA {
	if isSpecter {
		return color.RGBA{120, 150, 200, 255}
	}
	return color.RGBA{150, 150, 160, 255}
}

// drawCardSigil renders the sigil image on the card.
func drawCardSigil(screen *ebiten.Image, x, cardY float32, sigil *ebiten.Image) {
	if sigil == nil {
		return
	}
	sigilOpts := &ebiten.DrawImageOptions{}
	sigilX := x - float32(sigil.Bounds().Dx())/2
	sigilY := cardY + 40
	sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
	screen.DrawImage(sigil, sigilOpts)
}

// drawCardName renders the identity name on the card.
func drawCardName(screen *ebiten.Image, x, cardY float32, name string, style IdentityCardStyle, isSpecter bool) {
	nameY := cardY + style.CardHeight - style.NameYOffset
	nameColor := selectNameColor(isSpecter)
	displayName := name
	if displayName == "" {
		displayName = style.EmptyNameStr
	}
	DrawCenteredText(screen, displayName, x, nameY, style.NameSize, nameColor)
}

// selectNameColor returns the name color based on identity type.
func selectNameColor(isSpecter bool) color.RGBA {
	if isSpecter {
		return color.RGBA{140, 170, 230, 255}
	}
	return color.RGBA{200, 200, 210, 255}
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

// SuccessAnimationStyle defines parameters for the success checkmark animation.
type SuccessAnimationStyle struct {
	BaseRadius  float32 // Base circle radius
	PulseAmount float32 // Amount to pulse (multiplied by sin(phase))
	CheckOffset float32 // Offset for checkmark positioning
	StrokeWidth float32 // Width of checkmark strokes
	CircleColor color.RGBA
	CheckColor  color.RGBA
}

// DefaultSuccessStyle returns the standard success animation style (smaller).
func DefaultSuccessStyle() SuccessAnimationStyle {
	return SuccessAnimationStyle{
		BaseRadius:  40,
		PulseAmount: 0.1,
		CheckOffset: 15,
		StrokeWidth: 3,
		CircleColor: color.RGBA{100, 200, 130, 255},
		CheckColor:  color.RGBA{240, 240, 245, 255},
	}
}

// LargeSuccessStyle returns a larger success animation style.
func LargeSuccessStyle() SuccessAnimationStyle {
	return SuccessAnimationStyle{
		BaseRadius:  50,
		PulseAmount: 0.1,
		CheckOffset: 20,
		StrokeWidth: 4,
		CircleColor: color.RGBA{100, 200, 130, 255},
		CheckColor:  color.RGBA{240, 240, 245, 255},
	}
}

// DrawSuccessAnimation renders an animated success circle with checkmark.
// The animPhase parameter (0-1) controls the pulse cycle.
func DrawSuccessAnimation(screen *ebiten.Image, x, y float32, animPhase float64, style SuccessAnimationStyle) {
	pulse := float32(1 + float64(style.PulseAmount)*math.Sin(animPhase*2*math.Pi))
	radius := style.BaseRadius * pulse

	// Success circle
	vector.DrawFilledCircle(screen, x, y, radius, style.CircleColor, true)

	// Checkmark - two strokes forming a check shape
	off := style.CheckOffset
	vector.StrokeLine(screen, x-off, y, x-off/3, y+off*0.8, style.StrokeWidth, style.CheckColor, true)
	vector.StrokeLine(screen, x-off/3, y+off*0.8, x+off, y-off*0.67, style.StrokeWidth, style.CheckColor, true)
}
