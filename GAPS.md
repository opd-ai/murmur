# Implementation Gaps — 2026-05-08

## Wave type creation path bypasses type-specific implementations
- **Intended Behavior**: Selected Wave types (Surface, Reply, Veiled, Specter, Sigil, Abyssal, Masked, Beacon) should execute their type-specific creation logic and constraints.
- **Current State**: Active submit paths call generic `waves.Create(...)` for every type (`pkg/pulsemap/game.go:974`, `pkg/tui/views/waves.go:116`), bypassing specialized constructors/files.
- **Blocked Goal**: Correct behavior of multi-type Wave system and cross-layer semantics.
- **Implementation Path**: Add centralized type dispatch in `pkg/content/waves` (or handlers) that routes each type to proper constructor; reject unsupported combinations explicitly; extend submit-path tests for Pulse Map and TUI.
- **Dependencies**: Veiled crypto path hardening (below) should land before finalizing Veiled submission support.
- **Effort**: medium

## Veiled key wrapping uses simplified XOR instead of DH key agreement
- **Intended Behavior**: Veiled content encryption should use robust key agreement/wrapping compatible with documented cryptographic model.
- **Current State**: `encryptVeiledContent` notes simplified key wrapping and uses XOR-derived wrap key (`pkg/content/waves/veiled.go:184-186`).
- **Blocked Goal**: Trustworthy private/anonymous message confidentiality semantics.
- **Implementation Path**: Replace wrap/unwrap with X25519 shared-secret derivation + HKDF and AEAD-authenticated wrapping; add round-trip tests and tamper/failure tests.
- **Dependencies**: Wave type dispatch fix to ensure this path is used in production creation flows.
- **Effort**: medium

## Mobile invitation share-sheet missing platform bindings
- **Intended Behavior**: Mobile users should be able to trigger native share sheets for invitation text/email/QR.
- **Current State**: Mobile branch returns explicit runtime error (`pkg/identity/share.go:163`).
- **Blocked Goal**: Invite-first onboarding and growth flows on Android/iOS.
- **Implementation Path**: Add gomobile bridge layer for Android/iOS share intents/sheets and integrate with `OpenSystemShare`; add mobile-target smoke tests.
- **Dependencies**: Existing invitation URI generation logic can be reused.
- **Effort**: medium

## Completion-screen invite code is placeholder-like and not protocol invitation
- **Intended Behavior**: Onboarding completion should produce usable invitation payloads compatible with bootstrap acceptance.
- **Current State**: Completion screen generates synthetic `MURMUR-...` code (`pkg/onboarding/screens/completion_screen.go:324-345`) instead of signed invitation URI.
- **Blocked Goal**: Warm-start bootstrap via invitations from first-run UI.
- **Implementation Path**: Replace local code generation with `identity.GenerateSignedInvitation` and `EncodeURI`; wire copy/share callbacks to actual URI output; add e2e onboarding invitation acceptance test.
- **Dependencies**: None beyond identity key availability.
- **Effort**: small

## Invitation bootstrap fallback path is weak for legacy invites
- **Intended Behavior**: Invitation acceptance should provide dialable bootstrap addresses whenever possible.
- **Current State**: Fallback returns only `/p2p/<peerID>` with comment acknowledging incomplete addressing (`pkg/onboarding/bootstrap/invitation.go:41-45`).
- **Blocked Goal**: Reliable inviter-first connection in disconnected/bootstrap-poor environments.
- **Implementation Path**: Ensure generated invitations carry at least one validated multiaddr; harden fallback selection and report explicit errors for non-dialable fallback.
- **Dependencies**: Completion-screen invite integration should emit signed invitations with addresses.
- **Effort**: small

## Datacenter detection logic is present but non-functional
- **Intended Behavior**: Diversity heuristics should distinguish datacenter peers when scoring/selection.
- **Current State**: `IsDatacenter` always returns false (`pkg/networking/mesh/diversity.go:302-304`) and test notes non-implementation (`pkg/networking/mesh/diversity_test.go:269`).
- **Blocked Goal**: Full geographic/topology diversity heuristics promised by mesh subsystem.
- **Implementation Path**: Implement CIDR/provider-based classification and integrate into diversity decision points; add deterministic fixture tests.
- **Dependencies**: None.
- **Effort**: small

## Legacy resonance claims path is effectively dead beside pedersen adapter
- **Intended Behavior**: Single coherent ZK claim path should back mechanics gating.
- **Current State**: Runtime mechanics call pedersen verifier adapter (`pkg/anonymous/mechanics/councils/councils.go:469`), while `claims.go` generator/verifier constructors are test-only referenced.
- **Blocked Goal**: Maintainable and auditable cryptographic claim surface.
- **Implementation Path**: Deprecate/remove legacy path or wire it intentionally; if kept, enforce explicit adapter selection and add compatibility tests.
- **Dependencies**: Decision on canonical claim stack (pedersen-only vs dual path).
- **Effort**: small

