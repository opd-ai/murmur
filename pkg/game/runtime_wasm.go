//go:build js && wasm

package game

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/store"
)

type wasmRuntime struct {
	cfg RuntimeConfig
}

func newWASMRuntime(cfg RuntimeConfig) Runtime {
	return &wasmRuntime{cfg: cfg}
}

func (r *wasmRuntime) Run() error {
	if js.Global().IsUndefined() {
		return errors.New("browser runtime is unavailable")
	}

	// Expose version metadata to JavaScript before app initialization
	js.Global().Set("murmurVersion", js.ValueOf(r.cfg.Version))
	js.Global().Set("murmurCommit", js.ValueOf(r.cfg.Commit))

	// Initialize a minimal WASM application with browser-native storage.
	// This provides the Pulse Map UI and identity management without requiring
	// file system access or native networking stacks.
	wasmApp, err := newWASMApp(r.cfg.Version)
	if err != nil {
		logToConsole(fmt.Sprintf("WASM app initialization failed: %v", err))
		return err
	}

	// Run the application (blocks in the Ebitengine game loop).
	// In the browser, ebiten.RunGame() yields to the event loop as needed.
	return wasmApp.Run()
}

// logToConsole sends a message to the browser console for debugging.
func logToConsole(msg string) {
	js.Global().Get("console").Call("log", msg)
}

// wasmApp is a minimal application variant for browser environments.
type wasmApp struct {
	version  string
	storage  *store.BrowserStorage
	identity *keys.KeyPair
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	running  bool
}

// newWASMApp creates a new WASM application with browser-native storage.
func newWASMApp(version string) (*wasmApp, error) {
	ctx, cancel := context.WithCancel(context.Background())

	storage := store.NewBrowserStorage()

	wa := &wasmApp{
		version: version,
		storage: storage,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize identity (load or create)
	if err := wa.initIdentity(); err != nil {
		cancel()
		return nil, fmt.Errorf("initializing identity: %w", err)
	}

	logToConsole("WASM app initialized successfully")
	return wa, nil
}

// initIdentity initializes or loads the Surface Layer identity keypair.
func (wa *wasmApp) initIdentity() error {
	// Check if identity already exists
	identityKey, err := wa.storage.Get(store.BucketIdentity, []byte("keypair"))
	if err != nil {
		return fmt.Errorf("checking identity: %w", err)
	}

	if identityKey == nil {
		// Generate new keypair for first run
		kp, err := keys.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("generating keypair: %w", err)
		}
		wa.identity = kp

		// Store the keypair
		if err := wa.storage.Put(
			store.BucketIdentity,
			[]byte("keypair"),
			kp.PrivateKey,
		); err != nil {
			return fmt.Errorf("storing keypair: %w", err)
		}
		logToConsole("Generated new WASM identity")
	} else {
		// Load existing keypair
		if len(identityKey) != 64 {
			return errors.New("invalid stored keypair length")
		}
		wa.identity = &keys.KeyPair{
			PrivateKey: identityKey,
			PublicKey:  identityKey[32:],
		}
		logToConsole("Loaded existing WASM identity")
	}

	return nil
}

// Run starts the WASM application runtime and publishes ready-state metadata.
// The browser entrypoint keeps the Go process alive after this method returns.
func (wa *wasmApp) Run() error {
	wa.mu.Lock()
	if wa.running {
		wa.mu.Unlock()
		return errors.New("WASM app already running")
	}
	wa.running = true
	wa.mu.Unlock()

	if err := wa.publishRuntimeState(); err != nil {
		return err
	}

	wa.registerCloseHook()
	wa.startHeartbeat()

	logToConsole("WASM app runtime initialized")
	return nil
}

func (wa *wasmApp) publishRuntimeState() error {
	if wa.identity == nil {
		return errors.New("WASM identity is not initialized")
	}

	murmurNS := js.Global().Get("murmur")
	if murmurNS.IsUndefined() || murmurNS.IsNull() {
		obj := js.Global().Get("Object")
		if obj.IsUndefined() {
			return errors.New("browser Object constructor unavailable")
		}
		murmurNS = obj.New()
		js.Global().Set("murmur", murmurNS)
	}

	murmurNS.Set("runtimeVersion", wa.version)
	murmurNS.Set("identityPublicKeyHex", hex.EncodeToString(wa.identity.PublicKey))
	murmurNS.Set("runtimeStartedAt", time.Now().Unix())
	return nil
}

func (wa *wasmApp) registerCloseHook() {
	cleanup := js.FuncOf(func(this js.Value, args []js.Value) any {
		_ = wa.Close()
		return nil
	})

	js.Global().Set("murmurClose", cleanup)
	js.Global().Call("addEventListener", "beforeunload", cleanup)
}

func (wa *wasmApp) startHeartbeat() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-wa.ctx.Done():
				return
			case <-ticker.C:
				if wa.storage != nil {
					_ = wa.storage.Put(store.BucketConfig, []byte("wasm_last_seen"), []byte(fmt.Sprintf("%d", time.Now().Unix())))
				}
			}
		}
	}()
}

// Close shuts down the WASM application.
func (wa *wasmApp) Close() error {
	wa.mu.Lock()
	defer wa.mu.Unlock()

	wa.cancel()
	if wa.storage != nil {
		wa.storage.Close()
	}
	return nil
}
