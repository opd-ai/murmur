// Package ui — Phantom Council management panel stub for non-Ebiten builds.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"time"
)

// CouncilState represents the state of a council.
type CouncilState uint8

const (
	CouncilStateActive CouncilState = iota
	CouncilStateDormant
	CouncilStateDisbanded
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
	MemberStatusPending MemberStatus = iota
	MemberStatusActive
	MemberStatusExpelled
	MemberStatusDeparted
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
	VoteTypeAdmit VoteType = iota
	VoteTypeExpel
	VoteTypeProposal
)

// VoteValue represents a vote choice.
type VoteValue uint8

const (
	VoteValueFor VoteValue = iota
	VoteValueAgainst
	VoteValueAbstain
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
	SpecterKey [32]byte
	Name       string
	Status     MemberStatus
	JoinedAt   time.Time
}

// CouncilProposalInfo contains proposal info for UI display.
type CouncilProposalInfo struct {
	ID          [32]byte
	ProposerKey [32]byte
	Text        string
	CreatedAt   time.Time
	Votes       map[string]VoteValue
	Resolved    bool
	Passed      bool
}

// CouncilApplicationInfo contains application info for UI display.
type CouncilApplicationInfo struct {
	ApplicantKey  [32]byte
	ApplicantName string
	AppliedAt     time.Time
	Votes         map[string]VoteValue
	Resolved      bool
	Admitted      bool
}

// CouncilInfo contains council information for UI display.
type CouncilInfo struct {
	ID           [32]byte
	Name         string
	Purpose      string
	State        CouncilState
	CreatedAt    time.Time
	MinResonance float64
	MaxMembers   int
	Members      []CouncilMemberInfo
	Applications []CouncilApplicationInfo
	Proposals    []CouncilProposalInfo
	IsMember     bool
	IsCreator    bool
}

// CouncilPanelMode represents the panel display mode.
type CouncilPanelMode uint8

const (
	CouncilModeList CouncilPanelMode = iota
	CouncilModeCreate
	CouncilModeDetail
	CouncilModeMembers
	CouncilModeProposals
	CouncilModeInvite
	CouncilModePropose
	CouncilModeVote
)

// CouncilPanel is a stub for non-Ebiten builds.
type CouncilPanel struct {
	visible bool
}

// NewCouncilPanel creates a stub panel.
func NewCouncilPanel(theme Theme) *CouncilPanel {
	return &CouncilPanel{}
}

// SetTheme is a no-op stub.
func (cp *CouncilPanel) SetTheme(theme Theme) {}

// Show is a no-op stub.
func (cp *CouncilPanel) Show() { cp.visible = true }

// ShowCouncilDetail is a no-op stub.
func (cp *CouncilPanel) ShowCouncilDetail(council *CouncilInfo) { cp.visible = true }

// Hide is a no-op stub.
func (cp *CouncilPanel) Hide() { cp.visible = false }

// IsVisible returns visibility state.
func (cp *CouncilPanel) IsVisible() bool { return cp.visible }

// SetCouncils is a no-op stub.
func (cp *CouncilPanel) SetCouncils(councils []*CouncilInfo) {}

// SetCurrentCouncil is a no-op stub.
func (cp *CouncilPanel) SetCurrentCouncil(council *CouncilInfo) {}

// SetOnCreateCouncil is a no-op stub.
func (cp *CouncilPanel) SetOnCreateCouncil(cb func(name, purpose string, minResonance float64, maxMembers int) error) {
}

// SetOnInviteMember is a no-op stub.
func (cp *CouncilPanel) SetOnInviteMember(cb func(councilID, specterKey [32]byte) error) {}

// SetOnVoteAdmit is a no-op stub.
func (cp *CouncilPanel) SetOnVoteAdmit(cb func(councilID, applicantKey [32]byte, vote VoteValue) error) {
}

// SetOnVoteExpel is a no-op stub.
func (cp *CouncilPanel) SetOnVoteExpel(cb func(councilID, memberKey [32]byte, vote VoteValue) error) {
}

// SetOnVoteProposal is a no-op stub.
func (cp *CouncilPanel) SetOnVoteProposal(cb func(councilID, proposalID [32]byte, vote VoteValue) error) {
}

// SetOnCreateProposal is a no-op stub.
func (cp *CouncilPanel) SetOnCreateProposal(cb func(councilID [32]byte, text string) error) {}

// SetOnLeaveCouncil is a no-op stub.
func (cp *CouncilPanel) SetOnLeaveCouncil(cb func(councilID [32]byte) error) {}

// SetOnInitExpel is a no-op stub.
func (cp *CouncilPanel) SetOnInitExpel(cb func(councilID, memberKey [32]byte) error) {}

// Update is a no-op stub.
func (cp *CouncilPanel) Update() error { return nil }
