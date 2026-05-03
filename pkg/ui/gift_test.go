// Package ui - Tests for GiftPanel.
//
//go:build noebiten
// +build noebiten

package ui

import (
	"errors"
	"testing"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

func TestNewGiftPanel(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{}

	panel := NewGiftPanel(theme, callbacks)
	if panel == nil {
		t.Fatal("NewGiftPanel returned nil")
	}
	if panel.IsVisible() {
		t.Error("Panel should start hidden")
	}
	if panel.GetMode() != GiftModeEffectSelect {
		t.Error("Panel should start in effect select mode")
	}
}

func TestGiftPanelShowHide(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients: func() []RecipientInfo {
			return []RecipientInfo{
				{NodeID: "abc123", DisplayName: "TestNode", IsSurface: true},
			}
		},
	}

	panel := NewGiftPanel(theme, callbacks)

	panel.Show()
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("Panel should be hidden after Hide")
	}
}

func TestGiftPanelLoadsEffects(t *testing.T) {
	tests := []struct {
		name      string
		resonance int
		expected  int // Minimum expected effects
	}{
		{"no effects", 0, 0},
		{"basic effects", 25, 5},
		{"expanded effects", 50, 15},
		{"premium effects", 100, 25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			theme := DefaultTheme()
			callbacks := GiftPanelCallbacks{
				GetMyResonance: func() int { return tc.resonance },
			}

			panel := NewGiftPanel(theme, callbacks)
			panel.Show()

			effects := panel.GetAvailableEffects()
			if len(effects) < tc.expected {
				t.Errorf("Expected at least %d effects, got %d", tc.expected, len(effects))
			}
		})
	}
}

func TestGiftPanelLoadsRecipients(t *testing.T) {
	theme := DefaultTheme()
	recipients := []RecipientInfo{
		{NodeID: "abc123", DisplayName: "Node1", IsSurface: true},
		{NodeID: "def456", DisplayName: "Node2", IsSurface: false},
		{NodeID: "ghi789", DisplayName: "Self", IsSurface: true, IsSelf: true},
	}

	callbacks := GiftPanelCallbacks{
		GetRecipients: func() []RecipientInfo { return recipients },
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	loaded := panel.GetRecipients()
	if len(loaded) != 3 {
		t.Errorf("Expected 3 recipients loaded, got %d", len(loaded))
	}
}

func TestGiftPanelFiltersSelf(t *testing.T) {
	theme := DefaultTheme()
	recipients := []RecipientInfo{
		{NodeID: "abc123", DisplayName: "Node1", IsSurface: true},
		{NodeID: "self", DisplayName: "Self", IsSurface: true, IsSelf: true},
	}

	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients:  func() []RecipientInfo { return recipients },
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	// Select first recipient (should be valid since self is filtered).
	panel.SelectRecipient(0)
	selected := panel.GetSelectedRecipient()

	if selected == nil {
		t.Fatal("Expected a selected recipient")
	}
	if selected.NodeID == "self" {
		t.Error("Self should be filtered from selectable recipients")
	}
}

func TestGiftPanelSelectEffect(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 100 },
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	effects := panel.GetAvailableEffects()
	if len(effects) < 2 {
		t.Skip("Not enough effects available")
	}

	// Select second effect.
	panel.SelectEffect(1)
	selected := panel.GetSelectedEffect()

	if selected != effects[1] {
		t.Errorf("Expected effect %d, got %d", effects[1], selected)
	}
}

func TestGiftPanelSelectRecipient(t *testing.T) {
	theme := DefaultTheme()
	recipients := []RecipientInfo{
		{NodeID: "abc123", DisplayName: "Node1", IsSurface: true},
		{NodeID: "def456", DisplayName: "Node2", IsSurface: false},
	}

	callbacks := GiftPanelCallbacks{
		GetRecipients: func() []RecipientInfo { return recipients },
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	panel.SelectRecipient(1)
	selected := panel.GetSelectedRecipient()

	if selected == nil {
		t.Fatal("Expected a selected recipient")
	}
	if selected.NodeID != "def456" {
		t.Errorf("Expected NodeID 'def456', got '%s'", selected.NodeID)
	}
}

func TestGiftPanelConfirmSendSuccess(t *testing.T) {
	theme := DefaultTheme()
	var sentEffect gifts.EffectType
	var sentRecipient string

	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients: func() []RecipientInfo {
			return []RecipientInfo{
				{NodeID: "abc123", DisplayName: "TestNode", IsSurface: true},
			}
		},
		OnSendGift: func(effect gifts.EffectType, recipientID string) error {
			sentEffect = effect
			sentRecipient = recipientID
			return nil
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	panel.SelectEffect(0)
	panel.SelectRecipient(0)

	err := panel.ConfirmSend()
	if err != nil {
		t.Errorf("ConfirmSend failed: %v", err)
	}

	if panel.GetMode() != GiftModeSuccess {
		t.Errorf("Expected success mode, got %d", panel.GetMode())
	}

	if sentEffect == 0 {
		t.Error("OnSendGift was not called")
	}
	if sentRecipient != "abc123" {
		t.Errorf("Expected recipient 'abc123', got '%s'", sentRecipient)
	}
}

func TestGiftPanelConfirmSendError(t *testing.T) {
	theme := DefaultTheme()

	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients: func() []RecipientInfo {
			return []RecipientInfo{
				{NodeID: "abc123", DisplayName: "TestNode", IsSurface: true},
			}
		},
		OnSendGift: func(effect gifts.EffectType, recipientID string) error {
			return errors.New("daily limit exceeded")
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	panel.SelectEffect(0)
	panel.SelectRecipient(0)

	err := panel.ConfirmSend()
	if err == nil {
		t.Error("Expected error from ConfirmSend")
	}

	if panel.GetMode() != GiftModeError {
		t.Errorf("Expected error mode, got %d", panel.GetMode())
	}

	errMsg := panel.GetError()
	if errMsg != "daily limit exceeded" {
		t.Errorf("Expected error message 'daily limit exceeded', got '%s'", errMsg)
	}
}

func TestGiftPanelSetError(t *testing.T) {
	theme := DefaultTheme()
	panel := NewGiftPanel(theme, GiftPanelCallbacks{})

	panel.SetError("test error")

	if panel.GetMode() != GiftModeError {
		t.Errorf("Expected error mode, got %d", panel.GetMode())
	}
	if panel.GetError() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", panel.GetError())
	}
}

func TestGiftPanelRefreshRecipients(t *testing.T) {
	theme := DefaultTheme()
	callCount := 0

	callbacks := GiftPanelCallbacks{
		GetRecipients: func() []RecipientInfo {
			callCount++
			if callCount == 1 {
				return []RecipientInfo{
					{NodeID: "abc", DisplayName: "Node1"},
				}
			}
			return []RecipientInfo{
				{NodeID: "abc", DisplayName: "Node1"},
				{NodeID: "def", DisplayName: "Node2"},
			}
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	if len(panel.GetRecipients()) != 1 {
		t.Error("Expected 1 recipient initially")
	}

	panel.RefreshRecipients()

	if len(panel.GetRecipients()) != 2 {
		t.Error("Expected 2 recipients after refresh")
	}
}

func TestGiftPanelNoCallbacksGraceful(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	// Should not panic with nil callbacks.
	effects := panel.GetAvailableEffects()
	if len(effects) != 0 {
		t.Error("Expected no effects with nil GetMyResonance")
	}

	recipients := panel.GetRecipients()
	if len(recipients) != 0 {
		t.Error("Expected no recipients with nil GetRecipients")
	}
}

func TestGiftPanelModeTransitions(t *testing.T) {
	theme := DefaultTheme()
	panel := NewGiftPanel(theme, GiftPanelCallbacks{})

	// Test SetMode transitions.
	panel.SetMode(GiftModeRecipient)
	if panel.GetMode() != GiftModeRecipient {
		t.Error("SetMode to Recipient failed")
	}

	panel.SetMode(GiftModeConfirm)
	if panel.GetMode() != GiftModeConfirm {
		t.Error("SetMode to Confirm failed")
	}

	panel.SetMode(GiftModeSending)
	if panel.GetMode() != GiftModeSending {
		t.Error("SetMode to Sending failed")
	}
}

func TestGiftPanelInvalidSelections(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients: func() []RecipientInfo {
			return []RecipientInfo{
				{NodeID: "abc123", DisplayName: "TestNode", IsSurface: true},
			}
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	// Select invalid effect index.
	panel.SelectEffect(999)
	selected := panel.GetSelectedEffect()
	// Should remain at previous valid selection or be 0.

	// Select invalid recipient index.
	panel.SelectRecipient(999)
	selectedRecip := panel.GetSelectedRecipient()
	// Should remain at previous valid selection.

	_ = selected
	_ = selectedRecip
	// Test passes if no panic.
}

func TestGiftPanelConfirmNoRecipient(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 50 },
		GetRecipients:  func() []RecipientInfo { return []RecipientInfo{} },
		OnSendGift: func(effect gifts.EffectType, recipientID string) error {
			return nil
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	err := panel.ConfirmSend()
	// Should handle gracefully.
	_ = err

	if panel.GetMode() != GiftModeError {
		t.Error("Expected error mode when no recipients available")
	}
}

func TestGiftPanelConfirmNoEffect(t *testing.T) {
	theme := DefaultTheme()
	callbacks := GiftPanelCallbacks{
		GetMyResonance: func() int { return 0 }, // No effects available.
		GetRecipients: func() []RecipientInfo {
			return []RecipientInfo{
				{NodeID: "abc123", DisplayName: "TestNode"},
			}
		},
		OnSendGift: func(effect gifts.EffectType, recipientID string) error {
			return nil
		},
	}

	panel := NewGiftPanel(theme, callbacks)
	panel.Show()

	err := panel.ConfirmSend()
	_ = err

	if panel.GetMode() != GiftModeError {
		t.Error("Expected error mode when no effects available")
	}
}

func TestGiftSentEvent(t *testing.T) {
	// Test the event struct.
	event := GiftSentEvent{
		Effect:      gifts.EffectSoftGlowPulse,
		RecipientID: "abc123",
	}

	if event.Effect != gifts.EffectSoftGlowPulse {
		t.Error("Effect not set correctly")
	}
	if event.RecipientID != "abc123" {
		t.Error("RecipientID not set correctly")
	}
}
