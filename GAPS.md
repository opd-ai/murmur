# Implementation Gaps — 2026-04-13

This document identifies gaps between MURMUR's stated goals (as documented in README.md, DESIGN_DOCUMENT.md, TECHNICAL_IMPLEMENTATION.md, and related specification files) and the current implementation state.

---

## Gap 1: Bootstrap Node Infrastructure

- **Stated Goal**: Per NETWORK_ARCHITECTURE.md, new nodes should be able to join the network using hardcoded bootstrap node addresses without manual configuration.
- **Current State**: `pkg/config/config.go:14-17` contains `DefaultBootstrapPeers` as an empty slice with a TODO comment: "Replace with actual bootstrap node addresses once infrastructure is established."
- **Impact**: New users cannot discover the network. The application will start but remain isolated without manually configured peer addresses. This is a **blocking gap** for any user adoption.
- **Closing the Gap**:
  1. Deploy 3+ community-operated bootstrap nodes across different jurisdictions (EU, US, Asia)
  2. Configure stable public multiaddrs for each bootstrap node
  3. Update `DefaultBootstrapPeers` in `pkg/config/config.go` with the deployed addresses
  4. Establish monitoring and uptime guarantees (target: 99.9% availability)
  5. Document bootstrap node operation in `docs/BOOTSTRAP_OPERATION.md`
  6. **Validation**: New node joins network within 30 seconds using only default configuration

---

## Gap 2: Application Subsystem Wiring

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §8, the application should initialize ~8 persistent goroutines (main/Ebitengine, network/libp2p, layout/force-directed, expiry/GC, heartbeat, Shroud maintenance, event bus, DHT refresh) with a central event bus for coordination.
- **Current State**: `pkg/app/app.go:66-90` contains a `Run()` method with a TODO listing 7 subsystems but only prints a version string and blocks on context. Subsystems exist in `pkg/` but are not instantiated or wired together at startup.
- **Impact**: Running `go run ./cmd/murmur` starts an application that does nothing — no network connection, no UI, no storage initialization. The individual subsystems are implemented but never used.
- **Closing the Gap**:
  1. Add subsystem dependencies to `App` struct (store, identity, networking, content, anonymous, pulsemap, onboarding)
  2. Implement initialization in dependency order: Storage → Identity → Networking → Content → Anonymous → Pulse Map → Onboarding
  3. Start event bus goroutine for fan-out communication
  4. Wire GossipSub message handlers to content/propagation
  5. Start Ebitengine game loop with Pulse Map renderer
  6. **Validation**: `go run ./cmd/murmur` opens Pulse Map window and connects to network

---

## Gap 3: Identity Declaration Implementation

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §2.1, identity declarations are signed announcements broadcast via GossipSub on `/murmur/identity/1` topic, containing public key, display name, timestamp, and signature.
- **Current State**: `pkg/identity/declarations/declarations.go:7-20` contains only a `Declaration` struct with basic fields and a TODO comment: "Implement per TECHNICAL_IMPLEMENTATION.md §2.1." No methods for signing, validation, serialization, or broadcast exist.
- **Impact**: Users cannot announce their identity to the network. Profile information cannot be shared. The identity declaration message type defined in `proto/identity.proto` has no implementation.
- **Closing the Gap**:
  1. Implement `NewDeclaration(kp *keys.KeyPair, displayName string) *Declaration`
  2. Implement `Sign(kp *keys.KeyPair) error` using Ed25519
  3. Implement `Verify() error` for signature validation
  4. Implement `Marshal() ([]byte, error)` using protobuf
  5. Add broadcast method using `pkg/networking/gossip` on `TopicIdentity`
  6. **Validation**: `go test ./pkg/identity/declarations/... -v` passes with sign/verify round-trip tests

---

## Gap 4: Test Coverage for Rendering Packages

- **Stated Goal**: Per ROADMAP.md, target test coverage is >80% for core packages. All packages should have test files.
- **Current State**: `go test ./...` reports `[no test files]` for:
  - `github.com/opd-ai/murmur/cmd/murmur`
  - `github.com/opd-ai/murmur/pkg/onboarding/screens` (partial)
  - `github.com/opd-ai/murmur/pkg/pulsemap/overlays` (partial)
  - `github.com/opd-ai/murmur/pkg/pulsemap/rendering` (partial)
  - `github.com/opd-ai/murmur/pkg/pulsemap/rendering/effects` (partial)
- **Impact**: Rendering and UI logic changes cannot be validated automatically. Regressions may go undetected. CI cannot verify visual correctness.
- **Closing the Gap**:
  1. Add `cmd/murmur/main_test.go` with tests for `run()` error paths
  2. Add unit tests for screen state machines (independent of Ebitengine)
  3. Add headless rendering tests using Ebitengine's headless mode
  4. Add effect configuration tests for shader parameter validation
  5. **Validation**: `go test ./... | grep -c "\[no test files\]"` returns 0

---

## Gap 5: Real-World Network Validation

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §7, performance targets include <500ms Wave propagation across 3 hops, <3s Shroud circuit construction, 60fps @ 500 nodes, memory <256 MiB.
- **Current State**: All testing uses in-memory transports and controlled simulation environments. Actual network latency, packet loss, and NAT complexity are not validated. Cold start time is not measured.
- **Impact**: Performance may not meet targets under real network conditions. NAT traversal success rate is unknown. Users on residential networks may experience degraded performance.
- **Closing the Gap**:
  1. Deploy 10-node test network across residential and cloud environments
  2. Measure actual Wave propagation latency (target: <500ms)
  3. Validate NAT traversal success rate (target: >80% with DCUtR)
  4. Test Shroud circuit construction over real network (target: <3s)
  5. Measure cold start time (target: <5s) and memory usage (target: <256 MiB)
  6. Document results in `docs/PERFORMANCE_VALIDATION.md`
  7. **Validation**: 10 nodes maintain stable mesh for 72 hours with <1% message loss

---

## Gap 6: Mobile Platform Support

- **Stated Goal**: Per ROADMAP.md Priority 3, mobile platform support via Ebitengine's gomobile integration is planned for iOS and Android.
- **Current State**: No gomobile build configuration exists. No touch input adaptations in `pkg/pulsemap/interaction/`. No mobile-specific UI scaling.
- **Impact**: Social networks require mobile access. Desktop-only availability limits user adoption significantly.
- **Closing the Gap**:
  1. Configure gomobile build pipeline (`scripts/build-mobile.sh`)
  2. Adapt touch input handling in `pkg/pulsemap/interaction/input.go`
  3. Implement mobile-specific UI scaling and layout adjustments
  4. Test on representative device range (low-end to flagship)
  5. Address mobile-specific power and bandwidth constraints
  6. **Validation**: Pulse Map renders at 30fps on mid-range 2023 mobile devices

---

## Summary Table

| Gap | Severity | Blocker? | Effort |
|-----|----------|----------|--------|
| Gap 1: Bootstrap nodes | HIGH | Yes | External infrastructure |
| Gap 2: Subsystem wiring | HIGH | Yes | 1-2 weeks |
| Gap 3: Declaration impl | HIGH | Yes | 2-3 days |
| Gap 4: Rendering tests | MEDIUM | No | 1 week |
| Gap 5: Network validation | MEDIUM | No | 1-2 weeks |
| Gap 6: Mobile support | LOW | No | 4-6 weeks |

---

## Recommended Priority Order

1. **Gap 3** (Declaration impl) — Enables identity announcements; quick win
2. **Gap 2** (Subsystem wiring) — Makes the application functional
3. **Gap 1** (Bootstrap nodes) — Enables network growth (requires external resources)
4. **Gap 4** (Rendering tests) — Improves CI reliability
5. **Gap 5** (Network validation) — Validates performance claims
6. **Gap 6** (Mobile support) — Expands user base

---

*Gaps analysis generated: 2026-04-13*
*Status: 3 blocking gaps (bootstrap nodes, subsystem wiring, declarations)*
