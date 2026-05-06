// Package ui provides key rotation wizard UI stub for testing.

//go:build test
// +build test

package ui

import (
	"crypto/ed25519"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/proto"
)

// RotationState represents the state of key rotation process.
type RotationState int

const (
	RotationStateConfirm RotationState = iota
	RotationStateGeneratingKey
	RotationStateConfigureGracePeriod
	RotationStateCreatingDeclaration
	RotationStatePropagating
	RotationStateComplete
	RotationStateError
)

// KeyRotationWizard stub for testing.
type KeyRotationWizard struct {
	visible bool
}

// NewKeyRotationWizard creates a stub wizard.
func NewKeyRotationWizard(
	oldPrivateKey ed25519.PrivateKey,
	oldPublicKey ed25519.PublicKey,
	theme *Theme,
) *KeyRotationWizard {
	return &KeyRotationWizard{}
}

// SetOnComplete is a stub.
func (w *KeyRotationWizard) SetOnComplete(callback func(*proto.ContinuityDeclaration)) {}

// SetOnCancel is a stub.
func (w *KeyRotationWizard) SetOnCancel(callback func()) {}

// Show is a stub.
func (w *KeyRotationWizard) Show() {
	w.visible = true
}

// Hide is a stub.
func (w *KeyRotationWizard) Hide() {
	w.visible = false
}

// IsVisible returns visibility state.
func (w *KeyRotationWizard) IsVisible() bool {
	return w.visible
}

// Update is a stub.
func (w *KeyRotationWizard) Update() bool {
	return w.visible
}

// Draw is a stub.
func (w *KeyRotationWizard) Draw(screen *ebiten.Image) {}
