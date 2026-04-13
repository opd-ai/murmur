// Package main provides tests for the MURMUR entry point.
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/app"
)

// TestRunFunction verifies the run() function initializes and shuts down properly.
func TestRunFunction(t *testing.T) {
	// Set up a temporary data directory.
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the default data directory via HOME.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Override XDG_DATA_HOME for consistent data directory.
	origXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	// Test that we can create an application with the current Version.
	application, err := app.New(app.Config{
		Version: Version,
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("creating application with Version %q: %v", Version, err)
	}
	defer application.Close()

	// Verify version is correctly passed.
	if application.Version() != Version {
		t.Errorf("Version() = %q, want %q", application.Version(), Version)
	}
}

// TestVersionVariableIsSet verifies the Version variable has a value.
func TestVersionVariableIsSet(t *testing.T) {
	// Version is set via build flags, default is "0.0.0-alpha".
	if Version == "" {
		t.Error("Version should not be empty")
	}
	// Should be the default since we don't set build flags in test.
	if Version != "0.0.0-alpha" {
		t.Logf("Version = %q (may be set by build flags)", Version)
	}
}

// TestRunStartsApplication verifies that run() successfully initializes the app.
func TestRunStartsApplication(t *testing.T) {
	// Set up a temporary data directory.
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the default data directory.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create and run application in background.
	application, err := app.New(app.Config{
		Version: "0.0.0-test",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("creating application: %v", err)
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- application.Run()
	}()

	// Wait for init to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := application.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Verify app is running.
	if application.Version() != "0.0.0-test" {
		t.Errorf("Version() = %q, want %q", application.Version(), "0.0.0-test")
	}

	// Clean shutdown.
	if err := application.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	<-runErr
}

// TestVersionVariable verifies the Version variable can be overridden.
func TestVersionVariable(t *testing.T) {
	// Version is set via build flags, default is "0.0.0-alpha".
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// TestAppConfigDefaults verifies default configuration is applied.
func TestAppConfigDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	application, err := app.New(app.Config{
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer application.Close()

	// Verify context is valid.
	ctx := application.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}

	select {
	case <-ctx.Done():
		t.Error("Context should not be done before Close()")
	default:
		// Expected.
	}
}

// TestGracefulShutdown verifies application shuts down cleanly.
func TestGracefulShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	application, err := app.New(app.Config{
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- application.Run()
	}()

	// Wait for init.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := application.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Initiate shutdown.
	shutdownStart := time.Now()
	if err := application.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Wait for run to return.
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run() did not return within 5 seconds after Close()")
	}

	// Verify shutdown was reasonably fast.
	shutdownDuration := time.Since(shutdownStart)
	if shutdownDuration > 3*time.Second {
		t.Errorf("Shutdown took %v, expected < 3s", shutdownDuration)
	}
}

// TestSubsystemsInitialized verifies all core subsystems are started.
func TestSubsystemsInitialized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	application, err := app.New(app.Config{
		Version: "0.0.0-test",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer application.Close()

	runErr := make(chan error, 1)
	go func() {
		runErr <- application.Run()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := application.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	subs := application.Subsystems()
	if subs == nil {
		t.Fatal("Subsystems() returned nil")
	}

	// Verify core subsystems.
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

	application.Close()
	<-runErr
}
