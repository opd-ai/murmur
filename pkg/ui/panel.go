// Package ui provides immediate-mode UI panels for MURMUR.
// Per PULSE_MAP.md, the UI overlays the Pulse Map with panels for composing Waves,
// viewing node details, searching, and configuring settings.
//
// This package is the only one under pkg/ that implements Ebitengine UI components
// besides pkg/pulsemap/rendering. All panels use Ebitengine's vector drawing
// for consistent rendering across platforms.
package ui

import (
	"image/color"
)

// Panel is the interface implemented by all UI panels.
type Panel interface {
	// Update handles input and updates panel state.
	// Returns true if the panel consumed the input.
	Update() bool

	// Draw renders the panel.
	Draw(screen Screen)

	// Visible returns true if the panel is currently shown.
	Visible() bool

	// Show displays the panel.
	Show()

	// Hide hides the panel.
	Hide()

	// Toggle toggles panel visibility.
	Toggle()
}

// PanelAnimation handles common panel animation and error display logic.
type PanelAnimation struct {
	animTime     float64
	slideOffset  float64
	errorTime    float64
	errorMessage string
}

// UpdateAnimation updates the slide-in animation and error message timeout.
// Should be called once per frame. Returns true if error message was cleared.
func (a *PanelAnimation) UpdateAnimation() bool {
	// Animate slide-in.
	a.animTime += 1.0 / 60.0
	if a.slideOffset > 0 {
		a.slideOffset *= 0.85
		if a.slideOffset < 1 {
			a.slideOffset = 0
		}
	}

	// Clear error after 3 seconds.
	errorCleared := false
	if a.errorMessage != "" {
		a.errorTime += 1.0 / 60.0
		if a.errorTime > 3.0 {
			a.errorMessage = ""
			a.errorTime = 0
			errorCleared = true
		}
	}
	return errorCleared
}

// SetError sets an error message to display.
func (a *PanelAnimation) SetError(msg string) {
	a.errorMessage = msg
	a.errorTime = 0
}

// ErrorMessage returns the current error message.
func (a *PanelAnimation) ErrorMessage() string {
	return a.errorMessage
}

// SlideOffset returns the current slide offset for animations.
func (a *PanelAnimation) SlideOffset() float64 {
	return a.slideOffset
}

// AnimTime returns the current animation time for effects like blinking cursors.
func (a *PanelAnimation) AnimTime() float64 {
	return a.animTime
}

// ResetAnimation resets animation state (call when showing panel).
func (a *PanelAnimation) ResetAnimation() {
	a.animTime = 0
	a.slideOffset = 300
	a.errorMessage = ""
	a.errorTime = 0
}

// Screen is an abstraction over ebiten.Image for rendering.
// This allows testing without Ebitengine dependency.
type Screen interface {
	Bounds() Rectangle
	Fill(c color.Color)
}

// Rectangle represents a screen region.
type Rectangle struct {
	X, Y, Width, Height int
}

// Theme contains colors and styling for UI panels.
type Theme struct {
	// Background colors
	PanelBackground  color.RGBA
	PanelBorder      color.RGBA
	InputBackground  color.RGBA
	ButtonBackground color.RGBA
	ButtonHover      color.RGBA
	ButtonActive     color.RGBA

	// Text colors
	TextPrimary     color.RGBA
	TextSecondary   color.RGBA
	TextPlaceholder color.RGBA
	TextError       color.RGBA

	// Accent colors
	AccentPrimary   color.RGBA
	AccentSecondary color.RGBA

	// Status colors
	Success   color.RGBA
	Warning   color.RGBA
	Selection color.RGBA

	// Sizing
	FontSize     int
	Padding      int
	BorderRadius int
	ButtonHeight int
	InputHeight  int
}

// DefaultTheme returns the default dark theme for MURMUR.
// Per PULSE_MAP.md, the UI uses a dark color scheme with accent colors.
func DefaultTheme() Theme {
	return Theme{
		PanelBackground:  color.RGBA{20, 22, 30, 240},
		PanelBorder:      color.RGBA{60, 65, 80, 255},
		InputBackground:  color.RGBA{30, 33, 45, 255},
		ButtonBackground: color.RGBA{50, 55, 70, 255},
		ButtonHover:      color.RGBA{70, 75, 95, 255},
		ButtonActive:     color.RGBA{90, 100, 120, 255},
		TextPrimary:      color.RGBA{230, 235, 240, 255},
		TextSecondary:    color.RGBA{160, 170, 185, 255},
		TextPlaceholder:  color.RGBA{100, 110, 130, 255},
		TextError:        color.RGBA{255, 100, 100, 255},
		AccentPrimary:    color.RGBA{80, 150, 220, 255},
		AccentSecondary:  color.RGBA{100, 200, 160, 255},
		Success:          color.RGBA{80, 200, 120, 255},
		Warning:          color.RGBA{255, 180, 60, 255},
		Selection:        color.RGBA{60, 80, 120, 180},
		FontSize:         14,
		Padding:          12,
		BorderRadius:     6,
		ButtonHeight:     36,
		InputHeight:      40,
	}
}

// PanelPosition indicates where a panel should be anchored.
type PanelPosition int

const (
	PositionCenter PanelPosition = iota
	PositionTopLeft
	PositionTopRight
	PositionBottomLeft
	PositionBottomRight
	PositionLeft
	PositionRight
)

// Callback types for panel actions.
type (
	// WaveSubmitCallback is called when a Wave is composed and submitted.
	WaveSubmitCallback func(content string, waveType uint8, targetNodeID string)

	// SettingsChangeCallback is called when a setting is changed.
	SettingsChangeCallback func(key, value string)

	// NodeSelectCallback is called when a node is selected.
	NodeSelectCallback func(nodeID string)

	// SearchCallback is called when a search is performed.
	SearchCallback func(query string) []SearchResult
)

// SearchResult represents a search result entry.
type SearchResult struct {
	NodeID      string
	DisplayName string
	Pseudonym   string
	IsSpecter   bool
	Relevance   float32
	Resonance   float64
}
