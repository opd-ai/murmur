// UI initialization stub for headless builds.
//
//go:build noebiten
// +build noebiten

package app

import "fmt"

// runUI is a stub for headless builds.
func (a *App) runUI() error {
	fmt.Println("UI not available in headless build (noebiten tag).")
	<-a.ctx.Done()
	return nil
}
