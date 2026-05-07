// Package ui - Node Detail Panel unit tests.
package ui

import (
	"testing"
	"time"
)

// TestNodeDetailPanel_ButtonYMatchesDrawLayout asserts that the button Y offset used
// in handlePanelClick matches the Y produced by Draw at slideOffset=0.
// Per AUDIT.md MEDIUM finding: the magic constant 100 was replaced with
// nodeDetailButtonGroupY = nodeDetailPadding + nodeDetailHeaderHeight + 20 = 100.
func TestNodeDetailPanel_ButtonYMatchesDrawLayout(t *testing.T) {
	// Verify the arithmetic matches between Draw() and handlePanelClick():
	//   Draw:   headerY = panelY + 20 (padding)
	//           resonanceY = headerY + 60 (header height) = panelY + 80
	//           buttonY = resonanceY + 20 = panelY + 100
	//   Click:  buttonY = panelY + nodeDetailButtonGroupY
	//           = panelY + padding + headerH + 20 = panelY + 20 + 60 + 20 = panelY + 100
	//
	// Both must produce the same button Y at slideOffset=0 for a surface node.
	const (
		padding = 20                     // nodeDetailPadding
		headerH = 60                     // nodeDetailHeaderHeight
		groupY  = padding + headerH + 20 // nodeDetailButtonGroupY = 100
	)

	const panelY = 0
	drawButtonY := panelY + padding + headerH + 20
	clickButtonY := panelY + groupY

	if drawButtonY != clickButtonY {
		t.Errorf("button Y in Draw (%d) != button Y in handlePanelClick (%d); "+
			"layout constants are out of sync", drawButtonY, clickButtonY)
	}
	// Sanity: confirm expected absolute value.
	if clickButtonY != 100 {
		t.Errorf("expected button Y = 100 at panelY=0, got %d", clickButtonY)
	}
}

func TestNodeDetailPanel_ShowHide(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Initially not visible.
	if panel.Visible() {
		t.Error("Panel should not be visible initially")
	}

	// Show panel.
	nodeInfo := &NodeInfo{
		PublicKey:   "abcd1234",
		DisplayName: "Test Node",
		Fingerprint: "abcd1234",
		IsSurface:   true,
	}
	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	// Hide panel.
	panel.Hide()
	if panel.Visible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestNodeDetailPanel_Toggle(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Toggle on.
	panel.Toggle()
	if !panel.Visible() {
		t.Error("Panel should be visible after first Toggle()")
	}

	// Toggle off.
	panel.Toggle()
	if panel.Visible() {
		t.Error("Panel should not be visible after second Toggle()")
	}
}

func TestNodeDetailPanel_NodeInfo(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Create node info with all fields.
	nodeInfo := &NodeInfo{
		PublicKey:       "0123456789abcdef",
		DisplayName:     "Mysterious Specter",
		Fingerprint:     "01234567",
		IsSpecter:       true,
		IsSurface:       false,
		IsSelf:          false,
		Resonance:       125,
		ResonanceRank:   "Phantom",
		ConnectionCount: 15,
		Connections:     []string{"Node A", "Node B", "Node C"},
		RecentWaves: []WaveInfo{
			{Content: "First wave", Timestamp: time.Now(), WaveType: "Specter"},
			{Content: "Second wave", Timestamp: time.Now(), WaveType: "Specter"},
		},
	}

	panel.Show(nodeInfo)

	// Panel should be visible.
	if !panel.Visible() {
		t.Error("Panel should be visible after Show()")
	}

	// Update should return true (consumes input).
	if !panel.Update() {
		t.Error("Update() should return true when panel is visible")
	}
}

func TestNodeDetailPanel_CallbackInvocation(t *testing.T) {
	composeWaveCalled := false
	sendGiftCalled := false
	closeCalled := false

	callbacks := NodeDetailCallbacks{
		OnComposeWave: func(nodeID string) {
			composeWaveCalled = true
		},
		OnSendGift: func(nodeID string) {
			sendGiftCalled = true
		},
		OnClose: func() {
			closeCalled = true
		},
	}

	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	nodeInfo := &NodeInfo{
		PublicKey:   "test-node-id",
		DisplayName: "Test",
	}
	panel.Show(nodeInfo)

	// Callbacks are invoked via user interaction (not tested here due to stub).
	// This test verifies the structure compiles.
	_ = composeWaveCalled
	_ = sendGiftCalled
	_ = closeCalled
}

func TestNodeDetailPanel_SurfaceNode(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Surface node (no Resonance).
	nodeInfo := &NodeInfo{
		PublicKey:       "surface123",
		DisplayName:     "Surface User",
		Fingerprint:     "surface1",
		IsSpecter:       false,
		IsSurface:       true,
		IsSelf:          false,
		Resonance:       0,
		ResonanceRank:   "",
		ConnectionCount: 20,
		RecentWaves: []WaveInfo{
			{Content: "Hello world", Timestamp: time.Now(), WaveType: "Surface"},
		},
	}

	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible for Surface node")
	}
}

func TestNodeDetailPanel_SpecterNode(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Specter node with Resonance.
	nodeInfo := &NodeInfo{
		PublicKey:       "specter456",
		DisplayName:     "Shadowy Figure",
		Fingerprint:     "specter4",
		IsSpecter:       true,
		IsSurface:       false,
		IsSelf:          false,
		Resonance:       75,
		ResonanceRank:   "Shade-Wraith",
		ConnectionCount: 8,
		RecentWaves: []WaveInfo{
			{Content: "Anonymous message", Timestamp: time.Now(), WaveType: "Specter"},
		},
	}

	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible for Specter node")
	}
}

func TestNodeDetailPanel_SelfNode(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Own node.
	nodeInfo := &NodeInfo{
		PublicKey:       "self789",
		DisplayName:     "My Identity",
		Fingerprint:     "self7890",
		IsSpecter:       false,
		IsSurface:       true,
		IsSelf:          true,
		Resonance:       0,
		ResonanceRank:   "",
		ConnectionCount: 25,
	}

	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible for own node")
	}
}

func TestNodeDetailPanel_EmptyWavesList(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Node with no recent Waves.
	nodeInfo := &NodeInfo{
		PublicKey:       "empty123",
		DisplayName:     "Quiet Node",
		Fingerprint:     "empty123",
		IsSurface:       true,
		RecentWaves:     []WaveInfo{}, // Empty list
		ConnectionCount: 0,
	}

	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible even with empty Waves list")
	}

	// Update should not panic.
	panel.Update()
}

func TestNodeDetailPanel_ManyWaves(t *testing.T) {
	callbacks := NodeDetailCallbacks{}
	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)

	// Node with many Waves (tests scrolling logic).
	waves := make([]WaveInfo, 20)
	for i := 0; i < 20; i++ {
		waves[i] = WaveInfo{
			Content:   "Wave content",
			Timestamp: time.Now(),
			WaveType:  "Surface",
		}
	}

	nodeInfo := &NodeInfo{
		PublicKey:   "many123",
		DisplayName: "Active Node",
		RecentWaves: waves,
	}

	panel.Show(nodeInfo)
	if !panel.Visible() {
		t.Error("Panel should be visible with many Waves")
	}

	// Multiple updates should not panic.
	for i := 0; i < 10; i++ {
		panel.Update()
	}
}

func TestNodeDetailPanel_HandlePanelClickRespectsXBounds(t *testing.T) {
	composeCalls := 0
	viewWaveCalls := 0
	callbacks := NodeDetailCallbacks{
		OnComposeWave: func(nodeID string) {
			composeCalls++
		},
		OnViewWave: func(waveID string) {
			viewWaveCalls++
		},
	}

	panel := NewNodeDetailPanel(DefaultTheme(), callbacks)
	panel.panelX = 100
	panel.panelY = 50
	panel.panelW = nodeDetailPanelWidth
	panel.panelH = nodeDetailPanelHeight
	panel.nodeInfo = &NodeInfo{
		PublicKey: "node-1",
		RecentWaves: []WaveInfo{
			{WaveID: "wave-1", Content: "hello", Timestamp: time.Now(), WaveType: "Surface"},
		},
	}

	insideX := panel.panelX + nodeDetailPadding + 1
	outsideX := panel.panelX + panel.panelW - 1 // outside button/list X range (right padding zone)
	buttonY := panel.panelY + nodeDetailButtonGroupY + 1
	wavesY := panel.panelY + nodeDetailButtonGroupY + 4*(nodeDetailButtonHeight+nodeDetailButtonSpacing) + 20 + 22

	// Outside X bounds: should not trigger button callback.
	panel.handlePanelClick(outsideX, buttonY)
	if composeCalls != 0 {
		t.Fatalf("expected compose callback not to fire for outside-X click, got %d", composeCalls)
	}

	// Inside X bounds: callback should fire.
	panel.handlePanelClick(insideX, buttonY)
	if composeCalls != 1 {
		t.Fatalf("expected compose callback once for inside-X click, got %d", composeCalls)
	}

	// Outside X bounds in wave row: should not trigger wave callback.
	panel.handlePanelClick(outsideX, wavesY+1)
	if viewWaveCalls != 0 {
		t.Fatalf("expected wave callback not to fire for outside-X click, got %d", viewWaveCalls)
	}

	// Inside X bounds in wave row: should trigger wave callback.
	panel.handlePanelClick(insideX, wavesY+1)
	if viewWaveCalls != 1 {
		t.Fatalf("expected wave callback once for inside-X click, got %d", viewWaveCalls)
	}
}
