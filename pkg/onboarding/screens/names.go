// Package screens provides onboarding screen implementations.
// This file contains name generation utilities shared across build targets.

package screens

// specterAdjectives is the curated list of adjectives for Specter names.
var specterAdjectives = []string{
	"Silent", "Hollow", "Spectral", "Veiled", "Shadow",
	"Whispered", "Hidden", "Faded", "Drifting", "Phantom",
	"Ethereal", "Misted", "Shrouded", "Obscured", "Wandering",
	"Fleeting", "Echoing", "Elusive", "Distant", "Cloaked",
}

// specterNouns is the curated list of nouns for Specter names.
var specterNouns = []string{
	"Beacon", "Cipher", "Shade", "Wraith", "Echo",
	"Drift", "Veil", "Murmur", "Whisper", "Fog",
	"Mist", "Ghost", "Specter", "Phantom", "Shadow",
	"Presence", "Trace", "Remnant", "Glimmer", "Void",
}

// GenerateSpecterName creates a two-word pseudonym from public key bytes.
// The name is deterministic based on the first two bytes of the key.
// Per ANONYMOUS_GAME_MECHANICS.md, Specter names use adjective + noun format.
func GenerateSpecterName(pubKey []byte) string {
	if len(pubKey) < 2 {
		return "Unknown Specter"
	}

	adjIdx := int(pubKey[0]) % len(specterAdjectives)
	nounIdx := int(pubKey[1]) % len(specterNouns)

	return specterAdjectives[adjIdx] + " " + specterNouns[nounIdx]
}
