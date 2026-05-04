// Package assets provides embedded resources for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md, the build embeds the Specter name wordlist,
// default configuration, and onboarding assets via go:embed.
package assets

import (
	"bufio"
	"embed"
	"fmt"
	"strings"
)

//go:embed wordlists/specter-names.txt
var wordlistFS embed.FS

// SpecterWordlist contains all 65,536 entries for procedural name generation.
// Loaded once at startup for efficient Specter name generation.
var SpecterWordlist []string

func init() {
	SpecterWordlist = loadWordlist("wordlists/specter-names.txt")
}

// loadWordlist reads a wordlist file and returns the entries as a slice.
func loadWordlist(path string) []string {
	data, err := wordlistFS.ReadFile(path)
	if err != nil {
		// Fallback to empty - caller should check
		return nil
	}

	var words []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") {
			words = append(words, word)
		}
	}

	return words
}

// WordlistFS provides direct access to the embedded wordlist filesystem.
func WordlistFS() embed.FS {
	return wordlistFS
}

// LoadWordlist loads a wordlist from the embedded filesystem.
// This is a public wrapper around loadWordlist for external use.
func LoadWordlist(path string) ([]string, error) {
	words := loadWordlist(path)
	if words == nil {
		return nil, fmt.Errorf("failed to load wordlist from %s", path)
	}
	return words, nil
}

// GetSpecterName returns the Specter name at the given index.
// Per DESIGN_DOCUMENT.md, Specter names are deterministically generated
// from the public key using index % 65536.
func GetSpecterName(index int) (string, error) {
	if len(SpecterWordlist) == 0 {
		return "", fmt.Errorf("SpecterWordlist not loaded")
	}
	if index < 0 {
		return "", fmt.Errorf("index must be non-negative")
	}
	idx := index % len(SpecterWordlist)
	return SpecterWordlist[idx], nil
}

// SpecterWordlistSize returns the number of entries in the Specter wordlist.
func SpecterWordlistSize() int {
	return len(SpecterWordlist)
}

// ValidateWordlist checks that the Specter wordlist has the expected size.
// Per DESIGN_DOCUMENT.md, the wordlist must contain exactly 65,536 entries.
func ValidateWordlist() error {
	const expectedSize = 65536
	if len(SpecterWordlist) == 0 {
		return fmt.Errorf("SpecterWordlist is empty")
	}
	if len(SpecterWordlist) != expectedSize {
		return fmt.Errorf("SpecterWordlist has %d entries, expected %d", len(SpecterWordlist), expectedSize)
	}
	return nil
}
