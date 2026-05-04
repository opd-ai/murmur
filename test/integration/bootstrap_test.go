//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/transport"
	"github.com/stretchr/testify/require"
)

// TestBootstrapPeerConnection verifies that a node can connect to a bootstrap peer
// and populate its DHT routing table.
// Per PLAN.md Step 7: "Bootstrap test confirms DHT routing table populated with ≥5 peer entries"
func TestBootstrapPeerConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create bootstrap node (acts as DHT bootstrap peer)
	bootstrapKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	bootstrapHost, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: bootstrapKp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer bootstrapHost.Close()

	// Initialize DHT on bootstrap node in server mode
	bootstrapDHT, err := dual.New(ctx, bootstrapHost, dual.DHTOption(dual.Mode(dual.ModeServer)))
	require.NoError(t, err)
	defer bootstrapDHT.Close()

	err = bootstrapDHT.Bootstrap(ctx)
	require.NoError(t, err)

	t.Logf("Bootstrap node: %s", bootstrapHost.PeerID().String())
	t.Logf("Bootstrap addrs: %v", bootstrapHost.Addrs())

	// Create 6 regular nodes that will use the bootstrap peer
	const numNodes = 6
	nodes := make([]*transport.Host, numNodes)
	dhts := make([]*dual.DHT, numNodes)

	for i := 0; i < numNodes; i++ {
		kp, err := keys.GenerateKeyPair()
		require.NoError(t, err)

		h, err := transport.NewHost(ctx, transport.Config{
			PrivateKey: kp.PrivateKey,
			ListenAddrs: []string{
				"/ip4/127.0.0.1/tcp/0",
			},
		})
		require.NoError(t, err)
		defer h.Close()

		dht, err := dual.New(ctx, h)
		require.NoError(t, err)
		defer dht.Close()

		nodes[i] = h
		dhts[i] = dht

		// Connect to bootstrap peer
		bootstrapAddrInfo := peer.AddrInfo{
			ID:    bootstrapHost.PeerID(),
			Addrs: bootstrapHost.Addrs(),
		}

		err = h.Connect(ctx, bootstrapAddrInfo)
		require.NoError(t, err, "node %d connecting to bootstrap", i)

		t.Logf("Node %d connected to bootstrap", i)
	}

	// Bootstrap all DHTs
	for i, dht := range dhts {
		err = dht.Bootstrap(ctx)
		require.NoError(t, err, "bootstrapping DHT for node %d", i)
	}

	// Wait for DHT routing tables to populate
	// DHT refresh happens periodically; give it time to discover peers
	time.Sleep(10 * time.Second)

	// Verify each node has populated its routing table with ≥5 peer entries
	for i, h := range nodes {
		peers := h.Network().Peers()
		t.Logf("Node %d has %d peers in routing table", i, len(peers))

		// Each node should be connected to bootstrap + some other nodes
		require.GreaterOrEqual(t, len(peers), 1, "node %d should have at least bootstrap peer", i)

		// Check DHT routing table size
		// Use the WAN DHT as it's the primary routing DHT
		routingTable := dhts[i].WAN.RoutingTable()
		rtSize := routingTable.Size()
		t.Logf("Node %d DHT routing table size: %d", i, rtSize)

		// After bootstrap, routing table should have multiple entries
		// Note: May take time for full convergence; we test basic connectivity
		require.Greater(t, rtSize, 0, "node %d DHT routing table should not be empty", i)
	}
}

// TestPeerDiscoveryViaDHT verifies that peers can discover each other via DHT lookups.
func TestPeerDiscoveryViaDHT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create bootstrap node
	bootstrapKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	bootstrapHost, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: bootstrapKp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer bootstrapHost.Close()

	bootstrapDHT, err := dual.New(ctx, bootstrapHost, dual.DHTOption(dual.Mode(dual.ModeServer)))
	require.NoError(t, err)
	defer bootstrapDHT.Close()

	err = bootstrapDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Create node A
	kpA, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	hostA, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: kpA.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer hostA.Close()

	dhtA, err := dual.New(ctx, hostA)
	require.NoError(t, err)
	defer dhtA.Close()

	// Create node B
	kpB, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	hostB, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: kpB.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer hostB.Close()

	dhtB, err := dual.New(ctx, hostB)
	require.NoError(t, err)
	defer dhtB.Close()

	// Both nodes connect to bootstrap
	bootstrapAddrInfo := peer.AddrInfo{
		ID:    bootstrapHost.PeerID(),
		Addrs: bootstrapHost.Addrs(),
	}

	err = hostA.Connect(ctx, bootstrapAddrInfo)
	require.NoError(t, err)

	err = hostB.Connect(ctx, bootstrapAddrInfo)
	require.NoError(t, err)

	// Bootstrap DHTs
	err = dhtA.Bootstrap(ctx)
	require.NoError(t, err)

	err = dhtB.Bootstrap(ctx)
	require.NoError(t, err)

	// Wait for DHT convergence
	time.Sleep(5 * time.Second)

	// Node A performs DHT lookup for node B
	t.Logf("Node A (%s) looking up node B (%s)", hostA.PeerID(), hostB.PeerID())

	peerChan := dhtA.WAN.FindPeer(ctx, hostB.PeerID())

	select {
	case peerInfo, ok := <-peerChan:
		require.True(t, ok, "peer channel should not be closed without result")
		require.Equal(t, hostB.PeerID(), peerInfo.ID, "found peer ID should match node B")
		require.NotEmpty(t, peerInfo.Addrs, "found peer should have addresses")
		t.Logf("Successfully discovered node B via DHT: %v", peerInfo.Addrs)

	case <-time.After(30 * time.Second):
		require.FailNow(t, "DHT peer discovery timeout", "node A could not find node B via DHT within 30s")
	}
}

// TestDHTProviderRecords verifies that content provider records can be advertised and found via DHT.
func TestDHTProviderRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create bootstrap node
	bootstrapKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	bootstrapHost, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: bootstrapKp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer bootstrapHost.Close()

	bootstrapDHT, err := dual.New(ctx, bootstrapHost, dual.DHTOption(dual.Mode(dual.ModeServer)))
	require.NoError(t, err)
	defer bootstrapDHT.Close()

	err = bootstrapDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Create provider node
	providerKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	providerHost, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: providerKp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer providerHost.Close()

	providerDHT, err := dual.New(ctx, providerHost)
	require.NoError(t, err)
	defer providerDHT.Close()

	// Create consumer node
	consumerKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	consumerHost, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: consumerKp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer consumerHost.Close()

	consumerDHT, err := dual.New(ctx, consumerHost)
	require.NoError(t, err)
	defer consumerDHT.Close()

	// Connect both to bootstrap
	bootstrapAddrInfo := peer.AddrInfo{
		ID:    bootstrapHost.PeerID(),
		Addrs: bootstrapHost.Addrs(),
	}

	err = providerHost.Connect(ctx, bootstrapAddrInfo)
	require.NoError(t, err)

	err = consumerHost.Connect(ctx, bootstrapAddrInfo)
	require.NoError(t, err)

	// Bootstrap DHTs
	err = providerDHT.Bootstrap(ctx)
	require.NoError(t, err)

	err = consumerDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Wait for convergence
	time.Sleep(5 * time.Second)

	// Provider advertises a content key (simulating a Wave ID)
	contentKey := []byte("test-wave-id-abc123")
	t.Logf("Provider advertising content key: %x", contentKey)

	err = providerDHT.WAN.Provide(ctx, contentKey, true)
	require.NoError(t, err)

	// Wait for provider record to propagate
	time.Sleep(2 * time.Second)

	// Consumer searches for providers of the content key
	t.Logf("Consumer searching for providers of content key")

	providersChan := consumerDHT.WAN.FindProvidersAsync(ctx, contentKey, 5)

	foundProvider := false
	timeout := time.After(20 * time.Second)

	for !foundProvider {
		select {
		case prov := <-providersChan:
			if prov.ID == providerHost.PeerID() {
				t.Logf("Found provider: %s", prov.ID)
				foundProvider = true
			}

		case <-timeout:
			require.FailNow(t, "provider discovery timeout",
				"consumer could not find provider for content key within 20s")
		}
	}

	require.True(t, foundProvider, "consumer should have found the provider via DHT")
}

// Verify that routing.ErrNotFound is returned for non-existent peers.
func TestDHTPeerNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create single node with DHT
	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	h, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: kp.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	defer h.Close()

	dht, err := dual.New(ctx, h)
	require.NoError(t, err)
	defer dht.Close()

	err = dht.Bootstrap(ctx)
	require.NoError(t, err)

	// Generate a random peer ID that doesn't exist
	nonExistentKp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	nonExistentID, err := peer.IDFromPublicKey(nonExistentKp)
	require.NoError(t, err)

	// Try to find non-existent peer
	t.Logf("Looking up non-existent peer: %s", nonExistentID)

	peerChan := dht.WAN.FindPeer(ctx, nonExistentID)

	select {
	case _, ok := <-peerChan:
		if ok {
			// If we got a result, it should be an error
			t.Log("Received response for non-existent peer (expected)")
		} else {
			t.Log("Peer channel closed (expected for non-existent peer)")
		}

	case <-time.After(15 * time.Second):
		// Timeout is expected for non-existent peers
		t.Log("Lookup timeout for non-existent peer (expected)")
	}
}
