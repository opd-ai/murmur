// Package screens provides the Returning User screen.
// Per ROADMAP.md line 776, this screen provides fast bootstrap for existing identity.

//go:build !test
// +build !test

package screens

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// ReturningScreen handles returning user experience.
type ReturningScreen struct {
	startTime         time.Time
	animPhase         float64
	displayName       string
	pubKeyFingerprint string
	width, height     int
	callback          func()
}

// NewReturningScreen creates a returning user screen.
func NewReturningScreen(
	displayName string,
	keypair *keys.KeyPair,
	onContinue func(),
) *ReturningScreen {
	fingerprint := fmt.Sprintf("%x", keypair.PublicKey[:8])
	return &ReturningScreen{
		startTime:         time.Now(),
		displayName:       displayName,
		pubKeyFingerprint: fingerprint,
		callback:          onContinue,
	}
}

// Layout implements ebiten.Game Layout.
func (r *ReturningScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	r.width = outsideWidth
	r.height = outsideHeight
	return outsideWidth, outsideHeight
}

// Update implements ebiten.Game Update.
func (r *ReturningScreen) Update() error {
	elapsed := time.Since(r.startTime).Seconds()
	r.animPhase = elapsed

	// Auto-continue after 2 seconds
	if elapsed > 2.0 {
		if r.callback != nil {
			r.callback()
		}
	}

	return nil
}

// Draw implements ebiten.Game Draw.
func (r *ReturningScreen) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 20, 28, 255}) // Dark background

	centerX := float32(r.width) / 2
	centerY := float32(r.height) / 2

	// Pulse animation
	pulse := float32(0.5 + 0.3*math.Sin(r.animPhase*3.0))

	// Draw central glowing node
	nodeSize := 40.0 + 20.0*pulse
	vector.DrawFilledCircle(
		screen,
		centerX,
		centerY-80,
		nodeSize,
		color.RGBA{100, 160, 220, uint8(200 * pulse)},
		false,
	)
	vector.StrokeCircle(
		screen,
		centerX,
		centerY-80,
		nodeSize+10,
		2,
		color.RGBA{100, 160, 220, uint8(150 * pulse)},
		false,
	)

	// Welcome back text (fade in)
	fade := float64(1.0)
	if r.animPhase < 0.5 {
		fade = r.animPhase / 0.5
	}

	DrawCenteredText(
		screen,
		"Welcome back",
		float32(r.width)/2,
		float32(r.height)/2,
		32,
		color.RGBA{255, 255, 255, uint8(255 * fade)},
	)

	// Identity info
	if r.animPhase > 0.3 {
		infoFade := (r.animPhase - 0.3) / 0.7
		if infoFade > 1.0 {
			infoFade = 1.0
		}

		identityText := r.displayName
		if identityText == "" {
			identityText = r.pubKeyFingerprint
		}

		DrawCenteredText(
			screen,
			identityText,
			float32(r.width)/2,
			float32(r.height)/2+50,
			20,
			color.RGBA{180, 180, 200, uint8(200 * infoFade)},
		)

		DrawCenteredText(
			screen,
			"Connecting to network...",
			float32(r.width)/2,
			float32(r.height)/2+80,
			16,
			color.RGBA{140, 140, 160, uint8(150 * infoFade)},
		)
	}
}
