// Package relay provides NAT traversal, DCUtR hole punching, and relay fallback.
// This file implements relay node capacity limits.
// Per NETWORK_ARCHITECTURE.md §Relay Nodes:
// - Max 128 concurrent relay reservations
// - Max 128 KB/s per relayed connection
package relay

import (
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

// Default capacity limits per NETWORK_ARCHITECTURE.md §Relay Nodes.
const (
	// DefaultMaxReservations is the maximum concurrent relay reservations.
	// Per spec: "128 concurrent reservations".
	DefaultMaxReservations = 128

	// DefaultMaxDataPerConn is the max data per relayed connection (128 KB).
	// Per spec: "128 KB/s per relayed connection".
	DefaultMaxDataPerConn = 128 * 1024 // 128 KB

	// DefaultConnectionDuration is how long a relayed connection can last.
	// We use 2 minutes, after which data limits reset.
	DefaultConnectionDuration = 2 * time.Minute

	// DefaultReservationTTL is how long a reservation is valid.
	DefaultReservationTTL = 1 * time.Hour

	// DefaultMaxCircuitsPerPeer is max open circuits per peer.
	DefaultMaxCircuitsPerPeer = 16

	// DefaultMaxReservationsPerIP is max reservations per IP address.
	DefaultMaxReservationsPerIP = 8

	// DefaultMaxReservationsPerASN is max reservations per ASN.
	DefaultMaxReservationsPerASN = 32

	// DefaultBufferSize is the relay connection buffer size.
	DefaultBufferSize = 2048
)

// RelayCapacityConfig holds relay node capacity configuration.
type RelayCapacityConfig struct {
	// MaxReservations is the maximum concurrent relay reservations.
	MaxReservations int

	// MaxDataPerConnection is the max data per relayed connection in bytes.
	MaxDataPerConnection int64

	// ConnectionDuration is how long a relayed connection can last.
	ConnectionDuration time.Duration

	// ReservationTTL is how long a reservation is valid.
	ReservationTTL time.Duration

	// MaxCircuitsPerPeer is max open circuits per peer.
	MaxCircuitsPerPeer int

	// MaxReservationsPerIP is max reservations per IP address.
	MaxReservationsPerIP int

	// MaxReservationsPerASN is max reservations per ASN.
	MaxReservationsPerASN int

	// BufferSize is the relay connection buffer size.
	BufferSize int
}

// DefaultRelayCapacityConfig returns the default relay capacity configuration.
// Per NETWORK_ARCHITECTURE.md §Relay Nodes: 128 concurrent, 128 KB/s.
func DefaultRelayCapacityConfig() RelayCapacityConfig {
	return RelayCapacityConfig{
		MaxReservations:       DefaultMaxReservations,
		MaxDataPerConnection:  DefaultMaxDataPerConn,
		ConnectionDuration:    DefaultConnectionDuration,
		ReservationTTL:        DefaultReservationTTL,
		MaxCircuitsPerPeer:    DefaultMaxCircuitsPerPeer,
		MaxReservationsPerIP:  DefaultMaxReservationsPerIP,
		MaxReservationsPerASN: DefaultMaxReservationsPerASN,
		BufferSize:            DefaultBufferSize,
	}
}

// ToLibp2pResources converts the config to libp2p relay v2 Resources.
func (c RelayCapacityConfig) ToLibp2pResources() relayv2.Resources {
	return relayv2.Resources{
		Limit: &relayv2.RelayLimit{
			Duration: c.ConnectionDuration,
			Data:     c.MaxDataPerConnection,
		},
		ReservationTTL:        c.ReservationTTL,
		MaxReservations:       c.MaxReservations,
		MaxCircuits:           c.MaxCircuitsPerPeer,
		BufferSize:            c.BufferSize,
		MaxReservationsPerIP:  c.MaxReservationsPerIP,
		MaxReservationsPerASN: c.MaxReservationsPerASN,
	}
}

// ToLibp2pOptions converts the config to libp2p relay v2 options.
func (c RelayCapacityConfig) ToLibp2pOptions() []relayv2.Option {
	return []relayv2.Option{
		relayv2.WithResources(c.ToLibp2pResources()),
	}
}

// RelayService wraps a libp2p relay v2 service with capacity limits.
type RelayService struct {
	relay  *relayv2.Relay
	config RelayCapacityConfig
}

// NewRelayService creates a relay service with capacity limits.
// Per NETWORK_ARCHITECTURE.md §Relay Nodes, relay service should only be
// enabled for publicly reachable nodes.
func NewRelayService(h host.Host, config RelayCapacityConfig) (*RelayService, error) {
	opts := config.ToLibp2pOptions()
	r, err := relayv2.New(h, opts...)
	if err != nil {
		return nil, err
	}

	return &RelayService{
		relay:  r,
		config: config,
	}, nil
}

// NewRelayServiceWithDefaults creates a relay service with default capacity limits.
func NewRelayServiceWithDefaults(h host.Host) (*RelayService, error) {
	return NewRelayService(h, DefaultRelayCapacityConfig())
}

// Close shuts down the relay service.
func (r *RelayService) Close() error {
	return r.relay.Close()
}

// Config returns the relay capacity configuration.
func (r *RelayService) Config() RelayCapacityConfig {
	return r.config
}

// LowCapacityConfig returns a configuration for resource-constrained nodes.
// These nodes can still contribute to the network but with lower limits.
func LowCapacityConfig() RelayCapacityConfig {
	return RelayCapacityConfig{
		MaxReservations:       32,        // 25% of default
		MaxDataPerConnection:  64 * 1024, // 64 KB
		ConnectionDuration:    DefaultConnectionDuration,
		ReservationTTL:        DefaultReservationTTL,
		MaxCircuitsPerPeer:    8,
		MaxReservationsPerIP:  4,
		MaxReservationsPerASN: 16,
		BufferSize:            1024,
	}
}

// HighCapacityConfig returns a configuration for well-resourced nodes.
// These nodes can handle more relay traffic.
func HighCapacityConfig() RelayCapacityConfig {
	return RelayCapacityConfig{
		MaxReservations:       512,        // 4x default
		MaxDataPerConnection:  256 * 1024, // 256 KB
		ConnectionDuration:    DefaultConnectionDuration,
		ReservationTTL:        DefaultReservationTTL,
		MaxCircuitsPerPeer:    32,
		MaxReservationsPerIP:  16,
		MaxReservationsPerASN: 64,
		BufferSize:            4096,
	}
}
