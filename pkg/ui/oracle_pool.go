// Package ui - Oracle Pool panel implementation.
// Per ROADMAP.md line 462: "UI: Oracle Pool panel — create pool, submit prediction, view outcomes".
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
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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

	panelX, panelY := p.calculatePanelPosition(screen)
	p.drawPanelBackground(screen, panelX, panelY)
	p.drawPanelTitle(screen, panelX, panelY)

	if p.pool != nil {
		p.drawPoolDetails(screen, panelX, panelY, panelPadding)
	}

	p.drawErrorIfPresent(screen, panelX, panelY)
	p.drawInstructions(screen, panelX, panelY, panelHeight, panelPadding)
}

const (
	panelWidth   = 450
	panelHeight  = 350
	panelPadding = 16
)

// calculatePanelPosition computes centered panel coordinates.
func (p *OraclePoolPanel) calculatePanelPosition(screen *ebiten.Image) (float32, float32) {
	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	panelX := float32((screenW - panelWidth) / 2)
	panelY := float32((screenH - panelHeight) / 2)
	return panelX, panelY
}

// drawPanelBackground renders the panel background and border.
func (p *OraclePoolPanel) drawPanelBackground(screen *ebiten.Image, panelX, panelY float32) {
	vector.DrawFilledRect(screen, panelX, panelY, panelWidth, panelHeight, p.theme.PanelBackground, true)
	vector.StrokeRect(screen, panelX, panelY, panelWidth, panelHeight, 2, p.theme.PanelBorder, true)
}

// drawPanelTitle renders the mode-specific panel title.
func (p *OraclePoolPanel) drawPanelTitle(screen *ebiten.Image, panelX, panelY float32) {
	if defaultFont == nil {
		return
	}

	title := selectOraclePoolTitle(p.mode)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panelX+panelPadding), float64(panelY+panelPadding))
	op.ColorScale.ScaleWithColor(p.theme.TextPrimary)
	text.Draw(screen, title, defaultFont, op)
}

// selectOraclePoolTitle returns the appropriate title based on mode.
func selectOraclePoolTitle(mode OraclePoolPanelMode) string {
	switch mode {
	case OraclePoolModeCreate:
		return "Create Oracle Pool"
	case OraclePoolModePredict:
		return "Submit Prediction"
	default:
		return "Oracle Pool"
	}
}

// drawErrorIfPresent renders the error message if one exists.
func (p *OraclePoolPanel) drawErrorIfPresent(screen *ebiten.Image, panelX, panelY float32) {
	if p.errorMessage == "" || defaultFont == nil {
		return
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panelX+panelPadding), float64(panelY+panelHeight-50))
	op.ColorScale.ScaleWithColor(p.theme.TextError)
	text.Draw(screen, p.errorMessage, defaultFont, op)
}

// drawPoolDetails draws the pool information.
func (p *OraclePoolPanel) drawPoolDetails(screen *ebiten.Image, panelX, panelY, padding float32) {
	if defaultFont == nil {
		return
	}

	p.drawQuestionText(screen, panelX, panelY, padding)
	p.drawStatusText(screen, panelX, panelY, padding)
	p.drawDeadlineText(screen, panelX, panelY, padding)
	p.drawPredictionCount(screen, panelX, panelY, padding)
	p.drawMyPredictionIfCommitted(screen, panelX, panelY, padding)
	p.drawOutcomeIfResolved(screen, panelX, panelY, padding)
	p.drawPredictionInputIfPredictMode(screen, panelX, panelY, padding)
}

// drawQuestionText draws the pool question.
func (p *OraclePoolPanel) drawQuestionText(screen *ebiten.Image, panelX, panelY, padding float32) {
	questionText := p.pool.Question
	if len(questionText) > 50 {
		questionText = questionText[:50] + "..."
	}
	p.drawTextAtOffset(screen, "Q: "+questionText, panelX+padding, panelY+padding+30, p.theme.AccentPrimary)
}

// drawStatusText draws the pool status.
func (p *OraclePoolPanel) drawStatusText(screen *ebiten.Image, panelX, panelY, padding float32) {
	statusText := p.formatStatus()
	p.drawTextAtOffset(screen, "Status: "+statusText, panelX+padding, panelY+padding+55, p.theme.TextSecondary)
}

// drawDeadlineText draws the pool deadline.
func (p *OraclePoolPanel) drawDeadlineText(screen *ebiten.Image, panelX, panelY, padding float32) {
	deadlineText := p.formatDeadline()
	p.drawTextAtOffset(screen, "Deadline: "+deadlineText, panelX+padding, panelY+padding+80, p.theme.TextSecondary)
}

// drawPredictionCount draws the prediction count.
func (p *OraclePoolPanel) drawPredictionCount(screen *ebiten.Image, panelX, panelY, padding float32) {
	predText := fmt.Sprintf("Predictions: %d", p.pool.PredictionCount)
	p.drawTextAtOffset(screen, predText, panelX+padding, panelY+padding+105, p.theme.TextSecondary)
}

// drawMyPredictionIfCommitted draws the user's prediction if committed.
func (p *OraclePoolPanel) drawMyPredictionIfCommitted(screen *ebiten.Image, panelX, panelY, padding float32) {
	if !p.pool.MyCommitted {
		return
	}
	myPredText := "Your prediction: Committed"
	if p.pool.MyRevealed {
		myPredText = "Your prediction: " + p.pool.MyPrediction
	}
	p.drawTextAtOffset(screen, myPredText, panelX+padding, panelY+padding+130, p.theme.Success)
}

// drawOutcomeIfResolved draws the outcome if the pool is resolved.
func (p *OraclePoolPanel) drawOutcomeIfResolved(screen *ebiten.Image, panelX, panelY, padding float32) {
	if p.pool.State == OraclePoolStateResolved && p.pool.Outcome != "" {
		p.drawTextAtOffset(screen, "Outcome: "+p.pool.Outcome, panelX+padding, panelY+padding+160, p.theme.AccentPrimary)
	}
}

// drawPredictionInputIfPredictMode draws the prediction input box if in predict mode.
func (p *OraclePoolPanel) drawPredictionInputIfPredictMode(screen *ebiten.Image, panelX, panelY, padding float32) {
	if p.mode != OraclePoolModePredict {
		return
	}
	inputY := panelY + padding + 190
	vector.DrawFilledRect(screen, panelX+padding, inputY, 400, 30, p.theme.InputBackground, true)
	vector.StrokeRect(screen, panelX+padding, inputY, 400, 30, 1, p.theme.PanelBorder, true)

	displayText := p.predictionText
	if len(displayText) == 0 {
		displayText = "Enter prediction..."
	}
	p.drawTextAtOffset(screen, displayText, panelX+padding+8, inputY+8, p.theme.TextPrimary)
}

// drawTextAtOffset is a helper to draw text at a specific position with color.
func (p *OraclePoolPanel) drawTextAtOffset(screen *ebiten.Image, txt string, x, y float32, col color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(col)
	text.Draw(screen, txt, defaultFont, op)
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

	instructions := p.getInstructionsText()

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(panelX+padding), float64(panelY+panelHeight-30))
	op.ColorScale.ScaleWithColor(p.theme.TextSecondary)
	text.Draw(screen, instructions, defaultFont, op)
}

// getInstructionsText returns the appropriate instructions text based on current mode and pool state.
func (p *OraclePoolPanel) getInstructionsText() string {
	switch p.mode {
	case OraclePoolModeView:
		return p.getViewModeInstructions()
	case OraclePoolModePredict:
		return "Enter:Submit  Esc:Cancel"
	case OraclePoolModeCreate:
		return "Enter:Create  Esc:Cancel"
	default:
		return "Esc:Close"
	}
}

// getViewModeInstructions returns instructions text for view mode based on pool state.
func (p *OraclePoolPanel) getViewModeInstructions() string {
	if p.pool == nil {
		return "Esc:Close"
	}

	switch p.pool.State {
	case OraclePoolStatePending:
		return p.getPendingStateInstructions()
	case OraclePoolStateRevealing:
		return p.getRevealingStateInstructions()
	default:
		return "Esc:Close"
	}
}

// getPendingStateInstructions returns instructions for pending pool state.
func (p *OraclePoolPanel) getPendingStateInstructions() string {
	if !p.pool.MyCommitted {
		return "P:Predict  Esc:Close"
	}
	return "Prediction committed  Esc:Close"
}

// getRevealingStateInstructions returns instructions for revealing pool state.
func (p *OraclePoolPanel) getRevealingStateInstructions() string {
	if p.pool.MyCommitted && !p.pool.MyRevealed {
		return "R:Reveal  Esc:Close"
	}
	return "Esc:Close"
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
