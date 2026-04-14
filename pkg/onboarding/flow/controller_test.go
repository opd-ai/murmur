// Package flow tests verify the onboarding controller and state machine.
package flow

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewController(t *testing.T) {
	c := NewController(Callbacks{})
	if c == nil {
		t.Fatal("NewController returned nil")
	}
	if c.CurrentPhase() != PhaseWelcome {
		t.Errorf("expected initial phase Welcome, got %s", c.CurrentPhase())
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase Phase
		want  string
	}{
		{PhaseWelcome, "Welcome"},
		{PhaseIdentityCreation, "Identity Creation"},
		{PhaseModeSelection, "Mode Selection"},
		{PhaseNetworkBootstrap, "Network Bootstrap"},
		{PhaseGuidedExploration, "Guided Exploration"},
		{PhaseFirstWave, "First Wave"},
		{PhaseComplete, "Complete"},
		{Phase(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.want {
			t.Errorf("Phase(%d).String() = %q, want %q", tt.phase, got, tt.want)
		}
	}
}

func TestStartFlow(t *testing.T) {
	var startCalled atomic.Bool
	c := NewController(Callbacks{
		OnPhaseStart: func(phase Phase) {
			if phase == PhaseWelcome {
				startCalled.Store(true)
			}
		},
	})

	c.Start()
	time.Sleep(10 * time.Millisecond) // Wait for async callback

	if !startCalled.Load() {
		t.Error("OnPhaseStart callback was not called")
	}

	progress := c.Progress(PhaseWelcome)
	if progress == nil || !progress.Started {
		t.Error("Welcome phase should be started")
	}
}

func TestCompleteCurrentPhase(t *testing.T) {
	var mu sync.Mutex
	phases := make([]Phase, 0)
	c := NewController(Callbacks{
		OnPhaseComplete: func(phase Phase) {
			mu.Lock()
			phases = append(phases, phase)
			mu.Unlock()
		},
	})

	c.Start()

	// Complete Welcome phase
	c.CompleteCurrentPhase()
	time.Sleep(10 * time.Millisecond)

	if c.CurrentPhase() != PhaseIdentityCreation {
		t.Errorf("expected phase Identity Creation, got %s", c.CurrentPhase())
	}

	// Complete remaining phases
	for !c.IsComplete() {
		c.CompleteCurrentPhase()
	}
	time.Sleep(10 * time.Millisecond)

	if !c.IsComplete() {
		t.Error("expected flow to be complete")
	}

	mu.Lock()
	count := len(phases)
	mu.Unlock()

	if count != PhaseCount {
		t.Errorf("expected %d phase completions, got %d", PhaseCount, count)
	}
}

func TestSkipFlow(t *testing.T) {
	c := NewController(Callbacks{})
	c.Start()
	c.Skip()

	if !c.IsComplete() {
		t.Error("expected flow to be complete after skip")
	}
	if !c.IsSkipped() {
		t.Error("expected flow to be marked as skipped")
	}
}

func TestOverallProgress(t *testing.T) {
	c := NewController(Callbacks{})
	c.Start()

	if c.OverallProgress() != 0 {
		t.Errorf("expected 0%% progress at start, got %f", c.OverallProgress())
	}

	// Complete half the phases
	for i := 0; i < PhaseCount/2; i++ {
		c.CompleteCurrentPhase()
	}

	progress := c.OverallProgress()
	if progress < 40 || progress > 60 {
		t.Errorf("expected ~50%% progress, got %f", progress)
	}

	// Complete all phases
	for !c.IsComplete() {
		c.CompleteCurrentPhase()
	}

	if c.OverallProgress() != 100 {
		t.Errorf("expected 100%% progress at end, got %f", c.OverallProgress())
	}
}

func TestPhaseData(t *testing.T) {
	c := NewController(Callbacks{})
	c.Start()

	// Set data for Identity phase
	c.SetPhaseData(PhaseIdentityCreation, "displayName", "TestUser")
	c.SetPhaseData(PhaseIdentityCreation, "backupCreated", true)

	// Retrieve data
	name := c.GetPhaseData(PhaseIdentityCreation, "displayName")
	if name != "TestUser" {
		t.Errorf("expected displayName 'TestUser', got %v", name)
	}

	backup := c.GetPhaseData(PhaseIdentityCreation, "backupCreated")
	if backup != true {
		t.Errorf("expected backupCreated true, got %v", backup)
	}

	// Non-existent data
	missing := c.GetPhaseData(PhaseIdentityCreation, "nonexistent")
	if missing != nil {
		t.Errorf("expected nil for missing key, got %v", missing)
	}
}

func TestSaveRestoreState(t *testing.T) {
	c := NewController(Callbacks{})
	c.Start()
	c.CompleteCurrentPhase()
	c.CompleteCurrentPhase()
	c.SetPhaseData(PhaseIdentityCreation, "key", "value")

	// Save state
	saved := c.SaveState()
	if saved == nil {
		t.Fatal("SaveState returned nil")
	}

	// Create new controller and restore
	c2 := NewController(Callbacks{})
	restored := c2.RestoreState(saved)
	if !restored {
		t.Error("RestoreState returned false")
	}

	if c2.CurrentPhase() != c.CurrentPhase() {
		t.Errorf("restored phase mismatch: %s vs %s", c2.CurrentPhase(), c.CurrentPhase())
	}

	data := c2.GetPhaseData(PhaseIdentityCreation, "key")
	if data != "value" {
		t.Errorf("restored data mismatch: %v", data)
	}
}

func TestTotalTime(t *testing.T) {
	c := NewController(Callbacks{})

	// Before start
	if c.TotalTime() != 0 {
		t.Error("expected 0 time before start")
	}

	c.Start()
	time.Sleep(50 * time.Millisecond)

	// During flow
	if c.TotalTime() < 40*time.Millisecond {
		t.Error("expected time > 40ms during flow")
	}

	// Complete flow
	for !c.IsComplete() {
		c.CompleteCurrentPhase()
	}

	// After completion, time should be fixed
	completedTime := c.TotalTime()
	time.Sleep(50 * time.Millisecond)
	if c.TotalTime() != completedTime {
		t.Error("time should be fixed after completion")
	}
}

func TestGetPhaseInfo(t *testing.T) {
	info := GetPhaseInfo()
	if len(info) != PhaseCount {
		t.Errorf("expected %d phase infos, got %d", PhaseCount, len(info))
	}

	// Check each phase has required fields
	for _, pi := range info {
		if pi.Title == "" {
			t.Errorf("phase %s has empty title", pi.Phase)
		}
		if pi.Description == "" {
			t.Errorf("phase %s has empty description", pi.Phase)
		}
		if pi.Icon == "" {
			t.Errorf("phase %s has empty icon", pi.Phase)
		}
	}
}

func TestFlowCompletionCallback(t *testing.T) {
	var completedDuration atomic.Int64
	c := NewController(Callbacks{
		OnFlowComplete: func(duration time.Duration) {
			completedDuration.Store(int64(duration))
		},
	})

	c.Start()
	for !c.IsComplete() {
		c.CompleteCurrentPhase()
	}
	time.Sleep(10 * time.Millisecond)

	if completedDuration.Load() == 0 {
		t.Error("OnFlowComplete callback should provide duration")
	}
}

func TestRestoreNilState(t *testing.T) {
	c := NewController(Callbacks{})
	if c.RestoreState(nil) {
		t.Error("RestoreState(nil) should return false")
	}
}

func TestProgressForInvalidPhase(t *testing.T) {
	c := NewController(Callbacks{})
	if c.Progress(Phase(99)) != nil {
		t.Error("Progress for invalid phase should return nil")
	}
}
