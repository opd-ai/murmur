package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Fallback deterministic snapshot-style integration tests.
func TestModelSnapshots_OnboardingComposeSpecterPulseNav(t *testing.T) {
	m := NewModel(Config{})

	// onboarding view (tab 5)
	m.active = tabOnboarding
	mv, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mv.(Model)
	if !strings.Contains(m.View(), "Current phase") {
		t.Fatal("expected onboarding snapshot content")
	}

	// waves compose (tab 3)
	m.active = tabIdentity
	mv, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = mv.(Model)
	m.active = tabWaves
	mv, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = mv.(Model)
	mv, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mv.(Model)
	if !strings.Contains(m.View(), "Compose") {
		t.Fatal("expected waves compose snapshot content")
	}

	// specter switch (tab 4)
	m.active = tabAnonymous
	mv, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = mv.(Model)
	if !strings.Contains(m.View(), "Active Specter") {
		t.Fatal("expected anonymous snapshot content")
	}

	// pulse map nav (tab 1)
	m.active = tabPulseMap
	mv, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = mv.(Model)
	if !strings.Contains(m.View(), "Camera") {
		t.Fatal("expected pulse map snapshot content")
	}
}
