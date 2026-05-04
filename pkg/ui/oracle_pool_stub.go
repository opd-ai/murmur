// Package ui - Oracle Pool panel stub implementation.
// Per ROADMAP.md line 462: "UI: Oracle Pool panel — create pool, submit prediction, view outcomes".
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"
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
	MyPrediction     string
	MyCommitted      bool
	MyRevealed       bool
	Outcome          string
}

// OraclePoolPanel provides UI for Oracle Pool interaction (stub).
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

// NewOraclePoolPanel creates a new Oracle Pool panel (stub).
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

// Update handles input (stub).
func (p *OraclePoolPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
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

// RevealPrediction manually triggers prediction reveal (for testing).
func (p *OraclePoolPanel) RevealPrediction() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.pool == nil {
		return
	}
	if p.onReveal != nil {
		p.onReveal(p.pool.PoolID)
	}
}
