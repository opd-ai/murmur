package mechanics

import (
	"context"
	"testing"
)

type testMatch struct{}

func (testMatch) ID() [32]byte                             { return [32]byte{1} }
func (testMatch) Join(context.Context, [32]byte) error     { return nil }
func (testMatch) Leave(context.Context, [32]byte) error    { return nil }
func (testMatch) HandleEvent(context.Context, Event) error { return nil }
func (testMatch) State() MatchState                        { return MatchState{} }
func (testMatch) End(context.Context, Outcome) error       { return nil }

type testGameModule struct {
	id string
}

func (m testGameModule) Metadata() GameMetadata {
	return GameMetadata{ID: m.id, Name: "Test", MinParticipants: 1, Version: "1.0.0"}
}

func (m testGameModule) CreateMatch(context.Context, MatchConfig) (Match, error) {
	return testMatch{}, nil
}

func (m testGameModule) ValidateConfig(MatchConfig) error { return nil }

func TestRegisterGameModule(t *testing.T) {
	resetGameModuleRegistry()
	t.Cleanup(resetGameModuleRegistry)

	if err := RegisterGameModule(testGameModule{id: "cipher-puzzles"}); err != nil {
		t.Fatalf("RegisterGameModule failed: %v", err)
	}

	ids := RegisteredGameModules()
	if len(ids) != 1 || ids[0] != "cipher-puzzles" {
		t.Fatalf("unexpected registered game modules: %#v", ids)
	}

	if _, ok := GameModuleByID("cipher-puzzles"); !ok {
		t.Fatal("expected registered module lookup to succeed")
	}

	if err := RegisterGameModule(testGameModule{id: "cipher-puzzles"}); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestRegisterGameModuleRejectsInvalidInputs(t *testing.T) {
	resetGameModuleRegistry()
	t.Cleanup(resetGameModuleRegistry)

	if err := RegisterGameModule(nil); err == nil {
		t.Fatal("expected nil module error")
	}
	if err := RegisterGameModule(testGameModule{}); err == nil {
		t.Fatal("expected empty ID error")
	}
}
