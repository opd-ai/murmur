// Package ui - Phantom Gift sending interface panel stub.
// Per ROADMAP.md line 520: "UI: Gift sending panel — select gift type,
// choose recipient, confirm send".
// This is the noebiten stub for testing without graphics.
//
//go:build test
// +build test

package ui

import (
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/gifts"
)

// GiftPanelMode represents the panel display mode.
type GiftPanelMode uint8

const (
	GiftModeEffectSelect GiftPanelMode = iota // Select gift effect.
	GiftModeRecipient                         // Choose recipient.
	GiftModeConfirm                           // Confirm send.
	GiftModeSending                           // Sending in progress.
	GiftModeSuccess                           // Gift sent successfully.
	GiftModeError                             // Error occurred.
)

// RecipientInfo contains info about a potential gift recipient.
type RecipientInfo struct {
	NodeID      string // Hex-encoded public key.
	DisplayName string // Sigil name or pseudonym.
	IsSurface   bool   // True if Surface identity.
	IsSelf      bool   // True if own identity (prevent self-gifting).
}

// GiftPanelCallbacks provides callbacks for gift panel actions.
type GiftPanelCallbacks struct {
	// OnSendGift is called when user confirms sending a gift.
	// Returns error if send fails, nil on success.
	OnSendGift func(effect gifts.EffectType, recipientID string) error

	// OnClose is called when user closes the panel.
	OnClose func()

	// GetMyResonance returns the user's current Resonance level.
	GetMyResonance func() int

	// GetRecipients returns available recipients for gifting.
	GetRecipients func() []RecipientInfo

	// GetRemainingGiftsToday returns how many gifts user can still send today.
	GetRemainingGiftsToday func() int
}

// GiftPanel provides the UI for sending Phantom Gifts.
// Per ANONYMOUS_GAME_MECHANICS.md, gifts are tiered by Resonance level.
type GiftPanel struct {
	mu sync.RWMutex

	// State.
	visible    bool
	mode       GiftPanelMode
	theme      Theme
	callbacks  GiftPanelCallbacks
	errorMsg   string
	successMsg string

	// Gift selection.
	availableEffects []gifts.EffectType
	selectedEffect   int // Index into availableEffects.

	// Recipient selection.
	recipients        []RecipientInfo
	selectedRecipient int // Index into recipients.
	recipientScroll   int // Scroll offset for recipient list.
}

// NewGiftPanel creates a new gift sending panel.
func NewGiftPanel(theme Theme, callbacks GiftPanelCallbacks) *GiftPanel {
	return &GiftPanel{
		theme:     theme,
		callbacks: callbacks,
		mode:      GiftModeEffectSelect,
	}
}

// Show makes the panel visible and initializes state.
func (p *GiftPanel) Show() {
	p.initShowState()
}

// initShowState initializes the panel state for Show().
// Shared between gift.go and gift_stub.go.
func (p *GiftPanel) initShowState() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.visible = true
	p.mode = GiftModeEffectSelect
	p.selectedEffect = 0
	p.selectedRecipient = 0
	p.recipientScroll = 0
	p.errorMsg = ""
	p.successMsg = ""

	// Load available effects based on Resonance.
	if p.callbacks.GetMyResonance != nil {
		resonance := p.callbacks.GetMyResonance()
		catalog := gifts.GiftCatalog{}
		p.availableEffects = catalog.AvailableEffects(resonance)
	}

	// Load recipients.
	if p.callbacks.GetRecipients != nil {
		p.recipients = p.callbacks.GetRecipients()
	}
}

// Hide closes the panel.
func (p *GiftPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// IsVisible returns true if the panel is visible.
func (p *GiftPanel) IsVisible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// GetMode returns the current panel mode.
func (p *GiftPanel) GetMode() GiftPanelMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// SetMode sets the panel mode (for testing).
func (p *GiftPanel) SetMode(mode GiftPanelMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
}

// Update handles input and state updates.
func (p *GiftPanel) Update() {
	// Stub: No input handling in noebiten build.
}

// SelectEffect selects an effect by index (for testing).
func (p *GiftPanel) SelectEffect(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index >= 0 && index < len(p.availableEffects) {
		p.selectedEffect = index
	}
}

// SelectRecipient selects a recipient by index (for testing).
func (p *GiftPanel) SelectRecipient(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	validRecipients := p.getValidRecipients()
	if index >= 0 && index < len(validRecipients) {
		p.selectedRecipient = index
	}
}

// ConfirmSend confirms the gift send (for testing).
func (p *GiftPanel) ConfirmSend() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.callbacks.OnSendGift == nil {
		p.errorMsg = "Gift sending not available"
		p.mode = GiftModeError
		return nil
	}

	validRecipients := p.getValidRecipients()
	if p.selectedRecipient >= len(validRecipients) {
		p.errorMsg = "Invalid recipient selection"
		p.mode = GiftModeError
		return nil
	}

	if p.selectedEffect >= len(p.availableEffects) {
		p.errorMsg = "Invalid effect selection"
		p.mode = GiftModeError
		return nil
	}

	effect := p.availableEffects[p.selectedEffect]
	recipient := validRecipients[p.selectedRecipient]

	err := p.callbacks.OnSendGift(effect, recipient.NodeID)
	if err != nil {
		p.errorMsg = err.Error()
		p.mode = GiftModeError
		return err
	}

	p.successMsg = "Gift sent to " + recipient.DisplayName + "!"
	p.mode = GiftModeSuccess
	return nil
}

// getValidRecipients returns recipients excluding self.
func (p *GiftPanel) getValidRecipients() []RecipientInfo {
	valid := make([]RecipientInfo, 0, len(p.recipients))
	for _, r := range p.recipients {
		if !r.IsSelf {
			valid = append(valid, r)
		}
	}
	return valid
}

// GetSelectedEffect returns the currently selected effect.
func (p *GiftPanel) GetSelectedEffect() gifts.EffectType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.selectedEffect < len(p.availableEffects) {
		return p.availableEffects[p.selectedEffect]
	}
	return 0
}

// GetSelectedRecipient returns the currently selected recipient.
func (p *GiftPanel) GetSelectedRecipient() *RecipientInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	validRecipients := p.getValidRecipients()
	if p.selectedRecipient < len(validRecipients) {
		r := validRecipients[p.selectedRecipient]
		return &r
	}
	return nil
}

// SetError sets an error message and switches to error mode.
func (p *GiftPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMsg = msg
	p.mode = GiftModeError
}

// GetError returns the current error message.
func (p *GiftPanel) GetError() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errorMsg
}

// GetSuccessMsg returns the current success message.
func (p *GiftPanel) GetSuccessMsg() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.successMsg
}

// RefreshRecipients reloads the recipient list.
func (p *GiftPanel) RefreshRecipients() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.callbacks.GetRecipients != nil {
		p.recipients = p.callbacks.GetRecipients()
	}
	// Reset selection if out of bounds.
	validRecipients := p.getValidRecipients()
	if p.selectedRecipient >= len(validRecipients) {
		p.selectedRecipient = 0
	}
}

// GetAvailableEffects returns the available effects (for testing).
func (p *GiftPanel) GetAvailableEffects() []gifts.EffectType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.availableEffects
}

// GetRecipients returns the recipients list (for testing).
func (p *GiftPanel) GetRecipients() []RecipientInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.recipients
}

// GiftSentEvent represents a gift that was sent (for event bus).
type GiftSentEvent struct {
	Effect      gifts.EffectType
	RecipientID string
	SentAt      time.Time
}
