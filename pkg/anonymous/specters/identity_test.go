package specters

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if kp == nil {
		t.Fatal("keypair is nil")
	}

	// Check key lengths.
	if len(kp.Private) != 32 {
		t.Errorf("private key length = %d, want 32", len(kp.Private))
	}
	if len(kp.Public) != 32 {
		t.Errorf("public key length = %d, want 32", len(kp.Public))
	}

	// Ensure keys are not all zeros.
	var zeroes [32]byte
	if kp.Public == zeroes {
		t.Error("public key is all zeros")
	}
}

func TestKeyPairIndependence(t *testing.T) {
	// Generate multiple keypairs and verify they're independent.
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()

	if kp1.Private == kp2.Private {
		t.Error("two keypairs have same private key")
	}
	if kp1.Public == kp2.Public {
		t.Error("two keypairs have same public key")
	}
}

func TestNewSpecter(t *testing.T) {
	s, err := NewSpecter()
	if err != nil {
		t.Fatalf("NewSpecter failed: %v", err)
	}

	if s == nil {
		t.Fatal("specter is nil")
	}

	if s.Status != StatusActive {
		t.Errorf("status = %s, want %s", s.Status, StatusActive)
	}

	if s.Name == "" {
		t.Error("name is empty")
	}

	// Name should be two words.
	words := strings.Split(s.Name, " ")
	if len(words) != 2 {
		t.Errorf("name has %d words, want 2", len(words))
	}
}

func TestSpecterLifecycle(t *testing.T) {
	s, _ := NewSpecter()

	// Should be active initially.
	if !s.IsActive() {
		t.Error("specter should be active initially")
	}

	// Suspend.
	if err := s.Suspend(); err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	if s.IsActive() {
		t.Error("specter should not be active after suspend")
	}

	// Activate.
	if err := s.Activate(); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if !s.IsActive() {
		t.Error("specter should be active after activate")
	}

	// Delete.
	s.Delete()
	if s.Status != StatusDeleted {
		t.Errorf("status = %s, want %s", s.Status, StatusDeleted)
	}

	// Operations should fail after delete.
	if err := s.Suspend(); err != ErrDeleted {
		t.Errorf("expected ErrDeleted, got %v", err)
	}
	if err := s.Activate(); err != ErrDeleted {
		t.Errorf("expected ErrDeleted, got %v", err)
	}
}

func TestSpecterKeyZeroing(t *testing.T) {
	s, _ := NewSpecter()

	// Save original private key for comparison.
	originalKey := s.PrivateKey

	s.Delete()

	// Private key should be zeroed.
	var zeroes [32]byte
	if s.PrivateKey != zeroes {
		t.Error("private key not zeroed after delete")
	}
	if originalKey == zeroes {
		t.Error("original key was already zeros (test invalid)")
	}
}

func TestDeriveSharedSecret(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()

	// Derive shared secret from both sides.
	shared1, err := s1.DeriveSharedSecret(s2.PublicKey[:])
	if err != nil {
		t.Fatalf("DeriveSharedSecret failed: %v", err)
	}

	shared2, err := s2.DeriveSharedSecret(s1.PublicKey[:])
	if err != nil {
		t.Fatalf("DeriveSharedSecret failed: %v", err)
	}

	// Both should derive the same secret.
	if !bytes.Equal(shared1, shared2) {
		t.Error("shared secrets don't match")
	}
}

func TestDeriveSharedSecretErrors(t *testing.T) {
	s, _ := NewSpecter()

	// Invalid public key length.
	_, err := s.DeriveSharedSecret([]byte("short"))
	if err == nil {
		t.Error("expected error for short public key")
	}

	// After suspend.
	s.Suspend()
	_, err = s.DeriveSharedSecret(make([]byte, 32))
	if err != ErrSuspended {
		t.Errorf("expected ErrSuspended, got %v", err)
	}

	// After delete.
	s.Activate()
	s.Delete()
	_, err = s.DeriveSharedSecret(make([]byte, 32))
	if err != ErrDeleted {
		t.Errorf("expected ErrDeleted, got %v", err)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()

	plaintext := []byte("secret message for specter communication")

	// s1 encrypts for s2.
	ciphertext, err := s1.Encrypt(plaintext, s2.PublicKey[:])
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// s2 decrypts.
	decrypted, err := s2.Decrypt(ciphertext, s1.PublicKey[:])
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("decrypted text doesn't match original")
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()

	// Empty plaintext should work.
	ciphertext, err := s1.Encrypt([]byte{}, s2.PublicKey[:])
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := s2.Decrypt(ciphertext, s1.PublicKey[:])
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Error("decrypted should be empty")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	s1, _ := NewSpecter()
	s2, _ := NewSpecter()
	s3, _ := NewSpecter() // unintended recipient

	plaintext := []byte("secret message")
	ciphertext, _ := s1.Encrypt(plaintext, s2.PublicKey[:])

	// s3 should not be able to decrypt.
	_, err := s3.Decrypt(ciphertext, s1.PublicKey[:])
	if err != ErrDecryptionFail {
		t.Errorf("expected ErrDecryptionFail, got %v", err)
	}
}

func TestGenerateName(t *testing.T) {
	kp, _ := GenerateKeyPair()

	name := GenerateName(kp.Public[:])

	// Should be two words.
	words := strings.Split(name, " ")
	if len(words) != 2 {
		t.Errorf("name has %d words, want 2", len(words))
	}

	// Same public key should generate same name (deterministic).
	name2 := GenerateName(kp.Public[:])
	if name != name2 {
		t.Error("same public key should generate same name")
	}
}

func TestGenerateNameUniqueness(t *testing.T) {
	// Generate many names and check uniqueness.
	names := make(map[string]int)

	for i := 0; i < 100; i++ {
		kp, _ := GenerateKeyPair()
		name := GenerateName(kp.Public[:])
		names[name]++
	}

	// With ~100*100 = 10,000 possible combinations and 100 samples,
	// collisions are possible but should be rare.
	collisions := 0
	for _, count := range names {
		if count > 1 {
			collisions += count - 1
		}
	}

	// Allow some collisions (< 5%).
	if collisions > 5 {
		t.Errorf("too many collisions: %d", collisions)
	}
}

func TestGenerateNameWithPrefix(t *testing.T) {
	kp, _ := GenerateKeyPair()
	baseName := GenerateName(kp.Public[:])

	// With no existing names, should return base name.
	existing := make(map[string]bool)
	name := GenerateNameWithPrefix(kp.Public[:], existing)
	if name != baseName {
		t.Errorf("expected %s, got %s", baseName, name)
	}

	// With base name taken, should return different name.
	existing[baseName] = true
	name = GenerateNameWithPrefix(kp.Public[:], existing)
	if name == baseName {
		t.Error("should have generated different name when base collides")
	}
}

func TestNewSpecterFromKeyPair(t *testing.T) {
	kp, _ := GenerateKeyPair()

	s, err := NewSpecterFromKeyPair(kp)
	if err != nil {
		t.Fatalf("NewSpecterFromKeyPair failed: %v", err)
	}

	if s.PrivateKey != kp.Private {
		t.Error("private key mismatch")
	}
	if s.PublicKey != kp.Public {
		t.Error("public key mismatch")
	}
}

func TestNewSpecterFromKeyPairNil(t *testing.T) {
	_, err := NewSpecterFromKeyPair(nil)
	if err != ErrNilKeyPair {
		t.Errorf("expected ErrNilKeyPair, got %v", err)
	}
}

func TestSpecterConcurrency(t *testing.T) {
	s, _ := NewSpecter()
	s2, _ := NewSpecter()

	// Multiple concurrent operations should not race.
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			s.IsActive()
			s.DeriveSharedSecret(s2.PublicKey[:])
			s.Encrypt([]byte("test"), s2.PublicKey[:])
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
