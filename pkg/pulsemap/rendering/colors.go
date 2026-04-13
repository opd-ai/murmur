// Package rendering provides color utility functions for the Pulse Map.
// This file contains build-tag independent code shared by both Ebitengine
// and stub implementations.
package rendering

import "image/color"

// ColorFromHash derives a node color from a public key hash.
// Per PULSE_MAP.md: hue from byte 0 (0-360°), sat from byte 1 (40-80%), light from byte 2 (40-60%).
func ColorFromHash(hash []byte, isSpecter bool) color.RGBA {
	if len(hash) < 3 {
		return color.RGBA{128, 128, 128, 255} // Gray fallback
	}

	var hue float64
	if isSpecter {
		// Specter nodes: constrain hue to 200-280° (cool tones)
		hue = 200.0 + float64(hash[0])/255.0*80.0
	} else {
		// Surface nodes: full hue range
		hue = float64(hash[0]) / 255.0 * 360.0
	}

	// Saturation: 40-80%
	sat := 0.4 + float64(hash[1])/255.0*0.4
	// Lightness: 40-60%
	light := 0.4 + float64(hash[2])/255.0*0.2

	r, g, b := hslToRGB(hue, sat, light)
	return color.RGBA{r, g, b, 255}
}

// hslToRGB converts HSL to RGB.
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
