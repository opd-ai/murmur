// Package ui provides the Puzzle Solver panel for submitting solutions to Cipher Puzzles.
// Per ROADMAP.md line 419: "UI: puzzle solving interface — submit solution with feedback".
//

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PuzzleSolveCallback is called when a solution is submitted.
type PuzzleSolveCallback func(puzzleID [32]byte, solution string) (bool, string)

// PuzzleSolverPanel provides a UI for viewing and solving Cipher Puzzles.
type PuzzleSolverPanel struct {
	mu sync.RWMutex

	// Visibility and position.
	visible  bool
	x, y     int
	width    int
	height   int
	position PanelPosition

	// Current puzzle info.
	puzzleID     [32]byte
	puzzleType   PuzzleType
	difficulty   uint8
	expiresAt    time.Time
	participants int

	// Input state.
	solution      string
	cursorPos     int
	errorMessage  string
	successMsg    string
	feedbackTime  float64
	submitPending bool

	// Callbacks.
	onSubmit PuzzleSolveCallback

	// Styling.
	theme Theme

	// Animation.
	animTime    float64
	slideOffset float64

	// Screen dimensions.
	screenWidth, screenHeight int
}

// NewPuzzleSolverPanel creates a new puzzle solver panel.
func NewPuzzleSolverPanel(theme Theme, onSubmit PuzzleSolveCallback) *PuzzleSolverPanel {
	return &PuzzleSolverPanel{
		theme:    theme,
		onSubmit: onSubmit,
		width:    450,
		height:   400,
		position: PositionCenter,
	}
}

// Visible returns true if the panel is shown.
func (p *PuzzleSolverPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with animation.
func (p *PuzzleSolverPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.slideOffset = float64(p.height)
	p.animTime = 0
}

// Hide hides the panel.
func (p *PuzzleSolverPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	ResetPanelInputState(&p.visible, &p.solution, &p.errorMessage, &p.cursorPos)
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

// Update handles input and updates panel state.
func (p *PuzzleSolverPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	p.updateAnimations()
	p.handleTextInput()

	if p.handlePanelHotkeys() {
		return true
	}

	return true
}

// updateAnimations updates slide and feedback message animations.
func (p *PuzzleSolverPanel) updateAnimations() {
	p.animTime += 1.0 / 60.0
	if p.slideOffset > 0 {
		p.slideOffset *= 0.85
		if p.slideOffset < 1 {
			p.slideOffset = 0
		}
	}

	if p.errorMessage != "" || p.successMsg != "" {
		p.feedbackTime += 1.0 / 60.0
		if p.feedbackTime > 4.0 {
			p.errorMessage = ""
			p.successMsg = ""
			p.feedbackTime = 0
		}
	}
}

// handlePanelHotkeys processes Escape and Enter keys.
func (p *PuzzleSolverPanel) handlePanelHotkeys() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		p.submit()
		return true
	}

	return false
}

// handleTextInput processes keyboard input for the solution field.
func (p *PuzzleSolverPanel) handleTextInput() {
	p.handleCharacterInput()
	p.handleBackspace()
	p.handleDelete()
	p.handleCursorMovement()
}

// handleCharacterInput processes character key presses.
func (p *PuzzleSolverPanel) handleCharacterInput() {
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if len(p.solution) < 256 {
			p.insertCharAtCursor(ch)
		}
	}
}

// insertCharAtCursor inserts a character at the current cursor position.
func (p *PuzzleSolverPanel) insertCharAtCursor(ch rune) {
	p.solution, p.cursorPos = InsertRuneAtCursor(p.solution, p.cursorPos, ch)
}

// handleBackspace processes backspace key.
func (p *PuzzleSolverPanel) handleBackspace() {
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.KeyPressDuration(ebiten.KeyBackspace) > 20 {
		if p.cursorPos > 0 && len(p.solution) > 0 {
			runes := []rune(p.solution)
			if p.cursorPos <= len(runes) {
				p.solution = string(runes[:p.cursorPos-1]) + string(runes[p.cursorPos:])
				p.cursorPos--
			}
		}
	}
}

// handleDelete processes delete key.
func (p *PuzzleSolverPanel) handleDelete() {
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		runes := []rune(p.solution)
		if p.cursorPos < len(runes) {
			p.solution = string(runes[:p.cursorPos]) + string(runes[p.cursorPos+1:])
		}
	}
}

// handleCursorMovement processes arrow and home/end keys.
func (p *PuzzleSolverPanel) handleCursorMovement() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && p.cursorPos > 0 {
		p.cursorPos--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && p.cursorPos < len([]rune(p.solution)) {
		p.cursorPos++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		p.cursorPos = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		p.cursorPos = len([]rune(p.solution))
	}
}

// submit validates and submits the solution.
func (p *PuzzleSolverPanel) submit() {
	if err := p.validateSubmission(); err != nil {
		p.setError(err.Error())
		return
	}

	p.processSolution()
	p.clearSolution()
}

// validateSubmission checks if the solution can be submitted.
func (p *PuzzleSolverPanel) validateSubmission() error {
	if len(p.solution) == 0 {
		return fmt.Errorf("Solution cannot be empty")
	}
	if time.Now().After(p.expiresAt) {
		return fmt.Errorf("Puzzle has expired")
	}
	return nil
}

// setError sets an error message and resets feedback time.
func (p *PuzzleSolverPanel) setError(msg string) {
	p.errorMessage = msg
	p.feedbackTime = 0
}

// processSolution submits the solution and handles the result.
func (p *PuzzleSolverPanel) processSolution() {
	if p.onSubmit == nil {
		return
	}

	success, msg := p.onSubmit(p.puzzleID, p.solution)
	if success {
		p.setSuccessMessage(msg)
	} else {
		p.setErrorMessage(msg)
	}
	p.feedbackTime = 0
}

// setSuccessMessage sets the success message with a default fallback.
func (p *PuzzleSolverPanel) setSuccessMessage(msg string) {
	p.successMsg = msg
	if p.successMsg == "" {
		p.successMsg = "Solution accepted!"
	}
}

// setErrorMessage sets the error message with a default fallback.
func (p *PuzzleSolverPanel) setErrorMessage(msg string) {
	p.errorMessage = msg
	if p.errorMessage == "" {
		p.errorMessage = "Incorrect solution"
	}
}

// clearSolution resets the solution input.
func (p *PuzzleSolverPanel) clearSolution() {
	p.solution = ""
	p.cursorPos = 0
}

// Draw renders the panel to the screen.
func (p *PuzzleSolverPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ctx := InitPanelDrawWithScreen(screen, p.visible, p.calculatePosition, &p.screenWidth, &p.screenHeight)
	if ctx == nil {
		return
	}

	px := ctx.PanelX
	py := ctx.PanelY + int(p.slideOffset)

	p.drawBackground(screen, px, py)
	p.drawTitle(screen, px, py)
	p.drawPuzzleInfo(screen, px, py)
	p.drawSolutionField(screen, px, py)
	p.drawTimer(screen, px, py)
	p.drawButtons(screen, px, py)
	p.drawFeedback(screen, px, py)
}

// calculatePosition returns the panel's top-left corner.
func (p *PuzzleSolverPanel) calculatePosition(screenW, screenH int) (int, int) {
	margin := 20

	switch p.position {
	case PositionCenter:
		return (screenW - p.width) / 2, (screenH - p.height) / 2
	case PositionTopLeft:
		return margin, margin
	case PositionTopRight:
		return screenW - p.width - margin, margin
	case PositionBottomLeft:
		return margin, screenH - p.height - margin
	case PositionBottomRight:
		return screenW - p.width - margin, screenH - p.height - margin
	default:
		return (screenW - p.width) / 2, (screenH - p.height) / 2
	}
}

// drawBackground draws the panel background and border.
func (p *PuzzleSolverPanel) drawBackground(screen *ebiten.Image, px, py int) {
	shadowColor := color.RGBA{0, 0, 0, 60}
	vector.DrawFilledRect(screen, float32(px+4), float32(py+4),
		float32(p.width), float32(p.height), shadowColor, true)

	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)

	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 1.5, p.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (p *PuzzleSolverPanel) drawTitle(screen *ebiten.Image, px, py int) {
	titleHeight := 44
	titleBg := color.RGBA{
		R: p.theme.PanelBackground.R + 10,
		G: p.theme.PanelBackground.G + 10,
		B: p.theme.PanelBackground.B + 15,
		A: p.theme.PanelBackground.A,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(titleHeight), titleBg, true)
	// Title: "Solve Cipher Puzzle"
}

// drawPuzzleInfo draws puzzle details.
func (p *PuzzleSolverPanel) drawPuzzleInfo(screen *ebiten.Image, px, py int) {
	infoY := py + 54
	infoX := px + p.theme.Padding
	infoW := p.width - p.theme.Padding*2
	infoH := 80

	vector.DrawFilledRect(screen, float32(infoX), float32(infoY),
		float32(infoW), float32(infoH), p.theme.InputBackground, true)

	// Draw puzzle type indicator.
	typeX := infoX + 10
	typeY := infoY + 10
	var typeColor color.RGBA
	switch p.puzzleType {
	case PuzzleFragment:
		typeColor = color.RGBA{220, 100, 100, 255} // Red for competitive.
	case PuzzleMosaic:
		typeColor = color.RGBA{100, 180, 220, 255} // Blue for collaborative.
	case PuzzleCascade:
		typeColor = color.RGBA{180, 140, 220, 255} // Purple for sequential.
	}
	vector.DrawFilledCircle(screen, float32(typeX+8), float32(typeY+8), 8, typeColor, true)

	// Draw difficulty bar.
	diffY := infoY + 40
	diffX := infoX + 10
	diffW := infoW - 20
	barH := 8
	vector.DrawFilledRect(screen, float32(diffX), float32(diffY),
		float32(diffW), float32(barH), p.theme.ButtonBackground, true)

	// Fill based on difficulty (16-28 range).
	fillW := float64(diffW) * float64(p.difficulty-MinPuzzleDifficulty) / float64(MaxPuzzleDifficulty-MinPuzzleDifficulty)
	diffColor := color.RGBA{220, 160, 80, 255}
	if p.difficulty >= 24 {
		diffColor = color.RGBA{220, 100, 80, 255}
	}
	vector.DrawFilledRect(screen, float32(diffX), float32(diffY),
		float32(fillW), float32(barH), diffColor, true)

	// Draw participant count indicator.
	partY := infoY + 60
	partX := infoX + 10
	for i := 0; i < min(p.participants, 10); i++ {
		dotX := partX + i*14
		vector.DrawFilledCircle(screen, float32(dotX+5), float32(partY+5), 4, p.theme.AccentSecondary, true)
	}
}

// drawSolutionField draws the solution input area.
func (p *PuzzleSolverPanel) drawSolutionField(screen *ebiten.Image, px, py int) {
	fieldY := py + 145
	fieldX := px + p.theme.Padding
	fieldW := p.width - p.theme.Padding*2
	fieldH := 100

	vector.DrawFilledRect(screen, float32(fieldX), float32(fieldY),
		float32(fieldW), float32(fieldH), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(fieldX), float32(fieldY),
		float32(fieldW), float32(fieldH), 1.0, p.theme.PanelBorder, true)

	// Draw cursor (blinking).
	if int(p.animTime*2)%2 == 0 {
		cursorX := fieldX + 8 + p.cursorPos*8
		cursorY := fieldY + 10
		vector.DrawFilledRect(screen, float32(cursorX), float32(cursorY),
			2, 16, p.theme.TextPrimary, true)
	}
}

// drawTimer draws the countdown timer.
func (p *PuzzleSolverPanel) drawTimer(screen *ebiten.Image, px, py int) {
	timerY := py + 255
	timerX := px + p.theme.Padding
	timerW := p.width - p.theme.Padding*2
	timerH := 40

	remaining := time.Until(p.expiresAt)
	if remaining < 0 {
		remaining = 0
	}

	// Timer background.
	timerBg := p.theme.InputBackground
	if remaining < 5*time.Minute {
		timerBg = color.RGBA{80, 40, 40, 255} // Warning color.
	}
	vector.DrawFilledRect(screen, float32(timerX), float32(timerY),
		float32(timerW), float32(timerH), timerBg, true)

	// Timer progress bar.
	maxDuration := 60 * time.Minute // Assume max 60 min for progress.
	progress := float64(remaining) / float64(maxDuration)
	if progress > 1.0 {
		progress = 1.0
	}
	barW := float64(timerW-4) * progress
	barColor := p.theme.AccentPrimary
	if remaining < 5*time.Minute {
		barColor = color.RGBA{220, 80, 80, 255}
	}
	vector.DrawFilledRect(screen, float32(timerX+2), float32(timerY+timerH-8),
		float32(barW), 6, barColor, true)
}

// drawButtons draws submit and cancel buttons.
func (p *PuzzleSolverPanel) drawButtons(screen *ebiten.Image, px, py int) {
	enabled := len(p.solution) > 0
	DrawCancelSubmitButtons(screen, px, py, p.width, p.height, p.theme, 130, "Submit", enabled)
}

// drawFeedback draws success or error messages.
func (p *PuzzleSolverPanel) drawFeedback(screen *ebiten.Image, px, py int) {
	if p.errorMessage == "" && p.successMsg == "" {
		return
	}

	feedbackY := py + 305
	feedbackX := px + p.theme.Padding
	feedbackW := p.width - p.theme.Padding*2
	feedbackH := 30

	var bgColor color.RGBA
	if p.successMsg != "" {
		bgColor = color.RGBA{40, 80, 40, 220}
	} else {
		bgColor = color.RGBA{80, 40, 40, 220}
	}

	vector.DrawFilledRect(screen, float32(feedbackX), float32(feedbackY),
		float32(feedbackW), float32(feedbackH), bgColor, true)
}

// Getters for testing.

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

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
