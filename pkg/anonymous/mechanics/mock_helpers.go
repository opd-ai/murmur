// Package mechanics - Exported test helpers for mechanics subpackages.
package mechanics

import (
	"context"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// MockPublisher is an exported mock Publisher for testing in subpackages.
type MockPublisher struct {
	Published []PublishedMessage
}

// PublishedMessage holds a published message for testing.
type PublishedMessage struct {
	Topic string
	Data  []byte
}

// Publish implements the Publisher interface.
func (m *MockPublisher) Publish(_ context.Context, topic string, data []byte) error {
	m.Published = append(m.Published, PublishedMessage{Topic: topic, Data: data})
	return nil
}

// LastMessage retrieves the last published GossipMessage.
func (m *MockPublisher) LastMessage() (*pb.GossipMessage, error) {
	if len(m.Published) == 0 {
		return nil, nil
	}
	msg := &pb.GossipMessage{}
	err := proto.Unmarshal(m.Published[len(m.Published)-1].Data, msg)
	return msg, err
}

// MockResonanceGate is an exported test implementation of ResonanceGate.
type MockResonanceGate struct {
	Scores map[[32]byte]int
}

// GetResonance implements the ResonanceGate interface.
func (m *MockResonanceGate) GetResonance(specterKey [32]byte) (int, error) {
	if score, ok := m.Scores[specterKey]; ok {
		return score, nil
	}
	return 0, nil
}

// NewMockGate creates a new MockResonanceGate with a single specter score.
func NewMockGate(specterKey [32]byte, resonance int) *MockResonanceGate {
	return &MockResonanceGate{
		Scores: map[[32]byte]int{specterKey: resonance},
	}
}
