// UI initialization stub for headless builds.
//
//go:build test || noebiten
// +build test noebiten

package app

import "fmt"

// runUI is a stub for headless builds.
func (a *App) runUI() error {
	fmt.Println("UI not available in headless build (test tag).")
	<-a.ctx.Done()
	return nil
}
