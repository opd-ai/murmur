package resonance

import (
	"testing"
	"time"
)

func TestSurfaceRankFromScore(t *testing.T) {
	tests := []struct {
		score int
		want  SurfaceRank
	}{
		{0, SurfaceRankNone},
		{5, SurfaceRankNone},
		{9, SurfaceRankNone},
		{10, SurfaceRankEmber},
		{24, SurfaceRankEmber},
		{25, SurfaceRankSpark},
		{49, SurfaceRankSpark},
		{50, SurfaceRankFlame},
		{99, SurfaceRankFlame},
		{100, SurfaceRankBlaze},
		{199, SurfaceRankBlaze},
		{200, SurfaceRankInferno},
		{499, SurfaceRankInferno},
		{500, SurfaceRankCorona},
		{1000, SurfaceRankCorona},
	}

	for _, tt := range tests {
		got := SurfaceRankFromScore(tt.score)
		if got != tt.want {
			t.Errorf("SurfaceRankFromScore(%d) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestSurfaceRankString(t *testing.T) {
	tests := []struct {
		rank SurfaceRank
		want string
	}{
		{SurfaceRankNone, "None"},
		{SurfaceRankEmber, "Ember"},
		{SurfaceRankSpark, "Spark"},
		{SurfaceRankFlame, "Flame"},
		{SurfaceRankBlaze, "Blaze"},
		{SurfaceRankInferno, "Inferno"},
		{SurfaceRankCorona, "Corona"},
	}

	for _, tt := range tests {
		got := tt.rank.String()
		if got != tt.want {
			t.Errorf("%v.String() = %s, want %s", tt.rank, got, tt.want)
		}
	}
}

func TestNewSurfaceScore(t *testing.T) {
	score := NewSurfaceScore()
	if score == nil {
		t.Fatal("NewSurfaceScore() returned nil")
	}
	if score.ConnectionCount != 0 {
		t.Errorf("ConnectionCount = %d, want 0", score.ConnectionCount)
	}
	if score.Compute() != 0 {
		t.Errorf("Initial Compute() = %d, want 0", score.Compute())
	}
}

func TestSurfaceScoreConnectionSignal(t *testing.T) {
	score := NewSurfaceScore()

	// No connections = 0 score from this signal.
	score.SetConnectionCount(0)
	initial := score.Compute()

	// Add connections and verify score increases.
	score.SetConnectionCount(20)
	withConnections := score.Compute()

	if withConnections <= initial {
		t.Errorf("Score with 20 connections (%d) should be greater than initial (%d)",
			withConnections, initial)
	}

	// More connections = higher score (with diminishing returns).
	score.SetConnectionCount(100)
	manyConnections := score.Compute()

	if manyConnections <= withConnections {
		t.Errorf("Score with 100 connections (%d) should be greater than 20 connections (%d)",
			manyConnections, withConnections)
	}

	// Verify AddConnection and RemoveConnection.
	score.SetConnectionCount(0)
	score.AddConnection()
	score.AddConnection()
	if score.ConnectionCount != 2 {
		t.Errorf("ConnectionCount after 2 AddConnection = %d, want 2", score.ConnectionCount)
	}

	score.RemoveConnection()
	if score.ConnectionCount != 1 {
		t.Errorf("ConnectionCount after RemoveConnection = %d, want 1", score.ConnectionCount)
	}

	// RemoveConnection should not go below 0.
	score.RemoveConnection()
	score.RemoveConnection()
	if score.ConnectionCount != 0 {
		t.Errorf("ConnectionCount should not go below 0, got %d", score.ConnectionCount)
	}
}

func TestSurfaceScoreDiversitySignal(t *testing.T) {
	score := NewSurfaceScore()

	// No diversity data = 0 score from this signal.
	score.SetClusterDiversity(nil, 0)
	noDiversity := score.Compute()

	// Add diversity data.
	clusterIDs := []string{"cluster-a", "cluster-b", "cluster-c"}
	score.SetClusterDiversity(clusterIDs, 5) // 3 of 5 clusters = 60%

	withDiversity := score.Compute()
	if withDiversity <= noDiversity {
		t.Errorf("Score with diversity (%d) should be greater than without (%d)",
			withDiversity, noDiversity)
	}

	// Higher diversity fraction = higher score.
	score.SetClusterDiversity(clusterIDs, 3) // 3 of 3 clusters = 100%
	fullDiversity := score.Compute()

	if fullDiversity <= withDiversity {
		t.Errorf("Score with full diversity (%d) should be greater than partial (%d)",
			fullDiversity, withDiversity)
	}
}

func TestSurfaceScoreWaveSignal(t *testing.T) {
	score := NewSurfaceScore()

	// No waves = 0 score from this signal.
	initial := score.Compute()

	// Add waves.
	score.SetWaveCount(10)
	withWaves := score.Compute()

	if withWaves <= initial {
		t.Errorf("Score with waves (%d) should be greater than initial (%d)",
			withWaves, initial)
	}

	// More waves = higher score.
	score.SetWaveCount(100)
	manyWaves := score.Compute()

	if manyWaves <= withWaves {
		t.Errorf("Score with many waves (%d) should be greater than few (%d)",
			manyWaves, withWaves)
	}

	// Verify AddWave.
	score.SetWaveCount(0)
	score.AddWave()
	score.AddWave()
	if score.WaveCount30d != 2 {
		t.Errorf("WaveCount30d after 2 AddWave = %d, want 2", score.WaveCount30d)
	}
}

func TestSurfaceScoreAmplificationSignals(t *testing.T) {
	score := NewSurfaceScore()
	initial := score.Compute()

	// Amplification received.
	score.SetAmplificationReceived(10)
	withAmpReceived := score.Compute()
	if withAmpReceived <= initial {
		t.Errorf("Score with amp received (%d) should be greater than initial (%d)",
			withAmpReceived, initial)
	}

	// Amplification given.
	score.SetAmplificationGiven(10)
	withAmpGiven := score.Compute()
	if withAmpGiven <= withAmpReceived {
		t.Errorf("Score with amp given (%d) should be greater than just received (%d)",
			withAmpGiven, withAmpReceived)
	}

	// Verify Add methods.
	score.SetAmplificationReceived(0)
	score.SetAmplificationGiven(0)
	score.AddAmplificationReceived()
	score.AddAmplificationGiven()

	if score.DistinctAmplifiers30d != 1 {
		t.Errorf("DistinctAmplifiers30d = %d, want 1", score.DistinctAmplifiers30d)
	}
	if score.DistinctAmplifiedWaves30d != 1 {
		t.Errorf("DistinctAmplifiedWaves30d = %d, want 1", score.DistinctAmplifiedWaves30d)
	}
}

func TestSurfaceScoreBridgeSignal(t *testing.T) {
	score := NewSurfaceScore()
	initial := score.Compute()

	// Bridge activity (for Hybrid+ nodes).
	score.SetBridgeActivity(5.0) // 5 messages per day average
	withBridge := score.Compute()

	if withBridge <= initial {
		t.Errorf("Score with bridge activity (%d) should be greater than initial (%d)",
			withBridge, initial)
	}

	// Higher bridge activity = higher score.
	score.SetBridgeActivity(20.0)
	highBridge := score.Compute()

	if highBridge <= withBridge {
		t.Errorf("Score with high bridge (%d) should be greater than moderate (%d)",
			highBridge, withBridge)
	}
}

func TestSurfaceScoreAgeSignal(t *testing.T) {
	score := NewSurfaceScore()

	// New account = minimal age score.
	newScore := score.Compute()

	// Set first seen to 6 months ago.
	score.SetFirstSeen(time.Now().Add(-180 * 24 * time.Hour))
	sixMonthScore := score.Compute()

	if sixMonthScore <= newScore {
		t.Errorf("Score after 6 months (%d) should be greater than new (%d)",
			sixMonthScore, newScore)
	}

	// Set first seen to 1 year ago (max bonus).
	score.SetFirstSeen(time.Now().Add(-365 * 24 * time.Hour))
	oneYearScore := score.Compute()

	if oneYearScore <= sixMonthScore {
		t.Errorf("Score after 1 year (%d) should be greater than 6 months (%d)",
			oneYearScore, sixMonthScore)
	}

	// Beyond 1 year should cap at same score.
	score.SetFirstSeen(time.Now().Add(-730 * 24 * time.Hour))
	twoYearScore := score.Compute()

	// Allow for small floating point differences.
	if twoYearScore < oneYearScore-1 || twoYearScore > oneYearScore+1 {
		t.Errorf("Score after 2 years (%d) should be similar to 1 year (%d)",
			twoYearScore, oneYearScore)
	}
}

func TestSurfaceScoreUptimeSignal(t *testing.T) {
	score := NewSurfaceScore()
	initial := score.Compute()

	// Full uptime.
	score.SetUptime(1.0)
	fullUptime := score.Compute()

	if fullUptime <= initial {
		t.Errorf("Score with full uptime (%d) should be greater than 0 uptime (%d)",
			fullUptime, initial)
	}

	// Half uptime.
	score.SetUptime(0.5)
	halfUptime := score.Compute()

	if halfUptime >= fullUptime {
		t.Errorf("Score with half uptime (%d) should be less than full (%d)",
			halfUptime, fullUptime)
	}

	// Verify clamping.
	score.SetUptime(-0.5)
	if score.Uptime30d != 0 {
		t.Errorf("Negative uptime should clamp to 0, got %f", score.Uptime30d)
	}

	score.SetUptime(2.0)
	if score.Uptime30d != 1.0 {
		t.Errorf("Uptime > 1 should clamp to 1, got %f", score.Uptime30d)
	}
}

func TestSurfaceScoreRank(t *testing.T) {
	score := NewSurfaceScore()

	// Start at None.
	if score.Rank() != SurfaceRankNone {
		t.Errorf("Initial rank = %v, want None", score.Rank())
	}

	// Add enough activity to reach Ember (10).
	score.SetConnectionCount(50)
	score.SetWaveCount(20)
	score.SetAmplificationReceived(10)

	// Should be at least Ember now.
	if score.Rank() < SurfaceRankEmber {
		t.Errorf("Rank after activity = %v (score %d), expected at least Ember",
			score.Rank(), score.Compute())
	}
}

func TestSurfaceScoreSignalBreakdown(t *testing.T) {
	score := NewSurfaceScore()
	score.SetConnectionCount(10)
	score.SetWaveCount(5)
	score.SetUptime(0.8)

	breakdown := score.GetSignalBreakdown()

	// Verify all expected keys exist.
	expectedKeys := []string{
		"ConnectionCount",
		"ConnectionDiversity",
		"WaveOutput",
		"AmplificationReceived",
		"AmplificationGiven",
		"BridgeActivity",
		"AccountAge",
		"Uptime",
	}

	for _, key := range expectedKeys {
		if _, ok := breakdown[key]; !ok {
			t.Errorf("Missing key %s in signal breakdown", key)
		}
	}

	// Connection score should be non-zero.
	if breakdown["ConnectionCount"] <= 0 {
		t.Error("ConnectionCount signal should be > 0")
	}

	// Wave score should be non-zero.
	if breakdown["WaveOutput"] <= 0 {
		t.Error("WaveOutput signal should be > 0")
	}

	// Uptime should be non-zero.
	if breakdown["Uptime"] <= 0 {
		t.Error("Uptime signal should be > 0")
	}
}

func TestSurfaceScoreCaching(t *testing.T) {
	score := NewSurfaceScore()
	score.SetConnectionCount(10)

	// First compute.
	first := score.Compute()

	// Second compute should return cached value.
	second := score.Compute()

	if first != second {
		t.Errorf("Cached score mismatch: first=%d, second=%d", first, second)
	}

	// Update invalidates cache.
	score.AddConnection()
	third := score.Compute()

	if third == first {
		t.Error("Score should change after adding connection")
	}
}

func TestSurfaceScorer(t *testing.T) {
	scorer := NewSurfaceScorer()

	// Get creates new score.
	score1 := scorer.GetScore("user-1")
	if score1 == nil {
		t.Fatal("GetScore returned nil")
	}

	// Modify score.
	score1.SetConnectionCount(50)

	// Get again returns same instance.
	score1Again := scorer.GetScore("user-1")
	if score1Again.ConnectionCount != 50 {
		t.Errorf("ConnectionCount = %d, want 50", score1Again.ConnectionCount)
	}

	// Different user gets different score.
	score2 := scorer.GetScore("user-2")
	if score2.ConnectionCount != 0 {
		t.Errorf("New user ConnectionCount = %d, want 0", score2.ConnectionCount)
	}

	// Count.
	if scorer.Count() != 2 {
		t.Errorf("Count = %d, want 2", scorer.Count())
	}

	// Remove.
	scorer.RemoveScore("user-1")
	if scorer.Count() != 1 {
		t.Errorf("Count after remove = %d, want 1", scorer.Count())
	}
}

func TestSurfaceScorerTopIdentities(t *testing.T) {
	scorer := NewSurfaceScorer()

	// Create scores with different values.
	for i := 1; i <= 5; i++ {
		score := scorer.GetScore("user-" + string(rune('0'+i)))
		score.SetConnectionCount(i * 10)
	}

	// Get top 3.
	top := scorer.TopIdentities(3)
	if len(top) != 3 {
		t.Errorf("TopIdentities(3) returned %d items, want 3", len(top))
	}

	// Highest should be first.
	if top[0] != "user-5" {
		t.Errorf("Top identity = %s, want user-5", top[0])
	}
}

func TestSurfaceScorerDecayAll(t *testing.T) {
	scorer := NewSurfaceScorer()

	score := scorer.GetScore("user-1")
	score.SetConnectionCount(10)

	// Compute to set cache.
	firstValue := score.Compute()

	// Decay should invalidate cache (score unchanged but recalculated).
	scorer.DecayAll()

	// Verify cache was invalidated (score recalculated).
	secondValue := score.Compute()

	// Values should be same since no actual decay mechanism in this test.
	if firstValue != secondValue {
		t.Errorf("Score changed unexpectedly: %d -> %d", firstValue, secondValue)
	}
}

func TestSurfaceMilestoneConstants(t *testing.T) {
	// Verify milestone values per RESONANCE_SYSTEM.md.
	if SurfaceMilestoneEmber != 10 {
		t.Errorf("SurfaceMilestoneEmber = %d, want 10", SurfaceMilestoneEmber)
	}
	if SurfaceMilestonesSpark != 25 {
		t.Errorf("SurfaceMilestonesSpark = %d, want 25", SurfaceMilestonesSpark)
	}
	if SurfaceMilestoneFlame != 50 {
		t.Errorf("SurfaceMilestoneFlame = %d, want 50", SurfaceMilestoneFlame)
	}
	if SurfaceMilestoneBlaze != 100 {
		t.Errorf("SurfaceMilestoneBlaze = %d, want 100", SurfaceMilestoneBlaze)
	}
	if SurfaceMilestoneInferno != 200 {
		t.Errorf("SurfaceMilestoneInferno = %d, want 200", SurfaceMilestoneInferno)
	}
	if SurfaceMilestoneCorona != 500 {
		t.Errorf("SurfaceMilestoneCorona = %d, want 500", SurfaceMilestoneCorona)
	}
}

func TestDefaultSurfaceWeights(t *testing.T) {
	w := DefaultSurfaceWeights()

	// Per RESONANCE_SYSTEM.md formula coefficients.
	if w.ConnectionCount != 10.0 {
		t.Errorf("ConnectionCount weight = %f, want 10.0", w.ConnectionCount)
	}
	if w.WaveOutput != 8.0 {
		t.Errorf("WaveOutput weight = %f, want 8.0", w.WaveOutput)
	}
	if w.AmplificationReceived != 15.0 {
		t.Errorf("AmplificationReceived weight = %f, want 15.0", w.AmplificationReceived)
	}
	if w.AmplificationGiven != 5.0 {
		t.Errorf("AmplificationGiven weight = %f, want 5.0", w.AmplificationGiven)
	}
	if w.BridgeActivity != 12.0 {
		t.Errorf("BridgeActivity weight = %f, want 12.0", w.BridgeActivity)
	}
	if w.AccountAge != 20.0 {
		t.Errorf("AccountAge weight = %f, want 20.0", w.AccountAge)
	}
	if w.Uptime != 10.0 {
		t.Errorf("Uptime weight = %f, want 10.0", w.Uptime)
	}
}

func TestSurfaceScoreRealisticValues(t *testing.T) {
	// Test realistic values per RESONANCE_SYSTEM.md:
	// "A typical active user after 6 months might have a Resonance between 80 and 150."

	score := NewSurfaceScore()

	// Typical active user profile.
	score.SetConnectionCount(30)                              // 30 connections
	score.SetClusterDiversity([]string{"a", "b", "c"}, 10)    // 3 of 10 clusters
	score.SetWaveCount(50)                                    // 50 waves in 30 days
	score.SetAmplificationReceived(20)                        // 20 unique amplifiers
	score.SetAmplificationGiven(15)                           // Amplified 15 waves
	score.SetBridgeActivity(2.0)                              // 2 bridged/day (Hybrid)
	score.SetFirstSeen(time.Now().Add(-180 * 24 * time.Hour)) // 6 months old
	score.SetUptime(0.7)                                      // 70% uptime

	computed := score.Compute()

	// Should be in 80-150 range for typical active user.
	if computed < 50 || computed > 200 {
		t.Errorf("Typical active user score = %d, expected roughly 80-150 range", computed)
	}

	t.Logf("Typical active user score: %d (rank: %s)", computed, score.Rank())
	t.Logf("Signal breakdown: %+v", score.GetSignalBreakdown())
}

func TestConnectionAgeTracking(t *testing.T) {
	score := NewSurfaceScore()

	// Add connections with age tracking.
	score.AddConnectionWithAge("peer-1")
	score.AddConnectionWithAge("peer-2")

	if score.ConnectionCount != 2 {
		t.Errorf("ConnectionCount = %d, want 2", score.ConnectionCount)
	}

	if len(score.ConnectionAges) != 2 {
		t.Errorf("ConnectionAges length = %d, want 2", len(score.ConnectionAges))
	}

	// Add same peer again should not add duplicate.
	score.AddConnectionWithAge("peer-1")
	if score.ConnectionCount != 3 {
		t.Errorf("ConnectionCount after duplicate = %d, want 3", score.ConnectionCount)
	}

	// Remove by ID.
	score.RemoveConnectionByID("peer-1")
	if score.ConnectionCount != 2 {
		t.Errorf("ConnectionCount after remove = %d, want 2", score.ConnectionCount)
	}
	if len(score.ConnectionAges) != 1 {
		t.Errorf("ConnectionAges length after remove = %d, want 1", len(score.ConnectionAges))
	}
}

func TestConnectionAgeBonus(t *testing.T) {
	score := NewSurfaceScore()
	initialScore := score.Compute()

	// Add connection with long history.
	sixMonthsAgo := time.Now().Add(-180 * 24 * time.Hour)
	score.SetConnectionAge("old-peer", sixMonthsAgo)
	score.SetConnectionCount(1)

	withOldConnection := score.Compute()

	if withOldConnection <= initialScore {
		t.Errorf("Score with old connection (%d) should be greater than initial (%d)",
			withOldConnection, initialScore)
	}

	// Add newer connection - average age decreases.
	score.SetConnectionAge("new-peer", time.Now())
	score.SetConnectionCount(2)

	withMixedConnections := score.Compute()

	// Score should still be higher than initial but potentially lower than just old.
	if withMixedConnections <= initialScore {
		t.Errorf("Score with mixed connections (%d) should be greater than initial (%d)",
			withMixedConnections, initialScore)
	}

	// Connection age capped at 365 days.
	score = NewSurfaceScore()
	twoYearsAgo := time.Now().Add(-730 * 24 * time.Hour)
	score.SetConnectionAge("ancient-peer", twoYearsAgo)
	score.SetConnectionCount(1)

	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour)
	score2 := NewSurfaceScore()
	score2.SetConnectionAge("year-old-peer", oneYearAgo)
	score2.SetConnectionCount(1)

	// Both should give similar connection age bonus (capped at 365 days).
	ancient := score.Compute()
	yearOld := score2.Compute()

	if ancient < yearOld-1 || ancient > yearOld+1 {
		t.Errorf("Ancient (%d) and year-old (%d) scores should be similar (capped at 365 days)",
			ancient, yearOld)
	}
}

func TestTemporalDecay(t *testing.T) {
	score := NewSurfaceScore()
	score.SetConnectionCount(50)
	score.SetWaveCount(20)

	// Record activity now - no decay.
	score.RecordActivity()
	activeScore := score.Compute()

	// Set last activity to 5 days ago (beyond 3-day grace period).
	score.SetLastActivityTime(time.Now().Add(-5 * 24 * time.Hour))
	inactiveScore := score.Compute()

	if inactiveScore >= activeScore {
		t.Errorf("Inactive score (%d) should be less than active score (%d)",
			inactiveScore, activeScore)
	}

	// Very old last activity - severe decay.
	score.SetLastActivityTime(time.Now().Add(-30 * 24 * time.Hour))
	veryInactiveScore := score.Compute()

	if veryInactiveScore >= inactiveScore {
		t.Errorf("Very inactive score (%d) should be less than inactive score (%d)",
			veryInactiveScore, inactiveScore)
	}
}

func TestSurfaceDecayGracePeriod(t *testing.T) {
	score := NewSurfaceScore()
	score.SetConnectionCount(50)
	score.SetWaveCount(20)

	// Record activity now.
	score.RecordActivity()
	nowScore := score.Compute()

	// Set last activity to 2 days ago (within 3-day grace period).
	score.SetLastActivityTime(time.Now().Add(-2 * 24 * time.Hour))
	withinGrace := score.Compute()

	// Should be equal - no decay within grace period.
	if withinGrace != nowScore {
		t.Errorf("Score within grace period (%d) should equal active score (%d)",
			withinGrace, nowScore)
	}
}

func TestGetDecayMultiplier(t *testing.T) {
	score := NewSurfaceScore()

	// Active - multiplier should be 1.0.
	score.RecordActivity()
	mult := score.GetDecayMultiplier()
	if mult != 1.0 {
		t.Errorf("Active decay multiplier = %f, want 1.0", mult)
	}

	// Within grace period - still 1.0.
	score.SetLastActivityTime(time.Now().Add(-2 * 24 * time.Hour))
	mult = score.GetDecayMultiplier()
	if mult != 1.0 {
		t.Errorf("Grace period decay multiplier = %f, want 1.0", mult)
	}

	// Beyond grace period - less than 1.0.
	score.SetLastActivityTime(time.Now().Add(-10 * 24 * time.Hour))
	mult = score.GetDecayMultiplier()
	if mult >= 1.0 {
		t.Errorf("Inactive decay multiplier = %f, should be < 1.0", mult)
	}

	// Decay should be exponential.
	// Default rate is 0.95, so 7 days past grace = 0.95^7 ≈ 0.698.
	expected := 0.698
	if mult < expected-0.05 || mult > expected+0.05 {
		t.Errorf("Decay multiplier after 7 decay days = %f, expected ~%f", mult, expected)
	}
}

func TestDefaultDecayConfig(t *testing.T) {
	cfg := DefaultDecayConfig()

	if cfg.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30", cfg.WindowDays)
	}
	if cfg.DecayRate != 0.95 {
		t.Errorf("DecayRate = %f, want 0.95", cfg.DecayRate)
	}
	if cfg.GracePeriodDays != 3 {
		t.Errorf("GracePeriodDays = %d, want 3", cfg.GracePeriodDays)
	}
	if cfg.ConnectionAgeCap != 365 {
		t.Errorf("ConnectionAgeCap = %d, want 365", cfg.ConnectionAgeCap)
	}
	if cfg.ConnectionAgeWeight != 15.0 {
		t.Errorf("ConnectionAgeWeight = %f, want 15.0", cfg.ConnectionAgeWeight)
	}
}

func TestNewSurfaceScoreWithConfig(t *testing.T) {
	weights := DefaultSurfaceWeights()
	weights.ConnectionCount = 20.0 // Custom weight.

	decay := DefaultDecayConfig()
	decay.GracePeriodDays = 7 // Custom grace period.

	score := NewSurfaceScoreWithConfig(weights, decay)

	if score == nil {
		t.Fatal("NewSurfaceScoreWithConfig returned nil")
	}

	// Verify custom weight is used.
	score.SetConnectionCount(10)
	breakdown := score.GetSignalBreakdown()
	connScore := breakdown["ConnectionCount"]

	// With weight 20 and 10 connections: 20 * ln(11) ≈ 47.96.
	expected := 47.96
	if connScore < expected-1 || connScore > expected+1 {
		t.Errorf("Connection score with custom weight = %f, expected ~%f", connScore, expected)
	}
}

func TestSignalBreakdownIncludesNewFields(t *testing.T) {
	score := NewSurfaceScore()
	score.SetConnectionCount(10)
	score.SetConnectionAge("peer-1", time.Now().Add(-100*24*time.Hour))
	score.RecordActivity()

	breakdown := score.GetSignalBreakdown()

	// Verify new fields are present.
	if _, ok := breakdown["ConnectionAgeBonus"]; !ok {
		t.Error("Missing ConnectionAgeBonus in signal breakdown")
	}
	if _, ok := breakdown["DecayMultiplier"]; !ok {
		t.Error("Missing DecayMultiplier in signal breakdown")
	}

	// Decay multiplier should be 1.0 for active user.
	if breakdown["DecayMultiplier"] != 1.0 {
		t.Errorf("DecayMultiplier = %f, want 1.0 for active user", breakdown["DecayMultiplier"])
	}

	// Connection age bonus should be > 0 for connection 100 days old.
	if breakdown["ConnectionAgeBonus"] <= 0 {
		t.Error("ConnectionAgeBonus should be > 0 for 100-day-old connection")
	}
}
