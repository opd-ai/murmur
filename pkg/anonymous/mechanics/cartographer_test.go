// Package mechanics - Cartographer's Trail tests.
package mechanics

import (
	"testing"
	"time"
)

func TestNewCartographerTrail(t *testing.T) {
	key := [32]byte{1, 2, 3, 4}
	trail := NewCartographerTrail(key)

	if trail == nil {
		t.Fatal("NewCartographerTrail returned nil")
	}
	if trail.Count() != 0 {
		t.Errorf("Expected 0 discoveries, got %d", trail.Count())
	}
	if trail.GetSpecterKey() != key {
		t.Error("Specter key mismatch")
	}
}

func TestCartographerDiscovery(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// First discovery should succeed.
	isNew := trail.DiscoverTerritory("territory-alpha")
	if !isNew {
		t.Error("First discovery should return true")
	}
	if trail.Count() != 1 {
		t.Errorf("Expected 1 discovery, got %d", trail.Count())
	}

	// Duplicate discovery should fail.
	isNew = trail.DiscoverTerritory("territory-alpha")
	if isNew {
		t.Error("Duplicate discovery should return false")
	}
	if trail.Count() != 1 {
		t.Errorf("Expected still 1 discovery, got %d", trail.Count())
	}

	// Different territory.
	isNew = trail.DiscoverTerritory("territory-beta")
	if !isNew {
		t.Error("New territory should return true")
	}
	if trail.Count() != 2 {
		t.Errorf("Expected 2 discoveries, got %d", trail.Count())
	}
}

func TestCartographerIsDiscovered(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	trail.DiscoverTerritory("found")

	if !trail.IsDiscovered("found") {
		t.Error("IsDiscovered should return true for discovered territory")
	}
	if trail.IsDiscovered("not-found") {
		t.Error("IsDiscovered should return false for undiscovered territory")
	}
}

func TestCartographerScore(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// Empty trail should have score 0.
	score := trail.ComputeScore()
	if score != 0 {
		t.Errorf("Expected score 0 for empty trail, got %f", score)
	}

	// Add discoveries and check score increases.
	for i := 0; i < 10; i++ {
		trail.DiscoverTerritory("territory-" + string(rune('a'+i)))
	}

	score = trail.ComputeScore()
	// Score = 6 * ln(1 + 10) = 6 * ln(11) ≈ 14.38
	if score < 14 || score > 15 {
		t.Errorf("Expected score ~14.38, got %f", score)
	}
}

func TestCartographerBadges(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// No badge initially.
	if trail.GetBadge() != CartographerNone {
		t.Errorf("Expected no badge, got %d", trail.GetBadge())
	}

	// Add 5 discoveries -> Wanderer.
	for i := 0; i < 5; i++ {
		trail.DiscoverTerritory("t" + string(rune('a'+i)))
	}
	if trail.GetBadge() != CartographerBadgeWanderer {
		t.Errorf("Expected Wanderer badge at 5, got %d", trail.GetBadge())
	}

	// Add to 20 -> Pathfinder.
	for i := 5; i < 20; i++ {
		trail.DiscoverTerritory("t" + string(rune('a'+i)))
	}
	if trail.GetBadge() != CartographerBadgePathfinder {
		t.Errorf("Expected Pathfinder badge at 20, got %d", trail.GetBadge())
	}

	// Add to 50 -> Cartographer.
	for i := 20; i < 50; i++ {
		trail.DiscoverTerritory("t" + string(rune(i)))
	}
	if trail.GetBadge() != CartographerBadgeMaster {
		t.Errorf("Expected Cartographer badge at 50, got %d", trail.GetBadge())
	}
}

func TestCartographerBadgeString(t *testing.T) {
	tests := []struct {
		badge    CartographerBadge
		expected string
	}{
		{CartographerNone, ""},
		{CartographerBadgeWanderer, "Wanderer"},
		{CartographerBadgePathfinder, "Pathfinder"},
		{CartographerBadgeMaster, "Cartographer"},
	}

	for _, tc := range tests {
		if tc.badge.String() != tc.expected {
			t.Errorf("Badge %d: expected '%s', got '%s'", tc.badge, tc.expected, tc.badge.String())
		}
	}
}

func TestCartographerNextMilestone(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// At 0, next is Wanderer (5 needed).
	badge, needed := trail.GetNextMilestone()
	if badge != CartographerBadgeWanderer || needed != 5 {
		t.Errorf("Expected Wanderer/5, got %d/%d", badge, needed)
	}

	// At 3, need 2 more for Wanderer.
	for i := 0; i < 3; i++ {
		trail.DiscoverTerritory("t" + string(rune(i)))
	}
	badge, needed = trail.GetNextMilestone()
	if badge != CartographerBadgeWanderer || needed != 2 {
		t.Errorf("Expected Wanderer/2, got %d/%d", badge, needed)
	}

	// At 5, next is Pathfinder (15 needed).
	for i := 3; i < 5; i++ {
		trail.DiscoverTerritory("t" + string(rune(i)))
	}
	badge, needed = trail.GetNextMilestone()
	if badge != CartographerBadgePathfinder || needed != 15 {
		t.Errorf("Expected Pathfinder/15, got %d/%d", badge, needed)
	}

	// At 50, no next milestone.
	for i := 5; i < 50; i++ {
		trail.DiscoverTerritory("t" + string(rune(i+100)))
	}
	badge, needed = trail.GetNextMilestone()
	if badge != CartographerNone || needed != 0 {
		t.Errorf("Expected None/0 at max, got %d/%d", badge, needed)
	}
}

func TestCartographerGetDiscoveries(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	trail.DiscoverTerritory("alpha")
	trail.DiscoverTerritory("beta")

	discoveries := trail.GetDiscoveries()
	if len(discoveries) != 2 {
		t.Errorf("Expected 2 discoveries, got %d", len(discoveries))
	}

	// Verify it's a copy (modifying shouldn't affect original).
	discoveries[0].TerritoryHash = "modified"
	origDiscoveries := trail.GetDiscoveries()
	if origDiscoveries[0].TerritoryHash == "modified" {
		t.Error("GetDiscoveries should return a copy")
	}
}

func TestCartographerManager(t *testing.T) {
	manager := NewCartographerManager()

	if manager.Count() != 0 {
		t.Errorf("Expected 0 trails, got %d", manager.Count())
	}

	key1 := [32]byte{1}
	key2 := [32]byte{2}

	// Record discovery creates trail.
	manager.RecordDiscovery(key1, "territory-1")
	if manager.Count() != 1 {
		t.Errorf("Expected 1 trail, got %d", manager.Count())
	}

	// Same key, different territory.
	manager.RecordDiscovery(key1, "territory-2")
	if manager.Count() != 1 {
		t.Errorf("Expected still 1 trail, got %d", manager.Count())
	}

	// Different key.
	manager.RecordDiscovery(key2, "territory-1")
	if manager.Count() != 2 {
		t.Errorf("Expected 2 trails, got %d", manager.Count())
	}
}

func TestCartographerManagerGetScore(t *testing.T) {
	manager := NewCartographerManager()

	key := [32]byte{5}

	// No trail -> score 0.
	if manager.GetScore(key) != 0 {
		t.Error("Score should be 0 for unknown specter")
	}

	// Add discoveries.
	for i := 0; i < 5; i++ {
		manager.RecordDiscovery(key, "t"+string(rune(i)))
	}

	score := manager.GetScore(key)
	// Score = 6 * ln(1 + 5) = 6 * ln(6) ≈ 10.75
	if score < 10 || score > 11 {
		t.Errorf("Expected score ~10.75, got %f", score)
	}
}

func TestCartographerManagerGetBadge(t *testing.T) {
	manager := NewCartographerManager()

	key := [32]byte{6}

	// No trail -> no badge.
	if manager.GetBadge(key) != CartographerNone {
		t.Error("Badge should be None for unknown specter")
	}

	// Add 5 discoveries -> Wanderer.
	for i := 0; i < 5; i++ {
		manager.RecordDiscovery(key, "t"+string(rune(i)))
	}

	if manager.GetBadge(key) != CartographerBadgeWanderer {
		t.Errorf("Expected Wanderer badge, got %d", manager.GetBadge(key))
	}
}

func TestCartographerGetOrCreateTrail(t *testing.T) {
	manager := NewCartographerManager()

	key := [32]byte{7}

	trail1 := manager.GetOrCreateTrail(key)
	if trail1 == nil {
		t.Fatal("GetOrCreateTrail should create trail")
	}

	trail1.DiscoverTerritory("test")

	// Get same trail.
	trail2 := manager.GetOrCreateTrail(key)
	if trail2.Count() != 1 {
		t.Error("Should return same trail instance")
	}
}

func TestCartographerRecentDiscoveries(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// Add discoveries.
	trail.DiscoverTerritory("recent")

	recent := trail.GetRecentDiscoveries()
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent discovery, got %d", len(recent))
	}

	recentCount := trail.CountRecent()
	if recentCount != 1 {
		t.Errorf("Expected CountRecent 1, got %d", recentCount)
	}
}

func TestCartographerGarbageCollect(t *testing.T) {
	trail := NewCartographerTrail([32]byte{})

	// Add discovery.
	trail.DiscoverTerritory("test")

	// GC should not remove recent discovery.
	removed := trail.GarbageCollect()
	if removed != 0 {
		t.Errorf("Expected 0 removed, got %d", removed)
	}
	if trail.Count() != 1 {
		t.Errorf("Expected 1 discovery after GC, got %d", trail.Count())
	}
}

func TestCartographerManagerGarbageCollectAll(t *testing.T) {
	manager := NewCartographerManager()

	key1 := [32]byte{1}
	key2 := [32]byte{2}

	manager.RecordDiscovery(key1, "t1")
	manager.RecordDiscovery(key2, "t2")

	// GC should return 0 for recent discoveries.
	removed := manager.GarbageCollectAll()
	if removed != 0 {
		t.Errorf("Expected 0 removed, got %d", removed)
	}
}

func TestCartographerConstants(t *testing.T) {
	// Verify constants match spec.
	if CartographerWindow != 90*24*time.Hour {
		t.Error("CartographerWindow should be 90 days")
	}
	if CartographerWanderer != 5 {
		t.Error("CartographerWanderer should be 5")
	}
	if CartographerPathfinder != 20 {
		t.Error("CartographerPathfinder should be 20")
	}
	if CartographerMaster != 50 {
		t.Error("CartographerMaster should be 50")
	}
}
