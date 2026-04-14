package waves

import (
	"bytes"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestCreateAmplification(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	// Create an original wave.
	original := createTestSurfaceWave(t, authorKp, []byte("Original content"))

	// Create amplification.
	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	// Verify amplification properties.
	if amp.OriginalWave == nil {
		t.Error("OriginalWave is nil")
	}
	if !bytes.Equal(amp.AmplifierPubkey, amplifierKp.PublicKey) {
		t.Error("AmplifierPubkey mismatch")
	}
	if amp.AmplifiedAt == 0 {
		t.Error("AmplifiedAt is zero")
	}
	if len(amp.Signature) == 0 {
		t.Error("Signature is empty")
	}

	// Hop count should be incremented.
	if amp.OriginalWave.HopCount != original.HopCount+1 {
		t.Errorf("HopCount = %d, want %d", amp.OriginalWave.HopCount, original.HopCount+1)
	}
}

func TestCreateAmplificationWithComment(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))
	comment := []byte("Great post!")

	amp, err := CreateAmplificationWithComment(original, amplifierKp, comment)
	if err != nil {
		t.Fatalf("CreateAmplificationWithComment() error = %v", err)
	}

	if !bytes.Equal(amp.Comment, comment) {
		t.Error("Comment mismatch")
	}
	if !HasComment(amp) {
		t.Error("HasComment() = false, want true")
	}
}

func TestCreateAmplificationCommentTooLong(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))
	longComment := make([]byte, AmplificationMaxComment+1)

	_, err := CreateAmplificationWithComment(original, amplifierKp, longComment)
	if err != ErrCommentTooLong {
		t.Errorf("Expected ErrCommentTooLong, got %v", err)
	}
}

func TestCreateAmplificationSelfAmplification(t *testing.T) {
	kp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, kp, []byte("My own content"))

	_, err := CreateAmplification(original, kp, DefaultAmplificationOptions())
	if err != ErrSelfAmplification {
		t.Errorf("Expected ErrSelfAmplification, got %v", err)
	}
}

func TestCreateAmplificationNilWave(t *testing.T) {
	kp := generateTestKeyPair(t)

	_, err := CreateAmplification(nil, kp, DefaultAmplificationOptions())
	if err != ErrNilOriginalWave {
		t.Errorf("Expected ErrNilOriginalWave, got %v", err)
	}
}

func TestCreateAmplificationNilKeyPair(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	_, err := CreateAmplification(original, nil, DefaultAmplificationOptions())
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestCreateAmplificationExpiredWave(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	// Create a wave with a very short TTL.
	opts := DefaultCreateOptions()
	opts.Difficulty = 1
	opts.TTL = 1 * time.Nanosecond

	original, err := Create(TypeSurface, []byte("Short-lived"), authorKp, opts)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait for it to expire.
	time.Sleep(10 * time.Millisecond)

	_, err = CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != ErrAmplificationExpired {
		t.Errorf("Expected ErrAmplificationExpired, got %v", err)
	}
}

func TestCreateAmplificationWithHopReset(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	// Create wave with some hops.
	original := createTestSurfaceWave(t, authorKp, []byte("Original"))
	original.HopCount = 5

	amp, err := CreateAmplificationWithHopReset(original, amplifierKp)
	if err != nil {
		t.Fatalf("CreateAmplificationWithHopReset() error = %v", err)
	}

	// Hop count should be reset to 0.
	if amp.OriginalWave.HopCount != 0 {
		t.Errorf("HopCount = %d, want 0", amp.OriginalWave.HopCount)
	}
}

func TestValidateAmplification(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	// Validation should pass.
	if err := ValidateAmplification(amp, 1); err != nil {
		t.Errorf("ValidateAmplification() error = %v", err)
	}
}

func TestValidateAmplificationNil(t *testing.T) {
	err := ValidateAmplification(nil, 1)
	if err != ErrInvalidAmplification {
		t.Errorf("Expected ErrInvalidAmplification, got %v", err)
	}
}

func TestValidateAmplificationNilOriginal(t *testing.T) {
	amp := &pb.Amplification{
		OriginalWave: nil,
	}

	err := ValidateAmplification(amp, 1)
	if err != ErrNilOriginalWave {
		t.Errorf("Expected ErrNilOriginalWave, got %v", err)
	}
}

func TestValidateAmplificationBadSignature(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	// Corrupt the signature.
	amp.Signature[0] ^= 0xFF

	err = ValidateAmplification(amp, 1)
	if err != ErrInvalidSig {
		t.Errorf("Expected ErrInvalidSig, got %v", err)
	}
}

func TestGetAmplifiedWaveID(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	waveID := GetAmplifiedWaveID(amp)
	if !bytes.Equal(waveID, original.WaveId) {
		t.Error("WaveID mismatch")
	}

	// Test with nil.
	if GetAmplifiedWaveID(nil) != nil {
		t.Error("GetAmplifiedWaveID(nil) should return nil")
	}
}

func TestGetAmplifierPubkey(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	pubkey := GetAmplifierPubkey(amp)
	if !bytes.Equal(pubkey, amplifierKp.PublicKey) {
		t.Error("Pubkey mismatch")
	}

	if GetAmplifierPubkey(nil) != nil {
		t.Error("GetAmplifierPubkey(nil) should return nil")
	}
}

func TestGetAmplificationTime(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	before := time.Now().Add(-1 * time.Second) // Allow 1 second margin
	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}
	after := time.Now().Add(1 * time.Second) // Allow 1 second margin

	ampTime := GetAmplificationTime(amp)
	if ampTime.Before(before) || ampTime.After(after) {
		t.Errorf("AmplificationTime %v out of expected range [%v, %v]", ampTime, before, after)
	}

	if !GetAmplificationTime(nil).IsZero() {
		t.Error("GetAmplificationTime(nil) should return zero time")
	}
}

func TestAmplificationChain(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	chain := NewAmplificationChain(original.WaveId)

	// Add some amplifications.
	for i := 0; i < 3; i++ {
		amplifierKp := generateTestKeyPair(t)
		amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
		if err != nil {
			t.Fatalf("CreateAmplification() error = %v", err)
		}

		if err := chain.Add(amp); err != nil {
			t.Fatalf("chain.Add() error = %v", err)
		}
	}

	// Verify count.
	if chain.Count() != 3 {
		t.Errorf("Count() = %d, want 3", chain.Count())
	}

	// Verify amplifiers.
	amplifiers := chain.GetAmplifiers()
	if len(amplifiers) != 3 {
		t.Errorf("GetAmplifiers() length = %d, want 3", len(amplifiers))
	}
}

func TestAmplificationChainDuplicate(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	chain := NewAmplificationChain(original.WaveId)

	amp, err := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	if err != nil {
		t.Fatalf("CreateAmplification() error = %v", err)
	}

	// First add should succeed.
	if err := chain.Add(amp); err != nil {
		t.Fatalf("First Add() error = %v", err)
	}

	// Second add should fail (duplicate).
	err = chain.Add(amp)
	if err != ErrDuplicateAmplifier {
		t.Errorf("Expected ErrDuplicateAmplifier, got %v", err)
	}
}

func TestAmplificationChainHasAmplifier(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)
	otherKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))

	chain := NewAmplificationChain(original.WaveId)

	amp, _ := CreateAmplification(original, amplifierKp, DefaultAmplificationOptions())
	chain.Add(amp)

	if !chain.HasAmplifier(amplifierKp.PublicKey) {
		t.Error("HasAmplifier() = false, want true")
	}

	if chain.HasAmplifier(otherKp.PublicKey) {
		t.Error("HasAmplifier() = true for non-amplifier, want false")
	}
}

func TestCloneWave(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	original := createTestSurfaceWave(t, authorKp, []byte("Original"))
	original.Metadata = map[string][]byte{"key": []byte("value")}

	clone := cloneWave(original)

	// Verify clone is independent.
	if clone == original {
		t.Error("Clone should be a new instance")
	}

	// Verify values match.
	if !bytes.Equal(clone.WaveId, original.WaveId) {
		t.Error("WaveId mismatch")
	}
	if !bytes.Equal(clone.Content, original.Content) {
		t.Error("Content mismatch")
	}

	// Verify metadata is copied.
	if !bytes.Equal(clone.Metadata["key"], original.Metadata["key"]) {
		t.Error("Metadata mismatch")
	}

	// Modify clone and verify original is unchanged.
	clone.Content[0] = 'X'
	if bytes.Equal(clone.Content, original.Content) {
		t.Error("Modifying clone should not affect original")
	}
}

func TestCloneWaveNil(t *testing.T) {
	clone := cloneWave(nil)
	if clone != nil {
		t.Error("cloneWave(nil) should return nil")
	}
}

func TestDefaultAmplificationOptions(t *testing.T) {
	opts := DefaultAmplificationOptions()

	if opts.Comment != nil {
		t.Error("Default Comment should be nil")
	}
	if opts.ResetHops {
		t.Error("Default ResetHops should be false")
	}
	if !opts.SkipPoW {
		t.Error("Default SkipPoW should be true")
	}
}

func TestAmplifyAndIncrement(t *testing.T) {
	authorKp := generateTestKeyPair(t)
	amplifierKp := generateTestKeyPair(t)

	original := createTestSurfaceWave(t, authorKp, []byte("Original"))
	original.HopCount = 3

	amp, err := AmplifyAndIncrement(original, amplifierKp)
	if err != nil {
		t.Fatalf("AmplifyAndIncrement() error = %v", err)
	}

	// Hop count should be incremented from 3 to 4.
	if amp.OriginalWave.HopCount != 4 {
		t.Errorf("HopCount = %d, want 4", amp.OriginalWave.HopCount)
	}
}

func TestAmplificationChainNilHandling(t *testing.T) {
	var chain *AmplificationChain

	if chain.Count() != 0 {
		t.Error("Count() on nil chain should return 0")
	}
	if chain.HasAmplifier([]byte{1, 2, 3}) {
		t.Error("HasAmplifier() on nil chain should return false")
	}
	if chain.GetAmplifiers() != nil {
		t.Error("GetAmplifiers() on nil chain should return nil")
	}

	// Add should fail gracefully.
	err := chain.Add(&pb.Amplification{})
	if err != ErrInvalidAmplification {
		t.Errorf("Add() on nil chain should return ErrInvalidAmplification, got %v", err)
	}
}
