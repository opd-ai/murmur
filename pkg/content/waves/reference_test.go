package waves

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestParseWaveReferences(t *testing.T) {
	// Create a valid 32-byte Wave ID.
	waveID := make([]byte, 32)
	for i := range waveID {
		waveID[i] = byte(i)
	}
	hexID := hex.EncodeToString(waveID)

	content := []byte("Check out this wave wave://" + hexID + " it's great!")

	refs := ParseWaveReferences(content)
	if len(refs) != 1 {
		t.Fatalf("Expected 1 wave reference, got %d", len(refs))
	}

	ref := refs[0]
	if ref.Type != RefTypeWave {
		t.Errorf("Type = %s, want %s", ref.Type, RefTypeWave)
	}
	if !bytes.Equal(ref.ID, waveID) {
		t.Error("Wave ID mismatch")
	}
	if ref.HexEncoded != hexID {
		t.Errorf("HexEncoded = %s, want %s", ref.HexEncoded, hexID)
	}
	if ref.Start != 20 {
		t.Errorf("Start = %d, want 20", ref.Start)
	}
}

func TestParseMultipleWaveReferences(t *testing.T) {
	waveID1 := bytes.Repeat([]byte{0xaa}, 32)
	waveID2 := bytes.Repeat([]byte{0xbb}, 32)

	content := []byte("First: wave://" + hex.EncodeToString(waveID1) +
		" Second: wave://" + hex.EncodeToString(waveID2))

	refs := ParseWaveReferences(content)
	if len(refs) != 2 {
		t.Fatalf("Expected 2 wave references, got %d", len(refs))
	}

	if !bytes.Equal(refs[0].ID, waveID1) {
		t.Error("First Wave ID mismatch")
	}
	if !bytes.Equal(refs[1].ID, waveID2) {
		t.Error("Second Wave ID mismatch")
	}
}

func TestParseMentions(t *testing.T) {
	// Create an 8-byte user hash.
	userHash := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	hexHash := hex.EncodeToString(userHash)

	content := []byte("Hey @" + hexHash + " check this out!")

	refs := ParseMentions(content)
	if len(refs) != 1 {
		t.Fatalf("Expected 1 mention, got %d", len(refs))
	}

	ref := refs[0]
	if ref.Type != RefTypeMention {
		t.Errorf("Type = %s, want %s", ref.Type, RefTypeMention)
	}
	if !bytes.Equal(ref.ID, userHash) {
		t.Error("User hash mismatch")
	}
	if ref.HexEncoded != hexHash {
		t.Errorf("HexEncoded = %s, want %s", ref.HexEncoded, hexHash)
	}
}

func TestParseMultipleMentions(t *testing.T) {
	hash1 := bytes.Repeat([]byte{0x11}, 8)
	hash2 := bytes.Repeat([]byte{0x22}, 8)

	content := []byte("@" + hex.EncodeToString(hash1) + " and @" + hex.EncodeToString(hash2))

	refs := ParseMentions(content)
	if len(refs) != 2 {
		t.Fatalf("Expected 2 mentions, got %d", len(refs))
	}
}

func TestParseReferencesAll(t *testing.T) {
	waveID := bytes.Repeat([]byte{0xcc}, 32)
	userHash := bytes.Repeat([]byte{0xdd}, 8)

	content := []byte("wave://" + hex.EncodeToString(waveID) + " @" + hex.EncodeToString(userHash))

	refs := ParseReferences(content)
	if len(refs) != 2 {
		t.Fatalf("Expected 2 references, got %d", len(refs))
	}

	// Should have one of each type.
	waveCount := 0
	mentionCount := 0
	for _, ref := range refs {
		if ref.Type == RefTypeWave {
			waveCount++
		} else if ref.Type == RefTypeMention {
			mentionCount++
		}
	}
	if waveCount != 1 || mentionCount != 1 {
		t.Errorf("Counts: wave=%d, mention=%d, want 1 each", waveCount, mentionCount)
	}
}

func TestFormatWaveRef(t *testing.T) {
	waveID := bytes.Repeat([]byte{0xee}, 32)
	expected := "wave://" + hex.EncodeToString(waveID)

	result := FormatWaveRef(waveID)
	if result != expected {
		t.Errorf("FormatWaveRef() = %s, want %s", result, expected)
	}
}

func TestFormatWaveRefInvalidLength(t *testing.T) {
	shortID := []byte{0x01, 0x02, 0x03}
	result := FormatWaveRef(shortID)
	if result != "" {
		t.Error("FormatWaveRef() should return empty for invalid length")
	}
}

func TestFormatMention(t *testing.T) {
	userHash := bytes.Repeat([]byte{0xff}, 8)
	expected := "@" + hex.EncodeToString(userHash)

	result := FormatMention(userHash)
	if result != expected {
		t.Errorf("FormatMention() = %s, want %s", result, expected)
	}
}

func TestFormatMentionInvalidLength(t *testing.T) {
	shortHash := []byte{0x01, 0x02}
	result := FormatMention(shortHash)
	if result != "" {
		t.Error("FormatMention() should return empty for invalid length")
	}
}

func TestHasReferences(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x11}, 32)
	withRefs := []byte("wave://" + hex.EncodeToString(waveID))
	withoutRefs := []byte("Hello, no references here!")

	if !HasReferences(withRefs) {
		t.Error("HasReferences() = false for content with wave ref")
	}
	if HasReferences(withoutRefs) {
		t.Error("HasReferences() = true for content without refs")
	}
}

func TestHasWaveReferences(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x22}, 32)
	userHash := bytes.Repeat([]byte{0x33}, 8)

	withWaveRef := []byte("wave://" + hex.EncodeToString(waveID))
	withMention := []byte("@" + hex.EncodeToString(userHash))

	if !HasWaveReferences(withWaveRef) {
		t.Error("HasWaveReferences() = false for wave ref")
	}
	if HasWaveReferences(withMention) {
		t.Error("HasWaveReferences() = true for mention only")
	}
}

func TestHasMentions(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x44}, 32)
	userHash := bytes.Repeat([]byte{0x55}, 8)

	withWaveRef := []byte("wave://" + hex.EncodeToString(waveID))
	withMention := []byte("@" + hex.EncodeToString(userHash))

	if HasMentions(withWaveRef) {
		t.Error("HasMentions() = true for wave ref only")
	}
	if !HasMentions(withMention) {
		t.Error("HasMentions() = false for mention")
	}
}

func TestReplaceWaveRefs(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x66}, 32)
	content := []byte("Check wave://" + hex.EncodeToString(waveID) + " out")

	result := ReplaceWaveRefs(content, func(id []byte) string {
		return "[WAVE]"
	})

	expected := "Check [WAVE] out"
	if string(result) != expected {
		t.Errorf("ReplaceWaveRefs() = %s, want %s", result, expected)
	}
}

func TestReplaceMentions(t *testing.T) {
	userHash := bytes.Repeat([]byte{0x77}, 8)
	content := []byte("Hello @" + hex.EncodeToString(userHash) + "!")

	result := ReplaceMentions(content, func(hash []byte) string {
		return "@User"
	})

	expected := "Hello @User!"
	if string(result) != expected {
		t.Errorf("ReplaceMentions() = %s, want %s", result, expected)
	}
}

func TestExtractAllRefs(t *testing.T) {
	waveID1 := bytes.Repeat([]byte{0x88}, 32)
	waveID2 := bytes.Repeat([]byte{0x99}, 32)
	userHash := bytes.Repeat([]byte{0xaa}, 8)

	content := []byte("wave://" + hex.EncodeToString(waveID1) +
		" wave://" + hex.EncodeToString(waveID2) +
		" @" + hex.EncodeToString(userHash))

	refs := ExtractAllRefs(content)

	if len(refs.WaveIDs) != 2 {
		t.Errorf("WaveIDs count = %d, want 2", len(refs.WaveIDs))
	}
	if len(refs.UserHashes) != 1 {
		t.Errorf("UserHashes count = %d, want 1", len(refs.UserHashes))
	}
}

func TestExtractAllRefsDuplicates(t *testing.T) {
	waveID := bytes.Repeat([]byte{0xbb}, 32)

	// Same Wave ID referenced twice.
	content := []byte("wave://" + hex.EncodeToString(waveID) +
		" wave://" + hex.EncodeToString(waveID))

	refs := ExtractAllRefs(content)

	// Should deduplicate.
	if len(refs.WaveIDs) != 1 {
		t.Errorf("WaveIDs count = %d, want 1 (should dedupe)", len(refs.WaveIDs))
	}
}

func TestValidateWaveRefFormat(t *testing.T) {
	valid := "wave://" + hex.EncodeToString(bytes.Repeat([]byte{0xcc}, 32))
	invalid := "wave://invalid"
	short := "wave://" + hex.EncodeToString(bytes.Repeat([]byte{0xdd}, 16))

	if !ValidateWaveRefFormat(valid) {
		t.Error("ValidateWaveRefFormat() = false for valid ref")
	}
	if ValidateWaveRefFormat(invalid) {
		t.Error("ValidateWaveRefFormat() = true for invalid ref")
	}
	if ValidateWaveRefFormat(short) {
		t.Error("ValidateWaveRefFormat() = true for short ref")
	}
}

func TestValidateMentionFormat(t *testing.T) {
	valid := "@" + hex.EncodeToString(bytes.Repeat([]byte{0xee}, 8))
	invalid := "@invalid"
	short := "@abcd"

	if !ValidateMentionFormat(valid) {
		t.Error("ValidateMentionFormat() = false for valid mention")
	}
	if ValidateMentionFormat(invalid) {
		t.Error("ValidateMentionFormat() = true for invalid mention")
	}
	if ValidateMentionFormat(short) {
		t.Error("ValidateMentionFormat() = true for short mention")
	}
}

func TestParseWaveRefString(t *testing.T) {
	waveID := bytes.Repeat([]byte{0xff}, 32)
	ref := "wave://" + hex.EncodeToString(waveID)

	id, ok := ParseWaveRefString(ref)
	if !ok {
		t.Error("ParseWaveRefString() returned false")
	}
	if !bytes.Equal(id, waveID) {
		t.Error("Wave ID mismatch")
	}

	_, ok = ParseWaveRefString("invalid")
	if ok {
		t.Error("ParseWaveRefString() returned true for invalid")
	}
}

func TestParseMentionString(t *testing.T) {
	userHash := bytes.Repeat([]byte{0x12}, 8)
	mention := "@" + hex.EncodeToString(userHash)

	hash, ok := ParseMentionString(mention)
	if !ok {
		t.Error("ParseMentionString() returned false")
	}
	if !bytes.Equal(hash, userHash) {
		t.Error("User hash mismatch")
	}

	_, ok = ParseMentionString("@invalid")
	if ok {
		t.Error("ParseMentionString() returned true for invalid")
	}
}

func TestCountRefs(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x34}, 32)
	userHash := bytes.Repeat([]byte{0x56}, 8)

	content := []byte(
		"wave://" + hex.EncodeToString(waveID) +
			" wave://" + hex.EncodeToString(waveID) +
			" @" + hex.EncodeToString(userHash),
	)

	counts := CountRefs(content)
	if counts.WaveRefs != 2 {
		t.Errorf("WaveRefs = %d, want 2", counts.WaveRefs)
	}
	if counts.Mentions != 1 {
		t.Errorf("Mentions = %d, want 1", counts.Mentions)
	}
	if counts.Total != 3 {
		t.Errorf("Total = %d, want 3", counts.Total)
	}
}

func TestStripReferences(t *testing.T) {
	waveID := bytes.Repeat([]byte{0x78}, 32)
	userHash := bytes.Repeat([]byte{0x9a}, 8)

	content := []byte("Hello wave://" + hex.EncodeToString(waveID) +
		" @" + hex.EncodeToString(userHash) + " world")

	result := StripReferences(content)
	expected := "Hello world"
	if string(result) != expected {
		t.Errorf("StripReferences() = %q, want %q", result, expected)
	}
}

func TestParseReferencesEmpty(t *testing.T) {
	refs := ParseReferences([]byte{})
	if len(refs) != 0 {
		t.Error("ParseReferences() should return empty for empty content")
	}

	refs = ParseReferences(nil)
	if len(refs) != 0 {
		t.Error("ParseReferences() should return empty for nil content")
	}
}

func TestParseReferencesNoRefs(t *testing.T) {
	content := []byte("Just a normal message with no references.")
	refs := ParseReferences(content)
	if len(refs) != 0 {
		t.Errorf("Expected 0 references, got %d", len(refs))
	}
}

func TestCaseInsensitiveHex(t *testing.T) {
	waveIDUpper := "AABBCCDD" + "00112233" + "44556677" + "8899AABB" +
		"CCDDEEFF" + "00112233" + "44556677" + "8899AABB"
	waveIDLower := "aabbccdd" + "00112233" + "44556677" + "8899aabb" +
		"ccddeeff" + "00112233" + "44556677" + "8899aabb"

	contentUpper := []byte("wave://" + waveIDUpper)
	contentLower := []byte("wave://" + waveIDLower)

	refsUpper := ParseWaveReferences(contentUpper)
	refsLower := ParseWaveReferences(contentLower)

	if len(refsUpper) != 1 {
		t.Error("Should parse uppercase hex")
	}
	if len(refsLower) != 1 {
		t.Error("Should parse lowercase hex")
	}
}
