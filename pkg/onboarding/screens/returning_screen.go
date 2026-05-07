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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	continued         bool
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

	if !r.continued && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)) {
		r.continued = true
		if r.callback != nil {
			r.callback()
		}
		return ebiten.Termination
	}

	return nil
}

// Draw implements ebiten.Game Draw.
func (r *ReturningScreen) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 20, 28, 255})

	centerX := float32(r.width) / 2
	centerY := float32(r.height) / 2
	pulse := float32(0.5 + 0.3*math.Sin(r.animPhase*3.0))

	r.drawCentralNode(screen, centerX, centerY, pulse)
	r.drawWelcomeText(screen, centerX, centerY)
	r.drawIdentityInfo(screen, centerX, centerY)
}

// drawCentralNode draws the pulsing central node.
func (r *ReturningScreen) drawCentralNode(screen *ebiten.Image, centerX, centerY, pulse float32) {
	nodeSize := 40.0 + 20.0*pulse
	nodeY := centerY - 80

	vector.DrawFilledCircle(
		screen, centerX, nodeY, nodeSize,
		color.RGBA{100, 160, 220, uint8(200 * pulse)},
		false,
	)
	vector.StrokeCircle(
		screen, centerX, nodeY, nodeSize+10, 2,
		color.RGBA{100, 160, 220, uint8(150 * pulse)},
		false,
	)
}

// drawWelcomeText draws the "Welcome back" text with fade-in.
func (r *ReturningScreen) drawWelcomeText(screen *ebiten.Image, centerX, centerY float32) {
	fade := 1.0
	if r.animPhase < 0.5 {
		fade = r.animPhase / 0.5
	}

	DrawCenteredText(
		screen, "Welcome back", centerX, centerY, 32,
		color.RGBA{255, 255, 255, uint8(255 * fade)},
	)
}

// drawIdentityInfo draws the identity information and connection status.
func (r *ReturningScreen) drawIdentityInfo(screen *ebiten.Image, centerX, centerY float32) {
	if r.animPhase <= 0.3 {
		return
	}

	infoFade := (r.animPhase - 0.3) / 0.7
	if infoFade > 1.0 {
		infoFade = 1.0
	}

	identityText := r.displayName
	if identityText == "" {
		identityText = r.pubKeyFingerprint
	}

	DrawCenteredText(
		screen, identityText, centerX, centerY+50, 20,
		color.RGBA{180, 180, 200, uint8(200 * infoFade)},
	)

	DrawCenteredText(
		screen, "Connecting to network...", centerX, centerY+80, 16,
		color.RGBA{140, 140, 160, uint8(150 * infoFade)},
	)

	DrawCenteredText(
		screen, "Press Enter or click to continue", centerX, centerY+120, 13,
		color.RGBA{170, 170, 185, uint8(170 * infoFade)},
	)
}
