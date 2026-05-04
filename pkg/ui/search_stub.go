// Package ui - Search Bar stub for test builds.
//
//go:build test
// +build test

package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// SearchCallbacks provides callbacks for search bar actions.
type SearchCallbacks struct {
	OnSearch func(query string) []SearchResult
	OnSelect func(nodeID string)
	OnClose  func()
}

// SearchBar provides node search functionality (stub).
type SearchBar struct{}

// NewSearchBar creates a new search bar (stub).
func NewSearchBar(theme Theme, callbacks SearchCallbacks) *SearchBar {
	return &SearchBar{}
}

// Show displays the search bar (stub).
func (s *SearchBar) Show() {}

// Hide hides the search bar (stub).
func (s *SearchBar) Hide() {}

// Visible returns true if the search bar is currently shown (stub).
func (s *SearchBar) Visible() bool {
	return false
}

// Toggle toggles search bar visibility (stub).
func (s *SearchBar) Toggle() {}

// Update handles input and updates search bar state (stub).
func (s *SearchBar) Update() bool {
	return false
}

// Draw renders the search bar (stub).
func (s *SearchBar) Draw(screen *ebiten.Image) {}

// FilterResults performs case-insensitive substring matching (stub).
func FilterResults(query string, allResults []SearchResult) []SearchResult {
	return nil
}
