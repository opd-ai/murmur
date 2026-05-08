//go:build !js

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
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"

	"github.com/opd-ai/murmur/pkg/networking/transport/diagnostics"
	onrampi2p "github.com/opd-ai/murmur/pkg/networking/transport/onramp_i2p"
	onramptor "github.com/opd-ai/murmur/pkg/networking/transport/onramp_tor"
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

	// EnableTor enables Tor transport adapter for /onion3 addresses.
	// Per PLAN.md §5.5, Tor and I2P transports coexist with TCP/QUIC.
	EnableTor bool

	// EnableI2P enables I2P transport adapter for /garlic64 addresses.
	// Per PLAN.md §5.5, both adapters can be registered simultaneously.
	EnableI2P bool

	// TorControlAddr is the Tor control port address (default: 127.0.0.1:9051).
	TorControlAddr string

	// I2PSAMAddr is the I2P SAMv3 address (default: 127.0.0.1:7656).
	I2PSAMAddr string
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
		EnableTor:               false, // Disabled by default, enable with config flag
		EnableI2P:               false, // Disabled by default, enable with config flag
		TorControlAddr:          "127.0.0.1:9051",
		I2PSAMAddr:              "127.0.0.1:7656",
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
// Per PLAN.md §5.5, conditionally registers Tor and I2P transports based on config flags.
// Per PLAN.md §5.7, performs reachability diagnostics before constructing the host.
func NewHost(ctx context.Context, cfg Config) (*Host, error) {
	// Perform transport reachability diagnostics if anonymity transports are enabled.
	if err := performDiagnostics(ctx, cfg); err != nil {
		return nil, err
	}

	privKey, err := validateAndUnmarshalKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	listenAddrs, err := parseListenAddresses(cfg.ListenAddrs)
	if err != nil {
		return nil, err
	}

	var idht *dht.IpfsDHT
	opts := buildBaseOptions(privKey, listenAddrs, cfg.EnableWebSocket, cfg.EnableWebRTC)
	opts = appendConnectionManager(opts, cfg)
	opts, err = appendAnonymityTransports(opts, ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure anonymity transports: %w", err)
	}
	opts = appendRegisteredAdapters(opts, ctx)
	opts = appendDHTOption(opts, ctx, cfg, &idht)

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	return &Host{Host: h, dht: idht}, nil
}

// validateAndUnmarshalKey checks for nil key and unmarshals it.
func validateAndUnmarshalKey(privateKey []byte) (crypto.PrivKey, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("private key is required")
	}
	privKey, err := crypto.UnmarshalEd25519PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}
	return privKey, nil
}

// appendConnectionManager adds connection manager option if enabled.
func appendConnectionManager(opts []libp2p.Option, cfg Config) []libp2p.Option {
	if !cfg.EnableConnectionManager {
		return opts
	}

	lowWater, highWater := calculateWatermarks(cfg.MaxConnections)
	cm, err := connmgr.NewConnManager(
		lowWater,
		highWater,
		connmgr.WithGracePeriod(ConnectionGracePeriod),
	)
	if err != nil {
		return opts // Silently skip if creation fails.
	}
	return append(opts, libp2p.ConnectionManager(cm))
}

// calculateWatermarks computes low and high connection watermarks.
func calculateWatermarks(maxConns int) (low, high int) {
	if maxConns <= 0 {
		maxConns = MaxPeerConnections
	}
	low = maxConns * 80 / 100
	high = maxConns * 90 / 100
	if low < 1 {
		low = 1
	}
	if high <= low {
		high = low + 1
	}
	return low, high
}

// appendDHTOption adds DHT option if enabled.
func appendDHTOption(opts []libp2p.Option, ctx context.Context, cfg Config, idht **dht.IpfsDHT) []libp2p.Option {
	if cfg.EnableDHT {
		return append(opts, buildDHTOption(ctx, cfg.DHTServerMode, idht))
	}
	return opts
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

// appendAnonymityTransports conditionally adds Tor and I2P transport adapters.
// Per PLAN.md §5.5: Both adapters coexist with TCP/QUIC; peers can be reached
// via whichever address they advertise. The multiaddr selection logic prefers
// clearnet when available and anonymity is not required.
func appendAnonymityTransports(opts []libp2p.Option, ctx context.Context, cfg Config) ([]libp2p.Option, error) {
	if cfg.EnableTor {
		torOpt := buildTorTransportOption(ctx, cfg.TorControlAddr)
		opts = append(opts, torOpt)
	}

	if cfg.EnableI2P {
		i2pOpt := buildI2PTransportOption(ctx, cfg.I2PSAMAddr)
		opts = append(opts, i2pOpt)
	}

	return opts, nil
}

// buildTorTransportOption creates a libp2p transport option for Tor.
// Per PLAN.md §5.3: The transport constructor receives an upgrader from libp2p
// and wraps onramp.Onion to provide Dial and Listen semantics.
func buildTorTransportOption(ctx context.Context, controlAddr string) libp2p.Option {
	constructor := func(upgrader transport.Upgrader, rcmgr network.ResourceManager) (transport.Transport, error) {
		return onramptor.NewTransport(ctx, "murmur-tor", upgrader, rcmgr)
	}
	return libp2p.Transport(constructor)
}

// buildI2PTransportOption creates a libp2p transport option for I2P.
// Per PLAN.md §5.4: The transport constructor receives an upgrader from libp2p
// and wraps onramp.Garlic to provide Dial and Listen semantics.
func buildI2PTransportOption(ctx context.Context, samAddr string) libp2p.Option {
	constructor := func(upgrader transport.Upgrader, rcmgr network.ResourceManager) (transport.Transport, error) {
		// Empty options slice uses onramp defaults for tunnel parameters
		return onrampi2p.NewTransport(ctx, "murmur-i2p", samAddr, nil, upgrader, rcmgr)
	}
	return libp2p.Transport(constructor)
}

// performDiagnostics probes Tor and I2P transports for reachability.
// Per PLAN.md §5.7, surfaces actionable errors before host construction
// to fail fast with installation instructions rather than silent fallback.
func performDiagnostics(ctx context.Context, cfg Config) error {
	if !cfg.EnableTor && !cfg.EnableI2P {
		return nil // No diagnostics needed for clearnet-only mode.
	}

	_, err := diagnostics.CheckAll(ctx, cfg.EnableTor, cfg.TorControlAddr, cfg.EnableI2P, cfg.I2PSAMAddr)
	return err
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
