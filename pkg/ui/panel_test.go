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

// =============================================================================
// HuntTrackerPanel Tests
// =============================================================================

func TestNewHuntTrackerPanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	if panel == nil {
		t.Fatal("NewHuntTrackerPanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should start hidden")
	}
	if panel.GetSelectedTab() != 0 {
		t.Errorf("Expected initial tab 0, got %d", panel.GetSelectedTab())
	}
	if panel.GetHunt() != nil {
		t.Error("Hunt should be nil initially")
	}
}

func TestHuntTrackerPanelVisibility(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	// Test Show
	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show")
	}

	// Test Hide
	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should be hidden after Hide")
	}

	// Test Toggle
	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle from hidden")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should be hidden after Toggle from visible")
	}
}

func TestHuntTrackerSetHunt(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	huntID := [32]byte{1, 2, 3, 4, 5, 6, 7, 8}
	hunt := &HuntInfo{
		ID:            huntID,
		Theme:         "Shadow Quest",
		ExpiresAt:     time.Now().Add(60 * time.Minute),
		FragmentCount: 5,
		ClaimedCount:  2,
		Fragments: []FragmentInfo{
			{Index: 0, Claimed: true, ClaimedByMe: false, Clues: []string{"Look east"}, LocationHint: "Near the river"},
			{Index: 1, Claimed: false, ClaimedByMe: false, Clues: []string{"Under shadow"}, LocationHint: "In darkness"},
			{Index: 2, Claimed: true, ClaimedByMe: true, Clues: []string{"By the light"}, LocationHint: "At dawn"},
		},
		Leaderboard: []LeaderboardEntry{
			{Pseudonym: "ShadowWalker", Claims: 3, IsMe: false},
			{Pseudonym: "MySpecter", Claims: 1, IsMe: true},
		},
		SelectedFragment: -1,
		UserClaims:       1,
	}

	panel.SetHunt(hunt)
	retrievedHunt := panel.GetHunt()
	if retrievedHunt == nil {
		t.Fatal("Hunt should not be nil after SetHunt")
	}
	if retrievedHunt.Theme != "Shadow Quest" {
		t.Errorf("Expected theme 'Shadow Quest', got '%s'", retrievedHunt.Theme)
	}
	if len(retrievedHunt.Fragments) != 3 {
		t.Errorf("Expected 3 fragments, got %d", len(retrievedHunt.Fragments))
	}
}

func TestHuntTrackerTabSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	// Test valid tabs
	panel.SetSelectedTab(1) // Clues
	if panel.GetSelectedTab() != 1 {
		t.Errorf("Expected tab 1, got %d", panel.GetSelectedTab())
	}

	panel.SetSelectedTab(2) // Leaderboard
	if panel.GetSelectedTab() != 2 {
		t.Errorf("Expected tab 2, got %d", panel.GetSelectedTab())
	}

	panel.SetSelectedTab(0) // Fragments
	if panel.GetSelectedTab() != 0 {
		t.Errorf("Expected tab 0, got %d", panel.GetSelectedTab())
	}

	// Test invalid tabs (should be ignored)
	panel.SetSelectedTab(1) // set to valid first
	panel.SetSelectedTab(-1)
	if panel.GetSelectedTab() != 1 {
		t.Errorf("Tab should remain 1 after invalid -1, got %d", panel.GetSelectedTab())
	}

	panel.SetSelectedTab(3) // out of range
	if panel.GetSelectedTab() != 1 {
		t.Errorf("Tab should remain 1 after invalid 3, got %d", panel.GetSelectedTab())
	}
}

func TestHuntTrackerFragmentSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	// Without hunt, selection should fail gracefully
	panel.SetSelectedFragment(0)
	if panel.GetSelectedFragment() != -1 {
		t.Errorf("Fragment selection should return -1 without hunt, got %d", panel.GetSelectedFragment())
	}

	// Set up a hunt with fragments
	hunt := &HuntInfo{
		ID:               [32]byte{1, 2, 3},
		SelectedFragment: -1,
		Fragments: []FragmentInfo{
			{Index: 0, Claimed: false},
			{Index: 1, Claimed: true},
			{Index: 2, Claimed: false},
		},
	}
	panel.SetHunt(hunt)

	// Test valid selection
	panel.SetSelectedFragment(1)
	if panel.GetSelectedFragment() != 1 {
		t.Errorf("Expected selected fragment 1, got %d", panel.GetSelectedFragment())
	}

	// Test selection at boundary
	panel.SetSelectedFragment(2)
	if panel.GetSelectedFragment() != 2 {
		t.Errorf("Expected selected fragment 2, got %d", panel.GetSelectedFragment())
	}

	// Test invalid selection (out of range)
	panel.SetSelectedFragment(5)
	if panel.GetSelectedFragment() != 2 {
		t.Errorf("Selection should remain 2 after invalid index 5, got %d", panel.GetSelectedFragment())
	}
}

func TestHuntTrackerFragmentSelectCallback(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	var callbackHuntID [32]byte
	var callbackFragmentIdx int
	callbackCalled := false

	panel.SetOnFragmentSelect(func(huntID [32]byte, fragmentIndex int) {
		callbackCalled = true
		callbackHuntID = huntID
		callbackFragmentIdx = fragmentIndex
	})

	expectedHuntID := [32]byte{9, 8, 7, 6, 5}
	hunt := &HuntInfo{
		ID:               expectedHuntID,
		SelectedFragment: -1,
		Fragments: []FragmentInfo{
			{Index: 0, Claimed: false},
			{Index: 1, Claimed: false},
		},
	}
	panel.SetHunt(hunt)

	// Select a fragment using the callback-triggering method
	panel.SelectFragment(1)

	if !callbackCalled {
		t.Error("Fragment select callback should have been called")
	}
	if callbackHuntID != expectedHuntID {
		t.Errorf("Callback received wrong hunt ID")
	}
	if callbackFragmentIdx != 1 {
		t.Errorf("Expected callback fragment index 1, got %d", callbackFragmentIdx)
	}
}

func TestHuntTrackerClaimCallback(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	var callbackHuntID [32]byte
	var callbackFragmentIdx int
	callbackCalled := false

	panel.SetOnClaimAttempt(func(huntID [32]byte, fragmentIndex int) {
		callbackCalled = true
		callbackHuntID = huntID
		callbackFragmentIdx = fragmentIndex
	})

	expectedHuntID := [32]byte{4, 3, 2, 1}
	hunt := &HuntInfo{
		ID:               expectedHuntID,
		SelectedFragment: 0,
		Fragments: []FragmentInfo{
			{Index: 0, Claimed: false},
		},
	}
	panel.SetHunt(hunt)

	// Attempt claim
	panel.AttemptClaim()

	if !callbackCalled {
		t.Error("Claim callback should have been called")
	}
	if callbackHuntID != expectedHuntID {
		t.Errorf("Callback received wrong hunt ID")
	}
	if callbackFragmentIdx != 0 {
		t.Errorf("Expected callback fragment index 0, got %d", callbackFragmentIdx)
	}
}

func TestHuntTrackerClaimWithoutSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	callbackCalled := false
	panel.SetOnClaimAttempt(func(huntID [32]byte, fragmentIndex int) {
		callbackCalled = true
	})

	hunt := &HuntInfo{
		ID:               [32]byte{1},
		SelectedFragment: -1, // No fragment selected
		Fragments: []FragmentInfo{
			{Index: 0, Claimed: false},
		},
	}
	panel.SetHunt(hunt)

	// Attempt claim without selection
	panel.AttemptClaim()

	if callbackCalled {
		t.Error("Claim callback should not be called without a selected fragment")
	}
}

func TestHuntTrackerError(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	panel.SetError("Fragment already claimed")
	// Error is set internally; stub doesn't expose getter but full impl would display it
}

func TestHuntTrackerUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	// Update when hidden should not consume input
	consumed := panel.Update()
	if consumed {
		t.Error("Update should not consume input when hidden")
	}

	// Update when visible should consume input
	panel.Show()
	consumed = panel.Update()
	if !consumed {
		t.Error("Update should consume input when visible")
	}
}

func TestHuntTrackerNilHuntOperations(t *testing.T) {
	theme := DefaultTheme()
	panel := NewHuntTrackerPanel(theme)

	// These operations should handle nil hunt gracefully
	panel.SelectFragment(0)      // Should not panic
	panel.AttemptClaim()         // Should not panic
	panel.SetSelectedFragment(0) // Should not panic

	if panel.GetSelectedFragment() != -1 {
		t.Error("GetSelectedFragment should return -1 when hunt is nil")
	}
}

// =============================================================================
// TerritoryOverviewPanel Tests
// =============================================================================

func TestNewTerritoryOverviewPanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	if panel == nil {
		t.Fatal("NewTerritoryOverviewPanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should start hidden")
	}
	if panel.GetSelectedIndex() != -1 {
		t.Errorf("Expected initial selection -1, got %d", panel.GetSelectedIndex())
	}
	if panel.GetTerritoryCount() != 0 {
		t.Errorf("Expected 0 territories, got %d", panel.GetTerritoryCount())
	}
}

func TestTerritoryOverviewPanelVisibility(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should be hidden after Hide")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle from hidden")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should be hidden after Toggle from visible")
	}
}

func TestTerritoryOverviewSetTerritories(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	territories := []TerritoryInfo{
		{ID: "territory-alpha", IsControlled: true, Influence: 50.5, MemberCount: 15},
		{ID: "territory-beta", IsContested: true, Influence: 30.0, MemberCount: 8},
		{ID: "territory-gamma", Influence: 10.0, MemberCount: 5}, // Neutral.
	}

	panel.SetTerritories(territories)
	if panel.GetTerritoryCount() != 3 {
		t.Errorf("Expected 3 territories, got %d", panel.GetTerritoryCount())
	}
}

func TestTerritoryOverviewSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	territories := []TerritoryInfo{
		{ID: "t1"},
		{ID: "t2"},
		{ID: "t3"},
	}
	panel.SetTerritories(territories)

	// Select by index.
	panel.SetSelectedIndex(1)
	if panel.GetSelectedIndex() != 1 {
		t.Errorf("Expected selected index 1, got %d", panel.GetSelectedIndex())
	}
	if panel.GetSelectedTerritory() != "t2" {
		t.Errorf("Expected selected territory 't2', got '%s'", panel.GetSelectedTerritory())
	}

	// Invalid index should be ignored.
	panel.SetSelectedIndex(10)
	if panel.GetSelectedIndex() != 1 {
		t.Errorf("Selection should remain 1 after invalid index, got %d", panel.GetSelectedIndex())
	}
}

func TestTerritoryOverviewSelectByID(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	var callbackID string
	panel.SetOnTerritorySelect(func(territoryID string) {
		callbackID = territoryID
	})

	territories := []TerritoryInfo{
		{ID: "alpha"},
		{ID: "beta"},
		{ID: "gamma"},
	}
	panel.SetTerritories(territories)

	// Select by ID.
	found := panel.SelectTerritory("beta")
	if !found {
		t.Error("SelectTerritory should return true for existing territory")
	}
	if panel.GetSelectedTerritory() != "beta" {
		t.Errorf("Expected 'beta', got '%s'", panel.GetSelectedTerritory())
	}
	if callbackID != "beta" {
		t.Errorf("Callback should receive 'beta', got '%s'", callbackID)
	}

	// Non-existent ID.
	found = panel.SelectTerritory("nonexistent")
	if found {
		t.Error("SelectTerritory should return false for nonexistent territory")
	}
}

func TestTerritoryOverviewNavigate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	var navX, navY float64
	navCalled := false
	panel.SetOnNavigate(func(cx, cy float64) {
		navCalled = true
		navX = cx
		navY = cy
	})

	territories := []TerritoryInfo{
		{ID: "loc", CentroidX: 100.5, CentroidY: 200.5},
	}
	panel.SetTerritories(territories)
	panel.SetSelectedIndex(0)

	panel.NavigateToSelected()

	if !navCalled {
		t.Error("Navigate callback should have been called")
	}
	if navX != 100.5 {
		t.Errorf("Expected navX 100.5, got %f", navX)
	}
	if navY != 200.5 {
		t.Errorf("Expected navY 200.5, got %f", navY)
	}
}

func TestTerritoryOverviewCycleTime(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	cycleEnd := time.Now().Add(3 * 24 * time.Hour)
	panel.SetCycleEndTime(cycleEnd)

	// Stub exposes GetCycleEndTime.
	if panel.GetCycleEndTime() != cycleEnd {
		t.Error("Cycle end time should match what was set")
	}
}

func TestTerritoryOverviewMyInfluence(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	panel.SetMyInfluence(75.5)
	if panel.GetMyInfluence() != 75.5 {
		t.Errorf("Expected influence 75.5, got %f", panel.GetMyInfluence())
	}
}

func TestTerritoryOverviewUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

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

func TestTerritoryOverviewNoSelectionGetters(t *testing.T) {
	theme := DefaultTheme()
	panel := NewTerritoryOverviewPanel(theme)

	if panel.GetSelectedTerritory() != "" {
		t.Error("GetSelectedTerritory should return empty string when nothing selected")
	}

	// NavigateToSelected with no selection should not panic.
	panel.NavigateToSelected() // Should be a no-op.
}

// =============================================================================
// OraclePoolPanel Tests
// =============================================================================

func TestNewOraclePoolPanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	if panel == nil {
		t.Fatal("NewOraclePoolPanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should start hidden")
	}
	if panel.GetMode() != OraclePoolModeView {
		t.Errorf("Expected initial mode OraclePoolModeView, got %d", panel.GetMode())
	}
	if panel.GetPool() != nil {
		t.Error("Pool should be nil initially")
	}
}

func TestOraclePoolPanelVisibility(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should be hidden after Hide")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle from hidden")
	}
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should be hidden after Toggle from visible")
	}
}

func TestOraclePoolPanelSetPool(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	pool := &OraclePoolInfo{
		PoolID:          [32]byte{1, 2, 3},
		Question:        "Will message count exceed 1000?",
		State:           OraclePoolStatePending,
		PredictionCount: 5,
	}

	panel.SetPool(pool)
	retrieved := panel.GetPool()
	if retrieved == nil {
		t.Fatal("Pool should not be nil after SetPool")
	}
	if retrieved.Question != "Will message count exceed 1000?" {
		t.Errorf("Question mismatch")
	}
	if retrieved.PredictionCount != 5 {
		t.Errorf("Expected 5 predictions, got %d", retrieved.PredictionCount)
	}
}

func TestOraclePoolPanelModes(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	panel.SetMode(OraclePoolModeCreate)
	if panel.GetMode() != OraclePoolModeCreate {
		t.Errorf("Expected OraclePoolModeCreate, got %d", panel.GetMode())
	}

	panel.SetMode(OraclePoolModePredict)
	if panel.GetMode() != OraclePoolModePredict {
		t.Errorf("Expected OraclePoolModePredict, got %d", panel.GetMode())
	}

	panel.SetMode(OraclePoolModeView)
	if panel.GetMode() != OraclePoolModeView {
		t.Errorf("Expected OraclePoolModeView, got %d", panel.GetMode())
	}
}

func TestOraclePoolPanelPredictionText(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	panel.SetPredictionText("My prediction")
	if panel.GetPredictionText() != "My prediction" {
		t.Errorf("Expected 'My prediction', got '%s'", panel.GetPredictionText())
	}
}

func TestOraclePoolPanelSubmitPrediction(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	var callbackPoolID [32]byte
	var callbackPrediction string
	callbackCalled := false

	panel.SetOnPredict(func(poolID [32]byte, prediction string) {
		callbackCalled = true
		callbackPoolID = poolID
		callbackPrediction = prediction
	})

	expectedID := [32]byte{5, 6, 7}
	pool := &OraclePoolInfo{
		PoolID: expectedID,
		State:  OraclePoolStatePending,
	}
	panel.SetPool(pool)
	panel.SetMode(OraclePoolModePredict)
	panel.SetPredictionText("1500")

	panel.SubmitPrediction()

	if !callbackCalled {
		t.Error("Predict callback should have been called")
	}
	if callbackPoolID != expectedID {
		t.Error("Pool ID mismatch in callback")
	}
	if callbackPrediction != "1500" {
		t.Errorf("Expected prediction '1500', got '%s'", callbackPrediction)
	}
	// Should switch back to view mode.
	if panel.GetMode() != OraclePoolModeView {
		t.Error("Should switch to view mode after submission")
	}
}

func TestOraclePoolPanelSubmitEmpty(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	callbackCalled := false
	panel.SetOnPredict(func(poolID [32]byte, prediction string) {
		callbackCalled = true
	})

	pool := &OraclePoolInfo{PoolID: [32]byte{1}}
	panel.SetPool(pool)
	panel.SetMode(OraclePoolModePredict)
	panel.SetPredictionText("")

	panel.SubmitPrediction()

	if callbackCalled {
		t.Error("Should not submit empty prediction")
	}
	if panel.GetErrorMessage() != "Prediction cannot be empty" {
		t.Errorf("Expected empty prediction error, got '%s'", panel.GetErrorMessage())
	}
}

func TestOraclePoolPanelSubmitNoPool(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	callbackCalled := false
	panel.SetOnPredict(func(poolID [32]byte, prediction string) {
		callbackCalled = true
	})

	panel.SetMode(OraclePoolModePredict)
	panel.SetPredictionText("some prediction")

	panel.SubmitPrediction()

	if callbackCalled {
		t.Error("Should not submit without pool")
	}
	if panel.GetErrorMessage() != "No pool selected" {
		t.Errorf("Expected no pool error, got '%s'", panel.GetErrorMessage())
	}
}

func TestOraclePoolPanelReveal(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	var callbackPoolID [32]byte
	callbackCalled := false

	panel.SetOnReveal(func(poolID [32]byte) {
		callbackCalled = true
		callbackPoolID = poolID
	})

	expectedID := [32]byte{10, 11, 12}
	pool := &OraclePoolInfo{
		PoolID:      expectedID,
		State:       OraclePoolStateRevealing,
		MyCommitted: true,
		MyRevealed:  false,
	}
	panel.SetPool(pool)

	panel.RevealPrediction()

	if !callbackCalled {
		t.Error("Reveal callback should have been called")
	}
	if callbackPoolID != expectedID {
		t.Error("Pool ID mismatch in reveal callback")
	}
}

func TestOraclePoolPanelSetError(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	panel.SetError("Test error message")
	if panel.GetErrorMessage() != "Test error message" {
		t.Errorf("Error message mismatch")
	}
}

func TestOraclePoolPanelUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

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

func TestOraclePoolStates(t *testing.T) {
	// Verify state constants.
	if OraclePoolStatePending != 0 {
		t.Errorf("OraclePoolStatePending should be 0, got %d", OraclePoolStatePending)
	}
	if OraclePoolStateRevealing != 1 {
		t.Errorf("OraclePoolStateRevealing should be 1, got %d", OraclePoolStateRevealing)
	}
	if OraclePoolStateResolved != 2 {
		t.Errorf("OraclePoolStateResolved should be 2, got %d", OraclePoolStateResolved)
	}
	if OraclePoolStateExpired != 3 {
		t.Errorf("OraclePoolStateExpired should be 3, got %d", OraclePoolStateExpired)
	}
}

func TestOraclePoolPanelRevealNoPool(t *testing.T) {
	theme := DefaultTheme()
	panel := NewOraclePoolPanel(theme)

	callbackCalled := false
	panel.SetOnReveal(func(poolID [32]byte) {
		callbackCalled = true
	})

	// RevealPrediction with no pool should not panic or call callback.
	panel.RevealPrediction()

	if callbackCalled {
		t.Error("Reveal callback should not be called without pool")
	}
}

// ============================================================================
// ForgePanel Tests
// ============================================================================

func TestNewForgePanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	if panel == nil {
		t.Fatal("NewForgePanel returned nil")
	}
	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}
	if panel.GetMode() != ForgeModeView {
		t.Errorf("Expected ForgeModeView, got %d", panel.GetMode())
	}
}

func TestForgePanelVisibility(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	panel.Show()
	if !panel.Visible() {
		t.Error("Panel should be visible after Show")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should be hidden after Hide")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after Toggle from hidden")
	}

	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should be hidden after Toggle from visible")
	}
}

func TestForgePanelSetForge(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	forge := &ForgeInfo{
		ForgeID:   [32]byte{1, 2, 3},
		Type:      ForgeTypeSigilArt,
		Prompt:    "Create something beautiful",
		Duration:  30 * time.Minute,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Minute),
		IsActive:  true,
		Entries: []ForgeEntryInfo{
			{
				EntryID:        [32]byte{10},
				SpecterKey:     [32]byte{20},
				SpecterName:    "TestSpecter",
				Preview:        "My amazing entry",
				Amplifications: 5,
			},
		},
	}

	panel.SetForge(forge)

	retrieved := panel.GetForge()
	if retrieved == nil {
		t.Fatal("GetForge returned nil")
	}
	if retrieved.Prompt != "Create something beautiful" {
		t.Error("Prompt mismatch")
	}
	if len(retrieved.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(retrieved.Entries))
	}
}

func TestForgePanelModes(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	modes := []ForgePanelMode{
		ForgeModeView,
		ForgeModeCreate,
		ForgeModeSubmit,
		ForgeModeEntries,
	}

	for _, mode := range modes {
		panel.SetMode(mode)
		if panel.GetMode() != mode {
			t.Errorf("Mode mismatch: expected %d, got %d", mode, panel.GetMode())
		}
	}
}

func TestForgePanelEntryNavigation(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	forge := &ForgeInfo{
		ForgeID:  [32]byte{1},
		IsActive: true,
		Entries: []ForgeEntryInfo{
			{EntryID: [32]byte{1}, SpecterName: "Entry1"},
			{EntryID: [32]byte{2}, SpecterName: "Entry2"},
			{EntryID: [32]byte{3}, SpecterName: "Entry3"},
		},
	}
	panel.SetForge(forge)

	if panel.GetSelectedEntry() != 0 {
		t.Error("Initial selection should be 0")
	}

	panel.SetSelectedEntry(2)
	if panel.GetSelectedEntry() != 2 {
		t.Errorf("Expected selection 2, got %d", panel.GetSelectedEntry())
	}
}

func TestForgePanelTextInput(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	panel.SetEntryText("My creative submission")
	if panel.GetEntryText() != "My creative submission" {
		t.Error("Entry text mismatch")
	}

	panel.SetPromptText("Create a sigil representing hope")
	if panel.GetPromptText() != "Create a sigil representing hope" {
		t.Error("Prompt text mismatch")
	}
}

func TestForgePanelTypeSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	types := []ForgeType{
		ForgeTypeSigilArt,
		ForgeTypeMicroFic,
		ForgeTypeRemixChain,
	}

	for _, ft := range types {
		panel.SetSelectedType(ft)
		if panel.GetSelectedType() != ft {
			t.Errorf("Type mismatch: expected %d, got %d", ft, panel.GetSelectedType())
		}
	}
}

func TestForgePanelDurationChoice(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	if panel.GetDurationChoice() != 0 {
		t.Error("Initial duration choice should be 0")
	}

	panel.SetDurationChoice(1)
	if panel.GetDurationChoice() != 1 {
		t.Errorf("Expected duration choice 1, got %d", panel.GetDurationChoice())
	}
}

func TestForgePanelCallbacks(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	var createCalled bool
	var createType ForgeType
	var createPrompt string
	var createDuration time.Duration

	panel.SetOnCreate(func(ft ForgeType, prompt string, duration time.Duration) {
		createCalled = true
		createType = ft
		createPrompt = prompt
		createDuration = duration
	})

	panel.SetSelectedType(ForgeTypeMicroFic)
	panel.SetPromptText("Write a story about anonymity")
	panel.SetDurationChoice(1) // 60 minutes

	panel.TriggerCreate()

	if !createCalled {
		t.Fatal("Create callback not called")
	}
	if createType != ForgeTypeMicroFic {
		t.Errorf("Expected MicroFic type, got %d", createType)
	}
	if createPrompt != "Write a story about anonymity" {
		t.Error("Prompt mismatch in callback")
	}
	if createDuration != 60*time.Minute {
		t.Errorf("Expected 60m duration, got %v", createDuration)
	}
}

func TestForgePanelSubmitCallback(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	var submitCalled bool
	var submitForgeID [32]byte
	var submitContent string

	panel.SetOnSubmit(func(forgeID [32]byte, content string) {
		submitCalled = true
		submitForgeID = forgeID
		submitContent = content
	})

	forge := &ForgeInfo{
		ForgeID:  [32]byte{5, 6, 7},
		IsActive: true,
	}
	panel.SetForge(forge)
	panel.SetEntryText("My creative submission content")

	panel.TriggerSubmit()

	if !submitCalled {
		t.Fatal("Submit callback not called")
	}
	if submitForgeID != [32]byte{5, 6, 7} {
		t.Error("Forge ID mismatch in submit callback")
	}
	if submitContent != "My creative submission content" {
		t.Error("Content mismatch in submit callback")
	}
}

func TestForgePanelAmplifyCallback(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	var amplifyCalled bool
	var amplifyForgeID [32]byte
	var amplifyEntryID [32]byte

	panel.SetOnAmplify(func(forgeID, entryID [32]byte) {
		amplifyCalled = true
		amplifyForgeID = forgeID
		amplifyEntryID = entryID
	})

	forge := &ForgeInfo{
		ForgeID:  [32]byte{8, 9, 10},
		IsActive: true,
		Entries: []ForgeEntryInfo{
			{EntryID: [32]byte{100}, SpecterName: "Entry1"},
			{EntryID: [32]byte{101}, SpecterName: "Entry2"},
		},
	}
	panel.SetForge(forge)
	panel.SetSelectedEntry(1) // Select second entry

	panel.TriggerAmplify()

	if !amplifyCalled {
		t.Fatal("Amplify callback not called")
	}
	if amplifyForgeID != [32]byte{8, 9, 10} {
		t.Error("Forge ID mismatch in amplify callback")
	}
	if amplifyEntryID != [32]byte{101} {
		t.Error("Entry ID mismatch in amplify callback")
	}
}

func TestForgePanelError(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	panel.SetError("Test error message")
	if panel.GetErrorMessage() != "Test error message" {
		t.Error("Error message mismatch")
	}

	// Hide should clear error.
	panel.Hide()
	if panel.GetErrorMessage() != "" {
		t.Error("Error should be cleared after hide")
	}
}

func TestForgePanelUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	// Update should not panic.
	panel.Update()
	panel.Show()
	panel.Update()
}

func TestForgePanelConcurrency(t *testing.T) {
	theme := DefaultTheme()
	panel := NewForgePanel(theme)

	done := make(chan bool, 4)

	// Writer goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			panel.SetForge(&ForgeInfo{
				ForgeID: [32]byte{byte(i)},
			})
			panel.SetMode(ForgePanelMode(i % 4))
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			_ = panel.GetForge()
			_ = panel.GetMode()
			_ = panel.Visible()
		}
		done <- true
	}()

	// Toggle goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			panel.Toggle()
		}
		done <- true
	}()

	// Update goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			panel.Update()
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}

func TestForgeTypes(t *testing.T) {
	// Verify type constants.
	if ForgeTypeSigilArt != 0 {
		t.Errorf("ForgeTypeSigilArt should be 0, got %d", ForgeTypeSigilArt)
	}
	if ForgeTypeMicroFic != 1 {
		t.Errorf("ForgeTypeMicroFic should be 1, got %d", ForgeTypeMicroFic)
	}
	if ForgeTypeRemixChain != 2 {
		t.Errorf("ForgeTypeRemixChain should be 2, got %d", ForgeTypeRemixChain)
	}
}

func TestForgePanelModeConstants(t *testing.T) {
	// Verify mode constants.
	if ForgeModeView != 0 {
		t.Errorf("ForgeModeView should be 0, got %d", ForgeModeView)
	}
	if ForgeModeCreate != 1 {
		t.Errorf("ForgeModeCreate should be 1, got %d", ForgeModeCreate)
	}
	if ForgeModeSubmit != 2 {
		t.Errorf("ForgeModeSubmit should be 2, got %d", ForgeModeSubmit)
	}
	if ForgeModeEntries != 3 {
		t.Errorf("ForgeModeEntries should be 3, got %d", ForgeModeEntries)
	}
}

// Shadow Play Panel tests.

func TestNewShadowPlayPanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	if panel == nil {
		t.Fatal("NewShadowPlayPanel returned nil")
	}
	if panel.IsVisible() {
		t.Error("Panel should not be visible initially")
	}
	if panel.GetMode() != ShadowPlayModeOverview {
		t.Errorf("Initial mode should be Overview, got %v", panel.GetMode())
	}
}

func TestShadowPlayPanelShowHide(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	game := &ShadowPlayGameInfo{
		GameID:      [32]byte{1, 2, 3},
		State:       ShadowPlayStateActive,
		RoundNumber: 1,
		Players: []ShadowPlayPlayer{
			{Name: "Player1", Role: ShadowRoleEcho},
			{Name: "Player2", Role: ShadowRoleShade},
		},
	}

	panel.Show(game)
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestShadowPlayPanelShowNil(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	panel.Show(nil)
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show(nil)")
	}
}

func TestShadowPlayPanelModeAutoSelect(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	// Voting state without having voted should show vote mode.
	game := &ShadowPlayGameInfo{
		GameID:   [32]byte{1},
		State:    ShadowPlayStateVoting,
		HasVoted: false,
	}
	panel.Show(game)
	if panel.GetMode() != ShadowPlayModeVote {
		t.Errorf("Mode should be Vote for voting state, got %v", panel.GetMode())
	}

	// Voting state with already voted should show overview.
	game2 := &ShadowPlayGameInfo{
		GameID:   [32]byte{2},
		State:    ShadowPlayStateVoting,
		HasVoted: true,
	}
	panel.Show(game2)
	if panel.GetMode() != ShadowPlayModeOverview {
		t.Errorf("Mode should be Overview when already voted, got %v", panel.GetMode())
	}

	// Win state should show results.
	game3 := &ShadowPlayGameInfo{
		GameID: [32]byte{3},
		State:  ShadowPlayStateEchoesWin,
	}
	panel.Show(game3)
	if panel.GetMode() != ShadowPlayModeResults {
		t.Errorf("Mode should be Results for win state, got %v", panel.GetMode())
	}
}

func TestShadowPlayPanelShowRoleReveal(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	game := &ShadowPlayGameInfo{
		GameID: [32]byte{1},
		State:  ShadowPlayStateActive,
		MyRole: ShadowRoleShade,
	}

	panel.ShowRoleReveal(game)
	if !panel.IsVisible() {
		t.Error("Panel should be visible after ShowRoleReveal()")
	}
	if panel.GetMode() != ShadowPlayModeRole {
		t.Errorf("Mode should be Role, got %v", panel.GetMode())
	}
}

func TestShadowPlayPanelSetGame(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	game := &ShadowPlayGameInfo{
		GameID:      [32]byte{1, 2, 3},
		RoundNumber: 3,
	}

	panel.SetGame(game)
	if panel.GetGame() != game {
		t.Error("GetGame should return the set game")
	}
}

func TestShadowPlayPanelSetMode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	panel.SetMode(ShadowPlayModeVote)
	if panel.GetMode() != ShadowPlayModeVote {
		t.Errorf("Mode should be Vote, got %v", panel.GetMode())
	}

	panel.SetMode(ShadowPlayModeResults)
	if panel.GetMode() != ShadowPlayModeResults {
		t.Errorf("Mode should be Results, got %v", panel.GetMode())
	}
}

func TestShadowPlayPanelCallbacks(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	var voteCalled bool
	panel.SetOnVote(func(gameID, targetSpecter [32]byte) {
		voteCalled = true
	})

	var joinCalled bool
	panel.SetOnJoin(func(gameID [32]byte) {
		joinCalled = true
	})

	var leaveCalled bool
	panel.SetOnLeave(func(gameID [32]byte) {
		leaveCalled = true
	})

	// Callbacks should be set without error.
	// (Actual invocation would require UI interaction.)
	_ = voteCalled
	_ = joinCalled
	_ = leaveCalled
}

func TestShadowPlayPanelSelection(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	panel.SetSelectedIndex(5)
	if panel.GetSelectedIndex() != 5 {
		t.Errorf("SelectedIndex should be 5, got %d", panel.GetSelectedIndex())
	}
}

func TestShadowPlayPanelUpdate(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	// Update should not panic.
	err := panel.Update()
	if err != nil {
		t.Errorf("Update returned error: %v", err)
	}

	panel.Show(&ShadowPlayGameInfo{
		GameID: [32]byte{1},
		State:  ShadowPlayStateActive,
	})

	err = panel.Update()
	if err != nil {
		t.Errorf("Update returned error: %v", err)
	}
}

func TestShadowPlayStateString(t *testing.T) {
	tests := []struct {
		state ShadowPlayState
		want  string
	}{
		{ShadowPlayStateWaiting, "Waiting"},
		{ShadowPlayStateActive, "Active"},
		{ShadowPlayStateVoting, "Voting"},
		{ShadowPlayStateEchoesWin, "Echoes Win!"},
		{ShadowPlayStateShadesWin, "Shades Win!"},
		{ShadowPlayStateExpired, "Expired"},
		{ShadowPlayState(99), "Unknown"},
	}

	for _, tc := range tests {
		got := ShadowPlayStateString(tc.state)
		if got != tc.want {
			t.Errorf("ShadowPlayStateString(%d) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestShadowRoleString(t *testing.T) {
	tests := []struct {
		role ShadowPlayerRole
		want string
	}{
		{ShadowRoleUnknown, "Unknown"},
		{ShadowRoleEcho, "Echo"},
		{ShadowRoleShade, "Shade"},
	}

	for _, tc := range tests {
		got := ShadowRoleString(tc.role)
		if got != tc.want {
			t.Errorf("ShadowRoleString(%d) = %q, want %q", tc.role, got, tc.want)
		}
	}
}

func TestShadowPlayPanelConcurrency(t *testing.T) {
	theme := DefaultTheme()
	panel := NewShadowPlayPanel(theme)

	done := make(chan bool, 4)

	// Writer goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			panel.SetGame(&ShadowPlayGameInfo{
				GameID:      [32]byte{byte(i)},
				RoundNumber: i,
			})
			panel.SetMode(ShadowPlayPanelMode(i % 4))
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			_ = panel.GetGame()
			_ = panel.GetMode()
			_ = panel.IsVisible()
		}
		done <- true
	}()

	// Show/Hide goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				panel.Show(&ShadowPlayGameInfo{GameID: [32]byte{byte(i)}})
			} else {
				panel.Hide()
			}
		}
		done <- true
	}()

	// Update goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			panel.Update()
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}
