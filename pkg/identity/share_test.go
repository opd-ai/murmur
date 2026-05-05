package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestShare_Text(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := GenerateInvitation(peerID, pub, "Welcome to MURMUR!")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Share as text.
	opts := DefaultShareOptions()
	opts.Method = ShareText

	result, err := inv.Share(opts)
	if err != nil {
		t.Fatalf("sharing text: %v", err)
	}

	// Verify result is a valid URI.
	if !strings.HasPrefix(result, InviteURIScheme) {
		t.Errorf("expected URI scheme %q, got %q", InviteURIScheme, result[:len(InviteURIScheme)])
	}

	// Verify we can decode the shared URI.
	decoded, err := DecodeInvitation(result)
	if err != nil {
		t.Fatalf("decoding shared URI: %v", err)
	}

	if decoded.PeerID != inv.PeerID {
		t.Errorf("peer ID mismatch: got %v, want %v", decoded.PeerID, inv.PeerID)
	}
	if decoded.WelcomeMessage != inv.WelcomeMessage {
		t.Errorf("welcome message mismatch: got %q, want %q", decoded.WelcomeMessage, inv.WelcomeMessage)
	}
}

func TestShare_Email(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := GenerateInvitation(peerID, pub, "Join me!")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Share as email.
	opts := DefaultShareOptions()
	opts.Method = ShareEmail
	opts.Subject = "Test Invite"
	opts.Body = "Please join"

	result, err := inv.Share(opts)
	if err != nil {
		t.Fatalf("sharing email: %v", err)
	}

	// Verify result is a mailto: URL.
	if !strings.HasPrefix(result, "mailto:") {
		t.Errorf("expected mailto: URL, got %q", result)
	}

	// Verify subject and body are present (URL-encoded).
	if !strings.Contains(result, "subject=") {
		t.Error("mailto: URL missing subject parameter")
	}
	if !strings.Contains(result, "body=") {
		t.Error("mailto: URL missing body parameter")
	}

	// Verify invitation URI is included in body (may be URL-encoded).
	if !strings.Contains(result, "murmur") {
		t.Error("mailto: body missing invitation URI")
	}
}

func TestShare_QR(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := GenerateInvitation(peerID, pub, "Scan this!")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Create temporary directory for test.
	tempDir := t.TempDir()

	// Share as QR code.
	opts := DefaultShareOptions()
	opts.Method = ShareQR
	opts.QRSize = 256
	opts.TempDirPath = tempDir

	result, err := inv.Share(opts)
	if err != nil {
		t.Fatalf("sharing QR: %v", err)
	}

	// Verify result is a file path.
	if !strings.HasSuffix(result, ".png") {
		t.Errorf("expected .png file path, got %q", result)
	}

	// Verify file exists.
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("QR code file does not exist: %s", result)
	}

	// Verify file is in correct directory.
	dir := filepath.Dir(result)
	if dir != tempDir {
		t.Errorf("QR code file in wrong directory: got %q, want %q", dir, tempDir)
	}

	// Verify file is not empty.
	info, err := os.Stat(result)
	if err != nil {
		t.Fatalf("stat QR file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("QR code file is empty")
	}
}

func TestShare_InvalidMethod(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := GenerateInvitation(peerID, pub, "")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Share with invalid method.
	opts := DefaultShareOptions()
	opts.Method = ShareMethod(999)

	_, err = inv.Share(opts)
	if err == nil {
		t.Error("expected error for invalid share method, got nil")
	}
}

func TestDefaultShareOptions(t *testing.T) {
	opts := DefaultShareOptions()

	if opts.Method != ShareText {
		t.Errorf("default method: got %v, want %v", opts.Method, ShareText)
	}
	if opts.Subject == "" {
		t.Error("default subject is empty")
	}
	if opts.Body == "" {
		t.Error("default body is empty")
	}
	if opts.QRSize != 512 {
		t.Errorf("default QR size: got %d, want 512", opts.QRSize)
	}
	if opts.TempDirPath != os.TempDir() {
		t.Errorf("default temp dir: got %q, want %q", opts.TempDirPath, os.TempDir())
	}
}

func TestURLEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "spaces",
			input:    "hello world",
			expected: "hello%20world",
		},
		{
			name:     "newlines",
			input:    "line1\nline2",
			expected: "line1%0Aline2",
		},
		{
			name:     "colons",
			input:    "http://example.com",
			expected: "http%3A%2F%2Fexample.com",
		},
		{
			name:     "mixed",
			input:    "hello world\ntest:value",
			expected: "hello%20world%0Atest%3Avalue",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlEncode(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShare_QRCustomSize(t *testing.T) {
	// Generate test invitation.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	inv, err := GenerateInvitation(peerID, pub, "")
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Test different QR sizes.
	sizes := []int{128, 256, 512, 1024}
	tempDir := t.TempDir()

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			opts := DefaultShareOptions()
			opts.Method = ShareQR
			opts.QRSize = size
			opts.TempDirPath = tempDir

			result, err := inv.Share(opts)
			if err != nil {
				t.Fatalf("sharing QR with size %d: %v", size, err)
			}

			// Verify file exists and is not empty.
			info, err := os.Stat(result)
			if err != nil {
				t.Fatalf("stat QR file: %v", err)
			}
			if info.Size() == 0 {
				t.Errorf("QR code file is empty for size %d", size)
			}
		})
	}
}

func TestShare_EmailWithWelcomeMessage(t *testing.T) {
	// Generate test invitation with welcome message.
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}

	peerID, err := peer.Decode("12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp")
	if err != nil {
		t.Fatalf("creating peer ID: %v", err)
	}

	welcomeMsg := "Hello friend!"
	inv, err := GenerateInvitation(peerID, pub, welcomeMsg)
	if err != nil {
		t.Fatalf("generating invitation: %v", err)
	}

	// Share as email.
	opts := DefaultShareOptions()
	opts.Method = ShareEmail

	result, err := inv.Share(opts)
	if err != nil {
		t.Fatalf("sharing email: %v", err)
	}

	// Verify welcome message is included in body (URL-encoded).
	// Welcome message should appear somewhere in the URL.
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "friend") {
		t.Error("mailto: body missing welcome message content")
	}
}
