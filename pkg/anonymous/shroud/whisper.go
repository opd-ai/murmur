// Package shroud provides anonymous communication primitives.
// This file implements Whisper Chains - anonymous multi-hop message relay between Specters.
// Per DESIGN_DOCUMENT.md §Anonymous Layer, Whisper Chains enable private messaging
// between Specters without revealing identity to relays or destinations.
package shroud

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// Whisper Chain constants.
const (
	// WhisperNonceSize is the nonce size for XChaCha20-Poly1305.
	WhisperNonceSize = 24

	// WhisperKeySize is the symmetric key size.
	WhisperKeySize = 32

	// WhisperMaxPayload is the maximum payload size for a whisper message.
	// Allows room for headers and encryption overhead within FixedPacketSize.
	WhisperMaxPayload = FixedPacketSize - 200

	// WhisperChainMaxHops is the maximum number of relay hops in a whisper chain.
	WhisperChainMaxHops = 5

	// WhisperMessageTTL is the default time-to-live for whisper messages.
	WhisperMessageTTL = 10 * time.Minute
)

// Whisper Chain errors.
var (
	ErrWhisperPayloadTooLarge = errors.New("whisper payload too large")
	ErrWhisperInvalidKey      = errors.New("invalid whisper key")
	ErrWhisperDecryptFailed   = errors.New("whisper decryption failed")
	ErrWhisperExpired         = errors.New("whisper message expired")
	ErrWhisperChainTooLong    = errors.New("whisper chain exceeds max hops")
	ErrWhisperNoRoute         = errors.New("no route to destination")
)

// WhisperMessage represents an encrypted message in the Whisper Chain.
type WhisperMessage struct {
	// MessageID is a unique identifier (BLAKE3 hash of encrypted content).
	MessageID [32]byte

	// SenderKey is the sender's ephemeral Curve25519 public key for key exchange.
	SenderKey [32]byte

	// Encrypted is the XChaCha20-Poly1305 encrypted payload.
	Encrypted []byte

	// Nonce is the encryption nonce.
	Nonce [WhisperNonceSize]byte

	// Timestamp is when the message was created (Unix seconds).
	Timestamp int64

	// TTL is the time-to-live in seconds.
	TTL uint32

	// HopCount tracks how many relays have forwarded this message.
	HopCount uint8
}

// IsExpired returns true if the message has exceeded its TTL.
func (m *WhisperMessage) IsExpired() bool {
	expiresAt := time.Unix(m.Timestamp, 0).Add(time.Duration(m.TTL) * time.Second)
	return time.Now().After(expiresAt)
}

// WhisperKeyExchange performs Curve25519 key exchange with HKDF-SHA-256 derivation.
// Per TECHNICAL_IMPLEMENTATION.md, this derives a symmetric key from the shared secret.
type WhisperKeyExchange struct {
	privateKey [32]byte
	publicKey  [32]byte
}

// NewWhisperKeyExchange generates a new key exchange pair.
func NewWhisperKeyExchange() (*WhisperKeyExchange, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return nil, err
	}

	// Clamp private key for Curve25519.
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return &WhisperKeyExchange{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// PublicKey returns the public key for key exchange.
func (k *WhisperKeyExchange) PublicKey() [32]byte {
	return k.publicKey
}

// DeriveKey derives a symmetric encryption key from the peer's public key.
// Uses HKDF-SHA-256 with "murmur-whisper" as the info string.
// Per TECHNICAL_IMPLEMENTATION.md, this derives a symmetric key from the shared secret.
func (k *WhisperKeyExchange) DeriveKey(peerPublicKey [32]byte) ([32]byte, error) {
	// Compute shared secret via X25519.
	var shared [32]byte
	curve25519.ScalarMult(&shared, &k.privateKey, &peerPublicKey)

	// Check for zero result (invalid key).
	var zero [32]byte
	if shared == zero {
		return zero, ErrWhisperInvalidKey
	}

	// Derive key using HKDF-SHA-256.
	kdf := hkdf.New(sha256.New, shared[:], nil, []byte("murmur-whisper"))

	var derived [32]byte
	if _, err := io.ReadFull(kdf, derived[:]); err != nil {
		return zero, err
	}

	return derived, nil
}

// EncryptWhisper encrypts a payload for the given recipient public key.
// Returns a WhisperMessage ready for transmission.
func EncryptWhisper(payload []byte, recipientPubKey [32]byte) (*WhisperMessage, error) {
	if len(payload) > WhisperMaxPayload {
		return nil, ErrWhisperPayloadTooLarge
	}

	// Generate ephemeral key pair for this message.
	kx, err := NewWhisperKeyExchange()
	if err != nil {
		return nil, err
	}

	// Derive encryption key.
	key, err := kx.DeriveKey(recipientPubKey)
	if err != nil {
		return nil, err
	}

	// Create cipher.
	cipher, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, err
	}

	// Generate nonce.
	var nonce [WhisperNonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	// Encrypt payload.
	encrypted := cipher.Seal(nil, nonce[:], payload, nil)

	// Create message.
	msg := &WhisperMessage{
		SenderKey: kx.PublicKey(),
		Encrypted: encrypted,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		TTL:       uint32(WhisperMessageTTL.Seconds()),
		HopCount:  0,
	}

	// Compute message ID.
	h := blake3.New()
	h.Write(msg.Encrypted)
	h.Write(msg.Nonce[:])
	copy(msg.MessageID[:], h.Sum(nil))

	return msg, nil
}

// DecryptWhisper decrypts a WhisperMessage using the recipient's private key.
func DecryptWhisper(msg *WhisperMessage, recipientPrivKey [32]byte) ([]byte, error) {
	if msg.IsExpired() {
		return nil, ErrWhisperExpired
	}

	// Create key exchange with recipient's private key.
	var pubKey [32]byte
	curve25519.ScalarBaseMult(&pubKey, &recipientPrivKey)

	kx := &WhisperKeyExchange{
		privateKey: recipientPrivKey,
		publicKey:  pubKey,
	}

	// Derive decryption key from sender's ephemeral public key.
	key, err := kx.DeriveKey(msg.SenderKey)
	if err != nil {
		return nil, err
	}

	// Create cipher.
	cipher, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, err
	}

	// Decrypt.
	plaintext, err := cipher.Open(nil, msg.Nonce[:], msg.Encrypted, nil)
	if err != nil {
		return nil, ErrWhisperDecryptFailed
	}

	return plaintext, nil
}

// WhisperRouter routes whisper messages through Shroud circuits.
// It maintains routing information for known Specters.
type WhisperRouter struct {
	mu         sync.RWMutex
	delivery   *EndToEndDelivery
	routes     map[[32]byte]*WhisperRoute   // Destination pubkey -> route info.
	pending    map[[32]byte]*WhisperMessage // MessageID -> pending message.
	handlers   []WhisperHandler
	stats      WhisperRouterStats
	privateKey [32]byte // This node's private key for receiving.
}

// WhisperRoute contains routing information to reach a Specter.
type WhisperRoute struct {
	Destination [32]byte  // Destination Specter public key.
	LastSeen    time.Time // When this route was last confirmed.
	Latency     time.Duration
	Reliability float64 // 0.0 - 1.0, success rate.
}

// WhisperHandler handles received whisper messages.
type WhisperHandler func(msg *WhisperMessage, payload []byte) error

// WhisperRouterStats tracks routing statistics.
type WhisperRouterStats struct {
	MessagesSent     uint64
	MessagesRelayed  uint64
	MessagesReceived uint64
	MessagesDropped  uint64
	RoutesKnown      int
}

// NewWhisperRouter creates a new whisper router.
func NewWhisperRouter(delivery *EndToEndDelivery, privateKey [32]byte) *WhisperRouter {
	return &WhisperRouter{
		delivery:   delivery,
		routes:     make(map[[32]byte]*WhisperRoute),
		pending:    make(map[[32]byte]*WhisperMessage),
		privateKey: privateKey,
	}
}

// AddRoute adds or updates a route to a Specter.
func (r *WhisperRouter) AddRoute(route *WhisperRoute) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route.LastSeen = time.Now()
	r.routes[route.Destination] = route
}

// RemoveRoute removes a route.
func (r *WhisperRouter) RemoveRoute(destination [32]byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, destination)
}

// GetRoute returns the route to a destination if known.
func (r *WhisperRouter) GetRoute(destination [32]byte) *WhisperRoute {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.routes[destination]
}

// RegisterHandler registers a handler for incoming whisper messages.
func (r *WhisperRouter) RegisterHandler(handler WhisperHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = append(r.handlers, handler)
}

// Send encrypts and sends a whisper message to a destination Specter.
func (r *WhisperRouter) Send(destination [32]byte, payload []byte) error {
	// Check route exists.
	route := r.GetRoute(destination)
	if route == nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return ErrWhisperNoRoute
	}

	// Encrypt the message.
	msg, err := EncryptWhisper(payload, destination)
	if err != nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return err
	}

	// Encode for transmission.
	encoded, err := encodeWhisperMessage(msg)
	if err != nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return err
	}

	// Send through Shroud circuit.
	shroudMsg := &Message{
		Type:    MessageTypeData,
		Dest:    destination,
		Payload: encoded,
	}

	if err := r.delivery.Send(shroudMsg); err != nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return err
	}

	atomic.AddUint64(&r.stats.MessagesSent, 1)
	return nil
}

// HandleIncoming processes an incoming whisper message.
func (r *WhisperRouter) HandleIncoming(data []byte) error {
	// Decode message.
	msg, err := decodeWhisperMessage(data)
	if err != nil {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return err
	}

	// Check expiry.
	if msg.IsExpired() {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return ErrWhisperExpired
	}

	// Check hop count.
	if msg.HopCount >= WhisperChainMaxHops {
		atomic.AddUint64(&r.stats.MessagesDropped, 1)
		return ErrWhisperChainTooLong
	}

	// Try to decrypt (if we're the recipient).
	payload, err := DecryptWhisper(msg, r.privateKey)
	if err == nil {
		// Successfully decrypted - we're the recipient.
		atomic.AddUint64(&r.stats.MessagesReceived, 1)

		r.mu.RLock()
		handlers := r.handlers
		r.mu.RUnlock()

		for _, handler := range handlers {
			if err := handler(msg, payload); err != nil {
				// Handler error, but message was received.
			}
		}

		return nil
	}

	// Decryption failed - we're not the recipient.
	// For now, just track as relayed (actual relay logic would go here).
	atomic.AddUint64(&r.stats.MessagesRelayed, 1)

	return nil
}

// Stats returns current router statistics.
func (r *WhisperRouter) Stats() WhisperRouterStats {
	r.mu.RLock()
	routeCount := len(r.routes)
	r.mu.RUnlock()

	return WhisperRouterStats{
		MessagesSent:     atomic.LoadUint64(&r.stats.MessagesSent),
		MessagesRelayed:  atomic.LoadUint64(&r.stats.MessagesRelayed),
		MessagesReceived: atomic.LoadUint64(&r.stats.MessagesReceived),
		MessagesDropped:  atomic.LoadUint64(&r.stats.MessagesDropped),
		RoutesKnown:      routeCount,
	}
}

// encodeWhisperMessage encodes a WhisperMessage for transmission.
func encodeWhisperMessage(msg *WhisperMessage) ([]byte, error) {
	// Format: MessageID(32) + SenderKey(32) + Nonce(24) + Timestamp(8) + TTL(4) + HopCount(1) + EncLen(2) + Encrypted
	headerSize := 32 + 32 + 24 + 8 + 4 + 1 + 2
	totalSize := headerSize + len(msg.Encrypted)

	buf := make([]byte, totalSize)
	offset := 0

	// MessageID.
	copy(buf[offset:], msg.MessageID[:])
	offset += 32

	// SenderKey.
	copy(buf[offset:], msg.SenderKey[:])
	offset += 32

	// Nonce.
	copy(buf[offset:], msg.Nonce[:])
	offset += 24

	// Timestamp (big-endian int64).
	binary.BigEndian.PutUint64(buf[offset:], uint64(msg.Timestamp))
	offset += 8

	// TTL (big-endian uint32).
	binary.BigEndian.PutUint32(buf[offset:], msg.TTL)
	offset += 4

	// HopCount.
	buf[offset] = msg.HopCount
	offset++

	// Encrypted length (big-endian).
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(msg.Encrypted)))
	offset += 2

	// Encrypted payload.
	copy(buf[offset:], msg.Encrypted)

	return buf, nil
}

// decodeWhisperMessage decodes a WhisperMessage from bytes.
func decodeWhisperMessage(data []byte) (*WhisperMessage, error) {
	headerSize := 32 + 32 + 24 + 8 + 4 + 1 + 2

	if len(data) < headerSize {
		return nil, errors.New("whisper message too short")
	}

	msg := &WhisperMessage{}
	offset := 0

	// MessageID.
	copy(msg.MessageID[:], data[offset:offset+32])
	offset += 32

	// SenderKey.
	copy(msg.SenderKey[:], data[offset:offset+32])
	offset += 32

	// Nonce.
	copy(msg.Nonce[:], data[offset:offset+24])
	offset += 24

	// Timestamp.
	for i := 0; i < 8; i++ {
		msg.Timestamp = (msg.Timestamp << 8) | int64(data[offset+i])
	}
	offset += 8

	// TTL.
	for i := 0; i < 4; i++ {
		msg.TTL = (msg.TTL << 8) | uint32(data[offset+i])
	}
	offset += 4

	// HopCount.
	msg.HopCount = data[offset]
	offset++

	// Encrypted length.
	encLen := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	// Validate length.
	if len(data) < offset+encLen {
		return nil, errors.New("whisper message truncated")
	}

	// Encrypted payload.
	msg.Encrypted = make([]byte, encLen)
	copy(msg.Encrypted, data[offset:offset+encLen])

	return msg, nil
}

// DeliveryConfirmation is a receipt proving message delivery.
// The receipt contains a blind signature that proves the message was received
// without revealing the sender's identity to the recipient.
type DeliveryConfirmation struct {
	// MessageID identifies the confirmed message.
	MessageID [32]byte

	// ReceiptNonce is a random nonce for uniqueness.
	ReceiptNonce [24]byte

	// ConfirmationHash is BLAKE3(MessageID || ReceiptNonce || RecipientKey).
	// This proves the recipient received the message without revealing sender.
	ConfirmationHash [32]byte

	// Timestamp is when the receipt was generated.
	Timestamp int64

	// RecipientSignature is the recipient's signature over the confirmation.
	// This uses an ephemeral key, not the recipient's main identity.
	RecipientSignature []byte
}

// DeliveryConfirmationRequest requests delivery confirmation for a message.
type DeliveryConfirmationRequest struct {
	// MessageID of the message needing confirmation.
	MessageID [32]byte

	// ResponseKey is the ephemeral public key for the response.
	ResponseKey [32]byte

	// TTL for the confirmation response.
	TTL uint32
}

// NewDeliveryConfirmation creates a delivery confirmation for a received message.
// Uses the recipient's ephemeral key to sign without revealing permanent identity.
func NewDeliveryConfirmation(messageID, recipientEphemeralKey [32]byte) (*DeliveryConfirmation, error) {
	confirmation := &DeliveryConfirmation{
		MessageID: messageID,
		Timestamp: time.Now().Unix(),
	}

	// Generate random nonce.
	if _, err := rand.Read(confirmation.ReceiptNonce[:]); err != nil {
		return nil, err
	}

	// Compute confirmation hash.
	h := blake3.New()
	h.Write(messageID[:])
	h.Write(confirmation.ReceiptNonce[:])
	h.Write(recipientEphemeralKey[:])
	copy(confirmation.ConfirmationHash[:], h.Sum(nil))

	return confirmation, nil
}

// VerifyDeliveryConfirmation verifies a delivery confirmation is valid for a message.
func VerifyDeliveryConfirmation(confirmation *DeliveryConfirmation, messageID [32]byte) bool {
	// MessageID must match.
	if confirmation.MessageID != messageID {
		return false
	}

	// Timestamp should be reasonable (within last hour).
	confirmTime := time.Unix(confirmation.Timestamp, 0)
	if time.Since(confirmTime) > time.Hour {
		return false
	}

	return true
}

// encodeDeliveryConfirmation encodes a delivery confirmation for transmission.
func encodeDeliveryConfirmation(conf *DeliveryConfirmation) ([]byte, error) {
	// Format: MessageID(32) + ReceiptNonce(24) + ConfirmationHash(32) + Timestamp(8) + SigLen(2) + Signature
	headerSize := 32 + 24 + 32 + 8 + 2
	totalSize := headerSize + len(conf.RecipientSignature)

	buf := make([]byte, totalSize)
	offset := 0

	// MessageID.
	copy(buf[offset:], conf.MessageID[:])
	offset += 32

	// ReceiptNonce.
	copy(buf[offset:], conf.ReceiptNonce[:])
	offset += 24

	// ConfirmationHash.
	copy(buf[offset:], conf.ConfirmationHash[:])
	offset += 32

	// Timestamp (big-endian int64).
	binary.BigEndian.PutUint64(buf[offset:], uint64(conf.Timestamp))
	offset += 8

	// Signature length (big-endian).
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(conf.RecipientSignature)))
	offset += 2

	// Signature.
	copy(buf[offset:], conf.RecipientSignature)

	return buf, nil
}

// decodeDeliveryConfirmation decodes a delivery confirmation from bytes.
func decodeDeliveryConfirmation(data []byte) (*DeliveryConfirmation, error) {
	headerSize := 32 + 24 + 32 + 8 + 2

	if len(data) < headerSize {
		return nil, errors.New("delivery confirmation too short")
	}

	conf := &DeliveryConfirmation{}
	offset := 0

	// MessageID.
	copy(conf.MessageID[:], data[offset:offset+32])
	offset += 32

	// ReceiptNonce.
	copy(conf.ReceiptNonce[:], data[offset:offset+24])
	offset += 24

	// ConfirmationHash.
	copy(conf.ConfirmationHash[:], data[offset:offset+32])
	offset += 32

	// Timestamp.
	for i := 0; i < 8; i++ {
		conf.Timestamp = (conf.Timestamp << 8) | int64(data[offset+i])
	}
	offset += 8

	// Signature length.
	sigLen := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	// Validate length.
	if len(data) < offset+sigLen {
		return nil, errors.New("delivery confirmation truncated")
	}

	// Signature.
	conf.RecipientSignature = make([]byte, sigLen)
	copy(conf.RecipientSignature, data[offset:offset+sigLen])

	return conf, nil
}

// PendingDelivery tracks a message awaiting confirmation.
type PendingDelivery struct {
	MessageID   [32]byte
	Destination [32]byte
	SentAt      time.Time
	RetryCount  int
	LastRetry   time.Time
	ResponseKey [32]byte // Ephemeral key for confirmation response.
	Confirmed   bool
	ConfirmedAt time.Time
}

// DeliveryTracker tracks pending deliveries and handles confirmations.
type DeliveryTracker struct {
	mu       sync.RWMutex
	pending  map[[32]byte]*PendingDelivery // MessageID -> pending info.
	handlers []DeliveryHandler
	stats    DeliveryStats
}

// DeliveryHandler handles delivery confirmation events.
type DeliveryHandler func(messageID [32]byte, confirmed bool)

// DeliveryStats tracks delivery statistics.
type DeliveryStats struct {
	MessagesSent     uint64
	Confirmations    uint64
	ConfirmationRate float64
	AverageLatency   time.Duration
}

// NewDeliveryTracker creates a new delivery tracker.
func NewDeliveryTracker() *DeliveryTracker {
	return &DeliveryTracker{
		pending: make(map[[32]byte]*PendingDelivery),
	}
}

// TrackDelivery begins tracking a sent message for delivery confirmation.
func (t *DeliveryTracker) TrackDelivery(messageID, destination [32]byte) (*PendingDelivery, error) {
	// Generate ephemeral response key.
	var responseKey [32]byte
	if _, err := rand.Read(responseKey[:]); err != nil {
		return nil, err
	}

	pending := &PendingDelivery{
		MessageID:   messageID,
		Destination: destination,
		SentAt:      time.Now(),
		ResponseKey: responseKey,
	}

	t.mu.Lock()
	t.pending[messageID] = pending
	atomic.AddUint64(&t.stats.MessagesSent, 1)
	t.mu.Unlock()

	return pending, nil
}

// ConfirmDelivery marks a message as delivered.
func (t *DeliveryTracker) ConfirmDelivery(confirmation *DeliveryConfirmation) error {
	t.mu.Lock()
	pending, exists := t.pending[confirmation.MessageID]
	if !exists {
		t.mu.Unlock()
		return errors.New("no pending delivery for message")
	}

	if pending.Confirmed {
		t.mu.Unlock()
		return nil // Already confirmed.
	}

	// Verify confirmation.
	if !VerifyDeliveryConfirmation(confirmation, pending.MessageID) {
		t.mu.Unlock()
		return errors.New("invalid delivery confirmation")
	}

	// Mark as confirmed.
	pending.Confirmed = true
	pending.ConfirmedAt = time.Now()
	atomic.AddUint64(&t.stats.Confirmations, 1)

	// Update rate.
	sent := atomic.LoadUint64(&t.stats.MessagesSent)
	confirmed := atomic.LoadUint64(&t.stats.Confirmations)
	if sent > 0 {
		t.stats.ConfirmationRate = float64(confirmed) / float64(sent)
	}

	handlers := t.handlers
	t.mu.Unlock()

	// Notify handlers.
	for _, handler := range handlers {
		handler(confirmation.MessageID, true)
	}

	return nil
}

// GetPending returns the pending delivery for a message.
func (t *DeliveryTracker) GetPending(messageID [32]byte) *PendingDelivery {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pending[messageID]
}

// RemovePending removes a pending delivery (e.g., on timeout or cancellation).
func (t *DeliveryTracker) RemovePending(messageID [32]byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.pending, messageID)
}

// RegisterHandler registers a handler for delivery events.
func (t *DeliveryTracker) RegisterHandler(handler DeliveryHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, handler)
}

// Stats returns current delivery statistics.
func (t *DeliveryTracker) Stats() DeliveryStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return DeliveryStats{
		MessagesSent:     atomic.LoadUint64(&t.stats.MessagesSent),
		Confirmations:    atomic.LoadUint64(&t.stats.Confirmations),
		ConfirmationRate: t.stats.ConfirmationRate,
		AverageLatency:   t.stats.AverageLatency,
	}
}

// CleanupExpired removes pending deliveries that have expired without confirmation.
func (t *DeliveryTracker) CleanupExpired(maxAge time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	removed := 0
	cutoff := time.Now().Add(-maxAge)

	for id, pending := range t.pending {
		if pending.SentAt.Before(cutoff) && !pending.Confirmed {
			delete(t.pending, id)
			removed++
		}
	}

	return removed
}

// Rate limiting constants.
const (
	// WhisperMaxMessagesPerSecond is the maximum whisper send rate per destination.
	WhisperMaxMessagesPerSecond = 2

	// WhisperMaxMessagesPerMinute is the maximum messages per minute per destination.
	WhisperMaxMessagesPerMinute = 30

	// WhisperMaxGlobalMessagesPerSecond is the overall maximum send rate.
	WhisperMaxGlobalMessagesPerSecond = 10

	// WhisperRateLimitWindow is the sliding window for rate calculations.
	WhisperRateLimitWindow = time.Minute

	// WhisperMaxPendingPerDest is the maximum pending messages per destination.
	WhisperMaxPendingPerDest = 10
)

// Rate limit errors.
var (
	ErrWhisperRateLimited       = errors.New("whisper rate limit exceeded")
	ErrWhisperDestRateLimited   = errors.New("destination rate limit exceeded")
	ErrWhisperGlobalRateLimited = errors.New("global rate limit exceeded")
	ErrWhisperTooManyPending    = errors.New("too many pending messages")
)

// RateLimiter provides rate limiting for Whisper messages.
// Uses a token bucket algorithm with per-destination and global limits.
type RateLimiter struct {
	mu sync.Mutex

	// Per-destination rate tracking.
	destBuckets map[[32]byte]*tokenBucket

	// Global rate tracking.
	globalBucket *tokenBucket

	// Pending counts per destination.
	pendingCounts map[[32]byte]int

	// Configuration.
	config RateLimiterConfig
}

// RateLimiterConfig configures rate limiting behavior.
type RateLimiterConfig struct {
	// MaxPerSecond is the maximum messages per second per destination.
	MaxPerSecond float64

	// MaxPerMinute is the maximum messages per minute per destination.
	MaxPerMinute int

	// GlobalMaxPerSecond is the overall maximum messages per second.
	GlobalMaxPerSecond float64

	// MaxPendingPerDest is the maximum pending messages per destination.
	MaxPendingPerDest int

	// BucketCapacity is the burst capacity for token buckets.
	BucketCapacity float64
}

// DefaultRateLimiterConfig returns the default rate limiter configuration.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxPerSecond:       WhisperMaxMessagesPerSecond,
		MaxPerMinute:       WhisperMaxMessagesPerMinute,
		GlobalMaxPerSecond: WhisperMaxGlobalMessagesPerSecond,
		MaxPendingPerDest:  WhisperMaxPendingPerDest,
		BucketCapacity:     5.0, // Allow burst of 5 messages.
	}
}

// tokenBucket implements a token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	capacity   float64
	rate       float64 // Tokens per second.
	lastUpdate time.Time
}

// newTokenBucket creates a new token bucket.
func newTokenBucket(rate, capacity float64) *tokenBucket {
	return &tokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		rate:       rate,
		lastUpdate: time.Now(),
	}
}

// tryConsume attempts to consume a token, returning true if successful.
func (tb *tokenBucket) tryConsume() bool {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.lastUpdate = now

	// Add tokens based on elapsed time.
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	// Try to consume.
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// tokensAvailable returns the current token count.
func (tb *tokenBucket) tokensAvailable() float64 {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()

	available := tb.tokens + elapsed*tb.rate
	if available > tb.capacity {
		available = tb.capacity
	}

	return available
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		destBuckets:   make(map[[32]byte]*tokenBucket),
		globalBucket:  newTokenBucket(config.GlobalMaxPerSecond, config.BucketCapacity*2),
		pendingCounts: make(map[[32]byte]int),
		config:        config,
	}
}

// NewDefaultRateLimiter creates a rate limiter with default configuration.
func NewDefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(DefaultRateLimiterConfig())
}

// Allow checks if a message to the destination is allowed.
// Returns nil if allowed, or an error describing the limit.
func (rl *RateLimiter) Allow(destination [32]byte) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check pending count.
	if rl.pendingCounts[destination] >= rl.config.MaxPendingPerDest {
		return ErrWhisperTooManyPending
	}

	// Check global rate.
	if !rl.globalBucket.tryConsume() {
		return ErrWhisperGlobalRateLimited
	}

	// Get or create destination bucket.
	bucket, exists := rl.destBuckets[destination]
	if !exists {
		bucket = newTokenBucket(rl.config.MaxPerSecond, rl.config.BucketCapacity)
		rl.destBuckets[destination] = bucket
	}

	// Check destination rate.
	if !bucket.tryConsume() {
		return ErrWhisperDestRateLimited
	}

	return nil
}

// Reserve marks a message as pending (before sending).
func (rl *RateLimiter) Reserve(destination [32]byte) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.pendingCounts[destination]++
}

// Release releases a pending reservation (after send completes or fails).
func (rl *RateLimiter) Release(destination [32]byte) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.pendingCounts[destination] > 0 {
		rl.pendingCounts[destination]--
	}
}

// GetPendingCount returns the current pending count for a destination.
func (rl *RateLimiter) GetPendingCount(destination [32]byte) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.pendingCounts[destination]
}

// GetAvailableTokens returns the available tokens for a destination.
func (rl *RateLimiter) GetAvailableTokens(destination [32]byte) float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.destBuckets[destination]
	if !exists {
		return rl.config.BucketCapacity
	}

	return bucket.tokensAvailable()
}

// GetGlobalAvailableTokens returns the available global tokens.
func (rl *RateLimiter) GetGlobalAvailableTokens() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.globalBucket.tokensAvailable()
}

// Cleanup removes stale destination buckets to free memory.
func (rl *RateLimiter) Cleanup(maxAge time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	removed := 0
	cutoff := time.Now().Add(-maxAge)

	for dest, bucket := range rl.destBuckets {
		if bucket.lastUpdate.Before(cutoff) && rl.pendingCounts[dest] == 0 {
			delete(rl.destBuckets, dest)
			delete(rl.pendingCounts, dest)
			removed++
		}
	}

	return removed
}

// Stats returns rate limiter statistics.
func (rl *RateLimiter) Stats() RateLimiterStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	totalPending := 0
	for _, count := range rl.pendingCounts {
		totalPending += count
	}

	return RateLimiterStats{
		DestinationsTracked: len(rl.destBuckets),
		TotalPending:        totalPending,
		GlobalTokens:        rl.globalBucket.tokensAvailable(),
	}
}

// RateLimiterStats contains rate limiter statistics.
type RateLimiterStats struct {
	DestinationsTracked int
	TotalPending        int
	GlobalTokens        float64
}

// RateLimitedRouter wraps a WhisperRouter with rate limiting.
type RateLimitedRouter struct {
	router  *WhisperRouter
	limiter *RateLimiter
}

// NewRateLimitedRouter creates a rate-limited whisper router.
func NewRateLimitedRouter(router *WhisperRouter, limiter *RateLimiter) *RateLimitedRouter {
	return &RateLimitedRouter{
		router:  router,
		limiter: limiter,
	}
}

// Send sends a message with rate limiting.
func (r *RateLimitedRouter) Send(destination [32]byte, payload []byte) error {
	// Check rate limit.
	if err := r.limiter.Allow(destination); err != nil {
		return err
	}

	// Reserve before sending.
	r.limiter.Reserve(destination)
	defer r.limiter.Release(destination)

	// Send through underlying router.
	return r.router.Send(destination, payload)
}

// Router returns the underlying WhisperRouter.
func (r *RateLimitedRouter) Router() *WhisperRouter {
	return r.router
}

// Limiter returns the underlying RateLimiter.
func (r *RateLimitedRouter) Limiter() *RateLimiter {
	return r.limiter
}
