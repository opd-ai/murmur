// Package screens provides the Returning User screen stub for testing.

//go:build test
// +build test

package screens

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// ReturningScreen stub for testing.
type ReturningScreen struct {
	updateCount int
	callback    func()
	called      bool
}

// NewReturningScreen creates a stub returning user screen.
func NewReturningScreen(
	displayName string,
	keypair *keys.KeyPair,
	onContinue func(),
) *ReturningScreen {
	return &ReturningScreen{
		callback: onContinue,
	}
}

// Layout implements ebiten.Game Layout.
func (r *ReturningScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// Update implements ebiten.Game Update.
func (r *ReturningScreen) Update() error {
	r.updateCount++
	if r.updateCount > 120 && !r.called { // 2 seconds at 60fps
		if r.callback != nil {
			r.called = true
			r.callback()
		}
	}
	return nil
}

// Draw implements ebiten.Game Draw.
func (r *ReturningScreen) Draw(screen *ebiten.Image) {
	// No-op stub
}
