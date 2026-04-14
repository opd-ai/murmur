package gossip

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
)

func TestEphemeralTopicManager_CreateEventTopic(t *testing.T) {
	// Create a test libp2p host and PubSub
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewEphemeralTopicManager(ps, nil)

	// Create an event topic
	eventID := "test-event-1"
	topic, err := manager.CreateEventTopic(ctx, eventID, time.Hour)
	if err != nil {
		t.Fatalf("failed to create event topic: %v", err)
	}
	if topic == nil {
		t.Fatal("expected non-nil topic")
	}

	// Verify topic is tracked
	topics := manager.ActiveTopics()
	if len(topics) != 1 {
		t.Errorf("expected 1 active topic, got %d", len(topics))
	}

	// Verify event info
	createdAt, expiresAt, exists := manager.GetEventInfo(eventID)
	if !exists {
		t.Error("event should exist")
	}
	if createdAt.IsZero() {
		t.Error("createdAt should not be zero")
	}
	if expiresAt.Before(createdAt.Add(time.Hour - time.Second)) {
		t.Error("expiresAt should be ~1 hour after createdAt")
	}
}

func TestEphemeralTopicManager_CreateDuplicate(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewEphemeralTopicManager(ps, nil)

	eventID := "test-event-2"
	topic1, _ := manager.CreateEventTopic(ctx, eventID, time.Hour)
	topic2, _ := manager.CreateEventTopic(ctx, eventID, time.Hour)

	// Should return the same topic
	if topic1 != topic2 {
		t.Error("duplicate creation should return same topic")
	}
}

func TestEphemeralTopicManager_DurationClamping(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewEphemeralTopicManager(ps, nil)

	// Test zero duration defaults to DefaultEventDuration
	eventID1 := "test-event-3a"
	manager.CreateEventTopic(ctx, eventID1, 0)
	_, expiresAt1, _ := manager.GetEventInfo(eventID1)
	expectedExpiry := time.Now().Add(DefaultEventDuration)
	if expiresAt1.Before(expectedExpiry.Add(-time.Minute)) {
		t.Error("zero duration should default to DefaultEventDuration")
	}

	// Test excessive duration clamped to MaxEventDuration
	eventID2 := "test-event-3b"
	manager.CreateEventTopic(ctx, eventID2, 1000*time.Hour)
	createdAt2, expiresAt2, _ := manager.GetEventInfo(eventID2)
	if expiresAt2.After(createdAt2.Add(MaxEventDuration + time.Minute)) {
		t.Error("excessive duration should be clamped to MaxEventDuration")
	}
}

func TestEphemeralTopicManager_LeaveEventTopic(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewEphemeralTopicManager(ps, nil)

	eventID := "test-event-4"
	manager.CreateEventTopic(ctx, eventID, time.Hour)

	if len(manager.ActiveTopics()) != 1 {
		t.Error("expected 1 active topic")
	}

	err = manager.LeaveEventTopic(eventID)
	if err != nil {
		t.Errorf("failed to leave topic: %v", err)
	}

	if len(manager.ActiveTopics()) != 0 {
		t.Error("expected 0 active topics after leave")
	}

	// Leaving again should be safe
	err = manager.LeaveEventTopic(eventID)
	if err != nil {
		t.Error("leaving non-existent topic should not error")
	}
}

func TestEphemeralTopicManager_CleanupExpired(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewEphemeralTopicManager(ps, nil)

	// Create a topic that's already expired
	eventID := "test-event-5"
	manager.CreateEventTopic(ctx, eventID, time.Hour)

	// Manually set expiry to the past
	manager.mu.Lock()
	topicName := EventTopic(eventID)
	manager.topics[topicName].ExpiresAt = time.Now().Add(-time.Hour)
	manager.mu.Unlock()

	cleaned := manager.CleanupExpired()
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	if len(manager.ActiveTopics()) != 0 {
		t.Error("expected 0 active topics after cleanup")
	}
}

func TestCouncilTopicManager_JoinCouncilTopic(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewCouncilTopicManager(ps)

	councilID := "test-council-1"
	key := []byte("test-encryption-key-32-bytes-aaa")

	topic, err := manager.JoinCouncilTopic(ctx, councilID, key)
	if err != nil {
		t.Fatalf("failed to join council topic: %v", err)
	}
	if topic == nil {
		t.Fatal("expected non-nil topic")
	}

	// Verify council is tracked
	councils := manager.ActiveCouncils()
	if len(councils) != 1 {
		t.Errorf("expected 1 active council, got %d", len(councils))
	}

	createdAt, exists := manager.GetCouncilInfo(councilID)
	if !exists {
		t.Error("council should exist")
	}
	if createdAt.IsZero() {
		t.Error("createdAt should not be zero")
	}
}

func TestCouncilTopicManager_LeaveCouncilTopic(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewCouncilTopicManager(ps)

	councilID := "test-council-2"
	key := []byte("test-encryption-key-32-bytes-bbb")

	manager.JoinCouncilTopic(ctx, councilID, key)

	err = manager.LeaveCouncilTopic(councilID)
	if err != nil {
		t.Errorf("failed to leave council: %v", err)
	}

	if len(manager.ActiveCouncils()) != 0 {
		t.Error("expected 0 active councils after leave")
	}
}

func TestCouncilTopicManager_Stop(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	ps, err := New(ctx, h)
	if err != nil {
		t.Fatalf("failed to create pubsub: %v", err)
	}
	defer ps.Close()

	manager := NewCouncilTopicManager(ps)

	// Join multiple councils
	for i := 0; i < 3; i++ {
		councilID := "test-council-3-" + string(rune('a'+i))
		key := make([]byte, 32)
		manager.JoinCouncilTopic(ctx, councilID, key)
	}

	if len(manager.ActiveCouncils()) != 3 {
		t.Error("expected 3 active councils")
	}

	manager.Stop()

	if len(manager.ActiveCouncils()) != 0 {
		t.Error("expected 0 active councils after stop")
	}
}

func TestEncryptCouncilMessage(t *testing.T) {
	data := []byte("test message")

	// Test with valid key
	key := []byte("valid-key")
	encrypted, err := EncryptCouncilMessage(data, key)
	if err != nil {
		t.Errorf("encryption failed: %v", err)
	}
	if encrypted == nil {
		t.Error("expected non-nil encrypted data")
	}

	// Test with empty key
	_, err = EncryptCouncilMessage(data, []byte{})
	if err == nil {
		t.Error("expected error with empty key")
	}
}

func TestEphemeralTopicConstants(t *testing.T) {
	if DefaultEventDuration != 24*time.Hour {
		t.Errorf("DefaultEventDuration = %v, expected 24h", DefaultEventDuration)
	}
	if MaxEventDuration != 72*time.Hour {
		t.Errorf("MaxEventDuration = %v, expected 72h", MaxEventDuration)
	}
	if EphemeralTopicPrefix != "/murmur/event/" {
		t.Errorf("EphemeralTopicPrefix = %s, expected /murmur/event/", EphemeralTopicPrefix)
	}
	if CouncilTopicPrefix != "/murmur/council/" {
		t.Errorf("CouncilTopicPrefix = %s, expected /murmur/council/", CouncilTopicPrefix)
	}
}

func TestEventTopicFormat(t *testing.T) {
	eventID := "my-event-123"
	topic := EventTopic(eventID)
	expected := "/murmur/event/my-event-123/1.0"
	if topic != expected {
		t.Errorf("EventTopic(%s) = %s, expected %s", eventID, topic, expected)
	}
}

func TestCouncilTopicFormat(t *testing.T) {
	councilID := "my-council-456"
	topic := CouncilTopic(councilID)
	expected := "/murmur/council/my-council-456/1.0"
	if topic != expected {
		t.Errorf("CouncilTopic(%s) = %s, expected %s", councilID, topic, expected)
	}
}
