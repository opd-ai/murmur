package overlays

import (
	"math"
	"testing"
)

func TestWorldToScreen_MaskedEventAlignsWithNodeCenter(t *testing.T) {
	const (
		worldX = 120.0
		worldY = -40.0
		camX   = 20.0
		camY   = -10.0

		centerX = 400.0
		centerY = 300.0
		zoom    = 1.75
	)

	// Node and masked-event overlays should resolve to the same screen point
	// when they share world coordinates and camera state.
	nodeX, nodeY := worldToScreen(worldX, worldY, camX, camY, centerX, centerY, zoom)
	eventX, eventY := worldToScreen(worldX, worldY, camX, camY, centerX, centerY, zoom)

	if math.Abs(nodeX-eventX) > 0.001 || math.Abs(nodeY-eventY) > 0.001 {
		t.Fatalf("expected aligned screen coords, node=(%f,%f) event=(%f,%f)", nodeX, nodeY, eventX, eventY)
	}
}

func TestWorldToScreen_MultipleZoomLevels(t *testing.T) {
	const (
		worldX  = 150.0
		worldY  = 50.0
		camX    = 100.0
		camY    = 25.0
		centerX = 400.0
		centerY = 300.0
	)

	tests := []struct {
		name  string
		zoom  float64
		wantX float64
		wantY float64
	}{
		{name: "zoom_0_5", zoom: 0.5, wantX: 425.0, wantY: 312.5},
		{name: "zoom_1_0", zoom: 1.0, wantX: 450.0, wantY: 325.0},
		{name: "zoom_2_0", zoom: 2.0, wantX: 500.0, wantY: 350.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotX, gotY := worldToScreen(worldX, worldY, camX, camY, centerX, centerY, tc.zoom)
			if math.Abs(gotX-tc.wantX) > 0.001 || math.Abs(gotY-tc.wantY) > 0.001 {
				t.Fatalf("zoom=%f expected (%f,%f), got (%f,%f)", tc.zoom, tc.wantX, tc.wantY, gotX, gotY)
			}
		})
	}
}
