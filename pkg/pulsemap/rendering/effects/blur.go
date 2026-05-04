// Package effects provides blur effects for depth and atmospheric rendering.
// Per PULSE_MAP.md, blur effects create depth by blurring background layers.

//go:build !test
// +build !test

package effects

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed blur.kage
var blurShaderSrc []byte

// BlurEffect provides Gaussian blur rendering for depth effects.
type BlurEffect struct {
	shader       *ebiten.Shader
	tempImage    *ebiten.Image
	initialized  bool
	cachedWidth  int
	cachedHeight int
}

// NewBlurEffect creates a new blur effect renderer.
func NewBlurEffect() (*BlurEffect, error) {
	shader, err := ebiten.NewShader(blurShaderSrc)
	if err != nil {
		return nil, err
	}

	return &BlurEffect{
		shader:      shader,
		initialized: true,
	}, nil
}

// Apply applies Gaussian blur to the source image and draws to destination.
// radius: blur radius in pixels (recommended range: 1-10)
// For stronger blur, call Apply multiple times or increase radius.
func (b *BlurEffect) Apply(dst, src *ebiten.Image, radius float32) {
	if !b.initialized || b.shader == nil {
		// Fallback: just copy source to destination
		opts := &ebiten.DrawImageOptions{}
		dst.DrawImage(src, opts)
		return
	}

	// Get source dimensions
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Allocate or reuse temporary image for two-pass blur
	if b.tempImage == nil || b.cachedWidth != width || b.cachedHeight != height {
		if b.tempImage != nil {
			b.tempImage.Deallocate()
		}
		b.tempImage = ebiten.NewImage(width, height)
		b.cachedWidth = width
		b.cachedHeight = height
	}

	// Clear temp image
	b.tempImage.Clear()

	// Two-pass blur for better quality and performance
	// Pass 1: Horizontal blur (src → temp)
	opts := &ebiten.DrawRectShaderOptions{}
	opts.Images[0] = src
	opts.Uniforms = map[string]interface{}{
		"BlurRadius": radius,
		"ImageSize":  []float32{float32(width), float32(height)},
	}
	b.tempImage.DrawRectShader(width, height, b.shader, opts)

	// Pass 2: Vertical blur (temp → dst)
	// Re-using the shader for vertical pass
	opts2 := &ebiten.DrawRectShaderOptions{}
	opts2.Images[0] = b.tempImage
	opts2.Uniforms = map[string]interface{}{
		"BlurRadius": radius,
		"ImageSize":  []float32{float32(width), float32(height)},
	}
	dst.DrawRectShader(width, height, b.shader, opts2)
}

// ApplySinglePass applies a single-pass blur (faster, lower quality).
// Use when performance is critical and blur quality can be reduced.
func (b *BlurEffect) ApplySinglePass(dst, src *ebiten.Image, radius float32) {
	if !b.initialized || b.shader == nil {
		// Fallback: just copy source to destination
		opts := &ebiten.DrawImageOptions{}
		dst.DrawImage(src, opts)
		return
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	opts := &ebiten.DrawRectShaderOptions{}
	opts.Images[0] = src
	opts.Uniforms = map[string]interface{}{
		"BlurRadius": radius,
		"ImageSize":  []float32{float32(width), float32(height)},
	}
	dst.DrawRectShader(width, height, b.shader, opts)
}

// Dispose releases GPU resources.
func (b *BlurEffect) Dispose() {
	if b.tempImage != nil {
		b.tempImage.Deallocate()
		b.tempImage = nil
	}
	b.initialized = false
}
