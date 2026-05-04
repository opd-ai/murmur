// Package overlays - Tests for Specter Hunts overlay.
//

//go:build test
// +build test

package overlays

import (
	"testing"
	"time"
)

func TestNewHuntsOverlay(t *testing.T) {
	overlay := NewHuntsOverlay()
	if overlay == nil {
		t.Fatal("NewHuntsOverlay returned nil")
	}
}

func TestHuntsOverlay_SetVisible(t *testing.T) {
	overlay := NewHuntsOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("Expected IsVisible()=false after SetVisible(false)")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("Expected IsVisible()=true after SetVisible(true)")
	}
}

func TestHuntsOverlay_SetHunt(t *testing.T) {
	overlay := NewHuntsOverlay()

	huntID := [32]byte{1, 2, 3}
	hunt := &HuntInfo{
		HuntID:    huntID,
		State:     HuntActive,
		X:         500.0,
		Y:         600.0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(2 * time.Hour),
		Fragments: []FragmentInfo{
			{
				FragmentID: [32]byte{10, 11, 12},
				Index:      0,
				X:          100.0,
				Y:          200.0,
				State:      FragmentUnclaimed,
				ClueLevel:  0,
			},
			{
				FragmentID: [32]byte{13, 14, 15},
				Index:      1,
				X:          300.0,
				Y:          400.0,
				State:      FragmentClaimed,
				ClaimerKey: [32]byte{99, 98, 97},
				ClueLevel:  2,
			},
		},
		Leaderboard: map[[32]byte]int{
			{99, 98, 97}: 1,
		},
	}

	overlay.SetHunt(hunt)

	retrieved := overlay.GetHunt(huntID)
	if retrieved == nil {
		t.Error("GetHunt returned nil after SetHunt")
	}
}

func TestHuntsOverlay_RemoveHunt(t *testing.T) {
	overlay := NewHuntsOverlay()

	huntID := [32]byte{4, 5, 6}
	hunt := &HuntInfo{
		HuntID: huntID,
		State:  HuntCompleted,
		X:      700.0,
		Y:      800.0,
	}

	overlay.SetHunt(hunt)
	overlay.RemoveHunt(huntID)

	retrieved := overlay.GetHunt(huntID)
	if retrieved != nil {
		t.Error("GetHunt should return nil after RemoveHunt")
	}
}

func TestHuntsOverlay_GetAllHunts(t *testing.T) {
	overlay := NewHuntsOverlay()

	hunt1 := &HuntInfo{
		HuntID: [32]byte{7, 8, 9},
		State:  HuntActive,
		X:      900.0,
		Y:      1000.0,
	}

	hunt2 := &HuntInfo{
		HuntID: [32]byte{10, 11, 12},
		State:  HuntExpiring,
		X:      1100.0,
		Y:      1200.0,
	}

	overlay.SetHunt(hunt1)
	overlay.SetHunt(hunt2)

	all := overlay.GetAllHunts()
	if len(all) != 2 {
		t.Errorf("Expected 2 hunts, got %d", len(all))
	}
}

func TestHuntsOverlay_ClaimFragment(t *testing.T) {
	overlay := NewHuntsOverlay()

	fragID := [32]byte{20, 21, 22}
	claimerKey := [32]byte{50, 51, 52}

	// Should not panic even if fragment doesn't exist.
	overlay.ClaimFragment(fragID, claimerKey, nil)
}

func TestHuntsOverlay_RevealClue(t *testing.T) {
	overlay := NewHuntsOverlay()

	fragID := [32]byte{30, 31, 32}

	// Should not panic even if fragment doesn't exist.
	overlay.RevealClue(fragID)
}

func TestHuntsOverlay_Update(t *testing.T) {
	overlay := NewHuntsOverlay()

	// Should not panic.
	overlay.Update(0.016)
	overlay.Update(0.033)
}
