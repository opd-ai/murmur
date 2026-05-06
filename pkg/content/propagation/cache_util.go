// Package propagation - shared cache utilities.
package propagation

import "time"

// findOldestEntry returns the key with the oldest timestamp in a map.
// Returns empty string if the map is empty.
func findOldestEntry(cache map[string]time.Time) string {
	if len(cache) == 0 {
		return ""
	}

	var oldestID string
	var oldestTime time.Time
	first := true

	for id, t := range cache {
		if first || t.Before(oldestTime) {
			oldestID = id
			oldestTime = t
			first = false
		}
	}

	return oldestID
}
