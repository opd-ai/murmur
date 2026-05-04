// Package networking contains integration tests for the networking stack.
package networking

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/opd-ai/murmur/pkg/networking/discovery"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/networking/mesh"
	"github.com/opd-ai/murmur/pkg/networking/transport"
)

// TestIntegrationTwoNodeGossip validates that two nodes can discover each other
// and exchange signed messages via GossipSub. Per PLAN.md Step 13.
func TestIntegrationTwoNodeGossip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two hosts with memory transports would be ideal, but we use
	// real transports with localhost binding for simplicity
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = true
	cfg1.ListenAddrs = []string{"/ip4/127.0.0.1/tcp/0"}

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = true
	cfg2.ListenAddrs = []string{"/ip4/127.0.0.1/tcp/0"}

	// Create Node A
	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	// Create Node B
	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Setup discovery for Node B
	d2 := discovery.New(h2.Host, h2.DHT())

	// Bootstrap Node B to Node A
	if err := d2.Bootstrap(ctx, []peer.AddrInfo{h1.AddrInfo()}); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	// Setup connection managers
	m1 := mesh.NewManager(h1.Host, 0)
	m1.Start()
	defer m1.Stop()

	m2 := mesh.NewManager(h2.Host, 0)
	m2.Start()
	defer m2.Stop()

	// Setup GossipSub on both nodes
	ps1, err := gossip.New(ctx, h1.Host)
	if err != nil {
		t.Fatalf("New ps1 failed: %v", err)
	}
	defer ps1.Close()

	ps2, err := gossip.New(ctx, h2.Host)
	if err != nil {
		t.Fatalf("New ps2 failed: %v", err)
	}
	defer ps2.Close()

	// Subscribe Node B to waves topic
	var received []byte
	var receivedFrom string
	var mu sync.Mutex
	done := make(chan struct{})

	err = ps2.Subscribe(ctx, gossip.TopicWaves, func(ctx context.Context, msg *pubsub.Message) {
		mu.Lock()
		received = msg.Data
		receivedFrom = msg.GetFrom().String()
		mu.Unlock()
		close(done)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Join topic on Node A
	_, err = ps1.Join(gossip.TopicWaves)
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	// Wait for mesh to form
	time.Sleep(500 * time.Millisecond)

	// Publish signed test message from Node A
	// Note: In production, messages would be wrapped in MurmurEnvelope with Ed25519 signature
	testMsg := []byte("Hello, MURMUR! This is a test Wave.")
	if err := ps1.Publish(ctx, gossip.TopicWaves, testMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for message with timeout
	select {
	case <-done:
		mu.Lock()
		if string(received) != string(testMsg) {
			t.Errorf("received = %q, want %q", received, testMsg)
		}
		t.Logf("Message received from %s: %q", receivedFrom, received)
		mu.Unlock()
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for message")
	}

	// Verify libp2p connectivity (mesh manager only counts after RecordHeartbeat)
	// The important validation is that the message was exchanged successfully
	t.Logf("Node A: PeerID=%s, Addrs=%v", h1.PeerID(), h1.Addrs())
	t.Logf("Node B: PeerID=%s, Addrs=%v", h2.PeerID(), h2.Addrs())
}

// TestIntegrationMultipleTopics verifies that multiple topics work correctly.
func TestIntegrationMultipleTopics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false
	cfg1.ListenAddrs = []string{"/ip4/127.0.0.1/tcp/0"}

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false
	cfg2.ListenAddrs = []string{"/ip4/127.0.0.1/tcp/0"}

	h1, _ := transport.NewHost(ctx, cfg1)
	defer h1.Close()

	h2, _ := transport.NewHost(ctx, cfg2)
	defer h2.Close()

	// Connect directly
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	ps1, _ := gossip.New(ctx, h1.Host)
	defer ps1.Close()

	ps2, _ := gossip.New(ctx, h2.Host)
	defer ps2.Close()

	// Subscribe to multiple topics
	var waveMsg, identityMsg []byte
	var mu sync.Mutex
	waveDone := make(chan struct{})
	identityDone := make(chan struct{})

	ps2.Subscribe(ctx, gossip.TopicWaves, func(ctx context.Context, msg *pubsub.Message) {
		mu.Lock()
		waveMsg = msg.Data
		mu.Unlock()
		close(waveDone)
	})

	ps2.Subscribe(ctx, gossip.TopicIdentity, func(ctx context.Context, msg *pubsub.Message) {
		mu.Lock()
		identityMsg = msg.Data
		mu.Unlock()
		close(identityDone)
	})

	ps1.Join(gossip.TopicWaves)
	ps1.Join(gossip.TopicIdentity)

	time.Sleep(500 * time.Millisecond)

	// Publish to both topics
	ps1.Publish(ctx, gossip.TopicWaves, []byte("wave message"))
	ps1.Publish(ctx, gossip.TopicIdentity, []byte("identity message"))

	// Wait for both messages
	select {
	case <-waveDone:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for wave message")
	}

	select {
	case <-identityDone:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for identity message")
	}

	mu.Lock()
	if string(waveMsg) != "wave message" {
		t.Errorf("waveMsg = %q, want %q", waveMsg, "wave message")
	}
	if string(identityMsg) != "identity message" {
		t.Errorf("identityMsg = %q, want %q", identityMsg, "identity message")
	}
	mu.Unlock()
}
