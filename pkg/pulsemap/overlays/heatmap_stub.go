// Package overlays - Activity Heat Map stub for test builds.
//

//go:build test
// +build test

package overlays

import (
	"time"
)

// ActivitySample records Wave activity at a specific location and time.
type ActivitySample struct {
	X, Y      float64
	Timestamp time.Time
	Intensity float32
}

// HeatMapConfig holds heat map display configuration.
type HeatMapConfig struct {
	Enabled        bool
	WindowDuration time.Duration
	GridCellSize   float32
	BlurRadius     int
	MinIntensity   float32
	MaxIntensity   float32
}

// ActivityHeatMap stub for test builds.
type ActivityHeatMap struct {
	config  HeatMapConfig
	samples []ActivitySample
}

// NewActivityHeatMap creates a stub heat map.
func NewActivityHeatMap() *ActivityHeatMap {
	return &ActivityHeatMap{
		config: HeatMapConfig{
			Enabled:        false,
			WindowDuration: 60 * time.Minute,
			GridCellSize:   100,
			BlurRadius:     15,
			MinIntensity:   0.1,
			MaxIntensity:   1.0,
		},
		samples: make([]ActivitySample, 0),
	}
}

// NewActivityHeatMapWithConfig creates a stub heat map with custom config.
func NewActivityHeatMapWithConfig(config HeatMapConfig) *ActivityHeatMap {
	return &ActivityHeatMap{
		config:  config,
		samples: make([]ActivitySample, 0),
	}
}

// RecordActivity is a no-op in test builds.
func (h *ActivityHeatMap) RecordActivity(x, y float64, intensity float32) {
	sample := ActivitySample{
		X:         x,
		Y:         y,
		Timestamp: time.Now(),
		Intensity: intensity,
	}
	h.samples = append(h.samples, sample)
}

// Update is a no-op in test builds.
func (h *ActivityHeatMap) Update() {
}

// Render is a no-op in test builds.
func (h *ActivityHeatMap) Render(screen interface{}, cameraX, cameraY, zoom float64) {
}

// SetEnabled is a no-op in test builds.
func (h *ActivityHeatMap) SetEnabled(enabled bool) {
	h.config.Enabled = enabled
}

// IsEnabled returns configuration state in test builds.
func (h *ActivityHeatMap) IsEnabled() bool {
	return h.config.Enabled
}

// SetWindowDuration is a no-op in test builds.
func (h *ActivityHeatMap) SetWindowDuration(duration time.Duration) {
	h.config.WindowDuration = duration
}

// Clear resets samples in test builds.
func (h *ActivityHeatMap) Clear() {
	h.samples = h.samples[:0]
}

// SampleCount returns sample count in test builds.
func (h *ActivityHeatMap) SampleCount() int {
	return len(h.samples)
}

// Dispose is a no-op in test builds.
func (h *ActivityHeatMap) Dispose() {
}
