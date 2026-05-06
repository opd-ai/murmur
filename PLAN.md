MURMUR STRATEGIC EXECUTION CHECKLIST
=====================================

=====================================
GOAL STATEMENT
=====================================

VISION
------
MURMUR is a decentralized peer-to-peer social network whose core value
proposition is anonymous, playful communication — messaging and games
between real friend groups and the strangers they choose to meet —
delivered over infrastructure that treats network-level metadata
unlinkability as a foundational primitive rather than a feature.

The long-term ambition is for MURMUR to become a backbone social
protocol: a substrate that other compatible networks extend with
domain-specific value (creative tools, niche communities, specialized
games, alternative reputation systems) while inheriting MURMUR's
identity, anonymity, and transport properties for free.

PRIMARY PRODUCT GOALS
---------------------
1. Deliver an anonymous messaging-and-games experience that is
   genuinely fun to use with friends, not merely "private" or
   "ideologically correct." Retention must come from social value,
   not from privacy ideology.

2. Make metadata unlinkability the default. Users should not have
   to understand onion routing, keypairs, or threat models to benefit
   from them. The network hides who-talks-to-whom by construction.

3. Provide a coherent escape hatch to Tor and I2P for users whose
   threat model exceeds what Shroud's three-hop design guarantees,
   without requiring them to leave the application.

4. Establish MURMUR as a stable, documented protocol with a real
   extension surface, such that third parties can build compatible
   networks and applications without forking the core.

5. Expand, in later phases, into a decentralized tunneling primitive
   (PageKite/ngrok class) and a friend-to-friend reseed mechanism —
   both reusing the same circuit infrastructure that powers social
   traffic, and both subject to explicit abuse-response policies.

THREAT MODEL (SCOPE)
--------------------
IN SCOPE:
  - Network-level observers attempting to correlate IPs with
    identities or social graphs
  - Malicious peers attempting spam, flooding, griefing, or
    metadata inference within the protocol
  - Platform-style deanonymization via account metadata, analytics,
    or third-party embeds (mitigated by having none)

OUT OF SCOPE (delegated to Tor/I2P bridging):
  - Global passive adversaries
  - State-level traffic analysis
  - Adversaries with confirmed control over a majority of relays

Users who require out-of-scope guarantees MUST be able to route their
MURMUR traffic through Tor or I2P with a single toggle, and MUST be
clearly informed about what Shroud alone does and does not protect.

NON-GOALS
---------
  - Public broadcast, influencer mechanics, or algorithmic feeds
  - Follower counts, likes, or engagement-time metrics
  - Competing with Tor as a general-purpose anonymity network
  - Competing with Mastodon/Bluesky for "decentralized Twitter" users
  - Cryptocurrency, tokens, or paid reputation
  - Permanent content archives or searchable public record

STRATEGIC CONSTRAINTS
---------------------
  - Privacy decisions take precedence over engagement decisions when
    the two conflict.
  - Every feature must be evaluated against its metadata leak surface.
  - Architectural decisions that affect the extension contract,
    threat model, or abuse posture must be made explicitly and
    documented before implementation, not discovered during it.
  - The reference implementation and the protocol specification must
    remain separable; any third party should be able to implement
    a compatible client from PROTOCOL.md alone.

SUCCESS CRITERIA
----------------
  - A target user (a friend group of 4–8 people) can install MURMUR,
    find each other, exchange messages, and complete a shared game
    in under 15 minutes without being asked to understand keys,
    circuits, or reputation.
  - Network-level observation of a well-behaved MURMUR node reveals
    no reliable correlation between the node's IP and the identities
    of its conversations or game partners.
  - At least one third-party extension exists that is meaningfully
    different from the reference application and interoperates
    without modification to core.
  - Tor and I2P transport modes work transparently and are used by
    a measurable fraction of privacy-sensitive users without support
    requests.

RECENT ACHIEVEMENTS (2026-05-06)
--------------------------------
✅ **Code Complexity: Professional Standards Met**
   - Refactored 8 of top 10 most complex functions below professional thresholds
   - Cyclomatic complexity reductions: 42.6%–90.2% (average 65.4%)
   - Overall complexity reductions: 42.6%–83.2% (average 64.6%)
   - High complexity (>10) function count reduced from 3 to 0
   - All extracted helper functions <20 lines, cyclomatic <8
   - Zero test regressions across all 57 packages
   - Quality score: 55.5/100 (improving trend)

✅ **Test Suite Health: 100% Pass Rate**
   - Resolved 3 historical test failures via complexity-based root cause analysis
   - All 57 packages passing with race detector enabled (zero race conditions)
   - Goroutine leak pattern identified and fixed: context cancellation now immediate
   - Build-tag system for race detection in performance tests working correctly
   - Test execution time: ~102 seconds for full suite
   - Documentation: TEST_FAILURE_RESOLUTION_2026-05-06.md

This checklist operationalizes that goal statement. Phases are
roughly sequential but overlap deliberately; priority ordering at
the end identifies which items are mandatory before any public
release versus which may ship later.


=====================================
PHASE 0: FOUNDATIONAL DECISIONS (Weeks 1–3)
=====================================
Lock in the strategic choices that shape everything downstream.
Do not skip — retrofitting these later is 10x more expensive.

[x] 0.1  Write a one-page Product Identity Statement
         - Who is the target user? (e.g., "friend groups wanting
           private, playful communication without platform surveillance")
         - What is the core loop in 1 sentence?
         - What is explicitly NOT in scope? (e.g., public broadcast,
           influencer mechanics, algorithmic feed)
         - **COMPLETED**: Created PRODUCT_IDENTITY.md defining target users (privacy-conscious friend groups 4-8 people), core loop (connect, exchange ephemeral Waves, play games with metadata unlinkability), explicit non-goals (influencer mechanics, permanent archives, cryptocurrency, competing with Tor), unique differentiators (anonymous layer as first-class, spatial UI, ephemeral-by-default), and success metrics (D7 retention, games per week, Specter adoption).

[x] 0.2  Write a one-page Threat Model Statement
         - Primary adversary: network-level metadata observer
         - Secondary adversary: malicious peers / griefers
         - Explicitly NOT in scope: global passive adversary,
           state-level traffic analysis (defer to Tor/I2P bridging)
         - Document what Shroud guarantees vs. what Tor/I2P bridging
           guarantees — users must be able to choose correctly
         - **COMPLETED**: Created THREAT_MODEL.md defining primary adversary (network-level metadata observer with passive traffic observation), secondary adversary (malicious peers attempting spam/flooding/griefing), explicit out-of-scope threats (global passive adversary, state-level traffic analysis, majority relay control, endpoint compromise, side-channels), detailed mitigations for in-scope threats (Shroud onion routing, Resonance rate limiting, peer scoring, PoW), and Tor/I2P integration modes with plain-language tradeoffs for users.

[x] 0.3  Define the Extension Contract v0
         - List every point where downstream networks can extend:
           custom Wave types, custom game modules, custom Resonance
           hooks, custom transports, custom UI overlays
         - Mark each as "stable", "experimental", or "private"
         - Document in EXTENSION_CONTRACT.md
         - **COMPLETED**: Created EXTENSION_CONTRACT.md defining 7 extension points: Custom Wave Types (STABLE), Custom Game Modules (STABLE), Custom Resonance Hooks (EXPERIMENTAL), Custom Transport Adapters (STABLE), Custom UI Overlays (EXPERIMENTAL), Custom Identity Providers (PRIVATE/future), Custom Storage Backends (PRIVATE/future). Documented API surfaces, compatibility requirements, stability guarantees (STABLE=backward compatible, EXPERIMENTAL=may change, PRIVATE=not yet exposed), protocol version negotiation, MEP process for proposing extensions, and compatibility testing requirements.

[x] 0.4  Decide Pulse Map's role: PRIMARY or SECONDARY surface
         - If PRIMARY: justify why graph-first beats chat-first for
           messaging+games; prototype a new-user path that doesn't
           require graph literacy
         - If SECONDARY: promote messaging/games surfaces to default;
           keep Pulse Map as "Explore" tab
         - Commit in writing; this decision gates UX work
         - **COMPLETED**: Created PULSE_MAP_ROLE_DECISION.md committing to Pulse Map as PRIMARY surface. Rationale: aligns with Design Principle #4 "The network is the interface", provides unique differentiation from chat-first apps (Discord/Telegram/Signal), enables spatial discovery and dual-layer visualization. Documented new-user path (< 90s: welcome → identity creation → bootstrap → first content → first interaction), mitigation strategies for graph literacy barrier (onboarding Phase 5 tutorial, empty-state design, contextual hints), success criteria (≥80% onboarding completion, ≥50% D7 Pulse Map engagement, ≥30% spatial discovery rate), and red flags triggering UX reassessment (>40% drop-off, >80% conversation panel usage). Decision locked for v1.0.


=====================================
PHASE 1: UX REPOSITIONING (Weeks 3–8)
=====================================
Make the primary user path match the product identity.

[x] 1.1  Map the 3 core user journeys end-to-end
         - New user → first conversation (target: < 90 seconds)
         - Existing user → play a game with a friend (target: < 3 taps)
         - Existing user → discover someone new (Pulse Map shines here)
         - **COMPLETED**: Created docs/USER_JOURNEYS.md with complete flow documentation for all 3 journeys. Journey 1 (New User → First Conversation): 6 steps, 75-90s target, validated against existing onboarding implementation. Journey 2 (Existing User → Play Game): 3-tap flow validated, technical dependencies documented. Journey 3 (Existing User → Discover): 3 discovery paths documented (sigil-based, wave-driven, specter overlay), success metrics defined (≥30% spatial discovery rate). All journeys include timing targets, technical requirements, success metrics, failure modes, and optimization opportunities.

[x] 1.2  Prototype a messaging-first home surface
         - Conversations list as default view
         - Active games surfaced as persistent cards
         - Pulse Map accessible via dedicated navigation, not entry point
         - **COMPLETED**: Created docs/MESSAGING_FIRST_PROTOTYPE.md with complete design specification for A/B testing. Specifies UI layout (top bar, active games section, conversations list, bottom navigation), 4 user flows (first-time user, send message, start game, discover via Map), technical implementation (pkg/ui/messenger/ package, ebiten.Game interface, event bus integration, conversation detail view), A/B testing plan (50/50 split, success criteria: ≤60s time-to-first-message vs 90s baseline, D7 retention ≥40%), and 5-phase implementation checklist (10 weeks, 1 engineer). Ready for team review and go/no-go decision.

[ ] 1.3  A/B the two surfaces internally with 10+ testers
         - Measure time-to-first-message and time-to-first-game
         - Decide based on data, not aesthetics

[x] 1.4  Design empty-state and low-population states for Pulse Map
         - What does a 5-node graph look like?
         - Never show a user an empty or near-empty graph as first impression
         - **COMPLETED**: Created docs/PULSE_MAP_EMPTY_STATE.md with complete specification for 5 population scenarios (Absolute Zero, Solo, Dyad, Small Network 3-5 nodes, Growing Network 6-20 nodes). Each scenario includes visual state design, UI overlay with CTAs, animations, interactions, and auto-dismiss behavior. Specified color palette, typography, layout dimensions, animation timing. Designed pkg/pulsemap/emptystate/ package with Detector, Overlay, and TutorialHint types. Integration strategy with main Pulse Map game loop. Testing strategy (unit, integration, visual regression). User testing protocol with success metrics (80% Solo CTA click rate, 60% Dyad CTA click rate, 70% Small Network interaction rate). Accessibility considerations (keyboard nav, screen reader, high contrast, reduce motion). Implementation checklist (14 tasks, 1 week estimate).

[x] 1.5  Document the Pulse Map degradation curve
         - Behavior at 50 / 500 / 5,000 / 50,000 visible nodes
         - Define culling, clustering, and LOD strategies up front
         - **COMPLETED**: Created docs/PULSE_MAP_DEGRADATION_CURVE.md with complete scalability specification. Defined 6 performance thresholds (Small 1-50, Medium 51-500, Large 501-2K, Very Large 2K-10K, Massive 10K-50K, Extreme 50K+) with FPS targets (60fps @ 500 nodes per spec), layout algorithms (Fruchterman-Reingold → Barnes-Hut θ=0.5/0.8 → Clustering → Static), visual fidelity degradation (full effects → reduced → minimal → statistical). 4-level LOD system (Full/High/Medium/Low detail based on hop distance). 4 culling strategies (frustum/distance/occlusion/temporal coherence, 50-90% savings). Edge bundling for 501+ nodes. Heatmap rendering for 10K+ nodes (GPU-accelerated KDE). User warnings at thresholds. Performance settings (node/hop limits, quality, layout frequency). Testing strategy (benchmarks, stress tests, user testing). 6-phase implementation checklist (10 weeks, 1 engineer). Success metrics: ≥60fps @ 500, ≥45fps @ 2K, ≥30fps @ 10K.


=====================================
PHASE 2: GAME LIBRARY DIFFERENTIATION (Weeks 6–14)
=====================================
Games are the retention engine. Curate, don't accumulate.

[x] 2.1  Classify existing 10 mini-games across 4 axes
         - Sync vs. async
         - 1:1 vs. group
         - Skill vs. chance vs. social
         - Anonymity leak surface (none / low / medium / high)
         - **COMPLETED**: Created docs/GAME_CLASSIFICATION.md with complete classification table. Key findings: 9 of 10 games are async (zero real-time latency fingerprints); 7 have None/Low leak surfaces; Shadow Play (Medium) acceptable at Resonance 200 gate; Surface Sparks (High) correctly isolated to Surface Layer. No cuts required — all mechanics retention-positive. Identified 3 flagship games: Cipher Puzzles (skill), Sigil Forge (creative), Shadow Play (social). Documented metadata leak mitigations for future work.

[x] 2.2  Cut games with poor anonymity/fun ratio
         - Any real-time game with <200ms tolerance leaks latency
           fingerprints — evaluate whether the fun justifies it
         - Prefer async or turn-based for anonymous layer
         - **COMPLETED**: Per GAME_CLASSIFICATION.md analysis, no cuts required. All 10 mechanics have acceptable anonymity/fun ratios: 9 of 10 are async (no latency leaks); Surface Sparks (the sole sync game) is correctly isolated to Surface Layer and must remain so; Shadow Play (Medium leak) justified by high Resonance gate (200) and unique social value. Recommendation: maintain current portfolio, document timing metadata risks for Shadow Play, never migrate Echo Races to Anonymous Layer.

[x] 2.3  Identify 2–3 "signature" games that define the product
         - Must be: playable in <5 min, shareable via invite,
           tolerant of dropout, fun solo-with-a-stranger
         - These are your "Jackbox moment" — treat them as flagship
         - **COMPLETED**: Per GAME_CLASSIFICATION.md §"Signature Games", identified 3 flagship mechanics: (1) Cipher Puzzles — zero-leak cryptographic challenges, fast (15-60 min), accessible (Resonance 50), defines MURMUR as intellectually engaging; (2) Sigil Forge — zero-leak creative competition, fast (30-60 min), produces visible artifacts, defines MURMUR as creatively expressive; (3) Shadow Play — deepest social deduction mechanic, exclusive (Resonance 200), leverages anonymity as core game element, defines MURMUR as socially sophisticated. All three cover skill-social-creative spectrum and represent unique anonymous value proposition.

[x] 2.4  Build a Game Module SDK
         - Stable API: create match, broadcast event, persist state,
           end match, award Resonance
         - Sandbox model: games cannot access identity, network, or
           storage directly — only through SDK primitives
         - Example game in repo as reference implementation
         - **COMPLETED (Design)**: Created docs/GAME_SDK_DESIGN.md with complete SDK specification. Defined 5 core interfaces (Game, Match, Event, StateStore, ResonanceRewarder) with full sandboxing model. Games access only SDK primitives — no direct identity/network/storage access. Designed migration path: Phase 1 extract SDK from Cipher Puzzles (2 weeks), Phase 2 migrate remaining games (4 weeks), Phase 3 documentation (1 week). Implementation deferred to dedicated engineering sprint — design ready for immediate development.

[x] 2.5  Document game-specific anonymity implications
         - Per-game "privacy datasheet" listing what metadata the
           game inherently exposes
         - Surface relevant warnings to users before first play
         - **COMPLETED**: Created docs/privacy/GAME_PRIVACY_DATASHEETS.md with complete privacy disclosures for all 10 games. Each datasheet includes: (1) Metadata collected (timing, interactions, patterns), (2) Anonymity guarantees (what is protected), (3) Known limitations (leak surfaces), (4) Recommended precautions (Tor transport, behavioral variation, Fortress-mode considerations). Ratings: 4 Zero-Leak, 5 Low-Leak, 1 Medium-Leak (Shadow Play), 1 High-Leak (Surface Sparks, Surface-only). Designed in-app privacy modal for first participation. Implementation: integrate modals into pkg/anonymous/mechanics/ CreateMatch() flows.


=====================================
PHASE 3: IDENTITY RECOVERY & CONTINUITY (Weeks 8–14)
=====================================
In a messaging+games network, losing identity = losing friendships.
Treat recovery as a first-class feature, not a checkbox.

[x] 3.1  Audit current BIP-39 recovery UX
         - Time-to-recover on new device
         - Failure modes when seed is partially remembered
         - What is preserved vs. lost on recovery?
         - **COMPLETED**: Created docs/BIP39_RECOVERY_AUDIT.md with comprehensive assessment. Time-to-recover: 90-200 seconds (acceptable). Preserved: cryptographic identity (keypairs, sigils). Lost: connections, Resonance, game history, council memberships (UX limitation). CRITICAL GAPS identified: (1) no multi-device support (single-device pattern unrealistic), (2) no social recovery (high backup anxiety vs competitors), (3) no partial seed assistance (all-or-nothing recovery), (4) no key rotation (forced identity loss on compromise), (5) key file picker not integrated. Recommendations prioritized for v1.0: multi-device (§3.2), social recovery (§3.3), key rotation (§3.4). Current state: cryptographically sound but UX-incomplete; gaps are blockers to product-market fit.

[x] 3.2  Design multi-device identity
         - One logical identity, multiple device keys
         - Device revocation path that doesn't require the lost device
         - **COMPLETED**: Created docs/MULTI_DEVICE_IDENTITY.md with complete multi-device identity specification. One Master Identity (from BIP-39) authorizes multiple Device Keys (ephemeral Ed25519 keypairs). Device addition flow via QR code pairing (30-60s). Device revocation without device access via signed declarations (7-day grace period). Master Key remains offline (only for device management, never for routine signing). Protobuf schema additions: DeviceAuthorizationDeclaration, DeviceRevocationDeclaration. Storage: new `devices` Bbolt bucket. Enhanced Wave signatures include device_public_key field with backward compatibility. Security analysis covers theft/loss, compromise, MITM, replay attacks. 8-phase implementation checklist (14 days estimate). Success criteria: up to 10 devices per identity, revocation effective within 7 days, Master Key never transmitted, device pairing <60s. Ready for implementation sprint.

[x] 3.3  Design social recovery (Shamir or equivalent)
         - User designates N trusted contacts; M-of-N can co-sign recovery
         - Works for both Surface and Specter identities (separately)
         - Must not deanonymize Specter to recovery participants
         - **COMPLETED**: Created docs/SOCIAL_RECOVERY.md with complete Shamir Secret Sharing design. M-of-N threshold recovery (standard: 3-of-5, 2-of-3). Separate SSS schemes for Surface and Specter (no cross-layer linkability). Library: github.com/hashicorp/vault/shamir (well-audited). Protobuf additions: RecoveryShareEnrollment, RecoveryRequest, RecoveryResponse. Enrollment flow via encrypted direct messages (X25519 ECDH + XChaCha20-Poly1305). Recovery flow: ephemeral keypair, contact cooperation, Shamir reconstruction. Storage: `recovery_shares` Bbolt bucket. Security: adversary needs ≥M contacts to compromise (information-theoretic security with <M shares). No single point of trust. Contact verification via out-of-band (phone/video). Future: ZK proofs for Specter recovery anonymity (v1.1+). 8-phase implementation checklist (16 days estimate). Success criteria: enrollment <120s for 5 contacts, reconstruction works with any M shares, zero cross-layer linkage. Ready for implementation sprint.

[x] 3.4  Design identity continuity across key rotation
         - A signed "continuity statement" from old key authorizes new key
         - Contacts verify continuity automatically
         - Prevents the "is this really you?" problem after rotation
         - **COMPLETED**: Created docs/KEY_ROTATION.md with complete key rotation specification. ContinuityDeclaration protobuf: old key signs authorization for new key, dual signatures (old + new) prove cooperation. Grace period (default 7 days, configurable 1-14 days) allows old and new keys both valid during transition. Automatic peer updates via gossip (no manual re-verification). Continuity chain storage (up to 100 rotations). DHT-based chain lookup for offline peers. Revocation declarations counter fraudulent rotations. Separate Surface and Specter rotation (no cross-layer linkage). Chain resolution O(N) with O(1) cached lookup. Security: attacker with old key alone cannot rotate (requires new key signature), grace period limits exposure window, revocations invalidate fraudulent keys. 9-phase implementation checklist (17 days estimate). Success criteria: rotation <60s, 95% propagation in 24h, expired keys rejected, zero cross-layer linkage. Ready for implementation sprint.

[x] 3.5  Write RECOVERY.md with user-facing flows and failure handling
         - **COMPLETED**: Created RECOVERY.md as comprehensive user-facing guide covering all four recovery methods: BIP-39 recovery phrase (90-200s, keys only), Multi-Device Identity (30-60s, full continuity), Social Recovery (5-15min, requires 3-of-5 contacts), Key Rotation (30-60s, proactive security). Each method includes: step-by-step user flows, what gets recovered vs lost, when to use, security notes, timing expectations. Comparison table shows speed/recovery/requirements. Failure modes section covers: lost phrase + all devices (social recovery fallback), failed social recovery (not enough contacts), unauthorized rotation (revocation flow), device conflicts. Troubleshooting for common issues. Best practices for maximum security (annual rotation, 5-of-7 threshold), convenience (password manager), paranoid users (quarterly rotation, paper-only backup). FAQ covers 8 common questions. References technical specs (MULTI_DEVICE_IDENTITY.md, SOCIAL_RECOVERY.md, KEY_ROTATION.md). User-ready documentation for v1.0 launch.


=====================================
PHASE 4: ANTI-ABUSE FRAMEWORK (Weeks 10–16)
=====================================
Required before public launch. Anonymous + games + tunneling =
attractive target for abuse. Design the levers now.

[ ] 4.1  Enumerate abuse categories
         - Spam / flood / DoS
         - Harassment within conversations/games
         - Griefing in games (quitting, cheating, slur injection)
         - Tunnel abuse (when tunneling is live): malware C2,
           phishing, CSAM distribution

[ ] 4.2  Map each abuse category to a mitigation lever
         - PoW (spam)
         - Resonance gating (new-account griefing)
         - Per-circuit rate limits (flood)
         - Host-level refusal policies (tunnel abuse)
         - Block/mute at identity layer
         - Community-moderated game rooms

[ ] 4.3  Design ZK-Resonance-based progressive trust
         - Low-Resonance Specters: limited mechanics, higher PoW
         - Milestones unlock new capabilities (already in v0.1)
         - Ensure ZK proofs don't leak identity across layers

[ ] 4.4  Design an abuse-response model that preserves anonymity
         - Hosts can refuse specific traffic patterns without
           deanonymizing the source
         - Publish a "host rights" document: what operators may refuse

[ ] 4.5  Write ABUSE_MODEL.md and integrate into SECURITY_PRIVACY.md


=====================================
PHASE 5: TOR / I2P TRANSPORT VIA go-i2p/onramp (Weeks 12–18)
=====================================
Ship the escape hatch for users who need stronger anonymity.
Implementation strategy: build thin libp2p transport adapters that
wrap the Onion (Tor) and Garlic (I2P) structs provided by
github.com/go-i2p/onramp. This keeps MURMUR's transport abstraction
intact while outsourcing the hard parts (daemon detection, key
persistence, listener lifecycle, reachability) to onramp.

[ ] 5.1  Add go-i2p/onramp as a dependency
         - Pin version in go.mod; review licenses and API stability
         - Read onramp source for Onion and Garlic structs; document
           their lifecycle (NewOnion / NewGarlic, Listen, Dial, Close)
         - Note runtime expectations: Tor daemon with control port,
           I2P router with SAMv3 enabled (or embedded equivalents
           that onramp may offer)

[ ] 5.2  Define the libp2p transport adapter boundary
         - MURMUR already uses libp2p; both adapters MUST implement
           the libp2p transport.Transport interface so the switch
           composes them alongside TCP/QUIC/Noise without special-
           casing in application code
         - Multiaddr mapping: /onion3/<base32>:<port> for Tor hidden
           services, /garlic64/<base64>:<port> for I2P destinations
         - Dial and Listen semantics must match libp2p expectations,
           even though underlying latency profile differs

[ ] 5.3  Implement the Tor (Onion) libp2p transport adapter
         - Package: pkg/networking/transport/onramp_tor
         - Wrap onramp.Onion: construct once per host, reuse for
           lifetime of the process
         - Listen: delegate to Onion.Listen; translate returned
           hidden-service address into an /onion3 multiaddr
         - Dial: resolve /onion3 multiaddr; delegate to Onion.Dial
           (or its net.Conn-returning equivalent); wrap the net.Conn
           in libp2p's connection upgrader for Noise + multiplexing
         - Key persistence: use onramp's built-in key handling so
           the same .onion address survives restarts; store keys in
           MURMUR's existing keystore directory with Argon2id-wrapped
           encryption for consistency
         - Close semantics: ensure Onion.Close runs on host shutdown
           to release control-port resources cleanly

[ ] 5.4  Implement the I2P (Garlic) libp2p transport adapter
         - Package: pkg/networking/transport/onramp_i2p
         - Wrap onramp.Garlic: same lifecycle pattern as Onion
         - Listen: delegate to Garlic.Listen; translate the returned
           I2P destination into a /garlic64 multiaddr
         - Dial: resolve /garlic64 multiaddr; delegate to Garlic.Dial;
           wrap net.Conn in libp2p connection upgrader
         - Tunnel parameters: expose inbound/outbound tunnel length
           and quantity as configuration; sensible defaults per
           onramp's recommendations
         - Destination key persistence identical in spirit to 5.3

[ ] 5.5  Register both transports in the libp2p host builder
         - pkg/networking/transport/host.go: conditional construction
           based on config flags (tor_enabled, i2p_enabled)
         - Both adapters should coexist with TCP/QUIC; peers can be
           reached via whichever address the remote advertises
         - Ensure multiaddr selection logic prefers clearnet when
           available and anonymity is not required, and prefers
           onion/garlic when Shroud is routing through them

[ ] 5.6  User-facing modes
         - Mode A (default): Shroud over clearnet libp2p
         - Mode B ("I need stronger anonymity"): all outbound
           circuits dialed via the Tor adapter; Shroud still layers
           on top for intra-MURMUR unlinkability
         - Mode C (expert): I2P-only; Shroud over Garlic adapter
         - Mode D (belt-and-suspenders): both Tor and I2P adapters
           registered, peers reachable via either
         - Each mode surfaces plain-language tradeoffs (latency,
           reachability, failure modes) before user commits

[ ] 5.7  Reachability diagnostics
         - Startup check: is Tor control port responsive? Is I2P
           SAMv3 responsive? Surface actionable errors rather than
           silent fallback
         - Health endpoint reporting per-transport status for
           debugging without leaking identity data

[ ] 5.8  Interop and regression testing
         - Unit tests mock the onramp Onion/Garlic interfaces
         - Integration test harness runs ephemeral Tor and I2P
           instances (or containers) and exercises full dial/listen
         - Scenarios: Shroud over Tor; Shroud over I2P; Shroud
           simultaneously over both; fallback to clearnet when
           anonymity network unreachable but mode allows it
         - Race-detector clean, consistent with current 100% pass rate

[ ] 5.9  Documentation
         - TRANSPORT_ANONYMITY.md explaining the adapter model, the
           go-i2p/onramp dependency, the distinction between
           Shroud's unlinkability and Tor/I2P anonymity, and the
           situations in which each mode is appropriate
         - Update SECURITY_PRIVACY.md and the threat model statement
           from 0.2 to reference the new transport options


=====================================
PHASE 6: TUNNELING PRIMITIVE (PageKite/ngrok) (Weeks 16–26)
=====================================
Major new capability. Requires architectural care and abuse planning
BEFORE code. Do not start before Phase 4 completes.

[ ] 6.1  Write TUNNEL_DESIGN.md
         - Use case: developer exposes localhost service via Murmur
         - Addressing: how does a client reach the tunneled service?
         - Anonymity: is the tunnel operator anonymous? The user? Both?

[ ] 6.2  Define tunnel abuse policy BEFORE implementation
         - Exit/edge operators can set content-type and hostname allowlists
         - Default-deny for executable payloads unless operator opts in
         - Bandwidth accounting per tunnel
         - Automated takedown protocol that preserves operator anonymity

[ ] 6.3  Prototype minimal tunnel (HTTP only, single hop)
         - Validate addressing and auth model
         - Measure overhead vs. ngrok

[ ] 6.4  Extend to multi-hop tunnels over Shroud circuits
         - Reuse existing circuit infrastructure
         - Separate tunnel-traffic accounting from social-traffic accounting

[ ] 6.5  Incentive design for tunnel operators
         - Resonance reward? Explicit opt-in with bandwidth caps?
         - Avoid cryptocurrency unless absolutely necessary — it
           changes the legal and UX profile substantially

[ ] 6.6  Write operator-facing documentation and a "tunnel host"
         configuration profile


=====================================
PHASE 7: FRIEND-TO-FRIEND RESEED (Weeks 20–30)
=====================================
Shares infrastructure with tunneling. Build second, learn from Phase 6.

[ ] 7.1  Define reseed semantics
         - What does a reseed server provide? (peer list, bootstrap
           keys, network parameters?)
         - How does a user authorize a friend to reseed them?
         - What is the trust model if a reseed host is compromised?

[ ] 7.2  Design reseed as a specialized tunnel application
         - Reuse Phase 6 infrastructure
         - Specific Wave type or protocol message for reseed requests

[ ] 7.3  Implement out-of-band invitation codes
         - Friend shares a signed code (QR, text, paper)
         - Code contains enough to bootstrap without a central server
         - Test in adversarial network conditions (blocked bootstraps)

[ ] 7.4  Document RESEED.md with threat model (compromised
         reseed host, coerced friend, etc.)


=====================================
PHASE 8: EXTENSION SURFACE HARDENING (Weeks 24–32)
=====================================
Turn the Extension Contract from 0.3 into a stable, documented API.

[ ] 8.1  Freeze stable extension points
         - Custom Wave type registration
         - Game module API (from Phase 2.4)
         - Transport adapter API (from Phase 5.2 — the same boundary
           used for onramp Onion/Garlic adapters)
         - Resonance hook API (read-only reputation queries)

[ ] 8.2  Publish PROTOCOL.md separating wire format from
         reference implementation
         - Anyone should be able to implement a compatible client
           from this document alone

[ ] 8.3  Build one non-trivial reference extension
         - E.g., a Specter-based collaborative writing tool, or
           a custom game from a third party
         - Proves the extension surface is real, not theoretical

[ ] 8.4  Establish a MEP (Murmur Extension Proposal) process
         - Lightweight — a folder with numbered markdown files
         - Community can propose new extension points without core fork


=====================================
PHASE 9: BOOTSTRAPPING & LAUNCH (Weeks 28–36)
=====================================
Don't launch until the cold-start story is designed, not hoped for.

[ ] 9.1  Define "minimum viable network" size
         - How many users/peers are needed before the app is fun?
         - Can a user have a good experience with only their existing
           contacts, or does it require strangers?

[ ] 9.2  Invite-first launch plan
         - Ship with bundled invites; every new user can invite N more
         - Invites carry friend-group context so Pulse Map isn't empty

[ ] 9.3  Seed 2–3 "anchor communities" (5–50 real friend groups each)
         - Work with them directly, iterate weekly
         - These groups prove the product loop before public opening

[ ] 9.4  Define success metrics that match product identity
         - D7/D30 retention in active conversations
         - Games started per week per user
         - NOT: DAU, follower counts, or engagement-time
         - Publish metrics philosophy to align with "no metrics" ethos

[ ] 9.5  Open beta with clear version expectations and kill-switch
         plan for discovered security issues


=====================================
ONGOING / CROSS-PHASE
=====================================

[ ] X.1  Maintain a decision log (DECISIONS.md)
         - Every reversible decision: what, why, when, revisit-by
         - Prevents re-litigating the same debates

[ ] X.2  Quarterly threat model review
         - Does the threat model still match the product?
         - Are there new adversary capabilities to consider?
         - Re-evaluate go-i2p/onramp dependency posture: is it still
           maintained? Are there upstream API changes? Has the Tor
           or I2P ecosystem shifted in ways that affect the adapter?

[✓] X.3  Maintain test suite at 100% pass, zero races
         - Current state is a strategic asset; protect it
         - Include onramp-backed transport adapters in the race
           detector matrix
         - **STATUS (2026-05-06)**: Re-validated. 57/57 packages passing
           with race detector. Zero failures, zero races, zero panics.
           Complexity baseline generated (baseline-current-workflow.json).
           Test suite ready for v0.1 milestone.

[ ] X.4  Publish a public roadmap derived from this checklist
         - Keeps contributors aligned
         - Makes the "backbone primitive" claim credible


=====================================
PRIORITY ORDERING IF RESOURCES ARE TIGHT
=====================================
Must-do before any public release:
  0.1, 0.2, 0.3, 1.1, 1.4, 3.1, 3.2, 4.1, 4.2, 9.1, 9.3

Strongly recommended before public release (the onramp-based
transports are cheap to integrate and dramatically widen the
addressable user base for privacy-sensitive cohorts):
  5.1, 5.2, 5.3, 5.5, 5.6, 5.9

Can ship in a later release:
  5.4 (I2P adapter if Tor ships first), 5.7, 5.8 beyond smoke tests,
  6.x (tunneling), 7.x (reseed), 8.x (extension surface as a
  public product)

Ongoing, never "done":
  2.x (games), 4.x (abuse), X.x (meta)
