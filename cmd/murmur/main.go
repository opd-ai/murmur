// Package main provides the entry point for the MURMUR decentralized social network.
// MURMUR is a peer-to-peer social network with dual-layer identity architecture.
// See README.md and DESIGN_DOCUMENT.md for the complete specification.
package main

import (
	"fmt"
	"os"

	"github.com/opd-ai/murmur/pkg/app"
)

// Version is the current version of MURMUR. Set by build flags.
var Version = "0.0.0-alpha"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "murmur: %v\n", err)
		os.Exit(1)
	}
}

// run initializes and starts the MURMUR application.
func run() error {
	// Create application with default configuration.
	application, err := app.New(app.Config{
		Version: Version,
	})
	if err != nil {
		return fmt.Errorf("creating application: %w", err)
	}
	defer application.Close()

	// Start the application (blocks until shutdown).
	return application.Run()
}
