// Package screens - Shared completion screen logic (no build tags).
// This file contains logic common to both completion_screen.go (!test) and completion_screen_stub.go (test).

package screens

import (
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// derivePeerIDFromPubKey derives a libp2p peer.ID from an Ed25519 public key.
// This does not require a running network host.
// This function is shared between completion_screen.go and completion_screen_stub.go.
func derivePeerIDFromPubKey(pubKey []byte) peer.ID {
	libp2pPub, err := libp2pcrypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return ""
	}
	id, err := peer.IDFromPublicKey(libp2pPub)
	if err != nil {
		return ""
	}
	return id
}

// notifyInviteGenerated fires the OnInviteGenerated callback if registered.
// This method is shared between completion_screen.go and completion_screen_stub.go.
func (s *CompletionScreen) notifyInviteGenerated() {
	if s.callbacks.OnInviteGenerated != nil {
		s.callbacks.OnInviteGenerated(s.inviteCode)
	}
}

// generateInviteCode generates a simple invite code from public key prefix.
// This function is shared between completion_screen.go and completion_screen_stub.go.
func generateInviteCode(pubKey []byte) string {
	if len(pubKey) < 6 {
		return "MURMUR-XXXX-YYYY"
	}
	return "MURMUR-" + hexNibble(pubKey[0]) + hexNibble(pubKey[1]) + hexNibble(pubKey[2]) +
		"-" + hexNibble(pubKey[3]) + hexNibble(pubKey[4]) + hexNibble(pubKey[5])
}

// hexNibble converts a byte to a 4-character hex string.
// This function is shared between completion_screen.go and completion_screen_stub.go.
func hexNibble(b byte) string {
	const hex = "0123456789ABCDEF"
	return string(hex[(b>>4)&0x0F]) + string(hex[b&0x0F])
}
