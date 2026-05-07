//go:build js && wasm

package game

func newDesktopRuntime(cfg RuntimeConfig) Runtime {
	return &unsupportedRuntime{cfg: cfg}
}
