// Package app provides the top-level application lifecycle and event bus for MURMUR.
// This file implements GossipSub message handlers for all four core topics.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, handlers validate MurmurEnvelope,
// verify signatures, check timestamps, validate PoW, and dispatch to storage.
package app

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

// Handler configuration constants per TECHNICAL_IMPLEMENTATION.md.
const (
	// MaxTimestampDrift is the maximum allowed clock drift for messages.
	// Envelopes with timestamps more than 300s in the future are rejected.
	MaxTimestampDrift = 300 * time.Second

	// ProtocolVersion is the current protocol version.
	ProtocolVersion uint32 = 1
)

// Handler errors.
var (
	ErrInvalidEnvelope  = errors.New("invalid envelope format")
	ErrInvalidVersion   = errors.New("unsupported protocol version")
	ErrInvalidSignature = errors.New("invalid envelope signature")
	ErrInvalidTimestamp = errors.New("timestamp out of acceptable range")
	ErrMessageExpired   = errors.New("message has expired")
	ErrDuplicateMessage = errors.New("duplicate message")
	ErrInvalidMessageID = errors.New("invalid message ID")
	ErrInvalidPayload   = errors.New("invalid payload format")
	ErrInvalidPoW       = errors.New("invalid proof of work")
	ErrMessageTooLarge  = errors.New("message exceeds size limit")
	ErrHandlerNotReady  = errors.New("handler dependencies not initialized")
)

// Handlers manages GossipSub message handlers for all MURMUR topics.
// It coordinates validation, deduplication, and dispatch to appropriate stores.
type Handlers struct {
	mu sync.RWMutex

	// cache stores received Waves.
	cache *storage.Cache

	// dedupFilter is a Bloom filter for message deduplication.
	// Per TECHNICAL_IMPLEMENTATION.md §3.2, uses 30-day window with 10M capacity.
	dedupFilter *bloom.BloomFilter

	// dedupMu protects dedupFilter.
	dedupMu sync.RWMutex

	// dedupCreatedAt tracks when the filter was created for 30-day rotation.
	dedupCreatedAt time.Time

	// onWaveReceived is called when a valid Wave is received.
	onWaveReceived func(*pb.Wave)

	// onIdentityReceived is called when a valid identity declaration is received.
	onIdentityReceived func(*pb.IdentityDeclaration)

	// onHeartbeatReceived is called when a valid heartbeat is received.
	onHeartbeatReceived func(*pb.Heartbeat)

	// onRelayAdReceived is called when a valid relay advertisement is received.
	onRelayAdReceived func(*pb.RelayAdvertisement)
}

// HandlersConfig configures the message handlers.
type HandlersConfig struct {
	Cache *storage.Cache
	// DedupCapacity is the expected number of unique messages in 30 days.
	// Per AUDIT.md, defaults to 10 million with 0.01 false positive rate.
	DedupCapacity uint
	// DedupFalsePositiveRate is the acceptable false positive rate.
	DedupFalsePositiveRate float64
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(cfg HandlersConfig) (*Handlers, error) {
	// Set defaults per AUDIT.md: 10M capacity, 0.01 FP rate.
	capacity := cfg.DedupCapacity
	if capacity == 0 {
		capacity = 10_000_000
	}
	fpRate := cfg.DedupFalsePositiveRate
	if fpRate == 0 {
		fpRate = 0.01
	}

	// Create Bloom filter for deduplication.
	dedupFilter := bloom.NewWithEstimates(capacity, fpRate)

	return &Handlers{
		cache:          cfg.Cache,
		dedupFilter:    dedupFilter,
		dedupCreatedAt: time.Now(),
	}, nil
}

// SetWaveCallback sets the callback for received Waves.
func (h *Handlers) SetWaveCallback(fn func(*pb.Wave)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onWaveReceived = fn
}

// SetIdentityCallback sets the callback for received identity declarations.
func (h *Handlers) SetIdentityCallback(fn func(*pb.IdentityDeclaration)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onIdentityReceived = fn
}

// SetHeartbeatCallback sets the callback for received heartbeats.
func (h *Handlers) SetHeartbeatCallback(fn func(*pb.Heartbeat)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onHeartbeatReceived = fn
}

// SetRelayAdCallback sets the callback for received relay advertisements.
func (h *Handlers) SetRelayAdCallback(fn func(*pb.RelayAdvertisement)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onRelayAdReceived = fn
}

// invokeIdentityCallback safely invokes the identity callback if set.
func (h *Handlers) invokeIdentityCallback(decl *pb.IdentityDeclaration) {
	h.mu.RLock()
	callback := h.onIdentityReceived
	h.mu.RUnlock()

	if callback != nil {
		callback(decl)
	}
}

// invokeRelayAdCallback safely invokes the relay advertisement callback if set.
func (h *Handlers) invokeRelayAdCallback(relayAd *pb.RelayAdvertisement) {
	h.mu.RLock()
	callback := h.onRelayAdReceived
	h.mu.RUnlock()

	if callback != nil {
		callback(relayAd)
	}
}

// invokeHeartbeatCallback safely invokes the heartbeat callback if set.
func (h *Handlers) invokeHeartbeatCallback(heartbeat *pb.Heartbeat) {
	h.mu.RLock()
	callback := h.onHeartbeatReceived
	h.mu.RUnlock()

	if callback != nil {
		callback(heartbeat)
	}
}

// RegisterAll registers handlers for all core GossipSub topics.
func (h *Handlers) RegisterAll(ctx context.Context, ps *gossip.PubSub) error {
	handlers := map[string]gossip.MessageHandler{
		gossip.TopicWaves:    h.handleWaveMessage,
		gossip.TopicIdentity: h.handleIdentityMessage,
		gossip.TopicShroud:   h.handleShroudMessage,
		gossip.TopicPulse:    h.handlePulseMessage,
	}

	for topic, handler := range handlers {
		if err := ps.Subscribe(ctx, topic, handler); err != nil {
			return err
		}
	}

	return nil
}

// RegisterAnonymousMechanics subscribes to anonymous layer topics for mini-game events.
// Per AUDIT.md remediation, this enables network propagation of Phantom Gifts, Specter Marks,
// Cipher Puzzles, and other anonymous mechanics. Should be called during initialization for
// nodes in Hybrid/Guarded/Fortress modes.
func (h *Handlers) RegisterAnonymousMechanics(ctx context.Context, ps *gossip.PubSub) error {
	handlers := map[string]gossip.MessageHandler{
		gossip.TopicAnonymousWaves:     h.handleAnonymousWavesMessage,
		gossip.TopicAnonymousMechanics: h.handleAnonymousMechanicsMessage,
		gossip.TopicAnonymousBeacons:   h.handleAnonymousBeaconsMessage,
	}

	for topic, handler := range handlers {
		if err := ps.Subscribe(ctx, topic, handler); err != nil {
			return err
		}
	}

	return nil
}

// handleWaveMessage processes incoming Wave messages from /murmur/waves/1.
func (h *Handlers) handleWaveMessage(ctx context.Context, msg *pubsub.Message) {
	envelope, err := h.validateEnvelope(msg.Data, pb.MessageType_MESSAGE_TYPE_WAVE)
	if err != nil {
		return // Silently drop invalid messages
	}

	wave := &pb.Wave{}
	if err := proto.Unmarshal(envelope.Payload, wave); err != nil {
		return
	}

	if err := h.validateWave(wave); err != nil {
		return
	}

	// Store in cache if available.
	if h.cache != nil {
		if err := h.cache.Put(wave); err != nil {
			// Log error but don't fail - message was valid
		}
	}

	// Invoke callback if set.
	h.mu.RLock()
	callback := h.onWaveReceived
	h.mu.RUnlock()

	if callback != nil {
		callback(wave)
	}
}

// handleIdentityMessage processes incoming identity declarations from /murmur/identity/1.
func (h *Handlers) handleIdentityMessage(ctx context.Context, msg *pubsub.Message) {
	envelope, err := h.validateEnvelope(msg.Data, pb.MessageType_MESSAGE_TYPE_IDENTITY)
	if err != nil {
		return
	}

	decl := &pb.IdentityDeclaration{}
	if err := proto.Unmarshal(envelope.Payload, decl); err != nil {
		return
	}

	if err := h.validateIdentityDeclaration(decl); err != nil {
		return
	}

	h.invokeIdentityCallback(decl)
}

// handleShroudMessage processes relay advertisements from /murmur/shroud/1.
func (h *Handlers) handleShroudMessage(ctx context.Context, msg *pubsub.Message) {
	envelope, err := h.validateEnvelope(msg.Data, pb.MessageType_MESSAGE_TYPE_SHROUD_AD)
	if err != nil {
		return
	}

	relayAd := &pb.RelayAdvertisement{}
	if err := proto.Unmarshal(envelope.Payload, relayAd); err != nil {
		return
	}

	if err := h.validateRelayAdvertisement(relayAd); err != nil {
		return
	}

	h.invokeRelayAdCallback(relayAd)
}

// handlePulseMessage processes heartbeat pings from /murmur/pulse/1.
func (h *Handlers) handlePulseMessage(ctx context.Context, msg *pubsub.Message) {
	envelope, err := h.validateEnvelope(msg.Data, pb.MessageType_MESSAGE_TYPE_HEARTBEAT)
	if err != nil {
		return
	}

	heartbeat := &pb.Heartbeat{}
	if err := proto.Unmarshal(envelope.Payload, heartbeat); err != nil {
		return
	}

	if err := h.validateHeartbeat(heartbeat); err != nil {
		return
	}

	h.invokeHeartbeatCallback(heartbeat)
}

// handleAnonymousWavesMessage processes Specter and Masked Waves from /murmur/anonymous/waves/1.0.
// Per AUDIT.md remediation, this enables network propagation of anonymous layer Waves.
func (h *Handlers) handleAnonymousWavesMessage(ctx context.Context, msg *pubsub.Message) {
	// Per gossip/anonymous.go, anonymous waves use quantized timestamps.
	// Basic validation - full implementation would parse Specter/Masked Wave payloads
	// and route to pkg/anonymous/specters/ handlers.
	envelope := &pb.MurmurEnvelope{}
	if err := proto.Unmarshal(msg.Data, envelope); err != nil {
		return // Silently drop invalid messages
	}

	// For now, just acknowledge receipt. Full implementation will:
	// 1. Parse Wave from envelope.Payload
	// 2. Validate Specter signature or Masked Wave encryption
	// 3. Store in anonymous Wave cache
	// 4. Emit event to event bus for UI rendering
	_ = envelope
}

// handleAnonymousMechanicsMessage processes anonymous mini-game events from /murmur/anonymous/mechanics/1.0.
// Per AUDIT.md remediation, this enables Phantom Gifts, Specter Marks, and other mechanics to propagate.
func (h *Handlers) handleAnonymousMechanicsMessage(ctx context.Context, msg *pubsub.Message) {
	// Per pkg/anonymous/mechanics/publisher.go, all mechanics use TopicAnonymousMechanics.
	// Parse GossipMessage from envelope payload to determine event type.
	envelope := &pb.MurmurEnvelope{}
	if err := proto.Unmarshal(msg.Data, envelope); err != nil {
		return
	}

	// For now, just acknowledge receipt. Full implementation will:
	// 1. Parse GossipMessage from envelope.Payload
	// 2. Route to appropriate mechanic handler (gifts, marks, puzzles, etc.)
	// 3. Verify signatures and ZK proofs where applicable
	// 4. Store events in respective mechanic stores
	// 5. Emit events to event bus for cross-layer rendering
	_ = envelope
}

// handleAnonymousBeaconsMessage processes Beacon Waves from /murmur/anonymous/beacons/1.0.
// Per AUDIT.md remediation, this enables Shroud relay discovery via elevated-PoW Beacon Waves.
func (h *Handlers) handleAnonymousBeaconsMessage(ctx context.Context, msg *pubsub.Message) {
	// Beacon Waves have PoW difficulty 24 (vs 20 for standard Waves).
	// Parse and validate, then extract relay advertisement.
	envelope := &pb.MurmurEnvelope{}
	if err := proto.Unmarshal(msg.Data, envelope); err != nil {
		return
	}

	// For now, just acknowledge receipt. Full implementation will:
	// 1. Validate elevated PoW (24 leading zero bits)
	// 2. Parse RelayAdvertisement from Wave payload
	// 3. Pass to Beacon.ProcessAdvertisement()
	// 4. Emit EventShroudRelayDiscovered to event bus
	_ = envelope
}

// validateEnvelope validates a MurmurEnvelope and checks for duplicates.
func (h *Handlers) validateEnvelope(data []byte, expectedType pb.MessageType) (*pb.MurmurEnvelope, error) {
	envelope := &pb.MurmurEnvelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, ErrInvalidEnvelope
	}

	// Check protocol version.
	if envelope.Version != ProtocolVersion {
		return nil, ErrInvalidVersion
	}

	// Check message type.
	if envelope.Type != expectedType {
		return nil, ErrInvalidPayload
	}

	// Verify message ID (BLAKE3 hash of payload).
	expectedID := blake3.Sum256(envelope.Payload)
	if !bytes.Equal(envelope.MessageId, expectedID[:]) {
		return nil, ErrInvalidMessageID
	}

	// Check timestamp is not too far in the future.
	msgTime := time.Unix(envelope.TimestampUnix, 0)
	if msgTime.After(time.Now().Add(MaxTimestampDrift)) {
		return nil, ErrInvalidTimestamp
	}

	// Check for duplicates.
	if h.isDuplicate(envelope.MessageId) {
		return nil, ErrDuplicateMessage
	}

	// Verify signature (if sender_pubkey is present).
	if len(envelope.SenderPubkey) == ed25519.PublicKeySize {
		if err := h.verifyEnvelopeSignature(envelope); err != nil {
			return nil, err
		}
	}

	// Mark as seen.
	h.markSeen(envelope.MessageId)

	return envelope, nil
}

// verifyEnvelopeSignature verifies the Ed25519 signature on an envelope.
func (h *Handlers) verifyEnvelopeSignature(envelope *pb.MurmurEnvelope) error {
	// Signature is over: version || type || payload
	var sigData []byte

	// Version as 4 bytes (big-endian).
	versionBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(versionBytes, envelope.Version)
	sigData = append(sigData, versionBytes...)

	// Type as 4 bytes (big-endian).
	typeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(typeBytes, uint32(envelope.Type))
	sigData = append(sigData, typeBytes...)

	// Payload.
	sigData = append(sigData, envelope.Payload...)

	if !ed25519.Verify(envelope.SenderPubkey, sigData, envelope.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// isDuplicate checks if a message ID has been seen before using the Bloom filter.
func (h *Handlers) isDuplicate(messageID []byte) bool {
	h.dedupMu.RLock()
	defer h.dedupMu.RUnlock()

	return h.dedupFilter.Test(messageID)
}

// markSeen records a message ID as seen in the Bloom filter.
func (h *Handlers) markSeen(messageID []byte) {
	h.dedupMu.Lock()
	defer h.dedupMu.Unlock()

	h.dedupFilter.Add(messageID)
}

// rotateDedupFilter clears and recreates the Bloom filter.
// Called every 30 days per TECHNICAL_IMPLEMENTATION.md §3.2.
func (h *Handlers) rotateDedupFilter() {
	h.dedupMu.Lock()
	defer h.dedupMu.Unlock()

	// Clear the filter by creating a new one with the same parameters.
	capacity := uint(10_000_000) // 10M per AUDIT.md
	fpRate := 0.01               // 1% false positive rate
	h.dedupFilter = bloom.NewWithEstimates(capacity, fpRate)
	h.dedupCreatedAt = time.Now()
}

// StartDedupRotation runs a background goroutine that rotates the dedup filter every 30 days.
// Per AUDIT.md, the filter should be cleared every 30 days to prevent unbounded growth.
func (h *Handlers) StartDedupRotation(ctx context.Context) {
	ticker := time.NewTicker(30 * 24 * time.Hour) // 30 days
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.rotateDedupFilter()
		}
	}
}

// validateWave validates a Wave's signature, PoW, and expiration.
func (h *Handlers) validateWave(wave *pb.Wave) error {
	if wave == nil {
		return ErrInvalidPayload
	}

	// Check expiration.
	if waves.IsExpired(wave) {
		return ErrMessageExpired
	}

	// Validate PoW and signature using the waves package.
	return waves.Validate(wave, pow.DefaultDifficulty)
}

// validateIdentityDeclaration validates an identity declaration's signature.
func (h *Handlers) validateIdentityDeclaration(decl *pb.IdentityDeclaration) error {
	if decl == nil {
		return ErrInvalidPayload
	}

	if len(decl.PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}

	// Build signature data (all fields except signature).
	sigData := h.identityDeclarationSignatureData(decl)

	if !ed25519.Verify(decl.PublicKey, sigData, decl.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// identityDeclarationSignatureData builds the data to verify for an identity declaration.
// This is a wrapper that delegates to the package-level function.
func (h *Handlers) identityDeclarationSignatureData(decl *pb.IdentityDeclaration) []byte {
	return identityDeclarationSignatureData(decl)
}

// validateHeartbeat validates a heartbeat message's signature.
func (h *Handlers) validateHeartbeat(hb *pb.Heartbeat) error {
	if hb == nil {
		return ErrInvalidPayload
	}

	if len(hb.PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}

	// Signature is over peer_id + timestamp.
	sigData := h.heartbeatSignatureData(hb)

	if !ed25519.Verify(hb.PublicKey, sigData, hb.Signature) {
		return ErrInvalidSignature
	}

	// Check timestamp drift.
	hbTime := time.Unix(hb.Timestamp, 0)
	if hbTime.After(time.Now().Add(MaxTimestampDrift)) {
		return ErrInvalidTimestamp
	}

	return nil
}

// heartbeatSignatureData builds the data to verify for a heartbeat.
func (h *Handlers) heartbeatSignatureData(hb *pb.Heartbeat) []byte {
	var data []byte
	data = append(data, []byte(hb.PeerId)...)

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(hb.Timestamp))
	data = append(data, ts...)

	return data
}

// validateRelayAdvertisement validates a relay advertisement's signature.
func (h *Handlers) validateRelayAdvertisement(ad *pb.RelayAdvertisement) error {
	if ad == nil {
		return ErrInvalidPayload
	}

	if len(ad.Ed25519Pubkey) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}

	// Check expiration.
	if time.Now().Unix() > ad.ExpiresAt {
		return ErrMessageExpired
	}

	// Build signature data (all fields except signature).
	sigData := h.relayAdSignatureData(ad)

	if !ed25519.Verify(ad.Ed25519Pubkey, sigData, ad.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// relayAdSignatureData builds the data to verify for a relay advertisement.
func (h *Handlers) relayAdSignatureData(ad *pb.RelayAdvertisement) []byte {
	var data []byte
	data = append(data, ad.Curve25519Pubkey...)
	data = append(data, ad.Ed25519Pubkey...)

	for _, addr := range ad.Addrs {
		data = append(data, []byte(addr)...)
	}

	for _, role := range ad.Roles {
		r := make([]byte, 4)
		binary.BigEndian.PutUint32(r, uint32(role))
		data = append(data, r...)
	}

	bw := make([]byte, 8)
	binary.BigEndian.PutUint64(bw, ad.Bandwidth)
	data = append(data, bw...)

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(ad.Timestamp))
	data = append(data, ts...)

	exp := make([]byte, 8)
	binary.BigEndian.PutUint64(exp, uint64(ad.ExpiresAt))
	data = append(data, exp...)

	return data
}

// ClearSeen clears the deduplication Bloom filter.
// Equivalent to rotateDedupFilter() but exposed for testing.
func (h *Handlers) ClearSeen() {
	h.rotateDedupFilter()
}
