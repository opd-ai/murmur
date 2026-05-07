//go:build js && wasm

package game

import (
	"errors"
	"syscall/js"
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

	// Expose version metadata to JavaScript
	js.Global().Set("murmurVersion", js.ValueOf(r.cfg.Version))
	js.Global().Set("murmurCommit", js.ValueOf(r.cfg.Commit))

	// TODO: WASM requires custom architecture with browser-native storage (localStorage/IndexedDB)
	// and WebRTC-based networking instead of libp2p. Desktop app incompatible due to:
	// 1. bbolt requires Unix syscalls (file locking) unavailable in WASM
	// 2. go-libp2p requires native TCP/UDP and has WebRTC API incompatibilities
	// See AUDIT.md for detailed analysis.

	return nil
}
