// Package mechanics - Cipher Puzzles persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package mechanics

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentPuzzleStore wraps PuzzleStore with Bbolt persistence.
type PersistentPuzzleStore struct {
	*PuzzleStore
	db *store.DB
}

// NewPersistentPuzzleStore creates a puzzle store with Bbolt persistence.
func NewPersistentPuzzleStore(db *store.DB) (*PersistentPuzzleStore, error) {
	ps := &PersistentPuzzleStore{
		PuzzleStore: NewPuzzleStore(),
		db:          db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading puzzles from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all puzzles from Bbolt into memory.
func (ps *PersistentPuzzleStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketPuzzles, func(key, value []byte) error {
		var pbPuzzle pb.CipherPuzzle
		if err := proto.Unmarshal(value, &pbPuzzle); err != nil {
			return nil // Skip corrupt entries.
		}

		puzzle := protoToPuzzle(&pbPuzzle)
		if puzzle == nil {
			return nil
		}

		ps.PuzzleStore.mu.Lock()
		ps.PuzzleStore.puzzles[puzzle.ID] = puzzle
		if puzzle.State == PuzzleActive && !puzzle.IsExpired() {
			ps.PuzzleStore.active = append(ps.PuzzleStore.active, puzzle)
		} else {
			ps.PuzzleStore.history = append(ps.PuzzleStore.history, puzzle)
		}
		ps.PuzzleStore.mu.Unlock()

		return nil
	})
}

// AddPuzzle adds a new puzzle and persists it.
func (ps *PersistentPuzzleStore) AddPuzzle(p *Puzzle) error {
	ps.PuzzleStore.AddPuzzle(p)

	if ps.db != nil {
		if err := ps.persistPuzzle(p); err != nil {
			ps.PuzzleStore.mu.Lock()
			delete(ps.PuzzleStore.puzzles, p.ID)
			ps.PuzzleStore.mu.Unlock()
			return fmt.Errorf("persisting puzzle: %w", err)
		}
	}

	return nil
}

// persistPuzzle saves a puzzle to Bbolt.
func (ps *PersistentPuzzleStore) persistPuzzle(p *Puzzle) error {
	pbPuzzle := puzzleToProto(p)
	data, err := proto.Marshal(pbPuzzle)
	if err != nil {
		return fmt.Errorf("marshaling puzzle: %w", err)
	}
	return ps.db.Put(store.BucketPuzzles, p.ID[:], data)
}

// UpdateAndPersist updates puzzle state and persists changes.
func (ps *PersistentPuzzleStore) UpdateAndPersist(p *Puzzle) error {
	if ps.db != nil {
		return ps.persistPuzzle(p)
	}
	return nil
}

// puzzleToProto converts a Puzzle to its protobuf representation.
func puzzleToProto(p *Puzzle) *pb.CipherPuzzle {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state := pb.PuzzleState_PUZZLE_STATE_UNSPECIFIED
	switch p.State {
	case PuzzleActive:
		state = pb.PuzzleState_PUZZLE_STATE_ACTIVE
	case PuzzleSolved:
		state = pb.PuzzleState_PUZZLE_STATE_SOLVED
	case PuzzleExpired:
		state = pb.PuzzleState_PUZZLE_STATE_EXPIRED
	}

	pbPuzzle := &pb.CipherPuzzle{
		Id:            p.ID[:],
		CreatorPubkey: p.InitiatorKey[:],
		Difficulty:    uint32(p.Difficulty),
		CreatedAt:     p.CreatedAt.Unix(),
		ExpiresAt:     p.ExpiresAt.Unix(),
		State:         state,
		SolutionHash:  p.Seed[:], // Using seed as solution verification.
		Hint:          []byte{},
	}

	if p.WinnerKey != nil {
		pbPuzzle.WinnerPubkey = (*p.WinnerKey)[:]
	}
	if p.SolvedAt != nil {
		pbPuzzle.SolvedAt = p.SolvedAt.Unix()
	}

	return pbPuzzle
}

// protoToPuzzle converts a protobuf CipherPuzzle to a Puzzle.
func protoToPuzzle(pbPuzzle *pb.CipherPuzzle) *Puzzle {
	if len(pbPuzzle.Id) != 32 || len(pbPuzzle.CreatorPubkey) != 32 {
		return nil
	}

	state := PuzzlePending
	switch pbPuzzle.State {
	case pb.PuzzleState_PUZZLE_STATE_ACTIVE:
		state = PuzzleActive
	case pb.PuzzleState_PUZZLE_STATE_SOLVED:
		state = PuzzleSolved
	case pb.PuzzleState_PUZZLE_STATE_EXPIRED:
		state = PuzzleExpired
	}

	puzzle := &Puzzle{
		Type:       PuzzleFragment, // Default to Fragment type.
		Difficulty: uint8(pbPuzzle.Difficulty),
		CreatedAt:  time.Unix(pbPuzzle.CreatedAt, 0),
		ExpiresAt:  time.Unix(pbPuzzle.ExpiresAt, 0),
		State:      state,
		Solution:   pbPuzzle.EncryptedContent,
	}
	copy(puzzle.ID[:], pbPuzzle.Id)
	copy(puzzle.InitiatorKey[:], pbPuzzle.CreatorPubkey)
	copy(puzzle.Seed[:], pbPuzzle.SolutionHash)

	if len(pbPuzzle.WinnerPubkey) == 32 {
		var winnerKey [32]byte
		copy(winnerKey[:], pbPuzzle.WinnerPubkey)
		puzzle.WinnerKey = &winnerKey
	}
	if pbPuzzle.SolvedAt > 0 {
		solvedAt := time.Unix(pbPuzzle.SolvedAt, 0)
		puzzle.SolvedAt = &solvedAt
	}

	puzzle.Duration = puzzle.ExpiresAt.Sub(puzzle.CreatedAt)

	return puzzle
}
