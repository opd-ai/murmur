package waves

import (
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
)

func TestCreateSurface(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Hello, MURMUR!")

	// Use low difficulty for faster tests.
	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if wave == nil {
		t.Fatal("wave is nil")
	}

	if string(wave.Content) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", wave.Content, content)
	}

	if len(wave.WaveId) == 0 {
		t.Error("wave ID is empty")
	}

	if len(wave.Signature) == 0 {
		t.Error("signature is empty")
	}
}

func TestCreateValidate(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("Test content for validation")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, content, kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Validate with same difficulty.
	if err := Validate(wave, 8); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestCreateContentTooLarge(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := make([]byte, MaxContentSize+1)

	_, err = Create(TypeSurface, content, kp, DefaultCreateOptions())
	if err != ErrContentTooLarge {
		t.Errorf("expected ErrContentTooLarge, got %v", err)
	}
}

func TestCreateTTLValidation(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	content := []byte("TTL test")

	// Zero TTL should fail.
	opts := DefaultCreateOptions()
	opts.TTL = 0
	_, err = Create(TypeSurface, content, kp, opts)
	if err != ErrInvalidTTL {
		t.Errorf("expected ErrInvalidTTL for zero TTL, got %v", err)
	}

	// Negative TTL should fail.
	opts.TTL = -1 * time.Hour
	_, err = Create(TypeSurface, content, kp, opts)
	if err != ErrInvalidTTL {
		t.Errorf("expected ErrInvalidTTL for negative TTL, got %v", err)
	}

	// TTL too long should fail.
	opts.TTL = MaxTTL + time.Hour
	_, err = Create(TypeSurface, content, kp, opts)
	if err != ErrTTLTooLong {
		t.Errorf("expected ErrTTLTooLong, got %v", err)
	}
}

func TestCreateNilKeyPair(t *testing.T) {
	_, err := Create(TypeSurface, []byte("test"), nil, DefaultCreateOptions())
	if err != ErrNilKeyPair {
		t.Errorf("expected ErrNilKeyPair, got %v", err)
	}
}

func TestValidateInvalidSignature(t *testing.T) {
	kp1, _ := keys.GenerateKeyPair()
	kp2, _ := keys.GenerateKeyPair()

	content := []byte("Signature test")

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, content, kp1, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Replace author with different key.
	wave.AuthorPubkey = kp2.PublicKey

	err = Validate(wave, 8)
	if err != ErrInvalidSig {
		t.Errorf("expected ErrInvalidSig, got %v", err)
	}
}

func TestValidateInvalidPoW(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, []byte("PoW test"), kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Validate with higher difficulty should fail.
	err = Validate(wave, 16)
	if err != ErrInvalidPoW {
		t.Errorf("expected ErrInvalidPoW, got %v", err)
	}
}

func TestIsExpired(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, []byte("Expiry test"), kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Fresh wave should not be expired.
	if IsExpired(wave) {
		t.Error("fresh wave should not be expired")
	}

	// Set created_at to past to simulate expiration.
	wave.CreatedAt = time.Now().Add(-8 * 24 * time.Hour).Unix()
	if !IsExpired(wave) {
		t.Error("old wave should be expired")
	}
}

func TestExpiresAt(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	opts := DefaultCreateOptions()
	opts.TTL = 7 * 24 * time.Hour
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, []byte("ExpiresAt test"), kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	expiry := ExpiresAt(wave)
	expected := time.Unix(wave.CreatedAt, 0).Add(7 * 24 * time.Hour)

	if !expiry.Equal(expected) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", expiry, expected)
	}
}

func TestIncrementHop(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := Create(TypeSurface, []byte("Hop test"), kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if wave.HopCount != 0 {
		t.Errorf("initial hop count should be 0, got %d", wave.HopCount)
	}

	wave2 := IncrementHop(wave)
	if wave2.HopCount != 1 {
		t.Errorf("hop count after increment should be 1, got %d", wave2.HopCount)
	}

	// Original should be unchanged.
	if wave.HopCount != 0 {
		t.Error("original wave hop count was modified")
	}

	wave3 := IncrementHop(wave2)
	if wave3.HopCount != 2 {
		t.Errorf("hop count after second increment should be 2, got %d", wave3.HopCount)
	}
}

func TestCreateReply(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	// Create parent wave.
	opts := DefaultCreateOptions()
	opts.Difficulty = 8
	parent, err := Create(TypeSurface, []byte("Parent wave"), kp, opts)
	if err != nil {
		t.Fatalf("Create parent failed: %v", err)
	}

	// Create reply using CreateReply helper (but with lower difficulty).
	replyOpts := DefaultCreateOptions()
	replyOpts.ParentHash = parent.WaveId
	replyOpts.Difficulty = 8
	reply, err := Create(TypeReply, []byte("Reply content"), kp, replyOpts)
	if err != nil {
		t.Fatalf("CreateReply failed: %v", err)
	}

	if len(reply.ParentHash) == 0 {
		t.Error("reply should have parent hash")
	}

	if string(reply.ParentHash) != string(parent.WaveId) {
		t.Error("reply parent hash should match parent wave ID")
	}
}

func TestWaveTypes(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	types := []WaveType{
		TypeSurface,
		TypeReply,
		TypeVeiled,
		TypeSpecter,
		TypeSigil,
		TypeAbyssal,
		TypeMasked,
		TypeBeacon,
	}

	opts := DefaultCreateOptions()
	opts.Difficulty = 8

	for _, wt := range types {
		wave, err := Create(wt, []byte("Type test"), kp, opts)
		if err != nil {
			t.Errorf("Create(%d) failed: %v", wt, err)
			continue
		}

		if uint8(wave.WaveType) != uint8(wt) {
			t.Errorf("wave type mismatch: got %d, want %d", wave.WaveType, wt)
		}
	}
}

func TestValidateNil(t *testing.T) {
	err := Validate(nil, 8)
	if err == nil {
		t.Error("expected error for nil wave")
	}
}

func TestIncrementHopNil(t *testing.T) {
	result := IncrementHop(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestExpiresAtNil(t *testing.T) {
	result := ExpiresAt(nil)
	if !result.IsZero() {
		t.Error("expected zero time for nil input")
	}
}

// --- Abyssal Wave Tests ---

func TestDeriveAbyssalKeyPair(t *testing.T) {
	// Generate a Specter keypair.
	specterKP, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Derive Abyssal keypair.
	akp, err := DeriveAbyssalKeyPair(specterKP.PrivateKey)
	if err != nil {
		t.Fatalf("DeriveAbyssalKeyPair failed: %v", err)
	}

	// Keys should be different from Specter keys.
	if string(akp.PublicKey) == string(specterKP.PublicKey) {
		t.Error("Abyssal public key should differ from Specter public key")
	}

	// Nonce should be non-zero.
	var zeroNonce [32]byte
	if akp.Nonce == zeroNonce {
		t.Error("Nonce should be non-zero")
	}
}

func TestDeriveAbyssalKeyPairUniqueness(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	// Derive two keypairs - should be different.
	akp1, _ := DeriveAbyssalKeyPair(specterKP.PrivateKey)
	akp2, _ := DeriveAbyssalKeyPair(specterKP.PrivateKey)

	if string(akp1.PublicKey) == string(akp2.PublicKey) {
		t.Error("Each derivation should produce unique keys")
	}

	if akp1.Nonce == akp2.Nonce {
		t.Error("Each derivation should have unique nonce")
	}
}

func TestDeriveAbyssalKeyPairInvalidKey(t *testing.T) {
	_, err := DeriveAbyssalKeyPair([]byte("invalid"))
	if err != ErrAbyssalInvalidKey {
		t.Errorf("Expected ErrAbyssalInvalidKey, got %v", err)
	}
}

func TestCreateAbyssalWave(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	opts := DefaultAbyssalOptions()
	opts.Difficulty = 8 // Lower for testing.

	abyssalWave, err := CreateAbyssal(
		[]byte("Deeply anonymous content"),
		specterKP.PrivateKey,
		opts,
	)
	if err != nil {
		t.Fatalf("CreateAbyssal failed: %v", err)
	}

	if abyssalWave.Wave == nil {
		t.Fatal("Wave is nil")
	}

	// Author key should be one-time key, not Specter key.
	if string(abyssalWave.Wave.AuthorPubkey) == string(specterKP.PublicKey) {
		t.Error("Author should be one-time key, not Specter key")
	}

	// Author key should match derived keypair.
	if string(abyssalWave.Wave.AuthorPubkey) != string(abyssalWave.KeyPair.PublicKey) {
		t.Error("Author key should match derived keypair")
	}

	// Wave type should be Abyssal.
	if abyssalWave.Wave.WaveType != 0x06 {
		t.Errorf("Expected type 0x06, got %d", abyssalWave.Wave.WaveType)
	}
}

func TestValidateAbyssalWave(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	opts := DefaultAbyssalOptions()
	opts.Difficulty = 8

	abyssalWave, err := CreateAbyssal(
		[]byte("Test validation"),
		specterKP.PrivateKey,
		opts,
	)
	if err != nil {
		t.Fatalf("CreateAbyssal failed: %v", err)
	}

	// Validate.
	err = ValidateAbyssal(abyssalWave.Wave, 8)
	if err != nil {
		t.Errorf("ValidateAbyssal failed: %v", err)
	}
}

func TestAbyssalWaveContentTooLarge(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	largeContent := make([]byte, MaxContentSize+1)
	_, err := CreateAbyssal(largeContent, specterKP.PrivateKey, DefaultAbyssalOptions())
	if err != ErrContentTooLarge {
		t.Errorf("Expected ErrContentTooLarge, got %v", err)
	}
}

func TestCanProveAuthorship(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	opts := DefaultAbyssalOptions()
	opts.Difficulty = 8

	abyssalWave, _ := CreateAbyssal(
		[]byte("Prove authorship"),
		specterKP.PrivateKey,
		opts,
	)

	// Can prove with correct nonce.
	canProve := CanProveAuthorship(
		abyssalWave.Wave,
		specterKP.PrivateKey,
		abyssalWave.KeyPair.Nonce,
	)
	if !canProve {
		t.Error("Should be able to prove authorship with correct nonce")
	}

	// Cannot prove with wrong nonce.
	var wrongNonce [32]byte
	wrongNonce[0] = 0xFF
	canProve = CanProveAuthorship(
		abyssalWave.Wave,
		specterKP.PrivateKey,
		wrongNonce,
	)
	if canProve {
		t.Error("Should not be able to prove authorship with wrong nonce")
	}

	// Cannot prove with different Specter key.
	otherKP, _ := keys.GenerateKeyPair()
	canProve = CanProveAuthorship(
		abyssalWave.Wave,
		otherKP.PrivateKey,
		abyssalWave.KeyPair.Nonce,
	)
	if canProve {
		t.Error("Should not be able to prove with different Specter key")
	}
}

func TestAbyssalStore(t *testing.T) {
	store := NewAbyssalStore()

	waveID := []byte("test-wave-id")
	var nonce [32]byte
	nonce[0] = 0x01

	store.StoreNonce(waveID, nonce)

	if store.Count() != 1 {
		t.Errorf("Expected 1 nonce, got %d", store.Count())
	}

	retrieved, found := store.GetNonce(waveID)
	if !found {
		t.Error("Expected to find nonce")
	}
	if retrieved != nonce {
		t.Error("Retrieved nonce doesn't match")
	}

	store.RemoveNonce(waveID)
	if store.Count() != 0 {
		t.Error("Expected 0 nonces after removal")
	}
}

func TestAbyssalWaveID(t *testing.T) {
	specterKP, _ := keys.GenerateKeyPair()

	opts := DefaultAbyssalOptions()
	opts.Difficulty = 8

	wave1, _ := CreateAbyssal([]byte("Content 1"), specterKP.PrivateKey, opts)
	wave2, _ := CreateAbyssal([]byte("Content 2"), specterKP.PrivateKey, opts)

	id1 := AbyssalWaveID(wave1.Wave)
	id2 := AbyssalWaveID(wave2.Wave)

	// Different content should produce different IDs.
	if string(id1) == string(id2) {
		t.Error("Different waves should have different IDs")
	}

	// Nil input should return nil.
	if AbyssalWaveID(nil) != nil {
		t.Error("Expected nil for nil input")
	}
}
