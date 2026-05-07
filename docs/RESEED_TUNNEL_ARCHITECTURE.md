# Reseed as a Specialized Tunnel Application

Date: 2026-05-06
Scope: PLAN.md 7.2
Status: Design complete for implementation planning

## Objective

Reuse Phase 6 tunnel infrastructure for friend-to-friend reseed instead of introducing a separate transport stack.

## Architectural Decision

Reseed runs as a constrained application profile on top of tunnel primitives:
- same circuit construction path,
- same accounting hooks,
- same abuse controls,
- different protocol message types and stricter policy defaults.

This keeps implementation aligned with "build once, specialize by profile".

## Protocol Surface

Define reseed-specific protocol IDs and message envelopes:
- stream protocol: /murmur/reseed/1
- control message type: RESEED_REQUEST
- response message type: RESEED_BUNDLE
- error message type: RESEED_DENY

All messages remain protobuf-encoded and optionally wrapped in MurmurEnvelope where applicable.

## Message Concepts

1. ReseedRequest
- request_id
- capability_token
- client_version
- requested_transport_mode
- max_peers_requested

2. ReseedBundle
- request_id
- bundle_id
- issued_at/expires_at
- peers[]
- transport_hints[]
- reseed_host_signature

3. ReseedDeny
- request_id
- reason_code
- retry_after_seconds (optional)

## Policy Profile Differences vs General Tunnel

Reseed profile must enforce stricter defaults:
- no arbitrary destination routing,
- no content proxying,
- fixed payload type (bootstrap bundle only),
- small bounded response size,
- short request timeout,
- stronger per-peer rate limits.

## Reused Components

From Phase 6 tunnel work:
- tunnel addressing and registration lifecycle,
- Shroud multi-hop transport path,
- accounting recorder for quota enforcement,
- host policy evaluation and refusal reasons,
- diagnostics and health reporting patterns.

## New Components

- reseed handler package (bootstrap bundle generation + signing),
- capability verifier (scope, expiry, replay checks),
- reseed protocol codec helpers,
- reseed-specific metrics labels.

## Data Flow

1. Requestor opens reseed stream through tunnel path.
2. Host validates capability and local policy.
3. Host builds and signs bounded bootstrap bundle.
4. Host returns RESEED_BUNDLE or RESEED_DENY.
5. Requestor validates signature/expiry and attempts bootstrap.

## Safety and Abuse Constraints

- Max bundle peers: 64.
- Max bundle TTL: 24h.
- Capability default: single-use, 7-day expiry.
- Per-peer reseed request rate limit (example: 12/hour).
- Cooldown on repeated invalid capability attempts.

## Implementation Staging

Stage 1: Protocol definitions
- add reseed protobuf messages and constants.

Stage 2: Handler skeleton
- implement request validation and deny path.

Stage 3: Bundle generation
- generate signed bounded bundles from local peer view.

Stage 4: Integration
- mount handler on tunnel-aware transport path.

Stage 5: Validation
- integration tests for success, deny, replay, and stale bundle cases.

## Completion Criteria for PLAN 7.2

- Reseed explicitly defined as tunnel-application profile.
- Reuse boundaries and new components identified.
- Reseed message types/protocol surface defined.
- Abuse and safety constraints documented for implementation.
