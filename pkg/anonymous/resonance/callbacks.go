// Package resonance provides local reputation computation and rank thresholds.
// This file implements callback integration for mini-game result auto-scoring.
// Per ROADMAP.md line 380, this connects mechanics events to Resonance updates.
package resonance

import (
	"sync"
	"time"
)

// MiniGameEvent represents a mini-game result that affects Resonance scoring.
type MiniGameEvent struct {
	EventType  MiniGameEventType // Type of mini-game event.
	SpecterKey [32]byte          // Specter who participated.
	Timestamp  time.Time         // When the event occurred.
	Difficulty float64           // Optional difficulty multiplier.
	Won        bool              // Whether the participant won/succeeded.
	Bonus      float64           // Optional bonus multiplier.
}

// MiniGameEventType identifies the type of mini-game event.
type MiniGameEventType uint8

const (
	// EventPuzzleSolved - Cipher Puzzle solution submitted.
	EventPuzzleSolved MiniGameEventType = iota + 1

	// EventPuzzleInitiated - Cipher Puzzle created.
	EventPuzzleInitiated

	// EventHuntClaimed - Specter Hunt fragment claimed.
	EventHuntClaimed

	// EventHuntInitiated - Specter Hunt created.
	EventHuntInitiated

	// EventHuntCompleted - Hunt completed (all fragments claimed).
	EventHuntCompleted

	// EventForgeEntry - Sigil Forge entry submitted.
	EventForgeEntry

	// EventForgeWon - Sigil Forge won.
	EventForgeWon

	// EventForgeInitiated - Sigil Forge created.
	EventForgeInitiated

	// EventOraclePrediction - Oracle Pool prediction submitted.
	EventOraclePrediction

	// EventOracleResolved - Oracle Pool resolved with accuracy score.
	EventOracleResolved

	// EventOracleInitiated - Oracle Pool created.
	EventOracleInitiated

	// EventShadowPlayRound - Shadow Play round completed.
	EventShadowPlayRound

	// EventShadowPlayWin - Shadow Play game won.
	EventShadowPlayWin

	// EventShadowPlayInitiated - Shadow Play game created.
	EventShadowPlayInitiated

	// EventTerritoryControl - Territory control gained/maintained.
	EventTerritoryControl

	// EventTerritoryContested - Territory contested.
	EventTerritoryContested

	// EventCouncilAction - Phantom Council action.
	EventCouncilAction

	// EventIgnition - Proximity Ignition (first-contact) completed.
	// Per ROADMAP.md line 271: "first 10 = 3 Resonance each".
	EventIgnition
)

// String returns a human-readable event type name.
func (t MiniGameEventType) String() string {
	switch t {
	case EventPuzzleSolved:
		return "PuzzleSolved"
	case EventPuzzleInitiated:
		return "PuzzleInitiated"
	case EventHuntClaimed:
		return "HuntClaimed"
	case EventHuntInitiated:
		return "HuntInitiated"
	case EventHuntCompleted:
		return "HuntCompleted"
	case EventForgeEntry:
		return "ForgeEntry"
	case EventForgeWon:
		return "ForgeWon"
	case EventForgeInitiated:
		return "ForgeInitiated"
	case EventOraclePrediction:
		return "OraclePrediction"
	case EventOracleResolved:
		return "OracleResolved"
	case EventOracleInitiated:
		return "OracleInitiated"
	case EventShadowPlayRound:
		return "ShadowPlayRound"
	case EventShadowPlayWin:
		return "ShadowPlayWin"
	case EventShadowPlayInitiated:
		return "ShadowPlayInitiated"
	case EventTerritoryControl:
		return "TerritoryControl"
	case EventTerritoryContested:
		return "TerritoryContested"
	case EventCouncilAction:
		return "CouncilAction"
	case EventIgnition:
		return "Ignition"
	default:
		return "Unknown"
	}
}

// ResonanceCallback is a function called when a Specter's Resonance is updated.
type ResonanceCallback func(specterKey [32]byte, oldScore, newScore int)

// CallbackManager manages mini-game event callbacks and Resonance updates.
// It acts as the bridge between mechanics events and Resonance scoring.
type CallbackManager struct {
	mu sync.RWMutex

	// scores maps Specter keys to their SpecterScore instances.
	scores map[[32]byte]*SpecterScore

	// eventCallbacks are called when mini-game events are processed.
	eventCallbacks []func(MiniGameEvent)

	// resonanceCallbacks are called when Resonance scores change.
	resonanceCallbacks []ResonanceCallback

	// eventHistory stores recent events for auditing (last 1000 events).
	eventHistory []MiniGameEvent
	historyLimit int
}

// NewCallbackManager creates a new callback manager.
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		scores:             make(map[[32]byte]*SpecterScore),
		eventCallbacks:     make([]func(MiniGameEvent), 0),
		resonanceCallbacks: make([]ResonanceCallback, 0),
		eventHistory:       make([]MiniGameEvent, 0),
		historyLimit:       1000,
	}
}

// RegisterScore registers a SpecterScore for callback updates.
func (cm *CallbackManager) RegisterScore(specterKey [32]byte, score *SpecterScore) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.scores[specterKey] = score
}

// UnregisterScore removes a SpecterScore from callback updates.
func (cm *CallbackManager) UnregisterScore(specterKey [32]byte) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.scores, specterKey)
}

// GetScore retrieves the SpecterScore for a given key.
func (cm *CallbackManager) GetScore(specterKey [32]byte) *SpecterScore {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.scores[specterKey]
}

// OnEvent registers a callback for mini-game events.
func (cm *CallbackManager) OnEvent(cb func(MiniGameEvent)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.eventCallbacks = append(cm.eventCallbacks, cb)
}

// OnResonanceChange registers a callback for Resonance score changes.
func (cm *CallbackManager) OnResonanceChange(cb ResonanceCallback) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.resonanceCallbacks = append(cm.resonanceCallbacks, cb)
}

// ProcessEvent handles a mini-game event and updates Resonance accordingly.
// This is the main entry point for mechanics to trigger Resonance updates.
func (cm *CallbackManager) ProcessEvent(event MiniGameEvent) {
	cm.mu.Lock()

	// Record in history.
	cm.addToHistory(event)

	// Find or create the Specter's score.
	score := cm.scores[event.SpecterKey]
	if score == nil {
		score = NewSpecterScore()
		cm.scores[event.SpecterKey] = score
	}

	// Get old score before update.
	oldScore := score.Compute()

	// Apply the event to the score.
	cm.applyEventToScore(event, score)

	// Get new score after update.
	newScore := score.Compute()

	// Copy callbacks to avoid holding lock during calls.
	eventCBs := make([]func(MiniGameEvent), len(cm.eventCallbacks))
	copy(eventCBs, cm.eventCallbacks)

	resCBs := make([]ResonanceCallback, len(cm.resonanceCallbacks))
	copy(resCBs, cm.resonanceCallbacks)

	cm.mu.Unlock()

	// Fire event callbacks.
	for _, cb := range eventCBs {
		cb(event)
	}

	// Fire Resonance change callbacks if score changed.
	if oldScore != newScore {
		for _, cb := range resCBs {
			cb(event.SpecterKey, oldScore, newScore)
		}
	}
}

// addToHistory adds an event to the history, maintaining the limit.
func (cm *CallbackManager) addToHistory(event MiniGameEvent) {
	cm.eventHistory = append(cm.eventHistory, event)
	if len(cm.eventHistory) > cm.historyLimit {
		// Remove oldest events.
		cm.eventHistory = cm.eventHistory[len(cm.eventHistory)-cm.historyLimit:]
	}
}

// applyEventToScore updates the score based on the event type.
func (cm *CallbackManager) applyEventToScore(event MiniGameEvent, score *SpecterScore) {
	switch event.EventType {
	case EventPuzzleSolved:
		score.AddPuzzleSolution()
	case EventHuntClaimed:
		score.AddHuntClaim()
	case EventForgeEntry:
		score.AddForgeEntry()
	case EventOraclePrediction:
		score.AddOraclePrediction()
	case EventShadowPlayRound:
		score.AddShadowPlayRound()
	case EventTerritoryControl:
		cm.updateTerritoryControl(score, event)
	case EventTerritoryContested:
		cm.updateTerritoryContested(score, event)
	case EventCouncilAction:
		cm.updateCouncilAction(score, event)
	case EventIgnition:
		// Ignition grants 3 Resonance for first 10 contacts.
		score.AddIgnition()
	// Initiation events may grant smaller bonuses or track leadership.
	case EventPuzzleInitiated, EventHuntInitiated, EventForgeInitiated,
		EventOracleInitiated, EventShadowPlayInitiated:
		// Initiating grants a smaller bonus through event participation.
		score.AddEventParticipation()
	// Win events grant additional bonus on top of participation.
	case EventHuntCompleted, EventForgeWon, EventShadowPlayWin:
		cm.applyWinBonus(score, event)
	case EventOracleResolved:
		// Oracle resolution with accuracy is handled separately.
		cm.applyOracleResolution(score, event)
	}
}

// updateTerritoryControl updates controlled territory count.
func (cm *CallbackManager) updateTerritoryControl(score *SpecterScore, event MiniGameEvent) {
	score.mu.Lock()
	defer score.mu.Unlock()
	score.Territories.Controlled++
	score.invalidateCache()
}

// updateTerritoryContested updates contested territory count.
func (cm *CallbackManager) updateTerritoryContested(score *SpecterScore, event MiniGameEvent) {
	score.mu.Lock()
	defer score.mu.Unlock()
	score.Territories.Contested++
	score.invalidateCache()
}

// updateCouncilAction updates council membership count.
func (cm *CallbackManager) updateCouncilAction(score *SpecterScore, event MiniGameEvent) {
	score.mu.Lock()
	defer score.mu.Unlock()
	score.ActiveCouncilCount++
	score.invalidateCache()
}

// applyWinBonus applies bonus for winning a competitive event.
func (cm *CallbackManager) applyWinBonus(score *SpecterScore, event MiniGameEvent) {
	// Wins count as additional event participation.
	score.AddEventParticipation()

	// For forge wins, also count the entry.
	if event.EventType == EventForgeWon {
		score.AddForgeEntry()
	}
}

// applyOracleResolution handles Oracle resolution with accuracy.
func (cm *CallbackManager) applyOracleResolution(score *SpecterScore, event MiniGameEvent) {
	// If the prediction was accurate (Won flag), add participation.
	if event.Won {
		score.AddEventParticipation()
	}
}

// GetEventHistory returns a copy of recent events.
func (cm *CallbackManager) GetEventHistory() []MiniGameEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	history := make([]MiniGameEvent, len(cm.eventHistory))
	copy(history, cm.eventHistory)
	return history
}

// GetEventsSince returns events since a given timestamp.
func (cm *CallbackManager) GetEventsSince(since time.Time) []MiniGameEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]MiniGameEvent, 0)
	for _, event := range cm.eventHistory {
		if event.Timestamp.After(since) {
			result = append(result, event)
		}
	}
	return result
}

// GetEventsForSpecter returns events for a specific Specter.
func (cm *CallbackManager) GetEventsForSpecter(specterKey [32]byte) []MiniGameEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]MiniGameEvent, 0)
	for _, event := range cm.eventHistory {
		if event.SpecterKey == specterKey {
			result = append(result, event)
		}
	}
	return result
}

// ScoreCount returns the number of registered Specter scores.
func (cm *CallbackManager) ScoreCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.scores)
}

// ClearHistory clears the event history.
func (cm *CallbackManager) ClearHistory() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.eventHistory = make([]MiniGameEvent, 0)
}

// Helper functions for creating events from mechanics.

// NewPuzzleSolvedEvent creates an event for a puzzle solution.
func NewPuzzleSolvedEvent(specterKey [32]byte, difficulty float64) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventPuzzleSolved,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
		Difficulty: difficulty,
		Won:        true,
	}
}

// NewHuntClaimedEvent creates an event for a hunt fragment claim.
func NewHuntClaimedEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventHuntClaimed,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
		Won:        true,
	}
}

// NewForgeEntryEvent creates an event for a forge entry submission.
func NewForgeEntryEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventForgeEntry,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}

// NewForgeWonEvent creates an event for winning a forge.
func NewForgeWonEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventForgeWon,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
		Won:        true,
	}
}

// NewOraclePredictionEvent creates an event for an oracle prediction.
func NewOraclePredictionEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventOraclePrediction,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}

// NewOracleResolvedEvent creates an event for oracle resolution.
func NewOracleResolvedEvent(specterKey [32]byte, accurate bool, accuracy float64) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventOracleResolved,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
		Won:        accurate,
		Bonus:      accuracy,
	}
}

// NewShadowPlayRoundEvent creates an event for completing a Shadow Play round.
func NewShadowPlayRoundEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventShadowPlayRound,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}

// NewShadowPlayWinEvent creates an event for winning Shadow Play.
func NewShadowPlayWinEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventShadowPlayWin,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
		Won:        true,
	}
}

// NewTerritoryControlEvent creates an event for gaining territory control.
func NewTerritoryControlEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventTerritoryControl,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}

// NewCouncilActionEvent creates an event for a Phantom Council action.
func NewCouncilActionEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventCouncilAction,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}

// NewIgnitionEvent creates an event for a Proximity Ignition contact.
// Per ROADMAP.md line 271: "first 10 = 3 Resonance each".
func NewIgnitionEvent(specterKey [32]byte) MiniGameEvent {
	return MiniGameEvent{
		EventType:  EventIgnition,
		SpecterKey: specterKey,
		Timestamp:  time.Now(),
	}
}
