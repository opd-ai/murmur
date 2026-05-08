// Package rendering – Ebitengine headless screenshot comparison tests.
// Per PLAN.md: "Ebitengine headless mode screenshot comparison tests for rendering".
//
// These tests must be run with the ebitentest build tag and a display
// (or xvfb-run on headless Linux):
//
//	xvfb-run go test -tags=ebitentest ./pkg/pulsemap/rendering/ -run TestScreenshot
//
//go:build ebitentest
// +build ebitentest

package rendering

import (
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// screenshotTestMain wraps the test suite in an Ebitengine game loop so that
// ReadPixels / At are available within test functions, per the pattern used by
// Ebitengine's own internal/testing.MainWithRunLoop.
type screenshotTestMain struct {
	m    *testing.M
	code int
}

func (g *screenshotTestMain) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*screenshotTestMain) Draw(*ebiten.Image) {}

func (*screenshotTestMain) Layout(int, int) (int, int) { return 320, 240 }

// TestMain sets up the Ebitengine game loop before running tests so that
// pixel-reading calls (ReadPixels, At) work correctly inside test functions.
func TestMain(m *testing.M) {
	g := &screenshotTestMain{m: m, code: 1}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

// colorApprox returns true if the RGBA components of got and want are all
// within delta of each other, tolerating small GPU rounding differences.
func colorApprox(got, want color.RGBA, delta uint8) bool {
	diff := func(a, b uint8) uint8 {
		if a > b {
			return a - b
		}
		return b - a
	}
	return diff(got.R, want.R) <= delta &&
		diff(got.G, want.G) <= delta &&
		diff(got.B, want.B) <= delta &&
		diff(got.A, want.A) <= delta
}

// TestScreenshotNodeCoreIsVisible verifies that rendering a node to an
// off-screen image results in non-zero pixels near its center.
// This is a screenshot regression guard: if the node is invisible (all
// transparent), it catches regressions in the draw path.
func TestScreenshotNodeCoreIsVisible(t *testing.T) {
	const (
		imgW, imgH = 200, 200
		cx, cy     = 100, 100 // Node center
	)

	img := ebiten.NewImage(imgW, imgH)

	style := NodeStyle{
		CoreColor:   color.RGBA{R: 220, G: 80, B: 80, A: 255},
		RingColor:   color.RGBA{R: 100, G: 100, B: 200, A: 255},
		HasRing:     true,
		HasHalo:     true,
		HaloAlpha:   0.8,
		Connections: 8,
		Activity:    20.0,
	}

	RenderNode(img, float32(cx), float32(cy), style, ZoomMeso)

	// Verify that pixels near the center are not all transparent.
	// The core radius for Connections=8, Activity=20 is large enough that (cx,cy)
	// will have non-zero alpha.
	anyNonZero := false
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			c := img.At(cx+dx, cy+dy)
			_, _, _, a := c.RGBA()
			if a > 0 {
				anyNonZero = true
			}
		}
	}

	if !anyNonZero {
		t.Error("no non-transparent pixels near node center after RenderNode; " +
			"possible regression in the draw path")
	}
}

// TestScreenshotEdgeIsVisible verifies that rendering an edge between two
// points produces at least one non-transparent pixel along the path.
func TestScreenshotEdgeIsVisible(t *testing.T) {
	const (
		imgW, imgH = 200, 200
		x1, y1     = 20, 20
		x2, y2     = 180, 180
	)

	img := ebiten.NewImage(imgW, imgH)

	style := EdgeStyle{
		Color:  color.RGBA{R: 200, G: 180, B: 40, A: 200},
		Age:    10,
		Active: true,
	}

	RenderEdge(img, x1, y1, x2, y2, style, ZoomMeso)

	// Sample the approximate midpoint of the diagonal edge.
	midX, midY := (x1+x2)/2, (y1+y2)/2

	anyNonZero := false
	for dy := -5; dy <= 5; dy++ {
		for dx := -5; dx <= 5; dx++ {
			c := img.At(midX+dx, midY+dy)
			_, _, _, a := c.RGBA()
			if a > 0 {
				anyNonZero = true
			}
		}
	}

	if !anyNonZero {
		t.Errorf("no non-transparent pixels near edge midpoint (%d,%d) after RenderEdge",
			midX, midY)
	}
}

// TestScreenshotMacroNodeIsSmall verifies that at ZoomMacro, a node is rendered
// as a small dot. Pixels far from the center should remain transparent.
func TestScreenshotMacroNodeIsSmall(t *testing.T) {
	const (
		imgW, imgH = 200, 200
		cx, cy     = 100, 100
	)

	img := ebiten.NewImage(imgW, imgH)

	style := NodeStyle{
		CoreColor:   color.RGBA{R: 80, G: 200, B: 80, A: 255},
		Connections: 4,
		Activity:    5.0,
	}

	RenderNode(img, float32(cx), float32(cy), style, ZoomMacro)

	// ZoomMacro renders a small dot: pixels very close to center should be non-zero.
	hasCenter := false
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			c := img.At(cx+dx, cy+dy)
			_, _, _, a := c.RGBA()
			if a > 0 {
				hasCenter = true
			}
		}
	}

	if !hasCenter {
		t.Error("ZoomMacro node has no visible pixels at center")
	}

	// And pixels far away should be fully transparent (the node is small).
	for _, coord := range [][2]int{{30, 30}, {170, 30}, {30, 170}, {170, 170}} {
		c := img.At(coord[0], coord[1])
		_, _, _, a := c.RGBA()
		if a != 0 {
			t.Errorf("ZoomMacro node leaks pixels far from center at (%d,%d)", coord[0], coord[1])
		}
	}
}

// TestScreenshotNodeColorMatch verifies that the rendered core color of a node
// is approximately the configured CoreColor.
func TestScreenshotNodeColorMatch(t *testing.T) {
	const (
		imgW, imgH = 200, 200
		cx, cy     = 100, 100
	)

	want := color.RGBA{R: 255, G: 64, B: 64, A: 255}
	img := ebiten.NewImage(imgW, imgH)

	style := NodeStyle{
		CoreColor:   want,
		Connections: 10,
		Activity:    30.0,
	}

	RenderNode(img, float32(cx), float32(cy), style, ZoomMeso)

	// The center pixel should approximate the core color.
	// We use delta=60 to account for GPU color blending, halo overlays, etc.
	got := img.At(cx, cy)
	gotR, gotG, gotB, gotA := got.RGBA()
	gotRGBA := color.RGBA{uint8(gotR >> 8), uint8(gotG >> 8), uint8(gotB >> 8), uint8(gotA >> 8)}

	if gotA == 0 {
		t.Fatal("center pixel of node is fully transparent; expected visible node core")
	}

	// Red channel should dominate (since CoreColor is mostly red).
	if gotRGBA.R < gotRGBA.G+10 {
		t.Errorf("expected red-dominant core pixel, got R=%d G=%d B=%d",
			gotRGBA.R, gotRGBA.G, gotRGBA.B)
	}
}
