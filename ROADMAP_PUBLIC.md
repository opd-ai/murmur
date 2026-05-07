# MURMUR Public Roadmap

A transparent, community-facing view of MURMUR's development priorities and timeline.

---

## Mission Statement

MURMUR is a decentralized peer-to-peer social network delivering anonymous, playful communication with metadata unlinkability as a foundational primitive. This roadmap shows what's complete, in progress, and planned.

---

## Current Status: v0.1 Foundation (May 2026)

**Core Features**: ✅ 85-90% complete

| Subsystem | Status | Key Features |
|-----------|--------|--------------|
| **Networking** | ✅ Complete | libp2p transport, GossipSub v1.1, Kademlia DHT, NAT traversal |
| **Identity** | ✅ Complete | Ed25519/Curve25519 keypairs, BIP-39 recovery, Argon2id encryption, visual sigils |
| **Content** | ✅ Complete | 8 Wave types, SHA-256 PoW, TTL enforcement, threading |
| **Anonymous Layer** | ✅ Complete | Specters, 3-hop Shroud circuits, Resonance reputation, 10 mini-games |
| **Pulse Map** | ✅ Complete | Force-directed layout (60fps @ 500 nodes), Ebitengine rendering, visual effects |
| **Storage** | ✅ Complete | Bbolt persistence, typed accessors, LRU eviction |
| **Cryptography** | ✅ Complete | Ed25519, X25519, XChaCha20-Poly1305, BLAKE3, PoW, ZK proofs |
| **Onboarding** | ✅ Complete | 6-phase sequence: Welcome → Identity → Mode → Bootstrap → Exploration → First Wave |
| **Security** | ✅ Complete | Key zeroing, Bloom filter dedup, peer scoring, ZK Resonance proofs |

---

## Released Features (May 2026)

### Phase 0–4: Foundation & UX (Weeks 1–16)

- ✅ **Product Identity** defined (PRODUCT_IDENTITY.md)
- ✅ **Threat Model** published (THREAT_MODEL.md)
- ✅ **Extension Contract** (EXTENSION_CONTRACT.md)
- ✅ **Pulse Map role decision** locked (primary interface)
- ✅ **Messaging-first prototype** designed (optional A/B test)
- ✅ **Game library** audited (10 games, 3 signature titles)
- ✅ **Game SDK** published with API contract
- ✅ **Identity recovery** audit complete (multi-device, social recovery, key rotation designed)
- ✅ **Anti-abuse framework** (5 categories, progressive trust, abuse-response model)

### Phase 5: Tor/I2P Transport (Weeks 12–18)

- ✅ **go-i2p/onramp dependency** added
- ✅ **Tor transport adapter** (libp2p integration)
- ✅ **I2P transport adapter** (libp2p integration)
- ✅ **Both adapters** registered in host builder
- ✅ **4 transport modes** documented (Default, Tor, I2P, Both)
- ✅ **Reachability diagnostics** (fail-fast if daemons unreachable)
- ✅ **Interop testing** (modes coexist, Shroud-over-Tor works)
- ✅ **TRANSPORT_ANONYMITY.md** (20KB technical guide)

### Phase 6: Tunneling Primitive (Weeks 16–26)

- ✅ **Tunnel design** complete (TUNNEL_DESIGN.md)
- ✅ **Abuse policy** locked before implementation (TUNNEL_ABUSE_POLICY.md)
- ✅ **Minimal tunnel prototype** (HTTP, single-hop validation)
- ✅ **Multi-hop circuits** over Shroud (6-hop operator/relay/client paths)
- ✅ **Operator incentive model** (TSC → bounded Resonance)
- ✅ **Tunnel host profile** documentation (relay-only, exit-enabled configs)
- ✅ **Protocol & accounting** (bandwidth limits, quota enforcement)

### Phase 7: Friend-to-Friend Reseed (Weeks 20–30)

- ✅ **Reseed semantics** defined (capability grants, signed bundles)
- ✅ **Reseed as tunnel app** (reuse circuit infrastructure)
- ✅ **Out-of-band invitation codes** (Ed25519-signed, URI-encoded)
- ✅ **Bootstrap fallback** integration (multi-address support)
- ✅ **RESEED.md** user guide (threat model, mitigations)

### Phase 8: Extension Surface Hardening

- ✅ **Stable extension points** frozen (Wave types, Game SDK, Transport, Resonance hooks)
- ✅ **Reference extension** published (WordSpark game module)
- ✅ **PROTOCOL.md** (25KB wire-format spec for external implementers)
- ✅ **MEP process** established (MEPs/ folder, template, example)

### Planning & Documentation (Ongoing)

- ✅ **DECISIONS.md** (decision log with revisit dates)
- ✅ **Q2 2026 threat model review** (security audit, no blockers)

---

## In Progress / Planned

### Phase 9: Bootstrapping & Launch (Weeks 28–36)

| Item | ETA | Status |
|------|-----|--------|
| Anchor community seeding (3-5 groups, 24+ users) | 2026-06 | 🔶 Pending: requires real user recruitment |
| Invite-first launch mechanics | 2026-06 | ✅ Designed; implementation ready |
| Minimum viable network (5-8 concurrent users) | 2026-06 | ✅ Threshold defined (MINIMUM_VIABLE_NETWORK.md) |
| Success metrics (D7≥40%, D30≥30%) | 2026-06 | ✅ Defined (SUCCESS_METRICS.md) |
| Open beta with version expectations | 2026-Q3 | 🔶 Blocked: requires product readiness + anchor user group success |

**Blockers**:
- 🔶 **User recruitment**: Need to identify and recruit 3-5 friend groups (5-8 users each) for anchor community testing
- 🔶 **Beta infrastructure**: Monitoring, incident response, rollback strategy

### Deferred to v1.0 / Later

| Feature | Reason |
|---------|--------|
| **I2P transport** (adapter complete but not v0.1 launch requirement) | Tor covers primary use case; I2P adds parity |
| **Shroud circuit optimization** (5-hop analysis) | User feedback from v0.1 will inform |
| **Behavioral Sybil detection** | Low priority; peer scoring sufficient for v0.1 |
| **Specter-to-Surface bridge** | Requires additional threat model analysis |
| **Streaming tunnels** (video, large files) | v0.1 focuses on text + metadata |

---

## What's Stable (Can Build On)

### Public APIs (EXTENSION_CONTRACT.md)

- ✅ **Custom Wave types** — STABLE
- ✅ **Game module SDK** — STABLE
- ✅ **Transport adapters** — STABLE
- ✅ **Resonance hooks** — EXPERIMENTAL (may refine in v0.2)

### Wire Protocol (PROTOCOL.md)

- ✅ **MurmurEnvelope** format — STABLE
- ✅ **GossipSub topics** (waves, identity, shroud, pulse) — STABLE
- ✅ **Stream protocols** (Shroud circuits, Wave sync, peer exchange) — STABLE
- ✅ **Cryptography** (Ed25519, X25519, XChaCha20) — STABLE

**Implication**: External clients can be built against these APIs without fear of breaking changes.

---

## Design Principles (Immutable for v0.1+)

1. ✅ **Privacy is structural** — No opt-in/opt-out; encryption and anonymity by default
2. ✅ **No permanent record** — Waves expire; no archive server
3. ✅ **Identity is self-sovereign** — Keypairs local; no registration
4. ✅ **The network is the interface** — Pulse Map, not algorithmic feed
5. ✅ **Anonymity is first-class** — Specters + Shroud as core, not add-on
6. ✅ **Growth is organic** — No metrics, no viral mechanics, invite-first

---

## Timeline Estimate

| Phase | Weeks | Status |
|-------|-------|--------|
| **0–4**: Foundation & UX | 1–16 | ✅ Complete |
| **5**: Tor/I2P Transport | 12–18 | ✅ Complete |
| **6**: Tunneling | 16–26 | ✅ Code complete; testing |
| **7**: Reseed | 20–30 | ✅ Code complete; testing |
| **8**: Extension Hardening | 24–32 | ✅ In progress |
| **9**: Launch | 28–36 | 🔶 Ready once anchor users onboarded |

**Overall**: v0.1 soft launch plausible **June 2026** (with 24+ anchor users). Public beta **July 2026**.

---

## How We Measure Success

We **do NOT track** (per Design Principle #6):

- ❌ Daily Active Users (DAU)
- ❌ Time-on-app
- ❌ Follower counts
- ❌ Algorithmic engagement metrics

We **DO track** (per SUCCESS_METRICS.md):

- ✅ **D7 retention** ≥ 40% (users return voluntarily)
- ✅ **D30 retention** ≥ 30% (sustained engagement)
- ✅ **Games/week/user** ≥ 0.5 (quality engagement)
- ✅ **Waves/week/user** ≥ 3 (core communication loop)
- ✅ **Specter adoption** ≥ 30% (anonymous layer validated)
- ✅ **Viral coefficient K** ≥ 1.5 (invites convert to new users)

---

## Community Participation

### How to Contribute

1. **Report bugs** → GitHub Issues
2. **Propose features** → Create a MEP (MEPs/MEP-0-TEMPLATE.md)
3. **Build extensions** → Implement a GameModule or custom Wave type
4. **Run relays** → Operate a Shroud or Tunnel relay node (coming v1.0)

### Decision Making

- **Strategic decisions** documented in DECISIONS.md
- **Significant changes** go through MEP process (MEPs/)
- **Bugs** fixed via pull requests with tests
- **No steering committee** — merit-based discussion

---

## Links

- **Vision & Strategy**: [PRODUCT_IDENTITY.md](PRODUCT_IDENTITY.md), [DESIGN_DOCUMENT.md](DESIGN_DOCUMENT.md)
- **Technical Spec**: [TECHNICAL_IMPLEMENTATION.md](TECHNICAL_IMPLEMENTATION.md), [PROTOCOL.md](PROTOCOL.md)
- **Security**: [THREAT_MODEL.md](THREAT_MODEL.md), [SECURITY_PRIVACY.md](SECURITY_PRIVACY.md), [SECURITY_REVIEW_Q2_2026.md](SECURITY_REVIEW_Q2_2026.md)
- **Extension Docs**: [EXTENSION_CONTRACT.md](EXTENSION_CONTRACT.md), [MEPs/](MEPs/)
- **User Docs**: [README.md](README.md), [RECOVERY.md](RECOVERY.md), [RESEED.md](RESEED.md)
- **Decision Log**: [DECISIONS.md](DECISIONS.md)

---

## Questions?

- **Technical questions**: Check TECHNICAL_IMPLEMENTATION.md or open an issue
- **Security concerns**: Email security contact [TBD]
- **Feature requests**: Create a MEP in the MEPs/ folder
- **Bugs**: GitHub Issues with reproduction steps

---

**Roadmap Version**: 1.0  
**Last Updated**: 2026-05-07  
**Next Review**: 2026-Q3  
**Status**: Ready for public share
