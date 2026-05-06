// Package resonance provides local reputation computation and rank thresholds.
// This file implements the full Specter Resonance computation per RESONANCE_SYSTEM.md.
// Specter Resonance is computed from 16 observable signals on the Anonymous Layer.
package resonance

import (
	"math"
	"sync"
	"time"
)

// SpecterSignalWeights defines the coefficients for each Specter input signal.
type SpecterSignalWeights struct {
	ConnectionCount     float64 // Coefficient: 10
	ConnectionDiversity float64 // Normalized cluster count, scaled
	WaveOutput          float64 // Coefficient: 8
	AmpReceived         float64 // Coefficient: 15
	AmpGiven            float64 // Coefficient: 5
	GiftVolume          float64 // Coefficient: 6
	EventParticipation  float64 // Coefficient: 4
	MiniGameActivity    float64 // Coefficient: 7
	TerritoryInfluence  float64 // Coefficient: 3
	CartographerScore   float64 // Coefficient: 6
	WhisperChain        float64 // Coefficient: 5
	ZKClaimCount        float64 // Coefficient: 3 (linear, not log)
	ShroudOperation     float64 // Coefficient: 25
	CouncilMembership   float64 // Coefficient: 10 (linear)
	SpecterAge          float64 // Max: 20
	SpecterUptime       float64 // Coefficient: 10
}

// DefaultSpecterWeights returns the standard coefficients per RESONANCE_SYSTEM.md.
func DefaultSpecterWeights() SpecterSignalWeights {
	return SpecterSignalWeights{
		ConnectionCount:     10.0,
		ConnectionDiversity: 1.0, // Applied to normalized diversity
		WaveOutput:          8.0,
		AmpReceived:         15.0,
		AmpGiven:            5.0,
		GiftVolume:          6.0,
		EventParticipation:  4.0,
		MiniGameActivity:    7.0,
		TerritoryInfluence:  3.0,
		CartographerScore:   6.0,
		WhisperChain:        5.0,
		ZKClaimCount:        3.0,
		ShroudOperation:     25.0,
		CouncilMembership:   10.0,
		SpecterAge:          20.0,
		SpecterUptime:       10.0,
	}
}

// Proximity Ignition constants.
// Per ROADMAP.md line 271: "Resonance bonus for Ignition (first 10 = 3 Resonance each)".
const (
	IgnitionMaxBonusCount = 10 // First 10 Ignitions earn bonus.
	IgnitionBonus         = 3  // Resonance bonus per Ignition.
)

// MiniGameActivity tracks all mini-game participation in the 30-day window.
type MiniGameActivity struct {
	PuzzleSolutions30d   int // Cipher Puzzle solutions
	HuntClaims30d        int // Specter Hunt fragment claims
	ForgeEntries30d      int // Sigil Forge entries
	OraclePredictions30d int // Oracle Pool predictions
	ShadowPlayRounds30d  int // Shadow Play rounds played
}

// Total returns the sum of all mini-game activities.
func (m MiniGameActivity) Total() int {
	return m.PuzzleSolutions30d + m.HuntClaims30d + m.ForgeEntries30d +
		m.OraclePredictions30d + m.ShadowPlayRounds30d
}

// TerritoryStatus tracks territory control and contested status.
type TerritoryStatus struct {
	Controlled int // Territories where Specter is Controller
	Contested  int // Territories where Specter is Contested
}

// SpecterScore represents a Specter's full Resonance state.
// Per RESONANCE_SYSTEM.md, this is computed from 16 observable signals.
type SpecterScore struct {
	mu sync.RWMutex

	// Connection signals (Anonymous Layer).
	SpecterConnectionCount int
	SpecterClusterIDs      []string // Cluster IDs of Specter connections
	TotalSpecterClusters   int      // Total visible clusters in anonymous topology

	// Wave activity (trailing 30-day window).
	SpecterWaveCount30d         int // Specter, Veiled, Sigil, Abyssal Waves
	DistinctSpecterAmpifiers30d int // Unique Specter amplifiers
	DistinctSpecterAmplified30d int // Unique Specter Waves amplified

	// Phantom Gift activity.
	GiftsSent30d int

	// Masked Event participation.
	EventsParticipated30d int

	// Mini-game activity (composite).
	MiniGames MiniGameActivity

	// Territory influence.
	Territories TerritoryStatus

	// Cartographer (90-day window).
	DistinctTerritoriesVisited90d int

	// Whisper Chain contributions.
	ChainContributions30d int

	// ZK Claims.
	ValidZKClaimCount int

	// Shroud Node operation (Fortress mode).
	ShroudUptime30d float64 // Fraction of time operating as Shroud Node

	// Council membership.
	ActiveCouncilCount int

	// Proximity Ignition (first-contact bonuses).
	// Per ROADMAP.md line 271: "first 10 = 3 Resonance each".
	IgnitionCount int // Total Ignition contacts completed.

	// Time-based signals.
	SpecterFirstSeen time.Time
	SpecterUptime30d float64 // Fraction of time online as Specter

	// Computed values.
	cachedScore int
	cacheValid  bool
	weights     SpecterSignalWeights
}

// NewSpecterScore creates a new Specter Resonance score tracker.
func NewSpecterScore() *SpecterScore {
	now := time.Now()
	return &SpecterScore{
		SpecterFirstSeen:  now,
		SpecterClusterIDs: make([]string, 0),
		weights:           DefaultSpecterWeights(),
	}
}

// NewSpecterScoreWithWeights creates a score with custom weights.
func NewSpecterScoreWithWeights(w SpecterSignalWeights) *SpecterScore {
	now := time.Now()
	return &SpecterScore{
		SpecterFirstSeen:  now,
		SpecterClusterIDs: make([]string, 0),
		weights:           w,
	}
}

// SetConnectionCount updates the Specter connection count.
func (s *SpecterScore) SetConnectionCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterConnectionCount = count
	s.invalidateCache()
}

// AddConnection increments the Specter connection count.
func (s *SpecterScore) AddConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterConnectionCount++
	s.invalidateCache()
}

// SetClusterDiversity updates the cluster diversity data.
func (s *SpecterScore) SetClusterDiversity(clusterIDs []string, totalClusters int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterClusterIDs = clusterIDs
	s.TotalSpecterClusters = totalClusters
	s.invalidateCache()
}

// SetWaveCount updates the 30-day anonymous wave count.
func (s *SpecterScore) SetWaveCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterWaveCount30d = count
	s.invalidateCache()
}

// AddWave increments the anonymous wave count.
func (s *SpecterScore) AddWave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterWaveCount30d++
	s.invalidateCache()
}

// SetAmpReceived updates the distinct Specter amplifier count.
func (s *SpecterScore) SetAmpReceived(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctSpecterAmpifiers30d = count
	s.invalidateCache()
}

// SetAmpGiven updates the distinct amplified Specter waves count.
func (s *SpecterScore) SetAmpGiven(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctSpecterAmplified30d = count
	s.invalidateCache()
}

// SetGiftsSent updates the Phantom Gift count for the 30-day window.
func (s *SpecterScore) SetGiftsSent(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GiftsSent30d = count
	s.invalidateCache()
}

// AddGiftSent increments the gift count.
func (s *SpecterScore) AddGiftSent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GiftsSent30d++
	s.invalidateCache()
}

// SetEventsParticipated updates the Masked Event participation count.
func (s *SpecterScore) SetEventsParticipated(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EventsParticipated30d = count
	s.invalidateCache()
}

// AddEventParticipation increments the event count.
func (s *SpecterScore) AddEventParticipation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EventsParticipated30d++
	s.invalidateCache()
}

// SetMiniGameActivity updates all mini-game participation metrics.
func (s *SpecterScore) SetMiniGameActivity(activity MiniGameActivity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames = activity
	s.invalidateCache()
}

// AddPuzzleSolution increments the puzzle solution count.
func (s *SpecterScore) AddPuzzleSolution() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames.PuzzleSolutions30d++
	s.invalidateCache()
}

// AddHuntClaim increments the Specter Hunt claim count.
func (s *SpecterScore) AddHuntClaim() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames.HuntClaims30d++
	s.invalidateCache()
}

// AddForgeEntry increments the Sigil Forge entry count.
func (s *SpecterScore) AddForgeEntry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames.ForgeEntries30d++
	s.invalidateCache()
}

// AddOraclePrediction increments the Oracle Pool prediction count.
func (s *SpecterScore) AddOraclePrediction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames.OraclePredictions30d++
	s.invalidateCache()
}

// AddShadowPlayRound increments the Shadow Play round count.
func (s *SpecterScore) AddShadowPlayRound() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MiniGames.ShadowPlayRounds30d++
	s.invalidateCache()
}

// SetTerritoryStatus updates the territory influence data.
func (s *SpecterScore) SetTerritoryStatus(controlled, contested int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Territories = TerritoryStatus{
		Controlled: controlled,
		Contested:  contested,
	}
	s.invalidateCache()
}

// SetCartographerVisits updates the distinct territories visited (90-day window).
func (s *SpecterScore) SetCartographerVisits(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctTerritoriesVisited90d = count
	s.invalidateCache()
}

// SetChainContributions updates the Whisper Chain contribution count.
func (s *SpecterScore) SetChainContributions(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ChainContributions30d = count
	s.invalidateCache()
}

// AddChainContribution increments the chain contribution count.
func (s *SpecterScore) AddChainContribution() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ChainContributions30d++
	s.invalidateCache()
}

// SetZKClaimCount updates the valid ZK Claim count.
func (s *SpecterScore) SetZKClaimCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ValidZKClaimCount = count
	s.invalidateCache()
}

// AddZKClaim increments the ZK Claim count.
func (s *SpecterScore) AddZKClaim() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ValidZKClaimCount++
	s.invalidateCache()
}

// SetShroudUptime updates the Shroud Node operation uptime fraction.
func (s *SpecterScore) SetShroudUptime(fraction float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ShroudUptime30d = clampFraction(fraction)
	s.invalidateCache()
}

// SetCouncilCount updates the active Phantom Council membership count.
func (s *SpecterScore) SetCouncilCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveCouncilCount = count
	s.invalidateCache()
}

// AddCouncilMembership increments the council count.
func (s *SpecterScore) AddCouncilMembership() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveCouncilCount++
	s.invalidateCache()
}

// AddIgnition records a Proximity Ignition (first-contact) event.
// Per ROADMAP.md line 271: "first 10 = 3 Resonance each".
// Returns the Resonance bonus earned (3 for first 10, 0 after that).
func (s *SpecterScore) AddIgnition() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	bonus := 0
	if s.IgnitionCount < IgnitionMaxBonusCount {
		bonus = IgnitionBonus
	}
	s.IgnitionCount++
	s.invalidateCache()
	return bonus
}

// GetIgnitionCount returns the total number of Ignition contacts.
func (s *SpecterScore) GetIgnitionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IgnitionCount
}

// SetSpecterFirstSeen sets when the Specter was first observed.
func (s *SpecterScore) SetSpecterFirstSeen(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterFirstSeen = t
	s.invalidateCache()
}

// SetSpecterUptime updates the Specter uptime fraction.
func (s *SpecterScore) SetSpecterUptime(fraction float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpecterUptime30d = clampFraction(fraction)
	s.invalidateCache()
}

// invalidateCache marks the cached score as stale.
func (s *SpecterScore) invalidateCache() {
	s.cacheValid = false
}

// Compute calculates the current Specter Resonance score.
// Per RESONANCE_SYSTEM.md, this sums all 16 weighted signal scores.
func (s *SpecterScore) Compute() int {
	return computeWithCache(&s.mu, &s.cacheValid, &s.cachedScore, s.computeRawScore)
}

// computeRawScore computes the raw score from all signal components.
func (s *SpecterScore) computeRawScore() int {
	// Calculate each signal component per spec formulas.
	connectionScore := s.computeConnectionScore()
	diversityScore := s.computeDiversityScore()
	waveScore := s.computeWaveScore()
	ampReceivedScore := s.computeAmpReceivedScore()
	ampGivenScore := s.computeAmpGivenScore()
	giftScore := s.computeGiftScore()
	eventScore := s.computeEventScore()
	miniGameScore := s.computeMiniGameScore()
	territoryScore := s.computeTerritoryScore()
	cartographerScore := s.computeCartographerScore()
	chainScore := s.computeChainScore()
	zkScore := s.computeZKScore()
	shroudScore := s.computeShroudScore()
	councilScore := s.computeCouncilScore()
	ageScore := s.computeAgeScore()
	uptimeScore := s.computeUptimeScore()
	ignitionScore := s.computeIgnitionScore()

	// Sum all components.
	rawScore := connectionScore + diversityScore + waveScore +
		ampReceivedScore + ampGivenScore + giftScore + eventScore +
		miniGameScore + territoryScore + cartographerScore +
		chainScore + zkScore + shroudScore + councilScore +
		ageScore + uptimeScore + ignitionScore

	// Round to nearest integer.
	finalScore := int(math.Round(rawScore))
	if finalScore < 0 {
		finalScore = 0
	}

	return finalScore
}

// computeConnectionScore: 10 * ln(1 + specter_connection_count)
func (s *SpecterScore) computeConnectionScore() float64 {
	return s.weights.ConnectionCount * math.Log1p(float64(s.SpecterConnectionCount))
}

// computeDiversityScore: cluster diversity among Specter connections.
func (s *SpecterScore) computeDiversityScore() float64 {
	if s.TotalSpecterClusters == 0 {
		return 0
	}

	uniqueClusters := make(map[string]struct{})
	for _, clusterID := range s.SpecterClusterIDs {
		uniqueClusters[clusterID] = struct{}{}
	}

	diversity := float64(len(uniqueClusters)) / float64(s.TotalSpecterClusters)
	return diversity * 20 * s.weights.ConnectionDiversity
}

// computeWaveScore: 8 * ln(1 + specter_wave_count_30d)
func (s *SpecterScore) computeWaveScore() float64 {
	return s.weights.WaveOutput * math.Log1p(float64(s.SpecterWaveCount30d))
}

// computeAmpReceivedScore: 15 * ln(1 + distinct_specter_amplifier_count_30d)
func (s *SpecterScore) computeAmpReceivedScore() float64 {
	return s.weights.AmpReceived * math.Log1p(float64(s.DistinctSpecterAmpifiers30d))
}

// computeAmpGivenScore: 5 * ln(1 + distinct_specter_amplified_waves_30d)
func (s *SpecterScore) computeAmpGivenScore() float64 {
	return s.weights.AmpGiven * math.Log1p(float64(s.DistinctSpecterAmplified30d))
}

// computeGiftScore: 6 * ln(1 + gifts_sent_30d)
func (s *SpecterScore) computeGiftScore() float64 {
	return s.weights.GiftVolume * math.Log1p(float64(s.GiftsSent30d))
}

// computeEventScore: 4 * ln(1 + events_participated_30d)
func (s *SpecterScore) computeEventScore() float64 {
	return s.weights.EventParticipation * math.Log1p(float64(s.EventsParticipated30d))
}

// computeMiniGameScore: 7 * ln(1 + total_minigame_activity)
func (s *SpecterScore) computeMiniGameScore() float64 {
	return s.weights.MiniGameActivity * math.Log1p(float64(s.MiniGames.Total()))
}

// computeTerritoryScore: 3 * ln(1 + controlled + 0.5 * contested)
func (s *SpecterScore) computeTerritoryScore() float64 {
	influence := float64(s.Territories.Controlled) + 0.5*float64(s.Territories.Contested)
	return s.weights.TerritoryInfluence * math.Log1p(influence)
}

// computeCartographerScore: 6 * ln(1 + distinct_territories_visited_90d)
func (s *SpecterScore) computeCartographerScore() float64 {
	return s.weights.CartographerScore * math.Log1p(float64(s.DistinctTerritoriesVisited90d))
}

// computeChainScore: 5 * ln(1 + chain_contributions_30d)
func (s *SpecterScore) computeChainScore() float64 {
	return s.weights.WhisperChain * math.Log1p(float64(s.ChainContributions30d))
}

// computeZKScore: 3 * zk_claim_count (linear, not logarithmic)
func (s *SpecterScore) computeZKScore() float64 {
	return s.weights.ZKClaimCount * float64(s.ValidZKClaimCount)
}

// computeShroudScore: uptime_fraction_30d * 25
func (s *SpecterScore) computeShroudScore() float64 {
	return s.ShroudUptime30d * s.weights.ShroudOperation
}

// computeCouncilScore: 10 * active_council_count (linear)
func (s *SpecterScore) computeCouncilScore() float64 {
	return s.weights.CouncilMembership * float64(s.ActiveCouncilCount)
}

// computeAgeScore: min(days_since_first_seen / 365, 1.0) * 20
func (s *SpecterScore) computeAgeScore() float64 {
	daysSinceFirstSeen := time.Since(s.SpecterFirstSeen).Hours() / 24
	ageFraction := math.Min(daysSinceFirstSeen/365, 1.0)
	return ageFraction * s.weights.SpecterAge
}

// computeUptimeScore: specter_uptime_fraction_30d * 10
func (s *SpecterScore) computeUptimeScore() float64 {
	return s.SpecterUptime30d * s.weights.SpecterUptime
}

// computeIgnitionScore: 3 Resonance for each of first 10 Ignitions.
// Per ROADMAP.md line 271: "first 10 = 3 Resonance each".
func (s *SpecterScore) computeIgnitionScore() float64 {
	count := s.IgnitionCount
	if count > IgnitionMaxBonusCount {
		count = IgnitionMaxBonusCount
	}
	return float64(count * IgnitionBonus)
}

// Rank returns the current Specter rank based on computed score.
func (s *SpecterScore) Rank() Rank {
	return RankFromScore(s.Compute())
}

// GetSignalBreakdown returns individual signal scores for debugging/display.
func (s *SpecterScore) GetSignalBreakdown() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]float64{
		"ConnectionCount":     s.computeConnectionScore(),
		"ConnectionDiversity": s.computeDiversityScore(),
		"WaveOutput":          s.computeWaveScore(),
		"AmpReceived":         s.computeAmpReceivedScore(),
		"AmpGiven":            s.computeAmpGivenScore(),
		"GiftVolume":          s.computeGiftScore(),
		"EventParticipation":  s.computeEventScore(),
		"MiniGameActivity":    s.computeMiniGameScore(),
		"TerritoryInfluence":  s.computeTerritoryScore(),
		"CartographerScore":   s.computeCartographerScore(),
		"WhisperChain":        s.computeChainScore(),
		"ZKClaimCount":        s.computeZKScore(),
		"ShroudOperation":     s.computeShroudScore(),
		"CouncilMembership":   s.computeCouncilScore(),
		"SpecterAge":          s.computeAgeScore(),
		"SpecterUptime":       s.computeUptimeScore(),
		"Ignition":            s.computeIgnitionScore(),
	}
}

// SpecterScorer manages Specter Resonance scores for multiple Specters.
type SpecterScorer struct {
	*GenericScorer[*SpecterScore]
}

// NewSpecterScorer creates a new Specter Resonance scorer.
func NewSpecterScorer() *SpecterScorer {
	return &SpecterScorer{
		GenericScorer: NewGenericScorer(NewSpecterScore),
	}
}

// RemoveScore removes a Specter's score.
func (sc *SpecterScorer) RemoveScore(specterID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.scores, specterID)
}

// TopSpecters returns the top N Specters by Resonance score.
func (sc *SpecterScorer) TopSpecters(n int) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	type specterRank struct {
		id    string
		score int
	}

	var all []specterRank
	for id, s := range sc.scores {
		all = append(all, specterRank{id: id, score: s.Compute()})
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
func (sc *SpecterScorer) Count() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.scores)
}
