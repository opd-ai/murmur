# MURMUR — Technical Implementation

This document specifies the concrete implementation plan for MURMUR: languages, libraries, data formats, wire protocols, cryptographic constructions, algorithms, and performance targets. It bridges the design document (which describes *what*) and the codebase (which describes *how*).

---

## 1. Technology Stack

### 1.1 Language

MURMUR is implemented in Go (minimum version 1.22). Go's concurrency model — goroutines and channels — maps naturally onto the demands of a peer-to-peer application that must simultaneously manage network I/O across dozens of peer connections, run a real-time rendering loop, compute force-directed graph layouts, construct onion-routed circuits, and enforce Wave expiration. The garbage collector's sub-millisecond pause times in modern Go are acceptable for a 60fps rendering target. The single-binary compilation model simplifies distribution to end users who should never need to install a runtime.

### 1.2 Rendering

The user interface and Pulse Map are rendered with Ebitengine (v2.7+). Ebitengine provides a hardware-accelerated 2D rendering loop with a clean `Update()`/`Draw()` tick model that separates simulation logic from frame production. The Pulse Map's force-directed graph, ripple effects, node sigils, and ghost-layer overlays are all drawn through Ebitengine's image compositing and shader pipeline. Ebitengine's built-in support for custom Kage shaders is used for the Anonymous Layer's spectral glow effects, ripple propagation visuals, and node activity halos. Text rendering uses `ebiten/v2/text/v2` backed by OpenType fonts.

The UI layer — panels, menus, onboarding screens, input fields — is built as a lightweight immediate-mode system on top of Ebitengine's drawing primitives rather than importing a full GUI framework. This keeps the dependency tree shallow and ensures the entire application renders through a single pipeline.

### 1.3 Networking

The networking layer is built on `go-libp2p` (v0.36+). The specific modules in use are `noise` for authenticated transport encryption, `yamux` for stream multiplexing, `pubsub` (GossipSub) for topic-based message propagation, `dht` (Kademlia) for peer discovery and content-addressable routing, `relay` and `holepunch` for NAT traversal, `identify` for peer metadata exchange, and `ping` for liveness checks.

The libp2p host runs in its own goroutine. Communication between the network layer and the rest of the application flows through typed Go channels. A central event bus goroutine receives inbound network events (new peer, incoming Wave, Shroud circuit request) and dispatches them to the appropriate subsystem.

### 1.4 Cryptography

Cryptographic operations use Go's standard library and `golang.org/x/crypto` wherever possible. Ed25519 signing and verification use `crypto/ed25519`. Curve25519 Diffie-Hellman for Shroud circuit key exchange uses `golang.org/x/crypto/curve25519`. Symmetric encryption of Shroud onion layers uses XChaCha20-Poly1305 via `golang.org/x/crypto/chacha20poly1305`. SHA-256 for Proof of Work uses `crypto/sha256`. BLAKE3 for content-addressable Wave hashing and sigil seed generation uses `github.com/zeebo/blake3`. Key material held in memory is overwritten before the backing array becomes eligible for garbage collection; while Go does not guarantee zeroization, explicit zeroing of byte slices before releasing references is the practical best-effort measure.

### 1.5 Storage

Local persistence uses Bbolt (`go.etcd.io/bbolt`), an embedded key-value store derived from Howard Chu's LMDB design. Bbolt provides ACID transactions, memory-mapped reads, and a single-file database suitable for a desktop application that must not depend on external processes. The database is organized into the following buckets:

`identity` stores the local node's Surface keypair, Specter keypair(s), identity declarations, and privacy mode configuration. `peers` stores known peer records: public keys, last-seen timestamps, connection metadata, and trust scores. `waves` stores cached Wave content indexed by BLAKE3 hash, with TTL expiration metadata. `threads` stores reply-chain indices mapping parent Wave hashes to child Wave hashes. `shroud` stores active Shroud circuit state, relay peer lists, and circuit rotation schedules. `resonance` stores locally observed Specter interaction data used to compute Resonance scores.

A background goroutine runs every 60 seconds to scan the `waves` bucket, delete expired entries, and reclaim space via Bbolt's freelist. This is the garbage collector for ephemeral content.

### 1.6 Serialization

All wire-format messages and persisted records use Protocol Buffers (proto3) via `google.golang.org/protobuf`. Protobuf provides compact binary encoding, a well-defined schema evolution model, and generation of type-safe Go structs. Every message type is defined in `.proto` files under `proto/` and compiled with `protoc-gen-go`. JSON is never used on the wire or in storage. Human-readable debug output uses Protobuf's text format.

---

## 2. Module Architecture

> **Note**: The canonical package structure uses `pkg/` per ROADMAP.md. References to `internal/` in this document are historical; all implementation should use `pkg/` instead.

The application is structured as a set of Go packages under `pkg/`, with a thin `cmd/murmur/` entry point that wires them together. No package under `pkg/` imports the Ebitengine package except `pkg/pulsemap`. This boundary ensures that the networking, identity, content, and anonymous subsystems are testable without a graphics context.

### 2.1 Package Map

    cmd/
    └── murmur/
        └── main.go                 # Entry point, dependency wiring, ebitengine.RunGame()

    proto/
    ├── wave.proto                  # Wave, Reply, Amplification messages
    ├── identity.proto              # Identity declarations, peer records
    ├── shroud.proto                # Circuit construction, onion cell formats
    ├── gossip.proto                # GossipSub envelope wrappers
    └── resonance.proto             # Resonance observation records

    pkg/
    ├── app/
    │   ├── app.go                  # Top-level application struct, lifecycle
    │   └── eventbus.go             # Central event dispatcher (channel fan-out)
    │
    ├── config/
    │   └── config.go               # Configuration loading, defaults, validation
    │
    ├── networking/
    │   ├── transport/              # libp2p host construction, transport config
    │   ├── gossip/                 # GossipSub setup, topic management, scoring
    │   ├── discovery/              # Kademlia DHT bootstrap, peer routing
    │   ├── relay/                  # Relay reservation, hole punching, fallback
    │   └── mesh/                   # Peer scoring, mesh health, target 6-12 peers
    │
    ├── identity/
    │   ├── keys/                   # Ed25519/Curve25519 keypair generation, signing
    │   ├── sigils/                 # Deterministic visual sigil from public key hash
    │   ├── declarations/           # Identity declaration creation and parsing
    │   └── modes/                  # Privacy mode state machine (Open/Hybrid/Guarded/Fortress)
    │
    ├── content/
    │   ├── waves/                  # Wave struct, creation, validation
    │   ├── pow/                    # SHA-256 Proof of Work (2-5s target, difficulty 20)
    │   ├── propagation/            # Gossip relay logic, hop counting, deduplication
    │   ├── threads/                # Reply chain indexing, conversation reconstruction
    │   └── storage/                # Local cache, TTL enforcement, garbage collection
    │
    ├── anonymous/
    │   ├── specters/               # Specter lifecycle, name wordlist, identity
    │   ├── shroud/                 # Onion circuit construction, cell encryption
    │   ├── resonance/              # Resonance computation, rank thresholds, decay
    │   └── mechanics/              # Phantom Gifts, Games, Marks, Events, Councils
    │
    ├── store/
    │   ├── db.go                   # Bbolt initialization, bucket creation, shutdown
    │   └── ...                     # Domain-specific CRUD operations
    │
    ├── pulsemap/
    │   ├── layout/                 # Force-directed layout engine (Fruchterman-Reingold)
    │   ├── rendering/              # Ebitengine Draw(), camera transforms
    │   ├── interaction/            # Pan, zoom, node selection, navigation
    │   └── overlays/               # Anonymous layer overlay, activity heatmap
    │
    └── onboarding/
        ├── flow/                   # Six-phase sequence state machine
        ├── tutorials/              # Guided exploration, contextual hints
        └── bootstrap/              # First peer connection, network join

### 2.2 Dependency Flow

The dependency graph is strictly acyclic and flows in one direction. `cmd/murmur` depends on `pkg/app`. `pkg/app` depends on all subsystem packages and wires them together. `pkg/pulsemap` depends on Ebitengine and on domain packages (`content`, `identity`, `anonymous`) for data, but never on `pkg/networking` directly — all network events arrive through the event bus. `pkg/networking` depends on `pkg/identity` (to sign/verify messages), `pkg/content` (to validate incoming Waves), and `pkg/store` (to persist peer records). `pkg/anonymous` depends on `pkg/networking` for Shroud circuit construction and on `pkg/identity` for Specter key management. `pkg/store` depends on no other internal package; it operates on protobuf-generated types only.

This layering means the entire content, identity, and anonymous subsystem can be tested with an in-memory store and mock event bus, with no libp2p host and no Ebitengine window.

---

## 3. Wire Protocol

### 3.1 GossipSub Topics

All broadcast communication flows through GossipSub topics. MURMUR defines four topic strings.

`/murmur/waves/1` carries Wave messages (new publications, replies, amplifications). Every node subscribes to this topic. The topic uses GossipSub v1.1 with peer scoring enabled: peers that repeatedly send invalid Waves (bad signatures, failed PoW, expired TTL) accumulate negative scores and are eventually pruned from the mesh.

`/murmur/identity/1` carries identity declarations. When a node publishes or updates its identity declaration, the signed declaration is broadcast on this topic. Nodes cache declarations keyed by public key.

`/murmur/shroud/1` carries Shroud relay advertisements. Nodes willing to serve as Shroud relays periodically publish signed advertisements declaring their availability, bandwidth capacity, and supported circuit protocol version.

`/murmur/pulse/1` carries lightweight heartbeat pings — a 64-byte signed timestamp emitted every 30 seconds. These pings drive the Pulse Map's activity-recency coloring and allow nodes to detect peer liveness without relying solely on the DHT.

### 3.2 Stream Protocols

Point-to-point communication uses custom libp2p stream protocols for interactions that are inherently bilateral rather than broadcast.

`/murmur/shroud-circuit/1` is the stream protocol for constructing Shroud onion circuits. The initiator opens a stream to the first hop, performs a Curve25519 key exchange, then sends a relay cell instructing the first hop to extend the circuit to the second hop, and so on for three hops. Each hop sees only the previous and next hop. The cell format is defined in `proto/shroud.proto`.

`/murmur/wave-sync/1` is a request-response protocol for fetching specific Waves by hash. When a node sees a reply referencing a parent Wave it does not have, it opens a sync stream to any peer advertising the Wave's hash in the DHT and requests the missing content.

`/murmur/peer-exchange/1` is a protocol for exchanging peer lists during bootstrap. A new node with no DHT state can open this stream to a known bootstrap peer and receive a set of recently-seen peer addresses to accelerate mesh formation.

### 3.3 Message Envelope

Every GossipSub message is wrapped in a `MurmurEnvelope` protobuf:

    message MurmurEnvelope {
      uint32 version = 1;           // Protocol version (currently 1)
      MessageType type = 2;         // Enum: WAVE, IDENTITY, SHROUD_AD, HEARTBEAT
      bytes payload = 3;            // Serialized inner message (type-specific protobuf)
      bytes sender_pubkey = 4;      // 32-byte Ed25519 public key (zeroed for anonymous)
      bytes signature = 5;          // Ed25519 signature over (version || type || payload)
      int64 timestamp_unix = 6;     // Sender's Unix timestamp in seconds
      bytes message_id = 7;         // 32-byte BLAKE3 hash of payload (deduplication key)
    }

Nodes reject envelopes with timestamps more than 300 seconds in the future or more than the message's TTL in the past. The `message_id` field is used by GossipSub's seen-message cache to prevent reprocessing.

---

## 4. Cryptographic Constructions

### 4.1 Surface Identity

A Surface identity is an Ed25519 keypair generated from 32 bytes of `crypto/rand` entropy. The public key (32 bytes) is the node's permanent identifier. The keypair is serialized as a 64-byte seed+public concatenation and stored encrypted in the `identity` Bbolt bucket. The encryption key is derived from a user-provided passphrase via Argon2id (`golang.org/x/crypto/argon2`) with parameters: 3 iterations, 64 MiB memory, 4 threads, 32-byte output. The encrypted blob is a 24-byte XChaCha20-Poly1305 nonce followed by the ciphertext.

### 4.2 Specter Identity

A Specter identity is a Curve25519 keypair. The private scalar is generated from 32 bytes of `crypto/rand` and clamped per the Curve25519 specification. The public point serves as the Specter's identifier. Specter keypairs are stored in the same encrypted `identity` bucket as the Surface keypair but under a separate key prefix. The two keypairs share no derivation path and no mathematical relationship — compromising one does not reveal the other.

The Specter's human-readable name is generated by hashing the public point with BLAKE3, taking the first 4 bytes as two 16-bit indices, and looking up two words from a curated 65,536-entry wordlist stored in `assets/wordlists/specter-names.txt`. This produces names like "Hollow Tide" or "Silent Fracture." Collisions are cosmetically possible but irrelevant; the public point, not the name, is the canonical identifier.

### 4.3 Visual Sigils

A sigil is a deterministic visual icon generated from an identity's public key. The generation process takes the BLAKE3 hash of the public key (32 bytes) and interprets it as parameters for a procedural drawing routine: the first 3 bytes select a base hue (mapped to HSL), the next 2 bytes select a geometric pattern from a set of 12 base templates (concentric rings, radial spokes, tessellated polygons, spirograph curves, etc.), and the remaining 27 bytes control pattern parameters (symmetry order, stroke count, fill density, rotation offset, inner/outer radius ratios). The sigil is rendered into a 64×64 Ebitengine image at identity creation time and cached. Surface sigils use bright, saturated palettes. Specter sigils use a restricted palette of deep purples, indigos, and spectral greens.

### 4.4 Proof of Work

Each Wave carries a SHA-256-based Proof of Work. The PoW scheme is Hashcash-style: the sender must find a 64-bit nonce such that `SHA-256(wave_hash || nonce)` has at least `D` leading zero bits. The difficulty `D` is a network-wide constant initially set to 20 (targeting 2–5 seconds on a single core of mid-range 2024 consumer hardware). The PoW is computed by iterating nonces sequentially on a dedicated goroutine. If the goroutine does not find a solution within 30 seconds, the difficulty target is locally relaxed by 1 bit and the attempt restarts (this prevents locking on pathological inputs). PoW verification is a single SHA-256 evaluation and is essentially free.

Difficulty adjustment is not automatic in v1. It is a protocol-level constant that changes only with protocol version bumps. Future versions may implement adaptive difficulty based on observed network throughput.

### 4.5 Shroud Onion Construction

A Shroud circuit consists of three hops. The initiator selects three relay nodes from the pool of peers advertising on `/murmur/shroud/1`, excluding any peers in common with the initiator's direct mesh (to reduce correlation). Circuit construction proceeds as follows.

The initiator opens a `/murmur/shroud-circuit/1` stream to Hop 1 and performs a Curve25519 ephemeral key exchange, deriving a shared secret `S1`. Using `S1`, the initiator derives an encryption key `K1` and a MAC key `M1` via HKDF-SHA256. The initiator then sends a `RelayExtend` cell encrypted under `K1`, instructing Hop 1 to open a stream to Hop 2. Hop 1 does so and relays the inner handshake. The initiator performs a second Curve25519 exchange with Hop 2 (the handshake bytes tunneled through Hop 1), deriving `S2`, `K2`, `M2`. The process repeats for Hop 3.

To send a message through the circuit, the initiator encrypts the payload under `K3`, then encrypts the result under `K2`, then under `K1`. Each hop peels one layer and forwards the remainder. The exit hop (Hop 3) sees the plaintext payload and publishes it on the destination GossipSub topic as if it were the originator, but with `sender_pubkey` zeroed and `signature` replaced by a zero-knowledge proof of Specter identity (a Schnorr signature under the Specter's Curve25519 key, adapted to the Ed25519 verification equation via Ristretto point conversion using `github.com/bwesterb/go-ristretto`).

Circuit lifetime is 10 minutes. Circuits are pre-emptively rotated 60 seconds before expiry. A node maintains at most 3 active circuits simultaneously.

---

## 5. Pulse Map Engine

### 5.1 Layout Algorithm

The Pulse Map uses a modified Fruchterman-Reingold force-directed layout algorithm running in a dedicated goroutine. On every simulation tick (targeted at 60 ticks per second, decoupled from the rendering frame rate), the engine computes three forces for each node.

Repulsive force: every pair of nodes exerts a repulsive force inversely proportional to the square of their distance, with a constant `k_repel` tuned to produce comfortable spacing at the expected node count. For networks exceeding 500 visible nodes, a Barnes-Hut quadtree approximation reduces the pairwise computation from O(n²) to O(n log n). The quadtree is rebuilt every tick from scratch — profiling in Go shows this is cheaper than incremental maintenance due to allocation patterns.

Attractive force: every edge exerts a spring force proportional to the distance between its endpoints minus a rest length, with a constant `k_attract` tuned to keep connected nodes within a few node-diameters of each other. Edge rest length is modulated by interaction frequency — frequently interacting pairs have shorter rest lengths and cluster tighter.

Centering force: a weak global force pulls all nodes toward the viewport center, preventing the graph from drifting off-screen. This force is proportional to the distance from center and is weak enough to be invisible but strong enough to prevent runaway drift over minutes.

After computing net force on each node, the engine applies velocity-Verlet integration with a damping factor of 0.85 per tick. Node positions are written to a double-buffered slice: the layout goroutine writes to the back buffer and atomically swaps it with the front buffer, which the Ebitengine `Draw()` call reads. This avoids any lock contention between simulation and rendering.

### 5.2 Rendering Pipeline

Each Ebitengine `Draw()` call proceeds in four layers, composited back to front.

Layer 0 (Background): a solid dark color (#0A0A0F) fills the screen. A subtle grid of dim dots at regular intervals provides a sense of scale during zoom.

Layer 1 (Edges): edges are drawn as anti-aliased lines using `vector.StrokeLine` with thickness proportional to interaction frequency (1px minimum, 4px maximum). Edge color fades from the source node's hue to the destination node's hue. Edges connecting to anonymous Specter nodes use a desaturated purple. Edge opacity is modulated by the camera's zoom level — at extreme zoom-out, low-frequency edges become transparent to reduce visual noise.

Layer 2 (Nodes): each node is drawn as a circle at a size determined by `log2(connectionCount + 1) * baseRadius`. The node's pre-rendered sigil image is drawn centered on the node position, scaled to fit. An outer ring indicates activity recency: bright and fully saturated for activity within the last 5 minutes, fading through progressively dimmer shades to nearly transparent for nodes inactive more than 6 hours. Active Specter nodes have their glow rendered via the `glow.kage` shader, producing a soft spectral bloom in purple-indigo.

Layer 3 (Effects): Wave ripples are drawn as expanding concentric circles originating from the publishing node, with radius increasing at 200 pixels per second and opacity decreasing linearly over 2 seconds. The ripple color matches the Wave's author hue. Shroud circuit activity is visualized as faint particle trails along circuit edges (visible only to the local node for its own circuits, as a UX affordance — no information about other nodes' circuits is available).

Layer 4 (UI): the immediate-mode UI is drawn on top of the Pulse Map. Panels for Wave composition, thread viewing, identity management, and onboarding overlays are rendered here. UI elements consume input events before they reach the Pulse Map's camera controls.

### 5.3 Camera

The camera is a 2D affine transform (translation + uniform scale) applied to all Pulse Map drawing. Zoom ranges from 0.1x (extreme zoom-out showing the entire network as a galaxy of dots) to 5.0x (extreme zoom-in showing individual sigil detail). Pan is controlled by click-and-drag. Zoom is controlled by scroll wheel, with the zoom center anchored to the cursor position. All camera transitions use exponential smoothing (lerp factor 0.12 per frame) to prevent jarring jumps. Double-clicking a node initiates a smooth focus transition: the camera pans and zooms to center the selected node at 2.0x magnification over 400 milliseconds.

---

## 6. Proof of Work Implementation Detail

The PoW goroutine is launched when the user submits a new Wave. The UI displays a progress indicator showing elapsed time and estimated remaining time (based on the expected number of iterations for the current difficulty). The goroutine runs a tight loop:

    func findNonce(ctx context.Context, waveHash []byte, difficulty uint8) (uint64, error) {
        var buf [40]byte // 32-byte hash + 8-byte nonce
        copy(buf[:32], waveHash)
        for nonce := uint64(0); ; nonce++ {
            binary.BigEndian.PutUint64(buf[32:], nonce)
            hash := sha256.Sum256(buf[:])
            if leadingZeros(hash[:]) >= difficulty {
                return nonce, nil
            }
            if nonce%1_000_000 == 0 {
                select {
                case <-ctx.Done():
                    return 0, ctx.Err()
                default:
                    runtime.Gosched()
                }
            }
        }
    }

The `leadingZeros` function counts leading zero bits using `math/bits.LeadingZeros64` on successive 8-byte chunks of the hash. At difficulty 20, the expected number of iterations is approximately 1,048,576 (2²⁰), which completes in 2–5 seconds on a single core clocked at 3 GHz, assuming roughly 200 ns per SHA-256 evaluation in Go's `crypto/sha256` (which uses hardware acceleration on amd64 and arm64).

The PoW goroutine is cancellable via context — if the user discards the Wave before PoW completes, the goroutine exits immediately.

---

## 7. Resonance Computation

Resonance is computed locally by each node for each Specter it has observed. It is not a global consensus value; it is a local trust heuristic. The computation uses four input signals.

Wave consistency measures how regularly a Specter publishes. A Specter that publishes at least one Wave per day accumulates 1 consistency point per day, up to a maximum of 30. A gap of more than 3 days without a Wave reduces the consistency score by 2 points per day. This rewards regular participation without punishing occasional absence.

Reply engagement measures how often a Specter's Waves generate replies from distinct identities. Each unique replier to a Specter's Wave within a 24-hour window contributes 0.5 points, capped at 5 points per day. This rewards content that generates conversation.

Cross-layer bridging awards 2 points per day to Specters observed participating in threads that also involve Surface identities. This rewards Specters who engage with the non-anonymous community rather than retreating into an exclusively anonymous silo.

Longevity awards 0.1 points per day of continuous Specter existence (measured from first observation), uncapped. This provides a slow baseline accumulation that rewards persistence.

The final Resonance score is the sum of these four components, recomputed hourly. The score decays at 1% per day (multiplicative) to ensure that inactive Specters gradually lose rank. Rank thresholds (Shade at 25, Wraith at 50, Shade-Wraith at 75, Phantom at 100, Council-Eligible at 200) are checked against the locally computed Resonance and may differ between observers — a Specter might be a Wraith in one node's view and still a Shade in another's. This is by design: local computation prevents Resonance from becoming a gameable global leaderboard.

---

## 8. Concurrency Model

The application runs approximately 8 persistent goroutines plus transient goroutines for PoW computation and Shroud circuit construction.

The main goroutine runs the Ebitengine game loop (`ebiten.RunGame`), calling `Update()` and `Draw()` at the display's refresh rate (typically 60Hz). `Update()` drains the UI event channel, processes input, and updates UI state. `Draw()` reads the Pulse Map's front buffer and renders all four layers.

The network goroutine runs the libp2p swarm event loop, dispatching incoming messages to the event bus channel. A separate goroutine runs the DHT bootstrap and periodic peer refresh.

The layout goroutine runs the force-directed simulation, writing to the Pulse Map's back buffer and swapping every tick.

The expiry goroutine wakes every 60 seconds to garbage-collect expired Waves from Bbolt.

The heartbeat goroutine publishes a signed heartbeat to `/murmur/pulse/1` every 30 seconds.

The Shroud maintenance goroutine manages circuit lifecycle: constructing new circuits, rotating expiring circuits, and tearing down dead circuits.

The event bus goroutine receives all events (network, timer, user action) and fans them out to subscriber channels. Each subsystem registers its interest at startup and receives only relevant events.

Communication between goroutines uses exclusively typed channels. No shared mutable state is accessed without channel synchronization or atomic operations. The double-buffered Pulse Map node positions are the only data structure accessed from multiple goroutines simultaneously, and they use `atomic.Pointer` swaps to avoid locks.

---

## 9. Performance Targets

The following targets apply to a mid-range 2024 desktop (4-core CPU, integrated GPU, 16 GiB RAM) with a network of 1,000 peers and 200 cached Waves.

Rendering must sustain 60fps with up to 500 visible nodes and 2,000 visible edges. At higher node counts, the renderer culls off-screen nodes and edges, and the layout engine switches to Barnes-Hut, maintaining 60fps up to approximately 5,000 total nodes.

Wave propagation latency (time from publication on one node to receipt on a peer 3 hops away in the gossip mesh) must be below 500 milliseconds under normal network conditions.

PoW computation must complete within 2–5 seconds for the default difficulty.

Shroud circuit construction must complete within 3 seconds (three sequential round-trip key exchanges across the mesh).

Application cold start (launch to Pulse Map visible with initial peers connected) must be below 5 seconds. Subsequent starts with cached peer state must be below 2 seconds.

Memory usage must remain below 256 MiB during normal operation with 1,000 cached peer records and 200 cached Waves.

Bbolt database size must remain below 50 MiB under the same conditions. Wave expiration garbage collection must complete within 100 milliseconds per sweep.

---

## 10. Build and Distribution

The application is built with `go build` producing a single static binary per platform. Target platforms are linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64. Ebitengine's cross-compilation support (via Oto for audio and Purego for system calls) allows building all targets from a single Linux CI machine.

The build embeds the Specter name wordlist, default configuration, and onboarding assets via `go:embed`. No runtime file dependencies exist beyond the Bbolt database file, which is created on first launch in the user's platform-appropriate data directory (`$XDG_DATA_HOME/murmur/` on Linux, `~/Library/Application Support/murmur/` on macOS, `%APPDATA%\murmur\` on Windows).

Protobuf code generation is a build-time dependency only. Generated `.pb.go` files are checked into the repository so that contributors can build without installing `protoc`.

Releases are tagged with semantic versioning. The protocol version (carried in every `MurmurEnvelope`) is incremented independently of the release version and only changes when wire-format compatibility breaks.

---

## 11. Testing Strategy

Unit tests cover cryptographic operations (signing round-trips, PoW verification, onion encryption/decryption), data structure operations (graph manipulation, Resonance computation, TTL enforcement), and protobuf serialization round-trips. These tests use the standard `testing` package and require no external dependencies.

Integration tests cover subsystem interactions: publishing a Wave through the content pipeline and verifying it arrives in the store, constructing a Shroud circuit across three in-process libp2p hosts, and computing Resonance from a sequence of simulated observations. Integration tests use in-process libp2p hosts with memory transports to avoid binding real network ports.

Simulation tests construct networks of 10–100 in-process nodes and verify emergent properties: gossip propagation reaches all nodes within bounded time, Shroud circuits provide source anonymity against a passive observer controlling one hop, and the force-directed layout converges to a stable configuration within a bounded number of ticks. Simulation tests are gated behind a `//go:build simulation` tag because they are slow (30+ seconds).

No tests depend on Ebitengine. The rendering layer is tested manually and via screenshot comparison in CI using Ebitengine's headless mode.
