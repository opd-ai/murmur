// Package effects tests milestone visual effects.
//
//go:build noebiten
// +build noebiten

package effects

import (
	"testing"
)

func TestSurfaceMilestoneFromScore(t *testing.T) {
	tests := []struct {
		score    int
		expected SurfaceMilestone
	}{
		{0, MilestoneNone},
		{5, MilestoneNone},
		{10, MilestoneEmber},
		{15, MilestoneEmber},
		{25, MilestoneSpark},
		{49, MilestoneSpark},
		{50, MilestoneFlame},
		{99, MilestoneFlame},
		{100, MilestoneBlaze},
		{199, MilestoneBlaze},
		{200, MilestoneInferno},
		{499, MilestoneInferno},
		{500, MilestoneCorona},
		{1000, MilestoneCorona},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := SurfaceMilestoneFromScore(tt.score)
			if result != tt.expected {
				t.Errorf("SurfaceMilestoneFromScore(%d) = %d, want %d", tt.score, result, tt.expected)
			}
		})
	}
}

func TestSpecterMilestoneFromScore(t *testing.T) {
	tests := []struct {
		score    int
		expected SpecterMilestone
	}{
		{0, SpecterMilestoneNone},
		{5, SpecterMilestoneNone},
		{10, SpecterMilestoneWhisper},
		{24, SpecterMilestoneWhisper},
		{25, SpecterMilestoneShade},
		{49, SpecterMilestoneShade},
		{50, SpecterMilestoneWraith},
		{74, SpecterMilestoneWraith},
		{75, SpecterMilestoneShadeWraith},
		{99, SpecterMilestoneShadeWraith},
		{100, SpecterMilestonePhantom},
		{199, SpecterMilestonePhantom},
		{200, SpecterMilestoneRevenant},
		{499, SpecterMilestoneRevenant},
		{500, SpecterMilestoneAbyss},
		{1000, SpecterMilestoneAbyss},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := SpecterMilestoneFromScore(tt.score)
			if result != tt.expected {
				t.Errorf("SpecterMilestoneFromScore(%d) = %d, want %d", tt.score, result, tt.expected)
			}
		})
	}
}

func TestNewMilestoneEffects(t *testing.T) {
	m := NewMilestoneEffects()
	if m == nil {
		t.Fatal("NewMilestoneEffects returned nil")
	}
	if m.GetTime() != 0 {
		t.Errorf("initial time = %f, want 0", m.GetTime())
	}
}

func TestMilestoneEffectsUpdate(t *testing.T) {
	m := NewMilestoneEffects()

	// Update with dt
	m.Update(0.016) // ~60 FPS frame time
	if m.GetTime() == 0 {
		t.Error("time should advance after Update")
	}

	// Update again
	prevTime := m.GetTime()
	m.Update(0.016)
	if m.GetTime() <= prevTime {
		t.Error("time should continue advancing")
	}
}

func TestSurfaceMilestoneConstants(t *testing.T) {
	// Per RESONANCE_SYSTEM.md: Ember (10), Spark (25), Flame (50),
	// Blaze (100), Inferno (200), Corona (500)
	if MilestoneEmber != 10 {
		t.Errorf("MilestoneEmber = %d, want 10", MilestoneEmber)
	}
	if MilestoneSpark != 25 {
		t.Errorf("MilestoneSpark = %d, want 25", MilestoneSpark)
	}
	if MilestoneFlame != 50 {
		t.Errorf("MilestoneFlame = %d, want 50", MilestoneFlame)
	}
	if MilestoneBlaze != 100 {
		t.Errorf("MilestoneBlaze = %d, want 100", MilestoneBlaze)
	}
	if MilestoneInferno != 200 {
		t.Errorf("MilestoneInferno = %d, want 200", MilestoneInferno)
	}
	if MilestoneCorona != 500 {
		t.Errorf("MilestoneCorona = %d, want 500", MilestoneCorona)
	}
}

func TestSpecterMilestoneConstants(t *testing.T) {
	// Per RESONANCE_SYSTEM.md: Whisper (10), Shade (25), Wraith (50),
	// Shade-Wraith (75), Phantom (100), Revenant (200), Abyss (500)
	if SpecterMilestoneWhisper != 10 {
		t.Errorf("SpecterMilestoneWhisper = %d, want 10", SpecterMilestoneWhisper)
	}
	if SpecterMilestoneShade != 25 {
		t.Errorf("SpecterMilestoneShade = %d, want 25", SpecterMilestoneShade)
	}
	if SpecterMilestoneWraith != 50 {
		t.Errorf("SpecterMilestoneWraith = %d, want 50", SpecterMilestoneWraith)
	}
	if SpecterMilestoneShadeWraith != 75 {
		t.Errorf("SpecterMilestoneShadeWraith = %d, want 75", SpecterMilestoneShadeWraith)
	}
	if SpecterMilestonePhantom != 100 {
		t.Errorf("SpecterMilestonePhantom = %d, want 100", SpecterMilestonePhantom)
	}
	if SpecterMilestoneRevenant != 200 {
		t.Errorf("SpecterMilestoneRevenant = %d, want 200", SpecterMilestoneRevenant)
	}
	if SpecterMilestoneAbyss != 500 {
		t.Errorf("SpecterMilestoneAbyss = %d, want 500", SpecterMilestoneAbyss)
	}
}

func TestMilestoneThresholds(t *testing.T) {
	// Verify milestone progression
	tests := []struct {
		score   int
		surface SurfaceMilestone
		specter SpecterMilestone
	}{
		{9, MilestoneNone, SpecterMilestoneNone},
		{10, MilestoneEmber, SpecterMilestoneWhisper},
		{25, MilestoneSpark, SpecterMilestoneShade},
		{50, MilestoneFlame, SpecterMilestoneWraith},
		{75, MilestoneFlame, SpecterMilestoneShadeWraith}, // Surface has no 75 milestone
		{100, MilestoneBlaze, SpecterMilestonePhantom},
		{200, MilestoneInferno, SpecterMilestoneRevenant},
		{500, MilestoneCorona, SpecterMilestoneAbyss},
	}

	for _, tt := range tests {
		surface := SurfaceMilestoneFromScore(tt.score)
		specter := SpecterMilestoneFromScore(tt.score)

		if surface != tt.surface {
			t.Errorf("score %d: surface = %d, want %d", tt.score, surface, tt.surface)
		}
		if specter != tt.specter {
			t.Errorf("score %d: specter = %d, want %d", tt.score, specter, tt.specter)
		}
	}
}

func TestParticleCounters(t *testing.T) {
	m := NewMilestoneEffects()

	// Initial state
	if m.FlameParticleCount() != 0 {
		t.Errorf("FlameParticleCount = %d, want 0", m.FlameParticleCount())
	}
	if m.CoronaParticleCount() != 0 {
		t.Errorf("CoronaParticleCount = %d, want 0", m.CoronaParticleCount())
	}
	if m.SpecterParticleCount() != 0 {
		t.Errorf("SpecterParticleCount = %d, want 0", m.SpecterParticleCount())
	}
}
