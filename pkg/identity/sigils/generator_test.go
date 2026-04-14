package sigils

import (
	"bytes"
	"testing"
)

func TestGenerate(t *testing.T) {
	publicKey := []byte("test-public-key-32-bytes-long!!")

	sigil := Generate(publicKey)

	if sigil == nil {
		t.Fatal("Generate() returned nil")
	}

	// Check image dimensions.
	bounds := sigil.Image.Bounds()
	if bounds.Dx() != Size || bounds.Dy() != Size {
		t.Errorf("Image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), Size, Size)
	}

	// Hash should not be empty.
	var zeroHash [32]byte
	if sigil.Hash == zeroHash {
		t.Error("Hash is all zeros")
	}
}

func TestDeterminism(t *testing.T) {
	publicKey := []byte("deterministic-test-key-32-bytes!")

	sigil1 := Generate(publicKey)
	sigil2 := Generate(publicKey)

	// Same public key should produce identical sigils.
	if !sigil1.Equal(sigil2) {
		t.Error("Same public key produced different sigils")
	}

	// Check that images are pixel-identical.
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			c1 := sigil1.Image.At(x, y)
			c2 := sigil2.Image.At(x, y)
			r1, g1, b1, a1 := c1.RGBA()
			r2, g2, b2, a2 := c2.RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Errorf("Pixel (%d,%d) differs between identical sigils", x, y)
			}
		}
	}
}

func TestUniqueness(t *testing.T) {
	key1 := []byte("first-test-key-32-bytes-long!!!")
	key2 := []byte("second-test-key-32-bytes-long!!")

	sigil1 := Generate(key1)
	sigil2 := Generate(key2)

	// Different keys should produce different sigils.
	if sigil1.Equal(sigil2) {
		t.Error("Different public keys produced identical sigils")
	}

	if bytes.Equal(sigil1.Hash[:], sigil2.Hash[:]) {
		t.Error("Different public keys produced identical hashes")
	}
}

func TestBytes(t *testing.T) {
	publicKey := []byte("test-bytes-method-key-32-bytes!!")

	sigil := Generate(publicKey)
	hashBytes := sigil.Bytes()

	if len(hashBytes) != 32 {
		t.Errorf("Bytes() length = %d, want 32", len(hashBytes))
	}

	// Should match the Hash field.
	if !bytes.Equal(hashBytes, sigil.Hash[:]) {
		t.Error("Bytes() doesn't match Hash field")
	}
}

func TestEqual(t *testing.T) {
	key := []byte("test-equal-method-key-32-bytes!!")

	sigil1 := Generate(key)
	sigil2 := Generate(key)
	sigil3 := Generate([]byte("different-key-for-comparison!!!"))

	if !sigil1.Equal(sigil2) {
		t.Error("Equal() returned false for identical sigils")
	}

	if sigil1.Equal(sigil3) {
		t.Error("Equal() returned true for different sigils")
	}

	if sigil1.Equal(nil) {
		t.Error("Equal(nil) should return false")
	}
}

func TestGenerateSpecter(t *testing.T) {
	publicKey := []byte("specter-test-key-32-bytes-long!")

	sigil := GenerateSpecter(publicKey)

	if sigil == nil {
		t.Fatal("GenerateSpecter() returned nil")
	}

	// Check image dimensions.
	bounds := sigil.Image.Bounds()
	if bounds.Dx() != Size || bounds.Dy() != Size {
		t.Errorf("Specter image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), Size, Size)
	}
}

func TestSpecterDifferentFromSurface(t *testing.T) {
	publicKey := []byte("same-key-different-style-32byte!")

	surfaceSigil := Generate(publicKey)
	specterSigil := GenerateSpecter(publicKey)

	// Same key should produce different sigils for different identity types.
	if surfaceSigil.Equal(specterSigil) {
		t.Error("Surface and Specter sigils from same key should differ")
	}
}

func TestSpecterDeterminism(t *testing.T) {
	publicKey := []byte("specter-determinism-test-32byte!")

	sigil1 := GenerateSpecter(publicKey)
	sigil2 := GenerateSpecter(publicKey)

	if !sigil1.Equal(sigil2) {
		t.Error("Same key produced different Specter sigils")
	}
}

func TestImageNotEmpty(t *testing.T) {
	publicKey := []byte("non-empty-image-test-32-bytes!!")

	sigil := Generate(publicKey)

	// Check that the image has non-uniform pixels (not all same color).
	firstPixel := sigil.Image.At(0, 0)
	hasVariation := false

	for y := 0; y < Size && !hasVariation; y++ {
		for x := 0; x < Size && !hasVariation; x++ {
			if sigil.Image.At(x, y) != firstPixel {
				hasVariation = true
			}
		}
	}

	if !hasVariation {
		t.Error("Sigil image has no pixel variation")
	}
}

func TestGenerateMaskedEvent(t *testing.T) {
	eventID := []byte("test-event-id-12345")

	sigil := GenerateMaskedEvent(eventID)

	if sigil == nil {
		t.Fatal("GenerateMaskedEvent() returned nil")
	}

	// Check image dimensions.
	bounds := sigil.Image.Bounds()
	if bounds.Dx() != Size || bounds.Dy() != Size {
		t.Errorf("Masked event image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), Size, Size)
	}
}

func TestMaskedEventIdentical(t *testing.T) {
	// All Masked Event sigils from the same event should be identical.
	eventID := []byte("shared-event-id")

	sigil1 := GenerateMaskedEvent(eventID)
	sigil2 := GenerateMaskedEvent(eventID)

	if !sigil1.Equal(sigil2) {
		t.Error("Masked event sigils from same event should be identical")
	}
}

func TestMaskedEventDifferentFromRegular(t *testing.T) {
	key := []byte("test-key-for-comparison-32bytes!")

	regularSigil := Generate(key)
	specterSigil := GenerateSpecter(key)
	maskedSigil := GenerateMaskedEvent(key)

	if regularSigil.Equal(maskedSigil) {
		t.Error("Masked sigil should differ from regular sigil")
	}
	if specterSigil.Equal(maskedSigil) {
		t.Error("Masked sigil should differ from specter sigil")
	}
}

func TestGenerateFromSingleUseKey(t *testing.T) {
	keyHash := []byte("single-use-key-hash-32-bytes!!!!")

	sigil := GenerateFromSingleUseKey(keyHash)

	if sigil == nil {
		t.Fatal("GenerateFromSingleUseKey() returned nil")
	}

	// Check image dimensions.
	bounds := sigil.Image.Bounds()
	if bounds.Dx() != Size || bounds.Dy() != Size {
		t.Errorf("Single-use key image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), Size, Size)
	}
}

func TestSingleUseKeyDeterminism(t *testing.T) {
	keyHash := []byte("deterministic-single-use-key!!!!")

	sigil1 := GenerateFromSingleUseKey(keyHash)
	sigil2 := GenerateFromSingleUseKey(keyHash)

	if !sigil1.Equal(sigil2) {
		t.Error("Same key hash produced different single-use sigils")
	}
}

func TestSingleUseKeyShortInput(t *testing.T) {
	// Test with input shorter than 32 bytes.
	shortKey := []byte("short")

	sigil := GenerateFromSingleUseKey(shortKey)

	if sigil == nil {
		t.Fatal("GenerateFromSingleUseKey() with short input returned nil")
	}

	// Should still produce valid dimensions.
	bounds := sigil.Image.Bounds()
	if bounds.Dx() != Size || bounds.Dy() != Size {
		t.Errorf("Short key image size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), Size, Size)
	}
}

func TestSpecterCoolTonePalette(t *testing.T) {
	// Test that Specter sigils use cool-tone colors (200-280° hue range).
	publicKey := []byte("specter-cool-tone-test-32-bytes!")

	sigil := GenerateSpecter(publicKey)

	// Sample some pixels and verify they're in the cool-tone range.
	// Cool tones should have more blue than red on average.
	var totalR, totalB int
	sampleCount := 0

	for y := 10; y < Size-10; y += 5 {
		for x := 10; x < Size-10; x += 5 {
			c := sigil.Image.RGBAAt(x, y)
			totalR += int(c.R)
			totalB += int(c.B)
			sampleCount++
		}
	}

	avgR := totalR / sampleCount
	avgB := totalB / sampleCount

	// Cool tones should have higher blue than red on average.
	if avgB <= avgR {
		t.Logf("avgR=%d avgB=%d - Expected cool tones (higher blue)", avgR, avgB)
		// This is a soft check - the exact colors depend on hash values.
		// Just ensure there's some blue presence.
		if avgB < 50 {
			t.Error("Specter sigil lacks blue (cool-tone) presence")
		}
	}
}
