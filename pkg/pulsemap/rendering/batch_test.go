package rendering

import (
	"image/color"
	"testing"
)

func TestNewBatchRenderer(t *testing.T) {
	batch := NewBatchRenderer()
	if batch == nil {
		t.Fatal("NewBatchRenderer returned nil")
	}
}

func TestBatchRendererAddEdge(t *testing.T) {
	batch := NewBatchRenderer()

	style := EdgeStyle{
		Color:                color.RGBA{100, 120, 140, 255},
		Age:                  10.0,
		Active:               false,
		InteractionFrequency: 5.0,
	}

	// Add multiple edges - should not panic
	batch.AddEdge(0, 0, 100, 100, style, ZoomMeso)
	batch.AddEdge(50, 50, 150, 150, style, ZoomMeso)
	batch.AddEdge(100, 100, 200, 200, style, ZoomMeso)
}

func TestBatchRendererAddNode(t *testing.T) {
	batch := NewBatchRenderer()

	style := NodeStyle{
		CoreColor:   color.RGBA{200, 150, 100, 255},
		RingColor:   color.RGBA{255, 255, 255, 255},
		HasRing:     true,
		HasHalo:     true,
		HaloAlpha:   0.5,
		IsSpecter:   false,
		Selected:    false,
		Connections: 5,
		Activity:    10.0,
	}

	// Add multiple nodes - should not panic
	batch.AddNode(100, 100, 8.0, style)
	batch.AddNode(200, 200, 10.0, style)
	batch.AddNode(300, 300, 12.0, style)
}

func TestBatchRendererAddParticle(t *testing.T) {
	batch := NewBatchRenderer()

	particleColor := color.RGBA{150, 200, 250, 200}

	// Add multiple particles - should not panic
	batch.AddParticle(50, 50, 2.0, particleColor)
	batch.AddParticle(60, 60, 2.5, particleColor)
	batch.AddParticle(70, 70, 3.0, particleColor)
	batch.AddParticle(80, 80, 2.0, particleColor)
}

func TestBatchRendererAddTrail(t *testing.T) {
	batch := NewBatchRenderer()

	// Add multiple trails - should not panic
	batch.AddTrail(0, 0, 100, 100, 150.0, false, 1.0)
	batch.AddTrail(50, 50, 150, 150, 120.0, true, 1.5)
}

func TestBatchRendererClear(t *testing.T) {
	batch := NewBatchRenderer()

	// Add some items
	style := EdgeStyle{Color: color.RGBA{100, 120, 140, 255}}
	batch.AddEdge(0, 0, 100, 100, style, ZoomMeso)
	batch.AddNode(100, 100, 8.0, NodeStyle{CoreColor: color.RGBA{200, 150, 100, 255}})
	batch.AddParticle(50, 50, 2.0, color.RGBA{150, 200, 250, 200})
	batch.AddTrail(0, 0, 100, 100, 150.0, false, 1.0)

	// Clear should not panic
	batch.Clear()

	// Adding after clear should work
	batch.AddEdge(0, 0, 100, 100, style, ZoomMeso)
}

func TestBatchRendererMixedOperations(t *testing.T) {
	batch := NewBatchRenderer()

	// Add various items
	edgeStyle := EdgeStyle{Color: color.RGBA{100, 120, 140, 255}}
	nodeStyle := NodeStyle{CoreColor: color.RGBA{200, 150, 100, 255}}
	particleColor := color.RGBA{150, 200, 250, 200}

	batch.AddEdge(0, 0, 100, 100, edgeStyle, ZoomMeso)
	batch.AddEdge(50, 50, 150, 150, edgeStyle, ZoomMeso)
	batch.AddNode(100, 100, 8.0, nodeStyle)
	batch.AddParticle(50, 50, 2.0, particleColor)
	batch.AddTrail(0, 0, 100, 100, 150.0, false, 1.0)
	batch.AddNode(200, 200, 10.0, nodeStyle)
	batch.AddParticle(60, 60, 2.5, particleColor)
}

func TestBatchRendererFlush(t *testing.T) {
	batch := NewBatchRenderer()

	// Add items
	edgeStyle := EdgeStyle{Color: color.RGBA{100, 120, 140, 255}}
	batch.AddEdge(0, 0, 100, 100, edgeStyle, ZoomMeso)

	// Flush should not panic (even with nil dst in stub)
	batch.Flush(nil)
}

func TestBatchRendererSpecterNodes(t *testing.T) {
	batch := NewBatchRenderer()

	specterStyle := NodeStyle{
		CoreColor:   color.RGBA{100, 150, 200, 255},
		RingColor:   color.RGBA{150, 180, 220, 255},
		HasRing:     true,
		HasHalo:     true,
		HaloAlpha:   0.7,
		IsSpecter:   true,
		Connections: 3,
		Resonance:   50.0,
	}

	// Add Specter nodes - should not panic
	batch.AddNode(100, 100, 8.0, specterStyle)
	batch.AddNode(200, 200, 10.0, specterStyle)
}

func TestBatchRendererActiveEdges(t *testing.T) {
	batch := NewBatchRenderer()

	activeEdgeStyle := EdgeStyle{
		Color:                color.RGBA{100, 120, 140, 255},
		Age:                  10.0,
		Active:               true,
		InteractionFrequency: 15.0,
	}

	inactiveEdgeStyle := EdgeStyle{
		Color:                color.RGBA{100, 120, 140, 255},
		Age:                  50.0,
		Active:               false,
		InteractionFrequency: 2.0,
	}

	// Add both active and inactive edges - should not panic
	batch.AddEdge(0, 0, 100, 100, activeEdgeStyle, ZoomMeso)
	batch.AddEdge(50, 50, 150, 150, inactiveEdgeStyle, ZoomMeso)
	batch.AddEdge(100, 100, 200, 200, activeEdgeStyle, ZoomMeso)
}

func TestBatchRendererMacroZoomCulling(t *testing.T) {
	batch := NewBatchRenderer()

	// Old edges at macro zoom should be culled
	oldEdgeStyle := EdgeStyle{
		Color:                color.RGBA{100, 120, 140, 255},
		Age:                  40.0,
		Active:               false,
		InteractionFrequency: 1.0,
	}

	// Add edge at macro zoom - should be culled (no panic)
	batch.AddEdge(0, 0, 100, 100, oldEdgeStyle, ZoomMacro)
}
