// Package overlays — Forge overlay tests.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"image/color"
	"testing"
	"time"
)

func TestForgeOverlayCreation(t *testing.T) {
	overlay := NewForgeOverlay()
	if overlay == nil {
		t.Fatal("Expected non-nil overlay")
	}

	if !overlay.IsVisible() {
		t.Error("Expected overlay to be visible by default")
	}

	if overlay.ForgeCount() != 0 {
		t.Error("Expected zero forges initially")
	}
}

func TestForgeOverlayVisibility(t *testing.T) {
	overlay := NewForgeOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("Expected overlay to be hidden")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("Expected overlay to be visible")
	}
}

func TestForgeOverlaySetAndGet(t *testing.T) {
	overlay := NewForgeOverlay()

	var forgeID [32]byte
	copy(forgeID[:], []byte("test-forge-001"))

	forge := &ForgeEventInfo{
		ForgeID:   forgeID,
		Type:      ForgeSigilArt,
		State:     ForgeActive,
		X:         100.0,
		Y:         200.0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Minute),
		Entries:   nil,
	}

	overlay.SetForge(forge)

	if overlay.ForgeCount() != 1 {
		t.Errorf("Expected 1 forge, got %d", overlay.ForgeCount())
	}

	retrieved := overlay.GetForge(forgeID)
	if retrieved == nil {
		t.Fatal("Expected to retrieve forge")
	}

	if retrieved.X != 100.0 || retrieved.Y != 200.0 {
		t.Error("Forge position mismatch")
	}

	if retrieved.Type != ForgeSigilArt {
		t.Error("Forge type mismatch")
	}
}

func TestForgeOverlayUpdate(t *testing.T) {
	overlay := NewForgeOverlay()

	// Update should not panic.
	overlay.Update(0.016) // ~60fps.
	overlay.Update(0.016)
	overlay.Update(0.016)
}

func TestForgeOverlayRemove(t *testing.T) {
	overlay := NewForgeOverlay()

	var forgeID [32]byte
	copy(forgeID[:], []byte("test-forge-remove"))

	forge := &ForgeEventInfo{
		ForgeID: forgeID,
		Type:    ForgeMicroFic,
		State:   ForgeActive,
	}

	overlay.SetForge(forge)
	if overlay.ForgeCount() != 1 {
		t.Fatal("Expected 1 forge after set")
	}

	overlay.RemoveForge(forgeID)
	if overlay.ForgeCount() != 0 {
		t.Error("Expected 0 forges after remove")
	}

	if overlay.GetForge(forgeID) != nil {
		t.Error("Expected nil after remove")
	}
}

func TestForgeOverlayGetAll(t *testing.T) {
	overlay := NewForgeOverlay()

	// Add multiple forges.
	for i := 0; i < 3; i++ {
		var forgeID [32]byte
		forgeID[0] = byte(i + 1)

		forge := &ForgeEventInfo{
			ForgeID: forgeID,
			Type:    ForgeType(i % 3),
			State:   ForgeActive,
		}
		overlay.SetForge(forge)
	}

	forges := overlay.GetAllForges()
	if len(forges) != 3 {
		t.Errorf("Expected 3 forges, got %d", len(forges))
	}
}

func TestForgeOverlayClearCompleted(t *testing.T) {
	overlay := NewForgeOverlay()

	// Add active forge.
	var activeID [32]byte
	copy(activeID[:], []byte("active-forge"))
	activeForge := &ForgeEventInfo{
		ForgeID: activeID,
		State:   ForgeActive,
		EndTime: time.Now().Add(1 * time.Hour),
	}
	overlay.SetForge(activeForge)

	// Add completed forge that just ended.
	var recentID [32]byte
	copy(recentID[:], []byte("recent-completed"))
	recentForge := &ForgeEventInfo{
		ForgeID: recentID,
		State:   ForgeCompleted,
		EndTime: time.Now().Add(-1 * time.Hour),
	}
	overlay.SetForge(recentForge)

	// Add old completed forge.
	var oldID [32]byte
	copy(oldID[:], []byte("old-completed"))
	oldForge := &ForgeEventInfo{
		ForgeID: oldID,
		State:   ForgeCompleted,
		EndTime: time.Now().Add(-48 * time.Hour), // More than 24h ago.
	}
	overlay.SetForge(oldForge)

	if overlay.ForgeCount() != 3 {
		t.Fatalf("Expected 3 forges before clear, got %d", overlay.ForgeCount())
	}

	removed := overlay.ClearCompleted()
	if removed != 1 {
		t.Errorf("Expected 1 removed, got %d", removed)
	}

	if overlay.ForgeCount() != 2 {
		t.Errorf("Expected 2 forges after clear, got %d", overlay.ForgeCount())
	}

	// Active and recent should remain.
	if overlay.GetForge(activeID) == nil {
		t.Error("Active forge should remain")
	}
	if overlay.GetForge(recentID) == nil {
		t.Error("Recent completed forge should remain")
	}
	if overlay.GetForge(oldID) != nil {
		t.Error("Old completed forge should be removed")
	}
}

func TestForgeEntries(t *testing.T) {
	overlay := NewForgeOverlay()

	var forgeID [32]byte
	copy(forgeID[:], []byte("forge-with-entries"))

	entries := []ForgeEntry{
		{
			EntryID:    [32]byte{1},
			SpecterKey: [32]byte{10},
			SigilColor: color.RGBA{255, 0, 0, 255},
			Score:      50.0,
			IsWinner:   false,
		},
		{
			EntryID:    [32]byte{2},
			SpecterKey: [32]byte{20},
			SigilColor: color.RGBA{0, 255, 0, 255},
			Score:      75.0,
			IsWinner:   true,
		},
		{
			EntryID:    [32]byte{3},
			SpecterKey: [32]byte{30},
			SigilColor: color.RGBA{0, 0, 255, 255},
			Score:      25.0,
			IsWinner:   false,
		},
	}

	forge := &ForgeEventInfo{
		ForgeID:   forgeID,
		Type:      ForgeSigilArt,
		State:     ForgeCompleted,
		X:         0,
		Y:         0,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Entries:   entries,
	}

	overlay.SetForge(forge)

	retrieved := overlay.GetForge(forgeID)
	if len(retrieved.Entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(retrieved.Entries))
	}

	// Check winner.
	var winner *ForgeEntry
	for i := range retrieved.Entries {
		if retrieved.Entries[i].IsWinner {
			winner = &retrieved.Entries[i]
			break
		}
	}

	if winner == nil {
		t.Fatal("Expected to find winner")
	}

	if winner.Score != 75.0 {
		t.Errorf("Winner score mismatch: expected 75.0, got %v", winner.Score)
	}
}

func TestForgeTypeString(t *testing.T) {
	tests := []struct {
		ft       ForgeType
		expected string
	}{
		{ForgeSigilArt, "Sigil Art"},
		{ForgeMicroFic, "Micro-Fiction"},
		{ForgeRemixChain, "Remix Chain"},
		{ForgeType(99), "Unknown"},
	}

	for _, tt := range tests {
		result := ForgeTypeString(tt.ft)
		if result != tt.expected {
			t.Errorf("ForgeTypeString(%d) = %s, want %s", tt.ft, result, tt.expected)
		}
	}
}

func TestForgeStateString(t *testing.T) {
	tests := []struct {
		fs       ForgeState
		expected string
	}{
		{ForgeActive, "Active"},
		{ForgeEvaluate, "Evaluating"},
		{ForgeCompleted, "Completed"},
		{ForgeState(99), "Unknown"},
	}

	for _, tt := range tests {
		result := ForgeStateString(tt.fs)
		if result != tt.expected {
			t.Errorf("ForgeStateString(%d) = %s, want %s", tt.fs, result, tt.expected)
		}
	}
}

func TestForgeUpdateForge(t *testing.T) {
	overlay := NewForgeOverlay()

	var forgeID [32]byte
	copy(forgeID[:], []byte("update-forge"))

	// Initial forge.
	forge := &ForgeEventInfo{
		ForgeID: forgeID,
		Type:    ForgeSigilArt,
		State:   ForgeActive,
		X:       0,
		Y:       0,
	}
	overlay.SetForge(forge)

	// Update with new state.
	updatedForge := &ForgeEventInfo{
		ForgeID: forgeID,
		Type:    ForgeSigilArt,
		State:   ForgeCompleted,
		X:       100,
		Y:       200,
	}
	overlay.SetForge(updatedForge)

	retrieved := overlay.GetForge(forgeID)
	if retrieved.State != ForgeCompleted {
		t.Error("Forge state should be updated")
	}
	if retrieved.X != 100 || retrieved.Y != 200 {
		t.Error("Forge position should be updated")
	}

	// Should still be 1 forge (update, not add).
	if overlay.ForgeCount() != 1 {
		t.Errorf("Expected 1 forge, got %d", overlay.ForgeCount())
	}
}

func TestForgeOverlayConcurrency(t *testing.T) {
	overlay := NewForgeOverlay()

	done := make(chan bool, 4)

	// Writer goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			var forgeID [32]byte
			forgeID[0] = byte(i % 10)
			forge := &ForgeEventInfo{
				ForgeID: forgeID,
				State:   ForgeActive,
			}
			overlay.SetForge(forge)
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			var forgeID [32]byte
			forgeID[0] = byte(i % 10)
			_ = overlay.GetForge(forgeID)
		}
		done <- true
	}()

	// GetAll goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			_ = overlay.GetAllForges()
		}
		done <- true
	}()

	// Update goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			overlay.Update(0.016)
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}
