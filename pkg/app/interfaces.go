// Package app provides the top-level application lifecycle and event bus for MURMUR.
// This file defines interfaces for subsystem dependencies to reduce coupling.
package app

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	pb "github.com/opd-ai/murmur/proto"
)

// WaveStore defines the interface for Wave persistence operations.
// Implemented by pkg/content/storage.Cache.
type WaveStore interface {
	// Store saves a Wave to persistent storage.
	Store(ctx context.Context, wave *pb.Wave) error

	// Get retrieves a Wave by its ID.
	Get(ctx context.Context, waveID []byte) (*pb.Wave, error)

	// Delete removes a Wave from storage.
	Delete(ctx context.Context, waveID []byte) error

	// Expired returns all Waves that have exceeded their TTL.
	Expired(ctx context.Context) ([]*pb.Wave, error)

	// PruneExpired removes all expired Waves from storage.
	PruneExpired(ctx context.Context) (int, error)
}

// PeerRegistry defines the interface for peer connection management.
// Implemented by pkg/networking/transport.Host.
type PeerRegistry interface {
	// Host returns the underlying libp2p host.
	// This is needed for subsystems that require direct libp2p access.
	Host() host.Host

	// Connect establishes a connection to a peer.
	Connect(ctx context.Context, pi peer.AddrInfo) error

	// Disconnect closes the connection to a peer.
	Disconnect(ctx context.Context, peerID peer.ID) error

	// Peers returns all currently connected peers.
	Peers() []peer.ID
}

// IdentityProvider defines the interface for identity operations.
// Implemented by pkg/identity/keys.KeyPair.
type IdentityProvider interface {
	// PublicKey returns the Ed25519 public key bytes.
	PublicKey() []byte

	// Sign signs a message with the Ed25519 private key.
	Sign(message []byte) []byte

	// PeerID returns the libp2p peer ID derived from the public key.
	PeerID() peer.ID
}

// MessagePublisher defines the interface for publishing messages to GossipSub topics.
// Implemented by pkg/networking/gossip.PubSub.
type MessagePublisher interface {
	// Publish sends a message to a topic.
	Publish(ctx context.Context, topic string, data []byte) error

	// Subscribe subscribes to a topic and returns a message channel.
	Subscribe(ctx context.Context, topic string) (<-chan *pb.MurmurEnvelope, error)

	// Unsubscribe unsubscribes from a topic.
	Unsubscribe(ctx context.Context, topic string) error
}
