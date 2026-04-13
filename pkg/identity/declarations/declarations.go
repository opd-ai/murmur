// Package declarations provides identity declaration creation and parsing.
// Per DESIGN_DOCUMENT.md, identity declarations are signed announcements
// of identity information broadcast via GossipSub.
package declarations

// Declaration represents a signed identity announcement.
// TODO: Implement per TECHNICAL_IMPLEMENTATION.md §2.1.
type Declaration struct {
	// PublicKey is the Ed25519 public key of the identity.
	PublicKey []byte

	// DisplayName is the optional human-readable name.
	DisplayName string

	// Timestamp is the Unix timestamp of declaration creation.
	Timestamp int64

	// Signature is the Ed25519 signature over the declaration fields.
	Signature []byte
}
