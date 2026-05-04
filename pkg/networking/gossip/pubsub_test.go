package gossip

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/opd-ai/murmur/pkg/networking/transport"
)

func TestNewPubSub(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h.Host)
	if err != nil {
		t.Fatalf("New PubSub failed: %v", err)
	}
	defer ps.Close()

	if ps == nil {
		t.Error("New returned nil PubSub")
	}
}

func TestJoinTopic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h.Host)
	if err != nil {
		t.Fatalf("New PubSub failed: %v", err)
	}
	defer ps.Close()

	// Join waves topic
	topic, err := ps.Join(TopicWaves)
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}
	if topic == nil {
		t.Error("Join returned nil topic")
	}

	// Verify topic is in list
	topics := ps.Topics()
	found := false
	for _, name := range topics {
		if name == TopicWaves {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("TopicWaves not found in Topics(): %v", topics)
	}
}

func TestJoinTopicIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h.Host)
	if err != nil {
		t.Fatalf("New PubSub failed: %v", err)
	}
	defer ps.Close()

	// Join same topic twice
	topic1, _ := ps.Join(TopicWaves)
	topic2, _ := ps.Join(TopicWaves)

	if topic1 != topic2 {
		t.Error("Join should return same topic handle for same topic name")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two nodes
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Connect h2 to h1
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Create pubsub for both
	ps1, err := New(ctx, h1.Host)
	if err != nil {
		t.Fatalf("New ps1 failed: %v", err)
	}
	defer ps1.Close()

	ps2, err := New(ctx, h2.Host)
	if err != nil {
		t.Fatalf("New ps2 failed: %v", err)
	}
	defer ps2.Close()

	// Subscribe h2 to topic
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{})

	err = ps2.Subscribe(ctx, TopicWaves, func(ctx context.Context, msg *pubsub.Message) {
		mu.Lock()
		received = msg.Data
		mu.Unlock()
		close(done)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Join topic on h1 (publisher must also join)
	_, err = ps1.Join(TopicWaves)
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	// Wait for mesh to form
	time.Sleep(500 * time.Millisecond)

	// Publish from h1
	testMsg := []byte("hello murmur")
	if err := ps1.Publish(ctx, TopicWaves, testMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for message
	select {
	case <-done:
		mu.Lock()
		if string(received) != string(testMsg) {
			t.Errorf("received = %q, want %q", received, testMsg)
		}
		mu.Unlock()
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestTopicPeersEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h.Host)
	if err != nil {
		t.Fatalf("New PubSub failed: %v", err)
	}
	defer ps.Close()

	// No peers for non-joined topic
	peers := ps.TopicPeers(TopicWaves)
	if len(peers) != 0 {
		t.Errorf("TopicPeers = %v, want empty", peers)
	}
}

func TestTopicConstants(t *testing.T) {
	// Verify topic names match spec
	tests := []struct {
		name string
		want string
	}{
		{"TopicWaves", "/murmur/waves/1"},
		{"TopicIdentity", "/murmur/identity/1"},
		{"TopicShroud", "/murmur/shroud/1"},
		{"TopicPulse", "/murmur/pulse/1"},
	}

	for _, tt := range tests {
		var got string
		switch tt.name {
		case "TopicWaves":
			got = TopicWaves
		case "TopicIdentity":
			got = TopicIdentity
		case "TopicShroud":
			got = TopicShroud
		case "TopicPulse":
			got = TopicPulse
		}
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPeerRateLimiting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create two nodes
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	cfg1 := transport.DefaultConfig()
	cfg1.PrivateKey = priv1
	cfg1.EnableDHT = false

	cfg2 := transport.DefaultConfig()
	cfg2.PrivateKey = priv2
	cfg2.EnableDHT = false

	h1, err := transport.NewHost(ctx, cfg1)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := transport.NewHost(ctx, cfg2)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Connect h2 to h1
	if err := h2.Connect(ctx, h1.AddrInfo()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Create pubsub for both
	ps1, err := New(ctx, h1.Host)
	if err != nil {
		t.Fatalf("New ps1 failed: %v", err)
	}
	defer ps1.Close()

	ps2, err := New(ctx, h2.Host)
	if err != nil {
		t.Fatalf("New ps2 failed: %v", err)
	}
	defer ps2.Close()

	// Subscribe h2 to topic with counter
	var received int
	var mu sync.Mutex

	err = ps2.Subscribe(ctx, TopicWaves, func(ctx context.Context, msg *pubsub.Message) {
		mu.Lock()
		received++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Join topic on h1
	_, err = ps1.Join(TopicWaves)
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	// Wait for mesh to form
	time.Sleep(500 * time.Millisecond)

	// Flood h1 with 100 messages rapidly
	// Rate limiter should allow ~10/sec with burst of 20
	// So we expect ~20-30 messages to be received (burst + 1s worth)
	for i := 0; i < 100; i++ {
		testMsg := []byte("flood")
		if err := ps1.Publish(ctx, TopicWaves, testMsg); err != nil {
			t.Logf("Publish %d failed: %v", i, err)
		}
		// Small delay to ensure messages arrive
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Check received count
	mu.Lock()
	count := received
	mu.Unlock()

	t.Logf("Received %d messages out of 100 (rate limit should cap at ~20-30)", count)

	// Rate limiter should have dropped most messages
	// Allow burst of 20 + ~10/sec for 1 second = ~30 max
	if count > 40 {
		t.Errorf("Rate limiter did not limit: received %d messages, expected ≤40", count)
	}

	// Should have received at least burst amount
	if count < 15 {
		t.Errorf("Too few messages received: got %d, expected ≥15 (burst)", count)
	}
}

func TestGetRateLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	cfg := transport.DefaultConfig()
	cfg.PrivateKey = priv
	cfg.EnableDHT = false

	h, err := transport.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h.Host)
	if err != nil {
		t.Fatalf("New PubSub failed: %v", err)
	}
	defer ps.Close()

	// Create a test peer ID
	peerID := h.Host.ID()

	// Get rate limiter for peer
	limiter1 := ps.getRateLimiter(peerID)
	if limiter1 == nil {
		t.Fatal("getRateLimiter returned nil")
	}

	// Get again - should return same limiter
	limiter2 := ps.getRateLimiter(peerID)
	if limiter1 != limiter2 {
		t.Error("getRateLimiter should return same limiter for same peer")
	}

	// Check rate limiter parameters
	// Should allow burst of 20
	for i := 0; i < 20; i++ {
		if !limiter1.Allow() {
			t.Errorf("Limiter rejected message %d in burst (burst should be 20)", i+1)
		}
	}

	// 21st should be rejected (no time passed)
	if limiter1.Allow() {
		t.Error("Limiter allowed message beyond burst limit")
	}
}
