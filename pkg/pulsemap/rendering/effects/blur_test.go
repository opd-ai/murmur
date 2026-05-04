package effects

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBlurEffect verifies blur effect initialization.
func TestNewBlurEffect(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)
	assert.NotNil(t, blur)
	assert.True(t, blur.initialized)
}

// TestBlurEffectApply verifies basic Apply functionality.
func TestBlurEffectApply(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	// Create source and destination images
	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	// Apply blur with various radii
	blur.Apply(dst, src, 1.0)
	blur.Apply(dst, src, 5.0)
	blur.Apply(dst, src, 10.0)

	// Test completes without panics = success in test build
}

// TestBlurEffectApplySinglePass verifies single-pass blur.
func TestBlurEffectApplySinglePass(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	blur.ApplySinglePass(dst, src, 3.0)

	// Test completes without panics = success
}

// TestBlurEffectDispose verifies resource cleanup.
func TestBlurEffectDispose(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	blur.Dispose()
	assert.False(t, blur.initialized)
}

// TestBlurEffectMultipleApply verifies repeated blur applications.
func TestBlurEffectMultipleApply(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	// Apply blur multiple times
	for i := 0; i < 5; i++ {
		blur.Apply(dst, src, float32(i+1))
	}

	// Test completes without panics or memory leaks
}

// TestBlurEffectZeroRadius verifies handling of zero blur radius.
func TestBlurEffectZeroRadius(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	blur.Apply(dst, src, 0)

	// Should handle zero radius gracefully
}

// TestBlurEffectLargeRadius verifies handling of large blur radius.
func TestBlurEffectLargeRadius(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	blur.Apply(dst, src, 50.0)

	// Should handle large radius gracefully
}

// TestBlurEffectDifferentSizes verifies blur on different image sizes.
func TestBlurEffectDifferentSizes(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	sizes := []struct{ w, h int }{
		{10, 10},
		{100, 100},
		{256, 256},
		{800, 600},
	}

	for _, size := range sizes {
		src := ebiten.NewImage(size.w, size.h)
		dst := ebiten.NewImage(size.w, size.h)
		blur.Apply(dst, src, 3.0)
	}

	// Test completes without panics for all sizes
}

// TestBlurEffectSinglePassMultipleTimes verifies single-pass blur repeated.
func TestBlurEffectSinglePassMultipleTimes(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	// Single-pass blur multiple times
	for i := 0; i < 10; i++ {
		blur.ApplySinglePass(dst, src, 2.0)
	}

	// Test completes without panics
}

// TestBlurEffectDisposeAndReuse verifies dispose doesn't break subsequent use.
func TestBlurEffectDisposeAndReuse(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	// Use blur
	blur.Apply(dst, src, 5.0)

	// Dispose
	blur.Dispose()

	// Try to use after dispose (should handle gracefully or no-op in stub)
	blur.Apply(dst, src, 5.0)
}

// TestBlurEffectNonSquareImages verifies blur on non-square images.
func TestBlurEffectNonSquareImages(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(200, 100)
	dst := ebiten.NewImage(200, 100)

	blur.Apply(dst, src, 4.0)

	// Test completes without panics
}

// TestBlurEffectSmallRadius verifies handling of small blur radius.
func TestBlurEffectSmallRadius(t *testing.T) {
	blur, err := NewBlurEffect()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	blur.Apply(dst, src, 0.1)
	blur.Apply(dst, src, 0.5)

	// Should handle small radius gracefully
}
