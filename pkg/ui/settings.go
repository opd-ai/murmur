// Package ui provides the Settings panel for configuration.
//

package ui

import (
	"image/color"
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
}

// NewSettingsPanel creates a new settings panel.
func NewSettingsPanel(theme Theme, onChange SettingsChangeCallback) *SettingsPanel {
	return &SettingsPanel{
		theme:    theme,
		onChange: onChange,
		width:    500,
		height:   400,
		position: PositionCenter,
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

	p.animTime += 1.0 / 60.0

	// Handle escape to close.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.visible = false
		return true
	}

	// Handle category navigation.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			p.selected = (p.selected - 1 + len(p.categories)) % len(p.categories)
		} else {
			p.selected = (p.selected + 1) % len(p.categories)
		}
	}

	// Handle scrolling.
	_, dy := ebiten.Wheel()
	p.scrollY -= int(dy * 30)
	if p.scrollY < 0 {
		p.scrollY = 0
	}

	return true
}

// Draw renders the panel.
func (p *SettingsPanel) Draw(screen *ebiten.Image) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.visible {
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	px := (w - p.width) / 2
	py := (h - p.height) / 2

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

		// Tab text would be rendered with text/v2.
		_ = cat.Name
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

// drawSelect draws a dropdown selector.
func (p *SettingsPanel) drawSelect(screen *ebiten.Image, x, y, width int, value string, options []string) {
	selectW := width - 20
	selectH := 30
	selectX := x + 10
	selectY := y + 10

	vector.DrawFilledRect(screen, float32(selectX), float32(selectY),
		float32(selectW), float32(selectH), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(selectX), float32(selectY),
		float32(selectW), float32(selectH), 1.0, p.theme.PanelBorder, true)

	// Value text would be rendered with text/v2.
	_ = value
	_ = options
}

// drawTextInput draws a text input field.
func (p *SettingsPanel) drawTextInput(screen *ebiten.Image, x, y, width int, value string) {
	inputW := width - 20
	inputH := 30
	inputX := x + 10
	inputY := y + 10

	vector.DrawFilledRect(screen, float32(inputX), float32(inputY),
		float32(inputW), float32(inputH), p.theme.InputBackground, true)
	vector.StrokeRect(screen, float32(inputX), float32(inputY),
		float32(inputW), float32(inputH), 1.0, p.theme.PanelBorder, true)

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
		for j, s := range cat.Settings {
			if s.Key == key {
				p.categories[i].Settings[j].Value = value
				if p.onChange != nil {
					// Convert value to string for callback.
					var strValue string
					switch v := value.(type) {
					case bool:
						if v {
							strValue = "true"
						} else {
							strValue = "false"
						}
					case float64:
						strValue = ""
					case string:
						strValue = v
					}
					p.onChange(key, strValue)
				}
				return
			}
		}
	}
}

// Categories returns the setting categories (for testing).
func (p *SettingsPanel) Categories() []SettingCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.categories
}
