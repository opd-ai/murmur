package mesh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/networking/transport"
)

func TestDiversityManagerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	h2, _ := transport.NewHost(ctx, cfg2)
	defer h2.Close()

	m1 := NewManager(h1.Host, 0)
	if m1.diversityMgr == nil {
		t.Fatal("diversity manager should be initialized")
	}

	m1.Start()
	defer m1.Stop()

	// Connect
	h2.Connect(ctx, h1.AddrInfo())
	time.Sleep(100 * time.Millisecond)

	// Check diversity status
	status := m1.DiversityStatus()
	if status == nil {
		t.Fatal("DiversityStatus should not be nil")
	}
	if status.UniqueRegions < 0 {
		t.Error("UniqueRegions should be >= 0")
	}

	// Check ShouldAcceptPeerFromAddrs
	addrs := h2.Host.Addrs()
	if !m1.ShouldAcceptPeerFromAddrs(addrs) {
		t.Error("should accept peer when below MinUniqueRegions")
	}
}

func TestDiversityPruning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	m1 := NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	// Create and connect more than MaxPeers hosts from same region
	hosts := make([]*transport.Host, MaxPeers+2)
	for i := 0; i < MaxPeers+2; i++ {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		cfg := transport.DefaultConfig()
		cfg.PrivateKey = priv
		cfg.EnableDHT = false
		h, _ := transport.NewHost(ctx, cfg)
		hosts[i] = h
		defer h.Close()
	}

	// Connect all hosts
	for _, h := range hosts {
		h.Connect(ctx, h1.AddrInfo())
	}
	time.Sleep(200 * time.Millisecond)

	// We should have more than MaxPeers
	if m1.PeerCount() <= MaxPeers {
		t.Skipf("Not enough peers connected (%d), skipping diversity prune test", m1.PeerCount())
	}

	// PruneLowestPriority should now check diversity first
	pruned := m1.PruneLowestPriority()
	if pruned == "" {
		t.Error("PruneLowestPriority should prune when above MaxPeers")
	}
}
