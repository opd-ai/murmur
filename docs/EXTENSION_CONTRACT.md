# MURMUR Extension Contract v0

## Purpose

MURMUR is designed as an **extensible protocol**, not just an application. Third parties should be able to build compatible networks, custom games, and specialized UIs that inherit MURMUR's identity, anonymity, and transport guarantees without forking the core.

This document defines every **extension point** where downstream networks and applications can extend MURMUR functionality, the **stability guarantees** for each extension point, and the **compatibility requirements** for extensions.

---

## Extension Points

### 1. Custom Wave Types

**Status:** 🟢 **STABLE** — Public API, backward compatibility guaranteed

#### Description
Extend the Wave message format to support domain-specific content types (e.g., polls, collaborative documents, encrypted file metadata).

#### Extension Mechanism
- Register custom Wave type IDs in the range `0x40–0xFF` (types `0x01–0x3F` reserved for core MURMUR)
- Implement custom serialization/deserialization via Protocol Buffers `Any` type or custom protobuf messages
- Register an `ExtensionHandler` in `pkg/content/waves/extensions.go`; registered handlers are invoked during Wave validation for extension types
- Custom types propagate through GossipSub like standard Waves; subject to PoW and TTL enforcement

#### API Surface
```go
// pkg/content/waves/extensions.go
type WaveType uint8

const (
    // Core types (0x01–0x3F)
    TypeSurfaceWave WaveType = 0x01
    TypeReply       WaveType = 0x02
    // ... (8 core types)
    
    // Extension range (0x40–0xFF)
    TypeExtensionStart WaveType = 0x40
    TypeExtensionEnd   WaveType = 0xFF
)

// RegisterWaveType registers a custom Wave type handler.
func RegisterWaveType(typ WaveType, handler ExtensionHandler) error

type ExtensionHandler interface {
    Validate(wave *pb.Wave) error
    Render(wave *pb.Wave) ([]byte, error)
}
```

#### Examples
- **TypePoll (0x40):** Poll Wave with multiple choice options and vote aggregation
- **TypeCollaborativeDoc (0x41):** CRDTs for real-time collaborative editing
- **TypeFileMetadata (0x42):** Encrypted file chunk metadata for P2P file sharing

#### Compatibility Requirements
- Custom Wave types MUST include PoW (SHA-256, difficulty ≥20)
- Custom types MUST respect TTL enforcement (max 30 days)
- Custom types MUST be ≤2048 bytes after protobuf serialization
- Custom types MUST be signed with Ed25519 (Surface) or omit signature (Specter)

---

### 2. Custom Game Modules

**Status:** 🟢 **STABLE** — Public API, backward compatibility guaranteed

#### Description
Extend the mini-game library with new asynchronous or real-time games. Games are sandboxed and interact with the network only through the Game Module SDK.

#### Extension Mechanism
- Implement `GameModule` in `pkg/anonymous/mechanics/sdk.go`
- Register the module at startup via `RegisterGameModule(module)`
- SDK provides primitives: create match, broadcast event, persist state, end match, award Resonance

#### API Surface
```go
// pkg/anonymous/mechanics/sdk.go
type GameModule interface {
    Metadata() GameMetadata
    CreateMatch(ctx context.Context, config MatchConfig) (Match, error)
    ValidateConfig(config MatchConfig) error
}

type Match interface {
    ID() [32]byte
    Join(ctx context.Context, participantKey [32]byte) error
    Leave(ctx context.Context, participantKey [32]byte) error
    HandleEvent(ctx context.Context, event Event) error
    State() MatchState
    End(ctx context.Context, outcome Outcome) error
}
```

#### Examples
- **Specter Hunt (built-in):** Hide-and-seek game where Specters leave clues on Pulse Map
- **Cipher Puzzle (built-in):** Collaborative puzzle-solving with cryptographic challenges
- **Custom: Territory Drift (third-party):** Async strategy game where Specters claim Pulse Map regions

#### Compatibility Requirements
- Games MUST be sandboxed: no direct network, identity, or storage access
- Games MUST tolerate player dropout (no forced penalties for leaving mid-match)
- Games MUST NOT leak identity metadata (e.g., reaction times that fingerprint players)
- Games MUST respect Resonance milestone gating (e.g., real-time games require Phantom=100)

---

### 3. Custom Resonance Hooks

**Status:** 🟢 **STABLE** — Public API, backward compatibility guaranteed

#### Description
Extend Resonance computation with custom signal sources (e.g., domain-specific reputation from third-party oracles).

#### Extension Mechanism
- Implement `ResonanceHook` interface in `pkg/anonymous/resonance/hooks.go`
- Register hook at startup; invoked by the owning scorer/integration layer
- Hooks receive **read-only** Resonance score views via `ReadOnlyQuery`

#### API Surface
```go
// pkg/anonymous/resonance/hooks.go
type ReadOnlyQuery interface {
    SpecterScore(specterID string) (ReadOnlyScore, bool)
}

type ResonanceHook interface {
    Name() string
    ComputeSignal(ctx context.Context, specterID string, query ReadOnlyQuery) (float64, error)
}

// RegisterResonanceHook adds a custom signal provider.
func RegisterResonanceHook(hook ResonanceHook) error

// NewReadOnlyQuery adapts a scorer to the hook query surface.
func NewReadOnlyQuery(scorer interface{ LookupScore(string) (*Score, bool) }) ReadOnlyQuery
```

#### Examples
- **ThirdPartyOracleHook:** Query external reputation system (e.g., Gitcoin Passport score)
- **GameWinRateHook:** Boost Resonance for players with high win rates in specific games
- **LongevityHook:** Reward identities that have been active for >30 days

#### Compatibility Requirements
- Hooks MUST NOT block for >1 second (computation is async, but timeout enforced)
- Hooks MUST NOT modify identity state (read-only access)
- Hooks MUST NOT call external network services without user consent (privacy leak risk)
- Hooks MUST return normalized signals in range [0.0, 1.0]

---

### 4. Custom Transport Adapters

**Status:** 🟢 **STABLE** — Public API, backward compatibility guaranteed

#### Description
Extend MURMUR's transport layer with additional libp2p transports (e.g., Bluetooth, LoRa, satellite).

#### Extension Mechanism
- Implement libp2p `transport.Transport` interface
- Register the adapter via `pkg/networking/transport.RegisterAdapter`
- Adapter must support Noise encryption and yamux multiplexing

#### API Surface
```go
// pkg/networking/transport/extensions.go
type AdapterConstructor func(
    ctx context.Context,
    upgrader transport.Upgrader,
    rcmgr network.ResourceManager,
) (transport.Transport, error)

func RegisterAdapter(name string, constructor AdapterConstructor) error
```

#### Examples
- **Tor Onion Transport (planned):** Route MURMUR traffic through Tor hidden services
- **I2P Garlic Transport (planned):** Route MURMUR traffic through I2P destinations
- **Bluetooth Transport (third-party):** Mesh networking over Bluetooth Low Energy
- **LoRa Transport (third-party):** Long-range radio for off-grid communication

#### Compatibility Requirements
- Custom transports MUST support Noise XX or compatible encryption
- Custom transports MUST support yamux or mplex stream multiplexing
- Custom transports MUST provide `Listen()` and `Dial()` methods per libp2p interface
- Custom transports SHOULD handle NAT traversal if applicable (or document limitations)

---

### 5. Custom UI Overlays

**Status:** 🟡 **EXPERIMENTAL** — API may change in minor versions

#### Description
Extend the Pulse Map with custom visual overlays (e.g., heatmaps, domain-specific node decorations).

#### Extension Mechanism
- Implement `PulseMapOverlay` interface in `pkg/pulsemap/overlays/`
- Register overlay at startup; rendered on top of base Pulse Map
- Overlays receive node positions and render custom graphics via Ebitengine

#### API Surface
```go
// pkg/pulsemap/overlays/api.go
type PulseMapOverlay interface {
    Name() string
    Render(screen *ebiten.Image, positions map[string]Position, camera Camera) error
}

// RegisterOverlay adds a custom overlay to the Pulse Map rendering stack.
func RegisterOverlay(overlay PulseMapOverlay) error
```

#### Examples
- **AnonymousActivityHeatmap (built-in):** Visualize Specter activity density
- **GameZoneOverlay (third-party):** Highlight active game matches on Pulse Map
- **TerritoryOverlay (third-party):** Show claimed regions for Territory Drift game

#### Compatibility Requirements
- Overlays MUST render in <5ms to maintain 60fps
- Overlays MUST NOT block the main rendering thread
- Overlays MUST respect user privacy (no telemetry, no external network calls)
- Overlays SHOULD support toggle on/off via UI

---

### 6. Custom Identity Providers

**Status:** 🔴 **PRIVATE** — Not yet exposed as extension point

#### Description
(Future) Allow third-party identity systems to interoperate with MURMUR (e.g., DID, Ethereum Name Service).

#### Rationale for PRIVATE Status
- Identity is foundational to MURMUR's security model; premature extension creates risk
- Current implementation assumes Ed25519 keypairs; multi-provider support requires significant refactoring
- Planned for v1.1+ after core stabilization

#### Planned API (Tentative)
```go
// pkg/identity/providers/api.go (NOT YET IMPLEMENTED)
type IdentityProvider interface {
    Name() string
    CreateIdentity() (Identity, error)
    SignWave(wave *Wave, identity Identity) error
    VerifySignature(wave *Wave) (Identity, error)
}
```

---

### 7. Custom Storage Backends

**Status:** 🔴 **PRIVATE** — Not yet exposed as extension point

#### Description
(Future) Replace Bbolt with alternative storage backends (e.g., PostgreSQL, IPFS).

#### Rationale for PRIVATE Status
- Bbolt is tightly integrated with MURMUR's ACID guarantees and single-file database model
- Swapping backends requires careful consideration of consistency, performance, and failure modes
- Planned for v1.2+ after operational experience with Bbolt

#### Planned API (Tentative)
```go
// pkg/store/backend.go (NOT YET IMPLEMENTED)
type StorageBackend interface {
    Open(path string) error
    Close() error
    Get(bucket, key []byte) ([]byte, error)
    Put(bucket, key, value []byte) error
    Delete(bucket, key []byte) error
    Transaction(fn func(tx Transaction) error) error
}
```

---

## Extension Stability Guarantees

### 🟢 STABLE
- **API frozen**: No breaking changes without major version bump (v1.x → v2.x)
- **Backward compatibility**: Extensions built for v1.0 work on v1.x
- **Deprecation policy**: 12 months notice before removal
- **Examples:** Custom Wave Types, Custom Game Modules, Custom Resonance Hooks, Custom Transport Adapters

### 🟡 EXPERIMENTAL
- **API may change**: Breaking changes allowed in minor versions (v1.0 → v1.1)
- **Notice period**: 3 months before breaking change
- **Examples:** Custom UI Overlays

### 🔴 PRIVATE
- **Not an extension point**: Internal implementation detail
- **No compatibility guarantees**: Subject to change without notice
- **Examples:** Identity Providers (future), Storage Backends (future)

---

## Protocol Compatibility

### Wire Protocol Version: 1
All extensions MUST use Protocol Buffers proto3 with `MurmurEnvelope` wrapper:

```protobuf
message MurmurEnvelope {
  uint32 version = 1;           // Protocol version (currently 1)
  MessageType type = 2;         // Enum: WAVE, IDENTITY, SHROUD_AD, HEARTBEAT, EXTENSION
  bytes payload = 3;            // Serialized inner message (type-specific protobuf)
  bytes sender_pubkey = 4;      // 32-byte Ed25519 public key (zeroed for anonymous)
  bytes signature = 5;          // Ed25519 signature over (version || type || payload)
  int64 timestamp_unix = 6;     // Sender's Unix timestamp in seconds
  bytes message_id = 7;         // 32-byte BLAKE3 hash of payload
}
```

### Extension Message Type
Custom extensions use `MessageType.EXTENSION` (0x0A) and encode their own inner protobuf in `payload`.

### Version Negotiation
- MURMUR nodes reject envelopes with `version > 1` (forward incompatibility protection)
- Extensions MUST document minimum protocol version required

---

## Building a Compatible Client

To implement a MURMUR-compatible client from scratch:

1. **Identity Layer:** Generate Ed25519 keypair for Surface identity; optionally generate Curve25519 keypair for Specter
2. **Transport Layer:** Construct libp2p host with Noise + yamux; connect to bootstrap peers
3. **GossipSub:** Subscribe to `/murmur/waves/1` and `/murmur/identity/1` topics
4. **Wave Creation:** Generate Wave protobuf, compute SHA-256 PoW (difficulty 20), sign with Ed25519, wrap in `MurmurEnvelope`, publish to `/murmur/waves/1`
5. **Storage:** Use Bbolt (or custom backend) to cache Waves, enforce TTL, maintain thread indexes
6. **Pulse Map (optional):** Implement force-directed layout (Fruchterman-Reingold + Barnes-Hut); render with any 2D graphics library
7. **Shroud (optional):** Implement 3-hop onion routing with Curve25519 DH and XChaCha20-Poly1305 encryption

**Reference implementation:** See MURMUR Go codebase (`github.com/opd-ai/murmur`)  
**Protocol specification:** (TBD — to be extracted into standalone PROTOCOL.md)

---

## MEP (MURMUR Extension Proposal) Process

For proposing new extension points or changes to existing ones:

1. **Draft MEP:** Write a markdown document describing the extension, use cases, API surface, and compatibility impact
2. **Submit PR:** Open a pull request to `docs/meps/` directory with naming format `MEP-NNNN-title.md`
3. **Community Review:** Solicit feedback from maintainers and third-party developers (minimum 2 weeks)
4. **Decision:** Maintainers accept, defer, or reject based on:
   - Alignment with MURMUR's design principles
   - Security and privacy impact
   - Maintenance burden
   - Demand from third-party developers
5. **Implementation:** If accepted, implement in a feature branch; mark as 🟡 EXPERIMENTAL until stabilized

**Example MEPs:**
- MEP-0001: Custom Wave Type Registry (implemented)
- MEP-0002: Game Module SDK Sandboxing (implemented)
- MEP-0003: Tor/I2P Transport Adapters (planned)

---

## Compatibility Testing

Extensions MUST pass the **MURMUR Compatibility Test Suite** (TBD) to be listed in the official extension registry.

**Test categories:**
1. **Protocol conformance:** Messages validate against `MurmurEnvelope` schema
2. **PoW verification:** Custom Wave types pass SHA-256 PoW verification at difficulty 20
3. **TTL enforcement:** Extensions respect Wave expiration (max 30 days)
4. **Sandboxing:** Games and overlays do not access network/storage directly
5. **Performance:** Extensions do not degrade Pulse Map rendering below 60fps @ 500 nodes

---

## Extension Registry (Planned)

**Official registry URL:** (TBD — to be hosted after v1.0 release)

Third-party extensions can self-register by submitting:
- Extension name and repository URL
- Extension points used (Wave Types, Game Modules, etc.)
- Stability tier (STABLE, EXPERIMENTAL)
- License (must be OSI-approved)
- Compatibility test results

**No curation or endorsement:** Listing does not imply security audit or maintainer endorsement.

---

*Last updated: 2026-05-06*  
*Version: 0 (pre-release)*
