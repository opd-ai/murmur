// Package resonance - Generic scorer implementation.
// Consolidates duplicate GetScore patterns from score.go, specter.go, surface.go.
package resonance

import "sync"

// clampFraction clamps a value to the range [0, 1].
func clampFraction(fraction float64) float64 {
	if fraction < 0 {
		return 0
	}
	if fraction > 1 {
		return 1
	}
	return fraction
}

// GenericScorer provides thread-safe score retrieval and creation.
// T is the score type (e.g., *Score, *SpecterScore, *SurfaceScore).
type GenericScorer[T any] struct {
	mu      sync.RWMutex
	scores  map[string]T
	factory func() T
}

// NewGenericScorer creates a new generic scorer with the given factory function.
func NewGenericScorer[T any](factory func() T) *GenericScorer[T] {
	return &GenericScorer[T]{
		scores:  make(map[string]T),
		factory: factory,
	}
}

// GetScore retrieves or creates a score for the given ID.
func (sc *GenericScorer[T]) GetScore(id string) T {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if score, ok := sc.scores[id]; ok {
		return score
	}

	score := sc.factory()
	sc.scores[id] = score
	return score
}

// LookupScore retrieves a score without creating one.
func (sc *GenericScorer[T]) LookupScore(id string) (T, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	score, ok := sc.scores[id]
	return score, ok
}

// SetScore sets a score for the given ID.
func (sc *GenericScorer[T]) SetScore(id string, score T) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.scores[id] = score
}

// AllScores returns a copy of all scores.
func (sc *GenericScorer[T]) AllScores() map[string]T {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make(map[string]T, len(sc.scores))
	for k, v := range sc.scores {
		result[k] = v
	}
	return result
}
