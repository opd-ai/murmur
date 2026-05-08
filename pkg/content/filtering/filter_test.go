// Package filtering provides local content filtering for MURMUR.
package filtering

import (
	"fmt"
	"testing"

	pb "github.com/opd-ai/murmur/proto"
)

func TestFilterMuteAuthor(t *testing.T) {
	f := NewFilter()

	pubkey := []byte("testpubkey12345678901234567890ab")

	// Not muted initially.
	if f.IsMutedAuthor(pubkey) {
		t.Error("author should not be muted initially")
	}

	// Mute the author.
	f.MuteAuthor(pubkey)

	if !f.IsMutedAuthor(pubkey) {
		t.Error("author should be muted")
	}

	// Check MutedAuthors list.
	authors := f.MutedAuthors()
	if len(authors) != 1 {
		t.Errorf("expected 1 muted author, got %d", len(authors))
	}

	// Unmute the author.
	f.UnmuteAuthor(pubkey)

	if f.IsMutedAuthor(pubkey) {
		t.Error("author should not be muted after unmute")
	}
}

func TestFilterMuteKeyword(t *testing.T) {
	f := NewFilter()

	// Add a keyword.
	_ = f.MuteKeyword("spam")

	keywords := f.MutedKeywords()
	if len(keywords) != 1 || keywords[0] != "spam" {
		t.Errorf("expected ['spam'], got %v", keywords)
	}

	// Add duplicate - should not add again.
	_ = f.MuteKeyword("spam")
	if len(f.MutedKeywords()) != 1 {
		t.Error("duplicate keyword should not be added")
	}

	// Add another keyword.
	_ = f.MuteKeyword("casino")
	if len(f.MutedKeywords()) != 2 {
		t.Error("second keyword should be added")
	}

	// Remove keyword.
	f.UnmuteKeyword("spam")
	keywords = f.MutedKeywords()
	if len(keywords) != 1 || keywords[0] != "casino" {
		t.Errorf("expected ['casino'], got %v", keywords)
	}
}

func TestFilterMuteKeywordLimit(t *testing.T) {
	f := NewFilter()

	// Fill up to the limit.
	for i := 0; i < MaxKeywordPatterns; i++ {
		if err := f.MuteKeyword(fmt.Sprintf("keyword-%d", i)); err != nil {
			t.Fatalf("expected no error adding keyword %d, got %v", i, err)
		}
	}
	if len(f.MutedKeywords()) != MaxKeywordPatterns {
		t.Errorf("expected %d keywords, got %d", MaxKeywordPatterns, len(f.MutedKeywords()))
	}

	// Adding one more should fail.
	if err := f.MuteKeyword("overflow"); err == nil {
		t.Error("expected ErrTooManyPatterns, got nil")
	} else if err != ErrTooManyPatterns {
		t.Errorf("expected ErrTooManyPatterns, got %v", err)
	}
}


func TestFilterShouldFilterByAuthor(t *testing.T) {
	f := NewFilter()

	pubkey := []byte("testpubkey12345678901234567890ab")
	wave := &pb.Wave{
		AuthorPubkey: pubkey,
		Content:      []byte("hello world"),
	}

	// Not filtered initially.
	if f.ShouldFilter(wave) {
		t.Error("wave should not be filtered initially")
	}

	// Mute the author.
	f.MuteAuthor(pubkey)

	if !f.ShouldFilter(wave) {
		t.Error("wave should be filtered after muting author")
	}
}

func TestFilterShouldFilterByKeyword(t *testing.T) {
	f := NewFilter()

	wave := &pb.Wave{
		AuthorPubkey: []byte("pubkey"),
		Content:      []byte("Check out this SPAM offer!"),
	}

	// Not filtered initially.
	if f.ShouldFilter(wave) {
		t.Error("wave should not be filtered initially")
	}

	// Mute keyword (case-insensitive).
	_ = f.MuteKeyword("spam")

	if !f.ShouldFilter(wave) {
		t.Error("wave should be filtered due to keyword")
	}
}

func TestFilterKeywordWildcard(t *testing.T) {
	f := NewFilter()

	tests := []struct {
		pattern  string
		content  string
		expected bool
	}{
		// Simple contains.
		{"spam", "this is spam content", true},
		{"spam", "this is ham content", false},

		// Prefix wildcard - means "anything before spam".
		{"*spam", "this is spam", true},
		{"*spam", "spam is bad", true},
		{"*spam", "spammer detected", true}, // Contains spam.

		// Suffix wildcard - means "spam followed by anything".
		{"spam*", "spammer detected", true},
		{"spam*", "not about spam", true}, // Contains spam.

		// Contains wildcard (both sides).
		{"*spam*", "this is spam content", true},
		{"*spam*", "spammer detected", true},
		{"*spam*", "ham and eggs", false},

		// Multiple wildcards.
		{"buy*now", "buy cheap stuff now", true},
		{"buy*now", "buynow", true},
		{"buy*now", "buy stuff later", false},

		// Complex pattern.
		{"*free*money*", "get free easy money today", true},
		{"*free*money*", "free stuff", false},
	}

	for _, tc := range tests {
		f.Clear()
		_ = f.MuteKeyword(tc.pattern)

		wave := &pb.Wave{
			AuthorPubkey: []byte("pubkey"),
			Content:      []byte(tc.content),
		}

		filtered := f.ShouldFilter(wave)
		if filtered != tc.expected {
			t.Errorf("pattern=%q content=%q: expected filtered=%v, got %v",
				tc.pattern, tc.content, tc.expected, filtered)
		}
	}
}

func TestFilterResonance(t *testing.T) {
	f := NewFilter()

	// Set up mock Resonance lookup.
	resonanceScores := map[string]int{
		"high":   100,
		"medium": 50,
		"low":    10,
	}

	f.SetResonanceLookup(func(pubkey []byte) int {
		return resonanceScores[string(pubkey)]
	})

	// Set minimum Resonance threshold.
	f.SetMinResonance(25)

	tests := []struct {
		author   string
		expected bool // true = should be filtered
	}{
		{"high", false},   // 100 >= 25, not filtered
		{"medium", false}, // 50 >= 25, not filtered
		{"low", true},     // 10 < 25, filtered
	}

	for _, tc := range tests {
		wave := &pb.Wave{
			AuthorPubkey: []byte(tc.author),
			Content:      []byte("content"),
		}

		filtered := f.ShouldFilter(wave)
		if filtered != tc.expected {
			t.Errorf("author=%s: expected filtered=%v, got %v",
				tc.author, tc.expected, filtered)
		}
	}

	// Disable Resonance filtering.
	f.SetMinResonance(0)

	wave := &pb.Wave{
		AuthorPubkey: []byte("low"),
		Content:      []byte("content"),
	}
	if f.ShouldFilter(wave) {
		t.Error("wave should not be filtered when minResonance=0")
	}
}

func TestFilterWaves(t *testing.T) {
	f := NewFilter()

	// Mute one author.
	mutedPubkey := []byte("muted")
	f.MuteAuthor(mutedPubkey)

	waves := []*pb.Wave{
		{AuthorPubkey: []byte("good"), Content: []byte("hello")},
		{AuthorPubkey: mutedPubkey, Content: []byte("blocked")},
		{AuthorPubkey: []byte("other"), Content: []byte("world")},
	}

	filtered := f.FilterWaves(waves)
	if len(filtered) != 2 {
		t.Errorf("expected 2 waves, got %d", len(filtered))
	}

	for _, w := range filtered {
		if string(w.AuthorPubkey) == "muted" {
			t.Error("muted wave should have been filtered")
		}
	}
}

func TestFilterNilWave(t *testing.T) {
	f := NewFilter()

	if !f.ShouldFilter(nil) {
		t.Error("nil wave should be filtered")
	}
}

func TestFilterClear(t *testing.T) {
	f := NewFilter()

	f.MuteAuthor([]byte("author"))
	_ = f.MuteKeyword("spam")
	f.SetMinResonance(50)

	stats := f.Stats()
	if stats.MutedAuthorCount != 1 || stats.MutedKeywordCount != 1 || stats.MinResonance != 50 {
		t.Error("filter should have values before clear")
	}

	f.Clear()

	stats = f.Stats()
	if stats.MutedAuthorCount != 0 || stats.MutedKeywordCount != 0 || stats.MinResonance != 0 {
		t.Error("filter should be empty after clear")
	}
}

func TestFilterStats(t *testing.T) {
	f := NewFilter()

	stats := f.Stats()
	if stats.MutedAuthorCount != 0 || stats.MutedKeywordCount != 0 || stats.MinResonance != 0 {
		t.Error("new filter should have zero stats")
	}

	f.MuteAuthor([]byte("a1"))
	f.MuteAuthor([]byte("a2"))
	_ = f.MuteKeyword("k1")
	f.SetMinResonance(25)

	stats = f.Stats()
	if stats.MutedAuthorCount != 2 {
		t.Errorf("expected 2 muted authors, got %d", stats.MutedAuthorCount)
	}
	if stats.MutedKeywordCount != 1 {
		t.Errorf("expected 1 muted keyword, got %d", stats.MutedKeywordCount)
	}
	if stats.MinResonance != 25 {
		t.Errorf("expected minResonance=25, got %d", stats.MinResonance)
	}
}

func TestFilterCombinedCriteria(t *testing.T) {
	f := NewFilter()

	// Set up multiple filter criteria.
	f.MuteAuthor([]byte("banned"))
	_ = f.MuteKeyword("spam")
	f.SetMinResonance(25)
	f.SetResonanceLookup(func(pubkey []byte) int {
		if string(pubkey) == "lowres" {
			return 10
		}
		return 100
	})

	tests := []struct {
		name     string
		wave     *pb.Wave
		expected bool // true = filtered
	}{
		{
			name:     "good wave",
			wave:     &pb.Wave{AuthorPubkey: []byte("good"), Content: []byte("hello")},
			expected: false,
		},
		{
			name:     "banned author",
			wave:     &pb.Wave{AuthorPubkey: []byte("banned"), Content: []byte("hello")},
			expected: true,
		},
		{
			name:     "spam content",
			wave:     &pb.Wave{AuthorPubkey: []byte("good"), Content: []byte("spam offer")},
			expected: true,
		},
		{
			name:     "low resonance",
			wave:     &pb.Wave{AuthorPubkey: []byte("lowres"), Content: []byte("hello")},
			expected: true,
		},
		{
			name:     "multiple violations",
			wave:     &pb.Wave{AuthorPubkey: []byte("banned"), Content: []byte("spam")},
			expected: true,
		},
	}

	for _, tc := range tests {
		filtered := f.ShouldFilter(tc.wave)
		if filtered != tc.expected {
			t.Errorf("%s: expected filtered=%v, got %v", tc.name, tc.expected, filtered)
		}
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		text     string
		expected bool
	}{
		// Simple contains.
		{"abc", "xxabcxx", true},
		{"abc", "xxabXxx", false},

		// Exact match (no wildcard).
		{"abc", "abc", true},

		// Prefix only - means "abc somewhere in text".
		{"abc*", "abcdef", true},
		{"abc*", "xabc", true}, // Contains abc.

		// Suffix only - means "abc somewhere in text".
		{"*abc", "xyzabc", true},
		{"*abc", "abcx", true}, // Contains abc.

		// Both sides.
		{"*abc*", "xyzabcdef", true},
		{"*abc*", "xyz", false},

		// Multiple wildcards.
		{"a*b*c", "axbxc", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "axc", false},

		// Empty pattern.
		{"", "anything", true},
		{"*", "anything", true},
	}

	for _, tc := range tests {
		result := matchWildcard(tc.pattern, tc.text)
		if result != tc.expected {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v",
				tc.pattern, tc.text, result, tc.expected)
		}
	}
}
