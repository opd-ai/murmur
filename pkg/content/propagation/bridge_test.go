package propagation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

// TestBridgeCreation tests bridge initialization.
func TestBridgeCreation(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		bridge := NewBridge(BridgeConfig{})
		if bridge == nil {
			t.Fatal("NewBridge returned nil")
		}
		if bridge.IsEnabled() {
			t.Error("bridge should be disabled by default")
		}
	})

	t.Run("enabled config", func(t *testing.T) {
		bridge := NewBridge(BridgeConfig{Enabled: true})
		if !bridge.IsEnabled() {
			t.Error("bridge should be enabled")
		}
	})

	t.Run("with publishers", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		if bridge == nil {
			t.Fatal("NewBridgeWithPublishers returned nil")
		}
		if !bridge.IsEnabled() {
			t.Error("bridge should be enabled by default with publishers")
		}
	})
}

// TestBridgeEnableDisable tests enabling and disabling the bridge.
func TestBridgeEnableDisable(t *testing.T) {
	bridge := NewBridge(BridgeConfig{})

	bridge.SetEnabled(true)
	if !bridge.IsEnabled() {
		t.Error("bridge should be enabled")
	}

	bridge.SetEnabled(false)
	if bridge.IsEnabled() {
		t.Error("bridge should be disabled")
	}
}

// createTestVeiledWave creates a Veiled Wave for testing.
func createTestVeiledWave(id string) *pb.Wave {
	return &pb.Wave{
		WaveId:       []byte(id),
		WaveType:     pb.WaveType_WAVE_TYPE_VEILED,
		AuthorPubkey: make([]byte, 32),
		Content:      []byte("test veiled content"),
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   3600,
	}
}

// createTestSurfaceWave creates a Surface Wave for testing.
func createTestSurfaceWave(id string) *pb.Wave {
	return &pb.Wave{
		WaveId:       []byte(id),
		WaveType:     pb.WaveType_WAVE_TYPE_SURFACE,
		AuthorPubkey: make([]byte, 32),
		Content:      []byte("test surface content"),
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   3600,
	}
}

// TestInjectToSurface tests injecting Veiled Waves to Surface Layer.
func TestInjectToSurface(t *testing.T) {
	t.Run("successful injection", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		wave := createTestVeiledWave("test-wave-1")

		err := bridge.InjectToSurface(context.Background(), wave)
		if err != nil {
			t.Fatalf("InjectToSurface failed: %v", err)
		}

		if len(surfacePub.Published()) != 1 {
			t.Errorf("expected 1 published message, got %d", len(surfacePub.Published()))
		}

		stats := bridge.Stats()
		if stats.InjectedToSurface != 1 {
			t.Errorf("InjectedToSurface = %d, want 1", stats.InjectedToSurface)
		}
	})

	t.Run("bridge disabled", func(t *testing.T) {
		bridge := NewBridge(BridgeConfig{Enabled: false})
		wave := createTestVeiledWave("test-wave-2")

		err := bridge.InjectToSurface(context.Background(), wave)
		if err != ErrBridgeDisabled {
			t.Errorf("expected ErrBridgeDisabled, got %v", err)
		}
	})

	t.Run("not veiled wave", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		wave := createTestSurfaceWave("test-wave-3")

		err := bridge.InjectToSurface(context.Background(), wave)
		if err != ErrNotVeiledWave {
			t.Errorf("expected ErrNotVeiledWave, got %v", err)
		}

		stats := bridge.Stats()
		if stats.InvalidWaves != 1 {
			t.Errorf("InvalidWaves = %d, want 1", stats.InvalidWaves)
		}
	})

	t.Run("duplicate wave", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		wave := createTestVeiledWave("test-wave-4")

		// First injection should succeed.
		if err := bridge.InjectToSurface(context.Background(), wave); err != nil {
			t.Fatalf("first injection failed: %v", err)
		}

		// Second injection should be duplicate.
		err := bridge.InjectToSurface(context.Background(), wave)
		if err != ErrDuplicateWave {
			t.Errorf("expected ErrDuplicateWave, got %v", err)
		}

		stats := bridge.Stats()
		if stats.DuplicatesSkipped != 1 {
			t.Errorf("DuplicatesSkipped = %d, want 1", stats.DuplicatesSkipped)
		}
	})

	t.Run("no publisher", func(t *testing.T) {
		bridge := NewBridge(BridgeConfig{Enabled: true})
		wave := createTestVeiledWave("test-wave-5")

		err := bridge.InjectToSurface(context.Background(), wave)
		if err != ErrNoSurfacePublisher {
			t.Errorf("expected ErrNoSurfacePublisher, got %v", err)
		}
	})
}

// TestInjectToAnonymous tests injecting Veiled Waves to Anonymous Layer.
func TestInjectToAnonymous(t *testing.T) {
	t.Run("successful injection", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		wave := createTestVeiledWave("test-wave-anon-1")

		err := bridge.InjectToAnonymous(context.Background(), wave)
		if err != nil {
			t.Fatalf("InjectToAnonymous failed: %v", err)
		}

		if len(anonPub.Published()) != 1 {
			t.Errorf("expected 1 published message, got %d", len(anonPub.Published()))
		}

		stats := bridge.Stats()
		if stats.InjectedToAnonymous != 1 {
			t.Errorf("InjectedToAnonymous = %d, want 1", stats.InjectedToAnonymous)
		}
	})

	t.Run("no publisher", func(t *testing.T) {
		bridge := NewBridge(BridgeConfig{
			Enabled:          true,
			SurfacePublisher: NewMockPublisher(TopicSurfaceWaves),
			// No anonymous publisher
		})
		wave := createTestVeiledWave("test-wave-anon-2")

		err := bridge.InjectToAnonymous(context.Background(), wave)
		if err != ErrNoAnonymousPublisher {
			t.Errorf("expected ErrNoAnonymousPublisher, got %v", err)
		}
	})
}

// TestProcessAnonymousWave tests processing Waves from Anonymous Layer.
func TestProcessAnonymousWave(t *testing.T) {
	surfacePub := NewMockPublisher(TopicSurfaceWaves)
	anonPub := NewMockPublisher(TopicAnonymousWaves)

	bridge := NewBridgeWithPublishers(surfacePub, anonPub)

	t.Run("veiled wave is bridged", func(t *testing.T) {
		wave := createTestVeiledWave("process-anon-1")
		err := bridge.ProcessAnonymousWave(context.Background(), wave)
		if err != nil {
			t.Fatalf("ProcessAnonymousWave failed: %v", err)
		}

		if len(surfacePub.Published()) != 1 {
			t.Errorf("veiled wave should be bridged to surface")
		}
	})

	t.Run("non-veiled wave is not bridged", func(t *testing.T) {
		surfacePub.Clear()
		wave := createTestSurfaceWave("process-anon-2")
		wave.WaveType = pb.WaveType_WAVE_TYPE_SPECTER // Specter Wave

		err := bridge.ProcessAnonymousWave(context.Background(), wave)
		if err != nil {
			t.Fatalf("ProcessAnonymousWave failed: %v", err)
		}

		if len(surfacePub.Published()) != 0 {
			t.Error("non-veiled wave should not be bridged")
		}
	})
}

// TestProcessSurfaceWave tests processing Waves from Surface Layer.
func TestProcessSurfaceWave(t *testing.T) {
	surfacePub := NewMockPublisher(TopicSurfaceWaves)
	anonPub := NewMockPublisher(TopicAnonymousWaves)

	bridge := NewBridgeWithPublishers(surfacePub, anonPub)

	t.Run("veiled wave is bridged", func(t *testing.T) {
		wave := createTestVeiledWave("process-surface-1")
		err := bridge.ProcessSurfaceWave(context.Background(), wave)
		if err != nil {
			t.Fatalf("ProcessSurfaceWave failed: %v", err)
		}

		if len(anonPub.Published()) != 1 {
			t.Errorf("veiled wave should be bridged to anonymous")
		}
	})

	t.Run("non-veiled wave is not bridged", func(t *testing.T) {
		anonPub.Clear()
		wave := createTestSurfaceWave("process-surface-2")

		err := bridge.ProcessSurfaceWave(context.Background(), wave)
		if err != nil {
			t.Fatalf("ProcessSurfaceWave failed: %v", err)
		}

		if len(anonPub.Published()) != 0 {
			t.Error("non-veiled wave should not be bridged")
		}
	})
}

// TestBridgeRateLimiting tests rate limiting functionality.
func TestBridgeRateLimiting(t *testing.T) {
	surfacePub := NewMockPublisher(TopicSurfaceWaves)
	anonPub := NewMockPublisher(TopicAnonymousWaves)

	bridge := NewBridge(BridgeConfig{
		Enabled:            true,
		SurfacePublisher:   surfacePub,
		AnonymousPublisher: anonPub,
		MaxBridgeRate:      2.0, // 2 per second
	})

	// Rapidly inject waves.
	rateLimited := 0
	for i := 0; i < 10; i++ {
		wave := createTestVeiledWave("rate-test-" + string(rune('0'+i)))
		err := bridge.InjectToSurface(context.Background(), wave)
		// F-ERR-5 fix: Use errors.Is instead of == for sentinel error comparison
		if errors.Is(err, ErrRateLimited) {
			rateLimited++
		}
	}

	if rateLimited == 0 {
		t.Error("expected some requests to be rate limited")
	}

	stats := bridge.Stats()
	if stats.RateLimited == 0 {
		t.Error("RateLimited stat should be > 0")
	}
}

// TestBridgeDeduplication tests the injection cache.
func TestBridgeDeduplication(t *testing.T) {
	bridge := NewBridge(BridgeConfig{
		Enabled:          true,
		SurfacePublisher: NewMockPublisher(TopicSurfaceWaves),
		DeduplicationTTL: 100 * time.Millisecond,
	})

	wave := createTestVeiledWave("dedup-test")

	// First injection.
	if err := bridge.InjectToSurface(context.Background(), wave); err != nil {
		t.Fatalf("first injection failed: %v", err)
	}

	if bridge.InjectionCacheSize() != 1 {
		t.Errorf("InjectionCacheSize = %d, want 1", bridge.InjectionCacheSize())
	}

	// Wait for TTL to expire.
	time.Sleep(150 * time.Millisecond)

	// Clean expired entries.
	cleaned := bridge.CleanExpiredInjections()
	if cleaned != 1 {
		t.Errorf("CleanExpiredInjections = %d, want 1", cleaned)
	}

	if bridge.InjectionCacheSize() != 0 {
		t.Errorf("InjectionCacheSize after cleanup = %d, want 0", bridge.InjectionCacheSize())
	}

	// Now the same wave can be injected again.
	if err := bridge.InjectToSurface(context.Background(), wave); err != nil {
		t.Fatalf("re-injection failed: %v", err)
	}
}

// TestBridgeCleanupGoroutine tests the automatic cleanup.
func TestBridgeCleanupGoroutine(t *testing.T) {
	bridge := NewBridge(BridgeConfig{
		Enabled:          true,
		SurfacePublisher: NewMockPublisher(TopicSurfaceWaves),
		DeduplicationTTL: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup with short interval.
	stopCleanup := bridge.StartCleanup(ctx, 60*time.Millisecond)
	defer stopCleanup()

	// Inject a wave.
	wave := createTestVeiledWave("cleanup-goroutine-test")
	if err := bridge.InjectToSurface(context.Background(), wave); err != nil {
		t.Fatalf("injection failed: %v", err)
	}

	if bridge.InjectionCacheSize() != 1 {
		t.Error("wave should be in cache")
	}

	// Wait for cleanup to run.
	time.Sleep(150 * time.Millisecond)

	if bridge.InjectionCacheSize() != 0 {
		t.Error("cleanup should have removed expired entry")
	}
}

// TestBridgeRelay tests the combined BridgeRelay.
// Note: BridgeRelay calls Relay.Receive which validates waves.
// For unit tests, we test the Bridge component directly (above).
// This test verifies the integration path works.
func TestBridgeRelay(t *testing.T) {
	t.Run("structure", func(t *testing.T) {
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		br := NewBridgeRelay(surfacePub, anonPub)
		if br == nil {
			t.Fatal("NewBridgeRelay returned nil")
		}
		if br.Relay == nil {
			t.Error("BridgeRelay should have embedded Relay")
		}
		if br.Bridge == nil {
			t.Error("BridgeRelay should have embedded Bridge")
		}
		if !br.Bridge.IsEnabled() {
			t.Error("Bridge should be enabled")
		}
	})

	t.Run("bridge injection bypassing relay validation", func(t *testing.T) {
		// Test the bridging logic directly through Bridge, not BridgeRelay.
		// BridgeRelay adds relay validation which requires full wave structure.
		surfacePub := NewMockPublisher(TopicSurfaceWaves)
		anonPub := NewMockPublisher(TopicAnonymousWaves)

		bridge := NewBridgeWithPublishers(surfacePub, anonPub)
		wave := createTestVeiledWave("direct-bridge-test")

		// This tests the bridging without relay validation
		err := bridge.InjectToSurface(context.Background(), wave)
		if err != nil {
			t.Fatalf("InjectToSurface failed: %v", err)
		}

		if len(surfacePub.Published()) != 1 {
			t.Error("wave should be bridged to surface")
		}
	})
}

// TestBridgeConcurrency tests concurrent bridge operations.
func TestBridgeConcurrency(t *testing.T) {
	surfacePub := NewMockPublisher(TopicSurfaceWaves)
	anonPub := NewMockPublisher(TopicAnonymousWaves)

	bridge := NewBridgeWithPublishers(surfacePub, anonPub)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errCount atomic.Int32

	// Launch concurrent injections.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			wave := createTestVeiledWave("concurrent-" + string(rune(idx)))
			err := bridge.InjectToSurface(context.Background(), wave)
			if err == nil {
				successCount.Add(1)
			} else {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All unique waves should succeed (no duplicates since unique IDs).
	if successCount.Load() != 100 {
		t.Errorf("successCount = %d, want 100", successCount.Load())
	}
}

// TestIsVeiledWave tests the Veiled Wave detection.
func TestIsVeiledWave(t *testing.T) {
	tests := []struct {
		name     string
		wave     *pb.Wave
		expected bool
	}{
		{"nil wave", nil, false},
		{"veiled wave", &pb.Wave{WaveType: pb.WaveType_WAVE_TYPE_VEILED}, true},
		{"surface wave", &pb.Wave{WaveType: pb.WaveType_WAVE_TYPE_SURFACE}, false},
		{"reply wave", &pb.Wave{WaveType: pb.WaveType_WAVE_TYPE_REPLY}, false},
		{"specter wave", &pb.Wave{WaveType: pb.WaveType_WAVE_TYPE_SPECTER}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVeiledWave(tt.wave)
			if result != tt.expected {
				t.Errorf("isVeiledWave() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBridgeSetPublishers tests setting publishers dynamically.
func TestBridgeSetPublishers(t *testing.T) {
	bridge := NewBridge(BridgeConfig{Enabled: true})

	// Initially no publishers.
	wave := createTestVeiledWave("set-pub-test")
	err := bridge.InjectToSurface(context.Background(), wave)
	if err != ErrNoSurfacePublisher {
		t.Errorf("expected ErrNoSurfacePublisher, got %v", err)
	}

	// Set surface publisher.
	surfacePub := NewMockPublisher(TopicSurfaceWaves)
	bridge.SetSurfacePublisher(surfacePub)

	wave2 := createTestVeiledWave("set-pub-test-2")
	err = bridge.InjectToSurface(context.Background(), wave2)
	if err != nil {
		t.Fatalf("injection failed after setting publisher: %v", err)
	}

	if len(surfacePub.Published()) != 1 {
		t.Error("wave should be published")
	}
}

// TestBridgeTopicConstants tests that topic constants are correct.
func TestBridgeTopicConstants(t *testing.T) {
	if TopicSurfaceWaves != "/murmur/surface/waves/1.0" {
		t.Errorf("TopicSurfaceWaves = %s, expected /murmur/surface/waves/1.0", TopicSurfaceWaves)
	}
	if TopicAnonymousWaves != "/murmur/anonymous/waves/1.0" {
		t.Errorf("TopicAnonymousWaves = %s, expected /murmur/anonymous/waves/1.0", TopicAnonymousWaves)
	}
}

// TestBridgeStatsSnapshot tests that Stats() returns a snapshot.
func TestBridgeStatsSnapshot(t *testing.T) {
	bridge := NewBridgeWithPublishers(
		NewMockPublisher(TopicSurfaceWaves),
		NewMockPublisher(TopicAnonymousWaves),
	)

	wave := createTestVeiledWave("stats-snapshot")
	bridge.InjectToSurface(context.Background(), wave)

	stats1 := bridge.Stats()
	stats2 := bridge.Stats()

	// Both snapshots should have same value.
	if stats1.InjectedToSurface != stats2.InjectedToSurface {
		t.Error("stats snapshots should be consistent")
	}

	// Inject another wave.
	wave2 := createTestVeiledWave("stats-snapshot-2")
	bridge.InjectToSurface(context.Background(), wave2)

	// Original snapshot should be unchanged.
	if stats1.InjectedToSurface != 1 {
		t.Error("original snapshot should be unchanged")
	}

	// New snapshot should reflect the change.
	stats3 := bridge.Stats()
	if stats3.InjectedToSurface != 2 {
		t.Errorf("new snapshot should have 2, got %d", stats3.InjectedToSurface)
	}
}
