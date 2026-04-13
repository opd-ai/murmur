// Package interaction provides pan, zoom, node selection, and navigation.
// Per PULSE_MAP.md, users navigate the Pulse Map spatially.
package interaction

import (
	"math"
)

// Camera represents the viewport into the Pulse Map.
type Camera struct {
	X, Y        float64 // Center position in world coordinates
	Scale       float64 // Zoom level (1.0 = default)
	TargetX     float64 // Animation target X
	TargetY     float64 // Animation target Y
	TargetScale float64 // Animation target scale
	Animating   bool
}

// NewCamera creates a camera centered at the origin with default zoom.
func NewCamera() *Camera {
	return &Camera{
		X:           0,
		Y:           0,
		Scale:       1.0,
		TargetX:     0,
		TargetY:     0,
		TargetScale: 1.0,
	}
}

// MinScale is the minimum zoom level (wide view).
const MinScale = 0.1

// MaxScale is the maximum zoom level (close view).
const MaxScale = 5.0

// Pan moves the camera by the given screen-space delta.
func (c *Camera) Pan(dx, dy float64) {
	// Convert screen delta to world delta based on current scale
	c.X -= dx / c.Scale
	c.Y -= dy / c.Scale
	c.TargetX = c.X
	c.TargetY = c.Y
	c.Animating = false
}

// Zoom adjusts the zoom level, keeping the given screen point fixed.
func (c *Camera) Zoom(factor, screenX, screenY, screenWidth, screenHeight float64) {
	// Calculate world position under cursor before zoom
	worldX, worldY := c.ScreenToWorld(screenX, screenY, screenWidth, screenHeight)

	// Apply zoom
	newScale := c.Scale * factor
	if newScale < MinScale {
		newScale = MinScale
	}
	if newScale > MaxScale {
		newScale = MaxScale
	}
	c.Scale = newScale
	c.TargetScale = newScale

	// Calculate where that world point is now on screen
	newScreenX, newScreenY := c.WorldToScreen(worldX, worldY, screenWidth, screenHeight)

	// Adjust camera to keep the point under cursor
	c.X += (screenX - newScreenX) / c.Scale
	c.Y += (screenY - newScreenY) / c.Scale
	c.TargetX = c.X
	c.TargetY = c.Y
}

// AnimateTo starts an animation to the given world position.
func (c *Camera) AnimateTo(worldX, worldY float64) {
	c.TargetX = worldX
	c.TargetY = worldY
	c.Animating = true
}

// AnimateToWithZoom starts an animation with zoom change.
func (c *Camera) AnimateToWithZoom(worldX, worldY, scale float64) {
	c.TargetX = worldX
	c.TargetY = worldY
	c.TargetScale = math.Max(MinScale, math.Min(MaxScale, scale))
	c.Animating = true
}

// Update performs animation interpolation per tick.
func (c *Camera) Update() {
	if !c.Animating {
		return
	}

	const lerp = 0.1      // Interpolation factor for smooth animation
	const threshold = 0.5 // Stop animating when close enough

	dx := c.TargetX - c.X
	dy := c.TargetY - c.Y
	ds := c.TargetScale - c.Scale

	if math.Abs(dx) < threshold && math.Abs(dy) < threshold && math.Abs(ds) < 0.01 {
		c.X = c.TargetX
		c.Y = c.TargetY
		c.Scale = c.TargetScale
		c.Animating = false
		return
	}

	c.X += dx * lerp
	c.Y += dy * lerp
	c.Scale += ds * lerp
}

// ScreenToWorld converts screen coordinates to world coordinates.
func (c *Camera) ScreenToWorld(screenX, screenY, screenWidth, screenHeight float64) (float64, float64) {
	// Screen center offset
	centerX := screenWidth / 2
	centerY := screenHeight / 2

	// Convert to world coordinates
	worldX := c.X + (screenX-centerX)/c.Scale
	worldY := c.Y + (screenY-centerY)/c.Scale

	return worldX, worldY
}

// WorldToScreen converts world coordinates to screen coordinates.
func (c *Camera) WorldToScreen(worldX, worldY, screenWidth, screenHeight float64) (float64, float64) {
	centerX := screenWidth / 2
	centerY := screenHeight / 2

	screenX := centerX + (worldX-c.X)*c.Scale
	screenY := centerY + (worldY-c.Y)*c.Scale

	return screenX, screenY
}

// ViewBounds returns the world-space bounding box visible on screen.
func (c *Camera) ViewBounds(screenWidth, screenHeight float64) (minX, minY, maxX, maxY float64) {
	halfW := screenWidth / 2 / c.Scale
	halfH := screenHeight / 2 / c.Scale
	return c.X - halfW, c.Y - halfH, c.X + halfW, c.Y + halfH
}

// InputState tracks input for interaction handling.
type InputState struct {
	Dragging       bool
	DragStartX     float64
	DragStartY     float64
	LastX          float64
	LastY          float64
	SelectedNodeID string
}

// NewInputState creates a new input state tracker.
func NewInputState() *InputState {
	return &InputState{}
}

// StartDrag begins a drag operation.
func (s *InputState) StartDrag(x, y float64) {
	s.Dragging = true
	s.DragStartX = x
	s.DragStartY = y
	s.LastX = x
	s.LastY = y
}

// UpdateDrag updates the drag position and returns delta.
func (s *InputState) UpdateDrag(x, y float64) (dx, dy float64) {
	if !s.Dragging {
		return 0, 0
	}
	dx = x - s.LastX
	dy = y - s.LastY
	s.LastX = x
	s.LastY = y
	return dx, dy
}

// EndDrag ends the drag operation.
func (s *InputState) EndDrag() {
	s.Dragging = false
}

// SelectNode sets the selected node ID.
func (s *InputState) SelectNode(id string) {
	s.SelectedNodeID = id
}

// ClearSelection clears the node selection.
func (s *InputState) ClearSelection() {
	s.SelectedNodeID = ""
}

// HitTest checks if a point is within a node's radius.
func HitTest(nodeX, nodeY, pointX, pointY, radius float64) bool {
	dx := nodeX - pointX
	dy := nodeY - pointY
	return dx*dx+dy*dy <= radius*radius
}
