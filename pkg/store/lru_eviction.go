// Package store provides Bbolt-based persistent storage for MURMUR.
// This file implements LRU eviction for space-bounded storage.
// Per TECHNICAL_IMPLEMENTATION.md §6.4, Bbolt database size must remain below 50 MiB.
package store

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

const (
	// DefaultMaxDatabaseSize is the default maximum database size (50 MiB per spec).
	DefaultMaxDatabaseSize = 50 * 1024 * 1024

	// DefaultEvictionBatchSize is how many items to evict at once when over quota.
	DefaultEvictionBatchSize = 100

	// DefaultEvictionThreshold is the percentage of max size that triggers eviction.
	DefaultEvictionThreshold = 0.9 // Start evicting at 90% capacity
)

// EvictionConfig configures the LRU eviction policy.
type EvictionConfig struct {
	// MaxDatabaseSize is the maximum database size in bytes.
	MaxDatabaseSize int64
	// EvictionBatchSize is how many items to evict at once.
	EvictionBatchSize int
	// EvictionThreshold is the fraction of MaxDatabaseSize that triggers eviction.
	EvictionThreshold float64
}

// DefaultEvictionConfig returns the default eviction configuration.
func DefaultEvictionConfig() *EvictionConfig {
	return &EvictionConfig{
		MaxDatabaseSize:   DefaultMaxDatabaseSize,
		EvictionBatchSize: DefaultEvictionBatchSize,
		EvictionThreshold: DefaultEvictionThreshold,
	}
}

// LRUCache tracks recently accessed items for LRU eviction.
// It maintains an in-memory list of access times for efficient eviction.
type LRUCache struct {
	mu       sync.Mutex
	items    *list.List               // Doubly-linked list for O(1) eviction
	index    map[string]*list.Element // O(1) lookup by key
	bucket   []byte                   // Which bucket this cache is for
	maxItems int                      // Maximum items to track (0 = unlimited)
}

// lruItem stores access metadata.
type lruItem struct {
	key        []byte
	accessTime time.Time
}

// NewLRUCache creates a new LRU cache for a bucket.
func NewLRUCache(bucket []byte, maxItems int) *LRUCache {
	return &LRUCache{
		items:    list.New(),
		index:    make(map[string]*list.Element),
		bucket:   bucket,
		maxItems: maxItems,
	}
}

// Touch marks an item as recently accessed.
func (c *LRUCache) Touch(key []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)
	if elem, ok := c.index[keyStr]; ok {
		// Move to front (most recently used).
		c.items.MoveToFront(elem)
		elem.Value.(*lruItem).accessTime = time.Now()
		return
	}

	// Add new item.
	item := &lruItem{
		key:        key,
		accessTime: time.Now(),
	}
	elem := c.items.PushFront(item)
	c.index[keyStr] = elem

	// Trim if over capacity.
	for c.maxItems > 0 && c.items.Len() > c.maxItems {
		c.removeLRU()
	}
}

// Remove removes an item from the cache.
func (c *LRUCache) Remove(key []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)
	if elem, ok := c.index[keyStr]; ok {
		c.items.Remove(elem)
		delete(c.index, keyStr)
	}
}

// GetLRUKeys returns the N least recently used keys.
func (c *LRUCache) GetLRUKeys(n int) [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	var keys [][]byte
	elem := c.items.Back()
	for elem != nil && len(keys) < n {
		item := elem.Value.(*lruItem)
		keys = append(keys, item.key)
		elem = elem.Prev()
	}
	return keys
}

// removeLRU removes the least recently used item.
func (c *LRUCache) removeLRU() {
	elem := c.items.Back()
	if elem != nil {
		item := c.items.Remove(elem).(*lruItem)
		delete(c.index, string(item.key))
	}
}

// Len returns the number of tracked items.
func (c *LRUCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.items.Len()
}

// EvictionManager manages space-bounded storage with LRU eviction.
type EvictionManager struct {
	db     *DB
	config *EvictionConfig
	caches map[string]*LRUCache // One cache per evictable bucket
	mu     sync.RWMutex
}

// NewEvictionManager creates a new eviction manager.
func NewEvictionManager(db *DB, config *EvictionConfig) *EvictionManager {
	if config == nil {
		config = DefaultEvictionConfig()
	}
	return &EvictionManager{
		db:     db,
		config: config,
		caches: make(map[string]*LRUCache),
	}
}

// RegisterBucket registers a bucket for LRU tracking.
// maxItems is the maximum items to track in the LRU cache (0 for unlimited).
func (em *EvictionManager) RegisterBucket(bucket []byte, maxItems int) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.caches[string(bucket)] = NewLRUCache(bucket, maxItems)
}

// Touch marks an item in a bucket as recently accessed.
func (em *EvictionManager) Touch(bucket, key []byte) {
	em.mu.RLock()
	cache, ok := em.caches[string(bucket)]
	em.mu.RUnlock()

	if ok {
		cache.Touch(key)
	}
}

// Remove removes an item from LRU tracking.
func (em *EvictionManager) Remove(bucket, key []byte) {
	em.mu.RLock()
	cache, ok := em.caches[string(bucket)]
	em.mu.RUnlock()

	if ok {
		cache.Remove(key)
	}
}

// CheckAndEvict checks if eviction is needed and performs it.
// Returns the number of items evicted and any error.
func (em *EvictionManager) CheckAndEvict() (int, error) {
	size, err := em.db.DatabaseSize()
	if err != nil {
		return 0, fmt.Errorf("getting database size: %w", err)
	}

	threshold := int64(float64(em.config.MaxDatabaseSize) * em.config.EvictionThreshold)
	if size < threshold {
		return 0, nil // No eviction needed
	}

	return em.evict()
}

// evict performs LRU eviction across all registered buckets.
func (em *EvictionManager) evict() (int, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	totalEvicted := 0
	perBucketLimit := em.config.EvictionBatchSize / len(em.caches)
	if perBucketLimit < 1 {
		perBucketLimit = 1
	}

	for bucketName, cache := range em.caches {
		keys := cache.GetLRUKeys(perBucketLimit)
		for _, key := range keys {
			if err := em.db.Delete([]byte(bucketName), key); err != nil {
				return totalEvicted, fmt.Errorf("deleting from %s: %w", bucketName, err)
			}
			cache.Remove(key)
			totalEvicted++
		}
	}

	return totalEvicted, nil
}

// EvictExpiredWaves removes expired Waves from the database.
// Per TECHNICAL_IMPLEMENTATION.md §8.4, this runs every 60 seconds.
// Returns the number of waves evicted.
func (db *DB) EvictExpiredWaves() (int, error) {
	now := time.Now().Unix()
	var expiredKeys [][]byte

	// Find expired waves.
	err := db.ForEach(BucketWaves, func(key, value []byte) error {
		wave := &pb.Wave{}
		if err := proto.Unmarshal(value, wave); err != nil {
			return nil // Skip malformed waves
		}

		expiresAt := wave.CreatedAt + wave.TtlSeconds
		if now > expiresAt {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			expiredKeys = append(expiredKeys, keyCopy)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("scanning for expired waves: %w", err)
	}

	// Delete expired waves.
	for _, key := range expiredKeys {
		if err := db.Delete(BucketWaves, key); err != nil {
			return 0, fmt.Errorf("deleting expired wave: %w", err)
		}
	}

	return len(expiredKeys), nil
}
