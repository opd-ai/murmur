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
	Color  color.RGBA
	Age    float64
	Active bool
	IsDuel bool
}

// ColorFromHash derives a node color from a public key hash.
func ColorFromHash(hash []byte, isSpecter bool) color.RGBA {
	if len(hash) < 3 {
		return color.RGBA{128, 128, 128, 255}
	}

	var hue float64
	if isSpecter {
		hue = 200.0 + float64(hash[0])/255.0*80.0
	} else {
		hue = float64(hash[0]) / 255.0 * 360.0
	}

	sat := 0.4 + float64(hash[1])/255.0*0.4
	light := 0.4 + float64(hash[2])/255.0*0.2

	r, g, b := hslToRGB(hue, sat, light)
	return color.RGBA{r, g, b, 255}
}

func hslToRGB(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		gray := uint8(l * 255)
		return gray, gray, gray
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	hNorm := h / 360.0
	r = uint8(hueToRGB(p, q, hNorm+1.0/3.0) * 255)
	g = uint8(hueToRGB(p, q, hNorm) * 255)
	b = uint8(hueToRGB(p, q, hNorm-1.0/3.0) * 255)
	return r, g, b
}

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
	if t < 0.5 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
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

// Ensure math import is used.
var _ = math.Log
