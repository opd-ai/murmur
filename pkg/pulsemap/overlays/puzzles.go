// Package overlays - Cipher Puzzles Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Active Cipher Puzzles appear on the Pulse Map
// as rotating cryptographic symbols. Fragment puzzles show a rotating hexagon,
// Mosaic puzzles show interlocking pieces, and Cascade puzzles show flowing symbols".
// Per ROADMAP.md line 638: "Cipher Puzzles — rotating cryptographic symbol".
//

//go:build !test
// +build !test

package overlays

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering/effects"
)

// PuzzleInfo contains information about a Cipher Puzzle event.
type PuzzleInfo struct {
	PuzzleID  [32]byte    // Unique puzzle identifier.
	Type      PuzzleType  // Type of puzzle (Fragment/Mosaic/Cascade).
	State     PuzzleState // Current state (Active/Solved/Expired).
	X, Y      float64     // Position on the Pulse Map (world coordinates).
	StartTime time.Time   // When the puzzle was created.
	EndTime   time.Time   // When the puzzle expires.
	Progress  float32     // 0.0-1.0 for Mosaic contributions.
}

// PuzzleType represents the type of Cipher Puzzle.
type PuzzleType uint8

const (
	PuzzleFragment PuzzleType = iota // Fragment - rotating hexagon.
	PuzzleMosaic                     // Mosaic - interlocking pieces.
	PuzzleCascade                    // Cascade - flowing waterfall.
)

// PuzzleState represents the current state of a puzzle.
type PuzzleState uint8

const (
	PuzzleActive  PuzzleState = iota // Active - accepting solutions.
	PuzzleSolved                     // Solved - green checkmark overlay.
	PuzzleExpired                    // Expired - faded gray.
)

// PuzzlesOverlay renders Cipher Puzzle events on the Pulse Map.
type PuzzlesOverlay struct {
	mu sync.RWMutex

	visible bool
	puzzles map[[32]byte]*PuzzleInfo

	// Effects renderer for actual drawing.
	effects *effects.PuzzleEffects
}

// NewPuzzlesOverlay creates a new Cipher Puzzles overlay.
func NewPuzzlesOverlay() *PuzzlesOverlay {
	return &PuzzlesOverlay{
		visible: true,
		puzzles: make(map[[32]byte]*PuzzleInfo),
		effects: effects.NewPuzzleEffects(),
	}
}

// SetVisible controls visibility.
func (o *PuzzlesOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *PuzzlesOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetPuzzle adds or updates a puzzle event.
func (o *PuzzlesOverlay) SetPuzzle(puzzle *PuzzleInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.puzzles[puzzle.PuzzleID] = puzzle

	// Update effects renderer.
	o.effects.AddPuzzle(&effects.PuzzleVisual{
		ID:       puzzle.PuzzleID,
		X:        0, // Will be set in Draw with screen coordinates.
		Y:        0,
		Type:     mapPuzzleType(puzzle.Type),
		State:    mapPuzzleState(puzzle.State),
		Progress: puzzle.Progress,
	})
}

// RemovePuzzle removes a puzzle event.
func (o *PuzzlesOverlay) RemovePuzzle(puzzleID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.puzzles, puzzleID)
	o.effects.RemovePuzzle(puzzleID)
}

// GetPuzzle returns a puzzle event by ID.
func (o *PuzzlesOverlay) GetPuzzle(puzzleID [32]byte) *PuzzleInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.puzzles[puzzleID]
}

// GetAllPuzzles returns all active puzzle events.
func (o *PuzzlesOverlay) GetAllPuzzles() []*PuzzleInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	puzzles := make([]*PuzzleInfo, 0, len(o.puzzles))
	for _, p := range o.puzzles {
		puzzles = append(puzzles, p)
	}
	return puzzles
}

// Update advances animation state.
func (o *PuzzlesOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.effects.Update(float32(dt))
}

// Draw renders the puzzle events.
func (o *PuzzlesOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	screenW, screenH, centerX, centerY := getCameraSetup(screen)

	// Update puzzle positions in effects renderer with screen coordinates.
	for _, puzzle := range o.puzzles {
		sx, sy := worldToScreen(puzzle.X, puzzle.Y, cameraX, cameraY, centerX, centerY, zoom)

		// Skip if off-screen.
		if isOffScreen(sx, sy, screenW, screenH, 100) {
			continue
		}

		// Update the puzzle visual with current screen position.
		o.effects.AddPuzzle(&effects.PuzzleVisual{
			ID:       puzzle.PuzzleID,
			X:        float32(sx),
			Y:        float32(sy),
			Type:     mapPuzzleType(puzzle.Type),
			State:    mapPuzzleState(puzzle.State),
			Progress: puzzle.Progress,
		})
	}

	// Delegate rendering to effects.
	o.effects.Draw(screen)
}

// mapPuzzleType converts overlay PuzzleType to effects PuzzleType.
func mapPuzzleType(t PuzzleType) effects.PuzzleType {
	switch t {
	case PuzzleFragment:
		return effects.PuzzleTypeFragment
	case PuzzleMosaic:
		return effects.PuzzleTypeMosaic
	case PuzzleCascade:
		return effects.PuzzleTypeCascade
	default:
		return effects.PuzzleTypeFragment
	}
}

// mapPuzzleState converts overlay PuzzleState to effects PuzzleState.
func mapPuzzleState(s PuzzleState) effects.PuzzleState {
	switch s {
	case PuzzleActive:
		return effects.PuzzleStateActive
	case PuzzleSolved:
		return effects.PuzzleStateSolved
	case PuzzleExpired:
		return effects.PuzzleStateExpired
	default:
		return effects.PuzzleStateActive
	}
}
