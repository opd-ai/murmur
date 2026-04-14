// Package ui provides stub types for the Puzzle Solver panel.
// Per ROADMAP.md line 419: "UI: puzzle solving interface — submit solution with feedback".
//
//go:build noebiten
// +build noebiten

package ui

import (
	"sync"
	"time"
)

// PuzzleSolveCallback is called when a solution is submitted.
type PuzzleSolveCallback func(puzzleID [32]byte, solution string) (bool, string)

// PuzzleSolverPanel provides a UI for viewing and solving Cipher Puzzles (stub).
type PuzzleSolverPanel struct {
	mu sync.RWMutex

	visible      bool
	puzzleID     [32]byte
	puzzleType   PuzzleType
	difficulty   uint8
	expiresAt    time.Time
	participants int
	solution     string
	cursorPos    int
	errorMessage string
	successMsg   string
	onSubmit     PuzzleSolveCallback
	theme        Theme
}

// NewPuzzleSolverPanel creates a new puzzle solver panel (stub).
func NewPuzzleSolverPanel(theme Theme, onSubmit PuzzleSolveCallback) *PuzzleSolverPanel {
	return &PuzzleSolverPanel{
		theme:    theme,
		onSubmit: onSubmit,
	}
}

// Visible returns true if the panel is shown.
func (p *PuzzleSolverPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *PuzzleSolverPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *PuzzleSolverPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.solution = ""
	p.cursorPos = 0
	p.errorMessage = ""
	p.successMsg = ""
}

// Toggle toggles panel visibility.
func (p *PuzzleSolverPanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetPuzzle sets the puzzle to be solved.
func (p *PuzzleSolverPanel) SetPuzzle(id [32]byte, puzzleType PuzzleType, difficulty uint8, expiresAt time.Time, participants int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.puzzleID = id
	p.puzzleType = puzzleType
	p.difficulty = difficulty
	p.expiresAt = expiresAt
	p.participants = participants
	p.solution = ""
	p.cursorPos = 0
	p.errorMessage = ""
	p.successMsg = ""
}

// Update handles input and updates panel state (stub).
func (p *PuzzleSolverPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
}

// Submit triggers the submit callback (for testing).
func (p *PuzzleSolverPanel) Submit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.solution) == 0 {
		p.errorMessage = "Solution cannot be empty"
		return
	}

	if time.Now().After(p.expiresAt) {
		p.errorMessage = "Puzzle has expired"
		return
	}

	if p.onSubmit != nil {
		success, msg := p.onSubmit(p.puzzleID, p.solution)
		if success {
			p.successMsg = msg
			if p.successMsg == "" {
				p.successMsg = "Solution accepted!"
			}
		} else {
			p.errorMessage = msg
			if p.errorMessage == "" {
				p.errorMessage = "Incorrect solution"
			}
		}
	}

	p.solution = ""
	p.cursorPos = 0
}

// GetSolution returns the current solution text.
func (p *PuzzleSolverPanel) GetSolution() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.solution
}

// SetSolution sets the solution text.
func (p *PuzzleSolverPanel) SetSolution(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(s) > 256 {
		s = s[:256]
	}
	p.solution = s
	p.cursorPos = len([]rune(s))
}

// GetPuzzleID returns the current puzzle ID.
func (p *PuzzleSolverPanel) GetPuzzleID() [32]byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.puzzleID
}

// GetErrorMessage returns the current error message.
func (p *PuzzleSolverPanel) GetErrorMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorMessage
}

// GetSuccessMessage returns the current success message.
func (p *PuzzleSolverPanel) GetSuccessMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.successMsg
}
