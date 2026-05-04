// Package overlays - Echo Chain overlay tests.
//
//go:build !test
// +build !test

package overlays

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewEchoChainOverlay(t *testing.T) {
	overlay := NewEchoChainOverlay()

	if overlay == nil {
		t.Fatal("NewEchoChainOverlay returned nil")
	}
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible by default")
	}
	if overlay.ChainCount() != 0 {
		t.Error("expected no chains initially")
	}
}

func TestEchoChainOverlay_Visibility(t *testing.T) {
	overlay := NewEchoChainOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("expected overlay to be hidden")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible")
	}
}

func TestEchoChainOverlay_SetChain(t *testing.T) {
	overlay := NewEchoChainOverlay()

	chain := &EchoChainInfo{
		ChainID:    [32]byte{1, 2, 3},
		OriginalID: [32]byte{4, 5, 6},
		Layer:      ChainLayerSurface,
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, X: 100, Y: 100, Position: 0},
			{NodeID: [32]byte{20}, X: 200, Y: 200, Position: 1},
			{NodeID: [32]byte{30}, X: 300, Y: 300, Position: 2},
		},
		FormedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: false,
	}

	overlay.SetChain(chain)

	if overlay.ChainCount() != 1 {
		t.Errorf("expected 1 chain, got %d", overlay.ChainCount())
	}

	retrieved := overlay.GetChain(chain.ChainID)
	if retrieved == nil {
		t.Fatal("GetChain returned nil")
	}
	if len(retrieved.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(retrieved.Nodes))
	}
}

func TestEchoChainOverlay_SetChainNil(t *testing.T) {
	overlay := NewEchoChainOverlay()
	overlay.SetChain(nil)

	if overlay.ChainCount() != 0 {
		t.Error("setting nil chain should not add anything")
	}
}

func TestEchoChainOverlay_RemoveChain(t *testing.T) {
	overlay := NewEchoChainOverlay()

	chain := &EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	overlay.SetChain(chain)
	if overlay.ChainCount() != 1 {
		t.Error("chain not added")
	}

	overlay.RemoveChain(chain.ChainID)
	if overlay.ChainCount() != 0 {
		t.Error("chain not removed")
	}
}

func TestEchoChainOverlay_GetAllChains(t *testing.T) {
	overlay := NewEchoChainOverlay()

	for i := 0; i < 5; i++ {
		chain := &EchoChainInfo{
			ChainID:   [32]byte{byte(i)},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		overlay.SetChain(chain)
	}

	chains := overlay.GetAllChains()
	if len(chains) != 5 {
		t.Errorf("expected 5 chains, got %d", len(chains))
	}
}

func TestEchoChainOverlay_GetActiveChains(t *testing.T) {
	overlay := NewEchoChainOverlay()

	// Add active chains.
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{2},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	// Add expired chain.
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{3},
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	active := overlay.GetActiveChains()
	if len(active) != 2 {
		t.Errorf("expected 2 active chains, got %d", len(active))
	}

	if overlay.ActiveChainCount() != 2 {
		t.Errorf("ActiveChainCount expected 2, got %d", overlay.ActiveChainCount())
	}
}

func TestEchoChainOverlay_Update(t *testing.T) {
	overlay := NewEchoChainOverlay()

	// Update should not panic.
	overlay.Update(0.016)
	overlay.Update(0.016)
	overlay.Update(0.016)
}

func TestEchoChainOverlay_Draw(t *testing.T) {
	overlay := NewEchoChainOverlay()
	screen := ebiten.NewImage(800, 600)

	// Draw empty overlay.
	overlay.Draw(screen, 0, 0, 1.0)

	// Add chain and draw.
	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{1},
		Layer:      ChainLayerSurface,
		FormedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: false,
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, X: 100, Y: 100, Position: 0},
			{NodeID: [32]byte{20}, X: 200, Y: 200, Position: 1},
			{NodeID: [32]byte{30}, X: 300, Y: 300, Position: 2},
		},
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Draw with camera offset.
	overlay.Draw(screen, 50, 50, 1.5)

	// Draw when hidden.
	overlay.SetVisible(false)
	overlay.Draw(screen, 0, 0, 1.0)
}

func TestEchoChainOverlay_DrawAnonymousChain(t *testing.T) {
	overlay := NewEchoChainOverlay()
	screen := ebiten.NewImage(800, 600)

	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{1},
		Layer:      ChainLayerAnonymous,
		FormedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: true,
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, X: 100, Y: 100, Position: 0},
			{NodeID: [32]byte{20}, X: 200, Y: 200, Position: 1},
			{NodeID: [32]byte{30}, X: 300, Y: 300, Position: 2},
			{NodeID: [32]byte{40}, X: 400, Y: 400, Position: 3},
			{NodeID: [32]byte{50}, X: 500, Y: 500, Position: 4},
		},
	})

	overlay.Draw(screen, 0, 0, 1.0)
}

func TestEchoChainOverlay_ShimmeringChains(t *testing.T) {
	overlay := NewEchoChainOverlay()

	// Non-shimmering chain (less than 5 nodes).
	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{1},
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: false,
	})

	// Shimmering chain.
	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{2},
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: true,
	})

	if overlay.ShimmeringChainCount() != 1 {
		t.Errorf("expected 1 shimmering chain, got %d", overlay.ShimmeringChainCount())
	}
}

func TestEchoChainOverlay_ClearExpired(t *testing.T) {
	overlay := NewEchoChainOverlay()

	// Add expired chain.
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	// Add active chain.
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{2},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	removed := overlay.ClearExpired()

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	if overlay.ChainCount() != 1 {
		t.Errorf("expected 1 chain remaining, got %d", overlay.ChainCount())
	}
}

func TestEchoChainOverlay_UpdateNodePosition(t *testing.T) {
	overlay := NewEchoChainOverlay()

	nodeID := [32]byte{10}
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes: []*ChainNodeInfo{
			{NodeID: nodeID, X: 100, Y: 100},
		},
	})

	overlay.UpdateNodePosition(nodeID, 200, 300)

	chain := overlay.GetChain([32]byte{1})
	if chain.Nodes[0].X != 200 || chain.Nodes[0].Y != 300 {
		t.Error("node position not updated")
	}
}

func TestEchoChainOverlay_GetChainsByLayer(t *testing.T) {
	overlay := NewEchoChainOverlay()

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		Layer:     ChainLayerSurface,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{2},
		Layer:     ChainLayerSurface,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{3},
		Layer:     ChainLayerAnonymous,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	surface := overlay.GetChainsByLayer(ChainLayerSurface)
	if len(surface) != 2 {
		t.Errorf("expected 2 surface chains, got %d", len(surface))
	}

	anonymous := overlay.GetChainsByLayer(ChainLayerAnonymous)
	if len(anonymous) != 1 {
		t.Errorf("expected 1 anonymous chain, got %d", len(anonymous))
	}
}

func TestEchoChainOverlay_AddNodeToChain(t *testing.T) {
	overlay := NewEchoChainOverlay()

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, Position: 0},
			{NodeID: [32]byte{20}, Position: 1},
			{NodeID: [32]byte{30}, Position: 2},
		},
	})

	// Add node.
	overlay.AddNodeToChain([32]byte{1}, &ChainNodeInfo{
		NodeID: [32]byte{40},
		X:      400,
		Y:      400,
	})

	chain := overlay.GetChain([32]byte{1})
	if len(chain.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(chain.Nodes))
	}
	if chain.Nodes[3].Position != 3 {
		t.Errorf("expected position 3, got %d", chain.Nodes[3].Position)
	}

	// Add more nodes to trigger shimmer.
	overlay.AddNodeToChain([32]byte{1}, &ChainNodeInfo{NodeID: [32]byte{50}})
	if !chain.HasShimmer {
		t.Error("expected shimmer after 5 nodes")
	}
}

func TestEchoChainOverlay_AddNodeToChainNil(t *testing.T) {
	overlay := NewEchoChainOverlay()

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes:     []*ChainNodeInfo{},
	})

	overlay.AddNodeToChain([32]byte{1}, nil)

	chain := overlay.GetChain([32]byte{1})
	if len(chain.Nodes) != 0 {
		t.Error("adding nil node should not add anything")
	}
}

func TestEchoChainOverlay_GetLongestChain(t *testing.T) {
	overlay := NewEchoChainOverlay()

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}},
			{NodeID: [32]byte{20}},
		},
	})

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{2},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{30}},
			{NodeID: [32]byte{40}},
			{NodeID: [32]byte{50}},
			{NodeID: [32]byte{60}},
		},
	})

	overlay.SetChain(&EchoChainInfo{
		ChainID:   [32]byte{3},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{70}},
		},
	})

	longest := overlay.GetLongestChain()
	if longest == nil {
		t.Fatal("GetLongestChain returned nil")
	}
	if longest.ChainID != [32]byte{2} {
		t.Error("expected chain 2 to be longest")
	}
}

func TestChainLayerString(t *testing.T) {
	tests := []struct {
		layer    ChainLayer
		expected string
	}{
		{ChainLayerSurface, "Surface"},
		{ChainLayerAnonymous, "Anonymous"},
		{ChainLayer(99), "Unknown"},
	}

	for _, tc := range tests {
		result := ChainLayerString(tc.layer)
		if result != tc.expected {
			t.Errorf("ChainLayerString(%v) = %q, expected %q", tc.layer, result, tc.expected)
		}
	}
}

func TestEchoChainOverlay_DrawWithZoom(t *testing.T) {
	overlay := NewEchoChainOverlay()
	screen := ebiten.NewImage(800, 600)

	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{1},
		Layer:      ChainLayerSurface,
		FormedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		HasShimmer: true,
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, X: 100, Y: 100, Position: 0},
			{NodeID: [32]byte{20}, X: 300, Y: 300, Position: 1},
			{NodeID: [32]byte{30}, X: 500, Y: 200, Position: 2},
			{NodeID: [32]byte{40}, X: 400, Y: 400, Position: 3},
			{NodeID: [32]byte{50}, X: 200, Y: 500, Position: 4},
		},
	})

	// Test various zoom levels.
	for _, zoom := range []float64{0.1, 0.5, 1.0, 2.0, 5.0} {
		overlay.Draw(screen, 0, 0, zoom)
	}
}

func TestEchoChainOverlay_DrawFadingChain(t *testing.T) {
	overlay := NewEchoChainOverlay()
	screen := ebiten.NewImage(800, 600)

	// Chain that is almost expired (should be faded).
	overlay.SetChain(&EchoChainInfo{
		ChainID:    [32]byte{1},
		Layer:      ChainLayerSurface,
		FormedAt:   time.Now().Add(-55 * time.Minute),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		HasShimmer: false,
		Nodes: []*ChainNodeInfo{
			{NodeID: [32]byte{10}, X: 100, Y: 100, Position: 0},
			{NodeID: [32]byte{20}, X: 200, Y: 200, Position: 1},
		},
	})

	overlay.Draw(screen, 0, 0, 1.0)
}
