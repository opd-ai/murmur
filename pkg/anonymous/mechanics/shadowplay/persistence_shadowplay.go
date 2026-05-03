// Package mechanics - Shadow Play persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package shadowplay

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentShadowPlayStore wraps ShadowPlayStore with Bbolt persistence.
type PersistentShadowPlayStore struct {
	*ShadowPlayStore
	db *store.DB
}

// NewPersistentShadowPlayStore creates a shadow play store with Bbolt persistence.
func NewPersistentShadowPlayStore(db *store.DB) (*PersistentShadowPlayStore, error) {
	ps := &PersistentShadowPlayStore{
		ShadowPlayStore: NewShadowPlayStore(),
		db:              db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading shadow plays from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all shadow plays from Bbolt into memory.
func (ps *PersistentShadowPlayStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketShadowPlay, func(key, value []byte) error {
		var pbPlay pb.ShadowPlay
		if err := proto.Unmarshal(value, &pbPlay); err != nil {
			return nil // Skip corrupt entries.
		}

		game := protoToShadowPlay(&pbPlay)
		if game == nil {
			return nil
		}

		ps.ShadowPlayStore.mu.Lock()
		ps.ShadowPlayStore.games[hex.EncodeToString(game.ID[:])] = game
		ps.ShadowPlayStore.mu.Unlock()

		return nil
	})
}

// AddGame adds a new game and persists it.
func (ps *PersistentShadowPlayStore) AddGame(game *ShadowPlay) error {
	ps.ShadowPlayStore.AddGame(game)

	if ps.db != nil {
		if err := ps.persistGame(game); err != nil {
			ps.ShadowPlayStore.mu.Lock()
			delete(ps.ShadowPlayStore.games, hex.EncodeToString(game.ID[:]))
			ps.ShadowPlayStore.mu.Unlock()
			return fmt.Errorf("persisting shadow play: %w", err)
		}
	}

	return nil
}

// persistGame saves a shadow play to Bbolt.
func (ps *PersistentShadowPlayStore) persistGame(game *ShadowPlay) error {
	pbPlay := shadowPlayToProto(game)
	data, err := proto.Marshal(pbPlay)
	if err != nil {
		return fmt.Errorf("marshaling shadow play: %w", err)
	}
	return ps.db.Put(store.BucketShadowPlay, game.ID[:], data)
}

// UpdateAndPersist updates game state and persists changes.
func (ps *PersistentShadowPlayStore) UpdateAndPersist(game *ShadowPlay) error {
	if ps.db != nil {
		return ps.persistGame(game)
	}
	return nil
}

// PruneCompleted removes completed games from memory and database.
func (ps *PersistentShadowPlayStore) PruneCompleted(retention time.Duration) int {
	ps.ShadowPlayStore.mu.RLock()
	var toRemove [][32]byte
	cutoff := time.Now().Add(-retention)
	for _, game := range ps.ShadowPlayStore.games {
		if game.IsGameOver() && game.CreatedAt.Before(cutoff) {
			toRemove = append(toRemove, game.ID)
		}
	}
	ps.ShadowPlayStore.mu.RUnlock()

	pruned := ps.ShadowPlayStore.PruneCompleted(retention)

	if ps.db != nil {
		for _, id := range toRemove {
			_ = ps.db.Delete(store.BucketShadowPlay, id[:])
		}
	}

	return pruned
}

// shadowPlayToProto converts a ShadowPlay to its protobuf representation.
func shadowPlayToProto(game *ShadowPlay) *pb.ShadowPlay {
	game.mu.RLock()
	defer game.mu.RUnlock()

	state := pb.ShadowPlayState_SHADOW_PLAY_STATE_UNSPECIFIED
	switch game.State {
	case ShadowPlayWaiting:
		state = pb.ShadowPlayState_SHADOW_PLAY_STATE_CASTING
	case ShadowPlayActive:
		state = pb.ShadowPlayState_SHADOW_PLAY_STATE_PERFORMING
	case ShadowPlayVoting:
		state = pb.ShadowPlayState_SHADOW_PLAY_STATE_PERFORMING
	case ShadowPlayEchoesWin, ShadowPlayShadesWin:
		state = pb.ShadowPlayState_SHADOW_PLAY_STATE_COMPLETE
	}

	pbPlay := &pb.ShadowPlay{
		Id:              game.ID[:],
		DirectorPubkey:  game.InitiatorKey[:],
		Title:           fmt.Sprintf("Shadow Play (%d players)", game.MaxPlayers),
		ScheduledTime:   game.CreatedAt.Unix(),
		DurationSeconds: int64(game.Duration.Seconds()),
		State:           state,
		AudienceCount:   uint32(len(game.Players)),
	}

	// Convert players to actors.
	for i, player := range game.Players {
		roleStr := "Echo"
		if player.Role == RoleShade {
			roleStr = "Shade"
		}
		pbActor := &pb.ShadowPlayActor{
			SpecterPubkey: player.SpecterKey[:],
			Role:          roleStr,
			JoinedAt:      game.CreatedAt.Add(time.Duration(i) * time.Millisecond).Unix(),
		}
		pbPlay.Actors = append(pbPlay.Actors, pbActor)
	}

	return pbPlay
}

// protoToShadowPlay converts a protobuf ShadowPlay to a ShadowPlay.
func protoToShadowPlay(pbPlay *pb.ShadowPlay) *ShadowPlay {
	if len(pbPlay.Id) != 32 || len(pbPlay.DirectorPubkey) != 32 {
		return nil
	}

	state := ShadowPlayWaiting
	switch pbPlay.State {
	case pb.ShadowPlayState_SHADOW_PLAY_STATE_CASTING:
		state = ShadowPlayWaiting
	case pb.ShadowPlayState_SHADOW_PLAY_STATE_REHEARSING:
		state = ShadowPlayWaiting
	case pb.ShadowPlayState_SHADOW_PLAY_STATE_PERFORMING:
		state = ShadowPlayActive
	case pb.ShadowPlayState_SHADOW_PLAY_STATE_COMPLETE:
		state = ShadowPlayEchoesWin // Default to echoes win for completed.
	case pb.ShadowPlayState_SHADOW_PLAY_STATE_CANCELLED:
		state = ShadowPlayWaiting // No cancelled state, default to waiting.
	}

	duration := time.Duration(pbPlay.DurationSeconds) * time.Second
	createdAt := time.Unix(pbPlay.ScheduledTime, 0)

	game := &ShadowPlay{
		CreatedAt:    createdAt,
		Duration:     duration,
		MaxPlayers:   int(pbPlay.AudienceCount),
		State:        state,
		GameDeadline: createdAt.Add(duration),
		playerByKey:  make(map[string]*Player),
	}
	copy(game.ID[:], pbPlay.Id)
	copy(game.InitiatorKey[:], pbPlay.DirectorPubkey)

	// Convert actors to players.
	for i, pbActor := range pbPlay.Actors {
		if len(pbActor.SpecterPubkey) != 32 {
			continue
		}
		player := &Player{
			Role:            roleFromString(pbActor.Role),
			JoinIndex:       i,
			IsEliminated:    false,
			EliminatedRound: -1,
		}
		copy(player.SpecterKey[:], pbActor.SpecterPubkey)
		game.Players = append(game.Players, player)
		game.playerByKey[hex.EncodeToString(player.SpecterKey[:])] = player
	}

	if game.MaxPlayers == 0 {
		game.MaxPlayers = len(game.Players)
	}

	return game
}

// roleFromString converts a role string back to PlayerRole.
func roleFromString(s string) PlayerRole {
	switch s {
	case "Shade":
		return RoleShade
	default:
		return RoleEcho
	}
}
