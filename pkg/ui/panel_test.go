// Package ui provides tests for UI panels.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"testing"
	"time"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	if theme.FontSize != 14 {
		t.Errorf("Expected FontSize 14, got %d", theme.FontSize)
	}
	if theme.Padding != 12 {
		t.Errorf("Expected Padding 12, got %d", theme.Padding)
	}
	if theme.PanelBackground.A != 240 {
		t.Errorf("Expected panel background alpha 240, got %d", theme.PanelBackground.A)
	}
}

func TestNewComposePanel(t *testing.T) {
	theme := DefaultTheme()
	var submitted bool
	callback := func(content string, waveType uint8, targetNodeID string) {
		submitted = true
	}

	panel := NewComposePanel(theme, callback)
	if panel == nil {
		t.Fatal("NewComposePanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}

	// Test show/hide.
	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}

	// Test toggle.
	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle()")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should not be visible after second Toggle()")
	}

	_ = submitted // Used in real tests with submit.
}

func TestComposePanelContent(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)

	// Test content setting.
	panel.SetContent("Hello, World!")
	if panel.Content() != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", panel.Content())
	}

	// Test content length limit.
	longContent := make([]byte, MaxWaveLength+100)
	for i := range longContent {
		longContent[i] = 'a'
	}
	panel.SetContent(string(longContent))
	if len(panel.Content()) != MaxWaveLength {
		t.Errorf("Content should be truncated to %d, got %d", MaxWaveLength, len(panel.Content()))
	}
}

func TestComposePanelTargetNode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)

	panel.SetTargetNode("node123")
	panel.mu.RLock()
	target := panel.targetNodeID
	panel.mu.RUnlock()

	if target != "node123" {
		t.Errorf("Expected target node 'node123', got '%s'", target)
	}
}

func TestComposePanelWaveType(t *testing.T) {
	theme := DefaultTheme()
	panel := NewComposePanel(theme, nil)

	panel.SetWaveType(0x02) // Reply Wave.
	panel.mu.RLock()
	waveType := panel.waveType
	panel.mu.RUnlock()

	if waveType != 0x02 {
		t.Errorf("Expected wave type 0x02, got 0x%02x", waveType)
	}
}

func TestComposePanelSubmit(t *testing.T) {
	theme := DefaultTheme()
	var submittedContent string
	var submittedType uint8
	callback := func(content string, waveType uint8, targetNodeID string) {
		submittedContent = content
		submittedType = waveType
	}

	panel := NewComposePanel(theme, callback)
	panel.Show()
	panel.SetContent("Test Wave Content")
	panel.Submit()

	if submittedContent != "Test Wave Content" {
		t.Errorf("Expected submitted content 'Test Wave Content', got '%s'", submittedContent)
	}
	if submittedType != 0x01 {
		t.Errorf("Expected wave type 0x01, got 0x%02x", submittedType)
	}
	if panel.Visible() {
		t.Error("Panel should be hidden after submit")
	}
}

func TestComposePanelEmptySubmit(t *testing.T) {
	theme := DefaultTheme()
	submitted := false
	callback := func(content string, waveType uint8, targetNodeID string) {
		submitted = true
	}

	panel := NewComposePanel(theme, callback)
	panel.Show()
	panel.Submit() // Submit with empty content.

	if submitted {
		t.Error("Should not submit empty Wave")
	}
}

func TestNewSettingsPanel(t *testing.T) {
	theme := DefaultTheme()
	var changedKey string
	callback := func(key, value string) {
		changedKey = key
	}

	panel := NewSettingsPanel(theme, callback)
	if panel == nil {
		t.Fatal("NewSettingsPanel returned nil")
	}

	// Verify default categories.
	cats := panel.Categories()
	if len(cats) != 4 {
		t.Errorf("Expected 4 categories, got %d", len(cats))
	}

	_ = changedKey // Used in real tests.
}

func TestSettingsPanelVisibility(t *testing.T) {
	theme := DefaultTheme()
	panel := NewSettingsPanel(theme, nil)

	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}

	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle()")
	}
}

func TestSettingsPanelGetSetting(t *testing.T) {
	theme := DefaultTheme()
	panel := NewSettingsPanel(theme, nil)

	// Test getting existing setting.
	dhtEnabled := panel.GetSetting("dht_enabled")
	if dhtEnabled != true {
		t.Errorf("Expected dht_enabled to be true, got %v", dhtEnabled)
	}

	// Test getting non-existent setting.
	nonExistent := panel.GetSetting("non_existent")
	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent setting, got %v", nonExistent)
	}
}

func TestSettingsPanelSetSetting(t *testing.T) {
	theme := DefaultTheme()
	var changedKey, changedValue string
	callback := func(key, value string) {
		changedKey = key
		changedValue = value
	}

	panel := NewSettingsPanel(theme, callback)

	// Test setting a toggle.
	panel.SetSetting("dht_enabled", false)
	if changedKey != "dht_enabled" {
		t.Errorf("Expected changed key 'dht_enabled', got '%s'", changedKey)
	}
	if changedValue != "false" {
		t.Errorf("Expected changed value 'false', got '%s'", changedValue)
	}

	// Verify the value was stored.
	dhtEnabled := panel.GetSetting("dht_enabled")
	if dhtEnabled != false {
		t.Errorf("Expected dht_enabled to be false, got %v", dhtEnabled)
	}
}

func TestSettingsPanelUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewSettingsPanel(theme, nil)

	// Update when not visible should not consume input.
	consumed := panel.Update()
	if consumed {
		t.Error("Update should not consume input when panel is hidden")
	}

	// Update when visible should consume input.
	panel.Show()
	consumed = panel.Update()
	if !consumed {
		t.Error("Update should consume input when panel is visible")
	}
}

func TestPanelPositionConstants(t *testing.T) {
	// Verify position constants are distinct.
	positions := []PanelPosition{
		PositionCenter,
		PositionTopLeft,
		PositionTopRight,
		PositionBottomLeft,
		PositionBottomRight,
		PositionLeft,
		PositionRight,
	}

	seen := make(map[PanelPosition]bool)
	for _, pos := range positions {
		if seen[pos] {
			t.Errorf("Duplicate position constant: %d", pos)
		}
		seen[pos] = true
	}
}

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		NodeID:      "node123",
		DisplayName: "Alice",
		Pseudonym:   "ShadowWalker",
		IsSpecter:   true,
		Resonance:   75.5,
	}

	if result.NodeID != "node123" {
		t.Error("NodeID mismatch")
	}
	if result.DisplayName != "Alice" {
		t.Error("DisplayName mismatch")
	}
	if result.Pseudonym != "ShadowWalker" {
		t.Error("Pseudonym mismatch")
	}
	if !result.IsSpecter {
		t.Error("IsSpecter should be true")
	}
	if result.Resonance != 75.5 {
		t.Error("Resonance mismatch")
	}
}

// Tests for PuzzlePanel.

func TestNewPuzzlePanel(t *testing.T) {
	theme := DefaultTheme()
	var submitted bool
	callback := func(puzzleType PuzzleType, difficulty uint8, duration time.Duration, seed string) {
		submitted = true
	}

	panel := NewPuzzlePanel(theme, callback)
	if panel == nil {
		t.Fatal("NewPuzzlePanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}

	// Test show/hide.
	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}

	// Test toggle.
	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle()")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should not be visible after second Toggle()")
	}

	_ = submitted
}

func TestPuzzlePanelPuzzleType(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzlePanel(theme, nil)

	// Default should be Fragment.
	if panel.GetPuzzleType() != PuzzleFragment {
		t.Errorf("Expected default puzzle type PuzzleFragment, got %d", panel.GetPuzzleType())
	}

	// Test setting.
	panel.SetPuzzleType(PuzzleMosaic)
	if panel.GetPuzzleType() != PuzzleMosaic {
		t.Errorf("Expected PuzzleMosaic, got %d", panel.GetPuzzleType())
	}

	panel.SetPuzzleType(PuzzleCascade)
	if panel.GetPuzzleType() != PuzzleCascade {
		t.Errorf("Expected PuzzleCascade, got %d", panel.GetPuzzleType())
	}

	// Invalid type should not change.
	panel.SetPuzzleType(0)
	if panel.GetPuzzleType() != PuzzleCascade {
		t.Error("Invalid type should not change current type")
	}

	panel.SetPuzzleType(100)
	if panel.GetPuzzleType() != PuzzleCascade {
		t.Error("Invalid type should not change current type")
	}
}

func TestPuzzlePanelDifficulty(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzlePanel(theme, nil)

	// Default should be 20.
	if panel.GetDifficulty() != DefaultPuzzleDifficulty {
		t.Errorf("Expected default difficulty %d, got %d", DefaultPuzzleDifficulty, panel.GetDifficulty())
	}

	// Test valid difficulty.
	panel.SetDifficulty(24)
	if panel.GetDifficulty() != 24 {
		t.Errorf("Expected difficulty 24, got %d", panel.GetDifficulty())
	}

	// Test bounds.
	panel.SetDifficulty(MinPuzzleDifficulty)
	if panel.GetDifficulty() != MinPuzzleDifficulty {
		t.Errorf("Expected min difficulty %d, got %d", MinPuzzleDifficulty, panel.GetDifficulty())
	}

	panel.SetDifficulty(MaxPuzzleDifficulty)
	if panel.GetDifficulty() != MaxPuzzleDifficulty {
		t.Errorf("Expected max difficulty %d, got %d", MaxPuzzleDifficulty, panel.GetDifficulty())
	}

	// Test out-of-bounds values.
	panel.SetDifficulty(MinPuzzleDifficulty - 1)
	if panel.GetDifficulty() != MaxPuzzleDifficulty {
		t.Error("Below-min difficulty should not change current value")
	}

	panel.SetDifficulty(MaxPuzzleDifficulty + 1)
	if panel.GetDifficulty() != MaxPuzzleDifficulty {
		t.Error("Above-max difficulty should not change current value")
	}
}

func TestPuzzlePanelDuration(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzlePanel(theme, nil)

	// Default should be 30 minutes (index 1).
	if panel.GetDuration() != PuzzleDuration30Min {
		t.Errorf("Expected default duration 30m, got %v", panel.GetDuration())
	}

	// Test setting by index.
	panel.SetDurationIndex(0)
	if panel.GetDuration() != PuzzleDuration15Min {
		t.Errorf("Expected 15m, got %v", panel.GetDuration())
	}

	panel.SetDurationIndex(2)
	if panel.GetDuration() != PuzzleDuration60Min {
		t.Errorf("Expected 60m, got %v", panel.GetDuration())
	}

	// Invalid index should not change.
	panel.SetDurationIndex(-1)
	if panel.GetDuration() != PuzzleDuration60Min {
		t.Error("Invalid index should not change duration")
	}

	panel.SetDurationIndex(100)
	if panel.GetDuration() != PuzzleDuration60Min {
		t.Error("Invalid index should not change duration")
	}
}

func TestPuzzlePanelSeed(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzlePanel(theme, nil)

	// Default should be empty.
	if panel.GetSeed() != "" {
		t.Errorf("Expected empty seed, got '%s'", panel.GetSeed())
	}

	// Test setting.
	panel.SetSeed("test-seed-123")
	if panel.GetSeed() != "test-seed-123" {
		t.Errorf("Expected 'test-seed-123', got '%s'", panel.GetSeed())
	}

	// Test length limit (64 chars max).
	longSeed := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	panel.SetSeed(longSeed)
	if len(panel.GetSeed()) != 64 {
		t.Errorf("Seed should be truncated to 64 chars, got %d", len(panel.GetSeed()))
	}
}

func TestPuzzlePanelSubmit(t *testing.T) {
	theme := DefaultTheme()
	var submittedType PuzzleType
	var submittedDifficulty uint8
	var submittedDuration time.Duration
	var submittedSeed string

	callback := func(puzzleType PuzzleType, difficulty uint8, duration time.Duration, seed string) {
		submittedType = puzzleType
		submittedDifficulty = difficulty
		submittedDuration = duration
		submittedSeed = seed
	}

	panel := NewPuzzlePanel(theme, callback)
	panel.Show()
	panel.SetPuzzleType(PuzzleMosaic)
	panel.SetDifficulty(22)
	panel.SetDurationIndex(2)
	panel.SetSeed("my-seed")
	panel.Submit()

	if submittedType != PuzzleMosaic {
		t.Errorf("Expected PuzzleMosaic, got %d", submittedType)
	}
	if submittedDifficulty != 22 {
		t.Errorf("Expected difficulty 22, got %d", submittedDifficulty)
	}
	if submittedDuration != PuzzleDuration60Min {
		t.Errorf("Expected 60m duration, got %v", submittedDuration)
	}
	if submittedSeed != "my-seed" {
		t.Errorf("Expected seed 'my-seed', got '%s'", submittedSeed)
	}

	// Panel should be hidden after submit.
	if panel.Visible() {
		t.Error("Panel should be hidden after submit")
	}

	// Seed should be cleared after submit.
	if panel.GetSeed() != "" {
		t.Error("Seed should be cleared after submit")
	}
}

func TestPuzzlePanelUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzlePanel(theme, nil)

	// Update when not visible should not consume input.
	consumed := panel.Update()
	if consumed {
		t.Error("Update should not consume input when panel is hidden")
	}

	// Update when visible should consume input.
	panel.Show()
	consumed = panel.Update()
	if !consumed {
		t.Error("Update should consume input when panel is visible")
	}
}

func TestPuzzleTypeConstants(t *testing.T) {
	// Verify puzzle type constants match expected values.
	if PuzzleFragment != 1 {
		t.Errorf("PuzzleFragment should be 1, got %d", PuzzleFragment)
	}
	if PuzzleMosaic != 2 {
		t.Errorf("PuzzleMosaic should be 2, got %d", PuzzleMosaic)
	}
	if PuzzleCascade != 3 {
		t.Errorf("PuzzleCascade should be 3, got %d", PuzzleCascade)
	}
}

func TestPuzzleDurationConstants(t *testing.T) {
	// Verify duration constants.
	if PuzzleDuration15Min != 15*time.Minute {
		t.Error("PuzzleDuration15Min incorrect")
	}
	if PuzzleDuration30Min != 30*time.Minute {
		t.Error("PuzzleDuration30Min incorrect")
	}
	if PuzzleDuration60Min != 60*time.Minute {
		t.Error("PuzzleDuration60Min incorrect")
	}
}

// Tests for PuzzleSolverPanel.

func TestNewPuzzleSolverPanel(t *testing.T) {
	theme := DefaultTheme()
	var submitted bool
	callback := func(puzzleID [32]byte, solution string) (bool, string) {
		submitted = true
		return true, "Success!"
	}

	panel := NewPuzzleSolverPanel(theme, callback)
	if panel == nil {
		t.Fatal("NewPuzzleSolverPanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}

	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle()")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should not be visible after second Toggle()")
	}

	_ = submitted
}

func TestPuzzleSolverSetPuzzle(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzleSolverPanel(theme, nil)

	var puzzleID [32]byte
	copy(puzzleID[:], "test-puzzle-id-12345678901234")
	expiresAt := time.Now().Add(30 * time.Minute)

	panel.SetPuzzle(puzzleID, PuzzleMosaic, 22, expiresAt, 5)

	if panel.GetPuzzleID() != puzzleID {
		t.Error("Puzzle ID not set correctly")
	}
}

func TestPuzzleSolverSolution(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzleSolverPanel(theme, nil)

	// Default should be empty.
	if panel.GetSolution() != "" {
		t.Errorf("Expected empty solution, got '%s'", panel.GetSolution())
	}

	// Test setting.
	panel.SetSolution("test-solution-123")
	if panel.GetSolution() != "test-solution-123" {
		t.Errorf("Expected 'test-solution-123', got '%s'", panel.GetSolution())
	}

	// Test length limit (256 chars max).
	longSolution := make([]byte, 300)
	for i := range longSolution {
		longSolution[i] = 'x'
	}
	panel.SetSolution(string(longSolution))
	if len(panel.GetSolution()) != 256 {
		t.Errorf("Solution should be truncated to 256 chars, got %d", len(panel.GetSolution()))
	}
}

func TestPuzzleSolverSubmitSuccess(t *testing.T) {
	theme := DefaultTheme()
	var submittedID [32]byte
	var submittedSolution string

	callback := func(puzzleID [32]byte, solution string) (bool, string) {
		submittedID = puzzleID
		submittedSolution = solution
		return true, "Correct!"
	}

	panel := NewPuzzleSolverPanel(theme, callback)
	panel.Show()

	var puzzleID [32]byte
	copy(puzzleID[:], "test-puzzle-id-12345678901234")
	panel.SetPuzzle(puzzleID, PuzzleFragment, 20, time.Now().Add(30*time.Minute), 3)
	panel.SetSolution("my-answer")
	panel.Submit()

	if submittedID != puzzleID {
		t.Error("Puzzle ID not passed to callback")
	}
	if submittedSolution != "my-answer" {
		t.Errorf("Expected solution 'my-answer', got '%s'", submittedSolution)
	}
	if panel.GetSuccessMessage() != "Correct!" {
		t.Errorf("Expected success message 'Correct!', got '%s'", panel.GetSuccessMessage())
	}
	if panel.GetSolution() != "" {
		t.Error("Solution should be cleared after submit")
	}
}

func TestPuzzleSolverSubmitFailure(t *testing.T) {
	theme := DefaultTheme()
	callback := func(puzzleID [32]byte, solution string) (bool, string) {
		return false, "Wrong answer!"
	}

	panel := NewPuzzleSolverPanel(theme, callback)
	panel.Show()

	var puzzleID [32]byte
	panel.SetPuzzle(puzzleID, PuzzleFragment, 20, time.Now().Add(30*time.Minute), 3)
	panel.SetSolution("wrong-answer")
	panel.Submit()

	if panel.GetErrorMessage() != "Wrong answer!" {
		t.Errorf("Expected error message 'Wrong answer!', got '%s'", panel.GetErrorMessage())
	}
}

func TestPuzzleSolverSubmitEmpty(t *testing.T) {
	theme := DefaultTheme()
	submitted := false
	callback := func(puzzleID [32]byte, solution string) (bool, string) {
		submitted = true
		return true, ""
	}

	panel := NewPuzzleSolverPanel(theme, callback)
	panel.Show()

	var puzzleID [32]byte
	panel.SetPuzzle(puzzleID, PuzzleFragment, 20, time.Now().Add(30*time.Minute), 3)
	panel.Submit() // Submit with empty solution.

	if submitted {
		t.Error("Should not submit empty solution")
	}
	if panel.GetErrorMessage() != "Solution cannot be empty" {
		t.Errorf("Expected empty solution error, got '%s'", panel.GetErrorMessage())
	}
}

func TestPuzzleSolverSubmitExpired(t *testing.T) {
	theme := DefaultTheme()
	submitted := false
	callback := func(puzzleID [32]byte, solution string) (bool, string) {
		submitted = true
		return true, ""
	}

	panel := NewPuzzleSolverPanel(theme, callback)
	panel.Show()

	var puzzleID [32]byte
	panel.SetPuzzle(puzzleID, PuzzleFragment, 20, time.Now().Add(-1*time.Minute), 3) // Already expired.
	panel.SetSolution("my-answer")
	panel.Submit()

	if submitted {
		t.Error("Should not submit to expired puzzle")
	}
	if panel.GetErrorMessage() != "Puzzle has expired" {
		t.Errorf("Expected expired puzzle error, got '%s'", panel.GetErrorMessage())
	}
}

func TestPuzzleSolverUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPuzzleSolverPanel(theme, nil)

	consumed := panel.Update()
	if consumed {
		t.Error("Update should not consume input when hidden")
	}

	panel.Show()
	consumed = panel.Update()
	if !consumed {
		t.Error("Update should consume input when visible")
	}
}
