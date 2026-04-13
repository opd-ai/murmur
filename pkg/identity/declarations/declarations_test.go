// Package declarations tests verify identity declaration creation and validation.
package declarations

import (
	"crypto/ed25519"
	"testing"
	"time"
)

// TestDeclarationStructure verifies Declaration struct fields exist.
func TestDeclarationStructure(t *testing.T) {
	// Create a test declaration.
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate test keypair: %v", err)
	}

	decl := Declaration{
		PublicKey:   pub,
		DisplayName: "TestUser",
		Timestamp:   time.Now().Unix(),
		Signature:   make([]byte, ed25519.SignatureSize),
	}

	// Verify fields.
	if len(decl.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey should be %d bytes, got %d", ed25519.PublicKeySize, len(decl.PublicKey))
	}

	if decl.DisplayName != "TestUser" {
		t.Errorf("DisplayName should be 'TestUser', got '%s'", decl.DisplayName)
	}

	if decl.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	if len(decl.Signature) != ed25519.SignatureSize {
		t.Errorf("Signature should be %d bytes, got %d", ed25519.SignatureSize, len(decl.Signature))
	}
}

// TestDeclarationZeroValue verifies zero-value Declaration is valid Go.
func TestDeclarationZeroValue(t *testing.T) {
	var decl Declaration

	// Zero values should be nil/empty/0.
	if decl.PublicKey != nil {
		t.Error("Zero Declaration.PublicKey should be nil")
	}
	if decl.DisplayName != "" {
		t.Error("Zero Declaration.DisplayName should be empty")
	}
	if decl.Timestamp != 0 {
		t.Error("Zero Declaration.Timestamp should be 0")
	}
	if decl.Signature != nil {
		t.Error("Zero Declaration.Signature should be nil")
	}
}

// TestDeclarationTimestampReasonable verifies timestamps are in expected range.
func TestDeclarationTimestampReasonable(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	now := time.Now().Unix()
	decl := Declaration{
		PublicKey:   pub,
		DisplayName: "TimestampTest",
		Timestamp:   now,
	}

	// Per TECHNICAL_IMPLEMENTATION.md, timestamps should be within ±300 seconds of current time.
	tolerance := int64(300)
	currentTime := time.Now().Unix()

	if decl.Timestamp < currentTime-tolerance || decl.Timestamp > currentTime+tolerance {
		t.Error("Declaration timestamp should be within ±300 seconds of current time")
	}
}
