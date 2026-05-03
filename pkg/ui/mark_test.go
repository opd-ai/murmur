// Package ui tests for Specter Mark placement panel.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"errors"
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

func TestNewMarkPanel(t *testing.T) {
	theme := Theme{}
	callbacks := MarkPanelCallbacks{}
	panel := NewMarkPanel(theme, callbacks)

	if panel == nil {
		t.Fatal("Expected non-nil panel")
	}
	if panel.IsVisible() {
		t.Error("Panel should not be visible initially")
	}
	if panel.GetMode() != MarkModeCategorySelect {
		t.Error("Expected initial mode to be CategorySelect")
	}
}

func TestMarkPanelShowWithInsufficientResonance(t *testing.T) {
	theme := Theme{}
	callbacks := MarkPanelCallbacks{
		GetMyResonance: func() int { return 50 }, // Below 100 minimum
	}
	panel := NewMarkPanel(theme, callbacks)

	panel.Show()

	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show")
	}
	if panel.GetMode() != MarkModeError {
		t.Errorf("Expected Error mode, got %d", panel.GetMode())
	}
	if panel.GetErrorMessage() == "" {
		t.Error("Expected error message about resonance")
	}
}

func TestMarkPanelShowWithMaxMarks(t *testing.T) {
	theme := Theme{}
	callbacks := MarkPanelCallbacks{
		GetMyResonance:     func() int { return 100 },
		GetActiveMarkCount: func() int { return 5 }, // At max
	}
	panel := NewMarkPanel(theme, callbacks)

	panel.Show()

	if panel.GetMode() != MarkModeError {
		t.Errorf("Expected Error mode for max marks, got %d", panel.GetMode())
	}
	if panel.GetErrorMessage() == "" {
		t.Error("Expected error message about max marks")
	}
}

func TestMarkPanelShowSuccess(t *testing.T) {
	theme := Theme{}
	targets := []TargetInfo{
		{NodeID: "abc123", DisplayName: "TestNode", IsSurface: true},
	}
	callbacks := MarkPanelCallbacks{
		GetMyResonance:     func() int { return 150 },
		GetActiveMarkCount: func() int { return 2 },
		GetTargets:         func() []TargetInfo { return targets },
	}
	panel := NewMarkPanel(theme, callbacks)

	panel.Show()

	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show")
	}
	if panel.GetMode() != MarkModeCategorySelect {
		t.Errorf("Expected CategorySelect mode, got %d", panel.GetMode())
	}
}

func TestMarkPanelHide(t *testing.T) {
	closeCalled := false
	callbacks := MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		OnClose:        func() { closeCalled = true },
	}
	panel := NewMarkPanel(Theme{}, callbacks)

	panel.Show()
	panel.Hide()

	if panel.IsVisible() {
		t.Error("Panel should not be visible after Hide")
	}
	if !closeCalled {
		t.Error("OnClose callback should be called")
	}
}

func TestMarkPanelCategorySelection(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
	})
	panel.Show()

	// Default is Watcher (index 0).
	if panel.GetSelectedCategory() != marks.MarkWatcher {
		t.Errorf("Expected Watcher, got %d", panel.GetSelectedCategory())
	}

	// Select Ally.
	panel.SimulateSelectCategory(1)
	if panel.GetSelectedCategory() != marks.MarkAlly {
		t.Errorf("Expected Ally, got %d", panel.GetSelectedCategory())
	}

	// Select Rival.
	panel.SimulateSelectCategory(2)
	if panel.GetSelectedCategory() != marks.MarkRival {
		t.Errorf("Expected Rival, got %d", panel.GetSelectedCategory())
	}
}

func TestMarkPanelTargetSelection(t *testing.T) {
	targets := []TargetInfo{
		{NodeID: "node1", DisplayName: "Node One"},
		{NodeID: "node2", DisplayName: "Node Two"},
		{NodeID: "node3", DisplayName: "Node Three"},
	}
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return targets },
	})
	panel.Show()

	// Advance to target selection.
	panel.SimulateAdvanceMode()

	if panel.GetMode() != MarkModeTarget {
		t.Errorf("Expected Target mode, got %d", panel.GetMode())
	}

	// Select second target.
	panel.SimulateSelectTarget(1)
	selected := panel.GetSelectedTarget()
	if selected == nil {
		t.Fatal("Expected selected target")
	}
	if selected.NodeID != "node2" {
		t.Errorf("Expected node2, got %s", selected.NodeID)
	}
}

func TestMarkPanelCannotSelectSelf(t *testing.T) {
	targets := []TargetInfo{
		{NodeID: "self", DisplayName: "Self", IsSelf: true},
		{NodeID: "other", DisplayName: "Other"},
	}
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return targets },
	})
	panel.Show()
	panel.SimulateAdvanceMode() // To target mode

	// Select self and try to advance.
	panel.SimulateSelectTarget(0)
	panel.SimulateAdvanceMode() // Should NOT advance because it's self

	if panel.GetMode() != MarkModeTarget {
		t.Error("Should not advance when self is selected")
	}
}

func TestMarkPanelCannotSelectAlreadyMarked(t *testing.T) {
	targets := []TargetInfo{
		{NodeID: "marked", DisplayName: "Marked", HasMark: true},
		{NodeID: "other", DisplayName: "Other"},
	}
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return targets },
	})
	panel.Show()
	panel.SimulateAdvanceMode() // To target mode

	// Select marked node and try to advance.
	panel.SimulateSelectTarget(0)
	panel.SimulateAdvanceMode() // Should NOT advance

	if panel.GetMode() != MarkModeTarget {
		t.Error("Should not advance when target is already marked")
	}
}

func TestMarkPanelNoteInput(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets: func() []TargetInfo {
			return []TargetInfo{{NodeID: "test", DisplayName: "Test"}}
		},
	})
	panel.Show()
	panel.SimulateAdvanceMode() // To target mode
	panel.SimulateAdvanceMode() // To confirm mode

	// Set note.
	panel.SimulateSetNote("Test note for the mark")
	if panel.GetNote() != "Test note for the mark" {
		t.Errorf("Expected note to be set, got %q", panel.GetNote())
	}
}

func TestMarkPanelNoteMaxLength(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
	})

	// Try to set note longer than max.
	longNote := ""
	for i := 0; i < 200; i++ {
		longNote += "a"
	}
	panel.SimulateSetNote(longNote)

	if panel.GetNote() == longNote {
		t.Error("Should not accept note longer than max")
	}
}

func TestMarkPanelConfirmPlacement(t *testing.T) {
	placeCalled := false
	var placedCategory marks.MarkCategory
	var placedTarget string
	var placedNote string

	targets := []TargetInfo{
		{NodeID: "target123", DisplayName: "Target Node"},
	}
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return targets },
		OnPlaceMark: func(cat marks.MarkCategory, targetID, note string) error {
			placeCalled = true
			placedCategory = cat
			placedTarget = targetID
			placedNote = note
			return nil
		},
	})

	panel.Show()
	panel.SimulateSelectCategory(1) // Ally
	panel.SimulateAdvanceMode()     // To target
	panel.SimulateAdvanceMode()     // To confirm
	panel.SimulateSetNote("friendly mark")
	err := panel.SimulateConfirmPlacement()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !placeCalled {
		t.Error("OnPlaceMark should be called")
	}
	if placedCategory != marks.MarkAlly {
		t.Errorf("Expected Ally category, got %d", placedCategory)
	}
	if placedTarget != "target123" {
		t.Errorf("Expected target123, got %s", placedTarget)
	}
	if placedNote != "friendly mark" {
		t.Errorf("Expected note, got %q", placedNote)
	}
	if panel.GetMode() != MarkModeSuccess {
		t.Errorf("Expected Success mode, got %d", panel.GetMode())
	}
}

func TestMarkPanelPlacementError(t *testing.T) {
	targets := []TargetInfo{
		{NodeID: "target", DisplayName: "Target"},
	}
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return targets },
		OnPlaceMark: func(cat marks.MarkCategory, targetID, note string) error {
			return errors.New("network error")
		},
	})

	panel.Show()
	panel.SimulateAdvanceMode() // To target
	panel.SimulateAdvanceMode() // To confirm
	err := panel.SimulateConfirmPlacement()

	if err == nil {
		t.Error("Expected error")
	}
	if panel.GetMode() != MarkModeError {
		t.Errorf("Expected Error mode, got %d", panel.GetMode())
	}
	if panel.GetErrorMessage() != "network error" {
		t.Errorf("Expected error message, got %q", panel.GetErrorMessage())
	}
}

func TestMarkPanelSetError(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{})

	panel.SetError("custom error")

	if panel.GetMode() != MarkModeError {
		t.Errorf("Expected Error mode")
	}
	if panel.GetErrorMessage() != "custom error" {
		t.Errorf("Expected custom error message")
	}
}

func TestMarkPanelSetSuccess(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{})

	panel.SetSuccess("mark placed successfully")

	if panel.GetMode() != MarkModeSuccess {
		t.Errorf("Expected Success mode")
	}
	if panel.GetSuccessMessage() != "mark placed successfully" {
		t.Errorf("Expected success message")
	}
}

func TestMarkPanelUpdate(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
	})
	panel.Show()

	// Update should not panic.
	if err := panel.Update(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMarkPanelDraw(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
	})
	panel.Show()

	// Draw should not panic with nil screen in noebiten mode.
	panel.Draw(nil)
}

func TestMarkPanelNoTargets(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets:     func() []TargetInfo { return nil },
	})
	panel.Show()
	panel.SimulateAdvanceMode() // Try to advance with no targets

	// Mode should still be CategorySelect or Error.
	if panel.GetMode() == MarkModeTarget {
		t.Error("Should not advance to target mode with no targets")
	}
}

func TestMarkPanelInvalidTargetIndex(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets: func() []TargetInfo {
			return []TargetInfo{{NodeID: "test", DisplayName: "Test"}}
		},
	})
	panel.Show()

	// Try invalid index.
	panel.SimulateSelectTarget(100)

	// Should not crash.
	selected := panel.GetSelectedTarget()
	if selected != nil && selected.NodeID == "" {
		// Invalid selection handled.
	}
}

func TestMarkPanelConcurrentAccess(t *testing.T) {
	panel := NewMarkPanel(Theme{}, MarkPanelCallbacks{
		GetMyResonance: func() int { return 100 },
		GetTargets: func() []TargetInfo {
			return []TargetInfo{{NodeID: "test", DisplayName: "Test"}}
		},
	})

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			panel.Show()
			_ = panel.IsVisible()
			_ = panel.GetMode()
			_ = panel.GetSelectedCategory()
			_ = panel.GetSelectedTarget()
			_ = panel.GetNote()
			_ = panel.Update()
			panel.Hide()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
