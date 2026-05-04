// Package effects provides shader-based visual effects for the Pulse Map.
// This file implements Wave propagation ripple animations per ROADMAP.md line 620.
//

//go:build !test
// +build !test

package effects

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Ripple represents an expanding wave propagation animation.
// Per DESIGN_DOCUMENT.md, ripples expand outward from publishing nodes,
// travel along edges, and create interference patterns.
type Ripple struct {
	// OriginX, OriginY are the screen coordinates where the ripple started.
	OriginX, OriginY float32

	// StartTime is when the ripple animation began.
	StartTime time.Time

	// Color is the RGBA color of the ripple (matches publishing node's base color).
	Color [4]float32

	// MaxRadius is the maximum expansion radius before fading out (default 500 pixels).
	MaxRadius float32

	// Speed is the expansion speed in pixels per second (default 200).
	Speed float32

	// Width is the ring thickness in pixels (default 10).
	Width float32
}

// RippleManager tracks and updates active Wave propagation ripples.
type RippleManager struct {
	mu      sync.RWMutex
	ripples []*Ripple
	shaders *Shaders
}

// NewRippleManager creates a new ripple animation manager.
func NewRippleManager(shaders *Shaders) *RippleManager {
	return &RippleManager{
		ripples: make([]*Ripple, 0, 32), // Pre-allocate for 32 concurrent ripples
		shaders: shaders,
	}
}

// AddRipple registers a new ripple animation at the given screen coordinates.
// color should be [R, G, B, A] with values in [0, 1].
func (rm *RippleManager) AddRipple(x, y float32, color [4]float32) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	ripple := &Ripple{
		OriginX:   x,
		OriginY:   y,
		StartTime: time.Now(),
		Color:     color,
		MaxRadius: 500.0,
		Speed:     200.0,
		Width:     10.0,
	}

	rm.ripples = append(rm.ripples, ripple)
}

// Update advances all active ripples and removes expired ones.
// Call this once per frame in the game Update() loop.
func (rm *RippleManager) Update() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	active := rm.ripples[:0] // Reuse slice backing array

	for _, r := range rm.ripples {
		elapsed := now.Sub(r.StartTime).Seconds()
		radius := float32(elapsed) * r.Speed

		// Keep ripple if it hasn't exceeded max radius
		if radius <= r.MaxRadius {
			active = append(active, r)
		}
	}

	rm.ripples = active
}

// Draw renders all active ripples to the destination image.
// Call this during the rendering pipeline after drawing nodes/edges.
func (rm *RippleManager) Draw(dst *ebiten.Image) {
	if rm.shaders == nil || rm.shaders.Ripple == nil {
		return
	}

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	now := time.Now()
	for _, r := range rm.ripples {
		elapsed := float32(now.Sub(r.StartTime).Seconds())
		radius := elapsed * r.Speed

		uniforms := RippleUniforms{
			Time:         elapsed,
			RippleOrigin: [2]float32{r.OriginX, r.OriginY},
			RippleRadius: radius,
			RippleColor:  r.Color,
			RippleWidth:  r.Width,
		}

		rm.shaders.DrawRipple(dst, uniforms)
	}
}

// Count returns the number of active ripples (for debugging/diagnostics).
func (rm *RippleManager) Count() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.ripples)
}

// Clear removes all active ripples (useful for testing or scene transitions).
func (rm *RippleManager) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.ripples = rm.ripples[:0]
}
