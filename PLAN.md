# Implementation Plan: MURMUR v0.1 Foundation

## Project Context
- **What it does**: A decentralized, peer-to-peer social network with dual-layer identity architecture—no servers, no algorithms, no permanent record—where users navigate a force-directed Pulse Map instead of algorithmic feeds.
- **Current goal**: Establish a buildable Go codebase with basic P2P networking and identity (v0.1 Foundation per ROADMAP.md)
- **Estimated Scope**: Large (greenfield implementation from ~552KB of specifications)

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|-------------|---------------|---------------------|
| Decentralized P2P social network | ❌ No code exists | Yes - Priority 1-3 |
| Ed25519 cryptographic identity | ❌ No code exists | Yes - Priority 2 |
| Pulse Map force-directed UI | ❌ No code exists | No - Future phase |
| Ephemeral Waves with PoW/TTL | ❌ No code exists | Partial - Priority 4 |
| Anonymous Layer (Specters) | ❌ No code exists | No - Future phase |
| Shroud onion routing | ❌ No code exists | No - Future phase |
| Resonance reputation system | ❌ No code exists | No - Future phase |
| Anonymous mini-games | ❌ No code exists | No - Future phase |
| Privacy modes (Shadow Gradient) | ❌ No code exists | No - Future phase |
| Six-phase onboarding flow | ❌ No code exists | No - Future phase |
| libp2p networking stack | ❌ No code exists | Yes - Priority 3 |
| No engagement metrics | ✅ Achieved by design | N/A |

**Overall: 1/12 goals achieved (trivially, by design)**

## Metrics Summary

Since the project is **pre-implementation** with zero Go source files, `go-stats-generator` cannot produce metrics:

```
Error: analysis failed: no Go files found in /home/user/go/src/github.com/opd-ai/murmur
```

**Baseline metrics (all zero):**
- Complexity hotspots on goal-critical paths: 0 functions (none exist)
- Duplication ratio: 0% (no code to duplicate)
- Doc coverage: N/A (no exported symbols)
- Package coupling: N/A (no packages)

**Post-implementation targets** (per TECHNICAL_IMPLEMENTATION.md):
- Maximum function complexity: 10 (cyclomatic)
- Minimum doc coverage: 70% for exported symbols
- Maximum duplication ratio: 10%
- Target: 60fps rendering with 500 nodes (Pulse Map phase)

## External Research Summary

### go-libp2p Status (2025-2026)
- **Maintenance transition**: Shipyard ended primary maintenance September 2025; community stewardship underway
- **Breaking changes**: Circuit Relay v1 removed; WebTransport handshake changed (v0.47+)
- **Recommendation**: Pin to v0.48.x stable; avoid experimental WebTransport; monitor community governance

### GossipSub Best Practices (2025)
- Tune peer scoring parameters for network topology
- Implement IP colocation penalties for Sybil resistance
- Use IDONTWANT messages (GossipSub v1.2+) for bandwidth efficiency
- Direct peer marking for bootstrap/trusted nodes

### Competitive Landscape
- **Mastodon/Bluesky**: Federated, not true P2P; MURMUR differentiates with spatial UI and integrated anonymity
- **Scuttlebutt**: P2P gossip, append-only; MURMUR adds anonymous layer and visual topology
- **Session**: Onion-routed messaging; MURMUR extends with social mechanics beyond messaging

## Implementation Steps

### Step 1: Initialize Go Module and Dependencies
- **Deliverable**: `go.mod`, `go.sum` with all foundational dependencies; empty `cmd/murmur/main.go` that compiles
- **Dependencies**: None (first step)
- **Goal Impact**: Unblocks all subsequent development; establishes module path `github.com/opd-ai/murmur`
- **Acceptance**: `go build ./cmd/murmur` exits 0; `go mod verify` passes
- **Validation**: 
```bash
go build ./cmd/murmur && echo "SUCCESS" || echo "FAILED"
```

**Dependencies to add** (per TECHNICAL_IMPLEMENTATION.md §1):
```
github.com/libp2p/go-libp2p v0.48.0+
github.com/libp2p/go-libp2p-pubsub (GossipSub)
github.com/libp2p/go-libp2p-kad-dht (Kademlia)
golang.org/x/crypto (Ed25519, Curve25519, ChaCha20-Poly1305, Argon2)
github.com/zeebo/blake3 (BLAKE3 hashing)
go.etcd.io/bbolt (embedded storage)
google.golang.org/protobuf (wire format)
github.com/hajimehoshi/ebiten/v2 v2.10+ (rendering - deferred import)
```

---

### Step 2: Create Package Directory Structure
- **Deliverable**: Directory tree per ROADMAP.md with placeholder `.go` files containing `package X` declarations
- **Dependencies**: Step 1 (go.mod must exist)
- **Goal Impact**: Establishes architectural boundaries; enables parallel development across subsystems
- **Acceptance**: `go build ./...` succeeds; all packages are importable
- **Validation**:
```bash
go build ./... && go list ./... | wc -l
```

**Packages to create**:
```
cmd/murmur/
pkg/app/
pkg/config/
pkg/networking/transport/
pkg/networking/gossip/
pkg/networking/discovery/
pkg/networking/relay/
pkg/networking/mesh/
pkg/identity/keys/
pkg/identity/sigils/
pkg/identity/declarations/
pkg/identity/modes/
pkg/content/waves/
pkg/content/pow/
pkg/content/propagation/
pkg/content/threads/
pkg/content/storage/
pkg/anonymous/specters/
pkg/anonymous/shroud/
pkg/anonymous/resonance/
pkg/anonymous/mechanics/
pkg/store/
pkg/pulsemap/layout/
pkg/pulsemap/rendering/
pkg/pulsemap/interaction/
pkg/pulsemap/overlays/
pkg/onboarding/flow/
pkg/onboarding/tutorials/
pkg/onboarding/bootstrap/
proto/
```

---

### Step 3: Define Protobuf Schemas and Generate Go Code
- **Deliverable**: `proto/*.proto` files defining wire formats; generated `*.pb.go` files checked into repository
- **Dependencies**: Step 2 (package structure)
- **Goal Impact**: Enables all message serialization; establishes canonical message formats from spec
- **Acceptance**: `protoc --go_out=. proto/*.proto` succeeds; generated code compiles
- **Validation**:
```bash
go build ./proto/... 2>&1 | grep -c "error" | xargs test 0 -eq
```

**Proto files to define** (per TECHNICAL_IMPLEMENTATION.md §3):
- `wave.proto`: Wave, Reply, Amplification, MurmurEnvelope
- `identity.proto`: IdentityDeclaration, PeerRecord, ConnectionAnnouncement
- `shroud.proto`: CircuitRequest, OnionCell, RelayAdvertisement
- `gossip.proto`: GossipEnvelope wrapper
- `resonance.proto`: ResonanceObservation, ZKClaim

---

### Step 4: Implement Core Identity Package
- **Deliverable**: `pkg/identity/keys/` with Ed25519 keypair generation, signing, verification, keystore encryption
- **Dependencies**: Step 3 (protobuf types for declarations)
- **Goal Impact**: Directly achieves "Ed25519 cryptographic identity" goal; foundation for all authentication
- **Acceptance**: Unit tests pass for keypair generation, sign/verify round-trip, keystore encrypt/decrypt
- **Validation**:
```bash
go test -v ./pkg/identity/keys/... -run . 2>&1 | tail -5
```

**Implementation per SECURITY_PRIVACY.md §Cryptographic Primitives**:
- `crypto/ed25519` for signing (32-byte pubkey, 64-byte signature)
- `golang.org/x/crypto/curve25519` for X25519 key exchange via birational map
- `golang.org/x/crypto/argon2` (time=3, memory=64MiB, threads=4) for keystore KDF
- `golang.org/x/crypto/chacha20poly1305` for keystore encryption
- Explicit zeroing of key material before GC eligibility

---

### Step 5: Implement Sigil Generation
- **Deliverable**: `pkg/identity/sigils/` with deterministic 64×64 visual identity from BLAKE3 hash
- **Dependencies**: Step 4 (keypairs exist to derive sigils from)
- **Goal Impact**: Visual identity differentiation; enables UI layer to display identities distinctively
- **Acceptance**: Same public key always produces identical sigil; different keys produce visually distinct sigils
- **Validation**:
```bash
go test -v ./pkg/identity/sigils/... -run TestDeterminism
```

**Implementation per DESIGN_DOCUMENT.md Part II**:
- BLAKE3 hash of public key as seed
- Parametric generation: shapes, colors, patterns
- 64×64 raster output (PNG) for cross-platform consistency
- Specter sigils use distinct visual style (spectral glow, different shapes)

---

### Step 6: Implement Bbolt Storage Layer
- **Deliverable**: `pkg/store/` with database initialization, bucket CRUD, and domain-specific accessors
- **Dependencies**: Step 3 (protobuf types for serialization)
- **Goal Impact**: Enables persistence; required for identity storage, peer records, Wave caching
- **Acceptance**: Unit tests pass for bucket creation, key-value operations, transaction semantics
- **Validation**:
```bash
go test -v ./pkg/store/... 2>&1 | grep -E "(PASS|FAIL)"
```

**Buckets per TECHNICAL_IMPLEMENTATION.md §1.5**:
- `identity`: Local keypairs, declarations, privacy mode
- `peers`: Peer records, last-seen, trust scores
- `waves`: Cached Waves indexed by BLAKE3 hash, TTL metadata
- `threads`: Reply chain indices (parent→children)
- `shroud`: Circuit state, relay lists, rotation schedules
- `resonance`: Specter interaction observations
- `config`: Application configuration

---

### Step 7: Implement libp2p Host Construction
- **Deliverable**: `pkg/networking/transport/host.go` with libp2p host initialization, Noise encryption, multi-transport
- **Dependencies**: Step 4 (identity keypairs for Peer ID derivation)
- **Goal Impact**: Core networking enablement; foundation for all P2P communication
- **Acceptance**: Host starts, listens on configured addresses, derives Peer ID from identity
- **Validation**:
```bash
go test -v ./pkg/networking/transport/... -run TestHostCreation
```

**Implementation per NETWORK_ARCHITECTURE.md §4-5**:
- Noise XX handshake for transport encryption
- QUIC preferred, TCP fallback
- Peer ID from SHA-256 of Ed25519 public key (truncated to 160 bits)
- Connection limits, resource manager configuration

---

### Step 8: Implement Kademlia DHT Discovery
- **Deliverable**: `pkg/networking/discovery/dht.go` with DHT bootstrap, peer discovery, routing table maintenance
- **Dependencies**: Step 7 (libp2p host must exist)
- **Goal Impact**: Enables nodes to find each other; critical for network bootstrapping
- **Acceptance**: Node can bootstrap from hardcoded peers and discover additional peers via DHT walks
- **Validation**:
```bash
go test -v ./pkg/networking/discovery/... -run TestBootstrap
```

**Implementation per DESIGN_DOCUMENT.md Part II §5**:
- Hardcoded bootstrap node list (community-operated, multi-jurisdiction)
- User-configurable bootstrap nodes for community independence
- Peer exchange during gossip as DHT fallback
- Routing table refresh every 10 minutes

---

### Step 9: Implement GossipSub with Peer Scoring
- **Deliverable**: `pkg/networking/gossip/pubsub.go` with topic subscriptions, message validation, peer scoring
- **Dependencies**: Step 7 (libp2p host), Step 8 (peer discovery for mesh population)
- **Goal Impact**: Enables broadcast message propagation; core communication primitive
- **Acceptance**: Two nodes exchange messages via GossipSub topic within 10 seconds
- **Validation**:
```bash
go test -v ./pkg/networking/gossip/... -run TestTwoNodeGossip
```

**Topics per TECHNICAL_IMPLEMENTATION.md §3.1**:
- `/murmur/waves/1`: Wave messages
- `/murmur/identity/1`: Identity declarations
- `/murmur/shroud/1`: Shroud relay advertisements
- `/murmur/pulse/1`: Heartbeat pings (every 30s)

**Peer scoring per DESIGN_DOCUMENT.md Part II §7**:
- Penalize invalid signatures, failed PoW, expired TTL
- IP colocation penalty for Sybil resistance
- Prune consistently misbehaving peers

---

### Step 10: Implement Connection Manager
- **Deliverable**: `pkg/networking/mesh/manager.go` with 6-12 peer target, priority tiers, heartbeat monitoring
- **Dependencies**: Step 9 (gossip mesh must exist to manage)
- **Goal Impact**: Network health and stability; ensures reliable message propagation
- **Acceptance**: Node maintains 6-12 peers; disconnects unresponsive peers after 3 missed heartbeats
- **Validation**:
```bash
go test -v ./pkg/networking/mesh/... -run TestConnectionHealth
```

**Implementation per DESIGN_DOCUMENT.md Part II §6**:
- Priority tiers: identity connections > useful gossip peers > random peers
- 30-second heartbeat ping
- Geographic diversity heuristic (latency clustering)
- Exponential backoff for reconnection

---

### Step 11: Implement NAT Traversal
- **Deliverable**: `pkg/networking/relay/nat.go` with DCUtR hole punching, relay fallback, AutoNAT probing
- **Dependencies**: Step 7-10 (full networking stack)
- **Goal Impact**: Enables residential users behind NAT to participate; critical for real-world deployment
- **Acceptance**: Two NAT-bound nodes establish direct connection via hole punching or relay fallback
- **Validation**:
```bash
go test -v ./pkg/networking/relay/... -run TestNATTraversal
```

**Implementation per DESIGN_DOCUMENT.md Part II §8**:
- DCUtR (Direct Connection Upgrade through Relay) protocol
- AutoNAT probing at startup to detect NAT type
- Relay reservation for double-NAT scenarios
- Relay node selection based on latency and availability

---

### Step 12: Implement Basic CI Workflow ✅ COMPLETE
- **Deliverable**: `.github/workflows/ci.yml` with build, test, vet, format check
- **Dependencies**: Steps 1-11 (code must exist to test)
- **Goal Impact**: Quality gate for all future contributions; prevents regressions
- **Acceptance**: CI passes on push to main; fails on formatting violations or test failures
- **Status**: Implemented with build, test, vet, gofumpt, and license compliance checks

---

### Step 13: Integration Test - Two-Node Message Exchange ✅ COMPLETE
- **Deliverable**: `pkg/networking/integration_test.go` with in-memory two-node gossip test
- **Dependencies**: Steps 7-11 (full networking stack)
- **Goal Impact**: Validates end-to-end networking functionality; acceptance test for v0.1
- **Acceptance**: Two in-memory libp2p hosts discover each other and exchange signed messages within 10 seconds
- **Status**: Implemented with TestIntegrationTwoNodeGossip and TestIntegrationMultipleTopics

**Test results**:
```
=== RUN   TestIntegrationTwoNodeGossip
    integration_test.go:126: Message received from 12D3KooW...: "Hello, MURMUR! This is a test Wave."
--- PASS: TestIntegrationTwoNodeGossip
=== RUN   TestIntegrationMultipleTopics
--- PASS: TestIntegrationMultipleTopics
```

---

## Milestone: v0.1 Foundation Complete

**Definition of Done** (per ROADMAP.md):
- `go build ./...` succeeds
- Two nodes exchange signed test messages via GossipSub
- CI passes on main branch
- All packages have at least placeholder documentation

**Post-v0.1 metrics targets for `go-stats-generator`**:
```bash
go-stats-generator analyze . --skip-tests --format json | jq '
  .functions.above_complexity_threshold < 5 and
  .documentation.coverage >= 0.7 and
  .duplication.ratio <= 0.1
'
```

---

## Dependency Graph

```
Step 1 (go.mod) ──► Step 2 (packages) ──► Step 3 (protobuf)
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    ▼                         ▼                         ▼
              Step 4 (identity)         Step 6 (store)            Step 5 (sigils)
                    │                         │                         │
                    └─────────────────────────┼─────────────────────────┘
                                              ▼
                                        Step 7 (transport)
                                              │
                                              ▼
                                        Step 8 (discovery)
                                              │
                                              ▼
                                        Step 9 (gossip)
                                              │
                                              ▼
                                        Step 10 (mesh)
                                              │
                                              ▼
                                        Step 11 (NAT)
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    ▼                         ▼                         ▼
              Step 12 (CI)            Step 13 (integration)        v0.1 Done
```

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| go-libp2p breaking changes | Medium | High | Pin to v0.48.x; avoid WebTransport; track community |
| NAT traversal complexity | Medium | Medium | Ship with relay-only first; add hole-punching incrementally |
| Scope creep to Anonymous Layer | High | Medium | Strict phase gates; v0.1 is Surface Layer only |
| Dependency vulnerability | Low | High | Use `govulncheck`; subscribe to security advisories |

---

## Post-v0.1 Phases (Summary)

| Phase | Version | Focus | Effort |
|-------|---------|-------|--------|
| v0.2 | Identity | Backup/restore, full keystore, declarations | 1-2 weeks |
| v0.3 | Content | Waves with PoW, TTL, threading, storage | 2-3 weeks |
| v0.4 | Anonymous Foundation | Specters, privacy modes | 2 weeks |
| v0.5 | Shroud | Onion routing, circuit construction | 3-4 weeks |
| v0.6 | Resonance | Reputation scoring, milestones | 2 weeks |
| v0.7 | Mechanics | Gifts, Marks, Territory | 3 weeks |
| v0.8 | Pulse Map | Force-directed visualization | 4-6 weeks |
| v0.9 | Onboarding | Six-phase guided flow | 3 weeks |
| v1.0 | MVP | Integration, hardening, polish | 4-6 weeks |

**Total estimated effort to v1.0**: 6-9 months (2-4 engineers)

---

## Notes

- This plan addresses **Priority 1-3** of ROADMAP.md (Foundation, Identity, Networking)
- All implementation must follow conventions in TECHNICAL_IMPLEMENTATION.md
- Use `pkg/` directory structure per ROADMAP.md (not `internal/` despite references in TECHNICAL_IMPLEMENTATION.md)
- All wire formats use Protocol Buffers proto3; JSON is never used on wire or in storage
- Security-critical code (cryptographic operations) requires explicit code review notation in AUDIT.md
