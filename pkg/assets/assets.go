// Package assets provides embedded resources for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md, the build embeds the Specter name wordlist,
// default configuration, and onboarding assets via go:embed.
package assets

import (
	"bufio"
	"embed"
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
