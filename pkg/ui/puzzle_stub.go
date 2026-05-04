// Package ui provides stub types for the Puzzle panel.
// Per ROADMAP.md line 418: "UI: puzzle composition panel — create puzzle with
// difficulty and content inputs".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
)

// PuzzleType identifies the type of puzzle (mirrors mechanics.PuzzleType).
type PuzzleType uint8

const (
	// PuzzleFragment is competitive - first solver wins.
	PuzzleFragment PuzzleType = iota + 1
	// PuzzleMosaic is collaborative - multiple contributors.
	PuzzleMosaic
	// PuzzleCascade is sequential - chain of puzzles.
	PuzzleCascade
)

// PuzzleDuration options per ANONYMOUS_GAME_MECHANICS.md.
const (
	PuzzleDuration15Min = 15 * time.Minute
	PuzzleDuration30Min = 30 * time.Minute
	PuzzleDuration60Min = 60 * time.Minute
)

// Difficulty bounds.
const (
	MinPuzzleDifficulty     = 16
	MaxPuzzleDifficulty     = 28
	DefaultPuzzleDifficulty = 20
)

// PuzzleSubmitCallback is called when a puzzle is composed and submitted.
type PuzzleSubmitCallback func(puzzleType PuzzleType, difficulty uint8, duration time.Duration, seed string)

// PuzzlePanel provides a UI for composing and submitting Cipher Puzzles (stub).
type PuzzlePanel struct {
	mu sync.RWMutex

	visible     bool
	puzzleType  PuzzleType
	difficulty  uint8
	durationIdx int
	seed        string
	onSubmit    PuzzleSubmitCallback
	theme       Theme
}

// Available durations.
var puzzleDurations = []time.Duration{
	PuzzleDuration15Min,
	PuzzleDuration30Min,
	PuzzleDuration60Min,
}

// NewPuzzlePanel creates a new Cipher Puzzle composition panel (stub).
func NewPuzzlePanel(theme Theme, onSubmit PuzzleSubmitCallback) *PuzzlePanel {
	return &PuzzlePanel{
		theme:       theme,
		onSubmit:    onSubmit,
		puzzleType:  PuzzleFragment,
		difficulty:  DefaultPuzzleDifficulty,
		durationIdx: 1,
	}
}

// Visible returns true if the panel is shown.
func (p *PuzzlePanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *PuzzlePanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *PuzzlePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.seed = ""
}

// Toggle toggles panel visibility.
func (p *PuzzlePanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// Update handles input and updates panel state (stub).
func (p *PuzzlePanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
}

// Submit triggers the submit callback (for testing).
func (p *PuzzlePanel) Submit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.onSubmit != nil {
		p.onSubmit(p.puzzleType, p.difficulty, puzzleDurations[p.durationIdx], p.seed)
	}

	p.seed = ""
	p.visible = false
}

// GetPuzzleType returns the currently selected puzzle type.
func (p *PuzzlePanel) GetPuzzleType() PuzzleType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.puzzleType
}

// SetPuzzleType sets the puzzle type.
func (p *PuzzlePanel) SetPuzzleType(t PuzzleType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if t >= PuzzleFragment && t <= PuzzleCascade {
		p.puzzleType = t
	}
}

// GetDifficulty returns the currently selected difficulty.
func (p *PuzzlePanel) GetDifficulty() uint8 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.difficulty
}

// SetDifficulty sets the difficulty level.
func (p *PuzzlePanel) SetDifficulty(d uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if d >= MinPuzzleDifficulty && d <= MaxPuzzleDifficulty {
		p.difficulty = d
	}
}

// GetDuration returns the currently selected duration.
func (p *PuzzlePanel) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return puzzleDurations[p.durationIdx]
}

// SetDurationIndex sets the duration by index.
func (p *PuzzlePanel) SetDurationIndex(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if idx >= 0 && idx < len(puzzleDurations) {
		p.durationIdx = idx
	}
}

// GetSeed returns the optional seed value.
func (p *PuzzlePanel) GetSeed() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.seed
}

// SetSeed sets the seed value.
func (p *PuzzlePanel) SetSeed(seed string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(seed) > 64 {
		seed = seed[:64]
	}
	p.seed = seed
}
