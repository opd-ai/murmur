package app

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	app, err := New(Config{
		Version: "0.0.0-test",
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
	application, err := New(Config{})
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
	app, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	// Start the app in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- app.Run()
	}()

	// Give it time to start.
	time.Sleep(10 * time.Millisecond)

	// Second Run() should fail.
	err = app.Run()
	if err == nil {
		t.Error("second Run() should have returned an error")
	}

	// Clean up.
	app.Close()
	<-done
}
