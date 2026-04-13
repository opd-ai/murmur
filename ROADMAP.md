# Goal-Achievement Assessment

## Project Context

### What It Claims To Do

MURMUR claims to be a **decentralized, peer-to-peer social network with dual-layer identity** offering:

1. **No servers, no algorithms, no permanent record** — fully peer-to-peer architecture where every device is both client and server
2. **Pulse Map visualization** — force-directed graph where users navigate a spatial social topology instead of scrolling algorithmic feeds
3. **Self-sovereign identity** — Ed25519 keypairs, no email/phone/third-party verification required
4. **Ephemeral content** — Waves expire after configurable TTL (default 7 days, max 30 days)
5. **Anonymous Layer (Specters)** — pseudonymous identities routed through onion-style Shroud circuits, cryptographically unlinkable from main identity
6. **No engagement metrics** — no likes, no follower counts, no algorithmic amplification
7. **Resonance reputation system** — locally-computed reputation with milestone unlocks at 25/50/75/100/200
8. **Anonymous mini-games** — Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Phantom Councils
9. **Four privacy modes** — Open, Hybrid, Guarded, Fortress (escalating anonymity guarantees via Shadow Gradient)
10. **Six-phase onboarding flow** — guided introduction from first launch to active participation
11. **libp2p foundation** — GossipSub for message propagation, Kademlia DHT for peer discovery, NAT traversal via DCUtR

### Target Audience

- Privacy-conscious users seeking social networking without corporate intermediation
- People wanting self-sovereign identity without account creation requirements
- Communities interested in anonymous social mechanics as a first-class feature
- Users who value ephemeral, topology-based content propagation over algorithmic feeds

### Architecture (Documented)

| Subsystem | Description | Documented In |
|-----------|-------------|---------------|
| **Networking** | libp2p transport, GossipSub, Kademlia DHT, NAT traversal | NETWORK_ARCHITECTURE.md, DESIGN_DOCUMENT.md |
| **Identity** | Ed25519/Curve25519 keypairs, sigils, privacy modes | SHADOW_GRADIENT.md, DESIGN_DOCUMENT.md |
| **Content** | Waves with PoW, TTL, threading, amplification | WAVE_PROPAGATION.md, WAVES.md |
| **Anonymous Layer** | Specters, Shroud onion routing, Resonance, mini-games | SECURITY_PRIVACY.md, RESONANCE_SYSTEM.md, ANONYMOUS_GAME_MECHANICS.md |
| **Pulse Map** | Force-directed graph visualization | PULSE_MAP.md |
| **Onboarding** | Six-phase guided introduction | ONBOARDING.md, VIRAL_GROWTH_AND_ONBOARDING.md |

### Existing CI/Quality Gates

- **None** — no CI configuration, no Makefile, no test infrastructure

### Codebase Status

| Artifact | Present | Evidence |
|----------|---------|----------|
| Go source files | ❌ None | No `*.go` files exist |
| `go.mod` / `go.sum` | ❌ None | No module definition exists |
| Tests | ❌ None | No test files exist |
| Build scripts | ❌ None | No Makefile, no CI workflows |
| Documentation | ✅ Comprehensive | ~552KB across 15 markdown files |

The README explicitly states: *"Pre-implementation. The design document is complete. Everything else is ahead."*

---

## Goal-Achievement Summary

| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| Decentralized P2P social network | ❌ Missing | No Go code exists | Complete implementation required |
| Pulse Map force-directed graph UI | ❌ Missing | No visualization code | Needs full UI/rendering implementation |
| Ed25519 cryptographic identity | ❌ Missing | No crypto code | Key generation, storage, signing needed |
| Ephemeral Waves with PoW/TTL | ❌ Missing | No content system | Message format, PoW, propagation, expiration |
| Anonymous Layer (Specters) | ❌ Missing | No anonymous identity code | Separate keypairs, unlinkability, lifecycle |
| Shroud onion routing | ❌ Missing | No circuit construction | Three-hop onion encryption, relay protocol |
| Resonance reputation system | ❌ Missing | No reputation computation | Local scoring algorithm, milestone unlocks |
| Anonymous mini-games | ❌ Missing | No game mechanics | 6+ distinct mini-game implementations |
| Privacy modes (Open/Hybrid/Guarded/Fortress) | ❌ Missing | No mode switching | Mode transitions, traffic routing per mode |
| Six-phase onboarding flow | ❌ Missing | No UI code | Phase sequence with animations |
| libp2p networking stack | ❌ Missing | No dependencies | GossipSub, DHT, NAT traversal setup |
| No engagement metrics | ✅ Achieved | By design absence | Principle maintained by not implementing metrics |

**Overall: 1/12 goals achieved (trivially, by design default)**

---

## Competitive Landscape

MURMUR positions itself against existing decentralized social platforms:

| Platform | Model | MURMUR's Claimed Differentiation |
|----------|-------|----------------------------------|
| **Mastodon** | Federated servers (ActivityPub), 10-15M+ users | Fully P2P (no instances), spatial UI, integrated anonymity |
| **Bluesky** | Federated (AT Protocol), 40M+ users | True P2P from day one, no central org dependency |
| **Scuttlebutt** | P2P gossip, append-only logs | Anonymous layer, visual spatial UI, gamified mechanics |
| **Matrix** | Federated chat servers | Social-first design (not chat-focused), spatial topology |
| **Session** | Onion-routed messaging | Rich social mechanics beyond messaging |

**Unique differentiators if implemented**:
- **Shadow Gradient** — tiered anonymity as architectural foundation
- **Pulse Map** — spatial topology navigation replacing feeds
- **Anonymous mini-games** — gamified anonymous social mechanics
- **Cross-layer visibility** — anonymous effects visible to clearnet users

---

## Dependency Landscape (2025-2026)

### go-libp2p (v0.48.0 latest, March 2026)

**Status**: Active development, but **Shipyard ended primary maintenance September 2025**. Community stewardship transitioning.

**Breaking Changes**:
- v0.47.0 (Jan 2026): WebTransport handshake changes (experimental)
- AllAddrs() behavior changed

**Risk**: go-libp2p maintenance transition creates uncertainty. MURMUR should:
- Track community governance decisions
- Pin to stable versions (v0.48.x)
- Avoid experimental features (WebTransport) initially

### Ebitengine (v2.10.0, October 2025)

**Status**: Actively maintained, regular releases through 2026.

**Relevant features**:
- text/v2 package for internationalized text
- Kage shaders for visual effects
- Cross-platform: Windows, macOS, Linux, Web (WASM), iOS, Android

**Risk**: Low — stable project with clear upgrade path.

---

## Roadmap

### Priority 1: Establish Codebase Foundation

**Goal**: Transform documentation-only project into buildable Go codebase.

**Why First**: Nothing can be built without a module definition, dependencies, and package structure. This unblocks all subsequent work.

- [ ] Create `go.mod` with module path `github.com/opd-ai/murmur` and Go 1.22+
- [ ] Add foundational dependencies:
  - `github.com/libp2p/go-libp2p` v0.48+ — P2P networking foundation
  - `github.com/libp2p/go-libp2p-pubsub` — GossipSub for Wave propagation
  - `github.com/libp2p/go-libp2p-kad-dht` — Kademlia peer discovery
  - `golang.org/x/crypto` — Ed25519, Curve25519, ChaCha20-Poly1305, Argon2
  - `github.com/zeebo/blake3` — BLAKE3 hashing for identity/sigils
  - `go.etcd.io/bbolt` — embedded key-value storage
  - `google.golang.org/protobuf` — wire format serialization
- [ ] Create package structure per documented architecture:
  ```
  cmd/murmur/main.go              # Entry point
  proto/                           # .proto files
  pkg/
  ├── app/                         # Application lifecycle, event bus
  ├── config/                      # Configuration
  ├── networking/
  │   ├── transport/               # libp2p host
  │   ├── gossip/                  # GossipSub
  │   ├── discovery/               # Kademlia DHT
  │   ├── relay/                   # NAT traversal
  │   └── mesh/                    # Peer management
  ├── identity/
  │   ├── keys/                    # Ed25519/Curve25519
  │   ├── sigils/                  # Visual identity
  │   ├── declarations/            # Profile declarations
  │   └── modes/                   # Privacy modes
  ├── content/
  │   ├── waves/                   # Wave creation/validation
  │   ├── pow/                     # Proof of Work
  │   ├── propagation/             # Gossip relay
  │   ├── threads/                 # Reply chains
  │   └── storage/                 # Local cache
  ├── anonymous/
  │   ├── specters/                # Specter identities
  │   ├── shroud/                  # Onion routing
  │   ├── resonance/               # Reputation
  │   └── mechanics/               # Mini-games
  ├── store/                       # Bbolt operations
  └── pulsemap/
      ├── layout/                  # Force-directed graph
      ├── rendering/               # Ebitengine Draw()
      ├── interaction/             # Input handling
      └── overlays/                # Anonymous layer
  ```
- [ ] Add CI workflow (`.github/workflows/ci.yml`) with `go build`, `go test`, `go vet`
- [ ] Create placeholder `main.go` with basic libp2p node initialization
- [ ] **Validation**: `go build ./...` succeeds; CI passes on scaffold

**Estimated Effort**: 1-2 days

---

### Priority 2: Implement Identity System

**Goal**: Achieve claimed self-sovereign identity with Ed25519 keypairs.

**Why Second**: Identity is the foundation for all cryptographic operations — signing Waves, establishing connections, creating Specters.

- [ ] Implement `pkg/identity/keys/keypair.go`:
  - Ed25519 keypair generation via `crypto/ed25519`
  - Curve25519 derivation for key agreement (birational map)
  - Peer ID derivation (SHA-256 hash truncated to 160 bits)
- [ ] Implement `pkg/identity/keys/keystore.go`:
  - Encrypted keystore with Argon2id key derivation (time=3, memory=64MiB, threads=4)
  - XChaCha20-Poly1305 encryption for key material
  - Secure memory handling (explicit zeroing of byte slices)
- [ ] Implement `pkg/identity/keys/backup.go`:
  - BIP-39 mnemonic encoding/decoding for backup
  - Recovery flow from mnemonic phrase
  - Key export/import for device migration
- [ ] Implement `pkg/identity/sigils/generator.go`:
  - Deterministic visual identity from BLAKE3 hash of public key
  - Parametric generation (shapes, colors, patterns)
  - 64×64 raster output for cross-platform consistency
- [ ] Write comprehensive unit tests:
  - Keypair generation randomness verification
  - Round-trip encryption/decryption
  - Mnemonic backup/restore accuracy
  - Sigil determinism (same key → same sigil)
- [ ] **Validation**: `go test ./pkg/identity/...` passes; backup recovery restores identical keypair

**Estimated Effort**: 1-2 weeks

---

### Priority 3: Build Networking Layer

**Goal**: Achieve peer-to-peer mesh with gossip propagation.

**Why Third**: Networking enables communication between nodes — all content propagation depends on this.

- [ ] Implement `pkg/networking/transport/host.go`:
  - libp2p host initialization with Noise encryption
  - Multi-transport support (QUIC preferred, TCP fallback)
  - Peer ID from identity keypair
- [ ] Implement `pkg/networking/discovery/dht.go`:
  - Kademlia DHT bootstrap to hardcoded seed nodes
  - Peer discovery via DHT walks
  - Peer exchange during gossip for redundancy
- [ ] Implement `pkg/networking/gossip/pubsub.go`:
  - GossipSub subscription to topics:
    - `/murmur/waves/1` — standard content
    - `/murmur/identity/1` — identity declarations
    - `/murmur/shroud/1` — Shroud relay advertisements
    - `/murmur/pulse/1` — heartbeat pings
  - Message validation hooks for signature/PoW verification
  - Peer scoring for invalid message penalties
- [ ] Implement `pkg/networking/mesh/manager.go`:
  - Connection manager with 6-12 peer target
  - Priority tiers: identity connections > useful gossip peers > random peers
  - 30-second heartbeat with 3-miss disconnect
- [ ] Implement `pkg/networking/relay/nat.go`:
  - DCUtR hole punching coordination
  - Relay fallback for double-NAT scenarios
  - AutoNAT probing at startup
- [ ] Write integration tests:
  - Two-node in-memory discovery and message exchange
  - Multi-node gossip propagation simulation
- [ ] **Validation**: Two nodes exchange signed test messages within 10 seconds

**Estimated Effort**: 2-3 weeks

---

### Priority 4: Implement Wave Content System

**Goal**: Achieve ephemeral, signed content with PoW and TTL enforcement.

**Why Fourth**: Waves are the atomic content unit — users need to create and consume content for the network to have value.

- [ ] Define protobuf schema (`proto/wave.proto`):
  - Wave structure per spec: ID, author_pubkey, content, timestamp, parent_hash, reference_hash, ttl_hours, nonce, signature
  - MurmurEnvelope wrapper with version, type, payload, signature, message_id
  - 2,048 byte content limit enforcement
- [ ] Implement `pkg/content/pow/proof.go`:
  - SHA-256 PoW with difficulty 20 (target 2-5 second computation)
  - Nonce search with progress callback for UI
  - Device capability detection for difficulty adjustment
- [ ] Implement `pkg/content/waves/wave.go`:
  - Wave composition with automatic PoW and signing
  - Wave validation (signature, PoW, timestamp sanity ±300s)
  - Wave types: Surface (0x01), Reply (0x02), Veiled (0x03)
- [ ] Implement `pkg/content/propagation/relay.go`:
  - GossipSub broadcast to `/murmur/waves/1`
  - Hop count tracking (max 6 hops)
  - Duplicate detection via BLAKE3 hash cache (24-hour retention)
- [ ] Implement `pkg/content/storage/cache.go`:
  - LRU cache with configurable size limit (default 100MB)
  - Hourly TTL sweep for expired Waves (GC every 60s)
  - Bbolt persistence for offline access
- [ ] Implement `pkg/content/threads/threading.go`:
  - Reply chain construction via parent_hash references
  - Thread traversal and display ordering
- [ ] Write tests:
  - PoW difficulty calibration verification
  - TTL expiration accuracy
  - Propagation hop counting
  - Duplicate rejection
- [ ] **Validation**: Wave published on node A arrives at node B via gossip within 5 seconds; expired Waves are garbage collected

**Estimated Effort**: 2-3 weeks

---

### Priority 5: Implement Anonymous Layer Foundation

**Goal**: Achieve cryptographically unlinkable Specter identities.

**Why Fifth**: The anonymous layer is MURMUR's primary differentiator — it enables the Shadow Gradient that makes the network compelling.

- [ ] Implement `pkg/anonymous/specters/specter.go`:
  - Independent Curve25519 keypair generation (no derivation from main identity)
  - Separate encrypted keystore partition for Specter keys
  - Specter lifecycle: creation, suspension, deletion
- [ ] Implement `pkg/anonymous/specters/name.go`:
  - Procedural pseudonym generation from BLAKE3 hash
  - Two-word format: adjective + noun from curated wordlist (65,536 entries)
  - Collision avoidance via hash prefix variation
- [ ] Implement `pkg/anonymous/specters/sigil.go`:
  - Distinct visual style from Surface sigils (spectral glow, different shapes)
  - Procedural generation from Specter public key
  - Resonance-tier cosmetic variations
- [ ] Implement `pkg/identity/modes/mode.go`:
  - Mode enumeration: Open, Hybrid, Guarded, Fortress
  - Mode switching state machine
  - Mode-aware message routing (Surface vs Anonymous topics)
- [ ] Write tests:
  - Keypair independence verification (no mathematical link)
  - Mode transition state consistency
  - Specter name uniqueness probability
- [ ] **Validation**: Generated Specter keypair has no derivation relationship to main keypair

**Estimated Effort**: 2 weeks

---

### Priority 6: Implement Shroud Onion Routing

**Goal**: Achieve three-hop onion circuits for anonymous traffic.

**Why Sixth**: Shroud routing provides the traffic analysis resistance that makes Guarded/Fortress modes meaningful.

- [ ] Implement `pkg/anonymous/shroud/beacon.go`:
  - Shroud Node announcement on `/murmur/shroud/1`
  - Capability advertisement (bandwidth, uptime)
  - Discovery via DHT and beacon subscription
- [ ] Implement `pkg/anonymous/shroud/circuit.go`:
  - Three-hop relay chain selection
  - Diversity heuristics (no two hops in direct mesh)
  - Circuit handshake with X25519 key agreement
  - 10-minute circuit rotation
- [ ] Implement `pkg/anonymous/shroud/onion.go`:
  - Triple-layer Curve25519 key agreement
  - XChaCha20-Poly1305 for each hop layer
  - Fixed 1KB packet padding
  - Decryption at each relay hop
- [ ] Implement `pkg/anonymous/shroud/relay.go`:
  - Shroud Node message forwarding
  - Traffic mixing (random delay injection)
  - Dummy traffic generation (constant-rate padding)
- [ ] Write simulation tests:
  - Passive traffic analysis resistance
  - Circuit reconstruction on relay failure
- [ ] **Validation**: Anonymous Wave cannot be correlated to origin by passive observer in 100-node simulation

**Estimated Effort**: 3-4 weeks

---

### Priority 7: Implement Resonance Reputation

**Goal**: Achieve local reputation computation with milestone unlocks.

**Why Seventh**: Resonance gates access to anonymous mechanics — it's the progression system that creates engagement.

- [ ] Implement `pkg/anonymous/resonance/scorer.go`:
  - Local scoring algorithm per RESONANCE_SYSTEM.md spec
  - Four signal categories:
    - Publication consistency (regular Specter activity)
    - Mini-game quality (puzzle solutions, duel outcomes)
    - Gift activity (given and received)
    - Community endorsement (marks from high-Resonance Specters)
- [ ] Implement `pkg/anonymous/resonance/decay.go`:
  - Time decay for inactive Specters (half-life curve)
  - Activity refresh mechanics
- [ ] Implement `pkg/anonymous/resonance/milestones.go`:
  - Milestone thresholds: 25 (Shade), 50 (Wraith), 75 (Shade-Wraith), 100 (Phantom), 200 (Council-Eligible), 500 (Abyss)
  - Unlock tracking and notification
  - Rank display cosmetics per tier
- [ ] Implement `pkg/anonymous/resonance/claims.go`:
  - Zero-knowledge Resonance claims (Pedersen commitments + Bulletproofs)
  - Claim verification without scorer access
  - Claim freshness and replay prevention
- [ ] Write tests:
  - Scoring accuracy for activity patterns
  - Decay rate verification
  - ZK claim verification
- [ ] **Validation**: 7 days of consistent daily activity reaches Resonance 25

**Estimated Effort**: 2 weeks

---

### Priority 8: Implement Core Anonymous Mechanics

**Goal**: Achieve cross-layer social mechanics that drive Shadow Gradient adoption.

**Why Eighth**: These mechanics make the anonymous layer visible and compelling to Surface users.

- [ ] Implement `pkg/anonymous/mechanics/gifts.go`:
  - Phantom Gift creation (Resonance 25+ required)
  - Visual effect catalog (particle trails, glows, shimmers)
  - Gift application to Surface Layer nodes
  - 24-hour expiration
- [ ] Implement `pkg/anonymous/mechanics/marks.go`:
  - Specter Mark placement (Resonance 50+ required)
  - Mark categories (Watcher, Ally, Rival)
  - Persistent storage with 30-day decay
- [ ] Implement `pkg/anonymous/mechanics/territory.go`:
  - Territory Drift influence accumulation
  - Controller status at threshold
  - Weekly territory reset
- [ ] Implement `pkg/anonymous/mechanics/puzzles.go`:
  - Fragment Puzzle (basic Cipher Puzzle variant)
  - Puzzle generation from content hash
  - Solution verification and Resonance reward
- [ ] Write tests:
  - Gift application and expiration
  - Mark persistence and decay
  - Territory influence calculation
- [ ] **Validation**: Phantom Gift from Resonance 25+ Specter appears on recipient's Surface node

**Estimated Effort**: 3 weeks

---

### Priority 9: Build Pulse Map Visualization

**Goal**: Achieve spatial topology navigation as primary interface.

**Why Ninth**: The Pulse Map is the "face" of MURMUR — without it, there's no differentiation from text-based social apps.

- [ ] Add Ebitengine v2.10+ dependency
- [ ] Implement `pkg/pulsemap/layout/force.go`:
  - Force-directed graph layout (Fruchterman-Reingold algorithm)
  - Barnes-Hut optimization for O(n log n) performance (>500 nodes)
  - Incremental layout updates for node add/remove
  - Double-buffered positions with atomic.Pointer swaps
- [ ] Implement `pkg/pulsemap/rendering/nodes.go`:
  - Node rendering with sigil integration
  - Node size scaled by connection count
  - Activity pulse animation (recent Waves)
- [ ] Implement `pkg/pulsemap/rendering/edges.go`:
  - Connection edge rendering
  - Edge thickness by relationship strength
- [ ] Implement `pkg/pulsemap/rendering/effects.go`:
  - Wave propagation ripple effects (ripple.kage shader)
  - Phantom Gift particle trails
  - Node activity glow (glow.kage shader)
- [ ] Implement `pkg/pulsemap/overlays/anonymous.go`:
  - Ghost layer for Anonymous Layer visualization
  - Faint Specter nodes with spectral effects (spectra.kage)
- [ ] Implement `pkg/pulsemap/interaction/navigation.go`:
  - Pan, zoom (mouse and touch)
  - Node selection and profile expansion
  - Momentum scrolling
- [ ] Write visual regression tests:
  - Layout stability for fixed graph
  - Memory usage under large graphs
- [ ] **Validation**: 500-node graph renders at 60fps with smooth pan/zoom

**Estimated Effort**: 4-6 weeks

---

### Priority 10: Implement Onboarding Flow

**Goal**: Achieve guided introduction from first launch to active participation in under 5 minutes.

**Why Tenth**: Onboarding determines whether new users convert or churn.

- [ ] Implement `pkg/onboarding/flow/controller.go`:
  - Phase state machine: Welcome → Identity → Mode → Bootstrap → Exploration → Complete
  - Progress persistence for interrupted flows
- [ ] Implement Phase 1-2: Welcome + Identity Creation
  - Animated node visualization
  - Keypair generation ceremony with visual feedback
  - Display name selection and sigil reveal
  - Backup prompt (optional but emphasized)
- [ ] Implement Phase 3: Mode Selection
  - Animated Shadow Gradient visualization
  - Mode cards with capability comparison
  - Default recommendation (Hybrid)
- [ ] Implement Phase 4-5: Bootstrap + Exploration
  - Peer discovery progress indicator
  - Pulse Map introduction tour
  - First Wave prompt with guided composition
- [ ] Implement Phase 6: Completion
  - Summary of created identity
  - Invitation generation prompt
- [ ] Write flow tests:
  - Full path completion timing (<5 minutes)
  - State recovery after interruption
- [ ] **Validation**: New user completes onboarding in <5 minutes with formed identity and 1+ peer connection

**Estimated Effort**: 3 weeks

---

### Priority 11: Advanced Anonymous Mechanics

**Goal**: Complete the mini-game ecosystem for long-term engagement.

**Why Eleventh**: These mechanics provide depth and replayability.

- [ ] Implement Specter Hunts (network-wide scavenger hunts)
- [ ] Implement Oracle Pools (commit-reveal prediction markets)
- [ ] Implement Sigil Forge (timed creative challenges)
- [ ] Implement Shadow Play (anonymous social deduction)
- [ ] Implement Phantom Councils (threshold-signed anonymous deliberation)
- [ ] Implement Abyssal Waves (Fortress-exclusive 5-hop deep anonymity)
- [ ] **Validation**: Each mini-game completes end-to-end in multi-node simulation

**Estimated Effort**: 6-8 weeks

---

### Priority 12: Production Hardening

**Goal**: Achieve production-ready quality and operational stability.

**Why Last**: Hardening is valuable only after features are implemented.

- [ ] Error Handling:
  - User-friendly error messages
  - Graceful degradation (Shroud unavailable → Hybrid fallback)
  - Automatic retry with exponential backoff
- [ ] Resource Management:
  - Memory profiling (<256MiB during normal operation)
  - Connection limit enforcement
  - Bandwidth throttling options (~50KB/s sustained)
- [ ] Security Audit:
  - Cryptographic implementation review
  - Shroud routing privacy analysis
  - Key storage security assessment
  - Dependency vulnerability scan
- [ ] Testing Expansion:
  - Fuzz testing for message parsing
  - Chaos testing for network partitions
  - Long-running stability tests (72+ hours)
  - 10-100 node simulation tests (`//go:build simulation`)
- [ ] Documentation:
  - Deployment guide (desktop, mobile)
  - Bootstrap node operation manual
  - Shroud Node operation guide
- [ ] **Validation**: 72-hour simulation with 1000 nodes shows no memory leaks, panics, or deadlocks

**Estimated Effort**: 4-6 weeks

---

## Implementation Notes

### Technical Decisions Required

1. **UI Framework**: Ebitengine v2.10+ (per TECHNICAL_IMPLEMENTATION.md specification)
2. **Sigil Generation**: Hybrid approach — deterministic raster for cross-platform, shader enhancement for effects
3. **PoW Difficulty**: Adaptive based on device capability detection, targeting 2-5 second computation
4. **Shroud Node Incentives**: Resonance bonus for relay operation (to mitigate capacity risk)
5. **Bootstrap Strategy**: Hardcoded list + user-configurable + peer exchange fallback

### Dependency Risk Assessment

| Dependency | Risk Level | Mitigation |
|------------|------------|------------|
| go-libp2p | Medium (maintenance transition) | Pin to v0.48.x, track community governance |
| Ebitengine | Low | Active maintenance, stable API |
| golang.org/x/crypto | Low | Standard library, long-term support |
| bbolt | Low | Mature, embedded, minimal surface |
| zeebo/blake3 | Low | Pure Go, no CGO, widely used |

### Success Milestones

| Version | Milestone | Definition of Done |
|---------|-----------|-------------------|
| v0.1 | Foundation | `go build ./...` succeeds, two nodes exchange test messages |
| v0.2 | Identity | Users can create, backup, and restore identity keypairs |
| v0.3 | Content | Waves propagate with PoW validation and TTL expiration |
| v0.4 | Anonymous | Specters can publish cryptographically unlinkable Waves |
| v0.5 | Routing | Shroud circuits provide traffic analysis resistance |
| v0.6 | Reputation | Resonance scoring with milestone unlocks working |
| v0.7 | Mechanics | Core anonymous mechanics (Gifts, Marks, Territory) functional |
| v0.8 | Visualization | Pulse Map renders topology with basic interaction |
| v0.9 | Onboarding | Complete guided flow from launch to participation |
| v1.0 | MVP | End-to-end Surface + Anonymous participation works reliably |

---

## Conclusion

MURMUR currently exists as a **comprehensive design specification** (~552KB across 15 markdown files) with **zero implementation**. The documentation is exceptionally thorough, covering cryptographic primitives, network protocols, privacy architecture, and social mechanics in detail.

The gap between vision and reality is **complete** — no code exists. This roadmap prioritizes implementation in order of:

1. **Foundational capability** (identity, networking, content) — enables basic functionality
2. **Differentiating features** (anonymous layer, Shroud routing, Resonance) — creates unique value
3. **User experience** (Pulse Map, onboarding, mini-games) — makes value accessible

**Estimated total effort to MVP (v1.0)**: 6-9 months for a focused team of 2-4 engineers with Go/libp2p experience.

**Critical path**: Priorities 1-4 (Foundation → Identity → Networking → Content) must complete sequentially. Priorities 5-6 (Anonymous Layer → Shroud) can partially parallelize with Priority 9 (Pulse Map). Priorities 7-8 (Resonance → Mechanics) depend on anonymous layer completion.

**Key risks**:
1. **go-libp2p maintenance transition** — monitor community governance, avoid experimental features
2. **Shroud routing complexity** — can ship Hybrid mode first, add Guarded/Fortress incrementally
3. **Scope creep** — 11 claimed goals, 6+ mini-games; focus on end-to-end Surface Layer first

The project's ambition is well-matched to its architectural choices. The primary risk is scope: the full specification describes a complex system with many interacting components. A successful implementation path focuses on achieving end-to-end Surface Layer functionality first (v0.1-v0.4), then layering anonymous capabilities incrementally.
