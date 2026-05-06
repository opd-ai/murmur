# MURMUR — Frequently Asked Questions (FAQ)

**Version:** v0.1.0-rc1  
**Last Updated:** May 6, 2026

---

## General Questions

### What is MURMUR?

MURMUR is a decentralized peer-to-peer social network with dual-layer identity. There are no servers, no algorithms, and no permanent record. Every participant's device is both client and server — a node in a living mesh.

The network presents itself through the **Pulse Map**, a force-directed graph where users are glowing nodes, relationships are visible edges, and content ripples outward. Beneath the Surface Layer lies an **Anonymous Layer** of pseudonymous identities called Specters, routed through onion-style circuits.

### How is MURMUR different from other social networks?

1. **No Servers** — Fully peer-to-peer; your device is infrastructure
2. **Spatial Interface** — The Pulse Map replaces the infinite scroll
3. **Dual-Layer Identity** — Surface (attributed) + Specter (anonymous) identities
4. **Ephemeral by Default** — Content expires after 7-30 days
5. **No Metrics** — No likes, no follower counts, no engagement optimization
6. **Anonymous Mini-Games** — 10 game mechanics playable with pseudonyms

### Is MURMUR open source?

Yes. MURMUR is released under the **MIT License**. The source code is available at [github.com/opd-ai/murmur](https://github.com/opd-ai/murmur).

### Who should use MURMUR?

MURMUR is designed for:
- **Friend groups (4-8 people)** wanting private, playful communication
- **Privacy-conscious users** who understand metadata risks
- **Self-sovereign identity advocates** wanting cryptographic identity without blockchain
- **Early adopters** comfortable with peer-to-peer technology

### Who should NOT use MURMUR?

MURMUR is **not** for:
- Influencers seeking audience growth or public broadcast
- Users requiring 24/7 uptime guarantees
- Non-technical users unwilling to understand P2P concepts
- Users needing permanent archives or searchable history
- Cryptocurrency enthusiasts expecting token incentives

---

## Installation & Setup

### What platforms does MURMUR support?

**Desktop (Supported in v0.1.0-rc1):**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

**Mobile (Not Yet Supported):**
- Android/iOS support is planned post-v1.0

### How do I install MURMUR?

**Option 1: Binary Release (Recommended)**
1. Download the binary for your platform from [Releases](https://github.com/opd-ai/murmur/releases/tag/v0.1.0-rc1)
2. Extract the archive
3. Run the binary (`./murmur` on Linux/macOS, `murmur.exe` on Windows)

**Option 2: Build from Source**
```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
git checkout v0.1.0-rc1
go build -o murmur ./cmd/murmur
./murmur
```

### Do I need a display server?

Yes. MURMUR uses Ebitengine for rendering and requires a display. Headless operation is not supported in v0.1.0-rc1.

For CI/testing without a display, use `xvfb-run`:
```bash
xvfb-run -a ./murmur
```

### How much disk space does MURMUR require?

- **Binary**: 20-40 MB (compressed), 60-100 MB (uncompressed)
- **Database**: <50 MB during normal operation (grows with Wave cache)
- **Total**: ~150 MB recommended minimum

### How much bandwidth does MURMUR use?

- **Active use**: 1-5 MB/hour (varies with mesh size and Wave propagation)
- **Idle**: <100 KB/hour (heartbeat pings only)
- **Bootstrap**: 5-10 MB (DHT discovery, peer exchange)

---

## Identity & Privacy

### What is an identity in MURMUR?

MURMUR uses **self-sovereign cryptographic identities** based on public-key cryptography:

- **Surface Identity** — Ed25519 keypair + visual sigil; your attributed identity
- **Specter Identity** — Curve25519 keypair + procedural name/sigil; your anonymous identity

No email, no phone number, no third-party registration required.

### How do I back up my identity?

During onboarding (Phase 2), MURMUR generates a **24-word BIP-39 recovery phrase**. Write it down and store it securely (offline, ideally in multiple locations).

Your keystore files are stored in:
- `~/.murmur/surface.keystore` (Surface key)
- `~/.murmur/specter.keystore` (Specter key)
- `~/.murmur/fortress.keystore` (Fortress transport key, if enabled)

Each file is encrypted with your passphrase using Argon2id + XChaCha20-Poly1305.

### What are the four privacy modes?

MURMUR has four privacy modes (**Shadow Gradient**):

1. **Open** — Surface identity only; maximum visibility
2. **Hybrid** — Surface + Specter identities; balanced visibility
3. **Guarded** — Default-anonymous with selective Surface reveals
4. **Fortress** — Maximum anonymity; all traffic through Shroud circuits

You can change modes at any time (Settings → Privacy Mode).

### Are Surface and Specter identities linked?

**No.** Surface and Specter identities are cryptographically independent:
- Different key types (Ed25519 vs Curve25519)
- No shared derivation path
- Separate encrypted keystore files

Compromising one does **not** reveal the other.

### What is the Shroud?

The **Shroud** is MURMUR's onion routing network for anonymous traffic. It constructs **three-hop circuits** with layer-by-layer encryption (like Tor, but optimized for social traffic).

Shroud circuits are used for:
- Specter Wave propagation
- Anonymous game participation
- Phantom Gifts and Marks
- Fortress mode transport

### Is MURMUR as anonymous as Tor?

**No.** Shroud provides onion routing with three hops, but it does **not** defend against global passive adversaries or state-level traffic analysis.

**For users requiring Tor-level anonymity:**
- Use Fortress mode (default-anonymous)
- Route MURMUR traffic through Tor or I2P (planned for v0.2)
- Avoid correlating Surface and Specter identities through timing or behavior

See [SECURITY_PRIVACY.md](../SECURITY_PRIVACY.md) for the complete threat model.

---

## Content & Waves

### What is a Wave?

A **Wave** is a signed, ephemeral content unit (≤2048 bytes UTF-8 text) with:
- SHA-256 Proof of Work (2-5 second computation)
- Configurable TTL (default 7 days, max 30 days)
- GossipSub propagation across the mesh
- Automatic expiration and garbage collection

### Why are Waves limited to 2048 bytes?

1. **Performance** — Small messages propagate faster across the mesh
2. **Proof of Work** — Shorter messages make PoW feasible on consumer hardware
3. **Design Philosophy** — Forces concise communication (like early Twitter's 140 chars)

### Can I edit or delete a Wave after publishing?

**No.** Waves are immutable once published. You cannot edit or delete them.

However, Waves **expire automatically** after their TTL, and the network forgets them (no permanent archive).

### What are the 8 Wave types?

1. **Surface Wave (0x01)** — Public message from Surface identity
2. **Reply Wave (0x02)** — Threaded reply to another Wave
3. **Veiled Wave (0x03)** — Hybrid mode (Surface origin, Specter recipients)
4. **Specter Wave (0x04)** — Anonymous message from Specter identity
5. **Sigil Wave (0x05)** — Visual identity update (sigil customization)
6. **Abyssal Wave (0x06)** — Fortress-only (maximum anonymity)
7. **Masked Wave (0x07)** — Temporary event-bound identity
8. **Beacon Wave (0x08)** — Relay advertisement for Shroud circuits

### Can I send images, videos, or voice messages?

**Not in v0.1.0-rc1.** Waves are text-only.

Rich media support (images, audio, video) is planned for v0.2+.

### How long do Waves last?

Waves expire after their **TTL (Time To Live)**:
- Default: 7 days
- Maximum: 30 days
- Minimum: 1 day

After expiration, Waves are garbage-collected from local caches and stop propagating.

---

## Networking & Connectivity

### Do I need to open ports or configure my firewall?

**Usually not.** MURMUR uses **NAT traversal** (DCUtR hole punching) to establish connections through most NATs.

**If you have symmetric NAT:**
- You may need to forward a port (default: 4001 for libp2p)
- Or connect via a relay (automatic fallback)

### What is a relay?

A **relay** is a peer that forwards traffic for nodes behind restrictive NATs. MURMUR uses libp2p's Circuit Relay v2 protocol.

In v0.1.0-rc1, relays must be manually configured. Automated relay discovery is planned for v0.1.0-final.

### How do I bootstrap (join the network)?

Two methods:

1. **Invitation Link** (Recommended) — Someone already on MURMUR gives you a `murmur://invite/` link; you bootstrap directly through their node
2. **Public DHT** — Connect to hardcoded bootstrap nodes and discover peers via Kademlia DHT

Invitation links provide a **warm start** (you appear in your friend's neighborhood on the Pulse Map).

### What happens if I'm the first person to run MURMUR?

You'll connect to **bootstrap nodes** (hardcoded in the binary) to discover other peers. If the network is very small, you may see only a few nodes initially.

**For testing with friends:**
1. One person launches MURMUR and generates an invitation link
2. Others use that link to bootstrap through the first person's node
3. You'll all appear in the same region of the Pulse Map

### Can MURMUR work offline?

**No.** MURMUR is a peer-to-peer network; you need an internet connection to participate.

However, you can:
- View cached Waves while offline
- Explore your local Pulse Map snapshot
- Compose Waves (they'll be published when you reconnect)

---

## Pulse Map & Navigation

### What is the Pulse Map?

The **Pulse Map** is MURMUR's primary interface — a real-time force-directed graph where:
- **Nodes** = users (you and your connections)
- **Edges** = relationships (connections between users)
- **Ripples** = content propagating through the mesh
- **Colors/glows** = activity, privacy mode, Resonance level

The Pulse Map is **spatial** — you navigate by panning/zooming, not scrolling a feed.

### How do I navigate the Pulse Map?

- **Pan**: Click and drag (left mouse button) or arrow keys
- **Zoom**: Mouse wheel or `+`/`-` keys
- **Center on node**: Double-click a node
- **Select node**: Single-click a node (shows profile panel)
- **Switch to Anonymous Layer**: Press `Tab` key

### What do the colors mean?

- **Blue** — Surface identity (Open mode)
- **Purple** — Hybrid mode (Surface + Specter)
- **Green** — Guarded mode (default-anonymous)
- **Red** — Fortress mode (maximum anonymity)
- **Gray** — Specter identity (Anonymous Layer)

Glow intensity indicates recent activity.

### How many nodes can the Pulse Map handle?

**Performance targets:**
- 500 nodes @ 60fps (validated in v0.1.0-rc1)
- 1000 nodes @ 30fps (planned optimization for v0.1.0-final)

The layout engine uses **Barnes-Hut approximation** for >500 nodes to maintain performance.

---

## Anonymous Layer & Mini-Games

### What is Resonance?

**Resonance** is MURMUR's anonymous reputation metric. It measures participation in the Anonymous Layer (Specter activities, mini-games, Phantom Gifts).

Resonance is:
- **Locally computed** (no global leaderboard)
- **Non-transferable** (tied to your Specter identity)
- **Non-financialized** (no tokens or payments)

### How do I earn Resonance?

- Publish Specter Waves
- Participate in mini-games
- Send Phantom Gifts
- Place Specter Marks
- Complete Cipher Puzzles
- Win Territory Drift competitions

### What are the Resonance milestones?

1. **Shade (25)** — Unlock Phantom Gifts
2. **Wraith (50)** — Unlock Specter Marks
3. **Shade-Wraith (75)** — Unlock mini-game tournaments
4. **Phantom (100)** — Unlock Masked Events
5. **Council-Eligible (200)** — Unlock Phantom Councils
6. **Abyss (500)** — Fortress-mode only (maximum anonymity)

### What are the 10 mini-games?

1. **Cipher Puzzles** — Decrypt challenges for Resonance
2. **Specter Hunt** — Find hidden Specters on the Pulse Map
3. **Territory Drift** — Claim regions via Resonance density
4. **Oracle Pools** — Collaborative prediction markets
5. **Sigil Forge** — Visual identity customization
6. **Shadow Play** — Real-time anonymous charades
7. **Phantom Councils** — Anonymous governance (Council-eligible only)
8. **Masked Events** — Temporary ultra-anonymous spaces
9. **Whisper Chains** — Multi-hop encrypted message games
10. **Echo Index** — Recursive reputation verification

See [ANONYMOUS_GAME_MECHANICS.md](../ANONYMOUS_GAME_MECHANICS.md) for detailed rules.

### What is a Phantom Gift?

A **Phantom Gift** is a one-way anonymous gift from a Specter to a Surface identity. Gifts appear as glowing cosmetic effects on the recipient's Pulse Map node (e.g., halos, trails, particle effects).

Recipients **cannot** identify the sender (true anonymity). Gifts require Resonance ≥25 (Shade milestone).

---

## Troubleshooting

### MURMUR won't connect to any peers

**Possible causes:**
1. **Firewall blocking libp2p** — Allow port 4001 (default)
2. **No bootstrap nodes reachable** — Check your internet connection
3. **Symmetric NAT** — Enable relay fallback (automatic in v0.1.0-rc1)

**Solutions:**
- Check logs: `~/.murmur/murmur.log`
- Try using an invitation link (bypasses DHT discovery)
- Forward port 4001 (TCP/UDP) on your router

### Pulse Map rendering is slow or choppy

**Possible causes:**
1. **Too many nodes** — Performance degrades >500 nodes
2. **Integrated GPU** — Ebitengine prefers dedicated GPU
3. **High DPI display** — Rendering at 4K can be expensive

**Solutions:**
- Reduce window size (Settings → Display)
- Disable visual effects (Settings → Effects → Off)
- Update graphics drivers

### I forgot my passphrase

**Unfortunately, there is no recovery.** Your keystore files are encrypted with Argon2id; there is no backdoor.

**If you saved your BIP-39 recovery phrase:**
1. Reinstall MURMUR
2. During onboarding, choose "Recover from phrase"
3. Enter your 24-word recovery phrase
4. Create a new passphrase

**If you did not save your recovery phrase:**
- Your keys are lost
- You must create a new identity
- This is by design (self-sovereign identity)

### My Waves aren't propagating

**Possible causes:**
1. **Insufficient mesh density** — Need ≥6 peers for reliable gossip
2. **Proof of Work failure** — PoW computation timed out
3. **Low Resonance** — Some Wave types require Resonance thresholds

**Solutions:**
- Check peer count (status bar)
- Increase PoW timeout (Settings → Content → PoW Timeout)
- Build Resonance through participation

### How do I report a bug?

**GitHub Issues:** [github.com/opd-ai/murmur/issues](https://github.com/opd-ai/murmur/issues)

Please include:
1. **Platform** (Linux/macOS/Windows, amd64/arm64)
2. **Version** (check Menu → About)
3. **Steps to reproduce**
4. **Logs** (`~/.murmur/murmur.log`)

---

## Security & Threat Model

### Is my traffic encrypted?

**Yes.** All libp2p connections use **Noise XX** transport encryption (similar to TLS).

Additionally:
- Shroud circuits use **three layers of onion encryption** (XChaCha20-Poly1305)
- Keystores are encrypted with **Argon2id + XChaCha20-Poly1305**

### Can my ISP see what I'm doing on MURMUR?

Your ISP can see:
- You're using libp2p (encrypted P2P traffic)
- Your peer IPs (connection metadata)

Your ISP **cannot** see:
- Wave content (encrypted)
- Your Surface or Specter identities
- Who you're talking to (obfuscated via GossipSub + Shroud)

For maximum metadata protection, use **Fortress mode** and route through Tor/I2P (planned v0.2).

### What's the threat model?

**In Scope (MURMUR Defends Against):**
- Network-level observers correlating IPs with identities
- Malicious peers attempting spam, flooding, or metadata inference
- Platform-style deanonymization (mitigated by having no servers)

**Out of Scope (Use Tor/I2P Instead):**
- Global passive adversaries
- State-level traffic analysis
- Adversaries controlling majority of relays

See [SECURITY_PRIVACY.md](../SECURITY_PRIVACY.md) for the complete threat model.

---

## Contributing & Development

### How can I contribute?

Key contribution areas:
- **Tor/I2P Integration** — Bridge Shroud with Tor/I2P
- **Relay Discovery** — Automated Shroud relay discovery
- **Mobile Ports** — Android/iOS support
- **Performance** — Optimize Pulse Map for 1000+ nodes
- **Anonymous Mechanics** — New mini-games, Phantom Councils

See [CONTRIBUTING.md](../CONTRIBUTING.md) for code style and PR process.

### Where is the protocol specification?

Complete specification: [DESIGN_DOCUMENT.md](../DESIGN_DOCUMENT.md)  
Technical details: [TECHNICAL_IMPLEMENTATION.md](../TECHNICAL_IMPLEMENTATION.md)

### Is there a protocol extension API?

Yes. MURMUR is designed as a protocol with an **extension surface** for third-party networks. See [EXTENSION_CONTRACT.md](../EXTENSION_CONTRACT.md) for details.

---

## Roadmap

### What's next after v0.1.0-rc1?

**v0.1.0 Final** (Q2 2026)
- Bug fixes from RC1 feedback
- Relay discovery protocol
- Performance optimization (1000-node target)

**v0.2.0** (Q3 2026)
- Tor transport integration
- I2P transport integration
- Multi-device identity sync
- Rich media support (images, audio)

**v1.0.0** (Q4 2026)
- Protocol stabilization
- Extension contract finalization
- Mobile builds (Android/iOS)
- Production-ready relay network

See [ROADMAP.md](../ROADMAP.md) for the complete plan.

---

## Additional Resources

- **Website**: [murmurnet.org](https://murmurnet.org) *(placeholder — not yet live)*
- **GitHub**: [github.com/opd-ai/murmur](https://github.com/opd-ai/murmur)
- **Documentation**: `docs/` directory in the repository
- **Developer Chat**: Join the MURMUR network, find the `#dev` Wave thread

---

**Still have questions?** Ask on [GitHub Discussions](https://github.com/opd-ai/murmur/discussions) or join the MURMUR network!
