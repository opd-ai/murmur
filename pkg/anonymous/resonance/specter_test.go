package resonance

import (
	"testing"
	"time"
)

func TestNewSpecterScore(t *testing.T) {
	score := NewSpecterScore()
	if score == nil {
		t.Fatal("NewSpecterScore() returned nil")
	}
	if score.SpecterConnectionCount != 0 {
		t.Errorf("SpecterConnectionCount = %d, want 0", score.SpecterConnectionCount)
	}
	if score.Compute() != 0 {
		t.Errorf("Initial Compute() = %d, want 0", score.Compute())
	}
}

func TestSpecterScoreConnectionSignal(t *testing.T) {
	score := NewSpecterScore()

	initial := score.Compute()

	score.SetConnectionCount(20)
	withConnections := score.Compute()

	if withConnections <= initial {
		t.Errorf("Score with 20 connections (%d) should be greater than initial (%d)",
			withConnections, initial)
	}

	score.SetConnectionCount(100)
	manyConnections := score.Compute()

	if manyConnections <= withConnections {
		t.Errorf("Score with 100 connections (%d) should be greater than 20 (%d)",
			manyConnections, withConnections)
	}
}

func TestSpecterScoreDiversitySignal(t *testing.T) {
	score := NewSpecterScore()

	initial := score.Compute()

	clusterIDs := []string{"cluster-a", "cluster-b", "cluster-c"}
	score.SetClusterDiversity(clusterIDs, 5)

	withDiversity := score.Compute()
	if withDiversity <= initial {
		t.Errorf("Score with diversity (%d) should be greater than initial (%d)",
			withDiversity, initial)
	}
}

func TestSpecterScoreWaveSignal(t *testing.T) {
	score := NewSpecterScore()

	initial := score.Compute()

	score.SetWaveCount(10)
	withWaves := score.Compute()

	if withWaves <= initial {
		t.Errorf("Score with waves (%d) should be greater than initial (%d)",
			withWaves, initial)
	}

	score.AddWave()
	if score.SpecterWaveCount30d != 11 {
		t.Errorf("SpecterWaveCount30d = %d, want 11", score.SpecterWaveCount30d)
	}
}

func TestSpecterScoreAmplificationSignals(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetAmpReceived(10)
	withAmpReceived := score.Compute()
	if withAmpReceived <= initial {
		t.Errorf("Score with amp received (%d) should be greater than initial (%d)",
			withAmpReceived, initial)
	}

	score.SetAmpGiven(10)
	withAmpGiven := score.Compute()
	if withAmpGiven <= withAmpReceived {
		t.Errorf("Score with amp given (%d) should be greater than just received (%d)",
			withAmpGiven, withAmpReceived)
	}
}

func TestSpecterScoreGiftSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetGiftsSent(5)
	withGifts := score.Compute()

	if withGifts <= initial {
		t.Errorf("Score with gifts (%d) should be greater than initial (%d)",
			withGifts, initial)
	}

	score.AddGiftSent()
	if score.GiftsSent30d != 6 {
		t.Errorf("GiftsSent30d = %d, want 6", score.GiftsSent30d)
	}
}

func TestSpecterScoreEventSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetEventsParticipated(3)
	withEvents := score.Compute()

	if withEvents <= initial {
		t.Errorf("Score with events (%d) should be greater than initial (%d)",
			withEvents, initial)
	}

	score.AddEventParticipation()
	if score.EventsParticipated30d != 4 {
		t.Errorf("EventsParticipated30d = %d, want 4", score.EventsParticipated30d)
	}
}

func TestSpecterScoreMiniGameSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	activity := MiniGameActivity{
		PuzzleSolutions30d:   5,
		HuntClaims30d:        3,
		ForgeEntries30d:      2,
		OraclePredictions30d: 4,
		ShadowPlayRounds30d:  1,
	}
	score.SetMiniGameActivity(activity)

	withMiniGames := score.Compute()
	if withMiniGames <= initial {
		t.Errorf("Score with mini-games (%d) should be greater than initial (%d)",
			withMiniGames, initial)
	}

	// Test individual add methods.
	score2 := NewSpecterScore()
	score2.AddPuzzleSolution()
	score2.AddHuntClaim()
	score2.AddForgeEntry()
	score2.AddOraclePrediction()
	score2.AddShadowPlayRound()

	if score2.MiniGames.Total() != 5 {
		t.Errorf("MiniGames.Total() = %d, want 5", score2.MiniGames.Total())
	}
}

func TestSpecterScoreTerritorySignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetTerritoryStatus(3, 2) // 3 controlled, 2 contested

	withTerritory := score.Compute()
	if withTerritory <= initial {
		t.Errorf("Score with territory (%d) should be greater than initial (%d)",
			withTerritory, initial)
	}
}

func TestSpecterScoreCartographerSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetCartographerVisits(10)

	withCartographer := score.Compute()
	if withCartographer <= initial {
		t.Errorf("Score with cartographer (%d) should be greater than initial (%d)",
			withCartographer, initial)
	}
}

func TestSpecterScoreChainSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetChainContributions(5)
	withChain := score.Compute()

	if withChain <= initial {
		t.Errorf("Score with chain contributions (%d) should be greater than initial (%d)",
			withChain, initial)
	}

	score.AddChainContribution()
	if score.ChainContributions30d != 6 {
		t.Errorf("ChainContributions30d = %d, want 6", score.ChainContributions30d)
	}
}

func TestSpecterScoreZKClaimSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetZKClaimCount(3)
	withZKClaims := score.Compute()

	if withZKClaims <= initial {
		t.Errorf("Score with ZK claims (%d) should be greater than initial (%d)",
			withZKClaims, initial)
	}

	// ZK claims use linear scaling (3 * count), so adding 1 should add exactly 3.
	score.AddZKClaim()
	if score.ValidZKClaimCount != 4 {
		t.Errorf("ValidZKClaimCount = %d, want 4", score.ValidZKClaimCount)
	}
}

func TestSpecterScoreShroudSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetShroudUptime(1.0) // Full uptime as Shroud Node
	withShroud := score.Compute()

	if withShroud <= initial {
		t.Errorf("Score with Shroud operation (%d) should be greater than initial (%d)",
			withShroud, initial)
	}

	// Shroud operation has highest weight (25), so full uptime should add 25.
	breakdown := score.GetSignalBreakdown()
	if breakdown["ShroudOperation"] != 25.0 {
		t.Errorf("ShroudOperation score = %f, want 25.0", breakdown["ShroudOperation"])
	}

	// Test clamping.
	score.SetShroudUptime(-0.5)
	if score.ShroudUptime30d != 0 {
		t.Errorf("Negative uptime should clamp to 0, got %f", score.ShroudUptime30d)
	}

	score.SetShroudUptime(2.0)
	if score.ShroudUptime30d != 1.0 {
		t.Errorf("Uptime > 1 should clamp to 1, got %f", score.ShroudUptime30d)
	}
}

func TestSpecterScoreCouncilSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetCouncilCount(2)
	withCouncil := score.Compute()

	if withCouncil <= initial {
		t.Errorf("Score with council membership (%d) should be greater than initial (%d)",
			withCouncil, initial)
	}

	// Council uses linear scaling (10 * count).
	breakdown := score.GetSignalBreakdown()
	if breakdown["CouncilMembership"] != 20.0 {
		t.Errorf("CouncilMembership score = %f, want 20.0", breakdown["CouncilMembership"])
	}

	score.AddCouncilMembership()
	if score.ActiveCouncilCount != 3 {
		t.Errorf("ActiveCouncilCount = %d, want 3", score.ActiveCouncilCount)
	}
}

func TestSpecterScoreAgeSignal(t *testing.T) {
	score := NewSpecterScore()
	newScore := score.Compute()

	// Set first seen to 6 months ago.
	score.SetSpecterFirstSeen(time.Now().Add(-180 * 24 * time.Hour))
	sixMonthScore := score.Compute()

	if sixMonthScore <= newScore {
		t.Errorf("Score after 6 months (%d) should be greater than new (%d)",
			sixMonthScore, newScore)
	}

	// Set first seen to 1 year ago (max bonus).
	score.SetSpecterFirstSeen(time.Now().Add(-365 * 24 * time.Hour))
	oneYearScore := score.Compute()

	if oneYearScore <= sixMonthScore {
		t.Errorf("Score after 1 year (%d) should be greater than 6 months (%d)",
			oneYearScore, sixMonthScore)
	}
}

func TestSpecterScoreUptimeSignal(t *testing.T) {
	score := NewSpecterScore()
	initial := score.Compute()

	score.SetSpecterUptime(1.0)
	fullUptime := score.Compute()

	if fullUptime <= initial {
		t.Errorf("Score with full uptime (%d) should be greater than initial (%d)",
			fullUptime, initial)
	}

	// Test clamping.
	score.SetSpecterUptime(-0.5)
	if score.SpecterUptime30d != 0 {
		t.Errorf("Negative uptime should clamp to 0, got %f", score.SpecterUptime30d)
	}
}

func TestSpecterScoreRank(t *testing.T) {
	score := NewSpecterScore()

	if score.Rank() != RankNone {
		t.Errorf("Initial rank = %v, want None", score.Rank())
	}

	// Add enough activity to reach Shade (25).
	score.SetConnectionCount(50)
	score.SetWaveCount(30)
	score.SetAmpReceived(15)

	if score.Rank() < RankShade {
		t.Errorf("Rank after activity = %v (score %d), expected at least Shade",
			score.Rank(), score.Compute())
	}
}

func TestSpecterScoreSignalBreakdown(t *testing.T) {
	score := NewSpecterScore()
	score.SetConnectionCount(10)
	score.SetWaveCount(5)
	score.SetGiftsSent(3)
	score.SetSpecterUptime(0.8)

	breakdown := score.GetSignalBreakdown()

	expectedKeys := []string{
		"ConnectionCount",
		"ConnectionDiversity",
		"WaveOutput",
		"AmpReceived",
		"AmpGiven",
		"GiftVolume",
		"EventParticipation",
		"MiniGameActivity",
		"TerritoryInfluence",
		"CartographerScore",
		"WhisperChain",
		"ZKClaimCount",
		"ShroudOperation",
		"CouncilMembership",
		"SpecterAge",
		"SpecterUptime",
	}

	for _, key := range expectedKeys {
		if _, ok := breakdown[key]; !ok {
			t.Errorf("Missing key %s in signal breakdown", key)
		}
	}

	if breakdown["ConnectionCount"] <= 0 {
		t.Error("ConnectionCount signal should be > 0")
	}

	if breakdown["WaveOutput"] <= 0 {
		t.Error("WaveOutput signal should be > 0")
	}
}

func TestSpecterScoreCaching(t *testing.T) {
	score := NewSpecterScore()
	score.SetConnectionCount(10)

	first := score.Compute()
	second := score.Compute()

	if first != second {
		t.Errorf("Cached score mismatch: first=%d, second=%d", first, second)
	}

	score.AddConnection()
	third := score.Compute()

	if third == first {
		t.Error("Score should change after adding connection")
	}
}

func TestSpecterScorer(t *testing.T) {
	scorer := NewSpecterScorer()

	score1 := scorer.GetScore("specter-1")
	if score1 == nil {
		t.Fatal("GetScore returned nil")
	}

	score1.SetConnectionCount(50)

	score1Again := scorer.GetScore("specter-1")
	if score1Again.SpecterConnectionCount != 50 {
		t.Errorf("ConnectionCount = %d, want 50", score1Again.SpecterConnectionCount)
	}

	score2 := scorer.GetScore("specter-2")
	if score2.SpecterConnectionCount != 0 {
		t.Errorf("New specter ConnectionCount = %d, want 0", score2.SpecterConnectionCount)
	}

	if scorer.Count() != 2 {
		t.Errorf("Count = %d, want 2", scorer.Count())
	}

	scorer.RemoveScore("specter-1")
	if scorer.Count() != 1 {
		t.Errorf("Count after remove = %d, want 1", scorer.Count())
	}
}

func TestSpecterScorerTopSpecters(t *testing.T) {
	scorer := NewSpecterScorer()

	for i := 1; i <= 5; i++ {
		score := scorer.GetScore("specter-" + string(rune('0'+i)))
		score.SetConnectionCount(i * 10)
	}

	top := scorer.TopSpecters(3)
	if len(top) != 3 {
		t.Errorf("TopSpecters(3) returned %d items, want 3", len(top))
	}

	if top[0] != "specter-5" {
		t.Errorf("Top specter = %s, want specter-5", top[0])
	}
}

func TestDefaultSpecterWeights(t *testing.T) {
	w := DefaultSpecterWeights()

	// Per RESONANCE_SYSTEM.md formula coefficients.
	if w.ConnectionCount != 10.0 {
		t.Errorf("ConnectionCount weight = %f, want 10.0", w.ConnectionCount)
	}
	if w.WaveOutput != 8.0 {
		t.Errorf("WaveOutput weight = %f, want 8.0", w.WaveOutput)
	}
	if w.AmpReceived != 15.0 {
		t.Errorf("AmpReceived weight = %f, want 15.0", w.AmpReceived)
	}
	if w.AmpGiven != 5.0 {
		t.Errorf("AmpGiven weight = %f, want 5.0", w.AmpGiven)
	}
	if w.GiftVolume != 6.0 {
		t.Errorf("GiftVolume weight = %f, want 6.0", w.GiftVolume)
	}
	if w.EventParticipation != 4.0 {
		t.Errorf("EventParticipation weight = %f, want 4.0", w.EventParticipation)
	}
	if w.MiniGameActivity != 7.0 {
		t.Errorf("MiniGameActivity weight = %f, want 7.0", w.MiniGameActivity)
	}
	if w.TerritoryInfluence != 3.0 {
		t.Errorf("TerritoryInfluence weight = %f, want 3.0", w.TerritoryInfluence)
	}
	if w.CartographerScore != 6.0 {
		t.Errorf("CartographerScore weight = %f, want 6.0", w.CartographerScore)
	}
	if w.WhisperChain != 5.0 {
		t.Errorf("WhisperChain weight = %f, want 5.0", w.WhisperChain)
	}
	if w.ZKClaimCount != 3.0 {
		t.Errorf("ZKClaimCount weight = %f, want 3.0", w.ZKClaimCount)
	}
	if w.ShroudOperation != 25.0 {
		t.Errorf("ShroudOperation weight = %f, want 25.0", w.ShroudOperation)
	}
	if w.CouncilMembership != 10.0 {
		t.Errorf("CouncilMembership weight = %f, want 10.0", w.CouncilMembership)
	}
	if w.SpecterAge != 20.0 {
		t.Errorf("SpecterAge weight = %f, want 20.0", w.SpecterAge)
	}
	if w.SpecterUptime != 10.0 {
		t.Errorf("SpecterUptime weight = %f, want 10.0", w.SpecterUptime)
	}
}

func TestMiniGameActivityTotal(t *testing.T) {
	activity := MiniGameActivity{
		PuzzleSolutions30d:   5,
		HuntClaims30d:        3,
		ForgeEntries30d:      2,
		OraclePredictions30d: 4,
		ShadowPlayRounds30d:  1,
	}

	if activity.Total() != 15 {
		t.Errorf("MiniGameActivity.Total() = %d, want 15", activity.Total())
	}
}

func TestSpecterScoreRealisticValues(t *testing.T) {
	// Test a typical active Specter after 6 months.
	score := NewSpecterScore()

	// Connection signals.
	score.SetConnectionCount(25)
	score.SetClusterDiversity([]string{"a", "b", "c", "d"}, 8)

	// Wave activity.
	score.SetWaveCount(40)
	score.SetAmpReceived(15)
	score.SetAmpGiven(10)

	// Gift and event activity.
	score.SetGiftsSent(8)
	score.SetEventsParticipated(5)

	// Mini-games.
	score.SetMiniGameActivity(MiniGameActivity{
		PuzzleSolutions30d:   10,
		HuntClaims30d:        5,
		ForgeEntries30d:      3,
		OraclePredictions30d: 8,
		ShadowPlayRounds30d:  2,
	})

	// Territory and exploration.
	score.SetTerritoryStatus(2, 3)
	score.SetCartographerVisits(15)

	// Infrastructure contribution.
	score.SetChainContributions(12)
	score.SetZKClaimCount(2)

	// Time-based.
	score.SetSpecterFirstSeen(time.Now().Add(-180 * 24 * time.Hour))
	score.SetSpecterUptime(0.7)

	computed := score.Compute()

	// A fairly active Specter should score in the Phantom (100+) range.
	if computed < 80 || computed > 250 {
		t.Errorf("Typical active Specter score = %d, expected roughly 80-200 range", computed)
	}

	t.Logf("Typical active Specter score: %d (rank: %s)", computed, score.Rank())
	t.Logf("Signal breakdown: %+v", score.GetSignalBreakdown())
}

func TestSpecterScoreHighInfrastructure(t *testing.T) {
	// Test a Shroud Node operator with Council membership.
	score := NewSpecterScore()

	// Moderate connection/activity.
	score.SetConnectionCount(20)
	score.SetWaveCount(20)
	score.SetAmpReceived(10)

	// High infrastructure contribution.
	score.SetShroudUptime(0.95) // Full-time Shroud Node
	score.SetCouncilCount(2)    // 2 active Phantom Councils
	score.SetZKClaimCount(4)    // Several ZK Claims
	score.SetChainContributions(30)

	// Established account.
	score.SetSpecterFirstSeen(time.Now().Add(-300 * 24 * time.Hour))
	score.SetSpecterUptime(0.9)

	computed := score.Compute()

	// Shroud Node operators should score very high due to the 25x weight.
	if computed < 100 {
		t.Errorf("High-infrastructure Specter score = %d, expected 100+", computed)
	}

	t.Logf("Shroud Node operator score: %d (rank: %s)", computed, score.Rank())
}
