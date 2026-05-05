// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This stub file provides non-Ebitengine stubs for batch rendering testing.

//go:build test
// +build test

package rendering

import (
	"image/color"
)

// BatchRenderer stub for testing without Ebitengine.
type BatchRenderer struct {
	EdgeCount     int
	NodeCount     int
	ParticleCount int
	TrailCount    int
}

// NewBatchRenderer creates a stub BatchRenderer.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{}
}

// Clear resets counters.
func (b *BatchRenderer) Clear() {
	b.EdgeCount = 0
	b.NodeCount = 0
	b.ParticleCount = 0
	b.TrailCount = 0
}

// AddEdge increments edge counter.
func (b *BatchRenderer) AddEdge(x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel) {
	b.EdgeCount++
}

// AddNode increments node counter.
func (b *BatchRenderer) AddNode(x, y, radius float32, style NodeStyle) {
	b.NodeCount++
}

// AddParticle increments particle counter.
func (b *BatchRenderer) AddParticle(x, y, radius float32, particleColor color.RGBA) {
	b.ParticleCount++
}

// AddTrail increments trail counter.
func (b *BatchRenderer) AddTrail(x1, y1, x2, y2 float32, baseAlpha float64, hasComment bool, time float64) {
	b.TrailCount++
}

// Flush is a no-op in test stub.
func (b *BatchRenderer) Flush(dst interface{}) {
	// No-op for tests - accepts any dst including nil
}
