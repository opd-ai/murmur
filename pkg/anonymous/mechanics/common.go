// Package mechanics provides shared helpers for anonymous game mechanics stores.
package mechanics

import (
	"errors"

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
	var ts [8]byte
	ts[0] = byte(timestamp >> 56)
	ts[1] = byte(timestamp >> 48)
	ts[2] = byte(timestamp >> 40)
	ts[3] = byte(timestamp >> 32)
	ts[4] = byte(timestamp >> 24)
	ts[5] = byte(timestamp >> 16)
	ts[6] = byte(timestamp >> 8)
	ts[7] = byte(timestamp)
	return ts
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
