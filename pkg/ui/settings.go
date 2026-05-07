// Package ui provides the Settings panel for configuration.
//

//go:build !test
// +build !test

package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SettingCategory groups related settings.
type SettingCategory struct {
	Name     string
	Settings []Setting
}

// Setting represents a configurable option.
type Setting struct {
	Key         string
	Label       string
	Description string
	Type        SettingType
	Value       interface{}
	Options     []string // For SettingTypeSelect
	Min, Max    float64  // For SettingTypeSlider
}

// SettingType indicates the type of setting control.
type SettingType int

const (
	SettingTypeToggle SettingType = iota
	SettingTypeSlider
	SettingTypeSelect
	SettingTypeText
)

// SettingsPanel provides a UI for application settings.
type SettingsPanel struct {
	mu sync.RWMutex

	visible    bool
	x, y       int
	width      int
	height     int
	position   PanelPosition
	categories []SettingCategory
	selected   int // Currently selected category index.
	scrollY    int
	onChange   SettingsChangeCallback
	theme      Theme
	animTime   float64

	textFocusIndex int // -1 means no text control focused.
	sliderDragIdx  int // -1 means no slider drag in progress.
}

// NewSettingsPanel creates a new settings panel.
func NewSettingsPanel(theme Theme, onChange SettingsChangeCallback) *SettingsPanel {
	return &SettingsPanel{
		theme:          theme,
		onChange:       onChange,
		width:          500,
		height:         400,
		position:       PositionCenter,
		textFocusIndex: -1,
		sliderDragIdx:  -1,
		categories: []SettingCategory{
			{
				Name: "Network",
				Settings: []Setting{
					{Key: "dht_enabled", Label: "Enable DHT", Type: SettingTypeToggle, Value: true},
					{Key: "relay_enabled", Label: "Act as Relay", Type: SettingTypeToggle, Value: false},
					{Key: "max_peers", Label: "Max Peers", Type: SettingTypeSlider, Value: 50.0, Min: 6, Max: 100},
				},
			},
			{
				Name: "Privacy",
				Settings: []Setting{
					{
						Key: "privacy_mode", Label: "Privacy Mode", Type: SettingTypeSelect,
						Value: "Hybrid", Options: []string{"Open", "Hybrid", "Guarded", "Fortress"},
					},
					{Key: "specter_enabled", Label: "Enable Specter", Type: SettingTypeToggle, Value: true},
					{Key: "shroud_circuits", Label: "Shroud Circuits", Type: SettingTypeSlider, Value: 3.0, Min: 1, Max: 5},
				},
			},
			{
				Name: "Devices",
				Settings: []Setting{
					{Key: "device_count", Label: "Authorized Devices", Type: SettingTypeText, Value: "1"},
					{Key: "show_devices", Label: "Manage Devices", Type: SettingTypeToggle, Value: false},
				},
			},
			{
				Name: "Display",
				Settings: []Setting{
					{
						Key: "theme", Label: "Theme", Type: SettingTypeSelect,
						Value: "Dark", Options: []string{"Dark", "Light", "Abyss"},
					},
					{Key: "animations", Label: "Animations", Type: SettingTypeToggle, Value: true},
					{Key: "node_labels", Label: "Show Node Labels", Type: SettingTypeToggle, Value: true},
				},
			},
			{
				Name: "Waves",
				Settings: []Setting{
					{Key: "default_ttl", Label: "Default TTL (days)", Type: SettingTypeSlider, Value: 7.0, Min: 1, Max: 30},
					{Key: "pow_difficulty", Label: "PoW Difficulty", Type: SettingTypeSlider, Value: 20.0, Min: 16, Max: 24},
					{Key: "auto_amplify", Label: "Auto-Amplify", Type: SettingTypeToggle, Value: false},
				},
			},
		},
	}
}

// Visible returns true if the panel is shown.
func (p *SettingsPanel) Visible() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.visible
}

// Show displays the panel.
func (p *SettingsPanel) Show() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = true
	p.animTime = 0
}

// Hide hides the panel.
func (p *SettingsPanel) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visible = false
}

// Toggle toggles panel visibility.
func (p *SettingsPanel) Toggle() {
	if p.Visible() {
		p.Hide()
	} else {
		p.Show()
	}
}

// Update handles input and updates panel state.
func (p *SettingsPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.visible {
		return false
	}

	p.applyResponsiveLayout()

	p.animTime += FrameTime

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	p.handleCategoryNavigation()
	p.handleScrolling()
	p.handlePointerInput()
	p.handleFocusedTextInput()

	return true
}

func (p *SettingsPanel) handlePointerInput() {
	mx, my := ebiten.CursorPosition()
	panelX, panelY := p.panelOrigin()

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		p.sliderDragIdx = -1
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if p.handleCategoryTabClick(mx, my, panelX, panelY) {
			p.textFocusIndex = -1
			p.sliderDragIdx = -1
			return
		}
		if p.handleSettingControlClick(mx, my, panelX, panelY) {
			return
		}
		p.textFocusIndex = -1
	}

	if p.sliderDragIdx >= 0 && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p.updateSliderValueAt(p.sliderDragIdx, mx, panelX)
	}
}

func (p *SettingsPanel) handleCategoryTabClick(mx, my, panelX, panelY int) bool {
	tabsY := panelY + 50
	if my < tabsY || my >= tabsY+35 || mx < panelX || mx >= panelX+p.width {
		return false
	}
	tabWidth := p.width / len(p.categories)
	idx := (mx - panelX) / tabWidth
	if idx >= 0 && idx < len(p.categories) {
		p.selected = idx
		return true
	}
	return false
}

func (p *SettingsPanel) handleSettingControlClick(mx, my, panelX, panelY int) bool {
	if p.selected >= len(p.categories) {
		return false
	}

	const settingHeight = 50
	settingsY := panelY + 90
	padding := p.theme.Padding
	rowX := panelX + padding
	rowW := p.width - padding*2
	labelW := rowW / 2
	controlX := rowX + labelW
	controlW := rowW - labelW

	cat := &p.categories[p.selected]
	for i := range cat.Settings {
		settingY := settingsY + i*settingHeight - p.scrollY
		if settingY < settingsY-settingHeight || settingY > settingsY+p.height {
			continue
		}
		if mx < controlX || mx >= controlX+controlW || my < settingY || my >= settingY+settingHeight {
			continue
		}
		return p.activateSettingControl(i, mx, my, controlX, settingY, controlW)
	}
	return false
}

func (p *SettingsPanel) activateSettingControl(idx, mx, my, controlX, settingY, controlW int) bool {
	cat := &p.categories[p.selected]
	if idx < 0 || idx >= len(cat.Settings) {
		return false
	}

	s := &cat.Settings[idx]
	switch s.Type {
	case SettingTypeToggle:
		if v, ok := s.Value.(bool); ok {
			s.Value = !v
			p.notifyOnChange(s.Key, s.Value)
			return true
		}
	case SettingTypeSlider:
		p.sliderDragIdx = idx
		p.textFocusIndex = -1
		p.updateSliderValueAt(idx, mx, p.panelOriginXForControl(controlX))
		return true
	case SettingTypeSelect:
		if len(s.Options) == 0 {
			return true
		}
		current, _ := s.Value.(string)
		next := 0
		for i, opt := range s.Options {
			if opt == current {
				next = (i + 1) % len(s.Options)
				break
			}
		}
		s.Value = s.Options[next]
		p.notifyOnChange(s.Key, s.Value)
		return true
	case SettingTypeText:
		p.textFocusIndex = idx
		p.sliderDragIdx = -1
		return true
	}
	_ = my
	_ = controlW
	return false
}

func (p *SettingsPanel) panelOrigin() (int, int) {
	sw, sh := ebiten.WindowSize()
	return (sw - p.width) / 2, (sh - p.height) / 2
}

func (p *SettingsPanel) applyResponsiveLayout() {
	sw, sh := ebiten.WindowSize()

	if sw <= 768 {
		p.width = sw - 24
		if p.width < 320 {
			p.width = 320
		}
		p.height = sh - 24
		if p.height < 320 {
			p.height = 320
		}
		if p.height > 640 {
			p.height = 640
		}
		p.position = PositionCenter
		return
	}

	if sw <= 1024 {
		p.width = sw - 40
		if p.width > 640 {
			p.width = 640
		}
		if p.width < 420 {
			p.width = 420
		}
		p.height = sh - 40
		if p.height > 520 {
			p.height = 520
		}
		if p.height < 360 {
			p.height = 360
		}
		p.position = PositionCenter
		return
	}

	p.width = 500
	p.height = 400
	p.position = PositionCenter
}

func (p *SettingsPanel) panelOriginXForControl(controlX int) int {
	_ = controlX
	px, _ := p.panelOrigin()
	return px
}

func (p *SettingsPanel) updateSliderValueAt(settingIdx, mouseX, panelX int) {
	if p.selected >= len(p.categories) {
		return
	}
	cat := &p.categories[p.selected]
	if settingIdx < 0 || settingIdx >= len(cat.Settings) {
		return
	}
	s := &cat.Settings[settingIdx]
	if s.Type != SettingTypeSlider {
		return
	}

	padding := p.theme.Padding
	rowW := p.width - padding*2
	labelW := rowW / 2
	controlX := panelX + padding + labelW
	controlW := rowW - labelW
	sliderX := controlX + 10
	sliderW := controlW - 20
	if sliderW <= 0 || s.Max <= s.Min {
		return
	}

	ratio := float64(mouseX-sliderX) / float64(sliderW)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	value := s.Min + ratio*(s.Max-s.Min)
	s.Value = value
	p.notifyOnChange(s.Key, s.Value)
}

func (p *SettingsPanel) handleFocusedTextInput() {
	if p.selected >= len(p.categories) || p.textFocusIndex < 0 {
		return
	}
	cat := &p.categories[p.selected]
	if p.textFocusIndex >= len(cat.Settings) {
		p.textFocusIndex = -1
		return
	}
	s := &cat.Settings[p.textFocusIndex]
	if s.Type != SettingTypeText {
		p.textFocusIndex = -1
		return
	}

	value, _ := s.Value.(string)
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(value) > 0 {
		s.Value = value[:len(value)-1]
		p.notifyOnChange(s.Key, s.Value)
		return
	}
	chars := ebiten.AppendInputChars(nil)
	if len(chars) == 0 {
		return
	}
	s.Value = value + string(chars)
	p.notifyOnChange(s.Key, s.Value)
}

// handleCategoryNavigation processes Tab/Shift+Tab for category switching.
func (p *SettingsPanel) handleCategoryNavigation() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		return
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		p.selected = (p.selected - 1 + len(p.categories)) % len(p.categories)
	} else {
		p.selected = (p.selected + 1) % len(p.categories)
	}
}

// handleScrolling processes mouse wheel scrolling with bounds clamping.
func (p *SettingsPanel) handleScrolling() {
	_, dy := ebiten.Wheel()
	p.scrollY -= int(dy * 30)
	if p.scrollY < 0 {
		p.scrollY = 0
	}

	const (
		titleAreaH = 50 + 40
		settingH   = 50
	)
	if p.selected < len(p.categories) {
		maxScroll := len(p.categories[p.selected].Settings)*settingH - (p.height - titleAreaH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.scrollY > maxScroll {
			p.scrollY = maxScroll
		}
	}
}

// Draw renders the panel.
func (p *SettingsPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	px, py, w, h, shouldRender := CheckPanelVisibilityAndCenter(screen, p.visible, p.width, p.height)
	if !shouldRender {
		return
	}

	// Draw overlay.
	overlayColor := color.RGBA{0, 0, 0, 150}
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), overlayColor, true)

	// Draw panel background.
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), p.theme.PanelBackground, true)
	vector.StrokeRect(screen, float32(px), float32(py),
		float32(p.width), float32(p.height), 2.0, p.theme.PanelBorder, true)

	// Draw title.
	titleHeight := 50
	titleBg := color.RGBA{
		R: p.theme.PanelBackground.R + 15,
		G: p.theme.PanelBackground.G + 15,
		B: p.theme.PanelBackground.B + 20,
		A: 255,
	}
	vector.DrawFilledRect(screen, float32(px), float32(py),
		float32(p.width), float32(titleHeight), titleBg, true)

	// Draw category tabs.
	p.drawCategoryTabs(screen, px, py+titleHeight)

	// Draw settings for selected category.
	p.drawSettings(screen, px, py+titleHeight+40)
}

// drawCategoryTabs draws the category navigation tabs.
func (p *SettingsPanel) drawCategoryTabs(screen *ebiten.Image, px, py int) {
	tabWidth := p.width / len(p.categories)

	for i, cat := range p.categories {
		tabX := px + i*tabWidth
		tabBg := p.theme.ButtonBackground
		if i == p.selected {
			tabBg = p.theme.AccentPrimary
		}

		vector.DrawFilledRect(screen, float32(tabX), float32(py),
			float32(tabWidth-2), 35, tabBg, true)

		// Render the tab label centred inside the tab rectangle.
		tabCenterX := float64(tabX) + float64(tabWidth-2)/2
		tabCenterY := float64(py) + 17.5
		drawUICenteredText(screen, cat.Name, tabCenterX, tabCenterY, p.theme.TextPrimary)
	}
}

// drawSettings draws the settings for the selected category.
func (p *SettingsPanel) drawSettings(screen *ebiten.Image, px, py int) {
	if p.selected >= len(p.categories) {
		return
	}

	cat := p.categories[p.selected]
	settingHeight := 50
	padding := p.theme.Padding

	for i, setting := range cat.Settings {
		settingY := py + i*settingHeight - p.scrollY

		// Skip if scrolled out of view.
		if settingY < py-settingHeight || settingY > py+p.height {
			continue
		}

		// Draw setting row.
		p.drawSettingRow(screen, px+padding, settingY, p.width-padding*2, setting)
	}
}

// drawSettingRow draws a single setting control.
func (p *SettingsPanel) drawSettingRow(screen *ebiten.Image, x, y, width int, setting Setting) {
	// Label on the left.
	labelWidth := width / 2

	// Control on the right.
	controlX := x + labelWidth
	controlWidth := width - labelWidth

	switch setting.Type {
	case SettingTypeToggle:
		p.drawToggle(screen, controlX, y, controlWidth, setting.Value.(bool))
	case SettingTypeSlider:
		p.drawSlider(screen, controlX, y, controlWidth, setting.Value.(float64), setting.Min, setting.Max)
	case SettingTypeSelect:
		p.drawSelect(screen, controlX, y, controlWidth, setting.Value.(string), setting.Options)
	case SettingTypeText:
		p.drawTextInput(screen, controlX, y, controlWidth, setting.Value.(string))
	}
}

// drawToggle draws a toggle switch.
func (p *SettingsPanel) drawToggle(screen *ebiten.Image, x, y, width int, value bool) {
	toggleW := 50
	toggleH := 26
	toggleX := x + width - toggleW - 10
	toggleY := y + 12

	// Background.
	bg := p.theme.ButtonBackground
	if value {
		bg = p.theme.AccentPrimary
	}
	vector.DrawFilledRect(screen, float32(toggleX), float32(toggleY),
		float32(toggleW), float32(toggleH), bg, true)

	// Knob.
	knobX := toggleX + 3
	if value {
		knobX = toggleX + toggleW - 23
	}
	vector.DrawFilledCircle(screen, float32(knobX+10), float32(toggleY+toggleH/2),
		float32(toggleH/2-3), p.theme.TextPrimary, true)
}

// drawSlider draws a slider control.
func (p *SettingsPanel) drawSlider(screen *ebiten.Image, x, y, width int, value, min, max float64) {
	sliderW := width - 20
	sliderH := 8
	sliderX := x + 10
	sliderY := y + 20

	// Track.
	vector.DrawFilledRect(screen, float32(sliderX), float32(sliderY),
		float32(sliderW), float32(sliderH), p.theme.InputBackground, true)

	// Fill.
	fillRatio := (value - min) / (max - min)
	fillW := int(float64(sliderW) * fillRatio)
	vector.DrawFilledRect(screen, float32(sliderX), float32(sliderY),
		float32(fillW), float32(sliderH), p.theme.AccentPrimary, true)

	// Knob.
	knobX := sliderX + fillW
	vector.DrawFilledCircle(screen, float32(knobX), float32(sliderY+sliderH/2),
		10, p.theme.TextPrimary, true)
}

// drawInputBox draws a bordered input box at the specified position.
func (p *SettingsPanel) drawInputBox(screen *ebiten.Image, x, y, width, height int) {
	boxW := width - 20
	boxX := x + 10
	boxY := y + 10

	vector.DrawFilledRect(screen, float32(boxX), float32(boxY),
		float32(boxW), float32(height), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(boxX), float32(boxY),
		float32(boxW), float32(height), 1.0, p.theme.PanelBorder, true)
}

// drawSelect draws a dropdown selector.
func (p *SettingsPanel) drawSelect(screen *ebiten.Image, x, y, width int, value string, options []string) {
	p.drawInputBox(screen, x, y, width, 30)
	// Value text would be rendered with text/v2.
	_ = value
	_ = options
}

// drawTextInput draws a text input field.
func (p *SettingsPanel) drawTextInput(screen *ebiten.Image, x, y, width int, value string) {
	p.drawInputBox(screen, x, y, width, 30)
	// Value text would be rendered with text/v2.
	_ = value
}

// GetSetting returns the value of a setting by key.
func (p *SettingsPanel) GetSetting(key string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, cat := range p.categories {
		for _, s := range cat.Settings {
			if s.Key == key {
				return s.Value
			}
		}
	}
	return nil
}

// SetSetting updates the value of a setting by key.
func (p *SettingsPanel) SetSetting(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, cat := range p.categories {
		if p.updateSettingInCategory(i, key, value, &cat) {
			return
		}
	}
}

// updateSettingInCategory searches for and updates a setting within a category.
func (p *SettingsPanel) updateSettingInCategory(catIdx int, key string, value interface{}, cat *SettingCategory) bool {
	for j, s := range cat.Settings {
		if s.Key == key {
			p.categories[catIdx].Settings[j].Value = value
			p.notifyOnChange(key, value)
			return true
		}
	}
	return false
}

// notifyOnChange invokes the onChange callback with the setting key and string value.
func (p *SettingsPanel) notifyOnChange(key string, value interface{}) {
	if p.onChange == nil {
		return
	}
	p.onChange(key, p.convertValueToString(value))
}

// convertValueToString converts an interface{} value to a string representation.
func (p *SettingsPanel) convertValueToString(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Categories returns the setting categories (for testing).
func (p *SettingsPanel) Categories() []SettingCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.categories
}
