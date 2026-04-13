// Package effects provides stub types for testing without Ebitengine.
// This file is used when building with the noebiten tag.
//
//go:build noebiten
// +build noebiten

package effects

// Shaders holds compiled Kage shaders for Pulse Map effects.
// Stub version for noebiten builds.
type Shaders struct {
	Glow    interface{}
	Ripple  interface{}
	Spectra interface{}
}

// GlowUniforms contains uniforms for the glow shader.
type GlowUniforms struct {
	Time          float32
	GlowIntensity float32
	GlowColor     [4]float32 // RGBA
}

// RippleUniforms contains uniforms for the ripple shader.
type RippleUniforms struct {
	Time         float32
	RippleOrigin [2]float32
	RippleRadius float32
	RippleColor  [4]float32 // RGBA
	RippleWidth  float32
}

// SpectraUniforms contains uniforms for the spectra shader.
type SpectraUniforms struct {
	Time           float32
	SpecterOpacity float32
	Resonance      float32
}
