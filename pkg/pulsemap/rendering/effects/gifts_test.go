// Package effects tests for gift effect rendering integration.
// Per ROADMAP.md Priority 8 Validation: Phantom Gift from Resonance 25+ Specter
// appears on recipient's Surface node.
//
// These tests require a display/GPU and must be run with the ebitentest build tag:
//
//	go test -tags=ebitentest ./pkg/pulsemap/rendering/effects/...
//
//go:build ebitentest
// +build ebitentest

package effects

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// keyToHex converts a public key to a hex string for map keys.
func keyToHex(key []byte) string {
	if len(key) == 0 {
		return ""
	}
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(key)*2)
	for i, v := range key {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

// TestPhantomGiftVisibility validates that Phantom Gifts from Resonance 25+ Specters
// appear on recipient Surface nodes per ROADMAP.md Priority 8 validation criteria.
func TestPhantomGiftVisibility(t *testing.T) {
	// Create a gift store
	store := mechanics.NewGiftStore()

	// Create Specter identity (sender) with Resonance 25+
	var senderKey [32]byte
	_, err := rand.Read(senderKey[:])
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}

	// Create Surface identity (recipient) - Ed25519 keypair
	recipientPub, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate recipient keys: %v", err)
	}

	// Specter has Resonance 25 (Shade milestone - minimum for gifts)
	specterResonance := 25

	// Create a Phantom Gift with basic effect (Resonance 25+ required)
	gift, err := store.CreateGift(
		senderKey,
		recipientPub,
		mechanics.EffectSoftGlowPulse,
		specterResonance,
		recipientPriv, // Using recipient's key for signing in test
	)
	if err != nil {
		t.Fatalf("Failed to create gift: %v", err)
	}

	// Verify gift was created
	if gift == nil {
		t.Fatal("Gift should not be nil")
	}

	// Initialize gift renderer
	renderer := NewGiftRenderer()

	// Set active gifts for the recipient
	recipientKeyHex := keyToHex(recipientPub)
	gifts := store.GetGiftsForRecipient(recipientPub)
	renderer.SetActiveGifts(recipientKeyHex, gifts)

	// VALIDATION: Gift appears on recipient's node
	if !renderer.HasActiveGifts(recipientKeyHex) {
		t.Error("ROADMAP validation failed: Phantom Gift should appear on recipient's Surface node")
	}

	// Verify the effect configuration is correct for basic tier
	effects := renderer.GetGiftsForNode(recipientKeyHex)
	if len(effects) != 1 {
		t.Errorf("Expected 1 effect, got %d", len(effects))
	}

	// Verify effect type matches what was created
	if effects[0].EffectType != mechanics.EffectSoftGlowPulse {
		t.Errorf("Expected EffectSoftGlowPulse, got %v", effects[0].EffectType)
	}

	// Verify effect configuration is valid for rendering
	config := GetEffectConfig(effects[0].EffectType)
	if !config.HasGlow {
		t.Error("SoftGlowPulse effect should have glow")
	}
	if config.Intensity == 0 {
		t.Error("Effect should have non-zero intensity")
	}

	t.Logf("ROADMAP validation passed: Phantom Gift from Resonance %d Specter appears on recipient's Surface node", specterResonance)
}

// TestResonanceTieredEffects validates that different Resonance levels unlock
// different gift effects per ANONYMOUS_GAME_MECHANICS.md.
func TestResonanceTieredEffects(t *testing.T) {
	tests := []struct {
		name        string
		resonance   int
		effectType  mechanics.EffectType
		shouldWork  bool
		description string
	}{
		{
			name:        "Basic effect at Resonance 25",
			resonance:   25,
			effectType:  mechanics.EffectSoftGlowPulse,
			shouldWork:  true,
			description: "Shade milestone unlocks basic effects",
		},
		{
			name:        "Basic effect below threshold",
			resonance:   24,
			effectType:  mechanics.EffectSoftGlowPulse,
			shouldWork:  false,
			description: "Below Shade milestone cannot send gifts",
		},
		{
			name:        "Expanded effect at Resonance 50",
			resonance:   50,
			effectType:  mechanics.EffectOrbitingGeometric,
			shouldWork:  true,
			description: "Wraith milestone unlocks expanded effects",
		},
		{
			name:        "Expanded effect below threshold",
			resonance:   49,
			effectType:  mechanics.EffectOrbitingGeometric,
			shouldWork:  false,
			description: "Below Wraith cannot use expanded effects",
		},
		{
			name:        "Premium effect at Resonance 100",
			resonance:   100,
			effectType:  mechanics.EffectPhoenixFlame,
			shouldWork:  true,
			description: "Phantom milestone unlocks premium effects",
		},
		{
			name:        "Premium effect below threshold",
			resonance:   99,
			effectType:  mechanics.EffectPhoenixFlame,
			shouldWork:  false,
			description: "Below Phantom cannot use premium effects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mechanics.NewGiftStore()

			var senderKey [32]byte
			_, _ = rand.Read(senderKey[:])

			recipientPub, _, _ := ed25519.GenerateKey(rand.Reader)

			_, err := store.CreateGift(
				senderKey,
				recipientPub,
				tt.effectType,
				tt.resonance,
				nil, // No signing key for this test
			)

			if tt.shouldWork && err != nil {
				t.Errorf("Expected gift creation to succeed: %v", err)
			}
			if !tt.shouldWork && err == nil {
				t.Errorf("Expected gift creation to fail for: %s", tt.description)
			}
		})
	}
}

// TestGiftExpiration validates that expired gifts do not appear on nodes.
func TestGiftExpiration(t *testing.T) {
	store := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()

	var senderKey [32]byte
	_, _ = rand.Read(senderKey[:])

	recipientPub, _, _ := ed25519.GenerateKey(rand.Reader)
	recipientKeyHex := keyToHex(recipientPub)

	// Create a gift
	gift, err := store.CreateGift(
		senderKey,
		recipientPub,
		mechanics.EffectSoftGlowPulse,
		25, // Resonance 25
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create gift: %v", err)
	}

	// Initially the gift should be visible
	gifts := store.GetGiftsForRecipient(recipientPub)
	renderer.SetActiveGifts(recipientKeyHex, gifts)

	if !renderer.HasActiveGifts(recipientKeyHex) {
		t.Error("Fresh gift should be visible")
	}

	// Manually expire the gift for testing
	gift.ExpiresAt = time.Now().Add(-time.Hour)

	// Refresh active gifts - expired gift should be filtered
	gifts = store.GetGiftsForRecipient(recipientPub)
	renderer.SetActiveGifts(recipientKeyHex, gifts)

	if renderer.HasActiveGifts(recipientKeyHex) {
		t.Error("Expired gift should not be visible")
	}
}

// TestEffectConfigurations validates that all effect types have valid configurations.
func TestEffectConfigurations(t *testing.T) {
	effectTypes := []mechanics.EffectType{
		// Basic effects
		mechanics.EffectSoftGlowPulse,
		mechanics.EffectFaintHaloRing,
		mechanics.EffectGentleParticleDrift,
		mechanics.EffectShimmerOverlay,
		mechanics.EffectWarmthTintShift,
		// Expanded effects
		mechanics.EffectOrbitingGeometric,
		mechanics.EffectAuroraColorShift,
		mechanics.EffectCrystallineFracture,
		mechanics.EffectEmberTrails,
		mechanics.EffectRippleDistortion,
		// Premium effects
		mechanics.EffectMultiParticleSystem,
		mechanics.EffectVoidGravitation,
		mechanics.EffectPhoenixFlame,
		mechanics.EffectShadowWraith,
	}

	for _, et := range effectTypes {
		config := GetEffectConfig(et)

		// Every effect should have some rendering capability
		hasRendering := config.HasGlow || config.HasRipple || config.HasParticles
		if !hasRendering {
			t.Errorf("Effect %d has no rendering capability", et)
		}

		// Every effect should have positive intensity
		if config.Intensity <= 0 {
			t.Errorf("Effect %d has invalid intensity: %f", et, config.Intensity)
		}

		// Alpha should be non-zero for visibility
		if config.Color.A == 0 {
			t.Errorf("Effect %d has zero alpha (invisible)", et)
		}
	}
}

// TestGiftRendererLifecycle validates the gift renderer state management.
func TestGiftRendererLifecycle(t *testing.T) {
	renderer := NewGiftRenderer()

	// Initial state
	if renderer.ActiveGiftCount() != 0 {
		t.Error("New renderer should have no active gifts")
	}

	// Add gifts for two recipients
	renderer.SetActiveGifts("recipient1", []*mechanics.Gift{
		{Effect: mechanics.EffectSoftGlowPulse, ExpiresAt: time.Now().Add(time.Hour)},
		{Effect: mechanics.EffectFaintHaloRing, ExpiresAt: time.Now().Add(time.Hour)},
	})
	renderer.SetActiveGifts("recipient2", []*mechanics.Gift{
		{Effect: mechanics.EffectEmberTrails, ExpiresAt: time.Now().Add(time.Hour)},
	})

	if renderer.ActiveGiftCount() != 3 {
		t.Errorf("Expected 3 active gifts, got %d", renderer.ActiveGiftCount())
	}

	// Clear one recipient
	renderer.ClearGifts("recipient1")
	if renderer.ActiveGiftCount() != 1 {
		t.Errorf("Expected 1 active gift after clear, got %d", renderer.ActiveGiftCount())
	}

	// Clear all
	renderer.ClearAllGifts()
	if renderer.ActiveGiftCount() != 0 {
		t.Error("Expected 0 gifts after clear all")
	}
}

// TestGiftRendererAnimation validates animation time tracking.
func TestGiftRendererAnimation(t *testing.T) {
	renderer := NewGiftRenderer()

	initial := renderer.GetTime()
	if initial != 0 {
		t.Errorf("Initial time should be 0, got %f", initial)
	}

	// Simulate 1 second of updates
	for i := 0; i < 60; i++ {
		renderer.Update(1.0 / 60.0)
	}

	elapsed := renderer.GetTime()
	if elapsed < 0.99 || elapsed > 1.01 {
		t.Errorf("Expected ~1.0 seconds elapsed, got %f", elapsed)
	}
}
