// Package effects provides shader-based visual effects for the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, Phantom Gifts create visual effects on recipient nodes.
//
//go:build !noebiten
// +build !noebiten

package effects

import (
	"image/color"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// GiftEffect represents a visual effect to render for a Phantom Gift.
type GiftEffect struct {
	EffectType  mechanics.EffectType
	X, Y        float32 // Screen position
	Size        float32 // Effect size
	TimeSeconds float32 // Animation time
	Active      bool    // Whether the gift is active
}

// GiftEffectConfig maps effect types to rendering parameters.
type GiftEffectConfig struct {
	Color          color.RGBA
	Intensity      float32
	PulseRate      float32 // Pulse frequency in Hz
	ParticleCount  int     // For particle effects
	HasGlow        bool
	HasRipple      bool
	HasParticles   bool
	SecondaryColor color.RGBA
}

// defaultEffectConfig provides a fallback for unknown effect types.
var defaultEffectConfig = GiftEffectConfig{
	Color:     color.RGBA{200, 200, 200, 150},
	Intensity: 0.5,
	PulseRate: 0.5,
	HasGlow:   true,
}

// effectConfigs maps effect types to their rendering configurations.
// Per ANONYMOUS_GAME_MECHANICS.md, effects are tiered by Resonance level.
var effectConfigs = map[mechanics.EffectType]GiftEffectConfig{
	// Basic effects (Resonance 25+)
	mechanics.EffectSoftGlowPulse: {
		Color: color.RGBA{255, 220, 180, 180}, Intensity: 0.6, PulseRate: 0.5, HasGlow: true,
	},
	mechanics.EffectFaintHaloRing: {
		Color: color.RGBA{200, 200, 255, 150}, Intensity: 0.4, PulseRate: 0.3, HasGlow: true, HasRipple: true,
	},
	mechanics.EffectGentleParticleDrift: {
		Color: color.RGBA{255, 255, 200, 200}, Intensity: 0.5, PulseRate: 0.2, ParticleCount: 5, HasParticles: true,
	},
	mechanics.EffectShimmerOverlay: {
		Color: color.RGBA{255, 255, 255, 100}, Intensity: 0.3, PulseRate: 2.0, HasGlow: true,
	},
	mechanics.EffectWarmthTintShift: {
		Color: color.RGBA{255, 180, 100, 150}, SecondaryColor: color.RGBA{255, 200, 150, 150}, Intensity: 0.5, PulseRate: 0.4, HasGlow: true,
	},
	// Expanded effects (Resonance 50+)
	mechanics.EffectOrbitingGeometric: {
		Color: color.RGBA{100, 200, 255, 200}, Intensity: 0.7, PulseRate: 0.5, ParticleCount: 8, HasParticles: true, HasGlow: true,
	},
	mechanics.EffectAuroraColorShift: {
		Color: color.RGBA{100, 255, 200, 180}, SecondaryColor: color.RGBA{200, 100, 255, 180}, Intensity: 0.8, PulseRate: 0.3, HasGlow: true,
	},
	mechanics.EffectCrystallineFracture: {
		Color: color.RGBA{200, 220, 255, 220}, Intensity: 0.9, PulseRate: 0.2, ParticleCount: 12, HasParticles: true, HasGlow: true,
	},
	mechanics.EffectEmberTrails: {
		Color: color.RGBA{255, 150, 50, 220}, SecondaryColor: color.RGBA{255, 100, 0, 180}, Intensity: 0.85, PulseRate: 1.0, ParticleCount: 10, HasParticles: true, HasGlow: true,
	},
	mechanics.EffectRippleDistortion: {
		Color: color.RGBA{100, 150, 255, 150}, Intensity: 0.7, PulseRate: 0.6, HasRipple: true, HasGlow: true,
	},
	// Premium effects (Resonance 100+)
	mechanics.EffectMultiParticleSystem: {
		Color: color.RGBA{255, 200, 100, 255}, Intensity: 1.0, PulseRate: 0.4, ParticleCount: 20, HasParticles: true, HasGlow: true,
	},
	mechanics.EffectVoidGravitation: {
		Color: color.RGBA{50, 0, 100, 200}, Intensity: 1.0, PulseRate: 0.5, ParticleCount: 15, HasParticles: true, HasGlow: true, HasRipple: true,
	},
	mechanics.EffectPhoenixFlame: {
		Color: color.RGBA{255, 150, 0, 255}, SecondaryColor: color.RGBA{255, 50, 0, 220}, Intensity: 1.0, PulseRate: 0.8, ParticleCount: 25, HasParticles: true, HasGlow: true,
	},
	mechanics.EffectShadowWraith: {
		Color: color.RGBA{100, 50, 150, 200}, SecondaryColor: color.RGBA{50, 0, 100, 180}, Intensity: 1.0, PulseRate: 0.3, ParticleCount: 20, HasParticles: true, HasGlow: true,
	},
}

// GetEffectConfig returns the rendering configuration for a gift effect type.
func GetEffectConfig(effectType mechanics.EffectType) GiftEffectConfig {
	if cfg, ok := effectConfigs[effectType]; ok {
		return cfg
	}
	return defaultEffectConfig
}

// GiftRenderer manages rendering of Phantom Gift effects on nodes.
type GiftRenderer struct {
	shaders      *Shaders
	activeGifts  map[string][]GiftEffect // By recipient key (hex)
	timeAccum    float32
	initialized  bool
	shaderFailed bool // Track if shader loading failed
}

// NewGiftRenderer creates a new gift effect renderer.
func NewGiftRenderer() *GiftRenderer {
	return &GiftRenderer{
		activeGifts: make(map[string][]GiftEffect),
	}
}

// Initialize loads shaders for gift effects.
func (r *GiftRenderer) Initialize() error {
	if r.initialized {
		return nil
	}

	shaders, err := LoadShaders()
	if err != nil {
		r.shaderFailed = true
		// Don't fail - we can render without shaders using fallback
		return nil
	}

	r.shaders = shaders
	r.initialized = true
	return nil
}

// SetActiveGifts updates the active gifts for a recipient.
func (r *GiftRenderer) SetActiveGifts(recipientKeyHex string, gifts []*mechanics.Gift) {
	effects := make([]GiftEffect, 0, len(gifts))
	for _, gift := range gifts {
		if gift != nil && !gift.IsExpired() {
			effects = append(effects, GiftEffect{
				EffectType: gift.Effect,
				Active:     true,
			})
		}
	}
	r.activeGifts[recipientKeyHex] = effects
}

// GetGiftsForNode returns the active gift effects for a node.
func (r *GiftRenderer) GetGiftsForNode(recipientKeyHex string) []GiftEffect {
	return r.activeGifts[recipientKeyHex]
}

// HasActiveGifts returns true if the node has any active gift effects.
func (r *GiftRenderer) HasActiveGifts(recipientKeyHex string) bool {
	gifts := r.activeGifts[recipientKeyHex]
	return len(gifts) > 0
}

// Update advances the animation time for all effects.
func (r *GiftRenderer) Update(deltaSeconds float32) {
	r.timeAccum += deltaSeconds
	// Keep time bounded to prevent float precision issues
	if r.timeAccum > 3600 {
		r.timeAccum -= 3600
	}
}

// GetTime returns the current animation time.
func (r *GiftRenderer) GetTime() float32 {
	return r.timeAccum
}

// GetShaders returns the loaded shaders, or nil if not available.
func (r *GiftRenderer) GetShaders() *Shaders {
	return r.shaders
}

// IsInitialized returns true if the renderer is ready to draw.
func (r *GiftRenderer) IsInitialized() bool {
	return r.initialized || r.shaderFailed
}

// ClearGifts removes all active gifts for a recipient.
func (r *GiftRenderer) ClearGifts(recipientKeyHex string) {
	delete(r.activeGifts, recipientKeyHex)
}

// ClearAllGifts removes all active gifts.
func (r *GiftRenderer) ClearAllGifts() {
	r.activeGifts = make(map[string][]GiftEffect)
}

// ActiveGiftCount returns the total number of active gift effects.
func (r *GiftRenderer) ActiveGiftCount() int {
	count := 0
	for _, effects := range r.activeGifts {
		count += len(effects)
	}
	return count
}
