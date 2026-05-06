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

// RecoveryScreen handles identity recovery during onboarding.
// Per ROADMAP.md, supports mnemonic phrase entry and key file import.
type RecoveryScreen struct {
	method         RecoveryMethod
	mnemonicText   string
	keyFileData    []byte
	passphrase     string
	passphraseMode bool // True when entering passphrase for key file
	cursorPos      int
	errorMsg       string
	completed      bool
	recoveredKey   *keys.KeyPair

	// UI state
	hoveredButton int
}

// NewRecoveryScreen creates a new recovery screen.
func NewRecoveryScreen() *RecoveryScreen {
	return &RecoveryScreen{
		method:        RecoveryMethodNone,
		hoveredButton: -1,
	}
}

// Update handles input and state updates for the recovery screen.
func (s *RecoveryScreen) Update() error {
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
		if isInButton(float32(x), float32(y), 200, 500, 120, 40) {
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
	// Mnemonic button: center x, y=250
	centerX := 400.0
	buttonY := 250.0
	buttonWidth := 280.0
	buttonHeight := 50.0

	// Check mnemonic button.
	if isInButton(float32(x), float32(y), float32(centerX), float32(buttonY), float32(buttonWidth), float32(buttonHeight)) {
		s.method = RecoveryMethodMnemonic
		s.errorMsg = ""
		return
	}

	// Key file button: center x, y=320
	buttonY = 320.0
	if isInButton(float32(x), float32(y), float32(centerX), float32(buttonY), float32(buttonWidth), float32(buttonHeight)) {
		s.method = RecoveryMethodKeyFile
		s.errorMsg = ""
		// NOTE: In a real implementation, this would trigger a file picker dialog.
		// For now, we show instructions for manual file loading.
	}
}

// updateMnemonicEntry handles mnemonic phrase text input.
func (s *RecoveryScreen) updateMnemonicEntry() {
	// Append typed characters to mnemonic text.
	s.mnemonicText = appendTypedText(s.mnemonicText)

	// Handle backspace.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.mnemonicText) > 0 {
		s.mnemonicText = s.mnemonicText[:len(s.mnemonicText)-1]
	}

	// Handle Enter to attempt recovery.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.attemptRecovery()
	}

	// Back button click — handled here, not in Draw, to keep rendering side-effect-free.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if isInButton(float32(x), float32(y), 200, 500, 120, 40) {
			s.method = RecoveryMethodNone
			s.mnemonicText = ""
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
		s.passphrase = s.passphrase[:len(s.passphrase)-1]
	}
}

// handlePassphraseBackButton processes back button clicks during passphrase entry.
func (s *RecoveryScreen) handlePassphraseBackButton() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if isInButton(float32(x), float32(y), 200, 500, 120, 40) {
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

	// Attempt to restore keypair.
	kp, err := keys.RestoreFromMnemonic(mnemonic)
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
	// Title.
	DrawCenteredText(screen, "Recover Your Identity", 400, 100, 24, color.RGBA{255, 255, 255, 255})

	// Description.
	DrawCenteredText(screen, "Choose how you want to recover your MURMUR identity", 400, 150, 14, color.RGBA{200, 200, 210, 255})

	// Method buttons.
	style := DefaultButtonStyle()
	style.Width = 280
	DrawButton(screen, "Recovery Phrase (24 words)", 400, 250, style)
	DrawButton(screen, "Key File", 400, 320, style)

	// Error message if any.
	if s.errorMsg != "" {
		DrawCenteredText(screen, s.errorMsg, 400, 400, 12, color.RGBA{255, 100, 100, 255})
	}

	// Help text.
	DrawCenteredText(screen, "Recovery works offline - no network required", 400, 500, 12, color.RGBA{150, 150, 160, 255})
}

// drawMnemonicEntry renders the mnemonic phrase entry UI.
func (s *RecoveryScreen) drawMnemonicEntry(screen *ebiten.Image) {
	// Title.
	DrawCenteredText(screen, "Enter Recovery Phrase", 400, 80, 24, color.RGBA{255, 255, 255, 255})

	// Instructions.
	DrawCenteredText(screen, "Enter your 24-word recovery phrase", 400, 130, 14, color.RGBA{200, 200, 210, 255})
	DrawCenteredText(screen, "Separate words with spaces", 400, 155, 12, color.RGBA{150, 150, 160, 255})

	// Text input box.
	boxX := float32(150)
	boxY := float32(200)
	boxWidth := float32(500)
	boxHeight := float32(120)

	drawInputBox(screen, boxX, boxY, boxWidth, boxHeight)

	// Draw entered text (with wrapping).
	drawWrappedText(screen, s.mnemonicText, boxX+10, boxY+10, boxWidth-20, 14, color.RGBA{220, 220, 230, 255})

	// Error message.
	if s.errorMsg != "" {
		DrawCenteredText(screen, s.errorMsg, 400, 340, 12, color.RGBA{255, 100, 100, 255})
	}

	// Instructions.
	DrawCenteredText(screen, "Press Enter to recover", 400, 380, 12, color.RGBA{150, 150, 160, 255})

	// Back button (click is handled in Update, not here).
	style := DefaultButtonStyle()
	style.Width = 120
	DrawButton(screen, "← Back", 200, 500, style)
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
	return text.Measure(s, helperFace, 0)
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
	// Title.
	DrawCenteredText(screen, "Enter Key File Passphrase", 400, 80, 24, color.RGBA{255, 255, 255, 255})

	// Instructions.
	if !s.passphraseMode {
		DrawCenteredText(screen, "File picker integration pending", 400, 130, 14, color.RGBA{200, 200, 210, 255})
		DrawCenteredText(screen, "Key file import will be available in a future update", 400, 155, 12, color.RGBA{150, 150, 160, 255})
	} else {
		DrawCenteredText(screen, "Enter the passphrase for your encrypted key file", 400, 130, 14, color.RGBA{200, 200, 210, 255})

		// Passphrase input box.
		boxX := float32(250)
		boxY := float32(200)
		boxWidth := float32(300)
		boxHeight := float32(40)

		drawInputBox(screen, boxX, boxY, boxWidth, boxHeight)

		// Draw passphrase as asterisks.
		masked := strings.Repeat("*", len(s.passphrase))
		DrawCenteredText(screen, masked, 400, boxY+15, 14, color.RGBA{220, 220, 230, 255})

		// Error message.
		if s.errorMsg != "" {
			DrawCenteredText(screen, s.errorMsg, 400, 280, 12, color.RGBA{255, 100, 100, 255})
		}

		// Instructions.
		DrawCenteredText(screen, "Press Enter to recover", 400, 320, 12, color.RGBA{150, 150, 160, 255})
	}

	// Back button (click is handled in Update, not here).
	style := DefaultButtonStyle()
	style.Width = 120
	DrawButton(screen, "← Back", 200, 500, style)
}
