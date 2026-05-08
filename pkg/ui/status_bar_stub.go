// Package ui provides a stub StatusBar for test builds.
//
//go:build test
// +build test

package ui

// StatusBar is a stub for the network/crypto status indicator strip.
// Per PLAN.md: "Status indicators — identity publication, Shroud circuit,
// connection count, PoW progress".
type StatusBar struct {
	x, y   int
	width  int
	height int
	theme  Theme

	PeerCount         int
	ShroudActive      bool
	IdentityPublished bool
	PowBusy           bool
	PowProgress       float32
}

// NewStatusBar creates a stub StatusBar.
func NewStatusBar(x, y, width, height int, theme Theme) *StatusBar {
	return &StatusBar{x: x, y: y, width: width, height: height, theme: theme}
}

// Draw is a no-op in test mode.
func (s *StatusBar) Draw(_ interface{}) {}
