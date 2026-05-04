// Package discovery provides tests for resolver chain.
package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResolver is a test resolver that returns configured results.
type mockResolver struct {
	name  string
	peers []peer.AddrInfo
	err   error
	delay time.Duration
}

func (m *mockResolver) Resolve(ctx context.Context) ([]peer.AddrInfo, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.peers, m.err
}

func (m *mockResolver) Name() string {
	return m.name
}

func TestResolverChain_EmptyResolvers(t *testing.T) {
	chain := NewResolverChain(nil)
	peers, err := chain.Resolve(context.Background())
	assert.Error(t, err)
	assert.Empty(t, peers)
}

func TestResolverChain_UserPeersOnly(t *testing.T) {
	userPeers := []peer.AddrInfo{
		{ID: "12D3KooWTest1"},
	}
	chain := NewResolverChain(userPeers)
	peers, err := chain.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 1)
	assert.Equal(t, userPeers[0].ID, peers[0].ID)
}

func TestResolverChain_FirstResolverSucceeds(t *testing.T) {
	resolver1 := &mockResolver{
		name:  "resolver1",
		peers: []peer.AddrInfo{{ID: "12D3KooWTest1"}},
	}
	resolver2 := &mockResolver{
		name:  "resolver2",
		peers: []peer.AddrInfo{{ID: "12D3KooWTest2"}},
	}

	chain := NewResolverChain(nil, resolver1, resolver2)
	peers, err := chain.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 1)
	assert.Equal(t, peer.ID("12D3KooWTest1"), peers[0].ID)
}

func TestResolverChain_FallbackToSecond(t *testing.T) {
	resolver1 := &mockResolver{
		name: "resolver1",
		err:  errors.New("failed"),
	}
	resolver2 := &mockResolver{
		name:  "resolver2",
		peers: []peer.AddrInfo{{ID: "12D3KooWTest2"}},
	}

	chain := NewResolverChain(nil, resolver1, resolver2)
	peers, err := chain.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 1)
	assert.Equal(t, peer.ID("12D3KooWTest2"), peers[0].ID)
}

func TestResolverChain_AllResolversFail(t *testing.T) {
	resolver1 := &mockResolver{
		name: "resolver1",
		err:  errors.New("failed1"),
	}
	resolver2 := &mockResolver{
		name: "resolver2",
		err:  errors.New("failed2"),
	}

	chain := NewResolverChain(nil, resolver1, resolver2)
	peers, err := chain.Resolve(context.Background())
	assert.Error(t, err)
	assert.Empty(t, peers)
}

func TestResolverChain_ContextCancellation(t *testing.T) {
	resolver := &mockResolver{
		name:  "slow",
		delay: 100 * time.Millisecond,
		peers: []peer.AddrInfo{{ID: "12D3KooWTest1"}},
	}

	chain := NewResolverChain(nil, resolver)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := chain.Resolve(ctx)
	assert.Error(t, err)
	// Context cancellation should propagate through the mock resolver
}

func TestResolverChain_UserPeersMergedWithResolver(t *testing.T) {
	userPeers := []peer.AddrInfo{
		{ID: "12D3KooWUser1"},
	}
	resolver := &mockResolver{
		name:  "resolver",
		peers: []peer.AddrInfo{{ID: "12D3KooWResolver1"}},
	}

	chain := NewResolverChain(userPeers, resolver)
	peers, err := chain.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 2)
}

func TestResolverChain_Deduplication(t *testing.T) {
	peerID := peer.ID("12D3KooWTest1")
	userPeers := []peer.AddrInfo{{ID: peerID}}
	resolver := &mockResolver{
		name:  "resolver",
		peers: []peer.AddrInfo{{ID: peerID}}, // Duplicate
	}

	chain := NewResolverChain(userPeers, resolver)
	peers, err := chain.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 1, "duplicate peer should be removed")
}

func TestStaticResolver_EmptyList(t *testing.T) {
	resolver := NewStaticResolver(nil)
	peers, err := resolver.Resolve(context.Background())
	assert.Error(t, err)
	assert.Empty(t, peers)
}

func TestStaticResolver_ReturnsPeers(t *testing.T) {
	expectedPeers := []peer.AddrInfo{
		{ID: "12D3KooWTest1"},
		{ID: "12D3KooWTest2"},
	}
	resolver := NewStaticResolver(expectedPeers)
	peers, err := resolver.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedPeers, peers)
}

func TestStaticResolver_Name(t *testing.T) {
	resolver := NewStaticResolver(nil)
	assert.Equal(t, "static", resolver.Name())
}

func TestTimeoutResolver_Success(t *testing.T) {
	inner := &mockResolver{
		name:  "fast",
		delay: 10 * time.Millisecond,
		peers: []peer.AddrInfo{{ID: "12D3KooWTest1"}},
	}
	resolver := NewTimeoutResolver(inner, 100*time.Millisecond)
	peers, err := resolver.Resolve(context.Background())
	assert.NoError(t, err)
	assert.Len(t, peers, 1)
}

func TestTimeoutResolver_Timeout(t *testing.T) {
	inner := &mockResolver{
		name:  "slow",
		delay: 100 * time.Millisecond,
		peers: []peer.AddrInfo{{ID: "12D3KooWTest1"}},
	}
	resolver := NewTimeoutResolver(inner, 10*time.Millisecond)
	_, err := resolver.Resolve(context.Background())
	assert.Error(t, err)
}

func TestTimeoutResolver_Name(t *testing.T) {
	inner := &mockResolver{name: "inner"}
	resolver := NewTimeoutResolver(inner, time.Second)
	assert.Equal(t, "inner", resolver.Name())
}

func TestDeduplicatePeers(t *testing.T) {
	peers := []peer.AddrInfo{
		{ID: "12D3KooWTest1"},
		{ID: "12D3KooWTest2"},
		{ID: "12D3KooWTest1"}, // Duplicate
		{ID: "12D3KooWTest3"},
		{ID: "12D3KooWTest2"}, // Duplicate
	}

	result := deduplicatePeers(peers)
	assert.Len(t, result, 3)

	// Check uniqueness
	seen := make(map[peer.ID]bool)
	for _, p := range result {
		require.False(t, seen[p.ID], "peer ID should appear only once")
		seen[p.ID] = true
	}
}

func TestDeduplicatePeers_EmptyList(t *testing.T) {
	result := deduplicatePeers(nil)
	assert.Empty(t, result)
}

func TestDeduplicatePeers_NoDuplicates(t *testing.T) {
	peers := []peer.AddrInfo{
		{ID: "12D3KooWTest1"},
		{ID: "12D3KooWTest2"},
		{ID: "12D3KooWTest3"},
	}

	result := deduplicatePeers(peers)
	assert.Len(t, result, 3)
}
