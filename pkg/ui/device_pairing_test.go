package ui

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestDevicePairingPanel_Creation(t *testing.T) {
	theme := DefaultTheme()

	onComplete := func(pubkey ed25519.PublicKey, label string) error {
		return nil
	}

	onCancel := func() {
	}

	panel := NewDevicePairingPanel(theme, onComplete, onCancel)
	if panel == nil {
		t.Fatal("Expected panel to be created")
	}

	if panel.Visible() {
		t.Error("Panel should not be visible by default")
	}
}

func TestPairingToken_Encoding(t *testing.T) {
	// Generate a test master key
	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	token := &PairingToken{
		IPAddress: "192.168.1.1",
		Token:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		MasterKey: pubkey,
	}

	// Encode
	encoded, err := token.Encode()
	if err != nil {
		t.Fatalf("Failed to encode token: %v", err)
	}

	if encoded == "" {
		t.Error("Encoded token should not be empty")
	}

	// Decode
	decoded, err := DecodePairingToken(encoded)
	if err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	// Verify fields
	if decoded.IPAddress != token.IPAddress {
		t.Errorf("IP address mismatch: got %s, want %s", decoded.IPAddress, token.IPAddress)
	}

	if decoded.Token != token.Token {
		t.Error("Token mismatch")
	}

	if len(decoded.MasterKey) != ed25519.PublicKeySize {
		t.Errorf("Master key size mismatch: got %d, want %d", len(decoded.MasterKey), ed25519.PublicKeySize)
	}
}

func TestDevicePairingPanel_ShowHide(t *testing.T) {
	theme := DefaultTheme()
	panel := NewDevicePairingPanel(theme, nil, nil)

	// Generate a test master key
	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Show panel
	if err := panel.Show(pubkey); err != nil {
		t.Fatalf("Failed to show panel: %v", err)
	}

	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	// Hide panel
	panel.Hide()

	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestDevicePairingPanel_StateTransitions(t *testing.T) {
	theme := DefaultTheme()
	panel := NewDevicePairingPanel(theme, nil, nil)

	// Test state changes
	panel.SetState(PairingStateConnecting, "Connecting...")
	panel.SetState(PairingStateAuthorizing, "Authorizing...")
	panel.SetState(PairingStateComplete, "Complete!")
	panel.SetState(PairingStateError, "Error occurred")
}

func TestSettingsPanel_DevicesCategory(t *testing.T) {
	theme := DefaultTheme()
	panel := NewSettingsPanel(theme, nil)

	categories := panel.Categories()

	// Find Devices category
	var devicesCategory *SettingCategory
	for i := range categories {
		if categories[i].Name == "Devices" {
			devicesCategory = &categories[i]
			break
		}
	}

	if devicesCategory == nil {
		t.Fatal("Devices category not found in settings")
	}

	if len(devicesCategory.Settings) < 1 {
		t.Error("Devices category should have at least one setting")
	}

	// Verify device_count setting exists
	found := false
	for _, setting := range devicesCategory.Settings {
		if setting.Key == "device_count" {
			found = true
			break
		}
	}

	if !found {
		t.Error("device_count setting not found in Devices category")
	}
}
