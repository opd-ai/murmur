// Package overlays - Stub for Specter Hunts overlay (test build).
//

//go:build test
// +build test

package overlays

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// HuntInfo holds the display data for a single Specter Hunt shown on the Pulse Map overlay.
type HuntInfo struct {
	HuntID      [32]byte
	State       HuntStatus
	X, Y        float64
	StartTime   time.Time
	EndTime     time.Time
	Fragments   []FragmentInfo
	Leaderboard map[[32]byte]int
}

// FragmentInfo holds the display data for a single hunt fragment on the overlay.
type FragmentInfo struct {
	FragmentID [32]byte
	Index      int
	X, Y       float64
	State      FragmentStatus
	ClaimerKey [32]byte
	ClueLevel  int
}

// HuntStatus represents the lifecycle phase of a Specter Hunt for overlay rendering.
type HuntStatus uint8

const (
	HuntActive HuntStatus = iota
	HuntExpiring
	HuntCompleted
	HuntExpired
)

// FragmentStatus represents the claim state of a single hunt fragment.
type FragmentStatus uint8

const (
	FragmentUnclaimed FragmentStatus = iota
	FragmentClaimed
	FragmentExpired
)

// HuntsOverlay renders active Specter Hunt fragments as map-pinned icons on the Pulse Map.
type HuntsOverlay struct {
	visible bool
	hunts   map[[32]byte]*HuntInfo
}

// NewHuntsOverlay creates an empty, visible HuntsOverlay.
func NewHuntsOverlay() *HuntsOverlay {
	return &HuntsOverlay{
		visible: true,
		hunts:   make(map[[32]byte]*HuntInfo),
	}
}

func (o *HuntsOverlay) SetVisible(visible bool) {
	o.visible = visible
}

func (o *HuntsOverlay) IsVisible() bool {
	return o.visible
}

func (o *HuntsOverlay) SetHunt(hunt *HuntInfo) {
	o.hunts[hunt.HuntID] = hunt
}

func (o *HuntsOverlay) RemoveHunt(huntID [32]byte) {
	delete(o.hunts, huntID)
}

func (o *HuntsOverlay) GetHunt(huntID [32]byte) *HuntInfo {
	return o.hunts[huntID]
}

func (o *HuntsOverlay) GetAllHunts() []*HuntInfo {
	hunts := make([]*HuntInfo, 0, len(o.hunts))
	for _, h := range o.hunts {
		hunts = append(hunts, h)
	}
	return hunts
}

func (o *HuntsOverlay) ClaimFragment(fragID, claimerKey [32]byte, sigil *ebiten.Image) {}

func (o *HuntsOverlay) RevealClue(fragID [32]byte) {}

func (o *HuntsOverlay) Update(dt float64) {}

func (o *HuntsOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {}
