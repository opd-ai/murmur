package relay

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createAutoNATTestHost(t *testing.T) func() {
	t.Helper()
	// Return a cleanup function
	return func() {}
}

func TestReachability_String(t *testing.T) {
	tests := []struct {
		r    Reachability
		want string
	}{
		{ReachabilityUnknown, "unknown"},
		{ReachabilityPublic, "public"},
		{ReachabilityPrivate, "private"},
		{Reachability(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.r.String())
	}
}

func TestNewAutoNATService(t *testing.T) {
	// Create a test host with AutoNAT enabled
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	// Create AutoNAT service
	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	require.NotNil(t, ans)

	// Initial state should be unknown
	assert.Equal(t, ReachabilityUnknown, ans.Reachability())
	assert.False(t, ans.IsPublic())
	assert.False(t, ans.IsPrivate())

	// Stop should not error
	err = ans.Stop()
	assert.NoError(t, err)
}

func TestAutoNATService_StartStop(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)

	// Start should not block
	ans.Start()

	// Give it a moment
	time.Sleep(10 * time.Millisecond)

	// Stop should not error
	err = ans.Stop()
	assert.NoError(t, err)
}

func TestAutoNATService_Subscribe(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	defer ans.Stop()

	// Subscribe should return a channel
	ch := ans.Subscribe()
	require.NotNil(t, ch)

	// Unsubscribe should work
	ans.Unsubscribe(ch)
}

func TestAutoNATService_ReachabilityHelpers(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	defer ans.Stop()

	// Initially unknown
	assert.False(t, ans.IsPublic())
	assert.False(t, ans.IsPrivate())
	assert.True(t, ans.NeedsRelay()) // Unknown needs relay too
	assert.False(t, ans.ShouldProvideRelay())

	// Manually set to public for testing
	ans.mu.Lock()
	ans.reachability = ReachabilityPublic
	ans.mu.Unlock()

	assert.True(t, ans.IsPublic())
	assert.False(t, ans.IsPrivate())
	assert.False(t, ans.NeedsRelay())
	assert.True(t, ans.ShouldProvideRelay())

	// Set to private
	ans.mu.Lock()
	ans.reachability = ReachabilityPrivate
	ans.mu.Unlock()

	assert.False(t, ans.IsPublic())
	assert.True(t, ans.IsPrivate())
	assert.True(t, ans.NeedsRelay())
	assert.False(t, ans.ShouldProvideRelay())
}

func TestAutoNATService_WaitForReachability_AlreadyKnown(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	defer ans.Stop()

	// Set known reachability
	ans.mu.Lock()
	ans.reachability = ReachabilityPublic
	ans.mu.Unlock()

	// Should return immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	reach, err := ans.WaitForReachability(ctx)
	require.NoError(t, err)
	assert.Equal(t, ReachabilityPublic, reach)
}

func TestAutoNATService_WaitForReachability_Timeout(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	defer ans.Stop()

	// Reachability is unknown, should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	reach, err := ans.WaitForReachability(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, ReachabilityUnknown, reach)
}

func TestAutoNATService_HandleReachabilityEvent(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableNATService(),
	)
	require.NoError(t, err)
	defer h.Close()

	ans, err := NewAutoNATService(h)
	require.NoError(t, err)
	ans.Start()
	defer ans.Stop()

	// Subscribe to changes
	ch := ans.Subscribe()

	// Simulate a reachability event using the actual event type
	evt := event.EvtLocalReachabilityChanged{Reachability: network.ReachabilityPublic}
	ans.handleReachabilityEvent(evt)

	// Should receive the update
	select {
	case reach := <-ch:
		assert.Equal(t, ReachabilityPublic, reach)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected reachability update")
	}

	// Verify current state
	assert.Equal(t, ReachabilityPublic, ans.Reachability())
}

func TestAutoNATService_Constants(t *testing.T) {
	// Verify constants match spec
	assert.Equal(t, 30*time.Second, AutoNATProbeInterval)
	assert.Equal(t, 3, AutoNATMinPeers)
}
