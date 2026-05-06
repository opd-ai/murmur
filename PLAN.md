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

This checklist operationalizes that goal statement. Phases are
roughly sequential but overlap deliberately; priority ordering at
the end identifies which items are mandatory before any public
release versus which may ship later.


=====================================
PHASE 0: FOUNDATIONAL DECISIONS (Weeks 1–3)
=====================================
Lock in the strategic choices that shape everything downstream.
Do not skip — retrofitting these later is 10x more expensive.

[ ] 0.1  Write a one-page Product Identity Statement
         - Who is the target user? (e.g., "friend groups wanting
           private, playful communication without platform surveillance")
         - What is the core loop in 1 sentence?
         - What is explicitly NOT in scope? (e.g., public broadcast,
           influencer mechanics, algorithmic feed)

[ ] 0.2  Write a one-page Threat Model Statement
         - Primary adversary: network-level metadata observer
         - Secondary adversary: malicious peers / griefers
         - Explicitly NOT in scope: global passive adversary,
           state-level traffic analysis (defer to Tor/I2P bridging)
         - Document what Shroud guarantees vs. what Tor/I2P bridging
           guarantees — users must be able to choose correctly

[ ] 0.3  Define the Extension Contract v0
         - List every point where downstream networks can extend:
           custom Wave types, custom game modules, custom Resonance
           hooks, custom transports, custom UI overlays
         - Mark each as "stable", "experimental", or "private"
         - Document in EXTENSION_CONTRACT.md

[ ] 0.4  Decide Pulse Map's role: PRIMARY or SECONDARY surface
         - If PRIMARY: justify why graph-first beats chat-first for
           messaging+games; prototype a new-user path that doesn't
           require graph literacy
         - If SECONDARY: promote messaging/games surfaces to default;
           keep Pulse Map as "Explore" tab
         - Commit in writing; this decision gates UX work


=====================================
PHASE 1: UX REPOSITIONING (Weeks 3–8)
=====================================
Make the primary user path match the product identity.

[ ] 1.1  Map the 3 core user journeys end-to-end
         - New user → first conversation (target: < 90 seconds)
         - Existing user → play a game with a friend (target: < 3 taps)
         - Existing user → discover someone new (Pulse Map shines here)

[ ] 1.2  Prototype a messaging-first home surface
         - Conversations list as default view
         - Active games surfaced as persistent cards
         - Pulse Map accessible via dedicated navigation, not entry point

[ ] 1.3  A/B the two surfaces internally with 10+ testers
         - Measure time-to-first-message and time-to-first-game
         - Decide based on data, not aesthetics

[ ] 1.4  Design empty-state and low-population states for Pulse Map
         - What does a 5-node graph look like?
         - Never show a user an empty or near-empty graph as first impression

[ ] 1.5  Document the Pulse Map degradation curve
         - Behavior at 50 / 500 / 5,000 / 50,000 visible nodes
         - Define culling, clustering, and LOD strategies up front


=====================================
PHASE 2: GAME LIBRARY DIFFERENTIATION (Weeks 6–14)
=====================================
Games are the retention engine. Curate, don't accumulate.

[ ] 2.1  Classify existing 10 mini-games across 4 axes
         - Sync vs. async
         - 1:1 vs. group
         - Skill vs. chance vs. social
         - Anonymity leak surface (none / low / medium / high)

[ ] 2.2  Cut games with poor anonymity/fun ratio
         - Any real-time game with <200ms tolerance leaks latency
           fingerprints — evaluate whether the fun justifies it
         - Prefer async or turn-based for anonymous layer

[ ] 2.3  Identify 2–3 "signature" games that define the product
         - Must be: playable in <5 min, shareable via invite,
           tolerant of dropout, fun solo-with-a-stranger
         - These are your "Jackbox moment" — treat them as flagship

[ ] 2.4  Build a Game Module SDK
         - Stable API: create match, broadcast event, persist state,
           end match, award Resonance
         - Sandbox model: games cannot access identity, network, or
           storage directly — only through SDK primitives
         - Example game in repo as reference implementation

[ ] 2.5  Document game-specific anonymity implications
         - Per-game "privacy datasheet" listing what metadata the
           game inherently exposes
         - Surface relevant warnings to users before first play


=====================================
PHASE 3: IDENTITY RECOVERY & CONTINUITY (Weeks 8–14)
=====================================
In a messaging+games network, losing identity = losing friendships.
Treat recovery as a first-class feature, not a checkbox.

[ ] 3.1  Audit current BIP-39 recovery UX
         - Time-to-recover on new device
         - Failure modes when seed is partially remembered
         - What is preserved vs. lost on recovery?

[ ] 3.2  Design multi-device identity
         - One logical identity, multiple device keys
         - Device revocation path that doesn't require the lost device

[ ] 3.3  Design social recovery (Shamir or equivalent)
         - User designates N trusted contacts; M-of-N can co-sign recovery
         - Works for both Surface and Specter identities (separately)
         - Must not deanonymize Specter to recovery participants

[ ] 3.4  Design identity continuity across key rotation
         - A signed "continuity statement" from old key authorizes new key
         - Contacts verify continuity automatically
         - Prevents the "is this really you?" problem after rotation

[ ] 3.5  Write RECOVERY.md with user-facing flows and failure handling


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

[ ] X.3  Maintain test suite at 100% pass, zero races
         - Current state is a strategic asset; protect it
         - Include onramp-backed transport adapters in the race
           detector matrix

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
