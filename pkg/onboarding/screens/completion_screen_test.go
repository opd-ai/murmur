// Package screens provides tests for Completion screen.
//
//go:build noebiten
// +build noebiten

package screens

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

func TestCompletionScreenInitialState(t *testing.T) {
	callbacks := CompletionScreenCallbacks{}
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Hybrid, 5, callbacks)

	if screen.CompletionState() != CompletionStateSummary {
		t.Errorf("Expected initial state CompletionStateSummary, got %d", screen.CompletionState())
	}

	if screen.IsInviteGenerated() {
		t.Error("Expected no invite generated initially")
	}
}

func TestCompletionScreenOpenMode(t *testing.T) {
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Open, 5, CompletionScreenCallbacks{})

	// Verify mode is stored correctly
	if screen.selectedMode != modes.Open {
		t.Errorf("Expected Open mode, got %s", screen.selectedMode)
	}
}

func TestCompletionScreenHybridMode(t *testing.T) {
	screen := NewCompletionScreen("TestUser", nil, nil, "Shadow Specter", nil, modes.Hybrid, 5, CompletionScreenCallbacks{})

	if screen.selectedMode != modes.Hybrid {
		t.Errorf("Expected Hybrid mode, got %s", screen.selectedMode)
	}
	if screen.specterName != "Shadow Specter" {
		t.Errorf("Expected 'Shadow Specter', got '%s'", screen.specterName)
	}
}

func TestCompletionScreenNavigateToInvite(t *testing.T) {
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Open, 5, CompletionScreenCallbacks{})

	screen.SimulateGoToInvite()

	if screen.CompletionState() != CompletionStateInvite {
		t.Errorf("Expected state CompletionStateInvite, got %d", screen.CompletionState())
	}
}

func TestCompletionScreenGenerateInvite(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	var generatedCode string
	callbacks := CompletionScreenCallbacks{
		OnInviteGenerated: func(code string) { generatedCode = code },
	}
	screen := NewCompletionScreen("TestUser", nil, kp, "", nil, modes.Open, 5, callbacks)

	screen.SimulateGoToInvite()
	screen.SimulateGenerateInvite()

	if !screen.IsInviteGenerated() {
		t.Error("Expected invite to be generated")
	}

	if screen.InviteCode() == "" {
		t.Error("Expected invite code to be non-empty")
	}

	if generatedCode == "" {
		t.Error("Expected callback to receive invite code")
	}

	// Verify invite code format
	code := screen.InviteCode()
	if len(code) < 10 || code[:7] != "MURMUR-" {
		t.Errorf("Expected invite code format 'MURMUR-XXX-XXX', got '%s'", code)
	}
}

func TestCompletionScreenGenerateInviteNoKeypair(t *testing.T) {
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Open, 5, CompletionScreenCallbacks{})

	screen.SimulateGenerateInvite()

	// Should generate a fallback code
	if screen.InviteCode() != "MURMUR-XXXX-YYYY" {
		t.Errorf("Expected fallback code 'MURMUR-XXXX-YYYY', got '%s'", screen.InviteCode())
	}
}

func TestCompletionScreenContinueToComplete(t *testing.T) {
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Open, 5, CompletionScreenCallbacks{})

	screen.SimulateGoToInvite()
	screen.SimulateContinueToComplete()

	if screen.CompletionState() != CompletionStateDone {
		t.Errorf("Expected state CompletionStateDone, got %d", screen.CompletionState())
	}
}

func TestCompletionScreenFinish(t *testing.T) {
	var finished bool
	callbacks := CompletionScreenCallbacks{
		OnOnboardingFinish: func() { finished = true },
	}
	screen := NewCompletionScreen("TestUser", nil, nil, "", nil, modes.Open, 5, callbacks)

	screen.SimulateFinish()

	if !finished {
		t.Error("Expected OnOnboardingFinish callback to be called")
	}
}

func TestFullCompletionFlow(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	var inviteCode string
	var finished bool
	callbacks := CompletionScreenCallbacks{
		OnInviteGenerated:  func(code string) { inviteCode = code },
		OnOnboardingFinish: func() { finished = true },
	}
	screen := NewCompletionScreen("TestUser", nil, kp, "Shadow Wraith", nil, modes.Hybrid, 8, callbacks)

	// Start at summary
	if screen.CompletionState() != CompletionStateSummary {
		t.Fatalf("Expected initial state CompletionStateSummary")
	}

	// Go to invite
	screen.SimulateGoToInvite()
	if screen.CompletionState() != CompletionStateInvite {
		t.Fatalf("Expected state CompletionStateInvite")
	}

	// Generate invite
	screen.SimulateGenerateInvite()
	if inviteCode == "" {
		t.Error("Expected invite code to be generated")
	}

	// Continue to done
	screen.SimulateContinueToComplete()
	if screen.CompletionState() != CompletionStateDone {
		t.Fatalf("Expected state CompletionStateDone")
	}

	// Finish
	screen.SimulateFinish()
	if !finished {
		t.Error("Expected onboarding to finish")
	}
}

func TestInviteCodeDeterminism(t *testing.T) {
	pubKey := []byte{0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56}

	code1 := generateInviteCode(pubKey)
	code2 := generateInviteCode(pubKey)

	if code1 != code2 {
		t.Errorf("Expected deterministic codes, got '%s' and '%s'", code1, code2)
	}

	// Verify format
	expected := "MURMUR-ABCDEF-123456"
	if code1 != expected {
		t.Errorf("Expected '%s', got '%s'", expected, code1)
	}
}

func TestInviteCodeShortKey(t *testing.T) {
	shortKey := []byte{0xAB, 0xCD}
	code := generateInviteCode(shortKey)

	if code != "MURMUR-XXXX-YYYY" {
		t.Errorf("Expected fallback code for short key, got '%s'", code)
	}
}
