# Implementation Gaps — 2026-04-13

This document identifies gaps between MURMUR's stated goals (as documented in README.md, DESIGN_DOCUMENT.md, TECHNICAL_IMPLEMENTATION.md, and related specification files) and the current implementation state.

---

## Gap 1: Complete Codebase Missing

- **Stated Goal**: A decentralized, peer-to-peer social network implemented in Go 1.22+ with libp2p, Ebitengine, and supporting libraries.
- **Current State**: Zero Go source files exist. No `go.mod`, no packages, no tests. The repository contains only ~552KB of markdown specification documents.
- **Impact**: The project cannot be built, run, or used. All technical claims are aspirational rather than functional. Users cannot experience any MURMUR features.
- **Closing the Gap**: Execute the implementation plan in PLAN.md and ROADMAP.md:
  1. Create `go.mod` with module path `github.com/opd-ai/murmur` and Go 1.22+
  2. Add dependencies: go-libp2p v0.48+, golang.org/x/crypto, github.com/zeebo/blake3, go.etcd.io/bbolt, google.golang.org/protobuf, github.com/hajimehoshi/ebiten/v2
  3. Create `pkg/` directory structure per ROADMAP.md:143-178
  4. Implement packages according to ROADMAP.md priorities 1-12
  5. **Estimated effort**: 6-9 months (2-4 engineers) per PLAN.md

---

## Gap 2: Pulse Map Visualization Missing

- **Stated Goal**: A real-time force-directed graph visualization (the Pulse Map) as the primary UI, replacing traditional feeds with spatial navigation. Users navigate a 2D map where nodes represent identities and edges represent connections. Per PULSE_MAP.md, the layout uses Fruchterman-Reingold algorithm with Barnes-Hut optimization for O(n log n) complexity.
- **Current State**: No visualization code exists. No Ebitengine integration, no force-directed layout engine, no rendering pipeline.
- **Impact**: Users have no interface to interact with the network. The core differentiating feature (spatial topology navigation instead of algorithmic feeds) is entirely unavailable.
- **Closing the Gap**:
  1. Create `pkg/pulsemap/layout/` with force-directed graph engine (Fruchterman-Reingold, Barnes-Hut for >500 nodes)
  2. Create `pkg/pulsemap/rendering/` with Ebitengine Draw() implementation, camera transforms
  3. Create `pkg/pulsemap/interaction/` with pan, zoom, node selection, navigation
  4. Create `pkg/pulsemap/overlays/` for Anonymous Layer overlay, activity heatmap
  5. Implement Kage shaders in `pulsemap/shaders/`: glow.kage, ripple.kage, spectra.kage
  6. Target: 60fps with 500 nodes and 2,000 edges
  7. **Estimated effort**: 4-6 weeks per PLAN.md post-v0.7

---

## Gap 3: Cryptographic Identity System Missing

- **Stated Goal**: Self-sovereign Ed25519 identity with no registration server. Per TECHNICAL_IMPLEMENTATION.md §1.4: Ed25519 for signing, Curve25519 for key exchange, XChaCha20-Poly1305 for symmetric encryption, Argon2id for key derivation (time=3, memory=64MiB, threads=4).
- **Current State**: No cryptographic code exists. No keypair generation, no keystore, no signature operations.
- **Impact**: Users cannot create identities. No authentication is possible. All message signing and verification is unavailable.
- **Closing the Gap**:
  1. Create `pkg/identity/keys/keypair.go` with Ed25519 generation via `crypto/ed25519`
  2. Create `pkg/identity/keys/keystore.go` with Argon2id + XChaCha20-Poly1305 encryption
  3. Implement Curve25519 derivation for key agreement (birational map from Ed25519)
  4. Create `pkg/identity/keys/backup.go` with BIP-39 mnemonic encoding
  5. Implement explicit memory zeroing for key material
  6. **Estimated effort**: 1-2 weeks per ROADMAP.md Priority 2

---

## Gap 4: Wave Content System Missing

- **Stated Goal**: Eight Wave types (Surface, Reply, Veiled, Specter, Sigil, Abyssal, Masked, Beacon) with SHA-256 Proof of Work (2-5s target, difficulty 20), TTL enforcement (max 30 days), and 2,048-byte content limit.
- **Current State**: No Wave implementation exists. No content creation, no PoW, no TTL enforcement, no threading.
- **Impact**: Users cannot publish or receive content. The network has no content to display or propagate.
- **Closing the Gap**:
  1. Define protobuf schema in `proto/wave.proto` per TECHNICAL_IMPLEMENTATION.md §3
  2. Create `pkg/content/waves/wave.go` with Wave struct, creation, validation
  3. Create `pkg/content/pow/proof.go` with SHA-256 PoW (difficulty 20, ~2-5s compute)
  4. Create `pkg/content/propagation/relay.go` with GossipSub broadcast, hop counting, deduplication
  5. Create `pkg/content/storage/cache.go` with LRU cache, TTL sweep (every 60s)
  6. Create `pkg/content/threads/threading.go` for reply chain construction
  7. Implement all 8 Wave types per WAVES.md
  8. **Estimated effort**: 2-3 weeks per ROADMAP.md Priority 4

---

## Gap 5: Anonymous Layer (Specters) Missing

- **Stated Goal**: Pseudonymous Specter identities with Curve25519 keypairs, cryptographically unlinkable from main identity, procedurally generated names (adjective+noun from 65,536-word list), and distinct visual sigils.
- **Current State**: No Specter implementation exists. No anonymous identity creation, no name generation, no sigil differentiation.
- **Impact**: Users cannot participate anonymously. The Shadow Gradient (privacy mode escalation) has no anonymous endpoint. The core differentiator of "anonymity as first-class feature" is unavailable.
- **Closing the Gap**:
  1. Create `pkg/anonymous/specters/specter.go` with independent Curve25519 keypair generation
  2. Create `pkg/anonymous/specters/name.go` with procedural pseudonym generation (BLAKE3 hash → adjective+noun)
  3. Create `assets/wordlists/specter-names.txt` with 65,536 curated entries
  4. Create `pkg/anonymous/specters/sigil.go` with distinct spectral visual style
  5. Ensure no derivation path from main identity (cryptographic unlinkability)
  6. **Estimated effort**: 2 weeks per ROADMAP.md Priority 5

---

## Gap 6: Shroud Onion Routing Missing

- **Stated Goal**: Three-hop onion routing network for anonymous traffic. Per SECURITY_PRIVACY.md: X25519 key exchange, XChaCha20-Poly1305 onion layers, fixed 1KB packet padding, 10-minute circuit rotation, diversity heuristics (no two hops in direct mesh).
- **Current State**: No Shroud implementation exists. No circuit construction, no relay protocol, no traffic padding.
- **Impact**: Guarded and Fortress modes have no traffic analysis resistance. Anonymous traffic is not protected from correlation attacks. The "onion-style circuits" claim in README.md is unfulfilled.
- **Closing the Gap**:
  1. Create `pkg/anonymous/shroud/beacon.go` for Shroud Node announcements
  2. Create `pkg/anonymous/shroud/circuit.go` for three-hop relay chain selection with diversity heuristics
  3. Create `pkg/anonymous/shroud/onion.go` for triple-layer encryption (X25519 + XChaCha20-Poly1305)
  4. Create `pkg/anonymous/shroud/relay.go` for message forwarding with traffic mixing
  5. Implement fixed 1KB padding and 10-minute circuit rotation
  6. **Estimated effort**: 3-4 weeks per ROADMAP.md Priority 6

---

## Gap 7: Resonance Reputation System Missing

- **Stated Goal**: Locally-computed reputation metric with milestone unlocks (Shade/25, Wraith/50, Shade-Wraith/75, Phantom/100, Council-Eligible/200, Abyss/500). Per RESONANCE_SYSTEM.md: logarithmic scaling, four signal categories, time decay, ZK threshold proofs (Pedersen + Bulletproofs).
- **Current State**: No Resonance implementation exists. No scoring algorithm, no milestones, no ZK proofs.
- **Impact**: Specter progression system is unavailable. Anonymous mechanics that require Resonance thresholds cannot function. The engagement loop for anonymous participation is missing.
- **Closing the Gap**:
  1. Create `pkg/anonymous/resonance/scorer.go` with local scoring algorithm per RESONANCE_SYSTEM.md
  2. Create `pkg/anonymous/resonance/decay.go` with time decay (half-life curve)
  3. Create `pkg/anonymous/resonance/milestones.go` with threshold unlocks
  4. Create `pkg/anonymous/resonance/claims.go` with Pedersen commitments + Bulletproofs for ZK claims
  5. **Estimated effort**: 2 weeks per ROADMAP.md Priority 7

---

## Gap 8: Anonymous Mini-Games Missing

- **Stated Goal**: Seven mini-game types per ANONYMOUS_GAME_MECHANICS.md: Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Phantom Councils. Each mechanic is unlocked at specific Resonance milestones.
- **Current State**: No mini-game implementations exist. No game logic, no progression, no cross-layer visibility.
- **Impact**: The gamified anonymous social mechanics that differentiate MURMUR from competitors are entirely unavailable. Specter engagement beyond basic messaging is missing.
- **Closing the Gap**:
  1. Create `pkg/anonymous/mechanics/puzzles.go` for Cipher Puzzles
  2. Create `pkg/anonymous/mechanics/hunts.go` for Specter Hunts
  3. Create `pkg/anonymous/mechanics/territory.go` for Territory Drift
  4. Create `pkg/anonymous/mechanics/oracle.go` for Oracle Pools
  5. Create `pkg/anonymous/mechanics/forge.go` for Sigil Forge
  6. Create `pkg/anonymous/mechanics/shadow.go` for Shadow Play
  7. Create `pkg/anonymous/mechanics/councils.go` for Phantom Councils
  8. Implement Phantom Gifts, Specter Marks, Masked Events
  9. **Estimated effort**: 3 weeks per ROADMAP.md Priority 8

---

## Gap 9: Privacy Modes (Shadow Gradient) Missing

- **Stated Goal**: Four privacy modes — Open (Surface only), Hybrid (both layers), Guarded (encrypted bridge), Fortress (anonymous only). Per SHADOW_GRADIENT.md: mode-aware message routing, traffic isolation, mode transition state machine.
- **Current State**: No privacy mode implementation exists. No mode enumeration, no routing rules, no transition logic.
- **Impact**: Users cannot select or transition between privacy levels. The graduated privacy model (the "Shadow Gradient") is unavailable.
- **Closing the Gap**:
  1. Create `pkg/identity/modes/mode.go` with mode enumeration and state machine
  2. Implement mode-aware message routing (Surface vs Anonymous topics)
  3. Implement mode transition rules and confirmation dialogs
  4. Integrate with Shroud routing for Guarded/Fortress modes
  5. **Estimated effort**: Integrated with Priorities 5-6 per ROADMAP.md

---

## Gap 10: Six-Phase Onboarding Missing

- **Stated Goal**: Guided introduction from first launch to active participation. Per ONBOARDING.md: Welcome → Identity Creation → Mode Selection → Network Bootstrap → Guided Exploration (5 phases documented; other docs reference 6).
- **Current State**: No onboarding implementation exists. No UI screens, no guided flow, no tutorial system.
- **Impact**: New users face a steep learning curve with no guidance. The "complexity is revealed, not imposed" design principle cannot be achieved.
- **Closing the Gap**:
  1. Create `pkg/onboarding/flow/` with phase sequence controller
  2. Create `pkg/onboarding/tutorials/` with guided exploration, contextual hints
  3. Create `pkg/onboarding/bootstrap/` with initial peer connection, first Wave prompt
  4. Reconcile 5-phase vs 6-phase inconsistency across documentation
  5. **Estimated effort**: 3 weeks per ROADMAP.md post-v0.8

---

## Gap 11: libp2p Networking Stack Missing

- **Stated Goal**: Per NETWORK_ARCHITECTURE.md: libp2p v0.48+ with Noise XX transport encryption, QUIC/TCP transports, GossipSub v1.1 with peer scoring, Kademlia DHT, DCUtR hole punching, relay fallback, AutoNAT.
- **Current State**: No networking code exists. No libp2p host, no peer discovery, no gossip, no NAT traversal.
- **Impact**: Nodes cannot connect to each other. The peer-to-peer network does not exist. All "decentralized" claims are aspirational.
- **Closing the Gap**:
  1. Create `pkg/networking/transport/host.go` with libp2p host, Noise encryption
  2. Create `pkg/networking/discovery/dht.go` with Kademlia bootstrap, peer routing
  3. Create `pkg/networking/gossip/pubsub.go` with GossipSub topics, peer scoring
  4. Create `pkg/networking/mesh/manager.go` with 6-12 peer target, heartbeat
  5. Create `pkg/networking/relay/nat.go` with DCUtR, relay fallback, AutoNAT
  6. Establish bootstrap node infrastructure
  7. **Estimated effort**: 2-3 weeks per ROADMAP.md Priority 3

---

## Gap 12: Protobuf Wire Format Missing

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §3: All wire-format messages use Protocol Buffers proto3. MurmurEnvelope wrapper for GossipSub messages with version, type, payload, signature, message_id.
- **Current State**: No protobuf schemas exist. No `.proto` files, no generated `.pb.go` files.
- **Impact**: Wire format is undefined. Messages cannot be serialized or deserialized. Interoperability is impossible.
- **Closing the Gap**:
  1. Create `proto/wave.proto` with Wave, Reply, Amplification, MurmurEnvelope
  2. Create `proto/identity.proto` with IdentityDeclaration, PeerRecord
  3. Create `proto/shroud.proto` with CircuitRequest, OnionCell, RelayAdvertisement
  4. Create `proto/gossip.proto` with GossipEnvelope wrapper
  5. Create `proto/resonance.proto` with ResonanceObservation, ZKClaim
  6. Generate `.pb.go` files and check into repository
  7. **Estimated effort**: Part of ROADMAP.md Priority 4

---

## Gap 13: Storage Layer (Bbolt) Missing

- **Stated Goal**: Per TECHNICAL_IMPLEMENTATION.md §1.5: Bbolt embedded key-value store with buckets: identity, peers, waves, threads, shroud, resonance, config. Hourly TTL sweep, LRU eviction.
- **Current State**: No storage implementation exists. No Bbolt initialization, no bucket CRUD, no garbage collection.
- **Impact**: Data cannot be persisted. Users lose all state on application restart.
- **Closing the Gap**:
  1. Create `pkg/store/db.go` with Bbolt initialization, bucket creation
  2. Create `pkg/store/waves.go` with Wave CRUD, TTL-indexed queries
  3. Create `pkg/store/peers.go` with peer record CRUD
  4. Create `pkg/store/identity.go` with local identity persistence
  5. Create `pkg/store/resonance.go` with Resonance observation persistence
  6. Implement 60-second GC sweep goroutine
  7. **Estimated effort**: Part of ROADMAP.md Priority 2-3

---

## Gap 14: CI/CD Pipeline Missing

- **Stated Goal**: Per ROADMAP.md:179: CI workflow with `go build ./...`, `go test ./...`, `go vet ./...`, `gofumpt` formatting check.
- **Current State**: No GitHub Actions workflows exist. No automated build, test, or lint validation.
- **Impact**: No quality gate for contributions. Regressions can enter the codebase undetected.
- **Closing the Gap**:
  1. Create `.github/workflows/ci.yml` with build, test, vet, format check
  2. Add `govulncheck` for dependency vulnerability scanning
  3. Add `go-licenses` for license compliance checking
  4. Configure branch protection requiring CI pass
  5. **Estimated effort**: Part of ROADMAP.md Priority 1

---

## Gap 15: Sigil Generation Missing

- **Stated Goal**: Deterministic 64×64 visual identity from BLAKE3 hash of public key. Per DESIGN_DOCUMENT.md Part III §10: parametric generation (shapes, colors, patterns), distinct style for Specter sigils (spectral glow).
- **Current State**: No sigil generator exists. No visual identity differentiation.
- **Impact**: Users cannot visually distinguish identities at a glance. The "visual fingerprint" for identity recognition is unavailable.
- **Closing the Gap**:
  1. Create `pkg/identity/sigils/generator.go` with deterministic generation from BLAKE3 hash
  2. Implement parametric system: shapes, colors, patterns
  3. Output 64×64 PNG for cross-platform consistency
  4. Create distinct spectral style for Specter sigils
  5. **Estimated effort**: Part of ROADMAP.md Priority 2

---

## Gap Summary

| Gap | Subsystem | Priority | Estimated Effort |
|-----|-----------|----------|------------------|
| Gap 1: Complete Codebase | Foundation | 1 | 6-9 months total |
| Gap 11: libp2p Networking | Networking | 3 | 2-3 weeks |
| Gap 3: Cryptographic Identity | Identity | 2 | 1-2 weeks |
| Gap 15: Sigil Generation | Identity | 2 | Part of Priority 2 |
| Gap 13: Storage Layer | Storage | 2-3 | Part of Priority 2-3 |
| Gap 12: Protobuf Wire Format | Content | 4 | Part of Priority 4 |
| Gap 4: Wave Content System | Content | 4 | 2-3 weeks |
| Gap 5: Anonymous Layer (Specters) | Anonymous | 5 | 2 weeks |
| Gap 9: Privacy Modes | Anonymous | 5-6 | Part of Priority 5-6 |
| Gap 6: Shroud Onion Routing | Anonymous | 6 | 3-4 weeks |
| Gap 7: Resonance Reputation | Anonymous | 7 | 2 weeks |
| Gap 8: Anonymous Mini-Games | Anonymous | 8 | 3 weeks |
| Gap 2: Pulse Map Visualization | UI | 8 | 4-6 weeks |
| Gap 10: Six-Phase Onboarding | UI | 9 | 3 weeks |
| Gap 14: CI/CD Pipeline | Infrastructure | 1 | Part of Priority 1 |

**Total estimated effort to close all gaps**: 6-9 months with 2-4 engineers, per PLAN.md

---

## Documentation vs Implementation Gaps

Beyond the missing code, there are documentation inconsistencies that should be resolved during implementation:

| Inconsistency | Documents | Resolution |
|---------------|-----------|------------|
| `internal/` vs `pkg/` | TECHNICAL_IMPLEMENTATION.md vs ROADMAP.md | Use `pkg/` per ROADMAP.md ✅ RESOLVED |
| `src/` vs `pkg/` | README.md vs ROADMAP.md | Use `pkg/` per ROADMAP.md ✅ RESOLVED |
| `docs/unified-design-document.md` | README.md:24 | Reference `DESIGN_DOCUMENT.md` ✅ RESOLVED |
| 3 vs 4 privacy modes | DESIGN_DOCUMENT.md Part III vs SHADOW_GRADIENT.md | Use 4 modes per SHADOW_GRADIENT.md ✅ RESOLVED |
| 5 vs 6 onboarding phases | ONBOARDING.md vs other docs | Reconcile phase count ✅ RESOLVED |
| BLAKE2b vs BLAKE3 | SECURITY_PRIVACY.md vs TECHNICAL_IMPLEMENTATION.md | Use BLAKE3 per TECHNICAL_IMPLEMENTATION.md ✅ RESOLVED |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| go-libp2p maintenance transition | Medium | High | Pin to v0.48.x; track community governance |
| NAT traversal complexity | Medium | Medium | Ship relay-only first; add hole-punching incrementally |
| Scope creep to Anonymous Layer | High | Medium | Strict phase gates; v0.1 is Surface Layer only |
| Dependency vulnerability | Low | High | Use `govulncheck`; subscribe to security advisories |
| Contributor overwhelm (552KB docs) | Medium | Medium | Create CONTRIBUTING.md with reading order |

---

*Gaps analysis generated by functional audit workflow*
*Date: 2026-04-13*
*Auditor: Copilot CLI*
