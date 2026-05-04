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

	// Unlock before callback to avoid deadlock.
	p.mu.Unlock()
	defer p.mu.Lock()

	switch buttonIndex {
	case 0: // Compose Wave
		if p.callbacks.OnComposeWave != nil {
			p.callbacks.OnComposeWave(nodeID)
		}
	case 1: // Send Gift
		if p.callbacks.OnSendGift != nil {
			p.callbacks.OnSendGift(nodeID)
		}
	case 2: // Place Mark
		if p.callbacks.OnPlaceMark != nil {
			p.callbacks.OnPlaceMark(nodeID)
		}
	case 3: // Send Whisper
		if p.callbacks.OnSendWhisper != nil {
			p.callbacks.OnSendWhisper(nodeID)
		}
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
	// Draw display name (simplified - actual implementation uses text rendering).
	nameColor := p.theme.TextPrimary
	nameX := float32(p.panelX + nodeDetailPadding)
	vector.DrawFilledRect(screen, nameX, y, 200, 20, nameColor, true)

	// Draw fingerprint below name.
	fingerprintY := y + 30
	fingerprintColor := p.theme.TextSecondary
	vector.DrawFilledRect(screen, nameX, fingerprintY, 150, 15, fingerprintColor, true)

	// TODO: Replace rectangles with actual text rendering using ebiten/v2/text/v2.
	// Display p.nodeInfo.DisplayName and p.nodeInfo.Fingerprint.
}

// drawResonance draws the Resonance score and rank (for Specters).
func (p *NodeDetailPanel) drawResonance(screen *ebiten.Image, y float32) {
	resonanceX := float32(p.panelX + nodeDetailPadding)
	resonanceColor := p.theme.AccentSecondary
	vector.DrawFilledRect(screen, resonanceX, y, 100, 20, resonanceColor, true)

	// TODO: Replace with actual text rendering.
	// Display fmt.Sprintf("Resonance: %d (%s)", p.nodeInfo.Resonance, p.nodeInfo.ResonanceRank)
}

// drawActionButtons draws the action buttons.
func (p *NodeDetailPanel) drawActionButtons(screen *ebiten.Image, y float32) {
	buttonLabels := []string{"Compose Wave", "Send Gift", "Place Mark", "Send Whisper"}
	buttonX := float32(p.panelX + nodeDetailPadding)
	buttonW := float32(p.panelW - 2*nodeDetailPadding)

	for i, label := range buttonLabels {
		btnY := y + float32(i*(nodeDetailButtonHeight+nodeDetailButtonSpacing))
		btnColor := p.theme.ButtonBackground

		// Draw button background.
		vector.DrawFilledRect(screen, buttonX, btnY, buttonW, float32(nodeDetailButtonHeight), btnColor, true)

		// Draw button border.
		vector.StrokeRect(screen, buttonX, btnY, buttonW, float32(nodeDetailButtonHeight), 1.0, p.theme.PanelBorder, true)

		// Draw button label (simplified).
		labelColor := p.theme.TextPrimary
		labelX := buttonX + 10
		labelY := btnY + 10
		vector.DrawFilledRect(screen, labelX, labelY, float32(len(label)*8), 15, labelColor, true)

		// TODO: Replace with actual text rendering.
		// Display label centered in button.
	}
}

// drawRecentWaves draws the Recent Waves list.
func (p *NodeDetailPanel) drawRecentWaves(screen *ebiten.Image, y float32) {
	// Draw section title.
	titleColor := p.theme.TextPrimary
	titleX := float32(p.panelX + nodeDetailPadding)
	vector.DrawFilledRect(screen, titleX, y, 100, 20, titleColor, true)

	// Draw list items.
	listY := y + 30
	listX := float32(p.panelX + nodeDetailPadding)
	listW := float32(p.panelW - 2*nodeDetailPadding)

	visibleWaves := p.nodeInfo.RecentWaves
	if len(visibleWaves) > p.waveScroll {
		visibleWaves = visibleWaves[p.waveScroll:]
	}
	if len(visibleWaves) > nodeDetailMaxVisibleItems {
		visibleWaves = visibleWaves[:nodeDetailMaxVisibleItems]
	}

	for i := range visibleWaves {
		itemY := listY + float32(i*nodeDetailListItemHeight)

		// Draw item background.
		itemBg := p.theme.InputBackground
		vector.DrawFilledRect(screen, listX, itemY, listW, float32(nodeDetailListItemHeight-5), itemBg, true)

		// Draw Wave content (truncated).
		contentColor := p.theme.TextPrimary
		contentX := listX + 10
		contentY := itemY + 10
		vector.DrawFilledRect(screen, contentX, contentY, listW-20, 15, contentColor, true)

		// Draw timestamp.
		timestampColor := p.theme.TextSecondary
		timestampY := contentY + 25
		vector.DrawFilledRect(screen, contentX, timestampY, 100, 12, timestampColor, true)

		// TODO: Replace with actual text rendering.
		// Display visibleWaves[i].Content (truncated) and visibleWaves[i].Timestamp.Format("15:04").
	}

	// TODO: Draw scroll indicator if more items available.
}

// drawConnections draws the Connections list.
func (p *NodeDetailPanel) drawConnections(screen *ebiten.Image, y float32) {
	// Draw section title.
	titleColor := p.theme.TextPrimary
	titleX := float32(p.panelX + nodeDetailPadding)
	vector.DrawFilledRect(screen, titleX, y, 150, 20, titleColor, true)

	// Draw connection count.
	countY := y + 25
	countColor := p.theme.TextSecondary
	countText := fmt.Sprintf("Connections: %d", p.nodeInfo.ConnectionCount)
	vector.DrawFilledRect(screen, titleX, countY, float32(len(countText)*7), 15, countColor, true)

	// TODO: Draw connection list (similar to Waves list).
	// For now, just show count.
}
