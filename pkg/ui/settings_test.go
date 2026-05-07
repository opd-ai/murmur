package ui

import "testing"

func TestSettingsPanel_ActivateToggleControl(t *testing.T) {
	var changedKey, changedValue string
	panel := NewSettingsPanel(DefaultTheme(), func(key, value string) {
		changedKey = key
		changedValue = value
	})
	panel.selected = 0 // Network

	if ok := panel.activateSettingControl(0, 0, 0, 0, 0, 0); !ok {
		t.Fatal("expected toggle activation to return true")
	}
	if got := panel.categories[0].Settings[0].Value.(bool); got != false {
		t.Fatalf("expected toggle to flip to false, got %v", got)
	}
	if changedKey != "dht_enabled" || changedValue != "false" {
		t.Fatalf("expected onChange dht_enabled=false, got %s=%s", changedKey, changedValue)
	}
}

func TestSettingsPanel_ActivateSelectControl(t *testing.T) {
	var changedKey, changedValue string
	panel := NewSettingsPanel(DefaultTheme(), func(key, value string) {
		changedKey = key
		changedValue = value
	})
	panel.selected = 1 // Privacy

	if ok := panel.activateSettingControl(0, 0, 0, 0, 0, 0); !ok {
		t.Fatal("expected select activation to return true")
	}
	if got := panel.categories[1].Settings[0].Value.(string); got != "Guarded" {
		t.Fatalf("expected select to cycle Hybrid->Guarded, got %q", got)
	}
	if changedKey != "privacy_mode" || changedValue != "Guarded" {
		t.Fatalf("expected onChange privacy_mode=Guarded, got %s=%s", changedKey, changedValue)
	}
}

func TestSettingsPanel_SliderUpdateFromPointer(t *testing.T) {
	var changedKey, changedValue string
	panel := NewSettingsPanel(DefaultTheme(), func(key, value string) {
		changedKey = key
		changedValue = value
	})
	panel.selected = 0 // Network

	// Simulate pointer at far-right of slider track to reach max.
	panel.updateSliderValueAt(2, 10_000, 0)
	v := panel.categories[0].Settings[2].Value.(float64)
	if v < 99.0 {
		t.Fatalf("expected slider value near max, got %f", v)
	}
	if changedKey != "max_peers" || changedValue == "" {
		t.Fatalf("expected non-empty numeric callback for max_peers, got %s=%q", changedKey, changedValue)
	}
}

func TestSettingsPanel_TextControlFocus(t *testing.T) {
	panel := NewSettingsPanel(DefaultTheme(), nil)
	panel.selected = 2 // Devices

	if ok := panel.activateSettingControl(0, 0, 0, 0, 0, 0); !ok {
		t.Fatal("expected text control activation to return true")
	}
	if panel.textFocusIndex != 0 {
		t.Fatalf("expected text focus index 0, got %d", panel.textFocusIndex)
	}
}

func TestSettingsPanel_ConvertValueToStringFloat(t *testing.T) {
	panel := NewSettingsPanel(DefaultTheme(), nil)
	got := panel.convertValueToString(3.5)
	if got == "" {
		t.Fatal("expected non-empty float serialization")
	}
	if got != "3.5" {
		t.Fatalf("expected deterministic float serialization 3.5, got %q", got)
	}
}
