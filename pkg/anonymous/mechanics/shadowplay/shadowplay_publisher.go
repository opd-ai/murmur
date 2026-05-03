// Package mechanics - Shadow Play network propagation.
// Per ROADMAP.md line 489, broadcasts game state, votes, eliminations, outcomes.
package shadowplay

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// ShadowPlayPublisher handles publishing Shadow Play events to the anonymous mechanics topic.
// All shadow play events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type ShadowPlayPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewShadowPlayPublisher creates a new shadow play publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewShadowPlayPublisher(pub Publisher, privateKey ed25519.PrivateKey) *ShadowPlayPublisher {
	return &ShadowPlayPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishGameCreated broadcasts a new shadow play game announcement.
// Per ANONYMOUS_GAME_MECHANICS.md §6, Shadow Play is a social deduction game.
func (s *ShadowPlayPublisher) PublishGameCreated(
	ctx context.Context,
	game *ShadowPlay,
) error {
	if s.publisher == nil {
		return ErrPublisherNotSet
	}
	if game == nil {
		return fmt.Errorf("game cannot be nil")
	}
	if s.privateKey == nil {
		return ErrMissingPrivateKey
	}

	pbPlay := shadowPlayToProto(game)
	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CREATED,
		Play:      pbPlay,
		PlayId:    game.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return s.signAndPublish(ctx, event)
}

// PublishCastJoin broadcasts a player joining the shadow play cast.
// Players must join during the casting phase before the game starts.
func (s *ShadowPlayPublisher) PublishCastJoin(
	ctx context.Context,
	gameID [32]byte,
	player *Player,
) error {
	if s.publisher == nil {
		return ErrPublisherNotSet
	}
	if player == nil {
		return fmt.Errorf("player cannot be nil")
	}
	if s.privateKey == nil {
		return ErrMissingPrivateKey
	}

	roleStr := "Echo"
	if player.Role == RoleShade {
		roleStr = "Shade"
	}

	pbActor := &pb.ShadowPlayActor{
		SpecterPubkey: player.SpecterKey[:],
		Role:          roleStr,
		JoinedAt:      time.Now().Unix(),
	}

	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CAST_JOIN,
		PlayId:    gameID[:],
		Actor:     pbActor,
		Timestamp: time.Now().Unix(),
	}

	return s.signAndPublish(ctx, event)
}

// PublishGameStarted broadcasts that a shadow play game has started.
// The game transitions from casting to rehearsing/performing.
func (s *ShadowPlayPublisher) PublishGameStarted(
	ctx context.Context,
	game *ShadowPlay,
) error {
	if s.publisher == nil {
		return ErrPublisherNotSet
	}
	if game == nil {
		return fmt.Errorf("game cannot be nil")
	}
	if s.privateKey == nil {
		return ErrMissingPrivateKey
	}

	pbPlay := shadowPlayToProto(game)
	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_STARTED,
		Play:      pbPlay,
		PlayId:    game.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return s.signAndPublish(ctx, event)
}

// PublishGameEnded broadcasts that a shadow play game has ended.
// Includes final state with winner information and Resonance bonuses.
func (s *ShadowPlayPublisher) PublishGameEnded(
	ctx context.Context,
	game *ShadowPlay,
) error {
	if s.publisher == nil {
		return ErrPublisherNotSet
	}
	if game == nil {
		return fmt.Errorf("game cannot be nil")
	}
	if s.privateKey == nil {
		return ErrMissingPrivateKey
	}

	pbPlay := shadowPlayToProto(game)
	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_ENDED,
		Play:      pbPlay,
		PlayId:    game.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return s.signAndPublish(ctx, event)
}

// PublishGameCancelled broadcasts that a shadow play game was cancelled.
// Games can be cancelled if not enough players join before deadline.
func (s *ShadowPlayPublisher) PublishGameCancelled(
	ctx context.Context,
	gameID [32]byte,
) error {
	if s.publisher == nil {
		return ErrPublisherNotSet
	}
	if s.privateKey == nil {
		return ErrMissingPrivateKey
	}

	event := &pb.ShadowPlayEvent{
		EventType: pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CANCELLED,
		PlayId:    gameID[:],
		Timestamp: time.Now().Unix(),
	}

	return s.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it.
func (s *ShadowPlayPublisher) signAndPublish(ctx context.Context, event *pb.ShadowPlayEvent) error {
	// Sign the event.
	sigData := s.eventSignatureData(event)
	event.Signature = ed25519.Sign(s.privateKey, sigData)

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_ShadowPlayEvent{
			ShadowPlayEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal shadow play event: %w", err)
	}

	return s.publisher.Publish(ctx, s.topic, data)
}

// eventSignatureData creates the data that will be signed.
func (s *ShadowPlayPublisher) eventSignatureData(event *pb.ShadowPlayEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("shadowplay-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.PlayId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	if event.Actor != nil {
		hash.Write(event.Actor.SpecterPubkey)
	}
	return hash.Sum(nil)
}

// ShadowPlayReceiver handles incoming Shadow Play events from the network.
type ShadowPlayReceiver struct {
	shadowPlayStore *ShadowPlayStore
}

// NewShadowPlayReceiver creates a new shadow play event receiver.
func NewShadowPlayReceiver(store *ShadowPlayStore) *ShadowPlayReceiver {
	return &ShadowPlayReceiver{
		shadowPlayStore: store,
	}
}

// HandleMessage processes an incoming shadow play event message.
func (r *ShadowPlayReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	event := gossipMsg.GetShadowPlayEvent()
	if event == nil {
		return nil // Not a shadow play event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(event); err != nil {
		return err
	}

	// Process the event.
	return r.processEvent(event)
}

// verifyEventSignature verifies the event signature.
func (r *ShadowPlayReceiver) verifyEventSignature(event *pb.ShadowPlayEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}

	// For created/started/ended events, verify using director's key from play.
	if event.Play != nil && len(event.Play.DirectorPubkey) == ed25519.PublicKeySize {
		sigData := r.eventSignatureData(event)
		if !ed25519.Verify(event.Play.DirectorPubkey, sigData, event.Signature) {
			return ErrSignatureFailed
		}
		return nil
	}

	// For cast_join events, verify using actor's key.
	if event.Actor != nil && len(event.Actor.SpecterPubkey) == ed25519.PublicKeySize {
		sigData := r.eventSignatureData(event)
		if !ed25519.Verify(event.Actor.SpecterPubkey, sigData, event.Signature) {
			return ErrSignatureFailed
		}
		return nil
	}

	// For cancelled events without play data, we need game from store.
	if event.EventType == pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CANCELLED {
		var gameID [32]byte
		copy(gameID[:], event.PlayId)
		game := r.shadowPlayStore.GetGame(gameID)
		if game != nil {
			sigData := r.eventSignatureData(event)
			if !ed25519.Verify(game.InitiatorKey[:], sigData, event.Signature) {
				return ErrSignatureFailed
			}
			return nil
		}
	}

	return ErrMissingSignature
}

// eventSignatureData creates the data that was signed.
func (r *ShadowPlayReceiver) eventSignatureData(event *pb.ShadowPlayEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("shadowplay-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.PlayId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	if event.Actor != nil {
		hash.Write(event.Actor.SpecterPubkey)
	}
	return hash.Sum(nil)
}

// processEvent handles the specific event type.
func (r *ShadowPlayReceiver) processEvent(event *pb.ShadowPlayEvent) error {
	switch event.EventType {
	case pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CREATED:
		return r.handleGameCreated(event)
	case pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CAST_JOIN:
		return r.handleCastJoin(event)
	case pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_STARTED:
		return r.handleGameStarted(event)
	case pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_ENDED:
		return r.handleGameEnded(event)
	case pb.ShadowPlayEventType_SHADOW_PLAY_EVENT_CANCELLED:
		return r.handleGameCancelled(event)
	default:
		return fmt.Errorf("unknown shadow play event type: %v", event.EventType)
	}
}

// handleGameCreated processes a game creation event.
func (r *ShadowPlayReceiver) handleGameCreated(event *pb.ShadowPlayEvent) error {
	if event.Play == nil {
		return fmt.Errorf("game created event missing play data")
	}

	game := protoToShadowPlay(event.Play)
	if game == nil {
		return fmt.Errorf("failed to convert shadow play from protobuf")
	}

	// Add to store if not already present.
	if existing := r.shadowPlayStore.GetGame(game.ID); existing == nil {
		r.shadowPlayStore.AddGame(game)
	}

	return nil
}

// handleCastJoin processes a player joining event.
func (r *ShadowPlayReceiver) handleCastJoin(event *pb.ShadowPlayEvent) error {
	if event.Actor == nil {
		return fmt.Errorf("cast join event missing actor data")
	}

	var gameID [32]byte
	copy(gameID[:], event.PlayId)

	game := r.shadowPlayStore.GetGame(gameID)
	if game == nil {
		return fmt.Errorf("game not found: %x", gameID)
	}

	// Convert actor to player and add to game.
	var specterKey [32]byte
	copy(specterKey[:], event.Actor.SpecterPubkey)

	// Check if already joined.
	game.mu.RLock()
	_, exists := game.playerByKey[string(specterKey[:])]
	game.mu.RUnlock()

	if !exists {
		// Join the game (error ignored if already joined or game full).
		game.Join(specterKey)
	}

	return nil
}

// handleGameStarted processes a game started event.
func (r *ShadowPlayReceiver) handleGameStarted(event *pb.ShadowPlayEvent) error {
	var gameID [32]byte
	copy(gameID[:], event.PlayId)

	game := r.shadowPlayStore.GetGame(gameID)
	if game == nil {
		// If game not found but play data provided, create it.
		if event.Play != nil {
			newGame := protoToShadowPlay(event.Play)
			if newGame != nil {
				r.shadowPlayStore.AddGame(newGame)
			}
		}
		return nil
	}

	// Update game state if play data provided.
	if event.Play != nil {
		updatedGame := protoToShadowPlay(event.Play)
		if updatedGame != nil {
			game.mu.Lock()
			game.State = updatedGame.State
			game.CurrentRound = updatedGame.CurrentRound
			game.RoundDeadline = updatedGame.RoundDeadline
			game.mu.Unlock()
		}
	}

	return nil
}

// handleGameEnded processes a game ended event.
func (r *ShadowPlayReceiver) handleGameEnded(event *pb.ShadowPlayEvent) error {
	var gameID [32]byte
	copy(gameID[:], event.PlayId)

	game := r.shadowPlayStore.GetGame(gameID)
	if game == nil {
		// If game not found but play data provided, create it in ended state.
		if event.Play != nil {
			newGame := protoToShadowPlay(event.Play)
			if newGame != nil {
				r.shadowPlayStore.AddGame(newGame)
			}
		}
		return nil
	}

	// Update game state.
	if event.Play != nil {
		updatedGame := protoToShadowPlay(event.Play)
		if updatedGame != nil {
			game.mu.Lock()
			game.State = updatedGame.State
			game.Players = updatedGame.Players
			game.playerByKey = updatedGame.playerByKey
			game.mu.Unlock()
		}
	} else {
		game.mu.Lock()
		game.State = ShadowPlayEchoesWin // Mark as complete.
		game.mu.Unlock()
	}

	return nil
}

// handleGameCancelled processes a game cancellation event.
func (r *ShadowPlayReceiver) handleGameCancelled(event *pb.ShadowPlayEvent) error {
	var gameID [32]byte
	copy(gameID[:], event.PlayId)

	game := r.shadowPlayStore.GetGame(gameID)
	if game == nil {
		return fmt.Errorf("game not found: %x", gameID)
	}

	game.mu.Lock()
	game.State = ShadowPlayExpired // Use expired for cancelled.
	game.mu.Unlock()

	return nil
}
