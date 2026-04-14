// Package effects provides hunt fragment visualization for the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, active Hunts are shown as "dim, pulsing markers
// scattered across the topology". As fragments are claimed, their markers brighten
// and display the claimer's Specter sigil.
//
// Fragment visual styles:
// - Unclaimed: dim, pulsing amber glow
// - Claimed: bright glow with claimer's sigil overlay
//
// Hunt indicators:
// - Active hunt: faint connecting lines between fragments
// - Hunt expiring: red pulse warning effect
// - Hunt completed: all fragments show victory animation
//
//go:build !noebiten
// +build !noebiten

package effects

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// FragmentState represents the claim state of a hunt fragment.
type FragmentState int

const (
	FragmentUnclaimed FragmentState = iota // Dim pulsing marker.
	FragmentClaimed                        // Bright with claimer sigil.
	FragmentExpired                        // Hunt expired, faded.
)

// HuntState represents the overall state of a hunt.
type HuntState int

const (
	HuntStateActive    HuntState = iota // Normal operation.
	HuntStateExpiring                   // Warning pulse (< 5 min left).
	HuntStateCompleted                  // Victory animation.
	HuntStateExpired                    // All faded.
)

// FragmentVisual represents a hunt fragment to be rendered on the Pulse Map.
type FragmentVisual struct {
	ID           [32]byte      // Fragment identifier.
	HuntID       [32]byte      // Parent hunt.
	Index        int           // Fragment index in hunt.
	X, Y         float32       // Position in screen coordinates.
	State        FragmentState // Claim state.
	ClaimerSigil *ebiten.Image // Sigil to display if claimed.
	ClaimerKey   [32]byte      // Claimer's public key.
	ClueLevel    int           // Number of clues revealed (0-3).
}

// HuntEffects renders hunt fragment visualizations on the Pulse Map.
// Per ANONYMOUS_GAME_MECHANICS.md, fragments appear as scattered pulsing markers.
type HuntEffects struct {
	mu        sync.RWMutex
	time      float32
	fragments map[[32]byte]*FragmentVisual // Fragment ID -> visual.
	hunts     map[[32]byte]HuntState       // Hunt ID -> state.
}

// NewHuntEffects creates a new hunt effects renderer.
func NewHuntEffects() *HuntEffects {
	return &HuntEffects{
		fragments: make(map[[32]byte]*FragmentVisual),
		hunts:     make(map[[32]byte]HuntState),
	}
}

// Update advances animation time.
func (h *HuntEffects) Update(dt float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.time += dt
}

// AddFragment adds a fragment visual to the renderer.
func (h *HuntEffects) AddFragment(frag *FragmentVisual) {
	if frag == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fragments[frag.ID] = frag
}

// RemoveFragment removes a fragment visual.
func (h *HuntEffects) RemoveFragment(id [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.fragments, id)
}

// SetHuntState sets the state for an entire hunt.
func (h *HuntEffects) SetHuntState(huntID [32]byte, state HuntState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hunts[huntID] = state
}

// GetHuntState returns the current state of a hunt.
func (h *HuntEffects) GetHuntState(huntID [32]byte) HuntState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hunts[huntID]
}

// ClaimFragment marks a fragment as claimed.
func (h *HuntEffects) ClaimFragment(fragID, claimerKey [32]byte, sigil *ebiten.Image) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if frag, ok := h.fragments[fragID]; ok {
		frag.State = FragmentClaimed
		frag.ClaimerKey = claimerKey
		frag.ClaimerSigil = sigil
	}
}

// RevealClue increases the clue level for a fragment.
func (h *HuntEffects) RevealClue(fragID [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if frag, ok := h.fragments[fragID]; ok {
		if frag.ClueLevel < 3 {
			frag.ClueLevel++
		}
	}
}

// GetFragment returns a fragment visual by ID.
func (h *HuntEffects) GetFragment(id [32]byte) *FragmentVisual {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.fragments[id]
}

// FragmentCount returns the number of tracked fragments.
func (h *HuntEffects) FragmentCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.fragments)
}

// HuntCount returns the number of tracked hunts.
func (h *HuntEffects) HuntCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.hunts)
}

// ClearHunt removes all fragments for a hunt.
func (h *HuntEffects) ClearHunt(huntID [32]byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, frag := range h.fragments {
		if frag.HuntID == huntID {
			delete(h.fragments, id)
		}
	}
	delete(h.hunts, huntID)
}

// Draw renders all hunt fragments to the destination image.
func (h *HuntEffects) Draw(dst *ebiten.Image, shaders *Shaders) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Group fragments by hunt for connecting lines.
	huntFragments := make(map[[32]byte][]*FragmentVisual)
	for _, frag := range h.fragments {
		huntFragments[frag.HuntID] = append(huntFragments[frag.HuntID], frag)
	}

	// Draw connecting lines for active hunts.
	for huntID, frags := range huntFragments {
		state := h.hunts[huntID]
		if state == HuntStateActive || state == HuntStateExpiring {
			h.drawHuntConnections(dst, frags, state)
		}
	}

	// Draw individual fragments.
	for _, frag := range h.fragments {
		huntState := h.hunts[frag.HuntID]
		h.drawFragment(dst, frag, huntState, shaders)
	}
}

// drawHuntConnections draws faint lines between fragments of a hunt.
func (h *HuntEffects) drawHuntConnections(dst *ebiten.Image, frags []*FragmentVisual, state HuntState) {
	if len(frags) < 2 {
		return
	}

	// Connection line color varies by hunt state.
	var lineColor color.RGBA
	switch state {
	case HuntStateActive:
		// Faint amber connections.
		alpha := uint8(30 + 20*math.Sin(float64(h.time*2)))
		lineColor = color.RGBA{255, 191, 0, alpha}
	case HuntStateExpiring:
		// Pulsing red warning.
		alpha := uint8(50 + 40*math.Sin(float64(h.time*6)))
		lineColor = color.RGBA{255, 80, 80, alpha}
	default:
		return
	}

	// Draw lines connecting adjacent fragments (by index order).
	for i := 0; i < len(frags)-1; i++ {
		for j := i + 1; j < len(frags); j++ {
			if frags[j].Index == frags[i].Index+1 {
				vector.StrokeLine(
					dst,
					frags[i].X, frags[i].Y,
					frags[j].X, frags[j].Y,
					1.0,
					lineColor,
					false,
				)
			}
		}
	}
}

// drawFragment renders a single fragment with appropriate state effects.
func (h *HuntEffects) drawFragment(dst *ebiten.Image, frag *FragmentVisual, huntState HuntState, shaders *Shaders) {
	const baseSize float32 = 24.0

	// Determine visual parameters based on state.
	var (
		glowColor     color.RGBA
		glowIntensity float32
		pulseRate     float32
		drawSigil     bool
	)

	switch frag.State {
	case FragmentUnclaimed:
		// Dim pulsing amber glow.
		glowColor = color.RGBA{255, 191, 0, 150}
		glowIntensity = 0.4 + 0.2*float32(math.Sin(float64(h.time*2+float32(frag.Index))))
		pulseRate = 2.0
		// Increase brightness with each clue revealed.
		glowIntensity += float32(frag.ClueLevel) * 0.1

	case FragmentClaimed:
		// Bright glow with sigil.
		glowColor = color.RGBA{100, 255, 150, 220}
		glowIntensity = 0.8 + 0.1*float32(math.Sin(float64(h.time*3)))
		pulseRate = 3.0
		drawSigil = true

	case FragmentExpired:
		// Faded gray.
		glowColor = color.RGBA{128, 128, 128, 80}
		glowIntensity = 0.2
		pulseRate = 0.5
	}

	// Adjust for hunt state.
	if huntState == HuntStateExpiring && frag.State == FragmentUnclaimed {
		// Add red tint warning.
		glowColor.R = uint8(min(255, int(glowColor.R)+50))
		glowIntensity += 0.2 * float32(math.Sin(float64(h.time*6)))
	} else if huntState == HuntStateCompleted {
		// Victory pulse.
		glowColor = color.RGBA{255, 215, 0, 255} // Gold.
		glowIntensity = 0.9 + 0.1*float32(math.Sin(float64(h.time*4)))
	} else if huntState == HuntStateExpired {
		// All fragments fade.
		glowColor = color.RGBA{80, 80, 80, 60}
		glowIntensity = 0.15
	}

	// Draw glow effect using shader if available.
	if shaders != nil {
		shaders.DrawGlow(dst, frag.X, frag.Y, baseSize*2, GlowUniforms{
			Time:          h.time * pulseRate,
			GlowIntensity: glowIntensity,
			GlowColor:     [4]float32{float32(glowColor.R) / 255, float32(glowColor.G) / 255, float32(glowColor.B) / 255, float32(glowColor.A) / 255},
		})
	}

	// Draw fragment marker (diamond shape).
	h.drawFragmentMarker(dst, frag.X, frag.Y, baseSize, glowColor, glowIntensity)

	// Draw claimer sigil if claimed.
	if drawSigil && frag.ClaimerSigil != nil {
		op := &ebiten.DrawImageOptions{}
		sigilW, sigilH := frag.ClaimerSigil.Bounds().Dx(), frag.ClaimerSigil.Bounds().Dy()
		scale := float64(baseSize*0.6) / float64(max(sigilW, sigilH))
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(
			float64(frag.X)-float64(sigilW)*scale/2,
			float64(frag.Y)-float64(sigilH)*scale/2,
		)
		op.ColorScale.ScaleAlpha(0.9)
		dst.DrawImage(frag.ClaimerSigil, op)
	}

	// Draw clue indicator dots.
	if frag.ClueLevel > 0 && frag.State == FragmentUnclaimed {
		h.drawClueIndicators(dst, frag.X, frag.Y, baseSize, frag.ClueLevel)
	}
}

// drawFragmentMarker renders the diamond-shaped fragment marker.
func (h *HuntEffects) drawFragmentMarker(dst *ebiten.Image, x, y, size float32, c color.RGBA, intensity float32) {
	// Diamond shape: 4 points (top, right, bottom, left).
	halfSize := size / 2

	// Apply intensity to alpha.
	alpha := uint8(float32(c.A) * intensity)
	fillColor := color.RGBA{c.R, c.G, c.B, alpha}

	// Draw filled diamond using path.
	var path vector.Path
	path.MoveTo(x, y-halfSize) // Top.
	path.LineTo(x+halfSize, y) // Right.
	path.LineTo(x, y+halfSize) // Bottom.
	path.LineTo(x-halfSize, y) // Left.
	path.Close()

	// Fill the diamond.
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(fillColor.R) / 255
		vs[i].ColorG = float32(fillColor.G) / 255
		vs[i].ColorB = float32(fillColor.B) / 255
		vs[i].ColorA = float32(fillColor.A) / 255
	}
	dst.DrawTriangles(vs, is, emptyImage(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})

	// Draw outline.
	outlineAlpha := uint8(min(255, int(alpha)+50))
	outlineColor := color.RGBA{c.R, c.G, c.B, outlineAlpha}
	vector.StrokeLine(dst, x, y-halfSize, x+halfSize, y, 1.5, outlineColor, true)
	vector.StrokeLine(dst, x+halfSize, y, x, y+halfSize, 1.5, outlineColor, true)
	vector.StrokeLine(dst, x, y+halfSize, x-halfSize, y, 1.5, outlineColor, true)
	vector.StrokeLine(dst, x-halfSize, y, x, y-halfSize, 1.5, outlineColor, true)
}

// drawClueIndicators renders small dots below the fragment showing clue level.
func (h *HuntEffects) drawClueIndicators(dst *ebiten.Image, x, y, size float32, level int) {
	dotRadius := float32(3.0)
	dotSpacing := float32(8.0)
	startX := x - float32(level-1)*dotSpacing/2
	dotY := y + size/2 + 8

	for i := 0; i < level; i++ {
		dotX := startX + float32(i)*dotSpacing
		// Cyan dots for revealed clues.
		vector.DrawFilledCircle(dst, dotX, dotY, dotRadius, color.RGBA{0, 200, 255, 200}, true)
	}
}

// emptyImage returns a 1x1 white pixel image for DrawTriangles.
var (
	emptyImageOnce  sync.Once
	emptyImageCache *ebiten.Image
)

func emptyImage() *ebiten.Image {
	emptyImageOnce.Do(func() {
		emptyImageCache = ebiten.NewImage(1, 1)
		emptyImageCache.Fill(color.White)
	})
	return emptyImageCache
}
