// Package filtering provides local content filtering for MURMUR.
// Per DESIGN_DOCUMENT.md, all filtering is local - the network relays all content,
// but nodes can choose what to display. This respects user autonomy while
// maintaining censorship resistance.
package filtering

import (
	"encoding/hex"
	"strings"
	"sync"

	pb "github.com/opd-ai/murmur/proto"
)

// Filter provides local content filtering based on author, keywords, and Resonance.
// Per the design principles, filtering is purely local and does not affect propagation.
type Filter struct {
	mu sync.RWMutex

	// Muted authors by public key (hex-encoded).
	mutedAuthors map[string]struct{}

	// Keyword patterns (supports * wildcard).
	mutedKeywords []string

	// Minimum Resonance threshold (0 = no filter).
	minResonance int

	// Optional Resonance lookup function.
	resonanceLookup func(pubkey []byte) int
}

// NewFilter creates a new content filter.
func NewFilter() *Filter {
	return &Filter{
		mutedAuthors:  make(map[string]struct{}),
		mutedKeywords: make([]string, 0),
	}
}

// SetResonanceLookup sets the function used to lookup Resonance scores.
func (f *Filter) SetResonanceLookup(fn func(pubkey []byte) int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resonanceLookup = fn
}

// MuteAuthor adds an author's public key to the mute list.
func (f *Filter) MuteAuthor(pubkey []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mutedAuthors[hex.EncodeToString(pubkey)] = struct{}{}
}

// UnmuteAuthor removes an author from the mute list.
func (f *Filter) UnmuteAuthor(pubkey []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.mutedAuthors, hex.EncodeToString(pubkey))
}

// IsMutedAuthor checks if an author is muted.
func (f *Filter) IsMutedAuthor(pubkey []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, muted := f.mutedAuthors[hex.EncodeToString(pubkey)]
	return muted
}

// MutedAuthors returns a copy of all muted author public keys.
func (f *Filter) MutedAuthors() [][]byte {
	f.mu.RLock()
	defer f.mu.RUnlock()

	authors := make([][]byte, 0, len(f.mutedAuthors))
	for hexKey := range f.mutedAuthors {
		key, err := hex.DecodeString(hexKey)
		if err == nil {
			authors = append(authors, key)
		}
	}
	return authors
}

// MuteKeyword adds a keyword pattern to the mute list.
// Patterns support * as a wildcard that matches any characters.
// Examples: "spam*", "*casino*", "buy*now"
func (f *Filter) MuteKeyword(pattern string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	pattern = strings.ToLower(pattern)
	// Check for duplicates.
	for _, existing := range f.mutedKeywords {
		if existing == pattern {
			return
		}
	}
	f.mutedKeywords = append(f.mutedKeywords, pattern)
}

// UnmuteKeyword removes a keyword pattern from the mute list.
func (f *Filter) UnmuteKeyword(pattern string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	pattern = strings.ToLower(pattern)
	newKeywords := make([]string, 0, len(f.mutedKeywords))
	for _, kw := range f.mutedKeywords {
		if kw != pattern {
			newKeywords = append(newKeywords, kw)
		}
	}
	f.mutedKeywords = newKeywords
}

// MutedKeywords returns a copy of all muted keyword patterns.
func (f *Filter) MutedKeywords() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keywords := make([]string, len(f.mutedKeywords))
	copy(keywords, f.mutedKeywords)
	return keywords
}

// SetMinResonance sets the minimum Resonance score for content display.
// Waves from authors with Resonance below this threshold are filtered.
// Set to 0 to disable Resonance filtering.
func (f *Filter) SetMinResonance(min int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.minResonance = min
}

// MinResonance returns the current minimum Resonance threshold.
func (f *Filter) MinResonance() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.minResonance
}

// ShouldFilter returns true if the Wave should be hidden from display.
// This checks all filter criteria: author, keywords, and Resonance.
func (f *Filter) ShouldFilter(wave *pb.Wave) bool {
	if wave == nil {
		return true
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check author mute list.
	if len(wave.AuthorPubkey) > 0 {
		if _, muted := f.mutedAuthors[hex.EncodeToString(wave.AuthorPubkey)]; muted {
			return true
		}
	}

	// Check keyword filters.
	if f.containsMutedKeyword(wave.Content) {
		return true
	}

	// Check Resonance threshold.
	if f.minResonance > 0 && f.resonanceLookup != nil {
		authorResonance := f.resonanceLookup(wave.AuthorPubkey)
		if authorResonance < f.minResonance {
			return true
		}
	}

	return false
}

// containsMutedKeyword checks if content matches any muted keyword pattern.
func (f *Filter) containsMutedKeyword(content []byte) bool {
	if len(f.mutedKeywords) == 0 {
		return false
	}

	text := strings.ToLower(string(content))

	for _, pattern := range f.mutedKeywords {
		if matchWildcard(pattern, text) {
			return true
		}
	}
	return false
}

// matchWildcard performs wildcard pattern matching.
// The pattern can contain * which matches any sequence of characters.
// This is similar to glob matching but looks for the pattern anywhere in text.
func matchWildcard(pattern, text string) bool {
	// Handle simple case: no wildcards.
	if !strings.Contains(pattern, "*") {
		return strings.Contains(text, pattern)
	}

	// Handle empty pattern or just wildcards.
	pattern = strings.Trim(pattern, "*")
	if pattern == "" {
		return true
	}

	// Split by wildcards.
	parts := strings.Split(pattern, "*")

	// Empty parts list means pattern was just wildcards.
	if len(parts) == 0 {
		return true
	}

	// For a pattern like "*spam*", just check if all parts appear in order.
	pos := 0
	for _, part := range parts {
		if part == "" {
			continue
		}

		idx := strings.Index(text[pos:], part)
		if idx == -1 {
			return false
		}

		pos += idx + len(part)
	}

	return true
}

// FilterWaves returns only the Waves that should be displayed.
// This is a convenience function that filters a slice of Waves.
func (f *Filter) FilterWaves(waves []*pb.Wave) []*pb.Wave {
	result := make([]*pb.Wave, 0, len(waves))
	for _, wave := range waves {
		if !f.ShouldFilter(wave) {
			result = append(result, wave)
		}
	}
	return result
}

// Clear removes all filter rules.
func (f *Filter) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.mutedAuthors = make(map[string]struct{})
	f.mutedKeywords = make([]string, 0)
	f.minResonance = 0
}

// Stats returns filter statistics.
type Stats struct {
	MutedAuthorCount  int
	MutedKeywordCount int
	MinResonance      int
}

// Stats returns current filter statistics.
func (f *Filter) Stats() Stats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return Stats{
		MutedAuthorCount:  len(f.mutedAuthors),
		MutedKeywordCount: len(f.mutedKeywords),
		MinResonance:      f.minResonance,
	}
}
