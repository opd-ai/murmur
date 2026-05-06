// Package rendering provides artifact drawing functions for cross-layer mechanics.
// Per PLAN.md Step 6, these functions were extracted from drawCrossLayerArtifacts
// to reduce cyclomatic complexity from 34 to <15.

//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawSpecterMarks renders Specter Marks as orbiting icons around a node.
// Per ANONYMOUS_GAME_MECHANICS.md, marks appear as colored glowing circles
// that orbit the target node and decay over their lifetime.
func (r *Renderer) drawSpecterMarks(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return // No pubkey, can't query
	}

	marks, err := r.store.ListMarksForTarget(nodeData.PublicKey)
	if err != nil || len(marks) == 0 {
		return // No marks or query failed
	}

	for i, mark := range marks {
		if mark == nil {
			continue
		}

		// Calculate age for visibility decay (marks decay over 30 days).
		createdAt := time.Unix(mark.CreatedAt, 0)
		expiresAt := time.Unix(mark.ExpiresAt, 0)
		age := time.Since(createdAt)
		lifetime := expiresAt.Sub(createdAt)

		// Skip expired marks.
		if time.Now().After(expiresAt) {
			continue
		}

		// Calculate visibility (1.0 → 0.0 linear decay over lifetime).
		visibility := float32(1.0 - (float64(age) / float64(lifetime)))
		if visibility < 0 {
			visibility = 0
		}

		// Stack orbits for multiple marks.
		orbitRadius := 24.0 + float32(i)*6.0

		// Orbit angle based on elapsed time and mark ID for variety.
		orbitSpeed := 0.5 + float32(mark.Id[0]%64)/128.0 // 0.5 to 1.0 rad/sec
		orbitAngle := float32(r.time) * orbitSpeed

		// Calculate orbit position.
		x := nodeX + float32(math.Cos(float64(orbitAngle)))*orbitRadius
		y := nodeY + float32(math.Sin(float64(orbitAngle)))*orbitRadius

		// Draw mark icon as a small circle with pulsing glow.
		alpha := uint8(visibility * 200)
		markColor := color.RGBA{
			R: 100 + mark.SpecterPubkey[0]%100,
			G: 150,
			B: 200 + mark.SpecterPubkey[1]%55,
			A: alpha,
		}

		// Draw outer glow (pulsing).
		pulsePhase := float32(math.Sin(float64(r.time) * 2))
		glowRadius := 5.0 + pulsePhase*2.0
		glowAlpha := uint8(float32(alpha) * 0.3)
		glowColor := color.RGBA{markColor.R, markColor.G, markColor.B, glowAlpha}
		vector.DrawFilledCircle(screen, x, y, glowRadius, glowColor, false)

		// Draw core icon.
		vector.DrawFilledCircle(screen, x, y, 3.0, markColor, false)
	}
}

// drawPhantomGifts renders Phantom Gifts as floating particles around a node.
// Gifts appear as colored particles that orbit the recipient node and decay over time.
func (r *Renderer) drawPhantomGifts(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	gifts, err := r.store.GetActiveGiftsForRecipient(nodeData.PublicKey, time.Now().Unix())
	if err != nil || len(gifts) == 0 {
		return
	}

	for i, gift := range gifts {
		if gift == nil {
			continue
		}

		// Calculate visibility decay.
		createdAt := time.Unix(gift.CreatedAt, 0)
		expiresAt := time.Unix(gift.ExpiresAt, 0)
		age := time.Since(createdAt)
		lifetime := expiresAt.Sub(createdAt)
		visibility := float32(1.0 - (float64(age) / float64(lifetime)))
		if visibility < 0 {
			visibility = 0
		}

		// Draw particle animation for gifts.
		particleCount := 3 + i
		for p := 0; p < particleCount; p++ {
			angle := float32(p)*2.0*math.Pi/float32(particleCount) + float32(r.time)*0.5
			radius := 18.0 + float32(math.Sin(float64(r.time)*1.5+float64(p)))*4.0
			px := nodeX + float32(math.Cos(float64(angle)))*radius
			py := nodeY + float32(math.Sin(float64(angle)))*radius

			alpha := uint8(visibility * 180)
			giftColor := color.RGBA{
				R: 255,
				G: 200 - uint8(gift.EffectType*20),
				B: 150,
				A: alpha,
			}
			vector.DrawFilledCircle(screen, px, py, 2.5, giftColor, false)
		}
	}
}

// drawCipherPuzzles renders Cipher Puzzles as rotating hexagons near a node.
func (r *Renderer) drawCipherPuzzles(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	puzzles, err := r.store.GetActivePuzzlesNearNode(nodeData.PublicKey, 100.0)
	if err != nil || len(puzzles) == 0 {
		return
	}

	for i, puzzle := range puzzles {
		if puzzle == nil || i >= 3 { // Limit to 3 visible puzzles
			continue
		}

		// Draw rotating hexagon icon for puzzles.
		hexRadius := float32(8.0)
		hexX := nodeX + float32(i-1)*20.0
		hexY := nodeY - 30.0
		rotationAngle := float32(r.time) * 0.8

		// Draw hexagon (6 sides).
		puzzleColor := color.RGBA{R: 150, G: 100, B: 200, A: 200}
		for side := 0; side < 6; side++ {
			angle1 := rotationAngle + float32(side)*float32(math.Pi)/3.0
			angle2 := rotationAngle + float32(side+1)*float32(math.Pi)/3.0
			x1 := hexX + float32(math.Cos(float64(angle1)))*hexRadius
			y1 := hexY + float32(math.Sin(float64(angle1)))*hexRadius
			x2 := hexX + float32(math.Cos(float64(angle2)))*hexRadius
			y2 := hexY + float32(math.Sin(float64(angle2)))*hexRadius
			vector.StrokeLine(screen, x1, y1, x2, y2, 2.0, puzzleColor, false)
		}
	}
}

// drawSpecterHunts renders Specter Hunt fragments as glowing markers around a node.
func (r *Renderer) drawSpecterHunts(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	hunts, err := r.store.GetActiveHuntsWithFragmentsNear(nodeData.PublicKey, 100.0)
	if err != nil || len(hunts) == 0 || len(hunts) > 2 {
		return // Max 2 visible hunts
	}

	for i, hunt := range hunts {
		if hunt == nil {
			continue
		}

		// Draw scattered glowing markers for hunt fragments.
		fragmentCount := 4
		for f := 0; f < fragmentCount; f++ {
			angle := float32(f)*2.0*math.Pi/float32(fragmentCount) + float32(r.time)*0.3
			radius := 25.0 + float32(i)*5.0
			fx := nodeX + float32(math.Cos(float64(angle)))*radius
			fy := nodeY + float32(math.Sin(float64(angle)))*radius

			// Pulsing fragment markers.
			pulse := float32(math.Sin(float64(r.time)*3.0 + float64(f)))
			fragSize := 2.0 + pulse*1.0
			huntColor := color.RGBA{R: 200, G: 50, B: 50, A: 180}
			vector.DrawFilledCircle(screen, fx, fy, fragSize, huntColor, false)
		}
	}
}

// drawTerritoryInfluence renders Territory influence as a boundary around a node.
func (r *Renderer) drawTerritoryInfluence(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	territory, err := r.store.GetTerritoryInfluenceAt(nodeData.PublicKey)
	if err != nil || territory == nil || territory.Influence <= 0 {
		return
	}

	// Draw translucent boundary watermark around node.
	boundaryRadius := 35.0 * (float32(territory.Influence) / 100.0)
	boundaryAlpha := uint8(50 + territory.Influence/2)
	territoryColor := color.RGBA{
		R: 80,
		G: 120 + uint8(territory.Influence),
		B: 80,
		A: boundaryAlpha,
	}
	vector.StrokeCircle(screen, nodeX, nodeY, boundaryRadius, 1.5, territoryColor, false)
}

// drawOraclePools renders Oracle Pools as swirling vortex patterns.
func (r *Renderer) drawOraclePools(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	_, ok := getSingleActiveItem(nodeData.PublicKey, r.store.GetActiveOraclePoolsNearNode)
	if !ok {
		return
	}

	// Draw swirling vortex icon.
	spiralTurns := float32(2.0)
	spiralPoints := 20
	oracleColor := color.RGBA{R: 200, G: 150, B: 250, A: 160}

	for p := 0; p < spiralPoints; p++ {
		t := float32(p) / float32(spiralPoints)
		angle := t*spiralTurns*2.0*float32(math.Pi) + float32(r.time)*0.5
		radius := float32(12.0) * t
		vx := nodeX + float32(math.Cos(float64(angle)))*radius
		vy := nodeY + float32(math.Sin(float64(angle)))*radius
		vector.DrawFilledCircle(screen, vx, vy, 1.5, oracleColor, false)
	}
}

// drawForgeProjects renders Forge Projects as anvil-and-flame icons.
func (r *Renderer) drawForgeProjects(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	forges, err := r.store.GetActiveForgeEventsNearNode(nodeData.PublicKey, 100.0)
	if err != nil || len(forges) == 0 || len(forges) > 1 {
		return // Max 1 visible forge
	}

	forge := forges[0]
	if forge == nil {
		return
	}

	// Draw anvil-and-flame icon.
	anvilX := nodeX + 15.0
	anvilY := nodeY - 25.0

	// Anvil (rectangle).
	forgeColor := color.RGBA{R: 180, G: 100, B: 50, A: 200}
	vector.DrawFilledRect(screen, anvilX-4, anvilY+2, 8, 4, forgeColor, false)

	// Flame (animated dots).
	for i := 0; i < 3; i++ {
		flameX := anvilX + float32(i-1)*3.0
		flameY := anvilY - float32(math.Sin(float64(r.time)*5.0+float64(i)))*5.0
		flameColor := color.RGBA{R: 255, G: 150 - uint8(i*30), B: 0, A: 180}
		vector.DrawFilledCircle(screen, flameX, flameY, 1.5, flameColor, false)
	}
}

// drawShadowPlays renders Shadow Plays as dark domes with lightning effects.
func (r *Renderer) drawShadowPlays(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	_, ok := getSingleActiveItem(nodeData.PublicKey, r.store.GetActiveShadowPlayNearNode)
	if !ok {
		return
	}

	// Draw dark dome with lightning.
	domeRadius := float32(28.0)
	domeColor := color.RGBA{R: 40, G: 40, B: 80, A: 100}
	vector.StrokeCircle(screen, nodeX, nodeY, domeRadius, 2.0, domeColor, false)

	// Lightning (animated lines).
	if int(r.time*10)%3 == 0 {
		lightningColor := color.RGBA{R: 200, G: 200, B: 255, A: 200}
		lx1 := nodeX - 10.0
		ly1 := nodeY - domeRadius
		lx2 := nodeX + 5.0
		ly2 := nodeY - domeRadius/2
		vector.StrokeLine(screen, lx1, ly1, lx2, ly2, 1.5, lightningColor, false)
	}
}

// drawPhantomCouncils renders Phantom Council membership as constellation patterns.
func (r *Renderer) drawPhantomCouncils(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	if len(nodeData.PublicKey) == 0 {
		return
	}

	councils, err := r.store.GetCouncilsWithMember(nodeData.PublicKey)
	if err != nil || len(councils) == 0 {
		return
	}

	// Draw colored thread pattern for council membership.
	for i, council := range councils {
		if council == nil || i >= 2 { // Max 2 visible councils
			continue
		}

		// Draw constellation pattern (3 connected dots).
		constellationRadius := 20.0 + float32(i)*8.0
		councilColor := color.RGBA{
			R: 100 + uint8(council.Id[0]%100),
			G: 100 + uint8(council.Id[1]%100),
			B: 150,
			A: 150,
		}

		for c := 0; c < 3; c++ {
			angle := float32(c)*2.0*math.Pi/3.0 + float32(r.time)*0.2
			cx := nodeX + float32(math.Cos(float64(angle)))*constellationRadius
			cy := nodeY + float32(math.Sin(float64(angle)))*constellationRadius
			vector.DrawFilledCircle(screen, cx, cy, 2.0, councilColor, false)

			// Connect to next dot.
			nextAngle := float32((c+1)%3)*2.0*math.Pi/3.0 + float32(r.time)*0.2
			nextCx := nodeX + float32(math.Cos(float64(nextAngle)))*constellationRadius
			nextCy := nodeY + float32(math.Sin(float64(nextAngle)))*constellationRadius
			vector.StrokeLine(screen, cx, cy, nextCx, nextCy, 1.0, councilColor, false)
		}
	}
}

// getSingleActiveItem retrieves exactly one active item from a store query.
// Returns (item, true) if exactly one non-nil item is found, otherwise (nil, false).
func getSingleActiveItem[T any](pubkey []byte, fetchFn func([]byte, float64) ([]*T, error)) (*T, bool) {
	if len(pubkey) == 0 {
		return nil, false
	}

	items, err := fetchFn(pubkey, 100.0)
	if err != nil || len(items) == 0 || len(items) > 1 {
		return nil, false
	}

	item := items[0]
	if item == nil {
		return nil, false
	}

	return item, true
}
