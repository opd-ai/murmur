// Package screens provides the UI screens for the MURMUR onboarding flow.
// This file implements the identity recovery screen per ROADMAP.md Milestone v0.9.
package screens

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/identity/keys"
)

// RecoveryMethod represents the type of recovery the user selects.
type RecoveryMethod int

const (
	RecoveryMethodNone RecoveryMethod = iota
	RecoveryMethodMnemonic
	RecoveryMethodKeyFile
)

const (
	recoveryBackButtonWidth  = float32(120)
	recoveryBackButtonHeight = float32(44)
)

// RecoveryScreen handles identity recovery during onboarding.
// Per ROADMAP.md, supports mnemonic phrase entry and key file import.
type RecoveryScreen struct {
	method         RecoveryMethod
	mnemonicText   string
	keyFileData    []byte
	passphrase     string
	passphraseMode bool // True when entering passphrase for key file
	mnemonicPass   bool // True when entering passphrase for mnemonic recovery
	cursorPos      int
	errorMsg       string
	completed      bool
	recoveredKey   *keys.KeyPair

	// UI state
	hoveredButton int
	width         int
	height        int
}

// NewRecoveryScreen creates a new recovery screen.
func NewRecoveryScreen() *RecoveryScreen {
	return &RecoveryScreen{
		method:        RecoveryMethodNone,
		hoveredButton: -1,
		width:         800,
		height:        600,
	}
}

// Update handles input and state updates for the recovery screen.
func (s *RecoveryScreen) Update() error {
	s.refreshLayoutFromWindow()

	if s.method == RecoveryMethodNone {
		s.updateMethodSelection()
		return nil
	}

	s.updateSelectedMethod()
	return nil
}

// updateSelectedMethod handles input for the currently selected recovery method.
func (s *RecoveryScreen) updateSelectedMethod() {
	if s.method == RecoveryMethodMnemonic {
		s.updateMnemonicEntry()
	} else if s.method == RecoveryMethodKeyFile {
		s.updateKeyFileMethod()
	}
}

// updateKeyFileMethod handles key file recovery UI, including passphrase entry and back button.
func (s *RecoveryScreen) updateKeyFileMethod() {
	if s.passphraseMode {
		s.updatePassphraseEntry()
	} else {
		s.handleKeyFileBackButton()
	}
}

// handleKeyFileBackButton processes the back button when no file is loaded yet.
func (s *RecoveryScreen) handleKeyFileBackButton() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		backX, backY := s.backButtonCenter()
		if isInButton(float32(x), float32(y), backX, backY, recoveryBackButtonWidth, recoveryBackButtonHeight) {
			s.method = RecoveryMethodNone
			s.errorMsg = ""
		}
	}
}

// updateMethodSelection handles the recovery method selection UI.
func (s *RecoveryScreen) updateMethodSelection() {
	// Check for clicks on method buttons.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.handleMethodClick(x, y)
	}
}

// handleMethodClick processes clicks on method selection buttons.
func (s *RecoveryScreen) handleMethodClick(x, y int) {
	centerX := float64(s.width) / 2
	buttonY := float64(s.height)/2 - 50
	buttonWidth := 280.0
	buttonHeight := 50.0

	// Check mnemonic button.
	if isInButton(float32(x), float32(y), float32(centerX), float32(buttonY), float32(buttonWidth), float32(buttonHeight)) {
		s.method = RecoveryMethodMnemonic
		s.errorMsg = ""
		return
	}

	// Key file button.
	buttonY += 70
	if isInButton(float32(x), float32(y), float32(centerX), float32(buttonY), float32(buttonWidth), float32(buttonHeight)) {
		s.method = RecoveryMethodKeyFile
		s.errorMsg = ""
		// NOTE: In a real implementation, this would trigger a file picker dialog.
		// For now, we show instructions for manual file loading.
	}
}

// updateMnemonicEntry handles mnemonic phrase text input.
func (s *RecoveryScreen) updateMnemonicEntry() {
	// Tab toggles between mnemonic and passphrase entry fields.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		s.mnemonicPass = !s.mnemonicPass
	}

	if s.mnemonicPass {
		s.appendPassphraseInput()
		s.handlePassphraseBackspace()
	} else {
		// Append typed characters to mnemonic text.
		s.mnemonicText = appendTypedText(s.mnemonicText)

		// Handle backspace.
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.mnemonicText) > 0 {
			runes := []rune(s.mnemonicText)
			s.mnemonicText = string(runes[:len(runes)-1])
		}
	}

	// Handle Enter to attempt recovery.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.attemptRecovery()
	}

	// Back button click — handled here, not in Draw, to keep rendering side-effect-free.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		backX, backY := s.backButtonCenter()
		if isInButton(float32(x), float32(y), backX, backY, recoveryBackButtonWidth, recoveryBackButtonHeight) {
			s.method = RecoveryMethodNone
			s.mnemonicText = ""
			s.passphrase = ""
			s.mnemonicPass = false
			s.errorMsg = ""
		}
	}
}

// updatePassphraseEntry handles passphrase text input for key file recovery.
func (s *RecoveryScreen) updatePassphraseEntry() {
	s.appendPassphraseInput()
	s.handlePassphraseBackspace()

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.attemptKeyFileRecovery()
	}

	s.handlePassphraseBackButton()
}

// appendPassphraseInput adds typed characters to the passphrase.
func (s *RecoveryScreen) appendPassphraseInput() {
	typed := ebiten.AppendInputChars(nil)
	for _, r := range typed {
		if r >= 32 && r <= 126 {
			s.passphrase += string(r)
		}
	}
}

// handlePassphraseBackspace removes last character on backspace.
func (s *RecoveryScreen) handlePassphraseBackspace() {
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.passphrase) > 0 {
		runes := []rune(s.passphrase)
		s.passphrase = string(runes[:len(runes)-1])
	}
}

// handlePassphraseBackButton processes back button clicks during passphrase entry.
func (s *RecoveryScreen) handlePassphraseBackButton() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		backX, backY := s.backButtonCenter()
		if isInButton(float32(x), float32(y), backX, backY, recoveryBackButtonWidth, recoveryBackButtonHeight) {
			s.resetPassphraseState()
		}
	}
}

// resetPassphraseState clears passphrase entry state and returns to method selection.
func (s *RecoveryScreen) resetPassphraseState() {
	s.method = RecoveryMethodNone
	s.passphrase = ""
	s.passphraseMode = false
	s.keyFileData = nil
	s.errorMsg = ""
}

// attemptRecovery tries to recover the keypair from the entered mnemonic.
func (s *RecoveryScreen) attemptRecovery() {
	// Normalize whitespace.
	mnemonic := strings.TrimSpace(s.mnemonicText)
	if mnemonic == "" {
		s.errorMsg = "Please enter your recovery phrase"
		return
	}

	// Validate passphrase length (minimum 12 characters per F-CRYPTO-1).
	if len(s.passphrase) < 12 {
		s.errorMsg = "Passphrase must be at least 12 characters"
		return
	}

	// Attempt to restore keypair with passphrase.
	kp, err := keys.RestoreFromMnemonic(mnemonic, s.passphrase)
	if err != nil {
		s.errorMsg = "Invalid recovery phrase. Please check and try again."
		return
	}

	// Success!
	s.recoveredKey = kp
	s.completed = true
	s.errorMsg = ""
}

// SetKeyFileData sets the key file data for import.
// This would be called by file picker integration.
// Per ROADMAP.md, key file import supports offline recovery.
func (s *RecoveryScreen) SetKeyFileData(data []byte) {
	s.keyFileData = data
	s.passphraseMode = true
	s.errorMsg = ""
}

// attemptKeyFileRecovery tries to recover the keypair from key file and passphrase.
func (s *RecoveryScreen) attemptKeyFileRecovery() {
	if len(s.keyFileData) == 0 {
		s.errorMsg = "No key file loaded"
		return
	}

	passphrase := strings.TrimSpace(s.passphrase)
	if passphrase == "" {
		s.errorMsg = "Please enter your passphrase"
		return
	}

	// Attempt to import keypair from encrypted file.
	kp, err := keys.ImportKeyPairFromFile(s.keyFileData, passphrase)
	if err != nil {
		s.errorMsg = "Invalid passphrase or corrupted key file"
		return
	}

	// Success!
	s.recoveredKey = kp
	s.completed = true
	s.errorMsg = ""
	// Zero the key file data.
	keys.ZeroBytes(s.keyFileData)
	s.keyFileData = nil
}

// Draw renders the recovery screen.
func (s *RecoveryScreen) Draw(screen *ebiten.Image) {
	s.width = screen.Bounds().Dx()
	s.height = screen.Bounds().Dy()
	screen.Fill(color.RGBA{15, 15, 20, 255})

	if s.method == RecoveryMethodNone {
		s.drawMethodSelection(screen)
	} else if s.method == RecoveryMethodMnemonic {
		s.drawMnemonicEntry(screen)
	} else if s.method == RecoveryMethodKeyFile {
		s.drawKeyFileEntry(screen)
	}
}

// drawMethodSelection renders the recovery method selection UI.
func (s *RecoveryScreen) drawMethodSelection(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	methodY := float32(s.height)/2 - 50
	keyFileY := methodY + 70
	helpY := float32(s.height) - 100
	errorY := helpY - 40

	// Title.
	DrawCenteredText(screen, "Recover Your Identity", centerX, 100, 24, color.RGBA{255, 255, 255, 255})

	// Description.
	DrawCenteredText(screen, "Choose how you want to recover your MURMUR identity", centerX, 150, 14, color.RGBA{200, 200, 210, 255})

	// Method buttons.
	style := DefaultButtonStyle()
	style.Width = 280
	DrawButton(screen, "Recovery Phrase (24 words)", centerX, methodY, style)
	DrawButton(screen, "Key File", centerX, keyFileY, style)

	// Error message if any.
	if s.errorMsg != "" {
		DrawCenteredText(screen, s.errorMsg, centerX, errorY, 12, color.RGBA{255, 100, 100, 255})
	}

	// Help text.
	DrawCenteredText(screen, "Recovery works offline - no network required", centerX, helpY, 12, color.RGBA{150, 150, 160, 255})
}

// drawMnemonicEntry renders the mnemonic phrase entry UI.
func (s *RecoveryScreen) drawMnemonicEntry(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	boxWidth := float32(500)
	maxW := float32(s.width - 40)
	if maxW < 220 {
		maxW = 220
	}
	if boxWidth > maxW {
		boxWidth = maxW
	}
	mnemonicBoxHeight := float32(120)
	boxX := centerX - boxWidth/2
	mnemonicBoxY := float32(200)
	passphraseBoxY := mnemonicBoxY + mnemonicBoxHeight + 30
	passphraseBoxHeight := float32(40)
	errorY := passphraseBoxY + passphraseBoxHeight + 20
	instructionY := errorY + 40
	backX, backY := s.backButtonCenter()

	// Title.
	DrawCenteredText(screen, "Enter Recovery Phrase", centerX, 80, 24, color.RGBA{255, 255, 255, 255})

	// Instructions.
	DrawCenteredText(screen, "Enter your 24-word recovery phrase", centerX, 130, 14, color.RGBA{200, 200, 210, 255})
	DrawCenteredText(screen, "Separate words with spaces", centerX, 155, 12, color.RGBA{150, 150, 160, 255})

	drawInputBox(screen, boxX, mnemonicBoxY, boxWidth, mnemonicBoxHeight)

	// Draw entered text (with wrapping).
	drawWrappedText(screen, s.mnemonicText, boxX+10, mnemonicBoxY+10, boxWidth-20, 14, color.RGBA{220, 220, 230, 255})

	// Passphrase entry.
	DrawCenteredText(screen, "Recovery Passphrase (minimum 12 characters)", centerX, passphraseBoxY-12, 12, color.RGBA{180, 180, 190, 255})
	drawInputBox(screen, boxX, passphraseBoxY, boxWidth, passphraseBoxHeight)
	masked := strings.Repeat("*", len([]rune(s.passphrase)))
	masked = clipTextWithEllipsis(masked, float64(boxWidth-20))
	DrawCenteredText(screen, masked, centerX, passphraseBoxY+15, 14, color.RGBA{220, 220, 230, 255})

	// Error message.
	if s.errorMsg != "" {
		DrawCenteredText(screen, s.errorMsg, centerX, errorY, 12, color.RGBA{255, 100, 100, 255})
	}

	// Instructions.
	DrawCenteredText(screen, "Press Tab to switch input · Press Enter to recover", centerX, instructionY, 12, color.RGBA{150, 150, 160, 255})

	// Back button (click is handled in Update, not here).
	style := DefaultButtonStyle()
	style.Width = 120
	DrawButton(screen, "← Back", backX, backY, style)
}

// IsCompleted returns true if recovery was successful.
func (s *RecoveryScreen) IsCompleted() bool {
	return s.completed
}

// GetRecoveredKey returns the recovered keypair.
func (s *RecoveryScreen) GetRecoveredKey() *keys.KeyPair {
	return s.recoveredKey
}

// Reset resets the recovery screen state.
func (s *RecoveryScreen) Reset() {
	s.method = RecoveryMethodNone
	s.mnemonicText = ""
	s.passphrase = ""
	s.passphraseMode = false
	s.mnemonicPass = false
	s.errorMsg = ""
	s.completed = false
	s.recoveredKey = nil
	s.hoveredButton = -1
}

// appendTypedText appends characters typed during this frame.
func appendTypedText(text string) string {
	// Get all printable runes typed this frame.
	typed := ebiten.AppendInputChars(nil)
	for _, r := range typed {
		// Only accept lowercase letters and spaces for mnemonic.
		if (r >= 'a' && r <= 'z') || r == ' ' {
			text += string(r)
		}
	}
	return text
}

// drawWrappedText draws text with word wrapping.
// Per audit LOW finding: the previous implementation measured line width as
// len(testLine)*7 (bytes × 7 px), which is incorrect for non-ASCII text.
// text.Measure is used instead for accurate pixel-width calculations.
// drawInputBox draws a styled input box with background and border.
func drawInputBox(screen *ebiten.Image, x, y, width, height float32) {
	vector.DrawFilledRect(screen, x, y, width, height, color.RGBA{40, 40, 60, 255}, true)
	vector.StrokeRect(screen, x, y, width, height, 1, color.RGBA{70, 80, 120, 255}, true)
}

func drawWrappedText(screen *ebiten.Image, text string, x, y, maxWidth float32, size int, clr color.Color) {
	// Simple word wrapping implementation.
	words := strings.Fields(text)
	currentLine := ""
	lineY := y

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		// Use text.Measure for the exact rendered pixel width.
		measuredW, _ := measureLine(testLine)
		if measuredW > float64(maxWidth) {
			// Draw current line and start new one.
			if currentLine != "" {
				DrawCenteredText(screen, currentLine, x+maxWidth/2, lineY, float64(size), clr)
				lineY += float32(size + 4)
			}
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	// Draw remaining line.
	if currentLine != "" {
		DrawCenteredText(screen, currentLine, x+maxWidth/2, lineY, float64(size), clr)
	}
}

// measureLine returns the rendered pixel width and height of s using the shared
// helper font face from helpers.go.  Uses text.Measure for accurate pixel widths
// that account for glyph advance widths, not just byte count × 7.
func measureLine(s string) (float64, float64) {
	return text.Measure(s, helperFaceForSize(14), 0)
}

// clipTextWithEllipsis trims text to maxWidth and appends an ellipsis when clipped.
func clipTextWithEllipsis(content string, maxWidth float64) string {
	if content == "" {
		return content
	}
	w, _ := measureLine(content)
	if w <= maxWidth {
		return content
	}
	ellipsis := "..."
	maxTextWidth := maxWidth
	ew, _ := measureLine(ellipsis)
	if maxTextWidth > ew {
		maxTextWidth -= ew
	}
	runes := []rune(content)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := string(runes)
		cw, _ := measureLine(candidate)
		if cw <= maxTextWidth {
			return candidate + ellipsis
		}
	}
	return ellipsis
}

// isInButton checks if a point is within a button's bounds.
func isInButton(x, y, centerX, centerY, width, height float32) bool {
	left := centerX - width/2
	right := centerX + width/2
	top := centerY - height/2
	bottom := centerY + height/2
	return x >= left && x <= right && y >= top && y <= bottom
}

// drawKeyFileEntry renders the key file passphrase entry UI.
func (s *RecoveryScreen) drawKeyFileEntry(screen *ebiten.Image) {
	centerX := float32(s.width) / 2
	backX, backY := s.backButtonCenter()

	// Title.
	DrawCenteredText(screen, "Enter Key File Passphrase", centerX, 80, 24, color.RGBA{255, 255, 255, 255})

	// Instructions.
	if !s.passphraseMode {
		DrawCenteredText(screen, "File picker integration pending", centerX, 130, 14, color.RGBA{200, 200, 210, 255})
		DrawCenteredText(screen, "Key file import will be available in a future update", centerX, 155, 12, color.RGBA{150, 150, 160, 255})
	} else {
		DrawCenteredText(screen, "Enter the passphrase for your encrypted key file", centerX, 130, 14, color.RGBA{200, 200, 210, 255})

		// Passphrase input box.
		boxWidth := float32(300)
		maxW := float32(s.width - 40)
		if maxW < 220 {
			maxW = 220
		}
		if boxWidth > maxW {
			boxWidth = maxW
		}
		boxX := centerX - boxWidth/2
		boxY := float32(200)
		boxHeight := float32(40)

		drawInputBox(screen, boxX, boxY, boxWidth, boxHeight)

		// Draw passphrase as asterisks.
		masked := strings.Repeat("*", len([]rune(s.passphrase)))
		masked = clipTextWithEllipsis(masked, float64(boxWidth-20))
		DrawCenteredText(screen, masked, centerX, boxY+15, 14, color.RGBA{220, 220, 230, 255})

		// Error message.
		if s.errorMsg != "" {
			DrawCenteredText(screen, s.errorMsg, centerX, 280, 12, color.RGBA{255, 100, 100, 255})
		}

		// Instructions.
		DrawCenteredText(screen, "Press Enter to recover", centerX, 320, 12, color.RGBA{150, 150, 160, 255})
	}

	// Back button (click is handled in Update, not here).
	style := DefaultButtonStyle()
	style.Width = recoveryBackButtonWidth
	style.Height = recoveryBackButtonHeight
	DrawButton(screen, "← Back", backX, backY, style)
}

func (s *RecoveryScreen) refreshLayoutFromWindow() {
	w, h := ebiten.WindowSize()
	if w > 0 {
		s.width = w
	}
	if h > 0 {
		s.height = h
	}
	if s.width <= 0 {
		s.width = 800
	}
	if s.height <= 0 {
		s.height = 600
	}
}

func (s *RecoveryScreen) backButtonCenter() (float32, float32) {
	return 90, float32(s.height - 50)
}
