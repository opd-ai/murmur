package propagation

import (
	"context"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

func createTestWave(t *testing.T) *pb.Wave {
	t.Helper()

	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	opts := waves.DefaultCreateOptions()
	opts.Difficulty = 8 // Low difficulty for tests

	wave, err := waves.Create(waves.TypeSurface, []byte("test content"), kp, opts)
	if err != nil {
		t.Fatalf("waves.Create failed: %v", err)
	}

	return wave
}

func TestNewRelay(t *testing.T) {
	r := NewRelay()

	if r == nil {
		t.Fatal("NewRelay returned nil")
	}

	if r.maxHops != MaxHops {
		t.Errorf("maxHops = %d, want %d", r.maxHops, MaxHops)
	}

	if r.cacheTTL != DefaultCacheDuration {
		t.Errorf("cacheTTL = %v, want %v", r.cacheTTL, DefaultCacheDuration)
	}
}

func TestNewRelayWithConfig(t *testing.T) {
	cfg := RelayConfig{
		MaxHops:  5,
		CacheTTL: 1 * time.Hour,
	}

	r := NewRelayWithConfig(cfg)

	if r.maxHops != 5 {
		t.Errorf("maxHops = %d, want 5", r.maxHops)
	}

	if r.cacheTTL != 1*time.Hour {
		t.Errorf("cacheTTL = %v, want 1h", r.cacheTTL)
	}
}

func TestRelayReceive(t *testing.T) {
	r := NewRelay()
	wave := createTestWave(t)

	relayed, err := r.Receive(wave)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if relayed == nil {
		t.Fatal("relayed wave is nil")
	}

	// Hop count should be incremented.
	if relayed.HopCount != wave.HopCount+1 {
		t.Errorf("hop count = %d, want %d", relayed.HopCount, wave.HopCount+1)
	}
}

func TestRelayReceiveDuplicate(t *testing.T) {
	r := NewRelay()
	wave := createTestWave(t)

	// First receive should succeed.
	_, err := r.Receive(wave)
	if err != nil {
		t.Fatalf("first Receive failed: %v", err)
	}

	// Second receive should fail with duplicate error.
	_, err = r.Receive(wave)
	if err != ErrDuplicateWave {
		t.Errorf("expected ErrDuplicateWave, got %v", err)
	}
}

func TestRelayReceiveMaxHops(t *testing.T) {
	r := NewRelayWithConfig(RelayConfig{MaxHops: 5})
	wave := createTestWave(t)

	// Set hop count to max.
	wave.HopCount = 5

	_, err := r.Receive(wave)
	if err != ErrMaxHopsExceeded {
		t.Errorf("expected ErrMaxHopsExceeded, got %v", err)
	}
}

func TestRelayReceiveExpired(t *testing.T) {
	r := NewRelay()
	wave := createTestWave(t)

	// Set creation time far in the past.
	wave.CreatedAt = time.Now().Add(-8 * 24 * time.Hour).Unix()

	_, err := r.Receive(wave)
	if err != ErrExpiredWave {
		t.Errorf("expected ErrExpiredWave, got %v", err)
	}
}

func TestRelayReceiveNil(t *testing.T) {
	r := NewRelay()

	_, err := r.Receive(nil)
	if err != ErrInvalidWave {
		t.Errorf("expected ErrInvalidWave, got %v", err)
	}
}

func TestRelayHandler(t *testing.T) {
	handlerCalled := false
	var receivedWave *pb.Wave

	r := NewRelayWithConfig(RelayConfig{
		Handler: func(wave *pb.Wave) {
			handlerCalled = true
			receivedWave = wave
		},
	})

	wave := createTestWave(t)
	_, err := r.Receive(wave)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}

	if receivedWave == nil {
		t.Error("received wave is nil in handler")
	}
}

func TestRelayCleanExpired(t *testing.T) {
	r := NewRelayWithConfig(RelayConfig{
		CacheTTL: 100 * time.Millisecond,
	})

	// Receive some waves.
	for i := 0; i < 3; i++ {
		wave := createTestWave(t)
		r.Receive(wave)
	}

	if r.CacheSize() != 3 {
		t.Errorf("cache size = %d, want 3", r.CacheSize())
	}

	// Wait for cache to expire.
	time.Sleep(150 * time.Millisecond)

	cleaned := r.CleanExpired()
	if cleaned != 3 {
		t.Errorf("cleaned = %d, want 3", cleaned)
	}

	if r.CacheSize() != 0 {
		t.Errorf("cache size after clean = %d, want 0", r.CacheSize())
	}
}

func TestRelayStartCleanup(t *testing.T) {
	r := NewRelayWithConfig(RelayConfig{
		CacheTTL: 50 * time.Millisecond,
	})

	ctx := context.Background()
	cancel := r.StartCleanup(ctx, 50*time.Millisecond)
	defer cancel()

	// Receive a wave.
	wave := createTestWave(t)
	r.Receive(wave)

	if r.CacheSize() != 1 {
		t.Errorf("initial cache size = %d, want 1", r.CacheSize())
	}

	// Wait for cleanup to run.
	time.Sleep(150 * time.Millisecond)

	if r.CacheSize() != 0 {
		t.Errorf("cache size after cleanup = %d, want 0", r.CacheSize())
	}
}

func TestComputeWaveID(t *testing.T) {
	data := []byte("test data for wave ID")
	id := ComputeWaveID(data)

	if len(id) == 0 {
		t.Error("wave ID is empty")
	}

	// Same data should produce same ID.
	id2 := ComputeWaveID(data)
	if string(id) != string(id2) {
		t.Error("same data produced different IDs")
	}

	// Different data should produce different ID.
	id3 := ComputeWaveID([]byte("different data"))
	if string(id) == string(id3) {
		t.Error("different data produced same ID")
	}
}

func TestRelayConcurrent(t *testing.T) {
	r := NewRelay()

	// Create multiple waves.
	var waveList []*pb.Wave
	for i := 0; i < 10; i++ {
		waveList = append(waveList, createTestWave(t))
	}

	// Receive concurrently.
	done := make(chan bool)
	for _, w := range waveList {
		go func(wave *pb.Wave) {
			r.Receive(wave)
			done <- true
		}(w)
	}

	// Wait for all goroutines.
	for i := 0; i < 10; i++ {
		<-done
	}

	// All waves should be in cache.
	if r.CacheSize() != 10 {
		t.Errorf("cache size = %d, want 10", r.CacheSize())
	}
}
