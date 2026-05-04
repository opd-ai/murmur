// Package ui provides stub types for the Settings panel.
//
//go:build test
// +build test

package ui

import (
	"sync"
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
	Options     []string
	Min, Max    float64
}

// SettingType indicates the type of setting control.
type SettingType int

const (
	SettingTypeToggle SettingType = iota
	SettingTypeSlider
	SettingTypeSelect
	SettingTypeText
)

// SettingsPanel provides a UI for application settings (stub).
type SettingsPanel struct {
	mu sync.RWMutex

	visible    bool
	width      int
	height     int
	position   PanelPosition
	categories []SettingCategory
	selected   int
	onChange   SettingsChangeCallback
	theme      Theme
}

// NewSettingsPanel creates a new settings panel (stub).
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

// Update handles input and updates panel state (stub).
func (p *SettingsPanel) Update() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visible
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
		return ""
	default:
		return ""
	}
}

// Categories returns the setting categories.
func (p *SettingsPanel) Categories() []SettingCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.categories
}
