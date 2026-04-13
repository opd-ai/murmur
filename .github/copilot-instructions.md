# Project Overview

MURMUR is a decentralized, peer-to-peer social network with a dual-layer identity architecture. There are no servers, no algorithms, and no permanent record. Every participant's device is both client and server — a node in a living mesh. The network presents itself through the Pulse Map, a force-directed graph where users are glowing nodes, relationships are visible edges, and content ripples outward through the mesh. Beneath the Surface Layer lies an Anonymous Layer of pseudonymous identities called Specters, routed through onion-style Shroud circuits, with their own Resonance reputation system and social mechanics.

MURMUR is **pre-implementation**. The project currently consists of comprehensive design specification documents (~552KB across 15+ markdown files) with zero Go source code, no `go.mod`, no tests, and no CI. The specification documents are the authoritative source for all implementation decisions. The Copilot instructions in this file guide AI assistants in building the codebase from scratch according to those specifications, not maintaining an existing one. Always consult the relevant specification document before implementing any subsystem.

The primary technology stack is **Go 1.22+** (goroutines and channels for concurrency), **Ebitengine v2.7+** (2D rendering, Kage shaders, immediate-mode UI), **go-libp2p v0.36+** (GossipSub v1.1, Kademlia DHT, Noise transport, NAT traversal), **Bbolt** (embedded key-value storage), and **Protocol Buffers proto3** (all wire-format serialization). Target audience: privacy-conscious users, self-sovereign identity advocates, and communities wanting anonymous social mechanics as a first-class feature. Unique differentiators: Pulse Map spatial UI, Shadow Gradient (four privacy modes: Open/Hybrid/Guarded/Fortress), Specter anonymous identities, Resonance reputation, and ephemeral Waves.

## Technical Stack

- **Primary Language**: Go 1.22+ (goroutines, channels, context cancellation as concurrency model)
- **Rendering**: Ebitengine v2.7+ (2D game engine, `Update()`/`Draw()` tick model, Kage shaders for glow/ripple/spectra effects, `ebiten/v2/text/v2` for text)
- **Networking**: go-libp2p v0.36+ (Noise XX for transport encryption, yamux for stream multiplexing, GossipSub v1.1 with peer scoring, Kademlia DHT, DCUtR hole punching, relay, AutoNAT, identify, ping)
- **Cryptography**: `crypto/ed25519` (Surface Layer signing), `golang.org/x/crypto/curve25519` (Anonymous Layer key exchange), `golang.org/x/crypto/chacha20poly1305` (XChaCha20-Poly1305 symmetric encryption), `crypto/sha256` (PoW, content addressing), `github.com/zeebo/blake3` (identity hashing, sigil seeds, message_id in envelopes), `golang.org/x/crypto/argon2` (Argon2id passphrase-based key derivation), `github.com/bwesterb/go-ristretto` (Ristretto point conversion for ZK proofs), Pedersen commitments + Bulletproofs (ZK Resonance claims)
- **Storage**: Bbolt (`go.etcd.io/bbolt`) — embedded key-value store, single-file database, ACID transactions, memory-mapped reads
- **Serialization**: Protocol Buffers proto3 (`google.golang.org/protobuf`); JSON is never used on the wire or in storage
- **Testing**: Go built-in `testing` package for unit tests; in-process libp2p hosts with memory transports for integration tests; 10–100 node simulation tests behind `//go:build simulation` tag; no Ebitengine dependency in non-rendering tests
- **Build/Deploy**: `go build` single static binary per platform; `go:embed` for assets (wordlists, default config, onboarding); target platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- **Formatting/Linting**: `gofumpt -w -extra .` for formatting, `go vet ./...` for static analysis, `go-stats-generator` and additional static analysis toolchain; all code must be linter-clean before commit

## Code Assistance Guidelines

1. **Use `pkg/` for all packages, never `internal/`.** The project uses `pkg/` as the top-level source directory per the ROADMAP.md implementation plan. Each subsystem is a separate directory under `pkg/`. Do not use `internal/` despite references in TECHNICAL_IMPLEMENTATION.md — the ROADMAP (the newer implementation plan) supersedes it. The planned layout is:
   ```
   cmd/murmur/main.go              # Entry point, dependency wiring, ebiten.RunGame()
   proto/                           # .proto files (wave.proto, identity.proto, shroud.proto, gossip.proto, resonance.proto)
   pkg/
   ├── app/                         # Top-level application struct, lifecycle, event bus
   ├── config/                      # Configuration loading, defaults, validation
   ├── networking/
   │   ├── transport/               # libp2p host construction, Noise/QUIC/TCP transport config
   │   ├── gossip/                  # GossipSub setup, topic management, peer scoring
   │   ├── discovery/               # Kademlia DHT bootstrap, peer routing
   │   ├── relay/                   # NAT traversal, DCUtR hole punching, relay fallback
   │   └── mesh/                    # Peer scoring, mesh health, target 6–12 peers
   ├── identity/
   │   ├── keys/                    # Ed25519/Curve25519 generation, keystore, backup
   │   ├── sigils/                  # Deterministic visual identity generation
   │   ├── declarations/            # Profile declarations, trust anchors
   │   └── modes/                   # Open/Hybrid/Guarded/Fortress privacy mode state machine
   ├── content/
   │   ├── waves/                   # Wave creation, signing, validation, types
   │   ├── pow/                     # SHA-256 Proof of Work (2–5s target, difficulty 20)
   │   ├── propagation/             # Gossip relay, hop counting, deduplication
   │   ├── threads/                 # Reply chain indexing, conversation reconstruction
   │   └── storage/                 # Local cache, TTL enforcement, garbage collection
   ├── anonymous/
   │   ├── specters/                # Specter identity creation, name generation
   │   ├── shroud/                  # Three-hop onion circuit construction, cell encryption
   │   ├── resonance/               # Local reputation computation, rank thresholds, decay
   │   └── mechanics/               # Phantom Gifts, Mini-Games, Marks, Events, Councils
   ├── store/                       # Bbolt initialization, bucket CRUD per domain
   ├── pulsemap/
   │   ├── layout/                  # Force-directed graph engine (Fruchterman-Reingold, Barnes-Hut)
   │   ├── rendering/               # Ebitengine Draw(), camera transforms, node/edge drawing
   │   ├── interaction/             # Pan, zoom, node selection, navigation
   │   └── overlays/                # Anonymous layer overlay, activity heatmap
   └── onboarding/
       ├── flow/                    # Six-phase sequence controller
       ├── tutorials/               # Guided exploration, contextual hints
       └── bootstrap/               # Initial peer connection, first Wave prompt
   ```

2. **One purpose per package, separated by directories.** Each package serves a single responsibility. For example, `pkg/content/pow/` handles only Proof of Work computation and verification; `pkg/content/waves/` handles only Wave creation, signing, and validation. Do not merge unrelated concerns into a single package. When in doubt, split.

3. **Always provide complete implementations.** If partial implementations are absolutely required, insert clear inline `// TODO:` comments describing what remains, why it was deferred, and any known constraints. Example: `// TODO: Implement Shroud circuit rotation once relay discovery is available (see TECHNICAL_IMPLEMENTATION.md §4.5)`. Never leave code in a silently incomplete state.

4. **Design all exported types and functions with logical extension points** for both in-repo and downstream use. Prefer interfaces for subsystem boundaries — for example, define a `store.WaveStore` interface allowing mock implementations for testing. Every subsystem should be testable with mock dependencies injected through interfaces.

5. **Communicate between goroutines exclusively via typed channels.** No shared mutable state without channel synchronization or atomic operations. Follow the concurrency model from TECHNICAL_IMPLEMENTATION.md §8: ~8 persistent goroutines (main/Ebitengine loop, network/libp2p swarm, layout/force-directed, expiry/GC, heartbeat, Shroud maintenance, event bus, DHT refresh) with a central event bus goroutine for fan-out. The double-buffered Pulse Map node positions use `atomic.Pointer` swaps — this is the only exception to channel-only communication.

6. **Use the specified cryptographic primitives exactly.** Do not substitute algorithms. The mapping is:
   | Use Case | Algorithm | Package |
   |---|---|---|
   | Surface Layer signatures (identity, Waves, connections) | Ed25519 | `crypto/ed25519` |
   | Anonymous Layer key exchange (Shroud circuits, Whisper Chains) | X25519 (Curve25519 DH) | `golang.org/x/crypto/curve25519` |
   | Symmetric encryption (Shroud onion layers, keystore, Phantom Councils) | XChaCha20-Poly1305 | `golang.org/x/crypto/chacha20poly1305` |
   | Proof of Work and content addressing (Wave IDs, deduplication) | SHA-256 | `crypto/sha256` |
   | Identity hashing (sigils, pseudonyms, `message_id` in envelopes) | BLAKE3 | `github.com/zeebo/blake3` |
   | Passphrase-based key derivation (keystore encryption) | Argon2id (time=3, memory=64 MiB, threads=4, output=32 bytes) | `golang.org/x/crypto/argon2` |
   | ZK Resonance claims (threshold proofs) | Pedersen commitments + Bulletproofs | `github.com/bwesterb/go-ristretto` |
   | Key derivation from DH shared secrets | HKDF-SHA-256 | `golang.org/x/crypto/hkdf` |

7. **All wire-format messages use Protocol Buffers proto3.** Define messages in `proto/*.proto` files, generate Go code with `protoc-gen-go`, and check generated `.pb.go` files into the repository (so contributors can build without installing `protoc`). Every GossipSub message must be wrapped in the `MurmurEnvelope` protobuf:
   ```protobuf
   message MurmurEnvelope {
     uint32 version = 1;           // Protocol version (currently 1)
     MessageType type = 2;         // Enum: WAVE, IDENTITY, SHROUD_AD, HEARTBEAT
     bytes payload = 3;            // Serialized inner message (type-specific protobuf)
     bytes sender_pubkey = 4;      // 32-byte Ed25519 public key (zeroed for anonymous)
     bytes signature = 5;          // Ed25519 signature over (version || type || payload)
     int64 timestamp_unix = 6;     // Sender's Unix timestamp in seconds
     bytes message_id = 7;         // 32-byte BLAKE3 hash of payload (deduplication key)
   }
   ```
   Nodes reject envelopes with timestamps more than 300 seconds in the future or more than the message's TTL in the past.

8. **Write linter-clean, `gofumpt`-formatted code at all times.** All Go code must pass `go vet ./...` and static analysis via `go-stats-generator` and other linting tools without warnings. Format all code with `gofumpt -w -extra .` before committing. Prefer explicit error handling, named return values where they improve clarity, and consistent naming conventions per Effective Go. No `nolint` directives without a documented justification comment.

9. **Update planning documents after every completed task.** After finishing any implementation task, update the following project documents to reflect current state:
   - **`CHANGELOG.md`** — append an entry describing what was implemented, changed, or fixed, with the date
   - **`AUDIT.md`** — record any security-relevant decisions, deviations from spec, or areas requiring future review
   - **`PLAN.md`** — mark completed items, adjust estimates, and note any discovered blockers or scope changes
   - **`ROADMAP.md`** — update the goal-achievement table, check off completed roadmap items, and revise milestone status

   These documents are the project's source of truth for progress tracking. Stale planning docs are treated as a defect.

10. **Complete Feature Implementations**: Always prefer completing the full implementation of any feature rather than leaving partial or placeholder code. When a complete implementation is not feasible, insert clear inline `// TODO:` comments describing what remains, why it was deferred, and any known constraints (e.g., `// TODO: Implement retry logic once the error categorization schema is finalized`). Never leave code in a silently incomplete state.

## Project Context

### Domain

MURMUR uses precisely defined domain terminology from GLOSSARY.md. Use exact terms in code identifiers, protobuf message names, topic strings, and documentation:

| Term | Meaning | Do NOT Use |
|---|---|---|
| **Wave** | Signed, ephemeral content unit (≤2048 bytes) with PoW and TTL | "message", "post", "tweet" |
| **Specter** | Pseudonymous anonymous identity (Curve25519 keypair, procedural name/sigil) | "anonymous user", "alt account" |
| **Shroud** | Three-hop onion routing network for anonymous traffic | "onion router", "Tor-like network" |
| **Resonance** | Locally-computed reputation metric with milestone unlocks | "reputation score", "karma" |
| **Pulse Map** | Real-time force-directed graph — the primary spatial UI | "graph view", "network map" |
| **Phantom Gift** | One-way anonymous gift from Specter to Surface identity | "anonymous gift" |
| **Specter Mark** | Anonymous annotation on Surface node visible on Pulse Map | "anonymous comment" |
| **Shadow Gradient** | Four privacy modes: Open → Hybrid → Guarded → Fortress | "privacy levels" |
| **Sigil** | Deterministic visual icon from public key hash (64×64) | "avatar", "profile picture" |

The 7 design principles in priority order (higher overrides lower):
1. Privacy is structural, not contractual
2. No permanent record by default (Waves expire, max 30-day TTL)
3. Identity is self-sovereign (Ed25519 keypair, no registration server)
4. The network is the interface (Pulse Map is the social space)
5. Anonymity is a first-class feature (Anonymous Layer with own mechanics)
6. Growth must be organic (no paid acquisition, no algorithmic engagement)
7. Complexity is revealed, not imposed (progressive disclosure)

The 6 subsystems: **Networking** (libp2p transport, GossipSub, Kademlia DHT, NAT traversal), **Identity** (Ed25519/Curve25519 keypairs, sigils, privacy modes), **Content** (Waves with PoW, TTL, threading, amplification), **Anonymous Layer** (Specters, Shroud onion routing, Resonance, mini-games), **Pulse Map** (force-directed graph visualization), **Onboarding** (six-phase guided introduction).

### Architecture

- **Acyclic dependency graph**: `cmd/murmur` → `pkg/app` → all subsystem packages. `pkg/pulsemap` depends on Ebitengine; **no other package under `pkg/` imports Ebitengine**. `pkg/content`, `pkg/identity`, `pkg/anonymous` depend on `pkg/store` and networking interfaces but not on each other except through the event bus. `pkg/store` depends on no other internal package — it operates on protobuf-generated types only.
- **Event bus pattern**: A central event bus goroutine receives all events (network, timer, user action) and fans them out to subscriber channels. Each subsystem registers interest at startup.
- **Double-buffered Pulse Map**: The layout goroutine writes node positions to a back buffer and atomically swaps with the front buffer (via `atomic.Pointer`), which the Ebitengine `Draw()` call reads. Zero lock contention between simulation and rendering.
- **~8 persistent goroutines**: main (Ebitengine loop), network (libp2p swarm), layout (force-directed), expiry (GC every 60s), heartbeat (every 30s on `/murmur/pulse/1`), Shroud maintenance (circuit lifecycle), event bus (fan-out), DHT refresh. Transient goroutines for PoW computation and Shroud circuit construction.

### Key Directories

- `pkg/` — all subsystem source packages (networking, identity, content, anonymous, pulsemap, onboarding, store, app, config)
- `cmd/murmur/` — entry point (`main.go`), dependency wiring, `ebiten.RunGame()`
- `proto/` — `.proto` files (`wave.proto`, `identity.proto`, `shroud.proto`, `gossip.proto`, `resonance.proto`); generated `.pb.go` files checked in
- `assets/` — embedded resources: `wordlists/specter-names.txt` (65,536 entries), themes, onboarding assets
- `pulsemap/shaders/` — Kage shader files: `glow.kage`, `ripple.kage`, `spectra.kage`

### Configuration

**Bbolt bucket names** (canonical): `identity`, `peers`, `waves`, `threads`, `shroud`, `resonance`, `config`. Each bucket has CRUD operations under `pkg/store/`.

**GossipSub topic strings** (canonical):
- `/murmur/waves/1` — Wave messages (publications, replies, amplifications)
- `/murmur/identity/1` — Identity declarations and connection announcements
- `/murmur/shroud/1` — Shroud relay advertisements
- `/murmur/pulse/1` — Lightweight heartbeat pings (64-byte signed timestamp, every 30s)

**Stream protocol IDs** (canonical):
- `/murmur/shroud-circuit/1` — Shroud onion circuit construction (Curve25519 key exchange, relay cells)
- `/murmur/wave-sync/1` — Request-response for fetching Waves by hash
- `/murmur/peer-exchange/1` — Peer list exchange during bootstrap

**Build targets**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. Single static binary per platform via `go build`. All assets embedded via `go:embed`. No runtime file dependencies except the Bbolt database file created on first run.

**Wave types**: Surface Wave (0x01), Reply Wave (0x02), Veiled Wave (0x03), Specter Wave (0x04), Sigil Wave (0x05), Abyssal Wave (0x06), Masked Wave (0x07), Beacon Wave (0x08).

**Resonance milestones**: Shade (25), Wraith (50), Shade-Wraith (75), Phantom (100), Council-Eligible (200), Abyss (500, Fortress only).

### Specification Documents

These markdown files are the authoritative source for all implementation decisions:

| Document | Role |
|---|---|
| `README.md` | Project overview, architecture summary, quick reference |
| `TECHNICAL_IMPLEMENTATION.md` | Technology stack, module architecture, wire protocols, concurrency model, PoW, Resonance computation, performance targets, build strategy, testing strategy |
| `DESIGN_DOCUMENT.md` | Complete specification: 7 design principles, 6 subsystems in full detail, all cryptographic primitives, all subsystem behavior |
| `ROADMAP.md` | Goal-achievement assessment, 12-priority implementation plan, `pkg/` package structure, success milestones (v0.1–v1.0) |
| `SECURITY_PRIVACY.md` | Threat model (4 adversary classes), cryptographic primitives, attack surface analysis, privacy guarantees per mode |
| `SHADOW_GRADIENT.md` | Four privacy modes (Open/Hybrid/Guarded/Fortress), identity isolation, Shroud Network, mode transitions |
| `WAVES.md` | 8 Wave types (0x01–0x08), Wave structure fields, PoW algorithm, propagation mechanics, content window |
| `RESONANCE_SYSTEM.md` | Surface + Specter Resonance scoring formulas, input signals, milestones, Echo Index |
| `NETWORK_ARCHITECTURE.md` | libp2p foundation, transport layer, peer discovery, NAT traversal, GossipSub topics, topology management |
| `GLOSSARY.md` | All MURMUR-specific terminology definitions |
| `ANONYMOUS_GAME_MECHANICS.md` | Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Phantom Councils |
| `PULSE_MAP.md` | Pulse Map layout engine, rendering pipeline, camera system, visual effects |
| `ONBOARDING.md` | Six-phase onboarding flow, guided identity creation, Pulse Map tutorial |
| `WAVE_PROPAGATION.md` | Gossip propagation mechanics, hop counting, deduplication, TTL enforcement |
| `PRODUCT_VISION.md` | Product positioning, user journeys, growth philosophy |
| `VIRAL_GROWTH_AND_ONBOARDING.md` | Organic growth mechanics, invitation flow, network bootstrapping |

### Planning Documents

The following documents must be updated after every completed implementation task:

- **`CHANGELOG.md`** — Append dated entries for what was implemented, changed, or fixed
- **`AUDIT.md`** — Record security-relevant decisions, spec deviations, areas needing future review
- **`PLAN.md`** — Mark completed items, adjust estimates, note discovered blockers
- **`ROADMAP.md`** — Update goal-achievement table, check off completed items, revise milestone status

## Quality Standards

- **Formatting**: All code formatted with `gofumpt -w -extra .`. All code passes `go vet ./...`. Clean output from `go-stats-generator` and static analysis. No `nolint` directives without documented justification.

- **Testing**: Unit tests for all cryptographic operations (Ed25519 signing round-trips, PoW verification at boundary difficulties, Shroud onion encryption/decryption, Argon2id key derivation), data structure operations (graph manipulation, Resonance computation, TTL enforcement), and protobuf serialization round-trips. Integration tests using in-memory Bbolt stores and mock event buses — no libp2p host, no Ebitengine window. Simulation tests constructing 10–100 in-process libp2p nodes with memory transports (behind `//go:build simulation` tag) to verify gossip propagation, Shroud anonymity, and Resonance convergence. Target coverage: >80% for `pkg/identity/`, `pkg/content/`, `pkg/anonymous/`. No tests depend on Ebitengine; the rendering layer is tested via Ebitengine's headless mode and screenshot comparison.

- **Performance**: 60fps rendering with 500 visible nodes and 2,000 edges (Barnes-Hut for >500 nodes). Wave propagation latency <500ms across 3 hops. PoW computation 2–5 seconds at default difficulty (20 leading zero bits). Shroud circuit construction <3 seconds. Cold start <5 seconds, warm start <2 seconds. Memory <256 MiB during normal operation. Bbolt DB <50 MiB. GC sweep <100ms.

- **Security**: Use exact cryptographic primitives as specified — no algorithm substitution. Ed25519 for Surface, Curve25519 for Anonymous, ChaCha20-Poly1305 for symmetric, SHA-256 for PoW, BLAKE3 for identity hashing, Argon2id for key derivation. Shroud circuit construction must ensure hop diversity (no two hops in the initiator's direct mesh). Key material zeroed before backing arrays become GC-eligible. Surface and Specter keypairs share no derivation path — compromising one must not reveal the other.

- **Documentation**: All design decisions trace to specification documents. When implementing a subsystem, reference the relevant spec section in code comments (e.g., `// Per TECHNICAL_IMPLEMENTATION.md §4.4, PoW difficulty is 20 leading zero bits`). Planning documents (`CHANGELOG.md`, `AUDIT.md`, `PLAN.md`, `ROADMAP.md`) updated after every completed task — stale planning docs are treated as a defect.
