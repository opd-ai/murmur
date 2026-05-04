// Package ui - Oracle Pool panel implementation.
// Per ROADMAP.md line 462: "UI: Oracle Pool panel — create pool, submit prediction, view outcomes".
//

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// defaultFont holds the default font face for text rendering.
// It must be initialized before drawing text. If nil, text drawing is skipped.
// TODO: Initialize from embedded font via text/v2.NewGoTextFaceSource.
var defaultFont text.Face

// OraclePoolState represents the pool state for display.
type OraclePoolState uint8

const (
	OraclePoolStatePending   OraclePoolState = iota // Accepting predictions.
	OraclePoolStateRevealing                        // Reveal phase.
	OraclePoolStateResolved                         // Outcome determined.
	OraclePoolStateExpired                          // Expired without resolution.
)

// OraclePoolInfo contains pool information for display.
type OraclePoolInfo struct {
	PoolID           [32]byte
	Question         string
	ResolutionMethod string
	Deadline         time.Time
	ResolutionTime   time.Time
	State            OraclePoolState
	PredictionCount  int
	MyPrediction     string // User's submitted prediction (if any).
	MyCommitted      bool   // User has committed a prediction.
	MyRevealed       bool   // User has revealed their prediction.
	Outcome          string // Resolved outcome (if resolved).
}

// OraclePoolPanel provides UI for Oracle Pool interaction.
type OraclePoolPanel struct {
	mu sync.RWMutex

	visible        bool
	pool           *OraclePoolInfo
	mode           OraclePoolPanelMode
	predictionText string
	errorMessage   string
	theme          Theme

	onCreate  func(question, resolutionMethod string, deadline, resolution time.Time)
	onPredict func(poolID [32]byte, prediction string)
	onReveal  func(poolID [32]byte)
}

// OraclePoolPanelMode represents the panel mode.
type OraclePoolPanelMode uint8

const (
	OraclePoolModeView    OraclePoolPanelMode = iota // Viewing a pool.
	OraclePoolModeCreate                             // Creating a new pool.
	OraclePoolModePredict                            // Submitting a prediction.
)

// NewOraclePoolPanel creates a new Oracle Pool panel.
func NewOraclePoolPanel(theme Theme) *OraclePoolPanel {
	return &OraclePoolPanel{
		theme: theme,
		mode:  OraclePoolModeView,
	}
}

// Visible returns true if the panel is shown.
func (p *OraclePoolPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *OraclePoolPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *OraclePoolPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *OraclePoolPanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetPool sets the pool to display.
func (p *OraclePoolPanel) SetPool(pool *OraclePoolInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = pool
	p.mode = OraclePoolModeView
	p.errorMessage = ""
}

// SetMode sets the panel mode.
func (p *OraclePoolPanel) SetMode(mode OraclePoolPanelMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
	p.errorMessage = ""
}

// SetOnCreate sets the callback for pool creation.
func (p *OraclePoolPanel) SetOnCreate(callback func(question, resolutionMethod string, deadline, resolution time.Time)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onCreate = callback
}

// SetOnPredict sets the callback for prediction submission.
func (p *OraclePoolPanel) SetOnPredict(callback func(poolID [32]byte, prediction string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onPredict = callback
}

// SetOnReveal sets the callback for prediction reveal.
func (p *OraclePoolPanel) SetOnReveal(callback func(poolID [32]byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onReveal = callback
}

// Update handles input and updates panel state.
func (p *OraclePoolPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	p.handlePredictionTextInput()
	p.handleKeyboardActions()

	return true
}

// handlePredictionTextInput processes text input for predictions.
func (p *OraclePoolPanel) handlePredictionTextInput() {
	if p.mode != OraclePoolModePredict {
		return
	}
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(p.predictionText) < 64 {
			p.predictionText += string(r)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(p.predictionText) > 0 {
		p.predictionText = p.predictionText[:len(p.predictionText)-1]
	}
}

// handleKeyboardActions processes keyboard shortcuts and actions.
func (p *OraclePoolPanel) handleKeyboardActions() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.handleEnterKey()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		p.handleRevealKey()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		p.handlePredictModeKey()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.handleEscapeKey()
	}
}

// handleEnterKey submits a prediction when in predict mode.
func (p *OraclePoolPanel) handleEnterKey() {
	if p.mode == OraclePoolModePredict && p.pool != nil {
		p.submitPrediction()
	}
}

// handleRevealKey reveals a committed prediction during reveal phase.
func (p *OraclePoolPanel) handleRevealKey() {
	if p.mode != OraclePoolModeView || p.pool == nil || p.pool.State != OraclePoolStateRevealing {
		return
	}
	if p.pool.MyCommitted && !p.pool.MyRevealed {
		p.revealPrediction()
	}
}

// handlePredictModeKey switches to prediction mode when eligible.
func (p *OraclePoolPanel) handlePredictModeKey() {
	if p.mode != OraclePoolModeView || p.pool == nil || p.pool.State != OraclePoolStatePending {
		return
	}
	if !p.pool.MyCommitted {
		p.mode = OraclePoolModePredict
		p.predictionText = ""
	}
}

// handleEscapeKey exits predict mode or closes the panel.
func (p *OraclePoolPanel) handleEscapeKey() {
	if p.mode != OraclePoolModeView {
		p.mode = OraclePoolModeView
	} else {
		p.visible = false
	}
}

// submitPrediction attempts to submit the prediction.
func (p *OraclePoolPanel) submitPrediction() {
	if p.predictionText == "" {
		p.errorMessage = "Prediction cannot be empty"
		return
	}
	if p.pool == nil {
		p.errorMessage = "No pool selected"
		return
	}
	if p.onPredict != nil {
		p.onPredict(p.pool.PoolID, p.predictionText)
	}
	p.mode = OraclePoolModeView
	p.predictionText = ""
}

// revealPrediction triggers prediction reveal.
func (p *OraclePoolPanel) revealPrediction() {
	if p.pool == nil {
		return
	}
	if p.onReveal != nil {
		p.onReveal(p.pool.PoolID)
	}
}

// Draw renders the panel.
func (p *OraclePoolPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	const (
		panelWidth  = 450
		panelHeight = 350
		padding     = 16
	)

	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	panelX := float32((screenW - panelWidth) / 2)
	panelY := float32((screenH - panelHeight) / 2)

	// Background.
	vector.DrawFilledRect(screen, panelX, panelY, panelWidth, panelHeight, p.theme.PanelBackground, true)
	vector.StrokeRect(screen, panelX, panelY, panelWidth, panelHeight, 2, p.theme.PanelBorder, true)

	// Title.
	title := "Oracle Pool"
	if p.mode == OraclePoolModeCreate {
		title = "Create Oracle Pool"
	} else if p.mode == OraclePoolModePredict {
		title = "Submit Prediction"
	}
	if defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding))
		op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, title, defaultFont, op)
	}

	if p.pool != nil {
		p.drawPoolDetails(screen, panelX, panelY, padding)
	}

	// Error message.
	if p.errorMessage != "" && defaultFont != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(panelX+padding), float64(panelY+panelHeight-50))
		op.ColorScale.ScaleWithColor(p.theme.TextError)
		text.Draw(screen, p.errorMessage, defaultFont, op)
	}

	// Instructions.
	p.drawInstructions(screen, panelX, panelY, panelHeight, padding)
}

// drawPoolDetails draws the pool information.
func (p *OraclePoolPanel) drawPoolDetails(screen *ebiten.Image, panelX, panelY, padding float32) {
	if defaultFont == nil {
		return
	}

	// Question.
	questionText := p.pool.Question
	if len(questionText) > 50 {
		questionText = questionText[:50] + "..."
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+30))
	op.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
	text.Draw(screen, "Q: "+questionText, defaultFont, op)

	// Status.
	statusText := p.formatStatus()
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+55))
	op2.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, "Status: "+statusText, defaultFont, op2)

	// Deadline.
	deadlineText := p.formatDeadline()
	op3 := &text.DrawOptions{}
	op3.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+80))
	op3.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, "Deadline: "+deadlineText, defaultFont, op3)

	// Predictions.
	predText := fmt.Sprintf("Predictions: %d", p.pool.PredictionCount)
	op4 := &text.DrawOptions{}
	op4.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+105))
	op4.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, predText, defaultFont, op4)

	// My prediction.
	if p.pool.MyCommitted {
		myPredText := "Your prediction: Committed"
		if p.pool.MyRevealed {
			myPredText = "Your prediction: " + p.pool.MyPrediction
		}
		op5 := &text.DrawOptions{}
		op5.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+130))
		op5.ColorScale.ScaleWithColor(p.theme.Success)
		text.Draw(screen, myPredText, defaultFont, op5)
	}

	// Outcome (if resolved).
	if p.pool.State == OraclePoolStateResolved && p.pool.Outcome != "" {
		op6 := &text.DrawOptions{}
		op6.GeoM.Translate(float64(panelX+padding), float64(panelY+padding+160))
		op6.ColorScale.ScaleWithColor(p.theme.AccentPrimary)
		text.Draw(screen, "Outcome: "+p.pool.Outcome, defaultFont, op6)
	}

	// Prediction input (if in predict mode).
	if p.mode == OraclePoolModePredict {
		// Input box.
		inputY := panelY + padding + 190
		vector.DrawFilledRect(screen, panelX+padding, inputY, 400, 30, p.theme.InputBackground, true)
		vector.StrokeRect(screen, panelX+padding, inputY, 400, 30, 1, p.theme.PanelBorder, true)

		// Input text.
		displayText := p.predictionText
		if len(displayText) == 0 {
			displayText = "Enter prediction..."
		}
		op7 := &text.DrawOptions{}
		op7.GeoM.Translate(float64(panelX+padding+8), float64(inputY+8))
		op7.ColorScale.ScaleWithColor(p.theme.TextPrimary)
		text.Draw(screen, displayText, defaultFont, op7)
	}
}

// formatStatus returns a human-readable status.
func (p *OraclePoolPanel) formatStatus() string {
	switch p.pool.State {
	case OraclePoolStatePending:
		return "Accepting Predictions"
	case OraclePoolStateRevealing:
		return "Reveal Phase"
	case OraclePoolStateResolved:
		return "Resolved"
	case OraclePoolStateExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// formatDeadline returns a formatted deadline string.
func (p *OraclePoolPanel) formatDeadline() string {
	remaining := time.Until(p.pool.Deadline)
	if remaining <= 0 {
		return "Passed"
	}

	hours := int(remaining.Hours())
	if hours >= 24 {
		days := hours / 24
		return fmt.Sprintf("%dd %dh", days, hours%24)
	}
	return fmt.Sprintf("%dh %dm", hours, int(remaining.Minutes())%60)
}

// drawInstructions draws the instruction text.
func (p *OraclePoolPanel) drawInstructions(screen *ebiten.Image, panelX, panelY, panelHeight, padding float32) {
	if defaultFont == nil {
		return
	}

	var instructions string
	switch p.mode {
	case OraclePoolModeView:
		if p.pool != nil {
			switch p.pool.State {
			case OraclePoolStatePending:
				if !p.pool.MyCommitted {
					instructions = "P:Predict  Esc:Close"
				} else {
					instructions = "Prediction committed  Esc:Close"
				}
			case OraclePoolStateRevealing:
				if p.pool.MyCommitted && !p.pool.MyRevealed {
					instructions = "R:Reveal  Esc:Close"
				} else {
					instructions = "Esc:Close"
				}
			default:
				instructions = "Esc:Close"
			}
		} else {
			instructions = "Esc:Close"
		}
	case OraclePoolModePredict:
		instructions = "Enter:Submit  Esc:Cancel"
	case OraclePoolModeCreate:
		instructions = "Enter:Create  Esc:Cancel"
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panelX+padding), float64(panelY+panelHeight-30))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, instructions, defaultFont, op)
}

// GetPool returns the current pool info.
func (p *OraclePoolPanel) GetPool() *OraclePoolInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pool
}

// GetMode returns the current panel mode.
func (p *OraclePoolPanel) GetMode() OraclePoolPanelMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// GetPredictionText returns the current prediction input.
func (p *OraclePoolPanel) GetPredictionText() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.predictionText
}

// SetPredictionText sets the prediction input (for testing).
func (p *OraclePoolPanel) SetPredictionText(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.predictionText = text
}

// GetErrorMessage returns the current error message.
func (p *OraclePoolPanel) GetErrorMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorMessage
}

// SetError sets the error message.
func (p *OraclePoolPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMessage = msg
}

// SubmitPrediction manually triggers prediction submission (for testing).
func (p *OraclePoolPanel) SubmitPrediction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.submitPrediction()
}

// RevealPrediction manually triggers prediction reveal (for testing).
func (p *OraclePoolPanel) RevealPrediction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.revealPrediction()
}
