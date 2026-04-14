// Package mechanics provides shared helpers for anonymous game mechanics stores.
package mechanics

import "errors"

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
