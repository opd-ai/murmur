package declarations

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestNewSpecterDeclaration(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	spec, err := NewSpecterDeclaration(publicKey, "TestSpecter")
	if err != nil {
		t.Fatalf("NewSpecterDeclaration() error: %v", err)
	}

	if spec.Pseudonym != "TestSpecter" {
		t.Errorf("Pseudonym = %q, want %q", spec.Pseudonym, "TestSpecter")
	}
	if spec.InitialResonance != 0 {
		t.Errorf("InitialResonance = %d, want 0", spec.InitialResonance)
	}
}

func TestNewSpecterDeclarationInvalidKey(t *testing.T) {
	_, err := NewSpecterDeclaration([]byte("short"), "TestSpecter")
	if err != ErrInvalidSpecterKey {
		t.Errorf("Expected ErrInvalidSpecterKey, got %v", err)
	}
}

func TestNewSpecterDeclarationPseudonymTooLong(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	longPseudonym := "ThisPseudonymIsWayTooLongAndShouldFail"
	_, err := NewSpecterDeclaration(publicKey, longPseudonym)
	if err != ErrPseudonymTooLong {
		t.Errorf("Expected ErrPseudonymTooLong, got %v", err)
	}
}

func TestGeneratePseudonym(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	pseudonym := GeneratePseudonym(publicKey)
	if pseudonym == "" {
		t.Error("GeneratePseudonym() returned empty string")
	}

	// Should be deterministic.
	pseudonym2 := GeneratePseudonym(publicKey)
	if pseudonym != pseudonym2 {
		t.Errorf("GeneratePseudonym() not deterministic: %q != %q", pseudonym, pseudonym2)
	}

	// Different keys should (usually) produce different pseudonyms.
	otherKey := make([]byte, SpecterKeySize)
	rand.Read(otherKey)
	otherPseudonym := GeneratePseudonym(otherKey)
	// This could theoretically fail with tiny probability if both happen to hash to same indices.
	if pseudonym == otherPseudonym {
		t.Log("Warning: two random keys produced same pseudonym (unlikely but possible)")
	}
}

func TestSpecterPoWTarget(t *testing.T) {
	// Test that target has correct number of leading zeros.
	target := computePoWTarget(20)

	// First 2 bytes should be 0, third byte should have 4 leading zero bits.
	if target[0] != 0 || target[1] != 0 {
		t.Error("Target should have first 2 bytes as 0 for difficulty 20")
	}
	if target[2] != 0x0f { // 4 bits set in lower nibble
		t.Errorf("Target[2] = %02x, want 0x0f", target[2])
	}
}

func TestSpecterPoWVerification(t *testing.T) {
	// Use a low difficulty for testing (would be too slow otherwise).
	payload := []byte("test payload for pow")
	target := computePoWTarget(8) // Only 8 leading zero bits for fast test.

	// Find a valid nonce.
	var validNonce uint64
	for nonce := uint64(0); nonce < 1000000; nonce++ {
		if verifyPoWAttempt(payload, nonce, target) {
			validNonce = nonce
			break
		}
	}

	// Verify it passes.
	if !verifyPoWAttempt(payload, validNonce, target) {
		t.Error("Valid nonce failed verification")
	}
}

func TestSpecterSignAndVerify(t *testing.T) {
	// Generate Ed25519 keypair for signing.
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error: %v", err)
	}

	// Use the public key as Specter public key (simplified for testing).
	specterPub := make([]byte, SpecterKeySize)
	copy(specterPub, edPub[:SpecterKeySize])

	spec, err := NewSpecterDeclaration(specterPub, "TestSpecter")
	if err != nil {
		t.Fatalf("NewSpecterDeclaration() error: %v", err)
	}

	// Sign.
	if err := spec.Sign(edPriv); err != nil {
		t.Fatalf("Sign() error: %v", err)
	}

	// Verify.
	if err := spec.Verify(edPub); err != nil {
		t.Errorf("Verify() error: %v", err)
	}
}

func TestSpecterVerifyUnsigned(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	spec, _ := NewSpecterDeclaration(publicKey, "TestSpecter")

	edPub := make([]byte, ed25519.PublicKeySize)
	rand.Read(edPub)

	err := spec.Verify(edPub)
	if err != ErrSpecterNotSigned {
		t.Errorf("Expected ErrSpecterNotSigned, got %v", err)
	}
}

func TestSpecterMarshalUnmarshal(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	spec, _ := NewSpecterDeclaration(publicKey, "TestSpecter")
	spec.SigilPNG = []byte("fake png data")
	spec.PoWNonce = 12345

	data, err := spec.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	decoded, err := UnmarshalSpecter(data)
	if err != nil {
		t.Fatalf("UnmarshalSpecter() error: %v", err)
	}

	if decoded.Pseudonym != "TestSpecter" {
		t.Errorf("Pseudonym = %q, want %q", decoded.Pseudonym, "TestSpecter")
	}
	if decoded.PoWNonce != 12345 {
		t.Errorf("PoWNonce = %d, want 12345", decoded.PoWNonce)
	}
	if string(decoded.SigilPNG) != "fake png data" {
		t.Error("SigilPNG mismatch")
	}
}

func TestSpecterWordLists(t *testing.T) {
	// Verify word lists are reasonable size.
	if len(specterAdjectives) < 30 {
		t.Errorf("specterAdjectives has %d entries, want >= 30", len(specterAdjectives))
	}
	if len(specterNouns) < 30 {
		t.Errorf("specterNouns has %d entries, want >= 30", len(specterNouns))
	}

	// Check no empty strings.
	for i, adj := range specterAdjectives {
		if adj == "" {
			t.Errorf("specterAdjectives[%d] is empty", i)
		}
	}
	for i, noun := range specterNouns {
		if noun == "" {
			t.Errorf("specterNouns[%d] is empty", i)
		}
	}
}

func TestSpecterSetSigil(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	spec, _ := NewSpecterDeclaration(publicKey, "TestSpecter")

	sigilData := []byte("test sigil png data")
	spec.SetSigil(sigilData)

	if string(spec.SigilPNG) != string(sigilData) {
		t.Error("SetSigil() did not set SigilPNG correctly")
	}
}

func TestSpecterTimestampValidation(t *testing.T) {
	publicKey := make([]byte, SpecterKeySize)
	rand.Read(publicKey)

	spec, _ := NewSpecterDeclaration(publicKey, "TestSpecter")

	// Fresh timestamp should be valid.
	if err := spec.ValidateTimestamp(); err != nil {
		t.Errorf("Fresh timestamp validation failed: %v", err)
	}

	// Old timestamp should fail.
	spec.CreatedAt = 0
	if err := spec.ValidateTimestamp(); err != ErrTimestampTooOld {
		t.Errorf("Expected ErrTimestampTooOld, got %v", err)
	}
}
