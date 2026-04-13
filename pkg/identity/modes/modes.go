// Package modes implements the privacy mode state machine.
// Per SHADOW_GRADIENT.md, there are four privacy modes:
// Open, Hybrid, Guarded, and Fortress.
package modes

import (
	"errors"
	"sync"
	"time"
)

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

// Errors for mode operations.
var (
	ErrInvalidTransition  = errors.New("invalid mode transition")
	ErrCooldownActive     = errors.New("mode transition cooldown active")
	ErrMissingRequirement = errors.New("missing requirement for mode")
)

// TransitionCooldown is the minimum time between mode transitions.
const TransitionCooldown = 30 * time.Second

// Manager handles privacy mode state transitions.
// Per SHADOW_GRADIENT.md, mode transitions must follow allowed paths
// and respect cooldown periods.
type Manager struct {
	mu         sync.RWMutex
	current    Mode
	lastChange time.Time
	cooldown   time.Duration
	hasSpecter bool // Whether a Specter identity exists
	hasShroud  bool // Whether Shroud routing is available
	listeners  []func(old, new Mode)
}

// NewManager creates a new mode manager starting in Open mode.
func NewManager() *Manager {
	return &Manager{
		current:  Open,
		cooldown: TransitionCooldown,
	}
}

// NewManagerWithConfig creates a manager with custom settings.
func NewManagerWithConfig(initial Mode, cooldown time.Duration) *Manager {
	return &Manager{
		current:  initial,
		cooldown: cooldown,
	}
}

// Current returns the current privacy mode.
func (m *Manager) Current() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.current
}

// CanTransition checks if a mode transition is allowed.
func (m *Manager) CanTransition(target Mode) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.canTransitionLocked(target)
}

// canTransitionLocked checks transition validity (must hold read lock).
func (m *Manager) canTransitionLocked(target Mode) error {
	// Same mode is always allowed.
	if m.current == target {
		return nil
	}

	// Check cooldown.
	if time.Since(m.lastChange) < m.cooldown {
		return ErrCooldownActive
	}

	// Validate transition path per SHADOW_GRADIENT.md.
	if !isValidTransition(m.current, target) {
		return ErrInvalidTransition
	}

	// Check requirements for target mode.
	if target.AllowsSpecter() && !m.hasSpecter {
		return ErrMissingRequirement
	}
	if target.RequiresShroud() && !m.hasShroud {
		return ErrMissingRequirement
	}

	return nil
}

// isValidTransition checks if a mode transition is allowed.
// Per SHADOW_GRADIENT.md, transitions must be adjacent or from Fortress.
func isValidTransition(from, to Mode) bool {
	// All modes can return to Open.
	if to == Open {
		return true
	}

	// Can always move one step up or down the gradient.
	diff := int(to) - int(from)
	if diff == 1 || diff == -1 {
		return true
	}

	// Fortress can jump to any mode (emergency exit).
	if from == Fortress {
		return true
	}

	return false
}

// Transition attempts to switch to a new privacy mode.
func (m *Manager) Transition(target Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.canTransitionLocked(target); err != nil {
		return err
	}

	old := m.current
	m.current = target
	m.lastChange = time.Now()

	// Notify listeners.
	for _, listener := range m.listeners {
		go listener(old, target)
	}

	return nil
}

// ForceTransition sets the mode without validation (for initialization).
func (m *Manager) ForceTransition(target Mode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := m.current
	m.current = target
	m.lastChange = time.Now()

	for _, listener := range m.listeners {
		go listener(old, target)
	}
}

// SetSpecterAvailable marks whether a Specter identity exists.
func (m *Manager) SetSpecterAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasSpecter = available
}

// SetShroudAvailable marks whether Shroud routing is available.
func (m *Manager) SetShroudAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasShroud = available
}

// OnTransition registers a callback for mode transitions.
func (m *Manager) OnTransition(callback func(old, new Mode)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listeners = append(m.listeners, callback)
}

// TimeSinceChange returns time since last mode change.
func (m *Manager) TimeSinceChange() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return time.Since(m.lastChange)
}

// CooldownRemaining returns remaining cooldown time.
func (m *Manager) CooldownRemaining() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	remaining := m.cooldown - time.Since(m.lastChange)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// AvailableModes returns modes that can be transitioned to.
func (m *Manager) AvailableModes() []Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []Mode
	for _, mode := range []Mode{Open, Hybrid, Guarded, Fortress} {
		if m.canTransitionLocked(mode) == nil {
			available = append(available, mode)
		}
	}
	return available
}

// ModeCapabilities describes what a mode enables.
type ModeCapabilities struct {
	SurfaceAllowed bool
	SpecterAllowed bool
	ShroudRequired bool
	AnonymityLevel int // 0=none, 1=low, 2=medium, 3=high
}

// Capabilities returns the capabilities of a mode.
func (m Mode) Capabilities() ModeCapabilities {
	switch m {
	case Open:
		return ModeCapabilities{
			SurfaceAllowed: true,
			SpecterAllowed: false,
			ShroudRequired: false,
			AnonymityLevel: 0,
		}
	case Hybrid:
		return ModeCapabilities{
			SurfaceAllowed: true,
			SpecterAllowed: true,
			ShroudRequired: false,
			AnonymityLevel: 1,
		}
	case Guarded:
		return ModeCapabilities{
			SurfaceAllowed: true,
			SpecterAllowed: true,
			ShroudRequired: false,
			AnonymityLevel: 2,
		}
	case Fortress:
		return ModeCapabilities{
			SurfaceAllowed: false,
			SpecterAllowed: true,
			ShroudRequired: true,
			AnonymityLevel: 3,
		}
	default:
		return ModeCapabilities{}
	}
}
