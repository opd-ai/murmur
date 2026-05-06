// Package ui provides device pairing UI for multi-device identity management.

//go:build !test
// +build !test

package ui

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/skip2/go-qrcode"
)

// PairingState represents the state of a device pairing operation.
type PairingState int

const (
	PairingStateIdle PairingState = iota
	PairingStateGeneratingQR
	PairingStateWaitingForScan
	PairingStateConnecting
	PairingStateAuthorizing
	PairingStateComplete
	PairingStateError
)

// PairingToken contains the data encoded in the pairing QR code.
type PairingToken struct {
	IPAddress string
	Token     [32]byte
	ExpiresAt time.Time
	MasterKey ed25519.PublicKey
}

// Encode serializes the pairing token to a URL-safe Base64 string.
func (pt *PairingToken) Encode() (string, error) {
	buf := &bytes.Buffer{}

	// Write IP address (length-prefixed)
	ipBytes := []byte(pt.IPAddress)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(ipBytes))); err != nil {
		return "", err
	}
	buf.Write(ipBytes)

	// Write token
	buf.Write(pt.Token[:])

	// Write expiry
	if err := binary.Write(buf, binary.BigEndian, pt.ExpiresAt.Unix()); err != nil {
		return "", err
	}

	// Write master key
	buf.Write(pt.MasterKey)

	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// DecodePairingToken deserializes a pairing token from a Base64-encoded string.
func DecodePairingToken(encoded string) (*PairingToken, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	buf := bytes.NewReader(data)

	ipBytes, err := readIPAddress(buf)
	if err != nil {
		return nil, err
	}

	token, err := readToken(buf)
	if err != nil {
		return nil, err
	}

	expiryUnix, err := readExpiry(buf)
	if err != nil {
		return nil, err
	}

	masterKey, err := readMasterKey(buf)
	if err != nil {
		return nil, err
	}

	return &PairingToken{
		IPAddress: string(ipBytes),
		Token:     token,
		ExpiresAt: time.Unix(expiryUnix, 0),
		MasterKey: masterKey,
	}, nil
}

// readIPAddress reads the variable-length IP address field.
func readIPAddress(buf *bytes.Reader) ([]byte, error) {
	var ipLen uint16
	if err := binary.Read(buf, binary.BigEndian, &ipLen); err != nil {
		return nil, err
	}
	ipBytes := make([]byte, ipLen)
	if _, err := buf.Read(ipBytes); err != nil {
		return nil, err
	}
	return ipBytes, nil
}

// readToken reads the 32-byte token field.
func readToken(buf *bytes.Reader) ([32]byte, error) {
	var token [32]byte
	if _, err := buf.Read(token[:]); err != nil {
		return token, err
	}
	return token, nil
}

// readExpiry reads the 64-bit expiry timestamp.
func readExpiry(buf *bytes.Reader) (int64, error) {
	var expiryUnix int64
	if err := binary.Read(buf, binary.BigEndian, &expiryUnix); err != nil {
		return 0, err
	}
	return expiryUnix, nil
}

// readMasterKey reads the Ed25519 public key field.
func readMasterKey(buf *bytes.Reader) ([]byte, error) {
	masterKey := make([]byte, ed25519.PublicKeySize)
	if _, err := buf.Read(masterKey); err != nil {
		return nil, err
	}
	return masterKey, nil
}

// DevicePairingPanel displays device pairing UI.
type DevicePairingPanel struct {
	mu sync.RWMutex

	visible bool
	x, y    int
	width   int
	height  int
	theme   Theme

	state        PairingState
	qrImage      *ebiten.Image
	pairingToken *PairingToken
	errorMsg     string
	statusMsg    string

	onComplete func(devicePubkey ed25519.PublicKey, label string) error
	onCancel   func()
}

// NewDevicePairingPanel creates a new device pairing panel.
func NewDevicePairingPanel(theme Theme, onComplete func(ed25519.PublicKey, string) error, onCancel func()) *DevicePairingPanel {
	return &DevicePairingPanel{
		theme:      theme,
		width:      600,
		height:     700,
		state:      PairingStateIdle,
		onComplete: onComplete,
		onCancel:   onCancel,
	}
}

// Visible returns true if the panel is shown.
func (p *DevicePairingPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel and starts pairing flow.
func (p *DevicePairingPanel) Show(masterKey ed25519.PublicKey) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.visible = true
	p.state = PairingStateGeneratingQR
	p.errorMsg = ""

	// Generate pairing token
	token, err := p.generatePairingToken(masterKey)
	if err != nil {
		p.state = PairingStateError
		p.errorMsg = fmt.Sprintf("Failed to generate pairing token: %v", err)
		return err
	}
	p.pairingToken = token

	// Generate QR code
	qrImg, err := p.generateQRCode(token)
	if err != nil {
		p.state = PairingStateError
		p.errorMsg = fmt.Sprintf("Failed to generate QR code: %v", err)
		return err
	}
	p.qrImage = qrImg

	p.state = PairingStateWaitingForScan
	p.statusMsg = "Scan this QR code on your new device"

	return nil
}

// Hide hides the panel.
func (p *DevicePairingPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.qrImage = nil
	p.pairingToken = nil
	p.state = PairingStateIdle
}

// generatePairingToken creates a new pairing token with local IP and expiry.
func (p *DevicePairingPanel) generatePairingToken(masterKey ed25519.PublicKey) (*PairingToken, error) {
	// Generate random token
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	// Get local IP address
	localIP, err := p.getLocalIP()
	if err != nil {
		localIP = "127.0.0.1"
	}

	// Set 5-minute expiry per spec
	expiresAt := time.Now().Add(5 * time.Minute)

	return &PairingToken{
		IPAddress: localIP,
		Token:     token,
		ExpiresAt: expiresAt,
		MasterKey: masterKey,
	}, nil
}

// getLocalIP returns the first non-loopback IPv4 address.
func (p *DevicePairingPanel) getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("no non-loopback IP found")
}

// generateQRCode creates a QR code image from the pairing token.
func (p *DevicePairingPanel) generateQRCode(token *PairingToken) (*ebiten.Image, error) {
	encoded, err := token.Encode()
	if err != nil {
		return nil, err
	}

	// Generate QR code with murmur:// scheme
	uri := "murmur://pair/" + encoded
	qr, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	qr.DisableBorder = true
	img := qr.Image(400)

	// Convert to Ebitengine image
	return ebiten.NewImageFromImage(img), nil
}

// Update handles input and updates panel state.
func (p *DevicePairingPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Check for escape key to cancel
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		if p.onCancel != nil {
			p.onCancel()
		}
		return true
	}

	// Check token expiry
	if p.state == PairingStateWaitingForScan && p.pairingToken != nil {
		if time.Now().After(p.pairingToken.ExpiresAt) {
			p.state = PairingStateError
			p.errorMsg = "Pairing token expired. Please try again."
		}
	}

	return true
}

// Draw renders the panel.
func (p *DevicePairingPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	px, py, w, h, shouldRender := CheckPanelVisibilityAndCenter(screen, p.visible, p.width, p.height)
	if !shouldRender {
		return
	}

	// Draw overlay
	overlayColor := ebiten.ColorScale{}
	overlayColor.SetR(0)
	overlayColor.SetG(0)
	overlayColor.SetB(0)
	overlayColor.SetA(0.6)

	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), p.theme.PanelBackground, true)

	// Draw panel background
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 2.0, p.theme.PanelBorder, true)

	// Draw title
	titleY := py + 30
	drawUICenteredText(screen, "Add New Device", float64(px+p.width/2), float64(titleY), p.theme.TextPrimary)

	// Draw content based on state
	contentY := py + 80

	switch p.state {
	case PairingStateGeneratingQR:
		p.drawLoading(screen, px, contentY)

	case PairingStateWaitingForScan:
		p.drawQRCode(screen, px, contentY)

	case PairingStateConnecting:
		p.drawStatus(screen, px, contentY, "Connecting to new device...")

	case PairingStateAuthorizing:
		p.drawStatus(screen, px, contentY, "Authorizing device...")

	case PairingStateComplete:
		p.drawStatus(screen, px, contentY, "Device added successfully!")

	case PairingStateError:
		p.drawError(screen, px, contentY)
	}

	// Draw cancel button
	p.drawCancelButton(screen, px, py+p.height-60)
}

// drawLoading draws a loading indicator.
func (p *DevicePairingPanel) drawLoading(screen *ebiten.Image, x, y int) {
	msg := "Generating QR code..."
	drawUICenteredText(screen, msg, float64(x+p.width/2), float64(y+200), p.theme.TextSecondary)
}

// drawQRCode draws the QR code and instructions.
func (p *DevicePairingPanel) drawQRCode(screen *ebiten.Image, x, y int) {
	if p.qrImage == nil {
		return
	}

	// Draw QR code centered
	qrX := x + (p.width-400)/2
	qrY := y + 20

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(qrX), float64(qrY))
	screen.DrawImage(p.qrImage, opts)

	// Draw instructions
	instrY := qrY + 420
	drawUICenteredText(screen, p.statusMsg, float64(x+p.width/2), float64(instrY), p.theme.TextPrimary)

	// Draw expiry timer
	if p.pairingToken != nil {
		remaining := time.Until(p.pairingToken.ExpiresAt)
		if remaining > 0 {
			timerText := fmt.Sprintf("Expires in %d seconds", int(remaining.Seconds()))
			drawUICenteredText(screen, timerText, float64(x+p.width/2), float64(instrY+30), p.theme.TextSecondary)
		}
	}
}

// drawStatus draws a status message.
func (p *DevicePairingPanel) drawStatus(screen *ebiten.Image, x, y int, msg string) {
	drawUICenteredText(screen, msg, float64(x+p.width/2), float64(y+200), p.theme.TextPrimary)
}

// drawError draws an error message.
func (p *DevicePairingPanel) drawError(screen *ebiten.Image, x, y int) {
	drawUICenteredText(screen, "Error", float64(x+p.width/2), float64(y+150), p.theme.TextError)

	// Draw error message (wrapped if needed)
	errY := y + 200
	drawUICenteredText(screen, p.errorMsg, float64(x+p.width/2), float64(errY), p.theme.TextSecondary)
}

// drawCancelButton draws the cancel button.
func (p *DevicePairingPanel) drawCancelButton(screen *ebiten.Image, x, y int) {
	btnWidth := 120
	btnHeight := 40
	btnX := x + (p.width-btnWidth)/2

	vector.DrawFilledRect(screen, float32(btnX), float32(y),
		float32(btnWidth), float32(btnHeight), p.theme.ButtonBackground, true)
	vector.StrokeRect(screen, float32(btnX), float32(y),
		float32(btnWidth), float32(btnHeight), 1.0, p.theme.PanelBorder, true)

	drawUICenteredText(screen, "Cancel", float64(btnX+btnWidth/2), float64(y+btnHeight/2), p.theme.TextPrimary)
}

// SetState updates the pairing state (for external control).
func (p *DevicePairingPanel) SetState(state PairingState, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
	if msg != "" {
		if state == PairingStateError {
			p.errorMsg = msg
		} else {
			p.statusMsg = msg
		}
	}
}
