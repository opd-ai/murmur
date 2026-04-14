package discovery

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDNSDiscovery_Constants(t *testing.T) {
	// Verify constants are set correctly per NETWORK_ARCHITECTURE.md.
	assert.Equal(t, "_murmur._tcp", MDNSServiceName)
	assert.Equal(t, 10*time.Second, MDNSDiscoveryInterval)
	assert.Equal(t, 16, MDNSPeerBufferSize)
}

func TestNewMDNSDiscovery(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	handler := func(pi peer.AddrInfo) {
		// Handler for testing
	}

	mdnsDiscovery := NewMDNSDiscovery(h, handler)
	require.NotNil(t, mdnsDiscovery)
	assert.NotNil(t, mdnsDiscovery.h)
	assert.NotNil(t, mdnsDiscovery.peers)
	assert.False(t, mdnsDiscovery.IsStarted())
}

func TestMDNSDiscovery_StartStop(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	mdnsDiscovery := NewMDNSDiscovery(h, nil)
	require.NotNil(t, mdnsDiscovery)

	ctx := context.Background()

	// Start should succeed.
	err := mdnsDiscovery.Start(ctx)
	require.NoError(t, err)
	assert.True(t, mdnsDiscovery.IsStarted())

	// Starting again should be idempotent.
	err = mdnsDiscovery.Start(ctx)
	require.NoError(t, err)
	assert.True(t, mdnsDiscovery.IsStarted())

	// Stop should succeed.
	err = mdnsDiscovery.Stop()
	require.NoError(t, err)
	assert.False(t, mdnsDiscovery.IsStarted())

	// Stopping again should be idempotent.
	err = mdnsDiscovery.Stop()
	require.NoError(t, err)
	assert.False(t, mdnsDiscovery.IsStarted())
}

func TestMDNSDiscovery_HandlePeerFound_IgnoresSelf(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	handlerCalled := false
	handler := func(pi peer.AddrInfo) {
		handlerCalled = true
	}

	mdnsDiscovery := NewMDNSDiscovery(h, handler)

	// Simulate discovering self.
	mdnsDiscovery.HandlePeerFound(peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	})

	// Handler should not be called for self-discovery.
	assert.False(t, handlerCalled)
}

func TestMDNSDiscovery_HandlePeerFound_CallsHandler(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	var discoveredPeer peer.AddrInfo
	handlerCalled := false
	handler := func(pi peer.AddrInfo) {
		handlerCalled = true
		discoveredPeer = pi
	}

	mdnsDiscovery := NewMDNSDiscovery(h, handler)

	// Generate a fake peer ID.
	fakePeerID := generateFakePeerID(t)
	fakePeer := peer.AddrInfo{
		ID: fakePeerID,
	}

	mdnsDiscovery.HandlePeerFound(fakePeer)

	assert.True(t, handlerCalled)
	assert.Equal(t, fakePeerID, discoveredPeer.ID)
}

func TestMDNSDiscovery_Peers_Channel(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	mdnsDiscovery := NewMDNSDiscovery(h, nil)

	// Generate some fake peers.
	fakePeerID1 := generateFakePeerID(t)
	fakePeerID2 := generateFakePeerID(t)

	mdnsDiscovery.HandlePeerFound(peer.AddrInfo{ID: fakePeerID1})
	mdnsDiscovery.HandlePeerFound(peer.AddrInfo{ID: fakePeerID2})

	// Should receive peers from channel.
	peers := mdnsDiscovery.Peers()

	select {
	case pi := <-peers:
		assert.Contains(t, []peer.ID{fakePeerID1, fakePeerID2}, pi.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected to receive peer from channel")
	}

	select {
	case pi := <-peers:
		assert.Contains(t, []peer.ID{fakePeerID1, fakePeerID2}, pi.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected to receive second peer from channel")
	}
}

func TestMDNSDiscovery_NilHandler(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	// Should not panic with nil handler.
	mdnsDiscovery := NewMDNSDiscovery(h, nil)

	fakePeerID := generateFakePeerID(t)
	require.NotPanics(t, func() {
		mdnsDiscovery.HandlePeerFound(peer.AddrInfo{ID: fakePeerID})
	})
}

func TestMDNSDiscovery_ChannelOverflow(t *testing.T) {
	h := createMDNSTestHost(t)
	defer h.Close()

	mdnsDiscovery := NewMDNSDiscovery(h, nil)

	// Fill the buffer and then some.
	for i := 0; i < MDNSPeerBufferSize+5; i++ {
		fakePeerID := generateFakePeerID(t)
		require.NotPanics(t, func() {
			mdnsDiscovery.HandlePeerFound(peer.AddrInfo{ID: fakePeerID})
		})
	}

	// Channel should have dropped some, but shouldn't panic.
	count := 0
	timeout := time.After(100 * time.Millisecond)
drain:
	for {
		select {
		case <-mdnsDiscovery.Peers():
			count++
		case <-timeout:
			break drain
		}
	}
	// Should have at most buffer size peers.
	assert.LessOrEqual(t, count, MDNSPeerBufferSize)
}

// Helper functions.

func createMDNSTestHost(t *testing.T) host.Host {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(t, err)

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	require.NoError(t, err)
	return h
}

func generateFakePeerID(t *testing.T) peer.ID {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	require.NoError(t, err)

	peerID, err := peer.IDFromPrivateKey(privKey)
	require.NoError(t, err)

	return peerID
}
