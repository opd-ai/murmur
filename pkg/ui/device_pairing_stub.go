// Package ui provides device pairing UI stubs for testing.

//go:build test
// +build test

package ui

import (
	"crypto/ed25519"
	"image"
)

// PairingState represents the state of a device pairing operation.
type PairingState int

const (
	PairingStateIdle PairingState = iota
	PairingStateGeneratingQR
	PairingStateWaitingForScan
	PairingStateConnecting
	PairingStateAuthorizing
	PairingStateComplete
	PairingStateError
)

// PairingToken contains the data encoded in the pairing QR code.
type PairingToken struct {
	IPAddress string
	Token     [32]byte
	ExpiresAt interface{}
	MasterKey ed25519.PublicKey
}

// Encode serializes the pairing token (stub).
func (pt *PairingToken) Encode() (string, error) {
	return "stub_token", nil
}

// DecodePairingToken deserializes a pairing token (stub).
func DecodePairingToken(encoded string) (*PairingToken, error) {
	return &PairingToken{}, nil
}

// DevicePairingPanel displays device pairing UI (stub).
type DevicePairingPanel struct {
	visible bool
	state   PairingState
}

// NewDevicePairingPanel creates a new device pairing panel (stub).
func NewDevicePairingPanel(theme Theme, onComplete func(ed25519.PublicKey, string) error, onCancel func()) *DevicePairingPanel {
	return &DevicePairingPanel{}
}

// Visible returns true if the panel is shown (stub).
func (p *DevicePairingPanel) Visible() bool {
	return p.visible
}

// Show displays the panel (stub).
func (p *DevicePairingPanel) Show(masterKey ed25519.PublicKey) error {
	p.visible = true
	p.state = PairingStateWaitingForScan
	return nil
}

// Hide hides the panel (stub).
func (p *DevicePairingPanel) Hide() {
	p.visible = false
}

// Update handles input (stub).
func (p *DevicePairingPanel) Update() bool {
	return p.visible
}

// Draw renders the panel (stub).
func (p *DevicePairingPanel) Draw(screen image.Image) {
	// Stub: no rendering
}

// SetState updates the pairing state (stub).
func (p *DevicePairingPanel) SetState(state PairingState, msg string) {
	p.state = state
}
