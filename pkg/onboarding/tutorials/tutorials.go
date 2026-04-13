// Package tutorials provides guided exploration and contextual hints.
// Per ONBOARDING.md, tutorials help users learn MURMUR features.
package tutorials

import (
	"sync"
	"time"
)

// HintID is a unique identifier for contextual hints.
type HintID string

// Predefined hint identifiers for key UI elements.
const (
	HintPulseMapPan       HintID = "pulsemap.pan"
	HintPulseMapZoom      HintID = "pulsemap.zoom"
	HintPulseMapNodeClick HintID = "pulsemap.node_click"
	HintPulseMapEdges     HintID = "pulsemap.edges"
	HintWaveCreate        HintID = "wave.create"
	HintWaveReply         HintID = "wave.reply"
	HintWaveAmplify       HintID = "wave.amplify"
	HintIdentitySigil     HintID = "identity.sigil"
	HintIdentityMode      HintID = "identity.mode"
	HintSpecterCreate     HintID = "specter.create"
	HintSpecterSwitch     HintID = "specter.switch"
	HintResonance         HintID = "resonance.view"
)

// Hint contains the content and display rules for a contextual hint.
type Hint struct {
	ID          HintID
	Title       string
	Content     string
	Position    HintPosition
	Trigger     TriggerCondition
	ShowOnce    bool          // Only show once ever
	Cooldown    time.Duration // Minimum time between shows
	Priority    int           // Higher priority shown first
	Dismissible bool          // User can dismiss
}

// HintPosition specifies where the hint should appear.
type HintPosition struct {
	Anchor   AnchorPoint
	OffsetX  float32
	OffsetY  float32
	TargetID string // ID of UI element to anchor to
}

// AnchorPoint defines hint placement relative to target.
type AnchorPoint int

const (
	AnchorTopLeft AnchorPoint = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorCenterLeft
	AnchorCenter
	AnchorCenterRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
)

// TriggerCondition defines when a hint should be shown.
type TriggerCondition struct {
	Type    TriggerType
	Payload any // Type-specific data
}

// TriggerType specifies the kind of trigger.
type TriggerType int

const (
	TriggerOnEnter       TriggerType = iota // Shown when entering area/phase
	TriggerOnFirstAction                    // First time user performs action
	TriggerOnIdle                           // User idle for N seconds
	TriggerOnError                          // After user error
	TriggerManual                           // Triggered programmatically
)

// HintState tracks display state for a hint.
type HintState struct {
	Shown        bool
	DismissedAt  time.Time
	ShowCount    int
	LastShownAt  time.Time
	Acknowledged bool // User explicitly acknowledged
}

// Manager coordinates contextual hints during onboarding.
type Manager struct {
	mu         sync.RWMutex
	hints      map[HintID]*Hint
	states     map[HintID]*HintState
	activeHint *HintID
	enabled    bool
	callbacks  ManagerCallbacks
}

// ManagerCallbacks provides hooks for hint events.
type ManagerCallbacks struct {
	OnShow    func(hint *Hint)
	OnDismiss func(hintID HintID)
}

// NewManager creates a new hint manager.
func NewManager(callbacks ManagerCallbacks) *Manager {
	m := &Manager{
		hints:     make(map[HintID]*Hint),
		states:    make(map[HintID]*HintState),
		enabled:   true,
		callbacks: callbacks,
	}
	m.registerDefaultHints()
	return m
}

// Enable enables hint display.
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

// Disable disables hint display.
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.activeHint = nil
}

// IsEnabled returns whether hints are enabled.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// RegisterHint adds or updates a hint.
func (m *Manager) RegisterHint(hint *Hint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hints[hint.ID] = hint
	if _, exists := m.states[hint.ID]; !exists {
		m.states[hint.ID] = &HintState{}
	}
}

// TriggerHint attempts to show a hint if conditions are met.
func (m *Manager) TriggerHint(id HintID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.canShowHint(id) {
		return false
	}

	m.activateHint(id)
	return true
}

// canShowHint checks if a hint is eligible to be shown.
func (m *Manager) canShowHint(id HintID) bool {
	if !m.enabled {
		return false
	}

	hint, exists := m.hints[id]
	if !exists {
		return false
	}

	state := m.states[id]
	if !m.passesShowOnceConstraint(hint, state) {
		return false
	}
	if !m.passesCooldownConstraint(hint, state) {
		return false
	}
	return true
}

// passesShowOnceConstraint checks the show-once constraint.
func (m *Manager) passesShowOnceConstraint(hint *Hint, state *HintState) bool {
	return !hint.ShowOnce || !state.Shown
}

// passesCooldownConstraint checks the cooldown constraint.
func (m *Manager) passesCooldownConstraint(hint *Hint, state *HintState) bool {
	if hint.Cooldown <= 0 || state.LastShownAt.IsZero() {
		return true
	}
	return time.Since(state.LastShownAt) >= hint.Cooldown
}

// activateHint marks the hint as active and notifies.
func (m *Manager) activateHint(id HintID) {
	state := m.states[id]
	state.Shown = true
	state.ShowCount++
	state.LastShownAt = time.Now()
	m.activeHint = &id

	if m.callbacks.OnShow != nil {
		go m.callbacks.OnShow(m.hints[id])
	}
}

// DismissHint dismisses the currently active hint.
func (m *Manager) DismissHint() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeHint == nil {
		return
	}

	id := *m.activeHint
	if state, exists := m.states[id]; exists {
		state.DismissedAt = time.Now()
	}

	if m.callbacks.OnDismiss != nil {
		go m.callbacks.OnDismiss(id)
	}

	m.activeHint = nil
}

// AcknowledgeHint marks a hint as acknowledged.
func (m *Manager) AcknowledgeHint(id HintID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.states[id]; exists {
		state.Acknowledged = true
	}
}

// ActiveHint returns the currently displayed hint, if any.
func (m *Manager) ActiveHint() *Hint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeHint == nil {
		return nil
	}
	return m.hints[*m.activeHint]
}

// GetHintState returns the state of a specific hint.
func (m *Manager) GetHintState(id HintID) *HintState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.states[id]
}

// ResetHints clears all hint states.
func (m *Manager) ResetHints() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states = make(map[HintID]*HintState)
	for id := range m.hints {
		m.states[id] = &HintState{}
	}
	m.activeHint = nil
}

// GetAllHints returns all registered hints.
func (m *Manager) GetAllHints() []*Hint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hints := make([]*Hint, 0, len(m.hints))
	for _, hint := range m.hints {
		hints = append(hints, hint)
	}
	return hints
}

// registerDefaultHints adds the standard onboarding hints.
func (m *Manager) registerDefaultHints() {
	hints := buildDefaultHints()
	for _, hint := range hints {
		m.hints[hint.ID] = hint
		m.states[hint.ID] = &HintState{}
	}
}

// buildDefaultHints constructs the standard hint collection.
func buildDefaultHints() []*Hint {
	return []*Hint{
		buildPulseMapPanHint(),
		buildPulseMapZoomHint(),
		buildPulseMapNodeClickHint(),
		buildPulseMapEdgesHint(),
		buildWaveCreateHint(),
		buildWaveReplyHint(),
		buildWaveAmplifyHint(),
		buildIdentitySigilHint(),
		buildIdentityModeHint(),
		buildSpecterCreateHint(),
		buildSpecterSwitchHint(),
		buildResonanceHint(),
	}
}

func buildPulseMapPanHint() *Hint {
	return &Hint{
		ID:          HintPulseMapPan,
		Title:       "Navigate the Map",
		Content:     "Click and drag to pan around the Pulse Map.",
		Position:    HintPosition{Anchor: AnchorBottomCenter},
		Trigger:     TriggerCondition{Type: TriggerOnEnter},
		ShowOnce:    true,
		Priority:    100,
		Dismissible: true,
	}
}

func buildPulseMapZoomHint() *Hint {
	return &Hint{
		ID:          HintPulseMapZoom,
		Title:       "Zoom In and Out",
		Content:     "Use the mouse wheel to zoom in for detail or out for overview.",
		Position:    HintPosition{Anchor: AnchorBottomCenter},
		Trigger:     TriggerCondition{Type: TriggerOnFirstAction},
		ShowOnce:    true,
		Priority:    99,
		Dismissible: true,
	}
}

func buildPulseMapNodeClickHint() *Hint {
	return &Hint{
		ID:          HintPulseMapNodeClick,
		Title:       "Select a Node",
		Content:     "Click on any node to view their profile and recent Waves.",
		Position:    HintPosition{Anchor: AnchorCenterRight},
		Trigger:     TriggerCondition{Type: TriggerOnIdle},
		ShowOnce:    true,
		Priority:    98,
		Dismissible: true,
	}
}

func buildPulseMapEdgesHint() *Hint {
	return &Hint{
		ID:          HintPulseMapEdges,
		Title:       "Connection Lines",
		Content:     "Lines between nodes show connections. Brighter lines mean more recent activity.",
		Position:    HintPosition{Anchor: AnchorCenter},
		Trigger:     TriggerCondition{Type: TriggerOnIdle},
		ShowOnce:    true,
		Priority:    97,
		Dismissible: true,
	}
}

func buildWaveCreateHint() *Hint {
	return &Hint{
		ID:          HintWaveCreate,
		Title:       "Create a Wave",
		Content:     "Press W or tap the Wave button to compose your first message.",
		Position:    HintPosition{Anchor: AnchorBottomRight, TargetID: "wave_button"},
		Trigger:     TriggerCondition{Type: TriggerOnEnter},
		ShowOnce:    true,
		Priority:    90,
		Dismissible: true,
	}
}

func buildWaveReplyHint() *Hint {
	return &Hint{
		ID:          HintWaveReply,
		Title:       "Reply to a Wave",
		Content:     "Click the reply icon to respond to someone's Wave.",
		Position:    HintPosition{Anchor: AnchorCenterRight},
		Trigger:     TriggerCondition{Type: TriggerOnFirstAction},
		ShowOnce:    true,
		Priority:    89,
		Dismissible: true,
	}
}

func buildWaveAmplifyHint() *Hint {
	return &Hint{
		ID:          HintWaveAmplify,
		Title:       "Amplify a Wave",
		Content:     "Amplifying shares a Wave with your connections, extending its reach.",
		Position:    HintPosition{Anchor: AnchorCenterRight},
		Trigger:     TriggerCondition{Type: TriggerOnFirstAction},
		ShowOnce:    true,
		Priority:    88,
		Dismissible: true,
	}
}

func buildIdentitySigilHint() *Hint {
	return &Hint{
		ID:          HintIdentitySigil,
		Title:       "Your Sigil",
		Content:     "This unique visual pattern represents your identity on the network.",
		Position:    HintPosition{Anchor: AnchorTopRight, TargetID: "sigil_display"},
		Trigger:     TriggerCondition{Type: TriggerOnEnter},
		ShowOnce:    true,
		Priority:    80,
		Dismissible: true,
	}
}

func buildIdentityModeHint() *Hint {
	return &Hint{
		ID:          HintIdentityMode,
		Title:       "Privacy Modes",
		Content:     "Switch between Open, Hybrid, Guarded, and Fortress modes to control your visibility.",
		Position:    HintPosition{Anchor: AnchorTopRight, TargetID: "mode_selector"},
		Trigger:     TriggerCondition{Type: TriggerOnFirstAction},
		ShowOnce:    true,
		Priority:    79,
		Dismissible: true,
	}
}

func buildSpecterCreateHint() *Hint {
	return &Hint{
		ID:          HintSpecterCreate,
		Title:       "Create a Specter",
		Content:     "Specters are anonymous identities for participating without revealing yourself.",
		Position:    HintPosition{Anchor: AnchorBottomLeft},
		Trigger:     TriggerCondition{Type: TriggerOnEnter},
		ShowOnce:    true,
		Priority:    70,
		Dismissible: true,
	}
}

func buildSpecterSwitchHint() *Hint {
	return &Hint{
		ID:          HintSpecterSwitch,
		Title:       "Switch Identities",
		Content:     "Toggle between your Surface identity and Specter identities anytime.",
		Position:    HintPosition{Anchor: AnchorTopCenter},
		Trigger:     TriggerCondition{Type: TriggerOnFirstAction},
		ShowOnce:    true,
		Priority:    69,
		Dismissible: true,
	}
}

func buildResonanceHint() *Hint {
	return &Hint{
		ID:          HintResonance,
		Title:       "Your Resonance",
		Content:     "Resonance reflects how your Specter's contributions are valued by the community.",
		Position:    HintPosition{Anchor: AnchorCenterLeft},
		Trigger:     TriggerCondition{Type: TriggerOnEnter},
		ShowOnce:    true,
		Priority:    60,
		Dismissible: true,
	}
}

// TutorialStep represents one step in a guided tutorial.
type TutorialStep struct {
	ID          string
	Title       string
	Instruction string
	TargetID    string        // UI element to highlight
	Validation  func() bool   // Returns true when step is complete
	OnComplete  func()        // Called when step completes
	Timeout     time.Duration // Max time for step
	Optional    bool          // Can be skipped
	NextStep    string        // ID of next step
}

// Tutorial is a sequence of guided steps.
type Tutorial struct {
	ID          string
	Name        string
	Description string
	Steps       []*TutorialStep
	currentStep int
	started     bool
	completed   bool
	startTime   time.Time
}

// NewTutorial creates a new tutorial with the given steps.
func NewTutorial(id, name, description string, steps []*TutorialStep) *Tutorial {
	return &Tutorial{
		ID:          id,
		Name:        name,
		Description: description,
		Steps:       steps,
		currentStep: -1,
	}
}

// Start begins the tutorial.
func (t *Tutorial) Start() {
	t.started = true
	t.startTime = time.Now()
	t.currentStep = 0
}

// CurrentStep returns the current step.
func (t *Tutorial) CurrentStep() *TutorialStep {
	if t.currentStep < 0 || t.currentStep >= len(t.Steps) {
		return nil
	}
	return t.Steps[t.currentStep]
}

// Advance moves to the next step if the current is complete.
func (t *Tutorial) Advance() bool {
	if t.completed {
		return false
	}

	current := t.CurrentStep()
	if current != nil && current.OnComplete != nil {
		current.OnComplete()
	}

	t.currentStep++
	if t.currentStep >= len(t.Steps) {
		t.completed = true
		return false
	}
	return true
}

// Skip skips the current step if optional.
func (t *Tutorial) Skip() bool {
	current := t.CurrentStep()
	if current == nil || !current.Optional {
		return false
	}
	return t.Advance()
}

// IsComplete returns true if the tutorial is finished.
func (t *Tutorial) IsComplete() bool {
	return t.completed
}

// Progress returns the completion percentage.
func (t *Tutorial) Progress() float64 {
	if len(t.Steps) == 0 {
		return 100.0
	}
	if t.completed {
		return 100.0
	}
	return float64(t.currentStep) / float64(len(t.Steps)) * 100.0
}

// Duration returns how long the tutorial has been running.
func (t *Tutorial) Duration() time.Duration {
	if t.startTime.IsZero() {
		return 0
	}
	return time.Since(t.startTime)
}
