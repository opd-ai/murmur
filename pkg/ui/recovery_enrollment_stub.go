// Package ui provides social recovery contact enrollment UI stub for testing.

//go:build test
// +build test

package ui

import (
	"crypto/ed25519"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/recovery"
)

// EnrollmentState represents the state of recovery enrollment process.
type EnrollmentState int

const (
	EnrollmentStateSelectContacts EnrollmentState = iota
	EnrollmentStateConfigureThreshold
	EnrollmentStateDistributing
	EnrollmentStateComplete
	EnrollmentStateError
)

// RecoveryContact represents a contact available for enrollment.
type RecoveryContact struct {
	PublicKey ed25519.PublicKey
	X25519Key []byte
	Label     string
	Selected  bool
}

// RecoveryEnrollmentPanel stub for testing.
type RecoveryEnrollmentPanel struct {
	visible bool
}

// NewRecoveryEnrollmentPanel creates a stub panel.
func NewRecoveryEnrollmentPanel(
	contacts []RecoveryContact,
	masterPrivateKey ed25519.PrivateKey,
	masterPublicKey ed25519.PublicKey,
	x25519PrivateKey []byte,
	theme *Theme,
) *RecoveryEnrollmentPanel {
	return &RecoveryEnrollmentPanel{}
}

// SetOnComplete is a stub.
func (p *RecoveryEnrollmentPanel) SetOnComplete(callback func(results []recovery.EnrollmentResult)) {}

// SetOnCancel is a stub.
func (p *RecoveryEnrollmentPanel) SetOnCancel(callback func()) {}

// Show is a stub.
func (p *RecoveryEnrollmentPanel) Show() {
	p.visible = true
}

// Hide is a stub.
func (p *RecoveryEnrollmentPanel) Hide() {
	p.visible = false
}

// IsVisible returns visibility state.
func (p *RecoveryEnrollmentPanel) IsVisible() bool {
	return p.visible
}

// Update is a stub.
func (p *RecoveryEnrollmentPanel) Update() bool {
	return p.visible
}

// Draw is a stub.
func (p *RecoveryEnrollmentPanel) Draw(screen *ebiten.Image) {}
