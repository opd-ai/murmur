# Waves

**Category:** Core Mechanics — Content
**Version:** 0.4
**Status:** Draft

---

## Overview

Waves are MURMUR's content primitive. Every piece of user-generated content in the network — text posts, replies, announcements, anonymous broadcasts, cross-layer messages — is a Wave. Waves propagate through the network via amplification and gossip, spreading outward from the author's node like ripples in water. The metaphor is literal in the Pulse Map visualization, where publishing a Wave triggers an animated ripple effect emanating from the author's node.

Waves are not stored on a central server. Each Wave is a signed, self-contained data structure that propagates peer-to-peer through the network. Nodes store Waves they have received and forward them to connected peers according to propagation rules. There is no global feed, no algorithmic timeline, and no central index. A user's feed is composed of Waves that have reached their node through the network topology — Waves from their direct connections, Waves amplified by their connections, and Waves that propagated through gossip.

Every Wave requires a Proof-of-Work stamp before it can be published. This computational cost is the primary spam deterrent in the absence of centralized moderation.

---

## Wave Structure

A Wave is a binary-serialized data structure with the following fields.

**Wave ID.** A 32-byte unique identifier derived from the SHA-256 hash of the Wave's content, author public key, and timestamp. The Wave ID is the canonical reference for the Wave across the network.

**Author Public Key.** The Ed25519 public key of the Wave's author. For Surface Layer Waves, this is the author's main identity public key. For Anonymous Layer Waves, this is the author's Specter public key. For Masked Event Waves, this is the single-use Masked keypair's public key.

**Timestamp.** Unix timestamp in milliseconds at time of composition. Nodes reject Waves with timestamps more than 5 minutes in the future or more than 30 days in the past relative to the receiving node's local clock. The 30-day past limit enforces the network's rolling content window — Waves older than 30 days are eligible for garbage collection and are not guaranteed to be stored or forwarded by any node.

**Wave Type.** An enumerated byte indicating the Wave's type. Valid types are: Surface Wave (0x01), Reply Wave (0x02), Veiled Wave (0x03), Specter Wave (0x04), Sigil Wave (0x05), Abyssal Wave (0x06), Masked Wave (0x07), and Beacon Wave (0x08).

**Content.** The Wave's payload, variable length, maximum 2048 bytes. Content is UTF-8 encoded text. No binary attachments, images, or media embeds are supported in the initial protocol version. Content may include references to other Waves by Wave ID (formatted as `wave://[hex-encoded Wave ID]`), which clients render as inline links or quote-style embeds.

**Parent ID.** For Reply Waves only: the Wave ID of the Wave being replied to. Null (32 zero bytes) for non-reply Waves.

**PoW Nonce.** An 8-byte nonce that, when appended to the Wave's serialized content and hashed with SHA-256, produces a hash with the required number of leading zero bits. The difficulty target is configurable per network and defaults to 20 leading zero bits (approximately 1 million hash operations, taking 0.5–2 seconds on modern mobile hardware).

**PoW Hash.** The resulting SHA-256 hash demonstrating the Proof-of-Work. Included for efficient verification — recipients verify the hash rather than recomputing it from scratch, though they do verify that the hash matches the nonce and content.

**Signature.** An Ed25519 signature over the concatenation of all preceding fields (Wave ID, Author Public Key, Timestamp, Wave Type, Content, Parent ID, PoW Nonce, PoW Hash), signed with the author's private key corresponding to the Author Public Key.

**Hop Count.** A single byte indicating the number of hops the Wave has traveled from the author. Set to 0 by the author, incremented by each forwarding node. Not covered by the signature (it changes at each hop). Used for propagation limiting — nodes discard Waves with Hop Count exceeding the maximum (default: 20 hops).

**Metadata.** A variable-length key-value map for type-specific metadata. Encoded as a length-prefixed sequence of key-value pairs where keys and values are length-prefixed UTF-8 strings. Maximum total metadata size: 512 bytes. Metadata is covered by the signature.

---

## Wave Types

### Surface Wave

Type byte: 0x01. The standard Surface Layer content Wave. Authored with the user's main identity keypair. Visible to all users on the Surface Layer. Propagates through Surface Layer connections and amplification.

Surface Waves are the most common Wave type. They appear in the user's connections' feeds, on the Pulse Map as ripple animations, and in the author's profile Wave history. The author's current Resonance score at time of composition is included in the metadata under the key `resonance` for display purposes.

### Reply Wave

Type byte: 0x02. A Surface Layer Wave that references a parent Wave. The Parent ID field contains the Wave ID of the Wave being replied to. Reply Waves propagate independently from their parent — a node may receive a Reply Wave without having received the parent, in which case the client displays the reply with a "parent not found" indicator and may attempt to fetch the parent from peers.

Reply Waves form threaded conversations. The threading model is flat — replies reference a single parent, and replies to replies reference the reply as their parent, forming a tree structure. Clients render reply trees with indentation or collapsible threading depending on UI context.

Reply Waves carry the same PoW requirement as Surface Waves. This makes rapid-fire reply spam computationally expensive.

### Veiled Wave

Type byte: 0x03. A cross-layer Wave authored by a Specter but visible on both layers. The Author Public Key is the Specter's public key. The Wave propagates through both the Anonymous Layer (among Specters) and the Surface Layer (among all users).

On the Surface Layer, a Veiled Wave is displayed with the Specter's pseudonym and sigil as the author identity. Surface Layer users can read the content and amplify the Wave, but cannot reply to it (replies to Veiled Waves are only possible from the Anonymous Layer, by other Specters), cannot identify the author's main identity, and cannot interact with the author's Specter directly unless they are in Hybrid+ mode.

Veiled Waves are the primary mechanism for anonymous voices to reach the broader network. A Specter who publishes a Veiled Wave is choosing to address both the anonymous and public audiences simultaneously. The Wave's metadata includes the key `veil` with value `true` to signal its cross-layer nature.

Propagation mechanics for Veiled Waves are handled by bridge nodes. When a bridge node (Hybrid+) receives a Veiled Wave on the Anonymous Layer, it also injects the Wave into the Surface Layer gossip. This dual injection is the only point where the layers interact for Veiled Waves — the Wave itself is cryptographically identical on both layers.

### Specter Wave

Type byte: 0x04. An Anonymous Layer Wave authored by a Specter, visible only on the Anonymous Layer. The Author Public Key is the Specter's public key. The Wave propagates only through Anonymous Layer connections and is never injected into the Surface Layer.

Open-mode users cannot see Specter Waves. Hybrid+ users see Specter Waves in their Anonymous Layer feed when interacting through their Specter identity.

Specter Waves are the standard content type for anonymous conversations. They function identically to Surface Waves within the Anonymous Layer — they support replies (Reply Waves with a Specter author key and a Specter Wave as parent), amplification, and Resonance accrual.

### Sigil Wave

Type byte: 0x05. A Surface Layer Wave authored with the user's main identity keypair that contains an embedded Specter sigil in its metadata. The metadata key `sigil_hash` contains the SHA-256 hash of a randomly selected Specter's public key from the network. The client renders the corresponding sigil as a visual emblem attached to the Wave.

The embedded sigil is not the author's own Specter sigil. It is randomly selected from the set of Specters visible in the author's local topology. This creates plausible deniability — the presence of a sigil signals that the author participates in the Anonymous Layer, but the specific sigil does not identify the author's Specter.

Sigil Waves are a social signaling mechanic. By publishing a Sigil Wave, a Surface Layer user says "I have an anonymous identity" without revealing which one. The specific sigil chosen is cosmetic and carries no semantic meaning. Clients display a tooltip on the sigil explaining the mechanic to prevent misinterpretation.

### Abyssal Wave

Type byte: 0x06. The most deeply anonymous Wave type, available only to Fortress-mode Specters. Abyssal Waves are authored with a one-time keypair derived from the Specter's keypair using a deterministic but unlinkable derivation: `abyssal_key = Ed25519_keygen(SHA-256(specter_private_key || abyssal_nonce))` where `abyssal_nonce` is a random 32-byte value included in the Wave's metadata under the key `abyssal_nonce`.

The Abyssal Wave's Author Public Key is the one-time abyssal key. This key cannot be linked to the Specter's public key by any observer (the derivation is one-way due to the hash function). The author cannot prove they authored a specific Abyssal Wave without revealing the `abyssal_nonce` and their Specter private key.

Abyssal Waves propagate only on the Anonymous Layer. They cannot be replied to (there is no persistent identity to reply to — each Abyssal Wave has a unique, disposable author key). They can be amplified by other Specters.

Abyssal Waves are designed for situations where even Specter-level anonymity is insufficient — where the content is so sensitive that the author wants no persistent anonymous identity attached to it. Each Abyssal Wave is an island: authored by a key that exists for that Wave alone and is never reused.

The metadata includes `abyssal_nonce` (for the author's local records — this is not used by recipients) and `fortress_zk` — a ZK proof that the one-time key was derived from a valid Fortress-mode Specter with Resonance above a minimum threshold (default: 50). This proof prevents abuse by ensuring that only established Fortress-mode participants can author Abyssal Waves while revealing nothing about which Fortress-mode Specter authored the Wave.

### Masked Wave

Type byte: 0x07. A Wave authored during a Masked Event using a single-use Masked keypair. Masked Waves are scoped to the event — they propagate only among event participants and are tagged with the event's unique identifier in the metadata under the key `event_id`.

Masked Waves are authored with the Masked keypair generated specifically for the event. The keypair is unlinked to both the user's main identity and their Specter. Within the event, each participant is identified only by their Masked pseudonym (derived from the Masked public key hash, following the same two-word generation algorithm as Specter pseudonyms but using a different wordlist to prevent confusion).

After the event concludes, Masked Waves are included in the event's Summary Wave (see Beacon Wave below). Masked Waves older than 7 days after the event's conclusion are eligible for garbage collection.

### Beacon Wave

Type byte: 0x08. A system-generated Wave that serves as an announcement or summary. Beacon Waves are not authored by individual users — they are generated by event protocols and carry a null Author Public Key (32 zero bytes) with no signature. Beacon Waves are verified by their content structure and PoW stamp rather than by author signature.

Beacon Waves are used for Masked Event announcements (metadata key `beacon_type` value `event_announce`, containing event parameters), Masked Event summaries (metadata key `beacon_type` value `event_summary`, containing the Resonance Burst leaderboard and aggregate statistics), and network health announcements (metadata key `beacon_type` value `network_health`, containing Shroud Network capacity metrics).

Beacon Waves have a higher PoW difficulty than standard Waves (default: 24 leading zero bits) to prevent spam, since they lack author signatures as a trust signal.

---

## Proof-of-Work

### Purpose

Every Wave requires a valid Proof-of-Work stamp before it can be accepted by any node in the network. The PoW requirement serves as MURMUR's primary spam deterrent in the absence of centralized moderation or identity verification. An attacker who wants to flood the network with spam Waves must expend significant computational resources to do so.

### Algorithm

The PoW algorithm is a standard partial hash collision on SHA-256. The author serializes all Wave fields except PoW Nonce, PoW Hash, Hop Count, and Signature into a byte sequence called the PoW preimage. The author then iterates over nonce values (8-byte integers starting from 0), computing `SHA-256(pow_preimage || nonce)` for each, until the resulting hash has the required number of leading zero bits.

The difficulty target is expressed as the minimum number of leading zero bits in the hash. The default difficulty is 20 bits, meaning the hash must start with at least 20 zero bits. This requires approximately `2^20` (about 1 million) hash operations on average. On modern mobile hardware (2024-era smartphone), this takes approximately 0.5 to 2 seconds. On desktop hardware, it takes approximately 0.1 to 0.5 seconds.

### Difficulty Adjustment

The PoW difficulty is not globally fixed. Each node maintains a local difficulty parameter that it applies when validating incoming Waves. The default difficulty of 20 bits is used by all nodes at network launch. If a node's operator observes spam (high Wave volume from unfamiliar sources), they can increase their local difficulty threshold — rejecting Waves with insufficient PoW. This is a local decision; there is no network-wide difficulty adjustment protocol.

In practice, the community is expected to converge on a shared difficulty through social consensus. If the default difficulty proves insufficient against spam, node operators will share recommended difficulty settings through the normal social channels (Waves about network health, community discussions). The app's settings screen includes a "PoW Difficulty" slider with explanatory text.

### Verification

When a node receives a Wave, it verifies the PoW by recomputing `SHA-256(pow_preimage || nonce)` and checking that the result matches the included PoW Hash and meets the local difficulty threshold. This verification is fast (a single SHA-256 computation) and is performed before signature verification, allowing nodes to reject low-quality spam before expending the more expensive Ed25519 verification.

### Reply Chains

Reply Waves carry the same PoW difficulty as top-level Waves. This is a deliberate design decision: it prevents "reply bombing" (flooding a Wave's reply tree with low-cost replies) at the cost of making rapid conversation more effortful. The expected user experience is that composing a reply takes a few seconds of computation after the user finishes typing, during which the client displays a "Minting..." animation. This brief pause is accepted as a design tradeoff — it slows conversation velocity compared to centralized platforms but makes harassment campaigns computationally expensive.

---

## Propagation

### Gossip Protocol

Waves propagate through the network via a gossip protocol built on libp2p's GossipSub implementation. Each node subscribes to one or more gossip topics depending on its mode.

Surface Layer nodes subscribe to the `murmur/surface/waves/1.0` topic. Anonymous Layer nodes (Specters) subscribe to the `murmur/anonymous/waves/1.0` topic. Bridge nodes (Hybrid+) subscribe to both topics and handle cross-layer injection for Veiled Waves.

When a node publishes a Wave, it broadcasts the Wave to all peers on the appropriate topic. Each receiving peer validates the Wave (PoW check, signature check, timestamp check, hop count check) and, if valid, stores the Wave locally and rebroadcasts it to its own peers. The hop count is incremented at each rebroadcast. Waves that exceed the maximum hop count (default: 20) are stored locally but not rebroadcast.

Duplicate suppression is handled by Wave ID: each node maintains a set of recently seen Wave IDs (a Bloom filter with a 30-day window) and silently drops any Wave whose ID it has already seen.

### Amplification

Amplification is MURMUR's equivalent of a retweet or boost. When a user amplifies a Wave, their node rebroadcasts the original Wave with the hop count reset to 0 and a new metadata entry: `amplified_by` containing the amplifier's public key and signature. The amplification signature proves that the amplifier intentionally chose to rebroadcast the Wave (as opposed to routine gossip forwarding).

Amplification does not require PoW from the amplifier — the original Wave's PoW stamp is sufficient. This is because amplification is a curation act (selecting existing content for rebroadcast) rather than a content creation act, and requiring PoW for amplification would discourage the curation behavior that the Resonance system rewards.

When a node receives an amplified Wave, it checks both the original author's signature and the amplifier's signature. If both are valid and the node has not seen this specific amplification before (tracked by the tuple of Wave ID and amplifier public key), it stores the amplification event and rebroadcasts the Wave.

A Wave can be amplified multiple times by different users, creating a cascade of rebroadcasts that pushes the Wave deeper into the network. Each amplification resets the hop count, giving the Wave a fresh propagation radius from the amplifier's position in the topology. This is how high-quality content reaches users who are far from the original author in the network graph.

### Feed Construction

A node's local feed is the set of Waves it has received, filtered and ordered by the client. The client applies the following default ordering:

Waves from direct connections are shown first, ordered by timestamp (newest first). Waves from connections-of-connections (received via single-hop gossip or amplification by a direct connection) are shown next. Waves from more distant sources are shown last. Within each tier, Waves with more amplifications are ranked slightly higher.

This ordering is entirely local — each client computes it from its own stored Wave set. There is no server-side ranking algorithm. Users can customize the ordering: chronological only, amplification-weighted, connection-distance-weighted, or unfiltered (all Waves in receipt order).

The feed is not infinite-scrolling. The client displays Waves from the trailing 24 hours by default, with a "Load more" action to extend the window. Waves older than 30 days are not displayed (they may have been garbage collected).

### Anonymous Layer Propagation

Anonymous Layer Waves (Specter Waves, Abyssal Waves, anonymous Reply Waves) propagate through the Anonymous Layer gossip topic. The propagation mechanics are identical to Surface Layer propagation (gossip, hop count, duplicate suppression) but operate on the Anonymous Layer topology.

For Guarded and Fortress-mode users, anonymous Wave propagation traffic is routed through the Shroud Network. The user's client wraps each gossip message in onion encryption and sends it through a Shroud chain. The exit Shroud Node injects the Wave into the Anonymous Layer gossip topic. Return traffic (Waves received from the Anonymous Layer) travels through a different Shroud chain back to the user.

Bridge nodes handle the injection of Veiled Waves into the Surface Layer gossip topic. When a bridge node receives a Veiled Wave on the Anonymous Layer, it rebroadcasts the Wave on the Surface Layer gossip topic without modification. The bridge node does not sign the Wave or add any identifying information — it simply forwards it. Multiple bridge nodes may independently inject the same Veiled Wave into the Surface Layer; duplicate suppression handles the redundancy.

---

## Content Window and Storage

### 30-Day Window

MURMUR enforces a rolling 30-day content window. Waves older than 30 days are considered expired. Expired Waves are not guaranteed to be stored by any node, are not forwarded during gossip, and are not displayed in feeds or profiles.

The 30-day window is a core design decision. It reflects the belief that ephemeral content creates healthier social dynamics than permanent content: users are less afraid to post when they know the content will disappear, conversations feel more like spoken dialogue than permanent publications, and the network's storage requirements remain bounded.

Nodes are free to garbage collect expired Waves at any time. The default garbage collection policy deletes expired Waves every 24 hours. Nodes with limited storage may collect more aggressively; nodes with abundant storage may retain expired Waves longer, but they will not serve them to peers who request them (expired Waves are excluded from gossip and sync responses).

### Local Storage

Each node stores received Waves in a local database (BadgerDB, a Go-native embedded key-value store). The database is keyed by Wave ID and includes indexes for author public key, timestamp, wave type, and parent ID. The expected storage footprint for a moderately active node (receiving approximately 1,000 Waves per day) is approximately 2–5 MB per day, or 60–150 MB for the full 30-day window.

On mobile devices, the database is stored in the app's sandboxed data directory and is encrypted at rest using the device's file-level encryption. On desktop, the database is stored in the user's home directory under `~/.murmur/waves/` and is encrypted with a key derived from the user's local passphrase.

### Wave Requests

If a node receives a Reply Wave referencing a parent it does not have, or if a user navigates to a Wave ID that is not locally stored, the node can request the missing Wave from its peers. The request is a libp2p direct message to connected peers containing the missing Wave ID. Peers that have the Wave respond with the full Wave data. The requesting node validates the response (PoW, signature, timestamp) before accepting it.

Wave requests are best-effort. There is no guarantee that any peer has the requested Wave, especially if the Wave is near the end of its 30-day window. The client displays "Wave not found" after a configurable timeout (default: 10 seconds).

---

## Wave Interactions

### Reactions

MURMUR does not support emoji reactions or "likes" on Waves. The only interaction primitives are replies (which create new Waves) and amplifications (which rebroadcast existing Waves). This restriction is deliberate: it prevents low-effort engagement signals that create dopamine-loop dynamics and ensures that all engagement with a Wave is either substantive (a reply) or curatorial (an amplification).

### Mentions

Waves can reference other users by including their public key hash (first 8 bytes, hex-encoded) in the content text, prefixed with `@`. The client resolves the hash to a profile name (for Surface Layer mentions) or a Specter pseudonym (for Anonymous Layer mentions) and renders it as a clickable link.

Cross-layer mentions are not supported. A Surface Wave cannot mention a Specter, and a Specter Wave cannot mention a main identity. This restriction preserves identity isolation — allowing cross-layer mentions would create a channel for correlating main identities with Specters.

### Wave References

Waves can reference other Waves by including the Wave ID in the content text, formatted as `wave://[hex-encoded Wave ID]`. The client renders this as an inline embed (a bordered preview of the referenced Wave's content and author) if the referenced Wave is locally available, or as a clickable link if not. Wave references work across layers — a Surface Wave can reference a Veiled Wave (since Veiled Waves are visible on the Surface Layer), and a Specter Wave can reference another Specter Wave.

### Threads

Reply Waves form thread trees rooted at a top-level Wave. The client renders threads with visual nesting (indentation or vertical connector lines). Thread depth is unlimited in the protocol but the client caps visual nesting at 10 levels (deeper replies are displayed at the 10th level with a "deep reply" indicator).

Thread navigation is available from any Wave in the thread: the client can display the full thread tree by recursively resolving Parent IDs. Missing nodes in the tree (Waves not locally available) are displayed as gaps with "Wave not found" placeholders.

---

## Pulse Map Visualization

### Ripple Effect

When a Wave is published, the Pulse Map renders an animated ripple effect emanating from the author's node. The ripple expands outward through the connection graph, following the actual gossip propagation path. The ripple's color and intensity depend on the Wave type: Surface Waves produce warm-toned ripples (amber to white), Veiled Waves produce cool-toned ripples (blue to silver), Specter Waves produce ethereal ripples (pale violet), and Abyssal Waves produce dark ripples (deep purple to black).

The ripple animation is cosmetic and runs locally in the client based on the client's local view of the topology. It does not represent exact propagation timing (actual gossip propagation is asynchronous and variable-latency). The animation is a qualitative visualization of how content spreads through the network.

### Wave Density

Areas of the Pulse Map with high Wave activity (many Waves published recently) show increased visual activity: more frequent ripples, brighter node glows, and subtle particle effects. Areas with low activity appear calmer. This ambient visualization gives users an intuitive sense of where conversation is happening in the network without requiring them to read individual Waves.

### Amplification Trails

When a Wave is amplified, the Pulse Map renders a visual trail from the amplifier's node back to the original author's node, following the connection path between them. The trail is a brief, luminous arc that fades over 2–3 seconds. This visualization makes amplification behavior visible — users can see content flowing through the network in real time.

---

## Moderation

### Absence of Central Moderation

MURMUR has no central moderation authority. There is no content policy, no report button, no trust and safety team. This is a fundamental design commitment: in a decentralized, peer-to-peer network with anonymous participation, centralized moderation is impossible to implement meaningfully and undesirable to implement partially.

### Local Filtering

Each node can apply local filters to its incoming Wave stream. The client provides the following built-in filtering options.

**Mute by Author.** The user can mute any author (by public key). Waves from muted authors are hidden in the feed and do not trigger Pulse Map animations. Mute is local — it does not affect what other users see.

**Mute by Keyword.** The user can define keyword filters. Waves containing muted keywords are hidden. Keyword matching is case-insensitive and supports basic wildcard patterns.

**Minimum Resonance Filter.** The user can set a minimum Resonance threshold for their feed. Waves from authors with Resonance below the threshold (as computed by the user's node) are hidden. This filter is powerful but blunt — it effectively silences new or low-activity users.

**Hop Count Filter.** The user can limit their feed to Waves within a certain hop count (e.g., only Waves from direct connections and their direct connections, hop count ≤ 2). This creates a "close circle" feed that excludes distant, potentially unfamiliar voices.

**Wave Type Filter.** The user can hide specific Wave types (e.g., hide all Veiled Waves, hide all Beacon Waves).

### Community Filtering

While MURMUR has no built-in community moderation tools (no group moderators, no block lists, no vote-to-hide mechanics), the amplification system serves as an emergent quality filter. Waves that are amplified by many users propagate further and appear more prominently in feeds. Waves that no one amplifies fade into the background after their initial gossip propagation radius. High-Resonance users' amplification choices carry more influence on feed ranking (because amplifications from direct connections are weighted more heavily, and high-Resonance users tend to have more connections).

This creates a de facto quality filter driven by collective curation rather than top-down moderation. It is imperfect — it favors popular content over important content, and it can be gamed by coordinated amplification — but it aligns with the network's decentralized philosophy.

### Harassment Mitigation

The PoW requirement is the first line of defense against harassment. An attacker who wants to send 100 harassing replies to a target must compute 100 PoW stamps, costing minutes to hours of computation depending on difficulty.

Muting is the second line. A user who is being harassed can mute the harasser's public key. Since the harasser's public key is embedded in every Wave they author, the mute is comprehensive — all content from that key is filtered.

For persistent, key-rotating harassers (who create new identities to evade mutes), the Minimum Resonance Filter is effective: new identities start with zero Resonance and will be filtered by any user who sets a minimum threshold above zero.

For anonymous harassment via Specter or Abyssal Waves, the same defenses apply in the Anonymous Layer: mute by Specter public key, minimum Specter Resonance filter. Abyssal Waves, which use one-time keys, cannot be muted by author key — but they can be filtered by Wave type (a user can hide all Abyssal Waves) or by keyword.

The network does not attempt to prevent all harassment. It provides tools for users to protect themselves and relies on the computational cost of content creation and the social dynamics of amplification to keep harassment expensive and low-reward.

---

## Wave Lifecycle Summary

The lifecycle of a typical Surface Wave proceeds through the following stages. The author composes content in the client. The client serializes the Wave fields and begins computing the PoW nonce, displaying a "Minting..." animation. After 0.5–2 seconds, the PoW is found. The client adds the PoW Nonce and PoW Hash, signs the complete Wave, and broadcasts it to the Surface Layer gossip topic. Connected peers receive the Wave, validate it (PoW, signature, timestamp, hop count), store it locally, and rebroadcast to their peers. The Wave propagates outward through the topology, hop by hop, reaching nodes further from the author with each hop. The author's node and recipient nodes render the Wave in their local feeds and trigger Pulse Map ripple animations. If any user amplifies the Wave, it gets rebroadcast with a reset hop count, extending its reach. After 30 days, the Wave expires and nodes begin garbage collecting it from local storage. The Wave ceases to exist in the network.

Anonymous Layer Waves follow the same lifecycle with the substitution of Specter keypairs for main identity keypairs, Anonymous Layer gossip for Surface Layer gossip, and Shroud Network routing for Guarded/Fortress users' traffic.