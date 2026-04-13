# Security Model

**Category:** Infrastructure — Security and Threat Analysis
**Version:** 0.4
**Status:** Draft

---

## Overview

MURMUR is a decentralized social network that makes strong privacy claims. It promises that users can maintain anonymous identities unlinkable to their real-world selves, that anonymous communications cannot be traced to their authors, and that the network operates without any trusted central authority. These claims create a security surface that must be analyzed rigorously. A system that promises privacy and fails to deliver it is worse than a system that makes no privacy promises at all — it creates a false sense of safety that leads users to take risks they would not otherwise take.

This document describes MURMUR's security model: the cryptographic primitives the system relies on, the adversaries it defends against, the attacks it mitigates, the attacks it does not mitigate, and the known limitations and open risks that users should understand. The document is written for a technically informed audience and assumes familiarity with the system architecture described in other specification documents.

MURMUR's security posture can be summarized as follows. Against passive observers and moderate-resource adversaries, MURMUR provides strong identity privacy, communication confidentiality, and censorship resistance. Against state-level adversaries with the ability to observe large portions of internet traffic, MURMUR provides meaningful but degraded privacy guarantees. Against adversaries who compromise individual user devices, MURMUR provides no protection — a compromised endpoint is a compromised identity.

---

## Cryptographic Primitives

### Ed25519 Signatures

All identity and authentication in MURMUR is built on Ed25519, an elliptic curve digital signature algorithm operating on Curve25519. Ed25519 provides 128-bit security against classical attacks. Every identity in the system — main identities, Specter identities, Masked Event identities, and transport identities — is an Ed25519 keypair. Every message (Wave, mechanic event, protocol message) is signed with the author's Ed25519 private key and verified by recipients using the corresponding public key.

Ed25519 was chosen for its performance (fast signing and verification on all target platforms including mobile devices), its resistance to timing side-channel attacks (the algorithm's operations are constant-time), its small key and signature sizes (32-byte public keys, 64-byte signatures), and its wide availability in mature, audited cryptographic libraries.

MURMUR does not currently defend against quantum computing attacks. A sufficiently powerful quantum computer could break Ed25519 using Shor's algorithm. The migration path to post-quantum cryptography is discussed in the Known Limitations section.

### X25519 Key Exchange

All key exchanges in MURMUR (Shroud circuit establishment, Whisper Chain construction, Phantom Council group key agreement, encrypted direct messages) use X25519 Diffie-Hellman key exchange. X25519 operates on the same Curve25519 as Ed25519, and Ed25519 keys can be converted to X25519 keys for key exchange purposes using a well-defined birational map. This conversion allows MURMUR to use a single keypair for both signing and key exchange without maintaining separate key material.

X25519 provides 128-bit security against classical attacks and produces a 32-byte shared secret. The shared secret is processed through HKDF-SHA-256 (HMAC-based Key Derivation Function) to derive symmetric encryption keys, ensuring that the derived keys have uniform distribution regardless of any algebraic structure in the raw Diffie-Hellman output.

### ChaCha20-Poly1305 Symmetric Encryption

All symmetric encryption in MURMUR uses ChaCha20-Poly1305, an authenticated encryption with associated data (AEAD) construction. ChaCha20 provides the confidentiality layer (a stream cipher with 256-bit keys), and Poly1305 provides the integrity and authentication layer (a one-time authenticator producing 128-bit tags). The combination ensures that encrypted data cannot be read or tampered with without the symmetric key.

ChaCha20-Poly1305 was chosen over AES-GCM because ChaCha20 performs well in software on platforms without AES hardware acceleration (common on older mobile devices), its implementation is simpler and less susceptible to implementation errors, and it is not vulnerable to the timing attacks that affect some AES implementations.

### SHA-256 and BLAKE2b Hashing

MURMUR uses SHA-256 for Proof of Work stamp computation (the PoW hash function) and for content addressing (message IDs are SHA-256 hashes of the message content). SHA-256 provides 128-bit collision resistance and 256-bit preimage resistance.

BLAKE2b is used for identity-related hashing: sigil generation, pseudonym derivation, color derivation, and other operations where the hash output is used for non-security-critical purposes (visual identity rather than authentication). BLAKE2b was chosen for these applications because of its speed (faster than SHA-256 on most platforms) and its configurable output length.

### Pedersen Commitments and Bulletproofs

The Zero-Knowledge Claim system (used for Specter Resonance threshold proofs) uses Pedersen commitments on Curve25519 with Bulletproofs-style range proofs. A Pedersen commitment hides a value while allowing the committer to later prove properties of the value (such as "the committed value exceeds a threshold") without revealing the value itself.

Pedersen commitments provide information-theoretic hiding (the commitment reveals nothing about the committed value, even to an adversary with unlimited computing power) and computational binding (the committer cannot change the committed value after commitment, assuming the discrete logarithm problem on Curve25519 is hard).

Bulletproofs provide succinct, non-interactive zero-knowledge range proofs. A Bulletproof proves that a committed value lies within a specified range without revealing the value. The proof size is logarithmic in the range size (approximately 672 bytes for a 64-bit range), making it practical for inclusion in gossip messages. Verification is more computationally expensive than signature verification (approximately 10 milliseconds on a modern CPU) but acceptable for the relatively infrequent ZK Claim messages.

### Noise Protocol Framework

libp2p transport encryption uses the Noise protocol framework, specifically the Noise XX handshake pattern. The XX pattern provides mutual authentication with identity hiding — both parties authenticate to each other, but an eavesdropper cannot determine either party's identity from the handshake transcript. The handshake produces forward-secret session keys: even if a party's long-term private key is later compromised, past session keys cannot be recovered.

The Noise handshake is used for TCP and WebSocket connections. QUIC connections use TLS 1.3, which provides equivalent security properties (mutual authentication, forward secrecy, identity hiding from eavesdroppers) through a different protocol.

---

## Adversary Classes

MURMUR's threat model considers four classes of adversary, each with increasing capabilities.

### Class 1: Passive Local Observer

A passive local observer can monitor the network traffic of a single MURMUR node (for example, a Wi-Fi network operator, an ISP, or a co-located device performing ARP spoofing). The observer can see that the node is running MURMUR (from connection patterns and potentially from unencrypted DNS queries), can identify the IP addresses of the node's peers, and can observe the volume and timing of encrypted traffic. The observer cannot read the content of encrypted connections.

Against this adversary, MURMUR provides strong protection. All connections are encrypted, so message content is confidential. The observer can determine that the user is running MURMUR but cannot determine what the user is publishing, reading, or which identities the user controls. For Fortress-mode users routing through Shroud circuits, the observer can see connections to Shroud entry nodes but cannot determine the Specter identity or content associated with those connections.

Residual risk: the observer can perform traffic volume analysis. If the user publishes a large Wave (high outbound traffic) at a specific time, and a Wave appears on the network at that time, the observer can correlate timing. This risk is mitigated by the constant-rate traffic padding described in the Shroud Network design (Shroud circuits send fixed-size blobs at regular intervals regardless of actual message volume), but padding is imperfect — statistical analysis of long-term traffic patterns may still reveal correlations.

### Class 2: Malicious Peer

A malicious peer is a MURMUR node controlled by an adversary. The adversary runs one or more nodes that participate honestly in the network protocols (gossip, DHT, relay) while collecting data about other nodes' behavior. A malicious peer can observe all messages delivered through its gossip mesh connections, can observe DHT queries and responses it handles, can observe relay traffic it forwards (if serving as a Shroud Node), and can attempt to influence the network by selectively dropping or delaying messages.

Against this adversary, MURMUR provides moderate protection. Message content is public on the gossip layer (Waves are not encrypted), so a malicious peer can read all Waves it receives — but this is by design, as Waves are intended to be public. The adversary cannot forge messages (all messages are signed). The adversary cannot determine the authorship of Anonymous Layer messages received through gossip (the messages enter gossip through Shroud exit nodes, not from the authors directly). The adversary can disrupt its local mesh neighborhood by selectively dropping messages, but GossipSub's mesh redundancy ensures that messages propagate through alternate paths.

Residual risk: if the adversary controls a Shroud Node that is selected as part of a user's circuit, the adversary observes the encrypted blob passing through its relay but cannot decrypt it (only the exit node decrypts the outer layer). However, if the adversary controls both the entry node and the exit node of a 3-hop circuit, it can correlate the encrypted blob entering the circuit with the decrypted message exiting the circuit (timing correlation), de-anonymizing the author. MURMUR's circuit construction avoids selecting topologically close nodes for the same circuit, but a well-resourced adversary with many geographically distributed Shroud Nodes can still achieve entry-exit correlation with non-negligible probability.

### Class 3: Sybil Attacker

A Sybil attacker generates a large number of MURMUR identities (both main identities and Specter identities) to gain disproportionate influence over the network. The attacker's goals may include manipulating Specter Resonance rankings, swaying Specter Duel votes, overwhelming Masked Events with controlled participants, gaining admission to Phantom Councils, or operating enough Shroud Nodes to frequently capture both entry and exit positions in circuits.

Against this adversary, MURMUR's primary defense is Proof of Work. Every identity requires PoW to register (main identity PoW cost is approximately 0.5 seconds of CPU time). Every Wave requires PoW to publish. Operating a Shroud Node requires elevated PoW (24 leading zero bits, approximately 30 seconds per announcement). Creating thousands of identities and maintaining active participation across all of them requires sustained computational investment proportional to the number of identities.

Specter Resonance provides a secondary defense. Resonance is earned through sustained, quality participation over time. A Sybil attacker can create many Specters, but each Specter starts at Resonance 0 and must independently earn Resonance through Wave publication, amplification, duel participation, event participation, relay service, and connection formation. Building high Resonance across many Specters requires extensive, ongoing effort that cannot be shortcut.

Residual risk: PoW costs scale linearly with Sybil count, not exponentially. A well-resourced attacker (with access to cloud computing or specialized hardware) can generate thousands of identities at manageable cost. Resonance gating mitigates the impact of low-Resonance Sybils (they cannot vote in duel elections, join councils, or place marks), but the attacker can still use Sybils for gossip-layer disruption, DHT pollution, and traffic analysis.

### Class 4: Global Passive Adversary

A global passive adversary (GPA) can observe all internet traffic between all MURMUR nodes simultaneously. This adversary class models a nation-state intelligence agency with access to traffic data from major internet exchange points, ISPs, and undersea cables. The GPA cannot modify traffic (it is passive), but can observe the timing, volume, source, and destination of all encrypted connections.

Against this adversary, MURMUR provides limited protection. The GPA can observe which IP addresses are running MURMUR nodes (from connection patterns). The GPA can observe the timing of message publication by correlating outbound traffic from a node with the appearance of a new message on the gossip network (by monitoring the gossip from its own nodes or from compromised peers). For Anonymous Layer messages routed through Shroud circuits, the GPA can perform end-to-end timing correlation: observing a blob enter a Shroud circuit from the author's IP and exit the circuit at the exit node, correlating the two events by timing.

Shroud circuit padding (fixed-size blobs at regular intervals) provides some defense against timing correlation, but a GPA with sufficient traffic data and statistical sophistication can likely break this defense over time, particularly for users who publish frequently or at distinctive times.

Residual risk: MURMUR does not claim to protect against a GPA. The Shroud Network raises the cost of traffic analysis significantly compared to unprotected communication, but a determined GPA with months of traffic data can likely de-anonymize active Specters. Users who require GPA-level anonymity should combine MURMUR's protections with external measures (VPN, Tor, careful operational security).

---

## Attack Surface Analysis

### Identity Attacks

**Key Compromise.** If an adversary obtains a user's Ed25519 private key (main identity key, Specter key, or both), the adversary can impersonate the user, publish Waves in the user's name, read the user's encrypted messages, and forge the user's signatures. MURMUR cannot defend against key compromise — the private key is the user's identity. Users are responsible for protecting their private keys through device security, secure storage, and operational practices.

Mitigation: private keys are stored in the application's local data directory, encrypted at rest using a key derived from the user's device credentials (platform keychain on iOS/macOS, Keystore on Android, OS credential store on desktop). The private key never leaves the device, is never transmitted over the network, and is never included in any message or gossip payload.

**Identity Correlation.** An adversary who observes a user's Surface Layer activity and Anonymous Layer activity may attempt to correlate the two identities through behavioral analysis (writing style, topic interests, activity timing, social graph overlap). MURMUR's protocol-level separation (separate gossip topics, Shroud routing, distinct keypairs) prevents protocol-level correlation, but behavioral correlation is a user-level risk that the protocol cannot fully address.

Mitigation: the application does not provide tools to prevent behavioral correlation, but the documentation advises users to vary their behavior between layers (different topics, different writing style, different activity schedule) if they want strong unlinkability. The Masked Event mechanic provides a temporary escape from even Specter-level behavioral correlation by using single-use identities.

**Specter-Main Identity Linkage.** A specific and critical correlation attack targets the link between a user's Specter and their main identity. In Hybrid mode, the user's node participates in both layers from the same device. An adversary who compromises the device can access both keys. An adversary performing traffic analysis on the device's network can observe both Surface Layer and Anonymous Layer gossip traffic originating from the same IP. Shroud routing mitigates the gossip traffic correlation (Anonymous Layer messages exit from Shroud Nodes, not from the user's IP), but the device-level colocation of both identities is an irreducible risk of Hybrid mode.

Mitigation: Fortress mode eliminates this risk by disabling the Surface Layer entirely. Users who require strong unlinkability between their anonymous and public identities should use Fortress mode on a dedicated device.

### Network Attacks

**Eclipse Attack.** An adversary attempts to surround a target node with adversarial peers, isolating it from the honest network. An eclipsed node receives only adversary-controlled messages and can be fed false data (fabricated Waves, censored Waves, manipulated Resonance data).

Mitigation: the connection manager maintains connections across multiple DHT regions (Kademlia buckets), making it difficult for an adversary to occupy all connection slots. GossipSub mesh management selects mesh peers from diverse network locations. The DHT's Kademlia structure ensures that routing table entries span the full key space. An eclipse requires the adversary to control peers in many different DHT regions simultaneously, which requires a large Sybil population.

Residual risk: a sufficiently large Sybil population (controlling more than 50% of peers in every DHT region) can eclipse any target node. The PoW cost of maintaining such a Sybil population is significant but not prohibitive for a well-resourced adversary.

**Message Censorship.** An adversary controlling a node's mesh peers can selectively drop messages from specific publishers, effectively censoring them from the target node's view.

Mitigation: GossipSub's mesh redundancy (mesh degree 6) means that a message must be dropped by all mesh peers to be censored. The adversary must control all mesh peers of the target node for the relevant topic, which requires an eclipse-level attack. Additionally, GossipSub's gossip protocol (where non-mesh peers periodically share message metadata) allows the target to detect missing messages and request them from non-mesh peers.

**Gossip Flooding.** An adversary publishes a large volume of valid messages (meeting all PoW and format requirements) to overwhelm the gossip network's bandwidth and processing capacity.

Mitigation: PoW makes each message computationally expensive. Per-peer rate limits cap the message delivery rate from any single peer. GossipSub peer scoring penalizes peers that deliver excessive messages. The 2048-byte content limit bounds the size of individual messages. Together, these mechanisms ensure that flooding the network requires sustained computational investment proportional to the flood volume, while each honest node's processing load is bounded by its rate limits.

Residual risk: a determined adversary with sufficient computing power can produce valid messages at a rate that, while rate-limited per peer, still degrades network performance when aggregated across many adversarial peers. This is a fundamental limitation of PoW-based spam resistance — it raises the cost of spam but cannot eliminate it entirely.

**DHT Poisoning.** An adversary injects false records into the Kademlia DHT, directing nodes to non-existent peers, adversarial peers, or incorrect content locations.

Mitigation: DHT records in MURMUR are signed by the record's subject (peer records are signed by the peer's identity key, provider records are signed by the provider's key). Unsigned or incorrectly signed records are rejected. This prevents an adversary from publishing false records for peers it does not control. The adversary can still publish valid records for its own adversarial peers, but this is equivalent to the Sybil attack (which is bounded by PoW cost).

### Shroud Network Attacks

**Circuit Correlation.** An adversary controlling both the entry and exit nodes of a Shroud circuit can correlate traffic entering and exiting the circuit by timing, volume, or pattern, de-anonymizing the circuit's user.

Mitigation: circuit construction selects nodes that are topologically diverse (distant in Kademlia space). Circuits are rotated every 10 minutes, limiting the correlation window. Fixed-size blobs (4096 bytes) and regular transmission intervals reduce volume-based correlation. However, timing correlation remains a viable attack for adversaries who control a sufficient fraction of Shroud Nodes.

Quantitative risk: if the adversary controls fraction f of all Shroud Nodes, the probability of capturing both entry and exit positions of a random 3-hop circuit is approximately f². For f = 0.1 (10% of Shroud Nodes), the probability is 1%. For f = 0.3 (30%), the probability is 9%. Over many circuit rotations, the adversary accumulates correlation data: after 100 circuit rotations, an adversary controlling 10% of Shroud Nodes has a ~63% probability of having captured at least one circuit. This makes long-term de-anonymization feasible for adversaries with moderate Shroud Node penetration.

Mitigation: users are advised that Shroud routing provides probabilistic anonymity, not guaranteed anonymity. The anonymity guarantee degrades over time and with adversary Shroud Node penetration. Users requiring strong long-term anonymity should combine Shroud routing with external anonymity networks (Tor, VPN) and should avoid long-term behavioral patterns that aid correlation.

**Relay Denial of Service.** An adversary operating Shroud Nodes can selectively drop relay traffic, degrading circuit reliability and potentially forcing users to build circuits through adversary-controlled nodes (if honest Shroud Nodes become unreliable).

Mitigation: when a circuit fails (a relayed blob is not acknowledged within the timeout), the client constructs a new circuit excluding the failed node. Repeated failures from the same Shroud Node result in the node being blacklisted by the client. The Specter Resonance system deprioritizes Shroud Nodes with poor relay performance (the Shroud Node Uptime signal in the Resonance computation reflects observed relay success rates).

### Application-Level Attacks

**Client Compromise.** If an adversary compromises the MURMUR application on a user's device (through malware, supply chain attack, or physical access), the adversary gains access to all private keys, all locally stored messages, and the ability to impersonate the user on both layers. This is the most severe attack and MURMUR provides no defense against it — a compromised client is a compromised identity.

Mitigation: defense against client compromise is outside MURMUR's security boundary. Users are responsible for maintaining device security (OS updates, malware protection, physical security). The application code is open-source, allowing community audit of the client's behavior.

**Malicious Client Modifications.** An adversary distributes a modified MURMUR client that appears to function normally but exfiltrates private keys, logs activity, or weakens cryptographic operations.

Mitigation: the reference client is open-source with reproducible builds. Users can verify that their client binary was built from the published source code by reproducing the build and comparing hashes. Distribution through official channels (application stores with code signing) provides an additional layer of verification. However, users who install clients from unofficial sources or who do not verify builds are vulnerable to this attack.

**Content Injection via Wave Forgery.** An adversary attempts to publish a Wave that appears to come from another user by forging the signature.

Mitigation: this attack is computationally infeasible given the security of Ed25519. Forging a signature requires solving the discrete logarithm problem on Curve25519, which requires approximately 2¹²⁸ operations — far beyond any attacker's capabilities with current technology.

---

## Privacy Guarantees by Mode

### Open Mode

Open-mode users participate only on the Surface Layer. Their identity is their main Ed25519 keypair. Their Waves are signed with their main key and publicly attributable. Their connections are publicly declared. Their IP address is visible to their direct peers.

Privacy guarantees: connection encryption prevents passive observers from reading message content. The decentralized architecture prevents any single entity from having a complete view of the user's activity (unlike a centralized platform where the operator sees everything). The user's IP is visible to direct peers but not to the broader network (GossipSub does not propagate source IP information).

Privacy limitations: Open-mode users have no anonymity. Their identity is public, their activity is public, and their social graph is public. Any peer can observe their Waves, and their IP is known to their direct connections.

### Hybrid Mode

Hybrid-mode users participate on both layers. They have a main identity on the Surface Layer and a Specter identity on the Anonymous Layer. The two identities are cryptographically unlinked (different keypairs, different signatures). Anonymous Layer traffic is routed through Shroud circuits.

Privacy guarantees: all Open-mode guarantees apply to the Surface Layer identity. The Specter identity provides pseudonymous privacy — the Specter can interact on the Anonymous Layer without revealing the main identity. Shroud routing prevents network-layer observers from linking the Specter's gossip messages to the user's IP. Protocol-level separation (separate gossip topics, separate keypairs) prevents protocol-level correlation between the two identities.

Privacy limitations: both identities operate from the same device. A local observer (someone with access to the device's network traffic or the device itself) can determine that the same device is active on both layers. Behavioral correlation (writing style, timing, topic overlap) may link the two identities for a sufficiently motivated analyst. The Shroud Network provides probabilistic anonymity that degrades over time against adversaries with Shroud Node penetration.

### Fortress Mode

Fortress-mode users participate only on the Anonymous Layer. They have no Surface Layer identity. Their Peer ID is derived from a transport keypair unlinked to their Specter key. All traffic is routed through Shroud circuits.

Privacy guarantees: the Specter identity is the user's only identity. There is no Surface Layer identity to correlate with. The transport keypair (used for libp2p connections) is not linked to the Specter key at the protocol level. Shroud routing anonymizes the Specter's gossip messages. The user's IP is visible to direct peers but is linked only to the transport keypair, not to the Specter identity.

Privacy limitations: the user's IP is still visible to direct peers and can be linked to the transport keypair. While the transport keypair is not protocol-linked to the Specter key, a sophisticated traffic analysis attack could correlate the transport keypair's gossip activity with the Specter's message timing. Using a VPN or Tor for the underlying connection mitigates this risk. Behavioral correlation remains a risk if the user has other online identities with overlapping behavior patterns.

---

## Resonance System Security

### Manipulation Resistance

Specter Resonance is computed locally by each observing node based on publicly observable signals. This design eliminates the need for a trusted authority to compute or certify Resonance scores, but introduces the possibility of manipulation.

The Resonance signals and their manipulation resistance are as follows.

**Wave Publication Consistency** is based on the regularity and volume of a Specter's Wave publications over 30 days. This signal is difficult to fake because each Wave requires PoW, imposing a computational cost on publication volume. An attacker can boost this signal by publishing many Waves, but the cost scales linearly with the desired boost.

**Amplification Reception** is based on the number of amplifications a Specter's Waves receive from other Specters. This signal is vulnerable to Sybil manipulation — an attacker with many Specter identities can amplify their own Waves. However, amplifications from low-Resonance Specters contribute less to the signal than amplifications from high-Resonance Specters (the Resonance computation weights amplifications by the amplifier's Resonance). This creates a bootstrapping problem for Sybil attackers: their fake Specters have low Resonance and thus provide low-value amplifications.

**Connection Diversity** is based on the number and diversity of a Specter's connections to other Specters. This signal rewards Specters with connections to many distinct, high-Resonance Specters. Sybil manipulation requires creating many fake Specters with independent high Resonance, which is expensive (see above).

**Duel Record** is based on net wins in Specter Duels over 30 days. This signal is resistant to manipulation because duel outcomes are determined by audience votes weighted by Resonance. An attacker would need to control a large Resonance-weighted voting bloc to guarantee duel victories.

**Shroud Node Uptime** is based on the Specter's availability and reliability as a Shroud Node. This signal requires the Specter to actually operate a Shroud Node and relay traffic successfully, which imposes ongoing resource costs (bandwidth, availability).

**Whisper Chain Contributions** is a self-reported signal (each node tracks how many Whisper Chain blobs it has relayed). This is the weakest signal in the Resonance computation because it cannot be independently verified. It is weighted accordingly — it contributes at most 5% of total Resonance.

**Event Participation** is based on Resonance Bursts from Masked Event participation. Bursts are computed from amplification data within events, which is subject to the same Sybil considerations as general amplification but within the smaller, time-limited context of an event.

### ZK Claim Security

ZK Claims (used to prove Specter Resonance exceeds a threshold without revealing the exact value) rely on Pedersen commitments and Bulletproofs. The security of ZK Claims depends on the discrete logarithm assumption on Curve25519 (for commitment binding) and the soundness of the Bulletproofs protocol (for range proof validity).

A valid ZK Claim proves that the claimant's locally computed Resonance exceeds the stated threshold. It does not prove that the Resonance was computed correctly — a malicious node could compute an inflated Resonance value and produce a valid ZK Claim for that inflated value. However, other nodes can independently compute the claimant's Resonance from publicly observable signals and reject the claim if their independent computation disagrees. The ZK Claim is a convenience mechanism (allowing threshold checks without revealing exact values), not a trust mechanism (it does not replace independent verification).

---

## Content Security

### Content Authenticity

Every Wave is signed by its author's Ed25519 key. Recipients verify the signature before processing or forwarding the Wave. This guarantees that the content was published by the holder of the signing key and has not been modified in transit. Content authenticity is the strongest security property in MURMUR — it is guaranteed by the mathematical properties of Ed25519 and does not depend on any network behavior or trust assumption.

### Content Integrity

The combination of Ed25519 signatures and SHA-256 message IDs ensures that any modification to a Wave's content invalidates both its signature and its message ID. Recipients independently compute the message ID from the received content and verify it against the declared ID. Any discrepancy results in the message being rejected. Content integrity is absolute — no adversary can modify a Wave's content without being detected.

### Content Availability

Content availability is the weakest security property in MURMUR. Because there are no servers and no permanent storage, content exists only on the devices of nodes that have received and retained it. The 30-day content window means that all content is eventually garbage collected. Content from poorly connected publishers may be retained by only a handful of nodes. If those nodes go offline or clear their storage, the content is permanently lost.

MURMUR does not guarantee content availability. It provides best-effort availability through gossip propagation (which ensures wide initial distribution for well-connected publishers) and sync (which allows returning nodes to retrieve missed content from peers that retained it). Users who require durable content storage should maintain their own archives outside of MURMUR.

### Content Censorship

In MURMUR, censorship means preventing a Wave from reaching its intended audience. Because MURMUR has no central content authority, censorship requires network-level interference: an adversary must prevent the Wave from propagating through the gossip mesh. As discussed in the Network Attacks section, this requires an eclipse-level attack on either the publisher or the target audience.

For the Anonymous Layer, Shroud routing adds an additional layer of censorship resistance — an adversary must not only prevent the message from propagating through gossip but also prevent it from entering the gossip in the first place (by disrupting all of the publisher's Shroud circuits).

MURMUR provides no protection against self-censorship (a user's choice not to publish) or social censorship (community pressure that discourages publication). These are social phenomena outside the protocol's scope.

---

## Trust Model

### Zero Trust Architecture

MURMUR operates on a zero-trust model. No node trusts any other node. Every message is independently verified (signature, PoW, format, timestamp). Every computation is independently performed (Resonance scores, duel tallies, event summaries). Every claim is independently checked (ZK Claims are verified against local Resonance computations). The protocol is designed so that a node's security does not degrade even if all of its peers are adversarial — the node can verify all critical properties from the message data alone.

The one exception to this zero-trust model is the Whisper Chain Contributions signal in Resonance computation, which is self-reported and unverifiable. This signal is weighted minimally (at most 5% of total Resonance) to limit the impact of false reporting.

### Trust-on-First-Use for Peer Identity

When a node connects to a peer, it verifies the peer's identity through the Noise or TLS handshake (the peer proves possession of its Ed25519 private key). This is a trust-on-first-use (TOFU) model — the node trusts that the peer's public key is the peer's true identity because the peer proved possession of the corresponding private key. There is no certificate authority or web of trust validating that the public key belongs to a specific real-world entity.

This TOFU model is appropriate for MURMUR because identities are pseudonymous. The question is not "does this public key belong to Alice?" (which would require an identity authority) but "is this the same entity I communicated with previously?" (which is answered by key continuity). If a user needs to verify that a MURMUR identity corresponds to a specific real-world person, they must verify the public key fingerprint through an out-of-band channel (in person, voice call, or other pre-existing trusted communication).

### No Trusted Third Parties

MURMUR has no trusted third parties. There are no certificate authorities, no identity providers, no content moderators, no relay operators with special privileges, and no developers with backdoor access. The protocol is designed so that no entity, including MURMUR's developers, can read encrypted communications, forge identities, censor content, or de-anonymize users through the protocol itself.

Bootstrap nodes are the closest element to a trusted third party, as they are the first peers a new node contacts. However, bootstrap nodes have no special capabilities — they are ordinary nodes that happen to be listed in the default configuration. A malicious bootstrap node can attempt to eclipse a new node, but the new node's connection manager and DHT queries will quickly discover additional peers, diluting the bootstrap node's influence.

---

## Known Limitations

### Quantum Computing Vulnerability

All of MURMUR's public-key cryptography (Ed25519 signatures, X25519 key exchange, Pedersen commitments) is vulnerable to attack by a sufficiently powerful quantum computer. Shor's algorithm can solve the discrete logarithm and elliptic curve discrete logarithm problems in polynomial time, breaking the security of these primitives.

As of the current specification, no production-ready post-quantum cryptographic algorithms have been integrated into MURMUR. The NIST post-quantum cryptography standardization process has produced candidate algorithms (CRYSTALS-Dilithium for signatures, CRYSTALS-Kyber for key exchange) that could replace Ed25519 and X25519. Migration to post-quantum primitives is planned as a future protocol upgrade, implemented through the protocol versioning mechanism. The migration will require a breaking protocol change (new message formats with larger keys and signatures), managed through the 90-day deprecation process.

Users should be aware that encrypted communications captured today could be decrypted in the future by an adversary with a quantum computer ("harvest now, decrypt later" attack). Shroud circuit traffic and Whisper Chain messages are the most sensitive to this attack. Phantom Council communications encrypted with ChaCha20-Poly1305 are quantum-resistant at the symmetric layer (ChaCha20 with 256-bit keys provides 128-bit security against quantum attacks via Grover's algorithm), but the key exchange that establishes the symmetric key is vulnerable.

### Metadata Leakage

MURMUR's encryption protects content but does not fully protect metadata. The following metadata is observable by various adversary classes.

A passive local observer can determine that the user is running MURMUR, can observe the volume and timing of encrypted traffic, and can identify the IP addresses of the user's direct peers.

A malicious peer can observe which gossip topics the user subscribes to (indicating the user's layer participation and interest areas), can observe the timing of the user's message deliveries (which correlates with publication timing for the user's own messages), and can observe the user's DHT queries (which may reveal which peers or content the user is seeking).

A global passive adversary can correlate traffic timing across all nodes, potentially linking Shroud circuit entry and exit traffic, identifying Whisper Chain relay paths, and mapping the network's social graph from connection patterns.

MURMUR's Shroud routing, fixed-size padding, and regular-interval transmission mitigate some metadata leakage but do not eliminate it. Complete metadata protection (as provided by systems like Loopix or Vuvuzela) would require mixing and significant latency, which is incompatible with MURMUR's real-time social experience.

### Endpoint Security

MURMUR's security model assumes that the user's device is secure. If the device is compromised (malware, physical access, evil maid attack), all MURMUR security guarantees are void. The adversary has access to private keys, plaintext messages, identity linkages, and full activity history.

MURMUR does not implement any endpoint security measures beyond encrypting private keys at rest using the platform's credential store. Advanced endpoint protections (trusted execution environments, hardware security modules, memory encryption) are outside the current specification's scope and may be considered for future versions.

### Social Graph Privacy

MURMUR's Surface Layer social graph is public — connections between main identities are declared in signed connection messages visible to all participants. This is a design choice (public connections enable the Pulse Map topology visualization and social navigation features), but it exposes users' social relationships to any observer.

The Anonymous Layer social graph (Specter connections) is also public within the Anonymous Layer — Specter connections are visible to Anonymous Layer participants. This is less concerning because Specter identities are pseudonymous, but an adversary who de-anonymizes a Specter (through traffic analysis or behavioral correlation) learns the Specter's anonymous social graph.

Users who require social graph privacy should be aware that MURMUR does not provide it. The decision to make social graphs visible was a deliberate trade-off favoring social navigation and community formation over social graph confidentiality.

### Lack of Forward Secrecy for Published Content

Waves are signed with the publisher's long-term Ed25519 key. If the key is later compromised, the adversary can verify (but not forge — the adversary needs the private key to forge) all previously published Waves. More importantly, the adversary can impersonate the user going forward.

MURMUR does not implement key rotation for published content. A user whose key is compromised must generate a new identity and re-establish their social connections. There is no mechanism to revoke a compromised key (since there is no authority to certify revocations) — the user must communicate the compromise to their contacts through out-of-band channels.

For encrypted communications (Whisper Chain messages, Council messages), forward secrecy is provided by the ephemeral key exchanges used to establish session keys. Compromising a long-term key does not retroactively decrypt past encrypted communications. However, it does allow the adversary to impersonate the user in future key exchanges, potentially intercepting future encrypted communications.

### Scalability Limits

MURMUR's decentralized architecture has inherent scalability limits. Gossip propagation bandwidth scales linearly with network activity (more Waves means more gossip traffic). The force-directed layout algorithm scales approximately as O(n log n) with the number of visible nodes. DHT lookup latency scales as O(log n) with network size. Storage consumption scales linearly with retained content volume.

The practical scalability ceiling depends on user hardware. For the target hardware profile (consumer devices from 2020+), the estimated ceiling is approximately 100,000 active nodes with average activity rates. Beyond this scale, gossip bandwidth exceeds typical residential internet capacity, and the Pulse Map's layout computation exceeds mobile CPU budgets. Scaling beyond this ceiling would require protocol changes (gossip sharding, hierarchical routing, or federated relay infrastructure) that are outside the current specification's scope.

### No Content Moderation

MURMUR has no content moderation mechanism. There is no authority that can remove Waves, ban users, or enforce community standards. The mute function (hiding a user's content in the local view) is the only tool available for managing unwanted content, and it operates purely locally — it does not affect what other users see.

This is a deliberate design choice: content moderation requires a trusted authority, and MURMUR's architecture eliminates trusted authorities. The consequence is that MURMUR's network may contain content that some or all users find objectionable. Users must exercise personal judgment in navigating the network and managing their exposure to content through the mute function and selective connection management.

The 30-day content window provides a natural bound: all content eventually expires and is garbage collected, preventing permanent accumulation of objectionable material. However, this is a weak mitigation — 30 days is a long time for harmful content to persist.

---

## Security Recommendations for Users

Users seeking the strongest privacy and security posture should adopt the following practices.

For identity protection, use Fortress mode on a dedicated device. Do not operate both a Surface Layer identity and a Specter identity from the same device. Generate separate keypairs on separate devices. Do not reuse writing style, vocabulary, or topic interests between your public and anonymous identities.

For network privacy, use a VPN or Tor as the underlying transport for MURMUR, especially in Fortress mode. This hides your IP address from direct peers and adds an additional anonymization layer beneath Shroud routing.

For device security, keep your operating system and MURMUR client updated. Use full-disk encryption. Use a strong device passcode. Do not install MURMUR clients from unofficial sources. Verify reproducible builds when possible.

For key management, back up your private keys to a secure offline location (encrypted USB drive, paper backup in a secure location). If you suspect key compromise, generate a new identity immediately and notify your contacts through out-of-band channels.

For operational security, be aware that your activity patterns (publication timing, response latency, topic interests) can be used to correlate your identities across layers or across platforms. Vary your behavior if you want strong unlinkability. Do not discuss information on the Anonymous Layer that uniquely identifies you on the Surface Layer.