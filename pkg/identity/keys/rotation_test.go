package keys

import (
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestRotateKeyPair(t *testing.T) {
	// Generate an old keypair.
	oldKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate old keypair: %v", err)
	}
	defer oldKey.ZeroKeyPair()

	// Rotate the key.
	newKey, decl, err := RotateKeyPair(oldKey, 7, "Proactive rotation")
	if err != nil {
		t.Fatalf("RotateKeyPair failed: %v", err)
	}
	defer newKey.ZeroKeyPair()

	// Verify the declaration was created.
	if decl == nil {
		t.Fatal("ContinuityDeclaration is nil")
	}

	// Verify old and new public keys match.
	if !keysMatch(decl.OldPublicKey, oldKey.PublicKey) {
		t.Error("OldPublicKey in declaration doesn't match old keypair")
	}
	if !keysMatch(decl.NewPublicKey, newKey.PublicKey) {
		t.Error("NewPublicKey in declaration doesn't match new keypair")
	}

	// Verify grace period.
	if decl.GracePeriodDays != 7 {
		t.Errorf("expected GracePeriodDays=7, got %d", decl.GracePeriodDays)
	}

	// Verify reason.
	if decl.RotationReason != "Proactive rotation" {
		t.Errorf("expected RotationReason='Proactive rotation', got '%s'", decl.RotationReason)
	}

	// Verify timestamp is recent (within last 5 seconds).
	now := time.Now().Unix()
	if decl.RotationTimestampUnix < now-5 || decl.RotationTimestampUnix > now {
		t.Errorf("RotationTimestampUnix=%d is not recent (now=%d)", decl.RotationTimestampUnix, now)
	}

	// Verify signatures.
	sigData := buildRotationSignatureData(decl)

	if !Verify(oldKey.PublicKey, sigData, decl.OldKeySignature) {
		t.Error("OldKeySignature verification failed")
	}

	if !Verify(newKey.PublicKey, sigData, decl.NewKeySignature) {
		t.Error("NewKeySignature verification failed")
	}
}

func TestRotateKeyPairInvalidGracePeriod(t *testing.T) {
	oldKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate old keypair: %v", err)
	}
	defer oldKey.ZeroKeyPair()

	tests := []struct {
		name        string
		gracePeriod int64
	}{
		{"zero grace period", 0},
		{"negative grace period", -1},
		{"too long grace period", 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RotateKeyPair(oldKey, tt.gracePeriod, "Test")
			if err == nil {
				t.Errorf("expected error for gracePeriodDays=%d, got nil", tt.gracePeriod)
			}
		})
	}
}

func TestRotateKeyPairNilOldKey(t *testing.T) {
	_, _, err := RotateKeyPair(nil, 7, "Test")
	if err == nil {
		t.Error("expected error for nil old keypair, got nil")
	}
}

func TestRotateKeyPairDifferentKeys(t *testing.T) {
	oldKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate old keypair: %v", err)
	}
	defer oldKey.ZeroKeyPair()

	newKey, _, err := RotateKeyPair(oldKey, 7, "Test rotation")
	if err != nil {
		t.Fatalf("RotateKeyPair failed: %v", err)
	}
	defer newKey.ZeroKeyPair()

	// Verify old and new keys are different.
	if keysMatch(oldKey.PublicKey, newKey.PublicKey) {
		t.Error("new public key equals old public key (self-rotation)")
	}
}

func TestBuildRotationSignatureData(t *testing.T) {
	decl := &pb.ContinuityDeclaration{
		OldPublicKey:          make([]byte, 32),
		NewPublicKey:          make([]byte, 32),
		RotationTimestampUnix: 1234567890,
		GracePeriodDays:       7,
		RotationReason:        "Test reason",
	}

	// Fill with test data.
	for i := 0; i < 32; i++ {
		decl.OldPublicKey[i] = byte(i)
		decl.NewPublicKey[i] = byte(i + 32)
	}

	data := buildRotationSignatureData(decl)

	// Expected size: 32 (old) + 32 (new) + 8 (timestamp) + 8 (grace) + len("Test reason")
	expectedSize := 32 + 32 + 8 + 8 + len("Test reason")
	if len(data) != expectedSize {
		t.Errorf("expected data size %d, got %d", expectedSize, len(data))
	}

	// Verify old public key is first.
	for i := 0; i < 32; i++ {
		if data[i] != byte(i) {
			t.Errorf("old public key byte %d: expected %d, got %d", i, byte(i), data[i])
		}
	}

	// Verify new public key is next.
	for i := 0; i < 32; i++ {
		if data[32+i] != byte(i+32) {
			t.Errorf("new public key byte %d: expected %d, got %d", i, byte(i+32), data[32+i])
		}
	}
}
