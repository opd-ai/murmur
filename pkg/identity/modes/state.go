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
	ErrInvalidTransition      = errors.New("invalid mode transition")
	ErrCooldownActive         = errors.New("mode transition cooldown active")
	ErrMissingRequirement     = errors.New("missing requirement for mode")
	ErrTrafficPaddingStartErr = errors.New("failed to start traffic padding")
	ErrTrafficPaddingStopErr  = errors.New("failed to stop traffic padding")
)

// TransitionCooldown is the minimum time between mode transitions.
const TransitionCooldown = 30 * time.Second

// Manager handles privacy mode state transitions.
// Per SHADOW_GRADIENT.md, mode transitions must follow allowed paths
// and respect cooldown periods.
type Manager struct {
	mu               sync.RWMutex
	current          Mode
	lastChange       time.Time
	cooldown         time.Duration
	hasSpecter       bool // Whether a Specter identity exists
	hasShroud        bool // Whether Shroud routing is available
	listeners        []func(old, new Mode)
	specterDestroyer func() error // Called when transitioning to non-Specter mode

	// Traffic padding control per SHADOW_GRADIENT.md §Traffic Padding.
	paddingEnabled bool
	paddingStarter func() error // Called when entering Guarded/Fortress
	paddingStopper func() error // Called when leaving Guarded/Fortress
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

// ErrSpecterDestructionFailed indicates Specter destruction callback failed.
var ErrSpecterDestructionFailed = errors.New("failed to destroy Specter identity")

// Transition attempts to switch to a new privacy mode.
// Per SHADOW_GRADIENT.md, transitioning from a Specter-enabled mode
// (Hybrid/Guarded/Fortress) to Open destroys the Specter identity.
// Traffic padding is activated/deactivated when entering/leaving Guarded/Fortress.
func (m *Manager) Transition(target Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.canTransitionLocked(target); err != nil {
		return err
	}

	old := m.current

	// Per SHADOW_GRADIENT.md: when transitioning from a Specter-enabled mode
	// to Open, the Specter keypair must be destroyed to prevent correlation.
	if old.AllowsSpecter() && !target.AllowsSpecter() {
		if m.specterDestroyer != nil {
			if err := m.specterDestroyer(); err != nil {
				return ErrSpecterDestructionFailed
			}
			m.hasSpecter = false
		}
	}

	// Handle traffic padding activation/deactivation per SHADOW_GRADIENT.md §Traffic Padding.
	if err := m.handleTrafficPaddingTransition(old, target); err != nil {
		return err
	}

	m.current = target
	m.lastChange = time.Now()

	// Notify listeners.
	for _, listener := range m.listeners {
		go listener(old, target)
	}

	return nil
}

// handleTrafficPaddingTransition starts or stops traffic padding based on mode change.
func (m *Manager) handleTrafficPaddingTransition(old, target Mode) error {
	oldRequires := old.RequiresTrafficPadding()
	newRequires := target.RequiresTrafficPadding()

	if !oldRequires && newRequires {
		// Entering Guarded/Fortress - start traffic padding.
		if m.paddingStarter != nil {
			if err := m.paddingStarter(); err != nil {
				return ErrTrafficPaddingStartErr
			}
		}
		m.paddingEnabled = true
	} else if oldRequires && !newRequires {
		// Leaving Guarded/Fortress - stop traffic padding.
		if m.paddingStopper != nil {
			if err := m.paddingStopper(); err != nil {
				return ErrTrafficPaddingStopErr
			}
		}
		m.paddingEnabled = false
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

// SetSpecterDestroyer sets a callback to destroy Specter identity when
// transitioning from a Specter-enabled mode (Hybrid/Guarded/Fortress) to
// Open mode. Per SHADOW_GRADIENT.md, Specter keypair must be zeroed and
// destroyed when anonymity is no longer active to prevent correlation.
func (m *Manager) SetSpecterDestroyer(destroyer func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.specterDestroyer = destroyer
}

// SetTrafficPaddingCallbacks sets callbacks for traffic padding control.
// Per SHADOW_GRADIENT.md §Traffic Padding, Guarded and Fortress modes require
// constant-rate dummy packets (2/sec) to defeat traffic analysis.
// The starter is called when entering Guarded/Fortress from Open/Hybrid.
// The stopper is called when leaving Guarded/Fortress to Open/Hybrid.
func (m *Manager) SetTrafficPaddingCallbacks(starter, stopper func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.paddingStarter = starter
	m.paddingStopper = stopper
}

// IsTrafficPaddingEnabled returns true if traffic padding is currently active.
func (m *Manager) IsTrafficPaddingEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.paddingEnabled
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
	SurfaceAllowed       bool
	SpecterAllowed       bool
	ShroudRequired       bool
	TrafficPaddingActive bool // Per SHADOW_GRADIENT.md, Guarded/Fortress use traffic padding
	AnonymityLevel       int  // 0=none, 1=low, 2=medium, 3=high
}

// Capabilities returns the capabilities of a mode.
func (m Mode) Capabilities() ModeCapabilities {
	switch m {
	case Open:
		return ModeCapabilities{
			SurfaceAllowed:       true,
			SpecterAllowed:       false,
			ShroudRequired:       false,
			TrafficPaddingActive: false,
			AnonymityLevel:       0,
		}
	case Hybrid:
		return ModeCapabilities{
			SurfaceAllowed:       true,
			SpecterAllowed:       true,
			ShroudRequired:       false,
			TrafficPaddingActive: false,
			AnonymityLevel:       1,
		}
	case Guarded:
		return ModeCapabilities{
			SurfaceAllowed:       true,
			SpecterAllowed:       true,
			ShroudRequired:       false,
			TrafficPaddingActive: true, // Per SHADOW_GRADIENT.md §Traffic Padding
			AnonymityLevel:       2,
		}
	case Fortress:
		return ModeCapabilities{
			SurfaceAllowed:       false,
			SpecterAllowed:       true,
			ShroudRequired:       true,
			TrafficPaddingActive: true, // Per SHADOW_GRADIENT.md §Traffic Padding
			AnonymityLevel:       3,
		}
	default:
		return ModeCapabilities{}
	}
}

// RequiresTrafficPadding returns true if the mode requires traffic padding.
// Per SHADOW_GRADIENT.md, Guarded and Fortress modes use constant-rate padding.
func (m Mode) RequiresTrafficPadding() bool {
	return m == Guarded || m == Fortress
}
