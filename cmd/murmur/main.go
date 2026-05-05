// Package main provides the entry point for the MURMUR decentralized social network.
// MURMUR is a peer-to-peer social network with dual-layer identity architecture.
// See README.md and DESIGN_DOCUMENT.md for the complete specification.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/opd-ai/murmur/pkg/app"
	"github.com/opd-ai/murmur/pkg/murerr"
)

// Version is the current version of MURMUR. Set by build flags.
var Version = "0.0.0-alpha"

// appNew is a variable to allow testing with a mock app creator.
var appNew = app.New

func main() {
	if err := run(); err != nil {
		// Check if it's an InitError with formatting.
		var initErr *murerr.InitError
		if errors.As(err, &initErr) {
			fmt.Fprint(os.Stderr, initErr.Format())
		} else {
			fmt.Fprintf(os.Stderr, "murmur: %v\n", err)
		}
		os.Exit(1)
	}
}

// run initializes and starts the MURMUR application.
func run() error {
	// Parse command-line flags.
	cliMode := flag.Bool("cli", false, "Run in CLI mode (interactive REPL)")
	enableHealth := flag.Bool("enable-health", false, "Enable HTTP health check endpoint (for bootstrap nodes)")
	healthPort := flag.Int("health-port", 8080, "Port for health check endpoint")
	invite := flag.String("invite", "", "Accept an invitation (murmur://invite/... URI)")
	flag.Parse()

	return runWithConfig(app.Config{
		Version:              Version,
		SkipUI:               *cliMode, // Skip UI in CLI mode
		CLIMode:              *cliMode,
		EnableHealthEndpoint: *enableHealth,
		HealthEndpointPort:   *healthPort,
		InvitationURI:        *invite,
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
