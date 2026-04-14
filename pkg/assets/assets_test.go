package assets

import (
	"testing"
)

func TestSpecterWordlistLoaded(t *testing.T) {
	if len(SpecterWordlist) == 0 {
		t.Fatal("SpecterWordlist should not be empty")
	}

	// Per spec, the wordlist should have 65,536 entries
	if len(SpecterWordlist) != 65536 {
		t.Errorf("SpecterWordlist has %d entries, expected 65536", len(SpecterWordlist))
	}
}

func TestWordlistFS(t *testing.T) {
	fs := WordlistFS()

	// Should be able to read the wordlist file
	data, err := fs.ReadFile("wordlists/specter-names.txt")
	if err != nil {
		t.Fatalf("failed to read wordlist: %v", err)
	}

	if len(data) == 0 {
		t.Error("wordlist file should not be empty")
	}
}

func TestWordlistEntries(t *testing.T) {
	// Spot check some entries exist and are non-empty
	if len(SpecterWordlist) > 0 {
		first := SpecterWordlist[0]
		if first == "" {
			t.Error("first wordlist entry should not be empty")
		}

		last := SpecterWordlist[len(SpecterWordlist)-1]
		if last == "" {
			t.Error("last wordlist entry should not be empty")
		}
	}
}
