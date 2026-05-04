// UI initialization for Ebitengine builds.
//

//go:build !test
// +build !test

package app

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
	"github.com/opd-ai/murmur/pkg/onboarding/screens"
	"github.com/opd-ai/murmur/pkg/pulsemap"
)

// runUI initializes and starts the Pulse Map UI via Ebitengine.
// Per AUDIT.md remediation, this wires ebiten.RunGame() to enable the visualization.
// If firstRun is true, displays onboarding screens instead of Pulse Map.
func (a *App) runUI() error {
	// Check if this is first run.
	a.mu.RLock()
	isFirstRun := a.firstRun
	onboardingFlow := a.subsystems.OnboardingFlow
	a.mu.RUnlock()

	// If first run and onboarding flow exists, show onboarding screens.
	if isFirstRun && onboardingFlow != nil {
		return a.runOnboardingUI()
	}

	// Otherwise, show normal Pulse Map.
	return a.runPulseMapUI()
}

// runEbitenGame configures and runs an Ebitengine game instance.
func (a *App) runEbitenGame(game ebiten.Game, title, startMsg, errMsg string) error {
	fmt.Println(startMsg)
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	a.cancel()
	return nil
}

// runOnboardingUI displays the onboarding screens for first-run users.
func (a *App) runOnboardingUI() error {
	fmt.Println("Initializing onboarding UI...")

	// Get onboarding flow controller.
	a.mu.RLock()
	onboardingFlowInterface := a.subsystems.OnboardingFlow
	a.mu.RUnlock()

	// Convert interface to actual type.
	// The OnboardingFlow field is stored as interface{} to avoid circular deps,
	// but we know it's actually a *flow.Controller wrapped in flowControllerAdapter.
	flowAdapter, ok := onboardingFlowInterface.(onboardingFlowController)
	if !ok || flowAdapter == nil {
		return fmt.Errorf("onboarding flow not initialized")
	}

	// Extract the underlying flow.Controller from the adapter.
	// This is safe because we control the adapter type.
	adapter := flowAdapter.(*flowControllerAdapter)
	flowController := adapter.controller

	// Create onboarding screen with callbacks.
	screen := screens.NewScreen(flowController, screens.ScreenCallbacks{
		OnKeypairGenerated: func(kp *keys.KeyPair) {
			fmt.Printf("Onboarding: Keypair generated\n")
			// Keypair is already stored by initIdentity(), no action needed.
		},
		OnDisplayNameSet: func(name string) {
			fmt.Printf("Onboarding: Display name set to %q\n", name)
			// TODO: Store display name in identity declaration.
		},
		OnBackupComplete: func(method string) {
			fmt.Printf("Onboarding: Backup completed via %s\n", method)
		},
		OnPhaseComplete: func(phase flow.Phase) {
			fmt.Printf("Onboarding: Phase %s complete\n", phase.String())
			// Check if onboarding is complete.
			if flowController.IsComplete() {
				// Transition to Pulse Map.
				fmt.Println("Onboarding complete, transitioning to Pulse Map...")
				// TODO: Implement transition logic.
				// For now, user must restart app.
			}
		},
		OnSkipBackup: func() {
			fmt.Println("Onboarding: Backup skipped (warning shown to user)")
		},
	})

	return a.runEbitenGame(screen, "MURMUR — Onboarding", "Starting onboarding screens...", "running onboarding")
}

// runPulseMapUI displays the normal Pulse Map visualization.
func (a *App) runPulseMapUI() error {
	fmt.Println("Initializing Pulse Map UI...")

	// Ensure subsystems are initialized before creating game.
	a.mu.RLock()
	keypair := a.subsystems.Identity
	pubsub := a.subsystems.PubSub
	storage := a.subsystems.Storage
	a.mu.RUnlock()

	if keypair == nil {
		return fmt.Errorf("identity not initialized")
	}
	if pubsub == nil {
		return fmt.Errorf("pubsub not initialized")
	}
	if storage == nil {
		return fmt.Errorf("storage not initialized")
	}

	// Create the Pulse Map game instance with Wave publishing capability and store access.
	game, err := pulsemap.NewGame(a.ctx, keypair, pubsub, storage, a.config.DataDir)
	if err != nil {
		return fmt.Errorf("creating Pulse Map game: %w", err)
	}

	// Store in subsystems for access by other components.
	a.mu.Lock()
	a.subsystems.PulseMapUI = game
	a.mu.Unlock()

	return a.runEbitenGame(game, "MURMUR — Pulse Map", "Starting Pulse Map visualization...", "running Pulse Map")
}
