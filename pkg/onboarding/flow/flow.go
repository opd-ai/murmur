// Package flow provides the six-phase onboarding sequence controller.
// Per ONBOARDING.md, onboarding guides new users through identity creation,
// network bootstrap, and initial Pulse Map exploration.
package flow

// Phase represents an onboarding phase.
type Phase int

const (
	// PhaseWelcome introduces MURMUR to the user.
	PhaseWelcome Phase = iota

	// PhaseIdentityCreation generates the user's keypair and sigil.
	PhaseIdentityCreation

	// PhaseModeSelection chooses the initial privacy mode.
	PhaseModeSelection

	// PhaseNetworkBootstrap connects to the P2P network.
	PhaseNetworkBootstrap

	// PhaseGuidedExploration teaches Pulse Map navigation.
	PhaseGuidedExploration

	// PhaseFirstWave prompts the user to publish their first Wave.
	PhaseFirstWave
)

// String returns the string representation of a Phase.
func (p Phase) String() string {
	switch p {
	case PhaseWelcome:
		return "Welcome"
	case PhaseIdentityCreation:
		return "Identity Creation"
	case PhaseModeSelection:
		return "Mode Selection"
	case PhaseNetworkBootstrap:
		return "Network Bootstrap"
	case PhaseGuidedExploration:
		return "Guided Exploration"
	case PhaseFirstWave:
		return "First Wave"
	default:
		return "Unknown"
	}
}

// TODO: Implement onboarding flow per ONBOARDING.md.
