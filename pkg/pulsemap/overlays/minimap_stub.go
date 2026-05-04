// Package overlays - Minimap stub for test builds.
//

//go:build test
// +build test

package overlays

import (
	"image/color"
)

// MinimapConfig holds minimap display configuration.
type MinimapConfig struct {
	Width         float32
	Height        float32
	Margin        float32
	Position      MinimapPos
	BgColor       color.RGBA
	NodeColor     color.RGBA
	ViewportColor color.RGBA
	BorderColor   color.RGBA
}

// MinimapPos specifies which corner to place the minimap.
type MinimapPos int

const (
	MinimapTopRight MinimapPos = iota
	MinimapTopLeft
	MinimapBottomRight
	MinimapBottomLeft
)

// MinimapNode represents a node in the minimap.
type MinimapNode struct {
	X, Y float64
}

// Minimap stub for test builds.
type Minimap struct {
	config MinimapConfig
	nodes  []MinimapNode
}

// NewMinimap creates a stub minimap.
func NewMinimap() *Minimap {
	return &Minimap{
		nodes: make([]MinimapNode, 0),
	}
}

// NewMinimapWithConfig creates a stub minimap with custom configuration.
func NewMinimapWithConfig(config MinimapConfig) *Minimap {
	return &Minimap{
		config: config,
		nodes:  make([]MinimapNode, 0),
	}
}

// UpdateNodes is a no-op in test builds.
func (m *Minimap) UpdateNodes(nodes []MinimapNode) {
	m.nodes = nodes
}

// Render is a no-op in test builds.
func (m *Minimap) Render(screen interface{}, cameraX, cameraY, zoom float64, viewportWidth, viewportHeight int) {
}

// SetPosition is a no-op in test builds.
func (m *Minimap) SetPosition(pos MinimapPos) {
}

// SetSize is a no-op in test builds.
func (m *Minimap) SetSize(width, height float32) {
}

// IsVisible always returns false in test builds.
func (m *Minimap) IsVisible() bool {
	return false
}

// DistanceToEdge returns infinity in test builds.
func (m *Minimap) DistanceToEdge(screenX, screenY float32, screenWidth, screenHeight int) float32 {
	return 1000000
}

// ContainsPoint always returns false in test builds.
func (m *Minimap) ContainsPoint(screenX, screenY float32, screenWidth, screenHeight int) bool {
	return false
}
