// Package main provides the mobile entry point for MURMUR.
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

// Commit is the git commit hash. Set by build flags.
var Commit = "unknown"

// appNew is a variable to allow testing with a mock app creator.
var appNew = app.New

// Command-line flags (package-level to avoid redefinition on multiple calls).
var (
	cliMode      = flag.Bool("cli", false, "Run in CLI mode (interactive REPL)")
	enableHealth = flag.Bool("enable-health", false, "Enable HTTP health check endpoint (for bootstrap nodes)")
	healthPort   = flag.Int("health-port", 8080, "Port for health check endpoint")
	invite       = flag.String("invite", "", "Accept an invitation (murmur://invite/... URI)")
	version      = flag.Bool("version", false, "Print version and exit")
)

func main() {
	if err := run(); err != nil {
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
	if !flag.Parsed() {
		flag.Parse()
	}

	if *version {
		fmt.Printf("MURMUR %s (commit %s)\n", Version, Commit)
		return nil
	}

	return runWithConfig(app.Config{
		Version:              Version,
		SkipUI:               *cliMode,
		CLIMode:              *cliMode,
		EnableHealthEndpoint: *enableHealth,
		HealthEndpointPort:   *healthPort,
		InvitationURI:        *invite,
	})
}

// runWithConfig initializes and starts the MURMUR application with the given config.
func runWithConfig(cfg app.Config) error {
	application, err := appNew(cfg)
	if err != nil {
		return fmt.Errorf("creating application: %w", err)
	}
	defer application.Close()

	return application.Run()
}
