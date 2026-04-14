// Package overlays - Shadow Play Pulse Map visualization (stub for noebiten builds).
// Per ROADMAP.md line 491: "Pulse Map visualization — dark shimmering dome
// with lightning effects".
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"sync"
	"time"
)

// ShadowPlayState represents the current state of a Shadow Play game.
type ShadowPlayState uint8

const (
	ShadowPlayWaiting   ShadowPlayState = iota // Waiting for players.
	ShadowPlayActive                           // Game in progress.
	ShadowPlayVoting                           // Voting phase.
	ShadowPlayEchoesWin                        // Echoes won.
	ShadowPlayShadesWin                        // Shades won.
	ShadowPlayExpired                          // Game expired.
)

// ShadowPlayerRole represents a player's role (only revealed to viewer).
type ShadowPlayerRole uint8

const (
	ShadowRoleUnknown ShadowPlayerRole = iota // Role not known to viewer.
	ShadowRoleEcho                            // Standard participant.
	ShadowRoleShade                           // Hidden disruptor.
)

// ShadowPlayer represents a participant in a Shadow Play game.
type ShadowPlayer struct {
	SpecterKey   [32]byte         // Specter identity.
	Role         ShadowPlayerRole // Role (may be unknown to viewer).
	IsEliminated bool             // True if eliminated.
	X, Y         float64          // Position on Pulse Map.
}

// ShadowPlayInfo contains information about a Shadow Play game.
type ShadowPlayInfo struct {
	GameID      [32]byte        // Unique game identifier.
	State       ShadowPlayState // Current game state.
	X, Y        float64         // Center position (initiator's node).
	StartTime   time.Time       // When the game started.
	EndTime     time.Time       // When the game ends.
	RoundNumber int             // Current voting round.
	Players     []ShadowPlayer  // Participants.
}

// Lightning represents an animated lightning bolt effect (stub).
type Lightning struct {
	startAngle float64
	endAngle   float64
	intensity  float64
	startTime  time.Time
	duration   time.Duration
}

// ShadowPlayOverlay renders Shadow Play games on the Pulse Map (stub).
type ShadowPlayOverlay struct {
	mu     sync.RWMutex
	games  map[string]*ShadowPlayInfo
	phases map[string]float64
}

// NewShadowPlayOverlay creates a new Shadow Play overlay renderer (stub).
func NewShadowPlayOverlay() *ShadowPlayOverlay {
	return &ShadowPlayOverlay{
		games:  make(map[string]*ShadowPlayInfo),
		phases: make(map[string]float64),
	}
}

// AddGame adds a Shadow Play game to the overlay.
func (so *ShadowPlayOverlay) AddGame(info *ShadowPlayInfo) {
	if info == nil {
		return
	}
	so.mu.Lock()
	defer so.mu.Unlock()

	key := string(info.GameID[:])
	so.games[key] = info
	so.phases[key] = 0
}

// RemoveGame removes a Shadow Play game from the overlay.
func (so *ShadowPlayOverlay) RemoveGame(gameID [32]byte) {
	so.mu.Lock()
	defer so.mu.Unlock()

	key := string(gameID[:])
	delete(so.games, key)
	delete(so.phases, key)
}

// UpdateGame updates game state.
func (so *ShadowPlayOverlay) UpdateGame(info *ShadowPlayInfo) {
	if info == nil {
		return
	}
	so.mu.Lock()
	defer so.mu.Unlock()

	key := string(info.GameID[:])
	so.games[key] = info
}

// Update advances animations (stub - no-op).
func (so *ShadowPlayOverlay) Update() {
	// No-op in stub version.
}

// GetGame retrieves game info by ID.
func (so *ShadowPlayOverlay) GetGame(gameID [32]byte) *ShadowPlayInfo {
	so.mu.RLock()
	defer so.mu.RUnlock()
	return so.games[string(gameID[:])]
}

// GameCount returns the number of active games.
func (so *ShadowPlayOverlay) GameCount() int {
	so.mu.RLock()
	defer so.mu.RUnlock()
	return len(so.games)
}

// Clear removes all games from the overlay.
func (so *ShadowPlayOverlay) Clear() {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.games = make(map[string]*ShadowPlayInfo)
	so.phases = make(map[string]float64)
}

// ComputeDomeRadius calculates dome radius based on player count.
func (so *ShadowPlayOverlay) ComputeDomeRadius(info *ShadowPlayInfo) float64 {
	if info == nil {
		return 80.0
	}
	base := 80.0
	perPlayer := 8.0
	return base + float64(len(info.Players))*perPlayer
}
