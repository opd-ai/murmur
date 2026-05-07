// UI initialization for Ebitengine builds.
//

//go:build !test
// +build !test

package app

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
	"github.com/opd-ai/murmur/pkg/onboarding/screens"
	"github.com/opd-ai/murmur/pkg/pulsemap"
	"github.com/opd-ai/murmur/pkg/store"
)

// runUI initializes and starts the Pulse Map UI via Ebitengine.
// Per AUDIT.md remediation, this wires ebiten.RunGame() to enable the visualization.
// If firstRun is true, displays onboarding screens instead of Pulse Map.
// Per ROADMAP.md line 776, returning users see a welcome back screen before Pulse Map.
func (a *App) runUI() error {
	// Check if this is first run.
	a.mu.RLock()
	isFirstRun := a.firstRun
	onboardingFlow := a.subsystems.OnboardingFlow
	displayName, _ := a.subsystems.Storage.Get(store.BucketIdentity, []byte("display_name"))
	a.mu.RUnlock()

	// If first run and onboarding flow exists, show onboarding screens.
	if isFirstRun && onboardingFlow != nil {
		if err := a.runOnboardingUI(); err != nil {
			return err
		}

		// On first run, onboarding and Pulse Map run in one UI session via
		// onboardingTransitionGame. If onboarding exited before transition,
		// app cancellation (or context cancellation) controls shutdown here.
		return nil
	}

	// For returning users, show welcome back screen first.
	if !isFirstRun {
		if err := a.runReturningUserScreen(string(displayName)); err != nil {
			// If welcome screen fails, just continue to Pulse Map.
			fmt.Printf("Warning: returning user screen failed: %v\n", err)
		}
	}

	// Show normal Pulse Map.
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
	var transitionToPulseMap atomic.Bool

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
			if phase == flow.PhaseIdentityCreation && transitionToPulseMap.CompareAndSwap(false, true) {
				if err := a.subsystems.Storage.Put(store.BucketConfig, []byte("first_run_complete"), []byte("true")); err != nil {
					fmt.Printf("Warning: Failed to persist first-run flag: %v\n", err)
				} else {
					a.mu.Lock()
					a.firstRun = false
					a.mu.Unlock()
				}
				fmt.Println("Onboarding complete, transitioning to Pulse Map...")
			}
		},
		OnSkipBackup: func() {
			fmt.Println("Onboarding: Backup skipped (warning shown to user)")
		},
	})

	onboardingGame := &onboardingTransitionGame{
		game:                  screen,
		transitionToPulseMap:  &transitionToPulseMap,
		buildPulseMapGameFunc: a.buildPulseMapGame,
	}

	fmt.Println("Starting onboarding screens...")
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("MURMUR — Onboarding")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(onboardingGame); err != nil {
		return fmt.Errorf("running onboarding: %w", err)
	}

	a.cancel()
	return nil
}

type onboardingTransitionGame struct {
	game                  ebiten.Game
	transitionToPulseMap  *atomic.Bool
	buildPulseMapGameFunc func() (ebiten.Game, error)
	transitioned          bool
}

func (g *onboardingTransitionGame) Update() error {
	if g.transitionToPulseMap.Load() && !g.transitioned {
		pulseMapGame, err := g.buildPulseMapGameFunc()
		if err != nil {
			return fmt.Errorf("building Pulse Map game for transition: %w", err)
		}

		g.game = pulseMapGame
		g.transitioned = true

		ebiten.SetWindowTitle("MURMUR — Pulse Map")
		fmt.Println("Starting Pulse Map visualization...")
	}

	return g.game.Update()
}

func (g *onboardingTransitionGame) Draw(screen *ebiten.Image) {
	g.game.Draw(screen)
}

func (g *onboardingTransitionGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.game.Layout(outsideWidth, outsideHeight)
}

// runPulseMapUI displays the normal Pulse Map visualization.
func (a *App) runPulseMapUI() error {
	fmt.Println("Initializing Pulse Map UI...")

	game, err := a.buildPulseMapGame()
	if err != nil {
		return err
	}

	return a.runEbitenGame(game, "MURMUR — Pulse Map", "Starting Pulse Map visualization...", "running Pulse Map")
}

func (a *App) buildPulseMapGame() (ebiten.Game, error) {

	// Ensure subsystems are initialized before creating game.
	a.mu.RLock()
	keypair := a.subsystems.Identity
	pubsub := a.subsystems.PubSub
	storage := a.subsystems.Storage
	a.mu.RUnlock()

	if keypair == nil {
		return nil, fmt.Errorf("identity not initialized")
	}
	if pubsub == nil {
		return nil, fmt.Errorf("pubsub not initialized")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// Create the Pulse Map game instance with Wave publishing capability and store access.
	game, err := pulsemap.NewGame(a.ctx, keypair, pubsub, storage, a.config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("creating Pulse Map game: %w", err)
	}

	// Wire the Shadow Gradient modes.Manager so privacy_mode settings take effect.
	// Per SHADOW_GRADIENT.md, Open/Hybrid/Guarded/Fortress modes are user-controllable at runtime.
	modeMgr := modes.NewManager()
	game.SetModeManager(modeMgr)

	// Store in subsystems for access by other components.
	a.mu.Lock()
	a.subsystems.PulseMapUI = game
	a.subsystems.ModeManager = modeMgr
	a.mu.Unlock()

	return game, nil
}

// runReturningUserScreen displays a welcome back screen for returning users.
// Per ROADMAP.md line 776, this provides fast bootstrap with existing identity detection.
func (a *App) runReturningUserScreen(displayName string) error {
	fmt.Println("Showing returning user welcome screen...")

	// Get keypair for fingerprint display.
	a.mu.RLock()
	keypair := a.subsystems.Identity
	a.mu.RUnlock()

	if keypair == nil {
		return fmt.Errorf("identity not initialized")
	}

	// Create returning user screen.
	continueCh := make(chan struct{})
	var continueOnce sync.Once
	screen := screens.NewReturningScreen(
		displayName,
		keypair,
		func() {
			continueOnce.Do(func() {
				close(continueCh)
			})
		},
	)

	// Run returning screen without using runEbitenGame so this temporary
	// transition does not call a.cancel() before Pulse Map startup.
	fmt.Println("Loading...")
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("MURMUR — Welcome Back")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	errCh := make(chan error, 1)
	go func() {
		errCh <- ebiten.RunGame(screen)
	}()

	// Wait for either completion or error.
	select {
	case <-continueCh:
		// User continuing, screen will exit naturally.
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("running welcome screen: %w", err)
		}
		return nil
	case <-a.ctx.Done():
		return a.ctx.Err()
	}
}
