// Package ui - Quick-Action Radial Menu for context-sensitive node actions.
// Per ROADMAP.md line 657-663: "Quick-Action Radial Menu — right-click/long-press
// context menu with options: Compose Wave to node, Send Phantom Gift, Place Specter Mark,
// Send Whisper, Join active mini-game, View node detail".
//
// The radial menu appears centered on the selected node with actions arranged
// in a circle. Each action is represented by an icon and label. The menu is
// triggered by right-click (desktop) or long-press (mobile).

//go:build !test
// +build !test

package ui

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// RadialMenuAction represents an action available in the radial menu.
type RadialMenuAction uint8

const (
	// ActionComposeWave opens the Wave composition panel addressed to the selected node.
	ActionComposeWave RadialMenuAction = iota
	// ActionSendGift opens the Phantom Gift sending panel for the selected node.
	ActionSendGift
	// ActionPlaceMark opens the Specter Mark placement panel for the selected node.
	ActionPlaceMark
	// ActionSendWhisper opens the Whisper Chain composition for the selected node.
	ActionSendWhisper
	// ActionJoinGame joins an active mini-game associated with the selected node.
	ActionJoinGame
	// ActionViewDetail opens the node detail panel.
	ActionViewDetail
)

// RadialMenuCallbacks provides callbacks for radial menu actions.
type RadialMenuCallbacks struct {
	// OnAction is called when a menu action is selected.
	// nodeID is the public key of the node the menu was opened on.
	OnAction func(action RadialMenuAction, nodeID string)

	// IsActionAvailable returns true if the given action is available for the node.
	// Used to filter which actions are shown (e.g., hide "Send Gift" if insufficient Resonance).
	IsActionAvailable func(action RadialMenuAction, nodeID string) bool
}

// RadialMenuItem represents a single menu item.
type RadialMenuItem struct {
	Action   RadialMenuAction
	Label    string
	IconCode rune // Unicode icon character
}

// RadialMenu provides a circular context menu for node actions.
type RadialMenu struct {
	mu sync.RWMutex

	// State.
	visible   bool
	theme     Theme
	callbacks RadialMenuCallbacks

	// Position and target node.
	centerX, centerY float64 // Screen coordinates of menu center (node position).
	nodeID           string  // Public key of node the menu is for.

	// Menu items (filtered by availability).
	items []RadialMenuItem

	// Interaction state.
	hoveredIndex int // Index of hovered item, -1 if none.

	// Animation.
	animTime float64 // Time since menu opened, for fade-in animation.
}

// Menu geometry constants.
const (
	radialMenuRadius      = 100.0 // Distance from center to item center.
	radialMenuItemRadius  = 35.0  // Radius of each item circle.
	radialMenuInnerRadius = 20.0  // Radius of center circle (cancel zone).
	radialMenuFadeInTime  = 0.15  // Fade-in animation duration in seconds.
	radialMenuIconSize    = 20.0  // Icon font size.
	radialMenuLabelOffset = 15.0  // Distance from item center to label.
)

// allMenuItems defines all possible menu actions with labels and icons.
var allMenuItems = []RadialMenuItem{
	{ActionComposeWave, "Compose", '✉'}, // U+2709 Envelope
	{ActionSendGift, "Gift", '🎁'},       // U+1F381 Wrapped Gift
	{ActionPlaceMark, "Mark", '★'},      // U+2605 Black Star
	{ActionSendWhisper, "Whisper", '💬'}, // U+1F4AC Speech Balloon
	{ActionJoinGame, "Game", '🎮'},       // U+1F3AE Video Game
	{ActionViewDetail, "Detail", 'ℹ'},   // U+2139 Information Source
}

// NewRadialMenu creates a new radial menu.
func NewRadialMenu(theme Theme, callbacks RadialMenuCallbacks) *RadialMenu {
	return &RadialMenu{
		theme:        theme,
		callbacks:    callbacks,
		hoveredIndex: -1,
	}
}

// Show displays the radial menu at the given screen position for the given node.
func (r *RadialMenu) Show(screenX, screenY float64, nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.visible = true
	r.centerX = screenX
	r.centerY = screenY
	r.nodeID = nodeID
	r.hoveredIndex = -1
	r.animTime = 0

	// Filter items by availability.
	r.items = nil
	for _, item := range allMenuItems {
		if r.callbacks.IsActionAvailable == nil || r.callbacks.IsActionAvailable(item.Action, nodeID) {
			r.items = append(r.items, item)
		}
	}
}

// Hide hides the radial menu.
func (r *RadialMenu) Hide() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.visible = false
	r.hoveredIndex = -1
}

// Visible returns true if the menu is currently shown.
func (r *RadialMenu) Visible() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.visible
}

// Toggle toggles menu visibility (not typically used for radial menus).
func (r *RadialMenu) Toggle() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.visible = !r.visible
	if !r.visible {
		r.hoveredIndex = -1
	}
}

// Update handles input and updates menu state.
// Returns true if the menu consumed the input.
func (r *RadialMenu) Update() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.visible {
		return false
	}

	r.updateAnimation()
	fx, fy := r.getCursorPosition()
	dist := r.distanceFromCenter(fx, fy)

	if r.handleCenterZoneInput(dist) {
		return true
	}

	r.updateHoveredItem(fx, fy)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return r.handleClick()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		r.visible = false
		return true
	}

	return true
}

// Draw renders the radial menu.
func (r *RadialMenu) Draw(screen *ebiten.Image) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.visible {
		return
	}

	// Calculate fade-in alpha.
	alpha := r.animTime / radialMenuFadeInTime
	if alpha > 1.0 {
		alpha = 1.0
	}

	// Draw center circle (cancel zone).
	centerColor := r.theme.PanelBackground
	centerColor.A = uint8(float64(centerColor.A) * alpha * 0.8)
	vector.DrawFilledCircle(screen, float32(r.centerX), float32(r.centerY), radialMenuInnerRadius, centerColor, true)

	// Draw border around center.
	borderColor := r.theme.PanelBorder
	borderColor.A = uint8(float64(borderColor.A) * alpha)
	vector.StrokeCircle(screen, float32(r.centerX), float32(r.centerY), radialMenuInnerRadius, 2.0, borderColor, true)

	// Draw menu items.
	for i, item := range r.items {
		r.drawItem(screen, i, item, alpha)
	}

	// Draw connecting lines from center to items.
	lineColor := r.theme.PanelBorder
	lineColor.A = uint8(float64(lineColor.A) * alpha * 0.3)
	for i := range r.items {
		itemAngle := r.itemAngle(i)
		itemX := r.centerX + radialMenuRadius*math.Cos(itemAngle)
		itemY := r.centerY + radialMenuRadius*math.Sin(itemAngle)
		vector.StrokeLine(screen, float32(r.centerX), float32(r.centerY), float32(itemX), float32(itemY), 1.0, lineColor, true)
	}
}

// drawItem renders a single menu item.
func (r *RadialMenu) drawItem(screen *ebiten.Image, index int, item RadialMenuItem, alpha float64) {
	// Calculate item position.
	angle := r.itemAngle(index)
	itemX := r.centerX + radialMenuRadius*math.Cos(angle)
	itemY := r.centerY + radialMenuRadius*math.Sin(angle)

	// Determine if hovered.
	isHovered := (r.hoveredIndex == index)

	// Item background color.
	bgColor := r.theme.ButtonBackground
	if isHovered {
		bgColor = r.theme.ButtonHover
	}
	bgColor.A = uint8(float64(bgColor.A) * alpha)

	// Draw item circle.
	vector.DrawFilledCircle(screen, float32(itemX), float32(itemY), radialMenuItemRadius, bgColor, true)

	// Draw item border.
	borderColor := r.theme.PanelBorder
	if isHovered {
		borderColor = r.theme.AccentPrimary
	}
	borderColor.A = uint8(float64(borderColor.A) * alpha)
	vector.StrokeCircle(screen, float32(itemX), float32(itemY), radialMenuItemRadius, 2.0, borderColor, true)

	// Draw icon (simplified - actual implementation would use text rendering).
	// For now, we draw a small circle to represent the icon.
	iconColor := r.theme.TextPrimary
	if isHovered {
		iconColor = r.theme.AccentPrimary
	}
	iconColor.A = uint8(float64(iconColor.A) * alpha)
	vector.DrawFilledCircle(screen, float32(itemX), float32(itemY), 8.0, iconColor, true)

	// TODO: Draw actual icon using item.IconCode with text rendering.
	// This requires ebiten/v2/text/v2 which is used elsewhere in the project.
	// For now, the colored dot serves as a placeholder.

	// Draw label below item (simplified).
	// TODO: Use proper text rendering with item.Label.
	// For now, we draw a small rectangle to represent the label area.
	labelY := itemY + radialMenuItemRadius + radialMenuLabelOffset
	labelColor := r.theme.TextSecondary
	labelColor.A = uint8(float64(labelColor.A) * alpha)
	labelW := float32(len(item.Label) * 6) // Rough estimate
	labelH := float32(12)
	vector.DrawFilledRect(screen, float32(itemX)-labelW/2, float32(labelY), labelW, labelH, labelColor, true)
}

// itemAngle calculates the angle (in radians) for the item at the given index.
// Items are arranged in a circle starting from the top (angle = -π/2).
func (r *RadialMenu) itemAngle(index int) float64 {
	if len(r.items) == 0 {
		return 0
	}
	// Start at top (-π/2) and distribute evenly.
	angleStep := 2 * math.Pi / float64(len(r.items))
	return -math.Pi/2 + float64(index)*angleStep
}

// updateAnimation updates the menu's fade-in animation.
func (r *RadialMenu) updateAnimation() {
	r.animTime += 1.0 / 60.0
	if r.animTime > radialMenuFadeInTime {
		r.animTime = radialMenuFadeInTime
	}
}

// getCursorPosition returns the current cursor position as floats.
func (r *RadialMenu) getCursorPosition() (float64, float64) {
	cursorX, cursorY := ebiten.CursorPosition()
	return float64(cursorX), float64(cursorY)
}

// distanceFromCenter calculates distance from menu center to given point.
func (r *RadialMenu) distanceFromCenter(fx, fy float64) float64 {
	dx := fx - r.centerX
	dy := fy - r.centerY
	return math.Sqrt(dx*dx + dy*dy)
}

// handleCenterZoneInput handles input when cursor is in center circle.
// Returns true if input was handled in center zone.
func (r *RadialMenu) handleCenterZoneInput(dist float64) bool {
	if dist < radialMenuInnerRadius {
		r.hoveredIndex = -1
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			r.visible = false
			return true
		}
		return true
	}
	return false
}

// updateHoveredItem updates which menu item is currently hovered.
func (r *RadialMenu) updateHoveredItem(fx, fy float64) {
	r.hoveredIndex = -1
	if len(r.items) == 0 {
		return
	}

	for i := range r.items {
		if r.isItemHovered(i, fx, fy) {
			r.hoveredIndex = i
			break
		}
	}
}

// isItemHovered checks if the given item is under the cursor.
func (r *RadialMenu) isItemHovered(index int, fx, fy float64) bool {
	itemAngle := r.itemAngle(index)
	itemX := r.centerX + radialMenuRadius*math.Cos(itemAngle)
	itemY := r.centerY + radialMenuRadius*math.Sin(itemAngle)
	itemDx := fx - itemX
	itemDy := fy - itemY
	itemDist := math.Sqrt(itemDx*itemDx + itemDy*itemDy)
	return itemDist < radialMenuItemRadius
}

// handleClick processes left-click input on the menu.
func (r *RadialMenu) handleClick() bool {
	if r.hoveredIndex >= 0 {
		action := r.items[r.hoveredIndex].Action
		nodeID := r.nodeID
		r.visible = false
		r.mu.Unlock()
		if r.callbacks.OnAction != nil {
			r.callbacks.OnAction(action, nodeID)
		}
		r.mu.Lock()
		return true
	}
	r.visible = false
	return true
}
