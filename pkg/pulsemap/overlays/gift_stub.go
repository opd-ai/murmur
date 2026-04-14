// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file provides a stub implementation of GiftOverlay for noebiten builds.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// GiftEffect represents an active Phantom Gift effect on a node.
// Per ANONYMOUS_GAME_MECHANICS.md, gifts last 7 days with animated effects.
type GiftEffect struct {
	Effect    mechanics.EffectType // Type of visual effect
	Intensity float32              // 0-1, fades as gift nears expiration
	Phase     float32              // Animation phase (0 to 2π)
}

// GiftOverlay manages Phantom Gift visualization on the Pulse Map.
// Per ROADMAP.md line 519, shows animated cosmetic effects on recipient nodes.
type GiftOverlay struct {
	Effects map[string][]GiftEffect // Keyed by recipient node ID (hex pubkey)
}

// NewGiftOverlay creates a new gift overlay manager.
func NewGiftOverlay() *GiftOverlay {
	return &GiftOverlay{
		Effects: make(map[string][]GiftEffect),
	}
}

// AddEffect registers a gift effect for a recipient node.
func (o *GiftOverlay) AddEffect(nodeID string, effect mechanics.EffectType, intensity float32) {
	o.Effects[nodeID] = append(o.Effects[nodeID], GiftEffect{
		Effect:    effect,
		Intensity: intensity,
		Phase:     0,
	})
}

// RemoveEffect removes all effects for a node (e.g., when gift expires).
func (o *GiftOverlay) RemoveEffect(nodeID string) {
	delete(o.Effects, nodeID)
}

// RemoveExpiredEffect removes a specific effect by type from a node.
func (o *GiftOverlay) RemoveExpiredEffect(nodeID string, effect mechanics.EffectType) {
	effects := o.Effects[nodeID]
	filtered := effects[:0]
	for _, e := range effects {
		if e.Effect != effect {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		delete(o.Effects, nodeID)
	} else {
		o.Effects[nodeID] = filtered
	}
}

// Update advances animation phases for all effects.
// dt is delta time in seconds.
func (o *GiftOverlay) Update(dt float32) {
	const twoPi = 6.283185307
	for nodeID, effects := range o.Effects {
		for i := range effects {
			effects[i].Phase += dt * 2
			if effects[i].Phase > twoPi {
				effects[i].Phase -= twoPi
			}
		}
		o.Effects[nodeID] = effects
	}
}

// HasEffects returns true if the node has any active gift effects.
func (o *GiftOverlay) HasEffects(nodeID string) bool {
	effects, ok := o.Effects[nodeID]
	return ok && len(effects) > 0
}

// GetEffectTier returns the highest tier effect active on a node.
// Returns 0 if no effects, otherwise 25 (Basic), 50 (Expanded), or 100 (Premium).
func (o *GiftOverlay) GetEffectTier(nodeID string) int {
	effects, ok := o.Effects[nodeID]
	if !ok || len(effects) == 0 {
		return 0
	}
	maxTier := 0
	for _, e := range effects {
		tier := mechanics.RequiredResonance(e.Effect)
		if tier > maxTier {
			maxTier = tier
		}
	}
	return maxTier
}

// EffectCount returns the number of active effects for a node.
func (o *GiftOverlay) EffectCount(nodeID string) int {
	return len(o.Effects[nodeID])
}

// TotalEffectCount returns the total number of active effects across all nodes.
func (o *GiftOverlay) TotalEffectCount() int {
	total := 0
	for _, effects := range o.Effects {
		total += len(effects)
	}
	return total
}

// Clear removes all gift effects.
func (o *GiftOverlay) Clear() {
	o.Effects = make(map[string][]GiftEffect)
}

// UpdateIntensity updates the intensity of all effects for a node.
// Used to fade effects as gifts near expiration.
func (o *GiftOverlay) UpdateIntensity(nodeID string, intensity float32) {
	effects, ok := o.Effects[nodeID]
	if !ok {
		return
	}
	for i := range effects {
		effects[i].Intensity = intensity
	}
	o.Effects[nodeID] = effects
}
