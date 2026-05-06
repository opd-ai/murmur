// Package relay provides NAT traversal, DCUtR hole punching, and relay fallback.
// This file implements the AutoNAT service for NAT status detection.
// Per NETWORK_ARCHITECTURE.md §Hole Punching, AutoNAT probing runs at startup
// and periodically thereafter to determine reachability.
package relay

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

// AutoNATProbeInterval is the default interval between NAT probes.
// Per NETWORK_ARCHITECTURE.md, probing runs at startup and periodically.
const AutoNATProbeInterval = 30 * time.Second

// AutoNATMinPeers is the minimum number of peers needed for reliable NAT detection.
const AutoNATMinPeers = 3

// Reachability represents the detected reachability status.
type Reachability int

const (
	// ReachabilityUnknown means we haven't determined reachability yet.
	ReachabilityUnknown Reachability = iota
	// ReachabilityPublic means we are publicly reachable (no NAT or port forwarded).
	ReachabilityPublic
	// ReachabilityPrivate means we are behind NAT and need relay/hole-punching.
	ReachabilityPrivate
)

// String returns the string representation of the reachability status.
func (r Reachability) String() string {
	switch r {
	case ReachabilityPublic:
		return "public"
	case ReachabilityPrivate:
		return "private"
	default:
		return "unknown"
	}
}

// AutoNATService monitors NAT reachability using libp2p's AutoNAT protocol.
// Per NETWORK_ARCHITECTURE.md §Hole Punching, it asks peers to attempt inbound
// connections to determine if we are publicly reachable.
type AutoNATService struct {
	host         host.Host
	reachability Reachability
	listeners    []chan Reachability
	sub          event.Subscription

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAutoNATService creates an AutoNAT service for the given host.
// The host should be created with libp2p.EnableNATService() option.
func NewAutoNATService(h host.Host) (*AutoNATService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ans := &AutoNATService{
		host:         h,
		reachability: ReachabilityUnknown,
		listeners:    make([]chan Reachability, 0),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Subscribe to reachability events from libp2p's AutoNAT
	sub, err := h.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if err != nil {
		cancel()
		return nil, err
	}
	ans.sub = sub

	return ans, nil
}

// Start begins monitoring reachability changes.
func (a *AutoNATService) Start() {
	go a.monitorReachability()
}

// Stop stops the AutoNAT service.
func (a *AutoNATService) Stop() error {
	a.cancel()
	if a.sub != nil {
		return a.sub.Close()
	}
	return nil
}

// Reachability returns the current detected reachability status.
func (a *AutoNATService) Reachability() Reachability {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.reachability
}

// IsPublic returns true if we are publicly reachable.
func (a *AutoNATService) IsPublic() bool {
	return a.Reachability() == ReachabilityPublic
}

// IsPrivate returns true if we are behind NAT.
func (a *AutoNATService) IsPrivate() bool {
	return a.Reachability() == ReachabilityPrivate
}

// Subscribe returns a channel that receives reachability updates.
func (a *AutoNATService) Subscribe() <-chan Reachability {
	a.mu.Lock()
	defer a.mu.Unlock()

	ch := make(chan Reachability, 1)
	a.listeners = append(a.listeners, ch)

	// Send current status immediately if known
	if a.reachability != ReachabilityUnknown {
		select {
		case ch <- a.reachability:
		default:
		}
	}

	return ch
}

// Unsubscribe removes a listener channel.
func (a *AutoNATService) Unsubscribe(ch <-chan Reachability) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, listener := range a.listeners {
		if listener == ch {
			a.listeners = append(a.listeners[:i], a.listeners[i+1:]...)
			close(listener)
			return
		}
	}
}

// monitorReachability listens for AutoNAT events from libp2p.
func (a *AutoNATService) monitorReachability() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case evt, ok := <-a.sub.Out():
			if !ok {
				return
			}
			a.handleReachabilityEvent(evt.(event.EvtLocalReachabilityChanged))
		}
	}
}

// handleReachabilityEvent processes a reachability change event.
func (a *AutoNATService) handleReachabilityEvent(evt event.EvtLocalReachabilityChanged) {
	newReach := a.convertToReachability(evt.Reachability)

	a.mu.Lock()
	oldReach := a.reachability
	a.reachability = newReach
	listeners := a.copyListeners()
	a.mu.Unlock()

	a.notifyIfChanged(oldReach, newReach, listeners)
}

// convertToReachability maps libp2p reachability to our type.
func (a *AutoNATService) convertToReachability(reach network.Reachability) Reachability {
	switch reach {
	case network.ReachabilityPublic:
		return ReachabilityPublic
	case network.ReachabilityPrivate:
		return ReachabilityPrivate
	default:
		return ReachabilityUnknown
	}
}

// copyListeners creates a copy of listener slice while holding lock.
func (a *AutoNATService) copyListeners() []chan Reachability {
	listeners := make([]chan Reachability, len(a.listeners))
	copy(listeners, a.listeners)
	return listeners
}

// notifyIfChanged sends non-blocking notifications if reachability changed.
func (a *AutoNATService) notifyIfChanged(old, new Reachability, listeners []chan Reachability) {
	if new == old {
		return
	}

	for _, ch := range listeners {
		select {
		case ch <- new:
		default:
		}
	}
}

// NeedsRelay returns true if we need relay service for connectivity.
// Per NETWORK_ARCHITECTURE.md §Relay Nodes, relay is needed when behind NAT.
func (a *AutoNATService) NeedsRelay() bool {
	reach := a.Reachability()
	return reach == ReachabilityPrivate || reach == ReachabilityUnknown
}

// ShouldProvideRelay returns true if we should offer relay service.
// Per NETWORK_ARCHITECTURE.md §Relay Nodes, publicly reachable nodes
// can serve as relay nodes for the network.
func (a *AutoNATService) ShouldProvideRelay() bool {
	return a.Reachability() == ReachabilityPublic
}

// WaitForReachability blocks until reachability is determined or context is cancelled.
func (a *AutoNATService) WaitForReachability(ctx context.Context) (Reachability, error) {
	// Check if already known
	if reach := a.Reachability(); reach != ReachabilityUnknown {
		return reach, nil
	}

	// Subscribe and wait
	ch := a.Subscribe()
	defer a.Unsubscribe(ch)

	select {
	case <-ctx.Done():
		return ReachabilityUnknown, ctx.Err()
	case reach := <-ch:
		return reach, nil
	}
}
