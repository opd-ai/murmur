package store

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestStoreContinuityDeclaration(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	t.Run("first_rotation", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
			RotationReason:        "test",
		}

		err := db.StoreContinuityDeclaration(oldPub, decl)
		if err != nil {
			t.Fatalf("StoreContinuityDeclaration failed: %v", err)
		}

		// Verify chain created
		chain, err := db.GetContinuityChain(oldPub)
		if err != nil {
			t.Fatalf("GetContinuityChain failed: %v", err)
		}

		if len(chain.Declarations) != 1 {
			t.Errorf("declarations length = %d, want 1", len(chain.Declarations))
		}
		if string(chain.CurrentActiveKey) != string(newPub) {
			t.Errorf("current active key mismatch")
		}
		if string(chain.IdentityRootKey) != string(oldPub) {
			t.Errorf("identity root key mismatch")
		}
	})

	t.Run("duplicate_declaration_idempotent", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
			RotationReason:        "test",
		}

		// Store twice
		_ = db.StoreContinuityDeclaration(oldPub, decl)
		err := db.StoreContinuityDeclaration(oldPub, decl)
		if err != nil {
			t.Fatalf("duplicate store failed: %v", err)
		}

		// Verify only one declaration exists
		chain, _ := db.GetContinuityChain(oldPub)
		if len(chain.Declarations) != 1 {
			t.Errorf("duplicate not deduplicated: got %d declarations", len(chain.Declarations))
		}
	})

	t.Run("second_rotation", func(t *testing.T) {
		key3, _, _ := ed25519.GenerateKey(rand.Reader)

		decl2 := &pb.ContinuityDeclaration{
			OldPublicKey:          newPub,
			NewPublicKey:          key3,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
			RotationReason:        "test2",
		}

		err := db.StoreContinuityDeclaration(oldPub, decl2)
		if err != nil {
			t.Fatalf("second rotation failed: %v", err)
		}

		// Verify chain has 2 declarations
		chain, _ := db.GetContinuityChain(oldPub)
		if len(chain.Declarations) != 2 {
			t.Errorf("declarations length = %d, want 2", len(chain.Declarations))
		}
		if string(chain.CurrentActiveKey) != string(key3) {
			t.Errorf("current active key should be key3")
		}
	})

	t.Run("error_invalid_identity_root", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey: oldPub,
			NewPublicKey: newPub,
		}

		shortKey := make([]byte, 16)
		err := db.StoreContinuityDeclaration(shortKey, decl)
		if err == nil {
			t.Error("should fail with invalid identity root size")
		}
	})

	t.Run("error_nil_declaration", func(t *testing.T) {
		err := db.StoreContinuityDeclaration(oldPub, nil)
		if err == nil {
			t.Error("should fail with nil declaration")
		}
	})
}

func TestGetContinuityChain(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	t.Run("chain_not_found", func(t *testing.T) {
		unknownKey := make([]byte, 32)
		_, err := db.GetContinuityChain(unknownKey)
		if err != ErrChainNotFound {
			t.Errorf("error = %v, want ErrChainNotFound", err)
		}
	})

	t.Run("chain_found", func(t *testing.T) {
		// Store a declaration first
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
		}
		_ = db.StoreContinuityDeclaration(oldPub, decl)

		// Retrieve
		chain, err := db.GetContinuityChain(oldPub)
		if err != nil {
			t.Fatalf("GetContinuityChain failed: %v", err)
		}

		if chain == nil {
			t.Fatal("chain is nil")
		}
		if len(chain.Declarations) != 1 {
			t.Errorf("declarations length = %d, want 1", len(chain.Declarations))
		}
	})
}

func TestResolveCurrentKey(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	t.Run("no_rotation", func(t *testing.T) {
		_, err := db.ResolveCurrentKey(oldPub)
		if err != ErrChainNotFound {
			t.Errorf("error = %v, want ErrChainNotFound", err)
		}
	})

	t.Run("after_rotation", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
		}
		_ = db.StoreContinuityDeclaration(oldPub, decl)

		currentKey, err := db.ResolveCurrentKey(oldPub)
		if err != nil {
			t.Fatalf("ResolveCurrentKey failed: %v", err)
		}

		if string(currentKey) != string(newPub) {
			t.Errorf("current key mismatch")
		}
	})
}

func TestIsKeyValid(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	now := time.Now().Unix()

	t.Run("no_chain_identity_root", func(t *testing.T) {
		// No rotation, key must equal identity root
		valid, err := db.IsKeyValid(oldPub, oldPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if !valid {
			t.Error("identity root should be valid when no rotation")
		}

		// Different key should be invalid
		valid, err = db.IsKeyValid(oldPub, newPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if valid {
			t.Error("unknown key should be invalid")
		}
	})

	t.Run("current_active_key", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: now - 3600, // 1 hour ago
			GracePeriodDays:       7,
		}
		_ = db.StoreContinuityDeclaration(oldPub, decl)

		// New key should be valid
		valid, err := db.IsKeyValid(oldPub, newPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if !valid {
			t.Error("current active key should be valid")
		}
	})

	t.Run("old_key_within_grace", func(t *testing.T) {
		// Old key should still be valid (within 7-day grace period)
		valid, err := db.IsKeyValid(oldPub, oldPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if !valid {
			t.Error("old key should be valid within grace period")
		}
	})

	t.Run("old_key_expired", func(t *testing.T) {
		// Use fresh keys to avoid conflicts with previous tests
		expiredOldPub, _, _ := ed25519.GenerateKey(rand.Reader)
		expiredNewPub, _, _ := ed25519.GenerateKey(rand.Reader)

		// Simulate expired grace period
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          expiredOldPub,
			NewPublicKey:          expiredNewPub,
			RotationTimestampUnix: now - (8 * 86400), // 8 days ago
			GracePeriodDays:       7,
		}
		_ = db.StoreContinuityDeclaration(expiredOldPub, decl)

		// Old key should be invalid
		valid, err := db.IsKeyValid(expiredOldPub, expiredOldPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if valid {
			t.Error("old key should be invalid after grace period")
		}

		// New key should still be valid
		valid, err = db.IsKeyValid(expiredOldPub, expiredNewPub, now)
		if err != nil {
			t.Fatalf("IsKeyValid failed: %v", err)
		}
		if !valid {
			t.Error("new key should be valid")
		}
	})
}

func TestResolveIdentityRoot(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	oldPub, _, _ := ed25519.GenerateKey(rand.Reader)
	newPub, _, _ := ed25519.GenerateKey(rand.Reader)

	t.Run("key_not_found", func(t *testing.T) {
		unknownKey := make([]byte, 32)
		root, err := db.ResolveIdentityRoot(unknownKey)
		if err != nil {
			t.Fatalf("ResolveIdentityRoot failed: %v", err)
		}
		if root != nil {
			t.Error("unknown key should return nil")
		}
	})

	t.Run("find_by_old_key", func(t *testing.T) {
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          oldPub,
			NewPublicKey:          newPub,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
		}
		_ = db.StoreContinuityDeclaration(oldPub, decl)

		root, err := db.ResolveIdentityRoot(oldPub)
		if err != nil {
			t.Fatalf("ResolveIdentityRoot failed: %v", err)
		}
		if string(root) != string(oldPub) {
			t.Error("should find identity root by old key")
		}
	})

	t.Run("find_by_new_key", func(t *testing.T) {
		root, err := db.ResolveIdentityRoot(newPub)
		if err != nil {
			t.Fatalf("ResolveIdentityRoot failed: %v", err)
		}
		if string(root) != string(oldPub) {
			t.Error("should find identity root by new key")
		}
	})
}

func TestChainLengthLimit(t *testing.T) {
	db := openTestDB(t)
	defer closeTestDB(t, db)

	rootKey, _, _ := ed25519.GenerateKey(rand.Reader)
	currentKey := rootKey

	// Add exactly MaxChainLength declarations
	for i := 0; i < MaxChainLength; i++ {
		nextKey, _, _ := ed25519.GenerateKey(rand.Reader)
		decl := &pb.ContinuityDeclaration{
			OldPublicKey:          currentKey,
			NewPublicKey:          nextKey,
			RotationTimestampUnix: time.Now().Unix(),
			GracePeriodDays:       7,
		}
		err := db.StoreContinuityDeclaration(rootKey, decl)
		if err != nil {
			t.Fatalf("rotation %d failed: %v", i, err)
		}
		currentKey = nextKey
	}

	// 101st rotation should fail
	nextKey, _, _ := ed25519.GenerateKey(rand.Reader)
	decl := &pb.ContinuityDeclaration{
		OldPublicKey:          currentKey,
		NewPublicKey:          nextKey,
		RotationTimestampUnix: time.Now().Unix(),
		GracePeriodDays:       7,
	}
	err := db.StoreContinuityDeclaration(rootKey, decl)
	if err != ErrChainLengthExceeded {
		t.Errorf("error = %v, want ErrChainLengthExceeded", err)
	}
}

// openTestDB creates an in-memory database for testing.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return db
}

// closeTestDB closes and removes the test database.
func closeTestDB(t *testing.T, db *DB) {
	t.Helper()
	path := db.Path()
	if err := db.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Errorf("Remove failed: %v", err)
	}
}
