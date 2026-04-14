// Package mechanics implements anonymous game mechanics for the Anonymous Layer.
// This file implements trophy glyph generation per ANONYMOUS_GAME_MECHANICS.md §8.
package mechanics

import (
	"image"
	"image/color"
)

// GlyphSize is the pixel dimension of generated trophy glyphs (32×32).
const GlyphSize = 32

// TrophyGlyph represents a visual glyph for an unlocked trophy.
type TrophyGlyph struct {
	TrophyID TrophyID
	Image    *image.RGBA
	Animated bool
}

// TrophyGlyphGenerator creates visual glyphs for trophies.
type TrophyGlyphGenerator struct {
	// palettes holds color schemes for each trophy category.
	palettes map[int][]color.RGBA
}

// NewTrophyGlyphGenerator creates a new glyph generator.
func NewTrophyGlyphGenerator() *TrophyGlyphGenerator {
	return &TrophyGlyphGenerator{
		palettes: map[int][]color.RGBA{
			TrophyCategoryMilestone: {
				{R: 80, G: 60, B: 120, A: 255},   // Deep purple background.
				{R: 180, G: 140, B: 220, A: 255}, // Lavender foreground.
				{R: 220, G: 200, B: 255, A: 255}, // Light purple accent.
			},
			TrophyCategoryActivity: {
				{R: 40, G: 80, B: 100, A: 255},   // Teal background.
				{R: 80, G: 180, B: 160, A: 255},  // Cyan foreground.
				{R: 160, G: 240, B: 220, A: 255}, // Mint accent.
			},
			TrophyCategoryRare: {
				{R: 100, G: 60, B: 20, A: 255},   // Bronze/gold background.
				{R: 220, G: 180, B: 80, A: 255},  // Gold foreground.
				{R: 255, G: 240, B: 160, A: 255}, // Bright gold accent.
			},
		},
	}
}

// GenerateGlyph creates a visual glyph for a trophy.
func (g *TrophyGlyphGenerator) GenerateGlyph(id TrophyID) (*TrophyGlyph, error) {
	def, err := GetTrophyDefinition(id)
	if err != nil {
		return nil, err
	}

	glyph := &TrophyGlyph{
		TrophyID: id,
		Image:    image.NewRGBA(image.Rect(0, 0, GlyphSize, GlyphSize)),
		Animated: def.Animated,
	}

	palette := g.palettes[def.Category]
	if len(palette) < 3 {
		palette = g.palettes[TrophyCategoryMilestone] // Default fallback.
	}

	g.renderGlyph(glyph, def, palette)
	return glyph, nil
}

// renderGlyph generates the visual representation based on trophy type.
func (g *TrophyGlyphGenerator) renderGlyph(glyph *TrophyGlyph, def *TrophyDefinition, palette []color.RGBA) {
	bg, fg, accent := palette[0], palette[1], palette[2]

	g.fillBackground(glyph.Image, bg)

	switch def.Category {
	case TrophyCategoryMilestone:
		g.drawMilestoneSymbol(glyph.Image, def, fg, accent)
	case TrophyCategoryActivity:
		g.drawActivitySymbol(glyph.Image, def, fg, accent)
	case TrophyCategoryRare:
		g.drawRareSymbol(glyph.Image, def, fg, accent)
	}

	g.drawBorder(glyph.Image, accent)
}

// fillBackground fills the glyph with a gradient background.
func (g *TrophyGlyphGenerator) fillBackground(img *image.RGBA, bg color.RGBA) {
	for y := 0; y < GlyphSize; y++ {
		// Subtle vertical gradient.
		factor := uint8(y * 20 / GlyphSize)
		adjusted := color.RGBA{
			R: addClamp(bg.R, factor),
			G: addClamp(bg.G, factor),
			B: addClamp(bg.B, factor),
			A: 255,
		}
		for x := 0; x < GlyphSize; x++ {
			img.Set(x, y, adjusted)
		}
	}
}

// drawBorder draws a 1-pixel border with corner accents.
func (g *TrophyGlyphGenerator) drawBorder(img *image.RGBA, accent color.RGBA) {
	for i := 0; i < GlyphSize; i++ {
		img.Set(i, 0, accent)
		img.Set(i, GlyphSize-1, accent)
		img.Set(0, i, accent)
		img.Set(GlyphSize-1, i, accent)
	}
}

// drawMilestoneSymbol draws a star/level indicator for milestone trophies.
func (g *TrophyGlyphGenerator) drawMilestoneSymbol(img *image.RGBA, def *TrophyDefinition, fg, accent color.RGBA) {
	center := GlyphSize / 2

	// Draw concentric rings based on milestone threshold.
	rings := int(def.Threshold / 50)
	if rings < 1 {
		rings = 1
	}
	if rings > 5 {
		rings = 5
	}

	for r := rings; r >= 1; r-- {
		radius := r * 3
		g.drawRing(img, center, center, radius, fg)
	}

	// Draw center dot.
	g.drawFilledCircle(img, center, center, 2, accent)
}

// drawActivitySymbol draws an action-based symbol for activity trophies.
func (g *TrophyGlyphGenerator) drawActivitySymbol(img *image.RGBA, def *TrophyDefinition, fg, accent color.RGBA) {
	center := GlyphSize / 2

	// Draw based on trophy ID pattern.
	switch def.ID {
	case TrophyFirstGiftSent:
		g.drawGiftSymbol(img, center, fg, accent)
	case TrophyTenPuzzlesSolved:
		g.drawPuzzleSymbol(img, center, fg, accent)
	case TrophyFiveHuntsCompleted:
		g.drawHuntSymbol(img, center, fg, accent)
	case TrophyThreeForgesWon:
		g.drawForgeSymbol(img, center, fg, accent)
	case TrophyFirstShadowPlay:
		g.drawMaskSymbol(img, center, fg, accent)
	case TrophyFirstTerritoryCtrl:
		g.drawFlagSymbol(img, center, fg, accent)
	case TrophyHundredWaves:
		g.drawWaveSymbol(img, center, fg, accent)
	default:
		g.drawDiamondSymbol(img, center, fg, accent)
	}
}

// drawRareSymbol draws an ornate symbol for rare trophies.
func (g *TrophyGlyphGenerator) drawRareSymbol(img *image.RGBA, def *TrophyDefinition, fg, accent color.RGBA) {
	center := GlyphSize / 2

	// All rare trophies get a starburst background.
	g.drawStarburst(img, center, fg)

	// Draw specific symbol in center.
	switch def.ID {
	case TrophyCartographer:
		g.drawCompassSymbol(img, center, accent)
	case TrophyOracle:
		g.drawEyeSymbol(img, center, accent)
	case TrophyChainBreaker:
		g.drawChainSymbol(img, center, accent)
	case TrophyGhost:
		g.drawGhostSymbol(img, center, accent)
	case TrophyCouncilFounder:
		g.drawCrownSymbol(img, center, accent)
	default:
		g.drawFilledCircle(img, center, center, 4, accent)
	}
}

// drawRing draws a circular ring outline.
func (g *TrophyGlyphGenerator) drawRing(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for angle := 0; angle < 360; angle += 5 {
		x := cx + (radius*cosTable90(angle))/100
		y := cy + (radius*sinTable90(angle))/100
		if x >= 0 && x < GlyphSize && y >= 0 && y < GlyphSize {
			img.Set(x, y, c)
		}
	}
}

// drawFilledCircle draws a filled circle.
func (g *TrophyGlyphGenerator) drawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < GlyphSize && y >= 0 && y < GlyphSize {
					img.Set(x, y, c)
				}
			}
		}
	}
}

// drawStarburst draws radiating lines from center.
func (g *TrophyGlyphGenerator) drawStarburst(img *image.RGBA, center int, c color.RGBA) {
	for angle := 0; angle < 360; angle += 30 {
		for r := 4; r < GlyphSize/2-2; r++ {
			x := center + (r*cosTable90(angle))/100
			y := center + (r*sinTable90(angle))/100
			if x >= 0 && x < GlyphSize && y >= 0 && y < GlyphSize {
				img.Set(x, y, c)
			}
		}
	}
}

// Symbol drawing functions for specific trophies.

func (g *TrophyGlyphGenerator) drawGiftSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Draw a simple box with ribbon.
	for x := center - 4; x <= center+4; x++ {
		for y := center - 3; y <= center+4; y++ {
			if y == center-3 || y == center+4 || x == center-4 || x == center+4 {
				img.Set(x, y, fg)
			}
		}
	}
	// Vertical ribbon.
	for y := center - 3; y <= center+4; y++ {
		img.Set(center, y, accent)
	}
	// Horizontal ribbon.
	for x := center - 4; x <= center+4; x++ {
		img.Set(x, center, accent)
	}
}

func (g *TrophyGlyphGenerator) drawPuzzleSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Draw puzzle piece shape.
	for x := center - 4; x <= center+4; x++ {
		img.Set(x, center-2, fg)
		img.Set(x, center+2, fg)
	}
	for y := center - 2; y <= center+2; y++ {
		img.Set(center-4, y, fg)
		img.Set(center+4, y, fg)
	}
	// Puzzle notch.
	g.drawFilledCircle(img, center, center, 2, accent)
}

func (g *TrophyGlyphGenerator) drawHuntSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Draw crosshairs.
	for i := -6; i <= 6; i++ {
		if absInt(i) > 2 {
			img.Set(center+i, center, fg)
			img.Set(center, center+i, fg)
		}
	}
	g.drawRing(img, center, center, 5, accent)
}

func (g *TrophyGlyphGenerator) drawForgeSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Draw anvil shape.
	for x := center - 5; x <= center+5; x++ {
		img.Set(x, center+3, fg)
		img.Set(x, center+4, fg)
	}
	for x := center - 3; x <= center+3; x++ {
		for y := center - 2; y <= center+2; y++ {
			img.Set(x, y, fg)
		}
	}
	// Spark.
	img.Set(center-2, center-4, accent)
	img.Set(center+2, center-4, accent)
}

func (g *TrophyGlyphGenerator) drawMaskSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Draw theatrical mask outline.
	g.drawFilledCircle(img, center, center, 6, fg)
	// Eye holes.
	img.Set(center-2, center-1, accent)
	img.Set(center+2, center-1, accent)
	// Smile.
	for x := center - 2; x <= center+2; x++ {
		img.Set(x, center+2, accent)
	}
}

func (g *TrophyGlyphGenerator) drawFlagSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Flag pole.
	for y := center - 5; y <= center+5; y++ {
		img.Set(center-4, y, fg)
	}
	// Flag.
	for x := center - 3; x <= center+3; x++ {
		for y := center - 5; y <= center-1; y++ {
			img.Set(x, y, accent)
		}
	}
}

func (g *TrophyGlyphGenerator) drawWaveSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Sine wave pattern.
	for x := center - 6; x <= center+6; x++ {
		offset := (x - center + 6) % 4
		y1 := center - 2 + offset
		y2 := center + 2 - offset
		if y1 >= 0 && y1 < GlyphSize {
			img.Set(x, y1, fg)
		}
		if y2 >= 0 && y2 < GlyphSize {
			img.Set(x, y2, accent)
		}
	}
}

func (g *TrophyGlyphGenerator) drawDiamondSymbol(img *image.RGBA, center int, fg, accent color.RGBA) {
	// Diamond shape.
	for i := 0; i <= 5; i++ {
		img.Set(center-i, center-5+i, fg)
		img.Set(center+i, center-5+i, fg)
		img.Set(center-i, center+5-i, fg)
		img.Set(center+i, center+5-i, fg)
	}
	g.drawFilledCircle(img, center, center, 2, accent)
}

func (g *TrophyGlyphGenerator) drawCompassSymbol(img *image.RGBA, center int, accent color.RGBA) {
	// Compass rose.
	for i := -5; i <= 5; i++ {
		img.Set(center+i, center, accent)
		img.Set(center, center+i, accent)
	}
	// Diagonal points.
	for i := 1; i <= 3; i++ {
		img.Set(center+i, center-i, accent)
		img.Set(center-i, center-i, accent)
		img.Set(center+i, center+i, accent)
		img.Set(center-i, center+i, accent)
	}
}

func (g *TrophyGlyphGenerator) drawEyeSymbol(img *image.RGBA, center int, accent color.RGBA) {
	// Eye outline.
	for x := center - 5; x <= center+5; x++ {
		dist := absInt(x - center)
		y1 := center - (5-dist)/2
		y2 := center + (5-dist)/2
		img.Set(x, y1, accent)
		img.Set(x, y2, accent)
	}
	// Pupil.
	g.drawFilledCircle(img, center, center, 2, accent)
}

func (g *TrophyGlyphGenerator) drawChainSymbol(img *image.RGBA, center int, accent color.RGBA) {
	// Chain links.
	g.drawRing(img, center-4, center, 3, accent)
	g.drawRing(img, center, center, 3, accent)
	g.drawRing(img, center+4, center, 3, accent)
}

func (g *TrophyGlyphGenerator) drawGhostSymbol(img *image.RGBA, center int, accent color.RGBA) {
	// Ghost shape.
	g.drawFilledCircle(img, center, center-2, 4, accent)
	for y := center - 2; y <= center+4; y++ {
		img.Set(center-4, y, accent)
		img.Set(center+4, y, accent)
	}
	// Wavy bottom.
	for x := center - 4; x <= center+4; x++ {
		offset := (x - center + 4) % 2
		img.Set(x, center+4+offset, accent)
	}
}

func (g *TrophyGlyphGenerator) drawCrownSymbol(img *image.RGBA, center int, accent color.RGBA) {
	// Crown base.
	for x := center - 5; x <= center+5; x++ {
		img.Set(x, center+2, accent)
		img.Set(x, center+3, accent)
	}
	// Crown points.
	for y := center - 3; y <= center+2; y++ {
		img.Set(center-5, y, accent)
		img.Set(center, y, accent)
		img.Set(center+5, y, accent)
	}
	// Middle peaks.
	img.Set(center-3, center-1, accent)
	img.Set(center+3, center-1, accent)
}

// Helper functions.

func addClamp(v, delta uint8) uint8 {
	sum := int(v) + int(delta)
	if sum > 255 {
		return 255
	}
	return uint8(sum)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// cosTable90 returns cos(angle) * 100 using precomputed values.
func cosTable90(angleDegrees int) int {
	angle := angleDegrees % 360
	if angle < 0 {
		angle += 360
	}

	switch {
	case angle <= 90:
		return glyphCosTable[angle]
	case angle <= 180:
		return -glyphCosTable[180-angle]
	case angle <= 270:
		return -glyphCosTable[angle-180]
	default:
		return glyphCosTable[360-angle]
	}
}

// sinTable90 returns sin(angle) * 100.
func sinTable90(angleDegrees int) int {
	return cosTable90(angleDegrees - 90)
}

// glyphCosTable holds cos(angle) * 100 for angles 0-90.
// Values computed as: round(cos(angle * π / 180) * 100)
var glyphCosTable = [91]int{
	100, 100, 100, 100, 100, 100, 99, 99, 99, 99, // 0-9°
	98, 98, 98, 97, 97, 97, 96, 96, 95, 95, // 10-19°
	94, 93, 93, 92, 91, 91, 90, 89, 88, 87, // 20-29°
	87, 86, 85, 84, 83, 82, 81, 80, 79, 78, // 30-39°
	77, 75, 74, 73, 72, 71, 69, 68, 67, 66, // 40-49°
	64, 63, 62, 60, 59, 57, 56, 54, 53, 52, // 50-59°
	50, 48, 47, 45, 44, 42, 41, 39, 37, 36, // 60-69°
	34, 33, 31, 29, 28, 26, 24, 22, 21, 19, // 70-79°
	17, 16, 14, 12, 10, 9, 7, 5, 3, 2, // 80-89°
	0, // 90°
}

// GenerateAllGlyphs generates glyphs for all defined trophies.
func (g *TrophyGlyphGenerator) GenerateAllGlyphs() map[TrophyID]*TrophyGlyph {
	glyphs := make(map[TrophyID]*TrophyGlyph)
	for id := range allTrophyDefinitions {
		glyph, err := g.GenerateGlyph(id)
		if err == nil {
			glyphs[id] = glyph
		}
	}
	return glyphs
}

// GetUnlockedGlyphs returns glyphs only for unlocked trophies.
func (g *TrophyGlyphGenerator) GetUnlockedGlyphs(store *TrophyStore) map[TrophyID]*TrophyGlyph {
	glyphs := make(map[TrophyID]*TrophyGlyph)
	for _, unlock := range store.AllUnlocks() {
		glyph, err := g.GenerateGlyph(unlock.TrophyID)
		if err == nil {
			glyphs[unlock.TrophyID] = glyph
		}
	}
	return glyphs
}
