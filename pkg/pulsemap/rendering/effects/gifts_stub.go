// Package effects provides shader-based visual effects for the Pulse Map.
// This is the noebiten stub for GiftRenderer.
//
//go:build test
// +build test

package effects

import (
	"image/color"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

// GiftEffect represents a visual effect to render for a Phantom Gift.
type GiftEffect struct {
	EffectType  gifts.EffectType
	X, Y        float32
	Size        float32
	TimeSeconds float32
	Active      bool
}

// GiftEffectConfig maps effect types to rendering parameters.
type GiftEffectConfig struct {
	Color          color.RGBA
	Intensity      float32
	PulseRate      float32
	ParticleCount  int
	HasGlow        bool
	HasRipple      bool
	HasParticles   bool
	SecondaryColor color.RGBA
}

// GiftRenderer manages rendering of Phantom Gift effects on nodes.
type GiftRenderer struct {
	activeGifts map[string][]GiftEffect
	timeAccum   float32
	initialized bool
}

// NewGiftRenderer creates a new gift effect renderer.
func NewGiftRenderer() *GiftRenderer {
	return &GiftRenderer{
		activeGifts: make(map[string][]GiftEffect),
	}
}

// Initialize loads shaders for gift effects.
func (r *GiftRenderer) Initialize() error {
	r.initialized = true
	return nil
}

// SetActiveGifts updates the active gifts for a recipient.
func (r *GiftRenderer) SetActiveGifts(recipientKeyHex string, gifts []*gifts.Gift) {
	if gifts == nil {
		delete(r.activeGifts, recipientKeyHex)
		return
	}
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
	if r.timeAccum > 3600 {
		r.timeAccum -= 3600
	}
}

// GetTime returns the current animation time.
func (r *GiftRenderer) GetTime() float32 {
	return r.timeAccum
}

// IsInitialized returns true if the renderer is ready to draw.
func (r *GiftRenderer) IsInitialized() bool {
	return r.initialized
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

// GetEffectConfig returns the rendering configuration for a gift effect type.
func GetEffectConfig(effectType gifts.EffectType) GiftEffectConfig {
	return GiftEffectConfig{
		Color:     color.RGBA{200, 200, 200, 150},
		Intensity: 0.5,
		PulseRate: 0.5,
		HasGlow:   true,
	}
}
