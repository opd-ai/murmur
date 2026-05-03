// Package mechanics - Cartographer's Trail implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, Cartographer's Trail rewards Specters
// for exploring unfamiliar regions of the Pulse Map.
package mechanics

import (
	"math"
	"sync"
	"time"
)

// Cartographer constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// CartographerWindow is the lookback period for exploration (90 days).
	CartographerWindow = 90 * 24 * time.Hour

	// CartographerWanderer is territories needed for Wanderer badge.
	CartographerWanderer = 5

	// CartographerPathfinder is territories needed for Pathfinder badge.
	CartographerPathfinder = 20

	// CartographerMaster is territories needed for Cartographer badge.
	CartographerMaster = 50
)

// CartographerBadge represents exploration milestone badges.
type CartographerBadge uint8

const (
	CartographerNone            CartographerBadge = iota // No badge yet.
	CartographerBadgeWanderer                            // 5 territories: compass glyph.
	CartographerBadgePathfinder                          // 20 territories: animated compass.
	CartographerBadgeMaster                              // 50 territories: detailed map glyph.
)

// String returns the badge name.
func (b CartographerBadge) String() string {
	switch b {
	case CartographerBadgeWanderer:
		return "Wanderer"
	case CartographerBadgePathfinder:
		return "Pathfinder"
	case CartographerBadgeMaster:
		return "Cartographer"
	default:
		return ""
	}
}

// TerritoryDiscovery records when a territory was first visited.
type TerritoryDiscovery struct {
	TerritoryHash string    // Hash of the territory cluster.
	DiscoveredAt  time.Time // When the territory was discovered.
}

// CartographerTrail tracks a Specter's exploration progress.
type CartographerTrail struct {
	mu sync.RWMutex

	specterKey   [32]byte
	discoveries  []TerritoryDiscovery
	territorySet map[string]bool // For deduplication.
}

// NewCartographerTrail creates a new exploration tracker for a Specter.
func NewCartographerTrail(specterKey [32]byte) *CartographerTrail {
	return &CartographerTrail{
		specterKey:   specterKey,
		discoveries:  make([]TerritoryDiscovery, 0),
		territorySet: make(map[string]bool),
	}
}

// DiscoverTerritory records a new territory discovery.
// Returns true if this is a new discovery, false if already visited.
func (c *CartographerTrail) DiscoverTerritory(territoryHash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.territorySet[territoryHash] {
		return false // Already discovered.
	}

	c.territorySet[territoryHash] = true
	c.discoveries = append(c.discoveries, TerritoryDiscovery{
		TerritoryHash: territoryHash,
		DiscoveredAt:  time.Now(),
	})
	return true
}

// GetDiscoveries returns all territory discoveries.
func (c *CartographerTrail) GetDiscoveries() []TerritoryDiscovery {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]TerritoryDiscovery, len(c.discoveries))
	copy(result, c.discoveries)
	return result
}

// GetRecentDiscoveries returns discoveries within the lookback window.
func (c *CartographerTrail) GetRecentDiscoveries() []TerritoryDiscovery {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-CartographerWindow)
	var result []TerritoryDiscovery

	for _, d := range c.discoveries {
		if d.DiscoveredAt.After(cutoff) {
			result = append(result, d)
		}
	}
	return result
}

// Count returns the total number of discovered territories.
func (c *CartographerTrail) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.discoveries)
}

// CountRecent returns territories discovered within the lookback window.
func (c *CartographerTrail) CountRecent() int {
	return len(c.GetRecentDiscoveries())
}

// ComputeScore calculates the Cartographer score.
// Per ANONYMOUS_GAME_MECHANICS.md:
// cartographer_score = 6 * ln(1 + distinct_territories_visited_90d).
func (c *CartographerTrail) ComputeScore() float64 {
	recentCount := float64(c.CountRecent())
	return 6.0 * math.Log1p(recentCount)
}

// GetBadge returns the current exploration badge based on total discoveries.
func (c *CartographerTrail) GetBadge() CartographerBadge {
	count := c.Count()

	if count >= CartographerMaster {
		return CartographerBadgeMaster
	}
	if count >= CartographerPathfinder {
		return CartographerBadgePathfinder
	}
	if count >= CartographerWanderer {
		return CartographerBadgeWanderer
	}
	return CartographerNone
}

// IsDiscovered checks if a territory has been discovered.
func (c *CartographerTrail) IsDiscovered(territoryHash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.territorySet[territoryHash]
}

// GetNextMilestone returns the next badge milestone and territories needed.
func (c *CartographerTrail) GetNextMilestone() (CartographerBadge, int) {
	count := c.Count()

	if count < CartographerWanderer {
		return CartographerBadgeWanderer, CartographerWanderer - count
	}
	if count < CartographerPathfinder {
		return CartographerBadgePathfinder, CartographerPathfinder - count
	}
	if count < CartographerMaster {
		return CartographerBadgeMaster, CartographerMaster - count
	}
	// All badges earned.
	return CartographerNone, 0
}

// GetSpecterKey returns the Specter's public key.
func (c *CartographerTrail) GetSpecterKey() [32]byte {
	return c.specterKey
}

// GarbageCollect removes discoveries outside the lookback window.
// Note: This only removes from the recent count, not from badge progress.
// Badge progress (total discoveries) is permanent.
func (c *CartographerTrail) GarbageCollect() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-CartographerWindow)
	removed := 0

	// Filter discoveries but keep the territory set intact for badge progress.
	var kept []TerritoryDiscovery
	for _, d := range c.discoveries {
		if d.DiscoveredAt.After(cutoff) {
			kept = append(kept, d)
		} else {
			removed++
		}
	}

	c.discoveries = kept
	return removed
}

// CartographerManager manages multiple Specters' exploration trails.
type CartographerManager struct {
	mu     sync.RWMutex
	trails map[string]*CartographerTrail // Keyed by hex(specterKey).
}

// NewCartographerManager creates a new manager.
func NewCartographerManager() *CartographerManager {
	return &CartographerManager{
		trails: make(map[string]*CartographerTrail),
	}
}

// GetOrCreateTrail gets or creates a trail for a Specter.
func (m *CartographerManager) GetOrCreateTrail(specterKey [32]byte) *CartographerTrail {
	hex := KeyToHex(specterKey[:])

	m.mu.Lock()
	defer m.mu.Unlock()

	if trail, ok := m.trails[hex]; ok {
		return trail
	}

	trail := NewCartographerTrail(specterKey)
	m.trails[hex] = trail
	return trail
}

// GetTrail retrieves a trail for a Specter (nil if not found).
func (m *CartographerManager) GetTrail(specterKey [32]byte) *CartographerTrail {
	hex := KeyToHex(specterKey[:])

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.trails[hex]
}

// RecordDiscovery records a territory discovery for a Specter.
// Returns true if this is a new discovery.
func (m *CartographerManager) RecordDiscovery(specterKey [32]byte, territoryHash string) bool {
	trail := m.GetOrCreateTrail(specterKey)
	return trail.DiscoverTerritory(territoryHash)
}

// GetScore returns the Cartographer score for a Specter.
func (m *CartographerManager) GetScore(specterKey [32]byte) float64 {
	trail := m.GetTrail(specterKey)
	if trail == nil {
		return 0
	}
	return trail.ComputeScore()
}

// GetBadge returns the current badge for a Specter.
func (m *CartographerManager) GetBadge(specterKey [32]byte) CartographerBadge {
	trail := m.GetTrail(specterKey)
	if trail == nil {
		return CartographerNone
	}
	return trail.GetBadge()
}

// Count returns the number of Specters being tracked.
func (m *CartographerManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.trails)
}

// GarbageCollectAll runs garbage collection on all trails.
func (m *CartographerManager) GarbageCollectAll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, trail := range m.trails {
		total += trail.GarbageCollect()
	}
	return total
}
