// Package overlays - Minimap provides a full network overview with viewport indicator.
// Per ROADMAP.md line 635, the minimap shows a birds-eye view in the corner.
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MinimapConfig holds minimap display configuration.
type MinimapConfig struct {
	Width         float32    // Minimap width in pixels
	Height        float32    // Minimap height in pixels
	Margin        float32    // Distance from screen edges
	Position      MinimapPos // Corner position
	BgColor       color.RGBA // Background color
	NodeColor     color.RGBA // Node dot color
	ViewportColor color.RGBA // Viewport indicator color
	BorderColor   color.RGBA // Border color
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
	X, Y float64 // World coordinates
}

// Minimap renders a full network overview in the corner.
// Per PULSE_MAP.md, provides spatial context when zoomed in.
type Minimap struct {
	config MinimapConfig
	nodes  []MinimapNode

	// World bounds (min/max coordinates of all nodes)
	worldMinX, worldMaxX float64
	worldMinY, worldMaxY float64

	// Cache for rendering
	minimapImage *ebiten.Image
	needsRedraw  bool
}

// NewMinimap creates a new minimap overlay with default configuration.
func NewMinimap() *Minimap {
	return &Minimap{
		config: MinimapConfig{
			Width:         150,
			Height:        150,
			Margin:        10,
			Position:      MinimapBottomRight,
			BgColor:       color.RGBA{10, 15, 25, 200},
			NodeColor:     color.RGBA{100, 150, 255, 255},
			ViewportColor: color.RGBA{255, 255, 100, 150},
			BorderColor:   color.RGBA{50, 80, 120, 255},
		},
		nodes:       make([]MinimapNode, 0, 500),
		needsRedraw: true,
	}
}

// NewMinimapWithConfig creates a minimap with custom configuration.
func NewMinimapWithConfig(config MinimapConfig) *Minimap {
	return &Minimap{
		config:      config,
		nodes:       make([]MinimapNode, 0, 500),
		needsRedraw: true,
	}
}

// UpdateNodes refreshes the minimap with current node positions.
func (m *Minimap) UpdateNodes(nodes []MinimapNode) {
	m.nodes = nodes
	m.calculateWorldBounds()
	m.needsRedraw = true
}

// calculateWorldBounds computes the bounding box of all nodes.
func (m *Minimap) calculateWorldBounds() {
	if len(m.nodes) == 0 {
		m.worldMinX, m.worldMaxX = -500, 500
		m.worldMinY, m.worldMaxY = -500, 500
		return
	}

	m.worldMinX = m.nodes[0].X
	m.worldMaxX = m.nodes[0].X
	m.worldMinY = m.nodes[0].Y
	m.worldMaxY = m.nodes[0].Y

	for _, node := range m.nodes[1:] {
		if node.X < m.worldMinX {
			m.worldMinX = node.X
		}
		if node.X > m.worldMaxX {
			m.worldMaxX = node.X
		}
		if node.Y < m.worldMinY {
			m.worldMinY = node.Y
		}
		if node.Y > m.worldMaxY {
			m.worldMaxY = node.Y
		}
	}

	// Add padding (10% of range)
	rangeX := m.worldMaxX - m.worldMinX
	rangeY := m.worldMaxY - m.worldMinY
	if rangeX < 100 {
		rangeX = 100
	}
	if rangeY < 100 {
		rangeY = 100
	}

	m.worldMinX -= rangeX * 0.1
	m.worldMaxX += rangeX * 0.1
	m.worldMinY -= rangeY * 0.1
	m.worldMaxY += rangeY * 0.1
}

// worldToMinimap transforms world coordinates to minimap pixel coordinates.
func (m *Minimap) worldToMinimap(worldX, worldY float64) (float32, float32) {
	rangeX := m.worldMaxX - m.worldMinX
	rangeY := m.worldMaxY - m.worldMinY

	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	// Map to 0-1 range
	normalX := (worldX - m.worldMinX) / rangeX
	normalY := (worldY - m.worldMinY) / rangeY

	// Scale to minimap size (with small padding)
	padding := float32(5)
	x := padding + float32(normalX)*float32(m.config.Width-2*padding)
	y := padding + float32(normalY)*float32(m.config.Height-2*padding)

	return x, y
}

// getMinimapPosition returns the screen coordinates for the minimap corner.
func (m *Minimap) getMinimapPosition(screenWidth, screenHeight int) (float32, float32) {
	sw := float32(screenWidth)
	sh := float32(screenHeight)
	margin := m.config.Margin

	switch m.config.Position {
	case MinimapTopRight:
		return sw - m.config.Width - margin, margin
	case MinimapTopLeft:
		return margin, margin
	case MinimapBottomRight:
		return sw - m.config.Width - margin, sh - m.config.Height - margin
	case MinimapBottomLeft:
		return margin, sh - m.config.Height - margin
	default:
		return sw - m.config.Width - margin, sh - m.config.Height - margin
	}
}

// Render draws the minimap to the screen.
func (m *Minimap) Render(screen *ebiten.Image, cameraX, cameraY, zoom float64, viewportWidth, viewportHeight int) {
	if len(m.nodes) == 0 {
		return
	}

	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()

	// Get minimap corner position
	minimapX, minimapY := m.getMinimapPosition(screenW, screenH)

	// Draw background and border
	vector.DrawFilledRect(
		screen,
		minimapX,
		minimapY,
		m.config.Width,
		m.config.Height,
		m.config.BgColor,
		false,
	)

	vector.StrokeRect(
		screen,
		minimapX,
		minimapY,
		m.config.Width,
		m.config.Height,
		1.5,
		m.config.BorderColor,
		false,
	)

	// Draw all nodes as small dots
	for _, node := range m.nodes {
		localX, localY := m.worldToMinimap(node.X, node.Y)
		vector.DrawFilledCircle(
			screen,
			minimapX+localX,
			minimapY+localY,
			1.5,
			m.config.NodeColor,
			true,
		)
	}

	// Draw viewport indicator
	m.renderViewportIndicator(screen, minimapX, minimapY, cameraX, cameraY, zoom, viewportWidth, viewportHeight)
}

// renderViewportIndicator draws the current viewport rectangle on the minimap.
func (m *Minimap) renderViewportIndicator(screen *ebiten.Image, minimapX, minimapY float32, cameraX, cameraY, zoom float64, viewportWidth, viewportHeight int) {
	// Calculate world coordinates of viewport corners
	halfW := float64(viewportWidth) / 2.0
	halfH := float64(viewportHeight) / 2.0

	viewMinX := cameraX - halfW/zoom
	viewMaxX := cameraX + halfW/zoom
	viewMinY := cameraY - halfH/zoom
	viewMaxY := cameraY + halfH/zoom

	// Convert to minimap coordinates
	x1, y1 := m.worldToMinimap(viewMinX, viewMinY)
	x2, y2 := m.worldToMinimap(viewMaxX, viewMaxY)

	// Clamp to minimap bounds
	x1 = clampFloat32(x1, 0, m.config.Width)
	y1 = clampFloat32(y1, 0, m.config.Height)
	x2 = clampFloat32(x2, 0, m.config.Width)
	y2 = clampFloat32(y2, 0, m.config.Height)

	// Draw viewport rectangle
	w := x2 - x1
	h := y2 - y1

	if w > 0 && h > 0 {
		vector.StrokeRect(
			screen,
			minimapX+x1,
			minimapY+y1,
			w,
			h,
			2.0,
			m.config.ViewportColor,
			false,
		)
	}
}

// clampFloat32 restricts value to [min, max] range.
func clampFloat32(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// SetPosition changes the corner position of the minimap.
func (m *Minimap) SetPosition(pos MinimapPos) {
	m.config.Position = pos
}

// SetSize changes the dimensions of the minimap.
func (m *Minimap) SetSize(width, height float32) {
	m.config.Width = width
	m.config.Height = height
	m.needsRedraw = true
}

// IsVisible returns whether the minimap should be rendered.
// Can be extended to support user toggle preference.
func (m *Minimap) IsVisible() bool {
	return len(m.nodes) > 0
}

// DistanceToEdge returns distance from a screen point to the minimap edge.
// Useful for detecting clicks/taps on the minimap.
func (m *Minimap) DistanceToEdge(screenX, screenY float32, screenWidth, screenHeight int) float32 {
	minimapX, minimapY := m.getMinimapPosition(screenWidth, screenHeight)

	// Check if inside minimap bounds
	if screenX >= minimapX && screenX <= minimapX+m.config.Width &&
		screenY >= minimapY && screenY <= minimapY+m.config.Height {
		return 0 // Inside
	}

	// Calculate distance to nearest edge
	dx := float32(0)
	dy := float32(0)

	if screenX < minimapX {
		dx = minimapX - screenX
	} else if screenX > minimapX+m.config.Width {
		dx = screenX - (minimapX + m.config.Width)
	}

	if screenY < minimapY {
		dy = minimapY - screenY
	} else if screenY > minimapY+m.config.Height {
		dy = screenY - (minimapY + m.config.Height)
	}

	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// ContainsPoint checks if a screen coordinate is inside the minimap.
func (m *Minimap) ContainsPoint(screenX, screenY float32, screenWidth, screenHeight int) bool {
	minimapX, minimapY := m.getMinimapPosition(screenWidth, screenHeight)
	return screenX >= minimapX && screenX <= minimapX+m.config.Width &&
		screenY >= minimapY && screenY <= minimapY+m.config.Height
}
