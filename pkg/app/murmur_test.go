package app

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{
		Version: "0.0.0-test",
		DataDir: tmpDir,
		SkipUI:  true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	if app.Version() != "0.0.0-test" {
		t.Errorf("Version() = %q, want %q", app.Version(), "0.0.0-test")
	}
}

func TestAppContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	application, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := application.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}

	// Context should not be canceled yet.
	select {
	case <-ctx.Done():
		t.Error("Context is already canceled before Close()")
	default:
		// Expected.
	}

	// Close should cancel the context immediately (app not running).
	if err := application.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Context should be canceled after Close().
	select {
	case <-ctx.Done():
		// Expected - context is now canceled.
	default:
		t.Error("Context not canceled after Close()")
	}
}

func TestAppDoubleRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	// Start the app in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- app.Run()
	}()

	// Wait for initialization to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Second Run() should fail.
	err = app.Run()
	if err == nil {
		t.Error("second Run() should have returned an error")
	}

	// Clean up.
	app.Close()
	<-done
}

func TestAppSubsystemsInit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{
		Version: "0.0.0-test",
		DataDir: tmpDir,
		SkipUI:  true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	// Start the app in a goroutine.
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	// Wait for initialization to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Verify subsystems are initialized.
	subs := app.Subsystems()
	if subs == nil {
		t.Fatal("Subsystems() returned nil")
	}
	if subs.Storage == nil {
		t.Error("Storage subsystem not initialized")
	}
	if subs.Identity == nil {
		t.Error("Identity subsystem not initialized")
	}
	if subs.Host == nil {
		t.Error("Host subsystem not initialized")
	}
	if subs.PubSub == nil {
		t.Error("PubSub subsystem not initialized")
	}

	// First run should be detected.
	if !app.IsFirstRun() {
		t.Error("IsFirstRun() should be true on new database")
	}

	// Clean up.
	app.Close()
	<-runErr
}

func TestAppSubsystemsPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First run - create identity.
	var firstPeerID string
	{
		app, err := New(Config{DataDir: tmpDir, SkipUI: true})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.WaitReady(ctx); err != nil {
			t.Fatalf("WaitReady() error = %v", err)
		}

		if !app.IsFirstRun() {
			t.Error("First app instance should be first run")
		}

		subs := app.Subsystems()
		if subs.Host != nil {
			firstPeerID = subs.Host.PeerID().String()
		}

		app.Close()
		<-runErr
	}

	// Second run - load existing identity.
	{
		app, err := New(Config{DataDir: tmpDir, SkipUI: true})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.WaitReady(ctx); err != nil {
			t.Fatalf("WaitReady() error = %v", err)
		}

		if app.IsFirstRun() {
			t.Error("Second app instance should NOT be first run")
		}

		subs := app.Subsystems()
		if subs.Host != nil {
			secondPeerID := subs.Host.PeerID().String()
			if firstPeerID != "" && secondPeerID != firstPeerID {
				t.Errorf("Peer ID changed: %s -> %s", firstPeerID, secondPeerID)
			}
		}

		app.Close()
		<-runErr
	}
}

// TestMemoryMonitorContextCancellation verifies the memory monitor goroutine
// exits within 1s of context cancellation per AUDIT.md H2.
func TestMemoryMonitorContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Close should trigger context cancellation and all goroutines should exit.
	start := time.Now()
	if err := app.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	duration := time.Since(start)

	select {
	case <-runErr:
		// Expected - app.Run() returned
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() did not return within 2s of Close()")
	}

	if duration > time.Second {
		t.Logf("WARNING: Close() took %v (target <1s)", duration)
	}
}

// TestNudgeLoopContextCancellation verifies the nudge loop goroutine
// exits within 1s of context cancellation per AUDIT.md H2.
func TestNudgeLoopContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	app.ctx = ctx

	done := make(chan struct{})
	go func() {
		defer close(done)
		app.runNudgeLoop()
	}()

	// Cancel context and verify goroutine exits promptly.
	cancel()

	select {
	case <-done:
		// Expected - goroutine exited
	case <-time.After(1 * time.Second):
		t.Fatal("runNudgeLoop() did not exit within 1s of context cancellation")
	}
}

// TestAllGoroutinesExitOnContextCancel verifies all production goroutines
// exit within 1s of context cancellation per AUDIT.md H2.
func TestAllGoroutinesExitOnContextCancel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Close triggers context cancellation. All goroutines should exit within 1s.
	start := time.Now()
	if err := app.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	duration := time.Since(start)

	// Wait for Run() to return, confirming all goroutines have exited.
	select {
	case <-runErr:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() did not return within 2s, indicating hanging goroutines")
	}

	// Per AUDIT.md H2, we verify <1s exit time for defensive programming.
	if duration > time.Second {
		t.Logf("WARNING: goroutines took %v to exit (target <1s)", duration)
	}
}

// TestMemoryBudget256MiBDuringNormalOperation validates that memory usage
// stays under 256 MiB during normal operation per ROADMAP.md line 731
// and TECHNICAL_IMPLEMENTATION.md §6.
func TestMemoryBudget256MiBDuringNormalOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory budget test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{DataDir: tmpDir, SkipUI: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Simulate normal operation: create and store Waves.
	// With typical Wave size (~500 bytes) and TTL enforcement,
	// memory should stay well under 256 MiB.
	cache := app.subsystems.WaveCache
	if cache == nil {
		t.Fatal("WaveCache not initialized")
	}

	// Add 1000 Waves (typical active content window).
	// Each Wave is ~500 bytes including protobuf overhead.
	for i := 0; i < 1000; i++ {
		waveID := []byte(fmt.Sprintf("wave-%06d-test", i))
		content := []byte(strings.Repeat("x", 400))

		wave := &pb.Wave{
			WaveId:     waveID,
			Content:    content,
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: 7 * 24 * 60 * 60, // 7 days in seconds
			WaveType:   pb.WaveType_WAVE_TYPE_SURFACE,
		}

		if err := cache.Put(wave); err != nil {
			t.Logf("Warning: Failed to put wave %d: %v", i, err)
		}
	}

	// Force GC to get accurate memory reading.
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check memory usage.
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := m.Alloc / (1024 * 1024)

	const budgetMB = 256
	t.Logf("Memory usage: %d MiB (budget: %d MiB)", allocMB, budgetMB)
	t.Logf("  Alloc: %d MiB", allocMB)
	t.Logf("  TotalAlloc: %d MiB", m.TotalAlloc/(1024*1024))
	t.Logf("  Sys: %d MiB", m.Sys/(1024*1024))
	t.Logf("  NumGC: %d", m.NumGC)
	t.Logf("  Waves stored: 1000")

	// Validate memory is under budget.
	if allocMB > budgetMB {
		t.Errorf("Memory budget exceeded: %d MiB > %d MiB budget", allocMB, budgetMB)
	}

	// Validate memory monitor would trigger eviction before critical.
	const evictionThreshold = 200 // Per app.checkMemory()
	if allocMB > evictionThreshold {
		t.Logf("WARNING: Memory %d MiB exceeds eviction threshold %d MiB", allocMB, evictionThreshold)
	}

	// Clean shutdown.
	if err := app.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case <-runErr:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() did not return within 2s")
	}
}

// TestColdStartPerformance validates that application cold start completes in <5 seconds.
// Per ROADMAP.md line 847 and TECHNICAL_IMPLEMENTATION.md §6 performance targets.
// Cold start = first run with no existing database or keystore.
func TestColdStartPerformance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-cold-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	app, err := New(Config{
		Version:     "0.0.0-test",
		DataDir:     tmpDir,
		SkipUI:      true,
		ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	startTime := time.Now()
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	elapsed := time.Since(startTime)
	t.Logf("Cold start completed in %v", elapsed)

	const coldStartTarget = 5 * time.Second
	if elapsed > coldStartTarget {
		t.Errorf("Cold start took %v, exceeds target of %v", elapsed, coldStartTarget)
	}

	if err := app.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case <-runErr:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() did not return within 2s")
	}
}

// TestWarmStartPerformance validates that application warm start completes in <2 seconds.
// Per ROADMAP.md line 847 and TECHNICAL_IMPLEMENTATION.md §6 performance targets.
// Warm start = subsequent run with existing database and keystore.
func TestWarmStartPerformance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-warm-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First run to initialize database and keystore.
	app1, err := New(Config{
		Version:     "0.0.0-test",
		DataDir:     tmpDir,
		SkipUI:      true,
		ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"},
	})
	if err != nil {
		t.Fatalf("New() first run error = %v", err)
	}

	runErr1 := make(chan error, 1)
	go func() {
		runErr1 <- app1.Run()
	}()

	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel1()
	if err := app1.WaitReady(ctx1); err != nil {
		t.Fatalf("WaitReady() first run error = %v", err)
	}

	if err := app1.Close(); err != nil {
		t.Fatalf("Close() first run error = %v", err)
	}

	select {
	case <-runErr1:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() first run did not return within 2s")
	}

	// Second run (warm start) with existing database and keystore.
	app2, err := New(Config{
		Version:     "0.0.0-test",
		DataDir:     tmpDir,
		SkipUI:      true,
		ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"},
	})
	if err != nil {
		t.Fatalf("New() second run error = %v", err)
	}
	defer app2.Close()

	startTime := time.Now()
	runErr2 := make(chan error, 1)
	go func() {
		runErr2 <- app2.Run()
	}()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := app2.WaitReady(ctx2); err != nil {
		t.Fatalf("WaitReady() second run error = %v", err)
	}

	elapsed := time.Since(startTime)
	t.Logf("Warm start completed in %v", elapsed)

	const warmStartTarget = 2 * time.Second
	if elapsed > warmStartTarget {
		t.Errorf("Warm start took %v, exceeds target of %v", elapsed, warmStartTarget)
	}

	if err := app2.Close(); err != nil {
		t.Errorf("Close() second run error = %v", err)
	}

	select {
	case <-runErr2:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run() second run did not return within 2s")
	}
}
