# Implementation Gaps — 2026-05-04

This document identifies gaps between MURMUR's stated goals (from README.md, PRODUCT_VISION.md, DESIGN_DOCUMENT.md, and ROADMAP.md) and the current implementation state. Gaps are organized by user-facing impact and system criticality.

---

## Shadow Gradient Growth Mechanism

**Stated Goal**: "The Shadow Gradient is the defining design principle of MURMUR. Open-mode users see the anonymous layer's effects everywhere: encrypted shimmers drifting across their Pulse Map, ghostly sigils appearing on their friends' nodes, beautiful particle effects gifted by invisible entities, visual spectacles of anonymous mini-games, collaborative art emerging from anonymous chains, the deep glow of anonymous councils, and the arresting residue of content they cannot read. Every anonymous mechanic is designed to be visible to the clearnet and invisible in its content — creating permanent ambient curiosity."

**Current State**: 
- All 10 anonymous mechanics (Phantom Gifts, Specter Marks, Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Masked Events, Phantom Councils) are fully implemented with game logic, state machines, and Bbolt persistence.
- Visual effects for gifts (25+ types), marks (3 categories), and game artifacts exist in pkg/pulsemap/rendering/effects/ with Kage shaders (glow.kage, ripple.kage, spectra.kage).
- **BUT**: No rendering of anonymous artifacts on Surface Layer nodes. A Surface user (Open mode) looking at the Pulse Map sees only their own social graph. Specter Marks, Phantom Gifts, and mini-game visualizations are not overlaid on Surface nodes. The integration between pkg/anonymous/mechanics stores and pkg/pulsemap/rendering is missing.
- **AND**: Anonymous mechanics events are not propagated to the network. The `*_publisher.go` files in each mechanics subdirectory have `Publish()` methods, but these are never called from the main event loop. Events are created and stored locally but not gossiped on `/murmur/anonymous/mechanics/1.0` topic.

**Impact**: **CRITICAL for growth flywheel**. The core product thesis — that visibility of anonymous activity creates curiosity-driven pull toward deeper privacy tiers — is non-functional. Surface users have no reason to explore the Anonymous Layer because they cannot see it exists. Without this, MURMUR is just another P2P network with optional anonymity, not a system where "the shadows market themselves."

**Closing the Gap**:
1. **Wire anonymous mechanics to network** (2 days):
   - In pkg/app/murmur.go Run() after Handlers initialization, add subscriptions to anonymous topics:
     ```go
     if err := p.Handlers.SubscribeAnonymousMechanics(ctx, p.Subsystems.PubSub); err != nil {
         return err
     }
     ```
   - In pkg/app/handlers.go, implement `SubscribeAnonymousMechanics()` to subscribe to `/murmur/anonymous/mechanics/1.0`, `/murmur/anonymous/waves/1.0`, `/murmur/anonymous/beacons/1.0`.
   - For each mechanic (gifts, marks, puzzles, etc.), instantiate the corresponding publisher and store in Subsystems struct. Wire publisher to event bus so local mechanic events trigger network publish.
   - **Validation**: Create a Phantom Gift on node A (Specter), verify it appears on node B (Surface user viewing recipient's node) within 2 seconds.

2. **Implement cross-layer artifact rendering** (5 days):
   - In pkg/pulsemap/rendering/renderer.go Draw() loop (after drawing each node), query stores for anonymous artifacts associated with that node:
     ```go
     if marks := store.GetMarksForTarget(node.ID); len(marks) > 0 {
         effects.DrawSpecterMarks(screen, node.Position, marks)
     }
     if gifts := store.GetActiveGiftsForRecipient(node.ID); len(gifts) > 0 {
         for _, gift := range gifts {
             effects.DrawGiftEffect(screen, node.Position, gift.Type)
         }
     }
     ```
   - Implement query methods in pkg/store/typed_accessors.go: `GetMarksForTarget(nodeID)`, `GetActiveGiftsForRecipient(nodeID)`, `GetActivePuzzlesNearNode(nodeID, radius)`, etc.
   - In pkg/pulsemap/rendering/effects/, implement drawing functions for each artifact type:
     - `DrawSpecterMarks()` — orbiting sigil icons (Watcher/Ally/Rival)
     - `DrawGiftEffect()` — particle animations per gift tier (Basic/Expanded/Premium)
     - `DrawPuzzleIcon()` — rotating cryptographic symbol
     - `DrawHuntFragments()` — scattered glowing fragments
     - `DrawTerritoryOverlay()` — translucent watermarks with boundaries
     - `DrawOracleVortex()` — swirling vortex icon
     - `DrawForgeAnvil()` — anvil-and-flame with orbiting entries
     - `DrawShadowPlayDome()` — dark shimmering dome
     - `DrawMaskedEventDome()` — translucent dome with identical dots
     - `DrawCouncilConstellation()` — unique color threads between member nodes
   - Add per-artifact visibility rules: Surface users see effects but cannot read content. Hybrid+ users see effects AND content.
   - **Validation**: Surface user views Pulse Map, sees 5 different gift effects, 3 Marks, 2 active mini-games. Clicks gift effect, sees "Phantom Gift from [Specter]. Upgrade to Hybrid to send your own."

3. **Add visual curiosity hooks** (3 days):
   - When Surface user hovers over anonymous artifact, show tooltip: "Specter Mark: Watcher. [Upgrade to Hybrid to place your own marks]"
   - Add pulsing outline to nodes with anonymous activity (draw attention)
   - Add minimap overlay showing anonymous activity heatmap (colored dots for different mechanic types)
   - Add "Anonymous Layer" toggle in settings panel: dim Surface nodes, highlight Specter nodes and anonymous artifacts
   - **Validation**: User studies show >80% of Surface users notice anonymous activity within first 10 minutes.

**Timeline**: 10 days of focused implementation. This is the **highest-priority gap** because it directly impacts the product's core differentiator.

---

## Onboarding User Experience

**Stated Goal**: "Six-phase onboarding flow: (1) Welcome with philosophy animation, (2) Identity creation with keypair generation animation and key backup, (3) Mode selection with visual explanation of Shadow Gradient, (4) Network bootstrap with connection visualization, (5) Guided Pulse Map exploration with tooltips, (6) First Wave composition prompt. Post-onboarding: First-Week Nudges on Days 1–7 to encourage engagement."

**Current State**:
- Flow controller exists in pkg/onboarding/flow/ with full state machine (PhaseWelcome → PhaseIdentityCreation → PhaseMode → PhaseBootstrap → PhaseExploration → PhaseFirstAction).
- Callbacks and phase progress tracking implemented.
- Bootstrap logic in pkg/onboarding/bootstrap/ functional (connects to DHT peers).
- Tutorial framework in pkg/onboarding/tutorials/ has guided exploration primitives.
- **BUT**: pkg/onboarding/screens/ only has type definitions and stub files. No Ebitengine UI rendering code. The screens are not drawn. Users see a blank Pulse Map with no guidance on first run.
- **AND**: No first-run detection. App always skips onboarding and goes directly to main UI.

**Impact**: **HIGH for adoption**. New users are dropped into a blank force-directed graph with no explanation. They don't know what nodes are, how to create Waves, or how to explore the Anonymous Layer. First-run abandonment would be near 100% without guidance. The sophisticated onboarding design (6 phases, animations, progressive disclosure) is completely hidden.

**Closing the Gap**:
1. **Implement first-run detection** (1 day):
   - In pkg/app/murmur.go Run(), check if `~/.murmur/identity.bin` exists. If not, set `needsOnboarding = true`.
   - If `needsOnboarding`, initialize OnboardingFlow controller and skip Pulse Map initialization until onboarding completes.
   - Wire OnboardingFlow.OnComplete callback to transition to main UI.
   - **Validation**: Delete `~/.murmur/` directory, restart app, verify onboarding starts.

2. **Implement Phase 1: Welcome screen** (2 days):
   - In pkg/onboarding/screens/welcome.go, implement `WelcomeScreen` struct with:
     - `Draw(screen *ebiten.Image)` — render 3 sequential philosophy statements ("The network is the interface.", "Privacy is structural.", "Anonymity is first-class.") with fade-in animation.
     - `Update() bool` — advance on Enter key or "Begin" button click after 2-second delay.
   - Add pulsing node animation as background (single glowing orb).
   - **Validation**: First-run shows philosophy screens, advances to Phase 2 after "Begin" click.

3. **Implement Phase 2: Identity Creation screen** (3 days):
   - In pkg/onboarding/screens/identity.go, implement:
     - Keypair generation with spinning animation (visual feedback during Ed25519 generation).
     - Display public key fingerprint (truncated hex).
     - Text input for display name with live Pulse Map preview of own node.
     - Three buttons: "Save backup file", "Show recovery phrase" (BIP-39), "Skip backup (not recommended)".
   - Wire to pkg/identity/keys/ keypair generation and backup functions.
   - **Validation**: User enters name "Alice", sees node labeled "Alice" in preview, clicks backup, saves encrypted file.

4. **Implement Phase 3: Mode Selection screen** (3 days):
   - In pkg/onboarding/screens/mode.go, implement:
     - Four cards (Open, Hybrid, Guarded, Fortress) with visual icons and descriptions.
     - Animation showing Surface Layer (visible) + Anonymous Layer (ghostly overlay).
     - Recommendation logic: suggest Open for new users, Hybrid for privacy-curious, Fortress for activists.
     - On Hybrid/Guarded/Fortress selection, trigger Specter identity generation and show separate backup prompt for Anonymous Layer key.
   - **Validation**: User selects Hybrid, generates Specter, backs up two keys (Surface + Specter), mode saved to config.

5. **Implement Phase 4: Bootstrap screen** (2 days):
   - In pkg/onboarding/screens/bootstrap.go, implement:
     - Expanding dots animation as peers connect (dots appear and edges form).
     - Progress bar toward 5-peer target.
     - Status messages: "Connecting to bootstrap nodes...", "Discovered 2 peers via DHT...", "Establishing Shroud circuit..." (if Hybrid+).
     - Troubleshooting link if connection fails after 30 seconds.
   - **Validation**: New user sees peers connect animation, progress bar reaches 5/5, advances to Phase 5.

6. **Implement Phase 5: Guided Exploration screen** (2 days):
   - In pkg/onboarding/screens/exploration.go, implement:
     - Tooltip overlays on Pulse Map: "This is your network. Nodes are people. Edges are relationships."
     - Animated arrows pointing to own node, nearby nodes, edge connections.
     - For Hybrid+ users: second tooltip set explaining Anonymous Layer overlay, Specter nodes.
     - Dismissible with "Next" button.
   - **Validation**: User sees tooltips, can dismiss, Pulse Map becomes interactive after dismissal.

7. **Implement Phase 6: First Wave screen** (2 days):
   - In pkg/onboarding/screens/first_wave.go, implement:
     - Compose panel overlay with pre-filled text: "Hello, MURMUR"
     - "Mint" button triggers PoW computation with "Minting..." spinner animation.
     - On completion, show ripple propagation animation (Wave traveling along edges).
     - Success message: "Your first Wave is live!"
   - Wire to pkg/content/waves/ Wave creation and pkg/networking/gossip/ publish.
   - **Validation**: User clicks Mint, waits 3 seconds (PoW), sees Wave appear on Pulse Map and propagate.

8. **Add First-Week Nudges** (1 day):
   - In pkg/app/murmur.go, add background goroutine that checks account age.
   - On Day 1, show notification: "Try creating a reply to someone's Wave."
   - On Day 2: "Form a connection by finding someone nearby."
   - On Day 3 (if Hybrid+): "Explore the Anonymous Layer — place a Specter Mark."
   - On Day 5–7: Celebrate first Resonance milestone if achieved.
   - **Validation**: New user logs in on Day 3, sees nudge notification.

**Timeline**: 16 days. Critical for first-run retention. Should be prioritized after Shadow Gradient visibility.

---

## Network Propagation Performance

**Stated Goal**: "Wave propagation latency <500ms across 3 hops. GossipSub with peer scoring ensures fast, reliable dissemination. Per TECHNICAL_IMPLEMENTATION.md §3.3, target mesh degree is 6–12 peers, and hop count is limited to 20."

**Current State**:
- GossipSub fully implemented with flood publishing and peer scoring.
- Wave hop count tracking in pkg/content/propagation/ limits to 20 hops.
- Target mesh degree enforced in pkg/networking/mesh/ (bounds 4–12, target 6).
- **BUT**: No validation that 3-hop propagation actually happens in <500ms. No simulation tests with multi-node gossip. No latency measurement instrumentation.
- **AND**: No performance benchmarks for gossip under load (100 Waves/sec, 1000 Waves/sec).

**Impact**: **MEDIUM for reliability**. The target is stated and the mechanisms are implemented, but without measurement, there's no proof the system meets the goal. In production, slow propagation would make the UI feel unresponsive (Waves appear delayed, amplifications lag). Under spam load, latency could degrade to 5–10 seconds, breaking the "real-time" social experience.

**Closing the Gap**:
1. **Implement simulation test for 3-hop propagation** (3 days):
   - Create `test/simulation/gossip_propagation_test.go` with `//go:build simulation` tag.
   - Use in-memory libp2p transports to create 50 nodes in a mesh (each connected to 6 peers).
   - Publish a Wave from node 0, measure time until received by node at 3 hops away.
   - Assert <500ms for 99th percentile.
   - Also test 99% delivery across all 50 nodes within 3 seconds (allows multi-hop fanout).
   - Use `testing.T.Logf()` to print per-node latencies.
   - **Validation**: `go test -tags=simulation -v ./test/simulation -run TestGossipPropagation3Hops`

2. **Add latency instrumentation** (2 days):
   - In pkg/networking/gossip/handlers.go, measure time between Wave creation (timestamp in envelope) and local receipt.
   - Emit metric: `murmur_wave_propagation_latency_seconds{hops="N"}`.
   - Track as histogram with buckets [0.1, 0.25, 0.5, 1.0, 2.0, 5.0] seconds.
   - Log warning if any Wave exceeds 2 seconds (indicates network congestion or peer issues).
   - **Validation**: Publish Wave, check Prometheus /metrics, see latency histogram with p50, p95, p99.

3. **Benchmark gossip under spam load** (2 days):
   - In pkg/networking/gossip/gossip_bench_test.go, add benchmark: `BenchmarkGossipSpam`.
   - Create 10 in-memory nodes, flood with 1000 Waves/sec for 60 seconds.
   - Measure: (1) average propagation latency, (2) CPU usage, (3) memory growth.
   - Assert: latency <1s, CPU <50%, memory <100 MiB.
   - **Validation**: `go test -bench=BenchmarkGossipSpam -benchtime=60s pkg/networking/gossip`

4. **Add network partition test** (2 days):
   - In test/simulation/, create test with 50 nodes split into 2 partitions (no edges between partitions).
   - After 30 seconds, re-join partitions (add 5 bridge connections).
   - Publish Wave in partition A, verify it reaches partition B within 5 seconds of rejoin.
   - Assert: no message loss, no duplicate delivery, peer scoring heals correctly.
   - **Validation**: `go test -tags=simulation -v ./test/simulation -run TestPartitionRecovery`

**Timeline**: 9 days. Important for confidence in production deployment but not blocking for early alpha (low user count = low load).

---

## Shroud Anonymity Guarantees

**Stated Goal**: "Three-hop onion routing ensures no single node knows both origin and destination. Per SECURITY_PRIVACY.md §3, circuit construction uses hop diversity (no two hops in initiator's direct mesh), and mix network properties (random delay, cover traffic) prevent timing analysis."

**Current State**:
- Shroud circuit construction fully implemented in pkg/anonymous/shroud/circuit.go with Curve25519 key exchange and XChaCha20-Poly1305 encryption.
- Hop diversity enforced: SelectRelays() excludes initiator's direct peers.
- Random delay (exponential distribution, mean 200ms) implemented in relay forwarding.
- Cover traffic (2 dummy packets/sec) implemented when circuit is active.
- **BUT**: No test verifying that an adversary controlling 1 relay cannot link sender and receiver. No formal proof or simulation of anonymity properties.
- **AND**: No test for timing analysis resistance. If an adversary can correlate packet timing, they could de-anonymize.

**Impact**: **HIGH for trust**. The Anonymous Layer's entire value proposition is unlinkability. If Shroud circuits are vulnerable, the Shadow Gradient collapses — users won't upgrade to Hybrid/Fortress if anonymity is broken. Without validation, MURMUR cannot claim "structural privacy."

**Closing the Gap**:
1. **Implement anonymity simulation test** (5 days):
   - Create `test/simulation/shroud_anonymity_test.go` with 100 nodes, 10 acting as Shroud relays.
   - Adversary controls 3 relays (30% of relay pool).
   - Client A sends 50 Whisper Chain messages to Client B through Shroud circuits.
   - Adversary logs: (1) which messages passed through its relays, (2) message sizes, (3) timing.
   - Assert: Adversary cannot identify more than 5% of sender-receiver pairs (random guess baseline is 1/100 = 1%).
   - Use entropy analysis: if adversary can reduce uncertainty by >10 bits, anonymity is weak.
   - **Validation**: `go test -tags=simulation -v ./test/simulation -run TestShroudAnonymity`

2. **Add timing analysis resistance test** (3 days):
   - In same simulation, adversary records packet arrival times at relays it controls.
   - Use correlation attack: try to match message inter-arrival patterns at entry and exit.
   - Assert: Random delay (mean 200ms, exponential distribution) + cover traffic breaks correlation with >95% probability.
   - If test fails, increase delay or add padding to burst patterns.
   - **Validation**: Test shows correlation coefficient <0.3 (weak correlation).

3. **Add circuit diversity test** (2 days):
   - Create test with 50 nodes, client builds 100 circuits.
   - Assert: No node appears in >5% of circuits (ensures load distribution).
   - Assert: No two circuits share more than 1 hop (prevents intersection attacks).
   - **Validation**: `go test -run TestCircuitDiversity pkg/anonymous/shroud`

4. **Document threat model limitations** (1 day):
   - In SECURITY_PRIVACY.md, add section "Shroud Limitations":
     - Adversary controlling >50% of relays can perform intersection attacks.
     - Global passive adversary with network-wide traffic visibility can use traffic analysis (mitigated by cover traffic but not eliminated).
     - Shroud protects message content and relationship graph but not metadata like timing or packet count.
   - Cite simulation test results as validation of resistance against <30% adversarial relays.
   - **Validation**: Security researchers review threat model, no major objections.

**Timeline**: 11 days. Critical for credibility but not blocking for v0.1 (early users are adopters, not adversaries).

---

## Mobile Platform Support

**Stated Goal**: "Android APK and iOS xcframework distribution. Per ROADMAP.md milestone v1.0, Gomobile cross-compilation script exists. Target platforms include mobile for maximum reach."

**Current State**:
- scripts/build-mobile.sh exists and builds Android APK + iOS framework.
- Ebitengine supports mobile via `ebitengine/gomobile`.
- Touch interaction implemented in pkg/pulsemap/interaction/ (tap, pinch, swipe).
- **BUT**: No CI validation of mobile builds. Script could be broken and no one would know.
- **AND**: No mobile-specific UX (e.g., smaller font sizes, simplified UI for small screens, battery optimization).
- **AND**: No distribution mechanism (Google Play, App Store, F-Droid, TestFlight).

**Impact**: **LOW for v0.1** (desktop-first strategy is acceptable), **HIGH for growth** (mobile is where most social networking happens). Without mobile, MURMUR cannot achieve mainstream adoption. The target audience (privacy-conscious users) may prefer desktop, but network effects require mobile for "invite a friend" flows.

**Closing the Gap**:
1. **Add mobile CI build validation** (1 day):
   - Create `.github/workflows/mobile.yml` with Android build job.
   - Install Android NDK on `ubuntu-latest` runner.
   - Run `bash scripts/build-mobile.sh android`.
   - Upload APK as artifact.
   - Run on: push to main, weekly schedule (iOS requires macOS runner, defer to nightly).
   - **Validation**: Push commit, verify CI builds APK, download and install on Android device.

2. **Optimize UI for mobile** (5 days):
   - Reduce font sizes in pkg/ui/ compose panel (12pt → 10pt for mobile).
   - Simplify Pulse Map controls: remove keyboard shortcuts, add on-screen buttons for zoom/center.
   - Add battery-saving mode: reduce force-directed layout tick rate from 30Hz to 10Hz when on battery.
   - Test touch gestures on actual device (emulator is insufficient for gesture feel).
   - **Validation**: Install on Android phone, use for 30 minutes, battery drain <5%, UI readable.

3. **Prepare distribution** (3 days):
   - Create Google Play Store listing (screenshots, description, privacy policy).
   - Submit APK to Play Store as "early access" (unlisted, invite-only).
   - For iOS, enroll in Apple Developer Program ($99/year), prepare TestFlight build.
   - Add in-app update checker (query GitHub releases API for new versions).
   - **Validation**: 10 beta testers install via Play Store early access, report no install issues.

**Timeline**: 9 days. Defer to post-v0.1 unless mobile is deemed critical for early adopters.

---

## Production Monitoring and Observability

**Stated Goal**: "Prometheus metrics integration (connection count, message rates, Resonance distribution). OpenTelemetry tracing for subsystem interactions. Health check endpoint for bootstrap node operators. Per ROADMAP.md milestone v1.0, operators need visibility into node health."

**Current State**:
- No metrics exporter exists. No Prometheus endpoint.
- No tracing instrumentation. No OpenTelemetry integration.
- No health check endpoint. Bootstrap operators have no way to verify their node is reachable.
- Application logs exist but are unstructured (fmt.Printf, not structured logging with fields).

**Impact**: **MEDIUM for operators**, **HIGH for production stability**. Without metrics, operators cannot:
- Detect connection issues (are peers connecting? is DHT working?)
- Monitor load (message rate, bandwidth, CPU/memory usage)
- Diagnose performance problems (which subsystem is slow?)
- Set up alerting (Prometheus Alertmanager rules)

Without health checks, bootstrap nodes may be down and no one notices until users complain.

**Closing the Gap**:
1. **Add Prometheus metrics** (3 days):
   - Add `github.com/prometheus/client_golang/prometheus` to go.mod.
   - In pkg/app/murmur.go, create metrics registry:
     ```go
     murmur_connections = prometheus.NewGaugeVec("murmur_connections_total", []string{"type"})
     murmur_waves_received = prometheus.NewCounter("murmur_waves_received_total")
     murmur_waves_published = prometheus.NewCounter("murmur_waves_published_total")
     murmur_resonance_score = prometheus.NewGaugeVec("murmur_resonance_score", []string{"layer"})
     ```
   - Update metrics in event bus handlers (increment counters on Wave events, update gauges on connection changes).
   - Expose HTTP endpoint `/metrics` on separate port (default 9090, configurable).
   - **Validation**: `curl http://localhost:9090/metrics`, verify Prometheus format output with connection count.

2. **Add health check endpoint** (1 day):
   - On same HTTP server, add `/health` endpoint returning JSON:
     ```json
     {
       "status": "ok",
       "peer_id": "12D3Koo...",
       "connections": 8,
       "topics": ["/murmur/waves/1", "/murmur/identity/1"],
       "uptime_seconds": 3600
     }
     ```
   - Only enable if config flag `EnableMetrics` is true (default false for privacy; operators set true).
   - **Validation**: Start with `--enable-metrics`, curl `/health`, verify JSON response.

3. **Add structured logging** (2 days):
   - Replace all `fmt.Printf` with structured logger (use `go.uber.org/zap`).
   - Log format: JSON with fields (timestamp, level, msg, peer_id, subsystem).
   - Add log levels: DEBUG, INFO, WARN, ERROR. Default to INFO, configurable via `--log-level`.
   - Example: `logger.Info("Wave received", zap.String("wave_id", id), zap.Int("hop_count", hops))`
   - **Validation**: Start app, grep logs for `"subsystem":"gossip"`, verify structured JSON.

4. **Add OpenTelemetry tracing** (optional, 3 days):
   - Add `go.opentelemetry.io/otel` to go.mod.
   - Instrument critical paths: Wave creation → PoW → sign → publish → receive → validate → store.
   - Export traces to local Jaeger instance (for development) or OTLP endpoint (for production).
   - Add trace IDs to logs for correlation.
   - **Validation**: Publish Wave, view trace in Jaeger UI, see 7 spans with timing.

**Timeline**: 6 days (9 if including tracing). Important for operators but not blocking for single-user v0.1 testing.

---

## Security Hardening

**Stated Goal**: "Per SECURITY_PRIVACY.md, key material must be zeroed, PoW must be verified before signature (malleability resistance), DHT records must be signed (poisoning prevention), and rate limiting must be enforced per peer."

**Current State**:
- Cryptographic primitives are correct (Ed25519, Curve25519, ChaCha20-Poly1305).
- PoW verification exists and is called in MurmurEnvelope validation.
- DHT records are used but not signed (libp2p default is unsigned DHT records).
- Rate limiting is defined in spec but not enforced in code.
- Key material is not explicitly zeroed after use.

**Impact**: **CRITICAL for security**. These are known attack vectors:
- Unzeroed keys → memory scraping reveals private keys
- Missing deduplication → spam amplification (attacker publishes Wave once, it's processed 100x)
- No rate limiting → DoS attack (flood node with messages until it crashes)
- Unsigned DHT records → DHT poisoning (attacker injects fake peer addresses)

**Closing the Gap**:
1. **Zero key material** (1 day):
   - In pkg/identity/keys/keystore.go, after decrypting private key, zero the plaintext buffer before function return:
     ```go
     defer func() {
         for i := range plaintextKey {
             plaintextKey[i] = 0
         }
     }()
     ```
   - Apply same pattern to all functions handling Ed25519/Curve25519 private keys: GenerateKeypair(), ExportKey(), ImportKey().
   - In pkg/anonymous/shroud/circuit.go, zero Curve25519 shared secrets after deriving encryption keys.
   - **Validation**: Write test that generates key, zeros it, triggers GC, scans heap for key bytes (use debug.WriteHeapDump + grep).

2. **Add Wave deduplication filter** (2 days):
   - Add `github.com/bits-and-blooms/bloom/v3` to go.mod.
   - In pkg/app/handlers.go, add Bloom filter (10 million capacity, 0.01 false positive rate).
   - In handleWaveMessage() before storing Wave, check `if dedupFilter.Test(envelope.MessageId) { return }`.
   - After validation succeeds, add to filter: `dedupFilter.Add(envelope.MessageId)`.
   - Reset filter every 30 days (spawn goroutine with time.Ticker).
   - **Validation**: Publish same Wave twice, verify second is ignored, not stored.

3. **Enforce per-peer rate limiting** (2 days):
   - In pkg/networking/gossip/pubsub.go, add `rateLimiters map[peer.ID]*rate.Limiter` field.
   - In Subscribe() message handler, check rate limit before processing:
     ```go
     limiter := p.getRateLimiter(msg.From)
     if !limiter.Allow() {
         // Drop message
         return
     }
     ```
   - Use `golang.org/x/time/rate` with limit 10 msgs/sec, burst 20.
   - Periodically clean up limiters for disconnected peers (>5 min idle).
   - **Validation**: Write test simulating peer sending 100 msgs/sec, verify ~10/sec processed.

4. **Sign DHT records** (2 days):
   - Use libp2p's signed DHT records feature (currently experimental but available).
   - In pkg/networking/discovery/dht.go, enable signed records when creating DHT:
     ```go
     dht.ModeOpt(dht.ModeServer),
     dht.ValidateRecords(), // Enable record validation
     ```
   - Publish identity declaration to DHT with signature.
   - Verify signature when reading peer records.
   - **Validation**: Inject fake DHT record without valid signature, verify it's rejected.

**Timeline**: 7 days. **Must be done before v0.1 release** — these are security fundamentals.

---

## Documentation Completeness

**Stated Goal**: "Per ROADMAP.md milestone v1.0, API documentation for all exported types and functions is required. Operators need BOOTSTRAP_OPERATION.md and SHROUD_OPERATION.md guides."

**Current State**:
- Specification documents are comprehensive (15 markdown files, 552KB total).
- Code documentation is ~60% complete (estimated from package counts).
- Operational guides exist (docs/BOOTSTRAP_OPERATION.md, docs/SHROUD_OPERATION.md, docs/DEPLOYMENT.md).
- **BUT**: Many exported functions in pkg/store/, pkg/ui/, pkg/networking/ have no godoc comments.
- **AND**: No architecture decision records (ADRs) for key design choices.
- **AND**: No user-facing documentation (how to use MURMUR, how to invite friends, what each privacy mode means in practice).

**Impact**: **LOW for v0.1** (developers can read code), **HIGH for v1.0** (operators and contributors need docs). Good documentation accelerates onboarding of new contributors and reduces support burden.

**Closing the Gap**:
1. **Add godoc comments to all exported APIs** (5 days):
   - Start with high-traffic packages: pkg/store/, pkg/identity/keys/, pkg/content/waves/, pkg/networking/gossip/.
   - Format: `// FunctionName does X. It returns Y if Z. Example: ...`
   - Use `golangci-lint run --enable=golint` to find missing docs.
   - Target: 90% coverage (vs current 60%).
   - **Validation**: `go doc pkg/store.GetWave` shows full documentation, not just signature.

2. **Write architecture decision records** (3 days):
   - Create docs/adr/ directory with numbered markdown files.
   - Document key decisions:
     - ADR-001: Why GossipSub instead of Kademlia DHT for Wave propagation?
     - ADR-002: Why force-directed layout instead of hierarchical or circular?
     - ADR-003: Why XChaCha20-Poly1305 instead of AES-GCM for Shroud encryption?
     - ADR-004: Why BLAKE3 instead of SHA-256 for message IDs?
     - ADR-005: Why Bbolt instead of SQLite or BadgerDB?
   - Include: context, decision, consequences, alternatives considered.
   - **Validation**: New contributor reads ADRs, understands design rationale without asking questions.

3. **Write user-facing guide** (3 days):
   - Create docs/USER_GUIDE.md:
     - Getting Started: install, first run, onboarding.
     - Creating Waves: compose, PoW, TTL, reply, amplify.
     - Privacy Modes: what each mode means, how to switch, trade-offs.
     - Anonymous Layer: Specters, Phantom Gifts, mini-games overview.
     - Inviting Friends: Proximity Ignition, QR code, share invite link.
     - Troubleshooting: connection issues, bootstrap failures, slow PoW.
   - Add screenshots (use `ebiten.Screenshot()` to capture Pulse Map).
   - **Validation**: Non-technical user follows guide, successfully invites friend and sends Wave.

**Timeline**: 11 days. Important for v1.0 but can be deferred for v0.1 (internal testing only).

---

## Summary

**Total Gaps**: 8 major areas

**Highest Priority** (blocks core product thesis):
1. **Shadow Gradient visibility** (10 days) — anonymous mechanics must be visible from Surface Layer
2. **Security hardening** (7 days) — key zeroing, deduplication, rate limiting, signed DHT

**High Priority** (blocks user adoption):
3. **Onboarding UX** (16 days) — first-run experience is critical for retention

**Medium Priority** (important for production):
4. **Network propagation validation** (9 days) — prove 3-hop <500ms target
5. **Shroud anonymity testing** (11 days) — validate unlinkability claims
6. **Monitoring and observability** (6 days) — operators need metrics

**Lower Priority** (can defer to v0.2+):
7. **Mobile support** (9 days) — desktop-first is acceptable for v0.1
8. **Documentation completeness** (11 days) — sufficient for internal testing

**Total Estimated Effort**: 79 days (~16 weeks for one developer, ~8 weeks for two)

**Recommendation**: Focus on Shadow Gradient visibility (10 days) + Security hardening (7 days) + Onboarding UX (16 days) = **33 days of critical path work** to reach a minimally viable v0.1 for external testing.

---

**Generated by**: GitHub Copilot CLI functional audit
**Date**: 2026-05-04
**Source**: ROADMAP.md (145/463 items complete), README.md, PRODUCT_VISION.md, DESIGN_DOCUMENT.md, TECHNICAL_IMPLEMENTATION.md
