// Package mechanics implements anonymous layer game mechanics for MURMUR.
// This file contains ShadowPlayStore for managing Shadow Play game instances.
package shadowplay

import (
	"encoding/hex"
	"sort"
	"sync"
	"time"

)

// ShadowPlayStore manages Shadow Play game instances.
type ShadowPlayStore struct {
	mu    sync.RWMutex
	games map[string]*ShadowPlay
}

// NewShadowPlayStore creates a new ShadowPlayStore.
func NewShadowPlayStore() *ShadowPlayStore {
	return &ShadowPlayStore{
		games: make(map[string]*ShadowPlay),
	}
}

// AddGame adds a game to the store.
func (s *ShadowPlayStore) AddGame(game *ShadowPlay) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.games[hex.EncodeToString(game.ID[:])] = game
}

// GetGame retrieves a game by ID.
func (s *ShadowPlayStore) GetGame(id [32]byte) *ShadowPlay {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.games[hex.EncodeToString(id[:])]
}

// RemoveGame removes a game from the store.
func (s *ShadowPlayStore) RemoveGame(id [32]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.games, hex.EncodeToString(id[:]))
}

// GetWaitingGames returns all games waiting for players.
func (s *ShadowPlayStore) GetWaitingGames() []*ShadowPlay {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var waiting []*ShadowPlay
	for _, game := range s.games {
		if game.IsWaiting() {
			waiting = append(waiting, game)
		}
	}

	// Sort by creation time.
	sort.Slice(waiting, func(i, j int) bool {
		return waiting[i].CreatedAt.Before(waiting[j].CreatedAt)
	})

	return waiting
}

// GetActiveGames returns all games in progress.
func (s *ShadowPlayStore) GetActiveGames() []*ShadowPlay {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*ShadowPlay
	for _, game := range s.games {
		if game.IsActive() {
			active = append(active, game)
		}
	}
	return active
}

// Count returns the number of games in the store.
func (s *ShadowPlayStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.games)
}

// UpdateAllStates updates state for all stored games.
func (s *ShadowPlayStore) UpdateAllStates() {
	s.mu.RLock()
	games := make([]*ShadowPlay, 0, len(s.games))
	for _, g := range s.games {
		games = append(games, g)
	}
	s.mu.RUnlock()

	for _, game := range games {
		game.UpdateState()
	}
}

// PruneCompleted removes completed games older than the retention period.
func (s *ShadowPlayStore) PruneCompleted(retention time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	pruned := 0

	for id, game := range s.games {
		if game.IsGameOver() && game.CreatedAt.Before(cutoff) {
			delete(s.games, id)
			pruned++
		}
	}

	return pruned
}
