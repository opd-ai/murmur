// Package ui provides device pairing UI stubs for testing.

//go:build test
// +build test

package ui

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"time"
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
// Mirrors the real implementation in device_pairing.go (no Ebitengine dependency).
func (pt *PairingToken) Encode() (string, error) {
	buf := &bytes.Buffer{}
	ipBytes := []byte(pt.IPAddress)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(ipBytes))); err != nil {
		return "", err
	}
	buf.Write(ipBytes)
	buf.Write(pt.Token[:])
	if err := binary.Write(buf, binary.BigEndian, pt.ExpiresAt.Unix()); err != nil {
		return "", err
	}
	buf.Write(pt.MasterKey)
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// DecodePairingToken deserializes a pairing token from a Base64-encoded string.
// Mirrors the real implementation in device_pairing.go (no Ebitengine dependency).
func DecodePairingToken(encoded string) (*PairingToken, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}
	buf := bytes.NewReader(data)

	var ipLen uint16
	if err := binary.Read(buf, binary.BigEndian, &ipLen); err != nil {
		return nil, fmt.Errorf("reading IP length: %w", err)
	}
	ipBytes := make([]byte, ipLen)
	if _, err := buf.Read(ipBytes); err != nil {
		return nil, fmt.Errorf("reading IP: %w", err)
	}

	var token [32]byte
	if _, err := buf.Read(token[:]); err != nil {
		return nil, fmt.Errorf("reading token: %w", err)
	}

	var expiryUnix int64
	if err := binary.Read(buf, binary.BigEndian, &expiryUnix); err != nil {
		return nil, fmt.Errorf("reading expiry: %w", err)
	}

	masterKey := make([]byte, ed25519.PublicKeySize)
	if _, err := buf.Read(masterKey); err != nil {
		return nil, fmt.Errorf("reading master key: %w", err)
	}

	return &PairingToken{
		IPAddress: string(ipBytes),
		Token:     token,
		ExpiresAt: time.Unix(expiryUnix, 0),
		MasterKey: masterKey,
	}, nil
}

// DevicePairingPanel displays device pairing UI (stub).
type DevicePairingPanel struct {
	visible bool
	state   PairingState
}

// NewDevicePairingPanel creates a new device pairing panel (stub).
func NewDevicePairingPanel(theme Theme, onComplete func(ed25519.PublicKey, string) error, onCancel func()) *DevicePairingPanel {
	return &DevicePairingPanel{}
}

// Visible returns true if the panel is shown (stub).
func (p *DevicePairingPanel) Visible() bool {
	return p.visible
}

// Show displays the panel (stub).
func (p *DevicePairingPanel) Show(masterKey ed25519.PublicKey) error {
	p.visible = true
	p.state = PairingStateWaitingForScan
	return nil
}

// Hide hides the panel (stub).
func (p *DevicePairingPanel) Hide() {
	p.visible = false
}

// Update handles input (stub).
func (p *DevicePairingPanel) Update() bool {
	return p.visible
}

// Draw renders the panel (stub).
func (p *DevicePairingPanel) Draw(screen image.Image) {
	// Stub: no rendering
}

// SetState updates the pairing state (stub).
func (p *DevicePairingPanel) SetState(state PairingState, msg string) {
	p.state = state
}
