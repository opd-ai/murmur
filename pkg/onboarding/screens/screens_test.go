// Package screens tests for onboarding UI screens.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

func TestNewScreen(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})
	screen := NewScreen(controller, ScreenCallbacks{})

	if screen == nil {
		t.Fatal("NewScreen returned nil")
	}

	if screen.State() != StateWelcome {
		t.Errorf("expected initial state StateWelcome, got %d", screen.State())
	}
}

func TestScreenStateTransitions(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})
	screen := NewScreen(controller, ScreenCallbacks{})

	// Test welcome -> philosophy transition
	screen.SimulateWelcomeComplete()
	if screen.State() != StatePhilosophy {
		t.Errorf("expected StatePhilosophy after welcome, got %d", screen.State())
	}
}

func TestKeypairGeneration(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})

	var generatedKeypair *keys.KeyPair
	callbacks := ScreenCallbacks{
		OnKeypairGenerated: func(kp *keys.KeyPair) {
			generatedKeypair = kp
		},
	}

	screen := NewScreen(controller, callbacks)

	err := screen.SimulateKeypairGeneration()
	if err != nil {
		t.Fatalf("SimulateKeypairGeneration failed: %v", err)
	}

	if generatedKeypair == nil {
		t.Error("OnKeypairGenerated callback not called")
	}

	kp := screen.GetKeypair()
	if kp == nil {
		t.Error("GetKeypair returned nil after generation")
	}

	if screen.State() != StateDisplayName {
		t.Errorf("expected StateDisplayName after keypair gen, got %d", screen.State())
	}
}

func TestDisplayNameEntry(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})

	var setName string
	callbacks := ScreenCallbacks{
		OnDisplayNameSet: func(name string) {
			setName = name
		},
	}

	screen := NewScreen(controller, callbacks)
	screen.SimulateKeypairGeneration()

	testName := "TestUser"
	screen.SimulateDisplayName(testName)

	if setName != testName {
		t.Errorf("expected OnDisplayNameSet with %q, got %q", testName, setName)
	}

	if screen.GetDisplayName() != testName {
		t.Errorf("GetDisplayName returned %q, expected %q", screen.GetDisplayName(), testName)
	}

	if screen.State() != StateBackupPrompt {
		t.Errorf("expected StateBackupPrompt after display name, got %d", screen.State())
	}
}

func TestBackupCompletion(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})

	var backupMethod string
	callbacks := ScreenCallbacks{
		OnBackupComplete: func(method string) {
			backupMethod = method
		},
	}

	screen := NewScreen(controller, callbacks)
	screen.SimulateKeypairGeneration()
	screen.SimulateDisplayName("TestUser")

	screen.SimulateBackupComplete("mnemonic")

	if backupMethod != "mnemonic" {
		t.Errorf("expected backup method 'mnemonic', got %q", backupMethod)
	}

	if !screen.IsBackupDone() {
		t.Error("IsBackupDone should return true after backup")
	}

	if screen.State() != StateBackupComplete {
		t.Errorf("expected StateBackupComplete, got %d", screen.State())
	}
}

func TestPhaseCompletion(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})
	controller.Start()

	var completedPhase flow.Phase
	callbacks := ScreenCallbacks{
		OnPhaseComplete: func(phase flow.Phase) {
			completedPhase = phase
		},
	}

	screen := NewScreen(controller, callbacks)
	screen.SimulateKeypairGeneration()
	screen.SimulateDisplayName("TestUser")
	screen.SimulateBackupComplete("mnemonic")
	screen.SimulatePhaseComplete()

	if completedPhase != flow.PhaseIdentityCreation {
		t.Errorf("expected PhaseIdentityCreation completion, got %v", completedPhase)
	}

	// Controller should have advanced through Welcome and IdentityCreation
	if controller.CurrentPhase() != flow.PhaseModeSelection {
		t.Errorf("expected controller at PhaseModeSelection, got %v", controller.CurrentPhase())
	}
}

func TestFullOnboardingFlow(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})
	controller.Start()

	eventsReceived := make(map[string]bool)
	callbacks := ScreenCallbacks{
		OnKeypairGenerated: func(kp *keys.KeyPair) {
			eventsReceived["keypair"] = true
		},
		OnDisplayNameSet: func(name string) {
			eventsReceived["displayName"] = true
		},
		OnBackupComplete: func(method string) {
			eventsReceived["backup"] = true
		},
		OnPhaseComplete: func(phase flow.Phase) {
			eventsReceived["phaseComplete"] = true
		},
	}

	screen := NewScreen(controller, callbacks)

	// Simulate full flow
	screen.SimulateWelcomeComplete()
	screen.SimulateKeypairGeneration()
	screen.SimulateDisplayName("NewUser")
	screen.SimulateBackupComplete("file")
	screen.SimulatePhaseComplete()

	// Verify all events fired
	expectedEvents := []string{"keypair", "displayName", "backup", "phaseComplete"}
	for _, event := range expectedEvents {
		if !eventsReceived[event] {
			t.Errorf("expected event %q not received", event)
		}
	}

	// Verify final state
	if !screen.IsBackupDone() {
		t.Error("backup should be complete")
	}

	if screen.GetKeypair() == nil {
		t.Error("keypair should be set")
	}

	if screen.GetDisplayName() != "NewUser" {
		t.Errorf("display name should be 'NewUser', got %q", screen.GetDisplayName())
	}
}

func TestAnimationDuration(t *testing.T) {
	duration := AnimationDuration()
	if duration <= 0 {
		t.Error("AnimationDuration should return positive duration")
	}
}
