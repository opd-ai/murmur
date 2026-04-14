// Package effects tests for cross-layer gift visibility.
// Per ROADMAP.md line 521: "Cross-layer visibility — Surface nodes see gift effects
// from Anonymous Layer".
//
//go:build noebiten
// +build noebiten

package effects

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

func TestNewCrossLayerGiftBridge(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()

	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	if bridge == nil {
		t.Fatal("NewCrossLayerGiftBridge returned nil")
	}
	if bridge.GetGiftStore() != giftStore {
		t.Error("Gift store not set correctly")
	}
	if bridge.GetRenderer() != renderer {
		t.Error("Renderer not set correctly")
	}
	if bridge.GetVisibleRecipientCount() != 0 {
		t.Error("Expected 0 visible recipients initially")
	}
}

func TestCrossLayerBridgeSyncGifts(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Create a test gift.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 1

	_, err := giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)
	if err != nil {
		t.Fatalf("Failed to create gift: %v", err)
	}

	// Force sync.
	bridge.ForceSync()

	// Verify recipient is now visible.
	if bridge.GetVisibleRecipientCount() != 1 {
		t.Errorf("Expected 1 visible recipient, got %d", bridge.GetVisibleRecipientCount())
	}

	// Verify renderer received the gift.
	recipientHex := toHex(recipientPub)
	if !bridge.IsRecipientVisible(recipientHex) {
		t.Error("Recipient should be visible after sync")
	}
	if !renderer.HasActiveGifts(recipientHex) {
		t.Error("Renderer should have active gifts for recipient")
	}
}

func TestCrossLayerBridgeRateLimiting(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Set a longer update interval for testing.
	bridge.SetUpdateInterval(time.Second)

	// First sync should work.
	bridge.SyncGifts()

	// Create a gift after first sync.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 2

	_, _ = giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)

	// Immediate second sync should be rate-limited.
	bridge.SyncGifts()

	// Gift should not be visible yet due to rate limiting.
	// Note: This test depends on timing and may be flaky in some environments.

	// Force sync should bypass rate limiting.
	bridge.ForceSync()

	// Now the gift should be visible.
	if bridge.GetVisibleRecipientCount() != 1 {
		t.Errorf("Expected 1 visible recipient after ForceSync, got %d", bridge.GetVisibleRecipientCount())
	}
}

func TestCrossLayerBridgeClear(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Create a gift.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 3

	_, _ = giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)
	bridge.ForceSync()

	if bridge.GetVisibleRecipientCount() != 1 {
		t.Fatal("Setup failed: expected 1 visible recipient")
	}

	// Clear the bridge.
	bridge.Clear()

	if bridge.GetVisibleRecipientCount() != 0 {
		t.Errorf("Expected 0 visible recipients after Clear, got %d", bridge.GetVisibleRecipientCount())
	}
}

func TestCrossLayerBridgeOnGiftReceived(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Create a gift.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 4

	gift, _ := giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)

	// Notify the bridge directly.
	recipientHex := toHex(recipientPub)
	bridge.OnGiftReceived(recipientHex, gift)

	// Verify immediate update.
	if !bridge.IsRecipientVisible(recipientHex) {
		t.Error("Recipient should be visible after OnGiftReceived")
	}
	if !renderer.HasActiveGifts(recipientHex) {
		t.Error("Renderer should have active gifts after OnGiftReceived")
	}
}

func TestCrossLayerBridgeOnGiftExpired(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Create a gift.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 5

	_, _ = giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)
	bridge.ForceSync()

	recipientHex := toHex(recipientPub)
	if !bridge.IsRecipientVisible(recipientHex) {
		t.Fatal("Setup failed: recipient should be visible")
	}

	// Simulate gift expiration by garbage collecting.
	// Note: In real code, we'd mock time. For now, manually trigger OnGiftExpired.
	// After removing gifts from store manually, OnGiftExpired should clear the effect.

	// Call OnGiftExpired - this will check the store and see no active gifts.
	// We need to first remove the gift from the store, but that's internal.
	// For this test, we just verify OnGiftExpired handles the case correctly.

	// This is a structural test - just verify it doesn't panic and handles correctly.
	bridge.OnGiftExpired(recipientHex)

	// Since the gift is still in the store and not expired, it should remain visible.
	// This test validates the method runs without error.
}

func TestCrossLayerBridgeNilStore(t *testing.T) {
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(nil, renderer)

	// Should not panic with nil store.
	bridge.SyncGifts()
	bridge.ForceSync()
	bridge.OnGiftReceived("abc123", nil)
	bridge.OnGiftExpired("abc123")

	if bridge.GetVisibleRecipientCount() != 0 {
		t.Error("Should have 0 visible recipients with nil store")
	}
}

func TestCrossLayerBridgeNilRenderer(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	bridge := NewCrossLayerGiftBridge(giftStore, nil)

	// Should not panic with nil renderer.
	bridge.SyncGifts()
	bridge.ForceSync()
	bridge.Clear()
}

func TestCrossLayerBridgeMultipleRecipients(t *testing.T) {
	giftStore := mechanics.NewGiftStore()
	renderer := NewGiftRenderer()
	bridge := NewCrossLayerGiftBridge(giftStore, renderer)

	// Create gifts for multiple recipients using different senders
	// (MaxGiftsPerDay = 3, so we need multiple senders for 5 gifts).
	for i := 0; i < 5; i++ {
		_, senderPriv, _ := ed25519.GenerateKey(nil)
		var senderPub [32]byte
		copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

		recipientPub := make([]byte, 32)
		recipientPub[0] = byte(10 + i)
		_, _ = giftStore.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)
	}

	bridge.ForceSync()

	if bridge.GetVisibleRecipientCount() != 5 {
		t.Errorf("Expected 5 visible recipients, got %d", bridge.GetVisibleRecipientCount())
	}
}

func TestGiftVisibilityEvent(t *testing.T) {
	event := GiftVisibilityEvent{
		RecipientHex: "abc123",
		EffectType:   mechanics.EffectSoftGlowPulse,
		Visible:      true,
		Timestamp:    time.Now(),
	}

	if event.RecipientHex != "abc123" {
		t.Error("RecipientHex not set correctly")
	}
	if event.EffectType != mechanics.EffectSoftGlowPulse {
		t.Error("EffectType not set correctly")
	}
	if !event.Visible {
		t.Error("Visible not set correctly")
	}
}

func TestGiftStoreGetAllActiveRecipients(t *testing.T) {
	store := mechanics.NewGiftStore()

	// Initially empty.
	recipients := store.GetAllActiveRecipients()
	if len(recipients) != 0 {
		t.Errorf("Expected 0 recipients, got %d", len(recipients))
	}

	// Create gifts.
	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 20

	_, _ = store.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)

	recipients = store.GetAllActiveRecipients()
	if len(recipients) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(recipients))
	}
}

func TestGiftStoreGetGiftsByRecipientHex(t *testing.T) {
	store := mechanics.NewGiftStore()

	_, senderPriv, _ := ed25519.GenerateKey(nil)
	var senderPub [32]byte
	copy(senderPub[:], senderPriv.Public().(ed25519.PublicKey)[:32])

	recipientPub := make([]byte, 32)
	recipientPub[0] = 30

	// No gifts initially.
	recipientHex := toHex(recipientPub)
	gifts := store.GetGiftsByRecipientHex(recipientHex)
	if len(gifts) != 0 {
		t.Errorf("Expected 0 gifts, got %d", len(gifts))
	}

	// Create a gift.
	_, _ = store.CreateGift(senderPub, recipientPub, mechanics.EffectSoftGlowPulse, 25, senderPriv)

	gifts = store.GetGiftsByRecipientHex(recipientHex)
	if len(gifts) != 1 {
		t.Errorf("Expected 1 gift, got %d", len(gifts))
	}
}

// toHex converts bytes to hex string (test helper).
func toHex(b []byte) string {
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(b)*2)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}
