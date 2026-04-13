// Package mechanics provides shared helpers for anonymous game mechanics stores.
package mechanics

// GarbageCollectHistory removes old entries from a history slice and lookup map.
// It returns the number of entries removed and the new history slice.
// The caller must hold the write lock and update their fields with the results.
// The getID function extracts the ID from each item for map deletion.
func GarbageCollectHistory[T any](history []T, lookup map[[32]byte]T, maxHistory int, getID func(T) [32]byte) ([]T, int) {
	if len(history) <= maxHistory {
		return history, 0
	}

	removed := len(history) - maxHistory

	for i := 0; i < removed; i++ {
		delete(lookup, getID(history[i]))
	}

	return history[removed:], removed
}
