// Package onramp_tor provides a libp2p transport adapter for Tor using go-i2p/onramp.
// Per PLAN.md §5.3: Wrap onramp.Onion to provide libp2p transport interface.
package onramp_tor

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/go-i2p/onramp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	gtransport "github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"

	transport "github.com/opd-ai/murmur/pkg/networking/transport"
)

// Transport implements libp2p transport.Transport interface for Tor hidden services.
// Wraps onramp.Onion to provide Dial and Listen semantics compatible with libp2p.
type Transport struct {
	onion    *onramp.Onion
	upgrader gtransport.Upgrader
	rcmgr    network.ResourceManager
	mu       sync.Mutex
	closed   bool
}

// NewTransport creates a new Tor transport adapter.
// The underlying onion instance is constructed once per host and reused for the
// process lifetime per PLAN.md §5.3 lifecycle requirements.
// The name parameter identifies this Tor instance for key persistence.
func NewTransport(ctx context.Context, name string, upgrader gtransport.Upgrader, rcmgr network.ResourceManager) (*Transport, error) {
	if upgrader == nil {
		return nil, fmt.Errorf("upgrader cannot be nil")
	}
	if rcmgr == nil {
		return nil, fmt.Errorf("resource manager cannot be nil")
	}

	onion, err := onramp.NewOnion(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create Onion instance: %w", err)
	}

	return &Transport{
		onion:    onion,
		upgrader: upgrader,
		rcmgr:    rcmgr,
	}, nil
}

// Dial connects to a remote peer via Tor hidden service.
// Per PLAN.md §5.3: resolve /onion3 multiaddr, delegate to Onion.Dial,
// wrap net.Conn in libp2p's connection upgrader for Noise + multiplexing.
func (t *Transport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (gtransport.CapableConn, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	onionAddr, err := parseOnion3Addr(raddr)
	if err != nil {
		return nil, fmt.Errorf("invalid onion3 multiaddr: %w", err)
	}

	rawConn, err := t.onion.Dial("tcp", onionAddr)
	if err != nil {
		return nil, fmt.Errorf("onion dial failed: %w", err)
	}

	return transport.upgradeConnection(ctx, rawConn, t, t.upgrader, t.rcmgr, raddr, p)
}

// Listen creates a listener on a Tor hidden service.
// Per PLAN.md §5.3: delegate to Onion.Listen, translate returned hidden-service
// address into an /onion3 multiaddr.
func (t *Transport) Listen(laddr ma.Multiaddr) (transport.Listener, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	port, err := extractPort(laddr)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %w", err)
	}

	netListener, err := t.onion.Listen(port)
	if err != nil {
		return nil, fmt.Errorf("onion listen failed: %w", err)
	}

	onionAddr := netListener.Addr().String()
	listenerMultiaddr, err := onionAddrToMultiaddr(onionAddr)
	if err != nil {
		netListener.Close()
		return nil, fmt.Errorf("failed to convert onion address to multiaddr: %w", err)
	}

	maListener, err := manet.WrapNetListener(netListener)
	if err != nil {
		netListener.Close()
		return nil, fmt.Errorf("failed to wrap listener: %w", err)
	}

	gatedListener := t.upgrader.GateMaListener(maListener)
	upgradedListener := t.upgrader.UpgradeGatedMaListener(t, gatedListener)

	return &listener{
		Listener:  upgradedListener,
		multiaddr: listenerMultiaddr,
	}, nil
}

// CanDial returns true if this transport can dial the given multiaddr.
func (t *Transport) CanDial(addr ma.Multiaddr) bool {
	return hasOnion3Protocol(addr)
}

// Proxy returns true if this transport is a proxy transport.
func (t *Transport) Proxy() bool {
	return true
}

// Protocols returns the set of protocols handled by this transport.
func (t *Transport) Protocols() []int {
	return []int{ma.P_ONION3}
}

// Close shuts down the transport.
// Per PLAN.md §5.3: ensure Onion.Close runs on host shutdown to release
// control-port resources cleanly.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.onion != nil {
		return t.onion.Close()
	}
	return nil
}

// listener wraps a libp2p Listener with the correct multiaddr.
type listener struct {
	transport.Listener
	multiaddr ma.Multiaddr
}

func (l *listener) Multiaddr() ma.Multiaddr {
	return l.multiaddr
}

// parseOnion3Addr extracts the onion address and port from an onion3 multiaddr.
// Example input: /onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001
// Returns: "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd.onion:9001"
func parseOnion3Addr(maddr ma.Multiaddr) (string, error) {
	parts := ma.Split(maddr)
	for _, part := range parts {
		proto := part.Protocols()[0]
		if proto.Code == ma.P_ONION3 {
			addr, err := part.ValueForProtocol(ma.P_ONION3)
			if err != nil {
				return "", err
			}
			if !strings.Contains(addr, ":") {
				return "", fmt.Errorf("onion3 address missing port: %s", addr)
			}
			components := strings.SplitN(addr, ":", 2)
			return fmt.Sprintf("%s.onion:%s", components[0], components[1]), nil
		}
	}
	return "", fmt.Errorf("no onion3 protocol found in multiaddr")
}

// extractPort extracts the port number from a multiaddr for listening.
// Returns empty string for auto-assignment (port 0).
func extractPort(maddr ma.Multiaddr) (string, error) {
	parts := ma.Split(maddr)
	for _, part := range parts {
		proto := part.Protocols()[0]
		if proto.Code == ma.P_ONION3 {
			addr, err := part.ValueForProtocol(ma.P_ONION3)
			if err != nil {
				return "", err
			}
			if strings.Contains(addr, ":") {
				components := strings.SplitN(addr, ":", 2)
				return components[1], nil
			}
			return "0", nil
		}
		if proto.Code == ma.P_TCP {
			port, err := part.ValueForProtocol(ma.P_TCP)
			if err != nil {
				return "", err
			}
			return port, nil
		}
	}
	return "0", nil
}

// onionAddrToMultiaddr converts a Tor onion address to a multiaddr.
// Example input: "vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd.onion:9001"
// Returns: /onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001
func onionAddrToMultiaddr(addr string) (ma.Multiaddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid onion address format: %w", err)
	}

	host = strings.TrimSuffix(host, ".onion")
	if len(host) != 56 {
		return nil, fmt.Errorf("invalid onion3 address length: %d (expected 56)", len(host))
	}

	return ma.NewMultiaddr(fmt.Sprintf("/onion3/%s:%s", host, port))
}

// hasOnion3Protocol checks if a multiaddr contains the onion3 protocol.
func hasOnion3Protocol(maddr ma.Multiaddr) bool {
	for _, proto := range maddr.Protocols() {
		if proto.Code == ma.P_ONION3 {
			return true
		}
	}
	return false
}

var _ transport.Transport = (*Transport)(nil)
