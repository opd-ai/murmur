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

	js.Global().Set("murmurVersion", js.ValueOf(r.cfg.Version))
	js.Global().Set("murmurCommit", js.ValueOf(r.cfg.Commit))
	return nil
}
