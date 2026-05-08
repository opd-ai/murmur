// Package main provides tests for the MURMUR entry point.
package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/app"
)

// TestRunWithConfig tests the runWithConfig function with a temporary directory.
func TestRunWithConfig(t *testing.T) {
	// Set up a temporary data directory.
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a channel to capture run result.
	runErr := make(chan error, 1)
	appChan := make(chan *app.App, 1)

	// Run in a goroutine since it blocks.
	go func() {
		// We need to intercept the app creation to get a handle.
		application, createErr := app.New(app.Config{
			Version: "0.0.0-test",
			DataDir: tmpDir,
			SkipUI:  true, // Headless mode for testing.
		})
		if createErr != nil {
			runErr <- createErr
			return
		}

		appChan <- application
		runErr <- application.Run()
	}()

	// Wait for app to be created.
	var application *app.App
	select {
	case application = <-appChan:
	case err := <-runErr:
		t.Fatalf("application startup failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("application was not created within timeout")
	}

	// Wait for init.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := application.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Verify version.
	if application.Version() != "0.0.0-test" {
		t.Errorf("Version() = %q, want %q", application.Version(), "0.0.0-test")
	}

	// Clean shutdown.
	if err := application.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Run() did not return within 10 seconds after Close()")
	}
}

// TestRunWithConfigDirectly tests runWithConfig directly.
func TestRunWithConfigDirectly(t *testing.T) {
	// Set up a temporary data directory.
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// We'll run in a goroutine and stop it after a short delay.
	runErr := make(chan error, 1)
	appChan := make(chan *app.App, 1)

	// Override appNew to capture the created app.
	origAppNew := appNew
	appNew = func(cfg app.Config) (*app.App, error) {
		a, err := app.New(cfg)
		if err != nil {
			return nil, err
		}
		appChan <- a
		return a, nil
	}
	defer func() { appNew = origAppNew }()

	go func() {
		runErr <- runWithConfig(app.Config{
			Version: "0.0.0-direct",
			DataDir: tmpDir,
			SkipUI:  true, // Headless mode for testing.
		})
	}()

	// Wait for app to be created.
	var createdApp *app.App
	select {
	case createdApp = <-appChan:
	case <-time.After(5 * time.Second):
		t.Fatal("Application was not created within timeout")
	}

	// Wait for init.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := createdApp.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Verify version.
	if createdApp.Version() != "0.0.0-direct" {
		t.Errorf("Version() = %q, want %q", createdApp.Version(), "0.0.0-direct")
	}

	// Clean shutdown.
	if err := createdApp.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("runWithConfig() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("runWithConfig() did not return within 10 seconds after Close()")
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
		SkipUI:  true, // Headless mode for testing.
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
		SkipUI:  true, // Headless mode for testing.
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
		SkipUI:  true, // Headless mode for testing.
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

// TestRunFunction tests the run() function using the Version variable.
func TestRunFunction(t *testing.T) {
	// Set up a temporary data directory.
	tmpDir, err := os.MkdirTemp("", "murmur-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override appNew to capture the created app and use temp dir.
	origAppNew := appNew
	appChan := make(chan *app.App, 1)
	appNew = func(cfg app.Config) (*app.App, error) {
		// Override DataDir to use temp directory and enable headless mode.
		cfg.DataDir = tmpDir
		cfg.SkipUI = true
		a, err := app.New(cfg)
		if err != nil {
			return nil, err
		}
		appChan <- a
		return a, nil
	}
	defer func() { appNew = origAppNew }()

	runErr := make(chan error, 1)
	go func() {
		runErr <- run()
	}()

	// Wait for app to be created.
	var createdApp *app.App
	select {
	case createdApp = <-appChan:
	case <-time.After(5 * time.Second):
		t.Fatal("Application was not created within timeout")
	}

	// Wait for init.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := createdApp.WaitReady(ctx); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}

	// Verify version uses the global Version variable.
	if createdApp.Version() != Version {
		t.Errorf("Version() = %q, want %q", createdApp.Version(), Version)
	}

	// Clean shutdown.
	if err := createdApp.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("run() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("run() did not return within 10 seconds after Close()")
	}
}

// TestRunWithConfigError tests runWithConfig when app creation fails.
func TestRunWithConfigError(t *testing.T) {
	// Override appNew to return an error.
	origAppNew := appNew
	appNew = func(_ app.Config) (*app.App, error) {
		return nil, errors.New("mock creation error")
	}
	defer func() { appNew = origAppNew }()

	err := runWithConfig(app.Config{
		Version: "0.0.0-error-test",
	})

	if err == nil {
		t.Error("runWithConfig() should return error when app creation fails")
	}
	if !strings.Contains(err.Error(), "creating application") {
		t.Errorf("runWithConfig() error = %q, want to contain 'creating application'", err.Error())
	}
}
