// Package effects provides cross-layer visibility for Phantom Gifts.
// This is the test stub for testing without graphics.
//
//go:build test
// +build test

package effects

import (
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

// CrossLayerGiftBridge synchronizes Phantom Gift effects between the Anonymous
// Layer (mechanics) and the Surface Layer (Pulse Map rendering).
type CrossLayerGiftBridge struct {
	mu sync.RWMutex

	// Source: Anonymous Layer gift store.
	giftStore *gifts.GiftStore

	// Target: Surface Layer gift renderer.
	renderer *GiftRenderer

	// Tracking for visible recipients.
	visibleRecipients map[string]bool

	// Update interval.
	lastUpdate    time.Time
	updateMinimum time.Duration
}

// NewCrossLayerGiftBridge creates a new bridge between Anonymous and Surface layers.
func NewCrossLayerGiftBridge(giftStore *gifts.GiftStore, renderer *GiftRenderer) *CrossLayerGiftBridge {
	return &CrossLayerGiftBridge{
		giftStore:         giftStore,
		renderer:          renderer,
		visibleRecipients: make(map[string]bool),
		updateMinimum:     time.Second / 4,
	}
}

// SyncGifts synchronizes gift effects from the Anonymous Layer to the Surface Layer.
func (b *CrossLayerGiftBridge) SyncGifts() {
	b.syncGiftsImpl()
}

// syncGiftsImpl implements the gift synchronization logic.
// Shared between cross_layer.go and cross_layer_stub.go.
func (b *CrossLayerGiftBridge) syncGiftsImpl() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Sub(b.lastUpdate) < b.updateMinimum {
		return
	}
	b.lastUpdate = now

	if b.giftStore == nil || b.renderer == nil {
		return
	}

	activeRecipients := b.giftStore.GetAllActiveRecipients()

	for recipientHex := range b.visibleRecipients {
		if !b.contains(activeRecipients, recipientHex) {
			b.renderer.SetActiveGifts(recipientHex, nil)
			delete(b.visibleRecipients, recipientHex)
		}
	}

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
	b.lastUpdate = time.Time{}
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

// OnGiftReceived is called when a new gift is received.
func (b *CrossLayerGiftBridge) OnGiftReceived(recipientHex string, gift *gifts.Gift) {
	if gift == nil || gift.IsExpired() {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.renderer == nil || b.giftStore == nil {
		return
	}

	gifts := b.giftStore.GetGiftsByRecipientHex(recipientHex)
	b.renderer.SetActiveGifts(recipientHex, gifts)
	b.visibleRecipients[recipientHex] = true
}

// OnGiftExpired is called when a gift expires.
func (b *CrossLayerGiftBridge) OnGiftExpired(recipientHex string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.renderer == nil || b.giftStore == nil {
		return
	}

	gifts := b.giftStore.GetGiftsByRecipientHex(recipientHex)
	if len(gifts) == 0 {
		b.renderer.SetActiveGifts(recipientHex, nil)
		delete(b.visibleRecipients, recipientHex)
	} else {
		b.renderer.SetActiveGifts(recipientHex, gifts)
	}
}

// GiftVisibilityEvent represents a cross-layer visibility event.
type GiftVisibilityEvent struct {
	RecipientHex string
	EffectType   gifts.EffectType
	Visible      bool
	Timestamp    time.Time
}
