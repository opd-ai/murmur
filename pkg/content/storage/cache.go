// Package storage provides local Wave caching and garbage collection.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, Waves are stored in Bbolt
// with TTL metadata for expiration.
package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// GCInterval is the interval between garbage collection runs.
// Per TECHNICAL_IMPLEMENTATION.md, GC runs hourly.
const GCInterval = time.Hour

// GCTargetTime is the target maximum duration for garbage collection.
// Per TECHNICAL_IMPLEMENTATION.md, GC should complete in <100ms.
const GCTargetTime = 100 * time.Millisecond

// DefaultCacheSize is the default maximum number of Waves to cache.
const DefaultCacheSize = 10000

// MaxContentWindow is the maximum age of content (30 days).
// Per WAVES.md, Waves older than 30 days are garbage collected.
const MaxContentWindow = 30 * 24 * time.Hour

// Errors for storage operations.
var (
	ErrNotFound    = errors.New("wave not found")
	ErrCacheFull   = errors.New("cache is full")
	ErrStoreClosed = errors.New("store is closed")
	ErrInvalidWave = errors.New("invalid wave data")
	ErrNilStore    = errors.New("store is nil")
)

// Cache provides in-memory and persistent Wave storage with TTL enforcement.
type Cache struct {
	mu      sync.RWMutex
	db      *store.DB
	memory  map[string]*pb.Wave // wave ID -> Wave
	maxSize int
	closed  bool
}

// CacheConfig configures the Wave cache.
type CacheConfig struct {
	MaxSize int
}

// NewCache creates a new Wave cache with the given database.
func NewCache(db *store.DB) (*Cache, error) {
	if db == nil {
		return nil, ErrNilStore
	}

	return &Cache{
		db:      db,
		memory:  make(map[string]*pb.Wave),
		maxSize: DefaultCacheSize,
	}, nil
}

// NewCacheWithConfig creates a cache with custom configuration.
func NewCacheWithConfig(db *store.DB, cfg CacheConfig) (*Cache, error) {
	if db == nil {
		return nil, ErrNilStore
	}

	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}

	return &Cache{
		db:      db,
		memory:  make(map[string]*pb.Wave),
		maxSize: maxSize,
	}, nil
}

// Put stores a Wave in the cache and database.
func (c *Cache) Put(wave *pb.Wave) error {
	if err := validateWave(wave); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrStoreClosed
	}

	if err := c.ensureCapacityLocked(); err != nil {
		return err
	}

	c.memory[string(wave.WaveId)] = wave
	return c.persistWave(wave)
}

// validateWave checks if the wave is valid for storage.
func validateWave(wave *pb.Wave) error {
	if wave == nil || len(wave.WaveId) == 0 {
		return ErrInvalidWave
	}
	return nil
}

// ensureCapacityLocked ensures cache has room for a new wave. Must hold c.mu.
func (c *Cache) ensureCapacityLocked() error {
	if len(c.memory) < c.maxSize {
		return nil
	}

	c.evictExpiredLocked()
	if len(c.memory) >= c.maxSize {
		return ErrCacheFull
	}
	return nil
}

// persistWave serializes and stores the wave in the database.
func (c *Cache) persistWave(wave *pb.Wave) error {
	data, err := proto.Marshal(wave)
	if err != nil {
		return err
	}
	return c.db.Put(store.BucketWaves, wave.WaveId, data)
}

// Get retrieves a Wave by ID from cache or database.
func (c *Cache) Get(waveID []byte) (*pb.Wave, error) {
	wave, found, err := c.getFromMemory(waveID)
	if err != nil {
		return nil, err
	}
	if found {
		return wave, nil
	}
	return c.getFromDatabase(waveID)
}

// getFromMemory checks the memory cache for a Wave.
func (c *Cache) getFromMemory(waveID []byte) (*pb.Wave, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, false, ErrStoreClosed
	}

	wave, ok := c.memory[string(waveID)]
	return wave, ok, nil
}

// getFromDatabase retrieves a Wave from the database and caches it.
func (c *Cache) getFromDatabase(waveID []byte) (*pb.Wave, error) {
	data, err := c.db.Get(store.BucketWaves, waveID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, ErrNotFound
	}

	wave := &pb.Wave{}
	if err := proto.Unmarshal(data, wave); err != nil {
		return nil, err
	}

	c.cacheWave(waveID, wave)
	return wave, nil
}

// cacheWave adds a Wave to the memory cache if space is available.
func (c *Cache) cacheWave(waveID []byte, wave *pb.Wave) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed && len(c.memory) < c.maxSize {
		c.memory[string(waveID)] = wave
	}
}

// Delete removes a Wave from cache and database.
func (c *Cache) Delete(waveID []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrStoreClosed
	}

	delete(c.memory, string(waveID))
	return c.db.Delete(store.BucketWaves, waveID)
}

// Has checks if a Wave exists in cache or database.
func (c *Cache) Has(waveID []byte) bool {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false
	}

	if _, ok := c.memory[string(waveID)]; ok {
		c.mu.RUnlock()
		return true
	}
	c.mu.RUnlock()

	data, err := c.db.Get(store.BucketWaves, waveID)
	return err == nil && len(data) > 0
}

// Size returns the number of Waves in the memory cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.memory)
}

// evictExpiredLocked removes expired waves from memory cache.
// Must be called with c.mu held.
func (c *Cache) evictExpiredLocked() int {
	count := 0
	for id, wave := range c.memory {
		if waves.IsExpired(wave) {
			delete(c.memory, id)
			count++
		}
	}
	return count
}

// GarbageCollect removes expired Waves from cache and database.
// Per TECHNICAL_IMPLEMENTATION.md, this should complete in <100ms.
func (c *Cache) GarbageCollect() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, ErrStoreClosed
	}

	// Phase 1: Collect expired IDs from memory cache.
	memoryExpired := c.collectExpiredIDs()
	memoryExpiredSet := make(map[string]struct{}, len(memoryExpired))
	for _, id := range memoryExpired {
		memoryExpiredSet[string(id)] = struct{}{}
	}

	// Phase 2: Scan database for expired waves not already found in memory.
	dbExpired, err := c.collectExpiredFromDatabase(memoryExpiredSet)
	if err != nil {
		// Still clean up memory even if database scan fails.
		c.removeFromMemory(memoryExpired)
		c.removeFromDatabase(memoryExpired)
		return len(memoryExpired), err
	}

	// Phase 3: Remove all expired waves from memory and database.
	c.removeFromMemory(memoryExpired)
	allExpired := append(memoryExpired, dbExpired...)
	c.removeFromDatabase(allExpired)

	return len(allExpired), nil
}

// collectExpiredFromDatabase scans the database for expired waves.
// Returns IDs of waves that have expired and are not in skipSet.
func (c *Cache) collectExpiredFromDatabase(skipSet map[string]struct{}) ([][]byte, error) {
	var expiredIDs [][]byte

	err := c.db.ForEach(store.BucketWaves, func(key, value []byte) error {
		// Skip if already found in memory scan.
		if _, ok := skipSet[string(key)]; ok {
			return nil
		}

		wave := &pb.Wave{}
		if err := proto.Unmarshal(value, wave); err != nil {
			// Skip malformed entries.
			return nil
		}

		if waves.IsExpired(wave) {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			expiredIDs = append(expiredIDs, keyCopy)
		}
		return nil
	})

	return expiredIDs, err
}

// collectExpiredIDs returns IDs of all expired waves in memory.
func (c *Cache) collectExpiredIDs() [][]byte {
	var expiredIDs [][]byte
	for id, wave := range c.memory {
		if waves.IsExpired(wave) {
			expiredIDs = append(expiredIDs, []byte(id))
		}
	}
	return expiredIDs
}

// removeFromMemory deletes expired waves from the memory cache.
func (c *Cache) removeFromMemory(expiredIDs [][]byte) {
	for _, id := range expiredIDs {
		delete(c.memory, string(id))
	}
}

// removeFromDatabase deletes expired waves from persistent storage.
func (c *Cache) removeFromDatabase(expiredIDs [][]byte) {
	for _, id := range expiredIDs {
		c.db.Delete(store.BucketWaves, id)
	}
}

// StartGC runs periodic garbage collection.
// Returns a cancel function to stop the GC goroutine.
func (c *Cache) StartGC(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.GarbageCollect()
			}
		}
	}()

	return cancel
}

// Close closes the cache.
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.memory = nil
	return nil
}

// waveExpirationKey generates a key for expiration index.
func waveExpirationKey(expiresAt time.Time, waveID []byte) []byte {
	key := make([]byte, 8+len(waveID))
	binary.BigEndian.PutUint64(key[:8], uint64(expiresAt.Unix()))
	copy(key[8:], waveID)
	return key
}
