package resonance

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ReadOnlyScore exposes a hook-safe snapshot of Resonance state.
type ReadOnlyScore struct {
	Score     int
	Rank      Rank
	EchoIndex float64
}

// ReadOnlyQuery exposes read-only score lookups for extension hooks.
type ReadOnlyQuery interface {
	SpecterScore(specterID string) (ReadOnlyScore, bool)
}

// ResonanceHook computes an additional normalized signal without mutating state.
type ResonanceHook interface {
	Name() string
	ComputeSignal(ctx context.Context, specterID string, query ReadOnlyQuery) (float64, error)
}

type scorerReadOnlyQuery struct {
	scorer interface {
		LookupScore(specterID string) (*Score, bool)
	}
}

var (
	resonanceHookMu       sync.RWMutex
	resonanceHookRegistry = make(map[string]ResonanceHook)
)

// NewReadOnlyQuery adapts a scorer to the extension read-only query surface.
func NewReadOnlyQuery(scorer interface {
	LookupScore(specterID string) (*Score, bool)
}) ReadOnlyQuery {
	if scorer == nil {
		return nil
	}
	return scorerReadOnlyQuery{scorer: scorer}
}

// RegisterResonanceHook adds a hook to the extension registry.
func RegisterResonanceHook(hook ResonanceHook) error {
	if hook == nil {
		return fmt.Errorf("resonance hook is nil")
	}
	name := hook.Name()
	if name == "" {
		return fmt.Errorf("resonance hook name is required")
	}

	resonanceHookMu.Lock()
	defer resonanceHookMu.Unlock()

	if _, exists := resonanceHookRegistry[name]; exists {
		return fmt.Errorf("resonance hook %q already registered", name)
	}
	resonanceHookRegistry[name] = hook
	return nil
}

// RegisteredResonanceHooks returns the registered hook names in ascending order.
func RegisteredResonanceHooks() []string {
	resonanceHookMu.RLock()
	defer resonanceHookMu.RUnlock()

	names := make([]string, 0, len(resonanceHookRegistry))
	for name := range resonanceHookRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (q scorerReadOnlyQuery) SpecterScore(specterID string) (ReadOnlyScore, bool) {
	score, ok := q.scorer.LookupScore(specterID)
	if !ok || score == nil {
		return ReadOnlyScore{}, false
	}
	return ReadOnlyScore{
		Score:     score.Compute(),
		Rank:      score.Rank(),
		EchoIndex: score.EchoIndex(),
	}, true
}

func resetResonanceHookRegistry() {
	resonanceHookMu.Lock()
	defer resonanceHookMu.Unlock()
	resonanceHookRegistry = make(map[string]ResonanceHook)
}
