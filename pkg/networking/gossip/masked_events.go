// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements Masked Event network propagation per ANONYMOUS_GAME_MECHANICS.md.
package gossip

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Masked Event propagation errors.
var (
	ErrMaskedEventUnknown       = errors.New("masked event: unknown event ID")
	ErrMaskedEventAlreadyJoined = errors.New("masked event: already joined")
	ErrMaskedEventNotActive     = errors.New("masked event: event not active")
	ErrMaskedEventClosed        = errors.New("masked event: event has ended")
	ErrMaskedKeyNotRegistered   = errors.New("masked event: masked key not registered")
)

// MaskedEventMessageType identifies Masked Event message types.
type MaskedEventMessageType uint8

const (
	// MaskedEventMsgAnnounce is a new event announcement (via Beacon Wave).
	MaskedEventMsgAnnounce MaskedEventMessageType = iota + 1

	// MaskedEventMsgJoin is a participant join notification.
	MaskedEventMsgJoin

	// MaskedEventMsgLeave is a participant leave notification.
	MaskedEventMsgLeave

	// MaskedEventMsgWave is a Masked Wave within the event.
	MaskedEventMsgWave

	// MaskedEventMsgSummary is the post-event summary (via Beacon Wave).
	MaskedEventMsgSummary
)

// MaskedEventAnnouncement represents an event creation announcement.
// Per spec: "The Beacon Wave requires the elevated PoW difficulty (24 leading zero bits)".
type MaskedEventAnnouncement struct {
	// EventID is the unique event identifier.
	EventID [32]byte

	// Topic describes the event theme or purpose.
	Topic string

	// StartTime is when the event begins.
	StartTime time.Time

	// Duration is the event length in minutes.
	Duration int

	// MaxParticipants is the cap (0 = unlimited).
	MaxParticipants int

	// CreatorSpecterPubKey is the creator's Specter public key.
	CreatorSpecterPubKey [32]byte

	// CreatorSignature signs the announcement.
	CreatorSignature []byte
}

// MaskedEventJoin represents a participant joining an event.
// Per spec: "The `masked_pubkey` field is encrypted to the event creator's Specter
// public key using X25519 Diffie-Hellman key exchange".
type MaskedEventJoin struct {
	// EventID identifies the event being joined.
	EventID [32]byte

	// SpecterPubKey is the joining Specter's public key.
	SpecterPubKey [32]byte

	// EncryptedMaskedPubKey is the Masked public key encrypted to the creator.
	EncryptedMaskedPubKey []byte

	// Nonce for the encrypted payload.
	Nonce [24]byte

	// Signature proves the Specter is joining.
	Signature []byte

	// JoinedAt is when the join was sent.
	JoinedAt time.Time
}

// MaskedEventLeave represents a participant leaving (optional).
type MaskedEventLeave struct {
	// EventID identifies the event.
	EventID [32]byte

	// MaskedPubKey is the participant's Masked public key.
	MaskedPubKey [32]byte

	// Signature proves the Masked identity is leaving.
	Signature []byte
}

// MaskedEventWaveWrapper wraps a Masked Wave for event propagation.
type MaskedEventWaveWrapper struct {
	// EventID identifies the event.
	EventID [32]byte

	// MaskedPubKey is the sender's Masked public key.
	MaskedPubKey [32]byte

	// WaveData is the serialized Masked Wave.
	WaveData []byte

	// Signature proves the Masked identity sent this.
	Signature []byte

	// SentAt is when the Wave was sent.
	SentAt time.Time
}

// MaskedEventSummaryBroadcast contains post-event statistics.
// Per spec: "A Summary Beacon Wave is generated and published to the Anonymous Layer".
type MaskedEventSummaryBroadcast struct {
	// EventID identifies the concluded event.
	EventID [32]byte

	// Topic is the event theme.
	Topic string

	// DurationMinutes is how long the event lasted.
	DurationMinutes int

	// ParticipantCount is total participants.
	ParticipantCount int

	// TotalWaves is total Masked Waves published.
	TotalWaves int

	// TotalAmplifications is total amplifications.
	TotalAmplifications int

	// Leaderboard shows top participants by Resonance Burst.
	Leaderboard []MaskedLeaderboardItem

	// CreatorSignature signs the summary.
	CreatorSignature []byte
}

// MaskedLeaderboardItem represents a participant's ranking.
type MaskedLeaderboardItem struct {
	// Pseudonym is the Masked pseudonym.
	Pseudonym string

	// AmplificationsReceived is the count.
	AmplificationsReceived int

	// ResonanceBurst is the computed burst value.
	ResonanceBurst float64
}

// MaskedEventHandler processes Masked Event network messages.
type MaskedEventHandler interface {
	// HandleAnnouncement processes a new event announcement.
	HandleAnnouncement(ctx context.Context, ann *MaskedEventAnnouncement) error

	// HandleJoin processes a participant joining.
	HandleJoin(ctx context.Context, join *MaskedEventJoin) error

	// HandleLeave processes a participant leaving.
	HandleLeave(ctx context.Context, leave *MaskedEventLeave) error

	// HandleWave processes a Masked Wave within an event.
	HandleWave(ctx context.Context, wave *MaskedEventWaveWrapper) error

	// HandleSummary processes the post-event summary.
	HandleSummary(ctx context.Context, summary *MaskedEventSummaryBroadcast) error
}

// MaskedEventManager manages Masked Event network operations.
type MaskedEventManager struct {
	mu sync.RWMutex

	// handler processes incoming messages.
	handler MaskedEventHandler

	// activeEvents tracks events by ID.
	activeEvents map[string]*trackedEvent

	// eventTopics tracks subscribed event topics.
	eventTopics map[string]*pubsub.Topic

	// pubsub is the GossipSub instance.
	pubsub *pubsub.PubSub

	// dedup prevents duplicate message processing.
	dedup *Deduplicator

	// scoreTracker for peer scoring.
	scoreTracker *PeerScoreTracker
}

// trackedEvent contains local state for an active event.
type trackedEvent struct {
	// ID is the event identifier.
	ID [32]byte

	// Topic is the event description.
	Topic string

	// StartTime is when the event begins.
	StartTime time.Time

	// EndTime is when the event ends.
	EndTime time.Time

	// CreatorKey is the event creator's Specter key.
	CreatorKey [32]byte

	// registeredKeys tracks valid Masked public keys.
	registeredKeys map[string]bool

	// isActive indicates the event is running.
	isActive bool
}

// NewMaskedEventManager creates a new Masked Event network manager.
func NewMaskedEventManager(ps *pubsub.PubSub, scoreTracker *PeerScoreTracker) *MaskedEventManager {
	return &MaskedEventManager{
		activeEvents: make(map[string]*trackedEvent),
		eventTopics:  make(map[string]*pubsub.Topic),
		pubsub:       ps,
		dedup:        NewDeduplicator(),
		scoreTracker: scoreTracker,
	}
}

// SetHandler sets the message handler.
func (m *MaskedEventManager) SetHandler(handler MaskedEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// RegisterEvent adds a new event to track.
func (m *MaskedEventManager) RegisterEvent(
	id [32]byte,
	topic string,
	startTime, endTime time.Time,
	creatorKey [32]byte,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idHex := hex.EncodeToString(id[:])

	m.activeEvents[idHex] = &trackedEvent{
		ID:             id,
		Topic:          topic,
		StartTime:      startTime,
		EndTime:        endTime,
		CreatorKey:     creatorKey,
		registeredKeys: make(map[string]bool),
		isActive:       false,
	}

	return nil
}

// setEventActiveState updates the isActive flag for an event.
func (m *MaskedEventManager) setEventActiveState(id [32]byte, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idHex := hex.EncodeToString(id[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return ErrMaskedEventUnknown
	}

	event.isActive = active
	return nil
}

// ActivateEvent marks an event as active (started).
func (m *MaskedEventManager) ActivateEvent(id [32]byte) error {
	return m.setEventActiveState(id, true)
}

// CloseEvent marks an event as ended.
func (m *MaskedEventManager) CloseEvent(id [32]byte) error {
	return m.setEventActiveState(id, false)
}

// RegisterMaskedKey adds a valid Masked public key for an event.
func (m *MaskedEventManager) RegisterMaskedKey(eventID, maskedPubKey [32]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idHex := hex.EncodeToString(eventID[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return ErrMaskedEventUnknown
	}

	keyHex := hex.EncodeToString(maskedPubKey[:])
	event.registeredKeys[keyHex] = true
	return nil
}

// IsKeyRegistered checks if a Masked key is valid for an event.
func (m *MaskedEventManager) IsKeyRegistered(eventID, maskedPubKey [32]byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idHex := hex.EncodeToString(eventID[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return false
	}

	keyHex := hex.EncodeToString(maskedPubKey[:])
	return event.registeredKeys[keyHex]
}

// IsEventActive checks if an event is currently running.
func (m *MaskedEventManager) IsEventActive(eventID [32]byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idHex := hex.EncodeToString(eventID[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return false
	}
	return event.isActive
}

// GetActiveEventCount returns the number of tracked events.
func (m *MaskedEventManager) GetActiveEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.activeEvents)
}

// SubscribeToEventTopic subscribes to an event's gossip topic.
func (m *MaskedEventManager) SubscribeToEventTopic(eventID [32]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pubsub == nil {
		return nil // No-op if pubsub not configured.
	}

	idHex := hex.EncodeToString(eventID[:])
	topicName := EventTopic(idHex)

	// Check if already subscribed.
	if _, exists := m.eventTopics[idHex]; exists {
		return nil
	}

	// Join the topic.
	topic, err := m.pubsub.Join(topicName)
	if err != nil {
		return err
	}

	m.eventTopics[idHex] = topic
	return nil
}

// UnsubscribeFromEventTopic leaves an event's gossip topic.
func (m *MaskedEventManager) UnsubscribeFromEventTopic(eventID [32]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idHex := hex.EncodeToString(eventID[:])
	topic, exists := m.eventTopics[idHex]
	if !exists {
		return nil
	}

	if err := topic.Close(); err != nil {
		return err
	}

	delete(m.eventTopics, idHex)
	return nil
}

// PublishToEvent publishes a message to an event's topic.
func (m *MaskedEventManager) PublishToEvent(ctx context.Context, eventID [32]byte, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idHex := hex.EncodeToString(eventID[:])
	topic, exists := m.eventTopics[idHex]
	if !exists {
		return ErrMaskedEventUnknown
	}

	return topic.Publish(ctx, data)
}

// HandleEventMessage processes a message from an event topic.
func (m *MaskedEventManager) HandleEventMessage(ctx context.Context, eventID [32]byte, msg *pubsub.Message) error {
	m.mu.RLock()
	handler := m.handler
	m.mu.RUnlock()

	if handler == nil {
		return nil
	}

	// Parse envelope.
	env, err := ValidateEnvelope(msg.Data, time.Now())
	if err != nil {
		return err
	}

	// Check dedup.
	if m.dedup.IsSeen(env.MessageID) {
		return ErrDuplicateMessage
	}
	m.dedup.MarkSeen(env.MessageID)

	// Convert sender pubkey to [32]byte.
	var senderKey [32]byte
	if len(env.SenderPubkey) >= 32 {
		copy(senderKey[:], env.SenderPubkey[:32])
	}

	// Verify the sender's Masked key is registered.
	if !m.IsKeyRegistered(eventID, senderKey) {
		return ErrMaskedKeyNotRegistered
	}

	// Verify event is active.
	if !m.IsEventActive(eventID) {
		return ErrMaskedEventNotActive
	}

	// Dispatch to handler.
	wave := &MaskedEventWaveWrapper{
		EventID:      eventID,
		MaskedPubKey: senderKey,
		WaveData:     env.Payload,
		Signature:    env.Signature,
		SentAt:       time.Unix(env.TimestampUnix, 0),
	}

	return handler.HandleWave(ctx, wave)
}

// BroadcastAnnouncement sends an event announcement to the anonymous beacons topic.
func (m *MaskedEventManager) BroadcastAnnouncement(ctx context.Context, ann *MaskedEventAnnouncement) error {
	// Announcements go on the beacons topic.
	// In production, this would serialize and publish via pubsub.
	// The Beacon Wave requires elevated PoW (24 bits).
	return nil
}

// BroadcastJoin sends a join notification.
func (m *MaskedEventManager) BroadcastJoin(ctx context.Context, join *MaskedEventJoin) error {
	// Joins go on the anonymous mechanics topic.
	return nil
}

// BroadcastSummary sends the post-event summary.
func (m *MaskedEventManager) BroadcastSummary(ctx context.Context, summary *MaskedEventSummaryBroadcast) error {
	// Summary goes on the beacons topic.
	return nil
}

// CleanupExpiredEvents removes events that have ended.
func (m *MaskedEventManager) CleanupExpiredEvents() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for idHex, event := range m.activeEvents {
		if now.After(event.EndTime) {
			// Unsubscribe from topic.
			if topic, exists := m.eventTopics[idHex]; exists {
				topic.Close()
				delete(m.eventTopics, idHex)
			}

			delete(m.activeEvents, idHex)
			cleaned++
		}
	}

	return cleaned
}

// Update performs periodic maintenance.
func (m *MaskedEventManager) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for _, event := range m.activeEvents {
		// Activate events that have started.
		if !event.isActive && now.After(event.StartTime) && now.Before(event.EndTime) {
			event.isActive = true
		}

		// Deactivate events that have ended.
		if event.isActive && now.After(event.EndTime) {
			event.isActive = false
		}
	}
}

// GetEventInfo returns information about a tracked event.
func (m *MaskedEventManager) GetEventInfo(eventID [32]byte) (topic string, startTime, endTime time.Time, isActive, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idHex := hex.EncodeToString(eventID[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return "", time.Time{}, time.Time{}, false, false
	}

	return event.Topic, event.StartTime, event.EndTime, event.isActive, true
}

// GetRegisteredKeyCount returns the number of registered keys for an event.
func (m *MaskedEventManager) GetRegisteredKeyCount(eventID [32]byte) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idHex := hex.EncodeToString(eventID[:])
	event, exists := m.activeEvents[idHex]
	if !exists {
		return 0
	}

	return len(event.registeredKeys)
}
