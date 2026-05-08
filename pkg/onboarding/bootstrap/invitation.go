// Package bootstrap provides invitation acceptance integration.
package bootstrap

import (
	"fmt"
	"strings"

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
//
// When the invitation contains signed bootstrap addresses (produced by
// [identity.GenerateSignedInvitation]), the first address is returned; these are fully
// dialable multiaddreses including transport layer (e.g.
// /ip4/1.2.3.4/tcp/4001/p2p/<peerID>).
//
// For unsigned/legacy invitations that carry no bootstrap addresses, the function
// returns a bare /p2p/<peerID> address as a last-resort diagnostic value.  This
// address form is NOT directly dialable but can be used for peer routing and DHT
// lookups.  Callers MUST check [IsBareP2PAddr] and fall back to DHT discovery when
// only a bare address is available.
func BuildBootstrapAddrFromInvitation(inv *identity.Invitation) string {
	if len(inv.BootstrapAddrs) > 0 {
		return inv.BootstrapAddrs[0]
	}

	// Bare /p2p/<peerID>: last-resort diagnostic value.  Not directly dialable;
	// the networking layer must use DHT to resolve transport addresses before dialling.
	return fmt.Sprintf("/p2p/%s", inv.PeerID.String())
}

// BuildBootstrapAddrsFromInvitation returns all bootstrap addresses embedded in
// the invitation, with a /p2p fallback for legacy unsigned invitations.
//
// Callers should check whether the returned slice contains only bare /p2p addresses
// using [IsBareP2PAddr] and initiate DHT lookup if so.
func BuildBootstrapAddrsFromInvitation(inv *identity.Invitation) []string {
	if len(inv.BootstrapAddrs) > 0 {
		addrs := make([]string, 0, len(inv.BootstrapAddrs))
		for _, addr := range inv.BootstrapAddrs {
			trimmed := strings.TrimSpace(addr)
			if trimmed == "" {
				continue
			}
			addrs = append(addrs, trimmed)
		}
		if len(addrs) > 0 {
			return addrs
		}
	}

	return []string{BuildBootstrapAddrFromInvitation(inv)}
}

// IsBareP2PAddr reports whether addr is a bare /p2p/<peerID> multiaddr (no
// transport layer).  Callers that receive such addresses from
// [BuildBootstrapAddrFromInvitation] should not attempt direct dial and should
// instead use DHT-based peer routing to discover transport addresses.
func IsBareP2PAddr(addr string) bool {
	trimmed := strings.TrimSpace(addr)
	if !strings.HasPrefix(trimmed, "/p2p/") {
		return false
	}
	// A dialable address with a transport would have more path components, e.g.
	// /ip4/1.2.3.4/tcp/4001/p2p/QmXXX. A bare peer addr is exactly /p2p/<id>.
	rest := strings.TrimPrefix(trimmed, "/p2p/")
	return !strings.ContainsRune(rest, '/')
}
