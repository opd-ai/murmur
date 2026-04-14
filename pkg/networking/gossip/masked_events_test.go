package gossip

import (
	"context"
	"testing"
	"time"
)

func TestNewMaskedEventManager(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)
	if mgr == nil {
		t.Fatal("NewMaskedEventManager returned nil")
	}
	if mgr.GetActiveEventCount() != 0 {
		t.Error("New manager should have 0 active events")
	}
}

func TestMaskedEventManager_RegisterEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1, 2, 3}
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(30 * time.Minute)
	creatorKey := [32]byte{10, 20, 30}

	err := mgr.RegisterEvent(eventID, "Test Event", startTime, endTime, creatorKey)
	if err != nil {
		t.Fatalf("RegisterEvent failed: %v", err)
	}

	if mgr.GetActiveEventCount() != 1 {
		t.Errorf("GetActiveEventCount = %d, want 1", mgr.GetActiveEventCount())
	}
}

func TestMaskedEventManager_ActivateEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}
	mgr.RegisterEvent(eventID, "Test", time.Now(), time.Now().Add(1*time.Hour), [32]byte{10})

	if mgr.IsEventActive(eventID) {
		t.Error("Event should not be active initially")
	}

	err := mgr.ActivateEvent(eventID)
	if err != nil {
		t.Fatalf("ActivateEvent failed: %v", err)
	}

	if !mgr.IsEventActive(eventID) {
		t.Error("Event should be active after ActivateEvent")
	}
}

func TestMaskedEventManager_ActivateEvent_Unknown(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	err := mgr.ActivateEvent([32]byte{99})
	if err != ErrMaskedEventUnknown {
		t.Errorf("Expected ErrMaskedEventUnknown, got %v", err)
	}
}

func TestMaskedEventManager_CloseEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}
	mgr.RegisterEvent(eventID, "Test", time.Now(), time.Now().Add(1*time.Hour), [32]byte{10})
	mgr.ActivateEvent(eventID)

	err := mgr.CloseEvent(eventID)
	if err != nil {
		t.Fatalf("CloseEvent failed: %v", err)
	}

	if mgr.IsEventActive(eventID) {
		t.Error("Event should not be active after CloseEvent")
	}
}

func TestMaskedEventManager_RegisterMaskedKey(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}
	mgr.RegisterEvent(eventID, "Test", time.Now(), time.Now().Add(1*time.Hour), [32]byte{10})

	maskedKey := [32]byte{50, 60, 70}
	err := mgr.RegisterMaskedKey(eventID, maskedKey)
	if err != nil {
		t.Fatalf("RegisterMaskedKey failed: %v", err)
	}

	if !mgr.IsKeyRegistered(eventID, maskedKey) {
		t.Error("Key should be registered")
	}

	if mgr.IsKeyRegistered(eventID, [32]byte{99}) {
		t.Error("Unknown key should not be registered")
	}
}

func TestMaskedEventManager_RegisterMaskedKey_UnknownEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	err := mgr.RegisterMaskedKey([32]byte{99}, [32]byte{50})
	if err != ErrMaskedEventUnknown {
		t.Errorf("Expected ErrMaskedEventUnknown, got %v", err)
	}
}

func TestMaskedEventManager_GetRegisteredKeyCount(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}
	mgr.RegisterEvent(eventID, "Test", time.Now(), time.Now().Add(1*time.Hour), [32]byte{10})

	if mgr.GetRegisteredKeyCount(eventID) != 0 {
		t.Error("New event should have 0 registered keys")
	}

	mgr.RegisterMaskedKey(eventID, [32]byte{50})
	mgr.RegisterMaskedKey(eventID, [32]byte{60})

	if mgr.GetRegisteredKeyCount(eventID) != 2 {
		t.Errorf("GetRegisteredKeyCount = %d, want 2", mgr.GetRegisteredKeyCount(eventID))
	}
}

func TestMaskedEventManager_GetEventInfo(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}
	startTime := time.Now()
	endTime := startTime.Add(30 * time.Minute)
	mgr.RegisterEvent(eventID, "Test Topic", startTime, endTime, [32]byte{10})

	topic, start, end, isActive, exists := mgr.GetEventInfo(eventID)

	if !exists {
		t.Fatal("Event should exist")
	}
	if topic != "Test Topic" {
		t.Errorf("Topic = %q, want %q", topic, "Test Topic")
	}
	if start != startTime {
		t.Error("StartTime mismatch")
	}
	if end != endTime {
		t.Error("EndTime mismatch")
	}
	if isActive {
		t.Error("Event should not be active initially")
	}
}

func TestMaskedEventManager_GetEventInfo_NotFound(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	_, _, _, _, exists := mgr.GetEventInfo([32]byte{99})
	if exists {
		t.Error("Unknown event should not exist")
	}
}

func TestMaskedEventManager_Update(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	// Create an event that has already started.
	eventID := [32]byte{1}
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	mgr.RegisterEvent(eventID, "Test", startTime, endTime, [32]byte{10})

	// Event should not be active yet (manually registered).
	if mgr.IsEventActive(eventID) {
		t.Error("Event should not be active before Update")
	}

	mgr.Update()

	// After update, event should be activated since we're between start and end.
	if !mgr.IsEventActive(eventID) {
		t.Error("Event should be active after Update (past start time)")
	}
}

func TestMaskedEventManager_Update_EndEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	// Create an event that has already ended.
	eventID := [32]byte{1}
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	mgr.RegisterEvent(eventID, "Test", startTime, endTime, [32]byte{10})
	mgr.ActivateEvent(eventID)

	if !mgr.IsEventActive(eventID) {
		t.Fatal("Event should be active before Update")
	}

	mgr.Update()

	// After update, event should be deactivated since we're past end time.
	if mgr.IsEventActive(eventID) {
		t.Error("Event should not be active after Update (past end time)")
	}
}

func TestMaskedEventManager_CleanupExpiredEvents(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	// Create an expired event.
	expiredID := [32]byte{1}
	expiredStart := time.Now().Add(-2 * time.Hour)
	expiredEnd := time.Now().Add(-1 * time.Hour)
	mgr.RegisterEvent(expiredID, "Expired", expiredStart, expiredEnd, [32]byte{10})

	// Create a current event.
	currentID := [32]byte{2}
	currentStart := time.Now()
	currentEnd := time.Now().Add(1 * time.Hour)
	mgr.RegisterEvent(currentID, "Current", currentStart, currentEnd, [32]byte{20})

	if mgr.GetActiveEventCount() != 2 {
		t.Fatalf("Should have 2 events before cleanup")
	}

	cleaned := mgr.CleanupExpiredEvents()
	if cleaned != 1 {
		t.Errorf("CleanupExpiredEvents cleaned %d events, want 1", cleaned)
	}

	if mgr.GetActiveEventCount() != 1 {
		t.Errorf("Should have 1 event after cleanup, got %d", mgr.GetActiveEventCount())
	}

	// The current event should remain.
	_, _, _, _, exists := mgr.GetEventInfo(currentID)
	if !exists {
		t.Error("Current event should still exist")
	}

	// The expired event should be gone.
	_, _, _, _, exists = mgr.GetEventInfo(expiredID)
	if exists {
		t.Error("Expired event should not exist")
	}
}

func TestMaskedEventManager_SetHandler(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	// Should not panic with nil handler.
	mgr.SetHandler(nil)
}

func TestMaskedEventManager_SubscribeUnsubscribe_NoPubSub(t *testing.T) {
	// When pubsub is nil, subscribe/unsubscribe should be no-ops.
	mgr := NewMaskedEventManager(nil, nil)

	eventID := [32]byte{1}

	err := mgr.SubscribeToEventTopic(eventID)
	if err != nil {
		t.Errorf("SubscribeToEventTopic with nil pubsub should succeed: %v", err)
	}

	err = mgr.UnsubscribeFromEventTopic(eventID)
	if err != nil {
		t.Errorf("UnsubscribeFromEventTopic with nil pubsub should succeed: %v", err)
	}
}

func TestMaskedEventManager_PublishToEvent_Unknown(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	err := mgr.PublishToEvent(context.Background(), [32]byte{99}, []byte("test"))
	if err != ErrMaskedEventUnknown {
		t.Errorf("Expected ErrMaskedEventUnknown, got %v", err)
	}
}

func TestMaskedEventManager_IsKeyRegistered_UnknownEvent(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	if mgr.IsKeyRegistered([32]byte{99}, [32]byte{50}) {
		t.Error("Key should not be registered for unknown event")
	}
}

func TestMaskedEventManager_IsEventActive_Unknown(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	if mgr.IsEventActive([32]byte{99}) {
		t.Error("Unknown event should not be active")
	}
}

func TestMaskedEventManager_GetRegisteredKeyCount_Unknown(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	if mgr.GetRegisteredKeyCount([32]byte{99}) != 0 {
		t.Error("Unknown event should have 0 registered keys")
	}
}

func TestMaskedEventManager_BroadcastMethods(t *testing.T) {
	mgr := NewMaskedEventManager(nil, nil)

	// Test that broadcast methods don't panic.
	ctx := context.Background()

	err := mgr.BroadcastAnnouncement(ctx, &MaskedEventAnnouncement{})
	if err != nil {
		t.Errorf("BroadcastAnnouncement failed: %v", err)
	}

	err = mgr.BroadcastJoin(ctx, &MaskedEventJoin{})
	if err != nil {
		t.Errorf("BroadcastJoin failed: %v", err)
	}

	err = mgr.BroadcastSummary(ctx, &MaskedEventSummaryBroadcast{})
	if err != nil {
		t.Errorf("BroadcastSummary failed: %v", err)
	}
}
