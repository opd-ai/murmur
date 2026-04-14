package resonance

import (
	"sync"
	"testing"
	"time"
)

func TestNewCallbackManager(t *testing.T) {
	cm := NewCallbackManager()
	if cm == nil {
		t.Fatal("NewCallbackManager returned nil")
	}
	if cm.scores == nil {
		t.Error("scores map not initialized")
	}
	if cm.eventCallbacks == nil {
		t.Error("eventCallbacks slice not initialized")
	}
	if cm.resonanceCallbacks == nil {
		t.Error("resonanceCallbacks slice not initialized")
	}
	if cm.historyLimit != 1000 {
		t.Errorf("historyLimit = %d, want 1000", cm.historyLimit)
	}
}

func TestRegisterAndUnregisterScore(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	if cm.ScoreCount() != 1 {
		t.Errorf("ScoreCount() = %d, want 1", cm.ScoreCount())
	}

	got := cm.GetScore(key)
	if got != score {
		t.Error("GetScore returned wrong score")
	}

	cm.UnregisterScore(key)
	if cm.ScoreCount() != 0 {
		t.Errorf("ScoreCount() after unregister = %d, want 0", cm.ScoreCount())
	}

	got = cm.GetScore(key)
	if got != nil {
		t.Error("GetScore after unregister should return nil")
	}
}

func TestProcessEventPuzzleSolved(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	event := NewPuzzleSolvedEvent(key, 1.0)
	cm.ProcessEvent(event)

	if score.MiniGames.PuzzleSolutions30d != 1 {
		t.Errorf("PuzzleSolutions30d = %d, want 1", score.MiniGames.PuzzleSolutions30d)
	}

	// Process another.
	cm.ProcessEvent(NewPuzzleSolvedEvent(key, 2.0))
	if score.MiniGames.PuzzleSolutions30d != 2 {
		t.Errorf("PuzzleSolutions30d = %d, want 2", score.MiniGames.PuzzleSolutions30d)
	}
}

func TestProcessEventHuntClaimed(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewHuntClaimedEvent(key))
	cm.ProcessEvent(NewHuntClaimedEvent(key))
	cm.ProcessEvent(NewHuntClaimedEvent(key))

	if score.MiniGames.HuntClaims30d != 3 {
		t.Errorf("HuntClaims30d = %d, want 3", score.MiniGames.HuntClaims30d)
	}
}

func TestProcessEventForgeEntry(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewForgeEntryEvent(key))

	if score.MiniGames.ForgeEntries30d != 1 {
		t.Errorf("ForgeEntries30d = %d, want 1", score.MiniGames.ForgeEntries30d)
	}
}

func TestProcessEventOraclePrediction(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewOraclePredictionEvent(key))

	if score.MiniGames.OraclePredictions30d != 1 {
		t.Errorf("OraclePredictions30d = %d, want 1", score.MiniGames.OraclePredictions30d)
	}
}

func TestProcessEventShadowPlayRound(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewShadowPlayRoundEvent(key))
	cm.ProcessEvent(NewShadowPlayRoundEvent(key))

	if score.MiniGames.ShadowPlayRounds30d != 2 {
		t.Errorf("ShadowPlayRounds30d = %d, want 2", score.MiniGames.ShadowPlayRounds30d)
	}
}

func TestProcessEventTerritoryControl(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewTerritoryControlEvent(key))

	score.mu.RLock()
	controlled := score.Territories.Controlled
	score.mu.RUnlock()

	if controlled != 1 {
		t.Errorf("Territories.Controlled = %d, want 1", controlled)
	}
}

func TestProcessEventCouncilAction(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewCouncilActionEvent(key))

	score.mu.RLock()
	councils := score.ActiveCouncilCount
	score.mu.RUnlock()

	if councils != 1 {
		t.Errorf("ActiveCouncilCount = %d, want 1", councils)
	}
}

func TestEventCallbacks(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	var receivedEvents []MiniGameEvent
	var mu sync.Mutex

	cm.OnEvent(func(event MiniGameEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))
	cm.ProcessEvent(NewHuntClaimedEvent(key))

	mu.Lock()
	eventCount := len(receivedEvents)
	mu.Unlock()

	if eventCount != 2 {
		t.Errorf("received %d events, want 2", eventCount)
	}
}

func TestResonanceChangeCallbacks(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	var changes []struct {
		old, new int
	}
	var mu sync.Mutex

	cm.OnResonanceChange(func(specterKey [32]byte, oldScore, newScore int) {
		mu.Lock()
		changes = append(changes, struct{ old, new int }{oldScore, newScore})
		mu.Unlock()
	})

	// First event should trigger change from 0 to non-zero.
	cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))

	mu.Lock()
	changeCount := len(changes)
	mu.Unlock()

	if changeCount != 1 {
		t.Errorf("received %d resonance changes, want 1", changeCount)
	}
}

func TestEventHistory(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Process some events.
	for i := 0; i < 5; i++ {
		cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))
	}

	history := cm.GetEventHistory()
	if len(history) != 5 {
		t.Errorf("GetEventHistory returned %d events, want 5", len(history))
	}

	cm.ClearHistory()
	history = cm.GetEventHistory()
	if len(history) != 0 {
		t.Errorf("GetEventHistory after clear returned %d events, want 0", len(history))
	}
}

func TestEventHistoryLimit(t *testing.T) {
	cm := NewCallbackManager()
	cm.historyLimit = 10 // Small limit for testing.

	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Process more events than the limit.
	for i := 0; i < 20; i++ {
		cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))
	}

	history := cm.GetEventHistory()
	if len(history) != 10 {
		t.Errorf("GetEventHistory returned %d events, want 10 (limit)", len(history))
	}
}

func TestGetEventsSince(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Process some events.
	cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))
	time.Sleep(10 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(10 * time.Millisecond)
	cm.ProcessEvent(NewHuntClaimedEvent(key))
	cm.ProcessEvent(NewForgeEntryEvent(key))

	events := cm.GetEventsSince(cutoff)
	if len(events) != 2 {
		t.Errorf("GetEventsSince returned %d events, want 2", len(events))
	}
}

func TestGetEventsForSpecter(t *testing.T) {
	cm := NewCallbackManager()

	var key1 [32]byte
	copy(key1[:], []byte("test-specter-key-1-1234567890123"))

	var key2 [32]byte
	copy(key2[:], []byte("test-specter-key-2-1234567890123"))

	// Process events for both specters.
	cm.ProcessEvent(NewPuzzleSolvedEvent(key1, 1.0))
	cm.ProcessEvent(NewPuzzleSolvedEvent(key1, 1.0))
	cm.ProcessEvent(NewPuzzleSolvedEvent(key2, 1.0))

	events1 := cm.GetEventsForSpecter(key1)
	events2 := cm.GetEventsForSpecter(key2)

	if len(events1) != 2 {
		t.Errorf("GetEventsForSpecter(key1) returned %d events, want 2", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("GetEventsForSpecter(key2) returned %d events, want 1", len(events2))
	}
}

func TestMiniGameEventTypeString(t *testing.T) {
	tests := []struct {
		eventType MiniGameEventType
		want      string
	}{
		{EventPuzzleSolved, "PuzzleSolved"},
		{EventHuntClaimed, "HuntClaimed"},
		{EventForgeEntry, "ForgeEntry"},
		{EventOraclePrediction, "OraclePrediction"},
		{EventShadowPlayRound, "ShadowPlayRound"},
		{EventTerritoryControl, "TerritoryControl"},
		{EventCouncilAction, "CouncilAction"},
		{MiniGameEventType(255), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.eventType.String()
		if got != tt.want {
			t.Errorf("%d.String() = %s, want %s", tt.eventType, got, tt.want)
		}
	}
}

func TestAutoCreateScoreOnEvent(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Don't register score first.
	cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))

	// Score should be auto-created.
	score := cm.GetScore(key)
	if score == nil {
		t.Fatal("Score should be auto-created on event")
	}

	if score.MiniGames.PuzzleSolutions30d != 1 {
		t.Errorf("PuzzleSolutions30d = %d, want 1", score.MiniGames.PuzzleSolutions30d)
	}
}

func TestForgeWonEvent(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	cm.ProcessEvent(NewForgeWonEvent(key))

	// Forge win should count as forge entry + event participation.
	if score.MiniGames.ForgeEntries30d != 1 {
		t.Errorf("ForgeEntries30d = %d, want 1", score.MiniGames.ForgeEntries30d)
	}
	if score.EventsParticipated30d != 1 {
		t.Errorf("EventsParticipated30d = %d, want 1", score.EventsParticipated30d)
	}
}

func TestOracleResolvedEvent(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	// Accurate prediction.
	cm.ProcessEvent(NewOracleResolvedEvent(key, true, 0.95))

	if score.EventsParticipated30d != 1 {
		t.Errorf("EventsParticipated30d = %d, want 1 for accurate prediction", score.EventsParticipated30d)
	}

	// Inaccurate prediction.
	cm.ProcessEvent(NewOracleResolvedEvent(key, false, 0.1))

	// Should not increment for inaccurate.
	if score.EventsParticipated30d != 1 {
		t.Errorf("EventsParticipated30d = %d, want 1 (inaccurate should not add)", score.EventsParticipated30d)
	}
}

func TestInitiationEventsCountAsParticipation(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	// Initiation events.
	cm.ProcessEvent(MiniGameEvent{
		EventType:  EventPuzzleInitiated,
		SpecterKey: key,
		Timestamp:  time.Now(),
	})

	if score.EventsParticipated30d != 1 {
		t.Errorf("EventsParticipated30d = %d, want 1 for puzzle initiation", score.EventsParticipated30d)
	}
}

func TestConcurrentEventProcessing(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	score := NewSpecterScore()
	cm.RegisterScore(key, score)

	var wg sync.WaitGroup
	eventCount := 100

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.ProcessEvent(NewPuzzleSolvedEvent(key, 1.0))
		}()
	}

	wg.Wait()

	if score.MiniGames.PuzzleSolutions30d != eventCount {
		t.Errorf("PuzzleSolutions30d = %d, want %d", score.MiniGames.PuzzleSolutions30d, eventCount)
	}
}

func TestHelperFunctions(t *testing.T) {
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Test all helper functions create valid events.
	tests := []struct {
		name  string
		event MiniGameEvent
		want  MiniGameEventType
	}{
		{"PuzzleSolved", NewPuzzleSolvedEvent(key, 1.0), EventPuzzleSolved},
		{"HuntClaimed", NewHuntClaimedEvent(key), EventHuntClaimed},
		{"ForgeEntry", NewForgeEntryEvent(key), EventForgeEntry},
		{"ForgeWon", NewForgeWonEvent(key), EventForgeWon},
		{"OraclePrediction", NewOraclePredictionEvent(key), EventOraclePrediction},
		{"OracleResolved", NewOracleResolvedEvent(key, true, 0.9), EventOracleResolved},
		{"ShadowPlayRound", NewShadowPlayRoundEvent(key), EventShadowPlayRound},
		{"ShadowPlayWin", NewShadowPlayWinEvent(key), EventShadowPlayWin},
		{"TerritoryControl", NewTerritoryControlEvent(key), EventTerritoryControl},
		{"CouncilAction", NewCouncilActionEvent(key), EventCouncilAction},
		{"Ignition", NewIgnitionEvent(key), EventIgnition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.EventType != tt.want {
				t.Errorf("EventType = %v, want %v", tt.event.EventType, tt.want)
			}
			if tt.event.SpecterKey != key {
				t.Error("SpecterKey mismatch")
			}
			if tt.event.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

// TestProcessEventIgnition tests that Ignition events grant Resonance bonus.
func TestProcessEventIgnition(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Process first Ignition event.
	event := NewIgnitionEvent(key)
	cm.ProcessEvent(event)

	score := cm.GetScore(key)
	if score == nil {
		t.Fatal("Score not created for Specter")
	}

	// First Ignition should grant 3 Resonance.
	if score.GetIgnitionCount() != 1 {
		t.Errorf("IgnitionCount = %d, want 1", score.GetIgnitionCount())
	}

	// Score should include Ignition bonus.
	computed := score.Compute()
	if computed < IgnitionBonus {
		t.Errorf("Computed score %d < IgnitionBonus %d", computed, IgnitionBonus)
	}
}

// TestIgnitionBonusCap tests that only first 10 Ignitions grant bonus.
func TestIgnitionBonusCap(t *testing.T) {
	cm := NewCallbackManager()
	var key [32]byte
	copy(key[:], []byte("test-specter-key-12345678901234"))

	// Process 15 Ignition events.
	for i := 0; i < 15; i++ {
		event := NewIgnitionEvent(key)
		cm.ProcessEvent(event)
	}

	score := cm.GetScore(key)
	if score == nil {
		t.Fatal("Score not created")
	}

	// All 15 should be counted.
	if score.GetIgnitionCount() != 15 {
		t.Errorf("IgnitionCount = %d, want 15", score.GetIgnitionCount())
	}

	// But only first 10 should contribute to score.
	// Max Ignition bonus = 10 * 3 = 30.
	breakdown := score.GetSignalBreakdown()
	ignitionScore := breakdown["Ignition"]
	expectedMax := float64(IgnitionMaxBonusCount * IgnitionBonus)
	if ignitionScore != expectedMax {
		t.Errorf("Ignition score = %v, want %v (capped at 10)", ignitionScore, expectedMax)
	}
}

// TestIgnitionEventTypeString tests EventIgnition string representation.
func TestIgnitionEventTypeString(t *testing.T) {
	if EventIgnition.String() != "Ignition" {
		t.Errorf("EventIgnition.String() = %q, want %q", EventIgnition.String(), "Ignition")
	}
}
