# Anonymous Mechanics

**Category:** Core Mechanics — Anonymous Layer Social Features
**Version:** 0.4
**Status:** Draft

---

## Overview

Anonymous Mechanics are the social features exclusive to or primarily operating on the Anonymous Layer. They are the reason the Anonymous Layer exists as more than a privacy tool — they create a parallel social world with its own culture, rituals, competition, and generosity. Each mechanic is designed to be impossible or meaningless without anonymity, making the Anonymous Layer a place with experiences that the Surface Layer cannot replicate.

All Anonymous Mechanics use Specter identity unless otherwise noted. Participation requires Hybrid mode or higher. Several mechanics are gated by Specter Resonance milestones, ensuring that the most powerful anonymous social tools are earned through sustained anonymous participation.

The mechanics described in this document are: Phantom Gifts, Specter Duels, Masked Events, Phantom Councils, Specter Marks, and Whisper Chains.

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

## Specter Duels

### Concept

Specter Duels are structured, public, anonymous debates between two Specters. A duel is a time-limited exchange of arguments on a declared topic, judged by an anonymous audience vote. Duels are competitive, performative, and social — they are the Anonymous Layer's spectator sport.

Duels exist because anonymity enables a specific kind of intellectual honesty: when your identity is detached from your argument, you are free to argue positions you might not publicly endorse, to steelman opposing views, and to engage with ideas purely on their merits. Duels formalize this dynamic into a structured, entertaining mechanic.

### Initiating a Duel

A Specter initiates a duel by publishing a Duel Challenge Wave — a special Specter Wave (type 0x04) with metadata key `duel_challenge` set to `true` and additional metadata fields: `duel_topic` (UTF-8 string, the topic or proposition to be debated, max 256 bytes), `duel_stance` (the challenger's declared stance: "for" or "against"), and `duel_duration` (the duration of the duel in minutes, one of: 30, 60, or 120).

The Duel Challenge Wave propagates through the Anonymous Layer like any Specter Wave. Any Specter who sees the challenge can accept it by publishing a Duel Accept Wave: a Specter Wave with metadata key `duel_accept` set to `true`, `duel_challenge_id` containing the Wave ID of the challenge, and `duel_stance` set to the opposing stance.

If no Specter accepts within 24 hours, the challenge expires. If multiple Specters attempt to accept, the first valid acceptance (by timestamp) is canonical. The challenger's client validates the acceptance and publishes a Duel Start Wave with metadata confirming both participants and the start time.

### Duel Structure

Once a duel starts, the two participants exchange argument Waves within the duel's time window. Duel argument Waves are Specter Waves with metadata key `duel_id` containing the Duel Start Wave's ID and `duel_round` containing the round number (integer, starting from 1).

The duel proceeds in alternating rounds. The challenger publishes round 1. The acceptor publishes round 2. The challenger publishes round 3. And so on. Each participant has a maximum of 5 minutes per round to publish their argument. If a participant fails to publish within the time window, they forfeit the round (the round is displayed as "[Forfeit]" in the duel thread). Three consecutive forfeits end the duel with the other participant declared the winner by default.

The maximum number of rounds is determined by the duel duration: 30-minute duels allow up to 6 rounds (3 per participant), 60-minute duels allow up to 10 rounds (5 per participant), and 120-minute duels allow up to 16 rounds (8 per participant).

Each argument Wave has the same 2048-byte content limit as any Wave and requires the same PoW stamp. The arguments are public — any Specter following the duel can read the exchange in real time.

### Audience and Voting

Any Specter can follow an active duel by subscribing to its duel thread (Waves with the matching `duel_id` metadata). Following is passive — audience members read the arguments but do not publish Waves in the duel thread.

When the duel ends (all rounds complete, or a participant forfeits), a 15-minute voting window opens. The duel's final argument Wave includes metadata key `duel_voting_open` set to `true` and `duel_voting_deadline` containing the Unix timestamp of the voting deadline.

Audience members vote by publishing a Duel Vote Wave: a Specter Wave with metadata `duel_id`, `duel_vote` (the public key of the participant they judge to have won), and a standard PoW stamp. Each Specter can vote once per duel (duplicate votes from the same Specter public key are rejected by recipients).

### Vote Tallying and Resonance Consequences

After the voting deadline, each node locally tallies the votes. Votes are weighted by the voter's Specter Resonance as computed by the tallying node. A vote from a Specter with Resonance 100 carries ten times the weight of a vote from a Specter with Resonance 10 (linear weighting).

The weighted vote counts determine the winner. The tallying node generates a local Duel Result containing the winner, the weighted vote margin, and the total number of voters. Because Resonance is locally computed and may vary slightly between observers, different nodes may compute slightly different weighted tallies — but in practice, the winner is consistent across observers except in extremely close duels.

The winner receives a Specter Resonance bonus based on the vote margin. The bonus formula is `duel_bonus = 3 * ln(1 + weighted_margin / total_weighted_votes * voter_count)`. This rewards decisive victories with larger audiences more than narrow victories with small audiences. The bonus is temporary: it decays linearly to zero over 14 days.

The loser receives no bonus and loses no Resonance. Participation in a duel (win or lose) contributes to the Duel Record signal in the Specter Resonance computation (net wins in the trailing 30 days).

### Duel Visibility

Active duels are highlighted on the Anonymous Layer Pulse Map. The two dueling Specter nodes are connected by an animated "clash" visual — a jagged, sparking line between them. Audience members' nodes show a faint directional indicator pointing toward the duel, creating a visual sense of attention flowing toward the event.

Completed duels are archived as a thread of Waves that remains available for the standard 30-day content window. The Duel Result (including winner, margin, and voter count) is included as metadata in a final Beacon Wave published to the duel thread.

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

Phantom Councils are persistent, private, anonymous deliberation groups. A Council is a small group of high-Resonance Specters (minimum 3, maximum 13) who meet regularly to discuss, debate, and vote on matters of shared interest. Councils are the Anonymous Layer's governance and elite-discussion mechanic — they are exclusive, secretive, and powerful in the social dynamics of the network.

Councils are inspired by the idea that some conversations require both anonymity and trust: anonymous participants who have proven their commitment to the network (through high Specter Resonance) convening in a private, persistent group to deliberate on important topics. The combination of anonymity, exclusivity, and persistence creates a unique social dynamic unavailable on either the Surface Layer or the open Anonymous Layer.

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

Councils have a built-in voting mechanic for internal deliberation. Any Council member can publish a Council Proposal Wave: a Specter Wave on the Council topic with metadata `council_proposal` set to `true` and `proposal_text` containing the proposition. Members vote on proposals by publishing Council Proposal Vote Waves with metadata `council_proposal_id` and `proposal_vote` (values: "for," "against," or "abstain").

Votes are tallied locally by each member's client. The default decision threshold is simple majority of non-abstaining votes. The proposal outcome is not enforced by the protocol — Councils are deliberative bodies, not governance mechanisms with binding power. The vote is a tool for structured decision-making within the group.

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

### Duel → Resonance → Council Pipeline

Specter Duels are a primary vehicle for building Specter Resonance. Winning duels with large audiences generates significant Resonance bonuses. Specters who build high Resonance through duel victories become eligible for Phantom Council membership. This creates a meritocratic pipeline: the Anonymous Layer's most skilled debaters and thinkers rise to its most exclusive deliberative spaces.

### Event → Burst → Visibility Pipeline

Masked Events generate Resonance Bursts for active participants. A Specter who receives a large Burst temporarily has elevated Resonance, making them more visible on the Pulse Map (larger node), more influential in duel votes (higher vote weight), and more attractive as a Whisper Chain relay (preferred by chain selection). This temporary visibility spike creates a rhythm of emergence and recession: Specters who perform well in events briefly shine brighter before returning to their baseline Resonance.

### Mark → Mystery → Culture Pipeline

Specter Marks create persistent visual artifacts on the Pulse Map that other users observe and interpret. A node accumulating Marks becomes a focal point of anonymous attention. Other users speculate about why the node was marked. The marked user may change their behavior (posting different content, engaging with different communities) in response to the Marks. This dynamic creates a feedback loop between anonymous observation and public behavior — the anonymous world visibly influencing the public world through mysterious, ambiguous signals.

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