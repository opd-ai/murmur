package mechanics

import (
	"testing"
	"time"
)

func TestNewTrophyStore(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}

	store := NewTrophyStore(key)
	if store == nil {
		t.Fatal("NewTrophyStore returned nil")
	}
	if store.TrophyCount() != 0 {
		t.Errorf("Expected 0 trophies, got %d", store.TrophyCount())
	}
	if store.SpecterKey() != key {
		t.Error("Specter key mismatch")
	}
}

func TestUnlockTrophy(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)

	// Unlock a valid trophy
	err := store.UnlockTrophy(TrophyFirstShade, 25.0)
	if err != nil {
		t.Errorf("UnlockTrophy failed: %v", err)
	}

	// Verify it's unlocked
	if !store.HasTrophy(TrophyFirstShade) {
		t.Error("Trophy should be unlocked")
	}
	if store.TrophyCount() != 1 {
		t.Errorf("Expected 1 trophy, got %d", store.TrophyCount())
	}

	// Try to unlock again (should fail)
	err = store.UnlockTrophy(TrophyFirstShade, 25.0)
	if err != ErrTrophyAlreadyUnlocked {
		t.Errorf("Expected ErrTrophyAlreadyUnlocked, got %v", err)
	}
}

func TestUnlockInvalidTrophy(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)

	err := store.UnlockTrophy("invalid_trophy", 100.0)
	if err != ErrInvalidTrophyID {
		t.Errorf("Expected ErrInvalidTrophyID, got %v", err)
	}
}

func TestGetUnlock(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)

	// Trophy not found
	_, err := store.GetUnlock(TrophyFirstShade)
	if err != ErrTrophyNotFound {
		t.Errorf("Expected ErrTrophyNotFound, got %v", err)
	}

	// Unlock and retrieve
	store.UnlockTrophy(TrophyFirstShade, 30.0)
	unlock, err := store.GetUnlock(TrophyFirstShade)
	if err != nil {
		t.Errorf("GetUnlock failed: %v", err)
	}
	if unlock.TrophyID != TrophyFirstShade {
		t.Error("Trophy ID mismatch")
	}
	if unlock.Resonance != 30.0 {
		t.Errorf("Expected resonance 30.0, got %f", unlock.Resonance)
	}
}

func TestAllUnlocks(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)

	store.UnlockTrophy(TrophyFirstShade, 25.0)
	store.UnlockTrophy(TrophyWraithRising, 50.0)
	store.UnlockTrophy(TrophyFirstGiftSent, 10.0)

	unlocks := store.AllUnlocks()
	if len(unlocks) != 3 {
		t.Errorf("Expected 3 unlocks, got %d", len(unlocks))
	}
}

func TestTotalResonanceBonus(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)

	// Milestone trophies have 0 bonus
	store.UnlockTrophy(TrophyFirstShade, 25.0)
	if store.TotalResonanceBonus() != 0 {
		t.Errorf("Milestone trophy should have 0 bonus, got %d", store.TotalResonanceBonus())
	}

	// Activity trophy has +1 bonus
	store.UnlockTrophy(TrophyFirstGiftSent, 10.0)
	if store.TotalResonanceBonus() != 1 {
		t.Errorf("Expected 1 bonus, got %d", store.TotalResonanceBonus())
	}

	// Rare trophy has +3 bonus
	store.UnlockTrophy(TrophyCouncilFounder, 200.0)
	if store.TotalResonanceBonus() != 4 {
		t.Errorf("Expected 4 total bonus, got %d", store.TotalResonanceBonus())
	}
}

func TestGetTrophyDefinition(t *testing.T) {
	def, err := GetTrophyDefinition(TrophyFirstShade)
	if err != nil {
		t.Errorf("GetTrophyDefinition failed: %v", err)
	}
	if def.ID != TrophyFirstShade {
		t.Error("ID mismatch")
	}
	if def.Name != "First Shade" {
		t.Errorf("Name mismatch: %s", def.Name)
	}
	if def.Category != TrophyCategoryMilestone {
		t.Error("Category should be milestone")
	}
	if def.Threshold != 25 {
		t.Errorf("Threshold should be 25, got %d", def.Threshold)
	}

	// Invalid trophy
	_, err = GetTrophyDefinition("invalid")
	if err != ErrInvalidTrophyID {
		t.Errorf("Expected ErrInvalidTrophyID, got %v", err)
	}
}

func TestAllTrophyDefinitions(t *testing.T) {
	defs := AllTrophyDefinitions()
	if len(defs) == 0 {
		t.Error("Expected non-empty definitions")
	}

	// Check all trophies are present
	expectedCount := 17 // 5 milestone + 7 activity + 5 rare
	if len(defs) != expectedCount {
		t.Errorf("Expected %d definitions, got %d", expectedCount, len(defs))
	}
}

func TestTrophyEvaluator_MilestoneTrophies(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)
	counters := NewActivityCounters()
	eval := NewTrophyEvaluator(store, counters)

	// At resonance 25, should get FirstShade
	awarded := eval.CheckMilestoneTrophies(25.0)
	if len(awarded) != 1 || awarded[0] != TrophyFirstShade {
		t.Errorf("Expected FirstShade, got %v", awarded)
	}

	// At resonance 50, should get WraithRising
	awarded = eval.CheckMilestoneTrophies(50.0)
	if len(awarded) != 1 || awarded[0] != TrophyWraithRising {
		t.Errorf("Expected WraithRising, got %v", awarded)
	}

	// Check again at same resonance (should not re-award)
	awarded = eval.CheckMilestoneTrophies(50.0)
	if len(awarded) != 0 {
		t.Errorf("Should not re-award, got %v", awarded)
	}

	// At resonance 500, should get remaining 3
	awarded = eval.CheckMilestoneTrophies(500.0)
	if len(awarded) != 3 {
		t.Errorf("Expected 3 awards, got %d", len(awarded))
	}
}

func TestTrophyEvaluator_ActivityTrophies(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)
	counters := NewActivityCounters()
	eval := NewTrophyEvaluator(store, counters)

	// No activity, no trophies
	awarded := eval.CheckActivityTrophies(10.0)
	if len(awarded) != 0 {
		t.Errorf("Expected 0 awards, got %d", len(awarded))
	}

	// Send a gift
	counters.IncrementGiftsSent()
	awarded = eval.CheckActivityTrophies(10.0)
	if len(awarded) != 1 || awarded[0] != TrophyFirstGiftSent {
		t.Errorf("Expected FirstGiftSent, got %v", awarded)
	}

	// Solve 10 puzzles
	for i := 0; i < 10; i++ {
		counters.IncrementPuzzlesSolved()
	}
	awarded = eval.CheckActivityTrophies(10.0)
	if len(awarded) != 1 || awarded[0] != TrophyTenPuzzlesSolved {
		t.Errorf("Expected TenPuzzlesSolved, got %v", awarded)
	}
}

func TestTrophyEvaluator_RareTrophies(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)
	counters := NewActivityCounters()
	eval := NewTrophyEvaluator(store, counters)

	// Create a council
	counters.IncrementCouncilsCreated()
	awarded := eval.CheckRareTrophies(200.0)
	if len(awarded) != 1 || awarded[0] != TrophyCouncilFounder {
		t.Errorf("Expected CouncilFounder, got %v", awarded)
	}

	// Discover 50 territories
	for i := 0; i < 50; i++ {
		counters.IncrementTerritoriesDiscovered()
	}
	awarded = eval.CheckRareTrophies(200.0)
	if len(awarded) != 1 || awarded[0] != TrophyCartographer {
		t.Errorf("Expected Cartographer, got %v", awarded)
	}
}

func TestTrophyEvaluator_CheckAllTrophies(t *testing.T) {
	var key [32]byte
	store := NewTrophyStore(key)
	counters := NewActivityCounters()
	eval := NewTrophyEvaluator(store, counters)

	// Set up some achievements
	counters.IncrementGiftsSent()
	counters.IncrementCouncilsCreated()

	// Check all at once
	awarded := eval.CheckAllTrophies(200.0)

	// Should get: FirstShade, WraithRising, PhantomAscendant, Revenant,
	//             FirstGiftSent, CouncilFounder
	if len(awarded) != 6 {
		t.Errorf("Expected 6 awards, got %d", len(awarded))
	}

	// Verify specific trophies
	if !store.HasTrophy(TrophyRevenant) {
		t.Error("Should have Revenant")
	}
	if !store.HasTrophy(TrophyFirstGiftSent) {
		t.Error("Should have FirstGiftSent")
	}
	if !store.HasTrophy(TrophyCouncilFounder) {
		t.Error("Should have CouncilFounder")
	}
}

func TestActivityCounters_IncrementMethods(t *testing.T) {
	c := NewActivityCounters()

	c.IncrementGiftsSent()
	c.IncrementPuzzlesSolved()
	c.IncrementHuntsCompleted()
	c.IncrementForgesWon()
	c.IncrementShadowPlays()
	c.IncrementTerritoriesControlled()
	c.IncrementWavesPublished()
	c.IncrementTerritoriesDiscovered()
	c.IncrementCouncilsCreated()

	snap := c.GetSnapshot()
	if snap.GiftsSent != 1 {
		t.Errorf("GiftsSent = %d, want 1", snap.GiftsSent)
	}
	if snap.PuzzlesSolved != 1 {
		t.Errorf("PuzzlesSolved = %d, want 1", snap.PuzzlesSolved)
	}
	if snap.HuntsCompleted != 1 {
		t.Errorf("HuntsCompleted = %d, want 1", snap.HuntsCompleted)
	}
	if snap.ForgesWon != 1 {
		t.Errorf("ForgesWon = %d, want 1", snap.ForgesWon)
	}
	if snap.ShadowPlaysParticipated != 1 {
		t.Errorf("ShadowPlaysParticipated = %d, want 1", snap.ShadowPlaysParticipated)
	}
	if snap.TerritoriesControlled != 1 {
		t.Errorf("TerritoriesControlled = %d, want 1", snap.TerritoriesControlled)
	}
	if snap.WavesPublished != 1 {
		t.Errorf("WavesPublished = %d, want 1", snap.WavesPublished)
	}
	if snap.TerritoriesDiscovered != 1 {
		t.Errorf("TerritoriesDiscovered = %d, want 1", snap.TerritoriesDiscovered)
	}
	if snap.CouncilsCreated != 1 {
		t.Errorf("CouncilsCreated = %d, want 1", snap.CouncilsCreated)
	}
}

func TestActivityCounters_OracleStreak(t *testing.T) {
	c := NewActivityCounters()

	// Build up streak
	for i := 0; i < 10; i++ {
		c.IncrementOracleCorrectStreak(true)
	}
	if c.GetSnapshot().OracleCorrectStreak != 10 {
		t.Errorf("Streak should be 10, got %d", c.GetSnapshot().OracleCorrectStreak)
	}

	// Reset on wrong prediction
	c.IncrementOracleCorrectStreak(false)
	if c.GetSnapshot().OracleCorrectStreak != 0 {
		t.Error("Streak should reset to 0")
	}
}

func TestActivityCounters_EchoChainLength(t *testing.T) {
	c := NewActivityCounters()

	c.UpdateMaxEchoChainLength(5)
	if c.GetSnapshot().MaxEchoChainLength != 5 {
		t.Errorf("Max should be 5, got %d", c.GetSnapshot().MaxEchoChainLength)
	}

	// Lower value should not update
	c.UpdateMaxEchoChainLength(3)
	if c.GetSnapshot().MaxEchoChainLength != 5 {
		t.Error("Max should still be 5")
	}

	// Higher value should update
	c.UpdateMaxEchoChainLength(15)
	if c.GetSnapshot().MaxEchoChainLength != 15 {
		t.Errorf("Max should be 15, got %d", c.GetSnapshot().MaxEchoChainLength)
	}
}

func TestActivityCounters_ResonanceStreak(t *testing.T) {
	c := NewActivityCounters()

	// Set up a check from 2 days ago
	c.LastResonanceCheck = time.Now().Add(-48 * time.Hour)

	// Update with resonance above 100
	c.UpdateResonanceStreak(150.0)

	snap := c.GetSnapshot()
	if snap.ResonanceAbove100Days < 1 {
		t.Errorf("Should have at least 1 day, got %d", snap.ResonanceAbove100Days)
	}

	// Set up another check and drop below 100
	c.mu.Lock()
	c.LastResonanceCheck = time.Now().Add(-24 * time.Hour)
	c.mu.Unlock()

	c.UpdateResonanceStreak(50.0)
	snap = c.GetSnapshot()
	if snap.ResonanceAbove100Days != 0 {
		t.Errorf("Streak should reset, got %d", snap.ResonanceAbove100Days)
	}
}

func TestTrophyDefinition_Categories(t *testing.T) {
	// Verify category counts
	milestone, activity, rare := 0, 0, 0
	for _, def := range allTrophyDefinitions {
		switch def.Category {
		case TrophyCategoryMilestone:
			milestone++
		case TrophyCategoryActivity:
			activity++
		case TrophyCategoryRare:
			rare++
		}
	}

	if milestone != 5 {
		t.Errorf("Expected 5 milestone trophies, got %d", milestone)
	}
	if activity != 7 {
		t.Errorf("Expected 7 activity trophies, got %d", activity)
	}
	if rare != 5 {
		t.Errorf("Expected 5 rare trophies, got %d", rare)
	}
}

func TestTrophyDefinition_AnimatedFlag(t *testing.T) {
	// Only rare trophies should be animated
	for _, def := range allTrophyDefinitions {
		if def.Category == TrophyCategoryRare && !def.Animated {
			t.Errorf("Rare trophy %s should be animated", def.ID)
		}
		if def.Category != TrophyCategoryRare && def.Animated {
			t.Errorf("Non-rare trophy %s should not be animated", def.ID)
		}
	}
}

func TestTrophyDefinition_BonusValues(t *testing.T) {
	for _, def := range allTrophyDefinitions {
		switch def.Category {
		case TrophyCategoryMilestone:
			if def.Bonus != MilestoneTrophyBonus {
				t.Errorf("Milestone trophy %s has wrong bonus: %d", def.ID, def.Bonus)
			}
		case TrophyCategoryActivity:
			if def.Bonus != ActivityTrophyBonus {
				t.Errorf("Activity trophy %s has wrong bonus: %d", def.ID, def.Bonus)
			}
		case TrophyCategoryRare:
			if def.Bonus != RareTrophyBonus {
				t.Errorf("Rare trophy %s has wrong bonus: %d", def.ID, def.Bonus)
			}
		}
	}
}
