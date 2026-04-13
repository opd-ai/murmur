package proto

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestWaveRoundTrip verifies Wave messages marshal and unmarshal correctly.
func TestWaveRoundTrip(t *testing.T) {
	original := &Wave{
		WaveType:     WaveType_WAVE_TYPE_SURFACE,
		Content:      []byte("Hello, MURMUR network!"),
		AuthorPubkey: make([]byte, 32),
		Signature:    make([]byte, 64),
		CreatedAt:    1714500000,
		TtlSeconds:   604800, // 7 days
		PowNonce:     12345,
		ParentHash:   nil,
		HopCount:     0,
		WaveId:       make([]byte, 32),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &Wave{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetWaveType() != original.GetWaveType() {
		t.Errorf("WaveType = %v, want %v", decoded.GetWaveType(), original.GetWaveType())
	}
	if string(decoded.GetContent()) != string(original.GetContent()) {
		t.Errorf("Content = %q, want %q", decoded.GetContent(), original.GetContent())
	}
	if decoded.GetTtlSeconds() != original.GetTtlSeconds() {
		t.Errorf("TtlSeconds = %d, want %d", decoded.GetTtlSeconds(), original.GetTtlSeconds())
	}
	if decoded.GetPowNonce() != original.GetPowNonce() {
		t.Errorf("PowNonce = %d, want %d", decoded.GetPowNonce(), original.GetPowNonce())
	}
	if decoded.GetHopCount() != original.GetHopCount() {
		t.Errorf("HopCount = %d, want %d", decoded.GetHopCount(), original.GetHopCount())
	}
	if decoded.GetCreatedAt() != original.GetCreatedAt() {
		t.Errorf("CreatedAt = %d, want %d", decoded.GetCreatedAt(), original.GetCreatedAt())
	}
}

// TestMurmurEnvelopeRoundTrip verifies envelope messages marshal and unmarshal correctly.
func TestMurmurEnvelopeRoundTrip(t *testing.T) {
	waveData, _ := proto.Marshal(&Wave{
		WaveType:   WaveType_WAVE_TYPE_SURFACE,
		Content:    []byte("Test content"),
		TtlSeconds: 604800,
	})

	original := &MurmurEnvelope{
		Version:       1,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       waveData,
		SenderPubkey:  make([]byte, 32),
		Signature:     make([]byte, 64),
		TimestampUnix: 1714500000,
		MessageId:     make([]byte, 32),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &MurmurEnvelope{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetVersion() != original.GetVersion() {
		t.Errorf("Version = %d, want %d", decoded.GetVersion(), original.GetVersion())
	}
	if decoded.GetType() != original.GetType() {
		t.Errorf("Type = %v, want %v", decoded.GetType(), original.GetType())
	}
	if decoded.GetTimestampUnix() != original.GetTimestampUnix() {
		t.Errorf("TimestampUnix = %d, want %d", decoded.GetTimestampUnix(), original.GetTimestampUnix())
	}
}

// TestIdentityDeclarationRoundTrip verifies identity messages marshal correctly.
func TestIdentityDeclarationRoundTrip(t *testing.T) {
	original := &IdentityDeclaration{
		PublicKey:   make([]byte, 32),
		DisplayName: "shadow_walker_42",
		Bio:         "A mysterious traveler",
		CreatedAt:   1714500000,
		Version:     1,
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &IdentityDeclaration{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetDisplayName() != original.GetDisplayName() {
		t.Errorf("DisplayName = %q, want %q", decoded.GetDisplayName(), original.GetDisplayName())
	}
	if decoded.GetCreatedAt() != original.GetCreatedAt() {
		t.Errorf("CreatedAt = %d, want %d", decoded.GetCreatedAt(), original.GetCreatedAt())
	}
}

// TestHeartbeatRoundTrip verifies heartbeat messages marshal correctly.
func TestHeartbeatRoundTrip(t *testing.T) {
	original := &Heartbeat{
		Timestamp: 1714500000,
		Signature: make([]byte, 64),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &Heartbeat{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetTimestamp() != original.GetTimestamp() {
		t.Errorf("Timestamp = %d, want %d", decoded.GetTimestamp(), original.GetTimestamp())
	}
}

// TestMessageTypeEnumValues verifies enum values are correct.
func TestMessageTypeEnumValues(t *testing.T) {
	// Verify expected enum values exist.
	types := []MessageType{
		MessageType_MESSAGE_TYPE_UNSPECIFIED,
		MessageType_MESSAGE_TYPE_WAVE,
		MessageType_MESSAGE_TYPE_IDENTITY,
		MessageType_MESSAGE_TYPE_SHROUD_AD,
		MessageType_MESSAGE_TYPE_HEARTBEAT,
	}

	for _, mt := range types {
		// String() should not panic.
		_ = mt.String()
	}
}

// TestWaveTypeEnumValues verifies enum values are correct.
func TestWaveTypeEnumValues(t *testing.T) {
	types := []WaveType{
		WaveType_WAVE_TYPE_UNSPECIFIED,
		WaveType_WAVE_TYPE_SURFACE,
		WaveType_WAVE_TYPE_REPLY,
		WaveType_WAVE_TYPE_VEILED,
		WaveType_WAVE_TYPE_SPECTER,
		WaveType_WAVE_TYPE_SIGIL,
		WaveType_WAVE_TYPE_ABYSSAL,
		WaveType_WAVE_TYPE_MASKED,
		WaveType_WAVE_TYPE_BEACON,
	}

	for _, wt := range types {
		_ = wt.String()
	}
}

// TestRelayAdvertisementRoundTrip verifies relay ad messages marshal correctly.
func TestRelayAdvertisementRoundTrip(t *testing.T) {
	original := &RelayAdvertisement{
		Curve25519Pubkey: make([]byte, 32),
		Ed25519Pubkey:    make([]byte, 32),
		Addrs:            []string{"/ip4/127.0.0.1/tcp/4001"},
		Roles:            []RelayRole{RelayRole_RELAY_ROLE_ENTRY},
		Bandwidth:        10 * 1024 * 1024, // 10 MB/s
		Timestamp:        1714500000,
		Signature:        make([]byte, 64),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &RelayAdvertisement{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetBandwidth() != original.GetBandwidth() {
		t.Errorf("Bandwidth = %d, want %d", decoded.GetBandwidth(), original.GetBandwidth())
	}
	if len(decoded.GetAddrs()) != len(original.GetAddrs()) {
		t.Errorf("Addrs len = %d, want %d", len(decoded.GetAddrs()), len(original.GetAddrs()))
	}
	if decoded.GetTimestamp() != original.GetTimestamp() {
		t.Errorf("Timestamp = %d, want %d", decoded.GetTimestamp(), original.GetTimestamp())
	}
}

// TestOnionCellRoundTrip verifies onion cell messages marshal correctly.
func TestOnionCellRoundTrip(t *testing.T) {
	original := &OnionCell{
		CircuitId:        make([]byte, 16),
		CellType:         OnionCellType_ONION_CELL_TYPE_RELAY,
		EncryptedPayload: make([]byte, 256),
		Nonce:            make([]byte, 24),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &OnionCell{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetCellType() != original.GetCellType() {
		t.Errorf("CellType = %v, want %v", decoded.GetCellType(), original.GetCellType())
	}
	if len(decoded.GetCircuitId()) != len(original.GetCircuitId()) {
		t.Errorf("CircuitId len = %d, want %d", len(decoded.GetCircuitId()), len(original.GetCircuitId()))
	}
}

// TestResonanceScoreRoundTrip verifies resonance messages marshal correctly.
func TestResonanceScoreRoundTrip(t *testing.T) {
	original := &ResonanceScore{
		SpecterIdHash: make([]byte, 32),
		Score:         150.5,
		Milestone:     ResonanceMilestone_RESONANCE_MILESTONE_PHANTOM,
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &ResonanceScore{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.GetScore() != original.GetScore() {
		t.Errorf("Score = %f, want %f", decoded.GetScore(), original.GetScore())
	}
	if decoded.GetMilestone() != original.GetMilestone() {
		t.Errorf("Milestone = %v, want %v", decoded.GetMilestone(), original.GetMilestone())
	}
}

// TestEmptyMessages verifies empty messages marshal and unmarshal without errors.
func TestEmptyMessages(t *testing.T) {
	messages := []proto.Message{
		&Wave{},
		&MurmurEnvelope{},
		&IdentityDeclaration{},
		&Heartbeat{},
		&RelayAdvertisement{},
		&OnionCell{},
		&ResonanceScore{},
	}

	for _, msg := range messages {
		data, err := proto.Marshal(msg)
		if err != nil {
			t.Errorf("Marshal empty %T failed: %v", msg, err)
			continue
		}

		// Should be able to unmarshal back to a new instance.
		newMsg := proto.Clone(msg)
		proto.Reset(newMsg)
		if err := proto.Unmarshal(data, newMsg); err != nil {
			t.Errorf("Unmarshal empty %T failed: %v", msg, err)
		}
	}
}
