# Pulse Map Empty-State and Low-Population Design

**Version:** 1.0  
**Date:** 2026-05-06  
**Status:** Complete Specification  
**Purpose:** Define UX for Pulse Map with 0–20 visible nodes

---

## Problem Statement

Per PLAN.md §1.4, the Pulse Map must never show users "an empty or near-empty graph as first impression." This is critical for retention — a new user sees 1 node (themselves) and 0 edges, which communicates loneliness and abandonment rather than potential. The design must fill the visual space meaningfully while being honest about the network state.

**Design Principle:** Empty states are not failures; they are invitations to action.

---

## Node Population Scenarios

### Scenario 1: Absolute Zero (0 nodes visible)
**When:** Impossible state — user always sees their own node  
**Fallback:** If somehow triggered (e.g., identity not loaded), show full-screen error message:
- Heading: "Unable to load your identity"
- Message: "MURMUR cannot display the Pulse Map without your identity. Please restart the application."
- CTA: "Restart" button (calls `app.Restart()`)

### Scenario 2: Solo User (1 node = self)
**When:** User completed onboarding, has not sent or received any invitations  
**Visual State:**
- User node at center, glowing brightly (warm orange-gold)
- Dark background (deep blue-black gradient)
- Soft particles drifting in background (20–30 translucent dots, animated Brownian motion)
- No edges, no other nodes

**UI Overlay:**
```
┌─────────────────────────────────────────────────┐
│                                                 │
│              [Your Identity]                    │ ← User node (center)
│                                                 │
│                                                 │
│  ┌────────────────────────────────────────┐   │
│  │  You're the first node in your network │   │ ← Tooltip (above node)
│  │                                         │   │
│  │  Invite a friend to start connecting:  │   │
│  │  [Generate Invitation Link]            │   │ ← Primary CTA
│  │                                         │   │
│  │  or discover nearby peers:              │   │
│  │  [Scan Local Network]                  │   │ ← Secondary CTA (mDNS)
│  └────────────────────────────────────────┘   │
│                                                 │
│  "Your network starts here. Each connection    │ ← Footer text
│   is a node; each relationship is an edge."    │    (educational)
└─────────────────────────────────────────────────┘
```

**Interactions:**
- **Generate Invitation Link:** Opens invitation modal (QR code + shareable link)
- **Scan Local Network:** Starts mDNS discovery (finds peers on same WiFi)
- **Dismiss:** Close tooltip (small × in corner), but reshow on next launch until 1+ connections exist

**Animation:**
- User node pulses gently (1-second period, 10% scale variation)
- Background particles drift slowly (0.5 px/sec, wrap at edges)
- Tooltip fades in after 2 seconds (not immediate, allows user to orient)

### Scenario 3: Dyad (2 nodes)
**When:** User has 1 connection (invited friend or accepted invitation)  
**Visual State:**
- User node (left) and friend node (right), separated by ~300px
- Single edge connecting them (warm gradient, animated pulse traveling along edge)
- Background particles now cluster slightly around the two nodes (gravitational effect)

**UI Overlay:**
```
┌─────────────────────────────────────────────────┐
│                                                 │
│    [You]  ━━━━━━━━━━━━━━━━━━  [Alice]          │ ← Two nodes, one edge
│                                                 │
│                                                 │
│  ┌────────────────────────────────────────┐   │
│  │  Your network is growing!              │   │ ← Congratulatory tone
│  │                                         │   │
│  │  Invite more friends to expand:        │   │
│  │  [Invite Another Friend]               │   │ ← Primary CTA
│  │                                         │   │
│  │  or explore what you can do together:  │   │
│  │  [Send a Wave]  [Play a Game]          │   │ ← Action CTAs
│  └────────────────────────────────────────┘   │
│                                                 │
│  "Tap a node to see details and interact."     │ ← Footer (teaches interaction)
└─────────────────────────────────────────────────┘
```

**Interactions:**
- **Invite Another Friend:** Opens invitation modal (reusable flow)
- **Send a Wave:** Opens Wave composer targeting the friend
- **Play a Game:** Opens game browser filtered to games compatible with friend
- **Tap Node:** Opens node detail panel (same as full-population behavior)

**Animation:**
- Edge pulses every 2 seconds (glow travels from user → friend, simulates data flow)
- Nodes bob gently (sine wave, 2px amplitude, 3-second period, out of phase)
- Tooltip auto-dismisses after 10 seconds (or user interaction)

### Scenario 4: Small Network (3–5 nodes)
**When:** User has 2–4 connections  
**Visual State:**
- Nodes arranged in rough circle (force-directed layout, but constrained to viewport)
- Multiple edges (may overlap if connections form triangles)
- Background particles now form subtle "trails" between nodes (shows potential future connections)

**UI Overlay:**
```
┌─────────────────────────────────────────────────┐
│                                                 │
│         [Alice]                                 │
│          /   \                                  │ ← Small graph visible
│       [You]—[Bob]                               │
│          \   /                                  │
│        [Carol]                                  │
│                                                 │
│  ┌────────────────────────────────────────┐   │
│  │  Explore your network:                 │   │ ← Shift to exploration
│  │                                         │   │
│  │  • Tap a node to see their Waves       │   │ ← Tutorial hints
│  │  • Pan and zoom to navigate            │   │    (fewer CTAs, more
│  │  • Toggle Anonymous Layer to see       │   │     instructions)
│  │    Specters                             │   │
│  │                                         │   │
│  │  [Invite More Friends]    [Dismiss]    │   │
│  └────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

**Interactions:**
- Tooltip auto-dismisses after 15 seconds
- If user performs any Pulse Map interaction (tap node, pan, zoom), tooltip dismisses immediately
- "Dismiss" button sets flag `pulse_map_tutorial_dismissed` in Bbolt → never show again

**Animation:**
- Force-directed layout runs at full 60fps (no need for Barnes-Hut yet)
- Edges animate with subtle "data flow" particles (1–2 per edge, traveling at 20px/sec)
- Nodes glow brighter on hover (prepare user for tap interaction)

### Scenario 5: Growing Network (6–20 nodes)
**When:** User has 5–19 connections (approaching critical mass)  
**Visual State:**
- Graph spans most of viewport, but still easily navigable without pan/zoom
- Clusters may form if user has friend groups (e.g., work friends vs hobby friends)
- Background particles fade out (no longer needed, graph itself is visually rich)

**UI Overlay:**
- **No tooltip by default** (user has learned the basics)
- **Onboarding nudge appears if user hasn't used specific features:**
  - If never panned: "💡 Drag to pan the map" (small hint, top-right, auto-dismiss after 5s)
  - If never zoomed: "💡 Scroll to zoom in/out" (small hint, top-right, auto-dismiss after 5s)
  - If never toggled Anonymous Layer: "💡 Toggle Anonymous Layer to see Specters" (if in Hybrid/Fortress mode)

**Interactions:**
- Pulse Map behaves identically to full-population mode (no special empty-state UI)
- Tutorial hints are non-modal, non-blocking (user can ignore them)
- If user dismisses 3 tutorial hints, stop showing any hints (flag `pulse_map_expert_mode`)

**Animation:**
- Force-directed layout runs continuously (smooth, organic motion)
- Nodes have subtle "breathing" animation (scale 1.0 → 1.02 → 1.0, 4-second period)
- Edges have faint glow gradient (darker at midpoint, brighter near nodes)

---

## Visual Design Specifications

### Color Palette (Empty States)

| Element | Color | Rationale |
|---------|-------|-----------|
| Background (top) | `#0a0a14` (deep blue-black) | Matches space/network metaphor |
| Background (bottom) | `#050508` (pure black) | Creates depth gradient |
| User node (self) | `#ffa940` (warm orange-gold) | Positive, inviting, distinct from others |
| Friend node (Surface) | `#40a9ff` (cool blue) | Calm, trustworthy, contrasts with user |
| Specter node (if visible) | `#b37feb` (purple) | Mysterious, anonymous layer theming |
| Edge (Surface) | `#1890ff` → `#40a9ff` (blue gradient) | Data flow visualization |
| Edge (Specter) | `#722ed1` → `#b37feb` (purple gradient) | Anonymous layer theming |
| Background particles | `#ffffff` @ 8% opacity | Subtle, non-distracting |
| Tooltip background | `#1f1f28` @ 95% opacity | Semi-transparent, legible |
| Tooltip text | `#e0e0e8` (off-white) | High contrast, readable |
| CTA button (primary) | `#1890ff` (bright blue) | High visibility, action-oriented |
| CTA button (secondary) | `#595959` (medium gray) | Lower priority, still discoverable |

### Typography (Empty States)

| Element | Font | Size | Weight | Color |
|---------|------|------|--------|-------|
| Tooltip heading | Inter | 18px | 600 (SemiBold) | `#e0e0e8` |
| Tooltip body | Inter | 14px | 400 (Regular) | `#b0b0b8` |
| CTA button text | Inter | 14px | 500 (Medium) | `#ffffff` |
| Footer text | Inter | 12px | 400 (Regular) | `#808088` |
| Tutorial hint | Inter | 13px | 400 (Regular) | `#c0c0c8` |

### Layout Dimensions

| Element | Width | Height | Margin |
|---------|-------|--------|--------|
| Tooltip (Solo/Dyad) | 400px | Auto (min 180px) | 20px from all edges |
| Tooltip (Small Network) | 350px | Auto (min 200px) | 20px from all edges |
| CTA button (primary) | 100% of tooltip width | 40px | 12px top/bottom |
| CTA button (secondary) | 48% of tooltip width | 36px | 8px top/bottom |
| Tutorial hint | 250px | Auto (min 50px) | 12px from viewport edge |

### Animation Timing

| Animation | Duration | Easing | Frequency |
|-----------|----------|--------|-----------|
| Node pulse (Solo) | 1000ms | Ease-in-out | Continuous |
| Edge pulse (Dyad) | 2000ms | Linear | Every 2s |
| Node bob (Dyad) | 3000ms | Sine wave | Continuous |
| Tooltip fade-in | 400ms | Ease-out | Once on appear |
| Tooltip fade-out | 300ms | Ease-in | Once on dismiss |
| Background particles | 60000ms (drift loop) | Linear | Continuous |
| Tutorial hint fade-in | 300ms | Ease-out | Once on appear |
| Tutorial hint fade-out | 200ms | Ease-in | Once after 5s timeout |

---

## Implementation Strategy

### Package: `pkg/pulsemap/emptystate/`

```go
// Package emptystate provides UI overlays and guidance for low-population Pulse Map states.
package emptystate

import (
    "image"
    "time"
    
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/opd-ai/murmur/pkg/store"
)

// StateType represents the current empty-state scenario.
type StateType int

const (
    StateAbsoluteZero StateType = iota  // 0 nodes (error state)
    StateSolo                            // 1 node (self only)
    StateDyad                            // 2 nodes (user + 1 friend)
    StateSmallNetwork                    // 3–5 nodes
    StateGrowingNetwork                  // 6–20 nodes
    StateFullPopulation                  // 21+ nodes (no empty-state UI)
)

// Detector determines the current empty-state based on visible node count.
type Detector struct {
    db *store.DB
}

// NewDetector creates an empty-state detector with the given database.
func NewDetector(db *store.DB) *Detector {
    return &Detector{db: db}
}

// DetectState returns the current empty-state type based on node count.
func (d *Detector) DetectState(visibleNodes int) StateType {
    switch {
    case visibleNodes == 0:
        return StateAbsoluteZero
    case visibleNodes == 1:
        return StateSolo
    case visibleNodes == 2:
        return StateDyad
    case visibleNodes >= 3 && visibleNodes <= 5:
        return StateSmallNetwork
    case visibleNodes >= 6 && visibleNodes <= 20:
        return StateGrowingNetwork
    default:
        return StateFullPopulation
    }
}

// ShouldShowTooltip checks if the tooltip should be displayed for this state.
// Returns false if user dismissed it or reached expert mode.
func (d *Detector) ShouldShowTooltip(state StateType) bool {
    // Check if tooltip was dismissed for this state.
    dismissed, _ := d.db.Get(store.BucketConfig, []byte("pulse_map_tutorial_dismissed"))
    if len(dismissed) > 0 && dismissed[0] == 1 {
        return false
    }
    
    // Check if user is in expert mode (dismissed 3+ hints).
    expertMode, _ := d.db.Get(store.BucketConfig, []byte("pulse_map_expert_mode"))
    if len(expertMode) > 0 && expertMode[0] == 1 {
        return false
    }
    
    // Show tooltip for Solo, Dyad, SmallNetwork states.
    return state == StateSolo || state == StateDyad || state == StateSmallNetwork
}

// Overlay renders the empty-state UI overlay for the given state.
type Overlay struct {
    state       StateType
    fadeIn      float64  // 0.0 → 1.0 over fade-in duration
    showTime    time.Time
    onInvite    func()   // Callback when "Invite Friend" button tapped
    onSendWave  func()   // Callback when "Send a Wave" button tapped (Dyad only)
    onPlayGame  func()   // Callback when "Play a Game" button tapped (Dyad only)
    onDismiss   func()   // Callback when tooltip dismissed
}

// NewOverlay creates an empty-state overlay for the given state.
func NewOverlay(state StateType) *Overlay {
    return &Overlay{
        state:    state,
        fadeIn:   0.0,
        showTime: time.Now(),
    }
}

// Update processes input and updates overlay state.
func (o *Overlay) Update() error {
    // Animate fade-in.
    if o.fadeIn < 1.0 {
        elapsed := time.Since(o.showTime).Seconds()
        o.fadeIn = min(1.0, elapsed / 0.4)  // 400ms fade-in
    }
    
    // Handle button taps (check mouse position vs button bounds).
    // Implementation omitted for brevity — see full implementation in pkg/pulsemap/emptystate/overlay.go
    
    return nil
}

// Draw renders the overlay on the given screen.
func (o *Overlay) Draw(screen *ebiten.Image) {
    // Apply fade-in alpha.
    alpha := uint8(o.fadeIn * 255)
    
    switch o.state {
    case StateSolo:
        o.drawSoloOverlay(screen, alpha)
    case StateDyad:
        o.drawDyadOverlay(screen, alpha)
    case StateSmallNetwork:
        o.drawSmallNetworkOverlay(screen, alpha)
    case StateGrowingNetwork:
        o.drawGrowingNetworkOverlay(screen, alpha)
    }
}

// drawSoloOverlay renders the Solo state tooltip.
func (o *Overlay) drawSoloOverlay(screen *ebiten.Image, alpha uint8) {
    // Draw tooltip box at center of screen.
    // Implementation: rounded rect, drop shadow, heading + body text, 2 CTA buttons.
    // See full implementation in pkg/pulsemap/emptystate/overlay.go
}

// ... (drawDyadOverlay, drawSmallNetworkOverlay, drawGrowingNetworkOverlay implementations)

// TutorialHint represents a small, non-blocking hint shown in corner.
type TutorialHint struct {
    text      string
    showTime  time.Time
    dismissed bool
}

// NewTutorialHint creates a tutorial hint with the given text.
func NewTutorialHint(text string) *TutorialHint {
    return &TutorialHint{
        text:     text,
        showTime: time.Now(),
    }
}

// Update checks if hint should auto-dismiss after 5 seconds.
func (h *TutorialHint) Update() {
    if !h.dismissed && time.Since(h.showTime) > 5*time.Second {
        h.dismissed = true
    }
}

// Draw renders the hint in top-right corner (if not dismissed).
func (h *TutorialHint) Draw(screen *ebiten.Image) {
    if h.dismissed {
        return
    }
    
    // Implementation: small rounded rect, light background, hint text, close button.
    // See full implementation in pkg/pulsemap/emptystate/hint.go
}
```

### Integration with Pulse Map

In `pkg/pulsemap/game.go` (main Pulse Map game loop):

```go
// Game is the Ebitengine game implementing the Pulse Map.
type Game struct {
    // ... existing fields ...
    
    // Empty-state detection and overlay.
    emptyStateDetector *emptystate.Detector
    emptyStateOverlay  *emptystate.Overlay
    tutorialHints      []*emptystate.TutorialHint
}

// Update processes input and updates Pulse Map state.
func (g *Game) Update() error {
    // Detect current empty-state.
    visibleNodes := len(g.layout.GetVisibleNodes())
    state := g.emptyStateDetector.DetectState(visibleNodes)
    
    // Show overlay if applicable.
    if state != emptystate.StateFullPopulation {
        if g.emptyStateDetector.ShouldShowTooltip(state) {
            if g.emptyStateOverlay == nil || g.emptyStateOverlay.State() != state {
                g.emptyStateOverlay = emptystate.NewOverlay(state)
                g.emptyStateOverlay.OnInvite = g.openInvitationModal
                g.emptyStateOverlay.OnSendWave = g.openWaveComposer
                g.emptyStateOverlay.OnPlayGame = g.openGameBrowser
                g.emptyStateOverlay.OnDismiss = func() {
                    g.emptyStateDetector.MarkTooltipDismissed()
                    g.emptyStateOverlay = nil
                }
            }
            g.emptyStateOverlay.Update()
        }
    } else {
        // Full population — clear overlay.
        g.emptyStateOverlay = nil
    }
    
    // Update tutorial hints (if any).
    for _, hint := range g.tutorialHints {
        hint.Update()
    }
    
    // ... rest of Update logic ...
    
    return nil
}

// Draw renders the Pulse Map.
func (g *Game) Draw(screen *ebiten.Image) {
    // ... existing rendering ...
    
    // Draw empty-state overlay (if active).
    if g.emptyStateOverlay != nil {
        g.emptyStateOverlay.Draw(screen)
    }
    
    // Draw tutorial hints.
    for _, hint := range g.tutorialHints {
        hint.Draw(screen)
    }
}

// ShowTutorialHint displays a non-blocking tutorial hint.
// Called by interaction handlers (e.g., first pan, first zoom).
func (g *Game) ShowTutorialHint(text string) {
    // Check if user is in expert mode.
    expertMode, _ := g.db.Get(store.BucketConfig, []byte("pulse_map_expert_mode"))
    if len(expertMode) > 0 && expertMode[0] == 1 {
        return
    }
    
    hint := emptystate.NewTutorialHint(text)
    g.tutorialHints = append(g.tutorialHints, hint)
    
    // Track hint count — if user dismissed 3+, enable expert mode.
    hintCount, _ := g.db.Get(store.BucketConfig, []byte("pulse_map_hints_dismissed_count"))
    count := 0
    if len(hintCount) > 0 {
        count = int(hintCount[0])
    }
    count++
    g.db.Put(store.BucketConfig, []byte("pulse_map_hints_dismissed_count"), []byte{byte(count)})
    
    if count >= 3 {
        g.db.Put(store.BucketConfig, []byte("pulse_map_expert_mode"), []byte{1})
    }
}
```

---

## Testing Strategy

### Unit Tests

1. **State Detection:** Verify `Detector.DetectState()` returns correct state for node counts 0, 1, 2, 5, 10, 25
2. **Tooltip Visibility:** Verify `ShouldShowTooltip()` respects dismissed flag and expert mode
3. **Overlay Rendering:** Snapshot tests for each state (compare rendered image vs golden image)
4. **Tutorial Hint Auto-Dismiss:** Verify hint dismisses after 5 seconds
5. **Expert Mode Trigger:** Verify expert mode activates after 3 dismissed hints

### Integration Tests

1. **Solo → Dyad Transition:** Simulate connection establishment, verify overlay transitions
2. **Small Network Tutorial:** Simulate user interactions (tap, pan, zoom), verify hints trigger correctly
3. **Empty-State Persistence:** Verify dismissed state persists across app restarts
4. **Callback Execution:** Mock callbacks (onInvite, onSendWave), verify they fire on button tap

### Visual Regression Tests

Use Ebitengine's headless mode to render Pulse Map states and compare screenshots:

```bash
# Generate baseline screenshots for each empty-state
go test -tags=visual_regression ./pkg/pulsemap/emptystate -run TestVisualRegression

# Compare current rendering vs baseline (fail if >2% pixel difference)
go test -tags=visual_regression ./pkg/pulsemap/emptystate -run TestVisualRegression -update=false
```

---

## User Testing Protocol

### Scenario 1: First-Time User

**Setup:** Fresh install, no connections  
**Expected Behavior:**
- User sees Solo state overlay with invitation CTA
- Tooltip is non-intrusive (user can still explore Pulse Map)
- User taps "Generate Invitation Link" → invitation modal opens
- After sending invitation, tooltip persists until friend connects

**Success Criteria:**
- 80% of users tap invitation CTA within 30 seconds
- 0% of users report feeling "stuck" or "confused" in post-session survey

### Scenario 2: Dyad Formation

**Setup:** User has 1 connection  
**Expected Behavior:**
- User sees Dyad state overlay with congratulatory message
- Two action CTAs: "Send a Wave" and "Play a Game"
- User taps "Send a Wave" → Wave composer opens with friend pre-selected
- Tooltip auto-dismisses after 10 seconds if no interaction

**Success Criteria:**
- 60% of users tap either CTA within 20 seconds
- 90% of users send a Wave or start a game within first session after connecting

### Scenario 3: Small Network Exploration

**Setup:** User has 3–5 connections  
**Expected Behavior:**
- User sees Small Network state overlay with tutorial hints
- Overlay teaches basic interactions (tap, pan, zoom, toggle layer)
- User performs any interaction → overlay dismisses immediately
- If user doesn't interact, overlay auto-dismisses after 15 seconds

**Success Criteria:**
- 70% of users interact with Pulse Map (tap node, pan, zoom) within 30 seconds
- 50% of users toggle Anonymous Layer within first minute (if in Hybrid mode)
- 80% of users report "easy to navigate" in post-session survey

---

## Accessibility Considerations

1. **Keyboard Navigation:**
   - Tab key cycles through CTA buttons
   - Enter key activates focused button
   - Escape key dismisses tooltip

2. **Screen Reader Support:**
   - Tooltip heading announced as "heading level 2"
   - CTA buttons announced with role="button" and descriptive text
   - Tutorial hints announced as "status" (non-intrusive)

3. **High Contrast Mode:**
   - All text meets WCAG AA contrast ratio (4.5:1 minimum)
   - Buttons have 3px border in high-contrast mode (increased visibility)

4. **Reduce Motion:**
   - If OS-level "reduce motion" preference detected, disable animations:
     - No node pulse, edge pulse, particle drift
     - Overlay fade-in becomes instant (0ms)
     - Tutorial hints appear/disappear instantly (no fade)

---

## Open Questions

1. **Invitation Flow Placement:** Should invitation modal be full-screen or overlay?
   - **Recommendation:** Overlay (preserves Pulse Map context, less jarring)

2. **Tutorial Hint Persistence:** Should hints reappear if user returns to low-population state after growing network?
   - **Recommendation:** No — once dismissed or expert mode reached, never show again

3. **Localization:** How do empty-state tooltips handle non-English languages?
   - **Recommendation:** Defer to post-v0.1 — hardcode English strings for now, add i18n later

4. **Mobile Layout:** Tooltips are sized for desktop (400px width) — how to adapt for mobile (<400px screens)?
   - **Recommendation:** Responsive design — tooltip width = 90% of viewport width on mobile

---

## Success Metrics

Track via local telemetry (Bbolt bucket `telemetry`, no cloud transmission):

| Metric | Target | Measurement |
|--------|--------|-------------|
| Solo Overlay CTA Click Rate | ≥80% | % of Solo state sessions with invitation CTA tapped within 30s |
| Dyad Overlay CTA Click Rate | ≥60% | % of Dyad state sessions with action CTA tapped within 20s |
| Small Network Interaction Rate | ≥70% | % of Small Network sessions with Pulse Map interaction within 30s |
| Tutorial Hint Dismissal Rate | ≤50% | % of hints manually dismissed vs auto-dismissed |
| Expert Mode Adoption | ≤20% | % of users reaching expert mode (dismissed 3+ hints) |
| Post-Session Survey: "Easy to Navigate" | ≥80% | Likert scale 4–5 out of 5 |

---

## Implementation Checklist

- [ ] Create `pkg/pulsemap/emptystate/` package structure
- [ ] Implement `Detector` with `DetectState()` and `ShouldShowTooltip()`
- [ ] Implement `Overlay` with `Update()` and `Draw()` for all 5 states
- [ ] Implement `TutorialHint` with auto-dismiss timer
- [ ] Integrate with `pkg/pulsemap/game.go` Update() and Draw() loops
- [ ] Add invitation modal callback wiring
- [ ] Add Wave composer callback wiring
- [ ] Add game browser callback wiring
- [ ] Implement dismissed state persistence (Bbolt `pulse_map_tutorial_dismissed` flag)
- [ ] Implement expert mode persistence (Bbolt `pulse_map_expert_mode` flag)
- [ ] Write unit tests for state detection and tooltip visibility logic
- [ ] Write integration tests for state transitions
- [ ] Write visual regression tests for overlay rendering
- [ ] Conduct user testing with 10 participants (5 Solo, 3 Dyad, 2 Small Network)
- [ ] Analyze telemetry and refine thresholds
- [ ] Document findings in `docs/EMPTY_STATE_USER_TESTING_RESULTS.md`

---

**Document Status:** Complete specification, ready for implementation  
**Dependencies:** PLAN.md §1.4, USER_JOURNEYS.md (Journey 1), PULSE_MAP.md  
**Next Steps:** Implement `pkg/pulsemap/emptystate/` package (estimate 1 week, 1 engineer)
