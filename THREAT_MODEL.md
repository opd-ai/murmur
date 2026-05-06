# MURMUR Threat Model Statement

## Overview

MURMUR is designed to provide **metadata unlinkability** for social communication and games over a peer-to-peer network. This document defines which adversaries and attack vectors are in scope for MURMUR's security guarantees, and which are explicitly delegated to external systems (Tor, I2P).

**Core Security Goal:** Network-level observers cannot reliably correlate IP addresses with user identities, social graphs, or content consumption patterns.

---

## Primary Adversary: Network-Level Metadata Observer

### Adversary Profile
- **Capability:** Passive observation of network traffic patterns (IP addresses, packet sizes, timing)
- **Goal:** Deanonymize users by correlating IP addresses with identities or social connections
- **Examples:**
  - ISPs logging connection metadata
  - Corporate network administrators monitoring employee traffic
  - Censorship regimes analyzing P2P protocol patterns
  - Academic researchers studying social graph leakage

### MURMUR Mitigations (In Scope)

1. **Shroud Onion Routing**
   - Three-hop circuits with layered encryption (XChaCha20-Poly1305)
   - Each relay sees only the previous and next hop, not origin or destination
   - Circuit rotation every 10 minutes to limit long-term correlation
   - Hop diversity: no two relays in the same /16 subnet or ASN

2. **Identity Layer Separation**
   - Surface Layer: Ed25519 keypair for attributed identity
   - Anonymous Layer: Curve25519 keypair for Specter pseudonyms
   - Zero cryptographic linkability between the two layers (no shared key material or derivation paths)

3. **GossipSub Probabilistic Routing**
   - Messages propagate through mesh topology with configurable fanout (D=6)
   - Each peer relays messages from multiple sources, obscuring origin
   - No centralized message broker that could aggregate traffic patterns

4. **Timing Obfuscation**
   - Random jitter (±500ms) on message forwarding to prevent timing correlation attacks
   - Shroud circuits batch messages into fixed-size cells (512 bytes) to prevent size-based fingerprinting

5. **No Persistent Identifiers**
   - No email addresses, phone numbers, or third-party OAuth
   - PeerIDs are derived from ephemeral libp2p keypairs, not user identities
   - Wave IDs use BLAKE3 content hashing (no sequential numbering)

### Limitations (Not Fully Mitigated)

- **Traffic confirmation attacks:** If an adversary controls both the entry and exit of a Shroud circuit, they may correlate timing patterns. Mitigated by circuit rotation and hop diversity, but not eliminated.
- **Intersection attacks:** Observing which peers are online simultaneously over long periods can reveal social graphs. Mitigated by ephemeral connections and relay usage, but not eliminated.
- **Sybil attacks:** An adversary controlling a large fraction of relays can increase probability of circuit compromise. Mitigated by peer scoring and relay reputation (Resonance), but requires honest majority assumption.

**Recommendation for users requiring stronger guarantees:** Route MURMUR traffic through Tor or I2P (see "Integration with External Anonymity Networks" below).

---

## Secondary Adversary: Malicious Peers / Griefers

### Adversary Profile
- **Capability:** Participation in the MURMUR network as a peer
- **Goal:** Spam, flood, disrupt service, or infer metadata about other users
- **Examples:**
  - Botnet operators flooding the network with fake Waves
  - Malicious relays attempting to correlate Shroud circuit traffic
  - Griefers disrupting mini-games or harassing users
  - Researchers probing the network for deanonymization vulnerabilities

### MURMUR Mitigations (In Scope)

1. **Proof of Work (PoW) on Wave Publication**
   - SHA-256 PoW with difficulty 20 (default 2–5 seconds on consumer hardware)
   - Prevents low-cost spam; attackers must expend CPU cycles for each message
   - Difficulty adjustable per privacy mode (Fortress requires higher PoW)

2. **Resonance-Based Rate Limiting**
   - Low-Resonance Specters face stricter rate limits (1 Wave/10 minutes at Resonance < 25)
   - Milestone-based unlocks (Shade=25, Wraith=50, Phantom=100, Council=200)
   - Sybil resistance: new accounts cannot immediately flood the network

3. **Peer Scoring and Penalization**
   - GossipSub peer scoring tracks message validity, forwarding behavior, and latency
   - Shroud relay failure tracking penalizes unreliable or malicious relays for 1 hour
   - Nodes maintain local blocklists; repeated misbehavior results in disconnection

4. **Per-Peer Connection Limits**
   - Maximum 200 simultaneous connections (prevents resource exhaustion)
   - Four-tier priority: Social (trusted contacts), Mesh (good standing), DHT (bootstrap), Opportunistic (unauthenticated)
   - Low-priority connections evicted under resource pressure

5. **Bloom Filter Deduplication**
   - 100k-entry Bloom filter (0.1% false positive rate) prevents replay attacks
   - Each Wave ID (BLAKE3 hash) checked before propagation
   - Filter rotates every 24 hours to prevent memory exhaustion

6. **TTL Enforcement**
   - All Waves expire after configurable TTL (default 7 days, max 30 days)
   - Expired content not forwarded; garbage collection removes from local storage
   - Prevents long-term message accumulation attacks

### Limitations (Not Fully Mitigated)

- **Eclipse attacks:** If an adversary surrounds a victim's node with malicious peers, they can partition the victim from the honest network. Mitigated by DHT diversity and peer exchange, but requires victim to have at least one honest connection.
- **Targeted harassment:** Adversaries can still send unwanted messages to a specific user. Mitigated by blocking and muting at identity layer, but requires reactive user action.
- **Game-specific griefing:** Mini-games may have domain-specific abuse vectors (quitting, cheating, slur injection). Addressed per-game via the Game Module SDK sandbox model.

---

## Platform-Style Deanonymization (Mitigated by Design)

### Attack Vector
- **Traditional social platforms** correlate identities via:
  - Account metadata (email, phone, real name)
  - Analytics tracking (IP addresses, browser fingerprints, usage patterns)
  - Third-party embeds (tracking pixels, OAuth providers)

### MURMUR Design Eliminates These Vectors

1. **No Account Registration**
   - Self-sovereign Ed25519 keypairs; no email, phone, or centralized registration
   - Cannot be deanonymized via "forgot password" flows or account recovery attacks

2. **No Analytics or Telemetry**
   - No usage tracking, no A/B testing, no behavioral profiling
   - Prometheus metrics (optional) are localhost-only; never sent to external servers

3. **No Third-Party Dependencies**
   - No OAuth providers, no CDN embeds, no external fonts or assets
   - All resources bundled via `go:embed`; zero external HTTP requests during runtime

4. **No Centralized Servers**
   - P2P architecture eliminates single point of metadata aggregation
   - Even relay operators see only encrypted Shroud traffic, not user identities

---

## Explicitly OUT OF SCOPE

### Adversaries MURMUR Does NOT Defend Against

1. **Global Passive Adversary (GPA)**
   - **Definition:** Entity that can observe all network traffic globally (NSA-level surveillance)
   - **Why out of scope:** Defending against GPA requires cover traffic, traffic morphing, and long-lived anonymity sets beyond MURMUR's resource model
   - **Mitigation:** Delegate to Tor or I2P (see below)

2. **State-Level Traffic Analysis**
   - **Definition:** Nation-state actors with jurisdiction over ISPs and ability to compel traffic logs
   - **Why out of scope:** Requires legal protections and infrastructure beyond MURMUR's control
   - **Mitigation:** Delegate to Tor or I2P

3. **Adversary Controlling Majority of Relays**
   - **Definition:** Entity that operates >50% of Shroud relays and can statistically deanonymize users
   - **Why out of scope:** Honest majority assumption is required for P2P anonymity networks
   - **Mitigation:** Relay reputation (Resonance), geographic/ASN diversity, and Tor/I2P integration

4. **Endpoint Compromise**
   - **Definition:** Malware, keyloggers, or physical access to victim's device
   - **Why out of scope:** No network protocol can defend against compromised endpoints
   - **Mitigation:** User education, full-disk encryption, secure boot (outside MURMUR's scope)

5. **Side-Channel Attacks (Timing, Power, Cache)**
   - **Definition:** Cryptographic side-channels leaking key material
   - **Why out of scope:** Requires specialized hardware countermeasures (constant-time implementations, hardware enclaves)
   - **Mitigation:** Rely on audited cryptographic libraries (`golang.org/x/crypto`, `filippo.io/*`)

---

## Integration with External Anonymity Networks

### Tor and I2P Transport Modes

MURMUR provides **pluggable transport adapters** for users whose threat model exceeds Shroud's guarantees:

#### Mode A: Shroud Only (Default)
- **Protection:** Network-level metadata observer cannot correlate IPs with identities (within honest majority assumption)
- **Tradeoff:** Faster latency (~100–300ms per hop), higher throughput
- **Suitable for:** Most users concerned about ISP/corporate surveillance but not state-level adversaries

#### Mode B: Shroud over Tor
- **Protection:** All outbound MURMUR traffic routed through Tor hidden services; Shroud circuits constructed over Tor
- **Tradeoff:** Higher latency (~500ms–2s), lower throughput, requires Tor daemon
- **Suitable for:** Users facing state-level censorship or passive global adversaries

#### Mode C: Shroud over I2P
- **Protection:** All traffic routed through I2P garlic routing; Shroud circuits constructed over I2P destinations
- **Tradeoff:** Higher latency (~1–3s), requires I2P router, lower peer reachability
- **Suitable for:** Users requiring maximum unlinkability and willing to sacrifice performance

#### Mode D: Hybrid (Tor + I2P)
- **Protection:** Both Tor and I2P adapters registered; peers reachable via either
- **Tradeoff:** Maximum compatibility at cost of complexity
- **Suitable for:** Expert users running multi-network infrastructure

### User Education Requirement

**MURMUR MUST surface plain-language tradeoffs before users commit to a mode:**

- **Shroud alone:** "Your IP is hidden from other users, but a global network observer could correlate patterns if they control many relays. Good for everyday privacy."
- **Shroud over Tor:** "Your traffic is routed through Tor before Shroud. Protects against state-level surveillance. Slower, but stronger."
- **Shroud over I2P:** "Your traffic is routed through I2P. Maximum unlinkability. Slowest, but most private."

**Users who require protection against state-level adversaries MUST be informed that Shroud alone is insufficient.**

---

## Summary Table

| Threat | In Scope | MURMUR Mitigation | Out of Scope | Recommendation |
|--------|----------|-------------------|--------------|----------------|
| **Network metadata observer** | ✅ | Shroud onion routing, identity separation | Global passive adversary | Route via Tor/I2P |
| **Malicious peers (spam, flood)** | ✅ | PoW, Resonance rate limiting, peer scoring | — | Use default mitigations |
| **Malicious relays** | ✅ | Hop diversity, circuit rotation, relay reputation | Adversary controlling >50% relays | Honest majority assumption |
| **Platform deanonymization** | ✅ | No accounts, no analytics, no third-party dependencies | — | Design eliminates vector |
| **State-level traffic analysis** | ❌ | — | ✅ | Route via Tor/I2P |
| **Endpoint compromise** | ❌ | — | ✅ | User responsibility (FDE, secure boot) |
| **Cryptographic side-channels** | ❌ | — | ✅ | Rely on audited libraries |

---

## Responsible Disclosure

MURMUR welcomes security research. If you discover a vulnerability:

1. **Email security contact:** (TBD — to be established before public release)
2. **Provide:** Description, steps to reproduce, impact assessment, suggested fix (if any)
3. **Expected response time:** 72 hours for acknowledgment, 30 days for patch
4. **Public disclosure:** Coordinated disclosure after patch is available (or 90 days, whichever comes first)

**Bug bounty:** No formal program at this time. Acknowledged researchers listed in SECURITY_ACKNOWLEDGMENTS.md.

---

*Last updated: 2026-05-06*  
*Supercedes:** SECURITY_PRIVACY.md (superseded by this threat model statement; to be updated for consistency)*
