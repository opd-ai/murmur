// Package app provides the top-level application lifecycle and event bus for MURMUR.
// The application struct coordinates all subsystems and manages startup/shutdown.
// Per TECHNICAL_IMPLEMENTATION.md §2, the event bus uses channel fan-out for
// decoupled communication between subsystems.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Config holds application configuration options.
type Config struct {
	// Version is the application version string.
	Version string

	// DataDir is the directory for persistent storage.
	// Defaults to platform-appropriate user data directory.
	DataDir string

	// ListenAddrs are the multiaddrs to listen on.
	// Defaults to ["/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic-v1"].
	ListenAddrs []string

	// BootstrapPeers are initial peers to connect to.
	// Uses hardcoded defaults if empty.
	BootstrapPeers []string
}

// App is the top-level MURMUR application.
// It coordinates all subsystems and manages the application lifecycle.
type App struct {
	config Config

	// ctx is the application context, canceled on shutdown.
	ctx    context.Context
	cancel context.CancelFunc

	// wg tracks running goroutines for clean shutdown.
	wg sync.WaitGroup

	// mu protects state during concurrent access.
	mu sync.RWMutex

	// running indicates if the application is currently running.
	running bool
}

// New creates a new MURMUR application with the given configuration.
func New(cfg Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	return app, nil
}

// Run starts the application and blocks until shutdown.
// It initializes all subsystems and starts the main event loop.
func (a *App) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("application already running")
	}
	a.running = true
	a.mu.Unlock()

	// TODO: Initialize subsystems in dependency order:
	// 1. Storage (Bbolt)
	// 2. Identity (keys, sigils)
	// 3. Networking (libp2p host, GossipSub, DHT)
	// 4. Content (Waves, PoW, propagation)
	// 5. Anonymous (Specters, Shroud, Resonance)
	// 6. Pulse Map (UI, force-directed layout)
	// 7. Onboarding (if first run)

	fmt.Printf("MURMUR %s starting...\n", a.config.Version)

	// Block until context is canceled.
	<-a.ctx.Done()

	return nil
}

// Close shuts down the application gracefully.
// It cancels the context and waits for all goroutines to complete.
func (a *App) Close() error {
	a.mu.Lock()
	wasRunning := a.running
	a.running = false
	a.mu.Unlock()

	// Always cancel the context to signal shutdown.
	a.cancel()

	// Only wait for goroutines if the app was running.
	if wasRunning {
		a.wg.Wait()
	}

	return nil
}

// Context returns the application context.
// Subsystems should use this context for their operations.
func (a *App) Context() context.Context {
	return a.ctx
}

// Version returns the application version string.
func (a *App) Version() string {
	return a.config.Version
}
