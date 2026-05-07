// Package screens provides tests for Bootstrap screen.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

type fakePeerSource struct {
	handler func(peerID string)
}

func (f *fakePeerSource) SetOnPeerConnected(handler func(peerID string)) {
	f.handler = handler
}

func (f *fakePeerSource) Emit(peerID string) {
	if f.handler != nil {
		f.handler(peerID)
	}
}

func TestBootstrapScreenInitialState(t *testing.T) {
	controller := createTestController(t)
	// Move to NetworkBootstrap phase
	controller.CompleteCurrentPhase() // Welcome -> Identity
	controller.CompleteCurrentPhase() // Identity -> Mode
	controller.CompleteCurrentPhase() // Mode -> NetworkBootstrap

	callbacks := BootstrapScreenCallbacks{}
	screen := NewBootstrapScreen(controller, callbacks)

	if screen.BootstrapState() != BootstrapStateConnecting {
		t.Errorf("Expected initial state BootstrapStateConnecting, got %d", screen.BootstrapState())
	}

	if screen.PeersFound() != 0 {
		t.Errorf("Expected 0 peers found initially, got %d", screen.PeersFound())
	}

	if screen.IsDiscoveryDone() {
		t.Error("Expected discovery not done initially")
	}
}

func TestBootstrapPeerDiscovery(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var peerCounts []int
	callbacks := BootstrapScreenCallbacks{
		OnPeerFound: func(count int) { peerCounts = append(peerCounts, count) },
	}
	screen := NewBootstrapScreen(controller, callbacks)

	// Simulate finding peers
	for i := 0; i < 6; i++ {
		screen.SimulatePeerFound()
	}

	if screen.PeersFound() != 6 {
		t.Errorf("Expected 6 peers found, got %d", screen.PeersFound())
	}

	if len(peerCounts) != 6 {
		t.Errorf("Expected 6 callbacks, got %d", len(peerCounts))
	}

	if !screen.IsDiscoveryDone() {
		t.Error("Expected discovery to be done after finding target peers")
	}
}

func TestBootstrapPeerConnectedSourceForwarding(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	source := &fakePeerSource{}
	screen := NewBootstrapScreenWithPeerSource(controller, BootstrapScreenCallbacks{}, source)

	for i := 0; i < 6; i++ {
		source.Emit("peer")
	}

	// Forwarded peer notifications are drained during Update().
	if err := screen.Update(); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	if got := screen.PeersFound(); got != 6 {
		t.Fatalf("expected 6 peers from forwarded events, got %d", got)
	}
	if !screen.IsDiscoveryDone() {
		t.Fatal("expected discoveryDone=true after reaching target peers via forwarded events")
	}
}

func TestBootstrapAdvanceToTutorial(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	screen := NewBootstrapScreen(controller, BootstrapScreenCallbacks{})

	screen.SimulateDiscoveryComplete(8)
	screen.SimulateAdvanceToTutorial()

	if screen.BootstrapState() != BootstrapStatePulseMapIntro {
		t.Errorf("Expected state BootstrapStatePulseMapIntro, got %d", screen.BootstrapState())
	}
}

func TestBootstrapTutorialCompletion(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var pulseMapReady bool
	callbacks := BootstrapScreenCallbacks{
		OnPulseMapReady: func() { pulseMapReady = true },
	}
	screen := NewBootstrapScreen(controller, callbacks)

	screen.SimulateDiscoveryComplete(6)
	screen.SimulateAdvanceToTutorial()
	screen.SimulateCompleteTutorial()

	if !pulseMapReady {
		t.Error("Expected OnPulseMapReady callback to be called")
	}

	if screen.BootstrapState() != BootstrapStateFirstWavePrompt {
		t.Errorf("Expected state BootstrapStateFirstWavePrompt, got %d", screen.BootstrapState())
	}
}

func TestBootstrapFirstWave(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var sentWave string
	callbacks := BootstrapScreenCallbacks{
		OnFirstWaveSent: func(text string) { sentWave = text },
	}
	screen := NewBootstrapScreen(controller, callbacks)

	screen.SimulateDiscoveryComplete(6)
	screen.SimulateAdvanceToTutorial()
	screen.SimulateCompleteTutorial()
	screen.SimulateSendWave("Hello, network!")

	if sentWave != "Hello, network!" {
		t.Errorf("Expected 'Hello, network!', got '%s'", sentWave)
	}

	if !screen.WasSent() {
		t.Error("Expected WasSent to return true")
	}

	if screen.BootstrapState() != BootstrapStateComplete {
		t.Errorf("Expected state BootstrapStateComplete, got %d", screen.BootstrapState())
	}
}

func TestBootstrapSkipWave(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	screen := NewBootstrapScreen(controller, BootstrapScreenCallbacks{})

	screen.SimulateDiscoveryComplete(6)
	screen.SimulateAdvanceToTutorial()
	screen.SimulateCompleteTutorial()
	screen.SimulateSkipWave()

	if screen.WasSent() {
		t.Error("Expected WasSent to return false after skip")
	}

	if screen.BootstrapState() != BootstrapStateComplete {
		t.Errorf("Expected state BootstrapStateComplete, got %d", screen.BootstrapState())
	}
}

func TestBootstrapPhaseCompletion(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var completedPhase flow.Phase
	callbacks := BootstrapScreenCallbacks{
		OnPhaseComplete: func(p flow.Phase) { completedPhase = p },
	}
	screen := NewBootstrapScreen(controller, callbacks)

	screen.SimulateDiscoveryComplete(6)
	screen.SimulateAdvanceToTutorial()
	screen.SimulateCompleteTutorial()
	screen.SimulateSendWave("Test wave")
	screen.SimulateComplete()

	if completedPhase != flow.PhaseNetworkBootstrap {
		t.Errorf("Expected callback with PhaseNetworkBootstrap, got %d", completedPhase)
	}
}

func TestFullBootstrapFlow(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase() // Welcome -> Identity
	controller.CompleteCurrentPhase() // Identity -> Mode
	controller.CompleteCurrentPhase() // Mode -> NetworkBootstrap

	screen := NewBootstrapScreen(controller, BootstrapScreenCallbacks{})

	// Peer discovery
	screen.SimulateDiscoveryComplete(8)
	if screen.PeersFound() != 8 {
		t.Errorf("Expected 8 peers, got %d", screen.PeersFound())
	}

	// Tutorial
	screen.SimulateAdvanceToTutorial()
	for i := 0; i < 5; i++ {
		screen.SimulateTutorialStep()
	}
	screen.SimulateCompleteTutorial()

	// First Wave
	screen.SimulateSendWave("Testing MURMUR!")

	// Complete
	screen.SimulateComplete()

	// PhaseFirstWave is 5, after completing we should be at PhaseComplete (6)
	if controller.CurrentPhase() != flow.PhaseComplete {
		t.Errorf("Expected controller to be at PhaseComplete, got %d", controller.CurrentPhase())
	}
}
