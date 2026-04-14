// Package screens provides Ebitengine-based UI screens for onboarding.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// ScreenState tracks the current screen within a phase.
type ScreenState int

const (
	StateWelcome ScreenState = iota
	StatePhilosophy
	StateKeypairGen
	StateDisplayName
	StateBackupPrompt
	StateBackupMnemonic
	StateBackupFile
	StateBackupComplete
)

// Screen is a stub for non-Ebitengine builds.
type Screen struct {
	controller  *flow.Controller
	state       ScreenState
	keypair     *keys.KeyPair
	displayName string
	backupDone  bool
	callbacks   ScreenCallbacks
}

// ScreenCallbacks provides hooks for screen events.
type ScreenCallbacks struct {
	OnKeypairGenerated func(*keys.KeyPair)
	OnDisplayNameSet   func(string)
	OnBackupComplete   func(method string)
	OnPhaseComplete    func(flow.Phase)
	OnSkipBackup       func()
}

// NewScreen creates a new onboarding screen (stub).
func NewScreen(controller *flow.Controller, callbacks ScreenCallbacks) *Screen {
	return &Screen{
		controller: controller,
		state:      StateWelcome,
		callbacks:  callbacks,
	}
}

// Update is a no-op in stub builds.
func (s *Screen) Update() error {
	return nil
}

// HandleClick is a no-op in stub builds.
func (s *Screen) HandleClick(x, y int) {}

// HandleKeyInput is a no-op in stub builds.
func (s *Screen) HandleKeyInput(char rune) {}

// HandleBackspace is a no-op in stub builds.
func (s *Screen) HandleBackspace() {}

// GetKeypair returns nil in stub builds.
func (s *Screen) GetKeypair() *keys.KeyPair {
	return s.keypair
}

// GetDisplayName returns empty string in stub builds.
func (s *Screen) GetDisplayName() string {
	return s.displayName
}

// IsBackupDone returns false in stub builds.
func (s *Screen) IsBackupDone() bool {
	return s.backupDone
}

// State returns the current screen state.
func (s *Screen) State() ScreenState {
	return s.state
}

// SimulateWelcomeComplete advances past welcome for testing.
func (s *Screen) SimulateWelcomeComplete() {
	s.state = StatePhilosophy
}

// SimulateKeypairGeneration simulates keypair generation for testing.
func (s *Screen) SimulateKeypairGeneration() error {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		return err
	}
	s.keypair = kp
	s.state = StateDisplayName
	if s.callbacks.OnKeypairGenerated != nil {
		s.callbacks.OnKeypairGenerated(kp)
	}
	return nil
}

// SimulateDisplayName sets display name for testing.
func (s *Screen) SimulateDisplayName(name string) {
	s.displayName = name
	s.state = StateBackupPrompt
	if s.callbacks.OnDisplayNameSet != nil {
		s.callbacks.OnDisplayNameSet(name)
	}
}

// SimulateBackupComplete completes backup for testing.
func (s *Screen) SimulateBackupComplete(method string) {
	s.backupDone = true
	s.state = StateBackupComplete
	if s.callbacks.OnBackupComplete != nil {
		s.callbacks.OnBackupComplete(method)
	}
}

// SimulatePhaseComplete completes phases 1-2 for testing.
func (s *Screen) SimulatePhaseComplete() {
	s.controller.CompleteCurrentPhase()
	s.controller.CompleteCurrentPhase()
	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseIdentityCreation)
	}
}

// AnimationDuration returns a duration for testing animation timing.
func AnimationDuration() time.Duration {
	return 2500 * time.Millisecond
}
