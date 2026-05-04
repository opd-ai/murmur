package discovery

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/opd-ai/murmur/pkg/networking/transport"
)

func TestDiscoveryNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d := New(h.Host, h.DHT())
	if d == nil {
		t.Error("New returned nil")
	}
}

func TestDiscoveryBootstrapEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d := New(h.Host, h.DHT())

	// Bootstrap with empty peer list should succeed
	if err := d.Bootstrap(ctx, nil); err != nil {
		t.Errorf("Bootstrap with empty peers failed: %v", err)
	}
}

func TestDiscoveryBootstrapTwoNodes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create first node
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = true

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	// Create second node
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = true

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Create discovery for second node
	d2 := New(h2.Host, h2.DHT())

	// Bootstrap h2 to h1
	bootstrapPeers := []peer.AddrInfo{h1.AddrInfo()}
	if err := d2.Bootstrap(ctx, bootstrapPeers); err != nil {
		t.Errorf("Bootstrap failed: %v", err)
	}

	// Verify connection was established
	conns := h2.Network().ConnsToPeer(h1.PeerID())
	if len(conns) == 0 {
		t.Error("h2 has no connections to h1 after bootstrap")
	}
}

func TestDiscoveryFindPeersNoDHT(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false // No DHT

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d := New(h.Host, nil) // No DHT

	_, err = d.FindPeers(ctx)
	if err == nil {
		t.Error("FindPeers should fail without DHT")
	}
}

func TestDiscoveryNumPeers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d := New(h.Host, h.DHT())

	// Initially no peers
	if n := d.NumPeers(); n != 0 {
		t.Errorf("NumPeers = %d, want 0", n)
	}
}

func TestDiscoveryRoutingTableNil(t *testing.T) {
	ctx := context.Background()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d := New(h.Host, nil)
	if rt := d.RoutingTable(); rt != nil {
		t.Error("RoutingTable should be nil without DHT")
	}
}

// TestRoutingTableRefreshInterval documents the expected refresh interval.
func TestRoutingTableRefreshInterval(t *testing.T) {
	if RoutingTableRefreshInterval != 10*time.Minute {
		t.Errorf("RoutingTableRefreshInterval = %v, want 10m", RoutingTableRefreshInterval)
	}
}

// dummyDHT creates a DHT for testing. Since we can't easily create a real DHT
// without a lot of setup, we test the nil cases thoroughly.
func createTestDHTHost(t *testing.T, ctx context.Context) (*transport.Host, *dht.IpfsDHT) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}

	return h, h.DHT()
}

func TestDiscoveryBootstrapFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create bootstrap node
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = true

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	// Create client node
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = true

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Create discovery for client node
	d2 := New(h2.Host, h2.DHT())

	// Configure fallback resolver with h1's address
	fallbackResolver := NewStaticResolver([]peer.AddrInfo{h1.AddrInfo()})
	resolverChain := NewResolverChain(nil, fallbackResolver)
	d2.SetFallbackResolvers(resolverChain)

	// Bootstrap with invalid peers - should fall back to resolver
	invalidPeer := peer.AddrInfo{
		ID: "12D3KooWInvalidPeerID",
	}
	if err := d2.Bootstrap(ctx, []peer.AddrInfo{invalidPeer}); err != nil {
		// Error expected since invalid peer will fail, but fallback should work
		t.Logf("Bootstrap returned error (expected): %v", err)
	}

	// Wait for fallback resolution to complete
	time.Sleep(2 * time.Second)

	// Verify connection was established via fallback
	conns := h2.Network().ConnsToPeer(h1.PeerID())
	if len(conns) == 0 {
		t.Error("h2 has no connections to h1 after bootstrap with fallback")
	} else {
		t.Logf("Success: h2 connected to h1 via fallback resolver")
	}
}

func TestDiscoveryBootstrapNoFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create node
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = true

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	// Create discovery without fallback
	d := New(h.Host, h.DHT())

	// Bootstrap with invalid peers - should fail
	invalidPeer := peer.AddrInfo{
		ID: "12D3KooWInvalidPeerID",
	}
	err = d.Bootstrap(ctx, []peer.AddrInfo{invalidPeer})
	if err == nil {
		t.Error("Bootstrap should have failed with invalid peer and no fallback")
	} else {
		t.Logf("Bootstrap correctly failed: %v", err)
	}
}
