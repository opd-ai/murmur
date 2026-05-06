# Invite-First Launch Plan

**Document Version:** 1.0  
**Last Updated:** 2026-05-06  
**Status:** v0.1 Launch Planning  
**Prerequisite:** Minimum Viable Network Analysis (docs/MINIMUM_VIABLE_NETWORK.md)

---

## Executive Summary

MURMUR will launch with an **invite-first strategy** where every new user receives **5 bundled invites** carrying friend-group context. This approach mitigates the cold-start problem (empty Pulse Maps, no discovery) by seeding the network with cohesive social groups rather than isolated individuals.

**Key Design:** Invites are not just onboarding links — they embed the inviter's social graph context, enabling immediate meaningful connections on the Pulse Map for invitees.

---

## 1. Core Invite Mechanism

### 1.1 Invite Structure

Each invite is a **signed data structure** containing:

```protobuf
message Invite {
  bytes invite_id = 1;              // BLAKE3 hash of (inviter_pubkey || nonce)
  bytes inviter_pubkey = 2;         // Ed25519 Surface Layer public key
  string inviter_display_name = 3;  // Optional: human-readable name
  bytes inviter_sigil_seed = 4;     // Deterministic sigil generation seed
  repeated bytes group_member_keys = 5; // Public keys of inviter's connected contacts (0-10 keys)
  int64 created_timestamp = 6;      // Unix timestamp
  int64 expires_timestamp = 7;      // Default: 30 days from creation
  bytes signature = 8;              // Ed25519 signature over (invite_id || inviter_pubkey || expires_timestamp)
  uint32 remaining_uses = 9;        // Single-use (1) or multi-use (N, default 1)
}
```

### 1.2 Invite Encoding

**Format:** Base64-encoded protobuf wrapped in URI scheme  
**Schema:** `murmur://invite/<base64_invite_data>`

**Example:**
```
murmur://invite/CiAK...Qo1BhcnR5IGludml0ZQ==
```

**Properties:**
- **Shareable:** Copy-pasteable text, QR code, or deep link
- **Self-contained:** No server lookup required — all context embedded
- **Verifiable:** Ed25519 signature prevents tampering
- **Expirable:** 30-day default TTL prevents stale invites
- **Single-use default:** Prevents invite farming (configurable to multi-use for trusted contexts)

### 1.3 Friend-Group Context

The **critical innovation** in MURMUR invites is the inclusion of `group_member_keys` (0-10 public keys of the inviter's contacts).

**Purpose:**
- **Pre-populate Pulse Map:** Invitee's graph isn't empty — they see inviter + inviter's contacts (1 + N nodes)
- **Discovery seed:** Invitee can explore Waves from the inviter's social circle
- **Connection prompts:** UI suggests connecting to group members after accepting invite

**Privacy Consideration:** Including contact keys reveals social graph structure to invitee. This is acceptable because:
1. Inviter explicitly shares invite (consent to reveal connections)
2. Only public keys shared (no private content or metadata)
3. Invitee must accept invite (opt-in to receive context)

**Limit:** Max 10 contact keys per invite to bound invite size (~400 bytes baseline + 32 bytes/key × 10 = ~720 bytes).

---

## 2. Invite Quota System

### 2.1 Per-User Quota

**Default:** Every new user receives **5 invites** upon account creation.

**Rationale:**
- **5-8 MVN:** Each user can invite their core friend group (5 friends = 1 group)
- **Controlled growth:** Prevents spam/abuse while enabling organic expansion
- **Scarcity psychology:** Limited invites signal exclusivity, increase perceived value

### 2.2 Invite Regeneration

**Earn Additional Invites:**
- **Resonance milestones:** Unlocking Shade (25), Wraith (50), Phantom (100) grants +2 invites each
- **Active participation:** Users who post ≥5 Waves/week for 4 consecutive weeks earn +3 invites
- **Friend acceptance:** For every 3 accepted invites from your group, earn +1 invite

**Cap:** Maximum 20 invites per user at any time (prevents hoarding).

**Tracking:** Stored in Bbolt `identity` bucket:
```
invites_remaining: uint32
invites_used: uint32
invites_earned: uint32
```

### 2.3 Multi-Use Invites (Future)

**Use Case:** Anchor communities (e.g., 20-person Discord server migrating to MURMUR) need a single invite link.

**Design:** Inviter can mark 1 invite as multi-use with `remaining_uses = N` (e.g., 20).

**Cost:** Multi-use invites consume 3 single-use invites from quota.

**Tracking:** Each use decrements `remaining_uses` in the invite struct. Invite invalidated when `remaining_uses == 0` or `expires_timestamp` reached.

**Status:** Not implemented in v0.1 — deferred to v0.2 based on anchor community feedback.

---

## 3. Invite Flow (User Journey)

### 3.1 Inviter Journey

**Step 1: Generate Invite**
- User navigates to Settings → Invites
- Sees available invite count (e.g., "5 invites remaining")
- Clicks "Create Invite"
- UI generates invite with friend-group context:
  - Automatically includes inviter's display name (if set)
  - Embeds public keys of inviter's top 10 contacts (by connection strength: mutual Waves, games played)
  - Sets 30-day expiration
- Invite displayed as:
  - **Text link:** `murmur://invite/<base64_data>` (copy button)
  - **QR code:** Scannable with phone camera (iOS/Android deep link)
  - **Share button:** Opens system share sheet (SMS, email, messaging apps)

**Step 2: Share Invite**
- User shares invite via preferred channel (text message, Signal, WhatsApp, email)
- Invite is self-contained — no MURMUR network required for sharing

**Step 3: Monitor Acceptance**
- User sees "Pending Invites" list in Settings → Invites
- Each invite shows: recipient (if accepted), created date, expiry date, status (pending/accepted/expired)
- When invitee accepts, inviter receives in-app notification: "Alice accepted your invite!"

### 3.2 Invitee Journey

**Step 1: Receive Invite**
- Invitee receives `murmur://invite/<data>` link via text/email/etc.
- Clicks link → opens MURMUR app (or prompts install if not present)

**Step 2: Preview Invite**
- App decodes invite, verifies Ed25519 signature
- Displays invite preview modal:
  - "You've been invited by [InviterName]"
  - Inviter's sigil (deterministic visual identity)
  - "This invite connects you with [InviterName] and their friend group ([N] people)"
  - Expiry countdown: "Valid for 23 days"
  - "Accept Invite" button (primary CTA)

**Step 3: Accept Invite**
- Invitee clicks "Accept Invite"
- App creates new identity (Ed25519 keypair, BIP-39 seed)
- **Critical:** Invitee's Pulse Map is **pre-seeded** with:
  - Inviter node (connected, labeled with display name)
  - Inviter's friend group (1-10 nodes, labeled "Alice's friends")
  - Edges between inviter and group members (visible social structure)
- Invitee sees onboarding Phase 2 (Welcome → Identity → **Pulse Map Tour**)
- Onboarding highlights: "Your friend [Inviter] is here, and so are [N] of their contacts. Explore their Waves or send your first message!"

**Step 4: First Interaction**
- Invitee sends first Wave to inviter (pre-filled recipient)
- Invitee explores Waves from inviter's group (discovery via weak ties)
- Invitee connects to 1-2 group members (UI prompts: "Alice is also friends with Bob and Charlie. Send them a Wave?")

---

## 4. Technical Implementation

### 4.1 Protobuf Definition

Add to `proto/identity.proto`:

```protobuf
message Invite {
  bytes invite_id = 1;
  bytes inviter_pubkey = 2;
  string inviter_display_name = 3;
  bytes inviter_sigil_seed = 4;
  repeated bytes group_member_keys = 5;
  int64 created_timestamp = 6;
  int64 expires_timestamp = 7;
  bytes signature = 8;
  uint32 remaining_uses = 9;
}

message InviteMetadata {
  bytes invite_id = 1;
  InviteStatus status = 2;
  int64 accepted_timestamp = 3;
  bytes invitee_pubkey = 4;  // Filled after acceptance
}

enum InviteStatus {
  PENDING = 0;
  ACCEPTED = 1;
  EXPIRED = 2;
  REVOKED = 3;
}
```

### 4.2 Storage Schema

**Bbolt Bucket:** `invites` (new bucket)

**Key-Value Pairs:**
- `invite:<invite_id>` → `Invite` protobuf (for verification and metadata)
- `inviter:<inviter_pubkey>` → `[]InviteMetadata` (list of invites created by user)
- `invitee:<invitee_pubkey>` → `invite_id` (which invite the user accepted)
- `quota:<pubkey>` → `InviteQuota` protobuf

```protobuf
message InviteQuota {
  uint32 remaining = 1;
  uint32 used = 2;
  uint32 earned = 3;
  int64 last_regeneration = 4;
}
```

### 4.3 Core Functions

**Package:** `pkg/identity/invites/`

**Functions:**
1. `GenerateInvite(inviterKeys *keys.KeyPair, groupMemberKeys []ed25519.PublicKey, displayName string) (*Invite, error)`
   - Creates invite with embedded friend-group context
   - Signs with inviter's Ed25519 key
   - Sets 30-day expiration (configurable)
   - Returns base64-encoded invite string

2. `VerifyInvite(inviteData string) (*Invite, error)`
   - Decodes base64 invite
   - Verifies Ed25519 signature
   - Checks expiration timestamp
   - Returns parsed `Invite` struct or error

3. `AcceptInvite(invite *Invite, inviteeKeys *keys.KeyPair) error`
   - Validates invite not expired/revoked
   - Decrements `remaining_uses` (if multi-use)
   - Stores invitee public key in `InviteMetadata`
   - Pre-seeds invitee's Pulse Map with inviter + group members
   - Notifies inviter via in-app event

4. `GetInviteQuota(pubkey ed25519.PublicKey) (*InviteQuota, error)`
   - Retrieves current invite quota from Bbolt
   - Returns `InviteQuota` struct

5. `DecrementQuota(pubkey ed25519.PublicKey) error`
   - Decrements `remaining` by 1
   - Increments `used` by 1
   - Called after successful `GenerateInvite()`

6. `EarnInvites(pubkey ed25519.PublicKey, amount uint32, reason string) error`
   - Increments `remaining` by `amount`
   - Increments `earned` by `amount`
   - Caps at 20 max remaining
   - Logs reason (Resonance milestone, active participation, etc.)

### 4.4 UI Integration

**Invite Creation Screen** (`pkg/ui/invite_creator.go`):
- Shows current quota: "You have 5 invites remaining"
- "Create Invite" button → generates invite
- Displays invite as: Text link (copy), QR code, Share button
- Lists "Pending Invites" with status

**Invite Acceptance Modal** (`pkg/ui/invite_acceptor.go`):
- Triggered by deep link `murmur://invite/<data>`
- Displays inviter sigil + name
- Shows friend-group preview: "Connect with Alice and 7 of her friends"
- "Accept Invite" button (primary CTA)
- "Decline" button (secondary)

**Pulse Map Pre-Seeding** (`pkg/pulsemap/layout/preseed.go`):
- `PreseedGraphFromInvite(invite *Invite) error`
  - Adds inviter node to graph (labeled with display name)
  - Adds group member nodes (labeled "Alice's friend")
  - Adds edges between inviter and group members
  - Positions nodes in force-directed layout
  - Returns control to onboarding Phase 5 (Pulse Map Tour)

---

## 5. Launch Sequence

### 5.1 Phase 1: Anchor Community Seeding (Weeks 1-4)

**Goal:** Seed 3-5 friend groups (5-8 users each) with invites.

**Process:**
1. **Recruit Anchor Groups:**
   - Reach out to privacy-focused friend groups (Signal groups, privacy subreddits, tech communities)
   - Target groups of 5-8 IRL friends who already communicate
   - Invite 1 "group leader" to be first user

2. **First User Setup:**
   - Group leader installs MURMUR, completes onboarding
   - Receives 5 default invites
   - Generates 4-7 invites (one per friend)
   - Shares invites via existing communication channel (Signal, Discord, etc.)

3. **Group Formation:**
   - Friends accept invites → see group leader's Pulse Map context
   - Friends connect to each other via UI prompts
   - Group plays first game (Cipher Puzzles or Phantom Gifts)
   - Group exchanges Waves throughout Week 1

4. **Validation:**
   - D7 retention ≥40% (4+ of 8 users active after 7 days)
   - ≥1 game started per group per week
   - ≥3 Waves per user per week
   - If metrics met → recruit next anchor group
   - If metrics not met → iterate on onboarding/UX before expanding

### 5.2 Phase 2: Inter-Group Bridging (Weeks 5-8)

**Goal:** Connect anchor groups via "connector" users.

**Process:**
1. **Identify Connectors:**
   - Users who are active in Anchor Group A and know someone in Anchor Group B
   - Example: Alice (Group A) is IRL friends with Bob (Group B)

2. **Bridge Formation:**
   - Alice invites Bob (using 1 invite)
   - Bob accepts invite → sees Alice's Group A context
   - Bob shares Group B context with Alice via mutual connection

3. **Discovery Unlocks:**
   - Users in Group A discover Waves from Group B via Alice's connection
   - Weak-tie discovery: "Alice's friend Bob posted about X"
   - Specter layer activates: Anonymous interactions across group boundaries

4. **Validation:**
   - ≥2 inter-group connections formed
   - Weak-tie Waves drive discovery (≥10% of content from non-direct connections)
   - Specter adoption ≥30% across both groups

### 5.3 Phase 3: Organic Growth (Weeks 9+)

**Goal:** Enable invite-driven viral growth without paid acquisition.

**Process:**
1. **Invite Regeneration:**
   - Active users earn additional invites via Resonance milestones
   - Example: User reaches Wraith (50 Resonance) → earns +2 invites

2. **Viral Loop:**
   - Each user invites 3-5 friends over 4 weeks
   - Viral coefficient target: **K = 1.5**
     - 1 user invites 3 friends → 3 new users
     - 3 new users each invite 2 friends → 6 new users
     - Exponential growth: 1 → 3 → 9 → 27 → 81 users over 8 weeks

3. **Quality Control:**
   - Invite quota caps prevent spam (max 20 invites per user)
   - Single-use default prevents invite farming
   - Resonance gates ensure high-quality participants (abusers don't earn invites)

4. **Network Effects:**
   - At N=50-100, Pulse Map becomes primary discovery tool
   - Anonymous layer thrives (Shadow Play, Phantom Councils unlock)
   - Network transitions from "friend groups" to "social graph"

---

## 6. Abuse Prevention

### 6.1 Threat Model

**Threat 1: Invite Farming**
- Adversary creates fake accounts to generate unlimited invites
- **Mitigation:** Single-use invites by default, Resonance gates for earning invites, 20-invite cap

**Threat 2: Invite Spam**
- Adversary shares invites in public forums, attracts low-quality users
- **Mitigation:** Invite expiration (30 days), signature verification prevents tampering, inviter reputation tied to invitee behavior (if invitee is banned, inviter loses future invite regeneration)

**Threat 3: Social Graph Leakage**
- Adversary collects invites to map social graphs
- **Mitigation:** Only public keys shared (no private content), max 10 contacts per invite, friend-group context is opt-in (inviter chooses to share)

**Threat 4: Sybil Attacks**
- Adversary creates many fake accounts to manipulate Resonance
- **Mitigation:** PoW on Waves (2-5 seconds per Wave), Resonance requires interactions with diverse contacts (hard to fake), peer scoring in GossipSub

### 6.2 Monitoring

**Metrics to Track:**
- **Invite acceptance rate:** % of invites accepted within 7 days (target: ≥50%)
- **D7 retention of invitees:** % of invitees active 7 days after acceptance (target: ≥40%)
- **Inviter reputation:** Average D7 retention of all invitees per inviter (flag inviters with <20%)
- **Invite velocity:** Invites generated per user per week (flag users generating >5/week without Resonance milestones)

**Red Flags:**
- Inviter with <20% invitee D7 retention → suspend invite regeneration
- User generating >10 invites/week → manual review
- Invite shared in public forum (detected via DHT lookup spikes) → revoke invite

---

## 7. Success Metrics

### 7.1 Invite Funnel

| Stage | Metric | Target |
|-------|--------|--------|
| **Generated** | Invites created per user | ≥3 within first 2 weeks |
| **Shared** | Invites shared externally (copy/QR/share) | ≥80% of generated |
| **Delivered** | Invites received by invitee | ≥90% of shared |
| **Accepted** | Invites accepted within 7 days | ≥50% of delivered |
| **Active** | Invitees active 7 days after acceptance (D7 retention) | ≥40% of accepted |

### 7.2 Viral Coefficient

**Target:** K = 1.5 (each user invites 1.5 active users over 4 weeks)

**Calculation:**
```
K = (Invites accepted / User) × (D7 retention of invitees) × (Viral loop length in weeks / 4)
K = (3 / 1) × 0.40 × 1.0 = 1.2 (baseline, needs improvement)
K = (5 / 1) × 0.50 × 1.0 = 2.5 (ideal, sustainable growth)
```

**Levers to Increase K:**
1. Increase invite acceptance rate (target: 50% → 60%)
2. Increase D7 retention of invitees (target: 40% → 50%)
3. Increase invites generated per user (target: 3 → 5)

### 7.3 Network Growth

**Target:** 50-100 active users within 8 weeks of launch.

**Trajectory:**
- **Week 1:** 3 anchor groups × 8 users = 24 users
- **Week 2:** 24 users × 0.3 invites/user × 0.5 acceptance = 24 + 4 = 28 users
- **Week 4:** 28 users × 0.5 invites/user × 0.5 acceptance = 28 + 7 = 35 users
- **Week 6:** Inter-group bridging accelerates growth → 50 users
- **Week 8:** Viral coefficient K=1.5 sustains growth → 75 users
- **Week 12:** Network effects kick in → 100+ users

**Reality Check:** Assumes 40% D7 retention and 50% invite acceptance. If either metric drops below target, growth stalls.

---

## 8. Implementation Checklist

### 8.1 Backend (pkg/identity/invites/)

- [ ] `invites.go` — Core invite generation, verification, acceptance
- [ ] `quota.go` — Quota management (remaining, used, earned)
- [ ] `storage.go` — Bbolt bucket CRUD operations
- [ ] `regeneration.go` — Invite earning logic (Resonance milestones, active participation)
- [ ] `preseed.go` — Pulse Map pre-seeding from invite friend-group context

### 8.2 Protobuf

- [ ] `proto/identity.proto` — Add `Invite`, `InviteMetadata`, `InviteQuota`, `InviteStatus` messages
- [ ] Regenerate Go code: `protoc --go_out=. proto/identity.proto`
- [ ] Check in `proto/identity.pb.go`

### 8.3 UI

- [ ] `pkg/ui/invite_creator.go` — Invite creation screen (text link, QR code, share button)
- [ ] `pkg/ui/invite_acceptor.go` — Invite acceptance modal (preview, accept/decline)
- [ ] `pkg/ui/invite_list.go` — Pending/accepted invites list in Settings
- [ ] Deep link handler: Register `murmur://invite/<data>` scheme with OS

### 8.4 Onboarding Integration

- [ ] Update Phase 5 (Exploration) to highlight pre-seeded Pulse Map: "Your friend Alice is here!"
- [ ] Add connection prompts: "Alice is also friends with Bob. Send Bob a Wave?"
- [ ] Tutorial hint: "Invites connect you with friend groups, not isolated individuals"

### 8.5 Testing

- [ ] Unit tests: Invite generation, signature verification, expiration handling
- [ ] Integration tests: Invite acceptance flow, Pulse Map pre-seeding
- [ ] E2E tests: Generate invite → share → accept → validate graph structure
- [ ] Abuse tests: Expired invites, tampered signatures, quota exhaustion

### 8.6 Documentation

- [ ] Update ONBOARDING.md with invite acceptance flow
- [ ] Update PRODUCT_IDENTITY.md with invite-first growth strategy
- [ ] Update VIRAL_GROWTH_AND_ONBOARDING.md with detailed invite mechanics
- [ ] Add "Invites FAQ" to docs/FAQ.md

---

## 9. Open Questions (For Team Discussion)

### 9.1 Multi-Use Invites

**Question:** Should v0.1 support multi-use invites for anchor communities?

**Options:**
- **Yes:** Enables easier anchor community onboarding (1 link for 20-person Discord server)
- **No:** Adds complexity, single-use invites sufficient for initial launch

**Recommendation:** **Defer to v0.2**. Single-use invites validate core mechanics first.

### 9.2 Invite Expiration Period

**Question:** Should invites expire after 30 days, or allow longer/shorter periods?

**Options:**
- 7 days: Fast expiration prevents stale invites, forces urgency
- 30 days: Default, balances urgency with flexibility
- 90 days: Long expiration for slow-moving friend groups

**Recommendation:** **30 days default, configurable by inviter**. UI slider: 7 days (urgent) → 30 days (default) → 90 days (flexible).

### 9.3 Inviter Reputation

**Question:** Should inviters be penalized if invitees churn or misbehave?

**Options:**
- **Yes:** Tie inviter reputation to invitee D7 retention. Inviters with <20% invitee retention lose future invite regeneration.
- **No:** Churn is natural, not inviter's fault.

**Recommendation:** **Yes, but lenient**. Flag inviters with <20% invitee retention after 10+ invites. Suspend invite regeneration (not quota) if pattern continues.

### 9.4 Public vs Private Invites

**Question:** Should invites be shareable publicly (e.g., Reddit post), or private-only?

**Options:**
- **Private-only:** Invites must be shared 1:1 or in small groups (Signal, Discord)
- **Public allowed:** Invites can be posted in public forums

**Recommendation:** **Private-only for v0.1**. Public invites enable spam, undermine friend-group seeding strategy. Revisit in v0.2 if growth stalls.

---

## 10. Conclusion

**Invite-First Launch Strategy:** Every new user receives **5 invites** carrying friend-group context (inviter + up to 10 contacts). This mitigates the cold-start problem by seeding Pulse Maps with cohesive social graphs rather than isolated individuals.

**Key Benefits:**
- ✅ **Pre-seeded Pulse Map:** Invitees see inviter's social structure immediately, not an empty graph
- ✅ **Controlled growth:** 5-invite quota + Resonance gates ensure high-quality participants
- ✅ **Viral loop:** Invite regeneration (Resonance milestones, active participation) drives organic growth
- ✅ **Abuse resistance:** Single-use invites, expiration, signature verification prevent spam/farming

**Launch Timeline:**
- **Weeks 1-4:** Seed 3-5 anchor groups (24+ users)
- **Weeks 5-8:** Inter-group bridging (50+ users)
- **Weeks 9+:** Organic growth via invite regeneration (100+ users by Week 12)

**Success Criteria:**
- D7 retention ≥40% for invitees
- Invite acceptance rate ≥50%
- Viral coefficient K ≥1.5
- Network growth to 50-100 active users within 8 weeks

**Next Actions:**
1. ✅ **Task 9.1 Complete:** Minimum Viable Network defined (5-8 users)
2. ✅ **Task 9.2 Complete:** This document (Invite-First Launch Plan)
3. ⏭️ **Task 9.3:** Recruit 3-5 anchor communities (5-8 users each)
4. ⏭️ **Task 9.4:** Define success metrics (D7/D30 retention, games/week, Specter adoption)
5. ⏭️ **Task 9.5:** Draft open beta plan with kill-switch for security issues

**Status:** ✅ **Ready for implementation sprint and anchor community recruitment**

---

**Document Owner:** MURMUR Core Team  
**Review Cadence:** After each anchor community cohort (Weeks 2, 4, 8)  
**Revisions:** Update based on invite acceptance rate and D7 retention data  
**References:** PLAN.md (Phase 9), docs/MINIMUM_VIABLE_NETWORK.md, PRODUCT_IDENTITY.md, VIRAL_GROWTH_AND_ONBOARDING.md  
