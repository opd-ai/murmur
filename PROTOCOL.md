# MURMUR Wire Protocol Specification

Version 1.0 (stable)

This document specifies the MURMUR network wire format and protocols. It is intended for implementers of compatible MURMUR clients and network services. This specification is intentionally implementation-agnostic — a correct implementation may be written in any language and need not use libp2p, Ebitengine, or Go.

## 1. Overview

MURMUR is a decentralized peer-to-peer social network. The network consists of autonomous nodes that:

- Exchange signed ephemeral messages called Waves over a gossip protocol
- Maintain peer-to-peer relationships encoded in signed declarations
- Publish signed advertisements for Shroud relay services
- Construct multi-hop anonymity circuits using onion-style encryption

All wire-format messages use Protocol Buffers (proto3) serialization and are wrapped in a standardized envelope with signature and deduplication metadata.

## 2. Wire Format

### 2.1 Message Envelope

Every message broadcast over the network is wrapped in a `MurmurEnvelope`:

```protobuf
message MurmurEnvelope {
  uint32 version = 1;               // Protocol version (currently 1)
  MessageType type = 2;             // Message type discriminator (WAVE, IDENTITY, SHROUD_AD, HEARTBEAT)
  bytes payload = 3;                // Serialized inner message (type-specific protobuf)
  bytes sender_pubkey = 4;          // 32-byte Ed25519 public key (zeroed for anonymous messages)
  bytes signature = 5;              // Ed25519 signature over (version || type || payload)
  int64 timestamp_unix = 6;         // Sender's Unix timestamp in seconds
  bytes message_id = 7;             // 32-byte BLAKE3 hash of payload (deduplication key)
}

enum MessageType {
  MESSAGE_TYPE_UNSPECIFIED = 0;
  MESSAGE_TYPE_WAVE = 1;            // Wave content message
  MESSAGE_TYPE_IDENTITY = 2;        // Identity declaration or device authorization
  MESSAGE_TYPE_SHROUD_AD = 3;       // Shroud relay advertisement
  MESSAGE_TYPE_HEARTBEAT = 4;       // Lightweight peer presence signal
}
```

#### 2.1.1 Envelope Validation

Every received envelope MUST be validated as follows:

1. **Protocol version check**: `version` MUST equal 1. Reject envelopes with unknown versions.
2. **Timestamp bounds**: Reject if `timestamp_unix` is more than 300 seconds in the future (clock skew tolerance). Reject if `timestamp_unix` is older than the **message's advertised TTL** if available (e.g., for Waves, reject if `timestamp_unix < now - wave.ttl_seconds`). This prevents replay of expired content.
3. **Signature verification**: Verify the Ed25519 signature over the serialized bytes `[version, type, payload]` (little-endian encoding). If `sender_pubkey` is all zeros, this is an anonymous message; verify the signature against the ZK proof embedded in the payload (see §4.2).
4. **Message ID validation**: Recompute `BLAKE3(payload)` and compare to the provided `message_id`. Reject on mismatch (indicates corruption or tampering).
5. **Deduplication**: Maintain a seen-message cache (Bloom filter or equivalent) keyed by `message_id`. Reject if the message has already been processed in the current TTL window.

#### 2.1.2 Signature Encoding

The signature data structure is:

```
signature_data = serialize([
  version (4 bytes, little-endian uint32),
  type (4 bytes, little-endian uint32),
  payload (variable length)
])
signature = Ed25519Sign(sender_private_key, signature_data)
```

The signature MUST be a 64-byte Ed25519 signature per RFC 8032.

### 2.2 Message Types

#### 2.2.1 Wave (MESSAGE_TYPE_WAVE)

A Wave is an ephemeral content message with optional Proof of Work and cryptographic signatures.

```protobuf
message Wave {
  WaveType wave_type = 1;           // Type of Wave (0x01-0x08)
  bytes content = 2;                // Text content (UTF-8, max 2048 bytes)
  bytes author_pubkey = 3;          // 32-byte Ed25519 public key
  bytes signature = 4;              // Ed25519 signature over (wave_type || content || created_at || ttl)
  int64 created_at = 5;             // Creation timestamp (Unix seconds)
  int64 ttl_seconds = 6;            // Time-to-live (default 604800 seconds = 7 days, max 2592000 = 30 days)
  uint64 pow_nonce = 7;             // Proof of Work nonce
  bytes parent_hash = 8;            // BLAKE3 hash of parent Wave (empty for root messages)
  uint32 hop_count = 9;             // Hop count (incremented by each relay)
  bytes wave_id = 10;               // BLAKE3 hash of (content || author_pubkey || created_at || ttl[...])
  map<string, bytes> metadata = 11; // Type-specific metadata (encryption params, veil flags, etc.)
  bytes device_public_key = 12;     // Device key that signed this Wave (optional; see §2.2.1.1)
}

enum WaveType {
  WAVE_TYPE_UNSPECIFIED = 0;
  WAVE_TYPE_SURFACE = 1;            // Standard Surface Layer Wave
  WAVE_TYPE_REPLY = 2;              // Reply to another Wave
  WAVE_TYPE_VEILED = 3;             // Encrypted to specific recipients
  WAVE_TYPE_SPECTER = 4;            // Anonymous Specter Wave
  WAVE_TYPE_SIGIL = 5;              // Sigil update announcement
  WAVE_TYPE_ABYSSAL = 6;            // Deep anonymous content
  WAVE_TYPE_MASKED = 7;             // Partially revealed identity
  WAVE_TYPE_BEACON = 8;             // Network coordination signal
}
```

**Validation**:

1. Verify `ttl_seconds` is between 1 and 2,592,000 (30 days).
2. Verify the Wave's signature over `(wave_type || serialize(content) || created_at || ttl_seconds)`.
3. Verify Proof of Work: compute `SHA256(wave_id || pow_nonce)` and confirm it has at least 20 leading zero bits (default difficulty).
4. Reject if `hop_count` exceeds a reasonable limit (e.g., 32) to prevent looping.
5. If `device_public_key` is present, verify that the signature was generated by that device key. Then verify that the device key is authorized by `author_pubkey` (requires checking device authorization declarations on `/murmur/identity/1` topic).

**Multi-Device Signature Verification** (§2.2.1.1):

If `device_public_key` is non-empty:
1. Verify the signature against `device_public_key`.
2. Look up the device authorization chain: fetch `DeviceAuthorizationDeclaration` messages on `/murmur/identity/1` signed by `author_pubkey` that list `device_public_key` as authorized.
3. Verify the device authorization is not revoked (check for `DeviceRevocationDeclaration` messages signed by `author_pubkey` that list `device_public_key`).
4. If device authorization is not found or is revoked, reject the Wave.

If `device_public_key` is empty, the signature MUST be verifiable against `author_pubkey` itself (legacy single-device mode).

#### 2.2.2 Identity Declaration (MESSAGE_TYPE_IDENTITY)

Identity declarations broadcast identity metadata, connections, and device authorizations.

```protobuf
message IdentityDeclaration {
  bytes public_key = 1;             // Node's public key
  string username = 2;              // Optional human-readable name (max 64 bytes)
  bytes sigil_image = 3;            // Cached 64x64 sigil visual (PNG)
  ShadowGradient gradient_mode = 4; // Privacy mode (Open, Hybrid, Guarded, Fortress)
  int64 declared_at = 5;            // Declaration timestamp
  bytes signature = 6;              // Ed25519 signature over fields 1-5
}

message DeviceAuthorizationDeclaration {
  bytes master_public_key = 1;      // Master identity public key
  bytes device_public_key = 2;      // Authorized device public key
  int64 authorized_at = 3;          // Authorization timestamp
  int64 expires_at = 4;             // Expiry timestamp (0 = no expiry)
  bytes signature = 5;              // Master key signature
}

message DeviceRevocationDeclaration {
  bytes master_public_key = 1;      // Master identity public key
  bytes device_public_key = 2;      // Revoked device public key
  int64 revoked_at = 3;             // Revocation timestamp
  bytes signature = 4;              // Master key signature
}

enum ShadowGradient {
  SHADOW_GRADIENT_UNSPECIFIED = 0;
  SHADOW_GRADIENT_OPEN = 1;         // Identity publicly visible; Surface only
  SHADOW_GRADIENT_HYBRID = 2;       // Surface + Specter; Specter requires Shroud
  SHADOW_GRADIENT_GUARDED = 3;      // Anonymous on Specter; Surface visible to contacts
  SHADOW_GRADIENT_FORTRESS = 4;     // Fully anonymous; all traffic routed via Shroud
}
```

**Validation**:

1. Verify the signature over all fields (1-5) using `public_key`.
2. For device declarations: verify the signature using `master_public_key` (not `device_public_key`).
3. For device revocations: verify the signature using `master_public_key`.

#### 2.2.3 Shroud Relay Advertisement (MESSAGE_TYPE_SHROUD_AD)

A relay node that wishes to serve Shroud circuits publishes a signed advertisement declaring availability.

```protobuf
message RelayAdvertisement {
  bytes relay_pubkey = 1;           // Relay's Curve25519 public key (32 bytes)
  uint32 bandwidth_capacity = 2;    // Advertised capacity (megabits per second)
  uint32 max_circuits = 3;          // Maximum concurrent circuits
  int64 advertised_at = 4;          // Advertisement timestamp
  int64 expires_at = 5;             // Expiry timestamp
  bytes signature = 6;              // Ed25519 signature from the relay's Surface key
}
```

**Validation**:

1. Verify the signature using an Ed25519 public key associated with the relay (typically derived from the libp2p peer ID or advertised separately).
2. Reject if expired (`expires_at < now`).

#### 2.2.4 Heartbeat (MESSAGE_TYPE_HEARTBEAT)

A lightweight presence signal published every 30 seconds by every active node.

```protobuf
message Heartbeat {
  bytes node_pubkey = 1;            // Node's public key
  int64 timestamp = 2;              // Timestamp (Unix seconds)
  bytes signature = 3;              // Ed25519 signature over (node_pubkey || timestamp)
}
```

**Validation**:

1. Verify the signature.
2. Reject if `timestamp` is not recent (within 60 seconds of now).

### 2.3 Additional Message Types

MURMUR defines additional message types in the `GossipMessage` union for game events, connection declarations, and other metadata. See `proto/gossip.proto`, `proto/mechanics.proto`, and other proto files in the codebase for the complete list.

The envelope validation rules (§2.1.1) apply uniformly to all message types.

## 3. GossipSub Topics

All broadcast communication flows through **GossipSub v1.1** topics. Implementations MUST support topic filtering, message validation, and peer scoring as specified.

### 3.1 Topic Definitions

#### `/murmur/waves/1`

Carries all Wave content messages (new posts, replies, amplifications).

- **Message type**: `MESSAGE_TYPE_WAVE`
- **Payload type**: `Wave` protobuf
- **Frequency**: Variable; depends on user activity
- **Persistence**: Waves expire after TTL (default 7 days)
- **Peer scoring**: Nodes that repeatedly publish invalid Waves (bad signatures, failed PoW, expired TTL) accumulate negative scores and are pruned from the mesh
- **Deduplication window**: TTL of the Wave (up to 30 days)

#### `/murmur/identity/1`

Carries identity metadata, declarations, and device authorizations.

- **Message type**: `MESSAGE_TYPE_IDENTITY`
- **Payload type**: `IdentityDeclaration`, `DeviceAuthorizationDeclaration`, `DeviceRevocationDeclaration`, `ProfileUpdate`, etc.
- **Frequency**: Low; declarations are updated infrequently (e.g., when changing privacy mode or authorizing a new device)
- **Persistence**: Indefinite (identity declarations are cached by public key; device declarations stored in the identity bucket)
- **Peer scoring**: Moderate; duplicates and malformed declarations receive minor penalties
- **Deduplication window**: Indefinite (dedup by `(type, public_key, event_id)` tuple)

#### `/murmur/shroud/1`

Carries Shroud relay advertisements.

- **Message type**: `MESSAGE_TYPE_SHROUD_AD`
- **Payload type**: `RelayAdvertisement`
- **Frequency**: Every 5–10 minutes per relay node
- **Persistence**: Until expiry (typically 30 minutes ahead of now)
- **Peer scoring**: Moderate; relays that advertise unreachable/saturated bandwidth receive penalties
- **Deduplication window**: Until expiry

#### `/murmur/pulse/1`

Carries lightweight heartbeat pings.

- **Message type**: `MESSAGE_TYPE_HEARTBEAT`
- **Payload type**: `Heartbeat`
- **Frequency**: Every 30 seconds per node
- **Persistence**: Transient; old heartbeats are ignored
- **Peer scoring**: None (heartbeats are used for liveness detection, not peer scoring)
- **Deduplication window**: 30 seconds

### 3.2 Topic Subscription

Implementations MUST be subscribed to all four topics at startup. Unsubscribing from a topic MUST NOT happen during normal operation.

Topic subscription is managed via the libp2p PubSub interface:

```go
// Pseudo-code
pubsub := gossipsub.New(ctx, host)
waveTopic, _ := pubsub.Join("/murmur/waves/1")
identityTopic, _ := pubsub.Join("/murmur/identity/1")
shroudTopic, _ := pubsub.Join("/murmur/shroud/1")
pulseTopic, _ := pubsub.Join("/murmur/pulse/1")
```

### 3.3 GossipSub Peer Scoring

Nodes implement GossipSub peer scoring to penalize misbehaving peers. Scoring parameters:

- **First message decode penalty**: -20 points per peer for each message that fails protobuf decode
- **First message validation penalty**: -50 points per peer per invalid message (bad signature, failed PoW)
- **Invalid message cap**:Negative score floors at -500 after which the peer is pruned
- **Valid message delivery reward**: +0.5 points per peer for each valid message delivered first by that peer
- **Score decay**: Scores gradually decay to 0 at a rate of 10 points per 10 seconds of inactivity
- **IP colocation penalty**: If multiple peers are detected behind the same IP address, apply a colocation penalty of -5 points per peer beyond the first (to discourage Sybil attacks)

## 4. Point-to-Point Stream Protocols

Binary request-response communication uses libp2p stream protocols. All stream protocols use the same envelope format as GossipSub (§2.1).

### 4.1 Shroud Circuit Construction (`/murmur/shroud-circuit/1`)

Shroud is a three-hop onion-routing network for anonymous message delivery. Circuit construction uses a custom protocol on `/murmur/shroud-circuit/1`.

#### 4.1.1 Circuit Construction Flow

1. **Initiator selects 3 relays** from available `RelayAdvertisement` messages on `/murmur/shroud/1`, excluding direct mesh peers (to reduce correlation).

2. **Initiator → Hop 1 (Key Exchange)**:

   ```
   Initiator generates ephemeral Curve25519 keypair (e1_private, e1_public).
   Initiator sends: e1_public (32 bytes)
   Hop 1 generates ephemeral keypair (e1h_private, e1h_public).
   Hop 1 sends: e1h_public (32 bytes)
   Shared secret S1 = X25519(e1_private, e1h_public)
   Key derivation: K1 = HKDF-SHA256(S1, salt="murmur-shroud-hop-1", info="encrypt", length=32)
                   M1 = HKDF-SHA256(S1, salt="murmur-shroud-hop-1", info="mac", length=32)
   ```

3. **Initiator → Hop 1 (Relay Cell)**:

   ```
   Create TunnelExtendCell:
     relay_pubkey = Hop 2's public key
     next_hop_address = Hop 2's multiaddr (e.g., /ip4/192.0.2.1/tcp/30333/p2p/Qm...)
   Encrypt under K1:
     ciphertext = XChaCha20-Poly1305Enc(K1, TunnelExtendCell)
   Send: ciphertext (encrypted RelayExtend cell is forwarded to Hop 2)
   ```

4. **Hop 1 → Hop 2 (Key Exchange)** (relayed through Initiator):

   ```
   (Same as step 2, but between Hop 1 and Hop 2)
   ```

5. **Hop 1 → Initiator (Second Handshake)** (encrypted under K1):

   ```
   Hop 1 sends the public key from Hop 2's key exchange, encrypted under K1.
   Initiator decrypts and performs X25519 with Hop 2.
   ```

6. **Repeat for Hop 3**.

After successful construction, the initiator has three keys: `K1`, `K2`, `K3` for encryption/decryption at each hop.

#### 4.1.2 Circuit Message Format

```protobuf
message TunnelCell {
  uint32 cell_type = 1;             // RELAY_EXTEND, RELAY_DATA, RELAY_CLOSE, etc.
  uint32 stream_id = 2;             // Unique stream identifier within the circuit
  bytes payload = 3;                // Type-specific data
}

message TunnelExtendCell {
  bytes relay_pubkey = 1;           // Curve25519 public key of next relay
  string next_hop_address = 2;      // Multiaddr to dial (e.g., /ip4/.../p2p/...)
  bytes ephemeral_pubkey = 3;       // Initiator's ephemeral key for next hop
}

message TunnelDataCell {
  uint32 stream_id = 1;
  bytes data = 2;                   // Encrypted/layered message data
}
```

#### 4.1.3 Encryption and Decryption

To send a message through the circuit:

```
payload_0 = original_message
payload_1 = XChaCha20-Poly1305Enc(K3, payload_0)
payload_2 = XChaCha20-Poly1305Enc(K2, payload_1)
payload_3 = XChaCha20-Poly1305Enc(K1, payload_2)
Send: TunnelDataCell with payload_3
```

Each hop:
1. Decrypts the outermost layer
2. Forwards the result to the next hop

The exit hop (Hop 3) decrypts the final layer and sees the plaintext payload, then publishes it on the destination GossipSub topic.

**Encryption details**:

- Algorithm: XChaCha20-Poly1305 (AEAD)
- Key length: 32 bytes
- Nonce length: 24 bytes (generated randomly for each encryption)
- Nonce is prepended to ciphertext (transparently handled by the XChaCha20-Poly1305 library)

### 4.2 Wave Sync (`/murmur/wave-sync/1`)

When a node receives a Wave that references a parent Wave it does not have, it can sync the missing Wave.

```protobuf
message WaveSyncRequest {
  bytes wave_hash = 1;              // BLAKE3 hash of the desired Wave
}

message WaveSyncResponse {
  bytes wave_data = 1;              // Serialized Wave protobuf (empty if not found)
}
```

**Protocol**:

1. Initiator opens a `/murmur/wave-sync/1` stream to a peer.
2. Initiator sends: serialized `WaveSyncRequest`
3. Peer responds with: serialized `WaveSyncResponse`

### 4.3 Peer Exchange (`/murmur/peer-exchange/1`)

A new node with minimal DHT state can bootstrap peer connections via peer exchange.

```protobuf
message PeerExchangeRequest {
  uint32 count = 1;                 // Desired number of peers (typically 10-20)
}

message PeerExchangeResponse {
  repeated PeerInfo peers = 1;
}

message PeerInfo {
  bytes public_key = 1;             // Peer's libp2p public key (32+ bytes)
  repeated string multiaddrs = 2;   // Peer's multiaddrs (e.g., /ip4/.../tcp/30333/p2p/...)
  int64 last_seen = 3;              // Last connection timestamp
}
```

**Protocol**:

1. Initiator opens a `/murmur/peer-exchange/1` stream to a bootstrap peer.
2. Initiator sends: serialized `PeerExchangeRequest`
3. Peer responds with: serialized `PeerExchangeResponse` containing recently-seen peers

## 5. Cryptography

### 5.1 Signature Algorithm

All digital signatures use **Ed25519** (RFC 8032) unless explicitly noted (e.g., anonymous messages via Shroud use ZK proofs).

- Public key length: 32 bytes
- Signature length: 64 bytes
- Signing: `Ed25519Sign(privateKey, message)`
- Verification: `Ed25519Verify(publicKey, message, signature)`

Implementations MUST use a standards-conformant Ed25519 library. Go implementations can use `crypto/ed25519`. JavaScript implementations can use `tweetnacl.js` or `libsodium.js`. Rust implementations can use `ed25519-dalek`.

### 5.2 Symmetric Encryption

**XChaCha20-Poly1305** is used for all layer encryption and symmetric key encryption.

- Key length: 32 bytes
- Nonce length: 24 bytes
- AEAD cipher (provides both confidentiality and authenticity)

Implementations MUST use a standards-conformant XChaCha20-Poly1305 library. Go implementations can use `golang.org/x/crypto/chacha20poly1305`. JavaScript implementations can use `libsodium.js`.

### 5.3 Key Derivation

**HKDF-SHA256** derives subkeys from shared secrets.

```
Key = HKDF-SHA256(
  input_key_material = shared_secret,
  salt = context_bytesempty if not provided),
  info = info_string,
  length = desired_key_length
)
```

### 5.4 Proof of Work

Each Wave includes a Hashcash-style Proof of Work to prevent spam.

```
Difficulty D = 20 (default; may change with protocol version)
Wave ID = BLAKE3(wave_content || metadata)
Find nonce such that:
  SHA256(Wave ID || nonce) has at least D leading zero bits
```

Verification is a single SHA256 evaluation.

### 5.5 Cryptographic Hash Functions

- **BLAKE3**: Identity hashing, message deduplication, Wave ID generation
- **SHA-256**: Proof of Work

## 6. Implementation Checklist

A correct MURMUR client implementation MUST:

### 6.1 Networking

- [ ] Implement libp2p or equivalent (define custom transports if needed)
- [ ] Support Noise XX transport encryption
- [ ] Implement GossipSub v1.1 with peer scoring
- [ ] Subscribe to all 4 topics (`/murmur/waves/1`, `/murmur/identity/1`, `/murmur/shroud/1`, `/murmur/pulse/1`)
- [ ] Implement stream protocol handlers for `/murmur/shroud-circuit/1`, `/murmur/wave-sync/1`, `/murmur/peer-exchange/1`
- [ ] Implement Kademlia DHT for peer discovery

### 6.2 Cryptography

- [ ] Implement Ed25519 signing and verification
- [ ] Implement XChaCha20-Poly1305 encryption and decryption
- [ ] Implement HKDF-SHA256 key derivation
- [ ] Implement BLAKE3 hashing
- [ ] Implement SHA-256 and Proof of Work verification
- [ ] Generate and store keypairs securely (consider Argon2id for passphrase-based encryption)

### 6.3 Message Validation

- [ ] Implement envelope validation (§2.1.1): signature, timestamp, deduplication
- [ ] Implement Wave validation (§2.2.1): PoW, TTL, parent hash verification
- [ ] Implement device key validation if multi-device is supported

### 6.4 Storage

- [ ] Persistent store for keypairs and identity metadata
- [ ] Bloom filter or equivalent for message deduplication
- [ ] Cache for Wave and identity declaration messages
- [ ] TTL-based eviction (expire Waves after TTL_seconds)

### 6.5 Shroud Circuit Support

- [ ] Implement 3-hop circuit construction (§4.1)
- [ ] Implement onion encryption/decryption
- [ ] Relay advertisements and circuit routing

## 7. Compatibility

This specification defines MURMUR protocol version 1. Future versions may extend the wire format (e.g., additional message types, new topics) but MUST maintain backward compatibility for message validation — older clients MUST not crash on unknown message types or envelope fields.

Version negotiation is explicit: nodes negotiate supported versions via libp2p identify protocol or in-band (e.g., as part of bootstrap handshakes).

## 8. Security Considerations

### 8.1 Threat Model

This specification addresses the following threats in-scope:

- **Passive network observation**: Shroud onion routing protects against observers attempting to correlate IPs with identities
- **Active peer abuse**: PoW and peer scoring mitigate spam and DoS attacks
- **Message tampering**: All messages are signed and integrity-checked

The following are out-of-scope:

- **Global adversaries**: Tor or I2P integration is recommended for users with this threat model
- **Endpoint compromise**: This specification does not protect against malware on the user's device
- **Side-channel attacks**: Timing analysis, power analysis, and other side channels are not mitigated

### 8.2 Key Management

- Keypairs MUST be stored in encrypted form, using Argon2id or equivalent for key derivation
- Private keys MUST be zeroed (overwritten with zeros) after use
- Multi-device implementations MUST keep the master key offline and never use it for routine signing

### 8.3 Message Expiry

Implement strict TTL enforcement to prevent replay:

- Reject messages with `timestamp_unix` older than the message's TTL
- Do not relay messages that have exceeded their TTL
- Delete cached messages after TTL expiry

## Appendix: Proto File References

The authoritative protobuf definitions are in the MURMUR repository:

- `proto/gossip.proto` — `GossipMessage` union type
- `proto/wave.proto` — `Wave`, `Reply`, `Amplification`, `MurmurEnvelope`
- `proto/identity.proto` — `IdentityDeclaration`, `DeviceAuthorizationDeclaration`, etc.
- `proto/shroud.proto` — `TunnelExtendCell`, `TunnelDataCell`, etc.
- `proto/mechanics.proto` — Game event types
- `proto/resonance.proto` — Resonance declaration types

All proto files use `syntax = "proto3"` and are available in the public repository.

---

**Document Version**: 1.0  
**Date**: 2026-05-07  
**Status**: Stable (v0.1 release)
