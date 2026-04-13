# Resonance System

**Category:** Core Mechanics
**Version:** 0.4
**Status:** Draft

---

## Overview

Resonance is MURMUR's reputation and influence metric. It is a per-node score that reflects a user's contribution to the network through connection-building, content creation, amplification behavior, and infrastructure participation. Resonance is computed locally by each node from observable network data — there is no central authority that assigns or validates Resonance scores. Every node computes the Resonance of every other node it can see, using the same deterministic algorithm applied to locally available data.

Resonance exists on both the Surface Layer (main identity Resonance) and the Anonymous Layer (Specter Resonance). The two scores are completely independent. A user's main identity Resonance and their Specter Resonance have no mathematical or cryptographic relationship. They accrue separately, from different activities, on different layers.

Resonance is not currency. It cannot be transferred, spent, or traded. It is a visible, verifiable measure of a node's history and influence within the network as perceived by the observing node's local view of the topology.

---

## Surface Layer Resonance

### Input Signals

Surface Layer Resonance is computed from the following observable signals, weighted and combined into a single scalar score.

**Connection Count.** The number of active, mutual connections the node maintains. This signal has diminishing returns — the first 20 connections contribute significantly more per-connection than connections beyond 100. The diminishing return curve follows a logarithmic function: `connection_score = 10 * ln(1 + connection_count)`.

**Connection Diversity.** The number of distinct clusters the node's connections span. A node connected to 50 people all in the same tightly-knit cluster scores lower on diversity than a node connected to 30 people across 5 distinct clusters. Diversity is computed using the cluster detection algorithm (Louvain method applied to the local topology graph). Diversity score is the count of distinct cluster IDs among the node's direct connections, normalized against the total number of clusters visible in the local topology.

**Wave Output.** The number of Waves the node has authored in the trailing 30-day window. Diminishing returns apply: `wave_score = 8 * ln(1 + wave_count_30d)`.

**Amplification Received.** The total number of amplifications the node's Waves have received from distinct amplifiers in the trailing 30-day window. This is the strongest signal of content quality — it reflects how many unique nodes chose to pass the content along. `amp_received_score = 15 * ln(1 + distinct_amplifier_count_30d)`.

**Amplification Given.** The number of distinct Waves from distinct authors the node has amplified in the trailing 30-day window. This rewards curation behavior — nodes that find and amplify valuable content from others. `amp_given_score = 5 * ln(1 + distinct_amplified_waves_30d)`.

**Bridge Activity.** For Hybrid, Guarded, and Fortress nodes, the volume of cross-layer bridging activity the node performs (measured in bridged messages per day, averaged over 30 days). Bridge activity is observable by peers that receive bridged content from the node. `bridge_score = 12 * ln(1 + avg_bridged_per_day_30d)`. This signal rewards infrastructure contribution — bridge nodes are essential for the Shadow Gradient to function.

**Account Age.** The number of days since the node's first observed appearance in the topology. This signal grows slowly and linearly, capping at 365 days. `age_score = min(days_since_first_seen / 365, 1.0) * 20`. This makes Sybil identities (which are necessarily young) score lower than established nodes.

**Uptime.** The fraction of the trailing 30 days during which the node was observed online (responding to heartbeats, publishing topology updates). `uptime_score = uptime_fraction_30d * 10`. This rewards nodes that contribute to network stability by being reliably available.

### Computation

The raw Resonance score is the sum of all input signal scores:

`raw_resonance = connection_score + diversity_score + wave_score + amp_received_score + amp_given_score + bridge_score + age_score + uptime_score`

The displayed Resonance score is the raw score rounded to the nearest integer. There is no upper bound, but the logarithmic scaling of most inputs means that scores grow slowly at high levels. A typical active user after 6 months might have a Resonance between 80 and 150. A highly connected, highly active bridge node after a year might reach 250–400. A new node starts at 0.

### Local Computation

Every node computes Resonance independently from its own local view of the network. Because different nodes have different local topology views (they see different subsets of the network depending on their position and connections), Resonance scores for the same target node may vary slightly between observers. This is by design — Resonance is a local assessment, not a global consensus value. In practice, well-connected nodes that are visible to many peers will have Resonance scores that converge across observers, while peripheral nodes may show more variance.

### Display

Resonance is displayed on profile cards, on the Pulse Map (node size scales subtly with Resonance — a node with Resonance 200 is noticeably larger than a node with Resonance 20, but the scaling is logarithmic to prevent visual clutter), and in Wave metadata (the author's Resonance at time of composition is embedded in the Wave for display purposes, though recipients compute their own assessment independently).

---

## Specter Resonance

### Input Signals

Specter Resonance is computed from Anonymous Layer activity using the same local-computation model. The input signals are adapted to anonymous mechanics.

**Specter Connection Count.** The number of active Specter-to-Specter connections. Same logarithmic scaling: `specter_connection_score = 10 * ln(1 + specter_connection_count)`.

**Specter Connection Diversity.** Cluster diversity among Specter connections, computed identically to Surface Layer diversity but applied to the anonymous topology.

**Specter Wave Output.** Anonymous Waves authored in the trailing 30-day window. Includes Specter Waves, Veiled Waves, Sigil Waves, and Abyssal Waves. `specter_wave_score = 8 * ln(1 + specter_wave_count_30d)`.

**Anonymous Amplification Received.** Distinct Specter amplifiers of the node's anonymous Waves in the trailing 30 days. `specter_amp_received_score = 15 * ln(1 + distinct_specter_amplifier_count_30d)`.

**Anonymous Amplification Given.** Distinct anonymous Waves from distinct Specters amplified in the trailing 30 days. `specter_amp_given_score = 5 * ln(1 + distinct_specter_amplified_waves_30d)`.

**Phantom Gift Volume.** The number of Phantom Gifts the Specter has sent in the trailing 30 days. Gifting is a costly social action (gifts are crafted from Specter Resonance milestones) and rewards generous participation. `gift_score = 6 * ln(1 + gifts_sent_30d)`.

**Masked Event Participation.** The number of Masked Events the Specter has participated in during the trailing 30 days. `event_score = 4 * ln(1 + events_participated_30d)`.

**Duel Record.** Net duel wins (wins minus losses) in the trailing 30 days, floored at 0. `duel_score = 7 * ln(1 + max(0, net_duel_wins_30d))`.

**Whisper Chain Contributions.** The number of Whisper Chain hops the Specter has served as an intermediate node or initiator in the trailing 30 days. `chain_score = 5 * ln(1 + chain_contributions_30d)`.

**ZK Claim Count.** The number of valid, unexpired ZK Claims attached to the Specter profile. Each claim represents a verified anonymous proof of some attribute. `zk_score = 3 * zk_claim_count`. Linear scaling because claims are discrete, costly to produce, and small in number.

**Shroud Node Operation.** For Fortress-mode Specters operating a Shroud Node: average uptime fraction over the trailing 30 days. `shroud_score = uptime_fraction_30d * 25`. This is the highest single-signal weight in the system, reflecting the significant infrastructure contribution of Shroud Node operation.

**Council Membership.** For Specters that are members of active Phantom Councils: a flat bonus per active Council. `council_score = 10 * active_council_count`. Phantom Council membership is exclusive and requires Fortress mode, making this a high-prestige signal.

**Specter Age.** Days since the Specter's first observed appearance in the anonymous topology. `specter_age_score = min(specter_days_since_first_seen / 365, 1.0) * 20`.

**Specter Uptime.** Fraction of the trailing 30 days during which the Specter Host was observed online. `specter_uptime_score = specter_uptime_fraction_30d * 10`.

### Computation

`raw_specter_resonance = specter_connection_score + specter_diversity_score + specter_wave_score + specter_amp_received_score + specter_amp_given_score + gift_score + event_score + duel_score + chain_score + zk_score + shroud_score + council_score + specter_age_score + specter_uptime_score`

The displayed Specter Resonance score is the raw score rounded to the nearest integer.

### Identity Independence

The Specter Resonance computation uses only data observable on the Anonymous Layer. No Surface Layer data is used. A Specter's Resonance reflects only its anonymous activity, connections, and contributions. An observer on the Surface Layer cannot infer a Specter's Resonance from the main identity's Resonance or vice versa.

A user who is highly active on the Surface Layer but inactive as a Specter will have high Surface Resonance and low Specter Resonance. A user who is primarily active as a Specter will show the reverse. This separation is fundamental to identity isolation.

---

## Resonance Milestones

Resonance milestones are thresholds that unlock cosmetic rewards and social mechanics. They exist independently on both layers.

### Surface Layer Milestones

**Resonance 10 — Ember.** First milestone. Unlocks a subtle warm glow effect on the user's Pulse Map node.

**Resonance 25 — Spark.** Unlocks a pulsing ring animation around the node.

**Resonance 50 — Flame.** Unlocks a particle trail effect on the node. The node visibly sheds small luminous particles.

**Resonance 100 — Blaze.** Unlocks a custom color palette for the node's glow (choice of 8 palettes).

**Resonance 200 — Inferno.** Unlocks an animated aura effect (slow-shifting color field surrounding the node).

**Resonance 500 — Corona.** Unlocks the most elaborate Surface Layer cosmetic: a multi-layered corona effect with procedural animation. Visually unmistakable on the Pulse Map.

### Specter Resonance Milestones

**Specter Resonance 10 — Whisper.** First anonymous milestone. Unlocks a faint ghostly trail on the Specter node.

**Specter Resonance 25 — Shade.** Unlocks the ability to send Phantom Gifts. The Specter gains access to a basic set of 5 cosmetic gift effects.

**Specter Resonance 50 — Wraith.** Unlocks an expanded gift set (10 effects) and a personal Specter sigil (procedurally generated from the Specter's public key hash, rendered as a small animated emblem on the Specter node).

**Specter Resonance 100 — Phantom.** Unlocks the premium gift set (20 effects, including the most visually elaborate cosmetics in the app). Unlocks the ability to place Specter Marks.

**Specter Resonance 200 — Revenant.** Unlocks a dense particle field around the Specter node. The Specter is visually prominent in the Anonymous Layer.

**Specter Resonance 500 — Abyss.** The highest Specter milestone. Unlocks a unique Kage shader effect on the Specter node: a slowly rotating void-like distortion field. Only Specters who have invested significant time and social effort in the anonymous layer reach this level. These nodes are visually unmistakable and carry enormous implicit social authority.

### Milestone Visibility

Surface Layer milestones and their cosmetic effects are visible to all users on the Pulse Map. Specter Resonance milestones and their cosmetic effects are visible on the Anonymous Layer to all Specter-capable users (Hybrid+), and as ghostly visual hints on the Surface Layer to Open-mode users (the Specter node's glow intensity and particle effects are faintly visible in the Anonymous Layer rendering on the Surface Layer Pulse Map, even though Open-mode users cannot interact with the Specter).

---

## Echo Index

The Echo Index is a per-cluster metric that measures the ideological diversity of amplification within a cluster. It is designed to make echo chambers visible and quantifiable.

### Definition

A cluster is detected by the Louvain community detection algorithm applied to the local topology graph. For each cluster, the Echo Index is computed as the ratio of intra-cluster amplification to total amplification originating from that cluster.

`echo_index = intra_cluster_amplifications / total_amplifications_from_cluster`

The Echo Index ranges from 0.0 to 1.0. A score of 1.0 means the cluster amplifies only content from within itself — a perfect echo chamber. A score of 0.0 means the cluster amplifies only content from outside itself — maximally outward-looking. In practice, most clusters will score between 0.4 and 0.8.

### Computation

The Echo Index is computed locally by each node for each cluster visible in its local topology. It is recomputed every 24 hours from the trailing 30-day amplification data. The computation examines every amplification event originating from a node within the cluster and categorizes it as intra-cluster (the original Wave author is also in the same cluster) or extra-cluster.

### Display

The Echo Index is displayed as a color-coded badge on cluster boundaries in the Pulse Map. High Echo Index clusters (above 0.7) are tinted warm (amber to red), indicating insularity. Low Echo Index clusters (below 0.4) are tinted cool (blue to green), indicating openness. Mid-range clusters are tinted neutral (gray to white).

The Echo Index is also displayed on individual profile cards as "Your cluster's Echo Index: [value]." This makes users aware of their cluster's amplification patterns without judgment — the metric is informational, not punitive.

### Echo Shadow

The Echo Shadow is the Anonymous Layer counterpart of the Echo Index. It measures the same intra-cluster vs. extra-cluster amplification ratio for Specter clusters in the anonymous topology. The computation is identical but applied to anonymous amplification data.

The Echo Shadow is displayed as a subtle color field around Specter clusters on the Anonymous Layer of the Pulse Map. It provides the same informational value for anonymous communities: are they talking mostly to themselves, or are they engaging with the broader anonymous network?

### Design Intent

The Echo Index and Echo Shadow are not gamified. There is no reward for low scores or penalty for high scores. They exist to make a normally invisible social phenomenon — the tendency of like-minded groups to amplify only like-minded content — visible and measurable. The belief is that awareness alone changes behavior: users who see their cluster turning red may choose to amplify more diverse content, not because the app tells them to, but because they can see the pattern.

---

## Resonance and Anonymous Mechanics

Several anonymous mechanics use Resonance as a gating or weighting mechanism. In every case, the Resonance used is Specter Resonance, not Surface Layer Resonance, preserving identity isolation.

### Phantom Gift Gating

Phantom Gifts are unlocked at Specter Resonance 25 (Shade milestone). The available gift set expands at Resonance 50 (Wraith) and 100 (Phantom). The most visually elaborate gifts — the ones that create the most striking effects on recipient nodes — require the highest Specter Resonance to send. This ensures that the most beautiful anonymous cosmetics come from the most invested anonymous participants.

### Specter Mark Gating

Specter Marks are unlocked at Specter Resonance 100 (Phantom milestone). Marks are a social power — the ability to place a visible, mysterious sigil on another user's node — and gating them behind significant Specter Resonance ensures that marks carry weight and are not trivially spammable.

### Masked Event Resonance Burst

During a Masked Event, all participants use single-use Masked keypairs that carry no Resonance history. However, after the event concludes, a Resonance Burst is computed for each participant based on the amplification their contributions received during the event. The Burst is a temporary Specter Resonance bonus (decaying over 7 days) awarded to the participant's Specter identity. The mapping between Masked keypair and Specter identity is handled locally by the participant's client — no other participant or observer can link a Masked keypair to a Specter.

The Resonance Burst leaderboard (anonymous, showing Masked keypair pseudonyms and Burst values) is included in the post-event Summary Wave. This leaderboard is a competitive element that rewards high-quality contributions during events without compromising the event's anonymity.

### Duel Resonance Stakes

Specter Duels have Resonance consequences. The winner gains a Resonance bonus proportional to the audience vote margin. The loser gains no bonus but does not lose Resonance. The audience voting mechanism is weighted: each voter's vote is weighted by their own Specter Resonance, so high-Resonance Specters have more influence on duel outcomes. This creates a meritocratic competitive mechanic where anonymous reputation has tangible social consequences.

### ZK Claims

ZK Claims allow a Specter to prove attributes about itself without revealing its identity. The most common ZK Claim type is a Resonance range proof: the Specter proves that its Resonance score falls within a certain range (e.g., "above 100," "above 500") without revealing the exact score or any identifying information.

ZK Claims use Pedersen commitments and Bulletproofs-style range proofs, implemented in Go. The Specter commits to its locally-computed Resonance score and generates a non-interactive zero-knowledge proof that the committed value satisfies the claimed range. Verifiers (any node) can check the proof against the commitment without learning the exact value.

ZK Claims are attached to the Specter's anonymous profile and are visible to all modes. An Open-mode user viewing a Specter node on the Pulse Map can see: "ZK Claim: Resonance > 100 (verified)." This is powerful: it demonstrates that anonymous identities in the network have real, verified standing without revealing who they are.

Additional ZK Claim types include: Specter Age range proof ("active for more than 90 days"), Ignition Count range proof ("has met more than 10 peers in person via Proximity Ignition"), and Masked Event participation count ("has participated in more than 5 events"). Each follows the same Pedersen commitment + range proof structure.

### Phantom Council Gating

Phantom Councils require Fortress mode to initiate and a minimum Specter Resonance of 200 to join. Council invitations include a ZK Claim challenge: the invitee must present a valid ZK Claim proving their Specter Resonance exceeds the Council's minimum threshold. This ensures that Councils are composed of established, high-reputation anonymous participants.

### Shroud Node Weighting

Shroud Nodes operated by higher-Resonance Specters receive preferential routing in the mix protocol — other nodes are more likely to route traffic through high-Resonance Shroud Nodes, on the theory that high-Resonance operators have more to lose from misbehavior and are therefore more trustworthy as infrastructure providers. This is a soft preference, not an absolute gate — all valid Shroud Nodes participate in the mix network regardless of Resonance.

---

## Resonance and Sybil Resistance

Resonance is a key component of MURMUR's Sybil resistance strategy. Sybil attacks — where an adversary creates many fake identities to manipulate the network — are countered by several properties of the Resonance system.

**Time cost.** The Account Age and Specter Age signals mean that new identities start with low Resonance regardless of activity. Building Resonance to meaningful levels takes weeks to months.

**Social cost.** The Connection Count and Diversity signals require forming genuine, mutual connections with real nodes. An adversary controlling many Sybil nodes can connect them to each other, but they will form a single cluster with low diversity scores and will be poorly connected to the legitimate network.

**Activity cost.** The Proof-of-Work requirement for Waves means that each Wave from each Sybil identity has a computational cost. An adversary running 1,000 Sybils that each produce 10 Waves per day must compute 10,000 PoW solutions per day.

**Amplification cost.** Amplification Received rewards content that many distinct nodes choose to amplify. An adversary can have Sybil nodes amplify each other's content, but unless those amplifications reach into the legitimate network, they affect only the Sybil cluster's internal Resonance scores — which are not visible or meaningful to nodes outside the cluster.

**Resonance-gated mechanics.** The most powerful anonymous mechanics (Specter Marks, high-tier Phantom Gifts, Phantom Council membership) require high Resonance that is costly for Sybils to achieve.

**ZK Claim verification.** Resonance-gated mechanics that use ZK Claims as entry requirements force an adversary to not only build Resonance but also generate valid proofs — which requires actually having the claimed Resonance, computed from observable network data that other nodes can independently assess.

No Sybil defense is perfect. A sufficiently resourced adversary who is willing to invest months of time, significant computational resources, and social engineering effort can build high-Resonance Sybil identities. The goal is to make the cost of Sybil attacks high enough that they are impractical for casual attackers and expensive for sophisticated ones.

---

## Proximity Ignition

Proximity Ignition is an in-person connection mechanic. When two MURMUR users are physically co-located, they can establish a connection by exchanging connection data over a local channel (QR code scan, NFC tap, or mDNS local discovery with mutual confirmation).

### Ignition Process

Both users open the Proximity Ignition interface. User A's device displays a QR code containing their node's public key, current IP/port (or relay address), and a one-time authentication token. User B scans the QR code with their device's camera. User B's device initiates a direct libp2p connection to User A using the scanned connection data. The one-time token authenticates the connection attempt, preventing spoofing. User A's device confirms the incoming connection and displays User B's profile for mutual confirmation. Both users confirm. The connection is established and both nodes record the Ignition event.

NFC and mDNS variants follow the same mutual-confirmation flow with different data exchange mechanisms.

### Ignition Resonance Bonus

Proximity Ignition connections carry a Resonance bonus. The first 10 Ignitions contribute a bonus of 3 Resonance each. Subsequent Ignitions contribute 1 each, up to a cap of 50 Ignitions (maximum 80 bonus Resonance from Ignitions). This bonus is added to the raw Resonance calculation.

The Ignition bonus rewards in-person network building, which is inherently Sybil-resistant (an adversary cannot easily manufacture in-person encounters at scale) and creates strong social bonds.

### Ignition Counter and ZK Claims

The user's Ignition count (total number of Proximity Ignitions completed) is tracked locally and can be used to generate a ZK Claim: "This Specter has completed more than N Proximity Ignitions." This is a powerful claim because it proves in-person social activity without revealing who the person met or linking the Specter to a main identity. Specter Duels and Phantom Councils may use Ignition count ZK Claims as additional entry requirements.

### Visual Treatment

Ignition connections are displayed on the Pulse Map with a distinctive visual treatment: a brighter, warmer-colored link with a small flame icon at the midpoint. This makes in-person connections visually distinguishable from purely digital connections.