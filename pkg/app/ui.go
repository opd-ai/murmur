// UI initialization for Ebitengine builds.
//

//go:build !test
// +build !test

package app

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/pulsemap"
)

// runUI initializes and starts the Pulse Map UI via Ebitengine.
// Per AUDIT.md remediation, this wires ebiten.RunGame() to enable the visualization.
func (a *App) runUI() error {
	fmt.Println("Initializing Pulse Map UI...")

	// Ensure subsystems are initialized before creating game.
	a.mu.RLock()
	keypair := a.subsystems.Identity
	pubsub := a.subsystems.PubSub
	a.mu.RUnlock()

	if keypair == nil {
		return fmt.Errorf("identity not initialized")
	}
	if pubsub == nil {
		return fmt.Errorf("pubsub not initialized")
	}

	// Create the Pulse Map game instance with Wave publishing capability.
	game, err := pulsemap.NewGame(a.ctx, keypair, pubsub)
	if err != nil {
		return fmt.Errorf("creating Pulse Map game: %w", err)
	}

	// Store in subsystems for access by other components.
	a.mu.Lock()
	a.subsystems.PulseMapUI = game
	a.mu.Unlock()

	fmt.Println("Starting Pulse Map visualization...")
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("MURMUR — Pulse Map")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run the game loop (blocks until window is closed).
	// Per AUDIT.md: This replaces the blocking `<-a.ctx.Done()` call.
	if err := ebiten.RunGame(game); err != nil {
		return fmt.Errorf("running Pulse Map: %w", err)
	}

	// When the window closes, trigger graceful shutdown.
	a.cancel()

	return nil
}
