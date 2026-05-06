// Package onramp_i2p provides a libp2p transport adapter for I2P using go-i2p/onramp.
// Per PLAN.md §5.4: Wrap onramp.Garlic to provide libp2p transport interface.
package onramp_i2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/go-i2p/onramp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	gtransport "github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"

	transport "github.com/opd-ai/murmur/pkg/networking/transport/onramp"
)

const (
	// DefaultSAMAddr is the default SAMv3 bridge address.
	DefaultSAMAddr = "127.0.0.1:7656"
)

// Transport implements libp2p transport.Transport interface for I2P destinations.
// Wraps onramp.Garlic to provide Dial and Listen semantics compatible with libp2p.
type Transport struct {
	garlic   *onramp.Garlic
	upgrader gtransport.Upgrader
	rcmgr    network.ResourceManager
	mu       sync.Mutex
	closed   bool
}

// NewTransport creates a new I2P transport adapter.
// The underlying garlic instance is constructed once per host and reused for the
// process lifetime per PLAN.md §5.4 lifecycle requirements.
// The tunName parameter identifies this I2P tunnel for key persistence.
// The samAddr parameter specifies the SAMv3 bridge address (default: 127.0.0.1:7656).
// The options parameter configures I2P tunnel parameters (inbound/outbound length, quantity).
func NewTransport(ctx context.Context, tunName, samAddr string, options []string, upgrader gtransport.Upgrader, rcmgr network.ResourceManager) (*Transport, error) {
	if upgrader == nil {
		return nil, fmt.Errorf("upgrader cannot be nil")
	}
	if rcmgr == nil {
		return nil, fmt.Errorf("resource manager cannot be nil")
	}
	if samAddr == "" {
		samAddr = DefaultSAMAddr
	}

	garlic, err := onramp.NewGarlic(tunName, samAddr, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create Garlic instance: %w", err)
	}

	return &Transport{
		garlic:   garlic,
		upgrader: upgrader,
		rcmgr:    rcmgr,
	}, nil
}

// Dial connects to a remote peer via I2P destination.
// Per PLAN.md §5.4: resolve /garlic64 multiaddr, delegate to Garlic.Dial,
// wrap net.Conn in libp2p's connection upgrader for Noise + multiplexing.
func (t *Transport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (gtransport.CapableConn, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	garlicAddr, err := parseGarlicAddr(raddr)
	if err != nil {
		return nil, fmt.Errorf("invalid garlic multiaddr: %w", err)
	}

	rawConn, err := t.garlic.DialContext(ctx, "tcp", garlicAddr)
	if err != nil {
		return nil, fmt.Errorf("garlic dial failed: %w", err)
	}

	return transport.UpgradeConnection(ctx, rawConn, t, t.upgrader, t.rcmgr, raddr, p)
}

// Listen creates a listener on an I2P destination.
// Per PLAN.md §5.4: delegate to Garlic.Listen, translate the returned I2P
// destination into a /garlic64 multiaddr.
func (t *Transport) Listen(laddr ma.Multiaddr) (gtransport.Listener, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	netListener, err := t.garlic.Listen()
	if err != nil {
		return nil, fmt.Errorf("garlic listen failed: %w", err)
	}

	garlicAddr := netListener.Addr().String()
	listenerMultiaddr, err := garlicAddrToMultiaddr(garlicAddr)
	if err != nil {
		netListener.Close()
		return nil, fmt.Errorf("failed to convert garlic address to multiaddr: %w", err)
	}

	return transport.UpgradeListener(netListener, listenerMultiaddr, t, t.upgrader)
}

// CanDial returns true if this transport can dial the given multiaddr.
func (t *Transport) CanDial(addr ma.Multiaddr) bool {
	return hasGarlicProtocol(addr)
}

// Proxy returns true if this transport is a proxy transport.
func (t *Transport) Proxy() bool {
	return true
}

// Protocols returns the set of protocols handled by this transport.
func (t *Transport) Protocols() []int {
	return []int{ma.P_GARLIC64}
}

// Close shuts down the transport.
// Per PLAN.md §5.4: ensure Garlic.Close runs on host shutdown to release
// SAM session resources cleanly.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.garlic != nil {
		return t.garlic.Close()
	}
	return nil
}

// parseGarlicAddr extracts the I2P destination from a garlic64 multiaddr.
// Example input: /garlic64/<base64-encoded-destination>
// Returns the base64 destination string suitable for Garlic.Dial.
func parseGarlicAddr(maddr ma.Multiaddr) (string, error) {
	parts := ma.Split(maddr)
	for _, part := range parts {
		proto := part.Protocols()[0]
		if proto.Code == ma.P_GARLIC64 {
			addr, err := part.ValueForProtocol(ma.P_GARLIC64)
			if err != nil {
				return "", err
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("no garlic64 protocol found in multiaddr")
}

// garlicAddrToMultiaddr converts an I2P destination to a multiaddr.
// Example input: base64-encoded I2P destination (516+ bytes)
// Returns: /garlic64/<base64-destination>
func garlicAddrToMultiaddr(addr string) (ma.Multiaddr, error) {
	decoded, err := base64.StdEncoding.DecodeString(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 I2P destination: %w", err)
	}

	if len(decoded) < 387 {
		return nil, fmt.Errorf("I2P destination too short: %d bytes (expected ≥387)", len(decoded))
	}

	return ma.NewMultiaddr(fmt.Sprintf("/garlic64/%s", addr))
}

// hasGarlicProtocol checks if a multiaddr contains the garlic64 protocol.
func hasGarlicProtocol(maddr ma.Multiaddr) bool {
	for _, proto := range maddr.Protocols() {
		if proto.Code == ma.P_GARLIC64 {
			return true
		}
	}
	return false
}

// parsePort extracts a port from multiaddr components.
// I2P uses virtual ports within the tunnel, but the multiaddr may not include one.
// Returns empty string if no port is specified.
func parsePort(maddr ma.Multiaddr) string {
	parts := ma.Split(maddr)
	for _, part := range parts {
		proto := part.Protocols()[0]
		if proto.Code == ma.P_TCP || proto.Code == ma.P_UDP {
			port, err := part.ValueForProtocol(proto.Code)
			if err == nil {
				return port
			}
		}
	}
	return ""
}

// appendPortIfPresent appends a port to an I2P destination if present in the multiaddr.
// I2P destinations can include virtual ports (e.g., destination:9001).
func appendPortIfPresent(dest string, maddr ma.Multiaddr) string {
	port := parsePort(maddr)
	if port != "" && port != "0" && !strings.Contains(dest, ":") {
		return fmt.Sprintf("%s:%s", dest, port)
	}
	return dest
}

var _ gtransport.Transport = (*Transport)(nil)
