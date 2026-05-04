// Package ui - Node Detail Panel unit tests.
package ui

import (
	"testing"
	"time"
)

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
