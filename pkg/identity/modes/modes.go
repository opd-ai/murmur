// Package modes implements the privacy mode state machine.
// Per SHADOW_GRADIENT.md, there are four privacy modes:
// Open, Hybrid, Guarded, and Fortress.
package modes

// Mode represents a privacy mode in the Shadow Gradient.
type Mode int

const (
	// Open mode: Surface identity only, no anonymity features.
	Open Mode = iota

	// Hybrid mode: Both Surface and Specter identities active.
	Hybrid

	// Guarded mode: Enhanced privacy, limited Surface exposure.
	Guarded

	// Fortress mode: Anonymous only, full Shroud routing.
	Fortress
)

// String returns the string representation of a Mode.
func (m Mode) String() string {
	switch m {
	case Open:
		return "Open"
	case Hybrid:
		return "Hybrid"
	case Guarded:
		return "Guarded"
	case Fortress:
		return "Fortress"
	default:
		return "Unknown"
	}
}

// AllowsSurface returns true if the mode allows Surface identity usage.
func (m Mode) AllowsSurface() bool {
	return m == Open || m == Hybrid || m == Guarded
}

// AllowsSpecter returns true if the mode allows Specter identity usage.
func (m Mode) AllowsSpecter() bool {
	return m == Hybrid || m == Guarded || m == Fortress
}

// RequiresShroud returns true if the mode requires Shroud routing.
func (m Mode) RequiresShroud() bool {
	return m == Fortress
}
