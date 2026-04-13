# Anonymous Mechanics

**Category:** Core Mechanics — Anonymous Layer Social Features
**Version:** 0.5
**Status:** Draft

---

## Overview

Anonymous Mechanics are the social features exclusive to or primarily operating on the Anonymous Layer. They are the reason the Anonymous Layer exists as more than a privacy tool — they create a parallel social world with its own culture, rituals, competition, and generosity. Each mechanic is designed to be impossible or meaningless without anonymity, making the Anonymous Layer a place with experiences that the Surface Layer cannot replicate.

All Anonymous Mechanics use Specter identity unless otherwise noted. Participation requires Hybrid mode or higher. Several mechanics are gated by Specter Resonance milestones, ensuring that the most powerful anonymous social tools are earned through sustained anonymous participation.

The mechanics described in this document are: Phantom Gifts, Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play, Masked Events, Phantom Councils, Specter Marks, and Whisper Chains.

Additionally, this document describes mechanics that extend into underserved subsystems: Surface Sparks (Surface Layer), Cartographer's Trail (Discovery/Search), Echo Chains (Echo/Re-broadcast), Pulse Beats (Notifications/Vibrations), and Specter Trophies (Profile/Identity).

---

## Phantom Gifts

### Concept

Phantom Gifts are anonymous cosmetic gifts sent from a Specter to any node in the network — Surface Layer or Anonymous Layer. They are one-way gestures of generosity, recognition, or mystery. A Phantom Gift manifests as a temporary visual effect on the recipient's node in the Pulse Map, lasting 7 days before fading.

Phantom Gifts are the gentlest cross-layer mechanic: they allow anonymous users to reach into the visible world and leave a mark of kindness or acknowledgment without revealing themselves. They are also the primary driver of curiosity about the Anonymous Layer for Open-mode users, who receive gifts from invisible benefactors.

### Gift Tiers

Phantom Gifts are organized into three tiers, unlocked by Specter Resonance milestones.

**Basic Gifts (Specter Resonance 25 — Shade milestone).** A set of 5 subtle visual effects: a soft glow pulse, a faint halo ring, a gentle particle drift, a shimmer overlay, and a warmth tint shift. These effects are understated — they are noticeable on close inspection but do not dramatically alter the recipient's node appearance.

**Expanded Gifts (Specter Resonance 50 — Wraith milestone).** An additional 10 effects of moderate visual intensity. These include animated geometric patterns orbiting the recipient's node, color-shifting aurora effects, crystalline fracture patterns, ember trails, and rippling distortion fields. Expanded gifts are visually distinctive and clearly mark the recipient as having been noticed by an anonymous benefactor.

**Premium Gifts (Specter Resonance 100 — Phantom milestone).** An additional 20 effects of high visual intensity. These are the most elaborate cosmetics in the application: multi-layered particle systems, procedural fluid simulations rendered as node auras, complex geometric mandalas that slowly rotate and evolve, void-like gravitational distortion effects, and prismatic light refraction patterns. Premium gifts are visually spectacular and their presence on a node is immediately noticeable to anyone viewing the Pulse Map.

### Sending a Gift

The gifting flow begins when a Specter selects a target node on the Pulse Map (either a Surface Layer node or another Specter node). The client presents the available gift set based on the sender's Specter Resonance tier. The sender selects a gift effect and confirms.

The gift is encoded as a signed data structure containing the recipient's public key, the gift effect identifier, a timestamp, and the sender's Specter signature. The gift is broadcast on the Anonymous Layer gossip topic. Bridge nodes inject gifts targeting Surface Layer recipients into the Surface Layer gossip.

The recipient's node (and all observing nodes) validate the gift signature against the sender's Specter public key and render the visual effect on the recipient's node for 7 days. The sender's Specter pseudonym and sigil are attached to the gift and visible to the recipient and all observers.

### Gift Limits

A Specter can send a maximum of 3 Phantom Gifts per 24-hour period. This limit prevents gift spam and ensures that each gift carries social weight. The limit is enforced locally by the sender's client and validated by recipients (who reject gifts from Specters that have already sent 3 gifts in the current 24-hour window, as observable from the gossip stream).

### Cross-Layer Gift Display

When an Open-mode user receives a Phantom Gift, it appears on their Surface Layer profile with the visual effect and the label "Phantom Gift from [Specter pseudonym]." The recipient sees the Specter's pseudonym and sigil but cannot click through to a Specter profile (Open-mode users cannot interact with the Anonymous Layer). The gift is a one-way communication — the recipient cannot respond, thank the sender, or even verify the sender's existence.

When a Specter receives a Phantom Gift from another Specter, the interaction is bidirectional — the recipient can view the sender's Specter profile, send a gift in return, or initiate a Specter connection.

---

## Cipher Puzzles

### Concept

Cipher Puzzles are collaborative and competitive cryptographic challenges that Specters solve together or against each other. A Cipher Puzzle is a time-limited event where the network generates a puzzle — a cryptographic or pattern-based challenge derived from network state — and Specters race to solve it. Puzzles leverage anonymity by making the solver's identity irrelevant; only the solution matters.

Cipher Puzzles exist because anonymity enables pure intellectual competition. When identity is stripped from achievement, the focus shifts to the work itself. Puzzles formalize this dynamic into a structured, entertaining mechanic that rewards cryptographic intuition and collaborative problem-solving.

### Puzzle Types

**Fragment Puzzles (Competitive).** A Beacon Wave announces a puzzle containing a cryptographic fragment — a partial hash preimage, a ciphertext with a missing key byte, or a pattern-matching challenge derived from recent network entropy (e.g., "Find a nonce such that SHA-256(puzzle_seed || nonce) starts with 28 zero bits"). The first Specter to publish a valid solution Wave wins. Fragment Puzzles are individual competitions.

**Mosaic Puzzles (Collaborative).** A Beacon Wave announces a puzzle that requires multiple Specters to each contribute a piece. The puzzle is divided into N sub-problems (typically 3–7), each solvable independently. A Specter publishes a Puzzle Contribution Wave containing their sub-solution. When all sub-solutions are collected and combined, the full solution is verified. Mosaic Puzzles reward teamwork and coordination among anonymous participants.

**Cascade Puzzles (Sequential).** A chain of dependent puzzles where each solution unlocks the next. The first Specter to solve stage 1 publishes the solution, which reveals stage 2. Another Specter (or the same one) solves stage 2, revealing stage 3, and so on. Cascade Puzzles create a collaborative relay where the network collectively works through a multi-stage challenge.

### Puzzle Generation

Puzzles are generated deterministically from network entropy — the hash of recent Beacon Waves, the current epoch timestamp, and a puzzle-specific seed. Any node can independently verify that a puzzle was correctly generated from the declared inputs. No trusted authority is needed. Puzzle generation is encoded as a Go `PuzzleGenerator` interface:

```go
type PuzzleGenerator interface {
    Generate(seed []byte, epoch uint64) Puzzle
    Verify(puzzle Puzzle, solution []byte) bool
}
```

### Initiating a Puzzle

Any Specter with Resonance 50 or higher (Wraith milestone) can initiate a Cipher Puzzle by publishing a Puzzle Beacon Wave (type 0x08) with `beacon_type` set to `puzzle_announce`. The announcement includes `puzzle_id` (random 32-byte identifier), `puzzle_type` (fragment, mosaic, or cascade), `puzzle_seed` (the seed used to generate the puzzle), `puzzle_difficulty` (a difficulty parameter), and `puzzle_duration` (time limit in minutes: 15, 30, or 60).

### Solving and Rewards

Solutions are published as Specter Waves with metadata `puzzle_id` and `puzzle_solution` containing the solution bytes. Nodes validate solutions locally by running the puzzle's verification function. The first valid solution (by timestamp) wins a Fragment Puzzle. For Mosaic Puzzles, each contributor who provides a valid sub-solution earns a share of the reward.

The Resonance reward for puzzle participation follows the formula: `puzzle_bonus = 4 * ln(1 + difficulty_factor * participation_count)`. The bonus is temporary, decaying linearly to zero over 14 days. Participation (attempting a puzzle, even without winning) contributes to the Puzzle Activity signal in Specter Resonance.

### Puzzle Visibility

Active puzzles are highlighted on the Anonymous Layer Pulse Map as a glowing, rotating glyph at the approximate network centroid of participating Specters. The glyph pulses with increased intensity as solutions are submitted. Completed puzzles leave a brief celebratory particle burst at the winner's node.

---

## Specter Hunts

### Concept

Specter Hunts are network-wide, time-limited scavenger hunts across the Pulse Map. A Hunt scatters hidden fragments — cryptographic tokens embedded in the network topology — that Specters must discover by exploring the Pulse Map, decoding clues, and reaching specific network locations. Hunts leverage the spatial nature of the Pulse Map and the anonymity of the Anonymous Layer to create an exploration-driven game.

### Hunt Structure

A Hunt is initiated by a Specter with Resonance 75 or higher by publishing a Hunt Beacon Wave. The announcement includes `hunt_id`, `hunt_theme` (a short description), `hunt_duration` (30, 60, or 120 minutes), `hunt_fragment_count` (5–20 fragments to find), and a `hunt_seed` used for deterministic fragment placement.

Fragments are virtual tokens whose locations are determined by hashing the hunt seed with sequential fragment indices: `fragment_location = SHA-256(hunt_seed || fragment_index)`. The resulting hash is mapped to a region of the Pulse Map topology by XOR-distance to existing node Peer IDs. Each fragment is "near" a specific node, and the hunting Specter must navigate to that region of the Pulse Map and publish a Hunt Claim Wave while connected to a peer within 3 hops of the target node.

### Clue System

Each fragment has an associated clue published in the Hunt Beacon Wave's metadata. Clues are cryptographic hints: partial hashes of the target region, XOR-distance ranges, or encoded references to visible Pulse Map features (e.g., "Near the densest cluster on the eastern fringe"). Clues grow more specific over time — every 10 minutes, an additional hint Beacon Wave is published, narrowing the search space.

### Claiming Fragments

A Specter claims a fragment by publishing a Hunt Claim Wave: a Specter Wave with metadata `hunt_id`, `fragment_index`, and a proof-of-proximity — a signed attestation from a peer within 3 hops of the fragment's target node, or a gossip-observable proof that the claiming Specter recently exchanged messages with nodes near the target. The proof-of-proximity is verified locally by observing nodes.

### Rewards

The Specter who claims the most fragments wins the Hunt. The winner receives a Resonance bonus: `hunt_bonus = 5 * ln(1 + fragments_claimed)`. All participants who claim at least one fragment receive a smaller participation bonus: `participation_bonus = 2 * ln(1 + fragments_claimed)`. Bonuses decay over 14 days.

### Hunt Visibility

Active Hunts are shown on the Pulse Map as a constellation of dim, pulsing markers scattered across the topology. As fragments are claimed, their markers brighten and display the claimer's Specter sigil. The Hunt creates a visible wave of exploration activity — Specters navigating to unfamiliar regions of the map, briefly illuminating corners of the network they might never have visited.

---

## Territory Drift

### Concept

Territory Drift is a persistent, ambient game where Specters claim and contest regions of the Pulse Map through sustained activity. The Pulse Map is divided into dynamic territories based on the network's cluster structure. Specters accumulate influence over territories by publishing Waves, forming connections, and participating in mechanics within that region. Territory Drift makes the Anonymous Layer a living, contested landscape.

### Territory Definition

Territories are defined by the Louvain community detection algorithm applied to the Anonymous Layer topology. Each detected cluster constitutes a territory. Territories are dynamic — as the topology evolves, territories shift, merge, and split. Each territory has a centroid (the geometric center of its member nodes on the Pulse Map) and a boundary (the convex hull of its members' positions).

### Claiming Influence

A Specter accumulates influence over a territory by performing activity within it: publishing Waves that are amplified by nodes in the territory, forming connections with members of the territory, and participating in mechanics (puzzles, hunts, events) whose participants overlap with the territory. Influence is computed locally by each observer as: `influence = 8 * ln(1 + waves_amplified_in_territory_30d + connections_in_territory + mechanic_participations_in_territory)`.

### Territory Control

The Specter with the highest influence in a territory is its Controller. Controller status is computed locally and may vary slightly between observers. The Controller's Specter sigil is displayed as a subtle watermark in the territory's background on the Pulse Map. Controller status carries no protocol-level privilege — it is a visible social distinction, a mark of sustained anonymous presence in that region.

### Contest Mechanics

When two Specters have similar influence in a territory (within 20% of each other), the territory enters a Contested state displayed as a shimmering boundary with alternating sigil watermarks. Contested territories attract attention and activity, creating organic hotspots of anonymous competition.

### Resonance Integration

Territory Drift contributes to Specter Resonance through the Territory Influence signal: `territory_score = 3 * ln(1 + territories_controlled + 0.5 * territories_contested)`. Controlling or contesting territories rewards sustained participation in specific network regions. Territory Drift is available at Specter Resonance 25 (Shade milestone) — the mechanic is ambient and accessible early, giving new Specters an immediate sense of place and purpose.

---

## Oracle Pools

### Concept

Oracle Pools are anonymous, stake-free prediction markets where Specters forecast network events and earn Resonance for accuracy. An Oracle Pool poses a question about a future network-observable event, Specters submit predictions, and when the event resolves, accurate predictors earn Resonance rewards. Oracle Pools leverage anonymity to enable honest prediction without social pressure — your prediction is visible but your identity carries no reputational risk from being wrong.

### Pool Creation

A Specter with Resonance 100 or higher (Phantom milestone) creates an Oracle Pool by publishing a Pool Beacon Wave with `pool_id`, `pool_question` (UTF-8, max 256 bytes — the prediction question), `pool_resolution_method` (how the outcome is determined — must be a network-observable metric), `pool_deadline` (Unix timestamp for prediction submission cutoff), and `pool_resolution_time` (when the outcome is evaluated).

Pool questions must reference network-observable events: "Will daily gossip volume on murmur/anonymous/v1 exceed 10,000 messages on April 20?" or "Will a new territory form in the eastern cluster within 7 days?" or "Will Masked Event participation exceed 50 total participants this week?" Resolution is deterministic — any node can independently verify the outcome by observing the specified metric.

### Submitting Predictions

Specters submit predictions by publishing Pool Prediction Waves: Specter Waves with metadata `pool_id` and `pool_prediction` (a numeric value or boolean, depending on the pool type). Predictions are committed using a hash-then-reveal scheme: the Specter first publishes a commitment (SHA-256 hash of prediction + secret nonce), then after the prediction deadline, publishes the reveal (prediction + nonce). This prevents Specters from copying others' predictions.

### Resolution and Rewards

After the resolution time, each node locally evaluates the outcome by querying its own observable data. Predictions are scored by accuracy (exact match or closest to actual value). The top 25% of predictors by accuracy receive a Resonance bonus: `oracle_bonus = 3 * ln(1 + pool_participant_count / rank)`. The bonus decays over 14 days.

### Pool Visibility

Active Oracle Pools are displayed on the Pulse Map as floating question-mark glyphs near the network region relevant to the prediction (if applicable) or near the pool creator's node. Resolved pools display a brief results summary and the winning prediction's Specter sigil.

---

## Sigil Forge

### Concept

Sigil Forge events are timed creative challenges where Specters compete to produce the most compelling content within constraints. A Forge event provides a prompt, a time limit, and a medium (Sigil art, micro-fiction, remix chains), and Specters create and submit entries. The anonymous audience evaluates entries through amplification — the most amplified entry wins.

Sigil Forge exists because anonymity liberates creativity. Without identity attached to output, participants take creative risks they would avoid under their public persona. The Forge formalizes this into a structured event that produces visible creative artifacts.

### Forge Types

**Sigil Art Forge.** Participants create Sigil Waves — procedurally generated visual art seeded from a provided prompt. Entries are evaluated purely on visual appeal as measured by amplification count.

**Micro-Fiction Forge.** Participants publish short-form creative writing (max 2048 bytes) responding to a prompt. Entries are Specter Waves tagged with the forge event ID.

**Remix Chains.** A collaborative creative mechanic: the first participant publishes a seed creation, the next participant remixes it (responding with a derivative work), the next remixes that, and so on. The chain is evaluated as a whole, with all contributors sharing the reward.

### Forge Events

A Specter with Resonance 50 or higher (Wraith milestone) initiates a Sigil Forge by publishing a Forge Beacon Wave with `forge_id`, `forge_type`, `forge_prompt` (the creative prompt, max 256 bytes), `forge_duration` (30 or 60 minutes), and `forge_medium` (sigil_art, micro_fiction, or remix_chain).

### Evaluation and Rewards

Entries are evaluated by amplification. During the forge's duration, audience Specters amplify their favorite entries. After the forge concludes, each node locally tallies amplifications weighted by the amplifier's Resonance. The entry with the highest weighted amplification wins.

The winner receives: `forge_bonus = 4 * ln(1 + weighted_amplifications)`. All participants receive a smaller bonus: `participation_bonus = 2 * ln(1 + own_amplifications_received)`. Bonuses decay over 14 days.

### Forge Visibility

Active Forge events appear on the Pulse Map as a glowing anvil glyph. During a Sigil Art Forge, submitted Sigil Waves orbit the glyph as miniature animated thumbnails, creating a visible gallery in the network topology. The winning entry's Sigil is displayed prominently at the glyph's center for 24 hours after the event.

---

## Shadow Play

### Concept

Shadow Play is a social deduction game that leverages anonymity as its core mechanic. In a Shadow Play round, a small group of Specters are secretly assigned roles, and the group must identify the hidden roles through interaction, observation, and deduction — all while maintaining (or deliberately breaking) their anonymity. Shadow Play is the deepest mini-game mechanic, gated at Resonance 200 (Revenant milestone), and designed to be the Anonymous Layer's most compelling social experience.

### Game Setup

A Specter with Resonance 200 or higher initiates a Shadow Play by publishing a Play Beacon Wave. The announcement includes `play_id`, `play_duration` (30 or 60 minutes), and `play_size` (5–13 participants).

Participants join by publishing Join Waves (similar to Masked Event joining). Once the participant cap is reached, the initiator's client deterministically assigns roles from the play seed: `role_assignment = SHA-256(play_seed || participant_index)`. Each participant's client locally derives their own role from the seed and their position in the join order. Roles are not communicated over the network — each client computes them independently from shared inputs.

### Roles

**Echoes** (majority): Standard participants. Their goal is to identify the Shades through observation and interaction.

**Shades** (minority, typically 1–2): Hidden disruptors. Their goal is to remain undetected while subtly misdirecting the group. Shades know each other's identities (derived from the same seed).

### Gameplay

The game proceeds in rounds. Each round, participants publish Play Waves (Specter Waves tagged with `play_id` and `play_round`) discussing, accusing, and defending. At the end of each round, participants vote to eliminate one player by publishing Vote Waves. The eliminated player's role is revealed (locally computed by each observer from the seed). The game continues until all Shades are eliminated (Echoes win) or Shades equal or outnumber Echoes (Shades win).

### Resonance Rewards

Winners receive: `play_bonus = 5 * ln(1 + participant_count)`. Participants on the losing side receive a smaller bonus: `consolation_bonus = 2 * ln(1 + participant_count)`. Bonuses decay over 14 days.

### Play Visibility

Active Shadow Plays appear on the Pulse Map as a dark, swirling vortex glyph. Spectators outside the game cannot read Play Waves but can see the vortex pulsing with each round's activity. Eliminations cause a brief flash at the eliminated Specter's node. The game's social deduction dynamic — anonymous identities trying to identify other anonymous identities' hidden roles — is uniquely enabled by MURMUR's anonymity architecture.

---

## Masked Events

### Concept

Masked Events are time-limited, anonymous social gatherings where all participants shed even their Specter identities and interact through single-use Masked keypairs. Within a Masked Event, no participant knows who any other participant is — not their main identity, not their Specter. Everyone is equal, anonymous, and ephemeral.

Masked Events are designed for moments when Specter-level anonymity is insufficient: when the community wants to discuss a sensitive topic, hold an anonymous brainstorm, conduct anonymous peer feedback, or simply experience the social freedom of total unrecognizability. They are MURMUR's most radical anonymity mechanic.

### Event Creation

Any Specter in Hybrid+ mode can create a Masked Event by publishing a Beacon Wave (type 0x08) with `beacon_type` set to `event_announce`. The announcement metadata includes `event_id` (a random 32-byte identifier), `event_topic` (UTF-8 string, max 256 bytes, describing the event's theme or purpose), `event_start` (Unix timestamp), `event_duration` (minutes, one of: 30, 60, 120, or 240), and `event_max_participants` (integer, 0 for unlimited, or a cap between 5 and 100).

The Beacon Wave requires the elevated PoW difficulty (24 leading zero bits) to prevent spam event creation. The announcement propagates through the Anonymous Layer gossip.

### Joining an Event

A Specter joins an upcoming Masked Event by generating a fresh Ed25519 keypair — the Masked keypair — with no relationship to their Specter or main identity keypairs. The Specter's client derives a Masked pseudonym from the Masked public key hash using a dedicated wordlist distinct from the Specter pseudonym wordlist (to prevent confusion between Specter and Masked identities). The Masked pseudonym follows the same two-word pattern but uses event-themed vocabulary (e.g., "Flickering Mask," "Distant Echo," "Burning Question").

The Specter publishes a Join Wave on the Anonymous Layer: a Specter Wave with metadata `event_id` and `masked_pubkey` containing the Masked public key. This Join Wave links the Specter to the event (the Specter is publicly joining) but does not link the Specter to the Masked keypair in a way observable to others. The `masked_pubkey` field is encrypted to the event creator's Specter public key using X25519 Diffie-Hellman key exchange (the Ed25519 keys are converted to X25519 for this purpose). This allows the event creator to verify participation count and enforce the participant cap, but the event creator cannot link Masked keypairs to Specters during the event because the event creator also uses a Masked keypair and does not know which Masked keypair belongs to which Specter.

The encrypted join mechanism ensures that the participant count is accurate (the creator can count encrypted joins) while maintaining unlinkability during the event.

### Event Execution

When the event start time arrives, all participants switch to their Masked keypairs. The event runs on a dedicated gossip topic: `murmur/event/[event_id]/1.0`. Only Masked keypairs that were registered in valid Join Waves can publish to this topic (other nodes reject Waves from unregistered Masked keys).

Within the event, participants publish Masked Waves (type 0x07) using their Masked keypairs. The event functions as a temporary, anonymous chat room: participants post Waves, reply to each other's Waves, and interact as their Masked pseudonyms. The PoW requirement applies to Masked Waves (preventing spam within the event).

Participants cannot determine who else is in the event. They see Masked pseudonyms and Masked sigils, none of which are linkable to Specter or main identities. The event creator has no special powers during the event — they are an anonymous participant like everyone else.

### Event Conclusion

When the event duration expires, the event topic is closed. No new Masked Waves are accepted. Participants' clients delete their Masked private keys (the keys are ephemeral and were never persisted to permanent storage).

A Resonance Burst is computed for each participant based on the amplification their Masked Waves received during the event. The Burst is a temporary Specter Resonance bonus: `burst_value = 5 * ln(1 + amplifications_received_during_event)`. The Burst is applied to the participant's Specter Resonance and decays linearly to zero over 7 days.

The mapping between Masked keypair and Specter identity for Burst application is handled locally by each participant's client. The client knows which Masked keypair it generated and can compute the Burst from the amplification data observed during the event. No external party can link the Masked keypair to the Specter.

A Summary Beacon Wave is generated and published to the Anonymous Layer after the event. The summary includes the event topic, duration, participant count, total Waves published, total amplifications, and a Resonance Burst leaderboard showing Masked pseudonyms (not Specter pseudonyms) ranked by Burst value. The leaderboard is a competitive element that rewards quality contributions during the event without revealing participant identities.

### Post-Event Unlinkability

After the event concludes, the Masked Waves remain available on the event gossip topic for 7 days (a shorter retention window than the standard 30 days, reflecting the ephemeral nature of events). After 7 days, nodes garbage collect Masked Waves.

No participant can prove they authored a specific Masked Wave after the event, because the Masked private key was deleted. No observer can link a Masked pseudonym to a Specter or main identity. The event exists as a self-contained, anonymous social moment that leaves behind only a summary and participants' memories.

---

## Phantom Councils

### Concept

Phantom Councils are persistent, private, anonymous coordination groups. A Council is a small group of high-Resonance Specters (minimum 3, maximum 13) who meet regularly to discuss, coordinate, and vote on matters of shared interest. Councils are the Anonymous Layer's governance and elite-coordination mechanic — they are exclusive, secretive, and powerful in the social dynamics of the network.

Councils are inspired by the idea that some conversations require both anonymity and trust: anonymous participants who have proven their commitment to the network (through high Specter Resonance) convening in a private, persistent group to coordinate on important topics. The combination of anonymity, exclusivity, and persistence creates a unique social dynamic unavailable on either the Surface Layer or the open Anonymous Layer.

### Council Creation

Only Fortress-mode Specters can create a Phantom Council. The creating Specter publishes a Council Formation Wave on the Anonymous Layer: a Specter Wave with metadata `council_create` set to `true`, `council_id` (random 32-byte identifier), `council_name` (UTF-8 string, max 64 bytes), `council_purpose` (UTF-8 string, max 256 bytes), `council_min_resonance` (the minimum Specter Resonance required to join, minimum 200), and `council_max_members` (integer, 3 to 13).

The Council Formation Wave propagates through the Anonymous Layer. It serves as a public announcement that a new Council exists and is recruiting.

### Council Membership

A Specter who wishes to join a Council publishes a Council Application Wave: a Specter Wave with metadata `council_id` and `council_apply` set to `true`. The application must include a valid ZK Claim proving that the applicant's Specter Resonance exceeds the Council's minimum threshold. The ZK Claim is a Pedersen commitment with a Bulletproofs-style range proof as described in the Resonance System specification.

The Council's existing members vote on applications. Each member publishes a Council Vote Wave (Specter Wave with metadata `council_id`, `council_vote_target` containing the applicant's Specter public key, and `council_vote` set to "admit" or "reject"). Admission requires a unanimous vote from all existing members. This high bar ensures that every Council member is trusted by every other member.

Members can also vote to expel existing members using the same voting mechanism (with `council_vote` set to "expel"). Expulsion requires a two-thirds majority of the remaining members.

### Council Communication

Council communication takes place on a private gossip topic: `murmur/council/[council_id]/1.0`. Only validated Council members can publish to or read from this topic. The topic is encrypted: all messages are encrypted using a shared symmetric key derived from a group key agreement protocol (a tree-based Diffie-Hellman construction where each member contributes a key share).

The group key is rotated whenever a member is added or expelled. After rotation, the departing member loses access to future messages (they retain access to messages published before their departure, which they already received and decrypted).

Council Waves are a variant of Specter Waves published to the Council's private topic. They support the full range of Wave mechanics (replies, amplifications) within the Council's private context. Council Waves do not propagate outside the Council topic.

### Council Voting

Councils have a built-in voting mechanic for internal coordination. Any Council member can publish a Council Proposal Wave: a Specter Wave on the Council topic with metadata `council_proposal` set to `true` and `proposal_text` containing the proposition. Members vote on proposals by publishing Council Proposal Vote Waves with metadata `council_proposal_id` and `proposal_vote` (values: "for," "against," or "abstain").

Votes are tallied locally by each member's client. The default decision threshold is simple majority of non-abstaining votes. The proposal outcome is not enforced by the protocol — Councils are coordination bodies, not governance mechanisms with binding power. The vote is a tool for structured consensus-building within the group.

### Council Visibility

Council existence is public — the Council Formation Wave is visible on the Anonymous Layer. Council membership is partially public — Council Application Waves and admission votes are visible, so observers know which Specters are members. Council communication is private — the encrypted Council topic is unreadable by non-members.

This design creates a visible power structure on the Anonymous Layer: observers can see that Councils exist, who is in them, and when they are active (by observing gossip traffic patterns on Council topics), but cannot read the content of Council deliberations.

### Council Persistence

Councils persist as long as they have at least 3 members. If membership drops below 3 (through expulsions or voluntary departures), the Council enters a dormant state: the private topic remains encrypted and accessible to remaining members, but no new proposals or votes can be conducted. If membership recovers to 3+ (through new admissions), the Council reactivates.

Council Waves follow the same 30-day content window as all other Waves. Council deliberation history older than 30 days is garbage collected by member nodes.

---

## Specter Marks

### Concept

Specter Marks are visible, anonymous markers placed by a Specter on any node in the network. A Mark manifests as a small, glowing sigil attached to the target node's Pulse Map representation. Marks are visible to all users (Surface Layer and Anonymous Layer) and carry the marking Specter's pseudonym and sigil.

Marks are deliberately ambiguous. The protocol defines no semantic meaning for a Mark — it is simply a visible statement that a specific anonymous identity has chosen to mark a specific node. The meaning of a Mark is socially constructed by the community: it might signify respect, attention, warning, challenge, or something else entirely. Different communities within MURMUR may develop different Mark cultures.

### Placing a Mark

Placing a Mark requires Specter Resonance 100 or higher (Phantom milestone). The Specter selects a target node on the Pulse Map and confirms the marking action. The Mark is encoded as a signed data structure containing the target's public key (main identity key for Surface Layer targets, Specter key for Anonymous Layer targets), the marker's Specter public key, a timestamp, and the marker's Specter signature.

The Mark is broadcast on the Anonymous Layer gossip topic. Bridge nodes inject Marks targeting Surface Layer nodes into the Surface Layer gossip. Recipient nodes validate the signature and the marker's Specter Resonance (which must meet the 100 minimum at the time of marking, as assessed by the recipient's local Resonance computation).

### Mark Limits

A Specter can maintain a maximum of 5 active Marks at any time. Placing a 6th Mark requires removing one of the existing 5 first. This scarcity ensures that Marks are meaningful — a Specter must choose carefully where to invest their limited marking capacity.

### Mark Duration and Renewal

Marks last 30 days from placement. After 30 days, the Mark fades and disappears. The marking Specter can renew a Mark by re-placing it before expiration, resetting the 30-day timer. Renewal costs one of the Specter's 5 active Mark slots (or rather, the renewed Mark continues to occupy the slot it was already using).

### Mark Display

On the Pulse Map, a Mark appears as a small glowing sigil positioned near the target node. The sigil is the marking Specter's procedurally generated sigil, rendered at a small size (approximately 12×12 pixels at default zoom). Hovering over or tapping the sigil displays a tooltip with the marking Specter's pseudonym, the Mark's age, and a "Marked by [Specter pseudonym]" label.

A node can accumulate multiple Marks from different Specters. Multiple Marks are displayed as a cluster of small sigils around the target node, creating a visual sense of attention or significance. A node with many Marks is visually distinguished — it has been noticed by the anonymous world.

### Mark Removal

The marking Specter can remove their own Mark at any time. The target node cannot remove Marks placed on them. This asymmetry is deliberate: Marks are expressions of the marker's agency, not the target's. A target who is uncomfortable with a Mark can mute the marking Specter, which hides the Mark in the target's local view but does not remove it for other observers.

---

## Whisper Chains

### Concept

Whisper Chains are anonymous, multi-hop message relays between Specters. A Whisper Chain allows a Specter to send a private message to a distant Specter through a chain of intermediate Specters, where each intermediate node sees only the previous and next hop — not the origin, destination, or content of the message.

Whisper Chains are the Anonymous Layer's private messaging mechanic. Unlike Specter Waves (which are public broadcasts), Whisper Chain messages are point-to-point and encrypted. They provide a private communication channel between Specters who may not be directly connected.

### Chain Construction

The sender selects a target Specter. The sender's client computes a path through the Anonymous Layer topology from the sender to the target, selecting 2–4 intermediate Specters as relay hops. Hop selection prefers Specters with high Specter Resonance (similar to Shroud Node chain selection, but applied to the general Specter population).

The sender constructs the message using a layered encryption scheme analogous to the Shroud Network's onion routing. The message is encrypted in layers, one per hop. The innermost layer is encrypted to the target's Specter public key and contains the actual message content (max 1024 bytes, UTF-8 text). Each outer layer is encrypted to the corresponding relay Specter's public key and contains the next hop's Specter address and the encrypted inner payload.

The sender transmits the outermost layer to the first relay hop. The first relay peels its layer, discovers the second relay's address, and forwards the inner payload. This continues until the message reaches the target, who peels the final layer and reads the content.

### Relay Incentive

Serving as a Whisper Chain relay contributes to the Whisper Chain Contributions signal in Specter Resonance computation. Relays do not know the content of the messages they forward (they see only encrypted blobs) and do not know the chain's origin or final destination (they know only the previous and next hop). The Resonance reward incentivizes Specters to accept relay requests, maintaining the Whisper Chain infrastructure.

### Relay Consent

A Specter is not forced to serve as a relay. When a relay request arrives (an encrypted blob with a header indicating it is a Whisper Chain hop), the Specter's client can accept or reject it based on local policy. The default policy is to accept all relay requests. Users can configure their client to reject relay requests (opting out of Whisper Chain infrastructure) at the cost of forfeiting the Whisper Chain Contributions Resonance signal.

### Message Limits

A Specter can send a maximum of 10 Whisper Chain messages per 24-hour period. This limit prevents abuse of the relay network. Each Whisper Chain message requires a PoW stamp (same difficulty as a standard Wave) to further deter spam.

### Replies and Conversations

The target of a Whisper Chain message can reply by constructing a new Whisper Chain back to the sender. The sender's Specter address is not included in the message (to preserve sender anonymity), but the sender can optionally include a `reply_address` field in the message content — their Specter public key — to enable replies. If the reply address is omitted, the message is strictly one-way.

Extended Whisper Chain conversations between two Specters who wish to communicate privately may transition to a direct encrypted channel: after establishing mutual interest through Whisper Chain messages, the two Specters can form a Specter connection and communicate via direct encrypted messages over the Anonymous Layer without using relay hops. This transition is voluntary and requires both parties to reveal their Specter addresses to each other.

### Ephemeral Nature

Whisper Chain messages are not stored in the network. They exist only in transit (as encrypted blobs at relay nodes) and at the destination (in the recipient's local storage). Relay nodes delete forwarded blobs immediately after transmission. There is no global record of Whisper Chain activity. The Resonance signal for chain contributions is tracked locally by each relay node (it increments a counter when it forwards a blob) and is not independently verifiable by observers — it is a self-reported signal, which is the weakest signal type in the Resonance computation and is weighted accordingly.

---

## Mechanic Interactions

The anonymous mechanics interact with each other and with the broader system in several designed ways.

### Gift → Connection Pipeline

Phantom Gifts often serve as social icebreakers on the Anonymous Layer. A Specter sends a gift to another Specter. The recipient views the sender's Specter profile. If both are Hybrid+ mode, the recipient can send a gift back or initiate a Specter connection. This gift-exchange pipeline is the most common pathway to new Specter connections, creating an organic social dynamic where anonymous generosity leads to anonymous friendship.

### Puzzle → Resonance → Council Pipeline

Cipher Puzzles, Specter Hunts, and Sigil Forge events are primary vehicles for building Specter Resonance. Winning puzzles and hunts, and producing acclaimed creative work in Forges, generates significant Resonance bonuses. Specters who build high Resonance through mini-game participation become eligible for Phantom Council membership. This creates a meritocratic pipeline: the Anonymous Layer's most skilled and active participants rise to its most exclusive coordination spaces.

### Event → Burst → Visibility Pipeline

Masked Events generate Resonance Bursts for active participants. A Specter who receives a large Burst temporarily has elevated Resonance, making them more visible on the Pulse Map (larger node), more influential in Oracle Pool outcomes (higher prediction credibility), and more attractive as a Whisper Chain relay (preferred by chain selection). This temporary visibility spike creates a rhythm of emergence and recession: Specters who perform well in events briefly shine brighter before returning to their baseline Resonance.

### Territory → Hunt → Culture Pipeline

Territory Drift creates persistent ownership markers on the Pulse Map. Specter Hunts drive exploration into unfamiliar territories. The combination creates a cycle: Controllers defend their territory influence, Hunters explore contested regions, and the resulting activity reshapes territory boundaries. This dynamic creates a living, evolving anonymous landscape that rewards both sustained presence and bold exploration.

### Mark → Mystery → Culture Pipeline

Specter Marks create persistent visual artifacts on the Pulse Map that other users observe and interpret. A node accumulating Marks becomes a focal point of anonymous attention. Other users speculate about why the node was marked. The marked user may change their behavior (posting different content, engaging with different communities) in response to the Marks. This dynamic creates a feedback loop between anonymous observation and public behavior — the anonymous world visibly influencing the public world through mysterious, ambiguous signals.

### Shadow Play → Trust → Social Pipeline

Shadow Play creates intense social bonds between anonymous participants. Specters who successfully identify or deceive each other develop a shared experience that drives subsequent interactions — new Specter connections, Whisper Chain conversations, and Council invitations. The social deduction mechanic is a crucible for anonymous trust-building.

---

## Surface Sparks

### Concept

Surface Sparks are lightweight, Surface-Layer-exclusive challenge mechanics that give Open-mode users a taste of gamified interaction without requiring Anonymous Layer participation. Sparks are brief, spontaneous challenges between Surface Layer users that generate visible effects on the Pulse Map.

### Spark Types

**Wave Relay.** A user publishes a Spark Wave with a "relay prompt" — a creative constraint (e.g., "Describe your day in exactly 7 words"). Connected nodes that see the Spark can participate by publishing a Wave matching the constraint within 5 minutes. The Spark and all matching responses are visually linked on the Pulse Map by brief golden arcs.

**Echo Races.** A user initiates an Echo Race by publishing a Spark Wave tagged as a race. The challenge: which connected node can amplify the Wave first? The fastest amplifier's node displays a brief crown glyph. Echo Races are trivial — a few seconds of engagement — but they create visible micro-events that enliven the Surface Layer.

### Requirements

Surface Sparks have no Resonance gate — any Surface Layer user can initiate or participate. They are designed as the entry point to MURMUR's gamified mechanics, available from the first session. Sparks contribute to Surface Layer Resonance through the Wave Output and Amplification signals.

---

## Cartographer's Trail

### Concept

Cartographer's Trail rewards Specters for exploring unfamiliar regions of the Pulse Map. The mechanic maintains a personal exploration log — a set of network regions the Specter has visited and interacted with. Each new region discovered contributes to a Cartographer score, which feeds into Specter Resonance.

### Exploration Mechanics

A Specter "discovers" a territory by publishing a Wave or forming a connection within a previously unvisited cluster (as defined by the Louvain community detection algorithm). The discovery is logged locally by the client. The Cartographer score is: `cartographer_score = 6 * ln(1 + distinct_territories_visited_90d)`.

### Discovery Beacons

When a Specter visits a territory for the first time, their client publishes a Discovery Wave (a Specter Wave with metadata `discovery` set to `true` and `territory_hash` containing the cluster's identifier hash). This Wave is visible to other Specters and creates a brief luminous ping at the Specter's location on the Pulse Map, visible to all Anonymous Layer observers. Discovery Waves signal that someone is exploring, encouraging others to follow.

### Cartographer Milestones

At 5 territories discovered, the Specter earns the "Wanderer" badge (a small compass glyph displayed near their node). At 20 territories, "Pathfinder" (an upgraded compass with animated needle). At 50 territories, "Cartographer" (a detailed map glyph). Badges are visible on the Specter's profile and Pulse Map node.

---

## Echo Chains

### Concept

Echo Chains incentivize meaningful amplification by creating visible, rewarded chains of re-broadcast. When a Wave is amplified through a chain of at least 3 distinct amplifiers, an Echo Chain forms — a visible, luminous thread connecting all amplifiers on the Pulse Map.

### Chain Formation

An Echo Chain forms automatically when a Wave accumulates 3+ sequential amplifications: A publishes, B amplifies, C amplifies from B, D amplifies from C. The chain A→B→C→D is rendered on the Pulse Map as a thin, glowing arc connecting the chain's nodes, persisting for 1 hour.

### Chain Rewards

Each node in an Echo Chain of length N receives a small Resonance bonus: `echo_chain_bonus = 1 * ln(N)`. This rewards Specters who participate in meaningful amplification cascades — content that multiple distinct identities independently chose to propagate. The bonus is modest (preventing gaming) but visible, creating a light incentive to amplify quality content that is already being amplified.

### Chain Visibility

Echo Chains are rendered on the Pulse Map as golden (Surface Layer) or silver (Anonymous Layer) arcs connecting chain participants. Long chains (5+ amplifiers) display a subtle shimmer effect along the arc. The visual makes amplification behavior tangible and social — users can see content flowing through the network in chains of endorsement.

---

## Pulse Beats

### Concept

Pulse Beats are gamified notification events that transform routine network notifications into brief, engaging micro-interactions. Instead of a static notification badge, significant events trigger a Pulse Beat — a small, interactive animation at the edge of the Pulse Map viewport.

### Beat Types

**Gift Beat.** When a Phantom Gift arrives, a small gift-glyph pulses at the viewport edge, trailing particles in the gift's color. Tapping the Beat pans to the gifted node.

**Hunt Beat.** When a Specter Hunt begins in a nearby territory, a compass glyph spins at the viewport edge with a countdown timer.

**Forge Beat.** When a Sigil Forge is active, an anvil glyph glows at the viewport edge, showing the time remaining.

**Chain Beat.** When an Echo Chain passes through the user's node, a chain-link glyph flashes briefly.

**Territory Beat.** When the user's controlled territory is contested, a shield glyph pulses in a warning color at the viewport edge.

### Beat Collection

Pulse Beats are logged in a personal Beat Journal accessible from the Pulse Map's status panel. The journal records all Beats received, creating a personal history of notable network events. Users can review their Beat Journal to see patterns in their engagement — which territories they interact with, how often they receive gifts, how many chains they participate in.

---

## Specter Trophies

### Concept

Specter Trophies are achievements, milestones, and collectibles tied to identity progression on the Anonymous Layer. Trophies are unlocked by completing specific actions or reaching specific thresholds. They are displayed on the Specter's profile and contribute a small, fixed Resonance bonus.

### Trophy Categories

**Milestone Trophies.** Unlocked by reaching Specter Resonance milestones: "First Shade" (Resonance 25), "Wraith Rising" (50), "Phantom Ascendant" (100), "Revenant" (200), "Abyss Walker" (500). Each milestone trophy adds a small glyph to the Specter's profile.

**Activity Trophies.** Unlocked by cumulative actions: "First Gift Sent," "10 Puzzles Solved," "5 Hunts Completed," "3 Forges Won," "First Shadow Play," "First Territory Controlled," "100 Waves Published." Each trophy adds 1 Resonance (flat, non-decaying).

**Rare Trophies.** Unlocked by unusual or difficult achievements: "Cartographer" (50 territories discovered), "Oracle" (10 correct Oracle Pool predictions in a row), "Chain Breaker" (participate in an Echo Chain of length 10+), "Ghost" (maintain Resonance 100+ for 90 consecutive days), "Council Founder" (initiate a Phantom Council). Rare trophies add 3 Resonance each and display an animated glyph.

### Trophy Display

Trophies are visible on the Specter's profile in the Node Detail Panel. A summary count ("12 Trophies") is displayed below the Specter's Resonance score. At micro zoom, trophy glyphs orbit the Specter's node as tiny animated icons, creating a visual halo of achievement visible to other Anonymous Layer participants.

### Implementation Notes

Trophy state is tracked locally by each Specter's client. Trophy unlocks are published as Trophy Waves (Specter Waves with metadata `trophy_id`) for network visibility, but trophy verification relies on the client's local activity log. Trophies are self-reported — like Whisper Chain contributions, they are a weak signal in Resonance computation, weighted accordingly.

### Whisper → Duel Challenge Pipeline

Whisper Chains enable private negotiation before public duels. A Specter who wants to challenge another Specter to a duel can first send a Whisper Chain message to discuss terms, topics, and timing. This private negotiation channel makes duels more deliberate and less impulsive, improving the quality of public debate.

---

## Abuse Considerations

Each anonymous mechanic has been designed with abuse potential in mind.

Phantom Gifts are limited to 3 per day and are purely cosmetic. The worst-case abuse scenario is unwanted cosmetic effects on a target's node, which the target can mitigate by muting the gifting Specter (hiding the gift in their local view).

Specter Duels are voluntary — both participants must consent to the duel. Audience voting is weighted by Resonance, making vote manipulation expensive (an attacker needs many high-Resonance Specters to sway a vote). Duel arguments are public and subject to the same PoW and content window constraints as all Waves.

Masked Events are ephemeral and require PoW for participation. The participant cap (max 100) prevents mass-scale abuse within a single event. Masked keypairs are destroyed after the event, preventing persistent harassment from Masked identities.

Phantom Councils are gated by Fortress mode and high Resonance, making them inaccessible to casual abusers. Admission requires unanimous approval from existing members, giving every member veto power over new entrants. Expulsion requires a two-thirds majority.

Specter Marks are limited to 5 active Marks per Specter and gated by Resonance 100. The target cannot remove Marks, which is a potential harassment vector — but the Mark is purely visual, carries no functional consequence, and can be hidden by muting the marker. The 30-day expiration ensures that Marks are not permanent.

Whisper Chains are limited to 10 messages per day and require PoW. Relays do not see content. The reply-address mechanism is opt-in, so senders cannot be forced to reveal their identity. Unwanted Whisper Chain messages can be blocked by refusing to decrypt messages from unknown senders (a client-side filter).

The network does not attempt to prevent all abuse. It provides computational, economic, and social barriers that make abuse costly and low-reward while preserving the freedom and expressiveness that make anonymous mechanics valuable.