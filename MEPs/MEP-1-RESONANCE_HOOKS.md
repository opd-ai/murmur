# MEP-1: Custom Resonance Hooks for Third-Party Games

## Title

Allow game modules to contribute to Resonance scoring via public hooks

## Status

Proposed

## Motivation

Currently, Resonance milestones are hard-coded in the core (Shade@25, Wraith@50, etc.), and game-specific achievements don't contribute to Resonance. Third-party game developers want to reward players with Resonance points for in-game achievements (e.g., solving puzzles correctly, winning territory, etc.) without modifying the core scorer.

This MEP proposes a public hook interface that lets registered game modules submit Resonance contributions that are aggregated into the player's overall score.

## Proposed Interface

```go
// ResonanceHook is a game-provided callback for Resonance contributions.
type ResonanceHook interface {
    // GameID returns the game module ID (e.g., "cipher-puzzles").
    GameID() string
    
    // ComputeContribution analyzes a completed match and returns
    // Resonance point contributions for each participant.
    // Returns map[participantKey] -> points (may be negative for penalties).
    ComputeContribution(ctx context.Context, outcome Outcome) (map[[32]byte]int, error)
}

// RegisterResonanceHook publishes a Resonance hook.
func RegisterResonanceHook(hook ResonanceHook) error

// ResonanceContributions aggregates hook contributions for a key.
func ResonanceContributions(ctx context.Context, participantKey [32]byte) (int, error)
```

## Stability

Experimental (v0.1 ship with feature gating; refine interface based on feedback)

## Backward Compatibility

- Old game modules do not register hooks; they contribute 0 Resonance
- Old clients that don't understand Resonance hooks ignore them silently
- Resonance computation remains deterministic per participant: sum of all hook outputs plus base score

## Security Considerations

- **DoS**: A malicious hook could submit unlimited points. **Mitigation**: Core sets per-game caps on contribution (+/- 50 points per game per period).
- **Information leak**: A hook could observe all participants. **Mitigation**: Hooks only see the outcome (winners, summary); they must not log or transmit participant data beyond contributing scores.
- **Manipulation**: A hook favors certain players. **Mitigation**: Hooks are code-reviewed; obvious bias is detected during game approval.

## Design & Rationale

**Why hooks**: Games are sandboxed and have limited data access. Hooks let games contribute to global metrics without exposing internal state.

**Why read-only**: Games submit contributions; the core scorer decides how to weight them. Games don't modify other players' scores or read anyone else's scores.

**Why async**: Contributions are computed periodically (e.g., after each match ends), not in real-time. Avoids latency requirements.

## Implementation Notes

- Modify `pkg/anonymous/resonance/hooks.go` to add hook registry
- Update `pkg/anonymous/resonance/scorer.go` to query hooks during score aggregation
- Add per-game contribution caps in configuration
- Test with 5+ concurrent hooks, verify cap enforcement

## Example Usage

```go
// In a game module
type MyGameHook struct{}

func (h *MyGameHook) GameID() string {
    return "cipher-puzzles"
}

func (h *MyGameHook) ComputeContribution(ctx context.Context, outcome Outcome) (map[[32]byte]int, error) {
    contributions := make(map[[32]byte]int)
    
    // Award 5 points per winner
    for _, winner := range outcome.WinnerKeys {
        contributions[winner] += 5
    }
    
    return contributions, nil
}

// At game startup
hook := &MyGameHook{}
resonance.RegisterResonanceHook(hook)
```

## Testing Strategy

- Unit tests: Mock outcomes, verify hook computation
- Integration tests: 5 concurrent games submitting hooks, verify cap enforcement
- Property tests: Resonance is monotonic within caps; sum is deterministic

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Malicious game contributes huge scores | Per-game caps; core validates all inputs |
| Hook panics during computation | Recover and log; contribution = 0 |
| Hook is slow (blocks scorer) | Run in dedicated goroutine; timeout at 5s |
| Hook leaks participant data | Code review; no storage/network access in hook |

## Related Work

- EXTENSION_CONTRACT.md §Custom Resonance Hooks (EXPERIMENTAL)
- [Resonance System docs](../RESONANCE_SYSTEM.md)
- Issue #123 (hypothetical): "Games want to unlock features via Resonance"

## Questions for Reviewers

1. Should contributions be signed by the game module (proof that the game computed them)?
2. Should there be a dispute mechanism for invalid contributions (e.g., cheating detected)?
3. Should old Resonance scores be recalculated retroactively when new hooks are registered?

---

**Author**: MURMUR Core Team  
**Created**: 2026-05-07  
**Last Updated**: 2026-05-07
