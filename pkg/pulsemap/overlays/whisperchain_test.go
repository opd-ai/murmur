// Package overlays - Whisper Chain overlay tests.

package overlays

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestWhisperChainOverlay_Creation(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)
	if o == nil {
		t.Fatal("NewWhisperChainOverlay returned nil")
	}

	if !o.IsVisible() {
		t.Error("overlay should be visible by default")
	}

	if o.IndicatorCount() != 0 {
		t.Errorf("indicator count = %d, want 0", o.IndicatorCount())
	}

	if o.GetMinZoomLevel() != 3.0 {
		t.Errorf("min zoom level = %f, want 3.0", o.GetMinZoomLevel())
	}
}

func TestWhisperChainOverlay_AddWhisperDelivery(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	var messageID [32]byte
	rand.Read(messageID[:])

	// Add a whisper delivery.
	o.AddWhisperDelivery(messageID, recipientNode, 100.0, 200.0)

	if o.IndicatorCount() != 1 {
		t.Errorf("indicator count = %d, want 1", o.IndicatorCount())
	}

	indicators := o.GetActiveIndicators()
	if len(indicators) != 1 {
		t.Fatalf("active indicators = %d, want 1", len(indicators))
	}

	ind := indicators[0]
	if ind.NodeID != recipientNode {
		t.Errorf("node ID mismatch")
	}
	if ind.X != 100.0 || ind.Y != 200.0 {
		t.Errorf("position = (%f, %f), want (100, 200)", ind.X, ind.Y)
	}
}

func TestWhisperChainOverlay_RecipientOnly(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	var otherNode [32]byte
	rand.Read(otherNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	var messageID [32]byte
	rand.Read(messageID[:])

	// Try to add indicator for a different node.
	o.AddWhisperDelivery(messageID, otherNode, 100.0, 200.0)

	// Should not add indicator because it's not the recipient node.
	if o.IndicatorCount() != 0 {
		t.Errorf("indicator count = %d, want 0 (non-recipient should not add indicator)", o.IndicatorCount())
	}
}

func TestWhisperChainOverlay_UpdateNodePosition(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	var messageID [32]byte
	rand.Read(messageID[:])

	o.AddWhisperDelivery(messageID, recipientNode, 100.0, 200.0)

	// Update node position.
	o.UpdateNodePosition(recipientNode, 150.0, 250.0)

	indicators := o.GetActiveIndicators()
	if len(indicators) != 1 {
		t.Fatalf("active indicators = %d, want 1", len(indicators))
	}

	ind := indicators[0]
	if ind.X != 150.0 || ind.Y != 250.0 {
		t.Errorf("position = (%f, %f), want (150, 250)", ind.X, ind.Y)
	}
}

func TestWhisperChainOverlay_Expiration(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)
	o.SetPulseDuration(0.1) // 100ms duration for fast test.

	var messageID [32]byte
	rand.Read(messageID[:])

	o.AddWhisperDelivery(messageID, recipientNode, 100.0, 200.0)

	if o.IndicatorCount() != 1 {
		t.Errorf("indicator count = %d, want 1", o.IndicatorCount())
	}

	// Wait for expiration.
	time.Sleep(150 * time.Millisecond)

	// Update should remove expired indicator.
	o.Update(0.2)

	if o.IndicatorCount() != 0 {
		t.Errorf("indicator count = %d, want 0 after expiration", o.IndicatorCount())
	}
}

func TestWhisperChainOverlay_ClearExpired(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)
	o.SetPulseDuration(0.1)

	var messageID1, messageID2 [32]byte
	rand.Read(messageID1[:])
	rand.Read(messageID2[:])

	o.AddWhisperDelivery(messageID1, recipientNode, 100.0, 200.0)

	time.Sleep(150 * time.Millisecond)

	o.AddWhisperDelivery(messageID2, recipientNode, 300.0, 400.0)

	// messageID1 should be expired, messageID2 should be active.
	removed := o.ClearExpired()
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	if o.IndicatorCount() != 1 {
		t.Errorf("indicator count = %d, want 1", o.IndicatorCount())
	}
}

func TestWhisperChainOverlay_ClearAll(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	var messageID1, messageID2 [32]byte
	rand.Read(messageID1[:])
	rand.Read(messageID2[:])

	o.AddWhisperDelivery(messageID1, recipientNode, 100.0, 200.0)
	o.AddWhisperDelivery(messageID2, recipientNode, 300.0, 400.0)

	if o.IndicatorCount() != 2 {
		t.Errorf("indicator count = %d, want 2", o.IndicatorCount())
	}

	o.ClearAll()

	if o.IndicatorCount() != 0 {
		t.Errorf("indicator count = %d, want 0 after ClearAll", o.IndicatorCount())
	}
}

func TestWhisperChainOverlay_Visibility(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	if !o.IsVisible() {
		t.Error("overlay should be visible by default")
	}

	o.SetVisible(false)

	if o.IsVisible() {
		t.Error("overlay should be invisible after SetVisible(false)")
	}

	o.SetVisible(true)

	if !o.IsVisible() {
		t.Error("overlay should be visible after SetVisible(true)")
	}
}

func TestWhisperChainOverlay_SetRecipientNode(t *testing.T) {
	var recipientNode1 [32]byte
	rand.Read(recipientNode1[:])

	var recipientNode2 [32]byte
	rand.Read(recipientNode2[:])

	o := NewWhisperChainOverlay(recipientNode1)

	var messageID [32]byte
	rand.Read(messageID[:])

	// Add delivery for recipientNode1.
	o.AddWhisperDelivery(messageID, recipientNode1, 100.0, 200.0)

	if o.IndicatorCount() != 1 {
		t.Errorf("indicator count = %d, want 1", o.IndicatorCount())
	}

	// Change recipient node.
	o.SetRecipientNode(recipientNode2)

	// Try to add delivery for recipientNode1 (should fail).
	var messageID2 [32]byte
	rand.Read(messageID2[:])
	o.AddWhisperDelivery(messageID2, recipientNode1, 300.0, 400.0)

	// Should still be 1 (only the old one).
	if o.IndicatorCount() != 1 {
		t.Errorf("indicator count = %d, want 1", o.IndicatorCount())
	}

	// Add delivery for recipientNode2 (should succeed).
	var messageID3 [32]byte
	rand.Read(messageID3[:])
	o.AddWhisperDelivery(messageID3, recipientNode2, 500.0, 600.0)

	if o.IndicatorCount() != 2 {
		t.Errorf("indicator count = %d, want 2", o.IndicatorCount())
	}
}

func TestWhisperChainOverlay_MinZoomLevel(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	defaultZoom := o.GetMinZoomLevel()
	if defaultZoom != 3.0 {
		t.Errorf("default min zoom = %f, want 3.0", defaultZoom)
	}

	o.SetMinZoomLevel(5.0)

	if o.GetMinZoomLevel() != 5.0 {
		t.Errorf("min zoom = %f, want 5.0", o.GetMinZoomLevel())
	}
}

func TestWhisperChainOverlay_PulseDuration(t *testing.T) {
	var recipientNode [32]byte
	rand.Read(recipientNode[:])

	o := NewWhisperChainOverlay(recipientNode)

	// Default duration is 1.5 seconds.
	o.SetPulseDuration(2.0)

	var messageID [32]byte
	rand.Read(messageID[:])

	o.AddWhisperDelivery(messageID, recipientNode, 100.0, 200.0)

	indicators := o.GetActiveIndicators()
	if len(indicators) != 1 {
		t.Fatalf("active indicators = %d, want 1", len(indicators))
	}

	if indicators[0].Duration != 2.0 {
		t.Errorf("duration = %f, want 2.0", indicators[0].Duration)
	}
}
