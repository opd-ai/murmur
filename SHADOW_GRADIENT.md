# Shadow Gradient

**Category:** Core Mechanics — Identity & Anonymity
**Version:** 0.4
**Status:** Draft

---

## Overview

The Shadow Gradient is MURMUR's tiered identity system. It defines four modes of network participation, each offering a different balance between social visibility and anonymity. Every user selects a mode at onboarding and can change it at any time. Mode transitions are immediate but carry consequences: moving from a more anonymous mode to a less anonymous mode is always safe, but moving from a less anonymous mode to a more anonymous mode may require severing certain connections or forfeiting access to Surface Layer features.

The four modes, from most visible to most anonymous, are Open, Hybrid, Guarded, and Fortress. The gradient metaphor is deliberate — these are not binary states but positions on a spectrum, and the network is designed so that users at different positions can still interact with each other across the gradient.

The Shadow Gradient is fundamental to MURMUR's thesis: that a social network should allow a single user to maintain a public presence and a private one simultaneously, with cryptographic guarantees that the two cannot be linked by any observer, including the network itself.

---

## Mode Definitions

### Open Mode

Open is the default mode for new users and the most socially visible position on the gradient. Open-mode users participate fully in the Surface Layer: they have a named profile, form mutual connections, author and amplify Waves, appear on the Pulse Map with their full identity, and accumulate Surface Layer Resonance.

Open-mode users do not have a Specter. They have no presence on the Anonymous Layer. They cannot author anonymous Waves, form Specter connections, participate in Masked Events, send or receive Phantom Gifts, or engage in any Anonymous Layer mechanic.

Open-mode users can see the Anonymous Layer as a visual overlay on the Pulse Map — they see ghostly Specter nodes and faint connection lines — but they cannot interact with it. This visual presence is intentional: it makes the existence of the anonymous network visible to all users, creating curiosity and social pull toward the Hybrid mode.

Open-mode users can receive Phantom Gifts from Specters. The gift appears on their Surface Layer profile as a glowing cosmetic effect with the label "Phantom Gift from [Specter pseudonym]." The Open-mode user cannot respond to the gift, identify the sender, or interact with the sender's Specter identity. This one-way gifting mechanic creates a visible bridge between the layers — Open-mode users see evidence of anonymous generosity without being able to cross into the anonymous world themselves.

Open mode has no special requirements. Any user can select it at any time.

### Hybrid Mode

Hybrid is the first step into the anonymous world. A Hybrid-mode user maintains their full Surface Layer presence (named profile, connections, Waves, Resonance) and additionally gains a Specter — a cryptographically independent anonymous identity on the Anonymous Layer.

The Specter is generated locally on the user's device. It consists of a new Ed25519 keypair with no mathematical relationship to the user's main identity keypair. The Specter's public key is hashed to produce a procedurally generated pseudonym (two words from a curated wordlist, producing combinations like "Veiled Lantern" or "Hollow Tide") and a procedurally generated sigil (a small geometric emblem derived from the key hash). The pseudonym and sigil are the Specter's identity on the Anonymous Layer.

Hybrid-mode users can participate in all Anonymous Layer mechanics: anonymous Waves, Specter connections, Phantom Gifts, Specter Duels, and Masked Events. They accumulate Specter Resonance independently of their Surface Layer Resonance.

Hybrid-mode users also serve as bridge nodes in the cross-layer topology. Their device maintains connections to both Surface Layer peers and Anonymous Layer peers, and they route anonymous traffic between the layers. This bridge function is automatic and does not reveal any information about the user's own Specter identity — the user's node routes traffic for many Specters, making it impossible for an observer to determine which Specter belongs to the bridge node's operator. Bridge Activity contributes to Surface Layer Resonance as documented in the Resonance System specification.

Hybrid mode requires the user to generate a Specter keypair. The keypair generation happens locally and is triggered automatically when the user switches to Hybrid mode. There are no other requirements.

### Guarded Mode

Guarded mode adds a layer of traffic protection on top of the Hybrid mode's dual-layer participation. A Guarded-mode user has the same capabilities as a Hybrid-mode user — full Surface Layer presence, full Anonymous Layer presence via Specter, bridge node function — with one critical addition: all Anonymous Layer traffic originating from or destined to the user's Specter is routed through the Shroud Network.

The Shroud Network is MURMUR's mix-network layer, composed of Shroud Nodes operated by Fortress-mode users. Guarded-mode users do not operate Shroud Nodes; they are clients of the Shroud Network. When a Guarded-mode user's Specter sends an anonymous Wave, forms a Specter connection, or participates in any Anonymous Layer interaction, the traffic is encrypted in multiple layers (onion encryption) and routed through a chain of three Shroud Nodes before reaching its destination. Return traffic follows the reverse path through a different chain of three Shroud Nodes.

This routing makes it significantly harder for a network observer to correlate a Guarded-mode user's Surface Layer identity with their Specter activity. Without Shroud routing (as in Hybrid mode), a sophisticated network-level adversary could potentially perform timing analysis on a Hybrid node's traffic to correlate Surface and Anonymous Layer activity. Guarded mode's Shroud routing adds latency jitter and traffic mixing that defeats naive timing analysis.

Guarded mode requires the Shroud Network to be available — there must be a sufficient number of online Shroud Nodes to construct routing chains. If the Shroud Network is degraded (fewer than 9 online Shroud Nodes), Guarded-mode users are warned that anonymity guarantees are reduced and may be offered the option to temporarily fall back to Hybrid-mode routing or to queue anonymous traffic until the Shroud Network recovers.

### Fortress Mode

Fortress is the most anonymous position on the gradient and the most infrastructure-intensive. A Fortress-mode user operates a Shroud Node, contributing to the mix-network infrastructure that protects Guarded and Fortress users. In exchange, the Fortress-mode user receives the strongest anonymity guarantees the network can provide, access to the most exclusive anonymous social mechanics, and the highest Resonance rewards for infrastructure contribution.

A Fortress-mode user's Surface Layer presence is fully maintained — they have a named profile, connections, Waves, and Resonance like any other user. Their Specter operates through the Shroud Network (like Guarded mode) with the additional benefit that the user's own Shroud Node is part of the mix network, further obscuring their traffic patterns. An adversary would need to compromise multiple Shroud Nodes in the user's routing chain to correlate their Surface and Specter identities.

Fortress mode unlocks exclusive capabilities. Fortress-mode Specters can initiate Phantom Councils — private, anonymous deliberation groups with advanced voting mechanics. Fortress-mode Specters can create and moderate Abyssal Waves — the most deeply anonymous Wave type, with the strongest unlinkability guarantees. Fortress-mode Specters who reach Specter Resonance 500 (Abyss milestone) gain access to the Kage shader effect — the most visually distinctive cosmetic in the anonymous layer.

Operating a Shroud Node requires the user's device to be online and reachable for extended periods. On mobile devices, this is achieved through a persistent background service that maintains network connections and processes mix-network traffic even when the app is not in the foreground. On desktop, the Shroud Node runs as a lightweight daemon. The traffic volume processed by a Shroud Node is modest — mix-network packets are small (fixed 1KB size, padded), and the routing computation is minimal. The primary cost is bandwidth (estimated 50–200 MB/day depending on network size and traffic) and battery life on mobile (estimated 5–15% additional daily drain).

Fortress mode requires the user to accept the Shroud Node operation commitment and have a device capable of maintaining persistent background network connections. The user is warned about bandwidth and battery costs before enabling Fortress mode.

---

## Identity Isolation

The cryptographic separation between a user's main identity and their Specter is the most critical security property of the Shadow Gradient. The system is designed so that no observer — including other users, network infrastructure, and even a compromised subset of Shroud Nodes — can link a main identity to a Specter identity without compromising the user's local device.

### Keypair Independence

The main identity keypair (used for Surface Layer operations) and the Specter keypair (used for Anonymous Layer operations) are generated independently. There is no shared seed, no derivation relationship, no mathematical link. The main keypair is generated at account creation; the Specter keypair is generated when the user first enables Hybrid mode. Both are stored in the device's secure enclave (or equivalent OS-provided secure storage).

### Network Separation

Surface Layer traffic (Waves, connection handshakes, topology updates, heartbeats) is transmitted using the main identity's keypair for signing and encryption. Anonymous Layer traffic (anonymous Waves, Specter connection handshakes, anonymous topology updates) is transmitted using the Specter keypair. The two traffic streams use different libp2p protocol IDs and different multistream negotiation paths, ensuring that a network observer cannot trivially correlate them by protocol fingerprinting.

For Hybrid-mode users (who do not use Shroud routing), the two traffic streams originate from the same IP address and libp2p peer ID. This means that a network-level observer who can see the user's IP address knows that the user has both a main identity and a Specter, and can observe the timing of both traffic streams. The observer cannot determine which Specter belongs to the user (because the user's node routes traffic for many Specters as a bridge node), but timing analysis is a theoretical risk. This is the known limitation of Hybrid mode and the motivation for Guarded and Fortress modes.

For Guarded and Fortress-mode users, Anonymous Layer traffic is routed through the Shroud Network. The user's device wraps each anonymous packet in three layers of onion encryption (one for each Shroud Node in the chain) and sends it to the first Shroud Node. The first Shroud Node peels one layer, forwards to the second, which peels another layer, forwards to the third, which peels the final layer and delivers to the destination. At no point does any single Shroud Node know both the origin and the destination of the traffic. A network observer at the user's IP sees traffic going to a Shroud Node — which is indistinguishable from bridge traffic the user routes for other Specters.

### Traffic Padding

To further defeat timing analysis, all modes implement traffic padding. Each node generates a constant stream of dummy packets at a configurable rate (default: 1 packet per second). Dummy packets are encrypted, fixed-size (1KB), and indistinguishable from real packets by any observer who does not hold the decryption key. This constant-rate traffic stream masks the timing patterns of real activity.

For Guarded and Fortress modes, dummy packets are also routed through the Shroud Network, ensuring that the Shroud Network sees a constant traffic volume from each client regardless of actual activity.

### Device Compromise

If an adversary gains access to the user's device (physical access, malware, or OS-level compromise), they can extract both keypairs and trivially link the main identity to the Specter. This is the fundamental limitation of any local-key-based anonymity system. MURMUR does not attempt to protect against device compromise — the secure enclave provides the best available protection, but a sufficiently capable adversary (nation-state level) can defeat it.

Users are advised that their anonymity guarantee is bounded by their device security. The app's security settings screen includes a "Device Compromise" section that explains this limitation and links to platform-specific guides for device hardening.

---

## Specter Lifecycle

### Creation

A Specter is created when the user first switches from Open mode to Hybrid, Guarded, or Fortress mode. The creation process generates a new Ed25519 keypair, derives the pseudonym and sigil from the public key hash, and initializes the Specter's state (empty connection list, zero Specter Resonance, no ZK Claims).

The Specter's creation event is not announced to the network. The Specter simply begins participating in the Anonymous Layer topology — connecting to other Specters, publishing anonymous topology updates, and becoming visible to other Anonymous Layer participants. There is no "birth announcement" and no way for observers to determine exactly when a Specter was created (the Specter Age signal is based on first observed appearance, which may be later than actual creation if the user waits before making their Specter active).

### Pseudonym and Sigil

The Specter pseudonym is a two-word combination derived deterministically from the Specter's public key hash. The first word is drawn from a curated list of 256 atmospheric adjectives (e.g., "Veiled," "Hollow," "Silent," "Burning," "Fractured"). The second word is drawn from a curated list of 256 evocative nouns (e.g., "Lantern," "Tide," "Cipher," "Ember," "Threshold"). This produces 65,536 possible pseudonyms.

Collisions (two Specters with the same pseudonym) are possible but unlikely in networks smaller than several thousand Specters. In the event of a collision, the Specter's sigil (which is derived from a longer hash and is effectively unique) serves as the disambiguator. The UI always displays both pseudonym and sigil together.

The sigil is a procedurally generated geometric emblem: a small SVG composed of 3–7 geometric primitives (circles, lines, triangles, arcs) arranged according to parameters extracted from the public key hash. The generation algorithm is deterministic — the same public key always produces the same sigil. Sigils are visually distinctive at small sizes (16×16 pixels) and serve as the Specter's "face" in the Anonymous Layer UI.

### Rotation

A user can rotate their Specter at any time. Rotation destroys the current Specter (its keypair is securely deleted) and creates a new one with a fresh keypair, pseudonym, and sigil. All Specter connections, Specter Resonance, ZK Claims, Phantom Gifts received, Specter Marks, duel history, and other Anonymous Layer state are irrevocably lost.

Rotation is a drastic action, equivalent to abandoning an anonymous identity and starting over. It is provided as an escape hatch for users who believe their Specter identity has been compromised, correlated with their main identity, or otherwise rendered unsafe. The UI requires double confirmation and displays a warning about the permanent loss of all Specter state.

The old Specter simply stops appearing in the Anonymous Layer topology. Other Specters who were connected to it will see the connection go dead. There is no announcement of rotation and no link between the old and new Specters.

### Destruction

A user who switches from Hybrid, Guarded, or Fortress mode back to Open mode can choose to either preserve or destroy their Specter. Preserving the Specter means it remains in storage but goes dormant — it does not participate in the Anonymous Layer, does not accumulate Resonance, and is not visible to other Specters. If the user later switches back to a Specter-capable mode, the preserved Specter reactivates with its existing state intact.

Destroying the Specter on mode downgrade works identically to rotation: all state is permanently lost. The user is warned and must confirm.

---

## Cross-Layer Interaction

The Shadow Gradient creates two parallel social networks that coexist on the same physical infrastructure. The design includes specific, controlled points of interaction between the layers.

### Visual Overlay

The Pulse Map renders both layers simultaneously. The Surface Layer is the primary visual layer: named nodes with profile photos (or generated avatars), colored connection links, Wave ripple animations. The Anonymous Layer is rendered as a translucent overlay: ghostly nodes with sigils instead of faces, dimmer connection links, more ethereal visual effects.

Open-mode users see the Anonymous Layer overlay but cannot interact with it — no clicking on Specter nodes, no viewing Specter profiles, no sending messages to Specters. The overlay is purely atmospheric, creating a sense that "something else is happening here."

Hybrid, Guarded, and Fortress-mode users can toggle between a Surface-focused view (where the Anonymous Layer is a faint overlay) and an Anonymous-focused view (where the Surface Layer dims and the Anonymous Layer becomes the primary visual layer). In the Anonymous-focused view, the user interacts through their Specter identity — their own node on the map is their Specter, and other visible nodes are other Specters.

### Phantom Gifts (Cross-Layer)

Phantom Gifts are the primary one-way communication channel from the Anonymous Layer to the Surface Layer. A Specter can send a Phantom Gift to any Surface Layer node. The gift appears on the recipient's profile as a cosmetic effect. The recipient can see the gift's visual effect and the sending Specter's pseudonym, but cannot respond, identify the sender's main identity, or interact with the sender's Specter.

This mechanic allows anonymous users to express appreciation, support, or connection to public users without compromising their anonymity. It creates a tangible social link between the layers — Surface Layer users know that anonymous users are watching, engaging, and occasionally reaching out.

### Bridge Routing

Hybrid, Guarded, and Fortress-mode users serve as bridge nodes that route traffic between the Surface Layer and the Anonymous Layer. This routing is automatic and transparent. The bridge node's device maintains connections to both Surface Layer peers and Anonymous Layer peers and forwards traffic between them according to the routing protocol.

Bridge routing is the mechanism by which the Anonymous Layer exists at all — without bridge nodes, anonymous traffic would have no path to travel. The more bridge nodes in the network, the more robust and low-latency the Anonymous Layer becomes. This is why Bridge Activity is rewarded with Surface Layer Resonance — it incentivizes users to operate in Hybrid+ modes, which provides infrastructure for the anonymous network.

### Specter Marks (Cross-Layer)

Specter Marks are visible markers that a Specter (Resonance 100+) can place on any Surface Layer node. A mark appears as a small, glowing sigil attached to the marked user's Pulse Map node. The mark carries the marking Specter's pseudonym and sigil.

Marks are mysterious and ambiguous by design. The marked user knows they have been marked by a Specter and can see the Specter's pseudonym, but does not know why they were marked or what the mark means. Other users can see the mark on the marked user's node. Marks decay after 30 days unless renewed by the marking Specter.

Marks serve multiple social functions depending on community interpretation: acknowledgment ("I see you and respect you"), challenge ("I am watching you"), or mystery ("What does this mean?"). The deliberate ambiguity is a design choice — the mark mechanic is a canvas for emergent social behavior rather than a prescribed interaction.

### Wave Bridging

Certain Wave types can cross the layer boundary. A Veiled Wave is authored by a Specter but visible on both layers — it appears in the Surface Layer feed with the Specter's pseudonym and sigil as the author. Surface Layer users can read and amplify Veiled Waves, but cannot identify the author or respond to them anonymously.

A Sigil Wave is a Surface Layer Wave that includes an embedded Specter sigil — the author reveals that they have a Specter identity without revealing which Specter it is. The embedded sigil is not the author's Specter sigil; it is a randomly selected sigil from the network, creating plausible deniability. The mechanic allows Surface Layer users to signal "I participate in the anonymous world" without compromising their anonymity.

These Wave types are detailed fully in the Waves specification.

---

## Mode Transitions

### Open → Hybrid

The user generates a Specter keypair (automatic, local). The user's node begins participating in the Anonymous Layer topology and serving as a bridge node. No Surface Layer state is affected. This transition is immediate and has no negative consequences.

### Open → Guarded

Equivalent to Open → Hybrid, with the additional step of configuring Shroud Network routing. The user's node begins routing Specter traffic through Shroud chains. Requires the Shroud Network to be available (minimum 9 online Shroud Nodes).

### Open → Fortress

Equivalent to Open → Guarded, with the additional step of initializing a Shroud Node. The user's device begins accepting and processing mix-network traffic from other users. Requires accepting the bandwidth and battery cost commitment. Requires the device to support persistent background networking.

### Hybrid → Open

The user's node stops participating in the Anonymous Layer topology and stops serving as a bridge node. The user is prompted to preserve or destroy their Specter. Surface Layer state is unaffected.

### Hybrid → Guarded

The user's node begins routing Specter traffic through Shroud chains. No state is lost. Requires the Shroud Network to be available.

### Hybrid → Fortress

The user's node begins routing Specter traffic through Shroud chains and initializes a Shroud Node. No state is lost. Requires accepting the operational commitment.

### Guarded → Open

The user's node stops Shroud routing and stops participating in the Anonymous Layer. The user is prompted to preserve or destroy their Specter. Surface Layer state is unaffected.

### Guarded → Hybrid

The user's node stops Shroud routing but continues participating in the Anonymous Layer with direct routing. The user is warned that anonymity guarantees are reduced. No state is lost.

### Guarded → Fortress

The user's node initializes a Shroud Node. No state is lost. Requires accepting the operational commitment.

### Fortress → Open

The user's node shuts down its Shroud Node, stops Shroud routing, and stops participating in the Anonymous Layer. The user is prompted to preserve or destroy their Specter. Other Shroud Network participants who were routing through this node are automatically rerouted. Surface Layer state is unaffected.

### Fortress → Hybrid

The user's node shuts down its Shroud Node and stops Shroud routing but continues participating in the Anonymous Layer with direct routing. Anonymity guarantee reduction warning. Automatic rerouting of dependent traffic. No Specter state is lost, but Phantom Council membership requires Fortress mode and is suspended (not destroyed) — the user's seat is held for 72 hours, after which it is forfeited.

### Fortress → Guarded

The user's node shuts down its Shroud Node but continues Shroud routing as a client. Automatic rerouting of dependent traffic. Phantom Council membership is suspended for 72 hours as above. Shroud Node Resonance bonus ceases accruing.

---

## Shroud Network

### Architecture

The Shroud Network is a mix network composed of Shroud Nodes operated by Fortress-mode users. It provides traffic anonymization for Guarded and Fortress-mode users' Anonymous Layer activity.

The mix protocol is a simplified Sphinx-style construction. Each packet is 1KB, padded to fixed length. The sender constructs a routing header containing three layers of onion encryption, one per hop. Each Shroud Node in the chain peels one layer, extracts the next hop address, and forwards the packet. The final Shroud Node delivers the packet to the destination.

### Chain Selection

When a Guarded or Fortress-mode user needs to send an anonymous packet, their client selects a chain of three Shroud Nodes from the set of available nodes. Chain selection is weighted by Shroud Node Resonance (higher-Resonance nodes are preferred) and constrained by diversity (the three nodes must be in different network clusters to prevent a single cluster operator from controlling the entire chain).

Chain selection is refreshed periodically (default: every 10 minutes) and on demand when a chain node goes offline. The client maintains two active chains at all times — a primary and a backup — to minimize latency disruption during chain rotation.

### Mix Delay

Each Shroud Node introduces a random delay before forwarding each packet. The delay is drawn from an exponential distribution with a mean of 200 milliseconds. This mixing delay prevents a global network observer from correlating incoming and outgoing packets at a Shroud Node by timing. The cumulative delay across three hops is typically 300–900 milliseconds, adding noticeable but acceptable latency to Anonymous Layer interactions.

### Cover Traffic

Shroud Nodes generate cover traffic: dummy packets sent to randomly selected other Shroud Nodes at a constant rate (default: 2 packets per second). Cover traffic is indistinguishable from real traffic and ensures that the Shroud Network maintains a constant traffic volume regardless of actual user activity. This defeats traffic analysis attacks that rely on observing traffic volume changes.

### Shroud Node Discovery

Available Shroud Nodes are discovered through the Anonymous Layer topology gossip protocol. Each Fortress-mode Specter includes a Shroud Node advertisement in its topology updates: the Shroud Node's public key, its reachable addresses, and its current load (number of active chains routing through it). Clients use this information to build their local view of available Shroud Nodes for chain selection.

### Trust Model

The Shroud Network's security relies on the assumption that no single adversary controls all three nodes in a user's routing chain. If an adversary controls the first and third nodes (entry and exit), they can correlate the user's identity with their Specter activity despite the middle node's mixing. If the adversary controls only one or two non-adjacent nodes, the mixing and onion encryption prevent correlation.

The diversity constraint in chain selection (nodes from different clusters) provides some protection against cluster-level adversaries. The Resonance weighting in chain selection provides soft Sybil resistance — an adversary's Shroud Nodes will have low Resonance initially and will be less likely to be selected.

Users are informed in the security settings screen that Shroud routing provides strong but not absolute anonymity. The theoretical threat model (colluding Shroud Nodes, global network observer) is explained in plain language. Users who require absolute anonymity are advised to use MURMUR over an external anonymity network (Tor, I2P) as an additional layer.

---

## Mode Distribution and Network Health

The health of the Anonymous Layer depends on sufficient participation at each gradient level. The network needs bridge nodes (Hybrid+) to route anonymous traffic and Shroud Nodes (Fortress) to provide mix-network anonymization.

The system includes soft incentives at each level. Hybrid mode is incentivized by the social appeal of anonymous participation (Phantom Gifts, Masked Events, Specter Duels) and the Resonance bonus for Bridge Activity. Guarded mode is incentivized by the security improvement for privacy-conscious users. Fortress mode is incentivized by the high Shroud Node Resonance bonus, exclusive access to Phantom Councils and Abyssal Waves, and the social prestige of operating critical infrastructure.

The target mode distribution for a healthy network is approximately 30% Open, 40% Hybrid, 20% Guarded, and 10% Fortress. These are design targets, not enforced quotas. If the actual distribution deviates significantly (e.g., too few Fortress nodes to sustain the Shroud Network), the app may display informational prompts encouraging users to consider upgrading their mode — but never forces or coerces mode changes.

The network health dashboard (accessible from the Pulse Map's status panel) displays the current mode distribution, Shroud Network capacity (number of online Shroud Nodes, estimated chain availability), and Anonymous Layer latency estimates. This transparency allows users to make informed decisions about their mode selection based on current network conditions.