//go:build integration
// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestWavePropagation verifies that a Wave published by one node is received by connected peers.
// Per PLAN.md Step 7: "Wave propagation test confirms <500ms latency across 3 hops"
func TestWavePropagation(t *testing.T) {
	// Create 3-node network
	nodeA := NewTestNode(t, 1)
	nodeB := NewTestNode(t, 2)
	nodeC := NewTestNode(t, 3)

	// Connect nodes in a mesh (all-to-all)
	ConnectMesh(t, []*TestNode{nodeA, nodeB, nodeC})

	// Subscribe all nodes to /murmur/waves/1 topic
	chanA := nodeA.SubscribeWaves(t)
	chanB := nodeB.SubscribeWaves(t)
	chanC := nodeC.SubscribeWaves(t)

	// Wait for GossipSub mesh to stabilize
	WaitForGossipStability(t)

	// Node A creates a Wave
	waveContent := []byte("Hello from Node A - Integration Test")
	wave, err := waves.Create(
		waves.TypeSurface,
		waveContent,
		nodeA.KeyPair,
		waves.DefaultCreateOptions(),
	)
	require.NoError(t, err, "creating wave on node A")

	// Wrap in MurmurEnvelope
	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       mustMarshal(wave),
		SenderPubkey:  nodeA.KeyPair.PublicKey,
		Signature:     wave.Signature,
		TimestampUnix: time.Now().Unix(),
		MessageId:     wave.WaveId,
	}
	envelopeBytes := mustMarshal(envelope)

	// Measure propagation latency
	startTime := time.Now()

	// Node A publishes the Wave
	nodeA.PublishWave(t, envelopeBytes)

	// Node B and C should receive the Wave
	dataB := WaitForMessage(t, chanB, 5*time.Second)
	dataC := WaitForMessage(t, chanC, 5*time.Second)

	propagationLatency := time.Since(startTime)

	// Verify received envelopes
	var envB, envC pb.MurmurEnvelope
	err = proto.Unmarshal(dataB, &envB)
	require.NoError(t, err, "unmarshaling envelope at node B")
	err = proto.Unmarshal(dataC, &envC)
	require.NoError(t, err, "unmarshaling envelope at node C")

	// Verify envelope contents match
	require.Equal(t, envelope.MessageId, envB.MessageId, "message ID should match at node B")
	require.Equal(t, envelope.MessageId, envC.MessageId, "message ID should match at node C")

	// Unmarshal and verify Wave content
	var waveB pb.Wave
	err = proto.Unmarshal(envB.Payload, &waveB)
	require.NoError(t, err, "unmarshaling wave at node B")
	require.Equal(t, waveContent, waveB.Content, "wave content should match at node B")

	var waveC pb.Wave
	err = proto.Unmarshal(envC.Payload, &waveC)
	require.NoError(t, err, "unmarshaling wave at node C")
	require.Equal(t, waveContent, waveC.Content, "wave content should match at node C")

	// Verify propagation latency
	t.Logf("Wave propagation latency: %v", propagationLatency)
	require.Less(t, propagationLatency.Milliseconds(), int64(500),
		"propagation latency should be <500ms (actual: %v)", propagationLatency)
}

// TestWavePropagationMultipleMessages verifies that multiple Waves propagate correctly.
func TestWavePropagationMultipleMessages(t *testing.T) {
	// Create 2-node network
	nodeA := NewTestNode(t, 1)
	nodeB := NewTestNode(t, 2)

	// Connect nodes
	nodeA.ConnectTo(t, nodeB)
	nodeA.WaitForPeers(t, 1, 5*time.Second)
	nodeB.WaitForPeers(t, 1, 5*time.Second)

	// Subscribe both nodes
	chanB := nodeB.SubscribeWaves(t)

	// Wait for mesh stabilization
	WaitForGossipStability(t)

	// Send 5 waves from node A
	const numWaves = 5
	waveIDs := make([][]byte, numWaves)

	for i := 0; i < numWaves; i++ {
		content := []byte(string(rune('A'+i)) + " - Test Wave " + string(rune('0'+i)))
		wave, err := waves.Create(
			waves.TypeSurface,
			content,
			nodeA.KeyPair,
			waves.DefaultCreateOptions(),
		)
		require.NoError(t, err, "creating wave %d", i)

		waveIDs[i] = wave.WaveId

		envelope := &pb.MurmurEnvelope{
			Version:       1,
			Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
			Payload:       mustMarshal(wave),
			SenderPubkey:  nodeA.KeyPair.PublicKey,
			Signature:     wave.Signature,
			TimestampUnix: time.Now().Unix(),
			MessageId:     wave.WaveId,
		}

		nodeA.PublishWave(t, mustMarshal(envelope))
		time.Sleep(100 * time.Millisecond) // Small delay between waves
	}

	// Verify all waves received at node B
	receivedIDs := make(map[string]bool)
	timeout := time.After(10 * time.Second)

	for i := 0; i < numWaves; i++ {
		select {
		case data := <-chanB:
			var env pb.MurmurEnvelope
			err := proto.Unmarshal(data, &env)
			require.NoError(t, err, "unmarshaling envelope %d", i)
			receivedIDs[string(env.MessageId)] = true
		case <-timeout:
			require.FailNow(t, "timeout waiting for wave", "received %d/%d waves", i, numWaves)
		}
	}

	// Verify all wave IDs were received
	for i, id := range waveIDs {
		require.True(t, receivedIDs[string(id)], "wave %d should be received", i)
	}
}

// TestWavePropagationLinearTopology verifies propagation in a linear chain (A→B→C).
func TestWavePropagationLinearTopology(t *testing.T) {
	// Create 3 nodes
	nodeA := NewTestNode(t, 1)
	nodeB := NewTestNode(t, 2)
	nodeC := NewTestNode(t, 3)

	// Connect in linear topology: A↔B↔C (A and C are not directly connected)
	nodeA.ConnectTo(t, nodeB)
	nodeB.ConnectTo(t, nodeC)

	nodeA.WaitForPeers(t, 1, 5*time.Second)
	nodeB.WaitForPeers(t, 2, 5*time.Second)
	nodeC.WaitForPeers(t, 1, 5*time.Second)

	// Subscribe all nodes
	nodeC.SubscribeWaves(t)
	chanC := nodeC.SubscribeWaves(t)

	// Wait for mesh stabilization
	WaitForGossipStability(t)

	// Node A publishes a Wave
	content := []byte("Linear topology test message")
	wave, err := waves.Create(
		waves.TypeSurface,
		content,
		nodeA.KeyPair,
		waves.DefaultCreateOptions(),
	)
	require.NoError(t, err)

	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       mustMarshal(wave),
		SenderPubkey:  nodeA.KeyPair.PublicKey,
		Signature:     wave.Signature,
		TimestampUnix: time.Now().Unix(),
		MessageId:     wave.WaveId,
	}

	startTime := time.Now()
	nodeA.PublishWave(t, mustMarshal(envelope))

	// Node C should receive via relay through B
	dataC := WaitForMessage(t, chanC, 5*time.Second)
	latency := time.Since(startTime)

	// Verify received envelope
	var envC pb.MurmurEnvelope
	err = proto.Unmarshal(dataC, &envC)
	require.NoError(t, err)
	require.Equal(t, envelope.MessageId, envC.MessageId)

	t.Logf("Linear topology propagation latency (2 hops): %v", latency)
	require.Less(t, latency.Milliseconds(), int64(1000),
		"2-hop propagation should complete within 1 second (actual: %v)", latency)
}

// mustMarshal marshals a protobuf message, panicking on error.
func mustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic("failed to marshal proto: " + err.Error())
	}
	return data
}
