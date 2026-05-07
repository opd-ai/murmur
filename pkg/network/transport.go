// Package network defines a transport-neutral networking contract for desktop and WASM runtimes.
package network

import (
	"context"
	"errors"
)

var (
	// ErrNotImplemented indicates a platform adapter is still pending concrete implementation.
	ErrNotImplemented = errors.New("network adapter not implemented")
)

// Platform identifies networking target platform.
type Platform string

const (
	PlatformDesktop Platform = "desktop"
	PlatformWASM    Platform = "wasm"
)

// Message is a transport-neutral payload envelope for gameplay traffic.
type Message struct {
	Topic   string
	From    string
	Payload []byte
}

// Config configures adapter construction and bootstrap behavior.
type Config struct {
	Platform       Platform
	PeerID         string
	BootstrapPeers []string
	STUNServers    []string
	RelayPeers     []string
}

// Adapter abstracts network lifecycle and message exchange.
type Adapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(topic string) (<-chan Message, error)
	DialPeer(ctx context.Context, peerAddr string) error
	Name() string
}

// NewAdapter selects an adapter constructor for the configured target platform.
func NewAdapter(cfg Config) (Adapter, error) {
	switch cfg.Platform {
	case PlatformDesktop:
		return newDesktopAdapter(cfg)
	case PlatformWASM:
		return newWASMAdapter(cfg)
	default:
		return nil, errors.New("unknown network platform")
	}
}
