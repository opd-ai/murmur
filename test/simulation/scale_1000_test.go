//go:build simulation
// +build simulation

package simulation

import (
	"context"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestGossipPropagation1000NodesWithProfiling verifies Wave propagation across 1000 nodes
// with CPU and memory profiling for performance analysis.
// Target: 90% delivery within 10 seconds for very large-scale mesh
func TestGossipPropagation1000NodesWithProfiling(t *testing.T) {
	const nodeCount = 1000
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Start CPU profiling
	cpuProfile, err := os.Create("cpu_1000nodes.prof")
	require.NoError(t, err, "creating CPU profile")
	defer cpuProfile.Close()
	err = pprof.StartCPUProfile(cpuProfile)
	require.NoError(t, err, "starting CPU profile")
	defer pprof.StopCPUProfile()

	t.Logf("Creating %d simulation nodes...", nodeCount)
	startTime := time.Now()
	nodes := make([]*SimNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()

		if (i+1)%100 == 0 {
			t.Logf("  Created %d/%d nodes (%.1f%%)...", i+1, nodeCount, float64(i+1)/float64(nodeCount)*100)
		}
	}
	nodeCreationTime := time.Since(startTime)
	t.Logf("✓ Created %d nodes in %v", nodeCount, nodeCreationTime)

	// Connect nodes in a mesh topology (each connects to 8-12 random peers)
	t.Log("Establishing mesh topology...")
	meshStart := time.Now()
	connectMesh(t, nodes, 8, 12)
	meshTime := time.Since(meshStart)
	t.Logf("✓ Mesh topology established in %v", meshTime)

	// Wait for network to stabilize
	t.Log("Waiting for network to stabilize...")
	time.Sleep(5 * time.Second)

	// Subscribe all nodes to Wave topic
	topic := gossip.TopicWaves
	subs := make([]*pubsub.Subscription, nodeCount)
	var receivedCount atomic.Int32

	t.Log("Subscribing nodes to Wave topic...")
	subStart := time.Now()
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

		if (i+1)%200 == 0 {
			t.Logf("  Subscribed %d/%d nodes...", i+1, nodeCount)
		}
	}
	subTime := time.Since(subStart)
	t.Logf("✓ %d nodes subscribed in %v", nodeCount, subTime)

	// Wait for subscriptions to propagate
	t.Log("Waiting for subscriptions to propagate...")
	time.Sleep(5 * time.Second)

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

	// Wait for propagation (max 20 seconds)
	timeout := time.After(20 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			count := receivedCount.Load()
			t.Logf("Timeout after 20s: received=%d/%d (%.1f%%)",
				count, nodeCount-1, float64(count)/float64(nodeCount-1)*100)
			goto checkResults
		case <-ticker.C:
			count := receivedCount.Load()
			// Target: 90% of nodes (excluding origin) = 999 * 0.90 = 899 nodes
			if count >= 899 {
				elapsed := time.Since(publishTime)
				t.Logf("✓ Wave propagated to %d/%d nodes (%.1f%%) in %v",
					count, nodeCount-1, float64(count)/float64(nodeCount-1)*100, elapsed)
				goto checkResults
			}
		}
	}

checkResults:
	// Stop CPU profiling before collecting results
	pprof.StopCPUProfile()

	// Write heap profile
	heapProfile, err := os.Create("heap_1000nodes.prof")
	require.NoError(t, err, "creating heap profile")
	defer heapProfile.Close()
	runtime.GC()
	err = pprof.WriteHeapProfile(heapProfile)
	require.NoError(t, err, "writing heap profile")

	// Collect latencies
	latencies := make([]time.Duration, 0, nodeCount)
	for i, node := range nodes {
		if i == 0 {
			continue
		}
		node.mu.Lock()
		if len(node.Received) > 0 {
			latency := node.Received[0].Sub(publishTime)
			latencies = append(latencies, latency)
		}
		node.mu.Unlock()
	}

	// Calculate statistics
	deliveryRate := float64(len(latencies)) / float64(nodeCount-1)
	t.Logf("\n=== 1000-Node Simulation Results ===")
	t.Logf("Node creation time: %v", nodeCreationTime)
	t.Logf("Mesh connection time: %v", meshTime)
	t.Logf("Subscription time: %v", subTime)
	t.Logf("Delivery rate: %.2f%% (%d/%d nodes)", deliveryRate*100, len(latencies), nodeCount-1)

	// Verify delivery rate
	require.GreaterOrEqual(t, deliveryRate, 0.90, "delivery rate should be ≥90%%")

	// Calculate latency percentiles
	if len(latencies) > 0 {
		// Sort latencies
		sortedLatencies := make([]time.Duration, len(latencies))
		copy(sortedLatencies, latencies)
		for i := 0; i < len(sortedLatencies); i++ {
			for j := i + 1; j < len(sortedLatencies); j++ {
				if sortedLatencies[i] > sortedLatencies[j] {
					sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
				}
			}
		}

		p50 := sortedLatencies[len(sortedLatencies)/2]
		p95 := sortedLatencies[int(float64(len(sortedLatencies))*0.95)]
		p99 := sortedLatencies[int(float64(len(sortedLatencies))*0.99)]

		t.Logf("Latency p50: %v", p50)
		t.Logf("Latency p95: %v", p95)
		t.Logf("Latency p99: %v", p99)
		t.Logf("Profile files: cpu_1000nodes.prof, heap_1000nodes.prof")

		// Performance targets from TECHNICAL_IMPLEMENTATION.md
		require.Less(t, p50, 5*time.Second, "p50 latency should be <5s")
		require.Less(t, p99, 10*time.Second, "p99 latency should be <10s")
	}
}
