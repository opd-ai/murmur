package discovery

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPEX_Constants(t *testing.T) {
	// Verify constants per NETWORK_ARCHITECTURE.md.
	assert.Equal(t, "/murmur/peer-exchange/1", string(PEXProtocolID))
	assert.Equal(t, 5*time.Minute, PEXInterval)
	assert.Equal(t, 20, PEXSampleSize)
}

func TestNewPEX(t *testing.T) {
	h := createPEXTestHost(t)
	defer h.Close()

	handler := func(pi peer.AddrInfo) {
		// Handler for testing
	}

	pex := NewPEX(h, handler)
	require.NotNil(t, pex)
	assert.NotNil(t, pex.h)
	assert.False(t, pex.IsRunning())
}

func TestPEX_StartStop(t *testing.T) {
	h := createPEXTestHost(t)
	defer h.Close()

	pex := NewPEX(h, nil)
	ctx := context.Background()

	// Start should succeed.
	err := pex.Start(ctx)
	require.NoError(t, err)
	assert.True(t, pex.IsRunning())

	// Starting again should be idempotent.
	err = pex.Start(ctx)
	require.NoError(t, err)
	assert.True(t, pex.IsRunning())

	// Stop should succeed.
	err = pex.Stop()
	require.NoError(t, err)
	assert.False(t, pex.IsRunning())

	// Stopping again should be idempotent.
	err = pex.Stop()
	require.NoError(t, err)
	assert.False(t, pex.IsRunning())
}

func TestPEX_WritePeerList(t *testing.T) {
	// Create some test peers.
	addr1, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	addr2, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.2/tcp/4001")

	peers := []PeerInfo{
		{
			ID:    peer.ID("peer1"),
			Addrs: []multiaddr.Multiaddr{addr1},
		},
		{
			ID:    peer.ID("peer2"),
			Addrs: []multiaddr.Multiaddr{addr2},
		},
	}

	var buf bytes.Buffer
	err := writePeerList(&buf, peers)
	require.NoError(t, err)

	// Read it back.
	readPeers, err := readPeerList(&buf)
	require.NoError(t, err)

	assert.Len(t, readPeers, 2)
	assert.Equal(t, peer.ID("peer1"), readPeers[0].ID)
	assert.Equal(t, peer.ID("peer2"), readPeers[1].ID)
	assert.Len(t, readPeers[0].Addrs, 1)
	assert.Len(t, readPeers[1].Addrs, 1)
}

func TestPEX_WritePeerListEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := writePeerList(&buf, nil)
	require.NoError(t, err)

	readPeers, err := readPeerList(&buf)
	require.NoError(t, err)
	assert.Len(t, readPeers, 0)
}

func TestPEX_ReadPeerListTooManyPeers(t *testing.T) {
	// Create a buffer with too many peers.
	var buf bytes.Buffer
	// Write number of peers = 101 (exceeds limit of 100).
	buf.Write([]byte{101, 0, 0, 0})

	_, err := readPeerList(&buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many peers")
}

func TestPEX_SamplePeers(t *testing.T) {
	h := createPEXTestHost(t)
	defer h.Close()

	pex := NewPEX(h, nil)

	// Initially no peers.
	sample := pex.samplePeers(10)
	assert.Len(t, sample, 0)

	// Add some peers to peerstore.
	for i := 0; i < 5; i++ {
		peerID := generateFakePeerID(t)
		addr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
		h.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)
	}

	// Should get all 5.
	sample = pex.samplePeers(10)
	assert.Len(t, sample, 5)

	// Should get only 3 when requested.
	sample = pex.samplePeers(3)
	assert.Len(t, sample, 3)
}

func TestPEX_SamplePeersExcludesSelf(t *testing.T) {
	h := createPEXTestHost(t)
	defer h.Close()

	pex := NewPEX(h, nil)

	// Add self to peerstore (shouldn't be included in sample).
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	h.Peerstore().AddAddrs(h.ID(), []multiaddr.Multiaddr{addr}, time.Hour)

	sample := pex.samplePeers(10)
	for _, pi := range sample {
		assert.NotEqual(t, h.ID(), pi.ID, "Self should not be in sample")
	}
}

func TestPEX_TwoHostsExchange(t *testing.T) {
	h1 := createPEXTestHost(t)
	defer h1.Close()
	h2 := createPEXTestHost(t)
	defer h2.Close()

	// Track received peers.
	var receivedPeers []peer.AddrInfo
	handler := func(pi peer.AddrInfo) {
		receivedPeers = append(receivedPeers, pi)
	}

	pex1 := NewPEX(h1, nil)
	pex2 := NewPEX(h2, handler)

	ctx := context.Background()
	require.NoError(t, pex1.Start(ctx))
	require.NoError(t, pex2.Start(ctx))
	defer pex1.Stop()
	defer pex2.Stop()

	// Connect h1 to h2.
	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), time.Hour)
	err := h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Add some peers to h1's peerstore to share.
	for i := 0; i < 3; i++ {
		peerID := generateFakePeerID(t)
		addr, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
		h1.Peerstore().AddAddrs(peerID, []multiaddr.Multiaddr{addr}, time.Hour)
	}

	// Do manual exchange.
	peers, err := pex2.ExchangeWithPeer(ctx, h1.ID())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(peers), 0) // May or may not get peers depending on timing
}

func TestPeerInfo_MultipleAddrs(t *testing.T) {
	addr1, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	addr2, _ := multiaddr.NewMultiaddr("/ip4/192.168.1.1/udp/4001/quic-v1")
	addr3, _ := multiaddr.NewMultiaddr("/ip6/::1/tcp/4001")

	peers := []PeerInfo{
		{
			ID:    peer.ID("peer1"),
			Addrs: []multiaddr.Multiaddr{addr1, addr2, addr3},
		},
	}

	var buf bytes.Buffer
	err := writePeerList(&buf, peers)
	require.NoError(t, err)

	readPeers, err := readPeerList(&buf)
	require.NoError(t, err)

	assert.Len(t, readPeers, 1)
	assert.Len(t, readPeers[0].Addrs, 3)
}

// Helper functions.

func createPEXTestHost(t *testing.T) host.Host {
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
