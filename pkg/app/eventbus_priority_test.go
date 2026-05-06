// Package app provides tests for event bus behavior under load.
// Per AUDIT.md M1, these tests verify critical events are never dropped.
package app

import (
	"context"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

// TestEventBusCriticalEventsNeverDropped verifies that critical events
// (Reply, Circuit Built/Failed) are never dropped even under load.
func TestEventBusCriticalEventsNeverDropped(t *testing.T) {
	// Create an event bus with small buffer to force contention.
	eb := NewEventBus(EventBusConfig{BufferSize: 10})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the event bus.
	go eb.Start(ctx)

	// Create a subscriber for critical events.
	received := make(chan Event, 1000)
	eb.Subscribe([]EventType{
		EventReplyReceived,
		EventShroudCircuitBuilt,
		EventShroudCircuitFailed,
	}, received)

	// Fill the buffer with low-priority events first.
	for i := 0; i < 100; i++ {
		eb.Emit(Event{Type: EventHeartbeatReceived})
	}

	// Now emit critical events - these should never be dropped.
	criticalEvents := []EventType{
		EventReplyReceived,
		EventShroudCircuitBuilt,
		EventShroudCircuitFailed,
	}

	for _, eventType := range criticalEvents {
		eb.Emit(Event{
			Type: eventType,
			Payload: &WaveEvent{
				Wave: &pb.Wave{},
			},
		})
	}

	// Wait for all critical events to be received.
	timeout := time.After(5 * time.Second)
	receivedCount := 0
	for receivedCount < len(criticalEvents) {
		select {
		case e := <-received:
			// Verify it's one of the expected types.
			found := false
			for _, expected := range criticalEvents {
				if e.Type == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("received unexpected event type: %v", e.Type)
			}
			receivedCount++
		case <-timeout:
			t.Fatalf("timeout waiting for critical events; received %d of %d", receivedCount, len(criticalEvents))
		}
	}

	// Success - all critical events were delivered.
	t.Logf("All %d critical events delivered successfully under load", len(criticalEvents))
}

// TestEventBusBufferSizeIncrease verifies the buffer size was increased to 1024.
func TestEventBusBufferSizeIncrease(t *testing.T) {
	// Create event bus with default config.
	eb := NewEventBus(EventBusConfig{})

	// The buffer capacity should be 1024.
	if cap(eb.inbound) != 1024 {
		t.Errorf("EventBus buffer size = %d, want 1024", cap(eb.inbound))
	}
}

// TestEventBusHighLoad simulates high load to verify drop behavior.
func TestEventBusHighLoad(t *testing.T) {
	// Create event bus with small buffer.
	eb := NewEventBus(EventBusConfig{BufferSize: 50})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the event bus.
	go eb.Start(ctx)

	// Emit 10,000 low-priority events rapidly.
	for i := 0; i < 10000; i++ {
		eb.Emit(Event{Type: EventHeartbeatReceived})
	}

	// Low-priority events should be dropped (buffer will fill).
	// This test just verifies the system doesn't panic or deadlock.
	time.Sleep(100 * time.Millisecond)

	// Emit a critical event - should still be delivered.
	received := make(chan Event, 1)
	eb.Subscribe([]EventType{EventReplyReceived}, received)

	eb.Emit(Event{
		Type: EventReplyReceived,
		Payload: &WaveEvent{
			Wave: &pb.Wave{},
		},
	})

	select {
	case <-received:
		// Success - critical event delivered even under load.
	case <-time.After(5 * time.Second):
		t.Fatal("critical event not delivered within 5s under load")
	}
}
