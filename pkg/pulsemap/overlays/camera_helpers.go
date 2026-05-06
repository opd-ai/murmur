// Package overlays - Common camera transformation helpers.
package overlays

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// getCameraSetup extracts screen dimensions and calculates center points.
func getCameraSetup(screen *ebiten.Image) (screenW, screenH, centerX, centerY float64) {
	screenW = float64(screen.Bounds().Dx())
	screenH = float64(screen.Bounds().Dy())
	centerX = screenW / 2
	centerY = screenH / 2
	return screenW, screenH, centerX, centerY
}

// worldToScreen transforms world coordinates to screen coordinates.
func worldToScreen(worldX, worldY, cameraX, cameraY, centerX, centerY, zoom float64) (screenX, screenY float64) {
	screenX = centerX + (worldX-cameraX)*zoom
	screenY = centerY + (worldY-cameraY)*zoom
	return screenX, screenY
}

// isOffScreen checks if a screen coordinate is outside the viewport with a margin.
func isOffScreen(x, y, screenW, screenH, margin float64) bool {
	return x < -margin || x > screenW+margin || y < -margin || y > screenH+margin
}

// drawSegmentedCircle renders a stroked circle using line segments.
// This consolidates circle drawing across masked_event.go and shadowplay.go.
// The number of segments adapts to the radius for smooth rendering.
func drawSegmentedCircle(screen *ebiten.Image, cx, cy, radius, strokeWidth float64, col color.Color) {
	// Check for transparent color early return.
	if c, ok := col.(color.RGBA); ok && c.A == 0 {
		return
	}

	numSegments := int(math.Max(32, radius/2))
	angleStep := 2 * math.Pi / float64(numSegments)

	for i := 0; i < numSegments; i++ {
		angle1 := float64(i) * angleStep
		angle2 := float64(i+1) * angleStep

		x1 := float32(cx + radius*math.Cos(angle1))
		y1 := float32(cy + radius*math.Sin(angle1))
		x2 := float32(cx + radius*math.Cos(angle2))
		y2 := float32(cy + radius*math.Sin(angle2))

		vector.StrokeLine(screen, x1, y1, x2, y2, float32(strokeWidth), col, false)
	}
}
