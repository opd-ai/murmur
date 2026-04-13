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
const GCInterval = 60 * time.Second

// DefaultCacheSize is the default maximum number of Waves to cache.
const DefaultCacheSize = 10000

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
	if wave == nil || len(wave.WaveId) == 0 {
		return ErrInvalidWave
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrStoreClosed
	}

	waveID := string(wave.WaveId)

	// Check cache size.
	if len(c.memory) >= c.maxSize {
		// Evict expired waves first.
		c.evictExpiredLocked()

		// If still full, reject.
		if len(c.memory) >= c.maxSize {
			return ErrCacheFull
		}
	}

	// Store in memory.
	c.memory[waveID] = wave

	// Persist to database.
	data, err := proto.Marshal(wave)
	if err != nil {
		return err
	}

	return c.db.Put(store.BucketWaves, wave.WaveId, data)
}

// Get retrieves a Wave by ID from cache or database.
func (c *Cache) Get(waveID []byte) (*pb.Wave, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrStoreClosed
	}

	// Check memory cache first.
	if wave, ok := c.memory[string(waveID)]; ok {
		c.mu.RUnlock()
		return wave, nil
	}
	c.mu.RUnlock()

	// Try database.
	data, err := c.db.Get(store.BucketWaves, waveID)
	if err != nil {
		return nil, err
	}

	// Bbolt returns nil for missing keys.
	if data == nil {
		return nil, ErrNotFound
	}

	wave := &pb.Wave{}
	if err := proto.Unmarshal(data, wave); err != nil {
		return nil, err
	}

	// Add to memory cache.
	c.mu.Lock()
	if len(c.memory) < c.maxSize {
		c.memory[string(waveID)] = wave
	}
	c.mu.Unlock()

	return wave, nil
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
func (c *Cache) GarbageCollect() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, ErrStoreClosed
	}

	// Collect expired wave IDs.
	var expiredIDs [][]byte
	for id, wave := range c.memory {
		if waves.IsExpired(wave) {
			expiredIDs = append(expiredIDs, []byte(id))
		}
	}

	// Remove from memory.
	for _, id := range expiredIDs {
		delete(c.memory, string(id))
	}

	// Remove from database.
	for _, id := range expiredIDs {
		c.db.Delete(store.BucketWaves, id)
	}

	return len(expiredIDs), nil
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
