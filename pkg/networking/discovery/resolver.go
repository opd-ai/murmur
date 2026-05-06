// Package discovery provides bootstrap and peer discovery mechanisms.
// Per PLAN.md "Bootstrap Strategy: Zero-Infrastructure Peer Discovery".
package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// BootstrapResolver resolves bootstrap peers from a specific source.
// Multiple resolvers can be chained together with fallback behavior.
type BootstrapResolver interface {
	// Resolve attempts to discover bootstrap peers within the given context.
	// Returns a list of peer addresses on success, or an error if resolution fails.
	Resolve(ctx context.Context) ([]peer.AddrInfo, error)

	// Name returns a human-readable name for this resolver (for logging).
	Name() string
}

// ResolverChain tries multiple resolvers in priority order, returning on first success.
// Per PLAN.md §2 "Bootstrap Flow": resolvers execute in short-circuit order.
type ResolverChain struct {
	resolvers []BootstrapResolver
	userPeers []peer.AddrInfo // User-supplied peers (always merged first)
}

// NewResolverChain creates a new resolver chain.
// User-supplied peers are always merged in first before any resolver runs.
func NewResolverChain(userPeers []peer.AddrInfo, resolvers ...BootstrapResolver) *ResolverChain {
	return &ResolverChain{
		resolvers: resolvers,
		userPeers: userPeers,
	}
}

// Resolve attempts bootstrap using each resolver in order until one succeeds.
// Returns all discovered peers (user-supplied + resolver results).
// Per PLAN.md: "the first one that yields ≥1 successful connection halts the chain."
func (rc *ResolverChain) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	allPeers := rc.initializeWithUserPeers()

	for _, resolver := range rc.resolvers {
		if ctx.Err() != nil {
			return allPeers, ctx.Err()
		}

		if peers := rc.tryResolver(ctx, resolver); peers != nil {
			allPeers = append(allPeers, peers...)
			return deduplicatePeers(allPeers), nil
		}
	}

	return rc.finalizeResult(allPeers)
}

func (rc *ResolverChain) initializeWithUserPeers() []peer.AddrInfo {
	allPeers := make([]peer.AddrInfo, len(rc.userPeers))
	copy(allPeers, rc.userPeers)
	return allPeers
}

func (rc *ResolverChain) tryResolver(ctx context.Context, resolver BootstrapResolver) []peer.AddrInfo {
	peers, err := resolver.Resolve(ctx)
	if err == nil && len(peers) > 0 {
		return peers
	}
	return nil
}

func (rc *ResolverChain) finalizeResult(allPeers []peer.AddrInfo) ([]peer.AddrInfo, error) {
	if len(allPeers) > 0 {
		return allPeers, nil
	}
	return nil, fmt.Errorf("all resolvers failed")
}

// deduplicatePeers removes duplicate peer IDs from the list.
func deduplicatePeers(peers []peer.AddrInfo) []peer.AddrInfo {
	seen := make(map[peer.ID]bool, len(peers))
	result := make([]peer.AddrInfo, 0, len(peers))

	for _, p := range peers {
		if !seen[p.ID] {
			seen[p.ID] = true
			result = append(result, p)
		}
	}
	return result
}

// StaticResolver resolves from a static list of peers (used for fallback).
type StaticResolver struct {
	peers []peer.AddrInfo
}

// NewStaticResolver creates a resolver that returns a fixed peer list.
func NewStaticResolver(peers []peer.AddrInfo) *StaticResolver {
	return &StaticResolver{peers: peers}
}

// Resolve returns the static peer list.
func (s *StaticResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	if len(s.peers) == 0 {
		return nil, fmt.Errorf("static resolver: empty peer list")
	}
	return s.peers, nil
}

// Name returns the resolver name.
func (s *StaticResolver) Name() string {
	return "static"
}

// TimeoutResolver wraps another resolver with a timeout.
type TimeoutResolver struct {
	resolver BootstrapResolver
	timeout  time.Duration
}

// NewTimeoutResolver wraps a resolver with a timeout.
func NewTimeoutResolver(resolver BootstrapResolver, timeout time.Duration) *TimeoutResolver {
	return &TimeoutResolver{
		resolver: resolver,
		timeout:  timeout,
	}
}

// Resolve runs the wrapped resolver with a timeout.
func (t *TimeoutResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	return t.resolver.Resolve(ctx)
}

// Name returns the wrapped resolver name.
func (t *TimeoutResolver) Name() string {
	return t.resolver.Name()
}
