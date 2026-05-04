// Package rendering tests for amplification trail visualization.
//
//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestAmplificationTrailData(t *testing.T) {
	trail := AmplificationTrailData{
		AmplifierID:   "amplifier-node-123",
		OriginalID:    "original-author-456",
		AmplifiedAt:   time.Now().Unix(),
		WaveID:        []byte("wave-abc"),
		HasComment:    true,
		RecentSeconds: 30.0,
	}

	if trail.AmplifierID != "amplifier-node-123" {
		t.Errorf("AmplifierID mismatch: got %s", trail.AmplifierID)
	}
	if trail.OriginalID != "original-author-456" {
		t.Errorf("OriginalID mismatch: got %s", trail.OriginalID)
	}
	if !trail.HasComment {
		t.Error("HasComment should be true")
	}
}

func TestRendererAmplificationTrailMethods(t *testing.T) {
	// Create a renderer without layout engine for testing trail methods.
	r := &Renderer{
		nodeData:            make(map[string]*NodeData),
		edges:               make([]EdgeData, 0),
		amplificationTrails: make([]AmplificationTrailData, 0),
		backgroundColor:     color.RGBA{10, 12, 18, 255},
		screenWidth:         800,
		screenHeight:        600,
	}

	// Test AddAmplificationTrail.
	trail1 := AmplificationTrailData{
		AmplifierID:   "amp1",
		OriginalID:    "orig1",
		AmplifiedAt:   time.Now().Unix(),
		WaveID:        []byte("wave1"),
		HasComment:    false,
		RecentSeconds: 10.0,
	}
	r.AddAmplificationTrail(trail1)

	if len(r.amplificationTrails) != 1 {
		t.Fatalf("Expected 1 trail, got %d", len(r.amplificationTrails))
	}
	if r.amplificationTrails[0].AmplifierID != "amp1" {
		t.Errorf("AmplifierID mismatch: got %s", r.amplificationTrails[0].AmplifierID)
	}

	// Test adding multiple trails.
	trail2 := AmplificationTrailData{
		AmplifierID:   "amp2",
		OriginalID:    "orig1",
		AmplifiedAt:   time.Now().Unix(),
		WaveID:        []byte("wave2"),
		HasComment:    true,
		RecentSeconds: 5.0,
	}
	r.AddAmplificationTrail(trail2)

	if len(r.amplificationTrails) != 2 {
		t.Fatalf("Expected 2 trails, got %d", len(r.amplificationTrails))
	}

	// Test SetAmplificationTrails.
	newTrails := []AmplificationTrailData{
		{
			AmplifierID:   "amp3",
			OriginalID:    "orig3",
			AmplifiedAt:   time.Now().Unix(),
			WaveID:        []byte("wave3"),
			HasComment:    false,
			RecentSeconds: 15.0,
		},
	}
	r.SetAmplificationTrails(newTrails)

	if len(r.amplificationTrails) != 1 {
		t.Fatalf("Expected 1 trail after SetAmplificationTrails, got %d", len(r.amplificationTrails))
	}
	if r.amplificationTrails[0].AmplifierID != "amp3" {
		t.Errorf("AmplifierID mismatch after SetAmplificationTrails: got %s", r.amplificationTrails[0].AmplifierID)
	}

	// Test ClearAmplificationTrails.
	r.ClearAmplificationTrails()

	if len(r.amplificationTrails) != 0 {
		t.Fatalf("Expected 0 trails after ClearAmplificationTrails, got %d", len(r.amplificationTrails))
	}
}

func TestRenderAmplificationTrail(t *testing.T) {
	// Create a test image to render to.
	dst := ebiten.NewImage(800, 600)

	trail := AmplificationTrailData{
		AmplifierID:   "amp1",
		OriginalID:    "orig1",
		AmplifiedAt:   time.Now().Unix(),
		WaveID:        []byte("wave1"),
		HasComment:    true,
		RecentSeconds: 20.0,
	}

	// Test rendering doesn't panic.
	// We can't validate visual output in unit tests, but we can ensure no crashes.
	RenderAmplificationTrail(dst, 100, 100, 300, 400, trail, ZoomMicro, 5.0)

	// Test with nearly faded trail (should skip rendering).
	trail.RecentSeconds = 65.0
	RenderAmplificationTrail(dst, 100, 100, 300, 400, trail, ZoomMicro, 10.0)

	// Test with zero distance (nodes at same position).
	RenderAmplificationTrail(dst, 100, 100, 100, 100, trail, ZoomMicro, 15.0)
}

func TestAmplificationTrailFadeCalculation(t *testing.T) {
	tests := []struct {
		name          string
		recentSeconds float64
		expectVisible bool
	}{
		{"Fresh trail (5s ago)", 5.0, true},
		{"Mid-fade trail (30s ago)", 30.0, true},
		{"Nearly faded trail (55s ago)", 55.0, true},
		{"Faded trail (65s ago)", 65.0, false},
		{"Very old trail (120s ago)", 120.0, false},
	}

	dst := ebiten.NewImage(800, 600)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trail := AmplificationTrailData{
				AmplifierID:   "amp1",
				OriginalID:    "orig1",
				AmplifiedAt:   time.Now().Unix(),
				WaveID:        []byte("wave1"),
				HasComment:    false,
				RecentSeconds: tt.recentSeconds,
			}

			// Test rendering doesn't panic.
			RenderAmplificationTrail(dst, 100, 100, 400, 300, trail, ZoomMicro, 0.0)
		})
	}
}
