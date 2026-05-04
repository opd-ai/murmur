// Package overlays - Activity Heat Map tests.
//

//go:build !test
// +build !test

package overlays

import (
	"math"
	"testing"
	"time"
)

func TestNewActivityHeatMap(t *testing.T) {
	h := NewActivityHeatMap()

	if h == nil {
		t.Fatal("NewActivityHeatMap returned nil")
	}

	if h.config.Enabled {
		t.Error("expected heat map to be disabled by default")
	}

	if h.config.WindowDuration != 60*time.Minute {
		t.Errorf("expected 60 minute window, got %v", h.config.WindowDuration)
	}

	if h.config.GridCellSize != 100 {
		t.Errorf("expected grid cell size 100, got %f", h.config.GridCellSize)
	}

	if len(h.samples) != 0 {
		t.Errorf("expected 0 samples initially, got %d", len(h.samples))
	}
}

func TestNewActivityHeatMapWithConfig(t *testing.T) {
	config := HeatMapConfig{
		Enabled:        true,
		WindowDuration: 30 * time.Minute,
		GridCellSize:   50,
		BlurRadius:     10,
		MinIntensity:   0.2,
		MaxIntensity:   0.8,
	}

	h := NewActivityHeatMapWithConfig(config)

	if h == nil {
		t.Fatal("NewActivityHeatMapWithConfig returned nil")
	}

	if !h.config.Enabled {
		t.Error("expected heat map to be enabled")
	}

	if h.config.WindowDuration != 30*time.Minute {
		t.Errorf("expected 30 minute window, got %v", h.config.WindowDuration)
	}

	if h.config.GridCellSize != 50 {
		t.Errorf("expected grid cell size 50, got %f", h.config.GridCellSize)
	}

	if h.config.MinIntensity != 0.2 {
		t.Errorf("expected min intensity 0.2, got %f", h.config.MinIntensity)
	}
}

func TestRecordActivity(t *testing.T) {
	h := NewActivityHeatMap()

	h.RecordActivity(100, 200, 0.5)

	if len(h.samples) != 1 {
		t.Errorf("expected 1 sample, got %d", len(h.samples))
	}

	sample := h.samples[0]
	if sample.X != 100 || sample.Y != 200 {
		t.Errorf("expected position (100, 200), got (%f, %f)", sample.X, sample.Y)
	}

	if sample.Intensity != 0.5 {
		t.Errorf("expected intensity 0.5, got %f", sample.Intensity)
	}

	if !h.needsRedraw {
		t.Error("expected needsRedraw to be set after recording activity")
	}
}

func TestRecordActivityIntensityClamp(t *testing.T) {
	h := NewActivityHeatMap()

	// Test intensity clamping
	h.RecordActivity(0, 0, -0.5) // Below 0
	if h.samples[0].Intensity != 0 {
		t.Errorf("expected clamped intensity 0, got %f", h.samples[0].Intensity)
	}

	h.RecordActivity(0, 0, 1.5) // Above 1
	if h.samples[1].Intensity != 1 {
		t.Errorf("expected clamped intensity 1, got %f", h.samples[1].Intensity)
	}
}

func TestUpdate(t *testing.T) {
	h := NewActivityHeatMap()
	h.config.WindowDuration = 100 * time.Millisecond

	// Add samples at different times
	h.RecordActivity(0, 0, 0.5)
	time.Sleep(50 * time.Millisecond)
	h.RecordActivity(10, 10, 0.5)

	if len(h.samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(h.samples))
	}

	// Wait for first sample to expire
	time.Sleep(60 * time.Millisecond)
	h.Update()

	// First sample should be pruned, second should remain
	if len(h.samples) != 1 {
		t.Errorf("expected 1 sample after expiry, got %d", len(h.samples))
	}

	if h.samples[0].X != 10 || h.samples[0].Y != 10 {
		t.Errorf("wrong sample remained: (%f, %f)", h.samples[0].X, h.samples[0].Y)
	}
}

func TestSetEnabled(t *testing.T) {
	h := NewActivityHeatMap()

	if h.IsEnabled() {
		t.Error("expected heat map disabled by default")
	}

	h.SetEnabled(true)

	if !h.IsEnabled() {
		t.Error("expected heat map enabled after SetEnabled(true)")
	}

	if !h.needsRedraw {
		t.Error("expected needsRedraw to be set after enabling")
	}

	h.needsRedraw = false
	h.SetEnabled(false)

	if h.IsEnabled() {
		t.Error("expected heat map disabled after SetEnabled(false)")
	}

	if !h.needsRedraw {
		t.Error("expected needsRedraw to be set after disabling")
	}
}

func TestSetWindowDuration(t *testing.T) {
	h := NewActivityHeatMap()

	h.SetWindowDuration(30 * time.Minute)

	if h.config.WindowDuration != 30*time.Minute {
		t.Errorf("expected 30 minute window, got %v", h.config.WindowDuration)
	}

	if !h.needsRedraw {
		t.Error("expected needsRedraw to be set after changing window duration")
	}
}

func TestClear(t *testing.T) {
	h := NewActivityHeatMap()

	h.RecordActivity(0, 0, 0.5)
	h.RecordActivity(10, 10, 0.5)

	if len(h.samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(h.samples))
	}

	h.Clear()

	if len(h.samples) != 0 {
		t.Errorf("expected 0 samples after Clear, got %d", len(h.samples))
	}

	if !h.needsRedraw {
		t.Error("expected needsRedraw to be set after Clear")
	}
}

func TestSampleCount(t *testing.T) {
	h := NewActivityHeatMap()

	if h.SampleCount() != 0 {
		t.Errorf("expected 0 samples initially, got %d", h.SampleCount())
	}

	h.RecordActivity(0, 0, 0.5)
	h.RecordActivity(10, 10, 0.5)

	if h.SampleCount() != 2 {
		t.Errorf("expected 2 samples, got %d", h.SampleCount())
	}
}

func TestHeatMapColor(t *testing.T) {
	tests := []struct {
		intensity float32
		minR      uint8
		maxR      uint8
		minG      uint8
		maxG      uint8
		minB      uint8
		maxB      uint8
	}{
		{0.0, 0, 0, 0, 10, 95, 105},       // Blue (start of gradient)
		{0.25, 0, 0, 95, 105, 250, 255},   // Cyan (boundary)
		{0.5, 0, 10, 245, 255, 0, 10},     // Green (boundary) - B should be ~0
		{0.75, 245, 255, 245, 255, 0, 10}, // Yellow (boundary)
		{1.0, 245, 255, 95, 105, 0, 10},   // Red (end of gradient)
		{-0.5, 0, 0, 0, 10, 95, 105},      // Clamped to 0 (blue)
		{1.5, 245, 255, 95, 105, 0, 10},   // Clamped to 1 (red)
	}

	for _, tt := range tests {
		col := heatMapColor(tt.intensity)

		if col.R < tt.minR || col.R > tt.maxR {
			t.Errorf("intensity %f: expected R in [%d, %d], got %d", tt.intensity, tt.minR, tt.maxR, col.R)
		}

		if col.G < tt.minG || col.G > tt.maxG {
			t.Errorf("intensity %f: expected G in [%d, %d], got %d", tt.intensity, tt.minG, tt.maxG, col.G)
		}

		if col.B < tt.minB || col.B > tt.maxB {
			t.Errorf("intensity %f: expected B in [%d, %d], got %d", tt.intensity, tt.minB, tt.maxB, col.B)
		}

		if col.A != 180 {
			t.Errorf("intensity %f: expected A=180, got %d", tt.intensity, col.A)
		}
	}
}

func TestGridKeyGeneration(t *testing.T) {
	h := NewActivityHeatMap()
	h.config.GridCellSize = 100

	// Record activity and trigger grid generation
	h.RecordActivity(50, 50, 0.5)     // Cell (0, 0)
	h.RecordActivity(150, 150, 0.5)   // Cell (1, 1)
	h.RecordActivity(-150, -150, 0.5) // Cell (-2, -2) with floor division

	// Manually rebuild grid (simulating what regenerateHeatMap does)
	h.grid = make(map[gridKey]float32)
	for _, sample := range h.samples {
		cellX := int(math.Floor(sample.X / float64(h.config.GridCellSize)))
		cellY := int(math.Floor(sample.Y / float64(h.config.GridCellSize)))
		key := gridKey{cellX, cellY}
		h.grid[key] += sample.Intensity
	}

	// Check grid keys
	expectedKeys := []gridKey{
		{0, 0},
		{1, 1},
		{-2, -2},
	}

	if len(h.grid) != 3 {
		t.Errorf("expected 3 grid cells, got %d", len(h.grid))
	}

	for _, key := range expectedKeys {
		if _, ok := h.grid[key]; !ok {
			t.Errorf("expected grid cell %v not found", key)
		}
	}
}

func TestMultipleSamplesInSameCell(t *testing.T) {
	h := NewActivityHeatMap()
	h.config.GridCellSize = 100

	// Record multiple activities in the same grid cell
	h.RecordActivity(10, 10, 0.3)
	h.RecordActivity(20, 20, 0.2)
	h.RecordActivity(30, 30, 0.5)

	// All should map to cell (0, 0)
	h.grid = make(map[gridKey]float32)
	for _, sample := range h.samples {
		cellX := int(math.Floor(sample.X / float64(h.config.GridCellSize)))
		cellY := int(math.Floor(sample.Y / float64(h.config.GridCellSize)))
		key := gridKey{cellX, cellY}
		h.grid[key] += sample.Intensity
	}

	key := gridKey{0, 0}
	totalIntensity := h.grid[key]
	expectedTotal := float32(1.0) // 0.3 + 0.2 + 0.5

	if totalIntensity < expectedTotal-0.01 || totalIntensity > expectedTotal+0.01 {
		t.Errorf("expected total intensity ~%f in cell (0,0), got %f", expectedTotal, totalIntensity)
	}
}
