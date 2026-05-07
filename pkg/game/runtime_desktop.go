//go:build !js

package game

import (
	"fmt"

	"github.com/opd-ai/murmur/pkg/app"
)

type desktopRuntime struct {
	cfg RuntimeConfig
}

func newDesktopRuntime(cfg RuntimeConfig) Runtime {
	return &desktopRuntime{cfg: cfg}
}

func (r *desktopRuntime) Run() error {
	application, err := app.New(app.Config{Version: r.cfg.Version})
	if err != nil {
		return fmt.Errorf("creating desktop app runtime: %w", err)
	}
	defer application.Close()

	return application.Run()
}
