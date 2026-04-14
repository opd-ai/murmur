// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements message handlers for each GossipSub topic.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, handlers validate envelopes, verify PoW,
// check timestamps, and dispatch to storage.
package gossip

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/zeebo/blake3"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// Message validation constants per TECHNICAL_IMPLEMENTATION.md.
const (
	// MaxTimestampDrift is the maximum allowed timestamp drift (±300 seconds).
	MaxTimestampDrift = 300 * time.Second

	// CurrentProtocolVersion is the current envelope protocol version.
	CurrentProtocolVersion = 1

	// PublicKeySize is the size of Ed25519 public keys.
	PublicKeySize = 32

	// SignatureSize is the size of Ed25519 signatures.
	SignatureSize = 64

	// MessageIDSize is the size of BLAKE3 message IDs.
	MessageIDSize = 32
)

// MessageType indicates the type of message in the envelope.
type MessageType int

const (
	MessageTypeWave MessageType = iota + 1
	MessageTypeIdentity
	MessageTypeShroud
	MessageTypeHeartbeat
)

// ValidationError indicates a message validation failure.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Code + ": " + e.Message
}

var (
	ErrInvalidVersion   = &ValidationError{"INVALID_VERSION", "unsupported protocol version"}
	ErrInvalidSignature = &ValidationError{"INVALID_SIGNATURE", "signature verification failed"}
	ErrInvalidTimestamp = &ValidationError{"INVALID_TIMESTAMP", "timestamp out of acceptable range"}
	ErrInvalidMessageID = &ValidationError{"INVALID_MESSAGE_ID", "message ID mismatch"}
	ErrDuplicateMessage = &ValidationError{"DUPLICATE_MESSAGE", "message already processed"}
	ErrInvalidPayload   = &ValidationError{"INVALID_PAYLOAD", "failed to parse payload"}
	ErrMissingPublicKey = &ValidationError{"MISSING_PUBLIC_KEY", "sender public key required"}
)

// Envelope represents a validated MurmurEnvelope.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, all GossipSub messages use this format.
type Envelope struct {
	Version       uint32
	Type          MessageType
	Payload       []byte
	SenderPubkey  []byte
	Signature     []byte
	TimestampUnix int64
	MessageID     []byte
}

// ValidateEnvelope validates a raw message and returns a parsed envelope.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, validation checks:
// 1. Protocol version
// 2. Timestamp within ±300 seconds
// 3. Signature verification
// 4. Message ID (BLAKE3 hash of payload)
func ValidateEnvelope(data []byte, now time.Time) (*Envelope, error) {
	// Parse as GossipMessage first (it's a union type)
	var msg pb.GossipMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, ErrInvalidPayload
	}

	// Create envelope from GossipMessage
	env := &Envelope{
		Version: CurrentProtocolVersion,
		Payload: data,
	}

	// Determine type from content
	switch msg.GetContent().(type) {
	case *pb.GossipMessage_Wave, *pb.GossipMessage_Reply, *pb.GossipMessage_Amplification:
		env.Type = MessageTypeWave
		if wave := msg.GetWave(); wave != nil {
			env.SenderPubkey = wave.GetAuthorPubkey()
			env.Signature = wave.GetSignature()
			env.TimestampUnix = wave.GetCreatedAt()
		} else if reply := msg.GetReply(); reply != nil {
			if reply.GetWave() != nil {
				env.SenderPubkey = reply.GetWave().GetAuthorPubkey()
				env.Signature = reply.GetWave().GetSignature()
				env.TimestampUnix = reply.GetWave().GetCreatedAt()
			}
		} else if amp := msg.GetAmplification(); amp != nil {
			env.SenderPubkey = amp.GetAmplifierPubkey()
			env.Signature = amp.GetSignature()
			env.TimestampUnix = amp.GetAmplifiedAt()
		}
	case *pb.GossipMessage_IdentityDeclaration, *pb.GossipMessage_ConnectionAnnouncement:
		env.Type = MessageTypeIdentity
		if decl := msg.GetIdentityDeclaration(); decl != nil {
			env.SenderPubkey = decl.GetPublicKey()
			env.Signature = decl.GetSignature()
			env.TimestampUnix = decl.GetCreatedAt()
		} else if conn := msg.GetConnectionAnnouncement(); conn != nil {
			env.SenderPubkey = conn.GetPublicKey()
			env.Signature = conn.GetSignature()
			env.TimestampUnix = conn.GetTimestamp()
		}
	case *pb.GossipMessage_Heartbeat:
		env.Type = MessageTypeHeartbeat
		if hb := msg.GetHeartbeat(); hb != nil {
			env.SenderPubkey = hb.GetPublicKey()
			env.Signature = hb.GetSignature()
			env.TimestampUnix = hb.GetTimestamp()
		}
	case *pb.GossipMessage_RelayAdvertisement:
		env.Type = MessageTypeShroud
		if ad := msg.GetRelayAdvertisement(); ad != nil {
			env.SenderPubkey = ad.GetEd25519Pubkey()
			env.Signature = ad.GetSignature()
			env.TimestampUnix = ad.GetTimestamp()
		}
	default:
		return nil, ErrInvalidPayload
	}

	// Compute message ID
	env.MessageID = computeMessageID(data)

	// Validate timestamp
	if err := validateTimestamp(env.TimestampUnix, now); err != nil {
		return nil, err
	}

	// Validate signature (if sender pubkey present and non-zero)
	if len(env.SenderPubkey) > 0 && len(env.Signature) > 0 && !isAllZeros(env.SenderPubkey) {
		if err := validateSignature(env); err != nil {
			return nil, err
		}
	}

	return env, nil
}

// validateTimestamp checks if timestamp is within acceptable range.
func validateTimestamp(timestampUnix int64, now time.Time) error {
	msgTime := time.Unix(timestampUnix, 0)
	drift := now.Sub(msgTime)

	// Check future drift
	if drift < -MaxTimestampDrift {
		return ErrInvalidTimestamp
	}

	// Check past drift (use TTL if available, default to MaxTimestampDrift)
	if drift > MaxTimestampDrift {
		return ErrInvalidTimestamp
	}

	return nil
}

// validateSignature verifies the Ed25519 signature.
func validateSignature(env *Envelope) error {
	if len(env.SenderPubkey) != PublicKeySize {
		return ErrMissingPublicKey
	}
	if len(env.Signature) != SignatureSize {
		return ErrInvalidSignature
	}

	// Build signed data: version || type || payload
	signedData := buildSignedData(env.Version, env.Type, env.Payload)

	pubkey := ed25519.PublicKey(env.SenderPubkey)
	if !ed25519.Verify(pubkey, signedData, env.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// buildSignedData constructs the data that was signed.
func buildSignedData(version uint32, msgType MessageType, payload []byte) []byte {
	data := make([]byte, 4+4+len(payload))
	binary.BigEndian.PutUint32(data[0:4], version)
	binary.BigEndian.PutUint32(data[4:8], uint32(msgType))
	copy(data[8:], payload)
	return data
}

// computeMessageID computes the BLAKE3 hash of the payload.
// Per TECHNICAL_IMPLEMENTATION.md, message_id is 32-byte BLAKE3 hash.
func computeMessageID(payload []byte) []byte {
	h := blake3.Sum256(payload)
	return h[:]
}

// isAllZeros checks if a byte slice is all zeros.
func isAllZeros(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// MessageHandlers manages message handlers for all topics.
type MessageHandlers struct {
	waveHandler     WaveHandler
	identityHandler IdentityHandler
	shroudHandler   ShroudHandler
	pulseHandler    PulseHandler
	dedup           *Deduplicator

	mu sync.RWMutex
}

// WaveHandler processes Wave messages from /murmur/waves/1.
type WaveHandler interface {
	HandleWave(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error
}

// IdentityHandler processes identity messages from /murmur/identity/1.
type IdentityHandler interface {
	HandleIdentity(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error
}

// ShroudHandler processes Shroud relay ads from /murmur/shroud/1.
type ShroudHandler interface {
	HandleShroud(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error
}

// PulseHandler processes heartbeat pings from /murmur/pulse/1.
type PulseHandler interface {
	HandlePulse(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error
}

// NewMessageHandlers creates a new message handlers manager.
func NewMessageHandlers() *MessageHandlers {
	return &MessageHandlers{
		dedup: NewDeduplicator(),
	}
}

// SetWaveHandler sets the Wave message handler.
func (h *MessageHandlers) SetWaveHandler(handler WaveHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.waveHandler = handler
}

// SetIdentityHandler sets the identity message handler.
func (h *MessageHandlers) SetIdentityHandler(handler IdentityHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.identityHandler = handler
}

// SetShroudHandler sets the Shroud message handler.
func (h *MessageHandlers) SetShroudHandler(handler ShroudHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shroudHandler = handler
}

// SetPulseHandler sets the pulse/heartbeat message handler.
func (h *MessageHandlers) SetPulseHandler(handler PulseHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pulseHandler = handler
}

// HandleMessage is the main entry point for processing GossipSub messages.
// It validates the envelope, deduplicates, and dispatches to the appropriate handler.
func (h *MessageHandlers) HandleMessage(ctx context.Context, topic string, msg *pubsub.Message) error {
	// Validate envelope
	env, err := ValidateEnvelope(msg.Data, time.Now())
	if err != nil {
		return err
	}

	// Check for duplicates
	if h.dedup.IsSeen(env.MessageID) {
		return ErrDuplicateMessage
	}
	h.dedup.MarkSeen(env.MessageID)

	// Parse GossipMessage
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(msg.Data, &gossipMsg); err != nil {
		return ErrInvalidPayload
	}

	// Dispatch to appropriate handler based on topic
	switch topic {
	case TopicWaves:
		return h.handleWaveTopic(ctx, env, &gossipMsg)
	case TopicIdentity:
		return h.handleIdentityTopic(ctx, env, &gossipMsg)
	case TopicShroud:
		return h.handleShroudTopic(ctx, env, &gossipMsg)
	case TopicPulse:
		return h.handlePulseTopic(ctx, env, &gossipMsg)
	default:
		return errors.New("unknown topic: " + topic)
	}
}

func (h *MessageHandlers) handleWaveTopic(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error {
	h.mu.RLock()
	handler := h.waveHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil // No handler registered
	}
	return handler.HandleWave(ctx, env, msg)
}

func (h *MessageHandlers) handleIdentityTopic(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error {
	h.mu.RLock()
	handler := h.identityHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandleIdentity(ctx, env, msg)
}

func (h *MessageHandlers) handleShroudTopic(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error {
	h.mu.RLock()
	handler := h.shroudHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandleShroud(ctx, env, msg)
}

func (h *MessageHandlers) handlePulseTopic(ctx context.Context, env *Envelope, msg *pb.GossipMessage) error {
	h.mu.RLock()
	handler := h.pulseHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandlePulse(ctx, env, msg)
}

// CreateTopicHandler creates a MessageHandler function for a specific topic.
func (h *MessageHandlers) CreateTopicHandler(topic string) MessageHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		_ = h.HandleMessage(ctx, topic, msg)
	}
}

// Deduplicator tracks seen message IDs to prevent reprocessing.
// Per TECHNICAL_IMPLEMENTATION.md, uses BLAKE3 message_id with 30-day window.
type Deduplicator struct {
	seen map[string]time.Time
	mu   sync.RWMutex
}

// DeduplicationWindow is the time window for deduplication (30 days).
const DeduplicationWindow = 30 * 24 * time.Hour

// NewDeduplicator creates a new message deduplicator.
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		seen: make(map[string]time.Time),
	}
}

// IsSeen checks if a message ID has been seen before.
func (d *Deduplicator) IsSeen(messageID []byte) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := string(messageID)
	if seenAt, ok := d.seen[key]; ok {
		// Check if still within window
		if time.Since(seenAt) < DeduplicationWindow {
			return true
		}
	}
	return false
}

// MarkSeen marks a message ID as seen.
func (d *Deduplicator) MarkSeen(messageID []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seen[string(messageID)] = time.Now()
}

// Prune removes entries older than the deduplication window.
func (d *Deduplicator) Prune() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	pruned := 0
	cutoff := time.Now().Add(-DeduplicationWindow)

	for key, seenAt := range d.seen {
		if seenAt.Before(cutoff) {
			delete(d.seen, key)
			pruned++
		}
	}

	return pruned
}

// Size returns the number of entries in the deduplicator.
func (d *Deduplicator) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.seen)
}
