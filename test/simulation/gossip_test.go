//go:build simulation
// +build simulation

package simulation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

// SimNode represents a simulation node with minimal setup
type SimNode struct {
	ID       int
	Host     host.Host
	PubSub   *pubsub.PubSub
	KeyPair  *keys.KeyPair
	Received []time.Time
	mu       sync.Mutex
}

// TestGossipPropagation50Nodes verifies Wave propagation across 50 nodes
// Target: 99% delivery within 3 seconds (allowing 6x the 500ms 3-hop target)
func TestGossipPropagation50Nodes(t *testing.T) {
	const nodeCount = 50
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create 50 nodes with in-memory transports
	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()
	}

	// Connect nodes in a mesh topology (each connects to 6-8 random peers)
	connectMesh(t, nodes, 6, 8)

	// Wait for network to stabilize
	time.Sleep(2 * time.Second)

	// Subscribe all nodes to Wave topic
	topic := gossip.TopicWaves
	subs := make([]*pubsub.Subscription, nodeCount)
	var receivedCount atomic.Int32

	for i, node := range nodes {
		sub, err := node.PubSub.Subscribe(topic)
		require.NoError(t, err, "subscribing node %d", i)
		subs[i] = sub

		// Start message receivers
		go func(n *SimNode, s *pubsub.Subscription) {
			for {
				_, err := s.Next(ctx)
				if err != nil {
					return
				}
				n.mu.Lock()
				n.Received = append(n.Received, time.Now())
				n.mu.Unlock()
				receivedCount.Add(1)
			}
		}(node, sub)
	}

	// Wait for subscriptions to propagate
	time.Sleep(1 * time.Second)

	// Create a Wave from node 0
	originNode := nodes[0]
	wave, err := createTestWave(originNode.KeyPair)
	require.NoError(t, err, "creating test wave")

	// Wrap Wave in MurmurEnvelope
	envelope, err := wrapWave(wave, originNode.KeyPair)
	require.NoError(t, err, "wrapping wave")

	// Serialize envelope
	data, err := proto.Marshal(envelope)
	require.NoError(t, err, "marshaling envelope")

	// Publish Wave from node 0
	publishTime := time.Now()
	err = originNode.PubSub.Publish(topic, data)
	require.NoError(t, err, "publishing wave")

	// Wait for propagation (max 5 seconds)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for wave propagation")
		case <-ticker.C:
			count := receivedCount.Load()
			// Target: 99% of nodes (excluding origin) = 49 * 0.99 ≈ 48 nodes
			if count >= 48 {
				goto checkLatency
			}
		}
	}

checkLatency:
	// Collect latencies
	latencies := make([]time.Duration, 0, nodeCount)
	for i, node := range nodes {
		if i == 0 {
			continue // Skip origin node
		}
		node.mu.Lock()
		if len(node.Received) > 0 {
			latency := node.Received[0].Sub(publishTime)
			latencies = append(latencies, latency)
		}
		node.mu.Unlock()
	}

	// Verify delivery rate
	deliveryRate := float64(len(latencies)) / float64(nodeCount-1)
	require.GreaterOrEqual(t, deliveryRate, 0.99, "delivery rate should be ≥99%%")

	// Verify 99th percentile latency
	if len(latencies) > 0 {
		p99Index := int(float64(len(latencies)) * 0.99)
		if p99Index >= len(latencies) {
			p99Index = len(latencies) - 1
		}

		// Sort latencies for percentile calculation
		sortedLatencies := make([]time.Duration, len(latencies))
		copy(sortedLatencies, latencies)
		for i := 0; i < len(sortedLatencies); i++ {
			for j := i + 1; j < len(sortedLatencies); j++ {
				if sortedLatencies[i] > sortedLatencies[j] {
					sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
				}
			}
		}

		p99Latency := sortedLatencies[p99Index]
		t.Logf("Propagation stats: delivered=%d/%d (%.1f%%), p99_latency=%v",
			len(latencies), nodeCount-1, deliveryRate*100, p99Latency)

		// Target: <3 seconds for multi-hop fanout
		require.Less(t, p99Latency, 3*time.Second,
			"p99 latency should be <3s (allowing 6x the 500ms 3-hop target)")
	}
}

// TestGossipPropagation10NodesStress verifies propagation under message load
func TestGossipPropagation10NodesStress(t *testing.T) {
	const nodeCount = 10
	const waveCount = 100
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err)
		nodes[i] = node
		defer node.Host.Close()
	}

	connectMesh(t, nodes, 4, 6)
	time.Sleep(1 * time.Second)

	topic := gossip.TopicWaves
	var receivedTotal atomic.Int32

	for _, node := range nodes {
		sub, err := node.PubSub.Subscribe(topic)
		require.NoError(t, err)

		go func(s *pubsub.Subscription) {
			for {
				_, err := s.Next(ctx)
				if err != nil {
					return
				}
				receivedTotal.Add(1)
			}
		}(sub)
	}

	time.Sleep(1 * time.Second)

	// Publish 100 Waves from random nodes
	startTime := time.Now()
	for i := 0; i < waveCount; i++ {
		nodeIdx := i % nodeCount
		node := nodes[nodeIdx]

		wave, err := createTestWave(node.KeyPair)
		require.NoError(t, err)

		envelope, err := wrapWave(wave, node.KeyPair)
		require.NoError(t, err)

		data, err := proto.Marshal(envelope)
		require.NoError(t, err)

		err = node.PubSub.Publish(topic, data)
		require.NoError(t, err)

		// Small delay between publishes
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for propagation
	time.Sleep(10 * time.Second)
	duration := time.Since(startTime)

	total := receivedTotal.Load()
	expected := int32(waveCount * (nodeCount - 1)) // Each Wave to N-1 nodes
	deliveryRate := float64(total) / float64(expected)

	t.Logf("Stress test: published=%d waves, received=%d/%d (%.1f%%), duration=%v",
		waveCount, total, expected, deliveryRate*100, duration)

	require.GreaterOrEqual(t, deliveryRate, 0.95,
		"delivery rate should be ≥95%% under load")
}

// createSimNode creates a libp2p host with GossipSub for simulation
func createSimNode(ctx context.Context, id int) (*SimNode, error) {
	// Generate keypair
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}

	// Convert Ed25519 private key to libp2p crypto format
	libp2pPrivKey, _, err := libp2pcrypto.KeyPairFromStdKey(&kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("converting key: %w", err)
	}

	// Create libp2p host with memory transport
	h, err := libp2p.New(
		libp2p.Identity(libp2pPrivKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.DisableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating host: %w", err)
	}

	// Create GossipSub with default parameters for testing
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("creating pubsub: %w", err)
	}

	return &SimNode{
		ID:      id,
		Host:    h,
		PubSub:  ps,
		KeyPair: kp,
	}, nil
}

// connectMesh connects nodes in a random mesh topology
func connectMesh(t *testing.T, nodes []*SimNode, minDegree, maxDegree int) {
	t.Helper()
	for i, node := range nodes {
		// Connect to random peers
		degree := minDegree + (i % (maxDegree - minDegree + 1))
		for j := 0; j < degree; j++ {
			// Select random peer (not self)
			peerIdx := (i + j + 1) % len(nodes)
			peer := nodes[peerIdx]

			// Connect bidirectionally
			err := node.Host.Connect(context.Background(), peer.Host.Peerstore().PeerInfo(peer.Host.ID()))
			if err != nil {
				t.Logf("Warning: failed to connect node %d to %d: %v", i, peerIdx, err)
			}
		}
	}
}

// createTestWave creates a Wave with PoW for testing
func createTestWave(kp *keys.KeyPair) (*pb.Wave, error) {
	content := []byte(fmt.Sprintf("Test wave at %d", time.Now().UnixNano()))

	// Create Wave with low difficulty for fast testing
	opts := waves.DefaultCreateOptions()
	opts.Difficulty = 12 // Lower than default 20 for faster testing

	wave, err := waves.Create(waves.TypeSurface, content, kp, opts)
	if err != nil {
		return nil, fmt.Errorf("creating wave: %w", err)
	}

	return wave, nil
}

// wrapWave wraps a Wave in a MurmurEnvelope
func wrapWave(wave *pb.Wave, kp *keys.KeyPair) (*pb.MurmurEnvelope, error) {
	// Serialize Wave as payload
	payload, err := proto.Marshal(wave)
	if err != nil {
		return nil, fmt.Errorf("marshaling wave: %w", err)
	}

	// Compute message ID (BLAKE3 hash of payload)
	hasher := blake3.New()
	hasher.Write(payload)
	messageID := hasher.Sum(nil)

	// Create envelope
	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  kp.PublicKey,
		TimestampUnix: time.Now().Unix(),
		MessageId:     messageID,
	}

	// Sign envelope (version || type || payload)
	sigData := make([]byte, 0, 4+4+len(payload))
	sigData = append(sigData, byte(envelope.Version>>24), byte(envelope.Version>>16), byte(envelope.Version>>8), byte(envelope.Version))
	sigData = append(sigData, byte(envelope.Type>>24), byte(envelope.Type>>16), byte(envelope.Type>>8), byte(envelope.Type))
	sigData = append(sigData, payload...)

	signature := kp.Sign(sigData)
	envelope.Signature = signature

	return envelope, nil
}
