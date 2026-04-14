# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **2026-04-14**: Fix test failure in pkg/pulsemap/rendering on headless environments
  - `pkg/pulsemap/rendering/sigil_image_test.go`: Changed build tag from `!noebiten` to `ebitentest` to match the project's convention for Ebitengine-dependent tests (see `rendering_ebiten_test.go`)
  - Tests now properly skip in headless CI environments where no display is available
  - Category: Cat 1 (Implementation Bug) — test file build tag was misconfigured

### Added

- **2026-04-14**: Oracle Pools network propagation
  - `pkg/anonymous/mechanics/oracle_publisher.go`: Oracle pool event publishing/receiving
    - `OraclePublisher`: Broadcasts oracle events to GossipSub
    - `PublishPoolCreated()`: Announce new prediction pools
    - `PublishCommitment()`: Broadcast hashed prediction commitments
    - `PublishReveal()`: Broadcast revealed predictions
    - `PublishPoolClosed()`: Announce pool closed for predictions
    - `PublishOutcome()`: Announce pool resolution and outcome
    - `OracleReceiver`: Handles incoming oracle events
    - Ed25519 signed events with BLAKE3 signature data
    - Per ROADMAP.md line 459: "Network propagation — broadcast pool creation, commitments, reveals, outcomes"
  - `pkg/anonymous/mechanics/oracle_publisher_test.go`: 22 tests + 2 benchmarks
    - Publisher creation and configuration
    - Pool creation, commitment, reveal, close, outcome publishing
    - Receiver handling for all event types
    - Signature verification round-trip
    - Error handling (nil publisher, nil pool, missing signature)
    - Double resolution protection

- **2026-04-14**: Territory Drift network propagation
  - `pkg/anonymous/mechanics/territory_publisher.go`: Territory event publishing/receiving
    - `TerritoryPublisher`: Broadcasts territory events to GossipSub
    - `PublishInfluenceClaim()`: Announce Specter influence claims
    - `PublishControlChange()`: Announce controller changes
    - `PublishTerritoryDrift()`: Announce territory boundary shifts
    - `TerritoryReceiver`: Handles incoming territory events
    - `TerritoryStore`: In-memory territory storage with CRUD operations
    - Ed25519 signed events with BLAKE3 signature data
    - Per ROADMAP.md line 444: "Network propagation — broadcast influence claims and territory state changes"
  - `pkg/anonymous/mechanics/territory_publisher_test.go`: 22 tests + 2 benchmarks
    - Publisher creation and configuration
    - Influence claim publishing and receiving
    - Control change events
    - Territory drift events
    - Signature verification round-trip
    - Error handling (nil publisher, nil private key, invalid data)
    - TerritoryStore operations (add, get, update, list, remove)

- **2026-04-14**: Louvain community detection for Territory Drift
  - `pkg/anonymous/mechanics/louvain.go`: Louvain clustering algorithm (~420 lines)
    - `LouvainGraph`: Network graph with adjacency list representation
    - `LouvainNode`, `LouvainEdge`: Graph elements with positions and weights
    - `Louvain`: Community detection with modularity optimization
    - `DetectCommunities()`: Iterative modularity-based clustering
    - `DetectTerritories()`: Convert clusters to territory boundaries
    - `Modularity()`: Compute partition quality score
    - `UpdateTerritories()`: Integrate with TerritoryManager
    - Resolution parameter for cluster granularity control
    - Per ROADMAP.md line 443: "Louvain clustering algorithm for territory partitioning"
    - Per ANONYMOUS_GAME_MECHANICS.md: "Territories are defined by the Louvain community detection algorithm"
  - `pkg/anonymous/mechanics/louvain_test.go`: 15 tests + 2 benchmarks
    - Graph construction (nodes, edges)
    - Two-clique detection
    - Territory detection
    - Modularity computation
    - Concurrent access safety

- **2026-04-14**: Hunt Pulse Map visualization effects
  - `pkg/pulsemap/rendering/effects/hunts.go`: Hunt fragment rendering (~340 lines)
    - `HuntEffects`: Manages hunt fragment visuals on Pulse Map
    - `FragmentVisual`: Fragment markers with position, state, claimer info
    - Dim pulsing amber markers for unclaimed fragments
    - Bright glow with claimer sigil for claimed fragments
    - Connecting lines between fragments in active hunts
    - Red pulse warning for expiring hunts
    - Gold victory animation for completed hunts
    - Clue level indicators (cyan dots)
    - Per ROADMAP.md line 433: "Pulse Map visualization — scattered glowing fragments"
  - `pkg/pulsemap/rendering/effects/hunts_stub.go`: Non-Ebitengine stub for headless testing
  - `pkg/pulsemap/rendering/effects/hunts_test.go`: 14 tests + 2 benchmarks
    - Fragment add/remove/claim operations
    - Hunt state transitions
    - Clue reveal tracking
    - Hunt clearing
    - Concurrent access safety

- **2026-04-14**: Hunt network propagation for Specter Hunts
  - `pkg/anonymous/mechanics/hunt_publisher.go`: Hunt event publishing and receiving (~530 lines)
    - `HuntPublisher`: Publishes hunt events to `/murmur/anonymous/mechanics/1.0`
    - Event types: HuntCreated, FragmentClaim, ClueReveal, HuntCompleted, HuntExpired
    - Ed25519 signature verification on all events
    - `HuntReceiver`: Handles incoming hunt events and updates local state
    - Proto conversion functions for Hunt, Fragment, ProximityProof types
    - Per ROADMAP.md line 431: "Network propagation — broadcast Hunt events, fragment claims, clue reveals"
  - `proto/mechanics.proto`: Added Hunt, Fragment, ProximityProof, ProximityAttestation, HuntLeaderboardEntry messages
  - `pkg/anonymous/mechanics/hunt_publisher_test.go`: 17 tests + 2 benchmarks
    - All publish methods (HuntCreated, FragmentClaim, ClueReveal, Completed, Expired)
    - Receiver handling for all event types
    - Round-trip proto conversion tests
    - Error case coverage (invalid signature, hunt not found, missing publisher/key)

- **2026-04-14**: DHT-based proximity proofs for Specter Hunts
  - `pkg/anonymous/mechanics/proximity_proof.go`: Real topological proximity verification
    - `ProximityAttestation`: Signed statements from peers near target locations
    - `DHTProximityProof`: Collects attestations for proximity verification
    - `ProximityVerifier`: Creates attestations and verifies proofs
    - XOR distance calculation per Kademlia DHT topology
    - Hop-based threshold computation (42 bits per hop)
    - TTL enforcement on attestations (5 minutes)
    - Self-attestation rejection for security
    - Legacy proof adapter for backward compatibility
    - Per ROADMAP.md line 430: "Actual proximity proof via DHT routing"
  - `pkg/anonymous/mechanics/proximity_proof_test.go`: 23 tests + 3 benchmarks
    - Attestation creation and signature verification
    - Proof construction and validation
    - XOR distance computation
    - Threshold calculations
    - Legacy proof conversion

- **2026-04-14**: Cipher Puzzle Pulse Map visualization effects
  - `pkg/pulsemap/rendering/effects/puzzles.go`: Puzzle visual effects renderer
    - Rotating cryptographic symbols per puzzle type
    - Fragment: hexagon with inner triangle, golden glow
    - Mosaic: 5 interlocking squares in cross pattern
    - Cascade: 3 horizontal layers with wave animation
    - State indicators: pulsing (active), checkmark (solved), X (expired)
    - Per ROADMAP.md line 417: "Pulse Map visualization — rotating cryptographic symbol"
  - `pkg/pulsemap/rendering/effects/puzzles_stub.go`: Non-Ebitengine stub for headless testing
  - `pkg/pulsemap/rendering/effects/puzzles_test.go`: 12 tests + 2 benchmarks
  
- **2026-04-14**: Cipher Puzzle network propagation for Anonymous Layer
  - `pkg/anonymous/mechanics/puzzle_publisher.go`: Puzzle event publishing and receiving
    - `PuzzlePublisher`: Publishes puzzle events to `/murmur/anonymous/mechanics/1.0`
    - Event types: CREATED, SOLVED, EXPIRED, CONTRIBUTION, STAGE
    - Ed25519 signatures for event authenticity
    - Solution hash transmission (never raw solutions)
  - `PuzzleReceiver`: Handles incoming puzzle events from the network
    - Event verification with signature checking
    - Puzzle store integration for state updates
    - Per ROADMAP.md line 415: "Network propagation — publish puzzle events"
  - `pkg/anonymous/mechanics/puzzle_publisher_test.go`: 19 tests + 2 benchmarks
  - `proto/mechanics.proto`: Added event wrapper messages for all mechanics
    - PuzzleEvent, HuntEvent, TerritoryEvent, OracleEvent, ForgeEvent
    - ShadowPlayEvent, CouncilEvent, GiftEvent, MarkEvent
  - `proto/gossip.proto`: Extended GossipMessage union with mechanics events

- **2026-04-14**: Bulletproofs range proof generation for ZK Resonance claims
  - `pkg/anonymous/resonance/bulletproofs.go`: True Bulletproofs implementation
    - Uses `github.com/coinbase/kryptology/pkg/bulletproof` for cryptographic proofs
    - `BulletproofRangeProver`: Generates 64-bit range proofs (~672 bytes)
    - `BulletproofRangeVerifier`: Verifies range proofs with replay prevention
    - `BulletproofThresholdProver/Verifier`: Proves value >= threshold using delta range proof
    - Serialization/deserialization for network transmission
    - Per SECURITY_PRIVACY.md: "Bulletproofs range proof generation"
  - `pkg/anonymous/resonance/bulletproofs_test.go`: 14 tests + 2 benchmarks
    - Round-trip proof generation and verification
    - Large/zero value handling
    - Replay prevention and timestamp validation
    - Threshold proof success/failure cases

- **2026-04-14**: Echo Index visual color-coding on Pulse Map
  - `pkg/pulsemap/overlays/echoindex.go`: Echo Index overlay renderer
    - `EchoIndexOverlay` draws cluster boundaries with diversity color-coding
    - High Echo Index (>0.7) = warm colors (amber/red) indicating insularity
    - Low Echo Index (<0.4) = cool colors (blue/green) indicating openness
    - Mid-range = neutral gray
    - Animated badges at cluster centers with category icons
    - `NewEchoShadowOverlay()` for Anonymous Layer equivalent
    - Per RESONANCE_SYSTEM.md: color-coded badges on cluster boundaries
  - `pkg/pulsemap/overlays/echoindex_stub.go`: Non-Ebitengine build stub
  - `pkg/pulsemap/overlays/echoindex_test.go`: 13 tests + 1 benchmark

- **2026-04-14**: Surface and Specter milestone visual effects
  - `pkg/pulsemap/rendering/effects/milestones.go`: Milestone effect renderer
    - Surface milestones: Ember(10), Spark(25), Flame(50), Blaze(100), Inferno(200), Corona(500)
    - Specter milestones: Whisper(10), Shade(25), Wraith(50), ShadeWraith(75), Phantom(100), Revenant(200), Abyss(500)
    - `MilestoneEffects` struct with particle pooling and time-based animations
    - `DrawSurfaceMilestone()` dispatches to tier-specific warm/pulsing/particle effects
    - `DrawSpecterMilestone()` dispatches to tier-specific ghostly/ethereal effects
    - Abyss effect only renders in Fortress privacy mode per spec
  - `pkg/pulsemap/rendering/effects/milestones_stub.go`: Non-Ebitengine build stub
  - `pkg/pulsemap/rendering/effects/milestones_test.go`: Milestone threshold tests

- **2026-04-14**: Mutual confirmation protocol for Proximity Ignition
  - `pkg/identity/ignition/confirmation.go`: Two-way handshake verification
    - `ConfirmationSession` tracks challenge-response state machine
    - Order-independent session ID from XOR of public keys
    - `ChallengeMessage()` creates signed 152-byte challenge with nonce
    - `ProcessChallenge()` validates peer's challenge signature
    - `ConfirmationMessage()` responds with signed challenge response
    - `ConfirmationManager` for session lifecycle and cleanup
    - Per VIRAL_GROWTH_AND_ONBOARDING.md: "Both parties accept"
  - `pkg/identity/ignition/confirmation_test.go`: 16 tests + 1 benchmark
    - Full protocol round-trip (Alice initiates, both confirm)
    - Invalid challenge rejection
    - Session expiry (5-minute timeout per spec)
    - Cleanup loop verification

- **2026-04-14**: NFC tap exchange for Proximity Ignition
  - `pkg/identity/ignition/nfc.go`: Compact NFC format for tap-to-connect
    - `NFCIgnitionData` with optimized binary format for NTAG213 compatibility
    - Compact address encoding: IPv4 (6 bytes), IPv6 (18 bytes), PeerID (8 bytes)
    - 124 bytes for IPv4, 136 bytes for IPv6 (within 137-byte NTAG213 limit)
    - Timestamp reduced to uint32 (saves 4 bytes vs full int64)
    - NDEF record wrapper for standard NFC tag formatting
    - `ToIgnitionData()` conversion for protocol compatibility
  - `pkg/identity/ignition/nfc_test.go`: 20 tests + 2 benchmarks
    - Encode/decode round-trips for IPv4/IPv6/multiaddr formats
    - Payload size verification against 137-byte NFC limit
    - NDEF record creation and parsing
    - Signature verification after round-trip
    - Per VIRAL_GROWTH_AND_ONBOARDING.md: NFC enables instant mobile handshake

- **2026-04-14**: Proximity Ignition QR code generation
  - `pkg/identity/ignition/ignition.go`: Complete Proximity Ignition implementation
    - `IgnitionData` struct with public key, addresses, one-time token, and signature
    - `TokenManager` for one-time token generation with expiry and replay prevention
    - `GenerateIgnitionData()` creates signed ignition data for QR display
    - `Encode()`/`DecodeIgnitionData()` for URL-safe Base64 serialization
    - `QRCodeImage()`/`QRCodePNG()` for visual QR code generation
    - Signature verification for authenticity (Ed25519)
  - `pkg/identity/ignition/ignition_test.go`: 16 tests + 2 benchmarks
    - Token lifecycle tests (generate, validate, replay prevention, cleanup)
    - Encode/decode round-trip tests with multiple addresses
    - QR image generation tests (PNG format validation)
    - Per RESONANCE_SYSTEM.md: tokens valid for 5 minutes, ~220 char encoded string

- **2026-04-14**: Propagation latency tracking and validation
  - `pkg/content/propagation/latency.go`: `PropagationMetrics` type for tracking per-hop and per-wave latency
    - `RecordHopLatency()` and `RecordWaveHop()` for latency data collection
    - `ThreeHopLatencies()` for extracting 3-hop cumulative latencies
    - `MeetsTarget()` to validate <500ms requirement per TECHNICAL_IMPLEMENTATION.md §7.2
    - `Stats()` returns violation counts and average latencies
  - `pkg/content/propagation/latency_test.go`: Comprehensive test suite
    - `TestSimulatedThreeHopPropagation` validates <500ms target across 3 hops
    - `TestLatencyBudgetAnalysis` documents latency budget breakdown (~167ms per hop)
    - Concurrent access tests for thread safety

- **2026-04-14**: Oracle Pool and Shadow Play Resonance gating
  - `pkg/anonymous/mechanics/oracle.go`: Added `NewOraclePoolGated()` function enforcing Resonance ≥100 (Phantom milestone)
  - `pkg/anonymous/mechanics/shadowplay.go`: Added `NewShadowPlayGated()` function enforcing Resonance ≥200 (Revenant milestone)
  - Tests for both gated constructors verifying threshold enforcement

- **2026-04-14**: Complete Bbolt persistence for Anonymous Layer mechanics
  - `pkg/store/typed_accessors.go`: Added typed accessors for all mechanics types:
    - `CipherPuzzle`: Get, Put, Delete, List, ListActive
    - `SpecterHunt`: Get, Put, Delete, List, ListActive
    - `Territory`: Get, Put, Delete, List
    - `OraclePool`: Get, Put, Delete, List, ListOpen
    - `ForgeProject`: Get, Put, Delete, List
    - `ShadowPlay`: Get, Put, Delete, List
    - `PhantomCouncil`: Get, Put, Delete, List, ListActive
    - `PhantomGift`: Get, Put, Delete, List, ListGiftsForRecipient
    - `SpecterMark`: Get, Put, Delete, List, ListMarksForTarget
  - `pkg/store/typed_accessors_test.go`: Comprehensive tests for all new accessors

- **2026-04-13**: Pulse Map rendering implementation
  - `pkg/pulsemap/rendering/renderer.go`: Full `Renderer` type with camera transforms, node/edge drawing, glow effects, frustum culling, mouse interaction (pan/zoom/select), node focusing
  - `pkg/pulsemap/rendering/renderer_stub.go`: Stub for noebiten builds
  - `pkg/pulsemap/rendering/renderer_test.go`: 20 tests covering all renderer functionality

- **2026-04-13**: UI panels for Wave composition and settings
  - `pkg/ui/panel.go`: Core Panel interface, Theme system, DefaultTheme() with dark colors per PULSE_MAP.md
  - `pkg/ui/compose.go`: ComposePanel for Wave creation with text input, character limit (2048 per WAVES.md), submit/cancel
  - `pkg/ui/settings.go`: SettingsPanel with 4 categories (Network, Privacy, Display, Waves)
  - `pkg/ui/panel_test.go`: 14 tests for UI components
  - Stub files for noebiten builds

### Changed

- **2026-04-13**: Renamed 30 stuttering files to descriptive names
  - `config/config.go` → `config/defaults.go`
  - `resources/resources.go` → `resources/monitor.go`
  - `ui/ui.go` → `ui/panel.go`
  - `pow/pow.go` → `pow/work.go`
  - `storage/storage.go` → `storage/cache.go`
  - `threads/threads.go` → `threads/index.go`
  - `waves/waves.go` → `waves/types.go`
  - `transport/transport.go` → `transport/host.go`
  - `gossip/gossip.go` → `gossip/pubsub.go`
  - `discovery/discovery.go` → `discovery/dht.go`
  - `mesh/mesh.go` → `mesh/manager.go`
  - `relay/relay.go` → `relay/nat.go`
  - `keys/keys.go` → `keys/keypair.go`
  - `sigils/sigils.go` → `sigils/generator.go`
  - `modes/modes.go` → `modes/state.go`
  - `declarations/declarations.go` → `declarations/profile.go`
  - `specters/specters.go` → `specters/identity.go`
  - `shroud/shroud.go` → `shroud/circuit.go`
  - `resonance/resonance.go` → `resonance/score.go`
  - `layout/layout.go` → `layout/engine.go`
  - `interaction/interaction.go` → `interaction/input.go`
  - `rendering/rendering.go` → `rendering/draw.go`
  - `overlays/overlays.go` → `overlays/layer.go`
  - `effects/effects.go` → `effects/visual.go`
  - `flow/flow.go` → `flow/controller.go`
  - `tutorials/tutorials.go` → `tutorials/guide.go`
  - `bootstrap/bootstrap.go` → `bootstrap/network.go`
  - `screens/screens.go` → `screens/identity.go`
  - `propagation/propagation.go` → `propagation/relay.go`
  - `app/app.go` → `app/murmur.go`

### Completed

- **2026-04-13**: All AUDIT.md items completed and file deleted
  - Pulse Map rendering stubs → full implementation
  - UI panels (none existed) → ComposePanel + SettingsPanel
  - Onboarding screens verified as already implemented
  - Identifier naming violations verified as non-issues (package provides context)
  - File naming stuttering → 30 files renamed
  - Low cohesion files verified as acceptable design

### Added

- **2026-04-14**: Connection priority system for peer topology management
  - `pkg/networking/transport/priority.go`: Four-tier connection priority system (Social/Mesh/DHT/Opportunistic) with libp2p connection manager tagging per NETWORK_ARCHITECTURE.md §7
  - `pkg/networking/transport/priority_test.go`: Comprehensive tests for tier assignment, tag management, and priority enforcement (15 tests)
  - `pkg/networking/mesh/mesh_test.go`: Added 11 tests for priority-based pruning, heartbeat threshold, peer state management (coverage: 56.4% → 98.7%)
  - `pkg/networking/transport/transport_test.go`: Added 10 tests for DHT modes, invalid addresses, close paths, config defaults (coverage: 60.5% → 90.7%)
  - All networking packages now exceed 70% coverage threshold

- **2026-04-14**: Bootstrap node list and peer routing table persistence
  - `pkg/networking/discovery/bootstrap.go`: Hardcoded 8 bootstrap nodes (NA-East, NA-West, Europe, Asia-Pacific) with `mustParseAddrInfo()` parser per NETWORK_ARCHITECTURE.md §3
  - `pkg/networking/discovery/bootstrap_test.go`: 9 tests covering bootstrap node parsing, verification, and edge cases
  - `pkg/networking/discovery/persistence.go`: `PeerTable` type for persisting peer routing table across restarts with Load/Save/Update methods using Bbolt
  - `pkg/networking/discovery/persistence_test.go`: 16 tests for persistence layer (Load/Save/Update/Expiry/TTL)

### Changed

- **2026-04-14**: Renamed `pkg/errors/` to `pkg/murerr/` to avoid shadowing Go's standard `errors` package

### Completed

- **2026-04-14**: All 8 PLAN.md implementation steps completed
  - Step 1: GossipSub message handlers
  - Step 2: Event bus for cross-subsystem communication
  - Step 3: Wave publishing
  - Step 4: Bbolt persistence for game mechanics
  - Step 5: Resonance gating enforcement
  - Step 6: Shroud circuit rotation timer
  - Step 7: Networking package test coverage >70%
  - Step 8: Overlays code duplication extraction

- **2026-04-14**: PLAN.md deleted (all steps completed)

- **2026-04-14**: Added mobile platform build support
  - `scripts/build-mobile.sh`: Gomobile build script for Android APK and iOS xcframework
  - `pkg/pulsemap/interaction/touch.go`: TouchState with pan, pinch-to-zoom, and tap gestures
  - `pkg/pulsemap/interaction/touch_test.go`: 11 tests covering all touch gesture scenarios
  - Supports single-touch pan, two-finger pinch-to-zoom, and tap detection

- **2026-04-14**: Added test files for rendering packages
  - `pkg/onboarding/screens/names_test.go`: 5 tests for GenerateSpecterName determinism, edge cases, distribution
  - `pkg/pulsemap/overlays/overlays_logic_test.go`: Tests for LayerBlend, ParticleEmitter, MiniGameVisualization
  - `pkg/pulsemap/rendering/colors_test.go`: Tests for ColorFromHash, hslToRGB, NodeStyle, computeNodeRadius
  - All packages now pass tests with no `[no test files]` warnings

### Refactored

- **2026-04-14**: Split oversized source files to improve maintainability
  - Extracted `CouncilStore` from `councils.go` to `councils_store.go` (councils.go: 1044 → 949 lines)
  - Extracted `ShadowPlayStore` from `shadowplay.go` to `shadowplay_store.go` (shadowplay.go: 789 → 677 lines)
  - Store classes are separate responsibilities per single-responsibility principle

- **2026-04-14**: Reduced code duplication in Wave signing/key derivation
  - Extracted `deriveAbyssalKeypairFromNonce()` helper in `pkg/content/waves/abyssal.go` to share SHA-256 seed derivation logic
  - Created `Signer` interface in `pkg/content/waves/waves.go` for unified Wave signing
  - Created `signWaveAndComputePoW()` shared implementation used by both Surface and Abyssal Waves
  - Renamed type-specific wrappers: `signAndComputeWavePoW()` and `signAndComputeAbyssalPoW()`
  - Clone pairs reduced from 6 to 4 (33% reduction)

- **2026-04-13**: Reduced code duplication across the codebase
  - Extracted `GenerateSpecterName` to shared `names.go` (eliminates 21-line duplicate in mode_screen.go and mode_screen_stub.go)
  - Extracted `DrawSuccessAnimation` to helpers.go (consolidates success circle + checkmark from bootstrap_screen.go and completion_screen.go)
  - Added `int64ToBytes` helper function in waves.go (eliminates repeated 8-byte timestamp conversions)
  - Fixed duplicate comment on HandleClick in mode_screen.go
  - Duplication metrics improved: clone pairs 9→6, duplicated lines 154→93 (-40%)

- **2026-04-13**: Refactored 5 functions exceeding complexity thresholds
  - `PlaceMark` (marks.go): 8.3 → 5.7 overall (-31%), extracted `createMark`, `signMark`, `storeMark`
  - `drawModeIntro` (mode_screen.go): 7.5 → 5.7 overall (-24%), extracted `drawSurfaceLayerNode`, `drawAnonymousLayerNode`, `drawAnonParticles`, `drawIntroExplanationText`
  - `NewPhantomCouncil` (councils.go): 7.0 → 3.1 overall (-56%), extracted `validateCouncilParams`, `initCouncil`, `addFoundingMember`
  - `drawFirstWavePrompt` (bootstrap_screen.go): 5.7 → 1.3 overall (-77%), extracted `drawWaveInputArea`, `drawWaveSuggestions`, `drawWaveButtons`
  - `New` (gossip.go): 46 → 18 lines (-61%), extracted `buildPeerScoreParams`, `buildDefaultTopicParams`, `buildScoreThresholds`

### Updated

- **2026-04-13**: Updated GAPS.md to reflect current implementation status
  - All 15 original implementation gaps have been resolved
  - Only remaining gap is bootstrap nodes (requires external infrastructure, blocked)
  - Document now serves as historical record of completed work
  - Updated "Documentation vs Implementation Gaps" table showing all inconsistencies resolved

### Fixed

- **2026-04-13**: Fixed flaky `TestValidateInvalidPoW` test in `pkg/content/waves`
  - Root cause: Test assumed PoW at difficulty 8 would always fail validation at difficulty 16
  - Statistical flaw: ~1/256 probability the hash coincidentally has 16+ leading zeros
  - Fix: Corrupt nonce by +1 to deterministically invalidate PoW verification
  - Classified as Cat 2 (Test Spec Error) per complexity-based failure analysis

### Changed

- **2026-04-13**: Updated AUDIT.md bootstrap nodes item to `[~]` (blocked status)
  - The "No bootstrap nodes defined" finding requires live network infrastructure to resolve
  - Cannot be completed through code changes alone; requires community-operated bootstrap nodes
  - Added clarifying note explaining the external dependency

### Added

- **2026-04-13**: Extended stability simulation test infrastructure
  - Created `pkg/app/stability_simulation_test.go` with 1000-node, 72-hour stability test infrastructure
  - TestStabilityShortDuration: 1 minute, 100 nodes (CI-safe quick validation)
  - TestStabilityMediumDuration: 10 minutes, 500 nodes (medium validation)
  - TestStability1000Nodes: 1 hour, 1000 nodes (scale validation)
  - TestStability72Hour: Full 72-hour, 1000 nodes (production validation)
  - StabilityNetwork with metrics: memory tracking, panic recovery, deadlock detection
  - Run full test with: `go test -tags=simulation -run TestStability72Hour -timeout 73h`
  - Completes ROADMAP.md Priority 12 validation requirements

- **2026-04-13**: Test coverage for previously untested packages
  - Created `pkg/config/config_test.go` with tests for default listen addresses and bootstrap peers
  - Created `pkg/identity/declarations/declarations_test.go` with tests for Declaration struct
  - Improved test coverage for core configuration and identity packages

- **2026-04-13**: Phantom Gift visual effects integration for Pulse Map
  - Created `pkg/pulsemap/rendering/effects/gifts.go` with GiftRenderer and GiftEffectConfig
  - Created `pkg/pulsemap/rendering/effects/gifts_test.go` with comprehensive tests
  - TestPhantomGiftVisibility validates ROADMAP.md Priority 8 validation criteria
  - TestResonanceTieredEffects validates Resonance tier requirements (25/50/100)
  - TestGiftExpiration validates 7-day expiration per ANONYMOUS_GAME_MECHANICS.md spec
  - Effect configurations for all 25+ gift effect types (basic, expanded, premium)

### Changed

- **2026-04-13**: Updated ROADMAP.md Priority 12 validation to mark stability test infrastructure complete

## Notes

This changelog was established 2026-04-13 to track implementation progress. Prior work is documented in ROADMAP.md and AUDIT.md.
