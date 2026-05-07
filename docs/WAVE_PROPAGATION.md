# Wave Propagation

**Category:** Core Mechanics
**Version:** 0.4
**Status:** Draft

---

## Overview

Waves are the fundamental content unit in MURMUR. A Wave is a signed message that propagates through the social graph via multi-hop relay, attenuating with each hop. There is no global feed, no central distribution, and no algorithmic curation. Content reaches a user because someone in their trust chain chose to amplify it. The topology of trust IS the algorithm.

MURMUR supports multiple Wave types across both the Surface Layer and the Anonymous Layer. Each type has distinct propagation rules, encryption behavior, visual rendering, and cross-layer visibility characteristics.

---

## Wave Structure

Every Wave contains the following fields.

**ID.** A unique identifier derived from the hash of the Wave's content and author public key.

**Author.** The Ed25519 public key of the originating node (main identity for Surface Waves, Specter identity for Anonymous Waves).

**Signature.** Ed25519 signature over the Wave payload, verifiable by any node holding the author's public key.

**Timestamp.** Unix timestamp of composition. For Anonymous Layer Waves, timestamps are quantized to 5-minute buckets to prevent timing correlation.

**Content.** The Wave payload. Text (UTF-8, max 2,000 characters for standard Waves), or a CID reference for media Waves (images, audio). Media content is exchanged via Bitswap.

**Hop Count.** Current number of hops from the origin. Initialized to 0 by the author. Incremented by each relaying node.

**Max Hops.** Maximum propagation distance set by the author (default 5, maximum 20). The Wave is not relayed beyond this hop count.

**Proof-of-Work Nonce.** A nonce satisfying the PoW difficulty requirement for the Wave type and layer. PoW is computed by the author at composition time.

**Wave Type.** Enum indicating the Wave variant: Standard, Veiled, Abyssal, Sigil, Council, Summary, Whisper Artifact, or Mini-Game Record.

**Encryption Envelope.** Present only for encrypted Wave types (Veiled, Abyssal, Council). Contains the encrypted payload and key derivation metadata. Structure varies by type.

**Attachments.** Optional array of CID references for media content. Each CID is accompanied by a MIME type hint and a size declaration.

**ZK Claims.** Optional array of zero-knowledge proof attachments (see ZK Claims section).

**Amplification Signature Chain.** An ordered list of (public key, signature) pairs representing each node that amplified the Wave. Each amplifier signs the previous chain state plus their own public key, creating a verifiable amplification trail.

---

## Propagation Model

### Composition

A user composes a Wave in the client. The client computes the PoW nonce (this may take a few seconds depending on difficulty), signs the Wave, sets hop count to 0, and publishes it to the appropriate GossipSub topic.

### Reception and Display

When a node receives a Wave via GossipSub, it performs the following checks in order.

**Signature verification.** The Wave's signature is verified against the author's public key. Invalid signatures are dropped silently.

**PoW verification.** The nonce is verified against the required difficulty for the Wave type and layer. Insufficient PoW is dropped silently.

**Duplicate detection.** The Wave ID is checked against a local cache of recently seen Wave IDs. Duplicates are dropped silently.

**Hop count check.** If the hop count exceeds the max hops, the Wave is dropped.

**Trust path check.** The Wave must have arrived from a directly connected peer. The relaying peer's public key must be in the receiving node's connection list. Waves arriving from non-connected peers (possible in GossipSub due to mesh topology) are dropped. This ensures that content only reaches a user through their trust chain.

If all checks pass, the Wave is displayed to the user. The display position and visual treatment are determined by the Wave's hop count (closer hops are more prominent), the author's Resonance Score (higher Resonance Waves are slightly more visually prominent), and the Wave type (each type has distinct visual rendering).

### Amplification

A user who has received a Wave can choose to amplify it. Amplification re-publishes the Wave to the user's connections with the hop count incremented by 1 and the user's (public key, signature) pair appended to the amplification chain. This extends the Wave's reach by one hop in all directions from the amplifier.

Amplification is the only mechanism by which a Wave gains reach. There is no viral coefficient, no "share" button in the broadcast sense — only the deliberate choice of individual nodes to pass content along to their connections. Each amplification is a signed, attributable act. The amplification chain is visible to all recipients and contributes to the amplifier's Resonance Score.

### Decay

Waves attenuate with distance. The visual prominence of a Wave decreases with each hop from the origin. At hop 0 (direct from author), the Wave is rendered at full brightness and size. At each subsequent hop, brightness decreases by approximately 15% and size by approximately 10%. By max hops, the Wave is visually faint but still readable.

Waves also decay over time. Waves older than 30 days are gradually faded from the Pulse Map display. Waves older than 90 days are removed from local display but remain in local storage until the user's storage quota is reached, at which point oldest Waves are pruned first. There is no global expiration — a Wave persists as long as at least one node stores it locally.

### Propagation Example

Alice composes a Wave with max hops of 5. She publishes it (hop 0). Her direct connections Bob, Carol, and Dave receive it at hop 1. Bob amplifies it — his connections Eve and Frank receive it at hop 2. Eve amplifies it — her connection Grace receives it at hop 3. Grace does not amplify. The Wave's reach is Alice → {Bob, Carol, Dave} → {Eve, Frank} → {Grace}. Seven people saw the Wave. If Alice had no connections, zero people would have seen it. If every recipient amplified, the Wave would reach everyone within 5 hops of Alice in the trust graph.

---

## Surface Layer Waves

### Standard Waves

Standard Waves are the default content type. Text and optional media attachments. Signed by the author's main identity. Published to `/murmur/waves/1.0.0`. Rendered on the Pulse Map as pulsing orbs traveling along connection links. Proof-of-work difficulty: low (target ~100ms on a mid-range smartphone).

### Bulletin References

Bulletins are long-form content (up to 50,000 characters) published to `/murmur/bulletin/1.0.0`. A Bulletin is signed by a rotating Publisher Key that is linked to the author's main identity through a key attestation chain. Publisher Key rotation allows an author to repudiate old content by retiring the signing key, while maintaining a verifiable publication history.

Bulletin References are lightweight Waves published to `/murmur/waves/1.0.0` that contain a CID pointer to the full Bulletin content, a title, and a brief excerpt. They propagate through the trust graph like Standard Waves. Recipients who want to read the full Bulletin fetch it via Bitswap using the CID.

---

## Anonymous Layer Waves

All Anonymous Layer Waves are signed by the author's Specter identity and published to `/murmur/waves-anon/1.0.0`. They propagate through the anonymous trust topology (Specter-to-Specter connections) with the same hop-based model as Surface Layer Waves. Proof-of-work difficulty for all Anonymous Layer Waves is higher than Surface Layer equivalents (target ~2 seconds on a mid-range smartphone) to increase the cost of Sybil-driven spam.

Timestamps on all Anonymous Layer Waves are quantized to 5-minute buckets.

### Specter Waves

Specter Waves are the anonymous equivalent of Standard Waves. Text and optional media. Signed by the Specter identity. Propagate through the anonymous topology. Visible only to nodes with an active Specter Host (Hybrid mode and above).

### Veiled Waves

Veiled Waves are encrypted anonymous Waves whose content is accessible only to participants in the anonymous layer, but whose visual presence is broadcast to the entire network including Open-mode users.

**Encryption.** Veiled Wave content is encrypted with a rolling shared secret. The shared secret is derived from a key agreement protocol among active Specter Host participants. The key derivation uses a ratcheting scheme seeded by anonymous DHT participation proofs — periodic commitments published to the anonymous DHT that demonstrate active participation without revealing identity. The rolling secret rotates on a 24-hour cycle. New Specter Hosts joining the anonymous mesh receive the current epoch's shared secret through a key distribution protocol mediated by existing participants.

**Cross-layer visibility.** When a Veiled Wave is bridged to the Surface Layer by a bridge node, it appears as an encrypted shimmer: the Wave's visual motion (pulsing, propagating along links) is transmitted, but the content is replaced with an animated encryption artifact (pixelated, iridescent shimmer effect rendered by a Kage shader). Open-mode users see the shimmer and can tap it to see: "Veiled Wave. [N] amplifications. Content requires Specter." The shimmer is designed to be beautiful and attention-catching — it is the primary ambient pull mechanic of the Shadow Gradient.

**Engagement.** Veiled Waves track amplification count. High-amplification Veiled Waves produce more intense shimmer effects on the Surface Layer, creating a visible signal that popular content exists in the anonymous layer without revealing what it is.

### Abyssal Waves

Abyssal Waves are multi-tier encrypted Waves originating from Fortress-mode users. They are the deepest, most exclusive content type in MURMUR, and they are designed to be the most visually striking phenomenon on the Pulse Map.

**Multi-tier encryption.** An Abyssal Wave's content is encrypted in concentric layers corresponding to the network mode tiers. The innermost layer is encrypted with a shared secret available only to Fortress-mode Specters. The next layer is encrypted with a key available to Guarded-mode and above. The next layer is encrypted with a key available to Hybrid-mode and above. Each tier sees a different level of content, determined by the author at composition time.

The author composes the Wave with tier-specific content variants: the full content for Fortress readers, a partial or summary version for Guarded readers, a teaser or abstract for Hybrid readers. The author controls what each tier sees. The outermost layer — what Open-mode users see — is always empty of readable content.

**Content degradation.** When a lower-tier user encounters an Abyssal Wave, they decrypt the layers they have keys for and see the content designated for their tier. Layers they cannot decrypt are rendered as visual artifacts — increasingly abstract and distorted as the unreadable depth increases.

**Cross-layer visibility.** Abyssal Wave residue on the Surface Layer is the most visually elaborate effect in the app. It appears as deep-colored ripples with trailing particle effects, animated by a dedicated Kage shader with high visual fidelity. The residue carries no readable content but displays: engagement metrics (amplification count, tier-breakdown of engagement), the originating tier ("Fortress origin"), and a prompt showing the user's current tier and the required tier for content access.

**Design intent.** Abyssal Waves are designed so that the most active, most amplified, most visually arresting content in the network visibly originates from the deepest anonymity tier. Open-mode users see that the most interesting things happening in the network are things they cannot read. This is the deepest pull mechanic of the Shadow Gradient.

### Sigil Waves

Sigil Waves are procedurally generated visual art Waves available to Guarded-mode and above. They contain no text — the content IS the visual.

**Generation.** The author composes a Sigil Wave by providing a seed (text, image CID, or random). The client generates a Kage shader program seeded by the SHA-256 hash of the seed content. The shader produces an animated mandala, fractal pattern, or geometric composition unique to that seed. The Sigil Wave payload contains the shader parameters (not the compiled shader — parameters are compact and the shader template is built into the client) and the content hash.

**Rendering.** Every client renders the Sigil identically from the parameters. Sigils are animated — they rotate, pulse, and shimmer. They are the most beautiful static visual objects in the app.

**Cross-layer visibility.** Sigil Waves bridge to the Surface Layer and are rendered in full visual fidelity. They carry no readable text content, so there is nothing to encrypt or hide — the visual itself is the content. However, only Guarded+ users can create them. Every Sigil on the Surface Layer is implicitly a demonstration that Guarded-mode users produce the most beautiful content in the network.

### Council Waves

Council Waves are Waves published by Phantom Councils — anonymous groups of Fortress-mode Specters with verified membership. They are threshold-signed: a quorum of Council members must sign before the Wave is published.

**Structure.** A Council Wave contains the Council's public Sigil (a unique procedurally generated identifier for the Council), the threshold signature (a Schnorr multi-signature requiring k-of-n Council members), the Wave content, and an optional Council Echo (a brief summary of the deliberation process, included at the Council's discretion).

**Propagation.** Council Waves propagate through the anonymous topology like other Anonymous Layer Waves. They also bridge to the Surface Layer with full readability — Council Waves are not encrypted (unless the Council chooses to make them Veiled or Abyssal). The Council's authority is visible to all: "This Wave was published by Phantom Council [Sigil]. Verified [k]-of-[n] threshold signature."

**Cross-layer visibility.** Council Waves on the Surface Layer carry the full weight of anonymous quorum-verified authority. They are visually distinguished by the Council's Sigil and a unique border effect. The concept of hidden, self-governing anonymous groups publishing verified collective statements is designed to be inherently fascinating and drive conversation about the platform.

### Summary Waves

Summary Waves are special Waves published after the conclusion of time-limited anonymous events (Masked Events, completed Whisper Chains, concluded mini-games). They bridge to the Surface Layer with full readability and serve as post-event artifacts.

**Masked Event Summaries.** Contains: event topic, duration, participant count (anonymous), top-Resonance contributions (anonymous, attributed to Masked keypairs which are now expired), and the Resonance Burst leaderboard.

**Whisper Chain Artifacts.** Contains: the completed collaborative content, chain length (number of hops), and a visual thread representation.

**Mini-Game Records.** Contains: the game type, participant count (anonymous), performance highlights, winner pseudonym (if applicable), and outcome summary.

Summary Waves propagate through the Surface Layer topology like Standard Waves. They are designed to be compelling standalone content — each one tells a story of something that happened in the anonymous layer and is shareable outside the app.

### Whisper Chain Waves

Whisper Chain Waves are intermediate and final content units within a Whisper Chain — an anonymous collaborative content creation mechanic available to Guarded-mode and above.

**Chain construction.** The initiator composes the first segment of content and selects a chain length (3–10 hops). The client constructs an onion-encrypted routing path through the anonymous topology: each hop is a Specter node that has opted in to Whisper Chain participation (a per-node setting). The first segment is encrypted in layers — one layer per hop — so that each intermediate node can decrypt only the current state of the chain and their own instructions.

**Chain propagation.** Each intermediate node receives the chain, decrypts their layer, reads the accumulated content so far, adds their own contribution (constrained by the initiator's parameters: topic, max segment length, content type), re-encrypts for the next hop, and forwards. No intermediate node sees the full chain — only the content accumulated up to their position.

**Completion.** When the chain reaches its final hop, the final node publishes the completed collaborative work as a Summary Wave (Whisper Chain Artifact). The artifact contains the full concatenated content, the chain length, and no identifying information about any participant beyond their ephemeral chain-specific keypairs.

**Cross-layer visibility.** Active Whisper Chains are visible on the Pulse Map as glowing threads snaking through the Anonymous Layer. Each hop is represented as a node on the thread, and progress is visible (e.g., hop 3 of 5 complete). Completed artifacts bridge to the Surface Layer as readable content.

---

## Direct Messages

Direct messages are not Waves — they do not propagate through the topology. DMs are point-to-point encrypted communications between two nodes.

**Encryption.** DMs are end-to-end encrypted using the recipient's Ed25519 public key (converted to X25519 for Diffie-Hellman key agreement). A shared secret is derived via X25519 ECDH, and messages are encrypted with XChaCha20-Poly1305. Each message uses a unique nonce. Forward secrecy is achieved through periodic key ratcheting.

**Delivery.** DMs are delivered via direct libp2p streams when both parties are online. When the recipient is offline, the sender encrypts the message and publishes an encrypted routing hint to `/murmur/dm/1.0.0`. The routing hint contains the recipient's public key (so their node can identify messages destined for it) and the encrypted payload. When the recipient comes online, their node scans the topic for routing hints addressed to them, retrieves the encrypted payloads, and decrypts them.

**Specter DMs.** Specter identities can send and receive DMs independently of the main identity. Specter DMs are encrypted with the recipient Specter's public key and delivered through the anonymous mesh. No link between a Specter DM and a Surface Layer DM is exposed.

**Storage.** DMs are stored locally on both sender and recipient devices. There is no server-side storage. If both devices lose their local data, the messages are gone.

---

## Proof-of-Work

All Waves require a proof-of-work nonce to prevent spam and increase the cost of Sybil-driven flooding.

**Mechanism.** The PoW is a partial hash collision: the SHA-256 hash of (Wave ID + nonce) must have at least D leading zero bits, where D is the difficulty parameter for the Wave type and layer.

**Difficulty parameters.** Surface Layer Standard Waves: D = 16 (approximately 100ms on a mid-range smartphone). Surface Layer Bulletin References: D = 18 (approximately 400ms). Anonymous Layer Specter Waves: D = 20 (approximately 2 seconds). Anonymous Layer Veiled/Abyssal/Sigil/Council Waves: D = 22 (approximately 8 seconds). Whisper Chain intermediate Waves: D = 18 (each hop has moderate cost). These values are initial targets and may be adjusted based on observed spam levels.

**Verification.** Every receiving node verifies the PoW before processing the Wave. Invalid PoW results in silent drop.

---

## Media Waves

Waves can include media attachments (images, audio). Media content is not embedded in the Wave message — it is stored separately and addressed by CID.

**Publishing.** The author adds the media to their local Bitswap blockstore, computes the CID, and includes the CID in the Wave's attachments field. When recipients receive the Wave, they fetch the media content via Bitswap from the author or any peer that has cached it.

**Size limits.** Individual media attachments are limited to 10MB. Total attachments per Wave are limited to 25MB. These limits are enforced at the application layer.

**Anonymous media.** Media in Anonymous Layer Waves is stored in the anonymous Bitswap network (fetched over Tor/I2P transports). The author's Specter identity provides the content — the main identity is never used for anonymous media storage.

**Content types.** Supported MIME types at launch: image/jpeg, image/png, image/webp, audio/opus, audio/mp3. Video is not supported at launch due to bandwidth constraints in the P2P topology.

---

## Wave Interaction

Users can interact with Waves in the following ways.

**Amplify.** Re-publishes the Wave to the user's connections with incremented hop count and the user's signature appended to the amplification chain. This is the primary interaction and the only way content gains reach.

**Reply.** Composes a new Wave with a reference to the parent Wave's ID. Replies propagate independently through the topology — a reply may reach users who never saw the parent, and vice versa. Reply threads are reconstructed locally by matching parent references.

**React.** A lightweight interaction: a single emoji attached to the Wave ID, published as a small message on the same GossipSub topic. Reactions are aggregated locally. Reactions are not amplified — they are only visible to nodes that have both the original Wave and the reaction message.

**Cite.** A new Wave that embeds a reference to another Wave's ID and includes a quoted excerpt. Citations propagate independently. The cited Wave's author receives a citation notification if they are within the propagation range.

---

## Cross-Layer Visibility Summary

The following table describes how each Wave type appears on each layer.

**Standard Wave.** Surface Layer: full content. Anonymous Layer: not present (Surface-only type).

**Bulletin Reference.** Surface Layer: full content with CID link. Anonymous Layer: not present.

**Specter Wave.** Surface Layer: not bridged (Anonymous-only unless explicitly bridged by a future mechanic). Anonymous Layer: full content.

**Veiled Wave.** Surface Layer: encrypted shimmer (visual motion, no content, tap for info). Anonymous Layer: full decrypted content for Hybrid+.

**Abyssal Wave.** Surface Layer: deep-colored residue with particle effects (no content, engagement metrics, tier info). Anonymous Layer: tier-appropriate decrypted content (Fortress sees full, Guarded sees partial, Hybrid sees teaser).

**Sigil Wave.** Surface Layer: full visual rendering (no text to hide). Anonymous Layer: full visual rendering.

**Council Wave.** Surface Layer: full readable content with Council Sigil and threshold signature verification. Anonymous Layer: full content.

**Summary Wave.** Surface Layer: full readable content. Anonymous Layer: full content.

**Whisper Chain Artifact.** Surface Layer: full readable completed chain content. Anonymous Layer: full content.

**Mini-Game Record.** Surface Layer: full readable record with outcome summary. Anonymous Layer: full content.