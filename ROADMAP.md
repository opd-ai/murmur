# Goal-Achievement Assessment

## Project Context
- **What it claims to do**: A decentralized, peer-to-peer social network with dual-layer identity, no engagement algorithms, no permanent record, and a Pulse Map-first interface. Core promises include encrypted connectivity, onion-routed anonymous traffic, self-sovereign Ed25519 identity, ephemeral Waves (default 7d TTL, max 30d), first-class anonymity (Specters/Shroud/Resonance), and no likes/follower metrics (README).
- **Target audience**: Privacy-conscious users, self-sovereign identity advocates, and communities preferring anonymous social mechanics and non-algorithmic interaction models (README + docs positioning).
- **Architecture**: 78+ Go packages currently organized around app/runtime, networking, identity, content, anonymous mechanics, Pulse Map, onboarding, storage, transport adapters, and CLI/bootstrap/TUI binaries (`go list ./...`).
- **Terminal UI track**: `cmd/murmur-tui` and `pkg/tui/*` now provide a Bubble Tea-based parity track with ongoing matrix-driven implementation.
- **Terminal UI track**: feature matrix now reports all P0/P1 parity rows as implemented (`done`) for the Bubble Tea path; remaining items are P2 visual fidelity and optional enhancements.
- **Terminal UI track**: feature matrix now reports all P0/P1/P2 rows as implemented (`done`) with terminal-native equivalents for visual-only effects.
- **Module/dependency footprint**: module `github.com/opd-ai/murmur`, `go 1.25.7`, 29 direct + 128 indirect dependencies (`go.mod`).
- **Existing CI/quality gates**:
  - Cross-OS build/test/race/vet/lint/license checks in [.github/workflows/ci.yml](.github/workflows/ci.yml)
  - Complexity regression gates via go-stats-generator in [.github/workflows/ci.yml](.github/workflows/ci.yml)
  - Coverage job with critical-package thresholds in [.github/workflows/ci.yml](.github/workflows/ci.yml)
  - Cross-platform artifact builds in [.github/workflows/build.yml](.github/workflows/build.yml)
  - Release creation step in [.github/workflows/build.yml](.github/workflows/build.yml) now uses `ncipollo/release-action@v1`
  - WASM Pages deployment in [.github/workflows/pages-wasm.yml](.github/workflows/pages-wasm.yml)
  - Mobile build pipeline in [.github/workflows/mobile.yml](.github/workflows/mobile.yml), now targeting dedicated `cmd/murmur-mobile` gomobile entrypoint

## External Research (Brief)
- **Repository signal**: Public GitHub has 0 open issues and 0 open PRs at review time; no active public backlog/discussion signal from issues/PR flow.
- **Dependency posture**:
  - `github.com/libp2p/go-libp2p v0.48.0` is current major line and includes recent transport/interop fixes; known recent breaking surfaces are mostly around WebTransport/API evolution.
  - `github.com/hajimehoshi/ebiten/v2 v2.9.9` is current in the 2.9 series.
  - `go.etcd.io/bbolt` pinned to `v1.4.0` while upstream has newer `v1.4.3`.
  - `github.com/pion/webrtc/v4 v4.2.12` is current and includes post-security-fix dependency updates; earlier 4.2.x had a DTLS security advisory path.
- **Comparable landscape calibration**:
  - Mastodon: federated server model, community moderation, no algorithmic default feed.
  - Nostr: signed-event protocol with relay-based distribution, still rapidly evolving.
  - Scuttlebutt: decentralized social model with local-first/offline-friendly roots.
  - MURMUR ambition is higher than typical microblog alternatives due to combined spatial UI + dual-identity + built-in anonymous mechanics.

## Metrics & Baseline Health
- **go-stats-generator (`--skip-tests`)**:
  - LOC: 55,462
  - Files: 367
  - Packages (analyzed): 74
  - Functions: 1,615
  - Methods: 5,113
  - Average function complexity: 3.27
  - Duplication: 523 duplicated lines, 36 clone pairs, ratio 0.00454 (~0.45%)
  - Documentation coverage overall: 81.56%
- **Risk thresholds from prompt**:
  - Functions with cyclomatic complexity >15: 3
    - [pkg/identity/invitation.go](pkg/identity/invitation.go#L298) `unmarshalV2` (CC 20)
    - [pkg/network/adapter_wasm.go](pkg/network/adapter_wasm.go#L196) `createLoopbackPeerPair` (CC 26)
    - [pkg/pulsemap/game.go](pkg/pulsemap/game.go#L735) `handleTouchInput` (CC 16)
  - Functions with code length >50: 12 (examples include [pkg/network/adapter_wasm.go](pkg/network/adapter_wasm.go#L196), [pkg/pulsemap/game.go](pkg/pulsemap/game.go#L735), [pkg/app/ui.go](pkg/app/ui.go#L76)).
- **Executed checks**:
  - `go test -race ./...`: pass, no race reports
  - `go vet ./...`: clean
  - `pkg/tunneling/relay` hardening validation: plaintext `UNREGISTER` now rejected with `401 Unauthorized`; teardown accepted only via framed operator path (`TestPlaintextUnregisterIsRejected`).

## Goal-Achievement Summary
| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| Decentralized P2P network (no central server dependency) | ⚠️ Partial | Networking stack and discovery/gossip are implemented across `pkg/networking/*`; race/vet clean. | Bootstrap defaults and discovery still contain placeholders and invalid production seed entries: [pkg/config/defaults.go](pkg/config/defaults.go#L16), [pkg/networking/discovery/bootstrap.go](pkg/networking/discovery/bootstrap.go#L18), [pkg/networking/discovery/bootstrap.go](pkg/networking/discovery/bootstrap.go#L47). This weakens out-of-box network join reliability. |
| Network is the interface (Pulse Map primary UX) | ✅ Achieved | Pulse Map engine/rendering/interaction packages are present and wired (`pkg/pulsemap/*`), including force layout and rendering pipeline. | Residual polish gaps exist but primary interface behavior is implemented. |
| Privacy is structural (encrypted transport, onion-routed anonymous traffic) | ✅ Achieved | Ed25519 + X25519 + XChaCha20-Poly1305 implemented in [pkg/identity/keys/keypair.go](pkg/identity/keys/keypair.go#L1); 3-hop Shroud circuit primitives in [pkg/anonymous/shroud/circuit.go](pkg/anonymous/shroud/circuit.go#L1). | No critical implementation hole found in core cryptographic path during this review. |
| Identity is self-sovereign (keypair-based, no third-party registration) | ✅ Achieved | Identity generation/signing/keystore path implemented in [pkg/identity/keys/keypair.go](pkg/identity/keys/keypair.go#L44) and [pkg/identity/keys/keystore.go](pkg/identity/keys/keystore.go#L31). | One migration path is still unimplemented for legacy keystores: [pkg/identity/keys/keystore.go](pkg/identity/keys/keystore.go#L317). |
| Content is ephemeral (Waves with default 7d TTL, max 30d) | ✅ Achieved | TTL constants and validation in [pkg/content/waves/types.go](pkg/content/waves/types.go#L35), [pkg/content/waves/types.go](pkg/content/waves/types.go#L38), [pkg/content/waves/types.go](pkg/content/waves/types.go#L95). | No blocking gap found in declared TTL constraints. |
| Anonymity is first-class (Specters, Shroud, Resonance, mechanics) | ✅ Achieved | Anonymous subsystem breadth exists (`pkg/anonymous/*`); cross-layer spatial queries implemented in `pkg/store/typed_accessors.go` with `NodePositionFunc`; `ActionJoinGame` handler live in `pkg/pulsemap/game.go`. | UI-level integration test for network-backed Join Game flow remains. |
| No likes/follower-count mechanics | ✅ Achieved | No concrete like/follower feature path found in production packages during scan; interaction model centers Waves and mechanics. | Continued governance needed to avoid accidental metric creep. |
| Six-subsystem architecture claim (networking/identity/content/anonymous/pulsemap/onboarding) | ✅ Achieved | Package topology clearly maps to all six subsystems from `go list ./...`; flow controller confirms six onboarding phases in [pkg/onboarding/flow/controller.go](pkg/onboarding/flow/controller.go#L14). | None material. |
| “Core infrastructure fully operational” status claim | ⚠️ Partial | Baseline health checks pass (`go test -race`, `go vet`) and CI matrix exists. | Operational completeness is reduced by unresolved placeholders in bootstrap/TURN/discovery and a few explicit “not yet implemented” paths: [pkg/networking/relay/turn.go](pkg/networking/relay/turn.go#L178), [pkg/identity/share.go](pkg/identity/share.go#L163). |
| Performance/scalability quality bar (including Pulse Map at scale) | ⚠️ Partial | Performance tests/benchmarks are present under `pkg/pulsemap/layout/*_test.go` and propagation tests under `pkg/content/propagation/*_test.go`. | Current review did not execute simulation/performance-tag suites; claims remain test-backed in-repo but not independently re-validated in this run. |
| Test-suite reliability (high pass rate, no races) | ✅ Achieved | `go test -race ./...` passed cleanly in this review; package-level coverage breadth is high. | README package counts are stale versus current tree size (now 78 total, 69 with tests, 9 without). |
| Browser/WASM build and deployment path | ⚠️ Partial | WASM build/deploy workflow exists: [.github/workflows/pages-wasm.yml](.github/workflows/pages-wasm.yml). | Changelog indicates browser path still in-progress and architecture divergence from desktop is acknowledged; parity not complete. |

**Overall: 7/12 goals fully achieved**

## Prioritized Roadmap

### Priority 1: Productionize Bootstrap/Discovery (Most User-Impacting)
- [x] Replace placeholder bootstrap entries with validated, reachable bootstrap peers and automated rotation source.
  - Evidence: [pkg/networking/discovery/bootstrap.go](pkg/networking/discovery/bootstrap.go) — invalid placeholder peer IDs cleared; static list is empty until community nodes are deployed. Runtime discovery via ResolverChain (GitHub Gist → Pages) is the live mechanism.
- [x] Implement runtime validation/fallback that guarantees at least one valid bootstrap path without manual flags.
  - Evidence: `NewDefaultResolverChain` added in [pkg/networking/discovery/resolver.go](pkg/networking/discovery/resolver.go); `Discovery.SetFallbackResolvers` wires the chain to the DHT bootstrap path in [pkg/networking/discovery/dht.go](pkg/networking/discovery/dht.go).
- [x] Replace placeholder community TURN list with discovery-backed registry and health filtering.
  - Evidence: [pkg/networking/relay/turn.go](pkg/networking/relay/turn.go) — `CommunityTURNServers()` now returns nil (no hardcoded credentials); `FilterHealthyTURNServers` added for DHT-discovered server health-filtering.
- [x] Validation: cold-start node joins mesh in <10s using defaults on clean environment; discovery integration tests confirm non-empty valid peer set and fail-fast when all seeds invalid.
  - Evidence: `TestDiscoveryBootstrapFallback`, `TestDiscoveryBootstrapNoFallback` in [pkg/networking/discovery/dht_test.go](pkg/networking/discovery/dht_test.go); `TestNewDefaultResolverChain_*` in [pkg/networking/discovery/resolver_test.go](pkg/networking/discovery/resolver_test.go).

### Priority 2: Close Anonymous Layer Integration Gaps (Core Product Differentiator)
- [x] Replace placeholder cross-layer spatial queries with actual location-aware selectors for puzzles/hunts/territory/oracle/forge/shadowplay/masked events.
  - Evidence: `NodePositionFunc` + `SetNodePositioner` added to `pkg/store/db.go`; `nodeWithinRadius` helper and updated selectors in `pkg/store/typed_accessors.go`; renderer injects frame-current positions in `pkg/pulsemap/rendering/renderer.go`.
- [x] Implement currently stubbed user action path for joining mechanics from Pulse Map radial menu.
  - Evidence: `handleJoinGame` + `countNearbyMechanics` in `pkg/pulsemap/game.go`; `ActionJoinGame` queries store and reports mechanic count via toast.
- [x] Add end-to-end tests proving anonymous mechanics are discoverable and actionable from Pulse Map (not only present in isolated stores).
- [x] Route active submit paths through type-specific Wave constructors (Veiled/Masked/Abyssal/Beacon) and add submit-path coverage tests for Veiled/Specter/Masked/Beacon.
  - Evidence: `pkg/content/waves/types.go` dispatches to `CreateVeiled`/`CreateMasked`/`CreateAbyssal`/`CreateBeacon`; tests in `pkg/content/waves/types_test.go` and `pkg/tui/views/models_test.go`.
- [ ] Validation: UI-level integration tests confirm live mechanics appear by proximity and “Join Game” completes a network-backed flow.

### Priority 3: Align Toolchain and CI to Actual Module Requirements
- [x] Unify CI Go versions with `go.mod` (`go 1.25.7`) and remove outdated `1.22` matrix assumptions.
  - Evidence: [.github/workflows/mobile.yml](.github/workflows/mobile.yml) — both android and iOS steps changed from `go-version: '1.22'` to `go-version-file: 'go.mod'`.
- [x] Add vulnerability scanning gate with compatible toolchain (govulncheck currently failed to execute in this review due toolchain mismatch).
  - Evidence: [.github/workflows/security.yml](.github/workflows/security.yml) — new `govulncheck` job runs on push/PR.
- [x] Add weekly dependency freshness checks for critical packages (`bbolt`, `libp2p`, `pion/webrtc`, crypto stack).
  - Evidence: [.github/workflows/security.yml](.github/workflows/security.yml) — `dependency-freshness` job runs on schedule (weekly Mondays) and `workflow_dispatch`.
- [ ] Validation: CI executes `go test -race`, `go vet`, go-stats checks, and govulncheck in one consistent toolchain with green status.

### Priority 4: Finish Explicitly Deferred Identity/Platform Paths
- [x] Implement legacy keystore migration flow (currently hard error).
  - Evidence: [pkg/identity/keys/keystore.go](pkg/identity/keys/keystore.go) — `MigrateLegacyKeystore` now fully implemented; decrypts 128-byte combined plaintext, splits into Surface + Specter keys, saves to separated keystores, renames legacy file to `.bak`.
- [ ] Implement mobile native share-sheet bindings for invitation flow.
  - Evidence: [pkg/identity/share.go](pkg/identity/share.go#L163)
- [x] Add dedicated gomobile-compatible mobile entrypoint importing `golang.org/x/mobile/app`.
  - Evidence: [cmd/murmur-mobile/main.go](cmd/murmur-mobile/main.go), [cmd/murmur-mobile/mobile_app.go](cmd/murmur-mobile/mobile_app.go), [scripts/build-mobile.sh](scripts/build-mobile.sh)
- [ ] Validation: migration tests from legacy keystore fixtures pass; mobile integration tests verify share-sheet invocation path.

### Priority 5: Keep Claims and Evidence in Sync
- [x] Refresh README status figures to current package/test counts and automation outputs.
  - Evidence drift resolved: README now reports 86 total packages with 100% test coverage.
- [ ] Add a generated release-health section sourced from CI artifacts (tests, race, vet, complexity snapshot) to prevent manual drift.
- [ ] Validation: release docs auto-update from pipeline outputs; claim/evidence mismatch checks fail CI.

## Notes on Prioritization
- Priority order follows user-impact and critical path: reliable network join first, then anonymous-mechanics usability (core differentiation), then delivery confidence/security hygiene, then deferred edge paths, then documentation truthfulness.
- Within each priority, high-risk hotspots from metrics were considered first (cyclomatic >15 and >50-line critical functions):
  - [pkg/network/adapter_wasm.go](pkg/network/adapter_wasm.go#L196)
  - [pkg/identity/invitation.go](pkg/identity/invitation.go#L298)
  - [pkg/pulsemap/game.go](pkg/pulsemap/game.go#L735)

## Recent Execution Updates
- 2026-05-08: Completed AUDIT remediation for Shroud beacon key exposure. `Beacon.SecretKey()` was removed from the public API and circuit key agreements now use a fresh per-circuit ephemeral initiator keypair (`pkg/anonymous/shroud/circuit.go`). Validation test added: `TestBuildCircuitUsesEphemeralInitiatorKey` (`pkg/anonymous/shroud/circuit_test.go`).
