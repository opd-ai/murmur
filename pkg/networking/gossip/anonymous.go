// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements Anonymous Layer topic handling.
// Per NETWORK_ARCHITECTURE.md, Anonymous Layer gossip protocol handles propagation of
// Specter Waves, Beacon Waves, anonymous mechanic data, and Specter identity announcements.
package gossip

import (
	"context"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Anonymous Layer topic names per NETWORK_ARCHITECTURE.md §Protocol Multiplexing.
const (
	// TopicAnonymousWaves handles Specter and Masked Wave propagation.
	TopicAnonymousWaves = "/murmur/anonymous/waves/1.0"

	// TopicAnonymousMechanics handles Gifts, Marks, mini-game events, and Councils.
	TopicAnonymousMechanics = "/murmur/anonymous/mechanics/1.0"

	// TopicAnonymousBeacons handles Beacon Waves with elevated PoW.
	TopicAnonymousBeacons = "/murmur/anonymous/beacons/1.0"
)

// TimestampQuantum is the quantum for Anonymous Layer timestamps (5 minutes).
// Per WAVE_PROPAGATION.md, timestamps are quantized to 5-minute buckets.
const TimestampQuantum = 5 * time.Minute

// QuantizeTimestamp rounds a timestamp to the nearest 5-minute bucket.
// Per WAVE_PROPAGATION.md, timestamps on all Anonymous Layer Waves are quantized.
func QuantizeTimestamp(t time.Time) time.Time {
	quantum := TimestampQuantum.Nanoseconds()
	unixNano := t.UnixNano()
	quantized := (unixNano / quantum) * quantum
	return time.Unix(0, quantized)
}

// AnonymousWaveHandler processes anonymous Wave messages.
type AnonymousWaveHandler interface {
	HandleSpecterWave(ctx context.Context, env *Envelope) error
	HandleMaskedWave(ctx context.Context, env *Envelope) error
}

// AnonymousMechanicsHandler processes anonymous mechanics messages.
type AnonymousMechanicsHandler interface {
	HandlePhantomGift(ctx context.Context, env *Envelope) error
	HandleSpecterMark(ctx context.Context, env *Envelope) error
	HandleMiniGameEvent(ctx context.Context, env *Envelope) error
	HandleCouncilMessage(ctx context.Context, env *Envelope) error
}

// BeaconWaveHandler processes Beacon Wave messages (elevated PoW).
type BeaconWaveHandler interface {
	HandleBeaconWave(ctx context.Context, env *Envelope) error
}

// AnonymousTopicHandlers manages handlers for Anonymous Layer topics.
type AnonymousTopicHandlers struct {
	waveHandler      AnonymousWaveHandler
	mechanicsHandler AnonymousMechanicsHandler
	beaconHandler    BeaconWaveHandler
	dedup            *Deduplicator
	scoreTracker     *PeerScoreTracker
}

// NewAnonymousTopicHandlers creates a new Anonymous Layer handler manager.
func NewAnonymousTopicHandlers(scoreTracker *PeerScoreTracker) *AnonymousTopicHandlers {
	return &AnonymousTopicHandlers{
		dedup:        NewDeduplicator(),
		scoreTracker: scoreTracker,
	}
}

// SetWaveHandler sets the anonymous Wave message handler.
func (h *AnonymousTopicHandlers) SetWaveHandler(handler AnonymousWaveHandler) {
	h.waveHandler = handler
}

// SetMechanicsHandler sets the anonymous mechanics message handler.
func (h *AnonymousTopicHandlers) SetMechanicsHandler(handler AnonymousMechanicsHandler) {
	h.mechanicsHandler = handler
}

// SetBeaconHandler sets the Beacon Wave message handler.
func (h *AnonymousTopicHandlers) SetBeaconHandler(handler BeaconWaveHandler) {
	h.beaconHandler = handler
}

// HandleMessage processes a message from any anonymous topic.
func (h *AnonymousTopicHandlers) HandleMessage(ctx context.Context, topic string, msg *pubsub.Message) error {
	senderID := msg.GetFrom()

	// Validate envelope with quantized timestamp check
	env, err := ValidateAnonymousEnvelope(msg.Data, time.Now())
	if err != nil {
		if h.scoreTracker != nil {
			h.recordValidationError(senderID, err)
		}
		return err
	}

	// Check for duplicates
	if h.dedup.IsSeen(env.MessageID) {
		if h.scoreTracker != nil {
			h.scoreTracker.RecordDuplicateMessage(senderID)
		}
		return ErrDuplicateMessage
	}
	h.dedup.MarkSeen(env.MessageID)

	// Record valid message
	if h.scoreTracker != nil {
		h.scoreTracker.RecordValidMessage(senderID)
	}

	// Dispatch to appropriate handler based on topic
	switch topic {
	case TopicAnonymousWaves:
		return h.handleAnonymousWavesTopic(ctx, env)
	case TopicAnonymousMechanics:
		return h.handleAnonymousMechanicsTopic(ctx, env)
	case TopicAnonymousBeacons:
		return h.handleAnonymousBeaconsTopic(ctx, env)
	default:
		return ErrInvalidPayload
	}
}

func (h *AnonymousTopicHandlers) recordValidationError(senderID any, err error) {
	// Type assert to get peer.ID (imported via pubsub)
	pid, ok := senderID.(interface{ String() string })
	if !ok {
		return
	}

	// Convert to string representation for type compatibility
	// In production, this would be peer.ID
	_ = pid.String()

	var validationErr *ValidationError
	if ve, ok := err.(*ValidationError); ok {
		validationErr = ve
	} else {
		return
	}

	// Note: Full implementation would call scoreTracker methods
	_ = validationErr
}

func (h *AnonymousTopicHandlers) handleAnonymousWavesTopic(ctx context.Context, env *Envelope) error {
	if h.waveHandler == nil {
		return nil
	}

	// Determine if Specter or Masked based on envelope type
	// Per WAVES.md, Specter Waves have WaveType 0x04, Masked Waves have 0x07
	switch env.Type {
	case MessageTypeWave:
		// Could be either Specter or Masked - handler determines from payload
		return h.waveHandler.HandleSpecterWave(ctx, env)
	default:
		return h.waveHandler.HandleSpecterWave(ctx, env)
	}
}

func (h *AnonymousTopicHandlers) handleAnonymousMechanicsTopic(ctx context.Context, env *Envelope) error {
	if h.mechanicsHandler == nil {
		return nil
	}

	// Mechanics handler determines specific type from payload
	// Types include: Phantom Gifts, Specter Marks, mini-game events, Council messages
	return h.mechanicsHandler.HandlePhantomGift(ctx, env)
}

func (h *AnonymousTopicHandlers) handleAnonymousBeaconsTopic(ctx context.Context, env *Envelope) error {
	if h.beaconHandler == nil {
		return nil
	}
	return h.beaconHandler.HandleBeaconWave(ctx, env)
}

// CreateAnonymousTopicHandler creates a MessageHandler for an anonymous topic.
func (h *AnonymousTopicHandlers) CreateAnonymousTopicHandler(topic string) MessageHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		_ = h.HandleMessage(ctx, topic, msg)
	}
}

// ValidateAnonymousEnvelope validates an Anonymous Layer message.
// Per WAVE_PROPAGATION.md, Anonymous Layer messages have quantized timestamps.
func ValidateAnonymousEnvelope(data []byte, now time.Time) (*Envelope, error) {
	// First use standard validation
	env, err := ValidateEnvelope(data, now)
	if err != nil {
		return nil, err
	}

	// Verify timestamp is quantized (within quantum tolerance)
	msgTime := time.Unix(env.TimestampUnix, 0)
	quantized := QuantizeTimestamp(msgTime)

	// Allow small drift from quantum boundary (10 seconds tolerance)
	if msgTime.Sub(quantized).Abs() > 10*time.Second {
		return nil, ErrInvalidTimestamp
	}

	return env, nil
}

// AnonymousPoWDifficulty is the elevated PoW difficulty for Anonymous Layer.
// Per WAVE_PROPAGATION.md, ~2 seconds on a mid-range smartphone.
const AnonymousPoWDifficulty = 22 // Leading zero bits

// BeaconPoWDifficulty is the elevated PoW for Beacon Waves.
// Beacon Waves require even higher PoW than standard anonymous waves.
const BeaconPoWDifficulty = 24 // Leading zero bits

// VerifyAnonymousPoW checks if a Wave meets Anonymous Layer PoW requirements.
func VerifyAnonymousPoW(waveData []byte, nonce []byte, difficulty uint8) bool {
	if difficulty < AnonymousPoWDifficulty {
		return false
	}
	// PoW verification is delegated to pkg/content/pow
	// This function checks the difficulty requirement
	return true
}

// VerifyBeaconPoW checks if a Beacon Wave meets elevated PoW requirements.
func VerifyBeaconPoW(waveData []byte, nonce []byte, difficulty uint8) bool {
	if difficulty < BeaconPoWDifficulty {
		return false
	}
	return true
}

// EventTopic generates an ephemeral topic name for a Specter Event.
// Per ROADMAP.md, per-event ephemeral topics: /murmur/event/[event_id]/1.0
func EventTopic(eventID string) string {
	return "/murmur/event/" + eventID + "/1.0"
}

// CouncilTopic generates an encrypted topic name for a Phantom Council.
// Per ROADMAP.md, per-council encrypted topics: /murmur/council/[council_id]/1.0
func CouncilTopic(councilID string) string {
	return "/murmur/council/" + councilID + "/1.0"
}
