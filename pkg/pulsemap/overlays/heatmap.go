// Package overlays - Activity Heat Map overlay for recent Wave activity visualization.
// Per PULSE_MAP.md §Activity Heat Map, shows blue-to-red gradient based on 60-minute activity density.
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ActivitySample records Wave activity at a specific location and time.
type ActivitySample struct {
	X, Y      float64   // World coordinates
	Timestamp time.Time // When this activity occurred
	Intensity float32   // Activity intensity (0-1), based on PoW or other metrics
}

// HeatMapConfig holds heat map display configuration.
type HeatMapConfig struct {
	Enabled        bool          // Whether heat map is visible
	WindowDuration time.Duration // Trailing window (default 60 minutes)
	GridCellSize   float32       // Size of heat map grid cells in world coordinates
	BlurRadius     int           // Gaussian blur radius in pixels
	MinIntensity   float32       // Minimum intensity to display (0-1)
	MaxIntensity   float32       // Maximum intensity for color scaling (0-1)
}

// ActivityHeatMap renders a blurred heat map overlay of recent Wave activity.
// Per PULSE_MAP.md: "blue-to-red gradient, 60-minute trailing window, blurred background layer".
type ActivityHeatMap struct {
	config  HeatMapConfig
	samples []ActivitySample

	// Grid cells for aggregating activity
	grid map[gridKey]float32

	// Cached heat map image (regenerated when samples change)
	heatMapImage *ebiten.Image
	needsRedraw  bool

	// Last update time for expiry checking
	lastUpdate time.Time
}

// gridKey identifies a grid cell in world space.
type gridKey struct {
	x, y int
}

// NewActivityHeatMap creates a new heat map overlay with default configuration.
func NewActivityHeatMap() *ActivityHeatMap {
	return &ActivityHeatMap{
		config: HeatMapConfig{
			Enabled:        false, // Off by default per PULSE_MAP.md
			WindowDuration: 60 * time.Minute,
			GridCellSize:   100, // 100 world units per cell
			BlurRadius:     15,
			MinIntensity:   0.1,
			MaxIntensity:   1.0,
		},
		samples:     make([]ActivitySample, 0, 1000),
		grid:        make(map[gridKey]float32, 256),
		needsRedraw: true,
		lastUpdate:  time.Now(),
	}
}

// NewActivityHeatMapWithConfig creates a heat map with custom configuration.
func NewActivityHeatMapWithConfig(config HeatMapConfig) *ActivityHeatMap {
	return &ActivityHeatMap{
		config:      config,
		samples:     make([]ActivitySample, 0, 1000),
		grid:        make(map[gridKey]float32, 256),
		needsRedraw: true,
		lastUpdate:  time.Now(),
	}
}

// RecordActivity adds a Wave activity sample at the given location.
func (h *ActivityHeatMap) RecordActivity(x, y float64, intensity float32) {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}

	sample := ActivitySample{
		X:         x,
		Y:         y,
		Timestamp: time.Now(),
		Intensity: intensity,
	}

	h.samples = append(h.samples, sample)
	h.needsRedraw = true
}

// Update prunes expired samples and triggers redraw if needed.
func (h *ActivityHeatMap) Update() {
	now := time.Now()

	// Prune expired samples
	cutoff := now.Add(-h.config.WindowDuration)
	validSamples := h.samples[:0]

	for i := range h.samples {
		if h.samples[i].Timestamp.After(cutoff) {
			validSamples = append(validSamples, h.samples[i])
		}
	}

	if len(validSamples) < len(h.samples) {
		h.samples = validSamples
		h.needsRedraw = true
	}

	h.lastUpdate = now
}

// Render draws the heat map overlay to the screen.
func (h *ActivityHeatMap) Render(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	if !h.config.Enabled || len(h.samples) == 0 {
		return
	}

	if h.needsRedraw {
		h.regenerateHeatMap(screen, cameraX, cameraY, zoom)
		h.needsRedraw = false
	}

	if h.heatMapImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(0.6) // 60% opacity for subtle effect
		screen.DrawImage(h.heatMapImage, op)
	}
}

// regenerateHeatMap rebuilds the heat map image from current samples.
func (h *ActivityHeatMap) regenerateHeatMap(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()

	h.rebuildGrid()
	h.createOrRecreateImage(screenW, screenH)
	h.renderHeatMapCells(screenW, screenH, cameraX, cameraY, zoom)
}

// rebuildGrid clears and regenerates the grid from samples with decay.
func (h *ActivityHeatMap) rebuildGrid() {
	for k := range h.grid {
		delete(h.grid, k)
	}

	for _, sample := range h.samples {
		cellX := int(math.Floor(sample.X / float64(h.config.GridCellSize)))
		cellY := int(math.Floor(sample.Y / float64(h.config.GridCellSize)))
		key := gridKey{cellX, cellY}

		age := time.Since(sample.Timestamp)
		decay := 1.0 - float32(age.Seconds())/float32(h.config.WindowDuration.Seconds())
		if decay < 0 {
			decay = 0
		}

		h.grid[key] += sample.Intensity * decay
	}
}

// createOrRecreateImage creates or resizes the heat map image if needed.
func (h *ActivityHeatMap) createOrRecreateImage(screenW, screenH int) {
	if h.heatMapImage == nil || h.heatMapImage.Bounds().Dx() != screenW || h.heatMapImage.Bounds().Dy() != screenH {
		if h.heatMapImage != nil {
			h.heatMapImage.Deallocate()
		}
		h.heatMapImage = ebiten.NewImage(screenW, screenH)
	}
	h.heatMapImage.Clear()
}

// renderHeatMapCells draws all grid cells to the heat map image.
func (h *ActivityHeatMap) renderHeatMapCells(screenW, screenH int, cameraX, cameraY, zoom float64) {
	centerX := float64(screenW) / 2
	centerY := float64(screenH) / 2

	for key, intensity := range h.grid {
		if intensity < h.config.MinIntensity {
			continue
		}

		screenX, screenY, cellRadius := h.transformCellToScreen(key, centerX, centerY, cameraX, cameraY, zoom)

		if h.isCellOffScreen(screenX, screenY, cellRadius, screenW, screenH) {
			continue
		}

		h.drawHeatMapCell(screenX, screenY, cellRadius, intensity)
	}
}

// transformCellToScreen converts grid cell to screen coordinates.
func (h *ActivityHeatMap) transformCellToScreen(key gridKey, centerX, centerY, cameraX, cameraY, zoom float64) (float64, float64, float64) {
	worldCellX := float64(key.x)*float64(h.config.GridCellSize) + float64(h.config.GridCellSize)/2
	worldCellY := float64(key.y)*float64(h.config.GridCellSize) + float64(h.config.GridCellSize)/2
	screenX := centerX + (worldCellX-cameraX)*zoom
	screenY := centerY + (worldCellY-cameraY)*zoom
	cellRadius := float64(h.config.GridCellSize) * zoom / 2
	return screenX, screenY, cellRadius
}

// isCellOffScreen checks if a cell is outside the visible screen area.
func (h *ActivityHeatMap) isCellOffScreen(screenX, screenY, cellRadius float64, screenW, screenH int) bool {
	return screenX+cellRadius < 0 || screenX-cellRadius > float64(screenW) ||
		screenY+cellRadius < 0 || screenY-cellRadius > float64(screenH)
}

// drawHeatMapCell draws a single heat map cell at the given position.
func (h *ActivityHeatMap) drawHeatMapCell(screenX, screenY, cellRadius float64, intensity float32) {
	normalizedIntensity := (intensity - h.config.MinIntensity) / (h.config.MaxIntensity - h.config.MinIntensity)
	if normalizedIntensity > 1 {
		normalizedIntensity = 1
	}

	col := heatMapColor(normalizedIntensity)
	radius := float32(cellRadius)
	if radius < 5 {
		radius = 5
	}

	vector.DrawFilledCircle(h.heatMapImage, float32(screenX), float32(screenY), radius, col, true)
}

// heatMapColor maps intensity (0-1) to blue-to-red gradient.
// Per PULSE_MAP.md: "blue for low activity, red for high activity".
func heatMapColor(intensity float32) color.RGBA {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}

	// Blue (low) -> Cyan -> Green -> Yellow -> Red (high)
	// This is a multi-stop gradient for better visualization

	if intensity < 0.25 {
		// Blue to Cyan (0.0 - 0.25)
		t := intensity / 0.25
		return color.RGBA{
			R: 0,
			G: uint8(t * 100),
			B: uint8(100 + t*155),
			A: 180,
		}
	} else if intensity < 0.5 {
		// Cyan to Green (0.25 - 0.5)
		t := (intensity - 0.25) / 0.25
		return color.RGBA{
			R: 0,
			G: uint8(100 + t*155),
			B: uint8(255 - t*155),
			A: 180,
		}
	} else if intensity < 0.75 {
		// Green to Yellow (0.5 - 0.75)
		t := (intensity - 0.5) / 0.25
		return color.RGBA{
			R: uint8(t * 255),
			G: 255,
			B: 0,
			A: 180,
		}
	} else {
		// Yellow to Red (0.75 - 1.0)
		t := (intensity - 0.75) / 0.25
		return color.RGBA{
			R: 255,
			G: uint8(255 - t*155),
			B: 0,
			A: 180,
		}
	}
}

// SetEnabled toggles heat map visibility.
func (h *ActivityHeatMap) SetEnabled(enabled bool) {
	if h.config.Enabled != enabled {
		h.config.Enabled = enabled
		h.needsRedraw = true
	}
}

// IsEnabled returns whether the heat map is currently visible.
func (h *ActivityHeatMap) IsEnabled() bool {
	return h.config.Enabled
}

// SetWindowDuration changes the trailing activity window.
func (h *ActivityHeatMap) SetWindowDuration(duration time.Duration) {
	h.config.WindowDuration = duration
	h.needsRedraw = true
}

// Clear removes all activity samples and resets the heat map.
func (h *ActivityHeatMap) Clear() {
	h.samples = h.samples[:0]
	for k := range h.grid {
		delete(h.grid, k)
	}
	if h.heatMapImage != nil {
		h.heatMapImage.Clear()
	}
	h.needsRedraw = true
}

// SampleCount returns the number of active samples in the trailing window.
func (h *ActivityHeatMap) SampleCount() int {
	return len(h.samples)
}

// Dispose releases GPU resources.
func (h *ActivityHeatMap) Dispose() {
	if h.heatMapImage != nil {
		h.heatMapImage.Deallocate()
		h.heatMapImage = nil
	}
}
