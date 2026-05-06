# MURMUR

A decentralized, peer-to-peer social network with dual-layer identity. No servers. No algorithms. No permanent record.

The network replaces the infinite scroll with a living map — a force-directed graph where people are glowing nodes, relationships are visible edges, and content ripples outward through the mesh. Beneath the surface lies an Anonymous Layer: pseudonymous identities called Specters, routed through onion-style circuits, with their own reputation system and social mechanics.

---

## Why MURMUR

- **The network is the interface.** No feed. You navigate a real-time spatial graph — the Pulse Map — to discover content and people.
- **Privacy is structural.** All connections encrypted. Anonymous traffic onion-routed. Surface and Specter identities are cryptographically unlinkable.
- **Identity is self-sovereign.** Your identity is an Ed25519 keypair. No email. No phone. No third party.
- **Content is ephemeral.** Waves expire after a configurable TTL (default 7 days, max 30). The network forgets.
- **Anonymity is first-class.** The Anonymous Layer has its own identity system, reputation metric, and unlockable social mechanics.
- **No metrics.** No likes. No follower counts. Content that resonates generates conversation. Content that doesn't fades quietly.

---

## Architecture

Six subsystems: Networking (libp2p, GossipSub, Kademlia DHT), Identity (Ed25519 + Curve25519 keypairs, visual sigils, privacy modes), Content (Waves with PoW and TTL), Anonymous Layer (Specters, Shroud onion routing, Resonance reputation), Pulse Map (force-directed graph visualization), and Onboarding (six-phase guided introduction).

See `DESIGN_DOCUMENT.md` for the complete specification.

---

## Project Structure

    murmur/
    ├── README.md
    ├── DESIGN_DOCUMENT.md           # Complete specification
    ├── TECHNICAL_IMPLEMENTATION.md  # Technical details
    ├── cmd/murmur/                  # Application entry point
    ├── proto/                       # Protocol Buffer definitions
    ├── pkg/
    │   ├── app/                     # Application lifecycle, event bus
    │   ├── config/                  # Configuration
    │   ├── networking/
    │   │   ├── transport/           # libp2p host, Noise/QUIC/TCP
    │   │   ├── gossip/              # GossipSub configuration, topics
    │   │   ├── discovery/           # Kademlia DHT, peer discovery
    │   │   ├── relay/               # NAT traversal, hole punching
    │   │   └── mesh/                # Peer scoring, mesh health
    │   ├── identity/
    │   │   ├── keys/                # Ed25519/Curve25519 keypairs
    │   │   ├── sigils/              # Deterministic visual identity
    │   │   ├── declarations/        # Profile declarations
    │   │   └── modes/               # Open/Hybrid/Guarded/Fortress
    │   ├── content/
    │   │   ├── waves/               # Wave creation, validation
    │   │   ├── pow/                 # SHA-256 Proof of Work
    │   │   ├── propagation/         # Gossip relay, hop tracking
    │   │   ├── threads/             # Reply chains, threading
    │   │   └── storage/             # Local cache, expiration
    │   ├── anonymous/
    │   │   ├── specters/            # Specter identity creation
    │   │   ├── shroud/              # Three-hop onion circuits
    │   │   ├── resonance/           # Reputation computation
    │   │   └── mechanics/           # Phantom Gifts, Mini-Games, etc.
    │   ├── store/                   # Bbolt persistent storage
    │   ├── pulsemap/
    │   │   ├── layout/              # Force-directed graph engine
    │   │   ├── rendering/           # Ebitengine visualization
    │   │   ├── interaction/         # Pan, zoom, navigation
    │   │   └── overlays/            # Anonymous layer overlay
    │   └── onboarding/
    │       ├── flow /                # Six-phase sequence
    │       ├── tutorials/           # Guided exploration
    │       └── bootstrap/           # Initial peer connection
    └── assets/
        ├── wordlists/               # Specter name wordlists
        └── themes/                  # Pulse Map themes

---

## Quick Reference

| Concept | Summary |
|---|---|
| **Wave** | Signed, ephemeral text message (≤2048 bytes) with PoW and TTL |
| **Pulse Map** | Real-time force-directed social graph — the primary interface |
| **Specter** | Pseudonymous anonymous identity with procedural name and sigil |
| **Shroud** | Three-hop onion routing network for anonymous traffic |
| **Resonance** | Anonymous reputation metric; milestones unlock mechanics at 25/50/75/100/200 |
| **Privacy Modes** | Open (surface only), Hybrid (both layers), Guarded, Fortress (anonymous only) |

---

## Status

**v0.1 Foundation** — 85–90% complete. Core infrastructure fully operational:
- ✅ **Networking**: libp2p transport, GossipSub, Kademlia DHT, NAT traversal, relay
- ✅ **Identity**: Ed25519/Curve25519 keypairs, BIP-39 recovery, Argon2id keystore encryption, sigils
- ✅ **Content**: 8 Wave types, SHA-256 PoW (20-bit default), TTL enforcement, threading
- ✅ **Anonymous Layer**: Specters, 3-hop Shroud circuits, Resonance (13 milestones), 10 mini-games
- ✅ **Pulse Map**: Force-directed layout (60fps @ 500 nodes), Ebitengine rendering, visual effects
- ✅ **Storage**: Bbolt with 7 canonical buckets, typed accessors, LRU eviction
- ✅ **Security**: Key zeroing, Bloom filter deduplication, per-peer rate limiting, ZK Resonance proofs
- ✅ **Onboarding**: All 6 phases complete (Welcome, Identity, Mode, Bootstrap, Exploration, First Wave) with first-week nudges
- ✅ **Cross-Layer Visibility**: Specter Marks render on Surface Layer with orbit animations and tooltips; remaining mechanics (gifts, puzzles, mini-games) deferred to post-v0.1

Binary builds and connects to network. Test suite: 100% pass rate (64 packages with tests, 72 total packages), zero race conditions. Cross-platform validation: Ebitengine rendering and libp2p connectivity validated on Linux/macOS/Windows. See AUDIT.md for detailed goal-achievement assessment and ROADMAP.md for implementation checklist.