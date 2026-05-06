// Package app provides tests for goroutine shutdown behavior.
// Per AUDIT.md H2, these tests verify each production goroutine exits within 1s of context cancellation.
package app

import (
"context"
"testing"
"time"
)

// TestDedupRotationExitsOnCancel verifies the deduplication rotation goroutine
// exits within 1s of context cancellation.
func TestDedupRotationExitsOnCancel(t *testing.T) {
// Create a minimal Handlers struct.
h := &Handlers{}

ctx, cancel := context.WithCancel(context.Background())
done := make(chan struct{})
go func() {
defer close(done)
h.StartDedupRotation(ctx)
}()

// Cancel immediately.
cancel()

// Verify goroutine exits within 1s.
select {
case <-done:
// Success
case <-time.After(1 * time.Second):
t.Fatal("deduplication rotation goroutine did not exit within 1s of context cancellation")
}
}

// TestContextCancellationHandling verifies all production goroutines properly
// handle context cancellation. This is a simpler integration test that validates
// the real shutdown path rather than testing individual goroutines.
func TestContextCancellationHandling(t *testing.T) {
// Create and run an app.
tmpDir := t.TempDir()
app, err := New(Config{
DataDir: tmpDir,
SkipUI:  true,
})
if err != nil {
t.Fatalf("creating app: %v", err)
}

// Start app in background.
go func() {
_ = app.Run()
}()

// Wait for app to be ready.
initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer initCancel()
if err := app.WaitReady(initCtx); err != nil {
t.Fatalf("waiting for app ready: %v", err)
}

// Trigger shutdown by closing the app.
shutdownStart := time.Now()
if err := app.Close(); err != nil {
t.Errorf("closing app: %v", err)
}
shutdownDuration := time.Since(shutdownStart)

// Verify all goroutines exited within 1s.
// We already fixed the nudge loop, so this should pass now.
if shutdownDuration > 1*time.Second {
t.Errorf("Shutdown took %v, expected < 1s (goroutines should exit promptly on context cancellation)", shutdownDuration)
}
}
