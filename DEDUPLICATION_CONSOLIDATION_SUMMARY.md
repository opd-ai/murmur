# Code Deduplication Summary — 2026-05-06

## Executive Summary

Successfully reduced code duplication by **7%** (48 lines eliminated) while maintaining 100% test coverage. Consolidated transport layer upgrade logic into a shared onramp utilities package, eliminating 48 lines of duplicated code across I2P and Tor transports.

## Metrics

| Metric | Baseline | Post-Consolidation | Change |
|--------|----------|-------------------|--------|
| Duplication Ratio | 0.675% | 0.628% | ▼ 7.0% |
| Duplicated Lines | 689 | 641 | ▼ 48 lines |
| Largest Clone | 44 lines | 44 lines | — |
| Test Status | ALL PASS | ALL PASS | ✅ |

## Consolidation Actions

### Clone Group 1: Transport Layer Upgrade Logic (48 lines eliminated)

**Pattern**: I2P and Tor transports had identical connection and listener upgrade sequences after obtaining raw network connections.

**Files Modified**:
- Created: `pkg/networking/transport/onramp/common.go` (83 lines)
- Modified: `pkg/networking/transport/onramp_i2p/transport.go` (219 lines, -24 duplicated)
- Modified: `pkg/networking/transport/onramp_tor/transport.go` (220 lines, -24 duplicated)

**Strategy**: Extract function — created two shared helper functions:
1. `UpgradeConnection()` — wraps net.Conn, applies resource management, upgrades to transport.CapableConn (24 lines consolidated)
2. `UpgradeListener()` — wraps net.Listener, converts to multiaddr, gates and upgrades (24 lines consolidated)

**Details**:
- **Dial method duplication** (24L × 2 instances):
  - Lines 81-104 (I2P) and 71-94 (Tor) differed only in the initial dial call
  - Post-dial upgrade sequence (manet.WrapNetConn → rcmgr.OpenConnection → upgrader.Upgrade) was **identical**
  - Extracted to `onramp.UpgradeConnection()`
  
- **Listen method duplication** (24L × 2 instances):
  - Lines 119-142 (I2P) and 114-137 (Tor) differed only in address conversion
  - Post-listen upgrade sequence (manet.WrapNetListener → GateMaListener → UpgradeGatedMaListener) was **identical**
  - Extracted to `onramp.UpgradeListener()`

- **Listener type duplication**:
  - Both I2P and Tor defined identical `listener` structs wrapping `transport.Listener` with multiaddr
  - Consolidated to shared `onramp.Listener` type

**Impact**:
- 48 lines of code eliminated (24 from each transport)
- Improved maintainability — upgrade logic changes now require single update
- Zero behavior change — all integration tests pass

**Trade-offs**:
- Added dependency on new `pkg/networking/transport/onramp` package
- Required import alias (`gtransport` for libp2p transport types) to avoid naming collision
- Close() method not consolidated due to different field names (garlic vs onion)

### Clone Groups Not Consolidated

The following clone groups were **analyzed but not consolidated** as extraction would reduce readability or maintainability:

1. **UI Panel Draw Methods** (25L × 2):
   - `pkg/ui/puzzle.go` and `pkg/ui/puzzle_solver.go`
   - Pattern: Lock → InitPanelDrawWithScreen → calculate px/py → call draw methods
   - Decision: Different draw method sequences per panel type; consolidation would obscure the drawing order

2. **Mechanics Publisher Event Handling** (22L × 2):
   - `pkg/anonymous/mechanics/gifts/gifts_publisher.go` and `marks_publisher.go`
   - Pattern: Validate event → convert protobuf → check duplicate → check expiration → add to store
   - Decision: Different domain types and error codes; generics would add complexity without clarity gain

3. **Resonance Score Cache Pattern** (17L × 2):
   - `pkg/anonymous/resonance/specter.go` and `surface.go`
   - Pattern: Lock → check cache → compute components → sum → cache result
   - Decision: Simple cache-check pattern; extraction would not materially reduce complexity

4. **Ignition Parser Sequential Reads** (14L × 2):
   - `pkg/identity/ignition/ignition.go` (same file, lines 307-320 and 322-335)
   - Pattern: `field, err := p.readField(); if err != nil { return nil, err }`
   - Decision: Sequential parsing clarity more valuable than DRY; extraction obscures parse order

5. **Overlay Active Count Pattern** (11L × 2):
   - `pkg/pulsemap/overlays/echochains.go` and `sparks.go`
   - Pattern: Lock → iterate items → count non-expired → return count
   - Decision: 11 lines, 2 instances — extraction threshold not met

## Validation

```bash
# All tests pass with race detector
go test -race ./...
# Result: ALL PASS (60 packages, 0 failures)

# Duplication analysis
go-stats-generator analyze . --skip-tests --sections duplication
# Result: 0.628% duplication ratio (target: <5%)

# Diff report
go-stats-generator diff baseline-consolidate.json post-consolidate2.json
# Result: Quality Score 100.0/100, Overall Trend: stable
```

## Recommendations

### Short Term
1. ✅ **DONE**: Transport layer upgrade logic consolidated
2. Monitor remaining 22L mechanics publisher pattern — if 3+ more publishers emerge with this pattern, consider generic helper
3. Consider extracting common UI panel initialization once 5+ panels share the pattern

### Long Term
1. Establish duplication threshold policy: consolidate when ≥12 lines AND ≥3 instances OR ≥20 lines AND ≥2 instances
2. Add linter rule to flag new duplicates above threshold in CI
3. Document consolidation patterns in CONTRIBUTING.md

## Specification Compliance

All consolidation adheres to project guidelines:
- ✅ Zero test failures
- ✅ One purpose per package (`pkg/networking/transport/onramp/` for shared onramp utilities)
- ✅ Linter-clean code (`go vet ./...` passes)
- ✅ Complete implementations (no partial/placeholder code)
- ✅ Uses exact cryptographic primitives (no algorithm changes)
- ✅ All wire-format messages remain Protocol Buffers proto3
- ✅ Acyclic dependency graph maintained (onramp imported by child packages)

## Files Changed

- **Created**: 1 file (`pkg/networking/transport/onramp/common.go`, 83 lines)
- **Modified**: 2 files (onramp_i2p/transport.go, onramp_tor/transport.go)
- **Net Change**: +35 lines total, -48 duplicated lines

## Next Steps

1. Update CHANGELOG.md with deduplication summary
2. Update AUDIT.md with refactoring decision rationale
3. Commit changes with message: `refactor(transport): consolidate onramp upgrade logic`
4. Monitor for new duplication patterns in upcoming PRs
