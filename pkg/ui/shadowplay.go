// Package ui - Shadow Play game interface panel.
// Per ROADMAP.md line 492: "UI: Shadow Play game interface — role reveal,
// vote casting, round status, results".
// Per ANONYMOUS_GAME_MECHANICS.md: "Shadow Play is a social deduction game
// that leverages anonymity as its core mechanic."
//

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// ShadowPlayPanel provides UI for Shadow Play interaction.
type ShadowPlayPanel struct {
	mu sync.RWMutex

	visible      bool
	game         *ShadowPlayGameInfo
	mode         ShadowPlayPanelMode
	selectedIdx  int // Selected player index for voting.
	errorMessage string
	theme        Theme

	// Animation state.
	roleRevealPhase float64
	resultPhase     float64

	// Callbacks.
	onVote  func(gameID, targetSpecter [32]byte)
	onJoin  func(gameID [32]byte)
	onLeave func(gameID [32]byte)
}

// NewShadowPlayPanel creates a new Shadow Play game panel.
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
	sp.errorMessage = ""
	sp.selectedIdx = 0

	// Determine initial mode based on game state.
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
	sp.roleRevealPhase = 0
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

// Update handles input and animation.
func (sp *ShadowPlayPanel) Update() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.visible {
		return nil
	}

	// Update animations.
	sp.updateAnimations()

	// Handle input based on mode.
	switch sp.mode {
	case ShadowPlayModeVote:
		sp.handleVoteInput()
	case ShadowPlayModeRole:
		sp.handleRoleInput()
	case ShadowPlayModeResults:
		sp.handleResultsInput()
	default:
		sp.handleOverviewInput()
	}

	return nil
}

// updateAnimations advances animation state.
func (sp *ShadowPlayPanel) updateAnimations() {
	if sp.mode == ShadowPlayModeRole && sp.roleRevealPhase < 1.0 {
		sp.roleRevealPhase += 0.02
		if sp.roleRevealPhase > 1.0 {
			sp.roleRevealPhase = 1.0
		}
	}

	if sp.mode == ShadowPlayModeResults && sp.resultPhase < 1.0 {
		sp.resultPhase += 0.03
		if sp.resultPhase > 1.0 {
			sp.resultPhase = 1.0
		}
	}
}

// handleOverviewInput processes input in overview mode.
func (sp *ShadowPlayPanel) handleOverviewInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sp.visible = false
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyV) && sp.game != nil {
		if sp.game.State == ShadowPlayStateVoting && !sp.game.HasVoted {
			sp.mode = ShadowPlayModeVote
			sp.selectedIdx = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) && sp.game != nil {
		sp.mode = ShadowPlayModeRole
		sp.roleRevealPhase = 0
	}
}

// handleVoteInput processes input in vote mode.
func (sp *ShadowPlayPanel) handleVoteInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sp.mode = ShadowPlayModeOverview
		return
	}

	if sp.game == nil {
		return
	}

	eligible := sp.getEligibleVoteTargets()
	if len(eligible) == 0 {
		return
	}

	sp.handleVoteNavigation(len(eligible))
	sp.handleVoteConfirmation(eligible)
}

// handleVoteNavigation processes up/down arrow keys for vote selection.
func (sp *ShadowPlayPanel) handleVoteNavigation(eligibleCount int) {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		sp.selectedIdx--
		if sp.selectedIdx < 0 {
			sp.selectedIdx = eligibleCount - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		sp.selectedIdx++
		if sp.selectedIdx >= eligibleCount {
			sp.selectedIdx = 0
		}
	}
}

// handleVoteConfirmation processes Enter key to confirm vote.
func (sp *ShadowPlayPanel) handleVoteConfirmation(eligible []ShadowPlayPlayer) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if sp.selectedIdx < len(eligible) {
			target := eligible[sp.selectedIdx]
			if sp.onVote != nil {
				sp.onVote(sp.game.GameID, target.SpecterKey)
			}
			sp.game.HasVoted = true
			sp.game.MyVotedFor = target.SpecterKey
			sp.mode = ShadowPlayModeOverview
		}
	}
}

// handleRoleInput processes input in role reveal mode.
func (sp *ShadowPlayPanel) handleRoleInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		sp.mode = ShadowPlayModeOverview
	}
}

// handleResultsInput processes input in results mode.
func (sp *ShadowPlayPanel) handleResultsInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		sp.mode = ShadowPlayModeOverview
	}
}

// getEligibleVoteTargets returns players who can be voted for.
func (sp *ShadowPlayPanel) getEligibleVoteTargets() []ShadowPlayPlayer {
	if sp.game == nil {
		return nil
	}

	var eligible []ShadowPlayPlayer
	for _, player := range sp.game.Players {
		// Can't vote for eliminated players or self.
		if player.IsEliminated {
			continue
		}
		eligible = append(eligible, player)
	}
	return eligible
}

// Draw renders the panel.
func (sp *ShadowPlayPanel) Draw(screen *ebiten.Image) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if !sp.visible || sp.game == nil {
		return
	}

	// Panel dimensions.
	screenW := float32(screen.Bounds().Dx())
	screenH := float32(screen.Bounds().Dy())
	panelW := float32(400)
	panelH := float32(500)
	x := (screenW - panelW) / 2
	y := (screenH - panelH) / 2

	// Draw panel background.
	sp.drawPanelBackground(screen, x, y, panelW, panelH)

	// Draw content based on mode.
	switch sp.mode {
	case ShadowPlayModeVote:
		sp.drawVoteMode(screen, x, y, panelW, panelH)
	case ShadowPlayModeRole:
		sp.drawRoleMode(screen, x, y, panelW, panelH)
	case ShadowPlayModeResults:
		sp.drawResultsMode(screen, x, y, panelW, panelH)
	default:
		sp.drawOverviewMode(screen, x, y, panelW, panelH)
	}
}

// drawPanelBackground renders the panel background.
func (sp *ShadowPlayPanel) drawPanelBackground(
	screen *ebiten.Image,
	x, y, w, h float32,
) {
	// Background.
	vector.DrawFilledRect(screen, x, y, w, h, sp.theme.PanelBackground, false)

	// Border.
	vector.StrokeRect(screen, x, y, w, h, 2, sp.theme.PanelBorder, false)
}

// drawOverviewMode renders the game overview.
func (sp *ShadowPlayPanel) drawOverviewMode(
	screen *ebiten.Image,
	x, y, w, h float32,
) {
	if defaultFont == nil {
		return
	}

	// Title.
	title := "Shadow Play"
	titleOpts := &text.DrawOptions{}
	titleOpts.GeoM.Translate(float64(x+w/2-60), float64(y+20))
	titleOpts.ColorScale.ScaleWithColor(sp.theme.TextPrimary)
	text.Draw(screen, title, defaultFont, titleOpts)

	// Game state.
	stateText := fmt.Sprintf("Status: %s", ShadowPlayStateString(sp.game.State))
	stateOpts := &text.DrawOptions{}
	stateOpts.GeoM.Translate(float64(x+20), float64(y+60))
	stateOpts.ColorScale.ScaleWithColor(sp.theme.TextSecondary)
	text.Draw(screen, stateText, defaultFont, stateOpts)

	// Round number.
	roundText := fmt.Sprintf("Round: %d", sp.game.RoundNumber)
	roundOpts := &text.DrawOptions{}
	roundOpts.GeoM.Translate(float64(x+20), float64(y+85))
	roundOpts.ColorScale.ScaleWithColor(sp.theme.TextSecondary)
	text.Draw(screen, roundText, defaultFont, roundOpts)

	// Time remaining.
	var timeRemaining time.Duration
	if sp.game.State == ShadowPlayStateVoting {
		timeRemaining = time.Until(sp.game.RoundEndTime)
	} else {
		timeRemaining = time.Until(sp.game.GameEndTime)
	}
	if timeRemaining < 0 {
		timeRemaining = 0
	}
	timeText := fmt.Sprintf("Time: %s", formatDuration(timeRemaining))
	timeOpts := &text.DrawOptions{}
	timeOpts.GeoM.Translate(float64(x+20), float64(y+110))
	timeOpts.ColorScale.ScaleWithColor(sp.theme.TextSecondary)
	text.Draw(screen, timeText, defaultFont, timeOpts)

	// Player list header.
	playerHeader := fmt.Sprintf("Players (%d):", len(sp.game.Players))
	headerOpts := &text.DrawOptions{}
	headerOpts.GeoM.Translate(float64(x+20), float64(y+145))
	headerOpts.ColorScale.ScaleWithColor(sp.theme.TextPrimary)
	text.Draw(screen, playerHeader, defaultFont, headerOpts)

	// Player list.
	sp.drawPlayerList(screen, x+20, y+170, w-40, float32(len(sp.game.Players)*25))

	// Controls hint.
	controlsY := y + h - 60
	if sp.game.State == ShadowPlayStateVoting && !sp.game.HasVoted {
		sp.drawControlHint(screen, x+20, controlsY, "[V] Cast Vote")
	}
	sp.drawControlHint(screen, x+20, controlsY+20, "[R] View Role")
	sp.drawControlHint(screen, x+20, controlsY+40, "[ESC] Close")
}

// drawPlayerList renders the list of players.
func (sp *ShadowPlayPanel) drawPlayerList(
	screen *ebiten.Image,
	x, y, w, h float32,
) {
	if defaultFont == nil {
		return
	}

	for i, player := range sp.game.Players {
		playerY := y + float32(i)*25

		// Status indicator.
		var status string
		statusColor := sp.theme.TextSecondary
		if player.IsEliminated {
			status = "[X]"
			statusColor = sp.theme.TextError
		} else if player.HasVoted {
			status = "[✓]"
			statusColor = sp.theme.Success
		} else {
			status = "[ ]"
		}

		// Name and status.
		name := player.Name
		if name == "" {
			name = fmt.Sprintf("Specter %d", i+1)
		}

		lineText := fmt.Sprintf("%s %s", status, name)
		if player.VoteCount > 0 && sp.game.State == ShadowPlayStateVoting {
			lineText += fmt.Sprintf(" (%d votes)", player.VoteCount)
		}

		opts := &text.DrawOptions{}
		opts.GeoM.Translate(float64(x), float64(playerY))
		opts.ColorScale.ScaleWithColor(statusColor)
		text.Draw(screen, lineText, defaultFont, opts)
	}
}

// drawVoteMode renders the vote casting screen.
func (sp *ShadowPlayPanel) drawVoteMode(
	screen *ebiten.Image,
	x, y, w, h float32,
) {
	if defaultFont == nil {
		return
	}

	// Title.
	title := "Cast Your Vote"
	titleOpts := &text.DrawOptions{}
	titleOpts.GeoM.Translate(float64(x+w/2-70), float64(y+20))
	titleOpts.ColorScale.ScaleWithColor(sp.theme.TextPrimary)
	text.Draw(screen, title, defaultFont, titleOpts)

	// Instructions.
	instructions := "Select a player to eliminate:"
	instrOpts := &text.DrawOptions{}
	instrOpts.GeoM.Translate(float64(x+20), float64(y+60))
	instrOpts.ColorScale.ScaleWithColor(sp.theme.TextSecondary)
	text.Draw(screen, instructions, defaultFont, instrOpts)

	// Eligible players.
	eligible := sp.getEligibleVoteTargets()
	for i, player := range eligible {
		playerY := y + 100 + float32(i)*30

		// Selection indicator.
		prefix := "  "
		textColor := sp.theme.TextSecondary
		if i == sp.selectedIdx {
			prefix = "> "
			textColor = sp.theme.AccentPrimary

			// Highlight background.
			vector.DrawFilledRect(screen, x+15, playerY-5, w-30, 25, sp.theme.Selection, false)
		}

		name := player.Name
		if name == "" {
			name = fmt.Sprintf("Specter %d", i+1)
		}
		lineText := prefix + name

		opts := &text.DrawOptions{}
		opts.GeoM.Translate(float64(x+20), float64(playerY))
		opts.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, lineText, defaultFont, opts)
	}

	// Controls.
	controlsY := y + h - 60
	sp.drawControlHint(screen, x+20, controlsY, "[↑/↓] Navigate")
	sp.drawControlHint(screen, x+20, controlsY+20, "[Enter] Confirm")
	sp.drawControlHint(screen, x+20, controlsY+40, "[ESC] Cancel")
}

// drawRoleMode renders the role reveal screen.
func (sp *ShadowPlayPanel) drawRoleMode(screen *ebiten.Image, x, y, w, h float32) {
	if defaultFont == nil {
		return
	}

	sp.drawRoleTitle(screen, x, y, w)
	sp.drawRoleReveal(screen, x, y, w, h)
	sp.drawRoleDescription(screen, x, y, w, h)
	sp.drawContinueHintIfReady(screen, x, y, h)
}

// drawRoleTitle draws the "Your Role" title.
func (sp *ShadowPlayPanel) drawRoleTitle(screen *ebiten.Image, x, y, w float32) {
	titleOpts := &text.DrawOptions{}
	titleOpts.GeoM.Translate(float64(x+w/2-50), float64(y+20))
	titleOpts.ColorScale.ScaleWithColor(sp.theme.TextPrimary)
	text.Draw(screen, "Your Role", defaultFont, titleOpts)
}

// drawRoleReveal draws the animated role reveal text with role-specific color.
func (sp *ShadowPlayPanel) drawRoleReveal(screen *ebiten.Image, x, y, w, h float32) {
	roleY := y + h/2 - 40
	roleText := ShadowRoleString(sp.game.MyRole)
	alpha := uint8(255 * sp.roleRevealPhase)
	roleColor := sp.getRoleColor(alpha)

	roleOpts := &text.DrawOptions{}
	roleOpts.GeoM.Translate(float64(x+w/2-30), float64(roleY))
	roleOpts.ColorScale.ScaleWithColor(roleColor)
	text.Draw(screen, roleText, defaultFont, roleOpts)
}

// getRoleColor returns the role-specific color with alpha.
func (sp *ShadowPlayPanel) getRoleColor(alpha uint8) color.RGBA {
	if sp.game.MyRole == ShadowRoleEcho {
		return color.RGBA{R: 60, G: 180, B: 120, A: alpha}
	}
	if sp.game.MyRole == ShadowRoleShade {
		return color.RGBA{R: 180, G: 60, B: 80, A: alpha}
	}
	roleColor := sp.theme.TextPrimary
	roleColor.A = alpha
	return roleColor
}

// drawRoleDescription draws the role description text.
func (sp *ShadowPlayPanel) drawRoleDescription(screen *ebiten.Image, x, y, w, h float32) {
	roleY := y + h/2 - 40
	alpha := uint8(255 * sp.roleRevealPhase)
	description := sp.getRoleDescription()

	descOpts := &text.DrawOptions{}
	descOpts.GeoM.Translate(float64(x+w/2-100), float64(roleY+50))
	descColor := sp.theme.TextSecondary
	descColor.A = alpha
	descOpts.ColorScale.ScaleWithColor(descColor)
	text.Draw(screen, description, defaultFont, descOpts)
}

// getRoleDescription returns the description for the user's role.
func (sp *ShadowPlayPanel) getRoleDescription() string {
	if sp.game.MyRole == ShadowRoleEcho {
		return "Find and eliminate the Shades!"
	}
	if sp.game.MyRole == ShadowRoleShade {
		return "Blend in and survive..."
	}
	return "Your role is unknown."
}

// drawContinueHintIfReady draws the continue hint once reveal is complete.
func (sp *ShadowPlayPanel) drawContinueHintIfReady(screen *ebiten.Image, x, y, h float32) {
	if sp.roleRevealPhase >= 1.0 {
		sp.drawControlHint(screen, x+20, y+h-40, "[Enter] Continue")
	}
}

// drawResultsMode renders the game results screen.
func (sp *ShadowPlayPanel) drawResultsMode(
	screen *ebiten.Image,
	x, y, w, h float32,
) {
	if defaultFont == nil {
		return
	}

	title, titleColor := sp.getTitleAndColor()
	sp.drawText(screen, title, x+w/2-80, y+40, titleColor)

	resultText := sp.getPlayerResultText()
	sp.drawText(screen, resultText, x+w/2-70, y+100, sp.theme.TextSecondary)

	sp.drawText(screen, "Final Standings:", x+20, y+150, sp.theme.TextPrimary)
	sp.drawFinalStandings(screen, x, y)
	sp.drawControlHint(screen, x+20, y+h-40, "[Enter] Close")
}

func (sp *ShadowPlayPanel) getTitleAndColor() (string, color.RGBA) {
	titleColor := sp.theme.TextPrimary
	switch sp.game.State {
	case ShadowPlayStateEchoesWin:
		titleColor.R, titleColor.G, titleColor.B = 60, 180, 120
		return "Echoes Victory!", titleColor
	case ShadowPlayStateShadesWin:
		titleColor.R, titleColor.G, titleColor.B = 180, 60, 80
		return "Shades Victory!", titleColor
	default:
		return "Game Over", titleColor
	}
}

func (sp *ShadowPlayPanel) getPlayerResultText() string {
	wonAsEcho := sp.game.State == ShadowPlayStateEchoesWin && sp.game.MyRole == ShadowRoleEcho
	wonAsShade := sp.game.State == ShadowPlayStateShadesWin && sp.game.MyRole == ShadowRoleShade

	if wonAsEcho {
		return "You won as Echo!"
	}
	if wonAsShade {
		return "You won as Shade!"
	}
	if sp.game.MyRole == ShadowRoleEcho {
		return "You lost as Echo."
	}
	if sp.game.MyRole == ShadowRoleShade {
		return "You lost as Shade."
	}
	return "Game complete."
}

func (sp *ShadowPlayPanel) drawFinalStandings(screen *ebiten.Image, x, y float32) {
	for i, player := range sp.game.Players {
		playerY := y + 180 + float32(i)*25
		name := player.Name
		if name == "" {
			name = fmt.Sprintf("Specter %d", i+1)
		}

		status := ""
		if player.IsEliminated {
			status = " [Eliminated]"
		}
		lineText := fmt.Sprintf("%s - %s%s", name, ShadowRoleString(player.Role), status)

		lineColor := sp.theme.TextSecondary
		if player.Role == ShadowRoleShade {
			lineColor.R, lineColor.G, lineColor.B = 180, 80, 100
		}

		sp.drawText(screen, lineText, x+20, playerY, lineColor)
	}
}

func (sp *ShadowPlayPanel) drawText(screen *ebiten.Image, content string, x, y float32, clr color.RGBA) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	opts.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, content, defaultFont, opts)
}

// drawControlHint draws a control hint text.
func (sp *ShadowPlayPanel) drawControlHint(
	screen *ebiten.Image,
	x, y float32,
	hint string,
) {
	if defaultFont == nil {
		return
	}

	opts := &text.DrawOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	opts.ColorScale.ScaleWithColor(sp.theme.TextPlaceholder)
	text.Draw(screen, hint, defaultFont, opts)
}

// formatDuration formats a duration as MM:SS.
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0:00"
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
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
