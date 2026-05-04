// Package effects provides cross-layer visibility for Phantom Gifts.
// Per ROADMAP.md line 521: "Cross-layer visibility — Surface nodes see gift effects
// from Anonymous Layer".
// Per ANONYMOUS_GAME_MECHANICS.md: Gifts from Specters (Anonymous Layer) appear as
// visual effects on recipient nodes visible in the Surface Layer Pulse Map.
//

package effects

import (
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

// CrossLayerGiftBridge synchronizes Phantom Gift effects between the Anonymous
// Layer (mechanics) and the Surface Layer (Pulse Map rendering).
// Per DESIGN_DOCUMENT.md, anonymous activity should be visible to Surface users
// through overlay effects on the Pulse Map.
type CrossLayerGiftBridge struct {
	mu sync.RWMutex

	// Source: Anonymous Layer gift store.
	giftStore *gifts.GiftStore

	// Target: Surface Layer gift renderer.
	renderer *GiftRenderer

	// Tracking for visible recipients (Surface nodes that have received gifts).
	visibleRecipients map[string]bool // Key: recipient hex pubkey.

	// Update interval to avoid excessive synchronization.
	lastUpdate    time.Time
	updateMinimum time.Duration
}

// NewCrossLayerGiftBridge creates a new bridge between Anonymous and Surface layers.
func NewCrossLayerGiftBridge(giftStore *gifts.GiftStore, renderer *GiftRenderer) *CrossLayerGiftBridge {
	return &CrossLayerGiftBridge{
		giftStore:         giftStore,
		renderer:          renderer,
		visibleRecipients: make(map[string]bool),
		updateMinimum:     time.Second / 4, // Max 4 updates per second.
	}
}

// SyncGifts synchronizes gift effects from the Anonymous Layer to the Surface Layer.
// This should be called periodically (e.g., in the Pulse Map update loop).
func (b *CrossLayerGiftBridge) SyncGifts() {
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

// contains checks if a slice contains a string.
func (b *CrossLayerGiftBridge) contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// GetVisibleRecipientCount returns the number of Surface nodes showing gift effects.
func (b *CrossLayerGiftBridge) GetVisibleRecipientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.visibleRecipients)
}

// IsRecipientVisible returns true if the node is showing gift effects.
func (b *CrossLayerGiftBridge) IsRecipientVisible(recipientHex string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.visibleRecipients[recipientHex]
}

// ForceSync forces an immediate synchronization regardless of rate limiting.
func (b *CrossLayerGiftBridge) ForceSync() {
	b.mu.Lock()
	b.lastUpdate = time.Time{} // Reset timer.
	b.mu.Unlock()
	b.SyncGifts()
}

// SetUpdateInterval sets the minimum interval between sync updates.
func (b *CrossLayerGiftBridge) SetUpdateInterval(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.updateMinimum = d
}

// Clear removes all tracked effects.
func (b *CrossLayerGiftBridge) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for recipientHex := range b.visibleRecipients {
		if b.renderer != nil {
			b.renderer.SetActiveGifts(recipientHex, nil)
		}
	}
	b.visibleRecipients = make(map[string]bool)
}

// GetGiftStore returns the associated gift store.
func (b *CrossLayerGiftBridge) GetGiftStore() *gifts.GiftStore {
	return b.giftStore
}

// GetRenderer returns the associated gift renderer.
func (b *CrossLayerGiftBridge) GetRenderer() *GiftRenderer {
	return b.renderer
}

// OnGiftReceived is called when a new gift is received to trigger immediate sync.
// This can be connected to the event bus for reactive updates.
func (b *CrossLayerGiftBridge) OnGiftReceived(recipientHex string, gift *gifts.Gift) {
	if gift == nil || gift.IsExpired() {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.renderer == nil || b.giftStore == nil {
		return
	}

	// Update just this recipient immediately.
	gifts := b.giftStore.GetGiftsByRecipientHex(recipientHex)
	b.renderer.SetActiveGifts(recipientHex, gifts)
	b.visibleRecipients[recipientHex] = true
}

// OnGiftExpired is called when a gift expires to remove its effect.
func (b *CrossLayerGiftBridge) OnGiftExpired(recipientHex string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.renderer == nil || b.giftStore == nil {
		return
	}

	// Refresh gifts for this recipient.
	gifts := b.giftStore.GetGiftsByRecipientHex(recipientHex)
	if len(gifts) == 0 {
		b.renderer.SetActiveGifts(recipientHex, nil)
		delete(b.visibleRecipients, recipientHex)
	} else {
		b.renderer.SetActiveGifts(recipientHex, gifts)
	}
}

// GiftVisibilityEvent represents a cross-layer visibility event.
// Used for event bus integration.
type GiftVisibilityEvent struct {
	RecipientHex string
	EffectType   gifts.EffectType
	Visible      bool
	Timestamp    time.Time
}
