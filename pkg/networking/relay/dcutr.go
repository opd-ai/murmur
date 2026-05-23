// Package relay provides NAT traversal, DCUtR hole punching, and relay fallback.
// This file implements the DCUtR (Direct Connection Upgrade through Relay) service.
// Per NETWORK_ARCHITECTURE.md §Hole Punching, DCUtR coordinates simultaneous
// outbound connections from both NATed peers to punch through NATs.
package relay

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/multiformats/go-multiaddr"
)

// DCUtRTimeout is the maximum time to wait for hole punching to complete.
const DCUtRTimeout = 30 * time.Second

// DCUtRRetries is the number of times to retry hole punching before giving up.
const DCUtRRetries = 3

// HolePunchResult represents the outcome of a hole punch attempt.
type HolePunchResult int

const (
	// HolePunchUnknown means the result is not yet known.
	HolePunchUnknown HolePunchResult = iota
	// HolePunchSuccess means direct connection was established.
	HolePunchSuccess
	// HolePunchFailed means hole punch failed, staying on relay.
	HolePunchFailed
	// HolePunchTimeout means hole punch timed out.
	HolePunchTimeout
)

// String returns the string representation of the hole punch result.
func (r HolePunchResult) String() string {
	switch r {
	case HolePunchSuccess:
		return "success"
	case HolePunchFailed:
		return "failed"
	case HolePunchTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// DCUtRService manages DCUtR hole punching for NAT traversal.
// Per NETWORK_ARCHITECTURE.md, it coordinates simultaneous outbound connections
// from both NATed peers to upgrade relay connections to direct connections.
type DCUtRService struct {
	host       host.Host
	hpService  *holepunch.Service
	autoNAT    *AutoNATService
	results    map[peer.ID]HolePunchResult
	inProgress map[peer.ID]bool
	listeners  []chan HolePunchEvent

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// HolePunchEvent is emitted when a hole punch attempt completes.
type HolePunchEvent struct {
	Peer   peer.ID
	Result HolePunchResult
	Direct bool // true if now using direct connection
}

// NewDCUtRService creates a DCUtR service.
// The host should be created with libp2p.EnableHolePunching() option.
func NewDCUtRService(h host.Host, autoNAT *AutoNATService) *DCUtRService {
	ctx, cancel := context.WithCancel(context.Background())

	return &DCUtRService{
		host:       h,
		autoNAT:    autoNAT,
		results:    make(map[peer.ID]HolePunchResult),
		inProgress: make(map[peer.ID]bool),
		listeners:  make([]chan HolePunchEvent, 0),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetHolePunchService registers the libp2p hole punch service.
func (d *DCUtRService) SetHolePunchService(hps *holepunch.Service) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hpService = hps
}

// Start begins the DCUtR service.
func (d *DCUtRService) Start() {
	go d.monitorConnections()
}

// Stop stops the DCUtR service.
func (d *DCUtRService) Stop() {
	d.cancel()
}

// Subscribe returns a channel that receives hole punch events.
func (d *DCUtRService) Subscribe() <-chan HolePunchEvent {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan HolePunchEvent, 16)
	d.listeners = append(d.listeners, ch)
	return ch
}

// Unsubscribe removes a listener channel.
func (d *DCUtRService) Unsubscribe(ch <-chan HolePunchEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, listener := range d.listeners {
		if listener == ch {
			d.listeners = append(d.listeners[:i], d.listeners[i+1:]...)
			close(listener)
			return
		}
	}
}

// TryHolePunch attempts to establish a direct connection to the peer.
// If already connected via relay, attempts to upgrade to direct.
// Per NETWORK_ARCHITECTURE.md, this uses DCUtR protocol.
func (d *DCUtRService) TryHolePunch(ctx context.Context, peerID peer.ID) (HolePunchResult, error) {
	if !d.markInProgress(peerID) {
		return HolePunchUnknown, nil
	}
	defer d.unmarkInProgress(peerID)

	hps := d.getHolePunchService()
	if hps == nil {
		return HolePunchFailed, network.ErrNoConn
	}

	if d.isDirectlyConnected(peerID) {
		d.recordResult(peerID, HolePunchSuccess)
		return HolePunchSuccess, nil
	}

	ctx, cancel := context.WithTimeout(ctx, DCUtRTimeout)
	defer cancel()

	return d.attemptHolePunchWithRetries(ctx, peerID, hps)
}

// markInProgress marks a peer as having a hole punch in progress.
// Returns false if already in progress.
func (d *DCUtRService) markInProgress(peerID peer.ID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.inProgress[peerID] {
		return false
	}
	d.inProgress[peerID] = true
	return true
}

// unmarkInProgress removes the in-progress marker for a peer.
func (d *DCUtRService) unmarkInProgress(peerID peer.ID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.inProgress, peerID)
}

// getHolePunchService safely retrieves the hole punch service.
func (d *DCUtRService) getHolePunchService() *holepunch.Service {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.hpService
}

// attemptHolePunchWithRetries tries to establish a direct connection with retries.
func (d *DCUtRService) attemptHolePunchWithRetries(ctx context.Context, peerID peer.ID, hps *holepunch.Service) (HolePunchResult, error) {
	var lastErr error
	for i := 0; i < DCUtRRetries; i++ {
		// F-ERR-3 fix: Use errors.Is for context.Canceled check instead of direct comparison.
		result, err := d.tryDirectConnect(ctx, peerID, hps)
		if result != HolePunchFailed || errors.Is(err, context.Canceled) {
			return result, err
		}
		lastErr = err

		if ctx.Err() != nil {
			d.recordResult(peerID, HolePunchTimeout)
			return HolePunchTimeout, ctx.Err()
		}

		time.Sleep(100 * time.Millisecond)
	}

	d.recordResult(peerID, HolePunchFailed)
	return HolePunchFailed, lastErr
}

// tryDirectConnect attempts a single hole punch connection.
func (d *DCUtRService) tryDirectConnect(ctx context.Context, peerID peer.ID, hps *holepunch.Service) (HolePunchResult, error) {
	err := hps.DirectConnect(peerID)
	if err == nil && d.isDirectlyConnected(peerID) {
		d.recordResult(peerID, HolePunchSuccess)
		return HolePunchSuccess, nil
	}
	return HolePunchFailed, err
}

// isDirectlyConnected checks if we have a direct (non-relay) connection to peer.
func (d *DCUtRService) isDirectlyConnected(peerID peer.ID) bool {
	conns := d.host.Network().ConnsToPeer(peerID)
	for _, conn := range conns {
		// Check if connection is not relayed
		if !isRelayedConnection(conn) {
			return true
		}
	}
	return false
}

// isRelayedConnection checks if a connection goes through a relay.
func isRelayedConnection(conn network.Conn) bool {
	// Check if the remote multiaddr contains /p2p-circuit/
	raddr := conn.RemoteMultiaddr()
	if raddr == nil {
		return false
	}
	return containsCircuit(raddr.String())
}

// containsCircuit checks if an address string contains circuit relay.
func containsCircuit(addr string) bool {
	return contains(addr, "/p2p-circuit/")
}

// contains is a simple substring check.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// recordResult stores the result and notifies listeners.
func (d *DCUtRService) recordResult(peerID peer.ID, result HolePunchResult) {
	d.mu.Lock()
	d.results[peerID] = result
	listeners := make([]chan HolePunchEvent, len(d.listeners))
	copy(listeners, d.listeners)
	d.mu.Unlock()

	event := HolePunchEvent{
		Peer:   peerID,
		Result: result,
		Direct: result == HolePunchSuccess,
	}

	for _, ch := range listeners {
		select {
		case ch <- event:
		default:
		}
	}
}

// GetResult returns the last hole punch result for a peer.
func (d *DCUtRService) GetResult(peerID peer.ID) HolePunchResult {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.results[peerID]
}

// monitorConnections watches for relay connections that might benefit from hole punching.
func (d *DCUtRService) monitorConnections() {
	// Monitor for new relay connections
	notifee := &connectionNotifee{dcutr: d}
	d.host.Network().Notify(notifee)

	<-d.ctx.Done()
	d.host.Network().StopNotify(notifee)
}

// connectionNotifee implements network.Notifiee for connection monitoring.
type connectionNotifee struct {
	dcutr *DCUtRService
}

// Connected is called when a new connection is established. It attempts hole punching for relayed connections.
func (n *connectionNotifee) Connected(net network.Network, conn network.Conn) {
	// Check if this is a relay connection that we should try to upgrade
	if isRelayedConnection(conn) {
		peerID := conn.RemotePeer()

		// Check if AutoNAT indicates we're behind NAT
		if n.dcutr.autoNAT != nil && !n.dcutr.autoNAT.IsPublic() {
			// Attempt hole punch in background
			go func() {
				ctx, cancel := context.WithTimeout(n.dcutr.ctx, DCUtRTimeout)
				defer cancel()
				_, _ = n.dcutr.TryHolePunch(ctx, peerID)
			}()
		}
	}
}

// Disconnected is called when a connection is closed. No-op for DCUtR.
func (n *connectionNotifee) Disconnected(network.Network, network.Conn) {}

// Listen is called when a new listener is created. No-op for DCUtR.
func (n *connectionNotifee) Listen(network.Network, multiaddr.Multiaddr) {}

// ListenClose is called when a listener is closed. No-op for DCUtR.
func (n *connectionNotifee) ListenClose(network.Network, multiaddr.Multiaddr) {}
