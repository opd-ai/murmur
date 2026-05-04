// Package screens provides the Mode Selection screen for onboarding Phase 3.
// Per ONBOARDING.md, this screen introduces the dual-layer architecture and
// allows users to choose between Open, Hybrid, and Fortress modes.
//

//go:build !test
// +build !test

package screens

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/identity/sigils"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// ModeScreenState tracks mode selection sub-screens.
type ModeScreenState int

const (
	ModeStateIntro ModeScreenState = iota
	ModeStateCards
	ModeStateSpecterGen
	ModeStateConfirmation
)

// ModeScreen handles the mode selection phase.
type ModeScreen struct {
	controller *flow.Controller
	state      ModeScreenState
	startTime  time.Time
	animPhase  float64

	// User's surface identity (from Phase 1-2)
	surfaceKeypair *keys.KeyPair
	surfaceSigil   *ebiten.Image
	displayName    string

	// Specter generation (for Hybrid/Fortress)
	specterKeypair *keys.AnonymousKeyPair
	specterSigil   *ebiten.Image
	specterName    string

	// Mode selection
	selectedMode modes.Mode
	hoverMode    modes.Mode

	// Animation
	introProgress float64
	introStart    time.Time

	// UI state
	width, height int
	callbacks     ModeScreenCallbacks
}

// ModeScreenCallbacks provides hooks for mode selection events.
type ModeScreenCallbacks struct {
	OnModeSelected     func(modes.Mode)
	OnSpecterGenerated func(*keys.AnonymousKeyPair, string)
	OnPhaseComplete    func(flow.Phase)
}

// NewModeScreen creates a new mode selection screen.
func NewModeScreen(
	controller *flow.Controller,
	surfaceKP *keys.KeyPair,
	surfaceSigil *ebiten.Image,
	displayName string,
	callbacks ModeScreenCallbacks,
) *ModeScreen {
	return &ModeScreen{
		controller:     controller,
		state:          ModeStateIntro,
		startTime:      time.Now(),
		introStart:     time.Now(),
		surfaceKeypair: surfaceKP,
		surfaceSigil:   surfaceSigil,
		displayName:    displayName,
		selectedMode:   modes.Hybrid, // Default recommendation
		hoverMode:      modes.Hybrid,
		callbacks:      callbacks,
	}
}

// Update advances animations and handles timing.
func (s *ModeScreen) Update() error {
	dt := 1.0 / 60.0
	s.animPhase += dt * 0.5
	if s.animPhase > 1 {
		s.animPhase -= 1
	}

	// Introduction animation timing (5 seconds per ONBOARDING.md)
	if s.state == ModeStateIntro {
		s.introProgress = math.Min(1.0, time.Since(s.introStart).Seconds()/5.0)
	}

	return nil
}

// Draw renders the mode selection screen.
func (s *ModeScreen) Draw(screen *ebiten.Image) {
	s.width, s.height = screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dark background
	screen.Fill(color.RGBA{15, 15, 20, 255})

	switch s.state {
	case ModeStateIntro:
		s.drawModeIntro(screen)
	case ModeStateCards:
		s.drawModeCards(screen)
	case ModeStateSpecterGen:
		s.drawSpecterGeneration(screen)
	case ModeStateConfirmation:
		s.drawConfirmation(screen)
	}
}

// drawModeIntro renders the animated introduction to dual-layer architecture.
// Per ONBOARDING.md: warm Surface node, cool Anonymous Layer fading in behind.
func (s *ModeScreen) drawModeIntro(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 30

	s.drawCenteredText(screen, "Choose How You Participate", centerX, 60, 22, color.RGBA{220, 220, 225, 255})
	s.drawSurfaceLayerNode(screen, centerX-30, centerY)

	if s.introProgress > 0.2 {
		s.drawAnonymousLayerNode(screen, centerX+30, centerY+10)
	}
	if s.introProgress > 0.5 {
		s.drawIntroExplanationText(screen, centerX, centerY+100)
	}
	if s.introProgress >= 1.0 {
		s.drawButton(screen, "Continue", centerX, float32(s.height)-100, 0)
	}
}

// drawSurfaceLayerNode renders the warm-toned Surface layer node.
func (s *ModeScreen) drawSurfaceLayerNode(screen *ebiten.Image, x, y float32) {
	pulse := float32(0.5 + 0.5*math.Sin(s.animPhase*2*math.Pi))
	nodeColor := color.RGBA{230, 180, 100, uint8(200 + 55*pulse)}
	vector.DrawFilledCircle(screen, x, y, 25, nodeColor, true)
}

// drawAnonymousLayerNode renders the anonymous layer node with particles and glow.
func (s *ModeScreen) drawAnonymousLayerNode(screen *ebiten.Image, x, y float32) {
	alpha := uint8(255 * math.Min(1, (s.introProgress-0.2)/0.3))

	s.drawAnonParticles(screen, x, y, alpha)
	vector.DrawFilledCircle(screen, x, y, 20, color.RGBA{100, 150, 230, alpha}, true)
	vector.DrawFilledCircle(screen, x, y, 30, color.RGBA{80, 120, 200, uint8(float64(alpha) * 0.4)}, true)
}

// drawAnonParticles renders orbiting particles around the anonymous layer node.
func (s *ModeScreen) drawAnonParticles(screen *ebiten.Image, x, y float32, alpha uint8) {
	for i := 0; i < 8; i++ {
		angle := float64(i)*math.Pi/4 + s.animPhase*2*math.Pi
		px := x + 40*float32(math.Cos(angle))
		py := y + 40*float32(math.Sin(angle))
		vector.DrawFilledCircle(screen, px, py, 3, color.RGBA{100, 100, 200, uint8(float64(alpha) * 0.3)}, true)
	}
}

// drawIntroExplanationText renders the dual-layer explanation text.
func (s *ModeScreen) drawIntroExplanationText(screen *ebiten.Image, centerX, textY float32) {
	alpha := uint8(255 * math.Min(1, (s.introProgress-0.5)/0.3))
	textColor := color.RGBA{180, 180, 190, alpha}

	s.drawCenteredText(screen, "MURMUR has two layers.", centerX, textY, 16, textColor)
	s.drawCenteredText(screen, "The Surface Layer is public — your identity is visible.", centerX, textY+25, 14, textColor)
	s.drawCenteredText(screen, "The Anonymous Layer is private — participate through", centerX, textY+50, 14, textColor)
	s.drawCenteredText(screen, "an anonymous identity that cannot be linked to you.", centerX, textY+70, 14, textColor)
}

// drawModeCards renders the four mode selection cards.
func (s *ModeScreen) drawModeCards(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	// Title
	titleY := float32(50)
	s.drawCenteredText(screen, "Select Your Mode", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	// Mode cards layout
	cardWidth := float32(180)
	cardHeight := float32(240)
	cardSpacing := float32(20)
	totalWidth := 4*cardWidth + 3*cardSpacing
	startX := centerX - totalWidth/2
	cardY := float32(100)

	// Draw each mode card
	modesList := []modes.Mode{modes.Open, modes.Hybrid, modes.Guarded, modes.Fortress}
	for i, mode := range modesList {
		cardX := startX + float32(i)*(cardWidth+cardSpacing)
		s.drawModeCard(screen, mode, cardX, cardY, cardWidth, cardHeight, i)
	}

	// Guidance panel
	guidanceY := cardY + cardHeight + 30
	s.drawGuidancePanel(screen, centerX, guidanceY)

	// Confirm button
	buttonY := float32(s.height) - 80
	s.drawButton(screen, "Select "+s.selectedMode.String(), centerX, buttonY, 3)
}

// drawModeCard renders a single mode card.
func (s *ModeScreen) drawModeCard(screen *ebiten.Image, mode modes.Mode, x, y, w, h float32, index int) {
	isSelected := s.selectedMode == mode
	isHovered := s.hoverMode == mode
	iconCenterX := x + w/2

	s.drawCardBackground(screen, x, y, w, h, isSelected, isHovered)
	s.drawModeIcon(screen, mode, iconCenterX, y+40)
	s.drawCardLabels(screen, mode, iconCenterX, y)
}

// drawCardBackground renders the card frame with selection/hover styling.
func (s *ModeScreen) drawCardBackground(screen *ebiten.Image, x, y, w, h float32, isSelected, isHovered bool) {
	bgColor := color.RGBA{30, 35, 50, 255}
	if isSelected {
		bgColor = color.RGBA{50, 60, 90, 255}
	} else if isHovered {
		bgColor = color.RGBA{40, 45, 70, 255}
	}
	vector.DrawFilledRect(screen, x, y, w, h, bgColor, true)

	borderColor := color.RGBA{70, 80, 120, 255}
	if isSelected {
		borderColor = color.RGBA{120, 160, 255, 255}
	}
	vector.StrokeRect(screen, x, y, w, h, 2, borderColor, true)
}

// drawModeIcon renders the mode-specific icon at the given position.
func (s *ModeScreen) drawModeIcon(screen *ebiten.Image, mode modes.Mode, iconCenterX, iconY float32) {
	switch mode {
	case modes.Open:
		iconColor := color.RGBA{230, 180, 100, 255}
		vector.DrawFilledCircle(screen, iconCenterX, iconY, 20, iconColor, true)
	case modes.Hybrid:
		warmColor := color.RGBA{230, 180, 100, 200}
		coolColor := color.RGBA{100, 150, 230, 200}
		vector.DrawFilledCircle(screen, iconCenterX-12, iconY, 18, warmColor, true)
		vector.DrawFilledCircle(screen, iconCenterX+12, iconY+5, 16, coolColor, true)
	case modes.Guarded:
		warmColor := color.RGBA{200, 160, 100, 150}
		coolColor := color.RGBA{120, 140, 200, 255}
		vector.DrawFilledCircle(screen, iconCenterX, iconY, 18, coolColor, true)
		guardColor := color.RGBA{160, 140, 180, 200}
		vector.StrokeCircle(screen, iconCenterX, iconY, 22, 2, guardColor, true)
		vector.DrawFilledCircle(screen, iconCenterX-8, iconY-3, 6, warmColor, true)
	case modes.Fortress:
		coolColor := color.RGBA{100, 150, 230, 255}
		vector.DrawFilledCircle(screen, iconCenterX, iconY, 18, coolColor, true)
		shieldColor := color.RGBA{140, 160, 200, 180}
		vector.StrokeCircle(screen, iconCenterX, iconY, 25, 2, shieldColor, true)
	}
}

// drawCardLabels renders the mode name, description, and properties.
func (s *ModeScreen) drawCardLabels(screen *ebiten.Image, mode modes.Mode, iconCenterX, cardY float32) {
	nameY := cardY + 80
	s.drawCenteredText(screen, mode.String(), iconCenterX, nameY, 16, color.RGBA{220, 220, 230, 255})

	descY := cardY + 110
	s.drawCenteredText(screen, getModeDescription(mode), iconCenterX, descY, 11, color.RGBA{150, 150, 160, 255})

	for i, prop := range getModeProperties(mode) {
		propY := cardY + 140 + float32(i)*18
		s.drawCenteredText(screen, "• "+prop, iconCenterX, propY, 10, color.RGBA{130, 130, 140, 255})
	}
}

// drawGuidancePanel renders context-sensitive guidance.
func (s *ModeScreen) drawGuidancePanel(screen *ebiten.Image, centerX, y float32) {
	guidance := getModeGuidance(s.selectedMode)

	// Panel background
	panelWidth := float32(500)
	panelHeight := float32(60)
	panelX := centerX - panelWidth/2
	panelColor := color.RGBA{25, 30, 45, 255}
	vector.DrawFilledRect(screen, panelX, y, panelWidth, panelHeight, panelColor, true)

	// Guidance text
	textY := y + panelHeight/2 + 5
	s.drawCenteredText(screen, guidance, centerX, textY, 12, color.RGBA{170, 170, 180, 255})
}

// drawSpecterGeneration renders the Specter identity creation.
func (s *ModeScreen) drawSpecterGeneration(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 40

	titleY := float32(60)
	s.drawCenteredText(screen, "Creating Your Specter", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	if s.specterKeypair == nil {
		s.drawSpecterAnimation(screen, centerX, centerY)
	} else {
		s.drawSpecterComplete(screen, centerX, centerY)
	}
}

// drawSpecterAnimation renders the dark particle coalescence animation.
func (s *ModeScreen) drawSpecterAnimation(screen *ebiten.Image, centerX, centerY float32) {
	progress := math.Min(1, time.Since(s.startTime).Seconds()/2.5)
	s.drawCoalescingParticles(screen, centerX, centerY, progress)
	s.drawSpecterCore(screen, centerX, centerY, progress)

	if progress >= 1 {
		go s.generateSpecter()
	}
}

// drawCoalescingParticles renders dark particles converging.
func (s *ModeScreen) drawCoalescingParticles(screen *ebiten.Image, centerX, centerY float32, progress float64) {
	for i := 0; i < 20; i++ {
		angle := float64(i)*2*math.Pi/20 + s.animPhase*3*math.Pi
		radius := float32(80 * (1 - progress*0.9))
		px := centerX + radius*float32(math.Cos(angle))
		py := centerY + radius*float32(math.Sin(angle))
		pColor := color.RGBA{80, 100, 200, uint8(150 + 105*progress)}
		vector.DrawFilledCircle(screen, px, py, float32(3+2*(1-progress)), pColor, true)
	}
}

// drawSpecterCore renders the growing specter core.
func (s *ModeScreen) drawSpecterCore(screen *ebiten.Image, centerX, centerY float32, progress float64) {
	coreRadius := float32(5 + 15*progress)
	coreColor := color.RGBA{100, 140, 230, uint8(180 + 75*progress)}
	vector.DrawFilledCircle(screen, centerX, centerY, coreRadius, coreColor, true)
}

// drawSpecterComplete renders the completed Specter identity.
func (s *ModeScreen) drawSpecterComplete(screen *ebiten.Image, centerX, centerY float32) {
	if s.specterSigil != nil {
		sigilOpts := &ebiten.DrawImageOptions{}
		sigilX := centerX - float32(s.specterSigil.Bounds().Dx())/2
		sigilY := centerY - float32(s.specterSigil.Bounds().Dy())/2
		sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
		screen.DrawImage(s.specterSigil, sigilOpts)
	}

	nameY := centerY + 80
	s.drawCenteredText(screen, s.specterName, centerX, nameY, 18, color.RGBA{140, 170, 230, 255})

	explainY := nameY + 40
	s.drawCenteredText(screen, "This is your Specter — your anonymous identity", centerX, explainY, 14, color.RGBA{150, 150, 160, 255})
	s.drawCenteredText(screen, "on the Anonymous Layer.", centerX, explainY+20, 14, color.RGBA{150, 150, 160, 255})

	buttonY := float32(s.height) - 100
	s.drawButton(screen, "Continue", centerX, buttonY, 0)
}

// drawConfirmation renders the mode confirmation screen.
func (s *ModeScreen) drawConfirmation(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	// Title
	titleY := float32(60)
	s.drawCenteredText(screen, "Your Identity Configuration", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	centerY := float32(s.height) / 2

	switch s.selectedMode {
	case modes.Open:
		// Show only Surface identity
		s.drawIdentityCard(screen, centerX, centerY-40, "Surface Identity", s.displayName, s.surfaceSigil, false)

	case modes.Hybrid:
		// Show both identities side by side
		s.drawIdentityCard(screen, centerX-120, centerY-40, "Surface Identity", s.displayName, s.surfaceSigil, false)
		s.drawIdentityCard(screen, centerX+120, centerY-40, "Specter Identity", s.specterName, s.specterSigil, true)

	case modes.Fortress:
		// Show only Specter identity
		s.drawIdentityCard(screen, centerX, centerY-40, "Specter Identity", s.specterName, s.specterSigil, true)
	}

	// Enter the Network button
	buttonY := float32(s.height) - 80
	s.drawButton(screen, "Enter the Network", centerX, buttonY, 0)
}

// drawIdentityCard renders an identity preview card using the shared helper.
func (s *ModeScreen) drawIdentityCard(screen *ebiten.Image, x, y float32, label, name string, sigil *ebiten.Image, isSpecter bool) {
	DrawIdentityCard(screen, x, y, label, name, sigil, isSpecter, DefaultIdentityCardStyle())
}

// generateSpecter creates the Specter keypair.
func (s *ModeScreen) generateSpecter() {
	kp, err := keys.GenerateAnonymousKeyPair()
	if err != nil {
		return
	}

	s.specterKeypair = kp

	// Generate Specter sigil
	sigilData := sigils.GenerateSpecter(kp.PublicKey[:])
	s.specterSigil = ebiten.NewImageFromImage(sigilData.Image)

	// Generate Specter name using shared name generator
	s.specterName = GenerateSpecterName(kp.PublicKey[:])

	if s.callbacks.OnSpecterGenerated != nil {
		s.callbacks.OnSpecterGenerated(kp, s.specterName)
	}
}

func getModeDescription(mode modes.Mode) string {
	switch mode {
	case modes.Open:
		return "Participate publicly"
	case modes.Hybrid:
		return "Both layers, separate identities"
	case modes.Guarded:
		return "Enhanced privacy, limited exposure"
	case modes.Fortress:
		return "Anonymous Layer only"
	default:
		return ""
	}
}

func getModeProperties(mode modes.Mode) []string {
	switch mode {
	case modes.Open:
		return []string{"Identity visible", "Surface Layer only", "Simplest experience"}
	case modes.Hybrid:
		return []string{"Public + Anonymous", "Unlinked identities", "Recommended"}
	case modes.Guarded:
		return []string{"Selective visibility", "Both layers active", "Balanced privacy"}
	case modes.Fortress:
		return []string{"Maximum privacy", "No public identity", "Advanced mode"}
	default:
		return nil
	}
}

func getModeGuidance(mode modes.Mode) string {
	switch mode {
	case modes.Open:
		return "Open mode is ideal for a straightforward social experience. You can upgrade to Hybrid later."
	case modes.Hybrid:
		return "Hybrid mode gives you the full experience with separate identities on both layers. Recommended."
	case modes.Guarded:
		return "Guarded mode provides enhanced privacy controls while maintaining access to both Surface and Anonymous layers."
	case modes.Fortress:
		return "Fortress mode provides maximum anonymity but limits you to the Anonymous Layer only."
	default:
		return ""
	}
}

// HandleClick processes clicks on the mode selection screen.
func (s *ModeScreen) HandleClick(x, y int) {
	switch s.state {
	case ModeStateIntro:
		s.handleIntroClick(x, y)
	case ModeStateCards:
		s.handleCardsClick(x, y)
	case ModeStateSpecterGen:
		s.handleSpecterGenClick(x, y)
	case ModeStateConfirmation:
		s.handleConfirmationClick(x, y)
	}
}

// handleIntroClick processes clicks in the intro state.
func (s *ModeScreen) handleIntroClick(x, y int) {
	if s.introProgress < 1.0 {
		return
	}
	buttonY := s.height - 100
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.state = ModeStateCards
	}
}

// handleCardsClick processes clicks in the card selection state.
func (s *ModeScreen) handleCardsClick(x, y int) {
	s.checkModeCardClick(x, y)
	s.checkConfirmButtonClick(x, y)
}

// checkModeCardClick checks if a mode card was clicked and updates selection.
func (s *ModeScreen) checkModeCardClick(x, y int) {
	cardWidth, cardHeight, cardSpacing := 180, 240, 20
	totalWidth := 4*cardWidth + 3*cardSpacing
	startX := s.width/2 - totalWidth/2
	cardY := 100

	for i, mode := range []modes.Mode{modes.Open, modes.Hybrid, modes.Guarded, modes.Fortress} {
		cardX := startX + i*(cardWidth+cardSpacing)
		if s.isClickInRect(x, y, cardX, cardY, cardWidth, cardHeight) {
			s.selectedMode = mode
			s.hoverMode = mode
			return
		}
	}
}

// checkConfirmButtonClick handles confirm button clicks in card selection.
func (s *ModeScreen) checkConfirmButtonClick(x, y int) {
	buttonY := s.height - 80
	if !s.isClickOnButton(x, y, s.width/2, buttonY) {
		return
	}
	if s.callbacks.OnModeSelected != nil {
		s.callbacks.OnModeSelected(s.selectedMode)
	}
	s.transitionFromCardSelection()
}

// transitionFromCardSelection moves to the next state after mode selection.
func (s *ModeScreen) transitionFromCardSelection() {
	if s.selectedMode == modes.Open {
		s.state = ModeStateConfirmation
	} else {
		s.state = ModeStateSpecterGen
		s.startTime = time.Now()
	}
}

// handleSpecterGenClick processes clicks in the specter generation state.
func (s *ModeScreen) handleSpecterGenClick(x, y int) {
	if s.specterKeypair == nil {
		return
	}
	buttonY := s.height - 100
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.state = ModeStateConfirmation
	}
}

// handleConfirmationClick processes clicks in the confirmation state.
func (s *ModeScreen) handleConfirmationClick(x, y int) {
	buttonY := s.height - 80
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.completePhase3()
	}
}

// isClickInRect checks if coordinates are within a rectangle.
func (s *ModeScreen) isClickInRect(x, y, rectX, rectY, width, height int) bool {
	return x >= rectX && x <= rectX+width && y >= rectY && y <= rectY+height
}

// HandleMouseMove updates hover state for mode cards.
func (s *ModeScreen) HandleMouseMove(x, y int) {
	if s.state != ModeStateCards {
		return
	}

	centerX := s.width / 2
	cardWidth := 180
	cardHeight := 240
	cardSpacing := 20
	totalWidth := 4*cardWidth + 3*cardSpacing
	startX := centerX - totalWidth/2
	cardY := 100

	s.hoverMode = s.selectedMode // Reset to selected if not hovering

	for i, mode := range []modes.Mode{modes.Open, modes.Hybrid, modes.Guarded, modes.Fortress} {
		cardX := startX + i*(cardWidth+cardSpacing)
		if x >= cardX && x <= cardX+cardWidth && y >= cardY && y <= cardY+cardHeight {
			s.hoverMode = mode
		}
	}
}

func (s *ModeScreen) completePhase3() {
	s.controller.CompleteCurrentPhase() // PhaseModeSelection -> PhaseNetworkBootstrap

	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseModeSelection)
	}
}

func (s *ModeScreen) isClickOnButton(clickX, clickY, buttonCenterX, buttonCenterY int) bool {
	width := 200
	height := 44
	bx := buttonCenterX - width/2
	by := buttonCenterY - height/2
	return clickX >= bx && clickX <= bx+width && clickY >= by && clickY <= by+height
}

func (s *ModeScreen) drawButton(screen *ebiten.Image, label string, x, y float32, index int) {
	style := DefaultButtonStyle()
	style.Width = 200
	style.Height = 44
	style.TextSize = 16
	DrawButton(screen, label, x, y, style)
}

// drawCenteredText delegates to the shared DrawCenteredText helper.
func (s *ModeScreen) drawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	DrawCenteredText(screen, str, x, y, size, clr)
}

// GetSelectedMode returns the selected mode.
func (s *ModeScreen) GetSelectedMode() modes.Mode {
	return s.selectedMode
}

// GetSpecterKeypair returns the generated Specter keypair.
func (s *ModeScreen) GetSpecterKeypair() *keys.AnonymousKeyPair {
	return s.specterKeypair
}

// GetSpecterName returns the generated Specter name.
func (s *ModeScreen) GetSpecterName() string {
	return s.specterName
}

// State returns the current screen state.
func (s *ModeScreen) ModeState() ModeScreenState {
	return s.state
}
