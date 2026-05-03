// Package mechanics - Mark Voting system for community endorsement/challenge.
// Per ROADMAP.md: "Voting mechanics — community mark endorsement/challenge"
// This allows Specters to endorse (support) or challenge (dispute) existing marks,
// creating community-driven evaluation of anonymous annotations.
package marks

import (
	"crypto/ed25519"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Vote thresholds and constants.
const (
	// VoteMinResonance is the minimum Resonance to cast a vote on a mark.
	// Slightly lower than placing marks to encourage participation.
	VoteMinResonance = 50

	// VoteDuration is how long votes persist (matches mark duration).
	VoteDuration = MarkDuration

	// EndorsementBoost is the visibility multiplier from endorsements.
	// Each net endorsement adds to the mark's effective visibility.
	EndorsementBoost = 0.1

	// ChallengeThreshold is net challenges needed to hide a mark.
	// If challenges exceed endorsements by this margin, mark is hidden.
	ChallengeThreshold = 5
)

// MarkVoteType represents the type of vote on a mark.
// Named differently from council VoteType to avoid collision.
type MarkVoteType uint8

// Vote types for mark evaluation.
const (
	MarkVoteEndorse   MarkVoteType = iota + 1 // Endorse: support the mark
	MarkVoteChallenge                         // Challenge: dispute the mark
)

// Errors for voting operations.
var (
	ErrVoteInsufficientResonance = errors.New("insufficient resonance to vote")
	ErrVoteAlreadyCast           = errors.New("already voted on this mark")
	ErrVoteOnOwnMark             = errors.New("cannot vote on own mark")
	ErrVoteNotFound              = errors.New("vote not found")
	ErrInvalidMarkVoteType       = errors.New("invalid mark vote type")
)

// MarkVote represents a vote on a mark by a Specter.
type MarkVote struct {
	ID        [32]byte     // Unique vote ID (BLAKE3 hash).
	MarkID    [32]byte     // ID of the mark being voted on.
	VoterKey  [32]byte     // Voter's Curve25519 public key.
	VoteType  MarkVoteType // Endorse or Challenge.
	CreatedAt time.Time    // When the vote was cast.
	ExpiresAt time.Time    // When the vote expires.
	Signature []byte       // Ed25519 signature for verification.
}

// IsExpired returns true if the vote has expired.
func (v *MarkVote) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

// MarkVoteTypeString returns the human-readable name of a mark vote type.
func MarkVoteTypeString(vt MarkVoteType) string {
	switch vt {
	case MarkVoteEndorse:
		return "Endorse"
	case MarkVoteChallenge:
		return "Challenge"
	default:
		return "Unknown"
	}
}

// MarkVoteStore manages votes on marks.
type MarkVoteStore struct {
	mu         sync.RWMutex
	votes      map[[32]byte]*MarkVote // By vote ID.
	byMark     map[[32]byte][]*MarkVote
	byVoter    map[string][]*MarkVote       // By voter key (hex).
	voterMarks map[string]map[[32]byte]bool // voter -> markID -> voted.
	markStore  *MarkStore                   // Reference to mark store.
	markScores map[[32]byte]*MarkScore      // Aggregated scores.
}

// MarkScore tracks aggregated voting data for a mark.
type MarkScore struct {
	MarkID       [32]byte // Mark being scored.
	Endorsements int      // Count of endorsements.
	Challenges   int      // Count of challenges.
	NetScore     int      // Endorsements - Challenges.
	IsHidden     bool     // Hidden due to challenges.
	UpdatedAt    time.Time
}

// NewMarkVoteStore creates a new mark voting store.
func NewMarkVoteStore(markStore *MarkStore) *MarkVoteStore {
	return &MarkVoteStore{
		votes:      make(map[[32]byte]*MarkVote),
		byMark:     make(map[[32]byte][]*MarkVote),
		byVoter:    make(map[string][]*MarkVote),
		voterMarks: make(map[string]map[[32]byte]bool),
		markStore:  markStore,
		markScores: make(map[[32]byte]*MarkScore),
	}
}

// CanVote checks if a Specter can vote on a mark.
func (s *MarkVoteStore) CanVote(voterKey, markID [32]byte, resonance int) error {
	if resonance < VoteMinResonance {
		return ErrVoteInsufficientResonance
	}

	// Check mark exists.
	if s.markStore != nil {
		mark, err := s.markStore.GetMark(markID)
		if err != nil {
			return ErrMarkNotFound
		}
		// Cannot vote on own mark.
		if mark.MarkerKey == voterKey {
			return ErrVoteOnOwnMark
		}
	}

	// Check if already voted.
	voterHex := keyToHex(voterKey[:])
	s.mu.RLock()
	defer s.mu.RUnlock()

	if marks, ok := s.voterMarks[voterHex]; ok {
		if marks[markID] {
			return ErrVoteAlreadyCast
		}
	}

	return nil
}

// CastVote records a vote on a mark.
func (s *MarkVoteStore) CastVote(voterKey, markID [32]byte, voteType MarkVoteType, privKey ed25519.PrivateKey) (*MarkVote, error) {
	if voteType != MarkVoteEndorse && voteType != MarkVoteChallenge {
		return nil, ErrInvalidMarkVoteType
	}

	// Generate vote ID from voter + mark + type.
	h := blake3.New()
	h.Write(voterKey[:])
	h.Write(markID[:])
	h.Write([]byte{byte(voteType)})
	var id [32]byte
	copy(id[:], h.Sum(nil))

	now := time.Now()
	vote := &MarkVote{
		ID:        id,
		MarkID:    markID,
		VoterKey:  voterKey,
		VoteType:  voteType,
		CreatedAt: now,
		ExpiresAt: now.Add(VoteDuration),
	}

	// Sign the vote.
	if privKey != nil {
		signData := append(id[:], markID[:]...)
		signData = append(signData, byte(voteType))
		vote.Signature = ed25519.Sign(privKey, signData)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	voterHex := keyToHex(voterKey[:])

	// Initialize voter's mark map if needed.
	if s.voterMarks[voterHex] == nil {
		s.voterMarks[voterHex] = make(map[[32]byte]bool)
	}

	// Check double-vote under lock.
	if s.voterMarks[voterHex][markID] {
		return nil, ErrVoteAlreadyCast
	}

	// Store vote.
	s.votes[id] = vote
	s.byMark[markID] = append(s.byMark[markID], vote)
	s.byVoter[voterHex] = append(s.byVoter[voterHex], vote)
	s.voterMarks[voterHex][markID] = true

	// Update aggregated score.
	s.updateScoreLocked(markID, voteType, 1)

	return vote, nil
}

// updateScoreLocked updates the aggregated score for a mark.
// Must be called with lock held.
func (s *MarkVoteStore) updateScoreLocked(markID [32]byte, voteType MarkVoteType, delta int) {
	score, ok := s.markScores[markID]
	if !ok {
		score = &MarkScore{MarkID: markID}
		s.markScores[markID] = score
	}

	switch voteType {
	case MarkVoteEndorse:
		score.Endorsements += delta
	case MarkVoteChallenge:
		score.Challenges += delta
	}

	score.NetScore = score.Endorsements - score.Challenges
	score.IsHidden = score.NetScore < -ChallengeThreshold
	score.UpdatedAt = time.Now()
}

// GetVote retrieves a vote by ID.
func (s *MarkVoteStore) GetVote(id [32]byte) *MarkVote {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.votes[id]
}

// GetVotesOnMark returns all votes on a mark.
func (s *MarkVoteStore) GetVotesOnMark(markID [32]byte) []*MarkVote {
	s.mu.RLock()
	defer s.mu.RUnlock()

	votes := make([]*MarkVote, 0)
	for _, v := range s.byMark[markID] {
		if !v.IsExpired() {
			votes = append(votes, v)
		}
	}
	return votes
}

// GetVotesByVoter returns all votes cast by a Specter.
func (s *MarkVoteStore) GetVotesByVoter(voterKey [32]byte) []*MarkVote {
	s.mu.RLock()
	defer s.mu.RUnlock()

	voterHex := keyToHex(voterKey[:])
	votes := make([]*MarkVote, 0)
	for _, v := range s.byVoter[voterHex] {
		if !v.IsExpired() {
			votes = append(votes, v)
		}
	}
	return votes
}

// GetMarkScore returns the aggregated voting score for a mark.
func (s *MarkVoteStore) GetMarkScore(markID [32]byte) *MarkScore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.markScores[markID]
}

// IsMarkHidden returns true if a mark is hidden due to challenges.
func (s *MarkVoteStore) IsMarkHidden(markID [32]byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if score, ok := s.markScores[markID]; ok {
		return score.IsHidden
	}
	return false
}

// GetEffectiveVisibility returns a mark's visibility adjusted for votes.
// Base visibility is multiplied by endorsement boost and challenge penalty.
func (s *MarkVoteStore) GetEffectiveVisibility(markID [32]byte) float64 {
	if s.markStore == nil {
		return 0.0
	}

	mark, err := s.markStore.GetMark(markID)
	if err != nil || mark.IsExpired() {
		return 0.0
	}

	baseVisibility := mark.CurrentVisibility()

	s.mu.RLock()
	score := s.markScores[markID]
	s.mu.RUnlock()

	if score == nil {
		return baseVisibility
	}

	if score.IsHidden {
		return 0.0 // Hidden marks have no visibility.
	}

	// Apply endorsement boost.
	if score.NetScore > 0 {
		boost := float64(score.NetScore) * EndorsementBoost
		return math.Min(baseVisibility*(1.0+boost), 1.5) // Cap at 1.5x.
	}

	// Apply challenge penalty.
	if score.NetScore < 0 {
		penalty := float64(-score.NetScore) * EndorsementBoost
		return math.Max(baseVisibility*(1.0-penalty), 0.1) // Floor at 0.1.
	}

	return baseVisibility
}

// RemoveVote removes a vote (for cancellation).
func (s *MarkVoteStore) RemoveVote(id [32]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vote, ok := s.votes[id]
	if !ok {
		return ErrVoteNotFound
	}

	// Update score.
	s.updateScoreLocked(vote.MarkID, vote.VoteType, -1)

	// Remove from maps.
	voterHex := keyToHex(vote.VoterKey[:])
	delete(s.votes, id)
	delete(s.voterMarks[voterHex], vote.MarkID)

	// Remove from byMark slice.
	markVotes := s.byMark[vote.MarkID]
	for i, v := range markVotes {
		if v.ID == id {
			s.byMark[vote.MarkID] = append(markVotes[:i], markVotes[i+1:]...)
			break
		}
	}

	// Remove from byVoter slice.
	voterVotes := s.byVoter[voterHex]
	for i, v := range voterVotes {
		if v.ID == id {
			s.byVoter[voterHex] = append(voterVotes[:i], voterVotes[i+1:]...)
			break
		}
	}

	return nil
}

// PurgeExpiredVotes removes all expired votes.
func (s *MarkVoteStore) PurgeExpiredVotes() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired := make([][32]byte, 0)
	for id, vote := range s.votes {
		if vote.IsExpired() {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		vote := s.votes[id]
		voterHex := keyToHex(vote.VoterKey[:])

		// Update score.
		s.updateScoreLocked(vote.MarkID, vote.VoteType, -1)

		delete(s.votes, id)
		delete(s.voterMarks[voterHex], vote.MarkID)
	}

	return len(expired)
}

// CountVotesByType returns counts of each vote type on a mark.
func (s *MarkVoteStore) CountVotesByType(markID [32]byte) (endorsements, challenges int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.byMark[markID] {
		if !v.IsExpired() {
			switch v.VoteType {
			case MarkVoteEndorse:
				endorsements++
			case MarkVoteChallenge:
				challenges++
			}
		}
	}
	return endorsements, challenges
}

// HasVoted checks if a voter has voted on a mark.
func (s *MarkVoteStore) HasVoted(voterKey, markID [32]byte) bool {
	voterHex := keyToHex(voterKey[:])
	s.mu.RLock()
	defer s.mu.RUnlock()

	if marks, ok := s.voterMarks[voterHex]; ok {
		return marks[markID]
	}
	return false
}

// CountTotalVotes returns the total number of active votes.
func (s *MarkVoteStore) CountTotalVotes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, vote := range s.votes {
		if !vote.IsExpired() {
			count++
		}
	}
	return count
}

// GetHiddenMarks returns IDs of all marks hidden by community challenges.
func (s *MarkVoteStore) GetHiddenMarks() [][32]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hidden := make([][32]byte, 0)
	for markID, score := range s.markScores {
		if score.IsHidden {
			hidden = append(hidden, markID)
		}
	}
	return hidden
}
