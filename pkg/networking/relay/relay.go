// Package relay provides NAT traversal, DCUtR hole punching, and relay fallback.
// Per DESIGN_DOCUMENT.md Part II §8, relay nodes facilitate connections
// between NAT-bound peers.
//
// NAT Types:
// - Open: No NAT (public IP) - can receive direct connections
// - Cone: Single hole punch usually works (most residential)
// - Symmetric: Per-destination NAT - requires relay
//
// Traversal Strategy:
// 1. AutoNAT probing at startup to detect NAT type
// 2. DCUtR (Direct Connection Upgrade through Relay) for hole punching
// 3. Relay reservation for double-NAT scenarios
// 4. Relay node selection based on latency and availability
package relay

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"

	"github.com/multiformats/go-multiaddr"
)

// NATType represents the detected NAT configuration.
type NATType int

const (
	// NATTypeUnknown means NAT type not yet probed.
	NATTypeUnknown NATType = iota
	// NATTypePublic means no NAT (public IP).
	NATTypePublic
	// NATTypeCone means cone NAT (hole punch likely works).
	NATTypeCone
	// NATTypeSymmetric means symmetric NAT (relay required).
	NATTypeSymmetric
)

func (n NATType) String() string {
	switch n {
	case NATTypePublic:
		return "public"
	case NATTypeCone:
		return "cone"
	case NATTypeSymmetric:
		return "symmetric"
	default:
		return "unknown"
	}
}

// Traverser manages NAT traversal using DCUtR hole punching and relay fallback.
// Per DESIGN_DOCUMENT.md Part II §8, it enables residential users behind NAT
// to participate in the network.
type Traverser struct {
	host       host.Host
	hpService  *holepunch.Service
	natType    NATType
	relays     []peer.AddrInfo
	relayConns map[peer.ID]*client.Reservation

	mu sync.RWMutex
}

// New creates a NAT Traverser for the given host.
// The host should be created with relay and hole punch options enabled.
func New(h host.Host) *Traverser {
	return &Traverser{
		host:       h,
		natType:    NATTypeUnknown,
		relays:     make([]peer.AddrInfo, 0),
		relayConns: make(map[peer.ID]*client.Reservation),
	}
}

// SetHolePunchService registers the hole punch service for coordination.
// This should be called after the host is fully initialized.
func (t *Traverser) SetHolePunchService(hps *holepunch.Service) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hpService = hps
}

// NATType returns the detected NAT type.
func (t *Traverser) NATType() NATType {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.natType
}

// SetNATType sets the detected NAT type (called by AutoNAT probing).
func (t *Traverser) SetNATType(natType NATType) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.natType = natType
}

// AddRelays registers relay nodes for use in NAT traversal.
// Per DESIGN_DOCUMENT.md, relay nodes are selected based on latency and availability.
func (t *Traverser) AddRelays(relays []peer.AddrInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.relays = append(t.relays, relays...)
}

// Relays returns the list of known relay nodes.
func (t *Traverser) Relays() []peer.AddrInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]peer.AddrInfo, len(t.relays))
	copy(result, t.relays)
	return result
}

// ConnectViaRelay establishes a connection to the target peer through a relay.
// This is used when direct connection fails (e.g., symmetric NAT).
// Per DESIGN_DOCUMENT.md Part II §8, this is the fallback for double-NAT scenarios.
func (t *Traverser) ConnectViaRelay(ctx context.Context, target peer.AddrInfo) error {
	t.mu.RLock()
	relays := make([]peer.AddrInfo, len(t.relays))
	copy(relays, t.relays)
	t.mu.RUnlock()

	var lastErr error
	for _, relay := range relays {
		// Connect to relay first
		if err := t.host.Connect(ctx, relay); err != nil {
			lastErr = err
			continue
		}

		// Build relay address for target
		relayAddr, err := buildRelayAddr(relay.ID, target.ID)
		if err != nil {
			lastErr = err
			continue
		}

		// Try to connect via relay
		targetWithRelay := peer.AddrInfo{
			ID:    target.ID,
			Addrs: append(target.Addrs, relayAddr),
		}
		if err := t.host.Connect(ctx, targetWithRelay); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

// buildRelayAddr constructs a circuit v2 relay address.
func buildRelayAddr(relayID, targetID peer.ID) (multiaddr.Multiaddr, error) {
	// Circuit v2 address format: /p2p/<relay-id>/p2p-circuit/p2p/<target-id>
	addrStr := "/p2p/" + relayID.String() + "/p2p-circuit/p2p/" + targetID.String()
	return multiaddr.NewMultiaddr(addrStr)
}

// MakeReservation creates a relay reservation with the specified relay.
// This advertises our reachability through the relay.
func (t *Traverser) MakeReservation(ctx context.Context, relay peer.AddrInfo) error {
	// Connect to relay
	if err := t.host.Connect(ctx, relay); err != nil {
		return err
	}

	// Request reservation
	reservation, err := client.Reserve(ctx, t.host, relay)
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.relayConns[relay.ID] = reservation
	t.mu.Unlock()

	return nil
}

// HasReservation checks if we have an active reservation with a relay.
func (t *Traverser) HasReservation(relayID peer.ID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.relayConns[relayID]
	return ok
}

// DirectConnect attempts hole punching to establish a direct connection.
// Per DESIGN_DOCUMENT.md Part II §8, this uses DCUtR protocol.
func (t *Traverser) DirectConnect(ctx context.Context, target peer.ID) error {
	t.mu.RLock()
	hps := t.hpService
	t.mu.RUnlock()

	if hps == nil {
		// No hole punch service, fall back to relay
		return network.ErrNoConn
	}

	// The hole punch service will coordinate with the target
	return hps.DirectConnect(target)
}

// Config holds NAT traversal configuration options.
type Config struct {
	// EnableRelay enables circuit relay functionality.
	EnableRelay bool
	// EnableHolePunch enables DCUtR hole punching.
	EnableHolePunch bool
	// RelayOnly forces relay-only mode (no direct connections).
	RelayOnly bool
	// AutoNATProbeInterval is how often to probe NAT type.
	AutoNATProbeInterval time.Duration
}

// DefaultConfig returns a default NAT traversal configuration.
func DefaultConfig() Config {
	return Config{
		EnableRelay:          true,
		EnableHolePunch:      true,
		RelayOnly:            false,
		AutoNATProbeInterval: 30 * time.Second,
	}
}

// HostOptions returns libp2p host options for NAT traversal.
// These should be passed when constructing the libp2p host.
func HostOptions(cfg Config) []libp2p.Option {
	var opts []libp2p.Option

	if cfg.EnableRelay {
		// Enable circuit v2 client (for connecting via relays)
		opts = append(opts, libp2p.EnableRelay())
	}

	if cfg.EnableHolePunch {
		// Enable hole punching
		opts = append(opts, libp2p.EnableHolePunching())
	}

	// Always enable AutoNAT for NAT type detection
	opts = append(opts, libp2p.EnableNATService())

	return opts
}
