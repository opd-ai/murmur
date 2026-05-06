// Package overlays - Shared counting and filtering helpers.
package overlays

import "time"

// Expires represents an object with an expiration time.
type Expires interface {
	GetExpiresAt() time.Time
}

// CountNonExpired counts items in a slice that have not expired.
// This consolidates the duplicate pattern found in echochains.go, sparks.go, and other overlays.
func CountNonExpired[T Expires](items []T) int {
	now := time.Now()
	count := 0
	for _, item := range items {
		if now.Before(item.GetExpiresAt()) {
			count++
		}
	}
	return count
}

// CountNonExpiredInMap counts items in a map that have not expired.
func CountNonExpiredInMap[K comparable, T Expires](items map[K]T) int {
	now := time.Now()
	count := 0
	for _, item := range items {
		if now.Before(item.GetExpiresAt()) {
			count++
		}
	}
	return count
}
