// Package ui provides tests for UI panels.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"testing"
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
