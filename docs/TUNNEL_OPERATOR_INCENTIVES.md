# Tunnel Operator Incentive Design

Date: 2026-05-06
Scope: PLAN.md 6.5 (Phase 6)
Status: Drafted for implementation planning

## Goal

Define a non-cryptocurrency incentive model for tunnel operators that:
- increases reliable tunnel capacity,
- preserves privacy and abuse controls,
- avoids legal and UX complexity introduced by tokens.

This document extends:
- TUNNEL_DESIGN.md
- TUNNEL_ABUSE_POLICY.md
- ABUSE_MODEL.md

## Design Principles

1. No cryptocurrency, no tradable asset, no payments ledger.
2. Explicit operator opt-in only; tunnel hosting remains disabled by default.
3. Rewards are capped and abuse-sensitive.
4. First-party self-traffic never earns rewards.
5. Incentive signals must not deanonymize users or relay destinations.

## Incentive Unit

Use a local, non-transferable score input called Tunnel Service Credit (TSC).

TSC is:
- computed locally by observing nodes,
- converted into bounded Resonance input,
- never exported as currency,
- never redeemable for money or protocol control.

## Eligibility Requirements

An operator is reward-eligible only when all conditions hold:
- tunnel hosting is explicitly enabled,
- host policy is published and signed,
- content and hostname allowlists are configured,
- executable payload policy remains default-deny,
- per-tunnel and per-day bandwidth quotas are enforced,
- minimum uptime and successful relay threshold are met.

Recommended minimums:
- uptime window: >= 24h rolling,
- successful relay ratio: >= 95%,
- bandwidth accounting enabled and auditable.

## Reward Formula

Per epoch (e.g., 24h), each observer computes:

TSC_epoch = W1*valid_sessions + W2*successful_bytes + W3*policy_compliance - penalties

Where:
- valid_sessions: completed sessions not flagged as abusive,
- successful_bytes: capped transferred bytes for allowed content,
- policy_compliance: binary or tiered score for enforced policy,
- penalties: abuse events, quota violations, refusal policy mismatch, repeated instability.

Suggested defaults:
- W1 = 1.0
- W2 = 0.25 per normalized GiB (capped)
- W3 = 2.0 bonus when all mandatory policy checks pass

Hard caps:
- max rewarded bytes/day per operator,
- max rewarded sessions/day per operator,
- diminishing returns after cap threshold.

## Anti-Abuse and Anti-Sybil Controls

1. No reward for self-routed traffic:
- reject sessions where initiator/operator affinity is detected locally,
- reject circular paths that return to the same trust neighborhood.

2. Diversity requirement:
- only count sessions from a minimum distinct-peer set per epoch,
- down-weight repeated sessions from same peer cluster.

3. Delayed accrual:
- reward accrues after settlement window, allowing abuse flags to invalidate credit.

4. Slashing policy:
- severe abuse policy violations zero out current epoch rewards,
- repeated violations trigger cooldown and temporary ineligibility.

5. Privacy-safe telemetry:
- aggregate counters only,
- no destination plaintext logging requirement,
- no global identity linkage.

## Operator Tiers (Optional)

Tier 0 (Starter):
- low caps,
- no reward multiplier,
- probation period.

Tier 1 (Reliable):
- proven compliance,
- standard multiplier.

Tier 2 (High Reliability):
- sustained compliance and uptime,
- modest multiplier bounded by global cap.

Tiering must never bypass abuse policy constraints.

## User Experience

Operator onboarding must show plain-language tradeoffs:
- bandwidth usage,
- legal exposure by jurisdiction,
- abuse refusal rights,
- how rewards are calculated and capped,
- explicit statement: "No tokens, no payouts, no tradable rewards."

## Rollout Plan

Phase A: Score-only dry run
- compute TSC without applying Resonance changes.

Phase B: Capped Resonance integration
- apply low-impact bounded conversion to Resonance.

Phase C: Adaptive tuning
- tune weights and caps from observed abuse/fairness outcomes.

## Open Questions

1. Should Tier 2 require manual operator attestation?
2. What minimum diversity threshold best balances fairness and privacy?
3. Should cooldown duration scale with abuse severity class?

## Completion Criteria for PLAN 6.5

- Incentive model documented without cryptocurrency dependency.
- Explicit opt-in and bandwidth cap requirements defined.
- Abuse-resistant reward logic defined with anti-Sybil controls.
- Rollout path and tuning strategy documented.
