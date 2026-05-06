# Pulse Map Role Decision

## Decision: PRIMARY Surface

**Date:** 2026-05-06  
**Decision Maker:** Implementation Team (based on design documents)  
**Status:** COMMITTED

---

## Summary

The Pulse Map is MURMUR's **PRIMARY interface surface**, not a secondary "Explore" tab. The force-directed social graph replaces the infinite scroll and algorithmic feed as the core discovery and navigation mechanism.

---

## Rationale

### 1. Design Principle Alignment

Per DESIGN_DOCUMENT.md Design Principle #4: **"The network is the interface."**

- The Pulse Map makes relationships and content propagation **spatially visible**
- Discovery happens through mesh topology, not recommendation algorithms
- Treating Pulse Map as secondary contradicts this foundational principle

### 2. Differentiation from Competitors

**Why graph-first beats chat-first for MURMUR:**

| Feature | Chat-First (Discord/Telegram/Signal) | Graph-First (MURMUR Pulse Map) |
|---------|--------------------------------------|-------------------------------|
| **Discovery** | Algorithmic suggestions or search | Spatial navigation through visible network topology |
| **Relationships** | Hidden in contact lists | Visible as force-directed edges |
| **Content Propagation** | Push notifications, feeds | Ripple effects radiating from nodes |
| **Anonymity** | Binary (on/off) | Layered (Surface + Specter overlay) |
| **Engagement Model** | Time-on-platform, message counts | Spatial exploration, Resonance milestones |

**Unique value:** MURMUR is not "yet another messaging app with privacy." It's a **social network where the network structure is the UI.**

### 3. Differentiation from Existing Graph Visualizations

**Pulse Map vs. traditional social graphs:**

- **Real-time updates:** Nodes and edges update at 30 Hz; content ripples animate in real-time
- **Dual-layer rendering:** Surface (attributed) and Specter (anonymous) identities coexist on the same map
- **Spatial mechanics:** Mini-games (Specter Hunt, Territory Drift) use Pulse Map as the game board
- **Ephemeral content:** Waves fade as TTL approaches; no permanent archive clutter

### 4. Addressing Graph Literacy Barrier

**Challenge:** New users may lack "graph literacy" — understanding how to navigate and interpret a force-directed graph.

**Mitigation Strategy:**

1. **Onboarding Phase 5 (Guided Exploration)** — Interactive tutorial teaches:
   - Pan and zoom gestures
   - Node selection and inspection
   - Edge semantics (connections, amplifications, replies)
   - Specter overlay toggle

2. **Empty-state and low-population design:**
   - **5-node graph:** Show all nodes with descriptive labels and helpful arrows
   - **50-node graph:** Cluster by community; highlight user's immediate neighborhood
   - **500+ node graph:** Viewport culling + Barnes-Hut optimization; minimap for orientation

3. **Contextual hints during first week:**
   - "Tap a node to see their recent Waves"
   - "Pinch to zoom out and see your extended network"
   - "Toggle the Specter overlay to see anonymous activity"

4. **Chat-first fallback for power users:**
   - **Conversations panel** (slide-in from left) lists active conversations
   - Users can quick-switch between Pulse Map and message thread view
   - Pulse Map remains default, but conversation threads are always 1 swipe away

### 5. New-User Path (< 90 seconds)

**Goal:** New user creates identity → connects to network → sees first content → understands Pulse Map

**Flow:**

1. **Phase 1: Welcome (15s)**
   - Video or animation showing Pulse Map in action (content rippling, Specters appearing)
   - "MURMUR is a living network. This is your Pulse Map."

2. **Phase 2: Identity Creation (20s)**
   - Generate keypair and sigil (automatic, <5s)
   - Show user's node appearing on an otherwise-empty Pulse Map
   - "This is you. Your sigil is your visual identity."

3. **Phase 3: Network Bootstrap (30s)**
   - If invitation: Connect to inviter, see their node appear as first connection
   - If cold start: Connect to 3 bootstrap peers, see them appear as distant nodes
   - "Connecting to the mesh... your network is forming."

4. **Phase 4: First Content (15s)**
   - Bootstrap peers publish Waves that ripple across the graph
   - User's node receives the ripple effects (animated glow)
   - "Tap a glowing node to read their Wave."

5. **Phase 5: First Interaction (10s)**
   - User taps a node → sees Wave content in bottom panel
   - Prompt: "Reply to connect. Or explore the map to discover more."

**Total:** 90 seconds. User now understands:
- Pulse Map is the interface
- Nodes are people, edges are connections
- Content propagates spatially as ripples

---

## Implementation Consequences

### UX Changes Required

1. **Main Application Layout:**
   - Pulse Map occupies full screen on launch
   - Conversations panel accessible via swipe-in (not default view)
   - Settings, Specter toggle, and game launcher in top bar

2. **Navigation Hierarchy:**
   ```
   Pulse Map (default)
   ├── Node Detail Panel (bottom drawer)
   ├── Conversations Panel (left drawer)
   ├── Games Launcher (top right icon)
   └── Settings (top left hamburger menu)
   ```

3. **Empty-State Design:**
   - **0 connections:** Show tutorial prompt "Invite friends or explore nearby nodes"
   - **1-5 connections:** Show small graph with helpful labels
   - **Never show** an empty Pulse Map with zero nodes visible

4. **Performance Requirements:**
   - 60fps @ 500 nodes (achieved per CHANGELOG.md 2026-05-05)
   - 30fps @ 10,000 nodes (achieved with viewport culling and parallel Barnes-Hut)
   - Graceful degradation: cluster small nodes at extreme zoom-out levels

### Documentation Updates

- **ONBOARDING.md:** Update Phase 4 (Guided Exploration) with graph literacy curriculum
- **PULSE_MAP.md:** Add "Pulse Map as Primary Surface" section justifying the design choice
- **PRODUCT_IDENTITY.md:** Already reflects this decision ("The network is the interface")

### Feature Prioritization

**High Priority (before v1.0):**
- Minimap for large graphs (orientation when zoomed in)
- Node search (find a specific user by name or sigil)
- Cluster visualization for 1,000+ node graphs

**Medium Priority (v1.1+):**
- Custom map themes (dark mode, high contrast)
- Heatmap overlays (activity density, Resonance distribution)

**Low Priority (v2.0+):**
- 3D Pulse Map (experimental, for VR/AR)

---

## Alternative Considered and Rejected

### Chat-First with Pulse Map as "Explore" Tab

**Rationale for rejection:**

1. **Contradicts Design Principle #4** — "The network is the interface" becomes meaningless if Pulse Map is buried in a tab
2. **No differentiation** — MURMUR becomes "Signal with better privacy," not a novel social primitive
3. **Misaligned metrics** — Chat-first optimizes for message volume; graph-first optimizes for spatial discovery and Resonance
4. **Anonymous layer invisibility** — Specter activity is spatial; hiding Pulse Map makes anonymous mechanics less discoverable

**When to reconsider:** If user testing shows >50% of users never use Pulse Map after onboarding, despite graph literacy tutorial.

---

## Success Criteria

**Pulse Map is succeeding as PRIMARY surface if:**

1. **Onboarding completion rate ≥ 80%** (users complete Phase 5: Guided Exploration)
2. **D7 Pulse Map engagement ≥ 50%** (users interact with Pulse Map ≥1x per week)
3. **Spatial discovery rate ≥ 30%** (new connections made via Pulse Map exploration, not search or invites)
4. **Specter overlay usage ≥ 20%** (users toggle anonymous layer view ≥1x per session)

**Red flags (trigger UX reassessment):**

- **Onboarding drop-off > 40% during Phase 4** (Guided Exploration) → graph literacy barrier too high
- **Conversation panel usage > 80% of sessions** → users avoiding Pulse Map, preferring chat-first workflow
- **Support requests about "how to find friends"** → discoverability failure

---

## Commitment

This decision is **locked in for v1.0**. UX work proceeds on the assumption that Pulse Map is the default interface. If success criteria are not met by v1.0 release, Phase 1 UX Repositioning will be revisited.

**Next steps:**
- Complete PLAN.md Phase 1 tasks (1.1–1.5) with Pulse Map as PRIMARY assumption
- Update ONBOARDING.md with detailed graph literacy curriculum
- Design empty-state and low-population Pulse Map mockups

---

*Last updated: 2026-05-06*
