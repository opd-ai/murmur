// Package overlays - Stub for Cipher Puzzles overlay (test build).
//

//go:build test
// +build test

package overlays

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type PuzzleInfo struct {
	PuzzleID  [32]byte
	Type      PuzzleType
	State     PuzzleState
	X, Y      float64
	StartTime time.Time
	EndTime   time.Time
	Progress  float32
}

type PuzzleType uint8

const (
	PuzzleFragment PuzzleType = iota
	PuzzleMosaic
	PuzzleCascade
)

type PuzzleState uint8

const (
	PuzzleActive PuzzleState = iota
	PuzzleSolved
	PuzzleExpired
)

type PuzzlesOverlay struct {
	visible bool
	puzzles map[[32]byte]*PuzzleInfo
}

func NewPuzzlesOverlay() *PuzzlesOverlay {
	return &PuzzlesOverlay{
		visible: true,
		puzzles: make(map[[32]byte]*PuzzleInfo),
	}
}

func (o *PuzzlesOverlay) SetVisible(visible bool) {
	o.visible = visible
}

func (o *PuzzlesOverlay) IsVisible() bool {
	return o.visible
}

func (o *PuzzlesOverlay) SetPuzzle(puzzle *PuzzleInfo) {
	o.puzzles[puzzle.PuzzleID] = puzzle
}

func (o *PuzzlesOverlay) RemovePuzzle(puzzleID [32]byte) {
	delete(o.puzzles, puzzleID)
}

func (o *PuzzlesOverlay) GetPuzzle(puzzleID [32]byte) *PuzzleInfo {
	return o.puzzles[puzzleID]
}

func (o *PuzzlesOverlay) GetAllPuzzles() []*PuzzleInfo {
	puzzles := make([]*PuzzleInfo, 0, len(o.puzzles))
	for _, p := range o.puzzles {
		puzzles = append(puzzles, p)
	}
	return puzzles
}

func (o *PuzzlesOverlay) Update(dt float64) {}

func (o *PuzzlesOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {}
