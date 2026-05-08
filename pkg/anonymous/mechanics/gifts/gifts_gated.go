// Package gifts – Resonance-gated access to Phantom Gift creation.
// Per PLAN.md: "Resonance gating on all privileged actions (gifts, marks, games, councils)".
package gifts

import (
	"crypto/ed25519"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// GiftStoreGated wraps GiftStore and enforces Resonance gating by automatically
// fetching the caller's score from a ResonanceGate instead of requiring callers
// to pass the score manually.
type GiftStoreGated struct {
	store *GiftStore
	gate  mechanics.ResonanceGate
}

// NewGiftStoreGated creates a gated Gift store.  All CreateGift calls on the
// returned value will be blocked unless senderKey's Resonance satisfies the
// minimum threshold required for the requested effect.
func NewGiftStoreGated(store *GiftStore, gate mechanics.ResonanceGate) *GiftStoreGated {
	return &GiftStoreGated{store: store, gate: gate}
}

// CreateGift creates a Phantom Gift after verifying the sender's Resonance via
// the injected ResonanceGate.  The effect-specific minimum (GiftTierBasic /
// Expanded / Premium) is enforced in addition to the gate check.
func (g *GiftStoreGated) CreateGift(
	senderKey [32]byte,
	recipientKey []byte,
	effect EffectType,
	signingKey ed25519.PrivateKey,
) (*Gift, error) {
	resonance, err := g.gate.GetResonance(senderKey)
	if err != nil {
		return nil, err
	}
	if resonance < GiftTierBasic {
		return nil, ErrInsufficientResonance
	}
	return g.store.CreateGift(senderKey, recipientKey, effect, resonance, signingKey)
}
