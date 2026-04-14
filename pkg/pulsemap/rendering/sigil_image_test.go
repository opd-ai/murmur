// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file tests sigil-to-Ebitengine-image conversion.
//
//go:build !noebiten
// +build !noebiten

package rendering

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/sigils"
)

func TestNewSigilCache(t *testing.T) {
	cache := NewSigilCache()
	if cache == nil {
		t.Fatal("NewSigilCache() returned nil")
	}
	if cache.Size() != 0 {
		t.Errorf("New cache size = %d, want 0", cache.Size())
	}
}

func TestSigilCacheGetAndClear(t *testing.T) {
	cache := NewSigilCache()

	// Generate a test sigil.
	publicKey := []byte("test-public-key-for-caching!!!!!")
	sigil := sigils.Generate(publicKey)

	// First get should create the image.
	img1 := cache.Get(sigil)
	if img1 == nil {
		t.Fatal("cache.Get() returned nil for valid sigil")
	}

	// Cache should now have one entry.
	if cache.Size() != 1 {
		t.Errorf("Cache size = %d, want 1", cache.Size())
	}

	// Second get should return cached image.
	img2 := cache.Get(sigil)
	if img2 == nil {
		t.Fatal("cache.Get() returned nil on second call")
	}

	// Should be the same image object.
	if img1 != img2 {
		t.Error("Second Get() returned different image object")
	}

	// Clear should empty the cache.
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("After Clear(), cache size = %d, want 0", cache.Size())
	}
}

func TestSigilCacheRemove(t *testing.T) {
	cache := NewSigilCache()

	publicKey := []byte("test-key-for-remove-test-32bytes")
	sigil := sigils.Generate(publicKey)

	// Add to cache.
	_ = cache.Get(sigil)
	if cache.Size() != 1 {
		t.Errorf("Cache size = %d, want 1", cache.Size())
	}

	// Remove it.
	cache.Remove(sigil.Hash)
	if cache.Size() != 0 {
		t.Errorf("After Remove(), cache size = %d, want 0", cache.Size())
	}
}

func TestSigilCacheNilSigil(t *testing.T) {
	cache := NewSigilCache()

	// Should handle nil gracefully.
	img := cache.Get(nil)
	if img != nil {
		t.Error("cache.Get(nil) should return nil")
	}

	// Cache should still be empty.
	if cache.Size() != 0 {
		t.Errorf("Cache size after nil get = %d, want 0", cache.Size())
	}
}

func TestSigilToEbitenImage(t *testing.T) {
	publicKey := []byte("test-key-for-ebiten-conversion!!")
	sigil := sigils.Generate(publicKey)

	img := SigilToEbitenImage(sigil)
	if img == nil {
		t.Fatal("SigilToEbitenImage() returned nil")
	}

	// Check dimensions match sigil size.
	bounds := img.Bounds()
	if bounds.Dx() != sigils.Size || bounds.Dy() != sigils.Size {
		t.Errorf("Image size = %dx%d, want %dx%d",
			bounds.Dx(), bounds.Dy(), sigils.Size, sigils.Size)
	}
}

func TestSigilToEbitenImageNil(t *testing.T) {
	img := SigilToEbitenImage(nil)
	if img != nil {
		t.Error("SigilToEbitenImage(nil) should return nil")
	}
}

func TestNewSigilOverlay(t *testing.T) {
	overlay := NewSigilOverlay()
	if overlay == nil {
		t.Fatal("NewSigilOverlay() returned nil")
	}
	if overlay.cache == nil {
		t.Error("Overlay has nil cache")
	}
}

func TestSigilOverlaySetAndGet(t *testing.T) {
	overlay := NewSigilOverlay()

	publicKey := []byte("test-overlay-key-32-bytes-long!!")
	sigil := sigils.Generate(publicKey)

	// Initially should return nil.
	img := overlay.GetSigilImage("node1")
	if img != nil {
		t.Error("GetSigilImage() for unknown node should return nil")
	}

	// Set sigil for node.
	overlay.SetSigil("node1", sigil)

	// Should now return image.
	img = overlay.GetSigilImage("node1")
	if img == nil {
		t.Fatal("GetSigilImage() returned nil after SetSigil()")
	}
}

func TestSigilOverlayRemove(t *testing.T) {
	overlay := NewSigilOverlay()

	publicKey := []byte("test-remove-overlay-key-32bytes!")
	sigil := sigils.Generate(publicKey)

	overlay.SetSigil("node2", sigil)

	// Verify it's set.
	img := overlay.GetSigilImage("node2")
	if img == nil {
		t.Fatal("GetSigilImage() returned nil after SetSigil()")
	}

	// Remove it.
	overlay.RemoveSigil("node2")

	// Should now return nil.
	img = overlay.GetSigilImage("node2")
	if img != nil {
		t.Error("GetSigilImage() should return nil after RemoveSigil()")
	}
}

func TestSigilOverlayClear(t *testing.T) {
	overlay := NewSigilOverlay()

	publicKey1 := []byte("test-clear-key-one-32-bytes!!!!!")
	publicKey2 := []byte("test-clear-key-two-32-bytes!!!!!")
	sigil1 := sigils.Generate(publicKey1)
	sigil2 := sigils.Generate(publicKey2)

	overlay.SetSigil("node1", sigil1)
	overlay.SetSigil("node2", sigil2)

	// Both should exist.
	if overlay.GetSigilImage("node1") == nil {
		t.Error("node1 sigil not found")
	}
	if overlay.GetSigilImage("node2") == nil {
		t.Error("node2 sigil not found")
	}

	// Clear all.
	overlay.Clear()

	// Both should be gone.
	if overlay.GetSigilImage("node1") != nil {
		t.Error("node1 sigil should be cleared")
	}
	if overlay.GetSigilImage("node2") != nil {
		t.Error("node2 sigil should be cleared")
	}
}

func TestScaledSigilImage(t *testing.T) {
	publicKey := []byte("test-scaled-sigil-key-32-bytes!!")
	sigil := sigils.Generate(publicKey)

	// Scale to 32x32.
	scaled := ScaledSigilImage(sigil, 32)
	if scaled == nil {
		t.Fatal("ScaledSigilImage() returned nil")
	}

	bounds := scaled.Bounds()
	if bounds.Dx() != 32 || bounds.Dy() != 32 {
		t.Errorf("Scaled image size = %dx%d, want 32x32", bounds.Dx(), bounds.Dy())
	}
}

func TestScaledSigilImageNil(t *testing.T) {
	scaled := ScaledSigilImage(nil, 32)
	if scaled != nil {
		t.Error("ScaledSigilImage(nil, ...) should return nil")
	}
}

func TestScaledSigilImageInvalidSize(t *testing.T) {
	publicKey := []byte("test-invalid-size-key-32-bytes!!")
	sigil := sigils.Generate(publicKey)

	// Size 0 should return nil.
	scaled := ScaledSigilImage(sigil, 0)
	if scaled != nil {
		t.Error("ScaledSigilImage(..., 0) should return nil")
	}

	// Negative size should return nil.
	scaled = ScaledSigilImage(sigil, -10)
	if scaled != nil {
		t.Error("ScaledSigilImage(..., -10) should return nil")
	}
}

func TestSpecterSigilToEbitenImage(t *testing.T) {
	publicKey := []byte("specter-key-for-ebiten-32-bytes!")
	sigil := sigils.GenerateSpecter(publicKey)

	img := SigilToEbitenImage(sigil)
	if img == nil {
		t.Fatal("SigilToEbitenImage() returned nil for Specter sigil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != sigils.Size || bounds.Dy() != sigils.Size {
		t.Errorf("Specter image size = %dx%d, want %dx%d",
			bounds.Dx(), bounds.Dy(), sigils.Size, sigils.Size)
	}
}

func TestMaskedEventSigilToEbitenImage(t *testing.T) {
	eventID := []byte("test-event-id")
	sigil := sigils.GenerateMaskedEvent(eventID)

	img := SigilToEbitenImage(sigil)
	if img == nil {
		t.Fatal("SigilToEbitenImage() returned nil for Masked Event sigil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != sigils.Size || bounds.Dy() != sigils.Size {
		t.Errorf("Masked event image size = %dx%d, want %dx%d",
			bounds.Dx(), bounds.Dy(), sigils.Size, sigils.Size)
	}
}
