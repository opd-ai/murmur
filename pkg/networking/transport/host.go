// Package transport provides libp2p host construction and transport configuration.
// Per NETWORK_ARCHITECTURE.md, the transport layer uses Noise XX encryption,
// QUIC and TCP transports, and yamux for stream multiplexing.
//
// Transport Fallback Chain (NETWORK_ARCHITECTURE.md §4):
// The host tries transports in order of preference:
//  1. QUIC (UDP, fastest, best for modern networks)
//  2. TCP (reliable fallback, widely supported)
//  3. WebSocket (for browser clients, requires HTTP upgrade)
//  4. WebRTC (for browser-to-browser, uses ICE/STUN)
//
// libp2p handles fallback automatically: it attempts all available transports
// when dialing a peer and uses whatever succeeds first.
package transport

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
)

// Host configuration constants per NETWORK_ARCHITECTURE.md.
const (
	// DefaultConnectionTimeout is the timeout for establishing connections.
	DefaultConnectionTimeout = 30 * time.Second

	// DefaultStreamTimeout is the timeout for stream operations.
	DefaultStreamTimeout = 60 * time.Second

	// DefaultIdleTimeout is the idle timeout for connections.
	DefaultIdleTimeout = 30 * time.Second

	// MaxPeerConnections is the maximum number of simultaneous peer connections.
	// Per NETWORK_ARCHITECTURE.md §7: "Each node maintains a maximum of 200 simultaneous peer connections."
	MaxPeerConnections = 200

	// LowWaterMark is the low watermark for connection pruning (80% of max).
	// When connections fall below this, the connection manager stops pruning.
	LowWaterMark = 160

	// HighWaterMark is the high watermark for connection pruning (90% of max).
	// When connections exceed this, the connection manager starts pruning.
	HighWaterMark = 180

	// ConnectionGracePeriod is the grace period before a connection is eligible for pruning.
	ConnectionGracePeriod = 30 * time.Second
)

// Config holds configuration for constructing a libp2p host.
type Config struct {
	// PrivateKey is the Ed25519 private key for the host identity.
	PrivateKey ed25519.PrivateKey

	// ListenAddrs are the multiaddrs to listen on.
	ListenAddrs []string

	// BootstrapPeers are the initial peers to connect to.
	BootstrapPeers []peer.AddrInfo

	// EnableDHT enables Kademlia DHT for peer discovery.
	EnableDHT bool

	// DHTServerMode enables server mode for DHT (full participant vs client-only).
	DHTServerMode bool

	// EnableWebSocket enables WebSocket transport for browser clients.
	// Per NETWORK_ARCHITECTURE.md, WebSocket is used for browser connectivity.
	EnableWebSocket bool

	// EnableWebRTC enables WebRTC transport for browser-to-browser direct connections.
	// Per NETWORK_ARCHITECTURE.md, WebRTC is used for direct browser peer connections.
	EnableWebRTC bool

	// EnableConnectionManager enables the connection manager for enforcing connection limits.
	// Per NETWORK_ARCHITECTURE.md, max 200 simultaneous peer connections.
	EnableConnectionManager bool

	// MaxConnections overrides the default MaxPeerConnections (200).
	// Only used if EnableConnectionManager is true.
	MaxConnections int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		},
		EnableDHT:               true,
		DHTServerMode:           true,
		EnableWebSocket:         false, // Disabled by default, enable for nodes serving browsers
		EnableWebRTC:            false, // Disabled by default, enable for browser-to-browser
		EnableConnectionManager: true,  // Enabled by default per NETWORK_ARCHITECTURE.md
		MaxConnections:          MaxPeerConnections,
	}
}

// DefaultConfigWithWebSocket returns a Config with WebSocket enabled.
// Per NETWORK_ARCHITECTURE.md, WebSocket is the fallback transport for browser clients.
func DefaultConfigWithWebSocket() Config {
	return Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/tcp/0/ws",
		},
		EnableDHT:               true,
		DHTServerMode:           true,
		EnableWebSocket:         true,
		EnableWebRTC:            false,
		EnableConnectionManager: true,
		MaxConnections:          MaxPeerConnections,
	}
}

// DefaultConfigWithWebRTC returns a Config with WebRTC enabled for browser-to-browser.
// Per NETWORK_ARCHITECTURE.md, WebRTC enables direct browser peer connections using ICE/STUN/TURN.
func DefaultConfigWithWebRTC() Config {
	return Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		},
		EnableDHT:               true,
		DHTServerMode:           true,
		EnableWebSocket:         false,
		EnableWebRTC:            true,
		EnableConnectionManager: true,
		MaxConnections:          MaxPeerConnections,
	}
}

// DefaultConfigWithAllTransports returns a Config with all transports enabled.
// This is suitable for relay nodes that need maximum connectivity options.
func DefaultConfigWithAllTransports() Config {
	return Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		},
		EnableDHT:               true,
		DHTServerMode:           true,
		EnableWebSocket:         true,
		EnableWebRTC:            true,
		EnableConnectionManager: true,
		MaxConnections:          MaxPeerConnections,
	}
}

// Host wraps a libp2p host with MURMUR-specific functionality.
type Host struct {
	host.Host
	dht *dht.IpfsDHT
}

// NewHost creates a new libp2p host with the given configuration.
// Per NETWORK_ARCHITECTURE.md §4-5, the host uses Noise XX encryption,
// prefers QUIC transport with TCP fallback, and derives Peer ID from Ed25519 key.
func NewHost(ctx context.Context, cfg Config) (*Host, error) {
	if cfg.PrivateKey == nil {
		return nil, fmt.Errorf("private key is required")
	}

	privKey, err := crypto.UnmarshalEd25519PrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	listenAddrs, err := parseListenAddresses(cfg.ListenAddrs)
	if err != nil {
		return nil, err
	}

	var idht *dht.IpfsDHT
	opts := buildBaseOptions(privKey, listenAddrs, cfg.EnableWebSocket, cfg.EnableWebRTC)

	// Add connection manager if enabled.
	// Per NETWORK_ARCHITECTURE.md §7: max 200 simultaneous peer connections.
	if cfg.EnableConnectionManager {
		maxConns := cfg.MaxConnections
		if maxConns <= 0 {
			maxConns = MaxPeerConnections
		}
		// Calculate watermarks: low=80%, high=90% of max.
		lowWater := maxConns * 80 / 100
		highWater := maxConns * 90 / 100
		if lowWater < 1 {
			lowWater = 1
		}
		if highWater <= lowWater {
			highWater = lowWater + 1
		}

		cm, err := connmgr.NewConnManager(
			lowWater,
			highWater,
			connmgr.WithGracePeriod(ConnectionGracePeriod),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection manager: %w", err)
		}
		opts = append(opts, libp2p.ConnectionManager(cm))
	}

	if cfg.EnableDHT {
		opts = append(opts, buildDHTOption(ctx, cfg.DHTServerMode, &idht))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	return &Host{Host: h, dht: idht}, nil
}

// parseListenAddresses converts string addresses to multiaddrs.
func parseListenAddresses(addrs []string) ([]multiaddr.Multiaddr, error) {
	listenAddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid listen address %q: %w", addr, err)
		}
		listenAddrs = append(listenAddrs, ma)
	}
	return listenAddrs, nil
}

// buildBaseOptions returns common libp2p options for transport and security.
// Transport Fallback Chain per NETWORK_ARCHITECTURE.md §4:
// Transports are registered in preference order (QUIC → TCP → WebSocket → WebRTC).
// libp2p will try each transport when dialing and use the first that succeeds.
func buildBaseOptions(privKey crypto.PrivKey, listenAddrs []multiaddr.Multiaddr, enableWS, enableWebRTC bool) []libp2p.Option {
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrs(listenAddrs...),
		// Transport fallback chain: QUIC (preferred) → TCP → WebSocket → WebRTC
		libp2p.Transport(libp2pquic.NewTransport), // Fastest, modern networks
		libp2p.Transport(tcp.NewTCPTransport),     // Reliable fallback
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.ConnectionGater(nil),
	}

	// Add WebSocket transport if enabled (position 3 in fallback chain).
	// Per NETWORK_ARCHITECTURE.md, WebSocket is used for browser client connectivity.
	if enableWS {
		opts = append(opts, libp2p.Transport(websocket.New))
	}

	// Add WebRTC transport if enabled (position 4 in fallback chain).
	// Per NETWORK_ARCHITECTURE.md, WebRTC enables direct browser-to-browser connections.
	if enableWebRTC {
		opts = append(opts, libp2p.Transport(libp2pwebrtc.New))
	}

	return opts
}

// buildDHTOption creates a routing option that initializes the DHT.
func buildDHTOption(ctx context.Context, serverMode bool, idht **dht.IpfsDHT) libp2p.Option {
	return libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dhtOpts := []dht.Option{}
		if serverMode {
			dhtOpts = append(dhtOpts, dht.Mode(dht.ModeAutoServer))
		} else {
			dhtOpts = append(dhtOpts, dht.Mode(dht.ModeClient))
		}
		*idht, err = dht.New(ctx, h, dhtOpts...)
		return *idht, err
	})
}

// DHT returns the Kademlia DHT instance, or nil if DHT is disabled.
func (h *Host) DHT() *dht.IpfsDHT {
	return h.dht
}

// PeerID returns the libp2p peer ID derived from the host's public key.
func (h *Host) PeerID() peer.ID {
	return h.Host.ID()
}

// Addrs returns the multiaddrs the host is listening on.
func (h *Host) Addrs() []multiaddr.Multiaddr {
	return h.Host.Addrs()
}

// AddrInfo returns the peer.AddrInfo for this host.
func (h *Host) AddrInfo() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.PeerID(),
		Addrs: h.Addrs(),
	}
}

// Close shuts down the host and releases resources.
func (h *Host) Close() error {
	if h.dht != nil {
		if err := h.dht.Close(); err != nil {
			return fmt.Errorf("failed to close DHT: %w", err)
		}
	}
	if err := h.Host.Close(); err != nil {
		return fmt.Errorf("failed to close host: %w", err)
	}
	return nil
}
