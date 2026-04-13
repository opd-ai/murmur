// Package screens provides tests for name generation utilities.
//
//go:build noebiten
// +build noebiten

package screens

import "testing"

func TestGenerateSpecterName(t *testing.T) {
	tests := []struct {
		name   string
		pubKey []byte
		want   string
	}{
		{
			name:   "first indices",
			pubKey: []byte{0, 0, 1, 2, 3, 4},
			want:   "Silent Beacon",
		},
		{
			name:   "second indices",
			pubKey: []byte{1, 1, 1, 2, 3, 4},
			want:   "Hollow Cipher",
		},
		{
			name:   "wrap around adjectives",
			pubKey: []byte{20, 0, 1, 2, 3, 4}, // 20 % 20 = 0
			want:   "Silent Beacon",
		},
		{
			name:   "wrap around both",
			pubKey: []byte{21, 21, 1, 2, 3, 4}, // 21 % 20 = 1, 21 % 20 = 1
			want:   "Hollow Cipher",
		},
		{
			name:   "max byte values",
			pubKey: []byte{255, 255, 1, 2, 3, 4}, // 255 % 20 = 15, 255 % 20 = 15
			want:   "Fleeting Presence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSpecterName(tt.pubKey)
			if got != tt.want {
				t.Errorf("GenerateSpecterName(%v) = %q, want %q", tt.pubKey, got, tt.want)
			}
		})
	}
}

func TestGenerateSpecterNameShortInput(t *testing.T) {
	tests := []struct {
		name   string
		pubKey []byte
		want   string
	}{
		{
			name:   "nil input",
			pubKey: nil,
			want:   "Unknown Specter",
		},
		{
			name:   "empty input",
			pubKey: []byte{},
			want:   "Unknown Specter",
		},
		{
			name:   "single byte",
			pubKey: []byte{42},
			want:   "Unknown Specter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSpecterName(tt.pubKey)
			if got != tt.want {
				t.Errorf("GenerateSpecterName(%v) = %q, want %q", tt.pubKey, got, tt.want)
			}
		})
	}
}

func TestGenerateSpecterNameDeterminism(t *testing.T) {
	// Same input should always produce the same output
	pubKey := []byte{42, 137, 255, 0, 1, 2, 3, 4, 5, 6}

	name1 := GenerateSpecterName(pubKey)
	name2 := GenerateSpecterName(pubKey)

	if name1 != name2 {
		t.Errorf("GenerateSpecterName is not deterministic: %q != %q", name1, name2)
	}
}

func TestGenerateSpecterNameDistribution(t *testing.T) {
	// Test that different inputs produce different names (statistical check)
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		pubKey := []byte{byte(i), byte(i + 50), 1, 2, 3, 4}
		name := GenerateSpecterName(pubKey)
		seen[name] = true
	}

	// With 20 adjectives × 20 nouns = 400 combinations
	// Testing 100 different inputs should yield many unique names
	if len(seen) < 20 {
		t.Errorf("Expected at least 20 unique names from 100 inputs, got %d", len(seen))
	}
}

func TestSpecterWordLists(t *testing.T) {
	// Verify word lists are properly sized
	if len(specterAdjectives) != 20 {
		t.Errorf("Expected 20 adjectives, got %d", len(specterAdjectives))
	}

	if len(specterNouns) != 20 {
		t.Errorf("Expected 20 nouns, got %d", len(specterNouns))
	}

	// Check all words are non-empty
	for i, adj := range specterAdjectives {
		if adj == "" {
			t.Errorf("Empty adjective at index %d", i)
		}
	}

	for i, noun := range specterNouns {
		if noun == "" {
			t.Errorf("Empty noun at index %d", i)
		}
	}
}
