// Package ui - Search Bar stub for test builds.
//
//go:build test
// +build test

package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// SearchCallbacks provides callbacks for search bar actions.
type SearchCallbacks struct {
	OnSearch func(query string) []SearchResult
	OnSelect func(nodeID string)
	OnClose  func()
}

// SearchBar provides node search functionality (stub).
type SearchBar struct {
	visible bool
}

// NewSearchBar creates a new search bar (stub).
func NewSearchBar(theme Theme, callbacks SearchCallbacks) *SearchBar {
	return &SearchBar{}
}

// Show displays the search bar (stub).
func (s *SearchBar) Show() { s.visible = true }

// Hide hides the search bar (stub).
func (s *SearchBar) Hide() { s.visible = false }

// Visible returns true if the search bar is currently shown (stub).
func (s *SearchBar) Visible() bool { return s.visible }

// Toggle toggles search bar visibility (stub).
func (s *SearchBar) Toggle() { s.visible = !s.visible }

// Update handles input and updates search bar state (stub).
func (s *SearchBar) Update() bool { return s.visible }

// Draw renders the search bar (stub).
func (s *SearchBar) Draw(screen *ebiten.Image) {}

// FilterResults performs case-insensitive substring matching on a list of results.
// Mirrors the real implementation in search.go (no Ebitengine dependency).
func FilterResults(query string, allResults []SearchResult) []SearchResult {
	if query == "" {
		return allResults
	}
	query = strings.ToLower(query)
	var filtered []SearchResult
	for _, result := range allResults {
		if strings.Contains(strings.ToLower(result.DisplayName), query) ||
			strings.Contains(strings.ToLower(result.Pseudonym), query) ||
			strings.Contains(strings.ToLower(result.NodeID), query) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}
