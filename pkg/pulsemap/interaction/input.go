// Package interaction provides pan, zoom, node selection, and navigation.
// Per PULSE_MAP.md, users navigate the Pulse Map spatially.
package interaction

import (
	"math"
	"time"
)

// Camera represents the viewport into the Pulse Map.
type Camera struct {
	X, Y        float64 // Center position in world coordinates
	Scale       float64 // Zoom level (1.0 = default)
	TargetX     float64 // Animation target X
	TargetY     float64 // Animation target Y
	TargetScale float64 // Animation target scale
	Animating   bool

	// Momentum scrolling state
	velocityX float64 // Current pan velocity in world units per tick
	velocityY float64 // Current pan velocity in world units per tick

	lastUpdate time.Time // Time of the previous Update() call for dt integration
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
		lastUpdate:  time.Now(),
	}
}

// MinScale is the minimum zoom level (wide view).
const MinScale = 0.1

// MaxScale is the maximum zoom level (close view).
const MaxScale = 5.0

// ZoomLevel represents the level of detail based on zoom scale.
type ZoomLevel int

const (
	// ZoomLevelMacro shows full network with colored dots (scale < 0.5).
	ZoomLevelMacro ZoomLevel = iota
	// ZoomLevelMeso shows 50-200 node neighborhood with moderate detail (0.5 <= scale < 2.0).
	ZoomLevelMeso
	// ZoomLevelMicro shows 5-20 nodes at full detail with labels (scale >= 2.0).
	ZoomLevelMicro
)

// ZoomLevel returns the current level of detail based on scale.
func (c *Camera) ZoomLevel() ZoomLevel {
	if c.Scale < 0.5 {
		return ZoomLevelMacro
	}
	if c.Scale < 2.0 {
		return ZoomLevelMeso
	}
	return ZoomLevelMicro
}

// Deprecated: Use ZoomLevel instead.
func (c *Camera) GetZoomLevel() ZoomLevel {
	return c.ZoomLevel()
}

// Pan moves the camera by the given screen-space delta.
func (c *Camera) Pan(dx, dy float64) {
	// Convert screen delta to world delta based on current scale
	c.X -= dx / c.Scale
	c.Y -= dy / c.Scale
	c.TargetX = c.X
	c.TargetY = c.Y
	c.Animating = false
	// Reset momentum when user is actively panning
	c.velocityX = 0
	c.velocityY = 0
}

// Zoom adjusts the zoom level smoothly, keeping the given screen point fixed.
func (c *Camera) Zoom(factor, screenX, screenY, screenWidth, screenHeight float64) {
	// Calculate world position under cursor
	worldX, worldY := c.ScreenToWorld(screenX, screenY, screenWidth, screenHeight)

	// Calculate new target scale
	newScale := c.Scale * factor
	if newScale < MinScale {
		newScale = MinScale
	}
	if newScale > MaxScale {
		newScale = MaxScale
	}

	// Set target scale for smooth animation
	c.TargetScale = newScale
	c.Animating = true

	// Adjust target position to keep the world point under the cursor
	// as we animate toward the target scale
	c.TargetX = worldX - (screenX-screenWidth/2)/newScale
	c.TargetY = worldY - (screenY-screenHeight/2)/newScale
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

// AnimateToScreenPointWithZoom animates camera centering using a screen-space point.
// It converts the point to world coordinates first, then delegates to AnimateToWithZoom.
func (c *Camera) AnimateToScreenPointWithZoom(screenX, screenY, screenWidth, screenHeight, scale float64) {
	worldX, worldY := c.ScreenToWorld(screenX, screenY, screenWidth, screenHeight)
	c.AnimateToWithZoom(worldX, worldY, scale)
}

// SetZoomPresetMacro animates to Macro view (full network, scale 0.3).
// Per ROADMAP.md line 682, viewport controls provide preset zoom levels.
func (c *Camera) SetZoomPresetMacro() {
	c.TargetScale = 0.3
	c.Animating = true
	c.velocityX = 0
	c.velocityY = 0
}

// SetZoomPresetMeso animates to Meso view (50-200 node neighborhood, scale 1.0).
func (c *Camera) SetZoomPresetMeso() {
	c.TargetScale = 1.0
	c.Animating = true
	c.velocityX = 0
	c.velocityY = 0
}

// SetZoomPresetMicro animates to Micro view (5-20 nodes at full detail, scale 3.0).
func (c *Camera) SetZoomPresetMicro() {
	c.TargetScale = 3.0
	c.Animating = true
	c.velocityX = 0
	c.velocityY = 0
}

// Update performs animation interpolation per tick.
func (c *Camera) Update() {
	now := time.Now()
	dt := now.Sub(c.lastUpdate).Seconds()
	if dt <= 0 || dt < 1.0/240.0 || dt > 0.25 {
		dt = 1.0 / 60.0
	}
	c.lastUpdate = now

	// Apply momentum scrolling (inertial pan with deceleration)
	const momentumDeceleration = 0.95 // Velocity multiplier per tick for smooth deceleration
	const momentumThreshold = 0.1     // Stop momentum when velocity is negligible
	decel := math.Pow(momentumDeceleration, dt*60.0)

	if !c.Animating && (math.Abs(c.velocityX) > momentumThreshold || math.Abs(c.velocityY) > momentumThreshold) {
		// Apply velocity to camera position
		c.X += c.velocityX * dt * 60.0
		c.Y += c.velocityY * dt * 60.0
		c.TargetX = c.X
		c.TargetY = c.Y

		// Apply deceleration
		c.velocityX *= decel
		c.velocityY *= decel

		// Stop momentum when velocity is negligible
		if math.Abs(c.velocityX) <= momentumThreshold {
			c.velocityX = 0
		}
		if math.Abs(c.velocityY) <= momentumThreshold {
			c.velocityY = 0
		}
	}

	// Handle animation interpolation (overrides momentum)
	if !c.Animating {
		return
	}

	const lerp = 0.1      // Base interpolation factor at 60fps
	const threshold = 0.5 // Stop animating when close enough
	alpha := 1.0 - math.Pow(1.0-lerp, dt*60.0)

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

	c.X += dx * alpha
	c.Y += dy * alpha
	c.Scale += ds * alpha

	// Clear momentum when animating
	c.velocityX = 0
	c.velocityY = 0
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

// CenterOn smoothly animates the camera to center on the given world position.
// Per ROADMAP.md line 680: ego-centric view with own node centered.
func (c *Camera) CenterOn(worldX, worldY float64) {
	c.TargetX = worldX
	c.TargetY = worldY
	c.Animating = true
	// Reset momentum when starting animation
	c.velocityX = 0
	c.velocityY = 0
}

// CenterOnWithZoom smoothly animates the camera to center on a position with a target zoom.
// Per ROADMAP.md line 672: double-tap/click for node centering zoom.
func (c *Camera) CenterOnWithZoom(worldX, worldY, targetScale float64) {
	c.TargetX = worldX
	c.TargetY = worldY
	c.TargetScale = clampScale(targetScale)
	c.Animating = true
	c.velocityX = 0
	c.velocityY = 0
}

// IsCentered returns true if the camera is approximately centered on the given position.
func (c *Camera) IsCentered(worldX, worldY, tolerance float64) bool {
	dx := math.Abs(c.X - worldX)
	dy := math.Abs(c.Y - worldY)
	return dx < tolerance && dy < tolerance
}

// ApplyMomentum starts momentum scrolling based on the last pan velocity.
// screenDx and screenDy are the last screen-space deltas from the pan gesture.
// This should be called when the user releases a pan gesture.
func (c *Camera) ApplyMomentum(screenDx, screenDy float64) {
	// Convert screen delta to world velocity
	c.velocityX = -screenDx / c.Scale
	c.velocityY = -screenDy / c.Scale

	// Scale momentum based on drag speed (cap at reasonable values)
	const maxMomentumVelocity = 50.0 // Maximum velocity in world units per tick
	if math.Abs(c.velocityX) > maxMomentumVelocity {
		c.velocityX = math.Copysign(maxMomentumVelocity, c.velocityX)
	}
	if math.Abs(c.velocityY) > maxMomentumVelocity {
		c.velocityY = math.Copysign(maxMomentumVelocity, c.velocityY)
	}

	// Don't start momentum if velocity is negligible
	const minMomentumVelocity = 0.5
	if math.Abs(c.velocityX) < minMomentumVelocity && math.Abs(c.velocityY) < minMomentumVelocity {
		c.velocityX = 0
		c.velocityY = 0
	}
}

// InputState tracks input for interaction handling.
type InputState struct {
	Dragging       bool
	DragStartX     float64
	DragStartY     float64
	LastX          float64
	LastY          float64
	LastDx         float64 // Last drag delta for momentum calculation
	LastDy         float64 // Last drag delta for momentum calculation
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
	s.LastDx = 0
	s.LastDy = 0
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
	s.LastDx = dx
	s.LastDy = dy
	return dx, dy
}

// EndDrag ends the drag operation and returns the last delta for momentum.
func (s *InputState) EndDrag() (lastDx, lastDy float64) {
	lastDx = s.LastDx
	lastDy = s.LastDy
	s.Dragging = false
	s.LastDx = 0
	s.LastDy = 0
	return lastDx, lastDy
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

// clampScale constrains the scale to valid zoom range [MinScale, MaxScale].
func clampScale(scale float64) float64 {
	if scale < MinScale {
		return MinScale
	}
	if scale > MaxScale {
		return MaxScale
	}
	return scale
}
