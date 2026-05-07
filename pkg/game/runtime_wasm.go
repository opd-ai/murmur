//go:build js && wasm

package game

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/opd-ai/murmur/pkg/app"
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

	// Initialize the application with the same configuration as desktop.
	// WASM shares the full UI and networking stack with the desktop target.
	application, err := app.New(app.Config{
		Version: r.cfg.Version,
		// WASM runs full UI mode (no CLI option available in browser context)
	})
	if err != nil {
		return fmt.Errorf("creating WASM app runtime: %w", err)
	}
	defer application.Close()

	// Run the application (blocks in the Ebitengine game loop).
	// In the browser, ebiten.RunGame() yields to the event loop as needed.
	return application.Run()
}
