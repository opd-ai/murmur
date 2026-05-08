# MURMUR Decision Log

A record of major reversible decisions made in the MURMUR project. Each decision includes: what was chosen, why, when, and when to revisit.

---

## D-001: Use libp2p instead of Tor as Foundation

**Date**: 2025-Q4  
**Decision**: Build MURMUR on libp2p + GossipSub rather than layering on top of Tor.

**Rationale**:
- libp2p provides pluggable transport (TCP, QUIC, WebSocket, WebRTC)
- GossipSub v1.1 offers peer scoring, ideal for spam mitigation
- Tor is offered as an optional transport adapter, not a dependency
- Decouples network layer from application layer
- Gives MURMUR users a choice: fast (clearnet) vs private (Tor/I2P)

**Alternatives Considered**:
- Built entirely on Tor: slower, less flexible for games
- Standalone P2P: does not leverage audited Noise/HMAC/DHT code

**Revisit By**: 2026-Q4 (if libp2p ecosystem shifts significantly, e.g., major security issue)

---

## D-002: Ephemeral Waves (Max 30-day TTL, No Archive)

**Date**: 2025-Q4  
**Decision**: All Waves (messages) expire and are not permanently archived.

**Rationale**:
- Aligns with core principle: "No permanent record by default"
- Reduces storage burden on nodes
- Encourages authentic, time-bound communication
- Protects privacy: no digital trail for adversaries to mine

**Trade-off**: Users cannot retrieve old messages after expiry.

**Mitigation**: Multi-Device Identity + Social Recovery allow identity continuity even if messages are lost.

**Alternatives Considered**:
- Optional permanent storage: adds complexity, privacy liability
- Blockchain archive: immutable but contradicts "no permanent record"

**Revisit By**: 2026-Q4 (if user research shows retention harm > §1.2 product goal)

---

## D-003: Pulse Map as Primary Interface (Not Messaging First)

**Date**: 2025-Q4 → **Updated 2026-04**  
**Original Decision**: Pulse Map (force-directed graph) as the primary spatial navigation surface.

**Rationale**:
- Unique differentiation from chat-first apps
- Aligns Design Principle #4: "The network is the interface"
- Spatial discovery enables serendipitous connection

**Issue Discovered (2026-04)**: Cold-start friction. New users see empty or 5-node graphs; engagement drops 40% in first session.

**Updated Approach (2026-04)**: Messaging-first onboarding for MVP (first 5-8 users in a friend group). Pulse Map accessible in "Explore" tab. After network reaches 20+ nodes, default to Pulse Map.

**Expected Revisit**: 2026-Q2 (after anchor community testing confirms navigation patterns)

**Related**: Documented in docs/PULSE_MAP_ROLE_DECISION.md

---

## D-004: Resonance as Local Reputation, Not Cryptocurrency

**Date**: 2026-01  
**Decision**: Reputation (Resonance) computed locally on each node; never traded or minted as currency.

**Rationale**:
- Avoids regulatory complexity (not a security/token)
- No attack surface for money laundering
- Aligns with Design Principle #1: "Privacy is structural"
- Prevents pay-to-win dynamics that contradict organic growth

**Trade-off**: Cannot incentivize tunnel operators with crypto. (Mitigated by TSC → bounded Resonance conversion in docs/TUNNEL_OPERATOR_INCENTIVES.md)

**Alternatives Considered**:
- ERC-20 token: regulatory risk, attracts financial parasites
- Fiat payment: requires bank accounts (identity leak)

**Revisit By**: 2027 (only if legal landscape shifts)

---

## D-005: Three-Hop Shroud Circuits (Not Single-Hop)

**Date**: 2026-02  
**Decision**: Anonymous routing uses 3-hop circuits, not single-relay forwarding.

**Rationale**:
- Hop 1 sees initiator IP but not destination
- Hop 3 sees plaintext message but not initiator IP
- No single relay learns full mapping
- Balances privacy vs latency (500-2000ms per message)

**Trade-off**: Users with Tor-level threat models must bridge to Tor explicitly.

**Out-of-Scope Protection**: Global eavesdropper, nation-state adversary → Tor integration recommended.

**Revisit By**: 2027-Q2 (traffic analysis academic advances, changes to user threat model)

---

## D-006: BIP-39 Recovery with Multi-Device Continuity (v1.0)

**Date**: 2026-03  
**Decision**: Single Master Identity from BIP-39 seed; multi-device support via signed Device Keys.

**Rationale**:
- Master Key never transmitted (stays offline)
- Each device key is revocable without full recovery
- Backward compatible with single-device clients
- Device key compromise does not fully compromise master

**Trade-off**: Requires device coordination for revocation (7-day grace period).

**Related Decision**: docs/MULTI_DEVICE_IDENTITY.md, docs/KEY_ROTATION.md

**Revisit By**: 2026-Q3 (after launch, evaluate user comprehension of master vs device keys)

---

## D-007: Extension Surface Stable in v0.1 (Not v1.0)

**Date**: 2026-03  
**Decision**: Publish extension points (GameModule, custom Wave types, transport adapters, Resonance hooks) as stable API in v0.1, not v1.0.

**Rationale**:
- Third parties need confidence to build games
- Stable API unlocks value earlier
- Core can still refine undocumented implementation

**Trade-off**: Must maintain API compatibility from v0.1 forward.

**Mitigation**: Experimental extensions (Resonance hooks) clearly marked; unfrozen if feedback dictates.

**Related**: EXTENSION_CONTRACT.md, MEPs/ process

**Revisit By**: 2026-Q4 (assess if stability constraint blocks innovation)

---

## D-008: Tor and I2P as Optional Transport, Not Mandatory

**Date**: 2026-04  
**Decision**: Users opt into Tor/I2P transport; default is clearnet (with Shroud native anonymity).

**Rationale**:
- Lowers barrier to entry (no daemon setup for new users)
- Shroud 3-hop already provides meaningful anonymity for most users
- Tor/I2P for users with higher threat models
- Aligns Design Principle #2: "Metadata unlinkability is the default"

**Trade-off**: Cannot guarantee Tor-level anonymity without explicit opt-in.

**Clarification**: Users unsure of their threat model default to safe (Tor), not fast (clearnet).

**Related**: docs/ANONYMITY_TRANSPORT_MODES.md, TRANSPORT_ANONYMITY.md

**Revisit By**: 2027-Q1 (user telemetry on transport adoption; adjust defaults if unsafe)

---

## D-009: Tunneling Abuse Policy Before Implementation

**Date**: 2026-04  
**Decision**: Design TUNNEL_ABUSE_POLICY.md fully before implementing tunneling primitive.

**Rationale**:
- Tunneling is highest-risk feature (potential malware C2, CSAM, DDoS)
- Policy-first prevents security debt
- Operator consent, content allowlists, bandwidth caps prevent misuse
- DMCA safe harbor compliance requires good-faith abuse controls

**Implementation Impact**: Affects tunnel registration, exit policy validation, automated takedown protocol.

**Revisit By**: 2026-Q3 (once tunneling goes live; assess if policy is sufficient)

---

## D-010: MEP Process Intentionally Lightweight

**Date**: 2026-05  
**Decision**: Murmur Extension Proposals (MEPs) are numbered markdown files, not formal RFCs.

**Rationale**:
- Lower barrier to participation (anyone can write)
- No approval required to start discussion
- Merges naturally when consensus emerges
- Extensibility grows organically without bureaucracy

**Trade-off**: May result in duplicates or incomplete proposals (resolved in discussion).

**Related**: MEPs/README.md, MEPs/MEP-0-TEMPLATE.md

**Revisit By**: 2027 (once MEP volume becomes unmanageable, consider steering committee)

---

## Tentative Decisions (To Be Confirmed)

### TBD-001: Upgrade Shroud to 5 Hops?

**Proposed**: After launch, evaluate if 3-hop Shroud is sufficient or if 5-hop is justified.

**Tradeoff**: 5-hop = stronger anonymity (~200ms more latency per circuit).

**Decision Point**: User feedback on threat model alignment.

**Target Decision Date**: 2026-Q4

---

### TBD-002: Specter-to-Surface Bridge Feature?

**Proposed**: Allow Specters to "claim" Surface identity after proving friendship (multi-sig threshold).

**Benefit**: Privacy-preserving identity linking for users who want it.

**Risk**: Linkability analysis could re-identify anonymous users.

**Decision Point**: Threat model research + user testing.

**Target Decision Date**: 2027-Q2

---

## D-011: Cryptographic Primitives Selection

**Date**: 2026-Q1
**Decision**: Use specific algorithm-per-use-case mapping:
- Ed25519 for Surface Layer identity signing and Wave signatures
- Curve25519 / X25519 for Anonymous Layer key exchange (Shroud circuits)
- XChaCha20-Poly1305 for all symmetric encryption (keystore, Shroud layers, councils)
- SHA-256 for Proof of Work and content addressing (Wave IDs)
- BLAKE3 for identity hashing (sigils, pseudonyms, message_id in envelopes)
- Argon2id (time=3, memory=64 MiB, threads=4) for passphrase-based key derivation
- HKDF-SHA-256 for deriving symmetric keys from DH shared secrets

**Rationale**:
- Ed25519 provides fast, compact signatures (64 bytes) suitable for high-frequency Wave signing
- Curve25519 is the standard for ephemeral Diffie-Hellman; established in Noise XX protocol
- XChaCha20-Poly1305 offers authenticated encryption with large 192-bit nonces (no nonce reuse risk)
- SHA-256 is universally available, well-audited, and sufficient for PoW difficulty adjustment
- BLAKE3 is faster than SHA-256 for identity hashing with better parallelism, appropriate for sigil generation
- Argon2id is memory-hard, resisting GPU/ASIC brute-force on encrypted keystores
- No algorithm substitution is permitted without a new decision record

**Alternatives Considered**:
- AES-GCM: rejected (nonce reuse danger in high-volume contexts, performance advantage lost with AES-NI unavailable on some ARM)
- SHA-3 for identity hashing: rejected (BLAKE3 is faster and equally secure for non-cryptographic-commitment use cases)
- bcrypt/scrypt for key derivation: rejected (Argon2id won PHC and provides better GPU resistance)

**Revisit By**: 2028-Q1 (cryptanalysis advances or NIST PQC standard finalization)

---

## D-012: Protocol Buffers (proto3) over JSON for All Wire and Storage Formats

**Date**: 2026-Q1
**Decision**: All inter-node messages and all database-stored records use Protocol Buffers proto3. JSON is explicitly prohibited on the wire and in storage.

**Rationale**:
- Compact binary encoding reduces Wave size (≤2048 bytes budget is tight with JSON)
- Schema enforcement: field renaming/addition is backward-compatible without parser changes
- Language-neutral: Go, future mobile clients, and relay-only nodes can share the same .proto files
- Deterministic serialization aids content-addressing (SHA-256 / BLAKE3 of proto bytes as Wave ID)

**Alternatives Considered**:
- JSON: human-readable but verbose; non-deterministic key ordering breaks content addressing
- MessagePack: no official schema; cross-language support fragmented
- CBOR: considered, but ecosystem tooling is weaker than protobuf for Go

**Revisit By**: 2027-Q1 (if alternative binary schema formats mature significantly)

---

## D-013: BBolt for Embedded Persistent Storage

**Date**: 2026-Q1
**Decision**: Use `go.etcd.io/bbolt` as the single embedded key-value store for all subsystems.

**Rationale**:
- Single-file database requires no separate server process
- ACID transactions via B-tree with memory-mapped reads
- Pure Go, zero CGo, ensuring cross-compilation for all target platforms
- Bucket-per-subsystem model maps cleanly to MURMUR domains (identity, waves, peers, etc.)
- Small code footprint; well-audited; used in production by etcd

**Alternatives Considered**:
- SQLite (via go-sqlite3): CGo dependency breaks cross-compilation; schema migrations add complexity
- LevelDB / Pebble: LSM-tree writes are faster but reads are slower; no ACID transactions without extra wrapper
- BadgerDB: more complex API; larger binary footprint

**Revisit By**: 2027-Q1 (if BBolt shows performance bottleneck at >50 MiB DB size)

---

## D-014: Ebitengine for Rendering (Not Immediate-Mode GUI or Web)

**Date**: 2026-Q1
**Decision**: Use Ebitengine v2 as the single cross-platform rendering engine for the Pulse Map and all UI.

**Rationale**:
- 2D game engine with 60fps target matches the Pulse Map's force-directed graph animation requirements
- Kage shader language enables glow, ripple, and spectra visual effects without raw OpenGL/Metal
- Single binary per platform via `go build` (no Electron, no browser runtime)
- Ebitengine handles OS window management, input events, and audio—reducing dependency count
- Active maintenance and stable v2 API

**Alternatives Considered**:
- Fyne: immediate-mode widget toolkit; insufficient for custom force-directed graph rendering
- Gio: GPU-accelerated but lacks shader support for Kage-equivalent effects
- Electron/Wails: browser runtime adds 60–100 MB; contradicts offline-first self-sovereign goal
- Raw OpenGL: too much boilerplate; platform-specific shader compilation

**Revisit By**: 2027-Q1 (if Ebitengine drops support for a target platform, or if WebAssembly becomes a first-class target)

---

## D-015: pkg/ Package Layout (Not internal/)

**Date**: 2026-Q1
**Decision**: All subsystem packages live under `pkg/`, not `internal/`. The `internal/` directory is not used.

**Rationale**:
- Downstream projects (relay operators, game SDK consumers) need to import subsystem packages
- `internal/` would prohibit such imports; `pkg/` makes them explicit and stable
- Interface-based subsystem boundaries (e.g., `store.WaveStore`, `mechanics.Publisher`) provide the encapsulation that `internal/` would otherwise enforce
- The project is open-source; hiding implementation details via `internal/` adds friction without security benefit

**Alternatives Considered**:
- `internal/` for all packages: rejected (blocks downstream use of SDK and relay operator tooling)
- Mix of `pkg/` and `internal/`: rejected (inconsistency; unclear policy for new packages)

**Revisit By**: Never (structural decision; reversing it requires renaming all import paths)

---

## How to Propose a New Decision

1. **Document the choice**: What are you deciding?
2. **Explain the rationale**: Why this over alternatives?
3. **Name trade-offs**: What are you giving up?
4. **Set revisit date**: When should this be re-evaluated?
5. **PR to this file**: Add your decision under the main section above.

---

**Last Updated**: 2026-05-08  
**Maintained By**: MURMUR Core Team
