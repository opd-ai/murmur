// Package ui provides social recovery contact enrollment UI per SOCIAL_RECOVERY.md.

//go:build !test
// +build !test

package ui

import (
	"crypto/ed25519"
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/identity/recovery"
)

// EnrollmentState represents the state of recovery enrollment process.
type EnrollmentState int

const (
	EnrollmentStateSelectContacts EnrollmentState = iota
	EnrollmentStateConfigureThreshold
	EnrollmentStateDistributing
	EnrollmentStateComplete
	EnrollmentStateError
)

// RecoveryContact represents a contact available for enrollment.
type RecoveryContact struct {
	PublicKey ed25519.PublicKey
	X25519Key []byte
	Label     string
	Selected  bool
}

// RecoveryEnrollmentPanel manages UI for social recovery contact enrollment.
type RecoveryEnrollmentPanel struct {
	mu                sync.RWMutex
	visible           bool
	width, height     int
	theme             *Theme
	state             EnrollmentState
	contacts          []RecoveryContact
	selectedCount     int
	threshold         uint32 // M in M-of-N
	totalShares       uint32 // N in M-of-N
	scrollOffset      int
	errorMsg          string
	successMsg        string
	onComplete        func(results []recovery.EnrollmentResult)
	onCancel          func()
	masterPrivateKey  ed25519.PrivateKey
	masterPublicKey   ed25519.PublicKey
	x25519PrivateKey  []byte
	enrollmentResults []recovery.EnrollmentResult
}

// NewRecoveryEnrollmentPanel creates a new recovery enrollment panel.
func NewRecoveryEnrollmentPanel(
	contacts []RecoveryContact,
	masterPrivateKey ed25519.PrivateKey,
	masterPublicKey ed25519.PublicKey,
	x25519PrivateKey []byte,
	theme *Theme,
) *RecoveryEnrollmentPanel {
	return &RecoveryEnrollmentPanel{
		width:            600,
		height:           500,
		theme:            theme,
		state:            EnrollmentStateSelectContacts,
		contacts:         contacts,
		threshold:        3, // Default 3-of-5
		totalShares:      5,
		masterPrivateKey: masterPrivateKey,
		masterPublicKey:  masterPublicKey,
		x25519PrivateKey: x25519PrivateKey,
	}
}

// SetOnComplete sets callback for successful enrollment.
func (p *RecoveryEnrollmentPanel) SetOnComplete(callback func(results []recovery.EnrollmentResult)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onComplete = callback
}

// SetOnCancel sets callback for canceled enrollment.
func (p *RecoveryEnrollmentPanel) SetOnCancel(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onCancel = callback
}

// Show displays the panel.
func (p *RecoveryEnrollmentPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.state = EnrollmentStateSelectContacts
	p.selectedCount = 0
	p.errorMsg = ""
	p.successMsg = ""
	for i := range p.contacts {
		p.contacts[i].Selected = false
	}
}

// Hide hides the panel.
func (p *RecoveryEnrollmentPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// IsVisible returns whether panel is visible.
func (p *RecoveryEnrollmentPanel) IsVisible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Update handles input for the panel.
func (p *RecoveryEnrollmentPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	if p.handleEscapeKey() {
		return true
	}

	return p.handleStateUpdate()
}

// handleEscapeKey processes escape key input in cancelable states.
func (p *RecoveryEnrollmentPanel) handleEscapeKey() bool {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return false
	}

	if p.state == EnrollmentStateSelectContacts || p.state == EnrollmentStateConfigureThreshold {
		p.visible = false
		if p.onCancel != nil {
			p.onCancel()
		}
	}
	return true
}

// handleStateUpdate delegates to the appropriate state handler.
func (p *RecoveryEnrollmentPanel) handleStateUpdate() bool {
	switch p.state {
	case EnrollmentStateSelectContacts:
		return p.updateContactSelection()
	case EnrollmentStateConfigureThreshold:
		return p.updateThresholdConfiguration()
	case EnrollmentStateComplete:
		p.handleCompleteState()
	}
	return true
}

// handleCompleteState processes input in the completion state.
func (p *RecoveryEnrollmentPanel) handleCompleteState() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.visible = false
		if p.onComplete != nil {
			p.onComplete(p.enrollmentResults)
		}
	}
}

// updateContactSelection handles contact selection inputs.
func (p *RecoveryEnrollmentPanel) updateContactSelection() bool {
	p.handleContactToggle()
	p.handleContactNavigation()
	p.handleContactConfirmation()
	return true
}

// handleContactToggle toggles the selected state of the current contact.
func (p *RecoveryEnrollmentPanel) handleContactToggle() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(p.contacts) > 0 && p.scrollOffset < len(p.contacts) {
			p.contacts[p.scrollOffset].Selected = !p.contacts[p.scrollOffset].Selected
			p.selectedCount = p.countSelected()
		}
	}
}

// handleContactNavigation handles up/down arrow navigation through contacts.
func (p *RecoveryEnrollmentPanel) handleContactNavigation() {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && p.scrollOffset > 0 {
		p.scrollOffset--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && p.scrollOffset < len(p.contacts)-1 {
		p.scrollOffset++
	}
}

// handleContactConfirmation validates and proceeds to threshold configuration.
func (p *RecoveryEnrollmentPanel) handleContactConfirmation() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if p.selectedCount >= int(recovery.MinThreshold) && p.selectedCount <= int(recovery.MaxTotalShares) {
			p.totalShares = uint32(p.selectedCount)
			p.threshold = minUint32(3, p.totalShares)
			p.state = EnrollmentStateConfigureThreshold
		} else {
			p.errorMsg = fmt.Sprintf("Please select %d-%d contacts", recovery.MinThreshold, recovery.MaxTotalShares)
		}
	}
}

// updateThresholdConfiguration handles threshold adjustment inputs.
func (p *RecoveryEnrollmentPanel) updateThresholdConfiguration() bool {
	// Arrow keys adjust threshold
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && p.threshold < p.totalShares {
		p.threshold++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && p.threshold > recovery.MinThreshold {
		p.threshold--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		// Start enrollment
		p.state = EnrollmentStateDistributing
		go p.performEnrollment()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		// Go back to contact selection
		p.state = EnrollmentStateSelectContacts
		p.errorMsg = ""
	}

	return true
}

// performEnrollment executes the enrollment process in background.
func (p *RecoveryEnrollmentPanel) performEnrollment() {
	selectedContacts := p.extractSelectedContacts()

	results, err := recovery.EnrollRecoveryContacts(
		p.masterPrivateKey,
		p.masterPublicKey,
		p.x25519PrivateKey,
		selectedContacts,
		p.threshold,
		p.totalShares,
		"Primary Recovery",
	)

	p.mu.Lock()
	defer p.mu.Unlock()

	if err != nil {
		p.setEnrollmentError(err)
		return
	}

	p.finalizeEnrollmentResults(results)
}

// extractSelectedContacts returns a slice of selected contacts.
func (p *RecoveryEnrollmentPanel) extractSelectedContacts() []recovery.Contact {
	var selectedContacts []recovery.Contact
	for _, rc := range p.contacts {
		if rc.Selected {
			selectedContacts = append(selectedContacts, recovery.Contact{
				PublicKey: rc.PublicKey,
				X25519Key: rc.X25519Key,
				Label:     rc.Label,
			})
		}
	}
	return selectedContacts
}

// setEnrollmentError sets the error state for enrollment failure.
func (p *RecoveryEnrollmentPanel) setEnrollmentError(err error) {
	p.state = EnrollmentStateError
	p.errorMsg = fmt.Sprintf("Enrollment failed: %v", err)
}

// finalizeEnrollmentResults processes enrollment results and sets final state.
func (p *RecoveryEnrollmentPanel) finalizeEnrollmentResults(results []recovery.EnrollmentResult) {
	p.enrollmentResults = results
	failedCount := p.countFailedEnrollments(results)

	if failedCount == 0 {
		p.state = EnrollmentStateComplete
		p.successMsg = fmt.Sprintf("Successfully enrolled %d contacts with %d-of-%d threshold",
			len(results), p.threshold, p.totalShares)
	} else {
		p.state = EnrollmentStateError
		p.errorMsg = fmt.Sprintf("%d of %d contacts failed enrollment", failedCount, len(results))
	}
}

// countFailedEnrollments returns the number of failed enrollment attempts.
func (p *RecoveryEnrollmentPanel) countFailedEnrollments(results []recovery.EnrollmentResult) int {
	failedCount := 0
	for _, result := range results {
		if !result.Success {
			failedCount++
		}
	}
	return failedCount
}

// Draw renders the panel.
func (p *RecoveryEnrollmentPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	px, py, contentY := DrawModalWithTitle(screen, p.visible, p.width, p.height, *p.theme, "Social Recovery Setup")
	if px == 0 {
		return
	}

	// Draw content based on state
	switch p.state {
	case EnrollmentStateSelectContacts:
		p.drawContactSelection(screen, px, py, contentY)
	case EnrollmentStateConfigureThreshold:
		p.drawThresholdConfiguration(screen, px, py, contentY)
	case EnrollmentStateDistributing:
		p.drawDistributing(screen, px, contentY)
	case EnrollmentStateComplete:
		p.drawComplete(screen, px, py, contentY)
	case EnrollmentStateError:
		p.drawError(screen, px, contentY)
	}
}

// drawContactSelection renders contact selection UI.
func (p *RecoveryEnrollmentPanel) drawContactSelection(screen *ebiten.Image, px, py, startY int) {
	y := p.drawContactInstructions(screen, px, startY)
	y = p.drawContactList(screen, px, y)
	p.drawContactControls(screen, px, py, y)
}

// drawContactInstructions renders the instruction text at the top.
func (p *RecoveryEnrollmentPanel) drawContactInstructions(screen *ebiten.Image, px, startY int) int {
	instr := fmt.Sprintf("Select %d-%d contacts (currently: %d)", recovery.MinThreshold, recovery.MaxTotalShares, p.selectedCount)
	drawUIText(screen, instr, float64(px+20), float64(startY), p.theme.TextSecondary)
	return startY + 30
}

// drawContactList renders the scrollable contact list.
func (p *RecoveryEnrollmentPanel) drawContactList(screen *ebiten.Image, px, startY int) int {
	y := startY
	maxVisible := 8
	for i := 0; i < minInt(maxVisible, len(p.contacts)); i++ {
		idx := p.scrollOffset + i
		if idx >= len(p.contacts) {
			break
		}
		y = p.drawContactItem(screen, px, y, idx)
	}
	return y
}

// drawContactItem renders a single contact list item.
func (p *RecoveryEnrollmentPanel) drawContactItem(screen *ebiten.Image, px, y, idx int) int {
	contact := &p.contacts[idx]

	if idx == p.scrollOffset {
		vector.DrawFilledRect(screen, float32(px+15), float32(y-5),
			float32(p.width-30), 30, p.theme.PanelBorder, true)
	}

	checkbox := "[ ]"
	if contact.Selected {
		checkbox = "[X]"
	}
	drawUIText(screen, checkbox, float64(px+20), float64(y), p.theme.TextPrimary)

	label := contact.Label
	if label == "" {
		label = fmt.Sprintf("Contact %d", idx+1)
	}
	drawUIText(screen, label, float64(px+60), float64(y), p.theme.TextPrimary)

	return y + 35
}

// drawContactControls renders the control hints and error message.
func (p *RecoveryEnrollmentPanel) drawContactControls(screen *ebiten.Image, px, py, y int) {
	y = py + p.height - 80
	drawUIText(screen, "↑↓: Navigate  Space: Toggle  Enter: Next  Esc: Cancel",
		float64(px+20), float64(y), p.theme.TextSecondary)

	if p.errorMsg != "" {
		y += 25
		drawUIText(screen, p.errorMsg, float64(px+20), float64(y), p.theme.TextError)
	}
}

// drawThresholdConfiguration renders threshold configuration UI.
func (p *RecoveryEnrollmentPanel) drawThresholdConfiguration(screen *ebiten.Image, px, py, startY int) {
	y := startY

	// Explanation
	drawUIText(screen, fmt.Sprintf("Selected %d contacts", p.totalShares),
		float64(px+20), float64(y), p.theme.TextPrimary)
	y += 40

	drawUIText(screen, "Choose threshold (M of N):",
		float64(px+20), float64(y), p.theme.TextSecondary)
	y += 30

	// Threshold selector
	thresholdText := fmt.Sprintf("%d of %d contacts required for recovery", p.threshold, p.totalShares)
	drawUICenteredText(screen, thresholdText, float64(px+p.width/2), float64(y), p.theme.TextPrimary)
	y += 40

	// Recommendation
	recommendation := "Recommended: 3-of-5 for balance of security and convenience"
	drawUIText(screen, recommendation, float64(px+20), float64(y), p.theme.TextSecondary)
	y += 60

	// Warning
	warning := "⚠ You will need M contacts to recover. Don't set this too high!"
	drawUIText(screen, warning, float64(px+20), float64(y), p.theme.Warning)

	// Controls
	y = py + p.height - 80
	drawUIText(screen, "↑↓: Adjust Threshold  Enter: Confirm  Backspace: Back  Esc: Cancel",
		float64(px+20), float64(y), p.theme.TextSecondary)
}

// drawDistributing renders distributing state.
func (p *RecoveryEnrollmentPanel) drawDistributing(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Distributing encrypted shares...",
		float64(px+p.width/2), float64(y), p.theme.TextPrimary)
	y += 30
	drawUICenteredText(screen, "This may take a few moments",
		float64(px+p.width/2), float64(y), p.theme.TextSecondary)
}

// drawComplete renders completion state.
func (p *RecoveryEnrollmentPanel) drawComplete(screen *ebiten.Image, px, py, startY int) {
	y := startY + 60

	// Success message
	drawUICenteredText(screen, p.successMsg,
		float64(px+p.width/2), float64(y), p.theme.Success)
	y += 60

	// Results summary
	for _, result := range p.enrollmentResults {
		status := "✓"
		color := p.theme.Success
		if !result.Success {
			status = "✗"
			color = p.theme.TextError
		}
		text := fmt.Sprintf("%s %s", status, result.Contact.Label)
		drawUIText(screen, text, float64(px+40), float64(y), color)
		y += 25
	}

	// Instructions
	y = py + p.height - 60
	drawUICenteredText(screen, "Press Enter to close",
		float64(px+p.width/2), float64(y), p.theme.TextSecondary)
}

// drawError renders error state.
func (p *RecoveryEnrollmentPanel) drawError(screen *ebiten.Image, px, startY int) {
	y := startY + 100
	drawUICenteredText(screen, "Enrollment Error",
		float64(px+p.width/2), float64(y), p.theme.TextError)
	y += 40
	drawUICenteredText(screen, p.errorMsg,
		float64(px+p.width/2), float64(y), p.theme.TextSecondary)
	y += 60
	drawUICenteredText(screen, "Press Escape to close",
		float64(px+p.width/2), float64(y), p.theme.TextSecondary)
}

// countSelected counts selected contacts.
func (p *RecoveryEnrollmentPanel) countSelected() int {
	count := 0
	for _, c := range p.contacts {
		if c.Selected {
			count++
		}
	}
	return count
}

// Helper functions
func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
