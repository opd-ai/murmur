// Package overlays tests for layer blend logic.
// These tests use stub types and don't require Ebitengine rendering.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"
)

// TestLayerBlendDefault verifies default blend creation.
func TestLayerBlendDefault(t *testing.T) {
	b := NewDefaultBlend()
	if b == nil {
		t.Fatal("NewDefaultBlend() returned nil")
	}
	if b.SurfaceOpacity != 1.0 {
		t.Errorf("SurfaceOpacity = %v, want 1.0", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity != 0.5 {
		t.Errorf("AnonymousOpacity = %v, want 0.5", b.AnonymousOpacity)
	}
	if b.IsFortressMode {
		t.Error("IsFortressMode should be false")
	}
}

// TestLayerBlendFortress verifies fortress blend creation.
func TestLayerBlendFortress(t *testing.T) {
	b := NewFortressBlend()
	if b == nil {
		t.Fatal("NewFortressBlend() returned nil")
	}
	if b.SurfaceOpacity != 0.0 {
		t.Errorf("SurfaceOpacity = %v, want 0.0", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity != 1.0 {
		t.Errorf("AnonymousOpacity = %v, want 1.0", b.AnonymousOpacity)
	}
	if !b.IsFortressMode {
		t.Error("IsFortressMode should be true")
	}
}

// TestLayerBlendSetRatio verifies ratio adjustment.
func TestLayerBlendSetRatio(t *testing.T) {
	tests := []struct {
		name          string
		ratio         float32
		wantSurface   float32
		wantAnonymous float32
	}{
		{"zero", 0.0, 1.0, 0.0},
		{"half", 0.5, 0.5, 0.5},
		{"full", 1.0, 0.0, 1.0},
		{"negative clamps", -0.5, 1.0, 0.0},
		{"over one clamps", 1.5, 0.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewDefaultBlend()
			b.SetBlendRatio(tt.ratio)
			if b.SurfaceOpacity != tt.wantSurface {
				t.Errorf("SurfaceOpacity = %v, want %v", b.SurfaceOpacity, tt.wantSurface)
			}
			if b.AnonymousOpacity != tt.wantAnonymous {
				t.Errorf("AnonymousOpacity = %v, want %v", b.AnonymousOpacity, tt.wantAnonymous)
			}
		})
	}
}

// TestLayerBlendFortressLocked verifies fortress mode locks ratio.
func TestLayerBlendFortressLocked(t *testing.T) {
	b := NewFortressBlend()
	b.SetBlendRatio(0.5) // Should be ignored
	if b.SurfaceOpacity != 0.0 {
		t.Errorf("SurfaceOpacity changed in fortress mode: %v", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity != 1.0 {
		t.Errorf("AnonymousOpacity changed in fortress mode: %v", b.AnonymousOpacity)
	}
}

// TestParticleEmitterCreate verifies emitter creation.
func TestParticleEmitterCreate(t *testing.T) {
	e := NewParticleEmitter(100, 5.0)
	if e == nil {
		t.Fatal("NewParticleEmitter() returned nil")
	}
	if e.MaxParticles != 100 {
		t.Errorf("MaxParticles = %d, want 100", e.MaxParticles)
	}
	if e.EmitRate != 5.0 {
		t.Errorf("EmitRate = %v, want 5.0", e.EmitRate)
	}
	if len(e.Particles) != 0 {
		t.Errorf("Particles should be empty, got %d", len(e.Particles))
	}
}

// TestParticleEmitterUpdateBasic verifies basic update logic.
func TestParticleEmitterUpdateBasic(t *testing.T) {
	e := NewParticleEmitter(10, 10.0)
	// Update should emit particles
	e.Update(0.5, 100.0, 100.0, 20.0, 50.0)
	if len(e.Particles) == 0 {
		t.Error("Update should emit particles")
	}
}

// TestMiniGameVisualizationInit verifies visualization struct.
func TestMiniGameVisualizationInit(t *testing.T) {
	v := &MiniGameVisualization{
		Player1X:  100.0,
		Player1Y:  200.0,
		Player2X:  300.0,
		Player2Y:  400.0,
		Intensity: 0.5,
		Phase:     0.25,
	}
	if v.Player1X != 100.0 {
		t.Errorf("Player1X = %v, want 100.0", v.Player1X)
	}
	if v.Intensity != 0.5 {
		t.Errorf("Intensity = %v, want 0.5", v.Intensity)
	}
}
