package ui

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/opd-ai/murmur/proto"
)

func TestKeyRotationWizardCreation(t *testing.T) {
	// Generate test keys
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	theme := DefaultTheme()
	themePtr := &theme

	wizard := NewKeyRotationWizard(priv, pub, themePtr)

	if wizard == nil {
		t.Fatal("NewKeyRotationWizard returned nil")
	}

	if wizard.IsVisible() {
		t.Error("Wizard should not be visible on creation")
	}
}

func TestKeyRotationWizardShowHide(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	theme := DefaultTheme()
	themePtr := &theme
	wizard := NewKeyRotationWizard(priv, pub, themePtr)

	wizard.Show()
	if !wizard.IsVisible() {
		t.Error("Wizard should be visible after Show()")
	}

	wizard.Hide()
	if wizard.IsVisible() {
		t.Error("Wizard should not be visible after Hide()")
	}
}

func TestKeyRotationWizardCallbacks(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	theme := DefaultTheme()
	themePtr := &theme
	wizard := NewKeyRotationWizard(priv, pub, themePtr)

	completeCalled := false
	wizard.SetOnComplete(func(decl *proto.ContinuityDeclaration) {
		completeCalled = true
	})

	cancelCalled := false
	wizard.SetOnCancel(func() {
		cancelCalled = true
	})

	// Callbacks should be set without error
	if completeCalled || cancelCalled {
		t.Error("Callbacks should not be called immediately")
	}
}

func TestKeyRotationWizardUpdate(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	theme := DefaultTheme()
	themePtr := &theme
	wizard := NewKeyRotationWizard(priv, pub, themePtr)

	// Update should return false when not visible
	if wizard.Update() {
		t.Error("Update should return false when wizard is not visible")
	}

	wizard.Show()

	// Update should return true when visible
	if !wizard.Update() {
		t.Error("Update should return true when wizard is visible")
	}
}

func TestRotationStates(t *testing.T) {
	// Test state transitions
	states := []RotationState{
		RotationStateConfirm,
		RotationStateGeneratingKey,
		RotationStateConfigureGracePeriod,
		RotationStateCreatingDeclaration,
		RotationStatePropagating,
		RotationStateComplete,
		RotationStateError,
	}

	for _, state := range states {
		if state < RotationStateConfirm || state > RotationStateError {
			t.Errorf("Invalid state value: %d", state)
		}
	}
}
