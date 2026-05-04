// Package overlays - Minimap tests.
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"testing"
)

func TestNewMinimap(t *testing.T) {
	m := NewMinimap()

	if m == nil {
		t.Fatal("NewMinimap returned nil")
	}

	if m.config.Width != 150 {
		t.Errorf("expected default width 150, got %f", m.config.Width)
	}

	if m.config.Height != 150 {
		t.Errorf("expected default height 150, got %f", m.config.Height)
	}

	if m.config.Position != MinimapBottomRight {
		t.Errorf("expected default position MinimapBottomRight, got %d", m.config.Position)
	}

	if !m.needsRedraw {
		t.Error("expected needsRedraw to be true on initialization")
	}
}

func TestNewMinimapWithConfig(t *testing.T) {
	config := MinimapConfig{
		Width:         200,
		Height:        180,
		Margin:        15,
		Position:      MinimapTopLeft,
		BgColor:       color.RGBA{20, 20, 20, 255},
		NodeColor:     color.RGBA{255, 255, 255, 255},
		ViewportColor: color.RGBA{255, 0, 0, 200},
		BorderColor:   color.RGBA{100, 100, 100, 255},
	}

	m := NewMinimapWithConfig(config)

	if m == nil {
		t.Fatal("NewMinimapWithConfig returned nil")
	}

	if m.config.Width != 200 {
		t.Errorf("expected width 200, got %f", m.config.Width)
	}

	if m.config.Height != 180 {
		t.Errorf("expected height 180, got %f", m.config.Height)
	}

	if m.config.Position != MinimapTopLeft {
		t.Errorf("expected position MinimapTopLeft, got %d", m.config.Position)
	}
}

func TestUpdateNodes(t *testing.T) {
	m := NewMinimap()

	nodes := []MinimapNode{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
		{X: -50, Y: 50},
	}

	m.UpdateNodes(nodes)

	if len(m.nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(m.nodes))
	}

	if !m.needsRedraw {
		t.Error("expected needsRedraw to be set after UpdateNodes")
	}

	// Check world bounds were calculated
	if m.worldMinX >= m.worldMaxX {
		t.Errorf("invalid world X bounds: min=%f, max=%f", m.worldMinX, m.worldMaxX)
	}

	if m.worldMinY >= m.worldMaxY {
		t.Errorf("invalid world Y bounds: min=%f, max=%f", m.worldMinY, m.worldMaxY)
	}
}

func TestCalculateWorldBounds(t *testing.T) {
	m := NewMinimap()

	// Test empty nodes
	m.calculateWorldBounds()
	if m.worldMinX != -500 || m.worldMaxX != 500 {
		t.Errorf("expected default X bounds [-500, 500], got [%f, %f]", m.worldMinX, m.worldMaxX)
	}

	// Test with nodes
	m.nodes = []MinimapNode{
		{X: 0, Y: 0},
		{X: 100, Y: 200},
		{X: -100, Y: -200},
	}

	m.calculateWorldBounds()

	// Should have padding (10% of range)
	if m.worldMinX > -100 {
		t.Errorf("expected worldMinX <= -100 (with padding), got %f", m.worldMinX)
	}

	if m.worldMaxX < 100 {
		t.Errorf("expected worldMaxX >= 100 (with padding), got %f", m.worldMaxX)
	}

	if m.worldMinY > -200 {
		t.Errorf("expected worldMinY <= -200 (with padding), got %f", m.worldMinY)
	}

	if m.worldMaxY < 200 {
		t.Errorf("expected worldMaxY >= 200 (with padding), got %f", m.worldMaxY)
	}
}

func TestWorldToMinimap(t *testing.T) {
	m := NewMinimap()
	m.config.Width = 100
	m.config.Height = 100

	m.worldMinX = -100
	m.worldMaxX = 100
	m.worldMinY = -100
	m.worldMaxY = 100

	// Test center point
	x, y := m.worldToMinimap(0, 0)
	expectedCenter := float32(50)
	if x < expectedCenter-5 || x > expectedCenter+5 {
		t.Errorf("expected x near %f, got %f", expectedCenter, x)
	}
	if y < expectedCenter-5 || y > expectedCenter+5 {
		t.Errorf("expected y near %f, got %f", expectedCenter, y)
	}

	// Test corners (accounting for padding)
	x1, _ := m.worldToMinimap(-100, -100)
	if x1 < 0 || x1 > 10 {
		t.Errorf("expected top-left x near 5 (with padding), got %f", x1)
	}

	x2, y2 := m.worldToMinimap(100, 100)
	if x2 < 90 || x2 > 100 {
		t.Errorf("expected bottom-right x near 95 (with padding), got %f", x2)
	}
	if y2 < 90 || y2 > 100 {
		t.Errorf("expected bottom-right y near 95 (with padding), got %f", y2)
	}
}

func TestGetMinimapPosition(t *testing.T) {
	m := NewMinimap()
	m.config.Width = 150
	m.config.Height = 150
	m.config.Margin = 10

	screenW := 800
	screenH := 600

	tests := []struct {
		pos     MinimapPos
		expectX float32
		expectY float32
	}{
		{MinimapTopRight, 640, 10}, // 800 - 150 - 10 = 640
		{MinimapTopLeft, 10, 10},
		{MinimapBottomRight, 640, 440}, // 600 - 150 - 10 = 440
		{MinimapBottomLeft, 10, 440},
	}

	for _, tt := range tests {
		m.config.Position = tt.pos
		x, y := m.getMinimapPosition(screenW, screenH)

		if x != tt.expectX {
			t.Errorf("position %d: expected x=%f, got %f", tt.pos, tt.expectX, x)
		}

		if y != tt.expectY {
			t.Errorf("position %d: expected y=%f, got %f", tt.pos, tt.expectY, y)
		}
	}
}

func TestSetPosition(t *testing.T) {
	m := NewMinimap()

	m.SetPosition(MinimapTopLeft)
	if m.config.Position != MinimapTopLeft {
		t.Errorf("expected position MinimapTopLeft, got %d", m.config.Position)
	}

	m.SetPosition(MinimapBottomLeft)
	if m.config.Position != MinimapBottomLeft {
		t.Errorf("expected position MinimapBottomLeft, got %d", m.config.Position)
	}
}

func TestSetSize(t *testing.T) {
	m := NewMinimap()
	m.needsRedraw = false

	m.SetSize(200, 180)

	if m.config.Width != 200 {
		t.Errorf("expected width 200, got %f", m.config.Width)
	}

	if m.config.Height != 180 {
		t.Errorf("expected height 180, got %f", m.config.Height)
	}

	if !m.needsRedraw {
		t.Error("expected needsRedraw to be set after SetSize")
	}
}

func TestIsVisible(t *testing.T) {
	m := NewMinimap()

	// No nodes, should not be visible
	if m.IsVisible() {
		t.Error("expected minimap to be invisible with no nodes")
	}

	// Add nodes, should be visible
	m.UpdateNodes([]MinimapNode{{X: 0, Y: 0}})
	if !m.IsVisible() {
		t.Error("expected minimap to be visible with nodes")
	}
}

func TestContainsPoint(t *testing.T) {
	m := NewMinimap()
	m.config.Width = 150
	m.config.Height = 150
	m.config.Margin = 10
	m.config.Position = MinimapBottomRight

	screenW := 800
	screenH := 600

	// Minimap at (640, 440) with size 150x150

	// Point inside
	if !m.ContainsPoint(700, 500, screenW, screenH) {
		t.Error("expected point (700, 500) to be inside minimap")
	}

	// Point outside (left of minimap)
	if m.ContainsPoint(500, 500, screenW, screenH) {
		t.Error("expected point (500, 500) to be outside minimap")
	}

	// Point outside (above minimap)
	if m.ContainsPoint(700, 300, screenW, screenH) {
		t.Error("expected point (700, 300) to be outside minimap")
	}

	// Point on border (should be inside)
	if !m.ContainsPoint(640, 440, screenW, screenH) {
		t.Error("expected point (640, 440) on border to be inside minimap")
	}
}

func TestDistanceToEdge(t *testing.T) {
	m := NewMinimap()
	m.config.Width = 150
	m.config.Height = 150
	m.config.Margin = 10
	m.config.Position = MinimapBottomRight

	screenW := 800
	screenH := 600

	// Point inside minimap should return 0
	dist := m.DistanceToEdge(700, 500, screenW, screenH)
	if dist != 0 {
		t.Errorf("expected distance 0 for point inside minimap, got %f", dist)
	}

	// Point outside should return positive distance
	dist = m.DistanceToEdge(600, 500, screenW, screenH)
	if dist <= 0 {
		t.Errorf("expected positive distance for point outside minimap, got %f", dist)
	}
}

func TestClampFloat32(t *testing.T) {
	tests := []struct {
		value    float32
		min      float32
		max      float32
		expected float32
	}{
		{5, 0, 10, 5},   // Within range
		{-5, 0, 10, 0},  // Below min
		{15, 0, 10, 10}, // Above max
		{0, 0, 10, 0},   // At min
		{10, 0, 10, 10}, // At max
	}

	for _, tt := range tests {
		result := clampFloat32(tt.value, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("clampFloat32(%f, %f, %f) = %f, expected %f",
				tt.value, tt.min, tt.max, result, tt.expected)
		}
	}
}
