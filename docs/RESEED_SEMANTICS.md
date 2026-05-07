# Friend-to-Friend Reseed Semantics

Date: 2026-05-06
Scope: PLAN.md 7.1
Status: Design complete for implementation planning

## Goal

Define what a reseed host provides, how authorization works, and the trust model if a reseed host is compromised.

Reseed is designed as a bootstrap recovery mechanism for constrained or censored network environments. It is not a general directory service and must not become a central authority.

## What a Reseed Host Provides

A reseed host provides a signed bootstrap bundle containing:
- peer contact set (multiaddrs + peer IDs),
- bootstrap protocol/version compatibility metadata,
- freshness metadata (issued_at, expires_at),
- optional transport hints (clearnet/Tor/I2P availability),
- reseed host policy fingerprint.

A reseed host does not provide:
- private keys,
- user identity linkage,
- content history,
- global social graph snapshots.

## Bootstrap Bundle Format (Conceptual)

Fields:
- bundle_id
- reseed_host_pubkey
- issued_at_unix
- expires_at_unix
- network_id
- protocol_version
- peers[]
- transport_hints[]
- signature

Validation requirements:
- signature valid,
- not expired,
- protocol/network match local client,
- peer list non-empty and bounded by policy.

## Authorization Model

Authorization is friend-granted and capability-limited.

A user authorizes a friend reseed host by accepting a signed reseed capability that includes:
- host public key,
- capability scope (bootstrap only),
- max uses,
- expiry,
- optional transport restrictions,
- revocation token reference.

Rules:
- default single-use capability,
- short expiry (recommended 7 days),
- capability cannot grant messaging/content permissions,
- capability can be revoked locally before use.

## Recovery Flow

1. Requestor receives signed reseed capability out-of-band.
2. Requestor validates capability and host key fingerprint.
3. Requestor fetches bootstrap bundle through tunnel/reseed transport.
4. Requestor validates bundle signature, expiry, and network/version compatibility.
5. Requestor attempts bootstrap with bounded retries and peer diversity checks.
6. Capability marked spent (or decremented use count).

## Compromised Reseed Host Trust Model

Assume reseed host compromise can cause:
- malicious or censored peer lists,
- stale peers to degrade bootstrap success,
- selective network partitioning attempts.

Mitigations:
- bundle signatures tied to known host key,
- strict expiry windows,
- peer diversity minimums before acceptance,
- optional multi-host reseed quorum (N bundles, merge/intersect policy),
- post-bootstrap peer exchange to escape poisoned initial set,
- local denylist for compromised hosts.

Non-goal:
- reseed host compromise must not expose user identity keys or message contents.

## Security Requirements

- No long-term centralized trust anchor.
- Capability scopes are minimal and non-escalating.
- Replay resistance through nonce/expiry/use counters.
- No plaintext identity metadata in bundle payload.

## Privacy Requirements

- Requestor identity not embedded in reseed bundle.
- Host telemetry limited to aggregate counters.
- Reseed traffic may be routed over Shroud/Tor/I2P per selected mode.

## Operational Requirements

- Bundle expiration default: 24 hours.
- Capability expiration default: 7 days.
- Max peers in bundle: 64 (bounded payload and abuse surface).
- Minimum unique /16 or equivalent network diversity target for initial peers.

## Completion Criteria for PLAN 7.1

- Reseed host data scope explicitly defined.
- Friend authorization mechanism defined with bounded capability.
- Compromised-host trust model and mitigations documented.
- Security/privacy constraints specified for implementation phase.
