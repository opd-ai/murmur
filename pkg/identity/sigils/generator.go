// Package sigils provides deterministic visual identity generation from public keys.
// Per DESIGN_DOCUMENT.md Part II, sigils are 64×64 visual icons derived from
// BLAKE3 hashes of public keys, providing unique visual identification.
package sigils

import (
	"image"
	"image/color"

	"github.com/zeebo/blake3"
)

// Size is the pixel dimension of generated sigils (64×64).
const Size = 64

// Sigil represents a deterministically generated visual identity.
type Sigil struct {
	// Hash is the BLAKE3 hash of the source public key.
	Hash [32]byte

	// Image is the generated 64×64 visual representation.
	Image *image.RGBA
}

// Generate creates a deterministic sigil from a public key.
// The same public key will always produce the identical sigil.
// Per TECHNICAL_IMPLEMENTATION.md §1.4, BLAKE3 is used for identity hashing.
func Generate(publicKey []byte) *Sigil {
	hash := blake3.Sum256(publicKey)

	sigil := &Sigil{
		Hash:  hash,
		Image: image.NewRGBA(image.Rect(0, 0, Size, Size)),
	}

	sigil.render()
	return sigil
}

// render generates the visual representation from the hash.
// The algorithm uses hash bytes to determine colors, patterns, and shapes.
func (s *Sigil) render() {
	bgColor, fgColor := s.extractColors()
	s.fillBackground(bgColor)
	s.drawSymmetricPattern(fgColor)
	s.drawBorder(s.extractBorderColor())
}

// extractColors derives background and foreground colors from the hash.
func (s *Sigil) extractColors() (bg, fg color.RGBA) {
	bg = color.RGBA{R: s.Hash[0], G: s.Hash[1], B: s.Hash[2], A: 255}
	fg = color.RGBA{R: s.Hash[3], G: s.Hash[4], B: s.Hash[5], A: 255}
	return bg, fg
}

// extractBorderColor derives the border color from the hash.
func (s *Sigil) extractBorderColor() color.RGBA {
	return color.RGBA{R: s.Hash[6], G: s.Hash[7], B: s.Hash[8], A: 255}
}

// fillBackground fills the image with the background color.
func (s *Sigil) fillBackground(bgColor color.RGBA) {
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			s.Image.Set(x, y, bgColor)
		}
	}
}

// drawSymmetricPattern generates a mirrored 5x5 grid pattern.
func (s *Sigil) drawSymmetricPattern(fgColor color.RGBA) {
	gridSize := 5
	cellSize := Size / gridSize

	for row := 0; row < gridSize; row++ {
		for col := 0; col <= gridSize/2; col++ {
			if s.isCellFilled(row, col, gridSize) {
				s.fillCell(col, row, cellSize, fgColor)
				s.fillCell(gridSize-1-col, row, cellSize, fgColor)
			}
		}
	}
}

// isCellFilled determines if a cell should be filled based on hash.
func (s *Sigil) isCellFilled(row, col, gridSize int) bool {
	idx := (row*gridSize + col) % 32
	return s.Hash[idx]&(1<<uint(col)) != 0
}

// fillCell fills a cell in the grid pattern.
func (s *Sigil) fillCell(col, row, cellSize int, c color.RGBA) {
	startX := col * cellSize
	startY := row * cellSize

	// Add padding for visual separation.
	padding := 1

	for y := startY + padding; y < startY+cellSize-padding; y++ {
		for x := startX + padding; x < startX+cellSize-padding; x++ {
			if x >= 0 && x < Size && y >= 0 && y < Size {
				s.Image.Set(x, y, c)
			}
		}
	}
}

// drawBorder draws a 1-pixel border around the sigil.
func (s *Sigil) drawBorder(c color.RGBA) {
	for i := 0; i < Size; i++ {
		s.Image.Set(i, 0, c)
		s.Image.Set(i, Size-1, c)
		s.Image.Set(0, i, c)
		s.Image.Set(Size-1, i, c)
	}
}

// Bytes returns the hash bytes for the sigil.
func (s *Sigil) Bytes() []byte {
	return s.Hash[:]
}

// Equal returns true if two sigils have the same hash.
func (s *Sigil) Equal(other *Sigil) bool {
	if other == nil {
		return false
	}
	return s.Hash == other.Hash
}

// GenerateSpecter creates a sigil with the Specter visual style.
// Specter sigils use a distinct spectral glow appearance.
// Per DESIGN_DOCUMENT.md, Specter sigils have different shapes and effects.
func GenerateSpecter(publicKey []byte) *Sigil {
	// Prepend a domain separator to ensure Specter sigils
	// are cryptographically distinct from Surface sigils.
	input := append([]byte("specter:"), publicKey...)
	hash := blake3.Sum256(input)

	sigil := &Sigil{
		Hash:  hash,
		Image: image.NewRGBA(image.Rect(0, 0, Size, Size)),
	}

	sigil.renderSpecter()
	return sigil
}

// renderSpecter generates the spectral visual style.
func (s *Sigil) renderSpecter() {
	bgColor, fgColor := s.extractSpecterColors()
	s.fillGradientBackground(bgColor)
	s.drawDiamondPattern(fgColor)
	s.drawBorder(s.extractGlowColor())
}

// extractSpecterColors derives cool-tone spectral colors from the hash.
// Per DESIGN_DOCUMENT.md, Specter sigils use the cool-tone palette (200–280° hue range).
func (s *Sigil) extractSpecterColors() (bg, fg color.RGBA) {
	// Background: cool-tone with low saturation.
	bgHue := hueInRange(s.Hash[0], 200, 280)
	bgSat := float64(s.Hash[1]) / 255.0 * 0.3 // Low saturation (0-30%).
	bgLum := 0.15 + float64(s.Hash[2])/255.0*0.1
	bg = hslToRGB(bgHue, bgSat, bgLum)

	// Foreground: cool-tone with higher saturation and brightness.
	fgHue := hueInRange(s.Hash[3], 200, 280)
	fgSat := 0.5 + float64(s.Hash[4])/255.0*0.4 // Medium-high saturation (50-90%).
	fgLum := 0.5 + float64(s.Hash[5])/255.0*0.3
	fg = hslToRGB(fgHue, fgSat, fgLum)

	return bg, fg
}

// extractGlowColor derives the spectral glow border color from the hash.
// Per DESIGN_DOCUMENT.md, Specter sigils have a cool-tone glow.
func (s *Sigil) extractGlowColor() color.RGBA {
	hue := hueInRange(s.Hash[9], 200, 280)
	sat := 0.7 + float64(s.Hash[10])/255.0*0.3
	lum := 0.6 + float64(s.Hash[11])/255.0*0.2
	return hslToRGB(hue, sat, lum)
}

// hueInRange maps a byte value to a hue within the specified range.
func hueInRange(b byte, minHue, maxHue float64) float64 {
	return minHue + (float64(b)/255.0)*(maxHue-minHue)
}

// hslToRGB converts HSL color values to RGBA.
// h is hue in degrees (0-360), s and l are saturation/lightness (0-1).
func hslToRGB(h, s, l float64) color.RGBA {
	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hueToRGB(p, q, h/360.0+1.0/3.0)
		g = hueToRGB(p, q, h/360.0)
		b = hueToRGB(p, q, h/360.0-1.0/3.0)
	}

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hueToRGB is a helper for HSL to RGB conversion.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// fillGradientBackground fills with a vertical gradient effect.
func (s *Sigil) fillGradientBackground(bgColor color.RGBA) {
	for y := 0; y < Size; y++ {
		gradientFactor := uint8(y * 64 / Size)
		adjusted := color.RGBA{
			R: bgColor.R,
			G: bgColor.G,
			B: uint8(min(255, int(bgColor.B)+int(gradientFactor))),
			A: 255,
		}
		for x := 0; x < Size; x++ {
			s.Image.Set(x, y, adjusted)
		}
	}
}

// drawDiamondPattern draws a spectral diamond pattern.
func (s *Sigil) drawDiamondPattern(fgColor color.RGBA) {
	center := Size / 2
	for row := 0; row < Size; row++ {
		for col := 0; col < Size; col++ {
			if s.shouldFillDiamondPixel(col, row, center) {
				s.Image.Set(col, row, fgColor)
			}
		}
	}
}

// shouldFillDiamondPixel determines if a pixel in the diamond pattern is filled.
func (s *Sigil) shouldFillDiamondPixel(col, row, center int) bool {
	dist := abs(col-center) + abs(row-center)
	hashIdx := (dist / 4) % 32
	return s.Hash[hashIdx]&(1<<uint(row%8)) != 0 && dist < center
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GenerateMaskedEvent creates a blank sigil for Masked Event participants.
// Per DESIGN_DOCUMENT.md §26, Masked Events use identical blank nodes for all participants.
// The eventID provides deterministic but event-specific styling.
func GenerateMaskedEvent(eventID []byte) *Sigil {
	// Derive event-specific hash for consistent styling within the event.
	input := append([]byte("masked-event:"), eventID...)
	hash := blake3.Sum256(input)

	sigil := &Sigil{
		Hash:  hash,
		Image: image.NewRGBA(image.Rect(0, 0, Size, Size)),
	}

	sigil.renderMasked()
	return sigil
}

// renderMasked generates a blank, featureless sigil for anonymity.
// All Masked Event sigils look identical - a simple gradient with no distinguishing patterns.
func (s *Sigil) renderMasked() {
	// Use a neutral gray gradient for complete anonymity.
	for y := 0; y < Size; y++ {
		gray := uint8(40 + y*30/Size)
		c := color.RGBA{R: gray, G: gray, B: gray + 10, A: 255}
		for x := 0; x < Size; x++ {
			s.Image.Set(x, y, c)
		}
	}

	// Draw a subtle circular outline.
	center := Size / 2
	radius := Size/2 - 2
	borderColor := color.RGBA{R: 80, G: 80, B: 90, A: 255}
	s.drawCircle(center, center, radius, borderColor)
}

// drawCircle draws a circular outline at the given center and radius.
func (s *Sigil) drawCircle(cx, cy, radius int, c color.RGBA) {
	for angle := 0; angle < 360; angle++ {
		// Use integer approximation for circle.
		x := cx + (radius*cosApprox(angle))/100
		y := cy + (radius*sinApprox(angle))/100
		if x >= 0 && x < Size && y >= 0 && y < Size {
			s.Image.Set(x, y, c)
		}
	}
}

// cosApprox returns cos(angle) * 100 using integer approximation.
func cosApprox(angleDegrees int) int {
	// Simple lookup table for common angles.
	angle := angleDegrees % 360
	if angle < 0 {
		angle += 360
	}

	// Use symmetry to reduce lookup table size.
	var val int
	switch {
	case angle <= 90:
		val = cosTable[angle]
	case angle <= 180:
		val = -cosTable[180-angle]
	case angle <= 270:
		val = -cosTable[angle-180]
	default:
		val = cosTable[360-angle]
	}
	return val
}

// sinApprox returns sin(angle) * 100 using integer approximation.
func sinApprox(angleDegrees int) int {
	return cosApprox(angleDegrees - 90)
}

// cosTable holds cos(angle) * 100 for angles 0-90.
var cosTable = [91]int{
	100, 100, 100, 100, 99, 99, 99, 98, 98, 97, // 0-9
	97, 96, 95, 95, 94, 93, 92, 91, 90, 89, // 10-19
	88, 87, 86, 84, 83, 82, 80, 79, 77, 76, // 20-29
	74, 72, 71, 69, 67, 65, 63, 61, 59, 57, // 30-39
	55, 53, 51, 49, 47, 44, 42, 40, 37, 35, // 40-49
	33, 30, 28, 26, 23, 21, 19, 16, 14, 11, // 50-59
	9, 6, 4, 2, 0, -2, -5, -7, -10, -12, // 60-69
	-15, -17, -19, -22, -24, -26, -29, -31, -33, -36, // 70-79
	-38, -40, -42, -45, -47, -49, -51, -53, -55, -57, // 80-89
	-59, // 90
}

// GenerateFromSingleUseKey creates a sigil from a single-use key hash.
// Used for one-time event participation or ephemeral identities.
func GenerateFromSingleUseKey(keyHash []byte) *Sigil {
	// Use the key hash directly (already hashed).
	var hash [32]byte
	if len(keyHash) >= 32 {
		copy(hash[:], keyHash[:32])
	} else {
		// If shorter, hash it to get consistent 32 bytes.
		hash = blake3.Sum256(keyHash)
	}

	sigil := &Sigil{
		Hash:  hash,
		Image: image.NewRGBA(image.Rect(0, 0, Size, Size)),
	}

	// Use specter rendering for single-use keys.
	sigil.renderSpecter()
	return sigil
}
