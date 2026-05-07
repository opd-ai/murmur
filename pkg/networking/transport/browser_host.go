//go:build js && wasm

package transport

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// BrowserHost provides a minimal libp2p-compatible host for WASM environments.
// It doesn't support actual networking, but allows the app to initialize without errors.
// Real WASM networking will use WebRTC relay clients in a future phase.
type BrowserHost struct {
	peerID peer.ID
	addrs  []multiaddr.Multiaddr
	mu     sync.RWMutex
}

// NewBrowserHost creates a new browser-compatible host.
func NewBrowserHost(privKey ed25519.PrivateKey) (*Host, error) {
	// Derive peer ID from the private key
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("deriving peer ID: %w", err)
	}

	browser := &BrowserHost{
		peerID: peerID,
		addrs:  []multiaddr.Multiaddr{},
	}

	// Create a synthetic Host wrapper
	return &Host{
		h: browser,
	}, nil
}

// ID returns the peer ID.
func (bh *BrowserHost) ID() peer.ID {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	return bh.peerID
}

// Addrs returns the multiaddrs this host is listening on.
// For WASM, this returns a synthetic address.
func (bh *BrowserHost) Addrs() []multiaddr.Multiaddr {
	bh.mu.RLock()
	defer bh.mu.RUnlock()

	// If no addresses set, return a default WASM address
	if len(bh.addrs) == 0 {
		addr, _ := multiaddr.NewMultiaddr("/wasm")
		return []multiaddr.Multiaddr{addr}
	}
	return bh.addrs
}

// Host wraps a browser host with network-like functionality.
type Host struct {
	h *BrowserHost
}

// PeerID returns the host's peer ID.
func (h *Host) PeerID() peer.ID {
	return h.h.ID()
}

// Addrs returns the host's multiaddrs.
func (h *Host) Addrs() []multiaddr.Multiaddr {
	return h.h.Addrs()
}

// Connect is a no-op for browser hosts (WebRTC relay will implement this later).
func (h *Host) Connect(ctx context.Context, peerAddr peer.AddrInfo) error {
	// TODO: Implement WebRTC relay client connection
	return nil
}

// Close is a no-op for browser hosts.
func (h *Host) Close() error {
	return nil
}

// StreamHandler is a no-op for browser hosts.
func (h *Host) SetStreamHandler(protocol string, handler interface{}) {
	// TODO: Implement stream handlers over WebRTC
}

// BrowserGossipSub provides a minimal GossipSub-compatible interface for WASM.
// In a future phase, this will connect to relay peers and use WebRTC for actual messaging.
type BrowserGossipSub struct {
	topics map[string]map[peer.ID]chan []byte
	mu     sync.RWMutex
}

// NewBrowserGossipSub creates a new browser-compatible GossipSub.
func NewBrowserGossipSub(ctx context.Context, h *Host) (*PubSub, error) {
	bgossip := &BrowserGossipSub{
		topics: make(map[string]map[peer.ID]chan []byte),
	}

	return &PubSub{
		ps: bgossip,
	}, nil
}

// PubSub wraps a browser gossipsub with pubsub functionality.
type PubSub struct {
	ps *BrowserGossipSub
}

// Subscribe returns a subscription to the given topic.
func (ps *PubSub) Subscribe(ctx context.Context, topic string) (*Subscription, error) {
	ps.ps.mu.Lock()
	defer ps.ps.mu.Unlock()

	if ps.ps.topics[topic] == nil {
		ps.ps.topics[topic] = make(map[peer.ID]chan []byte)
	}

	// Create a channel for messages
	msgChan := make(chan []byte, 32)
	peerID := peer.ID("wasm-peer")
	ps.ps.topics[topic][peerID] = msgChan

	return &Subscription{
		topic: topic,
		ch:    msgChan,
		ps:    ps,
	}, nil
}

// Publish broadcasts a message to a topic.
func (ps *PubSub) Publish(ctx context.Context, topic string, data []byte) error {
	ps.ps.mu.RLock()
	subscribers := ps.ps.topics[topic]
	ps.ps.mu.RUnlock()

	// Deliver to all subscribers
	for _, ch := range subscribers {
		select {
		case ch <- data:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip
		}
	}
	return nil
}

// Close closes the pubsub.
func (ps *PubSub) Close() error {
	ps.ps.mu.Lock()
	defer ps.ps.mu.Unlock()

	for _, subscribers := range ps.ps.topics {
		for _, ch := range subscribers {
			close(ch)
		}
	}
	ps.ps.topics = make(map[string]map[peer.ID]chan []byte)
	return nil
}

// Subscription represents a subscription to a topic.
type Subscription struct {
	topic string
	ch    chan []byte
	ps    *PubSub
}

// Next returns the next message on the subscription.
func (sub *Subscription) Next(ctx context.Context) ([]byte, error) {
	select {
	case msg := <-sub.ch:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Cancel cancels the subscription.
func (sub *Subscription) Cancel() {
	sub.ps.ps.mu.Lock()
	defer sub.ps.ps.mu.Unlock()

	if ch := sub.ps.ps.topics[sub.topic][peer.ID("wasm-peer")]; ch != nil {
		close(ch)
		delete(sub.ps.ps.topics[sub.topic], peer.ID("wasm-peer"))
	}
}
