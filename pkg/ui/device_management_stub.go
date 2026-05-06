// Package ui provides device management UI stubs for testing.

//go:build test
// +build test

package ui

import (
	"crypto/ed25519"
	"image"
	"time"
)

// DeviceInfo represents an authorized device (stub).
type DeviceInfo struct {
	PublicKey       ed25519.PublicKey
	Label           string
	AuthorizedAt    time.Time
	ExpiresAt       time.Time
	IsCurrentDevice bool
}

// DeviceManagementPanel displays authorized devices (stub).
type DeviceManagementPanel struct {
	visible       bool
	confirmRevoke bool
}

// NewDeviceManagementPanel creates a new device management panel (stub).
func NewDeviceManagementPanel(theme Theme, onRevoke func(ed25519.PublicKey) error, onAddDevice, onClose func()) *DeviceManagementPanel {
	return &DeviceManagementPanel{}
}

// Visible returns true if the panel is shown (stub).
func (p *DeviceManagementPanel) Visible() bool {
	return p.visible
}

// Show displays the panel (stub).
func (p *DeviceManagementPanel) Show(devices []DeviceInfo) {
	p.visible = true
}

// Hide hides the panel (stub).
func (p *DeviceManagementPanel) Hide() {
	p.visible = false
}

// Update handles input (stub).
func (p *DeviceManagementPanel) Update() bool {
	return p.visible
}

// Draw renders the panel (stub).
func (p *DeviceManagementPanel) Draw(screen image.Image) {
	// Stub: no rendering
}

// RequestRevoke initiates revocation (stub).
func (p *DeviceManagementPanel) RequestRevoke(deviceIndex int) {
	p.confirmRevoke = true
}

// ConfirmRevoke executes revocation (stub).
func (p *DeviceManagementPanel) ConfirmRevoke() error {
	p.confirmRevoke = false
	return nil
}

// SetError sets an error message (stub).
func (p *DeviceManagementPanel) SetError(msg string) {
	// Stub: no error display
}
