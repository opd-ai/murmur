# Messaging-First Home Surface Prototype

**Version:** 1.0  
**Date:** 2026-05-06  
**Status:** Design Specification (Pre-Implementation)  
**Purpose:** A/B Testing Alternative to Pulse Map Primary Surface

---

## Executive Summary

This document specifies a **messaging-first home surface** prototype for MURMUR as an alternative default view to the Pulse Map. The prototype prioritizes conversations and active games as the entry point, relegating the Pulse Map to a dedicated navigation tab. This design addresses the concern raised in PLAN.md §1.2: whether a chat-first interface better serves messaging+games users than the current graph-first approach.

**Key Design Principle:** Minimize friction for the core loop (message exchange and game initiation) while preserving the Pulse Map as a discovery and exploration tool.

---

## Design Rationale

### Problem Statement

Per PULSE_MAP_ROLE_DECISION.md, MURMUR commits to the Pulse Map as PRIMARY surface, but PLAN.md §1.2 requires prototyping a messaging-first alternative for A/B testing. The rationale:

1. **User Literacy Barrier:** Force-directed graphs require spatial thinking; users from chat-first apps (Discord, Telegram, Signal) may struggle with the paradigm shift
2. **Retention Risk:** If users cannot quickly reach their core intent (message/game), they abandon during onboarding
3. **Content First-Use:** New users with empty networks see an empty Pulse Map (1 node, 0 edges) — a poor first impression
4. **Competitive Positioning:** Most privacy-focused social apps (Signal, Session, Briar) use conversation lists as primary UI

### Success Criteria

The messaging-first prototype succeeds if:
- **Time-to-First-Message** ≤ 60s (vs Pulse Map's 90s target per USER_JOURNEYS.md)
- **Time-to-First-Game** ≤ 20s from conversation view (vs Pulse Map's 3-tap target)
- **D7 Retention** ≥ 40% (vs baseline to be measured with Pulse Map first)
- **User Satisfaction** (post-session survey): "Easy to find conversations" ≥ 80% agreement

---

## UI Layout

### Screen Structure

```
┌──────────────────────────────────────────────────┐
│  [◀ Settings]    MURMUR         [⊕ New Message]  │ ← Top Bar
├──────────────────────────────────────────────────┤
│                                                  │
│  ┌────────────────────────────────────────┐     │
│  │  🎮 Active Games               [See All]│     │ ← Games Section
│  ├────────────────────────────────────────┤     │
│  │  ┌─────────────────────────────────┐   │     │
│  │  │ 🧩 Cipher Puzzle                │   │     │
│  │  │ with Alice, Bob                 │   │     │
│  │  │ 🟢 Your turn                     │   │     │
│  │  └─────────────────────────────────┘   │     │
│  │  ┌─────────────────────────────────┐   │     │
│  │  │ 🎯 Territory Drift              │   │     │
│  │  │ with Crimson Wisp (Specter)     │   │     │
│  │  │ 🔵 Waiting                       │   │     │
│  │  └─────────────────────────────────┘   │     │
│  └────────────────────────────────────────┘     │
│                                                  │
│  ┌────────────────────────────────────────┐     │
│  │  💬 Conversations          [Filter ▾]   │     │ ← Conversations Section
│  ├────────────────────────────────────────┤     │
│  │  ┌─────────────────────────────────┐   │     │
│  │  │ 👤 Alice                        │   │     │
│  │  │ Just got your invitation!      │   │     │
│  │  │ 🕐 2 min ago            [3 new] │   │     │
│  │  └─────────────────────────────────┘   │     │
│  │  ┌─────────────────────────────────┐   │     │
│  │  │ 👤 Bob                          │   │     │
│  │  │ Want to play a game?           │   │     │
│  │  │ 🕐 15 min ago                   │   │     │
│  │  └─────────────────────────────────┘   │     │
│  │  ┌─────────────────────────────────┐   │     │
│  │  │ 👻 Crimson Wisp (Specter)       │   │     │
│  │  │ [Anonymous message preview]    │   │     │
│  │  │ 🕐 1 hour ago                   │   │     │
│  │  └─────────────────────────────────┘   │     │
│  └────────────────────────────────────────┘     │
│                                                  │
├──────────────────────────────────────────────────┤
│  [💬 Messages] [🗺️ Map] [👻 Specters] [⚙️ More] │ ← Bottom Navigation
└──────────────────────────────────────────────────┘
```

### Components

#### 1. Top Bar
- **Left:** Settings icon (hamburger menu → privacy mode, backup, about)
- **Center:** "MURMUR" wordmark (tappable → refresh/home)
- **Right:** "New Message" button (⊕ icon) → opens contact picker

#### 2. Active Games Section
- **Heading:** "Active Games" with "See All" link (if >2 games)
- **Cards:** Horizontal scrollable list (max 3 visible)
  - Game icon + name
  - Participant list (display names or "Anonymous")
  - Status indicator:
    - 🟢 "Your turn" (requires action)
    - 🔵 "Waiting" (opponent's turn)
    - 🟡 "Starting" (lobby, waiting for accept)
    - 🔴 "Paused" (opponent offline >24h)
  - Tap → opens game interface directly
- **Empty State:** "No active games. [Start a game with a friend]" → opens game browser
- **Collapse:** Swipe down to minimize (shows "2 active games" compact header)

#### 3. Conversations Section
- **Heading:** "Conversations" with filter dropdown
  - All (default)
  - Surface Only
  - Specters Only
  - Unread
- **List Items:** Vertical scrollable list
  - Avatar/Sigil (left) — 48×48px
  - Display name or "Anonymous"
  - Last message preview (1 line, truncated)
  - Timestamp (relative: "2 min ago", absolute after 24h)
  - Unread badge (right) — red circle with count
  - Online status dot (green for online, gray for offline)
  - Tap → opens conversation detail view
- **Sorting:** Last activity (most recent first)
- **Empty State:** "No conversations yet. [Invite a friend]" → opens invitation flow
- **Long-Press Actions:**
  - Mark as read/unread
  - Pin to top
  - Delete conversation (local only, Waves remain in cache)

#### 4. Bottom Navigation
- **Messages** (active by default) — icon: 💬 speech bubble
- **Map** — icon: 🗺️ globe → opens Pulse Map view
- **Specters** — icon: 👻 ghost → filtered view (Hybrid/Fortress only, hidden in Open mode)
- **More** — icon: ⚙️ gear → settings, profile, help

---

## User Flows

### Flow 1: First-Time User (Post-Onboarding)

**Context:** User completed onboarding (identity creation, backup, network bootstrap) and invited a friend

1. **Entry Point:** App opens to messaging-first home surface
2. **Initial State:**
   - Active Games: empty state ("No active games")
   - Conversations: 1 item — "System" conversation with welcome message
     - "Welcome to MURMUR! Your invitation was sent. When your friend joins, their conversation will appear here."
     - System conversation is special: no replies, dismissable
3. **User Action:** Waits for friend to join
4. **Friend Joins:** New conversation appears at top of list
   - "🟢 Alice just joined!" (automatic system message in conversation)
   - Unread badge shows "1 new"
5. **User Action:** Taps conversation → opens conversation detail view
6. **Time to First Message:** ~60s (vs 90s with Pulse Map onboarding)

### Flow 2: Existing User → Send Message

1. **Entry Point:** App opens to messaging-first home surface
2. **Initial State:** 5 conversations visible, 2 active games
3. **User Action:** Taps conversation with "Alice"
4. **Transition:** Slide-in from right, conversation detail view
   - Shows message history (last 50 messages, scrollable)
   - Input field at bottom with "Send" button
   - Header shows Alice's sigil, display name, online status
5. **User Action:** Types message, taps "Send"
6. **System Response:**
   - Message appears immediately in conversation (optimistic UI)
   - PoW computation starts in background (2–5s per WAVES.md)
   - Spinner indicator next to message while PoW runs
   - When PoW completes, message publishes to GossipSub `/murmur/waves/1`
   - Checkmark appears when published
7. **Time to Message Sent:** ~10s (5s typing + 5s PoW average)

### Flow 3: Existing User → Start Game

**From Conversation View:**
1. **Entry Point:** User in conversation detail with "Bob"
2. **User Action:** Taps game controller icon in top-right header
3. **System Response:** Opens game browser modal (overlay, not full-screen)
   - Lists available games (same as USER_JOURNEYS.md Journey 2)
   - Filters by Bob's node compatibility
4. **User Action:** Taps "Cipher Puzzle" card
5. **System Response:** Shows game detail modal
   - Description, duration, privacy notice
   - "Start Game" button
6. **User Action:** Taps "Start Game"
7. **System Response:**
   - Sends GameInvitation via direct stream protocol
   - Closes modals, adds game card to Active Games section
   - Shows notification: "Game invitation sent to Bob"
8. **Bob Accepts:** Active Games card updates to "🟢 Your turn"
9. **User Action:** Taps game card → opens game interface
10. **Time to Game Started:** ~20s (3 taps + invitation delivery + acceptance)

**From Home Surface:**
1. **Entry Point:** User on home surface, sees Active Games section
2. **User Action:** Taps "See All" in Active Games heading
3. **System Response:** Opens full game dashboard (new screen)
   - Active games (expanded view with history)
   - Game browser (all available games)
   - Recent matches history
4. **User Action:** Navigates to "Start New Game" → picks friend → picks game
5. **Time to Game Started:** ~30s (more taps, but discoverable without entering conversation)

### Flow 4: Discover Someone New via Pulse Map

1. **Entry Point:** User on home surface
2. **User Action:** Taps "Map" in bottom navigation
3. **Transition:** Fade-out messaging view, fade-in Pulse Map (400ms transition)
4. **Pulse Map View:** Identical to current implementation
   - User can explore, tap nodes, send connection requests
5. **User Action:** Sends connection request to new person
6. **Connection Accepted:** User receives notification
7. **User Action:** Taps "Map" → "Messages" in bottom navigation
8. **System Response:** New conversation appears in list
9. **Round-Trip Time:** ~60s (30s Pulse Map exploration + 30s connection flow)

---

## Technical Implementation

### Package Structure

New package: `pkg/ui/messenger/` (mirrors existing `pkg/pulsemap/`)

```
pkg/ui/messenger/
├── home.go              # Home surface layout, Ebitengine Update()/Draw()
├── home_stub.go         # Build-tag stub for test builds
├── home_test.go         # Unit tests (mock subsystems)
├── conversations.go     # Conversation list rendering
├── games.go             # Active games card rendering
├── navigation.go        # Bottom navigation bar state machine
├── conversation_detail.go  # Message view + input field
├── message_composer.go  # Text input, PoW indicator, send button
└── types.go             # Shared types (ConversationItem, GameCard)
```

### Data Structures

```go
// ConversationItem represents a single conversation in the list.
type ConversationItem struct {
    PeerID        peer.ID
    DisplayName   string
    IsSpecter     bool              // True if conversation is with Specter
    LastMessage   *pb.Wave          // Most recent Wave in conversation
    UnreadCount   int
    IsOnline      bool
    IsPinned      bool
    LastActivity  time.Time
}

// GameCard represents an active game in the home surface.
type GameCard struct {
    MatchID       []byte
    GameType      string            // e.g., "cipher_puzzle"
    GameName      string            // Human-readable
    Participants  []string          // Display names
    Status        GameStatus        // YourTurn, Waiting, Starting, Paused
    LastUpdated   time.Time
}

// GameStatus represents the state of an active game.
type GameStatus int

const (
    GameStatusYourTurn GameStatus = iota  // 🟢 User needs to act
    GameStatusWaiting                     // 🔵 Opponent's turn
    GameStatusStarting                    // 🟡 Lobby, waiting for accept
    GameStatusPaused                      // 🔴 Opponent offline >24h
)

// HomeScreen is the Ebitengine game implementing the messaging-first surface.
type HomeScreen struct {
    ctx          context.Context
    eventBus     *app.EventBus
    storage      *store.DB
    waveCache    *storage.Cache
    threadIndex  *threads.Index
    
    // UI state
    conversations []ConversationItem
    games         []GameCard
    activeTab     NavigationTab     // Messages, Map, Specters, More
    
    // Input state
    cursorPos    image.Point
    hoveredConv  int               // Index of hovered conversation (-1 if none)
    scrollOffset int               // Pixels scrolled in conversation list
}

// NavigationTab represents the active bottom navigation tab.
type NavigationTab int

const (
    TabMessages NavigationTab = iota
    TabMap
    TabSpecters
    TabMore
)
```

### Rendering Pipeline

Per Ebitengine architecture (TECHNICAL_IMPLEMENTATION.md §5), implement `ebiten.Game` interface:

```go
// Update processes input and updates UI state (60 calls/sec).
func (h *HomeScreen) Update() error {
    // Handle tab switching.
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        x, y := ebiten.CursorPosition()
        if tab := h.detectTabTap(x, y); tab != -1 {
            h.activeTab = NavigationTab(tab)
            if h.activeTab == TabMap {
                // Transition to Pulse Map view (swap ebiten.Game instance).
                return h.transitionToPulseMap()
            }
        }
    }
    
    // Handle conversation taps.
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        if idx := h.detectConversationTap(); idx != -1 {
            return h.openConversationDetail(h.conversations[idx])
        }
    }
    
    // Handle game card taps.
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        if idx := h.detectGameCardTap(); idx != -1 {
            return h.openGameInterface(h.games[idx])
        }
    }
    
    // Update hover state for visual feedback.
    h.updateHoverState()
    
    // Poll event bus for new Waves, game invitations, connection events.
    h.pollEvents()
    
    return nil
}

// Draw renders the home surface (60 calls/sec).
func (h *HomeScreen) Draw(screen *ebiten.Image) {
    // Clear screen.
    screen.Fill(color.RGBA{20, 20, 25, 255}) // Dark background
    
    // Draw top bar.
    h.drawTopBar(screen)
    
    // Draw active games section.
    h.drawGamesSection(screen)
    
    // Draw conversations list.
    h.drawConversationsList(screen)
    
    // Draw bottom navigation.
    h.drawBottomNavigation(screen)
}

// Layout returns screen dimensions (called on window resize).
func (h *HomeScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
    return outsideWidth, outsideHeight
}
```

### Event Bus Integration

Home surface listens for these events:

1. **`WaveReceived`** → Update conversation list with new message
2. **`GameInvitationReceived`** → Add new game card to Active Games
3. **`GameStateChanged`** → Update game card status (YourTurn/Waiting)
4. **`ConnectionEstablished`** → Add new conversation to list
5. **`PeerOnline`** / **`PeerOffline`** → Update online status dots

Example event handler:

```go
func (h *HomeScreen) pollEvents() {
    select {
    case evt := <-h.eventBus.Subscribe("WaveReceived"):
        wave := evt.Payload.(*pb.Wave)
        h.handleNewWave(wave)
    case evt := <-h.eventBus.Subscribe("GameInvitationReceived"):
        invitation := evt.Payload.(*pb.GameInvitation)
        h.handleGameInvitation(invitation)
    default:
        // No events, continue.
    }
}

func (h *HomeScreen) handleNewWave(wave *pb.Wave) {
    // Find conversation by author peer ID.
    for i := range h.conversations {
        if h.conversations[i].PeerID == wave.AuthorPeerId {
            // Update last message and unread count.
            h.conversations[i].LastMessage = wave
            h.conversations[i].UnreadCount++
            h.conversations[i].LastActivity = time.Now()
            
            // Resort conversation list (most recent first).
            h.sortConversations()
            return
        }
    }
    
    // If conversation doesn't exist, create new one.
    h.conversations = append(h.conversations, ConversationItem{
        PeerID:       wave.AuthorPeerId,
        DisplayName:  h.lookupDisplayName(wave.AuthorPeerId),
        LastMessage:  wave,
        UnreadCount:  1,
        LastActivity: time.Now(),
    })
    h.sortConversations()
}
```

### Conversation Detail View

When user taps a conversation, transition to `ConversationDetailScreen` (separate `ebiten.Game`):

```go
// ConversationDetailScreen shows message history and input field.
type ConversationDetailScreen struct {
    peerID       peer.ID
    messages     []*pb.Wave        // Loaded from thread index
    inputBuffer  string            // Text being typed
    sendState    SendState         // Idle, ComputingPoW, Sending, Sent
}

// SendState tracks message sending progress.
type SendState int

const (
    SendStateIdle SendState = iota
    SendStateComputingPoW             // PoW computation in progress
    SendStateSending                  // Publishing to GossipSub
    SendStateSent                     // Success, show checkmark
)

// Update handles text input and send button.
func (c *ConversationDetailScreen) Update() error {
    // Handle back button (return to home).
    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
        return c.transitionToHome()
    }
    
    // Handle text input.
    c.inputBuffer = ebitenutil.AppendInputChars(c.inputBuffer)
    
    // Handle send button.
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        if c.detectSendButtonTap() && len(c.inputBuffer) > 0 {
            return c.sendMessage()
        }
    }
    
    return nil
}

func (c *ConversationDetailScreen) sendMessage() error {
    // Create Wave protobuf.
    wave := &pb.Wave{
        WaveType:      pb.WaveType_WAVE_TYPE_SURFACE,
        Content:       []byte(c.inputBuffer),
        Timestamp:     time.Now().Unix(),
        AuthorPeerId:  c.myPeerID(),
        AuthorPubkey:  c.myPublicKey(),
        // ParentHash left empty for root-level Wave.
    }
    
    // Start PoW computation in background goroutine.
    c.sendState = SendStateComputingPoW
    go func() {
        if err := pow.ComputeProof(wave, 20); err != nil {
            c.handleSendError(err)
            return
        }
        
        // Sign Wave.
        if err := waves.Sign(wave, c.myKeyPair()); err != nil {
            c.handleSendError(err)
            return
        }
        
        // Publish to GossipSub.
        c.sendState = SendStateSending
        if err := c.publishWave(wave); err != nil {
            c.handleSendError(err)
            return
        }
        
        // Success.
        c.sendState = SendStateSent
        c.messages = append(c.messages, wave)
        c.inputBuffer = ""
    }()
    
    return nil
}
```

---

## A/B Testing Plan

### Variants

1. **Variant A (Control):** Pulse Map as primary surface (current implementation)
2. **Variant B (Treatment):** Messaging-first home surface (this prototype)

### Randomization

- **Allocation:** 50/50 split on first run after implementation
- **Persistence:** Variant stored in Bbolt bucket `config`, key `ab_test_variant`
- **Opt-Out:** Users can switch variants in Settings → Experimental Features

### Metrics

Track per variant (local-only telemetry, no cloud transmission):

| Metric | Variant A (Pulse Map) | Variant B (Messaging) | Target |
|--------|----------------------|---------------------|--------|
| Time to First Message | ~90s (baseline) | ≤60s | 33% reduction |
| Time to First Game | ~30s (3 taps) | ≤20s | 33% reduction |
| D7 Retention Rate | TBD (baseline) | TBD | +10% absolute |
| Sessions per Week | TBD (baseline) | TBD | No regression |
| Pulse Map Engagement | TBD (baseline) | TBD | Acceptable if ≥50% of baseline |
| Messages per Session | TBD (baseline) | TBD | +20% (hypothesis) |
| User Satisfaction | TBD (survey) | TBD (survey) | ≥80% "easy to find conversations" |

### Duration

- **Pilot:** 2 weeks with 50 internal testers (25 per variant)
- **Public:** 4 weeks with all new users after v0.1 launch
- **Decision Point:** Week 6, analyze data and commit to one variant

### Success Criteria for Messaging-First

Variant B (messaging-first) wins if:
1. Time to First Message ≤60s (vs 90s baseline)
2. D7 Retention ≥ Variant A retention (no regression)
3. Pulse Map engagement ≥50% of Variant A (acceptable trade-off)
4. User satisfaction ≥80% "easy to find conversations"

If Variant B wins, promote to default. If Variant A wins, deprecate Variant B but offer as opt-in "Classic View" in settings.

---

## Migration Path

If messaging-first becomes default:

1. **Preserve Pulse Map:** Keep as dedicated tab (not removed, just demoted)
2. **Discovery Onboarding:** Add tutorial nudge: "Tap Map to discover new people" (first time user taps Map tab)
3. **Power User Mode:** Add "Advanced View" toggle in settings to switch back to Pulse Map primary
4. **Documentation Updates:**
   - Update ONBOARDING.md Phase 6 to show messaging-first screen
   - Update PULSE_MAP_ROLE_DECISION.md with A/B test results and new decision
   - Update USER_JOURNEYS.md Journey 1 to reflect 60s target

---

## Implementation Checklist

### Phase 1: Core Prototype (Week 1–2)
- [ ] Create `pkg/ui/messenger/` package structure
- [ ] Implement `HomeScreen` ebiten.Game with static mock data
- [ ] Render conversations list (hardcoded 3 items)
- [ ] Render active games section (hardcoded 1 game)
- [ ] Implement bottom navigation tab switching (Messages/Map transition)
- [ ] Implement conversation tap → detail view transition

### Phase 2: Data Integration (Week 3–4)
- [ ] Load conversations from `pkg/content/threads/` index
- [ ] Load active games from `pkg/store/` bucket `games`
- [ ] Subscribe to event bus for live updates (WaveReceived, GameInvitationReceived)
- [ ] Implement unread count calculation
- [ ] Implement online status checking (libp2p peer liveness)

### Phase 3: Conversation Detail (Week 5–6)
- [ ] Implement `ConversationDetailScreen` ebiten.Game
- [ ] Render message history (scrollable list)
- [ ] Implement text input field (Ebitengine IME or custom)
- [ ] Implement send button with PoW progress indicator
- [ ] Integrate with `pkg/content/waves/` for Wave creation
- [ ] Integrate with `pkg/content/pow/` for PoW computation

### Phase 4: Telemetry & A/B Testing (Week 7–8)
- [ ] Implement A/B variant assignment on first run
- [ ] Add telemetry event tracking (local-only, no cloud)
- [ ] Implement variant switcher in Settings → Experimental Features
- [ ] Create internal tester feedback form (embedded in app)
- [ ] Document A/B test protocol in TESTING.md

### Phase 5: Polish & Launch (Week 9–10)
- [ ] Empty state designs (no conversations, no games)
- [ ] Loading states (conversation list loading)
- [ ] Error states (PoW failed, network disconnected)
- [ ] Accessibility (keyboard navigation, screen reader support)
- [ ] Performance testing (60fps with 50 conversations)
- [ ] Deploy to internal testers (N=50, 25 per variant)

---

## Open Questions

1. **Text Input:** Ebitengine does not have native text input widgets. Options:
   - Use `ebiten/v2/inpututil` for basic text capture (no IME support)
   - Integrate system IME via cgo (platform-specific, complex)
   - Use HTML5 input overlay (requires WebView, not pure Go)
   - **Recommendation:** Start with `inpututil`, add IME in Phase 2 if needed

2. **Pulse Map Discoverability:** If messaging-first becomes default, will users discover the Pulse Map tab?
   - **Mitigation:** Add first-time tutorial nudge: "Tap Map to explore your network"
   - **Metric:** Track "Map tab opened at least once" within first week (target ≥60%)

3. **Game Browser Modal vs Full-Screen:** Should game selection be a modal overlay or full-screen view?
   - **Recommendation:** Modal for quick access from conversation, full-screen for "See All Games" entry point

4. **Specter Tab Visibility:** Should Specters tab be visible in Open mode (grayed out) or hidden entirely?
   - **Recommendation:** Hidden in Open mode to avoid confusion; show unlock prompt when user switches to Hybrid mode

---

## Next Steps

1. **Review with Team:** Circulate this design spec for feedback (deadline: Week 8)
2. **Approve for Implementation:** Decision by Week 9 (go/no-go)
3. **Assign Engineering Resources:** 1 engineer, 10 weeks (full-time)
4. **Create Figma Mockups:** High-fidelity UI designs for developer handoff
5. **Define Telemetry Schema:** Finalize local-only event tracking format
6. **Schedule User Testing:** Recruit 50 internal testers for pilot (Week 10)

---

**Document Status:** Complete design specification, ready for review  
**Dependencies:** PLAN.md §1.3 (A/B testing), USER_JOURNEYS.md (timing targets), PULSE_MAP_ROLE_DECISION.md (rationale)  
**Next Review:** After A/B test results (Week 16)
