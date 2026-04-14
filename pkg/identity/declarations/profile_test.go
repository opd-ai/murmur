// Package declarations tests verify identity declaration creation and validation.
package declarations

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

// TestDeclarationStructure verifies Declaration struct fields exist.
func TestDeclarationStructure(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate test keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := New(kp, "TestUser")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}

	// Verify fields.
	if len(decl.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey should be %d bytes, got %d", ed25519.PublicKeySize, len(decl.PublicKey))
	}

	if decl.DisplayName != "TestUser" {
		t.Errorf("DisplayName should be 'TestUser', got '%s'", decl.DisplayName)
	}

	if decl.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	if decl.Version != 1 {
		t.Errorf("Initial version should be 1, got %d", decl.Version)
	}
}

// TestDeclarationZeroValue verifies zero-value Declaration is valid Go.
func TestDeclarationZeroValue(t *testing.T) {
	var decl Declaration

	// Zero values should be nil/empty/0.
	if decl.PublicKey != nil {
		t.Error("Zero Declaration.PublicKey should be nil")
	}
	if decl.DisplayName != "" {
		t.Error("Zero Declaration.DisplayName should be empty")
	}
	if decl.Timestamp != 0 {
		t.Error("Zero Declaration.Timestamp should be 0")
	}
	if decl.Signature != nil {
		t.Error("Zero Declaration.Signature should be nil")
	}
}

// TestDeclarationTimestampReasonable verifies timestamps are in expected range.
func TestDeclarationTimestampReasonable(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	defer kp.ZeroKeyPair()

	decl, _ := New(kp, "TimestampTest")

	// Per TECHNICAL_IMPLEMENTATION.md, timestamps should be within ±300 seconds of current time.
	if err := decl.ValidateTimestamp(); err != nil {
		t.Errorf("Fresh declaration should have valid timestamp: %v", err)
	}
}

// TestSignAndVerify tests the sign/verify round-trip.
func TestSignAndVerify(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := New(kp, "SignTest")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}

	// Sign the declaration.
	if err := decl.Sign(kp); err != nil {
		t.Fatalf("Failed to sign declaration: %v", err)
	}

	// Verify the signature.
	if err := decl.Verify(); err != nil {
		t.Errorf("Signature verification failed: %v", err)
	}

	// Tamper with the declaration.
	decl.DisplayName = "Tampered"
	if err := decl.Verify(); err == nil {
		t.Error("Tampered declaration should fail verification")
	}
}

// TestMarshalUnmarshal tests protobuf serialization round-trip.
func TestMarshalUnmarshal(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	original, err := New(kp, "MarshalTest")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}
	original.Bio = "Test bio"
	original.SigilPNG = []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	original.PrivacyMode = modes.Hybrid

	if err := original.Sign(kp); err != nil {
		t.Fatalf("Failed to sign declaration: %v", err)
	}

	// Marshal to protobuf.
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal declaration: %v", err)
	}

	// Unmarshal from protobuf.
	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal declaration: %v", err)
	}

	// Verify fields match.
	if string(restored.PublicKey) != string(original.PublicKey) {
		t.Error("PublicKey mismatch after round-trip")
	}
	if restored.DisplayName != original.DisplayName {
		t.Error("DisplayName mismatch after round-trip")
	}
	if restored.Bio != original.Bio {
		t.Error("Bio mismatch after round-trip")
	}
	if restored.Timestamp != original.Timestamp {
		t.Error("Timestamp mismatch after round-trip")
	}
	if restored.Version != original.Version {
		t.Error("Version mismatch after round-trip")
	}
	if string(restored.Signature) != string(original.Signature) {
		t.Error("Signature mismatch after round-trip")
	}
	if string(restored.SigilPNG) != string(original.SigilPNG) {
		t.Error("SigilPNG mismatch after round-trip")
	}
	if restored.PrivacyMode != original.PrivacyMode {
		t.Error("PrivacyMode mismatch after round-trip")
	}

	// Verify signature still valid after round-trip.
	if err := restored.Verify(); err != nil {
		t.Errorf("Signature invalid after round-trip: %v", err)
	}
}

// TestNewWithNilKeyPair tests error handling for nil keypair.
func TestNewWithNilKeyPair(t *testing.T) {
	_, err := New(nil, "Test")
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

// TestSignWithNilKeyPair tests error handling for nil keypair on sign.
func TestSignWithNilKeyPair(t *testing.T) {
	decl := &Declaration{
		PublicKey:   make([]byte, ed25519.PublicKeySize),
		DisplayName: "Test",
		Timestamp:   time.Now().Unix(),
	}
	if err := decl.Sign(nil); err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

// TestDisplayNameTooLong tests validation of display name length.
func TestDisplayNameTooLong(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	defer kp.ZeroKeyPair()

	longName := strings.Repeat("x", MaxDisplayNameLen+1)
	_, err := New(kp, longName)
	if err != ErrDisplayNameTooLon {
		t.Errorf("Expected ErrDisplayNameTooLong, got %v", err)
	}
}

// TestBioTooLong tests validation of bio length.
func TestBioTooLong(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	defer kp.ZeroKeyPair()

	decl, _ := New(kp, "Test")
	longBio := strings.Repeat("x", MaxBioLen+1)
	if err := decl.SetBio(longBio); err != ErrBioTooLong {
		t.Errorf("Expected ErrBioTooLong, got %v", err)
	}
}

// TestInvalidPublicKeySize tests verification with wrong key size.
func TestInvalidPublicKeySize(t *testing.T) {
	decl := &Declaration{
		PublicKey:   make([]byte, 16), // Wrong size
		DisplayName: "Test",
		Timestamp:   time.Now().Unix(),
		Signature:   make([]byte, ed25519.SignatureSize),
	}
	if err := decl.Verify(); err != ErrInvalidPublicKey {
		t.Errorf("Expected ErrInvalidPublicKey, got %v", err)
	}
}

// TestVersionIncrement tests the version increment functionality.
func TestVersionIncrement(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	defer kp.ZeroKeyPair()

	decl, _ := New(kp, "Test")
	if decl.Version != 1 {
		t.Errorf("Initial version should be 1, got %d", decl.Version)
	}

	updated := decl.Update()
	if updated.Version != 2 {
		t.Errorf("Updated version should be 2, got %d", updated.Version)
	}
}

// TestPrivacyModeConversion tests privacy mode round-trip conversion.
func TestPrivacyModeConversion(t *testing.T) {
	testModes := []modes.Mode{modes.Open, modes.Hybrid, modes.Guarded, modes.Fortress}

	for _, mode := range testModes {
		kp, _ := keys.GenerateKeyPair()
		decl, _ := New(kp, "ModeTest")
		decl.SetPrivacyMode(mode)
		decl.Sign(kp)

		data, _ := decl.Marshal()
		restored, _ := Unmarshal(data)

		if restored.PrivacyMode != mode {
			t.Errorf("Mode %v not preserved after round-trip, got %v", mode, restored.PrivacyMode)
		}
		kp.ZeroKeyPair()
	}
}

// TestValidate tests full declaration validation.
func TestValidate(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	defer kp.ZeroKeyPair()

	decl, _ := New(kp, "ValidateTest")
	decl.Sign(kp)

	// Fresh, properly signed declaration should validate.
	if err := decl.Validate(); err != nil {
		t.Errorf("Valid declaration failed validation: %v", err)
	}
}

// TestDeclarationWithPoWStructure tests DeclarationWithPoW fields.
func TestDeclarationWithPoWStructure(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := NewWithPoW(kp, "PoWTest")
	if err != nil {
		t.Fatalf("Failed to create declaration with PoW: %v", err)
	}

	// Verify embedded Declaration fields.
	if len(decl.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey should be %d bytes, got %d", ed25519.PublicKeySize, len(decl.PublicKey))
	}
	if decl.DisplayName != "PoWTest" {
		t.Errorf("DisplayName should be 'PoWTest', got '%s'", decl.DisplayName)
	}
}

// TestIdentityPoWComputation tests PoW computation and verification.
// Uses reduced difficulty for faster test execution.
func TestIdentityPoWComputation(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := NewWithPoW(kp, "PoWComputeTest")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}

	// Sign the declaration first.
	if err := decl.Sign(kp); err != nil {
		t.Fatalf("Failed to sign declaration: %v", err)
	}

	// Compute PoW (may take a moment).
	if err := decl.ComputePoW(); err != nil {
		t.Fatalf("Failed to compute PoW: %v", err)
	}

	// Nonce should be set.
	if decl.PoWNonce == 0 {
		t.Log("Warning: PoW nonce is 0 (could be valid by chance)")
	}

	// Verify PoW.
	if err := decl.VerifyPoW(); err != nil {
		t.Errorf("PoW verification failed: %v", err)
	}
}

// TestIdentityPoWVerificationFailure tests invalid PoW detection.
func TestIdentityPoWVerificationFailure(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := NewWithPoW(kp, "PoWFailTest")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}

	// Don't compute PoW - just set an invalid nonce.
	decl.PoWNonce = 12345

	// Verification should fail.
	if err := decl.VerifyPoW(); err != ErrInvalidIdentityPoW {
		t.Errorf("Expected ErrInvalidIdentityPoW, got %v", err)
	}
}

// TestIdentityPoWTargetComputation tests target generation.
func TestIdentityPoWTargetComputation(t *testing.T) {
	tests := []struct {
		difficulty   int
		firstNonZero int  // Index of first byte that should be non-zero
		firstByte    byte // Expected value of first non-zero byte
	}{
		{8, 1, 0xff},  // 1 byte of zeros
		{16, 2, 0xff}, // 2 bytes of zeros
		{18, 2, 0x3f}, // 2 bytes of zeros + 2 bits
		{20, 2, 0x0f}, // 2 bytes of zeros + 4 bits
		{24, 3, 0xff}, // 3 bytes of zeros
	}

	for _, tc := range tests {
		target := computeIdentityPoWTarget(tc.difficulty)

		// Check leading zeros.
		for i := 0; i < tc.firstNonZero; i++ {
			if target[i] != 0 {
				t.Errorf("difficulty %d: byte %d should be 0, got %02x", tc.difficulty, i, target[i])
			}
		}

		// Check first non-zero byte.
		if target[tc.firstNonZero] != tc.firstByte {
			t.Errorf("difficulty %d: byte %d should be %02x, got %02x",
				tc.difficulty, tc.firstNonZero, tc.firstByte, target[tc.firstNonZero])
		}
	}
}

// TestValidateWithPoW tests full validation including PoW.
func TestValidateWithPoW(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	defer kp.ZeroKeyPair()

	decl, err := NewWithPoW(kp, "FullValidateTest")
	if err != nil {
		t.Fatalf("Failed to create declaration: %v", err)
	}

	// Sign the declaration.
	if err := decl.Sign(kp); err != nil {
		t.Fatalf("Failed to sign declaration: %v", err)
	}

	// Compute PoW.
	if err := decl.ComputePoW(); err != nil {
		t.Fatalf("Failed to compute PoW: %v", err)
	}

	// Full validation should pass.
	if err := decl.ValidateWithPoW(); err != nil {
		t.Errorf("Full validation failed: %v", err)
	}
}

// TestNewWithPoWNilKeyPair tests error handling for nil keypair.
func TestNewWithPoWNilKeyPair(t *testing.T) {
	_, err := NewWithPoW(nil, "Test")
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}
