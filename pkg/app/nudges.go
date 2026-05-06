// Package app provides the first-week nudges system.
// Per PLAN.md Step 3.8, nudges encourage new users to explore MURMUR features
// during their first week: Wave publishing (Day 1), connection formation (Day 2),
// Anonymous Layer exploration (Day 3), and Resonance milestone celebration (Days 5-7).

package app

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/store"
)

// Nudge represents a first-week encouragement notification.
type Nudge struct {
	Day     int
	Message string
	Mode    modes.Mode // Zero value (Open) means all modes
}

// nudgeSchedule defines the first-week nudge sequence per PLAN.md Step 3.8.
var nudgeSchedule = []Nudge{
	{Day: 1, Message: "Try replying to a Wave to join the conversation!", Mode: modes.Open},
	{Day: 2, Message: "Form a connection with a nearby node to strengthen your mesh.", Mode: modes.Open},
	{Day: 3, Message: "Ready to explore the Anonymous Layer? Place your first Specter Mark!", Mode: modes.Hybrid},
	{Day: 3, Message: "You're in Guarded mode. Try creating a Phantom Gift for someone!", Mode: modes.Guarded},
	{Day: 3, Message: "Fortress mode activated. Your first Cipher Puzzle awaits!", Mode: modes.Fortress},
	{Day: 5, Message: "Approaching your first Resonance milestone. Keep engaging!", Mode: modes.Open},
}

// runNudgeLoop is a background goroutine that checks account age and dispatches
// nudges at appropriate times during the first week. Per PLAN.md, this runs every
// 4 hours to avoid spamming but ensure timely delivery.
func (a *App) runNudgeLoop() {
	ticker := time.NewTicker(4 * time.Hour)
	defer ticker.Stop()

	// Also check immediately on startup (after 5-minute grace period).
	// Use a timer with context-awareness to avoid blocking shutdown.
	gracePeriod := time.NewTimer(5 * time.Minute)
	defer gracePeriod.Stop()

	select {
	case <-a.ctx.Done():
		return
	case <-gracePeriod.C:
		a.checkAndSendNudges()
	}

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.checkAndSendNudges()
		}
	}
}

// checkAndSendNudges queries account creation time and sends appropriate nudges.
func (a *App) checkAndSendNudges() {
	if !a.areSubsystemsReady() {
		return
	}

	accountAgeDays, ok := a.getAccountAgeDays()
	if !ok || accountAgeDays > 7 {
		return
	}

	currentMode := a.getCurrentMode()
	a.processNudgeSchedule(accountAgeDays, currentMode)
}

// areSubsystemsReady returns true if all required subsystems are initialized.
func (a *App) areSubsystemsReady() bool {
	return a.subsystems != nil && a.subsystems.Identity != nil && a.subsystems.Storage != nil
}

// getAccountAgeDays retrieves the account age in days from the identity declaration.
func (a *App) getAccountAgeDays() (int, bool) {
	pubKey := a.subsystems.Identity.PublicKey
	decl, err := a.subsystems.Storage.GetIdentityDeclaration(pubKey)
	if err != nil || decl == nil {
		return 0, false
	}

	accountAge := time.Since(time.Unix(decl.CreatedAt, 0))
	return int(accountAge.Hours() / 24), true
}

// processNudgeSchedule checks each scheduled nudge and sends if applicable.
func (a *App) processNudgeSchedule(accountAgeDays int, currentMode modes.Mode) {
	for _, nudge := range nudgeSchedule {
		if a.shouldSendNudge(nudge, accountAgeDays, currentMode) {
			nudgeKey := fmt.Sprintf("nudge_day%d_mode%d", nudge.Day, nudge.Mode)
			a.sendNudge(nudge)
			a.markNudgeShown(nudgeKey)
		}
	}
}

// shouldSendNudge returns true if the nudge should be sent based on timing, mode, and prior display.
func (a *App) shouldSendNudge(nudge Nudge, accountAgeDays int, currentMode modes.Mode) bool {
	if accountAgeDays < nudge.Day {
		return false // Not time yet
	}

	if nudge.Mode != modes.Open && nudge.Mode != currentMode {
		return false // Nudge not applicable to current mode
	}

	nudgeKey := fmt.Sprintf("nudge_day%d_mode%d", nudge.Day, nudge.Mode)
	return !a.wasNudgeShown(nudgeKey)
}

// getCurrentMode returns the user's current privacy mode from config.
// This reads from storage to handle mode changes during runtime.
func (a *App) getCurrentMode() modes.Mode {
	// Try to read from config bucket in storage.
	data, err := a.subsystems.Storage.Get(store.BucketConfig, []byte("privacy_mode"))
	if err != nil || len(data) != 1 {
		return modes.Open // Default to Open if not set
	}
	return modes.Mode(data[0])
}

// wasNudgeShown checks if a nudge was previously displayed.
// Uses the config bucket to persist nudge state across restarts.
func (a *App) wasNudgeShown(nudgeKey string) bool {
	val, err := a.subsystems.Storage.Get(store.BucketConfig, []byte("shown_"+nudgeKey))
	return err == nil && len(val) > 0
}

// markNudgeShown records that a nudge was displayed.
func (a *App) markNudgeShown(nudgeKey string) {
	_ = a.subsystems.Storage.Put(store.BucketConfig, []byte("shown_"+nudgeKey), []byte{1})
}

// sendNudge dispatches a nudge event to the event bus for UI display.
// In CLI mode, logs to stdout. In UI mode, the PulseMap can show a notification.
func (a *App) sendNudge(nudge Nudge) {
	if a.subsystems.EventBus != nil {
		// TODO: Define a proper NudgeEvent type in eventbus.go and dispatch here.
		// For now, log to stdout as MVP implementation.
		fmt.Printf("📢 MURMUR Nudge (Day %d): %s\n", nudge.Day, nudge.Message)
	} else {
		// Fallback: log directly.
		fmt.Printf("📢 MURMUR Nudge (Day %d): %s\n", nudge.Day, nudge.Message)
	}
}
