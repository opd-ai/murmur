// Package ui provides stub types for the Settings panel.
//
//go:build test
// +build test

package ui

import (
	"fmt"
	"strconv"
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
	Value       SettingValue
	Options     []string
	Min, Max    float64
}

// SettingValue stores typed setting values without interface{} storage.
type SettingValue struct {
	Toggle bool
	Slider float64
	Text   string
}

// ToggleValue creates a toggle setting value.
func ToggleValue(v bool) SettingValue {
	return SettingValue{Toggle: v}
}

// SliderValue creates a slider setting value.
func SliderValue(v float64) SettingValue {
	return SettingValue{Slider: v}
}

// TextValue creates a text/select setting value.
func TextValue(v string) SettingValue {
	return SettingValue{Text: v}
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
					{Key: "dht_enabled", Label: "Enable DHT", Type: SettingTypeToggle, Value: ToggleValue(true)},
					{Key: "relay_enabled", Label: "Act as Relay", Type: SettingTypeToggle, Value: ToggleValue(false)},
					{Key: "max_peers", Label: "Max Peers", Type: SettingTypeSlider, Value: SliderValue(50.0), Min: 6, Max: 100},
				},
			},
			{
				Name: "Privacy",
				Settings: []Setting{
					{
						Key: "privacy_mode", Label: "Privacy Mode", Type: SettingTypeSelect,
						Value: TextValue("Hybrid"), Options: []string{"Open", "Hybrid", "Guarded", "Fortress"},
					},
					{Key: "specter_enabled", Label: "Enable Specter", Type: SettingTypeToggle, Value: ToggleValue(true)},
					{Key: "shroud_circuits", Label: "Shroud Circuits", Type: SettingTypeSlider, Value: SliderValue(3.0), Min: 1, Max: 5},
				},
			},
			{
				Name: "Display",
				Settings: []Setting{
					{
						Key: "theme", Label: "Theme", Type: SettingTypeSelect,
						Value: TextValue("Dark"), Options: []string{"Dark", "Light", "Abyss"},
					},
					{Key: "animations", Label: "Animations", Type: SettingTypeToggle, Value: ToggleValue(true)},
					{Key: "node_labels", Label: "Show Node Labels", Type: SettingTypeToggle, Value: ToggleValue(true)},
				},
			},
			{
				Name: "Waves",
				Settings: []Setting{
					{Key: "default_ttl", Label: "Default TTL (days)", Type: SettingTypeSlider, Value: SliderValue(7.0), Min: 1, Max: 30},
					{Key: "pow_difficulty", Label: "PoW Difficulty", Type: SettingTypeSlider, Value: SliderValue(20.0), Min: 16, Max: 24},
					{Key: "auto_amplify", Label: "Auto-Amplify", Type: SettingTypeToggle, Value: ToggleValue(false)},
				},
			},
			{
				Name: "Devices",
				Settings: []Setting{
					{Key: "device_count", Label: "Authorized Devices", Type: SettingTypeText, Value: TextValue("1")},
					{Key: "device_sync", Label: "Sync Identity Across Devices", Type: SettingTypeToggle, Value: ToggleValue(true)},
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
// Deprecated: Use GetSettingBool, GetSettingSlider, or GetSettingText.
func (p *SettingsPanel) GetSetting(key string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, cat := range p.categories {
		for _, s := range cat.Settings {
			if s.Key == key {
				return valueForType(s.Type, s.Value)
			}
		}
	}
	return nil
}

// SetSetting updates the value of a setting by key.
// Deprecated: Use SetSettingBool, SetSettingSlider, or SetSettingText.
func (p *SettingsPanel) SetSetting(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, cat := range p.categories {
		if p.updateSettingInCategory(i, key, value, &cat) {
			return
		}
	}
}

// SetSettingBool updates a toggle setting by key.
func (p *SettingsPanel) SetSettingBool(key string, value bool) bool {
	return p.setTypedSetting(key, SettingTypeToggle, ToggleValue(value))
}

// SetSettingSlider updates a slider setting by key.
func (p *SettingsPanel) SetSettingSlider(key string, value float64) bool {
	return p.setTypedSetting(key, SettingTypeSlider, SliderValue(value))
}

// SetSettingText updates a text/select setting by key.
func (p *SettingsPanel) SetSettingText(key, value string) bool {
	return p.setTypedSetting(key, SettingTypeText, TextValue(value)) ||
		p.setTypedSetting(key, SettingTypeSelect, TextValue(value))
}

// updateSettingInCategory searches for and updates a setting within a category.
func (p *SettingsPanel) updateSettingInCategory(catIdx int, key string, value interface{}, cat *SettingCategory) bool {
	for j, s := range cat.Settings {
		if s.Key == key {
			typedValue, ok := settingValueFromType(s.Type, value)
			if !ok {
				return false
			}
			p.categories[catIdx].Settings[j].Value = typedValue
			p.notifyOnChange(key, valueForType(s.Type, typedValue))
			return true
		}
	}
	return false
}

// GetSettingBool returns a toggle setting and whether it exists with the expected type.
func (p *SettingsPanel) GetSettingBool(key string) (bool, bool) {
	if setting, ok := p.getSettingByKey(key); ok && setting.Type == SettingTypeToggle {
		return setting.Value.Toggle, true
	}
	return false, false
}

// GetSettingSlider returns a slider setting and whether it exists with the expected type.
func (p *SettingsPanel) GetSettingSlider(key string) (float64, bool) {
	if setting, ok := p.getSettingByKey(key); ok && setting.Type == SettingTypeSlider {
		return setting.Value.Slider, true
	}
	return 0, false
}

// GetSettingText returns a text/select setting and whether it exists with the expected type.
func (p *SettingsPanel) GetSettingText(key string) (string, bool) {
	if setting, ok := p.getSettingByKey(key); ok && (setting.Type == SettingTypeText || setting.Type == SettingTypeSelect) {
		return setting.Value.Text, true
	}
	return "", false
}

func (p *SettingsPanel) getSettingByKey(key string) (Setting, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, cat := range p.categories {
		for _, setting := range cat.Settings {
			if setting.Key == key {
				return setting, true
			}
		}
	}
	return Setting{}, false
}

func (p *SettingsPanel) setTypedSetting(key string, expected SettingType, value SettingValue) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, cat := range p.categories {
		for j, setting := range cat.Settings {
			if setting.Key == key && setting.Type == expected {
				p.categories[i].Settings[j].Value = value
				p.notifyOnChange(key, valueForType(expected, value))
				return true
			}
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

func valueForType(settingType SettingType, value SettingValue) interface{} {
	switch settingType {
	case SettingTypeToggle:
		return value.Toggle
	case SettingTypeSlider:
		return value.Slider
	case SettingTypeSelect, SettingTypeText:
		return value.Text
	default:
		return nil
	}
}

func settingValueFromType(settingType SettingType, raw interface{}) (SettingValue, bool) {
	switch settingType {
	case SettingTypeToggle:
		v, ok := raw.(bool)
		return ToggleValue(v), ok
	case SettingTypeSlider:
		v, ok := raw.(float64)
		return SliderValue(v), ok
	case SettingTypeSelect, SettingTypeText:
		v, ok := raw.(string)
		return TextValue(v), ok
	default:
		return SettingValue{}, false
	}
}

// Categories returns the setting categories.
func (p *SettingsPanel) Categories() []SettingCategory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.categories
}
