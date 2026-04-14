package waves

import (
	"bytes"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

// mockSigilSelector implements SigilSelector for testing.
type mockSigilSelector struct {
	specterPubKeys [][]byte
}

func newMockSigilSelector(count int) *mockSigilSelector {
	keys := make([][]byte, count)
	for i := 0; i < count; i++ {
		keys[i] = make([]byte, 32)
		for j := range keys[i] {
			keys[i][j] = byte(i*32 + j)
		}
	}
	return &mockSigilSelector{specterPubKeys: keys}
}

func (m *mockSigilSelector) SelectRandomSigil() []byte {
	if len(m.specterPubKeys) == 0 {
		return nil
	}
	// Return hash of first key for deterministic testing
	return m.specterPubKeys[0][:SigilHashSize]
}

func TestCreateSigil(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	selector := newMockSigilSelector(5)
	content := []byte("Test sigil wave content")
	opts := DefaultSigilOptions()
	opts.Difficulty = 1 // Low difficulty for fast tests

	wave, err := CreateSigil(content, kp, selector, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	// Verify wave properties.
	if wave.WaveType != pb.WaveType(TypeSigil) {
		t.Errorf("WaveType = %v, want %v", wave.WaveType, TypeSigil)
	}

	if !bytes.Equal(wave.Content, content) {
		t.Error("Content mismatch")
	}

	if !bytes.Equal(wave.AuthorPubkey, kp.PublicKey) {
		t.Error("AuthorPubkey mismatch")
	}

	// Verify sigil hash is present.
	sigilHash := GetSigilHash(wave)
	if len(sigilHash) != SigilHashSize {
		t.Errorf("SigilHash length = %d, want %d", len(sigilHash), SigilHashSize)
	}

	// Verify wave has ID and signature.
	if len(wave.WaveId) == 0 {
		t.Error("WaveId is empty")
	}
	if len(wave.Signature) == 0 {
		t.Error("Signature is empty")
	}
}

func TestCreateSigilNilKeyPair(t *testing.T) {
	_, err := CreateSigil([]byte("test"), nil, nil, DefaultSigilOptions())
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestCreateSigilContentTooLarge(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	content := make([]byte, MaxContentSize+1)
	_, err = CreateSigil(content, kp, nil, DefaultSigilOptions())
	if err != ErrContentTooLarge {
		t.Errorf("Expected ErrContentTooLarge, got %v", err)
	}
}

func TestCreateSigilInvalidTTL(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

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
			opts := SigilOptions{TTL: tt.ttl, Difficulty: 1}
			_, err := CreateSigil([]byte("test"), kp, nil, opts)
			if err != tt.want {
				t.Errorf("Expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestCreateSigilWithCustomHash(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	customHash := make([]byte, SigilHashSize)
	for i := range customHash {
		customHash[i] = byte(i * 7)
	}

	opts := SigilOptions{
		TTL:             DefaultTTL,
		Difficulty:      1,
		CustomSigilHash: customHash,
	}

	wave, err := CreateSigil([]byte("test"), kp, nil, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	sigilHash := GetSigilHash(wave)
	if !bytes.Equal(sigilHash, customHash) {
		t.Error("Custom sigil hash not used")
	}
}

func TestCreateSigilWithNilSelector(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	opts := DefaultSigilOptions()
	opts.Difficulty = 1

	// Should succeed with random hash when no selector
	wave, err := CreateSigil([]byte("test"), kp, nil, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	sigilHash := GetSigilHash(wave)
	if len(sigilHash) != SigilHashSize {
		t.Errorf("SigilHash length = %d, want %d", len(sigilHash), SigilHashSize)
	}
}

func TestIsSigil(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	opts := DefaultSigilOptions()
	opts.Difficulty = 1

	wave, err := CreateSigil([]byte("test"), kp, nil, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	if !IsSigil(wave) {
		t.Error("IsSigil() = false, want true")
	}

	// Test with non-sigil wave.
	surfaceWave := &pb.Wave{WaveType: pb.WaveType(TypeSurface)}
	if IsSigil(surfaceWave) {
		t.Error("IsSigil() = true for Surface wave, want false")
	}

	// Test with nil.
	if IsSigil(nil) {
		t.Error("IsSigil() = true for nil, want false")
	}
}

func TestGetSigilHash(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	customHash := make([]byte, SigilHashSize)
	for i := range customHash {
		customHash[i] = byte(i * 3)
	}

	opts := SigilOptions{
		TTL:             DefaultTTL,
		Difficulty:      1,
		CustomSigilHash: customHash,
	}

	wave, err := CreateSigil([]byte("test"), kp, nil, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	hash := GetSigilHash(wave)
	if !bytes.Equal(hash, customHash) {
		t.Error("GetSigilHash() returned wrong hash")
	}

	// Test with nil wave.
	if GetSigilHash(nil) != nil {
		t.Error("GetSigilHash(nil) should return nil")
	}

	// Test with wave without metadata.
	emptyWave := &pb.Wave{}
	if GetSigilHash(emptyWave) != nil {
		t.Error("GetSigilHash() should return nil for wave without metadata")
	}
}

func TestValidateSigil(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to create keypair: %v", err)
	}

	opts := DefaultSigilOptions()
	opts.Difficulty = 1

	wave, err := CreateSigil([]byte("test"), kp, nil, opts)
	if err != nil {
		t.Fatalf("CreateSigil() error = %v", err)
	}

	// Validation should pass.
	if err := ValidateSigil(wave, 1); err != nil {
		t.Errorf("ValidateSigil() error = %v", err)
	}
}

func TestValidateSigilNil(t *testing.T) {
	err := ValidateSigil(nil, 1)
	if err == nil {
		t.Error("Expected error for nil wave")
	}
}

func TestValidateSigilWrongType(t *testing.T) {
	wave := &pb.Wave{WaveType: pb.WaveType(TypeSurface)}
	err := ValidateSigil(wave, 1)
	if err == nil {
		t.Error("Expected error for wrong wave type")
	}
}

func TestRandomSigilSelector(t *testing.T) {
	keys := make([][]byte, 5)
	for i := range keys {
		keys[i] = make([]byte, 32)
		for j := range keys[i] {
			keys[i][j] = byte(i*32 + j)
		}
	}

	selector := NewRandomSigilSelector(keys)
	hash := selector.SelectRandomSigil()

	if len(hash) != SigilHashSize {
		t.Errorf("Hash length = %d, want %d", len(hash), SigilHashSize)
	}
}

func TestRandomSigilSelectorEmpty(t *testing.T) {
	selector := NewRandomSigilSelector(nil)
	hash := selector.SelectRandomSigil()

	if hash != nil {
		t.Error("Expected nil for empty selector")
	}
}

func TestGenerateRandomSigilHash(t *testing.T) {
	hash1, err := generateRandomSigilHash()
	if err != nil {
		t.Fatalf("generateRandomSigilHash() error = %v", err)
	}

	hash2, err := generateRandomSigilHash()
	if err != nil {
		t.Fatalf("generateRandomSigilHash() error = %v", err)
	}

	if len(hash1) != SigilHashSize {
		t.Errorf("Hash1 length = %d, want %d", len(hash1), SigilHashSize)
	}

	if len(hash2) != SigilHashSize {
		t.Errorf("Hash2 length = %d, want %d", len(hash2), SigilHashSize)
	}

	// Hashes should be different (with overwhelming probability).
	if bytes.Equal(hash1, hash2) {
		t.Error("Two random hashes should not be equal")
	}
}
