// Package discovery provides Kademlia DHT bootstrap and peer routing.
// Per DESIGN_DOCUMENT.md Part II §5, discovery uses hardcoded bootstrap
// nodes with user-configurable additions and peer exchange fallback.
package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// RoutingTableRefreshInterval is how often the DHT routing table is refreshed.
// Per DESIGN_DOCUMENT.md Part II §5.
const RoutingTableRefreshInterval = 10 * time.Minute

// BootstrapTimeout is the maximum time to wait for bootstrap connections.
const BootstrapTimeout = 30 * time.Second

// Discovery manages peer discovery via Kademlia DHT.
type Discovery struct {
	h             host.Host
	dht           *dht.IpfsDHT
	bootstrapOnce sync.Once
	fallbackChain *ResolverChain
}

// New creates a new Discovery instance with the given DHT.
func New(h host.Host, d *dht.IpfsDHT) *Discovery {
	return &Discovery{
		h:   h,
		dht: d,
	}
}

// SetFallbackResolvers configures fallback resolvers for bootstrap.
// These are tried if hardcoded bootstrap peers all fail.
func (d *Discovery) SetFallbackResolvers(chain *ResolverChain) {
	d.fallbackChain = chain
}

// Bootstrap connects to the given bootstrap peers and starts DHT discovery.
// This should be called once after the host is created.
// Per DESIGN_DOCUMENT.md, bootstrap nodes are community-operated, multi-jurisdiction.
func (d *Discovery) Bootstrap(ctx context.Context, peers []peer.AddrInfo) error {
	var bootstrapErr error
	d.bootstrapOnce.Do(func() {
		bootstrapErr = d.doBootstrap(ctx, peers)
	})
	return bootstrapErr
}

func (d *Discovery) doBootstrap(ctx context.Context, peers []peer.AddrInfo) error {
	ctx, cancel := context.WithTimeout(ctx, BootstrapTimeout)
	defer cancel()

	connected := 0
	var lastErr error
	if len(peers) > 0 {
		connected, lastErr = d.connectToPeers(ctx, peers)
	}

	// If no connections succeeded, try fallback resolvers
	if connected == 0 && d.fallbackChain != nil {
		fallbackPeers, err := d.fallbackChain.Resolve(ctx)
		if err == nil && len(fallbackPeers) > 0 {
			// Try connecting to fallback peers
			connected, lastErr = d.connectToPeers(ctx, fallbackPeers)
		}
	}

	if connected == 0 && lastErr != nil {
		return lastErr
	}

	return d.bootstrapDHT(ctx)
}

// connectToPeers connects to bootstrap peers in parallel.
func (d *Discovery) connectToPeers(ctx context.Context, peers []peer.AddrInfo) (int, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var connected int
	var lastErr error

	for _, pinfo := range peers {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			if d.connectToPeer(ctx, pi) {
				mu.Lock()
				connected++
				mu.Unlock()
			} else {
				mu.Lock()
				lastErr = fmt.Errorf("failed to connect to bootstrap peer %s", pi.ID)
				mu.Unlock()
			}
		}(pinfo)
	}

	wg.Wait()
	return connected, lastErr
}

// connectToPeer attempts to connect to a single peer.
func (d *Discovery) connectToPeer(ctx context.Context, pi peer.AddrInfo) bool {
	d.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)
	return d.h.Connect(ctx, pi) == nil
}

// bootstrapDHT initializes the DHT if available.
func (d *Discovery) bootstrapDHT(ctx context.Context) error {
	if d.dht == nil {
		return nil
	}
	if err := d.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("DHT bootstrap failed: %w", err)
	}
	return nil
}

// FindPeers discovers peers via DHT random walks.
// Returns a channel that emits discovered peers.
func (d *Discovery) FindPeers(ctx context.Context) (<-chan peer.AddrInfo, error) {
	if d.dht == nil {
		return nil, fmt.Errorf("DHT not initialized")
	}

	out := make(chan peer.AddrInfo, 32)
	go d.emitRoutingTablePeers(ctx, out)

	return out, nil
}

// emitRoutingTablePeers sends peers from the routing table to the output channel.
func (d *Discovery) emitRoutingTablePeers(ctx context.Context, out chan<- peer.AddrInfo) {
	defer close(out)

	routingTable := d.dht.RoutingTable()
	for _, p := range routingTable.ListPeers() {
		if !d.emitPeerIfHasAddrs(ctx, out, p) {
			return
		}
	}
}

// emitPeerIfHasAddrs sends a peer to the channel if it has addresses.
// Returns false if the context was cancelled.
func (d *Discovery) emitPeerIfHasAddrs(ctx context.Context, out chan<- peer.AddrInfo, p peer.ID) bool {
	addrs := d.h.Peerstore().Addrs(p)
	if len(addrs) == 0 {
		return true
	}

	select {
	case out <- peer.AddrInfo{ID: p, Addrs: addrs}:
		return true
	case <-ctx.Done():
		return false
	}
}

// RoutingTable returns the DHT routing table for inspection.
func (d *Discovery) RoutingTable() *kb.RoutingTable {
	if d.dht == nil {
		return nil
	}
	return d.dht.RoutingTable()
}

// NumPeers returns the number of peers in the routing table.
func (d *Discovery) NumPeers() int {
	rt := d.RoutingTable()
	if rt == nil {
		return 0
	}
	return rt.Size()
}

// Close shuts down discovery services.
func (d *Discovery) Close() error {
	// DHT is managed by the host, not closed here
	return nil
}
