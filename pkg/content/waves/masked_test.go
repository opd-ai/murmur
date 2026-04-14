package waves

import (
	"bytes"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestGenerateMaskedKeypair(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	// Verify public key is set.
	if len(mk.PublicKey()) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(mk.PublicKey()))
	}

	// Verify pseudonym is generated.
	if mk.Pseudonym() == "" {
		t.Error("Pseudonym is empty")
	}

	// Verify key hash is generated.
	if len(mk.KeyHash()) != 32 {
		t.Errorf("KeyHash length = %d, want 32", len(mk.KeyHash()))
	}

	// Verify not disposed.
	if mk.IsDisposed() {
		t.Error("New keypair should not be disposed")
	}
}

func TestMaskedKeypairSign(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	data := []byte("test message")
	sig := mk.Sign(data)

	if len(sig) != 64 {
		t.Errorf("Signature length = %d, want 64", len(sig))
	}
}

func TestMaskedKeypairDispose(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	// Dispose the keypair.
	mk.Dispose()

	// Verify disposed state.
	if !mk.IsDisposed() {
		t.Error("Keypair should be disposed")
	}

	// Verify signing returns nil.
	sig := mk.Sign([]byte("test"))
	if sig != nil {
		t.Error("Sign() should return nil after dispose")
	}

	// Verify public key returns nil.
	if mk.PublicKey() != nil {
		t.Error("PublicKey() should return nil after dispose")
	}

	// Double dispose should not panic.
	mk.Dispose()
}

func TestCreateMasked(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	content := []byte("Hello from Masked Event!")
	opts := DefaultMaskedOptions("event-123")
	opts.Difficulty = 1 // Low difficulty for fast tests

	wave, err := CreateMasked(content, mk, opts)
	if err != nil {
		t.Fatalf("CreateMasked() error = %v", err)
	}

	// Verify wave type.
	if wave.WaveType != pb.WaveType(TypeMasked) {
		t.Errorf("WaveType = %v, want %v", wave.WaveType, TypeMasked)
	}

	// Verify content.
	if !bytes.Equal(wave.Content, content) {
		t.Error("Content mismatch")
	}

	// Verify author public key.
	if !bytes.Equal(wave.AuthorPubkey, mk.PublicKey()) {
		t.Error("AuthorPubkey mismatch")
	}

	// Verify TTL is limited to MaskedTTL.
	expectedTTL := int64(MaskedTTL.Seconds())
	if wave.TtlSeconds != expectedTTL {
		t.Errorf("TtlSeconds = %d, want %d", wave.TtlSeconds, expectedTTL)
	}

	// Verify metadata.
	if !IsMasked(wave) {
		t.Error("IsMasked() = false, want true")
	}

	eventID := GetMaskedEventID(wave)
	if eventID != "event-123" {
		t.Errorf("EventID = %q, want %q", eventID, "event-123")
	}

	pseudonym := GetMaskedPseudonym(wave)
	if pseudonym != mk.Pseudonym() {
		t.Errorf("Pseudonym = %q, want %q", pseudonym, mk.Pseudonym())
	}

	keyHash := GetMaskedKeyHash(wave)
	if !bytes.Equal(keyHash, mk.KeyHash()) {
		t.Error("KeyHash mismatch")
	}
}

func TestCreateMaskedEnforcesTTL(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	// Try to create with longer TTL than allowed.
	opts := MaskedOptions{
		EventID:    "event-123",
		TTL:        30 * 24 * time.Hour, // 30 days
		Difficulty: 1,
	}

	wave, err := CreateMasked([]byte("test"), mk, opts)
	if err != nil {
		t.Fatalf("CreateMasked() error = %v", err)
	}

	// TTL should be capped at MaskedTTL.
	expectedTTL := int64(MaskedTTL.Seconds())
	if wave.TtlSeconds != expectedTTL {
		t.Errorf("TtlSeconds = %d, want %d (should be capped)", wave.TtlSeconds, expectedTTL)
	}
}

func TestCreateMaskedContentTooLarge(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	content := make([]byte, MaxContentSize+1)
	opts := DefaultMaskedOptions("event-123")

	_, err = CreateMasked(content, mk, opts)
	if err != ErrContentTooLarge {
		t.Errorf("Expected ErrContentTooLarge, got %v", err)
	}
}

func TestCreateMaskedMissingEventID(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	opts := MaskedOptions{
		EventID:    "", // Missing
		TTL:        MaskedTTL,
		Difficulty: 1,
	}

	_, err = CreateMasked([]byte("test"), mk, opts)
	if err != ErrMissingEventID {
		t.Errorf("Expected ErrMissingEventID, got %v", err)
	}
}

func TestCreateMaskedNilKeypair(t *testing.T) {
	opts := DefaultMaskedOptions("event-123")

	_, err := CreateMasked([]byte("test"), nil, opts)
	if err != ErrNilKeyPair {
		t.Errorf("Expected ErrNilKeyPair, got %v", err)
	}
}

func TestCreateMaskedDisposedKeypair(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	mk.Dispose()

	opts := DefaultMaskedOptions("event-123")
	_, err = CreateMasked([]byte("test"), mk, opts)
	if err != ErrMaskedKeyDisposed {
		t.Errorf("Expected ErrMaskedKeyDisposed, got %v", err)
	}
}

func TestIsMasked(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	opts := DefaultMaskedOptions("event-123")
	opts.Difficulty = 1

	wave, err := CreateMasked([]byte("test"), mk, opts)
	if err != nil {
		t.Fatalf("CreateMasked() error = %v", err)
	}

	if !IsMasked(wave) {
		t.Error("IsMasked() = false, want true")
	}

	// Test with non-Masked wave.
	surfaceWave := &pb.Wave{WaveType: pb.WaveType(TypeSurface)}
	if IsMasked(surfaceWave) {
		t.Error("IsMasked() = true for Surface wave, want false")
	}

	// Test with nil.
	if IsMasked(nil) {
		t.Error("IsMasked() = true for nil, want false")
	}

	// Test with Masked type but no metadata.
	noMetaWave := &pb.Wave{WaveType: pb.WaveType(TypeMasked)}
	if IsMasked(noMetaWave) {
		t.Error("IsMasked() = true for wave without event_id, want false")
	}
}

func TestValidateMasked(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	opts := DefaultMaskedOptions("event-123")
	opts.Difficulty = 1

	wave, err := CreateMasked([]byte("test"), mk, opts)
	if err != nil {
		t.Fatalf("CreateMasked() error = %v", err)
	}

	// Validation should pass.
	if err := ValidateMasked(wave, 1); err != nil {
		t.Errorf("ValidateMasked() error = %v", err)
	}
}

func TestValidateMaskedNil(t *testing.T) {
	err := ValidateMasked(nil, 1)
	if err != ErrInvalidMaskedWave {
		t.Errorf("Expected ErrInvalidMaskedWave, got %v", err)
	}
}

func TestValidateMaskedWrongType(t *testing.T) {
	wave := &pb.Wave{
		WaveType: pb.WaveType(TypeSurface),
		Metadata: map[string][]byte{MetaMaskedEventID: []byte("event-123")},
	}

	err := ValidateMasked(wave, 1)
	if err != ErrNotMaskedWave {
		t.Errorf("Expected ErrNotMaskedWave, got %v", err)
	}
}

func TestValidateMaskedMissingEventID(t *testing.T) {
	wave := &pb.Wave{
		WaveType: pb.WaveType(TypeMasked),
		Metadata: map[string][]byte{},
	}

	err := ValidateMasked(wave, 1)
	if err != ErrMissingEventID {
		t.Errorf("Expected ErrMissingEventID, got %v", err)
	}
}

func TestCreateMaskedReply(t *testing.T) {
	mk, err := GenerateMaskedKeypair()
	if err != nil {
		t.Fatalf("GenerateMaskedKeypair() error = %v", err)
	}

	// Create parent wave.
	parentContent := []byte("Parent message")
	opts := DefaultMaskedOptions("event-123")
	opts.Difficulty = 1

	parent, err := CreateMasked(parentContent, mk, opts)
	if err != nil {
		t.Fatalf("CreateMasked() error = %v", err)
	}

	// Create reply using the same keypair (same participant).
	mk2, _ := GenerateMaskedKeypair()
	replyContent := []byte("Reply message")

	// We need to set difficulty before calling CreateMaskedReply.
	replyOpts := DefaultMaskedOptions("event-123")
	replyOpts.Difficulty = 1
	replyOpts.ParentHash = parent.WaveId

	reply, err := CreateMasked(replyContent, mk2, replyOpts)
	if err != nil {
		t.Fatalf("CreateMasked reply error = %v", err)
	}

	// Verify reply references parent.
	if !bytes.Equal(reply.ParentHash, parent.WaveId) {
		t.Error("Reply should reference parent WaveId")
	}

	// Verify same event ID.
	if GetMaskedEventID(reply) != "event-123" {
		t.Error("Reply should have same event ID")
	}
}

func TestMaskedEventParticipant(t *testing.T) {
	eventEnd := time.Now().Add(1 * time.Hour)
	participant, err := NewMaskedEventParticipant("event-456", eventEnd)
	if err != nil {
		t.Fatalf("NewMaskedEventParticipant() error = %v", err)
	}

	// Verify pseudonym.
	if participant.Pseudonym() == "" {
		t.Error("Pseudonym is empty")
	}

	// Verify event not expired.
	if participant.IsEventExpired() {
		t.Error("Event should not be expired")
	}

	// Leave event.
	participant.LeaveEvent()

	// Verify keypair is disposed.
	if !participant.Keypair.IsDisposed() {
		t.Error("Keypair should be disposed after leaving")
	}
}

func TestMaskedEventParticipantExpired(t *testing.T) {
	// Create participant for expired event.
	eventEnd := time.Now().Add(-1 * time.Hour)
	participant, err := NewMaskedEventParticipant("event-789", eventEnd)
	if err != nil {
		t.Fatalf("NewMaskedEventParticipant() error = %v", err)
	}

	// Verify event is expired.
	if !participant.IsEventExpired() {
		t.Error("Event should be expired")
	}

	// Attempting to create wave should fail.
	_, err = participant.CreateWave([]byte("test"))
	if err != ErrMaskedEventExpired {
		t.Errorf("Expected ErrMaskedEventExpired, got %v", err)
	}
}

func TestGenerateMaskedPseudonym(t *testing.T) {
	// Test different key hashes produce different pseudonyms.
	hash1 := make([]byte, 32)
	hash1[0] = 0x00
	hash1[1] = 0x01
	hash1[2] = 0x02
	hash1[3] = 0x03

	hash2 := make([]byte, 32)
	hash2[0] = 0xFF
	hash2[1] = 0xFE
	hash2[2] = 0xFD
	hash2[3] = 0xFC

	pseudonym1 := generateMaskedPseudonym(hash1)
	pseudonym2 := generateMaskedPseudonym(hash2)

	if pseudonym1 == pseudonym2 {
		t.Error("Different hashes should produce different pseudonyms")
	}

	// Verify format (two words separated by space).
	if len(pseudonym1) == 0 {
		t.Error("Pseudonym should not be empty")
	}
}

func TestGenerateMaskedPseudonymInvalidHash(t *testing.T) {
	// Test with too short hash.
	shortHash := []byte{0x01, 0x02}
	pseudonym := generateMaskedPseudonym(shortHash)

	if pseudonym != "Unknown Mask" {
		t.Errorf("Expected 'Unknown Mask' for short hash, got %q", pseudonym)
	}
}

func TestGetMaskedMetadataNil(t *testing.T) {
	// Test all getters with nil wave.
	if GetMaskedEventID(nil) != "" {
		t.Error("GetMaskedEventID(nil) should return empty")
	}
	if GetMaskedPseudonym(nil) != "" {
		t.Error("GetMaskedPseudonym(nil) should return empty")
	}
	if GetMaskedKeyHash(nil) != nil {
		t.Error("GetMaskedKeyHash(nil) should return nil")
	}

	// Test with wave without metadata.
	emptyWave := &pb.Wave{}
	if GetMaskedEventID(emptyWave) != "" {
		t.Error("GetMaskedEventID() should return empty for wave without metadata")
	}
}

func TestDefaultMaskedOptions(t *testing.T) {
	opts := DefaultMaskedOptions("test-event")

	if opts.EventID != "test-event" {
		t.Errorf("EventID = %q, want %q", opts.EventID, "test-event")
	}
	if opts.TTL != MaskedTTL {
		t.Errorf("TTL = %v, want %v", opts.TTL, MaskedTTL)
	}
}

func TestMaskedKeypairNilHandling(t *testing.T) {
	var mk *MaskedKeypair

	// All methods should handle nil gracefully.
	if mk.PublicKey() != nil {
		t.Error("PublicKey() should return nil for nil keypair")
	}
	if mk.Pseudonym() != "" {
		t.Error("Pseudonym() should return empty for nil keypair")
	}
	if mk.KeyHash() != nil {
		t.Error("KeyHash() should return nil for nil keypair")
	}
	if mk.Sign([]byte("test")) != nil {
		t.Error("Sign() should return nil for nil keypair")
	}
	if !mk.IsDisposed() {
		t.Error("IsDisposed() should return true for nil keypair")
	}

	// Dispose should not panic.
	mk.Dispose()
}

func TestMaskedEventParticipantNilHandling(t *testing.T) {
	var p *MaskedEventParticipant

	if p.Pseudonym() != "" {
		t.Error("Pseudonym() should return empty for nil participant")
	}
	if !p.IsEventExpired() {
		t.Error("IsEventExpired() should return true for nil participant")
	}

	// LeaveEvent should not panic.
	p.LeaveEvent()
}
