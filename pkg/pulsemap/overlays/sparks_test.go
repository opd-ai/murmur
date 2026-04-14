// Package overlays - Surface Spark overlay tests.
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewSparkOverlay(t *testing.T) {
	overlay := NewSparkOverlay()

	if overlay == nil {
		t.Fatal("NewSparkOverlay returned nil")
	}
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible by default")
	}
	if overlay.SparkCount() != 0 {
		t.Error("expected no sparks initially")
	}
	if overlay.CrownCount() != 0 {
		t.Error("expected no crowns initially")
	}
}

func TestSparkOverlay_Visibility(t *testing.T) {
	overlay := NewSparkOverlay()

	overlay.SetVisible(false)
	if overlay.IsVisible() {
		t.Error("expected overlay to be hidden")
	}

	overlay.SetVisible(true)
	if !overlay.IsVisible() {
		t.Error("expected overlay to be visible")
	}
}

func TestSparkOverlay_SetSpark(t *testing.T) {
	overlay := NewSparkOverlay()

	spark := &SparkInfo{
		ID:        [32]byte{1, 2, 3},
		Type:      SparkWaveRelay,
		State:     SparkActive,
		X:         100,
		Y:         200,
		Prompt:    "Test prompt",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Responses: 0,
	}

	overlay.SetSpark(spark)

	if overlay.SparkCount() != 1 {
		t.Errorf("expected 1 spark, got %d", overlay.SparkCount())
	}

	retrieved := overlay.GetSpark(spark.ID)
	if retrieved == nil {
		t.Fatal("GetSpark returned nil")
	}
	if retrieved.Type != SparkWaveRelay {
		t.Errorf("expected WaveRelay type, got %v", retrieved.Type)
	}
}

func TestSparkOverlay_SetSparkNil(t *testing.T) {
	overlay := NewSparkOverlay()
	overlay.SetSpark(nil)

	if overlay.SparkCount() != 0 {
		t.Error("setting nil spark should not add anything")
	}
}

func TestSparkOverlay_RemoveSpark(t *testing.T) {
	overlay := NewSparkOverlay()

	spark := &SparkInfo{
		ID:    [32]byte{1},
		Type:  SparkEchoRace,
		State: SparkActive,
	}

	overlay.SetSpark(spark)
	if overlay.SparkCount() != 1 {
		t.Error("spark not added")
	}

	overlay.RemoveSpark(spark.ID)
	if overlay.SparkCount() != 0 {
		t.Error("spark not removed")
	}
}

func TestSparkOverlay_GetAllSparks(t *testing.T) {
	overlay := NewSparkOverlay()

	for i := 0; i < 5; i++ {
		spark := &SparkInfo{
			ID:    [32]byte{byte(i)},
			Type:  SparkWaveRelay,
			State: SparkActive,
		}
		overlay.SetSpark(spark)
	}

	sparks := overlay.GetAllSparks()
	if len(sparks) != 5 {
		t.Errorf("expected 5 sparks, got %d", len(sparks))
	}
}

func TestSparkOverlay_GetActiveSparks(t *testing.T) {
	overlay := NewSparkOverlay()

	// Add mix of active and inactive sparks.
	overlay.SetSpark(&SparkInfo{ID: [32]byte{1}, State: SparkActive})
	overlay.SetSpark(&SparkInfo{ID: [32]byte{2}, State: SparkActive})
	overlay.SetSpark(&SparkInfo{ID: [32]byte{3}, State: SparkCompleted})
	overlay.SetSpark(&SparkInfo{ID: [32]byte{4}, State: SparkExpired})
	overlay.SetSpark(&SparkInfo{ID: [32]byte{5}, State: SparkActive})

	active := overlay.GetActiveSparks()
	if len(active) != 3 {
		t.Errorf("expected 3 active sparks, got %d", len(active))
	}

	if overlay.ActiveSparkCount() != 3 {
		t.Errorf("ActiveSparkCount expected 3, got %d", overlay.ActiveSparkCount())
	}
}

func TestSparkOverlay_CrownHolder(t *testing.T) {
	overlay := NewSparkOverlay()

	holder := &CrownHolder{
		UserKey:   [32]byte{10, 20, 30},
		X:         50,
		Y:         60,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	overlay.SetCrownHolder(holder)

	if overlay.CrownCount() != 1 {
		t.Errorf("expected 1 crown holder, got %d", overlay.CrownCount())
	}

	if !overlay.HasCrown(holder.UserKey) {
		t.Error("expected user to have crown")
	}

	retrieved := overlay.GetCrownHolder(holder.UserKey)
	if retrieved == nil {
		t.Fatal("GetCrownHolder returned nil")
	}
	if retrieved.X != 50 || retrieved.Y != 60 {
		t.Error("crown holder position mismatch")
	}
}

func TestSparkOverlay_CrownHolderExpired(t *testing.T) {
	overlay := NewSparkOverlay()

	holder := &CrownHolder{
		UserKey:   [32]byte{1},
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired.
	}

	overlay.SetCrownHolder(holder)

	if overlay.HasCrown(holder.UserKey) {
		t.Error("expected expired crown to not count")
	}

	if overlay.CrownCount() != 0 {
		t.Error("expected 0 active crowns")
	}
}

func TestSparkOverlay_RemoveCrownHolder(t *testing.T) {
	overlay := NewSparkOverlay()

	holder := &CrownHolder{
		UserKey:   [32]byte{1},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	overlay.SetCrownHolder(holder)
	overlay.RemoveCrownHolder(holder.UserKey)

	if overlay.GetCrownHolder(holder.UserKey) != nil {
		t.Error("crown holder should be removed")
	}
}

func TestSparkOverlay_Update(t *testing.T) {
	overlay := NewSparkOverlay()

	// Add expired crown.
	holder := &CrownHolder{
		UserKey:   [32]byte{1},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}
	overlay.SetCrownHolder(holder)

	// Update should purge expired crowns.
	overlay.Update(0.1)

	// The crown should be purged.
	if overlay.GetCrownHolder(holder.UserKey) != nil {
		t.Error("expired crown should be purged on update")
	}
}

func TestSparkOverlay_Draw(t *testing.T) {
	overlay := NewSparkOverlay()
	screen := ebiten.NewImage(800, 600)

	// Draw empty overlay.
	overlay.Draw(screen, 0, 0, 1.0)

	// Add spark and crown, then draw.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{1},
		Type:      SparkWaveRelay,
		State:     SparkActive,
		X:         100,
		Y:         100,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Responses: 3,
	})

	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{2},
		Type:      SparkEchoRace,
		State:     SparkActive,
		X:         200,
		Y:         200,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	overlay.SetCrownHolder(&CrownHolder{
		UserKey:   [32]byte{10},
		X:         150,
		Y:         150,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	overlay.Draw(screen, 0, 0, 1.0)

	// Draw with camera offset.
	overlay.Draw(screen, 50, 50, 1.5)

	// Draw when hidden.
	overlay.SetVisible(false)
	overlay.Draw(screen, 0, 0, 1.0)
}

func TestSparkOverlay_DrawCompletedSparks(t *testing.T) {
	overlay := NewSparkOverlay()
	screen := ebiten.NewImage(800, 600)

	// Add completed sparks.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{1},
		Type:      SparkWaveRelay,
		State:     SparkCompleted,
		X:         100,
		Y:         100,
		ExpiresAt: time.Now(),
	})

	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{2},
		Type:      SparkEchoRace,
		State:     SparkExpired,
		X:         200,
		Y:         200,
		ExpiresAt: time.Now(),
	})

	overlay.Draw(screen, 0, 0, 1.0)
}

func TestSparkOverlay_ClearExpired(t *testing.T) {
	overlay := NewSparkOverlay()

	// Add expired spark.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{1},
		State:     SparkExpired,
		ExpiresAt: time.Now().Add(-2 * time.Hour),
	})

	// Add recent completed spark.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{2},
		State:     SparkCompleted,
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	})

	// Add active spark.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{3},
		State:     SparkActive,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	// Clear sparks older than 1 hour.
	removed := overlay.ClearExpired(1 * time.Hour)

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	if overlay.SparkCount() != 2 {
		t.Errorf("expected 2 sparks remaining, got %d", overlay.SparkCount())
	}
}

func TestSparkOverlay_UpdatePositions(t *testing.T) {
	overlay := NewSparkOverlay()

	spark := &SparkInfo{
		ID: [32]byte{1},
		X:  100,
		Y:  100,
	}
	overlay.SetSpark(spark)

	overlay.UpdateSparkPosition([32]byte{1}, 200, 300)

	updated := overlay.GetSpark([32]byte{1})
	if updated.X != 200 || updated.Y != 300 {
		t.Error("spark position not updated")
	}

	holder := &CrownHolder{
		UserKey:   [32]byte{10},
		X:         50,
		Y:         50,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	overlay.SetCrownHolder(holder)

	overlay.UpdateCrownPosition([32]byte{10}, 150, 250)

	updatedHolder := overlay.GetCrownHolder([32]byte{10})
	if updatedHolder.X != 150 || updatedHolder.Y != 250 {
		t.Error("crown position not updated")
	}
}

func TestSparkOverlay_SetSparkState(t *testing.T) {
	overlay := NewSparkOverlay()

	spark := &SparkInfo{
		ID:    [32]byte{1},
		State: SparkActive,
	}
	overlay.SetSpark(spark)

	overlay.SetSparkState([32]byte{1}, SparkCompleted)

	updated := overlay.GetSpark([32]byte{1})
	if updated.State != SparkCompleted {
		t.Error("spark state not updated")
	}
}

func TestSparkOverlay_SetSparkResponses(t *testing.T) {
	overlay := NewSparkOverlay()

	spark := &SparkInfo{
		ID:        [32]byte{1},
		Responses: 0,
	}
	overlay.SetSpark(spark)

	overlay.SetSparkResponses([32]byte{1}, 5)

	updated := overlay.GetSpark([32]byte{1})
	if updated.Responses != 5 {
		t.Error("spark responses not updated")
	}
}

func TestSparkTypeString(t *testing.T) {
	tests := []struct {
		sparkType SparkType
		expected  string
	}{
		{SparkWaveRelay, "Wave Relay"},
		{SparkEchoRace, "Echo Race"},
		{SparkType(99), "Unknown"},
	}

	for _, tc := range tests {
		result := SparkTypeString(tc.sparkType)
		if result != tc.expected {
			t.Errorf("SparkTypeString(%v) = %q, expected %q", tc.sparkType, result, tc.expected)
		}
	}
}

func TestSparkStateString(t *testing.T) {
	tests := []struct {
		state    SparkState
		expected string
	}{
		{SparkActive, "Active"},
		{SparkCompleted, "Completed"},
		{SparkExpired, "Expired"},
		{SparkCancelled, "Cancelled"},
		{SparkState(99), "Unknown"},
	}

	for _, tc := range tests {
		result := SparkStateString(tc.state)
		if result != tc.expected {
			t.Errorf("SparkStateString(%v) = %q, expected %q", tc.state, result, tc.expected)
		}
	}
}

func TestSparkOverlay_DrawWithZoom(t *testing.T) {
	overlay := NewSparkOverlay()
	screen := ebiten.NewImage(800, 600)

	// Add sparks.
	overlay.SetSpark(&SparkInfo{
		ID:        [32]byte{1},
		Type:      SparkWaveRelay,
		State:     SparkActive,
		X:         400,
		Y:         300,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Responses: 10, // More than 8 to test indicator.
	})

	// Test various zoom levels.
	for _, zoom := range []float64{0.1, 0.5, 1.0, 2.0, 5.0} {
		overlay.Draw(screen, 0, 0, zoom)
	}
}

func TestMin(t *testing.T) {
	if min(5, 10) != 5 {
		t.Error("min(5, 10) should be 5")
	}
	if min(10, 5) != 5 {
		t.Error("min(10, 5) should be 5")
	}
	if min(5, 5) != 5 {
		t.Error("min(5, 5) should be 5")
	}
}
