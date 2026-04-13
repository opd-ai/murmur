// Package effects provides shader-based visual effects for the Pulse Map.
// Per PULSE_MAP.md, effects include glow, ripple, and spectra shaders.
package effects

import (
	"embed"
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed glow.kage ripple.kage spectra.kage
var shaderFS embed.FS

// Shaders holds compiled Kage shaders for Pulse Map effects.
type Shaders struct {
	Glow    *ebiten.Shader
	Ripple  *ebiten.Shader
	Spectra *ebiten.Shader
}

// LoadShaders compiles and returns all effect shaders.
func LoadShaders() (*Shaders, error) {
	shaders := &Shaders{}

	// Load glow shader
	glowSrc, err := shaderFS.ReadFile("glow.kage")
	if err != nil {
		return nil, fmt.Errorf("reading glow.kage: %w", err)
	}
	shaders.Glow, err = ebiten.NewShader(glowSrc)
	if err != nil {
		return nil, fmt.Errorf("compiling glow shader: %w", err)
	}

	// Load ripple shader
	rippleSrc, err := shaderFS.ReadFile("ripple.kage")
	if err != nil {
		return nil, fmt.Errorf("reading ripple.kage: %w", err)
	}
	shaders.Ripple, err = ebiten.NewShader(rippleSrc)
	if err != nil {
		return nil, fmt.Errorf("compiling ripple shader: %w", err)
	}

	// Load spectra shader
	spectraSrc, err := shaderFS.ReadFile("spectra.kage")
	if err != nil {
		return nil, fmt.Errorf("reading spectra.kage: %w", err)
	}
	shaders.Spectra, err = ebiten.NewShader(spectraSrc)
	if err != nil {
		return nil, fmt.Errorf("compiling spectra shader: %w", err)
	}

	return shaders, nil
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

// DrawGlow renders a glow effect at the given position.
func (s *Shaders) DrawGlow(dst *ebiten.Image, x, y, size float32, uniforms GlowUniforms) {
	if s.Glow == nil {
		return
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":          uniforms.Time,
		"GlowIntensity": uniforms.GlowIntensity,
		"GlowColor":     uniforms.GlowColor,
	}
	op.GeoM.Translate(float64(x-size/2), float64(y-size/2))

	dst.DrawRectShader(int(size), int(size), s.Glow, op)
}

// DrawRipple renders a ripple effect.
func (s *Shaders) DrawRipple(dst *ebiten.Image, uniforms RippleUniforms) {
	if s.Ripple == nil {
		return
	}

	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":         uniforms.Time,
		"RippleOrigin": uniforms.RippleOrigin,
		"RippleRadius": uniforms.RippleRadius,
		"RippleColor":  uniforms.RippleColor,
		"RippleWidth":  uniforms.RippleWidth,
	}

	dst.DrawRectShader(w, h, s.Ripple, op)
}

// DrawSpectra renders a specter effect on a node.
func (s *Shaders) DrawSpectra(dst *ebiten.Image, x, y, size float32, uniforms SpectraUniforms, src *ebiten.Image) {
	if s.Spectra == nil || src == nil {
		return
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":           uniforms.Time,
		"SpecterOpacity": uniforms.SpecterOpacity,
		"Resonance":      uniforms.Resonance,
	}
	op.Images[0] = src
	op.GeoM.Translate(float64(x-size/2), float64(y-size/2))

	dst.DrawRectShader(int(size), int(size), s.Spectra, op)
}
