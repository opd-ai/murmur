// Package overlays provides Anonymous Layer overlay and activity heatmap.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/resonance"
)

func TestNewEchoIndexOverlay(t *testing.T) {
	computer := resonance.NewEchoIndexComputer()
	overlay := NewEchoIndexOverlay(computer)

	if overlay == nil {
		t.Fatal("NewEchoIndexOverlay returned nil")
	}
	if overlay.Computer != computer {
		t.Error("Computer not set correctly")
	}
	if overlay.IsAnonymousLayer {
		t.Error("Should not be anonymous layer")
	}
	if overlay.BadgeRadius != 12.0 {
		t.Errorf("BadgeRadius = %v, want 12.0", overlay.BadgeRadius)
	}
	if !overlay.ShowBadges {
		t.Error("ShowBadges should default to true")
	}
	if !overlay.ShowTint {
		t.Error("ShowTint should default to true")
	}
	if overlay.TintAlpha != 40 {
		t.Errorf("TintAlpha = %d, want 40", overlay.TintAlpha)
	}
}

func TestNewEchoShadowOverlay(t *testing.T) {
	shadow := resonance.NewEchoShadow()
	overlay := NewEchoShadowOverlay(shadow)

	if overlay == nil {
		t.Fatal("NewEchoShadowOverlay returned nil")
	}
	if overlay.Computer != shadow.EchoIndexComputer {
		t.Error("Computer not set correctly")
	}
	if !overlay.IsAnonymousLayer {
		t.Error("Should be anonymous layer")
	}
	if overlay.BadgeRadius != 10.0 {
		t.Errorf("BadgeRadius = %v, want 10.0 (smaller for anonymous)", overlay.BadgeRadius)
	}
	if overlay.TintAlpha != 30 {
		t.Errorf("TintAlpha = %d, want 30 (more subtle for anonymous)", overlay.TintAlpha)
	}
}

func TestSetClusterBoundary(t *testing.T) {
	overlay := NewEchoIndexOverlay(nil)

	tests := []struct {
		name       string
		clusterID  string
		vertices   []float32
		shouldSave bool
	}{
		{
			name:       "valid triangle",
			clusterID:  "cluster1",
			vertices:   []float32{0, 0, 10, 0, 5, 10},
			shouldSave: true,
		},
		{
			name:       "valid quad",
			clusterID:  "cluster2",
			vertices:   []float32{0, 0, 10, 0, 10, 10, 0, 10},
			shouldSave: true,
		},
		{
			name:       "too few vertices (line)",
			clusterID:  "cluster3",
			vertices:   []float32{0, 0, 10, 0},
			shouldSave: false,
		},
		{
			name:       "too few vertices (point)",
			clusterID:  "cluster4",
			vertices:   []float32{0, 0},
			shouldSave: false,
		},
		{
			name:       "empty",
			clusterID:  "cluster5",
			vertices:   []float32{},
			shouldSave: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay.SetClusterBoundary(tt.clusterID, tt.vertices)
			_, exists := overlay.ClusterBoundaries[tt.clusterID]
			if exists != tt.shouldSave {
				t.Errorf("boundary exists = %v, want %v", exists, tt.shouldSave)
			}
		})
	}
}

func TestSetClusterCenter(t *testing.T) {
	overlay := NewEchoIndexOverlay(nil)

	overlay.SetClusterCenter("cluster1", 100, 200)
	center, exists := overlay.ClusterCenters["cluster1"]

	if !exists {
		t.Fatal("Center not saved")
	}
	if center[0] != 100 || center[1] != 200 {
		t.Errorf("Center = %v, want [100, 200]", center)
	}
}

func TestRemoveCluster(t *testing.T) {
	overlay := NewEchoIndexOverlay(nil)

	// Add cluster data.
	overlay.SetClusterBoundary("cluster1", []float32{0, 0, 10, 0, 5, 10})
	overlay.SetClusterCenter("cluster1", 5, 5)

	// Verify it exists.
	if _, ok := overlay.ClusterBoundaries["cluster1"]; !ok {
		t.Fatal("Boundary not saved")
	}
	if _, ok := overlay.ClusterCenters["cluster1"]; !ok {
		t.Fatal("Center not saved")
	}

	// Remove.
	overlay.RemoveCluster("cluster1")

	// Verify removed.
	if _, ok := overlay.ClusterBoundaries["cluster1"]; ok {
		t.Error("Boundary not removed")
	}
	if _, ok := overlay.ClusterCenters["cluster1"]; ok {
		t.Error("Center not removed")
	}
}

func TestClear(t *testing.T) {
	overlay := NewEchoIndexOverlay(nil)

	// Add multiple clusters.
	overlay.SetClusterBoundary("cluster1", []float32{0, 0, 10, 0, 5, 10})
	overlay.SetClusterBoundary("cluster2", []float32{0, 0, 20, 0, 10, 20})
	overlay.SetClusterCenter("cluster1", 5, 5)
	overlay.SetClusterCenter("cluster2", 10, 10)

	// Clear.
	overlay.Clear()

	// Verify empty.
	if len(overlay.ClusterBoundaries) != 0 {
		t.Error("ClusterBoundaries not cleared")
	}
	if len(overlay.ClusterCenters) != 0 {
		t.Error("ClusterCenters not cleared")
	}
}

func TestGetClusterData(t *testing.T) {
	computer := resonance.NewEchoIndexComputer()

	// Add amplification data to create clusters.
	computer.RecordAmplificationSimple("node1", "author1", "wave1", "clusterA", "clusterA") // Intra
	computer.RecordAmplificationSimple("node2", "author2", "wave2", "clusterA", "clusterB") // Extra
	computer.RecordAmplificationSimple("node3", "author3", "wave3", "clusterB", "clusterB") // Intra
	computer.Compute()

	overlay := NewEchoIndexOverlay(computer)
	overlay.SetClusterCenter("clusterA", 100, 100)
	overlay.SetClusterCenter("clusterB", 200, 200)

	data := overlay.GetClusterData()

	if len(data) != 2 {
		t.Fatalf("Got %d clusters, want 2", len(data))
	}

	// Verify cluster data.
	clusterMap := make(map[string]ClusterData)
	for _, d := range data {
		clusterMap[d.ID] = d
	}

	// ClusterA: 1 intra, 1 extra -> Echo Index = 0.5 (neutral)
	if a, ok := clusterMap["clusterA"]; ok {
		if a.EchoIndex != 0.5 {
			t.Errorf("ClusterA EchoIndex = %v, want 0.5", a.EchoIndex)
		}
		if a.Category != resonance.EchoCategoryNeutral {
			t.Errorf("ClusterA Category = %v, want Neutral", a.Category)
		}
		if a.CenterX != 100 || a.CenterY != 100 {
			t.Errorf("ClusterA center = (%v, %v), want (100, 100)", a.CenterX, a.CenterY)
		}
	} else {
		t.Error("ClusterA not found in data")
	}

	// ClusterB: 1 intra, 0 extra -> Echo Index = 1.0 (insular)
	if b, ok := clusterMap["clusterB"]; ok {
		if b.EchoIndex != 1.0 {
			t.Errorf("ClusterB EchoIndex = %v, want 1.0", b.EchoIndex)
		}
		if b.Category != resonance.EchoCategoryInsular {
			t.Errorf("ClusterB Category = %v, want Insular", b.Category)
		}
	} else {
		t.Error("ClusterB not found in data")
	}
}

func TestGetClusterDataNilComputer(t *testing.T) {
	overlay := NewEchoIndexOverlay(nil)
	data := overlay.GetClusterData()

	if data != nil {
		t.Errorf("Expected nil data for nil computer, got %v", data)
	}
}

func TestClusterDataCategory(t *testing.T) {
	// Test category assignments based on Echo Index.
	// Thresholds: Open <= 0.4, Neutral 0.4-0.7, Insular >= 0.7
	tests := []struct {
		echoIndex float64
		want      resonance.EchoCategory
	}{
		{0.0, resonance.EchoCategoryOpen},
		{0.3, resonance.EchoCategoryOpen},
		{0.4, resonance.EchoCategoryOpen}, // Edge case: 0.4 is Open (<=)
		{0.5, resonance.EchoCategoryNeutral},
		{0.7, resonance.EchoCategoryInsular}, // Edge case: 0.7 is Insular (>=)
		{0.9, resonance.EchoCategoryInsular},
		{1.0, resonance.EchoCategoryInsular},
	}

	for _, tt := range tests {
		got := resonance.CategoryFromEchoIndex(tt.echoIndex)
		if got != tt.want {
			t.Errorf("CategoryFromEchoIndex(%v) = %v, want %v", tt.echoIndex, got, tt.want)
		}
	}
}

func TestColorForEchoIndex(t *testing.T) {
	// Test color assignments.
	tests := []struct {
		echoIndex   float64
		description string
	}{
		{0.0, "open (cool colors)"},
		{0.3, "open (cool colors)"},
		{0.5, "neutral (gray)"},
		{0.8, "insular (warm colors)"},
		{1.0, "insular (warm colors)"},
	}

	for _, tt := range tests {
		r, g, b := resonance.ColorForEchoIndex(tt.echoIndex)
		t.Logf("EchoIndex %.1f (%s): RGB(%d, %d, %d)", tt.echoIndex, tt.description, r, g, b)

		// Basic sanity checks.
		if tt.echoIndex >= resonance.EchoIndexEchoChamber {
			// Insular should have high red.
			if r < 200 {
				t.Errorf("Insular color should have high red, got %d", r)
			}
		}
		if tt.echoIndex <= resonance.EchoIndexOutwardOpen {
			// Open should have low red.
			if r > 50 {
				t.Errorf("Open color should have low red, got %d", r)
			}
		}
	}
}

func BenchmarkGetClusterData(b *testing.B) {
	computer := resonance.NewEchoIndexComputer()

	// Add data for 100 clusters.
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			computer.RecordAmplificationSimple(
				"node", "author", "wave",
				string(rune('A'+i)), string(rune('A'+i)),
			)
		}
	}
	computer.Compute()

	overlay := NewEchoIndexOverlay(computer)
	for i := 0; i < 100; i++ {
		overlay.SetClusterCenter(string(rune('A'+i)), float32(i*10), float32(i*10))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = overlay.GetClusterData()
	}
}
