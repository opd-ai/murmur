// Package resonance provides local reputation computation and rank thresholds.
// Per RESONANCE_SYSTEM.md, Resonance milestones unlock at 25/50/75/100/200/500.
package resonance

import (
	"math"
	"sync"
	"time"
)

// Milestones per RESONANCE_SYSTEM.md.
const (
	MilestoneShade       = 25
	MilestoneWraith      = 50
	MilestoneShadeWraith = 75
	MilestonePhantom     = 100
	MilestoneCouncil     = 200
	MilestoneAbyss       = 500
)

// Rank represents a Specter's achieved milestone.
type Rank int

const (
	RankNone Rank = iota
	RankShade
	RankWraith
	RankShadeWraith
	RankPhantom
	RankCouncil
	RankAbyss
)

// String returns the rank name.
func (r Rank) String() string {
	switch r {
	case RankShade:
		return "Shade"
	case RankWraith:
		return "Wraith"
	case RankShadeWraith:
		return "Shade-Wraith"
	case RankPhantom:
		return "Phantom"
	case RankCouncil:
		return "Council-Eligible"
	case RankAbyss:
		return "Abyss"
	default:
		return "None"
	}
}

// RankFromScore converts a Resonance score to a Rank.
func RankFromScore(score int) Rank {
	switch {
	case score >= MilestoneAbyss:
		return RankAbyss
	case score >= MilestoneCouncil:
		return RankCouncil
	case score >= MilestonePhantom:
		return RankPhantom
	case score >= MilestoneShadeWraith:
		return RankShadeWraith
	case score >= MilestoneWraith:
		return RankWraith
	case score >= MilestoneShade:
		return RankShade
	default:
		return RankNone
	}
}

// Signal categories per RESONANCE_SYSTEM.md.
type SignalWeights struct {
	PublicationConsistency float64 // Regular Specter activity
	MiniGameQuality        float64 // Puzzle solutions, duel outcomes
	GiftActivity           float64 // Given and received gifts
	CommunityEndorsement   float64 // Marks from high-Resonance Specters
}

// DefaultWeights returns the default signal weights.
func DefaultWeights() SignalWeights {
	return SignalWeights{
		PublicationConsistency: 0.30,
		MiniGameQuality:        0.25,
		GiftActivity:           0.20,
		CommunityEndorsement:   0.25,
	}
}

// Score represents a Specter's Resonance state.
type Score struct {
	mu sync.RWMutex

	// Raw signal values.
	Publications         int
	ConsecutiveDays      int
	PuzzlesSolved        int
	DuelsWon             int
	DuelsLost            int
	GiftsGiven           int
	GiftsReceived        int
	Endorsements         int
	HighTierEndorsements int // From Phantom+ Specters

	// Time tracking for decay.
	LastActivity time.Time
	CreatedAt    time.Time

	// Computed values.
	cachedScore int
	cacheValid  bool
	weights     SignalWeights
}

// NewScore creates a new Resonance score tracker.
func NewScore() *Score {
	now := time.Now()
	return &Score{
		LastActivity: now,
		CreatedAt:    now,
		weights:      DefaultWeights(),
	}
}

// NewScoreWithWeights creates a score with custom weights.
func NewScoreWithWeights(w SignalWeights) *Score {
	now := time.Now()
	return &Score{
		LastActivity: now,
		CreatedAt:    now,
		weights:      w,
	}
}

// AddPublication records a new Specter publication.
func (s *Score) AddPublication() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Publications++
	s.updateActivity()
}

// AddConsecutiveDay increments the consecutive day streak.
func (s *Score) AddConsecutiveDay() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ConsecutiveDays++
	s.updateActivity()
}

// ResetStreak resets the consecutive day streak.
func (s *Score) ResetStreak() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ConsecutiveDays = 0
	s.invalidateCache()
}

// AddPuzzleSolved records a puzzle solution.
func (s *Score) AddPuzzleSolved() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.PuzzlesSolved++
	s.updateActivity()
}

// AddDuelResult records a duel outcome.
func (s *Score) AddDuelResult(won bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if won {
		s.DuelsWon++
	} else {
		s.DuelsLost++
	}
	s.updateActivity()
}

// AddGiftGiven records giving a gift.
func (s *Score) AddGiftGiven() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.GiftsGiven++
	s.updateActivity()
}

// AddGiftReceived records receiving a gift.
func (s *Score) AddGiftReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.GiftsReceived++
	s.updateActivity()
}

// AddEndorsement records an endorsement from another Specter.
func (s *Score) AddEndorsement(fromHighTier bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Endorsements++
	if fromHighTier {
		s.HighTierEndorsements++
	}
	s.updateActivity()
}

// updateActivity updates last activity and invalidates cache.
func (s *Score) updateActivity() {
	s.LastActivity = time.Now()
	s.invalidateCache()
}

// invalidateCache marks the cached score as stale.
func (s *Score) invalidateCache() {
	s.cacheValid = false
}

// Compute calculates the current Resonance score.
// Per RESONANCE_SYSTEM.md, this is computed locally from activity signals.
func (s *Score) Compute() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cacheValid {
		return s.cachedScore
	}

	// Calculate each signal component.
	publicationScore := s.computePublicationScore()
	miniGameScore := s.computeMiniGameScore()
	giftScore := s.computeGiftScore()
	endorsementScore := s.computeEndorsementScore()

	// Weighted sum.
	rawScore := publicationScore*s.weights.PublicationConsistency +
		miniGameScore*s.weights.MiniGameQuality +
		giftScore*s.weights.GiftActivity +
		endorsementScore*s.weights.CommunityEndorsement

	// Apply time decay.
	decayFactor := s.computeDecay()
	finalScore := int(rawScore * decayFactor)

	// Clamp to non-negative.
	if finalScore < 0 {
		finalScore = 0
	}

	s.cachedScore = finalScore
	s.cacheValid = true

	return finalScore
}

// computePublicationScore calculates the publication consistency component.
func (s *Score) computePublicationScore() float64 {
	// Base score from publications with diminishing returns.
	pubScore := math.Log1p(float64(s.Publications)) * 10

	// Bonus for consecutive days (capped at 30 days).
	streakBonus := math.Min(float64(s.ConsecutiveDays), 30) * 2

	return pubScore + streakBonus
}

// computeMiniGameScore calculates the mini-game quality component.
func (s *Score) computeMiniGameScore() float64 {
	// Puzzle solutions are straightforward.
	puzzleScore := math.Log1p(float64(s.PuzzlesSolved)) * 15

	// Duel win rate matters.
	totalDuels := s.DuelsWon + s.DuelsLost
	var duelScore float64
	if totalDuels > 0 {
		winRate := float64(s.DuelsWon) / float64(totalDuels)
		duelScore = winRate * math.Log1p(float64(totalDuels)) * 20
	}

	return puzzleScore + duelScore
}

// computeGiftScore calculates the gift activity component.
func (s *Score) computeGiftScore() float64 {
	// Balance of giving and receiving.
	givenScore := math.Log1p(float64(s.GiftsGiven)) * 10
	receivedScore := math.Log1p(float64(s.GiftsReceived)) * 5

	// Bonus for generosity (giving more than receiving).
	generosityBonus := 0.0
	if s.GiftsGiven > s.GiftsReceived {
		generosityBonus = 10
	}

	return givenScore + receivedScore + generosityBonus
}

// computeEndorsementScore calculates the community endorsement component.
func (s *Score) computeEndorsementScore() float64 {
	// Base endorsement score.
	baseScore := math.Log1p(float64(s.Endorsements)) * 8

	// High-tier endorsements are worth more.
	highTierBonus := float64(s.HighTierEndorsements) * 5

	return baseScore + highTierBonus
}

// computeDecay applies time-based decay for inactive Specters.
// Per RESONANCE_SYSTEM.md, uses half-life curve.
func (s *Score) computeDecay() float64 {
	// Half-life of 7 days.
	halfLife := 7 * 24 * time.Hour
	daysSinceActivity := time.Since(s.LastActivity).Hours() / 24

	// No decay for first 3 days of inactivity.
	if daysSinceActivity <= 3 {
		return 1.0
	}

	// Exponential decay after grace period.
	effectiveDays := daysSinceActivity - 3
	decayFactor := math.Pow(0.5, effectiveDays/float64(halfLife.Hours()/24))

	// Minimum decay factor of 0.1 (never fully decay).
	return math.Max(decayFactor, 0.1)
}

// Rank returns the current rank based on computed score.
func (s *Score) Rank() Rank {
	return RankFromScore(s.Compute())
}

// EchoIndex returns the Echo Index (visibility metric).
// Per RESONANCE_SYSTEM.md, Echo Index determines Wave propagation reach.
func (s *Score) EchoIndex() float64 {
	score := s.Compute()

	// Logarithmic scaling for visibility.
	// Higher Resonance = wider propagation.
	if score == 0 {
		return 0.1
	}

	return math.Min(1.0, math.Log1p(float64(score))/6)
}

// Scorer manages Resonance scores for multiple Specters.
type Scorer struct {
	mu     sync.RWMutex
	scores map[string]*Score // specter_id -> Score
}

// NewScorer creates a new Resonance scorer.
func NewScorer() *Scorer {
	return &Scorer{
		scores: make(map[string]*Score),
	}
}

// GetScore retrieves or creates a Score for a Specter.
func (sc *Scorer) GetScore(specterID string) *Score {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if score, ok := sc.scores[specterID]; ok {
		return score
	}

	score := NewScore()
	sc.scores[specterID] = score
	return score
}

// SetScore sets a Score for a Specter.
func (sc *Scorer) SetScore(specterID string, score *Score) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.scores[specterID] = score
}

// RemoveScore removes a Specter's score.
func (sc *Scorer) RemoveScore(specterID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.scores, specterID)
}

// TopSpecters returns the top N Specters by Resonance score.
func (sc *Scorer) TopSpecters(n int) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	type specterScore struct {
		id    string
		score int
	}

	var all []specterScore
	for id, s := range sc.scores {
		all = append(all, specterScore{id: id, score: s.Compute()})
	}

	// Simple insertion sort for small n.
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(all); i++ {
		maxIdx := i
		for j := i + 1; j < len(all); j++ {
			if all[j].score > all[maxIdx].score {
				maxIdx = j
			}
		}
		all[i], all[maxIdx] = all[maxIdx], all[i]
		result = append(result, all[i].id)
	}

	return result
}

// Count returns the number of tracked Specters.
func (sc *Scorer) Count() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.scores)
}
