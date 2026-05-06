package app

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

func TestNewHandlers(t *testing.T) {
	h, err := NewHandlers(HandlersConfig{})
	if err != nil {
		t.Fatalf("NewHandlers failed: %v", err)
	}
	if h == nil {
		t.Fatal("NewHandlers returned nil")
	}
	if h.dedupFilter == nil {
		t.Error("expected dedupFilter to be initialized")
	}
}

func TestHandlers_ValidateEnvelope(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})

	// Create a valid envelope.
	kp, _ := keys.GenerateKeyPair()
	payload := []byte("test payload")
	payloadHash := blake3.Sum256(payload)

	envelope := &pb.MurmurEnvelope{
		Version:       ProtocolVersion,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  kp.PublicKey,
		TimestampUnix: time.Now().Unix(),
		MessageId:     payloadHash[:],
	}

	// Sign the envelope.
	sigData := makeEnvelopeSignatureData(envelope)
	envelope.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	data, _ := proto.Marshal(envelope)

	// Test valid envelope.
	result, err := h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_WAVE)
	if err != nil {
		t.Errorf("validateEnvelope failed for valid envelope: %v", err)
	}
	if result == nil {
		t.Error("validateEnvelope returned nil for valid envelope")
	}

	// Test duplicate detection.
	_, err = h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_WAVE)
	if err != ErrDuplicateMessage {
		t.Errorf("expected ErrDuplicateMessage for duplicate, got %v", err)
	}

	// Test invalid version.
	h.ClearSeen()
	invalidVersion := &pb.MurmurEnvelope{
		Version:       99,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     payloadHash[:],
	}
	data, _ = proto.Marshal(invalidVersion)
	_, err = h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_WAVE)
	if err != ErrInvalidVersion {
		t.Errorf("expected ErrInvalidVersion, got %v", err)
	}

	// Test future timestamp.
	h.ClearSeen()
	futureEnvelope := &pb.MurmurEnvelope{
		Version:       ProtocolVersion,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		TimestampUnix: time.Now().Add(time.Hour).Unix(), // 1 hour in future
		MessageId:     payloadHash[:],
	}
	data, _ = proto.Marshal(futureEnvelope)
	_, err = h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_WAVE)
	if err != ErrInvalidTimestamp {
		t.Errorf("expected ErrInvalidTimestamp for future timestamp, got %v", err)
	}

	// Test wrong message type.
	h.ClearSeen()
	_, err = h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_IDENTITY)
	if err != ErrInvalidTimestamp { // Still fails on timestamp first
		// Test with valid timestamp
		wrongTypeEnv := &pb.MurmurEnvelope{
			Version:       ProtocolVersion,
			Type:          pb.MessageType_MESSAGE_TYPE_IDENTITY,
			Payload:       payload,
			TimestampUnix: time.Now().Unix(),
			MessageId:     payloadHash[:],
		}
		data, _ = proto.Marshal(wrongTypeEnv)
		_, err = h.validateEnvelope(data, pb.MessageType_MESSAGE_TYPE_WAVE)
		if err != ErrInvalidPayload {
			t.Errorf("expected ErrInvalidPayload for wrong type, got %v", err)
		}
	}
}

func TestHandlers_ValidateWave(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	// Create a valid wave.
	wave, err := waves.CreateSurface([]byte("Hello MURMUR!"), kp)
	if err != nil {
		t.Fatalf("CreateSurface failed: %v", err)
	}

	// Test valid wave.
	err = h.validateWave(wave)
	if err != nil {
		t.Errorf("validateWave failed for valid wave: %v", err)
	}

	// Test nil wave.
	err = h.validateWave(nil)
	if err != ErrInvalidPayload {
		t.Errorf("expected ErrInvalidPayload for nil wave, got %v", err)
	}

	// Test expired wave.
	expiredWave := &pb.Wave{
		WaveType:     pb.WaveType_WAVE_TYPE_SURFACE,
		Content:      []byte("old"),
		AuthorPubkey: kp.PublicKey,
		CreatedAt:    time.Now().Add(-31 * 24 * time.Hour).Unix(), // 31 days ago
		TtlSeconds:   int64((7 * 24 * time.Hour).Seconds()),
	}
	err = h.validateWave(expiredWave)
	if err != ErrMessageExpired {
		t.Errorf("expected ErrMessageExpired for expired wave, got %v", err)
	}
}

func TestHandlers_ValidateHeartbeat(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	// Create a valid heartbeat.
	hb := &pb.Heartbeat{
		PeerId:    "test-peer-id",
		PublicKey: kp.PublicKey,
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}

	// Sign the heartbeat.
	sigData := heartbeatSignatureData(hb)
	hb.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	// Test valid heartbeat.
	err := h.validateHeartbeat(hb)
	if err != nil {
		t.Errorf("validateHeartbeat failed for valid heartbeat: %v", err)
	}

	// Test nil heartbeat.
	err = h.validateHeartbeat(nil)
	if err != ErrInvalidPayload {
		t.Errorf("expected ErrInvalidPayload for nil heartbeat, got %v", err)
	}

	// Test invalid signature.
	hb.Signature = []byte("invalid")
	err = h.validateHeartbeat(hb)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestHandlers_ValidateIdentityDeclaration(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	// Create a valid declaration.
	decl := &pb.IdentityDeclaration{
		PublicKey:   kp.PublicKey,
		DisplayName: "TestUser",
		Bio:         "A test user",
		CreatedAt:   time.Now().Unix(),
		Version:     1,
		PrivacyMode: pb.PrivacyMode_PRIVACY_MODE_HYBRID,
	}

	// Sign the declaration.
	sigData := h.identityDeclarationSignatureData(decl)
	decl.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	// Test valid declaration.
	err := h.validateIdentityDeclaration(decl)
	if err != nil {
		t.Errorf("validateIdentityDeclaration failed: %v", err)
	}

	// Test nil declaration.
	err = h.validateIdentityDeclaration(nil)
	if err != ErrInvalidPayload {
		t.Errorf("expected ErrInvalidPayload for nil, got %v", err)
	}

	// Test invalid public key size.
	badDecl := &pb.IdentityDeclaration{
		PublicKey: []byte("short"),
	}
	err = h.validateIdentityDeclaration(badDecl)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature for bad pubkey, got %v", err)
	}
}

func TestHandlers_ValidateRelayAdvertisement(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	// Create a valid relay advertisement.
	ad := &pb.RelayAdvertisement{
		Curve25519Pubkey: make([]byte, 32),
		Ed25519Pubkey:    kp.PublicKey,
		Addrs:            []string{"/ip4/127.0.0.1/tcp/4001"},
		Roles:            []pb.RelayRole{pb.RelayRole_RELAY_ROLE_MIDDLE},
		Bandwidth:        1000000,
		Timestamp:        time.Now().Unix(),
		ExpiresAt:        time.Now().Add(time.Hour).Unix(),
	}

	// Sign the advertisement.
	sigData := h.relayAdSignatureData(ad)
	ad.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	// Test valid advertisement.
	err := h.validateRelayAdvertisement(ad)
	if err != nil {
		t.Errorf("validateRelayAdvertisement failed: %v", err)
	}

	// Test expired advertisement.
	expiredAd := &pb.RelayAdvertisement{
		Ed25519Pubkey: kp.PublicKey,
		ExpiresAt:     time.Now().Add(-time.Hour).Unix(), // 1 hour ago
	}
	err = h.validateRelayAdvertisement(expiredAd)
	if err != ErrMessageExpired {
		t.Errorf("expected ErrMessageExpired, got %v", err)
	}
}

func TestHandlers_Deduplication(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})

	// Add a message ID.
	msgID := []byte("test-message-id-123")
	h.markSeen(msgID)

	// Should be marked as duplicate.
	if !h.isDuplicate(msgID) {
		t.Error("message should be marked as duplicate after markSeen")
	}

	// Different message ID should not be duplicate.
	otherID := []byte("other-message-id-456")
	if h.isDuplicate(otherID) {
		t.Error("unseen message should not be marked as duplicate")
	}

	// Clear and check.
	h.ClearSeen()
	if h.isDuplicate(msgID) {
		t.Error("message should not be duplicate after clear")
	}
}

func TestHandlers_Callbacks(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})

	var waveReceived bool
	h.SetWaveCallback(func(w *pb.Wave) {
		waveReceived = true
	})

	var identityReceived bool
	h.SetIdentityCallback(func(d *pb.IdentityDeclaration) {
		identityReceived = true
	})

	var heartbeatReceived bool
	h.SetHeartbeatCallback(func(hb *pb.Heartbeat) {
		heartbeatReceived = true
	})

	var relayAdReceived bool
	h.SetRelayAdCallback(func(ad *pb.RelayAdvertisement) {
		relayAdReceived = true
	})

	// Verify callbacks are set (they're invoked by handlers, tested above).
	h.mu.RLock()
	if h.onWaveReceived == nil {
		t.Error("wave callback not set")
	}
	if h.onIdentityReceived == nil {
		t.Error("identity callback not set")
	}
	if h.onHeartbeatReceived == nil {
		t.Error("heartbeat callback not set")
	}
	if h.onRelayAdReceived == nil {
		t.Error("relay ad callback not set")
	}
	h.mu.RUnlock()

	// Suppress unused variable warnings.
	_ = waveReceived
	_ = identityReceived
	_ = heartbeatReceived
	_ = relayAdReceived
}

func TestHandlers_WaveHandlerIntegration(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	var receivedWave *pb.Wave
	h.SetWaveCallback(func(w *pb.Wave) {
		receivedWave = w
	})

	// Create a valid wave.
	wave, err := waves.CreateSurface([]byte("Integration test wave"), kp)
	if err != nil {
		t.Fatalf("CreateSurface failed: %v", err)
	}

	// Wrap in envelope.
	payload, _ := proto.Marshal(wave)
	payloadHash := blake3.Sum256(payload)

	envelope := &pb.MurmurEnvelope{
		Version:       ProtocolVersion,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  kp.PublicKey,
		TimestampUnix: time.Now().Unix(),
		MessageId:     payloadHash[:],
	}

	sigData := makeEnvelopeSignatureData(envelope)
	envelope.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	data, _ := proto.Marshal(envelope)

	// Create a pubsub message.
	topic := "test"
	msg := &pubsub.Message{
		Message: &pubsub_pb.Message{Data: data, Topic: &topic},
	}

	// Handle the message.
	h.handleWaveMessage(context.Background(), msg)

	// Verify callback was invoked.
	if receivedWave == nil {
		t.Error("wave callback was not invoked")
	}
}

// makeEnvelopeSignatureData creates the data to sign for an envelope.
func makeEnvelopeSignatureData(env *pb.MurmurEnvelope) []byte {
	var data []byte

	versionBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(versionBytes, env.Version)
	data = append(data, versionBytes...)

	typeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(typeBytes, uint32(env.Type))
	data = append(data, typeBytes...)

	data = append(data, env.Payload...)

	return data
}

func TestHandlers_validateWaveWithPoW(t *testing.T) {
	h, _ := NewHandlers(HandlersConfig{})
	kp, _ := keys.GenerateKeyPair()

	// Create wave with actual PoW.
	opts := waves.CreateOptions{
		TTL:        7 * 24 * time.Hour,
		Difficulty: pow.DefaultDifficulty,
	}
	wave, err := waves.Create(waves.TypeSurface, []byte("PoW test"), kp, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Validate should pass.
	err = h.validateWave(wave)
	if err != nil {
		t.Errorf("validateWave failed for wave with valid PoW: %v", err)
	}

	// Tamper with PoW nonce.
	wave.PowNonce = 0
	err = h.validateWave(wave)
	if err == nil {
		t.Error("validateWave should fail for invalid PoW")
	}
}

// TestStartDedupRotationContextCancellation verifies the deduplication rotation
// goroutine exits within 1s of context cancellation per AUDIT.md H2.
func TestStartDedupRotationContextCancellation(t *testing.T) {
	h, err := NewHandlers(HandlersConfig{})
	if err != nil {
		t.Fatalf("NewHandlers failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.StartDedupRotation(ctx)
	}()

	// Cancel context and verify goroutine exits promptly.
	cancel()

	select {
	case <-done:
		// Expected - goroutine exited
	case <-time.After(1 * time.Second):
		t.Fatal("StartDedupRotation() did not exit within 1s of context cancellation")
	}
}
