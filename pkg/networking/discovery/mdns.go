// Package discovery provides peer discovery via mDNS for local network peers.
// Per NETWORK_ARCHITECTURE.md §3 (mDNS Local Discovery):
// "For nodes on the same local network (such as devices on the same Wi-Fi network),
// MURMUR uses multicast DNS (mDNS) to discover local peers without any external
// network connectivity."

package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// mDNS service name for MURMUR nodes.
// Per NETWORK_ARCHITECTURE.md, mDNS discovery broadcasts the node's presence
// on the local network and listens for broadcasts from other MURMUR nodes.
const MDNSServiceName = "_murmur._tcp"

// MDNSDiscoveryInterval is how often mDNS discovery runs.
const MDNSDiscoveryInterval = 10 * time.Second

// MDNSPeerBufferSize is the buffer size for the discovered peers channel.
const MDNSPeerBufferSize = 16

// MDNSDiscovery manages mDNS-based local network peer discovery.
// Per NETWORK_ARCHITECTURE.md §3: "mDNS discovery is enabled by default
// and can be disabled in settings for users who do not wish to advertise
// their MURMUR usage on the local network."
type MDNSDiscovery struct {
	h       host.Host
	service mdns.Service
	mu      sync.RWMutex
	peers   chan peer.AddrInfo
	handler PeerHandler
	started bool
}

// PeerHandler is called when a peer is discovered via mDNS.
type PeerHandler func(peer.AddrInfo)

// NewMDNSDiscovery creates a new mDNS discovery service.
// The handler is called when a new peer is discovered on the local network.
func NewMDNSDiscovery(h host.Host, handler PeerHandler) *MDNSDiscovery {
	return &MDNSDiscovery{
		h:       h,
		peers:   make(chan peer.AddrInfo, MDNSPeerBufferSize),
		handler: handler,
	}
}

// Start begins mDNS discovery on the local network.
// Per NETWORK_ARCHITECTURE.md §3: "This enables fully offline local MURMUR networks —
// two or more devices on the same LAN can form a MURMUR network without any internet connectivity."
func (m *MDNSDiscovery) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Create the mDNS service with the MURMUR service name.
	service := mdns.NewMdnsService(m.h, MDNSServiceName, m)
	if err := service.Start(); err != nil {
		return err
	}

	m.service = service
	m.started = true

	return nil
}

// Stop halts mDNS discovery.
func (m *MDNSDiscovery) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	if m.service != nil {
		if err := m.service.Close(); err != nil {
			return err
		}
		m.service = nil
	}

	m.started = false
	return nil
}

// HandlePeerFound implements mdns.Notifee interface.
// Called when a peer is discovered on the local network.
func (m *MDNSDiscovery) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == m.h.ID() {
		return
	}

	m.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)
	m.notifyHandler(pi)
	m.sendToPeersChannel(pi)
}

// notifyHandler calls the peer handler callback if set.
func (m *MDNSDiscovery) notifyHandler(pi peer.AddrInfo) {
	if m.handler != nil {
		m.handler(pi)
	}
}

// sendToPeersChannel sends peer info to channel, dropping oldest if full.
func (m *MDNSDiscovery) sendToPeersChannel(pi peer.AddrInfo) {
	select {
	case m.peers <- pi:
	default:
		m.drainAndResend(pi)
	}
}

// drainAndResend drops the oldest entry and retries sending.
func (m *MDNSDiscovery) drainAndResend(pi peer.AddrInfo) {
	select {
	case <-m.peers:
	default:
	}
	select {
	case m.peers <- pi:
	default:
	}
}

// Peers returns a channel that emits discovered peers.
func (m *MDNSDiscovery) Peers() <-chan peer.AddrInfo {
	return m.peers
}

// IsStarted returns true if mDNS discovery is running.
func (m *MDNSDiscovery) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// ConnectToPeer attempts to connect to a discovered peer.
// This is a convenience method for handlers that want to immediately connect.
func (m *MDNSDiscovery) ConnectToPeer(ctx context.Context, pi peer.AddrInfo) error {
	return m.h.Connect(ctx, pi)
}
