// Package flow provides the six-phase onboarding sequence controller.
// Per ONBOARDING.md, onboarding guides new users through identity creation,
// network bootstrap, and initial Pulse Map exploration.
package flow

import (
	"sync"
	"time"
)

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

	// PhaseComplete indicates onboarding is finished.
	PhaseComplete
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
	case PhaseComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// PhaseCount returns the total number of onboarding phases.
const PhaseCount = 6

// PhaseProgress contains progress state for a phase.
type PhaseProgress struct {
	Started   bool
	Completed bool
	StartTime time.Time
	EndTime   time.Time
	Data      map[string]any // Phase-specific data
}

// Controller manages the onboarding flow state machine.
// Per ONBOARDING.md, it tracks progress and allows resumption.
type Controller struct {
	mu            sync.RWMutex
	currentPhase  Phase
	progress      map[Phase]*PhaseProgress
	startTime     time.Time
	completedTime time.Time
	skipped       bool
	callbacks     Callbacks
}

// Callbacks provides hooks for phase transitions.
type Callbacks struct {
	OnPhaseStart    func(phase Phase)
	OnPhaseComplete func(phase Phase)
	OnFlowComplete  func(totalTime time.Duration)
	OnError         func(phase Phase, err error)
}

// NewController creates a new onboarding controller.
func NewController(callbacks Callbacks) *Controller {
	c := &Controller{
		currentPhase: PhaseWelcome,
		progress:     make(map[Phase]*PhaseProgress),
		callbacks:    callbacks,
	}
	// Initialize progress for all phases
	for p := PhaseWelcome; p <= PhaseFirstWave; p++ {
		c.progress[p] = &PhaseProgress{
			Data: make(map[string]any),
		}
	}
	return c
}

// Start begins the onboarding flow.
func (c *Controller) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.startTime = time.Now()
	c.startPhase(PhaseWelcome)
}

// CurrentPhase returns the current phase.
func (c *Controller) CurrentPhase() Phase {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentPhase
}

// IsComplete returns true if onboarding is finished.
func (c *Controller) IsComplete() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentPhase == PhaseComplete
}

// IsSkipped returns true if onboarding was skipped.
func (c *Controller) IsSkipped() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.skipped
}

// Progress returns the progress for a specific phase.
func (c *Controller) Progress(phase Phase) *PhaseProgress {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if p, ok := c.progress[phase]; ok {
		return p
	}
	return nil
}

// OverallProgress returns the percentage of onboarding completed.
func (c *Controller) OverallProgress() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentPhase == PhaseComplete {
		return 100.0
	}

	completed := 0
	for p := PhaseWelcome; p <= PhaseFirstWave; p++ {
		if c.progress[p].Completed {
			completed++
		}
	}
	return float64(completed) / float64(PhaseCount) * 100.0
}

// TotalTime returns the total time spent in onboarding.
func (c *Controller) TotalTime() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.startTime.IsZero() {
		return 0
	}
	if !c.completedTime.IsZero() {
		return c.completedTime.Sub(c.startTime)
	}
	return time.Since(c.startTime)
}

// CompleteCurrentPhase marks the current phase as complete and advances.
func (c *Controller) CompleteCurrentPhase() {
	c.mu.Lock()

	if c.currentPhase == PhaseComplete {
		c.mu.Unlock()
		return
	}

	// Mark current phase complete
	if p, ok := c.progress[c.currentPhase]; ok {
		p.Completed = true
		p.EndTime = time.Now()
	}

	completedPhase := c.currentPhase

	if c.callbacks.OnPhaseComplete != nil {
		go c.callbacks.OnPhaseComplete(completedPhase)
	}

	// Advance to next phase
	if c.currentPhase < PhaseFirstWave {
		c.currentPhase++
		c.startPhase(c.currentPhase)
		c.mu.Unlock()
	} else {
		// Flow complete
		c.currentPhase = PhaseComplete
		c.completedTime = time.Now()
		totalTime := c.completedTime.Sub(c.startTime)
		c.mu.Unlock()

		if c.callbacks.OnFlowComplete != nil {
			go c.callbacks.OnFlowComplete(totalTime)
		}
	}
}

// Skip skips the remaining onboarding phases.
func (c *Controller) Skip() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.skipped = true
	c.currentPhase = PhaseComplete
	c.completedTime = time.Now()
}

// SetPhaseData stores data for a specific phase.
func (c *Controller) SetPhaseData(phase Phase, key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p, ok := c.progress[phase]; ok {
		if p.Data == nil {
			p.Data = make(map[string]any)
		}
		p.Data[key] = value
	}
}

// GetPhaseData retrieves data for a specific phase.
func (c *Controller) GetPhaseData(phase Phase, key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if p, ok := c.progress[phase]; ok && p.Data != nil {
		return p.Data[key]
	}
	return nil
}

// startPhase begins a new phase.
func (c *Controller) startPhase(phase Phase) {
	if p, ok := c.progress[phase]; ok {
		p.Started = true
		p.StartTime = time.Now()
	}

	if c.callbacks.OnPhaseStart != nil {
		go c.callbacks.OnPhaseStart(phase)
	}
}

// RestoreState restores onboarding state from persistence.
// Returns true if there was state to restore.
func (c *Controller) RestoreState(state *SavedState) bool {
	if state == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentPhase = state.CurrentPhase
	c.startTime = state.StartTime
	c.skipped = state.Skipped

	for phase, progress := range state.Progress {
		if existing, ok := c.progress[phase]; ok {
			existing.Started = progress.Started
			existing.Completed = progress.Completed
			existing.StartTime = progress.StartTime
			existing.EndTime = progress.EndTime
			for k, v := range progress.Data {
				existing.Data[k] = v
			}
		}
	}

	return true
}

// SaveState returns the current state for persistence.
func (c *Controller) SaveState() *SavedState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := &SavedState{
		CurrentPhase: c.currentPhase,
		StartTime:    c.startTime,
		Skipped:      c.skipped,
		Progress:     make(map[Phase]*PhaseProgress),
	}

	for phase, progress := range c.progress {
		state.Progress[phase] = &PhaseProgress{
			Started:   progress.Started,
			Completed: progress.Completed,
			StartTime: progress.StartTime,
			EndTime:   progress.EndTime,
			Data:      make(map[string]any),
		}
		for k, v := range progress.Data {
			state.Progress[phase].Data[k] = v
		}
	}

	return state
}

// SavedState contains serializable onboarding state for persistence.
type SavedState struct {
	CurrentPhase Phase
	StartTime    time.Time
	Skipped      bool
	Progress     map[Phase]*PhaseProgress
}

// PhaseInfo returns information about each phase for UI display.
type PhaseInfo struct {
	Phase       Phase
	Title       string
	Description string
	Icon        string // Icon name/identifier
}

// GetPhaseInfo returns UI information for all phases.
func GetPhaseInfo() []PhaseInfo {
	return []PhaseInfo{
		{
			Phase:       PhaseWelcome,
			Title:       "Welcome to MURMUR",
			Description: "A decentralized social network where you own your identity.",
			Icon:        "wave",
		},
		{
			Phase:       PhaseIdentityCreation,
			Title:       "Create Your Identity",
			Description: "Generate your unique cryptographic identity and visual sigil.",
			Icon:        "key",
		},
		{
			Phase:       PhaseModeSelection,
			Title:       "Choose Your Mode",
			Description: "Select how visible you want to be on the network.",
			Icon:        "shield",
		},
		{
			Phase:       PhaseNetworkBootstrap,
			Title:       "Join the Network",
			Description: "Connect to the peer-to-peer mesh.",
			Icon:        "network",
		},
		{
			Phase:       PhaseGuidedExploration,
			Title:       "Explore the Pulse Map",
			Description: "Navigate the living topology of the network.",
			Icon:        "map",
		},
		{
			Phase:       PhaseFirstWave,
			Title:       "Send Your First Wave",
			Description: "Publish your first message to the network.",
			Icon:        "send",
		},
	}
}
