// Package mechanics - Shared test helpers for mechanics subpackages.
package mechanics

import (
	"context"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// mockPublisher is a mock Publisher for testing across all mechanics subpackages.
type mockPublisher struct {
	published []publishedMessage
}

type publishedMessage struct {
	topic string
	data  []byte
}

func (m *mockPublisher) Publish(_ context.Context, topic string, data []byte) error {
	m.published = append(m.published, publishedMessage{topic: topic, data: data})
	return nil
}

func (m *mockPublisher) lastMessage() (*pb.GossipMessage, error) {
	if len(m.published) == 0 {
		return nil, nil
	}
	msg := &pb.GossipMessage{}
	err := proto.Unmarshal(m.published[len(m.published)-1].data, msg)
	return msg, err
}

// mockResonanceGate is a test implementation of ResonanceGate for all mechanics tests.
type mockResonanceGate struct {
	scores map[[32]byte]int
}

func (m *mockResonanceGate) GetResonance(specterKey [32]byte) (int, error) {
	if score, ok := m.scores[specterKey]; ok {
		return score, nil
	}
	return 0, nil
}

func newMockGate(specterKey [32]byte, resonance int) *mockResonanceGate {
	return &mockResonanceGate{
		scores: map[[32]byte]int{specterKey: resonance},
	}
}
