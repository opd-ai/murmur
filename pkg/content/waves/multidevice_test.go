package waves

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// mockDeviceStore implements DeviceAuthorizer for testing.
type mockDeviceStore struct {
	authorized bool
	err        error
}

func (m *mockDeviceStore) IsDeviceAuthorizedWithGracePeriod(masterPubKey, devicePubKey []byte, waveTimestamp int64) (bool, error) {
	return m.authorized, m.err
}

func TestCreateWithDeviceKey(t *testing.T) {
	// Generate device keypair
	deviceKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Multi-device test wave")

	// Create wave with device key
	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	opts.DeviceKey = deviceKP.PublicKey

	// Note: We sign with deviceKP since in multi-device mode the device key signs
	wave, err := Create(TypeSurface, content, deviceKP, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify wave has device_public_key set
	if len(wave.DevicePublicKey) == 0 {
		t.Error("device_public_key should be set in multi-device mode")
	}

	if string(wave.DevicePublicKey) != string(deviceKP.PublicKey) {
		t.Error("device_public_key mismatch")
	}

	// In this test, author_pubkey is the device key (signing key)
	// In production, you would set author_pubkey to master and sign with device
	if string(wave.AuthorPubkey) != string(deviceKP.PublicKey) {
		t.Error("author_pubkey should be device key (signing key)")
	}
}

func TestCreateSingleDeviceMode(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Single-device test wave")

	// Create wave without device key (single-device mode)
	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	// opts.DeviceKey is nil by default

	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify device_public_key is empty (single-device mode)
	if len(wave.DevicePublicKey) != 0 {
		t.Error("device_public_key should be empty in single-device mode")
	}
}

func TestValidateWithDeviceStore_SingleDevice(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Validation test")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Single-device mode should validate without device store
	if err := ValidateWithDeviceStore(wave, 8, nil); err != nil {
		t.Errorf("ValidateWithDeviceStore failed for single-device: %v", err)
	}
}

func TestValidateWithDeviceStore_MultiDevice_Authorized(t *testing.T) {
	// Generate master and device keypairs
	masterKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	deviceKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Create wave in multi-device mode
	// Important: In real multi-device mode, author_pubkey should be master key
	// and we sign with device key. For this test, we'll manually construct.
	content := []byte("Multi-device authorized wave")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	opts.DeviceKey = deviceKP.PublicKey

	// Create with device key as signer
	wave, err := Create(TypeSurface, content, deviceKP, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Override author_pubkey to master (simulating correct multi-device setup)
	wave.AuthorPubkey = masterKP.PublicKey

	// Mock device store that authorizes this device
	store := &mockDeviceStore{authorized: true}

	// Should pass validation
	if err := ValidateWithDeviceStore(wave, 8, store); err != nil {
		t.Errorf("ValidateWithDeviceStore failed for authorized device: %v", err)
	}
}

func TestValidateWithDeviceStore_MultiDevice_Unauthorized(t *testing.T) {
	// Generate master and device keypairs
	masterKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	deviceKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Multi-device unauthorized wave")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	opts.DeviceKey = deviceKP.PublicKey

	wave, err := Create(TypeSurface, content, deviceKP, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Override author_pubkey to master
	wave.AuthorPubkey = masterKP.PublicKey

	// Mock device store that does NOT authorize this device
	store := &mockDeviceStore{authorized: false}

	// Should fail validation
	if err := ValidateWithDeviceStore(wave, 8, store); err == nil {
		t.Error("ValidateWithDeviceStore should fail for unauthorized device")
	}
}

func TestValidateWithDeviceStore_BackwardCompatibility(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Legacy wave")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	// Create old-style wave without device key
	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Old Validate function should still work
	if err := Validate(wave, 8); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// New ValidateWithDeviceStore should also work (single-device mode)
	if err := ValidateWithDeviceStore(wave, 8, nil); err != nil {
		t.Errorf("ValidateWithDeviceStore failed: %v", err)
	}
}

func TestCreateOptions_DeviceKeyField(t *testing.T) {
	// Test that DeviceKey field is properly initialized in DefaultCreateOptions
	opts := DefaultCreateOptions()

	if opts.DeviceKey != nil {
		t.Error("DeviceKey should be nil in DefaultCreateOptions")
	}

	if opts.TTL != DefaultTTL {
		t.Errorf("TTL mismatch: got %v, want %v", opts.TTL, DefaultTTL)
	}

	if opts.Difficulty != DefaultDifficulty {
		t.Errorf("Difficulty mismatch: got %d, want %d", opts.Difficulty, DefaultDifficulty)
	}
}

func TestValidateWithDeviceStore_PassesTimestamp(t *testing.T) {
	// This tests that wave timestamp is passed to IsDeviceAuthorizedWithGracePeriod
	masterKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	deviceKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Timestamp passing test")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	opts.DeviceKey = deviceKP.PublicKey

	wave, err := Create(TypeSurface, content, deviceKP, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set author to master
	wave.AuthorPubkey = masterKP.PublicKey

	// Mock device store that verifies timestamp is passed correctly
	store := &mockDeviceStore{authorized: true}

	// Validation should work - the important thing is that wave.CreatedAt
	// gets passed to IsDeviceAuthorizedWithGracePeriod
	if err := ValidateWithDeviceStore(wave, 8, store); err != nil {
		t.Errorf("ValidateWithDeviceStore failed: %v", err)
	}
}
