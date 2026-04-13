package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if db.Path() != path {
		t.Errorf("Path() = %q, want %q", db.Path(), path)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestPutGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Put a value.
	if err := db.Put(BucketConfig, key, value); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Get the value.
	got, err := db.Get(BucketConfig, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %q, want %q", got, value)
	}
}

func TestGetNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	got, err := db.Get(BucketConfig, []byte("nonexistent"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != nil {
		t.Errorf("Get() = %q, want nil for nonexistent key", got)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	key := []byte("delete-key")
	value := []byte("delete-value")

	// Put a value.
	if err := db.Put(BucketConfig, key, value); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Delete the value.
	if err := db.Delete(BucketConfig, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone.
	got, err := db.Get(BucketConfig, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != nil {
		t.Errorf("Get() = %q after Delete(), want nil", got)
	}
}

func TestAllBucketsCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	buckets := [][]byte{
		BucketIdentity,
		BucketPeers,
		BucketWaves,
		BucketThreads,
		BucketShroud,
		BucketResonance,
		BucketConfig,
	}

	// Verify each bucket exists by putting a test value.
	for _, bucket := range buckets {
		err := db.Put(bucket, []byte("test"), []byte("value"))
		if err != nil {
			t.Errorf("Put to bucket %s failed: %v", bucket, err)
		}
	}
}
