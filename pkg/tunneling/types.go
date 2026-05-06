// Package tunneling provides a minimal HTTP tunneling primitive for exposing
// localhost services through the MURMUR network. This is a Phase 6.3 prototype
// implementation (single-hop, HTTP-only) to validate addressing and auth models
// before building the full multi-hop Shroud-based tunneling system.
package tunneling

import (
	"crypto/ed25519"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/zeebo/blake3"
)

// TunnelID is a 16-character base32 identifier for a tunnel.
// Format: <name>-<hash> where hash is derived from operator's public key + name.
type TunnelID string

// GenerateTunnelID creates a deterministic tunnel ID from the operator's
// Ed25519 public key and a user-chosen tunnel name.
// Per TUNNEL_DESIGN.md: tunnel ID = BLAKE3(pubkey || name)[0:8] encoded as base32.
func GenerateTunnelID(pubkey ed25519.PublicKey, name string) TunnelID {
	hasher := blake3.New()
	hasher.Write(pubkey)
	hasher.Write([]byte(name))
	hash := hasher.Sum(nil)

	// Take first 8 bytes, encode as base32 without padding
	suffix := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:8])
	suffix = strings.ToLower(suffix) // lowercase for readability

	return TunnelID(fmt.Sprintf("%s-%s", name, suffix))
}

// Validate checks if a tunnel ID has the correct format.
func (id TunnelID) Validate() error {
	s := string(id)
	parts := strings.Split(s, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid tunnel ID format: expected 'name-hash', got %q", s)
	}
	if len(parts[0]) == 0 {
		return fmt.Errorf("tunnel name cannot be empty")
	}
	// The hash is the last part (handles names with dashes like "my-app")
	hash := parts[len(parts)-1]
	// base32 encoding of 8 bytes = 13 characters without padding
	if len(hash) != 13 {
		return fmt.Errorf("tunnel hash must be 13 characters (base32 of 8 bytes), got %d", len(hash))
	}
	return nil
}

// String returns the tunnel ID as a murmur:// URL.
func (id TunnelID) String() string {
	return fmt.Sprintf("murmur://tunnel/%s", string(id))
}

// ParseTunnelAddress extracts the tunnel ID from a murmur://tunnel/<id> URL.
func ParseTunnelAddress(addr string) (TunnelID, error) {
	const prefix = "murmur://tunnel/"
	if !strings.HasPrefix(addr, prefix) {
		return "", fmt.Errorf("invalid tunnel address: must start with %q", prefix)
	}
	id := TunnelID(strings.TrimPrefix(addr, prefix))
	if err := id.Validate(); err != nil {
		return "", fmt.Errorf("invalid tunnel address: %w", err)
	}
	return id, nil
}

// Config holds common configuration for tunneling components.
type Config struct {
	// LocalPort is the localhost port to forward traffic to/from.
	LocalPort int

	// TunnelName is the user-chosen tunnel name (e.g., "alice-dev").
	TunnelName string

	// Ephemeral indicates if the tunnel should expire after 24 hours.
	Ephemeral bool

	// ExitRelayAddr is the network address of the exit relay (for prototype: "host:port").
	ExitRelayAddr string
}
