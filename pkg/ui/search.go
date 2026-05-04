// Package ui - Search Bar for finding nodes by display name, fingerprint, or pseudonym.
// Per ROADMAP.md line 670: "Search bar — find by display name, fingerprint, or pseudonym".
//
// The search bar appears at the top of the screen when activated via Ctrl+F.
// Users can type partial matches and see filtered results in a dropdown.
// Clicking a result centers the camera on that node.

//go:build !test
// +build !test

package ui

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SearchCallbacks provides callbacks for search bar actions.
type SearchCallbacks struct {
	// OnSearch is called when user types in the search bar.
	// The callback should return matching results.
	OnSearch func(query string) []SearchResult

	// OnSelect is called when user clicks a result.
	OnSelect func(nodeID string)

	// OnClose is called when user closes the search bar.
	OnClose func()
}

// SearchBar provides node search functionality.
type SearchBar struct {
	mu sync.RWMutex

	// State
	visible   bool
	theme     Theme
	callbacks SearchCallbacks

	// Input state
	query         string // Current search query
	cursorPos     int    // Cursor position in query
	results       []SearchResult
	selectedIndex int // Selected result index (-1 = none)

	// Animation
	opacity float64 // Fade in/out animation

	// Dimensions (set in Draw)
	barX, barY int
	barW, barH int
}

// Search bar dimensions.
const (
	searchBarWidth        = 500 // Bar width in pixels
	searchBarHeight       = 50  // Bar height in pixels
	searchBarPadding      = 10  // Padding inside bar
	searchBarMarginTop    = 20  // Margin from top of screen
	searchResultHeight    = 40  // Height of each result item
	searchMaxResults      = 8   // Max visible results
	searchFadeSpeed       = 0.1 // Opacity change per frame
	searchCursorBlinkRate = 30  // Frames per blink cycle
)

// NewSearchBar creates a new search bar.
func NewSearchBar(theme Theme, callbacks SearchCallbacks) *SearchBar {
	return &SearchBar{
		theme:         theme,
		callbacks:     callbacks,
		selectedIndex: -1,
		opacity:       0.0,
	}
}

// Show displays the search bar.
func (s *SearchBar) Show() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visible = true
	s.query = ""
	s.cursorPos = 0
	s.results = nil
	s.selectedIndex = -1
}

// Hide hides the search bar.
func (s *SearchBar) Hide() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visible = false
	s.query = ""
	s.results = nil
	s.selectedIndex = -1
}

// Visible returns true if the search bar is currently shown.
func (s *SearchBar) Visible() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.visible
}

// Toggle toggles search bar visibility.
func (s *SearchBar) Toggle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visible = !s.visible
	if !s.visible {
		s.query = ""
		s.results = nil
	}
}

// Update handles input and updates search bar state.
// Returns true if the search bar consumed the input.
func (s *SearchBar) Update() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.visible {
		return false
	}

	s.updateFadeAnimation()
	s.calculateBarPosition()

	// Handle text input.
	if s.handleTextInput() {
		s.performSearch()
	}

	// Handle keyboard navigation.
	if s.handleKeyboardNav() {
		return true
	}

	// Handle mouse clicks on results.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return s.handleMouseClick()
	}

	// Handle Escape to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.visible = false
		if s.callbacks.OnClose != nil {
			s.mu.Unlock()
			s.callbacks.OnClose()
			s.mu.Lock()
		}
		return true
	}

	return true // Search bar is visible, consume all input
}

// updateFadeAnimation updates the fade in/out animation.
func (s *SearchBar) updateFadeAnimation() {
	if s.visible && s.opacity < 1.0 {
		s.opacity += searchFadeSpeed
		if s.opacity > 1.0 {
			s.opacity = 1.0
		}
	} else if !s.visible && s.opacity > 0 {
		s.opacity -= searchFadeSpeed
		if s.opacity < 0 {
			s.opacity = 0
		}
	}
}

// calculateBarPosition calculates search bar dimensions and position.
func (s *SearchBar) calculateBarPosition() {
	screenW, _ := ebiten.WindowSize()
	s.barW = searchBarWidth
	s.barH = searchBarHeight
	s.barX = (screenW - s.barW) / 2
	s.barY = searchBarMarginTop
}

// handleTextInput processes text input and returns true if query changed.
func (s *SearchBar) handleTextInput() bool {
	changed := false

	// Handle character input.
	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		// Insert at cursor position.
		before := s.query[:s.cursorPos]
		after := s.query[s.cursorPos:]
		s.query = before + string(r) + after
		s.cursorPos++
		changed = true
	}

	// Handle backspace.
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && s.cursorPos > 0 {
		before := s.query[:s.cursorPos-1]
		after := s.query[s.cursorPos:]
		s.query = before + after
		s.cursorPos--
		changed = true
	}

	// Handle delete.
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) && s.cursorPos < utf8.RuneCountInString(s.query) {
		before := s.query[:s.cursorPos]
		after := s.query[s.cursorPos+1:]
		s.query = before + after
		changed = true
	}

	// Handle cursor movement.
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && s.cursorPos > 0 {
		s.cursorPos--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && s.cursorPos < utf8.RuneCountInString(s.query) {
		s.cursorPos++
	}

	return changed
}

// performSearch executes the search query via callback.
func (s *SearchBar) performSearch() {
	if s.callbacks.OnSearch == nil {
		return
	}

	// Perform search asynchronously to avoid blocking UI.
	query := s.query
	s.mu.Unlock()
	results := s.callbacks.OnSearch(query)
	s.mu.Lock()

	s.results = results
	s.selectedIndex = -1
}

// handleKeyboardNav handles arrow keys and Enter for result selection.
func (s *SearchBar) handleKeyboardNav() bool {
	if len(s.results) == 0 {
		return false
	}

	// Arrow down - select next result.
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		s.selectedIndex++
		if s.selectedIndex >= len(s.results) {
			s.selectedIndex = 0
		}
		return true
	}

	// Arrow up - select previous result.
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		s.selectedIndex--
		if s.selectedIndex < 0 {
			s.selectedIndex = len(s.results) - 1
		}
		return true
	}

	// Enter - select current result.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && s.selectedIndex >= 0 && s.selectedIndex < len(s.results) {
		result := s.results[s.selectedIndex]
		s.visible = false
		if s.callbacks.OnSelect != nil {
			s.mu.Unlock()
			s.callbacks.OnSelect(result.NodeID)
			s.mu.Lock()
		}
		return true
	}

	return false
}

// handleMouseClick handles clicks on search results.
func (s *SearchBar) handleMouseClick() bool {
	if len(s.results) == 0 {
		return false
	}

	cursorX, cursorY := ebiten.CursorPosition()

	// Check if click is on a result.
	resultsY := s.barY + s.barH + 5
	for i := 0; i < len(s.results) && i < searchMaxResults; i++ {
		itemY := resultsY + i*searchResultHeight
		if cursorX >= s.barX && cursorX < s.barX+s.barW &&
			cursorY >= itemY && cursorY < itemY+searchResultHeight {
			result := s.results[i]
			s.visible = false
			if s.callbacks.OnSelect != nil {
				s.mu.Unlock()
				s.callbacks.OnSelect(result.NodeID)
				s.mu.Lock()
			}
			return true
		}
	}

	return false
}

// Draw renders the search bar.
func (s *SearchBar) Draw(screen *ebiten.Image) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.visible && s.opacity == 0 {
		return
	}

	// Apply opacity to colors.
	alpha := uint8(s.opacity * 255)

	// Draw search bar background.
	barBg := s.theme.PanelBackground
	barBg.A = alpha
	vector.DrawFilledRect(screen, float32(s.barX), float32(s.barY), float32(s.barW), float32(s.barH), barBg, true)

	// Draw search bar border.
	borderColor := s.theme.PanelBorder
	borderColor.A = alpha
	vector.StrokeRect(screen, float32(s.barX), float32(s.barY), float32(s.barW), float32(s.barH), 2.0, borderColor, true)

	// Draw search query text (simplified - actual implementation uses text rendering).
	textColor := s.theme.TextPrimary
	textColor.A = alpha
	textX := float32(s.barX + searchBarPadding)
	textY := float32(s.barY + searchBarPadding)
	if s.query == "" {
		// Draw placeholder.
		placeholderColor := s.theme.TextSecondary
		placeholderColor.A = uint8(float64(alpha) * 0.5)
		vector.DrawFilledRect(screen, textX, textY, 200, 20, placeholderColor, true)
		// TODO: Draw "Search by name, fingerprint..." text.
	} else {
		vector.DrawFilledRect(screen, textX, textY, float32(len(s.query)*8), 20, textColor, true)
		// TODO: Draw s.query text.
	}

	// Draw cursor (blinking).
	if s.visible && (ebiten.TPS()/searchCursorBlinkRate)%2 == 0 {
		cursorX := textX + float32(s.cursorPos*8)
		cursorY := textY
		vector.DrawFilledRect(screen, cursorX, cursorY, 2, 20, textColor, true)
	}

	// Draw results dropdown.
	if len(s.results) > 0 {
		s.drawResults(screen, alpha)
	}
}

// drawResults renders the search results dropdown.
func (s *SearchBar) drawResults(screen *ebiten.Image, alpha uint8) {
	resultsY := s.barY + s.barH + 5
	maxVisible := len(s.results)
	if maxVisible > searchMaxResults {
		maxVisible = searchMaxResults
	}

	// Draw results background.
	resultsBg := s.theme.PanelBackground
	resultsBg.A = alpha
	resultsH := maxVisible * searchResultHeight
	vector.DrawFilledRect(screen, float32(s.barX), float32(resultsY), float32(s.barW), float32(resultsH), resultsBg, true)

	// Draw results border.
	borderColor := s.theme.PanelBorder
	borderColor.A = alpha
	vector.StrokeRect(screen, float32(s.barX), float32(resultsY), float32(s.barW), float32(resultsH), 1.0, borderColor, true)

	// Draw each result.
	for i := 0; i < maxVisible; i++ {
		result := s.results[i]
		itemY := resultsY + i*searchResultHeight

		// Highlight selected item.
		if i == s.selectedIndex {
			highlightColor := s.theme.AccentPrimary
			highlightColor.A = uint8(float64(alpha) * 0.3)
			vector.DrawFilledRect(screen, float32(s.barX), float32(itemY), float32(s.barW), float32(searchResultHeight), highlightColor, true)
		}

		// Draw display name.
		nameColor := s.theme.TextPrimary
		nameColor.A = alpha
		nameX := float32(s.barX + searchBarPadding)
		nameY := float32(itemY + 10)
		vector.DrawFilledRect(screen, nameX, nameY, float32(len(result.DisplayName)*8), 15, nameColor, true)
		// TODO: Draw result.DisplayName text.

		// Draw pseudonym below name (if Specter).
		if result.IsSpecter && result.Pseudonym != "" {
			pseudonymColor := s.theme.TextSecondary
			pseudonymColor.A = alpha
			pseudonymY := nameY + 20
			vector.DrawFilledRect(screen, nameX, pseudonymY, 80, 12, pseudonymColor, true)
			// TODO: Draw result.Pseudonym text.
		}

		// Draw Specter badge if applicable.
		if result.IsSpecter {
			badgeColor := s.theme.AccentSecondary
			badgeColor.A = alpha
			badgeX := float32(s.barX + s.barW - 60)
			badgeY := nameY
			vector.DrawFilledRect(screen, badgeX, badgeY, 50, 15, badgeColor, true)
			// TODO: Draw "Specter" badge text.
		}
	}
}

// FilterResults performs case-insensitive substring matching on a list of results.
// This is a helper for implementing the OnSearch callback.
func FilterResults(query string, allResults []SearchResult) []SearchResult {
	if query == "" {
		return allResults
	}

	query = strings.ToLower(query)
	var filtered []SearchResult

	for _, result := range allResults {
		// Match display name, pseudonym, or node ID.
		if strings.Contains(strings.ToLower(result.DisplayName), query) ||
			strings.Contains(strings.ToLower(result.Pseudonym), query) ||
			strings.Contains(strings.ToLower(result.NodeID), query) {
			filtered = append(filtered, result)
		}
	}

	return filtered
}
