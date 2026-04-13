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

	application, err := New(Config{DataDir: tmpDir})
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

	app, err := New(Config{DataDir: tmpDir})
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
		app, err := New(Config{DataDir: tmpDir})
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
		app, err := New(Config{DataDir: tmpDir})
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
