// Package shroud provides three-hop onion circuit construction.
// Per SECURITY_PRIVACY.md, Shroud circuits use XChaCha20-Poly1305
// for layer encryption with Curve25519 key exchange.
package shroud

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opd-ai/murmur/pkg/networking/metrics"
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

// Error recovery constants.
const (
	// MaxConsecutiveFailures is the threshold before marking a relay as bad.
	MaxConsecutiveFailures = 3
	// RelayPenaltyDuration is how long a failed relay is excluded.
	RelayPenaltyDuration = 10 * time.Minute
	// CircuitRebuildBackoff is the minimum delay between rebuild attempts.
	CircuitRebuildBackoff = 5 * time.Second
)

// Errors for Shroud operations.
var (
	ErrInsufficientRelays = errors.New("insufficient relays for circuit")
	ErrCircuitClosed      = errors.New("circuit is closed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrInvalidPacket      = errors.New("invalid packet")
	ErrRelayNotFound      = errors.New("relay not found")
	ErrRelayFailure       = errors.New("relay failure detected")
	ErrAllCircuitsFailed  = errors.New("all circuits have failed")
	ErrReplayDetected     = errors.New("replay attack detected")
)

// CircuitError categorizes circuit-related errors.
type CircuitError struct {
	Err       error
	RelayID   string    // Which relay failed (if applicable).
	CircuitID [16]byte  // Which circuit experienced the error.
	Timestamp time.Time // When the error occurred.
	Transient bool      // Whether the error might be temporary.
}

// Error implements the error interface.
func (e *CircuitError) Error() string {
	if e.RelayID != "" {
		return "circuit error (relay " + e.RelayID + "): " + e.Err.Error()
	}
	return "circuit error: " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *CircuitError) Unwrap() error {
	return e.Err
}

// NewCircuitError creates a new CircuitError.
func NewCircuitError(err error, relayID string, circuitID [16]byte, transient bool) *CircuitError {
	return &CircuitError{
		Err:       err,
		RelayID:   relayID,
		CircuitID: circuitID,
		Timestamp: time.Now(),
		Transient: transient,
	}
}

// RelayFailureTracker tracks relay failures for error recovery.
type RelayFailureTracker struct {
	mu           sync.RWMutex
	failures     map[string]int       // Consecutive failure count per relay.
	penaltyUntil map[string]time.Time // When penalty expires.
}

// NewRelayFailureTracker creates a new failure tracker.
func NewRelayFailureTracker() *RelayFailureTracker {
	return &RelayFailureTracker{
		failures:     make(map[string]int),
		penaltyUntil: make(map[string]time.Time),
	}
}

// RecordFailure records a relay failure.
// Returns true if the relay has exceeded the failure threshold.
func (t *RelayFailureTracker) RecordFailure(relayID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.failures[relayID]++
	if t.failures[relayID] >= MaxConsecutiveFailures {
		t.penaltyUntil[relayID] = time.Now().Add(RelayPenaltyDuration)
		return true
	}
	return false
}

// RecordSuccess clears failure tracking for a relay.
func (t *RelayFailureTracker) RecordSuccess(relayID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.failures, relayID)
}

// IsPenalized returns true if the relay is currently penalized.
func (t *RelayFailureTracker) IsPenalized(relayID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	until, ok := t.penaltyUntil[relayID]
	if !ok {
		return false
	}

	if time.Now().After(until) {
		// Penalty expired, clean up (will be done on next write).
		return false
	}
	return true
}

// PenalizedRelays returns the list of currently penalized relay IDs.
func (t *RelayFailureTracker) PenalizedRelays() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	var penalized []string
	for relayID, until := range t.penaltyUntil {
		if until.After(now) {
			penalized = append(penalized, relayID)
		}
	}
	return penalized
}

// CleanExpired removes expired penalty entries.
func (t *RelayFailureTracker) CleanExpired() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for relayID, until := range t.penaltyUntil {
		if until.Before(now) {
			delete(t.penaltyUntil, relayID)
			delete(t.failures, relayID)
		}
	}
}

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

// DisableRelay removes this node from being a Shroud relay.
func (b *Beacon) DisableRelay() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.isRelay = false
	b.selfInfo = nil
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

// addRelayWithTime registers a relay with a specific SeenAt time (for testing).
func (b *Beacon) addRelayWithTime(info *RelayInfo) {
	if info == nil || info.PeerID == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

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

// SecretKey returns the beacon's Curve25519 secret key.
// Used for signing advertisements.
func (b *Beacon) SecretKey() [32]byte {
	return b.secretKey
}

// SelfInfo returns this node's relay info if it is a relay.
func (b *Beacon) SelfInfo() *RelayInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.selfInfo
}

// TeardownFunc is called when a circuit is torn down.
// It receives the circuit ID and the list of relay peer IDs.
type TeardownFunc func(circuitID []byte, relayPeerIDs []string)

// NonceSequencer handles nonce generation with replay protection.
type NonceSequencer struct {
	mu       sync.Mutex
	sequence uint64  // Monotonic sequence number.
	prefix   [8]byte // Random prefix for nonce uniqueness.
}

// NewNonceSequencer creates a new nonce sequencer.
func NewNonceSequencer() *NonceSequencer {
	ns := &NonceSequencer{}
	rand.Read(ns.prefix[:])
	return ns
}

// Next returns the next nonce in sequence.
// XChaCha20-Poly1305 uses 24-byte nonces: 8-byte prefix + 8-byte sequence + 8-byte random.
func (ns *NonceSequencer) Next() []byte {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.sequence++

	nonce := make([]byte, 24)
	copy(nonce[0:8], ns.prefix[:])

	// Encode sequence as big-endian.
	nonce[8] = byte(ns.sequence >> 56)
	nonce[9] = byte(ns.sequence >> 48)
	nonce[10] = byte(ns.sequence >> 40)
	nonce[11] = byte(ns.sequence >> 32)
	nonce[12] = byte(ns.sequence >> 24)
	nonce[13] = byte(ns.sequence >> 16)
	nonce[14] = byte(ns.sequence >> 8)
	nonce[15] = byte(ns.sequence)

	// Add randomness for additional unpredictability.
	rand.Read(nonce[16:24])

	return nonce
}

// Sequence returns the current sequence number.
func (ns *NonceSequencer) Sequence() uint64 {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	return ns.sequence
}

// ReplayDetector tracks seen nonces to detect replays.
type ReplayDetector struct {
	mu      sync.RWMutex
	window  uint64          // Window size for tracking.
	minSeq  uint64          // Minimum accepted sequence.
	seen    map[uint64]bool // Seen sequences in current window.
	maxSeen uint64          // Maximum seen sequence.
}

// NewReplayDetector creates a new replay detector with the given window size.
func NewReplayDetector(windowSize uint64) *ReplayDetector {
	return &ReplayDetector{
		window: windowSize,
		seen:   make(map[uint64]bool),
	}
}

// Check checks if a sequence number is valid (not replayed, not too old).
// Returns true if the sequence is acceptable.
func (rd *ReplayDetector) Check(seq uint64) bool {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.isSequenceTooOld(seq) || rd.isSequenceAlreadySeen(seq) {
		return false
	}

	rd.recordSequence(seq)
	rd.updateMaxSeenAndCleanup(seq)
	return true
}

// isSequenceTooOld checks if sequence is below minimum.
func (rd *ReplayDetector) isSequenceTooOld(seq uint64) bool {
	return seq < rd.minSeq
}

// isSequenceAlreadySeen checks if sequence was previously recorded.
func (rd *ReplayDetector) isSequenceAlreadySeen(seq uint64) bool {
	return rd.seen[seq]
}

// recordSequence marks sequence as seen.
func (rd *ReplayDetector) recordSequence(seq uint64) {
	rd.seen[seq] = true
}

// updateMaxSeenAndCleanup updates max seen and slides window forward.
func (rd *ReplayDetector) updateMaxSeenAndCleanup(seq uint64) {
	if seq <= rd.maxSeen {
		return
	}
	rd.maxSeen = seq
	if rd.maxSeen > rd.window {
		rd.slideWindowForward()
	}
}

// slideWindowForward advances the replay window and prunes old entries.
func (rd *ReplayDetector) slideWindowForward() {
	newMin := rd.maxSeen - rd.window
	if newMin <= rd.minSeq {
		return
	}
	for s := range rd.seen {
		if s < newMin {
			delete(rd.seen, s)
		}
	}
	rd.minSeq = newMin
}

// MaxSeen returns the maximum seen sequence number.
func (rd *ReplayDetector) MaxSeen() uint64 {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.maxSeen
}

// Circuit represents a three-hop onion circuit.
type Circuit struct {
	mu              sync.RWMutex
	circuitID       [16]byte // Random circuit identifier.
	hops            [CircuitLength]*RelayInfo
	sharedKeys      [CircuitLength][32]byte
	createdAt       time.Time
	closed          bool
	onTeardown      TeardownFunc                   // Optional callback for cleanup notification.
	destroySent     bool                           // Whether DESTROY cells have been sent.
	nonceSeq        *NonceSequencer                // Nonce sequencer for outgoing packets.
	replayDetectors [CircuitLength]*ReplayDetector // Replay detection per hop.
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
	defer b.recordBuildDuration(time.Now())

	circuitID, err := b.generateCircuitID()
	if err != nil {
		return nil, err
	}

	circuit := b.initializeCircuit(circuitID, relays)

	if err := b.performKeyAgreements(circuit, relays); err != nil {
		return nil, err
	}

	return circuit, nil
}

// recordBuildDuration tracks circuit build time for metrics.
func (b *Beacon) recordBuildDuration(start time.Time) {
	duration := time.Since(start).Seconds()
	metrics.ShroudCircuitBuildDurationSeconds.Observe(duration)
}

// generateCircuitID creates a random 16-byte circuit identifier.
func (b *Beacon) generateCircuitID() ([16]byte, error) {
	var circuitID [16]byte
	if _, err := rand.Read(circuitID[:]); err != nil {
		return circuitID, err
	}
	return circuitID, nil
}

// initializeCircuit creates a Circuit with replay detectors.
func (b *Beacon) initializeCircuit(circuitID [16]byte, relays [CircuitLength]*RelayInfo) *Circuit {
	const replayWindowSize = 1000

	circuit := &Circuit{
		circuitID: circuitID,
		hops:      relays,
		createdAt: time.Now(),
		nonceSeq:  NewNonceSequencer(),
	}

	for i := 0; i < CircuitLength; i++ {
		circuit.replayDetectors[i] = NewReplayDetector(replayWindowSize)
	}

	return circuit
}

// performKeyAgreements establishes shared keys with each hop.
func (b *Beacon) performKeyAgreements(circuit *Circuit, relays [CircuitLength]*RelayInfo) error {
	for i, relay := range relays {
		if relay == nil {
			return ErrRelayNotFound
		}

		sharedKey := b.deriveHopKey(relay, i)
		copy(circuit.sharedKeys[i][:], sharedKey[:32])
		b.zeroSensitiveData(sharedKey)
	}
	return nil
}

// deriveHopKey performs X25519 key agreement and derives the hop encryption key.
func (b *Beacon) deriveHopKey(relay *RelayInfo, hopIndex int) []byte {
	var shared [32]byte
	curve25519.ScalarMult(&shared, &b.secretKey, &relay.PublicKey)

	h := blake3.New()
	h.Write(shared[:])
	h.Write([]byte("shroud-hop-key"))
	h.Write([]byte{byte(hopIndex)})
	key := h.Sum(nil)

	b.zeroSensitiveData(shared[:])
	return key
}

// zeroSensitiveData overwrites key material before GC.
func (b *Beacon) zeroSensitiveData(data []byte) {
	for j := range data {
		data[j] = 0
	}
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
	// Use sequenced nonces for replay protection.
	for i := CircuitLength - 1; i >= 0; i-- {
		cipher, err := chacha20poly1305.NewX(c.sharedKeys[i][:])
		if err != nil {
			return nil, err
		}

		// Generate sequenced nonce for this hop.
		nonce := c.nonceSeq.Next()

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

// DecryptLayerWithReplayCheck decrypts and checks for replay attacks.
// The hopIndex specifies which hop's replay detector to use.
func (c *Circuit) DecryptLayerWithReplayCheck(data []byte, hopIndex int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.validateDecryptionRequest(hopIndex); err != nil {
		return nil, err
	}

	cipher, err := chacha20poly1305.NewX(c.sharedKeys[hopIndex][:])
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	nonce, ciphertext, err := splitNonceAndCiphertext(data, cipher.NonceSize())
	if err != nil {
		return nil, err
	}

	seq := extractSequenceFromNonce(nonce)
	if !c.replayDetectors[hopIndex].Check(seq) {
		return nil, ErrReplayDetected
	}

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// validateDecryptionRequest validates circuit state and hop index.
func (c *Circuit) validateDecryptionRequest(hopIndex int) error {
	if c.closed {
		return ErrCircuitClosed
	}
	if hopIndex < 0 || hopIndex >= CircuitLength {
		return ErrRelayNotFound
	}
	return nil
}

// splitNonceAndCiphertext splits data into nonce and ciphertext components.
func splitNonceAndCiphertext(data []byte, nonceSize int) ([]byte, []byte, error) {
	if len(data) < nonceSize {
		return nil, nil, ErrInvalidPacket
	}
	return data[:nonceSize], data[nonceSize:], nil
}

// extractSequenceFromNonce extracts the 64-bit sequence number from nonce bytes 8-15.
func extractSequenceFromNonce(nonce []byte) uint64 {
	return uint64(nonce[8])<<56 | uint64(nonce[9])<<48 |
		uint64(nonce[10])<<40 | uint64(nonce[11])<<32 |
		uint64(nonce[12])<<24 | uint64(nonce[13])<<16 |
		uint64(nonce[14])<<8 | uint64(nonce[15])
}

// NonceSequence returns the current nonce sequence for this circuit.
func (c *Circuit) NonceSequence() uint64 {
	return c.nonceSeq.Sequence()
}

// ReplayDetectorMaxSeen returns the max seen sequence for a hop.
func (c *Circuit) ReplayDetectorMaxSeen(hopIndex int) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if hopIndex < 0 || hopIndex >= CircuitLength {
		return 0
	}
	return c.replayDetectors[hopIndex].MaxSeen()
}

// IsExpired returns true if the circuit should be rotated.
func (c *Circuit) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.createdAt) > CircuitRotationInterval || c.closed
}

// Close closes the circuit and zeroes key material.
// If a teardown callback is set, it is invoked before key zeroing.
func (c *Circuit) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return // Already closed.
	}

	c.closed = true

	// Invoke teardown callback if set and DESTROY not yet sent.
	if c.onTeardown != nil && !c.destroySent {
		c.destroySent = true
		peerIDs := c.getRelayPeerIDs()
		// Call teardown in goroutine to avoid holding lock.
		go c.onTeardown(c.circuitID[:], peerIDs)
	}

	// Zero shared keys.
	for i := range c.sharedKeys {
		for j := range c.sharedKeys[i] {
			c.sharedKeys[i][j] = 0
		}
	}
}

// Teardown sends DESTROY cells to all relays and closes the circuit.
// This provides a clean circuit destruction mechanism.
func (c *Circuit) Teardown() {
	c.Close()
}

// SetOnTeardown sets the callback invoked when the circuit is torn down.
func (c *Circuit) SetOnTeardown(f TeardownFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onTeardown = f
}

// getRelayPeerIDs returns the peer IDs of all hops in the circuit.
// Must be called with lock held.
func (c *Circuit) getRelayPeerIDs() []string {
	var peerIDs []string
	for _, hop := range c.hops {
		if hop != nil {
			peerIDs = append(peerIDs, hop.PeerID)
		}
	}
	return peerIDs
}

// CircuitID returns the unique identifier for this circuit.
func (c *Circuit) CircuitID() [16]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.circuitID
}

// Hops returns the relay info for each hop in the circuit.
func (c *Circuit) Hops() [CircuitLength]*RelayInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hops
}

// IsClosed returns true if the circuit has been closed.
func (c *Circuit) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// CreatedAt returns when the circuit was created.
func (c *Circuit) CreatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.createdAt
}

// CreateDestroyCell creates a DESTROY cell for this circuit.
// This cell should be sent to each relay to clean up circuit state.
func (c *Circuit) CreateDestroyCell() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// DESTROY cell format: 1 byte type + 16 byte circuit ID + padding.
	cell := make([]byte, FixedPacketSize)
	cell[0] = 0x04 // ONION_CELL_TYPE_DESTROY
	copy(cell[1:17], c.circuitID[:])

	// Fill remaining with random padding.
	rand.Read(cell[17:])

	return cell, nil
}

// EncryptDestroyForHop encrypts a DESTROY cell for a specific hop.
// hopIndex is 0-based (0 for entry, 1 for middle, 2 for exit).
func (c *Circuit) EncryptDestroyForHop(hopIndex int) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if hopIndex < 0 || hopIndex >= CircuitLength {
		return nil, ErrRelayNotFound
	}

	// Create base DESTROY cell.
	cell := make([]byte, FixedPacketSize)
	cell[0] = 0x04 // ONION_CELL_TYPE_DESTROY
	copy(cell[1:17], c.circuitID[:])
	rand.Read(cell[17:])

	// Encrypt layers for all hops up to and including this one.
	// Entry hop (0) needs 1 layer, middle hop (1) needs 2, exit hop (2) needs 3.
	data := cell
	for i := hopIndex; i >= 0; i-- {
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
// Per SECURITY_PRIVACY.md, maintains dual active circuits for resilience.
type CircuitManager struct {
	mu                sync.RWMutex
	beacon            *Beacon
	primary           *Circuit // Primary active circuit.
	backup            *Circuit // Backup circuit for failover.
	exclude           []string // Peer IDs to exclude from circuit selection.
	rotationCount     uint64   // Track number of rotations.
	lastRotation      time.Time
	lastRebuild       time.Time                      // Last rebuild attempt time.
	failureTracker    *RelayFailureTracker           // Track relay failures.
	onRotation        func(primary, backup *Circuit) // Optional callback on rotation.
	onError           func(*CircuitError)            // Optional callback on errors.
	rebuildAttempts   uint64                         // Count of rebuild attempts.
	coverSender       CoverTrafficSender             // Callback for sending cover traffic.
	coverConfig       CoverTrafficConfig             // Cover traffic configuration.
	coverTrafficCount uint64                         // Count of cover packets sent.
}

// NewCircuitManager creates a circuit manager with dual circuit support.
func NewCircuitManager(beacon *Beacon, excludePeers []string) *CircuitManager {
	return &CircuitManager{
		beacon:         beacon,
		exclude:        excludePeers,
		failureTracker: NewRelayFailureTracker(),
	}
}

// GetCircuit returns the primary circuit, building a new one if needed.
func (m *CircuitManager) GetCircuit() (*Circuit, error) {
	m.mu.RLock()
	if m.primary != nil && !m.primary.IsExpired() {
		c := m.primary
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	return m.RotateCircuit()
}

// GetBackupCircuit returns the backup circuit.
func (m *CircuitManager) GetBackupCircuit() *Circuit {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.backup
}

// GetPrimaryCircuit returns the primary circuit.
func (m *CircuitManager) GetPrimaryCircuit() *Circuit {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.primary
}

// FailoverToBackup promotes the backup circuit to primary.
// Returns the new primary circuit and an error if no backup is available.
func (m *CircuitManager) FailoverToBackup() (*Circuit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backup == nil || m.backup.IsExpired() {
		return nil, ErrCircuitClosed
	}

	// Close the failed primary.
	if m.primary != nil {
		m.primary.Close()
	}

	// Promote backup to primary.
	m.primary = m.backup
	m.backup = nil

	// Try to build a new backup circuit asynchronously.
	go m.buildBackupCircuitAsync()

	return m.primary, nil
}

// RotateCircuit builds new primary and backup circuits.
// The old primary becomes the new backup, and a fresh circuit becomes primary.
func (m *CircuitManager) RotateCircuit() (*Circuit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newPrimary, err := m.buildNewPrimaryCircuit()
	if err != nil {
		return nil, err
	}

	m.rotateCircuits(newPrimary)
	m.ensureBackupCircuit()
	m.notifyRotation()

	return newPrimary, nil
}

// buildNewPrimaryCircuit selects relays and constructs a new circuit.
func (m *CircuitManager) buildNewPrimaryCircuit() (*Circuit, error) {
	relays, err := m.beacon.SelectRelays(m.exclude)
	if err != nil {
		return nil, err
	}
	return m.beacon.BuildCircuit(relays)
}

// rotateCircuits promotes new circuit to primary, demotes old primary to backup, and closes old backup.
func (m *CircuitManager) rotateCircuits(newPrimary *Circuit) {
	if m.backup != nil {
		m.backup.Close()
	}

	if m.primary != nil && !m.primary.IsExpired() {
		m.backup = m.primary
	} else if m.primary != nil {
		m.primary.Close()
		m.backup = nil
	}

	m.primary = newPrimary
	m.rotationCount++
	m.lastRotation = time.Now()
}

// ensureBackupCircuit builds a backup circuit if none exists.
func (m *CircuitManager) ensureBackupCircuit() {
	if m.backup == nil {
		m.buildBackupCircuitLocked()
	}
}

// notifyRotation invokes the rotation callback if set.
func (m *CircuitManager) notifyRotation() {
	if m.onRotation != nil {
		go m.onRotation(m.primary, m.backup)
	}
}

// buildBackupCircuitLocked builds a backup circuit. Must be called with lock held.
func (m *CircuitManager) buildBackupCircuitLocked() {
	excludeForBackup := m.buildExcludeList()
	relays := m.selectRelaysForBackup(excludeForBackup)
	if relays == [CircuitLength]*RelayInfo{} {
		return
	}

	backup, err := m.beacon.BuildCircuit(relays)
	if err != nil {
		return
	}

	m.backup = backup
}

// buildExcludeList creates exclusion list including primary circuit hops.
func (m *CircuitManager) buildExcludeList() []string {
	exclude := m.exclude
	if m.primary != nil {
		for _, hop := range m.primary.hops {
			if hop != nil {
				exclude = append(exclude, hop.PeerID)
			}
		}
	}
	return exclude
}

// selectRelaysForBackup attempts to select diverse relays, falling back if needed.
func (m *CircuitManager) selectRelaysForBackup(excludeForBackup []string) [CircuitLength]*RelayInfo {
	relays, err := m.beacon.SelectRelays(excludeForBackup)
	if err != nil {
		relays, err = m.beacon.SelectRelays(m.exclude)
		if err != nil {
			return [CircuitLength]*RelayInfo{}
		}
	}
	return relays
}

// buildBackupCircuitAsync builds a backup circuit asynchronously.
func (m *CircuitManager) buildBackupCircuitAsync() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backup != nil && !m.backup.IsExpired() {
		return // Already have a valid backup.
	}

	m.buildBackupCircuitLocked()
}

// BuildInitialCircuitAsync attempts to build an initial circuit at startup
// without blocking. Returns a channel that will receive the circuit result.
// Per AUDIT.md, prevents app hang when bootstrap peers are slow or unavailable.
func (m *CircuitManager) BuildInitialCircuitAsync(ctx context.Context) <-chan *CircuitResult {
	return m.RotateCircuitAsync(ctx)
}

// StartRotation runs periodic circuit rotation.
func (m *CircuitManager) StartRotation(ctx context.Context) {
	ticker := time.NewTicker(CircuitRotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.closeCircuits()
			return
		case <-ticker.C:
			// Use async rotation to avoid blocking the goroutine
			go m.RotateCircuitAsync(ctx)
		}
	}
}

// RotateCircuitAsync builds new circuits asynchronously without blocking.
// Returns a channel that will receive the new primary circuit or an error.
// Per AUDIT.md, this prevents startup blocking when bootstrap is slow.
func (m *CircuitManager) RotateCircuitAsync(ctx context.Context) <-chan *CircuitResult {
	resultCh := make(chan *CircuitResult, 1)

	go func() {
		// Create timeout context (30 seconds per AUDIT.md)
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Run circuit building with timeout
		done := make(chan *CircuitResult, 1)
		go func() {
			circuit, err := m.RotateCircuit()
			done <- &CircuitResult{Circuit: circuit, Err: err}
		}()

		select {
		case result := <-done:
			resultCh <- result
		case <-timeoutCtx.Done():
			// Timeout or context cancelled
			resultCh <- &CircuitResult{
				Circuit: nil,
				Err:     errors.New("circuit construction timeout (30s)"),
			}
		}
		close(resultCh)
	}()

	return resultCh
}

// CircuitResult holds the result of an async circuit build operation.
type CircuitResult struct {
	Circuit *Circuit
	Err     error
}

// SetOnRotation sets a callback that's invoked after each rotation.
func (m *CircuitManager) SetOnRotation(callback func(primary, backup *Circuit)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRotation = callback
}

// RotationCount returns the number of circuit rotations performed.
func (m *CircuitManager) RotationCount() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rotationCount
}

// LastRotation returns the time of the last rotation.
func (m *CircuitManager) LastRotation() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastRotation
}

// HasBackup returns true if a backup circuit is available.
func (m *CircuitManager) HasBackup() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.backup != nil && !m.backup.IsExpired()
}

// closeCircuits safely closes both primary and backup circuits.
func (m *CircuitManager) closeCircuits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.primary != nil {
		m.primary.Close()
	}
	if m.backup != nil {
		m.backup.Close()
	}
}

// closeCircuit safely closes the current circuit (legacy method for compatibility).
func (m *CircuitManager) closeCircuit() {
	m.closeCircuits()
}

// SetOnError sets a callback for circuit errors.
func (m *CircuitManager) SetOnError(callback func(*CircuitError)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onError = callback
}

// ReportRelayFailure reports a relay failure for error tracking.
// This triggers failover if the primary circuit uses the failed relay.
func (m *CircuitManager) ReportRelayFailure(relayID string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track the failure.
	exceeded := m.failureTracker.RecordFailure(relayID)

	// Create error for logging/callback.
	var circuitID [16]byte
	if m.primary != nil {
		circuitID = m.primary.circuitID
	}

	circuitErr := NewCircuitError(ErrRelayFailure, relayID, circuitID, !exceeded)

	// Notify error callback if set.
	if m.onError != nil {
		go m.onError(circuitErr)
	}

	// Check if primary circuit uses this relay.
	if m.primary != nil && m.circuitUsesRelay(m.primary, relayID) {
		return m.handlePrimaryFailure()
	}

	// Check if backup uses this relay - rebuild backup.
	if m.backup != nil && m.circuitUsesRelay(m.backup, relayID) {
		m.backup.Close()
		m.backup = nil
		go m.buildBackupCircuitAsync()
	}

	return nil
}

// circuitUsesRelay checks if a circuit includes the specified relay.
// Must be called with lock held.
func (m *CircuitManager) circuitUsesRelay(c *Circuit, relayID string) bool {
	if c == nil {
		return false
	}
	for _, hop := range c.hops {
		if hop != nil && hop.PeerID == relayID {
			return true
		}
	}
	return false
}

// handlePrimaryFailure handles primary circuit failure.
// Must be called with lock held.
func (m *CircuitManager) handlePrimaryFailure() error {
	// Close failed primary.
	if m.primary != nil {
		m.primary.Close()
	}

	// Try failover to backup.
	if m.backup != nil && !m.backup.IsExpired() {
		m.primary = m.backup
		m.backup = nil
		go m.buildBackupCircuitAsync()
		return nil
	}

	// No backup available, need to rebuild.
	m.primary = nil
	return m.rebuildCircuitsLocked()
}

// rebuildCircuitsLocked rebuilds circuits after total failure.
// Must be called with lock held.
func (m *CircuitManager) rebuildCircuitsLocked() error {
	// Check backoff.
	if time.Since(m.lastRebuild) < CircuitRebuildBackoff {
		return ErrAllCircuitsFailed
	}

	m.lastRebuild = time.Now()
	m.rebuildAttempts++

	// Get penalized relays to exclude.
	penalized := m.failureTracker.PenalizedRelays()
	exclude := append(m.exclude, penalized...)

	// Try to build primary.
	relays, err := m.beacon.SelectRelays(exclude)
	if err != nil {
		return err
	}

	primary, err := m.beacon.BuildCircuit(relays)
	if err != nil {
		return err
	}

	m.primary = primary
	m.buildBackupCircuitLocked()

	return nil
}

// RecoverFromError attempts to recover from a circuit error.
// This is the main entry point for error recovery.
func (m *CircuitManager) RecoverFromError(err *CircuitError) error {
	if err == nil {
		return nil
	}

	// For relay failures, use the specific handler.
	if err.RelayID != "" {
		return m.ReportRelayFailure(err.RelayID, err.Err)
	}

	// For other errors, attempt circuit rebuild.
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.rebuildCircuitsLocked()
}

// GetCircuitOrRecover gets the primary circuit, attempting recovery if needed.
// This is a resilient version of GetCircuit that handles errors automatically.
func (m *CircuitManager) GetCircuitOrRecover() (*Circuit, error) {
	m.mu.RLock()
	if m.primary != nil && !m.primary.IsExpired() {
		c := m.primary
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	// Need to build or recover.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if m.primary != nil && !m.primary.IsExpired() {
		return m.primary, nil
	}

	// Try failover first.
	if m.backup != nil && !m.backup.IsExpired() {
		m.primary = m.backup
		m.backup = nil
		go m.buildBackupCircuitAsync()
		return m.primary, nil
	}

	// Need full rebuild.
	err := m.rebuildCircuitsLocked()
	if err != nil {
		return nil, err
	}

	return m.primary, nil
}

// FailureTracker returns the relay failure tracker.
func (m *CircuitManager) FailureTracker() *RelayFailureTracker {
	return m.failureTracker
}

// RebuildAttempts returns the number of rebuild attempts.
func (m *CircuitManager) RebuildAttempts() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rebuildAttempts
}

// CircuitHealth represents the health status of the circuit manager.
type CircuitHealth struct {
	HasPrimary       bool
	HasBackup        bool
	PrimaryExpired   bool
	BackupExpired    bool
	RotationCount    uint64
	RebuildAttempts  uint64
	PenalizedRelays  int
	LastRotation     time.Time
	LastRebuild      time.Time
	PrimaryCreatedAt time.Time
	BackupCreatedAt  time.Time
	CoverTrafficSent uint64
}

// Health returns the current health status of the circuit manager.
func (m *CircuitManager) Health() CircuitHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := CircuitHealth{
		HasPrimary:       m.primary != nil,
		HasBackup:        m.backup != nil,
		RotationCount:    m.rotationCount,
		RebuildAttempts:  m.rebuildAttempts,
		PenalizedRelays:  len(m.failureTracker.PenalizedRelays()),
		LastRotation:     m.lastRotation,
		LastRebuild:      m.lastRebuild,
		CoverTrafficSent: atomic.LoadUint64(&m.coverTrafficCount),
	}

	if m.primary != nil {
		health.PrimaryExpired = m.primary.IsExpired()
		health.PrimaryCreatedAt = m.primary.createdAt
	}

	if m.backup != nil {
		health.BackupExpired = m.backup.IsExpired()
		health.BackupCreatedAt = m.backup.createdAt
	}

	return health
}

// CoverTrafficSender is a callback for sending cover traffic through circuits.
// The sender receives the encrypted packet and the entry relay's peer ID.
type CoverTrafficSender func(peerID string, data []byte) error

// CoverTrafficConfig holds configuration for cover traffic generation.
type CoverTrafficConfig struct {
	// Rate is the interval between cover packets (default: DummyPacketRate = 500ms).
	Rate time.Duration
	// Enabled controls whether cover traffic is active.
	Enabled bool
}

// DefaultCoverTrafficConfig returns the default cover traffic configuration.
// Per ROADMAP: constant-rate dummy packets (2 per second).
func DefaultCoverTrafficConfig() CoverTrafficConfig {
	return CoverTrafficConfig{
		Rate:    DummyPacketRate, // 500ms = 2 per second.
		Enabled: true,
	}
}

// SetCoverTrafficSender sets the callback for sending cover traffic.
func (m *CircuitManager) SetCoverTrafficSender(sender CoverTrafficSender) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coverSender = sender
}

// SetCoverTrafficConfig updates the cover traffic configuration.
func (m *CircuitManager) SetCoverTrafficConfig(config CoverTrafficConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coverConfig = config
}

// StartCoverTraffic begins constant-rate cover traffic generation.
// Per SHADOW_GRADIENT.md §Traffic Padding, cover traffic maintains constant rate
// to prevent traffic analysis attacks.
func (m *CircuitManager) StartCoverTraffic(ctx context.Context) {
	m.mu.RLock()
	config := m.coverConfig
	m.mu.RUnlock()

	// Use default rate if not configured.
	rate := config.Rate
	if rate == 0 {
		rate = DummyPacketRate
	}

	ticker := time.NewTicker(rate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sendCoverPacket()
		}
	}
}

// sendCoverPacket sends a single cover traffic packet through the primary circuit.
func (m *CircuitManager) sendCoverPacket() {
	m.mu.RLock()
	config := m.coverConfig
	sender := m.coverSender
	primary := m.primary
	m.mu.RUnlock()

	// Check if cover traffic is enabled.
	if !config.Enabled || sender == nil {
		return
	}

	// Need a valid primary circuit.
	if primary == nil || primary.IsClosed() {
		return
	}

	// Generate cover packet: random payload marked as dummy (0x00 first byte).
	payload := make([]byte, FixedPacketSize-100) // Leave room for encryption overhead.
	rand.Read(payload)
	payload[0] = 0x00 // Mark as dummy/cover traffic.

	// Encrypt through the circuit.
	encrypted, err := primary.Encrypt(payload)
	if err != nil {
		return
	}

	// Get entry relay's peer ID.
	hops := primary.Hops()
	if hops[0] == nil {
		return
	}
	entryPeerID := hops[0].PeerID

	// Send the cover packet.
	if err := sender(entryPeerID, encrypted); err != nil {
		return
	}

	atomic.AddUint64(&m.coverTrafficCount, 1)
}

// CoverTrafficCount returns the number of cover packets sent.
func (m *CircuitManager) CoverTrafficCount() uint64 {
	return atomic.LoadUint64(&m.coverTrafficCount)
}

// IsCoverPacket returns true if the packet is a cover/dummy packet.
// Cover packets have 0x00 as their first byte after decryption.
func IsCoverPacket(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return data[0] == 0x00
}

// MessageType identifies the type of message in Shroud packets.
type MessageType byte

const (
	// MessageTypeDummy is a cover traffic/dummy packet.
	MessageTypeDummy MessageType = 0x00
	// MessageTypeData is an actual data message.
	MessageTypeData MessageType = 0x01
	// MessageTypeControl is a control message.
	MessageTypeControl MessageType = 0x02
)

// Message represents a message to be sent through a Shroud circuit.
type Message struct {
	Type    MessageType
	Dest    [32]byte // Destination public key (for routing at exit).
	Payload []byte   // Actual message content.
}

// MessageHeader is the fixed-size header prepended to all messages.
// Per SECURITY_PRIVACY.md, all messages have uniform size.
type MessageHeader struct {
	Type    MessageType // 1 byte
	DestLen uint8       // 1 byte (0 for broadcast, 32 for directed)
	DataLen uint16      // 2 bytes, big-endian
}

const messageHeaderSize = 4 // 1 + 1 + 2 bytes

// EncodeMessage encodes a message with header for transmission.
func EncodeMessage(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}

	// Calculate required size.
	headerSize := messageHeaderSize
	destSize := 0
	if msg.Type == MessageTypeData && msg.Dest != [32]byte{} {
		destSize = 32
	}

	totalSize := headerSize + destSize + len(msg.Payload)

	// Check size limits.
	if totalSize > FixedPacketSize-100 { // Leave room for encryption overhead.
		return nil, errors.New("message too large")
	}

	// Allocate buffer.
	buf := make([]byte, totalSize)

	// Write header.
	buf[0] = byte(msg.Type)
	if destSize > 0 {
		buf[1] = 32
		copy(buf[headerSize:headerSize+32], msg.Dest[:])
	} else {
		buf[1] = 0
	}
	buf[2] = byte(len(msg.Payload) >> 8)
	buf[3] = byte(len(msg.Payload) & 0xFF)

	// Write payload.
	copy(buf[headerSize+destSize:], msg.Payload)

	return buf, nil
}

// DecodeMessage decodes a message from received data.
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < messageHeaderSize {
		return nil, errors.New("message too short")
	}

	msg := &Message{
		Type: MessageType(data[0]),
	}

	destLen := data[1]
	dataLen := int(data[2])<<8 | int(data[3])

	// Validate sizes.
	expectedLen := messageHeaderSize + int(destLen) + dataLen
	if len(data) < expectedLen {
		return nil, errors.New("message truncated")
	}

	// Read destination if present.
	offset := messageHeaderSize
	if destLen == 32 {
		copy(msg.Dest[:], data[offset:offset+32])
		offset += 32
	}

	// Read payload.
	msg.Payload = make([]byte, dataLen)
	copy(msg.Payload, data[offset:offset+dataLen])

	return msg, nil
}

// MessageSender handles sending messages through Shroud circuits.
type MessageSender struct {
	mu      sync.RWMutex
	manager *CircuitManager
	sender  func(peerID string, data []byte) error
	stats   MessageSenderStats
	onError func(error)
}

// MessageSenderStats tracks message sending statistics.
type MessageSenderStats struct {
	MessagesSent    uint64
	MessagesDropped uint64
	BytesSent       uint64
	LastSendTime    time.Time
	LastError       error
	LastErrorTime   time.Time
}

// NewMessageSender creates a new message sender.
func NewMessageSender(manager *CircuitManager, sender func(peerID string, data []byte) error) *MessageSender {
	return &MessageSender{
		manager: manager,
		sender:  sender,
	}
}

// Send sends a message through the Shroud circuit.
// This is the main entry point for end-to-end message delivery.
func (s *MessageSender) Send(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get circuit.
	circuit, err := s.manager.GetCircuitOrRecover()
	if err != nil {
		s.handleError(err)
		return err
	}

	// Encode message.
	encoded, err := EncodeMessage(msg)
	if err != nil {
		s.handleError(err)
		return err
	}

	// Encrypt through circuit.
	encrypted, err := circuit.Encrypt(encoded)
	if err != nil {
		s.handleError(err)
		return err
	}

	// Get entry relay.
	hops := circuit.Hops()
	if hops[0] == nil {
		err := errors.New("no entry relay in circuit")
		s.handleError(err)
		return err
	}

	// Send to entry relay.
	if err := s.sender(hops[0].PeerID, encrypted); err != nil {
		s.handleError(err)
		atomic.AddUint64(&s.stats.MessagesDropped, 1)
		return err
	}

	// Update stats.
	atomic.AddUint64(&s.stats.MessagesSent, 1)
	atomic.AddUint64(&s.stats.BytesSent, uint64(len(encrypted)))
	s.stats.LastSendTime = time.Now()

	return nil
}

// SendTo sends a message to a specific destination through the Shroud circuit.
func (s *MessageSender) SendTo(dest [32]byte, payload []byte) error {
	msg := &Message{
		Type:    MessageTypeData,
		Dest:    dest,
		Payload: payload,
	}
	return s.Send(msg)
}

// Broadcast sends a message without a specific destination.
func (s *MessageSender) Broadcast(payload []byte) error {
	msg := &Message{
		Type:    MessageTypeData,
		Payload: payload,
	}
	return s.Send(msg)
}

// handleError processes and tracks errors.
func (s *MessageSender) handleError(err error) {
	s.stats.LastError = err
	s.stats.LastErrorTime = time.Now()
	if s.onError != nil {
		go s.onError(err)
	}
}

// Stats returns current sender statistics.
func (s *MessageSender) Stats() MessageSenderStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return MessageSenderStats{
		MessagesSent:    atomic.LoadUint64(&s.stats.MessagesSent),
		MessagesDropped: atomic.LoadUint64(&s.stats.MessagesDropped),
		BytesSent:       atomic.LoadUint64(&s.stats.BytesSent),
		LastSendTime:    s.stats.LastSendTime,
		LastError:       s.stats.LastError,
		LastErrorTime:   s.stats.LastErrorTime,
	}
}

// SetOnError sets a callback for send errors.
func (s *MessageSender) SetOnError(callback func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = callback
}

// MessageReceiver handles receiving and processing messages from Shroud circuits.
type MessageReceiver struct {
	mu       sync.RWMutex
	handlers map[MessageType]MessageHandler
	stats    MessageReceiverStats
}

// MessageHandler is called when a message is received.
type MessageHandler func(msg *Message) error

// MessageReceiverStats tracks message receiving statistics.
type MessageReceiverStats struct {
	MessagesReceived uint64
	MessagesDropped  uint64
	BytesReceived    uint64
	LastReceiveTime  time.Time
}

// NewMessageReceiver creates a new message receiver.
func NewMessageReceiver() *MessageReceiver {
	return &MessageReceiver{
		handlers: make(map[MessageType]MessageHandler),
	}
}

// RegisterHandler registers a handler for a specific message type.
func (r *MessageReceiver) RegisterHandler(msgType MessageType, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[msgType] = handler
}

// HandlePacket processes a received packet, decoding and dispatching the message.
// This should be called after the packet has been decrypted by all circuit layers.
func (r *MessageReceiver) HandlePacket(data []byte) error {
	// Strip padding.
	data = unpadFromSize(data)

	// Decode message.
	msg, err := DecodeMessage(data)
	if err != nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return err
	}

	// Skip dummy packets.
	if msg.Type == MessageTypeDummy {
		return nil
	}

	r.mu.RLock()
	handler, ok := r.handlers[msg.Type]
	r.mu.RUnlock()

	if !ok {
		// No handler registered, drop.
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return nil
	}

	// Update stats.
	atomic.AddUint64(&r.stats.MessagesReceived, 1)
	atomic.AddUint64(&r.stats.BytesReceived, uint64(len(msg.Payload)))

	r.mu.Lock()
	r.stats.LastReceiveTime = time.Now()
	r.mu.Unlock()

	return handler(msg)
}

// Stats returns current receiver statistics.
func (r *MessageReceiver) Stats() MessageReceiverStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return MessageReceiverStats{
		MessagesReceived: atomic.LoadUint64(&r.stats.MessagesReceived),
		MessagesDropped:  atomic.LoadUint64(&r.stats.MessagesDropped),
		BytesReceived:    atomic.LoadUint64(&r.stats.BytesReceived),
		LastReceiveTime:  r.stats.LastReceiveTime,
	}
}

// EndToEndDelivery wraps circuit manager with message send/receive capabilities.
// This is the high-level API for Shroud message delivery.
type EndToEndDelivery struct {
	manager  *CircuitManager
	sender   *MessageSender
	receiver *MessageReceiver
}

// NewEndToEndDelivery creates an end-to-end delivery system.
func NewEndToEndDelivery(manager *CircuitManager, networkSender func(peerID string, data []byte) error) *EndToEndDelivery {
	return &EndToEndDelivery{
		manager:  manager,
		sender:   NewMessageSender(manager, networkSender),
		receiver: NewMessageReceiver(),
	}
}

// Send sends a message through the Shroud circuit.
func (e *EndToEndDelivery) Send(msg *Message) error {
	return e.sender.Send(msg)
}

// SendTo sends a message to a specific destination.
func (e *EndToEndDelivery) SendTo(dest [32]byte, payload []byte) error {
	return e.sender.SendTo(dest, payload)
}

// Broadcast sends a message to all.
func (e *EndToEndDelivery) Broadcast(payload []byte) error {
	return e.sender.Broadcast(payload)
}

// HandleIncoming processes an incoming packet.
func (e *EndToEndDelivery) HandleIncoming(data []byte) error {
	return e.receiver.HandlePacket(data)
}

// RegisterHandler registers a handler for a message type.
func (e *EndToEndDelivery) RegisterHandler(msgType MessageType, handler MessageHandler) {
	e.receiver.RegisterHandler(msgType, handler)
}

// Manager returns the circuit manager.
func (e *EndToEndDelivery) Manager() *CircuitManager {
	return e.manager
}

// Sender returns the message sender.
func (e *EndToEndDelivery) Sender() *MessageSender {
	return e.sender
}

// Receiver returns the message receiver.
func (e *EndToEndDelivery) Receiver() *MessageReceiver {
	return e.receiver
}

// BeaconWave constants for relay discovery.
const (
	// BeaconWaveType identifies a Shroud relay advertisement.
	BeaconWaveType byte = 0x08 // Per WAVES.md, Beacon Wave type.

	// BeaconWaveVersion is the current beacon wave protocol version.
	BeaconWaveVersion byte = 1

	// BeaconWaveTTL is the time-to-live for relay advertisements.
	BeaconWaveTTL = 5 * time.Minute

	// BeaconWaveInterval is the default advertisement broadcast interval.
	BeaconWaveInterval = 60 * time.Second

	// BeaconWaveMaxAge is the maximum age before a relay is considered stale.
	BeaconWaveMaxAge = 10 * time.Minute
)

// BeaconWaveError defines beacon wave errors.
var (
	ErrBeaconWaveInvalid    = errors.New("invalid beacon wave")
	ErrBeaconWaveExpired    = errors.New("beacon wave expired")
	ErrBeaconWaveBadVersion = errors.New("unsupported beacon wave version")
)

// BeaconWave represents a Shroud relay advertisement.
// Per ROADMAP line 298: Shroud relay discovery via Beacon Waves on Anonymous Layer.
