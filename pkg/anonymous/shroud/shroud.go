// Package shroud provides three-hop onion circuit construction.
// Per SECURITY_PRIVACY.md, Shroud circuits use XChaCha20-Poly1305
// for layer encryption with Curve25519 key exchange.
package shroud

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// CircuitLength is the number of hops in a Shroud circuit.
const CircuitLength = 3

// CircuitRotationInterval is how often circuits are rotated.
const CircuitRotationInterval = 10 * time.Minute

// FixedPacketSize is the padded packet size for traffic analysis resistance.
const FixedPacketSize = 1024

// Beacon advertisement interval.
const BeaconInterval = 5 * time.Minute

// Errors for Shroud operations.
var (
	ErrInsufficientRelays = errors.New("insufficient relays for circuit")
	ErrCircuitClosed      = errors.New("circuit is closed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrInvalidPacket      = errors.New("invalid packet")
	ErrRelayNotFound      = errors.New("relay not found")
)

// RelayInfo describes a Shroud relay node.
type RelayInfo struct {
	PeerID    string   // libp2p peer ID
	PublicKey [32]byte // Curve25519 public key
	Bandwidth uint64   // Advertised bandwidth in bytes/sec
	Uptime    time.Duration
	SeenAt    time.Time
}

// Beacon advertises a node's availability as a Shroud relay.
type Beacon struct {
	mu        sync.RWMutex
	relays    map[string]*RelayInfo
	selfInfo  *RelayInfo
	isRelay   bool
	publicKey [32]byte
	secretKey [32]byte
}

// NewBeacon creates a new Shroud beacon for relay discovery.
func NewBeacon() (*Beacon, error) {
	var secretKey [32]byte
	if _, err := rand.Read(secretKey[:]); err != nil {
		return nil, err
	}

	// Clamp secret key for Curve25519.
	secretKey[0] &= 248
	secretKey[31] &= 127
	secretKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &secretKey)

	return &Beacon{
		relays:    make(map[string]*RelayInfo),
		publicKey: publicKey,
		secretKey: secretKey,
	}, nil
}

// EnableRelay marks this node as a Shroud relay.
func (b *Beacon) EnableRelay(peerID string, bandwidth uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.isRelay = true
	b.selfInfo = &RelayInfo{
		PeerID:    peerID,
		PublicKey: b.publicKey,
		Bandwidth: bandwidth,
		SeenAt:    time.Now(),
	}
}

// IsRelay returns true if this node is a Shroud relay.
func (b *Beacon) IsRelay() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isRelay
}

// AddRelay registers a discovered relay.
func (b *Beacon) AddRelay(info *RelayInfo) {
	if info == nil || info.PeerID == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	info.SeenAt = time.Now()
	b.relays[info.PeerID] = info
}

// RemoveRelay removes a relay from the registry.
func (b *Beacon) RemoveRelay(peerID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.relays, peerID)
}

// GetRelay returns info for a specific relay.
func (b *Beacon) GetRelay(peerID string) (*RelayInfo, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	relay, ok := b.relays[peerID]
	return relay, ok
}

// ListRelays returns all known relays.
func (b *Beacon) ListRelays() []*RelayInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()

	relays := make([]*RelayInfo, 0, len(b.relays))
	for _, r := range b.relays {
		relays = append(relays, r)
	}
	return relays
}

// RelayCount returns the number of known relays.
func (b *Beacon) RelayCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.relays)
}

// PublicKey returns the beacon's Curve25519 public key.
func (b *Beacon) PublicKey() [32]byte {
	return b.publicKey
}

// Circuit represents a three-hop onion circuit.
type Circuit struct {
	mu         sync.RWMutex
	hops       [CircuitLength]*RelayInfo
	sharedKeys [CircuitLength][32]byte
	createdAt  time.Time
	closed     bool
}

// SelectRelays chooses three diverse relays for a circuit.
// Per SECURITY_PRIVACY.md, no two hops should be in the initiator's direct mesh.
func (b *Beacon) SelectRelays(excludePeers []string) ([CircuitLength]*RelayInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.relays) < CircuitLength {
		return [CircuitLength]*RelayInfo{}, ErrInsufficientRelays
	}

	eligible := b.filterEligibleRelays(excludePeers)
	if len(eligible) < CircuitLength {
		return [CircuitLength]*RelayInfo{}, ErrInsufficientRelays
	}

	return selectRandomRelays(eligible)
}

// filterEligibleRelays returns relays not in the exclusion list.
func (b *Beacon) filterEligibleRelays(excludePeers []string) []*RelayInfo {
	excluded := buildExclusionSet(excludePeers)
	var eligible []*RelayInfo
	for _, r := range b.relays {
		if !excluded[r.PeerID] {
			eligible = append(eligible, r)
		}
	}
	return eligible
}

// buildExclusionSet creates a lookup set from excluded peer IDs.
func buildExclusionSet(excludePeers []string) map[string]bool {
	excluded := make(map[string]bool)
	for _, p := range excludePeers {
		excluded[p] = true
	}
	return excluded
}

// selectRandomRelays picks CircuitLength relays without replacement.
func selectRandomRelays(eligible []*RelayInfo) ([CircuitLength]*RelayInfo, error) {
	var selected [CircuitLength]*RelayInfo
	used := make(map[int]bool)

	for i := 0; i < CircuitLength; i++ {
		idx := pickRandomUnusedIndex(len(eligible), used)
		used[idx] = true
		selected[i] = eligible[idx]
	}

	return selected, nil
}

// pickRandomUnusedIndex selects a random index not in the used set.
func pickRandomUnusedIndex(max int, used map[int]bool) int {
	var randomBytes [1]byte
	rand.Read(randomBytes[:])

	idx := int(randomBytes[0]) % max
	for used[idx] {
		idx = (idx + 1) % max
	}
	return idx
}

// BuildCircuit creates a new Shroud circuit through the selected relays.
func (b *Beacon) BuildCircuit(relays [CircuitLength]*RelayInfo) (*Circuit, error) {
	circuit := &Circuit{
		hops:      relays,
		createdAt: time.Now(),
	}

	// Perform key agreement with each hop.
	for i, relay := range relays {
		if relay == nil {
			return nil, ErrRelayNotFound
		}

		// X25519 key agreement.
		var shared [32]byte
		curve25519.ScalarMult(&shared, &b.secretKey, &relay.PublicKey)

		// Derive encryption key using BLAKE3.
		h := blake3.New()
		h.Write(shared[:])
		h.Write([]byte("shroud-hop-key"))
		h.Write([]byte{byte(i)})
		key := h.Sum(nil)

		copy(circuit.sharedKeys[i][:], key[:32])
	}

	return circuit, nil
}

// Encrypt wraps data in three layers of encryption (onion skin).
func (c *Circuit) Encrypt(plaintext []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrCircuitClosed
	}

	// Pad to fixed size for traffic analysis resistance.
	data := padToSize(plaintext, FixedPacketSize)

	// Encrypt in reverse order (outer layer first for decryption order).
	for i := CircuitLength - 1; i >= 0; i-- {
		cipher, err := chacha20poly1305.NewX(c.sharedKeys[i][:])
		if err != nil {
			return nil, err
		}

		nonce := make([]byte, cipher.NonceSize())
		rand.Read(nonce)

		data = append(nonce, cipher.Seal(nil, nonce, data, nil)...)
	}

	return data, nil
}

// DecryptLayer removes one layer of encryption (for relay forwarding).
func DecryptLayer(data []byte, key [32]byte) ([]byte, error) {
	cipher, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	if len(data) < cipher.NonceSize() {
		return nil, ErrInvalidPacket
	}

	nonce := data[:cipher.NonceSize()]
	ciphertext := data[cipher.NonceSize():]

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// IsExpired returns true if the circuit should be rotated.
func (c *Circuit) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.createdAt) > CircuitRotationInterval || c.closed
}

// Close closes the circuit and zeroes key material.
func (c *Circuit) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	// Zero shared keys.
	for i := range c.sharedKeys {
		for j := range c.sharedKeys[i] {
			c.sharedKeys[i][j] = 0
		}
	}
}

// padToSize pads data to a fixed size.
func padToSize(data []byte, size int) []byte {
	if len(data) >= size {
		return data[:size]
	}

	result := make([]byte, size)
	copy(result, data)

	// Fill remaining with random bytes for uniform distribution.
	rand.Read(result[len(data):])

	// Store original length in last 2 bytes for unpadding.
	if size >= 2 {
		result[size-2] = byte(len(data) >> 8)
		result[size-1] = byte(len(data) & 0xFF)
	}

	return result
}

// unpadFromSize extracts original data from padded packet.
func unpadFromSize(data []byte) []byte {
	if len(data) < 2 {
		return data
	}

	originalLen := int(data[len(data)-2])<<8 | int(data[len(data)-1])
	if originalLen > len(data)-2 || originalLen < 0 {
		return data
	}

	return data[:originalLen]
}

// CircuitManager manages circuit lifecycle and rotation.
type CircuitManager struct {
	mu      sync.RWMutex
	beacon  *Beacon
	circuit *Circuit
	exclude []string // Peer IDs to exclude from circuit selection.
}

// NewCircuitManager creates a circuit manager.
func NewCircuitManager(beacon *Beacon, excludePeers []string) *CircuitManager {
	return &CircuitManager{
		beacon:  beacon,
		exclude: excludePeers,
	}
}

// GetCircuit returns the current circuit, building a new one if needed.
func (m *CircuitManager) GetCircuit() (*Circuit, error) {
	m.mu.RLock()
	if m.circuit != nil && !m.circuit.IsExpired() {
		c := m.circuit
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	return m.RotateCircuit()
}

// RotateCircuit builds a new circuit.
func (m *CircuitManager) RotateCircuit() (*Circuit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close old circuit.
	if m.circuit != nil {
		m.circuit.Close()
	}

	// Select relays.
	relays, err := m.beacon.SelectRelays(m.exclude)
	if err != nil {
		return nil, err
	}

	// Build new circuit.
	circuit, err := m.beacon.BuildCircuit(relays)
	if err != nil {
		return nil, err
	}

	m.circuit = circuit
	return circuit, nil
}

// StartRotation runs periodic circuit rotation.
func (m *CircuitManager) StartRotation(ctx context.Context) {
	ticker := time.NewTicker(CircuitRotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.closeCircuit()
			return
		case <-ticker.C:
			m.RotateCircuit()
		}
	}
}

// closeCircuit safely closes the current circuit.
func (m *CircuitManager) closeCircuit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.circuit != nil {
		m.circuit.Close()
	}
}
