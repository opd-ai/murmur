// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file provides sigil-to-Ebitengine-image conversion for Pulse Map overlay.
//

//go:build !test
// +build !test

package rendering

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/sigils"
)

// glowCacheKey uniquely identifies a glow image by its parameters.
type glowCacheKey struct {
	size      int
	r, g, b   uint8
	intensity uint8 // scaled 0–255 from float32
}

// glowImageCache stores pre-rendered glow images keyed by parameters.
// Glow images are pure functions of their inputs so they are safe to reuse.
var (
	glowCacheMu sync.RWMutex
	glowImages  = make(map[glowCacheKey]*ebiten.Image, 64)
)

// sigilCacheMaxSize is the maximum number of sigil images held in the cache.
// Beyond this limit the oldest entry is evicted and its VRAM is freed.
const sigilCacheMaxSize = 512

// SigilCache caches Ebitengine images for sigils to avoid recreation each frame.
// Per PULSE_MAP.md, sigils are rendered as node overlays in the Pulse Map.
// The cache is bounded to sigilCacheMaxSize entries; oldest entries are evicted
// first to free VRAM (per audit MEDIUM finding: unbounded cache growth).
type SigilCache struct {
	mu          sync.RWMutex
	cache       map[[32]byte]*ebiten.Image
	insertOrder [][32]byte // FIFO order for eviction
}

// NewSigilCache creates a new sigil image cache.
func NewSigilCache() *SigilCache {
	return &SigilCache{
		cache:       make(map[[32]byte]*ebiten.Image),
		insertOrder: make([][32]byte, 0, sigilCacheMaxSize),
	}
}

// Get retrieves or creates an Ebitengine image for the given sigil.
// The result is cached by sigil hash for efficient reuse.
func (c *SigilCache) Get(sigil *sigils.Sigil) *ebiten.Image {
	if sigil == nil || sigil.Image == nil {
		return nil
	}

	c.mu.RLock()
	if img, ok := c.cache[sigil.Hash]; ok {
		c.mu.RUnlock()
		return img
	}
	c.mu.RUnlock()

	// Create Ebitengine image from sigil.
	img := SigilToEbitenImage(sigil)

	// Cache the result, evicting the oldest entry if the cache is full.
	c.mu.Lock()
	if len(c.cache) >= sigilCacheMaxSize && len(c.insertOrder) > 0 {
		oldest := c.insertOrder[0]
		c.insertOrder = c.insertOrder[1:]
		if evicted, ok := c.cache[oldest]; ok {
			evicted.Deallocate()
			delete(c.cache, oldest)
		}
	}
	c.cache[sigil.Hash] = img
	c.insertOrder = append(c.insertOrder, sigil.Hash)
	c.mu.Unlock()

	return img
}

// Remove evicts a sigil from the cache.
func (c *SigilCache) Remove(hash [32]byte) {
	c.mu.Lock()
	delete(c.cache, hash)
	c.mu.Unlock()
}

// Clear removes all cached sigil images and frees associated VRAM.
func (c *SigilCache) Clear() {
	c.mu.Lock()
	for _, img := range c.cache {
		img.Deallocate()
	}
	c.cache = make(map[[32]byte]*ebiten.Image)
	c.insertOrder = c.insertOrder[:0]
	c.mu.Unlock()
}

// Size returns the number of cached sigil images.
func (c *SigilCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// SigilToEbitenImage converts a sigil's image.RGBA to an Ebitengine image.
// This is the primary conversion function for displaying sigils in the Pulse Map.
func SigilToEbitenImage(sigil *sigils.Sigil) *ebiten.Image {
	if sigil == nil || sigil.Image == nil {
		return nil
	}

	return ebiten.NewImageFromImage(sigil.Image)
}

// RenderSigilAtNode draws a sigil image at the given node position.
// The sigil is scaled to fit within the node radius.
func RenderSigilAtNode(dst, sigilImg *ebiten.Image, x, y, nodeRadius float32) {
	if sigilImg == nil {
		return
	}

	// Get sigil dimensions.
	bounds := sigilImg.Bounds()
	sigilSize := float32(bounds.Dx())

	// Scale sigil to fit within node (slightly smaller for visual padding).
	scale := (nodeRadius * 1.8) / sigilSize

	// Calculate draw options.
	opts := &ebiten.DrawImageOptions{}

	// Center the sigil on the node position.
	opts.GeoM.Translate(-float64(sigilSize)/2, -float64(sigilSize)/2)
	opts.GeoM.Scale(float64(scale), float64(scale))
	opts.GeoM.Translate(float64(x), float64(y))

	// Draw with alpha blending.
	dst.DrawImage(sigilImg, opts)
}

// RenderSigilWithGlow draws a sigil with a glow effect around it.
// Used for active or selected nodes to highlight their sigil.
func RenderSigilWithGlow(dst, sigilImg *ebiten.Image, x, y, nodeRadius float32, glowColor color.RGBA, glowIntensity float32) {
	if sigilImg == nil {
		return
	}

	// Draw glow first (behind sigil).
	glowRadius := nodeRadius * 2.5
	drawGlowCircle(dst, x, y, glowRadius, glowColor, glowIntensity)

	// Draw sigil on top.
	RenderSigilAtNode(dst, sigilImg, x, y, nodeRadius)
}

// drawGlowCircle draws a soft glow circle effect.
// The rendered glow image is cached by (size, colour, intensity) to avoid
// allocating a new GPU texture on every call (per the audit HIGH finding).
func drawGlowCircle(dst *ebiten.Image, cx, cy, radius float32, c color.RGBA, intensity float32) {
	size := int(radius * 2)
	if size < 1 {
		return
	}

	key := buildGlowCacheKey(size, c, intensity)
	glowImg := getOrCreateGlowImage(key, size, c, intensity)
	drawGlowImage(dst, glowImg, cx, cy, size)
}

// buildGlowCacheKey creates a cache key for the given glow parameters.
func buildGlowCacheKey(size int, c color.RGBA, intensity float32) glowCacheKey {
	return glowCacheKey{
		size:      size,
		r:         c.R,
		g:         c.G,
		b:         c.B,
		intensity: uint8(intensity * 255),
	}
}

// getOrCreateGlowImage retrieves a cached glow image or creates and caches a new one.
func getOrCreateGlowImage(key glowCacheKey, size int, c color.RGBA, intensity float32) *ebiten.Image {
	glowCacheMu.RLock()
	glowImg, ok := glowImages[key]
	glowCacheMu.RUnlock()

	if !ok {
		glowImg = createGlowImage(size, c, intensity)
		glowCacheMu.Lock()
		glowImages[key] = glowImg
		glowCacheMu.Unlock()
	}

	return glowImg
}

// createGlowImage generates a new glow image with radial falloff.
func createGlowImage(size int, c color.RGBA, intensity float32) *ebiten.Image {
	glow := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float32(size) / 2
	maxDist := center * center

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			setGlowPixel(glow, x, y, center, maxDist, c, intensity)
		}
	}

	return ebiten.NewImageFromImage(glow)
}

// setGlowPixel sets a single glow pixel with radial falloff.
func setGlowPixel(glow *image.RGBA, x, y int, center, maxDist float32, c color.RGBA, intensity float32) {
	dx := float32(x) - center
	dy := float32(y) - center
	dist := dx*dx + dy*dy

	if dist < maxDist {
		falloff := 1.0 - dist/maxDist
		alpha := uint8(float32(c.A) * falloff * intensity)
		glow.SetRGBA(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha})
	}
}

// drawGlowImage draws the cached glow image at the specified position.
func drawGlowImage(dst, glowImg *ebiten.Image, cx, cy float32, size int) {
	center := float32(size) / 2
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(cx-center), float64(cy-center))
	dst.DrawImage(glowImg, opts)
}

// ScaledSigilImage creates a scaled version of a sigil image.
// This is useful for different zoom levels in the Pulse Map.
func ScaledSigilImage(sigil *sigils.Sigil, targetSize int) *ebiten.Image {
	if sigil == nil || sigil.Image == nil || targetSize <= 0 {
		return nil
	}

	// Create Ebitengine image from sigil.
	srcImg := ebiten.NewImageFromImage(sigil.Image)

	// Create target image.
	dstImg := ebiten.NewImage(targetSize, targetSize)

	// Calculate scale.
	srcSize := sigil.Image.Bounds().Dx()
	scale := float64(targetSize) / float64(srcSize)

	// Draw scaled.
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	dstImg.DrawImage(srcImg, opts)

	return dstImg
}

// SigilOverlay represents a sigil overlay on the Pulse Map.
// Used for rendering sigils as node decorations.
type SigilOverlay struct {
	cache *SigilCache

	// sigils maps node IDs to their sigils.
	sigils map[string]*sigils.Sigil
	mu     sync.RWMutex
}

// NewSigilOverlay creates a new sigil overlay manager.
func NewSigilOverlay() *SigilOverlay {
	return &SigilOverlay{
		cache:  NewSigilCache(),
		sigils: make(map[string]*sigils.Sigil),
	}
}

// SetSigil associates a sigil with a node ID.
func (o *SigilOverlay) SetSigil(nodeID string, sigil *sigils.Sigil) {
	o.mu.Lock()
	o.sigils[nodeID] = sigil
	o.mu.Unlock()
}

// RemoveSigil removes the sigil association for a node.
func (o *SigilOverlay) RemoveSigil(nodeID string) {
	o.mu.Lock()
	delete(o.sigils, nodeID)
	o.mu.Unlock()
}

// GetSigilImage returns the cached Ebitengine image for a node's sigil.
func (o *SigilOverlay) GetSigilImage(nodeID string) *ebiten.Image {
	o.mu.RLock()
	sigil, ok := o.sigils[nodeID]
	o.mu.RUnlock()

	if !ok {
		return nil
	}

	return o.cache.Get(sigil)
}

// RenderNodeSigil draws the sigil for a node at the given position.
func (o *SigilOverlay) RenderNodeSigil(dst *ebiten.Image, nodeID string, x, y, radius float32) {
	img := o.GetSigilImage(nodeID)
	if img == nil {
		return
	}

	RenderSigilAtNode(dst, img, x, y, radius)
}

// Clear removes all sigil associations and clears the cache.
func (o *SigilOverlay) Clear() {
	o.mu.Lock()
	o.sigils = make(map[string]*sigils.Sigil)
	o.mu.Unlock()
	o.cache.Clear()
}
