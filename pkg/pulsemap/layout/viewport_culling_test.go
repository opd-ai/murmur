// Package layout - Tests for viewport culling.
package layout

import (
	"math"
	"testing"
)

func TestNewViewportCulling(t *testing.T) {
	vc := NewViewportCulling()

	if vc == nil {
		t.Fatal("NewViewportCulling returned nil")
	}
	if vc.margin != 200.0 {
		t.Errorf("expected default margin 200, got %f", vc.margin)
	}
	if vc.zoom != 1.0 {
		t.Error("default zoom should be 1.0")
	}
}

func TestViewportCulling_SetMargin(t *testing.T) {
	vc := NewViewportCulling()

	vc.SetMargin(100.0)
	if vc.margin != 100.0 {
		t.Error("margin should be updated")
	}
}

func TestViewportCulling_SetCamera(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)

	vc.SetCamera(100, 200, 2.0)

	if vc.cameraX != 100 || vc.cameraY != 200 {
		t.Error("camera position not set correctly")
	}
	if vc.zoom != 2.0 {
		t.Error("zoom not set correctly")
	}

	// Check bounds calculation.
	// With zoom 2.0, visible area is half as large.
	// Screen 800x600 -> world size 400x300 at zoom 2.0
	// Center at (100, 200) -> bounds are (-100, 50) to (300, 350)
	minX, minY, maxX, maxY := vc.GetBounds()
	expectedMinX := 100 - (800/2)/2.0
	expectedMaxX := 100 + (800/2)/2.0
	expectedMinY := 200 - (600/2)/2.0
	expectedMaxY := 200 + (600/2)/2.0

	if math.Abs(minX-expectedMinX) > 0.01 || math.Abs(maxX-expectedMaxX) > 0.01 {
		t.Errorf("X bounds incorrect: got (%f, %f), expected (%f, %f)",
			minX, maxX, expectedMinX, expectedMaxX)
	}
	if math.Abs(minY-expectedMinY) > 0.01 || math.Abs(maxY-expectedMaxY) > 0.01 {
		t.Errorf("Y bounds incorrect: got (%f, %f), expected (%f, %f)",
			minY, maxY, expectedMinY, expectedMaxY)
	}
}

func TestViewportCulling_UpdateVisibility(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	// Create positions.
	positions := map[string]Position{
		"visible":  {X: 0, Y: 0},       // At center.
		"marginal": {X: 500, Y: 0},     // Outside viewport, in margin.
		"culled":   {X: 1000, Y: 1000}, // Far outside margin.
	}

	vc.UpdateVisibility(positions)

	if !vc.IsVisible("visible") {
		t.Error("center node should be visible")
	}
	if vc.IsVisible("marginal") {
		t.Error("marginal node should not be in viewport")
	}
	if !vc.IsInMargin("marginal") {
		t.Error("marginal node should be in margin")
	}
	if !vc.IsCulled("culled") {
		t.Error("far node should be culled")
	}
}

func TestViewportCulling_ShouldComputeForces(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	positions := map[string]Position{
		"visible":  {X: 0, Y: 0},
		"marginal": {X: 500, Y: 0},
		"culled":   {X: 1000, Y: 1000},
	}

	vc.UpdateVisibility(positions)

	if !vc.ShouldComputeForces("visible") {
		t.Error("visible node should compute forces")
	}
	if !vc.ShouldComputeForces("marginal") {
		t.Error("marginal node should compute forces")
	}
	if vc.ShouldComputeForces("culled") {
		t.Error("culled node should not compute forces")
	}
}

func TestViewportCulling_GetVisibleNodes(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)

	positions := map[string]Position{
		"a": {X: 0, Y: 0},
		"b": {X: 100, Y: 100},
		"c": {X: 1000, Y: 1000}, // Culled.
	}

	vc.UpdateVisibility(positions)

	visible := vc.GetVisibleNodes()

	// a and b should be visible.
	if len(visible) != 2 {
		t.Errorf("expected 2 visible nodes, got %d", len(visible))
	}
}

func TestViewportCulling_GetActiveNodes(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	positions := map[string]Position{
		"visible":  {X: 0, Y: 0},
		"marginal": {X: 500, Y: 0},
		"culled":   {X: 2000, Y: 2000},
	}

	vc.UpdateVisibility(positions)

	active := vc.GetActiveNodes()

	// visible and marginal should be active.
	if len(active) != 2 {
		t.Errorf("expected 2 active nodes, got %d", len(active))
	}
}

func TestViewportCulling_GetBoundsWithMargin(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(400, 300)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(50.0)

	minX, minY, maxX, maxY := vc.GetBoundsWithMargin()

	// Viewport is -200 to 200 in X, -150 to 150 in Y.
	// With 50 margin: -250 to 250 in X, -200 to 200 in Y.
	if math.Abs(minX-(-250)) > 0.01 || math.Abs(maxX-250) > 0.01 {
		t.Errorf("X bounds with margin incorrect: (%f, %f)", minX, maxX)
	}
	if math.Abs(minY-(-200)) > 0.01 || math.Abs(maxY-200) > 0.01 {
		t.Errorf("Y bounds with margin incorrect: (%f, %f)", minY, maxY)
	}
}

func TestViewportCulling_GetStats(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	positions := map[string]Position{
		"v1":     {X: 0, Y: 0},
		"v2":     {X: 50, Y: 50},
		"m1":     {X: 500, Y: 0},
		"culled": {X: 2000, Y: 2000},
	}

	vc.UpdateVisibility(positions)

	stats := vc.GetStats()

	if stats.VisibleCount != 2 {
		t.Errorf("expected 2 visible, got %d", stats.VisibleCount)
	}
	if stats.MarginalCount != 1 {
		t.Errorf("expected 1 marginal, got %d", stats.MarginalCount)
	}
	if stats.CulledCount != 1 {
		t.Errorf("expected 1 culled, got %d", stats.CulledCount)
	}
	if stats.TotalCount != 4 {
		t.Errorf("expected 4 total, got %d", stats.TotalCount)
	}
	if math.Abs(stats.CullRatio-0.25) > 0.01 {
		t.Errorf("expected cull ratio 0.25, got %f", stats.CullRatio)
	}
}

func TestViewportCulling_FilterEdges(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	positions := map[string]Position{
		"active1": {X: 0, Y: 0},
		"active2": {X: 100, Y: 100},
		"culled1": {X: 2000, Y: 2000},
		"culled2": {X: 3000, Y: 3000},
	}

	vc.UpdateVisibility(positions)

	edges := []Edge{
		{SourceID: "active1", TargetID: "active2"}, // Both active.
		{SourceID: "active1", TargetID: "culled1"}, // One active.
		{SourceID: "culled1", TargetID: "culled2"}, // Neither active.
	}

	filtered := vc.FilterEdges(edges)

	// First two edges should be kept.
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered edges, got %d", len(filtered))
	}
}

func TestViewportCulling_ContainsPoint(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)

	// Center should be contained.
	if !vc.ContainsPoint(0, 0) {
		t.Error("center should be contained")
	}

	// Point at edge should be contained.
	if !vc.ContainsPoint(399, 299) {
		t.Error("edge point should be contained")
	}

	// Point outside should not be contained.
	if vc.ContainsPoint(500, 0) {
		t.Error("outside point should not be contained")
	}
}

func TestViewportCulling_ContainsPointWithMargin(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(800, 600)
	vc.SetCamera(0, 0, 1.0)
	vc.SetMargin(100.0)

	// Point just outside viewport but in margin.
	if !vc.ContainsPointWithMargin(450, 0) {
		t.Error("point in margin should be contained")
	}

	// Point far outside.
	if vc.ContainsPointWithMargin(600, 0) {
		t.Error("far point should not be contained")
	}
}

func TestViewportCulling_DistanceToViewport(t *testing.T) {
	vc := NewViewportCulling()
	vc.SetScreenSize(400, 300) // Viewport -200 to 200 in X, -150 to 150 in Y.
	vc.SetCamera(0, 0, 1.0)

	// Point inside viewport.
	dist := vc.DistanceToViewport(0, 0)
	if dist != 0 {
		t.Errorf("inside point distance should be 0, got %f", dist)
	}

	// Point at edge.
	dist = vc.DistanceToViewport(200, 0)
	if dist != 0 {
		t.Errorf("edge point distance should be 0, got %f", dist)
	}

	// Point outside.
	dist = vc.DistanceToViewport(300, 0)
	if math.Abs(dist-100) > 0.01 {
		t.Errorf("expected distance 100, got %f", dist)
	}

	// Point outside diagonally.
	dist = vc.DistanceToViewport(203, 154)
	expected := math.Sqrt(9 + 16) // sqrt(3^2 + 4^2) = 5
	if math.Abs(dist-expected) > 0.01 {
		t.Errorf("expected diagonal distance %f, got %f", expected, dist)
	}
}

func TestNewCulledEngine(t *testing.T) {
	ce := NewCulledEngine()

	if ce == nil {
		t.Fatal("NewCulledEngine returned nil")
	}
	if ce.Engine == nil {
		t.Error("embedded Engine should not be nil")
	}
	if ce.culling == nil {
		t.Error("culling manager should not be nil")
	}
	if !ce.IsCullingEnabled() {
		t.Error("culling should be enabled by default")
	}
}

func TestCulledEngine_EnableDisable(t *testing.T) {
	ce := NewCulledEngine()

	ce.SetCullingEnabled(false)
	if ce.IsCullingEnabled() {
		t.Error("culling should be disabled")
	}

	ce.SetCullingEnabled(true)
	if !ce.IsCullingEnabled() {
		t.Error("culling should be enabled")
	}
}

func TestCulledEngine_Culling(t *testing.T) {
	ce := NewCulledEngine()

	culling := ce.Culling()
	if culling == nil {
		t.Error("Culling() should not return nil")
	}
	if culling != ce.culling {
		t.Error("Culling() should return the culling manager")
	}
}

func TestCulledEngine_TickWithCulling(t *testing.T) {
	ce := NewCulledEngine()
	ce.Culling().SetScreenSize(800, 600)
	ce.Culling().SetCamera(0, 0, 1.0)

	// Add some nodes.
	for i := 0; i < 10; i++ {
		ce.Engine.AddNode(&Node{
			ID:       generateClusterID(i),
			Activity: 1.0,
		})
	}

	// Run a tick with culling.
	ce.TickWithCulling()

	// Should have updated positions.
	positions := ce.Engine.Positions().Get()
	if len(positions) != 10 {
		t.Errorf("expected 10 positions, got %d", len(positions))
	}
}

func TestCulledEngine_TickWithoutCulling(t *testing.T) {
	ce := NewCulledEngine()
	ce.SetCullingEnabled(false)

	// Add some nodes.
	for i := 0; i < 5; i++ {
		ce.Engine.AddNode(&Node{
			ID:       generateClusterID(i),
			Activity: 1.0,
		})
	}

	// Run a tick without culling.
	ce.TickWithCulling()

	// Should still work.
	positions := ce.Engine.Positions().Get()
	if len(positions) != 5 {
		t.Errorf("expected 5 positions, got %d", len(positions))
	}
}

func TestCulledEngine_WithEdges(t *testing.T) {
	ce := NewCulledEngine()
	ce.Culling().SetScreenSize(800, 600)
	ce.Culling().SetCamera(0, 0, 1.0)

	// Add nodes and edges.
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = generateClusterID(i)
		ce.Engine.AddNode(&Node{
			ID:       ids[i],
			Activity: 1.0,
		})
	}

	for i := 0; i < 4; i++ {
		ce.Engine.AddEdge(Edge{
			SourceID: ids[i],
			TargetID: ids[i+1],
		})
	}

	// Run multiple ticks.
	for i := 0; i < 10; i++ {
		ce.TickWithCulling()
	}

	// Should have converged somewhat.
	positions := ce.Engine.Positions().Get()
	if len(positions) != 5 {
		t.Errorf("expected 5 positions, got %d", len(positions))
	}
}

func TestCullStats_Fields(t *testing.T) {
	stats := CullStats{
		VisibleCount:  100,
		MarginalCount: 50,
		CulledCount:   350,
		TotalCount:    500,
		CullRatio:     0.7,
	}

	if stats.VisibleCount != 100 {
		t.Error("VisibleCount mismatch")
	}
	if stats.TotalCount != 500 {
		t.Error("TotalCount mismatch")
	}
	if stats.CullRatio != 0.7 {
		t.Error("CullRatio mismatch")
	}
}
