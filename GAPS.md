# Implementation Gaps — 2026-04-13

This document identifies the gaps between MURMUR's stated goals and its current implementation, organized by impact and urgency.

---

## 1. Network Message Handling

- **Stated Goal**: "The network replaces the infinite scroll with a living map" (README.md). "Waves ripple outward through the mesh" (README.md:5). Per TECHNICAL_IMPLEMENTATION.md §3.1, GossipSub handlers should validate envelopes, verify PoW, check timestamps, and dispatch to storage.

- **Current State**: GossipSub topics are joined (`/murmur/waves/1`, `/murmur/identity/1`, `/murmur/shroud/1`, `/murmur/pulse/1`) but no `MessageHandler` functions are registered. The `Subscribe()` method exists in `pkg/networking/gossip/gossip.go:127-143` and accepts handlers, but `pkg/app/app.go` never calls it. Incoming messages from peers are silently dropped. The network is deaf.

- **Impact**: **CRITICAL** — The core value proposition (P2P social network) is non-functional. Users cannot see each other's Waves, cannot discover peers through identity announcements, cannot find Shroud relays, and cannot receive heartbeats. The application is a local-only tool despite having full network infrastructure.

- **Closing the Gap**:
  1. Create `pkg/app/handlers.go` with handler functions for each topic:
     ```go
     func (a *App) handleWaveMessage(ctx context.Context, msg *pubsub.Message) {
         envelope := &pb.MurmurEnvelope{}
         if err := proto.Unmarshal(msg.Data, envelope); err != nil { return }
         if !a.validateEnvelope(envelope) { return }
         wave := &pb.Wave{}
         if err := proto.Unmarshal(envelope.Payload, wave); err != nil { return }
         a.waveCache.Put(wave)
     }
     ```
  2. In `Run()`, after joining topics, call `ps.Subscribe(ctx, gossip.TopicWaves, a.handleWaveMessage)` for each topic.
  3. Add integration test with two in-process libp2p hosts verifying message delivery.

---

## 2. Wave Publishing

- **Stated Goal**: "Signed, ephemeral text message (≤2048 bytes) with PoW and TTL" that "ripples outward through the mesh" (README.md Quick Reference). Per WAVES.md, Waves should propagate via GossipSub with hop counting.

- **Current State**: `pkg/content/waves/Create()` builds valid Waves with BLAKE3 IDs, Ed25519 signatures, and SHA-256 PoW. `pkg/content/storage/Cache.Put()` persists them locally. However, no code path ever calls `PubSub.Publish()` to send Waves to peers. The `Publish()` method exists and works (`pkg/networking/gossip/gossip.go:177-183`), but is never invoked for Wave content.

- **Impact**: **CRITICAL** — Users cannot share content with the network. Waves are created, stored locally, and never transmitted. The "social network" aspect is completely absent.

- **Closing the Gap**:
  1. Add `BroadcastWave(ctx context.Context, wave *pb.Wave) error` method to `pkg/app/App`:
     ```go
     func (a *App) BroadcastWave(ctx context.Context, wave *pb.Wave) error {
         envelope := wrapInEnvelope(wave, a.subsystems.Identity)
         data, _ := proto.Marshal(envelope)
         return a.subsystems.PubSub.Publish(ctx, gossip.TopicWaves, data)
     }
     ```
  2. Wire this to UI action (when Compose panel is implemented) or expose via API.
  3. Verify with test: create Wave, broadcast, assert peers receive via handler.

---

## 3. Persistent Storage for Game Mechanics

- **Stated Goal**: Per ANONYMOUS_GAME_MECHANICS.md, Phantom Gifts, Specter Marks, Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, and Phantom Councils should persist across sessions. ROADMAP.md:416-550 lists "Bbolt persistence" as incomplete for all 9 mechanics.

- **Current State**: All 7 implemented game stores (`GiftStore`, `MarkStore`, `PuzzleStore`, `HuntStore`, `TerritoryManager`, `OraclePoolStore`, `ForgeStore`, `ShadowPlayStore`, `CouncilStore`) use in-memory `map[string]*` structures. No store accepts a `*store.DB` parameter. Application restart resets all game state — active hunts lost, councils dissolved, gifts vanished.

- **Impact**: **CRITICAL** — Game mechanics are transient. Users who close the application lose all progress. Councils that took hours to form disappear. This breaks trust in the Anonymous Layer's social contracts.

- **Closing the Gap**:
  1. Add `*store.DB` field to each store struct.
  2. Modify constructors: `NewGiftStore(db *store.DB)`.
  3. Replace `m.gifts[id] = gift` with:
     ```go
     data, _ := proto.Marshal(gift.ToProto())
     db.Put(store.BucketGifts, []byte(id), data)
     ```
  4. Add `Load()` method to restore state on startup.
  5. Add `//go:build persistence` integration tests.

---

## 4. Event Bus for Cross-Subsystem Communication

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §8, "A central event bus goroutine receives all events (network, timer, user action) and fans them out to subscriber channels." The package comment in `pkg/app/app.go:1-4` explicitly states: "the event bus uses channel fan-out for decoupled communication between subsystems."

- **Current State**: No `EventBus` type exists. No event channels, no subscriber registry, no fan-out goroutine. Subsystems communicate only through direct method calls on shared `Subsystems` struct. This couples components tightly and prevents features like "wave received" notifications, timer-driven GC, or heartbeat broadcasts.

- **Impact**: **HIGH** — Without an event bus, implementing features like "notify Pulse Map when Wave received," "broadcast heartbeat every 30s," or "trigger GC every 60s" requires ad-hoc goroutine spawning and direct coupling. The architecture diverges from its own specification.

- **Closing the Gap**:
  1. Create `pkg/app/eventbus.go`:
     ```go
     type EventType int
     const (
         EventWaveReceived EventType = iota
         EventPeerConnected
         EventTimerTick
     )
     type Event struct { Type EventType; Data any }
     type EventBus struct {
         subscribers map[EventType][]chan<- Event
         mu sync.RWMutex
     }
     func (eb *EventBus) Subscribe(et EventType, ch chan<- Event)
     func (eb *EventBus) Publish(e Event)
     ```
  2. Initialize in `App.Run()` before subsystem init.
  3. Wire network handlers, timers, and UI to publish events.
  4. Verify with test: publish event, assert N subscribers receive.

---

## 5. Shroud Relay Network Discovery

- **Stated Goal**: Per SHADOW_GRADIENT.md, "Shroud relays are discovered via Beacon Waves broadcast on the `/murmur/shroud/1` topic." Fortress-mode nodes should automatically discover relays and build circuits without manual configuration.

- **Current State**: `Beacon.AddRelay()` in `pkg/anonymous/shroud/shroud.go:102-112` exists but requires direct invocation with relay info. No code broadcasts `RelayAdvertisement` protobufs. No code listens on `/murmur/shroud/1` for incoming advertisements. The `BeaconInterval = 5 * time.Minute` constant is defined but unused.

- **Impact**: **HIGH** — Anonymous Layer is non-functional in practice. Users cannot discover Shroud relays automatically. Circuits can only be built with manually-configured peers, defeating the decentralized goal.

- **Closing the Gap**:
  1. Add `BroadcastBeacon(ctx context.Context, ps *gossip.PubSub) error` to `Beacon`:
     ```go
     func (b *Beacon) BroadcastBeacon(ctx context.Context, ps *gossip.PubSub) error {
         ad := &pb.RelayAdvertisement{PublicKey: b.publicKey[:], Bandwidth: b.selfInfo.Bandwidth}
         data, _ := proto.Marshal(ad)
         return ps.Publish(ctx, gossip.TopicShroud, data)
     }
     ```
  2. Start periodic goroutine: `go b.runBeaconBroadcaster(ctx, ps)`.
  3. Add handler in Shroud topic subscriber to call `AddRelay()` on valid advertisements.
  4. Verify with 3-node simulation: A advertises, B/C discover, C builds circuit through B.

---

## 6. Resonance Gating Enforcement

- **Stated Goal**: Per ANONYMOUS_GAME_MECHANICS.md and RESONANCE_SYSTEM.md, game mechanics require minimum Resonance thresholds: Phantom Gifts (25+), Cipher Puzzles (50+), Specter Hunts (75+), Masked Events (100+), Shadow Play (200+), Phantom Councils (200+ Fortress).

- **Current State**: Threshold constants are defined (`GiftTierBasic = 25`, etc.) but `GiftStore.Create()`, `NewHunt()`, `NewPuzzle()`, etc. do not verify the caller's Resonance before allowing creation. Any Specter can invoke any mechanic regardless of milestone.

- **Impact**: **HIGH** — The progression system is bypassed. New Specters with 0 Resonance can use premium features. This undermines the game loop that incentivizes participation.

- **Closing the Gap**:
  1. Add `ResonanceVerifier` interface:
     ```go
     type ResonanceVerifier interface {
         ScoreFor(specterPubKey [32]byte) (float64, error)
     }
     ```
  2. Inject verifier into each store constructor.
  3. At start of `Create()`, check: `if score < minRequired { return ErrInsufficientResonance }`.
  4. Verify with test: create mechanic with low Resonance, expect error.

---

## 7. Pulse Map Rendering

- **Stated Goal**: "The network is the interface. No feed. You navigate a real-time spatial graph — the Pulse Map" (README.md). Per PULSE_MAP.md, nodes should render as circles with sigil overlays, halos, and animated edges. Target: 60fps with 500 nodes.

- **Current State**: `pkg/pulsemap/layout/` computes node positions correctly (Fruchterman-Reingold + Barnes-Hut). `pkg/pulsemap/rendering/rendering.go` defines `Renderer` struct (160 lines) but contains minimal logic. `rendering_stub.go` provides placeholders. No `Draw(*ebiten.Image)` implementation that actually renders nodes to screen.

- **Impact**: **HIGH** — The "Pulse Map as primary interface" goal is unmet. Users see nothing despite layout computation working correctly. The unique differentiator of MURMUR is invisible.

- **Closing the Gap**:
  1. Implement `Renderer.Draw(screen *ebiten.Image)`:
     ```go
     func (r *Renderer) Draw(screen *ebiten.Image) {
         positions := r.engine.Positions().Get()
         for id, pos := range positions {
             r.drawNode(screen, id, pos)
         }
         for _, edge := range r.edges {
             r.drawEdge(screen, edge)
         }
     }
     ```
  2. Add Ebitengine `Game` implementation in `pkg/pulsemap/` or `cmd/murmur/`.
  3. Verify with headless Ebitengine screenshot test comparing output to reference.

---

## 8. UI Panels

- **Stated Goal**: ROADMAP.md:657-670 and DESIGN_DOCUMENT.md specify: Quick-Action Radial Menu, Node Detail Panel, Compose Wave panel, Settings panel, Search bar, Bookmarks.

- **Current State**: Zero UI implementation. No immediate-mode panel code. No text input. No buttons. The application has no way for users to compose Waves, change settings, or interact with nodes.

- **Impact**: **MEDIUM** — Without UI, the application is developer-only. Users cannot perform any action. This is expected for v0.1-v0.4 milestones, but blocks all user-facing testing.

- **Closing the Gap**:
  1. Create `pkg/ui/` package with Ebitengine immediate-mode widgets.
  2. Priority order: (1) Compose panel, (2) Settings panel, (3) Node detail panel.
  3. Wire Compose panel to `App.BroadcastWave()`.
  4. Verify with manual testing and screenshot comparison.

---

## 9. Privacy Mode Transition Enforcement

- **Stated Goal**: Per SHADOW_GRADIENT.md, downgrading from Hybrid→Open should destroy Specter identity. Upgrading from Open→Hybrid should preserve any existing Specter. Traffic padding should activate for Guarded/Fortress modes.

- **Current State**: `pkg/identity/modes/manager.go` has `CanTransition()` logic returning valid transitions, but `Transition()` method does not: (1) destroy Specter on downgrade, (2) generate Specter on upgrade to Hybrid, (3) start traffic padding for Guarded/Fortress. Mode changes are tracked but side effects are not enforced.

- **Impact**: **MEDIUM** — Privacy guarantees are not enforced. User who downgrades to Open still has Specter keys accessible. User who upgrades to Guarded does not get traffic padding protection.

- **Closing the Gap**:
  1. In `Transition()`, add side-effect handlers:
     ```go
     if from.HasSpecter() && !to.HasSpecter() {
         m.specterStore.Destroy()
         m.specterKeyPair.ZeroAnonymousKeyPair()
     }
     if !from.HasSpecter() && to.HasSpecter() {
         kp, _ := keys.GenerateAnonymousKeyPair()
         m.specterKeyPair = kp
     }
     if to == ModeGuarded || to == ModeFortress {
         go m.startTrafficPadding(ctx)
     }
     ```
  2. Verify with state machine tests covering all 12 transitions.

---

## 10. Circuit Rotation Timer

- **Stated Goal**: Per SHADOW_GRADIENT.md, Shroud circuits rotate every 10 minutes for security. `CircuitRotationInterval = 10 * time.Minute` is defined.

- **Current State**: `CircuitManager.StartRotation(ctx)` method exists in `pkg/anonymous/shroud/shroud.go:423-436` but is never called from `pkg/app/`. Circuits are created once and never rotated. The 10-minute security property is not enforced.

- **Impact**: **MEDIUM** — Long-lived circuits increase correlation attack surface. Traffic analysis becomes easier when circuits persist indefinitely.

- **Closing the Gap**:
  1. In `pkg/app/app.go` during Shroud subsystem init, call:
     ```go
     go a.circuitMgr.StartRotation(a.ctx)
     ```
  2. Verify with test that asserts `circuit.IsExpired()` returns true after 10 minutes and `GetCircuit()` returns different circuit.

---

## Summary Table

| Gap | Severity | Effort | Priority |
|-----|----------|--------|----------|
| Network message handlers | CRITICAL | Medium | 1 |
| Wave publishing | CRITICAL | Low | 2 |
| Bbolt persistence for mechanics | CRITICAL | High | 3 |
| Event bus | HIGH | Medium | 4 |
| Shroud relay discovery | HIGH | Medium | 5 |
| Resonance gating | HIGH | Low | 6 |
| Pulse Map rendering | HIGH | Medium | 7 |
| UI panels | MEDIUM | High | 8 |
| Privacy mode transitions | MEDIUM | Low | 9 |
| Circuit rotation timer | MEDIUM | Low | 10 |

**Estimated effort to close all gaps:** 3-4 developer-weeks focused work.

**Recommended approach:** Close gaps 1-3 first (network + persistence) to achieve a functional P2P prototype, then 4-6 (architecture + enforcement), then 7-10 (UI + polish).
