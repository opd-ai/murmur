// Package overlays - Sigil Forge overlay stub for noebiten builds.
// Per ROADMAP.md line 476: "Pulse Map visualization — anvil-and-flame icon
// with orbiting entries".
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"image/color"
	"sync"
	"time"
)

// ForgeType represents the type of Sigil Forge event.
type ForgeType uint8

const (
	ForgeSigilArt   ForgeType = iota // Sigil art creation challenge.
	ForgeMicroFic                    // Micro-fiction writing challenge.
	ForgeRemixChain                  // Collaborative remix chain.
)

// ForgeState represents the current state of a forge event.
type ForgeState uint8

const (
	ForgeActive    ForgeState = iota // Accepting submissions.
	ForgeEvaluate                    // Evaluation period.
	ForgeCompleted                   // Event finished, winner determined.
)

// ForgeEntry represents a submission to a Sigil Forge.
type ForgeEntry struct {
	EntryID    [32]byte   // Unique entry identifier.
	SpecterKey [32]byte   // Submitter's Specter key.
	SigilColor color.RGBA // Color derived from entry sigil.
	Score      float64    // Amplification score.
	IsWinner   bool       // True if this entry won.
}

// ForgeEventInfo contains information about a Sigil Forge event.
type ForgeEventInfo struct {
	ForgeID   [32]byte   // Unique forge identifier.
	Type      ForgeType  // Type of forge event.
	State     ForgeState // Current state.
	X, Y      float64    // Position on the Pulse Map.
	StartTime time.Time  // When the forge started.
	EndTime   time.Time  // When submissions close.
	Entries   []ForgeEntry
}

// ForgeOverlay renders Sigil Forge events on the Pulse Map.
// Stub implementation for noebiten builds.
type ForgeOverlay struct {
	mu      sync.RWMutex
	visible bool
	forges  map[[32]byte]*ForgeEventInfo
	time    float64
}

// NewForgeOverlay creates a new Sigil Forge overlay.
func NewForgeOverlay() *ForgeOverlay {
	return &ForgeOverlay{
		visible: true,
		forges:  make(map[[32]byte]*ForgeEventInfo),
	}
}

// SetVisible controls visibility.
func (o *ForgeOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *ForgeOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetForge adds or updates a forge event.
func (o *ForgeOverlay) SetForge(forge *ForgeEventInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.forges[forge.ForgeID] = forge
}

// RemoveForge removes a forge event.
func (o *ForgeOverlay) RemoveForge(forgeID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.forges, forgeID)
}

// GetForge returns a forge event by ID.
func (o *ForgeOverlay) GetForge(forgeID [32]byte) *ForgeEventInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.forges[forgeID]
}

// GetAllForges returns all active forge events.
func (o *ForgeOverlay) GetAllForges() []*ForgeEventInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	forges := make([]*ForgeEventInfo, 0, len(o.forges))
	for _, f := range o.forges {
		forges = append(forges, f)
	}
	return forges
}

// Update advances animation state.
func (o *ForgeOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.time += dt
}

// ForgeCount returns the number of active forge events.
func (o *ForgeOverlay) ForgeCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.forges)
}

// ClearCompleted removes all completed forge events.
func (o *ForgeOverlay) ClearCompleted() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	var removed int
	for id, forge := range o.forges {
		if forge.State == ForgeCompleted && time.Since(forge.EndTime) > 24*time.Hour {
			delete(o.forges, id)
			removed++
		}
	}
	return removed
}

// ForgeTypeString returns human-readable name for a forge type.
func ForgeTypeString(ft ForgeType) string {
	switch ft {
	case ForgeSigilArt:
		return "Sigil Art"
	case ForgeMicroFic:
		return "Micro-Fiction"
	case ForgeRemixChain:
		return "Remix Chain"
	default:
		return "Unknown"
	}
}

// ForgeStateString returns human-readable name for a forge state.
func ForgeStateString(fs ForgeState) string {
	switch fs {
	case ForgeActive:
		return "Active"
	case ForgeEvaluate:
		return "Evaluating"
	case ForgeCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}
