# MURMUR — Unified Design Document

**Version:** 1.0
**Status:** Draft Specification
**Date:** April 2026

---

## Part I — Vision and Foundations

### 1. What MURMUR Is

MURMUR is a decentralized, peer-to-peer social network with a dual-layer identity architecture. There are no servers. There is no company. There is no algorithm deciding what users see or who they connect with. Every participant's device is both client and server — a node in a living mesh that exists only because its participants choose to run it.

The network presents itself through a spatial visualization called the Pulse Map: a force-directed graph where every user is a glowing node, every relationship is a visible edge, and every piece of content propagates outward through the mesh like a ripple through water. Users do not scroll feeds. They navigate a living map of human connection.

Beneath the visible social surface lies an Anonymous Layer — a parallel world of pseudonymous identities called Specters, routed through onion-style circuits to prevent traffic analysis. Users can participate on the Surface Layer under their chosen display name, on the Anonymous Layer under a cryptographic pseudonym, or on both simultaneously. The two layers interact through carefully designed cross-layer mechanics — anonymous gifts, marks, duels, and events — that create social dynamics impossible on any conventional platform.

MURMUR is not a product. It is a protocol, a set of applications implementing that protocol, and the community of people who run those applications. It has no revenue model, no investor obligations, and no growth targets imposed by external stakeholders. Its only measure of success is whether people choose to use it and choose to stay.

### 2. Design Principles

Seven principles govern every design decision in MURMUR. When principles conflict — and they do — the resolution is determined by their ordering. A principle earlier in the list takes precedence over a principle later in the list.

**Privacy is structural, not contractual.** Privacy guarantees are enforced by cryptography and network architecture, not by policies or promises. If the system's operators wanted to violate user privacy, the architecture should make it impossible — or at minimum, computationally infeasible — for them to do so. MURMUR has no operators, which makes this principle both easier and harder to satisfy: there is no one to trust, and no one to betray that trust.

**No permanent record by default.** Content on MURMUR is ephemeral. Waves (the content primitive) have a maximum TTL of 30 days. The network does not maintain a permanent archive. Users who want permanence must actively choose it — and they can only ensure permanence of their own content on their own device. The network forgets by default and remembers only by active choice.

**Identity is self-sovereign.** Users create their own identities through local key generation. No registration server, no email verification, no phone number, no government ID. Identity is a keypair. Verification is cryptographic. Authority is mathematical.

**The network is the interface.** MURMUR does not hide its network topology behind an abstraction layer. The network's structure — who is connected to whom, how information flows, where clusters form — is the primary interface. The Pulse Map is not a visualization of metadata; it is the social space itself.

**Anonymity is a first-class feature, not a loophole.** The Anonymous Layer is not a workaround or an afterthought. It is a designed, supported, celebrated part of the system. Anonymous participation is not second-class participation — it has its own mechanics (Resonance, Duels, Councils, Gifts, Marks) that create social value visible to the entire network.

**Growth must be organic.** MURMUR has no mechanism for paid user acquisition, algorithmic engagement optimization, or attention capture. Growth comes from the network's value to its participants and their desire to share that value with others. If the network is not worth using, it should not grow. If it is worth using, it will.

**Complexity is revealed, not imposed.** The system has significant depth — the Anonymous Layer, the Resonance system, the Specter mechanics, the Phantom Councils — but none of this complexity is presented to a new user on first launch. Complexity is revealed progressively as the user's curiosity and engagement deepen. A user who never explores the Anonymous Layer should still have a complete, satisfying experience on the Surface Layer.

### 3. Architectural Overview

MURMUR's architecture has six major subsystems, each described in detail in subsequent parts of this document.

The **Networking Layer** handles peer discovery, connection management, and message propagation using libp2p. It provides the transport substrate on which everything else is built: gossip-based message propagation via GossipSub, a Kademlia DHT for peer and content discovery, NAT traversal for residential connections, and relay circuits for nodes behind restrictive firewalls.

The **Identity System** manages cryptographic identities for both the Surface Layer and the Anonymous Layer. Surface identities are Ed25519 keypairs with human-readable display names, deterministic visual sigils, and public Connection Declarations. Anonymous identities (Specters) are separate Curve25519 keypairs with procedurally generated pseudonyms and sigils, routed through onion circuits to prevent linkage to their creators' main identities.

The **Content System** defines Waves — the atomic content unit — and their propagation, storage, and expiration mechanics. Waves are signed, timestamped, and size-limited (2,048 bytes). They propagate through the gossip mesh, are cached by peers for up to 30 days, and expire automatically. The system supports replies, amplifications (re-broadcasts), and content-addressed references.

The **Anonymous Layer** encompasses the Specter identity system, the Shroud Network (onion-routed relay infrastructure), the Resonance reputation system, and the anonymous social mechanics: Phantom Gifts, Specter Marks, Specter Duels, Masked Events, and Phantom Councils.

The **Pulse Map** is the primary visual interface — a real-time, force-directed graph visualization of the network's social topology. It renders nodes, edges, Wave propagation animations, anonymous effects, and environmental details in a spatial layout that users navigate by panning, zooming, and tapping.

The **Onboarding System** is the sequence of screens and interactions that transform a new user from a first-launch novice to an oriented participant with a formed identity, an established presence on the Pulse Map, and an understanding of the system's basic mechanics.

These subsystems are deeply interdependent. The Networking Layer carries Identity System messages. The Content System uses Identity System keys for signing and verification. The Anonymous Layer uses the Networking Layer's relay infrastructure and the Content System's Wave format. The Pulse Map renders data from every other subsystem. The Onboarding System introduces all other subsystems in a carefully sequenced progression. No subsystem can be fully understood in isolation; this document presents them sequentially but the reader should understand that they form a tightly coupled whole.

---

## Part II — Networking Layer

### 4. Transport and Protocol Stack

MURMUR is built on libp2p, a modular peer-to-peer networking framework. The choice of libp2p provides transport multiplexing (simultaneous communication over TCP, QUIC, and WebSocket), protocol negotiation (peers automatically agree on compatible protocols during connection), peer identity (each node is identified by a Peer ID derived from its public key), and a well-tested implementation with active community support.

Each MURMUR node maintains persistent connections to a set of peers — its mesh neighborhood. The target mesh size is 6–12 direct peers for a typical residential node. Connections are established through a multi-stage process: the node discovers potential peers through the DHT, evaluates their suitability (latency, reliability, geographic diversity), initiates a libp2p connection with mutual authentication, and upgrades the connection to a gossip subscription.

Transport selection is automatic and adaptive. QUIC is preferred for its lower handshake latency and built-in encryption. TCP with Noise protocol encryption is the fallback for environments where QUIC is blocked. WebSocket transport is available for browser-based nodes and for traversing restrictive corporate firewalls that only permit HTTP traffic.

### 5. Peer Discovery

Peer discovery uses the Kademlia Distributed Hash Table (DHT), a structured overlay network where each peer maintains a routing table organized by XOR distance from its own Peer ID. The DHT serves two functions: peer discovery (finding nodes to connect with) and content routing (finding nodes that have specific data).

On first launch, a node bootstraps into the DHT by connecting to a set of hardcoded bootstrap nodes — long-running, publicly accessible nodes maintained by community volunteers. The bootstrap nodes provide the new node's initial routing table entries. From these initial entries, the node performs iterative lookups to populate its routing table, progressively discovering peers across the network.

Bootstrap node dependence is a centralization risk. If all bootstrap nodes go offline or are compromised, new nodes cannot join the network. Mitigation is multi-layered: the hardcoded bootstrap list contains nodes operated by independent community members across multiple jurisdictions, the application accepts user-specified bootstrap nodes (allowing communities to run their own), and peer exchange during gossip enables nodes to discover peers without DHT interaction once they have at least one active connection.

### 6. Connection Management

The connection manager maintains a tiered priority system for peer connections. Priority is determined by connection age (older connections are more trusted), gossip usefulness (peers that deliver unique, non-duplicate messages are more valuable), latency (lower-latency peers enable faster propagation), and identity relationship (peers with whom the user has a Connection Declaration are highest priority).

When the connection count exceeds the target range, the connection manager prunes the lowest-priority connections. When the count falls below the target range, the manager initiates new peer discovery. The connection manager also maintains a geographic diversity heuristic: it prefers to keep connections to peers in different network regions (as estimated by latency clustering) to ensure that the node's gossip mesh spans the network rather than clustering in a single neighborhood.

Connection quality is monitored continuously. Ping-pong heartbeat messages are exchanged every 30 seconds. A peer that fails three consecutive heartbeats is marked as disconnected and its connection slot is released for a new peer. Reconnection is attempted with exponential backoff for peers that have been reliable in the past.

### 7. Message Propagation

All MURMUR content — Waves, identity declarations, anonymous broadcasts, duel records — propagates through GossipSub, a pubsub protocol designed for decentralized networks. GossipSub maintains a full-message mesh (a subset of connected peers to whom messages are forwarded in full) and a metadata-only mesh (a larger set of peers to whom only message hashes are sent). When a peer on the metadata mesh reports a hash that the local node has not seen, the node requests the full message, ensuring reliable delivery even when the full-message mesh has gaps.

Messages are organized into topics. The primary topics are: `murmur/waves/v1` for standard content, `murmur/identity/v1` for identity declarations and connection announcements, `murmur/beacon/v1` for high-priority network-wide broadcasts (Shroud Node announcements, Masked Event invitations), and `murmur/anonymous/v1` for Anonymous Layer content (Specter Waves, Duel records, Council actions).

Each message carries a signature from its author (or a Specter signature for anonymous content). Peers validate signatures before forwarding — a message with an invalid signature is dropped and the sending peer's reputation is decremented. This prevents message injection and modification during propagation.

Deduplication is hash-based. Each message is identified by a SHA-256 hash of its content. A peer that receives a message it has already seen (by hash) drops the duplicate. The deduplication cache retains hashes for 24 hours, which is sufficient given MURMUR's message TTLs.

### 8. NAT Traversal

Most residential internet connections are behind Network Address Translation (NAT), which prevents inbound connections. MURMUR uses several techniques to traverse NAT.

The primary technique is hole punching via the libp2p DCUtR (Direct Connection Upgrade through Relay) protocol. When two NAT-bound nodes want to connect, they coordinate through a relay node: both nodes simultaneously send outbound packets to each other's observed external addresses, creating pinholes in their respective NATs. If successful, a direct peer-to-peer connection is established through the pinholes and the relay is no longer needed.

When hole punching fails (due to symmetric NAT or restrictive firewall rules), the connection falls back to relay mode. The relay node forwards traffic between the two peers, adding latency but maintaining connectivity. Relay nodes are contributed by community volunteers running nodes with public IP addresses. Running a relay is opt-in and incurs bandwidth costs, so the system cannot assume unlimited relay capacity.

AutoNAT probing runs at startup and periodically thereafter. The node asks peers to attempt inbound connections to its observed addresses. If inbound connections succeed, the node is publicly reachable and does not need relay services. If inbound connections fail, the node is behind NAT and activates hole punching and relay fallback strategies.

### 9. Bandwidth and Resource Management

MURMUR is designed to run on consumer hardware — laptops, desktops, phones, and single-board computers — without consuming unreasonable resources. The bandwidth budget for a typical node is approximately 50 KB/s sustained, with peaks of up to 200 KB/s during high-activity periods. The storage budget for cached content is approximately 100 MB, tunable by the user.

Bandwidth is managed through gossip topic subscription control. A node can reduce its bandwidth consumption by unsubscribing from topics it does not need (e.g., a Surface-only node can unsubscribe from `murmur/anonymous/v1`), by reducing its full-message mesh size (receiving more messages via metadata-mesh pull rather than full-mesh push), or by increasing its deduplication aggressiveness.

Storage is managed through an expiration sweep that runs hourly, deleting cached messages that have exceeded their TTL. The node also implements an LRU (Least Recently Used) eviction policy: when the cache exceeds its size budget, the oldest-accessed messages are evicted first, regardless of remaining TTL.

---

## Part III — Identity System

### 10. Surface Layer Identity

A MURMUR identity begins with key generation. On first launch, the application generates an Ed25519 keypair using the device's cryptographic random number generator. The private key is stored locally in an encrypted keystore (encrypted with a user-supplied passphrase via Argon2id key derivation). The public key is hashed (SHA-256, truncated to 160 bits) to produce the Peer ID — the node's unique, permanent identifier on the network.

The Peer ID is not human-readable. To create a human layer, the user chooses a display name (1–32 Unicode characters, not guaranteed unique) and the system generates a deterministic visual sigil from the public key's hash. The sigil is a small, distinctive geometric pattern rendered procedurally from the hash bytes, serving as a visual fingerprint that helps users recognize identities at a glance. Two different public keys will produce visually distinct sigils with overwhelming probability.

The identity is published to the network as an Identity Declaration message on the `murmur/identity/v1` gossip topic. The declaration contains the public key, display name, sigil parameters, a timestamp, and a signature. Other nodes cache the declaration and use it to verify future messages from this identity.

### 11. Connections

A connection between two users is established by mutual exchange of Connection Declaration messages. User A publishes a Connection Declaration naming User B's Peer ID, signed with User A's private key. User B publishes a reciprocal Connection Declaration naming User A's Peer ID. When both declarations exist, the connection is established and visible on the Pulse Map as an edge between the two nodes.

Connections are bilateral. A unilateral declaration (A declares a connection to B, but B does not reciprocate) is not rendered as an edge on the Pulse Map. This prevents unwanted connection impositions and ensures that all visible edges represent mutual relationships.

Connection declarations are revocable. Either party can publish a Connection Revocation message that invalidates the existing connection. Revocations are propagated through the same gossip topic as declarations and take effect immediately upon receipt.

Connections serve three functions: social (they represent acknowledged relationships between users), structural (they influence the Pulse Map layout by creating attractive forces between connected nodes), and routing (connected peers are given highest priority in the connection manager, ensuring reliable gossip delivery between socially connected users).

### 12. Key Management and Recovery

The private key is the identity. There is no password reset, no account recovery, no customer support. If the private key is lost, the identity is lost permanently. This is a deliberate design choice consistent with the self-sovereign identity principle — no external authority can restore, freeze, or revoke an identity, which means no external authority can help recover one either.

Users are strongly encouraged to create a key backup during onboarding. The backup is the private key encrypted with the user's passphrase, encoded as a QR code or Base64 string that can be printed, written down, or stored on a separate device. The onboarding flow presents the backup creation step with clear language: "If you lose this device and have no backup, your identity is gone forever. No one can recover it."

Key export and import enable identity migration between devices. A user can export their encrypted private key from one device and import it on another, seamlessly transferring their identity. The process requires the passphrase and produces a brief connectivity gap (the old device's connections must be re-established from the new device's network position) but preserves the identity, connections, and history.

Multi-device simultaneous use is not supported in the current design. The identity exists on one device at a time. Supporting simultaneous multi-device use would require a key synchronization protocol and a conflict resolution mechanism for simultaneous actions, both of which add significant complexity and are deferred to future work.

### 13. Privacy Modes

MURMUR offers three privacy modes, selectable during onboarding and changeable at any time in settings.

**Open Mode.** The user participates only on the Surface Layer. Their node is visible on the Pulse Map with their display name and sigil. They publish and receive Waves under their main identity. They can see the effects of the Anonymous Layer (Phantom Gifts, Specter Marks, Duel sparks) but cannot participate in anonymous mechanics. Open Mode is the simplest experience and is recommended for users who do not need anonymity.

**Hybrid Mode.** The user participates on both layers simultaneously. They have a main identity (visible on the Surface Layer) and a Specter identity (visible on the Anonymous Layer). The two identities are cryptographically unlinkable — they use separate keypairs, separate sigils, and separate communication channels. The user can switch between layers fluidly, publishing Waves under their main identity or Specter identity as they choose.

**Fortress Mode.** The user participates exclusively on the Anonymous Layer. They have no Surface Layer presence. Their node does not appear on the Surface Layer Pulse Map. All communication is routed through the Shroud Network. Fortress Mode provides the strongest privacy guarantees but sacrifices Surface Layer social participation entirely.

Mode transitions are designed to be safe. Upgrading from Open to Hybrid generates a new Specter keypair with no cryptographic relationship to the main identity. Downgrading from Hybrid to Open destroys the local Specter keypair and the Specter identity becomes orphaned — it continues to exist on the network as an inactive Specter but cannot be controlled. Mode changes are local operations; they are not announced to the network, preventing mode change events from being used for correlation.

---

## Part IV — Content System

### 14. Waves

The Wave is MURMUR's atomic content unit. A Wave is a compact, signed message with the following fields: a unique Wave ID (SHA-256 hash of content plus timestamp plus author), the author's public key (or Specter public key for anonymous Waves), the content body (UTF-8 text, maximum 2,048 bytes), a timestamp (Unix epoch milliseconds), an optional parent Wave ID (for replies), an optional reference Wave ID (for amplifications), a TTL (time-to-live in hours, default 168 — one week — maximum 720 — 30 days), a Proof of Work nonce, and an Ed25519 signature over all preceding fields.

The 2,048-byte content limit is deliberate. It is large enough for a substantial paragraph or a short multi-paragraph post, but small enough to prevent the network from being overwhelmed by large payloads. Binary content (images, audio, video) is not supported in the Wave format. MURMUR is a text network. This limitation is a feature: it reduces bandwidth requirements, simplifies content caching, and focuses the social experience on language rather than media.

### 15. Proof of Work

Every Wave requires a Proof of Work (PoW) — a computational puzzle that the author must solve before publishing. The puzzle requires finding a nonce such that the SHA-256 hash of the Wave content concatenated with the nonce has a specified number of leading zero bits. The current difficulty target is calibrated so that a modern consumer CPU solves the puzzle in approximately 2–5 seconds.

The PoW serves three functions. It rate-limits publication — a user cannot publish faster than one Wave every few seconds, preventing spam floods. It imposes a real cost on bulk content creation, discouraging automated content bots. And it provides a lightweight Sybil resistance mechanism — creating many fake identities and publishing from all of them simultaneously is computationally expensive.

The PoW difficulty is static in the current design. Adaptive difficulty (adjusting based on network-wide publication rate) would require global state coordination that is impractical in a decentralized network. If the static difficulty proves too low (allowing spam) or too high (frustrating legitimate users) in practice, the difficulty can be adjusted in a protocol version update.

### 16. Propagation and Caching

When a user publishes a Wave, the application broadcasts it to the `murmur/waves/v1` GossipSub topic. The Wave propagates through the gossip mesh: each peer that receives the Wave validates its signature and PoW, caches it locally, and forwards it to its mesh neighbors. Propagation is probabilistic — not every node receives every Wave, because the gossip mesh does not guarantee full delivery. Waves from well-connected nodes propagate further and faster than Waves from poorly connected nodes at the network's fringe.

Cached Waves are served to peers on request. When a node comes online after a period of absence, it performs a sync operation — requesting recent Waves from its connected peers. Peers respond with Waves they have cached that the requesting node has not yet seen (determined by exchanging Bloom filters of known Wave IDs). This catch-up mechanism ensures that intermittently connected nodes can maintain a reasonably complete view of their neighborhood's recent content.

Waves expire when their TTL elapses. Expired Waves are deleted from the cache during the hourly expiration sweep. There is no mechanism to extend a Wave's TTL after publication. If a user wants their content to persist beyond the maximum 30-day TTL, they must republish it as a new Wave. This mechanic reinforces the principle of ephemerality: persistence requires active, ongoing choice.

### 17. Replies and Amplifications

Replies reference a parent Wave by ID, creating a threaded conversation structure. A reply Wave includes the parent Wave's ID in its parent field, along with the reply content. Clients display replies as a tree beneath the parent Wave, enabling branching conversations.

Amplifications are re-broadcasts. When a user amplifies a Wave, they publish a new Wave with the amplified Wave's ID in its reference field and their own signature. The amplification carries the original content to the amplifier's neighborhood, extending the original Wave's reach beyond its natural gossip propagation. An amplification is a social act — the amplifier is endorsing the content and lending their network position to boost its visibility.

Neither replies nor amplifications require the original author's consent or involvement. Any user can reply to any Wave and amplify any Wave. There is no mechanism to disable replies or restrict amplification. This is consistent with the decentralized philosophy: once a Wave is published, it belongs to the network, and the network's participants are free to respond to it as they choose.

### 18. Content Absence as Feature

There are no likes, no view counts, no engagement metrics, no reshare counts, no follower counts. The absence of quantified social metrics is a deliberate design choice. Metrics create perverse incentives — users optimize for metric accumulation rather than authentic expression, and content is evaluated by its numbers rather than its substance.

In MURMUR, the only visible feedback on a Wave is the replies it generates and the propagation ripple visible on the Pulse Map. A Wave that resonates generates replies, amplifications, and extended propagation. A Wave that does not resonate fades quietly into expiration. There is no permanent record of failure (unlike a post with zero likes that remains visible as a monument to unpopularity). The absence of metrics creates a social environment where publication carries less social risk and more intrinsic motivation.

---

## Part V — Anonymous Layer

### 19. Specter Identity

A Specter is a pseudonymous identity on the Anonymous Layer. It is defined by a Curve25519 keypair generated locally on the user's device, completely independent of their main Ed25519 keypair. The Specter has a procedurally generated pseudonym (two words: an adjective and a noun, drawn from a curated wordlist of 4,096 entries each, yielding approximately 16 million possible combinations) and a procedurally generated sigil (visually distinct from main identity sigils by color palette and geometric style, to prevent cross-layer visual confusion).

The Specter identity is stored in a separate encrypted keystore partition from the main identity. The two keystores use independent encryption keys derived from separate passphrase inputs (or the same passphrase, at the user's discretion — but the key derivation salt differs, so the same passphrase produces different encryption keys). This separation ensures that compromise of one keystore does not compromise the other.

A Specter publishes Waves, participates in Duels, sends Phantom Gifts, places Specter Marks, attends Masked Events, and joins Phantom Councils — all under its pseudonymous identity, routed through the Shroud Network so that traffic analysis cannot link the Specter's actions to a specific network address.

### 20. The Shroud Network

The Shroud Network is MURMUR's onion-routing infrastructure for anonymous communication. It is conceptually similar to Tor but is implemented within the libp2p stack and optimized for MURMUR's gossip-based communication patterns rather than general-purpose web browsing.

A Shroud Circuit is a chain of three relay nodes (Shroud Nodes) between the user's device and the entry point of the gossip mesh. When a Specter publishes a Wave or performs any anonymous action, the message is encrypted in three layers: the innermost layer is encrypted to the exit node's public key, the middle layer to the middle node's public key, and the outermost layer to the entry node's public key. The user sends the triply-encrypted message to the entry node, which decrypts the outer layer and forwards to the middle node, which decrypts the middle layer and forwards to the exit node, which decrypts the inner layer and injects the message into the gossip mesh.

No single relay node knows both the origin and the destination of a message. The entry node knows the user's IP address but not the message content or destination. The exit node knows the message content (it injects it into gossip) but not the user's IP address. The middle node knows neither.

Shroud Nodes are volunteer-operated nodes that opt in to relay duty. They announce their availability on the `murmur/beacon/v1` topic. The announcements include the node's public key, available bandwidth, and uptime history. Users select Shroud Nodes for their circuits based on these properties, with a preference for diverse node operators (avoiding circuits where all three nodes are operated by the same entity, as estimated by IP address diversity and Peer ID independence).

Circuit rotation occurs every 10 minutes. A new circuit is built using a new set of Shroud Nodes, and traffic transitions to the new circuit. This limits the window during which a compromised Shroud Node can observe traffic patterns. Circuit construction uses an onion handshake protocol: the user establishes a Diffie-Hellman shared secret with each relay in sequence, negotiating forward-secret session keys for each circuit leg.

### 21. Specter Resonance

Resonance is the Specter reputation metric — a numerical value (starting at 0, uncapped) that reflects the quality and consistency of a Specter's anonymous participation over time. Resonance is computed locally by each node based on observable data; it is not a globally agreed-upon value. Different nodes may compute slightly different Resonance scores for the same Specter based on which of the Specter's actions they have observed.

Resonance increases through four signal categories.

**Wave Publication Consistency.** A Specter that publishes Waves regularly (daily or near-daily) receives a steady Resonance increase. The signal rewards consistency, not volume — publishing 10 Waves in one day and then going silent for a week scores lower than publishing 1–2 Waves per day for seven days.

**Duel Participation Quality.** A Specter that participates in Specter Duels and receives favorable audience evaluations gains Resonance from each duel. The gain is proportional to the audience size and the evaluation score. Dueling regularly against strong opponents builds Resonance faster than dueling occasionally against weak opponents.

**Phantom Gift Activity.** A Specter that sends Phantom Gifts to Surface Layer users gains a small amount of Resonance per gift. This signal rewards cross-layer engagement — Specters who contribute to the visible social fabric of the Surface Layer are valued more highly than Specters who participate only within the Anonymous Layer.

**Community Endorsement.** When a Specter's Marks and Gifts are well-received (measured by whether marked users reciprocate with engagement, and whether gifted users explore the Anonymous Layer), the Specter gains Resonance from the community's implicit endorsement. This signal is noisy and difficult to measure, so it contributes a smaller weight than the other signals.

Resonance decays over time. A Specter that stops participating sees its Resonance gradually decrease, ensuring that the metric reflects current engagement rather than historical accumulation. The decay rate is slow enough that a Specter can take a week-long break without significant Resonance loss, but fast enough that a Specter absent for several months returns to near-zero.

### 22. Resonance Milestones and Unlocks

Resonance milestones unlock new capabilities at defined thresholds, creating a progression system that rewards long-term anonymous engagement.

At Resonance 25, the Specter achieves the rank of **Shade** and unlocks the ability to send Phantom Gifts. Phantom Gifts are visual effects applied to a Surface Layer user's node — glowing auras, particle trails, subtle animations. The recipient sees the gift and the Specter's pseudonym but cannot identify the Specter's main identity. Gifts are one-way: the recipient cannot respond directly to the Specter.

At Resonance 50, the Specter achieves the rank of **Wraith** and unlocks the ability to initiate Specter Duels. A duel is a structured public debate between two Specters on a topic proposed by the initiator. The duel is visible to any node that subscribes to the Anonymous Layer topic. An audience gathers and votes on the exchange. Duel records are published as Waves and persist for the standard TTL.

At Resonance 75, the Specter achieves the rank of **Shade-Wraith** and unlocks the ability to place Specter Marks on Surface Layer nodes. A Specter Mark is a small, persistent sigil attached to a node's visual representation on the Pulse Map — a visible sign that an anonymous entity has noticed and evaluated this user. Marks are accumulative: a node can carry multiple Marks from different Specters, creating a visible mosaic of anonymous attention.

At Resonance 100, the Specter achieves the rank of **Phantom** and unlocks the ability to create and host Masked Events. Masked Events are temporary social spaces where all participants — even Specters — shed their pseudonyms and participate as completely anonymous entities. Inside a Masked Event, there are no names, no sigils, no Resonance scores. Every participant is a blank node. The event runs for a fixed duration (typically 1–4 hours) and then dissolves, leaving no record of individual participation.

At Resonance 200, the Specter achieves the rank of **Council-Eligible** and can be invited to join or form a Phantom Council. A Phantom Council is a permanent, secret deliberative body of 5–13 Specters who collaborate on governance, coordination, and community stewardship within the Anonymous Layer. Council discussions are encrypted to council member keys and are invisible to non-members. Council decisions — which take the form of collective Marks, coordinated Gifts, or published position statements — emerge from the council as if from a single entity, preserving the anonymity of individual council deliberations.

### 23. Phantom Gifts

A Phantom Gift is the primary positive cross-layer interaction. When a Shade-or-above Specter sends a Phantom Gift to a Surface Layer node, a visual effect appears on the recipient's node on the Pulse Map. The effect varies by gift type (chosen by the sender from a curated set): a gentle glow, a particle aura, a rhythmic pulse, a color shift, or a trailing luminescence.

The recipient sees the gift and can view the sender's Specter pseudonym and sigil but cannot respond, trace, or identify the sender further. The gift persists on the recipient's node for 24 hours before fading. Multiple gifts from different Specters can coexist on a single node, creating a rich visual display that signals the node's significance to the Anonymous Layer.

Gifts cannot carry text or links — they are purely visual. This prevents gifts from being used as a harassment vector (no anonymous messages) while preserving their positive social function (anonymous recognition and appreciation).

### 24. Specter Marks

A Specter Mark is a persistent anonymous annotation on a Surface Layer node. Unlike Phantom Gifts (which are temporary and purely aesthetic), Marks are durable (they persist for as long as the marking Specter maintains sufficient Resonance) and semantically meaningful (each Mark carries a category: "insight," "courage," "creativity," "integrity," "disruption," or "mystery").

A node that accumulates Marks from multiple high-Resonance Specters becomes visually distinctive on the Pulse Map — the Marks form a mosaic of small sigils around the node, creating a visible aura of anonymous evaluation. This aura is visible to all users and serves as a decentralized, anonymous reputation signal: the Anonymous Layer is saying "this person matters, and here is why."

Marks can be removed by their placing Specter at any time. They also decay if the placing Specter's Resonance drops below the required threshold (75). This ensures that Marks represent active, engaged anonymous participation rather than historical artifacts.

### 25. Specter Duels

A Specter Duel is a structured public debate between two Specters. The initiator proposes a topic (a question or thesis, maximum 256 characters), selects a duel format (timed rounds, open exchange, or single statement), and issues a challenge to either a specific Specter (by pseudonym) or an open challenge to any willing opponent.

When a challenge is accepted, the duel begins. In the timed-rounds format, each Specter publishes alternating statements of up to 1,024 bytes, with a 3-minute window per round, for a fixed number of rounds (typically 3–5). In the open-exchange format, both Specters publish freely for a fixed duration (typically 15–30 minutes). In the single-statement format, each Specter publishes a single statement of up to 2,048 bytes, and the audience evaluates based on the two statements alone.

The audience consists of any nodes that observe the duel Waves on the `murmur/anonymous/v1` topic. Audience members vote by publishing signed evaluation Waves indicating which Specter they found more persuasive. Votes are weighted by the voter's own Specter Resonance (if the voter is a Specter) or counted equally (if the voter is a Surface Layer observer). The duel outcome — a tally of weighted votes — is computed locally by each observing node and displayed on the duel's record.

Duels serve a social function beyond their content. They are spectacles — visible events on the Pulse Map that draw attention, generate conversation, and create social surplus moments. A well-attended duel on a provocative topic is a community event, and the duel record (published as a Wave) persists as a reference point for future discussions.

### 26. Masked Events

A Masked Event is a temporary social space created by a Phantom-rank Specter where all participants are completely anonymous. Inside the event, every participant is represented by an identical, blank node — no names, no sigils, no Resonance indicators. Participants can publish text messages visible only within the event space. The event has a fixed duration (set by the creator, minimum 30 minutes, maximum 4 hours) and a maximum participant count (set by the creator, minimum 5, maximum 50).

Entry to a Masked Event is by invitation (the creator publishes a signed event announcement on the Beacon topic, and interested participants send join requests routed through the Shroud Network). The creator approves join requests without seeing the requester's identity — approval is first-come, first-served until the participant cap is reached.

When the event's timer expires, the event space is destroyed. All messages published within the event are deleted from all participants' devices. No record of the event's content or participant list persists. The only trace is the initial event announcement on the Beacon topic (which contains the event's topic and duration but not its content or participants).

Masked Events serve as MURMUR's most radical anonymity feature — a space where even Specter pseudonyms are stripped away and participants interact as pure voices. This creates uniquely intimate social dynamics: without any identity to maintain or protect, participants often engage with unusual honesty, vulnerability, and creativity.

### 27. Phantom Councils

A Phantom Council is a persistent, secret deliberative body. Formation requires a minimum of 5 and maximum of 13 Council-Eligible Specters (Resonance 200+) who agree to form a council and perform a multi-party key generation ceremony. The ceremony produces a shared council keypair: the council's public key is published to the network (making the council's existence known), while the private key is threshold-shared among members (requiring a configurable quorum — typically 3 of 5, or 5 of 9 — to produce a valid signature).

Council internal communications are encrypted to the council's public key and are readable only by members who hold a key share. Council decisions (Marks, Gifts, position statements) are signed with the council key, identifying them as council actions without revealing which members participated in the decision.

Council membership is anonymous even to other council members. Each member interacts with the council through their Specter identity, and the multi-party cryptography ensures that no member can determine which Specter identities the other members control (beyond the pseudonyms they chose to reveal during formation). This double anonymity — anonymous to the public, and partially anonymous to each other — creates a unique deliberative dynamic.

Councils serve a proto-governance function. In a network without formal moderation, councils provide a mechanism for community stewardship: a council can collectively Mark nodes that consistently contribute valuable content, collectively Gift nodes that appear to be new and struggling, or publish position statements that influence community norms. A council's influence is proportional to its members' collective Resonance and the respect the community accords to its actions.

---

## Part VI — Pulse Map

### 28. Spatial-Social Paradigm

The Pulse Map is a real-time, interactive, force-directed graph visualization that serves as MURMUR's primary interface. Instead of a scrolling feed, users navigate a spatial representation of the social network: each user is a node (a glowing point of light), each mutual connection is an edge (a visible line), and the spatial arrangement reflects social topology (densely connected groups cluster together, isolated nodes drift to the periphery).

The Pulse Map is not a supplementary visualization layered on top of a conventional interface. It is the interface. Content discovery, social navigation, identity exploration, and anonymous observation all happen within the Map. A user exploring MURMUR is literally navigating a social space — panning across neighborhoods, zooming into clusters, tapping nodes to read Waves, watching propagation ripples, and observing anonymous effects drifting through the map.

The paradigm shift from feed to map has profound implications for social dynamics. In a feed, content competes for sequential attention — each post displaces the previous one. On a map, content coexists spatially — multiple Waves can be visible simultaneously in different regions, and the user chooses what to approach rather than being forced to process content in a predetermined order. This spatial coexistence reduces the winner-take-all dynamics that dominate feed-based platforms and gives more visibility to content from less popular sources.

### 29. Layout Engine

The Pulse Map uses a force-directed layout algorithm running continuously in the client application. The algorithm simulates a physical system where nodes are charged particles (repelling each other) and edges are springs (pulling connected nodes together). The simulation converges to a layout where connected clusters are tightly grouped and unconnected regions are spatially separated, producing an organic, readable map of the network's social topology.

The force model has four components: a repulsive force between all node pairs (preventing overlap), an attractive force along edges (pulling connected nodes together), a centering force (preventing the layout from drifting indefinitely in one direction), and a damping force (gradually reducing node velocity to allow convergence). The force parameters are tuned to produce aesthetically pleasing layouts at various zoom levels — tight clusters at close zoom, visible macro-structure at wide zoom.

The layout computation is performed incrementally. Each frame, the algorithm computes forces for a subset of nodes (prioritizing nodes near the viewport) and updates their positions. Full-network layout computation is performed in the background at lower priority. This approach ensures smooth frame rates on consumer hardware while maintaining layout accuracy over time.

For large networks (10,000+ nodes), the layout engine uses a Barnes-Hut approximation for the repulsive force calculation: distant nodes are grouped into aggregate "supernodes" whose collective repulsion is computed as a single force, reducing the computational complexity from O(n²) to O(n log n). At very large scales (50,000+ nodes), the layout engine uses hierarchical decomposition: distant regions of the network are represented as collapsed clusters that expand when the user zooms in, enabling map navigation at any scale without rendering the entire network simultaneously.

### 30. Node Rendering

Each node on the Pulse Map is rendered as a luminous point with visual properties that encode social information.

**Size** encodes connection count. Nodes with more connections are rendered larger. The scaling is logarithmic to prevent highly connected nodes from dominating the visual field while maintaining visible differentiation. A node with 1 connection is rendered at base size. A node with 10 connections is rendered at roughly 1.5x base size. A node with 100 connections is rendered at roughly 2x base size.

**Color** encodes activity recency. A node that has published a Wave within the last hour glows brightly in its base color. A node that has been active within the last day glows at moderate intensity. A node that has been inactive for more than a day dims toward a muted gray. A node that has been offline for more than a week dims to near-invisible, rendering only as a faint point. This activity-based coloring makes the Pulse Map a real-time activity heatmap: active regions glow, dormant regions fade.

**Pulse animation** encodes recent publication. When a node publishes a Wave, its visual representation pulses — a brief, visible expansion and contraction that draws the viewer's eye. The pulse propagates outward along edges, creating a ripple effect that shows the Wave's path through the gossip mesh. A Wave that propagates widely creates a visible cascade of pulses across the map — a spectacular visual event during high-engagement moments.

**Sigil overlay.** At sufficient zoom, each node displays the user's deterministic sigil — the geometric pattern derived from their public key hash. The sigil provides visual identity recognition at a glance, allowing users to identify specific nodes without reading display names.

**Specter Marks and Phantom Gifts** are rendered as additional visual layers around marked or gifted nodes: small sigils orbiting the node (for Marks) and ambient particle effects or glow overlays (for Gifts). These layers make anonymously recognized nodes visually distinctive, serving as visible social signals from the Anonymous Layer.

### 31. Edge Rendering

Edges between connected nodes are rendered as translucent lines whose visual properties encode relationship information.

**Opacity** encodes connection age. New connections are rendered as bright, fully opaque lines. Older connections fade to lower opacity, creating a visual distinction between fresh and established relationships. This aging effect gives the map a temporal dimension: a viewer can see at a glance which relationships are new (bright edges) and which are long-standing (faded edges).

**Thickness** encodes interaction frequency. Edges between nodes that frequently exchange Waves (replies, amplifications) are rendered thicker than edges between nodes that are connected but rarely interact. This encoding makes the map a communication topology: thick edges show active communication channels, thin edges show dormant connections.

**Pulse propagation.** When a Wave propagates along an edge (from publisher to connected peer), the edge briefly brightens and a subtle particle effect travels along its length. This propagation animation is the primary visual representation of information flow on the network: watching the Pulse Map during a high-activity period reveals cascading pulses flowing through the mesh like electrical signals through a neural network.

### 32. Wave Visualization on the Map

Waves are not displayed as a list or feed. They are spatial objects associated with the node that published them. When a user taps a node, a panel expands showing the node's recent Waves (within the user's cache), sorted by timestamp. The panel also shows Waves from the node's immediate neighborhood (one-hop connections) as contextual content.

Wave propagation is visualized as an expanding ripple originating from the publishing node. The ripple travels outward along edges, losing intensity with distance. The ripple's color matches the publishing node's base color, creating a visual signature that identifies the Wave's source even at a distance. Multiple simultaneous ripples (from different publishers) create an interference pattern — a complex, beautiful visualization of the network's information flow.

High-engagement Waves — those that generate many replies — create visible reply cascades: a ripple from the original Wave, followed by ripples from each reply, creating a sustained pulse pattern in the neighborhood where the conversation is occurring. These reply cascades serve as beacons of activity, drawing the attention of distant users who navigate toward the pulsing region to discover the conversation.

### 33. Anonymous Layer Visualization

The Anonymous Layer is rendered as a parallel visual layer on the Pulse Map, visually distinguished by a different color palette (deep purples, indigos, and electric blues versus the Surface Layer's warm golds, ambers, and whites) and a different rendering style (more ethereal, with greater transparency and particle effects).

Specters appear on the Anonymous Layer as ghostly nodes, drifting on paths influenced by their Shroud circuit routing rather than by social connection topology. (In reality, Specter positions are randomized — they cannot be placed by social topology without revealing social relationships — but the random placement is smoothly animated to create the illusion of purposeful drift.)

Users in Hybrid Mode see both layers simultaneously, with the ability to adjust the visual prominence of each layer via a slider. At one extreme, the Surface Layer is fully visible and the Anonymous Layer is a faint, ghostly overlay. At the other extreme, the Anonymous Layer is prominent and the Surface Layer fades to a dim background. This layered rendering creates the visual impression of two social worlds coexisting in the same space — a visible world of named identities and a shadow world of anonymous specters.

Duel events are rendered as sparking connections between two Specter nodes — a visible electric arc that draws attention and signals active debate. Masked Events appear as enclosed domes on the Anonymous Layer — visible from outside as shimmering hemispheres but opaque to non-participants. Council formations are rendered as constellation-like patterns: the council's member nodes (anonymized) connected by faint, shimmering lines forming a geometric shape.

### 34. Navigation and Interaction

The Pulse Map supports standard touch and pointer gestures for spatial navigation.

**Pan.** Click-and-drag or touch-and-drag to move the viewport across the map. Panning is smooth and momentum-based: a flick gesture imparts velocity that gradually decays, allowing the user to traverse large distances with a single gesture.

**Zoom.** Scroll wheel, pinch gesture, or double-tap to zoom in and out. Zoom is continuous and smooth, with progressive detail revelation: at wide zoom, only major clusters are visible. At medium zoom, individual nodes and edges appear. At close zoom, sigils, display names, and Wave content preview become visible.

**Tap on node.** Tapping a node selects it, displaying a detail panel with the user's display name, sigil, connection count, recent Waves, and any Specter Marks or Phantom Gifts. From the detail panel, the user can initiate a connection request, read the node's Waves, or navigate to the node's neighborhood.

**Tap on edge.** Tapping an edge displays information about the connection: the two connected users, the connection age, and recent interaction frequency.

**Tap on empty space.** Tapping empty map space deselects any selected node and dismisses detail panels, returning to the default map view.

**Search.** A search bar allows the user to search for a node by display name or Peer ID fragment. Search results highlight matching nodes on the map and offer to center the viewport on the selected result.

### 35. Performance Considerations

The Pulse Map must maintain smooth frame rates (target: 60 FPS, minimum acceptable: 30 FPS) on consumer hardware while rendering potentially thousands of nodes, edges, and animated effects. Performance is achieved through several techniques: viewport culling (only nodes and edges within the visible viewport are rendered), level-of-detail reduction (distant nodes are rendered as simple points without sigils or effects), batch rendering (nodes and edges are drawn in batched draw calls to minimize GPU overhead), animation budgeting (the number of simultaneous particle effects and ripple animations is capped, with oldest effects being retired when the cap is reached), and incremental layout (the force-directed layout is computed incrementally, not fully recalculated each frame).

On low-end devices, the application automatically reduces visual quality: disabling particle effects, reducing edge rendering (showing only high-priority edges), increasing the level-of-detail threshold, and lowering the target frame rate to 30 FPS. These degraded-mode settings are applied transparently based on detected hardware capabilities and measured frame times.

---

## Part VII — Onboarding

### 36. Onboarding Philosophy

The onboarding experience must accomplish three objectives simultaneously: orient the user (teach them how to use a fundamentally unfamiliar interface), create emotional investment (make them care about their identity and presence on the network), and advance the growth loop (move them toward a social surplus moment as quickly as possible).

These objectives are achieved through a six-phase sequence that takes approximately 5 minutes and delivers at least one moment of genuine delight — an experience that surprises, intrigues, or moves the user — before asking for any commitment.

### 37. Phase 1 — Welcome

The first screen the user sees is a black canvas with a single, slowly pulsing point of light at its center. There is no logo, no branding, no feature list. A brief text appears: "This is you. The rest is up to you." A single "Begin" button appears after a 2-second delay. The delay is intentional — it creates a moment of contemplation and prevents the user from clicking through without absorbing the visual.

### 38. Phase 2 — Identity Creation

The identity creation process is presented as a ceremony rather than a form. The user enters a display name, and the application generates the Ed25519 keypair in the background. The deterministic sigil renders in real time as the keypair is generated — the user watches their visual identity crystallize from mathematical noise into a unique geometric pattern. A brief explanation accompanies the animation: "Your identity is a mathematical key. No email, no phone, no password. Just you."

The key backup step follows. The application generates a QR code and a Base64 string representing the encrypted private key. The text is direct: "Save this. If you lose your device, this is the only way back. No one can recover it for you — not even us." The user is given the option to save the backup (screenshot, print, or export) or skip (with a confirmation dialog: "Are you sure? This identity cannot be recovered without a backup.").

### 39. Phase 3 — Mode Selection

The user is presented with the three privacy modes: Open, Hybrid, and Fortress. Each mode is explained in plain, non-technical language.

Open is described as: "You appear on the map under your chosen name. You can see the anonymous world, but you don't participate in it." Hybrid is described as: "You have two identities — your public self and an anonymous alter ego. You choose which to use, moment by moment." Fortress is described as: "You exist only in the anonymous layer. No public name, no visible node. Maximum privacy."

The default selection is Open, with Hybrid highlighted as "recommended for the full experience." The user can change their mode at any time in settings. The choice is not binding and is presented as a low-stakes starting point rather than a permanent commitment.

If the user selects Hybrid or Fortress, the Specter identity creation ceremony runs immediately: a separate keypair is generated, the Specter pseudonym and sigil are revealed with their own animation, and a brief explanation of the dual-identity model is provided.

### 40. Phase 4 — Network Bootstrap

The application begins connecting to the peer network. A visualization shows the user's node reaching out — tendrils of light extending toward the edges of the screen, connecting to discovered peers. Each successful connection is celebrated with a small visual event (the tendril connecting, the distant peer's node appearing). The visualization is both functional (it shows the real bootstrap process) and emotional (it creates the sensation of joining a living network).

If the user has an invitation, this is the phase where the invitation string is entered. The inviter's node appears prominently among the first connections, and the bootstrap visualization centers on the inviter's neighborhood — the user arrives in a familiar social context rather than a random network location.

During bootstrap, the application downloads cached content from connected peers, populating the local Wave cache. A progress indicator shows the sync process, but the visualization keeps the user engaged while they wait.

### 41. Phase 5 — Guided Exploration

The Pulse Map loads with the user's node at the center. A tutorial overlay provides contextual guidance: arrows and brief labels identify nodes ("These are other people on the network"), edges ("These lines are connections between people"), and Wave ripples ("When someone posts, you see it ripple through the map").

The tutorial prompts the user to perform three actions: pan the map (drag gesture), zoom in on a nearby node (pinch or scroll gesture), and tap a node to read its Waves. These three actions teach the fundamental navigation mechanics while giving the user their first taste of content discovery on the Pulse Map.

For Hybrid and Fortress users, an additional tutorial step reveals the Anonymous Layer: the layer slider is introduced, and the user is prompted to increase Anonymous Layer visibility, revealing the ghostly Specter nodes drifting beneath the Surface Layer. A label explains: "The anonymous world. You'll explore this more later."

### 42. Phase 6 — First Action

The onboarding concludes with a prompt to take a first action: publish a Wave, form a connection, or explore the map freely. The suggested Wave ("Hello, MURMUR") is pre-filled but editable. Tapping "publish" triggers the PoW computation (with a brief animation: "Your message is being sealed...") and then broadcasts the Wave, showing the propagation ripple expanding from the user's node.

After the first action, the tutorial overlay dismisses and the user has full control of the interface. A small, dismissable hint system remains available for the next few sessions, providing contextual tooltips when the user encounters unfamiliar UI elements.

---

## Part VIII — Growth

### 43. The Core Growth Loop

MURMUR's growth depends on a self-reinforcing loop: a user joins, discovers value, experiences a moment of social surplus (something they want to share with someone not yet on the network), invites that person, and the cycle repeats. Every design decision in MURMUR is evaluated against this loop. The target virality coefficient — the average number of successful invitations generated per user — must exceed 1.0 for the network to grow organically.

The loop's velocity depends on time to value — how quickly a new user reaches a social surplus moment. MURMUR targets under 5 minutes. The onboarding sequence is designed to deliver value at each phase: the identity ceremony creates ownership, the Pulse Map creates wonder, the first Wave creates presence, and the Anonymous Layer reveal creates curiosity. These experiences are intrinsic — they do not require the user to already know people on the network — which is critical during the early growth phase when most new users arrive to a sparse network.

### 44. Social Surplus Triggers

The system is designed to generate shareable moments frequently. Phantom Gifts from anonymous Specters create stories ("Someone anonymous just sent me this glowing thing"). Specter Duels are spectacles that generate curiosity in observers. Masked Events create intense post-event feelings ("I just had this incredible conversation with people I'll never identify"). The Pulse Map's visual beauty generates "what is this?" reactions when screenshotted or screen-recorded. Specter Marks create mystery and conversation. Each of these moments is a potential invitation trigger.

### 45. Invitation Mechanics

An invitation is a compact data structure containing the inviter's Peer ID, public key fingerprint, and optional welcome message, encoded as a URL-safe Base64 string with the `murmur://invite/` prefix. The total length is approximately 100–150 characters — short enough for a text message. The invitation functions as a bootstrap bypass: the new user's first connection is to the inviter, placing them in a familiar social neighborhood rather than a random network location.

Invitation generation is frictionless (two taps), unlimited, and available to every user. Invitation acceptance is integrated into onboarding, replacing the "find your first connection" step with a pre-solved version. The benefit is mutual: the invitee gets a warm start, the inviter gets a new connection that strengthens their mesh position. QR code encoding supports rapid in-person invitations.

### 46. Growth Phases

**Seeding (0–1,000 nodes).** The founding community establishes the initial network through personal outreach. Target communities include privacy enthusiasts, decentralization advocates, and creative technologists. The onboarding experience is tuned for sparse-network conditions, framing sparsity as pioneering rather than emptiness.

**Expansion (1,000–50,000 nodes).** The core growth loop activates as the network reaches sufficient density for organic social dynamics. Content critical mass (estimated at 5,000 active publishers) ensures that the network feels continuously alive. Communities form as visible clusters on the Pulse Map. Geographic and linguistic diversity becomes a priority.

**Sustaining (50,000+ nodes).** The network becomes self-sustaining. The growth strategy shifts from active promotion to defensive maintenance — ensuring performance scales with the user base, preventing community fragmentation, and maintaining novelty through ongoing feature depth. The Resonance progression system and evolving anonymous mechanics provide long-term retention.

### 47. Growth Without Telemetry

MURMUR collects no analytics. Growth is estimated through protocol-observable metrics: DHT peer count estimates, gossip volume trends, Connection Declaration frequency, and Beacon Wave frequency. These metrics are computed locally by any interested node. Community members voluntarily share network health observations as Waves, creating a decentralized, community-driven growth assessment practice.

---

## Part IX — Security Model

### 48. Threat Model

MURMUR's security model addresses four threat categories.

**Passive network observer.** An entity (ISP, government, or network administrator) that can observe network traffic between nodes but cannot modify it. The threat is metadata analysis — determining who communicates with whom, when, and how much. Mitigation: all libp2p connections are encrypted (Noise or TLS 1.3), preventing content inspection. Anonymous Layer traffic is routed through Shroud circuits, preventing the observer from linking a Specter's traffic to a specific node's IP address.

**Active network attacker.** An entity that can inject, modify, or block network traffic. The threat is message forgery, censorship, or targeted disruption. Mitigation: all messages are cryptographically signed, preventing forgery. GossipSub's mesh redundancy ensures that blocking a single path does not prevent message delivery. The DHT's redundant storage ensures that content remains available even if some peers are blocked.

**Malicious peer.** A node on the network that behaves adversarially — forwarding false messages, selectively dropping messages, or attempting to deanonymize Specters. Mitigation: message signatures prevent the malicious peer from forging content. GossipSub's peer scoring system penalizes peers that deliver invalid messages, eventually disconnecting them. The Shroud Network's three-hop circuit structure ensures that a single compromised relay cannot link a Specter to an IP address.

**Compromised device.** The user's own device is compromised (malware, physical seizure, or forensic analysis). Mitigation: the encrypted keystore protects keys at rest. The Specter keystore is separate from the main keystore, preventing compromise of one from exposing the other. Wave TTL ensures that cached content expires, limiting the forensic value of device seizure. Beyond these measures, a compromised device is fundamentally outside the system's ability to protect — if the attacker controls the hardware, they can observe everything the user does.

### 49. Cryptographic Primitives

MURMUR uses the following cryptographic primitives.

**Ed25519** for Surface Layer digital signatures (identity declarations, Wave signatures, connection declarations). Ed25519 provides 128-bit security with fast signing and verification, small key and signature sizes, and deterministic signatures (no random number generation required during signing, eliminating a class of implementation bugs).

**Curve25519 (X25519)** for Anonymous Layer key exchange and Shroud circuit handshakes. X25519 Diffie-Hellman key exchanges produce shared secrets used to derive symmetric encryption keys for Shroud circuit legs.

**ChaCha20-Poly1305** for symmetric encryption of Shroud circuit traffic and keystore encryption. ChaCha20-Poly1305 provides authenticated encryption with associated data (AEAD), ensuring both confidentiality and integrity.

**Argon2id** for key derivation from user passphrases. Argon2id is a memory-hard key derivation function resistant to GPU and ASIC brute-force attacks, protecting the keystore even if the encrypted keystore file is extracted from the device.

**SHA-256** for message hashing, Wave ID generation, Proof of Work, and sigil derivation. SHA-256 is the universal hash function throughout the system.

**HMAC-SHA-256** for keyed message authentication in Shroud circuit control messages and Phantom Council internal communications.

### 50. Cross-Layer Correlation Resistance

The most sensitive security property in MURMUR is cross-layer unlinkability — the guarantee that an observer cannot determine which main identity controls a given Specter identity. This property is maintained through several mechanisms.

The Specter keypair is cryptographically independent of the main keypair. There is no mathematical relationship between them. An observer who knows both keys cannot determine that they belong to the same person through cryptographic analysis.

Specter traffic is routed through the Shroud Network on separate circuits from main identity traffic. An observer monitoring a user's network connection sees two classes of traffic: direct gossip traffic (associated with the main identity) and Shroud circuit traffic (associated with the Specter, but indistinguishable from relay traffic being forwarded on behalf of other users). The observer cannot determine whether Shroud traffic originating from a node is the node's own Specter traffic or traffic being relayed for another user.

Behavioral correlation is the primary residual risk. If a user's main identity and Specter identity are consistently active at the same times, inactive at the same times, and interested in the same topics, a sophisticated observer might infer a link. The system does not attempt to mitigate behavioral correlation — doing so would require the system to modify user behavior, which is outside the scope of a technical architecture. Users who require strong cross-layer unlinkability are advised to deliberately vary their activity patterns.

Timing correlation is mitigated by traffic padding. The Shroud circuit sends cover traffic (encrypted random data) at a constant rate, masking the timing of real messages within the circuit's constant-rate stream. This prevents an observer from correlating the timing of a Specter's published Waves with traffic bursts from a specific node.

---

## Part X — Protocol Summary

### 51. Message Types

The protocol defines the following message types, all propagated via GossipSub.

**Identity Declaration.** Published on `murmur/identity/v1`. Contains the user's public key, display name, sigil parameters, timestamp, and signature. Declares the user's existence on the network.

**Connection Declaration.** Published on `murmur/identity/v1`. Contains the declaring user's Peer ID, the target user's Peer ID, a timestamp, and a signature. Declares one half of a mutual connection.

**Connection Revocation.** Published on `murmur/identity/v1`. Contains the revoking user's Peer ID, the target user's Peer ID, a timestamp, and a signature. Revokes a previously declared connection.

**Wave.** Published on `murmur/waves/v1`. Contains the Wave ID, author public key, content body, timestamp, optional parent ID, optional reference ID, TTL, PoW nonce, and signature. The atomic content unit.

**Specter Declaration.** Published on `murmur/anonymous/v1`. Contains the Specter's public key, pseudonym, sigil parameters, timestamp, and signature. Declares a Specter identity's existence. Routed through the Shroud Network.

**Specter Wave.** Published on `murmur/anonymous/v1`. Identical in structure to a standard Wave but signed with a Specter key and injected via a Shroud circuit exit node.

**Phantom Gift.** Published on `murmur/anonymous/v1`. Contains the gift type, the recipient node's Peer ID, the sender Specter's public key, a timestamp, and a Specter signature. Triggers a visual effect on the recipient's Pulse Map node.

**Specter Mark.** Published on `murmur/anonymous/v1`. Contains the mark category, the target node's Peer ID, the marking Specter's public key, a timestamp, and a Specter signature. Places a persistent visual annotation on the target's node.

**Duel Challenge.** Published on `murmur/anonymous/v1`. Contains the topic, format, challenged Specter's pseudonym (or "open"), the challenger Specter's public key, a timestamp, and a Specter signature.

**Duel Acceptance.** Published on `murmur/anonymous/v1`. Contains the challenge reference, the accepting Specter's public key, a timestamp, and a Specter signature.

**Duel Statement.** Published on `murmur/anonymous/v1`. Contains the duel reference, the statement content, the authoring Specter's public key, round number, timestamp, and a Specter signature.

**Duel Vote.** Published on `murmur/anonymous/v1`. Contains the duel reference, the voted-for Specter's public key, the voter's public key (main or Specter), a timestamp, and a signature.

**Event Announcement.** Published on `murmur/beacon/v1`. Contains the event topic, duration, participant cap, the hosting Specter's public key, start time, and a Specter signature.

**Event Join Request.** Sent directly to the hosting Specter's node via Shroud circuit. Contains the requesting identity's nonce (a single-use anonymous identifier, not linked to any persistent identity).

**Beacon.** Published on `murmur/beacon/v1`. Contains the beacon type (Shroud Node announcement, Event announcement, Council announcement), relevant parameters, the broadcasting entity's public key, a timestamp, and a signature.

**Council Announcement.** Published on `murmur/beacon/v1`. Contains the council's public key, the number of members (but not their identities), a formation timestamp, and a council signature.

**Council Action.** Published on `murmur/anonymous/v1`. Contains the action type (collective Mark, collective Gift, position statement), the action parameters, a timestamp, and a council signature.

### 52. Topic Architecture

The four GossipSub topics are designed to separate traffic by function and privacy level.

`murmur/identity/v1` carries identity and connection metadata. All nodes subscribe. Traffic volume scales linearly with network size (each new user generates one Identity Declaration and a handful of Connection Declarations).

`murmur/waves/v1` carries standard Wave content. All nodes subscribe. Traffic volume scales with both network size and per-user publication rate. This is the highest-volume topic and the primary bandwidth consumer.

`murmur/anonymous/v1` carries all Anonymous Layer content. Hybrid and Fortress nodes subscribe. Open-mode nodes may optionally subscribe to observe anonymous effects (Phantom Gifts, Specter Marks) that target their neighborhood, or may receive these effects via gossip from Hybrid-mode peers who forward relevant messages.

`murmur/beacon/v1` carries high-priority, network-wide broadcasts. All nodes subscribe. Traffic volume is low (Shroud Node announcements and event announcements are relatively rare) but messages on this topic are propagated with highest priority.

### 53. Protocol Versioning

All topic names include a version suffix (`/v1`). Future protocol versions will use new topic names (`/v2`), allowing gradual migration: nodes running the new version subscribe to both old and new topics during the transition period, and nodes running the old version continue to function on the old topics until they upgrade.

Breaking protocol changes (new message formats, new cryptographic primitives, new gossip parameters) are coordinated through community consensus. The absence of a central authority means that protocol upgrades require voluntary adoption by a supermajority of nodes — a high bar that ensures stability but limits the speed of protocol evolution.

---

## Part XI — Open Questions and Future Work

### 54. Scalability Ceiling

The current architecture is estimated to support approximately 100,000 active nodes before gossip bandwidth and layout computation become performance-limiting. Beyond this threshold, the network will require gossip topic sharding (partitioning high-traffic topics by geographic region or interest), hierarchical Pulse Map rendering (delegating layout of distant regions to representative nodes), and adaptive content routing (prioritizing gossip delivery of content from the user's social neighborhood over content from distant regions).

These scaling mechanisms are identified but not specified. Their design depends on empirical data from a running network at scale, which does not yet exist.

### 55. Content Moderation

MURMUR has no content moderation infrastructure. There are no moderators, no report mechanism, no content policy, and no takedown capability. Content is signed by its author and propagated by the gossip mesh; once published, it cannot be unpublished.

This is a deliberate design choice, consistent with the principles of self-sovereignty and decentralization. It is also a significant risk. Harmful content (harassment, illegal material, coordinated abuse) will appear on the network, and the system has no mechanism to address it except social pressure (users can sever connections with abusive nodes, reducing their gossip reach) and Phantom Council governance (councils can collectively Mark problematic nodes, signaling community disapproval).

Whether these mechanisms are sufficient to maintain a healthy social environment is an open question. Future work may explore reputation-weighted gossip (where messages from nodes with many severed connections propagate less effectively), client-side content filtering (where individual users can set content filters that hide Waves matching specified patterns), and community-defined norms enforcement through the Phantom Council system.

### 56. Economic Sustainability

MURMUR has no revenue model. Shroud Nodes are volunteer-operated. Bootstrap nodes are community-maintained. Development is contributed by volunteers. This model works for small, ideologically motivated communities but may not scale to a large, general-purpose social network.

Future work may explore incentive mechanisms for Shroud Node operators (reputation-based priority, Resonance rewards for relay contribution), donation-based sustainability models, and non-profit organizational structures to coordinate development and infrastructure maintenance.

### 57. Mobile Platform Constraints

Mobile operating systems impose constraints on background network activity: iOS and Android aggressively suspend background processes, limit background network connections, and restrict background CPU usage. These constraints affect MURMUR's ability to maintain persistent gossip mesh connections, relay traffic for other nodes, and compute Proof of Work for Wave publication.

Mitigation strategies include push-notification-based wake-up (using platform push notification services to wake the app when relevant gossip messages arrive, though this introduces a centralization dependency), lightweight background mode (maintaining a minimal gossip connection with reduced subscription breadth), and delegation to always-on companion nodes (a mobile user's desktop node maintains full gossip participation and relays relevant content to the mobile device when it wakes).

These mitigations involve trade-offs between decentralization purity and practical usability. Their design and implementation are deferred to platform-specific implementation specifications.

### 58. Accessibility

The Pulse Map's spatial-visual paradigm presents accessibility challenges for users with visual impairments. A screen-reader-compatible alternative interface — presenting network topology, Waves, and anonymous effects as structured text and audio cues — is necessary for accessibility compliance and ethical inclusivity.

The design of this alternative interface is identified as critical future work. It must convey the same social information as the Pulse Map (node identity, connection structure, Wave content, activity level, anonymous effects) through non-visual channels without losing the sense of spatial-social navigation that defines the MURMUR experience.

---

## Appendix A — Glossary

**Amplification.** The act of re-broadcasting another user's Wave, extending its reach to the amplifier's neighborhood.

**Anonymous Layer.** The parallel social layer on which Specter identities operate, routed through the Shroud Network and visually rendered as a ghostly overlay on the Pulse Map.

**Beacon Wave.** A high-priority broadcast on the `murmur/beacon/v1` topic, used for Shroud Node announcements, Masked Event invitations, and Council announcements.

**Connection Declaration.** A signed message declaring a unilateral connection from one user to another. Connections are bilateral — they require reciprocal declarations.

**Fortress Mode.** Privacy mode where the user participates exclusively on the Anonymous Layer with no Surface Layer presence.

**GossipSub.** The pubsub protocol used for message propagation across the MURMUR network.

**Hybrid Mode.** Privacy mode where the user participates on both the Surface Layer (under their main identity) and the Anonymous Layer (under a Specter identity).

**Identity Declaration.** A signed message publishing a user's public key, display name, and sigil to the network.

**Kademlia DHT.** The distributed hash table used for peer discovery and content routing.

**Masked Event.** A temporary social space where all participants are completely anonymous — no names, no sigils, no Resonance indicators.

**Open Mode.** Privacy mode where the user participates only on the Surface Layer.

**Phantom Council.** A persistent, secret deliberative body of 5–13 high-Resonance Specters who collaborate on anonymous governance and community stewardship.

**Phantom Gift.** A visual effect applied to a Surface Layer node by an anonymous Specter, visible to all users.

**Proof of Work (PoW).** A computational puzzle that must be solved before a Wave can be published, serving as rate limiting and Sybil resistance.

**Pulse Map.** The primary visual interface — a real-time, force-directed graph rendering the network's social topology.

**Resonance.** The Specter reputation metric, reflecting the quality and consistency of anonymous participation.

**Shroud Circuit.** A three-hop onion-routed relay chain connecting a user to the gossip mesh anonymously.

**Shroud Network.** The collective infrastructure of volunteer Shroud Nodes providing anonymous relay services.

**Shroud Node.** A volunteer-operated node that relays Shroud circuit traffic.

**Sigil.** A deterministic visual pattern generated from a public key hash, serving as a visual fingerprint for identity recognition.

**Specter.** A pseudonymous identity on the Anonymous Layer, defined by a Curve25519 keypair.

**Specter Duel.** A structured public debate between two Specters, judged by audience vote.

**Specter Mark.** A persistent anonymous annotation placed on a Surface Layer node by a high-Resonance Specter.

**Surface Layer.** The primary social layer where users participate under their chosen display name and sigil.

**Wave.** The atomic content unit — a signed, timestamped, ephemeral text message propagated via gossip.

---

## Appendix B — Design Constraints Summary

This appendix collects the key numerical constraints referenced throughout the document for quick reference.

Display name length: 1–32 Unicode characters. Wave content maximum: 2,048 bytes UTF-8. Wave TTL default: 168 hours (7 days). Wave TTL maximum: 720 hours (30 days). Target mesh size: 6–12 direct peers. Heartbeat interval: 30 seconds. Heartbeat failure threshold: 3 consecutive failures. Deduplication cache duration: 24 hours. Cache storage budget: approximately 100 MB (user-tunable). Bandwidth budget: approximately 50 KB/s sustained, 200 KB/s peak. Proof of Work target time: 2–5 seconds on modern consumer CPU. Shroud circuit length: 3 hops. Shroud circuit rotation interval: 10 minutes. Phantom Gift duration: 24 hours. Specter Resonance milestones: Shade at 25, Wraith at 50, Shade-Wraith at 75, Phantom at 100, Council-Eligible at 200. Masked Event duration: 30 minutes to 4 hours. Masked Event participant cap: 5 to 50. Phantom Council size: 5 to 13 members. Specter pseudonym wordlist: 4,096 adjectives × 4,096 nouns. Invitation string length: approximately 100–150 characters. Target time to value: under 5 minutes. Target frame rate: 60 FPS (minimum 30 FPS). Estimated scalability ceiling: approximately 100,000 active nodes.

---

*This is a living document. It describes MURMUR as it is designed, not as it is implemented. Implementation may reveal constraints, trade-offs, and opportunities that require revision. The document will evolve as the system does.*