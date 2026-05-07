# MURMUR Friend-to-Friend Reseed

Date: 2026-05-06
Status: Draft v1 (PLAN 7.4 completed)
Related: docs/RESEED_SEMANTICS.md, docs/RESEED_TUNNEL_ARCHITECTURE.md, TUNNEL_ABUSE_POLICY.md, SECURITY_PRIVACY.md

## Purpose

Reseed is a constrained recovery bootstrap path for users who cannot reach normal bootstrap peers.

Reseed is not:
- account recovery,
- identity/key recovery,
- content restoration,
- a permanent central bootstrap authority.

Reseed is:
- friend-authorized,
- capability-scoped,
- short-lived,
- bounded in payload and rate,
- designed to reduce lockout risk in hostile networks.

## User Flow

1. A friend (host) generates a reseed capability token out of band (text, QR, paper).
2. The requestor imports the token and verifies host fingerprint and expiry.
3. The client opens reseed protocol `/murmur/reseed/1` over the tunnel profile.
4. The host validates capability and policy.
5. Host returns a signed, short-lived bootstrap bundle.
6. Requestor validates bundle, then attempts bootstrap with bounded retries.
7. Capability is marked spent or decremented.

## Capability Model

Capability fields:
- host public key fingerprint,
- scope: bootstrap-only,
- max uses (default 1),
- expiry (default 7 days),
- optional transport restriction,
- nonce/replay marker.

Hard rules:
- No capability may grant messaging or content access.
- Capability cannot elevate privileges.
- Capability can be revoked locally before use.

## Bundle Model

Bundle includes only bootstrap-relevant data:
- bundle ID,
- host public key,
- issued/expires timestamps,
- network/protocol version,
- bounded peer set and transport hints,
- host signature.

Validation checks:
- signature valid,
- not expired,
- network/protocol compatible,
- bounded peer count,
- duplicate/replay rejection.

## Threat Model

### In Scope

- compromised reseed host returning poisoned peer lists,
- coerced friend host forced to return censored peers,
- replay of old capability or old bundle,
- malicious requestor abusing host resources,
- traffic shaping/censorship against known bootstrap endpoints.

### Out of Scope

- global passive adversary correlation resistance (Tor/I2P mode required),
- endpoint compromise on requestor device,
- full social graph protection against direct friend collusion.

## Threat Scenarios and Mitigations

### 1) Compromised Reseed Host

Risk:
- host serves attacker-controlled peers to isolate requestor.

Mitigations:
- signed bundle with short expiry,
- bounded bundle size and freshness window,
- peer diversity checks on client acceptance,
- optional multi-host quorum merge,
- rapid transition to normal peer exchange after first connect,
- local host denylist and capability revocation.

Residual risk:
- first-hop bootstrap quality still depends on host honesty if only one host is used.

### 2) Coerced Friend Host

Risk:
- friend is pressured to censor or bias bundle responses.

Mitigations:
- capability expiry and single-use defaults limit long-term abuse,
- optional multi-host reseed from independent friends,
- explicit warning when all peers come from one ASN/network cluster,
- host policy transparency and refusal reason logging.

Residual risk:
- a highly coerced local trust graph can still degrade reachability.

### 3) Replay of Capabilities/Bundles

Risk:
- attacker reuses stale token/bundle to force bad bootstrap attempts.

Mitigations:
- nonce and use counters,
- strict expiry checks,
- replay cache by capability ID and bundle ID,
- reject stale timestamps outside tolerance window.

Residual risk:
- bounded DoS attempts remain possible but are rate-limited.

### 4) Malicious Requestor Resource Abuse

Risk:
- brute-force capability probes, request floods, host exhaustion.

Mitigations:
- per-peer and per-capability rate limits,
- invalid-token strike counters and cooldown,
- strict payload size bounds,
- bootstrap-only protocol profile (no arbitrary proxying).

Residual risk:
- distributed abuse remains a general network concern; enforced host rights apply.

## Privacy Posture

- requestor identity is not embedded in bundles,
- host telemetry should remain aggregate-only,
- reseed transport may run over Shroud/Tor/I2P modes,
- no reseed mechanism may bridge Surface and Specter identity linkage.

## Operational Defaults

- bundle TTL: 24h,
- capability TTL: 7 days,
- default capability uses: 1,
- max peers per bundle: 64,
- strong recommendation: use at least two independent reseed hosts for high-risk users.

## Failure Handling

Client behavior:
- signature failure: hard reject,
- expired capability or bundle: hard reject with regeneration guidance,
- insufficient peer diversity: warn and request additional reseed source,
- repeated bootstrap failure: rotate transport mode and retry with fresh bundle.

Host behavior:
- invalid capability: deny + cooldown,
- policy violation: deny with reason code,
- load shedding: bounded retries and retry-after hint.

## Security Notes

Reseed must remain a narrow bootstrap primitive. Any expansion toward generic tunneling or content access increases abuse and deanonymization risk and requires separate review.

## Completion Mapping

This document fulfills PLAN 7.4 by providing:
- a threat model for compromised reseed hosts,
- a threat model for coerced friend hosts,
- concrete mitigations and residual risk discussion,
- operational defaults and failure handling guidance for implementation.
