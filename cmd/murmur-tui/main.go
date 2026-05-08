package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/opd-ai/murmur/pkg/tui"
	"github.com/opd-ai/murmur/pkg/tui/bridge"
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

	inbound := make(chan bridge.EventMsg, 64)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			inbound <- bridge.EventMsg{Type: "HeartbeatReceived"}
			inbound <- bridge.EventMsg{Type: "WaveReceived"}
		}
	}()

	program := tui.NewProgram(tui.Config{
		EventStream: bridge.NewEventStream(inbound, func() {}),
	})
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "murmur-tui: %v\n", err)
		os.Exit(1)
	}
}
