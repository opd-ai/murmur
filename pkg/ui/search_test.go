// Package ui - Search Bar tests.
package ui

import (
	"testing"
)

func TestFilterResults(t *testing.T) {
	allResults := []SearchResult{
		{NodeID: "node1", DisplayName: "Alice", Pseudonym: "ShadowWalker", IsSpecter: false},
		{NodeID: "node2", DisplayName: "Bob", Pseudonym: "", IsSpecter: true},
		{NodeID: "node3", DisplayName: "Charlie", Pseudonym: "", IsSpecter: false},
		{NodeID: "node4", DisplayName: "Specter-Delta", Pseudonym: "PhantomDelta", IsSpecter: true},
	}

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query returns all", "", 4},
		{"match display name", "alice", 1},
		{"match pseudonym", "phantom", 1},
		{"match specter name", "delta", 1},
		{"partial match", "e", 4}, // Alice, Charlie, Specter-Delta, PhantomDelta
		{"no match", "xyz", 0},
		{"case insensitive", "BOB", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := FilterResults(tt.query, allResults)
			if len(results) != tt.expected {
				t.Errorf("FilterResults(%q) returned %d results, expected %d", tt.query, len(results), tt.expected)
			}
		})
	}
}

func TestSearchBar_ShowHide(t *testing.T) {
	callbacks := SearchCallbacks{
		OnSearch: func(query string) []SearchResult { return nil },
		OnSelect: func(nodeID string) {},
		OnClose:  func() {},
	}

	bar := NewSearchBar(DefaultTheme(), callbacks)

	if bar.Visible() {
		t.Error("SearchBar should be hidden initially")
	}

	bar.Show()
	if !bar.Visible() {
		t.Error("SearchBar should be visible after Show()")
	}

	bar.Hide()
	if bar.Visible() {
		t.Error("SearchBar should be hidden after Hide()")
	}
}

func TestSearchBar_Toggle(t *testing.T) {
	callbacks := SearchCallbacks{
		OnSearch: func(query string) []SearchResult { return nil },
		OnSelect: func(nodeID string) {},
		OnClose:  func() {},
	}

	bar := NewSearchBar(DefaultTheme(), callbacks)

	// Initial state: hidden
	if bar.Visible() {
		t.Error("SearchBar should be hidden initially")
	}

	// Toggle to visible
	bar.Toggle()
	if !bar.Visible() {
		t.Error("SearchBar should be visible after first Toggle()")
	}

	// Toggle back to hidden
	bar.Toggle()
	if bar.Visible() {
		t.Error("SearchBar should be hidden after second Toggle()")
	}
}
