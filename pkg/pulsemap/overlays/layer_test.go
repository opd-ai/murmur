// Package overlays tests verify the Anonymous Layer visualization.
// Pure logic tests that don't require Ebitengine.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"
)

func TestNewDefaultBlend(t *testing.T) {
	b := NewDefaultBlend()
	if b.SurfaceOpacity != 1.0 {
		t.Errorf("expected SurfaceOpacity 1.0, got %f", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity != 0.5 {
		t.Errorf("expected AnonymousOpacity 0.5, got %f", b.AnonymousOpacity)
	}
	if b.IsFortressMode {
		t.Error("expected IsFortressMode false")
	}
}

func TestNewFortressBlend(t *testing.T) {
	b := NewFortressBlend()
	if b.SurfaceOpacity != 0.0 {
		t.Errorf("expected SurfaceOpacity 0.0, got %f", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity != 1.0 {
		t.Errorf("expected AnonymousOpacity 1.0, got %f", b.AnonymousOpacity)
	}
	if !b.IsFortressMode {
		t.Error("expected IsFortressMode true")
	}
}

func TestSetBlendRatio(t *testing.T) {
	b := NewDefaultBlend()

	// Set to anonymous-heavy
	b.SetBlendRatio(0.8)
	if b.SurfaceOpacity < 0.19 || b.SurfaceOpacity > 0.21 {
		t.Errorf("expected SurfaceOpacity ~0.2, got %f", b.SurfaceOpacity)
	}
	if b.AnonymousOpacity < 0.79 || b.AnonymousOpacity > 0.81 {
		t.Errorf("expected AnonymousOpacity ~0.8, got %f", b.AnonymousOpacity)
	}

	// Test clamping
	b.SetBlendRatio(-0.5)
	if b.SurfaceOpacity != 1.0 || b.AnonymousOpacity != 0.0 {
		t.Error("expected clamping to 0 ratio")
	}

	b.SetBlendRatio(1.5)
	if b.SurfaceOpacity != 0.0 || b.AnonymousOpacity != 1.0 {
		t.Error("expected clamping to 1 ratio")
	}
}

func TestFortressModeLocked(t *testing.T) {
	b := NewFortressBlend()
	initialSurface := b.SurfaceOpacity
	initialAnonymous := b.AnonymousOpacity

	// Try to change blend in Fortress mode - should be ignored
	b.SetBlendRatio(0.5)

	if b.SurfaceOpacity != initialSurface || b.AnonymousOpacity != initialAnonymous {
		t.Error("Fortress mode should lock blend ratio")
	}
}

func TestNewParticleEmitter(t *testing.T) {
	e := NewParticleEmitter(100, 10.0)
	if e.MaxParticles != 100 {
		t.Errorf("expected MaxParticles 100, got %d", e.MaxParticles)
	}
	if e.EmitRate != 10.0 {
		t.Errorf("expected EmitRate 10.0, got %f", e.EmitRate)
	}
	if len(e.Particles) != 0 {
		t.Errorf("expected 0 particles, got %d", len(e.Particles))
	}
}

func TestParticleEmitterUpdate(t *testing.T) {
	e := NewParticleEmitter(100, 10.0)

	// Update with large dt to trigger emission
	e.Update(1.0, 50, 50, 10, 50)

	if len(e.Particles) == 0 {
		t.Error("expected particles after update")
	}

	// Particles should have life
	for _, p := range e.Particles {
		if p.Life <= 0 {
			t.Error("particle should have positive life")
		}
	}
}

// NOTE: Tests that require Ebitengine rendering (ParticleEmitter.Render,
// ShroudIndicator, MiniGameVisualization.Render) are in overlays_render_test.go
// behind the "ebitentest" build tag per TECHNICAL_IMPLEMENTATION.md.

func TestTrigFunctions(t *testing.T) {
	// Simple sanity checks for approximate trig functions
	// These are approximations so we accept some error

	// cos(0) ≈ 1
	if cos(0) != 1.0 {
		t.Errorf("cos(0) should be 1.0, got %f", cos(0))
	}

	// sin(0) = 0
	if sin(0) != 0.0 {
		t.Errorf("sin(0) should be 0.0, got %f", sin(0))
	}
}
