// Package overlays - Common camera transformation helpers.
//

package overlays

import "github.com/hajimehoshi/ebiten/v2"

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
