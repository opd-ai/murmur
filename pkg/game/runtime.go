// Package game provides shared runtime bootstrapping for desktop and browser targets.
package game

import "fmt"

// Platform identifies the runtime target.
type Platform string

const (
	// PlatformDesktop runs with native OS windowing and networking.
	PlatformDesktop Platform = "desktop"
	// PlatformWASM runs in browser via WebAssembly.
	PlatformWASM Platform = "wasm"
)

// RuntimeConfig configures runtime selection and build metadata.
type RuntimeConfig struct {
	Platform Platform
	Version  string
	Commit   string
}

// Runtime is the target-specific game launcher.
type Runtime interface {
	Run() error
}

// NewRuntime creates a runtime for the configured target platform.
func NewRuntime(cfg RuntimeConfig) Runtime {
	switch cfg.Platform {
	case PlatformWASM:
		return newWASMRuntime(cfg)
	default:
		return newDesktopRuntime(cfg)
	}
}

// ErrUnsupportedPlatform reports invalid runtime platform configuration.
func ErrUnsupportedPlatform(platform Platform) error {
	return fmt.Errorf("unsupported runtime platform: %s", platform)
}

type unsupportedRuntime struct {
	cfg RuntimeConfig
}

func (r *unsupportedRuntime) Run() error {
	return ErrUnsupportedPlatform(r.cfg.Platform)
}
