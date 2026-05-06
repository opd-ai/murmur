# Identity Continuity Across Key Rotation

**Task:** PLAN.md §3.4  
**Status:** Design Complete — Ready for Implementation  
**Date:** 2026-05-06  
**Dependencies:** `pkg/identity/keys/`, `proto/identity.proto`, `pkg/networking/gossip/`

---

## Executive Summary

This document specifies MURMUR's **key rotation system**, allowing users to replace their Ed25519 keypairs (Surface or Specter) without losing identity continuity. When a key is compromised or proactively rotated, users generate a new keypair and broadcast a cryptographically signed **Continuity Declaration** linking the old key to the new key.

**Key Properties:**
- **Cryptographic continuity:** Old key signs a declaration authorizing the new key
- **Automatic peer updates:** Contacts observe the declaration and transparently update their peer store (no "is this really you?" friction)
- **Grace period:** Old key remains valid for 7 days to allow declaration propagation
- **Separate Surface and Specter rotation:** Rotating one layer does NOT affect the other (maintains cross-layer unlinkability)
- **Backward compatibility:** Legacy peers that don't support continuity declarations will see a new identity (graceful degradation)

**Design Philosophy:** Proactive security hygiene (routine key rotation) should not force users to lose their social graph. Cryptographic signatures prove continuity; peers trust the old key's assertion that "my new key is X".

---

## Problem Statement

### Current Limitation (per BIP39_RECOVERY_AUDIT.md §Gap 4)

**Existing behavior:**
- Keypairs never rotate
- If a key is compromised (device theft, malware, accidental exposure), user has two options:
  1. **Continue using compromised key** — attacker can impersonate user forever
  2. **Generate new identity** — lose all connections, Resonance, game history, council memberships

**Why this is unacceptable:**
1. **No recovery from compromise:** Key compromise is permanent identity loss (unlike password breach where you can change the password)
2. **No proactive rotation:** Security best practice is periodic key rotation (e.g., annually); MURMUR currently makes this impossible without starting over
3. **Realistic threat:** Mobile devices are routinely stolen/lost; malware occasionally extracts keys from app storage
4. **Social network context:** Losing identity = losing social graph. Users will tolerate compromised keys rather than start over.

---

## Design Goals

1. **Cryptographic Proof of Continuity**  
   Old key signs a declaration: "My new public key is Y. Treat messages from Y as from me." Peers verify the signature and trust the assertion.

2. **Transparent Updates for Peers**  
   Contacts observe continuity declarations via gossip and automatically update their peer store. No manual re-connection or "verify identity" flow.

3. **Grace Period for Propagation**  
   Old key remains valid for 7 days after rotation declaration. This allows time for the declaration to propagate to all peers (gossip is eventually consistent, not instant).

4. **Revocation of Old Key**  
   After grace period expires, messages signed with old key are rejected. Prevents attacker who compromised old key from impersonating user after rotation.

5. **Separate Surface and Specter Rotation**  
   Surface Ed25519 key and Specter Curve25519 key rotate independently. Rotating Surface key does NOT reveal or rotate Specter key (preserves anonymity).

6. **Chain of Continuity**  
   Users can rotate multiple times (Key A → Key B → Key C). Peers follow the chain and recognize the latest key as canonical.

---

## Architecture

### Continuity Chain

```
Original Identity Key (Key A)
  ↓ [ContinuityDeclaration signed by Key A]
New Identity Key (Key B)
  ↓ [ContinuityDeclaration signed by Key B]
Rotated Identity Key (Key C)
  ↓ ...
```

**Key Relationships:**
- Each key in the chain signs a declaration authorizing the next key
- Peers store the continuity chain and resolve "identity root" to "current active key"
- Identity root = first key in chain (immutable); current key = latest key in chain

**Constraint:** Chain length limited to 100 rotations (prevents unbounded storage growth; 100 rotations = 100 years of annual rotation).

---

## Protocol Messages

### 1. Continuity Declaration

**Purpose:** Old key authorizes a new key to act as the identity going forward.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message ContinuityDeclaration {
  bytes old_public_key = 1;           // Ed25519 public key being rotated out (32 bytes)
  bytes new_public_key = 2;           // Ed25519 public key being rotated in (32 bytes)
  int64 rotation_timestamp_unix = 3;  // When rotation takes effect
  int64 grace_period_days = 4;        // How long old key remains valid (default 7)
  string rotation_reason = 5;         // Optional: "Security incident", "Proactive rotation", "Device upgrade"
  bytes old_key_signature = 6;        // Ed25519 signature over (old_pubkey || new_pubkey || timestamp || grace_period || reason)
                                      // Signed by old_private_key (proves old key authorizes this rotation)
  bytes new_key_signature = 7;        // Ed25519 signature over same message, signed by new_private_key
                                      // Proves new key holder participated in rotation (prevents old-key-holder-only attacks)
}
```

**Validation Rules:**
- `old_key_signature` MUST verify against `old_public_key`
- `new_key_signature` MUST verify against `new_public_key`
- `rotation_timestamp_unix` MUST be within ±300 seconds of receiver's clock
- `grace_period_days` MUST be ≥1 and ≤30 (min 1 day for propagation, max 30 days to prevent indefinite dual-validity)
- `old_public_key` MUST NOT equal `new_public_key` (no self-rotation)
- `old_public_key` MUST be either an identity root OR the current active key in an existing continuity chain

**Propagation:**
- Broadcast on `/murmur/identity/1` GossipSub topic wrapped in `MurmurEnvelope`
- Peers update their continuity chain: append new key to chain for this identity
- Peers begin accepting messages from `new_public_key` AND `old_public_key` (dual validity during grace period)
- After grace period expires, peers reject messages from `old_public_key`

---

### 2. Chain Query Request (Optional)

**Purpose:** New peer joins network and needs to resolve an identity's current key. Queries DHT for continuity chain.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message ContinuityChainRequest {
  bytes identity_root_key = 1;  // Original identity public key (start of chain)
}

message ContinuityChainResponse {
  bytes identity_root_key = 1;
  repeated ContinuityDeclaration chain = 2;  // Full chain from root to current key
  bytes current_active_key = 3;              // Latest key in chain (convenience field)
}
```

**Usage:**
- Peer receives a Wave signed by key X
- Peer does not recognize key X
- Peer queries DHT: "Give me continuity chain for identity root Y" (where Y is derived from X via chain lookup)
- DHT responds with full chain
- Peer verifies each signature in chain, resolves X as current key for identity Y

---

## Rotation Flow

### User Experience

**Scenario:** User suspects their Surface identity key may have been compromised (device was stolen, later recovered). User wants to rotate to a new key proactively.

**Steps:**

1. **User initiates rotation:**
   - Navigate to Settings → Security → "Rotate Identity Key"
   - System displays: "Rotating your key will generate a new cryptographic identity. Your contacts will automatically recognize the new key. Old key remains valid for 7 days."
   - User confirms: "Rotate Key"

2. **System generates new keypair:**
   - Generate new Ed25519 keypair (Surface) OR Curve25519 keypair (Specter)
   - Store new keypair in encrypted keystore (Argon2id + XChaCha20-Poly1305)

3. **System constructs Continuity Declaration:**
   - Retrieve old private key from keystore (passphrase prompt if needed)
   - Construct `ContinuityDeclaration`:
     - `old_public_key` = current identity key
     - `new_public_key` = newly generated key
     - `rotation_timestamp_unix` = current time
     - `grace_period_days` = 7
     - `rotation_reason` = user-provided reason (optional, default "Proactive rotation")
   - Sign with old private key → `old_key_signature`
   - Sign with new private key → `new_key_signature`

4. **System broadcasts declaration:**
   - Wrap `ContinuityDeclaration` in `MurmurEnvelope`
   - Publish to `/murmur/identity/1` GossipSub topic
   - Display progress: "Broadcasting rotation declaration to network..."

5. **System updates local state:**
   - Mark old key as "rotated out" with expiry timestamp = now + 7 days
   - Set new key as active for signing future Waves
   - Append declaration to local continuity chain storage

6. **Grace period countdown:**
   - System displays: "✓ Key rotated successfully. Old key remains valid for 7 days to ensure all contacts receive the update."
   - Optional: System shows propagation status ("Declaration received by 85% of known contacts")

7. **After 7 days:**
   - System automatically stops using old key for any operations
   - System may optionally broadcast a **Revocation Declaration** (explicit notice that old key is no longer valid)
   - Old key can be securely deleted (zeroed before GC)

**Time:** 30–60 seconds (key generation + signing + broadcast).

---

### Peer Experience

**Scenario:** Alice rotates her key. Bob (Alice's contact) receives the continuity declaration.

**Steps:**

1. **Bob's node receives declaration:**
   - GossipSub delivers `ContinuityDeclaration` message on `/murmur/identity/1`
   - Bob's node validates:
     - `old_key_signature` verifies against Alice's old public key (which Bob has in peer store)
     - `new_key_signature` verifies against Alice's new public key (proves new key holder participated)
     - Timestamp is fresh (±300s)
   - Validation passes → Bob's node trusts the declaration

2. **Bob's node updates peer store:**
   - Retrieve Alice's existing peer record
   - Append continuity declaration to Alice's continuity chain
   - Set "active key" = Alice's new public key
   - Set "grace period expiry" = rotation_timestamp + 7 days
   - Store declaration in `continuity_chains` Bbolt bucket

3. **Dual validity during grace period:**
   - Bob's node accepts Waves signed by Alice's old key (if received within grace period)
   - Bob's node accepts Waves signed by Alice's new key (immediately)
   - Both signatures map to the same identity in Bob's UI (no duplicate contact entries)

4. **After grace period:**
   - Bob's node rejects Waves signed by Alice's old key
   - Bob's node only accepts Waves signed by Alice's new key
   - Old key signature is treated as invalid (as if from an unknown peer)

5. **Bob's UI:**
   - No visible change to Bob (Alice remains "Alice" in contact list)
   - Optional: Bob sees notification: "Alice rotated their identity key (proactive security measure)"
   - Alice's sigil remains unchanged (sigils are derived from identity root, not current key)

**Time:** Instantaneous upon receiving declaration (gossip latency is typically <1 second within mesh).

---

## Continuity Chain Management

### Chain Storage

**Bbolt Bucket:** `continuity_chains`

**Key-Value Structure:**

```
Key: identity_root_key (32 bytes)  // First key in chain (immutable)
Value: ContinuityChain (protobuf message)

message ContinuityChain {
  bytes identity_root_key = 1;
  repeated ContinuityDeclaration declarations = 2;  // Chronologically ordered
  bytes current_active_key = 3;                     // Latest key (cached for fast lookup)
  int64 last_updated_unix = 4;
}
```

### Chain Resolution Algorithm

**Input:** Signature verification request for Wave signed by key X

**Output:** True if X is the current active key for its identity; False otherwise

**Algorithm:**

```go
func IsKeyValid(signingKey []byte, timestamp int64) bool {
    // 1. Lookup identity root key for signingKey
    identityRoot := ResolveIdentityRoot(signingKey)
    if identityRoot == nil {
        return false  // Unknown identity
    }

    // 2. Retrieve continuity chain for this identity
    chain := store.GetContinuityChain(identityRoot)
    if chain == nil {
        // No chain exists; signingKey must be identity root
        return bytes.Equal(signingKey, identityRoot)
    }

    // 3. Find the latest declaration where:
    //    - new_public_key == signingKey
    //    - OR old_public_key == signingKey AND timestamp within grace period
    for _, decl := range chain.Declarations {
        if bytes.Equal(decl.NewPublicKey, signingKey) {
            return true  // Current active key
        }
        if bytes.Equal(decl.OldPublicKey, signingKey) {
            graceExpiry := decl.RotationTimestampUnix + (decl.GracePeriodDays * 86400)
            if timestamp <= graceExpiry {
                return true  // Old key still within grace period
            }
        }
    }

    return false  // Key not in chain or expired
}
```

**Performance:** O(N) where N = chain length (≤100). Acceptable for real-time validation.

**Caching:** `current_active_key` field cached in chain record for fast path (O(1) comparison).

---

## Security Analysis

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **Attacker with old key after rotation** | Old key expires after grace period (7 days). Peers reject signatures from expired keys. User can shorten grace period to 1 day if compromise is certain. |
| **Attacker forges continuity declaration** | Requires both old private key AND new private key (dual signature requirement). Attacker with only old key cannot rotate without user cooperation. |
| **Attacker rotates old key without user consent** | User receives notification of rotation (from their own node or contacts). User can broadcast a **Revocation Declaration** invalidating the fraudulent rotation and declaring correct new key. First valid declaration wins (determined by `rotation_timestamp_unix`). |
| **MITM intercepts rotation declaration** | Declaration is signed by both old and new keys (adversary cannot forge signatures). Noise-encrypted libp2p streams prevent traffic modification. |
| **Replay attacks on old declarations** | Each declaration has fresh `rotation_timestamp_unix`. Peers deduplicate by checking if declaration already exists in chain (based on old_pubkey + new_pubkey pair). |
| **Continuity chain poisoning (adversary adds bogus declarations)** | Each declaration must have valid signatures from both old and new keys. Adversary cannot add declaration without compromising keys. |
| **Denial-of-service via chain bloat** | Chain length limited to 100 declarations (enforced by peers; reject declarations that would exceed limit). 100 rotations = reasonable for 100+ years of annual rotation. |
| **Cross-layer deanonymization (Surface ↔ Specter)** | Surface and Specter rotation use independent declarations on separate gossip topics. Rotating Surface key does NOT trigger Specter rotation or vice versa. |

### Grace Period Rationale

**Why 7 days default?**

1. **Gossip propagation time:** GossipSub is eventually consistent. 99% of peers receive messages within 24 hours; 7 days provides 6-day margin.
2. **Offline peers:** Some peers may be offline for days (mobile devices, intermittent connectivity). 7 days ensures they see declaration before old key expires.
3. **User recovery time:** If attacker compromises key and rotates, user has 7 days to detect and counter-rotate before attacker's key becomes sole valid key.

**User-configurable:**
- Urgent rotation (confirmed breach): 1 day grace period
- Proactive rotation (routine hygiene): 7 days grace period
- Paranoid rotation (double-check everything): 14 days grace period

---

## Edge Cases

### 1. Simultaneous Rotation (Race Condition)

**Scenario:** User initiates rotation on Device A and Device B simultaneously. Two continuity declarations with same `old_public_key` but different `new_public_key` values.

**Resolution:**
- First declaration to reach a peer (by `rotation_timestamp_unix` comparison) is accepted
- Second declaration is rejected (conflict: old key already rotated)
- Losing device displays error: "Rotation failed. This key was already rotated on another device."
- User must rotate again from winning device

**Prevention:** Multi-device identity (§3.2) uses separate device keys; rotation is per-device, not per-identity. If User wants to rotate Master Key, only one device (holding Master Key) can initiate rotation.

---

### 2. Lost Declaration (Peer Never Receives It)

**Scenario:** Alice rotates key. Bob's node is offline during rotation. Bob comes back online after grace period expires. Bob's node never received the declaration.

**Resolution:**
- Bob receives a Wave from Alice signed with new key
- Bob's node does NOT recognize the new key (not in peer store)
- Bob's node queries DHT: "What is the continuity chain for Alice's identity?"
- DHT responds with full chain (including rotation declaration)
- Bob's node verifies chain, updates peer store, accepts the Wave

**Fallback:** If DHT query fails (Alice is offline, no DHT nodes have the chain), Bob's UI displays: "Received message from unknown key. Verify identity with contact." Bob can manually verify via out-of-band (phone call, QR code).

---

### 3. Revocation of Fraudulent Rotation

**Scenario:** Attacker compromises User's device, extracts private key, and broadcasts fraudulent rotation to attacker's own key.

**Steps:**

1. **Attacker action:**
   - Attacker generates new keypair (attacker_key)
   - Attacker constructs `ContinuityDeclaration`:
     - `old_public_key` = User's legitimate key
     - `new_public_key` = attacker_key
   - Attacker signs with compromised old key → `old_key_signature` (valid)
   - Attacker signs with attacker's new key → `new_key_signature` (valid)
   - Attacker broadcasts declaration

2. **User detection:**
   - User's legitimate device receives notification: "Your key was rotated to: [attacker_key fingerprint]"
   - User did NOT initiate this rotation → detects attack

3. **User countermeasure:**
   - User immediately broadcasts **Revocation Declaration**:
     - `revoked_key` = attacker_key
     - `correct_key` = User's new legitimate key (User generates fresh keypair)
     - `revocation_signature` = signed by User's old key (proves User, not attacker, is revoking)
   - Peers receive both declarations; prioritize by timestamp:
     - If User's revocation arrives first → attacker's rotation rejected
     - If attacker's rotation arrives first → User's revocation takes precedence (explicit revocation overrides rotation)

**Outcome:** Attacker's fraudulent key is invalidated. User's new legitimate key becomes active. Contacts may see brief confusion (attacker's key valid for <1 hour) but quickly corrected.

---

## Implementation Checklist

**Phase 1: Protobuf Schema (1 day)**
- [ ] Add `ContinuityDeclaration` to `proto/identity.proto`
- [ ] Add `ContinuityChain`, `ContinuityChainRequest`, `ContinuityChainResponse`
- [ ] Regenerate `.pb.go` files

**Phase 2: Key Generation and Signing (1 day)**
- [ ] Extend `pkg/identity/keys/keypair.go`:
  - [ ] `RotateKeyPair(oldKey *KeyPair) (*KeyPair, *ContinuityDeclaration, error)`
  - [ ] Generate new keypair, construct declaration, sign with both keys
- [ ] Unit tests for rotation flow

**Phase 3: Storage Layer (2 days)**
- [ ] Add `continuity_chains` bucket to Bbolt schema
- [ ] Implement `pkg/store/continuity.go`:
  - [ ] `StoreContinuityDeclaration(identityRoot, decl) error`
  - [ ] `GetContinuityChain(identityRoot) (*ContinuityChain, error)`
  - [ ] `ResolveCurrentKey(identityRoot) ([]byte, error)` — fast lookup of current active key
  - [ ] `IsKeyValid(signingKey, timestamp) (bool, error)` — validation algorithm from §"Chain Resolution"
- [ ] Unit tests with in-memory Bbolt

**Phase 4: Gossip Integration (2 days)**
- [ ] Extend `pkg/networking/gossip/` to handle `ContinuityDeclaration` messages:
  - [ ] Parse declarations from `/murmur/identity/1` topic
  - [ ] Validate dual signatures (old + new key)
  - [ ] Call `pkg/store/continuity.go` to persist declarations
  - [ ] Deduplicate (reject if chain already contains this rotation)
- [ ] Integration test: Node A broadcasts rotation, Node B receives and updates peer store

**Phase 5: Wave Validation Enhancement (2 days)**
- [ ] Modify `pkg/content/waves/validate.go`:
  - [ ] Replace direct signature verification with `IsKeyValid(signingKey, timestamp)`
  - [ ] Accept Waves signed by old keys within grace period
  - [ ] Reject Waves signed by expired keys
- [ ] Unit tests for grace period logic (before/during/after expiry)

**Phase 6: DHT Chain Lookup (2 days)**
- [ ] Implement `pkg/networking/discovery/continuity_lookup.go`:
  - [ ] DHT put: store continuity chains at key = identity_root_key
  - [ ] DHT get: query for chains when unknown key encountered
  - [ ] Verify chain on retrieval (all signatures valid)
- [ ] Integration test: Node A stores chain in DHT, Node C (new peer) queries and retrieves

**Phase 7: UI Flows (3 days)**
- [ ] Rotation UI (`pkg/onboarding/screens/rotation_screen.go`):
  - [ ] "Rotate Key" button in Settings → Security
  - [ ] Reason input field (optional)
  - [ ] Grace period slider (1–14 days)
  - [ ] Confirmation dialog: "Old key will expire in X days"
  - [ ] Progress indicator during broadcast
  - [ ] Success screen with new key fingerprint
- [ ] Notification UI:
  - [ ] Display alert when peer rotates key: "Alice rotated their identity key"
  - [ ] Display warning if User receives unexpected rotation: "Your key was rotated. Was this you?"
- [ ] Peer info screen: show continuity chain history (list of rotations with timestamps)

**Phase 8: Revocation System (2 days)**
- [ ] Implement `RevocationDeclaration` protobuf message:
  - [ ] `revoked_key`, `correct_key`, `revocation_signature`
- [ ] Extend gossip handler to process revocations (invalidate fraudulent rotations)
- [ ] UI: "Report Unauthorized Rotation" button if User detects attack

**Phase 9: Integration Testing (2 days)**
- [ ] End-to-end test: User A rotates key, User B accepts Waves from both old and new keys during grace period
- [ ] Test grace period expiry: after 7 days, old key rejected
- [ ] Test chain resolution: User A rotates 3 times (A→B→C→D), peers resolve D as current key
- [ ] Test DHT lookup: Node C (new peer) encounters unknown key, queries DHT, retrieves chain
- [ ] Test revocation: Attacker rotates, User revokes, peers accept correct key
- [ ] Race detector clean (`go test -race ./...`)

**Total Estimate:** 17 days (3.5 engineering weeks)

---

## Success Criteria

1. **Rotation Functionality:**
   - User can rotate key in <60 seconds
   - Declaration propagates to 95%+ of peers within 24 hours
   - Peers automatically recognize new key (no manual re-verification)

2. **Grace Period Behavior:**
   - During grace period: both old and new keys valid
   - After grace period: only new key valid
   - Peers enforce expiry correctly (reject expired old keys)

3. **Chain Resolution:**
   - Chains up to 100 rotations resolve correctly
   - DHT lookup retrieves chains for offline peers
   - Current active key lookup is O(1) (cached)

4. **Security:**
   - Attacker with old key cannot forge rotation (requires new key signature)
   - Revocation declarations invalidate fraudulent rotations
   - No cross-layer linkability (Surface and Specter rotate independently)

5. **UX:**
   - Peers see no visible disruption (contact remains "Alice", same sigil)
   - Rotation notifications are informational, not alarming
   - Revocation flow is accessible (Settings → Security → Report Unauthorized Rotation)

6. **Reliability:**
   - Zero race conditions (`go test -race ./...`)
   - Handles network partitions (declarations propagate when connectivity restored)
   - No data loss if rotation happens during network split (declarations stored locally, broadcast when online)

---

## Future Enhancements (Post-v1.0)

1. **Automatic Rotation Reminders:**
   - System prompts User annually: "Last key rotation was 365 days ago. Rotate now?"
   - One-click rotation flow with pre-filled default settings

2. **Emergency Rotation:**
   - Zero grace period (old key invalidated immediately)
   - Useful if User detects active compromise
   - Peers must query DHT immediately (no grace period caching)

3. **Hierarchical Rotation:**
   - Master Key rotation affects all Device Keys (multi-device identity from §3.2)
   - Device Key rotation does NOT require Master Key participation

4. **Rotation Metadata:**
   - Track rotation frequency in peer reputation (frequent rotation = potential security issue?)
   - Display rotation history in peer info screen (transparency)

5. **Cross-Client Compatibility Flags:**
   - Declare support for continuity declarations in libp2p identify protocol
   - Legacy clients without continuity support see rotations as new identities (graceful degradation)

---

## References

- **PLAN.md §3.4** — Original task specification
- **BIP39_RECOVERY_AUDIT.md §Gap 4** — Problem statement (no key rotation)
- **SECURITY_PRIVACY.md §2** — Key material handling and zeroing
- **MULTI_DEVICE_IDENTITY.md** — Device key management (rotation integrates with device authorization)
- **THREAT_MODEL.md** — Adversary classes (malicious peer, compromised device)
- **DESIGN_DOCUMENT.md §2 (Identity)** — Surface vs. Specter keypair independence
