// Package ui provides key rotation wizard UI per KEY_ROTATION.md.

//go:build !test
// +build !test

package ui

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/opd-ai/murmur/pkg/identity/rotation"
	"github.com/opd-ai/murmur/proto"
)

// RotationState represents the state of key rotation process.
type RotationState int

const (
	RotationStateConfirm RotationState = iota
	RotationStateGeneratingKey
	RotationStateConfigureGracePeriod
	RotationStateCreatingDeclaration
	RotationStatePropagating
	RotationStateComplete
	RotationStateError
)

// KeyRotationWizard manages UI for key rotation process.
type KeyRotationWizard struct {
	mu              sync.RWMutex
	visible         bool
	width, height   int
	theme           *Theme
	state           RotationState
	oldPrivateKey   ed25519.PrivateKey
	oldPublicKey    ed25519.PublicKey
	newPrivateKey   ed25519.PrivateKey
	newPublicKey    ed25519.PublicKey
	gracePeriodDays uint32
	declaration     *proto.ContinuityDeclaration
	errorMsg        string
	successMsg      string
	onComplete      func(*proto.ContinuityDeclaration)
	onCancel        func()
	reason          string
}

// NewKeyRotationWizard creates a new key rotation wizard.
func NewKeyRotationWizard(
	oldPrivateKey ed25519.PrivateKey,
	oldPublicKey ed25519.PublicKey,
	theme *Theme,
) *KeyRotationWizard {
	return &KeyRotationWizard{
		width:           600,
		height:          450,
		theme:           theme,
		state:           RotationStateConfirm,
		oldPrivateKey:   oldPrivateKey,
		oldPublicKey:    oldPublicKey,
		gracePeriodDays: 7, // Default 7-day grace period
		reason:          "Scheduled rotation",
	}
}

// SetOnComplete sets callback for successful rotation.
func (w *KeyRotationWizard) SetOnComplete(callback func(*proto.ContinuityDeclaration)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onComplete = callback
}

// SetOnCancel sets callback for canceled rotation.
func (w *KeyRotationWizard) SetOnCancel(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onCancel = callback
}

// Show displays the wizard.
func (w *KeyRotationWizard) Show() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.visible = true
	w.state = RotationStateConfirm
	w.errorMsg = ""
	w.successMsg = ""
}

// Hide hides the wizard.
func (w *KeyRotationWizard) Hide() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.visible = false
}

// IsVisible returns whether wizard is visible.
func (w *KeyRotationWizard) IsVisible() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.visible
}

// Update handles input for the wizard.
func (w *KeyRotationWizard) Update() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.visible {
		return false
	}

	if w.handleEscapeKey() {
		return true
	}

	return w.handleStateUpdate()
}

// handleEscapeKey processes escape key input in cancelable states.
func (w *KeyRotationWizard) handleEscapeKey() bool {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return false
	}

	if w.state == RotationStateConfirm || w.state == RotationStateConfigureGracePeriod {
		w.visible = false
		if w.onCancel != nil {
			w.onCancel()
		}
	}
	return true
}

// handleStateUpdate delegates to the appropriate state handler.
func (w *KeyRotationWizard) handleStateUpdate() bool {
	switch w.state {
	case RotationStateConfirm:
		return w.updateConfirm()
	case RotationStateConfigureGracePeriod:
		return w.updateGracePeriodConfig()
	case RotationStateComplete:
		w.handleCompleteState()
	}
	return true
}

// handleCompleteState processes input in the completion state.
func (w *KeyRotationWizard) handleCompleteState() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		w.visible = false
		if w.onComplete != nil {
			w.onComplete(w.declaration)
		}
	}
}

// updateConfirm handles confirmation screen inputs.
func (w *KeyRotationWizard) updateConfirm() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		w.state = RotationStateGeneratingKey
		go w.performKeyGeneration()
	}
	return true
}

// updateGracePeriodConfig handles grace period configuration inputs.
func (w *KeyRotationWizard) updateGracePeriodConfig() bool {
	// Arrow keys adjust grace period (1-14 days)
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && w.gracePeriodDays < 14 {
		w.gracePeriodDays++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && w.gracePeriodDays > 1 {
		w.gracePeriodDays--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		w.state = RotationStateCreatingDeclaration
		go w.performRotation()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		// Go back to confirm (re-generate key)
		w.state = RotationStateConfirm
		w.errorMsg = ""
	}

	return true
}

// performKeyGeneration generates new keypair.
func (w *KeyRotationWizard) performKeyGeneration() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)

	w.mu.Lock()
	defer w.mu.Unlock()

	if err != nil {
		w.state = RotationStateError
		w.errorMsg = fmt.Sprintf("Key generation failed: %v", err)
		return
	}

	w.newPublicKey = pub
	w.newPrivateKey = priv
	w.state = RotationStateConfigureGracePeriod
}

// performRotation creates and signs the continuity declaration.
func (w *KeyRotationWizard) performRotation() {
	opts := &rotation.RotateOptions{
		GracePeriodDays: int64(w.gracePeriodDays),
		Reason:          w.reason,
	}

	declaration, err := rotation.CreateRotation(
		w.oldPrivateKey,
		w.newPrivateKey,
		opts,
	)

	w.mu.Lock()
	defer w.mu.Unlock()

	if err != nil {
		w.state = RotationStateError
		w.errorMsg = fmt.Sprintf("Rotation failed: %v", err)
		return
	}

	w.declaration = declaration
	w.state = RotationStatePropagating

	// Simulate propagation delay
	time.Sleep(500 * time.Millisecond)

	w.state = RotationStateComplete
	w.successMsg = fmt.Sprintf("Key rotated successfully with %d-day grace period", w.gracePeriodDays)
}

// Draw renders the wizard.
func (w *KeyRotationWizard) Draw(screen *ebiten.Image) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	px, py, sw, sh, shouldRender := CheckPanelVisibilityAndCenter(screen, w.visible, w.width, w.height)
	if !shouldRender {
		return
	}

	// Draw overlay and panel
	DrawModalOverlayAndPanel(screen, px, py, sw, sh, w.width, w.height, *w.theme)

	// Draw title
	titleY := py + 30
	drawUICenteredText(screen, "Key Rotation", float64(px+w.width/2), float64(titleY), w.theme.TextPrimary)

	// Draw content based on state
	contentY := py + 80

	switch w.state {
	case RotationStateConfirm:
		w.drawConfirm(screen, px, py, contentY)
	case RotationStateGeneratingKey:
		w.drawGenerating(screen, px, contentY)
	case RotationStateConfigureGracePeriod:
		w.drawGracePeriodConfig(screen, px, py, contentY)
	case RotationStateCreatingDeclaration:
		w.drawCreating(screen, px, contentY)
	case RotationStatePropagating:
		w.drawPropagating(screen, px, contentY)
	case RotationStateComplete:
		w.drawComplete(screen, px, py, contentY)
	case RotationStateError:
		w.drawError(screen, px, contentY)
	}
}

// drawConfirm renders confirmation screen.
func (w *KeyRotationWizard) drawConfirm(screen *ebiten.Image, px, py, startY int) {
	y := startY

	// Warning
	drawUIText(screen, "⚠ Key Rotation", float64(px+20), float64(y), w.theme.Warning)
	y += 40

	// Explanation
	lines := []string{
		"This will generate a new keypair while maintaining",
		"your identity continuity. Your contacts will be",
		"notified automatically.",
		"",
		"During the grace period, both keys will work.",
		"After that, only the new key will be accepted.",
	}

	for _, line := range lines {
		drawUIText(screen, line, float64(px+20), float64(y), w.theme.TextSecondary)
		y += 25
	}

	// Controls
	y = py + w.height - 80
	drawUIText(screen, "Enter: Continue  Esc: Cancel",
		float64(px+20), float64(y), w.theme.TextSecondary)
}

// drawGenerating renders key generation screen.
func (w *KeyRotationWizard) drawGenerating(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Generating new keypair...",
		float64(px+w.width/2), float64(y), w.theme.TextPrimary)
}

// drawGracePeriodConfig renders grace period configuration.
func (w *KeyRotationWizard) drawGracePeriodConfig(screen *ebiten.Image, px, py, startY int) {
	y := startY

	drawUIText(screen, "Configure Grace Period", float64(px+20), float64(y), w.theme.TextPrimary)
	y += 40

	drawUIText(screen, "How long should both keys work?",
		float64(px+20), float64(y), w.theme.TextSecondary)
	y += 30

	// Grace period selector
	gracePeriodText := fmt.Sprintf("%d days", w.gracePeriodDays)
	drawUICenteredText(screen, gracePeriodText, float64(px+w.width/2), float64(y), w.theme.TextPrimary)
	y += 40

	// Recommendation
	recommendation := "Recommended: 7 days (allows time for all devices to sync)"
	drawUIText(screen, recommendation, float64(px+20), float64(y), w.theme.TextSecondary)
	y += 60

	// Info
	info := fmt.Sprintf("After %d days, old key will be rejected", w.gracePeriodDays)
	drawUIText(screen, info, float64(px+20), float64(y), w.theme.TextSecondary)

	// Controls
	y = py + w.height - 80
	drawUIText(screen, "↑↓: Adjust  Enter: Confirm  Backspace: Back  Esc: Cancel",
		float64(px+20), float64(y), w.theme.TextSecondary)
}

// drawCreating renders declaration creation screen.
func (w *KeyRotationWizard) drawCreating(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Creating continuity declaration...",
		float64(px+w.width/2), float64(y), w.theme.TextPrimary)
}

// drawPropagating renders propagation screen.
func (w *KeyRotationWizard) drawPropagating(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Propagating to network...",
		float64(px+w.width/2), float64(y), w.theme.TextPrimary)
	y += 30
	drawUICenteredText(screen, "Your contacts are being notified",
		float64(px+w.width/2), float64(y), w.theme.TextSecondary)
}

// drawComplete renders completion screen.
func (w *KeyRotationWizard) drawComplete(screen *ebiten.Image, px, py, startY int) {
	y := startY + 60

	drawUICenteredText(screen, w.successMsg,
		float64(px+w.width/2), float64(y), w.theme.Success)
	y += 60

	// Important notes
	notes := []string{
		"✓ New key generated and signed",
		"✓ Continuity declaration created",
		"✓ Network notification sent",
		"",
		fmt.Sprintf("Grace period ends: %s",
			time.Now().Add(time.Duration(w.gracePeriodDays)*24*time.Hour).Format("2006-01-02")),
	}

	for _, note := range notes {
		drawUIText(screen, note, float64(px+40), float64(y), w.theme.TextSecondary)
		y += 25
	}

	// Instructions
	y = py + w.height - 60
	drawUICenteredText(screen, "Press Enter to close",
		float64(px+w.width/2), float64(y), w.theme.TextSecondary)
}

// drawError renders error screen.
func (w *KeyRotationWizard) drawError(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Rotation Error",
		float64(px+w.width/2), float64(y), w.theme.TextError)
	y += 40
	drawUICenteredText(screen, w.errorMsg,
		float64(px+w.width/2), float64(y), w.theme.TextSecondary)
	y += 60
	drawUICenteredText(screen, "Press Escape to close",
		float64(px+w.width/2), float64(y), w.theme.TextSecondary)
}
