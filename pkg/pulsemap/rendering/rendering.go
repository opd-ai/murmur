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
func RenderNode(dst *ebiten.Image, x, y float32, style NodeStyle, zoom ZoomLevel) {
	// Calculate radius per PULSE_MAP.md formula
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
	radius := float32(rBase + rScale*math.Log(1+metric))

	// Render halo if active (within 60 minutes of activity)
	if style.HasHalo && style.HaloAlpha > 0 {
		haloRadius := radius * 2.0
		haloColor := color.RGBA{
			R: style.CoreColor.R,
			G: style.CoreColor.G,
			B: style.CoreColor.B,
			A: uint8(float32(80) * style.HaloAlpha),
		}
		vector.DrawFilledCircle(dst, x, y, haloRadius, haloColor, true)
	}

	// Render core circle
	vector.DrawFilledCircle(dst, x, y, radius, style.CoreColor, true)

	// Render ring based on mode
	if style.HasRing {
		ringThickness := float32(1.5)
		if style.Selected {
			ringThickness = 3.0
		}
		vector.StrokeCircle(dst, x, y, radius+ringThickness, ringThickness, style.RingColor, true)
	}

	// Selection highlight
	if style.Selected {
		selectColor := color.RGBA{255, 255, 255, 128}
		vector.StrokeCircle(dst, x, y, radius+6, 2.0, selectColor, true)
	}
}

// EdgeStyle contains visual properties for an edge.
type EdgeStyle struct {
	Color  color.RGBA
	Age    float64 // Connection age in days
	Active bool    // Recent wave propagation
	IsDuel bool    // Specter duel connection
}

// RenderEdge draws a connection edge between two nodes.
// Per PULSE_MAP.md, edges are quadratic Bézier curves with age-based styling.
func RenderEdge(dst *ebiten.Image, x1, y1, x2, y2 float32, style EdgeStyle, zoom ZoomLevel) {
	// Calculate edge opacity based on age
	var alpha uint8 = 50 // Base alpha (20-40% as per spec)
	if style.Age > 90 {
		alpha = 80 // Old connections more visible
	} else if style.Age < 7 {
		alpha = 40 // New connections dashed (simplified to lower alpha)
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
		// Draw a bright dot at midpoint to indicate activity
		mx := (x1 + x2) / 2
		my := (y1 + y2) / 2
		vector.DrawFilledCircle(dst, mx, my, 3, pulseColor, true)
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
