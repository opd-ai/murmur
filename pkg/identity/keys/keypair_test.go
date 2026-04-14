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

func TestGenerateIdentityBundle(t *testing.T) {
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Both keypairs should exist.
	if bundle.Surface == nil {
		t.Error("Surface keypair is nil")
	}
	if bundle.Specter == nil {
		t.Error("Specter keypair is nil")
	}
	if bundle.FortressTransport != nil {
		t.Error("FortressTransport should be nil without Fortress mode")
	}

	// Keys should be valid sizes.
	if len(bundle.Surface.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Surface public key size = %d, want %d",
			len(bundle.Surface.PublicKey), ed25519.PublicKeySize)
	}
	if len(bundle.Specter.PublicKey) != 32 {
		t.Errorf("Specter public key size = %d, want 32", len(bundle.Specter.PublicKey))
	}
}

func TestGenerateIdentityBundleWithFortress(t *testing.T) {
	bundle, err := GenerateIdentityBundleWithFortress()
	if err != nil {
		t.Fatalf("GenerateIdentityBundleWithFortress() error = %v", err)
	}
	defer bundle.Zero()

	// All three keypairs should exist.
	if bundle.Surface == nil {
		t.Error("Surface keypair is nil")
	}
	if bundle.Specter == nil {
		t.Error("Specter keypair is nil")
	}
	if bundle.FortressTransport == nil {
		t.Error("FortressTransport keypair is nil")
	}
}

func TestIdentityBundleValidateIndependence(t *testing.T) {
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Normal bundle should pass independence check.
	if !bundle.ValidateIndependence() {
		t.Error("Valid bundle failed independence check")
	}

	// Nil bundle should fail.
	nilBundle := &IdentityBundle{}
	if nilBundle.ValidateIndependence() {
		t.Error("Nil keypairs should fail independence check")
	}
}

func TestIdentityBundleZero(t *testing.T) {
	bundle, err := GenerateIdentityBundleWithFortress()
	if err != nil {
		t.Fatalf("GenerateIdentityBundleWithFortress() error = %v", err)
	}

	// Save original key bytes to verify zeroing.
	surfacePriv := make([]byte, len(bundle.Surface.PrivateKey))
	copy(surfacePriv, bundle.Surface.PrivateKey)

	specterPriv := make([]byte, len(bundle.Specter.PrivateKey))
	copy(specterPriv, bundle.Specter.PrivateKey[:])

	// Zero the bundle.
	bundle.Zero()

	// Check that private keys are zeroed.
	for i, b := range bundle.Surface.PrivateKey {
		if b != 0 {
			t.Errorf("Surface private key[%d] not zeroed", i)
			break
		}
	}

	for i, b := range bundle.Specter.PrivateKey {
		if b != 0 {
			t.Errorf("Specter private key[%d] not zeroed", i)
			break
		}
	}

	for i, b := range bundle.FortressTransport.PrivateKey {
		if b != 0 {
			t.Errorf("Fortress private key[%d] not zeroed", i)
			break
		}
	}
}

func TestIdentityBundleKeypairIndependence(t *testing.T) {
	// Generate many bundles and verify keys are always independent.
	for i := 0; i < 10; i++ {
		bundle, err := GenerateIdentityBundleWithFortress()
		if err != nil {
			t.Fatalf("Iteration %d: GenerateIdentityBundleWithFortress() error = %v", i, err)
		}

		// Surface and Specter should be different.
		if bytes.Equal(bundle.Surface.PublicKey[:32], bundle.Specter.PublicKey[:]) {
			t.Errorf("Iteration %d: Surface and Specter public keys match", i)
		}

		// Surface and Fortress should be different.
		if bytes.Equal(bundle.Surface.PublicKey, bundle.FortressTransport.PublicKey) {
			t.Errorf("Iteration %d: Surface and Fortress public keys match", i)
		}

		bundle.Zero()
	}
}
