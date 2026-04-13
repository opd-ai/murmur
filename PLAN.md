# Implementation Plan: Network Integration Layer

## Project Context
- **What it does**: MURMUR is a decentralized, peer-to-peer social network with dual-layer identity (Surface + Anonymous/Specter), using a Pulse Map spatial UI instead of traditional feeds.
- **Current goal**: Make the network functional — enable Waves to be sent/received between peers, completing the core value proposition.
- **Estimated Scope**: **Large** (>15 items affecting goal-critical paths, 318 remaining roadmap items)

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|-------------|----------------|---------------------|
| P2P content propagation ("Waves ripple through mesh") | ❌ Not functional | **Yes** — Steps 1-3 |
| GossipSub message handlers | ❌ Missing | **Yes** — Step 1 |
| Cross-subsystem event bus | ❌ Missing | **Yes** — Step 2 |
| Mini-game Bbolt persistence | ❌ Missing | **Yes** — Step 4 |
| Resonance gating enforcement | ❌ Missing | **Yes** — Step 5 |
| Circuit rotation timer | ❌ Not started | **Yes** — Step 6 |
| Pulse Map rendering | ⚠️ Stubs only | No — deferred |
| UI panels (Compose, Settings) | ❌ Missing | No — deferred |

## Metrics Summary

| Metric | Value | Assessment |
|--------|-------|------------|
| Lines of code | 8,847 | Solid foundation |
| Total functions | 313 | Well-decomposed |
| Functions > complexity 9.0 | **2** | ✅ Small scope |
| Duplication ratio | **0.38%** (75 lines) | ✅ Excellent |
| Doc coverage (overall) | **83.3%** | ✅ Good |
| Test coverage (median) | ~85% | ✅ Good |
| Lowest coverage packages | relay (47.9%), mesh (56.4%), transport (60.5%) | ⚠️ Needs attention |

### Complexity Hotspots on Goal-Critical Paths
- `closeSubsystems` (12.7) — `pkg/app/app.go` — cleanup logic, acceptable
- `HandleTouchMove` (10.6) — `pkg/pulsemap/interaction/touch.go` — not goal-critical
- `initIdentity` (8.8) — `pkg/app/app.go` — goal-critical, borderline acceptable

### Code Duplication (4 Clone Pairs)
| Location | Lines | Severity |
|----------|-------|----------|
| `overlays.go` ↔ `overlays_stub.go` | 32 | violation |
| `councils.go` (two internal) | 18 | warning |
| `gifts.go` ↔ `marks.go` | 13 | warning |
| `bootstrap_screen.go` ↔ `completion_screen.go` | 12 | warning |

### TODO Comments in Codebase
1. `pkg/config/config.go:14` — Replace bootstrap node addresses (blocked on infrastructure)
2. `pkg/app/app.go:201` — Add keystore passphrase support

---

## Implementation Steps

### Step 1: Implement GossipSub Message Handlers ✅ COMPLETED
- **Deliverable**: Create `pkg/app/handlers.go` with handler functions for all 4 GossipSub topics (`/murmur/waves/1`, `/murmur/identity/1`, `/murmur/shroud/1`, `/murmur/pulse/1`). Each handler: deserialize MurmurEnvelope protobuf, validate signature, verify timestamp ±300s, check PoW, deduplicate by message_id, dispatch to appropriate store.
- **Dependencies**: None (builds on existing `pkg/networking/gossip/` infrastructure)
- **Goal Impact**: **CRITICAL** — Enables network to receive content. Currently "the network is deaf" per GAPS.md.
- **Acceptance**: 
  - `go test ./pkg/app/... -run TestWaveHandler -v` passes ✅
  - Handler registered in `App.Run()` after topic join ✅
  - Integration test: two in-process hosts, one publishes Wave, other receives and stores
- **Validation**: `go test ./pkg/app/... -cover | grep 'pkg/app'` shows handler coverage
- **Completed**: 2026-04-13 — Created `pkg/app/handlers.go` with `Handlers` struct, validation for all message types, deduplication, callbacks.

### Step 2: Implement Event Bus for Cross-Subsystem Communication
- **Deliverable**: Create `pkg/app/eventbus.go` with:
  - `EventBus` struct with typed channels per event category
  - `EventType` enum: `NetworkEvent`, `WaveEvent`, `IdentityEvent`, `TimerEvent`, `UserActionEvent`
  - `Subscribe(EventType, chan<- Event)` method
  - Fan-out goroutine started in `Run()`
- **Dependencies**: Step 1 (handlers emit events to bus)
- **Goal Impact**: **HIGH** — Per TECHNICAL_IMPLEMENTATION.md §8, enables decoupled subsystem communication. Required for wave notifications, timer-driven GC, heartbeat broadcasts.
- **Acceptance**:
  - `go test ./pkg/app/... -run TestEventBus -v` passes
  - Demonstrates: emit event → all subscribers receive
  - No direct method calls between subsystems for notification paths
- **Validation**: `go-stats-generator analyze ./pkg/app/ --skip-tests --format json | jq '.overview'` shows EventBus types

### Step 3: Implement Wave Publishing
- **Deliverable**: Add `BroadcastWave(ctx context.Context, wave *pb.Wave) error` to `pkg/app/App`. Method wraps Wave in MurmurEnvelope (with signature, timestamp, message_id), serializes via protobuf, calls `PubSub.Publish()` on `/murmur/waves/1`.
- **Dependencies**: Step 1 (handlers must exist to receive published waves)
- **Goal Impact**: **CRITICAL** — Enables users to share content. Completes the "send" half of P2P messaging.
- **Acceptance**:
  - `go test ./pkg/app/... -run TestBroadcastWave -v` passes
  - Integration test: publish Wave, verify peer receives via handler from Step 1
  - Propagation latency <500ms across 3 hops (per TECHNICAL_IMPLEMENTATION.md)
- **Validation**: `go test ./pkg/app/... -cover | grep 'pkg/app'` shows broadcast coverage >80%

### Step 4: Add Bbolt Persistence to Game Mechanics
- **Deliverable**: Modify all 9 game stores in `pkg/anonymous/mechanics/` to accept `*store.DB` and persist state:
  - `GiftStore` — gifts bucket
  - `MarkStore` — marks bucket
  - `PuzzleStore` — puzzles bucket
  - `HuntStore` — hunts bucket
  - `TerritoryManager` — territory bucket
  - `OraclePoolStore` — oracles bucket
  - `ForgeStore` — forge bucket
  - `ShadowPlayStore` — shadowplay bucket
  - `CouncilStore` — councils bucket
- **Dependencies**: None (uses existing `pkg/store/` infrastructure)
- **Goal Impact**: **CRITICAL** — Game state survives restart. Users don't lose hours of Council progress.
- **Acceptance**:
  - `go test ./pkg/anonymous/mechanics/... -run TestPersistence -v` passes
  - Test: create gift → restart app (reopen DB) → gift retrievable
  - All 9 stores have `Load()` method called on init
- **Validation**: `go test ./pkg/anonymous/mechanics/... -cover | grep mechanics` shows >75% coverage

### Step 5: Enforce Resonance Gating on Privileged Actions
- **Deliverable**: Add `enforceResonance(specterPubKey []byte, minResonance int) error` helper. Integrate into:
  - `GiftStore.Create()` — requires Shade (25)
  - `GiftStore.Create()` expanded tiers — requires Wraith (50)
  - `PuzzleStore.Create()` — requires Wraith (50)
  - `HuntStore.NewHunt()` — requires Shade-Wraith (75)
  - `MarkStore.PlaceMark()` — requires Phantom (100)
  - `OraclePoolStore.Create()` — requires Phantom (100)
  - `ShadowPlayStore.Create()` — requires Revenant (200)
  - `CouncilStore.Create()` — requires Revenant (200)
- **Dependencies**: Step 4 (persistence must exist before gating)
- **Goal Impact**: **HIGH** — Anonymous Layer social contract depends on earned Resonance. Without gating, Sybil attacks trivialize all mechanics.
- **Acceptance**:
  - `go test ./pkg/anonymous/mechanics/... -run TestResonanceGating -v` passes
  - Test: attempt gift creation with Resonance=0 → `ErrInsufficientResonance`
  - Test: create gift with Resonance=30 → success
- **Validation**: `grep -c "enforceResonance" pkg/anonymous/mechanics/*.go` returns ≥8

### Step 6: Wire Shroud Circuit Rotation Timer
- **Deliverable**: In `pkg/app/app.go`, after Shroud subsystem initialization, start circuit rotation:
  ```go
  go a.subsystems.CircuitManager.StartRotation(a.ctx)
  ```
  Verify `StartRotation()` creates backup circuit, swaps primary every 10 minutes, tears down old circuit cleanly.
- **Dependencies**: None (method already exists at `pkg/anonymous/shroud/shroud.go:423-436`)
- **Goal Impact**: **HIGH** — Per SHADOW_GRADIENT.md, 10-minute rotation is core anonymity property. Currently not enforced.
- **Acceptance**:
  - `go test ./pkg/anonymous/shroud/... -run TestCircuitRotation -v` passes
  - Test (with short interval): rotation occurs, old circuit destroyed, traffic continues on new circuit
- **Validation**: `go test ./pkg/anonymous/shroud/... -cover | grep shroud` shows >90% coverage maintained

### Step 7: Increase Test Coverage on Networking Packages
- **Deliverable**: Add tests to bring low-coverage networking packages above 70%:
  - `pkg/networking/relay/` — from 47.9% to >70% (add `ForwardStream` error paths, capacity limits, concurrent close)
  - `pkg/networking/mesh/` — from 56.4% to >70% (add churn handling, partition detection)
  - `pkg/networking/transport/` — from 60.5% to >70% (add transport fallback, connection limits)
- **Dependencies**: Steps 1-3 (handlers provide realistic test scenarios)
- **Goal Impact**: **MEDIUM** — Network layer is goal-critical; low coverage risks undetected bugs in production.
- **Acceptance**:
  - `go test -cover ./pkg/networking/relay/...` shows >70%
  - `go test -cover ./pkg/networking/mesh/...` shows >70%
  - `go test -cover ./pkg/networking/transport/...` shows >70%
- **Validation**: `go test -cover ./pkg/networking/... 2>&1 | grep -E "coverage: [0-9]+" | awk '{print $2, $5}'`

### Step 8: Extract Duplicated Code in Overlays
- **Deliverable**: Extract the 32-line duplicate between `pkg/pulsemap/overlays/overlays.go:88-119` and `overlays_stub.go:82-110` into a shared helper function (e.g., `applyLayerBlendCommon()`).
- **Dependencies**: None
- **Goal Impact**: **LOW** — Reduces maintenance burden and duplication ratio.
- **Acceptance**:
  - `go-stats-generator analyze . --skip-tests --format json | jq '.duplication.largest_clone_size'` returns <32
  - All existing overlay tests pass: `go test ./pkg/pulsemap/overlays/...`
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '.duplication.clone_pairs'` returns ≤3

---

## Validation Commands

```bash
# Full test suite with race detector
go test -race ./...

# Coverage report (goal: >80% median)
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -20

# Metrics regeneration
go-stats-generator analyze . --skip-tests --format json > /tmp/metrics.json

# Check complexity hotspots
cat /tmp/metrics.json | jq '[.functions[] | select(.complexity.overall > 9.0)] | length'

# Check duplication
cat /tmp/metrics.json | jq '.duplication.duplication_ratio'

# Check doc coverage
cat /tmp/metrics.json | jq '.documentation.coverage.overall'

# Networking package coverage check
go test -cover ./pkg/networking/... 2>&1 | grep -E "^ok"
```

---

## Notes

- **Deferred work**: Pulse Map rendering and UI panels are important but not blocking the network integration layer. They can be addressed in a subsequent plan once the network is functional.
- **Bootstrap nodes**: The TODO in `pkg/config/config.go:14` cannot be resolved through code changes — it requires community infrastructure. Marked as blocked.
- **Proto coverage**: The 10.2% coverage on `proto/` package is acceptable as it consists of generated code; round-trip tests are covered in consuming packages.

---

*Generated 2026-04-13 from go-stats-generator metrics + ROADMAP.md/AUDIT.md/GAPS.md analysis*
