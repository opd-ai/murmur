// Package ui - Tests for Specter detail panel.
//

package ui

import (
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

func TestNewSpecterDetailPanel(t *testing.T) {
	theme := DefaultTheme()
	callbacks := SpecterDetailCallbacks{}

	panel := NewSpecterDetailPanel(theme, callbacks)

	if panel == nil {
		t.Fatal("NewSpecterDetailPanel returned nil")
	}
	if panel.Visible() {
		t.Error("panel should not be visible initially")
	}
	if panel.GetMode() != SpecterModeOverview {
		t.Error("initial mode should be SpecterModeOverview")
	}
	if panel.GetSelectedTrophy() != -1 {
		t.Error("no trophy should be selected initially")
	}
}

func TestSpecterDetailPanel_ShowHide(t *testing.T) {
	panel := NewSpecterDetailPanel(DefaultTheme(), SpecterDetailCallbacks{})

	if panel.Visible() {
		t.Error("panel should start hidden")
	}

	panel.Show()
	if !panel.Visible() {
		t.Error("panel should be visible after Show")
	}

	panel.Hide()
	if panel.Visible() {
		t.Error("panel should be hidden after Hide")
	}

	panel.Toggle()
	if !panel.Visible() {
		t.Error("panel should be visible after Toggle from hidden")
	}

	panel.Toggle()
	if panel.Visible() {
		t.Error("panel should be hidden after Toggle from visible")
	}
}

func TestSpecterDetailPanel_ShowForSpecter(t *testing.T) {
	trophies := []TrophyDisplayInfo{
		{
			Trophy: mechanics.TrophyUnlock{
				TrophyID:   mechanics.TrophyFirstShade,
				UnlockedAt: time.Now(),
			},
			Def: &mechanics.TrophyDefinition{
				ID:       mechanics.TrophyFirstShade,
				Name:     "First Shade",
				Category: mechanics.TrophyCategoryMilestone,
			},
		},
	}

	callbacks := SpecterDetailCallbacks{
		GetTrophies: func(specterID [32]byte) []TrophyDisplayInfo {
			return trophies
		},
	}

	panel := NewSpecterDetailPanel(DefaultTheme(), callbacks)

	info := &SpecterInfo{
		ID:        [32]byte{1, 2, 3},
		Pseudonym: "TestSpecter",
		Resonance: 50.5,
		Rank:      "Wraith",
	}

	panel.ShowForSpecter(info)

	if !panel.Visible() {
		t.Error("panel should be visible after ShowForSpecter")
	}
	if panel.GetSpecter() != info {
		t.Error("panel should have the specter info set")
	}
	if panel.TrophyCount() != 1 {
		t.Errorf("expected 1 trophy, got %d", panel.TrophyCount())
	}
	if panel.GetMode() != SpecterModeOverview {
		t.Error("mode should be reset to overview")
	}
}

func TestSpecterDetailPanel_SetSpecter(t *testing.T) {
	panel := NewSpecterDetailPanel(DefaultTheme(), SpecterDetailCallbacks{})

	info1 := &SpecterInfo{ID: [32]byte{1}, Pseudonym: "First"}
	info2 := &SpecterInfo{ID: [32]byte{2}, Pseudonym: "Second"}

	panel.SetSpecter(info1)
	if panel.GetSpecter() != info1 {
		t.Error("GetSpecter should return info1")
	}

	panel.SetSpecter(info2)
	if panel.GetSpecter() != info2 {
		t.Error("GetSpecter should return info2")
	}

	panel.SetSpecter(nil)
	if panel.GetSpecter() != nil {
		t.Error("GetSpecter should return nil")
	}
}

func TestSpecterDetailPanel_Modes(t *testing.T) {
	panel := NewSpecterDetailPanel(DefaultTheme(), SpecterDetailCallbacks{})

	modes := []SpecterDetailMode{
		SpecterModeOverview,
		SpecterModeTrophies,
		SpecterModeActivity,
		SpecterModeInteract,
	}

	for _, mode := range modes {
		panel.SetMode(mode)
		if panel.GetMode() != mode {
			t.Errorf("expected mode %d, got %d", mode, panel.GetMode())
		}
	}
}

func TestSpecterDetailPanel_TrophySelection(t *testing.T) {
	trophies := []TrophyDisplayInfo{
		{Trophy: mechanics.TrophyUnlock{TrophyID: mechanics.TrophyFirstShade}},
		{Trophy: mechanics.TrophyUnlock{TrophyID: mechanics.TrophyWraithRising}},
		{Trophy: mechanics.TrophyUnlock{TrophyID: mechanics.TrophyPhantomAscendant}},
	}

	callbacks := SpecterDetailCallbacks{
		GetTrophies: func(specterID [32]byte) []TrophyDisplayInfo {
			return trophies
		},
	}

	panel := NewSpecterDetailPanel(DefaultTheme(), callbacks)
	panel.ShowForSpecter(&SpecterInfo{ID: [32]byte{1}})

	// Initial state.
	if panel.GetSelectedTrophy() != -1 {
		t.Error("no trophy should be selected initially")
	}

	// Select first trophy.
	panel.SetSelectedTrophy(0)
	if panel.GetSelectedTrophy() != 0 {
		t.Error("trophy 0 should be selected")
	}

	// Select second trophy.
	panel.SetSelectedTrophy(1)
	if panel.GetSelectedTrophy() != 1 {
		t.Error("trophy 1 should be selected")
	}

	// Deselect.
	panel.SetSelectedTrophy(-1)
	if panel.GetSelectedTrophy() != -1 {
		t.Error("no trophy should be selected after deselect")
	}

	// Out of bounds - should not change.
	panel.SetSelectedTrophy(0)
	panel.SetSelectedTrophy(100)
	if panel.GetSelectedTrophy() != 0 {
		t.Error("out of bounds selection should be ignored")
	}
}

func TestSpecterDetailPanel_RefreshTrophies(t *testing.T) {
	callCount := 0
	trophies := []TrophyDisplayInfo{
		{Trophy: mechanics.TrophyUnlock{TrophyID: mechanics.TrophyFirstShade}},
	}

	callbacks := SpecterDetailCallbacks{
		GetTrophies: func(specterID [32]byte) []TrophyDisplayInfo {
			callCount++
			return trophies
		},
	}

	panel := NewSpecterDetailPanel(DefaultTheme(), callbacks)
	panel.ShowForSpecter(&SpecterInfo{ID: [32]byte{1}})

	initialCount := callCount

	panel.RefreshTrophies()

	if callCount <= initialCount {
		t.Error("RefreshTrophies should call GetTrophies callback")
	}
}

func TestSpecterDetailPanel_Callbacks(t *testing.T) {
	closeCalled := false
	giftCalled := false
	wavesCalled := false
	markCalled := false
	var lastID [32]byte

	callbacks := SpecterDetailCallbacks{
		OnClose: func() {
			closeCalled = true
		},
		OnSendGift: func(specterID [32]byte) {
			giftCalled = true
			lastID = specterID
		},
		OnViewWaves: func(specterID [32]byte) {
			wavesCalled = true
			lastID = specterID
		},
		OnAddMark: func(specterID [32]byte) {
			markCalled = true
			lastID = specterID
		},
	}

	panel := NewSpecterDetailPanel(DefaultTheme(), callbacks)
	info := &SpecterInfo{ID: [32]byte{42}}
	panel.ShowForSpecter(info)

	// Verify callbacks are set.
	if panel.callbacks.OnClose == nil {
		t.Error("OnClose callback should be set")
	}
	if panel.callbacks.OnSendGift == nil {
		t.Error("OnSendGift callback should be set")
	}
	if panel.callbacks.OnViewWaves == nil {
		t.Error("OnViewWaves callback should be set")
	}
	if panel.callbacks.OnAddMark == nil {
		t.Error("OnAddMark callback should be set")
	}

	// Test callbacks directly.
	panel.callbacks.OnClose()
	if !closeCalled {
		t.Error("OnClose should have been called")
	}

	panel.callbacks.OnSendGift(info.ID)
	if !giftCalled || lastID != info.ID {
		t.Error("OnSendGift should have been called with correct ID")
	}

	panel.callbacks.OnViewWaves(info.ID)
	if !wavesCalled || lastID != info.ID {
		t.Error("OnViewWaves should have been called with correct ID")
	}

	panel.callbacks.OnAddMark(info.ID)
	if !markCalled || lastID != info.ID {
		t.Error("OnAddMark should have been called with correct ID")
	}
}

func TestSpecterDetailPanel_TrophyCount(t *testing.T) {
	emptyCallbacks := SpecterDetailCallbacks{
		GetTrophies: func(specterID [32]byte) []TrophyDisplayInfo {
			return nil
		},
	}

	panel := NewSpecterDetailPanel(DefaultTheme(), emptyCallbacks)
	panel.ShowForSpecter(&SpecterInfo{ID: [32]byte{1}})

	if panel.TrophyCount() != 0 {
		t.Error("TrophyCount should be 0 with empty trophies")
	}

	// With trophies.
	withTrophies := SpecterDetailCallbacks{
		GetTrophies: func(specterID [32]byte) []TrophyDisplayInfo {
			return make([]TrophyDisplayInfo, 5)
		},
	}

	panel2 := NewSpecterDetailPanel(DefaultTheme(), withTrophies)
	panel2.ShowForSpecter(&SpecterInfo{ID: [32]byte{1}})

	if panel2.TrophyCount() != 5 {
		t.Errorf("TrophyCount should be 5, got %d", panel2.TrophyCount())
	}
}

func TestSpecterInfo_Fields(t *testing.T) {
	now := time.Now()
	info := &SpecterInfo{
		ID:             [32]byte{1, 2, 3, 4},
		Pseudonym:      "CrypticWanderer42",
		Resonance:      125.5,
		Rank:           "Phantom",
		CreatedAt:      now.Add(-30 * 24 * time.Hour),
		LastSeenAt:     now,
		WaveCount:      42,
		GiftsSent:      10,
		GiftsReceived:  15,
		PuzzlesSolved:  5,
		HuntsCompleted: 3,
		IsOwnSpecter:   false,
	}

	if info.Pseudonym != "CrypticWanderer42" {
		t.Error("Pseudonym mismatch")
	}
	if info.Resonance != 125.5 {
		t.Error("Resonance mismatch")
	}
	if info.Rank != "Phantom" {
		t.Error("Rank mismatch")
	}
	if info.WaveCount != 42 {
		t.Error("WaveCount mismatch")
	}
	if info.IsOwnSpecter {
		t.Error("IsOwnSpecter should be false")
	}
}

func TestTrophyDisplayInfo_Fields(t *testing.T) {
	def := &mechanics.TrophyDefinition{
		ID:          mechanics.TrophyFirstShade,
		Name:        "First Shade",
		Description: "Reach Resonance 25",
		Category:    mechanics.TrophyCategoryMilestone,
		Threshold:   25,
		Bonus:       0,
		Animated:    false,
	}

	trophy := TrophyDisplayInfo{
		Trophy: mechanics.TrophyUnlock{
			TrophyID:   mechanics.TrophyFirstShade,
			UnlockedAt: time.Now(),
			Resonance:  25.5,
		},
		Def:      def,
		Selected: true,
	}

	if trophy.Trophy.TrophyID != mechanics.TrophyFirstShade {
		t.Error("TrophyID mismatch")
	}
	if trophy.Def.Name != "First Shade" {
		t.Error("Def.Name mismatch")
	}
	if !trophy.Selected {
		t.Error("Selected should be true")
	}
}

func TestSpecterDetailMode_Constants(t *testing.T) {
	if SpecterModeOverview != 0 {
		t.Error("SpecterModeOverview should be 0")
	}
	if SpecterModeTrophies != 1 {
		t.Error("SpecterModeTrophies should be 1")
	}
	if SpecterModeActivity != 2 {
		t.Error("SpecterModeActivity should be 2")
	}
	if SpecterModeInteract != 3 {
		t.Error("SpecterModeInteract should be 3")
	}
}

func TestSpecterDetailPanel_NilSpecter(t *testing.T) {
	panel := NewSpecterDetailPanel(DefaultTheme(), SpecterDetailCallbacks{})

	// Update with nil specter should not panic.
	panel.Show()
	consumed := panel.Update()

	// Should return false when specter is nil.
	if consumed {
		t.Error("Update should return false with nil specter")
	}
}

func TestSpecterDetailPanel_NoCallbacks(t *testing.T) {
	// Panel with nil callbacks should not panic.
	panel := NewSpecterDetailPanel(DefaultTheme(), SpecterDetailCallbacks{})
	panel.ShowForSpecter(&SpecterInfo{ID: [32]byte{1}})

	// TrophyCount should be 0 when GetTrophies is nil.
	if panel.TrophyCount() != 0 {
		t.Error("TrophyCount should be 0 with nil GetTrophies")
	}

	// RefreshTrophies with nil callback should not panic.
	panel.RefreshTrophies()
}
