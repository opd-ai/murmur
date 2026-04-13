// Package rendering provides stub types for testing without Ebitengine.
// This file is used when building with the noebiten tag.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"image/color"
	"math"
)

// TargetFPS is the target rendering frame rate.
const TargetFPS = 60

// ZoomLevel represents the current detail level for rendering.
type ZoomLevel int

const (
	ZoomMacro ZoomLevel = iota
	ZoomMeso
	ZoomMicro
)

// NodeStyle contains visual properties for a node.
type NodeStyle struct {
	CoreColor   color.RGBA
	RingColor   color.RGBA
	HasRing     bool
	HasHalo     bool
	HaloAlpha   float32
	IsSpecter   bool
	Selected    bool
	Connections int
	Activity    float64
	Resonance   float64
}

// EdgeStyle contains visual properties for an edge.
type EdgeStyle struct {
	Color      color.RGBA
	Age        float64
	Active     bool
	IsMiniGame bool
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

// computeNodeRadius calculates node size based on connections/resonance.
// Stub implementation for noebiten builds.
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

// Ensure math import is used.
var _ = math.Log
