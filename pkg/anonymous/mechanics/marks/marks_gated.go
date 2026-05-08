// Package marks – Resonance-gated access to Specter Mark placement.
// Per PLAN.md: "Resonance gating on all privileged actions (gifts, marks, games, councils)".
package marks

import (
	"crypto/ed25519"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// MarkStoreGated wraps MarkStore and enforces Resonance gating by automatically
// fetching the caller's score from a ResonanceGate.
type MarkStoreGated struct {
	store *MarkStore
	gate  mechanics.ResonanceGate
}

// NewMarkStoreGated creates a gated Mark store.  All PlaceMark calls will be
// blocked unless markerKey's Resonance is >= MarkMinResonance (100).
func NewMarkStoreGated(store *MarkStore, gate mechanics.ResonanceGate) *MarkStoreGated {
	return &MarkStoreGated{store: store, gate: gate}
}

// PlaceMark places a Specter Mark after verifying the marker's Resonance via
// the injected ResonanceGate.  Returns ErrInsufficientResonance when the
// threshold is not met.
func (m *MarkStoreGated) PlaceMark(
	markerKey [32]byte,
	targetKey []byte,
	category MarkCategory,
	note string,
	signingKey ed25519.PrivateKey,
) (*Mark, error) {
	resonance, err := m.gate.GetResonance(markerKey)
	if err != nil {
		return nil, err
	}
	if resonance < MarkMinResonance {
		return nil, ErrMarkInsufficientResonance
	}
	return m.store.PlaceMark(markerKey, targetKey, category, note, resonance, signingKey)
}
