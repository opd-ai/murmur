package mechanics

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// GameMetadata describes a registered game module.
type GameMetadata struct {
	ID               string
	Name             string
	Description      string
	MinResonance     int
	MinParticipants  int
	MaxParticipants  int
	AllowedDurations []time.Duration
	Version          string
}

// MatchConfig captures the inputs for creating a match.
type MatchConfig struct {
	MatchID      [32]byte
	InitiatorKey [32]byte
	Duration     time.Duration
	MaxPlayers   int
	CustomParams map[string]any
}

// MatchStatus indicates the lifecycle stage for a game match.
type MatchStatus uint8

const (
	MatchPending MatchStatus = iota
	MatchActive
	MatchCompleted
	MatchExpired
	MatchCancelled
)

// MatchState is the read-only match snapshot exposed through the SDK.
type MatchState struct {
	MatchID      [32]byte
	Status       MatchStatus
	Participants [][32]byte
	CreatedAt    time.Time
	ExpiresAt    time.Time
	CustomState  any
}

// Event is a game-specific action emitted during a match.
type Event struct {
	MatchID   [32]byte
	EventType string
	Payload   []byte
	ActorKey  [32]byte
	Timestamp time.Time
}

// Outcome captures the final result of a match.
type Outcome struct {
	WinnerKeys [][32]byte
	Summary    []byte
}

// Match represents an active game instance.
type Match interface {
	ID() [32]byte
	Join(ctx context.Context, participantKey [32]byte) error
	Leave(ctx context.Context, participantKey [32]byte) error
	HandleEvent(ctx context.Context, event Event) error
	State() MatchState
	End(ctx context.Context, outcome Outcome) error
}

// GameModule is the stable extension point for third-party mechanics.
type GameModule interface {
	Metadata() GameMetadata
	CreateMatch(ctx context.Context, config MatchConfig) (Match, error)
	ValidateConfig(config MatchConfig) error
}

var (
	gameModuleMu       sync.RWMutex
	gameModuleRegistry = make(map[string]GameModule)
)

// RegisterGameModule publishes a game module under its stable metadata ID.
func RegisterGameModule(module GameModule) error {
	if module == nil {
		return fmt.Errorf("game module is nil")
	}

	meta := module.Metadata()
	if meta.ID == "" {
		return fmt.Errorf("game module metadata ID is required")
	}

	gameModuleMu.Lock()
	defer gameModuleMu.Unlock()

	if _, exists := gameModuleRegistry[meta.ID]; exists {
		return fmt.Errorf("game module %q already registered", meta.ID)
	}
	gameModuleRegistry[meta.ID] = module
	return nil
}

// GameModuleByID returns a registered game module.
func GameModuleByID(id string) (GameModule, bool) {
	gameModuleMu.RLock()
	defer gameModuleMu.RUnlock()
	module, ok := gameModuleRegistry[id]
	return module, ok
}

// RegisteredGameModules returns sorted stable game module IDs.
func RegisteredGameModules() []string {
	gameModuleMu.RLock()
	defer gameModuleMu.RUnlock()

	ids := make([]string, 0, len(gameModuleRegistry))
	for id := range gameModuleRegistry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func resetGameModuleRegistry() {
	gameModuleMu.Lock()
	defer gameModuleMu.Unlock()
	gameModuleRegistry = make(map[string]GameModule)
}
