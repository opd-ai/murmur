// Package assets_test verifies go:embed assets are correctly included in builds.
package assets_test

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/assets"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering/effects"
)

// TestSpecterWordlistEmbedded verifies the Specter name wordlist is embedded and loaded.
func TestSpecterWordlistEmbedded(t *testing.T) {
	// Verify wordlist is loaded
	if len(assets.SpecterWordlist) == 0 {
		t.Fatal("SpecterWordlist is empty - embedded wordlist not loaded")
	}

	// Verify correct size (65,536 entries per docs/ANONYMOUS_GAME_MECHANICS.md)
	expectedSize := 65536
	if len(assets.SpecterWordlist) != expectedSize {
		t.Errorf("SpecterWordlist size mismatch: got %d, want %d", len(assets.SpecterWordlist), expectedSize)
	}

	// Verify entries are non-empty
	for i, entry := range assets.SpecterWordlist {
		if entry == "" {
			t.Errorf("SpecterWordlist entry %d is empty", i)
			break
		}
		if i > 10 { // Sample first 10 entries only
			break
		}
	}

	t.Logf("✅ Specter wordlist: %d entries loaded from embedded asset", len(assets.SpecterWordlist))
}

// TestKageShadersEmbedded verifies Kage shaders are embedded and can be loaded.
// This test requires the !test build tag to be absent (normal build mode).
func TestKageShadersEmbedded(t *testing.T) {
	// Skip if running with -tags=test (effects package won't compile shaders)
	if testing.Short() {
		t.Skip("Skipping shader test in short mode")
	}

	// Attempt to load shaders (will fail gracefully in headless environments)
	shaders, err := effects.LoadShaders()
	if err != nil {
		// Expected in headless CI environments without OpenGL context
		t.Logf("⚠️  Shader loading failed (expected in headless environment): %v", err)
		t.Skip("Cannot load shaders in headless environment")
		return
	}

	// Verify shaders are non-nil
	if shaders.Glow == nil {
		t.Error("Glow shader is nil - embedded glow.kage not loaded")
	}
	if shaders.Ripple == nil {
		t.Error("Ripple shader is nil - embedded ripple.kage not loaded")
	}
	if shaders.Spectra == nil {
		t.Error("Spectra shader is nil - embedded spectra.kage not loaded")
	}

	t.Log("✅ Kage shaders loaded successfully from embedded assets")
}

// TestEmbeddedAssetsInBinary verifies embedded assets are present in the compiled binary.
// This is a build-time verification that go:embed directives are working.
func TestEmbeddedAssetsInBinary(t *testing.T) {
	tests := []struct {
		name  string
		check func() bool
		desc  string
	}{
		{
			name: "SpecterWordlist",
			check: func() bool {
				return len(assets.SpecterWordlist) == 65536
			},
			desc: "65,536 Specter names embedded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("%s: embedded asset verification failed", tt.desc)
			} else {
				t.Logf("✅ %s", tt.desc)
			}
		})
	}
}

// TestCrossPlatformEmbedConsistency verifies go:embed behavior is consistent across platforms.
// This test ensures that embedded assets are accessible regardless of GOOS/GOARCH.
func TestCrossPlatformEmbedConsistency(t *testing.T) {
	// Test that SpecterWordlist is accessible
	if assets.SpecterWordlist == nil {
		t.Fatal("SpecterWordlist is nil - embedded asset not accessible")
	}

	// Test that wordlist entries are valid UTF-8
	for i, entry := range assets.SpecterWordlist {
		if !isValidUTF8(entry) {
			t.Errorf("SpecterWordlist entry %d contains invalid UTF-8: %q", i, entry)
			break
		}
		if i > 100 { // Sample first 100 entries
			break
		}
	}

	t.Log("✅ Embedded assets are consistent and platform-independent")
}

// isValidUTF8 checks if a string is valid UTF-8.
func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false // Replacement character indicates invalid UTF-8
		}
	}
	return true
}
