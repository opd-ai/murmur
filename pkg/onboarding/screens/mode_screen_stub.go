// Package screens provides test stubs for Mode Selection screen.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// ModeScreenState tracks mode selection sub-screens.
type ModeScreenState int

const (
	ModeStateIntro ModeScreenState = iota
	ModeStateCards
	ModeStateSpecterGen
	ModeStateConfirmation
)

// ModeScreen handles the mode selection phase (stub for testing).
type ModeScreen struct {
	controller *flow.Controller
	state      ModeScreenState
	startTime  time.Time

	// User's surface identity
	surfaceKeypair *keys.KeyPair
	displayName    string

	// Specter generation
	specterKeypair *keys.AnonymousKeyPair
	specterName    string

	// Mode selection
	selectedMode modes.Mode
	callbacks    ModeScreenCallbacks
}

// ModeScreenCallbacks provides hooks for mode selection events.
type ModeScreenCallbacks struct {
	OnModeSelected     func(modes.Mode)
	OnSpecterGenerated func(*keys.AnonymousKeyPair, string)
	OnPhaseComplete    func(flow.Phase)
}

// NewModeScreen creates a new mode selection screen (stub).
func NewModeScreen(
	controller *flow.Controller,
	surfaceKP *keys.KeyPair,
	surfaceSigil interface{}, // *ebiten.Image in real build
	displayName string,
	callbacks ModeScreenCallbacks,
) *ModeScreen {
	return &ModeScreen{
		controller:     controller,
		state:          ModeStateIntro,
		startTime:      time.Now(),
		surfaceKeypair: surfaceKP,
		displayName:    displayName,
		selectedMode:   modes.Hybrid,
		callbacks:      callbacks,
	}
}

// Update advances animations (stub - no-op).
func (s *ModeScreen) Update() error {
	return nil
}

// GetSelectedMode returns the selected mode.
func (s *ModeScreen) GetSelectedMode() modes.Mode {
	return s.selectedMode
}

// GetSpecterKeypair returns the generated Specter keypair.
func (s *ModeScreen) GetSpecterKeypair() *keys.AnonymousKeyPair {
	return s.specterKeypair
}

// GetSpecterName returns the generated Specter name.
func (s *ModeScreen) GetSpecterName() string {
	return s.specterName
}

// ModeState returns the current screen state.
func (s *ModeScreen) ModeState() ModeScreenState {
	return s.state
}

// --- Simulation Methods for Testing ---

// SimulateIntroComplete simulates completing the intro animation.
func (s *ModeScreen) SimulateIntroComplete() {
	s.state = ModeStateCards
}

// SimulateSelectMode simulates selecting a mode.
func (s *ModeScreen) SimulateSelectMode(mode modes.Mode) {
	s.selectedMode = mode
	if s.callbacks.OnModeSelected != nil {
		s.callbacks.OnModeSelected(mode)
	}

	if mode == modes.Open {
		s.state = ModeStateConfirmation
	} else {
		s.state = ModeStateSpecterGen
	}
}

// SimulateSpecterGeneration simulates Specter identity creation.
func (s *ModeScreen) SimulateSpecterGeneration() error {
	kp, err := keys.GenerateAnonymousKeyPair()
	if err != nil {
		return err
	}

	s.specterKeypair = kp
	s.specterName = GenerateSpecterName(kp.PublicKey[:])

	if s.callbacks.OnSpecterGenerated != nil {
		s.callbacks.OnSpecterGenerated(kp, s.specterName)
	}

	s.state = ModeStateConfirmation
	return nil
}

// SimulateConfirmation simulates completing the mode selection phase.
func (s *ModeScreen) SimulateConfirmation() {
	s.controller.CompleteCurrentPhase()
	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseModeSelection)
	}
}
