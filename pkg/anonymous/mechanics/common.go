// Package mechanics provides shared helpers for anonymous game mechanics stores.
package mechanics

import (
	"errors"

	"github.com/opd-ai/murmur/pkg/encoding"
	"github.com/opd-ai/murmur/pkg/store"
)

// ResonanceGate provides an interface for checking Specter Resonance scores.
// This is used for Resonance gating on mini-game actions per ROADMAP.md line 414.
type ResonanceGate interface {
	// GetResonance returns the Resonance score for a Specter.
	GetResonance(specterKey [32]byte) (int, error)
}

// ResonanceGatingError is returned when Resonance requirements are not met.
var ErrResonanceRequirementNotMet = errors.New("resonance requirement not met")

// CheckResonanceGate verifies a Specter meets the minimum Resonance threshold.
// Returns nil if the requirement is met, ErrResonanceRequirementNotMet otherwise.
func CheckResonanceGate(gate ResonanceGate, specterKey [32]byte, minResonance int) error {
	if gate == nil {
		// No gate configured - allow all (for testing or permissionless mode).
		return nil
	}
	resonance, err := gate.GetResonance(specterKey)
	if err != nil {
		return err
	}
	if resonance < minResonance {
		return ErrResonanceRequirementNotMet
	}
	return nil
}

// ZKClaimVerifier provides an interface for verifying ZK claims.
// Per ROADMAP.md line 400: "ZK claims used for Council admission and mini-game thresholds".
type ZKClaimVerifier interface {
	// VerifyResonanceClaim verifies that a Specter has Resonance >= threshold.
	// The proof bytes are a serialized ZKClaim.
	VerifyResonanceClaim(proof []byte, minResonance int64) error
}

// ErrInvalidZKClaim is returned when ZK claim verification fails.
var ErrInvalidZKClaim = errors.New("invalid or failed ZK claim")

// ErrMissingZKClaim is returned when a ZK claim is required but not provided.
var ErrMissingZKClaim = errors.New("ZK claim required but not provided")

// GarbageCollectHistory removes old entries from a history slice and lookup map.
// It returns the number of entries removed and the new history slice.
// The caller must hold the write lock and update their fields with the results.
// The getID function extracts the ID from each item for map deletion.
func GarbageCollectHistory[T any](history []T, lookup map[[32]byte]T, maxHistory int, getID func(T) [32]byte) ([]T, int) {
	if len(history) <= maxHistory {
		return history, 0
	}

	removed := len(history) - maxHistory

	for i := 0; i < removed; i++ {
		delete(lookup, getID(history[i]))
	}

	return history[removed:], removed
}

// EncodeTimestamp encodes a Unix timestamp in seconds to 8 bytes (big-endian).
// This is used for event signature data construction across mechanics.
func EncodeTimestamp(timestamp int64) [8]byte {
	return encoding.Int64ToBytes(timestamp)
}

// Expirable represents an object that can expire based on time.
type Expirable interface {
	IsExpired() bool
}

// CollectExpiredFromMap scans a map for expired items and returns their IDs.
// The caller must hold the read lock. This is used by persistent stores
// to identify items to delete from both memory and database.
func CollectExpiredFromMap[T Expirable](items map[[32]byte]T) [][32]byte {
	var expiredIDs [][32]byte
	for id, item := range items {
		if item.IsExpired() {
			expiredIDs = append(expiredIDs, id)
		}
	}
	return expiredIDs
}

// GetItemByID retrieves an item from a map by ID, checking for expiration.
// This consolidates the duplicate Get pattern found in gifts.go and marks.go.
// Returns (item, nil) if found and not expired, (zero value, notFoundErr) if not found or expired.
func GetItemByID[T Expirable](items map[[32]byte]T, id [32]byte, notFoundErr error) (T, error) {
	var zero T
	item, ok := items[id]
	if !ok {
		return zero, notFoundErr
	}
	if item.IsExpired() {
		return zero, notFoundErr
	}
	return item, nil
}

// DeleteFromDB deletes a list of IDs from a Bbolt bucket.
// It ignores errors for robustness during garbage collection.
func DeleteFromDB(db *store.DB, bucket []byte, ids [][32]byte) {
	if db == nil {
		return
	}
	for _, id := range ids {
		_ = db.Delete(bucket, id[:])
	}
}

// StoreWithGC represents an in-memory store that supports garbage collection.
type StoreWithGC interface {
	GarbageCollect() int
}

// GarbageCollectWithDB performs GC on an in-memory store and syncs deletions to DB.
// itemsGetter retrieves the items map for expired item detection (under read lock).
// This consolidates the persistent GC pattern used across gifts, marks, and other mechanics.
func GarbageCollectWithDB[T Expirable](
	store StoreWithGC,
	db *store.DB,
	bucket []byte,
	itemsGetter func() map[[32]byte]T,
	lockFn func(),
	unlockFn func(),
) int {
	// Get list of expired item IDs before cleanup.
	lockFn()
	expiredIDs := CollectExpiredFromMap(itemsGetter())
	unlockFn()

	// Call parent garbage collection.
	removed := store.GarbageCollect()

	// Remove from database.
	DeleteFromDB(db, bucket, expiredIDs)

	return removed
}

// ValidateReceivedItem validates a converted item: checks duplicate and expiration.
// Consolidates the duplicate pattern from gifts_publisher.go:156-168 and marks_publisher.go:156-168.
//
// Steps after proto conversion:
//  1. Check for duplicate in store
//  2. Check expiration
//  3. Add to store
//
// Returns nil on success, appropriate error on validation/storage failure.
func ValidateReceivedItem[T Expirable](
	item T,
	checkDuplicate func() (T, error),
	addToStore func(T) error,
	duplicateErr error,
	expiredErr error,
) error {
	// Check for duplicate.
	var zero T
	existing, err := checkDuplicate()
	if err == nil && any(existing) != any(zero) {
		return duplicateErr
	}

	// Check expiration.
	if item.IsExpired() {
		return expiredErr
	}

	// Add to store.
	return addToStore(item)
}
