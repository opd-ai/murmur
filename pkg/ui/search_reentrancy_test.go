//go:build !test
// +build !test

package ui

import "testing"

func TestSearchBar_PerformSearch_DropsStaleResultsWhenHidden(t *testing.T) {
	bar := NewSearchBar(DefaultTheme(), SearchCallbacks{})
	bar.Show()
	bar.callbacks.OnSearch = func(query string) []SearchResult {
		bar.Hide() // Simulate concurrent close while search callback is running.
		return []SearchResult{{NodeID: "node1", DisplayName: "Alice"}}
	}

	bar.mu.Lock()
	bar.query = "alice"
	bar.performSearch()
	bar.mu.Unlock()

	if bar.visible {
		t.Fatal("expected search bar to be hidden by callback")
	}
	if len(bar.results) != 0 {
		t.Fatalf("expected stale results to be dropped, got %d", len(bar.results))
	}
}

func TestSearchBar_ShouldPerformSearch_DebounceTicks(t *testing.T) {
	bar := NewSearchBar(DefaultTheme(), SearchCallbacks{
		OnSearch: func(query string) []SearchResult { return nil },
	})
	bar.Show()

	bar.mu.Lock()
	bar.tickCount = 10
	bar.lastInputTick = 10
	bar.lastSearchTick = 0
	if bar.shouldPerformSearch() {
		bar.mu.Unlock()
		t.Fatal("expected debounce to block immediate search dispatch")
	}

	bar.tickCount = 12
	if bar.shouldPerformSearch() {
		bar.mu.Unlock()
		t.Fatal("expected debounce to block search before 3 ticks")
	}

	bar.tickCount = 13
	if !bar.shouldPerformSearch() {
		bar.mu.Unlock()
		t.Fatal("expected debounce to allow search at >= 3 ticks")
	}
	bar.mu.Unlock()
}
