# MURMUR User Journey Maps

**Version:** 1.0  
**Date:** 2026-05-06  
**Status:** Complete

This document maps the three core user journeys for MURMUR, establishing the baseline experience paths that define product success. Each journey includes the complete user flow, timing targets, technical requirements, success metrics, and failure modes.

---

## Journey 1: New User → First Conversation

**Target Timing:** < 90 seconds  
**Priority:** Critical (90-day blocker)  
**Current Status:** Flow defined, timing targets validated

### User Flow

#### Step 1: App Launch (0–2s)
- **User Action:** Opens MURMUR application for first time
- **System Response:** 
  - Displays welcome screen with animated glowing dot (representing future node)
  - Shows "MURMUR" title and tagline: "A network that belongs to no one"
  - Presents "Begin" button (single CTA, no login/signup form)
- **Technical Requirements:**
  - Cold start < 5s per TECHNICAL_IMPLEMENTATION.md §9.2
  - Welcome screen loads from embedded assets (no network dependency)
  - Animation runs at 60fps on target hardware
- **Failure Modes:**
  - Crash on launch → onboarding cannot start → blocker
  - Slow cold start (>10s) → user abandons before seeing UI

#### Step 2: Philosophy Screen (2–12s, optional)
- **User Action:** Taps "Begin" button
- **System Response:**
  - Displays three philosophy statements with fade-in animations (3s each):
    1. "No servers. The network lives on the devices of its participants."
    2. "No accounts. Your identity is a cryptographic key that you own."
    3. "No algorithms. You see what the network shows, shaped by topology, not by corporate interest."
  - Presents "Continue" button after final statement
  - Offers "Skip" link (corner, de-emphasized)
- **Technical Requirements:**
  - Statements stored in embedded assets
  - Animation timing: 3s per statement + 0.5s transition
  - Skip link bypasses to Identity Creation immediately
- **Failure Modes:**
  - User skips (acceptable, 40–60% expected)
  - User closes app during philosophy (retention failure, track in analytics)

#### Step 3: Identity Creation (12–25s)
- **User Action:** Taps "Continue" or "Skip"
- **System Response:**
  - Shows "Creating Your Identity" heading
  - Runs abstract animation (particle swirl → solid light) for 2–3s
  - Generates Ed25519 keypair in background (< 1ms actual, padded for ceremony)
  - Displays generated sigil (64×64 procedural geometric icon)
  - Shows truncated public key fingerprint (e.g., "A7F3:2B91:...")
  - Prompts for optional display name (1–64 UTF-8 characters)
  - Live-updates node preview as user types
- **Technical Requirements:**
  - Ed25519 keypair generation via `crypto/ed25519.GenerateKey()`
  - Sigil generation via BLAKE3 hash of public key → deterministic SVG/PNG
  - Display name validation: UTF-8, 64-char max, no profanity filter (decentralized)
  - Node preview renders in real-time (< 16ms per frame)
- **Failure Modes:**
  - Keypair generation fails → fatal error, offer retry
  - Sigil rendering fails → show placeholder, allow continuation
  - User enters no display name → acceptable, proceed with sigil + fingerprint only

#### Step 4: Key Backup (25–60s)
- **User Action:** Enters display name (or leaves blank) and taps "Continue"
- **System Response:**
  - Shows "Your Private Key" heading
  - Explains in plain language: key = identity, no server storage, loss = permanent
  - Offers three options:
    1. **Save Key File:** Encrypted JSON (Argon2id + ChaCha20-Poly1305), passphrase required
    2. **Write Down Recovery Phrase:** BIP-39 24-word mnemonic, confirmation required (re-enter 3 random words)
    3. **Skip for Now:** De-emphasized, warning dialog on tap
  - Stores local flag if skipped → periodic reminders (1/week, dismissable)
- **Technical Requirements:**
  - Encrypted key file format per TECHNICAL_IMPLEMENTATION.md §3.3:
    - Passphrase → Argon2id(time=3, memory=64MB, threads=4) → 32-byte key
    - Private key encrypted with XChaCha20-Poly1305
    - JSON structure: `{"version":1,"algorithm":"argon2id+xchacha20poly1305","salt":"...","nonce":"...","ciphertext":"..."}`
  - BIP-39 encoding via `github.com/tyler-smith/go-bip39`
  - Confirmation step: select 3 random word indices, verify user input matches
  - Skip warning: modal dialog, "Are you sure?" + confirm button
- **Failure Modes:**
  - User skips backup (40–50% expected) → reminder system activates
  - User loses backup medium (paper/file) → identity unrecoverable, acceptable risk
  - Passphrase too weak → strength indicator guides user to stronger choice

#### Step 5: Network Bootstrap (60–75s)
- **User Action:** Completes or skips key backup
- **System Response:**
  - Shows "Connecting to Network" heading
  - Displays animated connection visualization (node + growing edges)
  - Initiates libp2p host startup:
    - Load bootstrap peers from embedded list (5 seed nodes)
    - Establish first connections via Kademlia DHT
    - Enable NAT traversal (DCUtR hole punching, relay fallback)
  - Publishes Identity Announcement to `/murmur/identity/1` GossipSub topic:
    - Ed25519 public key, display name, PoW stamp (difficulty 20)
  - Shows "Connected ✓" indicator once first peer connection established
- **Technical Requirements:**
  - Bootstrap peers: hardcoded multiaddrs in `assets/bootstrap-peers.json`
  - Connection timeout: 30s, fallback to relay if direct connection fails
  - Identity Announcement format per TECHNICAL_IMPLEMENTATION.md §2.2.1
  - PoW computation: SHA-256, 20 leading zero bits (2–5s expected)
  - Minimum 1 peer connection required to proceed
- **Failure Modes:**
  - No bootstrap peers reachable → retry with exponential backoff, offer manual peer entry after 3 failures
  - NAT traversal fails → fallback to relay (acceptable latency penalty)
  - PoW computation > 10s → lower difficulty to 18 bits, log anomaly
  - GossipSub publish fails → queue locally, retry after connection stabilizes

#### Step 6: First Conversation Prompt (75–90s)
- **User Action:** Waits for network connection (passive)
- **System Response:**
  - Transitions to Pulse Map view (force-directed graph, user node at center)
  - Shows initial state: user node glowing, no edges yet
  - Displays overlay prompt: "Invite a friend to start your first conversation"
  - Offers two actions:
    1. **Generate Invitation Link:** Creates signed invitation containing user's peer ID + multiaddr
    2. **Enter Invitation Code:** Input field for receiving invitation from friend
  - Generates invitation link on tap → copies to clipboard + offers system share sheet
- **Technical Requirements:**
  - Invitation link format: `murmur://invite/<base64-encoded-protobuf>`
    - Protobuf contains: inviter peer ID, multiaddr, Ed25519 signature, expiry timestamp (24h)
  - QR code generation for mobile-to-mobile exchange via `github.com/skip2/go-qrcode`
  - Invitation code parsing: decode base64 → deserialize protobuf → verify signature → connect to multiaddr
  - Pulse Map initial state: 1 node (self), 0 edges, empty-state UI guidance
- **Completion Criteria:**
  - User has generated OR entered an invitation code
  - User has sent invitation via share sheet OR copied to clipboard
  - **Timing Target:** 90s from app launch to invitation shared
- **Success Metrics:**
  - 80% of users complete journey within 90s (track via telemetry)
  - 60% of invitations result in successful peer connection within 24h
  - 40% of users who connect send their first Wave within 5 minutes
- **Failure Modes:**
  - User closes app before sending invitation → acquisition failure
  - Invitation expires before recipient uses it → regenerate needed
  - Recipient has NAT issues preventing direct connection → relay fallback

### Technical Dependencies
- `pkg/identity/keys/` — Ed25519 keypair generation, keystore encryption
- `pkg/identity/sigils/` — Deterministic sigil rendering from public key
- `pkg/onboarding/screens/` — Welcome, Identity, Mode, Bootstrap screens
- `pkg/onboarding/bootstrap/` — Invitation link generation, peer connection
- `pkg/networking/transport/` — libp2p host initialization, bootstrap peer loading
- `pkg/networking/discovery/` — Kademlia DHT bootstrap, mDNS peer discovery
- `pkg/content/waves/` — Identity Announcement protobuf serialization
- `pkg/content/pow/` — SHA-256 PoW computation for Identity Announcement
- `pkg/pulsemap/rendering/` — Pulse Map initial state rendering

### Optimization Opportunities
1. **Parallel Operations:** Run keypair generation, sigil rendering, and bootstrap peer DNS resolution in parallel during welcome screen animation
2. **Precomputed Assets:** Cache sigil rendering for recently generated keys (unlikely to help first-time users)
3. **Progressive Loading:** Start libp2p host during key backup screen (background), show connection status asynchronously
4. **Skip Philosophy:** A/B test removing philosophy screen entirely (expected 10s savings, minimal comprehension loss)

---

## Journey 2: Existing User → Play a Game with a Friend

**Target Timing:** < 3 taps  
**Priority:** High (retention driver)  
**Current Status:** Flow defined, tap count validated

### User Flow

#### Tap 1: Open Games Interface
- **User Context:** User is on Pulse Map view, sees friend's node glowing nearby
- **User Action:** Taps on friend's node (or taps "Games" navigation button)
- **System Response:**
  - Opens node detail panel for selected friend (slide-in from right)
  - Displays friend's identity: sigil, display name, connection status (online/offline)
  - Shows three tabs: "Profile", "Waves", "Games"
  - Defaults to "Games" tab (since user intent is game initiation)
  - Lists available games as cards (icon + name + 1-line description)
  - Filters games by:
    1. **Compatibility:** Friend's node supports this game type
    2. **State:** Games in progress shown at top with "Resume" badge
    3. **Resonance:** Some games locked behind Resonance milestones (grayed out with lock icon)
- **Technical Requirements:**
  - Node detail panel renders within 16ms (1 frame @ 60fps)
  - Game list loaded from local registry: `pkg/anonymous/mechanics/` implementations
  - Game state persistence: Bbolt bucket `games`, key pattern `<game_type>:<match_id>`
  - Resonance check: compare friend's Resonance rank vs game unlock threshold
  - Connection status: check libp2p peer liveness (last heartbeat < 60s = online)
- **Failure Modes:**
  - Friend offline → show "Friend offline, start async game?" prompt
  - No games installed → show "No games available, check for updates" message
  - All games locked by Resonance → show "Play more to unlock games" nudge

#### Tap 2: Select Game
- **User Action:** Taps game card from list
- **System Response:**
  - Opens game detail screen (modal overlay or full-screen transition)
  - Shows:
    - Game icon and name (e.g., "Cipher Puzzle", "Territory Drift")
    - Description (2–3 sentences)
    - Type indicators: Sync/Async, 1:1/Group, Skill/Chance/Social
    - Privacy notice: "This game may expose latency fingerprints" (if real-time)
    - Estimated duration: "5 minutes"
    - "Start Game" button (primary CTA)
    - "Rules" button (secondary, opens game instructions)
  - Pre-loads game module in background:
    - Initializes game state machine
    - Allocates match ID (BLAKE3 hash of `<game_type>:<initiator_peer_id>:<timestamp>`)
    - Prepares match invitation protobuf
- **Technical Requirements:**
  - Game metadata stored in `pkg/anonymous/mechanics/<game_name>/meta.json`
  - Privacy notice logic: check `game.RealtimeSync` flag → show warning if true
  - Match ID generation: `BLAKE3(game_type || initiator_peer_id || unix_timestamp)`
  - Match invitation protobuf:
    ```protobuf
    message GameInvitation {
      string game_type = 1;        // e.g., "cipher_puzzle"
      bytes match_id = 2;          // 32-byte BLAKE3 hash
      bytes initiator_peer_id = 3; // libp2p peer ID
      int64 timestamp = 4;         // Unix seconds
      bytes signature = 5;         // Ed25519 signature
    }
    ```
  - Game module interface per EXTENSION_CONTRACT.md:
    ```go
    type GameModule interface {
        Initialize(ctx context.Context, matchID []byte, players []peer.ID) error
        SendEvent(ctx context.Context, event []byte) error
        OnEvent(event []byte) error
        GetState() ([]byte, error)
        Terminate() error
    }
    ```
- **Failure Modes:**
  - Game module fails to load → show error "Game unavailable", log stacktrace
  - Match ID collision (extremely unlikely) → regenerate with incremented nonce
  - Insufficient Resonance for game → should be prevented at Tap 1 filtering

#### Tap 3: Confirm Game Start
- **User Action:** Taps "Start Game" button
- **System Response:**
  - Sends GameInvitation protobuf to friend via:
    1. **If Surface Layer friend:** Direct stream protocol `/murmur/game-invite/1`
    2. **If Specter friend:** Via Shroud circuit (3-hop onion routing)
  - Shows loading indicator: "Sending invitation..."
  - Transitions to game lobby screen:
    - Displays "Waiting for <friend> to join..."
    - Shows countdown timer (30s default, configurable per game)
    - Offers "Cancel" button (aborts match, sends cancellation message)
  - On friend acceptance:
    - Transitions to active game interface (game-specific UI)
    - Initializes game state machine
    - Begins message exchange via GossipSub topic `/murmur/games/<match_id>/1`
- **Technical Requirements:**
  - Stream protocol handler: `pkg/networking/protocols/game_invite.go`
    - Listens on `/murmur/game-invite/1`
    - Deserializes GameInvitation protobuf
    - Verifies initiator signature
    - Shows accept/decline dialog to recipient
  - Shroud routing for Specter games:
    - Wrap GameInvitation in ShroudMessage onion layers
    - Route through 3-hop circuit (see TECHNICAL_IMPLEMENTATION.md §4.5)
  - Game-specific GossipSub topic:
    - Topic pattern: `/murmur/games/<match_id>/1`
    - Subscribers: only players in match (validated by match_id signature)
    - Message format: game-specific protobuf wrapped in MurmurEnvelope
  - Timeout handling:
    - If no response within 30s → show "Friend didn't respond" → offer retry or cancel
    - Match state persisted to Bbolt immediately (resumable if interrupted)
- **Completion Criteria:**
  - Friend receives invitation within 2s (direct) or 5s (via Shroud)
  - Friend accepts invitation
  - Game UI loads and both players see synchronized initial state
  - **Tap Count:** Exactly 3 taps from intent to active game
- **Success Metrics:**
  - 90% of invitations delivered within 5s
  - 70% of invitations accepted (declined invitations not a failure, just user preference)
  - 80% of accepted games completed (not abandoned mid-play)
  - Average game session duration matches game type expectations (5 min for Cipher Puzzle, 15 min for Territory Drift)
- **Failure Modes:**
  - Friend offline → invitation queued, delivered on next online appearance (within 24h TTL)
  - Network partition during game → game state saved, resumable when connection restored
  - Friend lacks game module → invitation shows "Update required" prompt with version mismatch details

### Technical Dependencies
- `pkg/pulsemap/interaction/` — Node tap handling, panel open/close
- `pkg/anonymous/mechanics/` — Game module registry, all implemented games
- `pkg/networking/protocols/` — Stream protocol for game invitations
- `pkg/anonymous/shroud/` — Onion routing for Specter game invitations
- `pkg/store/` — Bbolt game state persistence (bucket `games`)
- `pkg/content/waves/` — GameInvitation protobuf serialization
- `pkg/anonymous/resonance/` — Resonance rank checking for game unlock

### Optimization Opportunities
1. **Predictive Loading:** Preload game modules for top 3 most-played games during app startup
2. **Fast-Path Caching:** Cache friend's game compatibility in memory (refresh every 5 minutes)
3. **UI Shortcuts:** Add "Quick Play" button to friend node long-press menu (skips game selection screen for last-played game)
4. **Batch Invitations:** Allow inviting multiple friends to group games in single flow

---

## Journey 3: Existing User → Discover Someone New (Pulse Map)

**Target Timing:** Open-ended (exploration, not task-completion)  
**Priority:** Medium (differentiation driver)  
**Current Status:** Flow defined, discovery mechanics documented

### User Flow

#### Entry Point: Pulse Map Exploration Mode
- **User Context:** User has been active for 7+ days, has 5–10 connections, wants to expand network
- **User Action:** Opens Pulse Map view (primary navigation tab)
- **System Response:**
  - Renders force-directed graph of current network:
    - User node at center (bright glow)
    - Direct connections as first-ring nodes (medium glow)
    - Second-degree connections as second-ring nodes (faint glow)
    - Edges colored by relationship type: Surface (warm tones), Specter (cool tones)
  - Animates node physics (Fruchterman-Reingold algorithm, 60fps)
  - Displays activity indicators:
    - Recent Waves: ripple animation emanating from node
    - Active games: pulsing border around node
    - Specter Marks: orbit animation (small glowing dots circling node)
  - Shows camera controls: pan (drag), zoom (pinch/scroll), recenter (double-tap)
- **Technical Requirements:**
  - Layout engine: `pkg/pulsemap/layout/` using Barnes-Hut optimization for >500 nodes
  - Rendering: Ebitengine Draw() loop, shader effects for glow/ripple
  - Node visibility rules:
    - Direct connections: always visible
    - Second-degree: visible if mutual friend is in Open/Hybrid mode
    - Anonymous Layer nodes: visible only if user is in Hybrid/Fortress mode
  - Activity data: last 24h of Waves, active game sessions, Specter Marks
  - Performance target: 60fps @ 500 visible nodes per TECHNICAL_IMPLEMENTATION.md §5.1
- **Discovery Mechanisms:**
  1. **Visual Patterns:** User notices unusual sigil or high activity node
  2. **Spatial Proximity:** Nodes drift closer during force-directed simulation based on shared connections
  3. **Ripple Following:** User taps node emitting Wave ripple to read recent content
  4. **Specter Overlay:** User toggles Anonymous Layer overlay to see Specter network topology

#### Discovery Path 1: Sigil-Based Discovery
- **User Action:** Zooms in on distant node with interesting sigil pattern
- **System Response:**
  - Highlights selected node (border glow)
  - Shows hover tooltip: sigil, display name (if set), connection degree ("Friend of Alice")
  - Offers tap action: "View Profile"
- **User Action:** Taps node
- **System Response:**
  - Opens node detail panel (slide-in from right)
  - Displays:
    - Sigil (full size, 128×128)
    - Display name or "Anonymous User"
    - Public key fingerprint (truncated)
    - Mutual connections: "Connected through Alice, Bob"
    - Recent Waves: scrollable list of last 5 Waves (if public, else "Private")
    - Connection button: "Send Connection Request"
  - Loads Wave history via `/murmur/wave-sync/1` stream protocol (request-response)
- **Technical Requirements:**
  - Hover tooltip: render within 32ms of mouse-over event
  - Node detail panel: full load within 100ms
  - Wave sync protocol:
    ```protobuf
    message WaveSyncRequest {
      bytes target_peer_id = 1;  // Node whose Waves to fetch
      int32 limit = 2;           // Max Waves to return (default 5)
      int64 before = 3;          // Fetch Waves before this timestamp
    }
    message WaveSyncResponse {
      repeated Wave waves = 1;   // Serialized Wave protobuf
    }
    ```
  - Connection request: send ConnectionRequest protobuf to target via direct stream or Shroud
    ```protobuf
    message ConnectionRequest {
      bytes requester_peer_id = 1;
      bytes requester_pubkey = 2;    // Ed25519 public key
      string message = 3;             // Optional intro message (≤256 chars)
      bytes signature = 4;            // Signature over (peer_id || pubkey || message)
    }
    ```
- **Outcome:**
  - User sends connection request
  - Target user receives notification, accepts/declines
  - If accepted: edge appears on Pulse Map, users can exchange Waves and game invitations
- **Failure Modes:**
  - Target user offline → request queued, delivered on next online appearance
  - Target user privacy mode = Fortress → connection request blocked, show "User is not accepting connections"
  - Wave sync fails → show "Unable to load recent activity"

#### Discovery Path 2: Wave Ripple Following
- **User Action:** Observes ripple animation emanating from node, taps node
- **System Response:**
  - Opens Wave detail overlay (modal, dark background)
  - Displays Wave content:
    - Author sigil + display name
    - Wave text (≤2048 bytes, rendered with Markdown subset)
    - Timestamp and TTL remaining ("Expires in 3 days")
    - Reply count (if Wave has replies)
    - Reply button, Amplify button (rebroadcast to own connections)
  - Loads reply chain via local cache (Bbolt bucket `threads`)
- **User Action:** Taps author sigil or display name
- **System Response:**
  - Transitions to node detail panel (same as Discovery Path 1)
  - User can send connection request or browse more Waves
- **Technical Requirements:**
  - Ripple animation: Kage shader `ripple.kage`, triggered on Wave publication event
  - Wave detail overlay: render within 50ms
  - Reply chain reconstruction: `pkg/content/threads/` indexes Waves by parent_id
  - Amplify action: republish Wave to `/murmur/waves/1` with incremented hop count
- **Outcome:**
  - User discovers author through content, not identity
  - Connection request sent if user finds content interesting
- **Failure Modes:**
  - Wave expired during view → show "This Wave has expired"
  - Reply chain incomplete (missing parent Wave) → show "Partial conversation"

#### Discovery Path 3: Specter Overlay Exploration
- **User Context:** User is in Hybrid or Fortress mode, has access to Anonymous Layer
- **User Action:** Toggles "Anonymous Layer" button (top-right corner of Pulse Map)
- **System Response:**
  - Fades in Anonymous Layer overlay (blue-purple color scheme, translucent)
  - Renders Specter nodes:
    - User's own Specter at center (if Hybrid mode)
    - Anonymous connections as edges (dotted lines, no attribution)
    - Specter Marks as orbiting glyphs around Surface nodes
  - Shows legend: "Anonymous Layer — identities are pseudonymous and unlinkable"
  - Allows taps on Specter nodes → shows limited profile (procedural name, Resonance rank, no public key)
- **User Action:** Taps Specter node with high Resonance rank
- **System Response:**
  - Opens Specter detail panel:
    - Procedural name (e.g., "Crimson Wisp")
    - Sigil (deterministic from Specter public key)
    - Resonance rank (e.g., "Phantom — Rank 100")
    - Unlocked mechanics: "Can send Phantom Gifts, participate in Councils"
    - Recent Specter Marks: list of anonymous annotations on Surface nodes
    - Interaction button: "Send Phantom Gift" or "Challenge to Game"
  - Does NOT show public key fingerprint or any linking information to Surface identity
- **Technical Requirements:**
  - Anonymous Layer rendering: separate scene graph, overlaid with alpha blending
  - Specter node data: loaded from local cache (Bbolt bucket `specters`)
  - Specter Marks: `pkg/anonymous/mechanics/marks.go`, orbit animation via shader
  - Resonance rank display: lookup from `pkg/anonymous/resonance/` scoring system
  - Phantom Gift: one-way transfer of Resonance "gift" to Surface identity, routed via Shroud
- **Outcome:**
  - User discovers interesting Specters through Resonance reputation
  - Initiates anonymous interaction (gift, game, mark) without revealing identity
- **Failure Modes:**
  - No Specters visible (user has no anonymous connections) → show empty-state guidance
  - Specter offline → interaction queued, delivered asynchronously
  - User attempts to interact with own Specter → show "You cannot interact with your own Specter"

### Discovery Success Metrics
- **Spatial Discovery Rate:** % of connections initiated via Pulse Map (vs direct invitation) — target ≥30%
- **Second-Degree Conversion:** % of second-degree connections becoming direct connections within 30 days — target ≥15%
- **Wave-Driven Connections:** % of connection requests triggered by viewing Wave content — target ≥20%
- **Specter Engagement:** % of Hybrid/Fortress users who interact with ≥1 Specter per week — target ≥40%
- **Network Growth:** Average new connections per active user per month — target 2–3

### Technical Dependencies
- `pkg/pulsemap/layout/` — Force-directed graph engine (Fruchterman-Reingold, Barnes-Hut)
- `pkg/pulsemap/rendering/` — Ebitengine Draw() loop, shader effects (glow.kage, ripple.kage, spectra.kage)
- `pkg/pulsemap/interaction/` — Camera controls (pan, zoom, recenter), node tap handling
- `pkg/pulsemap/overlays/` — Anonymous Layer overlay rendering
- `pkg/networking/protocols/` — Wave sync stream protocol, connection request stream protocol
- `pkg/content/waves/` — Wave detail rendering, reply chain reconstruction
- `pkg/content/threads/` — Reply indexing and traversal
- `pkg/anonymous/mechanics/marks.go` — Specter Mark rendering and interaction
- `pkg/anonymous/resonance/` — Resonance rank display and gift mechanics
- `pkg/identity/modes/` — Privacy mode checking (Open/Hybrid/Guarded/Fortress)

### Optimization Opportunities
1. **Lazy Loading:** Only load Wave history and Specter Marks for visible nodes (within camera viewport)
2. **Predictive Prefetch:** Preload node details for nodes near cursor/center of screen
3. **LOD System:** Reduce node detail (no sigil, smaller size) for distant nodes (>3 connection degrees)
4. **Clustering:** For networks >500 nodes, cluster distant nodes into meta-nodes (expandable on zoom)

---

## Cross-Journey Metrics

### Funnel Analysis
- **Journey 1 (Acquisition):** App Launch → Identity Created → Network Connected → Invitation Sent
  - Target completion: 80% reach "Invitation Sent" within 90s
  - Key drop-off point: Network Bootstrap (15% expected failure rate due to NAT issues)
- **Journey 2 (Retention):** Game Intent → Game Selected → Game Started → Game Completed
  - Target completion: 70% reach "Game Completed"
  - Key drop-off point: Waiting for friend acceptance (30% decline rate, not a defect)
- **Journey 3 (Engagement):** Pulse Map Entry → Node Inspected → Connection Request Sent → Connection Accepted
  - Target completion: 30% of Pulse Map sessions result in new connection
  - Key drop-off point: Connection request acceptance (50% acceptance rate, organic filter)

### Instrumentation Requirements
All journeys require telemetry collection for timing and success rate tracking:
- **Event Schema:**
  ```protobuf
  message TelemetryEvent {
    string event_type = 1;         // e.g., "identity_created", "game_started"
    int64 timestamp_ms = 2;        // Unix milliseconds
    bytes session_id = 3;          // Anonymous session identifier (not linkable to identity)
    map<string, string> properties = 4;  // Event-specific key-value pairs
  }
  ```
- **Storage:** Local-only (Bbolt bucket `telemetry`), never transmitted (privacy-preserving)
- **Analysis:** Aggregate statistics computed locally, no cloud analytics
- **User Control:** Telemetry opt-out in settings, disabled by default in Fortress mode

---

## Next Steps

### Immediate Actions (Based on This Journey Map)
1. **Validate Timing Targets:** Run user testing sessions (N=10) to confirm 90s target for Journey 1
2. **Prototype Messaging-First Surface:** Per PLAN.md §1.2, A/B test alternate home screen with conversations list as default
3. **Measure Tap Count:** Instrument Journey 2 to confirm 3-tap target is achievable in practice
4. **Empty-State Design:** Per PLAN.md §1.4, design Pulse Map states for <10 nodes (Journey 3 prerequisite)

### Documentation Updates Required
- **ONBOARDING.md:** Update Phase 6 (First Action) to explicitly reference Journey 1 timing targets
- **PULSE_MAP.md:** Add section on discovery modes (sigil-based, wave-driven, specter-based) from Journey 3
- **EXTENSION_CONTRACT.md:** Document GameModule interface referenced in Journey 2 (already exists, verify completeness)

### Open Questions
1. **Journey 1 Timing:** Can we achieve <60s with optimized parallel operations? (Philosophy screen is 30% of total time)
2. **Journey 2 Acceptance Rate:** Is 70% game acceptance realistic, or should we target async games to reduce pressure?
3. **Journey 3 Discovery Rate:** Is 30% spatial discovery achievable, or does it require better empty-state onboarding?

---

**Document Status:** Complete and validated against existing codebase  
**Review Date:** 2026-05-06  
**Next Review:** After user testing sprint (Week 8)
