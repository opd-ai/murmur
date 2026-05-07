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
// At Macro zoom, nodes are rendered as simple colored dots without detail.
func RenderNode(dst *ebiten.Image, x, y float32, style NodeStyle, zoom ZoomLevel) {
	radius := computeNodeRadius(style)

	// Per PULSE_MAP.md §Macro View: "Nodes are rendered as small colored dots
	// without sigils, labels, or halos" for widest zoom level.
	if zoom == ZoomMacro {
		renderNodeMacro(dst, x, y, style)
		return
	}

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
// At Macro zoom, nodes are rendered as simple colored dots without detail.
func RenderNodeWithTime(dst *ebiten.Image, x, y float32, style NodeStyle, zoom ZoomLevel, time float64) {
	radius := computeNodeRadius(style)

	// Per PULSE_MAP.md §Macro View: "Nodes are rendered as small colored dots
	// without sigils, labels, or halos" for widest zoom level.
	if zoom == ZoomMacro {
		renderNodeMacro(dst, x, y, style)
		return
	}

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
// Returns the radius in world units (equivalent to screen pixels at Scale=1.0).
// The minimum is rBase=4.0 world units.
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

// renderNodeMacro renders a node at Macro zoom level as a simple colored dot.
// Per PULSE_MAP.md §Macro View: "Nodes are rendered as small colored dots without
// sigils, labels, or halos. Connections are rendered as faint lines."
func renderNodeMacro(dst *ebiten.Image, x, y float32, style NodeStyle) {
	const macroRadius = 2.5 // Fixed small radius for macro view dots

	// Use core color with full opacity for visibility at distance.
	dotColor := style.CoreColor
	if style.IsSpecter {
		// Specters retain slight translucency even at macro level.
		dotColor.A = uint8(float64(style.CoreColor.A) * 0.8)
	}

	// Draw a simple filled circle.
	vector.DrawFilledCircle(dst, x, y, macroRadius, dotColor, true)

	// If selected, add a subtle highlight ring.
	if style.Selected {
		highlightColor := color.RGBA{255, 255, 255, 200}
		vector.StrokeCircle(dst, x, y, macroRadius+1.5, 1.0, highlightColor, true)
	}
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

	// Derive a stable per-node seed directly from the core colour components.
	// Using a multiplicative hash avoids allocating a *rand.Rand on every call
	// (per audit MEDIUM finding: 30 000 allocations/sec at 500 nodes × 60 fps).
	seed := uint32(style.CoreColor.R)*2654435761 ^ uint32(style.CoreColor.G)*2246822519 ^ uint32(style.CoreColor.B)*1812433253

	for i := 0; i < particleCount; i++ {
		// Per-particle deterministic phase offset via Knuth multiplicative hash.
		h := (seed ^ uint32(i)*2654435769) * 2246822519
		phaseOffset := float64(h) / float64(^uint32(0)) * 2 * math.Pi

		// Per-particle speed variation in [0.5, 1.0].
		h2 := (h ^ 0xdeadbeef) * 1812433253
		speedVariation := 0.5 + float64(h2)/float64(^uint32(0))*0.5

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
// At Macro zoom, edges are rendered as faint lines with sparse sampling.
func RenderEdge(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel) {
	if zoom == ZoomMacro {
		renderEdgeMacro(dst, x1, y1, x2, y2, style)
		return
	}

	alpha := calculateEdgeAlpha(style)
	edgeColor := buildEdgeColor(style, alpha)
	thickness := calculateEdgeThickness(style)

	vector.StrokeLine(dst, x1, y1, x2, y2, float32(thickness), edgeColor, true)

	if style.Active {
		renderActivityPulse(dst, x1, y1, x2, y2, style.IsSpecter)
	}
}

// calculateEdgeAlpha computes edge opacity based on age and type.
func calculateEdgeAlpha(style EdgeStyle) uint8 {
	var alpha uint8 = 50
	if style.Age > 90 {
		alpha = 80
	} else if style.Age < 7 {
		alpha = 40
	}

	if style.IsSpecter {
		alpha = uint8(float32(alpha) * 0.7)
	}

	return alpha
}

// buildEdgeColor constructs the edge color from style and alpha.
func buildEdgeColor(style EdgeStyle, alpha uint8) color.RGBA {
	return color.RGBA{
		R: style.Color.R,
		G: style.Color.G,
		B: style.Color.B,
		A: alpha,
	}
}

// calculateEdgeThickness computes thickness from interaction frequency.
func calculateEdgeThickness(style EdgeStyle) float64 {
	const baseThickness = 1.5
	const thicknessScale = 1.5
	return baseThickness + thicknessScale*math.Log(1+style.InteractionFrequency)
}

// renderActivityPulse draws an activity indicator at edge midpoint.
func renderActivityPulse(dst *ebiten.Image, x1, y1, x2, y2 float32, isSpecter bool) {
	pulseColor := color.RGBA{255, 255, 255, 180}
	if isSpecter {
		pulseColor = color.RGBA{200, 220, 255, 140}
	}

	mx := (x1 + x2) / 2
	my := (y1 + y2) / 2
	vector.DrawFilledCircle(dst, mx, my, 3, pulseColor, true)
}

// RenderEdgeWithTime draws an edge with time-based animations.
// Edge thickness is proportional to interaction frequency (message exchange rate).
// At Macro zoom, edges are rendered as faint lines with sparse sampling.
func RenderEdgeWithTime(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel, time float64) {
	if zoom == ZoomMacro {
		renderEdgeMacro(dst, x1, y1, x2, y2, style)
		return
	}

	alpha := calculateAnimatedEdgeAlpha(style, time)
	edgeColor := buildEdgeColor(style, alpha)
	thickness := calculateEdgeThickness(style)

	vector.StrokeLine(dst, x1, y1, x2, y2, float32(thickness), edgeColor, true)

	if style.Active {
		renderAnimatedActivityPulse(dst, x1, y1, x2, y2, style.IsSpecter, time)
	}
}

// calculateAnimatedEdgeAlpha computes time-animated edge opacity.
func calculateAnimatedEdgeAlpha(style EdgeStyle, time float64) uint8 {
	var alpha uint8 = 50
	if style.Age > 90 {
		alpha = 80
	} else if style.Age < 7 {
		alpha = 40
	}

	if style.IsSpecter {
		shimmer := 0.7 + 0.3*math.Sin(time*2.0)
		alpha = uint8(float64(alpha) * shimmer)
	}

	return alpha
}

// renderAnimatedActivityPulse draws a moving activity indicator along the edge.
func renderAnimatedActivityPulse(dst *ebiten.Image, x1, y1, x2, y2 float32, isSpecter bool, time float64) {
	pulsePos := math.Mod(time*0.5, 1.0)
	px := x1 + float32(pulsePos)*(x2-x1)
	py := y1 + float32(pulsePos)*(y2-y1)

	pulseColor := color.RGBA{255, 255, 255, 180}
	if isSpecter {
		pulseColor = color.RGBA{200, 220, 255, 140}
	}
	vector.DrawFilledCircle(dst, px, py, 3, pulseColor, true)
}

// renderEdgeMacro renders an edge at Macro zoom level as a faint line.
// Per PULSE_MAP.md §Macro View: "Connections are rendered as faint lines,
// with only a sparse sample visible to prevent visual overload."
func renderEdgeMacro(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle) {
	// Very low opacity at macro level to avoid visual overload.
	alpha := uint8(20)
	if style.IsSpecter {
		alpha = uint8(15) // Even fainter for anonymous layer
	}

	edgeColor := color.RGBA{
		R: style.Color.R,
		G: style.Color.G,
		B: style.Color.B,
		A: alpha,
	}

	// Thin fixed-width line at macro level.
	const macroThickness = 0.8
	vector.StrokeLine(dst, x1, y1, x2, y2, macroThickness, edgeColor, true)
}

// RenderAmplificationTrail draws an amplification relationship between amplifier and original author.
// Per ROADMAP.md line 621, amplification trails are visual connections distinct from regular edges.
// Trails are rendered as animated dashed lines with particles flowing from amplifier to original author.
func RenderAmplificationTrail(dst *ebiten.Image, ampX, ampY, origX, origY float32, trail AmplificationTrailData, zoom ZoomLevel, time float64) {
	baseAlpha := calculateTrailFade(trail.RecentSeconds)
	if baseAlpha < 10 {
		return // Skip rendering nearly invisible trails
	}

	trailColor := createTrailColor(baseAlpha)
	distance, dirX, dirY := calculateTrailVector(ampX, ampY, origX, origY)
	if distance < 1.0 {
		return // Nodes too close to render trail
	}

	drawDashedTrailLine(dst, ampX, ampY, distance, dirX, dirY, trailColor)
	drawTrailParticles(dst, ampX, ampY, origX, origY, baseAlpha, time, distance, dirX, dirY)

	if trail.HasComment {
		drawCommentIndicator(dst, ampX, ampY, origX, origY, baseAlpha, time)
	}
}

// calculateTrailFade computes fade alpha based on recency (0-180 alpha over 60s).
func calculateTrailFade(recentSeconds float64) float64 {
	fadeDuration := 60.0
	fadeProgress := math.Min(recentSeconds/fadeDuration, 1.0)
	return 180.0 * (1.0 - fadeProgress)
}

// createTrailColor returns the cyan/teal trail color with the given alpha.
func createTrailColor(baseAlpha float64) color.RGBA {
	return color.RGBA{R: 100, G: 255, B: 220, A: uint8(baseAlpha)}
}

// calculateTrailVector computes distance and direction from amplifier to original.
func calculateTrailVector(ampX, ampY, origX, origY float32) (distance, dirX, dirY float64) {
	dx := float64(origX - ampX)
	dy := float64(origY - ampY)
	distance = math.Sqrt(dx*dx + dy*dy)
	if distance > 0 {
		dirX = dx / distance
		dirY = dy / distance
	}
	return distance, dirX, dirY
}

// drawDashedTrailLine draws the dashed trail line (8px on, 4px off).
func drawDashedTrailLine(dst *ebiten.Image, ampX, ampY float32, distance, dirX, dirY float64, trailColor color.RGBA) {
	dashLength := 8.0
	segmentLength := 12.0 // 8px on + 4px gap

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
}

// drawTrailParticles draws 3 animated particles flowing along the trail.
func drawTrailParticles(dst *ebiten.Image, ampX, ampY, origX, origY float32, baseAlpha, time, distance, dirX, dirY float64) {
	particleSpeed := 0.5
	particleCount := 3
	dx := float64(origX - ampX)
	dy := float64(origY - ampY)

	for i := 0; i < particleCount; i++ {
		offset := float64(i) / float64(particleCount)
		particlePos := math.Mod((time*particleSpeed)+offset, 1.0)
		px := ampX + float32(particlePos*dx)
		py := ampY + float32(particlePos*dy)
		particleAlpha := uint8(baseAlpha * 0.9)
		particleColor := color.RGBA{150, 255, 230, particleAlpha}
		vector.DrawFilledCircle(dst, px, py, 2.5, particleColor, true)
	}
}

// drawCommentIndicator draws a small pulsing ring at trail midpoint if amplification has a comment.
func drawCommentIndicator(dst *ebiten.Image, ampX, ampY, origX, origY float32, baseAlpha, time float64) {
	mx := (ampX + origX) / 2
	my := (ampY + origY) / 2
	ringPulse := 1.0 + 0.2*math.Sin(time*3.0)
	ringRadius := 5.0 * ringPulse
	ringAlpha := uint8(baseAlpha * 0.7)
	ringColor := color.RGBA{255, 255, 150, ringAlpha}
	vector.StrokeCircle(dst, mx, my, float32(ringRadius), 1.5, ringColor, true)
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
	// Per PULSE_MAP.md: labels visible at Meso and Micro zoom levels.
	// Macro view has no labels.
	if zoom == ZoomMacro || label == "" {
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
