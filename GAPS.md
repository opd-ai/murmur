# Implementation Gaps — 2026-05-07

## G1: WASM Runtime Is Placeholder (Critical Path)
- **Intended Behavior**: Browser/WASM target should initialize and run the same user-visible Pulse Map/UI runtime path promised by README and deployment docs.
- **Current State**: pkg/game/runtime_wasm.go:129 and pkg/game/runtime_wasm.go:138 block on context cancellation and explicitly defer full UI initialization.
- **Blocked Goal**: Browser build parity and real browser usability.
- **Implementation Path**: Implement wasm runtime bootstrap to instantiate app/game components, initialize UI + Pulse Map, and enter active render/update loop; ensure graceful close/unload semantics.
- **Dependencies**: Input + network adapter integration for js/wasm target, asset loading in browser context.
- **Effort**: large

## G2: Anonymous Mechanics Inbound Pipeline Is Partially Stubbed
- **Intended Behavior**: Anonymous mechanics messages should be decoded, validated, routed by mechanic type, persisted, and emitted to event bus.
- **Current State**: pkg/app/handlers.go:370 acknowledges receipt with comments indicating deferred implementation.
- **Blocked Goal**: Anonymous Layer first-class mechanics operationality.
- **Implementation Path**: Decode envelope payload into GossipMessage, route to gift/mark/puzzle/hunt/etc handlers, verify signatures/ZK where required, persist by mechanic domain, emit typed events.
- **Dependencies**: Stable mechanic payload schema and per-domain stores.
- **Effort**: large

## G3: Bootstrap Signature Verification Trust Chain Not Enforced
- **Intended Behavior**: Signed peer lists should be verified against embedded bootstrap public key before use.
- **Current State**: pkg/networking/discovery/verify_key.go:14 leaves key nil; pkg/networking/discovery/http_resolver.go:65 accepts empty key path (no verification).
- **Blocked Goal**: Security/privacy architecture around bootstrap authenticity.
- **Implementation Path**: Embed real provisioning key, require non-empty key in trusted resolver configs, reject unsigned/invalid data when verification is expected.
- **Dependencies**: Key provisioning workflow + CI secret handling.
- **Effort**: medium

## G4: Discovery Resolver Components Exist but Are Unreachable
- **Intended Behavior**: ResolverChain should orchestrate static + remote resolvers (pages/gist/ipfs) in production bootstrap flow.
- **Current State**: Constructors exist (pkg/networking/discovery/pages_resolver.go:23, gist_resolver.go:23, ipfs_gateway_resolver.go:31, resolver.go:33) but no production call sites were found.
- **Blocked Goal**: Multi-source bootstrap resilience described in planning docs.
- **Implementation Path**: Wire resolver chain creation in app/network startup, feed discovered peers into dial/bootstrap policy, log resolver outcomes.
- **Dependencies**: G3 (verification key wiring).
- **Effort**: medium

## G5: Legacy Browser Transport Surface Is Dead/Partial
- **Intended Behavior**: Browser transport should either be fully functional or removed from active architecture.
- **Current State**: pkg/networking/transport/browser_host.go has TODO no-ops in Connect and SetStreamHandler (lines 80, 91) and constructors with no production references.
- **Blocked Goal**: Clean, maintainable browser transport architecture.
- **Implementation Path**: Choose one path:
  1. Deprecate/remove browser_host and standardize on pkg/network wasm adapter.
  2. Fully implement relay/WebRTC connect and stream handlers, then wire from runtime.
- **Dependencies**: Runtime transport selection policy.
- **Effort**: medium

## G6: Identity Key Storage Diverges from Encryption Requirement
- **Intended Behavior**: Surface key material should be encrypted at rest using keystore policy.
- **Current State**: pkg/app/murmur.go:368 stores keypair raw with explicit TODO for passphrase support.
- **Blocked Goal**: Security/privacy compliance with technical specification.
- **Implementation Path**: Route identity writes through encrypted keystore API, support first-run passphrase capture, and migrate existing plaintext entries.
- **Dependencies**: UX flow for passphrase onboarding and recovery compatibility.
- **Effort**: medium

## G7: Nudge Feature Not Wired to Event Bus/UI
- **Intended Behavior**: Nudges should dispatch as typed events and render in UI notification flow.
- **Current State**: pkg/app/nudges.go:156 leaves TODO and prints to stdout.
- **Blocked Goal**: In-app onboarding retention nudges.
- **Implementation Path**: Add NudgeEvent type, emit through EventBus, subscribe in UI layer, add persistence for shown-state transitions.
- **Dependencies**: Existing event bus subscriber registration.
- **Effort**: small

## G8: Onboarding Display Name Is Captured But Not Persisted
- **Intended Behavior**: User-selected display name should be stored in identity declaration.
- **Current State**: pkg/app/ui.go:102 includes TODO and does not write name into identity/declaration pipeline.
- **Blocked Goal**: Self-sovereign identity profile completeness.
- **Implementation Path**: Persist callback value to declaration state/store and trigger declaration publish/update.
- **Dependencies**: Identity declaration publisher lifecycle.
- **Effort**: small

## G9: Observability Surface Partially Unwired
- **Intended Behavior**: Declared metrics should be driven by runtime behavior.
- **Current State**: Several metrics are defined in pkg/networking/metrics/metrics.go (e.g., lines 89, 107, 115, 123, 131, 140, 150) but only referenced in metrics tests.
- **Blocked Goal**: Operator visibility into bootstrap, shroud, memory/cache, scoring, and rate-limit health.
- **Implementation Path**: Add instrumentation in bootstrap attempts/success paths, shroud circuit lifecycle, cache manager updates, peer scoring/rate-limit drop points.
- **Dependencies**: Runtime hooks in networking/shroud/content subsystems.
- **Effort**: medium

## G10: App Contract Interfaces Are Not Wired
- **Intended Behavior**: pkg/app interfaces should be used as real dependency boundaries or removed.
- **Current State**: pkg/app/interfaces.go (lines 15, 34, 51, 64) defines contracts with comments claiming implementations, but there are no production references/implementations found.
- **Blocked Goal**: Clear dependency inversion and testable subsystem seams.
- **Implementation Path**: Either inject these interfaces into App/subsystems with concrete adapters and tests, or delete stale contracts and update docs/comments.
- **Dependencies**: Decision on architecture direction (DI boundary vs direct concrete deps).
- **Effort**: small

## G11: Beacon PoW Validation Deferred
- **Intended Behavior**: Beacon waves should enforce elevated PoW validation prior to acceptance.
- **Current State**: pkg/app/handlers.go:401 explicitly defers full PoW validation.
- **Blocked Goal**: Anti-spam guarantees for beacon channel.
- **Implementation Path**: Extend beacon payload/schema to include nonce/proof fields, validate leading-zero difficulty before relay registration.
- **Dependencies**: Proto/schema update and migration compatibility.
- **Effort**: medium

## Sequenced Implementation Roadmap
1. Close critical path: G1 and G2.
2. Close trust/bootstrap chain: G3 then G4.
3. Resolve browser transport architecture debt: G5.
4. Close security-at-rest gap: G6.
5. Finish partially wired UX and observability: G7, G8, G9.
6. Resolve architectural contract drift: G10.
7. Complete beacon anti-spam hardening: G11.

## Validation Checklist Per Gap Closure
- go build ./...
- go vet ./...
- go test affected packages with focused cases for newly implemented path
- Re-run go-stats-generator analyze . --skip-tests and compare TODO/stub deltas
