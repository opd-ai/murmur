package proto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestNewWaveEnvelope(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	wave := CreateSurfaceWave([]byte("test content"), privKey.Public().(ed25519.PublicKey))

	env, err := NewWaveEnvelope(wave, privKey)
	if err != nil {
		t.Fatalf("NewWaveEnvelope() error: %v", err)
	}

	if env.Version != CurrentProtocolVersion {
		t.Errorf("Version = %d, want %d", env.Version, CurrentProtocolVersion)
	}
	if env.Type != MessageType_MESSAGE_TYPE_WAVE {
		t.Errorf("Type = %v, want WAVE", env.Type)
	}
	if len(env.Signature) != SignatureLength {
		t.Errorf("Signature length = %d, want %d", len(env.Signature), SignatureLength)
	}

	// Validate the envelope
	if err := ValidateEnvelope(env); err != nil {
		t.Errorf("ValidateEnvelope() error: %v", err)
	}
}

func TestNewIdentityEnvelope(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	decl := &IdentityDeclaration{
		PublicKey:   pubKey,
		DisplayName: "Test User",
	}

	env, err := NewIdentityEnvelope(decl, privKey)
	if err != nil {
		t.Fatalf("NewIdentityEnvelope() error: %v", err)
	}

	if env.Type != MessageType_MESSAGE_TYPE_IDENTITY {
		t.Errorf("Type = %v, want IDENTITY", env.Type)
	}

	if err := ValidateEnvelope(env); err != nil {
		t.Errorf("ValidateEnvelope() error: %v", err)
	}
}

func TestNewShroudAdEnvelope(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	relay := &RelayAdvertisement{
		Curve25519Pubkey: make([]byte, 32),
		Ed25519Pubkey:    make([]byte, 32),
		Bandwidth:        1000,
	}

	env, err := NewShroudAdEnvelope(relay, privKey)
	if err != nil {
		t.Fatalf("NewShroudAdEnvelope() error: %v", err)
	}

	if env.Type != MessageType_MESSAGE_TYPE_SHROUD_AD {
		t.Errorf("Type = %v, want SHROUD_AD", env.Type)
	}

	if err := ValidateEnvelope(env); err != nil {
		t.Errorf("ValidateEnvelope() error: %v", err)
	}
}

func TestNewHeartbeatEnvelope(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	heartbeat := CreateHeartbeat("test-peer-id")

	env, err := NewHeartbeatEnvelope(heartbeat, privKey)
	if err != nil {
		t.Fatalf("NewHeartbeatEnvelope() error: %v", err)
	}

	if env.Type != MessageType_MESSAGE_TYPE_HEARTBEAT {
		t.Errorf("Type = %v, want HEARTBEAT", env.Type)
	}

	if err := ValidateEnvelope(env); err != nil {
		t.Errorf("ValidateEnvelope() error: %v", err)
	}
}

func TestNewAnonymousWaveEnvelope(t *testing.T) {
	wave := CreateSpecterWave([]byte("anonymous content"))

	env, err := NewAnonymousWaveEnvelope(wave)
	if err != nil {
		t.Fatalf("NewAnonymousWaveEnvelope() error: %v", err)
	}

	if env.Type != MessageType_MESSAGE_TYPE_WAVE {
		t.Errorf("Type = %v, want WAVE", env.Type)
	}

	// Check sender_pubkey is zeroed
	if !isZeroBytes(env.SenderPubkey) {
		t.Error("Anonymous envelope should have zeroed sender_pubkey")
	}

	// Check no signature
	if len(env.Signature) != 0 {
		t.Error("Anonymous envelope should have no signature")
	}
}

func TestUnwrapWave(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	wave := CreateSurfaceWave([]byte("test content"), pubKey)
	env, _ := NewWaveEnvelope(wave, privKey)

	unwrapped, err := UnwrapWave(env)
	if err != nil {
		t.Fatalf("UnwrapWave() error: %v", err)
	}

	if unwrapped.WaveType != WaveType_WAVE_TYPE_SURFACE {
		t.Errorf("WaveType = %v, want SURFACE", unwrapped.WaveType)
	}
	if string(unwrapped.Content) != "test content" {
		t.Errorf("Content = %q, want %q", string(unwrapped.Content), "test content")
	}
}

func TestUnwrapIdentityDeclaration(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	decl := &IdentityDeclaration{
		PublicKey:   pubKey,
		DisplayName: "Test User",
	}
	env, _ := NewIdentityEnvelope(decl, privKey)

	unwrapped, err := UnwrapIdentityDeclaration(env)
	if err != nil {
		t.Fatalf("UnwrapIdentityDeclaration() error: %v", err)
	}

	if unwrapped.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", unwrapped.DisplayName, "Test User")
	}
}

func TestUnwrapRelayAdvertisement(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	relay := &RelayAdvertisement{
		Curve25519Pubkey: make([]byte, 32),
		Ed25519Pubkey:    make([]byte, 32),
		Bandwidth:        1000,
	}
	env, _ := NewShroudAdEnvelope(relay, privKey)

	unwrapped, err := UnwrapRelayAdvertisement(env)
	if err != nil {
		t.Fatalf("UnwrapRelayAdvertisement() error: %v", err)
	}

	if unwrapped.Bandwidth != 1000 {
		t.Errorf("Bandwidth = %d, want %d", unwrapped.Bandwidth, 1000)
	}
}

func TestUnwrapHeartbeat(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	heartbeat := CreateHeartbeat("test-peer")
	env, _ := NewHeartbeatEnvelope(heartbeat, privKey)

	unwrapped, err := UnwrapHeartbeat(env)
	if err != nil {
		t.Fatalf("UnwrapHeartbeat() error: %v", err)
	}

	if unwrapped.PeerId != "test-peer" {
		t.Errorf("PeerId = %q, want %q", unwrapped.PeerId, "test-peer")
	}
}

func TestUnwrapWrongType(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	heartbeat := CreateHeartbeat("test-peer")
	env, _ := NewHeartbeatEnvelope(heartbeat, privKey)

	// Try to unwrap heartbeat as wave
	_, err := UnwrapWave(env)
	if err != ErrInvalidMessageType {
		t.Errorf("UnwrapWave(heartbeat) error = %v, want %v", err, ErrInvalidMessageType)
	}
}

func TestCreateSurfaceWave(t *testing.T) {
	content := []byte("hello world")
	pubKey := make([]byte, 32)

	wave := CreateSurfaceWave(content, pubKey)

	if wave.WaveType != WaveType_WAVE_TYPE_SURFACE {
		t.Errorf("WaveType = %v, want SURFACE", wave.WaveType)
	}
	if string(wave.Content) != "hello world" {
		t.Errorf("Content = %q, want %q", string(wave.Content), "hello world")
	}
	if wave.TtlSeconds != DefaultTTLSeconds {
		t.Errorf("TtlSeconds = %d, want %d", wave.TtlSeconds, DefaultTTLSeconds)
	}
	if wave.HopCount != 0 {
		t.Errorf("HopCount = %d, want 0", wave.HopCount)
	}
}

func TestCreateReplyWave(t *testing.T) {
	content := []byte("reply content")
	pubKey := make([]byte, 32)
	parentHash := make([]byte, 32)

	wave := CreateReplyWave(content, pubKey, parentHash)

	if wave.WaveType != WaveType_WAVE_TYPE_REPLY {
		t.Errorf("WaveType = %v, want REPLY", wave.WaveType)
	}
	if len(wave.ParentHash) != 32 {
		t.Errorf("ParentHash length = %d, want 32", len(wave.ParentHash))
	}
}

func TestCreateSpecterWave(t *testing.T) {
	content := []byte("anonymous content")

	wave := CreateSpecterWave(content)

	if wave.WaveType != WaveType_WAVE_TYPE_SPECTER {
		t.Errorf("WaveType = %v, want SPECTER", wave.WaveType)
	}
	if wave.AuthorPubkey != nil {
		t.Error("Specter wave should have nil AuthorPubkey")
	}
}

func TestCreateBeaconWave(t *testing.T) {
	content := []byte("beacon signal")
	pubKey := make([]byte, 32)

	wave := CreateBeaconWave(content, pubKey)

	if wave.WaveType != WaveType_WAVE_TYPE_BEACON {
		t.Errorf("WaveType = %v, want BEACON", wave.WaveType)
	}
}

func TestCreateHeartbeat(t *testing.T) {
	peerID := "12D3KooWTestPeerID"

	heartbeat := CreateHeartbeat(peerID)

	if heartbeat.PeerId != peerID {
		t.Errorf("PeerId = %q, want %q", heartbeat.PeerId, peerID)
	}
	if heartbeat.Timestamp == 0 {
		t.Error("Timestamp should not be 0")
	}
}

func TestRoundTrip(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	originalContent := []byte("round trip test content")
	wave := CreateSurfaceWave(originalContent, pubKey)

	// Create envelope
	env, err := NewWaveEnvelope(wave, privKey)
	if err != nil {
		t.Fatalf("NewWaveEnvelope() error: %v", err)
	}

	// Serialize
	data, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// Deserialize
	env2 := &MurmurEnvelope{}
	if err := proto.Unmarshal(data, env2); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	// Validate
	if err := ValidateEnvelope(env2); err != nil {
		t.Fatalf("ValidateEnvelope() error: %v", err)
	}

	// Unwrap
	wave2, err := UnwrapWave(env2)
	if err != nil {
		t.Fatalf("UnwrapWave() error: %v", err)
	}

	if string(wave2.Content) != string(originalContent) {
		t.Errorf("Content after round-trip = %q, want %q", string(wave2.Content), string(originalContent))
	}
}
