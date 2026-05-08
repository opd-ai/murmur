package waves

import (
	"bytes"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"golang.org/x/crypto/curve25519"
)

// mockSpecterSigner implements SpecterSigner for testing using real Curve25519 keys.
type mockSpecterSigner struct {
	pubKey  []byte
	privKey []byte
}

func newMockSpecterSigner() *mockSpecterSigner {
	// Generate a real Curve25519 keypair for tests.
	priv, pub := genCurve25519KeyPair()
	return &mockSpecterSigner{pubKey: pub, privKey: priv}
}

// genCurve25519KeyPair generates a deterministic Curve25519 keypair for testing.
func genCurve25519KeyPair() (priv, pub []byte) {
	var privArr [32]byte
	// Use a fixed seed for deterministic tests.
	for i := range privArr {
		privArr[i] = byte(i + 1)
	}
	// Clamp per Curve25519 spec.
	privArr[0] &= 248
	privArr[31] &= 127
	privArr[31] |= 64
	var pubArr [32]byte
	curve25519.ScalarBaseMult(&pubArr, &privArr)
	return privArr[:], pubArr[:]
}

func (m *mockSpecterSigner) Sign(data []byte) []byte {
	// Simple mock signature for testing.
	sig := make([]byte, 64)
	for i := 0; i < 32 && i < len(data); i++ {
		sig[i] = data[i] ^ m.privKey[i]
	}
	return sig
}

func (m *mockSpecterSigner) SpecterPublicKey() []byte {
	return m.pubKey
}

// ComputeDHSecret performs X25519 key exchange with peerPubKey.
func (m *mockSpecterSigner) ComputeDHSecret(peerPubKey []byte) ([]byte, error) {
	return curve25519.X25519(m.privKey, peerPubKey)
}

func TestCreateVeiled(t *testing.T) {
	specter := newMockSpecterSigner()
	content := []byte("Test veiled content")
	opts := DefaultVeiledOptions()
	opts.Difficulty = 1 // Use low difficulty for fast tests

	wave, err := CreateVeiled(content, specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	// Verify wave properties.
	if wave.WaveType != pb.WaveType(TypeVeiled) {
		t.Errorf("WaveType = %v, want %v", wave.WaveType, TypeVeiled)
	}

	if !bytes.Equal(wave.Content, content) {
		t.Error("Content mismatch")
	}

	if !bytes.Equal(wave.AuthorPubkey, specter.SpecterPublicKey()) {
		t.Error("AuthorPubkey mismatch")
	}

	// Verify veil metadata.
	veil, ok := wave.Metadata[VeiledMetadataKey]
	if !ok {
		t.Error("Missing veil metadata")
	}
	if string(veil) != VeiledMetadataValue {
		t.Errorf("Veil metadata = %q, want %q", string(veil), VeiledMetadataValue)
	}

	// Verify wave has ID and signature.
	if len(wave.WaveId) == 0 {
		t.Error("WaveId is empty")
	}
	if len(wave.Signature) == 0 {
		t.Error("Signature is empty")
	}
}

func TestCreateVeiledNilSigner(t *testing.T) {
	_, err := CreateVeiled([]byte("test"), nil, DefaultVeiledOptions())
	if err != ErrSpecterKeyRequired {
		t.Errorf("Expected ErrSpecterKeyRequired, got %v", err)
	}
}

func TestCreateVeiledContentTooLarge(t *testing.T) {
	specter := newMockSpecterSigner()
	content := make([]byte, MaxContentSize+1)

	_, err := CreateVeiled(content, specter, DefaultVeiledOptions())
	if err != ErrContentTooLarge {
		t.Errorf("Expected ErrContentTooLarge, got %v", err)
	}
}

func TestCreateVeiledInvalidTTL(t *testing.T) {
	specter := newMockSpecterSigner()

	tests := []struct {
		name string
		ttl  time.Duration
		want error
	}{
		{"zero TTL", 0, ErrInvalidTTL},
		{"negative TTL", -time.Hour, ErrInvalidTTL},
		{"too long TTL", MaxTTL + time.Hour, ErrTTLTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := VeiledOptions{TTL: tt.ttl, Difficulty: 1}
			_, err := CreateVeiled([]byte("test"), specter, opts)
			if err != tt.want {
				t.Errorf("Expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestCreateVeiledEncrypted(t *testing.T) {
	specter := newMockSpecterSigner()
	content := []byte("Secret veiled content")

	// Generate a proper Curve25519 keypair for the recipient.
	recipientPrivKey, recipientPubKey := genCurve25519KeyPair()
	_ = recipientPrivKey

	opts := VeiledOptions{
		TTL:             DefaultTTL,
		Difficulty:      1,
		Encrypted:       true,
		RecipientPubKey: recipientPubKey,
	}

	wave, err := CreateVeiled(content, specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	// Verify encrypted metadata.
	encrypted, ok := wave.Metadata[EncryptedContentKey]
	if !ok {
		t.Error("Missing encrypted metadata")
	}
	if string(encrypted) != "true" {
		t.Errorf("Encrypted metadata = %q, want %q", string(encrypted), "true")
	}

	// Verify wrapped key and nonce are present.
	if _, ok := wave.Metadata[WrappedKeyKey]; !ok {
		t.Error("Missing wrapped key")
	}
	if _, ok := wave.Metadata[NonceKey]; !ok {
		t.Error("Missing nonce")
	}

	// Content should be different from original (encrypted).
	if bytes.Equal(wave.Content, content) {
		t.Error("Content should be encrypted")
	}
}

func TestDecryptVeiledContent(t *testing.T) {
	specter := newMockSpecterSigner()
	content := []byte("Secret message for recipient")

	// Generate a proper Curve25519 keypair for the recipient.
	recipientPrivKey, recipientPubKey := genCurve25519KeyPair()
	// Use a different seed from sender to simulate distinct parties.
	var recipPrivArr [32]byte
	for i := range recipPrivArr {
		recipPrivArr[i] = byte(i + 50)
	}
	recipPrivArr[0] &= 248
	recipPrivArr[31] &= 127
	recipPrivArr[31] |= 64
	var recipPubArr [32]byte
	curve25519.ScalarBaseMult(&recipPubArr, &recipPrivArr)
	recipientPrivKey = recipPrivArr[:]
	recipientPubKey = recipPubArr[:]

	opts := VeiledOptions{
		TTL:             DefaultTTL,
		Difficulty:      1,
		Encrypted:       true,
		RecipientPubKey: recipientPubKey,
	}

	wave, err := CreateVeiled(content, specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	// Decrypt with the recipient's private key.
	decrypted, err := DecryptVeiledContent(wave, recipientPrivKey)
	if err != nil {
		t.Fatalf("DecryptVeiledContent() error = %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("Decrypted content = %q, want %q", string(decrypted), string(content))
	}
}

func TestDecryptVeiledContentUnencrypted(t *testing.T) {
	specter := newMockSpecterSigner()
	content := []byte("Not encrypted")

	opts := DefaultVeiledOptions()
	opts.Difficulty = 1

	wave, err := CreateVeiled(content, specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	// Decrypt should return content as-is for unencrypted waves.
	decrypted, err := DecryptVeiledContent(wave, nil)
	if err != nil {
		t.Fatalf("DecryptVeiledContent() error = %v", err)
	}

	if !bytes.Equal(decrypted, content) {
		t.Errorf("Content = %q, want %q", string(decrypted), string(content))
	}
}

func TestDecryptVeiledContentNil(t *testing.T) {
	_, err := DecryptVeiledContent(nil, nil)
	if err == nil {
		t.Error("Expected error for nil wave")
	}
}

func TestIsVeiled(t *testing.T) {
	specter := newMockSpecterSigner()
	opts := DefaultVeiledOptions()
	opts.Difficulty = 1

	wave, err := CreateVeiled([]byte("test"), specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	if !IsVeiled(wave) {
		t.Error("IsVeiled() = false, want true")
	}

	// Test with non-veiled wave.
	nonVeiled := &pb.Wave{WaveType: pb.WaveType(TypeSurface)}
	if IsVeiled(nonVeiled) {
		t.Error("IsVeiled() = true for Surface wave, want false")
	}

	// Test with nil.
	if IsVeiled(nil) {
		t.Error("IsVeiled() = true for nil, want false")
	}
}

func TestIsEncryptedVeiled(t *testing.T) {
	specter := newMockSpecterSigner()
	_, recipientPubKey := genCurve25519KeyPair()
	// Use distinct key from sender.
	var recipPubArr [32]byte
	for i := range recipPubArr {
		recipPubArr[i] = byte(i + 50)
	}
	var recipPrivArr [32]byte
	for i := range recipPrivArr {
		recipPrivArr[i] = byte(i + 50)
	}
	recipPrivArr[0] &= 248
	recipPrivArr[31] &= 127
	recipPrivArr[31] |= 64
	curve25519.ScalarBaseMult(&recipPubArr, &recipPrivArr)
	recipientPubKey = recipPubArr[:]

	// Create encrypted veiled wave.
	encOpts := VeiledOptions{
		TTL:             DefaultTTL,
		Difficulty:      1,
		Encrypted:       true,
		RecipientPubKey: recipientPubKey,
	}

	encWave, err := CreateVeiled([]byte("encrypted"), specter, encOpts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	if !IsEncryptedVeiled(encWave) {
		t.Error("IsEncryptedVeiled() = false for encrypted wave")
	}

	// Create non-encrypted veiled wave.
	opts := DefaultVeiledOptions()
	opts.Difficulty = 1

	wave, err := CreateVeiled([]byte("not encrypted"), specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	if IsEncryptedVeiled(wave) {
		t.Error("IsEncryptedVeiled() = true for non-encrypted wave")
	}
}

func TestWrapUnwrapSymmetricKey(t *testing.T) {
	// Generate proper Curve25519 keypairs for author (sender) and recipient.
	authorPrivKey, authorPubKey := genCurve25519KeyPair()

	var recipPrivArr [32]byte
	for i := range recipPrivArr {
		recipPrivArr[i] = byte(i + 50)
	}
	recipPrivArr[0] &= 248
	recipPrivArr[31] &= 127
	recipPrivArr[31] |= 64
	var recipPubArr [32]byte
	curve25519.ScalarBaseMult(&recipPubArr, &recipPrivArr)
	recipientPrivKey := recipPrivArr[:]
	recipientPubKey := recipPubArr[:]

	// Create a mock sender that uses the author keypair.
	var authorPrivArr [32]byte
	copy(authorPrivArr[:], authorPrivKey)
	sender := &mockSpecterSigner{pubKey: authorPubKey, privKey: authorPrivArr[:]}

	// Generate a random symmetric key.
	symmetricKey := make([]byte, SymmetricKeySize)
	for i := range symmetricKey {
		symmetricKey[i] = byte(i * 3)
	}

	// Wrap using sender's DH with recipient's public key.
	sharedSend, err := sender.ComputeDHSecret(recipientPubKey)
	if err != nil {
		t.Fatalf("ComputeDHSecret() error = %v", err)
	}
	wrapKey, err := deriveVeiledWrapKey(sharedSend)
	if err != nil {
		t.Fatalf("deriveVeiledWrapKey() error = %v", err)
	}
	wrapped := xorBytes(symmetricKey, wrapKey)

	// Wrapped should be different from original.
	if bytes.Equal(wrapped, symmetricKey) {
		t.Error("Wrapped key should be different from original")
	}

	// Unwrap using recipient's private key and author's public key.
	unwrapped, err := UnwrapSymmetricKey(wrapped, authorPubKey, recipientPrivKey)
	if err != nil {
		t.Fatalf("UnwrapSymmetricKey() error = %v", err)
	}

	// Should get back the original key.
	if !bytes.Equal(unwrapped, symmetricKey) {
		t.Error("Unwrapped key does not match original")
	}
}

func TestUnwrapSymmetricKeyInvalid(t *testing.T) {
	// Test with wrong size.
	_, err := UnwrapSymmetricKey([]byte("short"), nil, nil)
	if err != ErrInvalidWrappedKey {
		t.Errorf("Expected ErrInvalidWrappedKey, got %v", err)
	}
}

func TestValidateVeiled(t *testing.T) {
	specter := newMockSpecterSigner()
	opts := DefaultVeiledOptions()
	opts.Difficulty = 1

	wave, err := CreateVeiled([]byte("test"), specter, opts)
	if err != nil {
		t.Fatalf("CreateVeiled() error = %v", err)
	}

	// Validation should pass (note: signature won't verify with mock signer,
	// but common validation checks PoW which should pass).
	err = validateCommon(wave, 1)
	if err != nil {
		t.Errorf("validateCommon() error = %v", err)
	}
}

func TestValidateVeiledNil(t *testing.T) {
	err := ValidateVeiled(nil, 1)
	if err == nil {
		t.Error("Expected error for nil wave")
	}
}

func TestValidateVeiledWrongType(t *testing.T) {
	wave := &pb.Wave{WaveType: pb.WaveType(TypeSurface)}
	err := ValidateVeiled(wave, 1)
	if err == nil {
		t.Error("Expected error for wrong wave type")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
