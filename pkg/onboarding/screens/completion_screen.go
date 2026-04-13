// Package screens provides the Completion screen for onboarding Phase 6.
// Per ONBOARDING.md, this screen shows a summary of the created identity
// and prompts for invitation generation.
//
//go:build !noebiten
// +build !noebiten

package screens

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

// CompletionScreenState tracks completion sub-screens.
type CompletionScreenState int

const (
	CompletionStateSummary CompletionScreenState = iota
	CompletionStateInvite
	CompletionStateDone
)

// CompletionScreen handles Phase 6: Completion.
type CompletionScreen struct {
	state     CompletionScreenState
	startTime time.Time
	animPhase float64

	// Identity summary
	displayName    string
	surfaceSigil   *ebiten.Image
	surfaceKeypair *keys.KeyPair
	specterName    string
	specterSigil   *ebiten.Image
	selectedMode   modes.Mode
	peersConnected int

	// Invitation
	inviteGenerated bool
	inviteCode      string

	// UI state
	width, height int
	callbacks     CompletionScreenCallbacks
}

// CompletionScreenCallbacks provides hooks for completion events.
type CompletionScreenCallbacks struct {
	OnInviteGenerated  func(code string)
	OnOnboardingFinish func()
}

// NewCompletionScreen creates a new completion screen.
func NewCompletionScreen(
	displayName string,
	surfaceSigil *ebiten.Image,
	surfaceKeypair *keys.KeyPair,
	specterName string,
	specterSigil *ebiten.Image,
	selectedMode modes.Mode,
	peersConnected int,
	callbacks CompletionScreenCallbacks,
) *CompletionScreen {
	return &CompletionScreen{
		state:          CompletionStateSummary,
		startTime:      time.Now(),
		displayName:    displayName,
		surfaceSigil:   surfaceSigil,
		surfaceKeypair: surfaceKeypair,
		specterName:    specterName,
		specterSigil:   specterSigil,
		selectedMode:   selectedMode,
		peersConnected: peersConnected,
		callbacks:      callbacks,
	}
}

// Update advances animations.
func (s *CompletionScreen) Update() error {
	dt := 1.0 / 60.0
	s.animPhase += dt * 0.5
	if s.animPhase > 1 {
		s.animPhase -= 1
	}
	return nil
}

// Draw renders the completion screen.
func (s *CompletionScreen) Draw(screen *ebiten.Image) {
	s.width, s.height = screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dark background
	screen.Fill(color.RGBA{15, 15, 20, 255})

	switch s.state {
	case CompletionStateSummary:
		s.drawSummary(screen)
	case CompletionStateInvite:
		s.drawInvite(screen)
	case CompletionStateDone:
		s.drawDone(screen)
	}
}

// drawSummary renders the identity summary.
func (s *CompletionScreen) drawSummary(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height) / 2

	// Title
	titleY := float32(50)
	s.drawCenteredText(screen, "Your Identity", centerX, titleY, 24, color.RGBA{220, 220, 225, 255})

	// Identity card(s) based on mode
	cardY := centerY - 60

	switch s.selectedMode {
	case modes.Open:
		// Single Surface identity card
		s.drawIdentityCard(screen, centerX, cardY, "Surface Identity", s.displayName, s.surfaceSigil, false)

	case modes.Hybrid:
		// Both identities side by side
		s.drawIdentityCard(screen, centerX-130, cardY, "Surface", s.displayName, s.surfaceSigil, false)
		s.drawIdentityCard(screen, centerX+130, cardY, "Specter", s.specterName, s.specterSigil, true)

	case modes.Fortress:
		// Single Specter identity card
		s.drawIdentityCard(screen, centerX, cardY, "Specter Identity", s.specterName, s.specterSigil, true)
	}

	// Mode badge
	modeY := cardY + 140
	modeColor := getModeColor(s.selectedMode)
	s.drawCenteredText(screen, s.selectedMode.String()+" Mode", centerX, modeY, 14, modeColor)

	// Connection status
	statusY := modeY + 30
	connText := "Connected to " + itoa(s.peersConnected) + " peers"
	s.drawCenteredText(screen, connText, centerX, statusY, 12, color.RGBA{130, 170, 130, 255})

	// Buttons
	buttonY := float32(s.height) - 100
	s.drawButton(screen, "Invite Friends", centerX-100, buttonY, 0)
	s.drawButton(screen, "Enter MURMUR", centerX+100, buttonY, 1)
}

// drawIdentityCard renders an identity card.
func (s *CompletionScreen) drawIdentityCard(screen *ebiten.Image, x, y float32, label, name string, sigil *ebiten.Image, isSpecter bool) {
	cardWidth := float32(200)
	cardHeight := float32(180)
	cardX := x - cardWidth/2
	cardY := y - cardHeight/2

	// Card background with glow effect
	glowRadius := float32(3 + 2*math.Sin(s.animPhase*2*math.Pi))
	glowColor := color.RGBA{80, 100, 200, 30}
	if !isSpecter {
		glowColor = color.RGBA{200, 160, 80, 30}
	}
	vector.DrawFilledRect(screen, cardX-glowRadius, cardY-glowRadius, cardWidth+2*glowRadius, cardHeight+2*glowRadius, glowColor, true)

	// Card background
	bgColor := color.RGBA{30, 35, 50, 255}
	if isSpecter {
		bgColor = color.RGBA{25, 30, 55, 255}
	}
	vector.DrawFilledRect(screen, cardX, cardY, cardWidth, cardHeight, bgColor, true)

	// Border
	borderColor := color.RGBA{70, 80, 120, 255}
	if isSpecter {
		borderColor = color.RGBA{80, 100, 180, 255}
	}
	vector.StrokeRect(screen, cardX, cardY, cardWidth, cardHeight, 1.5, borderColor, true)

	// Label
	labelY := cardY + 20
	labelColor := color.RGBA{150, 150, 160, 255}
	if isSpecter {
		labelColor = color.RGBA{120, 150, 200, 255}
	}
	s.drawCenteredText(screen, label, x, labelY, 11, labelColor)

	// Sigil
	if sigil != nil {
		sigilOpts := &ebiten.DrawImageOptions{}
		sigilX := x - float32(sigil.Bounds().Dx())/2
		sigilY := cardY + 40
		sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
		screen.DrawImage(sigil, sigilOpts)
	}

	// Name
	nameY := cardY + cardHeight - 35
	nameColor := color.RGBA{200, 200, 210, 255}
	if isSpecter {
		nameColor = color.RGBA{140, 170, 230, 255}
	}
	displayName := name
	if displayName == "" {
		displayName = "(Anonymous)"
	}
	s.drawCenteredText(screen, displayName, x, nameY, 14, nameColor)
}

// drawInvite renders the invitation generation screen.
func (s *CompletionScreen) drawInvite(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 30

	// Title
	titleY := float32(60)
	s.drawCenteredText(screen, "Invite Friends", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	// Description
	descY := float32(110)
	s.drawCenteredText(screen, "MURMUR grows through trusted connections.", centerX, descY, 14, color.RGBA{150, 150, 160, 255})
	s.drawCenteredText(screen, "Share this invite code with people you trust.", centerX, descY+22, 14, color.RGBA{150, 150, 160, 255})

	// Invite code display
	codeY := centerY
	codeBoxWidth := float32(300)
	codeBoxHeight := float32(60)
	codeBoxX := centerX - codeBoxWidth/2

	codeBgColor := color.RGBA{25, 30, 45, 255}
	vector.DrawFilledRect(screen, codeBoxX, codeY, codeBoxWidth, codeBoxHeight, codeBgColor, true)
	codeBorderColor := color.RGBA{100, 120, 180, 255}
	vector.StrokeRect(screen, codeBoxX, codeY, codeBoxWidth, codeBoxHeight, 1.5, codeBorderColor, true)

	if s.inviteCode != "" {
		codeTextY := codeY + codeBoxHeight/2 + 6
		s.drawCenteredText(screen, s.inviteCode, centerX, codeTextY, 18, color.RGBA{180, 200, 255, 255})
	} else {
		placeholderY := codeY + codeBoxHeight/2 + 6
		s.drawCenteredText(screen, "Generating...", centerX, placeholderY, 14, color.RGBA{100, 100, 110, 255})
	}

	// Copy/generate button
	if s.inviteCode == "" {
		s.drawButton(screen, "Generate Invite", centerX, codeY+100, 0)
	} else {
		s.drawButton(screen, "Copy to Clipboard", centerX, codeY+100, 0)
	}

	// Continue button
	buttonY := float32(s.height) - 100
	s.drawButton(screen, "Continue", centerX, buttonY, 1)
}

// drawDone renders the final completion message.
func (s *CompletionScreen) drawDone(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 50

	// Title
	titleY := float32(60)
	s.drawCenteredText(screen, "Welcome to MURMUR", centerX, titleY, 26, color.RGBA{220, 220, 225, 255})

	// Animated success circle
	pulse := float32(1 + 0.1*math.Sin(s.animPhase*2*math.Pi))
	successColor := color.RGBA{100, 200, 130, 255}
	vector.DrawFilledCircle(screen, centerX, centerY, 50*pulse, successColor, true)

	// Checkmark
	checkColor := color.RGBA{240, 240, 245, 255}
	vector.StrokeLine(screen, centerX-20, centerY, centerX-5, centerY+20, 4, checkColor, true)
	vector.StrokeLine(screen, centerX-5, centerY+20, centerX+25, centerY-15, 4, checkColor, true)

	// Completion message
	msgY := centerY + 100
	s.drawCenteredText(screen, "Your identity is created.", centerX, msgY, 14, color.RGBA{150, 150, 160, 255})
	s.drawCenteredText(screen, "You're connected to the network.", centerX, msgY+22, 14, color.RGBA{150, 150, 160, 255})
	s.drawCenteredText(screen, "The Pulse Map awaits.", centerX, msgY+44, 14, color.RGBA{150, 150, 160, 255})

	// Enter button
	buttonY := float32(s.height) - 100
	s.drawButton(screen, "Enter the Network", centerX, buttonY, 0)
}

func getModeColor(mode modes.Mode) color.RGBA {
	switch mode {
	case modes.Open:
		return color.RGBA{230, 180, 100, 255}
	case modes.Hybrid:
		return color.RGBA{150, 180, 200, 255}
	case modes.Fortress:
		return color.RGBA{100, 150, 230, 255}
	default:
		return color.RGBA{150, 150, 160, 255}
	}
}

// HandleClick processes clicks on the completion screen.
func (s *CompletionScreen) HandleClick(x, y int) {
	centerX := s.width / 2

	switch s.state {
	case CompletionStateSummary:
		buttonY := s.height - 100
		// Invite Friends button
		if s.isClickOnButton(x, y, centerX-100, buttonY) {
			s.state = CompletionStateInvite
		}
		// Enter MURMUR button
		if s.isClickOnButton(x, y, centerX+100, buttonY) {
			s.finish()
		}

	case CompletionStateInvite:
		codeY := s.height/2 - 30
		// Generate/Copy button
		if s.isClickOnButton(x, y, centerX, codeY+100) {
			if s.inviteCode == "" {
				s.generateInvite()
			} else {
				// Copy to clipboard - would need OS integration
			}
		}
		// Continue button
		buttonY := s.height - 100
		if s.isClickOnButton(x, y, centerX, buttonY) {
			s.state = CompletionStateDone
		}

	case CompletionStateDone:
		buttonY := s.height - 100
		if s.isClickOnButton(x, y, centerX, buttonY) {
			s.finish()
		}
	}
}

func (s *CompletionScreen) generateInvite() {
	// Generate a simple invite code from public key
	if s.surfaceKeypair != nil {
		s.inviteCode = generateInviteCode(s.surfaceKeypair.PublicKey)
	} else {
		s.inviteCode = "MURMUR-XXXX-YYYY"
	}
	s.inviteGenerated = true

	if s.callbacks.OnInviteGenerated != nil {
		s.callbacks.OnInviteGenerated(s.inviteCode)
	}
}

func generateInviteCode(pubKey []byte) string {
	// Simple invite code generation from public key prefix
	if len(pubKey) < 6 {
		return "MURMUR-XXXX-YYYY"
	}
	return "MURMUR-" + hexNibble(pubKey[0]) + hexNibble(pubKey[1]) + hexNibble(pubKey[2]) +
		"-" + hexNibble(pubKey[3]) + hexNibble(pubKey[4]) + hexNibble(pubKey[5])
}

func hexNibble(b byte) string {
	const hex = "0123456789ABCDEF"
	return string(hex[(b>>4)&0x0F]) + string(hex[b&0x0F])
}

func (s *CompletionScreen) finish() {
	if s.callbacks.OnOnboardingFinish != nil {
		s.callbacks.OnOnboardingFinish()
	}
}

func (s *CompletionScreen) isClickOnButton(clickX, clickY, buttonCenterX, buttonCenterY int) bool {
	width := 160
	height := 40
	bx := buttonCenterX - width/2
	by := buttonCenterY - height/2
	return clickX >= bx && clickX <= bx+width && clickY >= by && clickY <= by+height
}

func (s *CompletionScreen) drawButton(screen *ebiten.Image, label string, x, y float32, index int) {
	width := float32(160)
	height := float32(40)
	bx := x - width/2
	by := y - height/2

	bgColor := color.RGBA{60, 70, 100, 255}
	vector.DrawFilledRect(screen, bx, by, width, height, bgColor, true)

	borderColor := color.RGBA{100, 120, 180, 255}
	vector.StrokeRect(screen, bx, by, width, height, 1.5, borderColor, true)

	s.drawCenteredText(screen, label, x, y+6, 13, color.RGBA{220, 220, 230, 255})
}

func (s *CompletionScreen) drawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	charWidth := float32(size) * 0.5
	textWidth := float32(len(str)) * charWidth
	textX := x - textWidth/2

	charColor := clr.(color.RGBA)
	charH := float32(size) * 0.8
	charW := charWidth * 0.8
	charY := y - charH/2

	for i, char := range str {
		if char == ' ' {
			continue
		}
		charX := textX + float32(i)*charWidth
		vector.DrawFilledRect(screen, charX, charY, charW, charH, charColor, true)
	}
}

// State returns the current screen state.
func (s *CompletionScreen) CompletionState() CompletionScreenState {
	return s.state
}

// InviteCode returns the generated invite code.
func (s *CompletionScreen) InviteCode() string {
	return s.inviteCode
}

// IsInviteGenerated returns whether an invite was generated.
func (s *CompletionScreen) IsInviteGenerated() bool {
	return s.inviteGenerated
}
