package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TestGenerateInvitation validates invitation generation.
func TestGenerateInvitation(t *testing.T) {
	// Generate test keys.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create a test peer ID.
	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("failed to decode peer ID: %v", err)
	}

	welcomeMsg := "Welcome to MURMUR!"

	// Generate invitation.
	inv, err := GenerateInvitation(peerID, pub, welcomeMsg)
	if err != nil {
		t.Fatalf("GenerateInvitation failed: %v", err)
	}

	// Validate fields.
	if inv.PeerID != peerID {
		t.Error("peer ID mismatch")
	}
	if string(inv.PublicKey) != string(pub) {
		t.Error("public key mismatch")
	}
	if inv.WelcomeMessage != welcomeMsg {
		t.Error("welcome message mismatch")
	}
}

// TestGenerateInvitationTruncatesLongMessage validates message truncation.
func TestGenerateInvitationTruncatesLongMessage(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")

	// Create a message longer than MaxWelcomeMessageLength.
	longMsg := strings.Repeat("a", MaxWelcomeMessageLength+50)

	inv, err := GenerateInvitation(peerID, pub, longMsg)
	if err != nil {
		t.Fatalf("GenerateInvitation failed: %v", err)
	}

	if len(inv.WelcomeMessage) != MaxWelcomeMessageLength {
		t.Errorf("message not truncated: got %d, want %d", len(inv.WelcomeMessage), MaxWelcomeMessageLength)
	}
}

// TestInvitationEncodeDecode validates round-trip encoding.
func TestInvitationEncodeDecode(t *testing.T) {
	// Generate test invitation.
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	welcomeMsg := "Join the mesh!"

	original, err := GenerateInvitation(peerID, pub, welcomeMsg)
	if err != nil {
		t.Fatalf("GenerateInvitation failed: %v", err)
	}

	// Encode.
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Validate encoding properties.
	if encoded == "" {
		t.Error("encoded string is empty")
	}

	// Per VIRAL_GROWTH_AND_ONBOARDING.md, invitations should be ~100-150 characters.
	if len(encoded) > 200 {
		t.Errorf("encoded invitation too long: %d characters", len(encoded))
	}

	// Decode.
	decoded, err := DecodeInvitation(encoded)
	if err != nil {
		t.Fatalf("DecodeInvitation failed: %v", err)
	}

	// Verify fields match.
	if decoded.PeerID != original.PeerID {
		t.Error("peer ID mismatch after round-trip")
	}
	if string(decoded.PublicKey) != string(original.PublicKey) {
		t.Error("public key mismatch after round-trip")
	}
	if decoded.WelcomeMessage != original.WelcomeMessage {
		t.Error("welcome message mismatch after round-trip")
	}
}

// TestInvitationEncodeURI validates URI encoding with murmur:// prefix.
func TestInvitationEncodeURI(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")

	inv, _ := GenerateInvitation(peerID, pub, "Test")

	// Encode as URI.
	uri, err := inv.EncodeURI()
	if err != nil {
		t.Fatalf("EncodeURI failed: %v", err)
	}

	// Verify prefix.
	if !strings.HasPrefix(uri, InviteURIScheme) {
		t.Errorf("URI missing prefix: %s", uri)
	}

	// Decode from URI.
	decoded, err := DecodeInvitation(uri)
	if err != nil {
		t.Fatalf("DecodeInvitation failed on URI: %v", err)
	}

	// Verify decoded matches original.
	if decoded.PeerID != inv.PeerID {
		t.Error("peer ID mismatch after URI round-trip")
	}
}

// TestDecodeInvitationInvalidData validates error handling.
func TestDecodeInvitationInvalidData(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"empty string", ""},
		{"invalid base64", "not-valid-base64!@#"},
		{"random bytes", "YWJjZGVmZ2hpamtsbW5vcA=="}, // valid base64 but invalid protobuf
		{"truncated", "murmur://invite/YQ=="},        // too short
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeInvitation(tt.encoded)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

// TestInvitationValidate validates the Validate method.
func TestInvitationValidate(t *testing.T) {
	// Valid invitation.
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	validInv, _ := GenerateInvitation(peerID, pub, "Hello")

	if err := validInv.Validate(); err != nil {
		t.Errorf("valid invitation failed validation: %v", err)
	}

	// Invalid: empty peer ID.
	invalidPeerID := &Invitation{
		PeerID:    "",
		PublicKey: pub,
	}
	if err := invalidPeerID.Validate(); err == nil {
		t.Error("expected error for empty peer ID")
	}

	// Invalid: wrong public key size.
	invalidKey := &Invitation{
		PeerID:    peerID,
		PublicKey: []byte{1, 2, 3},
	}
	if err := invalidKey.Validate(); err == nil {
		t.Error("expected error for invalid public key")
	}

	// Invalid: message too long.
	invalidMsg := &Invitation{
		PeerID:         peerID,
		PublicKey:      pub,
		WelcomeMessage: strings.Repeat("a", MaxWelcomeMessageLength+1),
	}
	if err := invalidMsg.Validate(); err == nil {
		t.Error("expected error for message too long")
	}
}

// TestInvitationString validates human-readable representation.
func TestInvitationString(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")

	// With message.
	invWithMsg, _ := GenerateInvitation(peerID, pub, "Join us!")
	str := invWithMsg.String()
	if !strings.Contains(str, "Join us!") {
		t.Errorf("String() missing welcome message: %s", str)
	}

	// Without message.
	invNoMsg, _ := GenerateInvitation(peerID, pub, "")
	str = invNoMsg.String()
	if !strings.Contains(str, peerID.ShortString()) {
		t.Errorf("String() missing peer ID: %s", str)
	}
}

// BenchmarkInvitationEncode benchmarks invitation encoding.
func BenchmarkInvitationEncode(b *testing.B) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	inv, _ := GenerateInvitation(peerID, pub, "Benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inv.Encode()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInvitationDecode benchmarks invitation decoding.
func BenchmarkInvitationDecode(b *testing.B) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	inv, _ := GenerateInvitation(peerID, pub, "Benchmark")
	encoded, _ := inv.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecodeInvitation(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestGenerateQRCode validates QR code generation.
func TestGenerateQRCode(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	inv, _ := GenerateInvitation(peerID, pub, "Scan me!")

	// Generate QR code image.
	img, err := inv.GenerateQRCode(256)
	if err != nil {
		t.Fatalf("GenerateQRCode failed: %v", err)
	}

	if img == nil {
		t.Error("QR code image is nil")
	}

	// Verify image dimensions.
	bounds := img.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 256 {
		t.Errorf("QR code size mismatch: got %dx%d, want 256x256", bounds.Dx(), bounds.Dy())
	}
}

// TestGenerateQRCodePNG validates PNG-encoded QR code generation.
func TestGenerateQRCodePNG(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	inv, _ := GenerateInvitation(peerID, pub, "PNG test")

	// Generate PNG.
	png, err := inv.GenerateQRCodePNG(256)
	if err != nil {
		t.Fatalf("GenerateQRCodePNG failed: %v", err)
	}

	if len(png) == 0 {
		t.Error("PNG data is empty")
	}

	// Verify PNG header.
	if len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Error("invalid PNG header")
	}
}

// BenchmarkGenerateQRCode benchmarks QR code generation.
func BenchmarkGenerateQRCode(b *testing.B) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	peerID, _ := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	inv, _ := GenerateInvitation(peerID, pub, "Benchmark QR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inv.GenerateQRCode(256)
		if err != nil {
			b.Fatal(err)
		}
	}
}
