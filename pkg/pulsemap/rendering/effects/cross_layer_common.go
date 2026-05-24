// Package effects - Shared cross-layer logic (no build tags).
// This file contains logic common to both cross_layer.go (!test) and cross_layer_stub.go (test).

package effects

import (
	"time"
)

// syncGiftsImpl implements the gift synchronization logic.
// Shared between cross_layer.go and cross_layer_stub.go.
func (b *CrossLayerGiftBridge) syncGiftsImpl() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Rate limit updates.
	now := time.Now()
	if now.Sub(b.lastUpdate) < b.updateMinimum {
		return
	}
	b.lastUpdate = now

	if b.giftStore == nil || b.renderer == nil {
		return
	}

	// Get all active recipients from the gift store.
	// Per mechanics package, gifts are indexed by recipient.
	activeRecipients := b.giftStore.GetAllActiveRecipients()

	// Clear recipients that no longer have active gifts.
	for recipientHex := range b.visibleRecipients {
		if !b.contains(activeRecipients, recipientHex) {
			b.renderer.SetActiveGifts(recipientHex, nil)
			delete(b.visibleRecipients, recipientHex)
		}
	}

	// Update effects for all active recipients.
	for _, recipientHex := range activeRecipients {
		gifts := b.giftStore.GetGiftsByRecipientHex(recipientHex)
		b.renderer.SetActiveGifts(recipientHex, gifts)
		b.visibleRecipients[recipientHex] = true
	}
}
