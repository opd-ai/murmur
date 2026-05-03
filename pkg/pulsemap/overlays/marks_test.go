// Package overlays provides Anonymous Layer overlay and activity heatmap.
// Tests for Specter Mark visualization.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

func TestNewMarkOverlay(t *testing.T) {
	overlay := NewMarkOverlay()
	if overlay == nil {
		t.Fatal("Expected non-nil overlay")
	}
	if overlay.marks == nil {
		t.Error("Expected marks map to be initialized")
	}
	if overlay.GetTotalMarkCount() != 0 {
		t.Error("Expected empty overlay")
	}
}

func TestMarkOverlayAddMark(t *testing.T) {
	overlay := NewMarkOverlay()

	// Create a mark.
	mark := createTestMark(t, marks.MarkWatcher)
	targetID := hex.EncodeToString(mark.TargetKey)

	overlay.AddMark(targetID, mark)

	if !overlay.HasMarks(targetID) {
		t.Error("Expected marks on target")
	}
	if overlay.GetMarkCount(targetID) != 1 {
		t.Errorf("Expected 1 mark, got %d", overlay.GetMarkCount(targetID))
	}
	if overlay.GetTotalMarkCount() != 1 {
		t.Errorf("Expected total 1, got %d", overlay.GetTotalMarkCount())
	}
}

func TestMarkOverlayAddNilMark(t *testing.T) {
	overlay := NewMarkOverlay()
	overlay.AddMark("test-target", nil)

	if overlay.GetTotalMarkCount() != 0 {
		t.Error("Should not add nil mark")
	}
}

func TestMarkOverlayDuplicatePrevention(t *testing.T) {
	overlay := NewMarkOverlay()

	mark := createTestMark(t, marks.MarkAlly)
	targetID := hex.EncodeToString(mark.TargetKey)

	overlay.AddMark(targetID, mark)
	overlay.AddMark(targetID, mark) // Duplicate

	if overlay.GetMarkCount(targetID) != 1 {
		t.Errorf("Expected 1 mark (no duplicate), got %d", overlay.GetMarkCount(targetID))
	}
}

func TestMarkOverlayRemoveMark(t *testing.T) {
	overlay := NewMarkOverlay()

	mark := createTestMark(t, marks.MarkRival)
	targetID := hex.EncodeToString(mark.TargetKey)

	overlay.AddMark(targetID, mark)
	if !overlay.HasMarks(targetID) {
		t.Fatal("Expected mark to be added")
	}

	overlay.RemoveMark(mark.ID)

	if overlay.HasMarks(targetID) {
		t.Error("Expected marks to be removed")
	}
	if overlay.GetTotalMarkCount() != 0 {
		t.Errorf("Expected 0 total, got %d", overlay.GetTotalMarkCount())
	}
}

func TestMarkOverlayRemoveAllForTarget(t *testing.T) {
	overlay := NewMarkOverlay()

	targetKey := make([]byte, 32)
	rand.Read(targetKey)
	targetID := hex.EncodeToString(targetKey)

	// Add multiple marks from different markers.
	for i := 0; i < 3; i++ {
		mark := createTestMarkForTarget(t, marks.MarkWatcher, targetKey)
		overlay.AddMark(targetID, mark)
	}

	if overlay.GetMarkCount(targetID) != 3 {
		t.Fatalf("Expected 3 marks, got %d", overlay.GetMarkCount(targetID))
	}

	overlay.RemoveAllMarksForTarget(targetID)

	if overlay.HasMarks(targetID) {
		t.Error("Expected all marks removed")
	}
}

func TestMarkOverlayUpdate(t *testing.T) {
	overlay := NewMarkOverlay()

	mark := createTestMark(t, marks.MarkWatcher)
	targetID := hex.EncodeToString(mark.TargetKey)
	overlay.AddMark(targetID, mark)

	// Get initial state.
	overlay.mu.RLock()
	initialAngle := overlay.marks[targetID][0].OrbitAngle
	overlay.mu.RUnlock()

	// Update with 1 second delta.
	overlay.Update(1.0)

	overlay.mu.RLock()
	newAngle := overlay.marks[targetID][0].OrbitAngle
	overlay.mu.RUnlock()

	if newAngle == initialAngle {
		t.Error("Expected orbit angle to advance")
	}
}

func TestMarkOverlayClearExpired(t *testing.T) {
	overlay := NewMarkOverlay()

	// Create an expired mark.
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	var markerKey [32]byte
	copy(markerKey[:], priv.Public().(ed25519.PublicKey)[:32])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)
	var targetKeyArr [32]byte
	copy(targetKeyArr[:], targetKey)

	mark := &marks.Mark{
		MarkerKey:  markerKey,
		TargetKey:  targetKey,
		Category:   marks.MarkWatcher,
		CreatedAt:  time.Now().Add(-31 * 24 * time.Hour), // Expired
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
		Visibility: 0,
	}
	copy(mark.ID[:], targetKey[:16])
	copy(mark.ID[16:], markerKey[:16])

	targetID := hex.EncodeToString(targetKey)
	overlay.AddMark(targetID, mark)

	// Clear expired.
	overlay.ClearExpiredMarks()

	if overlay.HasMarks(targetID) {
		t.Error("Expected expired mark to be cleared")
	}
}

func TestMarkOverlayGetDominantCategory(t *testing.T) {
	overlay := NewMarkOverlay()

	targetKey := make([]byte, 32)
	rand.Read(targetKey)
	targetID := hex.EncodeToString(targetKey)

	// Add 2 Ally marks and 1 Watcher.
	mark1 := createTestMarkForTarget(t, marks.MarkAlly, targetKey)
	mark2 := createTestMarkForTarget(t, marks.MarkAlly, targetKey)
	mark3 := createTestMarkForTarget(t, marks.MarkWatcher, targetKey)

	overlay.AddMark(targetID, mark1)
	overlay.AddMark(targetID, mark2)
	overlay.AddMark(targetID, mark3)

	dominant := overlay.GetDominantCategory(targetID)
	if dominant != marks.MarkAlly {
		t.Errorf("Expected dominant category Ally, got %d", dominant)
	}
}

func TestMarkOverlayRender(t *testing.T) {
	overlay := NewMarkOverlay()

	mark := createTestMark(t, marks.MarkWatcher)
	targetID := hex.EncodeToString(mark.TargetKey)
	overlay.AddMark(targetID, mark)

	// In noebiten mode, Render is a no-op but should not panic.
	overlay.Render(nil, targetID, 100.0, 100.0)
}

func TestMarkOverlaySyncFromStore(t *testing.T) {
	overlay := NewMarkOverlay()
	store := marks.NewMarkStore()

	// Create marks via store (need Resonance >= 100).
	var markerKey1, markerKey2 [32]byte
	rand.Read(markerKey1[:])
	rand.Read(markerKey2[:])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, _ = store.PlaceMark(markerKey1, targetKey, marks.MarkWatcher, "test", 100, nil)
	_, _ = store.PlaceMark(markerKey2, targetKey, marks.MarkAlly, "test", 100, nil)

	// Sync to overlay.
	overlay.SyncFromStore(store)

	targetID := hex.EncodeToString(targetKey)
	if overlay.GetMarkCount(targetID) != 2 {
		t.Errorf("Expected 2 marks from store, got %d", overlay.GetMarkCount(targetID))
	}
}

func TestMarkOverlaySyncNilStore(t *testing.T) {
	overlay := NewMarkOverlay()
	// Should not panic.
	overlay.SyncFromStore(nil)
}

func TestMarkOverlayMultipleTargets(t *testing.T) {
	overlay := NewMarkOverlay()

	// Add marks on 3 different targets.
	for i := 0; i < 3; i++ {
		mark := createTestMark(t, marks.MarkCategory(i+1))
		targetID := hex.EncodeToString(mark.TargetKey)
		overlay.AddMark(targetID, mark)
	}

	if overlay.GetTotalMarkCount() != 3 {
		t.Errorf("Expected 3 total marks, got %d", overlay.GetTotalMarkCount())
	}
}

func TestMarkOverlayConcurrentAccess(t *testing.T) {
	overlay := NewMarkOverlay()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			mark := createTestMark(t, marks.MarkWatcher)
			targetID := hex.EncodeToString(mark.TargetKey)
			overlay.AddMark(targetID, mark)
			overlay.Update(0.016)
			_ = overlay.GetTotalMarkCount()
			_ = overlay.HasMarks(targetID)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper to create a test mark.
func createTestMark(t *testing.T, category marks.MarkCategory) *marks.Mark {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	var markerKey [32]byte
	copy(markerKey[:], priv.Public().(ed25519.PublicKey)[:32])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark := &marks.Mark{
		MarkerKey:  markerKey,
		TargetKey:  targetKey,
		Category:   category,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		Visibility: 1.0,
	}
	copy(mark.ID[:], targetKey[:16])
	copy(mark.ID[16:], markerKey[:16])

	return mark
}

// Helper to create a test mark for a specific target.
func createTestMarkForTarget(t *testing.T, category marks.MarkCategory, targetKey []byte) *marks.Mark {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	var markerKey [32]byte
	copy(markerKey[:], priv.Public().(ed25519.PublicKey)[:32])

	mark := &marks.Mark{
		MarkerKey:  markerKey,
		TargetKey:  targetKey,
		Category:   category,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
		Visibility: 1.0,
	}
	copy(mark.ID[:], targetKey[:16])
	copy(mark.ID[16:], markerKey[:16])

	return mark
}
