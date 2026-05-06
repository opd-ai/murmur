// Package app provides the top-level application lifecycle and event bus for MURMUR.
// The application struct coordinates all subsystems and manages startup/shutdown.
// Per TECHNICAL_IMPLEMENTATION.md §2, the event bus uses channel fan-out for
// decoupled communication between subsystems.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/shroud"
	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/murerr"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/networking/health"
	"github.com/opd-ai/murmur/pkg/networking/transport"
	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
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

	// SkipUI controls whether the Pulse Map UI is started.
	// Set true for headless operation or testing.
	SkipUI bool

	// CLIMode enables interactive command-line interface.
	// When true, starts a REPL for Wave creation and peer management.
	CLIMode bool

	// EnableRelay enables this node as a Shroud relay.
	// Relays help route anonymous traffic for others.
	EnableRelay bool

	// RelayBandwidth is the advertised bandwidth for relay operations (bytes/sec).
	// Only relevant if EnableRelay is true. Defaults to 10 MiB/s.
	RelayBandwidth uint64

	// EnableHealthEndpoint enables HTTP health check endpoint for monitoring.
	// Default false for privacy; bootstrap nodes should set true.
	EnableHealthEndpoint bool

	// HealthEndpointPort is the port for the health check endpoint.
	// Only relevant if EnableHealthEndpoint is true. Defaults to 8080.
	HealthEndpointPort int

	// InvitationURI is an optional invitation to accept during onboarding.
	// Format: murmur://invite/[Base64]. If provided, onboarding will use
	// the invitation for warm-start bootstrap.
	InvitationURI string
}

// Subsystems holds references to all initialized subsystems.
// Each subsystem is initialized in dependency order during Run().
type Subsystems struct {
	// Storage is the Bbolt database instance.
	Storage *store.DB

	// Identity is the Surface Layer keypair.
	Identity *keys.KeyPair

	// Host is the libp2p network host.
	Host *transport.Host

	// PubSub is the GossipSub instance for topic messaging.
	PubSub *gossip.PubSub

	// Handlers manages GossipSub message handlers.
	Handlers *Handlers

	// WaveCache stores received Waves.
	WaveCache *storage.Cache

	// EventBus is the central event dispatcher.
	EventBus *EventBus

	// Beacon manages Shroud relay discovery and circuit construction.
	// Nil if this node is not acting as a relay.
	Beacon *shroud.Beacon

	// CircuitManager manages Shroud circuit lifecycle and rotation.
	// Nil if Anonymous Layer is not initialized.
	CircuitManager *shroud.CircuitManager

	// OnboardingFlow manages the six-phase onboarding sequence.
	// Nil if not first run or if SkipUI is true.
	OnboardingFlow interface{} // Actual type: *flow.Controller

	// HealthServer provides HTTP health check endpoint for monitoring.
	// Nil if EnableHealthEndpoint is false (default).
	HealthServer interface{} // Actual type: *health.Server

	// PulseMapUI is the Ebitengine game loop for the Pulse Map visualization.
	// Nil if SkipUI is true. Type is interface{} to avoid hard ebiten dependency
	// in the app package (actual type is *pulsemap.Game which implements ebiten.Game).
	PulseMapUI interface{}
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

	// initComplete is closed when subsystem initialization is complete.
	initComplete chan struct{}

	// subsystems holds references to all initialized subsystems.
	subsystems *Subsystems

	// firstRun indicates if this is the first application run.
	firstRun bool
}

// New creates a new MURMUR application with the given configuration.
func New(cfg Config) (*App, error) {
	// Apply defaults.
	if cfg.DataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		cfg.DataDir = filepath.Join(homeDir, ".murmur")
	}
	if len(cfg.ListenAddrs) == 0 {
		cfg.ListenAddrs = []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		}
	}
	// Apply default bootstrap peers if none configured.
	// Per AUDIT.md remediation: use config.DefaultBootstrapPeers.
	if len(cfg.BootstrapPeers) == 0 {
		// Import config package to access DefaultBootstrapPeers.
		// Note: Currently empty, but prepared for production deployment.
		cfg.BootstrapPeers = make([]string, 0) // Explicit empty for now
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		config:       cfg,
		ctx:          ctx,
		cancel:       cancel,
		subsystems:   &Subsystems{},
		initComplete: make(chan struct{}),
	}

	return app, nil
}

// Run starts the application and blocks until shutdown.
// It initializes all subsystems in dependency order and starts the event loop.
// Per TECHNICAL_IMPLEMENTATION.md §2, the initialization order is:
// Storage → Identity → Networking → Content → Anonymous → Pulse Map → Onboarding.
func (a *App) Run() error {
	startTime := time.Now()

	if err := a.checkNotRunning(); err != nil {
		return err
	}

	// Check for MURMUR_HEADLESS environment variable
	if os.Getenv("MURMUR_HEADLESS") == "1" {
		a.config.SkipUI = true
		fmt.Println("Running in headless mode (MURMUR_HEADLESS=1)")
	}

	fmt.Printf("MURMUR %s starting...\n", a.config.Version)

	if err := a.initializeSubsystems(); err != nil {
		return err
	}

	a.printStartupInfo()
	close(a.initComplete)

	elapsed := time.Since(startTime)
	fmt.Printf("[STARTUP] Application ready in %v\n", elapsed)

	if a.firstRun {
		a.startOnboarding()
	}

	return a.startRunMode()
}

func (a *App) checkNotRunning() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return errors.New("application already running")
	}
	a.running = true
	return nil
}

func (a *App) initializeSubsystems() error {
	t0 := time.Now()
	a.initEventBus()
	fmt.Printf("  [0/7] Event bus started (%v)\n", time.Since(t0))

	t1 := time.Now()
	if err := a.initStorage(); err != nil {
		return murerr.WrapStorageError(err)
	}
	fmt.Printf("  [1/7] Storage initialized (%v)\n", time.Since(t1))

	t2 := time.Now()
	if err := a.initIdentity(); err != nil {
		return murerr.WrapIdentityError(err)
	}
	fmt.Printf("  [2/7] Identity initialized (%v)\n", time.Since(t2))

	t3 := time.Now()
	if err := a.initNetworking(); err != nil {
		return murerr.WrapNetworkError(err)
	}
	fmt.Printf("  [3/7] Networking initialized (%v)\n", time.Since(t3))

	// Initialize health check endpoint if enabled
	if a.config.EnableHealthEndpoint {
		if err := a.initHealthServer(); err != nil {
			return fmt.Errorf("initializing health server: %w", err)
		}
	}

	t4 := time.Now()
	if err := a.initContent(); err != nil {
		return murerr.WrapContentError(err)
	}
	fmt.Printf("  [4/7] Content initialized (%v)\n", time.Since(t4))

	return a.initShroud()
}

func (a *App) initShroud() error {
	if err := a.initBeacon(); err != nil {
		return murerr.WrapBeaconError(err)
	}

	if a.config.EnableRelay {
		fmt.Println("  [5/7] Shroud initialized (relay mode)")
	} else {
		fmt.Println("  [5/7] Shroud initialized (client mode)")
	}

	fmt.Println("  [6-7] PulseMap/Onboarding: ready for lazy init")
	return nil
}

func (a *App) printStartupInfo() {
	fmt.Printf("MURMUR listening on %v\n", a.subsystems.Host.Addrs())
	fmt.Printf("Peer ID: %s\n", a.subsystems.Host.PeerID())

	if len(a.config.BootstrapPeers) == 0 {
		fmt.Println("Warning: No bootstrap peers configured. Running in isolated mode.")
	}
}

func (a *App) startRunMode() error {
	if a.config.CLIMode {
		return a.runCLI()
	}

	if !a.config.SkipUI {
		return a.runUI()
	}

	<-a.ctx.Done()
	return nil
}

// initEventBus creates and starts the event bus goroutine.
// Per TECHNICAL_IMPLEMENTATION.md §8, this is one of the ~8 persistent goroutines.
func (a *App) initEventBus() {
	a.subsystems.EventBus = NewEventBus(EventBusConfig{BufferSize: 256})

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Event bus goroutine exited")
		a.subsystems.EventBus.Start(a.ctx)
	}()
}

// initStorage initializes the Bbolt database.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, the database has buckets:
// identity, peers, waves, threads, shroud, resonance, config.
func (a *App) initStorage() error {
	dbPath := filepath.Join(a.config.DataDir, "murmur.db")
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	a.subsystems.Storage = db

	// Check if this is first run by looking for identity key.
	identityKey, err := db.Get(store.BucketIdentity, []byte("keypair"))
	if err != nil {
		return fmt.Errorf("checking identity: %w", err)
	}
	a.firstRun = identityKey == nil

	return nil
}

// initIdentity initializes or loads the Surface Layer identity keypair.
// Per SECURITY_PRIVACY.md, Surface identity uses Ed25519 for signatures.
func (a *App) initIdentity() error {
	if a.firstRun {
		// Generate new keypair for first run.
		kp, err := keys.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("generating keypair: %w", err)
		}
		a.subsystems.Identity = kp

		// Store the keypair (unencrypted for now; TODO: add passphrase).
		if err := a.subsystems.Storage.Put(
			store.BucketIdentity,
			[]byte("keypair"),
			kp.PrivateKey,
		); err != nil {
			return fmt.Errorf("storing keypair: %w", err)
		}
		fmt.Println("  -> Generated new identity")
	} else {
		// Load existing keypair.
		privKeyBytes, err := a.subsystems.Storage.Get(
			store.BucketIdentity,
			[]byte("keypair"),
		)
		if err != nil {
			return fmt.Errorf("loading keypair: %w", err)
		}
		if len(privKeyBytes) != 64 {
			return errors.New("invalid stored keypair length")
		}
		a.subsystems.Identity = &keys.KeyPair{
			PrivateKey: privKeyBytes,
			PublicKey:  privKeyBytes[32:],
		}
		fmt.Println("  -> Loaded existing identity")
	}
	return nil
}

// initNetworking initializes the libp2p host and GossipSub.
// Per NETWORK_ARCHITECTURE.md, uses Noise XX encryption and Kademlia DHT.
func (a *App) initNetworking() error {
	// Create libp2p host.
	hostCfg := transport.Config{
		PrivateKey:    a.subsystems.Identity.PrivateKey,
		ListenAddrs:   a.config.ListenAddrs,
		EnableDHT:     true,
		DHTServerMode: true,
	}
	host, err := transport.NewHost(a.ctx, hostCfg)
	if err != nil {
		return fmt.Errorf("creating host: %w", err)
	}
	a.subsystems.Host = host

	// Create GossipSub.
	ps, err := gossip.New(a.ctx, host.Host)
	if err != nil {
		host.Close()
		return fmt.Errorf("creating pubsub: %w", err)
	}
	a.subsystems.PubSub = ps

	// Join core topics per TECHNICAL_IMPLEMENTATION.md §3.1.
	for _, topic := range []string{
		gossip.TopicWaves,
		gossip.TopicIdentity,
		gossip.TopicShroud,
		gossip.TopicPulse,
	} {
		if _, err := ps.Join(topic); err != nil {
			host.Close()
			return fmt.Errorf("joining topic %s: %w", topic, err)
		}
	}

	return nil
}

// initHealthServer initializes the HTTP health check endpoint.
// Per AUDIT.md MEDIUM finding, this enables bootstrap node operators to monitor
// node status, peer connections, and subscribed topics.
func (a *App) initHealthServer() error {
	server := health.NewServer(a.subsystems.Host.Host, a.subsystems.PubSub)
	a.subsystems.HealthServer = server

	port := a.config.HealthEndpointPort
	if err := server.Start(a.ctx, port); err != nil {
		return fmt.Errorf("starting health server on port %d: %w", port, err)
	}

	fmt.Printf("  [3.5/7] Health server listening on port %d\n", port)
	return nil
}

// initContent initializes the content subsystem (Wave cache and GossipSub handlers).
// Per TECHNICAL_IMPLEMENTATION.md §3.1, handlers are registered for all core topics.
func (a *App) initContent() error {
	cache, handlers, err := a.setupContentComponents()
	if err != nil {
		return err
	}

	if err := a.registerContentHandlers(handlers); err != nil {
		return err
	}

	a.startContentGoroutines(cache, handlers)
	return nil
}

// setupContentComponents creates the cache and message handlers.
func (a *App) setupContentComponents() (*storage.Cache, *Handlers, error) {
	cache, err := storage.NewCache(a.subsystems.Storage)
	if err != nil {
		return nil, nil, fmt.Errorf("creating wave cache: %w", err)
	}
	a.subsystems.WaveCache = cache

	a.restorePersistedDifficulty(cache)

	handlers, err := NewHandlers(HandlersConfig{
		Cache:  cache,
		Beacon: a.subsystems.Beacon,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating handlers: %w", err)
	}
	a.subsystems.Handlers = handlers

	return cache, handlers, nil
}

// restorePersistedDifficulty loads and applies the saved PoW difficulty.
func (a *App) restorePersistedDifficulty(cache *storage.Cache) {
	if persistedDifficulty := cache.LoadPersistedDifficulty(); persistedDifficulty > 0 {
		cfg := pow.GetGlobalConfig()
		if cfg.SetStandard(persistedDifficulty) {
			fmt.Printf("Restored persisted PoW difficulty: %d bits\n", persistedDifficulty)
		}
	}
}

// registerContentHandlers registers all content-related GossipSub handlers.
func (a *App) registerContentHandlers(handlers *Handlers) error {
	if err := handlers.RegisterAll(a.ctx, a.subsystems.PubSub); err != nil {
		return fmt.Errorf("registering handlers: %w", err)
	}

	if err := handlers.RegisterAnonymousMechanics(a.ctx, a.subsystems.PubSub); err != nil {
		return fmt.Errorf("registering anonymous mechanics: %w", err)
	}

	return nil
}

// startContentGoroutines launches background workers for content management.
func (a *App) startContentGoroutines(cache *storage.Cache, handlers *Handlers) {
	a.startDedupRotation(handlers)
	a.startGarbageCollection(cache)
	a.startMemoryMonitor(cache)
	a.startFirstWeekNudges()
}

// startDedupRotation launches the deduplication filter rotation goroutine.
func (a *App) startDedupRotation(handlers *Handlers) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Deduplication rotation goroutine exited")
		handlers.StartDedupRotation(a.ctx)
	}()
}

// startGarbageCollection launches the cache garbage collection goroutine.
func (a *App) startGarbageCollection(cache *storage.Cache) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] GC goroutine exited")
		cache.StartGC(a.ctx, storage.GCInterval)
	}()
}

// startMemoryMonitor launches the memory budget enforcement goroutine.
func (a *App) startMemoryMonitor(cache *storage.Cache) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Memory monitor goroutine exited")
		a.runMemoryMonitor(cache)
	}()
}

// startFirstWeekNudges launches the first-week user onboarding nudges goroutine.
func (a *App) startFirstWeekNudges() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Nudge loop goroutine exited")
		a.runNudgeLoop()
	}()
}

// initBeacon initializes the Shroud beacon and circuit manager.
// Per SHADOW_GRADIENT.md, all nodes can use Shroud circuits; only relays advertise.
func (a *App) initBeacon() error {
	// Always create a beacon for relay discovery (receiving ads from others).
	beacon, err := shroud.NewBeacon()
	if err != nil {
		return fmt.Errorf("creating beacon: %w", err)
	}
	a.subsystems.Beacon = beacon

	// Enable relay mode if configured.
	if a.config.EnableRelay {
		bandwidth := a.config.RelayBandwidth
		if bandwidth == 0 {
			bandwidth = 10 * 1024 * 1024 // Default 10 MiB/s.
		}
		peerID := a.subsystems.Host.PeerID().String()
		beacon.EnableRelay(peerID, bandwidth)

		// Start periodic advertisement broadcasting.
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			defer fmt.Println("[SHUTDOWN] Beacon loop goroutine exited")
			a.runBeaconLoop()
		}()
	}

	// Wire up relay advertisement handler to process incoming advertisements.
	a.subsystems.Handlers.SetRelayAdCallback(func(ad *pb.RelayAdvertisement) {
		// Extract peer ID from advertisement addrs if possible.
		relayPeerID := ""
		if len(ad.Addrs) > 0 {
			relayPeerID = ad.Addrs[0] // Simplified: use first addr as identifier.
		}
		beacon.ProcessAdvertisement(ad, relayPeerID)

		// Emit event for relay discovery.
		if a.subsystems.EventBus != nil {
			a.subsystems.EventBus.Emit(Event{
				Type: EventShroudRelayDiscovered,
				Payload: ShroudEvent{
					RelayPeerID: relayPeerID,
				},
			})
		}
	})

	// Start periodic relay pruning.
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Relay prune loop goroutine exited")
		a.runRelayPruneLoop()
	}()

	// Create circuit manager for building Shroud circuits.
	// Exclude our own peer ID from relay selection.
	selfPeerID := a.subsystems.Host.PeerID().String()
	a.subsystems.CircuitManager = shroud.NewCircuitManager(beacon, []string{selfPeerID})

	// Start circuit rotation timer per TECHNICAL_IMPLEMENTATION.md §8.
	// This is one of the ~8 persistent goroutines.
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer fmt.Println("[SHUTDOWN] Circuit rotation goroutine exited")
		a.subsystems.CircuitManager.StartRotation(a.ctx)
	}()

	return nil
}

// runBeaconLoop broadcasts relay advertisements periodically.
// Per shroud.BeaconInterval (5 minutes), relays advertise their availability.
func (a *App) runBeaconLoop() {
	ticker := time.NewTicker(shroud.BeaconInterval)
	defer ticker.Stop()

	a.broadcastOnStartup()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.broadcastPeriodically()
		}
	}
}

// broadcastOnStartup sends initial relay advertisement immediately.
func (a *App) broadcastOnStartup() {
	if err := a.BroadcastRelayAdvertisement(a.ctx); err != nil && err != ErrNotRelay {
		// Log error but continue
	}
}

// broadcastPeriodically sends relay advertisements on timer.
func (a *App) broadcastPeriodically() {
	if err := a.BroadcastRelayAdvertisement(a.ctx); err != nil && err != ErrNotRelay {
		// Log error but continue
	}
}

// runRelayPruneLoop periodically removes expired relays.
// Per SHADOW_GRADIENT.md, relays not seen for 2x advertisement TTL are pruned.
func (a *App) runRelayPruneLoop() {
	ticker := time.NewTicker(shroud.AdvertisementTTL)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			beacon := a.subsystems.Beacon
			a.mu.RUnlock()

			if beacon != nil {
				beacon.PruneExpiredRelays(2 * shroud.AdvertisementTTL)
			}
		}
	}
}

// runMemoryMonitor periodically checks memory usage and database size, triggering eviction when needed.
// Per AUDIT.md HIGH "No memory budget enforcement", this enforces the
// 256 MiB memory budget stated in TECHNICAL_IMPLEMENTATION.md §6.
// Per ROADMAP.md line 835, monitors Bbolt database size (<50 MiB budget).
func (a *App) runMemoryMonitor(cache *storage.Cache) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.checkMemory(cache)
			a.checkDatabaseSize()
		}
	}
}

// checkMemory monitors memory usage and triggers eviction if needed.
func (a *App) checkMemory(cache *storage.Cache) {
	const (
		targetMemory  = 200 * 1024 * 1024 // 200 MiB - trigger eviction
		warningMemory = 240 * 1024 * 1024 // 240 MiB - log warning
	)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := m.Alloc / (1024 * 1024)

	if m.Alloc < targetMemory {
		return
	}

	// Memory pressure detected - evict oldest Waves.
	fmt.Printf("Memory pressure: %d MiB allocated (target 200 MiB), evicting Waves...\n", allocMB)
	evicted := cache.EvictOldest(1000)
	fmt.Printf("  -> Evicted %d oldest Waves\n", evicted)

	// Check if eviction was sufficient.
	runtime.ReadMemStats(&m)
	allocMB = m.Alloc / (1024 * 1024)

	if m.Alloc >= warningMemory {
		fmt.Printf("WARNING: Memory still high after eviction: %d MiB (target 256 MiB)\n", allocMB)
		fmt.Println("  -> Consider stopping to accept new Waves or increasing eviction threshold")
	}

	// Force GC to reclaim freed memory.
	runtime.GC()
}

// checkDatabaseSize monitors database file size and logs warnings if needed.
func (a *App) checkDatabaseSize() {
	const (
		targetDBSize  = 40 * 1024 * 1024 // 40 MiB - trigger cleanup
		warningDBSize = 45 * 1024 * 1024 // 45 MiB - log warning
	)

	if a.subsystems.Storage == nil {
		return
	}

	dbSize, err := a.subsystems.Storage.DatabaseSize()
	if err != nil {
		return
	}

	if dbSize < targetDBSize {
		return
	}

	dbSizeMB := dbSize / (1024 * 1024)
	fmt.Printf("Database size: %d MiB (target 40 MiB), cleanup recommended\n", dbSizeMB)

	if dbSize >= warningDBSize {
		fmt.Printf("WARNING: Database size %d MiB approaching limit (budget 50 MiB)\n", dbSizeMB)
	}
}

// Close shuts down the application gracefully with a 10-second timeout.
// It cancels the context and waits for all goroutines to complete.
// Subsystems are closed in reverse initialization order.
// Per ROADMAP.md line 814: ordered subsystem teardown with timeout.
func (a *App) Close() error {
	wasRunning := a.markStopped()
	pulseMapUI := a.subsystems.PulseMapUI

	a.shutdownPulseMapUI(pulseMapUI)
	a.cancel()

	if wasRunning {
		a.waitForGoroutines()
		return a.closeSubsystems()
	}

	return nil
}

// markStopped atomically marks the app as stopped and returns previous state.
func (a *App) markStopped() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	wasRunning := a.running
	a.running = false
	return wasRunning
}

// shutdownPulseMapUI signals the Pulse Map UI to shut down if present.
func (a *App) shutdownPulseMapUI(pulseMapUI interface{}) {
	if pulseMapUI == nil {
		return
	}

	type shutdowner interface {
		Shutdown()
	}
	if ui, ok := pulseMapUI.(shutdowner); ok {
		ui.Shutdown()
	}
}

// waitForGoroutines waits for all goroutines to complete with timeout.
func (a *App) waitForGoroutines() {
	const shutdownTimeout = 10 * time.Second

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		fmt.Fprintln(os.Stderr, "WARNING: Graceful shutdown timeout reached after 10 seconds")
	}
}

// closeSubsystems closes all subsystems in reverse initialization order.
// Per AUDIT.md H1, log duration for any subsystem taking >2s to close.
func (a *App) closeSubsystems() error {
	var errs []error

	errs = appendCloseError(errs, a.closeSubsystemWithTiming("WaveCache", a.closeWaveCache))
	errs = appendCloseError(errs, a.closeSubsystemWithTiming("PubSub", a.closePubSub))
	errs = appendCloseError(errs, a.closeSubsystemWithTiming("Host", a.closeHost))
	a.zeroIdentity()
	errs = appendCloseError(errs, a.closeSubsystemWithTiming("Storage", a.closeStorage))

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// closeSubsystemWithTiming wraps a closer function with timing instrumentation.
// Logs a warning if the close operation exceeds 2 seconds.
func (a *App) closeSubsystemWithTiming(name string, closer func() error) error {
	start := time.Now()
	err := closer()
	duration := time.Since(start)

	if duration > 2*time.Second {
		fmt.Fprintf(os.Stderr, "[SHUTDOWN] SLOW: %s took %v to close (expected <2s)\n", name, duration)
	}

	return err
}

// appendCloseError appends non-nil errors to the slice.
func appendCloseError(errs []error, err error) []error {
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

// closeWaveCache closes the wave cache subsystem.
func (a *App) closeWaveCache() error {
	if a.subsystems.WaveCache == nil {
		return nil
	}
	if err := a.subsystems.WaveCache.Close(); err != nil {
		return fmt.Errorf("closing wave cache: %w", err)
	}
	return nil
}

// closePubSub closes the pubsub subsystem.
func (a *App) closePubSub() error {
	if a.subsystems.PubSub == nil {
		return nil
	}
	if err := a.subsystems.PubSub.Close(); err != nil {
		return fmt.Errorf("closing pubsub: %w", err)
	}
	return nil
}

// closeHost closes the libp2p host.
func (a *App) closeHost() error {
	if a.subsystems.Host == nil {
		return nil
	}
	if err := a.subsystems.Host.Close(); err != nil {
		return fmt.Errorf("closing host: %w", err)
	}
	return nil
}

// zeroIdentity zeroes the identity keypair.
func (a *App) zeroIdentity() {
	if a.subsystems.Identity != nil {
		a.subsystems.Identity.ZeroKeyPair()
	}
}

// closeStorage closes the storage subsystem.
func (a *App) closeStorage() error {
	if a.subsystems.Storage == nil {
		return nil
	}
	if err := a.subsystems.Storage.Close(); err != nil {
		return fmt.Errorf("closing storage: %w", err)
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

// Subsystems returns the initialized subsystems.
// Returns nil if the application has not been started.
func (a *App) Subsystems() *Subsystems {
	return a.subsystems
}

// IsFirstRun returns true if this is the first application run.
func (a *App) IsFirstRun() bool {
	return a.firstRun
}

// WaitReady blocks until subsystem initialization is complete or context is canceled.
// Returns an error if the context is canceled before init completes.
func (a *App) WaitReady(ctx context.Context) error {
	select {
	case <-a.initComplete:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// startOnboarding initializes and starts the onboarding flow.
// Per AUDIT.md remediation, this guides new users through identity creation,
// network bootstrap, and Pulse Map exploration on first run.
func (a *App) startOnboarding() {
	fmt.Println("Starting onboarding flow (first run detected)...")

	// Import here to avoid circular dependency.
	// The flow package depends on app types through callbacks.
	onboardingFlow := newOnboardingFlow(a)

	a.mu.Lock()
	a.subsystems.OnboardingFlow = onboardingFlow
	a.mu.Unlock()

	// Start the onboarding flow.
	onboardingFlow.Start()

	fmt.Printf("Onboarding flow started. Current phase: %s\n", onboardingFlow.CurrentPhase())
}

// newOnboardingFlow creates a new onboarding flow controller with callbacks.
// This is a helper to avoid importing pkg/onboarding/flow in the app package
// (which would create dependency complexity).
func newOnboardingFlow(a *App) onboardingFlowController {
	// Callbacks are defined as no-ops for now. Full integration with UI
	// screens (pkg/onboarding/screens) requires wiring to the Pulse Map game loop,
	// which will be done in a subsequent task once the compose panel integration
	// is validated.
	return newFlowController(flowCallbacks{
		onPhaseStart: func(phase int) {
			fmt.Printf("Onboarding: Starting phase %d\n", phase)
		},
		onPhaseComplete: func(phase int) {
			fmt.Printf("Onboarding: Completed phase %d\n", phase)
		},
		onFlowComplete: func(totalTime time.Duration) {
			fmt.Printf("Onboarding: Complete! Total time: %v\n", totalTime)
			// Mark first run as complete in storage.
			if err := a.subsystems.Storage.Put(store.BucketConfig, []byte("first_run_complete"), []byte("true")); err != nil {
				fmt.Printf("Warning: Failed to persist first-run flag: %v\n", err)
			}
			a.firstRun = false
		},
		onError: func(phase int, err error) {
			fmt.Printf("Onboarding: Error in phase %d: %v\n", phase, err)
		},
	})
}

// onboardingFlowController is an interface abstraction over flow.Controller
// to avoid importing pkg/onboarding/flow directly in the app package.
type onboardingFlowController interface {
	Start()
	CurrentPhase() onboardingPhase
	CompleteCurrentPhase()
	IsComplete() bool
}

// onboardingPhase is an interface abstraction over flow.Phase.
type onboardingPhase interface {
	String() string
}

// flowCallbacks mirrors flow.Callbacks but uses int for phases to avoid import.
type flowCallbacks struct {
	onPhaseStart    func(phase int)
	onPhaseComplete func(phase int)
	onFlowComplete  func(totalTime time.Duration)
	onError         func(phase int, err error)
}

// newFlowController creates a flow.Controller with the provided callbacks.
// This function is implemented in onboarding_glue.go to avoid circular imports.
func newFlowController(callbacks flowCallbacks) onboardingFlowController {
	// Implementation moved to onboarding_glue.go to avoid circular dependency.
	return newFlowControllerImpl(callbacks)
}
