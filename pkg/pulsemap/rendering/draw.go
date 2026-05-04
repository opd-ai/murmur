// Package rendering provides Ebitengine-based node/edge drawing for the Pulse Map.
// Per TECHNICAL_IMPLEMENTATION.md §1.2, rendering uses Ebitengine v2.7+
// with Kage shaders for glow and ripple effects.
//

//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
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
	Color                color.RGBA
	Age                  float64 // Connection age in days
	Active               bool    // Recent wave propagation
	IsMiniGame           bool    // Active mini-game connection
	IsSpecter            bool    // Anonymous layer edge
	InteractionFrequency float64 // Message exchange rate (messages per day)
}

// RenderEdge draws a connection edge between two nodes.
// Per PULSE_MAP.md, edges are quadratic Bézier curves with age-based styling.
// Edge thickness is proportional to interaction frequency (message exchange rate).
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

	// Calculate edge thickness based on interaction frequency.
	// Base thickness: 1.5, scale logarithmically with frequency.
	// thickness = base + scale * ln(1 + frequency)
	// For frequency in messages/day: 0 msg/day → 1.5px, 10 msg/day → ~3px, 100 msg/day → ~6px
	baseThickness := 1.5
	thicknessScale := 1.5
	thickness := baseThickness + thicknessScale*math.Log(1+style.InteractionFrequency)

	// For simplicity, draw straight line (Bézier curves require more complex path)
	vector.StrokeLine(dst, x1, y1, x2, y2, float32(thickness), edgeColor, true)

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
// Edge thickness is proportional to interaction frequency (message exchange rate).
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

	// Calculate edge thickness based on interaction frequency.
	// Base thickness: 1.5, scale logarithmically with frequency.
	baseThickness := 1.5
	thicknessScale := 1.5
	thickness := baseThickness + thicknessScale*math.Log(1+style.InteractionFrequency)

	vector.StrokeLine(dst, x1, y1, x2, y2, float32(thickness), edgeColor, true)

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

// RenderAmplificationTrail draws an amplification relationship between amplifier and original author.
// Per ROADMAP.md line 621, amplification trails are visual connections distinct from regular edges.
// Trails are rendered as animated dashed lines with particles flowing from amplifier to original author.
func RenderAmplificationTrail(dst *ebiten.Image, ampX, ampY, origX, origY float32, trail AmplificationTrailData, zoom ZoomLevel, time float64) {
	// Calculate fade based on how recent the amplification is.
	// Trails fade over 60 seconds after amplification.
	fadeDuration := 60.0 // seconds
	fadeProgress := math.Min(trail.RecentSeconds/fadeDuration, 1.0)
	baseAlpha := 180.0 * (1.0 - fadeProgress) // Start at 180, fade to 0

	if baseAlpha < 10 {
		return // Skip rendering nearly invisible trails
	}

	// Amplification trail color: bright cyan/teal to distinguish from edges.
	// Per PULSE_MAP.md visual language: warm colors for Surface, cool for Anonymous.
	trailColor := color.RGBA{
		R: 100,
		G: 255,
		B: 220,
		A: uint8(baseAlpha),
	}

	// Draw dashed line from amplifier to original author.
	// Dash pattern: 8px on, 4px off.
	dashLength := 8.0
	gapLength := 4.0
	segmentLength := dashLength + gapLength

	dx := float64(origX - ampX)
	dy := float64(origY - ampY)
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 1.0 {
		return // Nodes too close to render trail
	}

	// Normalize direction.
	dirX := dx / distance
	dirY := dy / distance

	// Draw dashed segments.
	currentPos := 0.0
	for currentPos < distance {
		dashEnd := math.Min(currentPos+dashLength, distance)

		x1 := ampX + float32(currentPos*dirX)
		y1 := ampY + float32(currentPos*dirY)
		x2 := ampX + float32(dashEnd*dirX)
		y2 := ampY + float32(dashEnd*dirY)

		vector.StrokeLine(dst, x1, y1, x2, y2, 2.0, trailColor, true)

		currentPos += segmentLength
	}

	// Draw animated particles flowing along the trail.
	// Particles move from amplifier to original author.
	particleSpeed := 0.5 // units per second
	particleCount := 3

	for i := 0; i < particleCount; i++ {
		// Stagger particles along the trail.
		offset := float64(i) / float64(particleCount)
		particlePos := math.Mod((time*particleSpeed)+offset, 1.0)

		px := ampX + float32(particlePos*dx)
		py := ampY + float32(particlePos*dy)

		// Particle fades with trail.
		particleAlpha := uint8(baseAlpha * 0.9)
		particleColor := color.RGBA{150, 255, 230, particleAlpha}

		vector.DrawFilledCircle(dst, px, py, 2.5, particleColor, true)
	}

	// If the amplification has a comment, draw a small indicator at midpoint.
	if trail.HasComment {
		mx := (ampX + origX) / 2
		my := (ampY + origY) / 2

		// Comment indicator: small pulsing ring.
		ringPulse := 1.0 + 0.2*math.Sin(time*3.0)
		ringRadius := 5.0 * ringPulse
		ringAlpha := uint8(baseAlpha * 0.7)
		ringColor := color.RGBA{255, 255, 150, ringAlpha}

		vector.StrokeCircle(dst, mx, my, float32(ringRadius), 1.5, ringColor, true)
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

// defaultLabelFace is the font face for rendering node labels.
var defaultLabelFace = text.NewGoXFace(basicfont.Face7x13)

// RenderTextLabel draws a text label below a node at Micro zoom level.
// Per PULSE_MAP.md and ROADMAP.md, text labels show display name or pseudonym
// only at Micro zoom (close view, full detail).
func RenderTextLabel(dst *ebiten.Image, x, y float32, label string, isSpecter bool, zoom ZoomLevel) {
	// Only render text at Micro zoom level.
	if zoom != ZoomMicro || label == "" {
		return
	}

	// Position text below the node (offset by node radius + padding).
	textY := y + 20 // Approximate: node radius (~8-12px) + padding (~8px)

	// Set text color based on node type.
	var textColor color.RGBA
	if isSpecter {
		textColor = color.RGBA{180, 200, 220, 255} // Cool, light blue for Specters
	} else {
		textColor = color.RGBA{220, 220, 220, 255} // Light gray for Surface nodes
	}

	// Create text draw options.
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(textY))
	op.ColorScale.ScaleWithColor(textColor)

	// Center the text horizontally relative to the node.
	w, _ := text.Measure(label, defaultLabelFace, 0)
	op.GeoM.Translate(-w/2, 0)

	// Draw the text using basicfont.
	text.Draw(dst, label, defaultLabelFace, op)
}

// min returns the smaller of two uint8 values.
func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
