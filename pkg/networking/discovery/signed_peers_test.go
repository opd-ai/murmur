// Package discovery provides tests for signed peer list functionality.
package discovery

import (
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignedPeerList_SignAndVerify(t *testing.T) {
	// Generate keypair
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create peer list
	peerList := &SignedPeerList{
		Peers: []PeerEntry{
			{
				ID:    "12D3KooWTest1",
				Addrs: []string{"/ip4/127.0.0.1/tcp/4001"},
				Seen:  time.Now().Unix(),
			},
		},
	}

	// Sign
	err = peerList.Sign(privKey)
	require.NoError(t, err)
	assert.NotEmpty(t, peerList.Signature)
	assert.Equal(t, 1, peerList.Version)

	// Verify
	err = peerList.Verify(pubKey)
	assert.NoError(t, err)
}

func TestSignedPeerList_SignInvalidKey(t *testing.T) {
	peerList := &SignedPeerList{}
	invalidKey := make([]byte, 10) // Wrong size

	err := peerList.Sign(invalidKey)
	assert.Error(t, err)
}

func TestSignedPeerList_VerifyInvalidKey(t *testing.T) {
	peerList := &SignedPeerList{
		Signature: "test",
	}
	invalidKey := make([]byte, 10) // Wrong size

	err := peerList.Verify(invalidKey)
	assert.Error(t, err)
}

func TestSignedPeerList_VerifyWrongKey(t *testing.T) {
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Different key for verification
	wrongPubKey, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	peerList := &SignedPeerList{
		Peers: []PeerEntry{{ID: "12D3KooWTest1", Addrs: []string{"/ip4/127.0.0.1/tcp/4001"}, Seen: time.Now().Unix()}},
	}

	err = peerList.Sign(privKey)
	require.NoError(t, err)

	err = peerList.Verify(wrongPubKey)
	assert.Error(t, err, "verification should fail with wrong key")
}

func TestSignedPeerList_ToPeerAddrInfos(t *testing.T) {
	peerList := &SignedPeerList{
		Peers: []PeerEntry{
			{
				ID:    "12D3KooWEBfGja67JjFmartifhUpXFj2KL13h7E8zAMxV6fHc4Yk",
				Addrs: []string{"/ip4/127.0.0.1/tcp/4001"},
				Seen:  time.Now().Unix(),
			},
		},
	}

	peers, err := peerList.ToPeerAddrInfos()
	require.NoError(t, err)
	require.Len(t, peers, 1)

	assert.Equal(t, "12D3KooWEBfGja67JjFmartifhUpXFj2KL13h7E8zAMxV6fHc4Yk", peers[0].ID.String())
	require.Len(t, peers[0].Addrs, 1)
}

func TestSignedPeerList_ToPeerAddrInfos_InvalidPeerID(t *testing.T) {
	peerList := &SignedPeerList{
		Peers: []PeerEntry{
			{
				ID:    "invalid-peer-id",
				Addrs: []string{"/ip4/127.0.0.1/tcp/4001"},
				Seen:  time.Now().Unix(),
			},
		},
	}

	peers, err := peerList.ToPeerAddrInfos()
	require.NoError(t, err)
	assert.Empty(t, peers, "invalid peer ID should be skipped")
}

func TestSignedPeerList_ToPeerAddrInfos_InvalidMultiaddr(t *testing.T) {
	peerList := &SignedPeerList{
		Peers: []PeerEntry{
			{
				ID:    "12D3KooWEBfGja67JjFmartifhUpXFj2KL13h7E8zAMxV6fHc4Yk",
				Addrs: []string{"invalid-multiaddr"},
				Seen:  time.Now().Unix(),
			},
		},
	}

	peers, err := peerList.ToPeerAddrInfos()
	require.NoError(t, err)
	assert.Empty(t, peers, "peer with no valid addrs should be skipped")
}

func TestFromPeerAddrInfos(t *testing.T) {
	// Create test peer
	peerID, err := peer.Decode("12D3KooWEBfGja67JjFmartifhUpXFj2KL13h7E8zAMxV6fHc4Yk")
	require.NoError(t, err)

	ma, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	require.NoError(t, err)

	peers := []peer.AddrInfo{
		{
			ID:    peerID,
			Addrs: []multiaddr.Multiaddr{ma},
		},
	}

	peerList := FromPeerAddrInfos(peers)
	require.Len(t, peerList.Peers, 1)
	assert.Equal(t, peerID.String(), peerList.Peers[0].ID)
	assert.Equal(t, []string{"/ip4/127.0.0.1/tcp/4001"}, peerList.Peers[0].Addrs)
}

func TestFromPeerAddrInfos_EmptyID(t *testing.T) {
	peers := []peer.AddrInfo{
		{ID: ""}, // Empty ID should be skipped
	}

	peerList := FromPeerAddrInfos(peers)
	assert.Empty(t, peerList.Peers)
}

func TestFromPeerAddrInfos_NoAddrs(t *testing.T) {
	peerID, err := peer.Decode("12D3KooWEBfGja67JjFmartifhUpXFj2KL13h7E8zAMxV6fHc4Yk")
	require.NoError(t, err)

	peers := []peer.AddrInfo{
		{ID: peerID, Addrs: nil}, // No addrs should be skipped
	}

	peerList := FromPeerAddrInfos(peers)
	assert.Empty(t, peerList.Peers)
}

func TestSignedPeerList_PruneStale(t *testing.T) {
	now := time.Now().Unix()
	peerList := &SignedPeerList{
		Peers: []PeerEntry{
			{ID: "peer1", Addrs: []string{"/ip4/1.1.1.1/tcp/4001"}, Seen: now},
			{ID: "peer2", Addrs: []string{"/ip4/2.2.2.2/tcp/4001"}, Seen: now - 3600},  // 1 hour old
			{ID: "peer3", Addrs: []string{"/ip4/3.3.3.3/tcp/4001"}, Seen: now - 7200},  // 2 hours old
			{ID: "peer4", Addrs: []string{"/ip4/4.4.4.4/tcp/4001"}, Seen: now - 90000}, // 25 hours old
		},
	}

	peerList.PruneStale(24 * time.Hour)

	require.Len(t, peerList.Peers, 3, "peer4 should be pruned")
	assert.Equal(t, "peer1", peerList.Peers[0].ID)
	assert.Equal(t, "peer2", peerList.Peers[1].ID)
	assert.Equal(t, "peer3", peerList.Peers[2].ID)
}

func TestSignedPeerList_JSONRoundTrip(t *testing.T) {
	original := &SignedPeerList{
		Version:   1,
		Timestamp: time.Now().Unix(),
		Peers: []PeerEntry{
			{
				ID:    "12D3KooWTest1",
				Addrs: []string{"/ip4/127.0.0.1/tcp/4001"},
				Seen:  time.Now().Unix(),
			},
		},
		Signature: "test-signature",
		SignedBy:  "test-key",
	}

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded SignedPeerList
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)
	assert.Equal(t, original.Signature, decoded.Signature)
	assert.Equal(t, original.SignedBy, decoded.SignedBy)
	require.Len(t, decoded.Peers, 1)
	assert.Equal(t, original.Peers[0].ID, decoded.Peers[0].ID)
}

func TestEncodeDecodeBase64(t *testing.T) {
	testCases := [][]byte{
		{0x00},
		{0xFF},
		{0x01, 0x02, 0x03, 0x04},
		make([]byte, 32), // 32 zero bytes
	}

	for _, tc := range testCases {
		encoded := encodeBase64(tc)
		assert.NotEmpty(t, encoded)

		decoded, err := decodeBase64(encoded)
		require.NoError(t, err)
		assert.Equal(t, tc, decoded, "round-trip should preserve data")
	}
}

func TestDecodeBase64_InvalidHex(t *testing.T) {
	_, err := decodeBase64("zz") // Invalid hex characters
	assert.Error(t, err)
}

func TestDecodeBase64_OddLength(t *testing.T) {
	_, err := decodeBase64("abc") // Odd length
	assert.Error(t, err)
}
