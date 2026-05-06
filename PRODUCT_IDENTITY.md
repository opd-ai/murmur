# MURMUR Product Identity Statement

## Target User

**Friend groups of 4–8 people** who want private, playful communication without platform surveillance. Specifically:

- **Privacy-conscious users** who understand metadata risks but don't want to sacrifice usability
- **Friend groups and small communities** seeking alternatives to centralized platforms (Discord, Telegram, WhatsApp)
- **Early adopters** comfortable with peer-to-peer technology and willing to tolerate rough edges for privacy benefits
- **Digital natives** (18–35) familiar with ephemeral messaging (Snapchat, Signal Stories) and online gaming
- **Self-sovereign identity advocates** who want cryptographic identity without blockchain or tokens

### Anti-Personas (Not Our Target)
- Influencers seeking audience growth or public broadcast platforms
- Users requiring 24/7 uptime guarantees and enterprise SLAs
- Non-technical users unwilling to understand basic P2P concepts (peer connections, NAT traversal)
- Users seeking permanent archives or searchable message history
- Cryptocurrency enthusiasts expecting token incentives or blockchain integration

---

## Core Loop (One Sentence)

**Connect with friends or interesting strangers, exchange ephemeral messages through a spatial social graph, and play collaborative mini-games — all while your metadata remains unlinkable from your identity.**

### Expanded Core Loop (User Journey)

1. **Install and create identity** — Generate Ed25519 keypair and visual sigil; optionally create Specter pseudonym
2. **Connect to the mesh** — Bootstrap via invitation link or public DHT; see friends appear as nodes on Pulse Map
3. **Navigate the Pulse Map** — Explore the force-directed graph; discover content via spatial proximity
4. **Exchange Waves** — Publish short-form messages (≤2048 bytes) that propagate through the mesh and expire after TTL
5. **Play mini-games** — Engage in asynchronous or real-time games with friends or anonymous Specters
6. **Build Resonance** — Earn reputation through participation, unlocking new mechanics and game types
7. **Maintain privacy** — Switch between Surface (public) and Specter (anonymous) identities; route traffic through Shroud circuits

---

## Explicitly NOT In Scope

### Features We Will NOT Build

1. **Public broadcast or influencer mechanics**
   - No follower counts, no viral amplification algorithms, no trending feeds
   - Rationale: These features incentivize attention-seeking behavior and metadata leakage incompatible with our privacy model

2. **Permanent content archives**
   - No server-side message retention, no searchable history beyond local TTL
   - Rationale: Ephemeral-by-default reduces attack surface and aligns with privacy-first values

3. **Algorithmic feeds or recommendations**
   - No content ranking, no "you might like" suggestions, no engagement optimization
   - Rationale: Algorithms require behavior tracking and create filter bubbles; Pulse Map spatial navigation replaces this

4. **Cryptocurrency, tokens, or paid reputation**
   - No blockchain integration, no tokenomics, no financialization of social interactions
   - Rationale: Financial incentives attract adversarial behavior and regulatory scrutiny; complicates UX

5. **Centralized server infrastructure**
   - No authentication servers, no message relays (beyond peer-operated Shroud relays), no cloud sync
   - Rationale: Servers are single points of failure and surveillance; P2P architecture is foundational

6. **Competing with Tor as general-purpose anonymity network**
   - Shroud is optimized for social traffic (latency-tolerant, small messages), not web browsing or file transfer
   - Rationale: Tor/I2P are mature solutions for strong anonymity; MURMUR integrates with them rather than replacing them

7. **Enterprise features (SSO, admin controls, compliance dashboards)**
   - No organization accounts, no centralized moderation, no audit logs
   - Rationale: Self-sovereign identity is incompatible with centralized control; communities self-moderate

---

## What Makes MURMUR Different

### Unique Value Proposition

1. **Anonymous layer as first-class feature**
   - Most privacy tools treat anonymity as an add-on; MURMUR makes Specters and Shroud circuits core to the social experience
   - Users can seamlessly switch between Surface (attributed) and Specter (anonymous) identities within the same application

2. **Spatial UI replaces algorithmic feed**
   - Pulse Map force-directed graph makes relationships and content propagation visible
   - Discovery happens through spatial navigation, not recommendation engines

3. **Ephemeral by default**
   - Waves expire after configurable TTL (default 7 days, max 30); no permanent record
   - Reduces legal liability, storage costs, and long-term metadata exposure

4. **Metadata unlinkability built into architecture**
   - Shroud onion routing + GossipSub prevents network observers from correlating identities with traffic patterns
   - No phone numbers, no email addresses, no centralized account database

5. **Playful mechanics drive retention**
   - Mini-games (async and real-time) create social value beyond messaging
   - Resonance reputation system (non-transferable, non-financialized) unlocks mechanics without gamifying engagement

6. **Extension surface for compatible networks**
   - MURMUR is designed as a protocol, not just an app
   - Third parties can build domain-specific networks (creative tools, niche communities, specialized games) inheriting MURMUR's identity and transport layers

---

## Success Metrics

### Qualitative Measures
- **D7 retention in active conversations** (users exchanging ≥1 Wave in last 7 days)
- **Games started per week per active user** (measures playful engagement)
- **Specter identity adoption rate** (% of users who create and use anonymous identities)
- **Tor/I2P transport adoption** (% of users routing traffic through external anonymity networks)

### Anti-Metrics (What We Do NOT Measure)
- ❌ Daily Active Users (DAU) — rewards addiction, not value
- ❌ Time spent in app — rewards engagement maximization, not intentional use
- ❌ Follower counts / social graph size — incentivizes growth over privacy
- ❌ Message volume / content velocity — optimizes for noise over signal

### Success Threshold
**A target user (friend group of 4–8 people) can install MURMUR, find each other, exchange messages, and complete a shared game in under 15 minutes without being asked to understand keys, circuits, or reputation.**

If this threshold is not met, UX complexity is too high and must be reduced.

---

## Product Philosophy

### Core Tenets

1. **Privacy is structural, not contractual**
   - We cannot betray users because we never have their data
   - Metadata unlinkability is a protocol property, not a pinky promise

2. **Identity is self-sovereign**
   - Users own their Ed25519 keypairs; no intermediary can revoke or censor
   - BIP-39 recovery means users are responsible for their own backups

3. **The network is the interface**
   - Pulse Map spatial graph replaces infinite scroll and algorithmic feeds
   - Social discovery happens through mesh topology, not recommendation engines

4. **Complexity is revealed, not imposed**
   - Advanced users can dive into circuit construction, Resonance mechanics, game design
   - Casual users get "it just works" messaging with privacy by default

5. **Growth must be organic**
   - No paid acquisition, no viral growth hacks, no engagement algorithms
   - Network effects emerge from genuine social value, not manufactured virality

---

## Positioning Statement

**For privacy-conscious friend groups who want playful, ephemeral communication without platform surveillance, MURMUR is a peer-to-peer social network that makes metadata unlinkability and anonymous identity first-class features — unlike Discord, Telegram, or even Signal, which still centralize infrastructure and treat anonymity as an afterthought. MURMUR is designed as an extensible protocol, enabling third parties to build compatible networks that inherit our privacy guarantees while adding domain-specific value.**

---

*Last updated: 2026-05-06*
