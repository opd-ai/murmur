# Game Module SDK Design

**Status:** Design Complete, Implementation Pending  
**Date:** 2026-05-06  
**Purpose:** Define stable API for third-party game modules per PLAN.md §2.4

---

## Overview

The Game Module SDK provides a sandboxed, stable API for creating MURMUR mini-games (both first-party and third-party). Games built against this SDK cannot access identity, network, or storage directly — only through controlled SDK primitives. This ensures:

1. **Security:** Games cannot leak identity or compromise anonymity
2. **Stability:** SDK API remains backward-compatible across MURMUR versions
3. **Extensibility:** Third parties can build custom games without forking core
4. **Portability:** Games built for MURMUR work in any compatible network

---

## Core API Surface

### Package Structure

```
pkg/game/
├── sdk.go              # Public SDK interface
├── match.go            # Match lifecycle management
├── events.go           # Event broadcasting
├── persistence.go      # State persistence
├── resonance.go        # Resonance reward computation
└── sandbox.go          # Security isolation
```

### Primary Interfaces

#### 1. Game Interface

Every game module implements the `Game` interface:

```go
package game

import (
    "context"
    "time"
)

// Game is the interface that all game modules must implement.
type Game interface {
    // Metadata returns game identification and requirements.
    Metadata() GameMetadata
    
    // CreateMatch initializes a new match with the given configuration.
    CreateMatch(ctx context.Context, config MatchConfig) (Match, error)
    
    // ValidateConfig checks if a match configuration is valid.
    ValidateConfig(config MatchConfig) error
}

// GameMetadata describes a game module.
type GameMetadata struct {
    ID              string        // Unique game identifier (e.g., "cipher-puzzles")
    Name            string        // Human-readable name
    Description     string        // Short description (max 256 bytes)
    MinResonance    int           // Minimum Resonance to initiate
    MinParticipants int           // Minimum participant count
    MaxParticipants int           // Maximum participant count (0 = unlimited)
    AllowedDurations []time.Duration // Valid match durations
    Version         string        // Game module version (semver)
}

// MatchConfig holds configuration for creating a match.
type MatchConfig struct {
    MatchID      [32]byte      // Unique match ID
    InitiatorKey [32]byte      // Specter who initiated the match
    Duration     time.Duration // Match duration
    MaxPlayers   int           // Player cap (0 = unlimited)
    CustomParams map[string]interface{} // Game-specific parameters
}
```

#### 2. Match Interface

A `Match` represents an active game instance:

```go
// Match represents an active game instance.
type Match interface {
    // ID returns the match's unique identifier.
    ID() [32]byte
    
    // Join registers a participant.
    Join(ctx context.Context, participantKey [32]byte) error
    
    // Leave removes a participant (if game allows dropout).
    Leave(ctx context.Context, participantKey [32]byte) error
    
    // HandleEvent processes a game-specific event.
    HandleEvent(ctx context.Context, event Event) error
    
    // State returns current match state.
    State() MatchState
    
    // End finalizes the match and computes rewards.
    End(ctx context.Context, outcome Outcome) error
}

// MatchState represents the current state of a match.
type MatchState struct {
    MatchID       [32]byte
    Status        MatchStatus
    Participants  [][32]byte // Participant Specter keys
    CreatedAt     time.Time
    ExpiresAt     time.Time
    CustomState   interface{} // Game-specific state
}

// MatchStatus indicates match lifecycle stage.
type MatchStatus uint8

const (
    MatchPending   MatchStatus = iota // Waiting for participants
    MatchActive                       // In progress
    MatchCompleted                    // Ended with outcome
    MatchExpired                      // Timed out
    MatchCancelled                    // Cancelled by initiator
)
```

#### 3. Event Interface

Events are game-specific actions broadcast as Waves:

```go
// Event represents a game-specific action.
type Event struct {
    MatchID      [32]byte           // Match this event belongs to
    EventType    string             // Game-defined event type
    Payload      []byte             // Serialized event data
    ActorKey     [32]byte           // Participant who triggered event
    Timestamp    time.Time
}

// EventPublisher handles event broadcasting.
type EventPublisher interface {
    // Publish broadcasts an event to the network.
    Publish(ctx context.Context, event Event) error
    
    // Subscribe registers a handler for events of a specific type.
    Subscribe(matchID [32]byte, eventType string, handler EventHandler) error
}

// EventHandler processes received events.
type EventHandler func(ctx context.Context, event Event) error
```

#### 4. Persistence Interface

Games persist state to Bbolt via the SDK:

```go
// StateStore handles match state persistence.
type StateStore interface {
    // Save persists match state.
    Save(ctx context.Context, matchID [32]byte, state interface{}) error
    
    // Load retrieves match state.
    Load(ctx context.Context, matchID [32]byte, state interface{}) error
    
    // Delete removes match state (after expiration).
    Delete(ctx context.Context, matchID [32]byte) error
}
```

#### 5. Resonance Interface

Games compute rewards via SDK formulas:

```go
// ResonanceRewarder computes and applies Resonance bonuses.
type ResonanceRewarder interface {
    // AwardBonus applies a decaying Resonance bonus to participants.
    AwardBonus(ctx context.Context, rewards []Reward) error
}

// Reward specifies a Resonance bonus for a participant.
type Reward struct {
    SpecterKey  [32]byte      // Recipient's Specter key
    BonusAmount float64       // Bonus value (pre-decay)
    DecayDays   int           // Days until bonus decays to zero
    Source      string        // Reward source (e.g., "cipher-puzzle-win")
}
```

---

## Sandboxing Model

### What Games Can Access

✅ **Allowed:**
- Create and manage matches via `Match` interface
- Broadcast events via `EventPublisher`
- Persist state via `StateStore`
- Compute Resonance rewards via `ResonanceRewarder`
- Access match-scoped participant keys (Specter public keys of participants only)

❌ **Forbidden:**
- Direct access to identity keypairs (Ed25519, Curve25519)
- Direct network I/O (libp2p, GossipSub)
- Direct Bbolt database access
- Direct file system access
- Access to non-participant Specter data

### Enforcement

Sandboxing is enforced at the API boundary:

1. **Interface contracts:** Games receive only SDK interfaces, not concrete types with privileged methods
2. **Context timeouts:** All SDK calls require `context.Context` with enforced timeouts
3. **Rate limiting:** SDK tracks per-game event publication rates and rejects excessive calls
4. **Quota enforcement:** Games have storage quotas per match (default 10 MiB)

---

## Reference Implementation: Cipher Puzzles

The existing `pkg/anonymous/mechanics/puzzles/` package will be refactored to implement the SDK:

### Before (Current)

```go
// Direct Bbolt access, manual Wave construction
type PuzzleManager struct {
    db        *bbolt.DB
    publisher *mechanics.Publisher
    // ...
}
```

### After (SDK-based)

```go
// SDK-only access
type CipherPuzzleGame struct {
    sdk *game.SDK
}

func (g *CipherPuzzleGame) Metadata() game.GameMetadata {
    return game.GameMetadata{
        ID:              "cipher-puzzles",
        Name:            "Cipher Puzzles",
        Description:     "Cryptographic challenges for Specters",
        MinResonance:    50,
        MinParticipants: 1,
        MaxParticipants: 0, // Unlimited
        AllowedDurations: []time.Duration{15 * time.Minute, 30 * time.Minute, 60 * time.Minute},
        Version:         "1.0.0",
    }
}

func (g *CipherPuzzleGame) CreateMatch(ctx context.Context, config game.MatchConfig) (game.Match, error) {
    // Validate initiator Resonance via SDK
    if !g.sdk.CheckResonance(config.InitiatorKey, 50) {
        return nil, game.ErrInsufficientResonance
    }
    
    // Create match state
    puzzle := &CipherPuzzleMatch{
        id:       config.MatchID,
        sdk:      g.sdk,
        duration: config.Duration,
        // ... puzzle-specific fields
    }
    
    return puzzle, nil
}
```

---

## Stability Guarantees

The SDK follows semantic versioning:

- **STABLE (v1.x):** `Game`, `Match`, `Event`, `StateStore`, `ResonanceRewarder` interfaces
- **EXPERIMENTAL (v0.x):** Advanced features (custom Wave types, UI overlays) — may change
- **PRIVATE:** Internal SDK implementation — no compatibility promise

### Breaking Changes

Breaking changes to STABLE interfaces require:
1. 6-month deprecation notice in release notes
2. Parallel support for old and new interfaces during transition
3. Migration tooling for existing games

---

## Example: Custom Game Module

```go
package mytriviagame

import (
    "context"
    "github.com/opd-ai/murmur/pkg/game"
)

type TriviaGame struct {
    sdk *game.SDK
}

func (g *TriviaGame) Metadata() game.GameMetadata {
    return game.GameMetadata{
        ID:              "trivia-showdown",
        Name:            "Trivia Showdown",
        Description:     "Anonymous trivia competition",
        MinResonance:    25,
        MinParticipants: 2,
        MaxParticipants: 10,
        AllowedDurations: []time.Duration{10 * time.Minute},
        Version:         "1.0.0",
    }
}

func (g *TriviaGame) CreateMatch(ctx context.Context, config game.MatchConfig) (game.Match, error) {
    // Validate custom params
    category, ok := config.CustomParams["category"].(string)
    if !ok {
        return nil, game.ErrInvalidConfig
    }
    
    // Create match
    match := &TriviaMatch{
        sdk:      g.sdk,
        matchID:  config.MatchID,
        category: category,
        questions: g.loadQuestions(category),
    }
    
    return match, nil
}

// TriviaMatch implements game.Match
type TriviaMatch struct {
    sdk       *game.SDK
    matchID   [32]byte
    category  string
    questions []Question
    scores    map[[32]byte]int // Participant scores
}

func (m *TriviaMatch) HandleEvent(ctx context.Context, event game.Event) error {
    switch event.EventType {
    case "answer_submit":
        return m.handleAnswer(ctx, event)
    case "question_request":
        return m.sendNextQuestion(ctx, event.ActorKey)
    default:
        return game.ErrUnknownEventType
    }
}

func (m *TriviaMatch) End(ctx context.Context, outcome game.Outcome) error {
    // Compute rewards: top 3 participants
    rankedParticipants := m.rankByScore()
    rewards := []game.Reward{
        {SpecterKey: rankedParticipants[0], BonusAmount: 5.0, DecayDays: 14, Source: "trivia-1st"},
        {SpecterKey: rankedParticipants[1], BonusAmount: 3.0, DecayDays: 14, Source: "trivia-2nd"},
        {SpecterKey: rankedParticipants[2], BonusAmount: 2.0, DecayDays: 14, Source: "trivia-3rd"},
    }
    
    return m.sdk.ResonanceRewarder().AwardBonus(ctx, rewards)
}
```

---

## Testing Strategy

### Unit Tests

- Mock SDK interfaces for game logic testing
- No libp2p, no Bbolt, no Ebitengine dependencies in game tests

### Integration Tests

- In-memory SDK with fake event bus and storage
- Multi-participant match scenarios
- Dropout tolerance verification

### Example Test

```go
func TestCipherPuzzleSDK(t *testing.T) {
    sdk := game.NewMockSDK()
    game := &CipherPuzzleGame{sdk: sdk}
    
    config := game.MatchConfig{
        MatchID:      randomID(),
        InitiatorKey: testSpecterKey,
        Duration:     15 * time.Minute,
    }
    
    match, err := game.CreateMatch(context.Background(), config)
    require.NoError(t, err)
    
    // Join participants
    err = match.Join(context.Background(), participant1Key)
    require.NoError(t, err)
    
    // Submit solution
    event := game.Event{
        MatchID:   match.ID(),
        EventType: "solution_submit",
        Payload:   encodeSolution(solution),
        ActorKey:  participant1Key,
    }
    err = match.HandleEvent(context.Background(), event)
    require.NoError(t, err)
    
    // Verify reward computation
    rewards := sdk.GetAwardedRewards()
    require.Len(t, rewards, 1)
    require.Equal(t, participant1Key, rewards[0].SpecterKey)
}
```

---

## Migration Path

### Phase 1: Extract SDK from Cipher Puzzles (2 weeks)

1. Create `pkg/game/` with SDK interfaces
2. Implement SDK backed by existing infrastructure
3. Refactor Cipher Puzzles to use SDK
4. Verify zero behavioral change via integration tests

### Phase 2: Migrate Remaining Games (4 weeks)

5. Migrate Specter Hunts, Oracle Pools, Sigil Forge (1 week each)
6. Migrate Shadow Play (requires state machine refactor, 1 week)
7. Update tests to use mock SDK

### Phase 3: Documentation & Examples (1 week)

8. Write `docs/GAME_SDK.md` tutorial
9. Publish reference Cipher Puzzles implementation
10. Create example custom game (Trivia Showdown or similar)

---

## Success Criteria

✅ Cipher Puzzles refactored to SDK with zero test regressions  
✅ At least one non-core game (Trivia, Pictionary, etc.) implemented by third party  
✅ SDK API documented and published  
✅ All game state isolated from core identity/network/storage  

---

## Future Extensions (Post-v1.0)

- **Custom Wave Types:** Games can define Wave schema extensions
- **UI Overlays:** Games can register custom Pulse Map rendering layers
- **Adaptive Difficulty:** SDK provides player skill estimation for dynamic balancing
- **Cross-Game Tournaments:** Meta-games that orchestrate multiple game types
- **Game Discovery:** In-app marketplace for third-party games

These extensions will be added to SDK in EXPERIMENTAL status (v0.x) and promoted to STABLE after validation.
