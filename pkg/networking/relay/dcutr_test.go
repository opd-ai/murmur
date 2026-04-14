package relay

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHolePunchResult_String(t *testing.T) {
	tests := []struct {
		r    HolePunchResult
		want string
	}{
		{HolePunchUnknown, "unknown"},
		{HolePunchSuccess, "success"},
		{HolePunchFailed, "failed"},
		{HolePunchTimeout, "timeout"},
		{HolePunchResult(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.r.String())
	}
}

func TestNewDCUtRService(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableHolePunching(),
	)
	require.NoError(t, err)
	defer h.Close()

	autoNAT, err := NewAutoNATService(h)
	require.NoError(t, err)
	defer autoNAT.Stop()

	dcutr := NewDCUtRService(h, autoNAT)
	require.NotNil(t, dcutr)

	assert.Equal(t, h, dcutr.host)
	assert.Equal(t, autoNAT, dcutr.autoNAT)
}

func TestDCUtRService_StartStop(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableHolePunching(),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	require.NotNil(t, dcutr)

	dcutr.Start()
	time.Sleep(10 * time.Millisecond)
	dcutr.Stop()
}

func TestDCUtRService_Subscribe(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	ch := dcutr.Subscribe()
	require.NotNil(t, ch)

	dcutr.Unsubscribe(ch)
}

func TestDCUtRService_TryHolePunch_NoService(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	// Without hole punch service, should fail
	ctx := context.Background()
	result, err := dcutr.TryHolePunch(ctx, "QmUnknownPeer")
	assert.Equal(t, HolePunchFailed, result)
	assert.Error(t, err)
}

func TestDCUtRService_TryHolePunch_InProgress(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	// Simulate in-progress
	peerID := peer.ID("QmTestPeer")
	dcutr.mu.Lock()
	dcutr.inProgress[peerID] = true
	dcutr.mu.Unlock()

	ctx := context.Background()
	result, err := dcutr.TryHolePunch(ctx, peerID)
	assert.Equal(t, HolePunchUnknown, result)
	assert.NoError(t, err)
}

func TestDCUtRService_GetResult(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	peerID := peer.ID("QmTestPeer")

	// Initially unknown
	assert.Equal(t, HolePunchUnknown, dcutr.GetResult(peerID))

	// Record a result
	dcutr.recordResult(peerID, HolePunchSuccess)
	assert.Equal(t, HolePunchSuccess, dcutr.GetResult(peerID))
}

func TestDCUtRService_RecordResult_NotifiesListeners(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	ch := dcutr.Subscribe()
	peerID := peer.ID("QmTestPeer")

	// Record result
	dcutr.recordResult(peerID, HolePunchSuccess)

	// Should receive event
	select {
	case evt := <-ch:
		assert.Equal(t, peerID, evt.Peer)
		assert.Equal(t, HolePunchSuccess, evt.Result)
		assert.True(t, evt.Direct)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected hole punch event")
	}
}

func TestContainsCircuit(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"/ip4/1.2.3.4/tcp/4001/p2p/QmRelay/p2p-circuit/p2p/QmTarget", true},
		{"/ip4/1.2.3.4/tcp/4001/p2p/QmDirect", false},
		{"/ip4/1.2.3.4/tcp/4001", false},
		{"/p2p-circuit/", true},
		{"", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, containsCircuit(tt.addr), "addr: %s", tt.addr)
	}
}

func TestContains(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("hello world", "hello"))
	assert.False(t, contains("hello world", "xyz"))
	assert.False(t, contains("hi", "hello"))
	assert.True(t, contains("hello", "hello"))
}

func TestDCUtRService_Constants(t *testing.T) {
	assert.Equal(t, 30*time.Second, DCUtRTimeout)
	assert.Equal(t, 3, DCUtRRetries)
}

func TestDCUtRService_SetHolePunchService(t *testing.T) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.EnableHolePunching(),
	)
	require.NoError(t, err)
	defer h.Close()

	dcutr := NewDCUtRService(h, nil)
	defer dcutr.Stop()

	// Initially nil
	assert.Nil(t, dcutr.hpService)

	// Can set (we don't have a real service to set, just nil is fine for API test)
	dcutr.SetHolePunchService(nil)
	assert.Nil(t, dcutr.hpService)
}
