// Package resonance provides local reputation computation and rank thresholds.
// This file provides Bbolt persistence for Resonance scores.
// Per TECHNICAL_IMPLEMENTATION.md §6.4, all scores must survive application restarts.
package resonance

import (
	"fmt"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentScorer wraps Scorer with Bbolt persistence.
// It maintains backward compatibility with the in-memory scorer while
// persisting all state changes to disk.
type PersistentScorer struct {
	mu     sync.RWMutex
	scores map[string]*Score
	db     *store.DB
}

// NewPersistentScorer creates a Resonance scorer with Bbolt persistence.
// If db is nil, falls back to pure in-memory storage.
func NewPersistentScorer(db *store.DB) (*PersistentScorer, error) {
	ps := &PersistentScorer{
		scores: make(map[string]*Score),
		db:     db,
	}

	// Load existing scores from database.
	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading scores from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all Resonance scores from Bbolt into memory.
func (ps *PersistentScorer) loadFromDB() error {
	if ps.db == nil {
		return nil
	}

	return ps.db.ForEach(store.BucketResonance, func(key, value []byte) error {
		var pbScore pb.ResonanceScoreDetails
		if err := proto.Unmarshal(value, &pbScore); err != nil {
			// Skip corrupt entries.
			return nil
		}

		score := protoToScore(&pbScore)
		if score == nil {
			return nil
		}

		specterID := string(key)
		ps.scores[specterID] = score
		return nil
	})
}

// GetScore retrieves or creates a Score for a Specter.
func (ps *PersistentScorer) GetScore(specterID string) *Score {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if score, ok := ps.scores[specterID]; ok {
		return score
	}

	score := NewScore()
	ps.scores[specterID] = score
	ps.persistScore(specterID, score)
	return score
}

// LookupScore retrieves a Score without creating one.
func (ps *PersistentScorer) LookupScore(specterID string) (*Score, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	score, ok := ps.scores[specterID]
	return score, ok
}

// SetScore sets a Score for a Specter and persists it.
func (ps *PersistentScorer) SetScore(specterID string, score *Score) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.scores[specterID] = score
	ps.persistScore(specterID, score)
}

// RemoveScore removes a Specter's score from memory and database.
func (ps *PersistentScorer) RemoveScore(specterID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.scores, specterID)
	if ps.db != nil {
		ps.db.Delete(store.BucketResonance, []byte(specterID))
	}
}

// TopSpecters returns the top N Specters by Resonance score.
func (ps *PersistentScorer) TopSpecters(n int) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return topNSpectersByScore(ps.scores, n)
}

// Count returns the number of tracked Specters.
func (ps *PersistentScorer) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.scores)
}

// persistScore saves a score to the database.
func (ps *PersistentScorer) persistScore(specterID string, score *Score) {
	if ps.db == nil {
		return
	}

	pbScore := scoreToProto(score)
	data, err := proto.Marshal(pbScore)
	if err != nil {
		return
	}

	ps.db.Put(store.BucketResonance, []byte(specterID), data)
}

// Flush persists all in-memory scores to the database.
// Useful for periodic batch saves or shutdown.
func (ps *PersistentScorer) Flush() error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for specterID, score := range ps.scores {
		ps.persistScore(specterID, score)
	}
	return nil
}

// UpdateScore updates a Specter's score with a function and persists it.
// This is the preferred way to modify scores as it ensures persistence.
func (ps *PersistentScorer) UpdateScore(specterID string, fn func(*Score)) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	score, ok := ps.scores[specterID]
	if !ok {
		score = NewScore()
		ps.scores[specterID] = score
	}

	fn(score)
	ps.persistScore(specterID, score)
}

// scoreToProto converts a Score to its protobuf representation.
func scoreToProto(s *Score) *pb.ResonanceScoreDetails {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &pb.ResonanceScoreDetails{
		Publications:         int32(s.Publications),
		ConsecutiveDays:      int32(s.ConsecutiveDays),
		PuzzlesSolved:        int32(s.PuzzlesSolved),
		GamesWon:             int32(s.GamesWon),
		GamesLost:            int32(s.GamesLost),
		GiftsGiven:           int32(s.GiftsGiven),
		GiftsReceived:        int32(s.GiftsReceived),
		Endorsements:         int32(s.Endorsements),
		HighTierEndorsements: int32(s.HighTierEndorsements),
		LastActivityUnix:     s.LastActivity.Unix(),
		CreatedAtUnix:        s.CreatedAt.Unix(),
	}
}

// protoToScore converts a protobuf ResonanceScoreDetails to a Score.
func protoToScore(pb *pb.ResonanceScoreDetails) *Score {
	if pb == nil {
		return nil
	}

	return &Score{
		Publications:         int(pb.Publications),
		ConsecutiveDays:      int(pb.ConsecutiveDays),
		PuzzlesSolved:        int(pb.PuzzlesSolved),
		GamesWon:             int(pb.GamesWon),
		GamesLost:            int(pb.GamesLost),
		GiftsGiven:           int(pb.GiftsGiven),
		GiftsReceived:        int(pb.GiftsReceived),
		Endorsements:         int(pb.Endorsements),
		HighTierEndorsements: int(pb.HighTierEndorsements),
		LastActivity:         time.Unix(pb.LastActivityUnix, 0),
		CreatedAt:            time.Unix(pb.CreatedAtUnix, 0),
		weights:              DefaultWeights(),
	}
}
