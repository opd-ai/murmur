// Package screens provides the UI screens for the MURMUR onboarding flow.
package screens

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// TestRecoveryScreenCreation validates initial state.
func TestRecoveryScreenCreation(t *testing.T) {
	screen := NewRecoveryScreen()

	if screen.method != RecoveryMethodNone {
		t.Errorf("expected method None, got %v", screen.method)
	}

	if screen.IsCompleted() {
		t.Error("new screen should not be completed")
	}

	if screen.GetRecoveredKey() != nil {
		t.Error("new screen should have no recovered key")
	}
}

// TestRecoveryScreenReset validates state reset.
func TestRecoveryScreenReset(t *testing.T) {
	screen := NewRecoveryScreen()

	// Set some state.
	screen.method = RecoveryMethodMnemonic
	screen.mnemonicText = "test words"
	screen.errorMsg = "test error"
	screen.completed = true

	// Reset.
	screen.Reset()

	// Verify state cleared.
	if screen.method != RecoveryMethodNone {
		t.Error("method not reset")
	}
	if screen.mnemonicText != "" {
		t.Error("mnemonic text not cleared")
	}
	if screen.errorMsg != "" {
		t.Error("error message not cleared")
	}
	if screen.completed {
		t.Error("completed flag not cleared")
	}
}

// TestMnemonicRecovery validates mnemonic-based recovery.
func TestMnemonicRecovery(t *testing.T) {
	passphrase := "test-passphrase-secure-12345"
	// Generate a keypair with mnemonic.
	_, backup, err := keys.GenerateBackup(passphrase)
	if err != nil {
		t.Fatalf("failed to generate backup: %v", err)
	}

	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodMnemonic
	screen.mnemonicText = backup.Mnemonic
	screen.passphrase = passphrase

	// Attempt recovery.
	screen.attemptRecovery()

	// Verify success.
	if !screen.IsCompleted() {
		t.Errorf("recovery failed: %s", screen.errorMsg)
	}

	if screen.GetRecoveredKey() == nil {
		t.Error("no recovered key")
	}

	if screen.errorMsg != "" {
		t.Errorf("unexpected error: %s", screen.errorMsg)
	}
}

// TestInvalidMnemonicRecovery validates error handling for invalid mnemonic.
func TestInvalidMnemonicRecovery(t *testing.T) {
	passphrase := "test-passphrase-secure-12345"
	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodMnemonic
	screen.mnemonicText = "invalid mnemonic phrase that should fail"
	screen.passphrase = passphrase

	// Attempt recovery.
	screen.attemptRecovery()

	// Verify failure.
	if screen.IsCompleted() {
		t.Error("recovery should have failed")
	}

	if screen.errorMsg == "" {
		t.Error("expected error message for invalid mnemonic")
	}

	if screen.GetRecoveredKey() != nil {
		t.Error("should not have recovered key")
	}
}

// TestEmptyMnemonicRecovery validates error handling for empty mnemonic.
func TestEmptyMnemonicRecovery(t *testing.T) {
	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodMnemonic
	screen.mnemonicText = "   "

	// Attempt recovery.
	screen.attemptRecovery()

	// Verify failure.
	if screen.IsCompleted() {
		t.Error("recovery should have failed")
	}

	if screen.errorMsg == "" {
		t.Error("expected error message for empty mnemonic")
	}
}

// TestRecoveryScreenUpdate validates Update() doesn't panic.
func TestRecoveryScreenUpdate(t *testing.T) {
	screen := NewRecoveryScreen()

	// Update should not panic in any state.
	if err := screen.Update(); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	screen.method = RecoveryMethodMnemonic
	if err := screen.Update(); err != nil {
		t.Errorf("Update failed in mnemonic mode: %v", err)
	}
}

// TestRecoveryScreenDraw validates Draw() doesn't panic.
func TestRecoveryScreenDraw(t *testing.T) {
	screen := NewRecoveryScreen()
	img := ebiten.NewImage(800, 600)

	// Draw should not panic in any state.
	screen.Draw(img)

	screen.method = RecoveryMethodMnemonic
	screen.Draw(img)

	screen.method = RecoveryMethodKeyFile
	screen.Draw(img)
}

// TestAppendTypedText validates text input handling.
func TestAppendTypedText(t *testing.T) {
	// This function depends on ebiten's input state, so we can only test basic behavior.
	text := appendTypedText("test")

	// Should at least not panic and preserve existing text.
	if len(text) < 4 {
		t.Error("appendTypedText should preserve existing text")
	}
}

// TestIsInButton validates button hit detection.
func TestIsInButton(t *testing.T) {
	tests := []struct {
		name     string
		x, y     float32
		centerX  float32
		centerY  float32
		width    float32
		height   float32
		expected bool
	}{
		{"inside", 200, 250, 200, 250, 100, 50, true},
		{"left edge", 150, 250, 200, 250, 100, 50, true},
		{"right edge", 250, 250, 200, 250, 100, 50, true},
		{"top edge", 200, 225, 200, 250, 100, 50, true},
		{"bottom edge", 200, 275, 200, 250, 100, 50, true},
		{"outside left", 149, 250, 200, 250, 100, 50, false},
		{"outside right", 251, 250, 200, 250, 100, 50, false},
		{"outside top", 200, 224, 200, 250, 100, 50, false},
		{"outside bottom", 200, 276, 200, 250, 100, 50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInButton(tt.x, tt.y, tt.centerX, tt.centerY, tt.width, tt.height)
			if result != tt.expected {
				t.Errorf("isInButton(%v, %v, %v, %v, %v, %v) = %v, want %v",
					tt.x, tt.y, tt.centerX, tt.centerY, tt.width, tt.height, result, tt.expected)
			}
		})
	}
}

// TestKeyFileRecovery validates key file-based recovery with passphrase.
func TestKeyFileRecovery(t *testing.T) {
	// Generate a keypair and encrypt it.
	original, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	exported, err := keys.ExportKeyPair(original)
	if err != nil {
		t.Fatalf("failed to export keypair: %v", err)
	}

	passphrase := "test-passphrase-123"
	encrypted, err := keys.EncryptKeystore(exported, passphrase)
	if err != nil {
		t.Fatalf("failed to encrypt keystore: %v", err)
	}

	// Create recovery screen and set key file data.
	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodKeyFile
	screen.SetKeyFileData(encrypted)
	screen.passphrase = passphrase

	// Attempt recovery.
	screen.attemptKeyFileRecovery()

	// Verify success.
	if !screen.IsCompleted() {
		t.Errorf("recovery failed: %s", screen.errorMsg)
	}

	recovered := screen.GetRecoveredKey()
	if recovered == nil {
		t.Fatal("no recovered key")
	}

	// Verify keys match.
	if string(recovered.PublicKey) != string(original.PublicKey) {
		t.Error("recovered public key does not match")
	}
}

// TestKeyFileRecoveryWrongPassphrase validates error handling for wrong passphrase.
func TestKeyFileRecoveryWrongPassphrase(t *testing.T) {
	// Generate and encrypt a keypair.
	original, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	exported, err := keys.ExportKeyPair(original)
	if err != nil {
		t.Fatalf("failed to export keypair: %v", err)
	}

	encrypted, err := keys.EncryptKeystore(exported, "correct-passphrase")
	if err != nil {
		t.Fatalf("failed to encrypt keystore: %v", err)
	}

	// Create recovery screen with wrong passphrase.
	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodKeyFile
	screen.SetKeyFileData(encrypted)
	screen.passphrase = "wrong-passphrase"

	// Attempt recovery.
	screen.attemptKeyFileRecovery()

	// Verify failure.
	if screen.IsCompleted() {
		t.Error("recovery should have failed with wrong passphrase")
	}

	if screen.errorMsg == "" {
		t.Error("expected error message for wrong passphrase")
	}
}

// TestKeyFileRecoveryNoData validates error handling when no file is loaded.
func TestKeyFileRecoveryNoData(t *testing.T) {
	screen := NewRecoveryScreen()
	screen.method = RecoveryMethodKeyFile
	screen.passphraseMode = true
	screen.passphrase = "test-passphrase"

	// Attempt recovery without setting key file data.
	screen.attemptKeyFileRecovery()

	// Verify failure.
	if screen.IsCompleted() {
		t.Error("recovery should have failed with no data")
	}

	if screen.errorMsg == "" {
		t.Error("expected error message for no key file")
	}
}
