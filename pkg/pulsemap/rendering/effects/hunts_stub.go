// Package effects provides hunt fragment visualization for the Pulse Map.
// This stub file provides type definitions without Ebitengine dependencies.
//
//go:build test
// +build test

package effects

import "sync"

// FragmentState represents the claim state of a hunt fragment.
type FragmentState int

const (
	FragmentUnclaimed FragmentState = iota // Dim pulsing marker.
	FragmentClaimed                        // Bright with claimer sigil.
	FragmentExpired                        // Hunt expired, faded.
)

// HuntState represents the overall state of a hunt.
type HuntState int

const (
	HuntStateActive    HuntState = iota // Normal operation.
	HuntStateExpiring                   // Warning pulse (< 5 min left).
	HuntStateCompleted                  // Victory animation.
	HuntStateExpired                    // All faded.
)

// FragmentVisual represents a hunt fragment to be rendered on the Pulse Map.
type FragmentVisual struct {
	ID           [32]byte      // Fragment identifier.
	HuntID       [32]byte      // Parent hunt.
	Index        int           // Fragment index in hunt.
	X, Y         float32       // Position in screen coordinates.
	State        FragmentState // Claim state.
	ClaimerSigil interface{}   // Placeholder for sigil (no Ebitengine).
	ClaimerKey   [32]byte      // Claimer's public key.
	ClueLevel    int           // Number of clues revealed (0-3).
}

// HuntEffects renders hunt fragment visualizations on the Pulse Map.
type HuntEffects struct {
	mu        sync.RWMutex
	time      float32
	fragments map[[32]byte]*FragmentVisual
	hunts     map[[32]byte]HuntState
}

// NewHuntEffects creates a new hunt effects renderer.
func NewHuntEffects() *HuntEffects {
	return &HuntEffects{
		fragments: make(map[[32]byte]*FragmentVisual),
		hunts:     make(map[[32]byte]HuntState),
	}
}

// Update advances animation time.
func (h *HuntEffects) Update(dt float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.time += dt
}

// AddFragment adds a fragment visual to the renderer.
func (h *HuntEffects) AddFragment(frag *FragmentVisual) {
	if frag == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fragments[frag.ID] = frag
}

// RemoveFragment removes a fragment visual.
func (h *HuntEffects) RemoveFragment(id [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.fragments, id)
}

// SetHuntState sets the state for an entire hunt.
func (h *HuntEffects) SetHuntState(huntID [32]byte, state HuntState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hunts[huntID] = state
}

// GetHuntState returns the current state of a hunt.
func (h *HuntEffects) GetHuntState(huntID [32]byte) HuntState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hunts[huntID]
}

// ClaimFragment marks a fragment as claimed.
func (h *HuntEffects) ClaimFragment(fragID, claimerKey [32]byte, sigil interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if frag, ok := h.fragments[fragID]; ok {
		frag.State = FragmentClaimed
		frag.ClaimerKey = claimerKey
		frag.ClaimerSigil = sigil
	}
}

// RevealClue increases the clue level for a fragment.
func (h *HuntEffects) RevealClue(fragID [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if frag, ok := h.fragments[fragID]; ok {
		if frag.ClueLevel < 3 {
			frag.ClueLevel++
		}
	}
}

// GetFragment returns a fragment visual by ID.
func (h *HuntEffects) GetFragment(id [32]byte) *FragmentVisual {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.fragments[id]
}

// FragmentCount returns the number of tracked fragments.
func (h *HuntEffects) FragmentCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.fragments)
}

// HuntCount returns the number of tracked hunts.
func (h *HuntEffects) HuntCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.hunts)
}

// ClearHunt removes all fragments for a hunt.
func (h *HuntEffects) ClearHunt(huntID [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, frag := range h.fragments {
		if frag.HuntID == huntID {
			delete(h.fragments, id)
		}
	}
	delete(h.hunts, huntID)
}
