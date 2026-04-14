package declarations

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

func TestNewProfileUpdate(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	update, err := NewProfileUpdate(kp, 1)
	if err != nil {
		t.Fatalf("NewProfileUpdate() error: %v", err)
	}

	if update.Version != 2 {
		t.Errorf("Version = %d, want 2", update.Version)
	}
}

func TestNewProfileUpdateNilKeyPair(t *testing.T) {
	_, err := NewProfileUpdate(nil, 1)
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestProfileUpdateSetDisplayName(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	if err := update.SetDisplayName("NewName"); err != nil {
		t.Errorf("SetDisplayName() error: %v", err)
	}

	if update.DisplayName != "NewName" {
		t.Errorf("DisplayName = %q, want %q", update.DisplayName, "NewName")
	}
	if !update.HasChanges() {
		t.Error("HasChanges() should be true after SetDisplayName()")
	}
}

func TestProfileUpdateSetDisplayNameTooLong(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	longName := ""
	for i := 0; i < MaxDisplayNameLen+10; i++ {
		longName += "a"
	}

	err := update.SetDisplayName(longName)
	if err != ErrDisplayNameTooLon {
		t.Errorf("Expected ErrDisplayNameTooLon, got %v", err)
	}
}

func TestProfileUpdateSetBio(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	if err := update.SetBio("New bio text"); err != nil {
		t.Errorf("SetBio() error: %v", err)
	}

	if update.Bio != "New bio text" {
		t.Errorf("Bio = %q, want %q", update.Bio, "New bio text")
	}
	if !update.HasChanges() {
		t.Error("HasChanges() should be true after SetBio()")
	}
}

func TestProfileUpdateSetBioTooLong(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	longBio := ""
	for i := 0; i < MaxBioLen+10; i++ {
		longBio += "a"
	}

	err := update.SetBio(longBio)
	if err != ErrBioTooLong {
		t.Errorf("Expected ErrBioTooLong, got %v", err)
	}
}

func TestProfileUpdateSetSigil(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	sigilData := []byte("test sigil png")
	update.SetSigil(sigilData)

	if string(update.SigilPNG) != string(sigilData) {
		t.Error("SigilPNG mismatch")
	}
	if !update.HasChanges() {
		t.Error("HasChanges() should be true after SetSigil()")
	}
}

func TestProfileUpdateSetPrivacyMode(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	update.SetPrivacyMode(modes.Guarded)

	if update.PrivacyMode != modes.Guarded {
		t.Errorf("PrivacyMode = %v, want %v", update.PrivacyMode, modes.Guarded)
	}
	if !update.HasChanges() {
		t.Error("HasChanges() should be true after SetPrivacyMode()")
	}
}

func TestProfileUpdateHasNoChanges(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	if update.HasChanges() {
		t.Error("HasChanges() should be false with no changes")
	}
}

func TestProfileUpdateSignNoChanges(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	err := update.Sign(kp)
	if err != ErrNoChanges {
		t.Errorf("Expected ErrNoChanges, got %v", err)
	}
}

func TestProfileUpdateSignAndVerify(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)

	update.SetDisplayName("NewName")
	if err := update.Sign(kp); err != nil {
		t.Fatalf("Sign() error: %v", err)
	}

	if err := update.Verify(); err != nil {
		t.Errorf("Verify() error: %v", err)
	}
}

func TestProfileUpdateVerifyUnsigned(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("NewName")

	err := update.Verify()
	if err != ErrProfileNotSigned {
		t.Errorf("Expected ErrProfileNotSigned, got %v", err)
	}
}

func TestProfileUpdateValidateAgainst(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	decl, _ := New(kp, "OriginalName")
	decl.Version = 1

	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("NewName")

	if err := update.ValidateAgainst(decl); err != nil {
		t.Errorf("ValidateAgainst() error: %v", err)
	}
}

func TestProfileUpdateValidateAgainstWrongVersion(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	decl, _ := New(kp, "OriginalName")
	decl.Version = 5

	update, _ := NewProfileUpdate(kp, 1) // Version will be 2
	update.SetDisplayName("NewName")

	err := update.ValidateAgainst(decl)
	if err != ErrVersionNotHigher {
		t.Errorf("Expected ErrVersionNotHigher, got %v", err)
	}
}

func TestProfileUpdateValidateAgainstNil(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("NewName")

	// Should not error with nil previous.
	if err := update.ValidateAgainst(nil); err != nil {
		t.Errorf("ValidateAgainst(nil) error: %v", err)
	}
}

func TestProfileUpdateApplyTo(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	decl, _ := New(kp, "OriginalName")
	decl.Bio = "Original bio"
	decl.PrivacyMode = modes.Open

	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("NewName")
	update.SetPrivacyMode(modes.Hybrid)

	result := update.ApplyTo(decl)

	if result.DisplayName != "NewName" {
		t.Errorf("DisplayName = %q, want %q", result.DisplayName, "NewName")
	}
	if result.Bio != "Original bio" {
		t.Errorf("Bio should be unchanged: %q", result.Bio)
	}
	if result.PrivacyMode != modes.Hybrid {
		t.Errorf("PrivacyMode = %v, want %v", result.PrivacyMode, modes.Hybrid)
	}
}

func TestProfileUpdateMarshalUnmarshal(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()

	update, _ := NewProfileUpdate(kp, 5)
	update.SetDisplayName("TestName")
	update.SetBio("Test bio")
	update.SetPrivacyMode(modes.Guarded)
	update.Sign(kp)

	data, err := update.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	decoded, err := UnmarshalProfileUpdate(data)
	if err != nil {
		t.Fatalf("UnmarshalProfileUpdate() error: %v", err)
	}

	if decoded.DisplayName != "TestName" {
		t.Errorf("DisplayName = %q, want %q", decoded.DisplayName, "TestName")
	}
	if decoded.Bio != "Test bio" {
		t.Errorf("Bio = %q, want %q", decoded.Bio, "Test bio")
	}
	if decoded.PrivacyMode != modes.Guarded {
		t.Errorf("PrivacyMode = %v, want %v", decoded.PrivacyMode, modes.Guarded)
	}
	if decoded.Version != 6 {
		t.Errorf("Version = %d, want 6", decoded.Version)
	}
	if err := decoded.Verify(); err != nil {
		t.Errorf("Decoded Verify() error: %v", err)
	}
}

func TestProfileUpdateTimestampValidation(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	update, _ := NewProfileUpdate(kp, 1)
	update.SetDisplayName("Test")

	// Fresh timestamp should be valid.
	if err := update.ValidateTimestamp(); err != nil {
		t.Errorf("Fresh timestamp validation failed: %v", err)
	}

	// Old timestamp should fail.
	update.UpdatedAt = 0
	if err := update.ValidateTimestamp(); err != ErrTimestampTooOld {
		t.Errorf("Expected ErrTimestampTooOld, got %v", err)
	}
}
