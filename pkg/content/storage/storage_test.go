package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
)

func createTestDB(t *testing.T) (*store.DB, func()) {
	t.Helper()

	f, err := os.CreateTemp("", "murmur-storage-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()

	db, err := store.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(f.Name())
	}

	return db, cleanup
}

func createTestWave(t *testing.T) *pb.Wave {
	t.Helper()

	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	opts := waves.DefaultCreateOptions()
	opts.Difficulty = 8

	wave, err := waves.Create(waves.TypeSurface, []byte("test content"), kp, opts)
	if err != nil {
		t.Fatalf("waves.Create failed: %v", err)
	}

	return wave
}

func TestNewCache(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, err := NewCache(db)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	if cache == nil {
		t.Fatal("cache is nil")
	}

	if cache.maxSize != DefaultCacheSize {
		t.Errorf("maxSize = %d, want %d", cache.maxSize, DefaultCacheSize)
	}
}

func TestNewCacheNilDB(t *testing.T) {
	_, err := NewCache(nil)
	if err != ErrNilStore {
		t.Errorf("expected ErrNilStore, got %v", err)
	}
}

func TestCachePutGet(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)
	wave := createTestWave(t)

	// Put wave.
	if err := cache.Put(wave); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get wave.
	retrieved, err := cache.Get(wave.WaveId)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved.Content) != string(wave.Content) {
		t.Errorf("content mismatch")
	}
}

func TestCacheGetNotFound(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)

	_, err := cache.Get([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCacheDelete(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)
	wave := createTestWave(t)

	cache.Put(wave)

	if err := cache.Delete(wave.WaveId); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := cache.Get(wave.WaveId)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCacheHas(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)
	wave := createTestWave(t)

	if cache.Has(wave.WaveId) {
		t.Error("Has returned true for nonexistent wave")
	}

	cache.Put(wave)

	if !cache.Has(wave.WaveId) {
		t.Error("Has returned false for existing wave")
	}
}

func TestCacheSize(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)

	if cache.Size() != 0 {
		t.Errorf("initial size = %d, want 0", cache.Size())
	}

	for i := 0; i < 5; i++ {
		cache.Put(createTestWave(t))
	}

	if cache.Size() != 5 {
		t.Errorf("size = %d, want 5", cache.Size())
	}
}

func TestCachePutInvalidWave(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)

	if err := cache.Put(nil); err != ErrInvalidWave {
		t.Errorf("expected ErrInvalidWave for nil, got %v", err)
	}

	if err := cache.Put(&pb.Wave{}); err != ErrInvalidWave {
		t.Errorf("expected ErrInvalidWave for empty wave, got %v", err)
	}
}

func TestCacheGarbageCollect(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)

	// Create and store a wave.
	wave := createTestWave(t)
	cache.Put(wave)

	// Make it expired by modifying created_at.
	wave.CreatedAt = time.Now().Add(-8 * 24 * time.Hour).Unix()
	cache.Put(wave)

	// GC should remove it.
	count, err := cache.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect failed: %v", err)
	}

	if count != 1 {
		t.Errorf("GC count = %d, want 1", count)
	}
}

func TestCacheStartGC(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)

	ctx := context.Background()
	cancel := cache.StartGC(ctx, 50*time.Millisecond)
	defer cancel()

	// Add expired wave.
	wave := createTestWave(t)
	wave.CreatedAt = time.Now().Add(-8 * 24 * time.Hour).Unix()
	cache.Put(wave)

	// Wait for GC to run.
	time.Sleep(100 * time.Millisecond)

	// Wave should be gone from memory.
	if cache.Size() != 0 {
		t.Errorf("cache size after GC = %d, want 0", cache.Size())
	}
}

func TestCacheClose(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCache(db)
	cache.Put(createTestWave(t))

	if err := cache.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations should fail after close.
	if err := cache.Put(createTestWave(t)); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed after close, got %v", err)
	}

	if _, err := cache.Get([]byte("any")); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed for Get after close, got %v", err)
	}
}

func TestCacheMaxSize(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, _ := NewCacheWithConfig(db, CacheConfig{MaxSize: 3})

	// Fill cache.
	for i := 0; i < 3; i++ {
		if err := cache.Put(createTestWave(t)); err != nil {
			t.Fatalf("Put %d failed: %v", i, err)
		}
	}

	// Fourth should fail.
	if err := cache.Put(createTestWave(t)); err != ErrCacheFull {
		t.Errorf("expected ErrCacheFull, got %v", err)
	}
}

func TestCachePersistence(t *testing.T) {
	f, err := os.CreateTemp("", "murmur-storage-persist-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	wave := createTestWave(t)

	// Store in first cache instance.
	{
		db, err := store.Open(f.Name())
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		cache, _ := NewCache(db)
		cache.Put(wave)
		db.Close()
	}

	// Retrieve from new cache instance.
	{
		db, err := store.Open(f.Name())
		if err != nil {
			t.Fatalf("failed to reopen database: %v", err)
		}
		defer db.Close()

		cache, _ := NewCache(db)

		retrieved, err := cache.Get(wave.WaveId)
		if err != nil {
			t.Fatalf("Get from new instance failed: %v", err)
		}

		if string(retrieved.Content) != string(wave.Content) {
			t.Error("persisted content mismatch")
		}
	}
}
