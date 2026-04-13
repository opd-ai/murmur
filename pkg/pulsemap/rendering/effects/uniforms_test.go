// Package effects provides tests for shader uniform types.
// Tests that don't require Ebitengine rendering.
//
//go:build noebiten
// +build noebiten

package effects

import (
	"testing"
)

// TestGlowUniformsDefaults tests GlowUniforms struct initialization.
func TestGlowUniformsDefaults(t *testing.T) {
	u := GlowUniforms{
		Time:          0.5,
		GlowIntensity: 0.8,
		GlowColor:     [4]float32{1.0, 0.5, 0.0, 1.0},
	}

	if u.Time != 0.5 {
		t.Errorf("Expected Time 0.5, got %f", u.Time)
	}
	if u.GlowIntensity != 0.8 {
		t.Errorf("Expected GlowIntensity 0.8, got %f", u.GlowIntensity)
	}
	if u.GlowColor[0] != 1.0 || u.GlowColor[3] != 1.0 {
		t.Error("GlowColor not set correctly")
	}
}

// TestRippleUniformsDefaults tests RippleUniforms struct initialization.
func TestRippleUniformsDefaults(t *testing.T) {
	u := RippleUniforms{
		Time:         1.0,
		RippleOrigin: [2]float32{50, 50},
		RippleRadius: 30,
		RippleColor:  [4]float32{0.0, 0.5, 1.0, 0.8},
		RippleWidth:  5,
	}

	if u.Time != 1.0 {
		t.Errorf("Expected Time 1.0, got %f", u.Time)
	}
	if u.RippleOrigin[0] != 50 || u.RippleOrigin[1] != 50 {
		t.Error("RippleOrigin not set correctly")
	}
	if u.RippleRadius != 30 {
		t.Errorf("Expected RippleRadius 30, got %f", u.RippleRadius)
	}
	if u.RippleWidth != 5 {
		t.Errorf("Expected RippleWidth 5, got %f", u.RippleWidth)
	}
}

// TestSpectraUniformsDefaults tests SpectraUniforms struct initialization.
func TestSpectraUniformsDefaults(t *testing.T) {
	u := SpectraUniforms{
		Time:           0.5,
		SpecterOpacity: 0.7,
		Resonance:      75,
	}

	if u.Time != 0.5 {
		t.Errorf("Expected Time 0.5, got %f", u.Time)
	}
	if u.SpecterOpacity != 0.7 {
		t.Errorf("Expected SpecterOpacity 0.7, got %f", u.SpecterOpacity)
	}
	if u.Resonance != 75 {
		t.Errorf("Expected Resonance 75, got %f", u.Resonance)
	}
}

// TestGlowUniformsZeroValue tests zero-value GlowUniforms.
func TestGlowUniformsZeroValue(t *testing.T) {
	var u GlowUniforms

	if u.Time != 0 {
		t.Error("Zero GlowUniforms.Time should be 0")
	}
	if u.GlowIntensity != 0 {
		t.Error("Zero GlowUniforms.GlowIntensity should be 0")
	}
	for i, v := range u.GlowColor {
		if v != 0 {
			t.Errorf("Zero GlowColor[%d] should be 0", i)
		}
	}
}

// TestRippleUniformsZeroValue tests zero-value RippleUniforms.
func TestRippleUniformsZeroValue(t *testing.T) {
	var u RippleUniforms

	if u.Time != 0 {
		t.Error("Zero RippleUniforms.Time should be 0")
	}
	if u.RippleRadius != 0 {
		t.Error("Zero RippleUniforms.RippleRadius should be 0")
	}
	if u.RippleWidth != 0 {
		t.Error("Zero RippleUniforms.RippleWidth should be 0")
	}
}

// TestSpectraUniformsZeroValue tests zero-value SpectraUniforms.
func TestSpectraUniformsZeroValue(t *testing.T) {
	var u SpectraUniforms

	if u.Time != 0 {
		t.Error("Zero SpectraUniforms.Time should be 0")
	}
	if u.SpecterOpacity != 0 {
		t.Error("Zero SpectraUniforms.SpecterOpacity should be 0")
	}
	if u.Resonance != 0 {
		t.Error("Zero SpectraUniforms.Resonance should be 0")
	}
}

// TestShadersStructZeroValue tests zero-value Shaders struct.
func TestShadersStructZeroValue(t *testing.T) {
	var s Shaders

	if s.Glow != nil {
		t.Error("Zero Shaders.Glow should be nil")
	}
	if s.Ripple != nil {
		t.Error("Zero Shaders.Ripple should be nil")
	}
	if s.Spectra != nil {
		t.Error("Zero Shaders.Spectra should be nil")
	}
}
