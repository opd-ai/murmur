package effects

import (
	"crypto/rand"
	"testing"
)

func TestNewHuntEffects(t *testing.T) {
	h := NewHuntEffects()
	if h == nil {
		t.Fatal("expected HuntEffects, got nil")
	}
	if h.FragmentCount() != 0 {
		t.Errorf("expected 0 fragments, got %d", h.FragmentCount())
	}
	if h.HuntCount() != 0 {
		t.Errorf("expected 0 hunts, got %d", h.HuntCount())
	}
}

func TestHuntEffects_AddRemoveFragment(t *testing.T) {
	h := NewHuntEffects()

	var id [32]byte
	rand.Read(id[:])

	frag := &FragmentVisual{
		ID:    id,
		X:     100,
		Y:     200,
		State: FragmentUnclaimed,
		Index: 0,
	}

	h.AddFragment(frag)
	if h.FragmentCount() != 1 {
		t.Errorf("expected 1 fragment after add, got %d", h.FragmentCount())
	}

	retrieved := h.GetFragment(id)
	if retrieved == nil {
		t.Fatal("expected fragment, got nil")
	}
	if retrieved.X != 100 || retrieved.Y != 200 {
		t.Errorf("fragment position mismatch: got (%f, %f)", retrieved.X, retrieved.Y)
	}

	h.RemoveFragment(id)
	if h.FragmentCount() != 0 {
		t.Errorf("expected 0 fragments after remove, got %d", h.FragmentCount())
	}
}

func TestHuntEffects_AddNilFragment(t *testing.T) {
	h := NewHuntEffects()
	h.AddFragment(nil)
	if h.FragmentCount() != 0 {
		t.Errorf("expected 0 fragments after adding nil, got %d", h.FragmentCount())
	}
}

func TestHuntEffects_SetGetHuntState(t *testing.T) {
	h := NewHuntEffects()

	var huntID [32]byte
	rand.Read(huntID[:])

	h.SetHuntState(huntID, HuntStateActive)
	if h.GetHuntState(huntID) != HuntStateActive {
		t.Errorf("expected HuntStateActive, got %v", h.GetHuntState(huntID))
	}
	if h.HuntCount() != 1 {
		t.Errorf("expected 1 hunt, got %d", h.HuntCount())
	}

	h.SetHuntState(huntID, HuntStateExpiring)
	if h.GetHuntState(huntID) != HuntStateExpiring {
		t.Errorf("expected HuntStateExpiring, got %v", h.GetHuntState(huntID))
	}

	h.SetHuntState(huntID, HuntStateCompleted)
	if h.GetHuntState(huntID) != HuntStateCompleted {
		t.Errorf("expected HuntStateCompleted, got %v", h.GetHuntState(huntID))
	}
}

func TestHuntEffects_ClaimFragment(t *testing.T) {
	h := NewHuntEffects()

	var id, claimerKey [32]byte
	rand.Read(id[:])
	rand.Read(claimerKey[:])

	frag := &FragmentVisual{
		ID:    id,
		State: FragmentUnclaimed,
	}
	h.AddFragment(frag)

	h.ClaimFragment(id, claimerKey, nil)

	claimed := h.GetFragment(id)
	if claimed.State != FragmentClaimed {
		t.Errorf("expected FragmentClaimed, got %v", claimed.State)
	}
	if claimed.ClaimerKey != claimerKey {
		t.Error("claimer key mismatch")
	}
}

func TestHuntEffects_ClaimNonexistentFragment(t *testing.T) {
	h := NewHuntEffects()

	var id, claimerKey [32]byte
	rand.Read(id[:])
	rand.Read(claimerKey[:])

	// Should not panic.
	h.ClaimFragment(id, claimerKey, nil)
}

func TestHuntEffects_RevealClue(t *testing.T) {
	h := NewHuntEffects()

	var id [32]byte
	rand.Read(id[:])

	frag := &FragmentVisual{
		ID:        id,
		ClueLevel: 0,
	}
	h.AddFragment(frag)

	h.RevealClue(id)
	if h.GetFragment(id).ClueLevel != 1 {
		t.Errorf("expected clue level 1, got %d", h.GetFragment(id).ClueLevel)
	}

	h.RevealClue(id)
	h.RevealClue(id)
	if h.GetFragment(id).ClueLevel != 3 {
		t.Errorf("expected clue level 3, got %d", h.GetFragment(id).ClueLevel)
	}

	// Should cap at 3.
	h.RevealClue(id)
	if h.GetFragment(id).ClueLevel != 3 {
		t.Errorf("expected clue level to stay at 3, got %d", h.GetFragment(id).ClueLevel)
	}
}

func TestHuntEffects_RevealClueNonexistent(t *testing.T) {
	h := NewHuntEffects()

	var id [32]byte
	rand.Read(id[:])

	// Should not panic.
	h.RevealClue(id)
}

func TestHuntEffects_ClearHunt(t *testing.T) {
	h := NewHuntEffects()

	var huntID, otherHuntID [32]byte
	rand.Read(huntID[:])
	rand.Read(otherHuntID[:])

	// Add fragments for two hunts.
	for i := 0; i < 5; i++ {
		var id [32]byte
		rand.Read(id[:])
		h.AddFragment(&FragmentVisual{
			ID:     id,
			HuntID: huntID,
			Index:  i,
		})
	}
	for i := 0; i < 3; i++ {
		var id [32]byte
		rand.Read(id[:])
		h.AddFragment(&FragmentVisual{
			ID:     id,
			HuntID: otherHuntID,
			Index:  i,
		})
	}
	h.SetHuntState(huntID, HuntStateActive)
	h.SetHuntState(otherHuntID, HuntStateActive)

	if h.FragmentCount() != 8 {
		t.Errorf("expected 8 fragments, got %d", h.FragmentCount())
	}
	if h.HuntCount() != 2 {
		t.Errorf("expected 2 hunts, got %d", h.HuntCount())
	}

	// Clear one hunt.
	h.ClearHunt(huntID)

	if h.FragmentCount() != 3 {
		t.Errorf("expected 3 fragments after clear, got %d", h.FragmentCount())
	}
	if h.HuntCount() != 1 {
		t.Errorf("expected 1 hunt after clear, got %d", h.HuntCount())
	}
}

func TestHuntEffects_Update(t *testing.T) {
	h := NewHuntEffects()
	h.Update(0.016) // ~60fps frame
	h.Update(0.016)
	// No panic, time advances internally.
}

func TestFragmentState_Constants(t *testing.T) {
	// Verify constants are distinct.
	states := []FragmentState{FragmentUnclaimed, FragmentClaimed, FragmentExpired}
	seen := make(map[FragmentState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state value: %v", s)
		}
		seen[s] = true
	}
}

func TestHuntState_Constants(t *testing.T) {
	// Verify constants are distinct.
	states := []HuntState{HuntStateActive, HuntStateExpiring, HuntStateCompleted, HuntStateExpired}
	seen := make(map[HuntState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state value: %v", s)
		}
		seen[s] = true
	}
}

func TestHuntEffects_ConcurrentAccess(t *testing.T) {
	h := NewHuntEffects()

	done := make(chan bool)

	// Concurrent writes.
	go func() {
		for i := 0; i < 100; i++ {
			var id [32]byte
			rand.Read(id[:])
			h.AddFragment(&FragmentVisual{ID: id})
		}
		done <- true
	}()

	// Concurrent reads.
	go func() {
		for i := 0; i < 100; i++ {
			_ = h.FragmentCount()
		}
		done <- true
	}()

	// Concurrent state updates.
	go func() {
		for i := 0; i < 100; i++ {
			var huntID [32]byte
			rand.Read(huntID[:])
			h.SetHuntState(huntID, HuntState(i%4))
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func BenchmarkHuntEffects_AddFragment(b *testing.B) {
	h := NewHuntEffects()
	frags := make([]*FragmentVisual, b.N)
	for i := range frags {
		var id [32]byte
		rand.Read(id[:])
		frags[i] = &FragmentVisual{ID: id, Index: i}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.AddFragment(frags[i])
	}
}

func BenchmarkHuntEffects_GetFragment(b *testing.B) {
	h := NewHuntEffects()
	ids := make([][32]byte, 100)
	for i := range ids {
		rand.Read(ids[i][:])
		h.AddFragment(&FragmentVisual{ID: ids[i], Index: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.GetFragment(ids[i%100])
	}
}
