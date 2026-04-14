// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements peer scoring integration with message validation.
// Per DESIGN_DOCUMENT.md Part II §7, peer scoring penalizes invalid signatures,
// failed PoW, expired TTL, and rewards valid message deliveries.
package gossip

import (
	"context"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerScoreWeights defines weights for scoring behaviors.
// Per DESIGN_DOCUMENT.md Part II §7.
const (
	// WeightValidMessage rewards peers for delivering valid messages.
	WeightValidMessage = 1.0

	// WeightInvalidSignature penalizes peers for invalid signatures.
	WeightInvalidSignature = -10.0

	// WeightInvalidTimestamp penalizes peers for messages with bad timestamps.
	WeightInvalidTimestamp = -5.0

	// WeightDuplicateMessage slightly penalizes duplicate delivery.
	WeightDuplicateMessage = -0.1

	// WeightInvalidPayload penalizes malformed messages.
	WeightInvalidPayload = -5.0

	// WeightInvalidPoW penalizes insufficient Proof of Work.
	WeightInvalidPoW = -10.0

	// WeightExpiredTTL penalizes messages past their TTL.
	WeightExpiredTTL = -2.0
)

// ScoreDecayInterval is how often scores decay.
const ScoreDecayInterval = time.Minute

// ScoreDecayFactor is multiplied with scores each decay interval.
const ScoreDecayFactor = 0.95

// PeerScoreTracker tracks peer scores based on message validation results.
type PeerScoreTracker struct {
	scores   map[peer.ID]*peerScoreEntry
	mu       sync.RWMutex
	callback AppSpecificScoreCallback
}

// peerScoreEntry holds scoring data for a single peer.
type peerScoreEntry struct {
	ValidMessages   int64
	InvalidMessages int64
	DuplicateCount  int64
	LastSeen        time.Time
	Score           float64
}

// AppSpecificScoreCallback is called to report score changes.
type AppSpecificScoreCallback func(p peer.ID, score float64)

// NewPeerScoreTracker creates a new peer score tracker.
func NewPeerScoreTracker() *PeerScoreTracker {
	pst := &PeerScoreTracker{
		scores: make(map[peer.ID]*peerScoreEntry),
	}
	return pst
}

// SetCallback sets the callback for score changes.
func (pst *PeerScoreTracker) SetCallback(cb AppSpecificScoreCallback) {
	pst.mu.Lock()
	defer pst.mu.Unlock()
	pst.callback = cb
}

// RecordValidMessage records a valid message from a peer.
func (pst *PeerScoreTracker) RecordValidMessage(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.ValidMessages++
	entry.Score += WeightValidMessage
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordInvalidSignature records an invalid signature from a peer.
func (pst *PeerScoreTracker) RecordInvalidSignature(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.InvalidMessages++
	entry.Score += WeightInvalidSignature
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordInvalidTimestamp records an invalid timestamp from a peer.
func (pst *PeerScoreTracker) RecordInvalidTimestamp(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.InvalidMessages++
	entry.Score += WeightInvalidTimestamp
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordDuplicateMessage records a duplicate message from a peer.
func (pst *PeerScoreTracker) RecordDuplicateMessage(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.DuplicateCount++
	entry.Score += WeightDuplicateMessage
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordInvalidPayload records an invalid payload from a peer.
func (pst *PeerScoreTracker) RecordInvalidPayload(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.InvalidMessages++
	entry.Score += WeightInvalidPayload
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordInvalidPoW records insufficient Proof of Work from a peer.
func (pst *PeerScoreTracker) RecordInvalidPoW(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.InvalidMessages++
	entry.Score += WeightInvalidPoW
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// RecordExpiredTTL records an expired TTL message from a peer.
func (pst *PeerScoreTracker) RecordExpiredTTL(p peer.ID) {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	entry := pst.getOrCreateEntry(p)
	entry.InvalidMessages++
	entry.Score += WeightExpiredTTL
	entry.LastSeen = time.Now()

	if pst.callback != nil {
		pst.callback(p, entry.Score)
	}
}

// GetScore returns the current score for a peer.
func (pst *PeerScoreTracker) GetScore(p peer.ID) float64 {
	pst.mu.RLock()
	defer pst.mu.RUnlock()

	if entry, ok := pst.scores[p]; ok {
		return entry.Score
	}
	return 0
}

// GetStats returns statistics for a peer.
func (pst *PeerScoreTracker) GetStats(p peer.ID) (valid, invalid, duplicate int64, score float64) {
	pst.mu.RLock()
	defer pst.mu.RUnlock()

	if entry, ok := pst.scores[p]; ok {
		return entry.ValidMessages, entry.InvalidMessages, entry.DuplicateCount, entry.Score
	}
	return 0, 0, 0, 0
}

// getOrCreateEntry returns or creates a score entry for a peer.
// Caller must hold the lock.
func (pst *PeerScoreTracker) getOrCreateEntry(p peer.ID) *peerScoreEntry {
	if entry, ok := pst.scores[p]; ok {
		return entry
	}
	entry := &peerScoreEntry{LastSeen: time.Now()}
	pst.scores[p] = entry
	return entry
}

// DecayScores applies decay to all peer scores.
func (pst *PeerScoreTracker) DecayScores() {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	for _, entry := range pst.scores {
		entry.Score *= ScoreDecayFactor
	}
}

// PruneInactive removes peers not seen for the given duration.
func (pst *PeerScoreTracker) PruneInactive(maxAge time.Duration) int {
	pst.mu.Lock()
	defer pst.mu.Unlock()

	pruned := 0
	cutoff := time.Now().Add(-maxAge)

	for p, entry := range pst.scores {
		if entry.LastSeen.Before(cutoff) {
			delete(pst.scores, p)
			pruned++
		}
	}
	return pruned
}

// Size returns the number of tracked peers.
func (pst *PeerScoreTracker) Size() int {
	pst.mu.RLock()
	defer pst.mu.RUnlock()
	return len(pst.scores)
}

// ValidatingMessageHandlers wraps MessageHandlers with peer scoring.
type ValidatingMessageHandlers struct {
	*MessageHandlers
	scoreTracker *PeerScoreTracker
}

// NewValidatingMessageHandlers creates handlers with peer scoring integration.
func NewValidatingMessageHandlers(scoreTracker *PeerScoreTracker) *ValidatingMessageHandlers {
	return &ValidatingMessageHandlers{
		MessageHandlers: NewMessageHandlers(),
		scoreTracker:    scoreTracker,
	}
}

// HandleMessage validates and processes a message, updating peer scores.
func (vh *ValidatingMessageHandlers) HandleMessage(ctx context.Context, topic string, msg *pubsub.Message) error {
	// Get sender peer ID
	senderID := msg.GetFrom()

	// Validate envelope
	env, err := ValidateEnvelope(msg.Data, time.Now())
	if err != nil {
		// Record the specific validation failure
		vh.recordValidationError(senderID, err)
		return err
	}

	// Check for duplicates
	if vh.dedup.IsSeen(env.MessageID) {
		vh.scoreTracker.RecordDuplicateMessage(senderID)
		return ErrDuplicateMessage
	}
	vh.dedup.MarkSeen(env.MessageID)

	// Valid message - record success
	vh.scoreTracker.RecordValidMessage(senderID)

	// Continue with normal handler dispatch
	return vh.MessageHandlers.handleMessageWithEnvelope(ctx, topic, msg, env)
}

// recordValidationError records the appropriate penalty for each error type.
func (vh *ValidatingMessageHandlers) recordValidationError(p peer.ID, err error) {
	var validationErr *ValidationError
	if ve, ok := err.(*ValidationError); ok {
		validationErr = ve
	} else {
		vh.scoreTracker.RecordInvalidPayload(p)
		return
	}

	switch validationErr {
	case ErrInvalidSignature:
		vh.scoreTracker.RecordInvalidSignature(p)
	case ErrInvalidTimestamp:
		vh.scoreTracker.RecordInvalidTimestamp(p)
	case ErrInvalidPayload, ErrInvalidVersion, ErrInvalidMessageID, ErrMissingPublicKey:
		vh.scoreTracker.RecordInvalidPayload(p)
	default:
		vh.scoreTracker.RecordInvalidPayload(p)
	}
}

// CreateValidatingTopicHandler creates a handler that updates peer scores.
func (vh *ValidatingMessageHandlers) CreateValidatingTopicHandler(topic string) MessageHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		_ = vh.HandleMessage(ctx, topic, msg)
	}
}

// handleMessageWithEnvelope dispatches to topic handlers with pre-validated envelope.
func (h *MessageHandlers) handleMessageWithEnvelope(ctx context.Context, topic string, msg *pubsub.Message, env *Envelope) error {
	// Parse GossipMessage
	var gossipMsg GossipMessage
	if err := unmarshalGossipMessage(msg.Data, &gossipMsg); err != nil {
		return ErrInvalidPayload
	}

	// Dispatch to appropriate handler based on topic
	switch topic {
	case TopicWaves:
		return h.handleWaveTopicWithMsg(ctx, env, &gossipMsg)
	case TopicIdentity:
		return h.handleIdentityTopicWithMsg(ctx, env, &gossipMsg)
	case TopicShroud:
		return h.handleShroudTopicWithMsg(ctx, env, &gossipMsg)
	case TopicPulse:
		return h.handlePulseTopicWithMsg(ctx, env, &gossipMsg)
	default:
		return ErrInvalidPayload
	}
}

// GossipMessage is an interface to avoid import cycle with proto package.
type GossipMessage interface{}

// unmarshalGossipMessage unmarshals raw data into a GossipMessage.
func unmarshalGossipMessage(data []byte, msg *GossipMessage) error {
	// The actual implementation uses proto.Unmarshal
	// This is a placeholder to avoid import cycles
	return nil
}

// handleWaveTopicWithMsg handles wave topic with parsed message.
func (h *MessageHandlers) handleWaveTopicWithMsg(ctx context.Context, env *Envelope, _ *GossipMessage) error {
	h.mu.RLock()
	handler := h.waveHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	// Note: In full implementation, pass parsed proto message
	return handler.HandleWave(ctx, env, nil)
}

// handleIdentityTopicWithMsg handles identity topic with parsed message.
func (h *MessageHandlers) handleIdentityTopicWithMsg(ctx context.Context, env *Envelope, _ *GossipMessage) error {
	h.mu.RLock()
	handler := h.identityHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandleIdentity(ctx, env, nil)
}

// handleShroudTopicWithMsg handles shroud topic with parsed message.
func (h *MessageHandlers) handleShroudTopicWithMsg(ctx context.Context, env *Envelope, _ *GossipMessage) error {
	h.mu.RLock()
	handler := h.shroudHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandleShroud(ctx, env, nil)
}

// handlePulseTopicWithMsg handles pulse topic with parsed message.
func (h *MessageHandlers) handlePulseTopicWithMsg(ctx context.Context, env *Envelope, _ *GossipMessage) error {
	h.mu.RLock()
	handler := h.pulseHandler
	h.mu.RUnlock()

	if handler == nil {
		return nil
	}
	return handler.HandlePulse(ctx, env, nil)
}

// AppSpecificScoreFunc returns a GossipSub app-specific score function.
// This integrates the PeerScoreTracker with GossipSub's scoring system.
func (pst *PeerScoreTracker) AppSpecificScoreFunc() func(p peer.ID) float64 {
	return func(p peer.ID) float64 {
		return pst.GetScore(p)
	}
}

// StartDecayLoop starts a goroutine that periodically decays scores.
func (pst *PeerScoreTracker) StartDecayLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(ScoreDecayInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pst.DecayScores()
			}
		}
	}()
}
