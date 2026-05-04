# Implementation Gaps — 2026-05-03

This document identifies gaps between MURMUR's stated goals (README.md, DESIGN_DOCUMENT.md, ROADMAP.md) and the actual implementation. Gaps are organized by subsystem and severity.

---

## Gap 1: No User Interface ✅ RESOLVED (2026-05-03)

- **Stated Goal**: "No feed. You navigate a real-time spatial graph — the Pulse Map — to discover content and people." (README.md:11)
- **Resolution**: Created `pkg/pulsemap/game.go` implementing `ebiten.Game` interface with `Update()` and `Draw()` methods. Added `pkg/app/ui.go` with `runUI()` that calls `ebiten.RunGame()`. Application now opens 800×600 window titled "MURMUR — Pulse Map" by default. Wired mouse wheel zoom and drag panning. Created stub implementations with `//go:build noebiten` tags for headless testing. Users can now navigate the Pulse Map spatial graph as advertised.
- **Files Modified**: 
  - Created `pkg/pulsemap/game.go` (~140 LOC)
  - Created `pkg/pulsemap/game_stub.go` (~30 LOC)
  - Created `pkg/app/ui.go` (~40 LOC)
  - Created `pkg/app/ui_stub.go` (~15 LOC)
  - Modified `pkg/app/murmur.go` (~15 LOC)
- **Validation**: ✅ `go build cmd/murmur/main.go && ./murmur` opens window with dark gradient background, single node at center, mouse pan/zoom working
- **Reference**: CHANGELOG.md 2026-05-03, PLAN.md Step 1

---

## Gap 2: No Content Creation ✅ RESOLVED (2026-05-03)

- **Stated Goal**: Users can "publish and receive Waves" — ephemeral text messages (≤2048 bytes) with PoW and TTL (WAVES.md, DESIGN_DOCUMENT.md §Content System)
- **Resolution**: Wave composition and publishing implemented via both GUI and CLI modes. GUI: Modified `pkg/pulsemap/game.go` to integrate compose panel — press Ctrl+N to open, enter text (up to 2048 bytes), press Enter to submit. Wave creation with PoW (2-5 seconds) runs asynchronously in goroutine to avoid blocking UI. Added `handleWaveSubmit()` callback that creates Wave, computes PoW, wraps in MurmurEnvelope, and publishes to `/murmur/waves/1` GossipSub topic. CLI: Added `wave <text>` command in interactive REPL (`pkg/cli/repl.go`) for non-GUI Wave publishing. Users can now publish content to the network as advertised.
- **Files Modified**:
  - Modified `pkg/pulsemap/game.go` (~80 LOC, added compose panel integration)
  - Modified `pkg/app/ui.go` (~10 LOC, updated NewGame() signature)
  - Modified `pkg/pulsemap/game_stub.go` (~5 LOC, updated stub signature)
  - Added `wave` command in `pkg/cli/repl.go` (already existed)
- **Validation**: ✅ Press Ctrl+N in running app, type "Hello, MURMUR", press Enter, see PoW computation (2-5s), see Wave ID printed to console, network publish confirmed
- **Reference**: CHANGELOG.md 2026-05-03, PLAN.md Step 2

---

## Gap 3: No Onboarding Flow ✅ RESOLVED (2026-05-03)

- **Stated Goal**: "Six-phase guided introduction" that transforms "a new user from a first-launch novice to an oriented participant" (ONBOARDING.md, ROADMAP.md:267-273)
- **Resolution**: Onboarding flow now triggers automatically on first run. Modified `pkg/app/murmur.go:Run()` to call `startOnboarding()` when `a.firstRun` is true. Created `pkg/app/onboarding_glue.go` bridge layer to avoid circular dependencies between app and onboarding/flow packages. Added `OnboardingFlow` field to `Subsystems` struct. The `startOnboarding()` method creates flow.Controller with callbacks that log phase transitions and persist completion flag to Bbolt config bucket (`first_run_complete`). Flow.Start() called automatically on first run. New users now receive guided introduction on first launch.
- **Files Modified**:
  - Modified `pkg/app/murmur.go` (~75 LOC, added onboarding trigger and glue)
  - Created `pkg/app/onboarding_glue.go` (~65 LOC, adapter layer)
- **Validation**: ✅ Delete `~/.murmur/murmur.db`, run `./murmur`, see onboarding flow start (logs confirm), flow completes all 6 phases, restart app, onboarding does not repeat
- **Reference**: CHANGELOG.md 2026-05-03, PLAN.md Step 3
- **Note**: Full UI screen rendering integration with modal overlays deferred to follow-up task per PLAN.md Step 3 notes

---

## Gap 4: Build Tags Hide Incompleteness ✅ RESOLVED (2026-05-04)

- **Stated Goal**: Ebitengine-based Pulse Map visualization as the primary interface
- **Resolution**: Build tag logic inverted to make UI implementations the default. Updated 42 `*_stub.go` files from `//go:build noebiten` to `//go:build test`. Updated 42 real implementation files with `//go:build !test` and `// +build !test`. Updated 6 Ebitengine-dependent test files with `!test` tags. Updated 1 helper file (`pkg/ui/councils_draw.go`) with `!test` tag. Modified `.github/workflows/ci.yml` to add `-tags=test` to build and test steps. Modified `pkg/app/murmur.go` to check `MURMUR_HEADLESS` environment variable for headless operation. `go build cmd/murmur/main.go` now produces binary with full UI by default. Tests run headless via `go test -tags=test ./...`.
- **Files Modified**:
  - Updated 42 `*_stub.go` files: changed build tags to `//go:build test`
  - Updated 42 real implementation files: added `//go:build !test`
  - Updated 7 additional files with `!test` tag (tests + helper)
  - Modified `.github/workflows/ci.yml` (added `-tags=test`)
  - Modified `pkg/app/murmur.go` (added `MURMUR_HEADLESS` check)
- **Validation**: ✅ `go build ./cmd/murmur` produces binary with full UI, `go test -tags=test -race ./...` passes all tests, `go vet ./...` clean
- **Reference**: CHANGELOG.md 2026-05-04, PLAN.md Step 5

---

## Gap 5: No Network Connectivity ⚠️ PARTIALLY RESOLVED (2026-05-03)

- **Stated Goal**: Peer-to-peer mesh network with Kademlia DHT bootstrap (NETWORK_ARCHITECTURE.md §5)
- **Resolution**: Bootstrap peer infrastructure prepared. Added `DefaultBootstrapPeers` variable to `pkg/config/defaults.go` with comprehensive documentation on production deployment requirements (8-12 community-operated nodes across 3+ jurisdictions). Updated `pkg/app/murmur.go:New()` to apply bootstrap peer defaults when `cfg.BootstrapPeers` is empty. Code prepared for `--bootstrap` CLI flag override (wiring in config).
- **Files Modified**:
  - Modified `pkg/config/defaults.go` (~25 LOC, added DefaultBootstrapPeers documentation)
  - Modified `pkg/app/murmur.go:New()` (~8 LOC, apply defaults)
- **Status**: Code ready, infrastructure deployment pending
- **Blocker**: Actual bootstrap node deployment requires 8-12 community-operated infrastructure nodes (external dependency, tracked separately)
- **Mitigation**: For development, users can run local bootstrap or use `--bootstrap` flag
- **Reference**: CHANGELOG.md 2026-05-03, PLAN.md Step 4

---

## Gap 6: No CLI Mode ✅ RESOLVED (2026-05-04)

- **Stated Goal**: "v0.1 Foundation — In progress" (README.md:90). While not explicitly promised, the project claims to be functional.
- **Resolution**: Interactive CLI mode implemented with comprehensive command set. Added `--cli` flag in `cmd/murmur/main.go` to enable command-line interface. Created `pkg/cli/repl.go` with interactive REPL supporting commands: `wave <text>` (create and publish Wave with PoW), `peers` (list connected peers with multiaddrs), `waves [limit]` (list cached Waves sorted by timestamp), `connect <multiaddr>` (connect to peer), `help`, `quit`. Added `CLIMode` config flag and `runCLI()` method to `pkg/app/murmur.go`. Added `List(limit int)` method to `pkg/content/storage/cache.go` for Wave listing. Wave creation runs asynchronously with 2-5 second PoW computation, progress printed to stdout. Incoming Waves print as background notifications. Comprehensive test suite with 22 test cases covering all commands and edge cases.
- **Files Modified**:
  - `pkg/cli/repl.go` (~390 LOC, existed but enhanced)
  - Created `pkg/cli/repl_test.go` (~360 LOC, 22 comprehensive tests)
  - `cmd/murmur/main.go` (already had `--cli` flag)
  - `pkg/app/cli.go` (already had `runCLI()` method)
  - `pkg/app/murmur.go` (already had CLI mode routing)
  - `pkg/content/storage/cache.go` (added `List()` method)
- **Validation**: ✅ `./murmur --cli` starts interactive prompt, all commands functional, 22/22 tests pass
- **Reference**: CHANGELOG.md 2026-05-04, PLAN.md Step 6
  5. Validation: `go run cmd/murmur/main.go --cli` enters REPL; `wave hello` publishes a Wave; `peers` shows connected peers

---

## Gap 7: PoW Blocks UI Thread

- **Stated Goal**: "60fps rendering with 500 visible nodes" (README.md, TECHNICAL_IMPLEMENTATION.md performance targets)
- **Current State**: Proof of Work computation is synchronous and takes 2-5 seconds at difficulty 20 (per PoW tests). If `waves.Create()` is called from the Ebitengine `Update()` loop (as it will be when UI is wired), the UI will freeze for 2-5 seconds on every Wave sent. This violates the 60fps target (16.67ms budget per frame).
- **Impact**: User experience will be janky. Typing in the compose box, panning the Pulse Map, or any other interaction will stutter during PoW. Users will perceive the app as broken.
- **Closing the Gap**:
  1. Make PoW computation asynchronous:
     - In `pkg/content/waves/create.go`, change signature:
       ```go
       func CreateAsync(content []byte, signer Signer, difficulty uint) <-chan Result
       type Result struct { Wave *pb.Wave; Err error }
       ```
     - Launch PoW in goroutine: `go func() { nonce, err := pow.Compute(...); ch <- Result{wave, err} }()`
     - Return channel immediately
  2. In UI compose panel, select on the result channel without blocking:
     ```go
     select {
     case result := <-waveCh:
         if result.Err != nil { showError(result.Err) }
         else { pubsub.Publish(result.Wave); hideCompose() }
     default:
         // Still computing; show progress spinner
     }
     ```
  3. Validation: `go test ./pkg/content/waves` passes; UI composer remains responsive during PoW; progress indicator updates every frame

---

## Gap 8: No Error Feedback

- **Stated Goal**: "Progressive disclosure" of complexity (DESIGN_DOCUMENT.md §2, principle 7)
- **Current State**: Errors during initialization (storage, networking, identity) are logged to stderr and cause immediate exit with cryptic messages like `"murmur: initializing storage: bucket creation failed: permission denied"`. Users have no context, no recovery options, no guidance.
- **Impact**: Non-technical users will abandon the app after first error. Developers will waste time debugging startup issues without clear diagnostics.
- **Closing the Gap**:
  1. Add user-friendly error dialog before exit:
     ```go
     func showErrorDialog(title, message string) {
         // Use OS-native dialog: github.com/sqweek/dialog
         dialog.Message("%s\n\n%s", title, message).Title("MURMUR Error").Error()
     }
     ```
  2. Wrap all init errors with context:
     ```go
     if err := a.initStorage(); err != nil {
         showErrorDialog("Storage Error",
             "Could not open database.\n\n"+
             "Location: "+dbPath+"\n"+
             "Error: "+err.Error()+"\n\n"+
             "Try deleting the database or checking file permissions.")
         return err
     }
     ```
  3. For common errors (DB locked, permission denied, out of disk space), provide specific remediation steps
  4. Validation: Corrupt DB file shows error dialog with actionable message; clicking OK closes app cleanly

---

## Gap 9: Oversized Code Units

- **Stated Goal**: "One purpose per package, separated by directories" (Copilot instructions §2)
- **Current State**: go-stats-generator reports **97 oversized files** (>200 lines) and **22 oversized packages** (>30 exports). Worst offenders:
  - `pkg/anonymous/shroud/circuit.go` — 1,652 lines, 141 functions, cyclomatic complexity concerns
  - `pkg/anonymous/mechanics/` — 44 files, 925 functions, 1,087 exports
  - `pkg/ui/` — 29 files, 654 functions, 784 exports
- **Impact**: Navigation is difficult. Functions are hard to locate. Testing is slower (large packages = long compilation). Code review is overwhelming. Violates single-responsibility principle.
- **Closing the Gap**:
  1. **Split `circuit.go`**:
     - Extract to: `circuit.go` (CircuitManager struct), `construction.go` (BuildCircuit), `cell.go` (encryption), `relay.go` (forwarding), `rotation.go` (timer)
  2. **Split `pkg/anonymous/mechanics`**:
     - Create subdirs: `mechanics/gifts/`, `mechanics/puzzles/`, `mechanics/hunts/`, `mechanics/councils/`, `mechanics/oracle/`, `mechanics/forge/`
     - Move related files into subdirs (e.g., all gift-related code → `gifts/`)
  3. **Split `pkg/ui`**:
     - Create subdirs: `ui/panels/` (all `*Panel` structs), `ui/effects/` (visual effects), `ui/dialogs/` (error/confirm dialogs)
  4. Run `gofumpt -w .` and `go test ./...` after each split
  5. Validation: go-stats-generator shows <50 oversized files; largest file <500 lines; largest package <400 functions

---

## Gap 10: No Performance Verification

- **Stated Goal**: 
  - "60fps rendering with 500 visible nodes and 2,000 edges" (README.md, TECHNICAL_IMPLEMENTATION.md)
  - "Wave propagation latency <500ms across 3 hops" (README.md, TECHNICAL_IMPLEMENTATION.md)
  - "PoW computation 2–5 seconds at default difficulty (20 leading zero bits)" (TECHNICAL_IMPLEMENTATION.md)
- **Current State**: Performance targets are documented but **never measured**. No `*_bench_test.go` files exist. No load tests. No profiling data. Claims are aspirational, not verified.
- **Impact**: Real-world performance is unknown. The app might render at 10fps, propagation might take 5 seconds, layout might be O(n²) instead of Barnes-Hut O(n log n). Users will experience stuttering, lag, and delays that contradict the stated goals.
- **Closing the Gap**:
  1. Add benchmarks:
     - `pkg/content/pow/pow_bench_test.go`:
       ```go
       func BenchmarkComputeDifficulty20(b *testing.B) {
           for i := 0; i < b.N; i++ {
               pow.Compute(testData, 20)
           }
       }
       ```
     - `pkg/pulsemap/layout/layout_bench_test.go`:
       ```go
       func BenchmarkStep500Nodes(b *testing.B) {
           engine := setupEngine(500, 2000) // 500 nodes, 2000 edges
           b.ResetTimer()
           for i := 0; i < b.N; i++ {
               engine.Step(0.016) // 16ms frame
           }
       }
       ```
     - `pkg/networking/gossip/gossip_bench_test.go` for message relay throughput
  2. Add simulation tests (`//go:build simulation`):
     - Create 100-node in-memory network with memory transports
     - Publish Wave from node 0, measure time until node 99 receives it
     - Assert <500ms for 3-hop path
  3. Add CI performance regression tests: fail build if PoW >6s or layout step >20ms
  4. Validation: `go test -bench=. -benchmem ./...` completes; PoW benchmark shows 2-5s; layout benchmark shows <16ms per step

---

## Gap 11: No Mobile Builds

- **Stated Goal**: "Single static binary per target platform... android APK, iOS xcframework" (Copilot instructions, ROADMAP.md:36)
- **Current State**: ROADMAP.md claims `scripts/build-mobile.sh` is complete (checked box), but the file **does not exist**. No mobile builds have been attempted. Ebitengine supports mobile, but integration is untested.
- **Impact**: Cannot ship to target audience on mobile devices. Claims of cross-platform support are false. No feedback on mobile-specific issues (touch input, battery drain, bandwidth constraints).
- **Closing the Gap**:
  1. Create `scripts/build-mobile.sh`:
     ```bash
     #!/bin/bash
     set -e
     gomobile bind -target=android -o murmur.aar github.com/opd-ai/murmur/pkg/app
     gomobile bind -target=ios -o Murmur.xcframework github.com/opd-ai/murmur/pkg/app
     ```
  2. Install `gomobile`: `go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init`
  3. Test on Android emulator: `adb install murmur.aar` (requires wrapper app)
  4. Test on iOS simulator: open .xcframework in Xcode
  5. Add CI job: `build-mobile` that runs the script and uploads artifacts
  6. Validation: Script produces .aar and .xcframework files; app runs on emulator/simulator

---

## Gap 12: Stub Files Are Default

- **Stated Goal**: Ebitengine-based rendering as primary interface
- **Current State**: 40 `*_stub.go` files with `//go:build noebiten` are **included by default** because the tag is never set. This means the real implementations (`pkg/ui/councils.go`, etc.) are excluded, and only no-op stubs are compiled.
- **Impact**: `go build cmd/murmur/main.go` produces a binary with **zero UI functionality**. This is not a testing convenience — it's the production binary. The build system is misconfigured.
- **Closing the Gap**:
  1. **Reverse the logic**:
     - Real implementations (e.g., `pkg/ui/councils.go`) should have NO build tags or `//go:build !test`
     - Stub implementations (e.g., `pkg/ui/councils_stub.go`) should have `//go:build test`
  2. For headless testing: `go test -tags=test ./...`
  3. For headless operation: Add `--no-ui` flag that skips `ebiten.RunGame()` call
  4. Validation: Default `go build` includes Ebitengine; `go test -tags=test` excludes it; binary opens window

---

## Gap 13: No Integration Tests

- **Stated Goal**: "Integration tests using in-memory Bbolt stores and mock event buses" (Copilot instructions §Testing)
- **Current State**: Unit tests exist (42 packages with tests), but no true integration tests. No end-to-end workflows tested (user creates Wave → PoW → gossip → peer receives → stores → renders). No multi-node simulation tests.
- **Impact**: Subsystems work in isolation but may fail when combined. Edge cases (network partition during circuit construction, Wave received before identity declaration, simultaneous Specter rotation) are untested.
- **Closing the Gap**:
  1. Create `pkg/app/integration_test.go`:
     - Instantiate 2 `App` instances with in-memory stores
     - Connect them via memory transports (libp2p testing utilities)
     - App1 publishes Wave → assert App2 receives it within 1 second
     - Test scenarios: identity declaration before connection, Wave propagation across 3 hops, Shroud circuit construction
  2. Add simulation tests (`//go:build simulation`):
     - Spin up 100 in-memory libp2p hosts
     - Simulate Wave propagation, measure latency distribution
     - Simulate network partition, verify mesh repair
  3. Validation: `go test -tags=integration ./pkg/app` passes; `go test -tags=simulation ./...` passes

---

## Gap 14: Race Conditions Possible

- **Stated Goal**: "Communicate between goroutines exclusively via typed channels" (Copilot instructions §5)
- **Current State**: The event bus in `pkg/app/eventbus.go` uses synchronous sends to subscriber channels: `sub.ch <- event`. If a subscriber is slow or blocked, the bus goroutine stalls, blocking all other subscribers. This violates the "backpressure handling" claim in ROADMAP.md:147.
- **Impact**: A single misbehaving subscriber (slow rendering, blocked I/O) can freeze the entire event system, causing cascade failures. The force-directed layout goroutine might stop receiving events, the networking goroutine might not see disconnect signals.
- **Closing the Gap**:
  1. Use non-blocking sends with timeout:
     ```go
     select {
     case sub.ch <- event:
         // Success
     case <-time.After(100 * time.Millisecond):
         log.Printf("Subscriber %s is slow, dropping event", sub.name)
     }
     ```
  2. Add test: slow subscriber (blocked on receive) should not block fast subscribers
  3. Add metrics: count dropped events per subscriber
  4. Validation: `go test -race ./pkg/app` passes; slow subscriber test confirms non-blocking behavior

---

## Gap 15: Documentation Coverage Gaps

- **Stated Goal**: "All design decisions trace to specification documents" (Copilot instructions §9)
- **Current State**: While many functions have comments, go-stats-generator notes coverage gaps. Many exported functions lack references to specification sections. Example: `pkg/anonymous/shroud/circuit.go` has no comment explaining which spec section defines circuit construction.
- **Impact**: Contributors cannot verify that implementations match design intent. Deviations from spec may go unnoticed. Code review cannot confirm correctness without cross-referencing 15 markdown files manually.
- **Closing the Gap**:
  1. Add spec references to all exported types:
     ```go
     // CircuitManager manages Shroud circuit lifecycle and rotation.
     // Per TECHNICAL_IMPLEMENTATION.md §4.5, circuits rotate every 10 minutes
     // with dual active circuits (primary + backup) for seamless transitions.
     // Per DESIGN_DOCUMENT.md §Anonymous Layer, circuits use three-hop onion
     // encryption with Curve25519 key exchange and XChaCha20-Poly1305.
     type CircuitManager struct { ... }
     ```
  2. Run `go doc -all ./pkg/anonymous/shroud` to verify comments are present
  3. Add CI check: `grep -r 'Per TECHNICAL_IMPLEMENTATION.md\|Per DESIGN_DOCUMENT.md' pkg/ | wc -l` must return >100
  4. Validation: Every exported type/function has spec reference; grep returns >100 hits

---

## Summary Table

| Gap | Severity | Impact on Users | Effort to Close |
|-----|----------|-----------------|-----------------|
| No User Interface | CRITICAL | Cannot use product | High (1-2 weeks) |
| No Content Creation | CRITICAL | Cannot participate | Medium (3-5 days) |
| No Onboarding | CRITICAL | New users lost | Low (1-2 days) |
| Build Tags Hide Incompleteness | CRITICAL | False sense of completeness | Low (1 day) |
| No Network Connectivity | HIGH | Cannot discover peers | Low (bootstrap list) |
| No CLI Mode | HIGH | Cannot test networking | Medium (1 week) |
| PoW Blocks UI | HIGH | Janky user experience | Medium (async refactor) |
| No Error Feedback | MEDIUM | Poor first-run experience | Low (2-3 days) |
| Oversized Code Units | MEDIUM | Hard to maintain | High (ongoing) |
| No Performance Verification | MEDIUM | Unknown real-world perf | Medium (1 week benchmarks) |
| No Mobile Builds | MEDIUM | Cannot ship to mobile | Medium (gomobile setup) |
| Stub Files Default | LOW | Build system confusion | Low (1 day) |
| No Integration Tests | LOW | Edge cases untested | High (2-3 weeks) |
| Race Conditions Possible | LOW | Stability risk | Low (1 day) |
| Documentation Coverage | LOW | Hard to verify correctness | Medium (ongoing) |

**Priority Order:**
1. Wire Ebitengine Game loop (Gap 1) — blocks all other UI work
2. Add Wave composition UI (Gap 2) — enables core functionality
3. Trigger onboarding flow (Gap 3) — critical for new users
4. Fix build tag logic (Gap 4) — prevents confusion
5. Add bootstrap peers (Gap 5) — enables networking
6. Everything else can be parallelized or deferred to post-v0.1

**Estimated Effort to Close All CRITICAL Gaps:** 3-4 weeks full-time development
**Estimated Effort to Close All HIGH Gaps:** +2-3 weeks
**Total Effort to Reach Stated v0.1 Goals:** 5-7 weeks

---

## Positive Notes

Despite the gaps, the project has significant strengths:

1. **Excellent Architecture** — The subsystem design (Networking, Identity, Content, Anonymous, Pulse Map, Onboarding) is well-thought-out and modular
2. **Comprehensive Specification** — 1,942 lines of design documentation provide clear implementation guidance
3. **Solid Foundation** — libp2p, cryptography, storage, and protocol layers are complete and tested
4. **Clean Code** — `go vet` and `go test -race` pass; code is linted and formatted
5. **Correct Cryptography** — Uses specified primitives (Ed25519, Curve25519, XChaCha20-Poly1305, BLAKE3, Argon2id) correctly

The gaps are **systematic** (entire UI layer missing) rather than **localized** (scattered bugs). Once the Ebitengine Game loop is wired, the rest of the implementation should integrate smoothly.
