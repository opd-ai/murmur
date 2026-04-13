# AUDIT — 2026-04-13

## Project Goals

MURMUR is described as a **decentralized, peer-to-peer social network with dual-layer identity**. The README and design documents promise:

1. **No servers, no algorithms, no permanent record** — fully P2P architecture with ephemeral content
2. **Pulse Map as primary interface** — real-time force-directed graph replacing traditional feeds
3. **Self-sovereign identity** — Ed25519 keypairs with no registration, no third parties
4. **Privacy is structural** — Noise-encrypted transport, onion-routed anonymous traffic
5. **Dual-layer identity** — Surface Layer (public) + Anonymous Layer (Specters via Shroud)
6. **Ephemeral content** — Waves with PoW and TTL (7-day default, 30-day max)
7. **Anonymous mechanics** — Phantom Gifts, Specter Marks, mini-games requiring Resonance thresholds
8. **Six-phase onboarding** — guided introduction from Welcome → Identity → Mode → Bootstrap → Exploration → First Action

Target audience: privacy-conscious users, self-sovereign identity advocates, communities wanting anonymous social mechanics as a first-class feature.

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Go module with dependencies | ✅ Achieved | `go.mod`:1-132 — libp2p v0.48, Ebitengine v2.9, Bbolt, protobuf, BLAKE3 |
| Ed25519 Surface identity | ✅ Achieved | `pkg/identity/keys/keys.go`:44-54 — generation, signing, verification |
| Curve25519 Anonymous identity | ✅ Achieved | `pkg/identity/keys/keys.go`:58-76 — Specter keypair with X25519 clamping |
| Argon2id key derivation | ✅ Achieved | `pkg/identity/keys/keys.go`:103-149 — time=3, memory=64MiB, threads=4 |
| Sigil generation | ✅ Achieved | `pkg/identity/sigils/` — deterministic 64×64 from public key hash |
| Wave creation with PoW/TTL | ✅ Achieved | `pkg/content/waves/waves.go`:76-141 — 8 types, BLAKE3 ID, Ed25519 sig, SHA-256 PoW |
| Three-hop Shroud circuits | ✅ Achieved | `pkg/anonymous/shroud/shroud.go`:230-257 — X25519 key exchange, XChaCha20-Poly1305 layers |
| Force-directed Pulse Map layout | ✅ Achieved | `pkg/pulsemap/layout/layout.go`:174-306 — Fruchterman-Reingold + Barnes-Hut for >500 nodes |
| Double-buffered position swap | ✅ Achieved | `pkg/pulsemap/layout/layout.go`:62-82 — `atomic.Pointer` for lock-free handoff |
| GossipSub with peer scoring | ✅ Achieved | `pkg/networking/gossip/gossip.go`:44-105 — IP colocation penalty, invalid message penalty |
| Bbolt storage layer | ✅ Achieved | `pkg/store/db.go` — 7 buckets, CRUD operations, batch transactions |
| Privacy mode enum | ✅ Achieved | `pkg/identity/modes/` — Open/Hybrid/Guarded/Fortress with descriptions |
| Mini-games (Puzzles, Hunts, etc.) | ✅ Achieved | `pkg/anonymous/mechanics/` — 7 game types with state machines, scoring |
| Onboarding flow controller | ✅ Achieved | `pkg/onboarding/flow/flow.go` — 6-phase state machine with callbacks |
| Resonance scoring | ✅ Achieved | `pkg/anonymous/resonance/` — signals, milestones, decay framework |
| libp2p host with Noise/QUIC/TCP | ✅ Achieved | `pkg/networking/transport/` — host construction with DHT support |
| All tests pass with race detector | ✅ Achieved | `go test -race ./...` — 36 packages, 0 failures |
| Event bus for cross-subsystem comms | ⚠️ Partial | `pkg/app/app.go`:1-4 comments describe it, no implementation |
| GossipSub message handlers | ❌ Missing | Topics joined but no handlers registered (ROADMAP.md:107-119) |
| Wave network propagation | ❌ Missing | `Publish()` exists but never called for Waves |
| Shroud relay discovery via network | ❌ Missing | Manual `AddRelay()` only; no beacon broadcast |
| Circuit rotation timer | ❌ Missing | `CircuitRotationInterval` defined but timer not wired |
| Resonance gating enforcement | ❌ Missing | Thresholds defined but not enforced before actions |
| Bbolt persistence for mechanics | ❌ Missing | All stores use in-memory maps, not `pkg/store` |
| Pulse Map rendering | ⚠️ Partial | Layout done, rendering stubs exist, no actual Ebitengine draw calls |
| UI panels (Compose, Settings, etc.) | ❌ Missing | No UI implementation beyond layout engine |
| ~8 persistent goroutines | ⚠️ Partial | Only main, network, and context goroutines; missing 5 |

---

## Findings

### CRITICAL

- [x] **GossipSub message handlers not implemented** — `pkg/networking/gossip/gossip.go`:127-143 — Topics `/murmur/waves/1`, `/murmur/identity/1`, `/murmur/shroud/1`, `/murmur/pulse/1` are joined but no handlers are registered. Incoming messages are silently discarded. The `MessageHandler` type and `Subscribe()` method exist but are never called from `pkg/app/app.go`. **Remediation:** In `pkg/app/app.go`, after `ps.Join(topic)`, call `ps.Subscribe(ctx, topic, makeWaveHandler(cache, validator))` for each topic. Implement handlers in a new `pkg/app/handlers.go` that validate MurmurEnvelope, verify PoW, check timestamp ±300s, and dispatch to storage. Verify with `go test ./pkg/app/... -v -run TestMessageHandler`. **COMPLETED 2026-04-13**: Created `pkg/app/handlers.go` with handlers for all 4 topics, wired into `App.initContent()`.

- [x] **Waves never published to network** — `pkg/content/waves/waves.go`, `pkg/content/propagation/` — `Create()` builds Waves locally but no code path calls `PubSub.Publish()` to send them to peers. Content is write-only to local cache. **Remediation:** Add `PublishWave(ctx, wave)` method to `pkg/app/` that serializes Wave to MurmurEnvelope protobuf and calls `a.subsystems.PubSub.Publish(ctx, gossip.TopicWaves, envelope)`. Wire this to a user action in the UI layer. Verify with integration test: two hosts, one publishes, other receives. **COMPLETED 2026-04-13**: Created `pkg/app/broadcast.go` with `BroadcastWave()`, `BroadcastIdentity()`, `BroadcastHeartbeat()`, and convenience methods `CreateSurfaceWave()`, `CreateReplyWave()`. All tests pass.

- [x] **Mini-game stores lack Bbolt persistence** — `pkg/anonymous/mechanics/gifts.go`:86-125, `hunts.go`, `puzzles.go`, `councils.go`, `oracle.go`, `forge.go`, `shadowplay.go` — All 7 game mechanics use `map[string]*` in-memory storage. Application restart loses all active games, gifts, marks, and council state. **Remediation:** Inject `*store.DB` into each `*Store` constructor. Serialize state via protobuf. Replace `m.gifts[id]` with `db.Put(BucketGifts, id, data)`. Add load-on-init. Verify with `go test ./pkg/anonymous/mechanics/... -v -run TestPersistence`. **COMPLETED 2026-04-14**: Created protobuf definitions in `proto/mechanics.proto` for all 9 game mechanics. Added 10 new Bbolt buckets to `pkg/store/db.go`. Created persistence wrappers for all stores: PersistentGiftStore, PersistentMarkStore, PersistentPuzzleStore, PersistentHuntStore, PersistentCouncilStore, PersistentOracleStore, PersistentForgeStore, PersistentShadowPlayStore, PersistentTerritoryManager. All tests pass.

### HIGH

- [x] **Event bus not implemented** — `pkg/app/app.go`:1-4 — Package comment states "event bus uses channel fan-out" per TECHNICAL_IMPLEMENTATION.md §2, but no `EventBus` type, no typed event channels, no subscriber registration exists. Subsystems cannot communicate events (Wave received, identity updated, timer fired). **Remediation:** Create `pkg/app/eventbus.go` with `type EventBus struct`, typed channels per event type, `Subscribe(EventType, chan<- Event)` method, and fan-out goroutine. Wire in `Run()` before subsystem init. Verify with test: emit event, assert all subscribers receive. **COMPLETED 2026-04-14**: Created `pkg/app/eventbus.go` with 13 event types, typed event structs, Subscribe/Emit API, fan-out dispatch goroutine. Wired into `app.Run()` before other subsystems. Tests pass.

- [x] **Shroud relay discovery via network missing** — `pkg/anonymous/shroud/shroud.go`:101-120 — `Beacon.AddRelay()` requires manual invocation. Per SHADOW_GRADIENT.md, relays should be discovered via Beacon Waves on `/murmur/shroud/1`. No code broadcasts or receives RelayAdvertisement. **Remediation:** Add `BroadcastBeacon(ctx, ps)` method that publishes signed RelayAdvertisement protobuf every `BeaconInterval`. Add handler in message loop to call `AddRelay()` on valid advertisements. Verify with 3-node simulation test. **COMPLETED 2026-04-14**: Created `pkg/anonymous/shroud/advertisement.go` with `GenerateAdvertisement()`, `ValidateAdvertisement()`, `ProcessAdvertisement()`, `PruneExpiredRelays()`. Added `BroadcastRelayAdvertisement()` to `pkg/app/broadcast.go`. Wired periodic broadcasting and relay pruning in `pkg/app/app.go` via `initBeacon()`, `runBeaconLoop()`, `runRelayPruneLoop()`. Added `Beacon` to `Subsystems` struct. Tests pass.

- [x] **Circuit rotation timer not started** — `pkg/anonymous/shroud/shroud.go`:423-436 — `CircuitManager.StartRotation(ctx)` method exists but is never called from `pkg/app/`. Circuits are never rotated; the 10-minute security property is not enforced. **Remediation:** In `pkg/app/app.go` after Shroud subsystem init, call `go circuitMgr.StartRotation(a.ctx)`. Verify with test that asserts new circuit after 10 minutes. **COMPLETED 2026-04-14**: Added `CircuitManager` to `Subsystems` struct. Refactored `initBeacon()` to always create Beacon (for all nodes) and CircuitManager. `StartRotation(a.ctx)` is now called as one of the ~8 persistent goroutines. Tests pass.

- [x] **Resonance gating not enforced** — `pkg/anonymous/mechanics/gifts.go`:135, `hunts.go`:newHuntValidation, `puzzles.go` — Code defines `GiftTierBasic = 25`, `HuntMinResonance = 75`, etc., but creation methods do not verify caller's Resonance score. Any Specter can create any mechanic regardless of milestone. **Remediation:** Add `enforceResonance(specterPubKey, minResonance)` check to `GiftStore.Create()`, `NewHunt()`, etc. Wire Resonance lookup to `ResonanceComputer.ScoreFor(specterPubKey)`. Verify with test: attempt to create gift with Resonance 0, expect `ErrInsufficientResonance`. **COMPLETED 2026-04-14**: Added `initiatorResonance` parameter to `NewHunt()` (validates against `HuntMinResonance=75`), `NewSigilForge()` (validates against `ForgeMinResonance=50`), `NewPhantomCouncil()` (validates against `CouncilMinResonance=200`). `GiftStore.Create()` and `MarkStore.Create()` already enforced resonance. Added tests: `TestHuntInsufficientResonance`, `TestForgeInsufficientResonance`, `TestCouncilInsufficientCreatorResonance`. All tests pass.

- [x] **Privacy mode transition not enforced** — `pkg/identity/modes/manager.go` — `CanTransition()` logic exists but `Transition()` does not enforce Specter preservation/destruction rules. Downgrading Hybrid→Open should destroy Specter keypair; currently does nothing. **Remediation:** In `Transition()`, if `from.HasSpecter() && !to.HasSpecter()`, call `specterStore.Destroy()` and zero key material. Verify with state machine test. **COMPLETED 2026-04-14**: Added `specterDestroyer` callback to `Manager`, `SetSpecterDestroyer()` method, and updated `Transition()` to call destroyer when transitioning from Specter-enabled mode (Hybrid/Guarded/Fortress) to Open. After destruction, `hasSpecter` is set false. Added `ErrSpecterDestructionFailed` error. Added 4 tests: `TestTransitionDestroysSpecter`, `TestTransitionToOpenPreservesSpecterIfDestroyed`, `TestTransitionSpecterDestroyerFails`, `TestTransitionNoDestroyerWhenSpecterStaysEnabled`. All tests pass.

- [x] **Networking relay package coverage 47.9%** — `pkg/networking/relay/` — Less than 50% test coverage on critical NAT traversal code. `ForwardStream`, `Close` error paths untested. **Remediation:** Add tests for relay capacity limits, stream forwarding errors, and concurrent close. Target >80% coverage. Verify with `go test -cover ./pkg/networking/relay/...`. **COMPLETED 2026-04-14**: Added 8 new tests: `TestSetHolePunchService`, `TestDirectConnectNoService`, `TestConnectViaRelayNoRelays`, `TestConnectViaRelayFailedConnect`, `TestMakeReservationFailedConnect`, `TestHostOptionsDisabled`, `TestRelaysReturnsDefensiveCopy`. Coverage improved from 47.9% to 76.1%. Remaining uncovered code is successful relay connection paths requiring real relay nodes (integration test scope).

### MEDIUM

- [ ] **Pulse Map rendering stubs only** — `pkg/pulsemap/rendering/rendering_stub.go` — Full rendering implementation exists but `rendering.go` contains minimal logic (160 lines). No actual `Draw()` calls to Ebitengine; layout positions computed but never rendered. **Remediation:** Complete `Renderer.Draw(screen *ebiten.Image)` to iterate `PositionBuffer.Get()` and draw nodes/edges. Verify with Ebitengine headless screenshot test.

- [ ] **No UI panels** — ROADMAP.md:657-670 lists Quick-Action Radial Menu, Node Detail Panel, Search bar, Compose panel, Settings panel — none exist. User cannot interact with network beyond code. **Remediation:** Implement `pkg/ui/` package with Ebitengine immediate-mode panels. Priority: Compose panel for first Wave, then Settings.

- [ ] **cmd/murmur coverage 0%** — `cmd/murmur/main.go` — Entry point untested. Version variable injection, signal handling, flag parsing not verified. **Remediation:** Add `TestMainRuns` that invokes main() with mock args and checks exit code. Verify with `go test ./cmd/murmur/... -cover`.

- [ ] **Onboarding screens are stubs** — `pkg/onboarding/screens/*_stub.go` — All 10 screen files are stubs returning placeholder data. No actual Ebitengine rendering. **Remediation:** Implement real screen rendering in non-stub files, guarded by build tag if needed.

- [ ] **Proto package coverage 10.2%** — `proto/*.pb.go` — Generated protobuf code is largely untested. Marshal/unmarshal round-trips not verified for all message types. **Remediation:** Add `TestWaveProtoRoundTrip`, `TestEnvelopeProtoRoundTrip` etc. in `proto/proto_test.go`.

- [ ] **Code duplication 0.38% (75 lines)** — `go-stats-generator` detected 4 clone pairs, largest 32 lines in `pkg/pulsemap/overlays/`. **Remediation:** Extract duplicated `overlays.go:88-119` and `overlays_stub.go:82-110` into shared helper function.

- [ ] **Identifier naming violations** — `go-stats-generator` reports 39 identifier violations including `HuntParticipant` (stuttering), `OraclePoolStore` (stuttering), `TerritoryManager` (stuttering). **Remediation:** Rename to `Participant`, `Store`, `Manager` respectively since package name provides context.

- [ ] **Package `errors` collides with stdlib** — `pkg/errors/errors.go` — Package name shadows Go's standard `errors` package, causing import confusion. **Remediation:** Rename to `pkg/errs/` or `pkg/murerr/`.

### LOW

- [ ] **File naming stuttering** — `pkg/anonymous/resonance/resonance.go`, `pkg/config/config.go`, etc. — 14 files have stuttering names like `config/config.go`. Not a bug, but inconsistent with Go idioms. **Remediation:** Rename to `config/config.go` → `config/loader.go` or similar descriptive name.

- [ ] **Single-letter variables in public APIs** — `pkg/anonymous/mechanics/hunts.go:436` uses `k`, `pkg/anonymous/resonance/claims.go:169` uses `r`. Reduces readability. **Remediation:** Expand to descriptive names (`key`, `resonance`).

- [ ] **Low cohesion files** — `go-stats-generator` reports 14 files with cohesion <0.46, suggesting they mix unrelated concerns. Top offenders: `pkg/onboarding/flow/flow.go` (0.02), `pkg/onboarding/screens/screens_stub.go` (0.03). **Remediation:** Split into focused files by concern.

- [ ] **Missing `Makefile` or `mage`** — ROADMAP.md:37 lists build harness as incomplete. Manual `go build`, `go test`, `gofumpt` required. **Remediation:** Add `Makefile` with targets: `build`, `test`, `lint`, `fmt`, `proto`.

- [ ] **Missing CI pipeline** — ROADMAP.md:38 — No `.github/workflows/` CI configuration. PRs not automatically tested. **Remediation:** Add `ci.yml` with matrix: linux/amd64, darwin/amd64, windows/amd64 running `go test -race ./...`.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total Lines of Code | 8,847 |
| Total Functions | 313 |
| Total Methods | 852 |
| Total Structs | 165 |
| Total Interfaces | 3 |
| Total Packages | 34 |
| Total Files | 64 (source, excl. tests) |
| Average Function Length | 8.9 lines |
| Average Complexity | 3.1 |
| Functions > 50 lines | 0 (0.0%) |
| High Complexity (>10) | 0 functions |
| Highest Complexity | `closeSubsystems` (12.7) |
| Documentation Coverage | 83.3% overall |
| Package Doc Coverage | 91.2% |
| Function Doc Coverage | 91.8% |
| Duplication Ratio | 0.38% |
| Circular Dependencies | None detected |
| Test Coverage (median) | ~85% |
| Lowest Coverage Packages | `networking/relay` (47.9%), `networking/mesh` (56.4%), `networking/transport` (60.5%) |

---

## Validation Commands

```bash
# Full test suite with race detector
go test -race ./...

# Coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Static analysis
go vet ./...
gofumpt -l -d .

# Metrics regeneration
go-stats-generator analyze . --skip-tests

# Simulation tests (if implemented)
go test -tags=simulation -timeout=30m ./...
```

---

## Conclusion

MURMUR has achieved **~31% of its roadmap** (145/463 items). The **data structures and game mechanics are solid** (~68-94% coverage on core packages), but the **network integration layer is non-functional**: messages are never sent or received despite the transport being fully operational. The Pulse Map layout engine is complete, but rendering is stubbed. The privacy mode system and Resonance scoring exist as frameworks without enforcement.

**Highest priority gaps:**
1. Wire GossipSub message handlers to receive content
2. Add `Publish()` calls to send Waves/identities/beacons
3. Persist mini-game state to Bbolt
4. Implement event bus for cross-subsystem coordination
5. Complete Pulse Map rendering

The codebase is well-structured with clean separation of concerns, no circular dependencies, and consistently documented. The foundation is solid for completing the remaining 69% of features.
