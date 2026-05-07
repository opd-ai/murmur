//go:build js && wasm

// Package main provides the browser/WASM entry point for MURMUR.
package main

import (
	"fmt"
	"syscall/js"

	"github.com/opd-ai/murmur/pkg/game"
)

// Version is the current version of MURMUR. Set by build flags.
var Version = "0.0.0-alpha"

// Commit is the git commit hash. Set by build flags.
var Commit = "unknown"

func main() {
	// Expose version metadata to JavaScript
	js.Global().Set("murmurVersion", js.ValueOf(Version))
	js.Global().Set("murmurCommit", js.ValueOf(Commit))

	// Initialize the runtime
	runtime := game.NewRuntime(game.RuntimeConfig{
		Platform: game.PlatformWASM,
		Version:  Version,
		Commit:   Commit,
	})

	if err := runtime.Run(); err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("murmur wasm runtime failed: %v", err))
		// Signal initialization failure to boot.js
		triggerReadyCallback(false, fmt.Sprintf("Runtime error: %v", err))
		select {}
		return
	}

	// Signal successful initialization to boot.js so it can complete the loading flow
	triggerReadyCallback(true, "")

	// Keep the Go runtime alive for event handling, network operations, etc.
	select {}
}

// triggerReadyCallback invokes the onRuntimeReady callback set by boot.js.
// This allows boot.js to properly await initialization without blocking.
func triggerReadyCallback(success bool, errMsg string) {
	murmurNS := js.Global().Get("murmur")
	if murmurNS.IsUndefined() {
		return // Boot.js hasn't set up the namespace yet, skip
	}

	callback := murmurNS.Get("onRuntimeReady")
	if callback.IsNull() || callback.IsUndefined() {
		return // No callback registered
	}

	if success {
		callback.Invoke()
	} else {
		callback.Invoke(js.ValueOf(errMsg))
	}
}
