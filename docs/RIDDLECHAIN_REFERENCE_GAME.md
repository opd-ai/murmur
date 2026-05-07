# RiddleChain: Reference Game Extension for MURMUR

## Overview

RiddleChain is a reference implementation of the MURMUR `GameModule` interface. It demonstrates how third-party developers can build new games using the stable extension surface without forking the core codebase.

**Location**: `pkg/anonymous/mechanics/games/riddlechain/`

**Status**: Reference implementation (stable API)

## Mechanics

**Game Type**: Cooperative puzzle game  
**Players**: 2-8  
**Duration**: 15-30 minutes  
**Minimum Resonance**: 25 (Shade milestone)  

Players cooperatively solve 5 riddles. For each riddle:
1. The game presents a word riddle with optional hints
2. Players submit guesses
3. 3+ out of N players must agree on the same answer
4. The group advances to the next riddle when consensus is reached

The game ends early if all riddles are solved, or on timer if not.

## Why RiddleChain?

RiddleChain demonstrates key aspects of the GameModule SDK:

1. **Sandboxing**: The game only accesses the SDK interfaces (`Match`, `Event`, `Outcome`). It has no direct access to identity, storage, or network layers.

2. **Lifecycle Management**: Implements full match lifecycle (create, join, leave, events, end) with proper state transitions.

3. **Async-Safe**: All interaction is async via event handling — no real-time latency that would leak metadata.

4. **Resonance Integration**: Declares minimum Resonance requirements (25), allowing the core to gate access dynamically.

5. **Extensibility**: Uses `CustomState` and `CustomParams` for game-specific data without modifying the core SDK.

## Extension Surface Used

- **`GameModule` interface**: Full implementation with `Metadata()`, `CreateMatch()`, `ValidateConfig()`
- **`Match` interface**: All methods (ID, Join, Leave, HandleEvent, State, End)
- **`Event` type**: Custom event payload for answer submissions
- **`Outcome` type**: Winner determination and match summary

## Testing

RiddleChain includes comprehensive tests:

- `TestModuleMetadata`: Validates metadata structure
- `TestCreateMatch`: Verifies match instantiation
- `TestValidateConfig`: Tests configuration validation
- `TestMatchJoinLeave`: Validates player lifecycle
- `TestHandleEvent`: Simulates answer submission and consensus
- `TestMatchEnd`: Confirms match completion transitions

Run tests:
```bash
go test ./pkg/anonymous/mechanics/games/riddlechain/
```

## Build and Registration

To use RiddleChain in a MURMUR client:

```go
import "github.com/opd-ai/murmur/pkg/anonymous/mechanics/games/riddlechain"

// At startup
riddleModule := riddlechain.NewModule()
if err := mechanics.RegisterGameModule(riddleModule); err != nil {
    // Handle error
}
```

The game will automatically appear in the available games list once registered.

## Future Extensions

This reference could be extended to:

- **Difficulty levels**: Dynamic riddle selection based on player count/skill
- **Scoring**: Individual player scores with leaderboards (Resonance-gated)
- **Variations**: Multiple riddle packs (fairy tales, mythology, pop culture)
- **Hints system**: Players can spend points for hints
- **Ranked mode**: Faster-paced variant for competitive play

## Extension Contract Compliance

RiddleChain fully complies with the EXTENSION_CONTRACT.md:

- ✅ Implements the stable `GameModule` interface
- ✅ No direct access to identity, network, or storage
- ✅ Uses only the public SDK primitives
- ✅ Operates within configuration constraints
- ✅ Fully testable in isolation
- ✅ Backward compatible with SDK updates

---

**Reference Implementation**: RiddleChain v1.0.0  
**Created**: 2026-05-07  
**Status**: Stable (demonstrates extension surface is real, not theoretical)
