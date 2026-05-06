# MURMUR Success Metrics

**Document Version:** 1.0  
**Last Updated:** 2026-05-06  
**Status:** v0.1 Launch Planning  
**Prerequisite:** Minimum Viable Network Analysis (docs/MINIMUM_VIABLE_NETWORK.md), Invite-First Launch Plan (docs/INVITE_FIRST_LAUNCH.md)

---

## Executive Summary

MURMUR measures success through **retention, engagement quality, and privacy adoption** — not vanity metrics like DAU, follower counts, or time-on-platform. This document defines the core metrics framework aligned with Design Principle #6 ("No metrics") and the project's anti-surveillance philosophy.

**Core Philosophy:** If users return voluntarily, play games with friends, and adopt anonymous mechanics, the product is succeeding. If users stay because of algorithmic addiction, we've failed.

---

## 1. Primary Success Metrics

### 1.1 D7 Retention (7-Day Retention)

**Definition:** Percentage of users who return to the app 7 days after account creation.

**Formula:**
```
D7 Retention = (Users active on Day 7) / (Users who created account on Day 0) × 100%
```

**Target:** ≥40%

**Rationale:**
- **Friend-group dynamics:** If a user's friend group is active, they return. If the group churns, the user churns.
- **Product-market fit signal:** 40% D7 is exceptional for social apps without algorithmic engagement.
- **Comparison:** Instagram D7 ~45%, Twitter D7 ~30%, average social app D7 ~25%.

**Segmentation:**
- D7 for invitees (accepted invite) vs organic users (no invite)
- D7 by inviter (track inviter quality: inviters with <20% invitee D7 lose invite regeneration)
- D7 by anchor community (identify high-performing groups)

**Data Collection:**
- Store in Bbolt `identity` bucket: `created_timestamp`, `last_active_timestamp`
- Compute daily via background job
- Aggregate by cohort (e.g., "Week 2 anchor group")

---

### 1.2 D30 Retention (30-Day Retention)

**Definition:** Percentage of users active 30 days after account creation.

**Formula:**
```
D30 Retention = (Users active on Day 30) / (Users who created account on Day 0) × 100%
```

**Target:** ≥30%

**Rationale:**
- **Long-term engagement:** D30 measures whether product delivers sustained value.
- **Resonance progression:** Users reaching 30 days can achieve Wraith milestone (Resonance 50), unlocking advanced mechanics.
- **Churn indicator:** If D30 < 30%, product isn't sticky — users try it but don't adopt.

**Segmentation:**
- D30 by Resonance tier (Shade vs Wraith vs Phantom)
- D30 by Specter adoption (users with ≥1 Specter identity)
- D30 by game participation (users who played ≥1 game)

**Data Collection:**
- Same as D7, computed monthly
- Track via cohort analysis (e.g., "January 2027 cohort: 35% D30 retention")

---

### 1.3 Games Started per Week per User

**Definition:** Average number of games started per active user per week.

**Formula:**
```
Games/Week/User = (Total games started in week) / (Active users in week)
```

**Target:** ≥0.5

**Rationale:**
- **Engagement quality:** Users who play games are actively engaging with friends, not passively scrolling.
- **Friend-group health:** Games require coordination — if users play games, their friend group is cohesive.
- **Differentiation:** Games are MURMUR's unique value — this metric validates the product thesis.

**Benchmark:**
- ≥0.5 = Good (users play a game every 2 weeks)
- ≥1.0 = Excellent (users play 1+ games per week)
- <0.3 = Problem (users not discovering or enjoying games)

**Segmentation:**
- Games/week by game type (Cipher Puzzles vs Phantom Gifts vs Shadow Play)
- Games/week by Resonance tier (Shade users can't access all games)
- Games/week by network size (N=5 vs N=20 vs N=50)

**Data Collection:**
- Store in Bbolt `mechanics` bucket: `game_id`, `started_timestamp`, `participants`
- Aggregate weekly via background job
- Track game completion rate (started vs completed)

---

## 2. Secondary Engagement Metrics

### 2.1 Weekly Active Waves per User

**Definition:** Average number of Waves published per active user per week.

**Formula:**
```
Waves/Week/User = (Total Waves published in week) / (Active users in week)
```

**Target:** ≥3

**Rationale:**
- **Core product loop:** Waves are ephemeral messages — the primary communication primitive.
- **Health indicator:** If users publish <3 Waves/week, they're not using MURMUR for conversation.

**Benchmark:**
- ≥5 = Excellent (users actively conversing)
- 3-4 = Good (occasional use)
- <3 = At-risk (users not engaging)

**Segmentation:**
- Waves/week by Wave type (Surface vs Veiled vs Specter)
- Waves/week by network size (N=5 vs N=20)
- Waves/week by privacy mode (Open vs Hybrid vs Guarded vs Fortress)

**Data Collection:**
- Store in Bbolt `waves` bucket: `wave_id`, `author_pubkey`, `timestamp`
- Aggregate weekly
- Track Wave types distribution (Surface 60%, Veiled 20%, Specter 20% target)

---

### 2.2 Specter Adoption Rate

**Definition:** Percentage of active users who have created ≥1 Specter identity.

**Formula:**
```
Specter Adoption = (Users with ≥1 Specter) / (Total active users) × 100%
```

**Target:** ≥30%

**Rationale:**
- **Anonymous layer validation:** Specters are MURMUR's unique differentiator. If <30% adopt, anonymous layer isn't compelling.
- **Privacy indicator:** Users who create Specters value privacy/anonymity beyond basic encryption.

**Benchmark:**
- ≥40% = Excellent (anonymous layer thriving)
- 30-39% = Good (significant adoption)
- <30% = Problem (anonymous layer underutilized)

**Segmentation:**
- Specter adoption by privacy mode (Hybrid vs Guarded vs Fortress)
- Specter adoption by network size (N=5 vs N=20 — larger networks drive anonymity demand)
- Specter adoption by Resonance tier (Phantom users more likely to adopt)

**Data Collection:**
- Store in Bbolt `identity` bucket: `specter_identities` (list of Specter public keys)
- Count users with len(specter_identities) > 0
- Track Specter Wave count (validates active use, not just creation)

---

### 2.3 Resonance Progression Rate

**Definition:** Distribution of active users across Resonance milestones.

**Milestones:**
- Shade (25) — Basic mechanics unlocked
- Wraith (50) — Intermediate mechanics
- Shade-Wraith (75) — Advanced mechanics
- Phantom (100) — Full mechanics
- Council-Eligible (200) — Shadow Play, Phantom Councils
- Abyss (500) — Fortress-exclusive

**Target Distribution (at 30 days):**
- <25 (New Users): 30%
- 25-49 (Shade): 35%
- 50-99 (Wraith/Shade-Wraith): 25%
- 100-199 (Phantom): 8%
- 200+ (Council/Abyss): 2%

**Rationale:**
- **Engagement ladder:** Resonance progression indicates sustained engagement.
- **Milestone unlocks:** Users progressing to Phantom (100) unlock full game library.

**Data Collection:**
- Store in Bbolt `resonance` bucket: `pubkey` → `score`
- Aggregate distribution weekly
- Track progression velocity (days to Shade, days to Wraith, days to Phantom)

---

## 3. Anti-Metrics (What We Do NOT Track)

### 3.1 Daily Active Users (DAU)

**Why Not:**
- **Addiction signal:** High DAU often means algorithmic manipulation, not genuine value.
- **Platform metric:** DAU optimizes for engagement-time, not conversation quality.

**What We Track Instead:** D7/D30 retention (voluntary return, not compulsive use).

---

### 3.2 Follower Counts / Social Graph Size

**Why Not:**
- **Vanity metric:** Follower counts drive influencer dynamics, not peer relationships.
- **Privacy violation:** Revealing follower counts exposes social graph metadata.

**What We Track Instead:** Network size in aggregate (e.g., "Average user has 8 connections") without individual exposure.

---

### 3.3 Time-on-Platform / Session Duration

**Why Not:**
- **Engagement trap:** Long sessions can mean addictive doomscrolling, not value.
- **Design misalignment:** MURMUR's ephemeral Waves discourage infinite scroll.

**What We Track Instead:** Waves/week, games/week (quality engagement, not time spent).

---

### 3.4 Likes / Hearts / Engagement Counts

**Why Not:**
- **Popularity contest:** Engagement counts turn conversation into performance.
- **Anxiety driver:** Users optimize for reactions, not authentic expression.

**What We Track Instead:** Reply chains, game participation (actual interaction, not validation-seeking).

---

## 4. Privacy-Preserving Metrics

### 4.1 Data Collection Philosophy

**Principle:** Collect only metrics required to validate product health. No user-level tracking, no cross-device fingerprinting, no third-party analytics.

**Technical Implementation:**
- **Local-first:** Metrics computed on-device, aggregated anonymously
- **No user IDs:** Public keys hashed with salt before aggregation (k-anonymity)
- **No IP tracking:** Network-level metrics exclude IP addresses
- **No behavioral profiles:** Aggregate metrics only (e.g., "30% of users play games" not "Alice played 3 games")

### 4.2 Aggregate-Only Reporting

**Public Dashboard Metrics:**
- Total active users (count, no identities)
- D7/D30 retention (cohort averages, no individuals)
- Games/week per user (average, no names)
- Specter adoption rate (percentage, no identities)
- Resonance distribution (histogram, no users)

**Never Published:**
- Individual user activity
- Social graph structure (who-connects-to-whom)
- Wave content or metadata
- Game transcripts or outcomes
- Resonance scores per identity

### 4.3 Differential Privacy (Future)

**Status:** Not implemented in v0.1, planned for v0.2+.

**Goal:** Add calibrated noise to aggregated metrics to prevent inference attacks.

**Example:** If 100 users are active, report "~102 users" with Laplace noise (ε=1.0).

**Reference:** Differential Privacy for Social Graphs (Sala et al., 2011)

---

## 5. Red Flags & Alert Thresholds

### 5.1 Churn Alerts

| Metric | Threshold | Action |
|--------|-----------|--------|
| D7 retention | <35% | Investigate onboarding friction, anchor community health |
| D30 retention | <25% | Product not sticky — prioritize engagement features |
| Waves/week | <2 per user | Core loop broken — messaging UX issue or empty network |
| Games/week | <0.3 per user | Games not discoverable or not fun — UX audit required |

### 5.2 Growth Stagnation

| Metric | Threshold | Action |
|--------|-----------|--------|
| Invite acceptance rate | <40% | Invites not compelling or targeting wrong users |
| Viral coefficient K | <1.0 | Growth unsustainable — improve invite mechanics or product value |
| Network size | <50 users after 8 weeks | Anchor community seeding failed — recruit more groups |

### 5.3 Privacy Mode Adoption

| Metric | Threshold | Action |
|--------|-----------|--------|
| Specter adoption | <20% | Anonymous layer not compelling — improve onboarding or mechanics |
| Fortress mode | <5% | High-privacy mode unused — validate threat model or UX |
| Veiled Waves | <10% of total | Privacy modes underutilized — add education/nudges |

---

## 6. Metrics Dashboard (v0.2 Feature)

### 6.1 Public Dashboard

**URL:** `https://murmur.network/metrics` (future)

**Content:**
- Weekly active users (count)
- D7/D30 retention (line chart, 8-week trailing)
- Games/week per user (bar chart by game type)
- Specter adoption rate (percentage, trend line)
- Resonance distribution (histogram)

**Privacy:** All metrics aggregated, no user-level data, no cross-site tracking.

### 6.2 Internal Dashboard (Team-Only)

**Content:**
- All public metrics + segmentation (by cohort, anchor group, invite source)
- Churn analysis (why users leave: empty network, no games, friction)
- Invite funnel (generated → shared → accepted → D7 retention)
- Game completion rates (started vs finished)
- Resonance progression velocity (days to milestones)

**Access:** Local-only (no cloud analytics service), read-only for non-engineering team.

---

## 7. Success Criteria for v0.1 Launch

### 7.1 Anchor Community Phase (Weeks 1-4)

| Metric | Target | Status |
|--------|--------|--------|
| Anchor groups seeded | 3-5 groups | TBD |
| Users onboarded | 24+ (8 per group) | TBD |
| D7 retention | ≥40% | TBD |
| Games/week per user | ≥0.5 | TBD |
| Waves/week per user | ≥3 | TBD |
| Specter adoption | ≥30% | TBD |

**Go/No-Go Decision:** If ≥2 of 6 metrics met after Week 4, proceed to inter-group bridging. If <2 metrics met, iterate on onboarding/UX.

### 7.2 Inter-Group Bridging (Weeks 5-8)

| Metric | Target | Status |
|--------|--------|--------|
| Total active users | 50+ | TBD |
| Inter-group connections | ≥2 bridges | TBD |
| Weak-tie Waves | ≥10% of content | TBD |
| D7 retention (cohort 2) | ≥40% | TBD |
| Viral coefficient K | ≥1.2 | TBD |

**Go/No-Go Decision:** If K ≥1.2 and D7 ≥40%, enable organic growth phase. If K <1.0, invite mechanics need improvement.

### 7.3 Organic Growth (Weeks 9-12)

| Metric | Target | Status |
|--------|--------|--------|
| Total active users | 100+ | TBD |
| D30 retention | ≥30% | TBD |
| Games/week per user | ≥0.5 | TBD |
| Specter adoption | ≥30% | TBD |
| Resonance 100+ users | ≥5% | TBD |
| Viral coefficient K | ≥1.5 | TBD |

**Public Beta Ready:** If all 6 metrics met, proceed to open beta (Task 9.5). If <4 metrics met, extend closed beta.

---

## 8. Metrics Philosophy Statement (Public)

### 8.1 For Users

**Title:** "Why MURMUR Doesn't Track You"

**Content:**
> Most social platforms measure success by how long you stay, not whether you're happy. MURMUR is different.
>
> We track:
> - **D7/D30 retention:** Do you come back because the app is valuable?
> - **Games played per week:** Are you having fun with friends?
> - **Specter adoption:** Do you value anonymity?
>
> We don't track:
> - **Daily active users:** We don't care if you're here every day. We care if you're here by choice.
> - **Time-on-platform:** Longer sessions ≠ better experience. We want you to connect, then log off.
> - **Follower counts:** Your social graph is private. No influencer dynamics, no popularity contest.
> - **Likes or engagement:** Content that resonates generates conversation. Content that doesn't fades quietly.
>
> All metrics are aggregated and anonymized. We don't track individuals, build behavioral profiles, or sell data.

### 8.2 For Contributors

**Principle:** Every new metric must answer: "Does this help users, or manipulate them?"

**Decision Framework:**
1. **Does this metric measure voluntary engagement?** (Yes → consider, No → reject)
2. **Can this metric be computed locally without centralized tracking?** (Yes → consider, No → reject)
3. **Does this metric incentivize authenticity or performance?** (Authenticity → consider, Performance → reject)
4. **Would users be comfortable if we published this metric?** (Yes → consider, No → reject)

**Examples:**
- ✅ D7 retention: Measures voluntary return → Good metric
- ❌ Average session duration: Measures compulsive use → Bad metric
- ✅ Games/week: Measures quality engagement → Good metric
- ❌ Likes per Wave: Measures popularity contest → Bad metric

---

## 9. Implementation Checklist

### 9.1 Backend (pkg/metrics/)

- [ ] `retention.go` — Compute D7/D30 retention from Bbolt `identity` bucket
- [ ] `engagement.go` — Compute Waves/week, games/week from `waves` and `mechanics` buckets
- [ ] `adoption.go` — Compute Specter adoption, Resonance distribution from `identity` and `resonance` buckets
- [ ] `aggregator.go` — Aggregate metrics, apply k-anonymity (hash public keys with salt)
- [ ] `dashboard.go` — Export metrics for internal dashboard (team-only)

### 9.2 Storage

- [ ] Bbolt bucket: `metrics` — Store aggregated metrics per week/month
- [ ] Schema: `metrics:weekly:<YYYY-Wnn>` → `AggregatedMetrics` protobuf
- [ ] Protobuf: Add `AggregatedMetrics` message with retention, engagement, adoption fields

### 9.3 Monitoring

- [ ] Background job: Compute metrics weekly (cron-like scheduler)
- [ ] Alert system: Check red-flag thresholds, notify team if breached
- [ ] Logging: Write metrics to JSON files for team analysis (no cloud upload)

### 9.4 Documentation

- [ ] Update PRODUCT_IDENTITY.md with metrics philosophy
- [ ] Update VIRAL_GROWTH_AND_ONBOARDING.md with success criteria
- [ ] Add "Metrics FAQ" to docs/FAQ.md
- [ ] Create internal runbook: "How to Interpret Metrics Dashboard"

---

## 10. Conclusion

**MURMUR Success Metrics:** Retention (D7 ≥40%, D30 ≥30%), engagement quality (games/week ≥0.5, Waves/week ≥3), and privacy adoption (Specter adoption ≥30%).

**Anti-Metrics:** No DAU, no follower counts, no time-on-platform, no engagement vanity metrics.

**Philosophy:** If users return voluntarily, play games with friends, and adopt anonymous mechanics, the product succeeds. Algorithmic engagement is failure.

**Privacy:** All metrics aggregated and anonymized. No user-level tracking, no behavioral profiles, no data sales.

**Next Actions:**
1. ✅ **Task 9.1 Complete:** Minimum Viable Network defined (5-8 users)
2. ✅ **Task 9.2 Complete:** Invite-First Launch Plan created
3. ⏭️ **Task 9.3:** Recruit 3-5 anchor communities (5-8 users each)
4. ✅ **Task 9.4 Complete:** This document (Success Metrics)
5. ⏭️ **Task 9.5:** Draft open beta plan with kill-switch for security issues

**Status:** ✅ **Ready for anchor community recruitment and metrics implementation**

---

**Document Owner:** MURMUR Core Team  
**Review Cadence:** Monthly (after each cohort launch)  
**Revisions:** Update based on actual anchor community data  
**References:** PLAN.md (Phase 9), docs/MINIMUM_VIABLE_NETWORK.md, docs/INVITE_FIRST_LAUNCH.md, PRODUCT_IDENTITY.md  
