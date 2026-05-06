// Package ui provides device management UI for multi-device identity.

//go:build !test
// +build !test

package ui

import (
	"crypto/ed25519"
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DeviceInfo represents an authorized device.
type DeviceInfo struct {
	PublicKey       ed25519.PublicKey
	Label           string
	AuthorizedAt    time.Time
	ExpiresAt       time.Time
	IsCurrentDevice bool
}

// DeviceManagementPanel displays authorized devices and revocation controls.
type DeviceManagementPanel struct {
	mu sync.RWMutex

	visible bool
	x, y    int
	width   int
	height  int
	theme   Theme

	devices       []DeviceInfo
	selectedIndex int
	scrollY       int
	errorMsg      string
	confirmRevoke bool
	revokeTarget  *DeviceInfo

	onRevoke    func(devicePubkey ed25519.PublicKey) error
	onAddDevice func()
	onClose     func()
}

// NewDeviceManagementPanel creates a new device management panel.
func NewDeviceManagementPanel(theme Theme, onRevoke func(ed25519.PublicKey) error, onAddDevice, onClose func()) *DeviceManagementPanel {
	return &DeviceManagementPanel{
		theme:         theme,
		width:         700,
		height:        600,
		selectedIndex: -1,
		onRevoke:      onRevoke,
		onAddDevice:   onAddDevice,
		onClose:       onClose,
	}
}

// Visible returns true if the panel is shown.
func (p *DeviceManagementPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with the provided device list.
func (p *DeviceManagementPanel) Show(devices []DeviceInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.devices = devices
	p.selectedIndex = -1
	p.scrollY = 0
	p.errorMsg = ""
	p.confirmRevoke = false
	p.revokeTarget = nil
}

// Hide hides the panel.
func (p *DeviceManagementPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.confirmRevoke = false
	p.revokeTarget = nil
}

// Update handles input and updates panel state.
func (p *DeviceManagementPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Handle Escape to close or cancel revoke confirmation
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if p.confirmRevoke {
			p.confirmRevoke = false
			p.revokeTarget = nil
		} else {
			p.visible = false
			if p.onClose != nil {
				p.onClose()
			}
		}
		return true
	}

	// Handle scrolling
	p.handleScrolling()

	return true
}

// handleScrolling processes mouse wheel scrolling.
func (p *DeviceManagementPanel) handleScrolling() {
	_, dy := ebiten.Wheel()
	p.scrollY -= int(dy * 30)
	if p.scrollY < 0 {
		p.scrollY = 0
	}

	const deviceItemHeight = 80
	maxScroll := len(p.devices)*deviceItemHeight - (p.height - 200)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollY > maxScroll {
		p.scrollY = maxScroll
	}
}

// Draw renders the panel.
func (p *DeviceManagementPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	px, py, w, h, shouldRender := CheckPanelVisibilityAndCenter(screen, p.visible, p.width, p.height)
	if !shouldRender {
		return
	}

	DrawModalOverlayAndPanel(screen, px, py, w, h, p.width, p.height, p.theme)

	// Draw title
	titleY := py + 30
	drawUICenteredText(screen, "Device Management", float64(px+p.width/2), float64(titleY), p.theme.TextPrimary)

	// Draw confirmation dialog or device list
	if p.confirmRevoke && p.revokeTarget != nil {
		p.drawRevokeConfirmation(screen, px, py)
	} else {
		p.drawDeviceList(screen, px, py+80)
		p.drawButtons(screen, px, py+p.height-60)
	}

	// Draw error message if present
	if p.errorMsg != "" {
		errY := py + p.height - 100
		drawUICenteredText(screen, p.errorMsg, float64(px+p.width/2), float64(errY), p.theme.TextError)
	}
}

// drawDeviceList draws the list of authorized devices.
func (p *DeviceManagementPanel) drawDeviceList(screen *ebiten.Image, x, y int) {
	if len(p.devices) == 0 {
		noDevicesY := y + 150
		drawUICenteredText(screen, "No authorized devices", float64(x+p.width/2), float64(noDevicesY), p.theme.TextSecondary)
		return
	}

	const itemHeight = 80
	padding := 10

	for i, device := range p.devices {
		itemY := y + i*itemHeight - p.scrollY

		// Skip if scrolled out of view
		if itemY < y-itemHeight || itemY > y+p.height-200 {
			continue
		}

		// Draw device item background
		bgColor := p.theme.InputBackground
		if i == p.selectedIndex {
			bgColor = p.theme.Selection
		}

		vector.DrawFilledRect(screen, float32(x+padding), float32(itemY),
			float32(p.width-padding*2), float32(itemHeight-5), bgColor, true)

		// Draw device label
		labelY := itemY + 20
		labelText := device.Label
		if device.IsCurrentDevice {
			labelText += " (This Device)"
		}
		drawUIText(screen, labelText, float64(x+padding+10), float64(labelY), p.theme.TextPrimary)

		// Draw public key (truncated)
		keyY := itemY + 40
		keyText := fmt.Sprintf("Key: %x...", device.PublicKey[:8])
		drawUIText(screen, keyText, float64(x+padding+10), float64(keyY), p.theme.TextSecondary)

		// Draw authorization date
		dateY := itemY + 60
		dateText := fmt.Sprintf("Authorized: %s", device.AuthorizedAt.Format("2006-01-02 15:04"))
		drawUIText(screen, dateText, float64(x+padding+10), float64(dateY), p.theme.TextSecondary)

		// Draw revoke button (if not current device)
		if !device.IsCurrentDevice {
			p.drawRevokeButton(screen, x+p.width-120, itemY+25, i)
		}
	}
}

// drawRevokeButton draws a revoke button for a device.
func (p *DeviceManagementPanel) drawRevokeButton(screen *ebiten.Image, x, y, deviceIndex int) {
	btnWidth := 100
	btnHeight := 30

	vector.DrawFilledRect(screen, float32(x), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.Warning, true)

	drawUICenteredText(screen, "Revoke", float64(x+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
}

// drawButtons draws the control buttons.
func (p *DeviceManagementPanel) drawButtons(screen *ebiten.Image, x, y int) {
	btnWidth := 150
	btnHeight := 40
	btnSpacing := 20

	// Add Device button
	addX := x + (p.width/2 - btnWidth - btnSpacing/2)
	vector.DrawFilledRect(screen, float32(addX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.AccentPrimary, true)
	drawUICenteredText(screen, "Add Device", float64(addX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)

	// Close button
	closeX := x + (p.width/2 + btnSpacing/2)
	vector.DrawFilledRect(screen, float32(closeX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.ButtonBackground, true)
	vector.StrokeRect(screen, float32(closeX), float32(y),
		float32(btnWidth), float32(btnHeight), 1.0, p.theme.PanelBorder, true)
	drawUICenteredText(screen, "Close", float64(closeX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
}

// drawRevokeConfirmation draws the revocation confirmation dialog.
func (p *DeviceManagementPanel) drawRevokeConfirmation(screen *ebiten.Image, x, y int) {
	dialogWidth := 500
	dialogHeight := 250
	dialogX := x + (p.width-dialogWidth)/2
	dialogY := y + (p.height-dialogHeight)/2

	// Draw dialog background
	vector.DrawFilledRect(screen, float32(dialogX), float32(dialogY),
		float32(dialogWidth), float32(dialogHeight), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(dialogX), float32(dialogY),
		float32(dialogWidth), float32(dialogHeight), 2.0, p.theme.Warning, true)

	// Draw title
	titleY := dialogY + 30
	drawUICenteredText(screen, "Revoke Device?", float64(dialogX+dialogWidth/2), float64(titleY), p.theme.TextPrimary)

	// Draw warning message
	msgY := dialogY + 80
	msg := fmt.Sprintf("Revoke '%s'?", p.revokeTarget.Label)
	drawUICenteredText(screen, msg, float64(dialogX+dialogWidth/2), float64(msgY), p.theme.TextSecondary)

	msg2Y := dialogY + 110
	msg2 := "This device will no longer be able to send messages as you."
	drawUICenteredText(screen, msg2, float64(dialogX+dialogWidth/2), float64(msg2Y), p.theme.TextSecondary)

	// Draw confirmation buttons
	btnWidth := 120
	btnHeight := 40
	btnSpacing := 20
	btnY := dialogY + dialogHeight - 60

	// Cancel button
	cancelX := dialogX + (dialogWidth/2 - btnWidth - btnSpacing/2)
	vector.DrawFilledRect(screen, float32(cancelX), float32(btnY),
		float32(btnWidth), float32(btnHeight), p.theme.ButtonBackground, true)
	drawUICenteredText(screen, "Cancel", float64(cancelX+btnWidth/2), float64(btnY+btnHeight/2), p.theme.TextPrimary)

	// Confirm button
	confirmX := dialogX + (dialogWidth/2 + btnSpacing/2)
	vector.DrawFilledRect(screen, float32(confirmX), float32(btnY),
		float32(btnWidth), float32(btnHeight), p.theme.Warning, true)
	drawUICenteredText(screen, "Revoke", float64(confirmX+btnWidth/2), float64(btnY+btnHeight/2), p.theme.TextPrimary)
}

// RequestRevoke initiates the revocation confirmation dialog.
func (p *DeviceManagementPanel) RequestRevoke(deviceIndex int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if deviceIndex < 0 || deviceIndex >= len(p.devices) {
		return
	}

	p.revokeTarget = &p.devices[deviceIndex]
	p.confirmRevoke = true
}

// ConfirmRevoke executes the revocation.
func (p *DeviceManagementPanel) ConfirmRevoke() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.revokeTarget == nil || p.onRevoke == nil {
		return nil
	}

	err := p.onRevoke(p.revokeTarget.PublicKey)
	if err != nil {
		p.errorMsg = fmt.Sprintf("Failed to revoke: %v", err)
		return err
	}

	p.confirmRevoke = false
	p.revokeTarget = nil
	return nil
}

// SetError sets an error message to display.
func (p *DeviceManagementPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMsg = msg
}
