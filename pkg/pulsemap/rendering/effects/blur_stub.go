// Package effects provides blur effects for depth and atmospheric rendering (stub).

//go:build test
// +build test

package effects

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// BlurEffect provides Gaussian blur rendering for depth effects (stub).
type BlurEffect struct {
	initialized bool
}

// NewBlurEffect creates a new blur effect renderer (stub).
func NewBlurEffect() (*BlurEffect, error) {
	return &BlurEffect{initialized: true}, nil
}

// Apply applies Gaussian blur (no-op in test build).
func (b *BlurEffect) Apply(dst, src *ebiten.Image, radius float32) {
	// No-op in test build
}

// ApplySinglePass applies a single-pass blur (no-op in test build).
func (b *BlurEffect) ApplySinglePass(dst, src *ebiten.Image, radius float32) {
	// No-op in test build
}

// Dispose releases GPU resources (no-op in test build).
func (b *BlurEffect) Dispose() {
	b.initialized = false
}
