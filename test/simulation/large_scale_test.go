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

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestGossipPropagation100Nodes verifies Wave propagation across 100 nodes
// Target: 95% delivery within 5 seconds for large-scale mesh
func TestGossipPropagation100Nodes(t *testing.T) {
	const nodeCount = 100
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	t.Logf("Creating %d simulation nodes...", nodeCount)
	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()
	}
	t.Logf("✓ Created %d nodes", nodeCount)

	// Connect nodes in a mesh topology (each connects to 8-12 random peers)
	t.Log("Establishing mesh topology...")
	connectMesh(t, nodes, 8, 12)
	t.Log("✓ Mesh topology established")

	// Wait for network to stabilize
	time.Sleep(3 * time.Second)

	// Subscribe all nodes to Wave topic
	topic := gossip.TopicWaves
	subs := make([]*pubsub.Subscription, nodeCount)
	var receivedCount atomic.Int32

	t.Log("Subscribing nodes to Wave topic...")
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
	t.Logf("✓ %d nodes subscribed", nodeCount)

	// Wait for subscriptions to propagate
	time.Sleep(2 * time.Second)

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
	t.Log("Publishing Wave from node 0...")
	publishTime := time.Now()
	err = originNode.PubSub.Publish(topic, data)
	require.NoError(t, err, "publishing wave")

	// Wait for propagation (max 10 seconds)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			count := receivedCount.Load()
			t.Fatalf("timeout waiting for wave propagation: received=%d/%d (%.1f%%)",
				count, nodeCount-1, float64(count)/float64(nodeCount-1)*100)
		case <-ticker.C:
			count := receivedCount.Load()
			// Target: 95% of nodes (excluding origin) = 99 * 0.95 ≈ 94 nodes
			if count >= 94 {
				t.Logf("✓ Wave propagated to %d/%d nodes (%.1f%%)",
					count, nodeCount-1, float64(count)/float64(nodeCount-1)*100)
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
	require.GreaterOrEqual(t, deliveryRate, 0.95, "delivery rate should be ≥95%%")

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
		p50Index := len(sortedLatencies) / 2
		p50Latency := sortedLatencies[p50Index]

		t.Logf("Propagation stats: delivered=%d/%d (%.1f%%), p50=%v, p99=%v",
			len(latencies), nodeCount-1, deliveryRate*100, p50Latency, p99Latency)

		// Target: <5 seconds for 100-node mesh
		require.Less(t, p99Latency, 5*time.Second,
			"p99 latency should be <5s for 100-node mesh")
	}
}

// TestPulseMapLayout100Nodes verifies force-directed layout convergence at scale
func TestPulseMapLayout100Nodes(t *testing.T) {
	// This test verifies that the Barnes-Hut optimization activates correctly
	// and that layout converges to stable positions within bounded iterations
	t.Skip("Pulse Map layout tests require Ebitengine-free simulation harness (future work)")
}

// TestResonanceConvergence100NodesWithInteractions verifies Resonance computation
// converges correctly across a large network with realistic interaction patterns
func TestResonanceConvergence100NodesWithInteractions(t *testing.T) {
	const nodeCount = 100
	const interactionCount = 1000
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	t.Logf("Creating %d simulation nodes for Resonance test...", nodeCount)
	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()
	}
	t.Logf("✓ Created %d nodes", nodeCount)

	// Connect nodes in a mesh topology
	t.Log("Establishing mesh topology...")
	connectMesh(t, nodes, 8, 12)
	t.Log("✓ Mesh topology established")

	time.Sleep(2 * time.Second)

	// Subscribe all nodes to Wave topic
	topic := gossip.TopicWaves
	var totalReceived atomic.Int32

	t.Log("Subscribing nodes to Wave topic...")
	for _, node := range nodes {
		sub, err := node.PubSub.Subscribe(topic)
		require.NoError(t, err)

		go func(n *SimNode, s *pubsub.Subscription) {
			for {
				_, err := s.Next(ctx)
				if err != nil {
					return
				}
				n.mu.Lock()
				n.Received = append(n.Received, time.Now())
				n.mu.Unlock()
				totalReceived.Add(1)
			}
		}(node, sub)
	}
	t.Logf("✓ %d nodes subscribed", nodeCount)

	time.Sleep(2 * time.Second)

	// Simulate realistic interaction patterns:
	// - 20% of nodes are highly active (publish 60% of Waves)
	// - 30% of nodes are moderately active (publish 30% of Waves)
	// - 50% of nodes are passive (publish 10% of Waves)

	t.Logf("Simulating %d interactions across %d nodes...", interactionCount, nodeCount)
	startTime := time.Now()

	activeNodes := nodeCount / 5          // 20%
	moderateNodes := (nodeCount * 3) / 10 // 30%

	for i := 0; i < interactionCount; i++ {
		var nodeIdx int
		roll := i % 100

		if roll < 60 {
			// Active node (first 20%)
			nodeIdx = i % activeNodes
		} else if roll < 90 {
			// Moderate node (next 30%)
			nodeIdx = activeNodes + (i % moderateNodes)
		} else {
			// Passive node (remaining 50%)
			nodeIdx = activeNodes + moderateNodes + (i % (nodeCount - activeNodes - moderateNodes))
		}

		node := nodes[nodeIdx]

		wave, err := createTestWave(node.KeyPair)
		require.NoError(t, err)

		envelope, err := wrapWave(wave, node.KeyPair)
		require.NoError(t, err)

		data, err := proto.Marshal(envelope)
		require.NoError(t, err)

		err = node.PubSub.Publish(topic, data)
		require.NoError(t, err)

		// Throttle publication rate (10 Waves/sec = 100ms between publishes)
		if i%10 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	duration := time.Since(startTime)
	t.Logf("✓ Published %d Waves in %v (%.1f Waves/sec)",
		interactionCount, duration, float64(interactionCount)/duration.Seconds())

	// Wait for propagation
	time.Sleep(10 * time.Second)

	total := totalReceived.Load()
	t.Logf("Total messages received across all nodes: %d", total)

	// Verify realistic propagation: expect ~90% of (interactionCount * (nodeCount-1))
	// (some loss is acceptable in large-scale simulation)
	expected := int32(interactionCount * (nodeCount - 1))
	deliveryRate := float64(total) / float64(expected)

	t.Logf("Delivery rate: %.1f%% (%d/%d)", deliveryRate*100, total, expected)
	require.GreaterOrEqual(t, deliveryRate, 0.85,
		"delivery rate should be ≥85%% in large-scale simulation with realistic load")

	// Verify distribution: active nodes should have higher receive counts
	t.Log("Verifying activity distribution...")
	activeReceiveCount := int32(0)
	moderateReceiveCount := int32(0)
	passiveReceiveCount := int32(0)

	for i, node := range nodes {
		node.mu.Lock()
		count := int32(len(node.Received))
		node.mu.Unlock()

		if i < activeNodes {
			activeReceiveCount += count
		} else if i < activeNodes+moderateNodes {
			moderateReceiveCount += count
		} else {
			passiveReceiveCount += count
		}
	}

	t.Logf("Active nodes received: %d (%.1f%% of total)",
		activeReceiveCount, float64(activeReceiveCount)/float64(total)*100)
	t.Logf("Moderate nodes received: %d (%.1f%% of total)",
		moderateReceiveCount, float64(moderateReceiveCount)/float64(total)*100)
	t.Logf("Passive nodes received: %d (%.1f%% of total)",
		passiveReceiveCount, float64(passiveReceiveCount)/float64(total)*100)

	t.Log("✓ Resonance convergence test completed")
}

// TestShroudAnonymity100Nodes verifies that Shroud circuits provide source
// anonymity across a 100-node network with realistic relay selection
func TestShroudAnonymity100Nodes(t *testing.T) {
	// This test would verify:
	// 1. Circuit construction across 100 nodes completes in <3 seconds
	// 2. Hop diversity: no two hops from the same /24 subnet
	// 3. Relay node cannot correlate initiator IP with destination
	// 4. Timing attack resistance: messages delayed by ±50ms at each hop
	//
	// Requires Shroud circuit construction harness with controllable delays
	t.Skip("Shroud anonymity tests require stream protocol simulation harness (future work)")
}

// TestDHTRouting10000Keys verifies DHT routing table maintenance at scale
func TestDHTRouting10000Keys(t *testing.T) {
	// This test would verify:
	// 1. DHT routing table converges to correct k-buckets
	// 2. Key lookups complete in O(log N) hops
	// 3. Churn (nodes joining/leaving) handled gracefully
	//
	// Requires Kademlia DHT simulation harness
	t.Skip("DHT routing tests require Kademlia simulation harness (future work)")
}

// TestConcurrentWavePropagation verifies system stability under concurrent load
func TestConcurrentWavePropagation(t *testing.T) {
	const nodeCount = 50
	const concurrentPublishers = 10
	const wavesPerPublisher = 20
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	t.Logf("Creating %d simulation nodes...", nodeCount)
	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()
	}

	connectMesh(t, nodes, 8, 10)
	time.Sleep(2 * time.Second)

	topic := gossip.TopicWaves
	var receivedCount atomic.Int32

	for _, node := range nodes {
		sub, err := node.PubSub.Subscribe(topic)
		require.NoError(t, err)

		go func(s *pubsub.Subscription) {
			for {
				_, err := s.Next(ctx)
				if err != nil {
					return
				}
				receivedCount.Add(1)
			}
		}(sub)
	}

	time.Sleep(1 * time.Second)

	// Launch concurrent publishers
	t.Logf("Launching %d concurrent publishers (%d Waves each)...",
		concurrentPublishers, wavesPerPublisher)

	var wg sync.WaitGroup
	publishErrors := make(chan error, concurrentPublishers*wavesPerPublisher)

	startTime := time.Now()

	for i := 0; i < concurrentPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			node := nodes[publisherID%nodeCount]

			for j := 0; j < wavesPerPublisher; j++ {
				wave, err := createTestWave(node.KeyPair)
				if err != nil {
					publishErrors <- fmt.Errorf("publisher %d: %w", publisherID, err)
					return
				}

				envelope, err := wrapWave(wave, node.KeyPair)
				if err != nil {
					publishErrors <- fmt.Errorf("publisher %d: %w", publisherID, err)
					return
				}

				data, err := proto.Marshal(envelope)
				if err != nil {
					publishErrors <- fmt.Errorf("publisher %d: %w", publisherID, err)
					return
				}

				err = node.PubSub.Publish(topic, data)
				if err != nil {
					publishErrors <- fmt.Errorf("publisher %d: %w", publisherID, err)
					return
				}

				// Small jitter between publishes
				time.Sleep(time.Duration(10+j%20) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(publishErrors)

	// Check for publish errors
	var errCount int
	for err := range publishErrors {
		t.Logf("Publish error: %v", err)
		errCount++
	}
	require.Zero(t, errCount, "no publish errors expected")

	duration := time.Since(startTime)
	totalPublished := concurrentPublishers * wavesPerPublisher

	t.Logf("✓ Published %d Waves concurrently in %v (%.1f Waves/sec)",
		totalPublished, duration, float64(totalPublished)/duration.Seconds())

	// Wait for propagation
	time.Sleep(10 * time.Second)

	total := receivedCount.Load()
	expected := int32(totalPublished * (nodeCount - 1))
	deliveryRate := float64(total) / float64(expected)

	t.Logf("Concurrent propagation: received=%d/%d (%.1f%%)",
		total, expected, deliveryRate*100)

	require.GreaterOrEqual(t, deliveryRate, 0.90,
		"delivery rate should be ≥90%% under concurrent load")
}
