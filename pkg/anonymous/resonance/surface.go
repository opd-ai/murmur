// Package resonance provides local reputation computation and rank thresholds.
// This file implements Surface Layer Resonance computation.
// Per RESONANCE_SYSTEM.md, Surface Resonance is computed from 8 observable signals.
package resonance

import (
	"math"
	"sync"
	"time"
)

// Surface Layer milestones per RESONANCE_SYSTEM.md.
const (
	SurfaceMilestoneEmber   = 10
	SurfaceMilestonesSpark  = 25
	SurfaceMilestoneFlame   = 50
	SurfaceMilestoneBlaze   = 100
	SurfaceMilestoneInferno = 200
	SurfaceMilestoneCorona  = 500
)

// SurfaceRank represents a Surface Layer milestone achievement.
type SurfaceRank int

const (
	SurfaceRankNone SurfaceRank = iota
	SurfaceRankEmber
	SurfaceRankSpark
	SurfaceRankFlame
	SurfaceRankBlaze
	SurfaceRankInferno
	SurfaceRankCorona
)

// String returns the rank name.
func (r SurfaceRank) String() string {
	switch r {
	case SurfaceRankEmber:
		return "Ember"
	case SurfaceRankSpark:
		return "Spark"
	case SurfaceRankFlame:
		return "Flame"
	case SurfaceRankBlaze:
		return "Blaze"
	case SurfaceRankInferno:
		return "Inferno"
	case SurfaceRankCorona:
		return "Corona"
	default:
		return "None"
	}
}

// SurfaceRankFromScore converts a Resonance score to a SurfaceRank.
func SurfaceRankFromScore(score int) SurfaceRank {
	switch {
	case score >= SurfaceMilestoneCorona:
		return SurfaceRankCorona
	case score >= SurfaceMilestoneInferno:
		return SurfaceRankInferno
	case score >= SurfaceMilestoneBlaze:
		return SurfaceRankBlaze
	case score >= SurfaceMilestoneFlame:
		return SurfaceRankFlame
	case score >= SurfaceMilestonesSpark:
		return SurfaceRankSpark
	case score >= SurfaceMilestoneEmber:
		return SurfaceRankEmber
	default:
		return SurfaceRankNone
	}
}

// SurfaceSignalWeights defines the relative weight of each input signal.
// These are implicit in the formula coefficients.
type SurfaceSignalWeights struct {
	ConnectionCount       float64 // Coefficient: 10
	ConnectionDiversity   float64 // Normalized cluster count
	WaveOutput            float64 // Coefficient: 8
	AmplificationReceived float64 // Coefficient: 15
	AmplificationGiven    float64 // Coefficient: 5
	BridgeActivity        float64 // Coefficient: 12
	AccountAge            float64 // Max: 20
	Uptime                float64 // Coefficient: 10
}

// DefaultSurfaceWeights returns the default coefficients per spec.
func DefaultSurfaceWeights() SurfaceSignalWeights {
	return SurfaceSignalWeights{
		ConnectionCount:       10.0,
		ConnectionDiversity:   1.0, // Applied to normalized diversity
		WaveOutput:            8.0,
		AmplificationReceived: 15.0,
		AmplificationGiven:    5.0,
		BridgeActivity:        12.0,
		AccountAge:            20.0,
		Uptime:                10.0,
	}
}

// DecayConfig controls temporal decay behavior for signals.
type DecayConfig struct {
	// WindowDays is the rolling window for activity-based signals (default: 30).
	WindowDays int
	// DecayRate is the daily decay multiplier for inactive days (0.95 = 5%/day).
	DecayRate float64
	// GracePeriodDays is the number of inactive days before decay begins.
	GracePeriodDays int
	// ConnectionAgeCap is the max days for connection age bonus (default: 365).
	ConnectionAgeCap int
	// ConnectionAgeWeight is the max bonus for old connections (default: 15).
	ConnectionAgeWeight float64
}

// DefaultDecayConfig returns the standard decay configuration.
func DefaultDecayConfig() DecayConfig {
	return DecayConfig{
		WindowDays:          30,
		DecayRate:           0.95,
		GracePeriodDays:     3,
		ConnectionAgeCap:    365,
		ConnectionAgeWeight: 15.0,
	}
}

// SurfaceScore represents a Surface Layer identity's Resonance state.
// Per RESONANCE_SYSTEM.md, this is computed from 8 observable signals
// plus connection age bonus and temporal decay over the 30-day window.
type SurfaceScore struct {
	mu sync.RWMutex

	// Connection signals.
	ConnectionCount   int
	ClusterIDs        []string // Cluster IDs of connected nodes
	TotalClusterCount int      // Total visible clusters in topology

	// Connection age tracking - maps peer ID to connection start time.
	ConnectionAges map[string]time.Time

	// Wave activity (trailing 30-day window).
	WaveCount30d              int
	DistinctAmplifiers30d     int // Unique amplifiers of this node's Waves
	DistinctAmplifiedWaves30d int // Unique Waves this node amplified

	// Bridge activity (Hybrid+ nodes).
	AvgBridgedPerDay30d float64

	// Time-based signals.
	FirstSeen time.Time
	Uptime30d float64 // Fraction of time online in 30 days

	// Temporal decay tracking.
	LastActivityTime time.Time // Time of most recent activity

	// Computed values.
	cachedScore int
	cacheValid  bool
	weights     SurfaceSignalWeights
	decayConfig DecayConfig
}

// NewSurfaceScore creates a new Surface Resonance score tracker.
func NewSurfaceScore() *SurfaceScore {
	now := time.Now()
	return &SurfaceScore{
		FirstSeen:        now,
		LastActivityTime: now,
		ClusterIDs:       make([]string, 0),
		ConnectionAges:   make(map[string]time.Time),
		weights:          DefaultSurfaceWeights(),
		decayConfig:      DefaultDecayConfig(),
	}
}

// NewSurfaceScoreWithWeights creates a score with custom weights.
func NewSurfaceScoreWithWeights(w SurfaceSignalWeights) *SurfaceScore {
	now := time.Now()
	return &SurfaceScore{
		FirstSeen:        now,
		LastActivityTime: now,
		ClusterIDs:       make([]string, 0),
		ConnectionAges:   make(map[string]time.Time),
		weights:          w,
		decayConfig:      DefaultDecayConfig(),
	}
}

// NewSurfaceScoreWithConfig creates a score with custom weights and decay config.
func NewSurfaceScoreWithConfig(w SurfaceSignalWeights, d DecayConfig) *SurfaceScore {
	now := time.Now()
	return &SurfaceScore{
		FirstSeen:        now,
		LastActivityTime: now,
		ClusterIDs:       make([]string, 0),
		ConnectionAges:   make(map[string]time.Time),
		weights:          w,
		decayConfig:      d,
	}
}

// SetConnectionCount updates the connection count.
func (s *SurfaceScore) SetConnectionCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnectionCount = count
	s.invalidateCache()
}

// AddConnection increments the connection count.
func (s *SurfaceScore) AddConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnectionCount++
	s.invalidateCache()
}

// AddConnectionWithAge adds a connection and tracks when it started.
func (s *SurfaceScore) AddConnectionWithAge(peerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnectionCount++
	if s.ConnectionAges == nil {
		s.ConnectionAges = make(map[string]time.Time)
	}
	if _, exists := s.ConnectionAges[peerID]; !exists {
		s.ConnectionAges[peerID] = time.Now()
	}
	s.invalidateCache()
}

// SetConnectionAge sets the age of a specific connection.
func (s *SurfaceScore) SetConnectionAge(peerID string, startTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConnectionAges == nil {
		s.ConnectionAges = make(map[string]time.Time)
	}
	s.ConnectionAges[peerID] = startTime
	s.invalidateCache()
}

// RemoveConnection decrements the connection count.
func (s *SurfaceScore) RemoveConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConnectionCount > 0 {
		s.ConnectionCount--
	}
	s.invalidateCache()
}

// RemoveConnectionByID removes a connection and its age tracking.
func (s *SurfaceScore) RemoveConnectionByID(peerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ConnectionCount > 0 {
		s.ConnectionCount--
	}
	delete(s.ConnectionAges, peerID)
	s.invalidateCache()
}

// SetClusterDiversity updates the cluster diversity data.
func (s *SurfaceScore) SetClusterDiversity(clusterIDs []string, totalClusters int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ClusterIDs = clusterIDs
	s.TotalClusterCount = totalClusters
	s.invalidateCache()
}

// SetWaveCount updates the 30-day wave count.
func (s *SurfaceScore) SetWaveCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WaveCount30d = count
	s.invalidateCache()
}

// AddWave increments the wave count.
func (s *SurfaceScore) AddWave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WaveCount30d++
	s.invalidateCache()
}

// SetAmplificationReceived updates the amplification received count.
func (s *SurfaceScore) SetAmplificationReceived(distinctAmplifiers int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctAmplifiers30d = distinctAmplifiers
	s.invalidateCache()
}

// AddAmplificationReceived increments the amplification received.
func (s *SurfaceScore) AddAmplificationReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctAmplifiers30d++
	s.invalidateCache()
}

// SetAmplificationGiven updates the amplification given count.
func (s *SurfaceScore) SetAmplificationGiven(distinctWaves int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctAmplifiedWaves30d = distinctWaves
	s.invalidateCache()
}

// AddAmplificationGiven increments the amplification given.
func (s *SurfaceScore) AddAmplificationGiven() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistinctAmplifiedWaves30d++
	s.invalidateCache()
}

// SetBridgeActivity updates the bridge activity metric.
func (s *SurfaceScore) SetBridgeActivity(avgPerDay float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AvgBridgedPerDay30d = avgPerDay
	s.invalidateCache()
}

// SetFirstSeen sets when the node was first observed.
func (s *SurfaceScore) SetFirstSeen(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FirstSeen = t
	s.invalidateCache()
}

// SetUptime updates the uptime fraction.
func (s *SurfaceScore) SetUptime(fraction float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Uptime30d = clampFraction(fraction)
	s.invalidateCache()
}

// RecordActivity marks the current time as the last activity time.
// This is used for temporal decay calculation.
func (s *SurfaceScore) RecordActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityTime = time.Now()
	s.invalidateCache()
}

// SetLastActivityTime sets the last activity time directly.
func (s *SurfaceScore) SetLastActivityTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityTime = t
	s.invalidateCache()
}

// GetDecayMultiplier returns the current decay multiplier based on inactivity.
// Returns 1.0 if within grace period, decays exponentially after.
func (s *SurfaceScore) GetDecayMultiplier() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.computeDecayMultiplier()
}

// computeDecayMultiplier calculates decay based on days since last activity.
func (s *SurfaceScore) computeDecayMultiplier() float64 {
	if s.LastActivityTime.IsZero() {
		return 1.0
	}

	daysSinceActivity := time.Since(s.LastActivityTime).Hours() / 24

	// Within grace period = no decay.
	if daysSinceActivity <= float64(s.decayConfig.GracePeriodDays) {
		return 1.0
	}

	// Exponential decay after grace period.
	decayDays := daysSinceActivity - float64(s.decayConfig.GracePeriodDays)
	return math.Pow(s.decayConfig.DecayRate, decayDays)
}

// invalidateCache marks the cached score as stale.
func (s *SurfaceScore) invalidateCache() {
	s.cacheValid = false
}

// Compute calculates the current Surface Resonance score.
// Per RESONANCE_SYSTEM.md, this sums all 8 weighted signal scores
// plus connection age bonus, then applies temporal decay.
func (s *SurfaceScore) Compute() int {
	return computeWithCache(&s.mu, &s.cacheValid, &s.cachedScore, s.computeRawScore)
}

// computeRawScore computes the raw score from all signal components.
func (s *SurfaceScore) computeRawScore() int {
	// Calculate each signal component per spec formulas.
	connectionScore := s.computeConnectionScore()
	diversityScore := s.computeDiversityScore()
	waveScore := s.computeWaveScore()
	ampReceivedScore := s.computeAmpReceivedScore()
	ampGivenScore := s.computeAmpGivenScore()
	bridgeScore := s.computeBridgeScore()
	ageScore := s.computeAgeScore()
	uptimeScore := s.computeUptimeScore()
	connectionAgeBonus := s.computeConnectionAgeBonus()

	// Sum all components.
	rawScore := connectionScore + diversityScore + waveScore +
		ampReceivedScore + ampGivenScore + bridgeScore +
		ageScore + uptimeScore + connectionAgeBonus

	// Apply temporal decay for inactivity.
	decayMultiplier := s.computeDecayMultiplier()
	rawScore *= decayMultiplier

	// Round to nearest integer.
	finalScore := int(math.Round(rawScore))
	if finalScore < 0 {
		finalScore = 0
	}

	return finalScore
}

// computeConnectionAgeBonus calculates bonus points for long-standing connections.
// Per spec: reward sustained relationships, not just new connections.
func (s *SurfaceScore) computeConnectionAgeBonus() float64 {
	if len(s.ConnectionAges) == 0 {
		return 0
	}

	now := time.Now()
	var totalAgeBonus float64

	for _, startTime := range s.ConnectionAges {
		daysConnected := now.Sub(startTime).Hours() / 24
		// Cap at configured maximum (default 365 days).
		if daysConnected > float64(s.decayConfig.ConnectionAgeCap) {
			daysConnected = float64(s.decayConfig.ConnectionAgeCap)
		}
		// Normalized age fraction.
		ageFraction := daysConnected / float64(s.decayConfig.ConnectionAgeCap)
		totalAgeBonus += ageFraction
	}

	// Average age bonus across all connections, scaled by weight.
	avgAgeFraction := totalAgeBonus / float64(len(s.ConnectionAges))
	return avgAgeFraction * s.decayConfig.ConnectionAgeWeight
}

// computeConnectionScore implements: 10 * ln(1 + connection_count)
func (s *SurfaceScore) computeConnectionScore() float64 {
	return s.weights.ConnectionCount * math.Log1p(float64(s.ConnectionCount))
}

// computeDiversityScore computes cluster diversity.
// Per spec: count of distinct clusters among connections / total visible clusters.
func (s *SurfaceScore) computeDiversityScore() float64 {
	if s.TotalClusterCount == 0 {
		return 0
	}

	// Count unique clusters in connections.
	uniqueClusters := make(map[string]struct{})
	for _, clusterID := range s.ClusterIDs {
		uniqueClusters[clusterID] = struct{}{}
	}

	// Normalized diversity: fraction of clusters represented.
	diversity := float64(len(uniqueClusters)) / float64(s.TotalClusterCount)

	// Scale to reasonable range (0-20).
	return diversity * 20 * s.weights.ConnectionDiversity
}

// computeWaveScore implements: 8 * ln(1 + wave_count_30d)
func (s *SurfaceScore) computeWaveScore() float64 {
	return s.weights.WaveOutput * math.Log1p(float64(s.WaveCount30d))
}

// computeAmpReceivedScore implements: 15 * ln(1 + distinct_amplifier_count_30d)
func (s *SurfaceScore) computeAmpReceivedScore() float64 {
	return s.weights.AmplificationReceived * math.Log1p(float64(s.DistinctAmplifiers30d))
}

// computeAmpGivenScore implements: 5 * ln(1 + distinct_amplified_waves_30d)
func (s *SurfaceScore) computeAmpGivenScore() float64 {
	return s.weights.AmplificationGiven * math.Log1p(float64(s.DistinctAmplifiedWaves30d))
}

// computeBridgeScore implements: 12 * ln(1 + avg_bridged_per_day_30d)
func (s *SurfaceScore) computeBridgeScore() float64 {
	return s.weights.BridgeActivity * math.Log1p(s.AvgBridgedPerDay30d)
}

// computeAgeScore implements: min(days_since_first_seen / 365, 1.0) * 20
func (s *SurfaceScore) computeAgeScore() float64 {
	daysSinceFirstSeen := time.Since(s.FirstSeen).Hours() / 24
	ageFraction := math.Min(daysSinceFirstSeen/365, 1.0)
	return ageFraction * s.weights.AccountAge
}

// computeUptimeScore implements: uptime_fraction_30d * 10
func (s *SurfaceScore) computeUptimeScore() float64 {
	return s.Uptime30d * s.weights.Uptime
}

// Rank returns the current Surface rank based on computed score.
func (s *SurfaceScore) Rank() SurfaceRank {
	return SurfaceRankFromScore(s.Compute())
}

// GetSignalBreakdown returns individual signal scores for debugging/display.
func (s *SurfaceScore) GetSignalBreakdown() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]float64{
		"ConnectionCount":       s.computeConnectionScore(),
		"ConnectionDiversity":   s.computeDiversityScore(),
		"WaveOutput":            s.computeWaveScore(),
		"AmplificationReceived": s.computeAmpReceivedScore(),
		"AmplificationGiven":    s.computeAmpGivenScore(),
		"BridgeActivity":        s.computeBridgeScore(),
		"AccountAge":            s.computeAgeScore(),
		"Uptime":                s.computeUptimeScore(),
		"ConnectionAgeBonus":    s.computeConnectionAgeBonus(),
		"DecayMultiplier":       s.computeDecayMultiplier(),
	}
}

// SurfaceScorer manages Surface Resonance scores for multiple identities.
type SurfaceScorer struct {
	*GenericScorer[*SurfaceScore]
}

// NewSurfaceScorer creates a new Surface Resonance scorer.
func NewSurfaceScorer() *SurfaceScorer {
	return &SurfaceScorer{
		GenericScorer: NewGenericScorer(NewSurfaceScore),
	}
}

// RemoveScore removes an identity's score.
func (sc *SurfaceScorer) RemoveScore(identityID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.scores, identityID)
}

// TopIdentities returns the top N identities by Surface Resonance score.
func (sc *SurfaceScorer) TopIdentities(n int) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	type identityScore struct {
		id    string
		score int
	}

	var all []identityScore
	for id, s := range sc.scores {
		all = append(all, identityScore{id: id, score: s.Compute()})
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

// Count returns the number of tracked identities.
func (sc *SurfaceScorer) Count() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.scores)
}

// DecayAll applies temporal decay to all tracked scores.
// This should be called periodically (e.g., every 60 seconds).
func (sc *SurfaceScorer) DecayAll() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, score := range sc.scores {
		score.invalidateCache() // Force recomputation on next access
	}
}
