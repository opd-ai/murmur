// Package ui — Phantom Council management panel.
// Per ROADMAP.md line 550: "UI: Council management panel — create council,
// invite members, propose, vote".
// Per ANONYMOUS_GAME_MECHANICS.md: "Phantom Councils are persistent,
// private, anonymous coordination groups for high-Resonance Specters."
//

//go:build !test
// +build !test

package ui

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	cp.handleVoteNavigation()
	cp.handleVoteSubmission()
}

// handleVoteNavigation processes left/right arrow keys for vote selection.
func (cp *CouncilPanel) handleVoteNavigation() {
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
}

// handleVoteSubmission processes Enter key to submit vote.
func (cp *CouncilPanel) handleVoteSubmission() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEnter) || cp.currentCouncil == nil {
		return
	}

	var err error
	switch cp.voteType {
	case VoteTypeAdmit:
		err = cp.submitAdmitVote()
	case VoteTypeExpel:
		err = cp.submitExpelVote()
	case VoteTypeProposal:
		err = cp.submitProposalVote()
	}

	if err != nil {
		cp.errorMessage = err.Error()
	} else {
		cp.successMessage = "Vote submitted!"
		cp.mode = CouncilModeDetail
	}
}

// submitAdmitVote submits an admit vote.
func (cp *CouncilPanel) submitAdmitVote() error {
	if app, ok := cp.voteTarget.(*CouncilApplicationInfo); ok && cp.onVoteAdmit != nil {
		return cp.onVoteAdmit(cp.currentCouncil.ID, app.ApplicantKey, cp.selectedVote)
	}
	return nil
}

// submitExpelVote submits an expel vote.
func (cp *CouncilPanel) submitExpelVote() error {
	if member, ok := cp.voteTarget.(*CouncilMemberInfo); ok && cp.onVoteExpel != nil {
		return cp.onVoteExpel(cp.currentCouncil.ID, member.SpecterKey, cp.selectedVote)
	}
	return nil
}

// submitProposalVote submits a proposal vote.
func (cp *CouncilPanel) submitProposalVote() error {
	if prop, ok := cp.voteTarget.(*CouncilProposalInfo); ok && cp.onVoteProposal != nil {
		return cp.onVoteProposal(cp.currentCouncil.ID, prop.ID, cp.selectedVote)
	}
	return nil
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
