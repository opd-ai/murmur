// Package mechanics implements anonymous layer game mechanics for MURMUR.
// Shadow Play: Social deduction game leveraging anonymity where Specters
// are assigned secret roles (Echoes vs Shades) and must identify hidden roles.
// Per ANONYMOUS_GAME_MECHANICS.md, Shadow Play requires Resonance 200 (Revenant milestone).
package mechanics

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"sync"
	"time"
)

// Shadow Play constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// ShadowPlayMinResonance to initiate a Shadow Play (Revenant milestone).
	ShadowPlayMinResonance = 200

	// ShadowPlayMinPlayers is minimum participant count.
	ShadowPlayMinPlayers = 5

	// ShadowPlayMaxPlayers is maximum participant count.
	ShadowPlayMaxPlayers = 13

	// ShadowPlayDuration30Min is 30 minute game duration.
	ShadowPlayDuration30Min = 30 * time.Minute

	// ShadowPlayDuration60Min is 60 minute game duration.
	ShadowPlayDuration60Min = 60 * time.Minute

	// ShadowPlayRoundDuration is the duration of each voting round.
	ShadowPlayRoundDuration = 5 * time.Minute

	// ShadowPlayBonusDecayDays is the decay period for bonuses.
	ShadowPlayBonusDecayDays = 14
)

// ShadowPlayState represents the state of a Shadow Play game.
type ShadowPlayState uint8

const (
	// ShadowPlayWaiting means waiting for players to join.
	ShadowPlayWaiting ShadowPlayState = iota

	// ShadowPlayActive means game is in progress.
	ShadowPlayActive

	// ShadowPlayVoting means voting phase is active.
	ShadowPlayVoting

	// ShadowPlayEchoesWin means Echoes eliminated all Shades.
	ShadowPlayEchoesWin

	// ShadowPlayShadesWin means Shades equal/outnumber Echoes.
	ShadowPlayShadesWin

	// ShadowPlayExpired means game expired without completion.
	ShadowPlayExpired
)

// PlayerRole represents a player's hidden role.
type PlayerRole uint8

const (
	// RoleEcho is the standard participant role.
	RoleEcho PlayerRole = iota

	// RoleShade is the hidden disruptor role.
	RoleShade
)

// ShadowPlayStateString returns a human-readable string for ShadowPlayState.
func ShadowPlayStateString(s ShadowPlayState) string {
	switch s {
	case ShadowPlayWaiting:
		return "Waiting"
	case ShadowPlayActive:
		return "Active"
	case ShadowPlayVoting:
		return "Voting"
	case ShadowPlayEchoesWin:
		return "Echoes Win"
	case ShadowPlayShadesWin:
		return "Shades Win"
	case ShadowPlayExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// PlayerRoleString returns a human-readable string for PlayerRole.
func PlayerRoleString(r PlayerRole) string {
	switch r {
	case RoleEcho:
		return "Echo"
	case RoleShade:
		return "Shade"
	default:
		return "Unknown"
	}
}

// Shadow Play errors.
var (
	ErrShadowPlayInsufficientResonance = errors.New(
		"insufficient resonance to create shadow play")
	ErrShadowPlayInvalidDuration = errors.New("invalid shadow play duration")
	ErrShadowPlayInvalidSize     = errors.New(
		"invalid shadow play size (5-13 players)")
	ErrShadowPlayFull         = errors.New("shadow play is full")
	ErrShadowPlayStarted      = errors.New("shadow play already started")
	ErrShadowPlayNotStarted   = errors.New("shadow play not yet started")
	ErrShadowPlayNotPlayer    = errors.New("not a player in this shadow play")
	ErrShadowPlayEliminated   = errors.New("player has been eliminated")
	ErrShadowPlayNotVoting    = errors.New("not in voting phase")
	ErrShadowPlayAlreadyVoted = errors.New(
		"player has already voted this round")
	ErrShadowPlayGameOver = errors.New("shadow play game is over")
)

// Player represents a participant in Shadow Play.
type Player struct {
	// SpecterKey is the player's Specter public key.
	SpecterKey [32]byte

	// Role is the player's assigned role (Echo or Shade).
	Role PlayerRole

	// JoinIndex is the player's position in join order (for role assignment).
	JoinIndex int

	// IsEliminated indicates if player has been eliminated.
	IsEliminated bool

	// EliminatedRound is the round when eliminated (-1 if not eliminated).
	EliminatedRound int

	// VotedFor tracks who this player voted for in current round.
	VotedFor *[32]byte
}

// ShadowPlay represents a social deduction game instance.
type ShadowPlay struct {
	mu sync.RWMutex

	// ID uniquely identifies this game.
	ID [32]byte

	// Seed is used for deterministic role assignment.
	Seed [32]byte

	// InitiatorKey is the Specter who created the game.
	InitiatorKey [32]byte

	// CreatedAt is when the game was created.
	CreatedAt time.Time

	// Duration is the total game duration.
	Duration time.Duration

	// MaxPlayers is the target player count (5-13).
	MaxPlayers int

	// State is the current game state.
	State ShadowPlayState

	// Players is the list of participants.
	Players []*Player

	// playerByKey maps Specter key to Player.
	playerByKey map[string]*Player

	// CurrentRound is the current voting round (1-indexed).
	CurrentRound int

	// RoundDeadline is when current round ends.
	RoundDeadline time.Time

	// GameDeadline is when entire game expires.
	GameDeadline time.Time
}

// NewShadowPlay creates a new Shadow Play game.
func NewShadowPlay(
	initiator [32]byte,
	duration time.Duration,
	maxPlayers int,
) (*ShadowPlay, error) {
	// Validate duration.
	if duration != ShadowPlayDuration30Min && duration != ShadowPlayDuration60Min {
		return nil, ErrShadowPlayInvalidDuration
	}

	// Validate player count.
	if maxPlayers < ShadowPlayMinPlayers || maxPlayers > ShadowPlayMaxPlayers {
		return nil, ErrShadowPlayInvalidSize
	}

	now := time.Now()

	// Generate random game ID and seed.
	var id, seed [32]byte
	rand.Read(id[:])
	rand.Read(seed[:])

	return &ShadowPlay{
		ID:           id,
		Seed:         seed,
		InitiatorKey: initiator,
		CreatedAt:    now,
		Duration:     duration,
		MaxPlayers:   maxPlayers,
		State:        ShadowPlayWaiting,
		Players:      make([]*Player, 0, maxPlayers),
		playerByKey:  make(map[string]*Player),
		CurrentRound: 0,
		GameDeadline: now.Add(duration),
	}, nil
}

// IsWaiting returns true if game is waiting for players.
func (s *ShadowPlay) IsWaiting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == ShadowPlayWaiting
}

// IsActive returns true if game is in progress.
func (s *ShadowPlay) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == ShadowPlayActive || s.State == ShadowPlayVoting
}

// IsGameOver returns true if game has ended.
func (s *ShadowPlay) IsGameOver() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == ShadowPlayEchoesWin ||
		s.State == ShadowPlayShadesWin ||
		s.State == ShadowPlayExpired
}

// Join adds a player to the game.
func (s *ShadowPlay) Join(specter [32]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ShadowPlayWaiting {
		return ErrShadowPlayStarted
	}

	if len(s.Players) >= s.MaxPlayers {
		return ErrShadowPlayFull
	}

	// Check for duplicate.
	specterHex := keyToHex(specter[:])
	if _, exists := s.playerByKey[specterHex]; exists {
		return nil // Already joined, idempotent.
	}

	player := &Player{
		SpecterKey:      specter,
		JoinIndex:       len(s.Players),
		IsEliminated:    false,
		EliminatedRound: -1,
	}

	s.Players = append(s.Players, player)
	s.playerByKey[specterHex] = player

	return nil
}

// Start begins the game when enough players have joined.
func (s *ShadowPlay) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ShadowPlayWaiting {
		return ErrShadowPlayStarted
	}

	if len(s.Players) < ShadowPlayMinPlayers {
		return ErrShadowPlayInvalidSize
	}

	// Assign roles deterministically from seed.
	s.assignRoles()

	// Start first round.
	s.CurrentRound = 1
	s.RoundDeadline = time.Now().Add(ShadowPlayRoundDuration)
	s.State = ShadowPlayActive

	return nil
}

// assignRoles assigns Echo and Shade roles based on seed.
// Number of Shades: 1 for 5-7 players, 2 for 8-13 players.
func (s *ShadowPlay) assignRoles() {
	numShades := 1
	if len(s.Players) >= 8 {
		numShades = 2
	}

	// Determine which player indices are Shades.
	shadeIndices := make(map[int]bool)
	for i := 0; i < numShades; i++ {
		// Deterministic: SHA-256(seed || "shade" || i) mod playerCount
		var hashInput []byte
		hashInput = append(hashInput, s.Seed[:]...)
		hashInput = append(hashInput, []byte("shade")...)
		hashInput = append(hashInput, byte(i))
		hash := sha256.Sum256(hashInput)

		// Convert first 8 bytes to index.
		index := binary.BigEndian.Uint64(hash[:8]) % uint64(len(s.Players))

		// Handle collision by incrementing.
		for shadeIndices[int(index)] {
			index = (index + 1) % uint64(len(s.Players))
		}
		shadeIndices[int(index)] = true
	}

	// Assign roles.
	for i, player := range s.Players {
		if shadeIndices[i] {
			player.Role = RoleShade
		} else {
			player.Role = RoleEcho
		}
	}
}

// DeriveRole computes a player's role from the seed (for local derivation).
// Per spec: role_assignment = SHA-256(play_seed || participant_index).
func (s *ShadowPlay) DeriveRole(specter [32]byte) (PlayerRole, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	player := s.playerByKey[keyToHex(specter[:])]
	if player == nil {
		return RoleEcho, ErrShadowPlayNotPlayer
	}

	return player.Role, nil
}

// GetShades returns all players with Shade role (for Shades to know each other).
func (s *ShadowPlay) GetShades() []*Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var shades []*Player
	for _, player := range s.Players {
		if player.Role == RoleShade {
			shades = append(shades, player)
		}
	}
	return shades
}

// StartVoting begins the voting phase for current round.
func (s *ShadowPlay) StartVoting() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ShadowPlayActive {
		return ErrShadowPlayNotStarted
	}

	// Reset votes.
	for _, player := range s.Players {
		player.VotedFor = nil
	}

	s.State = ShadowPlayVoting
	return nil
}

// Vote casts a vote to eliminate a player.
func (s *ShadowPlay) Vote(voter, target [32]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ShadowPlayVoting {
		return ErrShadowPlayNotVoting
	}

	voterPlayer, err := s.validateVoter(voter)
	if err != nil {
		return err
	}

	if err := s.validateVoteTarget(target); err != nil {
		return err
	}

	voterPlayer.VotedFor = &target
	return nil
}

// validateVoter checks that the voter is a valid, active player who hasn't voted.
func (s *ShadowPlay) validateVoter(voter [32]byte) (*Player, error) {
	voterPlayer := s.playerByKey[keyToHex(voter[:])]
	if voterPlayer == nil {
		return nil, ErrShadowPlayNotPlayer
	}
	if voterPlayer.IsEliminated {
		return nil, ErrShadowPlayEliminated
	}
	if voterPlayer.VotedFor != nil {
		return nil, ErrShadowPlayAlreadyVoted
	}
	return voterPlayer, nil
}

// validateVoteTarget checks that the target is a valid, active player.
func (s *ShadowPlay) validateVoteTarget(target [32]byte) error {
	targetPlayer := s.playerByKey[keyToHex(target[:])]
	if targetPlayer == nil {
		return ErrShadowPlayNotPlayer
	}
	if targetPlayer.IsEliminated {
		return ErrShadowPlayEliminated
	}
	return nil
}

// TallyVotes counts votes and eliminates player with most votes.
// Returns eliminated player and whether game is over.
func (s *ShadowPlay) TallyVotes() (*Player, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ShadowPlayVoting {
		return nil, false, ErrShadowPlayNotVoting
	}

	eliminated := s.eliminateMostVoted()
	gameOver := s.checkWinConditions()

	if !gameOver {
		s.advanceToNextRound()
	}

	return eliminated, gameOver, nil
}

// countPlayerVotes tallies votes from active players.
func (s *ShadowPlay) countPlayerVotes() map[string]int {
	counts := make(map[string]int)
	for _, player := range s.Players {
		if player.IsEliminated || player.VotedFor == nil {
			continue
		}
		counts[keyToHex(player.VotedFor[:])]++
	}
	return counts
}

// findMostVoted returns the hex key of the player with most votes.
func (s *ShadowPlay) findMostVoted(counts map[string]int) (string, int) {
	var maxVotes int
	var maxHex string
	for hex, count := range counts {
		if count > maxVotes {
			maxVotes = count
			maxHex = hex
		}
	}
	return maxHex, maxVotes
}

// eliminateMostVoted removes the player with the most votes.
func (s *ShadowPlay) eliminateMostVoted() *Player {
	counts := s.countPlayerVotes()
	maxHex, maxVotes := s.findMostVoted(counts)

	if maxVotes == 0 {
		return nil
	}

	eliminated := s.playerByKey[maxHex]
	if eliminated != nil {
		eliminated.IsEliminated = true
		eliminated.EliminatedRound = s.CurrentRound
	}
	return eliminated
}

// advanceToNextRound sets up the next game round.
func (s *ShadowPlay) advanceToNextRound() {
	s.CurrentRound++
	s.RoundDeadline = time.Now().Add(ShadowPlayRoundDuration)
	s.State = ShadowPlayActive
}

// checkWinConditions checks if game has ended.
// Echoes win: all Shades eliminated.
// Shades win: Shades equal or outnumber Echoes.
func (s *ShadowPlay) checkWinConditions() bool {
	var activeEchoes, activeShades int

	for _, player := range s.Players {
		if player.IsEliminated {
			continue
		}
		if player.Role == RoleEcho {
			activeEchoes++
		} else {
			activeShades++
		}
	}

	if activeShades == 0 {
		s.State = ShadowPlayEchoesWin
		return true
	}

	if activeShades >= activeEchoes {
		s.State = ShadowPlayShadesWin
		return true
	}

	return false
}

// UpdateState updates game state based on time.
func (s *ShadowPlay) UpdateState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check game expiration.
	if now.After(s.GameDeadline) && !s.IsGameOver() {
		s.State = ShadowPlayExpired
	}
}

// GetPlayer retrieves a player by Specter key.
func (s *ShadowPlay) GetPlayer(specter [32]byte) *Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.playerByKey[keyToHex(specter[:])]
}

// GetActivePlayers returns all non-eliminated players.
func (s *ShadowPlay) GetActivePlayers() []*Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*Player
	for _, player := range s.Players {
		if !player.IsEliminated {
			active = append(active, player)
		}
	}
	return active
}

// GetEliminatedPlayers returns all eliminated players.
func (s *ShadowPlay) GetEliminatedPlayers() []*Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var eliminated []*Player
	for _, player := range s.Players {
		if player.IsEliminated {
			eliminated = append(eliminated, player)
		}
	}
	return eliminated
}

// PlayerCount returns the total number of players.
func (s *ShadowPlay) PlayerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Players)
}

// ComputeShadowPlayWinBonus calculates Resonance bonus for winners.
// Per spec: play_bonus = 5 * ln(1 + participant_count).
func ComputeShadowPlayWinBonus(participantCount int) float64 {
	return 5.0 * math.Log(1.0+float64(participantCount))
}

// ComputeShadowPlayLoseBonus calculates Resonance bonus for losers.
// Per spec: consolation_bonus = 2 * ln(1 + participant_count).
func ComputeShadowPlayLoseBonus(participantCount int) float64 {
	return 2.0 * math.Log(1.0+float64(participantCount))
}

// GetWinners returns the list of winning players.
func (s *ShadowPlay) GetWinners() []*Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.State != ShadowPlayEchoesWin && s.State != ShadowPlayShadesWin {
		return nil
	}

	var winners []*Player
	var winningRole PlayerRole

	if s.State == ShadowPlayEchoesWin {
		winningRole = RoleEcho
	} else {
		winningRole = RoleShade
	}

	for _, player := range s.Players {
		if player.Role == winningRole {
			winners = append(winners, player)
		}
	}

	return winners
}

// GetLosers returns the list of losing players.
func (s *ShadowPlay) GetLosers() []*Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.State != ShadowPlayEchoesWin && s.State != ShadowPlayShadesWin {
		return nil
	}

	var losers []*Player
	var losingRole PlayerRole

	if s.State == ShadowPlayEchoesWin {
		losingRole = RoleShade
	} else {
		losingRole = RoleEcho
	}

	for _, player := range s.Players {
		if player.Role == losingRole {
			losers = append(losers, player)
		}
	}

	return losers
}

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
