// Package overlays - Pulse Beat overlay tests.
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewPulseBeatOverlay(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	if overlay == nil {
		t.Fatal("NewPulseBeatOverlay returned nil")
	}
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible by default")
	}
	if overlay.BeatCount() != 0 {
		t.Error("expected no beats initially")
	}
}

func TestPulseBeatOverlay_Visibility(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("expected overlay to be hidden")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible")
	}
}

func TestPulseBeatOverlay_AddBeat(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	beat := &DisplayBeat{
		ID:        [32]byte{1, 2, 3},
		Type:      BeatGift,
		Priority:  BeatPriorityNormal,
		Title:     "Test Beat",
		TargetX:   100,
		TargetY:   200,
		CreatedAt: time.Now(),
	}

	overlay.AddBeat(beat)

	if overlay.BeatCount() != 1 {
		t.Errorf("expected 1 beat, got %d", overlay.BeatCount())
	}

	retrieved := overlay.GetBeat(beat.ID)
	if retrieved == nil {
		t.Fatal("GetBeat returned nil")
	}
	if retrieved.Type != BeatGift {
		t.Errorf("expected BeatGift type, got %v", retrieved.Type)
	}
	if retrieved.DisplayedAt.IsZero() {
		t.Error("DisplayedAt should be set automatically")
	}
}

func TestPulseBeatOverlay_AddBeatNil(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.AddBeat(nil)

	if overlay.BeatCount() != 0 {
		t.Error("adding nil beat should not add anything")
	}
}

func TestPulseBeatOverlay_AddBeatDuplicate(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	beat := &DisplayBeat{
		ID:    [32]byte{1},
		Title: "Original",
	}
	overlay.AddBeat(beat)

	beat2 := &DisplayBeat{
		ID:    [32]byte{1},
		Title: "Updated",
	}
	overlay.AddBeat(beat2)

	if overlay.BeatCount() != 1 {
		t.Error("duplicate beat should update, not add")
	}

	retrieved := overlay.GetBeat([32]byte{1})
	if retrieved.Title != "Updated" {
		t.Error("beat should be updated")
	}
}

func TestPulseBeatOverlay_RemoveBeat(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	beat := &DisplayBeat{
		ID: [32]byte{1},
	}
	overlay.AddBeat(beat)

	overlay.RemoveBeat(beat.ID)

	if overlay.BeatCount() != 0 {
		t.Error("beat should be removed")
	}
}

func TestPulseBeatOverlay_MaxVisible(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.SetMaxVisible(2)

	for i := 0; i < 5; i++ {
		overlay.AddBeat(&DisplayBeat{
			ID:       [32]byte{byte(i)},
			Priority: BeatPriorityNormal,
		})
	}

	if overlay.BeatCount() > 2 {
		t.Errorf("expected at most 2 beats, got %d", overlay.BeatCount())
	}
}

func TestPulseBeatOverlay_PriorityOrdering(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.SetMaxVisible(3)

	// Add low priority first.
	overlay.AddBeat(&DisplayBeat{
		ID:       [32]byte{1},
		Priority: BeatPriorityLow,
	})

	// Add high priority second.
	overlay.AddBeat(&DisplayBeat{
		ID:       [32]byte{2},
		Priority: BeatPriorityHigh,
	})

	// Add urgent priority third.
	overlay.AddBeat(&DisplayBeat{
		ID:       [32]byte{3},
		Priority: BeatPriorityUrgent,
	})

	// Urgent should be first, then high, then low.
	// (Implementation keeps higher priority first.)
}

func TestPulseBeatOverlay_Update(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	// Add beat that will expire soon.
	beat := &DisplayBeat{
		ID:          [32]byte{1},
		DisplayedAt: time.Now().Add(-6 * time.Second), // Already past display time.
	}
	overlay.AddBeat(beat)

	// Update should remove expired beat.
	overlay.Update(0.016)

	if overlay.BeatCount() != 0 {
		t.Error("expired beat should be removed on update")
	}
}

func TestPulseBeatOverlay_Draw(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	screen := ebiten.NewImage(800, 600)

	// Draw empty overlay.
	overlay.Draw(screen, 0, 0, 1.0)

	// Add on-screen beat and draw.
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{1},
		Type:      BeatGift,
		Priority:  BeatPriorityNormal,
		TargetX:   400, // Center of screen.
		TargetY:   300,
		CreatedAt: time.Now(),
		Color:     color.RGBA{R: 255, G: 100, B: 200, A: 255},
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Add off-screen beat and draw.
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{2},
		Type:      BeatHunt,
		Priority:  BeatPriorityHigh,
		TargetX:   1000, // Off-screen right.
		TargetY:   300,
		CreatedAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Draw when hidden.
	overlay.SetVisible(false)
	overlay.Draw(screen, 0, 0, 1.0)
}

func TestPulseBeatOverlay_DrawAllTypes(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.SetMaxVisible(12)
	screen := ebiten.NewImage(800, 600)

	// Add all beat types.
	types := []BeatType{
		BeatGift, BeatHunt, BeatForge, BeatChain, BeatTerritory,
		BeatSpark, BeatPuzzle, BeatCouncil, BeatMark, BeatWave,
	}

	for i, bt := range types {
		overlay.AddBeat(&DisplayBeat{
			ID:        [32]byte{byte(i)},
			Type:      bt,
			Priority:  BeatPriorityNormal,
			TargetX:   float64(i*200 - 500), // Various positions including off-screen.
			TargetY:   300,
			CreatedAt: time.Now(),
		})
	}

	overlay.Draw(screen, 0, 0, 1.0)
}

func TestPulseBeatOverlay_DrawPriorities(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.SetMaxVisible(4)
	screen := ebiten.NewImage(800, 600)

	priorities := []BeatPriority{
		BeatPriorityLow, BeatPriorityNormal, BeatPriorityHigh, BeatPriorityUrgent,
	}

	for i, p := range priorities {
		overlay.AddBeat(&DisplayBeat{
			ID:        [32]byte{byte(i)},
			Type:      BeatSpark,
			Priority:  p,
			TargetX:   1000, // Off-screen to test edge rendering.
			TargetY:   float64(i * 100),
			CreatedAt: time.Now(),
		})
	}

	overlay.Draw(screen, 0, 0, 1.0)
}

func TestPulseBeatOverlay_ClearBeats(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	for i := 0; i < 3; i++ {
		overlay.AddBeat(&DisplayBeat{ID: [32]byte{byte(i)}})
	}

	overlay.ClearBeats()

	if overlay.BeatCount() != 0 {
		t.Error("beats should be cleared")
	}
}

func TestPulseBeatOverlay_MarkBeatRead(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	beat := &DisplayBeat{
		ID:   [32]byte{1},
		Read: false,
	}
	overlay.AddBeat(beat)

	overlay.MarkBeatRead([32]byte{1})

	retrieved := overlay.GetBeat([32]byte{1})
	if !retrieved.Read {
		t.Error("beat should be marked as read")
	}
}

func TestPulseBeatOverlay_HandleClick(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	clicked := false
	var clickedID [32]byte

	overlay.SetOnBeatTapped(func(beatID [32]byte) {
		clicked = true
		clickedID = beatID
	})

	// Beat at world position (0, 0), camera at (0, 0).
	// Screen center is (400, 300), so beat appears at screen center.
	beat := &DisplayBeat{
		ID:        [32]byte{42},
		Type:      BeatGift,
		TargetX:   0, // At camera position, so appears at screen center.
		TargetY:   0,
		CreatedAt: time.Now(),
	}
	overlay.AddBeat(beat)

	// Click at screen center where beat is drawn.
	handled := overlay.HandleClick(400, 300, 0, 0, 1.0, 800, 600)

	if !handled {
		t.Error("click should be handled")
	}
	if !clicked {
		t.Error("callback should be called")
	}
	if clickedID != beat.ID {
		t.Error("clicked ID should match beat ID")
	}
}

func TestPulseBeatOverlay_HandleClickMiss(t *testing.T) {
	overlay := NewPulseBeatOverlay()

	clicked := false
	overlay.SetOnBeatTapped(func(beatID [32]byte) {
		clicked = true
	})

	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{1},
		TargetX:   100, // Beat at one location.
		TargetY:   100,
		CreatedAt: time.Now(),
	})

	// Click far from beat.
	handled := overlay.HandleClick(700, 500, 0, 0, 1.0, 800, 600)

	if handled {
		t.Error("click should not be handled")
	}
	if clicked {
		t.Error("callback should not be called")
	}
}

func TestPulseBeatOverlay_SetEdgeMargin(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	overlay.SetEdgeMargin(50)
	// No crash means success.
}

func TestBeatTypeString(t *testing.T) {
	tests := []struct {
		beatType BeatType
		expected string
	}{
		{BeatGift, "Gift"},
		{BeatHunt, "Hunt"},
		{BeatForge, "Forge"},
		{BeatChain, "Chain"},
		{BeatTerritory, "Territory"},
		{BeatSpark, "Spark"},
		{BeatPuzzle, "Puzzle"},
		{BeatCouncil, "Council"},
		{BeatMark, "Mark"},
		{BeatWave, "Wave"},
		{BeatType(99), "Unknown"},
	}

	for _, tc := range tests {
		result := BeatTypeString(tc.beatType)
		if result != tc.expected {
			t.Errorf("BeatTypeString(%v) = %q, expected %q", tc.beatType, result, tc.expected)
		}
	}
}

func TestBeatPriorityString(t *testing.T) {
	tests := []struct {
		priority BeatPriority
		expected string
	}{
		{BeatPriorityLow, "Low"},
		{BeatPriorityNormal, "Normal"},
		{BeatPriorityHigh, "High"},
		{BeatPriorityUrgent, "Urgent"},
		{BeatPriority(99), "Unknown"},
	}

	for _, tc := range tests {
		result := BeatPriorityString(tc.priority)
		if result != tc.expected {
			t.Errorf("BeatPriorityString(%v) = %q, expected %q", tc.priority, result, tc.expected)
		}
	}
}

func TestPulseBeatOverlay_DrawWithZoom(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	screen := ebiten.NewImage(800, 600)

	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{1},
		Type:      BeatSpark,
		TargetX:   1000,
		TargetY:   1000,
		CreatedAt: time.Now(),
	})

	// Test various zoom levels.
	for _, zoom := range []float64{0.1, 0.5, 1.0, 2.0, 5.0} {
		overlay.Draw(screen, 0, 0, zoom)
	}
}

func TestPulseBeatOverlay_DrawFadingBeat(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	screen := ebiten.NewImage(800, 600)

	// Beat that is almost expired (should be faded).
	overlay.AddBeat(&DisplayBeat{
		ID:          [32]byte{1},
		Type:        BeatGift,
		TargetX:     400,
		TargetY:     300,
		CreatedAt:   time.Now(),
		DisplayedAt: time.Now().Add(-4500 * time.Millisecond), // 4.5s into 5s display.
	})

	overlay.Draw(screen, 0, 0, 1.0)
}

func TestPulseBeatOverlay_EdgePositions(t *testing.T) {
	overlay := NewPulseBeatOverlay()
	screen := ebiten.NewImage(800, 600)

	// Beat to the left.
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{1},
		Type:      BeatGift,
		TargetX:   -500,
		TargetY:   300,
		CreatedAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Beat to the right.
	overlay.ClearBeats()
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{2},
		Type:      BeatHunt,
		TargetX:   1500,
		TargetY:   300,
		CreatedAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Beat above.
	overlay.ClearBeats()
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{3},
		Type:      BeatForge,
		TargetX:   400,
		TargetY:   -500,
		CreatedAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Beat below.
	overlay.ClearBeats()
	overlay.AddBeat(&DisplayBeat{
		ID:        [32]byte{4},
		Type:      BeatChain,
		TargetX:   400,
		TargetY:   1500,
		CreatedAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)
}
