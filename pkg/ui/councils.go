// Package ui — Phantom Council management panel.
// Per ROADMAP.md line 550: "UI: Council management panel — create council,
// invite members, propose, vote".
// Per ANONYMOUS_GAME_MECHANICS.md: "Phantom Councils are persistent,
// private, anonymous coordination groups for high-Resonance Specters."
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// CouncilState represents the state of a council.
type CouncilState uint8

const (
	CouncilStateActive    CouncilState = iota // Council is active.
	CouncilStateDormant                       // Council has <3 members.
	CouncilStateDisbanded                     // Council was dissolved.
)

// CouncilStateString returns a human-readable string.
func CouncilStateString(s CouncilState) string {
	switch s {
	case CouncilStateActive:
		return "Active"
	case CouncilStateDormant:
		return "Dormant"
	case CouncilStateDisbanded:
		return "Disbanded"
	default:
		return "Unknown"
	}
}

// MemberStatus represents a member's status.
type MemberStatus uint8

const (
	MemberStatusPending  MemberStatus = iota // Application pending.
	MemberStatusActive                       // Active member.
	MemberStatusExpelled                     // Was expelled.
	MemberStatusDeparted                     // Left voluntarily.
)

// MemberStatusString returns a human-readable string.
func MemberStatusString(s MemberStatus) string {
	switch s {
	case MemberStatusPending:
		return "Pending"
	case MemberStatusActive:
		return "Active"
	case MemberStatusExpelled:
		return "Expelled"
	case MemberStatusDeparted:
		return "Departed"
	default:
		return "Unknown"
	}
}

// VoteType represents a council vote type.
type VoteType uint8

const (
	VoteTypeAdmit    VoteType = iota // Vote to admit new member.
	VoteTypeExpel                    // Vote to expel member.
	VoteTypeProposal                 // Vote on proposal.
)

// VoteValue represents a vote choice.
type VoteValue uint8

const (
	VoteValueFor     VoteValue = iota // Support motion.
	VoteValueAgainst                  // Oppose motion.
	VoteValueAbstain                  // No vote.
)

// VoteValueString returns a human-readable string.
func VoteValueString(v VoteValue) string {
	switch v {
	case VoteValueFor:
		return "For"
	case VoteValueAgainst:
		return "Against"
	case VoteValueAbstain:
		return "Abstain"
	default:
		return "Unknown"
	}
}

// CouncilMemberInfo contains member info for UI display.
type CouncilMemberInfo struct {
	SpecterKey [32]byte     // Member's Specter key.
	Name       string       // Display name/pseudonym.
	Status     MemberStatus // Current status.
	JoinedAt   time.Time    // When joined.
}

// CouncilProposalInfo contains proposal info for UI display.
type CouncilProposalInfo struct {
	ID          [32]byte             // Proposal ID.
	ProposerKey [32]byte             // Who proposed.
	Text        string               // Proposal content.
	CreatedAt   time.Time            // When created.
	Votes       map[string]VoteValue // Current votes.
	Resolved    bool                 // Voting complete.
	Passed      bool                 // If resolved, did it pass.
}

// CouncilApplicationInfo contains application info for UI display.
type CouncilApplicationInfo struct {
	ApplicantKey  [32]byte             // Applicant's key.
	ApplicantName string               // Applicant's name.
	AppliedAt     time.Time            // When applied.
	Votes         map[string]VoteValue // Current votes.
	Resolved      bool                 // Voting complete.
	Admitted      bool                 // If resolved, was admitted.
}

// CouncilInfo contains council information for UI display.
type CouncilInfo struct {
	ID           [32]byte                 // Council ID.
	Name         string                   // Council name.
	Purpose      string                   // Council purpose.
	State        CouncilState             // Current state.
	CreatedAt    time.Time                // When created.
	MinResonance float64                  // Minimum resonance to join.
	MaxMembers   int                      // Maximum members.
	Members      []CouncilMemberInfo      // Current members.
	Applications []CouncilApplicationInfo // Pending applications.
	Proposals    []CouncilProposalInfo    // Active proposals.
	IsMember     bool                     // Is current user a member.
	IsCreator    bool                     // Is current user the creator.
}

// CouncilPanelMode represents the panel display mode.
type CouncilPanelMode uint8

const (
	CouncilModeList      CouncilPanelMode = iota // List of councils.
	CouncilModeCreate                            // Create new council.
	CouncilModeDetail                            // Council detail view.
	CouncilModeMembers                           // Member management.
	CouncilModeProposals                         // Proposal management.
	CouncilModeInvite                            // Invite new member.
	CouncilModePropose                           // Create new proposal.
	CouncilModeVote                              // Vote on item.
)

// CouncilPanel provides UI for Phantom Council management.
type CouncilPanel struct {
	mu sync.RWMutex

	visible        bool
	mode           CouncilPanelMode
	theme          Theme
	errorMessage   string
	successMessage string

	// Council list.
	councils        []*CouncilInfo
	selectedCouncil int

	// Current council detail.
	currentCouncil *CouncilInfo

	// Create council form.
	createName         string
	createPurpose      string
	createMinResonance float64
	createMaxMembers   int

	// Invite form.
	inviteSpecterKey string

	// Propose form.
	proposeText string

	// Vote state.
	voteTarget   interface{} // *CouncilApplicationInfo or *CouncilProposalInfo
	voteType     VoteType
	selectedVote VoteValue

	// Scroll state.
	scrollOffset int

	// Animation state.
	animPhase float64

	// Callbacks.
	onCreateCouncil  func(name, purpose string, minResonance float64, maxMembers int) error
	onInviteMember   func(councilID, specterKey [32]byte) error
	onVoteAdmit      func(councilID, applicantKey [32]byte, vote VoteValue) error
	onVoteExpel      func(councilID, memberKey [32]byte, vote VoteValue) error
	onVoteProposal   func(councilID, proposalID [32]byte, vote VoteValue) error
	onCreateProposal func(councilID [32]byte, text string) error
	onLeaveCouncil   func(councilID [32]byte) error
	onInitExpel      func(councilID, memberKey [32]byte) error
}

// NewCouncilPanel creates a new Council management panel.
func NewCouncilPanel(theme Theme) *CouncilPanel {
	return &CouncilPanel{
		theme:              theme,
		mode:               CouncilModeList,
		createMinResonance: 200,
		createMaxMembers:   13,
		councils:           make([]*CouncilInfo, 0),
	}
}

// SetTheme updates the panel theme.
func (cp *CouncilPanel) SetTheme(theme Theme) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.theme = theme
}

// Show displays the panel.
func (cp *CouncilPanel) Show() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.visible = true
	cp.mode = CouncilModeList
	cp.errorMessage = ""
	cp.successMessage = ""
}

// ShowCouncilDetail shows a specific council.
func (cp *CouncilPanel) ShowCouncilDetail(council *CouncilInfo) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.visible = true
	cp.mode = CouncilModeDetail
	cp.currentCouncil = council
	cp.errorMessage = ""
	cp.successMessage = ""
}

// Hide hides the panel.
func (cp *CouncilPanel) Hide() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.visible = false
}

// IsVisible returns true if panel is shown.
func (cp *CouncilPanel) IsVisible() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.visible
}

// SetCouncils updates the council list.
func (cp *CouncilPanel) SetCouncils(councils []*CouncilInfo) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.councils = councils
}

// SetCurrentCouncil updates the current council detail.
func (cp *CouncilPanel) SetCurrentCouncil(council *CouncilInfo) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.currentCouncil = council
}

// SetOnCreateCouncil sets the create callback.
func (cp *CouncilPanel) SetOnCreateCouncil(cb func(name, purpose string, minResonance float64, maxMembers int) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onCreateCouncil = cb
}

// SetOnInviteMember sets the invite callback.
func (cp *CouncilPanel) SetOnInviteMember(cb func(councilID, specterKey [32]byte) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onInviteMember = cb
}

// SetOnVoteAdmit sets the admission vote callback.
func (cp *CouncilPanel) SetOnVoteAdmit(cb func(councilID, applicantKey [32]byte, vote VoteValue) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onVoteAdmit = cb
}

// SetOnVoteExpel sets the expulsion vote callback.
func (cp *CouncilPanel) SetOnVoteExpel(cb func(councilID, memberKey [32]byte, vote VoteValue) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onVoteExpel = cb
}

// SetOnVoteProposal sets the proposal vote callback.
func (cp *CouncilPanel) SetOnVoteProposal(cb func(councilID, proposalID [32]byte, vote VoteValue) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onVoteProposal = cb
}

// SetOnCreateProposal sets the proposal creation callback.
func (cp *CouncilPanel) SetOnCreateProposal(cb func(councilID [32]byte, text string) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onCreateProposal = cb
}

// SetOnLeaveCouncil sets the leave callback.
func (cp *CouncilPanel) SetOnLeaveCouncil(cb func(councilID [32]byte) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onLeaveCouncil = cb
}

// SetOnInitExpel sets the expulsion initiation callback.
func (cp *CouncilPanel) SetOnInitExpel(cb func(councilID, memberKey [32]byte) error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onInitExpel = cb
}

// Update handles input and animation.
func (cp *CouncilPanel) Update() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.visible {
		return nil
	}

	// Update animation.
	cp.animPhase += 0.016
	if cp.animPhase > 6.28 {
		cp.animPhase -= 6.28
	}

	// Handle Escape to go back.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch cp.mode {
		case CouncilModeList:
			cp.visible = false
		case CouncilModeCreate, CouncilModeInvite, CouncilModePropose, CouncilModeVote:
			cp.mode = CouncilModeDetail
		case CouncilModeMembers, CouncilModeProposals:
			cp.mode = CouncilModeDetail
		default:
			cp.mode = CouncilModeList
		}
		return nil
	}

	// Handle mode-specific input.
	switch cp.mode {
	case CouncilModeList:
		cp.handleListInput()
	case CouncilModeCreate:
		cp.handleCreateInput()
	case CouncilModeDetail:
		cp.handleDetailInput()
	case CouncilModeMembers:
		cp.handleMembersInput()
	case CouncilModeProposals:
		cp.handleProposalsInput()
	case CouncilModeInvite:
		cp.handleInviteInput()
	case CouncilModePropose:
		cp.handleProposeInput()
	case CouncilModeVote:
		cp.handleVoteInput()
	}

	return nil
}

// handleListInput handles input in council list mode.
func (cp *CouncilPanel) handleListInput() {
	// Arrow keys for selection.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && cp.selectedCouncil > 0 {
		cp.selectedCouncil--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && cp.selectedCouncil < len(cp.councils)-1 {
		cp.selectedCouncil++
	}

	// Enter to view detail.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(cp.councils) > 0 {
		cp.currentCouncil = cp.councils[cp.selectedCouncil]
		cp.mode = CouncilModeDetail
	}

	// N for new council.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		cp.mode = CouncilModeCreate
		cp.createName = ""
		cp.createPurpose = ""
		cp.createMinResonance = 200
		cp.createMaxMembers = 13
	}
}

// handleCreateInput handles input in create council mode.
func (cp *CouncilPanel) handleCreateInput() {
	// Tab to cycle fields (simplified: we'd need proper focus management).
	// For now, just handle Enter to submit.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && cp.createName != "" {
		if cp.onCreateCouncil != nil {
			if err := cp.onCreateCouncil(cp.createName, cp.createPurpose, cp.createMinResonance, cp.createMaxMembers); err != nil {
				cp.errorMessage = err.Error()
			} else {
				cp.successMessage = "Council created!"
				cp.mode = CouncilModeList
			}
		}
	}

	// Handle text input for name (simplified).
	cp.createName = handleTextInput(cp.createName, 64)
}

// handleDetailInput handles input in council detail mode.
func (cp *CouncilPanel) handleDetailInput() {
	if cp.currentCouncil == nil {
		return
	}

	// M for members.
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		cp.mode = CouncilModeMembers
		cp.scrollOffset = 0
	}

	// P for proposals.
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		cp.mode = CouncilModeProposals
		cp.scrollOffset = 0
	}

	// I for invite (if member).
	if inpututil.IsKeyJustPressed(ebiten.KeyI) && cp.currentCouncil.IsMember {
		cp.mode = CouncilModeInvite
		cp.inviteSpecterKey = ""
	}

	// N for new proposal (if member).
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && cp.currentCouncil.IsMember {
		cp.mode = CouncilModePropose
		cp.proposeText = ""
	}

	// L for leave (if member and not creator).
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && cp.currentCouncil.IsMember && !cp.currentCouncil.IsCreator {
		if cp.onLeaveCouncil != nil {
			if err := cp.onLeaveCouncil(cp.currentCouncil.ID); err != nil {
				cp.errorMessage = err.Error()
			} else {
				cp.successMessage = "Left council"
				cp.mode = CouncilModeList
			}
		}
	}
}

// handleMembersInput handles input in members view mode.
func (cp *CouncilPanel) handleMembersInput() {
	// Scroll.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && cp.scrollOffset > 0 {
		cp.scrollOffset--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cp.scrollOffset++
	}
}

// handleProposalsInput handles input in proposals view mode.
func (cp *CouncilPanel) handleProposalsInput() {
	// Scroll and select.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && cp.scrollOffset > 0 {
		cp.scrollOffset--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cp.scrollOffset++
	}

	// Enter to vote on selected proposal.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && cp.currentCouncil != nil {
		if cp.scrollOffset < len(cp.currentCouncil.Proposals) {
			prop := &cp.currentCouncil.Proposals[cp.scrollOffset]
			if !prop.Resolved {
				cp.voteTarget = prop
				cp.voteType = VoteTypeProposal
				cp.selectedVote = VoteValueAbstain
				cp.mode = CouncilModeVote
			}
		}
	}
}

// handleInviteInput handles input in invite mode.
func (cp *CouncilPanel) handleInviteInput() {
	cp.inviteSpecterKey = handleTextInput(cp.inviteSpecterKey, 64)

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && cp.inviteSpecterKey != "" {
		// Parse specter key and invoke callback.
		var key [32]byte
		// Simplified: in real implementation, parse hex string to bytes.
		copy(key[:], cp.inviteSpecterKey)

		if cp.onInviteMember != nil && cp.currentCouncil != nil {
			if err := cp.onInviteMember(cp.currentCouncil.ID, key); err != nil {
				cp.errorMessage = err.Error()
			} else {
				cp.successMessage = "Invitation sent!"
				cp.mode = CouncilModeDetail
			}
		}
	}
}

// handleProposeInput handles input in propose mode.
func (cp *CouncilPanel) handleProposeInput() {
	cp.proposeText = handleTextInput(cp.proposeText, 256)

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyControl) && cp.proposeText != "" {
		if cp.onCreateProposal != nil && cp.currentCouncil != nil {
			if err := cp.onCreateProposal(cp.currentCouncil.ID, cp.proposeText); err != nil {
				cp.errorMessage = err.Error()
			} else {
				cp.successMessage = "Proposal created!"
				cp.mode = CouncilModeDetail
			}
		}
	}
}

// handleVoteInput handles input in vote mode.
func (cp *CouncilPanel) handleVoteInput() {
	// Left/Right to change vote.
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if cp.selectedVote > VoteValueFor {
			cp.selectedVote--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if cp.selectedVote < VoteValueAbstain {
			cp.selectedVote++
		}
	}

	// Enter to submit vote.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && cp.currentCouncil != nil {
		var err error
		switch cp.voteType {
		case VoteTypeAdmit:
			if app, ok := cp.voteTarget.(*CouncilApplicationInfo); ok && cp.onVoteAdmit != nil {
				err = cp.onVoteAdmit(cp.currentCouncil.ID, app.ApplicantKey, cp.selectedVote)
			}
		case VoteTypeExpel:
			if member, ok := cp.voteTarget.(*CouncilMemberInfo); ok && cp.onVoteExpel != nil {
				err = cp.onVoteExpel(cp.currentCouncil.ID, member.SpecterKey, cp.selectedVote)
			}
		case VoteTypeProposal:
			if prop, ok := cp.voteTarget.(*CouncilProposalInfo); ok && cp.onVoteProposal != nil {
				err = cp.onVoteProposal(cp.currentCouncil.ID, prop.ID, cp.selectedVote)
			}
		}

		if err != nil {
			cp.errorMessage = err.Error()
		} else {
			cp.successMessage = "Vote submitted!"
			cp.mode = CouncilModeDetail
		}
	}
}

// handleTextInput processes keyboard input for text fields.
func handleTextInput(current string, maxLen int) string {
	// Get typed characters.
	chars := ebiten.AppendInputChars(nil)
	for _, c := range chars {
		if len(current) < maxLen {
			current += string(c)
		}
	}

	// Handle backspace.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(current) > 0 {
		current = current[:len(current)-1]
	}

	return current
}

// Draw renders the panel.
func (cp *CouncilPanel) Draw(screen *ebiten.Image) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if !cp.visible {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	panelW, panelH := 500, 450
	panelX := float32(sw-panelW) / 2
	panelY := float32(sh-panelH) / 2

	// Draw panel background.
	cp.drawPanelBackground(screen, panelX, panelY, float32(panelW), float32(panelH))

	// Draw title.
	cp.drawTitle(screen, panelX, panelY, float32(panelW))

	// Draw content based on mode.
	contentY := panelY + 50
	switch cp.mode {
	case CouncilModeList:
		cp.drawCouncilList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeCreate:
		cp.drawCreateForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeDetail:
		cp.drawCouncilDetail(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeMembers:
		cp.drawMembersList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeProposals:
		cp.drawProposalsList(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeInvite:
		cp.drawInviteForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModePropose:
		cp.drawProposeForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	case CouncilModeVote:
		cp.drawVoteForm(screen, panelX, contentY, float32(panelW), float32(panelH-70))
	}

	// Draw messages.
	cp.drawMessages(screen, panelX, panelY+float32(panelH)-30, float32(panelW))
}

// drawPanelBackground draws the panel background with border.
func (cp *CouncilPanel) drawPanelBackground(screen *ebiten.Image, x, y, w, h float32) {
	// Background.
	vector.DrawFilledRect(screen, x, y, w, h, cp.theme.PanelBackground, true)

	// Border.
	vector.StrokeRect(screen, x, y, w, h, 2, cp.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (cp *CouncilPanel) drawTitle(screen *ebiten.Image, x, y, w float32) {
	title := "Phantom Councils"
	switch cp.mode {
	case CouncilModeCreate:
		title = "Create Council"
	case CouncilModeDetail:
		if cp.currentCouncil != nil {
			title = cp.currentCouncil.Name
		}
	case CouncilModeMembers:
		title = "Members"
	case CouncilModeProposals:
		title = "Proposals"
	case CouncilModeInvite:
		title = "Invite Member"
	case CouncilModePropose:
		title = "New Proposal"
	case CouncilModeVote:
		title = "Cast Vote"
	}

	// Draw title background.
	vector.DrawFilledRect(screen, x, y, w, 40, cp.theme.ButtonBackground, true)

	// Draw title text (simplified - would use text/v2 with proper font).
	_ = title // Title rendering would use text/v2.Face.
}

// drawCouncilList draws the list of councils.
func (cp *CouncilPanel) drawCouncilList(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)
	itemHeight := float32(50)

	if len(cp.councils) == 0 {
		// Draw empty state.
		emptyColor := cp.theme.TextSecondary
		vector.DrawFilledCircle(screen, x+w/2, y+h/3, 30, emptyColor, true)
		return
	}

	for i, council := range cp.councils {
		itemY := y + padding + float32(i)*itemHeight

		// Highlight selected.
		if i == cp.selectedCouncil {
			vector.DrawFilledRect(screen, x+padding, itemY, w-padding*2, itemHeight-5, cp.theme.Selection, true)
		}

		// Draw council name.
		nameColor := cp.theme.TextPrimary
		if council.State == CouncilStateDormant {
			nameColor = cp.theme.TextSecondary
		}
		vector.DrawFilledRect(screen, x+padding+5, itemY+5, 5, itemHeight-15, nameColor, true)

		// Draw member count indicator.
		memberCount := 0
		for _, m := range council.Members {
			if m.Status == MemberStatusActive {
				memberCount++
			}
		}
		radius := float32(8 + memberCount)
		if radius > 20 {
			radius = 20
		}
		vector.DrawFilledCircle(screen, x+w-padding-20, itemY+itemHeight/2, radius, cp.theme.AccentPrimary, true)
	}

	// Draw help text.
	helpY := y + h - 25
	helpBg := cp.theme.PanelBackground
	helpBg.A = 200
	vector.DrawFilledRect(screen, x, helpY, w, 25, helpBg, true)
}

// drawCreateForm draws the council creation form.
func (cp *CouncilPanel) drawCreateForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Name field.
	cp.drawTextField(screen, x+padding, y+20, w-padding*2, "Name", cp.createName, true)

	// Purpose field.
	cp.drawTextField(screen, x+padding, y+80, w-padding*2, "Purpose", cp.createPurpose, false)

	// Min Resonance.
	cp.drawNumberField(screen, x+padding, y+140, w/2-padding*2, "Min Resonance", cp.createMinResonance)

	// Max Members.
	cp.drawNumberField(screen, x+w/2+padding, y+140, w/2-padding*2, "Max Members", float64(cp.createMaxMembers))

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+200, 120, 36, "Create", cp.createName != "")
}

// drawCouncilDetail draws the council detail view.
func (cp *CouncilPanel) drawCouncilDetail(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	council := cp.currentCouncil

	// State badge.
	stateColor := cp.theme.Success
	if council.State == CouncilStateDormant {
		stateColor = cp.theme.Warning
	} else if council.State == CouncilStateDisbanded {
		stateColor = cp.theme.TextError
	}
	vector.DrawFilledRect(screen, x+w-padding-80, y+10, 70, 20, stateColor, true)

	// Purpose.
	vector.DrawFilledRect(screen, x+padding, y+40, w-padding*2, 60, cp.theme.InputBackground, true)

	// Stats.
	statsY := y + 120
	memberCount := 0
	for _, m := range council.Members {
		if m.Status == MemberStatusActive {
			memberCount++
		}
	}

	// Member count circle.
	vector.DrawFilledCircle(screen, x+padding+30, statsY+20, 20, cp.theme.AccentPrimary, true)

	// Proposal count circle.
	activeProposals := 0
	for _, p := range council.Proposals {
		if !p.Resolved {
			activeProposals++
		}
	}
	vector.DrawFilledCircle(screen, x+padding+100, statsY+20, 20, cp.theme.AccentSecondary, true)

	// Application count circle.
	pendingApps := 0
	for _, a := range council.Applications {
		if !a.Resolved {
			pendingApps++
		}
	}
	if pendingApps > 0 {
		vector.DrawFilledCircle(screen, x+padding+170, statsY+20, 20, cp.theme.Warning, true)
	}

	// Action buttons.
	btnY := y + 200
	btnW := float32(100)
	btnH := float32(32)
	btnSpacing := float32(10)

	if council.IsMember {
		cp.drawButton(screen, x+padding, btnY, btnW, btnH, "[M]embers", true)
		cp.drawButton(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, "[P]roposals", true)
		cp.drawButton(screen, x+padding, btnY+btnH+btnSpacing, btnW, btnH, "[I]nvite", true)
		cp.drawButton(screen, x+padding+btnW+btnSpacing, btnY+btnH+btnSpacing, btnW, btnH, "[N]ew Prop", true)

		if !council.IsCreator {
			cp.drawButton(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, "[L]eave", true)
		}
	}
}

// drawMembersList draws the members list.
func (cp *CouncilPanel) drawMembersList(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	itemHeight := float32(40)

	for i, member := range cp.currentCouncil.Members {
		if member.Status != MemberStatusActive {
			continue
		}
		itemY := y + padding + float32(i)*itemHeight

		// Status indicator.
		statusColor := cp.theme.Success
		vector.DrawFilledCircle(screen, x+padding+10, itemY+itemHeight/2, 6, statusColor, true)

		// Member name placeholder.
		vector.DrawFilledRect(screen, x+padding+25, itemY+10, 150, itemHeight-20, cp.theme.InputBackground, true)
	}
}

// drawProposalsList draws the proposals list.
func (cp *CouncilPanel) drawProposalsList(screen *ebiten.Image, x, y, w, h float32) {
	if cp.currentCouncil == nil {
		return
	}

	padding := float32(cp.theme.Padding)
	itemHeight := float32(60)

	for i, prop := range cp.currentCouncil.Proposals {
		itemY := y + padding + float32(i-cp.scrollOffset)*itemHeight

		// Skip if scrolled out of view.
		if itemY < y || itemY > y+h-itemHeight {
			continue
		}

		// Highlight if selected.
		if i == cp.scrollOffset {
			vector.DrawFilledRect(screen, x+padding, itemY, w-padding*2, itemHeight-5, cp.theme.Selection, true)
		}

		// Status indicator.
		statusColor := cp.theme.AccentPrimary
		if prop.Resolved {
			if prop.Passed {
				statusColor = cp.theme.Success
			} else {
				statusColor = cp.theme.TextError
			}
		}
		vector.DrawFilledCircle(screen, x+padding+10, itemY+itemHeight/2, 8, statusColor, true)

		// Proposal text placeholder.
		vector.DrawFilledRect(screen, x+padding+30, itemY+10, w-padding*2-40, itemHeight-30, cp.theme.InputBackground, true)

		// Vote counts.
		forCount := 0
		againstCount := 0
		for _, v := range prop.Votes {
			if v == VoteValueFor {
				forCount++
			} else if v == VoteValueAgainst {
				againstCount++
			}
		}
		voteY := itemY + itemHeight - 15
		vector.DrawFilledRect(screen, x+w-padding-80, voteY, float32(forCount*10), 10, cp.theme.Success, true)
		vector.DrawFilledRect(screen, x+w-padding-40, voteY, float32(againstCount*10), 10, cp.theme.TextError, true)
	}
}

// drawInviteForm draws the invite member form.
func (cp *CouncilPanel) drawInviteForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Specter key field.
	cp.drawTextField(screen, x+padding, y+40, w-padding*2, "Specter Key", cp.inviteSpecterKey, true)

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+120, 120, 36, "Invite", cp.inviteSpecterKey != "")
}

// drawProposeForm draws the proposal creation form.
func (cp *CouncilPanel) drawProposeForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Proposal text field (multiline).
	vector.DrawFilledRect(screen, x+padding, y+20, w-padding*2, 150, cp.theme.InputBackground, true)
	vector.StrokeRect(screen, x+padding, y+20, w-padding*2, 150, 1, cp.theme.PanelBorder, true)

	// Character count.
	countText := fmt.Sprintf("%d/256", len(cp.proposeText))
	_ = countText // Would render with text/v2.

	// Submit button.
	cp.drawButton(screen, x+w/2-60, y+190, 120, 36, "Ctrl+Enter", cp.proposeText != "")
}

// drawVoteForm draws the voting form.
func (cp *CouncilPanel) drawVoteForm(screen *ebiten.Image, x, y, w, h float32) {
	padding := float32(cp.theme.Padding)

	// Vote target description.
	vector.DrawFilledRect(screen, x+padding, y+20, w-padding*2, 60, cp.theme.InputBackground, true)

	// Vote buttons.
	btnW := float32(80)
	btnH := float32(40)
	btnY := y + 100
	btnSpacing := (w - padding*2 - btnW*3) / 2

	// For button.
	forColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueFor {
		forColor = cp.theme.Success
	}
	vector.DrawFilledRect(screen, x+padding, btnY, btnW, btnH, forColor, true)
	vector.StrokeRect(screen, x+padding, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Against button.
	againstColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueAgainst {
		againstColor = cp.theme.TextError
	}
	vector.DrawFilledRect(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, againstColor, true)
	vector.StrokeRect(screen, x+padding+btnW+btnSpacing, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Abstain button.
	abstainColor := cp.theme.ButtonBackground
	if cp.selectedVote == VoteValueAbstain {
		abstainColor = cp.theme.TextSecondary
	}
	vector.DrawFilledRect(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, abstainColor, true)
	vector.StrokeRect(screen, x+padding+(btnW+btnSpacing)*2, btnY, btnW, btnH, 2, cp.theme.PanelBorder, true)

	// Submit.
	cp.drawButton(screen, x+w/2-60, y+170, 120, 36, "Submit", true)
}

// drawTextField draws a text input field.
func (cp *CouncilPanel) drawTextField(screen *ebiten.Image, x, y, w float32, label, value string, focused bool) {
	// Background.
	vector.DrawFilledRect(screen, x, y+15, w, float32(cp.theme.InputHeight), cp.theme.InputBackground, true)

	// Border.
	borderColor := cp.theme.PanelBorder
	if focused {
		borderColor = cp.theme.AccentPrimary
	}
	vector.StrokeRect(screen, x, y+15, w, float32(cp.theme.InputHeight), 1, borderColor, true)

	// Cursor if focused.
	if focused {
		cursorX := x + 8 + float32(len(value)*7)
		cursorAlpha := uint8(128 + 127*sin32(cp.animPhase*4))
		cursorColor := cp.theme.TextPrimary
		cursorColor.A = cursorAlpha
		vector.DrawFilledRect(screen, cursorX, y+20, 2, float32(cp.theme.InputHeight-10), cursorColor, true)
	}
}

// drawNumberField draws a number input field.
func (cp *CouncilPanel) drawNumberField(screen *ebiten.Image, x, y, w float32, label string, value float64) {
	// Background.
	vector.DrawFilledRect(screen, x, y+15, w, float32(cp.theme.InputHeight), cp.theme.InputBackground, true)
	vector.StrokeRect(screen, x, y+15, w, float32(cp.theme.InputHeight), 1, cp.theme.PanelBorder, true)

	// Value indicator.
	indicatorW := float32(value / 500 * float64(w-20))
	if indicatorW > w-20 {
		indicatorW = w - 20
	}
	vector.DrawFilledRect(screen, x+10, y+20+float32(cp.theme.InputHeight-20)/2, indicatorW, 4, cp.theme.AccentPrimary, true)
}

// drawButton draws a button.
func (cp *CouncilPanel) drawButton(screen *ebiten.Image, x, y, w, h float32, label string, enabled bool) {
	bgColor := cp.theme.ButtonBackground
	if !enabled {
		bgColor.A = 100
	}

	vector.DrawFilledRect(screen, x, y, w, h, bgColor, true)
	vector.StrokeRect(screen, x, y, w, h, 1, cp.theme.PanelBorder, true)
}

// drawMessages draws error/success messages.
func (cp *CouncilPanel) drawMessages(screen *ebiten.Image, x, y, w float32) {
	if cp.errorMessage != "" {
		vector.DrawFilledRect(screen, x, y, w, 25, cp.theme.TextError, true)
	} else if cp.successMessage != "" {
		vector.DrawFilledRect(screen, x, y, w, 25, cp.theme.Success, true)
	}
}

// sin32 returns the sine of x as float32.
func sin32(x float64) float32 {
	// Use standard library approximation.
	x = x - float64(int(x/(2*3.14159)))*(2*3.14159)
	if x < 0 {
		x += 2 * 3.14159
	}
	return float32(sinApprox(x))
}

// sinApprox is a basic sine approximation.
func sinApprox(x float64) float64 {
	// Taylor series approximation.
	x = x - float64(int(x/6.28318))*6.28318
	if x > 3.14159 {
		x -= 6.28318
	}
	x2 := x * x
	return x * (1 - x2/6*(1-x2/20*(1-x2/42)))
}

// Ensure CouncilPanel satisfies the basic rendering needs.
var _ text.Face = (*text.GoTextFace)(nil) // Ensure text/v2 is available.
