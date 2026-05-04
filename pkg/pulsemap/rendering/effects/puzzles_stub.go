// Package effects provides puzzle visual effects stubs for non-Ebitengine builds.
//
//go:build test
// +build test

package effects

import "sync"

// PuzzleType identifies the visual style for a puzzle.
type PuzzleType int

const (
	PuzzleTypeFragment PuzzleType = iota + 1
	PuzzleTypeMosaic
	PuzzleTypeCascade
)

// PuzzleState determines the visual indicator overlay.
type PuzzleState int

const (
	PuzzleStateActive PuzzleState = iota
	PuzzleStateSolved
	PuzzleStateExpired
)

// PuzzleVisual represents a puzzle to be rendered on the Pulse Map.
type PuzzleVisual struct {
	ID       [32]byte
	X, Y     float32
	Type     PuzzleType
	State    PuzzleState
	Progress float32
}

// PuzzleEffects is a stub for non-Ebitengine builds.
type PuzzleEffects struct {
	mu      sync.RWMutex
	puzzles map[[32]byte]*PuzzleVisual
}

// NewPuzzleEffects creates a new puzzle effects renderer (stub).
func NewPuzzleEffects() *PuzzleEffects {
	return &PuzzleEffects{
		puzzles: make(map[[32]byte]*PuzzleVisual),
	}
}

// AddPuzzle adds a puzzle to be rendered (stub).
func (p *PuzzleEffects) AddPuzzle(pv *PuzzleVisual) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.puzzles[pv.ID] = pv
}

// RemovePuzzle removes a puzzle from rendering (stub).
func (p *PuzzleEffects) RemovePuzzle(id [32]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.puzzles, id)
}

// UpdatePuzzle updates a puzzle's state (stub).
func (p *PuzzleEffects) UpdatePuzzle(id [32]byte, state PuzzleState, progress float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pv, ok := p.puzzles[id]; ok {
		pv.State = state
		pv.Progress = progress
	}
}

// Update advances animation state (stub).
func (p *PuzzleEffects) Update(deltaTime float32) {}

// Draw renders all puzzle effects (stub).
func (p *PuzzleEffects) Draw(dst interface{}) {}

// Clear removes all puzzles (stub).
func (p *PuzzleEffects) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.puzzles = make(map[[32]byte]*PuzzleVisual)
}

// Count returns the number of active puzzles being rendered (stub).
func (p *PuzzleEffects) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.puzzles)
}
