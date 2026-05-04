// Package ui provides the Puzzle composition panel for creating Cipher Puzzles.
// Per ROADMAP.md line 418: "UI: puzzle composition panel — create puzzle with
// difficulty and content inputs".
//

//go:build !test
// +build !test

package ui

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// PuzzlePanel provides a UI for composing and submitting Cipher Puzzles.
// Per ANONYMOUS_GAME_MECHANICS.md, puzzles require Resonance ≥50 to create.
type PuzzlePanel struct {
	mu sync.RWMutex

	// Visibility and position.
	visible  bool
	x, y     int
	width    int
	height   int
	position PanelPosition

	// Input state.
	puzzleType   PuzzleType
	difficulty   uint8
	durationIdx  int // Index into duration options.
	seed         string
	errorMessage string
	errorTime    float64

	// Selection state.
	selectedField int // 0=type, 1=difficulty, 2=duration, 3=seed.
	seedCursorPos int

	// Callbacks.
	onSubmit PuzzleSubmitCallback

	// Styling.
	theme Theme

	// Animation.
	animTime    float64
	slideOffset float64

	// Screen dimensions (updated each frame).
	screenWidth, screenHeight int
}

// Available durations.
var puzzleDurations = []time.Duration{
	PuzzleDuration15Min,
	PuzzleDuration30Min,
	PuzzleDuration60Min,
}

// NewPuzzlePanel creates a new Cipher Puzzle composition panel.
func NewPuzzlePanel(theme Theme, onSubmit PuzzleSubmitCallback) *PuzzlePanel {
	return &PuzzlePanel{
		theme:       theme,
		onSubmit:    onSubmit,
		width:       420,
		height:      380,
		position:    PositionCenter,
		puzzleType:  PuzzleFragment,
		difficulty:  DefaultPuzzleDifficulty,
		durationIdx: 1, // Default 30 minutes.
	}
}

// Visible returns true if the panel is shown.
func (p *PuzzlePanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with animation.
func (p *PuzzlePanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.slideOffset = float64(p.height)
	p.animTime = 0
	p.selectedField = 0
}

// Hide hides the panel.
func (p *PuzzlePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	ResetPanelInputState(&p.visible, &p.seed, &p.errorMessage, &p.seedCursorPos)
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

// Update handles input and updates panel state.
// Returns true if the panel consumed the input.
func (p *PuzzlePanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Animate slide-in.
	p.animTime += 1.0 / 60.0
	if p.slideOffset > 0 {
		p.slideOffset *= 0.85
		if p.slideOffset < 1 {
			p.slideOffset = 0
		}
	}

	// Clear error after 3 seconds.
	if p.errorMessage != "" {
		p.errorTime += 1.0 / 60.0
		if p.errorTime > 3.0 {
			p.errorMessage = ""
			p.errorTime = 0
		}
	}

	// Handle field navigation.
	p.handleFieldNavigation()

	// Handle field-specific input.
	p.handleFieldInput()

	// Handle escape to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	// Handle enter to submit.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		p.submit()
		return true
	}

	return true // Panel consumes all input when visible.
}

// handleFieldNavigation processes Tab/Arrow keys to move between fields.
func (p *PuzzlePanel) handleFieldNavigation() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		p.handleTabNavigation()
	}
	if p.selectedField != 3 {
		p.handleArrowNavigation()
	}
}

// handleTabNavigation processes Tab/Shift+Tab field cycling.
func (p *PuzzlePanel) handleTabNavigation() {
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		p.moveToPreviousField()
	} else {
		p.moveToNextField()
	}
}

// moveToPreviousField moves selection to previous field (wrap around).
func (p *PuzzlePanel) moveToPreviousField() {
	p.selectedField--
	if p.selectedField < 0 {
		p.selectedField = 3
	}
}

// moveToNextField moves selection to next field (wrap around).
func (p *PuzzlePanel) moveToNextField() {
	p.selectedField++
	if p.selectedField > 3 {
		p.selectedField = 0
	}
}

// handleArrowNavigation processes Up/Down arrow navigation (no wrap).
func (p *PuzzlePanel) handleArrowNavigation() {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if p.selectedField > 0 {
			p.selectedField--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if p.selectedField < 3 {
			p.selectedField++
		}
	}
}

// handleFieldInput processes input for the currently selected field.
func (p *PuzzlePanel) handleFieldInput() {
	switch p.selectedField {
	case 0: // Puzzle type.
		p.handlePuzzleTypeInput()
	case 1: // Difficulty.
		p.handleDifficultyInput()
	case 2: // Duration.
		p.handleDurationInput()
	case 3: // Seed (text input).
		p.handleSeedInput()
	}
}

// handlePuzzleTypeInput handles left/right to change puzzle type.
func (p *PuzzlePanel) handlePuzzleTypeInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if p.puzzleType > PuzzleFragment {
			p.puzzleType--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if p.puzzleType < PuzzleCascade {
			p.puzzleType++
		}
	}
}

// handleDifficultyInput handles left/right to adjust difficulty.
func (p *PuzzlePanel) handleDifficultyInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if p.difficulty > MinPuzzleDifficulty {
			p.difficulty--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if p.difficulty < MaxPuzzleDifficulty {
			p.difficulty++
		}
	}
}

// handleDurationInput handles left/right to change duration.
func (p *PuzzlePanel) handleDurationInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if p.durationIdx > 0 {
			p.durationIdx--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if p.durationIdx < len(puzzleDurations)-1 {
			p.durationIdx++
		}
	}
}

// handleSeedInput handles text input for the seed field.
func (p *PuzzlePanel) handleSeedInput() {
	p.handleCharacterInput()
	p.handleBackspace()
	p.handleCursorMovement()
}

// handleCharacterInput processes character input for the seed field.
func (p *PuzzlePanel) handleCharacterInput() {
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if len(p.seed) >= 64 {
			continue
		}
		runes := []rune(p.seed)
		newRunes := make([]rune, 0, len(runes)+1)
		newRunes = append(newRunes, runes[:p.seedCursorPos]...)
		newRunes = append(newRunes, ch)
		newRunes = append(newRunes, runes[p.seedCursorPos:]...)
		p.seed = string(newRunes)
		p.seedCursorPos++
	}
}

// handleBackspace processes backspace key for seed editing.
func (p *PuzzlePanel) handleBackspace() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) <= 20 {
		return
	}
	if p.seedCursorPos > 0 && len(p.seed) > 0 {
		runes := []rune(p.seed)
		if p.seedCursorPos <= len(runes) {
			p.seed = string(runes[:p.seedCursorPos-1]) + string(runes[p.seedCursorPos:])
			p.seedCursorPos--
		}
	}
}

// handleCursorMovement processes left/right arrow keys for seed cursor.
func (p *PuzzlePanel) handleCursorMovement() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && p.seedCursorPos > 0 {
		p.seedCursorPos--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && p.seedCursorPos < len([]rune(p.seed)) {
		p.seedCursorPos++
	}
}

// submit validates and submits the puzzle.
func (p *PuzzlePanel) submit() {
	if p.onSubmit != nil {
		p.onSubmit(p.puzzleType, p.difficulty, puzzleDurations[p.durationIdx], p.seed)
	}

	// Reset and hide after submit.
	p.seed = ""
	p.seedCursorPos = 0
	p.visible = false
}

// Draw renders the panel to the screen.
func (p *PuzzlePanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ctx := InitPanelDrawWithScreen(screen, p.visible, p.calculatePosition, &p.screenWidth, &p.screenHeight)
	if ctx == nil {
		return
	}

	px := ctx.PanelX
	py := ctx.PanelY + int(p.slideOffset)

	// Draw panel background with border.
	p.drawBackground(screen, px, py)

	// Draw title.
	p.drawTitle(screen, px, py)

	// Draw fields.
	p.drawPuzzleTypeField(screen, px, py)
	p.drawDifficultyField(screen, px, py)
	p.drawDurationField(screen, px, py)
	p.drawSeedField(screen, px, py)

	// Draw buttons.
	p.drawButtons(screen, px, py)

	// Draw error message if present.
	if p.errorMessage != "" {
		p.drawError(screen, px, py)
	}
}

// calculatePosition returns the panel's top-left corner based on position setting.
func (p *PuzzlePanel) calculatePosition(screenW, screenH int) (int, int) {
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
func (p *PuzzlePanel) drawBackground(screen *ebiten.Image, px, py int) {
	// Draw shadow.
	shadowColor := color.RGBA{0, 0, 0, 60}
	vector.DrawFilledRect(screen, float32(px+4), float32(py+4),
		float32(p.width), float32(p.height), shadowColor, true)

	// Draw background.
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)

	// Draw border.
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 1.5, p.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (p *PuzzlePanel) drawTitle(screen *ebiten.Image, px, py int) {
	titleHeight := 44
	titleBg := color.RGBA{
		R: p.theme.PanelBackground.R + 10,
		G: p.theme.PanelBackground.G + 10,
		B: p.theme.PanelBackground.B + 15,
		A: p.theme.PanelBackground.A,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(titleHeight), titleBg, true)

	// Title text rendering requires font setup.
	// Title: "Create Cipher Puzzle"
}

// drawInputFieldBackground draws the background rect for an input field with selection highlighting.
func (p *PuzzlePanel) drawInputFieldBackground(screen *ebiten.Image, px, py, yOffset, fieldIndex int) (fieldX, fieldY, fieldW, fieldH int) {
	fieldY = py + yOffset
	fieldX = px + p.theme.Padding
	fieldW = p.width - p.theme.Padding*2
	fieldH = 50

	bgColor := p.theme.InputBackground
	if p.selectedField == fieldIndex {
		bgColor = color.RGBA{
			R: p.theme.AccentPrimary.R,
			G: p.theme.AccentPrimary.G,
			B: p.theme.AccentPrimary.B,
			A: 40,
		}
	}

	vector.DrawFilledRect(screen, float32(fieldX), float32(fieldY),
		float32(fieldW), float32(fieldH), bgColor, true)
	vector.StrokeRect(screen, float32(fieldX), float32(fieldY),
		float32(fieldW), float32(fieldH), 1.0, p.theme.PanelBorder, true)
	return fieldX, fieldY, fieldW, fieldH
}

// drawPuzzleTypeField draws the puzzle type selector.
func (p *PuzzlePanel) drawPuzzleTypeField(screen *ebiten.Image, px, py int) {
	fieldX, fieldY, fieldW, fieldH := p.drawInputFieldBackground(screen, px, py, 60, 0)

	// Draw type options.
	optionW := (fieldW - 20) / 3
	for i := 0; i < 3; i++ {
		optX := fieldX + 10 + i*optionW
		optY := fieldY + 10
		optH := fieldH - 20

		optBg := p.theme.ButtonBackground
		if PuzzleType(i+1) == p.puzzleType {
			optBg = p.theme.AccentPrimary
		}

		vector.DrawFilledRect(screen, float32(optX), float32(optY),
			float32(optionW-10), float32(optH), optBg, true)
	}
}

// drawDifficultyField draws the difficulty slider.
func (p *PuzzlePanel) drawDifficultyField(screen *ebiten.Image, px, py int) {
	fieldX, fieldY, fieldW, fieldH := p.drawInputFieldBackground(screen, px, py, 120, 1)

	// Draw slider track.
	trackY := fieldY + fieldH/2 - 3
	trackX := fieldX + 80
	trackW := fieldW - 100
	vector.DrawFilledRect(screen, float32(trackX), float32(trackY),
		float32(trackW), 6, p.theme.ButtonBackground, true)

	// Draw slider knob.
	progress := float64(p.difficulty-MinPuzzleDifficulty) / float64(MaxPuzzleDifficulty-MinPuzzleDifficulty)
	knobX := trackX + int(float64(trackW)*progress) - 6
	knobY := fieldY + fieldH/2 - 8
	vector.DrawFilledCircle(screen, float32(knobX+6), float32(knobY+8), 8, p.theme.AccentPrimary, true)
}

// drawDurationField draws the duration selector.
func (p *PuzzlePanel) drawDurationField(screen *ebiten.Image, px, py int) {
	fieldX, fieldY, fieldW, fieldH := p.drawInputFieldBackground(screen, px, py, 180, 2)

	// Draw duration options.
	optionW := (fieldW - 20) / 3
	for i := 0; i < 3; i++ {
		optX := fieldX + 10 + i*optionW
		optY := fieldY + 10
		optH := fieldH - 20

		optBg := p.theme.ButtonBackground
		if i == p.durationIdx {
			optBg = p.theme.AccentPrimary
		}

		vector.DrawFilledRect(screen, float32(optX), float32(optY),
			float32(optionW-10), float32(optH), optBg, true)
	}
}

// drawSeedField draws the optional seed input field.
func (p *PuzzlePanel) drawSeedField(screen *ebiten.Image, px, py int) {
	fieldX, fieldY, _, _ := p.drawInputFieldBackground(screen, px, py, 240, 3)

	// Draw cursor if selected and blinking.
	if p.selectedField == 3 && int(p.animTime*2)%2 == 0 {
		cursorX := fieldX + 8 + p.seedCursorPos*8
		cursorY := fieldY + 15
		vector.DrawFilledRect(screen, float32(cursorX), float32(cursorY),
			2, 20, p.theme.TextPrimary, true)
	}
}

// drawButtons draws the submit and cancel buttons.
func (p *PuzzlePanel) drawButtons(screen *ebiten.Image, px, py int) {
	DrawCancelSubmitButtons(screen, px, py, p.width, p.height, p.theme, 120, "Create", true)
}

// drawError draws the error message.
func (p *PuzzlePanel) drawError(screen *ebiten.Image, px, py int) {
	errorY := py + p.height - p.theme.Padding - p.theme.ButtonHeight - 30

	errorBg := color.RGBA{80, 30, 30, 200}
	vector.DrawFilledRect(screen, float32(px+p.theme.Padding), float32(errorY),
		float32(p.width-p.theme.Padding*2), 22, errorBg, true)
}

// Getters for testing.

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
	p.seedCursorPos = len([]rune(seed))
}
