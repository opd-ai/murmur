// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This stub file provides no-op implementations when Ebitengine is not available.
//
//go:build test
// +build test

package rendering

import (
	"sync"

	"github.com/opd-ai/murmur/pkg/identity/sigils"
)

// SigilCache caches images for sigils (stub implementation).
type SigilCache struct {
	mu    sync.RWMutex
	cache map[[32]byte]interface{}
}

// NewSigilCache creates a new sigil image cache (stub).
func NewSigilCache() *SigilCache {
	return &SigilCache{
		cache: make(map[[32]byte]interface{}),
	}
}

// Get retrieves an image for the given sigil (stub returns nil).
func (c *SigilCache) Get(_ *sigils.Sigil) interface{} {
	return nil
}

// Remove evicts a sigil from the cache (stub).
func (c *SigilCache) Remove(hash [32]byte) {
	c.mu.Lock()
	delete(c.cache, hash)
	c.mu.Unlock()
}

// Clear removes all cached sigil images (stub).
func (c *SigilCache) Clear() {
	c.mu.Lock()
	c.cache = make(map[[32]byte]interface{})
	c.mu.Unlock()
}

// Size returns the number of cached sigil images (stub).
func (c *SigilCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// SigilOverlay represents a sigil overlay on the Pulse Map (stub).
type SigilOverlay struct {
	cache *SigilCache
	mu    sync.RWMutex
}

// NewSigilOverlay creates a new sigil overlay manager (stub).
func NewSigilOverlay() *SigilOverlay {
	return &SigilOverlay{
		cache: NewSigilCache(),
	}
}

// SetSigil associates a sigil with a node ID (stub).
func (o *SigilOverlay) SetSigil(_ string, _ *sigils.Sigil) {}

// RemoveSigil removes the sigil association for a node (stub).
func (o *SigilOverlay) RemoveSigil(_ string) {}

// Clear removes all sigil associations (stub).
func (o *SigilOverlay) Clear() {
	o.cache.Clear()
}
