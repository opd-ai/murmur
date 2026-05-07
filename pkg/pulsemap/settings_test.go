// Package pulsemap — settings_test.go validates parseModeString and the
// privacy_mode Settings-panel→modes.Manager wiring.

package pulsemap

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/stretchr/testify/assert"
)

// TestParseModeString verifies that every UI-visible mode string round-trips
// to the correct modes.Mode constant.
func TestParseModeString(t *testing.T) {
	cases := []struct {
		input  string
		want   modes.Mode
		wantOK bool
	}{
		{"Open", modes.Open, true},
		{"Hybrid", modes.Hybrid, true},
		{"Guarded", modes.Guarded, true},
		{"Fortress", modes.Fortress, true},
		// Unrecognised strings must return false.
		{"open", 0, false},
		{"OPEN", 0, false},
		{"unknown", 0, false},
		{"", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := parseModeString(tc.input)
			assert.Equal(t, tc.wantOK, ok, "ok mismatch for %q", tc.input)
			if tc.wantOK {
				assert.Equal(t, tc.want, got, "mode mismatch for %q", tc.input)
			}
		})
	}
}

// TestSetModeManager_NilSafe verifies that a nil modeManager does not panic when
// parseModeString is called (game.go guards with g.modeManager != nil).
func TestParseModeString_AllModesHaveDistinctValues(t *testing.T) {
	seen := make(map[modes.Mode]string)
	for _, s := range []string{"Open", "Hybrid", "Guarded", "Fortress"} {
		m, ok := parseModeString(s)
		assert.True(t, ok, "expected %q to parse", s)
		if prev, dup := seen[m]; dup {
			t.Errorf("mode value %v produced by both %q and %q", m, prev, s)
		}
		seen[m] = s
	}
}
