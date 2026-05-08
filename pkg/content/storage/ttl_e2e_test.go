// Package storage provides end-to-end TTL expiration correctness tests.
// Per PLAN.md: "Wave TTL expiration correctness (end-to-end validation)".
package storage

import (
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

// createExpiredWave builds a Wave whose CreatedAt is set to make it
// expire immediately with the given TTL.  The signed/PoW wave is created
// normally (so all other validations pass) then backdated.
func createExpiredWave(t *testing.T, ttl time.Duration) *pb.Wave {
	t.Helper()

	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	opts := waves.DefaultCreateOptions()
	opts.TTL = ttl
	opts.Difficulty = 8

	wave, err := waves.Create(waves.TypeSurface, []byte("expiry e2e test"), kp, opts)
	if err != nil {
		t.Fatalf("waves.Create: %v", err)
	}

	// Backdate so that created_at + ttl is in the past.
	wave.CreatedAt = time.Now().Add(-(ttl + time.Second)).Unix()
	return wave
}

// createWaveWithTTL creates a fresh (non-expired) wave with the given TTL.
func createWaveWithTTL(t *testing.T, ttl time.Duration) *pb.Wave {
	t.Helper()

	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	opts := waves.DefaultCreateOptions()
	opts.TTL = ttl
	opts.Difficulty = 8

	wave, err := waves.Create(waves.TypeSurface, []byte("ttl e2e test"), kp, opts)
	if err != nil {
		t.Fatalf("waves.Create: %v", err)
	}

	return wave
}

// TestTTLLifecycleFreshWaveRetrievable verifies a wave with a future expiration
// is stored and retrieved successfully.
func TestTTLLifecycleFreshWaveRetrievable(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, err := NewCache(db)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	wave := createWaveWithTTL(t, waves.DefaultTTL)

	if err := cache.Put(wave); err != nil {
		t.Fatalf("Put: %v", err)
	}

	retrieved, err := cache.Get(wave.WaveId)
	if err != nil {
		t.Fatalf("Get for fresh wave: %v", err)
	}

	if string(retrieved.WaveId) != string(wave.WaveId) {
		t.Error("retrieved wave ID does not match original")
	}

	if waves.IsExpired(retrieved) {
		t.Error("retrieved fresh wave reports as expired")
	}
}

// TestTTLLifecycleExpiredWaveGCed verifies that an expired wave is removed
// from both the memory cache and the database after GC runs.
func TestTTLLifecycleExpiredWaveGCed(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, err := NewCache(db)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	ttl := 7 * 24 * time.Hour
	wave := createExpiredWave(t, ttl)

	// Store the already-expired wave.
	if err := cache.Put(wave); err != nil {
		t.Fatalf("Put expired wave: %v", err)
	}

	// Confirm it is in memory before GC.
	if cache.Size() == 0 {
		t.Fatal("wave should be in memory before GC")
	}

	count, err := cache.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect: %v", err)
	}

	if count == 0 {
		t.Error("GC should have collected at least one expired wave, got 0")
	}

	if cache.Size() != 0 {
		t.Errorf("memory cache should be empty after GC, got %d wave(s)", cache.Size())
	}

	_, err = cache.Get(wave.WaveId)
	if err != ErrNotFound {
		t.Errorf("Get after GC: expected ErrNotFound, got %v", err)
	}
}

// TestTTLLifecycleDatabaseOnlyExpiredWaveGCed verifies that an expired wave
// that was evicted from the memory cache is still removed by GC via the
// database scan path.
func TestTTLLifecycleDatabaseOnlyExpiredWaveGCed(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Use a tiny cache so the wave gets evicted from memory immediately.
	cache, err := NewCacheWithConfig(db, CacheConfig{MaxSize: 1})
	if err != nil {
		t.Fatalf("NewCacheWithConfig: %v", err)
	}
	defer cache.Close()

	expiredWave := createExpiredWave(t, 7*24*time.Hour)
	freshWave := createWaveWithTTL(t, waves.DefaultTTL)

	if err := cache.Put(expiredWave); err != nil {
		t.Fatalf("Put expired wave: %v", err)
	}

	// Putting the fresh wave evicts the expired one from memory (max size 1).
	if err := cache.Put(freshWave); err != nil {
		t.Fatalf("Put fresh wave: %v", err)
	}

	// The expired wave should be database-only at this point.
	count, err := cache.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect: %v", err)
	}

	if count == 0 {
		t.Error("GC should have collected the database-only expired wave")
	}

	_, err = cache.Get(expiredWave.WaveId)
	if err != ErrNotFound {
		t.Errorf("Get expired DB-only wave after GC: expected ErrNotFound, got %v", err)
	}

	// Fresh wave must still be retrievable.
	if _, err := cache.Get(freshWave.WaveId); err != nil {
		t.Errorf("fresh wave should still be retrievable after GC, got %v", err)
	}
}

// TestTTLLifecycleFreshWaveNotEvictedByGC verifies that GC does not remove
// a wave that has not yet reached its TTL.
func TestTTLLifecycleFreshWaveNotEvictedByGC(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, err := NewCache(db)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	wave := createWaveWithTTL(t, waves.DefaultTTL)

	if err := cache.Put(wave); err != nil {
		t.Fatalf("Put: %v", err)
	}

	count, err := cache.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect: %v", err)
	}

	if count != 0 {
		t.Errorf("GC removed %d wave(s) that should not be expired yet", count)
	}

	if _, err := cache.Get(wave.WaveId); err != nil {
		t.Errorf("fresh wave should be retrievable after GC, got %v", err)
	}
}

// TestTTLListFiltersExpired verifies that List never returns expired waves.
func TestTTLListFiltersExpired(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	cache, err := NewCache(db)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer cache.Close()

	freshWave := createWaveWithTTL(t, waves.DefaultTTL)
	expiredWave := createExpiredWave(t, 7*24*time.Hour)

	// Both waves have different IDs; ensure uniqueness.
	expiredWave.WaveId = append(expiredWave.WaveId, byte(0xFF))

	if err := cache.Put(freshWave); err != nil {
		t.Fatalf("Put fresh wave: %v", err)
	}
	if err := cache.Put(expiredWave); err != nil {
		t.Fatalf("Put expired wave: %v", err)
	}

	listed, err := cache.List(100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, w := range listed {
		if waves.IsExpired(w) {
			t.Errorf("List returned an expired wave (ID %x)", w.WaveId)
		}
	}
}

// TestTTLMaxContentWindowEnforced verifies that a wave with the maximum allowed
// TTL (30 days) is still correctly identified as expired after that period.
func TestTTLMaxContentWindowEnforced(t *testing.T) {
	wave := createExpiredWave(t, waves.MaxTTL)

	if !waves.IsExpired(wave) {
		t.Error("wave backdated beyond MaxTTL should be reported as expired")
	}
}
