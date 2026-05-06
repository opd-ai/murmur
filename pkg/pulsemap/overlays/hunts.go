// Package overlays - Specter Hunts Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Active Hunts are shown as dim, pulsing markers
// scattered across the topology. As fragments are claimed, their markers brighten
// and display the claimer's Specter sigil".
// Per ROADMAP.md line 639: "Specter Hunts — scattered glowing fragments".
//

//go:build !test
// +build !test

package overlays

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering/effects"
)

// HuntInfo contains information about a Specter Hunt event.
type HuntInfo struct {
	HuntID      [32]byte   // Unique hunt identifier.
	State       HuntStatus // Current state (Active/Expiring/Completed/Expired).
	X, Y        float64    // Position on the Pulse Map (world coordinates).
	StartTime   time.Time  // When the hunt was created.
	EndTime     time.Time  // When the hunt expires.
	Fragments   []FragmentInfo
	Leaderboard map[[32]byte]int // Specter key -> fragment count.
}

// FragmentInfo represents a single hunt fragment.
type FragmentInfo struct {
	FragmentID [32]byte       // Unique fragment identifier.
	Index      int            // Fragment index in hunt (0-N).
	X, Y       float64        // Position on the Pulse Map (world coordinates).
	State      FragmentStatus // Unclaimed/Claimed/Expired.
	ClaimerKey [32]byte       // Specter public key of claimer (if claimed).
	ClueLevel  int            // Number of clues revealed (0-3).
}

// HuntStatus represents the overall state of a hunt.
type HuntStatus uint8

const (
	HuntActive    HuntStatus = iota // Normal operation.
	HuntExpiring                    // Warning pulse (< 5 min left).
	HuntCompleted                   // Victory animation.
	HuntExpired                     // All faded.
)

// FragmentStatus represents the claim state of a fragment.
type FragmentStatus uint8

const (
	FragmentUnclaimed FragmentStatus = iota // Dim pulsing marker.
	FragmentClaimed                         // Bright with claimer sigil.
	FragmentExpired                         // Hunt expired, faded.
)

// HuntsOverlay renders Specter Hunt events on the Pulse Map.
type HuntsOverlay struct {
	mu sync.RWMutex

	visible bool
	hunts   map[[32]byte]*HuntInfo

	// Effects renderer for actual drawing.
	effects *effects.HuntEffects
}

// NewHuntsOverlay creates a new Specter Hunts overlay.
func NewHuntsOverlay() *HuntsOverlay {
	return &HuntsOverlay{
		visible: true,
		hunts:   make(map[[32]byte]*HuntInfo),
		effects: effects.NewHuntEffects(),
	}
}

// SetVisible controls visibility.
func (o *HuntsOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *HuntsOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetHunt adds or updates a hunt event.
func (o *HuntsOverlay) SetHunt(hunt *HuntInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.hunts[hunt.HuntID] = hunt

	// Update hunt state in effects renderer.
	o.effects.SetHuntState(hunt.HuntID, mapHuntStatus(hunt.State))

	// Update all fragments for this hunt.
	for _, frag := range hunt.Fragments {
		o.effects.AddFragment(&effects.FragmentVisual{
			ID:         frag.FragmentID,
			HuntID:     hunt.HuntID,
			Index:      frag.Index,
			X:          0, // Will be set in Draw with screen coordinates.
			Y:          0,
			State:      mapFragmentStatus(frag.State),
			ClaimerKey: frag.ClaimerKey,
			ClueLevel:  frag.ClueLevel,
		})
	}
}

// RemoveHunt removes a hunt event.
func (o *HuntsOverlay) RemoveHunt(huntID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if hunt, ok := o.hunts[huntID]; ok {
		// Remove all fragments.
		for _, frag := range hunt.Fragments {
			o.effects.RemoveFragment(frag.FragmentID)
		}
		delete(o.hunts, huntID)
	}
}

// GetHunt returns a hunt event by ID.
func (o *HuntsOverlay) GetHunt(huntID [32]byte) *HuntInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.hunts[huntID]
}

// GetAllHunts returns all active hunt events.
func (o *HuntsOverlay) GetAllHunts() []*HuntInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	hunts := make([]*HuntInfo, 0, len(o.hunts))
	for _, h := range o.hunts {
		hunts = append(hunts, h)
	}
	return hunts
}

// ClaimFragment marks a fragment as claimed.
func (o *HuntsOverlay) ClaimFragment(fragID, claimerKey [32]byte, sigil *ebiten.Image) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Update in hunts map.
	for _, hunt := range o.hunts {
		for i := range hunt.Fragments {
			if hunt.Fragments[i].FragmentID == fragID {
				hunt.Fragments[i].State = FragmentClaimed
				hunt.Fragments[i].ClaimerKey = claimerKey
				break
			}
		}
	}

	// Update in effects renderer.
	o.effects.ClaimFragment(fragID, claimerKey, sigil)
}

// RevealClue increases the clue level for a fragment.
func (o *HuntsOverlay) RevealClue(fragID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Update in hunts map.
	for _, hunt := range o.hunts {
		if o.revealClueInHunt(hunt, fragID) {
			break
		}
	}

	// Update in effects renderer.
	o.effects.RevealClue(fragID)
}

// revealClueInHunt increments clue level for fragment in hunt. Returns true if found.
func (o *HuntsOverlay) revealClueInHunt(hunt *HuntInfo, fragID [32]byte) bool {
	for i := range hunt.Fragments {
		if hunt.Fragments[i].FragmentID != fragID {
			continue
		}
		if hunt.Fragments[i].ClueLevel < 3 {
			hunt.Fragments[i].ClueLevel++
		}
		return true
	}
	return false
}

// Update advances animation state.
func (o *HuntsOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.effects.Update(float32(dt))
}

// Draw renders the hunt events.
func (o *HuntsOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	screenW, screenH, centerX, centerY := getCameraSetup(screen)
	o.updateFragmentPositions(screenW, screenH, cameraX, cameraY, centerX, centerY, zoom)
}

func (o *HuntsOverlay) updateFragmentPositions(screenW, screenH, cameraX, cameraY, centerX, centerY, zoom float64) {
	for _, hunt := range o.hunts {
		for _, frag := range hunt.Fragments {
			sx, sy := worldToScreen(frag.X, frag.Y, cameraX, cameraY, centerX, centerY, zoom)

			if isOffScreen(sx, sy, screenW, screenH, 100) {
				continue
			}

			o.addFragmentToEffects(hunt.HuntID, frag)
		}
	}
}

func (o *HuntsOverlay) addFragmentToEffects(huntID [32]byte, frag FragmentInfo) {
	o.effects.AddFragment(&effects.FragmentVisual{
		ID:         frag.FragmentID,
		HuntID:     huntID,
		Index:      frag.Index,
		X:          float32(frag.X),
		Y:          float32(frag.Y),
		State:      mapFragmentStatus(frag.State),
		ClaimerKey: frag.ClaimerKey,
		ClueLevel:  frag.ClueLevel,
	})
}

// mapHuntStatus converts overlay HuntStatus to effects HuntState.
func mapHuntStatus(s HuntStatus) effects.HuntState {
	switch s {
	case HuntActive:
		return effects.HuntStateActive
	case HuntExpiring:
		return effects.HuntStateExpiring
	case HuntCompleted:
		return effects.HuntStateCompleted
	case HuntExpired:
		return effects.HuntStateExpired
	default:
		return effects.HuntStateActive
	}
}

// mapFragmentStatus converts overlay FragmentStatus to effects FragmentState.
func mapFragmentStatus(s FragmentStatus) effects.FragmentState {
	switch s {
	case FragmentUnclaimed:
		return effects.FragmentUnclaimed
	case FragmentClaimed:
		return effects.FragmentClaimed
	case FragmentExpired:
		return effects.FragmentExpired
	default:
		return effects.FragmentUnclaimed
	}
}
