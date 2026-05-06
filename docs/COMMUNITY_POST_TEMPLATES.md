# MURMUR Community Post Templates

Social media and forum templates for announcing MURMUR v0.1.0-rc1.

---

## Short Form (Twitter/X, Mastodon, Bluesky)

### Option 1: Privacy Focus

```
🌐 MURMUR v0.1.0-rc1 is live!

Decentralized P2P social network. No servers. No algorithms. No permanent record.

✨ Dual-layer identity (Surface + Specter)
✨ Ephemeral content (7-30 day TTL)
✨ Anonymous mini-games
✨ Force-directed Pulse Map interface

Join the mesh: https://github.com/opd-ai/murmur

#privacy #decentralized #opensource
```

### Option 2: Technology Focus

```
🚀 Just released MURMUR v0.1.0-rc1!

P2P social network built with:
- Go + libp2p (GossipSub, DHT, NAT traversal)
- Ebitengine (force-directed graph viz)
- Ed25519 + Curve25519 (dual-layer crypto)
- 3-hop Shroud circuits (onion routing)

100% test pass rate, zero race conditions.

Repo: https://github.com/opd-ai/murmur

#golang #libp2p #p2p
```

### Option 3: Social Focus

```
💬 Introducing MURMUR — a different kind of social network

- Spatial UI (force-directed graph, not infinite scroll)
- Anonymous identities (Specters with procedural names)
- Ephemeral content (messages expire, network forgets)
- No likes, no followers, no metrics

Early adopter release: v0.1.0-rc1

https://github.com/opd-ai/murmur

#social #privacy #p2p
```

---

## Medium Form (Hacker News, Reddit, Lobste.rs)

### Title Options

1. **MURMUR v0.1.0-rc1: Decentralized P2P social network with dual-layer identity and onion routing**
2. **Show HN: MURMUR — P2P social network with spatial UI and anonymous mini-games**
3. **MURMUR: No servers, no algorithms, no permanent record — v0.1.0-rc1 released**

### Reddit Post (r/programming, r/privacy, r/golang)

```
# MURMUR v0.1.0-rc1 Released — Decentralized P2P Social Network

I'm excited to share the first release candidate of **MURMUR**, a decentralized peer-to-peer social network I've been working on.

## What Makes MURMUR Different

- **No Servers** — Fully P2P; your device is infrastructure
- **Spatial Interface** — Force-directed graph (Pulse Map) replaces infinite scroll
- **Dual-Layer Identity** — Surface (Ed25519) + Specter (Curve25519 anonymous)
- **Ephemeral Content** — Messages expire after 7-30 days
- **Anonymous Mini-Games** — 10 game mechanics (Cipher Puzzles, Specter Hunts, Territory Drift, etc.)
- **No Metrics** — No likes, no follower counts

## Tech Stack

- **Go 1.22+** with goroutines/channels for concurrency
- **libp2p** (GossipSub v1.1, Kademlia DHT, Noise transport, NAT traversal)
- **Ebitengine** for 2D rendering (60fps @ 500 nodes)
- **Bbolt** for embedded storage
- **Protocol Buffers** for wire serialization

## Quality Metrics

- 64/64 test packages passing (72 total)
- Zero race conditions (`-race -count=1`)
- Max cyclomatic complexity ≤10 (6,257 functions analyzed)
- Cross-platform: Linux, macOS, Windows (amd64 + arm64)

## Current Status

v0.1.0-rc1 is **85-90% feature complete** for the Foundation milestone:
- ✅ Full networking stack (libp2p, GossipSub, DHT, NAT traversal)
- ✅ Dual-layer identity (Ed25519/Curve25519, BIP-39 recovery, Argon2id keystore)
- ✅ 8 Wave types with SHA-256 PoW
- ✅ 3-hop Shroud onion circuits
- ✅ Resonance reputation (13 milestones)
- ✅ 10 anonymous mini-games
- ✅ Pulse Map visualization
- ✅ 6-phase onboarding

## Known Limitations

- Text-only content (no images/audio/video yet)
- Desktop-only (mobile support post-v1.0)
- Manual relay configuration (auto-discovery planned v0.1.0-final)
- No Tor/I2P integration yet (planned v0.2)

## Links

- **GitHub**: https://github.com/opd-ai/murmur
- **Release Notes**: [RELEASE_NOTES_v0.1.0-rc1.md](link)
- **Complete Spec**: [DESIGN_DOCUMENT.md](link)
- **FAQ**: [docs/FAQ.md](link)

## Target Audience

MURMUR is for friend groups (4-8 people) who want private, playful communication without platform surveillance. If you value self-sovereign identity, metadata unlinkability, and ephemeral-by-default content, this might interest you.

**Not for**: influencers, users needing 24/7 uptime, or those expecting token incentives.

## Installation

Binary releases for Linux/macOS/Windows: https://github.com/opd-ai/murmur/releases/tag/v0.1.0-rc1

Or build from source:
```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
git checkout v0.1.0-rc1
go build -o murmur ./cmd/murmur
./murmur
```

Feedback welcome! This is an early release — rough edges expected, protocol may change.

---

*MIT License | Built with Go + libp2p + Ebitengine*
```

### Hacker News Comment (Show HN)

```
Author here. Happy to answer questions!

Some context on design decisions:

**Why P2P instead of federated?** Metadata unlinkability requires no trusted servers. Federation (Mastodon, Matrix) still leaks who-talks-to-whom to server operators.

**Why ephemeral?** Permanent archives create legal liability, storage costs, and long-term metadata exposure. The network forgets by design.

**Why spatial UI?** Algorithmic feeds require behavior tracking (privacy violation). Spatial navigation (Pulse Map force-directed graph) makes relationships and propagation visible without algorithms.

**Why anonymous mini-games?** Retention through social value, not engagement optimization. Games create playful moments that are genuinely fun with friends.

**Why not just use Tor?** Shroud is optimized for social traffic (latency-tolerant, small messages). It's not a Tor replacement — we're planning Tor/I2P bridging in v0.2 for users who need stronger guarantees.

**Why Go?** Excellent concurrency primitives (goroutines/channels), strong stdlib crypto, mature libp2p implementation, single-binary deployment.

**Why Ebitengine?** Cross-platform 2D rendering, Kage shaders for effects, good performance (60fps @ 500 nodes). No web engine dependencies (Electron, Tauri).

The project has been in development for ~6 months with ~50,000 lines of Go across 72 packages. This RC is the first public release — feedback appreciated!
```

---

## Long Form (Blog Post, Dev.to, Medium)

### Title

**MURMUR: Building a Decentralized Social Network Without Servers, Algorithms, or Permanent Records**

### Introduction

```
After six months of development, I'm releasing MURMUR v0.1.0-rc1 — a decentralized peer-to-peer social network that challenges core assumptions of mainstream social platforms.

No servers. No algorithms. No permanent record. No metrics.

Instead:
- A force-directed graph interface (Pulse Map) where relationships are visible edges
- Dual-layer identity (attributed Surface + anonymous Specter)
- Ephemeral content that expires after 7-30 days
- Anonymous mini-games driving retention through play, not engagement optimization

This post explains what MURMUR is, why I built it, and how it works technically.
```

### Section Outline

1. **The Problem** — Why mainstream social networks are broken (surveillance capitalism, engagement optimization, platform power, metadata leakage)
2. **The Vision** — What MURMUR aims to be (spatial interface, self-sovereign identity, metadata unlinkability, playful mechanics)
3. **Architecture** — How it works (libp2p networking, dual-layer crypto, Shroud onion routing, Pulse Map visualization)
4. **Anonymous Layer** — Specters, Resonance reputation, mini-games, Phantom Gifts
5. **Current State** — What works in v0.1.0-rc1, what's coming next
6. **Try It** — Installation instructions, invitation link
7. **Contributing** — How to get involved

### Call to Action

```
MURMUR is open source (MIT License) and available now:

- **GitHub**: https://github.com/opd-ai/murmur
- **Releases**: https://github.com/opd-ai/murmur/releases/tag/v0.1.0-rc1
- **Documentation**: Complete specification in repo

This is an early release. Rough edges are expected. The protocol may change. But if you value privacy, self-sovereignty, and playful communication over algorithmic engagement, I'd love your feedback.

Join the mesh. Explore the Pulse Map. Become a Specter.
```

---

## Forum Post (Privacy Forums, P2P Communities)

### Title

**MURMUR v0.1.0-rc1: P2P Social Network with Onion Routing and Dual-Layer Identity**

### Body

```
I've been building a decentralized P2P social network focused on metadata unlinkability and playful anonymous mechanics. First release candidate is now available.

**Core Privacy Features:**

1. **No Servers** — Fully peer-to-peer; your device is infrastructure. No centralized logs, no analytics, no platform surveillance.

2. **Dual-Layer Identity** — Surface (Ed25519) for attributed content + Specter (Curve25519) for anonymous participation. Cryptographically unlinkable.

3. **Shroud Onion Routing** — Three-hop circuits with XChaCha20-Poly1305 encryption per layer. Not Tor-level anonymity (yet), but defends against network-level observers.

4. **Ephemeral by Default** — Content expires after 7-30 days. No permanent archives, reduced long-term metadata exposure.

5. **No Metrics** — No likes, no follower counts, no engagement tracking. Conversations drive retention.

**Threat Model:**

- ✅ Defends against: Network observers, malicious peers, platform deanonymization
- ❌ Out of scope: Global passive adversaries, state-level traffic analysis (→ use Tor/I2P)

Tor/I2P bridging planned for v0.2 to provide escape hatch for users with stricter threat models.

**Tech Stack:**

- Go 1.22+ (goroutines/channels, strong crypto)
- libp2p (GossipSub, Kademlia DHT, Noise transport, NAT traversal)
- Ebitengine (2D rendering, force-directed graph)
- Argon2id + XChaCha20-Poly1305 (keystore encryption)
- BIP-39 (recovery phrases)

**Cryptographic Primitives:**

| Use Case | Algorithm |
|---|---|
| Surface signatures | Ed25519 |
| Anonymous key exchange | Curve25519 (X25519) |
| Symmetric encryption | XChaCha20-Poly1305 |
| Proof of Work | SHA-256 |
| Identity hashing | BLAKE3 |
| Key derivation | Argon2id |
| ZK Resonance proofs | Pedersen + Bulletproofs |

**Current Limitations:**

- Desktop-only (mobile post-v1.0)
- Text-only content (rich media planned v0.2)
- Manual relay configuration (auto-discovery planned v0.1.0-final)
- Shroud not Tor-equivalent (bridging planned v0.2)

**Links:**

- GitHub: https://github.com/opd-ai/murmur
- Security Model: SECURITY_PRIVACY.md in repo
- Complete Spec: DESIGN_DOCUMENT.md in repo

Feedback from the privacy community would be especially valuable — the threat model is documented, but additional review welcome.

MIT License | 100% open source
```

---

## Email Template (Mailing List Announcement)

### Subject

**Announcing MURMUR v0.1.0-rc1 — Decentralized P2P Social Network**

### Body

```
Hi everyone,

I'm excited to announce the first release candidate of MURMUR, a decentralized peer-to-peer social network with dual-layer identity and onion routing.

## What is MURMUR?

MURMUR is a P2P social network with:
- No servers (fully decentralized)
- No algorithms (spatial navigation via force-directed graph)
- No permanent record (content expires after 7-30 days)
- Dual-layer identity (attributed Surface + anonymous Specter)
- Anonymous mini-games (10 mechanics for playful participation)

## Why v0.1.0-rc1 Matters

This release represents 85-90% feature completion for the Foundation milestone:
- ✅ Full libp2p networking stack
- ✅ Dual-layer cryptographic identity (Ed25519 + Curve25519)
- ✅ 3-hop Shroud onion routing
- ✅ Pulse Map visualization (60fps @ 500 nodes)
- ✅ 8 Wave types with PoW
- ✅ 10 anonymous mini-games
- ✅ Cross-platform support (Linux, macOS, Windows)

## Quality Metrics

- 64/64 test packages passing
- Zero race conditions
- Max cyclomatic complexity ≤10
- MIT License, 100% open source

## Installation

Binary releases: https://github.com/opd-ai/murmur/releases/tag/v0.1.0-rc1

Or build from source:
```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur && git checkout v0.1.0-rc1
go build -o murmur ./cmd/murmur
./murmur
```

## Documentation

- Complete specification: DESIGN_DOCUMENT.md
- Technical details: TECHNICAL_IMPLEMENTATION.md
- FAQ: docs/FAQ.md
- Security model: SECURITY_PRIVACY.md

## Roadmap

- v0.1.0-final (Q2 2026): Bug fixes, relay discovery, performance optimization
- v0.2.0 (Q3 2026): Tor/I2P integration, multi-device sync, rich media
- v1.0.0 (Q4 2026): Protocol stabilization, mobile builds, extension API

## Get Involved

Contributions welcome! Key areas:
- Tor/I2P transport integration
- Relay discovery protocol
- Mobile ports (Android/iOS)
- Performance optimization
- Anonymous game mechanics

See CONTRIBUTING.md for details.

## Links

- GitHub: https://github.com/opd-ai/murmur
- Issues: https://github.com/opd-ai/murmur/issues
- Discussions: https://github.com/opd-ai/murmur/discussions

Feedback welcome — this is an early release, and I'd love to hear what the community thinks.

Thanks,
[Your Name]

---

MURMUR — No servers. No algorithms. No permanent record.
MIT License | github.com/opd-ai/murmur
```

---

## Notes for Community Managers

### Key Talking Points

1. **Privacy by design**: Metadata unlinkability through P2P + onion routing
2. **Novel UI**: Spatial navigation (force-directed graph) vs algorithmic feeds
3. **Anonymous mechanics**: First-class pseudonymous identities + mini-games
4. **Ephemeral by default**: Content expires, reducing long-term exposure
5. **Open source**: MIT License, complete specification available

### Framing Guidance

- **For privacy communities**: Emphasize threat model, cryptographic primitives, Shroud onion routing
- **For tech communities**: Highlight architecture (Go + libp2p + Ebitengine), quality metrics, test coverage
- **For social platform critics**: Focus on no-metrics design, spatial UI, playful mechanics over engagement optimization
- **For P2P enthusiasts**: Emphasize libp2p stack, NAT traversal, relay network, gossip propagation

### Hashtags

Primary: `#privacy` `#decentralized` `#opensource` `#p2p`  
Secondary: `#golang` `#libp2p` `#socialnetwork` `#anonymity`  
Platform-specific: `#infosec` (Twitter), `#privacy` (Mastodon), `#golang` (Reddit r/golang)

### Engagement Strategy

1. **Week 1**: Initial announcement across platforms, respond to questions
2. **Week 2**: Technical deep-dive posts (architecture, crypto primitives)
3. **Week 3**: User stories, screenshots, video demos
4. **Week 4**: Contribution guide, roadmap discussion, community feedback

### Response Templates

**Q: Why not just use Tor/Matrix/Signal?**
> MURMUR isn't replacing those — it's a different design point. Tor: anonymity network. Matrix: federated messaging. Signal: mobile-first E2EE. MURMUR: spatial P2P social with dual-layer identity. We're planning Tor/I2P bridging in v0.2 for users who need stronger guarantees.

**Q: How do you prevent spam without metrics?**
> Proof of Work (SHA-256, 2-5s per message) rate-limits spam. Resonance reputation unlocks mechanics but isn't transferable/gameable. No follower counts = no incentive for spam-for-growth. Local moderation (you control your mesh) instead of platform moderation.

**Q: Why Go instead of Rust?**
> Go: mature libp2p, excellent concurrency (goroutines/channels), strong stdlib crypto, single-binary deployment, fast compile times. Rust would work too, but Go was a better fit for this project. Open to Rust implementations from the community.

**Q: Is this production-ready?**
> RC1 = early adopter release. Core functionality works, test suite passes, but expect rough edges. Protocol may change pre-v1.0. Best for friend groups (4-8 people) willing to tolerate bugs in exchange for privacy benefits.

---

## Media Kit

### Project Logo

See `assets/logo/` for vector and raster formats.

### Screenshots

See `assets/screenshots/` for:
- Pulse Map (full mesh)
- Anonymous Layer overlay
- Wave propagation animation
- Mini-game interface
- Onboarding flow

### Project Tagline

**"No servers. No algorithms. No permanent record."**

### One-Sentence Description

**"MURMUR is a decentralized P2P social network with dual-layer identity (attributed + anonymous), spatial UI (force-directed graph), and ephemeral content (7-30 day TTL)."**

### Boilerplate

**"MURMUR is an open-source (MIT License) decentralized peer-to-peer social network. It uses libp2p for networking, Ed25519/Curve25519 for dual-layer identity, and Ebitengine for spatial visualization. The project is pre-v1.0 and actively seeking contributors and early adopters."**
