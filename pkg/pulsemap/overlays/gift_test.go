// Package overlays provides Anonymous Layer overlay and activity heatmap.
// Tests for GiftOverlay.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

func TestNewGiftOverlay(t *testing.T) {
	overlay := NewGiftOverlay()
	if overlay == nil {
		t.Fatal("NewGiftOverlay returned nil")
	}
	if overlay.Effects == nil {
		t.Error("Effects map is nil")
	}
	if len(overlay.Effects) != 0 {
		t.Error("Expected empty Effects map")
	}
}

func TestGiftOverlayAddEffect(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "abc123"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)

	if !overlay.HasEffects(nodeID) {
		t.Error("Expected node to have effects after AddEffect")
	}
	if overlay.EffectCount(nodeID) != 1 {
		t.Errorf("Expected 1 effect, got %d", overlay.EffectCount(nodeID))
	}

	// Add another effect
	overlay.AddEffect(nodeID, gifts.EffectOrbitingGeometric, 0.8)
	if overlay.EffectCount(nodeID) != 2 {
		t.Errorf("Expected 2 effects, got %d", overlay.EffectCount(nodeID))
	}
}

func TestGiftOverlayRemoveEffect(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect(nodeID, gifts.EffectFaintHaloRing, 1.0)

	if overlay.EffectCount(nodeID) != 2 {
		t.Errorf("Expected 2 effects, got %d", overlay.EffectCount(nodeID))
	}

	overlay.RemoveEffect(nodeID)

	if overlay.HasEffects(nodeID) {
		t.Error("Expected no effects after RemoveEffect")
	}
	if overlay.EffectCount(nodeID) != 0 {
		t.Errorf("Expected 0 effects, got %d", overlay.EffectCount(nodeID))
	}
}

func TestGiftOverlayRemoveExpiredEffect(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect(nodeID, gifts.EffectFaintHaloRing, 1.0)
	overlay.AddEffect(nodeID, gifts.EffectOrbitingGeometric, 1.0)

	// Remove only the halo effect
	overlay.RemoveExpiredEffect(nodeID, gifts.EffectFaintHaloRing)

	if overlay.EffectCount(nodeID) != 2 {
		t.Errorf("Expected 2 effects after removing one, got %d", overlay.EffectCount(nodeID))
	}

	// Verify correct effects remain
	for _, e := range overlay.Effects[nodeID] {
		if e.Effect == gifts.EffectFaintHaloRing {
			t.Error("FaintHaloRing effect should have been removed")
		}
	}
}

func TestGiftOverlayRemoveExpiredEffectClearsEmpty(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)
	overlay.RemoveExpiredEffect(nodeID, gifts.EffectSoftGlowPulse)

	if overlay.HasEffects(nodeID) {
		t.Error("Expected node to have no effects after removing last one")
	}
	if _, exists := overlay.Effects[nodeID]; exists {
		t.Error("Expected node entry to be deleted from map when empty")
	}
}

func TestGiftOverlayUpdate(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)

	initialPhase := overlay.Effects[nodeID][0].Phase
	overlay.Update(0.5) // 0.5 seconds

	newPhase := overlay.Effects[nodeID][0].Phase
	if newPhase <= initialPhase {
		t.Errorf("Phase should increase after Update, got initial=%f, new=%f", initialPhase, newPhase)
	}
}

func TestGiftOverlayUpdateWrapsPhase(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)
	// Force phase near 2π
	overlay.Effects[nodeID][0].Phase = 6.0

	overlay.Update(1.0) // Should wrap

	newPhase := overlay.Effects[nodeID][0].Phase
	if newPhase > 6.5 {
		t.Errorf("Phase should wrap around 2π, got %f", newPhase)
	}
}

func TestGiftOverlayGetEffectTier(t *testing.T) {
	tests := []struct {
		name     string
		effects  []gifts.EffectType
		expected int
	}{
		{"no effects", nil, 0},
		{"basic only", []gifts.EffectType{gifts.EffectSoftGlowPulse}, 25},
		{"expanded only", []gifts.EffectType{gifts.EffectOrbitingGeometric}, 50},
		{"premium only", []gifts.EffectType{gifts.EffectMultiParticleSystem}, 100},
		{"mixed - returns highest", []gifts.EffectType{
			gifts.EffectSoftGlowPulse,
			gifts.EffectOrbitingGeometric,
			gifts.EffectMultiParticleSystem,
		}, 100},
		{"basic and expanded - returns expanded", []gifts.EffectType{
			gifts.EffectSoftGlowPulse,
			gifts.EffectAuroraColorShift,
		}, 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overlay := NewGiftOverlay()
			nodeID := "testnode"

			for _, effect := range tc.effects {
				overlay.AddEffect(nodeID, effect, 1.0)
			}

			tier := overlay.GetEffectTier(nodeID)
			if tier != tc.expected {
				t.Errorf("Expected tier %d, got %d", tc.expected, tier)
			}
		})
	}
}

func TestGiftOverlayHasEffects(t *testing.T) {
	overlay := NewGiftOverlay()

	if overlay.HasEffects("nonexistent") {
		t.Error("HasEffects should return false for nonexistent node")
	}

	overlay.AddEffect("exists", gifts.EffectSoftGlowPulse, 1.0)
	if !overlay.HasEffects("exists") {
		t.Error("HasEffects should return true for node with effects")
	}
}

func TestGiftOverlayTotalEffectCount(t *testing.T) {
	overlay := NewGiftOverlay()

	if overlay.TotalEffectCount() != 0 {
		t.Error("Empty overlay should have 0 total effects")
	}

	overlay.AddEffect("node1", gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect("node1", gifts.EffectFaintHaloRing, 1.0)
	overlay.AddEffect("node2", gifts.EffectOrbitingGeometric, 1.0)

	if overlay.TotalEffectCount() != 3 {
		t.Errorf("Expected 3 total effects, got %d", overlay.TotalEffectCount())
	}
}

func TestGiftOverlayClear(t *testing.T) {
	overlay := NewGiftOverlay()

	overlay.AddEffect("node1", gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect("node2", gifts.EffectOrbitingGeometric, 1.0)
	overlay.AddEffect("node3", gifts.EffectMultiParticleSystem, 1.0)

	if overlay.TotalEffectCount() != 3 {
		t.Fatal("Setup failed")
	}

	overlay.Clear()

	if overlay.TotalEffectCount() != 0 {
		t.Errorf("Clear should remove all effects, got %d", overlay.TotalEffectCount())
	}
	if len(overlay.Effects) != 0 {
		t.Errorf("Clear should empty map, got %d entries", len(overlay.Effects))
	}
}

func TestGiftOverlayUpdateIntensity(t *testing.T) {
	overlay := NewGiftOverlay()
	nodeID := "node1"

	overlay.AddEffect(nodeID, gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect(nodeID, gifts.EffectFaintHaloRing, 1.0)

	overlay.UpdateIntensity(nodeID, 0.5)

	for _, e := range overlay.Effects[nodeID] {
		if e.Intensity != 0.5 {
			t.Errorf("Expected intensity 0.5, got %f", e.Intensity)
		}
	}
}

func TestGiftOverlayUpdateIntensityNonexistent(t *testing.T) {
	overlay := NewGiftOverlay()

	// Should not panic
	overlay.UpdateIntensity("nonexistent", 0.5)
}

func TestGiftEffectTypes(t *testing.T) {
	// Verify basic effects return tier 25
	basicEffects := []gifts.EffectType{
		gifts.EffectSoftGlowPulse,
		gifts.EffectFaintHaloRing,
		gifts.EffectGentleParticleDrift,
		gifts.EffectShimmerOverlay,
		gifts.EffectWarmthTintShift,
	}

	for _, effect := range basicEffects {
		tier := gifts.RequiredResonance(effect)
		if tier != 25 {
			t.Errorf("Basic effect %d should require tier 25, got %d", effect, tier)
		}
	}

	// Verify expanded effects return tier 50
	expandedEffects := []gifts.EffectType{
		gifts.EffectOrbitingGeometric,
		gifts.EffectAuroraColorShift,
		gifts.EffectCrystallineFracture,
		gifts.EffectEmberTrails,
		gifts.EffectRippleDistortion,
		gifts.EffectStarlightSparkle,
	}

	for _, effect := range expandedEffects {
		tier := gifts.RequiredResonance(effect)
		if tier != 50 {
			t.Errorf("Expanded effect %d should require tier 50, got %d", effect, tier)
		}
	}

	// Verify premium effects return tier 100
	premiumEffects := []gifts.EffectType{
		gifts.EffectMultiParticleSystem,
		gifts.EffectFluidSimulation,
		gifts.EffectGeometricMandala,
		gifts.EffectVoidGravitation,
		gifts.EffectPrismaticRefraction,
		gifts.EffectNebulaeCloud,
		gifts.EffectElectricArc,
		gifts.EffectCrystalGrowth,
		gifts.EffectPhoenixFlame,
		gifts.EffectShadowWraith,
	}

	for _, effect := range premiumEffects {
		tier := gifts.RequiredResonance(effect)
		if tier != 100 {
			t.Errorf("Premium effect %d should require tier 100, got %d", effect, tier)
		}
	}
}

func TestGiftOverlayMultipleNodesIndependent(t *testing.T) {
	overlay := NewGiftOverlay()

	overlay.AddEffect("node1", gifts.EffectSoftGlowPulse, 1.0)
	overlay.AddEffect("node2", gifts.EffectOrbitingGeometric, 0.8)
	overlay.AddEffect("node3", gifts.EffectMultiParticleSystem, 0.6)

	// Remove effects from node2
	overlay.RemoveEffect("node2")

	// Verify node1 and node3 still have effects
	if !overlay.HasEffects("node1") {
		t.Error("node1 should still have effects")
	}
	if overlay.HasEffects("node2") {
		t.Error("node2 should not have effects")
	}
	if !overlay.HasEffects("node3") {
		t.Error("node3 should still have effects")
	}

	// Verify intensities are independent
	overlay.UpdateIntensity("node1", 0.3)
	if overlay.Effects["node3"][0].Intensity != 0.6 {
		t.Error("node3 intensity should not be affected by node1 update")
	}
}
