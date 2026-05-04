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

func TestLoadWordlist(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		words, err := LoadWordlist("wordlists/specter-names.txt")
		if err != nil {
			t.Fatalf("LoadWordlist failed: %v", err)
		}
		if len(words) == 0 {
			t.Error("LoadWordlist should return non-empty slice")
		}
		if len(words) != 65536 {
			t.Errorf("LoadWordlist returned %d words, expected 65536", len(words))
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := LoadWordlist("nonexistent.txt")
		if err == nil {
			t.Error("LoadWordlist should return error for invalid path")
		}
	})
}

func TestGetSpecterName(t *testing.T) {
	t.Run("valid index", func(t *testing.T) {
		name, err := GetSpecterName(0)
		if err != nil {
			t.Fatalf("GetSpecterName failed: %v", err)
		}
		if name == "" {
			t.Error("GetSpecterName should return non-empty name")
		}

		// Test another index
		name2, err := GetSpecterName(100)
		if err != nil {
			t.Fatalf("GetSpecterName failed: %v", err)
		}
		if name2 == "" {
			t.Error("GetSpecterName should return non-empty name")
		}
	})

	t.Run("wrapping index", func(t *testing.T) {
		// Test that index wraps around
		size := SpecterWordlistSize()
		name1, _ := GetSpecterName(0)
		name2, _ := GetSpecterName(size)
		if name1 != name2 {
			t.Error("GetSpecterName should wrap around at wordlist size")
		}
	})

	t.Run("negative index", func(t *testing.T) {
		_, err := GetSpecterName(-1)
		if err == nil {
			t.Error("GetSpecterName should return error for negative index")
		}
	})
}

func TestSpecterWordlistSize(t *testing.T) {
	size := SpecterWordlistSize()
	if size != 65536 {
		t.Errorf("SpecterWordlistSize = %d, expected 65536", size)
	}
	if size != len(SpecterWordlist) {
		t.Error("SpecterWordlistSize should match len(SpecterWordlist)")
	}
}

func TestValidateWordlist(t *testing.T) {
	err := ValidateWordlist()
	if err != nil {
		t.Errorf("ValidateWordlist failed: %v", err)
	}
}
