package mechanics

import (
	"testing"
)

func TestNewTrophyGlyphGenerator(t *testing.T) {
	gen := NewTrophyGlyphGenerator()
	if gen == nil {
		t.Fatal("NewTrophyGlyphGenerator returned nil")
	}
	if len(gen.palettes) != 3 {
		t.Errorf("expected 3 palettes, got %d", len(gen.palettes))
	}
}

func TestTrophyGlyphGenerator_GenerateGlyph(t *testing.T) {
	gen := NewTrophyGlyphGenerator()

	tests := []struct {
		name     string
		id       TrophyID
		wantErr  bool
		animated bool
	}{
		// Milestone trophies.
		{"FirstShade", TrophyFirstShade, false, false},
		{"WraithRising", TrophyWraithRising, false, false},
		{"PhantomAscendant", TrophyPhantomAscendant, false, false},
		{"Revenant", TrophyRevenant, false, false},
		{"AbyssWalker", TrophyAbyssWalker, false, false},

		// Activity trophies.
		{"FirstGiftSent", TrophyFirstGiftSent, false, false},
		{"TenPuzzlesSolved", TrophyTenPuzzlesSolved, false, false},
		{"FiveHuntsCompleted", TrophyFiveHuntsCompleted, false, false},
		{"ThreeForgesWon", TrophyThreeForgesWon, false, false},
		{"FirstShadowPlay", TrophyFirstShadowPlay, false, false},
		{"FirstTerritoryCtrl", TrophyFirstTerritoryCtrl, false, false},
		{"HundredWaves", TrophyHundredWaves, false, false},

		// Rare trophies (animated).
		{"Cartographer", TrophyCartographer, false, true},
		{"Oracle", TrophyOracle, false, true},
		{"ChainBreaker", TrophyChainBreaker, false, true},
		{"Ghost", TrophyGhost, false, true},
		{"CouncilFounder", TrophyCouncilFounder, false, true},

		// Invalid.
		{"Invalid", TrophyID("nonexistent"), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glyph, err := gen.GenerateGlyph(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if glyph == nil {
				t.Fatal("glyph is nil")
			}
			if glyph.TrophyID != tt.id {
				t.Errorf("TrophyID = %v, want %v", glyph.TrophyID, tt.id)
			}
			if glyph.Animated != tt.animated {
				t.Errorf("Animated = %v, want %v", glyph.Animated, tt.animated)
			}
			if glyph.Image == nil {
				t.Error("Image is nil")
			}
			bounds := glyph.Image.Bounds()
			if bounds.Dx() != GlyphSize || bounds.Dy() != GlyphSize {
				t.Errorf("Image size = %dx%d, want %dx%d",
					bounds.Dx(), bounds.Dy(), GlyphSize, GlyphSize)
			}
		})
	}
}

func TestTrophyGlyphGenerator_GenerateAllGlyphs(t *testing.T) {
	gen := NewTrophyGlyphGenerator()
	glyphs := gen.GenerateAllGlyphs()

	if len(glyphs) != len(allTrophyDefinitions) {
		t.Errorf("generated %d glyphs, expected %d",
			len(glyphs), len(allTrophyDefinitions))
	}

	for id, glyph := range glyphs {
		if glyph.TrophyID != id {
			t.Errorf("glyph ID mismatch: %v != %v", glyph.TrophyID, id)
		}
	}
}

func TestTrophyGlyphGenerator_GetUnlockedGlyphs(t *testing.T) {
	gen := NewTrophyGlyphGenerator()
	var specterKey [32]byte
	store := NewTrophyStore(specterKey)

	// No unlocks - should return empty.
	glyphs := gen.GetUnlockedGlyphs(store)
	if len(glyphs) != 0 {
		t.Errorf("expected 0 glyphs, got %d", len(glyphs))
	}

	// Unlock some trophies.
	store.UnlockTrophy(TrophyFirstShade, 25.0)
	store.UnlockTrophy(TrophyFirstGiftSent, 30.0)
	store.UnlockTrophy(TrophyCartographer, 150.0)

	glyphs = gen.GetUnlockedGlyphs(store)
	if len(glyphs) != 3 {
		t.Errorf("expected 3 glyphs, got %d", len(glyphs))
	}

	// Verify correct trophies.
	if _, ok := glyphs[TrophyFirstShade]; !ok {
		t.Error("missing FirstShade glyph")
	}
	if _, ok := glyphs[TrophyFirstGiftSent]; !ok {
		t.Error("missing FirstGiftSent glyph")
	}
	if _, ok := glyphs[TrophyCartographer]; !ok {
		t.Error("missing Cartographer glyph")
	}
}

func TestGlyphImageContent(t *testing.T) {
	gen := NewTrophyGlyphGenerator()

	// Test that glyphs have non-uniform content (actually rendered).
	trophies := []TrophyID{
		TrophyFirstShade,
		TrophyFirstGiftSent,
		TrophyCartographer,
	}

	for _, id := range trophies {
		t.Run(string(id), func(t *testing.T) {
			glyph, err := gen.GenerateGlyph(id)
			if err != nil {
				t.Fatalf("error generating glyph: %v", err)
			}

			// Check that we have color variation.
			colors := make(map[uint32]bool)
			for y := 0; y < GlyphSize; y++ {
				for x := 0; x < GlyphSize; x++ {
					c := glyph.Image.RGBAAt(x, y)
					key := uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
					colors[key] = true
				}
			}

			if len(colors) < 2 {
				t.Error("glyph appears to be uniform (no color variation)")
			}
		})
	}
}

func TestGlyphCategoryPalettes(t *testing.T) {
	gen := NewTrophyGlyphGenerator()

	// Verify each category has distinct palettes.
	categories := []int{
		TrophyCategoryMilestone,
		TrophyCategoryActivity,
		TrophyCategoryRare,
	}

	for _, cat := range categories {
		palette, ok := gen.palettes[cat]
		if !ok {
			t.Errorf("missing palette for category %d", cat)
			continue
		}
		if len(palette) < 3 {
			t.Errorf("palette for category %d has only %d colors, need 3",
				cat, len(palette))
		}
	}
}

func TestAddClamp(t *testing.T) {
	tests := []struct {
		v, delta uint8
		want     uint8
	}{
		{0, 0, 0},
		{100, 50, 150},
		{200, 100, 255}, // Clamped.
		{255, 1, 255},   // Clamped.
		{255, 255, 255}, // Clamped.
	}

	for _, tt := range tests {
		got := addClamp(tt.v, tt.delta)
		if got != tt.want {
			t.Errorf("addClamp(%d, %d) = %d, want %d",
				tt.v, tt.delta, got, tt.want)
		}
	}
}

func TestAbsInt(t *testing.T) {
	tests := []struct {
		x, want int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-100, 100},
	}

	for _, tt := range tests {
		got := absInt(tt.x)
		if got != tt.want {
			t.Errorf("absInt(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func TestTrigFunctions(t *testing.T) {
	// Test cos and sin at known angles.
	tests := []struct {
		angle int
		cos   int // Expected cos * 100.
		sin   int // Expected sin * 100.
	}{
		{0, 100, 0},
		{90, 0, 100},
		{180, -100, 0},
		{270, 0, -100},
	}

	for _, tt := range tests {
		gotCos := cosTable90(tt.angle)
		gotSin := sinTable90(tt.angle)

		// Allow some tolerance due to table approximation.
		if absInt(gotCos-tt.cos) > 5 {
			t.Errorf("cosTable90(%d) = %d, want ~%d", tt.angle, gotCos, tt.cos)
		}
		if absInt(gotSin-tt.sin) > 5 {
			t.Errorf("sinTable90(%d) = %d, want ~%d", tt.angle, gotSin, tt.sin)
		}
	}
}

func BenchmarkGenerateGlyph(b *testing.B) {
	gen := NewTrophyGlyphGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.GenerateGlyph(TrophyCartographer)
	}
}

func BenchmarkGenerateAllGlyphs(b *testing.B) {
	gen := NewTrophyGlyphGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateAllGlyphs()
	}
}
