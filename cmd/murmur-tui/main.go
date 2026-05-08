package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/opd-ai/murmur/pkg/tui"
)

var (
	version = flag.Bool("version", false, "Print version and exit")
)

// Version is set by ldflags.
var Version = "0.0.0-alpha"

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("murmur-tui %s\n", Version)
		return
	}

	program := tui.NewProgram(tui.Config{})
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "murmur-tui: %v\n", err)
		os.Exit(1)
	}
}
