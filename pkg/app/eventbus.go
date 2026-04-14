// Package app - Central event dispatcher for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §2, the event bus uses channel fan-out for
// decoupled communication between subsystems. Events include network events
// (new peer, incoming Wave, Shroud circuit request), timer events, and user actions.
package app

import (
	"context"
	"sync"

	pb "github.com/opd-ai/murmur/proto"
)

// EventType identifies the type of event being dispatched.
type EventType uint8

const (
	// EventWaveReceived indicates a new Wave was received from the network.
	EventWaveReceived EventType = iota + 1

	// EventWaveCreated indicates the local user created a new Wave.
	EventWaveCreated

	// EventPeerConnected indicates a new peer connected to the network.
	EventPeerConnected

	// EventPeerDisconnected indicates a peer disconnected from the network.
	EventPeerDisconnected

	// EventIdentityUpdated indicates a peer's identity declaration was updated.
	EventIdentityUpdated

	// EventHeartbeatReceived indicates a heartbeat ping was received.
	EventHeartbeatReceived

	// EventShroudRelayDiscovered indicates a Shroud relay was discovered.
	EventShroudRelayDiscovered

	// EventShroudCircuitBuilt indicates a Shroud circuit was successfully built.
	EventShroudCircuitBuilt

	// EventShroudCircuitFailed indicates a Shroud circuit construction failed.
	EventShroudCircuitFailed

	// EventResonanceUpdated indicates a Specter's Resonance score changed.
	EventResonanceUpdated

	// EventMechanicStateChanged indicates a game mechanic state changed.
	EventMechanicStateChanged

	// EventTimerExpired indicates a scheduled timer fired.
	EventTimerExpired

	// EventUserAction indicates a user action from the UI.
	EventUserAction

	// EventReplyReceived indicates a reply to one of the user's Waves was received.
	EventReplyReceived
)

// String returns a human-readable name for the event type.
func (et EventType) String() string {
	switch et {
	case EventWaveReceived:
		return "WaveReceived"
	case EventWaveCreated:
		return "WaveCreated"
	case EventPeerConnected:
		return "PeerConnected"
	case EventPeerDisconnected:
		return "PeerDisconnected"
	case EventIdentityUpdated:
		return "IdentityUpdated"
	case EventHeartbeatReceived:
		return "HeartbeatReceived"
	case EventShroudRelayDiscovered:
		return "ShroudRelayDiscovered"
	case EventShroudCircuitBuilt:
		return "ShroudCircuitBuilt"
	case EventShroudCircuitFailed:
		return "ShroudCircuitFailed"
	case EventResonanceUpdated:
		return "ResonanceUpdated"
	case EventMechanicStateChanged:
		return "MechanicStateChanged"
	case EventTimerExpired:
		return "TimerExpired"
	case EventUserAction:
		return "UserAction"
	case EventReplyReceived:
		return "ReplyReceived"
	default:
		return "Unknown"
	}
}

// Event represents a typed event dispatched through the event bus.
// The Payload field contains type-specific data based on the EventType.
type Event struct {
	// Type identifies the event type.
	Type EventType

	// Payload contains event-specific data.
	// For EventWaveReceived: *WaveEvent
	// For EventPeerConnected/Disconnected: *PeerEvent
	// For EventIdentityUpdated: *IdentityEvent
	// For EventHeartbeatReceived: *HeartbeatEvent
	// For EventShroudRelayDiscovered: *ShroudEvent
	// For EventTimerExpired: *TimerEvent
	// For EventUserAction: *UserActionEvent
	Payload any
}

// WaveEvent contains details about a Wave event.
type WaveEvent struct {
	// Wave is the received or created Wave protobuf.
	Wave *pb.Wave

	// FromPeer is the peer ID the Wave was received from (empty if local).
	FromPeer string

	// IsLocal indicates if this Wave was created locally.
	IsLocal bool
}

// PeerEvent contains details about a peer connection event.
type PeerEvent struct {
	// PeerID is the libp2p peer ID.
	PeerID string

	// Multiaddrs are the peer's known addresses.
	Multiaddrs []string
}

// IdentityEvent contains details about an identity update.
type IdentityEvent struct {
	// PeerID is the peer whose identity was updated.
	PeerID string

	// PublicKey is the peer's Ed25519 public key (32 bytes).
	PublicKey []byte

	// DisplayName is the peer's display name (if declared).
	DisplayName string
}

// HeartbeatEvent contains details about a heartbeat ping.
type HeartbeatEvent struct {
	// PeerID is the peer that sent the heartbeat.
	PeerID string

	// Timestamp is the heartbeat's timestamp (Unix seconds).
	Timestamp int64
}

// ShroudEvent contains details about Shroud network events.
type ShroudEvent struct {
	// RelayPeerID is the peer ID of the relay (for discovery events).
	RelayPeerID string

	// CircuitID is the circuit identifier (for circuit events).
	CircuitID [32]byte

	// ErrorMessage is set for failed events.
	ErrorMessage string
}

// TimerEvent contains details about a timer expiration.
type TimerEvent struct {
	// TimerID identifies which timer expired.
	TimerID string

	// ScheduledFor is when the timer was scheduled to fire.
	ScheduledFor int64
}

// UserActionEvent contains details about a user action from the UI.
type UserActionEvent struct {
	// Action is the action type (e.g., "compose_wave", "select_node").
	Action string

	// TargetID is an optional target identifier.
	TargetID string

	// Data contains action-specific data.
	Data map[string]any
}

// ReplyEvent contains details about a reply to a user's Wave.
type ReplyEvent struct {
	// ParentWave is the Wave that was replied to.
	ParentWave *pb.Wave

	// ReplyWave is the reply Wave.
	ReplyWave *pb.Wave

	// FromPeer is the peer ID the reply was received from.
	FromPeer string

	// ThreadDepth is the depth of the reply in the thread.
	ThreadDepth int
}

// subscription represents a registered subscriber.
type subscription struct {
	types   map[EventType]bool
	channel chan<- Event
}

// EventBus dispatches events to registered subscribers using channel fan-out.
// Per TECHNICAL_IMPLEMENTATION.md §2, all subsystem communication flows
// through the event bus to maintain decoupled architecture.
type EventBus struct {
	mu sync.RWMutex

	// subscribers holds all registered subscriptions.
	subscribers []*subscription

	// inbound receives events to be dispatched.
	inbound chan Event

	// closed indicates if the event bus has been closed.
	closed bool
}

// EventBusConfig holds configuration for the event bus.
type EventBusConfig struct {
	// BufferSize is the size of the inbound event buffer.
	// Defaults to 256 if not specified.
	BufferSize int
}

// NewEventBus creates a new event bus with the given configuration.
func NewEventBus(cfg EventBusConfig) *EventBus {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}

	return &EventBus{
		subscribers: make([]*subscription, 0),
		inbound:     make(chan Event, cfg.BufferSize),
	}
}

// Start begins the event dispatch loop. It blocks until the context is canceled.
// Per TECHNICAL_IMPLEMENTATION.md §8, this runs as one of the ~8 persistent goroutines.
func (eb *EventBus) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			eb.mu.Lock()
			eb.closed = true
			eb.mu.Unlock()
			return
		case event := <-eb.inbound:
			eb.dispatch(event)
		}
	}
}

// dispatch fans out an event to all subscribers interested in that event type.
func (eb *EventBus) dispatch(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, sub := range eb.subscribers {
		// Check if subscriber is interested in this event type.
		if sub.types[event.Type] {
			// Non-blocking send to prevent slow subscribers from blocking.
			select {
			case sub.channel <- event:
			default:
				// Channel full, drop event for this subscriber.
				// TODO: Add metrics for dropped events.
			}
		}
	}
}

// Subscribe registers a subscriber to receive events of the specified types.
// The channel should be buffered; unbuffered channels may miss events.
// Returns an unsubscribe function that must be called to clean up.
func (eb *EventBus) Subscribe(types []EventType, ch chan<- Event) func() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return func() {} // No-op if closed.
	}

	typeMap := make(map[EventType]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	sub := &subscription{
		types:   typeMap,
		channel: ch,
	}
	eb.subscribers = append(eb.subscribers, sub)

	// Return unsubscribe function.
	return func() {
		eb.unsubscribe(sub)
	}
}

// SubscribeAll registers a subscriber to receive all event types.
func (eb *EventBus) SubscribeAll(ch chan<- Event) func() {
	return eb.Subscribe([]EventType{
		EventWaveReceived,
		EventWaveCreated,
		EventPeerConnected,
		EventPeerDisconnected,
		EventIdentityUpdated,
		EventHeartbeatReceived,
		EventShroudRelayDiscovered,
		EventShroudCircuitBuilt,
		EventShroudCircuitFailed,
		EventResonanceUpdated,
		EventMechanicStateChanged,
		EventTimerExpired,
		EventUserAction,
		EventReplyReceived,
	}, ch)
}

// unsubscribe removes a subscription from the subscribers list.
func (eb *EventBus) unsubscribe(sub *subscription) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for i, s := range eb.subscribers {
		if s == sub {
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			return
		}
	}
}

// Emit sends an event to the event bus for dispatch.
// Non-blocking: if the inbound buffer is full, the event is dropped.
func (eb *EventBus) Emit(event Event) {
	eb.mu.RLock()
	if eb.closed {
		eb.mu.RUnlock()
		return
	}
	eb.mu.RUnlock()

	select {
	case eb.inbound <- event:
	default:
		// Buffer full, drop event.
		// TODO: Add metrics for dropped events.
	}
}

// EmitWaveReceived emits a WaveReceived event.
func (eb *EventBus) EmitWaveReceived(wave *pb.Wave, fromPeer string) {
	eb.Emit(Event{
		Type: EventWaveReceived,
		Payload: &WaveEvent{
			Wave:     wave,
			FromPeer: fromPeer,
			IsLocal:  false,
		},
	})
}

// EmitWaveCreated emits a WaveCreated event for a locally created Wave.
func (eb *EventBus) EmitWaveCreated(wave *pb.Wave) {
	eb.Emit(Event{
		Type: EventWaveReceived,
		Payload: &WaveEvent{
			Wave:    wave,
			IsLocal: true,
		},
	})
}

// EmitPeerConnected emits a PeerConnected event.
func (eb *EventBus) EmitPeerConnected(peerID string, addrs []string) {
	eb.Emit(Event{
		Type: EventPeerConnected,
		Payload: &PeerEvent{
			PeerID:     peerID,
			Multiaddrs: addrs,
		},
	})
}

// EmitPeerDisconnected emits a PeerDisconnected event.
func (eb *EventBus) EmitPeerDisconnected(peerID string) {
	eb.Emit(Event{
		Type: EventPeerDisconnected,
		Payload: &PeerEvent{
			PeerID: peerID,
		},
	})
}

// EmitIdentityUpdated emits an IdentityUpdated event.
func (eb *EventBus) EmitIdentityUpdated(peerID string, pubKey []byte, displayName string) {
	eb.Emit(Event{
		Type: EventIdentityUpdated,
		Payload: &IdentityEvent{
			PeerID:      peerID,
			PublicKey:   pubKey,
			DisplayName: displayName,
		},
	})
}

// EmitHeartbeat emits a HeartbeatReceived event.
func (eb *EventBus) EmitHeartbeat(peerID string, timestamp int64) {
	eb.Emit(Event{
		Type: EventHeartbeatReceived,
		Payload: &HeartbeatEvent{
			PeerID:    peerID,
			Timestamp: timestamp,
		},
	})
}

// EmitShroudRelayDiscovered emits a ShroudRelayDiscovered event.
func (eb *EventBus) EmitShroudRelayDiscovered(relayPeerID string) {
	eb.Emit(Event{
		Type: EventShroudRelayDiscovered,
		Payload: &ShroudEvent{
			RelayPeerID: relayPeerID,
		},
	})
}

// EmitTimerExpired emits a TimerExpired event.
func (eb *EventBus) EmitTimerExpired(timerID string, scheduledFor int64) {
	eb.Emit(Event{
		Type: EventTimerExpired,
		Payload: &TimerEvent{
			TimerID:      timerID,
			ScheduledFor: scheduledFor,
		},
	})
}

// EmitUserAction emits a UserAction event.
func (eb *EventBus) EmitUserAction(action, targetID string, data map[string]any) {
	eb.Emit(Event{
		Type: EventUserAction,
		Payload: &UserActionEvent{
			Action:   action,
			TargetID: targetID,
			Data:     data,
		},
	})
}

// SubscriberCount returns the number of active subscribers.
// Useful for testing and monitoring.
func (eb *EventBus) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers)
}

// IsClosed returns true if the event bus has been closed.
func (eb *EventBus) IsClosed() bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.closed
}

// EmitReplyReceived emits a ReplyReceived event.
// This is called when a reply to one of the user's Waves is received.
func (eb *EventBus) EmitReplyReceived(parentWave, replyWave *pb.Wave, fromPeer string, depth int) {
	eb.Emit(Event{
		Type: EventReplyReceived,
		Payload: &ReplyEvent{
			ParentWave:  parentWave,
			ReplyWave:   replyWave,
			FromPeer:    fromPeer,
			ThreadDepth: depth,
		},
	})
}
