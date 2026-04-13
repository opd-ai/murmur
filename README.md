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

See `docs/unified-design-document.md` for the complete specification.

---

## Project Structure

    murmur/
    ├── README.md
    ├── docs/
    │   ├── unified-design-document.md
    │   ├── networking.md
    │   ├── identity.md
    │   ├── content.md
    │   ├── anonymous-layer.md
    │   ├── pulse-map.md
    │   ├── onboarding.md
    │   └── growth.md
    ├── src/
    │   ├── networking/
    │   │   ├── transport/          # Noise/TLS, connection management
    │   │   ├── gossip/             # GossipSub configuration, topic management
    │   │   ├── discovery/          # Kademlia DHT, peer discovery, bootstrap
    │   │   ├── relay/              # NAT traversal, hole punching, relay fallback
    │   │   └── mesh/               # Peer scoring, mesh health, target 6-12 peers
    │   ├── identity/
    │   │   ├── keys/               # Ed25519/Curve25519 generation and storage
    │   │   ├── sigils/             # Deterministic visual identity generation
    │   │   ├── declarations/       # Profile declarations, trust anchors
    │   │   └── modes/              # Open / Hybrid / Fortress privacy modes
    │   ├── content/
    │   │   ├── waves/              # Wave creation, signing, validation
    │   │   ├── pow/                # SHA-256 Proof of Work (2-5s target)
    │   │   ├── propagation/        # Gossip relay, hop tracking, TTL enforcement
    │   │   ├── threads/            # Reply chains, conversation threading
    │   │   └── storage/            # Local cache, expiration, garbage collection
    │   ├── anonymous/
    │   │   ├── specters/           # Specter identity creation, name generation
    │   │   ├── shroud/             # Three-hop onion circuit construction
    │   │   ├── resonance/          # Local reputation computation, rank thresholds
    │   │   └── mechanics/          # Phantom Gifts, Duels, Marks, Events, Councils
    │   ├── pulsemap/
    │   │   ├── layout/             # Force-directed graph engine
    │   │   ├── rendering/          # Node/edge drawing, ripple effects, ghost layer
    │   │   ├── interaction/        # Pan, zoom, node selection, navigation
    │   │   └── overlays/           # Anonymous layer overlay, activity heatmap
    │   └── onboarding/
    │       ├── flow/               # Six-phase sequence controller
    │       ├── tutorials/          # Guided exploration, contextual hints
    │       └── bootstrap/          # Initial peer connection, first Wave prompt
    ├── tests/
    │   ├── unit/
    │   ├── integration/
    │   └── simulation/             # Multi-node network simulations
    └── assets/
        ├── wordlists/              # Specter name generation wordlists
        └── themes/                 # Pulse Map color themes

---

## Quick Reference

| Concept | Summary |
|---|---|
| **Wave** | Signed, ephemeral text message (≤2048 bytes) with PoW and TTL |
| **Pulse Map** | Real-time force-directed social graph — the primary interface |
| **Specter** | Pseudonymous anonymous identity with procedural name and sigil |
| **Shroud** | Three-hop onion routing network for anonymous traffic |
| **Resonance** | Anonymous reputation metric; milestones unlock mechanics at 25/50/75/100/200 |
| **Privacy Modes** | Open (surface only), Hybrid (both layers), Fortress (anonymous only) |

---

## Status

Pre-implementation. The design document is complete. Everything else is ahead.