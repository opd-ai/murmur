// Package tutorials tests verify hint management and tutorial flow.
package tutorials

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if !m.IsEnabled() {
		t.Error("manager should be enabled by default")
	}
}

func TestDefaultHints(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	hints := m.GetAllHints()

	if len(hints) == 0 {
		t.Error("expected default hints to be registered")
	}

	// Check for specific important hints
	found := map[HintID]bool{
		HintPulseMapPan:  false,
		HintPulseMapZoom: false,
		HintWaveCreate:   false,
	}

	for _, hint := range hints {
		if _, ok := found[hint.ID]; ok {
			found[hint.ID] = true
		}
	}

	for id, wasFound := range found {
		if !wasFound {
			t.Errorf("expected default hint %s not found", id)
		}
	}
}

func TestTriggerHint(t *testing.T) {
	var shown atomic.Bool
	m := NewManager(ManagerCallbacks{
		OnShow: func(hint *Hint) {
			shown.Store(true)
		},
	})

	// Trigger a hint
	if !m.TriggerHint(HintPulseMapPan) {
		t.Error("TriggerHint should return true")
	}
	time.Sleep(10 * time.Millisecond)

	if !shown.Load() {
		t.Error("OnShow callback should have been called")
	}

	// Active hint should be set
	active := m.ActiveHint()
	if active == nil || active.ID != HintPulseMapPan {
		t.Error("ActiveHint should return triggered hint")
	}
}

func TestShowOnceConstraint(t *testing.T) {
	m := NewManager(ManagerCallbacks{})

	// First trigger should work
	if !m.TriggerHint(HintPulseMapPan) {
		t.Error("first trigger should succeed")
	}

	m.DismissHint()

	// Second trigger should fail (show-once)
	if m.TriggerHint(HintPulseMapPan) {
		t.Error("second trigger should fail for show-once hint")
	}
}

func TestHintCooldown(t *testing.T) {
	m := NewManager(ManagerCallbacks{})

	// Register a hint with cooldown (not show-once)
	m.RegisterHint(&Hint{
		ID:       "test.cooldown",
		Title:    "Test",
		Content:  "Test content",
		ShowOnce: false,
		Cooldown: 100 * time.Millisecond,
	})

	// First trigger works
	if !m.TriggerHint("test.cooldown") {
		t.Error("first trigger should succeed")
	}
	m.DismissHint()

	// Immediate second trigger fails (cooldown)
	if m.TriggerHint("test.cooldown") {
		t.Error("trigger during cooldown should fail")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Now it should work
	if !m.TriggerHint("test.cooldown") {
		t.Error("trigger after cooldown should succeed")
	}
}

func TestDismissHint(t *testing.T) {
	var dismissed atomic.Bool
	m := NewManager(ManagerCallbacks{
		OnDismiss: func(id HintID) {
			dismissed.Store(true)
		},
	})

	m.TriggerHint(HintPulseMapPan)
	m.DismissHint()
	time.Sleep(10 * time.Millisecond)

	if !dismissed.Load() {
		t.Error("OnDismiss callback should have been called")
	}

	if m.ActiveHint() != nil {
		t.Error("ActiveHint should be nil after dismiss")
	}
}

func TestDisableHints(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	m.Disable()

	if m.IsEnabled() {
		t.Error("manager should be disabled")
	}

	if m.TriggerHint(HintPulseMapPan) {
		t.Error("trigger should fail when disabled")
	}

	m.Enable()
	if !m.TriggerHint(HintPulseMapPan) {
		t.Error("trigger should succeed after re-enable")
	}
}

func TestAcknowledgeHint(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	m.TriggerHint(HintPulseMapPan)
	m.AcknowledgeHint(HintPulseMapPan)

	state := m.GetHintState(HintPulseMapPan)
	if state == nil || !state.Acknowledged {
		t.Error("hint should be acknowledged")
	}
}

func TestResetHints(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	m.TriggerHint(HintPulseMapPan)
	m.ResetHints()

	state := m.GetHintState(HintPulseMapPan)
	if state == nil {
		t.Fatal("state should exist after reset")
	}
	if state.Shown {
		t.Error("state should be reset")
	}

	// Should be able to trigger again
	if !m.TriggerHint(HintPulseMapPan) {
		t.Error("trigger should succeed after reset")
	}
}

func TestTriggerNonexistentHint(t *testing.T) {
	m := NewManager(ManagerCallbacks{})
	if m.TriggerHint("nonexistent.hint") {
		t.Error("trigger of nonexistent hint should fail")
	}
}

func TestNewTutorial(t *testing.T) {
	steps := []*TutorialStep{
		{ID: "step1", Title: "Step 1", Instruction: "Do something"},
		{ID: "step2", Title: "Step 2", Instruction: "Do something else"},
	}
	tutorial := NewTutorial("test", "Test Tutorial", "A test", steps)

	if tutorial == nil {
		t.Fatal("NewTutorial returned nil")
	}
	if tutorial.ID != "test" {
		t.Error("tutorial ID mismatch")
	}
	if len(tutorial.Steps) != 2 {
		t.Error("steps count mismatch")
	}
}

func TestTutorialFlow(t *testing.T) {
	completed := false
	steps := []*TutorialStep{
		{ID: "step1", Title: "Step 1", OnComplete: func() {}},
		{ID: "step2", Title: "Step 2", OnComplete: func() { completed = true }},
	}
	tutorial := NewTutorial("test", "Test", "Test", steps)

	if tutorial.CurrentStep() != nil {
		t.Error("should have no current step before start")
	}

	tutorial.Start()
	if tutorial.CurrentStep() == nil || tutorial.CurrentStep().ID != "step1" {
		t.Error("should be on step1 after start")
	}

	if tutorial.Progress() != 0 {
		t.Error("progress should be 0 at start")
	}

	tutorial.Advance()
	if tutorial.CurrentStep().ID != "step2" {
		t.Error("should be on step2 after advance")
	}

	if tutorial.Progress() != 50 {
		t.Errorf("progress should be 50, got %f", tutorial.Progress())
	}

	tutorial.Advance()
	if !tutorial.IsComplete() {
		t.Error("tutorial should be complete")
	}
	if completed != true {
		t.Error("OnComplete should have been called")
	}

	if tutorial.Progress() != 100 {
		t.Errorf("progress should be 100, got %f", tutorial.Progress())
	}
}

func TestTutorialSkip(t *testing.T) {
	steps := []*TutorialStep{
		{ID: "step1", Optional: false},
		{ID: "step2", Optional: true},
		{ID: "step3", Optional: false},
	}
	tutorial := NewTutorial("test", "Test", "Test", steps)
	tutorial.Start()

	// Cannot skip non-optional step
	if tutorial.Skip() {
		t.Error("should not be able to skip non-optional step")
	}

	tutorial.Advance()
	// Can skip optional step
	if !tutorial.Skip() {
		t.Error("should be able to skip optional step")
	}

	if tutorial.CurrentStep().ID != "step3" {
		t.Error("should be on step3 after skip")
	}
}

func TestTutorialDuration(t *testing.T) {
	tutorial := NewTutorial("test", "Test", "Test", nil)

	if tutorial.Duration() != 0 {
		t.Error("duration should be 0 before start")
	}

	tutorial.Start()
	time.Sleep(50 * time.Millisecond)

	if tutorial.Duration() < 40*time.Millisecond {
		t.Error("duration should be > 40ms")
	}
}

func TestEmptyTutorial(t *testing.T) {
	tutorial := NewTutorial("empty", "Empty", "No steps", nil)
	tutorial.Start()

	if tutorial.Progress() != 100 {
		t.Error("empty tutorial should have 100% progress")
	}
}
