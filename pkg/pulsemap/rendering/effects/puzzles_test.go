package effects

import (
	"testing"
)

func TestPuzzleTypeConstants(t *testing.T) {
	// Verify puzzle types are distinct.
	types := []PuzzleType{PuzzleTypeFragment, PuzzleTypeMosaic, PuzzleTypeCascade}
	seen := make(map[PuzzleType]bool)
	for _, pt := range types {
		if seen[pt] {
			t.Errorf("duplicate puzzle type: %d", pt)
		}
		seen[pt] = true
	}

	// Verify non-zero values.
	for _, pt := range types {
		if pt == 0 {
			t.Error("puzzle type should not be zero")
		}
	}
}

func TestPuzzleStateConstants(t *testing.T) {
	// Verify puzzle states are distinct.
	states := []PuzzleState{PuzzleStateActive, PuzzleStateSolved, PuzzleStateExpired}
	seen := make(map[PuzzleState]bool)
	for _, ps := range states {
		if seen[ps] {
			t.Errorf("duplicate puzzle state: %d", ps)
		}
		seen[ps] = true
	}
}

func TestNewPuzzleEffects(t *testing.T) {
	pe := NewPuzzleEffects()
	if pe == nil {
		t.Fatal("NewPuzzleEffects returned nil")
	}
	if pe.Count() != 0 {
		t.Errorf("expected 0 puzzles, got %d", pe.Count())
	}
}

func TestPuzzleEffects_AddRemove(t *testing.T) {
	pe := NewPuzzleEffects()

	var id1 [32]byte
	copy(id1[:], []byte("puzzle-id-one-1234567890123"))
	pv1 := &PuzzleVisual{
		ID:    id1,
		X:     100,
		Y:     200,
		Type:  PuzzleTypeFragment,
		State: PuzzleStateActive,
	}

	pe.AddPuzzle(pv1)
	if pe.Count() != 1 {
		t.Errorf("expected 1 puzzle, got %d", pe.Count())
	}

	var id2 [32]byte
	copy(id2[:], []byte("puzzle-id-two-1234567890123"))
	pv2 := &PuzzleVisual{
		ID:    id2,
		X:     300,
		Y:     400,
		Type:  PuzzleTypeMosaic,
		State: PuzzleStateActive,
	}

	pe.AddPuzzle(pv2)
	if pe.Count() != 2 {
		t.Errorf("expected 2 puzzles, got %d", pe.Count())
	}

	pe.RemovePuzzle(id1)
	if pe.Count() != 1 {
		t.Errorf("expected 1 puzzle after removal, got %d", pe.Count())
	}

	pe.RemovePuzzle(id2)
	if pe.Count() != 0 {
		t.Errorf("expected 0 puzzles after removal, got %d", pe.Count())
	}
}

func TestPuzzleEffects_Update(t *testing.T) {
	pe := NewPuzzleEffects()

	var id [32]byte
	copy(id[:], []byte("puzzle-id-update-123456789"))
	pv := &PuzzleVisual{
		ID:       id,
		X:        100,
		Y:        200,
		Type:     PuzzleTypeMosaic,
		State:    PuzzleStateActive,
		Progress: 0.0,
	}

	pe.AddPuzzle(pv)
	pe.UpdatePuzzle(id, PuzzleStateSolved, 1.0)

	// Access via the stored reference.
	pe.mu.RLock()
	stored := pe.puzzles[id]
	pe.mu.RUnlock()

	if stored.State != PuzzleStateSolved {
		t.Errorf("expected state Solved, got %d", stored.State)
	}
	if stored.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", stored.Progress)
	}
}

func TestPuzzleEffects_UpdateNonExistent(t *testing.T) {
	pe := NewPuzzleEffects()

	var id [32]byte
	copy(id[:], []byte("nonexistent-puzzle-id-12345"))

	// Should not panic.
	pe.UpdatePuzzle(id, PuzzleStateSolved, 1.0)
}

func TestPuzzleEffects_Clear(t *testing.T) {
	pe := NewPuzzleEffects()

	for i := 0; i < 5; i++ {
		var id [32]byte
		id[0] = byte(i)
		pv := &PuzzleVisual{
			ID:    id,
			X:     float32(i * 100),
			Y:     float32(i * 100),
			Type:  PuzzleType((i % 3) + 1),
			State: PuzzleStateActive,
		}
		pe.AddPuzzle(pv)
	}

	if pe.Count() != 5 {
		t.Errorf("expected 5 puzzles, got %d", pe.Count())
	}

	pe.Clear()

	if pe.Count() != 0 {
		t.Errorf("expected 0 puzzles after clear, got %d", pe.Count())
	}
}

func TestPuzzleEffects_AnimationUpdate(t *testing.T) {
	pe := NewPuzzleEffects()

	// Call Update multiple times - should not panic.
	for i := 0; i < 100; i++ {
		pe.Update(0.016) // ~60fps delta time.
	}
}

func TestPuzzleVisual_AllTypes(t *testing.T) {
	pe := NewPuzzleEffects()

	types := []PuzzleType{PuzzleTypeFragment, PuzzleTypeMosaic, PuzzleTypeCascade}
	states := []PuzzleState{PuzzleStateActive, PuzzleStateSolved, PuzzleStateExpired}

	for i, pt := range types {
		for j, ps := range states {
			var id [32]byte
			id[0] = byte(i)
			id[1] = byte(j)

			pv := &PuzzleVisual{
				ID:       id,
				X:        float32(i*100 + j*10),
				Y:        float32(i*100 + j*10),
				Type:     pt,
				State:    ps,
				Progress: float32(j) / 3.0,
			}
			pe.AddPuzzle(pv)
		}
	}

	expected := len(types) * len(states)
	if pe.Count() != expected {
		t.Errorf("expected %d puzzles, got %d", expected, pe.Count())
	}
}

func TestPuzzleEffects_ConcurrentAccess(t *testing.T) {
	pe := NewPuzzleEffects()

	done := make(chan bool)

	// Writer goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			var id [32]byte
			id[0] = byte(i)
			pv := &PuzzleVisual{
				ID:    id,
				X:     float32(i),
				Y:     float32(i),
				Type:  PuzzleType((i % 3) + 1),
				State: PuzzleStateActive,
			}
			pe.AddPuzzle(pv)
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			_ = pe.Count()
		}
		done <- true
	}()

	// Updater goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			pe.Update(0.016)
		}
		done <- true
	}()

	// Wait for all goroutines.
	for i := 0; i < 3; i++ {
		<-done
	}
}

func BenchmarkPuzzleEffects_Update(b *testing.B) {
	pe := NewPuzzleEffects()

	// Add some puzzles.
	for i := 0; i < 20; i++ {
		var id [32]byte
		id[0] = byte(i)
		pv := &PuzzleVisual{
			ID:    id,
			X:     float32(i * 50),
			Y:     float32(i * 50),
			Type:  PuzzleType((i % 3) + 1),
			State: PuzzleStateActive,
		}
		pe.AddPuzzle(pv)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pe.Update(0.016)
	}
}

func BenchmarkPuzzleEffects_AddRemove(b *testing.B) {
	pe := NewPuzzleEffects()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id [32]byte
		id[0] = byte(i & 0xFF)
		id[1] = byte((i >> 8) & 0xFF)

		pv := &PuzzleVisual{
			ID:    id,
			X:     100,
			Y:     200,
			Type:  PuzzleTypeFragment,
			State: PuzzleStateActive,
		}
		pe.AddPuzzle(pv)
		pe.RemovePuzzle(id)
	}
}
