//go:build integration
// +build integration

// Package integration provides end-to-end integration tests for MURMUR core workflows.
// These tests verify subsystem interactions using real libp2p hosts and stores.
package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/stretchr/testify/require"
)

// TestIdentityPersistence verifies that a keypair can be generated, persisted to Bbolt,
// and successfully restored after the database is closed and reopened.
// Per PLAN.md Step 7: "Identity test confirms keypair survives Bbolt close/reopen cycle"
func TestIdentityPersistence(t *testing.T) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "identity-test.db")

	// Phase 1: Generate and persist keypair
	var originalPubKey []byte
	{
		db, err := store.Open(dbPath)
		require.NoError(t, err, "opening database for first time")

		// Generate new keypair
		kp, err := keys.GenerateKeyPair()
		require.NoError(t, err, "generating keypair")
		require.NotNil(t, kp)
		require.Len(t, kp.PrivateKey, 64, "Ed25519 private key should be 64 bytes")
		require.Len(t, kp.PublicKey, 32, "Ed25519 public key should be 32 bytes")

		// Store keypair in identity bucket
		err = db.Put(store.BucketIdentity, []byte("keypair"), kp.PrivateKey)
		require.NoError(t, err, "storing keypair")

		// Remember public key for comparison
		originalPubKey = make([]byte, len(kp.PublicKey))
		copy(originalPubKey, kp.PublicKey)

		// Close database
		err = db.Close()
		require.NoError(t, err, "closing database after write")
	}

	// Verify database file was created
	info, err := os.Stat(dbPath)
	require.NoError(t, err, "database file should exist")
	require.Greater(t, info.Size(), int64(0), "database file should not be empty")

	// Phase 2: Reopen database and restore keypair
	{
		db, err := store.Open(dbPath)
		require.NoError(t, err, "reopening database")
		defer db.Close()

		// Retrieve keypair from identity bucket
		privKeyBytes, err := db.Get(store.BucketIdentity, []byte("keypair"))
		require.NoError(t, err, "retrieving keypair")
		require.NotNil(t, privKeyBytes, "keypair should exist in database")
		require.Len(t, privKeyBytes, 64, "stored private key should be 64 bytes")

		// Reconstruct keypair
		restoredKp := &keys.KeyPair{
			PrivateKey: privKeyBytes,
			PublicKey:  privKeyBytes[32:],
		}

		// Verify public key matches original
		require.Equal(t, originalPubKey, []byte(restoredKp.PublicKey), "restored public key should match original")

		// Verify keypair can sign and verify a message
		message := []byte("test message for signature verification")
		signature := restoredKp.Sign(message)
		require.Len(t, signature, 64, "Ed25519 signature should be 64 bytes")

		valid := keys.Verify(restoredKp.PublicKey, message, signature)
		require.True(t, valid, "signature should verify with restored keypair")

		// Verify signature fails with wrong message
		wrongMessage := []byte("wrong message")
		invalidSig := keys.Verify(restoredKp.PublicKey, wrongMessage, signature)
		require.False(t, invalidSig, "signature should not verify with wrong message")
	}
}

// TestIdentityMultipleKeys verifies that multiple identity keys can be stored and retrieved.
func TestIdentityMultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multi-identity-test.db")

	db, err := store.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Generate and store Surface identity
	surfaceKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)
	err = db.Put(store.BucketIdentity, []byte("surface-keypair"), surfaceKp.PrivateKey)
	require.NoError(t, err)

	// Generate and store Specter identity (Anonymous Layer)
	specterKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)
	err = db.Put(store.BucketIdentity, []byte("specter-keypair"), specterKp.PrivateKey)
	require.NoError(t, err)

	// Retrieve both keypairs
	surfaceBytes, err := db.Get(store.BucketIdentity, []byte("surface-keypair"))
	require.NoError(t, err)
	require.Equal(t, []byte(surfaceKp.PrivateKey), surfaceBytes)

	specterBytes, err := db.Get(store.BucketIdentity, []byte("specter-keypair"))
	require.NoError(t, err)
	require.Equal(t, []byte(specterKp.PrivateKey), specterBytes)

	// Verify keypairs are distinct
	require.NotEqual(t, surfaceKp.PublicKey, specterKp.PublicKey, "Surface and Specter identities should be different")
}

// TestIdentityFirstRunDetection verifies first-run detection logic.
func TestIdentityFirstRunDetection(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "first-run-test.db")

	// First run: no keypair exists
	{
		db, err := store.Open(dbPath)
		require.NoError(t, err)

		keypairBytes, err := db.Get(store.BucketIdentity, []byte("keypair"))
		require.NoError(t, err)
		require.Nil(t, keypairBytes, "keypair should not exist on first run")

		// Simulate first-run initialization
		kp, err := keys.GenerateKeyPair()
		require.NoError(t, err)
		err = db.Put(store.BucketIdentity, []byte("keypair"), kp.PrivateKey)
		require.NoError(t, err)

		db.Close()
	}

	// Second run: keypair exists
	{
		db, err := store.Open(dbPath)
		require.NoError(t, err)
		defer db.Close()

		keypairBytes, err := db.Get(store.BucketIdentity, []byte("keypair"))
		require.NoError(t, err)
		require.NotNil(t, keypairBytes, "keypair should exist on subsequent runs")
		require.Len(t, keypairBytes, 64)
	}
}
