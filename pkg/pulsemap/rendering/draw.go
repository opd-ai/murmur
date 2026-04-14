// Package rendering provides Ebitengine-based node/edge drawing for the Pulse Map.
// Per TECHNICAL_IMPLEMENTATION.md §1.2, rendering uses Ebitengine v2.7+
// with Kage shaders for glow and ripple effects.
//
//go:build !noebiten
// +build !noebiten

package rendering

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TargetFPS is the target rendering frame rate.
const TargetFPS = 60

// NOTE: This package is the only one under pkg/ that imports Ebitengine.
// Per TECHNICAL_IMPLEMENTATION.md §2, no other package should import Ebitengine.

// ZoomLevel represents the current detail level for rendering.
type ZoomLevel int

const (
	ZoomMacro ZoomLevel = iota // Wide view, minimal detail
	ZoomMeso                   // Medium view, labels visible
	ZoomMicro                  // Close view, full detail
)

// Specter visual constants per DESIGN_DOCUMENT.md §33.
const (
	// SpecterBaseAlpha is the base translucency for Specter nodes (0.0-1.0).
	// Per PULSE_MAP.md: Specters are rendered with 60-80% opacity.
	SpecterBaseAlpha = 0.7

	// SpecterParticleCount is the number of particles around a Specter node.
	SpecterParticleCount = 5

	// SpecterParticleRadius is the radius of individual particles.
	SpecterParticleRadius = 1.5

	// SpecterParticleOrbitRadius is how far particles orbit from node center.
	SpecterParticleOrbitRadius = 1.8

	// SpecterShimmerSpeed controls the shimmer animation speed.
	SpecterShimmerSpeed = 2.0
)

// NodeStyle contains visual properties for a node.
type NodeStyle struct {
	CoreColor   color.RGBA
	RingColor   color.RGBA
	HasRing     bool
	HasHalo     bool
	HaloAlpha   float32 // 0-1, decay based on activity
	IsSpecter   bool    // Anonymous layer node
	Selected    bool
	Connections int
	Activity    float64 // Activity metric for sizing
	Resonance   float64 // For Specter nodes
}

// RenderNode draws a node at the given screen position.
// Per PULSE_MAP.md, radius = r_base + r_scale * ln(1 + metric).
// Specter nodes are rendered with translucency and particle emissions.
func RenderNode(dst *ebiten.Image, x, y float32, style NodeStyle, zoom ZoomLevel) {
	radius := computeNodeRadius(style)

	if style.IsSpecter {
		// Render Specter node with special effects.
		renderSpecterNode(dst, x, y, radius, style)
	} else {
		// Render Surface node normally.
		renderNodeHalo(dst, x, y, radius, style)
		renderNodeCore(dst, x, y, radius, style)
		renderNodeRing(dst, x, y, radius, style)
	}

	renderNodeSelection(dst, x, y, radius, style)
}

// RenderNodeWithTime draws a node with time-based animations.
// Use this version when you need animated particle effects.
func RenderNodeWithTime(dst *ebiten.Image, x, y float32, style NodeStyle, zoom ZoomLevel, time float64) {
	radius := computeNodeRadius(style)

	if style.IsSpecter {
		renderSpecterNodeAnimated(dst, x, y, radius, style, time)
	} else {
		renderNodeHalo(dst, x, y, radius, style)
		renderNodeCore(dst, x, y, radius, style)
		renderNodeRing(dst, x, y, radius, style)
	}

	renderNodeSelection(dst, x, y, radius, style)
}

// computeNodeRadius calculates node size based on connections/resonance.
func computeNodeRadius(style NodeStyle) float32 {
	const (
		rBase  = 4.0
		rScale = 3.0
	)
	var metric float64
	if style.IsSpecter {
		metric = style.Resonance
	} else {
		metric = float64(style.Connections) + style.Activity
	}
	return float32(rBase + rScale*math.Log(1+metric))
}

// renderSpecterNode renders a Specter node with translucency and ghostly effects.
// Per DESIGN_DOCUMENT.md §33: Specters appear as ghostly nodes with greater
// transparency and particle effects, using a cool-tone color palette.
func renderSpecterNode(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	// Apply translucency to core color.
	alpha := uint8(float64(style.CoreColor.A) * SpecterBaseAlpha)
	translucentCore := color.RGBA{
		R: style.CoreColor.R,
		G: style.CoreColor.G,
		B: style.CoreColor.B,
		A: alpha,
	}

	// Draw ethereal halo (larger, more diffuse than Surface).
	if style.HasHalo || style.Resonance > 0 {
		haloAlpha := uint8(40 + int(style.Resonance/10))
		if haloAlpha > 100 {
			haloAlpha = 100
		}
		haloColor := color.RGBA{
			R: style.CoreColor.R,
			G: style.CoreColor.G,
			B: style.CoreColor.B,
			A: haloAlpha,
		}
		vector.DrawFilledCircle(dst, x, y, radius*2.5, haloColor, true)
	}

	// Draw the translucent core.
	vector.DrawFilledCircle(dst, x, y, radius, translucentCore, true)

	// Draw ghostly edge glow (softer outer ring).
	edgeColor := color.RGBA{
		R: min(style.CoreColor.R+30, 255),
		G: min(style.CoreColor.G+30, 255),
		B: min(style.CoreColor.B+50, 255), // Slightly more blue.
		A: uint8(float64(alpha) * 0.6),
	}
	vector.StrokeCircle(dst, x, y, radius+1, 1.5, edgeColor, true)

	// Draw static particle positions (no animation).
	renderSpecterParticlesStatic(dst, x, y, radius, style)

	// Draw ring if present.
	if style.HasRing {
		ringAlpha := uint8(float64(style.RingColor.A) * SpecterBaseAlpha)
		ringColor := color.RGBA{
			R: style.RingColor.R,
			G: style.RingColor.G,
			B: style.RingColor.B,
			A: ringAlpha,
		}
		ringThickness := float32(1.5)
		if style.Selected {
			ringThickness = 3.0
		}
		vector.StrokeCircle(dst, x, y, radius+ringThickness, ringThickness, ringColor, true)
	}
}

// renderSpecterNodeAnimated renders a Specter node with time-based animations.
func renderSpecterNodeAnimated(dst *ebiten.Image, x, y, radius float32, style NodeStyle, time float64) {
	// Compute shimmer effect.
	shimmer := 0.9 + 0.1*math.Sin(time*SpecterShimmerSpeed)
	alpha := uint8(float64(style.CoreColor.A) * SpecterBaseAlpha * shimmer)

	translucentCore := color.RGBA{
		R: style.CoreColor.R,
		G: style.CoreColor.G,
		B: style.CoreColor.B,
		A: alpha,
	}

	// Draw ethereal halo with pulsing effect.
	if style.HasHalo || style.Resonance > 0 {
		haloShimmer := 0.8 + 0.2*math.Sin(time*1.5)
		haloAlpha := uint8((40 + int(style.Resonance/10)) * int(haloShimmer*100) / 100)
		if haloAlpha > 100 {
			haloAlpha = 100
		}
		haloColor := color.RGBA{
			R: style.CoreColor.R,
			G: style.CoreColor.G,
			B: style.CoreColor.B,
			A: haloAlpha,
		}
		vector.DrawFilledCircle(dst, x, y, radius*2.5, haloColor, true)
	}

	// Draw the translucent core.
	vector.DrawFilledCircle(dst, x, y, radius, translucentCore, true)

	// Draw ghostly edge glow.
	edgeColor := color.RGBA{
		R: min(style.CoreColor.R+30, 255),
		G: min(style.CoreColor.G+30, 255),
		B: min(style.CoreColor.B+50, 255),
		A: uint8(float64(alpha) * 0.6),
	}
	vector.StrokeCircle(dst, x, y, radius+1, 1.5, edgeColor, true)

	// Draw animated particles.
	renderSpecterParticlesAnimated(dst, x, y, radius, style, time)

	// Draw ring if present.
	if style.HasRing {
		ringAlpha := uint8(float64(style.RingColor.A) * SpecterBaseAlpha * shimmer)
		ringColor := color.RGBA{
			R: style.RingColor.R,
			G: style.RingColor.G,
			B: style.RingColor.B,
			A: ringAlpha,
		}
		ringThickness := float32(1.5)
		if style.Selected {
			ringThickness = 3.0
		}
		vector.StrokeCircle(dst, x, y, radius+ringThickness, ringThickness, ringColor, true)
	}
}

// renderSpecterParticlesStatic draws static particle emissions around a Specter.
func renderSpecterParticlesStatic(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	// Particle count increases with Resonance.
	particleCount := SpecterParticleCount
	if style.Resonance > 50 {
		particleCount += int(style.Resonance / 50)
	}
	if particleCount > 12 {
		particleCount = 12
	}

	particleColor := color.RGBA{
		R: min(style.CoreColor.R+50, 255),
		G: min(style.CoreColor.G+50, 255),
		B: min(style.CoreColor.B+80, 255), // Extra blue for cool tones.
		A: 150,
	}

	orbitRadius := radius * SpecterParticleOrbitRadius
	for i := 0; i < particleCount; i++ {
		angle := float64(i) * (2 * math.Pi / float64(particleCount))
		px := x + float32(math.Cos(angle))*orbitRadius
		py := y + float32(math.Sin(angle))*orbitRadius
		vector.DrawFilledCircle(dst, px, py, SpecterParticleRadius, particleColor, true)
	}
}

// renderSpecterParticlesAnimated draws animated particles orbiting a Specter.
func renderSpecterParticlesAnimated(dst *ebiten.Image, x, y, radius float32, style NodeStyle, time float64) {
	particleCount := SpecterParticleCount
	if style.Resonance > 50 {
		particleCount += int(style.Resonance / 50)
	}
	if particleCount > 12 {
		particleCount = 12
	}

	orbitRadius := radius * SpecterParticleOrbitRadius

	// Seeded random for consistent particle offsets per node.
	rng := rand.New(rand.NewSource(int64(style.CoreColor.R) + int64(style.CoreColor.G)*256 + int64(style.CoreColor.B)*65536))

	for i := 0; i < particleCount; i++ {
		// Each particle has a unique phase offset.
		phaseOffset := rng.Float64() * 2 * math.Pi
		speedVariation := 0.5 + rng.Float64()*0.5

		angle := phaseOffset + time*speedVariation
		px := x + float32(math.Cos(angle))*orbitRadius
		py := y + float32(math.Sin(angle))*orbitRadius

		// Particle alpha varies with time for twinkling effect.
		twinkle := 0.5 + 0.5*math.Sin(time*3.0+phaseOffset*2)
		particleColor := color.RGBA{
			R: min(style.CoreColor.R+50, 255),
			G: min(style.CoreColor.G+50, 255),
			B: min(style.CoreColor.B+80, 255),
			A: uint8(150 * twinkle),
		}

		vector.DrawFilledCircle(dst, px, py, SpecterParticleRadius, particleColor, true)
	}
}

// renderNodeHalo draws the activity halo if present.
func renderNodeHalo(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	if !style.HasHalo || style.HaloAlpha <= 0 {
		return
	}
	haloColor := color.RGBA{
		R: style.CoreColor.R,
		G: style.CoreColor.G,
		B: style.CoreColor.B,
		A: uint8(float32(80) * style.HaloAlpha),
	}
	vector.DrawFilledCircle(dst, x, y, radius*2.0, haloColor, true)
}

// renderNodeCore draws the filled node circle.
func renderNodeCore(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	vector.DrawFilledCircle(dst, x, y, radius, style.CoreColor, true)
}

// renderNodeRing draws the mode ring if enabled.
func renderNodeRing(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	if !style.HasRing {
		return
	}
	ringThickness := float32(1.5)
	if style.Selected {
		ringThickness = 3.0
	}
	vector.StrokeCircle(dst, x, y, radius+ringThickness, ringThickness, style.RingColor, true)
}

// renderNodeSelection draws the selection highlight.
func renderNodeSelection(dst *ebiten.Image, x, y, radius float32, style NodeStyle) {
	if !style.Selected {
		return
	}
	selectColor := color.RGBA{255, 255, 255, 128}
	if style.IsSpecter {
		// Specter selection is more ghostly/translucent.
		selectColor = color.RGBA{200, 220, 255, 100}
	}
	vector.StrokeCircle(dst, x, y, radius+6, 2.0, selectColor, true)
}

// EdgeStyle contains visual properties for an edge.
type EdgeStyle struct {
	Color      color.RGBA
	Age        float64 // Connection age in days
	Active     bool    // Recent wave propagation
	IsMiniGame bool    // Active mini-game connection
	IsSpecter  bool    // Anonymous layer edge
}

// RenderEdge draws a connection edge between two nodes.
// Per PULSE_MAP.md, edges are quadratic Bézier curves with age-based styling.
func RenderEdge(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel) {
	// Calculate edge opacity based on age.
	var alpha uint8 = 50 // Base alpha (20-40% as per spec)
	if style.Age > 90 {
		alpha = 80 // Old connections more visible
	} else if style.Age < 7 {
		alpha = 40 // New connections dashed (simplified to lower alpha)
	}

	// Specter edges are more translucent.
	if style.IsSpecter {
		alpha = uint8(float32(alpha) * 0.7)
	}

	edgeColor := color.RGBA{
		R: style.Color.R,
		G: style.Color.G,
		B: style.Color.B,
		A: alpha,
	}

	// For simplicity, draw straight line (Bézier curves require more complex path)
	vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, edgeColor, true)

	// Activity pulse animation (simplified)
	if style.Active {
		pulseColor := color.RGBA{255, 255, 255, 180}
		if style.IsSpecter {
			pulseColor = color.RGBA{200, 220, 255, 140} // Cooler, more ghostly
		}
		// Draw a bright dot at midpoint to indicate activity.
		mx := (x1 + x2) / 2
		my := (y1 + y2) / 2
		vector.DrawFilledCircle(dst, mx, my, 3, pulseColor, true)
	}
}

// RenderEdgeWithTime draws an edge with time-based animations.
func RenderEdgeWithTime(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel, time float64) {
	var alpha uint8 = 50
	if style.Age > 90 {
		alpha = 80
	} else if style.Age < 7 {
		alpha = 40
	}

	if style.IsSpecter {
		// Animated shimmer for Specter edges.
		shimmer := 0.7 + 0.3*math.Sin(time*2.0)
		alpha = uint8(float64(alpha) * shimmer)
	}

	edgeColor := color.RGBA{
		R: style.Color.R,
		G: style.Color.G,
		B: style.Color.B,
		A: alpha,
	}

	vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, edgeColor, true)

	if style.Active {
		// Animated pulse moving along the edge.
		pulsePos := math.Mod(time*0.5, 1.0)
		px := x1 + float32(pulsePos)*(x2-x1)
		py := y1 + float32(pulsePos)*(y2-y1)

		pulseColor := color.RGBA{255, 255, 255, 180}
		if style.IsSpecter {
			pulseColor = color.RGBA{200, 220, 255, 140}
		}
		vector.DrawFilledCircle(dst, px, py, 3, pulseColor, true)
	}
}

// ZoomLevelFromScale determines the zoom level from a scale factor.
func ZoomLevelFromScale(scale float64) ZoomLevel {
	if scale < 0.3 {
		return ZoomMacro
	}
	if scale < 1.5 {
		return ZoomMeso
	}
	return ZoomMicro
}

// min returns the smaller of two uint8 values.
func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
