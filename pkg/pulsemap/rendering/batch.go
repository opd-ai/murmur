// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file implements batched draw calls for improved rendering performance.
// Per ROADMAP.md line 692, batched rendering groups similar operations together
// to reduce draw call overhead and improve GPU utilization.

//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BatchRenderer accumulates draw commands and executes them in batches.
// This reduces draw call overhead by grouping similar rendering operations.
type BatchRenderer struct {
	// Edge batches grouped by style
	edgeBatches map[edgeStyleKey][]edgeDrawCommand

	// Node batches grouped by basic style
	nodeBatches map[nodeStyleKey][]nodeDrawCommand

	// Particle batches for effects
	particleBatches []particleDrawCommand

	// Trail batches for amplifications
	trailBatches []trailDrawCommand
}

// edgeStyleKey is a compact representation of edge style for batching.
type edgeStyleKey struct {
	r, g, b, a uint8
	thickness  uint8 // Quantized to 16 levels (0-15)
	active     bool
	isSpecter  bool
}

// nodeStyleKey is a compact representation of node style for batching.
type nodeStyleKey struct {
	r, g, b, a uint8
	isSpecter  bool
	hasRing    bool
	hasHalo    bool
}

// edgeDrawCommand holds parameters for a single edge draw.
type edgeDrawCommand struct {
	x1, y1, x2, y2 float32
	thickness      float32
}

// nodeDrawCommand holds parameters for a single node draw.
type nodeDrawCommand struct {
	x, y          float32
	radius        float32
	ringColor     color.RGBA
	ringThickness float32
	haloAlpha     float32
	selected      bool
}

// particleDrawCommand holds parameters for particle effects.
type particleDrawCommand struct {
	x, y   float32
	radius float32
	color  color.RGBA
}

// trailDrawCommand holds parameters for amplification trails.
type trailDrawCommand struct {
	x1, y1, x2, y2 float32
	baseAlpha      float64
	hasComment     bool
	time           float64
}

// NewBatchRenderer creates a new BatchRenderer.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{
		edgeBatches:     make(map[edgeStyleKey][]edgeDrawCommand),
		nodeBatches:     make(map[nodeStyleKey][]nodeDrawCommand),
		particleBatches: make([]particleDrawCommand, 0, 512),
		trailBatches:    make([]trailDrawCommand, 0, 128),
	}
}

// Clear resets all batches for a new frame.
func (b *BatchRenderer) Clear() {
	// Clear edge batches
	for k := range b.edgeBatches {
		delete(b.edgeBatches, k)
	}

	// Clear node batches
	for k := range b.nodeBatches {
		delete(b.nodeBatches, k)
	}

	// Reset slices (keep capacity)
	b.particleBatches = b.particleBatches[:0]
	b.trailBatches = b.trailBatches[:0]
}

// AddEdge adds an edge to the batch queue.
func (b *BatchRenderer) AddEdge(x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel) {
	// Skip edges at macro zoom level if they're too faint
	if zoom == ZoomMacro && style.Age > 30 && !style.Active {
		return
	}

	// Calculate alpha
	var alpha uint8 = 50
	if style.Age > 90 {
		alpha = 80
	} else if style.Age < 7 {
		alpha = 40
	}
	if style.IsSpecter {
		alpha = uint8(float32(alpha) * 0.7)
	}

	// Calculate thickness
	baseThickness := 1.5
	thicknessScale := 1.5
	thickness := baseThickness + thicknessScale*math.Log(1+style.InteractionFrequency)

	// Quantize thickness to reduce batch fragmentation (16 levels)
	thicknessQuantized := uint8(math.Min(thickness*2, 15))

	// Create style key
	key := edgeStyleKey{
		r:         style.Color.R,
		g:         style.Color.G,
		b:         style.Color.B,
		a:         alpha,
		thickness: thicknessQuantized,
		active:    style.Active,
		isSpecter: style.IsSpecter,
	}

	// Add command to batch
	cmd := edgeDrawCommand{
		x1:        x1,
		y1:        y1,
		x2:        x2,
		y2:        y2,
		thickness: float32(thickness),
	}

	b.edgeBatches[key] = append(b.edgeBatches[key], cmd)
}

// AddNode adds a node to the batch queue.
func (b *BatchRenderer) AddNode(x, y, radius float32, style NodeStyle) {
	// Create style key (simplified for batching)
	key := nodeStyleKey{
		r:         style.CoreColor.R,
		g:         style.CoreColor.G,
		b:         style.CoreColor.B,
		a:         style.CoreColor.A,
		isSpecter: style.IsSpecter,
		hasRing:   style.HasRing,
		hasHalo:   style.HasHalo,
	}

	// Add command to batch
	cmd := nodeDrawCommand{
		x:             x,
		y:             y,
		radius:        radius,
		ringColor:     style.RingColor,
		ringThickness: 1.5,
		haloAlpha:     style.HaloAlpha,
		selected:      style.Selected,
	}
	if style.Selected {
		cmd.ringThickness = 3.0
	}

	b.nodeBatches[key] = append(b.nodeBatches[key], cmd)
}

// AddParticle adds a particle to the batch queue.
func (b *BatchRenderer) AddParticle(x, y, radius float32, particleColor color.RGBA) {
	cmd := particleDrawCommand{
		x:      x,
		y:      y,
		radius: radius,
		color:  particleColor,
	}
	b.particleBatches = append(b.particleBatches, cmd)
}

// AddTrail adds an amplification trail to the batch queue.
func (b *BatchRenderer) AddTrail(x1, y1, x2, y2 float32, baseAlpha float64, hasComment bool, time float64) {
	cmd := trailDrawCommand{
		x1:         x1,
		y1:         y1,
		x2:         x2,
		y2:         y2,
		baseAlpha:  baseAlpha,
		hasComment: hasComment,
		time:       time,
	}
	b.trailBatches = append(b.trailBatches, cmd)
}

// Flush executes all batched draw commands.
func (b *BatchRenderer) Flush(dst *ebiten.Image) {
	// Safety check for tests
	if dst == nil {
		return
	}

	// Per PLAN.md Step 7, this function was refactored to reduce cyclomatic
	// complexity from 22 to <10 by extracting draw operations into helpers.
	b.flushEdges(dst)
	b.flushNodes(dst)
	b.flushEffects(dst)
}

// flushEdges draws all batched edges.
func (b *BatchRenderer) flushEdges(dst *ebiten.Image) {
	for key, batch := range b.edgeBatches {
		if len(batch) == 0 {
			continue
		}

		edgeColor := color.RGBA{R: key.r, G: key.g, B: key.b, A: key.a}

		// Draw all edges in this batch
		for _, cmd := range batch {
			vector.StrokeLine(dst, cmd.x1, cmd.y1, cmd.x2, cmd.y2, cmd.thickness, edgeColor, true)
		}

		// Draw activity pulses if active
		if key.active {
			pulseColor := color.RGBA{255, 255, 255, 180}
			if key.isSpecter {
				pulseColor = color.RGBA{200, 220, 255, 140}
			}
			for _, cmd := range batch {
				mx := (cmd.x1 + cmd.x2) / 2
				my := (cmd.y1 + cmd.y2) / 2
				vector.DrawFilledCircle(dst, mx, my, 3, pulseColor, true)
			}
		}
	}
}

// flushNodes draws all batched nodes with halos, rings, and selection highlights.
func (b *BatchRenderer) flushNodes(dst *ebiten.Image) {
	for key, batch := range b.nodeBatches {
		if len(batch) == 0 {
			continue
		}

		coreColor := color.RGBA{R: key.r, G: key.g, B: key.b, A: key.a}

		// Draw halos first (underneath nodes)
		if key.hasHalo {
			for _, cmd := range batch {
				haloAlpha := uint8(float32(80) * cmd.haloAlpha)
				haloColor := color.RGBA{
					R: coreColor.R,
					G: coreColor.G,
					B: coreColor.B,
					A: haloAlpha,
				}
				vector.DrawFilledCircle(dst, cmd.x, cmd.y, cmd.radius*2.0, haloColor, true)
			}
		}

		// Draw node cores
		if key.isSpecter {
			// Specter nodes with translucency
			alpha := uint8(float64(coreColor.A) * SpecterBaseAlpha)
			translucentCore := color.RGBA{R: coreColor.R, G: coreColor.G, B: coreColor.B, A: alpha}
			for _, cmd := range batch {
				vector.DrawFilledCircle(dst, cmd.x, cmd.y, cmd.radius, translucentCore, true)
			}
		} else {
			// Surface nodes opaque
			for _, cmd := range batch {
				vector.DrawFilledCircle(dst, cmd.x, cmd.y, cmd.radius, coreColor, true)
			}
		}

		// Draw rings if present
		if key.hasRing {
			for _, cmd := range batch {
				vector.StrokeCircle(dst, cmd.x, cmd.y, cmd.radius+cmd.ringThickness, cmd.ringThickness, cmd.ringColor, true)
			}
		}

		// Draw selection highlights
		for _, cmd := range batch {
			if cmd.selected {
				selectColor := color.RGBA{255, 255, 255, 128}
				if key.isSpecter {
					selectColor = color.RGBA{200, 220, 255, 100}
				}
				vector.StrokeCircle(dst, cmd.x, cmd.y, cmd.radius+6, 2.0, selectColor, true)
			}
		}
	}
}

// flushEffects draws all batched particles and trails.
func (b *BatchRenderer) flushEffects(dst *ebiten.Image) {
	// Draw all particles in one batch
	for _, cmd := range b.particleBatches {
		vector.DrawFilledCircle(dst, cmd.x, cmd.y, cmd.radius, cmd.color, true)
	}

	// Draw all trails
	for _, cmd := range b.trailBatches {
		drawBatchedTrail(dst, cmd)
	}
}

// drawBatchedTrail renders a single amplification trail.
func drawBatchedTrail(dst *ebiten.Image, cmd trailDrawCommand) {
	if cmd.baseAlpha < 10 {
		return
	}

	trailColor := color.RGBA{R: 100, G: 255, B: 220, A: uint8(cmd.baseAlpha)}

	// Calculate direction and distance
	dx := float64(cmd.x2 - cmd.x1)
	dy := float64(cmd.y2 - cmd.y1)
	distance := math.Sqrt(dx*dx + dy*dy)
	if distance < 1.0 {
		return
	}

	dirX := dx / distance
	dirY := dy / distance

	// Draw dashed line
	dashLength := 8.0
	segmentLength := 12.0
	currentPos := 0.0

	for currentPos < distance {
		dashEnd := math.Min(currentPos+dashLength, distance)
		x1 := cmd.x1 + float32(currentPos*dirX)
		y1 := cmd.y1 + float32(currentPos*dirY)
		x2 := cmd.x1 + float32(dashEnd*dirX)
		y2 := cmd.y1 + float32(dashEnd*dirY)
		vector.StrokeLine(dst, x1, y1, x2, y2, 2.0, trailColor, true)
		currentPos += segmentLength
	}

	// Draw particles
	particleSpeed := 0.5
	particleCount := 3
	for i := 0; i < particleCount; i++ {
		offset := float64(i) / float64(particleCount)
		particlePos := math.Mod((cmd.time*particleSpeed)+offset, 1.0)
		px := cmd.x1 + float32(particlePos*dx)
		py := cmd.y1 + float32(particlePos*dy)
		particleAlpha := uint8(cmd.baseAlpha * 0.9)
		particleColor := color.RGBA{150, 255, 230, particleAlpha}
		vector.DrawFilledCircle(dst, px, py, 2.5, particleColor, true)
	}

	// Draw comment indicator if present
	if cmd.hasComment {
		mx := (cmd.x1 + cmd.x2) / 2
		my := (cmd.y1 + cmd.y2) / 2
		ringPulse := 1.0 + 0.2*math.Sin(cmd.time*3.0)
		ringRadius := 5.0 * ringPulse
		ringAlpha := uint8(cmd.baseAlpha * 0.7)
		ringColor := color.RGBA{255, 255, 150, ringAlpha}
		vector.StrokeCircle(dst, mx, my, float32(ringRadius), 1.5, ringColor, true)
	}
}
