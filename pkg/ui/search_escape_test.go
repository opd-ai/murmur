//go:build !test
// +build !test

package ui

import "testing"

func TestSearchBar_Update_VisibleConsumesInput(t *testing.T) {
	bar := NewSearchBar(DefaultTheme(), SearchCallbacks{})
	bar.Show()

	consumed := bar.Update()
	if !consumed {
		t.Fatal("expected Update() to return true while search is visible to block click-through input")
	}
}
