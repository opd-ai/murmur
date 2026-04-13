// Package effects tests verify shader loading and effect rendering.
//
//go:build ebitentest
// +build ebitentest

package effects

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestLoadShaders(t *testing.T) {
	shaders, err := LoadShaders()
	if err != nil {
		t.Fatalf("LoadShaders failed: %v", err)
	}

	if shaders.Glow == nil {
		t.Error("Glow shader is nil")
	}
	if shaders.Ripple == nil {
		t.Error("Ripple shader is nil")
	}
	if shaders.Spectra == nil {
		t.Error("Spectra shader is nil")
	}
}

func TestDrawGlow(t *testing.T) {
	shaders, err := LoadShaders()
	if err != nil {
		t.Skipf("Could not load shaders: %v", err)
	}

	dst := ebiten.NewImage(100, 100)
	uniforms := GlowUniforms{
		Time:          0.5,
		GlowIntensity: 0.8,
		GlowColor:     [4]float32{1.0, 0.5, 0.0, 1.0},
	}

	// Should not panic
	shaders.DrawGlow(dst, 50, 50, 30, uniforms)
}

func TestDrawRipple(t *testing.T) {
	shaders, err := LoadShaders()
	if err != nil {
		t.Skipf("Could not load shaders: %v", err)
	}

	dst := ebiten.NewImage(100, 100)
	uniforms := RippleUniforms{
		Time:         1.0,
		RippleOrigin: [2]float32{50, 50},
		RippleRadius: 30,
		RippleColor:  [4]float32{0.0, 0.5, 1.0, 0.8},
		RippleWidth:  5,
	}

	// Should not panic
	shaders.DrawRipple(dst, uniforms)
}

func TestDrawSpectra(t *testing.T) {
	shaders, err := LoadShaders()
	if err != nil {
		t.Skipf("Could not load shaders: %v", err)
	}

	dst := ebiten.NewImage(100, 100)
	src := ebiten.NewImage(30, 30)
	uniforms := SpectraUniforms{
		Time:           0.5,
		SpecterOpacity: 0.7,
		Resonance:      75,
	}

	// Should not panic
	shaders.DrawSpectra(dst, 50, 50, 30, uniforms, src)
}

func TestNilShadersSafe(t *testing.T) {
	shaders := &Shaders{} // All nil

	dst := ebiten.NewImage(100, 100)

	// Should not panic with nil shaders
	shaders.DrawGlow(dst, 50, 50, 30, GlowUniforms{})
	shaders.DrawRipple(dst, RippleUniforms{})
	shaders.DrawSpectra(dst, 50, 50, 30, SpectraUniforms{}, nil)
}
