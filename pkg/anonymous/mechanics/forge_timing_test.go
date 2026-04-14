package mechanics

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestForgePhaseString(t *testing.T) {
	tests := []struct {
		phase ForgePhase
		want  string
	}{
		{ForgePhaseWarmup, "Warmup"},
		{ForgePhaseOpen, "Open"},
		{ForgePhaseEvaluation, "Evaluation"},
		{ForgePhaseComplete, "Complete"},
		{ForgePhaseCancelled, "Cancelled"},
		{ForgePhase(99), "Unknown"},
	}

	for _, tc := range tests {
		got := ForgePhaseString(tc.phase)
		if got != tc.want {
			t.Errorf("ForgePhaseString(%d) = %q, want %q", tc.phase, got, tc.want)
		}
	}
}

func TestDefaultForgeTimingConfig(t *testing.T) {
	cfg := DefaultForgeTimingConfig(ForgeDuration30Min)

	if cfg.WarmupDuration != 0 {
		t.Errorf("WarmupDuration = %v, want 0", cfg.WarmupDuration)
	}
	if cfg.SubmissionDuration != ForgeDuration30Min {
		t.Errorf("SubmissionDuration = %v, want %v", cfg.SubmissionDuration, ForgeDuration30Min)
	}
	if cfg.EvaluationDuration != 5*time.Minute {
		t.Errorf("EvaluationDuration = %v, want 5m", cfg.EvaluationDuration)
	}
}

func TestForgeTimingConfig30Min(t *testing.T) {
	cfg := ForgeTimingConfig30Min()
	if cfg.SubmissionDuration != 30*time.Minute {
		t.Errorf("SubmissionDuration = %v, want 30m", cfg.SubmissionDuration)
	}
}

func TestForgeTimingConfig60Min(t *testing.T) {
	cfg := ForgeTimingConfig60Min()
	if cfg.SubmissionDuration != 60*time.Minute {
		t.Errorf("SubmissionDuration = %v, want 60m", cfg.SubmissionDuration)
	}
}

func TestNewForgeTiming_NoWarmup(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)

	if ft.CurrentPhase != ForgePhaseOpen {
		t.Errorf("CurrentPhase = %v, want ForgePhaseOpen", ft.CurrentPhase)
	}
	if !ft.IsSubmissionOpen() {
		t.Error("Expected IsSubmissionOpen() to be true")
	}
}

func TestNewForgeTiming_WithWarmup(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     5 * time.Minute,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)

	if ft.CurrentPhase != ForgePhaseWarmup {
		t.Errorf("CurrentPhase = %v, want ForgePhaseWarmup", ft.CurrentPhase)
	}
	if ft.IsSubmissionOpen() {
		t.Error("Expected IsSubmissionOpen() to be false during warmup")
	}
}

func TestNewForgeTimingAt(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     5 * time.Minute,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	startTime := time.Now().Add(-10 * time.Minute) // Started 10 minutes ago
	ft := NewForgeTimingAt(cfg, startTime)

	if ft.StartTime != startTime {
		t.Errorf("StartTime = %v, want %v", ft.StartTime, startTime)
	}

	// After update, should transition past warmup.
	ft.Update()
	if ft.CurrentPhase != ForgePhaseOpen {
		t.Errorf("After update, CurrentPhase = %v, want ForgePhaseOpen", ft.CurrentPhase)
	}
}

func TestForgeTiming_PhaseTransitions(t *testing.T) {
	// Use very short durations for testing.
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 50 * time.Millisecond,
		EvaluationDuration: 50 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	// Should start open.
	if ft.Phase() != ForgePhaseOpen {
		t.Errorf("Initial phase = %v, want ForgePhaseOpen", ft.Phase())
	}

	// Wait for submission to close.
	time.Sleep(60 * time.Millisecond)
	ft.Update()

	if ft.Phase() != ForgePhaseEvaluation {
		t.Errorf("After submission, phase = %v, want ForgePhaseEvaluation", ft.Phase())
	}

	// Wait for evaluation to complete.
	time.Sleep(60 * time.Millisecond)
	ft.Update()

	if ft.Phase() != ForgePhaseComplete {
		t.Errorf("After evaluation, phase = %v, want ForgePhaseComplete", ft.Phase())
	}
}

func TestForgeTiming_OnPhaseChange(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 10 * time.Millisecond,
		EvaluationDuration: 10 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	var callCount int32
	var lastOld, lastNew ForgePhase

	ft.OnPhaseChange(func(old, new ForgePhase) {
		atomic.AddInt32(&callCount, 1)
		lastOld = old
		lastNew = new
	})

	// Wait and trigger transition.
	time.Sleep(15 * time.Millisecond)
	changed := ft.Update()

	if !changed {
		t.Error("Expected Update() to return true on phase change")
	}

	if atomic.LoadInt32(&callCount) == 0 {
		t.Error("Expected phase change callback to be called")
	}

	if lastOld != ForgePhaseOpen {
		t.Errorf("Callback old phase = %v, want ForgePhaseOpen", lastOld)
	}
	if lastNew != ForgePhaseEvaluation {
		t.Errorf("Callback new phase = %v, want ForgePhaseEvaluation", lastNew)
	}
}

func TestForgeTiming_Cancel(t *testing.T) {
	cfg := ForgeTimingConfig30Min()
	ft := NewForgeTiming(cfg)

	ft.Cancel()

	if !ft.IsCancelled() {
		t.Error("Expected IsCancelled() to be true")
	}
	if ft.Phase() != ForgePhaseCancelled {
		t.Errorf("Phase = %v, want ForgePhaseCancelled", ft.Phase())
	}
}

func TestForgeTiming_ForceComplete(t *testing.T) {
	cfg := ForgeTimingConfig30Min()
	ft := NewForgeTiming(cfg)

	ft.ForceComplete()

	if !ft.IsComplete() {
		t.Error("Expected IsComplete() to be true")
	}
	if ft.Phase() != ForgePhaseComplete {
		t.Errorf("Phase = %v, want ForgePhaseComplete", ft.Phase())
	}
}

func TestForgeTiming_TimeRemaining(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)

	remaining := ft.TimeRemaining()

	// Should be close to 30 minutes.
	if remaining < 29*time.Minute || remaining > 30*time.Minute {
		t.Errorf("TimeRemaining = %v, expected ~30m", remaining)
	}
}

func TestForgeTiming_SubmissionTimeRemaining(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)

	remaining := ft.SubmissionTimeRemaining()

	if remaining < 29*time.Minute || remaining > 30*time.Minute {
		t.Errorf("SubmissionTimeRemaining = %v, expected ~30m", remaining)
	}

	// After completion, should be 0.
	ft.ForceComplete()
	if ft.SubmissionTimeRemaining() != 0 {
		t.Error("SubmissionTimeRemaining should be 0 after completion")
	}
}

func TestForgeTiming_Progress(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 100 * time.Millisecond,
		EvaluationDuration: 100 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	// Initially, progress should be near 0.
	initial := ft.Progress()
	if initial > 0.1 {
		t.Errorf("Initial progress = %v, expected near 0", initial)
	}

	// Wait for half the submission period.
	time.Sleep(50 * time.Millisecond)
	mid := ft.Progress()
	if mid < 0.3 || mid > 0.7 {
		t.Errorf("Mid progress = %v, expected 0.3-0.7", mid)
	}

	// After completion, progress should be 1.0.
	ft.ForceComplete()
	final := ft.Progress()
	if final != 1.0 {
		t.Errorf("Final progress = %v, expected 1.0", final)
	}
}

func TestForgeTiming_TotalProgress(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 100 * time.Millisecond,
		EvaluationDuration: 100 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	initial := ft.TotalProgress()
	if initial > 0.1 {
		t.Errorf("Initial total progress = %v, expected near 0", initial)
	}
}

func TestForgeTiming_PauseResume(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 100 * time.Millisecond,
		EvaluationDuration: 100 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	if ft.IsPaused() {
		t.Error("Should not be paused initially")
	}

	ft.Pause()
	if !ft.IsPaused() {
		t.Error("Should be paused after Pause()")
	}

	// Progress should not advance while paused.
	progressBefore := ft.Progress()
	time.Sleep(50 * time.Millisecond)
	progressAfter := ft.Progress()

	if progressAfter != progressBefore {
		t.Error("Progress should not advance while paused")
	}

	ft.Resume()
	if ft.IsPaused() {
		t.Error("Should not be paused after Resume()")
	}
}

func TestForgeTiming_ElapsedTime(t *testing.T) {
	cfg := ForgeTimingConfig30Min()
	ft := NewForgeTiming(cfg)

	time.Sleep(50 * time.Millisecond)
	elapsed := ft.ElapsedTime()

	if elapsed < 50*time.Millisecond {
		t.Errorf("ElapsedTime = %v, expected >= 50ms", elapsed)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{-1 * time.Second, "0:00"},
		{30 * time.Second, "00:30"},
		{1 * time.Minute, "01:00"},
		{1*time.Minute + 30*time.Second, "01:30"},
		{10 * time.Minute, "10:00"},
		{59*time.Minute + 59*time.Second, "59:59"},
		{1 * time.Hour, "01:00:00"},
		{1*time.Hour + 30*time.Minute + 45*time.Second, "01:30:45"},
	}

	for _, tc := range tests {
		got := FormatDuration(tc.d)
		if got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestForgeTiming_FormatTimeRemaining(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)
	formatted := ft.FormatTimeRemaining()

	// Should be approximately "29:XX" or "30:00".
	if len(formatted) < 5 {
		t.Errorf("FormatTimeRemaining = %q, expected MM:SS format", formatted)
	}
}

func TestForgeTiming_Snapshot(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 30 * time.Minute,
		EvaluationDuration: 5 * time.Minute,
	}

	ft := NewForgeTiming(cfg)
	snap := ft.Snapshot()

	if snap.Phase != ForgePhaseOpen {
		t.Errorf("Snapshot.Phase = %v, want ForgePhaseOpen", snap.Phase)
	}
	if !snap.IsSubmissionOpen {
		t.Error("Snapshot.IsSubmissionOpen should be true")
	}
	if snap.IsPaused {
		t.Error("Snapshot.IsPaused should be false")
	}
	if snap.TimeRemaining < 29*time.Minute {
		t.Errorf("Snapshot.TimeRemaining = %v, expected ~30m", snap.TimeRemaining)
	}
	if snap.FormattedRemaining == "" {
		t.Error("Snapshot.FormattedRemaining should not be empty")
	}
}

func TestNewForgeTimingIntegration(t *testing.T) {
	// Create a test forge.
	var initiator [32]byte
	copy(initiator[:], "test-initiator-key-12345678901")

	forge, err := NewSigilForge(
		ForgeSigilArt,
		"Test prompt",
		initiator,
		ForgeDuration30Min,
		100, // Sufficient resonance.
	)
	if err != nil {
		t.Fatalf("NewSigilForge failed: %v", err)
	}

	fti := NewForgeTimingIntegration(forge)

	if fti.Timing() == nil {
		t.Error("Timing() returned nil")
	}

	if !fti.CanSubmit() {
		t.Error("CanSubmit() should be true for new forge")
	}
}

func TestForgeTimingIntegration_SubmitEntry(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-key-12345678901")

	forge, err := NewSigilForge(
		ForgeSigilArt,
		"Test prompt",
		initiator,
		ForgeDuration30Min,
		100,
	)
	if err != nil {
		t.Fatalf("NewSigilForge failed: %v", err)
	}

	fti := NewForgeTimingIntegration(forge)

	var specter [32]byte
	copy(specter[:], "test-specter-key-123456789012")

	entry, err := fti.SubmitEntry(specter, []byte("test content"), [32]byte{})
	if err != nil {
		t.Errorf("SubmitEntry failed: %v", err)
	}
	if entry == nil {
		t.Error("SubmitEntry returned nil entry")
	}
}

func TestForgeTimingIntegration_Sync(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-key-12345678901")

	forge, err := NewSigilForge(
		ForgeSigilArt,
		"Test prompt",
		initiator,
		ForgeDuration30Min,
		100,
	)
	if err != nil {
		t.Fatalf("NewSigilForge failed: %v", err)
	}

	fti := NewForgeTimingIntegration(forge)

	// Sync should not panic.
	fti.Sync()

	// Force to evaluation phase.
	fti.Timing().mu.Lock()
	fti.Timing().CurrentPhase = ForgePhaseEvaluation
	fti.Timing().mu.Unlock()

	fti.Sync()

	// Force to complete phase.
	fti.Timing().mu.Lock()
	fti.Timing().CurrentPhase = ForgePhaseComplete
	fti.Timing().mu.Unlock()

	fti.Sync()
}

func TestForgeTiming_TerminalStatesStable(t *testing.T) {
	cfg := ForgeTimingConfig{
		WarmupDuration:     0,
		SubmissionDuration: 10 * time.Millisecond,
		EvaluationDuration: 10 * time.Millisecond,
	}

	ft := NewForgeTiming(cfg)

	// Complete the forge.
	ft.ForceComplete()

	// Even after Update(), phase should remain Complete.
	time.Sleep(20 * time.Millisecond)
	ft.Update()

	if ft.Phase() != ForgePhaseComplete {
		t.Errorf("Phase changed from Complete to %v", ft.Phase())
	}

	// Same for cancelled.
	ft2 := NewForgeTiming(cfg)
	ft2.Cancel()
	time.Sleep(20 * time.Millisecond)
	ft2.Update()

	if ft2.Phase() != ForgePhaseCancelled {
		t.Errorf("Phase changed from Cancelled to %v", ft2.Phase())
	}
}

func TestForgeTimingIntegration_CannotSubmitWhenClosed(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-key-12345678901")

	forge, err := NewSigilForge(
		ForgeSigilArt,
		"Test prompt",
		initiator,
		ForgeDuration30Min,
		100,
	)
	if err != nil {
		t.Fatalf("NewSigilForge failed: %v", err)
	}

	fti := NewForgeTimingIntegration(forge)

	// Force timing to evaluation phase.
	fti.Timing().mu.Lock()
	fti.Timing().CurrentPhase = ForgePhaseEvaluation
	fti.Timing().mu.Unlock()

	if fti.CanSubmit() {
		t.Error("CanSubmit() should be false during evaluation")
	}

	var specter [32]byte
	copy(specter[:], "test-specter-key-123456789012")

	_, err = fti.SubmitEntry(specter, []byte("test content"), [32]byte{})
	if err != ErrForgeClosed {
		t.Errorf("SubmitEntry error = %v, want ErrForgeClosed", err)
	}
}
