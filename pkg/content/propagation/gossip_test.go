package propagation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

func createTestWaveForGossip(t *testing.T) *pb.Wave {
	t.Helper()

	// Generate Ed25519 keypair directly.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	content := []byte("test wave content")
	now := time.Now().Unix()

	wave := &pb.Wave{
		WaveId:       make([]byte, 32),
		AuthorPubkey: pubKey,
		CreatedAt:    now,
		WaveType:     pb.WaveType(0x01),
		Content:      content,
		TtlSeconds:   3600,
		HopCount:     0,
	}

	// Copy wave ID from content hash (simplified).
	copy(wave.WaveId, content)

	// Compute PoW with test difficulty.
	powData := append(wave.Content, wave.AuthorPubkey...)
	work, err := pow.Compute(powData, 1)
	if err != nil {
		t.Fatalf("failed to compute PoW: %v", err)
	}
	wave.PowNonce = work.Nonce

	// Sign wave.
	sigData := append(wave.WaveId, wave.Content...)
	wave.Signature = ed25519.Sign(privKey, sigData)

	return wave
}

func TestNewGossipRelay(t *testing.T) {
	pub := NewMockPublisher(TopicWaves)
	gr := NewGossipRelay(pub)

	if gr == nil {
		t.Fatal("NewGossipRelay returned nil")
	}
	if gr.Relay == nil {
		t.Error("GossipRelay.Relay is nil")
	}
	if gr.publisher == nil {
		t.Error("GossipRelay.publisher is nil")
	}
}

func TestMockPublisher(t *testing.T) {
	pub := NewMockPublisher(TopicWaves)

	if pub.Topic() != TopicWaves {
		t.Errorf("Topic = %s, want %s", pub.Topic(), TopicWaves)
	}

	// Test publish.
	ctx := context.Background()
	data := []byte("test data")
	if err := pub.Publish(ctx, data); err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	if len(pub.Published()) != 1 {
		t.Errorf("Published count = %d, want 1", len(pub.Published()))
	}

	// Test error injection.
	testErr := errors.New("publish error")
	pub.SetError(testErr)
	if err := pub.Publish(ctx, data); err != testErr {
		t.Errorf("Publish error = %v, want %v", err, testErr)
	}

	// Test clear.
	pub.Clear()
	if len(pub.Published()) != 0 {
		t.Error("Published not cleared")
	}
	if pub.err != nil {
		t.Error("Error not cleared")
	}
}

func TestGossipRelayPublish(t *testing.T) {
	pub := NewMockPublisher(TopicWaves)
	gr := NewGossipRelay(pub)

	wave := createTestWaveForGossip(t)
	ctx := context.Background()

	err := gr.Publish(ctx, wave)
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	if len(pub.Published()) != 1 {
		t.Errorf("Published count = %d, want 1", len(pub.Published()))
	}

	// Verify envelope structure.
	var envelope pb.MurmurEnvelope
	if err := proto.Unmarshal(pub.Published()[0], &envelope); err != nil {
		t.Errorf("Failed to unmarshal envelope: %v", err)
	}

	if envelope.Version != 1 {
		t.Errorf("Envelope version = %d, want 1", envelope.Version)
	}
	if envelope.Type != pb.MessageType_MESSAGE_TYPE_WAVE {
		t.Errorf("Envelope type = %v, want WAVE", envelope.Type)
	}
}

func TestGossipRelayPublishNil(t *testing.T) {
	pub := NewMockPublisher(TopicWaves)
	gr := NewGossipRelay(pub)

	ctx := context.Background()
	err := gr.Publish(ctx, nil)
	if err == nil {
		t.Error("Publish(nil) should fail")
	}
}

func TestGossipRelayNoPublisher(t *testing.T) {
	gr := &GossipRelay{
		Relay:     NewRelay(),
		publisher: nil,
	}

	wave := createTestWaveForGossip(t)
	ctx := context.Background()

	err := gr.Publish(ctx, wave)
	if err == nil {
		t.Error("Publish without publisher should fail")
	}
}

func TestTopicConstants(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		expected string
	}{
		{"waves", TopicWaves, "/murmur/waves/1"},
		{"identity", TopicIdentity, "/murmur/identity/1"},
		{"shroud", TopicShroud, "/murmur/shroud/1"},
		{"pulse", TopicPulse, "/murmur/pulse/1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.topic != tt.expected {
				t.Errorf("Topic = %s, want %s", tt.topic, tt.expected)
			}
		})
	}
}

func TestWrapWaveInEnvelope(t *testing.T) {
	wave := createTestWaveForGossip(t)

	envelope, err := wrapWaveInEnvelope(wave)
	if err != nil {
		t.Fatalf("wrapWaveInEnvelope failed: %v", err)
	}

	if envelope.Version != 1 {
		t.Errorf("Version = %d, want 1", envelope.Version)
	}
	if envelope.Type != pb.MessageType_MESSAGE_TYPE_WAVE {
		t.Errorf("Type = %v, want WAVE", envelope.Type)
	}
	if len(envelope.Payload) == 0 {
		t.Error("Payload is empty")
	}

	// Verify sender info matches wave.
	if string(envelope.SenderPubkey) != string(wave.AuthorPubkey) {
		t.Error("SenderPubkey mismatch")
	}
	if envelope.TimestampUnix != wave.CreatedAt {
		t.Error("TimestampUnix mismatch")
	}
	if string(envelope.MessageId) != string(wave.WaveId) {
		t.Error("MessageId mismatch")
	}

	// Verify payload can be unmarshaled back to wave.
	var decoded pb.Wave
	if err := proto.Unmarshal(envelope.Payload, &decoded); err != nil {
		t.Errorf("Failed to unmarshal payload: %v", err)
	}
	if string(decoded.WaveId) != string(wave.WaveId) {
		t.Error("Decoded WaveId mismatch")
	}
}

func TestMaxHopsConstant(t *testing.T) {
	// Per WAVES.md, max hops should be 20.
	if MaxHops != 20 {
		t.Errorf("MaxHops = %d, want 20", MaxHops)
	}
}

func TestGossipRelayWithConfig(t *testing.T) {
	pub := NewMockPublisher(TopicWaves)

	cfg := RelayConfig{
		MaxHops:  15,
		CacheTTL: time.Hour,
		Handler: func(_ *pb.Wave) {
			// Handler registered but not invoked in this test.
		},
	}

	gr := NewGossipRelayWithConfig(cfg, pub)

	if gr == nil {
		t.Fatal("NewGossipRelayWithConfig returned nil")
	}
	if gr.maxHops != 15 {
		t.Errorf("maxHops = %d, want 15", gr.maxHops)
	}
	if gr.cacheTTL != time.Hour {
		t.Errorf("cacheTTL = %v, want %v", gr.cacheTTL, time.Hour)
	}
}

func TestPublisherInterface(t *testing.T) {
	// Verify MockPublisher implements Publisher interface.
	var _ Publisher = (*MockPublisher)(nil)
}
