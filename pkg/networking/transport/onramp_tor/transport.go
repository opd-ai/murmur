// Package onramp_tor provides a libp2p transport adapter for Tor using go-i2p/onramp.
// Per PLAN.md §5.3: Wrap onramp.Onion to provide libp2p transport interface.
package onramp_tor

import (
	"context"
	"fmt"

	"github.com/go-i2p/onramp"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"
)

// Transport implements libp2p transport.Transport interface for Tor hidden services.
// Wraps onramp.Onion to provide Dial and Listen semantics compatible with libp2p.
type Transport struct {
	onion *onramp.Onion
}

// NewTransport creates a new Tor transport adapter.
// The underlying onion instance is constructed once per host and reused for the
// process lifetime per PLAN.md §5.3 lifecycle requirements.
func NewTransport(ctx context.Context) (*Transport, error) {
	// TODO: Implement Onion construction with key persistence.
	// Per PLAN.md §5.3: use onramp's built-in key handling so the same .onion
	// address survives restarts; store keys in MURMUR's existing keystore
	// directory with Argon2id-wrapped encryption for consistency.
	return nil, fmt.Errorf("not yet implemented")
}

// Dial connects to a remote peer via Tor hidden service.
// Per PLAN.md §5.3: resolve /onion3 multiaddr, delegate to Onion.Dial,
// wrap net.Conn in libp2p's connection upgrader for Noise + multiplexing.
func (t *Transport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (transport.CapableConn, error) {
	// TODO: Implement dial logic.
	return nil, fmt.Errorf("not yet implemented")
}

// Listen creates a listener on a Tor hidden service.
// Per PLAN.md §5.3: delegate to Onion.Listen, translate returned hidden-service
// address into an /onion3 multiaddr.
func (t *Transport) Listen(laddr ma.Multiaddr) (transport.Listener, error) {
	// TODO: Implement listen logic.
	return nil, fmt.Errorf("not yet implemented")
}

// CanDial returns true if this transport can dial the given multiaddr.
func (t *Transport) CanDial(addr ma.Multiaddr) bool {
	// TODO: Check if addr is an /onion3 multiaddr.
	return false
}

// Proxy returns true if this transport is a proxy transport.
func (t *Transport) Proxy() bool {
	return true
}

// Protocols returns the set of protocols handled by this transport.
func (t *Transport) Protocols() []int {
	// TODO: Return appropriate protocol identifiers for Tor.
	return []int{}
}

// Close shuts down the transport.
// Per PLAN.md §5.3: ensure Onion.Close runs on host shutdown to release
// control-port resources cleanly.
func (t *Transport) Close() error {
	// TODO: Implement close logic.
	return nil
}

var _ transport.Transport = (*Transport)(nil)
