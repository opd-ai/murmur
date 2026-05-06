package ui

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"
)

func TestDeviceManagementPanel_Creation(t *testing.T) {
	theme := DefaultTheme()

	onRevoke := func(pubkey ed25519.PublicKey) error {
		return nil
	}

	onAddDevice := func() {
	}

	onClose := func() {
	}

	panel := NewDeviceManagementPanel(theme, onRevoke, onAddDevice, onClose)
	if panel == nil {
		t.Fatal("Expected panel to be created")
	}

	if panel.Visible() {
		t.Error("Panel should not be visible by default")
	}
}

func TestDeviceManagementPanel_ShowHide(t *testing.T) {
	theme := DefaultTheme()
	panel := NewDeviceManagementPanel(theme, nil, nil, nil)

	// Create test devices
	pubkey1, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	pubkey2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	devices := []DeviceInfo{
		{
			PublicKey:       pubkey1,
			Label:           "Phone A",
			AuthorizedAt:    time.Now().Add(-24 * time.Hour),
			IsCurrentDevice: true,
		},
		{
			PublicKey:    pubkey2,
			Label:        "Laptop B",
			AuthorizedAt: time.Now().Add(-7 * 24 * time.Hour),
		},
	}

	// Show panel
	panel.Show(devices)

	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	// Hide panel
	panel.Hide()

	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestDeviceManagementPanel_RevocationFlow(t *testing.T) {
	theme := DefaultTheme()

	revokeCalled := false
	var revokedKey ed25519.PublicKey

	onRevoke := func(pubkey ed25519.PublicKey) error {
		revokeCalled = true
		revokedKey = pubkey
		return nil
	}

	panel := NewDeviceManagementPanel(theme, onRevoke, nil, nil)

	// Create test device
	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	devices := []DeviceInfo{
		{
			PublicKey:    pubkey,
			Label:        "Old Phone",
			AuthorizedAt: time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	panel.Show(devices)

	// Request revoke
	panel.RequestRevoke(0)

	// Confirm revoke
	if err := panel.ConfirmRevoke(); err != nil {
		t.Fatalf("Failed to confirm revoke: %v", err)
	}

	if !revokeCalled {
		t.Error("onRevoke callback should have been called")
	}

	if len(revokedKey) != ed25519.PublicKeySize {
		t.Error("Revoked key should have correct size")
	}
}

func TestDeviceManagementPanel_RevocationError(t *testing.T) {
	theme := DefaultTheme()

	onRevoke := func(pubkey ed25519.PublicKey) error {
		return errors.New("revocation failed")
	}

	panel := NewDeviceManagementPanel(theme, onRevoke, nil, nil)

	// Create test device
	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	devices := []DeviceInfo{
		{
			PublicKey:    pubkey,
			Label:        "Test Device",
			AuthorizedAt: time.Now(),
		},
	}

	panel.Show(devices)
	panel.RequestRevoke(0)

	// Confirm revoke (should return error)
	if err := panel.ConfirmRevoke(); err == nil {
		t.Error("Expected error from ConfirmRevoke")
	}
}

func TestDeviceManagementPanel_EmptyList(t *testing.T) {
	theme := DefaultTheme()
	panel := NewDeviceManagementPanel(theme, nil, nil, nil)

	// Show with empty device list
	panel.Show([]DeviceInfo{})

	if !panel.Visible() {
		t.Error("Panel should be visible even with empty list")
	}
}

func TestDeviceManagementPanel_CurrentDeviceProtection(t *testing.T) {
	theme := DefaultTheme()

	onRevoke := func(pubkey ed25519.PublicKey) error {
		return nil
	}

	panel := NewDeviceManagementPanel(theme, onRevoke, nil, nil)

	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	devices := []DeviceInfo{
		{
			PublicKey:       pubkey,
			Label:           "This Device",
			AuthorizedAt:    time.Now(),
			IsCurrentDevice: true,
		},
	}

	panel.Show(devices)

	// In the real UI, revoke button wouldn't be shown for current device,
	// but we test the RequestRevoke logic anyway
	panel.RequestRevoke(0)

	// The UI should handle this gracefully (not crash)
	// In production, the button wouldn't exist for current device
}
