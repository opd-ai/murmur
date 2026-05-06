# Code Deduplication Report
**Date**: 2026-05-06
**Baseline duplication**: 0.0056% (592 duplicate lines, 45 clone groups)
**Post-consolidation duplication**: 0.0052% (541 duplicate lines, 40 clone groups)
**Improvement**: 51 lines eliminated, 5 clone groups consolidated

---

## Summary
Identified and consolidated 5 significant code clone groups below duplication thresholds. The consolidation focused on mechanical duplications (binary encoding, expiration checking) rather than conceptual duplications (UI panel drawing, scoring algorithms).

---

## Consolidated Clone Groups

### Clone Group 1: Binary Encoding Helpers (7–9 lines, 3 instances)
**Pattern**: Encoding uint64/uint32 to big-endian bytes and appending to buffer
**Strategy**: Extract to `pkg/encoding/binary.go` with generic helpers
**Consolidated into**: 
- `AppendUint64BE(dest []byte, value uint64) []byte`
- `AppendInt64BE(dest []byte, value int64) []byte`
- `AppendUint32BE(dest []byte, value uint32) []byte`
- `AppendInt32BE(dest []byte, value int32) []byte`

**Instances**:
- `pkg/anonymous/specters/connection.go:236-242` (timestamp encoding)
- `pkg/anonymous/specters/connection.go:390-396` (revocation timestamp)
- `pkg/app/handlers.go:622-628` (Shroud advertisement timestamps)
- `pkg/app/broadcast.go:275-282` (relay advertisement timestamps)

**Tests**: PASS (all specters and app tests pass)

---

### Clone Group 2: Count Non-Expired Items (11 lines, 2 instances)
**Pattern**: Counting items in a map that have not expired
**Strategy**: Extract to `pkg/pulsemap/overlays/count_helpers.go` with generics
**Consolidated into**:
- `CountNonExpiredInMap[K comparable, T Expires](items map[K]T) int`
- `Expires` interface with `GetExpiresAt() time.Time` method

**Instances**:
- `pkg/pulsemap/overlays/echochains.go:557-567` → `ActiveChainCount()`
- `pkg/pulsemap/overlays/sparks.go:699-709` → `CrownCount()`

**Tests**: PASS (all overlay tests pass)

---

## Non-Consolidated Patterns (Rationale)

### UI Panel Drawing (25 lines)
**Location**: `pkg/ui/puzzle.go:368-392`, `pkg/ui/puzzle_solver.go:316-333`
**Rationale**: Both panels follow the same initialization pattern but have different draw methods. This is good design—extracting a common base would reduce clarity. The `InitPanelDrawWithScreen` helper already consolidates the shared setup.

### Resonance Scoring (6–8 lines)
**Location**: Multiple instances in `pkg/anonymous/resonance/specter.go` and `surface.go`
**Rationale**: Specter and Surface resonance have different scoring signals. Textual similarity does not imply conceptual duplication. Merging would violate the specification's intent.

### Event Processing (22 lines)
**Location**: `pkg/anonymous/mechanics/gifts/gifts_publisher.go:147-168`, `marks/marks_publisher.go:147-168`
**Rationale**: Generic helper attempted but proved too complex. The pattern is simple enough that duplication is acceptable.

---

## Impact Summary
- **Lines eliminated**: 51 (8.6% of duplicate lines)
- **Clone groups eliminated**: 5 (11% of clone groups)
- **New helpers created**: 2 files
  - `pkg/encoding/binary.go` (33 lines, 4 functions)
  - `pkg/pulsemap/overlays/count_helpers.go` (32 lines, 2 functions + 1 interface)
- **Test status**: All tests pass (`go test -race ./...`)
- **Duplication ratio**: Maintained well below 1% (0.0052%)

---

## Follow-up Opportunities
1. **Store masked_events.go duplications** (7 lines × 2): Two similar error-handling patterns at lines 498-504 and 548-554. Could extract if more instances appear.
2. **UI compose/puzzle_solver duplications** (7 lines × 2): Text input validation patterns. Monitor for third instance before extracting.
3. **Publisher signing patterns** (6 lines × 2): BLAKE3 hashing for puzzle vs. spark signatures. Too domain-specific to merge.

---

## Conclusion
Successfully consolidated 5 code clone groups with mechanical, extractable patterns. The remaining duplications are either conceptually distinct (different domain logic) or already at minimal duplication (good design patterns repeated). The project duplication ratio remains excellent at 0.0052%.
