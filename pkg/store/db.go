//go:build !js

// Package store provides Bbolt-based persistent storage for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, the database is organized into buckets:
// identity, peers, waves, threads, shroud, resonance, and config.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"go.etcd.io/bbolt"
)

// NodePositionFunc is a function that resolves the 2-D Pulse Map layout
// coordinates for a node identified by its public key.  It is used by
// spatial query methods to filter game mechanics by proximity.
// Returns (x, y, true) when a position is known, (0, 0, false) otherwise.
// Per PULSE_MAP.md, coordinates are in force-directed layout units.
type NodePositionFunc func(pubkey []byte) (x, y float64, ok bool)

// DB wraps a Bbolt database with MURMUR-specific operations.
type DB struct {
	bolt         *bbolt.DB
	path         string
	nodePosition atomic.Pointer[NodePositionFunc]
}

// Open opens or creates a MURMUR database at the given path.
// It creates all required buckets if they don't exist.
func Open(path string) (*DB, error) {
	// Ensure directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Open Bbolt database.
	boltDB, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{
		bolt: boltDB,
		path: path,
	}

	// Initialize buckets.
	if err := db.initBuckets(); err != nil {
		boltDB.Close()
		return nil, fmt.Errorf("initializing buckets: %w", err)
	}

	return db, nil
}

// initBuckets creates all required buckets if they don't exist.
func (db *DB) initBuckets() error {
	buckets := [][]byte{
		BucketIdentity,
		BucketPeers,
		BucketWaves,
		BucketThreads,
		BucketShroud,
		BucketResonance,
		BucketConfig,
		// Mechanics buckets.
		BucketGifts,
		BucketMarks,
		BucketPuzzles,
		BucketHunts,
		BucketTerritories,
		BucketOracles,
		BucketForge,
		BucketShadowPlay,
		BucketCouncils,
		BucketDailyLimits,
		BucketMaskedEvents,
		BucketDevices,
		BucketContinuityChains,
	}

	return db.bolt.Update(func(tx *bbolt.Tx) error {
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("creating bucket %s: %w", name, err)
			}
		}
		return nil
	})
}

// Close closes the database.
func (db *DB) Close() error {
	return db.bolt.Close()
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// SetNodePositioner wires a layout-engine position resolver into the DB so that
// spatial query methods (GetActivePuzzlesNearNode, etc.) can filter by actual
// Pulse Map proximity rather than returning all records.
// fn may be nil to clear a previously set positioner.
// Per PULSE_MAP.md §2, position coordinates are in force-directed layout units.
func (db *DB) SetNodePositioner(fn NodePositionFunc) {
	if fn == nil {
		db.nodePosition.Store(nil)
		return
	}
	db.nodePosition.Store(&fn)
}

// getNodePosition looks up the current layout position for pubkey.
// Returns (0, 0, false) when no positioner is configured or the node is unknown.
func (db *DB) getNodePosition(pubkey []byte) (x, y float64, ok bool) {
	ptr := db.nodePosition.Load()
	if ptr == nil {
		return 0, 0, false
	}
	return (*ptr)(pubkey)
}


// Put stores a key-value pair in the specified bucket.
func (db *DB) Put(bucket, key, value []byte) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put(key, value)
	})
}

// Get retrieves a value from the specified bucket.
// Returns nil if the key doesn't exist.
func (db *DB) Get(bucket, key []byte) ([]byte, error) {
	var value []byte
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get(key)
		if v != nil {
			// Copy value since it's only valid during transaction.
			value = make([]byte, len(v))
			copy(value, v)
		}
		return nil
	})
	return value, err
}

// Delete removes a key from the specified bucket.
func (db *DB) Delete(bucket, key []byte) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete(key)
	})
}

// ForEach iterates over all key-value pairs in a bucket.
// The callback receives copies of the key and value, safe to use outside the callback.
func (db *DB) ForEach(bucket []byte, fn func(key, value []byte) error) error {
	return db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.ForEach(func(k, v []byte) error {
			// Copy key and value since they're only valid during transaction.
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			valueCopy := make([]byte, len(v))
			copy(valueCopy, v)
			return fn(keyCopy, valueCopy)
		})
	})
}

// Count returns the number of keys in a bucket.
func (db *DB) Count(bucket []byte) (int, error) {
	var count int
	err := db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		count = b.Stats().KeyN
		return nil
	})
	return count, err
}

// PrefixScan iterates over all key-value pairs in a bucket whose keys start with the given prefix.
// The callback receives copies of the key and value, safe to use outside the callback.
func (db *DB) PrefixScan(bucket, prefix []byte, fn func(key, value []byte) error) error {
	return db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && hasPrefix(k, prefix); k, v = c.Next() {
			// Copy key and value since they're only valid during transaction.
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			valueCopy := make([]byte, len(v))
			copy(valueCopy, v)
			if err := fn(keyCopy, valueCopy); err != nil {
				return err
			}
		}
		return nil
	})
}

// RangeScan iterates over all key-value pairs in a bucket whose keys are >= start and < end.
// If end is nil, iterates from start to the end of the bucket.
// The callback receives copies of the key and value, safe to use outside the callback.
func (db *DB) RangeScan(bucket, start, end []byte, fn func(key, value []byte) error) error {
	return db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		c := b.Cursor()
		for k, v := c.Seek(start); k != nil; k, v = c.Next() {
			// Stop if we've passed the end key.
			if end != nil && compareBytes(k, end) >= 0 {
				break
			}
			// Copy key and value since they're only valid during transaction.
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			valueCopy := make([]byte, len(v))
			copy(valueCopy, v)
			if err := fn(keyCopy, valueCopy); err != nil {
				return err
			}
		}
		return nil
	})
}

// hasPrefix returns true if key starts with prefix.
func hasPrefix(key, prefix []byte) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i := range prefix {
		if key[i] != prefix[i] {
			return false
		}
	}
	return true
}

// compareBytes compares two byte slices lexicographically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareBytes(a, b []byte) int {
	if cmp := compareCommonPrefix(a, b); cmp != 0 {
		return cmp
	}
	return compareLengths(a, b)
}

// compareCommonPrefix compares the common prefix of two byte slices.
func compareCommonPrefix(a, b []byte) int {
	minLen := minInt(len(a), len(b))
	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// compareLengths compares the lengths of two byte slices.
func compareLengths(a, b []byte) int {
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BucketStats contains statistics for a single bucket.
type BucketStats struct {
	// KeyCount is the number of keys in the bucket.
	KeyCount int
	// TotalKeySize is the total size of all keys in bytes.
	TotalKeySize int64
	// TotalValueSize is the total size of all values in bytes.
	TotalValueSize int64
	// TotalSize is TotalKeySize + TotalValueSize.
	TotalSize int64
}

// Stats returns statistics for a bucket.
func (db *DB) Stats(bucket []byte) (*BucketStats, error) {
	stats := &BucketStats{}

	err := db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		// Get key count from Bbolt's internal stats.
		stats.KeyCount = b.Stats().KeyN

		// Calculate sizes by iterating.
		return b.ForEach(func(k, v []byte) error {
			stats.TotalKeySize += int64(len(k))
			stats.TotalValueSize += int64(len(v))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	stats.TotalSize = stats.TotalKeySize + stats.TotalValueSize
	return stats, nil
}

// AllStats returns statistics for all buckets.
func (db *DB) AllStats() (map[string]*BucketStats, error) {
	buckets := [][]byte{
		BucketIdentity,
		BucketPeers,
		BucketWaves,
		BucketThreads,
		BucketShroud,
		BucketResonance,
		BucketConfig,
		BucketGifts,
		BucketMarks,
		BucketPuzzles,
		BucketHunts,
		BucketTerritories,
		BucketOracles,
		BucketForge,
		BucketShadowPlay,
		BucketCouncils,
		BucketDailyLimits,
	}

	result := make(map[string]*BucketStats)
	for _, bucket := range buckets {
		stats, err := db.Stats(bucket)
		if err != nil {
			return nil, fmt.Errorf("getting stats for %s: %w", bucket, err)
		}
		result[string(bucket)] = stats
	}
	return result, nil
}

// DatabaseSize returns the total size of the database file in bytes.
func (db *DB) DatabaseSize() (int64, error) {
	info, err := os.Stat(db.path)
	if err != nil {
		return 0, fmt.Errorf("getting database file info: %w", err)
	}
	return info.Size(), nil
}
