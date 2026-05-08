// Package councils – Resonance-gated access to Phantom Council creation.
// Per PLAN.md: "Resonance gating on all privileged actions (gifts, marks, games, councils)".
package councils

import (
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// NewPhantomCouncilGated creates a Phantom Council after verifying the creator's
// Resonance via the injected ResonanceGate.  Returns ErrCouncilInsufficientResonance
// when the creator has fewer than CouncilMinResonance (200) Resonance points.
//
// This is the recommended entry point for council creation in production; the
// ungated NewPhantomCouncil is kept for testing and admin tooling.
func NewPhantomCouncilGated(
	creator [32]byte,
	name, purpose string,
	minResonance float64,
	maxMembers int,
	isFortressMode bool,
	gate mechanics.ResonanceGate,
) (*PhantomCouncil, error) {
	if err := mechanics.CheckResonanceGate(gate, creator, CouncilMinResonance); err != nil {
		return nil, ErrCouncilInsufficientResonance
	}
	creatorResonance, err := gate.GetResonance(creator)
	if err != nil {
		return nil, err
	}
	return NewPhantomCouncil(creator, name, purpose, minResonance, maxMembers, creatorResonance, isFortressMode)
}
