// Package ui - Node Detail Panel test stubs.

//go:build test
// +build test

package ui

import "time"

// NodeInfo contains information about a node for display.
type NodeInfo struct {
	PublicKey       string
	DisplayName     string
	Fingerprint     string
	IsSpecter       bool
	IsSurface       bool
	IsSelf          bool
	Resonance       int
	ResonanceRank   string
	ConnectionCount int
	Connections     []string
	RecentWaves     []WaveInfo
}

// WaveInfo contains information about a Wave for display.
type WaveInfo struct {
	Content   string
	Timestamp time.Time
	WaveType  string
}

// NodeDetailCallbacks provides callbacks for node detail panel actions.
type NodeDetailCallbacks struct {
	OnComposeWave func(nodeID string)
	OnSendGift    func(nodeID string)
	OnPlaceMark   func(nodeID string)
	OnSendWhisper func(nodeID string)
	OnViewWave    func(waveID string)
	OnClose       func()
}

// NodeDetailPanel displays detailed information about a selected node.
type NodeDetailPanel struct {
	visible  bool
	nodeInfo *NodeInfo
}

// NewNodeDetailPanel creates a new node detail panel.
func NewNodeDetailPanel(theme Theme, callbacks NodeDetailCallbacks) *NodeDetailPanel {
	return &NodeDetailPanel{}
}

// Show displays the panel for the given node.
func (p *NodeDetailPanel) Show(nodeInfo *NodeInfo) {
	p.visible = true
	p.nodeInfo = nodeInfo
}

// Hide hides the panel.
func (p *NodeDetailPanel) Hide() {
	p.visible = false
	p.nodeInfo = nil
}

// Visible returns true if the panel is currently shown.
func (p *NodeDetailPanel) Visible() bool {
	return p.visible
}

// Toggle toggles panel visibility.
func (p *NodeDetailPanel) Toggle() {
	p.visible = !p.visible
	if !p.visible {
		p.nodeInfo = nil
	}
}

// Update handles input (stub).
func (p *NodeDetailPanel) Update() bool {
	return p.visible
}

// Draw renders the panel (stub).
func (p *NodeDetailPanel) Draw(screen Screen) {}
