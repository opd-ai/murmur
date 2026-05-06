// Package ui - Node Detail Panel for displaying comprehensive node information.
// Per ROADMAP.md line 664-669: "Node Detail Panel — slide-in card with:
// Profile information (display name, Sigil, public key fingerprint),
// Recent Waves list, Connection list, Specter Resonance (for anonymous nodes),
// Interaction options".
//
// The panel slides in from the right side of the screen with smooth animation.
// It shows detailed information about the selected node and provides interaction
// buttons for common actions (compose Wave, send gift, etc.).

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// NodeInfo contains information about a node for display.
type NodeInfo struct {
	// Identity
	PublicKey   string // Hex-encoded public key
	DisplayName string // Display name or pseudonym
	Fingerprint string // Short fingerprint (first 8 chars of hex)
	IsSpecter   bool   // True if anonymous Specter node
	IsSurface   bool   // True if Surface identity node
	IsSelf      bool   // True if this is the user's own node

	// Resonance (for Specters)
	Resonance     int    // Resonance score (0 if not Specter)
	ResonanceRank string // Milestone name (Shade, Wraith, etc.)

	// Connections
	ConnectionCount int      // Number of connections
	Connections     []string // List of connected node display names

	// Recent Waves
	RecentWaves []WaveInfo // Last 10 Waves from this node
}

// WaveInfo contains information about a Wave for display.
type WaveInfo struct {
	Content   string    // Wave text content (truncated if >100 chars)
	Timestamp time.Time // Publication time
	WaveType  string    // "Surface", "Specter", "Veiled", etc.
}

// NodeDetailCallbacks provides callbacks for node detail panel actions.
type NodeDetailCallbacks struct {
	// OnComposeWave is called when user clicks "Compose Wave" button.
	OnComposeWave func(nodeID string)

	// OnSendGift is called when user clicks "Send Gift" button.
	OnSendGift func(nodeID string)

	// OnPlaceMark is called when user clicks "Place Mark" button.
	OnPlaceMark func(nodeID string)

	// OnSendWhisper is called when user clicks "Send Whisper" button.
	OnSendWhisper func(nodeID string)

	// OnViewWave is called when user clicks a Wave in the list.
	OnViewWave func(waveID string)

	// OnClose is called when user closes the panel.
	OnClose func()
}

// NodeDetailPanel displays detailed information about a selected node.
type NodeDetailPanel struct {
	mu sync.RWMutex

	// State
	visible   bool
	theme     Theme
	callbacks NodeDetailCallbacks
	nodeInfo  *NodeInfo

	// Animation
	slideOffset float64 // 0 = fully visible, 1 = fully off-screen
	animTime    float64 // Animation progress time

	// Scroll state
	waveScroll       int // Scroll position for Waves list
	connectionScroll int // Scroll position for Connections list

	// Dimensions (set in Draw)
	panelX, panelY int
	panelW, panelH int
}

// Panel dimensions.
const (
	nodeDetailPanelWidth      = 400  // Panel width in pixels
	nodeDetailPanelHeight     = 600  // Panel height in pixels
	nodeDetailPadding         = 20   // Padding inside panel
	nodeDetailSectionSpacing  = 15   // Spacing between sections
	nodeDetailButtonHeight    = 40   // Height of action buttons
	nodeDetailButtonSpacing   = 10   // Spacing between buttons
	nodeDetailListItemHeight  = 60   // Height of each list item (Wave/Connection)
	nodeDetailMaxVisibleItems = 5    // Max visible list items before scrolling
	nodeDetailSlideSpeed      = 0.15 // Slide animation speed (progress per frame)
)

// NewNodeDetailPanel creates a new node detail panel.
func NewNodeDetailPanel(theme Theme, callbacks NodeDetailCallbacks) *NodeDetailPanel {
	return &NodeDetailPanel{
		theme:       theme,
		callbacks:   callbacks,
		slideOffset: 1.0, // Start fully off-screen
	}
}

// Show displays the panel for the given node.
func (p *NodeDetailPanel) Show(nodeInfo *NodeInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.visible = true
	p.nodeInfo = nodeInfo
	p.slideOffset = 1.0 // Start from off-screen
	p.animTime = 0
	p.waveScroll = 0
	p.connectionScroll = 0
}

// Hide hides the panel.
func (p *NodeDetailPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
	p.slideOffset = 1.0
	p.nodeInfo = nil
}

// Visible returns true if the panel is currently shown.
func (p *NodeDetailPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Toggle toggles panel visibility.
func (p *NodeDetailPanel) Toggle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = !p.visible
	if !p.visible {
		p.nodeInfo = nil
	}
}

// Update handles input and updates panel state.
// Returns true if the panel consumed the input.
func (p *NodeDetailPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	p.updateSlideAnimation()
	p.calculatePanelPosition()
	cursorX, cursorY := ebiten.CursorPosition()
	mouseOverPanel := p.isMouseOverPanel(cursorX, cursorY)

	p.handleScrollInput(mouseOverPanel)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if mouseOverPanel {
			return p.handlePanelClick(cursorY)
		}
		return p.handleOutsideClick()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return p.handleEscapeKey()
	}

	return true
}

// updateSlideAnimation updates the slide-in animation state.
func (p *NodeDetailPanel) updateSlideAnimation() {
	if p.slideOffset > 0 {
		p.slideOffset -= nodeDetailSlideSpeed
		if p.slideOffset < 0 {
			p.slideOffset = 0
		}
	}
	p.animTime += 1.0 / 60.0
}

// calculatePanelPosition calculates panel dimensions and position.
func (p *NodeDetailPanel) calculatePanelPosition() {
	screenW, screenH := ebiten.WindowSize()
	p.panelW = nodeDetailPanelWidth
	p.panelH = nodeDetailPanelHeight
	p.panelX = screenW - p.panelW + int(p.slideOffset*float64(p.panelW))
	p.panelY = (screenH - p.panelH) / 2
}

// isMouseOverPanel checks if the cursor is within the panel bounds.
func (p *NodeDetailPanel) isMouseOverPanel(cursorX, cursorY int) bool {
	return cursorX >= p.panelX && cursorX < p.panelX+p.panelW &&
		cursorY >= p.panelY && cursorY < p.panelY+p.panelH
}

// handleScrollInput processes mouse wheel input for scrolling.
func (p *NodeDetailPanel) handleScrollInput(mouseOverPanel bool) {
	_, wheelY := ebiten.Wheel()
	if !mouseOverPanel || wheelY == 0 || p.nodeInfo == nil {
		return
	}

	p.waveScroll -= int(wheelY * 3)
	if p.waveScroll < 0 {
		p.waveScroll = 0
	}
	maxScroll := len(p.nodeInfo.RecentWaves) - nodeDetailMaxVisibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.waveScroll > maxScroll {
		p.waveScroll = maxScroll
	}
}

// handlePanelClick processes clicks inside the panel.
func (p *NodeDetailPanel) handlePanelClick(cursorY int) bool {
	if p.nodeInfo != nil {
		buttonY := p.panelY + nodeDetailPadding + 100
		if cursorY >= buttonY && cursorY < buttonY+nodeDetailButtonHeight*4+nodeDetailButtonSpacing*3 {
			buttonIndex := (cursorY - buttonY) / (nodeDetailButtonHeight + nodeDetailButtonSpacing)
			p.handleButtonClick(buttonIndex)
		}
	}
	return true
}

// handleOutsideClick processes clicks outside the panel to close it.
func (p *NodeDetailPanel) handleOutsideClick() bool {
	p.visible = false
	if p.callbacks.OnClose != nil {
		p.mu.Unlock()
		p.callbacks.OnClose()
		p.mu.Lock()
	}
	return true
}

// handleEscapeKey processes Escape key to close the panel.
func (p *NodeDetailPanel) handleEscapeKey() bool {
	p.visible = false
	if p.callbacks.OnClose != nil {
		p.mu.Unlock()
		p.callbacks.OnClose()
		p.mu.Lock()
	}
	return true
}

// handleButtonClick handles action button clicks.
func (p *NodeDetailPanel) handleButtonClick(buttonIndex int) {
	if p.nodeInfo == nil {
		return
	}

	nodeID := p.nodeInfo.PublicKey

	p.mu.Unlock()
	defer p.mu.Lock()

	p.dispatchButtonAction(buttonIndex, nodeID)
}

// dispatchButtonAction invokes the appropriate callback for the button index.
func (p *NodeDetailPanel) dispatchButtonAction(buttonIndex int, nodeID [32]byte) {
	switch buttonIndex {
	case 0:
		p.invokeCallback(p.callbacks.OnComposeWave, nodeID)
	case 1:
		p.invokeCallback(p.callbacks.OnSendGift, nodeID)
	case 2:
		p.invokeCallback(p.callbacks.OnPlaceMark, nodeID)
	case 3:
		p.invokeCallback(p.callbacks.OnSendWhisper, nodeID)
	}
}

// invokeCallback calls the provided callback function if it's not nil.
func (p *NodeDetailPanel) invokeCallback(callback func([32]byte), nodeID [32]byte) {
	if callback != nil {
		callback(nodeID)
	}
}

// Draw renders the panel.
func (p *NodeDetailPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible || p.nodeInfo == nil {
		return
	}

	// Draw panel background.
	panelBg := p.theme.PanelBackground
	vector.DrawFilledRect(screen, float32(p.panelX), float32(p.panelY), float32(p.panelW), float32(p.panelH), panelBg, true)

	// Draw panel border.
	vector.StrokeRect(screen, float32(p.panelX), float32(p.panelY), float32(p.panelW), float32(p.panelH), 2.0, p.theme.PanelBorder, true)

	// Draw header section (node name, fingerprint).
	headerY := float32(p.panelY + nodeDetailPadding)
	p.drawHeader(screen, headerY)

	// Draw Resonance section (for Specters).
	resonanceY := headerY + 60
	if p.nodeInfo.IsSpecter {
		p.drawResonance(screen, resonanceY)
		resonanceY += 40
	}

	// Draw action buttons.
	buttonY := resonanceY + 20
	p.drawActionButtons(screen, buttonY)

	// Draw Recent Waves section.
	wavesY := buttonY + float32(4*(nodeDetailButtonHeight+nodeDetailButtonSpacing)+20)
	p.drawRecentWaves(screen, wavesY)

	// Draw Connections section.
	connectionsY := wavesY + float32(nodeDetailMaxVisibleItems*nodeDetailListItemHeight+60)
	p.drawConnections(screen, connectionsY)
}

// drawHeader draws the node name and fingerprint.
func (p *NodeDetailPanel) drawHeader(screen *ebiten.Image, y float32) {
	nameX := float64(p.panelX + nodeDetailPadding)

	// Display name (or fingerprint when name is empty).
	displayName := p.nodeInfo.DisplayName
	if displayName == "" {
		displayName = p.nodeInfo.Fingerprint
	}
	drawUIText(screen, displayName, nameX, float64(y), p.theme.TextPrimary)

	// Fingerprint on the line below.
	fingerprintY := float64(y) + 18
	drawUIText(screen, p.nodeInfo.Fingerprint, nameX, fingerprintY, p.theme.TextSecondary)
}

// drawResonance draws the Resonance score and rank (for Specters).
func (p *NodeDetailPanel) drawResonance(screen *ebiten.Image, y float32) {
	resonanceX := float64(p.panelX + nodeDetailPadding)
	resonanceText := fmt.Sprintf("Resonance: %d (%s)", p.nodeInfo.Resonance, p.nodeInfo.ResonanceRank)
	drawUIText(screen, resonanceText, resonanceX, float64(y), p.theme.AccentSecondary)
}

// drawActionButtons draws the action buttons.
func (p *NodeDetailPanel) drawActionButtons(screen *ebiten.Image, y float32) {
	buttonLabels := []string{"Compose Wave", "Send Gift", "Place Mark", "Send Whisper"}
	buttonX := float32(p.panelX + nodeDetailPadding)
	buttonW := float32(p.panelW - 2*nodeDetailPadding)

	for i, label := range buttonLabels {
		btnY := y + float32(i*(nodeDetailButtonHeight+nodeDetailButtonSpacing))

		// Draw button background.
		vector.DrawFilledRect(screen, buttonX, btnY, buttonW, float32(nodeDetailButtonHeight), p.theme.ButtonBackground, true)

		// Draw button border.
		vector.StrokeRect(screen, buttonX, btnY, buttonW, float32(nodeDetailButtonHeight), 1.0, p.theme.PanelBorder, true)

		// Draw centered button label.
		cx := float64(buttonX) + float64(buttonW)/2
		ty := float64(btnY) + float64(nodeDetailButtonHeight)/2 - 4
		drawUICenteredText(screen, label, cx, ty, p.theme.TextPrimary)
	}
}

// drawRecentWaves draws the Recent Waves list.
func (p *NodeDetailPanel) drawRecentWaves(screen *ebiten.Image, y float32) {
	listX := float32(p.panelX + nodeDetailPadding)
	listW := float32(p.panelW - 2*nodeDetailPadding)

	// Section title.
	drawUIText(screen, "Recent Waves", float64(listX), float64(y), p.theme.TextPrimary)

	// Draw list items.
	listY := y + 22
	visibleWaves := p.nodeInfo.RecentWaves
	if len(visibleWaves) > p.waveScroll {
		visibleWaves = visibleWaves[p.waveScroll:]
	}
	if len(visibleWaves) > nodeDetailMaxVisibleItems {
		visibleWaves = visibleWaves[:nodeDetailMaxVisibleItems]
	}

	for i, wave := range visibleWaves {
		itemY := listY + float32(i*nodeDetailListItemHeight)

		// Item background.
		vector.DrawFilledRect(screen, listX, itemY, listW, float32(nodeDetailListItemHeight-5), p.theme.InputBackground, true)

		// Wave content (truncated to ~40 runes; rune-safe for any Unicode content).
		content := truncateRunes(wave.Content, 40)
		drawUIText(screen, content, float64(listX)+8, float64(itemY)+6, p.theme.TextPrimary)

		// Timestamp on second line.
		timestamp := wave.Timestamp.Format("15:04")
		drawUIText(screen, timestamp, float64(listX)+8, float64(itemY)+20, p.theme.TextSecondary)
	}
}

// drawConnections draws the Connections list.
func (p *NodeDetailPanel) drawConnections(screen *ebiten.Image, y float32) {
	titleX := float64(p.panelX + nodeDetailPadding)

	// Section title.
	drawUIText(screen, "Connections", titleX, float64(y), p.theme.TextPrimary)

	// Connection count on the line below.
	countText := fmt.Sprintf("Connections: %d", p.nodeInfo.ConnectionCount)
	drawUIText(screen, countText, titleX, float64(y)+18, p.theme.TextSecondary)
}
