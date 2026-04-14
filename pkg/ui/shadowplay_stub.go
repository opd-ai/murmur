// Package ui - Shadow Play game interface panel (stub for noebiten builds).
// Per ROADMAP.md line 492: "UI: Shadow Play game interface — role reveal,
// vote casting, round status, results".
//
//go:build noebiten
// +build noebiten

package ui

import (
	"sync"
	"time"
)

// ShadowPlayState represents the game state for UI display.
type ShadowPlayState uint8

const (
	ShadowPlayStateWaiting   ShadowPlayState = iota // Waiting for players.
	ShadowPlayStateActive                           // Game in progress.
	ShadowPlayStateVoting                           // Voting phase.
	ShadowPlayStateEchoesWin                        // Echoes won.
	ShadowPlayStateShadesWin                        // Shades won.
	ShadowPlayStateExpired                          // Game expired.
)

// ShadowPlayStateString returns a human-readable string.
func ShadowPlayStateString(s ShadowPlayState) string {
	switch s {
	case ShadowPlayStateWaiting:
		return "Waiting"
	case ShadowPlayStateActive:
		return "Active"
	case ShadowPlayStateVoting:
		return "Voting"
	case ShadowPlayStateEchoesWin:
		return "Echoes Win!"
	case ShadowPlayStateShadesWin:
		return "Shades Win!"
	case ShadowPlayStateExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// ShadowPlayerRole represents a player's hidden role.
type ShadowPlayerRole uint8

const (
	ShadowRoleUnknown ShadowPlayerRole = iota // Role not known.
	ShadowRoleEcho                            // Standard participant.
	ShadowRoleShade                           // Hidden disruptor.
)

// ShadowRoleString returns a human-readable role name.
func ShadowRoleString(r ShadowPlayerRole) string {
	switch r {
	case ShadowRoleEcho:
		return "Echo"
	case ShadowRoleShade:
		return "Shade"
	default:
		return "Unknown"
	}
}

// ShadowPlayPlayer contains player info for display.
type ShadowPlayPlayer struct {
	SpecterKey   [32]byte         // Specter identity.
	Name         string           // Display name.
	Role         ShadowPlayerRole // Role (may be unknown to viewer).
	IsEliminated bool             // True if eliminated.
	VoteCount    int              // Votes received this round.
	HasVoted     bool             // True if has voted this round.
}

// ShadowPlayGameInfo contains game information for UI display.
type ShadowPlayGameInfo struct {
	GameID       [32]byte           // Unique game identifier.
	State        ShadowPlayState    // Current game state.
	RoundNumber  int                // Current round number.
	RoundEndTime time.Time          // When current round ends.
	GameEndTime  time.Time          // When game ends.
	Players      []ShadowPlayPlayer // All players.
	MyRole       ShadowPlayerRole   // Current user's role.
	MyVotedFor   [32]byte           // Who user voted for (if any).
	HasVoted     bool               // User has voted this round.
}

// ShadowPlayPanelMode represents the panel display mode.
type ShadowPlayPanelMode uint8

const (
	ShadowPlayModeOverview ShadowPlayPanelMode = iota // Game overview.
	ShadowPlayModeVote                                // Vote casting screen.
	ShadowPlayModeResults                             // Round/game results.
	ShadowPlayModeRole                                // Role reveal screen.
)

// ShadowPlayPanel provides UI for Shadow Play interaction (stub).
type ShadowPlayPanel struct {
	mu sync.RWMutex

	visible     bool
	game        *ShadowPlayGameInfo
	mode        ShadowPlayPanelMode
	selectedIdx int
	theme       Theme

	onVote  func(gameID, targetSpecter [32]byte)
	onJoin  func(gameID [32]byte)
	onLeave func(gameID [32]byte)
}

// NewShadowPlayPanel creates a new Shadow Play game panel (stub).
func NewShadowPlayPanel(theme Theme) *ShadowPlayPanel {
	return &ShadowPlayPanel{
		theme: theme,
		mode:  ShadowPlayModeOverview,
	}
}

// SetTheme updates the panel theme.
func (sp *ShadowPlayPanel) SetTheme(theme Theme) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.theme = theme
}

// Show displays the panel with game info.
func (sp *ShadowPlayPanel) Show(game *ShadowPlayGameInfo) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.visible = true
	sp.game = game
	sp.selectedIdx = 0

	if game != nil {
		switch game.State {
		case ShadowPlayStateVoting:
			if !game.HasVoted {
				sp.mode = ShadowPlayModeVote
			} else {
				sp.mode = ShadowPlayModeOverview
			}
		case ShadowPlayStateEchoesWin, ShadowPlayStateShadesWin:
			sp.mode = ShadowPlayModeResults
		default:
			sp.mode = ShadowPlayModeOverview
		}
	}
}

// ShowRoleReveal shows the role reveal screen.
func (sp *ShadowPlayPanel) ShowRoleReveal(game *ShadowPlayGameInfo) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.visible = true
	sp.game = game
	sp.mode = ShadowPlayModeRole
}

// Hide hides the panel.
func (sp *ShadowPlayPanel) Hide() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.visible = false
}

// IsVisible returns true if panel is shown.
func (sp *ShadowPlayPanel) IsVisible() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.visible
}

// SetGame updates the game info.
func (sp *ShadowPlayPanel) SetGame(game *ShadowPlayGameInfo) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.game = game
}

// SetMode sets the panel mode.
func (sp *ShadowPlayPanel) SetMode(mode ShadowPlayPanelMode) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.mode = mode
}

// SetOnVote sets the vote callback.
func (sp *ShadowPlayPanel) SetOnVote(cb func(gameID, targetSpecter [32]byte)) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.onVote = cb
}

// SetOnJoin sets the join callback.
func (sp *ShadowPlayPanel) SetOnJoin(cb func(gameID [32]byte)) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.onJoin = cb
}

// SetOnLeave sets the leave callback.
func (sp *ShadowPlayPanel) SetOnLeave(cb func(gameID [32]byte)) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.onLeave = cb
}

// Update handles input (stub - no-op).
func (sp *ShadowPlayPanel) Update() error {
	return nil
}

// GetMode returns the current panel mode.
func (sp *ShadowPlayPanel) GetMode() ShadowPlayPanelMode {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.mode
}

// GetGame returns the current game info.
func (sp *ShadowPlayPanel) GetGame() *ShadowPlayGameInfo {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.game
}

// GetSelectedIndex returns the selected player index in vote mode.
func (sp *ShadowPlayPanel) GetSelectedIndex() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.selectedIdx
}

// SetSelectedIndex sets the selected player index.
func (sp *ShadowPlayPanel) SetSelectedIndex(idx int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.selectedIdx = idx
}

// GetEligibleVoteTargets returns players who can be voted for.
func (sp *ShadowPlayPanel) GetEligibleVoteTargets() []ShadowPlayPlayer {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.game == nil {
		return nil
	}

	var eligible []ShadowPlayPlayer
	for _, player := range sp.game.Players {
		if !player.IsEliminated {
			eligible = append(eligible, player)
		}
	}
	return eligible
}
