// Package overlays - Tests for Cipher Puzzles overlay.
//

//go:build test
// +build test

package overlays

import (
	"testing"
	"time"
)

func TestNewPuzzlesOverlay(t *testing.T) {
	overlay := NewPuzzlesOverlay()
	if overlay == nil {
		t.Fatal("NewPuzzlesOverlay returned nil")
	}
}

func TestPuzzlesOverlay_SetVisible(t *testing.T) {
	overlay := NewPuzzlesOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("Expected IsVisible()=false after SetVisible(false)")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("Expected IsVisible()=true after SetVisible(true)")
	}
}

func TestPuzzlesOverlay_SetPuzzle(t *testing.T) {
	overlay := NewPuzzlesOverlay()

	puzzleID := [32]byte{1, 2, 3}
	puzzle := &PuzzleInfo{
		PuzzleID:  puzzleID,
		Type:      PuzzleFragment,
		State:     PuzzleActive,
		X:         100.0,
		Y:         200.0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
		Progress:  0.5,
	}

	overlay.SetPuzzle(puzzle)

	retrieved := overlay.GetPuzzle(puzzleID)
	if retrieved == nil {
		t.Error("GetPuzzle returned nil after SetPuzzle")
	}
}

func TestPuzzlesOverlay_RemovePuzzle(t *testing.T) {
	overlay := NewPuzzlesOverlay()

	puzzleID := [32]byte{4, 5, 6}
	puzzle := &PuzzleInfo{
		PuzzleID: puzzleID,
		Type:     PuzzleMosaic,
		State:    PuzzleSolved,
		X:        300.0,
		Y:        400.0,
	}

	overlay.SetPuzzle(puzzle)
	overlay.RemovePuzzle(puzzleID)

	retrieved := overlay.GetPuzzle(puzzleID)
	if retrieved != nil {
		t.Error("GetPuzzle should return nil after RemovePuzzle")
	}
}

func TestPuzzlesOverlay_GetAllPuzzles(t *testing.T) {
	overlay := NewPuzzlesOverlay()

	puzzle1 := &PuzzleInfo{
		PuzzleID: [32]byte{7, 8, 9},
		Type:     PuzzleFragment,
		State:    PuzzleActive,
		X:        500.0,
		Y:        600.0,
	}

	puzzle2 := &PuzzleInfo{
		PuzzleID: [32]byte{10, 11, 12},
		Type:     PuzzleCascade,
		State:    PuzzleExpired,
		X:        700.0,
		Y:        800.0,
	}

	overlay.SetPuzzle(puzzle1)
	overlay.SetPuzzle(puzzle2)

	all := overlay.GetAllPuzzles()
	if len(all) != 2 {
		t.Errorf("Expected 2 puzzles, got %d", len(all))
	}
}

func TestPuzzlesOverlay_Update(t *testing.T) {
	overlay := NewPuzzlesOverlay()

	// Should not panic.
	overlay.Update(0.016)
	overlay.Update(0.033)
}
