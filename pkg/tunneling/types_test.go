package tunneling

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func TestGenerateTunnelID(t *testing.T) {
	pubkey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	id := GenerateTunnelID(pubkey, "alice-dev")

	// Check format: name-hash
	if !strings.HasPrefix(string(id), "alice-dev-") {
		t.Errorf("expected tunnel ID to start with 'alice-dev-', got %q", id)
	}

	// Check hash length (13 base32 characters for 8 bytes) - it's the last part
	parts := strings.Split(string(id), "-")
	if len(parts) < 2 {
		t.Errorf("expected at least 2 parts, got %d", len(parts))
	}
	hash := parts[len(parts)-1]
	if len(hash) != 13 {
		t.Errorf("expected hash length 13, got %d", len(hash))
	}

	// Check determinism
	id2 := GenerateTunnelID(pubkey, "alice-dev")
	if id != id2 {
		t.Errorf("tunnel ID not deterministic: %q != %q", id, id2)
	}

	// Check different name produces different ID
	id3 := GenerateTunnelID(pubkey, "bob-test")
	if id == id3 {
		t.Errorf("different names should produce different IDs: %q == %q", id, id3)
	}
}

func TestTunnelIDValidate(t *testing.T) {
	tests := []struct {
		id      TunnelID
		wantErr bool
	}{
		{"alice-dev-abcdefghijklm", false},   // valid
		{"test-1234567890123", false},        // valid
		{"my-app-name-abcdefghijklm", false}, // name with dashes ok
		{"no-hash", true},                    // hash too short
		{"empty--abcdefghijklm", false},      // double dash in name ok (name="empty-")
		{"-abcdefghijklm", true},             // empty name
		{"name", true},                       // no hash
		{"name-", true},                      // empty hash
		{"name-tooshort", true},              // hash too short
		{"name-waytoolongforhash", true},     // hash too long
	}

	for _, tt := range tests {
		err := tt.id.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("TunnelID(%q).Validate() error = %v, wantErr %v", tt.id, err, tt.wantErr)
		}
	}
}

func TestTunnelIDString(t *testing.T) {
	id := TunnelID("alice-dev-a3f2b9c1d5e7f")
	want := "murmur://tunnel/alice-dev-a3f2b9c1d5e7f"
	if got := id.String(); got != want {
		t.Errorf("TunnelID.String() = %q, want %q", got, want)
	}
}

func TestParseTunnelAddress(t *testing.T) {
	tests := []struct {
		addr    string
		wantID  TunnelID
		wantErr bool
	}{
		{"murmur://tunnel/alice-dev-abcdefghijklm", "alice-dev-abcdefghijklm", false},
		{"murmur://tunnel/test-1234567890123", "test-1234567890123", false},
		{"http://example.com", "", true},                     // wrong scheme
		{"murmur://other/alice-dev-abcdefghijklm", "", true}, // wrong type
		{"murmur://tunnel/", "", true},                       // empty ID
		{"murmur://tunnel/invalid", "", true},                // invalid format
		{"murmur://tunnel/name-short", "", true},             // hash too short
	}

	for _, tt := range tests {
		id, err := ParseTunnelAddress(tt.addr)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTunnelAddress(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && id != tt.wantID {
			t.Errorf("ParseTunnelAddress(%q) = %q, want %q", tt.addr, id, tt.wantID)
		}
	}
}
