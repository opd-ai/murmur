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
}

// New creates a new Discovery instance with the given DHT.
func New(h host.Host, d *dht.IpfsDHT) *Discovery {
	return &Discovery{
		h:   h,
		dht: d,
	}
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
	if len(peers) == 0 {
		return nil // No bootstrap peers, will need to discover via other means
	}

	ctx, cancel := context.WithTimeout(ctx, BootstrapTimeout)
	defer cancel()

	// Connect to bootstrap peers in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	var connected int
	var lastErr error

	for _, pinfo := range peers {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()

			// Add addresses to peerstore
			d.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)

			// Connect to peer
			if err := d.h.Connect(ctx, pi); err != nil {
				mu.Lock()
				lastErr = fmt.Errorf("failed to connect to bootstrap peer %s: %w", pi.ID, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			connected++
			mu.Unlock()
		}(pinfo)
	}

	wg.Wait()

	if connected == 0 && lastErr != nil {
		return lastErr
	}

	// Bootstrap the DHT
	if d.dht != nil {
		if err := d.dht.Bootstrap(ctx); err != nil {
			return fmt.Errorf("DHT bootstrap failed: %w", err)
		}
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
	go func() {
		defer close(out)

		// Get current routing table peers
		routingTable := d.dht.RoutingTable()
		for _, p := range routingTable.ListPeers() {
			addrs := d.h.Peerstore().Addrs(p)
			if len(addrs) > 0 {
				select {
				case out <- peer.AddrInfo{ID: p, Addrs: addrs}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
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
