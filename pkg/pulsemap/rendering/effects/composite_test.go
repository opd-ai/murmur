package effects

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLayerCompositor verifies compositor initialization.
func TestNewLayerCompositor(t *testing.T) {
	comp, err := NewLayerCompositor(800, 600)
	require.NoError(t, err)
	assert.NotNil(t, comp)
	assert.True(t, comp.initialized)
	assert.Equal(t, 800, comp.cachedWidth)
	assert.Equal(t, 600, comp.cachedHeight)
}

// TestLayerCompositorGetBuffers verifies buffer access.
func TestLayerCompositorGetBuffers(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	surfBuf := comp.GetSurfaceBuffer()
	anonyBuf := comp.GetAnonymousBuffer()

	assert.NotNil(t, surfBuf)
	assert.NotNil(t, anonyBuf)
	// Buffers may or may not be the same object in test builds, that's ok
}

// TestLayerCompositorClearBuffers verifies buffer clearing.
func TestLayerCompositorClearBuffers(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	comp.ClearBuffers()
	// Test completes without panics = success
}

// TestLayerCompositorResize verifies buffer resizing.
func TestLayerCompositorResize(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	comp.Resize(200, 150)
	assert.Equal(t, 200, comp.cachedWidth)
	assert.Equal(t, 150, comp.cachedHeight)
}

// TestLayerCompositorResizeSameSize verifies no-op on same size resize.
func TestLayerCompositorResizeSameSize(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	comp.Resize(100, 100)
	assert.Equal(t, 100, comp.cachedWidth)
	assert.Equal(t, 100, comp.cachedHeight)
}

// TestLayerCompositorComposite verifies basic compositing.
func TestLayerCompositorComposite(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	dst := ebiten.NewImage(100, 100)

	// Test various opacity combinations
	comp.Composite(dst, 1.0, 1.0)
	comp.Composite(dst, 1.0, 0.5)
	comp.Composite(dst, 0.5, 1.0)
	comp.Composite(dst, 0.5, 0.5)

	// Test completes without panics = success
}

// TestLayerCompositorFortressMode verifies Fortress mode rendering.
func TestLayerCompositorFortressMode(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	dst := ebiten.NewImage(100, 100)

	// Fortress mode: Surface opacity = 0, Anonymous opacity = 1
	comp.Composite(dst, 0.0, 1.0)

	// Test completes without panics = success
}

// TestLayerCompositorSurfaceOnly verifies Surface-only mode.
func TestLayerCompositorSurfaceOnly(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	dst := ebiten.NewImage(100, 100)

	// Surface only: Surface opacity = 1, Anonymous opacity = 0
	comp.Composite(dst, 1.0, 0.0)

	// Test completes without panics = success
}

// TestLayerCompositorDispose verifies resource cleanup.
func TestLayerCompositorDispose(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	comp.Dispose()
	assert.False(t, comp.initialized)
}

// TestLayerCompositorMultipleResize verifies repeated resizing.
func TestLayerCompositorMultipleResize(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	sizes := []struct{ w, h int }{
		{200, 200},
		{300, 250},
		{150, 100},
		{800, 600},
	}

	for _, size := range sizes {
		comp.Resize(size.w, size.h)
		assert.Equal(t, size.w, comp.cachedWidth)
		assert.Equal(t, size.h, comp.cachedHeight)
	}
}

// TestLayerCompositorBlendRange verifies opacity range handling.
func TestLayerCompositorBlendRange(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	dst := ebiten.NewImage(100, 100)

	// Test opacity range 0.0 to 1.0 in 0.25 increments
	for surf := float32(0.0); surf <= 1.0; surf += 0.25 {
		for anon := float32(0.0); anon <= 1.0; anon += 0.25 {
			comp.Composite(dst, surf, anon)
		}
	}

	// Test completes without panics = success
}

// TestLayerCompositorAfterClear verifies compositing after clear.
func TestLayerCompositorAfterClear(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	dst := ebiten.NewImage(100, 100)

	comp.ClearBuffers()
	comp.Composite(dst, 0.5, 0.5)

	// Test completes without panics = success
}

// TestLayerCompositorResizeAndComposite verifies resize then composite.
func TestLayerCompositorResizeAndComposite(t *testing.T) {
	comp, err := NewLayerCompositor(100, 100)
	require.NoError(t, err)

	comp.Resize(200, 200)
	dst := ebiten.NewImage(200, 200)
	comp.Composite(dst, 0.8, 0.6)

	// Test completes without panics = success
}

// TestLayerCompositorNonSquare verifies non-square dimensions.
func TestLayerCompositorNonSquare(t *testing.T) {
	comp, err := NewLayerCompositor(800, 600)
	require.NoError(t, err)

	dst := ebiten.NewImage(800, 600)
	comp.Composite(dst, 0.7, 0.7)

	// Test completes without panics = success
}
