# MURMUR v0.1.0-rc1 — First Public Release Candidate

**Release Date:** May 6, 2026  
**Release Type:** Release Candidate (RC1)  
**Target:** Early Adopters, Privacy Advocates, Friend Groups

---

## What is MURMUR?

MURMUR is a **decentralized peer-to-peer social network** with dual-layer identity architecture. There are no servers, no algorithms, and no permanent record. Every participant's device is both client and server — a node in a living mesh.

### Core Features

- **Spatial Social Interface** — The Pulse Map: a real-time force-directed graph where people are glowing nodes, relationships are visible edges, and content ripples outward through the mesh
- **Dual-Layer Identity** — Surface Layer (attributed Ed25519 identity) + Anonymous Layer (pseudonymous Specters routed through onion circuits)
- **Ephemeral Content** — Waves expire after 7-30 days; the network forgets by design
- **Anonymous Games** — 10 mini-games (Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, etc.) playable with anonymous identities
- **Four Privacy Modes** — Open → Hybrid → Guarded → Fortress (Shadow Gradient system)
- **No Metrics** — No likes, no follower counts; conversations drive retention, not engagement optimization

---

## Why v0.1.0-rc1 Matters

This is the **first public release candidate** for MURMUR's v0.1 Foundation milestone. It represents **85-90% feature completion** of the core protocol and reference implementation.

### What Works Now

✅ **Full Network Stack** — libp2p transport, GossipSub gossip, Kademlia DHT, NAT traversal, relay fallback  
✅ **Dual-Layer Identity** — Ed25519 Surface keys, Curve25519 Specter keys, BIP-39 recovery, Argon2id keystore encryption  
✅ **8 Wave Types** — Surface, Reply, Veiled, Specter, Sigil, Abyssal, Masked, Beacon  
✅ **SHA-256 Proof of Work** — 20-bit default difficulty, 2-5 second target computation time  
✅ **3-Hop Shroud Circuits** — Onion routing for anonymous traffic with hop diversity enforcement  
✅ **Resonance Reputation** — 13 milestones (Shade → Abyss), locally computed, non-transferable  
✅ **10 Anonymous Mini-Games** — All mechanics implemented and tested  
✅ **Pulse Map Visualization** — 60fps @ 500 nodes, force-directed layout, Kage shaders for glow/ripple/spectra  
✅ **6-Phase Onboarding** — Welcome → Identity → Mode → Bootstrap → Exploration → First Wave  
✅ **Cross-Platform Support** — Linux, macOS, Windows (amd64 + arm64)

### What's Not In RC1

❌ **Tor/I2P Integration** — Planned for v0.2; Shroud provides onion routing but not Tor-level guarantees  
❌ **Mobile Builds** — Android/iOS support deferred post-v1.0  
❌ **Relay Discovery** — Manual relay configuration required; automated discovery planned  
❌ **Multi-Device Sync** — Single device per identity in RC1  
❌ **Rich Media** — Text-only Waves; images/audio/video deferred

---

## Quality Metrics

- **Test Coverage**: 64/64 packages with tests passing (72 total packages)
- **Race Detector**: Zero race conditions detected (`-race -count=1`)
- **Cyclomatic Complexity**: Maximum CC ≤10 (6,257 functions analyzed)
- **Build Targets**: 5 platforms (linux/darwin/windows on amd64/arm64)
- **Performance**: 60fps Pulse Map rendering @ 500 nodes, <500ms Wave propagation across 3 hops
- **Security**: Key zeroing, Bloom filter deduplication, ZK Resonance proofs, transport encryption

---

## Installation

### Binary Releases (Recommended)

Download platform-specific binaries from the [Releases page](https://github.com/opd-ai/murmur/releases/tag/v0.1.0-rc1):

```bash
# Linux (amd64)
wget https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-linux-amd64.tar.gz
tar -xzf murmur-linux-amd64.tar.gz
./murmur

# macOS (amd64)
wget https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-darwin-amd64.tar.gz
tar -xzf murmur-darwin-amd64.tar.gz
./murmur

# Windows (amd64)
# Download murmur-windows-amd64.zip and extract
murmur.exe
```

### Build from Source

```bash
# Prerequisites: Go 1.22+, git
git clone https://github.com/opd-ai/murmur.git
cd murmur
git checkout v0.1.0-rc1

# Build
go build -o murmur ./cmd/murmur

# Test (optional)
go test -race ./...

# Run
./murmur
```

---

## Getting Started

### First Launch

1. **Launch MURMUR** — Run the binary; onboarding begins automatically
2. **Create Identity** — Generate Ed25519 keypair + visual sigil (Phase 2)
3. **Choose Privacy Mode** — Select Open, Hybrid, Guarded, or Fortress (Phase 3)
4. **Bootstrap** — Connect to the network via invitation or DHT (Phase 4)
5. **Explore Pulse Map** — Navigate the spatial graph, see nodes appear (Phase 5)
6. **Publish First Wave** — Write ≤2048 bytes, compute PoW, watch it propagate (Phase 6)

### Inviting Friends

MURMUR grows through invitations. To invite someone:

1. Open the menu (press `Esc` → "Invite a Friend")
2. Copy the `murmur://invite/` link or show the QR code
3. Share with your friend via any channel (text, email, etc.)
4. They bootstrap directly through your node — warm start guaranteed

### Creating a Specter (Anonymous Identity)

1. Switch to Hybrid or Fortress mode (Settings → Privacy Mode)
2. Navigate to Anonymous Layer (press `Tab` key)
3. Create Specter identity (Menu → "Create Specter")
4. Receive procedurally-generated name + sigil (e.g., "Crimson Whisper")
5. Build Resonance through participation to unlock mechanics

---

## Known Limitations

### Networking
- **Manual relay configuration required** — No automated relay discovery in RC1
- **Limited NAT traversal** — May require port forwarding for symmetric NAT
- **No Tor/I2P bridging** — Shroud provides onion routing but not Tor-level guarantees

### Content
- **Text-only Waves** — No images, audio, video support
- **No edit/delete** — Published Waves are immutable until expiration
- **No search** — Content discovery via spatial navigation only

### Platform Support
- **No mobile builds** — Desktop-only (Linux, macOS, Windows)
- **No browser version** — Native app required

### Multi-Device
- **Single device per identity** — No cross-device sync in RC1

---

## Target Audience

MURMUR is designed for:

✅ **Friend groups (4-8 people)** seeking private, playful communication  
✅ **Privacy-conscious users** who understand metadata risks  
✅ **Self-sovereign identity advocates** wanting cryptographic identity without blockchain  
✅ **Early adopters** comfortable with peer-to-peer technology  
✅ **Digital natives** familiar with ephemeral messaging and online gaming

### Anti-Personas (Not Our Target)

❌ Influencers seeking audience growth or public broadcast platforms  
❌ Users requiring 24/7 uptime guarantees and enterprise SLAs  
❌ Non-technical users unwilling to understand basic P2P concepts  
❌ Users seeking permanent archives or searchable message history  
❌ Cryptocurrency enthusiasts expecting token incentives

---

## Architecture Highlights

### Technology Stack

- **Go 1.22+** — Goroutines and channels for concurrency
- **Ebitengine v2.7+** — 2D rendering, Kage shaders, immediate-mode UI
- **go-libp2p v0.36+** — GossipSub, Kademlia DHT, Noise transport, NAT traversal
- **Bbolt** — Embedded key-value storage (single-file database)
- **Protocol Buffers proto3** — All wire-format serialization

### Cryptographic Primitives

| Use Case | Algorithm | Package |
|---|---|---|
| Surface signatures | Ed25519 | `crypto/ed25519` |
| Anonymous key exchange | Curve25519 | `golang.org/x/crypto/curve25519` |
| Symmetric encryption | XChaCha20-Poly1305 | `golang.org/x/crypto/chacha20poly1305` |
| Proof of Work | SHA-256 | `crypto/sha256` |
| Identity hashing | BLAKE3 | `github.com/zeebo/blake3` |
| Key derivation | Argon2id | `golang.org/x/crypto/argon2` |
| ZK Resonance proofs | Pedersen + Bulletproofs | `github.com/bwesterb/go-ristretto` |

### Concurrency Model

- **~8 persistent goroutines**: main (Ebitengine loop), network (libp2p swarm), layout (force-directed), expiry (GC every 60s), heartbeat (every 30s), Shroud maintenance, event bus, DHT refresh
- **Double-buffered Pulse Map**: Layout goroutine writes to back buffer, atomically swaps with front buffer read by rendering thread (zero lock contention)
- **Channel-based communication**: No shared mutable state except atomic.Pointer swaps

---

## Contributing

We welcome contributions! Key areas for contribution:

- **Tor/I2P Transport Integration** — Bridging Shroud with Tor/I2P for stronger anonymity
- **Relay Discovery Protocol** — Automated discovery and health monitoring for Shroud relays
- **Mobile Ports** — Android/iOS support (requires Ebitengine mobile backend)
- **Performance Optimization** — Pulse Map rendering at 1000+ nodes, Wave propagation latency reduction
- **Anonymous Mechanics** — New mini-games, Phantom Council governance, Masked Events

See [CONTRIBUTING.md](../CONTRIBUTING.md) for code style, package structure, and PR process.

---

## Documentation

- **Quick Start**: `README.md`
- **Complete Specification**: `DESIGN_DOCUMENT.md`
- **Technical Details**: `TECHNICAL_IMPLEMENTATION.md`
- **Implementation Plan**: `ROADMAP.md`
- **Security Model**: `SECURITY_PRIVACY.md`
- **Privacy Modes**: `SHADOW_GRADIENT.md`
- **Wave Types**: `WAVES.md`
- **Reputation System**: `RESONANCE_SYSTEM.md`
- **Anonymous Games**: `ANONYMOUS_GAME_MECHANICS.md`
- **Pulse Map**: `PULSE_MAP.md`
- **Onboarding**: `ONBOARDING.md`
- **Glossary**: `GLOSSARY.md`

---

## Support & Community

- **GitHub Issues**: [Report bugs and request features](https://github.com/opd-ai/murmur/issues)
- **GitHub Discussions**: [Ask questions and share feedback](https://github.com/opd-ai/murmur/discussions)
- **Developer Chat**: Join the MURMUR network and find the `#dev` Wave thread

---

## License

MIT License. See [LICENSE](../LICENSE) for details.

---

## Acknowledgments

MURMUR builds on:
- **libp2p** — Protocol Labs' modular networking stack
- **Ebitengine** — Hajime Hoshi's 2D game engine
- **Bbolt** — Etcd's embedded database (fork of Bolt)
- **Protocol Buffers** — Google's serialization format

Special thanks to the privacy and P2P communities for inspiration and guidance.

---

## What's Next

### Post-RC1 Roadmap

**v0.1.0 Final** (Target: Q2 2026)
- [ ] Bug fixes and polish from RC1 feedback
- [ ] Documentation improvements
- [ ] Relay discovery protocol
- [ ] Performance optimization (1000-node Pulse Map target)

**v0.2.0** (Target: Q3 2026)
- [ ] Tor transport integration
- [ ] I2P transport integration
- [ ] Multi-device identity sync
- [ ] Rich media support (images, audio)

**v1.0.0** (Target: Q4 2026)
- [ ] Protocol stabilization
- [ ] Extension contract for third-party networks
- [ ] Mobile builds (Android/iOS)
- [ ] Production-ready relay network

---

## Final Words

MURMUR is a **new kind of social network** — one where the network itself is the interface, where anonymity is a first-class feature, and where privacy is structural rather than contractual.

This is an **early release**. Rough edges are expected. The protocol may change. Keys may be invalidated. Bugs will be found.

But if you value privacy, self-sovereignty, and playful communication over algorithmic engagement, MURMUR offers something genuinely different.

**Join the mesh. Explore the Pulse Map. Become a Specter.**

---

*For technical details, see [RELEASE_NOTES_v0.1.0-rc1.md](../RELEASE_NOTES_v0.1.0-rc1.md)*
