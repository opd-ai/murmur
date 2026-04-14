// Package ui - Phantom Gift sending interface panel.
// Per ROADMAP.md line 520: "UI: Gift sending panel — select gift type,
// choose recipient, confirm send".
// Per ANONYMOUS_GAME_MECHANICS.md: Phantom Gifts are one-way gestures of
// generosity from a Specter to any node (Surface or Anonymous).
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
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
	OnSendGift func(effect mechanics.EffectType, recipientID string) error

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
	availableEffects []mechanics.EffectType
	selectedEffect   int // Index into availableEffects.

	// Recipient selection.
	recipients        []RecipientInfo
	selectedRecipient int // Index into recipients.
	recipientScroll   int // Scroll offset for recipient list.

	// Dimensions (set in Draw).
	panelX, panelY int
	panelW, panelH int
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
		catalog := mechanics.GiftCatalog{}
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

// Update handles input and state updates.
func (p *GiftPanel) Update() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return
	}

	// Handle escape key to go back or close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.handleEscape()
		return
	}

	// Handle input based on mode.
	switch p.mode {
	case GiftModeEffectSelect:
		p.updateEffectSelect()
	case GiftModeRecipient:
		p.updateRecipientSelect()
	case GiftModeConfirm:
		p.updateConfirm()
	case GiftModeSuccess, GiftModeError:
		p.updateResult()
	}
}

// handleEscape handles escape key presses.
func (p *GiftPanel) handleEscape() {
	switch p.mode {
	case GiftModeEffectSelect:
		p.visible = false
		if p.callbacks.OnClose != nil {
			p.callbacks.OnClose()
		}
	case GiftModeRecipient:
		p.mode = GiftModeEffectSelect
	case GiftModeConfirm:
		p.mode = GiftModeRecipient
	case GiftModeSuccess, GiftModeError:
		p.mode = GiftModeEffectSelect
		p.errorMsg = ""
		p.successMsg = ""
	}
}

// updateEffectSelect handles input in effect selection mode.
func (p *GiftPanel) updateEffectSelect() {
	if len(p.availableEffects) == 0 {
		return
	}

	// Navigate effects with up/down.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		if p.selectedEffect > 0 {
			p.selectedEffect--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		if p.selectedEffect < len(p.availableEffects)-1 {
			p.selectedEffect++
		}
	}

	// Select effect with Enter.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.mode = GiftModeRecipient
	}
}

// updateRecipientSelect handles input in recipient selection mode.
func (p *GiftPanel) updateRecipientSelect() {
	// Filter out self from recipients.
	validRecipients := p.getValidRecipients()
	if len(validRecipients) == 0 {
		return
	}

	// Navigate recipients with up/down.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		if p.selectedRecipient > 0 {
			p.selectedRecipient--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		if p.selectedRecipient < len(validRecipients)-1 {
			p.selectedRecipient++
		}
	}

	// Scroll adjustment.
	visibleCount := 8
	if p.selectedRecipient < p.recipientScroll {
		p.recipientScroll = p.selectedRecipient
	}
	if p.selectedRecipient >= p.recipientScroll+visibleCount {
		p.recipientScroll = p.selectedRecipient - visibleCount + 1
	}

	// Select recipient with Enter.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.mode = GiftModeConfirm
	}
}

// updateConfirm handles input in confirmation mode.
func (p *GiftPanel) updateConfirm() {
	// Y or Enter to confirm, N to cancel.
	if inpututil.IsKeyJustPressed(ebiten.KeyY) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		p.sendGift()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		p.mode = GiftModeRecipient
	}
}

// updateResult handles input in success/error mode.
func (p *GiftPanel) updateResult() {
	// Any key to return to effect selection.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		p.mode = GiftModeEffectSelect
		p.errorMsg = ""
		p.successMsg = ""
	}
}

// sendGift attempts to send the selected gift.
func (p *GiftPanel) sendGift() {
	if p.callbacks.OnSendGift == nil {
		p.errorMsg = "Gift sending not available"
		p.mode = GiftModeError
		return
	}

	validRecipients := p.getValidRecipients()
	if p.selectedRecipient >= len(validRecipients) {
		p.errorMsg = "Invalid recipient selection"
		p.mode = GiftModeError
		return
	}

	if p.selectedEffect >= len(p.availableEffects) {
		p.errorMsg = "Invalid effect selection"
		p.mode = GiftModeError
		return
	}

	p.mode = GiftModeSending
	effect := p.availableEffects[p.selectedEffect]
	recipient := validRecipients[p.selectedRecipient]

	err := p.callbacks.OnSendGift(effect, recipient.NodeID)
	if err != nil {
		p.errorMsg = err.Error()
		p.mode = GiftModeError
		return
	}

	p.successMsg = fmt.Sprintf("Gift sent to %s!", recipient.DisplayName)
	p.mode = GiftModeSuccess
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

// Draw renders the panel.
func (p *GiftPanel) Draw(dst *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	bounds := dst.Bounds()
	screenW, screenH := bounds.Dx(), bounds.Dy()

	// Center the panel.
	p.panelW = 400
	p.panelH = 500
	p.panelX = (screenW - p.panelW) / 2
	p.panelY = (screenH - p.panelH) / 2

	// Draw panel background.
	vector.DrawFilledRect(dst, float32(p.panelX), float32(p.panelY),
		float32(p.panelW), float32(p.panelH), p.theme.PanelBackground, true)

	// Draw border.
	vector.StrokeRect(dst, float32(p.panelX), float32(p.panelY),
		float32(p.panelW), float32(p.panelH), 2, p.theme.PanelBorder, true)

	// Draw title.
	title := p.getTitleForMode()
	p.drawText(dst, title, p.panelX+p.theme.Padding, p.panelY+p.theme.Padding, p.theme.TextPrimary)

	// Draw content based on mode.
	contentY := p.panelY + p.theme.Padding + 30

	switch p.mode {
	case GiftModeEffectSelect:
		p.drawEffectSelect(dst, contentY)
	case GiftModeRecipient:
		p.drawRecipientSelect(dst, contentY)
	case GiftModeConfirm:
		p.drawConfirm(dst, contentY)
	case GiftModeSending:
		p.drawSending(dst, contentY)
	case GiftModeSuccess:
		p.drawSuccess(dst, contentY)
	case GiftModeError:
		p.drawError(dst, contentY)
	}

	// Draw footer with hints.
	p.drawFooter(dst)
}

// getTitleForMode returns the title for the current mode.
func (p *GiftPanel) getTitleForMode() string {
	switch p.mode {
	case GiftModeEffectSelect:
		return "Send Phantom Gift - Select Effect"
	case GiftModeRecipient:
		return "Send Phantom Gift - Choose Recipient"
	case GiftModeConfirm:
		return "Send Phantom Gift - Confirm"
	case GiftModeSending:
		return "Sending Gift..."
	case GiftModeSuccess:
		return "Gift Sent!"
	case GiftModeError:
		return "Error"
	default:
		return "Phantom Gift"
	}
}

// drawEffectSelect renders the effect selection screen.
func (p *GiftPanel) drawEffectSelect(dst *ebiten.Image, startY int) {
	y := startY

	// Show Resonance and remaining gifts.
	if p.callbacks.GetMyResonance != nil {
		resonance := p.callbacks.GetMyResonance()
		info := fmt.Sprintf("Your Resonance: %d", resonance)
		p.drawText(dst, info, p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
		y += 20
	}

	if p.callbacks.GetRemainingGiftsToday != nil {
		remaining := p.callbacks.GetRemainingGiftsToday()
		info := fmt.Sprintf("Gifts remaining today: %d/%d", remaining, mechanics.MaxGiftsPerDay)
		p.drawText(dst, info, p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
		y += 25
	}

	// Check if any effects available.
	if len(p.availableEffects) == 0 {
		p.drawText(dst, "Resonance too low for gifts.", p.panelX+p.theme.Padding, y, p.theme.TextError)
		p.drawText(dst, "Reach Resonance 25 to unlock!", p.panelX+p.theme.Padding, y+20, p.theme.TextSecondary)
		return
	}

	// Draw effect list.
	p.drawText(dst, "Available Effects:", p.panelX+p.theme.Padding, y, p.theme.TextPrimary)
	y += 25

	visibleCount := 10
	for i := 0; i < visibleCount && i < len(p.availableEffects); i++ {
		effect := p.availableEffects[i]
		isSelected := i == p.selectedEffect

		// Highlight selected.
		if isSelected {
			vector.DrawFilledRect(dst,
				float32(p.panelX+p.theme.Padding-2), float32(y-2),
				float32(p.panelW-p.theme.Padding*2+4), 22,
				p.theme.Selection, true)
		}

		// Effect name with tier indicator.
		tier := mechanics.RequiredResonance(effect)
		tierStr := p.tierString(tier)
		name := mechanics.EffectName(effect)
		text := fmt.Sprintf("%s [%s]", name, tierStr)

		textColor := p.theme.TextPrimary
		if isSelected {
			textColor = p.theme.AccentPrimary
		}
		p.drawText(dst, text, p.panelX+p.theme.Padding+10, y, textColor)
		y += 22
	}
}

// tierString returns a tier indicator string.
func (p *GiftPanel) tierString(tier int) string {
	switch tier {
	case mechanics.GiftTierBasic:
		return "Basic"
	case mechanics.GiftTierExpanded:
		return "Expanded"
	case mechanics.GiftTierPremium:
		return "Premium"
	default:
		return "Unknown"
	}
}

// drawRecipientSelect renders the recipient selection screen.
func (p *GiftPanel) drawRecipientSelect(dst *ebiten.Image, startY int) {
	y := startY

	validRecipients := p.getValidRecipients()

	// Show selected effect.
	if p.selectedEffect < len(p.availableEffects) {
		effect := p.availableEffects[p.selectedEffect]
		effectName := mechanics.EffectName(effect)
		p.drawText(dst, fmt.Sprintf("Effect: %s", effectName), p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
		y += 25
	}

	if len(validRecipients) == 0 {
		p.drawText(dst, "No recipients available.", p.panelX+p.theme.Padding, y, p.theme.TextError)
		p.drawText(dst, "Connect with more nodes to gift!", p.panelX+p.theme.Padding, y+20, p.theme.TextSecondary)
		return
	}

	p.drawText(dst, "Select Recipient:", p.panelX+p.theme.Padding, y, p.theme.TextPrimary)
	y += 25

	// Draw recipient list with scroll.
	visibleCount := 8
	for i := p.recipientScroll; i < p.recipientScroll+visibleCount && i < len(validRecipients); i++ {
		r := validRecipients[i]
		isSelected := i == p.selectedRecipient

		// Highlight selected.
		if isSelected {
			vector.DrawFilledRect(dst,
				float32(p.panelX+p.theme.Padding-2), float32(y-2),
				float32(p.panelW-p.theme.Padding*2+4), 22,
				p.theme.Selection, true)
		}

		// Recipient info.
		layerStr := "Specter"
		if r.IsSurface {
			layerStr = "Surface"
		}
		text := fmt.Sprintf("%s (%s)", r.DisplayName, layerStr)

		textColor := p.theme.TextPrimary
		if isSelected {
			textColor = p.theme.AccentPrimary
		}
		p.drawText(dst, text, p.panelX+p.theme.Padding+10, y, textColor)
		y += 22
	}

	// Scroll indicator.
	if len(validRecipients) > visibleCount {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)",
			p.recipientScroll+1,
			min(p.recipientScroll+visibleCount, len(validRecipients)),
			len(validRecipients))
		p.drawText(dst, scrollInfo, p.panelX+p.theme.Padding, y+5, p.theme.TextPlaceholder)
	}
}

// drawConfirm renders the confirmation screen.
func (p *GiftPanel) drawConfirm(dst *ebiten.Image, startY int) {
	y := startY

	validRecipients := p.getValidRecipients()

	p.drawText(dst, "Confirm Gift:", p.panelX+p.theme.Padding, y, p.theme.TextPrimary)
	y += 30

	// Effect.
	if p.selectedEffect < len(p.availableEffects) {
		effect := p.availableEffects[p.selectedEffect]
		p.drawText(dst, fmt.Sprintf("Effect: %s", mechanics.EffectName(effect)),
			p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
		y += 25
	}

	// Recipient.
	if p.selectedRecipient < len(validRecipients) {
		r := validRecipients[p.selectedRecipient]
		p.drawText(dst, fmt.Sprintf("To: %s", r.DisplayName),
			p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
		y += 25
	}

	// Gift duration.
	p.drawText(dst, fmt.Sprintf("Duration: %d days", int(mechanics.GiftDuration.Hours()/24)),
		p.panelX+p.theme.Padding, y, p.theme.TextSecondary)
	y += 40

	// Confirmation prompt.
	p.drawText(dst, "Send this gift? (Y/N)", p.panelX+p.theme.Padding, y, p.theme.AccentPrimary)
}

// drawSending renders the sending in progress screen.
func (p *GiftPanel) drawSending(dst *ebiten.Image, startY int) {
	p.drawText(dst, "Sending gift...", p.panelX+p.theme.Padding, startY, p.theme.TextSecondary)
}

// drawSuccess renders the success screen.
func (p *GiftPanel) drawSuccess(dst *ebiten.Image, startY int) {
	p.drawText(dst, p.successMsg, p.panelX+p.theme.Padding, startY, p.theme.Success)
	p.drawText(dst, "The recipient will see your gift effect for 7 days.",
		p.panelX+p.theme.Padding, startY+25, p.theme.TextSecondary)
	p.drawText(dst, "Press Enter to continue...",
		p.panelX+p.theme.Padding, startY+60, p.theme.TextPlaceholder)
}

// drawError renders the error screen.
func (p *GiftPanel) drawError(dst *ebiten.Image, startY int) {
	p.drawText(dst, "Error:", p.panelX+p.theme.Padding, startY, p.theme.TextError)
	p.drawText(dst, p.errorMsg, p.panelX+p.theme.Padding, startY+25, p.theme.TextSecondary)
	p.drawText(dst, "Press Enter to try again...",
		p.panelX+p.theme.Padding, startY+60, p.theme.TextPlaceholder)
}

// drawFooter renders the footer with navigation hints.
func (p *GiftPanel) drawFooter(dst *ebiten.Image) {
	y := p.panelY + p.panelH - p.theme.Padding - 15

	var hint string
	switch p.mode {
	case GiftModeEffectSelect:
		hint = "↑↓: Select | Enter: Next | Esc: Close"
	case GiftModeRecipient:
		hint = "↑↓: Select | Enter: Next | Esc: Back"
	case GiftModeConfirm:
		hint = "Y: Send | N: Cancel | Esc: Back"
	case GiftModeSuccess, GiftModeError:
		hint = "Enter: Continue | Esc: Back"
	default:
		hint = "Esc: Back"
	}

	p.drawText(dst, hint, p.panelX+p.theme.Padding, y, p.theme.TextPlaceholder)
}

// drawText draws text at the specified position.
// Note: In a real implementation, this would use text rendering.
// For now, we use a placeholder that draws a colored rectangle.
func (p *GiftPanel) drawText(dst *ebiten.Image, text string, x, y int, col color.RGBA) {
	// Placeholder: In production, use ebiten/v2/text/v2.
	// For now, draw small rectangles to indicate text position.
	textWidth := len(text) * 7
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(textWidth), 14, col, true)
}

// GetSelectedEffect returns the currently selected effect.
func (p *GiftPanel) GetSelectedEffect() mechanics.EffectType {
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

// min returns the minimum of two integers.
func giftMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GiftSentEvent represents a gift that was sent (for event bus).
type GiftSentEvent struct {
	Effect      mechanics.EffectType
	RecipientID string
	SentAt      time.Time
}
