package app

import (
	"fmt"

	"github.com/opd-ai/murmur/pkg/cli"
)

// runCLI starts the interactive command-line interface.
// Per AUDIT.md remediation, this provides a REPL for Wave creation
// and peer management without requiring the GUI.
func (a *App) runCLI() error {
	fmt.Println("\nStarting CLI mode...")

	repl, err := cli.NewREPL(cli.Config{
		Host:      a.subsystems.Host,
		PubSub:    a.subsystems.PubSub,
		KeyPair:   a.subsystems.Identity,
		WaveCache: a.subsystems.WaveCache,
	})
	if err != nil {
		return fmt.Errorf("creating REPL: %w", err)
	}

	return repl.Run()
}
