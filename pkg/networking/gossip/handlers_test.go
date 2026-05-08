package gossip

import (
	"context"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestValidationError(t *testing.T) {
	err := &ValidationError{Code: "TEST", Message: "test message"}
	assert.Equal(t, "TEST: test message", err.Error())
}

func TestValidationConstants(t *testing.T) {
	assert.Equal(t, 300*time.Second, MaxTimestampDrift)
	assert.Equal(t, uint32(1), uint32(CurrentProtocolVersion))
	assert.Equal(t, 32, PublicKeySize)
	assert.Equal(t, 64, SignatureSize)
	assert.Equal(t, 32, MessageIDSize)
}

func TestComputeMessageID(t *testing.T) {
	payload := []byte("test payload")
	id := computeMessageID(payload)

	assert.Len(t, id, MessageIDSize)

	// Same payload should produce same ID
	id2 := computeMessageID(payload)
	assert.Equal(t, id, id2)

	// Different payload should produce different ID
	id3 := computeMessageID([]byte("different"))
	assert.NotEqual(t, id, id3)
}

func TestBuildSignedData(t *testing.T) {
	data := buildSignedData(1, MessageTypeWave, []byte("test"))

	// Should have 4 bytes version + 4 bytes type + payload
	assert.Len(t, data, 8+4)
}

func TestValidateTimestamp(t *testing.T) {
	now := time.Now()

	// Valid - within range
	err := validateTimestamp(now.Unix(), 0, now)
	assert.NoError(t, err)

	// Valid - slight drift
	err = validateTimestamp(now.Add(-100*time.Second).Unix(), 0, now)
	assert.NoError(t, err)

	// Invalid - too far in the future
	err = validateTimestamp(now.Add(400*time.Second).Unix(), 0, now)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTimestamp, err)

	// Invalid - too far in the past (no TTL)
	err = validateTimestamp(now.Add(-400*time.Second).Unix(), 0, now)
	assert.Error(t, err)

	// Valid - 400s old but Wave has 3600s TTL
	err = validateTimestamp(now.Add(-400*time.Second).Unix(), 3600, now)
	assert.NoError(t, err)

	// Invalid - older than Wave TTL
	err = validateTimestamp(now.Add(-7200*time.Second).Unix(), 3600, now)
	assert.Error(t, err)
}

func TestNewDeduplicator(t *testing.T) {
	d := NewDeduplicator()
	require.NotNil(t, d)
	assert.Equal(t, 0, d.Size())
}

func TestDeduplicator_MarkAndCheck(t *testing.T) {
	d := NewDeduplicator()

	msgID := []byte("test-message-id-32-bytes-xxxxxxx")

	// Not seen initially
	assert.False(t, d.IsSeen(msgID))

	// Mark as seen
	d.MarkSeen(msgID)

	// Now should be seen
	assert.True(t, d.IsSeen(msgID))
	assert.Equal(t, 1, d.Size())

	// Different ID should not be seen
	assert.False(t, d.IsSeen([]byte("different-id-32-bytes-xxxxxxxxxx")))
}

func TestDeduplicator_Prune(t *testing.T) {
	d := NewDeduplicator()

	// Mark a message
	msgID := []byte("test-message-id-32-bytes-xxxxxxx")
	d.MarkSeen(msgID)

	// Prune should remove nothing (recent)
	pruned := d.Prune()
	assert.Equal(t, 0, pruned)
	assert.Equal(t, 1, d.Size())

	// Artificially age the entry
	d.mu.Lock()
	d.seen[string(msgID)] = time.Now().Add(-31 * 24 * time.Hour)
	d.mu.Unlock()

	// Now prune should remove it
	pruned = d.Prune()
	assert.Equal(t, 1, pruned)
	assert.Equal(t, 0, d.Size())
}

func TestDeduplicationWindow(t *testing.T) {
	assert.Equal(t, 30*24*time.Hour, DeduplicationWindow)
}

func TestNewMessageHandlers(t *testing.T) {
	h := NewMessageHandlers()
	require.NotNil(t, h)
	require.NotNil(t, h.dedup)
}

func TestMessageHandlers_SetHandlers(t *testing.T) {
	h := NewMessageHandlers()

	// Set mock handlers
	waveHandler := &mockWaveHandler{}
	h.SetWaveHandler(waveHandler)

	identityHandler := &mockIdentityHandler{}
	h.SetIdentityHandler(identityHandler)

	shroudHandler := &mockShroudHandler{}
	h.SetShroudHandler(shroudHandler)

	pulseHandler := &mockPulseHandler{}
	h.SetPulseHandler(pulseHandler)

	assert.NotNil(t, h.waveHandler)
	assert.NotNil(t, h.identityHandler)
	assert.NotNil(t, h.shroudHandler)
	assert.NotNil(t, h.pulseHandler)
}

func TestMessageHandlers_CreateTopicHandler(t *testing.T) {
	h := NewMessageHandlers()
	handler := h.CreateTopicHandler(TopicWaves)
	assert.NotNil(t, handler)
}

func TestValidateEnvelope_InvalidPayload(t *testing.T) {
	_, err := ValidateEnvelope([]byte("invalid"), time.Now())
	assert.Error(t, err)
}

func TestValidateEnvelope_Wave(t *testing.T) {
	now := time.Now()

	wave := &pb.Wave{
		WaveType:     pb.WaveType_WAVE_TYPE_SURFACE,
		Content:      []byte("test content"),
		AuthorPubkey: make([]byte, 32),
		Signature:    make([]byte, 64),
		CreatedAt:    now.Unix(),
		TtlSeconds:   3600,
	}

	msg := &pb.GossipMessage{
		Content: &pb.GossipMessage_Wave{Wave: wave},
	}

	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	env, err := ValidateEnvelope(data, now)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeWave, env.Type)
	assert.NotEmpty(t, env.MessageID)
}

func TestValidateEnvelope_Heartbeat(t *testing.T) {
	now := time.Now()

	hb := &pb.Heartbeat{
		PeerId:    "QmTestPeer",
		PublicKey: make([]byte, 32),
		Timestamp: now.Unix(),
		Signature: make([]byte, 64),
		Sequence:  1,
	}

	msg := &pb.GossipMessage{
		Content: &pb.GossipMessage_Heartbeat{Heartbeat: hb},
	}

	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	env, err := ValidateEnvelope(data, now)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeHeartbeat, env.Type)
}

func TestValidateEnvelope_Identity(t *testing.T) {
	now := time.Now()

	decl := &pb.IdentityDeclaration{
		PublicKey:   make([]byte, 32),
		DisplayName: "Test User",
		CreatedAt:   now.Unix(),
		Signature:   make([]byte, 64),
	}

	msg := &pb.GossipMessage{
		Content: &pb.GossipMessage_IdentityDeclaration{IdentityDeclaration: decl},
	}

	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	env, err := ValidateEnvelope(data, now)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeIdentity, env.Type)
}

func TestValidateEnvelope_Shroud(t *testing.T) {
	now := time.Now()

	ad := &pb.RelayAdvertisement{
		Curve25519Pubkey: make([]byte, 32),
		Ed25519Pubkey:    make([]byte, 32),
		Timestamp:        now.Unix(),
		Signature:        make([]byte, 64),
	}

	msg := &pb.GossipMessage{
		Content: &pb.GossipMessage_RelayAdvertisement{RelayAdvertisement: ad},
	}

	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	env, err := ValidateEnvelope(data, now)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeShroud, env.Type)
}

func TestMessageHandlers_HandleMessage_Duplicate(t *testing.T) {
	h := NewMessageHandlers()
	waveHandler := &mockWaveHandler{}
	h.SetWaveHandler(waveHandler)

	now := time.Now()
	wave := &pb.Wave{
		WaveType:     pb.WaveType_WAVE_TYPE_SURFACE,
		Content:      []byte("test"),
		AuthorPubkey: make([]byte, 32),
		CreatedAt:    now.Unix(),
	}
	msg := &pb.GossipMessage{
		Content: &pb.GossipMessage_Wave{Wave: wave},
	}
	data, _ := proto.Marshal(msg)

	psMsg := &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}

	// First call should succeed
	err := h.HandleMessage(context.Background(), TopicWaves, psMsg)
	assert.NoError(t, err)
	assert.Equal(t, 1, waveHandler.calls)

	// Second call should be duplicate
	err = h.HandleMessage(context.Background(), TopicWaves, psMsg)
	assert.Equal(t, ErrDuplicateMessage, err)
	assert.Equal(t, 1, waveHandler.calls) // Still 1
}

// Mock handlers for testing.
type mockWaveHandler struct {
	calls int
}

func (m *mockWaveHandler) HandleWave(_ context.Context, _ *Envelope, _ *pb.GossipMessage) error {
	m.calls++
	return nil
}

type mockIdentityHandler struct{}

func (m *mockIdentityHandler) HandleIdentity(_ context.Context, _ *Envelope, _ *pb.GossipMessage) error {
	return nil
}

type mockShroudHandler struct{}

func (m *mockShroudHandler) HandleShroud(_ context.Context, _ *Envelope, _ *pb.GossipMessage) error {
	return nil
}

type mockPulseHandler struct{}

func (m *mockPulseHandler) HandlePulse(_ context.Context, _ *Envelope, _ *pb.GossipMessage) error {
	return nil
}
