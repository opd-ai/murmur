//go:build !js

package game

func newWASMRuntime(cfg RuntimeConfig) Runtime {
	return &unsupportedRuntime{cfg: cfg}
}
