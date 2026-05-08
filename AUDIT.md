# IMPLEMENTATION GAP AUDIT — 2026-05-07

## Project Architecture Overview
MURMUR is intended to be a decentralized dual-layer social network (Surface + Anonymous Layer) with six major subsystems: networking, identity, content, anonymous mechanics, Pulse Map UI, and onboarding. The planned architecture is documented in README, docs/TECHNICAL_IMPLEMENTATION.md, docs/DESIGN_DOCUMENT.md, ROADMAP.md, and PLAN.md.

Phase-0 mapping and baseline evidence collected:
- Module path: github.com/opd-ai/murmur (Go 1.25.7, toolchain go1.25.9)
- Package inventory: 78 packages from go list ./...
- Dependency graph size: 906 transitive packages from go list -deps ./...
- go-stats overview (skip-tests): 55,083 LOC, 1,610 functions, 5,103 methods, 844 structs, 51 interfaces, 74 packages, 367 files
- go-stats documentation coverage: overall 81.54% (functions 92.52%, methods 77.23%, types 84.90%)
- Build baseline: go build ./... passed
- Vet baseline: go vet ./... passed

Online research (<=10 min) on github.com/opd-ai/murmur:
- Open issues: 0
- Open pull requests: 0 (5 closed)
- Open projects: 0
- Milestones listed on repository navigation: 0

## Gap Summary
| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 6 | 1 | 2 | 2 | 1 |
| Dead Code | 3 | 0 | 2 | 1 | 0 |
| Partially Wired | 6 | 1 | 3 | 2 | 0 |
| Interface Gaps | 1 | 0 | 0 | 1 | 0 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package
| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|--------------------|-------------|-------|------|----------|
| pkg/game | 7 | 6 | 1 | 0 | N/A (skip-tests run) |
| pkg/app | 51 | 46 | 4 | 1 | N/A (skip-tests run) |
| pkg/networking/discovery | 58 | 52 | 1 | 2 | N/A (skip-tests run) |
| pkg/networking/transport | 30 | 26 | 2 | 2 | N/A (skip-tests run) |
| pkg/networking/metrics | 0 (functions), 19 exported vars | 12 wired | 0 | 7 unwired vars | N/A (skip-tests run) |
| pkg/networking/gossip | 90 | 88 | 0 | 0 | N/A (skip-tests run) |
| pkg/pulsemap | 17 | 15 | 2 | 0 | N/A (skip-tests run) |
| pkg/content/waves | 117 | 117 | 0 | 0 | N/A (skip-tests run) |

## Findings
### CRITICAL
- [ ] WASM runtime main path is still a placeholder and does not initialize Pulse Map/UI loop — pkg/game/runtime_wasm.go:129, pkg/game/runtime_wasm.go:138 — browser/WASM runtime cannot fulfill stated browser deployment goal in README and PLAN — **Remediation:** Implement wasmApp.Run to construct and run the actual game/UI runtime (same core flow as desktop runtime selection), including input loop, rendering startup, and shutdown path; add js/wasm integration test that asserts runtime reaches usable state; validate with go build ./... and js/wasm smoke execution.
- [ ] Anonymous mechanics topic handling acknowledges messages but does not parse, route, verify, persist, or emit events — pkg/app/handlers.go:370 — blocks Anonymous Layer mechanics from being operational on inbound gossip path — **Remediation:** Replace placeholder block with real GossipMessage decode + mechanic type dispatch (gifts/marks/puzzles/hunts/etc.), signature/ZK validation, store write, and EventBus emission; add handler tests per mechanic type and malformed payloads; validate with go test ./pkg/app -run Anonymous and go build ./....

### HIGH
- [ ] Bootstrap peer-list signature verification is effectively bypassable because verification key is unset and empty-key path returns success — pkg/networking/discovery/verify_key.go:14, pkg/networking/discovery/http_resolver.go:65 — blocks documented trust model for signed bootstrap lists — **Remediation:** Provision and embed real 32-byte Ed25519 public key, wire BootstrapVerifyKey into resolver construction, and fail closed for signed sources when key is required; add tests for invalid signature rejection; validate with go test ./pkg/networking/discovery and go vet ./....
- [ ] Discovery remote resolvers are not connected to runtime bootstrap path (constructors only defined, not used in production wiring) — pkg/networking/discovery/pages_resolver.go:23, pkg/networking/discovery/gist_resolver.go:23, pkg/networking/discovery/ipfs_gateway_resolver.go:31, pkg/networking/discovery/resolver.go:33 — feature set exists structurally but is unreachable from main execution path — **Remediation:** Instantiate ResolverChain in app/network bootstrap wiring with ordered resolvers (user/static/pages/gist/ipfs fallback), pass verification key and timeouts, and integrate result into dial flow; add integration test proving at least one remote resolver path is exercised.
- [ ] Legacy browser_host transport path contains explicit TODO no-op networking methods and is not wired from runtime — pkg/networking/transport/browser_host.go:80, pkg/networking/transport/browser_host.go:91, pkg/networking/transport/browser_host.go:25, pkg/networking/transport/browser_host.go:103 — creates dead/partial browser transport surface and maintenance risk — **Remediation:** Either remove/deprecate this path in favor of pkg/network adapter path, or fully implement relay/WebRTC connect + stream handler registration and wire from runtime; add coverage for connect/subscribe/publish over browser transport.
- [ ] Identity private key is stored unencrypted in first-run init path — pkg/app/murmur.go:368 — diverges from technical spec requirement for encrypted keystore handling and weakens at-rest protection — **Remediation:** Replace raw storage Put with keystore encryption path (Argon2id + XChaCha20-Poly1305 using passphrase flow), include migration for legacy cleartext key entries, and add round-trip and migration tests; validate with go test ./pkg/identity/keys ./pkg/app.

### MEDIUM
- [ ] Nudge subsystem is not event-bus integrated; it prints to stdout even when EventBus exists — pkg/app/nudges.go:156 — partially wired user-facing feature (no UI delivery contract) — **Remediation:** Define NudgeEvent in pkg/app/eventbus.go, emit via EventBus, and subscribe/render in UI layer; add eventbus delivery test and UI-side consumer test.
- [ ] Onboarding display name callback is not persisted into identity declarations — pkg/app/ui.go:102 — onboarding captures data but does not complete identity declaration path — **Remediation:** Persist display name into declaration storage/publisher flow in onboarding callback; add test verifying declaration contains entered display name post-onboarding.
- [ ] Several Prometheus metrics are defined but not used in production paths (only declaration/tests): ShroudCircuitsActiveGauge, DHTBootstrapAttemptsTotal, DHTBootstrapSuccessesTotal, MemoryAllocatedBytesGauge, WaveCacheEntriesGauge, PeerScoreGauge, RateLimitDropsTotal — pkg/networking/metrics/metrics.go:89, pkg/networking/metrics/metrics.go:107, pkg/networking/metrics/metrics.go:115, pkg/networking/metrics/metrics.go:123, pkg/networking/metrics/metrics.go:131, pkg/networking/metrics/metrics.go:140, pkg/networking/metrics/metrics.go:150 — observability surface is partially wired — **Remediation:** instrument these counters/gauges in actual lifecycle paths (bootstrap, shroud maintenance, cache manager, scoring, rate limiter) and assert increments in integration tests.
- [ ] App-level subsystem interfaces are documented as implemented but currently unused/unimplemented in production code paths — pkg/app/interfaces.go:15, pkg/app/interfaces.go:34, pkg/app/interfaces.go:51, pkg/app/interfaces.go:64 — abstraction layer exists without concrete wiring — **Remediation:** either wire interfaces into App dependencies and adapters, or remove stale interfaces and comments to match real architecture; validate by building App with injected mock implementations in tests.

### LOW
- [ ] Beacon-wave handler defers full PoW verification due missing nonce in beacon format — pkg/app/handlers.go:401 — non-critical correctness/security debt in anonymous beacon path — **Remediation:** extend beacon schema to carry nonce/difficulty evidence, verify before registry add, and add invalid-PoW rejection tests.
- [ ] Pulse Map still has TODO placeholders for display name/pseudonym fields in node details — pkg/pulsemap/game.go:480, pkg/pulsemap/game.go:1194 — minor UX completeness gap — **Remediation:** extend node data model with pseudonym/display-name sourcing and update render path + tests.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| return nil patterns in proto/*.pb.go | Generated protobuf code; expected and not implementation stubs. |
| TODOs in *_test.go (e.g., pkg/content/storage/cache_test.go:398) | Test-scope debt, not production execution gap. |
| *_stub.go files under build tags (test/js/wasm splits) | Intentional build-constraint stubs to support cross-target compilation/tests. |
| Extension interfaces with zero in-repo implementations (e.g., pkg/content/waves/extensions.go) | Intentional extension contract surface; designed for downstream plugins. |
| Exported functions with no internal callers in adapter packages | May be public API for external consumers; not flagged without contradictory spec/tests. |

## Evidence and Commands Executed
- Prerequisite: which go-stats-generator || go install github.com/opd-ai/go-stats-generator@latest
- Baseline metrics: go-stats-generator analyze . --skip-tests --format json --sections functions,documentation,packages,patterns,interfaces,structs,duplication
- Baseline validation: go build ./... ; go vet ./...
- Package graph: go list ./... ; go list -deps ./...
- Gap scans: rg TODO/FIXME/HACK/XXX + targeted placeholder and wiring searches
- Dependency check: go mod why -m for all direct dependencies (no unused direct module found)
