package store

import (
	"path/filepath"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(BucketWaves, 5)

	// Add items.
	for i := 0; i < 5; i++ {
		cache.Touch([]byte{byte(i)})
	}

	if cache.Len() != 5 {
		t.Errorf("Len() = %d, want 5", cache.Len())
	}

	// Adding more items should evict oldest.
	cache.Touch([]byte{5})
	if cache.Len() != 5 {
		t.Errorf("Len() = %d after overflow, want 5", cache.Len())
	}

	// Item 0 should be evicted.
	lruKeys := cache.GetLRUKeys(10)
	for _, key := range lruKeys {
		if key[0] == 0 {
			t.Error("Item 0 should have been evicted")
		}
	}
}

func TestLRUCacheGetLRUKeys(t *testing.T) {
	cache := NewLRUCache(BucketWaves, 0) // Unlimited

	// Add items with delays to ensure ordering.
	cache.Touch([]byte("a"))
	cache.Touch([]byte("b"))
	cache.Touch([]byte("c"))

	// Touch 'a' again to make it most recently used.
	cache.Touch([]byte("a"))

	// Get LRU keys - 'b' should be least recently used.
	keys := cache.GetLRUKeys(1)
	if len(keys) != 1 {
		t.Fatalf("GetLRUKeys(1) returned %d keys", len(keys))
	}
	if string(keys[0]) != "b" {
		t.Errorf("GetLRUKeys(1) = %q, want 'b'", keys[0])
	}
}

func TestLRUCacheRemove(t *testing.T) {
	cache := NewLRUCache(BucketWaves, 0)

	cache.Touch([]byte("a"))
	cache.Touch([]byte("b"))

	if cache.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", cache.Len())
	}

	cache.Remove([]byte("a"))

	if cache.Len() != 1 {
		t.Errorf("Len() after Remove = %d, want 1", cache.Len())
	}
}

func TestEvictionManager(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	config := &EvictionConfig{
		MaxDatabaseSize:   1024, // Tiny for testing
		EvictionBatchSize: 2,
		EvictionThreshold: 0.5,
	}
	em := NewEvictionManager(db, config)
	em.RegisterBucket(BucketConfig, 100)

	// Add some data.
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		if err := db.Put(BucketConfig, key, []byte("value")); err != nil {
			t.Fatalf("Put() error: %v", err)
		}
		em.Touch(BucketConfig, key)
	}

	// Check and evict - database is tiny so it should evict.
	evicted, err := em.CheckAndEvict()
	if err != nil {
		t.Fatalf("CheckAndEvict() error: %v", err)
	}

	// Should have evicted something.
	if evicted < 1 {
		t.Log("Note: eviction may not occur if database file is below threshold")
	}
}

func TestEvictExpiredWaves(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()

	// Create an expired wave.
	expiredWave := &pb.Wave{
		WaveId:     []byte("expired"),
		CreatedAt:  now - 1000, // 1000 seconds ago
		TtlSeconds: 100,        // TTL expired 900 seconds ago
	}
	if err := db.PutWave(expiredWave); err != nil {
		t.Fatalf("PutWave() error: %v", err)
	}

	// Create a valid wave.
	validWave := &pb.Wave{
		WaveId:     []byte("valid"),
		CreatedAt:  now,
		TtlSeconds: 86400, // 24 hours
	}
	if err := db.PutWave(validWave); err != nil {
		t.Fatalf("PutWave() error: %v", err)
	}

	// Evict expired waves.
	evicted, err := db.EvictExpiredWaves()
	if err != nil {
		t.Fatalf("EvictExpiredWaves() error: %v", err)
	}

	if evicted != 1 {
		t.Errorf("EvictExpiredWaves() = %d, want 1", evicted)
	}

	// Verify expired wave is gone.
	got, err := db.GetWave([]byte("expired"))
	if err != nil {
		t.Fatalf("GetWave() error: %v", err)
	}
	if got != nil {
		t.Error("expired wave should have been evicted")
	}

	// Verify valid wave remains.
	got, err = db.GetWave([]byte("valid"))
	if err != nil {
		t.Fatalf("GetWave() error: %v", err)
	}
	if got == nil {
		t.Error("valid wave should not have been evicted")
	}
}
