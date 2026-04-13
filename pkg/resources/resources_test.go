package resources

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryMonitor(t *testing.T) {
	m := NewMemoryMonitor(256)

	status := m.Check()
	if status.LimitMB != 256 {
		t.Errorf("LimitMB = %d, want 256", status.LimitMB)
	}
	if status.UsageRatio < 0 || status.UsageRatio > 1 {
		t.Errorf("UsageRatio = %v, want 0-1", status.UsageRatio)
	}
}

func TestMemoryMonitorWarning(t *testing.T) {
	// Use a very low limit to trigger warning.
	m := NewMemoryMonitor(1) // 1 MB limit.

	status := m.Check()
	// With a 1 MB limit, we're likely over threshold.
	if status.UsedMB > 0 && !status.Warning {
		t.Error("should warn when over 80% of very low limit")
	}
}

func TestConnectionLimiter(t *testing.T) {
	l := NewConnectionLimiter(3)

	if l.Max() != 3 {
		t.Errorf("Max() = %d, want 3", l.Max())
	}

	// Acquire 3 connections.
	for i := 0; i < 3; i++ {
		if !l.TryAcquire() {
			t.Errorf("TryAcquire %d should succeed", i)
		}
	}

	if l.Current() != 3 {
		t.Errorf("Current() = %d, want 3", l.Current())
	}

	if l.Available() != 0 {
		t.Errorf("Available() = %d, want 0", l.Available())
	}

	// Fourth should fail.
	if l.TryAcquire() {
		t.Error("TryAcquire should fail at limit")
	}

	// Release one.
	l.Release()
	if l.Current() != 2 {
		t.Errorf("Current() = %d, want 2", l.Current())
	}

	// Now can acquire again.
	if !l.TryAcquire() {
		t.Error("TryAcquire should succeed after release")
	}
}

func TestConnectionLimiterConcurrent(t *testing.T) {
	l := NewConnectionLimiter(10)

	var wg sync.WaitGroup
	acquired := make(chan bool, 20)

	// Try to acquire 20 times concurrently.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquired <- l.TryAcquire()
		}()
	}

	wg.Wait()
	close(acquired)

	successCount := 0
	for success := range acquired {
		if success {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("should have exactly 10 successful acquisitions, got %d", successCount)
	}
}

func TestBandwidthThrottler(t *testing.T) {
	// 1000 bytes per second.
	throttler := NewBandwidthThrottler(1000)

	if !throttler.IsEnabled() {
		t.Error("throttler should be enabled by default")
	}

	// Consume within budget.
	allowed, wait := throttler.Consume(500)
	if allowed != 500 {
		t.Errorf("allowed = %d, want 500", allowed)
	}
	if wait != 0 {
		t.Errorf("wait = %v, want 0", wait)
	}

	// Consume remaining.
	allowed, wait = throttler.Consume(500)
	if allowed != 500 {
		t.Errorf("allowed = %d, want 500", allowed)
	}
	if wait != 0 {
		t.Errorf("wait = %v, want 0", wait)
	}

	// Now bucket is empty, should have wait time.
	allowed, wait = throttler.Consume(100)
	if allowed != 0 {
		t.Errorf("allowed = %d, want 0", allowed)
	}
	if wait == 0 {
		t.Error("wait should be non-zero when bucket empty")
	}
}

func TestBandwidthThrottlerRefill(t *testing.T) {
	throttler := NewBandwidthThrottler(1000)

	// Empty the bucket.
	throttler.Consume(1000)

	// Wait for refill.
	time.Sleep(100 * time.Millisecond)

	// Should have ~100 bytes available.
	allowed, wait := throttler.Consume(50)
	if allowed < 50 {
		t.Errorf("allowed = %d, should be at least 50 after 100ms", allowed)
	}
	if wait != 0 && allowed >= 50 {
		t.Errorf("wait = %v, should be 0 if allowed enough", wait)
	}
}

func TestBandwidthThrottlerDisable(t *testing.T) {
	throttler := NewBandwidthThrottler(10) // Very low limit.

	// Disable throttling.
	throttler.Disable()

	if throttler.IsEnabled() {
		t.Error("throttler should be disabled")
	}

	// Should allow any amount when disabled.
	allowed, wait := throttler.Consume(10000)
	if allowed != 10000 {
		t.Errorf("disabled throttler should allow all, got %d", allowed)
	}
	if wait != 0 {
		t.Errorf("disabled throttler should have no wait, got %v", wait)
	}

	// Re-enable.
	throttler.Enable()
	if !throttler.IsEnabled() {
		t.Error("throttler should be enabled")
	}
}

func TestManager(t *testing.T) {
	m := NewManager()

	if m.Memory == nil {
		t.Error("Memory should not be nil")
	}
	if m.Connections == nil {
		t.Error("Connections should not be nil")
	}
	if m.Bandwidth == nil {
		t.Error("Bandwidth should not be nil")
	}

	status := m.Status()
	if status.MaxConnections != DefaultMaxConnections {
		t.Errorf("MaxConnections = %d, want %d", status.MaxConnections, DefaultMaxConnections)
	}
	if !status.BandwidthEnabled {
		t.Error("Bandwidth should be enabled by default")
	}
}

func TestManagerWithConfig(t *testing.T) {
	m := NewManagerWithConfig(512, 64, 100000)

	status := m.Status()
	if status.Memory.LimitMB != 512 {
		t.Errorf("Memory.LimitMB = %d, want 512", status.Memory.LimitMB)
	}
	if status.MaxConnections != 64 {
		t.Errorf("MaxConnections = %d, want 64", status.MaxConnections)
	}
}

func TestBandwidthWait(t *testing.T) {
	// High rate so test completes quickly.
	throttler := NewBandwidthThrottler(100000)

	start := time.Now()
	throttler.Wait(1000)
	elapsed := time.Since(start)

	// Should complete very quickly with high rate.
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait took too long: %v", elapsed)
	}
}
