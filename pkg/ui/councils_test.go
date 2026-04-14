// Package ui — Phantom Council management panel tests.
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewCouncilPanel(t *testing.T) {
	theme := DefaultTheme()
	panel := NewCouncilPanel(theme)

	if panel == nil {
		t.Fatal("NewCouncilPanel returned nil")
	}
	if panel.IsVisible() {
		t.Error("expected panel to be hidden by default")
	}
}

func TestCouncilPanel_ShowHide(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	panel.Show()
	if !panel.IsVisible() {
		t.Error("expected panel to be visible after Show()")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("expected panel to be hidden after Hide()")
	}
}

func TestCouncilPanel_ShowCouncilDetail(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	council := &CouncilInfo{
		ID:       [32]byte{1, 2, 3},
		Name:     "Test Council",
		Purpose:  "Testing",
		State:    CouncilStateActive,
		IsMember: true,
	}

	panel.ShowCouncilDetail(council)

	if !panel.IsVisible() {
		t.Error("expected panel to be visible after ShowCouncilDetail()")
	}
}

func TestCouncilPanel_SetCouncils(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	councils := []*CouncilInfo{
		{ID: [32]byte{1}, Name: "Council One"},
		{ID: [32]byte{2}, Name: "Council Two"},
		{ID: [32]byte{3}, Name: "Council Three"},
	}

	panel.SetCouncils(councils)
	// No crash means success for this basic test.
}

func TestCouncilPanel_Callbacks(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	// Test all callback setters don't panic.
	panel.SetOnCreateCouncil(func(name, purpose string, minResonance float64, maxMembers int) error {
		return nil
	})

	panel.SetOnInviteMember(func(councilID, specterKey [32]byte) error {
		return nil
	})

	panel.SetOnVoteAdmit(func(councilID, applicantKey [32]byte, vote VoteValue) error {
		return nil
	})

	panel.SetOnVoteExpel(func(councilID, memberKey [32]byte, vote VoteValue) error {
		return nil
	})

	panel.SetOnVoteProposal(func(councilID, proposalID [32]byte, vote VoteValue) error {
		return nil
	})

	panel.SetOnCreateProposal(func(councilID [32]byte, text string) error {
		return nil
	})

	panel.SetOnLeaveCouncil(func(councilID [32]byte) error {
		return nil
	})

	panel.SetOnInitExpel(func(councilID, memberKey [32]byte) error {
		return nil
	})
}

func TestCouncilPanel_Update(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	// Update when hidden should be no-op.
	err := panel.Update()
	if err != nil {
		t.Errorf("Update() returned error: %v", err)
	}

	// Update when visible.
	panel.Show()
	err = panel.Update()
	if err != nil {
		t.Errorf("Update() when visible returned error: %v", err)
	}
}

func TestCouncilPanel_Draw(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())
	screen := ebiten.NewImage(800, 600)

	// Draw when hidden should be no-op.
	panel.Draw(screen)

	// Draw when visible.
	panel.Show()
	panel.Draw(screen)

	// Draw with council detail.
	council := &CouncilInfo{
		ID:       [32]byte{1},
		Name:     "Test",
		Purpose:  "Testing",
		State:    CouncilStateActive,
		IsMember: true,
		Members: []CouncilMemberInfo{
			{SpecterKey: [32]byte{10}, Name: "Member 1", Status: MemberStatusActive},
			{SpecterKey: [32]byte{20}, Name: "Member 2", Status: MemberStatusActive},
		},
		Proposals: []CouncilProposalInfo{
			{ID: [32]byte{100}, Text: "Proposal 1", Votes: map[string]VoteValue{}},
		},
	}
	panel.ShowCouncilDetail(council)
	panel.Draw(screen)
}

func TestCouncilPanel_SetTheme(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())

	newTheme := DefaultTheme()
	newTheme.FontSize = 16

	panel.SetTheme(newTheme)
	// Verify it doesn't panic.
}

func TestCouncilStateString(t *testing.T) {
	tests := []struct {
		state    CouncilState
		expected string
	}{
		{CouncilStateActive, "Active"},
		{CouncilStateDormant, "Dormant"},
		{CouncilStateDisbanded, "Disbanded"},
		{CouncilState(99), "Unknown"},
	}

	for _, tc := range tests {
		result := CouncilStateString(tc.state)
		if result != tc.expected {
			t.Errorf("CouncilStateString(%v) = %q, expected %q", tc.state, result, tc.expected)
		}
	}
}

func TestMemberStatusString(t *testing.T) {
	tests := []struct {
		status   MemberStatus
		expected string
	}{
		{MemberStatusPending, "Pending"},
		{MemberStatusActive, "Active"},
		{MemberStatusExpelled, "Expelled"},
		{MemberStatusDeparted, "Departed"},
		{MemberStatus(99), "Unknown"},
	}

	for _, tc := range tests {
		result := MemberStatusString(tc.status)
		if result != tc.expected {
			t.Errorf("MemberStatusString(%v) = %q, expected %q", tc.status, result, tc.expected)
		}
	}
}

func TestVoteValueString(t *testing.T) {
	tests := []struct {
		vote     VoteValue
		expected string
	}{
		{VoteValueFor, "For"},
		{VoteValueAgainst, "Against"},
		{VoteValueAbstain, "Abstain"},
		{VoteValue(99), "Unknown"},
	}

	for _, tc := range tests {
		result := VoteValueString(tc.vote)
		if result != tc.expected {
			t.Errorf("VoteValueString(%v) = %q, expected %q", tc.vote, result, tc.expected)
		}
	}
}

func TestCouncilPanel_ListMode(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())
	screen := ebiten.NewImage(800, 600)

	// Empty list.
	panel.Show()
	panel.Draw(screen)

	// With councils.
	councils := []*CouncilInfo{
		{ID: [32]byte{1}, Name: "Council One", State: CouncilStateActive},
		{ID: [32]byte{2}, Name: "Council Two", State: CouncilStateDormant},
	}
	panel.SetCouncils(councils)
	panel.Draw(screen)
}

func TestCouncilPanel_WithFullCouncilData(t *testing.T) {
	panel := NewCouncilPanel(DefaultTheme())
	screen := ebiten.NewImage(800, 600)

	council := &CouncilInfo{
		ID:           [32]byte{1, 2, 3},
		Name:         "Full Test Council",
		Purpose:      "A council for comprehensive testing",
		State:        CouncilStateActive,
		CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
		MinResonance: 200,
		MaxMembers:   13,
		IsMember:     true,
		IsCreator:    true,
		Members: []CouncilMemberInfo{
			{SpecterKey: [32]byte{10}, Name: "Creator", Status: MemberStatusActive, JoinedAt: time.Now()},
			{SpecterKey: [32]byte{20}, Name: "Member 2", Status: MemberStatusActive, JoinedAt: time.Now()},
			{SpecterKey: [32]byte{30}, Name: "Member 3", Status: MemberStatusActive, JoinedAt: time.Now()},
			{SpecterKey: [32]byte{40}, Name: "Former", Status: MemberStatusDeparted, JoinedAt: time.Now()},
		},
		Applications: []CouncilApplicationInfo{
			{
				ApplicantKey:  [32]byte{50},
				ApplicantName: "Applicant 1",
				AppliedAt:     time.Now(),
				Votes:         map[string]VoteValue{},
			},
		},
		Proposals: []CouncilProposalInfo{
			{
				ID:        [32]byte{100},
				Text:      "Proposal to do something",
				CreatedAt: time.Now(),
				Votes:     map[string]VoteValue{"key1": VoteValueFor, "key2": VoteValueAgainst},
			},
			{
				ID:        [32]byte{101},
				Text:      "Another proposal",
				CreatedAt: time.Now(),
				Resolved:  true,
				Passed:    true,
				Votes:     map[string]VoteValue{"key1": VoteValueFor, "key2": VoteValueFor},
			},
		},
	}

	panel.ShowCouncilDetail(council)
	panel.Draw(screen)
}

func TestHandleTextInput(t *testing.T) {
	// Test that handleTextInput respects max length.
	current := "hello"
	result := handleTextInput(current, 10)

	// Without actual key events, this should return unchanged.
	if result != current {
		t.Errorf("expected %q, got %q", current, result)
	}
}

func TestSin32(t *testing.T) {
	// Basic sanity check for sin approximation.
	result := sin32(0)
	if result > 0.1 || result < -0.1 {
		t.Errorf("sin32(0) = %f, expected ~0", result)
	}

	result = sin32(1.5708) // pi/2
	if result < 0.9 || result > 1.1 {
		t.Errorf("sin32(pi/2) = %f, expected ~1", result)
	}
}
