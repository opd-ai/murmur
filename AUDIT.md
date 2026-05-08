# IMPLEMENTATION GAP AUDIT — 2026-05-08

## Project Architecture Overview
- **Intended system**: decentralized P2P social network with dual identity layers (Surface + Specter), Wave-based ephemeral content, Shroud anonymity routing, Resonance reputation, Pulse Map UI, and onboarding flow (README.md, docs/DESIGN_DOCUMENT.md, docs/TECHNICAL_IMPLEMENTATION.md).
- **Current module state**: `go.mod` is `github.com/opd-ai/murmur` on Go 1.25.7 with 86 packages discovered via `go list ./...`.
- **Dependency graph highlights** (from `go list` import graph):
  - Highest fan-out packages: `pkg/app` (19 internal imports), `pkg/tui/views` (13), `pkg/pulsemap` (12).
  - Highest fan-in packages: `proto` (36 dependents), `pkg/store` (16), `pkg/anonymous/mechanics` (11).
- **Baseline tooling results**:
  - `go-stats-generator analyze . --skip-tests`: LOC 57,038; functions 1,649; methods 5,183; packages analyzed 79; overall doc coverage 81.64%.
  - `go build ./...`: fails in this environment due missing system X11 headers (`X11/Xlib.h`) while compiling Ebitengine GLFW backend (`tmp/gap-build-results.txt`).
  - `go vet ./...`: same environment-level X11 header failure (`tmp/gap-vet-results.txt`).
- **Online research (brief)**:
  - Open issues: 0.
  - Open PRs: 0.
  - Recent closed PRs are incremental backlog closures (#11, #10, #9, #8), indicating active staged implementation rather than a stable frozen architecture.

## Gap Summary
| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 3 | 0 | 2 | 1 | 0 |
| Dead Code | 2 | 0 | 0 | 1 | 1 |
| Partially Wired | 2 | 1 | 0 | 1 | 0 |
| Interface Gaps | 1 | 0 | 1 | 0 | 0 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package
| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|-------------------:|------------:|------:|-----:|----------|
| `pkg/content/waves` | 117 | 114 | 2 | 1 | N/A (`--skip-tests`) |
| `pkg/pulsemap` | 68 | 67 | 0 | 1 | N/A (`--skip-tests`) |
| `pkg/tui/views` | 56 | 55 | 0 | 1 | N/A (`--skip-tests`) |
| `pkg/identity` | 220 | 218 | 1 | 1 | N/A (`--skip-tests`) |
| `pkg/onboarding/screens` | 124 | 123 | 1 | 0 | N/A (`--skip-tests`) |
| `pkg/onboarding/bootstrap` | 35 | 34 | 1 | 0 | N/A (`--skip-tests`) |
| `pkg/anonymous/resonance` | 211 | 208 | 1 | 2 | N/A (`--skip-tests`) |
| `pkg/networking/mesh` | 66 | 65 | 1 | 1 | N/A (`--skip-tests`) |

> Note: completeness counts are audit heuristics from confirmed findings; they are not raw go-stats fields.

## Findings
### CRITICAL
- [x] **Wave type execution path ignores type-specific constructors on active submit paths** — `pkg/pulsemap/game.go:974`, `pkg/tui/views/waves.go:116`, `pkg/content/waves/types.go:83` — both Pulse Map and TUI call generic `waves.Create(...)` for all selected types, but specialized Wave implementations exist separately (`pkg/content/waves/veiled.go`, `masked.go`, `abyssal.go`, `beacon.go`) and are bypassed in live creation flows. This blocks the stated goal that each Wave type has its own semantics/crypto path. — **Remediation:** Route Wave creation by type in submit handlers (or centrally inside `waves.Create`) to call the type-specific constructors and enforce type-specific validation; add integration tests covering at least Veiled/Specter/Masked/Beacon submissions from Pulse Map and TUI; validate with `go build ./...`, `go vet ./...`, and targeted package tests.

### HIGH
- [ ] **Veiled encryption key wrapping is explicitly simplified and not the intended DH-based scheme** — `pkg/content/waves/veiled.go:184-186` — code uses XOR-wrapped symmetric key with derived hash material and comment states proper X25519 DH is not implemented. This conflicts with documented cryptographic intent for anonymous/private exchange and can produce incorrect security guarantees. — **Remediation:** Replace `wrapSymmetricKey`/`UnwrapSymmetricKey` flow with X25519 shared-secret derivation + HKDF and authenticated key wrapping; update `encryptVeiledContent` and decryption parity tests; verify with deterministic crypto round-trip tests and malformed-key negative tests.
- [ ] **Resonance claim implementation contract mismatch (Pedersen/Bulletproof claim vs simplified arithmetic model)** — `pkg/anonymous/resonance/claims.go:1-3`, `pkg/anonymous/resonance/claims.go:149-153` — file-level contract claims Pedersen-style ZK path, but commitment combines points via XOR and comments call out simplification. This creates a documented-vs-implemented contract gap for security-sensitive claims. — **Remediation:** Either (a) delete/deprecate this legacy claim path and wire all claim generation/verification through `pedersen.go` adapter, or (b) implement real group operations and proofs; add compile-time and runtime wiring tests ensuring only one claim path is active.

### MEDIUM
- [ ] **Mobile invitation share-sheet path is stubbed with explicit runtime error** — `pkg/identity/share.go:159-163` — Android/iOS path returns "not yet implemented" and does not invoke platform bindings, blocking documented invite sharing behavior on mobile. — **Remediation:** Implement platform-specific mobile bridge (gomobile bindings) in `OpenSystemShare`; add mobile-targeted unit/integration tests for text/email/QR share requests; validate with mobile build pipeline.
- [ ] **Onboarding completion invite generation is disconnected from signed invitation protocol** — `pkg/onboarding/screens/completion_screen.go:324-345` — completion flow generates display-only `MURMUR-XXXX-YYYY`-style code from key bytes instead of using `identity.GenerateSignedInvitation` + URI encoding; this bypasses invitation metadata and bootstrap addresses expected by onboarding bootstrap acceptance (`pkg/onboarding/bootstrap/invitation.go:14-30`). — **Remediation:** Replace `generateInviteCode` path with actual invitation generation/encoding and attach resulting URI to copy/share actions; add onboarding acceptance E2E test using generated code from completion screen.
- [ ] **Invitation bootstrap fallback emits `/p2p/<peerID>` without transport address** — `pkg/onboarding/bootstrap/invitation.go:41-45` — fallback comment explicitly states this is incomplete and relies on later discovery. This weakens warm-start guarantees when signed bootstrap addrs are absent. — **Remediation:** Include at least one dialable multiaddr in invitation payloads by default and only use `/p2p/` as a last-resort diagnostic value; add failure-mode tests for unsigned/legacy invitation acceptance.
- [ ] **Datacenter classification feature exists but is non-functional and unintegrated** — `pkg/networking/mesh/diversity.go:302-304` — `IsDatacenter` always returns `false`; test explicitly notes non-implementation (`pkg/networking/mesh/diversity_test.go:269`). This leaves geographic diversity scoring partially implemented. — **Remediation:** Implement CIDR/provider-backed classification and wire into diversity decisions; add tests with known datacenter/non-datacenter fixtures.

### LOW
- [ ] **Legacy resonance claims API appears dead in runtime wiring** — `pkg/anonymous/resonance/claims.go:80`, `pkg/anonymous/resonance/claims.go:203` — symbol search shows runtime mechanics use `pedersen.go` verifier adapter (`pkg/anonymous/mechanics/councils/councils.go:469`), while `NewClaimGenerator` / `NewClaimVerifier` are only referenced in tests. — **Remediation:** Remove or deprecate unused API surface (or wire it intentionally); if retained for external API, mark as legacy and add explicit compatibility tests.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| Build-tagged `*_stub.go` files under `pkg/ui`, `pkg/pulsemap`, `pkg/network`, `pkg/app` | Rejected as intentional test/headless/platform shims (`//go:build test`, `//go:build js && wasm`, etc.), not accidental unfinished production paths. |
| `go build`/`go vet` failures as intrinsic code defect | Rejected as environment dependency issue (missing system X11 headers for Ebitengine GLFW) rather than direct implementation gap in repository logic. |
| TODO in `pkg/content/storage/cache_test.go:398` | Rejected as test-only deferred scenario; does not block current production behavior. |
| Exported invitation helpers with no internal caller | Rejected as possible public API surface; only flagged when coupled to an explicit broken wiring path (completion screen invite generation). |
