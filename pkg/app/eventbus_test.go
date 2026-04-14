// Package app - Event bus tests.
package app

import (
	"context"
	"sync"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

// TestEventBusBasic tests basic event emission and reception.
func TestEventBusBasic(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 16})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start event bus in background.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	// Create subscriber channel.
	ch := make(chan Event, 10)
	unsub := eb.Subscribe([]EventType{EventWaveReceived}, ch)
	defer unsub()

	// Emit event.
	testWave := &pb.Wave{}
	eb.EmitWaveReceived(testWave, "test-peer")

	// Wait for event.
	select {
	case event := <-ch:
		if event.Type != EventWaveReceived {
			t.Errorf("expected EventWaveReceived, got %s", event.Type)
		}
		payload, ok := event.Payload.(*WaveEvent)
		if !ok {
			t.Fatal("expected WaveEvent payload")
		}
		if payload.Wave != testWave {
			t.Error("wave mismatch")
		}
		if payload.FromPeer != "test-peer" {
			t.Errorf("expected peer 'test-peer', got '%s'", payload.FromPeer)
		}
		if payload.IsLocal {
			t.Error("expected IsLocal=false")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	cancel()
	wg.Wait()
}

// TestEventBusFiltering tests that subscribers only receive their subscribed event types.
func TestEventBusFiltering(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 16})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	// Subscribe only to peer events.
	ch := make(chan Event, 10)
	unsub := eb.Subscribe([]EventType{EventPeerConnected, EventPeerDisconnected}, ch)
	defer unsub()

	// Emit different event types.
	eb.EmitWaveReceived(&pb.Wave{}, "peer1")
	eb.EmitPeerConnected("peer2", []string{"/ip4/127.0.0.1/tcp/1234"})
	eb.EmitHeartbeat("peer3", time.Now().Unix())

	// Should only receive peer connected event.
	select {
	case event := <-ch:
		if event.Type != EventPeerConnected {
			t.Errorf("expected EventPeerConnected, got %s", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Should not receive any more events.
	select {
	case event := <-ch:
		t.Errorf("unexpected event: %s", event.Type)
	case <-time.After(100 * time.Millisecond):
		// Expected - no more events.
	}

	cancel()
	wg.Wait()
}

// TestEventBusMultipleSubscribers tests fan-out to multiple subscribers.
func TestEventBusMultipleSubscribers(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 16})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	// Create multiple subscribers.
	ch1 := make(chan Event, 10)
	ch2 := make(chan Event, 10)
	ch3 := make(chan Event, 10)

	unsub1 := eb.Subscribe([]EventType{EventPeerConnected}, ch1)
	unsub2 := eb.Subscribe([]EventType{EventPeerConnected}, ch2)
	unsub3 := eb.SubscribeAll(ch3)
	defer unsub1()
	defer unsub2()
	defer unsub3()

	if eb.SubscriberCount() != 3 {
		t.Errorf("expected 3 subscribers, got %d", eb.SubscriberCount())
	}

	// Emit event.
	eb.EmitPeerConnected("test-peer", nil)

	// All subscribers should receive the event.
	for i, ch := range []chan Event{ch1, ch2, ch3} {
		select {
		case event := <-ch:
			if event.Type != EventPeerConnected {
				t.Errorf("subscriber %d: expected EventPeerConnected, got %s", i, event.Type)
			}
		case <-time.After(time.Second):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}

	cancel()
	wg.Wait()
}

// TestEventBusUnsubscribe tests that unsubscribe works correctly.
func TestEventBusUnsubscribe(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 16})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	ch := make(chan Event, 10)
	unsub := eb.Subscribe([]EventType{EventPeerConnected}, ch)

	if eb.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", eb.SubscriberCount())
	}

	// Unsubscribe.
	unsub()

	if eb.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", eb.SubscriberCount())
	}

	// Emit event - should not be received.
	eb.EmitPeerConnected("test-peer", nil)

	select {
	case <-ch:
		t.Error("received event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event.
	}

	cancel()
	wg.Wait()
}

// TestEventBusAllEventTypes tests all event type convenience methods.
func TestEventBusAllEventTypes(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 32})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	ch := make(chan Event, 32)
	unsub := eb.SubscribeAll(ch)
	defer unsub()

	// Emit all event types.
	eb.EmitWaveReceived(&pb.Wave{}, "peer1")
	eb.EmitWaveCreated(&pb.Wave{})
	eb.EmitPeerConnected("peer2", []string{"/ip4/127.0.0.1/tcp/1234"})
	eb.EmitPeerDisconnected("peer3")
	eb.EmitIdentityUpdated("peer4", []byte("pubkey"), "Alice")
	eb.EmitHeartbeat("peer5", 1234567890)
	eb.EmitShroudRelayDiscovered("relay-peer")
	eb.EmitTimerExpired("timer1", 1234567890)
	eb.EmitUserAction("click", "node1", map[string]any{"x": 100, "y": 200})
	eb.EmitReplyReceived(&pb.Wave{WaveId: []byte("parent")}, &pb.Wave{WaveId: []byte("reply")}, "peer6", 1)

	// Collect all received events.
	received := make(map[EventType]bool)
	timeout := time.After(time.Second)

loop:
	for {
		select {
		case event := <-ch:
			received[event.Type] = true
		case <-timeout:
			break loop
		}
	}

	// Check we received at least these event types.
	expectedTypes := []EventType{
		EventWaveReceived, // WaveReceived emits this (WaveCreated also emits WaveReceived)
		EventPeerConnected,
		EventPeerDisconnected,
		EventIdentityUpdated,
		EventHeartbeatReceived,
		EventShroudRelayDiscovered,
		EventTimerExpired,
		EventUserAction,
		EventReplyReceived,
	}

	for _, et := range expectedTypes {
		if !received[et] {
			t.Errorf("did not receive event type: %s", et)
		}
	}

	cancel()
	wg.Wait()
}

// TestEventBusConcurrency tests concurrent event emission and subscription.
func TestEventBusConcurrency(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 256})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dispatchWg sync.WaitGroup
	dispatchWg.Add(1)
	go func() {
		defer dispatchWg.Done()
		eb.Start(ctx)
	}()

	const numEmitters = 10
	const numSubscribers = 5
	const eventsPerEmitter = 50

	// Create subscribers.
	var subWg sync.WaitGroup
	receivedCounts := make([]int, numSubscribers)
	var mu sync.Mutex

	for i := 0; i < numSubscribers; i++ {
		ch := make(chan Event, 256)
		idx := i
		eb.SubscribeAll(ch)

		subWg.Add(1)
		go func() {
			defer subWg.Done()
			for {
				select {
				case <-ch:
					mu.Lock()
					receivedCounts[idx]++
					mu.Unlock()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Emit events concurrently.
	var emitWg sync.WaitGroup
	for i := 0; i < numEmitters; i++ {
		emitWg.Add(1)
		go func(emitterID int) {
			defer emitWg.Done()
			for j := 0; j < eventsPerEmitter; j++ {
				eb.EmitPeerConnected("peer", nil)
			}
		}(i)
	}

	emitWg.Wait()

	// Give time for events to propagate.
	time.Sleep(100 * time.Millisecond)

	cancel()
	dispatchWg.Wait()
	subWg.Wait()

	// Verify subscribers received events.
	for i, count := range receivedCounts {
		if count == 0 {
			t.Errorf("subscriber %d received no events", i)
		}
		// With buffered channels and concurrent access, some events may be dropped.
		// Just verify each subscriber received at least some events.
		t.Logf("subscriber %d received %d events", i, count)
	}
}

// TestEventTypeString tests EventType.String() method.
func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventWaveReceived, "WaveReceived"},
		{EventWaveCreated, "WaveCreated"},
		{EventPeerConnected, "PeerConnected"},
		{EventPeerDisconnected, "PeerDisconnected"},
		{EventIdentityUpdated, "IdentityUpdated"},
		{EventHeartbeatReceived, "HeartbeatReceived"},
		{EventShroudRelayDiscovered, "ShroudRelayDiscovered"},
		{EventShroudCircuitBuilt, "ShroudCircuitBuilt"},
		{EventShroudCircuitFailed, "ShroudCircuitFailed"},
		{EventResonanceUpdated, "ResonanceUpdated"},
		{EventMechanicStateChanged, "MechanicStateChanged"},
		{EventTimerExpired, "TimerExpired"},
		{EventUserAction, "UserAction"},
		{EventType(255), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.et.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.et, got, tt.want)
		}
	}
}

// TestEventBusClose tests that closed event bus behaves correctly.
func TestEventBusClose(t *testing.T) {
	eb := NewEventBus(EventBusConfig{BufferSize: 16})
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		eb.Start(ctx)
	}()

	// Close the event bus.
	cancel()
	wg.Wait()

	if !eb.IsClosed() {
		t.Error("expected event bus to be closed")
	}

	// Subscribe after close should return no-op unsubscribe.
	ch := make(chan Event, 10)
	unsub := eb.Subscribe([]EventType{EventWaveReceived}, ch)
	unsub() // Should not panic.

	// Emit after close should not panic.
	eb.EmitPeerConnected("peer", nil)
}
