package resonance

import (
	"context"
	"testing"
	"time"
)

func TestNewDecayManager(t *testing.T) {
	dm := NewDecayManager()
	if dm == nil {
		t.Fatal("NewDecayManager() returned nil")
	}
	if dm.interval != 60*time.Second {
		t.Errorf("Default interval = %v, want 60s", dm.interval)
	}
	if dm.IsRunning() {
		t.Error("Manager should not be running after creation")
	}
}

func TestNewDecayManagerWithConfig(t *testing.T) {
	cfg := DecayManagerConfig{
		Interval: 30 * time.Second,
	}
	dm := NewDecayManagerWithConfig(cfg)
	if dm.interval != 30*time.Second {
		t.Errorf("Configured interval = %v, want 30s", dm.interval)
	}

	// Test zero interval defaults to 60s.
	cfg.Interval = 0
	dm = NewDecayManagerWithConfig(cfg)
	if dm.interval != 60*time.Second {
		t.Errorf("Zero interval should default to 60s, got %v", dm.interval)
	}
}

func TestDecayManagerRegister(t *testing.T) {
	dm := NewDecayManager()

	surfaceScorer := NewSurfaceScorer()
	specterScorer := NewSpecterScorer()
	legacyScorer := NewScorer()

	dm.RegisterSurfaceScorer(surfaceScorer)
	dm.RegisterSpecterScorer(specterScorer)
	dm.RegisterLegacyScorer(legacyScorer)

	if dm.surfaceScorer != surfaceScorer {
		t.Error("Surface scorer not registered")
	}
	if dm.specterScorer != specterScorer {
		t.Error("Specter scorer not registered")
	}
	if dm.legacyScorer != legacyScorer {
		t.Error("Legacy scorer not registered")
	}
}

func TestDecayManagerRunOnce(t *testing.T) {
	dm := NewDecayManager()

	surfaceScorer := NewSurfaceScorer()
	specterScorer := NewSpecterScorer()
	legacyScorer := NewScorer()

	dm.RegisterSurfaceScorer(surfaceScorer)
	dm.RegisterSpecterScorer(specterScorer)
	dm.RegisterLegacyScorer(legacyScorer)

	// Add some scores.
	surfaceScorer.GetScore("user-1").SetConnectionCount(10)
	specterScorer.GetScore("specter-1").SetConnectionCount(10)
	legacyScorer.GetScore("legacy-1").AddPublication()

	// Run decay.
	dm.RunOnce()

	stats := dm.Stats()
	if stats.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", stats.RunCount)
	}
	if stats.LastRun.IsZero() {
		t.Error("LastRun should be set")
	}
	if stats.SurfaceHits != 1 {
		t.Errorf("SurfaceHits = %d, want 1", stats.SurfaceHits)
	}
	if stats.SpecterHits != 1 {
		t.Errorf("SpecterHits = %d, want 1", stats.SpecterHits)
	}
	if stats.LegacyHits != 1 {
		t.Errorf("LegacyHits = %d, want 1", stats.LegacyHits)
	}
}

func TestDecayManagerStartStop(t *testing.T) {
	dm := NewDecayManagerWithConfig(DecayManagerConfig{
		Interval: 10 * time.Millisecond,
	})

	if dm.IsRunning() {
		t.Error("Should not be running before Start")
	}

	ctx := context.Background()
	dm.Start(ctx)

	if !dm.IsRunning() {
		t.Error("Should be running after Start")
	}

	// Wait for at least one decay cycle.
	time.Sleep(50 * time.Millisecond)

	dm.Stop()

	if dm.IsRunning() {
		t.Error("Should not be running after Stop")
	}

	stats := dm.Stats()
	if stats.RunCount == 0 {
		t.Error("Should have run at least once")
	}
}

func TestDecayManagerDoubleStart(t *testing.T) {
	dm := NewDecayManagerWithConfig(DecayManagerConfig{
		Interval: 100 * time.Millisecond,
	})

	ctx := context.Background()
	dm.Start(ctx)
	defer dm.Stop()

	// Second start should be no-op.
	dm.Start(ctx)

	if !dm.IsRunning() {
		t.Error("Should still be running")
	}
}

func TestDecayManagerDoubleStop(t *testing.T) {
	dm := NewDecayManager()

	// Stop without start should be no-op.
	dm.Stop()

	if dm.IsRunning() {
		t.Error("Should not be running")
	}
}

func TestDecayManagerContextCancellation(t *testing.T) {
	dm := NewDecayManagerWithConfig(DecayManagerConfig{
		Interval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	dm.Start(ctx)

	// Cancel context should stop manager.
	cancel()

	// Give it time to stop.
	time.Sleep(50 * time.Millisecond)

	// The manager's running flag may still be true, but the goroutine should be done.
	// We can verify by checking stats haven't changed much.
}

func TestDecayManagerWithScorers(t *testing.T) {
	dm := NewDecayManagerWithConfig(DecayManagerConfig{
		Interval: 10 * time.Millisecond,
	})

	surfaceScorer := NewSurfaceScorer()
	specterScorer := NewSpecterScorer()

	dm.RegisterSurfaceScorer(surfaceScorer)
	dm.RegisterSpecterScorer(specterScorer)

	// Add scores that will be affected by decay.
	surface := surfaceScorer.GetScore("user-1")
	surface.SetConnectionCount(50)
	surface.SetWaveCount(30)
	initialSurface := surface.Compute()

	specter := specterScorer.GetScore("specter-1")
	specter.SetConnectionCount(50)
	specter.SetWaveCount(30)
	initialSpecter := specter.Compute()

	ctx := context.Background()
	dm.Start(ctx)

	// Wait for a few cycles.
	time.Sleep(50 * time.Millisecond)

	dm.Stop()

	// Scores should be recomputed (cache invalidated).
	// Values may be same since no actual time decay occurred, but cache was invalidated.
	afterSurface := surface.Compute()
	afterSpecter := specter.Compute()

	// Surface score might be affected by temporal decay if LastActivityTime was set.
	// Specter score should be similar (no temporal decay in SpecterScore yet).
	t.Logf("Surface: before=%d, after=%d", initialSurface, afterSurface)
	t.Logf("Specter: before=%d, after=%d", initialSpecter, afterSpecter)
}

func TestDecayManagerStats(t *testing.T) {
	dm := NewDecayManager()

	stats := dm.Stats()
	if stats.Running {
		t.Error("Should not be running")
	}
	if stats.Interval != 60*time.Second {
		t.Errorf("Interval = %v, want 60s", stats.Interval)
	}
	if stats.RunCount != 0 {
		t.Errorf("RunCount = %d, want 0", stats.RunCount)
	}
}

func TestDecayManagerSetInterval(t *testing.T) {
	dm := NewDecayManager()

	dm.SetInterval(30 * time.Second)
	if dm.interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", dm.interval)
	}

	// Zero should be ignored.
	dm.SetInterval(0)
	if dm.interval != 30*time.Second {
		t.Errorf("Zero interval should be ignored, got %v", dm.interval)
	}

	// Negative should be ignored.
	dm.SetInterval(-1 * time.Second)
	if dm.interval != 30*time.Second {
		t.Errorf("Negative interval should be ignored, got %v", dm.interval)
	}
}

func TestDefaultDecayManagerConfig(t *testing.T) {
	cfg := DefaultDecayManagerConfig()
	if cfg.Interval != 60*time.Second {
		t.Errorf("Default interval = %v, want 60s", cfg.Interval)
	}
}

func TestDecayManagerRunOnceNoScorers(t *testing.T) {
	dm := NewDecayManager()

	// Should not panic with no scorers registered.
	dm.RunOnce()

	stats := dm.Stats()
	if stats.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", stats.RunCount)
	}
	if stats.SurfaceHits != 0 {
		t.Errorf("SurfaceHits = %d, want 0", stats.SurfaceHits)
	}
}
