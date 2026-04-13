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

// appNew is a variable to allow testing with a mock app creator.
var appNew = app.New

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "murmur: %v\n", err)
		os.Exit(1)
	}
}

// run initializes and starts the MURMUR application.
func run() error {
	return runWithConfig(app.Config{
		Version: Version,
	})
}

// runWithConfig initializes and starts the MURMUR application with the given config.
func runWithConfig(cfg app.Config) error {
	// Create application with the given configuration.
	application, err := appNew(cfg)
	if err != nil {
		return fmt.Errorf("creating application: %w", err)
	}
	defer application.Close()

	// Start the application (blocks until shutdown).
	return application.Run()
}
