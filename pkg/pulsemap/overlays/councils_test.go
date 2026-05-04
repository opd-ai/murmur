// Package overlays — Phantom Council overlay tests.
//
//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewCouncilOverlay(t *testing.T) {
	overlay := NewCouncilOverlay()

	if overlay == nil {
		t.Fatal("NewCouncilOverlay returned nil")
	}
	if !overlay.Visible {
		t.Error("expected overlay to be visible by default")
	}
	if overlay.Opacity != 0.8 {
		t.Errorf("expected opacity 0.8, got %f", overlay.Opacity)
	}
	if overlay.CouncilCount() != 0 {
		t.Errorf("expected 0 councils, got %d", overlay.CouncilCount())
	}
}

func TestCouncilOverlay_AddRemove(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{1, 2, 3}
	info := NewCouncilInfo(id, "Test Council")
	info.AddMember([32]byte{10}, 100, 200)
	info.AddMember([32]byte{20}, 150, 250)
	info.AddMember([32]byte{30}, 200, 300)

	overlay.AddCouncil(info)

	if overlay.CouncilCount() != 1 {
		t.Errorf("expected 1 council, got %d", overlay.CouncilCount())
	}

	retrieved := overlay.GetCouncil(id)
	if retrieved == nil {
		t.Fatal("GetCouncil returned nil for added council")
	}
	if retrieved.Name != "Test Council" {
		t.Errorf("expected name 'Test Council', got '%s'", retrieved.Name)
	}
	if len(retrieved.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(retrieved.Members))
	}

	// Remove council.
	overlay.RemoveCouncil(id)
	if overlay.CouncilCount() != 0 {
		t.Errorf("expected 0 councils after removal, got %d", overlay.CouncilCount())
	}
	if overlay.GetCouncil(id) != nil {
		t.Error("expected nil after removal")
	}
}

func TestCouncilOverlay_Update(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{1, 2, 3}
	info := NewCouncilInfo(id, "Test Council")
	info.AddMember([32]byte{10}, 100, 200)
	info.AddMember([32]byte{20}, 150, 250)
	info.IsActive = true

	overlay.AddCouncil(info)

	// Initial phase should be 0.
	retrieved := overlay.GetCouncil(id)
	initialPhase := retrieved.AnimationPhase

	// Update with dt.
	overlay.Update(0.1)

	// Phase should have advanced.
	retrieved = overlay.GetCouncil(id)
	if retrieved.AnimationPhase == initialPhase {
		t.Error("expected animation phase to advance after Update")
	}
}

func TestCouncilOverlay_SetActive(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{5, 6, 7}
	info := NewCouncilInfo(id, "Active Test")
	info.AddMember([32]byte{10}, 0, 0)
	overlay.AddCouncil(info)

	// Initially not active.
	if overlay.GetCouncil(id).IsActive {
		t.Error("expected council to not be active initially")
	}

	// Set active.
	overlay.SetCouncilActive(id, true)
	if !overlay.GetCouncil(id).IsActive {
		t.Error("expected council to be active after SetCouncilActive(true)")
	}

	// Set inactive.
	overlay.SetCouncilActive(id, false)
	if overlay.GetCouncil(id).IsActive {
		t.Error("expected council to not be active after SetCouncilActive(false)")
	}
}

func TestCouncilOverlay_SetMemberCommunicating(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{8, 9, 10}
	memberKey := [32]byte{100}
	info := NewCouncilInfo(id, "Comm Test")
	info.AddMember(memberKey, 50, 50)
	overlay.AddCouncil(info)

	// Initially not communicating.
	retrieved := overlay.GetCouncil(id)
	if retrieved.Members[0].IsCommunicating {
		t.Error("expected member to not be communicating initially")
	}

	// Set communicating.
	overlay.SetMemberCommunicating(id, memberKey, true)
	retrieved = overlay.GetCouncil(id)
	if !retrieved.Members[0].IsCommunicating {
		t.Error("expected member to be communicating after SetMemberCommunicating(true)")
	}
}

func TestCouncilOverlay_UpdateMemberPosition(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{11, 12, 13}
	memberKey := [32]byte{200}
	info := NewCouncilInfo(id, "Position Test")
	info.AddMember(memberKey, 0, 0)
	overlay.AddCouncil(info)

	overlay.UpdateMemberPosition(id, memberKey, 500, 600)

	retrieved := overlay.GetCouncil(id)
	if retrieved.Members[0].X != 500 || retrieved.Members[0].Y != 600 {
		t.Errorf("expected position (500, 600), got (%f, %f)",
			retrieved.Members[0].X, retrieved.Members[0].Y)
	}
}

func TestCouncilOverlay_Clear(t *testing.T) {
	overlay := NewCouncilOverlay()

	// Add multiple councils.
	for i := 0; i < 5; i++ {
		id := [32]byte{byte(i)}
		info := NewCouncilInfo(id, "Council")
		info.AddMember([32]byte{byte(i + 10)}, float64(i*100), float64(i*100))
		overlay.AddCouncil(info)
	}

	if overlay.CouncilCount() != 5 {
		t.Errorf("expected 5 councils, got %d", overlay.CouncilCount())
	}

	overlay.Clear()

	if overlay.CouncilCount() != 0 {
		t.Errorf("expected 0 councils after clear, got %d", overlay.CouncilCount())
	}
}

func TestCouncilOverlay_SetVisibility(t *testing.T) {
	overlay := NewCouncilOverlay()

	overlay.SetVisible(false)
	if overlay.Visible {
		t.Error("expected Visible to be false")
	}

	overlay.SetVisible(true)
	if !overlay.Visible {
		t.Error("expected Visible to be true")
	}
}

func TestCouncilOverlay_SetOpacity(t *testing.T) {
	overlay := NewCouncilOverlay()

	overlay.SetOpacity(0.5)
	if overlay.Opacity != 0.5 {
		t.Errorf("expected opacity 0.5, got %f", overlay.Opacity)
	}

	// Test clamping.
	overlay.SetOpacity(-0.1)
	if overlay.Opacity != 0 {
		t.Errorf("expected opacity 0 (clamped), got %f", overlay.Opacity)
	}

	overlay.SetOpacity(1.5)
	if overlay.Opacity != 1 {
		t.Errorf("expected opacity 1 (clamped), got %f", overlay.Opacity)
	}
}

func TestGenerateCouncilColor(t *testing.T) {
	// Test that different IDs produce different colors.
	id1 := [32]byte{0}
	id2 := [32]byte{100}
	id3 := [32]byte{200}

	c1 := GenerateCouncilColor(id1)
	c2 := GenerateCouncilColor(id2)
	c3 := GenerateCouncilColor(id3)

	// Colors should be valid (non-zero).
	if c1.A == 0 || c2.A == 0 || c3.A == 0 {
		t.Error("expected non-zero alpha values")
	}

	// Colors should be different (at least in one channel).
	if c1.R == c2.R && c1.G == c2.G && c1.B == c2.B {
		t.Error("expected different colors for different IDs")
	}

	// Colors should be in cool-tone range (200-280° hue means more blue/purple).
	// This is hard to test directly but we can verify it's not pure red.
	if c1.R > c1.B && c1.G < 50 {
		t.Error("expected cool-tone color, got warm tone")
	}
}

func TestNewCouncilInfo(t *testing.T) {
	id := [32]byte{1, 2, 3, 4}
	info := NewCouncilInfo(id, "My Council")

	if info.ID != id {
		t.Error("ID mismatch")
	}
	if info.Name != "My Council" {
		t.Errorf("expected name 'My Council', got '%s'", info.Name)
	}
	if len(info.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(info.Members))
	}
	if info.Color == (color.RGBA{}) {
		t.Error("expected non-zero color")
	}
}

func TestCouncilInfo_AddMember(t *testing.T) {
	info := NewCouncilInfo([32]byte{1}, "Test")

	info.AddMember([32]byte{10}, 100, 200)
	info.AddMember([32]byte{20}, 300, 400)

	if len(info.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(info.Members))
	}

	m1 := info.Members[0]
	if m1.SpecterKey != [32]byte{10} || m1.X != 100 || m1.Y != 200 {
		t.Error("first member data mismatch")
	}

	m2 := info.Members[1]
	if m2.SpecterKey != [32]byte{20} || m2.X != 300 || m2.Y != 400 {
		t.Error("second member data mismatch")
	}
}

func TestCouncilOverlay_Draw(t *testing.T) {
	overlay := NewCouncilOverlay()

	// Create a test council with members.
	id := [32]byte{1, 2, 3}
	info := NewCouncilInfo(id, "Test Council")
	info.AddMember([32]byte{10}, -100, -100)
	info.AddMember([32]byte{20}, 100, 100)
	info.AddMember([32]byte{30}, 0, 150)
	info.IsActive = true
	overlay.AddCouncil(info)

	// Create a test screen.
	screen := ebiten.NewImage(800, 600)

	// Draw should not panic.
	overlay.Draw(screen, 0, 0, 1.0)

	// Draw with different zoom levels.
	overlay.Draw(screen, 0, 0, 0.5)
	overlay.Draw(screen, 0, 0, 2.0)

	// Draw with camera offset.
	overlay.Draw(screen, 50, -50, 1.0)

	// Draw when not visible should be a no-op.
	overlay.SetVisible(false)
	overlay.Draw(screen, 0, 0, 1.0)
}

func TestCouncilOverlay_UpdateCouncil(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{1, 2, 3}
	info := NewCouncilInfo(id, "Original Name")
	info.AddMember([32]byte{10}, 0, 0)
	overlay.AddCouncil(info)

	// Update with new info.
	newInfo := NewCouncilInfo(id, "Updated Name")
	newInfo.AddMember([32]byte{10}, 100, 100)
	newInfo.AddMember([32]byte{20}, 200, 200) // New member.
	overlay.UpdateCouncil(newInfo)

	retrieved := overlay.GetCouncil(id)
	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", retrieved.Name)
	}
	if len(retrieved.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(retrieved.Members))
	}
}

func TestCouncilOverlay_Tick(t *testing.T) {
	overlay := NewCouncilOverlay()

	id := [32]byte{1}
	info := NewCouncilInfo(id, "Tick Test")
	info.AddMember([32]byte{10}, 0, 0)
	overlay.AddCouncil(info)

	// Tick should not panic.
	overlay.Tick()
	overlay.Tick()
	overlay.Tick()
}

func TestCouncilOverlay_NilSafety(t *testing.T) {
	overlay := NewCouncilOverlay()

	// Adding nil should not panic.
	overlay.AddCouncil(nil)
	overlay.UpdateCouncil(nil)

	// Getting non-existent council.
	if overlay.GetCouncil([32]byte{99}) != nil {
		t.Error("expected nil for non-existent council")
	}

	// Removing non-existent council.
	overlay.RemoveCouncil([32]byte{99})

	// Operations on non-existent council.
	overlay.SetCouncilActive([32]byte{99}, true)
	overlay.SetMemberCommunicating([32]byte{99}, [32]byte{1}, true)
	overlay.UpdateMemberPosition([32]byte{99}, [32]byte{1}, 0, 0)
}

func TestHSVtoRGB(t *testing.T) {
	tests := []struct {
		h, s, v float64
	}{
		{0, 1, 1},       // Red.
		{120, 1, 1},     // Green.
		{240, 1, 1},     // Blue.
		{200, 0.7, 0.8}, // Cool tone.
		{280, 0.6, 0.9}, // Purple.
	}

	for _, tc := range tests {
		r, g, b := hsvToRGB(tc.h, tc.s, tc.v)

		// Values should be in [0, 1].
		if r < 0 || r > 1 || g < 0 || g > 1 || b < 0 || b > 1 {
			t.Errorf("hsvToRGB(%f, %f, %f) = (%f, %f, %f): out of range",
				tc.h, tc.s, tc.v, r, g, b)
		}
	}
}
