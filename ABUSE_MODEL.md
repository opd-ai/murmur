# MURMUR Abuse Model & Response Framework

This document enumerates abuse categories, maps each to mitigation mechanisms, defines the ZK-Resonance-based progressive trust model, and establishes a host rights framework that preserves user anonymity.

---

## 1. Abuse Categories

MURMUR's dual-layer architecture (Surface + Anonymous), peer-to-peer topology, and planned tunneling feature create distinct abuse surfaces. Each category requires specific mitigations that preserve the network's privacy guarantees.

### 1.1 Spam & Flooding

**Definition**: Automated high-volume message/Wave publication intended to congest the network, exhaust peer resources, or bury legitimate content.

**Attack Vectors**:
- **Surface Spam**: Botnet publishes thousands of Surface Waves per minute across multiple identities
- **Specter Spam**: Automated Specter creation + Wave publication to evade identity-based blocks
- **Gossip Flooding**: Malicious peer re-publishes the same Wave repeatedly to saturate GossipSub mesh
- **DHT Pollution**: Injecting garbage keys into Kademlia DHT to degrade routing table quality

**Risk Level**: **HIGH** — Decentralized networks have no single gatekeeper; spam directly degrades user experience and network performance.

**Active Mitigations**:
- SHA-256 Proof of Work (difficulty 20) on all Waves: 2–5 seconds per Wave (see `pkg/content/pow/`)
- GossipSub peer scoring: low-scoring peers pruned from mesh (see `pkg/networking/gossip/`)
- Bloom filter deduplication: duplicate Waves rejected at ingress (see `pkg/content/storage/`)
- Per-peer rate limiting: max 10 Waves/minute from single peer (see `pkg/networking/mesh/`)

**Planned Enhancements** (v1.1+):
- Adaptive PoW difficulty: increase to 22–24 for Surface identities with <7 days history
- Specter Resonance gates: new Specters (Resonance <25) face higher PoW (difficulty 24) and lower Wave visibility (1-hop propagation only)
- DHT signature verification: require Ed25519 signatures on DHT provider records

---

### 1.2 Denial of Service (DoS)

**Definition**: Attacks targeting node availability, resource exhaustion, or protocol-level vulnerabilities to disrupt individual peers or the entire network.

**Attack Vectors**:
- **Connection Exhaustion**: Opening thousands of libp2p connections to a single peer to exhaust file descriptors
- **Shroud Circuit Abuse**: Requesting hundreds of Shroud circuits from a relay to exhaust memory
- **PoW Verification Bomb**: Submitting Waves with incorrect PoW nonces to force expensive verification attempts
- **Pulse Map Rendering DoS**: Publishing Waves with crafted sigils to trigger pathological rendering paths

**Risk Level**: **MEDIUM** — Peer-to-peer topology provides resilience (no single point of failure), but individual nodes remain vulnerable.

**Active Mitigations**:
- libp2p connection limits: max 500 inbound, 50 outbound connections (see `pkg/networking/transport/`)
- Shroud circuit quotas: max 10 concurrent circuits per peer, max 3 new circuits/minute (see `pkg/anonymous/shroud/`)
- PoW nonce validation: fast rejection for nonces >2^32 or invalid Wave structure before expensive SHA-256 (see `pkg/content/pow/`)
- Pulse Map node culling: limit visible nodes to 2,000, LOD system for distant nodes (see `PULSE_MAP_DEGRADATION_CURVE.md`)

**Planned Enhancements** (v1.1+):
- Connection rate limiting: max 10 new connections/minute from same /24 subnet
- Shroud circuit blacklisting: peers causing >3 circuit failures in 10 minutes blacklisted for 1 hour
- Resource accounting: Prometheus metrics for CPU/memory/bandwidth per peer; auto-disconnect outliers

---

### 1.3 Harassment & Targeted Abuse

**Definition**: Persistent unwanted contact, threats, doxxing attempts, or coordinated harassment campaigns against specific users.

**Attack Vectors**:
- **Direct Message Harassment**: Sending unsolicited threatening/abusive Waves to a victim's identity
- **Specter Stalking**: Creating multiple Specters to circumvent blocks and continue harassment
- **Doxxing via Correlation**: Cross-referencing Surface + Specter activity to de-anonymize victims
- **Coordinated Brigading**: Multiple accounts amplifying abusive content or spamming victim's connections

**Risk Level**: **HIGH** — Harassment is a primary threat to user retention and community health; decentralization makes moderation harder.

**Active Mitigations**:
- Identity-level blocking: users can block Surface identities; blocked peers' Waves invisible (see `pkg/identity/`)
- Mute lists: temporary mute (7/30 days) without permanent block
- Connection filtering: privacy modes (Guarded/Fortress) reject Waves from non-connection identities
- Resonance visibility: Fortress-mode users only see Waves from connections or high-Resonance (≥100) Specters

**Planned Enhancements** (v1.1+):
- Specter blocking: allow blocking Specter identities (requires Resonance ≥50 to prevent evasion via new Specters)
- Shared block lists: users can export/import block lists from trusted connections (opt-in)
- Harassment detection heuristics: flag identities sending >20 Waves/day to blocked users; surface warnings to victim's connections
- DHT-based reputation anchors: publish encrypted block lists to DHT for cross-device sync

---

### 1.4 Game Griefing

**Definition**: Intentional disruption of anonymous mini-games (Cipher Puzzles, Sigil Forge, Shadow Play, etc.) to degrade experience for legitimate players.

**Attack Vectors**:
- **Rage Quitting**: Joining games, then immediately disconnecting to waste opponents' time
- **Cheating**: Exploiting game logic bugs (e.g., submitting pre-computed Cipher Puzzle solutions)
- **Slur/Toxicity Injection**: Submitting offensive text in Sigil Forge submissions or Shadow Play chat
- **Sybil Gaming**: Creating multiple Specters to collude in social deduction games (Shadow Play, Oracle Pools)

**Risk Level**: **MEDIUM** — Game griefing degrades user experience but doesn't threaten network infrastructure; Resonance gates limit impact.

**Active Mitigations**:
- Resonance entry barriers: most games require Resonance ≥25; Shadow Play requires ≥200 (see `pkg/anonymous/mechanics/`)
- Quit penalties: early disconnection from games reduces Resonance by 5 points
- Content filtering: UTF-8 validation, max length enforcement, profanity filter (opt-in) on Sigil Forge submissions
- Game state validation: server-side (Specter-coordinated) validation of game moves to detect cheating

**Planned Enhancements** (v1.1+):
- Player reputation: per-game ratings visible to matchmaking; low-rated players matched together
- Time commitment bonds: games require upfront time commitment (e.g., "I can play for 30 minutes"); early quit forfeits bond
- Sybil resistance: Shadow Play uses timing analysis to detect correlated Specters (same jitter patterns); flag suspicious groups

---

### 1.5 Tunnel Abuse (Post-Tunneling Feature Launch)

**Definition**: Malicious use of MURMUR's tunneling primitive (PageKite/ngrok-style HTTP tunneling over Shroud circuits) for illegal or harmful purposes.

**Attack Vectors**:
- **Malware C2**: Using tunnels to exfiltrate data from compromised hosts or deliver command-and-control traffic
- **Phishing**: Hosting phishing sites behind MURMUR tunnels to evade domain-based blocklists
- **CSAM Distribution**: Hosting illegal content via tunnels to evade takedown
- **Network Scanning**: Tunneling port scans or exploit attempts to hide attacker's real IP

**Risk Level**: **CRITICAL** — Tunnel abuse exposes relay operators and the project to legal liability; must be addressed before tunneling launches.

**Planned Mitigations** (Phase 6, before tunnel feature launch):
- **Content-Type Allowlists**: Tunnel operators can restrict allowed MIME types (e.g., `text/html`, `application/json` only); default-deny for executables
- **Hostname Allowlists**: Operators can restrict tunneled traffic to specific destination domains (e.g., only `*.example.com`)
- **Bandwidth Accounting**: Per-tunnel quotas (e.g., 1 GB/day); exceeded quotas trigger auto-teardown
- **Automated Takedown Protocol**: Relay operators can refuse traffic matching abuse signatures (e.g., malware C2 patterns) without deanonymizing the initiator
- **Exit Operator Opt-In**: Tunneling is **disabled by default**; operators must explicitly opt in and accept legal risk
- **Abuse Reporting Channel**: Users can report abusive tunnels; reports forwarded to exit operator with anonymity preserved

**Host Rights** (see §3):
- Operators MAY refuse tunnel traffic based on content-type, destination, bandwidth, or abuse signatures
- Operators MUST NOT attempt to de-anonymize tunnel initiators
- Operators MUST publish their refusal policies in a machine-readable format (`tunnel-policy.json`)

---

## 2. Mitigation Lever Mapping

Each abuse category maps to one or more mitigation mechanisms. This matrix guides implementation priorities.

| Abuse Category | Primary Mitigations | Secondary Mitigations | Residual Risks |
|----------------|---------------------|------------------------|----------------|
| **Spam & Flooding** | PoW (difficulty 20–24), GossipSub peer scoring, Bloom deduplication, Per-peer rate limits | Adaptive PoW, Resonance gates, DHT signatures | Distributed botnets can still aggregate spam across many peers |
| **Denial of Service** | Connection limits (500 in/50 out), Shroud circuit quotas (10 concurrent/peer), PoW nonce fast-fail | Connection rate limits, Circuit blacklisting, Resource accounting | Coordinated DDoS from 1000+ peers can still degrade performance |
| **Harassment** | Identity blocking, Mute lists, Privacy modes (Guarded/Fortress), Resonance visibility | Specter blocking, Shared block lists, Harassment heuristics | Persistent attackers can rotate identities faster than victims block |
| **Game Griefing** | Resonance gates (25/200), Quit penalties (-5 Resonance), Content filtering, Game state validation | Player reputation, Time commitment bonds, Sybil detection | Dedicated griefers with high Resonance can still disrupt |
| **Tunnel Abuse** | Content-Type/Hostname allowlists, Bandwidth quotas, Takedown protocol, Operator opt-in | Exit operator blocklists, Abuse reporting, Legal compliance framework | Operators face legal risk even with best-effort controls |

**Design Principle**: Layered defense. No single mitigation is sufficient; each layer reduces attack surface and increases attacker cost.

---

## 3. ZK-Resonance-Based Progressive Trust

Resonance (anonymous reputation) is MURMUR's primary mechanism for establishing trust without identity linkage. Zero-knowledge proofs allow Specters to prove Resonance thresholds without revealing their exact score or interaction history.

### 3.1 Resonance Tiers & Capabilities

| Tier | Threshold | Capabilities Unlocked | Abuse Mitigation |
|------|-----------|----------------------|------------------|
| **Shade** | 25 | Basic games (Cipher Puzzles), 3-hop Shroud circuits, standard PoW (difficulty 20) | Minimal — new Specters expected to participate cautiously |
| **Wraith** | 50 | Sigil Forge, Territory Drift, Specter Hunts, can block other Specters | Moderate — blocking capability limits harassment evasion |
| **Shade-Wraith** | 75 | Oracle Pools, Shadow Play (observer only), gift exchange | Higher — social mechanics require demonstrated history |
| **Phantom** | 100 | Shadow Play (full participation), Fortress-mode visibility (can reach Fortress users) | High — only trusted Specters reach Fortress users |
| **Council-Eligible** | 200 | Phantom Councils (governance), reduced PoW (difficulty 18), 4-hop Shroud circuits | Very High — governance participation reserved for established Specters |
| **Abyss** | 500 | Fortress-mode only; no additional mechanics (prestige tier) | Maximum — only highest-reputation Specters qualify |

**Progression Rate**: Designed for ~4 weeks (28 days) of normal activity to reach Phantom (100), ~12 weeks to reach Council-Eligible (200). Prevents throwaway accounts while allowing legitimate new users to participate.

### 3.2 Low-Resonance Restrictions

To mitigate abuse from new or throwaway Specters:

**Resonance <25 (Unranked)**:
- PoW difficulty **24** (4x harder: 8–20 seconds/Wave)
- Shroud circuits limited to **2-hop** (reduces anonymity set; incentivizes progression)
- Waves visible only to **direct connections** (1-hop propagation; no gossip beyond mesh)
- Cannot participate in any mini-games
- Rate limit: **5 Waves/hour** (vs 10/hour for Shade+)

**Resonance 25–49 (Shade)**:
- PoW difficulty **22** (2x harder: 4–10 seconds/Wave)
- Shroud circuits **3-hop** (standard anonymity)
- Waves propagate **2 hops** (limited viral reach)
- Basic games only (Cipher Puzzles)
- Rate limit: **10 Waves/hour**

**Rationale**: Progressive relaxation of restrictions as Resonance increases aligns incentives: legitimate users progress naturally through participation; spammers face exponentially increasing costs to create throwaway accounts with high Resonance.

### 3.3 Zero-Knowledge Resonance Proofs

**Problem**: Specters must prove Resonance ≥N to unlock capabilities, but revealing exact Resonance scores enables correlation attacks (e.g., "this Specter has exactly 127 Resonance" narrows the anonymity set).

**Solution**: Bulletproofs-based range proofs using Pedersen commitments over the Ristretto255 group (see `pkg/security/zkp/`).

**Protocol**:
1. Specter computes Resonance locally (see `pkg/anonymous/resonance/`)
2. Specter generates Pedersen commitment `C = rG + vH` where `v` is Resonance, `r` is random blinding factor
3. Specter generates Bulletproof that `v ≥ threshold` (e.g., 100 for Phantom)
4. Verifier checks proof validity without learning `v`

**Proof Size**: ~700 bytes for single range proof; ~1.2 KB for aggregated proof (multiple thresholds in one proof).

**Verification Time**: ~5 ms per proof on modern CPUs.

**Security**: Computational soundness under discrete log assumption; zero-knowledge under random oracle model. Proofs do not leak Resonance value, interaction history, or identity across sessions.

**Implementation Status**: ZK proof infrastructure complete (see `pkg/security/zkp/bulletproof.go`); integration with Resonance thresholds in game mechanics implemented (see `pkg/anonymous/mechanics/`).

---

## 4. Abuse-Response Model That Preserves Anonymity

MURMUR's architecture must allow relay operators and peers to refuse abusive traffic **without de-anonymizing the source**. This requires protocol-level support for refusal reasons and threat-pattern detection that operates on encrypted traffic.

### 4.1 Host Rights Framework

Relay operators (Shroud circuit hops, tunnel exits) have the following rights and responsibilities:

**Operators MAY**:
- Refuse connections based on resource limits (e.g., "I'm at 500 concurrent circuits; try another relay")
- Refuse traffic based on published content-type, destination, or bandwidth policies
- Refuse traffic matching known abuse signatures (e.g., malware C2 patterns detected via timing analysis)
- Refuse to relay Waves from peers with consistently low GossipSub scores
- Refuse to participate in the network at all (no operator is compelled to run a relay)

**Operators MUST NOT**:
- Attempt to de-anonymize traffic sources (e.g., logging circuit initiator IPs beyond what's necessary for refusal)
- Collude with other operators to correlate traffic across hops
- Selectively refuse traffic to censor specific content (beyond published policies)
- Refuse traffic based on the identity of the ultimate destination (except for tunnel hostname allowlists)

**Operators MUST**:
- Publish refusal policies in machine-readable `host-policy.json` format (see §4.3)
- Provide refusal reasons to initiators (e.g., "Bandwidth quota exceeded" vs "Content-type not allowed")
- Delete logs of refused traffic after 7 days (except aggregated metrics for abuse reporting)

**Rationale**: Operators need tools to protect themselves from legal liability and resource abuse, but must not become censors or de-anonymizers. Published policies allow initiators to route around restrictive relays.

### 4.2 Abuse Signature Detection (Non-Deanonymizing)

**Challenge**: How can a relay detect malware C2 traffic without decrypting Shroud layers or inspecting packet contents?

**Approach**: Timing and volume analysis at the relay hop only.

**Example — Malware C2 Detection**:
- Typical C2 traffic: periodic beacons (e.g., every 60 seconds ±5%) with small payload (~200 bytes)
- Legitimate traffic: bursty, variable inter-arrival times, diverse payload sizes
- **Detection heuristic**: If a circuit sends 10+ requests with inter-arrival times within 5% of mean **and** payload sizes within 20% of mean over 10 minutes → flag as "potential C2"
- **Response**: Relay adds jitter (+0–500ms) to requests, or refuses circuit with reason "Traffic pattern violated host policy"

**Example — Bandwidth Abuse Detection**:
- Relay tracks per-circuit bandwidth: if circuit exceeds 1 GB/hour for 3 consecutive hours → flag as "high-bandwidth"
- **Response**: Relay refuses circuit with reason "Bandwidth quota exceeded; try another relay"

**Crucially**: Detection operates on **encrypted traffic metadata** (timing, volume) only. Relay never learns plaintext content, initiator identity, or ultimate destination.

### 4.3 Machine-Readable Host Policies

Each relay operator publishes a `host-policy.json` file via DHT and gossip:

```json
{
  "version": 1,
  "relay_id": "ed25519:ABCD1234...",
  "policies": {
    "max_concurrent_circuits": 100,
    "max_circuit_duration": "24h",
    "max_bandwidth_per_circuit": "1GB/hour",
    "allowed_tunnel_content_types": ["text/html", "application/json"],
    "allowed_tunnel_destinations": ["*.example.com"],
    "abuse_detection": {
      "periodic_traffic_threshold": 0.05,
      "bandwidth_high_water_mark": "1GB/hour"
    }
  },
  "refusal_reasons": [
    "Resource limit exceeded",
    "Bandwidth quota exceeded",
    "Content-type not allowed",
    "Destination not allowed",
    "Traffic pattern violated policy"
  ],
  "contact": "abuse@example.com",
  "last_updated": "2026-05-06T05:00:00Z"
}
```

**Usage**:
- Circuit initiators fetch `host-policy.json` from candidate relays via DHT before constructing circuits
- Initiators can avoid relays with restrictive policies (e.g., no tunneling, low bandwidth quotas)
- Verifiers can audit operator policies for censorship (e.g., "Operator X refuses all traffic to Tor exit nodes")

---

## 5. Integration with SECURITY_PRIVACY.md

This abuse model complements the threat model in `SECURITY_PRIVACY.md` (which focuses on cryptographic adversaries and metadata observers) by addressing **application-layer abuse** and **social threats**.

**Cross-References**:
- `SECURITY_PRIVACY.md §2.1` (Passive Network Observer) — Shroud mitigates IP correlation; abuse model adds circuit refusal without de-anonymization
- `SECURITY_PRIVACY.md §2.2` (Malicious Peer) — GossipSub peer scoring and PoW mitigate spam; abuse model adds Resonance gates
- `SECURITY_PRIVACY.md §3.1` (Cryptographic Primitives) — ZK Resonance proofs use Bulletproofs + Pedersen commitments; abuse model specifies thresholds
- `SECURITY_PRIVACY.md §5` (Privacy Modes) — Fortress mode relies on Resonance ≥100 for Specter visibility; abuse model defines progression

**Additions to SECURITY_PRIVACY.md** (to be integrated):
- New §6: "Application-Layer Abuse Mitigations" summarizing PoW, Resonance gates, peer scoring, and progressive trust
- New §7: "Host Rights & Anonymity-Preserving Refusal" defining operator rights and machine-readable policies
- Update §2.2 (Malicious Peer) to reference abuse categories from this document

---

## 6. Roadmap & Open Questions

### 6.1 Pre-Launch Requirements (v1.0)

**MUST HAVE**:
- [x] PoW (difficulty 20) on all Waves
- [x] GossipSub peer scoring + pruning
- [x] Bloom filter deduplication
- [x] Identity-level blocking + mute lists
- [x] Resonance gates on mini-games (25/50/100/200)
- [x] ZK Resonance proofs (Bulletproofs)
- [ ] Low-Resonance restrictions (PoW 24, 1-hop propagation for <25 Resonance) — **IMPLEMENTATION REQUIRED**
- [ ] Shroud circuit quotas (10 concurrent, 3/minute rate limit) — **IMPLEMENTATION REQUIRED**
- [ ] Host policy publishing (JSON format + DHT distribution) — **DESIGN COMPLETE, IMPLEMENTATION PENDING**

**NICE TO HAVE** (v1.0):
- Specter blocking (requires Resonance ≥50)
- Shared block lists (opt-in)
- Harassment detection heuristics

### 6.2 Tunnel Abuse Mitigations (Phase 6, v1.1+)

**MUST HAVE BEFORE TUNNELING LAUNCHES**:
- [ ] Content-Type/Hostname allowlists in tunnel configuration
- [ ] Per-tunnel bandwidth quotas + accounting
- [ ] Automated takedown protocol (operator can refuse traffic by signature without de-anonymization)
- [ ] Exit operator opt-in (tunneling disabled by default)
- [ ] Legal compliance framework documentation

**OPEN QUESTIONS**:
1. **Liability Shield**: Can relay operators use "common carrier" defense if they implement best-effort abuse controls? (Requires legal review per jurisdiction)
2. **Abuse Reporting**: Should MURMUR provide a centralized abuse reporting system, or rely on per-operator reporting? (Centralization conflicts with decentralization philosophy)
3. **CSAM Detection**: Can we integrate PhotoDNA or perceptual hashing without compromising end-to-end encryption? (Likely requires exit-hop plaintext inspection, which breaks anonymity)

### 6.3 Long-Term Enhancements (v1.2+)

- **Reputation-Weighted PoW**: High-Resonance Specters face lower PoW (difficulty 16–18); low-Resonance face higher (24+)
- **Federated Block Lists**: Trusted community moderators publish signed block lists; users can subscribe (opt-in)
- **Game-Specific Reputation**: Separate reputation scores per mini-game (Cipher Puzzles, Shadow Play, etc.) to isolate griefing impact
- **Sybil-Resistant Resonance**: Require proof-of-personhood (e.g., CAPTCHA, social graph analysis) for Resonance >200 to prevent botnet Council infiltration

---

## 7. Conclusion

MURMUR's abuse model balances **user safety**, **network health**, and **anonymity preservation** through layered defense:

1. **Technical barriers**: PoW, rate limits, resource quotas raise attacker costs
2. **Reputation systems**: Resonance gates and ZK proofs enable trust without identity linkage
3. **User controls**: Blocking, muting, privacy modes empower victims
4. **Operator rights**: Host policies and refusal mechanisms protect relay operators without de-anonymization
5. **Progressive disclosure**: Complexity revealed gradually; new users face restrictions, established users gain freedom

**No system is abuse-proof**, especially in decentralized networks. This framework provides **necessary but not sufficient** protections. Long-term success requires:

- **Community norms**: Social pressure against griefing, amplified through Resonance penalties
- **Ongoing iteration**: Abuse tactics evolve; mitigations must adapt (quarterly reviews per PLAN.md §X.2)
- **Legal preparedness**: Clear documentation of operator rights, abuse response protocols, and jurisdictional compliance strategies

This document will be updated as new abuse vectors emerge and mitigation strategies are validated through deployment.
