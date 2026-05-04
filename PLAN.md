# Implementation Plan: Shadow Gradient Visibility & Production Readiness

**Generated**: 2026-05-04  
**Scope**: Medium (31% complete → v0.1 target)  
**Timeline**: 33 days critical path, 79 days for full v0.1  

---

## Project Context

**What it does**: MURMUR is a decentralized peer-to-peer social network with dual-layer identity (Surface + Anonymous), force-directed graph UI (Pulse Map), and ephemeral content (Waves). No servers, no algorithms, no permanent record.

**Current Goal**: **Achieve Shadow Gradient visibility** — the core product thesis that anonymous layer activity (Phantom Gifts, Specter Marks, mini-games) must be visible to Surface users to create "curiosity-driven pull toward deeper privacy tiers." Per GAPS.md, this is "CRITICAL for growth flywheel" and currently non-functional despite all mechanics being implemented.

**Estimated Scope**: Medium
- 31% implementation complete (145/463 roadmap items)
- Core data structures ✅, game mechanics ✅, cryptography ✅
- Network integration ⚠️, UI rendering ⚠️, cross-layer visibility ❌

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|-------------|---------------|---------------------|
| Shadow Gradient visibility (anonymous artifacts on Surface Layer) | ❌ | **Yes** — Step 1 & 2 |
| Onboarding UX (6-phase flow with screens) | ⚠️ (logic exists, no UI) | **Yes** — Step 3 |
| Security hardening (key zeroing, rate limiting, deduplication) | ⚠️ (partial) | **Yes** — Step 4 |
| Network propagation <500ms across 3 hops | ❓ (no validation) | No — defer to Step 5 |
| Shroud anonymity guarantees | ❓ (no testing) | No — defer to Step 6 |
| Production monitoring (metrics, health checks) | ❌ | No — defer to Step 7 |
| Mobile platform support | ⚠️ (CI missing) | No — defer post-v0.1 |
| Documentation completeness (godoc, ADRs, user guide) | ⚠️ (82% coverage) | No — defer post-v0.1 |

---

## Metrics Summary

**From go-stats-generator analysis** (`/tmp/metrics.json`, 2026-05-04):
- **Total functions**: 5,204
- **Complexity hotspots** (overall > 9.0): 151 functions (2.9%)
- **Highest complexity**: 14.5 (`updateInteract` in pkg/ui/)
- **Top 5 hotspots on critical paths**:
  1. `updateInteract` (ui, 14.5) — UI event handling
  2. `printIncomingWaves` (cli, 14.2) — CLI message display
  3. `attemptReconnection` (mesh, 14.0) — network recovery
  4. `ValidateEnvelope` (proto, 12.7) — message validation
  5. `RecordAmplification` (mechanics, 13.2) — resonance tracking
- **Documentation coverage**: 82.3% overall (functions 92.9%, methods 78.3%)
- **Duplication ratio**: 1.48% (acceptable — UI type definitions, binary serialization, distinct formulas)
- **TODO comments**: 16 (zero flagged as critical/blocking per grep analysis)
- **Anti-patterns detected**: 5 (from static analysis)
- **Linter status**: `go vet ./...` clean (exit 0)

**Critical Gap**: No cross-layer artifact rendering. Per GAPS.md analysis, all 10 anonymous mechanics (Phantom Gifts, Specter Marks, Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Masked Events, Phantom Councils) have complete game logic + Bbolt persistence + visual effects implementations — but zero integration with Surface Layer rendering. A Surface user sees only their own social graph, not anonymous activity.

---

## Implementation Steps

### Step 1: Wire Anonymous Mechanics to Network (2 days)

**Deliverable**: Events from anonymous mechanics (gifts, marks, mini-games) are gossiped on GossipSub topics and received by all peers.

**Dependencies**: None (GossipSub, publishers, and event bus all exist)

**Goal Impact**: Enables Shadow Gradient visibility (core product differentiator)

**Acceptance**: 
- Create Phantom Gift on node A (Specter identity), verify it appears on node B (Surface user viewing recipient node) within 2 seconds
- Metrics: `murmur_anonymous_events_published_total`, `murmur_anonymous_events_received_total` > 0

**Implementation**:
1. In `pkg/app/murmur.go` `Run()` after Handlers initialization, add:
   ```go
   if err := p.Handlers.SubscribeAnonymousMechanics(ctx, p.Subsystems.PubSub); err != nil {
       return err
   }
   ```
2. In `pkg/app/handlers.go`, implement `SubscribeAnonymousMechanics()` subscribing to `/murmur/anonymous/mechanics/1.0`, `/murmur/anonymous/waves/1.0`, `/murmur/anonymous/beacons/1.0`
3. For each mechanic (gifts, marks, puzzles, hunts, territory, oracle, forge, shadowplay, councils), instantiate publisher in Subsystems and wire to event bus
4. On local mechanic events (e.g., `GiftCreated`), publisher.Publish() to GossipSub

**Validation**:
```bash
# Terminal 1: Node A (Fortress mode, Specter identity)
./murmur --mode=fortress --data-dir=/tmp/murmur-a

# Terminal 2: Node B (Open mode, Surface only)
./murmur --mode=open --data-dir=/tmp/murmur-b --bootstrap=<node-a-multiaddr>

# In Node A: Create gift via CLI or UI
# Expected: Node B receives gift event within 2s, logs "Received anonymous event: gift"
go test -v -run TestAnonymousEventPropagation ./test/integration
```

**Files Modified**:
- `pkg/app/murmur.go` (+5 lines)
- `pkg/app/handlers.go` (+80 lines: SubscribeAnonymousMechanics, handleAnonymousMechanicsMessage)
- `pkg/app/subsystems.go` (+10 lines: add publisher fields)

---

### Step 2: Cross-Layer Artifact Rendering (5 days)

**Deliverable**: Surface users see anonymous artifacts (gifts, marks, mini-games) overlaid on their Pulse Map.

**Dependencies**: Step 1 (network events must arrive)

**Goal Impact**: Completes Shadow Gradient visibility loop — "the shadows market themselves"

**Acceptance**:
- Surface user views Pulse Map, sees 5 gift effects, 3 Marks, 2 active mini-games (Cipher Puzzle + Hunt)
- Click gift effect → tooltip: "Phantom Gift from Specter. Upgrade to Hybrid to send your own."
- Hover mini-game icon → tooltip: "Cipher Puzzle: 3 participants. Resonance ≥50 to join."
- Metrics: `murmur_anonymous_artifacts_rendered_total{type="gift"}` increments per frame

**Implementation**:
1. In `pkg/pulsemap/rendering/renderer.go` `Draw()` after each node:
   ```go
   if marks := r.store.GetMarksForTarget(node.ID); len(marks) > 0 {
       r.effects.DrawSpecterMarks(screen, node.Position, marks)
   }
   if gifts := r.store.GetActiveGiftsForRecipient(node.ID); len(gifts) > 0 {
       for _, gift := range gifts {
           r.effects.DrawGiftEffect(screen, node.Position, gift.Type)
       }
   }
   // Repeat for puzzles, hunts, territory, oracle, forge, shadowplay, councils
   ```
2. In `pkg/store/typed_accessors.go`, add query methods:
   - `GetMarksForTarget(nodeID) []*Mark` — reverse index scan
   - `GetActiveGiftsForRecipient(nodeID) []*Gift` — TTL filter
   - `GetActivePuzzlesNearNode(nodeID, radius) []*Puzzle` — spatial query
   - `GetActiveHuntsWithFragmentsNear(nodeID, radius) []*Hunt`
   - `GetTerritoryInfluenceAt(nodeID) *TerritoryState`
   - `GetActiveOraclePoolsNearNode(nodeID, radius) []*OraclePool`
   - `GetActiveForgeEventsNearNode(nodeID, radius) []*Forge`
   - `GetActiveShadowPlayNearNode(nodeID, radius) []*ShadowPlay`
   - `GetMaskedEventsNearNode(nodeID, radius) []*MaskedEvent`
   - `GetCouncilsWithMember(nodeID) []*Council`
3. In `pkg/pulsemap/rendering/effects/`, implement drawing functions (leverage existing overlays):
   - `DrawSpecterMarks()` — orbiting sigils (Watcher/Ally/Rival)
   - `DrawGiftEffect()` — particle animations (Basic/Expanded/Premium)
   - `DrawPuzzleIcon()` — rotating hexagon (Fragment/Mosaic/Cascade)
   - `DrawHuntFragments()` — scattered glowing markers
   - `DrawTerritoryOverlay()` — translucent boundary watermarks
   - `DrawOracleVortex()` — swirling vortex icon
   - `DrawForgeAnvil()` — anvil-and-flame with entries
   - `DrawShadowPlayDome()` — dark dome with lightning
   - `DrawMaskedEventDome()` — translucent dome with dots
   - `DrawCouncilConstellation()` — colored thread pattern
4. Add visibility rules:
   - Surface users see effects but not content
   - Hybrid+ users see effects AND content (click → detail panel)
5. Add hover tooltips:
   - In `pkg/ui/tooltips.go`, add `AnonymousArtifactTooltip(artifactType, resonanceRequired)`
   - Render near cursor when mouse over artifact

**Validation**:
```bash
# Terminal 1: Node A (Fortress, create gift/mark/puzzle)
./murmur --mode=fortress --data-dir=/tmp/murmur-a

# Terminal 2: Node B (Open, observe Pulse Map)
./murmur --mode=open --data-dir=/tmp/murmur-b

# In Node A: Create gift, mark, puzzle
# Expected: Node B Pulse Map shows animated gift particles, orbiting mark sigil, rotating puzzle icon
go test -v -run TestCrossLayerArtifactRendering ./test/integration
```

**Files Modified**:
- `pkg/pulsemap/rendering/renderer.go` (+100 lines: artifact query + draw loops)
- `pkg/store/typed_accessors.go` (+250 lines: 10 new query methods)
- `pkg/pulsemap/rendering/effects/cross_layer.go` (+400 lines: 10 drawing functions)
- `pkg/ui/tooltips.go` (+80 lines: artifact tooltips)

---

### Step 3: Onboarding UX Screens (16 days)

**Deliverable**: First-run users see 6-phase guided onboarding: Welcome → Identity → Mode → Bootstrap → Exploration → First Wave.

**Dependencies**: None (flow controller exists, screens missing)

**Goal Impact**: Critical for retention — "first-run abandonment would be near 100% without guidance"

**Acceptance**:
- Delete `~/.murmur/`, restart app → onboarding starts (not blank Pulse Map)
- Phase 1: Philosophy screens with "Begin" button
- Phase 2: Name input, key backup prompt → identity created
- Phase 3: Mode selection (Open/Hybrid/Guarded/Fortress) → Specter created if Hybrid+
- Phase 4: Connection progress bar → 5 peers connected
- Phase 5: Pulse Map tooltips → dismissible
- Phase 6: First Wave compose prompt → PoW animation → Wave visible
- User study: >80% new users complete onboarding within 10 minutes

**Implementation**:

**3.1 First-Run Detection (1 day)**:
- In `pkg/app/murmur.go` `Run()`, check `os.Stat(dataDir + "/identity.bin")`
- If not exist, set `needsOnboarding = true`
- Initialize `OnboardingFlow`, wire `OnComplete()` callback → transition to main UI
- Skip Pulse Map init until onboarding complete

**3.2 Phase 1: Welcome Screen (2 days)**:
- In `pkg/onboarding/screens/welcome.go`:
  - Struct `WelcomeScreen` with `Draw(screen *ebiten.Image)` and `Update() bool`
  - Three sequential philosophy statements with fade-in animation (each 3s)
  - Pulsing node background animation (single glowing orb via vector graphics)
  - "Begin" button with 2-second delay (prevent accidental skip)
  - Advance on Enter or button click

**3.3 Phase 2: Identity Creation Screen (3 days)**:
- In `pkg/onboarding/screens/identity.go`:
  - Keypair generation with spinner animation (visual during Ed25519 gen)
  - Display truncated public key fingerprint (first 8 + last 8 hex chars)
  - Text input for display name with live Pulse Map preview
  - Three backup buttons: "Save file", "Show BIP-39 phrase", "Skip (warning)"
  - Wire to `pkg/identity/keys/` `GenerateKeypair()`, `EncryptAndSave()`, `GenerateMnemonic()`
  - On completion, write `identity.bin` and advance

**3.4 Phase 3: Mode Selection Screen (3 days)**:
- In `pkg/onboarding/screens/mode.go`:
  - Four cards (Open/Hybrid/Guarded/Fortress) with icons and 2-sentence descriptions
  - Animation: Surface Layer (visible) + Anonymous Layer (ghostly overlay)
  - Recommendation logic: Open for new, Hybrid for curious, Fortress for activists
  - On Hybrid+: generate Specter, separate backup prompt
  - Save mode to config, advance

**3.5 Phase 4: Bootstrap Screen (2 days)**:
- In `pkg/onboarding/screens/bootstrap.go`:
  - Expanding dots animation (dots appear, edges form)
  - Progress bar toward 5-peer target (current/5)
  - Status messages: "Connecting to bootstrap nodes...", "Discovered 2 peers via DHT...", "Establishing Shroud circuit..." (Hybrid+)
  - Troubleshooting link if >30s with 0 peers
  - Poll `app.PeerCount()` every 500ms, advance when ≥5

**3.6 Phase 5: Guided Exploration Screen (2 days)**:
- In `pkg/onboarding/screens/exploration.go`:
  - Tooltip overlays on Pulse Map: "Nodes are people. Edges are relationships."
  - Animated arrows pointing to own node, nearby nodes, edges
  - For Hybrid+: second tooltip set for Anonymous Layer
  - "Next" button to dismiss

**3.7 Phase 6: First Wave Screen (2 days)**:
- In `pkg/onboarding/screens/first_wave.go`:
  - Compose panel with pre-filled "Hello, MURMUR"
  - "Mint" button → PoW spinner animation (2–5s at difficulty 20)
  - On completion, ripple propagation animation (Wave travels along edges)
  - Success message: "Your first Wave is live!"
  - Wire to `pkg/content/waves/` `Create()`, `pkg/networking/gossip/` `Publish()`

**3.8 First-Week Nudges (1 day)**:
- In `pkg/app/murmur.go`, add background goroutine checking account age
- Day 1: notification "Try replying to a Wave"
- Day 2: "Form a connection nearby"
- Day 3 (Hybrid+): "Place a Specter Mark"
- Day 5–7: Celebrate first Resonance milestone

**Validation**:
```bash
# Delete identity, restart
rm -rf ~/.murmur
./murmur

# Expected:
# - Philosophy screens → Enter → Identity creation → "Alice" → Save backup
# - Mode selection → Hybrid → Specter backup
# - Bootstrap → 5 peers connected
# - Exploration tooltips → Next
# - First Wave → "Hello, MURMUR" → Mint → 3s PoW → Wave visible

go test -v -run TestOnboardingFlow ./test/integration
```

**Files Created**:
- `pkg/onboarding/screens/welcome.go` (200 lines)
- `pkg/onboarding/screens/identity.go` (350 lines)
- `pkg/onboarding/screens/mode.go` (300 lines)
- `pkg/onboarding/screens/bootstrap.go` (250 lines)
- `pkg/onboarding/screens/exploration.go` (200 lines)
- `pkg/onboarding/screens/first_wave.go` (250 lines)
- `pkg/onboarding/screens/nudges.go` (150 lines)

**Files Modified**:
- `pkg/app/murmur.go` (+30 lines: first-run detection, onboarding init)

---

### Step 4: Security Hardening (7 days)

**Deliverable**: Key zeroing, Wave deduplication, per-peer rate limiting, signed DHT records.

**Dependencies**: None (foundational security)

**Goal Impact**: **Must be done before v0.1 release** — prevents key scraping, spam amplification, DoS, DHT poisoning

**Acceptance**:
- Memory scan after key use finds no private key bytes
- Publishing same Wave twice → second is ignored
- Flooding peer at 100 msg/sec → only ~10/sec processed
- Fake DHT record without signature → rejected
- Zero security warnings from `go vet`, `gosec`, or security scanners

**Implementation**:

**4.1 Zero Key Material (1 day)**:
- In `pkg/identity/keys/keystore.go`, after decrypting private key:
  ```go
  defer func() {
      for i := range plaintextKey {
          plaintextKey[i] = 0
      }
  }()
  ```
- Apply to: `GenerateKeypair()`, `ExportKey()`, `ImportKey()`, `UnlockKeystore()`
- In `pkg/anonymous/shroud/circuit.go`, zero Curve25519 shared secrets after HKDF key derivation

**4.2 Wave Deduplication Filter (2 days)**:
- Add `github.com/bits-and-blooms/bloom/v3` to `go.mod`
- In `pkg/app/handlers.go`, add Bloom filter (10M capacity, 0.01 FP rate)
- In `handleWaveMessage()` before storing:
  ```go
  if h.dedupFilter.Test(envelope.MessageId) {
      return // silent drop
  }
  ```
- After validation: `h.dedupFilter.Add(envelope.MessageId)`
- Reset filter every 30 days (goroutine with `time.Ticker`)

**4.3 Per-Peer Rate Limiting (2 days)**:
- In `pkg/networking/gossip/pubsub.go`, add `rateLimiters map[peer.ID]*rate.Limiter`
- In Subscribe() handler:
  ```go
  limiter := p.getRateLimiter(msg.From)
  if !limiter.Allow() {
      logger.Warn("Rate limit exceeded", zap.String("peer", msg.From.String()))
      return // drop message
  }
  ```
- Limits: 10 msg/sec, burst 20
- Cleanup idle limiters (>5 min) every 10 minutes

**4.4 Sign DHT Records (2 days)**:
- In `pkg/networking/discovery/dht.go`, enable signed records:
  ```go
  dht.ModeOpt(dht.ModeServer),
  dht.ValidateRecords(), // Enable validation
  ```
- When publishing identity, sign record with Ed25519 key
- When reading peer records, verify signature before accepting

**Validation**:
```bash
# Test key zeroing
go test -v -run TestKeyZeroing ./pkg/identity/keys

# Test deduplication
go test -v -run TestDuplicateWaveFiltering ./pkg/app

# Test rate limiting
go test -v -run TestPeerRateLimiting ./pkg/networking/gossip

# Test signed DHT
go test -v -run TestDHTRecordValidation ./pkg/networking/discovery

# Security scan
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
# Expected: zero findings (or documented justifications)
```

**Files Modified**:
- `pkg/identity/keys/keystore.go` (+20 lines: defer zeroing in 4 functions)
- `pkg/anonymous/shroud/circuit.go` (+15 lines: zero shared secrets)
- `pkg/app/handlers.go` (+60 lines: Bloom filter init + check)
- `pkg/networking/gossip/pubsub.go` (+80 lines: rate limiter map + cleanup)
- `pkg/networking/discovery/dht.go` (+30 lines: signed records)
- `go.mod` (+1 dependency: bloom filter)

**Tests Created**:
- `pkg/identity/keys/keystore_security_test.go` (key zeroing)
- `pkg/app/handlers_dedup_test.go` (duplication filter)
- `pkg/networking/gossip/ratelimit_test.go` (rate limiter)
- `pkg/networking/discovery/dht_signed_test.go` (signed DHT)

---

### Step 5: Network Propagation Validation (9 days) — **DEFER POST-v0.1**

**Deliverable**: Simulation tests proving 3-hop propagation <500ms and 99% delivery in 3s.

**Dependencies**: None (can run in parallel with other work)

**Goal Impact**: Confidence in production performance, not blocking for early alpha

**Acceptance**:
- Simulation test: 50 nodes, publish Wave from node 0 → 99th percentile latency to node at 3 hops <500ms
- Simulation test: 99% of 50 nodes receive Wave within 3s
- Latency histogram exported to Prometheus with p50/p95/p99
- Network partition test: split 50 nodes into 2 partitions, rejoin after 30s → no message loss

**Why Defer**: Low user count in v0.1 means load is low. Network is proven functional (gossip works), just not validated at scale. Priority: visibility and UX before performance validation.

---

### Step 6: Shroud Anonymity Testing (11 days) — **DEFER POST-v0.1**

**Deliverable**: Simulation tests proving adversary controlling 30% of relays cannot de-anonymize.

**Dependencies**: None (security validation)

**Goal Impact**: Trust in Anonymous Layer claims, not blocking for friendly alpha users

**Acceptance**:
- Simulation: 100 nodes, 10 relays, adversary controls 3 → cannot identify >5% of sender-receiver pairs
- Timing analysis resistance: correlation coefficient <0.3 despite packet timing logs
- Circuit diversity: no node appears in >5% of circuits, no two circuits share >1 hop

**Why Defer**: Early adopters are not adversaries. Anonymity mechanisms are implemented (onion routing, cover traffic, random delay). Validation proves it works but doesn't change implementation. Priority: visibility and UX before paranoia-level testing.

---

### Step 7: Monitoring & Observability (6 days) — **DEFER POST-v0.1**

**Deliverable**: Prometheus metrics, health check endpoint, structured logging.

**Dependencies**: None (operational tooling)

**Goal Impact**: Critical for production operators, not needed for single-user testing

**Acceptance**:
- `/metrics` endpoint exports Prometheus format counters/gauges
- `/health` endpoint returns JSON with peer count, uptime, topic subscriptions
- Logs are structured JSON with zap (timestamp, level, msg, peer_id, subsystem)
- OpenTelemetry traces show Wave creation → PoW → publish → receive → validate → store

**Why Defer**: v0.1 is local testing, not multi-node deployment. Operators need metrics only when running bootstrap nodes (post-v0.1). Priority: visibility and UX before operational tooling.

---

## Priority Ordering & Dependencies

```
Critical Path (33 days):
  Step 1 (2d) → Step 2 (5d) → Step 3 (16d) → Step 4 (7d) → [v0.1 ready]
             ↓ (parallel)
             Step 4 (7d) can start immediately

Post-v0.1 (46 days):
  Step 5 (9d) [parallel with Step 6]
  Step 6 (11d) [parallel with Step 5]
  Step 7 (6d) [after Steps 1–6]
```

**Rationale**: Steps 1–4 address the **highest-priority gaps** from GAPS.md:
1. Shadow Gradient visibility (product thesis)
2. Onboarding UX (retention)
3. Security hardening (table stakes)

Steps 5–7 are important but not blocking for initial user testing with friendly early adopters.

---

## Risk Mitigation

**Risk 1**: Cross-layer rendering (Step 2) breaks Pulse Map performance  
**Mitigation**: Artifact queries run once per visible node per frame (O(N) where N=visible nodes, not total nodes). With viewport culling (target 500 visible), worst case is 500 × 10 queries = 5,000 store lookups/frame. At 60fps, that's 300K lookups/sec. Bbolt read throughput is ~10M reads/sec on SSD. Performance headroom: 33×.

**Risk 2**: Onboarding screens (Step 3) create UI complexity  
**Mitigation**: Each screen is self-contained `Update()/Draw()` pair. Flow controller orchestrates phase transitions. No shared state between screens. If one screen breaks, others unaffected. Progressive testing: validate each screen independently before integration.

**Risk 3**: Security hardening (Step 4) introduces regressions  
**Mitigation**: Every security change has dedicated test. Key zeroing test scans heap dump. Dedup test publishes same Wave twice. Rate limit test floods peer. DHT test injects fake record. All tests run in CI before merge.

**Risk 4**: Timeline slippage due to unforeseen complexity  
**Mitigation**: Steps 1–4 are sequential by design. If Step 1 takes 3 days instead of 2, total slips by 1 day. 20% buffer (33 days → ~40 days) is reasonable for medium complexity.

---

## Success Criteria (v0.1 Release Gate)

**Must Pass**:
1. ✅ All 8 steps from GAPS.md "Highest Priority" complete (Steps 1–4 in this plan)
2. ✅ `go test -race ./...` passes with zero failures
3. ✅ `go vet ./...` passes with zero warnings
4. ✅ `gosec ./...` passes with zero MEDIUM+ findings
5. ✅ Manual test: Open-mode user sees anonymous artifacts from Fortress-mode user
6. ✅ Manual test: New user completes onboarding in <10 minutes
7. ✅ Manual test: 3 nodes (Open, Hybrid, Fortress) exchange Waves with <2s latency

**Nice to Have** (post-v0.1):
- Simulation test: 50-node gossip propagation <500ms (Step 5)
- Simulation test: Shroud anonymity vs 30% adversary (Step 6)
- Prometheus metrics + health check (Step 7)

---

## Validation Commands

**Step 1 Validation**:
```bash
# Start two nodes with different modes
./murmur --mode=fortress --data-dir=/tmp/murmur-a --log-level=debug &
./murmur --mode=open --data-dir=/tmp/murmur-b --bootstrap=$(cat /tmp/murmur-a/multiaddr.txt) --log-level=debug &

# Create anonymous event in node A (via CLI or UI)
# Verify node B logs: "Received anonymous event: gift from [Specter]"

# Check metrics
curl http://localhost:9090/metrics | grep murmur_anonymous_events
```

**Step 2 Validation**:
```bash
# With same two nodes running, create gift/mark/puzzle in node A
# Node B UI should show:
# - Particle animation on recipient node (gift)
# - Orbiting sigil icon (mark)
# - Rotating puzzle icon near node

# Hover over artifact → tooltip with upgrade prompt
# Click artifact → detail panel with "Resonance ≥X to participate"
```

**Step 3 Validation**:
```bash
# Delete identity and restart
rm -rf ~/.murmur
./murmur

# Verify 6 phases:
# 1. Philosophy screens with "Begin" button
# 2. Name input "Alice" → keypair gen → backup prompt
# 3. Mode selection → Hybrid → Specter gen → backup
# 4. Bootstrap → progress bar → 5 peers
# 5. Exploration → tooltips → Next
# 6. First Wave → "Hello, MURMUR" → Mint → 3s PoW → visible

# All phases advance without crashes
```

**Step 4 Validation**:
```bash
# Key zeroing
go test -v -run TestKeyZeroing ./pkg/identity/keys
# Expected: heap scan finds zero key bytes after zeroing

# Deduplication
go test -v -run TestDuplicateWaveFiltering ./pkg/app
# Expected: second identical Wave is silently dropped

# Rate limiting
go test -v -run TestPeerRateLimiting ./pkg/networking/gossip
# Expected: flooding peer at 100 msg/sec → only ~10/sec processed

# Signed DHT
go test -v -run TestDHTRecordValidation ./pkg/networking/discovery
# Expected: fake DHT record without signature → validation error

# Security scan
gosec ./...
# Expected: zero MEDIUM+ findings (or documented exceptions in AUDIT.md)
```

**Integration Test**:
```bash
# Full scenario: 3 nodes, 3 modes, 5 minutes
./scripts/test-three-node-scenario.sh

# Expected:
# - Node A (Fortress): Creates Phantom Gift
# - Node B (Open): Sees gift particles on recipient node, hover → tooltip
# - Node C (Hybrid): Sees gift + places Specter Mark
# - Node B sees Mark sigil on marked node
# - All nodes exchange Waves with <2s latency
# - Zero crashes, zero panics, zero data loss
```

---

## Post-v0.1 Roadmap

After Steps 1–4 complete (33 days), **v0.1 is shippable for friendly alpha testing**. Remaining work for v0.2:

**High Priority** (v0.2 target):
- Step 5: Network propagation validation (9 days)
- Step 6: Shroud anonymity testing (11 days)
- Step 7: Monitoring & observability (6 days)
- Mobile UI optimization (5 days per GAPS.md)
- Bootstrap node deployment (3 days setup)

**Medium Priority** (v0.3 target):
- Documentation completeness (godoc + ADRs + user guide, 11 days)
- Mobile CI validation (1 day)
- Production monitoring integration (Grafana dashboards, alerting)

**Lower Priority** (v0.4+ target):
- Mobile distribution (Google Play, App Store, F-Droid)
- Performance optimization (Barnes-Hut for >500 nodes, GPU batching)
- Advanced mini-game mechanics (per ANONYMOUS_GAME_MECHANICS.md)

---

## Development Workflow

**Daily Checklist**:
1. Start with `go test -race ./...` (zero failures)
2. Implement one sub-task from current step
3. Write tests for new code (aim for 80% coverage on new functions)
4. Run `gofumpt -w -extra .` before commit
5. Run `go vet ./...` (zero warnings)
6. Update CHANGELOG.md with completed work
7. Update PLAN.md progress (check off completed items)

**Weekly Review**:
- Thursday: Review progress vs timeline, adjust estimates
- Friday: Update ROADMAP.md with completed items from PLAN.md
- Friday: Run full test suite + integration tests
- Friday: Update AUDIT.md with any security decisions made

**Before v0.1 Release**:
- [ ] All Steps 1–4 complete (33 days elapsed)
- [ ] CHANGELOG.md updated with all changes
- [ ] ROADMAP.md updated (v0.1 items checked)
- [ ] AUDIT.md reviewed for security gaps
- [ ] README.md updated with v0.1 status
- [ ] Tag release: `git tag -a v0.1.0 -m "v0.1: Shadow Gradient Visibility + Onboarding + Security"`

---

## Estimated Timeline

| Step | Description | Days | Cumulative |
|------|-------------|------|------------|
| 1 | Wire anonymous mechanics to network | 2 | 2d |
| 2 | Cross-layer artifact rendering | 5 | 7d |
| 3 | Onboarding UX screens (6 phases) | 16 | 23d |
| 4 | Security hardening (parallel with Step 3) | 7 | 23d |
| **Total (Critical Path)** | **Steps 1–4** | **23** | **23d** |
| 5 | Network propagation validation (defer) | 9 | 32d |
| 6 | Shroud anonymity testing (defer) | 11 | 43d |
| 7 | Monitoring & observability (defer) | 6 | 49d |
| **Total (Full v0.1)** | **Steps 1–7** | **49** | **49d** |

**Reality Check**: Estimate assumes 1 developer working full-time with zero context-switching. Historical velocity from CHANGELOG.md (2026-05-04 entries) suggests ~2–3 days per feature of this complexity. Timeline is realistic with 20% buffer.

**Recommendation**: Ship v0.1 after Step 4 (23 days critical path). Steps 5–7 are validations, not blockers. Early alpha testing with friendly users will surface real-world issues faster than simulation tests.

---

## Dependencies & Blockers

**External Dependencies**:
- ✅ Go 1.22+ (installed, verified)
- ✅ Ebitengine v2.9.9 (in go.mod)
- ✅ libp2p v0.48.0 (in go.mod)
- ✅ Bbolt v1.3.11 (in go.mod)
- ✅ All cryptographic libraries (Ed25519, Curve25519, ChaCha20-Poly1305, BLAKE3, Argon2id)

**Internal Blockers**:
- ❌ None identified. All subsystems exist; this plan is integration + UI.

**Risks**:
- ⚠️ Onboarding UX (Step 3) requires Ebitengine UI framework familiarity (learning curve: 2–3 days)
- ⚠️ Cross-layer rendering (Step 2) may expose Pulse Map performance issues (mitigation: viewport culling already implemented)

---

## Metrics Tracking

**Code Health** (baseline from go-stats-generator 2026-05-04):
- Functions: 5,204 → target 5,500 (+300 new functions from Steps 1–4)
- Complexity hotspots: 151 → target <160 (keep new functions simple)
- Documentation: 82.3% → target 85% (document all new exported functions)
- Duplication: 1.48% → target <2% (avoid copy-paste in onboarding screens)

**Test Suite Health** (2026-05-04):
- **Status**: ✅ 100% pass rate (38/38 packages with tests)
- **Race Detector**: ✅ Zero race conditions detected (`go test -race ./...`)
- **Duration**: ~100s total, longest 10.1s (shadowplay mechanics)
- **Complexity Baseline**: 4.9 MB JSON captured (4 functions >12 cyclomatic, all well-tested)
- **Coverage**: Comprehensive unit tests for cryptography, integration tests for networking/storage
- **Validation**: Previous failures (TestAppDoubleRun, TestAppSubsystemsInit) resolved as Cat 2 Test Spec Errors
- **Target**: Maintain 100% pass rate, add simulation tests in Steps 5–6 for gossip propagation

**Roadmap Progress** (from ROADMAP.md):
- Implemented: 145/463 (31%) → target 200/463 (43%) after Steps 1–4
- v0.1 items: 145 → target 200 (Step 3 alone adds ~30 items)

**Test Coverage** (current: no simulation tests):
- Unit tests: ✅ (all mechanics, crypto, data structures)
- Integration tests: ⚠️ (network + storage, no cross-subsystem)
- Simulation tests: ❌ (defer to Steps 5–6)
- Target after Step 4: 80% line coverage on new code in Steps 1–4

---

## References

**Specification Documents**:
- GAPS.md — Critical gap analysis (Shadow Gradient visibility)
- ROADMAP.md — Feature checklist (145/463 items complete)
- DESIGN_DOCUMENT.md — Full system specification
- TECHNICAL_IMPLEMENTATION.md — Architecture and concurrency model
- PULSE_MAP.md — Rendering pipeline and visual effects
- SHADOW_GRADIENT.md — Privacy modes and cross-layer mechanics
- SECURITY_PRIVACY.md — Threat model and cryptographic primitives
- RESONANCE_SYSTEM.md — Reputation formulas and milestone thresholds

**Planning Documents**:
- CHANGELOG.md — Implementation history (updated daily)
- AUDIT.md — Security decisions and deviations
- PLAN.md — This document (sprint-level task tracking)

**Metrics Baseline**:
- `go-stats-generator` output (2026-05-04): 5,204 functions, 151 complexity hotspots, 82.3% doc coverage, 1.48% duplication

---

**Generated by**: GitHub Copilot CLI  
**Source**: GAPS.md critical path analysis + ROADMAP.md completion status + go-stats-generator metrics  
**Next Review**: After Step 1 completion (2 days) — validate network propagation, adjust Step 2 timeline if needed
