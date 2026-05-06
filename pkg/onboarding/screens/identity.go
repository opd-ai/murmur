// Package screens provides Ebitengine-based UI screens for onboarding.
// Per ONBOARDING.md, this implements the six-phase onboarding flow with
// animated visualizations and interactive elements.
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
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/sigils"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// ScreenState tracks the current screen within a phase.
type ScreenState int

const (
	StateWelcome ScreenState = iota
	StatePhilosophy
	StateKeypairGen
	StateDisplayName
	StateBackupPrompt
	StateBackupMnemonic
	StateBackupFile
	StateBackupComplete
)

// Screen is the main onboarding screen manager.
type Screen struct {
	controller *flow.Controller
	state      ScreenState
	startTime  time.Time
	phaseStart time.Time
	animPhase  float64 // Animation phase (0-1)

	// Welcome screen
	welcomePulse float64

	// Philosophy screen
	philosophyIndex int
	philosophyStart time.Time
	philosophyTexts []string

	// Identity creation
	keypair      *keys.KeyPair
	sigil        *ebiten.Image
	displayName  string
	inputFocused bool
	cursorBlink  float64

	// Backup
	mnemonic   string
	backupDone bool

	// UI state
	width, height int
	buttonHover   int // -1 for none, 0+ for button index
	callbacks     ScreenCallbacks
}

// ScreenCallbacks provides hooks for screen events.
type ScreenCallbacks struct {
	OnKeypairGenerated func(*keys.KeyPair)
	OnDisplayNameSet   func(string)
	OnBackupComplete   func(method string)
	OnPhaseComplete    func(flow.Phase)
	OnSkipBackup       func()
}

// NewScreen creates a new onboarding screen.
func NewScreen(controller *flow.Controller, callbacks ScreenCallbacks) *Screen {
	return &Screen{
		controller:  controller,
		state:       StateWelcome,
		startTime:   time.Now(),
		phaseStart:  time.Now(),
		buttonHover: -1,
		callbacks:   callbacks,
		philosophyTexts: []string{
			"No servers. The network lives on the devices of its participants.",
			"No accounts. Your identity is a cryptographic key that you own.",
			"No algorithms. You see what the network shows, shaped by topology, not by corporate interest.",
		},
	}
}

// Update handles input and advances animations.
func (s *Screen) Update() error {
	s.updateAnimations()
	s.updatePhilosophyTiming()
	s.handleUserInput()
	return nil
}

// updateAnimations updates all animation states.
func (s *Screen) updateAnimations() {
	dt := 1.0 / 60.0
	s.animPhase += dt * 0.5
	if s.animPhase > 1 {
		s.animPhase -= 1
	}

	s.welcomePulse = 0.5 + 0.5*math.Sin(s.animPhase*2*math.Pi)

	s.cursorBlink += dt * 2
	if s.cursorBlink > 1 {
		s.cursorBlink -= 1
	}
}

// updatePhilosophyTiming advances philosophy statement display.
func (s *Screen) updatePhilosophyTiming() {
	if s.state != StatePhilosophy {
		return
	}

	elapsed := time.Since(s.philosophyStart)
	index := int(elapsed.Seconds() / 3)
	if index != s.philosophyIndex && index < len(s.philosophyTexts) {
		s.philosophyIndex = index
	}
}

// handleUserInput processes mouse clicks and keyboard input.
func (s *Screen) handleUserInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.HandleClick(x, y)
	}

	for _, ch := range ebiten.AppendInputChars(nil) {
		s.HandleKeyInput(ch)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		s.HandleBackspace()
	}
}

// Draw renders the current screen.
func (s *Screen) Draw(screen *ebiten.Image) {
	s.width, s.height = screen.Bounds().Dx(), screen.Bounds().Dy()

	// Dark background
	screen.Fill(color.RGBA{15, 15, 20, 255})

	switch s.state {
	case StateWelcome:
		s.drawWelcome(screen)
	case StatePhilosophy:
		s.drawPhilosophy(screen)
	case StateKeypairGen:
		s.drawKeypairGen(screen)
	case StateDisplayName:
		s.drawDisplayName(screen)
	case StateBackupPrompt:
		s.drawBackupPrompt(screen)
	case StateBackupMnemonic:
		s.drawBackupMnemonic(screen)
	case StateBackupFile:
		s.drawBackupFile(screen)
	case StateBackupComplete:
		s.drawBackupComplete(screen)
	}
}

// drawWelcome renders Phase 1: Welcome screen.
// Per ONBOARDING.md: softly glowing dot, "MURMUR" title, "A network that belongs to no one."
func (s *Screen) drawWelcome(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 50

	// Pulsing dot (represents user's future node)
	pulseRadius := float32(15 + 5*s.welcomePulse)
	pulseAlpha := uint8(100 + 80*s.welcomePulse)

	// Outer glow
	for r := pulseRadius + 20; r > pulseRadius; r -= 5 {
		alpha := uint8(float64(pulseAlpha) * float64(pulseRadius+20-r) / 20)
		glowColor := color.RGBA{180, 200, 255, alpha}
		vector.DrawFilledCircle(screen, centerX, centerY, r, glowColor, true)
	}

	// Core dot
	coreColor := color.RGBA{220, 235, 255, 255}
	vector.DrawFilledCircle(screen, centerX, centerY, pulseRadius, coreColor, true)

	// Title: "MURMUR"
	titleY := centerY + 80
	s.drawCenteredText(screen, "MURMUR", centerX, titleY, 32, color.RGBA{220, 220, 225, 255})

	// Tagline
	taglineY := titleY + 40
	s.drawCenteredText(screen, "A network that belongs to no one.", centerX, taglineY, 16, color.RGBA{140, 140, 150, 255})

	// Begin button
	buttonY := float32(s.height) - 120
	s.drawButton(screen, "Begin", centerX, buttonY, 0)
}

// drawPhilosophy renders the philosophy statements.
func (s *Screen) drawPhilosophy(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height) / 2

	if s.philosophyIndex < len(s.philosophyTexts) {
		s.drawPhilosophyText(screen, centerX, centerY)
		s.drawProgressDots(screen, centerX, centerY+60)
	}

	s.drawButton(screen, "Continue", centerX, float32(s.height)-120, 0)
	s.drawCenteredText(screen, "Skip", centerX, float32(s.height)-60, 12, color.RGBA{100, 100, 110, 255})
}

// drawPhilosophyText renders the current philosophy statement with fade animation.
func (s *Screen) drawPhilosophyText(screen *ebiten.Image, centerX, centerY float32) {
	text := s.philosophyTexts[s.philosophyIndex]
	alpha := s.calculateFadeAlpha()
	textColor := color.RGBA{200, 200, 210, uint8(255 * alpha)}
	s.drawCenteredText(screen, text, centerX, centerY, 18, textColor)
}

// calculateFadeAlpha computes the alpha value for fade in/out animation.
func (s *Screen) calculateFadeAlpha() float64 {
	elapsed := time.Since(s.philosophyStart).Seconds()
	withinPhase := elapsed - float64(s.philosophyIndex)*3
	if withinPhase < 0.5 {
		return withinPhase / 0.5 // Fade in
	} else if withinPhase > 2.5 {
		return (3 - withinPhase) / 0.5 // Fade out
	}
	return 1
}

// drawProgressDots renders the progress indicator dots.
func (s *Screen) drawProgressDots(screen *ebiten.Image, centerX, dotY float32) {
	for i := 0; i < len(s.philosophyTexts); i++ {
		dotX := centerX + float32((i-1)*20)
		dotColor := color.RGBA{80, 80, 90, 255}
		if i == s.philosophyIndex {
			dotColor = color.RGBA{180, 180, 200, 255}
		}
		vector.DrawFilledCircle(screen, dotX, dotY, 4, dotColor, true)
	}
}

// drawKeypairGen renders the keypair generation animation.
// Per ONBOARDING.md: particles coalescing into a solid point of light.
func (s *Screen) drawKeypairGen(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height)/2 - 50

	titleY := float32(80)
	s.drawCenteredText(screen, "Creating Your Identity", centerX, titleY, 24, color.RGBA{220, 220, 225, 255})

	if s.keypair == nil {
		s.drawKeypairAnimation(screen, centerX, centerY)
	} else {
		s.drawKeypairComplete(screen, centerX, centerY)
	}
}

// drawKeypairAnimation renders the particle convergence animation.
func (s *Screen) drawKeypairAnimation(screen *ebiten.Image, centerX, centerY float32) {
	progress := math.Min(1, time.Since(s.phaseStart).Seconds()/2.5)
	s.drawConvergingParticles(screen, centerX, centerY, progress)
	s.drawGrowingCore(screen, centerX, centerY, progress)

	if progress >= 1 {
		go s.generateKeypair()
	}
}

// drawConvergingParticles renders particles spiraling toward center.
func (s *Screen) drawConvergingParticles(screen *ebiten.Image, centerX, centerY float32, progress float64) {
	numParticles := 24
	for i := 0; i < numParticles; i++ {
		angle := float64(i)*2*math.Pi/float64(numParticles) + s.animPhase*4*math.Pi
		radius := float32(100 * (1 - progress*0.9))
		px := centerX + radius*float32(math.Cos(angle))
		py := centerY + radius*float32(math.Sin(angle))
		pSize := float32(3 + 2*(1-progress))
		pAlpha := uint8(100 + 155*progress)
		vector.DrawFilledCircle(screen, px, py, pSize, color.RGBA{180, 200, 255, pAlpha}, true)
	}
}

// drawGrowingCore renders the central core that grows as particles converge.
func (s *Screen) drawGrowingCore(screen *ebiten.Image, centerX, centerY float32, progress float64) {
	coreRadius := float32(5 + 15*progress)
	coreColor := color.RGBA{220, 235, 255, uint8(150 + 105*progress)}
	vector.DrawFilledCircle(screen, centerX, centerY, coreRadius, coreColor, true)
}

// drawKeypairComplete renders the completed identity with sigil and fingerprint.
func (s *Screen) drawKeypairComplete(screen *ebiten.Image, centerX, centerY float32) {
	if s.sigil != nil {
		sigilOpts := &ebiten.DrawImageOptions{}
		sigilX := centerX - float32(s.sigil.Bounds().Dx())/2
		sigilY := centerY - float32(s.sigil.Bounds().Dy())/2
		sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
		screen.DrawImage(s.sigil, sigilOpts)
	}

	fingerprint := formatFingerprint(s.keypair.PublicKey[:8])
	fpY := centerY + 80
	s.drawCenteredText(screen, fingerprint, centerX, fpY, 14, color.RGBA{140, 140, 150, 255})

	buttonY := float32(s.height) - 120
	s.drawButton(screen, "Continue", centerX, buttonY, 0)
}

// drawDisplayName renders the display name input screen.
func (s *Screen) drawDisplayName(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	titleY := float32(80)
	s.drawCenteredText(screen, "Choose a name others will see", centerX, titleY, 20, color.RGBA{220, 220, 225, 255})

	s.drawSigilPreview(screen, centerX, float32(140))
	s.drawNameInputField(screen, centerX)

	buttonY := float32(s.height) - 120
	s.drawButton(screen, "Continue", centerX, buttonY, 0)
}

// drawSigilPreview renders the sigil preview image.
func (s *Screen) drawSigilPreview(screen *ebiten.Image, centerX, sigilY float32) {
	if s.sigil == nil {
		return
	}
	sigilOpts := &ebiten.DrawImageOptions{}
	sigilX := centerX - float32(s.sigil.Bounds().Dx())/2
	sigilOpts.GeoM.Translate(float64(sigilX), float64(sigilY))
	screen.DrawImage(s.sigil, sigilOpts)
}

// drawNameInputField renders the input field with placeholder and cursor.
func (s *Screen) drawNameInputField(screen *ebiten.Image, centerX float32) {
	inputY := float32(s.height)/2 + 20
	inputWidth, inputHeight := float32(300), float32(40)
	inputX := centerX - inputWidth/2

	vector.DrawFilledRect(screen, inputX, inputY, inputWidth, inputHeight, color.RGBA{30, 30, 40, 255}, true)

	borderColor := color.RGBA{80, 80, 100, 255}
	if s.inputFocused {
		borderColor = color.RGBA{120, 140, 200, 255}
	}
	vector.StrokeRect(screen, inputX, inputY, inputWidth, inputHeight, 1.5, borderColor, true)

	textY := inputY + inputHeight/2 + 6
	s.drawNameInputText(screen, centerX, textY)

	noteY := inputY + inputHeight + 30
	s.drawCenteredText(screen, "You can change this anytime. Your sigil and fingerprint are permanent.", centerX, noteY, 12, color.RGBA{100, 100, 110, 255})
}

// drawNameInputText renders the input text or placeholder.
func (s *Screen) drawNameInputText(screen *ebiten.Image, centerX, textY float32) {
	if s.displayName == "" && !s.inputFocused {
		s.drawCenteredText(screen, "Optional display name", centerX, textY, 14, color.RGBA{80, 80, 90, 255})
		return
	}
	displayText := s.displayName
	if s.inputFocused && s.cursorBlink < 0.5 {
		displayText += "|"
	}
	s.drawCenteredText(screen, displayText, centerX, textY, 14, color.RGBA{200, 200, 210, 255})
}

// drawBackupPrompt renders the backup options screen.
func (s *Screen) drawBackupPrompt(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	// Title
	titleY := float32(80)
	s.drawCenteredText(screen, "Your Private Key", centerX, titleY, 24, color.RGBA{220, 220, 225, 255})

	// Warning text
	warningY := float32(140)
	warnings := []string{
		"Your private key is your sole proof of identity.",
		"MURMUR does not store your key on any server.",
		"If you lose this key, your identity is lost permanently.",
		"No one can recover a lost key.",
	}
	for i, w := range warnings {
		y := warningY + float32(i*25)
		s.drawCenteredText(screen, w, centerX, y, 14, color.RGBA{180, 160, 140, 255})
	}

	// Backup options
	optionY := float32(s.height) / 2
	s.drawButton(screen, "Save Key File", centerX, optionY, 0)
	s.drawButton(screen, "Write Down Recovery Phrase", centerX, optionY+60, 1)

	// Skip option (de-emphasized)
	skipY := float32(s.height) - 80
	s.drawCenteredText(screen, "Skip for now", centerX, skipY, 12, color.RGBA{80, 80, 90, 255})
}

// drawBackupMnemonic renders the mnemonic display screen.
func (s *Screen) drawBackupMnemonic(screen *ebiten.Image) {
	centerX := float32(s.width) / 2

	// Title
	titleY := float32(60)
	s.drawCenteredText(screen, "Write down these words", centerX, titleY, 20, color.RGBA{220, 220, 225, 255})

	// Instructions
	instrY := float32(100)
	s.drawCenteredText(screen, "Store them securely. Anyone with these words can access your identity.", centerX, instrY, 12, color.RGBA{180, 160, 140, 255})

	// Mnemonic words (placeholder - actual implementation would show real words)
	if s.mnemonic == "" {
		s.mnemonic = "example words for display purposes only" // Would be real mnemonic
	}

	mnemonicY := float32(150)
	s.drawCenteredText(screen, s.mnemonic, centerX, mnemonicY, 14, color.RGBA{200, 200, 210, 255})

	// Done button
	buttonY := float32(s.height) - 120
	s.drawButton(screen, "I've written them down", centerX, buttonY, 0)
}

// drawBackupFile renders the file backup screen.
func (s *Screen) drawBackupFile(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height) / 2

	s.drawCenteredText(screen, "Choose a passphrase for your key file", centerX, centerY-50, 18, color.RGBA{220, 220, 225, 255})

	// In a full implementation, this would show a passphrase input and file save dialog
	s.drawCenteredText(screen, "(Passphrase input would appear here)", centerX, centerY, 14, color.RGBA{140, 140, 150, 255})

	buttonY := float32(s.height) - 120
	s.drawButton(screen, "Save File", centerX, buttonY, 0)
}

// drawBackupComplete renders the backup confirmation screen.
func (s *Screen) drawBackupComplete(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	centerY := float32(s.height) / 2

	// Checkmark
	checkColor := color.RGBA{100, 200, 100, 255}
	vector.DrawFilledCircle(screen, centerX, centerY-50, 30, checkColor, true)

	// Text
	s.drawCenteredText(screen, "Backup Complete", centerX, centerY+20, 24, color.RGBA{220, 220, 225, 255})
	s.drawCenteredText(screen, "Your identity is safely backed up.", centerX, centerY+60, 14, color.RGBA{140, 140, 150, 255})

	// Continue button
	buttonY := float32(s.height) - 120
	s.drawButton(screen, "Continue", centerX, buttonY, 0)
}

// drawButton draws a styled button.
func (s *Screen) drawButton(screen *ebiten.Image, label string, x, y float32, index int) {
	width := float32(200)
	height := float32(44)
	bx := x - width/2
	by := y - height/2

	// Button background
	bgColor := color.RGBA{60, 70, 100, 255}
	if s.buttonHover == index {
		bgColor = color.RGBA{80, 90, 130, 255}
	}

	// Rounded corners approximation (filled rect for now)
	vector.DrawFilledRect(screen, bx, by, width, height, bgColor, true)

	// Border
	borderColor := color.RGBA{100, 120, 180, 255}
	vector.StrokeRect(screen, bx, by, width, height, 1.5, borderColor, true)

	// Label
	s.drawCenteredText(screen, label, x, y+6, 16, color.RGBA{220, 220, 230, 255})
}

// drawCenteredText delegates to the shared DrawCenteredText helper.
func (s *Screen) drawCenteredText(screen *ebiten.Image, str string, x, y float32, size float64, clr color.Color) {
	DrawCenteredText(screen, str, x, y, size, clr)
}

// generateKeypair creates a new keypair and sigil.
func (s *Screen) generateKeypair() {
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		return // Handle error in production
	}

	s.keypair = kp

	// Generate sigil - sigils.Generate takes []byte and returns *Sigil with Image field
	sigilData := sigils.Generate(kp.PublicKey)
	s.sigil = ebiten.NewImageFromImage(sigilData.Image)

	if s.callbacks.OnKeypairGenerated != nil {
		s.callbacks.OnKeypairGenerated(kp)
	}
}

// formatFingerprint formats a public key fingerprint for display.
func formatFingerprint(data []byte) string {
	if len(data) < 8 {
		return ""
	}
	return formatHex(data[:4]) + ":" + formatHex(data[4:8]) + ":..."
}

func formatHex(data []byte) string {
	const hex = "0123456789ABCDEF"
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = hex[b>>4]
		result[i*2+1] = hex[b&0x0F]
	}
	return string(result)
}

// HandleClick processes a click at the given position.
func (s *Screen) HandleClick(x, y int) {
	switch s.state {
	case StateWelcome:
		s.handleWelcomeClick(x, y)
	case StatePhilosophy:
		s.handlePhilosophyClick(x, y)
	case StateKeypairGen:
		s.handleKeypairGenClick(x, y)
	case StateDisplayName:
		s.handleDisplayNameClick(x, y)
	case StateBackupPrompt:
		s.handleBackupPromptClick(x, y)
	case StateBackupMnemonic:
		s.handleBackupMnemonicClick(x, y)
	case StateBackupFile:
		s.handleBackupFileClick(x, y)
	case StateBackupComplete:
		s.handleBackupCompleteClick(x, y)
	}
}

// handleWelcomeClick processes clicks on the welcome screen.
func (s *Screen) handleWelcomeClick(x, y int) {
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.transitionToPhilosophy()
	}
}

// handlePhilosophyClick processes clicks on the philosophy screen.
func (s *Screen) handlePhilosophyClick(x, y int) {
	centerX := s.width / 2
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, centerX, buttonY) {
		s.transitionToKeypairGen()
		return
	}
	skipY := s.height - 60
	if abs(y-skipY) < 20 && abs(x-centerX) < 50 {
		s.transitionToKeypairGen()
	}
}

// handleKeypairGenClick processes clicks on the keypair generation screen.
func (s *Screen) handleKeypairGenClick(x, y int) {
	if s.keypair == nil {
		return
	}
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.transitionToDisplayName()
	}
}

// handleDisplayNameClick processes clicks on the display name screen.
func (s *Screen) handleDisplayNameClick(x, y int) {
	centerX := s.width / 2
	s.updateInputFocus(x, y, centerX)

	buttonY := s.height - 120
	if s.isClickOnButton(x, y, centerX, buttonY) {
		if s.callbacks.OnDisplayNameSet != nil {
			s.callbacks.OnDisplayNameSet(s.displayName)
		}
		s.transitionToBackupPrompt()
	}
}

// updateInputFocus updates input field focus state based on click position.
func (s *Screen) updateInputFocus(x, y, centerX int) {
	inputY := s.height/2 + 20
	inputWidth, inputHeight := 300, 40
	inputX := centerX - inputWidth/2
	s.inputFocused = x >= inputX && x <= inputX+inputWidth &&
		y >= inputY && y <= inputY+inputHeight
}

// handleBackupPromptClick processes clicks on the backup prompt screen.
func (s *Screen) handleBackupPromptClick(x, y int) {
	centerX := s.width / 2
	optionY := s.height / 2

	if s.isClickOnButton(x, y, centerX, optionY) {
		s.transitionToBackupFile()
		return
	}
	if s.isClickOnButton(x, y, centerX, optionY+60) {
		s.transitionToBackupMnemonic()
		return
	}
	s.handleSkipBackupClick(x, y, centerX)
}

// handleSkipBackupClick handles the skip backup link click.
func (s *Screen) handleSkipBackupClick(x, y, centerX int) {
	skipY := s.height - 80
	if abs(y-skipY) < 20 && abs(x-centerX) < 80 {
		if s.callbacks.OnSkipBackup != nil {
			s.callbacks.OnSkipBackup()
		}
		s.completePhase1And2()
	}
}

// handleBackupMnemonicClick processes clicks on the mnemonic backup screen.
func (s *Screen) handleBackupMnemonicClick(x, y int) {
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.completeBackup("mnemonic")
	}
}

// handleBackupFileClick processes clicks on the file backup screen.
func (s *Screen) handleBackupFileClick(x, y int) {
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.completeBackup("file")
	}
}

// completeBackup marks backup as done and transitions to completion.
func (s *Screen) completeBackup(method string) {
	s.backupDone = true
	if s.callbacks.OnBackupComplete != nil {
		s.callbacks.OnBackupComplete(method)
	}
	s.transitionToBackupComplete()
}

// handleBackupCompleteClick processes clicks on the backup complete screen.
func (s *Screen) handleBackupCompleteClick(x, y int) {
	buttonY := s.height - 120
	if s.isClickOnButton(x, y, s.width/2, buttonY) {
		s.completePhase1And2()
	}
}

// HandleKeyInput processes keyboard input for text entry.
func (s *Screen) HandleKeyInput(char rune) {
	if s.state == StateDisplayName && s.inputFocused {
		if len(s.displayName) < 64 {
			s.displayName += string(char)
		}
	}
}

// HandleBackspace removes the last character from input.
func (s *Screen) HandleBackspace() {
	if s.state == StateDisplayName && s.inputFocused && len(s.displayName) > 0 {
		s.displayName = s.displayName[:len(s.displayName)-1]
	}
}

// isClickOnButton checks if a click is within a button's bounds.
func (s *Screen) isClickOnButton(clickX, clickY, buttonCenterX, buttonCenterY int) bool {
	width := 200
	height := 44
	bx := buttonCenterX - width/2
	by := buttonCenterY - height/2
	return clickX >= bx && clickX <= bx+width && clickY >= by && clickY <= by+height
}

func (s *Screen) transitionToPhilosophy() {
	s.state = StatePhilosophy
	s.philosophyStart = time.Now()
	s.philosophyIndex = 0
}

func (s *Screen) transitionToKeypairGen() {
	s.state = StateKeypairGen
	s.phaseStart = time.Now()
}

func (s *Screen) transitionToDisplayName() {
	s.state = StateDisplayName
}

func (s *Screen) transitionToBackupPrompt() {
	s.state = StateBackupPrompt
}

func (s *Screen) transitionToBackupMnemonic() {
	s.state = StateBackupMnemonic
	// In production, generate actual mnemonic here
}

func (s *Screen) transitionToBackupFile() {
	s.state = StateBackupFile
}

func (s *Screen) transitionToBackupComplete() {
	s.state = StateBackupComplete
}

func (s *Screen) completePhase1And2() {
	// Complete identity creation phase
	s.controller.CompleteCurrentPhase() // PhaseWelcome -> PhaseIdentityCreation
	s.controller.CompleteCurrentPhase() // PhaseIdentityCreation -> PhaseModeSelection

	if s.callbacks.OnPhaseComplete != nil {
		s.callbacks.OnPhaseComplete(flow.PhaseIdentityCreation)
	}
}

// GetKeypair returns the generated keypair.
func (s *Screen) GetKeypair() *keys.KeyPair {
	return s.keypair
}

// GetDisplayName returns the entered display name.
func (s *Screen) GetDisplayName() string {
	return s.displayName
}

// IsBackupDone returns whether the user completed backup.
func (s *Screen) IsBackupDone() bool {
	return s.backupDone
}

// State returns the current screen state.
func (s *Screen) State() ScreenState {
	return s.state
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Layout returns the screen dimensions for Ebitengine.
// This implements the ebiten.Game interface.
func (s *Screen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
