// Package waves provides Wave creation, signing, and validation.
// This file implements Wave reference parsing for inline links per WAVES.md.
// Supports wave://[hex-id] references and @[hash] user mentions.
package waves

import (
	"encoding/hex"
	"regexp"
	"strings"

	pb "github.com/opd-ai/murmur/proto"
)

// Reference types for Wave content parsing.
const (
	RefTypeWave    = "wave"    // wave://[hex-encoded Wave ID]
	RefTypeMention = "mention" // @[first 8 bytes of pubkey, hex-encoded]
)

// Reference represents a parsed reference in Wave content.
type Reference struct {
	Type       string // RefTypeWave or RefTypeMention
	ID         []byte // Wave ID (32 bytes) or user hash (8 bytes)
	Raw        string // Original matched text
	Start      int    // Start position in content
	End        int    // End position in content
	HexEncoded string // Hex-encoded ID for display
}

// Regular expressions for parsing references.
var (
	// wave://[64 hex chars] - references a Wave by its full ID.
	waveRefPattern = regexp.MustCompile(`wave://([0-9a-fA-F]{64})`)

	// @[16 hex chars] - mentions a user by first 8 bytes of their pubkey hash.
	mentionPattern = regexp.MustCompile(`@([0-9a-fA-F]{16})`)
)

// ParseReferences extracts all Wave and mention references from content.
func ParseReferences(content []byte) []Reference {
	text := string(content)
	var refs []Reference

	// Find wave:// references.
	waveMatches := waveRefPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range waveMatches {
		if len(match) >= 4 {
			hexID := text[match[2]:match[3]]
			id, err := hex.DecodeString(hexID)
			if err == nil && len(id) == 32 {
				refs = append(refs, Reference{
					Type:       RefTypeWave,
					ID:         id,
					Raw:        text[match[0]:match[1]],
					Start:      match[0],
					End:        match[1],
					HexEncoded: hexID,
				})
			}
		}
	}

	// Find @mention references.
	mentionMatches := mentionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range mentionMatches {
		if len(match) >= 4 {
			hexHash := text[match[2]:match[3]]
			hash, err := hex.DecodeString(hexHash)
			if err == nil && len(hash) == 8 {
				refs = append(refs, Reference{
					Type:       RefTypeMention,
					ID:         hash,
					Raw:        text[match[0]:match[1]],
					Start:      match[0],
					End:        match[1],
					HexEncoded: hexHash,
				})
			}
		}
	}

	return refs
}

// ParseWaveReferences extracts only Wave references from content.
func ParseWaveReferences(content []byte) []Reference {
	all := ParseReferences(content)
	var refs []Reference
	for _, ref := range all {
		if ref.Type == RefTypeWave {
			refs = append(refs, ref)
		}
	}
	return refs
}

// ParseMentions extracts only user mentions from content.
func ParseMentions(content []byte) []Reference {
	all := ParseReferences(content)
	var refs []Reference
	for _, ref := range all {
		if ref.Type == RefTypeMention {
			refs = append(refs, ref)
		}
	}
	return refs
}

// FormatWaveRef formats a Wave ID as a wave:// reference string.
func FormatWaveRef(waveID []byte) string {
	if len(waveID) != 32 {
		return ""
	}
	return "wave://" + hex.EncodeToString(waveID)
}

// FormatMention formats a user's public key hash as an @mention string.
// Takes the first 8 bytes of the public key or its hash.
func FormatMention(pubKeyHash []byte) string {
	if len(pubKeyHash) < 8 {
		return ""
	}
	return "@" + hex.EncodeToString(pubKeyHash[:8])
}

// HasReferences checks if content contains any references.
func HasReferences(content []byte) bool {
	return len(ParseReferences(content)) > 0
}

// HasWaveReferences checks if content contains Wave references.
func HasWaveReferences(content []byte) bool {
	return len(ParseWaveReferences(content)) > 0
}

// HasMentions checks if content contains user mentions.
func HasMentions(content []byte) bool {
	return len(ParseMentions(content)) > 0
}

// GetWaveRefs returns all Wave IDs referenced in a Wave's content.
func GetWaveRefs(wave *pb.Wave) [][]byte {
	if wave == nil {
		return nil
	}
	refs := ParseWaveReferences(wave.Content)
	ids := make([][]byte, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}
	return ids
}

// GetMentionedUsers returns all user hashes mentioned in a Wave's content.
func GetMentionedUsers(wave *pb.Wave) [][]byte {
	if wave == nil {
		return nil
	}
	refs := ParseMentions(wave.Content)
	hashes := make([][]byte, len(refs))
	for i, ref := range refs {
		hashes[i] = ref.ID
	}
	return hashes
}

// ReplaceWaveRefs replaces Wave references with custom text using a formatter.
// The formatter receives the Wave ID and returns the replacement text.
func ReplaceWaveRefs(content []byte, formatter func(waveID []byte) string) []byte {
	text := string(content)
	refs := ParseWaveReferences(content)

	// Replace in reverse order to maintain correct positions.
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		replacement := formatter(ref.ID)
		text = text[:ref.Start] + replacement + text[ref.End:]
	}

	return []byte(text)
}

// ReplaceMentions replaces @mentions with custom text using a formatter.
// The formatter receives the user hash and returns the replacement text.
func ReplaceMentions(content []byte, formatter func(userHash []byte) string) []byte {
	text := string(content)
	refs := ParseMentions(content)

	// Replace in reverse order to maintain correct positions.
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		replacement := formatter(ref.ID)
		text = text[:ref.Start] + replacement + text[ref.End:]
	}

	return []byte(text)
}

// ExtractRefIDs returns all unique Wave IDs and user hashes from content.
type ExtractedRefs struct {
	WaveIDs    [][]byte
	UserHashes [][]byte
}

// ExtractAllRefs extracts all unique references from content.
func ExtractAllRefs(content []byte) ExtractedRefs {
	refs := ParseReferences(content)

	waveIDSet := make(map[string][]byte)
	userHashSet := make(map[string][]byte)

	for _, ref := range refs {
		key := string(ref.ID)
		switch ref.Type {
		case RefTypeWave:
			waveIDSet[key] = ref.ID
		case RefTypeMention:
			userHashSet[key] = ref.ID
		}
	}

	result := ExtractedRefs{
		WaveIDs:    make([][]byte, 0, len(waveIDSet)),
		UserHashes: make([][]byte, 0, len(userHashSet)),
	}

	for _, id := range waveIDSet {
		result.WaveIDs = append(result.WaveIDs, id)
	}
	for _, hash := range userHashSet {
		result.UserHashes = append(result.UserHashes, hash)
	}

	return result
}

// ValidateWaveRefFormat validates that a string is a valid wave:// reference.
func ValidateWaveRefFormat(ref string) bool {
	return waveRefPattern.MatchString(ref)
}

// ValidateMentionFormat validates that a string is a valid @mention.
func ValidateMentionFormat(mention string) bool {
	return mentionPattern.MatchString(mention)
}

// ParseWaveRefString parses a wave:// reference string and returns the Wave ID.
func ParseWaveRefString(ref string) ([]byte, bool) {
	matches := waveRefPattern.FindStringSubmatch(ref)
	if len(matches) < 2 {
		return nil, false
	}

	id, err := hex.DecodeString(matches[1])
	if err != nil || len(id) != 32 {
		return nil, false
	}

	return id, true
}

// ParseMentionString parses an @mention string and returns the user hash.
func ParseMentionString(mention string) ([]byte, bool) {
	matches := mentionPattern.FindStringSubmatch(mention)
	if len(matches) < 2 {
		return nil, false
	}

	hash, err := hex.DecodeString(matches[1])
	if err != nil || len(hash) != 8 {
		return nil, false
	}

	return hash, true
}

// CountReferences counts the number of each reference type in content.
type RefCounts struct {
	WaveRefs int
	Mentions int
	Total    int
}

// CountRefs counts all references in content.
func CountRefs(content []byte) RefCounts {
	refs := ParseReferences(content)
	counts := RefCounts{Total: len(refs)}

	for _, ref := range refs {
		switch ref.Type {
		case RefTypeWave:
			counts.WaveRefs++
		case RefTypeMention:
			counts.Mentions++
		}
	}

	return counts
}

// StripReferences removes all references from content, leaving plain text.
func StripReferences(content []byte) []byte {
	text := string(content)

	// Remove wave:// references.
	text = waveRefPattern.ReplaceAllString(text, "")

	// Remove @mentions.
	text = mentionPattern.ReplaceAllString(text, "")

	// Clean up double spaces.
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return []byte(strings.TrimSpace(text))
}
