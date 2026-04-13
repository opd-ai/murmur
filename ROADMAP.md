# Goal-Achievement Assessment

## Project Context

### What It Claims To Do

MURMUR claims to be a **decentralized, peer-to-peer social network with dual-layer identity** featuring:

1. **No servers, no algorithms, no permanent record** — fully P2P architecture
2. **Pulse Map visualization** — force-directed graph where users navigate a spatial social topology instead of scrolling feeds
3. **Privacy as structural guarantee** — Ed25519 cryptographic identity, no email/phone/third-party verification
4. **Ephemeral content** — Waves expire after configurable TTL (default 7 days, max 30)
5. **Anonymous Layer (Specters)** — pseudonymous identities routed through onion-style Shroud circuits, cryptographically unlinkable from main identity
6. **No engagement metrics** — no likes, follower counts, or algorithmic amplification
7. **Resonance reputation system** — local computation, unlockable mechanics at milestones (25/50/75/100/200)
8. **Anonymous mini-games** — Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Phantom Councils
9. **Three/four privacy modes** — Open, Hybrid, Guarded, Fortress (escalating anonymity guarantees)
10. **Onboarding flow** — six-phase guided introduction from first launch to active participation
11. **libp2p foundation** — GossipSub for message propagation, Kademlia DHT for peer discovery, NAT traversal via DCUtR

### Target Audience

- Users seeking privacy-first social networking without corporate intermediation
- People who want self-sovereign identity without email/phone requirements
- Those interested in anonymous social mechanics as a first-class feature
- Communities who value ephemeral, topology-based content propagation

### Architecture (Documented)

| Subsystem | Description | Status |
|-----------|-------------|--------|
| **Networking** | libp2p transport, GossipSub, Kademlia DHT, NAT traversal | ❌ Not implemented |
| **Identity** | Ed25519/Curve25519 keypairs, sigils, privacy modes | ❌ Not implemented |
| **Content** | Waves with PoW, TTL, threading, amplification | ❌ Not implemented |
| **Anonymous Layer** | Specters, Shroud onion routing, Resonance, mini-games | ❌ Not implemented |
| **Pulse Map** | Force-directed graph visualization | ❌ Not implemented |
| **Onboarding** | Six-phase guided introduction | ❌ Not implemented |

### Existing CI/Quality Gates

- **None** — no CI configuration, no Makefile, no test infrastructure

### Codebase Status

| Artifact | Present |
|----------|---------|
| Go source files | ❌ None |
| `go.mod` | ❌ None |
| Tests | ❌ None |
| Build scripts | ❌ None |
| Documentation | ✅ Comprehensive (~550KB across 18 markdown files) |

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
| Onboarding flow | ❌ Missing | No UI code | Five-phase sequence with animations |
| libp2p networking stack | ❌ Missing | No dependencies | GossipSub, DHT, NAT traversal setup |
| No engagement metrics | ✅ Achieved | By absence | Design principle maintained by not implementing metrics |

**Overall: 1/12 goals achieved (by design default)**

---

## Competitive Landscape Context

MURMUR positions itself against existing decentralized social platforms:

| Platform | Model | MURMUR's Differentiation |
|----------|-------|--------------------------|
| **Mastodon** | Federated servers (ActivityPub) | Fully P2P, no instance fragmentation |
| **Bluesky** | Centralized with data portability (AT Protocol) | True decentralization from day one |
| **Scuttlebutt** | P2P gossip | Visual spatial UI, anonymous layer, mini-games |
| **Matrix** | Federated chat/collab | Social-first design, not chat-focused |

MURMUR's distinguishing features (if implemented):
- **Shadow Gradient** — tiered anonymity as a design pillar, not an afterthought
- **Pulse Map** — spatial topology navigation replacing algorithmic feeds
- **Anonymous mini-games** — gamified anonymous social mechanics unique in the space

---

## Roadmap

### Priority 1: Establish Codebase Foundation

**Goal**: Transform documentation-only project into implementable Go codebase.

- [ ] Create `go.mod` with module path `github.com/opd-ai/murmur` and Go version 1.22+
- [ ] Add core dependencies:
  - `github.com/libp2p/go-libp2p` — P2P networking foundation
  - `github.com/libp2p/go-libp2p-pubsub` — GossipSub for Wave propagation
  - `github.com/libp2p/go-libp2p-kad-dht` — Kademlia peer discovery
  - `golang.org/x/crypto` — Ed25519/Curve25519, ChaCha20-Poly1305, Argon2id
- [ ] Create package structure per README (`src/networking/`, `src/identity/`, etc.)
- [ ] Add CI workflow (`.github/workflows/ci.yml`) with `go build`, `go test`, `go vet`
- [ ] **Validation**: `go build ./...` succeeds; CI passes on empty test scaffold

### Priority 2: Implement Identity System

**Goal**: Achieve claimed self-sovereign identity with Ed25519 keypairs.

- [ ] Implement `src/identity/keys/` — Ed25519 keypair generation via `crypto/ed25519`
- [ ] Implement secure keystore with Argon2id + ChaCha20-Poly1305 encryption
- [ ] Implement Peer ID derivation (SHA-256 truncated to 160 bits)
- [ ] Implement `src/identity/sigils/` — deterministic visual identity generation from public key hash
- [ ] Implement BIP-39 mnemonic backup encoding/decoding
- [ ] Implement key export/import for device migration
- [ ] Write unit tests for keypair generation, storage encryption, backup/restore
- [ ] **Validation**: `go test ./src/identity/...` passes; round-trip backup recovery works

### Priority 3: Build Networking Layer

**Goal**: Achieve peer-to-peer mesh with gossip propagation.

- [ ] Implement `src/networking/transport/` — libp2p host with Noise/TLS encryption
- [ ] Implement `src/networking/discovery/` — Kademlia DHT bootstrap and peer discovery
- [ ] Implement `src/networking/gossip/` — GossipSub subscription to `murmur/waves/v1`, `murmur/identity/v1`
- [ ] Implement `src/networking/mesh/` — connection manager with 6-12 peer target, tiered priority
- [ ] Implement `src/networking/relay/` — NAT traversal via DCUtR, relay fallback
- [ ] Implement peer scoring to penalize invalid/duplicate messages
- [ ] Write integration tests with multi-node simulation
- [ ] **Validation**: Two nodes on same LAN discover each other and exchange test messages

### Priority 4: Implement Wave Content System

**Goal**: Achieve ephemeral, signed content with PoW and TTL.

- [ ] Define Wave protobuf/JSON schema per spec (ID, author key, content, timestamp, parent, reference, TTL, nonce, signature)
- [ ] Implement `src/content/pow/` — SHA-256 PoW with configurable difficulty (2-5s target)
- [ ] Implement `src/content/waves/` — Wave creation, signing, validation
- [ ] Implement `src/content/propagation/` — GossipSub broadcast, hop tracking, deduplication
- [ ] Implement `src/content/storage/` — local cache with LRU eviction, hourly TTL sweep
- [ ] Implement `src/content/threads/` — reply chains, amplification references
- [ ] Write tests for PoW verification, signature validation, TTL expiration
- [ ] **Validation**: Wave published on node A arrives at node B within 5 seconds; expired Waves are garbage collected

### Priority 5: Implement Anonymous Layer Foundation

**Goal**: Achieve cryptographically unlinkable Specter identities.

- [ ] Implement `src/anonymous/specters/` — Curve25519 keypair generation (independent from main Ed25519)
- [ ] Implement procedural pseudonym generation (adjective + noun from 4096-word lists)
- [ ] Implement Specter sigil generation (distinct visual style from Surface sigils)
- [ ] Implement separate encrypted keystore partition for Specter keys
- [ ] Implement `src/identity/modes/` — Open/Hybrid/Fortress mode switching
- [ ] Implement mode-aware message routing (Surface vs Anonymous topics)
- [ ] Write tests for keypair independence verification, mode transition safety
- [ ] **Validation**: Specter keypair has no mathematical relationship to main keypair (verified by independent generation)

### Priority 6: Implement Shroud Onion Routing

**Goal**: Achieve three-hop onion circuits for anonymous traffic.

- [ ] Implement `src/anonymous/shroud/` — Shroud Node beacon announcements on `murmur/beacon/v1`
- [ ] Implement circuit construction — three-hop relay chain selection with diversity heuristics
- [ ] Implement onion encryption (triple-layer Curve25519 + ChaCha20)
- [ ] Implement circuit handshake with forward-secret session keys
- [ ] Implement 10-minute circuit rotation
- [ ] Implement traffic padding (1KB fixed packets, 1 packet/second dummy traffic)
- [ ] Write simulation tests for traffic analysis resistance
- [ ] **Validation**: Anonymous Wave from Guarded node cannot be traced to origin IP by passive observer in simulation

### Priority 7: Implement Resonance Reputation

**Goal**: Achieve local reputation computation with milestone unlocks.

- [ ] Implement `src/anonymous/resonance/` — local scoring algorithm per spec
- [ ] Implement four signal categories: publication consistency, mini-game quality, gift activity, community endorsement
- [ ] Implement time decay for inactive Specters
- [ ] Implement milestone unlocks at 25/50/75/100/200 Resonance
- [ ] Implement rank progression: Shade → Wraith → Shade-Wraith → Phantom → Council-Eligible
- [ ] Write tests for scoring accuracy, decay rates, unlock thresholds
- [ ] **Validation**: Consistent daily activity over 7 days reaches Resonance 25 unlock

### Priority 8: Implement Core Anonymous Mechanics

**Goal**: Achieve cross-layer social mechanics that drive Shadow Gradient pull.

- [ ] Implement `src/anonymous/mechanics/gifts.go` — Phantom Gifts (visual effects on Surface nodes)
- [ ] Implement `src/anonymous/mechanics/marks.go` — Specter Marks (persistent annotations with categories)
- [ ] Implement `src/anonymous/mechanics/territory.go` — Territory Drift (influence accumulation, Controller status)
- [ ] Implement basic Cipher Puzzle (Fragment Puzzle variant)
- [ ] Implement Masked Event hosting (temporary anonymous social space)
- [ ] Write tests for gift application/expiration, mark persistence/decay, territory influence computation
- [ ] **Validation**: Phantom Gift from Resonance 25+ Specter appears on recipient's node for 24 hours

### Priority 9: Build Pulse Map Visualization

**Goal**: Achieve spatial topology navigation as primary interface.

- [ ] Select rendering framework (likely Ebiten for Go, or WebGL via wasm)
- [ ] Implement `src/pulsemap/layout/` — force-directed graph engine (Barnes-Hut optimization for large graphs)
- [ ] Implement `src/pulsemap/rendering/` — node/edge drawing, connection animations, sigil rendering
- [ ] Implement `src/pulsemap/interaction/` — pan, zoom, node selection, navigation
- [ ] Implement Wave propagation ripple effects
- [ ] Implement Anonymous Layer overlay (ghostly nodes, faint edges, shimmer effects)
- [ ] Implement Phantom Gift/Mark visual effects on nodes
- [ ] Write visual regression tests
- [ ] **Validation**: 100-node graph renders at 60fps with smooth interaction

### Priority 10: Implement Onboarding Flow

**Goal**: Achieve guided introduction from first launch to active participation.

- [ ] Implement `src/onboarding/flow/` — phase controller for Welcome → Identity → Mode → Bootstrap → Exploration
- [ ] Implement Welcome screen with animated node visualization
- [ ] Implement Identity Creation with keypair generation ceremony, display name, backup prompt
- [ ] Implement Mode Selection with animated layer introduction, mode cards
- [ ] Implement Network Bootstrap with peer discovery progress, first connection celebration
- [ ] Implement Guided Exploration with contextual hints, first Wave prompt
- [ ] Implement "Skip for Now" with appropriate warnings and reminder system
- [ ] **Validation**: New user completes onboarding in <5 minutes with clear understanding of core concepts

### Priority 11: Advanced Anonymous Mechanics

**Goal**: Complete the mini-game ecosystem.

- [ ] Implement Specter Hunts (network-wide scavenger hunts with cryptographic fragments)
- [ ] Implement Oracle Pools (commit-reveal prediction markets on network metrics)
- [ ] Implement Sigil Forge (timed creative challenges with audience amplification judging)
- [ ] Implement Shadow Play (anonymous social deduction game)
- [ ] Implement Phantom Councils (5-13 member secret coordination bodies, threshold voting)
- [ ] Implement Abyssal Waves (Fortress-exclusive deep anonymity Waves)
- [ ] Write integration tests for game completion flows
- [ ] **Validation**: Full mini-game lifecycle from initiation to resolution works correctly

### Priority 12: Production Hardening

**Goal**: Achieve production-ready quality.

- [ ] Implement comprehensive error handling with user-friendly messages
- [ ] Implement graceful degradation (Shroud Network unavailable, low peer count)
- [ ] Add telemetry-free metrics (local-only) for debugging
- [ ] Write fuzzing tests for message parsing, cryptographic operations
- [ ] Conduct security audit of Shroud routing implementation
- [ ] Add rate limiting and backpressure for resource management
- [ ] Create deployment documentation (desktop, mobile, SBC)
- [ ] **Validation**: 72-hour network simulation with 1000 nodes shows no memory leaks, panics, or deadlocks

---

## Implementation Notes

### Technical Decisions Required

1. **UI Framework**: Go options include Ebiten (game engine), Gio (immediate mode GUI), or WebAssembly targeting browser. Mobile requires additional consideration (gomobile, or separate codebase).

2. **Sigil Generation**: Spec mentions "procedural generation from hash bytes" — recommend implementing as parametric SVG or shader-based rendering for cross-platform consistency.

3. **PoW Tuning**: 2-5 second target on "modern consumer CPU" needs benchmarking across devices. Consider difficulty adjustment based on device capability detection.

4. **Shroud Node Incentives**: Spec relies on volunteer operation. May need to revisit if insufficient relay capacity emerges.

### Risk Areas

| Risk | Mitigation |
|------|------------|
| No implementation exists | Start with minimal viable networking + identity to prove feasibility |
| Shroud routing complexity | Can ship Hybrid mode first, add Guarded/Fortress later |
| Pulse Map performance | Barnes-Hut algorithm, level-of-detail rendering for large graphs |
| Mobile background operation | Platform-specific persistent service implementation required |
| Bootstrap node centralization | Document community-run bootstrap infrastructure strategy |

### Success Criteria Per Phase

| Phase | Milestone | Definition of Done |
|-------|-----------|-------------------|
| Foundation | v0.1 | Two nodes exchange messages over libp2p |
| Identity | v0.2 | Users can create, backup, and restore identity |
| Content | v0.3 | Waves propagate with PoW validation and TTL expiration |
| Anonymous | v0.4 | Specters can publish unlinkable anonymous Waves |
| Visualization | v0.5 | Pulse Map renders topology with basic interaction |
| MVP | v1.0 | Full onboarding → Surface + Anonymous participation works end-to-end |

---

## Conclusion

MURMUR currently exists as a **comprehensive design document** (~550KB of detailed specification) with **zero implementation**. The README explicitly acknowledges this: "Pre-implementation. The design document is complete. Everything else is ahead."

The design is ambitious and well-specified, with clear architectural decisions documented for:
- Cryptographic primitives (Ed25519, Curve25519, ChaCha20-Poly1305, Argon2id)
- Network protocols (libp2p, GossipSub, Kademlia DHT)
- Privacy architecture (Shroud onion routing, traffic padding)
- Social mechanics (Resonance, mini-games, cross-layer interactions)

The gap between vision and reality is **complete** — no code exists. The roadmap above prioritizes implementation in order of:
1. **Foundational capability** (identity, networking, content)
2. **Differentiating features** (anonymous layer, Shroud routing)
3. **User experience** (Pulse Map, onboarding, mini-games)

Estimated effort to MVP (v1.0): **12-18 months** for a small team (2-4 engineers), assuming Go/libp2p expertise and no mobile-first requirement initially.
