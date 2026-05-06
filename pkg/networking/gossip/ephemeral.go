// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements ephemeral and encrypted topic management.
// Per ROADMAP.md, per-event ephemeral topics and per-council encrypted topics.
package gossip

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"golang.org/x/crypto/chacha20poly1305"
)

// EphemeralTopicPrefix is the prefix for event-specific ephemeral topics.
const EphemeralTopicPrefix = "/murmur/event/"

// CouncilTopicPrefix is the prefix for council-specific encrypted topics.
const CouncilTopicPrefix = "/murmur/council/"

// EphemeralTopicVersion is the version suffix for ephemeral topics.
const EphemeralTopicVersion = "/1.0"

// DefaultEventDuration is the default lifetime for event topics.
const DefaultEventDuration = 24 * time.Hour

// MaxEventDuration is the maximum allowed event duration.
const MaxEventDuration = 72 * time.Hour

// EphemeralTopicManager manages short-lived event topics.
type EphemeralTopicManager struct {
	ps       *PubSub
	topics   map[string]*ephemeralTopicEntry
	mu       sync.RWMutex
	stopCh   chan struct{}
	handlers *AnonymousTopicHandlers
}

// ephemeralTopicEntry tracks an ephemeral topic's lifecycle.
type ephemeralTopicEntry struct {
	TopicID   string
	EventID   string
	CreatedAt time.Time
	ExpiresAt time.Time
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
}

// subscribeToTopic is a helper that handles the common subscription pattern.
// It checks if the topic exists, verifies not already subscribed, and creates a subscription.
// The entry.sub field is updated before releasing the lock.
func subscribeToEphemeralTopic(
	mu *sync.RWMutex,
	topics map[string]*ephemeralTopicEntry,
	topicName string,
) (*pubsub.Subscription, error) {
	mu.Lock()
	defer mu.Unlock()

	entry, ok := topics[topicName]
	if !ok {
		return nil, ErrInvalidPayload
	}

	if entry.sub != nil {
		return nil, nil // Already subscribed
	}

	sub, err := entry.topic.Subscribe()
	if err != nil {
		return nil, err
	}

	entry.sub = sub
	return sub, nil
}

// subscribeToCouncilTopic is a helper for council topic subscription.
func subscribeToCouncilTopic(
	mu *sync.RWMutex,
	topics map[string]*councilTopicEntry,
	topicName string,
) (*pubsub.Subscription, []byte, error) {
	mu.Lock()
	defer mu.Unlock()

	entry, ok := topics[topicName]
	if !ok {
		return nil, nil, ErrInvalidPayload
	}

	if entry.sub != nil {
		return nil, nil, nil // Already subscribed
	}

	sub, err := entry.topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}

	entry.sub = sub
	return sub, entry.EncryptKey, nil
}

// cleanupTopicEntry closes the subscription and topic for a topic entry.
// Caller must hold the lock. Does not delete from map or zero keys.
func cleanupTopicEntry(entry interface{}) {
	// Handle both ephemeralTopicEntry and councilTopicEntry
	type topicEntry interface {
		getSub() *pubsub.Subscription
		getTopic() *pubsub.Topic
	}

	// Type assert to either ephemeral or council entry
	var sub *pubsub.Subscription
	var topic *pubsub.Topic

	switch e := entry.(type) {
	case *ephemeralTopicEntry:
		sub = e.sub
		topic = e.topic
	case *councilTopicEntry:
		sub = e.sub
		topic = e.topic
	default:
		return
	}

	if sub != nil {
		sub.Cancel()
	}
	if topic != nil {
		_ = topic.Close()
	}
}

// NewEphemeralTopicManager creates a new ephemeral topic manager.
func NewEphemeralTopicManager(ps *PubSub, handlers *AnonymousTopicHandlers) *EphemeralTopicManager {
	return &EphemeralTopicManager{
		ps:       ps,
		topics:   make(map[string]*ephemeralTopicEntry),
		stopCh:   make(chan struct{}),
		handlers: handlers,
	}
}

// CreateEventTopic creates an ephemeral topic for a Specter Event.
func (m *EphemeralTopicManager) CreateEventTopic(ctx context.Context, eventID string, duration time.Duration) (*pubsub.Topic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	topicName := EventTopic(eventID)

	// Check if already exists
	if entry, ok := m.topics[topicName]; ok {
		return entry.topic, nil
	}

	// Clamp duration
	if duration <= 0 {
		duration = DefaultEventDuration
	}
	if duration > MaxEventDuration {
		duration = MaxEventDuration
	}

	// Join the topic
	topic, err := m.ps.Join(topicName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	entry := &ephemeralTopicEntry{
		TopicID:   topicName,
		EventID:   eventID,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
		topic:     topic,
	}
	m.topics[topicName] = entry

	return topic, nil
}

// SubscribeToEventTopic subscribes to an event topic with a message handler.
func (m *EphemeralTopicManager) SubscribeToEventTopic(ctx context.Context, eventID string, handler MessageHandler) error {
	topicName := EventTopic(eventID)

	sub, err := subscribeToEphemeralTopic(&m.mu, m.topics, topicName)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Already subscribed
	}

	// Start handler goroutine
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				return
			}
			handler(ctx, msg)
		}
	}()

	return nil
}

// LeaveEventTopic leaves and cleans up an event topic.
func (m *EphemeralTopicManager) LeaveEventTopic(eventID string) error {
	return m.leaveTopic(EventTopic(eventID), nil)
}

// leaveTopic handles common topic leave logic with optional pre-delete callback.
func (m *EphemeralTopicManager) leaveTopic(topicName string, preDeleteFn func(*ephemeralTopicEntry)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.topics[topicName]
	if !ok {
		return nil // Already gone
	}

	cleanupTopicEntry(entry)
	if preDeleteFn != nil {
		preDeleteFn(entry)
	}
	delete(m.topics, topicName)
	return nil
}

// ActiveTopics returns a list of active event topic IDs.
func (m *EphemeralTopicManager) ActiveTopics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	topics := make([]string, 0, len(m.topics))
	for topicID := range m.topics {
		topics = append(topics, topicID)
	}
	return topics
}

// GetEventInfo returns information about an event topic.
func (m *EphemeralTopicManager) GetEventInfo(eventID string) (createdAt, expiresAt time.Time, exists bool) {
	topicName := EventTopic(eventID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.topics[topicName]
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return entry.CreatedAt, entry.ExpiresAt, true
}

// CleanupExpired removes expired event topics.
func (m *EphemeralTopicManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for topicID, entry := range m.topics {
		if now.After(entry.ExpiresAt) {
			if entry.sub != nil {
				entry.sub.Cancel()
			}
			if entry.topic != nil {
				_ = entry.topic.Close()
			}
			delete(m.topics, topicID)
			cleaned++
		}
	}

	return cleaned
}

// StartCleanupLoop starts a goroutine to periodically clean up expired topics.
func (m *EphemeralTopicManager) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.CleanupExpired()
			}
		}
	}()
}

// Stop stops the cleanup loop.
func (m *EphemeralTopicManager) Stop() {
	close(m.stopCh)
}

// CouncilTopicManager manages encrypted council topics.
type CouncilTopicManager struct {
	ps     *PubSub
	topics map[string]*councilTopicEntry
	mu     sync.RWMutex
	stopCh chan struct{}
}

// councilTopicEntry tracks a council topic's state.
type councilTopicEntry struct {
	TopicID    string
	CouncilID  string
	CreatedAt  time.Time
	topic      *pubsub.Topic
	sub        *pubsub.Subscription
	EncryptKey []byte // Council symmetric key
}

// NewCouncilTopicManager creates a new council topic manager.
func NewCouncilTopicManager(ps *PubSub) *CouncilTopicManager {
	return &CouncilTopicManager{
		ps:     ps,
		topics: make(map[string]*councilTopicEntry),
		stopCh: make(chan struct{}),
	}
}

// JoinCouncilTopic joins an encrypted council topic.
// The encryptKey is the council's symmetric encryption key.
func (m *CouncilTopicManager) JoinCouncilTopic(ctx context.Context, councilID string, encryptKey []byte) (*pubsub.Topic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	topicName := CouncilTopic(councilID)

	// Check if already joined
	if entry, ok := m.topics[topicName]; ok {
		return entry.topic, nil
	}

	// Join the topic
	topic, err := m.ps.Join(topicName)
	if err != nil {
		return nil, err
	}

	entry := &councilTopicEntry{
		TopicID:    topicName,
		CouncilID:  councilID,
		CreatedAt:  time.Now(),
		topic:      topic,
		EncryptKey: encryptKey,
	}
	m.topics[topicName] = entry

	return topic, nil
}

// SubscribeToCouncilTopic subscribes to a council topic.
// Messages are encrypted with the council key.
func (m *CouncilTopicManager) SubscribeToCouncilTopic(ctx context.Context, councilID string, handler MessageHandler) error {
	topicName := CouncilTopic(councilID)

	sub, encryptKey, err := subscribeToCouncilTopic(&m.mu, m.topics, topicName)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Already subscribed
	}

	// Start decrypting handler goroutine
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				return
			}
			// Decrypt message before passing to handler
			decrypted := decryptCouncilMessage(msg, encryptKey)
			if decrypted != nil {
				handler(ctx, decrypted)
			}
		}
	}()

	return nil
}

// LeaveCouncilTopic leaves a council topic.
func (m *CouncilTopicManager) LeaveCouncilTopic(councilID string) error {
	topicName := CouncilTopic(councilID)

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.topics[topicName]
	if !ok {
		return nil
	}

	cleanupTopicEntry(entry)

	// Zero out the encryption key
	for i := range entry.EncryptKey {
		entry.EncryptKey[i] = 0
	}

	delete(m.topics, topicName)
	return nil
}

// ActiveCouncils returns a list of active council IDs.
func (m *CouncilTopicManager) ActiveCouncils() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	councils := make([]string, 0, len(m.topics))
	for _, entry := range m.topics {
		councils = append(councils, entry.CouncilID)
	}
	return councils
}

// GetCouncilInfo returns information about a council topic.
func (m *CouncilTopicManager) GetCouncilInfo(councilID string) (createdAt time.Time, exists bool) {
	topicName := CouncilTopic(councilID)

	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.topics[topicName]
	if !ok {
		return time.Time{}, false
	}
	return entry.CreatedAt, true
}

// Stop closes all council topics.
func (m *CouncilTopicManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for topicID, entry := range m.topics {
		if entry.sub != nil {
			entry.sub.Cancel()
		}
		if entry.topic != nil {
			_ = entry.topic.Close()
		}
		// Zero out keys
		for i := range entry.EncryptKey {
			entry.EncryptKey[i] = 0
		}
		delete(m.topics, topicID)
	}
}

// decryptCouncilMessage decrypts a council message with the symmetric key.
// Per DESIGN_DOCUMENT.md, councils use XChaCha20-Poly1305.
func decryptCouncilMessage(msg *pubsub.Message, key []byte) *pubsub.Message {
	if len(key) == 0 || msg == nil || msg.Message == nil {
		return nil
	}

	msgData := msg.GetData()
	if len(msgData) < chacha20poly1305.NonceSizeX {
		return nil
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil
	}

	nonce := msgData[:chacha20poly1305.NonceSizeX]
	ciphertext := msgData[chacha20poly1305.NonceSizeX:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Decryption failed - wrong key or tampered message
		return nil
	}

	// Create new pb.Message with decrypted data, copying fields from original
	pbMsg := &pb.Message{
		Data: plaintext,
	}
	if msg.Message != nil {
		pbMsg.From = msg.Message.From
		pbMsg.Seqno = msg.Message.Seqno
		pbMsg.Topic = msg.Message.Topic
		pbMsg.Signature = msg.Message.Signature
		pbMsg.Key = msg.Message.Key
	}

	// Return a copy with decrypted data
	return &pubsub.Message{
		Message:       pbMsg,
		ID:            msg.ID,
		ReceivedFrom:  msg.ReceivedFrom,
		ValidatorData: msg.ValidatorData,
		Local:         msg.Local,
	}
}

// EncryptCouncilMessage encrypts a message for council publication.
// Per DESIGN_DOCUMENT.md, councils use XChaCha20-Poly1305 with a random nonce.
func EncryptCouncilMessage(data, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrInvalidPayload
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce
	ciphertext := aead.Seal(nil, nonce, data, nil)
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return result, nil
}
