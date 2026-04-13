// Package screens provides test stubs for Bootstrap screen.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"time"

	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// BootstrapScreenState tracks bootstrap sub-screens.
type BootstrapScreenState int

const (
	BootstrapStateConnecting BootstrapScreenState = iota
	BootstrapStatePulseMapIntro
	BootstrapStateFirstWavePrompt
	BootstrapStateComplete
)

// BootstrapScreen handles Phase 4-5 (stub for testing).
type BootstrapScreen struct {
	controller *flow.Controller
	state      BootstrapScreenState
	startTime  time.Time

	// Peer discovery
	peersFound    int
	targetPeers   int
	discoveryDone bool

	// First Wave
	firstWaveText string
	waveSent      bool
	tutorialStep  int

	callbacks BootstrapScreenCallbacks
}

// BootstrapScreenCallbacks provides hooks for bootstrap events.
type BootstrapScreenCallbacks struct {
	OnPeerDiscoveryStart func()
	OnPeerFound          func(count int)
	OnPulseMapReady      func()
	OnFirstWaveSent      func(text string)
	OnPhaseComplete      func(flow.Phase)
}

// NewBootstrapScreen creates a new bootstrap screen (stub).
func NewBootstrapScreen(controller *flow.Controller, callbacks BootstrapScreenCallbacks) *BootstrapScreen {
	return &BootstrapScreen{
		controller:  controller,
		state:       BootstrapStateConnecting,
		startTime:   time.Now(),
		targetPeers: 6,
		callbacks:   callbacks,
	}
}

// Update advances animations (stub - no-op).
func (s *BootstrapScreen) Update() error {
	return nil
}

// BootstrapState returns the current screen state.
func (s *BootstrapScreen) BootstrapState() BootstrapScreenState {
	return s.state
}

// PeersFound returns the number of discovered peers.
func (s *BootstrapScreen) PeersFound() int {
	return s.peersFound
}

// IsDiscoveryDone returns whether peer discovery is complete.
func (s *BootstrapScreen) IsDiscoveryDone() bool {
	return s.discoveryDone
}

// WasSent returns whether the first Wave was sent.
func (s *BootstrapScreen) WasSent() bool {
	return s.waveSent
}

// --- Simulation Methods for Testing ---

// SimulatePeerFound simulates finding a peer.
func (s *BootstrapScreen) SimulatePeerFound() {
	s.peersFound++
	if s.callbacks.OnPeerFound != nil {
		s.callbacks.OnPeerFound(s.peersFound)
	}
	if s.peersFound >= s.targetPeers {
		s.discoveryDone = true
	}
}

// SimulateDiscoveryComplete simulates completing peer discovery.
func (s *BootstrapScreen) SimulateDiscoveryComplete(peerCount int) {
	s.peersFound = peerCount
	s.discoveryDone = true
}

// SimulateAdvanceToTutorial advances to the Pulse Map tutorial.
func (s *BootstrapScreen) SimulateAdvanceToTutorial() {
	if !s.discoveryDone {
		s.discoveryDone = true
	}
	s.state = BootstrapStatePulseMapIntro
	s.tutorialStep = 0
}

// SimulateTutorialStep advances the tutorial.
func (s *BootstrapScreen) SimulateTutorialStep() {
	if s.tutorialStep < 4 {
		s.tutorialStep++
	}
}

// SimulateCompleteTutorial completes the tutorial.
func (s *BootstrapScreen) SimulateCompleteTutorial() {
	s.tutorialStep = 4
	s.state = BootstrapStateFirstWavePrompt
	if s.callbacks.OnPulseMapReady != nil {
		s.callbacks.OnPulseMapReady()
	}
}

// SimulateSendWave simulates sending the first Wave.
func (s *BootstrapScreen) SimulateSendWave(text string) {
	s.firstWaveText = text
	s.waveSent = true
	if s.callbacks.OnFirstWaveSent != nil {
		s.callbacks.OnFirstWaveSent(text)
	}
	s.state = BootstrapStateComplete
}

// SimulateSkipWave simulates skipping the first Wave.
func (s *BootstrapScreen) SimulateSkipWave() {
	s.state = BootstrapStateComplete
}

// SimulateComplete completes the bootstrap phase.
func (s *BootstrapScreen) SimulateComplete() {
	// Bootstrap screen handles NetworkBootstrap (3) -> GuidedExploration (4) -> FirstWave (5) -> Complete (6)
	s.controller.CompleteCurrentPhase() // NetworkBootstrap -> GuidedExploration
	s.controller.CompleteCurrentPhase() // GuidedExploration -> FirstWave
	s.controller.CompleteCurrentPhase() // FirstWave -> Complete

	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseNetworkBootstrap)
	}
}
