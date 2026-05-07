//go:build !test
// +build !test

package ui

import "testing"

func TestSearchBar_Update_NoEscapeDoesNotConsumeInput(t *testing.T) {
	bar := NewSearchBar(DefaultTheme(), SearchCallbacks{})
	bar.Show()

	consumed := bar.Update()
	if consumed {
		t.Fatal("expected Update() to return false when Escape is not pressed and no input occurred")
	}
}
