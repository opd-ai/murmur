// Package bootstrap provides invitation acceptance integration.
package bootstrap

import (
	"fmt"

	"github.com/opd-ai/murmur/pkg/identity"
)

// AcceptInvitation processes an invitation URI and returns the decoded invitation.
// Per VIRAL_GROWTH_AND_ONBOARDING.md and ROADMAP.md line 788, invitation
// acceptance is integrated into the onboarding flow for warm-start bootstrap.
func AcceptInvitation(uri string) (*identity.Invitation, error) {
	if uri == "" {
		return nil, fmt.Errorf("empty invitation URI")
	}

	// Decode the invitation from the URI.
	inv, err := identity.DecodeInvitation(uri)
	if err != nil {
		return nil, fmt.Errorf("decoding invitation: %w", err)
	}

	// Validate the invitation structure.
	if err := inv.Validate(); err != nil {
		return nil, fmt.Errorf("invalid invitation: %w", err)
	}

	return inv, nil
}

// BuildBootstrapAddrFromInvitation constructs a bootstrap peer address from an invitation.
// Returns a libp2p multiaddr string that can be used to connect to the inviter's node.
// Per VIRAL_GROWTH_AND_ONBOARDING.md, invitations enable direct bootstrap bypass.
func BuildBootstrapAddrFromInvitation(inv *identity.Invitation) string {
	// Build a basic multiaddr from the peer ID.
	// In practice, this would include the inviter's network addresses,
	// but for now we return just the peer ID format.
	// The networking layer will use DHT to discover the actual addresses.
	return fmt.Sprintf("/p2p/%s", inv.PeerID.String())
}
