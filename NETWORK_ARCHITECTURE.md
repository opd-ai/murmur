# Network Architecture

**Category:** Infrastructure — Networking Foundation
**Version:** 0.4
**Status:** Draft

---

## Overview

MURMUR is a fully decentralized, peer-to-peer social network with no servers, no central coordination points, and no infrastructure owned or operated by any single entity. Every participant runs the same software, contributes the same resources, and has the same capabilities. The network exists only because its participants exist — when the last node goes offline, the network ceases to be.

This document describes the networking layer that makes MURMUR possible: how nodes discover each other, how they establish connections, how they exchange data, how the network topology is managed, and how the dual-layer architecture (Surface Layer and Anonymous Layer) is implemented at the protocol level. The networking layer is built on libp2p, an open-source modular networking stack originally developed for IPFS and now used by a wide range of decentralized systems.

The networking layer has three primary responsibilities. First, it must enable any two MURMUR nodes to communicate, regardless of their network environment (home NAT, mobile carrier, corporate firewall, VPN). Second, it must propagate social data (Waves, identity announcements, connection declarations, anonymous mechanics) efficiently across the network without centralized routing. Third, it must support the Anonymous Layer's privacy guarantees by implementing traffic routing through Shroud Nodes, gossip separation between layers, and bridge-mediated cross-layer communication.

---

## libp2p Foundation

### Why libp2p

MURMUR uses libp2p as its networking foundation because libp2p provides a transport-agnostic, modular, peer-to-peer networking stack that handles the hardest problems of decentralized networking: peer identity, peer discovery, connection establishment across NATs, multiplexed streams, and protocol negotiation. Building on libp2p allows MURMUR to focus on its social and privacy-specific protocols rather than reinventing low-level networking primitives.

libp2p also provides a multi-transport architecture that is critical for MURMUR's reach. A single MURMUR node can communicate over TCP, QUIC, WebSocket, WebTransport, or WebRTC depending on the network environment. This flexibility ensures that MURMUR can run on desktop applications (using TCP or QUIC), in web browsers (using WebSocket, WebTransport, or WebRTC), and on mobile devices (using QUIC or WebSocket), all within the same network.

### Peer Identity

Every MURMUR node has a libp2p Peer ID derived from the node's main identity Ed25519 public key. The Peer ID is the node's network-layer address — it identifies the node to other peers regardless of transport or IP address. The Peer ID is a multihash of the public key, formatted as a base58-encoded string.

For Fortress-mode nodes (which have no Surface Layer identity), the Peer ID is derived from a dedicated transport keypair that is distinct from both the main identity key (which does not exist for Fortress users) and the Specter key. The transport keypair is used only for libp2p connection establishment and is not linked to the Specter identity at the protocol level. The Shroud Network provides additional anonymization between the transport keypair and the Specter identity, as described in the Anonymous Layer section.

### Protocol Multiplexing

MURMUR uses libp2p's protocol multiplexing to run multiple application protocols over a single peer connection. Each protocol is identified by a string ID and version number. The protocols registered by a MURMUR node are as follows.

The Surface Layer gossip protocol, identified as `/murmur/surface/1.0`, handles propagation of Surface Layer data: Waves, identity announcements, connection declarations, and cross-layer bridge injections.

The Anonymous Layer gossip protocol, identified as `/murmur/anonymous/1.0`, handles propagation of Anonymous Layer data: Specter Waves, Beacon Waves, anonymous mechanic data (mini-games, events, councils, gifts, marks), and Specter identity announcements.

The Shroud relay protocol, identified as `/murmur/shroud/1.0`, handles encrypted relay traffic for Shroud Network circuits. This protocol carries onion-encrypted blobs between Shroud Nodes and between Shroud Nodes and endpoint clients.

The Whisper Chain relay protocol, identified as `/murmur/whisper/1.0`, handles encrypted relay traffic for Whisper Chain messages between Specters.

The peer exchange protocol, identified as `/murmur/peerex/1.0`, handles peer discovery information exchange — nodes share their known peer lists with newly connected peers.

The sync protocol, identified as `/murmur/sync/1.0`, handles historical data synchronization for nodes that have been offline and need to catch up on missed Waves and events within the 30-day content window.

All protocols are negotiated using libp2p's multistream-select protocol during connection establishment. A node advertises the protocols it supports, and the connecting peer selects the protocols it wishes to use. Fortress-mode nodes do not advertise or accept the Surface Layer gossip protocol. Open-mode nodes do not advertise or accept the Anonymous Layer gossip protocol, the Shroud relay protocol, or the Whisper Chain relay protocol.

---

## Transport Layer

### Supported Transports

MURMUR supports four transport protocols, selected automatically based on the node's runtime environment.

**QUIC.** The preferred transport for native desktop and mobile applications. QUIC provides encrypted, multiplexed, connection-oriented communication over UDP. It offers lower latency than TCP (especially for connection establishment, which completes in a single round trip), built-in encryption via TLS 1.3 (eliminating the need for a separate encryption layer), and native multiplexing (eliminating head-of-line blocking that affects TCP-based multiplexing). MURMUR uses QUIC as the default transport whenever UDP connectivity is available.

**TCP.** The fallback transport for environments where UDP is blocked or unreliable. TCP connections are encrypted using the Noise protocol framework (specifically the Noise XX handshake pattern with Ed25519 keys), which provides mutual authentication and forward secrecy. TCP connections are multiplexed using yamux (Yet Another Multiplexer), a lightweight stream multiplexer that provides multiple logical streams over a single TCP connection.

**WebSocket.** The transport for web browser clients. WebSocket connections are established over TCP (typically on port 443 with TLS) and provide a bidirectional communication channel compatible with browser security policies. WebSocket connections use the same Noise encryption and yamux multiplexing as plain TCP connections, layered over the WebSocket framing.

**WebRTC.** An alternative transport for browser-to-browser communication that does not require a relay server. WebRTC connections are established using the libp2p WebRTC transport, which uses the browser's built-in WebRTC stack for NAT traversal (ICE/STUN/TURN) and encryption (DTLS). WebRTC is used when both peers are browser-based and direct connectivity is possible.

### Transport Selection Logic

When a node initiates a connection to a peer, the transport selection follows a priority order. If the node is a native application, it attempts QUIC first. If QUIC fails (UDP blocked), it falls back to TCP. If the node is a browser client, it attempts WebTransport first (a QUIC-based transport accessible from browsers, where supported). If WebTransport is unavailable, it attempts WebSocket. If the target peer is also a browser, it may attempt WebRTC for a direct connection.

Transport selection is automatic and invisible to the user. The Pulse Map does not indicate which transport is in use for any given connection.

### Connection Encryption

All MURMUR connections are encrypted regardless of transport. QUIC connections use TLS 1.3 with the node's Ed25519 identity key. TCP and WebSocket connections use the Noise XX handshake with Ed25519 keys. WebRTC connections use DTLS with keys bound to the node's libp2p identity. In all cases, the encryption provides mutual authentication (both peers verify each other's Peer ID), confidentiality (all data is encrypted), integrity (all data is authenticated), and forward secrecy (session keys are ephemeral and not derivable from long-term keys).

### Connection Limits

Each node maintains a maximum of 200 simultaneous peer connections. This limit is a practical constraint to manage memory, CPU, and bandwidth consumption. The connection manager (described in the Topology Management section) selects which peers to maintain connections with based on social proximity, network utility, and connection quality.

---

## Peer Discovery

### Bootstrap Nodes

When a MURMUR node starts for the first time, it has no knowledge of any peers. It must discover at least one existing peer to join the network. MURMUR uses a set of bootstrap nodes — well-known peers whose network addresses are hardcoded into the application — as initial entry points.

Bootstrap nodes are operated by community volunteers and are listed in the application's configuration. The default configuration includes 8–12 bootstrap nodes distributed across different geographic regions and network providers. Bootstrap nodes are ordinary MURMUR nodes with no special privileges — they run the same software as every other node. Their only distinction is that their addresses are known in advance, allowing new nodes to find them without prior network knowledge.

A new node connects to 2–3 randomly selected bootstrap nodes and uses them as initial peers for peer discovery. Once the node has discovered additional peers through the peer exchange protocol, its dependence on bootstrap nodes diminishes. Long-running nodes rarely communicate with bootstrap nodes at all.

Bootstrap node addresses are updated with each application release. If all hardcoded bootstrap nodes are offline, the application prompts the user to manually enter a peer address (obtainable from community forums, social media, or direct communication with an existing MURMUR user). This manual entry mechanism ensures the network can recover even if all bootstrap nodes fail simultaneously.

### Distributed Hash Table

MURMUR uses the Kademlia distributed hash table (DHT) implementation provided by libp2p for peer discovery and peer routing. The DHT allows any node to find the network address of any other node given its Peer ID, without requiring a central directory.

Each node maintains a Kademlia routing table — a set of buckets containing known peers, organized by XOR distance from the node's own Peer ID. The routing table is populated through the peer exchange protocol and through DHT queries. When a node needs to find a specific peer (for example, to establish a direct connection to a social contact), it performs an iterative DHT lookup: it queries the peers in its routing table that are closest to the target Peer ID, then queries the peers returned by those queries, and so on, converging on the target in O(log n) steps where n is the network size.

The DHT is also used for content routing — finding nodes that hold specific data. MURMUR uses DHT provider records to advertise that a node holds Waves from a specific publisher (identified by the publisher's public key hash). Nodes that want to follow a specific publisher can query the DHT for provider records and connect to peers that hold that publisher's content.

### Peer Exchange Protocol

The peer exchange protocol (`/murmur/peerex/1.0`) is a simple protocol where connected peers periodically share their known peer lists. Every 5 minutes, each node sends a random sample of 20 peers from its routing table to each connected peer. The sample includes each peer's Peer ID, known network addresses, observed latency, and supported protocols.

Receiving nodes merge the sample into their own routing table, attempting connections to newly discovered peers if their connection count is below the target. This protocol provides continuous, passive peer discovery that supplements the DHT — even if the DHT is slow or partially partitioned, peer exchange ensures that well-connected nodes rapidly learn about new peers.

### mDNS Local Discovery

For nodes on the same local network (such as devices on the same Wi-Fi network), MURMUR uses multicast DNS (mDNS) to discover local peers without any external network connectivity. mDNS discovery broadcasts the node's presence on the local network and listens for broadcasts from other MURMUR nodes. This enables fully offline local MURMUR networks — two or more devices on the same LAN can form a MURMUR network without any internet connectivity.

mDNS discovery is enabled by default and can be disabled in settings for users who do not wish to advertise their MURMUR usage on the local network.

---

## NAT Traversal

### The NAT Problem

The majority of consumer internet connections place devices behind Network Address Translation (NAT) — a router that maps private internal IP addresses to a single public IP address. Devices behind NAT can initiate outbound connections, but cannot receive inbound connections without special configuration. Since MURMUR is a peer-to-peer network where every node must be reachable by other nodes, NAT traversal is essential.

### Hole Punching

MURMUR's primary NAT traversal strategy is hole punching, facilitated by libp2p's AutoNAT and Circuit Relay v2 protocols. The process works as follows.

First, the node determines its NAT status using the AutoNAT protocol. The node asks several connected peers to attempt inbound connections to it. If the peers succeed, the node is publicly reachable (no NAT, or NAT with port forwarding). If the peers fail, the node is behind NAT.

Second, if the node is behind NAT, it registers itself with one or more relay nodes — publicly reachable peers that agree to relay connections on its behalf. The relay node's address is included in the NATed node's advertised addresses as a relayed multiaddress (for example, `/ip4/203.0.113.1/tcp/4001/p2p/QmRelay/p2p-circuit/p2p/QmNATedNode`).

Third, when another peer wants to connect to the NATed node, it first connects to the relay and requests a relayed connection. The relay forwards the connection request to the NATed node. Once both peers are connected to the relay, they attempt a direct connection using hole punching — the DCUtR (Direct Connection Upgrade through Relay) protocol coordinates simultaneous outbound connections from both NATed peers to each other, punching through both NATs. If hole punching succeeds, the relay connection is upgraded to a direct connection and the relay is released. If hole punching fails, the relayed connection persists (at reduced performance due to the relay hop).

### Relay Nodes

Any publicly reachable MURMUR node can serve as a relay node. Relay service is enabled by default for nodes that the AutoNAT protocol confirms are publicly reachable. Relay nodes accept a limited number of relay reservations (default: 128 concurrent reservations) and impose bandwidth limits on relayed connections (default: 128 KB/s per relayed connection) to prevent resource exhaustion.

Relay service is a community resource — nodes that are publicly reachable contribute relay capacity to the network, enabling NATed nodes to participate. There is no explicit incentive for relay service beyond the general principle that the network functions only because participants contribute resources.

### TURN Fallback for WebRTC

For browser-based nodes using WebRTC, NAT traversal uses the browser's built-in ICE (Interactive Connectivity Establishment) framework, which combines STUN (Session Traversal Utilities for NAT) and TURN (Traversal Using Relays around NAT). MURMUR configures ICE with public STUN servers for address discovery and falls back to community-operated TURN servers when direct connectivity fails. TURN server addresses are distributed via the DHT as provider records with a well-known service key.

---

## Gossip Protocols

### GossipSub

MURMUR uses GossipSub v1.1 (the libp2p pubsub implementation) as its primary data propagation protocol. GossipSub is a topic-based publish-subscribe protocol optimized for decentralized networks. Nodes subscribe to topics and receive messages published on those topics by any node in the network. Message propagation is achieved through a mesh overlay: each node maintains a mesh of peers for each subscribed topic, and messages are forwarded through the mesh with bounded redundancy.

GossipSub provides several properties critical to MURMUR. It propagates messages to all subscribers within O(log n) hops. It is resilient to node churn (nodes joining and leaving). It is resistant to message censorship (an adversary must control a large fraction of mesh peers to suppress a message). And it supports message validation — nodes can validate messages before forwarding them, preventing propagation of invalid or spam messages.

### Topic Structure

MURMUR's gossip is organized into topics. Each topic is a named channel that nodes subscribe to and publish on. The topic structure is as follows.

**`murmur/surface/waves/1.0`** carries all Surface Layer Waves (type 0x01) and their associated metadata (replies, amplifications). This is the highest-traffic topic on the Surface Layer.

**`murmur/surface/identity/1.0`** carries Surface Layer identity announcements (new identity registrations, display name changes, profile updates) and connection declarations (new connections, connection revocations).

**`murmur/anonymous/waves/1.0`** carries all Anonymous Layer Waves (Specter Waves type 0x02, Masked Waves type 0x07) and their associated metadata. This is the highest-traffic topic on the Anonymous Layer.

**`murmur/anonymous/mechanics/1.0`** carries Anonymous Layer mechanic data: Phantom Gift declarations, Specter Mark placements, Mini-Game Event/Join/Action/Result Waves, Masked Event announcements and join records, and Phantom Council formation/application/vote records.

**`murmur/anonymous/beacons/1.0`** carries Beacon Waves (type 0x08) — high-effort, high-visibility broadcasts on the Anonymous Layer.

**`murmur/event/[event_id]/1.0`** is a per-event topic for Masked Event communication. These topics are ephemeral — they are created when the event starts and garbage collected 7 days after the event ends.

**`murmur/council/[council_id]/1.0`** is a per-council topic for Phantom Council private communication. These topics are persistent as long as the council is active. Messages on council topics are encrypted with the council's shared symmetric key.

### Message Validation

Every message published on a MURMUR gossip topic is validated by receiving nodes before being forwarded to mesh peers. Validation checks are topic-specific but share a common set of baseline checks.

The baseline validation checks are as follows. The message must contain a valid Ed25519 signature from the declared publisher's public key. The message's PoW stamp must meet the minimum difficulty threshold for its type (16 leading zero bits for standard Waves, 24 for Beacon Waves). The message's timestamp must be within a 5-minute window of the receiving node's local clock (to prevent replay attacks while accommodating clock drift). The message must not be a duplicate of a previously received message (checked against a message ID cache that stores IDs for the trailing 60 minutes).

Topic-specific validation adds further checks. Waves are validated for correct format and content size limits. Mini-game Waves are validated for correct event reference, action sequencing, and participant identity. Masked Event Waves are validated for membership in the event's registered participant set. Council Waves are validated for council membership (though content is encrypted and cannot be validated by non-members, the signature must belong to a known council member).

Messages that fail validation are dropped and not forwarded. The publishing peer is scored negatively in GossipSub's peer scoring system (described in the Spam Resistance section).

### Mesh Management

GossipSub maintains a mesh of peers for each topic. The target mesh degree (number of mesh peers per topic) is 6, with a lower bound of 4 and an upper bound of 12. When the mesh degree drops below 4, the node grafts additional peers from its pool of topic subscribers. When the mesh degree exceeds 12, the node prunes excess peers.

The mesh is maintained through periodic heartbeat messages (every 1 second). During each heartbeat, the node evaluates its mesh peers' health (based on message delivery performance, latency, and peer score) and replaces underperforming peers with better candidates from the subscriber pool.

### Message Propagation Latency

Under normal network conditions (nodes with typical residential internet connections, network size up to 100,000 nodes), a Wave published by any node reaches 99% of subscribed nodes within 3 seconds. This latency budget is the target for GossipSub configuration tuning. The primary factors affecting propagation latency are mesh degree (higher degree means more parallel forwarding paths), heartbeat interval (faster heartbeats detect mesh problems sooner), and message validation cost (slower validation increases per-hop latency).

---

## Topology Management

### Connection Manager

Each node runs a connection manager that determines which peers to maintain connections with, subject to the 200-connection maximum. The connection manager balances three competing goals: social utility (maintaining connections to peers whose content the user cares about), network utility (maintaining connections that contribute to a well-connected mesh topology), and resource efficiency (dropping connections that consume resources without providing value).

The connection manager classifies connections into four priority tiers.

**Tier 1: Social Connections.** Peers who are the user's declared social connections (Surface Layer connections or Specter connections). These are the highest priority and are never dropped unless the social connection itself is revoked. Maintaining direct network connections to social contacts ensures low-latency content delivery for the content the user cares about most.

**Tier 2: Mesh Obligations.** Peers who are mesh partners in the user's subscribed GossipSub topics. These connections are essential for gossip propagation. Dropping a mesh partner degrades message delivery until the mesh is repaired (which takes one heartbeat cycle).

**Tier 3: DHT Neighbors.** Peers who are in the user's Kademlia routing table, particularly those in sparse buckets (buckets with few entries). These connections maintain the DHT's routing structure and enable peer discovery for the broader network.

**Tier 4: Opportunistic.** Peers discovered through peer exchange or DHT queries that are not currently serving any specific role. These connections provide redundancy and are candidates for promotion to higher tiers if a mesh or DHT role becomes available.

When the connection count approaches the maximum, the connection manager drops Tier 4 connections first, then Tier 3 if necessary. Tier 1 and Tier 2 connections are never dropped by the connection manager (only by explicit user action or protocol-level events).

### Churn Handling

Node churn — peers going offline and coming online unpredictably — is a fundamental challenge of peer-to-peer networks. MURMUR handles churn through several mechanisms.

GossipSub's mesh management automatically repairs mesh damage caused by peer departures. When a mesh peer goes offline, the next heartbeat detects the failure and grafts a replacement peer. The mesh repair latency is at most 1 heartbeat cycle (1 second) plus the time to establish a new connection.

The Kademlia DHT handles churn through its bucket refresh protocol. Buckets are periodically refreshed by querying random IDs in the bucket's range, which discovers new peers to replace departed ones. Bucket refresh runs every 10 minutes.

The peer exchange protocol continuously introduces new peers to replace departed ones. The 5-minute exchange cycle ensures that a node's peer list is refreshed frequently enough to track network churn.

For social connections, the connection manager attempts reconnection to departed Tier 1 peers on an exponential backoff schedule: 5 seconds, 10 seconds, 20 seconds, 40 seconds, and so on up to a maximum interval of 10 minutes. Reconnection attempts continue indefinitely until the peer returns or the social connection is revoked.

### Network Partitioning

Network partitions — events where the network splits into two or more disconnected components — are rare in a well-connected peer-to-peer network but can occur due to widespread internet outages or coordinated node failures. MURMUR does not implement explicit partition detection or healing. Instead, it relies on the natural redundancy of the mesh topology (mesh degree 6 means that a partition requires the simultaneous failure of all mesh paths between two components) and on the reconnection behavior of the connection manager (which continuously attempts to reconnect to known peers, automatically healing the partition when connectivity is restored).

During a partition, nodes in each component continue to operate normally — Waves propagate within the component, social mechanics function, and the Pulse Map displays the component's topology. When the partition heals, the sync protocol reconciles missed messages between the formerly disconnected components.

---

## Data Synchronization

### The Sync Problem

When a node goes offline and later returns, it has missed Waves, identity updates, connection changes, and mechanic events that occurred during its absence. The sync protocol (`/murmur/sync/1.0`) enables the returning node to retrieve missed data from its peers.

### Sync Protocol

The sync protocol is a request-response protocol. The returning node sends a sync request to several connected peers, specifying the timestamp of the last message it received on each subscribed topic. Each peer responds with a batch of messages published after the requested timestamp, up to the 30-day content window limit.

The sync response is a stream of messages in chronological order, formatted identically to gossip messages. The returning node validates each synced message using the same validation logic as live gossip messages (signature verification, PoW check, timestamp check, duplicate detection). Valid messages are integrated into the node's local data store.

### Sync Limits

To prevent sync from overwhelming the returning node or the responding peers, several limits are imposed. A single sync request retrieves at most 1,000 messages per topic. If more messages were missed, the node issues follow-up sync requests with updated timestamps. Each node serves at most 5 concurrent sync requests from peers (additional requests are queued). Sync responses are rate-limited to 100 messages per second per peer to prevent bandwidth saturation.

### Selective Sync

A returning node does not need to sync all topics. It syncs only the topics it is subscribed to. A node subscribed only to the Surface Layer topics does not sync Anonymous Layer topics, and vice versa. Within each topic, the node can request selective sync — for example, syncing only Waves from specific publishers (identified by public key) rather than all Waves on the topic. Selective sync reduces bandwidth consumption for nodes with narrow social interests.

### Sync Consistency

Because MURMUR has no global state or total ordering of messages, sync does not guarantee that all nodes have an identical view of the network's history. Different nodes may have received different subsets of messages depending on their connectivity, subscription patterns, and sync peers. This eventual consistency model is acceptable for a social network — minor inconsistencies in message delivery do not compromise the user experience, and the 30-day content window ensures that all discrepancies resolve within a bounded time period.

---

## Anonymous Layer Networking

### Gossip Separation

The Surface Layer and Anonymous Layer use separate GossipSub topics, as described in the Topic Structure section. This separation ensures that gossip traffic from the two layers does not mix at the protocol level. A node subscribing only to Surface Layer topics never receives Anonymous Layer messages, and vice versa.

This separation is critical for privacy. Surface Layer gossip carries messages signed by main identity keys — a node that receives Surface Layer gossip can identify the publishers. Anonymous Layer gossip carries messages signed by Specter keys — a node that receives Anonymous Layer gossip can identify the Specter pseudonyms but not the underlying main identities. If the two gossip streams were mixed, traffic analysis could correlate message timing and content between layers, potentially linking Specters to main identities.

### Shroud Network Integration

Hybrid+ and Fortress-mode nodes route their Anonymous Layer gossip through Shroud Network circuits. The Shroud Network is an onion-routing overlay (described in the Specter Identity and Anonymous Layer specification) that anonymizes the network-layer origin of Anonymous Layer messages.

When a Hybrid+ or Fortress-mode node publishes a Specter Wave, the message is not published directly on the Anonymous Layer gossip topic from the node's own Peer ID. Instead, the message is encapsulated in an onion-encrypted blob and sent through a 3-hop Shroud circuit. The exit node of the circuit publishes the message on the Anonymous Layer gossip topic, using the exit node's Peer ID as the gossip origin. This prevents observers from correlating the message's gossip origin with the author's Peer ID.

The Shroud relay protocol (`/murmur/shroud/1.0`) carries onion-encrypted blobs between Shroud Nodes. Each blob is a fixed 4096 bytes (padded to prevent size-based traffic analysis). Blobs are forwarded with minimal latency — each relay node decrypts one onion layer and forwards the inner blob to the next hop within 100 milliseconds. The 3-hop circuit adds approximately 300 milliseconds of latency to Anonymous Layer message publication.

### Shroud Circuit Construction

A node constructs a Shroud circuit by selecting 3 Shroud Nodes from the available pool. Shroud Nodes are identified by their Specter public keys and advertised through Beacon Waves on the Anonymous Layer (with `beacon_type` set to `shroud_node_announce`). The selecting node performs a key exchange with each Shroud Node in sequence (using X25519 Diffie-Hellman) to establish shared secrets, then constructs the onion-encrypted blob by encrypting the message in 3 layers (innermost encrypted to the exit node, middle encrypted to the middle relay, outermost encrypted to the entry relay).

Circuit selection avoids choosing Shroud Nodes that are topologically close to each other or to the selecting node (based on Kademlia XOR distance in the DHT). This diversity requirement makes it harder for an adversary controlling a cluster of nearby nodes to capture all three hops of a circuit.

Circuits are rotated every 10 minutes. A new circuit is constructed with freshly selected Shroud Nodes and fresh key exchanges. The old circuit's shared secrets are securely erased. Circuit rotation limits the window during which a compromised Shroud Node can correlate traffic.

### Bridge Nodes

Bridge nodes are nodes running in Hybrid mode that participate in both Surface Layer and Anonymous Layer gossip. They serve as conduits for cross-layer data: Phantom Gifts targeting Surface Layer nodes, Specter Marks targeting Surface Layer nodes, and Echo propagation of Surface Layer Waves into the Anonymous Layer.

Bridge nodes subscribe to both sets of gossip topics. When they receive a cross-layer message on the Anonymous Layer (identified by metadata indicating a Surface Layer target), they re-publish the message on the appropriate Surface Layer topic. The bridge node does not modify the message — it simply moves it from one gossip domain to the other. The message retains its original Specter signature, so Surface Layer recipients can verify the anonymous origin.

Bridge functionality is automatic for Hybrid-mode nodes. There is no opt-in or configuration required. The presence of many bridge nodes (any Hybrid-mode user is a bridge) ensures robust cross-layer communication.

### Fortress Mode Networking

Fortress-mode nodes subscribe exclusively to Anonymous Layer gossip topics. They do not participate in Surface Layer gossip, the Surface Layer DHT namespace, or Surface Layer sync. Their Peer ID is derived from a transport keypair unrelated to any main identity. Their gossip publications are routed through Shroud circuits.

A Fortress-mode node's network presence is minimal: it appears as a Peer ID in the DHT and GossipSub mesh, participating in Anonymous Layer topics. It cannot be linked to any Surface Layer identity through protocol-level observation. The only metadata leaked by a Fortress-mode node is its IP address (visible to its direct peers), which is mitigated by the Shroud circuit routing — the node's direct peers see the node's IP, but cannot link it to any Specter identity because the node's Specter messages enter the gossip via a Shroud exit node rather than the node's own Peer ID.

For maximum anonymity, Fortress-mode users can run their node behind a VPN or Tor, further obscuring their IP address from direct peers. MURMUR does not require or configure this — it is an optional operational security measure.

---

## Bandwidth and Storage

### Bandwidth Consumption

MURMUR's bandwidth consumption depends on the node's subscription pattern and the network's activity level. The following estimates assume a network of 50,000 active nodes with an average publication rate of 1 Wave per user per hour.

A Surface Layer-only node (Open mode) subscribed to all Surface Layer topics receives approximately 50,000 Waves per hour (each approximately 2.5 KB including headers, signatures, and PoW stamps), consuming approximately 125 MB per hour or approximately 35 KB/s sustained. With GossipSub's mesh redundancy (messages received from multiple mesh peers), the actual bandwidth is approximately 1.5 times the unique message volume, resulting in approximately 50 KB/s sustained.

An Anonymous Layer node (Hybrid or Fortress mode) subscribed to all Anonymous Layer topics receives an additional volume of Anonymous Layer messages proportional to the Anonymous Layer's activity. Assuming 20% of users are active on the Anonymous Layer, this adds approximately 10,000 messages per hour, consuming approximately 15 KB/s sustained (with mesh redundancy).

Shroud relay traffic adds bandwidth for nodes serving as Shroud Nodes: each relayed blob is 4096 bytes, and a busy Shroud Node may relay up to 100 blobs per minute, adding approximately 7 KB/s.

These estimates assume full subscription to all topics. In practice, most nodes subscribe to a subset of topics (only topics relevant to their social connections), significantly reducing bandwidth consumption. Topic-based subscription filtering is the primary bandwidth management tool available to users.

### Storage Consumption

Each node stores the Waves, identity data, and mechanic events it has received, up to the 30-day content window. At the estimated message rates above, 30 days of Surface Layer data is approximately 90 GB for a fully-subscribing node. This is impractical for many devices, particularly mobile devices.

To manage storage, nodes employ a tiered retention strategy. Waves from the user's direct social connections (Tier 1) are retained for the full 30-day window. Waves from Tier 2 sources (connections-of-connections) are retained for 7 days. Waves from Tier 3 and Tier 4 sources are retained for 24 hours. This tiered retention reduces storage consumption by approximately 90%, bringing 30-day storage to approximately 9 GB for a typical user with 50 direct connections.

Users can configure retention aggressiveness in settings, trading storage consumption against content availability. A "minimal storage" mode retains only Tier 1 content for 7 days, consuming under 1 GB. A "full archive" mode retains all received content for 30 days, consuming the full estimated volume.

### Content Availability

Because MURMUR is a peer-to-peer network with no servers, content availability depends on at least one node retaining and serving the content. If all nodes that received a specific Wave go offline or garbage collect it before other nodes can sync it, the Wave is permanently lost.

In practice, content loss is rare for Waves from well-connected publishers — the Wave propagates to many nodes, and the probability of all of them losing it within the 30-day window is negligible. Content loss is more likely for Waves from poorly connected publishers (publishers with few connections, whose Waves reach few nodes) or during periods of high churn.

MURMUR does not attempt to guarantee content availability. It accepts the inherent fragility of decentralized storage as a trade-off for the absence of centralized infrastructure. The 30-day content window further bounds the problem — MURMUR is not a permanent archive, and users should not expect content to persist indefinitely.

---

## Spam Resistance at the Network Layer

### Proof of Work Validation

Every message on every MURMUR gossip topic carries a PoW stamp. Nodes validate the PoW stamp as part of message validation before forwarding. Messages with invalid or insufficient PoW are dropped. This ensures that every message in the network has consumed a minimum computational cost, making large-scale spam economically expensive.

The PoW difficulty thresholds are: 16 leading zero bits for standard Waves (approximately 0.5 seconds of computation on a modern CPU), 24 leading zero bits for Beacon Waves (approximately 30 seconds), and 16 leading zero bits for all mechanic-specific messages (mini-games, events, councils, gifts, marks, votes).

### GossipSub Peer Scoring

GossipSub v1.1 includes a peer scoring system that tracks the quality of each mesh peer's behavior. MURMUR configures the peer scoring parameters to penalize peers that publish invalid messages (messages that fail validation), peers that publish excessive messages (exceeding rate limits), peers that consistently deliver stale messages (messages with old timestamps), and peers that graft aggressively (attempting to force their way into meshes they have been pruned from).

A peer's score is a floating-point value that decays toward zero over time. Negative scores result from bad behavior (invalid messages, spam, staleness). Positive scores result from good behavior (delivering valid messages promptly, maintaining stable mesh participation). Peers with scores below a configurable threshold are disconnected and temporarily banned from reconnecting (ban duration: 10 minutes, escalating exponentially for repeat offenders).

### Rate Limiting

In addition to peer scoring, each node enforces per-peer rate limits on message delivery. A single peer is allowed to deliver at most 100 messages per minute across all topics. Messages exceeding this rate are dropped. The rate limit prevents a compromised or malicious peer from flooding the node with messages, even if the messages are individually valid.

Per-topic rate limits are also enforced: a single peer delivers at most 50 messages per minute on any single topic. These limits are generous enough to accommodate legitimate high-traffic scenarios (a popular publisher's Waves being forwarded through a mesh peer) while capping the maximum load from any single peer.

---

## Resilience and Failure Modes

### Single Node Failure

The failure of any single node has minimal impact on the network. The node's mesh peers detect the failure within one heartbeat cycle (1 second) and graft replacement peers. The node's DHT entries expire and are replaced by the DHT's refresh protocol. The node's social connections are temporarily unreachable from the departed node but remain fully functional in the network. When the node returns, the sync protocol restores its missed data.

### Bootstrap Node Failure

The failure of one or more bootstrap nodes affects only new nodes attempting to join the network. Existing nodes are unaffected (they are already connected to the mesh). New nodes fail over to other bootstrap nodes from the hardcoded list. If all bootstrap nodes are offline simultaneously, new nodes must join via manual peer entry.

### Mass Churn Events

If a large fraction of the network goes offline simultaneously (for example, due to a widespread internet outage affecting a geographic region), the remaining network continues operating with reduced capacity. GossipSub mesh repair runs continuously, adapting the mesh to the reduced peer population. When the offline nodes return, they sync missed data and rejoin the mesh.

The network's resilience to mass churn depends on the mesh degree and the node population. A mesh degree of 6 provides tolerance to approximately 50% simultaneous node loss before the mesh becomes disconnected. Higher mesh degrees improve resilience but increase bandwidth consumption.

### Adversarial Node Injection

An adversary who injects a large number of malicious nodes into the network can attempt to degrade network performance by publishing invalid messages, disrupting mesh topology, or selectively dropping messages. MURMUR's defenses against this attack are peer scoring (which detects and bans misbehaving nodes), PoW requirements (which make Sybil attacks expensive), and mesh diversity (which ensures that any single node is surrounded by peers from many different sources, making it unlikely that all of a node's mesh partners are adversarial).

The Eclipse attack — where an adversary surrounds a target node with adversarial peers, isolating it from the honest network — is mitigated by the connection manager's Tier diversity requirement. The connection manager ensures that the node's connections span multiple DHT regions (Kademlia buckets), making it difficult for an adversary to occupy all connection slots without controlling peers in many different DHT regions.

---

## Protocol Versioning and Upgrades

### Version Negotiation

All MURMUR protocols include a version number in their protocol ID string (e.g., `/murmur/surface/1.0`). When a node connects to a peer, libp2p's multistream-select protocol negotiates the protocol version. If both peers support the same version, that version is used. If one peer supports a newer version, the peers fall back to the highest version both support.

### Forward Compatibility

MURMUR message formats include a version byte and reserve space for extension fields. Messages with unrecognized extension fields are processed normally (unrecognized fields are ignored). This forward-compatible design allows new features to be added without breaking older nodes' ability to process messages.

### Breaking Changes

Protocol changes that are incompatible with previous versions (changes to message format, validation rules, or PoW parameters) require a new protocol version. When a breaking change is deployed, nodes running the new version advertise the new protocol ID. Nodes running the old version advertise the old protocol ID. The network temporarily operates with two protocol versions, and peers negotiate the highest compatible version for each connection.

A deprecation period of 90 days is observed for old protocol versions. After 90 days, the old version is removed from the codebase, and nodes that have not upgraded can no longer participate in the network. This deprecation period provides a migration window for users to update their software.

---

## Implementation Notes

### Platform-Specific Networking

The MURMUR reference client is implemented using the Rust libp2p implementation (rust-libp2p) for native desktop and mobile builds, and the JavaScript libp2p implementation (js-libp2p) for the web browser client. Both implementations support the same protocol IDs and message formats, ensuring interoperability between native and browser nodes.

Native nodes have access to all transports (QUIC, TCP, WebSocket). Browser nodes are limited to WebSocket, WebTransport, and WebRTC. Browser nodes cannot serve as Shroud Nodes (because browser environments do not support the long-running, high-availability operation required for relay service) but can use Shroud circuits built through native Shroud Nodes.

### Mobile Considerations

Mobile nodes face additional constraints: intermittent connectivity, aggressive operating system background process management, and battery limitations. MURMUR's mobile client adapts to these constraints by reducing GossipSub mesh participation when the app is backgrounded (the node demotes itself from a full mesh peer to a gossip-only peer that receives messages but does not forward them), by aggressively compacting local storage to respect mobile storage limits, and by batching sync requests to minimize connection establishment overhead when connectivity is restored after a period of disconnection.

When the mobile app is foregrounded, the node re-promotes itself to full mesh participation within one heartbeat cycle (1 second). The transition from backgrounded to foregrounded is seamless to the user — the Pulse Map updates within 2–3 seconds of the app being opened.

### Firewall and Corporate Network Considerations

Some users operate behind restrictive firewalls that block outbound connections on non-standard ports or that perform deep packet inspection. MURMUR accommodates these environments by supporting WebSocket connections on port 443 (which is typically allowed through firewalls) with TLS encryption (which prevents deep packet inspection from identifying MURMUR traffic). The WebSocket transport is available to both native and browser clients.

For environments where even WebSocket on port 443 is blocked, MURMUR supports domain-fronting as an optional transport configuration — the WebSocket connection is routed through a CDN domain that is not blocked by the firewall. Domain-fronting configuration is manual (the user provides the CDN endpoint) and is intended as a last-resort connectivity option.