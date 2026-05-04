// Package mechanics implements anonymous game mechanics for the Anonymous Layer.
// This file implements Specter Trophies per ANONYMOUS_GAME_MECHANICS.md.
package mechanics

import (
	"errors"
	"sync"
	"time"
)

// Trophy categories per ANONYMOUS_GAME_MECHANICS.md.
const (
	TrophyCategoryMilestone = iota + 1 // Resonance milestone achievements
	TrophyCategoryActivity             // Cumulative action achievements
	TrophyCategoryRare                 // Difficult or unusual achievements
)

// TrophyID uniquely identifies a trophy type.
type TrophyID string

// Milestone trophies - unlocked by Resonance milestones.
const (
	TrophyFirstShade       TrophyID = "first_shade"       // Resonance 25
	TrophyWraithRising     TrophyID = "wraith_rising"     // Resonance 50
	TrophyPhantomAscendant TrophyID = "phantom_ascendant" // Resonance 100
	TrophyRevenant         TrophyID = "revenant"          // Resonance 200
	TrophyAbyssWalker      TrophyID = "abyss_walker"      // Resonance 500
)

// Activity trophies - unlocked by cumulative actions.
const (
	TrophyFirstGiftSent      TrophyID = "first_gift_sent"
	TrophyTenPuzzlesSolved   TrophyID = "ten_puzzles_solved"
	TrophyFiveHuntsCompleted TrophyID = "five_hunts_completed"
	TrophyThreeForgesWon     TrophyID = "three_forges_won"
	TrophyFirstShadowPlay    TrophyID = "first_shadow_play"
	TrophyFirstTerritoryCtrl TrophyID = "first_territory_controlled"
	TrophyHundredWaves       TrophyID = "hundred_waves_published"
)

// Rare trophies - unusual or difficult achievements.
const (
	TrophyCartographer   TrophyID = "cartographer"    // 50 territories discovered
	TrophyOracle         TrophyID = "oracle"          // 10 correct predictions in a row
	TrophyChainBreaker   TrophyID = "chain_breaker"   // Echo chain length 10+
	TrophyGhost          TrophyID = "ghost"           // Resonance 100+ for 90 days
	TrophyCouncilFounder TrophyID = "council_founder" // Initiate a Phantom Council
)

// TrophyResonanceBonus defines Resonance bonuses per category.
const (
	MilestoneTrophyBonus = 0 // Milestone trophies: no bonus, just recognition
	ActivityTrophyBonus  = 1 // Activity trophies: +1 Resonance (non-decaying)
	RareTrophyBonus      = 3 // Rare trophies: +3 Resonance (non-decaying)
)

// Trophy errors.
var (
	ErrTrophyAlreadyUnlocked = errors.New("trophy already unlocked")
	ErrTrophyNotFound        = errors.New("trophy not found")
	ErrInvalidTrophyID       = errors.New("invalid trophy ID")
)

// TrophyDefinition describes a trophy's metadata.
type TrophyDefinition struct {
	ID          TrophyID
	Name        string
	Description string
	Category    int
	Threshold   int64 // Resonance or count threshold to unlock
	Bonus       int   // Resonance bonus when unlocked
	Animated    bool  // Whether glyph is animated (rare trophies)
}

// TrophyUnlock records when a Specter unlocked a trophy.
type TrophyUnlock struct {
	TrophyID   TrophyID
	UnlockedAt time.Time
	Resonance  float64 // Resonance at time of unlock
}

// TrophyStore tracks a Specter's unlocked trophies.
type TrophyStore struct {
	mu        sync.RWMutex
	unlocks   map[TrophyID]*TrophyUnlock
	specter   [32]byte
	createdAt time.Time
}

// ActivityCounters tracks counts for activity-based trophies.
type ActivityCounters struct {
	mu                      sync.RWMutex
	GiftsSent               int
	PuzzlesSolved           int
	HuntsCompleted          int
	ForgesWon               int
	ShadowPlaysParticipated int
	TerritoriesControlled   int
	WavesPublished          int
	TerritoriesDiscovered   int
	OracleCorrectStreak     int
	MaxEchoChainLength      int
	CouncilsCreated         int
	ResonanceAbove100Days   int
	LastResonanceCheck      time.Time
}

// allTrophyDefinitions is the master list of trophy definitions.
var allTrophyDefinitions = map[TrophyID]*TrophyDefinition{
	// Milestone trophies
	TrophyFirstShade: {
		ID:          TrophyFirstShade,
		Name:        "First Shade",
		Description: "Reached Resonance 25",
		Category:    TrophyCategoryMilestone,
		Threshold:   25,
		Bonus:       MilestoneTrophyBonus,
		Animated:    false,
	},
	TrophyWraithRising: {
		ID:          TrophyWraithRising,
		Name:        "Wraith Rising",
		Description: "Reached Resonance 50",
		Category:    TrophyCategoryMilestone,
		Threshold:   50,
		Bonus:       MilestoneTrophyBonus,
		Animated:    false,
	},
	TrophyPhantomAscendant: {
		ID:          TrophyPhantomAscendant,
		Name:        "Phantom Ascendant",
		Description: "Reached Resonance 100",
		Category:    TrophyCategoryMilestone,
		Threshold:   100,
		Bonus:       MilestoneTrophyBonus,
		Animated:    false,
	},
	TrophyRevenant: {
		ID:          TrophyRevenant,
		Name:        "Revenant",
		Description: "Reached Resonance 200",
		Category:    TrophyCategoryMilestone,
		Threshold:   200,
		Bonus:       MilestoneTrophyBonus,
		Animated:    false,
	},
	TrophyAbyssWalker: {
		ID:          TrophyAbyssWalker,
		Name:        "Abyss Walker",
		Description: "Reached Resonance 500",
		Category:    TrophyCategoryMilestone,
		Threshold:   500,
		Bonus:       MilestoneTrophyBonus,
		Animated:    false,
	},
	// Activity trophies
	TrophyFirstGiftSent: {
		ID:          TrophyFirstGiftSent,
		Name:        "First Gift Sent",
		Description: "Sent your first Phantom Gift",
		Category:    TrophyCategoryActivity,
		Threshold:   1,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyTenPuzzlesSolved: {
		ID:          TrophyTenPuzzlesSolved,
		Name:        "Puzzle Master",
		Description: "Solved 10 Cipher Puzzles",
		Category:    TrophyCategoryActivity,
		Threshold:   10,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyFiveHuntsCompleted: {
		ID:          TrophyFiveHuntsCompleted,
		Name:        "Hunter",
		Description: "Completed 5 Specter Hunts",
		Category:    TrophyCategoryActivity,
		Threshold:   5,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyThreeForgesWon: {
		ID:          TrophyThreeForgesWon,
		Name:        "Forgemaster",
		Description: "Won 3 Sigil Forge competitions",
		Category:    TrophyCategoryActivity,
		Threshold:   3,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyFirstShadowPlay: {
		ID:          TrophyFirstShadowPlay,
		Name:        "Shadow Actor",
		Description: "Participated in your first Shadow Play",
		Category:    TrophyCategoryActivity,
		Threshold:   1,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyFirstTerritoryCtrl: {
		ID:          TrophyFirstTerritoryCtrl,
		Name:        "Territory Holder",
		Description: "Controlled your first territory",
		Category:    TrophyCategoryActivity,
		Threshold:   1,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	TrophyHundredWaves: {
		ID:          TrophyHundredWaves,
		Name:        "Wave Weaver",
		Description: "Published 100 Waves",
		Category:    TrophyCategoryActivity,
		Threshold:   100,
		Bonus:       ActivityTrophyBonus,
		Animated:    false,
	},
	// Rare trophies
	TrophyCartographer: {
		ID:          TrophyCartographer,
		Name:        "Cartographer",
		Description: "Discovered 50 territories",
		Category:    TrophyCategoryRare,
		Threshold:   50,
		Bonus:       RareTrophyBonus,
		Animated:    true,
	},
	TrophyOracle: {
		ID:          TrophyOracle,
		Name:        "Oracle",
		Description: "10 correct Oracle Pool predictions in a row",
		Category:    TrophyCategoryRare,
		Threshold:   10,
		Bonus:       RareTrophyBonus,
		Animated:    true,
	},
	TrophyChainBreaker: {
		ID:          TrophyChainBreaker,
		Name:        "Chain Breaker",
		Description: "Participated in an Echo Chain of length 10+",
		Category:    TrophyCategoryRare,
		Threshold:   10,
		Bonus:       RareTrophyBonus,
		Animated:    true,
	},
	TrophyGhost: {
		ID:          TrophyGhost,
		Name:        "Ghost",
		Description: "Maintained Resonance 100+ for 90 consecutive days",
		Category:    TrophyCategoryRare,
		Threshold:   90,
		Bonus:       RareTrophyBonus,
		Animated:    true,
	},
	TrophyCouncilFounder: {
		ID:          TrophyCouncilFounder,
		Name:        "Council Founder",
		Description: "Initiated a Phantom Council",
		Category:    TrophyCategoryRare,
		Threshold:   1,
		Bonus:       RareTrophyBonus,
		Animated:    true,
	},
}

// NewTrophyStore creates a new trophy store for a Specter.
func NewTrophyStore(specterKey [32]byte) *TrophyStore {
	return &TrophyStore{
		unlocks:   make(map[TrophyID]*TrophyUnlock),
		specter:   specterKey,
		createdAt: time.Now(),
	}
}

// GetTrophyDefinition returns the definition for a trophy ID.
func GetTrophyDefinition(id TrophyID) (*TrophyDefinition, error) {
	def, ok := allTrophyDefinitions[id]
	if !ok {
		return nil, ErrInvalidTrophyID
	}
	return def, nil
}

// AllTrophyDefinitions returns all trophy definitions.
func AllTrophyDefinitions() []*TrophyDefinition {
	defs := make([]*TrophyDefinition, 0, len(allTrophyDefinitions))
	for _, def := range allTrophyDefinitions {
		defs = append(defs, def)
	}
	return defs
}

// UnlockTrophy marks a trophy as unlocked.
func (s *TrophyStore) UnlockTrophy(id TrophyID, resonance float64) error {
	if _, ok := allTrophyDefinitions[id]; !ok {
		return ErrInvalidTrophyID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.unlocks[id]; exists {
		return ErrTrophyAlreadyUnlocked
	}

	s.unlocks[id] = &TrophyUnlock{
		TrophyID:   id,
		UnlockedAt: time.Now(),
		Resonance:  resonance,
	}
	return nil
}

// HasTrophy checks if a trophy has been unlocked.
func (s *TrophyStore) HasTrophy(id TrophyID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.unlocks[id]
	return exists
}

// GetUnlock returns the unlock record for a trophy.
func (s *TrophyStore) GetUnlock(id TrophyID) (*TrophyUnlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	unlock, exists := s.unlocks[id]
	if !exists {
		return nil, ErrTrophyNotFound
	}
	return unlock, nil
}

// AllUnlocks returns all unlocked trophies.
func (s *TrophyStore) AllUnlocks() []*TrophyUnlock {
	s.mu.RLock()
	defer s.mu.RUnlock()

	unlocks := make([]*TrophyUnlock, 0, len(s.unlocks))
	for _, u := range s.unlocks {
		unlocks = append(unlocks, u)
	}
	return unlocks
}

// TrophyCount returns the total number of unlocked trophies.
func (s *TrophyStore) TrophyCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.unlocks)
}

// TotalResonanceBonus calculates total Resonance bonus from trophies.
func (s *TrophyStore) TotalResonanceBonus() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for id := range s.unlocks {
		if def, ok := allTrophyDefinitions[id]; ok {
			total += def.Bonus
		}
	}
	return total
}

// SpecterKey returns the associated Specter's public key.
func (s *TrophyStore) SpecterKey() [32]byte {
	return s.specter
}

// TrophyEvaluator checks and awards trophies based on Resonance and activity.
type TrophyEvaluator struct {
	store    *TrophyStore
	counters *ActivityCounters
}

// NewTrophyEvaluator creates a new trophy evaluator.
func NewTrophyEvaluator(store *TrophyStore, counters *ActivityCounters) *TrophyEvaluator {
	return &TrophyEvaluator{
		store:    store,
		counters: counters,
	}
}

// CheckMilestoneTrophies evaluates Resonance milestones.
func (e *TrophyEvaluator) CheckMilestoneTrophies(resonance float64) []TrophyID {
	var awarded []TrophyID

	milestones := []struct {
		id        TrophyID
		threshold float64
	}{
		{TrophyFirstShade, 25},
		{TrophyWraithRising, 50},
		{TrophyPhantomAscendant, 100},
		{TrophyRevenant, 200},
		{TrophyAbyssWalker, 500},
	}

	for _, m := range milestones {
		if resonance >= m.threshold && !e.store.HasTrophy(m.id) {
			if err := e.store.UnlockTrophy(m.id, resonance); err == nil {
				awarded = append(awarded, m.id)
			}
		}
	}
	return awarded
}

// CheckActivityTrophies evaluates activity-based trophies.
func (e *TrophyEvaluator) CheckActivityTrophies(resonance float64) []TrophyID {
	var awarded []TrophyID
	c := e.counters

	c.mu.RLock()
	defer c.mu.RUnlock()

	checks := []struct {
		id    TrophyID
		count int
		min   int
	}{
		{TrophyFirstGiftSent, c.GiftsSent, 1},
		{TrophyTenPuzzlesSolved, c.PuzzlesSolved, 10},
		{TrophyFiveHuntsCompleted, c.HuntsCompleted, 5},
		{TrophyThreeForgesWon, c.ForgesWon, 3},
		{TrophyFirstShadowPlay, c.ShadowPlaysParticipated, 1},
		{TrophyFirstTerritoryCtrl, c.TerritoriesControlled, 1},
		{TrophyHundredWaves, c.WavesPublished, 100},
	}

	for _, check := range checks {
		if check.count >= check.min && !e.store.HasTrophy(check.id) {
			if err := e.store.UnlockTrophy(check.id, resonance); err == nil {
				awarded = append(awarded, check.id)
			}
		}
	}
	return awarded
}

// CheckRareTrophies evaluates rare achievement trophies.
func (e *TrophyEvaluator) CheckRareTrophies(resonance float64) []TrophyID {
	var awarded []TrophyID
	c := e.counters

	c.mu.RLock()
	defer c.mu.RUnlock()

	e.checkCartographerTrophy(c, resonance, &awarded)
	e.checkOracleTrophy(c, resonance, &awarded)
	e.checkChainBreakerTrophy(c, resonance, &awarded)
	e.checkGhostTrophy(c, resonance, &awarded)
	e.checkCouncilFounderTrophy(c, resonance, &awarded)

	return awarded
}

// checkCartographerTrophy checks and awards Cartographer trophy.
func (e *TrophyEvaluator) checkCartographerTrophy(c *ActivityCounters, resonance float64, awarded *[]TrophyID) {
	if c.TerritoriesDiscovered >= 50 && !e.store.HasTrophy(TrophyCartographer) {
		if err := e.store.UnlockTrophy(TrophyCartographer, resonance); err == nil {
			*awarded = append(*awarded, TrophyCartographer)
		}
	}
}

// checkOracleTrophy checks and awards Oracle trophy.
func (e *TrophyEvaluator) checkOracleTrophy(c *ActivityCounters, resonance float64, awarded *[]TrophyID) {
	if c.OracleCorrectStreak >= 10 && !e.store.HasTrophy(TrophyOracle) {
		if err := e.store.UnlockTrophy(TrophyOracle, resonance); err == nil {
			*awarded = append(*awarded, TrophyOracle)
		}
	}
}

// checkChainBreakerTrophy checks and awards Chain Breaker trophy.
func (e *TrophyEvaluator) checkChainBreakerTrophy(c *ActivityCounters, resonance float64, awarded *[]TrophyID) {
	if c.MaxEchoChainLength >= 10 && !e.store.HasTrophy(TrophyChainBreaker) {
		if err := e.store.UnlockTrophy(TrophyChainBreaker, resonance); err == nil {
			*awarded = append(*awarded, TrophyChainBreaker)
		}
	}
}

// checkGhostTrophy checks and awards Ghost trophy.
func (e *TrophyEvaluator) checkGhostTrophy(c *ActivityCounters, resonance float64, awarded *[]TrophyID) {
	if c.ResonanceAbove100Days >= 90 && !e.store.HasTrophy(TrophyGhost) {
		if err := e.store.UnlockTrophy(TrophyGhost, resonance); err == nil {
			*awarded = append(*awarded, TrophyGhost)
		}
	}
}

// checkCouncilFounderTrophy checks and awards Council Founder trophy.
func (e *TrophyEvaluator) checkCouncilFounderTrophy(c *ActivityCounters, resonance float64, awarded *[]TrophyID) {
	if c.CouncilsCreated >= 1 && !e.store.HasTrophy(TrophyCouncilFounder) {
		if err := e.store.UnlockTrophy(TrophyCouncilFounder, resonance); err == nil {
			*awarded = append(*awarded, TrophyCouncilFounder)
		}
	}
}

// CheckAllTrophies evaluates all trophy categories.
func (e *TrophyEvaluator) CheckAllTrophies(resonance float64) []TrophyID {
	var awarded []TrophyID
	awarded = append(awarded, e.CheckMilestoneTrophies(resonance)...)
	awarded = append(awarded, e.CheckActivityTrophies(resonance)...)
	awarded = append(awarded, e.CheckRareTrophies(resonance)...)
	return awarded
}

// NewActivityCounters creates a new activity counter tracker.
func NewActivityCounters() *ActivityCounters {
	return &ActivityCounters{
		LastResonanceCheck: time.Now(),
	}
}

// IncrementGiftsSent increments the gifts sent counter.
func (c *ActivityCounters) IncrementGiftsSent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.GiftsSent++
}

// IncrementPuzzlesSolved increments the puzzles solved counter.
func (c *ActivityCounters) IncrementPuzzlesSolved() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PuzzlesSolved++
}

// IncrementHuntsCompleted increments the hunts completed counter.
func (c *ActivityCounters) IncrementHuntsCompleted() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.HuntsCompleted++
}

// IncrementForgesWon increments the forges won counter.
func (c *ActivityCounters) IncrementForgesWon() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ForgesWon++
}

// IncrementShadowPlays increments the shadow plays participated counter.
func (c *ActivityCounters) IncrementShadowPlays() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ShadowPlaysParticipated++
}

// IncrementTerritoriesControlled increments the territories controlled counter.
func (c *ActivityCounters) IncrementTerritoriesControlled() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TerritoriesControlled++
}

// IncrementWavesPublished increments the waves published counter.
func (c *ActivityCounters) IncrementWavesPublished() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WavesPublished++
}

// IncrementTerritoriesDiscovered increments the territories discovered counter.
func (c *ActivityCounters) IncrementTerritoriesDiscovered() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TerritoriesDiscovered++
}

// IncrementOracleCorrectStreak increments or resets the oracle streak.
func (c *ActivityCounters) IncrementOracleCorrectStreak(correct bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if correct {
		c.OracleCorrectStreak++
	} else {
		c.OracleCorrectStreak = 0
	}
}

// UpdateMaxEchoChainLength updates the max echo chain if higher.
func (c *ActivityCounters) UpdateMaxEchoChainLength(length int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if length > c.MaxEchoChainLength {
		c.MaxEchoChainLength = length
	}
}

// IncrementCouncilsCreated increments the councils created counter.
func (c *ActivityCounters) IncrementCouncilsCreated() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CouncilsCreated++
}

// UpdateResonanceStreak updates the consecutive days above Resonance 100.
func (c *ActivityCounters) UpdateResonanceStreak(resonance float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	daysSince := int(now.Sub(c.LastResonanceCheck).Hours() / 24)

	if daysSince >= 1 {
		if resonance >= 100 {
			c.ResonanceAbove100Days += daysSince
		} else {
			c.ResonanceAbove100Days = 0
		}
		c.LastResonanceCheck = now
	}
}

// GetSnapshot returns a copy of all counter values.
func (c *ActivityCounters) GetSnapshot() ActivityCounters {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return ActivityCounters{
		GiftsSent:               c.GiftsSent,
		PuzzlesSolved:           c.PuzzlesSolved,
		HuntsCompleted:          c.HuntsCompleted,
		ForgesWon:               c.ForgesWon,
		ShadowPlaysParticipated: c.ShadowPlaysParticipated,
		TerritoriesControlled:   c.TerritoriesControlled,
		WavesPublished:          c.WavesPublished,
		TerritoriesDiscovered:   c.TerritoriesDiscovered,
		OracleCorrectStreak:     c.OracleCorrectStreak,
		MaxEchoChainLength:      c.MaxEchoChainLength,
		CouncilsCreated:         c.CouncilsCreated,
		ResonanceAbove100Days:   c.ResonanceAbove100Days,
		LastResonanceCheck:      c.LastResonanceCheck,
	}
}
