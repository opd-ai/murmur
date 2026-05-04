// Package discovery provides IPFS DHT namespace-based bootstrap resolver.
// Per PLAN.md "Layer 3 — Public IPFS DHT Namespace (passive, organic, zero-maintenance)".
package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// DHTNamespaceResolver discovers peers via IPFS DHT namespace advertisement.
// Per PLAN.md: "wraps go-libp2p routingDiscovery, calls Advertise() + FindPeers()
// on /murmur/bootstrap/v1. 12s timeout."
type DHTNamespaceResolver struct {
	host          host.Host
	namespace     string
	routingDisc   *routing.RoutingDiscovery
	advertiseOnce bool
}

// NewDHTNamespaceResolver creates a DHT namespace-based bootstrap resolver.
// namespace is the DHT rendezvous key (e.g., "/murmur/bootstrap/v1")
func NewDHTNamespaceResolver(h host.Host, routingDisc *routing.RoutingDiscovery, namespace string) *DHTNamespaceResolver {
	return &DHTNamespaceResolver{
		host:        h,
		namespace:   namespace,
		routingDisc: routingDisc,
	}
}

// Resolve discovers peers by querying the DHT namespace.
// Per PLAN.md: "Every running MURMUR node — CI ephemeral or real user — announces
// itself under the DHT key /murmur/bootstrap/v1 using routingDiscovery.Advertise()."
func (d *DHTNamespaceResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	if d.routingDisc == nil {
		return nil, fmt.Errorf("routing discovery not initialized")
	}

	d.advertiseOnceIfNeeded(ctx)

	findCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	peerChan, err := d.routingDisc.FindPeers(findCtx, d.namespace)
	if err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	return d.collectPeers(findCtx, peerChan)
}

// advertiseOnceIfNeeded advertises this node to the DHT namespace on first call.
func (d *DHTNamespaceResolver) advertiseOnceIfNeeded(ctx context.Context) {
	if d.advertiseOnce {
		return
	}

	advCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, _ = d.routingDisc.Advertise(advCtx, d.namespace)
	d.advertiseOnce = true
}

// collectPeers gathers peers from the discovery channel until timeout or target count.
func (d *DHTNamespaceResolver) collectPeers(ctx context.Context, peerChan <-chan peer.AddrInfo) ([]peer.AddrInfo, error) {
	const targetPeerCount = 10
	var peers []peer.AddrInfo

	for {
		select {
		case p, ok := <-peerChan:
			if !ok {
				return d.finalizePeerList(peers)
			}
			peers = d.addPeerIfValid(peers, p, targetPeerCount)
			if len(peers) >= targetPeerCount {
				return peers, nil
			}
		case <-ctx.Done():
			return d.finalizePeerList(peers)
		}
	}
}

// addPeerIfValid adds a peer to the list if it passes validation.
func (d *DHTNamespaceResolver) addPeerIfValid(peers []peer.AddrInfo, p peer.AddrInfo, targetCount int) []peer.AddrInfo {
	if d.shouldIncludePeer(p) {
		return append(peers, p)
	}
	return peers
}

// shouldIncludePeer returns true if the peer should be added to the list.
func (d *DHTNamespaceResolver) shouldIncludePeer(p peer.AddrInfo) bool {
	return p.ID != d.host.ID()
}

// finalizePeerList validates the collected peer list and returns an error if empty.
func (d *DHTNamespaceResolver) finalizePeerList(peers []peer.AddrInfo) ([]peer.AddrInfo, error) {
	if len(peers) == 0 {
		return nil, fmt.Errorf("no peers found")
	}
	return peers, nil
}

// Name returns the resolver name for logging.
func (d *DHTNamespaceResolver) Name() string {
	return "dht-namespace"
}
