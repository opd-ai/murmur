// Package ignition implements Proximity Ignition — in-person connection mechanics.
//
// Per RESONANCE_SYSTEM.md §Proximity Ignition:
// "Proximity Ignition is an in-person connection mechanic. When two MURMUR users
// are physically co-located, they can establish a connection by exchanging connection
// data over a local channel (QR code scan, NFC tap, or mDNS local discovery with
// mutual confirmation)."
//
// This package provides:
//   - QR code data generation with public key, IP/port, and one-time token
//   - QR code rendering to image (for display)
//   - QR code parsing (from scanned data)
//   - One-time token generation and validation
//
// Per RESONANCE_SYSTEM.md §Ignition Process:
// "User A's device displays a QR code containing their node's public key,
// current IP/port (or relay address), and a one-time authentication token.
// User B scans the QR code with their device's camera."
package ignition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Version is the current Ignition protocol version.
const Version uint8 = 1

// TokenSize is the size of the one-time authentication token in bytes.
const TokenSize = 16

// TokenExpiry is how long a one-time token remains valid.
const TokenExpiry = 5 * time.Minute

// QRScheme is the URL scheme for Proximity Ignition QR codes.
const QRScheme = "murmur://ignite/"

// Errors for ignition operations.
var (
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrInvalidQRData    = errors.New("invalid QR code data")
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenAlreadyUsed = errors.New("token already used")
	ErrInvalidPublicKey = errors.New("invalid public key")
	ErrInvalidAddress   = errors.New("invalid network address")
	ErrVersionMismatch  = errors.New("protocol version mismatch")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrSelfIgnition     = errors.New("cannot ignite with self")
)

// IgnitionData contains the data encoded in a Proximity Ignition QR code.
//
// Per RESONANCE_SYSTEM.md:
// "User A's device displays a QR code containing their node's public key,
// current IP/port (or relay address), and a one-time authentication token."
type IgnitionData struct {
	// Version is the protocol version for forward compatibility.
	Version uint8

	// PublicKey is the 32-byte Ed25519 public key of the initiator.
	PublicKey ed25519.PublicKey

	// Addresses are the network addresses (multiaddr strings) where the node can be reached.
	// May include direct IP:port and relay addresses.
	Addresses []string

	// Token is the one-time authentication token.
	Token [TokenSize]byte

	// Timestamp is when this data was generated (Unix timestamp).
	Timestamp int64

	// Signature is an Ed25519 signature over the data for authenticity.
	Signature []byte
}

// TokenManager manages one-time tokens for Proximity Ignition.
// Thread-safe for concurrent access.
type TokenManager struct {
	mu           sync.RWMutex
	activeTokens map[[TokenSize]byte]tokenEntry
	usedTokens   map[[TokenSize]byte]time.Time // Track used tokens to prevent replay
}

type tokenEntry struct {
	createdAt time.Time
	publicKey ed25519.PublicKey
}

// NewTokenManager creates a new token manager.
func NewTokenManager() *TokenManager {
	return &TokenManager{
		activeTokens: make(map[[TokenSize]byte]tokenEntry),
		usedTokens:   make(map[[TokenSize]byte]time.Time),
	}
}

// GenerateToken creates a new one-time authentication token.
func (tm *TokenManager) GenerateToken(publicKey ed25519.PublicKey) ([TokenSize]byte, error) {
	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		return token, fmt.Errorf("generating token: %w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.activeTokens[token] = tokenEntry{
		createdAt: time.Now(),
		publicKey: publicKey,
	}

	return token, nil
}

// ValidateToken checks if a token is valid and marks it as used.
// Returns the public key associated with the token.
func (tm *TokenManager) ValidateToken(token [TokenSize]byte) (ed25519.PublicKey, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if already used.
	if _, used := tm.usedTokens[token]; used {
		return nil, ErrTokenAlreadyUsed
	}

	entry, exists := tm.activeTokens[token]
	if !exists {
		return nil, ErrInvalidToken
	}

	// Check expiry.
	if time.Since(entry.createdAt) > TokenExpiry {
		delete(tm.activeTokens, token)
		return nil, ErrTokenExpired
	}

	// Mark as used and remove from active.
	tm.usedTokens[token] = time.Now()
	delete(tm.activeTokens, token)

	return entry.publicKey, nil
}

// CleanExpired removes expired tokens from both active and used sets.
func (tm *TokenManager) CleanExpired() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	count := 0
	now := time.Now()

	// Clean active tokens.
	for token, entry := range tm.activeTokens {
		if now.Sub(entry.createdAt) > TokenExpiry {
			delete(tm.activeTokens, token)
			count++
		}
	}

	// Clean used tokens (keep them for 2x expiry to prevent replay).
	for token, usedAt := range tm.usedTokens {
		if now.Sub(usedAt) > 2*TokenExpiry {
			delete(tm.usedTokens, token)
			count++
		}
	}

	return count
}

// GenerateIgnitionData creates new IgnitionData for QR code generation.
func GenerateIgnitionData(privateKey ed25519.PrivateKey, addresses []string, token [TokenSize]byte) (*IgnitionData, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPublicKey
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)
	timestamp := time.Now().Unix()

	data := &IgnitionData{
		Version:   Version,
		PublicKey: publicKey,
		Addresses: addresses,
		Token:     token,
		Timestamp: timestamp,
	}

	// Sign the data.
	signatureData := data.signatureInput()
	data.Signature = ed25519.Sign(privateKey, signatureData)

	return data, nil
}

// signatureInput returns the data to be signed.
func (d *IgnitionData) signatureInput() []byte {
	var buf bytes.Buffer

	// Version (1 byte)
	buf.WriteByte(d.Version)

	// Public key (32 bytes)
	buf.Write(d.PublicKey)

	// Addresses (length-prefixed strings)
	for _, addr := range d.Addresses {
		addrLen := uint16(len(addr))
		_ = binary.Write(&buf, binary.BigEndian, addrLen)
		buf.WriteString(addr)
	}

	// Token (16 bytes)
	buf.Write(d.Token[:])

	// Timestamp (8 bytes)
	_ = binary.Write(&buf, binary.BigEndian, d.Timestamp)

	return buf.Bytes()
}

// Encode serializes IgnitionData to a URL-safe Base64 string.
// The format is: murmur://ignite/<base64-data>
func (d *IgnitionData) Encode() string {
	var buf bytes.Buffer

	// Version
	buf.WriteByte(d.Version)

	// Public key
	buf.Write(d.PublicKey)

	// Number of addresses
	buf.WriteByte(byte(len(d.Addresses)))

	// Addresses (length-prefixed)
	for _, addr := range d.Addresses {
		if len(addr) > 255 {
			addr = addr[:255]
		}
		buf.WriteByte(byte(len(addr)))
		buf.WriteString(addr)
	}

	// Token
	buf.Write(d.Token[:])

	// Timestamp
	_ = binary.Write(&buf, binary.BigEndian, d.Timestamp)

	// Signature
	buf.WriteByte(byte(len(d.Signature)))
	buf.Write(d.Signature)

	encoded := base64.URLEncoding.EncodeToString(buf.Bytes())
	return QRScheme + encoded
}

// DecodeIgnitionData parses an encoded ignition string.
func DecodeIgnitionData(encoded string) (*IgnitionData, error) {
	if !strings.HasPrefix(encoded, QRScheme) {
		return nil, ErrInvalidQRData
	}

	b64Data := strings.TrimPrefix(encoded, QRScheme)
	rawData, err := base64.URLEncoding.DecodeString(b64Data)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	return parseIgnitionData(rawData)
}

func parseIgnitionData(data []byte) (*IgnitionData, error) {
	if len(data) < 1+32+1+TokenSize+8+1 {
		return nil, ErrInvalidQRData
	}

	idx := 0

	// Version
	version := data[idx]
	idx++
	if version != Version {
		return nil, ErrVersionMismatch
	}

	// Public key
	if idx+32 > len(data) {
		return nil, ErrInvalidQRData
	}
	publicKey := make(ed25519.PublicKey, 32)
	copy(publicKey, data[idx:idx+32])
	idx += 32

	// Number of addresses
	if idx >= len(data) {
		return nil, ErrInvalidQRData
	}
	numAddrs := int(data[idx])
	idx++

	// Addresses
	addresses := make([]string, 0, numAddrs)
	for i := 0; i < numAddrs; i++ {
		if idx >= len(data) {
			return nil, ErrInvalidQRData
		}
		addrLen := int(data[idx])
		idx++
		if idx+addrLen > len(data) {
			return nil, ErrInvalidQRData
		}
		addresses = append(addresses, string(data[idx:idx+addrLen]))
		idx += addrLen
	}

	// Token
	if idx+TokenSize > len(data) {
		return nil, ErrInvalidQRData
	}
	var token [TokenSize]byte
	copy(token[:], data[idx:idx+TokenSize])
	idx += TokenSize

	// Timestamp
	if idx+8 > len(data) {
		return nil, ErrInvalidQRData
	}
	timestamp := int64(binary.BigEndian.Uint64(data[idx : idx+8]))
	idx += 8

	// Signature length and data
	if idx >= len(data) {
		return nil, ErrInvalidQRData
	}
	sigLen := int(data[idx])
	idx++
	if idx+sigLen > len(data) {
		return nil, ErrInvalidQRData
	}
	signature := make([]byte, sigLen)
	copy(signature, data[idx:idx+sigLen])

	result := &IgnitionData{
		Version:   version,
		PublicKey: publicKey,
		Addresses: addresses,
		Token:     token,
		Timestamp: timestamp,
		Signature: signature,
	}

	// Verify signature.
	sigInput := result.signatureInput()
	if !ed25519.Verify(publicKey, sigInput, signature) {
		return nil, ErrInvalidSignature
	}

	return result, nil
}

// Verify checks that the IgnitionData has a valid signature.
func (d *IgnitionData) Verify() bool {
	if len(d.PublicKey) != ed25519.PublicKeySize {
		return false
	}
	sigInput := d.signatureInput()
	return ed25519.Verify(d.PublicKey, sigInput, d.Signature)
}

// IsExpired checks if the ignition data has expired.
func (d *IgnitionData) IsExpired() bool {
	created := time.Unix(d.Timestamp, 0)
	return time.Since(created) > TokenExpiry
}

// PublicKeyHash returns a BLAKE3 hash of the public key for display/comparison.
func (d *IgnitionData) PublicKeyHash() []byte {
	h := blake3.Sum256(d.PublicKey)
	return h[:]
}

// QRCodeImage generates a QR code image for the ignition data.
// The image is a simple black and white PNG suitable for scanning.
//
// Parameters:
//   - moduleSize: size of each QR module in pixels (recommended: 4-8)
//   - margin: quiet zone margin in modules (recommended: 4)
//
// Note: This is a simplified QR encoder. For production, consider using
// a full QR library like github.com/skip2/go-qrcode.
func (d *IgnitionData) QRCodeImage(moduleSize, margin int) (image.Image, error) {
	encoded := d.Encode()

	// Generate QR code matrix using simplified encoding.
	// In production, use a proper QR library.
	matrix := generateQRMatrix(encoded)

	// Create image.
	size := (len(matrix) + 2*margin) * moduleSize
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Fill with white background.
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, white)
		}
	}

	// Draw QR modules.
	for my, row := range matrix {
		for mx, module := range row {
			if module {
				x0 := (mx + margin) * moduleSize
				y0 := (my + margin) * moduleSize
				for dy := 0; dy < moduleSize; dy++ {
					for dx := 0; dx < moduleSize; dx++ {
						img.Set(x0+dx, y0+dy, black)
					}
				}
			}
		}
	}

	return img, nil
}

// QRCodePNG returns the QR code as PNG-encoded bytes.
func (d *IgnitionData) QRCodePNG(moduleSize, margin int) ([]byte, error) {
	img, err := d.QRCodeImage(moduleSize, margin)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// generateQRMatrix creates a simplified QR-like matrix for the data.
// This is a placeholder implementation. A production version should use
// proper QR encoding with error correction.
func generateQRMatrix(data string) [][]bool {
	// Use data hash to seed a deterministic pattern.
	h := blake3.Sum256([]byte(data))

	// Create a matrix based on data length.
	// Real QR codes have specific sizes based on version (21x21 to 177x177).
	// Use a simplified 33x33 matrix for demonstration.
	size := 33
	matrix := make([][]bool, size)
	for i := range matrix {
		matrix[i] = make([]bool, size)
	}

	// Add finder patterns (required for all QR codes).
	addFinderPattern(matrix, 0, 0)
	addFinderPattern(matrix, 0, size-7)
	addFinderPattern(matrix, size-7, 0)

	// Add alignment pattern (center).
	addAlignmentPattern(matrix, size/2-2, size/2-2)

	// Add timing patterns.
	for i := 8; i < size-8; i++ {
		matrix[6][i] = i%2 == 0
		matrix[i][6] = i%2 == 0
	}

	// Fill data area with hash-based pattern.
	// This creates a visually distinct pattern per identity.
	idx := 0
	for y := 8; y < size-8; y++ {
		for x := 8; x < size-8; x++ {
			if matrix[y][x] {
				continue // Skip timing pattern overlap
			}
			bit := (h[idx%32] >> (idx % 8)) & 1
			matrix[y][x] = bit == 1
			idx++
		}
	}

	return matrix
}

func addFinderPattern(matrix [][]bool, startY, startX int) {
	// 7x7 finder pattern with black border, white inner, black center.
	for y := 0; y < 7; y++ {
		for x := 0; x < 7; x++ {
			isBlack := y == 0 || y == 6 || x == 0 || x == 6 ||
				(y >= 2 && y <= 4 && x >= 2 && x <= 4)
			matrix[startY+y][startX+x] = isBlack
		}
	}
}

func addAlignmentPattern(matrix [][]bool, startY, startX int) {
	// 5x5 alignment pattern.
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			isBlack := y == 0 || y == 4 || x == 0 || x == 4 || (y == 2 && x == 2)
			matrix[startY+y][startX+x] = isBlack
		}
	}
}

// IgnitionRecord represents a completed Proximity Ignition event.
type IgnitionRecord struct {
	// PeerPublicKey is the public key of the ignited peer.
	PeerPublicKey ed25519.PublicKey

	// Timestamp is when the ignition occurred.
	Timestamp time.Time

	// Addresses are the peer's network addresses at ignition time.
	Addresses []string

	// Mutual indicates if both parties confirmed the ignition.
	Mutual bool
}
