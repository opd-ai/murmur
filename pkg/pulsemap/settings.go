// Package pulsemap — settings.go maps SettingsPanel key/value pairs to
// live subsystem state.  This file has no Ebitengine dependency and is
// compiled in both test and non-test builds.

package pulsemap

import "github.com/opd-ai/murmur/pkg/identity/modes"

// parseModeString maps the string values used by SettingsPanel to a modes.Mode.
// The canonical strings are the values shown in the UI:
//
//	"Open", "Hybrid", "Guarded", "Fortress"
//
// Returns (mode, true) on success; (0, false) if the string is unrecognised.
// Per SHADOW_GRADIENT.md, all four modes must be reachable from the settings UI.
func parseModeString(s string) (modes.Mode, bool) {
	switch s {
	case "Open":
		return modes.Open, true
	case "Hybrid":
		return modes.Hybrid, true
	case "Guarded":
		return modes.Guarded, true
	case "Fortress":
		return modes.Fortress, true
	}
	return 0, false
}
