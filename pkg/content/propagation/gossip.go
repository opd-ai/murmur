// Package propagation provides gossip relay logic, hop counting, and deduplication.
// This file defines the GossipSub publisher interface for network propagation.

package propagation

import (
	"context"
	"errors"
	"sync"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// GossipSub topic names per TECHNICAL_IMPLEMENTATION.md §2.2.
const (
	// TopicWaves is the GossipSub topic for Wave messages.
	TopicWaves = "/murmur/waves/1"

	// TopicIdentity is the GossipSub topic for identity declarations.
	TopicIdentity = "/murmur/identity/1"

	// TopicShroud is the GossipSub topic for Shroud relay advertisements.
	TopicShroud = "/murmur/shroud/1"

	// TopicPulse is the GossipSub topic for heartbeat pings.
	TopicPulse = "/murmur/pulse/1"
)

// Publisher defines the interface for publishing messages to GossipSub.
// Implementations wrap the libp2p pubsub.Topic interface.
type Publisher interface {
	// Publish sends a message to the GossipSub topic.
	// The data should be a serialized MurmurEnvelope protobuf.
	Publish(ctx context.Context, data []byte) error

	// Topic returns the topic string.
	Topic() string
}

// GossipRelay extends Relay with network publishing capabilities.
type GossipRelay struct {
	*Relay
	publisher Publisher
}

// NewGossipRelay creates a relay with GossipSub publishing.
func NewGossipRelay(publisher Publisher) *GossipRelay {
	return &GossipRelay{
		Relay:     NewRelay(),
		publisher: publisher,
	}
}

// NewGossipRelayWithConfig creates a gossip relay with custom configuration.
func NewGossipRelayWithConfig(cfg RelayConfig, publisher Publisher) *GossipRelay {
	return &GossipRelay{
		Relay:     NewRelayWithConfig(cfg),
		publisher: publisher,
	}
}

// Publish sends a Wave to the GossipSub network.
// The Wave is wrapped in a MurmurEnvelope before publishing.
func (gr *GossipRelay) Publish(ctx context.Context, wave *pb.Wave) error {
	if wave == nil {
		return errors.New("cannot publish nil wave")
	}
	if gr.publisher == nil {
		return errors.New("no publisher configured")
	}

	envelope, err := wrapWaveInEnvelope(wave)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	return gr.publisher.Publish(ctx, data)
}

// ReceiveAndRelay processes an incoming Wave and relays if valid.
// Returns the relayed Wave or an error if the Wave should not be relayed.
func (gr *GossipRelay) ReceiveAndRelay(ctx context.Context, wave *pb.Wave) (*pb.Wave, error) {
	// Process through base relay first.
	relayWave, err := gr.Relay.Receive(wave)
	if err != nil {
		return nil, err
	}

	// Only relay if hop count is still under max.
	if relayWave.HopCount < gr.maxHops {
		if pubErr := gr.Publish(ctx, relayWave); pubErr != nil {
			// Log but don't fail - wave was still valid and processed.
			// The caller can decide what to do based on the error.
			return relayWave, pubErr
		}
	}

	return relayWave, nil
}

// wrapWaveInEnvelope creates a MurmurEnvelope for a Wave.
func wrapWaveInEnvelope(wave *pb.Wave) (*pb.MurmurEnvelope, error) {
	payload, err := proto.Marshal(wave)
	if err != nil {
		return nil, err
	}

	return &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  wave.AuthorPubkey,
		Signature:     wave.Signature,
		TimestampUnix: wave.CreatedAt,
		MessageId:     wave.WaveId,
	}, nil
}

// MockPublisher is a test implementation of Publisher.
type MockPublisher struct {
	mu        sync.Mutex
	topic     string
	published [][]byte
	err       error
}

// NewMockPublisher creates a mock publisher for testing.
func NewMockPublisher(topic string) *MockPublisher {
	return &MockPublisher{
		topic:     topic,
		published: make([][]byte, 0),
	}
}

// Publish records the data for later verification.
func (m *MockPublisher) Publish(_ context.Context, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, data)
	return nil
}

// Topic returns the mock topic string.
func (m *MockPublisher) Topic() string {
	return m.topic
}

// Published returns all published messages.
func (m *MockPublisher) Published() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.published
}

// SetError sets an error to return on next publish.
func (m *MockPublisher) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Clear clears the published messages.
func (m *MockPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = make([][]byte, 0)
	m.err = nil
}
