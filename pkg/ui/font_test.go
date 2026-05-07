package ui

import "testing"

func TestMeasureRuneAdvance_MultiByteRuneOffsets(t *testing.T) {
	const s = "日本語"

	one := measureRuneAdvance(s, 1)
	two := measureRuneAdvance(s, 2)
	three := measureRuneAdvance(s, 3)
	clamped := measureRuneAdvance(s, 99)

	if one <= 0 {
		t.Fatalf("expected positive advance for first rune, got %d", one)
	}
	if !(one < two && two < three) {
		t.Fatalf("expected increasing rune advances, got one=%d two=%d three=%d", one, two, three)
	}
	if clamped != three {
		t.Fatalf("expected out-of-range rune index to clamp to full width: got %d want %d", clamped, three)
	}
}
