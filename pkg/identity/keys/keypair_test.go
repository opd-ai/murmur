package keys

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	defer kp.ZeroKeyPair()

	if len(kp.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey size = %d, want %d", len(kp.PublicKey), ed25519.PublicKeySize)
	}

	if len(kp.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("PrivateKey size = %d, want %d", len(kp.PrivateKey), ed25519.PrivateKeySize)
	}
}

func TestSignVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	defer kp.ZeroKeyPair()

	message := []byte("test message for signing")
	signature := kp.Sign(message)

	if len(signature) != ed25519.SignatureSize {
		t.Errorf("Signature size = %d, want %d", len(signature), ed25519.SignatureSize)
	}

	// Verify with correct public key.
	if !Verify(kp.PublicKey, message, signature) {
		t.Error("Verify() returned false for valid signature")
	}

	// Verify with wrong message.
	wrongMessage := []byte("wrong message")
	if Verify(kp.PublicKey, wrongMessage, signature) {
		t.Error("Verify() returned true for wrong message")
	}

	// Verify with corrupted signature.
	corruptedSig := make([]byte, len(signature))
	copy(corruptedSig, signature)
	corruptedSig[0] ^= 0xFF
	if Verify(kp.PublicKey, message, corruptedSig) {
		t.Error("Verify() returned true for corrupted signature")
	}
}

func TestGenerateAnonymousKeyPair(t *testing.T) {
	kp, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair() error = %v", err)
	}
	defer kp.ZeroAnonymousKeyPair()

	// Check key sizes.
	if len(kp.PublicKey) != 32 {
		t.Errorf("PublicKey size = %d, want 32", len(kp.PublicKey))
	}

	if len(kp.PrivateKey) != 32 {
		t.Errorf("PrivateKey size = %d, want 32", len(kp.PrivateKey))
	}

	// Keys should not be all zeros.
	var zero [32]byte
	if kp.PublicKey == zero {
		t.Error("PublicKey is all zeros")
	}
	if kp.PrivateKey == zero {
		t.Error("PrivateKey is all zeros")
	}
}

func TestDeriveSharedSecret(t *testing.T) {
	// Generate two keypairs.
	alice, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair() for Alice error = %v", err)
	}
	defer alice.ZeroAnonymousKeyPair()

	bob, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair() for Bob error = %v", err)
	}
	defer bob.ZeroAnonymousKeyPair()

	// Derive shared secrets.
	aliceShared, err := alice.DeriveSharedSecret(bob.PublicKey)
	if err != nil {
		t.Fatalf("Alice DeriveSharedSecret() error = %v", err)
	}

	bobShared, err := bob.DeriveSharedSecret(alice.PublicKey)
	if err != nil {
		t.Fatalf("Bob DeriveSharedSecret() error = %v", err)
	}

	// Shared secrets should be equal.
	if !bytes.Equal(aliceShared, bobShared) {
		t.Error("Shared secrets do not match")
	}

	// Shared secret should be 32 bytes.
	if len(aliceShared) != 32 {
		t.Errorf("Shared secret size = %d, want 32", len(aliceShared))
	}
}

func TestEncryptDecryptKeystore(t *testing.T) {
	plaintext := []byte("super secret key material")
	passphrase := "test-passphrase-123"

	encrypted, err := EncryptKeystore(plaintext, passphrase)
	if err != nil {
		t.Fatalf("EncryptKeystore() error = %v", err)
	}

	// Encrypted data should be longer than plaintext.
	minLen := SaltSize + NonceSize + len(plaintext)
	if len(encrypted) < minLen {
		t.Errorf("Encrypted length = %d, expected >= %d", len(encrypted), minLen)
	}

	// Decrypt with correct passphrase.
	decrypted, err := DecryptKeystore(encrypted, passphrase)
	if err != nil {
		t.Fatalf("DecryptKeystore() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted = %q, want %q", decrypted, plaintext)
	}

	// Decrypt with wrong passphrase should fail.
	_, err = DecryptKeystore(encrypted, "wrong-passphrase")
	if err == nil {
		t.Error("DecryptKeystore() with wrong passphrase should fail")
	}
}

func TestDecryptKeystoreTooShort(t *testing.T) {
	shortData := []byte("too short")
	_, err := DecryptKeystore(shortData, "any")
	if err == nil {
		t.Error("DecryptKeystore() with short data should fail")
	}
}

func TestZeroBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ZeroBytes(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("data[%d] = %d, want 0", i, b)
		}
	}
}

func TestKeyPairUniqueness(t *testing.T) {
	// Generate multiple keypairs and ensure they're different.
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()
	defer kp1.ZeroKeyPair()
	defer kp2.ZeroKeyPair()

	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("Two generated keypairs have identical public keys")
	}
}
