// Package ui provides stub types for the Compose panel.
//
//go:build test
// +build test

package ui

import (
	"sync"
	"unicode/utf8"
)

// MaxWaveLength is the maximum content length for a Wave (2048 bytes per WAVES.md).
const MaxWaveLength = 2048

// ComposePanel provides a UI for composing and submitting Waves (stub).
type ComposePanel struct {
	mu sync.RWMutex

	visible      bool
	x, y         int
	width        int
	height       int
	position     PanelPosition
	content      string
	cursorPos    int
	targetNodeID string
	waveType     uint8
	errorMessage string
	onSubmit     WaveSubmitCallback
	theme        Theme
}

// NewComposePanel creates a new Wave composition panel (stub).
func NewComposePanel(theme Theme, onSubmit WaveSubmitCallback) *ComposePanel {
	return &ComposePanel{
		theme:    theme,
		onSubmit: onSubmit,
		width:    400,
		height:   280,
		position: PositionBottomRight,
		waveType: 0x01,
	}
}

// Visible returns true if the panel is shown.
func (p *ComposePanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *ComposePanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
}

// Hide hides the panel.
func (p *ComposePanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	ResetPanelInputState(&p.visible, &p.content, &p.errorMessage, &p.cursorPos)
}

// Toggle toggles panel visibility.
func (p *ComposePanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetTargetNode sets the node to send the Wave to.
func (p *ComposePanel) SetTargetNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targetNodeID = nodeID
}

// SetWaveType sets the Wave type to create.
func (p *ComposePanel) SetWaveType(waveType uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waveType = waveType
}

// Update handles input and updates panel state (stub).
func (p *ComposePanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
}

// Content returns the current input content.
func (p *ComposePanel) Content() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.content
}

// SetContent sets the input content.
func (p *ComposePanel) SetContent(content string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(content) > MaxWaveLength {
		content = content[:MaxWaveLength]
	}
	p.content = content
	p.cursorPos = utf8.RuneCountInString(content)
}

// Submit triggers the submit callback (for testing).
func (p *ComposePanel) Submit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.content) == 0 {
		p.errorMessage = "Cannot send empty Wave"
		return
	}

	if p.onSubmit != nil {
		p.onSubmit(p.content, p.waveType, p.targetNodeID)
	}

	p.content = ""
	p.cursorPos = 0
	p.visible = false
}

// SimulateClick simulates a left-click at (cx, cy) and returns true if a button was hit.
// Mirrors the handleClickAt logic in compose.go for test coverage.
func (p *ComposePanel) SimulateClick(cx, cy int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.visible {
		return false
	}
	const submitWidth = 100
	const cancelWidth = 80
	buttonY := p.y + p.height - p.theme.Padding - p.theme.ButtonHeight
	submitX := p.x + p.width - p.theme.Padding - submitWidth
	cancelX := p.x + p.theme.Padding
	if cy >= buttonY && cy < buttonY+p.theme.ButtonHeight {
		if cx >= submitX && cx < submitX+submitWidth {
			// Inline submit logic (no Ebitengine dependency).
			if len(p.content) == 0 {
				p.errorMessage = "Cannot send empty Wave"
				return true
			}
			if p.onSubmit != nil {
				p.onSubmit(p.content, p.waveType, p.targetNodeID)
			}
			p.content = ""
			p.cursorPos = 0
			p.visible = false
			return true
		}
		if cx >= cancelX && cx < cancelX+cancelWidth {
			p.visible = false
			return true
		}
	}
	return false
}
