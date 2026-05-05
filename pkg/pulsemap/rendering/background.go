// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file implements procedural background generation with gradient and noise.
//
//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// BackgroundRenderer generates and renders the Pulse Map background.
// Per ROADMAP.md line 686, this creates a dark blue-gray gradient with procedural noise.
type BackgroundRenderer struct {
	// backgroundImage is the cached background texture.
	backgroundImage *ebiten.Image

	// width and height track the current background dimensions.
	width  int
	height int

	// noiseScale controls the frequency of noise patterns.
	noiseScale float64

	// noiseAmplitude controls the intensity of noise (0.0-1.0).
	noiseAmplitude float64
}

// NewBackgroundRenderer creates a new background renderer with default settings.
func NewBackgroundRenderer() *BackgroundRenderer {
	return &BackgroundRenderer{
		noiseScale:     0.015, // Low frequency for subtle, organic patterns
		noiseAmplitude: 0.12,  // Subtle noise intensity
	}
}

// Draw renders the background to the screen.
// Regenerates the background if screen dimensions have changed.
func (b *BackgroundRenderer) Draw(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Regenerate background if size changed or not yet created.
	if b.backgroundImage == nil || b.width != w || b.height != h {
		b.regenerate(w, h)
	}

	// Draw the cached background.
	if b.backgroundImage != nil {
		screen.DrawImage(b.backgroundImage, nil)
	}
}

// regenerate creates a new background image with gradient and noise.
func (b *BackgroundRenderer) regenerate(width, height int) {
	b.width = width
	b.height = height

	// Create new image.
	b.backgroundImage = ebiten.NewImage(width, height)

	// Fill with gradient and noise.
	pixels := make([]byte, width*height*4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate gradient factor (0.0 at top, 1.0 at bottom).
			gradientFactor := float64(y) / float64(height)

			// Base gradient colors (dark blue-gray).
			// Top: darker blue-gray (8, 10, 16)
			// Bottom: slightly lighter blue-gray (14, 18, 26)
			r := interpolate(8, 14, gradientFactor)
			g := interpolate(10, 18, gradientFactor)
			blueBase := interpolate(16, 26, gradientFactor)

			// Apply procedural noise.
			noise := b.perlinNoise(float64(x), float64(y))
			noiseOffset := noise * b.noiseAmplitude * 30.0 // Scale noise to [0, ~3.6] for subtle variation

			// Apply noise to all channels for organic texture.
			r += noiseOffset
			g += noiseOffset
			blueBase += noiseOffset

			// Clamp to valid range [0, 255].
			r = clamp(r, 0, 255)
			g = clamp(g, 0, 255)
			blueBase = clamp(blueBase, 0, 255)

			idx := (y*width + x) * 4
			pixels[idx] = uint8(r)
			pixels[idx+1] = uint8(g)
			pixels[idx+2] = uint8(blueBase)
			pixels[idx+3] = 255 // Fully opaque
		}
	}

	b.backgroundImage.WritePixels(pixels)
}

// perlinNoise generates a simplified Perlin-like noise value at (x, y).
// Returns a value in the range [-1.0, 1.0].
func (b *BackgroundRenderer) perlinNoise(x, y float64) float64 {
	// Scale input coordinates.
	x *= b.noiseScale
	y *= b.noiseScale

	// Use multiple octaves for more organic noise.
	noise := 0.0
	amplitude := 1.0
	frequency := 1.0
	maxValue := 0.0

	// 3 octaves for detail at multiple scales.
	for i := 0; i < 3; i++ {
		noise += amplitude * simplexNoise(x*frequency, y*frequency)
		maxValue += amplitude
		amplitude *= 0.5
		frequency *= 2.0
	}

	// Normalize to [-1, 1].
	return noise / maxValue
}

// simplexNoise is a simplified 2D noise function based on gradient noise.
// Returns a value in the range [-1.0, 1.0].
func simplexNoise(x, y float64) float64 {
	// Grid cell coordinates.
	xi := int(math.Floor(x))
	yi := int(math.Floor(y))

	// Fractional part within cell.
	xf := x - float64(xi)
	yf := y - float64(yi)

	// Smooth the fractional part (fade curves).
	u := fade(xf)
	v := fade(yf)

	// Hash coordinates of the 4 cell corners.
	aa := hash2D(xi, yi)
	ab := hash2D(xi, yi+1)
	ba := hash2D(xi+1, yi)
	bb := hash2D(xi+1, yi+1)

	// Gradient vectors for each corner (simplified to random dot products).
	g1 := gradient(aa, xf, yf)
	g2 := gradient(ba, xf-1, yf)
	g3 := gradient(ab, xf, yf-1)
	g4 := gradient(bb, xf-1, yf-1)

	// Bilinear interpolation.
	x1 := lerp(g1, g2, u)
	x2 := lerp(g3, g4, u)
	return lerp(x1, x2, v)
}

// hash2D generates a pseudo-random hash from 2D coordinates.
func hash2D(x, y int) int {
	// Simple integer hash function.
	n := x + y*57
	n = (n << 13) ^ n
	return (n*(n*n*15731+789221) + 1376312589) & 0x7fffffff
}

// gradient computes a dot product with a pseudo-random gradient vector.
func gradient(hash int, x, y float64) float64 {
	// Use hash to pick a gradient direction (simplified to 4 directions).
	h := hash & 3
	switch h {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	default:
		return -x - y
	}
}

// fade applies a smoothing curve (6t^5 - 15t^4 + 10t^3) for interpolation.
func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

// lerp performs linear interpolation between a and b by factor t.
func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// interpolate linearly interpolates between two color channel values.
func interpolate(start, end, factor float64) float64 {
	return start + factor*(end-start)
}

// clamp constrains a value to the range [min, max].
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// SetNoiseScale adjusts the frequency of noise patterns.
// Lower values create larger, smoother patterns. Default is 0.015.
func (b *BackgroundRenderer) SetNoiseScale(scale float64) {
	b.noiseScale = scale
	// Invalidate cached background to force regeneration.
	b.backgroundImage = nil
}

// SetNoiseAmplitude adjusts the intensity of noise.
// Range [0.0, 1.0]. Default is 0.12 for subtle variation.
func (b *BackgroundRenderer) SetNoiseAmplitude(amplitude float64) {
	b.noiseAmplitude = clamp(amplitude, 0.0, 1.0)
	// Invalidate cached background to force regeneration.
	b.backgroundImage = nil
}

// GetBaseColor returns the average background color (for UI contrast calculations).
func (b *BackgroundRenderer) GetBaseColor() color.RGBA {
	// Return the mid-gradient color.
	return color.RGBA{11, 14, 21, 255}
}
