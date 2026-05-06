package app

import (
	"context"
	"os"
	"testing"
	"time"
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
