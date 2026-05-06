// Package screens provides the Bootstrap and Exploration screens for onboarding Phase 4-5.
// Per ONBOARDING.md, these screens handle peer discovery and Pulse Map introduction.
//

//go:build !test
// +build !test

package screens

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// BootstrapScreenState tracks bootstrap sub-screens.
type BootstrapScreenState int

const (
	BootstrapStateConnecting BootstrapScreenState = iota
	BootstrapStatePulseMapIntro
	BootstrapStateFirstWavePrompt
	BootstrapStateComplete
)

// BootstrapScreen handles Phase 4-5: Bootstrap + Exploration.
type BootstrapScreen struct {
	controller *flow.Controller
	state      BootstrapScreenState
	startTime  time.Time
	animPhase  float64

	// Peer discovery
	peersFound     int
	targetPeers    int
	discoveryDone  bool
	discoveryStart time.Time

	// Pulse Map exploration
	pulseMapSeen  bool
	nodesInView   int
	tutorialStep  int
	tutorialShown []bool

	// First Wave
	firstWaveText string
	waveSent      bool

	// UI state
	width, height int
	callbacks     BootstrapScreenCallbacks
}

// BootstrapScreenCallbacks provides hooks for bootstrap events.
type BootstrapScreenCallbacks struct {
	OnPeerDiscoveryStart func()
	OnPeerFound          func(count int)
	OnPulseMapReady      func()
	OnFirstWaveSent      func(text string)
	OnPhaseComplete      func(flow.Phase)
}

// NewBootstrapScreen creates a new bootstrap/exploration screen.
func NewBootstrapScreen(controller *flow.Controller, callbacks BootstrapScreenCallbacks) *BootstrapScreen {
	return &BootstrapScreen{
		controller:     controller,
		state:          BootstrapStateConnecting,
		startTime:      time.Now(),
		discoveryStart: time.Now(),
		targetPeers:    6,
		tutorialShown:  make([]bool, 5),
		callbacks:      callbacks,
	}
}

// Update advances animations and handles state transitions.
func (s *BootstrapScreen) Update() error {
	dt := 1.0 / 60.0
	s.animPhase += dt * 0.5
	if s.animPhase > 1 {
		s.animPhase -= 1
	}

	// Handle mouse clicks.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.HandleClick(x, y)
	}

	// Handle text input and backspace for the first Wave compose field.
	if s.state == BootstrapStateFirstWavePrompt {
		for _, ch := range ebiten.AppendInputChars(nil) {
			if len(s.firstWaveText) < 2048 {
				s.firstWaveText += string(ch)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.firstWaveText) > 0 {
			s.firstWaveText = s.firstWaveText[:len(s.firstWaveText)-1]
		}
	}

	return nil
}

// Draw renders the bootstrap screen.
func (s *BootstrapScreen) Draw(screen *ebiten.Image) {
	s.width, s.height = screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dark background
	screen.Fill(color.RGBA{15, 15, 20, 255})

	switch s.state {
	case BootstrapStateConnecting:
		s.drawConnecting(screen)
	case BootstrapStatePulseMapIntro:
		s.drawPulseMapIntro(screen)
	case BootstrapStateFirstWavePrompt:
		s.drawFirstWavePrompt(screen)
	case BootstrapStateComplete:
		s.drawComplete(screen)
	}
}

// drawConnecting renders the peer discovery progress.
func (s *BootstrapScreen) drawConnecting(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 50

	s.drawConnectingTitle(screen, centerX)
	s.drawUserNode(screen, centerX, centerY)
	s.drawPeerNodes(screen, centerX, centerY)
	s.drawDiscoveryParticles(screen, centerX, centerY)
	s.drawConnectionProgress(screen, centerX, centerY)
	s.drawConnectingButton(screen, centerX)
}

// drawConnectingTitle renders the title text.
func (s *BootstrapScreen) drawConnectingTitle(screen *ebiten.Image, centerX float32) {
	titleY := float32(60)
	s.drawCenteredText(screen, "Joining the Network", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})
}

// drawUserNode renders the central user node with pulse effect.
func (s *BootstrapScreen) drawUserNode(screen *ebiten.Image, centerX, centerY float32) {
	userRadius := float32(20)
	userPulse := float32(0.3 + 0.2*math.Sin(s.animPhase*2*math.Pi))
	userColor := color.RGBA{230, 180, 100, 255}
	vector.DrawFilledCircle(screen, centerX, centerY, userRadius+userPulse*5, userColor, true)
}

// drawPeerNodes renders orbiting peer nodes and their connections.
func (s *BootstrapScreen) drawPeerNodes(screen *ebiten.Image, centerX, centerY float32) {
	for i := 0; i < s.peersFound; i++ {
		px, py := s.computePeerPosition(centerX, centerY, i)
		s.drawPeerConnection(screen, centerX, centerY, px, py)
		s.drawPeerCircle(screen, px, py)
	}
}

// computePeerPosition calculates the position of an orbiting peer.
func (s *BootstrapScreen) computePeerPosition(centerX, centerY float32, index int) (float32, float32) {
	baseAngle := float64(index) * 2 * math.Pi / float64(s.targetPeers)
	angle := baseAngle + s.animPhase*0.5*math.Pi
	radius := float32(100)
	px := centerX + radius*float32(math.Cos(angle))
	py := centerY + radius*float32(math.Sin(angle))
	return px, py
}

// drawPeerConnection renders the connection line to a peer.
func (s *BootstrapScreen) drawPeerConnection(screen *ebiten.Image, cx, cy, px, py float32) {
	lineColor := color.RGBA{80, 120, 180, 100}
	vector.StrokeLine(screen, cx, cy, px, py, 1, lineColor, true)
}

// drawPeerCircle renders a single peer node circle.
func (s *BootstrapScreen) drawPeerCircle(screen *ebiten.Image, px, py float32) {
	peerColor := color.RGBA{100, 150, 230, 200}
	vector.DrawFilledCircle(screen, px, py, 10, peerColor, true)
}

// drawDiscoveryParticles renders animated search particles.
func (s *BootstrapScreen) drawDiscoveryParticles(screen *ebiten.Image, centerX, centerY float32) {
	if s.discoveryDone {
		return
	}
	for i := 0; i < 12; i++ {
		s.drawSingleParticle(screen, centerX, centerY, i)
	}
}

// drawSingleParticle renders one discovery particle.
func (s *BootstrapScreen) drawSingleParticle(screen *ebiten.Image, centerX, centerY float32, index int) {
	angle := float64(index)*math.Pi/6 + s.animPhase*4*math.Pi
	progress := math.Mod(s.animPhase*3+float64(index)*0.1, 1)
	radius := float32(150 * (1 - progress))
	px := centerX + radius*float32(math.Cos(angle))
	py := centerY + radius*float32(math.Sin(angle))
	alpha := uint8(255 * (1 - progress))
	particleColor := color.RGBA{100, 100, 200, alpha}
	vector.DrawFilledCircle(screen, px, py, 3, particleColor, true)
}

// drawConnectionProgress renders the progress text.
func (s *BootstrapScreen) drawConnectionProgress(screen *ebiten.Image, centerX, centerY float32) {
	progressY := centerY + 150
	progressText := s.getProgressText()
	s.drawCenteredText(screen, progressText, centerX, progressY, 14, color.RGBA{150, 150, 160, 255})
}

// getProgressText returns the appropriate progress message.
func (s *BootstrapScreen) getProgressText() string {
	if s.peersFound == 0 {
		return "Searching for peers..."
	}
	if s.peersFound < s.targetPeers {
		return "Connected to " + itoa(s.peersFound) + " peers..."
	}
	return "Connected to " + itoa(s.peersFound) + " peers ✓"
}

// drawConnectingButton renders the continue button when discovery is complete.
func (s *BootstrapScreen) drawConnectingButton(screen *ebiten.Image, centerX float32) {
	if s.discoveryDone {
		buttonY := float32(s.height) - 100
		s.drawButton(screen, "Continue", centerX, buttonY, 0)
	}
}

// drawPulseMapIntro renders the Pulse Map introduction tour.
func (s *BootstrapScreen) drawPulseMapIntro(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height) / 2

	// Title
	titleY := float32(50)
	s.drawCenteredText(screen, "Welcome to the Pulse Map", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	// Draw simplified Pulse Map preview
	s.drawPulseMapPreview(screen, centerX, centerY-20)

	// Tutorial overlays
	tutorialY := centerY + 140
	tutorial := getTutorialStep(s.tutorialStep)
	s.drawCenteredText(screen, tutorial, centerX, tutorialY, 14, color.RGBA{180, 180, 190, 255})

	// Navigation
	if s.tutorialStep > 0 {
		s.drawButton(screen, "← Back", centerX-100, float32(s.height)-100, 1)
	}
	if s.tutorialStep < 4 {
		s.drawButton(screen, "Next →", centerX+100, float32(s.height)-100, 2)
	} else {
		s.drawButton(screen, "Continue", centerX, float32(s.height)-100, 0)
	}
}

// drawPulseMapPreview renders a simplified Pulse Map.
func (s *BootstrapScreen) drawPulseMapPreview(screen *ebiten.Image, centerX, centerY float32) {
	s.drawPreviewBackground(screen, centerX, centerY)
	s.drawPreviewUserNode(screen, centerX, centerY)
	peerPositions := s.getPreviewPeerPositions(centerX, centerY)
	s.drawPreviewPeers(screen, centerX, centerY, peerPositions)
	s.drawPreviewHighlight(screen, centerX, centerY, peerPositions)
}

// drawPreviewBackground renders the preview area background and border.
func (s *BootstrapScreen) drawPreviewBackground(screen *ebiten.Image, centerX, centerY float32) {
	previewWidth, previewHeight := float32(400), float32(200)
	previewX := centerX - previewWidth/2
	previewY := centerY - previewHeight/2
	vector.DrawFilledRect(screen, previewX, previewY, previewWidth, previewHeight, color.RGBA{20, 25, 35, 255}, true)
	vector.StrokeRect(screen, previewX, previewY, previewWidth, previewHeight, 1, color.RGBA{50, 60, 80, 255}, true)
}

// drawPreviewUserNode renders the central user node with pulse effect.
func (s *BootstrapScreen) drawPreviewUserNode(screen *ebiten.Image, centerX, centerY float32) {
	userPulse := float32(1 + 0.1*math.Sin(s.animPhase*2*math.Pi))
	vector.DrawFilledCircle(screen, centerX, centerY, 15*userPulse, color.RGBA{230, 180, 100, 255}, true)
}

// getPreviewPeerPositions returns static peer positions for the preview.
func (s *BootstrapScreen) getPreviewPeerPositions(centerX, centerY float32) []struct{ x, y float32 } {
	return []struct{ x, y float32 }{
		{centerX - 60, centerY - 40},
		{centerX + 70, centerY - 30},
		{centerX - 50, centerY + 50},
		{centerX + 65, centerY + 45},
		{centerX + 120, centerY - 10},
	}
}

// drawPreviewPeers renders peer nodes and their connections.
func (s *BootstrapScreen) drawPreviewPeers(screen *ebiten.Image, centerX, centerY float32, positions []struct{ x, y float32 }) {
	peerColor := color.RGBA{100, 150, 230, 200}
	lineColor := color.RGBA{60, 90, 140, 100}
	for _, pos := range positions {
		vector.StrokeLine(screen, centerX, centerY, pos.x, pos.y, 1, lineColor, true)
		vector.DrawFilledCircle(screen, pos.x, pos.y, 8, peerColor, true)
	}
}

// drawPreviewHighlight renders tutorial step highlights.
func (s *BootstrapScreen) drawPreviewHighlight(screen *ebiten.Image, centerX, centerY float32, positions []struct{ x, y float32 }) {
	switch s.tutorialStep {
	case 0:
		vector.StrokeCircle(screen, centerX, centerY, 25, 2, color.RGBA{255, 220, 100, 100}, true)
	case 1:
		for _, pos := range positions {
			vector.StrokeLine(screen, centerX, centerY, pos.x, pos.y, 3, color.RGBA{100, 200, 255, 100}, true)
		}
	case 2:
		for _, pos := range positions {
			vector.StrokeCircle(screen, pos.x, pos.y, 15, 2, color.RGBA{100, 200, 255, 100}, true)
		}
	}
}

// drawFirstWavePrompt renders the first Wave composition screen.
func (s *BootstrapScreen) drawFirstWavePrompt(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	s.drawCenteredText(screen, "Send Your First Wave", centerX, 60, 22, color.RGBA{220, 220, 225, 255})

	promptY := float32(120)
	s.drawCenteredText(screen, "A Wave is an ephemeral message that ripples through the network.", centerX, promptY, 14, color.RGBA{150, 150, 160, 255})
	s.drawCenteredText(screen, "It's not permanent — it will fade after its TTL expires.", centerX, promptY+20, 14, color.RGBA{150, 150, 160, 255})

	s.drawWaveInputArea(screen, centerX, 200)
	s.drawWaveSuggestions(screen, centerX, 320)
	s.drawWaveButtons(screen, centerX, float32(s.height)-100)
}

// drawWaveInputArea renders the text input box for the first Wave.
func (s *BootstrapScreen) drawWaveInputArea(screen *ebiten.Image, centerX, inputY float32) {
	inputWidth, inputHeight := float32(400), float32(80)
	inputX := centerX - inputWidth/2

	vector.DrawFilledRect(screen, inputX, inputY, inputWidth, inputHeight, color.RGBA{25, 30, 45, 255}, true)
	vector.StrokeRect(screen, inputX, inputY, inputWidth, inputHeight, 1.5, color.RGBA{70, 80, 120, 255}, true)

	textY := inputY + inputHeight/2 + 5
	if s.firstWaveText != "" {
		s.drawCenteredText(screen, s.firstWaveText, centerX, textY, 14, color.RGBA{220, 220, 230, 255})
	} else {
		s.drawCenteredText(screen, "Type your first Wave...", centerX, textY, 14, color.RGBA{100, 100, 110, 255})
	}
}

// drawWaveSuggestions renders the suggested Wave prompts.
func (s *BootstrapScreen) drawWaveSuggestions(screen *ebiten.Image, centerX, suggestY float32) {
	s.drawCenteredText(screen, "Suggestions:", centerX, suggestY, 12, color.RGBA{130, 130, 140, 255})

	suggestions := []string{"Hello, network!", "Just joined MURMUR", "Testing the waves..."}
	for i, suggestion := range suggestions {
		suggY := suggestY + float32(20+i*22)
		suggWidth, suggHeight := float32(150), float32(20)
		suggX := centerX - suggWidth/2
		vector.DrawFilledRect(screen, suggX, suggY-suggHeight/2, suggWidth, suggHeight, color.RGBA{35, 40, 55, 255}, true)
		s.drawCenteredText(screen, suggestion, centerX, suggY+4, 11, color.RGBA{160, 160, 170, 255})
	}
}

// drawWaveButtons renders the Send and Skip buttons.
func (s *BootstrapScreen) drawWaveButtons(screen *ebiten.Image, centerX, buttonY float32) {
	if s.firstWaveText != "" {
		s.drawButton(screen, "Send Wave", centerX-80, buttonY, 0)
	}
	s.drawButton(screen, "Skip", centerX+80, buttonY, 1)
}

// drawComplete renders the bootstrap completion screen.
func (s *BootstrapScreen) drawComplete(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 40

	// Title
	titleY := float32(60)
	s.drawCenteredText(screen, "You're Connected!", centerX, titleY, 22, color.RGBA{220, 220, 225, 255})

	// Success animation using shared helper
	DrawSuccessAnimation(screen, centerX, centerY, s.animPhase, DefaultSuccessStyle())

	// Stats
	statsY := centerY + 80
	s.drawCenteredText(screen, "Connected to "+itoa(s.peersFound)+" peers", centerX, statsY, 14, color.RGBA{150, 150, 160, 255})

	if s.waveSent {
		s.drawCenteredText(screen, "First Wave sent ✓", centerX, statsY+25, 14, color.RGBA{150, 200, 160, 255})
	}

	// Enter button
	buttonY := float32(s.height) - 100
	s.drawButton(screen, "Enter MURMUR", centerX, buttonY, 0)
}

func getTutorialStep(step int) string {
	tutorials := []string{
		"This glowing node is you — your place in the network.",
		"Lines show your connections to peers.",
		"Other nodes are participants in the network.",
		"Pan and zoom to explore the Pulse Map.",
		"Waves ripple outward through visible connections.",
	}
	if step < 0 || step >= len(tutorials) {
		return ""
	}
	return tutorials[step]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// HandleClick processes clicks on the bootstrap screen.
func (s *BootstrapScreen) HandleClick(x, y int) {
	switch s.state {
	case BootstrapStateConnecting:
		s.handleConnectingClick(x, y)
	case BootstrapStatePulseMapIntro:
		s.handlePulseMapIntroClick(x, y)
	case BootstrapStateFirstWavePrompt:
		s.handleFirstWaveClick(x, y)
	case BootstrapStateComplete:
		s.handleCompleteClick(x, y)
	}
}

// handleConnectingClick processes clicks in the connecting state.
func (s *BootstrapScreen) handleConnectingClick(x, y int) {
	if !s.discoveryDone {
		return
	}
	buttonY := s.height - 100
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.state = BootstrapStatePulseMapIntro
		s.tutorialStep = 0
	}
}

// handlePulseMapIntroClick processes clicks in the Pulse Map intro tutorial.
func (s *BootstrapScreen) handlePulseMapIntroClick(x, y int) {
	centerX := s.width / 2
	buttonY := s.height - 100

	if s.handleBackButtonClick(x, y, centerX, buttonY) {
		return
	}
	s.handleNextButtonClick(x, y, centerX, buttonY)
}

// handleBackButtonClick handles the back button in the tutorial.
func (s *BootstrapScreen) handleBackButtonClick(x, y, centerX, buttonY int) bool {
	if s.tutorialStep > 0 && s.isClickOnButton(x, y, centerX-100, buttonY) {
		s.tutorialStep--
		return true
	}
	return false
}

// handleNextButtonClick handles the next/continue button in the tutorial.
func (s *BootstrapScreen) handleNextButtonClick(x, y, centerX, buttonY int) {
	if s.tutorialStep < 4 {
		if s.isClickOnButton(x, y, centerX+100, buttonY) {
			s.tutorialStep++
		}
	} else if s.isClickOnButton(x, y, centerX, buttonY) {
		s.state = BootstrapStateFirstWavePrompt
		if s.callbacks.OnPulseMapReady != nil {
			s.callbacks.OnPulseMapReady()
		}
	}
}

// handleFirstWaveClick processes clicks in the first wave prompt state.
func (s *BootstrapScreen) handleFirstWaveClick(x, y int) {
	centerX := s.width / 2
	buttonY := s.height - 100

	s.handleSendWaveClick(x, y, centerX, buttonY)
	s.handleSkipWaveClick(x, y, centerX, buttonY)
	s.handleSuggestionClick(x, y, centerX)
}

// handleSendWaveClick handles the send wave button click.
func (s *BootstrapScreen) handleSendWaveClick(x, y, centerX, buttonY int) {
	if s.firstWaveText == "" {
		return
	}
	if s.isClickOnButton(x, y, centerX-80, buttonY) {
		s.waveSent = true
		if s.callbacks.OnFirstWaveSent != nil {
			s.callbacks.OnFirstWaveSent(s.firstWaveText)
		}
		s.state = BootstrapStateComplete
	}
}

// handleSkipWaveClick handles the skip button click.
func (s *BootstrapScreen) handleSkipWaveClick(x, y, centerX, buttonY int) {
	if s.isClickOnButton(x, y, centerX+80, buttonY) {
		s.state = BootstrapStateComplete
	}
}

// handleSuggestionClick checks if a suggestion was clicked.
func (s *BootstrapScreen) handleSuggestionClick(x, y, centerX int) {
	suggestions := []string{
		"Hello, network!",
		"Just joined MURMUR",
		"Testing the waves...",
	}
	suggestY := 200 + 80 + 40 // inputY + inputHeight + offset

	for i, suggestion := range suggestions {
		if s.isSuggestionClicked(x, y, centerX, suggestY, i) {
			s.firstWaveText = suggestion
			return
		}
	}
}

// isSuggestionClicked checks if a specific suggestion was clicked.
func (s *BootstrapScreen) isSuggestionClicked(x, y, centerX, baseY, index int) bool {
	suggY := baseY + 20 + index*22
	suggWidth, suggHeight := 150, 20
	suggX := centerX - suggWidth/2
	return x >= suggX && x <= suggX+suggWidth &&
		y >= suggY-suggHeight/2 && y <= suggY+suggHeight/2
}

// handleCompleteClick processes clicks in the complete state.
func (s *BootstrapScreen) handleCompleteClick(x, y int) {
	buttonY := s.height - 100
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.completePhase()
	}
}

// HandleTextInput handles text input for the first Wave.
func (s *BootstrapScreen) HandleTextInput(text string) {
	if s.state != BootstrapStateFirstWavePrompt {
		return
	}
	s.firstWaveText = text
}

// SimulatePeerFound simulates finding a peer.
func (s *BootstrapScreen) SimulatePeerFound() {
	s.peersFound++
	if s.callbacks.OnPeerFound != nil {
		s.callbacks.OnPeerFound(s.peersFound)
	}
	if s.peersFound >= s.targetPeers {
		s.discoveryDone = true
	}
}

// SimulateDiscoveryComplete simulates completing peer discovery.
func (s *BootstrapScreen) SimulateDiscoveryComplete(peerCount int) {
	s.peersFound = peerCount
	s.discoveryDone = true
}

func (s *BootstrapScreen) completePhase() {
	// Bootstrap screen handles NetworkBootstrap (3) -> GuidedExploration (4) -> FirstWave (5) -> Complete (6)
	s.controller.CompleteCurrentPhase() // NetworkBootstrap -> GuidedExploration
	s.controller.CompleteCurrentPhase() // GuidedExploration -> FirstWave
	s.controller.CompleteCurrentPhase() // FirstWave -> Complete

	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseNetworkBootstrap)
	}
}

func (s *BootstrapScreen) isClickOnButton(clickX, clickY, buttonCenterX, buttonCenterY int) bool {
	width := 200
	height := 44
	bx := buttonCenterX - width/2
	by := buttonCenterY - height/2
	return clickX >= bx && clickX <= bx+width && clickY >= by && clickY <= by+height
}

func (s *BootstrapScreen) drawButton(screen *ebiten.Image, label string, x, y float32, index int) {
	style := DefaultButtonStyle()
	style.Width = 140
	DrawButton(screen, label, x, y, style)
}

// drawCenteredText delegates to the shared DrawCenteredText helper.
func (s *BootstrapScreen) drawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	DrawCenteredText(screen, str, x, y, size, clr)
}

// State returns the current screen state.
func (s *BootstrapScreen) BootstrapState() BootstrapScreenState {
	return s.state
}

// PeersFound returns the number of discovered peers.
func (s *BootstrapScreen) PeersFound() int {
	return s.peersFound
}

// IsDiscoveryDone returns whether peer discovery is complete.
func (s *BootstrapScreen) IsDiscoveryDone() bool {
	return s.discoveryDone
}

// WasSent returns whether the first Wave was sent.
func (s *BootstrapScreen) WasSent() bool {
	return s.waveSent
}
