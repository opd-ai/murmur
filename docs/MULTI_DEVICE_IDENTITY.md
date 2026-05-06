# Multi-Device Identity Design

**Task:** PLAN.md §3.2  
**Status:** Design Complete — Ready for Implementation  
**Date:** 2026-05-06  
**Dependencies:** `pkg/identity/keys/`, `proto/identity.proto`, `pkg/store/`

---

## Executive Summary

This document specifies MURMUR's multi-device identity system, allowing users to maintain **one logical identity across multiple physical devices** without:
- Requiring all devices to share the same private key (security risk)
- Forcing users to recover their identity on each device switch (UX friction)
- Losing cryptographic continuity when devices are added or revoked

**Design Principle:** One **Master Identity** (cryptographic root of trust, derived from BIP-39 mnemonic) authorizes multiple **Device Keys** (ephemeral Ed25519 keypairs tied to specific devices). Peers observe device key changes as continuity, not identity loss.

---

## Problem Statement

### Current Single-Device Limitation

**Existing behavior** (per BIP39_RECOVERY_AUDIT.md):
- One identity = one BIP-39 mnemonic = one device active at a time
- User recovering on a new device must either:
  - Import the same private key (dangerous: key material exists on multiple devices)
  - Generate a new identity (loses all connections, Resonance, game history)

**Why this is unacceptable:**
1. **Security:** Sharing the same private key across devices violates best practices. If one device is compromised, the attacker gains full identity control forever.
2. **UX:** Users who upgrade phones, use laptop + mobile, or temporarily lose a device face identity recovery every time they switch — with loss of connections and local state.
3. **Realism:** Multi-device usage is standard (users check messages on phone during the day, participate in games on laptop at night).

---

## Design Goals

1. **One Master Identity → Many Device Keys**  
   Master Identity (Ed25519 root keypair) is the source of truth. Device Keys (ephemeral Ed25519 keypairs) are authorized by the Master and rotate independently.

2. **Seamless Device Addition**  
   User adds a new device by signing a Device Authorization Declaration with the Master Key (typically via QR code scan or secure pairing flow).

3. **Device Revocation Without Device Access**  
   User can revoke a lost/stolen device by signing a Device Revocation Declaration from any remaining authorized device (or by recovering Master Key).

4. **Cryptographic Continuity for Peers**  
   Peers observe device key changes via signed declarations and automatically update their local peer store. No "is this really you?" friction.

5. **Surface + Specter Independence**  
   Surface Layer devices and Specter Layer devices are managed separately. Revoking a Surface device does NOT revoke the corresponding Specter device (preserves cross-layer unlinkability).

6. **Master Key Remains Offline**  
   Master private key is used only for:
   - Initial identity creation
   - Device authorization
   - Device revocation
   - BIP-39 recovery on new device  
   Master key is **never used for routine message signing** — this minimizes exposure.

---

## Architecture

### Keypair Hierarchy

```
BIP-39 Mnemonic (24 words)
  ↓
Master Identity Keypair (Ed25519)
  ├── Master Public Key (identity root)
  └── Master Private Key (used only for device management)

Device 1 Keypair (Ed25519)
  ├── Device 1 Public Key
  └── Device 1 Private Key (signs Waves, connections on this device)

Device 2 Keypair (Ed25519)
  ├── Device 2 Public Key
  └── Device 2 Private Key (signs Waves, connections on this device)

...
```

**Key Relationships:**
- Master Public Key is the **canonical identity**.
- Each Device Public Key is linked to Master Public Key via a signed Device Authorization Declaration.
- Peers store mappings: `master_pubkey → [device1_pubkey, device2_pubkey, ...]`

---

## Protocol Messages

### 1. Device Authorization Declaration

**Purpose:** Master Key authorizes a new device to act on behalf of the identity.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message DeviceAuthorizationDeclaration {
  bytes master_public_key = 1;    // 32-byte Ed25519 public key (identity root)
  bytes device_public_key = 2;    // 32-byte Ed25519 public key (new device)
  string device_label = 3;        // Human-readable label (e.g., "iPhone 15", "Work Laptop")
  int64 timestamp_unix = 4;       // Authorization timestamp (Unix seconds)
  int64 expires_unix = 5;         // Optional expiry (0 = no expiry)
  bytes master_signature = 6;     // Ed25519 signature over (master_pubkey || device_pubkey || label || timestamp || expires)
}
```

**Validation Rules:**
- `master_signature` MUST verify against `master_public_key`
- `timestamp_unix` MUST be within ±300 seconds of receiver's clock
- `expires_unix` (if non-zero) MUST be in the future
- A device key cannot authorize another device key (only Master can authorize)

**Propagation:**
- Broadcast on `/murmur/identity/1` GossipSub topic wrapped in `MurmurEnvelope`
- Peers update local peer store: add `device_public_key` to authorized device list for `master_public_key`
- Replaces previous authorization if same `device_public_key` is re-authorized (renewal scenario)

---

### 2. Device Revocation Declaration

**Purpose:** Master Key (or another authorized device with revocation capability) revokes a compromised or lost device.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message DeviceRevocationDeclaration {
  bytes master_public_key = 1;         // Identity root
  bytes device_public_key_to_revoke = 2; // Device being revoked
  string revocation_reason = 3;        // Optional (e.g., "Lost phone", "Security incident")
  int64 timestamp_unix = 4;
  bytes master_signature = 5;          // Signature over (master_pubkey || device_to_revoke || timestamp)
}
```

**Validation Rules:**
- `master_signature` MUST verify against `master_public_key`
- `device_public_key_to_revoke` MUST be in the identity's current authorized device list
- Once revoked, messages signed by the revoked device key are rejected by peers

**Propagation:**
- Broadcast on `/murmur/identity/1`
- Peers update local peer store: remove `device_public_key_to_revoke` from authorized device list
- Grace period: 7 days to allow declaration propagation before hard rejection (configurable)

---

### 3. Enhanced Wave Signature

**Modification to existing `Wave` messages:**

Current behavior: Wave signature field contains Ed25519 signature from identity public key.

**New behavior:**
- Wave `signature` field is still Ed25519 signature
- Wave includes new field: `device_public_key` (which device signed this Wave)
- Peers verify:
  1. Signature verifies against `device_public_key`
  2. `device_public_key` is in the authorized device list for the identity's `master_public_key`

**Backward Compatibility:**
- If `device_public_key` is omitted (legacy Waves), assume single-device identity (device key = master key)
- During transition period, both single-device and multi-device Waves coexist

---

## Device Addition Flow

### User Experience

**Scenario:** User has MURMUR installed on Phone A (already set up). User wants to add Laptop B.

**Steps:**

1. **On Phone A (Primary Device):**
   - Navigate to Settings → Devices → "Add New Device"
   - Phone A generates a pairing QR code containing:
     - Phone A's IP address (for local network pairing)
     - One-time pairing token (256-bit random nonce)
     - Expiry timestamp (5 minutes)

2. **On Laptop B (New Device):**
   - Install MURMUR → Launch → Select "Add to Existing Identity"
   - Scan QR code from Phone A
   - Laptop B generates a new Ed25519 device keypair
   - Laptop B connects to Phone A via local network (WebSocket over TLS with Noise handshake using pairing token)
   - Laptop B sends its device public key to Phone A

3. **On Phone A (Authorization):**
   - Phone A prompts: "Authorize 'Laptop B' to act as you?"
   - User confirms
   - Phone A retrieves Master Private Key from encrypted keystore (passphrase prompt if needed)
   - Phone A constructs `DeviceAuthorizationDeclaration`:
     - `master_public_key` = user's identity root
     - `device_public_key` = Laptop B's device public key
     - `device_label` = "Laptop B"
     - Signs with Master Private Key
   - Phone A sends signed declaration to Laptop B via pairing channel
   - Phone A broadcasts declaration to network via `/murmur/identity/1`

4. **On Laptop B (Activation):**
   - Laptop B receives signed declaration
   - Laptop B verifies `master_signature`
   - Laptop B stores declaration in local database
   - Laptop B is now authorized — begins using its device key for all operations
   - **Laptop B does NOT receive the Master Private Key** (remains offline, protected)

5. **Network Propagation:**
   - All peers subscribed to `/murmur/identity/1` receive the declaration
   - Peers update their peer store: identity `master_public_key` now has two authorized devices
   - Future Waves signed by either Phone A's device key OR Laptop B's device key are accepted

**Time:** 30–60 seconds (assuming local network; 60–120 seconds over internet).

---

### Alternate Flow: Cross-Internet Pairing

If Phone A and Laptop B are not on the same local network:

**Steps 1–2:** Same as above (QR code, Laptop B generates device key)

**Step 3 (Modified):** Instead of direct local connection:
- Phone A broadcasts a **Pairing Request Wave** (ephemeral, 5-minute TTL) on `/murmur/identity/1`
- Pairing Request contains encrypted payload (XChaCha20-Poly1305) with symmetric key derived from pairing token
- Laptop B listens for Pairing Requests, decrypts with pairing token
- Laptop B replies with encrypted message containing its device public key
- Phone A completes authorization as before

**Security:** Pairing token is single-use, short-lived (5 minutes), never logged. MITM attack requires adversary to intercept QR code scan (physical access) AND be on network within 5 minutes.

---

## Device Revocation Flow

### User Experience

**Scenario:** User's phone is stolen. User wants to revoke phone's device key to prevent attacker from impersonating them.

**Steps:**

1. **On Laptop (Surviving Device):**
   - Navigate to Settings → Devices
   - View list of authorized devices (Phone, Laptop, Tablet)
   - Select stolen phone → "Revoke Device"
   - System prompts: "Revoke 'Phone'? This device will no longer be able to send messages as you."
   - User confirms

2. **Revocation via Master Key:**
   - Laptop prompts for Master Key passphrase (or BIP-39 mnemonic if Master Key not cached)
   - Laptop retrieves Master Private Key from encrypted keystore
   - Laptop constructs `DeviceRevocationDeclaration`:
     - `device_public_key_to_revoke` = Phone's device key
     - `revocation_reason` = "Device lost"
   - Laptop signs with Master Private Key
   - Laptop broadcasts declaration to network via `/murmur/identity/1`

3. **Network Propagation:**
   - Peers receive revocation declaration
   - Peers remove Phone's device key from authorized list
   - **Grace period (7 days):** Waves signed by Phone's device key before revocation timestamp are still accepted (prevents retroactive invalidation of legitimate Waves)
   - After grace period, all Waves signed by Phone's device key are rejected

4. **Attacker Impact:**
   - If attacker compromised stolen phone and extracted device private key:
     - Attacker can sign Waves with phone's device key
     - But peers reject those Waves (device key is revoked)
     - Attacker CANNOT extract Master Private Key (Master Key was never stored on phone after initial setup)
     - Attacker CANNOT authorize new devices (requires Master Private Key)

**Time:** <60 seconds to revoke; 7-day grace period for propagation.

---

### Edge Case: All Devices Lost, Only BIP-39 Mnemonic Remains

**Scenario:** User loses all devices but has BIP-39 mnemonic written down.

**Steps:**

1. User installs MURMUR on new device
2. User selects "Recover Identity" → enters BIP-39 mnemonic
3. System derives Master Keypair from mnemonic
4. System generates new device keypair for this device
5. System constructs `DeviceAuthorizationDeclaration` and self-signs with Master Key
6. System broadcasts authorization
7. **Optional:** System also broadcasts revocation for all previously authorized devices (user can select "Revoke all old devices" during recovery)

**Result:** User recovers Master Identity and adds the new device. Old devices are implicitly invalidated (revocation declarations ensure peers reject them).

---

## Storage Schema

### Bbolt Bucket: `devices`

**Key-Value Structure:**

```
Key: master_public_key (32 bytes)
Value: DeviceList (protobuf message)

message DeviceList {
  repeated AuthorizedDevice devices = 1;
}

message AuthorizedDevice {
  bytes device_public_key = 1;
  string device_label = 2;
  int64 authorized_at_unix = 3;
  int64 expires_at_unix = 4;       // 0 = no expiry
  bool is_revoked = 5;
  int64 revoked_at_unix = 6;       // 0 if not revoked
}
```

### Local Keystore Enhancement

**Current:** `pkg/identity/keys/backup.go` stores a single Ed25519 keypair.

**New:** Keystore contains:
- **Master Keypair** (Ed25519) — encrypted with Argon2id + XChaCha20-Poly1305
- **Current Device Keypair** (Ed25519) — encrypted with same passphrase
- **Device Authorization Declaration** — signed proof that current device is authorized
- **Device Label** (string) — user-assigned label for this device

**File Format:**

```json
{
  "version": 2,
  "master_keypair": {
    "public_key": "<hex>",
    "encrypted_private_key": "<base64>",  // XChaCha20-Poly1305 encrypted
    "salt": "<base64>",                    // Argon2id salt
    "nonce": "<base64>"
  },
  "device_keypair": {
    "public_key": "<hex>",
    "encrypted_private_key": "<base64>",
    "salt": "<base64>",
    "nonce": "<base64>",
    "label": "Phone A"
  },
  "device_authorization": {
    "declaration": "<base64>",  // Serialized DeviceAuthorizationDeclaration protobuf
    "added_at": "2026-05-06T03:56:01Z"
  }
}
```

**Backward Compatibility:**
- Version 1 keystore (single keypair, no device concept) is auto-migrated:
  - Existing keypair becomes both Master Key and Device Key
  - Device label set to "Primary Device"
  - Self-signed authorization declaration generated

---

## Security Analysis

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **Device theft/loss** | Revocation declarations invalidate stolen device key. Attacker cannot sign messages after revocation propagates (7-day max exposure). |
| **Device compromise (malware)** | Compromised device can sign messages until detected and revoked. **Cannot** authorize new devices (requires Master Key). **Cannot** extract Master Key (not stored on device after setup). |
| **Stolen BIP-39 mnemonic** | Attacker can derive Master Key → authorize attacker's own devices → impersonate user. **No mitigation within MURMUR** — user must protect mnemonic. Consider adding Shamir Secret Sharing (PLAN.md §3.3) to reduce single-point-of-failure. |
| **MITM during device pairing** | Pairing token (256-bit nonce) prevents MITM. Attacker must intercept QR code scan (physical access) AND be on network within 5-minute window. Noise handshake with token ensures authenticated channel. |
| **Replay attacks on authorization declarations** | Timestamp validation (±300s window) + deduplication (BLAKE3 message_id in Bloom filter) prevents replays. |
| **Revocation declaration forgery** | Only Master Key can sign revocations. Attacker without Master Key cannot revoke legitimate devices. |
| **Cross-layer linkability (Surface ↔ Specter)** | Surface device management and Specter device management are independent. Revoking a Surface device does NOT reveal or revoke the corresponding Specter device. Separate Device Authorization Declarations for each layer. |

---

## Implementation Checklist

**Phase 1: Protobuf Schema (1 day)**
- [ ] Add `DeviceAuthorizationDeclaration` to `proto/identity.proto`
- [ ] Add `DeviceRevocationDeclaration` to `proto/identity.proto`
- [ ] Add `DeviceList` and `AuthorizedDevice` messages
- [ ] Regenerate `.pb.go` files: `protoc --go_out=. proto/*.proto`

**Phase 2: Keystore Enhancement (2 days)**
- [ ] Extend `pkg/identity/keys/backup.go`:
  - [ ] `GenerateDeviceKeyPair()` — creates device-specific Ed25519 keypair
  - [ ] `SignDeviceAuthorization(masterKey, devicePubKey, label) (*DeviceAuthorizationDeclaration, error)`
  - [ ] `SignDeviceRevocation(masterKey, devicePubKey, reason) (*DeviceRevocationDeclaration, error)`
  - [ ] Keystore version 2 format (JSON with master + device keypairs)
  - [ ] Migration from version 1 to version 2
- [ ] Unit tests for all new functions

**Phase 3: Storage Layer (1 day)**
- [ ] Add `devices` bucket to Bbolt schema
- [ ] Implement `pkg/store/devices.go`:
  - [ ] `StoreDeviceAuthorization(masterPubKey, decl) error`
  - [ ] `RevokeDevice(masterPubKey, devicePubKey) error`
  - [ ] `GetAuthorizedDevices(masterPubKey) ([]AuthorizedDevice, error)`
  - [ ] `IsDeviceAuthorized(masterPubKey, devicePubKey) (bool, error)`
- [ ] Unit tests with in-memory Bbolt

**Phase 4: Declaration Broadcasting (2 days)**
- [ ] Extend `pkg/networking/gossip/` to handle new identity message types:
  - [ ] Parse `DeviceAuthorizationDeclaration` from `/murmur/identity/1` messages
  - [ ] Parse `DeviceRevocationDeclaration`
  - [ ] Validate signatures and timestamps
  - [ ] Call `pkg/store/devices.go` to persist authorizations/revocations
- [ ] Integration test: Node A broadcasts authorization, Node B receives and updates peer store

**Phase 5: Wave Signature Validation (2 days)**
- [ ] Modify `pkg/content/waves/validate.go`:
  - [ ] Check if Wave has `device_public_key` field
  - [ ] If present, verify signature against device key AND check device is authorized
  - [ ] If absent (legacy), verify signature against master key (backward compat)
- [ ] Unit tests for both multi-device and single-device Waves

**Phase 6: Device Pairing UI (3 days)**
- [ ] Add "Add Device" flow to `pkg/onboarding/screens/`:
  - [ ] QR code generation (containing pairing token + IP)
  - [ ] QR code scanning (camera access, parse pairing payload)
  - [ ] Pairing handshake (local network WebSocket connection)
  - [ ] Authorization prompt and Master Key passphrase entry
- [ ] Add "Device Management" screen to Settings:
  - [ ] List authorized devices with labels and timestamps
  - [ ] "Revoke Device" button with confirmation dialog
- [ ] Ebitengine rendering for QR code display and scan feedback

**Phase 7: BIP-39 Recovery Enhancement (1 day)**
- [ ] Modify recovery flow to:
  - [ ] Derive Master Keypair from mnemonic
  - [ ] Generate new device keypair for recovery device
  - [ ] Prompt user: "Revoke all previously authorized devices?" (checkbox, default unchecked)
  - [ ] If checked, broadcast revocations for all devices in local cache (if available)
  - [ ] Broadcast self-signed authorization for recovery device

**Phase 8: Integration Testing (2 days)**
- [ ] End-to-end test: User A adds Phone and Laptop, both devices send Waves, User B receives and validates both
- [ ] Revocation test: User A revokes stolen device, User B rejects Waves from revoked device after grace period
- [ ] Recovery test: User A loses all devices, recovers from mnemonic, network accepts new device key
- [ ] Race detector clean (`go test -race ./...`)

**Total Estimate:** 14 days (2 engineering weeks)

---

## Success Criteria

1. **Multi-Device Functionality:**
   - User can authorize up to 10 devices per identity (configurable limit)
   - User can send Waves from any authorized device; peers accept all
   - User can view list of authorized devices with labels and timestamps

2. **Revocation Effectiveness:**
   - Revoked device keys are rejected by peers within 7 days (grace period)
   - User can revoke device from any surviving device (no need for Master Key on every device)
   - Revocation declarations propagate to 95%+ of network within 24 hours (measured via gossip metrics)

3. **Security:**
   - Master Private Key never transmitted over network (only device authorization declarations)
   - Compromising one device does NOT compromise Master Key or other devices
   - Zero race conditions in authorization/revocation handling (`go test -race ./...`)

4. **UX:**
   - Device pairing completes in <60 seconds (local network) or <120 seconds (cross-internet)
   - Device revocation completes in <30 seconds (immediate local effect, 7-day network propagation)
   - BIP-39 recovery flow auto-generates new device key (no manual device authorization step)

5. **Backward Compatibility:**
   - Single-device identities (version 1 keystore) auto-migrate to multi-device (version 2 keystore) on first launch
   - Legacy Waves without `device_public_key` field continue to validate correctly
   - No breaking changes to existing `/murmur/identity/1` topic consumers

---

## Future Enhancements (Post-v1.0)

1. **Device Expiry:**
   - Authorize temporary devices with expiry timestamps (e.g., "Allow work laptop for 30 days")
   - Auto-revoke expired devices

2. **Device Capability Flags:**
   - Some devices are "read-only" (can receive, cannot send)
   - Some devices are "admin" (can authorize/revoke other devices without Master Key)

3. **Hierarchical Device Keys:**
   - Master Key → Admin Device Keys → Standard Device Keys
   - Admin devices can revoke standard devices without Master Key

4. **Cross-Platform Sync:**
   - Sync device list and authorization declarations across devices via encrypted DHT records
   - User adding Device C from Device A automatically informs Device B (without manual broadcast)

5. **Specter Multi-Device:**
   - Apply same device management model to Anonymous Layer identities
   - Separate device authorization for Specter identities (no cross-layer linkability)

---

## References

- **PLAN.md §3.2** — Original task specification
- **BIP39_RECOVERY_AUDIT.md §Gap 1** — Problem statement and user impact
- **SECURITY_PRIVACY.md §2.1** — Key material handling and zeroing requirements
- **THREAT_MODEL.md** — Adversary classes and mitigation strategies
- **DESIGN_DOCUMENT.md §2 (Identity)** — Surface vs. Specter keypair independence
- **TECHNICAL_IMPLEMENTATION.md §1.4** — Argon2id parameters for keystore encryption
