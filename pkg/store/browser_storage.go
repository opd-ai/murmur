//go:build js && wasm

// Package store provides browser-compatible storage via localStorage.
package store

import (
	"fmt"
	"sync"
	"syscall/js"
)

// BrowserStorage implements the DB interface using browser localStorage.
// For WASM environments, this replaces file-based Bbolt with in-browser storage.
// Data is organized hierarchically: localStorage[bucket:key] = value.
type BrowserStorage struct {
	mu    sync.RWMutex
	items map[string]map[string][]byte
}

// NewBrowserStorage creates a new in-memory storage adapter for browser environments.
// It uses an in-memory map directly rather than localStorage for performance and flexibility.
// In production, this could be backed by IndexedDB for larger data volumes.
func NewBrowserStorage() *BrowserStorage {
	return &BrowserStorage{
		items: make(map[string]map[string][]byte),
	}
}

// Close is a no-op for browser storage.
func (bs *BrowserStorage) Close() error {
	return nil
}

// Path returns a synthetic path for logging purposes.
func (bs *BrowserStorage) Path() string {
	return "memory://browser-storage"
}

// Put stores a key-value pair in the specified bucket.
func (bs *BrowserStorage) Put(bucket, key, value []byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.items[string(bucket)] == nil {
		bs.items[string(bucket)] = make(map[string][]byte)
	}

	// Copy the value to avoid external mutations
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	bs.items[string(bucket)][string(key)] = valueCopy
	return nil
}

// Get retrieves a value from the specified bucket.
// Returns nil if the key doesn't exist.
func (bs *BrowserStorage) Get(bucket, key []byte) ([]byte, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	bucketMap := bs.items[string(bucket)]
	if bucketMap == nil {
		return nil, nil
	}

	value := bucketMap[string(key)]
	if value == nil {
		return nil, nil
	}

	// Return a copy to prevent external mutations
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	return valueCopy, nil
}

// Delete removes a key from the specified bucket.
func (bs *BrowserStorage) Delete(bucket, key []byte) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bucketMap := bs.items[string(bucket)]
	if bucketMap != nil {
		delete(bucketMap, string(key))
	}
	return nil
}

// Scan returns all key-value pairs in the specified bucket.
// Callback is called for each key-value pair. Return false from callback to stop iteration.
func (bs *BrowserStorage) Scan(bucket []byte, callback func(key, value []byte) bool) error {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	bucketMap := bs.items[string(bucket)]
	if bucketMap == nil {
		return nil
	}

	for keyStr, value := range bucketMap {
		key := []byte(keyStr)
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		if !callback(key, valueCopy) {
			break
		}
	}
	return nil
}

// Update executes a function within a write transaction.
// For browser storage, this is just a lock wrapper.
func (bs *BrowserStorage) Update(fn func(*Tx) error) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	tx := &Tx{storage: bs, isWrite: true}
	return fn(tx)
}

// View executes a function within a read transaction.
// For browser storage, this is just a lock wrapper.
func (bs *BrowserStorage) View(fn func(*Tx) error) error {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	tx := &Tx{storage: bs, isWrite: false}
	return fn(tx)
}

// Tx represents a transaction for browser storage.
type Tx struct {
	storage *BrowserStorage
	isWrite bool
}

// Bucket returns a bucket for the given name.
func (tx *Tx) Bucket(name []byte) *TxBucket {
	return &TxBucket{
		tx:   tx,
		name: name,
	}
}

// TxBucket represents a bucket within a transaction.
type TxBucket struct {
	tx   *Tx
	name []byte
}

// Get retrieves a value from the bucket.
func (b *TxBucket) Get(key []byte) []byte {
	bucketMap := b.tx.storage.items[string(b.name)]
	if bucketMap == nil {
		return nil
	}
	return bucketMap[string(key)]
}

// Put stores a key-value pair in the bucket.
func (b *TxBucket) Put(key, value []byte) error {
	if !b.tx.isWrite {
		return fmt.Errorf("cannot write in read-only transaction")
	}

	if b.tx.storage.items[string(b.name)] == nil {
		b.tx.storage.items[string(b.name)] = make(map[string][]byte)
	}

	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	b.tx.storage.items[string(b.name)][string(key)] = valueCopy
	return nil
}

// Delete removes a key from the bucket.
func (b *TxBucket) Delete(key []byte) error {
	if !b.tx.isWrite {
		return fmt.Errorf("cannot write in read-only transaction")
	}

	if bucketMap := b.tx.storage.items[string(b.name)]; bucketMap != nil {
		delete(bucketMap, string(key))
	}
	return nil
}

// ForEach iterates over all key-value pairs in the bucket.
func (b *TxBucket) ForEach(fn func(k, v []byte) error) error {
	bucketMap := b.tx.storage.items[string(b.name)]
	if bucketMap == nil {
		return nil
	}

	for keyStr, value := range bucketMap {
		key := []byte(keyStr)
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		if err := fn(key, valueCopy); err != nil {
			return err
		}
	}
	return nil
}

// LogToConsole sends a log message to the browser console.
func LogToConsole(msg string) {
	js.Global().Get("console").Call("log", msg)
}
