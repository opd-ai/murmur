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
	runtime := game.NewRuntime(game.RuntimeConfig{
		Platform: game.PlatformWASM,
		Version:  Version,
		Commit:   Commit,
	})
	js.Global().Set("murmurReady", js.ValueOf(true))

	if err := runtime.Run(); err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("murmur wasm runtime failed: %v", err))
	}

	select {}
}
