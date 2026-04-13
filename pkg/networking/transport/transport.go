// Package transport provides libp2p host construction and transport configuration.
// Per NETWORK_ARCHITECTURE.md, the transport layer uses Noise XX encryption,
// QUIC and TCP transports, and yamux for stream multiplexing.
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
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
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
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ListenAddrs: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		},
		EnableDHT:     true,
		DHTServerMode: true,
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
	opts := buildBaseOptions(privKey, listenAddrs)

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
func buildBaseOptions(privKey crypto.PrivKey, listenAddrs []multiaddr.Multiaddr) []libp2p.Option {
	return []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.ConnectionGater(nil),
	}
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
