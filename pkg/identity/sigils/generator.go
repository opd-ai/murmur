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

// extractSpecterColors derives spectral colors from the hash.
func (s *Sigil) extractSpecterColors() (bg, fg color.RGBA) {
	bg = color.RGBA{
		R: s.Hash[0] / 4,
		G: s.Hash[1] / 4,
		B: s.Hash[2]/2 + 64,
		A: 255,
	}
	fg = color.RGBA{
		R: s.Hash[3]/2 + 64,
		G: s.Hash[4]/2 + 64,
		B: s.Hash[5]/2 + 128,
		A: 200,
	}
	return bg, fg
}

// extractGlowColor derives the spectral glow border color from the hash.
func (s *Sigil) extractGlowColor() color.RGBA {
	return color.RGBA{
		R: s.Hash[9] / 2,
		G: s.Hash[10] / 2,
		B: s.Hash[11]/2 + 128,
		A: 255,
	}
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
