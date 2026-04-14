// Package ui provides the Hunt Tracker overlay for viewing Specter Hunt progress.
// Per ROADMAP.md line 434: "UI: Hunt tracker overlay — fragment locations, clue display, leaderboard".
//
//go:build !noebiten
// +build !noebiten

package ui

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// HuntInfo contains information about a hunt for display.
type HuntInfo struct {
	ID               [32]byte
	Theme            string
	ExpiresAt        time.Time
	FragmentCount    int
	ClaimedCount     int
	Fragments        []FragmentInfo
	Leaderboard      []LeaderboardEntry
	SelectedFragment int // Index of selected fragment, -1 if none.
	UserClaims       int // How many fragments the user has claimed.
}

// FragmentInfo contains information about a hunt fragment.
type FragmentInfo struct {
	Index        int
	Claimed      bool
	ClaimedByMe  bool
	Clues        []string
	LocationHint string // Derived from visible clues.
}

// LeaderboardEntry represents a participant in the leaderboard.
type LeaderboardEntry struct {
	Pseudonym string
	Claims    int
	IsMe      bool
}

// HuntTrackerPanel provides a UI overlay for tracking Specter Hunts.
type HuntTrackerPanel struct {
	mu sync.RWMutex

	// Visibility and position.
	visible  bool
	x, y     int
	width    int
	height   int
	position PanelPosition

	// Current hunt info.
	hunt *HuntInfo

	// UI state.
	selectedTab    int // 0=Fragments, 1=Clues, 2=Leaderboard.
	scrollOffset   int
	showOnlyActive bool
	errorMessage   string
	errorTime      float64

	// Callbacks.
	onFragmentSelect func(huntID [32]byte, fragmentIndex int)
	onClaimAttempt   func(huntID [32]byte, fragmentIndex int)

	// Styling.
	theme Theme

	// Animation.
	animTime    float64
	slideOffset float64
	pulseTime   float64 // For pulsing unclaimed fragments.

	// Screen dimensions.
	screenWidth, screenHeight int
}

// NewHuntTrackerPanel creates a new hunt tracker panel.
func NewHuntTrackerPanel(theme Theme) *HuntTrackerPanel {
	return &HuntTrackerPanel{
		theme:          theme,
		width:          380,
		height:         450,
		position:       PositionRight,
		selectedTab:    0,
		showOnlyActive: true,
	}
}

// Visible returns true if the panel is shown.
func (p *HuntTrackerPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel with animation.
func (p *HuntTrackerPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.slideOffset = float64(p.width)
	p.animTime = 0
}

// Hide hides the panel.
func (p *HuntTrackerPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *HuntTrackerPanel) Toggle() {
	p.mu.Lock()
	visible := p.visible
	p.mu.Unlock()

	if visible {
		p.Hide()
	} else {
		p.Show()
	}
}

// SetHunt sets the current hunt to track.
func (p *HuntTrackerPanel) SetHunt(hunt *HuntInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hunt = hunt
	p.scrollOffset = 0
}

// SetOnFragmentSelect sets the callback for fragment selection.
func (p *HuntTrackerPanel) SetOnFragmentSelect(callback func(huntID [32]byte, fragmentIndex int)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onFragmentSelect = callback
}

// SetOnClaimAttempt sets the callback for claim attempts.
func (p *HuntTrackerPanel) SetOnClaimAttempt(callback func(huntID [32]byte, fragmentIndex int)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onClaimAttempt = callback
}

// Update handles input and updates panel state.
func (p *HuntTrackerPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	// Animate slide-in.
	p.animTime += 1.0 / 60.0
	p.pulseTime += 1.0 / 60.0
	if p.slideOffset > 0 {
		p.slideOffset *= 0.85
		if p.slideOffset < 1 {
			p.slideOffset = 0
		}
	}

	// Clear error after 3 seconds.
	if p.errorMessage != "" {
		p.errorTime += 1.0 / 60.0
		if p.errorTime > 3.0 {
			p.errorMessage = ""
			p.errorTime = 0
		}
	}

	// Handle tab switching.
	p.handleTabInput()

	// Handle scrolling.
	p.handleScrollInput()

	// Handle fragment selection.
	p.handleFragmentInput()

	// Handle escape to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	return true
}

// handleTabInput handles Tab key to switch tabs.
func (p *HuntTrackerPanel) handleTabInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			p.selectedTab--
			if p.selectedTab < 0 {
				p.selectedTab = 2
			}
		} else {
			p.selectedTab++
			if p.selectedTab > 2 {
				p.selectedTab = 0
			}
		}
		p.scrollOffset = 0
	}

	// Direct number keys.
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		p.selectedTab = 0
		p.scrollOffset = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		p.selectedTab = 1
		p.scrollOffset = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		p.selectedTab = 2
		p.scrollOffset = 0
	}
}

// handleScrollInput handles scrolling with arrow keys.
func (p *HuntTrackerPanel) handleScrollInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if p.scrollOffset > 0 {
			p.scrollOffset--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		p.scrollOffset++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		p.scrollOffset -= 5
		if p.scrollOffset < 0 {
			p.scrollOffset = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		p.scrollOffset += 5
	}
}

// handleFragmentInput handles fragment selection and claim attempts.
func (p *HuntTrackerPanel) handleFragmentInput() {
	if p.hunt == nil || p.selectedTab != 0 {
		return
	}

	// Number keys 0-9 for quick fragment selection.
	for i := 0; i <= 9; i++ {
		key := ebiten.Key0 + ebiten.Key(i)
		if inpututil.IsKeyJustPressed(key) {
			idx := i
			if idx < len(p.hunt.Fragments) {
				p.hunt.SelectedFragment = idx
				if p.onFragmentSelect != nil {
					p.onFragmentSelect(p.hunt.ID, idx)
				}
			}
		}
	}

	// Enter to attempt claim on selected fragment.
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && p.hunt.SelectedFragment >= 0 {
		if p.onClaimAttempt != nil {
			p.onClaimAttempt(p.hunt.ID, p.hunt.SelectedFragment)
		}
	}
}

// Draw renders the panel to the screen.
func (p *HuntTrackerPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	p.screenWidth = w
	p.screenHeight = h

	px, py := p.calculatePosition(w, h)
	px += int(p.slideOffset) // Slide from right.

	p.drawBackground(screen, px, py)
	p.drawTitle(screen, px, py)
	p.drawHuntInfo(screen, px, py)
	p.drawTabs(screen, px, py)
	p.drawTabContent(screen, px, py)
	p.drawTimer(screen, px, py)

	if p.errorMessage != "" {
		p.drawError(screen, px, py)
	}
}

// calculatePosition returns the panel's top-left corner.
func (p *HuntTrackerPanel) calculatePosition(screenW, screenH int) (int, int) {
	margin := 10

	switch p.position {
	case PositionRight:
		return screenW - p.width - margin, margin
	case PositionLeft:
		return margin, margin
	case PositionCenter:
		return (screenW - p.width) / 2, (screenH - p.height) / 2
	default:
		return screenW - p.width - margin, margin
	}
}

// drawBackground draws the panel background.
func (p *HuntTrackerPanel) drawBackground(screen *ebiten.Image, px, py int) {
	// Semi-transparent background.
	bgColor := color.RGBA{
		R: p.theme.PanelBackground.R,
		G: p.theme.PanelBackground.G,
		B: p.theme.PanelBackground.B,
		A: 230,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), bgColor, true)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 1.5, p.theme.PanelBorder, true)
}

// drawTitle draws the panel title.
func (p *HuntTrackerPanel) drawTitle(screen *ebiten.Image, px, py int) {
	titleH := 40
	titleBg := color.RGBA{
		R: p.theme.PanelBackground.R + 15,
		G: p.theme.PanelBackground.G + 15,
		B: p.theme.PanelBackground.B + 20,
		A: 255,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(titleH), titleBg, true)

	// Hunt icon (magnifying glass shape).
	iconX := px + 15
	iconY := py + 12
	iconColor := p.theme.AccentPrimary
	vector.DrawFilledCircle(screen, float32(iconX+8), float32(iconY+8), 7, iconColor, true)
	vector.StrokeCircle(screen, float32(iconX+8), float32(iconY+8), 7, 2, p.theme.PanelBackground, true)
	vector.StrokeLine(screen, float32(iconX+12), float32(iconY+12),
		float32(iconX+18), float32(iconY+18), 2, iconColor, true)

	// Title: "Specter Hunt" - text rendering requires font.
}

// drawHuntInfo draws basic hunt information.
func (p *HuntTrackerPanel) drawHuntInfo(screen *ebiten.Image, px, py int) {
	if p.hunt == nil {
		return
	}

	infoY := py + 45
	infoX := px + p.theme.Padding
	infoW := p.width - p.theme.Padding*2
	infoH := 60

	vector.DrawFilledRect(screen, float32(infoX), float32(infoY),
		float32(infoW), float32(infoH), p.theme.InputBackground, true)

	// Progress bar (claimed / total).
	progressY := infoY + 40
	progressW := infoW - 20
	progressH := 8
	vector.DrawFilledRect(screen, float32(infoX+10), float32(progressY),
		float32(progressW), float32(progressH), p.theme.ButtonBackground, true)

	if p.hunt.FragmentCount > 0 {
		fillW := float64(progressW) * float64(p.hunt.ClaimedCount) / float64(p.hunt.FragmentCount)
		vector.DrawFilledRect(screen, float32(infoX+10), float32(progressY),
			float32(fillW), float32(progressH), p.theme.AccentSecondary, true)
	}
}

// drawTabs draws the tab buttons.
func (p *HuntTrackerPanel) drawTabs(screen *ebiten.Image, px, py int) {
	tabY := py + 115
	tabW := (p.width - p.theme.Padding*2) / 3
	tabH := 32

	tabs := []string{"Fragments", "Clues", "Leaders"}
	for i := range tabs {
		tabX := px + p.theme.Padding + i*tabW

		tabBg := p.theme.ButtonBackground
		if i == p.selectedTab {
			tabBg = p.theme.AccentPrimary
		}

		vector.DrawFilledRect(screen, float32(tabX), float32(tabY),
			float32(tabW-2), float32(tabH), tabBg, true)
	}
}

// drawTabContent draws the content for the selected tab.
func (p *HuntTrackerPanel) drawTabContent(screen *ebiten.Image, px, py int) {
	contentY := py + 155
	contentX := px + p.theme.Padding
	contentW := p.width - p.theme.Padding*2
	contentH := p.height - 215

	// Content area background.
	vector.DrawFilledRect(screen, float32(contentX), float32(contentY),
		float32(contentW), float32(contentH), p.theme.InputBackground, true)

	if p.hunt == nil {
		return
	}

	switch p.selectedTab {
	case 0:
		p.drawFragmentsTab(screen, contentX, contentY, contentW, contentH)
	case 1:
		p.drawCluesTab(screen, contentX, contentY, contentW, contentH)
	case 2:
		p.drawLeaderboardTab(screen, contentX, contentY, contentW, contentH)
	}
}

// drawFragmentsTab draws the fragment list.
func (p *HuntTrackerPanel) drawFragmentsTab(screen *ebiten.Image, x, y, w, h int) {
	itemH := 36
	visibleItems := h / itemH

	startIdx := p.scrollOffset
	if startIdx > len(p.hunt.Fragments)-visibleItems {
		startIdx = len(p.hunt.Fragments) - visibleItems
		if startIdx < 0 {
			startIdx = 0
		}
	}

	for i := 0; i < visibleItems && startIdx+i < len(p.hunt.Fragments); i++ {
		frag := p.hunt.Fragments[startIdx+i]
		itemY := y + i*itemH + 4

		// Item background.
		itemBg := color.RGBA{40, 42, 50, 255}
		if frag.Claimed {
			if frag.ClaimedByMe {
				itemBg = color.RGBA{40, 60, 40, 255} // Green tint for my claims.
			} else {
				itemBg = color.RGBA{50, 45, 45, 255} // Red tint for others' claims.
			}
		} else if p.hunt.SelectedFragment == startIdx+i {
			itemBg = color.RGBA{50, 60, 80, 255} // Highlight selected.
		}

		vector.DrawFilledRect(screen, float32(x+4), float32(itemY),
			float32(w-8), float32(itemH-4), itemBg, true)

		// Fragment icon.
		iconX := x + 12
		iconY := itemY + 10
		iconColor := p.theme.AccentPrimary
		if frag.Claimed {
			iconColor = p.theme.TextSecondary
		} else {
			// Pulse effect for unclaimed.
			pulse := 0.5 + 0.5*float64(1+int(p.pulseTime*2)%2)
			iconColor = color.RGBA{
				R: uint8(float64(p.theme.AccentPrimary.R) * pulse),
				G: uint8(float64(p.theme.AccentPrimary.G) * pulse),
				B: uint8(float64(p.theme.AccentPrimary.B) * pulse),
				A: 255,
			}
		}
		vector.DrawFilledRect(screen, float32(iconX), float32(iconY),
			12, 12, iconColor, true)

		// Status indicator.
		statusX := x + w - 30
		statusY := itemY + 12
		if frag.Claimed {
			// Checkmark shape.
			vector.StrokeLine(screen, float32(statusX), float32(statusY+4),
				float32(statusX+4), float32(statusY+8), 2, p.theme.AccentSecondary, true)
			vector.StrokeLine(screen, float32(statusX+4), float32(statusY+8),
				float32(statusX+10), float32(statusY), 2, p.theme.AccentSecondary, true)
		} else {
			// Empty circle.
			vector.StrokeCircle(screen, float32(statusX+5), float32(statusY+4), 5, 1.5, p.theme.TextSecondary, true)
		}
	}

	// Scroll indicators.
	if startIdx > 0 {
		vector.DrawFilledRect(screen, float32(x+w/2-10), float32(y+2), 20, 3, p.theme.AccentPrimary, true)
	}
	if startIdx+visibleItems < len(p.hunt.Fragments) {
		vector.DrawFilledRect(screen, float32(x+w/2-10), float32(y+h-5), 20, 3, p.theme.AccentPrimary, true)
	}
}

// drawCluesTab draws the clues for the selected fragment.
func (p *HuntTrackerPanel) drawCluesTab(screen *ebiten.Image, x, y, w, h int) {
	if p.hunt.SelectedFragment < 0 || p.hunt.SelectedFragment >= len(p.hunt.Fragments) {
		// No fragment selected - draw hint.
		hintY := y + h/2 - 10
		hintBg := color.RGBA{50, 52, 60, 200}
		vector.DrawFilledRect(screen, float32(x+20), float32(hintY),
			float32(w-40), 30, hintBg, true)
		return
	}

	frag := p.hunt.Fragments[p.hunt.SelectedFragment]
	clueH := 50

	for i := range frag.Clues {
		if i >= 4 { // Max 4 clues.
			break
		}
		clueY := y + 10 + i*clueH

		// Clue background.
		clueBg := color.RGBA{45, 47, 55, 255}
		vector.DrawFilledRect(screen, float32(x+8), float32(clueY),
			float32(w-16), float32(clueH-6), clueBg, true)

		// Clue number indicator.
		numX := x + 16
		numY := clueY + 8
		numColor := p.theme.AccentSecondary
		vector.DrawFilledCircle(screen, float32(numX+8), float32(numY+8), 10, numColor, true)
	}
}

// drawLeaderboardTab draws the leaderboard.
func (p *HuntTrackerPanel) drawLeaderboardTab(screen *ebiten.Image, x, y, w, h int) {
	itemH := 40
	visibleItems := h / itemH

	for i := 0; i < visibleItems && i < len(p.hunt.Leaderboard); i++ {
		entry := p.hunt.Leaderboard[i]
		itemY := y + i*itemH + 4

		// Item background.
		itemBg := color.RGBA{40, 42, 50, 255}
		if entry.IsMe {
			itemBg = color.RGBA{50, 60, 70, 255}
		}

		vector.DrawFilledRect(screen, float32(x+4), float32(itemY),
			float32(w-8), float32(itemH-4), itemBg, true)

		// Rank indicator.
		rankX := x + 12
		rankY := itemY + 10
		rankColor := p.theme.TextSecondary
		if i == 0 {
			rankColor = color.RGBA{255, 215, 0, 255} // Gold.
		} else if i == 1 {
			rankColor = color.RGBA{192, 192, 192, 255} // Silver.
		} else if i == 2 {
			rankColor = color.RGBA{205, 127, 50, 255} // Bronze.
		}
		vector.DrawFilledCircle(screen, float32(rankX+10), float32(rankY+10), 12, rankColor, true)

		// Claims bar.
		maxClaims := p.hunt.FragmentCount
		if maxClaims > 0 {
			barX := x + 80
			barY := itemY + itemH/2 - 4
			barW := w - 110
			barH := 8
			vector.DrawFilledRect(screen, float32(barX), float32(barY),
				float32(barW), float32(barH), p.theme.ButtonBackground, true)

			fillW := float64(barW) * float64(entry.Claims) / float64(maxClaims)
			vector.DrawFilledRect(screen, float32(barX), float32(barY),
				float32(fillW), float32(barH), p.theme.AccentPrimary, true)
		}
	}
}

// drawTimer draws the countdown timer.
func (p *HuntTrackerPanel) drawTimer(screen *ebiten.Image, px, py int) {
	if p.hunt == nil {
		return
	}

	timerY := py + p.height - 50
	timerX := px + p.theme.Padding
	timerW := p.width - p.theme.Padding*2
	timerH := 40

	remaining := time.Until(p.hunt.ExpiresAt)
	if remaining < 0 {
		remaining = 0
	}

	// Timer background.
	timerBg := p.theme.InputBackground
	if remaining < 5*time.Minute {
		timerBg = color.RGBA{80, 40, 40, 255}
	}
	vector.DrawFilledRect(screen, float32(timerX), float32(timerY),
		float32(timerW), float32(timerH), timerBg, true)

	// Timer icon (clock shape).
	clockX := timerX + 15
	clockY := timerY + 12
	clockColor := p.theme.TextPrimary
	if remaining < 5*time.Minute {
		clockColor = p.theme.TextError
	}
	vector.StrokeCircle(screen, float32(clockX+8), float32(clockY+8), 8, 1.5, clockColor, true)
	vector.StrokeLine(screen, float32(clockX+8), float32(clockY+8),
		float32(clockX+8), float32(clockY+4), 1.5, clockColor, true)
	vector.StrokeLine(screen, float32(clockX+8), float32(clockY+8),
		float32(clockX+12), float32(clockY+8), 1.5, clockColor, true)
}

// drawError draws the error message.
func (p *HuntTrackerPanel) drawError(screen *ebiten.Image, px, py int) {
	errorY := py + p.height - 90
	errorBg := color.RGBA{80, 40, 40, 220}
	vector.DrawFilledRect(screen, float32(px+p.theme.Padding), float32(errorY),
		float32(p.width-p.theme.Padding*2), 28, errorBg, true)
}

// Getters for testing.

// GetHunt returns the current hunt info.
func (p *HuntTrackerPanel) GetHunt() *HuntInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hunt
}

// GetSelectedTab returns the selected tab index.
func (p *HuntTrackerPanel) GetSelectedTab() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedTab
}

// SetSelectedTab sets the selected tab index.
func (p *HuntTrackerPanel) SetSelectedTab(tab int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if tab >= 0 && tab <= 2 {
		p.selectedTab = tab
		p.scrollOffset = 0
	}
}

// GetSelectedFragment returns the selected fragment index from the current hunt.
func (p *HuntTrackerPanel) GetSelectedFragment() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.hunt == nil {
		return -1
	}
	return p.hunt.SelectedFragment
}

// SetSelectedFragment sets the selected fragment index.
func (p *HuntTrackerPanel) SetSelectedFragment(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hunt == nil {
		return
	}
	if idx >= -1 && idx < len(p.hunt.Fragments) {
		p.hunt.SelectedFragment = idx
	}
}

// SetError sets the error message.
func (p *HuntTrackerPanel) SetError(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorMessage = msg
	p.errorTime = 0
}
