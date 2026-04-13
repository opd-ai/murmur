// Package store provides Bbolt-based persistent storage for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, the database is organized into buckets:
// identity, peers, waves, threads, shroud, resonance, and config.
package store

import (
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// Bucket names per TECHNICAL_IMPLEMENTATION.md §1.5.
var (
	BucketIdentity  = []byte("identity")
	BucketPeers     = []byte("peers")
	BucketWaves     = []byte("waves")
	BucketThreads   = []byte("threads")
	BucketShroud    = []byte("shroud")
	BucketResonance = []byte("resonance")
	BucketConfig    = []byte("config")
	// Mechanics buckets for Anonymous Layer game state persistence.
	BucketGifts       = []byte("gifts")
	BucketMarks       = []byte("marks")
	BucketPuzzles     = []byte("puzzles")
	BucketHunts       = []byte("hunts")
	BucketTerritories = []byte("territories")
	BucketOracles     = []byte("oracles")
	BucketForge       = []byte("forge")
	BucketShadowPlay  = []byte("shadowplay")
	BucketCouncils    = []byte("councils")
	BucketDailyLimits = []byte("daily_limits")
)

// DB wraps a Bbolt database with MURMUR-specific operations.
type DB struct {
	bolt *bbolt.DB
	path string
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
