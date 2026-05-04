// Package effects provides shader-based visual effects for the Pulse Map.
// This is the stub implementation for test builds.
//

//go:build test
// +build test

package effects

// Ripple represents an expanding wave propagation animation (stub).
type Ripple struct {
	OriginX, OriginY float32
	Color            [4]float32
	MaxRadius        float32
	Speed            float32
	Width            float32
}

// RippleManager tracks and updates active Wave propagation ripples (stub).
type RippleManager struct{}

// NewRippleManager creates a new ripple animation manager (stub).
func NewRippleManager(shaders *Shaders) *RippleManager {
	return &RippleManager{}
}

// AddRipple registers a new ripple animation (stub).
func (rm *RippleManager) AddRipple(x, y float32, color [4]float32) {}

// Update advances all active ripples (stub).
func (rm *RippleManager) Update() {}

// Draw renders all active ripples (stub).
func (rm *RippleManager) Draw(dst interface{}) {}

// Count returns the number of active ripples (stub).
func (rm *RippleManager) Count() int { return 0 }

// Clear removes all active ripples (stub).
func (rm *RippleManager) Clear() {}
