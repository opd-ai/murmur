package ui

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/recovery"
	"golang.org/x/crypto/curve25519"
)

func TestRecoveryEnrollmentPanelCreation(t *testing.T) {
	// Generate test keys
	masterPub, masterPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	x25519Priv := make([]byte, 32)
	if _, err := rand.Read(x25519Priv); err != nil {
		t.Fatalf("Failed to generate X25519 key: %v", err)
	}

	// Create test contacts
	contacts := make([]RecoveryContact, 5)
	for i := 0; i < 5; i++ {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate contact key: %v", err)
		}
		x25519Key := make([]byte, 32)
		if _, err := rand.Read(x25519Key); err != nil {
			t.Fatalf("Failed to generate X25519 key: %v", err)
		}
		contacts[i] = RecoveryContact{
			PublicKey: pub,
			X25519Key: x25519Key,
			Label:     "Test Contact",
		}
	}

	theme := DefaultTheme()
	themePtr := &theme

	panel := NewRecoveryEnrollmentPanel(contacts, masterPriv, masterPub, x25519Priv, themePtr)

	if panel == nil {
		t.Fatal("NewRecoveryEnrollmentPanel returned nil")
	}

	if panel.IsVisible() {
		t.Error("Panel should not be visible on creation")
	}
}

func TestRecoveryEnrollmentPanelShowHide(t *testing.T) {
	masterPub, masterPriv, _ := ed25519.GenerateKey(rand.Reader)
	x25519Priv := make([]byte, 32)
	rand.Read(x25519Priv)

	contacts := []RecoveryContact{
		{PublicKey: make([]byte, 32), X25519Key: make([]byte, 32), Label: "Contact 1"},
	}

	theme := DefaultTheme()
	themePtr := &theme
	panel := NewRecoveryEnrollmentPanel(contacts, masterPriv, masterPub, x25519Priv, themePtr)

	panel.Show()
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestRecoveryEnrollmentPanelCallbacks(t *testing.T) {
	masterPub, masterPriv, _ := ed25519.GenerateKey(rand.Reader)
	x25519Priv := make([]byte, 32)
	rand.Read(x25519Priv)

	contacts := []RecoveryContact{
		{PublicKey: make([]byte, 32), X25519Key: make([]byte, 32), Label: "Contact 1"},
	}

	theme := DefaultTheme()
	themePtr := &theme
	panel := NewRecoveryEnrollmentPanel(contacts, masterPriv, masterPub, x25519Priv, themePtr)

	completeCalled := false
	panel.SetOnComplete(func(results []recovery.EnrollmentResult) {
		completeCalled = true
	})

	cancelCalled := false
	panel.SetOnCancel(func() {
		cancelCalled = true
	})

	// Callbacks should be set without error
	if completeCalled || cancelCalled {
		t.Error("Callbacks should not be called immediately")
	}
}

func TestRecoveryEnrollmentPanelUpdate(t *testing.T) {
	masterPub, masterPriv, _ := ed25519.GenerateKey(rand.Reader)
	x25519Priv := make([]byte, 32)
	rand.Read(x25519Priv)

	contacts := []RecoveryContact{
		{PublicKey: make([]byte, 32), X25519Key: make([]byte, 32), Label: "Contact 1"},
	}

	theme := DefaultTheme()
	themePtr := &theme
	panel := NewRecoveryEnrollmentPanel(contacts, masterPriv, masterPub, x25519Priv, themePtr)

	// Update should return false when not visible
	if panel.Update() {
		t.Error("Update should return false when panel is not visible")
	}

	panel.Show()

	// Update should return true when visible
	if !panel.Update() {
		t.Error("Update should return true when panel is visible")
	}
}

func TestRecoveryEnrollmentStates(t *testing.T) {
	// Test state transitions
	states := []EnrollmentState{
		EnrollmentStateSelectContacts,
		EnrollmentStateConfigureThreshold,
		EnrollmentStateDistributing,
		EnrollmentStateComplete,
		EnrollmentStateError,
	}

	for _, state := range states {
		if state < EnrollmentStateSelectContacts || state > EnrollmentStateError {
			t.Errorf("Invalid state value: %d", state)
		}
	}
}

func TestRecoveryContactStruct(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	x25519Key := make([]byte, 32)
	rand.Read(x25519Key)

	contact := RecoveryContact{
		PublicKey: pub,
		X25519Key: x25519Key,
		Label:     "Test",
		Selected:  true,
	}

	if len(contact.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Invalid public key size: got %d, want %d", len(contact.PublicKey), ed25519.PublicKeySize)
	}

	if len(contact.X25519Key) != curve25519.ScalarSize {
		t.Errorf("Invalid X25519 key size: got %d, want %d", len(contact.X25519Key), curve25519.ScalarSize)
	}

	if contact.Label != "Test" {
		t.Errorf("Invalid label: got %s, want Test", contact.Label)
	}

	if !contact.Selected {
		t.Error("Contact should be selected")
	}
}
