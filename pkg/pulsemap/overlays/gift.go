// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file implements Phantom Gift visual effects for recipient nodes on the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, gifts produce animated cosmetic effects in 3 tiers:
// Basic (Resonance 25+), Expanded (Resonance 50+), and Premium (Resonance 100+).
//
//go:build !noebiten
// +build !noebiten

package overlays

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// GiftEffect represents an active Phantom Gift effect on a node.
// Per ANONYMOUS_GAME_MECHANICS.md, gifts last 7 days with animated effects.
type GiftEffect struct {
	Effect    mechanics.EffectType // Type of visual effect
	Intensity float32              // 0-1, fades as gift nears expiration
	Phase     float32              // Animation phase (0 to 2π)
}

// GiftOverlay manages Phantom Gift visualization on the Pulse Map.
// Per ROADMAP.md line 519, shows animated cosmetic effects on recipient nodes.
type GiftOverlay struct {
	Effects map[string][]GiftEffect // Keyed by recipient node ID (hex pubkey)
}

// NewGiftOverlay creates a new gift overlay manager.
func NewGiftOverlay() *GiftOverlay {
	return &GiftOverlay{
		Effects: make(map[string][]GiftEffect),
	}
}

// AddEffect registers a gift effect for a recipient node.
func (o *GiftOverlay) AddEffect(nodeID string, effect mechanics.EffectType, intensity float32) {
	o.Effects[nodeID] = append(o.Effects[nodeID], GiftEffect{
		Effect:    effect,
		Intensity: intensity,
		Phase:     0,
	})
}

// RemoveEffect removes all effects for a node (e.g., when gift expires).
func (o *GiftOverlay) RemoveEffect(nodeID string) {
	delete(o.Effects, nodeID)
}

// RemoveExpiredEffect removes a specific effect by type from a node.
func (o *GiftOverlay) RemoveExpiredEffect(nodeID string, effect mechanics.EffectType) {
	effects := o.Effects[nodeID]
	filtered := effects[:0]
	for _, e := range effects {
		if e.Effect != effect {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		delete(o.Effects, nodeID)
	} else {
		o.Effects[nodeID] = filtered
	}
}

// Update advances animation phases for all effects.
// dt is delta time in seconds.
func (o *GiftOverlay) Update(dt float32) {
	for nodeID, effects := range o.Effects {
		for i := range effects {
			effects[i].Phase += dt * 2 // ~1 full cycle per 3 seconds
			if effects[i].Phase > math.Pi*2 {
				effects[i].Phase -= math.Pi * 2
			}
		}
		o.Effects[nodeID] = effects
	}
}

// HasEffects returns true if the node has any active gift effects.
func (o *GiftOverlay) HasEffects(nodeID string) bool {
	effects, ok := o.Effects[nodeID]
	return ok && len(effects) > 0
}

// GetEffectTier returns the highest tier effect active on a node.
// Returns 0 if no effects, otherwise 25 (Basic), 50 (Expanded), or 100 (Premium).
func (o *GiftOverlay) GetEffectTier(nodeID string) int {
	effects, ok := o.Effects[nodeID]
	if !ok || len(effects) == 0 {
		return 0
	}
	maxTier := 0
	for _, e := range effects {
		tier := mechanics.RequiredResonance(e.Effect)
		if tier > maxTier {
			maxTier = tier
		}
	}
	return maxTier
}

// Render draws all gift effects for a node.
// Per ANONYMOUS_GAME_MECHANICS.md, effects vary by tier:
// - Basic: soft glows, halos, gentle drifts
// - Expanded: orbiting geometrics, auroras, embers
// - Premium: particle systems, fluid sims, mandalas
func (o *GiftOverlay) Render(dst *ebiten.Image, nodeID string, nodeX, nodeY, nodeRadius, cameraX, cameraY, scale float32) {
	effects, ok := o.Effects[nodeID]
	if !ok || len(effects) == 0 {
		return
	}

	// Transform to screen coordinates
	screenX := (nodeX-cameraX)*scale + float32(dst.Bounds().Dx())/2
	screenY := (nodeY-cameraY)*scale + float32(dst.Bounds().Dy())/2
	screenRadius := nodeRadius * scale

	for _, e := range effects {
		o.renderEffect(dst, e, screenX, screenY, screenRadius)
	}
}

// renderEffect draws a single gift effect based on its type and tier.
func (o *GiftOverlay) renderEffect(dst *ebiten.Image, e GiftEffect, x, y, radius float32) {
	tier := mechanics.RequiredResonance(e.Effect)
	alpha := float32(200) * e.Intensity

	switch tier {
	case mechanics.GiftTierBasic:
		o.renderBasicEffect(dst, e, x, y, radius, alpha)
	case mechanics.GiftTierExpanded:
		o.renderExpandedEffect(dst, e, x, y, radius, alpha)
	case mechanics.GiftTierPremium:
		o.renderPremiumEffect(dst, e, x, y, radius, alpha)
	}
}

// renderBasicEffect draws Basic tier effects (Resonance 25+).
// Effects: soft glow pulse, faint halo ring, gentle particle drift, shimmer, warmth tint.
func (o *GiftOverlay) renderBasicEffect(dst *ebiten.Image, e GiftEffect, x, y, radius, alpha float32) {
	alphaU8 := uint8(alpha)
	sinPhase := float32(math.Sin(float64(e.Phase)))
	cosPhase := float32(math.Cos(float64(e.Phase)))

	switch e.Effect {
	case mechanics.EffectSoftGlowPulse:
		// Pulsing glow circle expanding and contracting
		glowRadius := radius * (1.3 + 0.2*sinPhase)
		pulseAlpha := uint8(float32(alphaU8) * (0.3 + 0.3*sinPhase))
		c := color.RGBA{255, 230, 180, pulseAlpha}
		vector.DrawFilledCircle(dst, x, y, glowRadius, c, true)

	case mechanics.EffectFaintHaloRing:
		// Rotating halo ring around node
		haloRadius := radius * 1.5
		c := color.RGBA{220, 200, 255, alphaU8 / 2}
		vector.StrokeCircle(dst, x, y, haloRadius, 2, c, true)

	case mechanics.EffectGentleParticleDrift:
		// Small particles drifting upward
		for i := 0; i < 5; i++ {
			offset := float32(i) * 0.4
			pY := y - radius*(0.5+float32(math.Mod(float64(e.Phase+offset), math.Pi*2))/math.Pi)
			pX := x + radius*0.3*float32(math.Sin(float64(e.Phase*2+offset)))
			c := color.RGBA{255, 255, 200, alphaU8 / 3}
			vector.DrawFilledCircle(dst, pX, pY, 2, c, true)
		}

	case mechanics.EffectShimmerOverlay:
		// Shimmering spots across node surface
		for i := 0; i < 3; i++ {
			angle := e.Phase + float32(i)*2.1
			shimmerX := x + radius*0.5*float32(math.Cos(float64(angle)))
			shimmerY := y + radius*0.5*float32(math.Sin(float64(angle)))
			shimmerAlpha := uint8(float32(alphaU8) * (0.5 + 0.5*float32(math.Sin(float64(angle*3)))))
			c := color.RGBA{255, 255, 255, shimmerAlpha}
			vector.DrawFilledCircle(dst, shimmerX, shimmerY, 3, c, true)
		}

	case mechanics.EffectWarmthTintShift:
		// Warm glow that shifts in intensity
		warmthAlpha := uint8(float32(alphaU8) * (0.2 + 0.15*sinPhase))
		c := color.RGBA{255, 150, 100, warmthAlpha}
		vector.DrawFilledCircle(dst, x, y, radius*1.2, c, true)

	default:
		// Fallback: simple glow
		c := color.RGBA{255, 220, 150, alphaU8 / 3}
		vector.DrawFilledCircle(dst, x, y, radius*1.3, c, true)
	}

	_ = cosPhase // Silence unused variable warning if not used
}

// renderExpandedEffect draws Expanded tier effects (Resonance 50+).
// Effects: orbiting geometrics, auroras, crystalline, embers, ripples, starlight.
func (o *GiftOverlay) renderExpandedEffect(dst *ebiten.Image, e GiftEffect, x, y, radius, alpha float32) {
	alphaU8 := uint8(alpha)
	sinPhase := float32(math.Sin(float64(e.Phase)))
	cosPhase := float32(math.Cos(float64(e.Phase)))

	switch e.Effect {
	case mechanics.EffectOrbitingGeometric:
		// 3 small geometric shapes orbiting the node
		for i := 0; i < 3; i++ {
			angle := e.Phase + float32(i)*2.09 // 120 degrees apart
			orbitRadius := radius * 1.6
			ox := x + orbitRadius*float32(math.Cos(float64(angle)))
			oy := y + orbitRadius*float32(math.Sin(float64(angle)))
			c := color.RGBA{150, 200, 255, alphaU8}
			// Draw small diamond shape
			vector.StrokeLine(dst, ox-4, oy, ox, oy-6, 2, c, true)
			vector.StrokeLine(dst, ox, oy-6, ox+4, oy, 2, c, true)
			vector.StrokeLine(dst, ox+4, oy, ox, oy+6, 2, c, true)
			vector.StrokeLine(dst, ox, oy+6, ox-4, oy, 2, c, true)
		}

	case mechanics.EffectAuroraColorShift:
		// Color-shifting aurora bands
		for band := 0; band < 3; band++ {
			bandY := y - radius*0.5 - float32(band)*8
			hueShift := e.Phase + float32(band)*0.5
			r := uint8(128 + 127*float32(math.Sin(float64(hueShift))))
			g := uint8(128 + 127*float32(math.Sin(float64(hueShift+2))))
			b := uint8(128 + 127*float32(math.Sin(float64(hueShift+4))))
			c := color.RGBA{r, g, b, alphaU8 / 2}
			vector.StrokeLine(dst, x-radius, bandY, x+radius, bandY, 3, c, true)
		}

	case mechanics.EffectCrystallineFracture:
		// Crystalline fracture lines radiating from center
		for i := 0; i < 6; i++ {
			angle := float32(i) * 1.047 // 60 degrees
			length := radius * (1.0 + 0.3*sinPhase)
			c := color.RGBA{200, 230, 255, alphaU8}
			endX := x + length*float32(math.Cos(float64(angle)))
			endY := y + length*float32(math.Sin(float64(angle)))
			vector.StrokeLine(dst, x, y, endX, endY, 1, c, true)
		}

	case mechanics.EffectEmberTrails:
		// Glowing ember particles rising
		for i := 0; i < 4; i++ {
			offset := float32(i) * 0.5
			pY := y - radius*float32(math.Mod(float64(e.Phase+offset), math.Pi))/math.Pi*2
			pX := x + radius*0.4*sinPhase
			emberAlpha := uint8(float32(alphaU8) * (1 - float32(math.Mod(float64(e.Phase+offset), math.Pi))/math.Pi))
			c := color.RGBA{255, 100, 50, emberAlpha}
			vector.DrawFilledCircle(dst, pX+float32(i)*5, pY, 3, c, true)
		}

	case mechanics.EffectRippleDistortion:
		// Expanding ripple circles
		for ring := 0; ring < 3; ring++ {
			rippleRadius := radius * (1.0 + float32(math.Mod(float64(e.Phase+float32(ring)*0.7), math.Pi*2))/math.Pi)
			rippleAlpha := uint8(float32(alphaU8) * (1 - float32(math.Mod(float64(e.Phase+float32(ring)*0.7), math.Pi*2))/(math.Pi*2)))
			c := color.RGBA{180, 200, 255, rippleAlpha}
			vector.StrokeCircle(dst, x, y, rippleRadius, 1, c, true)
		}

	case mechanics.EffectStarlightSparkle:
		// Twinkling star sparkles
		for i := 0; i < 6; i++ {
			angle := e.Phase*0.3 + float32(i)*1.05
			dist := radius * (0.8 + 0.4*float32(math.Sin(float64(angle*3))))
			starX := x + dist*float32(math.Cos(float64(angle*2)))
			starY := y + dist*float32(math.Sin(float64(angle*2)))
			starAlpha := uint8(float32(alphaU8) * (0.5 + 0.5*float32(math.Sin(float64(e.Phase*4+float32(i))))))
			c := color.RGBA{255, 255, 255, starAlpha}
			vector.DrawFilledCircle(dst, starX, starY, 2, c, true)
		}

	default:
		// Fallback: orbiting glow
		orbitX := x + radius*1.5*cosPhase
		orbitY := y + radius*1.5*sinPhase
		c := color.RGBA{200, 200, 255, alphaU8}
		vector.DrawFilledCircle(dst, orbitX, orbitY, 5, c, true)
	}
}

// renderPremiumEffect draws Premium tier effects (Resonance 100+).
// Effects: multi-particle systems, fluid sims, mandalas, void gravitation.
func (o *GiftOverlay) renderPremiumEffect(dst *ebiten.Image, e GiftEffect, x, y, radius, alpha float32) {
	alphaU8 := uint8(alpha)

	switch e.Effect {
	case mechanics.EffectMultiParticleSystem:
		o.renderMultiParticleSystem(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectFluidSimulation:
		o.renderFluidSimulation(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectGeometricMandala:
		o.renderGeometricMandala(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectVoidGravitation:
		o.renderVoidGravitation(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectPrismaticRefraction:
		o.renderPrismaticRefraction(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectNebulaeCloud:
		o.renderNebulaeCloud(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectElectricArc:
		o.renderElectricArc(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectCrystalGrowth:
		o.renderCrystalGrowth(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectPhoenixFlame:
		o.renderPhoenixFlame(dst, e, x, y, radius, alphaU8)
	case mechanics.EffectShadowWraith:
		o.renderShadowWraith(dst, e, x, y, radius, alphaU8)
	default:
		o.renderFallbackGlow(dst, x, y, radius, alphaU8)
	}
}

// renderMultiParticleSystem renders dense particle cloud orbiting.
func (o *GiftOverlay) renderMultiParticleSystem(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for i := 0; i < 20; i++ {
		angle := e.Phase + float32(i)*0.314
		orbitRadius := radius * (1.2 + 0.4*float32(math.Sin(float64(angle*2))))
		px := x + orbitRadius*float32(math.Cos(float64(angle)))
		py := y + orbitRadius*float32(math.Sin(float64(angle)))
		particleAlpha := uint8(float32(alphaU8) * (0.3 + 0.3*float32(math.Sin(float64(e.Phase+float32(i))))))
		c := color.RGBA{200, 220, 255, particleAlpha}
		vector.DrawFilledCircle(dst, px, py, 2, c, true)
	}
}

// renderFluidSimulation renders flowing liquid-like curves.
func (o *GiftOverlay) renderFluidSimulation(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for curve := 0; curve < 4; curve++ {
		baseAngle := e.Phase + float32(curve)*1.57
		for seg := 0; seg < 8; seg++ {
			t := float32(seg) / 8.0
			flow := radius * (1.0 + 0.3*float32(math.Sin(float64(baseAngle+t*2))))
			fx := x + flow*float32(math.Cos(float64(baseAngle+t)))
			fy := y + flow*float32(math.Sin(float64(baseAngle+t)))
			segAlpha := uint8(float32(alphaU8) * (0.5 + 0.3*t))
			c := color.RGBA{100, 150, 255, segAlpha}
			vector.DrawFilledCircle(dst, fx, fy, 3-t*2, c, true)
		}
	}
}

// renderGeometricMandala renders rotating mandala pattern.
func (o *GiftOverlay) renderGeometricMandala(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	layers := 3
	for layer := 0; layer < layers; layer++ {
		layerRadius := radius * (1.0 + float32(layer)*0.3)
		layerAngle := e.Phase * (1 + float32(layer)*0.3)
		points := 6 + layer*2
		for i := 0; i < points; i++ {
			angle := layerAngle + float32(i)*2*math.Pi/float32(points)
			px := x + layerRadius*float32(math.Cos(float64(angle)))
			py := y + layerRadius*float32(math.Sin(float64(angle)))
			c := color.RGBA{220, 200, 255, alphaU8}
			vector.DrawFilledCircle(dst, px, py, 3, c, true)
			nextAngle := layerAngle + float32(i+1)*2*math.Pi/float32(points)
			nx := x + layerRadius*float32(math.Cos(float64(nextAngle)))
			ny := y + layerRadius*float32(math.Sin(float64(nextAngle)))
			vector.StrokeLine(dst, px, py, nx, ny, 1, c, true)
		}
	}
}

// renderVoidGravitation renders dark void with particles being pulled in.
func (o *GiftOverlay) renderVoidGravitation(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	voidAlpha := uint8(float32(alphaU8) * 0.3)
	vector.DrawFilledCircle(dst, x, y, radius*0.5, color.RGBA{20, 20, 40, voidAlpha}, true)
	for i := 0; i < 12; i++ {
		spiralPhase := e.Phase + float32(i)*0.52
		spiralRadius := radius * (2.0 - float32(math.Mod(float64(spiralPhase), math.Pi*2))/(math.Pi))
		angle := spiralPhase * 2
		px := x + spiralRadius*float32(math.Cos(float64(angle)))
		py := y + spiralRadius*float32(math.Sin(float64(angle)))
		c := color.RGBA{150, 100, 200, alphaU8}
		vector.DrawFilledCircle(dst, px, py, 2, c, true)
	}
}

// renderPrismaticRefraction renders rainbow light refraction beams.
func (o *GiftOverlay) renderPrismaticRefraction(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for beam := 0; beam < 6; beam++ {
		beamAngle := e.Phase*0.5 + float32(beam)*1.05
		beamLength := radius * 2
		hue := float32(beam) / 6.0
		r := uint8(255 * (0.5 + 0.5*float32(math.Sin(float64(hue*6.28)))))
		g := uint8(255 * (0.5 + 0.5*float32(math.Sin(float64(hue*6.28+2)))))
		b := uint8(255 * (0.5 + 0.5*float32(math.Sin(float64(hue*6.28+4)))))
		c := color.RGBA{r, g, b, alphaU8 / 2}
		endX := x + beamLength*float32(math.Cos(float64(beamAngle)))
		endY := y + beamLength*float32(math.Sin(float64(beamAngle)))
		vector.StrokeLine(dst, x, y, endX, endY, 2, c, true)
	}
}

// renderNebulaeCloud renders colorful nebula clouds.
func (o *GiftOverlay) renderNebulaeCloud(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for cloud := 0; cloud < 8; cloud++ {
		cloudAngle := e.Phase*0.3 + float32(cloud)*0.79
		cloudDist := radius * (0.8 + 0.5*float32(math.Sin(float64(cloudAngle*2))))
		cx := x + cloudDist*float32(math.Cos(float64(cloudAngle)))
		cy := y + cloudDist*float32(math.Sin(float64(cloudAngle)))
		cloudSize := 8 + 4*float32(math.Sin(float64(e.Phase+float32(cloud))))
		c := color.RGBA{
			uint8(100 + 100*float32(math.Sin(float64(cloudAngle)))),
			uint8(50 + 50*float32(math.Sin(float64(cloudAngle+2)))),
			uint8(150 + 100*float32(math.Sin(float64(cloudAngle+4)))),
			alphaU8 / 3,
		}
		vector.DrawFilledCircle(dst, cx, cy, cloudSize, c, true)
	}
}

// renderElectricArc renders electric lightning arcs.
func (o *GiftOverlay) renderElectricArc(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for arc := 0; arc < 4; arc++ {
		arcAngle := e.Phase + float32(arc)*1.57
		arcLength := radius * 1.5
		prevX, prevY := x, y
		segments := 6
		for seg := 1; seg <= segments; seg++ {
			t := float32(seg) / float32(segments)
			jitter := 5 * float32(math.Sin(float64(e.Phase*10+float32(seg+arc))))
			endX := x + arcLength*t*float32(math.Cos(float64(arcAngle))) + jitter
			endY := y + arcLength*t*float32(math.Sin(float64(arcAngle))) + jitter
			c := color.RGBA{200, 220, 255, alphaU8}
			vector.StrokeLine(dst, prevX, prevY, endX, endY, 2, c, true)
			prevX, prevY = endX, endY
		}
	}
}

// renderCrystalGrowth renders growing crystal formations.
func (o *GiftOverlay) renderCrystalGrowth(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	crystals := 5
	for c := 0; c < crystals; c++ {
		crystalAngle := float32(c) * 1.26
		crystalLength := radius * (0.8 + 0.4*float32(math.Sin(float64(e.Phase+float32(c)))))
		endX := x + crystalLength*float32(math.Cos(float64(crystalAngle)))
		endY := y + crystalLength*float32(math.Sin(float64(crystalAngle)))
		col := color.RGBA{180, 230, 255, alphaU8}
		vector.StrokeLine(dst, x, y, endX, endY, 3, col, true)
		vector.DrawFilledCircle(dst, endX, endY, 4, col, true)
	}
}

// renderPhoenixFlame renders rising flame particles.
func (o *GiftOverlay) renderPhoenixFlame(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	for flame := 0; flame < 15; flame++ {
		flamePhase := e.Phase + float32(flame)*0.2
		flameY := y - radius*float32(math.Mod(float64(flamePhase), math.Pi))/math.Pi*3
		flameX := x + radius*0.3*float32(math.Sin(float64(flamePhase*3)))
		flameAlpha := uint8(float32(alphaU8) * (1 - float32(math.Mod(float64(flamePhase), math.Pi))/math.Pi))
		r := uint8(255)
		g := uint8(150 + 100*float32(math.Mod(float64(flamePhase), math.Pi))/math.Pi)
		b := uint8(50)
		c := color.RGBA{r, g, b, flameAlpha}
		size := 4 * (1 - float32(math.Mod(float64(flamePhase), math.Pi))/math.Pi)
		vector.DrawFilledCircle(dst, flameX, flameY, size, c, true)
	}
}

// renderShadowWraith renders ethereal shadow wisps.
func (o *GiftOverlay) renderShadowWraith(dst *ebiten.Image, e GiftEffect, x, y, radius float32, alphaU8 uint8) {
	sinPhase := float32(math.Sin(float64(e.Phase)))
	for wisp := 0; wisp < 6; wisp++ {
		wispAngle := e.Phase*0.7 + float32(wisp)*1.05
		wispDist := radius * (1.0 + 0.5*sinPhase)
		wx := x + wispDist*float32(math.Cos(float64(wispAngle)))
		wy := y + wispDist*float32(math.Sin(float64(wispAngle)))
		for trail := 0; trail < 4; trail++ {
			trailT := float32(trail) * 0.25
			tx := wx - 10*trailT*float32(math.Cos(float64(wispAngle)))
			ty := wy - 10*trailT*float32(math.Sin(float64(wispAngle)))
			trailAlpha := uint8(float32(alphaU8) * (1 - trailT) * 0.5)
			c := color.RGBA{80, 60, 120, trailAlpha}
			vector.DrawFilledCircle(dst, tx, ty, 4-trailT*2, c, true)
		}
	}
}

// renderFallbackGlow renders elaborate glow as fallback.
func (o *GiftOverlay) renderFallbackGlow(dst *ebiten.Image, x, y, radius float32, alphaU8 uint8) {
	for ring := 0; ring < 3; ring++ {
		ringRadius := radius * (1.2 + float32(ring)*0.3)
		c := color.RGBA{200, 180, 255, alphaU8 / uint8(ring+1)}
		vector.StrokeCircle(dst, x, y, ringRadius, 2, c, true)
	}
}

// RenderAll draws gift effects for multiple nodes in a batch.
// nodePositions maps node IDs to their (x, y) world positions.
func (o *GiftOverlay) RenderAll(dst *ebiten.Image, nodePositions map[string][2]float32, nodeRadius, cameraX, cameraY, scale float32) {
	for nodeID := range o.Effects {
		pos, ok := nodePositions[nodeID]
		if !ok {
			continue
		}
		o.Render(dst, nodeID, pos[0], pos[1], nodeRadius, cameraX, cameraY, scale)
	}
}

// EffectCount returns the number of active effects for a node.
func (o *GiftOverlay) EffectCount(nodeID string) int {
	return len(o.Effects[nodeID])
}

// TotalEffectCount returns the total number of active effects across all nodes.
func (o *GiftOverlay) TotalEffectCount() int {
	total := 0
	for _, effects := range o.Effects {
		total += len(effects)
	}
	return total
}

// Clear removes all gift effects.
func (o *GiftOverlay) Clear() {
	o.Effects = make(map[string][]GiftEffect)
}

// UpdateIntensity updates the intensity of all effects for a node.
// Used to fade effects as gifts near expiration.
func (o *GiftOverlay) UpdateIntensity(nodeID string, intensity float32) {
	effects, ok := o.Effects[nodeID]
	if !ok {
		return
	}
	for i := range effects {
		effects[i].Intensity = intensity
	}
	o.Effects[nodeID] = effects
}
