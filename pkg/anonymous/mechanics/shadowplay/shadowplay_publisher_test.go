// Package mechanics - Shadow Play publisher tests.
// Per ROADMAP.md line 489, tests for game state, votes, eliminations, outcomes.
package shadowplay

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// TestNewShadowPlayPublisher tests ShadowPlayPublisher creation.
func TestNewShadowPlayPublisher(t *testing.T) {
	pub := NewShadowPlayPublisher(nil, nil)
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if pub.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("expected topic %s, got %s", mechanics.TopicAnonymousMechanics, pub.topic)
	}
}

// TestShadowPlayPublisher_PublishGameCreated tests game creation publishing.
func TestShadowPlayPublisher_PublishGameCreated(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, err := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	ctx := context.Background()
	err = pub.PublishGameCreated(ctx, game)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.Published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if event.EventType != pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CREATED {
		t.Errorf("expected CREATED event type, got %v", event.EventType)
	}
	if event.Play == nil {
		t.Fatal("expected play in event")
	}
}

// TestShadowPlayPublisher_PublishGameCreated_NoPublisher tests without publisher.
func TestShadowPlayPublisher_PublishGameCreated_NoPublisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(nil, privKey)

	var initiatorKey [32]byte
	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	err := pub.PublishGameCreated(context.Background(), game)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

// TestShadowPlayPublisher_PublishGameCreated_NilGame tests with nil game.
func TestShadowPlayPublisher_PublishGameCreated_NilGame(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	err := pub.PublishGameCreated(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil game")
	}
}

// TestShadowPlayPublisher_PublishGameCreated_NoPrivateKey tests without private key.
func TestShadowPlayPublisher_PublishGameCreated_NoPrivateKey(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pub := NewShadowPlayPublisher(mockPub, nil)

	var initiatorKey [32]byte
	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	err := pub.PublishGameCreated(context.Background(), game)
	if err != mechanics.ErrMissingPrivateKey {
		t.Errorf("expected mechanics.ErrMissingPrivateKey, got %v", err)
	}
}

// TestShadowPlayPublisher_PublishCastJoin tests player join publishing.
func TestShadowPlayPublisher_PublishCastJoin(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var gameID [32]byte
	gameID[0] = 1

	var playerKey [32]byte
	copy(playerKey[:], privKey.Public().(ed25519.PublicKey))

	player := &Player{
		SpecterKey:   playerKey,
		Role:         RoleEcho,
		IsEliminated: false,
	}

	ctx := context.Background()
	err := pub.PublishCastJoin(ctx, gameID, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.Published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if event.EventType != pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CAST_JOIN {
		t.Errorf("expected CAST_JOIN event type, got %v", event.EventType)
	}
	if event.Actor == nil {
		t.Fatal("expected actor in event")
	}
}

// TestShadowPlayPublisher_PublishCastJoin_NilPlayer tests with nil player.
func TestShadowPlayPublisher_PublishCastJoin_NilPlayer(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var gameID [32]byte
	err := pub.PublishCastJoin(context.Background(), gameID, nil)
	if err == nil {
		t.Error("expected error for nil player")
	}
}

// TestShadowPlayPublisher_PublishGameStarted tests game start publishing.
func TestShadowPlayPublisher_PublishGameStarted(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	ctx := context.Background()
	err := pub.PublishGameStarted(ctx, game)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.Published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if event.EventType != pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_STARTED {
		t.Errorf("expected STARTED event type, got %v", event.EventType)
	}
}

// TestShadowPlayPublisher_PublishGameEnded tests game end publishing.
func TestShadowPlayPublisher_PublishGameEnded(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	ctx := context.Background()
	err := pub.PublishGameEnded(ctx, game)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.Published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if event.EventType != pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_ENDED {
		t.Errorf("expected ENDED event type, got %v", event.EventType)
	}
}

// TestShadowPlayPublisher_PublishGameCancelled tests game cancellation publishing.
func TestShadowPlayPublisher_PublishGameCancelled(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var gameID [32]byte
	gameID[0] = 1

	ctx := context.Background()
	err := pub.PublishGameCancelled(ctx, gameID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.Published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if event.EventType != pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CANCELLED {
		t.Errorf("expected CANCELLED event type, got %v", event.EventType)
	}
}

// TestNewShadowPlayReceiver tests ShadowPlayReceiver creation.
func TestNewShadowPlayReceiver(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)
	if receiver == nil {
		t.Fatal("expected non-nil receiver")
	}
	if receiver.shadowPlayStore != store {
		t.Error("store not set correctly")
	}
}

// TestShadowPlayReceiver_HandleGameCreated tests game creation handling.
func TestShadowPlayReceiver_HandleGameCreated(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create and publish a game.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	err := publisher.PublishGameCreated(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify game was stored.
	stored := store.GetGame(game.ID)
	if stored == nil {
		t.Fatal("expected game to be stored")
	}
}

// TestShadowPlayReceiver_HandleCastJoin tests player joining handling.
func TestShadowPlayReceiver_HandleCastJoin(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create and store a game first.
	_, gamePrivKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], gamePrivKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)
	store.AddGame(game)

	// Create player and publish join.
	_, playerPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, playerPrivKey)

	var playerKey [32]byte
	copy(playerKey[:], playerPrivKey.Public().(ed25519.PublicKey))

	player := &Player{
		SpecterKey:   playerKey,
		Role:         RoleEcho,
		IsEliminated: false,
	}

	err := publisher.PublishCastJoin(context.Background(), game.ID, player)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify player was added.
	stored := store.GetGame(game.ID)
	if stored == nil {
		t.Fatal("expected game to be stored")
	}
	if len(stored.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(stored.Players))
	}
}

// TestShadowPlayReceiver_HandleGameStarted tests game start handling.
func TestShadowPlayReceiver_HandleGameStarted(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create and store a game.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)
	store.AddGame(game)

	// Publish start event.
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	err := publisher.PublishGameStarted(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Game should still be in store.
	stored := store.GetGame(game.ID)
	if stored == nil {
		t.Fatal("expected game to be stored")
	}
}

// TestShadowPlayReceiver_HandleGameEnded tests game end handling.
func TestShadowPlayReceiver_HandleGameEnded(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create and store a game.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)
	store.AddGame(game)

	// Mark game as complete for the test.
	game.State = ShadowPlayEchoesWin

	// Publish end event.
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	err := publisher.PublishGameEnded(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify game state updated.
	stored := store.GetGame(game.ID)
	if stored == nil {
		t.Fatal("expected game to be stored")
	}
	if stored.State != ShadowPlayEchoesWin {
		t.Errorf("expected state ShadowPlayEchoesWin, got %v", stored.State)
	}
}

// TestShadowPlayReceiver_HandleGameCancelled tests game cancellation handling.
func TestShadowPlayReceiver_HandleGameCancelled(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create and store a game.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)
	store.AddGame(game)

	// Publish cancellation event.
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	err := publisher.PublishGameCancelled(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify game state updated.
	stored := store.GetGame(game.ID)
	if stored == nil {
		t.Fatal("expected game to be stored")
	}
	if stored.State != ShadowPlayExpired {
		t.Errorf("expected state ShadowPlayExpired, got %v", stored.State)
	}
}

// TestShadowPlayReceiver_HandleMessage_InvalidData tests with invalid data.
func TestShadowPlayReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	err := receiver.HandleMessage([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestShadowPlayReceiver_HandleMessage_NonShadowPlayEvent tests with non-shadow play event.
func TestShadowPlayReceiver_HandleMessage_NonShadowPlayEvent(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create a gossip message without a shadow play event.
	gossipMsg := &pb.GossipMessage{}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != nil {
		t.Errorf("expected nil for non-shadow play event, got %v", err)
	}
}

// TestShadowPlayReceiver_HandleMessage_MissingSignature tests with missing signature.
func TestShadowPlayReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	// Create an unsigned shadow play event.
	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CREATED,
		PlayId:    []byte("test"),
	}
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_ShadowPlayEvent{
			ShadowPlayEvent: event,
		},
	}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != mechanics.ErrMissingSignature {
		t.Errorf("expected mechanics.ErrMissingSignature, got %v", err)
	}
}

// TestShadowPlayEventSignatureRoundTrip tests signature verification.
func TestShadowPlayEventSignatureRoundTrip(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	err := publisher.PublishGameCreated(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Unmarshal and verify signature.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		t.Fatal("expected shadow play event")
	}
	if len(event.Signature) == 0 {
		t.Fatal("expected signature")
	}

	// Verify the signature through receiver.
	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)
	err = receiver.verifyEventSignature(event)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

// BenchmarkShadowPlayPublisher_PublishGameCreated benchmarks game creation publishing.
func BenchmarkShadowPlayPublisher_PublishGameCreated(b *testing.B) {
	mockPub := &mechanics.MockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPub.Published = nil
		pub.PublishGameCreated(ctx, game)
	}
}

// BenchmarkShadowPlayReceiver_HandleMessage benchmarks message handling.
func BenchmarkShadowPlayReceiver_HandleMessage(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	publisher := NewShadowPlayPublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	game, _ := NewShadowPlay(initiatorKey, ShadowPlayDuration30Min, ShadowPlayMinPlayers)

	publisher.PublishGameCreated(context.Background(), game)
	data := mockPub.Published[0].Data

	store := NewShadowPlayStore()
	receiver := NewShadowPlayReceiver(store)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		receiver.HandleMessage(data)
	}
}

// Suppress unused import warning.
var _ = time.Now
