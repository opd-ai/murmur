package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

func TestBroadcastWave(t *testing.T) {
	// Create a temporary directory for the test.
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Start the app in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Wait for initialization.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// Create a Wave.
	kp := app.Subsystems().Identity
	wave, err := waves.CreateSurface([]byte("Test broadcast wave"), kp)
	if err != nil {
		t.Fatalf("CreateSurface failed: %v", err)
	}

	// Broadcast the Wave.
	err = app.BroadcastWave(ctx, wave)
	if err != nil {
		t.Errorf("BroadcastWave failed: %v", err)
	}

	// Verify Wave is in local cache.
	cachedWave, err := app.Subsystems().WaveCache.Get(wave.WaveId)
	if err != nil {
		t.Errorf("Wave not in cache: %v", err)
	}
	if cachedWave == nil {
		t.Error("Cached wave is nil")
	}

	// Clean up.
	app.Close()
}

func TestBroadcastWaveNilWave(t *testing.T) {
	// Create a temporary directory for the test.
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// Try to broadcast nil Wave.
	err = app.BroadcastWave(ctx, nil)
	if err != ErrNilWave {
		t.Errorf("expected ErrNilWave, got %v", err)
	}

	app.Close()
}

func TestBroadcastWaveNotInitialized(t *testing.T) {
	cfg := Config{
		DataDir: t.TempDir(),
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Try to broadcast without running the app.
	wave := &pb.Wave{
		WaveType: pb.WaveType_WAVE_TYPE_SURFACE,
		Content:  []byte("test"),
	}

	err = app.BroadcastWave(context.Background(), wave)
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got %v", err)
	}
}

func TestBroadcastHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// Broadcast a heartbeat.
	err = app.BroadcastHeartbeat(ctx)
	if err != nil {
		t.Errorf("BroadcastHeartbeat failed: %v", err)
	}

	app.Close()
}

func TestBroadcastIdentity(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// Create an identity declaration.
	decl := &pb.IdentityDeclaration{
		PublicKey:   app.Subsystems().Identity.PublicKey,
		DisplayName: "TestUser",
		Bio:         "Test bio",
		CreatedAt:   time.Now().Unix(),
		Version:     1,
		PrivacyMode: pb.PrivacyMode_PRIVACY_MODE_OPEN,
	}

	// Broadcast the declaration.
	err = app.BroadcastIdentity(ctx, decl)
	if err != nil {
		t.Errorf("BroadcastIdentity failed: %v", err)
	}

	// Verify signature was added.
	if len(decl.Signature) == 0 {
		t.Error("Declaration signature was not added")
	}

	app.Close()
}

func TestCreateSurfaceWave(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// Create and broadcast a surface wave.
	content := []byte("Hello from CreateSurfaceWave test!")
	wave, err := app.CreateSurfaceWave(ctx, content)
	if err != nil {
		t.Fatalf("CreateSurfaceWave failed: %v", err)
	}

	if wave == nil {
		t.Fatal("CreateSurfaceWave returned nil wave")
	}

	if wave.WaveType != pb.WaveType_WAVE_TYPE_SURFACE {
		t.Errorf("expected WAVE_TYPE_SURFACE, got %v", wave.WaveType)
	}

	if string(wave.Content) != string(content) {
		t.Errorf("content mismatch: expected %s, got %s", content, wave.Content)
	}

	// Verify Wave is in local cache.
	cachedWave, err := app.Subsystems().WaveCache.Get(wave.WaveId)
	if err != nil {
		t.Errorf("Wave not in cache: %v", err)
	}
	if cachedWave == nil {
		t.Error("Cached wave is nil")
	}

	app.Close()
}

func TestCreateReplyWave(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		DataDir: tmpDir,
		SkipUI:  true,
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady failed: %v", err)
	}

	// First create a parent wave.
	parentContent := []byte("Parent wave content")
	parentWave, err := app.CreateSurfaceWave(ctx, parentContent)
	if err != nil {
		t.Fatalf("CreateSurfaceWave for parent failed: %v", err)
	}

	// Create a reply.
	replyContent := []byte("This is a reply!")
	replyWave, err := app.CreateReplyWave(ctx, replyContent, parentWave.WaveId)
	if err != nil {
		t.Fatalf("CreateReplyWave failed: %v", err)
	}

	if replyWave == nil {
		t.Fatal("CreateReplyWave returned nil wave")
	}

	if replyWave.WaveType != pb.WaveType_WAVE_TYPE_REPLY {
		t.Errorf("expected WAVE_TYPE_REPLY, got %v", replyWave.WaveType)
	}

	if string(replyWave.ParentHash) != string(parentWave.WaveId) {
		t.Error("reply parent hash does not match parent wave ID")
	}

	app.Close()
}

func TestCreateSignedEnvelope(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	payload := []byte("test payload")

	envelope, err := createSignedEnvelope(pb.MessageType_MESSAGE_TYPE_WAVE, payload, kp)
	if err != nil {
		t.Fatalf("createSignedEnvelope failed: %v", err)
	}

	if envelope.Version != ProtocolVersion {
		t.Errorf("version mismatch: expected %d, got %d", ProtocolVersion, envelope.Version)
	}

	if envelope.Type != pb.MessageType_MESSAGE_TYPE_WAVE {
		t.Errorf("type mismatch: expected %v, got %v", pb.MessageType_MESSAGE_TYPE_WAVE, envelope.Type)
	}

	if string(envelope.Payload) != string(payload) {
		t.Error("payload mismatch")
	}

	if len(envelope.Signature) != 64 {
		t.Errorf("expected 64-byte signature, got %d bytes", len(envelope.Signature))
	}

	if len(envelope.MessageId) != 32 {
		t.Errorf("expected 32-byte message ID, got %d bytes", len(envelope.MessageId))
	}
}

func TestEnvelopeSignatureData(t *testing.T) {
	env := &pb.MurmurEnvelope{
		Version: 1,
		Type:    pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload: []byte("test"),
	}

	data := envelopeSignatureData(env)

	// Expected: 4 bytes version + 4 bytes type + payload
	expected := []byte{0, 0, 0, 1, 0, 0, 0, 1, 't', 'e', 's', 't'}
	if string(data) != string(expected) {
		t.Errorf("signature data mismatch: got %v, expected %v", data, expected)
	}
}

func TestIdentityDeclarationSignatureData(t *testing.T) {
	kp, _ := keys.GenerateKeyPair()
	decl := &pb.IdentityDeclaration{
		PublicKey:   kp.PublicKey,
		DisplayName: "Test",
		Bio:         "Bio",
		CreatedAt:   12345,
		Version:     1,
		PrivacyMode: pb.PrivacyMode_PRIVACY_MODE_OPEN,
	}

	data := identityDeclarationSignatureData(decl)

	// Should include all fields.
	if len(data) < 32 { // At minimum, public key is 32 bytes
		t.Errorf("signature data too short: %d bytes", len(data))
	}
}

func TestHeartbeatSignatureData(t *testing.T) {
	hb := &pb.Heartbeat{
		PeerId:    "test-peer",
		Timestamp: 12345,
	}

	data := heartbeatSignatureData(hb)

	// Should include peer ID (9 bytes) + timestamp (8 bytes) = 17 bytes
	if len(data) != 17 {
		t.Errorf("expected 17 bytes, got %d", len(data))
	}
}

// Cleanup helper to remove test directories.
func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Failed to cleanup test dir: %v", err)
	}
}

func mustTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func init() {
	// Ensure test temp directories are cleaned up.
	_ = filepath.Clean
}
