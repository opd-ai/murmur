# Glossary

**Category:** Reference — Terminology
**Version:** 0.4
**Status:** Draft

---

## Overview

This glossary defines all terminology specific to MURMUR or used in a MURMUR-specific sense throughout the specification documents. Terms are listed alphabetically. Where a term has both a general computing meaning and a MURMUR-specific meaning, the MURMUR-specific meaning is given.

---

## Terms

### Amplification

An amplification is a public endorsement of a Wave by a node other than the Wave's author. When a node amplifies a Wave, it publishes an Amplification Wave (type 0x03) referencing the original Wave's ID. Amplifications increase the original Wave's visibility on the Pulse Map (triggering a brightness flash on the author's node) and contribute to the author's Specter Resonance if the Wave is on the Anonymous Layer. Each node can amplify a given Wave at most once. Amplifications require a Proof of Work stamp.

### Anonymous Layer

The Anonymous Layer is one of MURMUR's two social layers. It is a parallel social space where participants interact through pseudonymous Specter identities that are cryptographically unlinked to their main identities. The Anonymous Layer has its own gossip topics, its own connection graph, its own visual aesthetic (cool blue-purple tones, translucent nodes, particle effects), and its own exclusive social mechanics (Phantom Gifts, mini-games, Masked Events, Phantom Councils, Specter Marks, Whisper Chains). The Anonymous Layer is accessible to users in Hybrid mode and Fortress mode. Open-mode users cannot see or interact with the Anonymous Layer.

### AutoNAT

AutoNAT is a libp2p protocol that allows a node to determine whether it is publicly reachable from the internet or is behind a NAT device. The node asks several connected peers to attempt inbound connections to it. If the peers succeed, the node is publicly reachable. If they fail, the node is behind NAT and must use relay-assisted connectivity (hole punching or circuit relay) to receive inbound connections.

### Beacon Wave

A Beacon Wave (type 0x08) is a high-visibility broadcast on the Anonymous Layer that requires elevated Proof of Work difficulty (24 leading zero bits, approximately 30 seconds of computation). Beacon Waves are used for significant announcements: Shroud Node availability advertisements, Masked Event announcements, Phantom Council formations, and other declarations that justify the elevated computational cost. The higher PoW threshold makes Beacon Waves scarce and socially significant — they are expensive to produce and therefore signal importance.

### Bootstrap Node

A bootstrap node is a well-known MURMUR peer whose network address is hardcoded into the application's default configuration. Bootstrap nodes serve as initial entry points for new nodes joining the network. They are ordinary MURMUR nodes with no special privileges or capabilities — their only distinction is that their addresses are known in advance. The default configuration includes 8–12 bootstrap nodes distributed across geographic regions. Long-running nodes rarely communicate with bootstrap nodes after their initial peer discovery phase.

### Bridge Node

A bridge node is any node running in Hybrid mode that participates in both Surface Layer and Anonymous Layer gossip. Bridge nodes serve as conduits for cross-layer data: they receive cross-layer messages on the Anonymous Layer (such as Phantom Gifts or Specter Marks targeting Surface Layer nodes) and re-publish them on the appropriate Surface Layer gossip topic. Bridge functionality is automatic for all Hybrid-mode nodes — there is no opt-in or configuration required.

### Bulletproofs

Bulletproofs is a zero-knowledge proof system used in MURMUR for Specter Resonance range proofs. A Bulletproof proves that a committed value (hidden inside a Pedersen commitment) lies within a specified range without revealing the value itself. In MURMUR, Bulletproofs are used in Zero-Knowledge Claims to prove that a Specter's Resonance exceeds a required threshold (for example, to join a Phantom Council) without disclosing the exact Resonance score. Bulletproofs produce compact proofs (approximately 672 bytes for a 64-bit range) suitable for inclusion in gossip messages.

### ChaCha20-Poly1305

ChaCha20-Poly1305 is the authenticated encryption algorithm used for all symmetric encryption in MURMUR. ChaCha20 provides confidentiality (a stream cipher with 256-bit keys) and Poly1305 provides integrity and authentication (a one-time authenticator producing 128-bit tags). It is used for encrypting Phantom Council communications, Shroud Network onion layers, Whisper Chain message layers, encrypted key backup files, and any other context requiring symmetric encryption.

### Connection

A connection in MURMUR is a declared, mutual social bond between two identities. On the Surface Layer, connections are formed between main identities and are declared publicly through signed Connection Declaration messages. On the Anonymous Layer, Specter connections are formed between Specter identities. All connections require mutual consent — both parties must sign the connection declaration. Connections influence Pulse Map topology (connected nodes are drawn together by spring forces), content delivery (Waves propagate preferentially through connections), and Specter Resonance (the Connection Diversity signal).

### Connection Declaration

A Connection Declaration is a signed message published on the gossip network that declares a social connection between two identities. The declaration contains both parties' public keys and both parties' signatures, proving mutual consent. Connection Declarations are published on the identity gossip topic and are visible to all participants on the relevant layer.

### Connection Manager

The connection manager is the component of each node that determines which peers to maintain network connections with, subject to the 200-connection maximum. It classifies connections into four priority tiers: Tier 1 (social connections — highest priority, never dropped), Tier 2 (GossipSub mesh partners), Tier 3 (Kademlia DHT neighbors), and Tier 4 (opportunistic connections — lowest priority, dropped first when capacity is reached).

### Content Window

The content window is the 30-day retention period for all MURMUR content. Waves, identity announcements, mechanic events, and all other gossip messages are retained by nodes for at most 30 days, after which they are garbage collected. The content window bounds storage consumption and ensures that MURMUR is not a permanent archive. Masked Event content has a shortened content window of 7 days, reflecting the ephemeral nature of events.

### DCUtR

DCUtR (Direct Connection Upgrade through Relay) is a libp2p protocol that coordinates NAT hole punching. When two NATed nodes are connected through a relay, DCUtR coordinates simultaneous outbound connections from both nodes to each other, attempting to punch through both NATs and establish a direct connection. If successful, the relay connection is replaced by the direct connection.

### Display Name

A display name is an optional, user-chosen label (1–64 UTF-8 characters) that appears below a node's sigil on the Pulse Map. Display names are not unique and are not verified — any user can choose any name. They can be changed at any time. If no display name is set, the node is identified by its public key fingerprint.

### Distributed Hash Table (DHT)

The distributed hash table is a decentralized key-value store used for peer discovery and peer routing in MURMUR. MURMUR uses the Kademlia DHT implementation provided by libp2p. The DHT allows any node to find the network address of any other node given its Peer ID, without requiring a central directory. It is also used for content routing — finding nodes that hold specific publishers' Waves.

### Mini-Game

See Anonymous Mini-Games (Cipher Puzzles, Specter Hunts, Territory Drift, Oracle Pools, Sigil Forge, Shadow Play).

### Echo

An Echo is the re-broadcast of a Wave beyond its original propagation range. When a node encounters a Wave from outside its immediate neighborhood (received through browsing or sync rather than direct gossip propagation), it can Echo the Wave by publishing an Echo Wave (type 0x05) that references the original Wave's ID and includes the original content. Echoes extend a Wave's reach but carry the echoing node's identity rather than the original author's, making it clear that the content has been relayed. On the Pulse Map, Echoes are visualized as secondary shockwaves emanating from the echoing node.

### Eclipse Attack

An eclipse attack is a network-level attack where an adversary surrounds a target node with adversarial peers, isolating it from the honest network. An eclipsed node receives only adversary-controlled messages and can be fed false or censored data. MURMUR mitigates eclipse attacks through connection manager tier diversity (connections spanning multiple DHT regions), GossipSub mesh management, and Kademlia routing table structure.

### Ed25519

Ed25519 is the elliptic curve digital signature algorithm used for all identity and authentication in MURMUR. Every identity (main identity, Specter identity, Masked identity, transport identity) is an Ed25519 keypair. Every message is signed with the author's Ed25519 private key and verified by recipients using the public key. Ed25519 provides 128-bit security against classical attacks, fast constant-time operations, and compact key/signature sizes (32-byte public keys, 64-byte signatures).

### Ego-Centric View

The ego-centric view is the default Pulse Map viewport arrangement where the current user's node is positioned at the center, with their direct connections in the innermost ring and more distant nodes radiating outward. This framing gives users a natural sense of their place in the network. Users can toggle to network-centric view for a more objective perspective.

### Fingerprint

A fingerprint is a truncated hexadecimal representation of a node's public key, used as a fallback identifier when no display name is set. The fingerprint is derived from the public key hash and displayed in colon-separated groups (e.g., "A7F3:2B91:..."). Fingerprints are also used for out-of-band identity verification — two users can compare fingerprints to confirm they are communicating with the intended party.

### Force-Directed Layout

The force-directed layout is the algorithm that computes node positions on the Pulse Map. It models nodes as charged particles that repel each other and connections as springs that attract connected nodes. The algorithm uses four forces (repulsion, spring attraction, center gravity, and damping) to produce a layout where densely connected clusters are positioned close together and weakly connected nodes drift to the periphery. The layout runs continuously at 30 ticks per second and uses hierarchical aggregation for distant nodes to maintain performance.

### Fortress Mode

Fortress mode is the most private of MURMUR's three participation modes. A Fortress-mode user has no Surface Layer identity — their only presence in the network is their Specter identity on the Anonymous Layer. All traffic is routed through Shroud circuits. The Peer ID is derived from a transport keypair unrelated to the Specter key. Fortress mode provides maximum anonymity at the cost of being unable to interact with the Surface Layer. Fortress mode cannot be upgraded to Hybrid or Open mode, as no Surface Layer identity exists.

### Global Passive Adversary (GPA)

A global passive adversary is a threat model adversary class representing an entity (typically a nation-state intelligence agency) that can observe all internet traffic between all MURMUR nodes simultaneously. The GPA can observe timing, volume, source, and destination of encrypted connections but cannot modify traffic. MURMUR provides limited protection against a GPA — Shroud routing raises the cost of traffic analysis but does not guarantee anonymity against long-term statistical correlation.

### GossipSub

GossipSub v1.1 is the libp2p publish-subscribe protocol used for data propagation in MURMUR. Nodes subscribe to topics and receive messages published on those topics by any node in the network. Messages propagate through a mesh overlay with bounded redundancy. GossipSub provides O(log n) propagation latency, resilience to churn, resistance to censorship, and support for message validation before forwarding.

### Halo

The halo is an optional outer glow effect on a Pulse Map node that indicates recent activity. A node that has published a Wave within the last 60 minutes has a soft radial glow extending beyond its ring. The halo's intensity decays linearly, fading to invisible after 60 minutes of inactivity.

### Heat Map

The activity heat map is an optional Pulse Map overlay that colors regions of the background based on the density of Wave publications in the trailing 60 minutes. It uses a blue-to-red gradient (blue for low activity, red for high activity) rendered as a blurred layer behind nodes. The heat map is toggled via a viewport control button and is off by default.

### Hole Punching

Hole punching is a NAT traversal technique where two NATed peers simultaneously initiate outbound connections to each other, exploiting the behavior of NAT devices that allow outbound traffic and its return traffic. In MURMUR, hole punching is coordinated by the DCUtR protocol after an initial relayed connection is established. If hole punching succeeds, the relay is released and the peers communicate directly.

### Hybrid Mode

Hybrid mode is the recommended participation mode for most MURMUR users. A Hybrid-mode user has a main identity on the Surface Layer and a Specter identity on the Anonymous Layer. The two identities are cryptographically unlinked (separate keypairs, separate signatures). Anonymous Layer traffic is routed through Shroud circuits. Hybrid-mode nodes also function as bridge nodes, carrying cross-layer data between the Surface Layer and Anonymous Layer gossip domains.

### Identity Announcement

An Identity Announcement is a signed message published on the identity gossip topic that declares a node's existence and public properties: public key, display name, and protocol capabilities. Identity Announcements are published during onboarding and whenever the user's public properties change (such as a display name update). They include a Proof of Work stamp to prevent spam identity creation.

### Invitation

An invitation is a compact data structure generated by an existing MURMUR user and shared with a potential new user to facilitate joining the network. The invitation contains the inviter's Peer ID (for direct bootstrap), public key fingerprint (for connection verification), and an optional welcome message. Invitations are encoded as URL-safe Base64 strings with the `murmur://invite/` prefix and are approximately 100–150 characters long.

### Kademlia

Kademlia is the distributed hash table algorithm used by MURMUR's DHT implementation. Kademlia organizes peers by XOR distance from the local node's ID, maintaining a routing table of buckets containing peers at various distances. Lookups converge in O(log n) steps. MURMUR uses the libp2p Kademlia implementation for peer discovery, peer routing, and content routing.

### Layer Blend

The layer blend is a slider control on the Pulse Map that adjusts the relative visibility of the Surface Layer and Anonymous Layer. At one extreme, only the Surface Layer is visible. At the other extreme, only the Anonymous Layer is visible. In the middle, both layers are visible with adjustable relative opacity. The layer blend is available only to Hybrid-mode users. Fortress-mode users see only the Anonymous Layer, and Open-mode users see only the Surface Layer.

### libp2p

libp2p is the open-source modular peer-to-peer networking stack that provides MURMUR's networking foundation. It handles peer identity, peer discovery, connection establishment across NATs, transport multiplexing, protocol negotiation, and publish-subscribe messaging. MURMUR uses the Rust implementation (rust-libp2p) for native clients and the JavaScript implementation (js-libp2p) for browser clients.

### Macro View

The macro view is the widest zoom level on the Pulse Map, showing the entire network or a large portion of it. Nodes are rendered as small colored dots without sigils, labels, or halos. Connections are sparse. Cluster structure is visible as bright clouds of dots. Active Masked Events, mini-games, and Phantom Councils are shown as overlay icons. The macro view is useful for understanding overall network structure before zooming in.

### Main Identity

The main identity is a user's primary Ed25519 keypair, used on the Surface Layer. The main identity's public key is the user's permanent, verifiable identifier. The main identity is created during onboarding and is present for Open-mode and Hybrid-mode users. Fortress-mode users do not have a main identity.

### Masked Event

A Masked Event is a time-limited, anonymous social gathering on the Anonymous Layer where all participants interact through single-use Masked keypairs. Within a Masked Event, no participant knows who any other participant is — not their main identity, not their Specter. Events have a declared topic, a fixed duration (30, 60, 120, or 240 minutes), and an optional participant cap (5 to 100). Events are created by Beacon Wave (requiring elevated PoW), and Masked keypairs are destroyed after the event ends, ensuring post-event unlinkability. On the Pulse Map, active events appear as translucent domes containing identical, featureless dots.

### Masked Keypair

A Masked keypair is a single-use Ed25519 keypair generated by a Specter for participation in a Masked Event. The Masked keypair is unrelated to the participant's Specter key or main identity key. After the event, the Masked private key is deleted, making it impossible for anyone (including the participant) to prove authorship of Masked Waves after the fact.

### Masked Pseudonym

A Masked pseudonym is a two-word identifier derived from a Masked public key hash, used during Masked Events. Masked pseudonyms are drawn from a dedicated event-themed wordlist distinct from the Specter pseudonym wordlist (e.g., "Flickering Mask," "Distant Echo," "Burning Question"), preventing confusion between Specter and Masked identities.

### Masked Wave

A Masked Wave (type 0x07) is a message published within a Masked Event using a Masked keypair. Masked Waves function like Specter Waves but are authored by ephemeral Masked identities. They are published on the event's dedicated gossip topic and are retained for 7 days after the event ends (shorter than the standard 30-day content window).

### mDNS

mDNS (multicast DNS) is a local network discovery protocol used by MURMUR to find peers on the same LAN without external network connectivity. mDNS broadcasts the node's presence on the local network and listens for broadcasts from other MURMUR nodes. This enables fully offline local MURMUR networks. mDNS is enabled by default and can be disabled in settings.

### Meso View

The meso view is the medium zoom level on the Pulse Map, showing a neighborhood of 50–200 nodes. Nodes are rendered with core circles, rings, halos, and labels. Sigils are visible but small. Connections display color and style encoding. Wave propagation pulses and anonymous mechanic visualizations are visible. The meso view is the primary navigation and browsing zoom level.

### Micro View

The micro view is the closest zoom level on the Pulse Map, showing 5–20 nodes at full detail. Nodes display large core circles with clearly visible sigils, full labels, all Specter Marks, all Phantom Gift effects, and full-intensity halos. Wave content cards are displayed as floating text panels. The micro view is the primary interaction and reading zoom level.

### Minimap

The minimap is a small overlay in the corner of the Pulse Map viewport that shows the full network layout at macro scale with the current viewport highlighted as a rectangle. Users can click or tap on the minimap to jump to a different region of the network.

### Mode

A mode is one of MURMUR's three participation configurations: Open, Hybrid, or Fortress. The mode determines which layers the user participates in, which gossip topics the node subscribes to, whether a Specter identity exists, and the level of anonymity protection provided. Mode is selected during onboarding. Open mode can be upgraded to Hybrid mode. Hybrid and Fortress modes cannot be downgraded.

### Mute

Muting is a local-only action that hides a specific node's content (Waves, Phantom Gifts, Specter Marks) from the muting user's view. Muting does not affect what other users see and does not communicate anything to the muted node or the network. It is MURMUR's only content management tool, operating purely in the local client.

### MURMUR

MURMUR is the name of the decentralized, peer-to-peer social network described by this specification. The name evokes quiet, persistent communication — a murmur that spreads through a crowd without a single identifiable source. MURMUR has no servers, no central authority, and no trusted third parties. The network exists solely as a mesh of participant-operated nodes.

### Network-Centric View

The network-centric view is an alternative Pulse Map viewport arrangement where the layout is centered on the network's topological center (the node or cluster with the highest closeness centrality) rather than on the user's own node. This view provides a more objective picture of the network's structure. It is toggled from the default ego-centric view via a viewport control.

### Node

In the Pulse Map context, a node is the visual representation of a single identity — a main identity on the Surface Layer or a Specter identity on the Anonymous Layer. A node is a composite visual element consisting of a core circle, sigil, ring, halo, Specter Mark sigils, and Phantom Gift effects. Node size encodes social significance, node color encodes identity, and node opacity encodes layer membership.

In the networking context, a node is a single instance of the MURMUR application participating in the peer-to-peer network.

### Node Detail Panel

The Node Detail Panel is a slide-in interface panel that appears when a user selects a node on the Pulse Map. It displays the node's profile information (display name, sigil, fingerprint, or Specter pseudonym), recent Waves, connection list, Specter Resonance (for Anonymous Layer nodes), and available interaction options (send Wave, send Phantom Gift, place Specter Mark, initiate Whisper Chain, join mini-game, etc.).

### Noise Protocol Framework

The Noise protocol framework is a cryptographic handshake framework used by libp2p for TCP and WebSocket connection encryption. MURMUR uses the Noise XX handshake pattern with Ed25519 keys, which provides mutual authentication, identity hiding from eavesdroppers, and forward-secret session keys.

### Onboarding

Onboarding is the five-phase first-run experience that takes a new user from initial application launch to full network participation. The phases are Welcome (atmospheric introduction), Identity Creation (keypair generation, display name, key backup), Mode Selection (Open, Hybrid, or Fortress), Network Bootstrap (peer connection establishment), and Guided Exploration (interactive Pulse Map tutorial). The flow takes 2–5 minutes.

### Onion Routing

Onion routing is the technique used by the Shroud Network to anonymize Anonymous Layer traffic. A message is encrypted in multiple layers (one per relay hop), like the layers of an onion. Each relay node decrypts one layer, revealing the address of the next hop and the encrypted inner payload, which it forwards. No relay sees both the origin and the destination of the message. MURMUR's Shroud circuits use 3-hop onion routing with fixed-size 4096-byte blobs.

### Open Mode

Open mode is the simplest of MURMUR's three participation modes. An Open-mode user participates only on the Surface Layer with their main identity. They can publish Waves, form connections, and appear on the Pulse Map, but cannot access the Anonymous Layer, see Specter nodes, or use any anonymous mechanics. Open mode can be upgraded to Hybrid mode at any time.

### Pedersen Commitment

A Pedersen commitment is a cryptographic commitment scheme used in MURMUR's Zero-Knowledge Claim system. A Pedersen commitment hides a value (the Specter Resonance score) while allowing the committer to later prove properties of the value (such as exceeding a threshold) without revealing it. Pedersen commitments provide information-theoretic hiding (the commitment reveals nothing about the value, even to an adversary with unlimited computing power) and computational binding (the committer cannot change the committed value after commitment).

### Peer Exchange Protocol

The peer exchange protocol (`/murmur/peerex/1.0`) is a protocol where connected peers periodically share their known peer lists. Every 5 minutes, each node sends a random sample of 20 peers from its routing table to each connected peer. This provides continuous, passive peer discovery that supplements the DHT.

### Peer ID

A Peer ID is a node's network-layer identifier in the libp2p framework. It is derived from the node's Ed25519 public key (or transport keypair for Fortress-mode nodes) as a multihash, formatted as a base58-encoded string. The Peer ID identifies the node to other peers regardless of transport protocol or IP address.

### Phantom Council

A Phantom Council is a persistent, private, anonymous coordination group on the Anonymous Layer. A Council consists of 3–13 high-Resonance Specters who meet on an encrypted private gossip topic. Councils have structured voting mechanics for internal coordination. Council creation requires Fortress mode. Council membership requires a minimum Specter Resonance of 200 and unanimous approval from existing members. Council existence and membership are publicly visible, but Council communications are encrypted and private.

### Phantom Gift

A Phantom Gift is an anonymous cosmetic gift sent from a Specter to any node in the network. A gift manifests as a temporary visual effect on the recipient's Pulse Map node, lasting 7 days. Gifts are organized into three tiers (Basic, Expanded, Premium) unlocked by Specter Resonance milestones (25, 50, 100). A Specter can send at most 3 gifts per 24-hour period. Gifts are one-way gestures visible to all observers, carrying the sender's Specter pseudonym and sigil.

### Phantom (Resonance Milestone)

Phantom is the third Specter Resonance milestone, achieved at Resonance 100. Reaching Phantom unlocks Premium Phantom Gifts (the most elaborate cosmetic effects) and the ability to place Specter Marks. The name "Phantom" denotes a fully realized anonymous presence — a Specter with significant influence on the Anonymous Layer.

### Proof of Work (PoW)

Proof of Work is MURMUR's primary spam resistance mechanism. Every message published on the gossip network must include a PoW stamp — a nonce value that, when hashed together with the message content using SHA-256, produces a hash with a specified number of leading zero bits. Standard Waves require 16 leading zero bits (approximately 0.5 seconds of computation). Beacon Waves require 24 leading zero bits (approximately 30 seconds). PoW makes message publication computationally expensive, deterring mass-scale spam and Sybil attacks.

### Proof of Work Stamp

A PoW stamp is the data structure attached to every gossip message that proves the publisher performed the required Proof of Work computation. It contains the nonce value and the resulting hash. Receiving nodes independently verify the stamp by recomputing the hash and checking the leading zero bits before accepting or forwarding the message.

### Pulse Map

The Pulse Map is MURMUR's primary user interface — a real-time, interactive, two-dimensional visualization of the network as a living topology. Nodes represent identities, connections represent social bonds, and animations represent activity (Wave propagation, amplifications, anonymous mechanics). The Pulse Map uses a force-directed layout driven by actual network topology, creating a spatial representation where proximity reflects social connectivity. The map operates on both layers with visual distinction (warm tones for Surface, cool tones for Anonymous) and supports three zoom levels (macro, meso, micro).

### Pulse Notification

A Pulse Notification is a brief, non-intrusive visual indicator at the edge of the Pulse Map viewport that alerts the user to a significant event (new Wave from a connection, Phantom Gift received, Specter Mark placed, mini-game event, Whisper Chain message, etc.). Tapping a notification pans the viewport to the event's location. Notifications are prioritized (high, medium, low) to prevent overload.

### QUIC

QUIC is the preferred network transport protocol for native MURMUR clients. It provides encrypted, multiplexed, connection-oriented communication over UDP with lower latency than TCP, built-in TLS 1.3 encryption, and native multiplexing without head-of-line blocking. MURMUR uses QUIC as the default transport when UDP connectivity is available.

### Quick-Action Menu

The quick-action menu is a radial context menu that appears when a user right-clicks (desktop) or long-presses (mobile) a node on the Pulse Map. It provides the most common interaction options (send Wave, send gift, view profile, etc.) in a circular arrangement around the selected node. Menu items are context-sensitive, varying based on the target node's layer and the user's relationship to it.

### Recovery Phrase

A recovery phrase is a 24-word BIP-39-compatible mnemonic encoding of a private key, used for key backup and identity recovery. The user writes the words on paper and stores them securely. The private key can be reconstructed from the recovery phrase on any device, enabling identity recovery after device loss.

### Relay Node

A relay node is a publicly reachable MURMUR node that relays connections on behalf of NATed nodes that cannot receive inbound connections directly. Relay service is enabled by default for publicly reachable nodes. Each relay node accepts up to 128 concurrent relay reservations with bandwidth limits of 128 KB/s per relayed connection. Relay service is a community resource with no explicit incentive beyond supporting network connectivity.

### Reply Wave

A Reply Wave (type 0x06) is a Wave that references another Wave's ID as its parent, creating a threaded conversation structure. Replies are displayed as content cards connected to their parent Wave's card by thin lines. Reply chains can be expanded or collapsed on the Pulse Map.

### Resonance

See Specter Resonance.

### Resonance Burst

A Resonance Burst is a temporary Specter Resonance bonus earned through active participation in a Masked Event. The Burst value is computed from the amplifications a participant's Masked Waves received during the event, using the formula `burst_value = 5 * ln(1 + amplifications_received)`. Bursts decay linearly to zero over 7 days. The Burst is applied to the participant's Specter Resonance without revealing the link between their Masked identity and their Specter identity (the mapping is known only to the participant's local client).

### Resonance Milestone

A Resonance milestone is a Specter Resonance threshold that unlocks new capabilities on the Anonymous Layer. The three milestones are Shade (Resonance 25, unlocking Basic Phantom Gifts), Wraith (Resonance 50, unlocking Expanded Phantom Gifts), and Phantom (Resonance 100, unlocking Premium Phantom Gifts and Specter Marks).

### Ring

The ring is a thin circular border surrounding a node's core circle on the Pulse Map. Ring appearance encodes the node's participation mode: no ring for Open mode, a single blue ring for Hybrid mode, and a double silver ring for Fortress mode.

### Shade (Resonance Milestone)

Shade is the first Specter Resonance milestone, achieved at Resonance 25. Reaching Shade unlocks Basic Phantom Gifts — a set of 5 subtle visual effects that can be sent to other nodes. The name "Shade" denotes an emerging anonymous presence — a Specter that has begun to establish itself on the Anonymous Layer.

### Shroud Network

The Shroud Network is MURMUR's built-in onion-routing overlay that anonymizes Anonymous Layer traffic. Hybrid+ and Fortress-mode nodes route their Anonymous Layer gossip publications through 3-hop Shroud circuits, preventing observers from linking a Specter's messages to the author's IP address or Peer ID. The Shroud Network uses fixed-size 4096-byte blobs to prevent size-based traffic analysis and rotates circuits every 10 minutes to limit correlation windows.

### Shroud Node

A Shroud Node is a MURMUR node that volunteers to serve as a relay in the Shroud Network. Shroud Nodes are advertised through Beacon Waves on the Anonymous Layer (requiring 24 leading zero bits of PoW). Serving as a Shroud Node contributes to the Shroud Node Uptime signal in Specter Resonance computation. Shroud Nodes see only encrypted blobs passing through them and cannot read message content or determine circuit endpoints.

### Sigil

A sigil is a procedurally generated geometric visual identity derived from a public key hash. Every identity in MURMUR (main identity, Specter identity, Masked identity) has a unique, deterministic sigil. The sigil generation algorithm uses the public key hash as a seed for a deterministic random number generator that produces geometric primitives (shapes, colors, symmetry patterns). Sigils serve as visual fingerprints — they allow users to recognize identities at a glance without reading text. Surface Layer sigils use warm tones; Anonymous Layer sigils use a restricted cool-tone palette.

### Specter

A Specter is an anonymous identity on the Anonymous Layer. Each Specter is an independent Ed25519 keypair with its own pseudonym, sigil, connection graph, and Resonance score. A Specter is cryptographically unlinked to its owner's main identity. Specters are created during onboarding for Hybrid and Fortress-mode users. The word "Specter" evokes an insubstantial presence — visible but untouchable, identifiable but unattributable.

### Specter Connection

A Specter connection is a mutual social bond between two Specter identities on the Anonymous Layer. Like Surface Layer connections, Specter connections are formed through signed mutual declarations. Specter connections influence Anonymous Layer Pulse Map topology and contribute to the Connection Diversity signal in Specter Resonance.

### Anonymous Mini-Games

Anonymous mini-games are a diverse ecosystem of lightweight interactive games played on the Anonymous Layer that leverage anonymity as a core mechanic. The six primary mini-game types are: **Cipher Puzzles** (collaborative and competitive cryptographic challenges, unlocked at Wraith/Resonance 50), **Specter Hunts** (timed scavenger hunts across Pulse Map topology, unlocked at Shade-Wraith/Resonance 75), **Territory Drift** (persistent ambient territory claim game, unlocked at Shade/Resonance 25), **Oracle Pools** (anonymous prediction markets on network events, unlocked at Phantom/Resonance 100), **Sigil Forge** (timed creative challenges, unlocked at Wraith/Resonance 50), and **Shadow Play** (social deduction game, unlocked at Revenant/Resonance 200). All mini-games contribute to Specter Resonance through the Mini-Game Activity signal. On the Pulse Map, active mini-games appear as distinctive visual events with type-specific icons and effects.

### Specter Mark

A Specter Mark is a visible, anonymous marker placed by a high-Resonance Specter (Resonance 100+) on any node in the network. A Mark manifests as a small glowing sigil (the marker's sigil) attached to the target node's Pulse Map representation, lasting 30 days. Marks carry no protocol-defined semantic meaning — their significance is socially constructed by the community. A Specter can maintain at most 5 active Marks. The target cannot remove Marks but can mute the marker to hide them locally.

### Specter Pseudonym

A Specter pseudonym is a two-word identifier derived from a Specter public key hash, generated by mapping hash bytes to entries in a curated wordlist of approximately 2,048 words themed around mystery, shadow, and presence (e.g., "Hollow Beacon," "Silent Fracture"). The pseudonym is the Specter's primary label on the Anonymous Layer, displayed below the Specter's node on the Pulse Map.

### Specter Resonance

Specter Resonance is a composite metric representing a Specter's standing, influence, and contribution on the Anonymous Layer. Resonance is computed locally by each observing node from publicly observable signals: Wave Publication Consistency, Amplification Reception, Connection Diversity, Mini-Game Record, Shroud Node Uptime, Whisper Chain Contributions, and Event Participation. Resonance determines a Specter node's visual size on the Pulse Map, unlocks anonymous mechanics at milestones (25, 50, 75, 100, 200), weights mini-game outcomes, and qualifies Specters for Phantom Council membership. Because Resonance is computed locally, different observers may compute slightly different values for the same Specter.

### Specter Wave

A Specter Wave (type 0x02) is a Wave published on the Anonymous Layer using a Specter keypair. Specter Waves function identically to Surface Layer Waves (signed content with PoW stamps, supporting replies and amplifications) but are attributed to a Specter identity rather than a main identity. They propagate on the Anonymous Layer gossip topic and are visible only to Anonymous Layer participants.

### Spring Attraction

Spring attraction is one of the four forces in the Pulse Map's force-directed layout algorithm. Every connection acts as a spring pulling the two connected nodes toward each other. The spring's rest length is proportional to the connection's age (older connections have shorter rest lengths), causing long-standing connections to draw nodes closer together over time.

### Surface Layer

The Surface Layer is one of MURMUR's two social layers. It is the public social space where participants interact through their main identities. Activity on the Surface Layer is openly attributable — every Wave is signed by the author's main identity key and visible to all Surface Layer participants. The Surface Layer has its own gossip topics, its own connection graph, and its own visual aesthetic on the Pulse Map (warm tones, full opacity). The Surface Layer is accessible to Open-mode and Hybrid-mode users.

### Sync Protocol

The sync protocol (`/murmur/sync/1.0`) enables nodes that have been offline to retrieve missed data from peers. The returning node sends a sync request specifying the timestamp of the last message it received on each subscribed topic. Peers respond with batches of messages published after that timestamp, up to the 30-day content window. Synced messages are validated identically to live gossip messages. Sync supports selective retrieval (specific topics, specific publishers) to reduce bandwidth.

### Transport Keypair

A transport keypair is a dedicated Ed25519 keypair used by Fortress-mode nodes for libp2p connection establishment. The transport keypair serves as the node's Peer ID source and is distinct from both the main identity key (which does not exist for Fortress users) and the Specter key. The separation prevents network-layer observers from linking the node's Peer ID to its Specter identity.

### Wave

A Wave is the fundamental unit of content in MURMUR. A Wave is a signed, timestamped message with a maximum content size of 2,048 bytes, published on a gossip topic and propagated through the network. Every Wave carries a Proof of Work stamp. Waves are typed: Standard Waves (0x01) on the Surface Layer, Specter Waves (0x02) on the Anonymous Layer, Amplification Waves (0x03), Mini-Game Waves (0x04), Echo Waves (0x05), Reply Waves (0x06), Masked Waves (0x07), and Beacon Waves (0x08). Waves are retained for 30 days (7 days for Masked Waves) before being garbage collected.

### Wave ID

A Wave ID is a unique identifier for a Wave, computed as the SHA-256 hash of the Wave's content (including metadata, signature, and PoW stamp). Wave IDs are used for deduplication (preventing the same Wave from being processed twice), for referencing (replies and amplifications reference their target Wave by ID), and for sync (requesting specific Waves by ID).

### WebRTC

WebRTC is a transport protocol option for browser-to-browser MURMUR connections. It uses the browser's built-in WebRTC stack for NAT traversal (ICE/STUN/TURN) and encryption (DTLS). WebRTC enables direct peer-to-peer communication between browser clients without requiring a relay server.

### WebSocket

WebSocket is the transport protocol used by browser-based MURMUR clients when WebTransport is unavailable. WebSocket connections are established over TCP (typically on port 443 with TLS) and encrypted using the Noise protocol framework with yamux multiplexing. WebSocket on port 443 is also used as a firewall-traversal transport for native clients in restrictive network environments.

### WebTransport

WebTransport is a QUIC-based transport protocol accessible from web browsers, used as the preferred browser transport when supported. It provides the performance benefits of QUIC (low latency, multiplexing, no head-of-line blocking) in a browser-compatible form.

### Whisper Chain

A Whisper Chain is an anonymous, multi-hop private message relay between Specters on the Anonymous Layer. The sender constructs a message encrypted in layers (one per relay hop, analogous to onion routing), with 2–4 intermediate Specters as relays. Each relay sees only the previous and next hop, not the origin, destination, or content. Whisper Chains provide private, point-to-point communication between Specters who may not be directly connected. A Specter can send at most 10 Whisper Chain messages per 24-hour period. Serving as a relay contributes to the Whisper Chain Contributions signal in Specter Resonance.

### Wraith (Resonance Milestone)

Wraith is the second Specter Resonance milestone, achieved at Resonance 50. Reaching Wraith unlocks Expanded Phantom Gifts — 10 additional visual effects of moderate intensity. The name "Wraith" denotes a growing anonymous presence — a Specter with established participation and emerging influence.

### X25519

X25519 is the elliptic curve Diffie-Hellman key exchange function used for all key exchanges in MURMUR. It operates on Curve25519 (the same curve as Ed25519), and Ed25519 keys are converted to X25519 keys for key exchange using a birational map. X25519 produces 32-byte shared secrets that are processed through HKDF-SHA-256 to derive symmetric encryption keys. It is used in Shroud circuit establishment, Whisper Chain construction, Phantom Council group key agreement, and encrypted direct messaging.

### Zero-Knowledge Claim (ZK Claim)

A Zero-Knowledge Claim is a cryptographic proof that a Specter's Resonance exceeds a specified threshold without revealing the exact Resonance value. ZK Claims use Pedersen commitments (to hide the Resonance value) with Bulletproofs-style range proofs (to prove the value exceeds the threshold). ZK Claims are used for Phantom Council admission (proving Resonance exceeds the Council's minimum requirement) and for any other context where a Specter needs to demonstrate a Resonance level without disclosing it. ZK Claims are verified independently by each receiving node, which checks the proof against its own local computation of the claimant's Resonance.