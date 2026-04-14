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
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/shroud"
	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
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

	// EnableRelay enables this node as a Shroud relay.
	// Relays help route anonymous traffic for others.
	EnableRelay bool

	// RelayBandwidth is the advertised bandwidth for relay operations (bytes/sec).
	// Only relevant if EnableRelay is true. Defaults to 10 MiB/s.
	RelayBandwidth uint64
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
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("application already running")
	}
	a.running = true
	a.mu.Unlock()

	fmt.Printf("MURMUR %s starting...\n", a.config.Version)

	// Initialize event bus first (other subsystems may emit events).
	a.initEventBus()
	fmt.Println("  [0/7] Event bus started")

	// Initialize subsystems in dependency order.
	if err := a.initStorage(); err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	fmt.Println("  [1/7] Storage initialized")

	if err := a.initIdentity(); err != nil {
		return fmt.Errorf("initializing identity: %w", err)
	}
	fmt.Println("  [2/7] Identity initialized")

	if err := a.initNetworking(); err != nil {
		return fmt.Errorf("initializing networking: %w", err)
	}
	fmt.Println("  [3/7] Networking initialized")

	// Initialize content subsystem (Wave cache and handlers).
	if err := a.initContent(); err != nil {
		return fmt.Errorf("initializing content: %w", err)
	}
	fmt.Println("  [4/7] Content initialized")

	// Initialize Shroud beacon and circuit manager.
	if err := a.initBeacon(); err != nil {
		return fmt.Errorf("initializing beacon: %w", err)
	}
	if a.config.EnableRelay {
		fmt.Println("  [5/7] Shroud initialized (relay mode)")
	} else {
		fmt.Println("  [5/7] Shroud initialized (client mode)")
	}

	// Pulse Map and Onboarding subsystems are initialized
	// by their respective packages when messages arrive or UI events occur.
	fmt.Println("  [6-7] PulseMap/Onboarding: ready for lazy init")

	fmt.Printf("MURMUR listening on %v\n", a.subsystems.Host.Addrs())
	fmt.Printf("Peer ID: %s\n", a.subsystems.Host.PeerID())

	if len(a.config.BootstrapPeers) == 0 {
		fmt.Println("Warning: No bootstrap peers configured. Running in isolated mode.")
	}

	// Signal that initialization is complete.
	close(a.initComplete)

	// Block until context is canceled.
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

// initContent initializes the content subsystem (Wave cache and GossipSub handlers).
// Per TECHNICAL_IMPLEMENTATION.md §3.1, handlers are registered for all core topics.
func (a *App) initContent() error {
	// Create Wave cache.
	cache, err := storage.NewCache(a.subsystems.Storage)
	if err != nil {
		return fmt.Errorf("creating wave cache: %w", err)
	}
	a.subsystems.WaveCache = cache

	// Create message handlers.
	handlers, err := NewHandlers(HandlersConfig{
		Cache: cache,
	})
	if err != nil {
		return fmt.Errorf("creating handlers: %w", err)
	}
	a.subsystems.Handlers = handlers

	// Register handlers for all topics.
	if err := handlers.RegisterAll(a.ctx, a.subsystems.PubSub); err != nil {
		return fmt.Errorf("registering handlers: %w", err)
	}

	// Start garbage collection goroutine.
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		cache.StartGC(a.ctx, storage.GCInterval)
	}()

	return nil
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
		a.subsystems.CircuitManager.StartRotation(a.ctx)
	}()

	return nil
}

// runBeaconLoop broadcasts relay advertisements periodically.
// Per shroud.BeaconInterval (5 minutes), relays advertise their availability.
func (a *App) runBeaconLoop() {
	ticker := time.NewTicker(shroud.BeaconInterval)
	defer ticker.Stop()

	// Broadcast immediately on startup.
	if err := a.BroadcastRelayAdvertisement(a.ctx); err != nil && err != ErrNotRelay {
		// Log error but continue.
	}

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.BroadcastRelayAdvertisement(a.ctx); err != nil && err != ErrNotRelay {
				// Log error but continue.
			}
		}
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

// Close shuts down the application gracefully.
// It cancels the context and waits for all goroutines to complete.
// Subsystems are closed in reverse initialization order.
func (a *App) Close() error {
	a.mu.Lock()
	wasRunning := a.running
	a.running = false
	a.mu.Unlock()

	// Always cancel the context to signal shutdown.
	a.cancel()

	// Only close subsystems if the app was running.
	if wasRunning {
		a.wg.Wait()
		return a.closeSubsystems()
	}

	return nil
}

// closeSubsystems closes all subsystems in reverse initialization order.
func (a *App) closeSubsystems() error {
	var errs []error

	errs = appendCloseError(errs, a.closeWaveCache())
	errs = appendCloseError(errs, a.closePubSub())
	errs = appendCloseError(errs, a.closeHost())
	a.zeroIdentity()
	errs = appendCloseError(errs, a.closeStorage())

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
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
