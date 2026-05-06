# MURMUR Mini-Games Classification

**Status:** Complete  
**Date:** 2026-05-06  
**Purpose:** Classify all 10 mini-games across 4 critical axes to inform curation, retention strategy, and anonymity risk management per PLAN.md §2.1.

---

## Classification Framework

Each game is evaluated across four axes:

1. **Sync vs. Async** — Does the game require real-time interaction or can it proceed asynchronously?
2. **1:1 vs. Group** — Is the game played between two participants or within a larger group?
3. **Skill vs. Chance vs. Social** — What is the primary success factor?
4. **Anonymity Leak Surface** — What metadata does the game inherently expose?

### Anonymity Leak Surface Ratings

- **None:** No timing, behavior, or interaction metadata beyond standard Wave propagation
- **Low:** Participation timing observable but not distinctive; no latency fingerprints
- **Medium:** Interaction patterns or timing characteristics that could aid deanonymization with effort
- **High:** Real-time latency fingerprints or behavioral patterns that significantly narrow anonymity set

---

## Game Classifications

### 1. Cipher Puzzles (Fragment / Mosaic / Cascade)

**Implementation:** `pkg/anonymous/mechanics/puzzles/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | Solutions submitted over 15–60 minute windows; no real-time requirement |
| **1:1 vs. Group** | Group | Multiple Specters compete or collaborate per puzzle type |
| **Primary Factor** | Skill | Cryptographic/pattern-matching challenges reward computational/analytical skill |
| **Anonymity Leak** | **None** | Solution submission timing is coarse-grained (minutes); no distinctive behavioral fingerprint |

**Resonance Gating:** Wraith milestone (50) to initiate  
**Duration:** 15, 30, or 60 minutes  
**Verdict:** ✅ **Retain** — Core anonymous mechanic with zero anonymity cost and high skill ceiling.

---

### 2. Specter Hunts

**Implementation:** `pkg/anonymous/mechanics/hunts/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | 30–120 minute duration; exploration-driven, no real-time pressure |
| **1:1 vs. Group** | Group | Network-wide scavenger hunt with 5–20 fragments to claim |
| **Primary Factor** | Skill + Social | Navigation skill, clue decoding, plus social knowledge of Pulse Map topology |
| **Anonymity Leak** | **Low** | Fragment claim timing observable; location proximity reveals rough network position but not distinctive latency |

**Resonance Gating:** Shade-Wraith milestone (75) to initiate  
**Duration:** 30, 60, or 120 minutes  
**Verdict:** ✅ **Retain** — Exploration mechanic that drives Pulse Map engagement; leak surface acceptable given coarse timing and network-scale participation.

---

### 3. Territory Drift

**Implementation:** `pkg/anonymous/mechanics/territory/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | Persistent ambient game; influence accrues over 30-day windows |
| **1:1 vs. Group** | Group | Multiple Specters contest each territory via sustained activity |
| **Primary Factor** | Social | Sustained presence and community engagement in a network region |
| **Anonymity Leak** | **None** | Territory influence computed from existing activity (Waves, connections, mechanics); introduces no new metadata |

**Resonance Gating:** Shade milestone (25) — accessible early  
**Duration:** Persistent (weekly resets)  
**Verdict:** ✅ **Retain** — Ambient mechanic that creates persistent social stakes on the Pulse Map with zero additional anonymity cost.

---

### 4. Oracle Pools

**Implementation:** `pkg/anonymous/mechanics/oracle/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | Commitment phase → resolution period (hours to days); no real-time requirement |
| **1:1 vs. Group** | Group | Any number of Specters can submit predictions per pool |
| **Primary Factor** | Skill | Network observation and prediction accuracy |
| **Anonymity Leak** | **None** | Hash-then-reveal scheme prevents timing-based fingerprints; resolution is deterministic from network observables |

**Resonance Gating:** Phantom milestone (100) to create pools  
**Duration:** Variable (hours to days)  
**Verdict:** ✅ **Retain** — Zero-leak prediction market; commitment scheme eliminates timing metadata.

---

### 5. Sigil Forge

**Implementation:** `pkg/anonymous/mechanics/forge/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | 30 or 60 minute submission windows; evaluation via amplification over time |
| **1:1 vs. Group** | Group | Creative competition with audience evaluation |
| **Primary Factor** | Skill + Social | Creative skill evaluated by social amplification |
| **Anonymity Leak** | **None** | Entry submission and amplification follow standard Wave timing; no distinctive patterns |

**Resonance Gating:** Wraith milestone (50) to initiate  
**Duration:** 30 or 60 minutes  
**Types:** Sigil Art, Micro Fiction, Remix Chains  
**Verdict:** ✅ **Retain** — High-value creative mechanic with zero anonymity cost; drives content generation and audience engagement.

---

### 6. Shadow Play

**Implementation:** `pkg/anonymous/mechanics/shadowplay/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | 30 or 60 minute duration with 5-minute voting rounds; turn-based, not real-time |
| **1:1 vs. Group** | Group | 5–13 participants in social deduction game (Echoes vs. Shades) |
| **Primary Factor** | Social | Role identification through discussion, accusation, and voting |
| **Anonymity Leak** | **Medium** | Discussion/voting timing patterns and interaction frequency may reveal behavioral fingerprints; 5-minute round structure mitigates but doesn't eliminate timing metadata |

**Resonance Gating:** Revenant milestone (200) — highest gate  
**Duration:** 30 or 60 minutes  
**Verdict:** ⚠️ **Retain with documentation** — Most engaging social game but Medium leak surface. High Resonance gate (200) restricts to committed Specters. Document timing metadata risk clearly; consider timing obfuscation (random delays, batched message reveals) in future iterations.

---

### 7. Masked Events

**Implementation:** `pkg/anonymous/mechanics/masked_events.go`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | 30–240 minute duration; chat-style interaction with no real-time requirement |
| **1:1 vs. Group** | Group | 5–100 participants; ephemeral anonymous chat room |
| **Primary Factor** | Social | Anonymous discussion and content creation within event theme |
| **Anonymity Leak** | **Low** | Posting timing observable but ephemeral keypairs destroyed after event; interaction patterns exist but unlinkable to Specter identity post-event |

**Resonance Gating:** Phantom milestone (100)  
**Duration:** 30, 60, 120, or 240 minutes  
**Verdict:** ✅ **Retain** — Deepest anonymity mechanic (ephemeral keypairs) justifies Low leak surface; post-event unlinkability is strong mitigation.

---

### 8. Phantom Councils

**Implementation:** `pkg/anonymous/mechanics/councils/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | Persistent coordination groups; no real-time requirement for voting or discussion |
| **1:1 vs. Group** | Group | 3–13 members per council |
| **Primary Factor** | Social | Coordination, voting, and consensus-building among high-Resonance Specters |
| **Anonymity Leak** | **Low** | Membership public (visible admission votes); communication encrypted; interaction timing observable within council but limited to trusted members |

**Resonance Gating:** Fortress mode + minimum 200 Resonance for membership  
**Duration:** Persistent (30-day Wave retention)  
**Verdict:** ✅ **Retain** — Exclusive coordination mechanic for highest-trust participants; Low leak surface acceptable given unanimous admission requirement and encrypted communication.

---

### 9. Surface Sparks (Wave Relay / Echo Races)

**Implementation:** `pkg/anonymous/mechanics/sparks/`

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Sync | 5-minute response windows; Echo Races reward first amplifier with sub-second latency significance |
| **1:1 vs. Group** | Group | Open participation from connected Surface Layer nodes |
| **Primary Factor** | Chance + Social | Echo Races are speed-based (latency-sensitive); Wave Relays reward creative constraint matching |
| **Anonymity Leak** | **High** | **Echo Races leak latency fingerprints** — first-amplifier timing reveals sub-second network delays that could aid network-position inference |

**Resonance Gating:** None — available to all Surface Layer users  
**Surface Layer Only:** Not available on Anonymous Layer  
**Verdict:** ⚠️ **Surface Layer Only; Anonymity Incompatible** — Echo Races must remain Surface Layer exclusive due to High leak surface. If any variant is considered for Anonymous Layer in future, latency obfuscation (randomized delays, batched reveals) would be mandatory.

---

### 10. Whisper Chains (not a mini-game but private messaging mechanic)

**Implementation:** Described in ANONYMOUS_GAME_MECHANICS.md (not yet in codebase audit scope)

| Axis | Rating | Notes |
|------|--------|-------|
| **Sync vs. Async** | Async | Multi-hop message relay; no real-time requirement |
| **1:1 vs. Group** | 1:1 | Point-to-point private messaging through intermediate relays |
| **Primary Factor** | N/A | Communication primitive, not a competitive mechanic |
| **Anonymity Leak** | **Low** | Relay nodes see encrypted blobs and hop sequence; onion routing prevents origin/destination correlation; timing analysis possible but mitigated by multi-hop delays |

**Resonance Gating:** None — available to all Specters  
**Rate Limit:** 10 messages per 24 hours  
**Verdict:** ✅ **Retain** — Essential private communication primitive; onion routing provides strong anonymity; timing metadata acceptable given multi-hop obfuscation.

---

## Summary Table

| Game | Sync/Async | 1:1/Group | Primary Factor | Leak Surface | Resonance Gate | Verdict |
|------|------------|-----------|----------------|--------------|----------------|---------|
| Cipher Puzzles | Async | Group | Skill | **None** | 50 | ✅ Retain |
| Specter Hunts | Async | Group | Skill + Social | **Low** | 75 | ✅ Retain |
| Territory Drift | Async | Group | Social | **None** | 25 | ✅ Retain |
| Oracle Pools | Async | Group | Skill | **None** | 100 | ✅ Retain |
| Sigil Forge | Async | Group | Skill + Social | **None** | 50 | ✅ Retain |
| Shadow Play | Async | Group | Social | **Medium** | 200 | ⚠️ Retain + Document |
| Masked Events | Async | Group | Social | **Low** | 100 | ✅ Retain |
| Phantom Councils | Async | Group | Social | **Low** | 200 (Fortress) | ✅ Retain |
| Surface Sparks | **Sync** | Group | Chance + Social | **High** | None (Surface only) | ⚠️ Surface Only |
| Whisper Chains | Async | 1:1 | N/A (messaging) | **Low** | None | ✅ Retain |

---

## Analysis & Recommendations

### Anonymity/Fun Tradeoffs

**No cuts required.** All 10 mechanics are retention-positive and have acceptable or zero anonymity leak surfaces given their design constraints:

- **9 of 10 games are async** — eliminates real-time latency fingerprints.
- **2 of 10 have Medium or High leak surfaces:**
  - **Shadow Play (Medium):** Gated at Resonance 200; leak acceptable for highest-trust Specters. Recommend documenting timing metadata risk in user-facing descriptions.
  - **Surface Sparks (High):** Already Surface Layer exclusive; Echo Races incompatible with anonymity and must never migrate to Anonymous Layer.

### Signature Games (per PLAN.md §2.3)

**Recommendation:** Promote the following 3 games as flagship mechanics:

1. **Cipher Puzzles** — Zero-leak cryptographic challenges; accessible (Resonance 50); fast (15–60 min); shareable via puzzle announcements; tolerant of dropout (solutions stand alone). Defines MURMUR as intellectually engaging.

2. **Sigil Forge** — Zero-leak creative competition; accessible (Resonance 50); fast (30–60 min); produces visible artifacts (Sigil Art, Micro Fiction); audience participation via amplification. Defines MURMUR as creatively expressive.

3. **Shadow Play** — Deepest social mechanic; exclusive (Resonance 200); moderate duration (30–60 min); leverages anonymity as core game element (hidden roles). Defines MURMUR as socially sophisticated. Despite Medium leak surface, the game's uniqueness and high Resonance gate justify flagship status.

**Rationale:** These three cover the skill-social-creative spectrum, are all accessible within a single play session (<60 min), and represent MURMUR's unique value: anonymous intellectual/creative/social engagement.

### Metadata Leak Mitigation (Future Work)

For **Shadow Play** (Medium leak surface):

- **Timing obfuscation:** Introduce randomized delays (10–60 seconds) before revealing Waves during discussion rounds.
- **Batched reveals:** Hold messages for 30-second intervals and reveal all simultaneously per round.
- **Cover traffic:** Generate synthetic discussion Waves during voting rounds to mask real interaction patterns.

These mitigations can be deferred post-v1.0 if user testing shows no deanonymization concerns at Resonance 200 threshold.

For **Surface Sparks** (already Surface-only):

- Maintain strict isolation — no Echo Race variants on Anonymous Layer.
- If future demand for fast-paced Anonymous Layer games emerges, design from scratch with latency obfuscation as a first-order constraint (e.g., commit-then-reveal schemes with enforced minimum delays).

---

## Implementation Notes

### Game Module SDK (PLAN.md §2.4)

All 10 mechanics share common patterns suitable for SDK abstraction:

**Core API surface:**
- `CreateMatch(matchID, config) error` — Initialize game state
- `JoinMatch(matchID, participantKey) error` — Register participant
- `BroadcastEvent(matchID, eventType, payload) error` — Publish game-specific Wave
- `PersistState(matchID, state) error` — Save intermediate state to Bbolt
- `EndMatch(matchID, outcome) error` — Finalize and compute Resonance rewards
- `AwardResonance(participantKeys, bonusFormula) error` — Apply decaying bonus per spec formulas

**Sandboxing:**
- Games access only SDK primitives — no direct identity, network, or storage access.
- Resonance computation handled by SDK, not game logic.
- Wave signing/validation handled by SDK.

**Reference implementation:** Use **Cipher Puzzles** as reference game in SDK repo — simplest state machine, well-documented, zero-leak anonymity model.

### Per-Game Privacy Datasheets (PLAN.md §2.5)

Create `docs/privacy/GAME_[NAME]_PRIVACY.md` for each game with:

1. **Metadata Collected:** List all observable data (timing, interaction patterns, participation records)
2. **Anonymity Guarantees:** What the game promises (e.g., "No real-time latency fingerprints")
3. **Known Limitations:** What the game cannot hide (e.g., "Participation timing observable to network")
4. **Recommended Precautions:** User guidance (e.g., "Use Tor transport for Shadow Play if adversary can observe network timing")

Surface these datasheets in-app before first play: "This game observes [X]. You can mitigate by [Y]. Proceed?"

---

## Completion Status

✅ **PLAN.md §2.1 complete** — All 10 mini-games classified across 4 axes  
⏭ **Next:** PLAN.md §2.2 (cut games with poor anonymity/fun ratio) — **No cuts required; proceed to §2.3**  
⏭ **Next:** PLAN.md §2.3 (identify 2–3 signature games) — **3 flagship games identified above**  

**Task classification complete.** All 10 mechanics are retention-positive and meet MURMUR's anonymity standards.
