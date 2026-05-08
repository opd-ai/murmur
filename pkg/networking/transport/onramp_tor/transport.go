// Package onramptor provides a libp2p transport adapter for Tor using go-i2p/onramp.
// Per PLAN.md §5.3: Wrap onramp.Onion to provide libp2p transport interface.
package onramptor

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

	transport "github.com/opd-ai/murmur/pkg/networking/transport/onramp"
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

	return transport.UpgradeConnection(ctx, rawConn, t, t.upgrader, t.rcmgr, raddr, p)
}

// Listen creates a listener on a Tor hidden service.
// Per PLAN.md §5.3: delegate to Onion.Listen, translate returned hidden-service
// address into an /onion3 multiaddr.
func (t *Transport) Listen(laddr ma.Multiaddr) (gtransport.Listener, error) {
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

	return transport.UpgradeListener(netListener, listenerMultiaddr, t, t.upgrader)
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
	return transport.SafeClose(&t.mu, &t.closed, t.onion)
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
		if port := tryExtractPortFromComponent(part); port != "" {
			return port, nil
		}
	}
	return "0", nil
}

// tryExtractPortFromComponent attempts to extract port from multiaddr component.
func tryExtractPortFromComponent(part ma.Component) string {
	proto := part.Protocols()[0]

	if port := tryExtractOnion3Port(part, proto); port != "" {
		return port
	}

	if port := tryExtractTCPPort(part, proto); port != "" {
		return port
	}

	return ""
}

// tryExtractOnion3Port extracts port from onion3 protocol.
func tryExtractOnion3Port(part ma.Component, proto ma.Protocol) string {
	if proto.Code != ma.P_ONION3 {
		return ""
	}

	addr, err := part.ValueForProtocol(ma.P_ONION3)
	if err != nil {
		return ""
	}

	if strings.Contains(addr, ":") {
		components := strings.SplitN(addr, ":", 2)
		return components[1]
	}
	return "0"
}

// tryExtractTCPPort extracts port from TCP protocol.
func tryExtractTCPPort(part ma.Component, proto ma.Protocol) string {
	if proto.Code != ma.P_TCP {
		return ""
	}

	port, err := part.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return ""
	}
	return port
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

var _ gtransport.Transport = (*Transport)(nil)
