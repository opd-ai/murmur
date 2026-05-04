// Package effects provides layer compositing for translucency blending (stub).

//go:build test
// +build test

package effects

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// LayerCompositor handles translucency blending of Surface and Anonymous layers (stub).
type LayerCompositor struct {
	surfaceBuffer   *ebiten.Image
	anonymousBuffer *ebiten.Image
	initialized     bool
	cachedWidth     int
	cachedHeight    int
}

// NewLayerCompositor creates a new layer compositor (stub).
func NewLayerCompositor(width, height int) (*LayerCompositor, error) {
	return &LayerCompositor{
		surfaceBuffer:   ebiten.NewImage(width, height),
		anonymousBuffer: ebiten.NewImage(width, height),
		initialized:     true,
		cachedWidth:     width,
		cachedHeight:    height,
	}, nil
}

// Resize updates layer buffers to new dimensions (stub).
func (c *LayerCompositor) Resize(width, height int) {
	if c.cachedWidth == width && c.cachedHeight == height {
		return
	}
	if c.surfaceBuffer != nil {
		c.surfaceBuffer.Deallocate()
	}
	if c.anonymousBuffer != nil {
		c.anonymousBuffer.Deallocate()
	}
	c.surfaceBuffer = ebiten.NewImage(width, height)
	c.anonymousBuffer = ebiten.NewImage(width, height)
	c.cachedWidth = width
	c.cachedHeight = height
}

// GetSurfaceBuffer returns the Surface Layer render target (stub).
func (c *LayerCompositor) GetSurfaceBuffer() *ebiten.Image {
	return c.surfaceBuffer
}

// GetAnonymousBuffer returns the Anonymous Layer render target (stub).
func (c *LayerCompositor) GetAnonymousBuffer() *ebiten.Image {
	return c.anonymousBuffer
}

// ClearBuffers clears both layer buffers (stub).
func (c *LayerCompositor) ClearBuffers() {
	if c.surfaceBuffer != nil {
		c.surfaceBuffer.Clear()
	}
	if c.anonymousBuffer != nil {
		c.anonymousBuffer.Clear()
	}
}

// Composite blends layers (no-op in test build).
func (c *LayerCompositor) Composite(dst *ebiten.Image, surfaceOpacity, anonymousOpacity float32) {
	// No-op in test build
}

// Dispose releases GPU resources (stub).
func (c *LayerCompositor) Dispose() {
	c.initialized = false
}
