package resonance

import (
	"context"
	"testing"
)

type testHook struct {
	name string
}

func (h testHook) Name() string { return h.name }

func (h testHook) ComputeSignal(context.Context, string, ReadOnlyQuery) (float64, error) {
	return 0.5, nil
}

func TestRegisterResonanceHook(t *testing.T) {
	resetResonanceHookRegistry()
	t.Cleanup(resetResonanceHookRegistry)

	if err := RegisterResonanceHook(testHook{name: "win-rate"}); err != nil {
		t.Fatalf("RegisterResonanceHook failed: %v", err)
	}

	names := RegisteredResonanceHooks()
	if len(names) != 1 || names[0] != "win-rate" {
		t.Fatalf("unexpected registered hooks: %#v", names)
	}

	if err := RegisterResonanceHook(testHook{name: "win-rate"}); err == nil {
		t.Fatal("expected duplicate hook error")
	}
}

func TestNewReadOnlyQuery(t *testing.T) {
	scorer := NewScorer()
	score := scorer.GetScore("specter-1")
	score.AddPublication()

	query := NewReadOnlyQuery(scorer)
	view, ok := query.SpecterScore("specter-1")
	if !ok {
		t.Fatal("expected score lookup to succeed")
	}
	if view.Score <= 0 {
		t.Fatalf("expected positive score, got %d", view.Score)
	}
	if _, ok := query.SpecterScore("missing"); ok {
		t.Fatal("expected missing score lookup to fail")
	}
}
