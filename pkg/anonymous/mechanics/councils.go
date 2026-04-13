// Package mechanics implements anonymous layer game mechanics for MURMUR.
// Phantom Councils: Persistent, private, anonymous coordination groups for
// high-Resonance Specters. Per ANONYMOUS_GAME_MECHANICS.md, Councils require
// Fortress mode and minimum Resonance 200.
package mechanics

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

// Phantom Council constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// CouncilMinResonance is the minimum Specter Resonance to create a council.
	CouncilMinResonance = 200

	// CouncilMinMembers is the minimum member count for active council.
	CouncilMinMembers = 3

	// CouncilMaxMembers is the maximum member count.
	CouncilMaxMembers = 13

	// CouncilMaxNameLength is the max council name length.
	CouncilMaxNameLength = 64

	// CouncilMaxPurposeLength is the max council purpose length.
	CouncilMaxPurposeLength = 256

	// CouncilWaveMaxAge is the content window (30 days).
	CouncilWaveMaxAge = 30 * 24 * time.Hour

	// CouncilAdmissionThreshold requires unanimous vote.
	CouncilAdmissionThreshold = 1.0

	// CouncilExpulsionThreshold requires two-thirds majority.
	CouncilExpulsionThreshold = 2.0 / 3.0

	// CouncilProposalThreshold requires simple majority.
	CouncilProposalThreshold = 0.5
)

// CouncilState represents the state of a Phantom Council.
type CouncilState uint8

const (
	// CouncilActive means council is active and operational.
	CouncilActive CouncilState = iota

	// CouncilDormant means council has fewer than 3 members.
	CouncilDormant

	// CouncilDisbanded means council has been dissolved.
	CouncilDisbanded
)

// MemberStatus represents a member's status in the council.
type MemberStatus uint8

const (
	// MemberPending means application is being voted on.
	MemberPending MemberStatus = iota

	// MemberActive means member is active.
	MemberActive

	// MemberExpelled means member was expelled.
	MemberExpelled

	// MemberDeparted means member left voluntarily.
	MemberDeparted
)

// VoteType represents the type of council vote.
type VoteType uint8

const (
	// VoteAdmit is a vote to admit a new member.
	VoteAdmit VoteType = iota

	// VoteExpel is a vote to expel a member.
	VoteExpel

	// VoteProposal is a vote on a proposal.
	VoteProposal
)

// VoteValue represents a vote choice.
type VoteValue uint8

const (
	// VoteFor supports the motion.
	VoteFor VoteValue = iota

	// VoteAgainst opposes the motion.
	VoteAgainst

	// VoteAbstain is a non-vote.
	VoteAbstain
)

// CouncilStateString returns a human-readable string for CouncilState.
func CouncilStateString(s CouncilState) string {
	switch s {
	case CouncilActive:
		return "Active"
	case CouncilDormant:
		return "Dormant"
	case CouncilDisbanded:
		return "Disbanded"
	default:
		return "Unknown"
	}
}

// MemberStatusString returns a human-readable string for MemberStatus.
func MemberStatusString(s MemberStatus) string {
	switch s {
	case MemberPending:
		return "Pending"
	case MemberActive:
		return "Active"
	case MemberExpelled:
		return "Expelled"
	case MemberDeparted:
		return "Departed"
	default:
		return "Unknown"
	}
}

// VoteValueString returns a human-readable string for VoteValue.
func VoteValueString(v VoteValue) string {
	switch v {
	case VoteFor:
		return "For"
	case VoteAgainst:
		return "Against"
	case VoteAbstain:
		return "Abstain"
	default:
		return "Unknown"
	}
}

// Council errors.
var (
	ErrCouncilInsufficientResonance = errors.New(
		"insufficient resonance for council")
	ErrCouncilNameTooLong    = errors.New("council name too long")
	ErrCouncilPurposeTooLong = errors.New("council purpose too long")
	ErrCouncilInvalidSize    = errors.New(
		"invalid council size (3-13 members)")
	ErrCouncilInvalidMinResonance = errors.New(
		"invalid minimum resonance (must be >= 200)")
	ErrCouncilNotActive     = errors.New("council is not active")
	ErrCouncilFull          = errors.New("council is full")
	ErrCouncilNotMember     = errors.New("not a member of this council")
	ErrCouncilNotPending    = errors.New("application not pending")
	ErrCouncilAlreadyMember = errors.New(
		"already a member of this council")
	ErrCouncilAlreadyVoted     = errors.New("already voted on this matter")
	ErrCouncilVoteNotFound     = errors.New("vote not found")
	ErrCouncilProposalNotFound = errors.New(
		"proposal not found")
)

// CouncilMember represents a member of a Phantom Council.
type CouncilMember struct {
	// SpecterKey is the member's Specter public key.
	SpecterKey [32]byte

	// Status is the member's current status.
	Status MemberStatus

	// JoinedAt is when the member was admitted.
	JoinedAt time.Time

	// DepartedAt is when the member left (if applicable).
	DepartedAt time.Time
}

// CouncilApplication represents an application to join a council.
type CouncilApplication struct {
	// ApplicantKey is the applicant's Specter public key.
	ApplicantKey [32]byte

	// AppliedAt is when the application was submitted.
	AppliedAt time.Time

	// Votes maps voter key to their vote.
	Votes map[string]VoteValue

	// ZKProof contains the Resonance threshold proof (placeholder).
	ZKProof []byte

	// Resolved indicates if voting is complete.
	Resolved bool

	// Admitted indicates if applicant was admitted.
	Admitted bool
}

// CouncilProposal represents a proposal for council voting.
type CouncilProposal struct {
	// ID uniquely identifies this proposal.
	ID [32]byte

	// ProposerKey is the proposing member's Specter key.
	ProposerKey [32]byte

	// Text is the proposal content.
	Text string

	// CreatedAt is when the proposal was created.
	CreatedAt time.Time

	// Votes maps voter key to their vote.
	Votes map[string]VoteValue

	// Resolved indicates if voting is complete.
	Resolved bool

	// Passed indicates if proposal passed.
	Passed bool
}

// ExpulsionVote represents a vote to expel a member.
type ExpulsionVote struct {
	// TargetKey is the member being voted on for expulsion.
	TargetKey [32]byte

	// InitiatedBy is the member who started the expulsion vote.
	InitiatedBy [32]byte

	// InitiatedAt is when the vote was initiated.
	InitiatedAt time.Time

	// Votes maps voter key to their vote.
	Votes map[string]VoteValue

	// Resolved indicates if voting is complete.
	Resolved bool

	// Expelled indicates if member was expelled.
	Expelled bool
}

// PhantomCouncil represents a persistent anonymous coordination group.
type PhantomCouncil struct {
	mu sync.RWMutex

	// ID uniquely identifies this council.
	ID [32]byte

	// Name is the council's display name.
	Name string

	// Purpose describes the council's focus.
	Purpose string

	// CreatorKey is the founding member's Specter key.
	CreatorKey [32]byte

	// CreatedAt is when the council was created.
	CreatedAt time.Time

	// MinResonance is the minimum Resonance to join.
	MinResonance float64

	// MaxMembers is the maximum member count.
	MaxMembers int

	// State is the current council state.
	State CouncilState

	// Members is the list of all members (including former).
	Members []*CouncilMember

	// memberByKey maps Specter key to member.
	memberByKey map[string]*CouncilMember

	// Applications is the list of pending applications.
	Applications []*CouncilApplication

	// applicationByKey maps applicant key to application.
	applicationByKey map[string]*CouncilApplication

	// Proposals is the list of proposals.
	Proposals []*CouncilProposal

	// ExpulsionVotes is the list of active expulsion votes.
	ExpulsionVotes []*ExpulsionVote

	// GroupKey is the shared symmetric encryption key.
	GroupKey [32]byte
}

// NewPhantomCouncil creates a new Phantom Council.
func NewPhantomCouncil(
	creator [32]byte,
	name, purpose string,
	minResonance float64,
	maxMembers int,
) (*PhantomCouncil, error) {
	if err := validateCouncilParams(name, purpose, minResonance, maxMembers); err != nil {
		return nil, err
	}

	council := initCouncil(creator, name, purpose, minResonance, maxMembers)
	addFoundingMember(council, creator)

	return council, nil
}

// validateCouncilParams validates the council creation parameters.
func validateCouncilParams(name, purpose string, minResonance float64, maxMembers int) error {
	if len(name) > CouncilMaxNameLength {
		return ErrCouncilNameTooLong
	}
	if len(purpose) > CouncilMaxPurposeLength {
		return ErrCouncilPurposeTooLong
	}
	if minResonance < CouncilMinResonance {
		return ErrCouncilInvalidMinResonance
	}
	if maxMembers < CouncilMinMembers || maxMembers > CouncilMaxMembers {
		return ErrCouncilInvalidSize
	}
	return nil
}

// initCouncil creates a new PhantomCouncil with generated ID and group key.
func initCouncil(creator [32]byte, name, purpose string, minResonance float64, maxMembers int) *PhantomCouncil {
	var id, groupKey [32]byte
	rand.Read(id[:])
	rand.Read(groupKey[:])

	return &PhantomCouncil{
		ID: id, Name: name, Purpose: purpose, CreatorKey: creator,
		CreatedAt: time.Now(), MinResonance: minResonance, MaxMembers: maxMembers,
		State: CouncilDormant, GroupKey: groupKey,
		Members: make([]*CouncilMember, 0), memberByKey: make(map[string]*CouncilMember),
		Applications: make([]*CouncilApplication, 0), applicationByKey: make(map[string]*CouncilApplication),
		Proposals: make([]*CouncilProposal, 0), ExpulsionVotes: make([]*ExpulsionVote, 0),
	}
}

// addFoundingMember adds the creator as the first council member.
func addFoundingMember(council *PhantomCouncil, creator [32]byte) {
	member := &CouncilMember{SpecterKey: creator, Status: MemberActive, JoinedAt: time.Now()}
	council.Members = append(council.Members, member)
	council.memberByKey[keyToHex(creator[:])] = member
}

// IsActive returns true if council is active.
func (c *PhantomCouncil) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == CouncilActive
}

// IsDormant returns true if council is dormant.
func (c *PhantomCouncil) IsDormant() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == CouncilDormant
}

// ActiveMemberCount returns the number of active members.
func (c *PhantomCouncil) ActiveMemberCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, m := range c.Members {
		if m.Status == MemberActive {
			count++
		}
	}
	return count
}

// GetActiveMembers returns all active members.
func (c *PhantomCouncil) GetActiveMembers() []*CouncilMember {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var active []*CouncilMember
	for _, m := range c.Members {
		if m.Status == MemberActive {
			active = append(active, m)
		}
	}
	return active
}

// IsMember checks if a Specter is an active member.
func (c *PhantomCouncil) IsMember(specter [32]byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	member := c.memberByKey[keyToHex(specter[:])]
	return member != nil && member.Status == MemberActive
}

// Apply submits an application to join the council.
func (c *PhantomCouncil) Apply(applicant [32]byte, zkProof []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.validateApplicantLocked(applicant); err != nil {
		return err
	}

	c.createApplicationLocked(applicant, zkProof)
	return nil
}

// validateApplicantLocked checks membership and capacity constraints.
// Must be called with c.mu held.
func (c *PhantomCouncil) validateApplicantLocked(applicant [32]byte) error {
	if c.isActiveMemberLocked(applicant) {
		return ErrCouncilAlreadyMember
	}
	if c.isAtCapacityLocked() {
		return ErrCouncilFull
	}
	return nil
}

// isActiveMemberLocked checks if the applicant is already an active member.
func (c *PhantomCouncil) isActiveMemberLocked(applicant [32]byte) bool {
	member := c.memberByKey[keyToHex(applicant[:])]
	return member != nil && member.Status == MemberActive
}

// isAtCapacityLocked checks if the council has reached max members.
func (c *PhantomCouncil) isAtCapacityLocked() bool {
	activeCount := 0
	for _, m := range c.Members {
		if m.Status == MemberActive {
			activeCount++
		}
	}
	return activeCount >= c.MaxMembers
}

// createApplicationLocked creates and registers a new application.
// Idempotent: returns early if application already exists.
func (c *PhantomCouncil) createApplicationLocked(applicant [32]byte, zkProof []byte) {
	key := keyToHex(applicant[:])
	if c.applicationByKey[key] != nil {
		return // Idempotent
	}

	app := &CouncilApplication{
		ApplicantKey: applicant,
		AppliedAt:    time.Now(),
		Votes:        make(map[string]VoteValue),
		ZKProof:      zkProof,
		Resolved:     false,
		Admitted:     false,
	}
	c.Applications = append(c.Applications, app)
	c.applicationByKey[key] = app
}

// VoteOnApplication casts a vote on a pending application.
func (c *PhantomCouncil) VoteOnApplication(
	voter, applicant [32]byte,
	vote VoteValue,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify voter is a member.
	voterMember := c.memberByKey[keyToHex(voter[:])]
	if voterMember == nil || voterMember.Status != MemberActive {
		return ErrCouncilNotMember
	}

	// Find application.
	app := c.applicationByKey[keyToHex(applicant[:])]
	if app == nil || app.Resolved {
		return ErrCouncilNotPending
	}

	// Check for duplicate vote.
	voterHex := keyToHex(voter[:])
	if _, exists := app.Votes[voterHex]; exists {
		return ErrCouncilAlreadyVoted
	}

	app.Votes[voterHex] = vote

	// Check if all members have voted.
	c.checkApplicationVotes(app)

	return nil
}

// voteCounts holds vote tallying results.
type voteCounts struct {
	eligible int // Members eligible to vote
	voted    int // Members who have voted
	forVote  int // Votes in favor
	against  int // Votes against
	abstain  int // Abstentions
}

// countVotes tallies votes from active members.
// excludeKey optionally excludes a member (e.g., the vote target).
func (c *PhantomCouncil) countVotes(votes map[string]VoteValue, excludeKey *[32]byte) voteCounts {
	var counts voteCounts
	for _, m := range c.Members {
		if m.Status != MemberActive {
			continue
		}
		if excludeKey != nil && m.SpecterKey == *excludeKey {
			continue
		}
		counts.eligible++

		vote, voted := votes[keyToHex(m.SpecterKey[:])]
		if !voted {
			continue
		}
		counts.voted++
		switch vote {
		case VoteFor:
			counts.forVote++
		case VoteAgainst:
			counts.against++
		case VoteAbstain:
			counts.abstain++
		}
	}
	return counts
}

// checkApplicationVotes evaluates if application voting is complete.
// Admission requires unanimous vote from all active members.
func (c *PhantomCouncil) checkApplicationVotes(app *CouncilApplication) {
	if app.Resolved {
		return
	}

	counts := c.countVotes(app.Votes, nil)

	// Any rejection immediately fails admission (unanimous required).
	if counts.against > 0 {
		app.Resolved = true
		app.Admitted = false
		return
	}

	// Not all members have voted yet.
	if counts.voted < counts.eligible {
		return
	}

	// All members voted unanimously for admission.
	app.Resolved = true
	app.Admitted = counts.forVote == counts.eligible
	if app.Admitted {
		c.admitMember(app.ApplicantKey)
	}
}

// admitMember adds a new member to the council.
func (c *PhantomCouncil) admitMember(specter [32]byte) {
	member := &CouncilMember{
		SpecterKey: specter,
		Status:     MemberActive,
		JoinedAt:   time.Now(),
	}

	c.Members = append(c.Members, member)
	c.memberByKey[keyToHex(specter[:])] = member

	// Update council state.
	c.updateState()

	// Rotate group key on membership change.
	c.rotateGroupKey()
}

// InitiateExpulsion starts an expulsion vote against a member.
func (c *PhantomCouncil) InitiateExpulsion(
	initiator, target [32]byte,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify initiator is a member.
	if !c.isMemberLocked(initiator) {
		return ErrCouncilNotMember
	}

	// Verify target is a member.
	if !c.isMemberLocked(target) {
		return ErrCouncilNotMember
	}

	// Check for existing expulsion vote.
	for _, ev := range c.ExpulsionVotes {
		if ev.TargetKey == target && !ev.Resolved {
			return nil // Already pending.
		}
	}

	ev := &ExpulsionVote{
		TargetKey:   target,
		InitiatedBy: initiator,
		InitiatedAt: time.Now(),
		Votes:       make(map[string]VoteValue),
		Resolved:    false,
		Expelled:    false,
	}

	// Initiator automatically votes for expulsion.
	ev.Votes[keyToHex(initiator[:])] = VoteFor

	c.ExpulsionVotes = append(c.ExpulsionVotes, ev)

	return nil
}

// VoteOnExpulsion casts a vote on a pending expulsion.
func (c *PhantomCouncil) VoteOnExpulsion(
	voter, target [32]byte,
	vote VoteValue,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify voter is a member.
	if !c.isMemberLocked(voter) {
		return ErrCouncilNotMember
	}

	// Find expulsion vote.
	var ev *ExpulsionVote
	for _, e := range c.ExpulsionVotes {
		if e.TargetKey == target && !e.Resolved {
			ev = e
			break
		}
	}
	if ev == nil {
		return ErrCouncilVoteNotFound
	}

	// Check for duplicate vote.
	voterHex := keyToHex(voter[:])
	if _, exists := ev.Votes[voterHex]; exists {
		return ErrCouncilAlreadyVoted
	}

	ev.Votes[voterHex] = vote

	// Check if voting is complete.
	c.checkExpulsionVotes(ev)

	return nil
}

// checkExpulsionVotes evaluates if expulsion voting is complete.
// Expulsion requires two-thirds majority.
func (c *PhantomCouncil) checkExpulsionVotes(ev *ExpulsionVote) {
	if ev.Resolved {
		return
	}

	counts := c.countVotes(ev.Votes, &ev.TargetKey)
	threshold := c.calculateExpulsionThreshold(counts.eligible)

	if c.resolveExpulsion(ev, counts, threshold) {
		return
	}

	// Check if success is mathematically impossible.
	remainingVotes := counts.eligible - counts.voted
	if counts.forVote+remainingVotes < threshold {
		ev.Resolved = true
		ev.Expelled = false
	}
}

// calculateExpulsionThreshold returns the minimum votes needed for expulsion.
func (c *PhantomCouncil) calculateExpulsionThreshold(eligible int) int {
	threshold := int(float64(eligible) * CouncilExpulsionThreshold)
	if threshold == 0 {
		threshold = 1
	}
	return threshold
}

// resolveExpulsion checks if expulsion has been decided and applies it.
func (c *PhantomCouncil) resolveExpulsion(ev *ExpulsionVote, counts voteCounts, threshold int) bool {
	if counts.forVote >= threshold {
		ev.Resolved = true
		ev.Expelled = true
		c.expelMember(ev.TargetKey)
		return true
	}
	return false
}

// expelMember removes a member from the council.
func (c *PhantomCouncil) expelMember(specter [32]byte) {
	member := c.memberByKey[keyToHex(specter[:])]
	if member != nil {
		member.Status = MemberExpelled
		member.DepartedAt = time.Now()
	}

	// Update council state.
	c.updateState()

	// Rotate group key on membership change.
	c.rotateGroupKey()
}

// Leave voluntarily departs from the council.
func (c *PhantomCouncil) Leave(specter [32]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	member := c.memberByKey[keyToHex(specter[:])]
	if member == nil || member.Status != MemberActive {
		return ErrCouncilNotMember
	}

	member.Status = MemberDeparted
	member.DepartedAt = time.Now()

	c.updateState()
	c.rotateGroupKey()

	return nil
}

// isMemberLocked checks membership without acquiring lock.
func (c *PhantomCouncil) isMemberLocked(specter [32]byte) bool {
	member := c.memberByKey[keyToHex(specter[:])]
	return member != nil && member.Status == MemberActive
}

// updateState updates council state based on membership.
func (c *PhantomCouncil) updateState() {
	activeCount := 0
	for _, m := range c.Members {
		if m.Status == MemberActive {
			activeCount++
		}
	}

	if activeCount >= CouncilMinMembers {
		c.State = CouncilActive
	} else if activeCount > 0 {
		c.State = CouncilDormant
	} else {
		c.State = CouncilDisbanded
	}
}

// rotateGroupKey generates a new group encryption key.
func (c *PhantomCouncil) rotateGroupKey() {
	// Generate new random key.
	rand.Read(c.GroupKey[:])
}

// CreateProposal creates a new proposal for council voting.
func (c *PhantomCouncil) CreateProposal(
	proposer [32]byte,
	text string,
) (*CouncilProposal, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check council is active.
	if c.State != CouncilActive {
		return nil, ErrCouncilNotActive
	}

	// Verify proposer is a member.
	if !c.isMemberLocked(proposer) {
		return nil, ErrCouncilNotMember
	}

	// Generate proposal ID.
	var idSeed []byte
	idSeed = append(idSeed, c.ID[:]...)
	idSeed = append(idSeed, proposer[:]...)
	idSeed = append(idSeed, []byte(text)...)
	var nonce [8]byte
	rand.Read(nonce[:])
	idSeed = append(idSeed, nonce[:]...)
	id := sha256.Sum256(idSeed)

	proposal := &CouncilProposal{
		ID:          id,
		ProposerKey: proposer,
		Text:        text,
		CreatedAt:   time.Now(),
		Votes:       make(map[string]VoteValue),
		Resolved:    false,
		Passed:      false,
	}

	c.Proposals = append(c.Proposals, proposal)

	return proposal, nil
}

// VoteOnProposal casts a vote on a pending proposal.
func (c *PhantomCouncil) VoteOnProposal(
	voter [32]byte,
	proposalID [32]byte,
	vote VoteValue,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify voter is a member.
	if !c.isMemberLocked(voter) {
		return ErrCouncilNotMember
	}

	// Find proposal.
	var proposal *CouncilProposal
	for _, p := range c.Proposals {
		if p.ID == proposalID && !p.Resolved {
			proposal = p
			break
		}
	}
	if proposal == nil {
		return ErrCouncilProposalNotFound
	}

	// Check for duplicate vote.
	voterHex := keyToHex(voter[:])
	if _, exists := proposal.Votes[voterHex]; exists {
		return ErrCouncilAlreadyVoted
	}

	proposal.Votes[voterHex] = vote

	// Check if voting is complete.
	c.checkProposalVotes(proposal)

	return nil
}

// checkProposalVotes evaluates if proposal voting is complete.
// Proposals pass with simple majority of non-abstaining votes.
func (c *PhantomCouncil) checkProposalVotes(p *CouncilProposal) {
	if p.Resolved {
		return
	}

	counts := c.countVotes(p.Votes, nil)

	// Wait for all members to vote.
	if counts.voted < counts.eligible {
		return
	}

	p.Resolved = true
	p.Passed = c.calculateProposalOutcome(counts)
}

// calculateProposalOutcome determines if a proposal passes.
func (c *PhantomCouncil) calculateProposalOutcome(counts voteCounts) bool {
	nonAbstain := counts.forVote + counts.against
	if nonAbstain == 0 {
		return false
	}
	return float64(counts.forVote)/float64(nonAbstain) > CouncilProposalThreshold
}

// GetPendingApplications returns all unresolved applications.
func (c *PhantomCouncil) GetPendingApplications() []*CouncilApplication {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var pending []*CouncilApplication
	for _, app := range c.Applications {
		if !app.Resolved {
			pending = append(pending, app)
		}
	}
	return pending
}

// GetPendingProposals returns all unresolved proposals.
func (c *PhantomCouncil) GetPendingProposals() []*CouncilProposal {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var pending []*CouncilProposal
	for _, p := range c.Proposals {
		if !p.Resolved {
			pending = append(pending, p)
		}
	}
	return pending
}

// GetPendingExpulsions returns all unresolved expulsion votes.
func (c *PhantomCouncil) GetPendingExpulsions() []*ExpulsionVote {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var pending []*ExpulsionVote
	for _, ev := range c.ExpulsionVotes {
		if !ev.Resolved {
			pending = append(pending, ev)
		}
	}
	return pending
}

// CouncilStore manages Phantom Council instances.
type CouncilStore struct {
	mu       sync.RWMutex
	councils map[string]*PhantomCouncil
}

// NewCouncilStore creates a new CouncilStore.
func NewCouncilStore() *CouncilStore {
	return &CouncilStore{
		councils: make(map[string]*PhantomCouncil),
	}
}

// AddCouncil adds a council to the store.
func (s *CouncilStore) AddCouncil(council *PhantomCouncil) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.councils[hex.EncodeToString(council.ID[:])] = council
}

// GetCouncil retrieves a council by ID.
func (s *CouncilStore) GetCouncil(id [32]byte) *PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.councils[hex.EncodeToString(id[:])]
}

// RemoveCouncil removes a council from the store.
func (s *CouncilStore) RemoveCouncil(id [32]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.councils, hex.EncodeToString(id[:]))
}

// GetActiveCouncils returns all active councils.
func (s *CouncilStore) GetActiveCouncils() []*PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*PhantomCouncil
	for _, c := range s.councils {
		if c.IsActive() {
			active = append(active, c)
		}
	}

	// Sort by creation time.
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt.Before(active[j].CreatedAt)
	})

	return active
}

// GetCouncilsForMember returns councils where Specter is a member.
func (s *CouncilStore) GetCouncilsForMember(specter [32]byte) []*PhantomCouncil {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var memberCouncils []*PhantomCouncil
	for _, c := range s.councils {
		if c.IsMember(specter) {
			memberCouncils = append(memberCouncils, c)
		}
	}
	return memberCouncils
}

// Count returns the number of councils in the store.
func (s *CouncilStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.councils)
}

// PruneDisbanded removes disbanded councils.
func (s *CouncilStore) PruneDisbanded() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	pruned := 0
	for id, c := range s.councils {
		if c.State == CouncilDisbanded {
			delete(s.councils, id)
			pruned++
		}
	}
	return pruned
}
