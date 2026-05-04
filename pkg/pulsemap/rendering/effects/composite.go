// Package effects provides layer compositing for translucency blending.
// Per PULSE_MAP.md §5.3, layers are rendered to separate framebuffers and composited.

//go:build !test
// +build !test

package effects

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed composite.kage
var compositeShaderSrc []byte

// LayerCompositor handles translucency blending of Surface and Anonymous layers.
// Per PULSE_MAP.md: "Each layer is rendered to a separate framebuffer and composited
// with appropriate blend modes."
type LayerCompositor struct {
	shader          *ebiten.Shader
	surfaceBuffer   *ebiten.Image
	anonymousBuffer *ebiten.Image
	initialized     bool
	cachedWidth     int
	cachedHeight    int
}

// NewLayerCompositor creates a new layer compositor.
func NewLayerCompositor(width, height int) (*LayerCompositor, error) {
	shader, err := ebiten.NewShader(compositeShaderSrc)
	if err != nil {
		return nil, err
	}

	return &LayerCompositor{
		shader:          shader,
		surfaceBuffer:   ebiten.NewImage(width, height),
		anonymousBuffer: ebiten.NewImage(width, height),
		initialized:     true,
		cachedWidth:     width,
		cachedHeight:    height,
	}, nil
}

// Resize updates layer buffers to new dimensions.
func (c *LayerCompositor) Resize(width, height int) {
	if c.cachedWidth == width && c.cachedHeight == height {
		return
	}

	// Deallocate old buffers
	if c.surfaceBuffer != nil {
		c.surfaceBuffer.Deallocate()
	}
	if c.anonymousBuffer != nil {
		c.anonymousBuffer.Deallocate()
	}

	// Allocate new buffers
	c.surfaceBuffer = ebiten.NewImage(width, height)
	c.anonymousBuffer = ebiten.NewImage(width, height)
	c.cachedWidth = width
	c.cachedHeight = height
}

// GetSurfaceBuffer returns the Surface Layer render target.
// Render Surface nodes to this buffer.
func (c *LayerCompositor) GetSurfaceBuffer() *ebiten.Image {
	return c.surfaceBuffer
}

// GetAnonymousBuffer returns the Anonymous Layer render target.
// Render Specter nodes to this buffer.
func (c *LayerCompositor) GetAnonymousBuffer() *ebiten.Image {
	return c.anonymousBuffer
}

// ClearBuffers clears both layer buffers.
func (c *LayerCompositor) ClearBuffers() {
	if c.surfaceBuffer != nil {
		c.surfaceBuffer.Clear()
	}
	if c.anonymousBuffer != nil {
		c.anonymousBuffer.Clear()
	}
}

// Composite blends layers and renders to destination with adjustable opacity.
// surfaceOpacity: Surface Layer opacity (0-1, per PULSE_MAP.md layer blend slider)
// anonymousOpacity: Anonymous Layer opacity (0-1)
// In Fortress mode, surfaceOpacity = 0, anonymousOpacity = 1.
func (c *LayerCompositor) Composite(dst *ebiten.Image, surfaceOpacity, anonymousOpacity float32) {
	if !c.initialized || c.shader == nil {
		// Fallback: draw Surface, then Anonymous with transparency
		opts1 := &ebiten.DrawImageOptions{}
		if surfaceOpacity < 1.0 {
			opts1.ColorScale.ScaleAlpha(surfaceOpacity)
		}
		dst.DrawImage(c.surfaceBuffer, opts1)

		opts2 := &ebiten.DrawImageOptions{}
		if anonymousOpacity < 1.0 {
			opts2.ColorScale.ScaleAlpha(anonymousOpacity)
		}
		dst.DrawImage(c.anonymousBuffer, opts2)
		return
	}

	// Use shader for proper Porter-Duff "over" blending
	opts := &ebiten.DrawRectShaderOptions{}
	opts.Images[0] = c.surfaceBuffer
	opts.Images[1] = c.anonymousBuffer
	opts.Uniforms = map[string]interface{}{
		"Layer1Opacity": surfaceOpacity,
		"Layer2Opacity": anonymousOpacity,
	}

	dst.DrawRectShader(c.cachedWidth, c.cachedHeight, c.shader, opts)
}

// Dispose releases GPU resources.
func (c *LayerCompositor) Dispose() {
	if c.surfaceBuffer != nil {
		c.surfaceBuffer.Deallocate()
		c.surfaceBuffer = nil
	}
	if c.anonymousBuffer != nil {
		c.anonymousBuffer.Deallocate()
		c.anonymousBuffer = nil
	}
	c.initialized = false
}
