// Package screens provides tests for Mode Selection screen.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

func TestModeScreenInitialState(t *testing.T) {
	controller := createTestController(t)
	// Move to Mode Selection phase
	controller.CompleteCurrentPhase() // Welcome -> Identity
	controller.CompleteCurrentPhase() // Identity -> Mode

	callbacks := ModeScreenCallbacks{}
	screen := NewModeScreen(controller, nil, nil, "TestUser", callbacks)

	if screen.ModeState() != ModeStateIntro {
		t.Errorf("Expected initial state ModeStateIntro, got %d", screen.ModeState())
	}

	// Default should be Hybrid
	if screen.GetSelectedMode() != modes.Hybrid {
		t.Errorf("Expected default mode Hybrid, got %s", screen.GetSelectedMode())
	}
}

func TestModeScreenIntroToCards(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	screen := NewModeScreen(controller, nil, nil, "TestUser", ModeScreenCallbacks{})

	screen.SimulateIntroComplete()

	if screen.ModeState() != ModeStateCards {
		t.Errorf("Expected state ModeStateCards, got %d", screen.ModeState())
	}
}

func TestModeScreenOpenModeSelection(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var selectedMode modes.Mode
	callbacks := ModeScreenCallbacks{
		OnModeSelected: func(m modes.Mode) { selectedMode = m },
	}
	screen := NewModeScreen(controller, nil, nil, "TestUser", callbacks)

	screen.SimulateIntroComplete()
	screen.SimulateSelectMode(modes.Open)

	if selectedMode != modes.Open {
		t.Errorf("Expected callback with Open, got %s", selectedMode)
	}

	// Open mode skips Specter generation
	if screen.ModeState() != ModeStateConfirmation {
		t.Errorf("Expected state ModeStateConfirmation for Open mode, got %d", screen.ModeState())
	}
}

func TestModeScreenHybridModeWithSpecterGen(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var specterKP *keys.AnonymousKeyPair
	var specterName string
	callbacks := ModeScreenCallbacks{
		OnSpecterGenerated: func(kp *keys.AnonymousKeyPair, name string) {
			specterKP = kp
			specterName = name
		},
	}
	screen := NewModeScreen(controller, nil, nil, "TestUser", callbacks)

	screen.SimulateIntroComplete()
	screen.SimulateSelectMode(modes.Hybrid)

	// Hybrid mode goes to Specter generation
	if screen.ModeState() != ModeStateSpecterGen {
		t.Errorf("Expected state ModeStateSpecterGen for Hybrid mode, got %d", screen.ModeState())
	}

	err := screen.SimulateSpecterGeneration()
	if err != nil {
		t.Fatalf("Specter generation failed: %v", err)
	}

	if specterKP == nil {
		t.Error("Expected Specter keypair to be generated")
	}
	if specterName == "" {
		t.Error("Expected Specter name to be generated")
	}
	if screen.GetSpecterKeypair() == nil {
		t.Error("Expected GetSpecterKeypair to return the keypair")
	}
	if screen.GetSpecterName() == "" {
		t.Error("Expected GetSpecterName to return the name")
	}

	if screen.ModeState() != ModeStateConfirmation {
		t.Errorf("Expected state ModeStateConfirmation after Specter gen, got %d", screen.ModeState())
	}
}

func TestModeScreenFortressModeWithSpecterGen(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	screen := NewModeScreen(controller, nil, nil, "TestUser", ModeScreenCallbacks{})

	screen.SimulateIntroComplete()
	screen.SimulateSelectMode(modes.Fortress)

	// Fortress mode also goes to Specter generation
	if screen.ModeState() != ModeStateSpecterGen {
		t.Errorf("Expected state ModeStateSpecterGen for Fortress mode, got %d", screen.ModeState())
	}

	err := screen.SimulateSpecterGeneration()
	if err != nil {
		t.Fatalf("Specter generation failed: %v", err)
	}

	if screen.ModeState() != ModeStateConfirmation {
		t.Errorf("Expected state ModeStateConfirmation after Specter gen, got %d", screen.ModeState())
	}
}

func TestModeScreenPhaseCompletion(t *testing.T) {
	controller := createTestController(t)
	controller.CompleteCurrentPhase()
	controller.CompleteCurrentPhase()

	var completedPhase flow.Phase
	callbacks := ModeScreenCallbacks{
		OnPhaseComplete: func(p flow.Phase) { completedPhase = p },
	}
	screen := NewModeScreen(controller, nil, nil, "TestUser", callbacks)

	// Complete full flow for Open mode
	screen.SimulateIntroComplete()
	screen.SimulateSelectMode(modes.Open)
	screen.SimulateConfirmation()

	if completedPhase != flow.PhaseModeSelection {
		t.Errorf("Expected callback with PhaseModeSelection, got %d", completedPhase)
	}

	// Controller should have advanced to NetworkBootstrap
	if controller.CurrentPhase() != flow.PhaseNetworkBootstrap {
		t.Errorf("Expected controller phase to be NetworkBootstrap, got %d", controller.CurrentPhase())
	}
}

func TestSpecterNameGeneration(t *testing.T) {
	// Test that Specter names are deterministic
	pubKey := []byte{0x00, 0x00, 0x01, 0x02}
	name1 := generateSpecterName(pubKey)
	name2 := generateSpecterName(pubKey)

	if name1 != name2 {
		t.Errorf("Expected deterministic names, got %s and %s", name1, name2)
	}

	// Test different keys produce different names
	pubKey2 := []byte{0x05, 0x10, 0x01, 0x02}
	name3 := generateSpecterName(pubKey2)

	if name1 == name3 {
		t.Error("Expected different keys to produce different names")
	}
}

func TestSpecterNameWithShortKey(t *testing.T) {
	shortKey := []byte{0x00}
	name := generateSpecterName(shortKey)

	if name != "Unknown Specter" {
		t.Errorf("Expected 'Unknown Specter' for short key, got %s", name)
	}
}

func createTestController(t *testing.T) *flow.Controller {
	t.Helper()
	controller := flow.NewController(flow.Callbacks{})
	return controller
}
