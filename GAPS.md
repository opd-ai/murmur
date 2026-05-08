# Implementation Gaps — 2026-05-07

## Closed in This Remediation

## G1: WASM Runtime Initialization
- **Status**: CLOSED
- **Intended Behavior**: Browser runtime initializes cleanly and signals readiness.
- **Current State**: Implemented runtime-state publish, unload hook, heartbeat, and non-blocking readiness return in `pkg/game/runtime_wasm.go`.
- **Validation**: `GOOS=js GOARCH=wasm go build ./cmd/wasm` PASS.

## G2: Anonymous Mechanics/Waves Inbound Pipeline
- **Status**: CLOSED
- **Intended Behavior**: Anonymous topics parse/route validated payloads.
- **Current State**: Implemented envelope-or-raw payload decoding, gossip message parsing, event-type classification, wave validation/cache callback, and beacon structural validation in `pkg/app/handlers.go`.
- **Validation**: `go test ./pkg/app` PASS.

## G3: Bootstrap Signature Verification Key Placeholder
- **Status**: CLOSED
- **Intended Behavior**: Signed peer lists verify against embedded key.
- **Current State**: Embedded non-empty verify key (`pkg/networking/discovery/verify_key.go`) and strict key-size checks in resolver verification paths.
- **Validation**: `go test ./pkg/networking/discovery` PASS.

## G4: Resolver Chain Not Connected to Startup
- **Status**: CLOSED
- **Intended Behavior**: Startup path wires resolver-chain fallback.
- **Current State**: Added resolver-chain build + bootstrap integration in `pkg/app/murmur.go`; discovery bootstrap now attempts fallback even when explicit peers are empty (`pkg/networking/discovery/dht.go`).
- **Validation**: `go test ./pkg/app ./pkg/networking/discovery` PASS.

## G5: Browser Host TODO No-Ops
- **Status**: CLOSED
- **Intended Behavior**: Browser host implements concrete behavior for connect/handler registration.
- **Current State**: Added in-memory peer bookkeeping and stream handler registry in `pkg/networking/transport/browser_host.go`.
- **Validation**: Native `go build ./...` PASS.

## G6: Identity Key Encryption at Rest
- **Status**: CLOSED
- **Intended Behavior**: Surface key persisted encrypted with migration path.
- **Current State**: Added encrypted key storage (`keypair_encrypted`), passphrase file generation/loading, and legacy plaintext migration in `pkg/app/murmur.go`.
- **Validation**: `go test ./pkg/app ./pkg/identity/keys` PASS.

## G7: Nudge Event Not Wired
- **Status**: CLOSED
- **Intended Behavior**: Nudge dispatch through EventBus.
- **Current State**: Added `EventNudge`, `NudgeEvent`, and `EmitNudge` in `pkg/app/eventbus.go`; integrated in `pkg/app/nudges.go`.
- **Validation**: `go test ./pkg/app` PASS.

## G8: Onboarding Display Name Not Persisted
- **Status**: CLOSED
- **Intended Behavior**: Display name persisted and reflected in identity declaration.
- **Current State**: Onboarding callback now stores `display_name`, updates/creates signed declaration in `pkg/app/ui.go`.
- **Validation**: `go test ./pkg/app` PASS.

## G9: Metrics Partially Unwired
- **Status**: PARTIALLY CLOSED
- **Intended Behavior**: Runtime metrics reflect live system behavior.
- **Current State**: Wired bootstrap attempt/success counters and memory/cache/circuit gauges in `pkg/app/murmur.go`.
- **Remaining**: Peer-score and rate-limit counters still require dedicated hook points.
- **Validation**: `go test ./pkg/networking/metrics ./pkg/app` PASS.

## G10: Pulse Map TODO Placeholders
- **Status**: CLOSED
- **Intended Behavior**: Bookmark/search labels use real node display data.
- **Current State**: Bookmark label now uses renderer node data with nil-guard; search pseudonym uses display-name fallback in `pkg/pulsemap/game.go`.
- **Validation**: `go test ./pkg/pulsemap` PASS.

## Remaining Open Gaps

## R1: Full Cross-OS Desktop Build Matrix in This Environment
- **Status**: OPEN (environmental)
- **Intended Behavior**: All listed desktop target triples compile in CI-like matrix from this host.
- **Current State**: Host-native and wasm command target compile; non-host desktop cross-compilation is blocked by upstream Ebitengine/OpenGL/Metal symbol/toolchain constraints in this environment.
- **Blocked Goal**: Complete local proof for every desktop target triple.
- **Implementation Path**: Validate matrix in CI runners or host-native machines per target OS/graphics stack.
- **Dependencies**: Platform-specific toolchains/SDKs and graphics backend compatibility.
- **Effort**: medium

## R2: App Interface Abstraction Cleanup
- **Status**: OPEN (architecture)
- **Intended Behavior**: `pkg/app/interfaces.go` contracts are either actively injected or removed.
- **Current State**: Contracts remain as abstraction surface without direct runtime injection wiring.
- **Blocked Goal**: Reduced architectural drift and clearer dependency boundaries.
- **Implementation Path**: Either wire real DI boundaries for `WaveStore`/`PeerRegistry`/`IdentityProvider`/`MessagePublisher`, or remove stale contracts and comments.
- **Dependencies**: App dependency-wiring decision.
- **Effort**: small

## Validation Snapshot
- `go build ./...` PASS
- `go test ./...` PASS
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` PASS
