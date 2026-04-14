# MURMUR — Implementation Roadmap

> Last updated: 2026-04-13
>
> This document tracks every feature, mechanic, and integration described in MURMUR's
> specification documents against the current codebase. Items are organized by milestone
> (v0.1 → v1.0) and subsystem. Each checkbox represents a discrete deliverable.
> Checked items (- [x]) are implemented; unchecked items (- [ ]) remain.

---

## Table of Contents

1. [Milestone v0.1 — Core Foundation](#milestone-v01--core-foundation)
2. [Milestone v0.2 — Network & Messaging](#milestone-v02--network--messaging)
3. [Milestone v0.3 — Content System](#milestone-v03--content-system)
4. [Milestone v0.4 — Identity & Privacy](#milestone-v04--identity--privacy)
5. [Milestone v0.5 — Anonymous Layer Core](#milestone-v05--anonymous-layer-core)
6. [Milestone v0.6 — Resonance & Reputation](#milestone-v06--resonance--reputation)
7. [Milestone v0.7 — Mini-Games & Mechanics](#milestone-v07--mini-games--mechanics)
8. [Milestone v0.8 — Pulse Map & Visualization](#milestone-v08--pulse-map--visualization)
9. [Milestone v0.9 — Onboarding & Growth](#milestone-v09--onboarding--growth)
10. [Milestone v1.0 — Production Readiness](#milestone-v10--production-readiness)

---

## Milestone v0.1 — Core Foundation

### Project Scaffold & Build System

- [x] `go.mod` with all required dependencies (libp2p, Ebitengine, Bbolt, protobuf, BLAKE3, x/crypto)
- [x] `cmd/murmur/main.go` entry point with `ebiten.RunGame()` wiring
- [x] `pkg/app/app.go` top-level App struct with lifecycle management (Run, Shutdown, WaitReady)
- [x] `pkg/config/` configuration loading with defaults (DataDir, ListenAddrs, BootstrapPeers)
- [x] `pkg/errors/` custom error types with structured error handling
- [x] `scripts/build-mobile.sh` Gomobile cross-compilation for Android APK and iOS xcframework
- [x] `Makefile` or `mage` build harness for `go build`, `go test`, `gofumpt`, `go vet`, `protoc` generation
- [x] CI pipeline (GitHub Actions) for lint, build, test on linux/amd64, darwin/amd64, windows/amd64
- [x] `go:embed` asset bundling for wordlists, shaders, default config, onboarding assets
- [x] Single static binary builds per target platform (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)

### Protocol Buffers

- [x] `proto/wave.proto` — Wave, Reply, Amplification, MurmurEnvelope, WaveType enum (8 types)
- [x] `proto/identity.proto` — IdentityDeclaration, ConnectionAnnouncement, PrivacyMode enum
- [x] `proto/resonance.proto` — ResonanceScore, ResonanceMilestone enum, InteractionType enum, ZKResonanceClaim
- [x] `proto/gossip.proto` — GossipMessage, TopicSubscription, PeerScore, DeduplicationEntry
- [x] `proto/shroud.proto` — RelayAdvertisement, CircuitConstruction, ShroudCell
- [x] Generated `.pb.go` files checked into repository
- [x] Protobuf validation helpers (envelope signature verification, timestamp range checks)
- [x] Default message factories for each envelope type

### Storage Layer (`pkg/store/`)

- [x] Bbolt database open/close with directory setup
- [x] Bucket initialization (identity, peers, waves, threads, shroud, resonance, config)
- [x] Raw CRUD: Put(bucket, key, value), Get(bucket, key), Delete(bucket, key)
- [x] Batch transaction support
- [x] Typed accessor methods per bucket (e.g., `GetWave(id) (*pb.Wave, error)`)
- [x] Protobuf marshal/unmarshal helpers integrated into store
- [x] Prefix scan and iteration for range queries
- [x] Bucket-level statistics (key count, size)
- [x] Schema migration system for version upgrades
- [x] LRU eviction policy for space-bounded storage (per TECHNICAL_IMPLEMENTATION.md)

---

## Milestone v0.2 — Network & Messaging

### Transport (`pkg/networking/transport/`)

- [x] libp2p host construction with Ed25519 peer identity
- [x] Noise XX transport security protocol
- [x] TCP transport with yamux stream multiplexing
- [x] QUIC transport (UDP, TLS 1.3, native multiplexing)
- [x] WebSocket transport for browser clients (per NETWORK_ARCHITECTURE.md)
- [x] WebRTC transport for browser-to-browser direct connections (ICE/STUN/TURN)
- [x] Transport fallback chain: QUIC → TCP → WebSocket
- [x] Connection limit enforcement (max 200 simultaneous peers per NETWORK_ARCHITECTURE.md)
- [x] Four-tier connection priority system (Social, Mesh, DHT, Opportunistic)

### Peer Discovery (`pkg/networking/discovery/`)

- [x] Kademlia DHT bootstrap peer connection
- [x] DHT peer lookup and address advertisement
- [x] mDNS local network peer discovery
- [x] Peer Exchange (PEX) protocol — `/murmur/peer-exchange/1` stream handler
- [x] PEX 5-minute periodic peer list sharing (per NETWORK_ARCHITECTURE.md)
- [x] Bootstrap node list (8–12 hardcoded entry points)
- [x] Peer routing table persistence across restarts

### NAT Traversal (`pkg/networking/relay/`)

- [x] Circuit relay protocol handler
- [x] Stream forwarding between relayed peers
- [x] AutoNAT service for NAT status detection
- [x] DCUtR (Direct Connection Upgrade through Relay) hole punching
- [x] Relay node capacity limits (max 128 concurrent, 128 KB/s per connection)
- [x] TURN server fallback for WebRTC ICE

### GossipSub (`pkg/networking/gossip/`)

- [x] GossipSub v1.1 initialization with peer scoring
- [x] Topic subscriptions: `/murmur/waves/1`, `/murmur/identity/1`, `/murmur/shroud/1`, `/murmur/pulse/1`
- [x] Peer scoring parameters (IP colocation penalty, invalid message penalty)
- [x] Flood publish enabled
- [x] **Message handler for `/murmur/waves/1`** — receive, validate, store, relay Waves
- [x] **Message handler for `/murmur/identity/1`** — receive identity declarations and connections
- [x] **Message handler for `/murmur/shroud/1`** — receive Shroud relay advertisements
- [x] **Message handler for `/murmur/pulse/1`** — receive and process heartbeat pings
- [x] MurmurEnvelope validation pipeline (version, signature, timestamp ±300s, PoW, dedup)
- [x] Message deduplication via BLAKE3 message_id (Bloom filter, 30-day window)
- [ ] Peer scoring integration with message validation (reward valid, penalize invalid)
- [ ] Additional topic: `/murmur/anonymous/waves/1.0` — Specter/Masked Waves
- [ ] Additional topic: `/murmur/anonymous/mechanics/1.0` — Gifts, Marks, mini-game events, Councils
- [ ] Additional topic: `/murmur/anonymous/beacons/1.0` — Beacon Waves (elevated PoW)
- [ ] Per-event ephemeral topics: `/murmur/event/[event_id]/1.0`
- [ ] Per-council encrypted topics: `/murmur/council/[council_id]/1.0`

### Mesh Management (`pkg/networking/mesh/`)

- [x] Connection notifees (connect/disconnect callbacks)
- [x] Heartbeat monitoring (30-second interval)
- [x] Missed heartbeat tracking (3-miss threshold)
- [x] Peer priority tiers (Identity > Gossip > Random)
- [x] Reconnection with exponential backoff
- [ ] Target mesh degree 6 (bounds 4–12) enforcement
- [ ] Dynamic connection pruning of low-score peers
- [ ] Multi-region diversity constraints for eclipse resistance
- [ ] Churn handling: mesh repair, DHT refresh on disconnect
- [ ] Network partition detection and graceful degradation
- [ ] Healing protocol on reconnection after partition

### Data Synchronization

- [ ] Wave sync protocol — `/murmur/wave-sync/1` stream handler (request-response for fetching Waves by hash)
- [ ] Sync limits: 1000 messages per request, 5 concurrent sync sessions, 100 msg/sec rate limit
- [ ] Selective sync by topic and by publisher
- [ ] Missed-message catch-up on reconnection

### Event Bus (`pkg/app/`)

- [ ] **Central event bus goroutine** with typed channel fan-out (per TECHNICAL_IMPLEMENTATION.md §8)
- [ ] Event type definitions (NetworkEvent, WaveEvent, IdentityEvent, TimerEvent, UserActionEvent)
- [ ] Subsystem subscriber registration at startup
- [ ] Fan-out to all subscribers per event type
- [ ] Backpressure handling for slow subscribers

---

## Milestone v0.3 — Content System

### Wave Creation (`pkg/content/waves/`)

- [x] Wave struct with all 8 type constants (Surface 0x01 through Beacon 0x08)
- [x] Wave.Create() with content validation (≤2048 bytes UTF-8)
- [x] BLAKE3-based Wave ID computation
- [x] Ed25519 signing via Signer interface
- [x] Proof of Work integration (pow.Compute)
- [x] TTL constraints (default 7 days, max 30 days)
- [x] Abyssal Wave creation with one-time Ed25519 keypair and nonce-derived key
- [ ] **Veiled Wave encryption** — Cross-layer Wave with Specter authorship and symmetric key wrapping
- [ ] **Sigil Wave payload structure** — Embedded random Specter sigil within Surface Wave
- [ ] **Beacon Wave construction** — System-generated high-visibility broadcast with 24-bit PoW
- [ ] **Masked Wave ephemeral handling** — 7-day TTL, single-use event keypair, auto-destruct
- [ ] Parent chain validation for Reply Waves (recursive thread integrity check)
- [ ] Wave reference parsing — inline `wave://[id]` and mention `@[hash]` links
- [ ] Amplification creation with optional PoW-free signature and hop count reset

### Proof of Work (`pkg/content/pow/`)

- [x] SHA-256 based PoW with configurable difficulty
- [x] Leading zero bit verification
- [x] Nonce iteration up to MaxNonce
- [ ] Default difficulty 20 leading zero bits (standard) per TECHNICAL_IMPLEMENTATION.md
- [ ] Elevated difficulty 24 bits for Beacon Waves
- [ ] Reduced difficulty 16 bits for standard Waves per WAVES.md
- [ ] Dynamic difficulty adjustment (local per-node configuration)
- [ ] PoW verification before signature check (malleability resistance per SECURITY_PRIVACY.md)
- [ ] Target computation time: 2–5 seconds at difficulty 20

### Wave Propagation (`pkg/content/propagation/`)

- [x] Wave TTL decay tracking
- [x] Delivery limit enforcement
- [ ] Gossip relay via GossipSub publish (actual network send)
- [ ] Hop count tracking and enforcement (max 20 hops, discard beyond)
- [ ] Deduplication by Wave ID before relay
- [ ] Bridge injection for cross-layer propagation (Hybrid nodes relay Veiled Waves)
- [ ] Propagation latency target <500ms across 3 hops

### Threading (`pkg/content/threads/`)

- [x] Reply chain indexing
- [x] Thread reconstruction from parent hashes
- [x] Reply depth tracking
- [ ] Conversation tree building (full recursive thread assembly)
- [ ] Thread root lookup for deeply nested replies
- [ ] Reply notification events to event bus

### Content Storage (`pkg/content/storage/`)

- [x] Wave persistence (create, read)
- [x] Reply storage
- [x] Amplification tracking
- [ ] TTL enforcement with automatic expiration (30-day content window)
- [ ] Hourly garbage collection sweep (<100ms target per TECHNICAL_IMPLEMENTATION.md)
- [ ] LRU eviction when storage exceeds 50 MiB budget
- [ ] Bbolt persistence (currently in-memory only for some stores)

### Content Interaction

- [ ] Muting by author public key (local filtering)
- [ ] Muting by keyword with wildcard pattern matching
- [ ] Resonance-based content filtering (minimum score threshold)

---

## Milestone v0.4 — Identity & Privacy

### Key Management (`pkg/identity/keys/`)

- [x] Ed25519 keypair generation for Surface Layer identity
- [x] Curve25519 keypair generation for Anonymous Layer (Specter)
- [x] Ed25519 signing and verification
- [x] Argon2id passphrase-based key derivation (time=3, memory=64 MiB, threads=4, output=32 bytes)
- [ ] Encrypted keystore (Argon2id + XChaCha20-Poly1305) for at-rest key protection
- [ ] Key backup to encrypted file
- [ ] BIP-39 mnemonic recovery phrase generation and restoration
- [ ] Key export/import for cross-device identity migration
- [ ] Key material zeroing before backing arrays become GC-eligible (per SECURITY_PRIVACY.md)
- [ ] Keypair independence enforcement — Surface and Specter share no derivation path
- [ ] Fortress-mode dedicated transport Ed25519 keypair (separate from Specter key)

### Sigil Generation (`pkg/identity/sigils/`)

- [x] Deterministic 64×64 PNG generation from public key hash
- [x] Geometric pattern rendering with color, shapes, symmetry
- [ ] Specter sigil with cool-tone palette (200–280° hue range per DESIGN_DOCUMENT.md)
- [ ] Masked event sigil generation from single-use key hash
- [ ] Sigil rendering as Ebitengine image for Pulse Map overlay

### Identity Declarations (`pkg/identity/declarations/`)

- [x] Declaration struct with public key, display name, sigil parameters
- [x] Signed metadata for identity announcements
- [ ] Connection Declaration — bilateral signed relationship announcement
- [ ] Connection Revocation — cancellation message
- [ ] Specter Declaration — pseudonym and sigil registration on Anonymous Layer
- [ ] Profile Update — display name change with new declaration
- [ ] Identity publication via GossipSub `/murmur/identity/1` topic
- [ ] PoW requirement for identity creation (anti-spam)

### Privacy Modes (`pkg/identity/modes/`)

- [x] Privacy mode enum (Open, Hybrid, Guarded, Fortress)
- [x] Mode descriptions and properties
- [ ] **Mode transition state machine** — Open ↔ Hybrid ↔ Guarded ↔ Fortress with rules
- [ ] Specter preservation on upgrade (Open → Hybrid)
- [ ] Specter destruction on downgrade (Hybrid → Open)
- [ ] Traffic padding activation for Guarded/Fortress (constant-rate dummy packets, 2/sec)
- [ ] Network separation enforcement — distinct gossip topics per layer
- [ ] Behavioral separation guidance — activity pattern differentiation

### Proximity Ignition

- [ ] QR code generation with public key, IP/port, one-time token
- [ ] NFC tap exchange (shorter data payload)
- [ ] mDNS auto-detection for local network peers
- [ ] Mutual confirmation protocol (both devices verify)
- [ ] Resonance bonus for Ignition (first 10 = 3 Resonance each)
- [ ] ZK Claim support for Ignition count ("Completed >N Ignitions")

---

## Milestone v0.5 — Anonymous Layer Core

### Specter Identity (`pkg/anonymous/specters/`)

- [x] Curve25519 keypair generation for Specter
- [x] Two-word pseudonym generation (adjective + noun from curated wordlist)
- [x] Specter sigil generation (cool-tone geometric pattern)
- [ ] Specter creation without network announcement (per SHADOW_GRADIENT.md)
- [ ] Specter rotation — destroy and create new identity (irreversible)
- [ ] Specter destruction on mode downgrade
- [ ] Specter Connection — Anonymous Layer bilateral relationship
- [ ] Specter visual properties — translucency, particle emissions, cool tones on Pulse Map

### Shroud Network (`pkg/anonymous/shroud/`)

- [x] Three-hop circuit construction with Curve25519 key exchange
- [x] XChaCha20-Poly1305 multi-layer onion encryption
- [x] Traffic padding to fixed 1024-byte packets
- [x] Relay registry with bandwidth advertising (RelayInfo)
- [x] Relay selection excluding initiator's direct mesh neighbors (hop diversity)
- [ ] **Circuit rotation timer** — 10-minute rotation cycle with dual active circuits (primary + backup)
- [ ] **Circuit close/teardown mechanism** — clean circuit destruction
- [ ] **Shroud relay discovery** via Beacon Waves on Anonymous Layer (not manual AddRelay)
- [ ] **Error recovery for relay failure** — failover to backup circuit, circuit rebuild
- [ ] Nonce sequencing for replay protection (proper ordering per circuit)
- [ ] Mix network properties: random delay (exponential distribution, mean 200ms)
- [ ] Cover traffic: constant-rate dummy packets (2 per second) on active circuits
- [ ] Shroud Node operation for Fortress-mode users (serve as relay)
- [ ] Shroud Node capacity metrics advertisement
- [ ] End-to-end message delivery through Shroud circuits (actual network send/receive)

### Whisper Chains

- [ ] Anonymous multi-hop message relay between Specters
- [ ] End-to-end encryption via Curve25519 DH + HKDF-SHA-256 key derivation
- [ ] XChaCha20-Poly1305 message encryption
- [ ] Message routing through Shroud circuits
- [ ] Delivery confirmation without sender reveal
- [ ] Rate limiting to prevent abuse

### Cross-Layer Interactions

- [ ] Visual overlay blending — Surface (visible) + Anonymous (ghostly) on Pulse Map
- [ ] Bridge routing by Hybrid nodes — relay between Surface and Anonymous gossip topics
- [ ] Wave bridging — Veiled Waves propagated on both layers by bridge nodes
- [ ] Sigil Waves signaling Specter presence on Surface Layer

---

## Milestone v0.6 — Resonance & Reputation

### Surface Resonance Computation (`pkg/anonymous/resonance/`)

- [x] ResonanceScore struct with signal tracking
- [x] Signal weighting configuration (publications, puzzles, games, gifts, endorsements)
- [x] Milestone lookup — RankFromScore with 6 thresholds
- [x] Cache invalidation on updates
- [ ] **Full Surface Resonance formula** with all 8 input signals:
  - [ ] Connection Count
  - [ ] Connection Diversity (ratio of unique clusters)
  - [ ] Wave Output (publications in 30-day window)
  - [ ] Amplification Received
  - [ ] Amplification Given
  - [ ] Bridge Activity (cross-layer relay count for Hybrid nodes)
  - [ ] Account Age
  - [ ] Uptime (fraction of time online)
- [ ] Surface milestones with visual effects:
  - [ ] Ember (10) — warm glow effect
  - [ ] Spark (25) — pulsing ring animation
  - [ ] Flame (50) — particle trail effect
  - [ ] Blaze (100) — custom color palette
  - [ ] Inferno (200) — animated aura
  - [ ] Corona (500) — multi-layered corona effect
- [ ] Temporal decay over 30-day window for most signals
- [ ] Connection Age bonus (longevity reward)

### Specter Resonance Computation

- [x] AddPublication, AddGameResult, AddGiftGiven/Received methods
- [x] Endorsement tracking with high-tier weighting
- [x] Decay calculation framework
- [ ] **Full Specter Resonance formula** with all 15+ input signals:
  - [ ] Specter Connection Count & Diversity
  - [ ] Specter Wave Output
  - [ ] Anonymous Amplification Received/Given
  - [ ] Phantom Gift Volume
  - [ ] Masked Event Participation
  - [ ] Mini-Game Activity
  - [ ] Territory Influence
  - [ ] Cartographer Score
  - [ ] Whisper Chain Contributions
  - [ ] ZK Claim Count
  - [ ] Shroud Node Operation credit
  - [ ] Council Membership
  - [ ] Specter Age & Uptime
- [ ] Specter milestones with visual effects:
  - [ ] Whisper (10) — ghostly trail
  - [ ] Shade (25) — Phantom Gift access
  - [ ] Wraith (50) — expanded gifts, Cipher Puzzles, Sigil Forge
  - [ ] Shade-Wraith (75) — Specter Hunts
  - [ ] Phantom (100) — Masked Events, Oracle Pools, Specter Marks
  - [ ] Revenant (200) — Shadow Play, Phantom Council eligibility
  - [ ] Abyss (500) — Kage shader effect (Fortress only)
- [ ] **Decay actually applied** — periodic background computation (every 60s)
- [ ] Resonance integration with mini-game result callbacks (auto-update scores)

### Echo Index & Echo Shadow

- [x] Echo Index architecture (cluster insularity metrics)
- [ ] Echo Index computation — intra-cluster amplification ratio
- [ ] Echo Shadow — Anonymous Layer equivalent of Echo Index
- [ ] Visual color-coding on Pulse Map (healthy diversity vs echo chamber indicators)

### Zero-Knowledge Proofs

- [ ] **Pedersen commitment generation** for Resonance score hiding (using `go-ristretto`)
- [ ] **Bulletproofs range proof generation** — prove Resonance within threshold without revealing exact value
- [ ] ZK Claim types:
  - [ ] Specter Resonance range ("Resonance > 100")
  - [ ] Specter Age range ("active > 90 days")
  - [ ] Ignition count range ("met > 10 peers in person")
  - [ ] Masked Event participation ("participated > 5 events")
- [ ] Non-interactive proof object (~672 bytes for 64-bit range)
- [ ] Claim verification by any node (~10ms per claim)
- [ ] ZK claims used for Council admission and mini-game thresholds

---

## Milestone v0.7 — Mini-Games & Mechanics

### Cipher Puzzles (`pkg/anonymous/mechanics/`)

- [x] Three puzzle types: Fragment, Mosaic, Cascade
- [x] Full state machine with creation, solving, expiration
- [x] Solution submission with SHA-256 cryptographic verification
- [x] PuzzleStore with active/history tracking and garbage collection
- [x] Resonance bonus calculation: `4 * ln(1 + difficulty_factor * participation_count)`
- [x] TTL and expiration with state transitions
- [ ] **Resonance gating enforcement** — only Resonance ≥50 Specters can create puzzles
- [ ] **Network propagation** — publish puzzle events to `/murmur/anonymous/mechanics/1.0`
- [ ] **Bbolt persistence** — PuzzleStore backed by `pkg/store` instead of in-memory maps
- [ ] **Pulse Map visualization** — rotating cryptographic symbol at puzzle location
- [ ] **UI: puzzle composition panel** — create puzzle with difficulty and content inputs
- [ ] **UI: puzzle solving interface** — submit solution with feedback

### Specter Hunts

- [x] Fragment generation with deterministic SHA-256 location hashing
- [x] Progressive clue revealing system (timed intervals)
- [x] Proximity proofs for fragment claiming
- [x] Leaderboard calculation
- [x] HuntStore with state management
- [x] Resonance bonus: `5.0 * ratio * fragmentsClaimed`
- [ ] **Resonance gating** — only Resonance ≥75 Specters can initiate Hunts
- [ ] **Actual proximity proof via DHT routing** — replace simplified logic with real topological proof
- [ ] **Network propagation** — broadcast Hunt events, fragment claims, clue reveals
- [ ] **Bbolt persistence** — HuntStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — scattered glowing fragments across map topology
- [ ] **UI: Hunt tracker overlay** — fragment locations, clue display, leaderboard

### Territory Drift

- [x] Influence calculation: `8 * ln(1 + activities)`
- [x] Territory state machine (Neutral, Controlled, Contested)
- [x] Weekly reset cycle with 30-day activity window
- [x] TerritoryManager with influence computation
- [x] Resonance score: `3 * ln(1 + controlled + 0.5 * contested)`
- [ ] **Louvain clustering algorithm** for territory partitioning (per ANONYMOUS_GAME_MECHANICS.md)
- [ ] **Network propagation** — broadcast influence claims and territory state changes
- [ ] **Bbolt persistence** — territory state backed by `pkg/store`
- [ ] **Pulse Map visualization** — translucent watermarks with territory boundaries
- [ ] **UI: Territory overview panel** — controlled regions, influence scores, weekly cycle status
- [ ] **Cartographer's Trail** — territory exploration tracking with badges

### Oracle Pools

- [x] Commitment-reveal voting scheme for predictions
- [x] Boolean and numeric prediction types
- [x] Accuracy scoring: `1 / (1 + |prediction - outcome|)`
- [x] Top 25% reward distribution
- [x] Resonance bonus: `3 * ln(1 + participant_count / rank)`
- [x] OraclePoolStore with state transitions
- [ ] **Resonance gating** — only Resonance ≥100 Specters can create Oracle Pools
- [ ] **Network propagation** — broadcast pool creation, commitments, reveals, outcomes
- [ ] **Bbolt persistence** — OraclePoolStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — swirling vortex icon at pool location
- [ ] **UI: Oracle Pool panel** — create pool, submit prediction, view outcomes
- [ ] **Outcome verification** — network-observable event confirmation protocol

### Sigil Forge

- [x] Three forge types: SigilArt, MicroFiction, RemixChain
- [x] Entry submission with deduplication and size limits
- [x] Amplification tracking with weighted Resonance scaling
- [x] Remix chain score distribution (equal sharing)
- [x] Evaluation with ranking
- [x] Resonance bonuses: winner `4 * ln(1 + amplifications)`, participants `2 * ln(1 + own_amplifications)`
- [ ] **Resonance gating** — only Resonance ≥50 Specters can participate
- [ ] **Network propagation** — broadcast forge events, entries, votes
- [ ] **Bbolt persistence** — ForgeStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — anvil-and-flame icon with orbiting entries
- [ ] **UI: Forge submission panel** — create/submit entries, view competitors
- [ ] **Timed creative challenge mechanics** — countdown timer, submission window

### Shadow Play

- [x] Role assignment (Echo/Shade) deterministic from seed
- [x] Voting round mechanics with vote tallying
- [x] Win conditions (Echoes eliminate all Shades, Shades ≥ Echoes)
- [x] Vote elimination with MissedHeartbeat tracking
- [x] Resonance bonuses: winners `5 * ln(1 + participants)`, losers `2 * ln(1 + participants)`
- [x] ShadowPlayStore with state management
- [ ] **Resonance gating** — only Resonance ≥200 Specters can initiate Shadow Play
- [ ] **Network propagation** — broadcast game state, votes, eliminations, outcomes
- [ ] **Bbolt persistence** — ShadowPlayStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — dark shimmering dome with lightning effects
- [ ] **UI: Shadow Play game interface** — role reveal, vote casting, round status, results
- [ ] **Communication phase** — in-game discussion between rounds via encrypted group channel

### Masked Events

- [ ] **Masked Event hosting** — create temporary anonymous gathering (Resonance ≥100)
- [ ] Single-use Ed25519 keypair generation per event
- [ ] Masked Pseudonym — event-themed two-word identifier
- [ ] Masked Sigil generation from Masked key hash
- [ ] Ephemeral per-event GossipSub topic: `/murmur/event/[event_id]/1.0`
- [ ] Masked Wave (0x07) — 7-day TTL ephemeral Wave within event
- [ ] Post-event keypair destruction and unlinkability guarantee
- [ ] **Network propagation** — event announcements, join/leave, Masked Waves
- [ ] **Bbolt persistence** — event metadata and lifecycle
- [ ] **Pulse Map visualization** — translucent dome with identical featureless dots inside
- [ ] **UI: Event lobby** — create event, join event, compose Masked Waves

### Phantom Gifts (`pkg/anonymous/mechanics/`)

- [x] Three-tier system (Basic@25, Expanded@50, Premium@100 Resonance)
- [x] 25 effect types with tiered unlock
- [x] Daily rate limiting (3 gifts/24h)
- [x] 7-day expiration with automatic garbage collection
- [x] GiftStore with recipient/sender indexes
- [x] Ed25519 signature verification
- [ ] **Network propagation** — broadcast gifts via `/murmur/anonymous/mechanics/1.0`
- [ ] **Bbolt persistence** — GiftStore backed by `pkg/store`
- [ ] **Pulse Map integration** — animated cosmetic effects on recipient nodes (3 tiers)
- [ ] **UI: Gift sending panel** — select gift type, choose recipient, confirm send
- [ ] **Cross-layer visibility** — Surface nodes see gift effects from Anonymous Layer

### Specter Marks

- [x] Three mark categories (Watcher, Ally, Rival)
- [x] 30-day linear visibility decay
- [x] Deduplication (1 mark per Specter per target)
- [x] MarkStore with target/marker reverse indexes
- [x] Dominant mark determination algorithm
- [ ] **Resonance gating** — only Resonance ≥100 Specters can place Marks
- [ ] **Network propagation** — broadcast marks via `/murmur/anonymous/mechanics/1.0`
- [ ] **Bbolt persistence** — MarkStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — orbiting sigil icons on marked Surface nodes
- [ ] **UI: Mark placement panel** — choose mark type, select target node
- [ ] **Voting mechanics** — community mark endorsement/challenge

### Phantom Councils

- [x] Member admission voting (unanimous threshold)
- [x] Expulsion voting (2/3 majority)
- [x] Proposal voting (simple majority)
- [x] Member status tracking (Pending, Active, Expelled, Departed)
- [x] CouncilStore with state management
- [ ] **Resonance gating** — Fortress mode + Resonance ≥200 for creation
- [ ] **ZK Claim verification** — verify Resonance threshold via Bulletproofs before admission
- [ ] **Encrypted GossipSub topic** — `/murmur/council/[council_id]/1.0` with group key
- [ ] **Network propagation** — council creation, admission, proposals, votes
- [ ] **Bbolt persistence** — CouncilStore backed by `pkg/store`
- [ ] **Pulse Map visualization** — constellation pattern connecting member nodes, unique color threads
- [ ] **UI: Council management panel** — create council, invite members, propose, vote
- [ ] **Council size constraints** — 3–13 members per spec

### Surface Sparks

- [ ] Wave Relay challenge — fastest relay earns bonus
- [ ] Echo Races — competitive amplification chain building
- [ ] Surface Spark event creation and lifecycle
- [ ] Network propagation via GossipSub
- [ ] Pulse Map visualization for active Sparks

### Echo Chains

- [ ] Visible re-broadcast chain tracking
- [ ] Chain length bonuses for participants
- [ ] Pulse Map visualization — animated amplification trail between nodes

### Pulse Beats

- [ ] Gamified notification events with visual indicators
- [ ] Edge-of-viewport notification rendering
- [ ] Event aggregation and priority ordering

### Specter Trophies

- [ ] Achievement milestone tracking per Specter
- [ ] Visual glyph unlocks at achievement thresholds
- [ ] Trophy display on Specter node detail panel

---

## Milestone v0.8 — Pulse Map & Visualization

### Force-Directed Layout (`pkg/pulsemap/layout/`)

- [x] Fruchterman-Reingold force-directed algorithm
- [x] Barnes-Hut approximation for large graphs (>500 nodes)
- [x] Coulomb repulsion + spring attraction forces
- [x] Temperature-based convergence and damping
- [x] Camera system with pan/zoom
- [ ] **Double-buffered position swap** — `atomic.Pointer` for lock-free layout → render handoff
- [ ] **Hierarchical aggregation** — cluster representatives for >500 visible nodes
- [ ] **Incremental layout** — 30 ticks/second background goroutine
- [ ] **Performance**: 60fps with 500 nodes and 2000 edges (target per TECHNICAL_IMPLEMENTATION.md)
- [ ] **Viewport culling** — only compute forces for visible nodes
- [ ] **Data update throttling** — 30Hz nodes, 10Hz state, 5Hz connections, 2Hz content

### Node Rendering (`pkg/pulsemap/rendering/`)

- [x] Core circle with logarithmic radius scaling
- [x] Halo glow for activity recency
- [x] Ring border for mode indication
- [x] Selection highlight rendering
- [x] Edge drawing with age-based opacity and thickness
- [x] NodeStyle with colors, rings, halos, activity
- [x] ZoomLevel support (Macro, Meso, Micro)
- [ ] **Sigil overlay** — render deterministic geometric pattern on node
- [ ] **Specter node appearance** — translucency, particle emissions, cool-tone coloring
- [ ] **Specter Mark sigils** — orbiting small icons around marked nodes
- [ ] **Phantom Gift overlays** — animated cosmetic effects on gifted nodes
- [ ] **Connection age visual encoding** — bright new, faded old transitions
- [ ] **Interaction frequency thickness** — edge width proportional to message exchange rate
- [ ] **Pulse animation** — Waves traveling along edges as light pulses
- [ ] **Text labels** — display name/pseudonym at Micro zoom level

### Visual Effects (`pkg/pulsemap/rendering/effects/`)

- [x] Kage shader system (Glow, Ripple, Spectra)
- [x] GiftRenderer with 25+ effect configurations
- [x] Resonance-tiered effects (25, 50, 100 thresholds)
- [ ] **Ripple propagation animation** — Wave publication visualization radiating outward
- [ ] **Amplification trail** — visual connection between amplifier and original author
- [ ] **Activity halo decay** — 60-minute intensity decay curve
- [ ] **GPU particle system** — efficient ambient + mechanic-specific particle rendering
- [ ] **Milestone visual effects** — Ember glow, Spark pulse, Flame trail, Blaze palette, Inferno aura, Corona layers
- [ ] **Specter milestone effects** — Whisper trail, Shade glow, Wraith particles, through Abyss shader
- [ ] **Blur effects** — background animation for depth
- [ ] **Translucency compositing** — layer separation blending

### Overlays (`pkg/pulsemap/overlays/`)

- [x] Layer Blend Slider (Surface/Anonymous opacity control)
- [x] Fortress mode toggle
- [x] Particle emitter for Specter effects
- [x] Mini-game visualization tracking
- [ ] **Minimap** — full network overview in corner with viewport indicator
- [ ] **Activity Heat Map overlay** — blue-to-red gradient, 60-minute trailing window, blurred background layer
- [ ] **Mini-game visualization icons**:
  - [ ] Cipher Puzzles — rotating cryptographic symbol
  - [ ] Specter Hunts — scattered glowing fragments
  - [ ] Territory Drift — translucent watermarks with boundaries
  - [ ] Oracle Pools — swirling vortex icon
  - [ ] Sigil Forge — anvil-and-flame with orbiting entries
  - [ ] Shadow Play — dark shimmering dome with lightning
  - [ ] Masked Events — translucent dome with identical dots
- [ ] **Council visualization** — constellation pattern, unique color threads, glow during active communication
- [ ] **Whisper Chain indicator** — subtle pulse (recipient only, per privacy spec)

### Interaction (`pkg/pulsemap/interaction/`)

- [x] Touch/mouse click detection and node selection
- [x] Pan gesture (single-touch)
- [x] Pinch-to-zoom (two-finger)
- [x] Tap detection
- [x] Long-press detection
- [ ] **Momentum scrolling** — inertial pan with deceleration
- [ ] **Double-tap/click** — node centering zoom
- [ ] **Quick-Action Radial Menu** — right-click/long-press context menu with options:
  - [ ] Compose Wave to node
  - [ ] Send Phantom Gift
  - [ ] Place Specter Mark
  - [ ] Send Whisper
  - [ ] Join active mini-game
  - [ ] View node detail
- [ ] **Node Detail Panel** — slide-in card with:
  - [ ] Profile information (display name, Sigil, public key fingerprint)
  - [ ] Recent Waves list
  - [ ] Connection list
  - [ ] Specter Resonance (for anonymous nodes)
  - [ ] Interaction options
- [ ] **Search bar** — find by display name, fingerprint, or pseudonym
- [ ] **Bookmarks** — save and navigate to specific nodes
- [ ] **"Find Self" button** — center view on own node

### Zoom & Navigation

- [ ] **Continuous smooth zooming** with level-of-detail transitions
- [ ] **Macro View** — full network, colored dots, cluster visibility
- [ ] **Meso View** — 50–200 node neighborhood
- [ ] **Micro View** — 5–20 nodes at full detail
- [ ] **Ego-centric view** (default, own node centered)
- [ ] **Network-centric view** (alternative, global perspective)
- [ ] **Viewport controls** — buttons for Macro/Meso/Micro preset zoom levels

### Background & Atmosphere

- [ ] **Map background** — dark blue-gray gradient with procedural noise
- [ ] **Ambient particle field** — sparse drifting particles
- [ ] **Framebuffer layering** — separate layers composited for background/nodes/overlays/UI

### Rendering Pipeline Performance

- [ ] **Batched draw calls** — grouped rendering by type
- [ ] **Level-of-detail culling** — skip detail below zoom threshold
- [ ] **GPU particle system** — hardware-accelerated particle rendering
- [ ] **60 FPS target** with 500 visible nodes
- [ ] **30 FPS minimum** acceptable threshold
- [ ] **10,000 visible nodes** at Meso zoom without frame drop
- [ ] **100,000 total nodes** with viewport culling
- [ ] **Memory <256 MiB** during normal operation

---

## Milestone v0.9 — Onboarding & Growth

### Phase 1: Welcome (`pkg/onboarding/`)

- [x] Flow controller with 6-phase state machine
- [x] PhaseProgress tracking (started, completed, timing)
- [x] Callback hooks (OnPhaseStart, OnPhaseComplete, OnFlowComplete)
- [ ] **Animated pulsing node** — welcome screen centerpiece
- [ ] **Philosophy screen** — three sequential statements about MURMUR principles
- [ ] **"Begin" button** with 2-second intentional delay

### Phase 2: Identity Creation

- [x] Name entry screen with validation
- [x] Keypair generation with Ed25519
- [ ] **Keypair generation animation** — visual representation of key creation
- [ ] **Public key fingerprint display** — truncated hex shown to user
- [ ] **Display name input** with live Pulse Map preview of own node
- [ ] **Key backup options**:
  - [ ] Save encrypted file
  - [ ] Generate BIP-39 recovery phrase
  - [ ] Skip backup (with warning)

### Phase 3: Mode Selection

- [x] Mode selection screen (Surface vs Fortress)
- [ ] **Mode introduction animation** — Surface + Anonymous Layer visual explanation
- [ ] **Three mode cards** — Open, Hybrid, Fortress with descriptions
- [ ] **Guarded mode card** — missing from current implementation
- [ ] **Context-sensitive guidance** — recommendations based on user profile
- [ ] **Specter identity generation** — triggered for Hybrid/Guarded/Fortress selection
- [ ] **Specter key backup** — separate backup flow for Anonymous Layer key
- [ ] **Configuration confirmation** — summary before committing

### Phase 4: Network Bootstrap

- [x] Bootstrap screen with peer connection
- [x] DHT bootstrap integration
- [ ] **Connection visualization** — expanding dots animation as peers connect
- [ ] **Target 5 peers** — progress indicator toward connection goal
- [ ] **Peer exchange protocol discovery** — find additional peers through connected peers
- [ ] **Shroud circuit establishment** — for Hybrid/Guarded/Fortress (visual: shield icon + "Establishing")
- [ ] **Gossip topic subscription** — subscribe to all relevant GossipSub topics
- [ ] **Troubleshooting guidance** — help text for connection failures
- [ ] **Retry logic** — automatic reconnection on bootstrap failure

### Phase 5: Guided Exploration

- [ ] **Pulse Map introduction tooltip** — "This is your network"
- [ ] **Node explanation** — what nodes represent
- [ ] **Connection explanation** — what edges mean
- [ ] **Wave publishing tutorial** — how to create first Wave
- [ ] **Layer introduction** — Anonymous Layer for Hybrid/Fortress users
- [ ] **Anonymous mechanics preview** — teaser of Phantom Gifts, Marks, mini-games
- [ ] **Connection suggestion** — invite friend, browse nearby, or explore

### Phase 6: First Action

- [ ] **First Wave compose prompt** — guided Wave creation
- [ ] **"Hello, MURMUR" suggestion** — default first Wave text
- [ ] **PoW "Minting…" animation** — visual feedback during PoW computation
- [ ] **Propagation ripple visualization** — see Wave spread across Pulse Map
- [ ] **Tutorial overlay dismissal** — option to close all guidance

### Post-Onboarding

- [ ] **First-Week Nudges**:
  - [ ] Wave publication encouragement (Day 1)
  - [ ] Connection formation suggestion (Day 2)
  - [ ] Layer exploration prompt for Hybrid/Fortress (Day 3)
  - [ ] Milestone celebration at first Resonance threshold (Day 5–7)
- [ ] **Returning User Experience** — existing identity detection, fast bootstrap
- [ ] **Identity Recovery**:
  - [ ] Key file import
  - [ ] Recovery phrase entry
  - [ ] Offline recovery (no network required)

### Invitation & Growth

- [ ] **Invitation generation** — two-tap frictionless invite creation
- [ ] **Invite encoding** — URL-safe Base64, ~100–150 characters: `murmur://invite/[Base64]`
- [ ] **QR code rendering** — shareable QR for invite URL
- [ ] **Sharing integration** — system share sheet (text, email, QR)
- [ ] **Invitation acceptance** — integrated into onboarding flow
- [ ] **Bootstrap advantage** — invitee placed in inviter's Pulse Map neighborhood
- [ ] **Warm start** — first connection pre-formed between inviter and invitee

---

## Milestone v1.0 — Production Readiness

### Application Wiring (`pkg/app/`)

- [x] Subsystem initialization (storage, identity, networking)
- [x] Context lifecycle management
- [x] First-run detection
- [ ] **Event bus goroutine** — central fan-out for all cross-subsystem events
- [ ] **~8 persistent goroutines** per TECHNICAL_IMPLEMENTATION.md §8:
  - [ ] Main/Ebitengine loop
  - [ ] Network/libp2p swarm event handler
  - [ ] Layout/force-directed computation
  - [ ] Expiry/GC sweep (every 60s)
  - [ ] Heartbeat (every 30s on `/murmur/pulse/1`)
  - [ ] Shroud maintenance (circuit lifecycle)
  - [ ] Event bus (fan-out)
  - [ ] DHT refresh
- [ ] **UI renderer orchestration** — Ebitengine Game interface with Update()/Draw() delegation
- [ ] **Graceful shutdown** — ordered subsystem teardown with timeout
- [ ] **Cold start <5 seconds, warm start <2 seconds** performance targets

### Security Hardening

- [x] `pkg/security/` — security audit trail and threat assessment
- [ ] **Key material zeroing** — zero sensitive bytes before GC eligibility
- [ ] **Keystore separation** — Surface and Specter keys in separate encrypted files
- [ ] **PoW verification before signature** — ordering per SECURITY_PRIVACY.md
- [ ] **Signed DHT records** — prevent DHT poisoning
- [ ] **Multi-region connection diversity** — eclipse attack resistance
- [ ] **Rate limiting per peer** — per-peer message rate caps
- [ ] **Gossip flooding defense** — PoW cost + peer scoring + rate limits
- [ ] **Relay attack mitigation** — Shroud circuits with mixing and timing padding

### Monitoring & Observability

- [ ] Prometheus metrics integration (connection count, message rates, Resonance distribution)
- [ ] OpenTelemetry tracing for subsystem interactions
- [ ] Health check endpoint for bootstrap node operators
- [ ] Memory usage monitoring (<256 MiB budget enforcement)
- [ ] Bbolt database size monitoring (<50 MiB budget)
- [ ] GC sweep duration monitoring (<100ms target)

### Testing

- [x] Unit tests for cryptographic operations (Ed25519 round-trip, PoW verification)
- [x] Unit tests for game mechanics (puzzles, hunts, territory, oracle, forge, shadowplay)
- [x] Unit tests for touch interactions, overlays, colors, rendering
- [x] Stability simulation infrastructure (1000-node, 72-hour)
- [ ] **Integration tests** — in-memory Bbolt + mock event bus, no libp2p, no Ebitengine
- [ ] **Simulation tests** — 10–100 in-process libp2p nodes with memory transports (`//go:build simulation`)
  - [ ] Gossip propagation latency verification (<3s to 99% of subscribers)
  - [ ] Shroud anonymity verification (no single node knows origin + destination)
  - [ ] Resonance convergence verification across network
  - [ ] Wave TTL expiration correctness
  - [ ] Mini-game network propagation end-to-end
- [ ] **Coverage targets**: >80% for `pkg/identity/`, `pkg/content/`, `pkg/anonymous/`
- [ ] **Ebitengine headless mode** screenshot comparison tests for rendering

### Documentation

- [x] README.md — project overview
- [x] CHANGELOG.md — implementation history
- [x] CONTRIBUTING.md — contributor guidelines
- [x] docs/BOOTSTRAP_OPERATION.md — bootstrap node procedures
- [x] docs/DEPLOYMENT.md — deployment guide
- [x] docs/SHROUD_OPERATION.md — Shroud relay operation
- [ ] AUDIT.md — security decisions and deviations log
- [ ] PLAN.md — sprint-level task tracking
- [ ] API documentation for all exported types and functions
- [ ] Architecture decision records (ADRs) for key design choices

### Deployment

- [ ] Bootstrap node infrastructure (8–12 community-operated nodes)
- [ ] Docker container image for bootstrap/relay operators
- [ ] Platform-specific binary releases (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- [ ] Android APK distribution
- [ ] iOS xcframework distribution
- [ ] Version upgrade protocol — dual-subscription migration (v1 + v2 topics)

---

## Cross-Cutting Concerns

### Anti-Sybil & Spam Resistance

- [x] Proof of Work computational cost per Wave
- [x] Peer scoring with invalid message penalties
- [ ] Rate limiting per peer (configurable msg/sec caps)
- [ ] Resonance gating on all privileged actions (gifts, marks, games, councils)
- [ ] Connection pruning for consistently low-score peers
- [ ] PoW requirement for identity creation
- [ ] Sybil defense: PoW cost scales linearly per identity

### Protocol Versioning

- [ ] Topic versioning in GossipSub topic strings
- [ ] MurmurEnvelope version field handling (currently always 1)
- [ ] Protocol negotiation via multistream-select
- [ ] Gradual migration: new-version nodes subscribe to both v1 and v2 topics
- [ ] Breaking change consensus mechanism

### Accessibility & UX

- [ ] Compose panel — Wave input interface with character count
- [ ] Settings panel — configuration options (privacy mode, difficulty, filters)
- [ ] Help button — reopen onboarding tutorials at any time
- [ ] Modal dialogs — confirmations and warnings for destructive actions
- [ ] Status indicators — identity publication, Shroud circuit, connection count, PoW progress

---

## Summary Statistics

| Category | Implemented | Remaining | Total |
|----------|------------|-----------|-------|
| Core Foundation | 15 | 8 | 23 |
| Network & Messaging | 16 | 37 | 53 |
| Content System | 12 | 21 | 33 |
| Identity & Privacy | 9 | 18 | 27 |
| Anonymous Layer Core | 8 | 19 | 27 |
| Resonance & Reputation | 8 | 28 | 36 |
| Mini-Games & Mechanics | 41 | 62 | 103 |
| Pulse Map & Visualization | 18 | 55 | 73 |
| Onboarding & Growth | 7 | 33 | 40 |
| Production Readiness | 9 | 27 | 36 |
| Cross-Cutting | 2 | 10 | 12 |
| **Total** | **145** | **318** | **463** |

> **Implementation progress: ~31% complete** — Core data structures and game mechanics are
> solid, but network integration, UI rendering, persistence, and cross-subsystem wiring
> represent the majority of remaining work.
