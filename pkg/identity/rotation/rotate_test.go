package rotation

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestCreateRotation(t *testing.T) {
	t.Run("success_default_options", func(t *testing.T) {
		_, oldPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey failed: %v", err)
		}
		_, newPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey failed: %v", err)
		}

		decl, err := CreateRotation(oldPriv, newPriv, nil)
		if err != nil {
			t.Fatalf("CreateRotation failed: %v", err)
		}

		// Validate structure
		if len(decl.OldPublicKey) != Ed25519PublicKeySize {
			t.Errorf("old public key size = %d, want %d", len(decl.OldPublicKey), Ed25519PublicKeySize)
		}
		if len(decl.NewPublicKey) != Ed25519PublicKeySize {
			t.Errorf("new public key size = %d, want %d", len(decl.NewPublicKey), Ed25519PublicKeySize)
		}
		if decl.GracePeriodDays != DefaultGracePeriodDays {
			t.Errorf("grace period = %d, want %d", decl.GracePeriodDays, DefaultGracePeriodDays)
		}
		if decl.RotationReason != "Proactive rotation" {
			t.Errorf("reason = %q, want %q", decl.RotationReason, "Proactive rotation")
		}
		if len(decl.OldKeySignature) != ed25519.SignatureSize {
			t.Errorf("old signature size = %d, want %d", len(decl.OldKeySignature), ed25519.SignatureSize)
		}
		if len(decl.NewKeySignature) != ed25519.SignatureSize {
			t.Errorf("new signature size = %d, want %d", len(decl.NewKeySignature), ed25519.SignatureSize)
		}

		// Validate timestamp is recent
		now := time.Now().Unix()
		if decl.RotationTimestampUnix < now-5 || decl.RotationTimestampUnix > now+5 {
			t.Errorf("timestamp = %d, now = %d, delta = %d (should be <5s)",
				decl.RotationTimestampUnix, now, now-decl.RotationTimestampUnix)
		}
	})

	t.Run("success_custom_grace_period", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		opts := &RotateOptions{
			GracePeriodDays: 1, // Urgent rotation
			Reason:          "Security incident",
		}

		decl, err := CreateRotation(oldPriv, newPriv, opts)
		if err != nil {
			t.Fatalf("CreateRotation failed: %v", err)
		}

		if decl.GracePeriodDays != 1 {
			t.Errorf("grace period = %d, want 1", decl.GracePeriodDays)
		}
		if decl.RotationReason != "Security incident" {
			t.Errorf("reason = %q, want %q", decl.RotationReason, "Security incident")
		}
	})

	t.Run("truncate_long_reason", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		longReason := string(make([]byte, 500)) // 500 bytes
		opts := &RotateOptions{
			GracePeriodDays: 7,
			Reason:          longReason,
		}

		decl, err := CreateRotation(oldPriv, newPriv, opts)
		if err != nil {
			t.Fatalf("CreateRotation failed: %v", err)
		}

		if len(decl.RotationReason) != 256 {
			t.Errorf("reason length = %d, want 256 (truncated)", len(decl.RotationReason))
		}
	})

	t.Run("error_invalid_old_key", func(t *testing.T) {
		shortKey := make([]byte, 32) // Not a valid Ed25519 private key (should be 64 bytes)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		_, err := CreateRotation(shortKey, newPriv, nil)
		if err != ErrMissingOldPrivateKey {
			t.Errorf("error = %v, want ErrMissingOldPrivateKey", err)
		}
	})

	t.Run("error_invalid_new_key", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		shortKey := make([]byte, 32) // Not a valid Ed25519 private key

		_, err := CreateRotation(oldPriv, shortKey, nil)
		if err != ErrMissingNewPrivateKey {
			t.Errorf("error = %v, want ErrMissingNewPrivateKey", err)
		}
	})

	t.Run("error_grace_period_too_short", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		opts := &RotateOptions{
			GracePeriodDays: 0, // Below minimum
		}

		_, err := CreateRotation(oldPriv, newPriv, opts)
		if err != ErrGracePeriodInvalid {
			t.Errorf("error = %v, want ErrGracePeriodInvalid", err)
		}
	})

	t.Run("error_grace_period_too_long", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		opts := &RotateOptions{
			GracePeriodDays: 15, // Above maximum
		}

		_, err := CreateRotation(oldPriv, newPriv, opts)
		if err != ErrGracePeriodInvalid {
			t.Errorf("error = %v, want ErrGracePeriodInvalid", err)
		}
	})
}

func TestValidateDeclaration(t *testing.T) {
	t.Run("success_valid_declaration", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		decl, err := CreateRotation(oldPriv, newPriv, nil)
		if err != nil {
			t.Fatalf("CreateRotation failed: %v", err)
		}

		// Validate should succeed
		if err := ValidateDeclaration(decl); err != nil {
			t.Errorf("ValidateDeclaration failed: %v", err)
		}
	})

	t.Run("error_nil_declaration", func(t *testing.T) {
		err := ValidateDeclaration(nil)
		if err == nil {
			t.Error("ValidateDeclaration should fail for nil declaration")
		}
	})

	t.Run("error_invalid_old_key_size", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey: make([]byte, 16), // Wrong size
			NewPublicKey: make([]byte, 32),
		}

		err := ValidateDeclaration(decl)
		if err != ErrInvalidOldKey {
			t.Errorf("error = %v, want ErrInvalidOldKey", err)
		}
	})

	t.Run("error_invalid_new_key_size", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey: make([]byte, 32),
			NewPublicKey: make([]byte, 16), // Wrong size
		}

		err := ValidateDeclaration(decl)
		if err != ErrInvalidNewKey {
			t.Errorf("error = %v, want ErrInvalidNewKey", err)
		}
	})

	t.Run("error_bad_old_key_signature", func(t *testing.T) {
		oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
		newPub, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
			RotationReason:        "test",
		}

		sigData := buildSignatureData(decl)
		decl.OldKeySignature = make([]byte, ed25519.SignatureSize) // Invalid signature (all zeros)
		decl.NewKeySignature = ed25519.Sign(newPriv, sigData)

		err := ValidateDeclaration(decl)
		if err == nil || err.Error() != "rotation: old key signature verification failed" {
			t.Errorf("error = %v, want old key signature verification failed", err)
		}
	})

	t.Run("error_bad_new_key_signature", func(t *testing.T) {
		oldPub, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		newPub, _, _ := ed25519.GenerateKey(rand.Reader)

		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
			RotationReason:        "test",
		}

		sigData := buildSignatureData(decl)
		decl.OldKeySignature = ed25519.Sign(oldPriv, sigData)
		decl.NewKeySignature = make([]byte, ed25519.SignatureSize) // Invalid signature

		err := ValidateDeclaration(decl)
		if err == nil || err.Error() != "rotation: new key signature verification failed" {
			t.Errorf("error = %v, want new key signature verification failed", err)
		}
	})

	t.Run("error_timestamp_too_old", func(t *testing.T) {
		_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
		_, newPriv, _ := ed25519.GenerateKey(rand.Reader)

		opts := &RotateOptions{GracePeriodDays: 7, Reason: "test"}
		decl, _ := CreateRotation(oldPriv, newPriv, opts)

		// Set timestamp to 400 seconds ago (outside ±300s window)
		decl.RotationTimestampUnix = time.Now().Unix() - 400

		err := ValidateDeclaration(decl)
		if err == nil {
			t.Error("ValidateDeclaration should fail for old timestamp")
		}
	})
}

func TestIsKeyValidForTimestamp(t *testing.T) {
	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	now := time.Now().Unix()

	t.Run("no_chain_identity_root", func(t *testing.T) {
		chain := &pb.ContinuityChain{
			IdentityRootKey:  oldPub,
			CurrentActiveKey: oldPub,
			Declarations:     nil, // No rotations yet
		}

		// Original key should be valid
		if !IsKeyValidForTimestamp(oldPub, now, chain) {
			t.Error("identity root key should be valid when no rotations")
		}

		// Different key should be invalid
		if IsKeyValidForTimestamp(newPub, now, chain) {
			t.Error("unknown key should be invalid")
		}
	})

	t.Run("current_active_key_valid", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: now - 3600, // 1 hour ago
			GracePeriodDays:       7,
		}

		chain := &pb.ContinuityChain{
			IdentityRootKey:  oldPub,
			CurrentActiveKey: newPub,
			Declarations:     []*pb.ContinuityDeclaration{decl},
		}

		// New key should be valid (current active key)
		if !IsKeyValidForTimestamp(newPub, now, chain) {
			t.Error("current active key should be valid")
		}
	})

	t.Run("old_key_within_grace_period", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: now - 3600, // 1 hour ago
			GracePeriodDays:       7,
		}

		chain := &pb.ContinuityChain{
			IdentityRootKey:  oldPub,
			CurrentActiveKey: newPub,
			Declarations:     []*pb.ContinuityDeclaration{decl},
		}

		// Old key should still be valid (within 7-day grace period)
		if !IsKeyValidForTimestamp(oldPub, now, chain) {
			t.Error("old key should be valid within grace period")
		}
	})

	t.Run("old_key_expired", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: now - (8 * 86400), // 8 days ago
			GracePeriodDays:       7,
		}

		chain := &pb.ContinuityChain{
			IdentityRootKey:  oldPub,
			CurrentActiveKey: newPub,
			Declarations:     []*pb.ContinuityDeclaration{decl},
		}

		// Old key should be invalid (grace period expired)
		if IsKeyValidForTimestamp(oldPub, now, chain) {
			t.Error("old key should be invalid after grace period")
		}

		// New key should still be valid
		if !IsKeyValidForTimestamp(newPub, now, chain) {
			t.Error("new key should be valid")
		}
	})

	t.Run("multi_rotation_chain", func(t *testing.T) {
		key2, _, _ := ed25519.GenerateKey(rand.Reader)
		key3, _, _ := ed25519.GenerateKey(rand.Reader)

		decl1 := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: now - (10 * 86400), // 10 days ago
			GracePeriodDays:       7,
		}
		decl2 := &pb.ContinuityDeclaration{
			OldPublicKey:          newPub,
			NewPublicKey:          key2,
			RotationTimestampUnix: now - (5 * 86400), // 5 days ago
			GracePeriodDays:       7,
		}
		decl3 := &pb.ContinuityDeclaration{
			OldPublicKey:          key2,
			NewPublicKey:          key3,
			RotationTimestampUnix: now - (1 * 86400), // 1 day ago
			GracePeriodDays:       7,
		}

		chain := &pb.ContinuityChain{
			IdentityRootKey:  oldPub,
			CurrentActiveKey: key3,
			Declarations:     []*pb.ContinuityDeclaration{decl1, decl2, decl3},
		}

		// oldPub: expired (10 days ago, 7-day grace)
		if IsKeyValidForTimestamp(oldPub, now, chain) {
			t.Error("first key should be expired")
		}

		// newPub: within grace period (5 days ago, 7-day grace)
		if !IsKeyValidForTimestamp(newPub, now, chain) {
			t.Error("second key should be within grace period")
		}

		// key2: within grace period (1 day ago, 7-day grace)
		if !IsKeyValidForTimestamp(key2, now, chain) {
			t.Error("third key should be within grace period")
		}

		// key3: current active key
		if !IsKeyValidForTimestamp(key3, now, chain) {
			t.Error("fourth key should be current active key")
		}
	})
}

func TestBytesEqual(t *testing.T) {
	a := []byte{1, 2, 3, 4}
	b := []byte{1, 2, 3, 4}
	c := []byte{1, 2, 3, 5}
	d := []byte{1, 2, 3}

	if !bytesEqual(a, b) {
		t.Error("identical slices should be equal")
	}
	if bytesEqual(a, c) {
		t.Error("different content should not be equal")
	}
	if bytesEqual(a, d) {
		t.Error("different lengths should not be equal")
	}
}
