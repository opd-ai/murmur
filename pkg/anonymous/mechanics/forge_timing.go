// Package mechanics implements anonymous layer game mechanics for MURMUR.
// Sigil Forge timing: Countdown timer and submission window management.
// Per ROADMAP.md line 478: "Timed creative challenge mechanics — countdown
// timer, submission window".
// Per ANONYMOUS_GAME_MECHANICS.md: "A Forge event provides a prompt, a time
// limit, and a medium (Sigil art, micro-fiction, remix chains)."
package mechanics

import (
	"sync"
	"time"
)

// ForgePhase represents the current phase of a timed forge challenge.
type ForgePhase uint8

const (
	// ForgePhaseWarmup is the period before submissions open (optional).
	ForgePhaseWarmup ForgePhase = iota

	// ForgePhaseOpen is when submissions are being accepted.
	ForgePhaseOpen

	// ForgePhaseEvaluation is when amplifications are being tallied.
	ForgePhaseEvaluation

	// ForgePhaseComplete is when the winner has been determined.
	ForgePhaseComplete

	// ForgePhaseCancelled is when the forge was cancelled (no participants).
	ForgePhaseCancelled
)

// ForgePhaseString returns a human-readable string for ForgePhase.
func ForgePhaseString(p ForgePhase) string {
	switch p {
	case ForgePhaseWarmup:
		return "Warmup"
	case ForgePhaseOpen:
		return "Open"
	case ForgePhaseEvaluation:
		return "Evaluation"
	case ForgePhaseComplete:
		return "Complete"
	case ForgePhaseCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// ForgeTimingConfig configures the timing parameters for a forge challenge.
type ForgeTimingConfig struct {
	// WarmupDuration is the optional warmup period before submissions open.
	// Set to 0 to skip warmup and start immediately.
	WarmupDuration time.Duration

	// SubmissionDuration is the main submission window (30 or 60 minutes).
	SubmissionDuration time.Duration

	// EvaluationDuration is the period for tallying amplifications.
	// Per spec, evaluation happens immediately after submission closes.
	// A short buffer allows late-arriving amplifications to propagate.
	EvaluationDuration time.Duration
}

// DefaultForgeTimingConfig returns the default timing configuration.
func DefaultForgeTimingConfig(duration time.Duration) ForgeTimingConfig {
	return ForgeTimingConfig{
		WarmupDuration:     0, // No warmup by default.
		SubmissionDuration: duration,
		EvaluationDuration: 5 * time.Minute, // 5-minute evaluation buffer.
	}
}

// ForgeTimingConfig30Min returns timing config for 30-minute forges.
func ForgeTimingConfig30Min() ForgeTimingConfig {
	return DefaultForgeTimingConfig(ForgeDuration30Min)
}

// ForgeTimingConfig60Min returns timing config for 60-minute forges.
func ForgeTimingConfig60Min() ForgeTimingConfig {
	return DefaultForgeTimingConfig(ForgeDuration60Min)
}

// ForgeTiming manages the countdown timer and submission window for a forge.
type ForgeTiming struct {
	mu sync.RWMutex

	// Config holds the timing configuration.
	Config ForgeTimingConfig

	// StartTime is when the forge was initiated.
	StartTime time.Time

	// WarmupEndTime is when warmup ends and submissions open.
	WarmupEndTime time.Time

	// SubmissionDeadline is when submissions close.
	SubmissionDeadline time.Time

	// EvaluationDeadline is when evaluation completes.
	EvaluationDeadline time.Time

	// CurrentPhase tracks the current forge phase.
	CurrentPhase ForgePhase

	// PausedAt is set if the forge timing is paused (for testing).
	PausedAt time.Time

	// TotalPauseDuration accumulates paused time.
	TotalPauseDuration time.Duration

	// callbacks for phase transitions.
	onPhaseChange []func(ForgePhase, ForgePhase)

	// lastUpdateTime tracks when phase was last checked.
	lastUpdateTime time.Time
}

// NewForgeTiming creates a new timing manager for a forge challenge.
func NewForgeTiming(config ForgeTimingConfig) *ForgeTiming {
	now := time.Now()

	warmupEnd := now
	if config.WarmupDuration > 0 {
		warmupEnd = now.Add(config.WarmupDuration)
	}

	submissionEnd := warmupEnd.Add(config.SubmissionDuration)
	evaluationEnd := submissionEnd.Add(config.EvaluationDuration)

	phase := ForgePhaseWarmup
	if config.WarmupDuration == 0 {
		phase = ForgePhaseOpen
	}

	return &ForgeTiming{
		Config:             config,
		StartTime:          now,
		WarmupEndTime:      warmupEnd,
		SubmissionDeadline: submissionEnd,
		EvaluationDeadline: evaluationEnd,
		CurrentPhase:       phase,
		lastUpdateTime:     now,
	}
}

// NewForgeTimingAt creates a timing manager starting at a specific time.
// Useful for reconstructing state from storage.
func NewForgeTimingAt(config ForgeTimingConfig, startTime time.Time) *ForgeTiming {
	warmupEnd := startTime
	if config.WarmupDuration > 0 {
		warmupEnd = startTime.Add(config.WarmupDuration)
	}

	submissionEnd := warmupEnd.Add(config.SubmissionDuration)
	evaluationEnd := submissionEnd.Add(config.EvaluationDuration)

	phase := ForgePhaseWarmup
	if config.WarmupDuration == 0 {
		phase = ForgePhaseOpen
	}

	return &ForgeTiming{
		Config:             config,
		StartTime:          startTime,
		WarmupEndTime:      warmupEnd,
		SubmissionDeadline: submissionEnd,
		EvaluationDeadline: evaluationEnd,
		CurrentPhase:       phase,
		lastUpdateTime:     startTime,
	}
}

// OnPhaseChange registers a callback for phase transitions.
// The callback receives (oldPhase, newPhase).
func (ft *ForgeTiming) OnPhaseChange(cb func(ForgePhase, ForgePhase)) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.onPhaseChange = append(ft.onPhaseChange, cb)
}

// Update checks the current time and updates the phase if needed.
// Returns true if the phase changed.
func (ft *ForgeTiming) Update() bool {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	return ft.updateLocked()
}

// updateLocked performs the phase update (caller must hold lock).
func (ft *ForgeTiming) updateLocked() bool {
	now := ft.effectiveTime()
	oldPhase := ft.CurrentPhase

	newPhase := ft.computePhase(now)
	if newPhase == oldPhase {
		return false
	}

	ft.CurrentPhase = newPhase
	ft.lastUpdateTime = time.Now()

	// Fire callbacks outside the lock to avoid deadlocks.
	callbacks := ft.onPhaseChange
	ft.mu.Unlock()
	for _, cb := range callbacks {
		cb(oldPhase, newPhase)
	}
	ft.mu.Lock()

	return true
}

// computePhase determines the phase based on effective time.
func (ft *ForgeTiming) computePhase(now time.Time) ForgePhase {
	// Don't transition from terminal states.
	if ft.CurrentPhase == ForgePhaseComplete ||
		ft.CurrentPhase == ForgePhaseCancelled {
		return ft.CurrentPhase
	}

	if now.Before(ft.WarmupEndTime) {
		return ForgePhaseWarmup
	}

	if now.Before(ft.SubmissionDeadline) {
		return ForgePhaseOpen
	}

	if now.Before(ft.EvaluationDeadline) {
		return ForgePhaseEvaluation
	}

	return ForgePhaseComplete
}

// effectiveTime returns the current time accounting for pauses.
func (ft *ForgeTiming) effectiveTime() time.Time {
	if !ft.PausedAt.IsZero() {
		return ft.PausedAt
	}
	return time.Now().Add(-ft.TotalPauseDuration)
}

// Phase returns the current phase.
func (ft *ForgeTiming) Phase() ForgePhase {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.CurrentPhase
}

// IsSubmissionOpen returns true if submissions are currently accepted.
func (ft *ForgeTiming) IsSubmissionOpen() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.CurrentPhase == ForgePhaseOpen
}

// IsEvaluating returns true if in evaluation phase.
func (ft *ForgeTiming) IsEvaluating() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.CurrentPhase == ForgePhaseEvaluation
}

// IsComplete returns true if the forge has completed.
func (ft *ForgeTiming) IsComplete() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.CurrentPhase == ForgePhaseComplete
}

// IsCancelled returns true if the forge was cancelled.
func (ft *ForgeTiming) IsCancelled() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.CurrentPhase == ForgePhaseCancelled
}

// Cancel marks the forge as cancelled.
func (ft *ForgeTiming) Cancel() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	oldPhase := ft.CurrentPhase
	ft.CurrentPhase = ForgePhaseCancelled

	callbacks := ft.onPhaseChange
	ft.mu.Unlock()
	for _, cb := range callbacks {
		cb(oldPhase, ForgePhaseCancelled)
	}
	ft.mu.Lock()
}

// ForceComplete marks the forge as complete (after evaluation finishes).
func (ft *ForgeTiming) ForceComplete() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	oldPhase := ft.CurrentPhase
	ft.CurrentPhase = ForgePhaseComplete

	callbacks := ft.onPhaseChange
	ft.mu.Unlock()
	for _, cb := range callbacks {
		cb(oldPhase, ForgePhaseComplete)
	}
	ft.mu.Lock()
}

// TimeRemaining returns the time remaining until the next phase transition.
func (ft *ForgeTiming) TimeRemaining() time.Duration {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	return ft.timeRemainingLocked()
}

// timeRemainingLocked computes remaining time (caller must hold lock).
func (ft *ForgeTiming) timeRemainingLocked() time.Duration {
	now := ft.effectiveTime()

	switch ft.CurrentPhase {
	case ForgePhaseWarmup:
		return ft.WarmupEndTime.Sub(now)
	case ForgePhaseOpen:
		return ft.SubmissionDeadline.Sub(now)
	case ForgePhaseEvaluation:
		return ft.EvaluationDeadline.Sub(now)
	default:
		return 0
	}
}

// SubmissionTimeRemaining returns time remaining in the submission window.
func (ft *ForgeTiming) SubmissionTimeRemaining() time.Duration {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := ft.effectiveTime()

	if ft.CurrentPhase == ForgePhaseWarmup {
		// Include warmup + submission time.
		return ft.SubmissionDeadline.Sub(now)
	}

	if ft.CurrentPhase != ForgePhaseOpen {
		return 0
	}

	return ft.SubmissionDeadline.Sub(now)
}

// EvaluationTimeRemaining returns time remaining in the evaluation phase.
func (ft *ForgeTiming) EvaluationTimeRemaining() time.Duration {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := ft.effectiveTime()

	if ft.CurrentPhase != ForgePhaseEvaluation {
		return 0
	}

	return ft.EvaluationDeadline.Sub(now)
}

// ElapsedTime returns total time since forge started.
func (ft *ForgeTiming) ElapsedTime() time.Duration {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := ft.effectiveTime()
	return now.Sub(ft.StartTime)
}

// Progress returns the progress through the current phase (0.0 to 1.0).
func (ft *ForgeTiming) Progress() float64 {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	return ft.progressLocked()
}

// progressLocked computes phase progress (caller must hold lock).
func (ft *ForgeTiming) progressLocked() float64 {
	now := ft.effectiveTime()

	var start, end time.Time
	switch ft.CurrentPhase {
	case ForgePhaseWarmup:
		start = ft.StartTime
		end = ft.WarmupEndTime
	case ForgePhaseOpen:
		start = ft.WarmupEndTime
		end = ft.SubmissionDeadline
	case ForgePhaseEvaluation:
		start = ft.SubmissionDeadline
		end = ft.EvaluationDeadline
	default:
		return 1.0
	}

	total := end.Sub(start)
	elapsed := now.Sub(start)

	if total <= 0 {
		return 1.0
	}

	progress := float64(elapsed) / float64(total)
	if progress < 0 {
		return 0.0
	}
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// TotalProgress returns overall progress through all phases (0.0 to 1.0).
func (ft *ForgeTiming) TotalProgress() float64 {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := ft.effectiveTime()
	total := ft.EvaluationDeadline.Sub(ft.StartTime)
	elapsed := now.Sub(ft.StartTime)

	if total <= 0 {
		return 1.0
	}

	progress := float64(elapsed) / float64(total)
	if progress < 0 {
		return 0.0
	}
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// Pause pauses the timing (for testing or administrative purposes).
func (ft *ForgeTiming) Pause() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.PausedAt.IsZero() {
		ft.PausedAt = time.Now()
	}
}

// Resume resumes timing after a pause.
func (ft *ForgeTiming) Resume() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if !ft.PausedAt.IsZero() {
		ft.TotalPauseDuration += time.Since(ft.PausedAt)
		ft.PausedAt = time.Time{}
	}
}

// IsPaused returns true if timing is currently paused.
func (ft *ForgeTiming) IsPaused() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return !ft.PausedAt.IsZero()
}

// FormatTimeRemaining returns a human-readable string of time remaining.
func (ft *ForgeTiming) FormatTimeRemaining() string {
	remaining := ft.TimeRemaining()
	return FormatDuration(remaining)
}

// FormatSubmissionTimeRemaining returns formatted submission time remaining.
func (ft *ForgeTiming) FormatSubmissionTimeRemaining() string {
	remaining := ft.SubmissionTimeRemaining()
	return FormatDuration(remaining)
}

// FormatDuration formats a duration as a human-readable countdown string.
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return formatHMS(hours, minutes, seconds)
	}
	return formatMS(minutes, seconds)
}

// formatHMS formats as H:MM:SS.
func formatHMS(h, m, s int) string {
	return padTwo(h) + ":" + padTwo(m) + ":" + padTwo(s)
}

// formatMS formats as M:SS.
func formatMS(m, s int) string {
	return padTwo(m) + ":" + padTwo(s)
}

// padTwo pads a number with leading zero if needed.
func padTwo(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// TimingSnapshot captures the current state of a ForgeTiming for UI display.
type TimingSnapshot struct {
	Phase               ForgePhase
	PhaseProgress       float64
	TotalProgress       float64
	TimeRemaining       time.Duration
	SubmissionRemaining time.Duration
	EvaluationRemaining time.Duration
	FormattedRemaining  string
	IsSubmissionOpen    bool
	IsPaused            bool
}

// Snapshot captures the current timing state for thread-safe UI access.
func (ft *ForgeTiming) Snapshot() TimingSnapshot {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	remaining := ft.timeRemainingLocked()

	return TimingSnapshot{
		Phase:               ft.CurrentPhase,
		PhaseProgress:       ft.progressLocked(),
		TotalProgress:       float64(ft.effectiveTime().Sub(ft.StartTime)) / float64(ft.EvaluationDeadline.Sub(ft.StartTime)),
		TimeRemaining:       remaining,
		SubmissionRemaining: ft.SubmissionDeadline.Sub(ft.effectiveTime()),
		EvaluationRemaining: ft.EvaluationDeadline.Sub(ft.effectiveTime()),
		FormattedRemaining:  FormatDuration(remaining),
		IsSubmissionOpen:    ft.CurrentPhase == ForgePhaseOpen,
		IsPaused:            !ft.PausedAt.IsZero(),
	}
}

// ForgeTimingIntegration provides methods for integrating timing with SigilForge.
type ForgeTimingIntegration struct {
	forge  *SigilForge
	timing *ForgeTiming
}

// NewForgeTimingIntegration creates a timing integration for a forge.
func NewForgeTimingIntegration(forge *SigilForge) *ForgeTimingIntegration {
	config := DefaultForgeTimingConfig(forge.Duration)

	return &ForgeTimingIntegration{
		forge:  forge,
		timing: NewForgeTimingAt(config, forge.CreatedAt),
	}
}

// Timing returns the underlying timing manager.
func (fti *ForgeTimingIntegration) Timing() *ForgeTiming {
	return fti.timing
}

// Sync synchronizes the forge state with timing state.
func (fti *ForgeTimingIntegration) Sync() {
	fti.timing.Update()

	phase := fti.timing.Phase()

	switch phase {
	case ForgePhaseOpen:
		// Forge should be active.
		if fti.forge.State != ForgeActive {
			fti.forge.mu.Lock()
			fti.forge.State = ForgeActive
			fti.forge.mu.Unlock()
		}

	case ForgePhaseEvaluation:
		// Trigger evaluation if not already done.
		if fti.forge.State == ForgeActive {
			fti.forge.UpdateState()
		}

	case ForgePhaseComplete:
		// Ensure forge is evaluated and completed.
		if fti.forge.State == ForgeEvaluating {
			_ = fti.forge.Evaluate()
		}
	}
}

// CanSubmit returns true if submissions are currently accepted.
func (fti *ForgeTimingIntegration) CanSubmit() bool {
	return fti.timing.IsSubmissionOpen() && fti.forge.IsActive()
}

// SubmitEntry wraps forge entry submission with timing validation.
func (fti *ForgeTimingIntegration) SubmitEntry(
	specter [32]byte,
	content []byte,
	parentID [32]byte,
) (*ForgeEntry, error) {
	if !fti.CanSubmit() {
		return nil, ErrForgeClosed
	}
	return fti.forge.SubmitEntry(specter, content, parentID)
}
